# Step 9 Frontend Recording Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement Step 9 from `docs/mvp-spec.md` so users can record an answer in the interview session page, stop recording, and play back the recorded answer locally in the browser.

**Architecture:** Keep this as a frontend-only MVP slice on top of the existing `/interviews/{id}/session` page. Add browser `MediaRecorder` handling directly inside `InterviewSessionPage` for now, with focused state for recording, local preview Blob URL, media stream cleanup, and permission/unsupported-browser errors. Do not add upload APIs, answer persistence, status changes, result-page behavior, transcription, or completion flow in this step.

**Tech Stack:** React, TypeScript, Vite, Tailwind CSS, browser `MediaRecorder`, browser `navigator.mediaDevices.getUserMedia`, Vitest, Testing Library.

---

## Scope

Implement only Step 9 from `docs/mvp-spec.md`:

```text
- 使用 MediaRecorder.
- 支援開始錄音、停止錄音、回放錄音.
```

Acceptance criteria:

```text
使用者可錄製回答，並在前端回放錄音
```

Do not implement:

- Answer audio upload
- `POST /api/interviews/{interview_id}/questions/{question_id}/answer`
- Backend local audio storage
- Interview completion status
- Result page
- STT or transcript generation
- Persistent recordings across page refreshes
- Per-question saved draft storage
- Recording duration countdown or timer UI beyond button state

## File Structure

- Modify `frontend/src/pages/InterviewSessionPage.tsx`: add MediaRecorder state, start/stop controls, local audio preview, cleanup on question change/unmount, and unsupported/permission error messages.
- Modify `frontend/src/App.test.tsx`: add MediaRecorder/getUserMedia/Object URL test doubles and recording interaction tests.
- Modify `docs/development-progress.md`: mark Step 9 completed after verification.
- Modify `docs/DEVELOPMENT_PLAN.md`: mark Step 9 completed after verification.
- No backend files should change for Step 9.
- No `docs/API.md` change is needed because Step 9 does not add or change an API endpoint.
- No `.env.example` change is needed because browser recording needs no environment variable.
- No `README.md` setup/startup change is needed because Step 9 uses existing frontend startup commands.

## Task 1: Add MediaRecorder Test Harness

**Files:**
- Modify: `frontend/src/App.test.tsx`

- [ ] **Step 1: Add MediaRecorder test helpers**

In `frontend/src/App.test.tsx`, below `installSpeechSynthesisMock`, add:

```tsx
type MockMediaRecorderInstance = {
  stream: MediaStream
  state: 'inactive' | 'recording'
  ondataavailable: ((event: BlobEvent) => void) | null
  onstop: (() => void) | null
  start: ReturnType<typeof vi.fn>
  stop: ReturnType<typeof vi.fn>
}

function installMediaRecorderMock() {
  const trackStop = vi.fn()
  const stream = {
    getTracks: () => [{ stop: trackStop }],
  } as unknown as MediaStream
  const getUserMedia = vi.fn().mockResolvedValue(stream)
  const recorders: MockMediaRecorderInstance[] = []

  class MockMediaRecorder {
    stream: MediaStream
    state: 'inactive' | 'recording' = 'inactive'
    ondataavailable: ((event: BlobEvent) => void) | null = null
    onstop: (() => void) | null = null

    constructor(stream: MediaStream) {
      this.stream = stream
      recorders.push(this as unknown as MockMediaRecorderInstance)
    }

    start = vi.fn(() => {
      this.state = 'recording'
    })

    stop = vi.fn(() => {
      this.state = 'inactive'
      this.ondataavailable?.({ data: new Blob(['recorded-answer'], { type: 'audio/webm' }) } as BlobEvent)
      this.onstop?.()
    })

    static isTypeSupported(type: string) {
      return type === 'audio/webm'
    }
  }

  Object.defineProperty(navigator, 'mediaDevices', {
    configurable: true,
    value: { getUserMedia },
  })
  vi.stubGlobal('MediaRecorder', MockMediaRecorder)

  return { getUserMedia, recorders, stream, trackStop }
}
```

