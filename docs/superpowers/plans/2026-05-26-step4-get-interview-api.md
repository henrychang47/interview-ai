# Step 4 Get Interview API Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement `GET /api/interviews/{id}` so the backend returns one interview with its questions and answers.

**Architecture:** Extend the Step 3 backend slice without adding new MVP scope. The handler parses the route id and maps service errors to HTTP responses; the service delegates read behavior to the repository; the pgx repository fetches interview, questions, and answers from PostgreSQL using ordered queries. Existing `POST /api/interviews` behavior must remain unchanged.

**Tech Stack:** Go 1.22, chi, pgx/v5, PostgreSQL, Docker Compose, Go `testing` + `httptest`.

---

## Scope

Implement only Step 4 from `docs/mvp-spec.md`:

```http
GET /api/interviews/{interview_id}
```

Do not implement:

- `POST /api/interviews/{id}/start`
- answer upload
- audio file serving
- frontend pages
- LLM integration

## API Contract

Endpoint:

```http
GET /api/interviews/{interview_id}
```

Success response:

```http
HTTP/1.1 200 OK
Content-Type: application/json
```

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
  "answers": [
    {
      "id": "answer_uuid_1",
      "question_id": "question_uuid_1",
      "audio_path": "storage/audio/interview_uuid/question_uuid_1.webm",
      "transcript_text": null,
      "created_at": "2026-05-26T12:00:00Z"
    }
  ]
}
```

Errors:

- Empty path id should not match this route.
- Not found: `404 {"error":"interview not found"}`
- DB failure: `500 {"error":"failed to get interview"}`

## File Structure

- Modify `backend/internal/model/interview.go`: add detail response models and answer model.
- Modify `backend/internal/repository/interview_repository.go`: add `GetByID`.
- Modify `backend/internal/repository/interview_repository_test.go`: add integration test for interview detail read.
- Modify `backend/internal/service/interview_service.go`: add `GetInterview` and `ErrInterviewNotFound`.
- Modify `backend/internal/service/interview_service_test.go`: add service tests for success and not found propagation.
- Modify `backend/internal/handler/interview_handler.go`: add `GET /{interviewID}` route and HTTP error mapping.
- Modify `backend/internal/handler/interview_handler_test.go`: add handler tests for success, not found, and server error.
- Modify `docs/API.md`: document `GET /api/interviews/{id}`.
- Modify `docs/development-progress.md`: mark Step 4 completed after verification.
- Modify `docs/DEVELOPMENT_PLAN.md`: update current status and progress table.
- Modify `README.md`: add curl/PowerShell verification for querying an interview.

## Task 1: Add Detail Models

**Files:**
- Modify: `backend/internal/model/interview.go`

- [ ] **Step 1: Add response/domain structs**

Append these types to `backend/internal/model/interview.go`:

```go
type InterviewDetail struct {
	ID             string
	JobTitle       string
	JobDescription string
	UserProfile    string
	QuestionCount  int
	Status         string
	Questions      []Question
	Answers        []Answer
}

type Answer struct {
	ID             string
	InterviewID    string
	QuestionID     string
	AudioPath      *string
	TranscriptText *string
	CreatedAt      time.Time
}

type InterviewDetailResponse struct {
	ID             string             `json:"id"`
	JobTitle       string             `json:"job_title"`
	JobDescription string             `json:"job_description"`
	UserProfile    string             `json:"user_profile"`
	QuestionCount  int                `json:"question_count"`
	Status         string             `json:"status"`
	Questions      []QuestionResponse `json:"questions"`
	Answers        []AnswerResponse   `json:"answers"`
}

type QuestionResponse struct {
	ID    string `json:"id"`
	Order int    `json:"order"`
	Text  string `json:"text"`
}

