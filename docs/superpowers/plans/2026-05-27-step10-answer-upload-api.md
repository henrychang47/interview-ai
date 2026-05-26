# Step 10 Answer Upload API Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement Step 10 from `docs/mvp-spec.md` so the backend accepts a recorded answer audio file, stores it under local `backend/storage/audio`, and creates or updates the matching `answers` database row.

**Architecture:** Add a backend-only answer upload slice under the existing interview route tree: `POST /api/interviews/{interview_id}/questions/{question_id}/answer`. Keep validation and orchestration in a new `AnswerService`, database writes in a new `AnswerRepository`, and file persistence in a new `storage.LocalAudioStorage`. Do not wire the frontend upload button, mark interviews completed, expose a static audio route, implement transcription, or create the result page in this step.

**Tech Stack:** Go, chi, pgx, PostgreSQL, multipart/form-data, local filesystem storage, `httptest`, Go unit tests, Docker Compose for manual verification.

---

## Scope

Implement only Step 10 from `docs/mvp-spec.md`:

```text
- 實作 multipart upload API.
- 儲存音檔到 local storage.
- 建立 answers 資料.
```

Acceptance criteria:

```text
使用者錄音後可上傳
後端 storage/audio 中可看到音檔
DB answers 表有對應資料
```

Do not implement:

- Frontend upload UI or session page upload state.
- Automatic interview `completed` status after all answers exist. That is Step 11.
- Result page playback. That is Step 12.
- Static `GET /audio/...` serving. Keep the stored `audio_path` in API responses for now.
- STT/transcript generation.
- Cloud object storage.
- Auth, ownership, or user accounts.
- New environment variables. Use a local default storage root in `main.go`.

## File Structure

- Modify `backend/internal/model/interview.go`: add answer upload response DTO.
- Create `backend/internal/storage/audio_storage.go`: define `AudioStorage` interface and `LocalAudioStorage`.
- Create `backend/internal/storage/audio_storage_test.go`: verify local file save path and bytes.
- Create `backend/internal/repository/answer_repository.go`: validate interview/question relation and upsert answer row.
- Modify `backend/internal/repository/interview_repository.go`: no functional change expected; read only for consistency.
- Modify `backend/internal/repository/interview_repository_test.go`: add integration coverage for answer upsert only if no new dedicated repository test file is used.
- Create `backend/internal/repository/answer_repository_test.go`: integration tests for relation validation and upsert.
- Create `backend/internal/service/answer_service.go`: validate upload input, call storage, call repository, return response.
- Create `backend/internal/service/answer_service_test.go`: service unit tests with fake repository and fake storage.
- Create `backend/internal/handler/answer_handler.go`: multipart HTTP endpoint and error mapping.
- Create `backend/internal/handler/answer_handler_test.go`: HTTP tests for success and error cases.
- Modify `backend/internal/handler/interview_handler.go`: mount nested answer route under existing `/api/interviews`.
- Modify `backend/internal/handler/interview_handler_test.go`: update stub interface if needed.
- Modify `backend/cmd/api/main.go`: instantiate `LocalAudioStorage`, `AnswerRepository`, `AnswerService`, and pass answer service into `NewInterviewHandler`.
- Modify `docs/API.md`: document `POST /api/interviews/{interview_id}/questions/{question_id}/answer` with curl verification.
- Modify `docs/development-progress.md`: mark Step 10 completed after verification.
- Modify `docs/DEVELOPMENT_PLAN.md`: mark Step 10 completed after verification.
- No `.env.example` change is needed because no new environment variable is introduced.
- No frontend files should change for Step 10.

## API Contract

Endpoint:

```http
POST /api/interviews/{interview_id}/questions/{question_id}/answer
Content-Type: multipart/form-data
```

Multipart fields:

```text
audio: audio file, required
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

Validation/error responses:

```http
400 Bad Request
```

```json
{"error":"audio file is required"}
```

```http
400 Bad Request
```

```json
{"error":"audio file must be audio/webm"}
```

```http
404 Not Found
```

```json
{"error":"interview not found"}
```

```http
404 Not Found
```

```json
{"error":"question not found for interview"}
```

Server/storage errors:

```http
500 Internal Server Error
```

```json
{"error":"failed to save answer audio"}
```

```http
500 Internal Server Error
```

```json
{"error":"failed to save answer"}
```

## Task 1: Add Answer Upload DTO

**Files:**
- Modify: `backend/internal/model/interview.go`

- [ ] **Step 1: Add response type**

In `backend/internal/model/interview.go`, after `AnswerResponse`, add:

```go
type UploadAnswerResponse struct {
	ID             string  `json:"id"`
	InterviewID    string  `json:"interview_id"`
	QuestionID     string  `json:"question_id"`
	AudioPath      string  `json:"audio_path"`
	TranscriptText *string `json:"transcript_text"`
}
```

- [ ] **Step 2: Run model package tests**

Run:

```powershell
Push-Location backend
Remove-Item Env:\DATABASE_URL -ErrorAction SilentlyContinue
go test ./internal/model
Pop-Location
```

Expected: package reports `[no test files]` and exits 0.

## Task 2: Add Local Audio Storage

**Files:**
- Create: `backend/internal/storage/audio_storage.go`
- Create: `backend/internal/storage/audio_storage_test.go`

- [ ] **Step 1: Write failing storage test**

Create `backend/internal/storage/audio_storage_test.go`:

```go
package storage

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalAudioStorageSavesAnswerAudio(t *testing.T) {
	root := t.TempDir()
	storage := NewLocalAudioStorage(root)

	audioPath, err := storage.SaveAnswerAudio(
		context.Background(),
		"interview-id",
		"question-id",
		strings.NewReader("webm-bytes"),
	)

	if err != nil {
		t.Fatalf("SaveAnswerAudio returned error: %v", err)
	}
	expectedPath := filepath.Join("storage", "audio", "interview-id", "question-id.webm")
	if audioPath != expectedPath {
		t.Fatalf("expected audio path %q, got %q", expectedPath, audioPath)
	}

	savedBytes, err := os.ReadFile(filepath.Join(root, "interview-id", "question-id.webm"))
	if err != nil {
		t.Fatalf("read saved audio: %v", err)
	}
	if string(savedBytes) != "webm-bytes" {
		t.Fatalf("expected saved bytes, got %q", string(savedBytes))
	}
}
```

- [ ] **Step 2: Run storage test to verify it fails**

Run:

```powershell
Push-Location backend
go test ./internal/storage
Pop-Location
```

Expected: fail because package `storage` or `NewLocalAudioStorage` is not defined.

- [ ] **Step 3: Implement local storage**

Create `backend/internal/storage/audio_storage.go`:

```go
package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"
)

