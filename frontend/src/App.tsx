import { useState } from 'react'

import InterviewDetailPage from './pages/InterviewDetailPage'
import InterviewSessionPage from './pages/InterviewSessionPage'
import NewInterviewPage from './pages/NewInterviewPage'

const workflowItems = ['輸入職位資訊', '產生面試問題', '錄音回答', '查看結果']

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

export default function App() {
  const [route, setRoute] = useState(() => getRoute(window.location.pathname))

  function navigate(path: string) {
    window.history.pushState({}, '', path)
    setRoute(getRoute(path))
  }

  if (route.name === 'new') {
    return <NewInterviewPage onCreated={(interviewID) => navigate(`/interviews/${interviewID}`)} />
  }

  if (route.name === 'session') {
    return <InterviewSessionPage interviewID={route.interviewID} />
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
