# Step 8 TTS Question Playback Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement Step 8 from `docs/mvp-spec.md` so users can press a button on the interview session page and hear the current question read aloud by the browser.

**Architecture:** Keep this as a frontend-only MVP slice on top of the existing `/interviews/{id}/session` page. Add `SpeechSynthesis` handling directly inside `InterviewSessionPage` for now, with test coverage around cancellation, utterance creation, playback state, and question navigation cleanup. Do not add backend APIs, external TTS services, audio files, recording, upload, or result-page behavior.

**Tech Stack:** React, TypeScript, Vite, Tailwind CSS, browser `SpeechSynthesis`, Vitest, Testing Library.

---

## Scope

Implement only Step 8 from `docs/mvp-spec.md`:

```text
- 使用 SpeechSynthesis 朗讀目前題目.
- 加入播放按鈕.
```

Acceptance criteria:

```text
使用者按下播放後，瀏覽器會朗讀目前題目
```

Do not implement:

- MediaRecorder recording
- answer preview or upload
- `POST /api/interviews/{id}/start`
- backend TTS or generated audio files
- STT
- result page
- AI scoring, suggestions, or follow-up questions
- persistent playback preferences
- voice picker UI

## File Structure

- Modify `frontend/src/pages/InterviewSessionPage.tsx`: add TTS playback state, play button, cleanup on question change/unmount, and unsupported-browser fallback.
- Modify `frontend/src/App.test.tsx`: add `SpeechSynthesis` test doubles and TTS interaction tests.
- Modify `docs/development-progress.md`: mark Step 8 completed after verification.
- Modify `docs/DEVELOPMENT_PLAN.md`: mark Step 8 completed after verification.
- No backend files should change for Step 8.
- No `docs/API.md` change is needed because Step 8 does not add or change an API endpoint.
- No `.env.example` change is needed because browser TTS needs no environment variable.
- No `README.md` setup/startup change is needed because Step 8 uses existing frontend startup commands.

## Task 1: Add SpeechSynthesis Test Harness

**Files:**
- Modify: `frontend/src/App.test.tsx`

- [ ] **Step 1: Add SpeechSynthesis test helpers**

In `frontend/src/App.test.tsx`, below `mockFetchOnce`, add:

```tsx
type MockUtterance = {
  text: string
  lang: string
  onend: (() => void) | null
  onerror: (() => void) | null
}

function installSpeechSynthesisMock() {
  const speak = vi.fn()
  const cancel = vi.fn()
  const utterances: MockUtterance[] = []

  class MockSpeechSynthesisUtterance {
    text: string
    lang = ''
    onend: (() => void) | null = null
    onerror: (() => void) | null = null

    constructor(text: string) {
      this.text = text
      utterances.push(this)
    }
  }

  vi.stubGlobal('speechSynthesis', {
    speak,
    cancel,
  })
  vi.stubGlobal('SpeechSynthesisUtterance', MockSpeechSynthesisUtterance)

  return { speak, cancel, utterances }
}
```

- [ ] **Step 2: Run tests**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd test --prefix frontend -- App
```

Expected: all existing tests still pass. This helper is unused so far, but it must not break the suite.

## Task 2: Add TTS Playback Button

**Files:**
- Modify: `frontend/src/App.test.tsx`
- Modify: `frontend/src/pages/InterviewSessionPage.tsx`

- [ ] **Step 1: Write failing TTS playback test**

Add this test to `frontend/src/App.test.tsx` inside the existing `describe('App', () => { ... })` block:

```tsx
  it('speaks the current session question when the play button is clicked', async () => {
    const speech = installSpeechSynthesisMock()
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
    fireEvent.click(screen.getByRole('button', { name: '朗讀題目' }))

    expect(speech.cancel).toHaveBeenCalledTimes(1)
    expect(speech.speak).toHaveBeenCalledTimes(1)
    expect(speech.utterances[0].text).toBe('請介紹你過去與後端開發相關的經驗。')
    expect(speech.utterances[0].lang).toBe('zh-TW')
    expect(screen.getByRole('button', { name: '朗讀中' })).toBeDisabled()

    speech.utterances[0].onend?.()

    expect(screen.getByRole('button', { name: '朗讀題目' })).toBeEnabled()
  })
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd test --prefix frontend -- App
```

Expected: fail because the session page does not yet render a `朗讀題目` button.

- [ ] **Step 3: Add playback state and helpers**

Modify `frontend/src/pages/InterviewSessionPage.tsx`.

Change the import:

```tsx
import { useEffect, useMemo, useState } from 'react'
```

to:

```tsx
import { useEffect, useMemo, useState } from 'react'
```

No import change is needed if it already matches this exact line.

Inside `InterviewSessionPage`, after the existing `error` state, add:

```tsx
  const [isPlayingQuestion, setIsPlayingQuestion] = useState(false)
