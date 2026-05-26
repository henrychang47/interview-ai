# Step 7 Interview Session Page Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement Step 7 from `docs/mvp-spec.md` so users can open `/interviews/{id}/session` and browse interview questions one at a time.

**Architecture:** Keep this as a frontend-only MVP slice. Reuse the existing `getInterview` API client and `InterviewDetail`/`Question` types, add a focused `InterviewSessionPage`, and extend the existing lightweight route parser in `App.tsx`. Do not implement backend start status transitions, TTS, MediaRecorder, answer upload, or result routing in this step.

**Tech Stack:** React, TypeScript, Vite, Tailwind CSS, Vitest, Testing Library.

---

## Scope

Implement only Step 7 from `docs/mvp-spec.md`:

```text
- 建立 `/interviews/{id}/session`.
- 顯示目前題目.
- 支援上一題 / 下一題或完成本題後前進.
```

Do not implement:

- `POST /api/interviews/{id}/start`
- SpeechSynthesis/TTS playback
- MediaRecorder recording
- answer preview or upload
- completed interview status updates
- `/interviews/{id}/result`
- STT
- AI scoring or suggestions

## File Structure

- Modify `frontend/src/App.tsx`: add session route parsing and render `InterviewSessionPage`.
- Create `frontend/src/pages/InterviewSessionPage.tsx`: load interview detail and manage current question index.
- Modify `frontend/src/pages/InterviewDetailPage.tsx`: add a start-session link to `/interviews/{id}/session` when questions exist.
- Modify `frontend/src/App.test.tsx`: add route and navigation tests for the session page.
- Modify `docs/development-progress.md`: mark Step 7 completed after implementation verification.
- Modify `docs/DEVELOPMENT_PLAN.md`: mark Step 7 completed after implementation verification.
- No backend files should change for Step 7.
- No `docs/API.md` change is needed because Step 7 does not add or change an API endpoint.
- No `.env.example` change is needed because Step 7 adds no environment variables.
- No `README.md` setup/startup change is needed because Step 7 uses existing startup commands.

## Task 1: Add Session Route Rendering

**Files:**
- Modify: `frontend/src/App.test.tsx`
- Modify: `frontend/src/App.tsx`
- Create: `frontend/src/pages/InterviewSessionPage.tsx`

- [ ] **Step 1: Write failing route test**

Add this test to `frontend/src/App.test.tsx` inside the existing `describe('App', () => { ... })` block:

```tsx
  it('loads the interview session page at /interviews/:id/session', async () => {
    mockPathname('/interviews/interview-123/session')
    vi.stubGlobal(
      'fetch',
      mockFetchOnce({
        id: 'interview-123',
        job_title: '後端工程師',
        job_description: '需要熟悉 Go、PostgreSQL、REST API',
        user_profile: '有 Java 和 Go 學習經驗',
        question_count: 2,
        status: 'questions_ready',
        questions: [
          { id: 'question-1', order: 1, text: '請介紹你過去與後端開發相關的經驗。' },
          { id: 'question-2', order: 2, text: '你如何設計一個 REST API？' },
        ],
        answers: [],
      }),
    )

    render(<App />)

    expect(await screen.findByRole('heading', { name: '模擬面試進行中' })).toBeInTheDocument()
    expect(screen.getByText('後端工程師')).toBeInTheDocument()
    expect(screen.getByText('第 1 題 / 共 2 題')).toBeInTheDocument()
    expect(screen.getByText('請介紹你過去與後端開發相關的經驗。')).toBeInTheDocument()
  })
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd test --prefix frontend -- App
```

Expected: fail because `/interviews/:id/session` currently falls through to the home route, so the heading `模擬面試進行中` is missing.

- [ ] **Step 3: Create minimal session page**

Create `frontend/src/pages/InterviewSessionPage.tsx`:

```tsx
import { useEffect, useMemo, useState } from 'react'

import { getInterview } from '../api/interviews'
import type { InterviewDetail } from '../types/interview'

type InterviewSessionPageProps = {
  interviewID: string
}

export default function InterviewSessionPage({ interviewID }: InterviewSessionPageProps) {
  const [interview, setInterview] = useState<InterviewDetail | null>(null)
  const [currentQuestionIndex, setCurrentQuestionIndex] = useState(0)
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
          setCurrentQuestionIndex(0)
        }
      } catch (error) {
        if (isMounted) {
          setError(error instanceof Error ? error.message : '載入面試失敗')
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

  const questions = interview?.questions ?? []
  const currentQuestion = questions[currentQuestionIndex]
  const isFirstQuestion = currentQuestionIndex === 0
  const isLastQuestion = currentQuestionIndex === questions.length - 1

  const progressPercent = useMemo(() => {
    if (questions.length === 0) {
      return 0
    }
    return ((currentQuestionIndex + 1) / questions.length) * 100
  }, [currentQuestionIndex, questions.length])

  return (
    <main className="min-h-screen bg-slate-50 text-slate-950">
      <section className="mx-auto w-full max-w-4xl px-6 py-10">
        <a
          href={`/interviews/${interviewID}`}
          className="text-sm font-medium text-teal-700 hover:text-teal-800"
        >
          返回面試詳情
        </a>

        {isLoading ? <p className="mt-8 text-slate-600">載入面試中...</p> : null}

        {error ? (
          <div className="mt-8 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
            {error}
          </div>
        ) : null}

        {interview ? (
          <div className="mt-8">
            <div className="border-b border-slate-200 pb-6">
              <p className="text-sm font-semibold uppercase tracking-wide text-teal-700">
                Interview Session
              </p>
              <h1 className="mt-3 text-3xl font-bold leading-tight">模擬面試進行中</h1>
              <p className="mt-3 text-lg font-medium text-slate-800">{interview.job_title}</p>
            </div>

            {currentQuestion ? (
              <section className="mt-8">
                <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                  <p className="text-sm font-semibold text-teal-700">
                    第 {currentQuestionIndex + 1} 題 / 共 {questions.length} 題
                  </p>
                  <p className="text-sm text-slate-600">問題 {currentQuestion.order}</p>
                </div>

                <div className="mt-4 h-2 overflow-hidden rounded-full bg-slate-200">
                  <div
                    className="h-full rounded-full bg-teal-700"
                    style={{ width: `${progressPercent}%` }}
                  />
                </div>

                <article className="mt-6 rounded-lg border border-slate-200 bg-white p-6 shadow-sm">
                  <p className="text-xl font-semibold leading-8 text-slate-950">
                    {currentQuestion.text}
                  </p>
                </article>

                <div className="mt-6 flex flex-col gap-3 sm:flex-row sm:justify-between">
                  <button
                    type="button"
                    onClick={() => setCurrentQuestionIndex((index) => Math.max(index - 1, 0))}
                    disabled={isFirstQuestion}
                    className="min-h-11 rounded-md border border-slate-300 bg-white px-5 py-2 text-sm font-semibold text-slate-800 hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    上一題
                  </button>
                  <button
                    type="button"
                    onClick={() =>
                      setCurrentQuestionIndex((index) => Math.min(index + 1, questions.length - 1))
                    }
                    disabled={isLastQuestion}
                    className="min-h-11 rounded-md bg-teal-700 px-5 py-2 text-sm font-semibold text-white hover:bg-teal-800 disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    下一題
                  </button>
                </div>
              </section>
            ) : (
              <div className="mt-8 rounded-md border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800">
                這場面試目前沒有題目。
              </div>
            )}
          </div>
        ) : null}
      </section>
    </main>
  )
}
```

- [ ] **Step 4: Wire session route in App**

Modify `frontend/src/App.tsx`.

Add the import:

```tsx
import InterviewSessionPage from './pages/InterviewSessionPage'
```

Update `getRoute` so the session route is checked before the detail route:

```tsx
function getRoute(pathname: string) {
  if (pathname === '/interviews/new') {
    return { name: 'new' as const }
  }

  const sessionMatch = pathname.match(/^\/interviews\/([^/]+)\/session$/)
  if (sessionMatch) {
    return { name: 'session' as const, interviewID: decodeURIComponent(sessionMatch[1]) }
  }

  const detailMatch = pathname.match(/^\/interviews\/([^/]+)$/)
  if (detailMatch) {
    return { name: 'detail' as const, interviewID: decodeURIComponent(detailMatch[1]) }
  }

  return { name: 'home' as const }
}
```

Add the render branch before the detail branch:

```tsx
  if (route.name === 'session') {
    return <InterviewSessionPage interviewID={route.interviewID} />
  }
```

- [ ] **Step 5: Run route test**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd test --prefix frontend -- App
```

Expected: the new session route test passes.

## Task 2: Add Question Navigation Behavior

**Files:**
- Modify: `frontend/src/App.test.tsx`
- Modify: `frontend/src/pages/InterviewSessionPage.tsx` only if the failing test reveals a gap.

- [ ] **Step 1: Write failing navigation test**

Add this test to `frontend/src/App.test.tsx`:

```tsx
  it('moves between session questions with previous and next buttons', async () => {
    mockPathname('/interviews/interview-123/session')
    vi.stubGlobal(
      'fetch',
      mockFetchOnce({
        id: 'interview-123',
        job_title: '後端工程師',
        job_description: '需要熟悉 Go、PostgreSQL、REST API',
        user_profile: '有 Java 和 Go 學習經驗',
        question_count: 2,
        status: 'questions_ready',
        questions: [
          { id: 'question-1', order: 1, text: '請介紹你過去與後端開發相關的經驗。' },
          { id: 'question-2', order: 2, text: '你如何設計一個 REST API？' },
        ],
        answers: [],
      }),
    )

    render(<App />)

    expect(await screen.findByText('請介紹你過去與後端開發相關的經驗。')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '上一題' })).toBeDisabled()
    expect(screen.getByRole('button', { name: '下一題' })).toBeEnabled()

    fireEvent.click(screen.getByRole('button', { name: '下一題' }))

    expect(screen.getByText('第 2 題 / 共 2 題')).toBeInTheDocument()
    expect(screen.getByText('你如何設計一個 REST API？')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '上一題' })).toBeEnabled()
    expect(screen.getByRole('button', { name: '下一題' })).toBeDisabled()

    fireEvent.click(screen.getByRole('button', { name: '上一題' }))

    expect(screen.getByText('第 1 題 / 共 2 題')).toBeInTheDocument()
    expect(screen.getByText('請介紹你過去與後端開發相關的經驗。')).toBeInTheDocument()
  })
```

- [ ] **Step 2: Run navigation test to verify it fails or passes for the right reason**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd test --prefix frontend -- App
```

Expected: if Task 1 already implemented navigation exactly as shown, this test passes. If it fails, the failure should identify a missing button state or question-index update.

- [ ] **Step 3: Implement only missing navigation behavior**

If the test fails, update `frontend/src/pages/InterviewSessionPage.tsx` only enough to satisfy the assertions:

```tsx
const isFirstQuestion = currentQuestionIndex === 0
const isLastQuestion = currentQuestionIndex === questions.length - 1
```

Use clamped state updates:

```tsx
onClick={() => setCurrentQuestionIndex((index) => Math.max(index - 1, 0))}
```

```tsx
onClick={() =>
  setCurrentQuestionIndex((index) => Math.min(index + 1, questions.length - 1))
}
```

Disable boundary buttons:

```tsx
disabled={isFirstQuestion}
```

```tsx
disabled={isLastQuestion}
```

