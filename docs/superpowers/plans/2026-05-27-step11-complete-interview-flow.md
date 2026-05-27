# Step 11 Complete Interview Flow Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Complete MVP Step 11 so the user can record and upload every interview answer from the session page, and the backend marks the interview `completed` after every question has an answer.

**Architecture:** Keep the Step 10 upload endpoint as the only audio upload API and extend its backend service/repository flow to update interview status when all questions have answers. Add a small frontend API helper for multipart uploads, persist uploaded answer state in `InterviewSessionPage`, and finish the session by navigating to `/interviews/{id}/result` without implementing the Step 12 result page contents.

**Tech Stack:** Go, chi, pgx, PostgreSQL, React, TypeScript, Vite, Vitest, React Testing Library, browser MediaRecorder, multipart/form-data.

---

## Scope

Implement only Step 11 from `docs/mvp-spec.md`:

```text
- 每題錄音並上傳。
- 所有題目回答完成後將 interview status 更新為 completed。
```

Acceptance criteria:

```text
使用者可完成一整場面試
DB 中每題都有 answer
interview status 為 completed
```

Do not implement:

- Full result page content. That is Step 12.
- Static audio file serving. That belongs with Step 12 playback.
- STT/transcript generation.
- AI scoring, feedback, retry queues, auth, user ownership, or cloud storage.
- New environment variables.
- A new endpoint just to complete interviews. Completion is derived from answers after upload.

## File Structure

- Modify `backend/internal/model/interview.go`: add `InterviewStatusCompleted`.
- Modify `backend/internal/repository/answer_repository.go`: add completion update after answers cover all questions.
- Modify `backend/internal/repository/answer_repository_test.go`: add integration tests for completed status after final answer and non-completed status before final answer.
- Modify `backend/internal/service/answer_service.go`: call repository completion check after answer upsert.
- Modify `backend/internal/service/answer_service_test.go`: verify completion check happens after successful upsert and is skipped when upload/upsert fails.
- Modify `docs/API.md`: document that `GET /api/interviews/{id}` returns `status: "completed"` after all answers exist.
- Modify `frontend/src/types/interview.ts`: add `UploadAnswerResponse`.
- Modify `frontend/src/api/interviews.ts`: add `uploadAnswerAudio`.
- Modify `frontend/src/pages/InterviewSessionPage.tsx`: store recorded `Blob`, upload the current answer, track uploaded answers by question id, prevent moving forward without upload, and finish by navigating to `/interviews/{id}/result`.
- Modify `frontend/src/App.tsx`: recognize `/interviews/:id/result` with a minimal Step 11 completion handoff view that does not list results.
- Modify `frontend/src/App.test.tsx`: add upload flow tests and result-route handoff tests.
- Modify `docs/development-progress.md`: mark Step 11 completed after verification.
- Modify `docs/DEVELOPMENT_PLAN.md`: mark Step 11 completed after verification.
- No `.env.example` change is needed.

## Backend Behavior

After `POST /api/interviews/{interview_id}/questions/{question_id}/answer` succeeds:

1. Validate the interview exists and the question belongs to it.
2. Save audio.
3. Upsert answer.
4. Count questions and distinct answers for the interview.
5. If the counts match and the question count is greater than zero, set interview `status` to `completed`.

The upload response body remains the Step 10 shape:

```json
{
  "id": "answer_uuid",
  "interview_id": "interview_uuid",
  "question_id": "question_uuid",
  "audio_path": "storage/audio/interview_uuid/question_uuid.webm",
  "transcript_text": null
}
```

Clients confirm completion through:

```http
GET /api/interviews/{interview_id}
```

Expected completed detail:

```json
{
  "id": "interview_uuid",
  "status": "completed",
  "questions": [
    {"id": "question_uuid_1", "order": 1, "text": "問題一"}
  ],
  "answers": [
    {
      "id": "answer_uuid_1",
      "question_id": "question_uuid_1",
      "audio_path": "storage/audio/interview_uuid/question_uuid_1.webm",
      "transcript_text": null,
      "created_at": "2026-05-27T06:30:04Z"
    }
  ]
}
```