- [ ] **Step 2: Add Object URL test helpers**

Still below `installSpeechSynthesisMock`, add:

```tsx
function installObjectURLMock() {
  const createObjectURL = vi.fn(() => 'blob:recorded-answer')
  const revokeObjectURL = vi.fn()

  Object.defineProperty(URL, 'createObjectURL', {
    configurable: true,
    value: createObjectURL,
  })
  Object.defineProperty(URL, 'revokeObjectURL', {
    configurable: true,
    value: revokeObjectURL,
  })

  return { createObjectURL, revokeObjectURL }
}
```

- [ ] **Step 3: Run tests**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd test --prefix frontend -- App
```

Expected: all existing tests still pass. These helpers are unused so far, but they must not break the suite.

## Task 2: Start Recording

**Files:**
- Modify: `frontend/src/App.test.tsx`
- Modify: `frontend/src/pages/InterviewSessionPage.tsx`

- [ ] **Step 1: Write failing start-recording test**

Add this test to `frontend/src/App.test.tsx` inside the existing `describe('App', () => { ... })` block:

```tsx
  it('starts recording an answer for the current session question', async () => {
    const media = installMediaRecorderMock()
    installObjectURLMock()
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
    fireEvent.click(screen.getByRole('button', { name: '開始錄音' }))

    await waitFor(() => {
      expect(media.getUserMedia).toHaveBeenCalledWith({ audio: true })
    })
    expect(media.recorders).toHaveLength(1)
    expect(media.recorders[0].start).toHaveBeenCalledTimes(1)
    expect(screen.getByRole('button', { name: '錄音中' })).toBeDisabled()
    expect(screen.getByRole('button', { name: '停止錄音' })).toBeEnabled()
  })
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd test --prefix frontend -- App
```

Expected: fail because the session page does not yet render a `開始錄音` button.

- [ ] **Step 3: Add recording refs and state**

Modify `frontend/src/pages/InterviewSessionPage.tsx`.

Change the import:

```tsx
import { useEffect, useMemo, useState } from 'react'
```

to:

```tsx
import { useEffect, useMemo, useRef, useState } from 'react'
```

Inside `InterviewSessionPage`, after the existing `isPlayingQuestion` state, add:

```tsx
  const [isRecordingAnswer, setIsRecordingAnswer] = useState(false)
  const [recordedAnswerURL, setRecordedAnswerURL] = useState<string | null>(null)
  const [recordingError, setRecordingError] = useState<string | null>(null)
  const mediaRecorderRef = useRef<MediaRecorder | null>(null)
  const mediaStreamRef = useRef<MediaStream | null>(null)
  const recordedChunksRef = useRef<Blob[]>([])
```

After `canSpeakQuestion`, add:

```tsx
  const canRecordAnswer =
    typeof navigator !== 'undefined' &&
    Boolean(navigator.mediaDevices?.getUserMedia) &&
    typeof MediaRecorder !== 'undefined'
```

After `stopQuestionPlayback`, add:

```tsx
  function revokeRecordedAnswerURL() {
    setRecordedAnswerURL((currentURL) => {
      if (currentURL) {
        URL.revokeObjectURL(currentURL)
      }
      return null
    })
  }

  function stopMediaStream() {
    mediaStreamRef.current?.getTracks().forEach((track) => track.stop())
    mediaStreamRef.current = null
  }

  async function startAnswerRecording() {
    if (!canRecordAnswer || isRecordingAnswer) {
      return
    }

    stopQuestionPlayback()
    revokeRecordedAnswerURL()
    setRecordingError(null)
    recordedChunksRef.current = []

    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
      mediaStreamRef.current = stream

      const recorderOptions =
        typeof MediaRecorder.isTypeSupported === 'function' &&
        MediaRecorder.isTypeSupported('audio/webm')
          ? { mimeType: 'audio/webm' }
          : undefined
      const recorder = new MediaRecorder(stream, recorderOptions)

      recorder.ondataavailable = (event) => {
        if (event.data.size > 0) {
          recordedChunksRef.current.push(event.data)
        }
      }
      recorder.onstop = () => {
        const recordedBlob = new Blob(recordedChunksRef.current, { type: 'audio/webm' })
        setRecordedAnswerURL(URL.createObjectURL(recordedBlob))
        setIsRecordingAnswer(false)
        stopMediaStream()
      }

      mediaRecorderRef.current = recorder
      recorder.start()
      setIsRecordingAnswer(true)
    } catch (error) {
      setIsRecordingAnswer(false)
      stopMediaStream()
      setRecordingError(error instanceof Error ? error.message : '無法開始錄音')
    }
  }
