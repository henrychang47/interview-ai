# Immersive Interview Flow Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the two-stage interview setup, asynchronous question preparation, hidden-question prep flow, and immersive auto-play/auto-record interview session.

**Architecture:** Keep the current Go/chi/PostgreSQL backend and React/Vite frontend. Add one database migration for `question_language` and `generating_questions`; move question generation behind a background service flow; keep upload queue persistence in React component memory only. Keep the API surface small: create, poll get, start, upload answer, get result.

**Tech Stack:** Go, chi, pgx, PostgreSQL migrations, React 18, TypeScript, Vite, Tailwind CSS, Vitest, Testing Library, browser SpeechSynthesis, browser MediaRecorder.

---

## File Structure

- Modify `backend/migrations/000001_create_interview_tables.up.sql` and `.down.sql`: add `question_language`, allow `generating_questions`.
- Modify `backend/internal/model/interview.go`: add language/status constants and response fields.
- Modify `backend/internal/llm/question_generator.go`: add `QuestionLanguage` to input, produce Chinese or English mock questions.
- Modify `backend/internal/llm/gemini_question_generator.go`: pass language into the prompt.
- Modify `backend/internal/service/interview_service.go`: validate language, create interview in `generating_questions`, run question generation in background, hide questions before start, start interview.
- Modify `backend/internal/repository/interview_repository.go`: split create from question insertion, update statuses, start interview.
- Modify `backend/internal/handler/interview_handler.go`: add `POST /api/interviews/{interviewID}/start`, expose validation/status errors.
- Modify backend tests in existing test files for service, repository, handler, llm, and main router.
- Modify `frontend/src/types/interview.ts`: add `question_language`.
- Modify `frontend/src/api/interviews.ts`: add `startInterview`.
- Modify `frontend/src/pages/NewInterviewPage.tsx`: convert to two-stage wizard with microphone test.
- Modify `frontend/src/pages/InterviewDetailPage.tsx`: convert detail into preparation/polling page.
- Modify `frontend/src/pages/InterviewSessionPage.tsx`: replace manual controls with state machine and in-memory upload queue.
- Modify `frontend/src/App.test.tsx`: update route behavior tests.
- Modify `.env.example`, `docs/API.md`, `docs/DEVELOPMENT_PLAN.md`, and `README.md`.

---

## Task 1: Backend Data Model, Language, and Async Question Generation

**Files:**
- Modify: `backend/migrations/000001_create_interview_tables.up.sql`
- Modify: `backend/migrations/000001_create_interview_tables.down.sql`
- Modify: `backend/internal/model/interview.go`
- Modify: `backend/internal/llm/question_generator.go`
- Modify: `backend/internal/llm/gemini_question_generator.go`
- Modify: `backend/internal/service/interview_service.go`
- Modify: `backend/internal/repository/interview_repository.go`
- Test: `backend/internal/service/interview_service_test.go`
- Test: `backend/internal/repository/interview_repository_test.go`
- Test: `backend/internal/llm/gemini_question_generator_test.go`

- [ ] **Step 1: Write failing service tests for async create and language validation**

Add these tests to `backend/internal/service/interview_service_test.go`. Update the existing `stubInterviewRepository` to support the new methods shown in Step 5.

```go
func TestCreateInterviewCreatesGeneratingInterviewAndStartsQuestionGeneration(t *testing.T) {
	generator := &stubQuestionGenerator{}
	repository := &stubInterviewRepository{}
	service := NewInterviewServiceWithRunner(generator, repository, func(task func()) { task() })

	response, err := service.CreateInterview(context.Background(), model.CreateInterviewRequest{
		JobTitle:         " 後端工程師 ",
		JobDescription:   " 需要熟悉 Go、PostgreSQL、REST API ",
		UserProfile:      " 有 Java 和 Go 學習經驗 ",
		QuestionCount:    3,
		QuestionLanguage: model.QuestionLanguageZhTW,
	})

	if err != nil {
		t.Fatalf("CreateInterview returned error: %v", err)
	}
	if response.ID != "interview-id" {
		t.Fatalf("expected interview id, got %q", response.ID)
	}
	if response.Status != model.InterviewStatusGeneratingQuestions {
		t.Fatalf("expected generating_questions, got %q", response.Status)
	}
	if generator.input.QuestionLanguage != model.QuestionLanguageZhTW {
		t.Fatalf("expected zh-TW generator language, got %q", generator.input.QuestionLanguage)
	}
	if len(repository.savedQuestions) != 3 {
		t.Fatalf("expected 3 generated questions saved, got %d", len(repository.savedQuestions))
	}
	if repository.savedQuestionsInterviewID != "interview-id" {
		t.Fatalf("expected generated questions saved for interview-id, got %q", repository.savedQuestionsInterviewID)
	}
}

func TestCreateInterviewDefaultsQuestionLanguageToZhTW(t *testing.T) {
	generator := &stubQuestionGenerator{}
	repository := &stubInterviewRepository{}
	service := NewInterviewServiceWithRunner(generator, repository, func(task func()) { task() })

	_, err := service.CreateInterview(context.Background(), validCreateInterviewRequest(func(input *model.CreateInterviewRequest) {
		input.QuestionLanguage = ""
	}))

	if err != nil {
		t.Fatalf("CreateInterview returned error: %v", err)
	}
	if repository.input.QuestionLanguage != model.QuestionLanguageZhTW {
		t.Fatalf("expected default language zh-TW, got %q", repository.input.QuestionLanguage)
	}
}

func TestCreateInterviewRejectsUnsupportedQuestionLanguage(t *testing.T) {
	service := NewInterviewServiceWithRunner(&stubQuestionGenerator{}, &stubInterviewRepository{}, func(task func()) { task() })

	_, err := service.CreateInterview(context.Background(), validCreateInterviewRequest(func(input *model.CreateInterviewRequest) {
		input.QuestionLanguage = "ja-JP"
	}))

	if !errors.Is(err, ErrQuestionLanguageUnsupported) {
		t.Fatalf("expected ErrQuestionLanguageUnsupported, got %v", err)
	}
}

func TestCreateInterviewMarksQuestionGenerationFailed(t *testing.T) {
	generator := &stubQuestionGenerator{err: errors.New("generator failed")}
	repository := &stubInterviewRepository{}
	service := NewInterviewServiceWithRunner(generator, repository, func(task func()) { task() })

	_, err := service.CreateInterview(context.Background(), validCreateInterviewRequest(func(input *model.CreateInterviewRequest) {
		input.QuestionLanguage = model.QuestionLanguageEnUS
	}))

	if err != nil {
		t.Fatalf("CreateInterview should return created interview before background failure, got %v", err)
	}
	if repository.failedInterviewID != "interview-id" {
		t.Fatalf("expected interview marked failed, got %q", repository.failedInterviewID)
	}
}
```