type AudioStorage interface {
	SaveAnswerAudio(ctx context.Context, interviewID string, questionID string, file io.Reader) (string, error)
}

type LocalAudioStorage struct {
	root string
}

func NewLocalAudioStorage(root string) *LocalAudioStorage {
	return &LocalAudioStorage{root: root}
}

func (s *LocalAudioStorage) SaveAnswerAudio(ctx context.Context, interviewID string, questionID string, file io.Reader) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	interviewDir := filepath.Join(s.root, interviewID)
	if err := os.MkdirAll(interviewDir, 0o755); err != nil {
		return "", err
	}

	filePath := filepath.Join(interviewDir, questionID+".webm")
	destination, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer destination.Close()

	if _, err := io.Copy(destination, file); err != nil {
		return "", err
	}

	return filepath.Join("storage", "audio", interviewID, questionID+".webm"), nil
}
```

- [ ] **Step 4: Run storage test**

Run:

```powershell
Push-Location backend
go test ./internal/storage
Pop-Location
```

Expected: `ok interview-ai/backend/internal/storage`.

## Task 3: Add Answer Repository

**Files:**
- Create: `backend/internal/repository/answer_repository.go`
- Create: `backend/internal/repository/answer_repository_test.go`

- [ ] **Step 1: Write failing repository integration tests**

Create `backend/internal/repository/answer_repository_test.go`:

```go
package repository

import (
	"context"
	"errors"
	"os"
	"testing"

	"interview-ai/backend/internal/llm"
	"interview-ai/backend/internal/model"
	"interview-ai/backend/internal/service"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestUpsertAnswerCreatesAndUpdatesAnswer(t *testing.T) {
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

	interviewRepository := NewInterviewRepository(pool)
	created, err := interviewRepository.CreateWithQuestions(ctx, model.CreateInterviewRequest{
		JobTitle:       "後端工程師",
		JobDescription: "需要熟悉 Go、PostgreSQL、REST API",
		UserProfile:    "有 Go 學習經驗",
		QuestionCount:  1,
	}, []llm.GeneratedQuestion{{Order: 1, Text: "問題一"}})
	if err != nil {
		t.Fatalf("CreateWithQuestions returned error: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM interviews WHERE id = $1", created.ID)
	})

	var questionID string
	if err := pool.QueryRow(ctx, `
		SELECT id
		FROM questions
		WHERE interview_id = $1 AND question_order = 1
	`, created.ID).Scan(&questionID); err != nil {
		t.Fatalf("query question id: %v", err)
	}

	repository := NewAnswerRepository(pool)
	answer, err := repository.UpsertAnswer(ctx, created.ID, questionID, "storage/audio/first.webm")
	if err != nil {
		t.Fatalf("UpsertAnswer returned error: %v", err)
	}
	if answer.ID == "" {
		t.Fatal("expected answer id")
	}
	if answer.InterviewID != created.ID {
		t.Fatalf("expected interview id %q, got %q", created.ID, answer.InterviewID)
	}
	if answer.QuestionID != questionID {
		t.Fatalf("expected question id %q, got %q", questionID, answer.QuestionID)
	}
	if answer.AudioPath == nil || *answer.AudioPath != "storage/audio/first.webm" {
		t.Fatalf("expected first audio path, got %+v", answer.AudioPath)
	}
	if answer.TranscriptText != nil {
		t.Fatalf("expected nil transcript text, got %+v", answer.TranscriptText)
	}

	updated, err := repository.UpsertAnswer(ctx, created.ID, questionID, "storage/audio/second.webm")
	if err != nil {
		t.Fatalf("second UpsertAnswer returned error: %v", err)
	}
	if updated.ID != answer.ID {
		t.Fatalf("expected same answer id after upsert, got %q then %q", answer.ID, updated.ID)
	}
	if updated.AudioPath == nil || *updated.AudioPath != "storage/audio/second.webm" {
		t.Fatalf("expected updated audio path, got %+v", updated.AudioPath)
	}

	var answerCount int
	if err := pool.QueryRow(ctx, `
		SELECT count(*)
		FROM answers
		WHERE interview_id = $1 AND question_id = $2
	`, created.ID, questionID).Scan(&answerCount); err != nil {
		t.Fatalf("count answers: %v", err)
	}
	if answerCount != 1 {
		t.Fatalf("expected one answer row after upsert, got %d", answerCount)
	}
}

func TestUpsertAnswerRejectsMissingInterview(t *testing.T) {
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

	repository := NewAnswerRepository(pool)
	_, err = repository.UpsertAnswer(ctx, "00000000-0000-0000-0000-000000000000", "00000000-0000-0000-0000-000000000001", "storage/audio/missing.webm")

	if !errors.Is(err, service.ErrInterviewNotFound) {
		t.Fatalf("expected ErrInterviewNotFound, got %v", err)
	}
}

