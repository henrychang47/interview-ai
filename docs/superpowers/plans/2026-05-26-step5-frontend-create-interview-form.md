# Step 5 Frontend Create Interview Form Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the frontend `/interviews/new` flow so a user can create an interview from the browser and land on `/interviews/{id}` showing generated questions.

**Architecture:** Keep Step 5 as a frontend-only MVP slice that consumes the existing Step 3 and Step 4 backend APIs. Add a small typed API client, a minimal browser-location based route parser in `App`, a create-interview form page, and a read-only interview detail page. Do not add React Router or other dependencies yet; this keeps the MVP small and avoids introducing routing architecture before it is needed.

**Tech Stack:** React 18, TypeScript, Vite, Tailwind CSS, Vitest, Testing Library, browser `fetch`.

---

## Scope

Implement only Step 5 from `docs/mvp-spec.md`:

```text
Route: /interviews/new

- 輸入職位名稱
- 輸入職位要求及說明
- 輸入個人資訊
- 選擇題目數量
- 送出後建立 interview
- 成功後導向 /interviews/{id}
- 建立成功後可看到題目列表
```

Do not implement:

- `POST /api/interviews/{id}/start`
- `/interviews/{id}/session`
- TTS
- MediaRecorder
- answer upload
- result page
- real LLM integration

## UX Contract

Routes:

```text
/                      Existing homepage
/interviews/new         Create interview form
/interviews/{id}        Minimal detail page with questions
```

Create form fields:

- `job_title`
- `job_description`
- `user_profile`
- `question_count`, default `3`, min `1`, max `10`

Submit behavior:

1. Disable submit button while submitting.
2. Call `POST /api/interviews`.
3. On success, navigate to `/interviews/{id}`.
4. Detail page calls `GET /api/interviews/{id}`.
5. Detail page displays job title, status, and question list.
6. API errors display a visible error message.

## File Structure

- Create `frontend/src/types/interview.ts`: shared frontend API types.
- Create `frontend/src/api/interviews.ts`: typed `createInterview` and `getInterview`.
- Modify `frontend/vite.config.ts`: proxy browser `/api` requests to the backend in dev/Docker.
- Modify `docker-compose.yml`: set `VITE_API_PROXY_TARGET=http://backend:8080` for the frontend service.
- Modify `.env.example`: document `VITE_API_PROXY_TARGET`.
- Create `frontend/src/pages/NewInterviewPage.tsx`: form page.
- Create `frontend/src/pages/InterviewDetailPage.tsx`: minimal read-only detail page.
- Modify `frontend/src/App.tsx`: route between home, new form, and detail page.
- Modify `frontend/src/App.test.tsx`: cover routing, create flow, detail loading, and error states.
- Modify `README.md`: add frontend manual verification steps for Step 5.
- Modify `docs/development-progress.md`: mark Step 5 completed after implementation verification.
- Modify `docs/DEVELOPMENT_PLAN.md`: mark Step 5 completed after implementation verification.

## API Types

Use these frontend TypeScript shapes:

```ts
export type CreateInterviewRequest = {
  job_title: string
  job_description: string
  user_profile: string
  question_count: number
}

export type CreateInterviewResponse = {
  id: string
  status: string
}

export type Question = {
  id: string
  order: number
  text: string
}

export type Answer = {
  id: string
  question_id: string
  audio_path: string | null
  transcript_text: string | null
  created_at: string
}

export type InterviewDetail = {
  id: string
  job_title: string
  job_description: string
  user_profile: string
  question_count: number
  status: string
  questions: Question[]
  answers: Answer[]
}
```

## Task 1: Add Frontend API Proxy and Typed API Client

**Files:**
- Create: `frontend/src/types/interview.ts`
- Create: `frontend/src/api/interviews.ts`
- Modify: `frontend/vite.config.ts`
- Modify: `docker-compose.yml`
- Modify: `.env.example`

- [ ] **Step 1: Add Vite API proxy**

Update `frontend/vite.config.ts`:

```ts
import react from '@vitejs/plugin-react'
import { defineConfig } from 'vitest/config'

const apiProxyTarget = process.env.VITE_API_PROXY_TARGET ?? 'http://localhost:8080'

export default defineConfig({
  plugins: [react()],
  server: {
    host: '0.0.0.0',
    port: 5173,
    proxy: {
      '/api': {
        target: apiProxyTarget,
        changeOrigin: true,
      },
    },
  },
  test: {
    environment: 'jsdom',
    globals: false,
  },
})
```