- [ ] **Step 2: Run service tests and verify they fail**

Run:

```powershell
go test ./internal/service
```

Expected: FAIL with compile errors for `NewInterviewServiceWithRunner`, `QuestionLanguage`, `InterviewStatusGeneratingQuestions`, `ErrQuestionLanguageUnsupported`, and repository stub method mismatches.

- [ ] **Step 3: Write failing repository tests for generated question persistence and start-ready status**

Add to `backend/internal/repository/interview_repository_test.go`:

```go
func TestCreatePendingPersistsGeneratingInterviewWithLanguage(t *testing.T) {
	pool := testPool(t)
	repository := NewInterviewRepository(pool)
	ctx := context.Background()

	response, err := repository.CreatePending(ctx, model.CreateInterviewRequest{
		JobTitle:         "Backend Engineer",
		JobDescription:   "Build APIs",
		UserProfile:      "Go experience",
		QuestionCount:    2,
		QuestionLanguage: model.QuestionLanguageEnUS,
	})
	if err != nil {
		t.Fatalf("CreatePending returned error: %v", err)
	}
	defer pool.Exec(ctx, "DELETE FROM interviews WHERE id = $1", response.ID)

	if response.Status != model.InterviewStatusGeneratingQuestions {
		t.Fatalf("expected generating_questions, got %q", response.Status)
	}

	var status, language string
	if err := pool.QueryRow(ctx, `
		SELECT status, question_language
		FROM interviews
		WHERE id = $1
	`, response.ID).Scan(&status, &language); err != nil {
		t.Fatalf("query interview: %v", err)
	}
	if status != model.InterviewStatusGeneratingQuestions || language != model.QuestionLanguageEnUS {
		t.Fatalf("unexpected status/language: %q/%q", status, language)
	}
}

func TestSaveGeneratedQuestionsMarksInterviewReady(t *testing.T) {
	pool := testPool(t)
	repository := NewInterviewRepository(pool)
	ctx := context.Background()

	created, err := repository.CreatePending(ctx, model.CreateInterviewRequest{
		JobTitle:         "後端工程師",
		JobDescription:   "需要熟悉 Go",
		UserProfile:      "有 Go 經驗",
		QuestionCount:    2,
		QuestionLanguage: model.QuestionLanguageZhTW,
	})
	if err != nil {
		t.Fatalf("CreatePending returned error: %v", err)
	}
	defer pool.Exec(ctx, "DELETE FROM interviews WHERE id = $1", created.ID)

	err = repository.SaveGeneratedQuestions(ctx, created.ID, []llm.GeneratedQuestion{
		{Order: 1, Text: "問題一"},
		{Order: 2, Text: "問題二"},
	})
	if err != nil {
		t.Fatalf("SaveGeneratedQuestions returned error: %v", err)
	}

	var status string
	var questionCount int
	if err := pool.QueryRow(ctx, `
		SELECT status, (SELECT count(*) FROM questions WHERE interview_id = $1)
		FROM interviews
		WHERE id = $1
	`, created.ID).Scan(&status, &questionCount); err != nil {
		t.Fatalf("query generated status: %v", err)
	}
	if status != model.InterviewStatusQuestionsReady || questionCount != 2 {
		t.Fatalf("expected ready with 2 questions, got %q with %d", status, questionCount)
	}
}

func TestMarkQuestionGenerationFailedMarksInterviewFailed(t *testing.T) {
	pool := testPool(t)
	repository := NewInterviewRepository(pool)
	ctx := context.Background()

	created, err := repository.CreatePending(ctx, model.CreateInterviewRequest{
		JobTitle:         "後端工程師",
		JobDescription:   "需要熟悉 Go",
		UserProfile:      "有 Go 經驗",
		QuestionCount:    1,
		QuestionLanguage: model.QuestionLanguageZhTW,
	})
	if err != nil {
		t.Fatalf("CreatePending returned error: %v", err)
	}
	defer pool.Exec(ctx, "DELETE FROM interviews WHERE id = $1", created.ID)

	if err := repository.MarkQuestionGenerationFailed(ctx, created.ID); err != nil {
		t.Fatalf("MarkQuestionGenerationFailed returned error: %v", err)
	}

	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM interviews WHERE id = $1`, created.ID).Scan(&status); err != nil {
		t.Fatalf("query status: %v", err)
	}
	if status != model.InterviewStatusFailed {
		t.Fatalf("expected failed, got %q", status)
	}
}
```

- [ ] **Step 4: Run repository tests and verify they fail**

Run:

```powershell
go test ./internal/repository
```

Expected: FAIL with missing repository methods and missing DB column/check value.

- [ ] **Step 5: Implement backend model, migrations, repository, service, and LLM language support**

Change `backend/migrations/000001_create_interview_tables.up.sql`:

```sql
CREATE TABLE interviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_title TEXT NOT NULL,
    job_description TEXT NOT NULL,
    user_profile TEXT NOT NULL,
    question_count INTEGER NOT NULL CHECK (question_count BETWEEN 1 AND 10),
    question_language TEXT NOT NULL DEFAULT 'zh-TW' CHECK (
        question_language IN ('zh-TW', 'en-US')
    ),
    status TEXT NOT NULL DEFAULT 'created' CHECK (
        status IN ('created', 'generating_questions', 'questions_ready', 'in_progress', 'completed', 'failed')
    ),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

Change `backend/internal/model/interview.go` constants and structs:

```go
const (
	InterviewStatusCreated             = "created"
	InterviewStatusGeneratingQuestions = "generating_questions"
	InterviewStatusQuestionsReady      = "questions_ready"
	InterviewStatusInProgress          = "in_progress"
	InterviewStatusCompleted           = "completed"
	InterviewStatusFailed              = "failed"

	QuestionLanguageZhTW = "zh-TW"
	QuestionLanguageEnUS = "en-US"
)

type CreateInterviewRequest struct {
	JobTitle         string `json:"job_title"`
	JobDescription   string `json:"job_description"`
	UserProfile      string `json:"user_profile"`
	QuestionCount    int    `json:"question_count"`
	QuestionLanguage string `json:"question_language"`
}
```