## Frontend Behavior

For each session question:

1. User records an answer.
2. Page shows the local preview.
3. User clicks `上傳本題回答`.
4. Page posts `FormData` field `audio` to the Step 10 endpoint.
5. On success, the question is marked uploaded and `answers` state is updated.
6. `下一題` is enabled only after the current question has an uploaded answer.
7. On the final question, the button label is `完成面試`; clicking it after upload navigates to `/interviews/{id}/result`.

The `/interviews/{id}/result` route may show a minimal handoff message only:

```text
面試已完成
結果頁將在下一步顯示題目與回答音檔。
```

This is not the Step 12 result page. Do not display question lists or audio players in this step.

## Task 1: Backend Completion Status

**Files:**
- Modify: `backend/internal/model/interview.go`
- Modify: `backend/internal/repository/answer_repository.go`
- Modify: `backend/internal/repository/answer_repository_test.go`
- Modify: `backend/internal/service/answer_service.go`
- Modify: `backend/internal/service/answer_service_test.go`

- [ ] **Step 1: Add failing repository integration test for incomplete interview**

Append this test to `backend/internal/repository/answer_repository_test.go`:

```go
func TestCompleteInterviewIfAllQuestionsAnsweredKeepsInterviewOpenBeforeFinalAnswer(t *testing.T) {
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
		JobDescription: "需要熟悉 Go",
		UserProfile:    "有 Go 學習經驗",
		QuestionCount:  2,
	}, []llm.GeneratedQuestion{
		{Order: 1, Text: "第一題"},
		{Order: 2, Text: "第二題"},
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

	repository := NewAnswerRepository(pool)
	if _, err := repository.UpsertAnswer(ctx, created.ID, firstQuestionID, "storage/audio/first.webm"); err != nil {
		t.Fatalf("UpsertAnswer returned error: %v", err)
	}
	if err := repository.CompleteInterviewIfAllQuestionsAnswered(ctx, created.ID); err != nil {
		t.Fatalf("CompleteInterviewIfAllQuestionsAnswered returned error: %v", err)
	}

	var status string
	if err := pool.QueryRow(ctx, `
		SELECT status
		FROM interviews
		WHERE id = $1
	`, created.ID).Scan(&status); err != nil {
		t.Fatalf("query interview status: %v", err)
	}
	if status != model.InterviewStatusQuestionsReady {
		t.Fatalf("expected status %q, got %q", model.InterviewStatusQuestionsReady, status)
	}
}
```

- [ ] **Step 2: Add failing repository integration test for final answer**

Append this test to `backend/internal/repository/answer_repository_test.go`:

```go
func TestCompleteInterviewIfAllQuestionsAnsweredMarksInterviewCompleted(t *testing.T) {
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
		JobDescription: "需要熟悉 Go",
		UserProfile:    "有 Go 學習經驗",
		QuestionCount:  2,
	}, []llm.GeneratedQuestion{
		{Order: 1, Text: "第一題"},
		{Order: 2, Text: "第二題"},
	})
	if err != nil {
		t.Fatalf("CreateWithQuestions returned error: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM interviews WHERE id = $1", created.ID)
	})

	rows, err := pool.Query(ctx, `
		SELECT id
		FROM questions
		WHERE interview_id = $1
		ORDER BY question_order
	`, created.ID)
	if err != nil {
		t.Fatalf("query questions: %v", err)
	}
	defer rows.Close()

	questionIDs := make([]string, 0, 2)
	for rows.Next() {
		var questionID string
		if err := rows.Scan(&questionID); err != nil {
			t.Fatalf("scan question id: %v", err)
		}
		questionIDs = append(questionIDs, questionID)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate questions: %v", err)
	}
	if len(questionIDs) != 2 {
		t.Fatalf("expected 2 questions, got %d", len(questionIDs))
	}

	repository := NewAnswerRepository(pool)
	for index, questionID := range questionIDs {
		if _, err := repository.UpsertAnswer(ctx, created.ID, questionID, "storage/audio/answer-"+strconv.Itoa(index+1)+".webm"); err != nil {
			t.Fatalf("UpsertAnswer question %d returned error: %v", index+1, err)
		}
	}
	if err := repository.CompleteInterviewIfAllQuestionsAnswered(ctx, created.ID); err != nil {
		t.Fatalf("CompleteInterviewIfAllQuestionsAnswered returned error: %v", err)
	}

	var status string
	if err := pool.QueryRow(ctx, `
		SELECT status
		FROM interviews
		WHERE id = $1
	`, created.ID).Scan(&status); err != nil {
		t.Fatalf("query interview status: %v", err)
	}
	if status != model.InterviewStatusCompleted {
		t.Fatalf("expected status %q, got %q", model.InterviewStatusCompleted, status)
	}
}
```

