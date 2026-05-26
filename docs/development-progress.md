# Development Progress

本文件用於紀錄模擬面試應用 MVP 的開發進度。主要規格請參考 `docs/mvp-spec.md`。

## Current Status

- Current step: Step 6 - LLM 產生問題
- Status: Completed
- Last updated: 2026-05-26

## Step Progress

| Step | 功能 | 狀態 | 驗收重點 |
|---:|---|---|---|
| 1 | 專案骨架 | Completed | Docker Compose 可啟動，health check 正常，前端首頁可見 |
| 2 | DB schema 與 migration | Completed | PostgreSQL 可看到 interviews、questions、answers 三張表 |
| 3 | 建立面試 API | Completed | 可建立 interview 與 mock questions |
| 4 | 查詢面試 API | Completed | 可查詢 interview、questions、answers |
| 5 | 前端建立面試表單 | Completed | 使用者可從 UI 建立面試 |
| 6 | LLM 產生問題 | Completed | 問題可根據輸入動態產生，無 API key 時有 mock/fallback |
| 7 | 模擬面試頁 | Not started | 使用者可逐題瀏覽面試問題 |
| 8 | TTS 朗讀題目 | Not started | 瀏覽器可朗讀目前題目 |
| 9 | 前端錄音 | Not started | 使用者可錄音並回放 |
| 10 | 回答音檔上傳 API | Not started | 後端可保存音檔並建立 answer |
| 11 | 完成整場面試流程 | Not started | 每題都有 answer，interview status 變成 completed |
| 12 | 面試結果頁 | Not started | 可查看題目與回答音檔 |

## Completed Work

### Step 1 - 建立專案骨架

Completed on 2026-05-26.

Implemented:

- Initialized git repository.
- Added Go backend skeleton with chi router.
- Added `GET /health` endpoint returning `{"status":"ok"}`.
- Added React + TypeScript + Vite frontend skeleton.
- Added Tailwind CSS setup.
- Added basic homepage for the MVP.
- Added PostgreSQL service in `docker-compose.yml`.
- Added backend and frontend Dockerfiles.
- Added `.env.example`.
- Added `README.md` with setup and verification commands.
- Added `.gitignore` and Docker ignore files.
- Added basic backend and frontend tests.

Verification:

- `go test ./...` passed in `backend`.
- `npm test` passed in `frontend`.
- `npm run build` passed in `frontend`.
- `docker compose up --build -d` started `postgres`, `backend`, and `frontend`.
- `curl http://localhost:8080/health` returned `{"status":"ok"}`.
- Browser verification confirmed `http://localhost:5173` renders the homepage.

Notes:

- Step 1 intentionally does not connect the backend to PostgreSQL.
- Step 1 intentionally does not add migrations, interview APIs, routing, TTS, recording, upload, LLM, STT, login, or non-MVP features.
- npm reported 5 moderate audit findings in installed dependencies; they did not block Step 1 verification.
- `.env` is local-only and ignored by git.

### Step 2 - DB schema 與 migration

Completed on 2026-05-26.

Implemented:

- Added SQL migrations for `interviews`, `questions`, and `answers`.
- Added Docker Compose migration service.
- Documented migration execution and verification commands.

Verification:

- `docker compose config` passed.
- `docker compose run --rm migrate` applied migration version 1.
- PostgreSQL listed `interviews`, `questions`, `answers`, and `schema_migrations`.
- PostgreSQL schema inspection confirmed UUID primary keys, cascade foreign keys, and unique constraints.

### Step 3 - 建立面試 API

Completed on 2026-05-26.

Implemented:

- Added `POST /api/interviews`.
- Added mock question generation.
- Persisted interviews and questions to PostgreSQL in one transaction.
- Added validation errors for required fields and `question_count` range.

Verification:

- `go test ./...` passed in `backend`.
- `POST /api/interviews` returned an interview id with `questions_ready`.
- PostgreSQL contained the created interview row and 3 generated question rows.

### Step 4 - 查詢面試 API

Completed on 2026-05-26.

Implemented:

- Added `GET /api/interviews/{id}`.
- Returned interview details with generated questions.
- Returned answers array for future uploaded answers.
- Added 404 handling for missing interviews.

Verification:

- `go test ./...` passed in `backend`.
- `GET /api/interviews/{id}` returned the interview created by `POST /api/interviews`.
- Response included `questions` and `answers`.

### Step 5 - 前端建立面試表單

Completed on 2026-05-26.

Implemented:

- Added `/interviews/new` frontend route.
- Added create interview form.
- Submitted form data to `POST /api/interviews`.
- Navigated to `/interviews/{id}` after successful creation.
- Added minimal detail page that loads `GET /api/interviews/{id}` and displays questions.

Verification:

- `npm test --prefix frontend` passed.
- `npm run build --prefix frontend` passed.
- Manual browser verification confirmed creating an interview shows the generated question list.

### Step 6 - LLM 產生問題

Completed on 2026-05-26.

Implemented:

- Kept the existing `QuestionGenerator` interface.
- Added OpenAI-backed question generation.
- Added optional `OPENAI_API_KEY` and configurable `OPENAI_MODEL`.
- Preserved mock question generation when no API key is configured.
- Validated OpenAI JSON output before saving questions.

Verification:

- `go test ./...` passed in `backend`.
- Mock-mode create interview flow returned generated questions without `OPENAI_API_KEY`.
- OpenAI-mode verification was skipped because no local API key was configured.

## Next Step

Step 7 - 建立模擬面試頁.

Expected work:

- Build `/interviews/{id}/session`.
- Display the current interview question.
- Support moving between questions.