Add `QuestionLanguage string` to `Interview`, `InterviewDetail`, and `InterviewDetailResponse` with JSON field `question_language`.

Change `backend/internal/llm/question_generator.go`:

```go
type GenerateQuestionsInput struct {
	JobTitle         string
	JobDescription   string
	UserProfile      string
	QuestionCount    int
	QuestionLanguage string
}
```

Update `MockQuestionGenerator.GenerateQuestions` to choose English base questions when `input.QuestionLanguage == "en-US"`:

```go
englishQuestions := []string{
	"Please introduce your previous experience related to backend development.",
	"How would you design a REST API?",
	"What do you pay attention to when working with PostgreSQL?",
	"Please share an experience debugging a backend service issue.",
	"How do you keep APIs maintainable and testable?",
	"How do you handle database transactions and error recovery?",
	"How would you design a scalable service architecture?",
	"Please share your experience working with cross-functional teams.",
	"How do you evaluate system performance bottlenecks?",
	"Describe how you learn and apply a new technology in a project.",
}
```

Change `buildQuestionPrompt` in `backend/internal/llm/gemini_question_generator.go` to include:

```go
languageInstruction := "繁體中文"
if input.QuestionLanguage == "en-US" {
	languageInstruction = "English"
}
return fmt.Sprintf(`你是協助使用者準備面試的面試官。
請根據使用者提供的職位名稱、職位要求及說明、個人資訊，產生 %d 題%s面試問題。
...
`, input.QuestionCount, languageInstruction, input.JobTitle, input.JobDescription, input.UserProfile)
```

Replace repository interface methods in `backend/internal/service/interview_service.go`:

```go
type InterviewRepository interface {
	CreatePending(ctx context.Context, input model.CreateInterviewRequest) (model.CreateInterviewResponse, error)
	SaveGeneratedQuestions(ctx context.Context, interviewID string, questions []llm.GeneratedQuestion) error
	MarkQuestionGenerationFailed(ctx context.Context, interviewID string) error
	GetByID(ctx context.Context, interviewID string) (model.InterviewDetail, error)
	Start(ctx context.Context, interviewID string) (model.CreateInterviewResponse, error)
}
```

Add:

```go
var (
	ErrQuestionLanguageUnsupported = errors.New("question_language must be zh-TW or en-US")
	ErrInterviewNotReady           = errors.New("interview is not ready to start")
)

type asyncRunner func(func())

func defaultAsyncRunner(task func()) {
	go task()
}

func NewInterviewService(generator llm.QuestionGenerator, repository InterviewRepository) *InterviewService {
	return NewInterviewServiceWithRunner(generator, repository, defaultAsyncRunner)
}

func NewInterviewServiceWithRunner(generator llm.QuestionGenerator, repository InterviewRepository, runner asyncRunner) *InterviewService {
	return &InterviewService{generator: generator, repository: repository, runner: runner}
}
```

Implement `CreateInterview` to trim inputs, default empty language to `zh-TW`, validate `zh-TW`/`en-US`, call `CreatePending`, then run:

```go
s.runner(func() {
	generationCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	questions, err := s.generator.GenerateQuestions(generationCtx, llm.GenerateQuestionsInput{
		JobTitle:         input.JobTitle,
		JobDescription:   input.JobDescription,
		UserProfile:      input.UserProfile,
		QuestionCount:    input.QuestionCount,
		QuestionLanguage: input.QuestionLanguage,
	})
	if err != nil {
		_ = s.repository.MarkQuestionGenerationFailed(context.Background(), created.ID)
		return
	}
	if err := s.repository.SaveGeneratedQuestions(context.Background(), created.ID, questions); err != nil {
		_ = s.repository.MarkQuestionGenerationFailed(context.Background(), created.ID)
	}
})
```

Implement repository methods in `backend/internal/repository/interview_repository.go`:

```go
func (r *InterviewRepository) CreatePending(ctx context.Context, input model.CreateInterviewRequest) (model.CreateInterviewResponse, error) {
	var interviewID string
	err := r.pool.QueryRow(ctx, `
		INSERT INTO interviews (job_title, job_description, user_profile, question_count, question_language, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, input.JobTitle, input.JobDescription, input.UserProfile, input.QuestionCount, input.QuestionLanguage, model.InterviewStatusGeneratingQuestions).Scan(&interviewID)
	if err != nil {
		return model.CreateInterviewResponse{}, err
	}
	return model.CreateInterviewResponse{ID: interviewID, Status: model.InterviewStatusGeneratingQuestions}, nil
}
```

`SaveGeneratedQuestions` must insert all questions in a transaction, delete any existing questions for that interview first, and update `interviews.status` to `questions_ready`. `MarkQuestionGenerationFailed` updates status to `failed`. Update `GetByID` SELECT to include `question_language`.

Update test stubs in `backend/internal/service/interview_service_test.go`:

```go
type stubQuestionGenerator struct {
	input llm.GenerateQuestionsInput
	err   error
}

