# ✂️ CalCut - ics 파일 분할 도구

**대용량 iCalendar(.ics) 파일을 손쉽게 분할하는 무료 온라인 도구**

구글 캘린더, 네이버 캘린더, 아웃룩에서 내보낸 ics 파일이 너무 커서 가져오기 실패하나요?  
CalCut으로 512KB, 1MB 등 원하는 크기로 분할하세요. **한글 깨짐 없이 완벽 지원!**

🔗 **Live**: [https://calcut-app.pages.dev](https://calcut-app.pages.dev)

---

## 이런 문제를 해결합니다

- ❌ "ics 파일이 너무 커서 업로드가 안 돼요"
- ❌ "구글 캘린더 가져오기 실패"
- ❌ "다른 분할 도구 쓰니까 한글이 깨져요"
- ❌ "10MB 캘린더 백업 파일을 나눠야 해요"

## 특징

| 기능 | 설명 |
|------|------|
| 🇰🇷 **한글 완벽 지원** | UTF-8 인코딩 정확히 처리, 한글 깨짐 없음 |
| 🔒 **100% 브라우저 처리** | 파일이 서버로 전송되지 않음 (프라이버시 보장) |
| 📦 **크기 기준 분할** | 512KB, 1MB, 2MB 등 원하는 크기로 분할 |
| 📄 **이벤트 단위 분할** | 이벤트당 1개 파일로 분할 |
| 💾 **ZIP 다운로드** | 분할된 파일을 한 번에 다운로드 |
| ⚡ **빠른 처리** | WebAssembly 기반으로 대용량 파일도 빠르게 처리 |

## 사용법

1. [CalCut](https://calcut-app.pages.dev) 접속
2. .ics 파일 드래그 앤 드롭 또는 파일 선택
3. 분할 옵션 선택
   - **크기 기준**: 512KB (권장), 1MB, 2MB, 5MB, 10MB
   - **이벤트 단위**: 이벤트당 1개 파일
4. "분할하기" 클릭
5. 개별 다운로드 또는 ZIP으로 한번에 다운로드

## 지원 캘린더 서비스

- ✅ Google Calendar (구글 캘린더)
- ✅ Apple Calendar (애플 캘린더)
- ✅ Microsoft Outlook (아웃룩)
- ✅ 네이버 캘린더
- ✅ 카카오 캘린더
- ✅ 기타 iCalendar(.ics) 형식 지원 서비스

## 왜 CalCut인가?

### 기존 도구의 문제점
- 대부분 영문 기반 → **한글 깨짐 발생**
- 서버 업로드 필요 → **프라이버시 우려**
- 이벤트 개수로만 분할 → **용량 제한 대응 불가**

### CalCut의 해결책
- **Go + WebAssembly** 기반 정확한 UTF-8 처리
- **100% 클라이언트** 처리로 파일 유출 걱정 없음
- **용량 기준 분할** 지원 (Google Calendar 1MB 제한 대응)

## 기술 스택

- **Frontend**: HTML, CSS, JavaScript (Vanilla)
- **Core Logic**: Go → WebAssembly (WASM)
- **Hosting**: Cloudflare Pages

## CLI 버전

웹 외에도 커맨드라인에서 사용 가능:

```bash
# macOS/Linux 빌드
go build -o calcut ./cmd/

# Windows 빌드
GOOS=windows GOARCH=amd64 go build -o calcut.exe ./cmd/

# 사용 예시
./calcut calendar.ics --max-size 512K --output-dir ./output
./calcut calendar.ics --max-size 1M --prefix meeting
```

## 로컬 개발

```bash
# 웹 서버 실행
cd web
python3 -m http.server 8080

# http://localhost:8080 접속
```

## 관련 키워드

`ics 파일 분할` `ics splitter` `iCalendar 분할` `구글 캘린더 ics 나누기` `캘린더 파일 용량 줄이기` `ics 파일 너무 큼` `한글 캘린더 깨짐` `Google Calendar import failed` `ics file too large`

## 라이선스

MIT License

## 기여

이슈나 PR 환영합니다!