type AnswerResponse struct {
	ID             string  `json:"id"`
	QuestionID     string  `json:"question_id"`
	AudioPath      *string `json:"audio_path"`
	TranscriptText *string `json:"transcript_text"`
	CreatedAt      string  `json:"created_at"`
}
```

- [ ] **Step 2: Run model package tests**

Run:

```powershell
$env:GOCACHE='D:\projects\interview-ai\.cache\go-build'
$env:GOMODCACHE='D:\projects\interview-ai\.cache\gomod'
go test ./internal/model
```

Expected: package compiles with `[no test files]`.

## Task 2: Add Repository Read Query

**Files:**
- Modify: `backend/internal/repository/interview_repository_test.go`
- Modify: `backend/internal/repository/interview_repository.go`

- [ ] **Step 1: Write failing repository integration test**

Append this test to `backend/internal/repository/interview_repository_test.go`:

```go
func TestGetByIDReturnsInterviewQuestionsAndAnswers(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect database: %v", err)
	}
	defer pool.Close()

	repository := NewInterviewRepository(pool)
	created, err := repository.CreateWithQuestions(ctx, model.CreateInterviewRequest{
		JobTitle:       "後端工程師",
		JobDescription: "需要熟悉 Go、PostgreSQL、REST API",
		UserProfile:    "有 Java 和 Go 學習經驗",
		QuestionCount:  2,
	}, []llm.GeneratedQuestion{
		{Order: 1, Text: "問題一"},
		{Order: 2, Text: "問題二"},
	})
	if err != nil {
		t.Fatalf("CreateWithQuestions returned error: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM interviews WHERE id = $1", created.ID)
	})

	var firstQuestionID string
	if err := pool.QueryRow(ctx, `
		SELECT id
		FROM questions
		WHERE interview_id = $1 AND question_order = 1
	`, created.ID).Scan(&firstQuestionID); err != nil {
		t.Fatalf("query first question id: %v", err)
	}

	audioPath := "storage/audio/" + created.ID + "/" + firstQuestionID + ".webm"
	_, err = pool.Exec(ctx, `
		INSERT INTO answers (interview_id, question_id, audio_path, transcript_text)
		VALUES ($1, $2, $3, NULL)
	`, created.ID, firstQuestionID, audioPath)
	if err != nil {
		t.Fatalf("insert answer: %v", err)
	}

	detail, err := repository.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}

	if detail.ID != created.ID {
		t.Fatalf("expected interview id %q, got %q", created.ID, detail.ID)
	}
	if detail.JobTitle != "後端工程師" {
		t.Fatalf("expected job title, got %q", detail.JobTitle)
	}
	if len(detail.Questions) != 2 {
		t.Fatalf("expected 2 questions, got %d", len(detail.Questions))
	}
	if detail.Questions[0].Order != 1 || detail.Questions[0].Text != "問題一" {
		t.Fatalf("expected first ordered question, got %+v", detail.Questions[0])
	}
	if len(detail.Answers) != 1 {
		t.Fatalf("expected 1 answer, got %d", len(detail.Answers))
	}
	if detail.Answers[0].QuestionID != firstQuestionID {
		t.Fatalf("expected answer question id %q, got %q", firstQuestionID, detail.Answers[0].QuestionID)
	}
	if detail.Answers[0].AudioPath == nil || *detail.Answers[0].AudioPath != audioPath {
		t.Fatalf("expected audio path %q, got %+v", audioPath, detail.Answers[0].AudioPath)
	}
	if detail.Answers[0].TranscriptText != nil {
		t.Fatalf("expected nil transcript text, got %+v", detail.Answers[0].TranscriptText)
	}
}
```

- [ ] **Step 2: Run repository test to verify it fails**

Run:

```powershell
$env:GOCACHE='D:\projects\interview-ai\.cache\go-build'
$env:GOMODCACHE='D:\projects\interview-ai\.cache\gomod'
$env:DATABASE_URL='postgres://interview_ai:interview_ai_dev_password@localhost:5432/interview_ai?sslmode=disable'
go test ./internal/repository
```

Expected: fail with `repository.GetByID undefined`.

- [ ] **Step 3: Implement repository `GetByID`**

Add imports to `backend/internal/repository/interview_repository.go`:

```go
import (
	"context"
	"errors"

	"interview-ai/backend/internal/llm"
	"interview-ai/backend/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)
```

Append this method:

```go
func (r *InterviewRepository) GetByID(ctx context.Context, interviewID string) (model.InterviewDetail, error) {
	var detail model.InterviewDetail
	err := r.pool.QueryRow(ctx, `
		SELECT id, job_title, job_description, user_profile, question_count, status
		FROM interviews
		WHERE id = $1
	`, interviewID).Scan(
		&detail.ID,
		&detail.JobTitle,
		&detail.JobDescription,
		&detail.UserProfile,
		&detail.QuestionCount,
		&detail.Status,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.InterviewDetail{}, pgx.ErrNoRows
		}
		return model.InterviewDetail{}, err
	}

	questionRows, err := r.pool.Query(ctx, `
		SELECT id, interview_id, question_order, question_text, created_at
		FROM questions
		WHERE interview_id = $1
		ORDER BY question_order
	`, interviewID)
	if err != nil {
		return model.InterviewDetail{}, err
	}
	defer questionRows.Close()

	questions, err := pgx.CollectRows(questionRows, pgx.RowToStructByName[model.Question])
	if err != nil {
		return model.InterviewDetail{}, err
	}

	answerRows, err := r.pool.Query(ctx, `
		SELECT id, interview_id, question_id, audio_path, transcript_text, created_at
		FROM answers
		WHERE interview_id = $1
		ORDER BY created_at
	`, interviewID)
	if err != nil {
		return model.InterviewDetail{}, err
	}
	defer answerRows.Close()

	answers, err := pgx.CollectRows(answerRows, pgx.RowToStructByName[model.Answer])
	if err != nil {
		return model.InterviewDetail{}, err
	}

	detail.Questions = questions
	detail.Answers = answers
	return detail, nil
}
```

- [ ] **Step 4: If struct mapping fails, replace with explicit scans**

If `pgx.RowToStructByName` cannot map `question_order` to `Order` or `question_text` to `Text`, replace the question collection block with:

```go
questions := make([]model.Question, 0)
for questionRows.Next() {
	var question model.Question
	if err := questionRows.Scan(
		&question.ID,
		&question.InterviewID,
		&question.Order,
		&question.Text,
		&question.CreatedAt,
	); err != nil {
		return model.InterviewDetail{}, err
	}
	questions = append(questions, question)
}
if err := questionRows.Err(); err != nil {
	return model.InterviewDetail{}, err
}
```

Replace the answer collection block with:

```go
answers := make([]model.Answer, 0)
for answerRows.Next() {
	var answer model.Answer
	if err := answerRows.Scan(
		&answer.ID,
		&answer.InterviewID,
		&answer.QuestionID,
		&answer.AudioPath,
		&answer.TranscriptText,
		&answer.CreatedAt,
	); err != nil {
		return model.InterviewDetail{}, err
	}
	answers = append(answers, answer)
}
if err := answerRows.Err(); err != nil {
	return model.InterviewDetail{}, err
}
```

- [ ] **Step 5: Run repository test to verify it passes**

Run:

```powershell
$env:GOCACHE='D:\projects\interview-ai\.cache\go-build'
$env:GOMODCACHE='D:\projects\interview-ai\.cache\gomod'
$env:DATABASE_URL='postgres://interview_ai:interview_ai_dev_password@localhost:5432/interview_ai?sslmode=disable'
go test ./internal/repository
```

Expected: repository tests pass when local PostgreSQL is reachable; otherwise record the connection failure and use Docker verification in Task 6.

## Task 3: Add Service Read Method and Response Mapping

**Files:**
- Modify: `backend/internal/service/interview_service_test.go`
- Modify: `backend/internal/service/interview_service.go`

- [ ] **Step 1: Write failing service tests**

Append these tests to `backend/internal/service/interview_service_test.go`:

```go
func TestGetInterviewReturnsDetailResponse(t *testing.T) {
	audioPath := "storage/audio/interview-id/question-id.webm"
	createdAt := time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC)
	repository := &stubInterviewRepository{
		detail: model.InterviewDetail{
			ID:             "interview-id",
			JobTitle:       "後端工程師",
			JobDescription: "需要熟悉 Go",
			UserProfile:    "有 Go 學習經驗",
			QuestionCount:  1,
			Status:         model.InterviewStatusQuestionsReady,
			Questions: []model.Question{
				{ID: "question-id", InterviewID: "interview-id", Order: 1, Text: "問題一"},
			},
			Answers: []model.Answer{
				{ID: "answer-id", InterviewID: "interview-id", QuestionID: "question-id", AudioPath: &audioPath, TranscriptText: nil, CreatedAt: createdAt},
			},
		},
	}
	service := NewInterviewService(&stubQuestionGenerator{}, repository)

	response, err := service.GetInterview(context.Background(), "interview-id")

	if err != nil {
		t.Fatalf("GetInterview returned error: %v", err)
	}
	if repository.requestedID != "interview-id" {
		t.Fatalf("expected repository lookup id, got %q", repository.requestedID)
	}
	if response.ID != "interview-id" {
		t.Fatalf("expected interview id, got %q", response.ID)
	}
	if len(response.Questions) != 1 || response.Questions[0].Text != "問題一" {
		t.Fatalf("expected mapped questions, got %+v", response.Questions)
	}
	if len(response.Answers) != 1 {
		t.Fatalf("expected 1 answer, got %d", len(response.Answers))
	}
	if response.Answers[0].AudioPath == nil || *response.Answers[0].AudioPath != audioPath {
		t.Fatalf("expected mapped audio path, got %+v", response.Answers[0].AudioPath)
	}
	if response.Answers[0].TranscriptText != nil {
		t.Fatalf("expected nil transcript text, got %+v", response.Answers[0].TranscriptText)
	}
	if response.Answers[0].CreatedAt != "2026-05-26T12:00:00Z" {
		t.Fatalf("expected RFC3339 created_at, got %q", response.Answers[0].CreatedAt)
	}
}