func (s *stubQuestionGenerator) GenerateQuestions(ctx context.Context, input llm.GenerateQuestionsInput) ([]llm.GeneratedQuestion, error) {
	s.input = input
	if s.err != nil {
		return nil, s.err
	}
	questions := make([]llm.GeneratedQuestion, 0, input.QuestionCount)
	for index := 0; index < input.QuestionCount; index++ {
		questions = append(questions, llm.GeneratedQuestion{Order: index + 1, Text: "mock question"})
	}
	return questions, nil
}
```

- [ ] **Step 6: Run backend service, repository, and llm tests**

Run:

```powershell
go test ./internal/service ./internal/repository ./internal/llm
```

Expected: PASS. If repository integration tests require `DATABASE_URL`, run the unit packages first and run integration tests after local database setup.

- [ ] **Step 7: Commit Task 1**

```powershell
git add backend/migrations backend/internal/model backend/internal/llm backend/internal/service backend/internal/repository
git commit -m "feat: prepare interviews asynchronously"
```

---

## Task 2: Backend Start API and Question Visibility

**Files:**
- Modify: `backend/internal/service/interview_service.go`
- Modify: `backend/internal/repository/interview_repository.go`
- Modify: `backend/internal/handler/interview_handler.go`
- Modify: `backend/cmd/api/main_test.go`
- Test: `backend/internal/service/interview_service_test.go`
- Test: `backend/internal/handler/interview_handler_test.go`
- Test: `backend/internal/repository/interview_repository_test.go`

- [ ] **Step 1: Write failing service tests for question hiding and start**

Add to `backend/internal/service/interview_service_test.go`:

```go
func TestGetInterviewHidesQuestionsBeforeInterviewStarts(t *testing.T) {
	repository := &stubInterviewRepository{
		detail: model.InterviewDetail{
			ID:               "interview-id",
			JobTitle:         "後端工程師",
			JobDescription:   "需要熟悉 Go",
			UserProfile:      "有 Go 經驗",
			QuestionCount:    1,
			QuestionLanguage: model.QuestionLanguageZhTW,
			Status:           model.InterviewStatusQuestionsReady,
			Questions: []model.Question{
				{ID: "question-id", InterviewID: "interview-id", Order: 1, Text: "不能在準備頁顯示"},
			},
		},
	}
	service := NewInterviewServiceWithRunner(&stubQuestionGenerator{}, repository, func(task func()) { task() })

	response, err := service.GetInterview(context.Background(), "interview-id")

	if err != nil {
		t.Fatalf("GetInterview returned error: %v", err)
	}
	if len(response.Questions) != 0 {
		t.Fatalf("expected questions hidden before start, got %+v", response.Questions)
	}
}

func TestGetInterviewShowsQuestionsInProgressAndCompleted(t *testing.T) {
	for _, status := range []string{model.InterviewStatusInProgress, model.InterviewStatusCompleted} {
		repository := &stubInterviewRepository{
			detail: model.InterviewDetail{
				ID:               "interview-id",
				QuestionCount:    1,
				QuestionLanguage: model.QuestionLanguageZhTW,
				Status:           status,
				Questions: []model.Question{
					{ID: "question-id", InterviewID: "interview-id", Order: 1, Text: "可顯示題目"},
				},
			},
		}
		service := NewInterviewServiceWithRunner(&stubQuestionGenerator{}, repository, func(task func()) { task() })

		response, err := service.GetInterview(context.Background(), "interview-id")
		if err != nil {
			t.Fatalf("GetInterview(%s) returned error: %v", status, err)
		}
		if len(response.Questions) != 1 || response.Questions[0].Text != "可顯示題目" {
			t.Fatalf("expected questions visible for %s, got %+v", status, response.Questions)
		}
	}
}

func TestStartInterviewTransitionsReadyInterview(t *testing.T) {
	repository := &stubInterviewRepository{startResponse: model.CreateInterviewResponse{ID: "interview-id", Status: model.InterviewStatusInProgress}}
	service := NewInterviewServiceWithRunner(&stubQuestionGenerator{}, repository, func(task func()) { task() })

	response, err := service.StartInterview(context.Background(), "interview-id")

	if err != nil {
		t.Fatalf("StartInterview returned error: %v", err)
	}
	if repository.startedID != "interview-id" {
		t.Fatalf("expected start id, got %q", repository.startedID)
	}
	if response.Status != model.InterviewStatusInProgress {
		t.Fatalf("expected in_progress, got %q", response.Status)
	}
}
```

- [ ] **Step 2: Run service tests and verify failure**

Run:

```powershell
go test ./internal/service
```

Expected: FAIL for missing `StartInterview`, question visibility behavior, and stub methods.

- [ ] **Step 3: Write failing handler test for start endpoint**

Add to `backend/internal/handler/interview_handler_test.go`:

```go
func TestStartInterviewReturnsInProgress(t *testing.T) {
	handler := NewInterviewHandler(&stubInterviewService{
		startResponse: model.CreateInterviewResponse{ID: "interview-id", Status: model.InterviewStatusInProgress},
	}, nil)
	request := httptest.NewRequest(http.MethodPost, "/interview-id/start", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	var body model.CreateInterviewResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected JSON response, got error: %v", err)
	}
	if body.Status != model.InterviewStatusInProgress {
		t.Fatalf("expected in_progress, got %q", body.Status)
	}
}

func TestStartInterviewReturnsConflictWhenNotReady(t *testing.T) {
	handler := NewInterviewHandler(&stubInterviewService{startErr: service.ErrInterviewNotReady}, nil)
	request := httptest.NewRequest(http.MethodPost, "/interview-id/start", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	assertErrorResponse(t, response, http.StatusConflict, "interview is not ready to start")
}
```

Extend `stubInterviewService`:

```go
startResponse model.CreateInterviewResponse
startErr      error

func (s *stubInterviewService) StartInterview(ctx context.Context, interviewID string) (model.CreateInterviewResponse, error) {
	return s.startResponse, s.startErr
}
```

- [ ] **Step 4: Run handler tests and verify failure**

Run:

```powershell
go test ./internal/handler
```

Expected: FAIL until handler interface and route are implemented.

- [ ] **Step 5: Implement start and visibility**

In `backend/internal/service/interview_service.go`, add:

```go
func (s *InterviewService) StartInterview(ctx context.Context, interviewID string) (model.CreateInterviewResponse, error) {
	response, err := s.repository.Start(ctx, interviewID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, ErrInterviewNotFound) {
			return model.CreateInterviewResponse{}, ErrInterviewNotFound
		}
		if errors.Is(err, ErrInterviewNotReady) {
			return model.CreateInterviewResponse{}, ErrInterviewNotReady
		}
		return model.CreateInterviewResponse{}, err
	}
	return response, nil
}
```

In `mapInterviewDetailResponse`, only map questions when:

```go
showQuestions := detail.Status == model.InterviewStatusInProgress || detail.Status == model.InterviewStatusCompleted
if showQuestions {
	for _, question := range detail.Questions {
		questions = append(questions, model.QuestionResponse{ID: question.ID, Order: question.Order, Text: question.Text})
	}
}
```

In `backend/internal/repository/interview_repository.go`, implement:

```go
func (r *InterviewRepository) Start(ctx context.Context, interviewID string) (model.CreateInterviewResponse, error) {
	commandTag, err := r.pool.Exec(ctx, `
		UPDATE interviews
		SET status = $2, updated_at = now()
		WHERE id = $1 AND status = $3
	`, interviewID, model.InterviewStatusInProgress, model.InterviewStatusQuestionsReady)
	if err != nil {
		return model.CreateInterviewResponse{}, err
	}
	if commandTag.RowsAffected() == 1 {
		return model.CreateInterviewResponse{ID: interviewID, Status: model.InterviewStatusInProgress}, nil
	}

	var exists bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM interviews WHERE id = $1)`, interviewID).Scan(&exists); err != nil {
		return model.CreateInterviewResponse{}, err
	}
	if !exists {
		return model.CreateInterviewResponse{}, service.ErrInterviewNotFound
	}
	return model.CreateInterviewResponse{}, service.ErrInterviewNotReady
}
```