```

- [ ] **Step 4: Render initial recording controls**

In `frontend/src/pages/InterviewSessionPage.tsx`, inside the `currentQuestion ? (...)` section, place this block after the existing TTS button block and before the previous/next button row:

```tsx
                <div className="mt-5 rounded-md border border-slate-200 bg-white p-5">
                  <div className="flex flex-col gap-3 sm:flex-row">
                    <button
                      type="button"
                      onClick={startAnswerRecording}
                      disabled={!canRecordAnswer || isRecordingAnswer}
                      className="min-h-11 rounded-md bg-teal-700 px-5 py-2 text-sm font-semibold text-white hover:bg-teal-800 disabled:cursor-not-allowed disabled:opacity-50"
                    >
                      {isRecordingAnswer ? '錄音中' : '開始錄音'}
                    </button>
                    <button
                      type="button"
                      disabled
                      className="min-h-11 rounded-md border border-slate-300 bg-white px-5 py-2 text-sm font-semibold text-slate-800 disabled:cursor-not-allowed disabled:opacity-50"
                    >
                      停止錄音
                    </button>
                  </div>
                  {!canRecordAnswer ? (
                    <p className="mt-3 text-sm text-slate-600">此瀏覽器不支援錄音。</p>
                  ) : null}
                  {recordingError ? (
                    <p className="mt-3 text-sm text-red-700">{recordingError}</p>
                  ) : null}
                </div>
```

- [ ] **Step 5: Run start-recording test**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd test --prefix frontend -- App
```

Expected: the start-recording test passes.

## Task 3: Stop Recording and Create Local Playback URL

**Files:**
- Modify: `frontend/src/App.test.tsx`
- Modify: `frontend/src/pages/InterviewSessionPage.tsx`

- [ ] **Step 1: Write failing stop-recording and preview test**

Add this test to `frontend/src/App.test.tsx`:

```tsx
  it('stops recording and shows an audio preview for the recorded answer', async () => {
    const media = installMediaRecorderMock()
    const objectURL = installObjectURLMock()
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
    fireEvent.click(screen.getByRole('button', { name: '開始錄音' }))

    await waitFor(() => {
      expect(media.recorders).toHaveLength(1)
    })
    fireEvent.click(screen.getByRole('button', { name: '停止錄音' }))

    expect(media.recorders[0].stop).toHaveBeenCalledTimes(1)
    expect(media.trackStop).toHaveBeenCalledTimes(1)
    expect(objectURL.createObjectURL).toHaveBeenCalledTimes(1)
    expect(screen.getByLabelText('回答錄音預覽')).toHaveAttribute('src', 'blob:recorded-answer')
    expect(screen.getByRole('button', { name: '開始錄音' })).toBeEnabled()
  })
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd test --prefix frontend -- App
```

Expected: fail because the stop button is always disabled and no preview audio is rendered.

- [ ] **Step 3: Add stop handler**

In `frontend/src/pages/InterviewSessionPage.tsx`, after `startAnswerRecording`, add:

```tsx
  function stopAnswerRecording() {
    const recorder = mediaRecorderRef.current
    if (!recorder || recorder.state === 'inactive') {
      return
    }

    recorder.stop()
  }
```