```

After `progressPercent`, add:

```tsx
  const canSpeakQuestion =
    typeof window !== 'undefined' &&
    'speechSynthesis' in window &&
    typeof SpeechSynthesisUtterance !== 'undefined'

  function playCurrentQuestion() {
    if (!currentQuestion || !canSpeakQuestion) {
      return
    }

    window.speechSynthesis.cancel()

    const utterance = new SpeechSynthesisUtterance(currentQuestion.text)
    utterance.lang = 'zh-TW'
    utterance.onend = () => setIsPlayingQuestion(false)
    utterance.onerror = () => setIsPlayingQuestion(false)

    setIsPlayingQuestion(true)
    window.speechSynthesis.speak(utterance)
  }
```

- [ ] **Step 4: Render the TTS button**

In `frontend/src/pages/InterviewSessionPage.tsx`, inside the `currentQuestion ? (...)` section, place this block after the question `<article>` and before the previous/next button row:

```tsx
                <div className="mt-5">
                  <button
                    type="button"
                    onClick={playCurrentQuestion}
                    disabled={!canSpeakQuestion || isPlayingQuestion}
                    className="min-h-11 rounded-md border border-teal-700 bg-white px-5 py-2 text-sm font-semibold text-teal-800 hover:bg-teal-50 disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    {isPlayingQuestion ? '朗讀中' : '朗讀題目'}
                  </button>
                  {!canSpeakQuestion ? (
                    <p className="mt-2 text-sm text-slate-600">此瀏覽器不支援題目朗讀。</p>
                  ) : null}
                </div>
```

- [ ] **Step 5: Run TTS playback test**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd test --prefix frontend -- App
```

Expected: the new TTS playback test passes.

## Task 3: Prevent Overlapping Playback on Repeated Clicks

**Files:**
- Modify: `frontend/src/App.test.tsx`
- Modify: `frontend/src/pages/InterviewSessionPage.tsx` only if the test reveals a gap.

- [ ] **Step 1: Write failing overlap-prevention test**

Add this test to `frontend/src/App.test.tsx`:

```tsx
  it('cancels existing speech before speaking the question again', async () => {
    const speech = installSpeechSynthesisMock()
    mockPathname('/interviews/interview-123/session')
    vi.stubGlobal(
      'fetch',
      mockFetchOnce({
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
    )

    render(<App />)

    expect(await screen.findByText('請介紹你過去與後端開發相關的經驗。')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: '朗讀題目' }))
    speech.utterances[0].onend?.()
    fireEvent.click(screen.getByRole('button', { name: '朗讀題目' }))

    expect(speech.cancel).toHaveBeenCalledTimes(2)
    expect(speech.speak).toHaveBeenCalledTimes(2)
  })
```

- [ ] **Step 2: Run overlap-prevention test**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd test --prefix frontend -- App
```

Expected: pass if Task 2 already calls `window.speechSynthesis.cancel()` before every `speak`. If it fails, the failure should show `cancel` was not called for each playback.

- [ ] **Step 3: Implement only missing overlap prevention**

If the test fails, ensure `playCurrentQuestion` in `frontend/src/pages/InterviewSessionPage.tsx` contains this line before constructing or speaking the utterance:

```tsx
    window.speechSynthesis.cancel()
```

- [ ] **Step 4: Run frontend tests**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd test --prefix frontend
```

Expected: all frontend tests pass.

## Task 4: Stop Playback on Question Change and Unmount

**Files:**
- Modify: `frontend/src/App.test.tsx`
- Modify: `frontend/src/pages/InterviewSessionPage.tsx`

- [ ] **Step 1: Write failing question-change cleanup test**

Add this test to `frontend/src/App.test.tsx`:

```tsx
  it('stops speech when moving to another question', async () => {
    const speech = installSpeechSynthesisMock()
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
    fireEvent.click(screen.getByRole('button', { name: '朗讀題目' }))
    fireEvent.click(screen.getByRole('button', { name: '下一題' }))

    expect(speech.cancel).toHaveBeenCalledTimes(2)
    expect(screen.getByRole('button', { name: '朗讀題目' })).toBeEnabled()
  })
```