Avoid import cycles: if repository cannot import `service`, create repository-local sentinel errors or move shared errors to model. Prefer adding repository-local errors in `repository` and translating them in service only if an import cycle appears.

In `backend/internal/handler/interview_handler.go`, extend interface:

```go
StartInterview(ctx context.Context, interviewID string) (model.CreateInterviewResponse, error)
```

Add route before answer route:

```go
router.Post("/{interviewID}/start", startInterview(interviewService))
```

Handler:

```go
func startInterview(interviewService InterviewService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		interviewID := chi.URLParam(r, "interviewID")
		response, err := interviewService.StartInterview(r.Context(), interviewID)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrInterviewNotFound):
				writeError(w, http.StatusNotFound, err.Error())
			case errors.Is(err, service.ErrInterviewNotReady):
				writeError(w, http.StatusConflict, err.Error())
			default:
				slog.Error("start interview", "error", err)
				writeError(w, http.StatusInternalServerError, "failed to start interview")
			}
			return
		}
		writeJSON(w, http.StatusOK, response)
	}
}
```

- [ ] **Step 6: Run backend tests**

Run:

```powershell
go test ./...
```

Expected: PASS.

- [ ] **Step 7: Commit Task 2**

```powershell
git add backend
git commit -m "feat: add interview start flow"
```

---

## Task 3: Frontend API Types, Two-Stage Setup, and Microphone Gate

**Files:**
- Modify: `frontend/src/types/interview.ts`
- Modify: `frontend/src/api/interviews.ts`
- Modify: `frontend/src/pages/NewInterviewPage.tsx`
- Test: `frontend/src/App.test.tsx`

- [ ] **Step 1: Write failing frontend tests for wizard and mic gate**

Replace the create-form tests in `frontend/src/App.test.tsx` with tests that assert:

```tsx
it('renders the first setup step before interview settings', () => {
  mockPathname('/interviews/new')
  render(<App />)

  expect(screen.getByRole('heading', { name: '建立模擬面試' })).toBeInTheDocument()
  expect(screen.getByLabelText('職位名稱')).toBeInTheDocument()
  expect(screen.getByLabelText('職位要求及說明')).toBeInTheDocument()
  expect(screen.getByLabelText('個人資訊')).toBeInTheDocument()
  expect(screen.queryByLabelText('題目數量')).not.toBeInTheDocument()
  expect(screen.queryByRole('button', { name: '建立面試' })).not.toBeInTheDocument()
})

it('requires a successful microphone test before creating an interview', async () => {
  mockPathname('/interviews/new')
  const media = installMediaRecorderMock()
  const fetchMock = mockFetchOnce({ id: 'interview-123', status: 'generating_questions' }, { status: 201 })
  vi.stubGlobal('fetch', fetchMock)
  const pushState = vi.spyOn(window.history, 'pushState')

  render(<App />)

  fireEvent.change(screen.getByLabelText('職位名稱'), { target: { value: 'Backend Engineer' } })
  fireEvent.change(screen.getByLabelText('職位要求及說明'), { target: { value: 'Build APIs' } })
  fireEvent.change(screen.getByLabelText('個人資訊'), { target: { value: 'Go experience' } })
  fireEvent.click(screen.getByRole('button', { name: '下一步' }))

  expect(screen.getByLabelText('題目數量')).toHaveValue(3)
  expect(screen.getByLabelText('題目語言')).toHaveValue('zh-TW')
  expect(screen.getByRole('button', { name: '建立面試' })).toBeDisabled()

  fireEvent.click(screen.getByRole('button', { name: '測試麥克風' }))
  await waitFor(() => expect(media.getUserMedia).toHaveBeenCalledWith({ audio: true }))
  expect(screen.getByText('麥克風測試完成')).toBeInTheDocument()

  fireEvent.change(screen.getByLabelText('題目語言'), { target: { value: 'en-US' } })
  fireEvent.click(screen.getByRole('button', { name: '建立面試' }))

  await waitFor(() => {
    expect(fetchMock).toHaveBeenCalledWith('/api/interviews', expect.objectContaining({
      method: 'POST',
      body: JSON.stringify({
        job_title: 'Backend Engineer',
        job_description: 'Build APIs',
        user_profile: 'Go experience',
        question_count: 3,
        question_language: 'en-US',
      }),
    }))
  })
  expect(pushState).toHaveBeenCalledWith({}, '', '/interviews/interview-123')
})
```

- [ ] **Step 2: Run frontend tests and verify failure**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd test -- App.test.tsx
```

Expected: FAIL because the current page is a single form and request body lacks `question_language`.

- [ ] **Step 3: Implement frontend types and API**

In `frontend/src/types/interview.ts`:

```ts
export type QuestionLanguage = 'zh-TW' | 'en-US'

