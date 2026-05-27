# Development Plan

主要 MVP 規格請參考 `docs/mvp-spec.md`。詳細歷史紀錄目前同步維護於 `docs/development-progress.md`。

## Current Status

- Current step: Step 11 - 完成整場面試流程
- Status: Completed
- Last updated: 2026-05-27

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
| 8 | TTS 朗讀題目 | Completed |
| 9 | 前端錄音 | Completed |
| 10 | 回答音檔上傳 API | Completed |
| 11 | 完成整場面試流程 | Completed |
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
- Replaced OpenAI-backed question generation with Gemini-backed question generation.
- Added optional `GEMINI_API_KEY`, configurable `GEMINI_MODEL`, and `GEMINI_FALLBACK_MODEL`.
- Preserved mock question generation when no API key is configured.
- Added timeout, retry handling for transient Gemini `429` and `503` responses, and model fallback.
- Fixed Gemini response body decoding so the request timeout context is released after the body is read.
- Validated Gemini JSON output before saving questions.

Verification:

- `go test ./...` passed in `backend`.
- Mock-mode create interview flow returned generated questions without `GEMINI_API_KEY`.
- Gemini live API call returned structured JSON with `gemini-2.5-flash`.
- Manual create-interview verification with `GEMINI_API_KEY` configured completed successfully.

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

## Step 8 Completion

Completed on 2026-05-26.

Implemented:

- Added a browser SpeechSynthesis playback button to the session page.
- Read the current question aloud with `zh-TW` language.
- Cancelled existing speech before starting playback to avoid overlap.
- Stopped speech when moving between questions or leaving the session page.
- Disabled playback with a clear message when the browser does not support TTS.

Verification:

- `npm test --prefix frontend` passed.
- `npm run build --prefix frontend` passed.
- `go test ./...` passed in `backend`.
- Manual browser verification confirmed pressing `朗讀題目` starts playback for the current question, moving to `下一題` stops prior playback, and pressing `朗讀題目` again starts playback for the new current question.

## Step 9 Completion

Completed on 2026-05-27.

Implemented:

- Added browser MediaRecorder controls to the session page.
- Requested microphone permission with `navigator.mediaDevices.getUserMedia`.
- Supported start recording and stop recording for the current question.
- Created a local answer audio preview after recording stops.
- Stopped microphone tracks and cleared local previews when moving between questions or leaving the session page.
- Disabled recording with a clear message when the browser does not support MediaRecorder.
- Displayed microphone permission errors.

Verification:

- `npm test --prefix frontend` passed.
- `npm run build --prefix frontend` passed.
- Browser verification confirmed the session page renders recording controls and surfaces local recording device errors.
- Automated MediaRecorder tests confirmed users can record an answer and play it back locally.

## Step 10 Completion

Completed on 2026-05-27.

Implemented:

- Added `POST /api/interviews/{interview_id}/questions/{question_id}/answer`.
- Accepted required multipart `audio` uploads with `audio/webm` content type.
- Saved answer audio to local `backend/storage/audio`.
- Created or updated `answers` rows for each interview/question pair.
- Returned uploaded answer metadata with `transcript_text` as `null`.

Verification:

- `go test ./...` passed in `backend`.
- Repository integration tests passed with `DATABASE_URL` configured.
- Manual API verification uploaded `answer.webm`, confirmed the file in local storage, and confirmed the answer through `GET /api/interviews/{id}`.

## Step 11 Completion

Completed on 2026-05-27.

Implemented:

- Added frontend answer upload from recorded `audio/webm` blobs.
- Tracked uploaded answers by question id in the session page.
- Required the current answer to be uploaded before moving forward or finishing.
- Marked interviews `completed` after all questions have answers.
- Added a minimal `/interviews/{id}/result` completion handoff route for Step 12.

Verification:

- `go test ./...` passed in `backend`.
- Repository integration tests passed with `DATABASE_URL` configured.
- `npm test --prefix frontend` passed.
- `npm run build --prefix frontend` passed.
- Manual Docker Compose verification confirmed answer rows, uploaded files, and completed interview status.

## Backend Logger Maintenance

Completed on 2026-05-27.

Implemented:

- Added `LOG_LEVEL` with `debug`, `info`, `warn`, and `error` support.
- Switched backend startup and handler error logs to structured `log/slog` logging.
- Added request logging middleware for method, path, status, and duration.
- Updated `.env.example` with the default `LOG_LEVEL=info`.

Verification:

- `go test ./...` passed in `backend`.

## Next Step

Step 12 - 建立面試結果頁.

Expected work:

- Load completed interview details on `/interviews/{id}/result`.
- Display interview metadata, each question, and each uploaded answer.
- Add playable audio controls after static audio serving or playable URLs are available.
- Show `transcript_text` or `尚未轉錄`.