Add `strconv` to the import block in `backend/internal/repository/answer_repository_test.go`:

```go
import (
	"context"
	"errors"
	"os"
	"strconv"
	"testing"
)
```

- [ ] **Step 3: Run repository tests to verify they fail**

Run:

```powershell
$env:DATABASE_URL='postgres://interview_ai:interview_ai_dev_password@localhost:5432/interview_ai?sslmode=disable'
Push-Location backend
go test ./internal/repository -run CompleteInterviewIfAllQuestionsAnswered -count=1
Pop-Location
Remove-Item Env:\DATABASE_URL -ErrorAction SilentlyContinue
```

Expected: fail because `CompleteInterviewIfAllQuestionsAnswered` and `model.InterviewStatusCompleted` are not defined.

- [ ] **Step 4: Add completed status constant**

In `backend/internal/model/interview.go`, change the status const block to:

```go
const (
	InterviewStatusCreated        = "created"
	InterviewStatusQuestionsReady = "questions_ready"
	InterviewStatusCompleted      = "completed"
)
```

- [ ] **Step 5: Implement repository completion method**

In `backend/internal/repository/answer_repository.go`, add this method after `UpsertAnswer`:

```go
func (r *AnswerRepository) CompleteInterviewIfAllQuestionsAnswered(ctx context.Context, interviewID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE interviews
		SET status = $2, updated_at = now()
		WHERE id = $1
		  AND (
			SELECT count(*)
			FROM questions
			WHERE interview_id = $1
		  ) > 0
		  AND (
			SELECT count(DISTINCT question_id)
			FROM answers
			WHERE interview_id = $1
		  ) = (
			SELECT count(*)
			FROM questions
			WHERE interview_id = $1
		  )
	`, interviewID, model.InterviewStatusCompleted)
	return err
}
```

- [ ] **Step 6: Run repository completion tests**

Run:

```powershell
$env:DATABASE_URL='postgres://interview_ai:interview_ai_dev_password@localhost:5432/interview_ai?sslmode=disable'
Push-Location backend
go test ./internal/repository -run CompleteInterviewIfAllQuestionsAnswered -count=1
Pop-Location
Remove-Item Env:\DATABASE_URL -ErrorAction SilentlyContinue
```

Expected: repository completion tests pass.

- [ ] **Step 7: Add failing service test for completion call**

Append this test to `backend/internal/service/answer_service_test.go`:

```go
func TestUploadAnswerCompletesInterviewAfterSavingAnswer(t *testing.T) {
	repository := &stubAnswerRepository{
		answer: model.Answer{
			ID:          "answer-id",
			InterviewID: "interview-id",
			QuestionID:  "question-id",
			AudioPath:   stringPointer("storage/audio/interview-id/question-id.webm"),
		},
	}
	service := NewAnswerService(&stubAudioStorage{}, repository)

	_, err := service.UploadAnswer(context.Background(), UploadAnswerInput{
		InterviewID: "interview-id",
		QuestionID:  "question-id",
		ContentType: "audio/webm",
		Audio:       strings.NewReader("webm-bytes"),
	})

	if err != nil {
		t.Fatalf("UploadAnswer returned error: %v", err)
	}
	if repository.completeInterviewID != "interview-id" {
		t.Fatalf("expected completion check for interview-id, got %q", repository.completeInterviewID)
	}
}
```

Extend `stubAnswerRepository` in the same file:

```go
type stubAnswerRepository struct {
	audioPath           string
	answer              model.Answer
	validateErr         error
	err                 error
	completeInterviewID string
	completeErr         error
}

