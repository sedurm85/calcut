#!/usr/bin/env python3
"""
Google iCalendar (.ics) 파일 분할 도구

사용법:
    python split_ical.py <입력파일.ics> [--output-dir <출력디렉토리>] [--prefix <파일접두사>]
"""

import argparse
import re
import sys
from pathlib import Path


def parse_ical(content: str) -> dict:
    lines = content.splitlines()

    header_lines: list[str] = []
    timezones: list[str] = []
    events: list[dict] = []

    current_block: list[str] = []
    block_type: str | None = None
    # RFC 5545: BEGIN/END can nest (e.g. VALARM inside VEVENT)
    nesting = 0

    for line in lines:
        stripped = line.strip()

        if stripped == "BEGIN:VCALENDAR" or stripped == "END:VCALENDAR":
            continue

        if stripped.startswith("BEGIN:") and block_type is None:
            block_type = stripped.split(":", 1)[1]
            current_block = [line]
            nesting = 1
            continue

        if block_type is not None:
            current_block.append(line)

            if stripped.startswith("BEGIN:"):
                nesting += 1
            elif stripped.startswith("END:"):
                nesting -= 1

            if nesting == 0:
                block_text = "\n".join(current_block)

                if block_type == "VTIMEZONE":
                    timezones.append(block_text)
                elif block_type == "VEVENT":
                    summary = _extract_property(block_text, "SUMMARY")
                    uid = _extract_property(block_text, "UID")
                    dtstart = _extract_property(block_text, "DTSTART")
                    events.append(
                        {
                            "text": block_text,
                            "summary": summary,
                            "uid": uid,
                            "dtstart": dtstart,
                        }
                    )

                block_type = None
                current_block = []
            continue

        if stripped:
            header_lines.append(line)

    return {
        "header_lines": header_lines,
        "timezones": timezones,
        "events": events,
    }


def _extract_property(block: str, prop_name: str) -> str:
    for line in block.splitlines():
        # handles both "PROP:value" and "PROP;PARAM=x:value" (RFC 5545 §3.2)
        if line.startswith(prop_name + ":") or line.startswith(prop_name + ";"):
            _, _, value = line.partition(":")
            return value.strip()
    return ""


def sanitize_filename(name: str) -> str:
    sanitized = re.sub(r'[<>:"/\\|?*]', "", name)
    sanitized = sanitized.replace(" ", "_")
    sanitized = re.sub(r"_+", "_", sanitized)
    return sanitized.strip("_") or "untitled"


def build_ics(
    header_lines: list[str], timezones: list[str], event_texts: list[str]
) -> str:
    parts = ["BEGIN:VCALENDAR"]
    parts.extend(header_lines)
    parts.extend(timezones)
    parts.extend(event_texts)
    parts.append("END:VCALENDAR")
    return "\n".join(parts) + "\n"


def _calc_skeleton_size(header_lines: list[str], timezones: list[str]) -> int:
    skeleton = build_ics(header_lines, timezones, [])
    return len(skeleton.encode("utf-8"))


def parse_size(size_str: str) -> int:
    size_str = size_str.strip().upper()
    multipliers = {
        "K": 1024,
        "KB": 1024,
        "M": 1024**2,
        "MB": 1024**2,
        "G": 1024**3,
        "GB": 1024**3,
    }
    for suffix, mult in sorted(multipliers.items(), key=lambda x: -len(x[0])):
        if size_str.endswith(suffix):
            return int(float(size_str[: -len(suffix)]) * mult)
    return int(size_str)


def split_ical(
    input_path: str,
    output_dir: str = "./split_output",
    prefix: str = "",
    max_size_bytes: int = 0,
) -> list[str]:
    input_file = Path(input_path)
    if not input_file.exists():
        print(f"오류: 파일을 찾을 수 없습니다 - {input_path}", file=sys.stderr)
        sys.exit(1)

    content = input_file.read_text(encoding="utf-8")
    parsed = parse_ical(content)

    if not parsed["events"]:
        print("경고: 이벤트가 없습니다.", file=sys.stderr)
        return []

    out_path = Path(output_dir)
    out_path.mkdir(parents=True, exist_ok=True)

    if max_size_bytes > 0:
        return _split_by_size(parsed, out_path, prefix, max_size_bytes)
    else:
        return _split_per_event(parsed, out_path, prefix)