func TestUpsertAnswerRejectsQuestionOutsideInterview(t *testing.T) {
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

	interviewRepository := NewInterviewRepository(pool)
	first, err := interviewRepository.CreateWithQuestions(ctx, model.CreateInterviewRequest{
		JobTitle:       "後端工程師",
		JobDescription: "需要熟悉 Go",
		UserProfile:    "有 Go 學習經驗",
		QuestionCount:  1,
	}, []llm.GeneratedQuestion{{Order: 1, Text: "第一場問題"}})
	if err != nil {
		t.Fatalf("create first interview: %v", err)
	}
	second, err := interviewRepository.CreateWithQuestions(ctx, model.CreateInterviewRequest{
		JobTitle:       "前端工程師",
		JobDescription: "需要熟悉 React",
		UserProfile:    "有 React 學習經驗",
		QuestionCount:  1,
	}, []llm.GeneratedQuestion{{Order: 1, Text: "第二場問題"}})
	if err != nil {
		t.Fatalf("create second interview: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM interviews WHERE id = ANY($1)", []string{first.ID, second.ID})
	})

	var secondQuestionID string
	if err := pool.QueryRow(ctx, `
		SELECT id
		FROM questions
		WHERE interview_id = $1 AND question_order = 1
	`, second.ID).Scan(&secondQuestionID); err != nil {
		t.Fatalf("query second question id: %v", err)
	}

	repository := NewAnswerRepository(pool)
	_, err = repository.UpsertAnswer(ctx, first.ID, secondQuestionID, "storage/audio/wrong-question.webm")

	if !errors.Is(err, service.ErrQuestionNotFoundForInterview) {
		t.Fatalf("expected ErrQuestionNotFoundForInterview, got %v", err)
	}
}
```

- [ ] **Step 2: Run repository tests to verify they fail**

Run with a database if available:

```powershell
$env:POSTGRES_USER='interview_ai'
$env:POSTGRES_PASSWORD='interview_ai_dev_password'
$env:POSTGRES_DB='interview_ai'
docker compose up -d postgres
docker compose run --rm migrate
$env:DATABASE_URL='postgres://interview_ai:interview_ai_dev_password@localhost:5432/interview_ai?sslmode=disable'
Push-Location backend
go test ./internal/repository -run Answer -count=1
Pop-Location
Remove-Item Env:\DATABASE_URL -ErrorAction SilentlyContinue
```

Expected: fail because `NewAnswerRepository` and service errors are not defined.

- [ ] **Step 3: Add service errors used by repository**

In `backend/internal/service/interview_service.go`, extend the existing `var (` block:

```go
	ErrQuestionNotFoundForInterview = errors.New("question not found for interview")
```

The block should contain:

```go
var (
	ErrJobTitleRequired             = errors.New("job_title is required")
	ErrJobDescriptionRequired       = errors.New("job_description is required")
	ErrUserProfileRequired          = errors.New("user_profile is required")
	ErrQuestionCountRange           = errors.New("question_count must be between 1 and 10")
	ErrInterviewNotFound            = errors.New("interview not found")
	ErrQuestionNotFoundForInterview = errors.New("question not found for interview")
)
```

- [ ] **Step 4: Implement answer repository**

Create `backend/internal/repository/answer_repository.go`:

```go
package repository

import (
	"context"
	"errors"

	"interview-ai/backend/internal/model"
	"interview-ai/backend/internal/service"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AnswerRepository struct {
	pool *pgxpool.Pool
}

func NewAnswerRepository(pool *pgxpool.Pool) *AnswerRepository {
	return &AnswerRepository{pool: pool}
}

func (r *AnswerRepository) UpsertAnswer(ctx context.Context, interviewID string, questionID string, audioPath string) (model.Answer, error) {
	var interviewExists bool
	if err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM interviews
			WHERE id = $1
		)
	`, interviewID).Scan(&interviewExists); err != nil {
		return model.Answer{}, err
	}
	if !interviewExists {
		return model.Answer{}, service.ErrInterviewNotFound
	}

	var questionExists bool
	if err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM questions
			WHERE id = $1 AND interview_id = $2
		)
	`, questionID, interviewID).Scan(&questionExists); err != nil {
		return model.Answer{}, err
	}
	if !questionExists {
		return model.Answer{}, service.ErrQuestionNotFoundForInterview
	}

	var answer model.Answer
	err := r.pool.QueryRow(ctx, `
		INSERT INTO answers (interview_id, question_id, audio_path, transcript_text)
		VALUES ($1, $2, $3, NULL)
		ON CONFLICT (interview_id, question_id)
		DO UPDATE SET audio_path = EXCLUDED.audio_path
		RETURNING id, interview_id, question_id, audio_path, transcript_text, created_at
	`, interviewID, questionID, audioPath).Scan(
		&answer.ID,
		&answer.InterviewID,
		&answer.QuestionID,
		&answer.AudioPath,
		&answer.TranscriptText,
		&answer.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Answer{}, service.ErrQuestionNotFoundForInterview
		}
		return model.Answer{}, err
	}

	return answer, nil
}
```

- [ ] **Step 5: Run repository tests**

Run:

```powershell
$env:DATABASE_URL='postgres://interview_ai:interview_ai_dev_password@localhost:5432/interview_ai?sslmode=disable'
Push-Location backend
go test ./internal/repository -run Answer -count=1
Pop-Location
Remove-Item Env:\DATABASE_URL -ErrorAction SilentlyContinue
```

Expected: answer repository tests pass.

## Task 4: Add Answer Service

**Files:**
- Create: `backend/internal/service/answer_service.go`
- Create: `backend/internal/service/answer_service_test.go`

- [ ] **Step 1: Write failing service tests**

Create `backend/internal/service/answer_service_test.go`:

```go
package service

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"interview-ai/backend/internal/model"
)