func (r *stubAnswerRepository) CompleteInterviewIfAllQuestionsAnswered(ctx context.Context, interviewID string) error {
	r.completeInterviewID = interviewID
	return r.completeErr
}
```

- [ ] **Step 8: Run service test to verify it fails**

Run:

```powershell
Push-Location backend
go test ./internal/service -run CompletesInterviewAfterSavingAnswer -count=1
Pop-Location
```

Expected: fail because `AnswerRepository` does not require or call `CompleteInterviewIfAllQuestionsAnswered`.

- [ ] **Step 9: Update service repository interface and upload flow**

In `backend/internal/service/answer_service.go`, change `AnswerRepository` to:

```go
type AnswerRepository interface {
	EnsureQuestionForInterview(ctx context.Context, interviewID string, questionID string) error
	UpsertAnswer(ctx context.Context, interviewID string, questionID string, audioPath string) (model.Answer, error)
	CompleteInterviewIfAllQuestionsAnswered(ctx context.Context, interviewID string) error
}
```

In `UploadAnswer`, after successful `UpsertAnswer`, add:

```go
	if err := s.repository.CompleteInterviewIfAllQuestionsAnswered(ctx, input.InterviewID); err != nil {
		return model.UploadAnswerResponse{}, err
	}
```

The block should be placed before constructing `model.UploadAnswerResponse`.

- [ ] **Step 10: Run backend focused tests**

Run:

```powershell
Push-Location backend
go test ./internal/service ./internal/handler -count=1
Pop-Location
```

Expected: service and handler tests pass.

## Task 2: Frontend API Helper

**Files:**
- Modify: `frontend/src/types/interview.ts`
- Modify: `frontend/src/api/interviews.ts`
- Modify: `frontend/src/App.test.tsx`

- [ ] **Step 1: Add failing frontend API expectation through session test**

In `frontend/src/App.test.tsx`, add this test before `moves between session questions with previous and next buttons`:

```tsx
it('uploads the recorded answer for the current question', async () => {
  const media = installMediaRecorderMock()
  installObjectURLMock()
  mockPathname('/interviews/interview-123/session')
  const fetchMock = vi
    .fn()
    .mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          id: 'interview-123',
          job_title: '後端工程師',
          job_description: '需要熟悉 Go、PostgreSQL、REST API',
          user_profile: '有 Java 和 Go 學習經驗',
          question_count: 1,
          status: 'questions_ready',
          questions: [
            { id: 'question-1', order: 1, text: '請介紹你過去與後端開發相關的經驗。' },
          ],
          answers: [],
        }),
        { headers: { 'Content-Type': 'application/json' } },
      ),
    )
    .mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          id: 'answer-1',
          interview_id: 'interview-123',
          question_id: 'question-1',
          audio_path: 'storage/audio/interview-123/question-1.webm',
          transcript_text: null,
        }),
        { status: 201, headers: { 'Content-Type': 'application/json' } },
      ),
    )
  vi.stubGlobal('fetch', fetchMock)

  render(<App />)

  expect(await screen.findByText('請介紹你過去與後端開發相關的經驗。')).toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: '開始錄音' }))

  await waitFor(() => {
    expect(media.recorders).toHaveLength(1)
  })
  fireEvent.click(screen.getByRole('button', { name: '停止錄音' }))
  fireEvent.click(await screen.findByRole('button', { name: '上傳本題回答' }))

  await waitFor(() => {
    expect(fetchMock).toHaveBeenCalledTimes(2)
  })
  expect(fetchMock).toHaveBeenLastCalledWith(
    '/api/interviews/interview-123/questions/question-1/answer',
    expect.objectContaining({
      method: 'POST',
      body: expect.any(FormData),
    }),
  )
  expect(await screen.findByText('本題回答已上傳')).toBeInTheDocument()
})
```

- [ ] **Step 2: Run frontend test to verify it fails**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd test --prefix frontend -- --runInBand
```

