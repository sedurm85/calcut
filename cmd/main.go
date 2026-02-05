package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

type Event struct {
	Text    string
	Summary string
	UID     string
	DTStart string
}

type ParsedCalendar struct {
	HeaderLines []string
	Timezones   []string
	Events      []Event
}

func parseIcal(content string) ParsedCalendar {
	lines := strings.Split(content, "\n")

	var headerLines []string
	var timezones []string
	var events []Event

	var currentBlock []string
	blockType := ""
	// RFC 5545: BEGIN/END can nest (e.g. VALARM inside VEVENT)
	nesting := 0

	for _, line := range lines {
		stripped := strings.TrimSpace(line)

		if stripped == "BEGIN:VCALENDAR" || stripped == "END:VCALENDAR" {
			continue
		}

		if strings.HasPrefix(stripped, "BEGIN:") && blockType == "" {
			blockType = strings.SplitN(stripped, ":", 2)[1]
			currentBlock = []string{line}
			nesting = 1
			continue
		}

		if blockType != "" {
			currentBlock = append(currentBlock, line)

			if strings.HasPrefix(stripped, "BEGIN:") {
				nesting++
			} else if strings.HasPrefix(stripped, "END:") {
				nesting--
			}

			if nesting == 0 {
				blockText := strings.Join(currentBlock, "\n")

				switch blockType {
				case "VTIMEZONE":
					timezones = append(timezones, blockText)
				case "VEVENT":
					events = append(events, Event{
						Text:    blockText,
						Summary: extractProperty(blockText, "SUMMARY"),
						UID:     extractProperty(blockText, "UID"),
						DTStart: extractProperty(blockText, "DTSTART"),
					})
				}

				blockType = ""
				currentBlock = nil
			}
			continue
		}

		if stripped != "" {
			headerLines = append(headerLines, line)
		}
	}

	return ParsedCalendar{
		HeaderLines: headerLines,
		Timezones:   timezones,
		Events:      events,
	}
}

func extractProperty(block, propName string) string {
	for _, line := range strings.Split(block, "\n") {
		// handles both "PROP:value" and "PROP;PARAM=x:value" (RFC 5545 §3.2)
		if strings.HasPrefix(line, propName+":") || strings.HasPrefix(line, propName+";") {
			idx := strings.Index(line, ":")
			if idx >= 0 {
				return strings.TrimSpace(line[idx+1:])
			}
		}
	}
	return ""
}

var unsafeChars = regexp.MustCompile(`[<>:"/\\|?*]`)
var multiUnderscore = regexp.MustCompile(`_+`)

func sanitizeFilename(name string) string {
	s := unsafeChars.ReplaceAllString(name, "")
	s = strings.ReplaceAll(s, " ", "_")
	s = multiUnderscore.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	if s == "" {
		return "untitled"
	}
	return s
}

func buildICS(headerLines, timezones []string, eventTexts []string) string {
	var b strings.Builder
	b.WriteString("BEGIN:VCALENDAR\n")
	for _, h := range headerLines {
		b.WriteString(h)
		b.WriteByte('\n')
	}
	for _, tz := range timezones {
		b.WriteString(tz)
		b.WriteByte('\n')
	}
	for _, ev := range eventTexts {
		b.WriteString(ev)
		b.WriteByte('\n')
	}
	b.WriteString("END:VCALENDAR\n")
	return b.String()
}

func skeletonSize(headerLines, timezones []string) int {
	s := buildICS(headerLines, timezones, nil)
	return utf8.RuneCountInString(s)*0 + len(s)
}

func parseSize(s string) (int64, error) {
	s = strings.TrimSpace(strings.ToUpper(s))
	suffixes := map[string]int64{
		"GB": 1024 * 1024 * 1024,
		"MB": 1024 * 1024,
		"KB": 1024,
		"G":  1024 * 1024 * 1024,
		"M":  1024 * 1024,
		"K":  1024,
	}
	for _, suffix := range []string{"GB", "MB", "KB", "G", "M", "K"} {
		if strings.HasSuffix(s, suffix) {
			numStr := s[:len(s)-len(suffix)]
			num, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return 0, fmt.Errorf("잘못된 크기: %s", s)
			}
			return int64(num * float64(suffixes[suffix])), nil
		}
	}
	return strconv.ParseInt(s, 10, 64)
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

func splitPerEvent(parsed ParsedCalendar, outDir, prefix string) ([]string, error) {
	var created []string
	total := len(parsed.Events)

	for i, event := range parsed.Events {
		idx := i + 1
		summaryPart := "event"
		if event.Summary != "" {
			summaryPart = sanitizeFilename(event.Summary)
		}

		var filename string
		if prefix != "" {
			filename = fmt.Sprintf("%s_%03d_%s.ics", prefix, idx, summaryPart)
		} else {
			filename = fmt.Sprintf("%03d_%s.ics", idx, summaryPart)
		}

		content := buildICS(parsed.HeaderLines, parsed.Timezones, []string{event.Text})
		filePath := filepath.Join(outDir, filename)
		if err := writeFile(filePath, content); err != nil {
			return nil, err
		}
		created = append(created, filePath)

		fmt.Printf("  [%d/%d] %s\n", idx, total, filename)
		if event.Summary != "" {
			fmt.Printf("        제목: %s\n", event.Summary)
		}
	}
	return created, nil
}