- [ ] **Step 4: Run frontend tests**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd test --prefix frontend
```

Expected: all frontend tests pass.

## Task 3: Add Entry Link From Interview Detail

**Files:**
- Modify: `frontend/src/App.test.tsx`
- Modify: `frontend/src/pages/InterviewDetailPage.tsx`

- [ ] **Step 1: Write failing detail-page entry test**

Update the existing `loads interview details and displays questions at /interviews/:id` test in `frontend/src/App.test.tsx` by adding this assertion at the end:

```tsx
    expect(screen.getByRole('link', { name: '開始模擬面試' })).toHaveAttribute(
      'href',
      '/interviews/interview-123/session',
    )
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd test --prefix frontend -- App
```

Expected: fail because the detail page does not yet render a `開始模擬面試` link.

- [ ] **Step 3: Add session link to detail page**

In `frontend/src/pages/InterviewDetailPage.tsx`, add this link after the questions list section and inside the `{interview ? (...) : null}` block:

```tsx
            {interview.questions.length > 0 ? (
              <div className="mt-8">
                <a
                  href={`/interviews/${interview.id}/session`}
                  className="inline-flex min-h-11 items-center rounded-md bg-teal-700 px-5 py-2 text-sm font-semibold text-white hover:bg-teal-800"
                >
                  開始模擬面試
                </a>
              </div>
            ) : null}
```

Place it after:

```tsx
            <section className="mt-8">
              <h2 className="text-xl font-semibold text-slate-900">面試問題</h2>
              <ol className="mt-4 space-y-3">
                {interview.questions.map((question) => (
                  <li
                    key={question.id}
                    className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm"
                  >
                    <p className="text-sm font-semibold text-teal-700">問題 {question.order}</p>
                    <p className="mt-2 text-slate-900">{question.text}</p>
                  </li>
                ))}
              </ol>
            </section>
```

- [ ] **Step 4: Run frontend tests**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd test --prefix frontend
```

Expected: all frontend tests pass.

## Task 4: Handle Session Loading and Empty/Error States

**Files:**
- Modify: `frontend/src/App.test.tsx`
- Modify: `frontend/src/pages/InterviewSessionPage.tsx` only if tests reveal gaps.

- [ ] **Step 1: Write failing empty-state test**

Add this test to `frontend/src/App.test.tsx`:

```tsx
  it('shows an empty state when a session has no questions', async () => {
    mockPathname('/interviews/interview-empty/session')
    vi.stubGlobal(
      'fetch',
      mockFetchOnce({
        id: 'interview-empty',
        job_title: '後端工程師',
        job_description: '需要熟悉 Go',
        user_profile: '有 Go 學習經驗',
        question_count: 0,
        status: 'questions_ready',
        questions: [],
        answers: [],
      }),
    )

    render(<App />)

    expect(await screen.findByText('這場面試目前沒有題目。')).toBeInTheDocument()
  })
```

- [ ] **Step 2: Write failing error-state test**

Add this test to `frontend/src/App.test.tsx`:

```tsx
  it('shows an API error when loading a session fails', async () => {
    mockPathname('/interviews/missing/session')
    vi.stubGlobal('fetch', mockFetchOnce({ error: 'interview not found' }, { status: 404 }))

    render(<App />)

    expect(await screen.findByText('interview not found')).toBeInTheDocument()
  })
```

- [ ] **Step 3: Run tests to verify failures or confirm existing coverage**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd test --prefix frontend -- App
```

Expected: if Task 1 already included the shown empty and error states, these tests pass. If they fail, the failure should point to missing text or missing API error handling.

- [ ] **Step 4: Implement only missing state handling**

If the tests fail, ensure `frontend/src/pages/InterviewSessionPage.tsx` includes:

```tsx
{isLoading ? <p className="mt-8 text-slate-600">載入面試中...</p> : null}
```

```tsx
{error ? (
  <div className="mt-8 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
    {error}
  </div>
) : null}
```

```tsx
{currentQuestion ? (
  /* question UI */
) : (
  <div className="mt-8 rounded-md border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800">
    這場面試目前沒有題目。
  </div>
)}
```

- [ ] **Step 5: Run frontend tests**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd test --prefix frontend
```

Expected: all frontend tests pass.

## Task 5: Update Progress Documentation

**Files:**
- Modify: `docs/development-progress.md`
- Modify: `docs/DEVELOPMENT_PLAN.md`

- [ ] **Step 1: Update `docs/development-progress.md`**

Change current status:

```md
- Current step: Step 7 - 模擬面試頁
- Status: Completed
- Last updated: 2026-05-26
```

