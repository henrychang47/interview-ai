# Development Plan

主要 MVP 規格請參考 `docs/mvp-spec.md`。詳細歷史紀錄目前同步維護於 `docs/development-progress.md`。

## Current Status

- Current step: Step 4 - 查詢面試 API
- Status: Completed
- Last updated: 2026-05-26

## Progress

| Step | 功能 | 狀態 |
|---:|---|---|
| 1 | 專案骨架 | Completed |
| 2 | DB schema 與 migration | Completed |
| 3 | 建立面試 API | Completed |
| 4 | 查詢面試 API | Completed |
| 5 | 前端建立面試表單 | Not started |
| 6 | LLM 產生問題 | Not started |
| 7 | 模擬面試頁 | Not started |
| 8 | TTS 朗讀題目 | Not started |
| 9 | 前端錄音 | Not started |
| 10 | 回答音檔上傳 API | Not started |
| 11 | 完成整場面試流程 | Not started |
| 12 | 面試結果頁 | Not started |

## Step 3 Completion

Completed on 2026-05-26.

Implemented:

- Added `POST /api/interviews`.
- Added mock question generation.
- Persisted interviews and questions to PostgreSQL in one transaction.

Verification:

- `go test ./...` passed in `backend`.
- `POST /api/interviews` returned an interview id with `questions_ready`.
- PostgreSQL contained the created interview row and requested number of question rows.

## Step 4 Completion

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

## Next Step

Step 5 - 前端建立面試表單.
