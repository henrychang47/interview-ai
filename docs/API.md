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
