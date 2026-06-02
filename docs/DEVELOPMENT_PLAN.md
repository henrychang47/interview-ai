# Development Plan

主要 MVP 規格請參考 `docs/mvp-spec.md`。詳細歷史紀錄目前同步維護於 `docs/development-progress.md`。

## Current Status

- Current milestone: MVP v1 completed
- Status: Completed
- Last updated: 2026-06-01

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

## Gemini Answer Analysis

Completed on 2026-06-01.

Implemented:

- Added migration `000003_add_answer_analysis_fields` for answer analysis status, suggestions, error text, and analyzed timestamp.
- Added background answer analysis after successful audio upload.
- Added mock answer analysis when `GEMINI_API_KEY` is empty.
- Added Gemini-backed audio analysis with structured JSON output for `transcript_text` and `improvement_suggestions`.
- Updated answer upload and interview detail responses with analysis metadata.
- Updated the result page to show pending/processing, completed, and failed analysis states.
- Added result-page polling while any answer analysis is pending or processing.
- Updated `.env.example`, `docs/API.md`, and `README.md` with the new configuration, API shape, and privacy notes.
- Updated Gemini answer analysis to include the related job title, job requirements, user profile, and question text when generating suggestions.
- Updated the result page to render Gemini improvement suggestions with basic Markdown formatting and preserved line breaks.

Verification:

- `go test ./...` passed in `backend`.
- `npm test -- App.test.tsx` passed in `frontend`.
- `go test ./internal/service ./internal/llm ./internal/repository` passed in `backend`.
- `npm test -- App.test.tsx` passed in `frontend` after adding Markdown rendering coverage.

Manual verification:

- Run migrations with `docker compose run --rm migrate`.
- Complete a mock interview and upload all answer recordings.
- Confirm the result page first shows `AI 分析中`.
- In mock mode, confirm the result page updates to fixed mock transcript and improvement suggestion text.
- With `GEMINI_API_KEY` configured, confirm answer audio is analyzed by Gemini and suggestions reflect the related question or job requirements.
- Confirm Markdown in improvement suggestions, such as headings, bold text, lists, and line breaks, is rendered as formatted content on the result page.

## LLM Call Logging

Completed on 2026-06-01.

Implemented:

- Added migration `000004_create_llm_call_logs` for persistent LLM call logs.
- Recorded operation, provider, model, related interview/question/answer IDs, status, latency, input/output/total tokens, and sanitized error details.
- Logged every real Gemini `GenerateContent` attempt for question generation, including retries and fallback model attempts.
- Logged every real Gemini `GenerateContent` attempt for answer analysis, linked to the answer, question, and interview.
- Kept mock generator and mock answer analyzer from writing LLM call logs.
- Kept public API response shapes unchanged and intentionally did not add `request_id`.

Verification:

- `go test ./...` passed in `backend`.
- `docker compose run --rm migrate` applied migration version 4.
- `docker compose exec -T postgres psql -U interview_ai -d interview_ai -c "\d llm_call_logs"` confirmed the table, token columns, indexes, and nullable foreign keys.

## Interview Info Modal Display

Completed on 2026-06-01.

Implemented:

- Updated the preparation/detail page and result page to show job information and user profile through modal dialogs.
- Kept both sections hidden by default until the user clicks `職位資訊` or `個人資訊`.
- Added a `關閉` button and background-click dismissal for each modal.
- Preserved line breaks and allowed long text to wrap inside the modal body.

Verification:

- `npm test -- App.test.tsx -t "starts a ready interview|loads the completed interview result page"` passed in `frontend`.
- `npm test -- App.test.tsx` passed in `frontend`.
- `npm run build` passed in `frontend`.

Manual verification:

- Open an interview preparation/detail page and confirm `職位資訊` and `個人資訊` do not show their full text by default.
- Click each section and confirm a modal opens with the multi-line content preserving line breaks and wrapping within the modal.
- Confirm the modal closes with the `關閉` button.
- Open the result page and confirm clicking outside the modal closes it.

## Interview Session TTS Voice Selection

Completed on 2026-06-02.

Implemented:

- Updated the interview session page to load browser SpeechSynthesis voices.
- Selected a matching voice for the interview question language before speaking.
- Preferred `Google 國語（臺灣）` for Chinese questions, with language-based fallback if that voice is unavailable.
- Applied Chinese question playback tuning with `rate: 1.1` and `pitch: 0.8`.
- Preserved automatic playback, recording handoff, and speech cancellation behavior.

Verification:

- `npm test -- App.test.tsx` passed in `frontend`.
- `npm run build` passed in `frontend`.

Manual verification:

- Create or open an interview using `question_language: zh-TW`.
- Start the session page and confirm the first question is spoken with `Google 國語（臺灣）` when the browser provides it.
- Confirm Chinese playback uses rate `1.1` and pitch `0.8`.
- Repeat with `question_language: en-US` and confirm English questions still use an English voice when available.

