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

前端首頁：

```text
http://localhost:5173
```

應可看到「模擬面試應用」首頁。

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
