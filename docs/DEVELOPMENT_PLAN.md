# Development Plan

主要 MVP 規格請參考 `docs/mvp-spec.md`。詳細歷史紀錄目前同步維護於 `docs/development-progress.md`。

## Current Status

- Current milestone: MVP v1 completed
- Status: Completed
- Last updated: 2026-05-31

## MVP Completion Summary

MVP v1 is complete. Steps 1-12 are implemented and the `docs/mvp-spec.md` section 16 first-version completion definition is satisfied: users can create an interview, generate or mock questions, start the simulated interview, hear each question through browser TTS, record answers, upload each answer audio file, complete the interview, and review questions with uploaded answer audio on the result page.

Future work starts after the MVP baseline. STT, real transcription providers, export/download flows, scoring, AI feedback, and follow-up questions are post-MVP / Phase 2+ work unless explicitly reprioritized.

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
| 12 | 面試結果頁 | Completed |

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

## Step 12 Completion

Completed on 2026-05-27.

Implemented:

- Added `/audio/{interview_id}/{question_id}.webm` static audio serving for uploaded local answer files.
- Replaced the Step 11 result handoff with a full result page.
- Displayed interview metadata, all questions, answer audio controls, transcript text, and `尚未轉錄` placeholders.
- Displayed `尚未上傳回答` for questions without answers.

Verification:

- `go test ./...` passed in `backend`.
- `npm test --prefix frontend` passed.
- `npm run build --prefix frontend` passed.
- Manual Docker Compose verification confirmed the result page displays uploaded answers and audio controls can load `/audio/...webm`.

## Backend Logger Maintenance

Completed on 2026-05-27.

Implemented:

- Added `LOG_LEVEL` with `debug`, `info`, `warn`, and `error` support.
- Switched backend startup and handler error logs to structured `log/slog` logging.
- Added request logging middleware for method, path, status, and duration.
- Updated `.env.example` with the default `LOG_LEVEL=info`.

Verification:

- `go test ./...` passed in `backend`.

## Immersive Interview Flow Completion

Completed on 2026-05-28.

Implemented:

- Added two-stage interview setup with microphone test.
- Added question language selection for `zh-TW` and `en-US`.
- Added asynchronous question generation with `generating_questions` status.
- Added preparation page polling and hidden pre-start questions.
- Added start-interview API and `in_progress` transition.
- Replaced manual session controls with automatic question playback, recording, answer-end control, replay-question behavior, and in-memory background upload queue.
- Added configurable `VITE_MAX_ANSWER_RECORDING_SECONDS`, defaulting to 180 seconds.

Verification:

- `go test ./...` passed in `backend`.
- `npm test` passed in `frontend`.
- `npm run build` passed in `frontend`.

## Gemini SDK Maintenance

Completed on 2026-05-31.

Implemented:

- Kept the existing `QuestionGenerator` interface.
- Kept the existing `GeminiQuestionGenerator` provider type.
- Replaced the hand-written Gemini REST request layer with `google.golang.org/genai` v1.58.0.
- Reused one GenAI client per `GeminiQuestionGenerator` instance.
- Simplified production `genai.ClientConfig` to only provide `APIKey` and rely on SDK defaults.
- Preserved mock-mode behavior when `GEMINI_API_KEY` is empty.
- Preserved configured primary/fallback Gemini models and generated-question JSON validation.
- Updated local backend Go requirement to Go 1.24.

Verification:

- `go test ./internal/llm` passed in `backend`.
- `go test ./...` passed in `backend`.

## Frontend Design Refresh

Completed on 2026-05-31.

Implemented:

- Applied the `frontend/DESIGN.md` Calm Interviewer visual system to the React frontend.
- Added Tailwind design tokens, Google font and Material Symbols links, and shared UI primitives for cards, buttons, badges, icons, shell layout, and setup progress.
- Restyled the homepage, interview setup, preparation/detail, session, and result pages using the static `frontend/design/` pages as visual references.
- Extracted the homepage into its own page component and shared the standard topbar across non-session pages.
- Kept MVP scope unchanged by omitting non-MVP history, export, AI feedback, scoring, and account features.
- Preserved existing routes, API contracts, microphone test, TTS, recording, upload queue, retry, and result audio playback behavior.

Verification:

- `npm test` passed in `frontend`.
- `npm run build` passed in `frontend`.

## Microphone Test Recording Preview

Completed on 2026-06-01.

Implemented:

- Updated the new interview setup microphone test to record a short sample through `MediaRecorder`.
- Added a stop-recording control and local audio preview so users can play back the sample before creating an interview.
- Kept the create-interview button disabled until a microphone sample is recorded successfully.
- Preserved microphone permission and unsupported-browser error handling.

Verification:

- `npm test -- App.test.tsx` passed in `frontend`.
- `npm test` passed in `frontend`.
- `npm run build` passed in `frontend`.

Manual verification:

- Open `/interviews/new`, fill the profile step, and continue to settings.
- Click `測試麥克風`, speak briefly, then click `停止錄音`.
- Play `測試錄音預覽` and confirm the recorded audio is audible.
- Confirm `建立面試` becomes enabled only after the preview is available.

## Interview Setup Input Length Limits

Completed on 2026-06-01.

Implemented:

- Limited the new interview job title field to 50 characters.
- Limited job requirements and user profile fields to 4000 characters.
- Added right-aligned `目前字數/4000` counters below the job requirements and user profile fields.
- Kept the job title field without a visible character counter.

Verification:

- `npm test -- App.test.tsx` passed in `frontend`.
- `npm test` passed in `frontend`.
- `npm run build` passed in `frontend`.

Manual verification:

- Open `/interviews/new`.
- Confirm `職位名稱` stops accepting input after 50 characters and shows no counter.
- Confirm `職位要求及說明` and `個人資訊` stop at 4000 characters.
- Confirm both long text fields display counters such as `目前字數/4000` in the lower-right area below the field.

## Next Step

Post-MVP / Phase 2: Step 13 - STT mock.

Expected work:

- Add a mock transcription flow that writes test `transcript_text`.
- Display populated transcript text on the existing result page.
- Keep real STT provider integration for a later step.