Expected: fail because there is no `上傳本題回答` button and no upload API helper.

- [ ] **Step 3: Add upload response type**

In `frontend/src/types/interview.ts`, after `Answer`, add:

```ts
export type UploadAnswerResponse = {
  id: string
  interview_id: string
  question_id: string
  audio_path: string
  transcript_text: string | null
}
```

- [ ] **Step 4: Add upload API helper**

In `frontend/src/api/interviews.ts`, change the import to:

```ts
import type {
  CreateInterviewRequest,
  CreateInterviewResponse,
  InterviewDetail,
  UploadAnswerResponse,
} from '../types/interview'
```

Add this function after `getInterview`:

```ts
export async function uploadAnswerAudio(
  interviewID: string,
  questionID: string,
  audio: Blob,
): Promise<UploadAnswerResponse> {
  const formData = new FormData()
  formData.append('audio', audio, 'answer.webm')

  const response = await fetch(
    `${API_BASE_URL}/api/interviews/${interviewID}/questions/${questionID}/answer`,
    {
      method: 'POST',
      body: formData,
    },
  )

  return parseJSONResponse<UploadAnswerResponse>(response, '上傳回答失敗')
}
```

Do not set a `Content-Type` header. The browser must add the multipart boundary.

## Task 3: Frontend Session Upload Flow

**Files:**
- Modify: `frontend/src/pages/InterviewSessionPage.tsx`
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/App.test.tsx`

- [ ] **Step 1: Store recorded Blob**

In `frontend/src/pages/InterviewSessionPage.tsx`, change imports to:

```ts
import { getInterview, uploadAnswerAudio } from '../api/interviews'
import type { Answer, InterviewDetail } from '../types/interview'
```

Add state near the existing recording state:

```ts
const [recordedAnswerBlob, setRecordedAnswerBlob] = useState<Blob | null>(null)
const [uploadedAnswersByQuestionID, setUploadedAnswersByQuestionID] = useState<Record<string, Answer>>({})
const [isUploadingAnswer, setIsUploadingAnswer] = useState(false)
const [uploadError, setUploadError] = useState<string | null>(null)
```

After `const currentQuestion = questions[currentQuestionIndex]`, add:

```ts
const currentUploadedAnswer = currentQuestion
  ? uploadedAnswersByQuestionID[currentQuestion.id]
  : undefined
