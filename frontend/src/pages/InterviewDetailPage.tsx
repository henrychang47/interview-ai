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
          </div>
        ) : null}
      </section>
    </main>
  )
}
