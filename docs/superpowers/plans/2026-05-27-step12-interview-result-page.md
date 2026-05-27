# Step 12 Interview Result Page Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Complete MVP Step 12 so a completed interview result page displays interview metadata, every question, transcript placeholders, and playable uploaded answer audio.

**Architecture:** Keep `GET /api/interviews/{id}` as the data source for result content. Add a small backend static audio route that serves files already saved by the Step 10 upload flow, then replace the Step 11 result handoff in `App.tsx` with a dedicated `InterviewResultPage` component that maps answer audio paths to playable URLs. Do not add STT, scoring, auth, cloud storage, or any new result-generation endpoint.

**Tech Stack:** Go, chi, `http.FileServer`, React, TypeScript, Vite, Vitest, React Testing Library, HTML `<audio>` controls.

---

## Scope

Implement only Step 12 from `docs/mvp-spec.md`:

```text
- 建立 `/interviews/{id}/result`。
- 顯示面試資訊、題目、回答音檔播放器。
```

Acceptance criteria:

```text
使用者可查看完整面試結果
每題回答音檔可播放
```

Do not implement:

- Speech-to-text generation.
- AI scoring or feedback.
- Download/export.
- Authentication or ownership checks.
- Cloud object storage.
- New database tables or migrations.
- New environment variables.
- A separate result API. Use the existing `GET /api/interviews/{id}`.

## File Structure

- Modify `backend/cmd/api/main.go`: add `/audio/*` static file serving from local `storage/audio`.
- Modify `backend/cmd/api/main_test.go`: add static audio route tests.
- Create `frontend/src/pages/InterviewResultPage.tsx`: load interview details and render result UI.
- Modify `frontend/src/App.tsx`: replace inline Step 11 handoff result branch with `InterviewResultPage`.
- Modify `frontend/src/App.test.tsx`: add result-page tests and remove/replace the Step 11 handoff expectation.
- Modify `docs/API.md`: document `GET /audio/{interview_id}/{question_id}.webm`.
- Modify `docs/development-progress.md`: mark Step 12 completed after verification.
- Modify `docs/DEVELOPMENT_PLAN.md`: mark Step 12 completed after verification.
- No `.env.example` change is needed.

## Backend Behavior

Serve uploaded answer audio from the existing local storage path:

```http
GET /audio/{interview_id}/{question_id}.webm
```

Expected behavior:

- `GET /audio/interview-id/question-id.webm` reads `storage/audio/interview-id/question-id.webm`.
- Existing files return `200 OK` and `Content-Type: audio/webm`.
- Missing files return `404 Not Found`.
- This route is read-only and does not expose directory listing as part of the UI.

## Frontend Behavior

For `/interviews/{id}/result`:

1. Load interview details with `getInterview(id)`.
2. Show loading and error states consistent with existing pages.
3. Display job title, job description, user profile, and interview status.
4. Display every question in question order.
5. For each question, find the matching answer by `question_id`.
6. If the answer has `audio_path`, show an `<audio controls>` element using the public `/audio/...` URL.
7. If the answer has `transcript_text`, show it.
8. If `transcript_text` is `null`, show `尚未轉錄`.
9. If a question has no answer, show `尚未上傳回答`.

`audio_path` values returned by the API currently look like:

```text
storage/audio/interview_uuid/question_uuid.webm
```

The frontend should convert those to browser URLs:

```text
/audio/interview_uuid/question_uuid.webm
```

## Task 1: Backend Static Audio Route

**Files:**
- Modify: `backend/cmd/api/main.go`
- Modify: `backend/cmd/api/main_test.go`

- [ ] **Step 1: Add failing route test for serving existing audio**

Append this test to `backend/cmd/api/main_test.go`:

```go
func TestAudioRouteServesUploadedAudio(t *testing.T) {
	tempDir := t.TempDir()
	audioDir := filepath.Join(tempDir, "audio")
	if err := os.MkdirAll(filepath.Join(audioDir, "interview-id"), 0o755); err != nil {
		t.Fatalf("create audio dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(audioDir, "interview-id", "question-id.webm"), []byte("webm-bytes"), 0o644); err != nil {
		t.Fatalf("write audio file: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/audio/interview-id/question-id.webm", nil)
	response := httptest.NewRecorder()

	newRouter(nil, audioDir).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if response.Body.String() != "webm-bytes" {
		t.Fatalf("expected audio bytes, got %q", response.Body.String())
	}
	if contentType := response.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "audio/webm") {
		t.Fatalf("expected audio/webm content type, got %q", contentType)
	}
}
```

Update the import block in `backend/cmd/api/main_test.go` to include:

```go
import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)
```

- [ ] **Step 2: Add failing route test for missing audio**

Append this test to `backend/cmd/api/main_test.go`:

```go
func TestAudioRouteReturnsNotFoundForMissingAudio(t *testing.T) {
	audioDir := t.TempDir()
	request := httptest.NewRequest(http.MethodGet, "/audio/interview-id/missing.webm", nil)
	response := httptest.NewRecorder()

	newRouter(nil, audioDir).ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", response.Code)
	}
}
```

- [ ] **Step 3: Run audio route tests to verify they fail**

Run:

```powershell
Push-Location backend
go test ./cmd/api -run AudioRoute -count=1
Pop-Location
```

Expected: fail because `newRouter` does not accept `audioDir` and no `/audio` route exists.

- [ ] **Step 4: Update router signature and call site**

In `backend/cmd/api/main.go`, change:

```go
if err := http.ListenAndServe(":8080", newRouter(interviewHandler)); err != nil {
```

to:

```go
if err := http.ListenAndServe(":8080", newRouter(interviewHandler, "storage/audio")); err != nil {
```

Change:

```go
func newRouter(interviewHandler http.Handler) http.Handler {
```

to:

```go
func newRouter(interviewHandler http.Handler, audioDir string) http.Handler {
```

Update existing tests that call `newRouter(nil)` to call:

```go
newRouter(nil, t.TempDir())
```

- [ ] **Step 5: Add static audio route**

In `backend/cmd/api/main.go`, inside `newRouter` after the `/health` route and before the API mount, add:

```go
	if audioDir != "" {
		audioFileServer := http.StripPrefix("/audio/", http.FileServer(http.Dir(audioDir)))
		router.Get("/audio/*", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "audio/webm")
			audioFileServer.ServeHTTP(w, r)
		})
	}
```

- [ ] **Step 6: Run backend focused tests**

Run:

```powershell
Push-Location backend
go test ./cmd/api -count=1
Pop-Location
```

Expected: `cmd/api` tests pass.

## Task 2: Frontend Result Page