func TestUploadAnswerSavesAudioAndPersistsAnswer(t *testing.T) {
	storage := &stubAudioStorage{}
	repository := &stubAnswerRepository{
		answer: model.Answer{
			ID:          "answer-id",
			InterviewID: "interview-id",
			QuestionID:  "question-id",
			AudioPath:   stringPointer("storage/audio/interview-id/question-id.webm"),
		},
	}
	service := NewAnswerService(storage, repository)

	response, err := service.UploadAnswer(context.Background(), UploadAnswerInput{
		InterviewID: "interview-id",
		QuestionID:  "question-id",
		ContentType: "audio/webm",
		Audio:       strings.NewReader("webm-bytes"),
	})

	if err != nil {
		t.Fatalf("UploadAnswer returned error: %v", err)
	}
	if storage.interviewID != "interview-id" {
		t.Fatalf("expected storage interview id, got %q", storage.interviewID)
	}
	if storage.questionID != "question-id" {
		t.Fatalf("expected storage question id, got %q", storage.questionID)
	}
	if storage.audioBytes != "webm-bytes" {
		t.Fatalf("expected storage bytes, got %q", storage.audioBytes)
	}
	if repository.audioPath != "storage/audio/interview-id/question-id.webm" {
		t.Fatalf("expected repository audio path, got %q", repository.audioPath)
	}
	if response.ID != "answer-id" {
		t.Fatalf("expected answer id, got %q", response.ID)
	}
	if response.InterviewID != "interview-id" {
		t.Fatalf("expected interview id, got %q", response.InterviewID)
	}
	if response.QuestionID != "question-id" {
		t.Fatalf("expected question id, got %q", response.QuestionID)
	}
	if response.AudioPath != "storage/audio/interview-id/question-id.webm" {
		t.Fatalf("expected response audio path, got %q", response.AudioPath)
	}
	if response.TranscriptText != nil {
		t.Fatalf("expected nil transcript text, got %+v", response.TranscriptText)
	}
}

func TestUploadAnswerRequiresAudio(t *testing.T) {
	service := NewAnswerService(&stubAudioStorage{}, &stubAnswerRepository{})

	_, err := service.UploadAnswer(context.Background(), UploadAnswerInput{
		InterviewID: "interview-id",
		QuestionID:  "question-id",
		ContentType: "audio/webm",
		Audio:       nil,
	})

	if !errors.Is(err, ErrAudioFileRequired) {
		t.Fatalf("expected ErrAudioFileRequired, got %v", err)
	}
}

func TestUploadAnswerRequiresWebMAudio(t *testing.T) {
	service := NewAnswerService(&stubAudioStorage{}, &stubAnswerRepository{})

	_, err := service.UploadAnswer(context.Background(), UploadAnswerInput{
		InterviewID: "interview-id",
		QuestionID:  "question-id",
		ContentType: "audio/wav",
		Audio:       strings.NewReader("wav-bytes"),
	})

	if !errors.Is(err, ErrUnsupportedAudioType) {
		t.Fatalf("expected ErrUnsupportedAudioType, got %v", err)
	}
}

func TestUploadAnswerReturnsStorageError(t *testing.T) {
	service := NewAnswerService(&stubAudioStorage{err: errors.New("disk full")}, &stubAnswerRepository{})

	_, err := service.UploadAnswer(context.Background(), UploadAnswerInput{
		InterviewID: "interview-id",
		QuestionID:  "question-id",
		ContentType: "audio/webm",
		Audio:       strings.NewReader("webm-bytes"),
	})

	if !errors.Is(err, ErrSaveAnswerAudioFailed) {
		t.Fatalf("expected ErrSaveAnswerAudioFailed, got %v", err)
	}
}

func TestUploadAnswerReturnsRepositoryError(t *testing.T) {
	service := NewAnswerService(&stubAudioStorage{}, &stubAnswerRepository{err: ErrQuestionNotFoundForInterview})

	_, err := service.UploadAnswer(context.Background(), UploadAnswerInput{
		InterviewID: "interview-id",
		QuestionID:  "question-id",
		ContentType: "audio/webm",
		Audio:       strings.NewReader("webm-bytes"),
	})

	if !errors.Is(err, ErrQuestionNotFoundForInterview) {
		t.Fatalf("expected ErrQuestionNotFoundForInterview, got %v", err)
	}
}

type stubAudioStorage struct {
	interviewID string
	questionID  string
	audioBytes  string
	err         error
}

func (s *stubAudioStorage) SaveAnswerAudio(ctx context.Context, interviewID string, questionID string, file io.Reader) (string, error) {
	s.interviewID = interviewID
	s.questionID = questionID
	bytes, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}
	s.audioBytes = string(bytes)
	if s.err != nil {
		return "", s.err
	}
	return "storage/audio/interview-id/question-id.webm", nil
}

type stubAnswerRepository struct {
	interviewID string
	questionID  string
	audioPath   string
	answer      model.Answer
	err         error
}

func (s *stubAnswerRepository) UpsertAnswer(ctx context.Context, interviewID string, questionID string, audioPath string) (model.Answer, error) {
	s.interviewID = interviewID
	s.questionID = questionID
	s.audioPath = audioPath
	if s.err != nil {
		return model.Answer{}, s.err
	}
	return s.answer, nil
}