const canUploadCurrentAnswer = Boolean(currentQuestion && recordedAnswerBlob && !isUploadingAnswer)
const canMoveToNextQuestion = Boolean(currentQuestion && currentUploadedAnswer && !isUploadingAnswer)
```

- [ ] **Step 2: Initialize uploaded answer state from loaded interview**

Inside `loadInterview`, before `setInterview(detail)`, add:

```ts
const uploadedAnswers = detail.answers.reduce<Record<string, Answer>>((answersByQuestionID, answer) => {
  answersByQuestionID[answer.question_id] = answer
  return answersByQuestionID
}, {})
```

Then set state:

```ts
setInterview(detail)
setUploadedAnswersByQuestionID(uploadedAnswers)
setCurrentQuestionIndex(0)
```

- [ ] **Step 3: Keep recorded Blob after stop**

Inside `recorder.onstop`, replace:

```ts
const recordedBlob = new Blob(recordedChunksRef.current, { type: 'audio/webm' })
const recordedURL = URL.createObjectURL(recordedBlob)
recordedAnswerURLRef.current = recordedURL
setRecordedAnswerURL(recordedURL)
```

with:

```ts
const recordedBlob = new Blob(recordedChunksRef.current, { type: 'audio/webm' })
const recordedURL = URL.createObjectURL(recordedBlob)
recordedAnswerURLRef.current = recordedURL
setRecordedAnswerBlob(recordedBlob)
setRecordedAnswerURL(recordedURL)
setUploadError(null)
```

In `revokeRecordedAnswerURL`, add:

```ts
setRecordedAnswerBlob(null)
setUploadError(null)
```

- [ ] **Step 4: Add upload action**

Add this function before `stopAnswerRecording`:

```ts
async function uploadCurrentAnswer() {
  if (!currentQuestion || !recordedAnswerBlob || isUploadingAnswer) {
    return
  }

  setIsUploadingAnswer(true)
  setUploadError(null)

  try {
    const uploadedAnswer = await uploadAnswerAudio(interviewID, currentQuestion.id, recordedAnswerBlob)
    setUploadedAnswersByQuestionID((answersByQuestionID) => ({
      ...answersByQuestionID,
      [uploadedAnswer.question_id]: {
        id: uploadedAnswer.id,
        question_id: uploadedAnswer.question_id,
        audio_path: uploadedAnswer.audio_path,
        transcript_text: uploadedAnswer.transcript_text,
        created_at: new Date().toISOString(),
      },
    }))
  } catch (error) {
    setUploadError(error instanceof Error ? error.message : '上傳回答失敗')
  } finally {
    setIsUploadingAnswer(false)
  }
}
```

- [ ] **Step 5: Add upload controls to UI**

Inside the recording panel, after the audio preview block, add:

```tsx
{recordedAnswerURL ? (
  <div className="mt-4 flex flex-col gap-3 sm:flex-row sm:items-center">
    <button
      type="button"
      onClick={uploadCurrentAnswer}
      disabled={!canUploadCurrentAnswer || Boolean(currentUploadedAnswer)}
      className="min-h-11 rounded-md bg-teal-700 px-5 py-2 text-sm font-semibold text-white hover:bg-teal-800 disabled:cursor-not-allowed disabled:opacity-50"
    >
      {isUploadingAnswer ? '上傳中' : '上傳本題回答'}
    </button>
    {currentUploadedAnswer ? (
      <p className="text-sm font-medium text-teal-700">本題回答已上傳</p>
    ) : null}
  </div>
) : null}
{uploadError ? <p className="mt-3 text-sm text-red-700">{uploadError}</p> : null}
```

- [ ] **Step 6: Gate question navigation and finish session**

Replace the `下一題` button block with:

```tsx
<button
  type="button"
  onClick={() => {
    if (!canMoveToNextQuestion) {
      return
    }
    stopQuestionPlayback()
    resetAnswerRecording()
    if (isLastQuestion) {
      window.history.pushState({}, '', `/interviews/${interviewID}/result`)
      window.dispatchEvent(new PopStateEvent('popstate'))
      return
    }
    setCurrentQuestionIndex((index) => Math.min(index + 1, questions.length - 1))
  }}
  disabled={!canMoveToNextQuestion}
  className="min-h-11 rounded-md bg-teal-700 px-5 py-2 text-sm font-semibold text-white hover:bg-teal-800 disabled:cursor-not-allowed disabled:opacity-50"
>
  {isLastQuestion ? '完成面試' : '下一題'}