def _split_per_event(parsed: dict, out_path: Path, prefix: str) -> list[str]:
    created_files: list[str] = []
    events = parsed["events"]

    for idx, event in enumerate(events, start=1):
        summary_part = (
            sanitize_filename(event["summary"]) if event["summary"] else "event"
        )
        if prefix:
            filename = f"{prefix}_{idx:03d}_{summary_part}.ics"
        else:
            filename = f"{idx:03d}_{summary_part}.ics"

        file_content = build_ics(
            parsed["header_lines"], parsed["timezones"], [event["text"]]
        )

        file_path = out_path / filename
        file_path.write_text(file_content, encoding="utf-8")
        created_files.append(str(file_path))

        print(f"  [{idx}/{len(events)}] {filename}")
        if event["summary"]:
            print(f"        제목: {event['summary']}")

    return created_files


def _split_by_size(
    parsed: dict, out_path: Path, prefix: str, max_size_bytes: int
) -> list[str]:
    header_lines = parsed["header_lines"]
    timezones = parsed["timezones"]
    events = parsed["events"]
    skeleton_size = _calc_skeleton_size(header_lines, timezones)

    created_files: list[str] = []
    current_events: list[str] = []
    current_size = skeleton_size
    chunk_idx = 1
    event_count_in_chunk = 0

    for event in events:
        event_bytes = len(event["text"].encode("utf-8")) + 1
        projected_size = current_size + event_bytes

        if event_bytes + skeleton_size > max_size_bytes:
            if current_events:
                _flush_chunk(
                    header_lines,
                    timezones,
                    current_events,
                    out_path,
                    prefix,
                    chunk_idx,
                    created_files,
                )
                chunk_idx += 1
                current_events = []
                current_size = skeleton_size

            print(
                f"  ⚠️  이벤트 '{event['summary']}' ({event_bytes + skeleton_size:,} bytes) 단독으로도 {max_size_bytes:,} bytes 초과"
            )
            _flush_chunk(
                header_lines,
                timezones,
                [event["text"]],
                out_path,
                prefix,
                chunk_idx,
                created_files,
            )
            chunk_idx += 1
            continue

        if projected_size > max_size_bytes and current_events:
            _flush_chunk(
                header_lines,
                timezones,
                current_events,
                out_path,
                prefix,
                chunk_idx,
                created_files,
            )
            chunk_idx += 1
            current_events = []
            current_size = skeleton_size

        current_events.append(event["text"])
        current_size += event_bytes

    if current_events:
        _flush_chunk(
            header_lines,
            timezones,
            current_events,
            out_path,
            prefix,
            chunk_idx,
            created_files,
        )

    return created_files


def _flush_chunk(
    header_lines: list[str],
    timezones: list[str],
    event_texts: list[str],
    out_path: Path,
    prefix: str,
    chunk_idx: int,
    created_files: list[str],
) -> None:
    tag = prefix or "part"
    filename = f"{tag}_{chunk_idx:03d}.ics"
    file_content = build_ics(header_lines, timezones, event_texts)
    file_size = len(file_content.encode("utf-8"))

    file_path = out_path / filename
    file_path.write_text(file_content, encoding="utf-8")
    created_files.append(str(file_path))

    print(
        f"  [{chunk_idx}] {filename}  ({file_size:,} bytes, {len(event_texts)} events)"
    )


def main():
    parser = argparse.ArgumentParser(
        description="Google iCalendar (.ics) 파일을 이벤트별로 분할합니다.",
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    parser.add_argument("input", help="입력 iCalendar (.ics) 파일 경로")
    parser.add_argument(
        "--output-dir",
        "-o",
        default="./split_output",
        help="출력 디렉토리 (기본: ./split_output)",
    )
    parser.add_argument(
        "--prefix", "-p", default="", help="출력 파일명 접두사 (기본: 없음)"
    )
    parser.add_argument(
        "--max-size",
        "-s",
        default="",
        help="파일당 최대 크기 (예: 1M, 512K, 2MB). 미지정시 이벤트당 1파일",
    )

    args = parser.parse_args()

    max_size_bytes = parse_size(args.max_size) if args.max_size else 0

    print(f"\n📅 iCalendar 분할 시작")
    print(f"   입력: {args.input}")
    print(f"   출력: {args.output_dir}")
    if max_size_bytes > 0:
        print(f"   최대 크기: {max_size_bytes:,} bytes ({args.max_size})")
    else:
        print(f"   모드: 이벤트당 1파일")
    print()

    files = split_ical(args.input, args.output_dir, args.prefix, max_size_bytes)

    print(f"\n✅ 완료: {len(files)}개 파일 생성됨 → {args.output_dir}/\n")


if __name__ == "__main__":
    main()
