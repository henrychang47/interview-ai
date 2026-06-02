# API

Base URL: `http://localhost:8080`

This API set covers the completed MVP v1 core flow and the post-MVP answer analysis flow: create an interview, generate questions, start the interview, upload answer audio, complete the interview, analyze uploaded answers in the background, and serve uploaded answer audio for the result page.

## Health Check

```http
GET /health
```

Response:

```json
{"status":"ok"}
```

## Create Interview

```http
POST /api/interviews
Content-Type: application/json
```

Request:

```json
{
  "job_title": "後端工程師",
  "job_description": "需要熟悉 Go、PostgreSQL、REST API",
  "user_profile": "有 Java 和 Go 學習經驗，正在準備後端工程師面試",
  "question_count": 3,
  "question_language": "zh-TW"
}
```

Success response:

```http
201 Created
```

```json
{
  "id": "interview_uuid",
  "status": "generating_questions"
}
```

Validation errors:

```json
{"error":"invalid JSON request body"}
```

```json
{"error":"job_title is required"}
```

```json
{"error":"job_description is required"}
```

```json
{"error":"user_profile is required"}
```

```json
{"error":"question_count must be between 1 and 10"}
```

```json
{"error":"question_language must be zh-TW or en-US"}
```

Server error:

```json
{"error":"failed to create interview"}
```

Question generation:

- If `GEMINI_API_KEY` is empty, the backend uses mock interview questions.
- If `GEMINI_API_KEY` is set, the backend calls Gemini to generate questions from `job_title`, `job_description`, and `user_profile`.
- `question_language` must be `zh-TW` or `en-US`. Empty values default to `zh-TW`.
- The create API returns immediately with `generating_questions`; question generation finishes in the background.
- Gemini requests use `google.golang.org/genai` v1.58.0 with `GEMINI_MODEL` first and fall back to `GEMINI_FALLBACK_MODEL` after retrying transient `429` / `RESOURCE_EXHAUSTED` or `503` / `UNAVAILABLE` failures.
- The backend validates the LLM JSON response before saving questions.

## Get Interview

```http
GET /api/interviews/{interview_id}
```

Success response:

```json
{
  "id": "interview_uuid",
  "job_title": "後端工程師",
  "job_description": "需要熟悉 Go、PostgreSQL、REST API",
  "user_profile": "有 Java 和 Go 學習經驗，正在準備後端工程師面試",
  "question_count": 3,
  "question_language": "zh-TW",
  "status": "questions_ready",
  "questions": [],
  "answers": [
    {
      "id": "answer_uuid",
      "question_id": "question_uuid",
      "audio_path": "storage/audio/interview_uuid/question_uuid.webm",
      "transcript_text": null,
      "analysis_status": "pending",
      "improvement_suggestions": null,
      "analysis_error": null,
      "analyzed_at": null,
      "created_at": "2026-05-27T06:30:04Z"
    }
  ]
}
```

`questions` is intentionally empty until the interview status is `in_progress` or `completed`.

Answer analysis statuses:

- `pending`: answer audio was saved and analysis is queued.
- `processing`: the background worker is sending the audio to the configured analyzer.
- `completed`: `transcript_text` and `improvement_suggestions` are available.
- `failed`: analysis failed; `analysis_error` contains a short diagnostic message.

Errors:

```json
{"error":"interview not found"}
```

```json
{"error":"failed to get interview"}
```

## Start Interview

```http
POST /api/interviews/{interview_id}/start
```

Success response:

```json
{
  "id": "interview_uuid",
  "status": "in_progress"
}
```

Errors:

```json
{"error":"interview is not ready to start"}
```

```json
{"error":"failed to start interview"}
```

## Generate Question TTS Audio

```http
POST /api/interviews/{interview_id}/questions/{question_id}/tts
```

Request body: empty.

Success response:

```http
200 OK
Content-Type: audio/wav
```

The response body is a WAV audio file generated for the question text. The audio is generated on demand and is not saved to the database or local storage.

Behavior:

- The backend verifies that the interview exists and that the question belongs to that interview.
- If `GEMINI_API_KEY` is set, the backend uses `google.golang.org/genai` v1.58.0 to call Gemini TTS with `GEMINI_TTS_MODEL`, then `GEMINI_TTS_FALLBACK_MODEL` after transient failures.
- Gemini TTS returns inline PCM audio; the backend wraps it as `audio/wav` for browser playback.
- If the key is missing or Gemini TTS is unavailable, the endpoint returns `503`; the frontend falls back to browser `SpeechSynthesis`.

Errors:

```json
{"error":"interview not found"}
```

```json
{"error":"question not found for interview"}
```

```json
{"error":"question TTS is unavailable"}
```

Curl verification:

```bash
curl -X POST \
  -o question.wav \
  http://localhost:8080/api/interviews/{interview_id}/questions/{question_id}/tts
```

## Upload Answer Audio

```http
POST /api/interviews/{interview_id}/questions/{question_id}/answer
Content-Type: multipart/form-data
```

Form fields:

```text
audio: required WebM audio file
```

Success response:

```http
201 Created
```

```json
{
  "id": "answer_uuid",
  "interview_id": "interview_uuid",
  "question_id": "question_uuid",
  "audio_path": "storage/audio/interview_uuid/question_uuid.webm",
  "transcript_text": null,
  "analysis_status": "pending",
  "improvement_suggestions": null,
  "analysis_error": null,
  "analyzed_at": null
}
```

Completion behavior:

- After each successful upload, the backend checks whether every question in the interview has an answer.
- When all questions have answers, `GET /api/interviews/{interview_id}` returns `"status":"completed"`.
- Answer analysis runs in the backend background worker after the upload response. Poll `GET /api/interviews/{interview_id}` to observe `analysis_status` changes and read completed transcript/suggestions.
- If `GEMINI_API_KEY` is empty, the backend uses mock answer analysis. If it is set, the backend sends the saved WebM audio plus the related `job_title`, `job_description`, `user_profile`, and question text to Gemini using `GEMINI_ANSWER_MODEL` and `GEMINI_ANSWER_FALLBACK_MODEL` when configured.
- Gemini answer analysis treats the job description and user profile as background data only; they must not change the required JSON response shape.

Errors:

```json
{"error":"audio file is required"}
```

```json
{"error":"audio file must be audio/webm"}
```

```json
{"error":"interview not found"}
```

```json
{"error":"question not found for interview"}
```

```json
{"error":"failed to save answer audio"}
```

```json
{"error":"failed to save answer"}
```

Curl verification:

```bash
curl -X POST \
  -F "audio=@answer.webm;type=audio/webm" \
  http://localhost:8080/api/interviews/{interview_id}/questions/{question_id}/answer
```

## Get Answer Audio

```http
GET /audio/{interview_id}/{question_id}.webm
```

Success response:

```http
200 OK
Content-Type: audio/webm
```

The response body is the uploaded WebM audio bytes saved by `POST /api/interviews/{interview_id}/questions/{question_id}/answer`.

Errors:

```http
404 Not Found
```