## Gemini Question TTS Playback

Completed on 2026-06-02.

Implemented:

- Added `POST /api/interviews/{interview_id}/questions/{question_id}/tts` for on-demand question audio.
- Added Gemini TTS generation through `google.golang.org/genai` v1.58.0 with configurable TTS model, fallback model, and voice.
- Wrapped Gemini inline PCM audio as `audio/wav` for browser playback.
- Validated that requested questions belong to the requested interview before generating audio.
- Added LLM call logging for real Gemini question TTS attempts.
- Updated the session page to try backend Gemini TTS first, then fall back to browser `SpeechSynthesis` when the API key is missing, Gemini is unavailable, or browser audio playback fails.
- Kept generated question audio ephemeral; it is not saved to the database or local storage.
- Updated `.env.example`, `docs/API.md`, and `README.md` with configuration, API usage, and privacy notes.

Verification:

- `go test ./internal/config ./internal/llm ./internal/service ./internal/handler` passed in `backend`.
- `npm test -- App.test.tsx` passed in `frontend`.

Manual verification:

- With empty `GEMINI_API_KEY`, start an interview session and confirm question playback falls back to browser TTS.
- With `GEMINI_API_KEY` configured, start an interview session and confirm the frontend requests `/api/interviews/{interview_id}/questions/{question_id}/tts`.
- Temporarily configure an invalid TTS model and confirm the session still falls back to browser TTS and starts recording after playback.

## Frontend Question Audio Prefetch

Completed on 2026-06-02.

Implemented:

- Added a frontend in-memory question audio cache for generated TTS blobs.
- Added `POST /api/interviews/{interview_id}/questions/tts` so the preparation page can fetch generated audio for every saved question before the session starts while question text remains hidden.
- Updated the preparation page to automatically prepare question audio when questions are ready, store successful WAV blobs in browser memory, and keep showing `題目準備中` until audio preparation completes.
- Updated the preparation page so `題目已準備完成` and `開始模擬面試` appear only after both questions and question audio are ready.
- Updated start behavior so pressing `開始模擬面試` after audio is ready starts the interview and navigates directly to the session first question.
- Kept the preparation page visible during question-audio preparation so users do not see the intermediate `in_progress` detail view before the session route opens.
- Updated the session page to prefetch any missing Gemini TTS audio before playback when an in-progress interview is opened directly or after refresh.
- Updated the session page to play prefetched in-memory audio first, then fall back to browser `SpeechSynthesis` if generated audio is unavailable.
- Added a per-interview prefetch-complete marker so failed prefetches do not cause a new generated-TTS request at the start of every question.
- Kept generated question audio non-persistent: no database columns, no backend audio files, no localStorage, and no IndexedDB.
- Preserved replay behavior so `重新播放題目` uses the same cache-first playback flow.
- Updated `docs/API.md` and `README.md` with the prefetch and privacy behavior.

Verification:

- `npm test -- App.test.tsx` passed in `frontend` with 28 tests.
- `npm run build` passed in `frontend`.
- `go test ./...` passed in `backend`.

Manual verification:

- With `GEMINI_API_KEY` configured, wait on the preparation page and confirm `/api/interviews/{interview_id}/questions/tts` runs before `開始模擬面試` becomes enabled.
- Confirm the page continues showing `題目準備中` while question audio is being prepared and does not show `準備題目語音...`.
- Click `開始模擬面試` after the button is enabled and confirm it navigates directly to the session first question.
- Confirm the session plays the prefetched question audio without re-requesting `/tts` for the same question.
- Refresh or directly open the session page and confirm it prefetches missing question audio before playback starts.
- Temporarily make Gemini TTS unavailable and confirm the session falls back to browser TTS without retrying `/tts` on every question.

## Interview Session End Confirmation

Completed on 2026-06-02.

Implemented:

- Updated the session page `結束面試` control to open a confirmation modal instead of navigating immediately.
- Warned users that the current simulated interview will directly end and unfinished or unuploaded answers will not be retained.
- On confirmation, stopped question playback, discarded any active recording or queued uploads, and returned to the homepage.
- Preserved cancellation behavior so users can close the modal and continue the current session.

Verification:

- `npm test -- App.test.tsx -t "confirms before ending an active interview session and returning home"` passed in `frontend`.
- `npm test -- App.test.tsx` passed in `frontend`.
- `npm run build` passed in `frontend`.

Manual verification:

- Start an interview session and click `結束面試`.
- Confirm a modal appears before navigation and says the simulated interview will directly end.
- Click `取消` and confirm the session remains in progress.
- Click `結束面試` again, then `確認結束`, and confirm the app returns to the homepage.

## Next Step

Post-MVP / Phase 2: continue hardening Gemini answer analysis and consider export/download flows.

Expected work:

- Add retry controls or re-analysis API for failed answer analysis.
- Consider a durable job table if the background queue needs to survive backend restarts.
- Add export/download once the analyzed result format is stable.