**Files:**
- Create: `frontend/src/pages/InterviewResultPage.tsx`
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/App.test.tsx`

- [ ] **Step 1: Add failing result page content test**

In `frontend/src/App.test.tsx`, add this test near the other route tests:

```tsx
it('loads the completed interview result page with playable answers', async () => {
  mockPathname('/interviews/interview-123/result')
  vi.stubGlobal(
    'fetch',
    mockFetchOnce({
      id: 'interview-123',
      job_title: '後端工程師',
      job_description: '需要熟悉 Go、PostgreSQL、REST API',
      user_profile: '有 Java 和 Go 學習經驗',
      question_count: 2,
      status: 'completed',
      questions: [
        { id: 'question-1', order: 1, text: '請介紹你過去與後端開發相關的經驗。' },
        { id: 'question-2', order: 2, text: '你如何設計一個 REST API？' },
      ],
      answers: [
        {
          id: 'answer-1',
          question_id: 'question-1',
          audio_path: 'storage/audio/interview-123/question-1.webm',
          transcript_text: null,
          created_at: '2026-05-27T06:30:04Z',
        },
        {
          id: 'answer-2',
          question_id: 'question-2',
          audio_path: 'storage/audio/interview-123/question-2.webm',
          transcript_text: '我會先確認需求，再設計 resource 與錯誤格式。',
          created_at: '2026-05-27T06:31:04Z',
        },
      ],
    }),
  )

  render(<App />)

  expect(await screen.findByRole('heading', { name: '面試結果' })).toBeInTheDocument()
  expect(screen.getByRole('heading', { name: '後端工程師' })).toBeInTheDocument()
  expect(screen.getByText('需要熟悉 Go、PostgreSQL、REST API')).toBeInTheDocument()
  expect(screen.getByText('有 Java 和 Go 學習經驗')).toBeInTheDocument()
  expect(screen.getByText('completed')).toBeInTheDocument()
  expect(screen.getByText('請介紹你過去與後端開發相關的經驗。')).toBeInTheDocument()
  expect(screen.getByText('你如何設計一個 REST API？')).toBeInTheDocument()
  expect(screen.getByText('尚未轉錄')).toBeInTheDocument()
  expect(screen.getByText('我會先確認需求，再設計 resource 與錯誤格式。')).toBeInTheDocument()
  expect(screen.getByLabelText('問題 1 回答音檔')).toHaveAttribute(
    'src',
    '/audio/interview-123/question-1.webm',
  )
  expect(screen.getByLabelText('問題 2 回答音檔')).toHaveAttribute(
    'src',
    '/audio/interview-123/question-2.webm',
  )
})
```

- [ ] **Step 2: Add failing missing-answer display test**

In `frontend/src/App.test.tsx`, add:

```tsx
it('shows a missing-answer state on the result page', async () => {
  mockPathname('/interviews/interview-123/result')
  vi.stubGlobal(
    'fetch',
    mockFetchOnce({
      id: 'interview-123',
      job_title: '後端工程師',
      job_description: '需要熟悉 Go',
      user_profile: '有 Go 學習經驗',
      question_count: 1,
      status: 'completed',
      questions: [{ id: 'question-1', order: 1, text: '第一題' }],
      answers: [],
    }),
  )

  render(<App />)

  expect(await screen.findByText('第一題')).toBeInTheDocument()
  expect(screen.getByText('尚未上傳回答')).toBeInTheDocument()
  expect(screen.queryByLabelText('問題 1 回答音檔')).not.toBeInTheDocument()
})
```

- [ ] **Step 3: Add failing result page loading error test**

In `frontend/src/App.test.tsx`, add:

```tsx
it('shows an API error when loading a result page fails', async () => {
  mockPathname('/interviews/missing/result')
  vi.stubGlobal('fetch', mockFetchOnce({ error: 'interview not found' }, { status: 404 }))

  render(<App />)

  expect(await screen.findByText('interview not found')).toBeInTheDocument()
})
```

- [ ] **Step 4: Run frontend tests to verify they fail**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd test --prefix frontend
```

Expected: fail because `/interviews/{id}/result` still renders the Step 11 handoff and does not call `getInterview`.

- [ ] **Step 5: Create the result page component**

Create `frontend/src/pages/InterviewResultPage.tsx`:

```tsx
import { useEffect, useMemo, useState } from 'react'

import { getInterview } from '../api/interviews'
import type { Answer, InterviewDetail } from '../types/interview'

type InterviewResultPageProps = {
  interviewID: string
}

function answerAudioURL(audioPath: string | null) {
  if (!audioPath) {
    return null
  }

  return '/' + audioPath.replace(/^storage\/audio\//, 'audio/')
}

export default function InterviewResultPage({ interviewID }: InterviewResultPageProps) {
  const [interview, setInterview] = useState<InterviewDetail | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let isMounted = true

    async function loadInterview() {
      setIsLoading(true)
      setError(null)

      try {
        const detail = await getInterview(interviewID)
        if (isMounted) {
          setInterview(detail)
        }
      } catch (error) {
        if (isMounted) {
          setError(error instanceof Error ? error.message : '載入面試結果失敗')
        }
      } finally {
        if (isMounted) {
          setIsLoading(false)
        }
      }
    }

    loadInterview()

    return () => {
      isMounted = false
    }
  }, [interviewID])

  const answersByQuestionID = useMemo(() => {
    return (interview?.answers ?? []).reduce<Record<string, Answer>>((answers, answer) => {
      answers[answer.question_id] = answer
      return answers
    }, {})
  }, [interview?.answers])

  return (
    <main className="min-h-screen bg-slate-50 text-slate-950">
      <section className="mx-auto w-full max-w-5xl px-6 py-10">
        <a
          href={`/interviews/${interviewID}`}
          className="text-sm font-medium text-teal-700 hover:text-teal-800"
        >
          返回面試詳情
        </a>

        {isLoading ? <p className="mt-8 text-slate-600">載入面試結果中...</p> : null}

        {error ? (
          <div className="mt-8 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
            {error}
          </div>
        ) : null}

        {interview ? (
          <div className="mt-8">
            <div className="border-b border-slate-200 pb-6">
              <p className="text-sm font-semibold uppercase tracking-wide text-teal-700">
                Interview Result
              </p>
              <h1 className="mt-3 text-3xl font-bold leading-tight">面試結果</h1>
              <h2 className="mt-4 text-2xl font-semibold text-slate-950">
                {interview.job_title}
              </h2>
              <p className="mt-3 max-w-3xl leading-7 text-slate-700">
                {interview.job_description}
              </p>
              <div className="mt-5 grid gap-4 md:grid-cols-[1fr_auto] md:items-start">
                <div>
                  <p className="text-sm font-semibold text-slate-700">個人資訊</p>
                  <p className="mt-2 leading-7 text-slate-700">{interview.user_profile}</p>
                </div>
                <span className="inline-flex w-fit rounded-md border border-teal-200 bg-teal-50 px-3 py-1 text-sm font-medium text-teal-800">
                  {interview.status}
                </span>
              </div>
            </div>

            <section className="mt-8">
              <h3 className="text-xl font-semibold text-slate-900">題目與回答</h3>
              <ol className="mt-4 space-y-4">
                {interview.questions.map((question) => {
                  const answer = answersByQuestionID[question.id]
                  const audioURL = answerAudioURL(answer?.audio_path ?? null)

                  return (
                    <li
                      key={question.id}
                      className="rounded-lg border border-slate-200 bg-white p-5 shadow-sm"
                    >
                      <p className="text-sm font-semibold text-teal-700">問題 {question.order}</p>
                      <p className="mt-2 text-lg font-medium leading-8 text-slate-950">
                        {question.text}
                      </p>

                      {answer ? (
                        <div className="mt-4 space-y-4">
                          {audioURL ? (
                            <div>
                              <p className="mb-2 text-sm font-medium text-slate-700">回答音檔</p>
                              <audio
                                aria-label={`問題 ${question.order} 回答音檔`}
                                controls
                                src={audioURL}
                              />
                            </div>
                          ) : (
                            <p className="text-sm text-slate-600">尚未上傳回答</p>
                          )}
                          <div>
                            <p className="text-sm font-medium text-slate-700">轉錄文字</p>
                            <p className="mt-2 leading-7 text-slate-700">
                              {answer.transcript_text ?? '尚未轉錄'}
                            </p>
                          </div>
                        </div>
                      ) : (
                        <p className="mt-4 text-sm text-slate-600">尚未上傳回答</p>
                      )}
                    </li>
                  )
                })}
              </ol>
            </section>
          </div>
        ) : null}
      </section>
    </main>
  )
}
```

- [ ] **Step 6: Wire the result route to the new component**

In `frontend/src/App.tsx`, add:

```ts
import InterviewResultPage from './pages/InterviewResultPage'
```

Replace the current `if (route.name === 'result')` branch with:

```tsx
if (route.name === 'result') {
  return <InterviewResultPage interviewID={route.interviewID} />
}
```