func stringPointer(value string) *string {
	return &value
}
```

- [ ] **Step 2: Run service tests to verify they fail**

Run:

```powershell
Push-Location backend
go test ./internal/service -run UploadAnswer -count=1
Pop-Location
```

Expected: fail because `NewAnswerService`, `UploadAnswerInput`, and upload errors are undefined.

- [ ] **Step 3: Implement answer service**

Create `backend/internal/service/answer_service.go`:

```go
package service

import (
	"context"
	"errors"
	"io"

	"interview-ai/backend/internal/model"
)

var (
	ErrAudioFileRequired     = errors.New("audio file is required")
	ErrUnsupportedAudioType  = errors.New("audio file must be audio/webm")
	ErrSaveAnswerAudioFailed = errors.New("failed to save answer audio")
)

type UploadAnswerInput struct {
	InterviewID string
	QuestionID  string
	ContentType string
	Audio       io.Reader
}

type AudioStorage interface {
	SaveAnswerAudio(ctx context.Context, interviewID string, questionID string, file io.Reader) (string, error)
}

type AnswerRepository interface {
	UpsertAnswer(ctx context.Context, interviewID string, questionID string, audioPath string) (model.Answer, error)
}

type AnswerService struct {
	storage    AudioStorage
	repository AnswerRepository
}

func NewAnswerService(storage AudioStorage, repository AnswerRepository) *AnswerService {
	return &AnswerService{storage: storage, repository: repository}
}

func (s *AnswerService) UploadAnswer(ctx context.Context, input UploadAnswerInput) (model.UploadAnswerResponse, error) {
	if input.Audio == nil {
		return model.UploadAnswerResponse{}, ErrAudioFileRequired
	}
	if input.ContentType != "audio/webm" {
		return model.UploadAnswerResponse{}, ErrUnsupportedAudioType
	}

	audioPath, err := s.storage.SaveAnswerAudio(ctx, input.InterviewID, input.QuestionID, input.Audio)
	if err != nil {
		return model.UploadAnswerResponse{}, ErrSaveAnswerAudioFailed
	}

	answer, err := s.repository.UpsertAnswer(ctx, input.InterviewID, input.QuestionID, audioPath)
	if err != nil {
		return model.UploadAnswerResponse{}, err
	}

	response := model.UploadAnswerResponse{
		ID:          answer.ID,
		InterviewID: answer.InterviewID,
		QuestionID:  answer.QuestionID,
		AudioPath:   audioPath,
	}
	if answer.AudioPath != nil {
		response.AudioPath = *answer.AudioPath
	}
	response.TranscriptText = answer.TranscriptText

	return response, nil
}
```

- [ ] **Step 4: Run service tests**

Run:

```powershell
Push-Location backend
go test ./internal/service -run UploadAnswer -count=1
Pop-Location
```

Expected: upload answer service tests pass.

## Task 5: Add HTTP Answer Upload Handler

**Files:**
- Create: `backend/internal/handler/answer_handler.go`
- Create or modify: `backend/internal/handler/answer_handler_test.go`
- Modify: `backend/internal/handler/interview_handler.go`
- Modify: `backend/internal/handler/interview_handler_test.go`

- [ ] **Step 1: Write failing handler tests**

Create `backend/internal/handler/answer_handler_test.go`:

```go
package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"interview-ai/backend/internal/model"
	"interview-ai/backend/internal/service"
)

func TestUploadAnswerReturnsCreatedResponse(t *testing.T) {
	answerService := &stubAnswerService{
		response: model.UploadAnswerResponse{
			ID:          "answer-id",
			InterviewID: "interview-id",
			QuestionID:  "question-id",
			AudioPath:   "storage/audio/interview-id/question-id.webm",
		},
	}
	handler := NewInterviewHandler(&stubInterviewService{}, answerService)
	body, contentType := multipartBody(t, "audio", "answer.webm", "audio/webm", "webm-bytes")
	request := httptest.NewRequest(http.MethodPost, "/interview-id/questions/question-id/answer", body)
	request.Header.Set("Content-Type", contentType)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", response.Code)
	}
	if answerService.input.InterviewID != "interview-id" {
		t.Fatalf("expected interview id, got %q", answerService.input.InterviewID)
	}
	if answerService.input.QuestionID != "question-id" {
		t.Fatalf("expected question id, got %q", answerService.input.QuestionID)
	}
	if answerService.input.ContentType != "audio/webm" {
		t.Fatalf("expected audio/webm, got %q", answerService.input.ContentType)
	}
	uploadedBytes, err := ioReadAllString(answerService.input.Audio)
	if err != nil {
		t.Fatalf("read uploaded audio: %v", err)
	}
	if uploadedBytes != "webm-bytes" {
		t.Fatalf("expected uploaded bytes, got %q", uploadedBytes)
	}

	var responseBody model.UploadAnswerResponse
	if err := json.Unmarshal(response.Body.Bytes(), &responseBody); err != nil {
		t.Fatalf("expected JSON response, got error: %v", err)
	}
	if responseBody.ID != "answer-id" {
		t.Fatalf("expected answer id, got %q", responseBody.ID)
	}
	if responseBody.AudioPath != "storage/audio/interview-id/question-id.webm" {
		t.Fatalf("expected audio path, got %q", responseBody.AudioPath)
	}
}

func TestUploadAnswerRequiresMultipartAudio(t *testing.T) {
	handler := NewInterviewHandler(&stubInterviewService{}, &stubAnswerService{})
	request := httptest.NewRequest(http.MethodPost, "/interview-id/questions/question-id/answer", strings.NewReader(""))
	request.Header.Set("Content-Type", "multipart/form-data; boundary=missing")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	assertErrorResponse(t, response, http.StatusBadRequest, "audio file is required")
}

