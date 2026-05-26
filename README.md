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
} | ConvertTo-Json -Compress

Invoke-RestMethod -Uri http://localhost:8080/api/interviews `
  -Method Post `
  -ContentType 'application/json' `
  -Body $body
```

預期回應：

```json
{"id":"<interview_uuid>","status":"questions_ready"}
```

確認資料庫最新建立的 interview 與 questions：

```powershell
docker compose exec postgres psql -U interview_ai -d interview_ai -c "SELECT id, status, question_count FROM interviews ORDER BY created_at DESC LIMIT 1;"
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

- 輸入職位名稱、職位要求及說明、個人資訊、題目數量。
- 送出表單後，頁面導向 `/interviews/{id}`。
- 詳情頁顯示職位名稱、狀態與產生的題目列表。

When running the frontend outside Docker, make sure the backend is on `http://localhost:8080`, or set:

```powershell
$env:VITE_API_PROXY_TARGET='http://localhost:8080'
```

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
