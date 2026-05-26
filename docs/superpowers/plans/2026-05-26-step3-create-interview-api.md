# Step 3 Create Interview API Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement `POST /api/interviews` so the backend creates an interview, generates mock questions, stores both in PostgreSQL, and returns the interview id with `questions_ready` status.

**Architecture:** Add a small backend application structure under `backend/internal` while keeping Step 3 limited to the create-interview workflow. The HTTP handler validates JSON and delegates to a service; the service uses a `QuestionGenerator` interface and a pgx-backed repository transaction to insert the interview and questions. The existing `/health` endpoint remains unchanged.

**Tech Stack:** Go 1.22, chi, pgx/v5, PostgreSQL, Docker Compose, Go `testing` + `httptest`.

---

## File Structure

- Modify `backend/go.mod`: add `github.com/jackc/pgx/v5`.
- Modify `backend/cmd/api/main.go`: load `DATABASE_URL`, create a pgx pool, wire routes and dependencies.
- Modify `backend/cmd/api/main_test.go`: keep health test working with the new router constructor.
- Create `backend/internal/config/config.go`: reads `DATABASE_URL`.
- Create `backend/internal/model/interview.go`: request, response, and domain structs.
- Create `backend/internal/llm/question_generator.go`: interface and mock generator.
- Create `backend/internal/repository/interview_repository.go`: pgx transaction that inserts interview and questions.
- Create `backend/internal/service/interview_service.go`: orchestration and validation boundary.
- Create `backend/internal/handler/interview_handler.go`: `POST /api/interviews` HTTP handler.
- Create focused tests for handler validation, mock question generation, service behavior, and repository integration.
- Modify `README.md`: add curl and psql verification commands for Step 3.
- Modify `docs/development-progress.md`: mark Step 3 completed after implementation verification.

## API Contract

Endpoint:

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
HTTP/1.1 201 Created
Content-Type: application/json
```

```json
{
  "id": "interview_uuid",
  "status": "questions_ready"
}
```

Validation errors:

- Invalid JSON: `400 {"error":"invalid JSON request body"}`
- Empty `job_title`: `400 {"error":"job_title is required"}`
- Empty `job_description`: `400 {"error":"job_description is required"}`
- Empty `user_profile`: `400 {"error":"user_profile is required"}`
- `question_count` outside `1..10`: `400 {"error":"question_count must be between 1 and 10"}`

Server errors:

- Generator or DB failure: `500 {"error":"failed to create interview"}`

## Task 1: Add Config and Router Wiring

**Files:**
- Modify: `backend/go.mod`
- Modify: `backend/cmd/api/main.go`
- Modify: `backend/cmd/api/main_test.go`
- Create: `backend/internal/config/config.go`

- [ ] **Step 1: Add pgx dependency**

Run:

```powershell
cd backend
go get github.com/jackc/pgx/v5@v5.7.1
go mod tidy
```

Expected: `backend/go.mod` includes `github.com/jackc/pgx/v5`, and `backend/go.sum` is updated.

- [ ] **Step 2: Create config package**

Create `backend/internal/config/config.go`:

```go
package config

import (
	"errors"
	"os"
)

type Config struct {
	DatabaseURL string
}

func Load() (Config, error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}

	return Config{DatabaseURL: databaseURL}, nil
}
```

- [ ] **Step 3: Refactor router constructor for dependency injection**

Change `backend/cmd/api/main.go` so `newRouter` accepts an optional interview handler dependency:

```go
func newRouter(interviewHandler http.Handler) http.Handler {
	router := chi.NewRouter()

	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	if interviewHandler != nil {
		router.Mount("/api/interviews", interviewHandler)
	}

	return router
}
```

In `main`, load config, connect pgx pool, build repository/service/handler dependencies, and call:

```go
if err := http.ListenAndServe(":8080", newRouter(interviewHandler)); err != nil {
	log.Fatalf("server stopped: %v", err)
}
```

- [ ] **Step 4: Update health test**

Change `backend/cmd/api/main_test.go` to call:

```go
newRouter(nil).ServeHTTP(response, request)
```

- [ ] **Step 5: Run tests**

Run:

```powershell
$env:GOCACHE='D:\projects\interview-ai\.cache\go-build'
$env:GOMODCACHE='D:\projects\interview-ai\.cache\gomod'
go test ./...
```

Expected: existing health test passes.

## Task 2: Add Models and Mock Question Generator

**Files:**
- Create: `backend/internal/model/interview.go`
- Create: `backend/internal/llm/question_generator.go`
- Create: `backend/internal/llm/question_generator_test.go`

- [ ] **Step 1: Write failing generator test**

Create `backend/internal/llm/question_generator_test.go`:

```go
package llm