export type CreateInterviewRequest = {
  job_title: string
  job_description: string
  user_profile: string
  question_count: number
  question_language: QuestionLanguage
}
```

Add `question_language: QuestionLanguage` to `InterviewDetail`.

In `frontend/src/api/interviews.ts`, add:

```ts
export async function startInterview(interviewID: string): Promise<CreateInterviewResponse> {
  const response = await fetch(`${API_BASE_URL}/api/interviews/${interviewID}/start`, {
    method: 'POST',
  })
  return parseJSONResponse<CreateInterviewResponse>(response, '開始面試失敗')
}
```

- [ ] **Step 4: Implement two-stage wizard**

In `frontend/src/pages/NewInterviewPage.tsx`, keep one component but split state:

```ts
type SetupStep = 'profile' | 'settings'
const initialForm: CreateInterviewRequest = {
  job_title: '',
  job_description: '',
  user_profile: '',
  question_count: 3,
  question_language: 'zh-TW',
}
```

Add microphone test state:

```ts
const [step, setStep] = useState<SetupStep>('profile')
const [isTestingMicrophone, setIsTestingMicrophone] = useState(false)
const [microphoneReady, setMicrophoneReady] = useState(false)
const [microphoneError, setMicrophoneError] = useState<string | null>(null)
```

Implement:

```ts
async function testMicrophone() {
  setIsTestingMicrophone(true)
  setMicrophoneError(null)
  try {
    const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
    stream.getTracks().forEach((track) => track.stop())
    setMicrophoneReady(true)
  } catch (error) {
    setMicrophoneReady(false)
    setMicrophoneError(getRecordingErrorMessage(error))
  } finally {
    setIsTestingMicrophone(false)
  }
}
```

Use existing `getRecordingErrorMessage` logic by moving it to `frontend/src/lib/mediaErrors.ts` and importing it in both `NewInterviewPage.tsx` and `InterviewSessionPage.tsx`, or duplicate the function in this task and extract in a refactor step.

Create button must be disabled unless `microphoneReady && !isSubmitting`.

- [ ] **Step 5: Run frontend tests**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd test -- App.test.tsx
```

Expected: PASS for new wizard tests; update any existing create tests that still expect the single-step form.

- [ ] **Step 6: Commit Task 3**

```powershell
git add frontend/src/types/interview.ts frontend/src/api/interviews.ts frontend/src/pages/NewInterviewPage.tsx frontend/src/App.test.tsx
git commit -m "feat: add two-stage interview setup"
```

---

## Task 4: Preparation Page Polling and Hidden Questions

**Files:**
- Modify: `frontend/src/pages/InterviewDetailPage.tsx`
- Test: `frontend/src/App.test.tsx`

- [ ] **Step 1: Write failing tests for preparation states**

Replace the current detail-page test that displays questions with:

```tsx
it('shows question preparation while questions are generating', async () => {
  vi.useFakeTimers()
  mockPathname('/interviews/interview-123')
  const fetchMock = vi.fn().mockResolvedValue(
    new Response(JSON.stringify({
      id: 'interview-123',
      job_title: '後端工程師',
      job_description: '需要熟悉 Go',
      user_profile: '有 Go 經驗',
      question_count: 2,
      question_language: 'zh-TW',
      status: 'generating_questions',
      questions: [],
      answers: [],
    }), { headers: { 'Content-Type': 'application/json' } }),
  )
  vi.stubGlobal('fetch', fetchMock)

  render(<App />)

  expect(await screen.findByText('題目準備中')).toBeInTheDocument()
  expect(screen.queryByText('問題 1')).not.toBeInTheDocument()
  expect(screen.queryByRole('button', { name: '開始模擬面試' })).not.toBeInTheDocument()

  act(() => vi.advanceTimersByTime(2000))
  await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2))
  vi.useRealTimers()
})

it('starts a ready interview from the preparation page', async () => {
  mockPathname('/interviews/interview-123')
  const fetchMock = vi
    .fn()
    .mockResolvedValueOnce(new Response(JSON.stringify({
      id: 'interview-123',
      job_title: '後端工程師',
      job_description: '需要熟悉 Go',
      user_profile: '有 Go 經驗',
      question_count: 2,
      question_language: 'zh-TW',
      status: 'questions_ready',
      questions: [],
      answers: [],
    }), { headers: { 'Content-Type': 'application/json' } }))
    .mockResolvedValueOnce(new Response(JSON.stringify({ id: 'interview-123', status: 'in_progress' }), {
      headers: { 'Content-Type': 'application/json' },
    }))
  vi.stubGlobal('fetch', fetchMock)
  const pushState = vi.spyOn(window.history, 'pushState')

  render(<App />)

  expect(await screen.findByText('題目已準備完成')).toBeInTheDocument()
  expect(screen.queryByText('請介紹你過去與後端開發相關的經驗。')).not.toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: '開始模擬面試' }))

  await waitFor(() => expect(fetchMock).toHaveBeenLastCalledWith('/api/interviews/interview-123/start', { method: 'POST' }))
  expect(pushState).toHaveBeenCalledWith({}, '', '/interviews/interview-123/session')
})
```

- [ ] **Step 2: Run tests and verify failure**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd test -- App.test.tsx
```

Expected: FAIL because current detail page shows generated questions and uses a direct link.

- [ ] **Step 3: Implement preparation page**

In `frontend/src/pages/InterviewDetailPage.tsx`:

- Keep `getInterview` loading.
- Add polling when status is `generating_questions`:

```ts
useEffect(() => {
  if (interview?.status !== 'generating_questions') {
    return
  }
  const intervalID = window.setInterval(() => {
    loadInterview()
  }, 2000)
  return () => window.clearInterval(intervalID)
}, [interview?.status, interviewID])
```

Make `loadInterview` stable with `useCallback`.

- Replace question list with status copy:
  - `generating_questions`: show `題目準備中`
  - `questions_ready`: show `題目已準備完成`
  - `failed`: show `題目產生失敗，請建立另一場面試。`
  - `in_progress`: show button/link to continue session
  - `completed`: show link to result
- Start button calls `startInterview(interview.id)` then navigates to `/interviews/${id}/session`.
- Never render `interview.questions.map` on this page.

- [ ] **Step 4: Run frontend tests**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd test -- App.test.tsx
```

Expected: PASS.

- [ ] **Step 5: Commit Task 4**

```powershell
git add frontend/src/pages/InterviewDetailPage.tsx frontend/src/App.test.tsx
git commit -m "feat: add interview preparation page"
```

---

## Task 5: Immersive Session State Machine and Upload Queue

**Files:**
- Modify: `frontend/src/pages/InterviewSessionPage.tsx`
- Test: `frontend/src/App.test.tsx`
- Modify: `.env.example`

- [ ] **Step 1: Write failing tests for automatic TTS to recording**

Replace manual session playback/recording tests with:

```tsx
it('automatically plays the first question and starts recording after playback ends', async () => {
  const speech = installSpeechSynthesisMock()
  const media = installMediaRecorderMock()
  installObjectURLMock()
  mockPathname('/interviews/interview-123/session')
  vi.stubGlobal('fetch', mockFetchOnce({
    id: 'interview-123',
    job_title: '後端工程師',
    job_description: '需要熟悉 Go',
    user_profile: '有 Go 經驗',
    question_count: 1,
    question_language: 'zh-TW',
    status: 'in_progress',
    questions: [{ id: 'question-1', order: 1, text: '請介紹你過去與後端開發相關的經驗。' }],
    answers: [],
  }))

  render(<App />)

  expect(await screen.findByText('正在播放題目')).toBeInTheDocument()
  expect(screen.queryByText('請介紹你過去與後端開發相關的經驗。')).not.toBeInTheDocument()
  expect(speech.speak).toHaveBeenCalledTimes(1)
  expect(speech.utterances[0].lang).toBe('zh-TW')

  act(() => speech.utterances[0].onend?.())

  await waitFor(() => expect(media.getUserMedia).toHaveBeenCalledWith({ audio: true }))
  expect(screen.getByText('正在錄音')).toBeInTheDocument()
  expect(screen.getByRole('button', { name: '回答結束' })).toBeEnabled()
})
```

- [ ] **Step 2: Write failing test for replay discarding current recording**

```tsx
it('replays the question during recording and discards the current recording', async () => {
  const speech = installSpeechSynthesisMock()
  const media = installMediaRecorderMock()
  installObjectURLMock()
  mockPathname('/interviews/interview-123/session')
  vi.stubGlobal('fetch', mockFetchOnce({
    id: 'interview-123',
    job_title: '後端工程師',
    job_description: '需要熟悉 Go',
    user_profile: '有 Go 經驗',
    question_count: 1,
    question_language: 'zh-TW',
    status: 'in_progress',
    questions: [{ id: 'question-1', order: 1, text: '問題一' }],
    answers: [],
  }))

  render(<App />)
  expect(await screen.findByText('正在播放題目')).toBeInTheDocument()
  act(() => speech.utterances[0].onend?.())
  await waitFor(() => expect(media.recorders).toHaveLength(1))

  fireEvent.click(screen.getByRole('button', { name: '重新播放題目' }))

  expect(media.recorders[0].stop).toHaveBeenCalledTimes(1)
  expect(screen.queryByLabelText('回答錄音預覽')).not.toBeInTheDocument()
  expect(speech.speak).toHaveBeenCalledTimes(2)
})
```

- [ ] **Step 3: Write failing test for answer end, background upload, and final wait**

```tsx
it('queues uploads in the background and waits before opening the result page', async () => {
  const speech = installSpeechSynthesisMock()
  const media = installMediaRecorderMock()
  installObjectURLMock()
  mockPathname('/interviews/interview-123/session')
  const fetchMock = vi
    .fn()
    .mockResolvedValueOnce(new Response(JSON.stringify({
      id: 'interview-123',
      job_title: '後端工程師',
      job_description: '需要熟悉 Go',
      user_profile: '有 Go 經驗',
      question_count: 1,
      question_language: 'zh-TW',
      status: 'in_progress',
      questions: [{ id: 'question-1', order: 1, text: '問題一' }],
      answers: [],
    }), { headers: { 'Content-Type': 'application/json' } }))
    .mockResolvedValueOnce(new Response(JSON.stringify({
      id: 'answer-1',
      interview_id: 'interview-123',
      question_id: 'question-1',
      audio_path: 'storage/audio/interview-123/question-1.webm',
      transcript_text: null,
    }), { status: 201, headers: { 'Content-Type': 'application/json' } }))
  vi.stubGlobal('fetch', fetchMock)

  render(<App />)
  expect(await screen.findByText('正在播放題目')).toBeInTheDocument()
  act(() => speech.utterances[0].onend?.())
  await waitFor(() => expect(media.recorders).toHaveLength(1))

  fireEvent.click(screen.getByRole('button', { name: '回答結束' }))

  expect(await screen.findByText('正在完成面試')).toBeInTheDocument()
  await waitFor(() => expect(fetchMock).toHaveBeenCalledWith(
    '/api/interviews/interview-123/questions/question-1/answer',
    expect.objectContaining({ method: 'POST', body: expect.any(FormData) }),
  ))
  await waitFor(() => expect(window.location.pathname).toBe('/interviews/interview-123/result'))
})
```

- [ ] **Step 4: Run frontend tests and verify failure**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd test -- App.test.tsx
```

Expected: FAIL because current session is manual and shows question text.

- [ ] **Step 5: Implement recording limit env**

Add to `.env.example`:

```dotenv
VITE_MAX_ANSWER_RECORDING_SECONDS=180
```

In `InterviewSessionPage.tsx`:

```ts
const maxRecordingSeconds = Number(import.meta.env.VITE_MAX_ANSWER_RECORDING_SECONDS ?? 180)
const safeMaxRecordingSeconds = Number.isFinite(maxRecordingSeconds) && maxRecordingSeconds > 0 ? maxRecordingSeconds : 180
```

- [ ] **Step 6: Implement session state machine**

Use this state shape:

```ts
type SessionPhase =
  | 'loading'
  | 'playing_question'
  | 'recording_answer'
  | 'advancing'
  | 'finishing_uploads'
  | 'blocked'

