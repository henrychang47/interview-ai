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