func TestGetInterviewReturnsNotFound(t *testing.T) {
	repository := &stubInterviewRepository{getErr: ErrInterviewNotFound}
	service := NewInterviewService(&stubQuestionGenerator{}, repository)

	_, err := service.GetInterview(context.Background(), "missing-id")

	if !errors.Is(err, ErrInterviewNotFound) {
		t.Fatalf("expected ErrInterviewNotFound, got %v", err)
	}
}
```

Update imports in `backend/internal/service/interview_service_test.go`:

```go
import (
	"context"
	"errors"
	"testing"
	"time"

	"interview-ai/backend/internal/llm"
	"interview-ai/backend/internal/model"
)
```

Extend `stubInterviewRepository` in the same file:

```go
type stubInterviewRepository struct {
	input       model.CreateInterviewRequest
	questions   []llm.GeneratedQuestion
	detail      model.InterviewDetail
	getErr      error
	requestedID string
}
```

Add method:

```go
func (s *stubInterviewRepository) GetByID(ctx context.Context, interviewID string) (model.InterviewDetail, error) {
	s.requestedID = interviewID
	return s.detail, s.getErr
}
```

- [ ] **Step 2: Run service test to verify it fails**

Run:

```powershell
$env:GOCACHE='D:\projects\interview-ai\.cache\go-build'
$env:GOMODCACHE='D:\projects\interview-ai\.cache\gomod'
go test ./internal/service
```

Expected: fail with `service.GetInterview undefined` and/or `ErrInterviewNotFound undefined`.

- [ ] **Step 3: Implement service method**

Add import to `backend/internal/service/interview_service.go`:

```go
import (
	"context"
	"errors"
	"strings"
	"time"

	"interview-ai/backend/internal/llm"
	"interview-ai/backend/internal/model"

	"github.com/jackc/pgx/v5"
)
```

Add error:

```go
ErrInterviewNotFound = errors.New("interview not found")
```

Extend repository interface:

```go
type InterviewRepository interface {
	CreateWithQuestions(ctx context.Context, input model.CreateInterviewRequest, questions []llm.GeneratedQuestion) (model.CreateInterviewResponse, error)
	GetByID(ctx context.Context, interviewID string) (model.InterviewDetail, error)
}
```

Append method and mapper:

```go
func (s *InterviewService) GetInterview(ctx context.Context, interviewID string) (model.InterviewDetailResponse, error) {
	detail, err := s.repository.GetByID(ctx, interviewID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, ErrInterviewNotFound) {
			return model.InterviewDetailResponse{}, ErrInterviewNotFound
		}
		return model.InterviewDetailResponse{}, err
	}

	return mapInterviewDetailResponse(detail), nil
}