import (
	"context"
	"testing"
)

func TestMockQuestionGeneratorReturnsRequestedCount(t *testing.T) {
	generator := MockQuestionGenerator{}

	questions, err := generator.GenerateQuestions(context.Background(), GenerateQuestionsInput{
		JobTitle:       "後端工程師",
		JobDescription: "需要熟悉 Go、PostgreSQL、REST API",
		UserProfile:    "有 Java 和 Go 學習經驗",
		QuestionCount:  3,
	})

	if err != nil {
		t.Fatalf("GenerateQuestions returned error: %v", err)
	}
	if len(questions) != 3 {
		t.Fatalf("expected 3 questions, got %d", len(questions))
	}
	for index, question := range questions {
		expectedOrder := index + 1
		if question.Order != expectedOrder {
			t.Fatalf("expected order %d, got %d", expectedOrder, question.Order)
		}
		if question.Text == "" {
			t.Fatalf("expected question text at order %d", expectedOrder)
		}
	}
}
```

Run:

```powershell
go test ./internal/llm
```

Expected: fail because `MockQuestionGenerator` is undefined.

- [ ] **Step 2: Add domain models**

Create `backend/internal/model/interview.go`:

```go
package model

import "time"

const (
	InterviewStatusCreated        = "created"
	InterviewStatusQuestionsReady = "questions_ready"
)

type CreateInterviewRequest struct {
	JobTitle       string `json:"job_title"`
	JobDescription string `json:"job_description"`
	UserProfile    string `json:"user_profile"`
	QuestionCount  int    `json:"question_count"`
}

type CreateInterviewResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type Interview struct {
	ID             string
	JobTitle       string
	JobDescription string
	UserProfile    string
	QuestionCount  int
	Status         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Question struct {
	ID          string
	InterviewID string
	Order       int
	Text        string
	CreatedAt   time.Time
}
```

- [ ] **Step 3: Add generator interface and mock implementation**

Create `backend/internal/llm/question_generator.go`:

```go
package llm

import (
	"context"
	"fmt"
)

type QuestionGenerator interface {
	GenerateQuestions(ctx context.Context, input GenerateQuestionsInput) ([]GeneratedQuestion, error)
}

type GenerateQuestionsInput struct {
	JobTitle       string
	JobDescription string
	UserProfile    string
	QuestionCount  int
}

type GeneratedQuestion struct {
	Order int
	Text  string
}

type MockQuestionGenerator struct{}

func (MockQuestionGenerator) GenerateQuestions(ctx context.Context, input GenerateQuestionsInput) ([]GeneratedQuestion, error) {
	baseQuestions := []string{
		"請介紹你過去與後端開發相關的經驗。",
		"你如何設計一個 REST API？",
		"你使用 PostgreSQL 時會注意哪些事情？",
		"請分享你排查後端服務問題的經驗。",
		"你如何確保 API 的可維護性與可測試性？",
		"請說明你如何處理資料庫交易與錯誤回復。",
		"你如何設計一個可擴充的服務架構？",
		"請分享你與跨職能團隊合作的經驗。",
		"你如何評估系統效能瓶頸？",
		"請描述你學習新技術並應用到專案中的方式。",
	}

	questions := make([]GeneratedQuestion, 0, input.QuestionCount)
	for index := 0; index < input.QuestionCount; index++ {
		text := baseQuestions[index%len(baseQuestions)]
		if input.JobTitle != "" && index >= len(baseQuestions) {
			text = fmt.Sprintf("針對%s，%s", input.JobTitle, text)
		}
		questions = append(questions, GeneratedQuestion{
			Order: index + 1,
			Text:  text,
		})
	}

	return questions, nil
}
```

- [ ] **Step 4: Run generator test**

Run:

```powershell
go test ./internal/llm
```

Expected: pass.

## Task 3: Add Repository Transaction

**Files:**
- Create: `backend/internal/repository/interview_repository.go`
- Create: `backend/internal/repository/interview_repository_test.go`

- [ ] **Step 1: Write repository integration test**

Create `backend/internal/repository/interview_repository_test.go` with a test that:

- reads `DATABASE_URL`
- skips if it is empty
- inserts one interview and three questions
- queries counts from `interviews` and `questions`
- deletes the inserted interview at cleanup

Use this core assertion:

```go
if interviewID == "" {
	t.Fatal("expected interview id")
}
if status != "questions_ready" {
	t.Fatalf("expected questions_ready, got %q", status)
}
```

Run:

```powershell
$env:DATABASE_URL='postgres://interview_ai:interview_ai_dev_password@localhost:5432/interview_ai?sslmode=disable'
go test ./internal/repository
```

Expected: fail because repository is not implemented.

- [ ] **Step 2: Implement repository**

Create `backend/internal/repository/interview_repository.go`:

```go
package repository

import (
	"context"

	"interview-ai/backend/internal/llm"
	"interview-ai/backend/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type InterviewRepository struct {
	pool *pgxpool.Pool
}

func NewInterviewRepository(pool *pgxpool.Pool) *InterviewRepository {
	return &InterviewRepository{pool: pool}
}

func (r *InterviewRepository) CreateWithQuestions(
	ctx context.Context,
	input model.CreateInterviewRequest,
	questions []llm.GeneratedQuestion,
) (model.CreateInterviewResponse, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return model.CreateInterviewResponse{}, err
	}
	defer tx.Rollback(ctx)

	var interviewID string
	err = tx.QueryRow(ctx, `
		INSERT INTO interviews (job_title, job_description, user_profile, question_count, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, input.JobTitle, input.JobDescription, input.UserProfile, input.QuestionCount, model.InterviewStatusQuestionsReady).Scan(&interviewID)
	if err != nil {
		return model.CreateInterviewResponse{}, err
	}

	for _, question := range questions {
		_, err = tx.Exec(ctx, `
			INSERT INTO questions (interview_id, question_order, question_text)
			VALUES ($1, $2, $3)
		`, interviewID, question.Order, question.Text)
		if err != nil {
			return model.CreateInterviewResponse{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return model.CreateInterviewResponse{}, err
	}

	return model.CreateInterviewResponse{
		ID:     interviewID,
		Status: model.InterviewStatusQuestionsReady,
	}, nil
}
```

- [ ] **Step 3: Run repository test**

Run:

```powershell
$env:DATABASE_URL='postgres://interview_ai:interview_ai_dev_password@localhost:5432/interview_ai?sslmode=disable'
go test ./internal/repository
```

Expected: pass after migrations have been applied.

## Task 4: Add Service Validation and Orchestration

**Files:**
- Create: `backend/internal/service/interview_service.go`
- Create: `backend/internal/service/interview_service_test.go`

- [ ] **Step 1: Write service tests**

Create tests for:

- valid request calls generator and repository and returns `questions_ready`
- empty `job_title` returns `job_title is required`
- `question_count` 0 or 11 returns `question_count must be between 1 and 10`

Run:

```powershell
go test ./internal/service
```

Expected: fail because service is not implemented.

- [ ] **Step 2: Implement service**

Create `backend/internal/service/interview_service.go` with:

```go
package service

import (
	"context"
	"errors"
	"strings"

	"interview-ai/backend/internal/llm"
	"interview-ai/backend/internal/model"
)

var (
	ErrJobTitleRequired       = errors.New("job_title is required")
	ErrJobDescriptionRequired = errors.New("job_description is required")
	ErrUserProfileRequired    = errors.New("user_profile is required")
	ErrQuestionCountRange     = errors.New("question_count must be between 1 and 10")
)

type InterviewRepository interface {
	CreateWithQuestions(ctx context.Context, input model.CreateInterviewRequest, questions []llm.GeneratedQuestion) (model.CreateInterviewResponse, error)
}

type InterviewService struct {
	generator  llm.QuestionGenerator
	repository InterviewRepository
}

func NewInterviewService(generator llm.QuestionGenerator, repository InterviewRepository) *InterviewService {
	return &InterviewService{generator: generator, repository: repository}
}

func (s *InterviewService) CreateInterview(ctx context.Context, input model.CreateInterviewRequest) (model.CreateInterviewResponse, error) {
	input.JobTitle = strings.TrimSpace(input.JobTitle)
	input.JobDescription = strings.TrimSpace(input.JobDescription)
	input.UserProfile = strings.TrimSpace(input.UserProfile)

	if input.JobTitle == "" {
		return model.CreateInterviewResponse{}, ErrJobTitleRequired
	}
	if input.JobDescription == "" {
		return model.CreateInterviewResponse{}, ErrJobDescriptionRequired
	}
	if input.UserProfile == "" {
		return model.CreateInterviewResponse{}, ErrUserProfileRequired
	}
	if input.QuestionCount < 1 || input.QuestionCount > 10 {
		return model.CreateInterviewResponse{}, ErrQuestionCountRange
	}

	questions, err := s.generator.GenerateQuestions(ctx, llm.GenerateQuestionsInput{
		JobTitle:       input.JobTitle,
		JobDescription: input.JobDescription,
		UserProfile:    input.UserProfile,
		QuestionCount:  input.QuestionCount,
	})
	if err != nil {
		return model.CreateInterviewResponse{}, err
	}

	return s.repository.CreateWithQuestions(ctx, input, questions)
}
```

- [ ] **Step 3: Run service tests**

Run:

```powershell
go test ./internal/service
```

Expected: pass.

## Task 5: Add HTTP Handler and Route

**Files:**
- Create: `backend/internal/handler/interview_handler.go`
- Create: `backend/internal/handler/interview_handler_test.go`
- Modify: `backend/cmd/api/main.go`

- [ ] **Step 1: Write handler tests**

Create tests for:

- valid JSON returns `201` with `id` and `status`
- invalid JSON returns `400`
- service validation error returns `400`
- unexpected service error returns `500`

Run:

```powershell
go test ./internal/handler
```

Expected: fail because handler is not implemented.

- [ ] **Step 2: Implement handler**

Create `backend/internal/handler/interview_handler.go`:

```go
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"interview-ai/backend/internal/model"
	"interview-ai/backend/internal/service"

	"github.com/go-chi/chi/v5"
)

type InterviewService interface {
	CreateInterview(ctx context.Context, input model.CreateInterviewRequest) (model.CreateInterviewResponse, error)
}

func NewInterviewHandler(interviewService InterviewService) http.Handler {
	router := chi.NewRouter()
	router.Post("/", createInterview(interviewService))
	return router
}

func createInterview(interviewService InterviewService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input model.CreateInterviewRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON request body")
			return
		}

		response, err := interviewService.CreateInterview(r.Context(), input)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrJobTitleRequired),
				errors.Is(err, service.ErrJobDescriptionRequired),
				errors.Is(err, service.ErrUserProfileRequired),
				errors.Is(err, service.ErrQuestionCountRange):
				writeError(w, http.StatusBadRequest, err.Error())
			default:
				log.Printf("create interview: %v", err)
				writeError(w, http.StatusInternalServerError, "failed to create interview")
			}
			return
		}

		writeJSON(w, http.StatusCreated, response)
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("write json response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
```

- [ ] **Step 3: Wire handler in main**

In `backend/cmd/api/main.go`, wire:

```go
cfg, err := config.Load()
pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
interviewRepository := repository.NewInterviewRepository(pool)
questionGenerator := llm.MockQuestionGenerator{}
interviewService := service.NewInterviewService(questionGenerator, interviewRepository)
interviewHandler := handler.NewInterviewHandler(interviewService)
```

Ensure `defer pool.Close()` is called after successful pool creation.

- [ ] **Step 4: Run handler and full backend tests**

Run:

```powershell
go test ./internal/handler
go test ./...
```

Expected: pass.

## Task 6: Verify API Against PostgreSQL

**Files:**
- No production code changes.

- [ ] **Step 1: Ensure database is migrated**

Run:

```powershell
docker compose up -d postgres
docker compose run --rm migrate
```

Expected: migration succeeds or reports no change.

- [ ] **Step 2: Rebuild and start backend**

Run:

```powershell
docker compose up --build -d backend
```

Expected: backend container is running on port `8080`.

- [ ] **Step 3: Call create interview API**

Run:

```powershell
curl.exe -X POST http://localhost:8080/api/interviews `
  -H "Content-Type: application/json" `
  -d "{\"job_title\":\"後端工程師\",\"job_description\":\"需要熟悉 Go、PostgreSQL、REST API\",\"user_profile\":\"有 Java 和 Go 學習經驗，正在準備後端工程師面試\",\"question_count\":3}"
```

Expected response:

```json
{"id":"<uuid>","status":"questions_ready"}
```

- [ ] **Step 4: Verify database rows**

Run:

```powershell
docker compose exec postgres psql -U interview_ai -d interview_ai -c "SELECT id, status, question_count FROM interviews ORDER BY created_at DESC LIMIT 1;"
docker compose exec postgres psql -U interview_ai -d interview_ai -c "SELECT question_order, question_text FROM questions WHERE interview_id = (SELECT id FROM interviews ORDER BY created_at DESC LIMIT 1) ORDER BY question_order;"
```

Expected:

- Latest interview has `status = questions_ready`.
- Latest interview has `question_count = 3`.
- Query returns 3 question rows with order `1`, `2`, `3`.

## Task 7: Update Documentation, Progress, and Commit

**Files:**
- Modify: `README.md`
- Modify: `docs/development-progress.md`

- [ ] **Step 1: Add README API verification**

Add a `Create Interview API` section with the curl command from Task 6 and a psql verification command.

- [ ] **Step 2: Update progress document**

Update current status:

```md
- Current step: Step 3 - 建立面試 API
- Status: Completed
- Last updated: 2026-05-26
```

Change Step 3 in the table to `Completed`, and add:

```md
### Step 3 - 建立面試 API

Completed on 2026-05-26.

Implemented:

- Added `POST /api/interviews`.
- Added mock question generation.
- Persisted interviews and questions to PostgreSQL in one transaction.

Verification:

- `go test ./...` passed.
- `curl POST /api/interviews` returned an interview id with `questions_ready`.
- PostgreSQL contained 1 interview row and the requested number of question rows.
```

- [ ] **Step 3: Run final checks**

Run:

```powershell
go test ./...
npm test
npm run build
docker compose config
```

Expected: all commands exit 0.

- [ ] **Step 4: Commit Step 3**

Run:

```powershell
git add backend README.md docs/development-progress.md
git commit -m "feat: add create interview api"
```

Expected: commit succeeds.

## Assumptions

- Step 3 only implements `POST /api/interviews`; it does not implement `GET /api/interviews/{id}`.
- Step 3 uses `MockQuestionGenerator`; it does not call a real LLM.
- Step 3 stores generated questions but does not return them; Step 4 will add the read API.
- Step 3 keeps the frontend unchanged.
- Step 3 treats `question_count` as `1..10`, matching the migration constraint and MVP limits.
- Existing database rows may remain from manual verification; tests should clean up only rows they insert.
