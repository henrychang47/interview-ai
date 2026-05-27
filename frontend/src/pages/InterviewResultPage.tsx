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