Change the Step 7 table row status to `Completed`.

Add this section after Step 6:

```md
### Step 7 - 模擬面試頁

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
```

Set next step:

```md
Step 8 - TTS 朗讀題目.
```

Set expected work:

```md
- Use browser `SpeechSynthesis` to read the current question.
- Add a play button and playback state.
- Prevent overlapping playback.
```

- [ ] **Step 2: Update `docs/DEVELOPMENT_PLAN.md`**

Change current status:

```md
- Current step: Step 7 - 模擬面試頁
- Status: Completed
- Last updated: 2026-05-26
```

Change the Step 7 table row status to `Completed`.

Add this section after Step 6:

```md
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
```

Set next step:

```md
Step 8 - TTS 朗讀題目.
```

## Task 6: Final Verification and Commit

**Files:**
- All Step 7 files.

- [ ] **Step 1: Run frontend tests**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd test --prefix frontend
```

Expected: all tests pass.

- [ ] **Step 2: Run frontend build**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd run build --prefix frontend
```

Expected: TypeScript and Vite build pass.

- [ ] **Step 3: Run backend tests as regression check**

Run:

```powershell
$env:GOCACHE='D:\projects\interview-ai\backend\.cache\go-build'
$env:GOMODCACHE='D:\projects\interview-ai\backend\.cache\gomod'
Push-Location backend
Remove-Item Env:\DATABASE_URL -ErrorAction SilentlyContinue
go test ./...
Pop-Location
```

Expected: all backend tests pass.

- [ ] **Step 4: Run diff checks**

Run:

```powershell
git diff --check
git status --short
git diff --stat
```

Expected changed files:

- `frontend/src/App.tsx`
- `frontend/src/App.test.tsx`
- `frontend/src/pages/InterviewDetailPage.tsx`
- `frontend/src/pages/InterviewSessionPage.tsx`
- `docs/DEVELOPMENT_PLAN.md`
- `docs/development-progress.md`

- [ ] **Step 5: Manual browser verification**

Start services if needed:

```powershell
docker compose up -d postgres
docker compose run --rm migrate
docker compose up --build -d backend frontend
```

Create or reuse an interview, then verify:

```text
1. Open http://localhost:5173/interviews/new.
2. Create an interview with 2 or 3 questions.
3. On /interviews/{id}, click 開始模擬面試.
4. Confirm /interviews/{id}/session loads.
5. Confirm the page shows 第 1 題 / 共 N 題 and the first question.
6. Click 下一題 and confirm the displayed question changes.
7. Click 上一題 and confirm the displayed question changes back.
8. Confirm 上一題 is disabled on the first question.
9. Confirm 下一題 is disabled on the last question.
```

- [ ] **Step 6: Commit Step 7**

Run:

```powershell
git add frontend/src/App.tsx frontend/src/App.test.tsx frontend/src/pages/InterviewDetailPage.tsx frontend/src/pages/InterviewSessionPage.tsx docs/DEVELOPMENT_PLAN.md docs/development-progress.md
git commit -m "feat: add interview session page"
```

Expected: commit succeeds.

## Assumptions

- Step 7 is frontend-only because the Step 7 acceptance criteria only require browsing questions.
- The detail page can link directly to `/interviews/{id}/session`; it must not call `POST /api/interviews/{id}/start` until the backend start API is explicitly implemented in a future step.
- The session page reloads interview data with `GET /api/interviews/{id}` so refresh recovery works with existing API behavior.
- Current question position is local React state only; persisting the active question is outside MVP Step 7.
- No API docs update is required because no API endpoint changes.

## Self-Review

- Spec coverage: Covers `/interviews/{id}/session`, current question display, and previous/next question navigation from Step 7.
- Scope check: Does not implement start API, TTS, recording, upload, completion status, result page, STT, or scoring.
- Placeholder scan: No placeholder markers remain.
- Type consistency: Uses existing `InterviewDetail`, `Question`, `getInterview`, and route shape.
- Verification: Includes unit tests, build, backend regression tests, diff checks, manual browser verification, docs updates, and commit command.