- [ ] **Step 2: Write failing unmount cleanup test**

Add this test to `frontend/src/App.test.tsx`:

```tsx
  it('stops speech when leaving the session page', async () => {
    const speech = installSpeechSynthesisMock()
    mockPathname('/interviews/interview-123/session')
    vi.stubGlobal(
      'fetch',
      mockFetchOnce({
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
    )

    const { unmount } = render(<App />)

    expect(await screen.findByText('請介紹你過去與後端開發相關的經驗。')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: '朗讀題目' }))
    unmount()

    expect(speech.cancel).toHaveBeenCalledTimes(2)
  })
```

- [ ] **Step 3: Run cleanup tests to verify they fail**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd test --prefix frontend -- App
```

Expected: fail because question navigation and component unmount do not yet cancel active speech.

- [ ] **Step 4: Add cleanup helper**

In `frontend/src/pages/InterviewSessionPage.tsx`, after `playCurrentQuestion`, add:

```tsx
  function stopQuestionPlayback() {
    if (canSpeakQuestion) {
      window.speechSynthesis.cancel()
    }
    setIsPlayingQuestion(false)
  }
```

Update the previous button:

```tsx
                  <button
                    type="button"
                    onClick={() => {
                      stopQuestionPlayback()
                      setCurrentQuestionIndex((index) => Math.max(index - 1, 0))
                    }}
                    disabled={isFirstQuestion}
                    className="min-h-11 rounded-md border border-slate-300 bg-white px-5 py-2 text-sm font-semibold text-slate-800 hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    上一題
                  </button>
```

Update the next button:

```tsx
                  <button
                    type="button"
                    onClick={() => {
                      stopQuestionPlayback()
                      setCurrentQuestionIndex((index) => Math.min(index + 1, questions.length - 1))
                    }}
                    disabled={isLastQuestion}
                    className="min-h-11 rounded-md bg-teal-700 px-5 py-2 text-sm font-semibold text-white hover:bg-teal-800 disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    下一題
                  </button>
```

Add this effect after `stopQuestionPlayback`:

```tsx
  useEffect(() => {
    return () => {
      if (
        typeof window !== 'undefined' &&
        'speechSynthesis' in window &&
        typeof SpeechSynthesisUtterance !== 'undefined'
      ) {
        window.speechSynthesis.cancel()
      }
    }
  }, [])
```

- [ ] **Step 5: Run frontend tests**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd test --prefix frontend
```

Expected: all frontend tests pass.

## Task 5: Handle Unsupported Browser TTS

**Files:**
- Modify: `frontend/src/App.test.tsx`
- Modify: `frontend/src/pages/InterviewSessionPage.tsx` only if the test reveals a gap.

- [ ] **Step 1: Write unsupported-browser test**

Add this test to `frontend/src/App.test.tsx`:

```tsx
  it('disables question playback when speech synthesis is unavailable', async () => {
    vi.stubGlobal('speechSynthesis', undefined)
    vi.stubGlobal('SpeechSynthesisUtterance', undefined)
    mockPathname('/interviews/interview-123/session')
    vi.stubGlobal(
      'fetch',
      mockFetchOnce({
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
    )

    render(<App />)

    expect(await screen.findByText('請介紹你過去與後端開發相關的經驗。')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '朗讀題目' })).toBeDisabled()
    expect(screen.getByText('此瀏覽器不支援題目朗讀。')).toBeInTheDocument()
  })
```

- [ ] **Step 2: Run unsupported-browser test**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd test --prefix frontend -- App
```

Expected: pass if Task 2 already implemented `canSpeakQuestion` and unsupported copy. If it fails, the failure should identify missing disabled state or missing fallback text.

- [ ] **Step 3: Implement only missing unsupported-browser behavior**

If the test fails, ensure `frontend/src/pages/InterviewSessionPage.tsx` includes:

```tsx
  const canSpeakQuestion =
    typeof window !== 'undefined' &&
    'speechSynthesis' in window &&
    typeof SpeechSynthesisUtterance !== 'undefined'
```

and the playback button includes:

```tsx
disabled={!canSpeakQuestion || isPlayingQuestion}
```

and the fallback copy:

```tsx
{!canSpeakQuestion ? (
  <p className="mt-2 text-sm text-slate-600">此瀏覽器不支援題目朗讀。</p>
) : null}
```

- [ ] **Step 4: Run frontend tests**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd test --prefix frontend
```