func TestUploadAnswerMapsValidationErrors(t *testing.T) {
	handler := NewInterviewHandler(&stubInterviewService{}, &stubAnswerService{err: service.ErrUnsupportedAudioType})
	body, contentType := multipartBody(t, "audio", "answer.wav", "audio/wav", "wav-bytes")
	request := httptest.NewRequest(http.MethodPost, "/interview-id/questions/question-id/answer", body)
	request.Header.Set("Content-Type", contentType)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	assertErrorResponse(t, response, http.StatusBadRequest, "audio file must be audio/webm")
}

func TestUploadAnswerMapsNotFoundErrors(t *testing.T) {
	handler := NewInterviewHandler(&stubInterviewService{}, &stubAnswerService{err: service.ErrQuestionNotFoundForInterview})
	body, contentType := multipartBody(t, "audio", "answer.webm", "audio/webm", "webm-bytes")
	request := httptest.NewRequest(http.MethodPost, "/interview-id/questions/question-id/answer", body)
	request.Header.Set("Content-Type", contentType)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	assertErrorResponse(t, response, http.StatusNotFound, "question not found for interview")
}

func TestUploadAnswerMapsStorageErrors(t *testing.T) {
	handler := NewInterviewHandler(&stubInterviewService{}, &stubAnswerService{err: service.ErrSaveAnswerAudioFailed})
	body, contentType := multipartBody(t, "audio", "answer.webm", "audio/webm", "webm-bytes")
	request := httptest.NewRequest(http.MethodPost, "/interview-id/questions/question-id/answer", body)
	request.Header.Set("Content-Type", contentType)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	assertErrorResponse(t, response, http.StatusInternalServerError, "failed to save answer audio")
}