- [ ] **Step 2: Configure Docker frontend proxy target**

Update `docker-compose.yml` frontend service:

```yaml
  frontend:
    build:
      context: ./frontend
    environment:
      VITE_API_PROXY_TARGET: ${VITE_API_PROXY_TARGET:-http://backend:8080}
    ports:
      - "5173:5173"
    depends_on:
      - backend
```

- [ ] **Step 3: Document proxy environment variable**

Add to `.env.example`:

```text
VITE_API_PROXY_TARGET=http://backend:8080
```

- [ ] **Step 4: Create frontend interview types**

Create `frontend/src/types/interview.ts`:

```ts
export type CreateInterviewRequest = {
  job_title: string
  job_description: string
  user_profile: string
  question_count: number
}

export type CreateInterviewResponse = {
  id: string
  status: string
}

export type Question = {
  id: string
  order: number
  text: string
}

export type Answer = {
  id: string
  question_id: string
  audio_path: string | null
  transcript_text: string | null
  created_at: string
}

export type InterviewDetail = {
  id: string
  job_title: string
  job_description: string
  user_profile: string
  question_count: number
  status: string
  questions: Question[]
  answers: Answer[]
}
```

- [ ] **Step 5: Create API client**

Create `frontend/src/api/interviews.ts`:

```ts
import type {
  CreateInterviewRequest,
  CreateInterviewResponse,
  InterviewDetail,
} from '../types/interview'

const API_BASE_URL = ''

async function parseJSONResponse<T>(response: Response, fallbackMessage: string): Promise<T> {
  let body: unknown = null
  try {
    body = await response.json()
  } catch {
    body = null
  }

  if (!response.ok) {
    const message =
      body && typeof body === 'object' && 'error' in body && typeof body.error === 'string'
        ? body.error
        : fallbackMessage
    throw new Error(message)
  }

  return body as T
}

export async function createInterview(input: CreateInterviewRequest): Promise<CreateInterviewResponse> {
  const response = await fetch(`${API_BASE_URL}/api/interviews`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(input),
  })

  return parseJSONResponse<CreateInterviewResponse>(response, '建立面試失敗')
}

export async function getInterview(interviewID: string): Promise<InterviewDetail> {
  const response = await fetch(`${API_BASE_URL}/api/interviews/${interviewID}`)

  return parseJSONResponse<InterviewDetail>(response, '載入面試失敗')
}
```

- [ ] **Step 6: Run build and compose checks**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd run build --prefix frontend
docker compose config
```

Expected: build passes and Docker Compose config is valid.

## Task 2: Add App Routing Tests

**Files:**
- Modify: `frontend/src/App.test.tsx`

- [ ] **Step 1: Replace App tests with route and flow coverage**

Replace `frontend/src/App.test.tsx` with:

```tsx
import '@testing-library/jest-dom/vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import App from './App'

const originalLocation = window.location

function mockPathname(pathname: string) {
  window.history.pushState({}, '', pathname)
}

function mockFetchOnce(response: unknown, init: ResponseInit = {}) {
  return vi.fn().mockResolvedValue(
    new Response(JSON.stringify(response), {
      status: init.status ?? 200,
      headers: { 'Content-Type': 'application/json' },
    }),
  )
}