func mapInterviewDetailResponse(detail model.InterviewDetail) model.InterviewDetailResponse {
	questions := make([]model.QuestionResponse, 0, len(detail.Questions))
	for _, question := range detail.Questions {
		questions = append(questions, model.QuestionResponse{
			ID:    question.ID,
			Order: question.Order,
			Text:  question.Text,
		})
	}

	answers := make([]model.AnswerResponse, 0, len(detail.Answers))
	for _, answer := range detail.Answers {
		answers = append(answers, model.AnswerResponse{
			ID:             answer.ID,
			QuestionID:     answer.QuestionID,
			AudioPath:      answer.AudioPath,
			TranscriptText: answer.TranscriptText,
			CreatedAt:      answer.CreatedAt.Format(time.RFC3339),
		})
	}

	return model.InterviewDetailResponse{
		ID:             detail.ID,
		JobTitle:       detail.JobTitle,
		JobDescription: detail.JobDescription,
		UserProfile:    detail.UserProfile,
		QuestionCount:  detail.QuestionCount,
		Status:         detail.Status,
		Questions:      questions,
		Answers:        answers,
	}
}
```

- [ ] **Step 4: Run service test to verify it passes**

Run:

```powershell
$env:GOCACHE='D:\projects\interview-ai\.cache\go-build'
$env:GOMODCACHE='D:\projects\interview-ai\.cache\gomod'
go test ./internal/service
```

Expected: pass.

## Task 4: Add HTTP Handler Route

**Files:**
- Modify: `backend/internal/handler/interview_handler_test.go`
- Modify: `backend/internal/handler/interview_handler.go`

- [ ] **Step 1: Write failing handler tests**

Append these tests to `backend/internal/handler/interview_handler_test.go`:

```go
func TestGetInterviewReturnsDetail(t *testing.T) {
	handler := NewInterviewHandler(&stubInterviewService{
		detailResponse: model.InterviewDetailResponse{
			ID:             "interview-id",
			JobTitle:       "後端工程師",
			JobDescription: "需要熟悉 Go",
			UserProfile:    "有 Go 學習經驗",
			QuestionCount:  1,
			Status:         model.InterviewStatusQuestionsReady,
			Questions: []model.QuestionResponse{
				{ID: "question-id", Order: 1, Text: "問題一"},
			},
			Answers: []model.AnswerResponse{},
		},
	})
	request := httptest.NewRequest(http.MethodGet, "/interview-id", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	var body model.InterviewDetailResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected JSON response, got error: %v", err)
	}
	if body.ID != "interview-id" {
		t.Fatalf("expected interview id, got %q", body.ID)
	}
	if len(body.Questions) != 1 || body.Questions[0].Text != "問題一" {
		t.Fatalf("expected questions response, got %+v", body.Questions)
	}
}

