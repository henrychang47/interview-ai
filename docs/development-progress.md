# Development Progress

本文件用於紀錄模擬面試應用 MVP 的開發進度。主要規格請參考 `docs/mvp-spec.md`。

## Current Status

- Current step: Step 1 - 建立專案骨架
- Status: Completed
- Last updated: 2026-05-26

## Step Progress

| Step | 功能 | 狀態 | 驗收重點 |
|---:|---|---|---|
| 1 | 專案骨架 | Completed | Docker Compose 可啟動，health check 正常，前端首頁可見 |
| 2 | DB schema 與 migration | Not started | PostgreSQL 可看到 interviews、questions、answers 三張表 |
| 3 | 建立面試 API | Not started | 可建立 interview 與 mock questions |
| 4 | 查詢面試 API | Not started | 可查詢 interview、questions、answers |
| 5 | 前端建立面試表單 | Not started | 使用者可從 UI 建立面試 |
| 6 | LLM 產生問題 | Not started | 問題可根據輸入動態產生，無 API key 時有 mock/fallback |
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

## Next Step

Step 2 - 建立資料庫 schema 與 migration.

Expected work:

- Add migrations for `interviews`, `questions`, and `answers`.
- Decide and document the migration execution method.
- Verify PostgreSQL contains the three MVP tables after migration.