- [ ] **Step 4: Enable stop button and render audio preview**

In the recording control block in `frontend/src/pages/InterviewSessionPage.tsx`, replace the stop button with:

```tsx
                    <button
                      type="button"
                      onClick={stopAnswerRecording}
                      disabled={!isRecordingAnswer}
                      className="min-h-11 rounded-md border border-slate-300 bg-white px-5 py-2 text-sm font-semibold text-slate-800 hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-50"
                    >
                      停止錄音
                    </button>
```

Below the `recordingError` paragraph, add:

```tsx
                  {recordedAnswerURL ? (
                    <div className="mt-4">
                      <p className="mb-2 text-sm font-medium text-slate-700">回答錄音預覽</p>
                      <audio aria-label="回答錄音預覽" controls src={recordedAnswerURL} />
                    </div>
                  ) : null}
```

- [ ] **Step 5: Run frontend tests**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd test --prefix frontend -- App
```

Expected: the stop-recording and preview test passes.

## Task 4: Handle Permission Errors and Unsupported Browser Recording

**Files:**
- Modify: `frontend/src/App.test.tsx`
- Modify: `frontend/src/pages/InterviewSessionPage.tsx`

- [ ] **Step 1: Write permission-error test**

Add this test to `frontend/src/App.test.tsx`:

```tsx
  it('shows an error when microphone permission is denied', async () => {
    installObjectURLMock()
    Object.defineProperty(navigator, 'mediaDevices', {
      configurable: true,
      value: {
        getUserMedia: vi.fn().mockRejectedValue(new Error('Permission denied')),
      },
    })
    vi.stubGlobal('MediaRecorder', class MockMediaRecorder {})
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
    fireEvent.click(screen.getByRole('button', { name: '開始錄音' }))

    expect(await screen.findByText('Permission denied')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '開始錄音' })).toBeEnabled()
  })
```

- [ ] **Step 2: Write unsupported-browser recording test**

Add this test to `frontend/src/App.test.tsx`:

```tsx
  it('disables answer recording when MediaRecorder is unavailable', async () => {
    installObjectURLMock()
    Object.defineProperty(navigator, 'mediaDevices', {
      configurable: true,
      value: undefined,
    })
    vi.stubGlobal('MediaRecorder', undefined)
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
    expect(screen.getByRole('button', { name: '開始錄音' })).toBeDisabled()
    expect(screen.getByRole('button', { name: '停止錄音' })).toBeDisabled()
    expect(screen.getByText('此瀏覽器不支援錄音。')).toBeInTheDocument()
  })
```

- [ ] **Step 3: Run tests**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd test --prefix frontend -- App
```

Expected: both tests pass if Tasks 2 and 3 already implemented the guard and error message. If either fails, implement only the missing disabled state or error rendering described in the failure.

## Task 5: Cleanup Recording on Question Change and Unmount

**Files:**
- Modify: `frontend/src/App.test.tsx`
- Modify: `frontend/src/pages/InterviewSessionPage.tsx`

- [ ] **Step 1: Write question-change cleanup test**

Add this test to `frontend/src/App.test.tsx`:

```tsx
  it('stops active recording and clears preview when moving to another question', async () => {
    const media = installMediaRecorderMock()
    const objectURL = installObjectURLMock()
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
    fireEvent.click(screen.getByRole('button', { name: '開始錄音' }))

    await waitFor(() => {
      expect(media.recorders).toHaveLength(1)
    })
    fireEvent.click(screen.getByRole('button', { name: '停止錄音' }))
    expect(screen.getByLabelText('回答錄音預覽')).toHaveAttribute('src', 'blob:recorded-answer')

    fireEvent.click(screen.getByRole('button', { name: '下一題' }))

    expect(objectURL.revokeObjectURL).toHaveBeenCalledWith('blob:recorded-answer')
    expect(screen.queryByLabelText('回答錄音預覽')).not.toBeInTheDocument()
  })
```