func TestGetInterviewReturnsNotFound(t *testing.T) {
	handler := NewInterviewHandler(&stubInterviewService{getErr: service.ErrInterviewNotFound})
	request := httptest.NewRequest(http.MethodGet, "/missing-id", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	assertErrorResponse(t, response, http.StatusNotFound, "interview not found")
}

func TestGetInterviewReturnsServerError(t *testing.T) {
	handler := NewInterviewHandler(&stubInterviewService{getErr: errors.New("db failed")})
	request := httptest.NewRequest(http.MethodGet, "/interview-id", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	assertErrorResponse(t, response, http.StatusInternalServerError, "failed to get interview")
}
```

Extend `stubInterviewService`:

```go
type stubInterviewService struct {
	response       model.CreateInterviewResponse
	err            error
	detailResponse model.InterviewDetailResponse
	getErr         error
}
```

Add method:

```go
func (s *stubInterviewService) GetInterview(ctx context.Context, interviewID string) (model.InterviewDetailResponse, error) {
	return s.detailResponse, s.getErr
}
```

- [ ] **Step 2: Run handler test to verify it fails**

Run:

```powershell
$env:GOCACHE='D:\projects\interview-ai\.cache\go-build'
$env:GOMODCACHE='D:\projects\interview-ai\.cache\gomod'
go test ./internal/handler
```

Expected: fail because `InterviewService` lacks `GetInterview` and the route is not implemented.

- [ ] **Step 3: Implement handler route**

Update `InterviewService` interface in `backend/internal/handler/interview_handler.go`:

```go
type InterviewService interface {
	CreateInterview(ctx context.Context, input model.CreateInterviewRequest) (model.CreateInterviewResponse, error)
	GetInterview(ctx context.Context, interviewID string) (model.InterviewDetailResponse, error)
}
```

Update router:

```go
func NewInterviewHandler(interviewService InterviewService) http.Handler {
	router := chi.NewRouter()
	router.Post("/", createInterview(interviewService))
	router.Get("/{interviewID}", getInterview(interviewService))
	return router
}
```

Append handler:

```go
func getInterview(interviewService InterviewService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		interviewID := chi.URLParam(r, "interviewID")
		response, err := interviewService.GetInterview(r.Context(), interviewID)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrInterviewNotFound):
				writeError(w, http.StatusNotFound, err.Error())
			default:
				log.Printf("get interview: %v", err)
				writeError(w, http.StatusInternalServerError, "failed to get interview")
			}
			return
		}

		writeJSON(w, http.StatusOK, response)
	}
}
```

- [ ] **Step 4: Run handler test to verify it passes**

Run:

```powershell
$env:GOCACHE='D:\projects\interview-ai\.cache\go-build'
$env:GOMODCACHE='D:\projects\interview-ai\.cache\gomod'
go test ./internal/handler
```

Expected: pass.

## Task 5: Run Full Backend Tests

**Files:**
- No production code changes.

- [ ] **Step 1: Format Go files**

Run:

```powershell
gofmt -w cmd\api\main.go cmd\api\main_test.go internal\config\config.go internal\model\interview.go internal\llm\question_generator.go internal\llm\question_generator_test.go internal\repository\interview_repository.go internal\repository\interview_repository_test.go internal\service\interview_service.go internal\service\interview_service_test.go internal\handler\interview_handler.go internal\handler\interview_handler_test.go
```

Expected: command exits 0.

- [ ] **Step 2: Run all backend tests with repository integration skipped**

Run:

```powershell
$env:GOCACHE='D:\projects\interview-ai\.cache\go-build'
$env:GOMODCACHE='D:\projects\interview-ai\.cache\gomod'
Remove-Item Env:\DATABASE_URL -ErrorAction SilentlyContinue
go test ./...
```

Expected: all backend tests pass; repository integration tests that require `DATABASE_URL` are skipped.

## Task 6: Verify API Against PostgreSQL

**Files:**
- No production code changes.

- [ ] **Step 1: Ensure database is migrated**

Run:

```powershell
docker compose up -d postgres
docker compose run --rm migrate
```

Expected: migration succeeds or reports `no change`.

- [ ] **Step 2: Rebuild and start backend**

Run:

```powershell
docker compose up --build -d backend
```

Expected: backend container is running on port `8080`.

- [ ] **Step 3: Create an interview**

Run:

```powershell
$body = @{
  job_title = '後端工程師'
  job_description = '需要熟悉 Go、PostgreSQL、REST API'
  user_profile = '有 Java 和 Go 學習經驗，正在準備後端工程師面試'
  question_count = 3
} | ConvertTo-Json -Compress