describe('App', () => {
  beforeEach(() => {
    mockPathname('/')
    vi.restoreAllMocks()
  })

  afterEach(() => {
    vi.restoreAllMocks()
    window.history.pushState({}, '', '/')
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: originalLocation,
    })
  })

  it('renders the interview practice homepage', () => {
    render(<App />)

    expect(screen.getByRole('heading', { name: '模擬面試應用' })).toBeInTheDocument()
    expect(screen.getByText('建立面試、產生題目、錄音回答，逐步打通 MVP 主流程。')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: '建立新的模擬面試' })).toHaveAttribute(
      'href',
      '/interviews/new',
    )
  })

  it('renders the create interview form at /interviews/new', () => {
    mockPathname('/interviews/new')

    render(<App />)

    expect(screen.getByRole('heading', { name: '建立模擬面試' })).toBeInTheDocument()
    expect(screen.getByLabelText('職位名稱')).toBeInTheDocument()
    expect(screen.getByLabelText('職位要求及說明')).toBeInTheDocument()
    expect(screen.getByLabelText('個人資訊')).toBeInTheDocument()
    expect(screen.getByLabelText('題目數量')).toHaveValue(3)
  })

  it('submits the create interview form and navigates to detail route', async () => {
    mockPathname('/interviews/new')
    const fetchMock = mockFetchOnce({ id: 'interview-123', status: 'questions_ready' })
    vi.stubGlobal('fetch', fetchMock)
    const pushState = vi.spyOn(window.history, 'pushState')

    render(<App />)

    fireEvent.change(screen.getByLabelText('職位名稱'), { target: { value: '後端工程師' } })
    fireEvent.change(screen.getByLabelText('職位要求及說明'), {
      target: { value: '需要熟悉 Go、PostgreSQL、REST API' },
    })
    fireEvent.change(screen.getByLabelText('個人資訊'), {
      target: { value: '有 Java 和 Go 學習經驗' },
    })
    fireEvent.change(screen.getByLabelText('題目數量'), { target: { value: '3' } })
    fireEvent.click(screen.getByRole('button', { name: '建立面試' }))

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith('/api/interviews', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          job_title: '後端工程師',
          job_description: '需要熟悉 Go、PostgreSQL、REST API',
          user_profile: '有 Java 和 Go 學習經驗',
          question_count: 3,
        }),
      })
    })
    expect(pushState).toHaveBeenCalledWith({}, '', '/interviews/interview-123')
  })

  it('shows an API error when create interview fails', async () => {
    mockPathname('/interviews/new')
    vi.stubGlobal(
      'fetch',
      mockFetchOnce({ error: 'job_title is required' }, { status: 400 }),
    )

    render(<App />)

    fireEvent.click(screen.getByRole('button', { name: '建立面試' }))

    expect(await screen.findByText('job_title is required')).toBeInTheDocument()
  })

  it('loads interview details and displays questions at /interviews/:id', async () => {
    mockPathname('/interviews/interview-123')
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

    expect(await screen.findByRole('heading', { name: '後端工程師' })).toBeInTheDocument()
    expect(screen.getByText('questions_ready')).toBeInTheDocument()
    expect(screen.getByText('請介紹你過去與後端開發相關的經驗。')).toBeInTheDocument()
    expect(screen.getByText('你如何設計一個 REST API？')).toBeInTheDocument()
  })
})
```

- [ ] **Step 2: Run frontend tests to verify they fail**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd test --prefix frontend
```

Expected: tests fail because `/interviews/new`, API calls, and detail page are not implemented.

## Task 3: Add Create Interview Page

**Files:**
- Create: `frontend/src/pages/NewInterviewPage.tsx`

- [ ] **Step 1: Implement `NewInterviewPage`**

Create `frontend/src/pages/NewInterviewPage.tsx`:

```tsx
import { FormEvent, useState } from 'react'

import { createInterview } from '../api/interviews'
import type { CreateInterviewRequest } from '../types/interview'

type NewInterviewPageProps = {
  onCreated: (interviewID: string) => void
}

const initialForm: CreateInterviewRequest = {
  job_title: '',
  job_description: '',
  user_profile: '',
  question_count: 3,
}

export default function NewInterviewPage({ onCreated }: NewInterviewPageProps) {
  const [form, setForm] = useState<CreateInterviewRequest>(initialForm)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setIsSubmitting(true)
    setError(null)

    try {
      const response = await createInterview({
        job_title: form.job_title.trim(),
        job_description: form.job_description.trim(),
        user_profile: form.user_profile.trim(),
        question_count: form.question_count,
      })
      onCreated(response.id)
    } catch (error) {
      setError(error instanceof Error ? error.message : '建立面試失敗')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <main className="min-h-screen bg-slate-50 text-slate-950">
      <section className="mx-auto w-full max-w-3xl px-6 py-10">
        <a href="/" className="text-sm font-medium text-teal-700 hover:text-teal-800">
          返回首頁
        </a>
        <div className="mt-8">
          <p className="text-sm font-semibold uppercase tracking-wide text-teal-700">
            Interview Setup
          </p>
          <h1 className="mt-3 text-3xl font-bold leading-tight">建立模擬面試</h1>
        </div>

        <form onSubmit={handleSubmit} className="mt-8 space-y-6 rounded-lg border border-slate-200 bg-white p-6 shadow-sm">
          {error ? (
            <div className="rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
              {error}
            </div>
          ) : null}

          <label className="block">
            <span className="text-sm font-medium text-slate-800">職位名稱</span>
            <input
              className="mt-2 w-full rounded-md border border-slate-300 px-3 py-2 text-base outline-none focus:border-teal-600 focus:ring-2 focus:ring-teal-100"
              value={form.job_title}
              onChange={(event) => setForm((current) => ({ ...current, job_title: event.target.value }))}
              required
            />
          </label>

          <label className="block">
            <span className="text-sm font-medium text-slate-800">職位要求及說明</span>
            <textarea
              className="mt-2 min-h-32 w-full rounded-md border border-slate-300 px-3 py-2 text-base outline-none focus:border-teal-600 focus:ring-2 focus:ring-teal-100"
              value={form.job_description}
              onChange={(event) => setForm((current) => ({ ...current, job_description: event.target.value }))}
              required
            />
          </label>

          <label className="block">
            <span className="text-sm font-medium text-slate-800">個人資訊</span>
            <textarea
              className="mt-2 min-h-32 w-full rounded-md border border-slate-300 px-3 py-2 text-base outline-none focus:border-teal-600 focus:ring-2 focus:ring-teal-100"
              value={form.user_profile}
              onChange={(event) => setForm((current) => ({ ...current, user_profile: event.target.value }))}
              required
            />
          </label>

          <label className="block max-w-40">
            <span className="text-sm font-medium text-slate-800">題目數量</span>
            <input
              type="number"
              min={1}
              max={10}
              className="mt-2 w-full rounded-md border border-slate-300 px-3 py-2 text-base outline-none focus:border-teal-600 focus:ring-2 focus:ring-teal-100"
              value={form.question_count}
              onChange={(event) =>
                setForm((current) => ({
                  ...current,
                  question_count: Number(event.target.value),
                }))
              }
              required
            />
          </label>

          <button
            type="submit"
            disabled={isSubmitting}
            className="inline-flex min-h-11 items-center rounded-md bg-teal-700 px-5 py-2 text-sm font-semibold text-white hover:bg-teal-800 disabled:cursor-not-allowed disabled:bg-slate-400"
          >
            {isSubmitting ? '建立中...' : '建立面試'}
          </button>
        </form>
      </section>
    </main>
  )
}
```

- [ ] **Step 2: Run frontend tests**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd test --prefix frontend
```

Expected: tests still fail because App routing and detail page are not implemented yet.

## Task 4: Add Interview Detail Page

**Files:**
- Create: `frontend/src/pages/InterviewDetailPage.tsx`

- [ ] **Step 1: Implement `InterviewDetailPage`**

Create `frontend/src/pages/InterviewDetailPage.tsx`:

```tsx
import { useEffect, useState } from 'react'

import { getInterview } from '../api/interviews'
import type { InterviewDetail } from '../types/interview'

type InterviewDetailPageProps = {
  interviewID: string
}

export default function InterviewDetailPage({ interviewID }: InterviewDetailPageProps) {
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

  return (
    <main className="min-h-screen bg-slate-50 text-slate-950">
      <section className="mx-auto w-full max-w-4xl px-6 py-10">
        <a href="/interviews/new" className="text-sm font-medium text-teal-700 hover:text-teal-800">
          建立另一場面試
        </a>

        {isLoading ? <p className="mt-8 text-slate-600">載入面試中...</p> : null}

        {error ? (
          <div className="mt-8 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
            {error}
          </div>
        ) : null}

        {interview ? (
          <div className="mt-8">
            <div className="flex flex-col gap-3 border-b border-slate-200 pb-6 sm:flex-row sm:items-start sm:justify-between">
              <div>
                <p className="text-sm font-semibold uppercase tracking-wide text-teal-700">
                  Interview Detail
                </p>
                <h1 className="mt-3 text-3xl font-bold leading-tight">{interview.job_title}</h1>
                <p className="mt-3 max-w-2xl text-slate-700">{interview.job_description}</p>
              </div>
              <span className="inline-flex w-fit rounded-md border border-teal-200 bg-teal-50 px-3 py-1 text-sm font-medium text-teal-800">
                {interview.status}
              </span>
            </div>

            <section className="mt-8">
              <h2 className="text-xl font-semibold text-slate-900">面試問題</h2>
              <ol className="mt-4 space-y-3">
                {interview.questions.map((question) => (
                  <li key={question.id} className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm">
                    <p className="text-sm font-semibold text-teal-700">問題 {question.order}</p>
                    <p className="mt-2 text-slate-900">{question.text}</p>
                  </li>
                ))}
              </ol>
            </section>
          </div>
        ) : null}
      </section>
    </main>
  )
}
```

- [ ] **Step 2: Run frontend tests**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd test --prefix frontend
```

