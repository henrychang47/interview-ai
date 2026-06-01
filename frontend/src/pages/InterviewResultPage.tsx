import { useEffect, useMemo, useRef, useState } from 'react'

import { getInterview } from '../api/interviews'
import { MarkdownText } from '../components/MarkdownText'
import { Card, Icon, InfoDisclosure, PageShell, StatusBadge, TopBar } from '../components/ui'
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
  const isPollingRef = useRef(false)

  useEffect(() => {
    let isMounted = true

    async function loadInitialInterview() {
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

    loadInitialInterview()

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

  const hasPendingAnalysis = useMemo(() => {
    return (interview?.answers ?? []).some((answer) =>
      answer.analysis_status === 'pending' || answer.analysis_status === 'processing',
    )
  }, [interview?.answers])

  useEffect(() => {
    if (!hasPendingAnalysis) {
      return
    }

    const intervalID = window.setInterval(() => {
      if (isPollingRef.current) {
        return
      }

      isPollingRef.current = true
      getInterview(interviewID)
        .then((detail) => {
          setInterview(detail)
          setError(null)
        })
        .catch((error) => {
          setError(error instanceof Error ? error.message : '載入面試結果失敗')
        })
        .finally(() => {
          isPollingRef.current = false
        })
    }, 3000)

    return () => {
      window.clearInterval(intervalID)
      isPollingRef.current = false
    }
  }, [hasPendingAnalysis, interviewID])

  return (
    <>
      <TopBar
        action={{ href: `/interviews/${interviewID}`, label: '返回面試詳情', icon: 'arrow_back' }}
      />
      <PageShell maxWidth="max-w-container-max">
      {isLoading ? <p className="mt-lg text-body-md text-on-surface-variant">載入面試結果中...</p> : null}

      {error ? (
        <Card className="mt-lg flex items-start gap-sm border-error/20 bg-error-container p-md text-on-error-container">
          <Icon name="warning" className="mt-xs" />
          <p className="text-body-sm">{error}</p>
        </Card>
      ) : null}

      {interview ? (
        <div>
          <div className="mb-lg flex flex-col gap-md md:flex-row md:items-start md:justify-between">
            <div>
              <p className="text-label-md font-bold uppercase text-primary">Interview Result</p>
              <h1 className="mt-sm font-headline text-headline-lg-mobile font-bold text-on-surface md:text-headline-lg">
                面試結果
              </h1>
              <h2 className="mt-md font-headline text-headline-md text-on-surface">
                {interview.job_title}
              </h2>
            </div>
            <StatusBadge tone="primary">{interview.status}</StatusBadge>
          </div>

          <div className="mb-xl grid grid-cols-1 gap-md md:grid-cols-2">
            <InfoDisclosure title="職位資訊">{interview.job_description}</InfoDisclosure>
            <InfoDisclosure title="個人資訊">{interview.user_profile}</InfoDisclosure>
          </div>

          <section>
            <h3 className="mb-md font-headline text-headline-md text-on-surface">面試問答回顧</h3>
            <ol className="space-y-lg">
              {interview.questions.map((question) => {
                const answer = answersByQuestionID[question.id]
                const audioURL = answerAudioURL(answer?.audio_path ?? null)

                return (
                  <li key={question.id}>
                    <Card className="overflow-hidden transition-all hover:border-outline hover:shadow-calm-lg">
                      <div className="flex items-start gap-md border-b border-outline-variant bg-surface-container-low p-md">
                        <div
                          className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-lg text-label-md font-bold ${
                            answer
                              ? 'bg-primary text-on-primary'
                              : 'bg-outline-variant text-on-surface-variant'
                          }`}
                        >
                          Q{question.order}
                        </div>
                        <div className="min-w-0 flex-1">
                          <h4
                            className={`text-body-lg font-semibold leading-8 ${
                              answer ? 'text-on-surface' : 'text-on-surface-variant'
                            }`}
                          >
                            {question.text}
                          </h4>
                        </div>
                      </div>

                      <div className="space-y-md p-md md:p-lg">
                        {answer ? (
                          <>
                            {audioURL ? (
                              <div className="rounded-xl border border-outline-variant bg-surface-container-lowest p-md">
                                <p className="mb-sm flex items-center gap-xs text-label-md font-bold text-on-surface-variant">
                                  <Icon name="play_arrow" className="text-[18px]" />
                                  回答音檔
                                </p>
                                <audio
                                  aria-label={`問題 ${question.order} 回答音檔`}
                                  controls
                                  src={audioURL}
                                  className="w-full"
                                />
                              </div>
                            ) : (
                              <div className="rounded-xl border border-error/30 bg-error-container/40 p-md text-body-sm text-on-error-container">
                                尚未上傳回答
                              </div>
                            )}

                            {answer.analysis_status === 'failed' ? (
                              <div className="rounded-xl border border-error/30 bg-error-container/40 p-md text-on-error-container">
                                <p className="flex items-center gap-xs text-label-md font-bold">
                                  <Icon name="warning" className="text-[16px]" />
                                  分析失敗，請稍後重新上傳回答
                                </p>
                                {answer.analysis_error ? (
                                  <p className="mt-sm text-body-sm">{answer.analysis_error}</p>
                                ) : null}
                              </div>
                            ) : answer.analysis_status === 'completed' ? (
                              <>
                                <div className="relative rounded-xl border border-outline-variant bg-surface-bright p-md">
                                  <p className="mb-sm flex items-center gap-xs text-label-md font-bold text-primary">
                                    <Icon name="notes" className="text-[16px]" />
                                    轉錄文字
                                  </p>
                                  <p className="text-body-md leading-7 text-on-surface">
                                    {answer.transcript_text ?? '尚未轉錄'}
                                  </p>
                                </div>
                                <div className="relative rounded-xl border border-outline-variant bg-surface-bright p-md">
                                  <p className="mb-sm flex items-center gap-xs text-label-md font-bold text-primary">
                                    <Icon name="tips_and_updates" className="text-[16px]" />
                                    改進建議
                                  </p>
                                  <MarkdownText text={answer.improvement_suggestions ?? '尚無建議'} />
                                </div>
                              </>
                            ) : (
                              <div className="rounded-xl border border-outline-variant bg-surface-bright p-md">
                                <p className="flex items-center gap-xs text-label-md font-bold text-primary">
                                  <Icon name="hourglass_empty" className="text-[16px]" />
                                  AI 分析中
                                </p>
                                <p className="mt-sm text-body-sm text-on-surface-variant">
                                  Gemini 正在產生逐字稿與改進建議，稍後會自動更新。
                                </p>
                              </div>
                            )}
                          </>
                        ) : (
                          <div className="flex flex-col items-center justify-center rounded-xl border border-error/30 bg-error-container/40 py-lg text-center">
                            <Icon name="mic_off" className="mb-sm text-error" />
                            <p className="text-label-md font-bold text-error">尚未上傳回答</p>
                          </div>
                        )}
                      </div>
                    </Card>
                  </li>
                )
              })}
            </ol>
          </section>
        </div>
      ) : null}
      </PageShell>
    </>
  )
}