</button>
```

Keep `上一題` enabled by its existing first-question logic, but call `resetAnswerRecording()` when moving backward.

- [ ] **Step 7: Add minimal result handoff route**

In `frontend/src/App.tsx`, add this route match before the detail match:

```ts
const resultMatch = pathname.match(/^\/interviews\/([^/]+)\/result$/)
if (resultMatch) {
  return { name: 'result' as const, interviewID: decodeURIComponent(resultMatch[1]) }
}
```

Add this render branch before the detail branch:

```tsx
if (route.name === 'result') {
  return (
    <main className="min-h-screen bg-slate-50 text-slate-950">
      <section className="mx-auto flex min-h-screen w-full max-w-3xl flex-col justify-center px-6 py-12">
        <p className="text-sm font-semibold uppercase tracking-wide text-teal-700">
          Interview Complete
        </p>
        <h1 className="mt-4 text-4xl font-bold leading-tight">面試已完成</h1>
        <p className="mt-5 text-lg leading-8 text-slate-700">
          結果頁將在下一步顯示題目與回答音檔。
        </p>
        <a
          href={`/interviews/${route.interviewID}`}
          className="mt-8 inline-flex min-h-11 items-center rounded-md bg-teal-700 px-5 py-2 text-sm font-semibold text-white hover:bg-teal-800"
        >
          返回面試詳情
        </a>
      </section>
    </main>
  )
}
```

- [ ] **Step 8: Add final-question navigation test**

In `frontend/src/App.test.tsx`, add:

```tsx
it('finishes the session after uploading the final answer', async () => {
  const media = installMediaRecorderMock()
  installObjectURLMock()
  mockPathname('/interviews/interview-123/session')
  const fetchMock = vi
    .fn()
    .mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          id: 'interview-123',
          job_title: '後端工程師',
          job_description: '需要熟悉 Go、PostgreSQL、REST API',
          user_profile: '有 Java 和 Go 學習經驗',
          question_count: 1,
          status: 'questions_ready',
          questions: [
            { id: 'question-1', order: 1, text: '請介紹你過去與後端開發相關的經驗。' },
          ],
          answers: [],
        }),
        { headers: { 'Content-Type': 'application/json' } },
      ),
    )
    .mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          id: 'answer-1',
          interview_id: 'interview-123',
          question_id: 'question-1',
          audio_path: 'storage/audio/interview-123/question-1.webm',
          transcript_text: null,
        }),
        { status: 201, headers: { 'Content-Type': 'application/json' } },
      ),
    )
  vi.stubGlobal('fetch', fetchMock)

  render(<App />)

  expect(await screen.findByText('請介紹你過去與後端開發相關的經驗。')).toBeInTheDocument()
  expect(screen.getByRole('button', { name: '完成面試' })).toBeDisabled()
  fireEvent.click(screen.getByRole('button', { name: '開始錄音' }))

  await waitFor(() => {
    expect(media.recorders).toHaveLength(1)
  })
  fireEvent.click(screen.getByRole('button', { name: '停止錄音' }))
  fireEvent.click(await screen.findByRole('button', { name: '上傳本題回答' }))

  expect(await screen.findByText('本題回答已上傳')).toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: '完成面試' }))

  expect(await screen.findByRole('heading', { name: '面試已完成' })).toBeInTheDocument()
  expect(window.location.pathname).toBe('/interviews/interview-123/result')
})
```

- [ ] **Step 9: Run frontend tests**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd test --prefix frontend
```

Expected: all frontend tests pass.

## Task 4: Manual End-to-End Verification

**Files:**
- No code changes.

- [ ] **Step 1: Start Docker services**

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
docker compose up --build -d backend frontend
```

Expected: backend is available at `http://localhost:8080`; frontend is available at `http://localhost:5173`.

- [ ] **Step 2: Verify backend completion with API**

Create a one-question interview, upload one answer, and read status:

```powershell
New-Item -ItemType Directory -Path .cache -Force | Out-Null
Set-Content -LiteralPath .cache\answer.webm -Value 'webm-bytes' -NoNewline
$createResponse = Invoke-RestMethod -Method Post -Uri http://localhost:8080/api/interviews -ContentType 'application/json' -Body '{
  "job_title": "後端工程師",
  "job_description": "需要熟悉 Go、PostgreSQL、REST API",
  "user_profile": "有 Java 和 Go 學習經驗",
  "question_count": 1
}'
$detail = Invoke-RestMethod -Method Get -Uri "http://localhost:8080/api/interviews/$($createResponse.id)"
$questionID = $detail.questions[0].id
curl.exe -s -X POST -F "audio=@.cache/answer.webm;type=audio/webm" "http://localhost:8080/api/interviews/$($createResponse.id)/questions/$questionID/answer"
$afterUpload = Invoke-RestMethod -Method Get -Uri "http://localhost:8080/api/interviews/$($createResponse.id)"
$afterUpload.status
$afterUpload.answers.Count
```

Expected output:

```text
completed
1
```

- [ ] **Step 3: Verify frontend workflow in browser**

Manual browser steps:

```text
1. Open http://localhost:5173/interviews/new.
2. Create an interview with question_count = 1.
3. Open the session page.
4. Record a short answer.
5. Confirm the audio preview appears.
6. Click 上傳本題回答.
7. Confirm 本題回答已上傳 appears.
8. Click 完成面試.
9. Confirm the browser navigates to /interviews/{id}/result and shows 面試已完成.
10. Run docker compose exec backend sh -lc "find /app/storage/audio -name '*.webm' -print" and confirm at least one uploaded file exists.
```