Expected: tests still fail because `App` routing is not wired yet.

## Task 5: Wire App Routes and Navigation

**Files:**
- Modify: `frontend/src/App.tsx`

- [ ] **Step 1: Replace App with minimal route handling**

Replace `frontend/src/App.tsx` with:

```tsx
import { useState } from 'react'

import InterviewDetailPage from './pages/InterviewDetailPage'
import NewInterviewPage from './pages/NewInterviewPage'

const workflowItems = ['輸入職位資訊', '產生面試問題', '錄音回答', '查看結果']

function getRoute(pathname: string) {
  if (pathname === '/interviews/new') {
    return { name: 'new' as const }
  }

  const detailMatch = pathname.match(/^\/interviews\/([^/]+)$/)
  if (detailMatch) {
    return { name: 'detail' as const, interviewID: decodeURIComponent(detailMatch[1]) }
  }

  return { name: 'home' as const }
}

export default function App() {
  const [route, setRoute] = useState(() => getRoute(window.location.pathname))

  function navigate(path: string) {
    window.history.pushState({}, '', path)
    setRoute(getRoute(path))
  }

  if (route.name === 'new') {
    return <NewInterviewPage onCreated={(interviewID) => navigate(`/interviews/${interviewID}`)} />
  }

  if (route.name === 'detail') {
    return <InterviewDetailPage interviewID={route.interviewID} />
  }

  return (
    <main className="min-h-screen bg-slate-50 text-slate-950">
      <section className="mx-auto flex min-h-screen w-full max-w-5xl flex-col justify-center px-6 py-12">
        <div className="max-w-3xl">
          <p className="text-sm font-semibold uppercase tracking-wide text-teal-700">
            Interview Practice MVP
          </p>
          <h1 className="mt-4 text-4xl font-bold leading-tight sm:text-5xl">模擬面試應用</h1>
          <p className="mt-5 max-w-2xl text-lg leading-8 text-slate-700">
            建立面試、產生題目、錄音回答，逐步打通 MVP 主流程。
          </p>
          <a
            href="/interviews/new"
            className="mt-8 inline-flex min-h-11 items-center rounded-md bg-teal-700 px-5 py-2 text-sm font-semibold text-white hover:bg-teal-800"
          >
            建立新的模擬面試
          </a>
        </div>

        <div className="mt-10 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {workflowItems.map((item, index) => (
            <div key={item} className="rounded-lg border border-slate-200 bg-white p-5 shadow-sm">
              <div className="flex h-9 w-9 items-center justify-center rounded-full bg-teal-100 text-sm font-bold text-teal-800">
                {index + 1}
              </div>
              <p className="mt-4 font-medium text-slate-900">{item}</p>
            </div>
          ))}
        </div>
      </section>
    </main>
  )
}
```

- [ ] **Step 2: Run frontend tests to verify they pass**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd test --prefix frontend
```

Expected: all frontend tests pass.

- [ ] **Step 3: Run frontend build**

Run:

```powershell
C:\nvm4w\nodejs\npm.cmd run build --prefix frontend
```

Expected: build passes.

## Task 6: Verify Manually in Browser

**Files:**
- No source edits.

- [ ] **Step 1: Start backend and frontend**

Run:

```powershell
docker compose up -d postgres
docker compose run --rm migrate
docker compose up --build -d backend frontend
```

Expected: backend listens on `http://localhost:8080`; frontend listens on `http://localhost:5173`.

- [ ] **Step 2: Manual browser verification**

Open:

```text
http://localhost:5173/interviews/new
```

Use this test data:

```text
職位名稱: 後端工程師
職位要求及說明: 需要熟悉 Go、PostgreSQL、REST API
個人資訊: 有 Java 和 Go 學習經驗，正在準備後端工程師面試
題目數量: 3
```

Expected:

- Submit creates an interview.
- Browser URL changes to `/interviews/{id}`.
- Detail page shows `後端工程師`.
- Detail page shows `questions_ready`.
- Detail page shows 3 generated questions.