func multipartBody(t *testing.T, fieldName string, fileName string, contentType string, content string) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreatePart(map[string][]string{
		"Content-Disposition": {`form-data; name="` + fieldName + `"; filename="` + fileName + `"`},
		"Content-Type":        {contentType},
	})
	if err != nil {
		t.Fatalf("create multipart part: %v", err)
	}
	if _, err := part.Write([]byte(content)); err != nil {
		t.Fatalf("write multipart part: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	return body, writer.FormDataContentType()
}

func ioReadAllString(reader io.Reader) (string, error) {
	bytes, err := io.ReadAll(reader)
	return string(bytes), err
}

type stubAnswerService struct {
	input    service.UploadAnswerInput
	response model.UploadAnswerResponse
	err      error
}

func (s *stubAnswerService) UploadAnswer(ctx context.Context, input service.UploadAnswerInput) (model.UploadAnswerResponse, error) {
	s.input = input
	if s.err != nil {
		return model.UploadAnswerResponse{}, s.err
	}
	return s.response, nil
}
```

- [ ] **Step 2: Run handler tests to verify they fail**

Run:

```powershell
Push-Location backend
go test ./internal/handler -run UploadAnswer -count=1
Pop-Location
```

Expected: fail because `NewInterviewHandler` does not accept an answer service and upload routing is missing.

- [ ] **Step 3: Implement answer handler**

Create `backend/internal/handler/answer_handler.go`:

```go
package handler

import (
	"context"
	"errors"
	"log"
	"net/http"

	"interview-ai/backend/internal/model"
	"interview-ai/backend/internal/service"

	"github.com/go-chi/chi/v5"
)

const maxAnswerAudioBytes = 20 << 20

type AnswerService interface {
	UploadAnswer(ctx context.Context, input service.UploadAnswerInput) (model.UploadAnswerResponse, error)
}

func uploadAnswer(answerService AnswerService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if answerService == nil {
			writeError(w, http.StatusNotFound, "answer upload is not configured")
			return
		}

		interviewID := chi.URLParam(r, "interviewID")
		questionID := chi.URLParam(r, "questionID")

		if err := r.ParseMultipartForm(maxAnswerAudioBytes); err != nil {
			writeError(w, http.StatusBadRequest, service.ErrAudioFileRequired.Error())
			return
		}

		file, header, err := r.FormFile("audio")
		if err != nil {
			writeError(w, http.StatusBadRequest, service.ErrAudioFileRequired.Error())
			return
		}
		defer file.Close()

		contentType := header.Header.Get("Content-Type")
		response, err := answerService.UploadAnswer(r.Context(), service.UploadAnswerInput{
			InterviewID: interviewID,
			QuestionID:  questionID,
			ContentType: contentType,
			Audio:       http.MaxBytesReader(w, file, maxAnswerAudioBytes),
		})
		if err != nil {
			switch {
			case errors.Is(err, service.ErrAudioFileRequired),
				errors.Is(err, service.ErrUnsupportedAudioType):
				writeError(w, http.StatusBadRequest, err.Error())
			case errors.Is(err, service.ErrInterviewNotFound),
				errors.Is(err, service.ErrQuestionNotFoundForInterview):
				writeError(w, http.StatusNotFound, err.Error())
			case errors.Is(err, service.ErrSaveAnswerAudioFailed):
				log.Printf("save answer audio: %v", err)
				writeError(w, http.StatusInternalServerError, service.ErrSaveAnswerAudioFailed.Error())
			default:
				log.Printf("upload answer: %v", err)
				writeError(w, http.StatusInternalServerError, "failed to save answer")
			}
			return
		}

		writeJSON(w, http.StatusCreated, response)
	}
}
```

- [ ] **Step 4: Mount answer route in interview handler**

Change the signature in `backend/internal/handler/interview_handler.go` from:

```go
func NewInterviewHandler(interviewService InterviewService) http.Handler {
```

to:

```go
func NewInterviewHandler(interviewService InterviewService, answerService AnswerService) http.Handler {
```

Then update route setup:

```go
func NewInterviewHandler(interviewService InterviewService, answerService AnswerService) http.Handler {
	router := chi.NewRouter()
	router.Post("/", createInterview(interviewService))
	router.Get("/{interviewID}", getInterview(interviewService))
	router.Post("/{interviewID}/questions/{questionID}/answer", uploadAnswer(answerService))
	return router
}
```

- [ ] **Step 5: Update existing handler tests for new constructor**

In `backend/internal/handler/interview_handler_test.go`, update every `NewInterviewHandler(...)` call to pass `nil` as the second argument because existing interview tests do not exercise answer upload routing.

For populated stubs, use this shape:

```go
handler := NewInterviewHandler(&stubInterviewService{
	response: model.CreateInterviewResponse{
		ID:     "interview-id",
		Status: model.InterviewStatusQuestionsReady,
	},
}, nil)
```

For empty stubs, use:

```go
handler := NewInterviewHandler(&stubInterviewService{}, nil)
```

- [ ] **Step 6: Run handler tests**

Run:

```powershell
Push-Location backend
go test ./internal/handler -count=1
Pop-Location
```

Expected: handler tests pass.

## Task 6: Wire Answer Upload in Main

**Files:**
- Modify: `backend/cmd/api/main.go`

- [ ] **Step 1: Write failing compile check**

Run:

```powershell
Push-Location backend
go test ./cmd/api -count=1
Pop-Location
```

Expected: fail after Task 5 because `NewInterviewHandler` now requires `answerService`.

- [ ] **Step 2: Wire storage, repository, and service**

Modify imports in `backend/cmd/api/main.go` to include storage:

```go
	"interview-ai/backend/internal/storage"
```

Inside `main`, replace:

```go
	interviewRepository := repository.NewInterviewRepository(pool)
	questionGenerator := questionGeneratorForConfig(cfg)
	interviewService := service.NewInterviewService(questionGenerator, interviewRepository)
	interviewHandler := handler.NewInterviewHandler(interviewService)
```

with:

```go
	interviewRepository := repository.NewInterviewRepository(pool)
	answerRepository := repository.NewAnswerRepository(pool)
	audioStorage := storage.NewLocalAudioStorage("storage/audio")
	questionGenerator := questionGeneratorForConfig(cfg)
	interviewService := service.NewInterviewService(questionGenerator, interviewRepository)
	answerService := service.NewAnswerService(audioStorage, answerRepository)
	interviewHandler := handler.NewInterviewHandler(interviewService, answerService)
```

- [ ] **Step 3: Run backend tests**

Run:

```powershell
Push-Location backend
Remove-Item Env:\DATABASE_URL -ErrorAction SilentlyContinue
go test ./...
Pop-Location
```

Expected: all non-integration backend tests pass; repository integration tests skip without `DATABASE_URL`.

## Task 7: Manual API Verification

**Files:**
- No code changes.

- [ ] **Step 1: Start local services**

Run:

```powershell
$env:POSTGRES_USER='interview_ai'
$env:POSTGRES_PASSWORD='interview_ai_dev_password'
$env:POSTGRES_DB='interview_ai'
$env:GEMINI_API_KEY=''
$env:GEMINI_MODEL='gemini-2.5-flash'
$env:GEMINI_FALLBACK_MODEL='gemini-2.5-flash-lite'
docker compose up -d postgres
docker compose run --rm migrate
docker compose up --build -d backend
```

Expected: postgres healthy, migrations report `no change` or apply successfully, backend starts on `localhost:8080`.

- [ ] **Step 2: Create an interview**

Run:

```powershell
$createResponse = Invoke-RestMethod -Method Post -Uri http://localhost:8080/api/interviews -ContentType 'application/json' -Body '{
  "job_title": "後端工程師",
  "job_description": "需要熟悉 Go、PostgreSQL、REST API",
  "user_profile": "有 Java 和 Go 學習經驗",
  "question_count": 1
}'
$interviewID = $createResponse.id
$detail = Invoke-RestMethod -Method Get -Uri "http://localhost:8080/api/interviews/$interviewID"
$questionID = $detail.questions[0].id
$interviewID
$questionID
```

Expected: prints a non-empty interview id and question id.

- [ ] **Step 3: Upload a WebM answer**

Create a small test WebM-like file and upload it:

```powershell
Set-Content -LiteralPath .\answer.webm -Value 'webm-bytes'
$form = @{
  audio = Get-Item .\answer.webm
}
$answer = Invoke-RestMethod -Method Post -Uri "http://localhost:8080/api/interviews/$interviewID/questions/$questionID/answer" -Form $form
$answer
```

Expected response:

```text
id              : <non-empty answer id>
interview_id    : <created interview id>
question_id     : <created question id>
audio_path      : storage/audio/<interview id>/<question id>.webm
transcript_text : 
```

- [ ] **Step 4: Verify file exists in backend container**

Run:

```powershell
docker compose exec backend sh -lc "ls -l /app/storage/audio/$interviewID/$questionID.webm && cat /app/storage/audio/$interviewID/$questionID.webm"
```

Expected: file exists and prints `webm-bytes`.

- [ ] **Step 5: Verify answer row through API**

Run:

```powershell
$detailAfterUpload = Invoke-RestMethod -Method Get -Uri "http://localhost:8080/api/interviews/$interviewID"
$detailAfterUpload.answers
```

Expected: one answer with the uploaded `question_id`, `audio_path`, `transcript_text` empty/null, and `created_at`.

## Task 8: Update API and Progress Documentation

**Files:**
- Modify: `docs/API.md`
- Modify: `docs/development-progress.md`
- Modify: `docs/DEVELOPMENT_PLAN.md`

- [ ] **Step 1: Update `docs/API.md`**

After the existing `Get Interview` section, add:

````md
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
````

- [ ] **Step 2: Update `docs/development-progress.md`**

Change current status:

```md
- Current step: Step 10 - 回答音檔上傳 API
- Status: Completed
- Last updated: 2026-05-27
```

Change the Step 10 table row status to `Completed`.

Add this section after Step 9:

```md
### Step 10 - 回答音檔上傳 API

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
```

Set next step:

```md
Step 11 - 完成整場面試流程.
```

Set expected work:

```md
- Add frontend upload action after recording.
- Track uploaded answers per question.
- Move through all questions and finish the session flow.
- Update interview status to completed when all questions have answers.
```

- [ ] **Step 3: Update `docs/DEVELOPMENT_PLAN.md`**

Change current status:

```md
- Current step: Step 10 - 回答音檔上傳 API
- Status: Completed
- Last updated: 2026-05-27
```

Change the Step 10 table row status to `Completed`.

Add this section after Step 9:

```md
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
```

Set next step:

```md
Step 11 - 完成整場面試流程.
```

## Task 9: Final Verification, Review, and Commit

**Files:**
- All Step 10 files.

- [ ] **Step 1: Run backend unit tests**

Run:

```powershell
Push-Location backend
Remove-Item Env:\DATABASE_URL -ErrorAction SilentlyContinue
go test ./...
Pop-Location
```

Expected: all backend tests pass, repository integration tests skip without `DATABASE_URL`.

- [ ] **Step 2: Run backend repository integration tests**

Run:

```powershell
$env:DATABASE_URL='postgres://interview_ai:interview_ai_dev_password@localhost:5432/interview_ai?sslmode=disable'
Push-Location backend
go test ./internal/repository -count=1
Pop-Location
Remove-Item Env:\DATABASE_URL -ErrorAction SilentlyContinue
```

Expected: repository tests pass against the local migrated PostgreSQL database.

- [ ] **Step 3: Run frontend regression tests**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd test --prefix frontend
```

Expected: frontend tests pass. Step 10 should not change frontend code.

- [ ] **Step 4: Run frontend build regression**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd run build --prefix frontend
```

Expected: frontend build passes. Step 10 should not change frontend code.

- [ ] **Step 5: Run diff checks**

Run:

```powershell
git diff --check
git status --short
git diff --stat
```

Expected changed files:

- `backend/cmd/api/main.go`
- `backend/internal/handler/answer_handler.go`
- `backend/internal/handler/answer_handler_test.go`
- `backend/internal/handler/interview_handler.go`
- `backend/internal/handler/interview_handler_test.go`
- `backend/internal/model/interview.go`
- `backend/internal/repository/answer_repository.go`
- `backend/internal/repository/answer_repository_test.go`
- `backend/internal/service/answer_service.go`
- `backend/internal/service/answer_service_test.go`
- `backend/internal/service/interview_service.go`
- `backend/internal/storage/audio_storage.go`
- `backend/internal/storage/audio_storage_test.go`
- `docs/API.md`
- `docs/DEVELOPMENT_PLAN.md`
- `docs/development-progress.md`

No frontend source files should change.

- [ ] **Step 6: Request code review**

Ask a reviewer to inspect the completed Step 10 diff.

Provide:

- Scope: Step 10 backend multipart answer upload API only.
- Plan: `docs/superpowers/plans/2026-05-27-step10-answer-upload-api.md`.
- Spec: `docs/mvp-spec.md` Step 10, Section 6.5, and Section 14.2/14.4.
- Verification output summary.

Expected: no blocking or important issues. Fix any critical or important review findings before commit.

- [ ] **Step 7: Commit Step 10**

Run:

```powershell
git add backend/cmd/api/main.go backend/internal/handler backend/internal/model/interview.go backend/internal/repository backend/internal/service backend/internal/storage docs/API.md docs/DEVELOPMENT_PLAN.md docs/development-progress.md
git commit -m "feat: add answer audio upload API"
```

Expected: commit succeeds.

## Assumptions

- Step 10 is backend-only because Step 11 will wire the frontend session flow to upload answers.
- `audio/webm` is the only accepted upload content type for MVP because Step 9 records WebM and the spec recommends WebM.
- Local audio files are saved under `backend/storage/audio` when running the backend locally and `/app/storage/audio` when running inside the backend container.
- The API stores `storage/audio/{interview_id}/{question_id}.webm` as `audio_path`, matching the MVP spec examples.
- Re-uploading an answer for the same interview/question overwrites the local file path and updates the existing `answers` row through the existing unique constraint.
- Completing the interview after all answers exist is Step 11, not Step 10.
- Static audio serving is deferred until Step 12/result-page playback unless explicitly requested earlier.

## Self-Review

- Spec coverage: Covers multipart answer upload API, local audio storage, answer row creation/update, interview existence validation, question ownership validation, unsupported audio type, missing file, storage failure, API docs, progress docs, and curl/manual verification.
- Scope check: Does not implement frontend upload wiring, completed status, result page, static audio serving, STT, cloud storage, auth, or non-MVP features.
- Forbidden marker scan: No banned marker terms remain; commands and snippets are concrete.
- Type consistency: `UploadAnswerInput`, `UploadAnswerResponse`, `AnswerService`, `AnswerRepository`, and `AudioStorage` names are consistent across tasks.
- Verification: Includes unit tests, repository integration tests, backend regression tests, frontend regression checks, manual API verification, diff checks, review, and commit command.