- [ ] **Step 2: Write unmount cleanup test**

Add this test to `frontend/src/App.test.tsx`:

```tsx
  it('stops media tracks when leaving the session page during recording', async () => {
    const media = installMediaRecorderMock()
    installObjectURLMock()
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
    fireEvent.click(screen.getByRole('button', { name: '開始錄音' }))

    await waitFor(() => {
      expect(media.recorders).toHaveLength(1)
    })
    unmount()

    expect(media.trackStop).toHaveBeenCalledTimes(1)
  })
```

- [ ] **Step 3: Run cleanup tests to verify they fail**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd test --prefix frontend -- App
```

Expected: fail because question navigation and unmount do not yet clean up recording state.

- [ ] **Step 4: Add recording cleanup helper**

In `frontend/src/pages/InterviewSessionPage.tsx`, after `stopAnswerRecording`, add:

```tsx
  function resetAnswerRecording() {
    const recorder = mediaRecorderRef.current
    if (recorder && recorder.state !== 'inactive') {
      recorder.stop()
    } else {
      stopMediaStream()
    }

    mediaRecorderRef.current = null
    recordedChunksRef.current = []
    setIsRecordingAnswer(false)
    setRecordingError(null)
    revokeRecordedAnswerURL()
  }
```

- [ ] **Step 5: Use cleanup helper on question navigation**

Update the previous button handler:

```tsx
                    onClick={() => {
                      stopQuestionPlayback()
                      resetAnswerRecording()
                      setCurrentQuestionIndex((index) => Math.max(index - 1, 0))
                    }}
```

Update the next button handler:

```tsx
                    onClick={() => {
                      stopQuestionPlayback()
                      resetAnswerRecording()
                      setCurrentQuestionIndex((index) => Math.min(index + 1, questions.length - 1))
                    }}
```

- [ ] **Step 6: Add unmount cleanup**

Add this effect after `resetAnswerRecording`:

```tsx
  useEffect(() => {
    return () => {
      const recorder = mediaRecorderRef.current
      if (recorder && recorder.state !== 'inactive') {
        recorder.stop()
      } else {
        mediaStreamRef.current?.getTracks().forEach((track) => track.stop())
      }
    }
  }, [])
```

- [ ] **Step 7: Run frontend tests**

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
- Current step: Step 9 - 前端錄音
- Status: Completed
- Last updated: 2026-05-27
```

Change the Step 9 table row status to `Completed`.

Add this section after Step 8:

```md
### Step 9 - 前端錄音

Completed on 2026-05-27.

Implemented:

- Added browser MediaRecorder controls to the session page.
- Requested microphone permission with `navigator.mediaDevices.getUserMedia`.
- Supported start recording and stop recording for the current question.
- Created a local answer audio preview after recording stops.
- Stopped microphone tracks and cleared local previews when moving between questions or leaving the session page.
- Disabled recording with a clear message when the browser does not support MediaRecorder.
- Displayed microphone permission errors.

Verification:

- `npm test --prefix frontend` passed.
- `npm run build --prefix frontend` passed.
- Manual browser verification confirmed users can record an answer and play it back locally.
```

Set next step:

```md
Step 10 - 回答音檔上傳 API.
```

Set expected work:

```md
- Implement multipart answer upload API.
- Save uploaded audio to local storage.
- Create or update answer records in the database.
```

- [ ] **Step 2: Update `docs/DEVELOPMENT_PLAN.md`**

Change current status:

```md
- Current step: Step 9 - 前端錄音
- Status: Completed
- Last updated: 2026-05-27
```

Change the Step 9 table row status to `Completed`.

Add this section after Step 8:

