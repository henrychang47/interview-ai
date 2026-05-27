# API

Base URL: `http://localhost:8080`

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
  "question_count": 3
}
```

Success response:

```http
201 Created
```

```json
{
  "id": "interview_uuid",
  "status": "questions_ready"
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

Server error:

```json
{"error":"failed to create interview"}
```

Question generation:

- If `GEMINI_API_KEY` is empty, the backend uses mock interview questions.
- If `GEMINI_API_KEY` is set, the backend calls Gemini to generate questions from `job_title`, `job_description`, and `user_profile`.
- Gemini requests use `GEMINI_MODEL` first and fall back to `GEMINI_FALLBACK_MODEL` after retrying transient `429` or `503` failures.
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
  "status": "questions_ready",
  "questions": [
    {
      "id": "question_uuid_1",
      "order": 1,
      "text": "請介紹你過去與後端開發相關的經驗。"
    }
  ],
  "answers": []
}
```

Errors:

```json
{"error":"interview not found"}
```

```json
{"error":"failed to get interview"}
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
  "transcript_text": null
}
```

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