$created = Invoke-RestMethod -Uri http://localhost:8080/api/interviews `
  -Method Post `
  -ContentType 'application/json' `
  -Body $body

$created
```

Expected: output contains `id` and `status = questions_ready`.

- [ ] **Step 4: Query the created interview**

Run:

```powershell
Invoke-RestMethod -Uri "http://localhost:8080/api/interviews/$($created.id)" `
  -Method Get |
  ConvertTo-Json -Depth 5
```

Expected:

- response contains the same `id`
- response contains `job_title`
- response contains `questions`
- `questions` has 3 rows ordered `1`, `2`, `3`
- `answers` is an empty array

- [ ] **Step 5: Verify not found behavior**

Run:

```powershell
try {
  Invoke-RestMethod -Uri "http://localhost:8080/api/interviews/00000000-0000-0000-0000-000000000000" -Method Get
} catch {
  $_.Exception.Response.StatusCode.value__
}
```

Expected: `404`.

## Task 7: Update Documentation and Progress

**Files:**
- Modify: `docs/API.md`
- Modify: `README.md`
- Modify: `docs/development-progress.md`
- Modify: `docs/DEVELOPMENT_PLAN.md`

- [ ] **Step 1: Update API docs**

Append this section to `docs/API.md`:

````md
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
````

