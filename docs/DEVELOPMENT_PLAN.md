# Development Plan

主要 MVP 規格請參考 `docs/mvp-spec.md`。詳細歷史紀錄目前同步維護於 `docs/development-progress.md`。

## Current Status

- Current step: Step 7 - 模擬面試頁
- Status: Completed
- Last updated: 2026-05-26

## Progress

| Step | 功能 | 狀態 |
|---:|---|---|
| 1 | 專案骨架 | Completed |
| 2 | DB schema 與 migration | Completed |
| 3 | 建立面試 API | Completed |
| 4 | 查詢面試 API | Completed |
| 5 | 前端建立面試表單 | Completed |
| 6 | LLM 產生問題 | Completed |
| 7 | 模擬面試頁 | Completed |
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

## Step 5 Completion

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

## Step 6 Completion

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

## Step 7 Completion

Completed on 2026-05-26.

Implemented:

- Added `/interviews/{id}/session` frontend route.
- Loaded interview details through the existing `GET /api/interviews/{id}` client.
- Displayed one current question at a time.
- Added previous and next navigation with boundary button states.
- Added a start-session link from the interview detail page.

Verification:

- `npm test --prefix frontend` passed.
- `npm run build --prefix frontend` passed.
- Manual browser verification confirmed users can open a session and move between questions.

## Next Step

Step 8 - TTS 朗讀題目.
