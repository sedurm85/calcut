# ✂️ CalCut

한글 iCalendar(.ics) 파일을 완벽하게 분할하는 웹 도구

🔗 **Live**: [https://calcut.kr](https://calcut.kr) *(도메인 설정 후 수정)*

## 특징

- **🇰🇷 한글 완벽 지원** - UTF-8 인코딩 정확히 처리
- **🔒 100% 브라우저 처리** - 파일이 서버로 전송되지 않음
- **📦 크기 기준 분할** - 1MB, 2MB 등 원하는 크기로 분할
- **📄 이벤트 단위 분할** - 이벤트당 1개 파일로 분할
- **💾 ZIP 다운로드** - 분할된 파일을 한 번에 다운로드

## 사용법

1. [CalCut](https://calcut.kr) 접속
2. .ics 파일 드래그 앤 드롭 또는 파일 선택
3. 분할 옵션 선택 (크기 기준 / 이벤트 단위)
4. 분할하기 클릭
5. 개별 또는 ZIP으로 다운로드

## 지원 캘린더

- Google Calendar
- Apple Calendar
- Microsoft Outlook
- 네이버 캘린더
- 기타 iCalendar(.ics) 형식 지원 서비스

## 기술 스택

- **Frontend**: HTML, CSS, JavaScript (Vanilla)
- **Core Logic**: Go → WebAssembly
- **Hosting**: Cloudflare Pages

## 로컬 실행

```bash
cd web
python3 -m http.server 8080
# http://localhost:8080 접속
```

## CLI 도구

웹 외에도 CLI로 사용 가능:

```bash
# 빌드
go build -o calcut ./cmd/

# 사용
./calcut calendar.ics -max-size 1M -output-dir ./output
```

## 라이선스

MIT License