```md
## Step 9 Completion

Completed on 2026-05-27.

Implemented:

- Added browser MediaRecorder controls to the session page.
- Requested microphone permission with `navigator.mediaDevices.getUserMedia`.
- Supported start recording and stop recording for the current question.
- Created a local answer audio preview after recording stops.
- Stopped microphone tracks and cleared local previews when moving between questions or leaving the session page.
- Disabled recording with a clear message when the browser does not support MediaRecorder.
- Displayed microphone permission errors.

Verification:

- `npm test --prefix frontend` passed.
- `npm run build --prefix frontend` passed.
- Manual browser verification confirmed users can record an answer and play it back locally.
```

Set next step:

```md
Step 10 - 回答音檔上傳 API.
```

## Task 7: Final Verification, Review, and Commit

**Files:**
- All Step 9 files.

- [ ] **Step 1: Run frontend tests**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd test --prefix frontend
```

Expected: all frontend tests pass.

- [ ] **Step 2: Run frontend build**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd run build --prefix frontend
```

Expected: TypeScript and Vite build pass.

- [ ] **Step 3: Run backend tests as regression check**

Run:

```powershell
$env:GOCACHE='C:\Users\Henry\.codex\worktrees\22ac\interview-ai\backend\.cache\go-build'
$env:GOMODCACHE='C:\Users\Henry\.codex\worktrees\22ac\interview-ai\backend\.cache\gomod'
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
$env:POSTGRES_USER='interview_ai'
$env:POSTGRES_PASSWORD='interview_ai_dev_password'
$env:POSTGRES_DB='interview_ai'
$env:OPENAI_API_KEY=''
$env:OPENAI_MODEL='gpt-5.4-mini'
docker compose up -d postgres
docker compose run --rm migrate
docker compose up --build -d backend frontend
```

Create or reuse an interview, then verify:

```text
1. Open http://localhost:5173/interviews/new.
2. Create an interview with 2 questions.
3. On /interviews/{id}, click 開始模擬面試.
4. Confirm /interviews/{id}/session loads.
5. Click 開始錄音.
6. Grant microphone permission if the browser asks.
7. Confirm the UI shows 錄音中 and enables 停止錄音.
8. Speak a short answer, then click 停止錄音.
9. Confirm 回答錄音預覽 appears.
10. Play the preview audio and confirm the recorded answer is audible.
11. Click 下一題.
12. Confirm the previous preview is cleared and recording controls are ready for the next question.
```

- [ ] **Step 6: Request code review**

Ask a reviewer to inspect the completed Step 9 diff.

Provide:

- Scope: Step 9 browser recording only.
- Plan: `docs/superpowers/plans/2026-05-27-step9-frontend-recording.md`.
- Spec: `docs/mvp-spec.md` Step 9 and Section 9.2.
- Verification output summary.

Expected: no blocking or important issues. Fix any critical or important review findings before commit.

- [ ] **Step 7: Commit Step 9**

Run:

```powershell
git add frontend/src/App.test.tsx frontend/src/pages/InterviewSessionPage.tsx docs/DEVELOPMENT_PLAN.md docs/development-progress.md
git commit -m "feat: add frontend answer recording"
```

Expected: commit succeeds.

## Assumptions

- Step 9 is frontend-only because the upload API is explicitly Step 10.
- A single local recording preview for the current question is enough for the MVP slice.
- Moving to another question clears the local preview because answer persistence is outside Step 9.
- `audio/webm` is preferred when supported, matching the MVP spec recommendation.
- Browser recording support and microphone permission behavior vary by browser; the UI handles unsupported and denied-permission cases.
- No API docs update is required because no endpoint changes.

## Self-Review

- Spec coverage: Covers browser `MediaRecorder`, starting recording, stopping recording, local answer playback, unsupported-browser handling, permission errors, and media cleanup.
- Scope check: Does not implement upload, backend storage, answer DB records, completion flow, result page, STT, AI scoring, or follow-up questions.
- Placeholder scan: No placeholder markers remain.
- Type consistency: Uses existing `InterviewSessionPage` state and current question data; test helper names match test usage.
- Verification: Includes unit tests, build, backend regression tests, manual browser verification, review, docs updates, and commit command.