- [ ] **Step 2: Update README verification**

Add a `Get Interview API` verification snippet after the create interview verification:

````md
Get Interview API:

```powershell
Invoke-RestMethod -Uri "http://localhost:8080/api/interviews/<interview_uuid>" `
  -Method Get |
  ConvertTo-Json -Depth 5
```
````

- [ ] **Step 3: Update progress documents**

In `docs/development-progress.md`:

```md
- Current step: Step 4 - 查詢面試 API
- Status: Completed
- Last updated: 2026-05-26
```

Change Step 4 table status to `Completed`.

Add:

```md
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
```

In `docs/DEVELOPMENT_PLAN.md`, update current status and Step 4 table status to `Completed`, then set next step to Step 5.

## Task 8: Final Checks and Commit

**Files:**
- All Step 4 files.

- [ ] **Step 1: Run final checks**

Run:

```powershell
$env:GOCACHE='D:\projects\interview-ai\.cache\go-build'
$env:GOMODCACHE='D:\projects\interview-ai\.cache\gomod'
Remove-Item Env:\DATABASE_URL -ErrorAction SilentlyContinue
go test ./...
```

Run from repo root:

```powershell
C:\nvm4w\nodejs\npm.cmd test --prefix frontend
C:\nvm4w\nodejs\npm.cmd run build --prefix frontend
docker compose config
git diff --check
```

Expected: all commands exit 0.

- [ ] **Step 2: Review changed files**

Run:

```powershell
git status --short
git diff --stat
```

Expected: changed files are limited to Step 4 backend code/tests and docs.

- [ ] **Step 3: Commit Step 4**

Run:

```powershell
git add backend README.md docs/API.md docs/DEVELOPMENT_PLAN.md docs/development-progress.md
git commit -m "feat: add get interview api"
```

Expected: commit succeeds.

## Assumptions

- Step 4 returns `answers: []` when no answers exist.
- Step 4 does not create answers; answer creation belongs to Step 10.
- Step 4 does not validate UUID format separately. Unknown or malformed ids are treated as not found or DB errors depending on pgx/PostgreSQL behavior. If malformed UUID returns a PostgreSQL invalid input syntax error, map it to `404 {"error":"interview not found"}` only if implementation confirms that is safe and covered by a test.
- Step 4 keeps the frontend unchanged.
- Existing manual verification rows may remain in the database.

## Self-Review

- Spec coverage: Covers MVP Step 4 only: `GET /api/interviews/{id}` returns interview, questions, and answers.
- Scope check: Does not implement Step 5 frontend or later backend endpoints.
- Placeholder scan: No TBD/TODO placeholders remain.
- Type consistency: `InterviewDetail`, `QuestionResponse`, and `AnswerResponse` are used consistently across repository, service, and handler tasks.