type UploadQueueItem = {
  questionID: string
  audio: Blob
  attempts: number
  status: 'queued' | 'uploading' | 'uploaded' | 'failed'
  error: string | null
}
```

Behavior:

- On load, require `interview.status === 'in_progress'`; otherwise show an error telling user to start from prep page.
- Do not render `currentQuestion.text` anywhere in session UI.
- `playCurrentQuestion` sets phase `playing_question`, creates `SpeechSynthesisUtterance(currentQuestion.text)`, sets `utterance.lang = interview.question_language`, and on end calls `startAnswerRecording`.
- `startAnswerRecording` requests microphone, creates `MediaRecorder`, starts countdown from `safeMaxRecordingSeconds`, and sets phase `recording_answer`.
- `stopAnswerRecording({ discard: false })` creates a Blob, queues upload, and advances to next question immediately when one exists.
- `stopAnswerRecording({ discard: true })` stops recorder, ignores chunks, and replays same question.
- `回答結束` calls `stopAnswerRecording({ discard: false })`.
- `重新播放題目` calls `stopAnswerRecording({ discard: true })`.
- When countdown reaches zero, call `stopAnswerRecording({ discard: false })`.
- After final question is queued, set phase `finishing_uploads`.

- [ ] **Step 7: Implement in-memory upload queue**

Use a `useEffect` that picks the first `queued` item and uploads it:

```ts
useEffect(() => {
  const nextItem = uploadQueue.find((item) => item.status === 'queued')
  if (!nextItem) {
    return
  }
  let cancelled = false

  async function upload() {
    setUploadQueue((items) => items.map((item) =>
      item.questionID === nextItem.questionID ? { ...item, status: 'uploading', attempts: item.attempts + 1, error: null } : item,
    ))
    try {
      await uploadAnswerAudio(interviewID, nextItem.questionID, nextItem.audio)
      if (!cancelled) {
        setUploadQueue((items) => items.map((item) =>
          item.questionID === nextItem.questionID ? { ...item, status: 'uploaded', error: null } : item,
        ))
      }
    } catch (error) {
      if (!cancelled) {
        setUploadQueue((items) => items.map((item) =>
          item.questionID === nextItem.questionID
            ? { ...item, status: item.attempts + 1 >= 3 ? 'failed' : 'queued', error: error instanceof Error ? error.message : '上傳回答失敗' }
            : item,
        ))
      }
    }
  }

  upload()
  return () => {
    cancelled = true
  }
}, [uploadQueue, interviewID])
```

When phase is `finishing_uploads` and every queue item is `uploaded`, navigate to `/interviews/${interviewID}/result`. If any item is `failed`, show `重試上傳` and `重新回答本題` controls. `重試上傳` changes failed items to `queued` with existing blobs. `重新回答本題` sets `currentQuestionIndex` to that question and phase `playing_question`.

- [ ] **Step 8: Run frontend tests**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd test -- App.test.tsx
```

Expected: PASS.

- [ ] **Step 9: Commit Task 5**

```powershell
git add frontend/src/pages/InterviewSessionPage.tsx frontend/src/App.test.tsx .env.example
git commit -m "feat: add immersive interview session"
```

---

## Task 6: Documentation and Final Verification

**Files:**
- Modify: `docs/API.md`
- Modify: `docs/DEVELOPMENT_PLAN.md`
- Modify: `README.md`

- [ ] **Step 1: Update API documentation**

In `docs/API.md`, update:

- `POST /api/interviews` request to include:

```json
{
  "job_title": "後端工程師",
  "job_description": "需要熟悉 Go、PostgreSQL、REST API",
  "user_profile": "有 Java 和 Go 學習經驗，正在準備後端工程師面試",
  "question_count": 3,
  "question_language": "zh-TW"
}
```

- Success response status:

```json
{
  "id": "interview_uuid",
  "status": "generating_questions"
}
```

- Document `question_language must be zh-TW or en-US`.
- Document `POST /api/interviews/{interview_id}/start`.
- Document that `GET /api/interviews/{id}` returns an empty `questions` array until status is `in_progress` or `completed`.

- [ ] **Step 2: Update development plan**

Append a new section to `docs/DEVELOPMENT_PLAN.md`:

```md
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
- `npm test --prefix frontend` passed.
- `npm run build --prefix frontend` passed.
```

- [ ] **Step 3: Update README**

Add `VITE_MAX_ANSWER_RECORDING_SECONDS=180` to frontend environment setup if README documents env usage. Update manual verification steps:

```md
Manual immersive flow verification:

1. Open `http://localhost:5173/interviews/new`.
2. Fill job information, click `下一步`.
3. Select question count and language, click `測試麥克風`, allow microphone access.
4. Click `建立面試`; confirm the preparation page shows `題目準備中` and does not show question text.
5. Wait until `題目已準備完成`, click `開始模擬面試`.
6. Confirm the session plays audio automatically, hides question text, starts recording after playback, and supports `回答結束` and `重新播放題目`.
7. Finish all questions and confirm the result page shows question text and playable answer audio.
```

- [ ] **Step 4: Run full backend tests**

Run:

```powershell
go test ./...
```

Expected: PASS.

- [ ] **Step 5: Run full frontend tests**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd test
```

Expected: PASS.

- [ ] **Step 6: Run frontend build**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd run build
```

Expected: PASS with Vite build output and no TypeScript errors.

- [ ] **Step 7: Commit docs**

```powershell
git add docs/API.md docs/DEVELOPMENT_PLAN.md README.md
git commit -m "docs: update immersive interview flow"
```

---

## Manual Verification

After implementation, run:

```powershell
docker compose up --build
```

Then verify:

1. `GET http://localhost:8080/health` returns `{"status":"ok"}`.
2. `http://localhost:5173/interviews/new` shows a two-stage setup form.
3. Create a Chinese interview and confirm the prep page shows `題目準備中`, then `題目已準備完成`.
4. Confirm prep page does not show question text.
5. Start the interview and confirm session hides question text, plays the question aloud, records automatically, and supports `回答結束`.
6. Press `重新播放題目` during recording and confirm the current recording is discarded and restarted.
7. Complete the final question and confirm the finishing screen waits for uploads before navigating to result.
8. Confirm result page shows question text and answer audio controls.

API curl checks:

```bash
curl -X POST http://localhost:8080/api/interviews \
  -H "Content-Type: application/json" \
  -d '{"job_title":"Backend Engineer","job_description":"Build Go APIs","user_profile":"Go experience","question_count":2,"question_language":"en-US"}'
```

```bash
curl http://localhost:8080/api/interviews/{interview_id}
```

Expected before start: status eventually `questions_ready`, `questions: []`.

```bash
curl -X POST http://localhost:8080/api/interviews/{interview_id}/start
```

Expected: `{"id":"...","status":"in_progress"}`.

```bash
curl http://localhost:8080/api/interviews/{interview_id}
```

Expected after start: status `in_progress`, questions include text.

---

## Self-Review

- Spec coverage: The plan covers two-stage setup, mic gate, language, async generation, prep polling, hidden questions, start API, immersive session, replay behavior, in-memory upload queue, final wait, docs, and verification.
- Placeholder scan: No placeholder markers remain; every task has concrete files, commands, and expected outcomes.
- Type consistency: Uses `question_language`, `QuestionLanguage`, `generating_questions`, `in_progress`, `StartInterview`, `startInterview`, and `VITE_MAX_ANSWER_RECORDING_SECONDS` consistently across backend and frontend.
