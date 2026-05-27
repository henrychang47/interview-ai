import { useCallback, useEffect, useState } from 'react'

import { getInterview, startInterview } from '../api/interviews'
import type { InterviewDetail } from '../types/interview'

type InterviewDetailPageProps = {
  interviewID: string
}

export default function InterviewDetailPage({ interviewID }: InterviewDetailPageProps) {
  const [interview, setInterview] = useState<InterviewDetail | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [isStarting, setIsStarting] = useState(false)

  const loadInterview = useCallback(async () => {
    setIsLoading(true)
    setError(null)

    try {
      const detail = await getInterview(interviewID)
      setInterview(detail)
    } catch (error) {
      setError(error instanceof Error ? error.message : '載入面試失敗')
    } finally {
      setIsLoading(false)
    }
  }, [interviewID])

  useEffect(() => {
    loadInterview()
  }, [loadInterview])

  useEffect(() => {
    if (interview?.status !== 'generating_questions') {
      return
    }

    const intervalID = window.setInterval(() => {
      loadInterview()
    }, 2000)

    return () => window.clearInterval(intervalID)
  }, [interview?.status, loadInterview])

  async function handleStartInterview() {
    if (!interview) {
      return
    }

    setIsStarting(true)
    setError(null)

    try {
      await startInterview(interview.id)
      const path = `/interviews/${interview.id}/session`
      window.history.pushState({}, '', path)
      window.dispatchEvent(new PopStateEvent('popstate'))
    } catch (error) {
      setError(error instanceof Error ? error.message : '開始面試失敗')
    } finally {
      setIsStarting(false)
    }
  }

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

            <section className="mt-8 space-y-4">
              {interview.status === 'generating_questions' ? (
                <h2 className="text-xl font-semibold text-slate-900">題目準備中</h2>
              ) : null}

              {interview.status === 'questions_ready' ? (
                <>
                  <h2 className="text-xl font-semibold text-slate-900">題目已準備完成</h2>
                  <button
                    type="button"
                    onClick={handleStartInterview}
                    disabled={isStarting}
                    className="inline-flex min-h-11 items-center rounded-md bg-teal-700 px-5 py-2 text-sm font-semibold text-white hover:bg-teal-800 disabled:cursor-not-allowed disabled:bg-slate-400"
                  >
                    {isStarting ? '開始中...' : '開始模擬面試'}
                  </button>
                </>
              ) : null}

              {interview.status === 'failed' ? (
                <p className="text-red-700">題目產生失敗，請建立另一場面試。</p>
              ) : null}

              {interview.status === 'in_progress' ? (
                <>
                  <h2 className="text-xl font-semibold text-slate-900">面試進行中</h2>
                  <a
                    href={`/interviews/${interview.id}/session`}
                    className="inline-flex min-h-11 items-center rounded-md bg-teal-700 px-5 py-2 text-sm font-semibold text-white hover:bg-teal-800"
                  >
                    繼續面試
                  </a>
                </>
              ) : null}

              {interview.status === 'completed' ? (
                <>
                  <h2 className="text-xl font-semibold text-slate-900">面試已完成</h2>
                  <a
                    href={`/interviews/${interview.id}/result`}
                    className="inline-flex min-h-11 items-center rounded-md bg-teal-700 px-5 py-2 text-sm font-semibold text-white hover:bg-teal-800"
                  >
                    查看結果
                  </a>
                </>
              ) : null}
            </section>
          </div>
        ) : null}
      </section>
    </main>
  )
}
