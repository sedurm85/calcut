# CalCut 배포 가이드

## 프로젝트 구조

```
ical/
├── web/                    ← 배포할 정적 파일들
│   ├── index.html
│   ├── favicon.svg
│   ├── css/style.css
│   └── js/
│       ├── app.js
│       ├── ical.wasm      ← Go WASM 모듈
│       └── wasm_exec.js   ← Go WASM 런타임
├── wasm/main.go           ← WASM 소스
├── cmd/main.go            ← CLI 소스
└── ...
```

## 로컬 테스트

```bash
cd web
python3 -m http.server 8080
# 브라우저에서 http://localhost:8080 접속
```

## Cloudflare Pages 배포

### 방법 1: GitHub 연동 (추천)

1. GitHub에 레포지토리 생성 및 푸시
2. [Cloudflare Dashboard](https://dash.cloudflare.com) → Pages
3. "Create a project" → "Connect to Git"
4. 레포지토리 선택
5. 빌드 설정:
   - **Build command**: (비워둠)
   - **Build output directory**: `web`
6. "Save and Deploy"

### 방법 2: Direct Upload

```bash
# Cloudflare CLI 설치
npm install -g wrangler

# 로그인
wrangler login

# 배포
wrangler pages deploy web --project-name=calcut
```

## 커스텀 도메인 연결

1. Cloudflare Dashboard → Pages → 프로젝트 선택
2. "Custom domains" 탭
3. "Set up a custom domain"
4. 도메인 입력 (예: `ical.yourdomain.com`)
5. DNS 레코드 자동 설정됨 (Cloudflare DNS 사용시)
6. SSL 인증서 자동 발급 (무료)

## Vercel 배포 (대안)

```bash
# Vercel CLI 설치
npm install -g vercel

# 배포
cd web
vercel --prod
```

## GitHub Pages 배포 (대안)

1. 레포지토리 Settings → Pages
2. Source: "Deploy from a branch"
3. Branch: `main` / `/web` (또는 web 폴더를 루트로)
4. 커스텀 도메인 설정 가능

## WASM 재빌드

소스 수정 후:

```bash
GOOS=js GOARCH=wasm go build -o web/js/ical.wasm ./wasm/
```

## 성능 최적화

### WASM 압축

```bash
# gzip 압축 (서버에서 자동 처리되는 경우 불필요)
gzip -k web/js/ical.wasm
```

Cloudflare는 자동으로 Brotli/Gzip 압축을 적용합니다.

### 캐싱

Cloudflare Pages는 기본적으로 정적 자산을 글로벌 CDN에 캐싱합니다.

## 모니터링

Cloudflare Dashboard에서 확인 가능:
- 방문자 수
- 대역폭 사용량
- 국가별 트래픽
- 성능 메트릭