- [ ] **Step 7: Run frontend tests**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd test --prefix frontend
```

Expected: all frontend tests pass.

## Task 3: Documentation and Progress Updates

**Files:**
- Modify: `docs/API.md`
- Modify: `docs/development-progress.md`
- Modify: `docs/DEVELOPMENT_PLAN.md`

- [ ] **Step 1: Document static audio route**

In `docs/API.md`, after `Upload Answer Audio`, add:

````md
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
````

- [ ] **Step 2: Update progress docs**

In both `docs/development-progress.md` and `docs/DEVELOPMENT_PLAN.md`, change current status:

```md
- Current step: Step 12 - 建立面試結果頁
- Status: Completed
- Last updated: 2026-05-27
```

Change the Step 12 table row status to `Completed`.

Add this Step 12 completion section after Step 11:

```md
### Step 12 - 建立面試結果頁

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
```

Set next step:

```md
Step 13 - STT mock.
```

Expected work:

```md
- Add a mock transcription flow that writes test `transcript_text`.
- Display populated transcript text on the existing result page.
- Keep real STT provider integration for a later step.
```

## Task 4: Final Verification and Commit

**Files:**
- No code changes beyond Task 1-3 files.

- [ ] **Step 1: Run backend tests**

Run:

```powershell
Push-Location backend
go test ./...
Pop-Location
```

Expected: all backend tests pass.

- [ ] **Step 2: Run frontend tests and build**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd test --prefix frontend
C:\nvm4w\nodejs\npm.cmd run build --prefix frontend
```

Expected: frontend tests and production build pass.

- [ ] **Step 3: Run Docker Compose manual verification**

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
curl.exe -i "http://localhost:8080/audio/$($createResponse.id)/$questionID.webm"
```

Expected:

```text
HTTP/1.1 200 OK
Content-Type: audio/webm

webm-bytes
```

Manual browser steps:

```text
1. Open http://localhost:5173/interviews/{created_id}/result.
2. Confirm the page heading is 面試結果.
3. Confirm job title, job description, user profile, and completed status are visible.
4. Confirm 問題 1 is visible with the generated question.
5. Confirm an audio control labeled 問題 1 回答音檔 is visible.
6. Confirm 尚未轉錄 is visible.
7. Press play on the audio control and confirm the browser attempts to load /audio/{created_id}/{question_id}.webm.
```

- [ ] **Step 4: Run diff checks**

Run:

```powershell
git diff --check
git status --short
git diff --stat
```

Expected:

- `git diff --check` exits 0.
- Changed files are limited to backend audio serving, frontend result page, tests, and docs.

- [ ] **Step 5: Commit Step 12**

Run:

```powershell
git add backend/cmd/api/main.go backend/cmd/api/main_test.go frontend/src/App.tsx frontend/src/App.test.tsx frontend/src/pages/InterviewResultPage.tsx docs/API.md docs/DEVELOPMENT_PLAN.md docs/development-progress.md
git commit -m "feat: add interview result page"
```

Expected: commit succeeds.

## Assumptions

- The Step 10 upload flow stores local WebM files under `storage/audio/{interview_id}/{question_id}.webm`.
- A plain static `/audio/...` route is sufficient for the MVP because auth and ownership are not part of the current scope.
- The full result page may show missing-answer states for robustness, even though a normally completed interview should have answers for every question.
- `transcript_text` remains nullable in Step 12; `尚未轉錄` is the correct MVP placeholder until a later STT step.

## Self-Review

- Spec coverage: Covers `/interviews/{id}/result`, interview metadata, every question, answer audio controls, and transcript placeholder display.
- Scope check: Does not implement STT, scoring, auth, export, cloud storage, or a new result endpoint.
- Placeholder scan: No placeholder markers or vague "add tests" instructions remain; every code-changing step includes concrete code or commands.
- Type consistency: Uses existing `InterviewDetail`, `Question`, and `Answer` frontend types from `frontend/src/types/interview.ts`.
- Verification: Includes backend unit tests, frontend route tests, frontend build, Docker Compose API verification, browser verification, diff checks, and commit command.