Expected: all frontend tests pass.

## Task 6: Update Progress Documentation

**Files:**
- Modify: `docs/development-progress.md`
- Modify: `docs/DEVELOPMENT_PLAN.md`

- [ ] **Step 1: Update `docs/development-progress.md`**

Change current status:

```md
- Current step: Step 8 - TTS 朗讀題目
- Status: Completed
- Last updated: 2026-05-26
```

Change the Step 8 table row status to `Completed`.

Add this section after Step 7:

```md
### Step 8 - TTS 朗讀題目

Completed on 2026-05-26.

Implemented:

- Added a browser SpeechSynthesis playback button to the session page.
- Read the current question aloud with `zh-TW` language.
- Cancelled existing speech before starting playback to avoid overlap.
- Stopped speech when moving between questions or leaving the session page.
- Disabled playback with a clear message when the browser does not support TTS.

Verification:

- `npm test --prefix frontend` passed.
- `npm run build --prefix frontend` passed.
- Manual browser verification confirmed pressing `朗讀題目` reads the current question.
```

Set next step:

```md
Step 9 - 前端錄音.
```

Set expected work:

```md
- Use browser `MediaRecorder`.
- Support start recording, stop recording, and answer playback.
- Handle microphone permission errors.
```

- [ ] **Step 2: Update `docs/DEVELOPMENT_PLAN.md`**

Change current status:

```md
- Current step: Step 8 - TTS 朗讀題目
- Status: Completed
- Last updated: 2026-05-26
```

Change the Step 8 table row status to `Completed`.

Add this section after Step 7:

```md
## Step 8 Completion

Completed on 2026-05-26.

Implemented:

- Added a browser SpeechSynthesis playback button to the session page.
- Read the current question aloud with `zh-TW` language.
- Cancelled existing speech before starting playback to avoid overlap.
- Stopped speech when moving between questions or leaving the session page.
- Disabled playback with a clear message when the browser does not support TTS.

Verification:

- `npm test --prefix frontend` passed.
- `npm run build --prefix frontend` passed.
- Manual browser verification confirmed pressing `朗讀題目` reads the current question.
```

Set next step:

```md
Step 9 - 前端錄音.
```

## Task 7: Final Verification, Review, and Commit

**Files:**
- All Step 8 files.

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

- `frontend/src/App.test.tsx`
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
5. Click 朗讀題目.
6. Confirm the browser reads the current question aloud.
7. While speech is active or after replaying, click 下一題.
8. Confirm the visible question changes and prior playback stops.
9. Click 朗讀題目 again.
10. Confirm the browser reads the new current question aloud.
```

- [ ] **Step 6: Request code review**

Ask a subagent reviewer to inspect the completed Step 8 diff.

Provide:

- Scope: Step 8 browser TTS only.
- Plan: `docs/superpowers/plans/2026-05-26-step8-tts-question-playback.md`.
- Spec: `docs/mvp-spec.md` Step 8 and Section 9.1.
- Verification output summary.

Expected: no blocking or important issues. Fix any critical or important review findings before commit.

- [ ] **Step 7: Commit Step 8**

Run:

```powershell
git add frontend/src/App.test.tsx frontend/src/pages/InterviewSessionPage.tsx docs/DEVELOPMENT_PLAN.md docs/development-progress.md
git commit -m "feat: add question text-to-speech playback"
```

Expected: commit succeeds.

## Assumptions

- Step 8 is frontend-only because MVP TTS uses the browser built-in `SpeechSynthesis`.
- The TTS button lives on the existing session page next to the current question.
- The spoken language is `zh-TW`, matching the MVP spec example.
- Cancelling before playback and on question navigation is enough to prevent overlapping speech for the MVP.
- Browser voice selection is outside Step 8.
- No API docs update is required because no endpoint changes.

## Self-Review

- Spec coverage: Covers browser `SpeechSynthesis`, current-question playback, play button, cancellation before playback, and visible playback state.
- Scope check: Does not implement recording, upload, backend TTS, generated audio, result page, STT, scoring, or voice selection.
- Placeholder scan: No placeholder markers remain.
- Type consistency: Uses existing `InterviewSessionPage` state and current question data.
- Verification: Includes unit tests, build, backend regression tests, manual browser verification, subagent review, docs updates, and commit command.
