# Interview AI MVP

模擬面試應用 MVP。Step 1 建立專案骨架，包含 Go 後端、React 前端與 PostgreSQL container。

## Prerequisites

- Docker Desktop
- Docker Compose
- Go 1.22
- Node.js 20

## Setup

```powershell
Copy-Item .env.example .env
docker compose up --build
```

## Database Migrations

Start PostgreSQL:

```powershell
docker compose up -d postgres
```

Run migrations:

```powershell
docker compose run --rm migrate
```

Verify tables:

```powershell
docker compose exec postgres psql -U interview_ai -d interview_ai -c "\dt"
```

Expected tables:

- `interviews`
- `questions`
- `answers`

## Verification

後端健康檢查：

```powershell
curl http://localhost:8080/health
```

預期回應：

```json
{"status":"ok"}
```

Create Interview API:

```powershell
$body = @{
  job_title = '後端工程師'
  job_description = '需要熟悉 Go、PostgreSQL、REST API'
  user_profile = '有 Java 和 Go 學習經驗，正在準備後端工程師面試'
  question_count = 3
  question_language = 'zh-TW'
} | ConvertTo-Json -Compress

Invoke-RestMethod -Uri http://localhost:8080/api/interviews `
  -Method Post `
  -ContentType 'application/json' `
  -Body $body
```

預期回應：

```json
{"id":"<interview_uuid>","status":"generating_questions"}
```

確認資料庫最新建立的 interview 與 questions：

```powershell
docker compose exec postgres psql -U interview_ai -d interview_ai -c "SELECT id, status, question_count, question_language FROM interviews ORDER BY created_at DESC LIMIT 1;"
docker compose exec postgres psql -U interview_ai -d interview_ai -c "SELECT question_order, question_text FROM questions WHERE interview_id = (SELECT id FROM interviews ORDER BY created_at DESC LIMIT 1) ORDER BY question_order;"
```

Get Interview API:

```powershell
Invoke-RestMethod -Uri "http://localhost:8080/api/interviews/<interview_uuid>" `
  -Method Get |
  ConvertTo-Json -Depth 5
```

前端首頁：

```text
http://localhost:5173
```

應可看到「模擬面試應用」首頁。

建立面試表單：

```text
http://localhost:5173/interviews/new
```

手動驗收：

- 開啟 `http://localhost:5173/interviews/new`。
- 輸入職位資訊後按「下一步」。
- 選擇題目數量與語言，按「測試麥克風」，並允許瀏覽器麥克風權限。
- 按「建立面試」後確認準備頁顯示「題目準備中」，且不顯示題目文字。
- 等到「題目已準備完成」後按「開始模擬面試」。
- 確認 session 自動朗讀題目、隱藏題目文字、朗讀後自動錄音，並支援「回答結束」與「重新播放題目」。
- 完成全部題目後確認結果頁顯示題目與可播放的回答音檔。

When running the frontend outside Docker, make sure the backend is on `http://localhost:8080`, or set:

```powershell
$env:VITE_API_PROXY_TARGET='http://localhost:8080'
$env:VITE_MAX_ANSWER_RECORDING_SECONDS='180'
```

## Question Generation Mode

By default, the backend uses mock question generation when `GEMINI_API_KEY` is empty.

To enable Gemini-backed question generation:

```powershell
$env:GEMINI_API_KEY='<your_api_key>'
$env:GEMINI_MODEL='gemini-2.5-flash'
$env:GEMINI_FALLBACK_MODEL='gemini-2.5-flash-lite'
docker compose up --build -d backend frontend
```

Never commit real API keys. Keep local secrets in `.env`.

When Gemini mode is enabled, the backend sends the interview `job_title`, `job_description`, `user_profile`, and selected `question_language` to Gemini to generate questions. Transient Gemini `429` and `503` responses are retried before falling back from `GEMINI_MODEL` to `GEMINI_FALLBACK_MODEL`. Test data is stored in PostgreSQL, and answer audio files are stored under `backend/storage/audio`; for local cleanup, remove test rows from PostgreSQL and delete local audio files.

## Local Checks

後端：

```powershell
cd backend
go test ./...
```

前端：

```powershell
cd frontend
npm install
npm run build
```