func splitBySize(parsed ParsedCalendar, outDir, prefix string, maxBytes int64) ([]string, error) {
	skelSize := int64(skeletonSize(parsed.HeaderLines, parsed.Timezones))
	var created []string
	var currentEvents []string
	currentSize := skelSize
	chunkIdx := 1
	tag := prefix
	if tag == "" {
		tag = "part"
	}

	flush := func() error {
		filename := fmt.Sprintf("%s_%03d.ics", tag, chunkIdx)
		content := buildICS(parsed.HeaderLines, parsed.Timezones, currentEvents)
		fileSize := len(content)
		filePath := filepath.Join(outDir, filename)
		if err := writeFile(filePath, content); err != nil {
			return err
		}
		created = append(created, filePath)
		fmt.Printf("  [%d] %s  (%s, %d events)\n", chunkIdx, filename, formatBytes(int64(fileSize)), len(currentEvents))
		chunkIdx++
		currentEvents = nil
		currentSize = skelSize
		return nil
	}

	for _, event := range parsed.Events {
		eventBytes := int64(len(event.Text)) + 1
		projected := currentSize + eventBytes

		if eventBytes+skelSize > maxBytes {
			if len(currentEvents) > 0 {
				if err := flush(); err != nil {
					return nil, err
				}
			}
			fmt.Printf("  ⚠️  이벤트 '%s' (%s) 단독으로도 %s 초과\n",
				event.Summary, formatBytes(eventBytes+skelSize), formatBytes(maxBytes))
			currentEvents = []string{event.Text}
			if err := flush(); err != nil {
				return nil, err
			}
			continue
		}

		if projected > maxBytes && len(currentEvents) > 0 {
			if err := flush(); err != nil {
				return nil, err
			}
		}

		currentEvents = append(currentEvents, event.Text)
		currentSize += eventBytes
	}

	if len(currentEvents) > 0 {
		if err := flush(); err != nil {
			return nil, err
		}
	}

	return created, nil
}

func formatBytes(b int64) string {
	switch {
	case b >= 1024*1024:
		return fmt.Sprintf("%.1f MB", float64(b)/(1024*1024))
	case b >= 1024:
		return fmt.Sprintf("%.1f KB", float64(b)/1024)
	default:
		return fmt.Sprintf("%d bytes", b)
	}
}

func main() {
	outputDir := flag.String("output-dir", "./split_output", "출력 디렉토리")
	prefix := flag.String("prefix", "", "출력 파일명 접두사")
	maxSize := flag.String("max-size", "", "파일당 최대 크기 (예: 1M, 512K, 2MB)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "사용법: split-ical [옵션] <입력파일.ics>\n\n옵션:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n예시:\n")
		fmt.Fprintf(os.Stderr, "  split-ical calendar.ics\n")
		fmt.Fprintf(os.Stderr, "  split-ical -max-size 1M -output-dir ./결과 calendar.ics\n")
	}
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}
	inputPath := flag.Arg(0)

	data, err := os.ReadFile(inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "오류: 파일을 읽을 수 없습니다 - %s\n", err)
		os.Exit(1)
	}

	parsed := parseIcal(string(data))
	if len(parsed.Events) == 0 {
		fmt.Fprintln(os.Stderr, "경고: 이벤트가 없습니다.")
		os.Exit(0)
	}

	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "오류: 디렉토리 생성 실패 - %s\n", err)
		os.Exit(1)
	}

	var maxBytes int64
	if *maxSize != "" {
		maxBytes, err = parseSize(*maxSize)
		if err != nil {
			fmt.Fprintf(os.Stderr, "오류: %s\n", err)
			os.Exit(1)
		}
	}

	fmt.Printf("\n📅 iCalendar 분할 시작\n")
	fmt.Printf("   입력: %s (%s, %d events)\n", inputPath, formatBytes(int64(len(data))), len(parsed.Events))
	fmt.Printf("   출력: %s\n", *outputDir)
	if maxBytes > 0 {
		fmt.Printf("   최대 크기: %s (%s)\n", formatBytes(maxBytes), *maxSize)
	} else {
		fmt.Printf("   모드: 이벤트당 1파일\n")
	}
	fmt.Println()

	var files []string
	if maxBytes > 0 {
		files, err = splitBySize(parsed, *outputDir, *prefix, maxBytes)
	} else {
		files, err = splitPerEvent(parsed, *outputDir, *prefix)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "오류: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✅ 완료: %d개 파일 생성됨 → %s/\n\n", len(files), *outputDir)
}