If browser submission cannot reach the backend, first verify that `docker compose config` shows `VITE_API_PROXY_TARGET=http://backend:8080` for the frontend service and that `frontend/vite.config.ts` proxies `/api`.

## Task 7: Update Documentation and Progress

**Files:**
- Modify: `README.md`
- Modify: `docs/development-progress.md`
- Modify: `docs/DEVELOPMENT_PLAN.md`

- [ ] **Step 1: Update README manual verification**

Add this section under frontend verification:

````md
建立面試表單：

```text
http://localhost:5173/interviews/new
```

手動驗收：

- 輸入職位名稱、職位要求及說明、個人資訊、題目數量。
- 送出表單後，頁面導向 `/interviews/{id}`。
- 詳情頁顯示職位名稱、狀態與產生的題目列表。
````

Also add this local dev note:

````md
When running the frontend outside Docker, make sure the backend is on `http://localhost:8080`, or set:

```powershell
$env:VITE_API_PROXY_TARGET='http://localhost:8080'
```
````

- [ ] **Step 2: Update progress documents**

In `docs/development-progress.md`:

```md
- Current step: Step 5 - 前端建立面試表單
- Status: Completed
- Last updated: 2026-05-26
```

Change Step 5 table status to `Completed`.

Add:

```md
### Step 5 - 前端建立面試表單

Completed on 2026-05-26.

Implemented:

- Added `/interviews/new` frontend route.
- Added create interview form.
- Submitted form data to `POST /api/interviews`.
- Navigated to `/interviews/{id}` after successful creation.
- Added minimal detail page that loads `GET /api/interviews/{id}` and displays questions.

Verification:

- `npm test --prefix frontend` passed.
- `npm run build --prefix frontend` passed.
- Manual browser verification confirmed creating an interview shows the generated question list.
```

In `docs/DEVELOPMENT_PLAN.md`, update current status and Step 5 table status to `Completed`, then set next step to Step 6.

## Task 8: Final Checks, Review, and Commit

**Files:**
- All Step 5 files.

- [ ] **Step 1: Run final checks**

Run from repo root:

```powershell
$env:GOCACHE='D:\projects\interview-ai\.cache\go-build'
$env:GOMODCACHE='D:\projects\interview-ai\.cache\gomod'
Push-Location backend
Remove-Item Env:\DATABASE_URL -ErrorAction SilentlyContinue
go test ./...
Pop-Location
C:\nvm4w\nodejs\npm.cmd test --prefix frontend
C:\nvm4w\nodejs\npm.cmd run build --prefix frontend
docker compose config
git diff --check
```

Expected: all commands exit 0.

- [ ] **Step 2: Use subagent review after implementation**

Ask a subagent reviewer to inspect the completed Step 5 diff. Provide:

- Scope: Step 5 frontend create interview form only.
- Plan: `docs/superpowers/plans/2026-05-26-step5-frontend-create-interview-form.md`.
- Spec: `docs/mvp-spec.md` Step 5.
- Verification output summary.

Expected: no blocking issues. Fix any critical or important review findings before commit.

- [ ] **Step 3: Review changed files**

Run:

```powershell
git status --short
git diff --stat
```

Expected: changed files are limited to Step 5 frontend code/tests and docs.

- [ ] **Step 4: Commit Step 5**

Run:

```powershell
git add frontend/src frontend/vite.config.ts docker-compose.yml .env.example README.md docs/DEVELOPMENT_PLAN.md docs/development-progress.md
git commit -m "feat: add frontend create interview flow"
```

Expected: commit succeeds.

## Assumptions

- Step 5 may include a minimal `/interviews/{id}` detail page because the acceptance criteria says the user should see the generated question list after creation.
- Step 5 does not implement the full Step 8.2 detail page start button because `POST /api/interviews/{id}/start` is later backend scope and is not yet implemented.
- Step 5 does not add React Router. A tiny route parser is enough for the current MVP routes.
- The frontend API client uses relative `/api` URLs. Vite proxies `/api` to the backend during dev/Docker runs.
- Tests use `fireEvent` because `@testing-library/user-event` is not currently installed.

## Self-Review

- Spec coverage: Covers `/interviews/new`, form input, POST submission, successful navigation, and generated question display.
- Scope check: Does not implement start/session/TTS/recording/upload/result.
- Placeholder scan: No TBD/TODO placeholders remain.
- Type consistency: Request/response property names match `docs/API.md`.