## Task 5: Documentation and Final Verification

**Files:**
- Modify: `docs/API.md`
- Modify: `docs/development-progress.md`
- Modify: `docs/DEVELOPMENT_PLAN.md`

- [ ] **Step 1: Update `docs/API.md`**

In the `Upload Answer Audio` section, after the success response, add:

```md
Completion behavior:

- After each successful upload, the backend checks whether every question in the interview has an answer.
- When all questions have answers, `GET /api/interviews/{interview_id}` returns `"status":"completed"`.
```

- [ ] **Step 2: Update progress docs**

In both `docs/development-progress.md` and `docs/DEVELOPMENT_PLAN.md`, change current status:

```md
- Current step: Step 11 - 完成整場面試流程
- Status: Completed
- Last updated: 2026-05-27
```

Change the Step 11 table row status to `Completed`.

Add this Step 11 completion section after Step 10:

```md
### Step 11 - 完成整場面試流程

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
```

Set next step:

```md
Step 12 - 建立面試結果頁.
```

Expected work:

```md
- Load completed interview details on `/interviews/{id}/result`.
- Display interview metadata, each question, and each uploaded answer.
- Add playable audio controls after static audio serving or playable URLs are available.
- Show `transcript_text` or `尚未轉錄`.
```

- [ ] **Step 3: Run final verification**

Run:

```powershell
Push-Location backend
Remove-Item Env:\DATABASE_URL -ErrorAction SilentlyContinue
go test ./...
Pop-Location

$env:DATABASE_URL='postgres://interview_ai:interview_ai_dev_password@localhost:5432/interview_ai?sslmode=disable'
Push-Location backend
go test ./internal/repository -count=1
Pop-Location
Remove-Item Env:\DATABASE_URL -ErrorAction SilentlyContinue

C:\nvm4w\nodejs\npm.cmd test --prefix frontend
C:\nvm4w\nodejs\npm.cmd run build --prefix frontend
git diff --check
git status --short
git diff --stat
```

Expected:

- Backend tests pass.
- Repository integration tests pass.
- Frontend tests pass.
- Frontend build passes.
- `git diff --check` exits 0.
- Changed files are limited to backend completion logic, frontend upload/session flow, and docs.

- [ ] **Step 4: Commit Step 11**

Run:

```powershell
git add backend/internal/model/interview.go backend/internal/repository/answer_repository.go backend/internal/repository/answer_repository_test.go backend/internal/service/answer_service.go backend/internal/service/answer_service_test.go frontend/src/App.tsx frontend/src/App.test.tsx frontend/src/api/interviews.ts frontend/src/pages/InterviewSessionPage.tsx frontend/src/types/interview.ts docs/API.md docs/DEVELOPMENT_PLAN.md docs/development-progress.md
git commit -m "feat: complete interview answer upload flow"
```

Expected: commit succeeds.

## Assumptions

- Step 11 can use the existing Step 10 upload endpoint and does not need a separate complete-interview endpoint.
- The upload response does not need to include interview status; the canonical status remains available through `GET /api/interviews/{id}`.
- Existing answer upsert behavior supports re-uploading the same question.
- A minimal `/interviews/{id}/result` handoff route is acceptable because Step 12 owns the full result page with audio playback.
- Local recordings remain `audio/webm`, matching the Step 10 backend validation.

## Self-Review

- Spec coverage: Covers recording then upload per question, every question having an answer, and backend status becoming `completed`.
- Scope check: Does not implement the Step 12 result page, audio serving, STT, scoring, auth, or non-MVP features.
- Forbidden marker scan: No banned marker terms remain; every code-changing step includes concrete code or exact commands.
- Type consistency: `UploadAnswerResponse`, `uploadAnswerAudio`, `InterviewStatusCompleted`, and `CompleteInterviewIfAllQuestionsAnswered` are defined before later tasks use them.
- Verification: Includes backend unit tests, repository integration tests, frontend tests, frontend build, Docker Compose API verification, manual browser verification, diff checks, and commit command.
