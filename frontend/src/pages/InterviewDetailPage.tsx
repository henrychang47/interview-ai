import { useCallback, useEffect, useRef, useState } from 'react'

import { getInterview, startInterview } from '../api/interviews'
import { Button, Card, Icon, InfoDisclosure, LinkButton, PageShell, StatusBadge, TopBar } from '../components/ui'
import type { InterviewDetail } from '../types/interview'

type InterviewDetailPageProps = {
  interviewID: string
}

export default function InterviewDetailPage({ interviewID }: InterviewDetailPageProps) {
  const [interview, setInterview] = useState<InterviewDetail | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [isStarting, setIsStarting] = useState(false)
  const isPollingRef = useRef(false)

  const loadInterview = useCallback(async ({ showLoading = true } = {}) => {
    if (showLoading) {
      setIsLoading(true)
    }
    setError(null)

    try {
      const detail = await getInterview(interviewID)
      setInterview(detail)
    } catch (error) {
      setError(error instanceof Error ? error.message : '載入面試失敗')
    } finally {
      if (showLoading) {
        setIsLoading(false)
      }
    }
  }, [interviewID])

  useEffect(() => {
    loadInterview()
  }, [loadInterview])

  useEffect(() => {
    if (interview?.status !== 'generating_questions') {
      return
    }

    const intervalID = window.setInterval(async () => {
      if (isPollingRef.current) {
        return
      }

      isPollingRef.current = true
      try {
        await loadInterview({ showLoading: false })
      } finally {
        isPollingRef.current = false
      }
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
    <>
      <TopBar
        action={{ href: '/interviews/new', label: '建立另一場面試', icon: 'add_circle' }}
      />
      <PageShell maxWidth="max-w-container-max">
      {isLoading ? (
        <Card className="flex min-h-[45vh] flex-col items-center justify-center p-xl text-center">
          <div className="relative mb-lg h-16 w-16">
            <div className="absolute inset-0 rounded-full border-4 border-primary/20" />
            <div className="absolute inset-0 animate-spin rounded-full border-4 border-primary border-t-transparent" />
          </div>
          <p className="text-body-md text-on-surface-variant">載入面試中...</p>
        </Card>
      ) : null}

      {error ? (
        <Card className="mb-lg flex items-start gap-sm border-error/20 bg-error-container p-md text-on-error-container">
          <Icon name="warning" className="mt-xs" />
          <p className="text-body-sm">{error}</p>
        </Card>
      ) : null}

      {interview ? (
        <div>
          <div className="mb-lg flex flex-col gap-md md:flex-row md:items-start md:justify-between">
            <div>
              <p className="text-label-md font-bold uppercase text-primary">Interview Detail</p>
              <h1 className="mt-sm font-headline text-headline-lg-mobile font-bold text-on-surface md:text-headline-lg">
                {interview.job_title}
              </h1>
            </div>
            <StatusBadge tone="primary">{interview.status}</StatusBadge>
          </div>

          <div className="mb-xl grid grid-cols-1 gap-md md:grid-cols-2">
            <InfoDisclosure title="職位資訊">{interview.job_description}</InfoDisclosure>
            <InfoDisclosure title="個人資訊">{interview.user_profile}</InfoDisclosure>
          </div>

          {interview.status === 'generating_questions' ? (
            <Card className="flex min-h-[45vh] flex-col items-center justify-center p-xl text-center">
              <div className="relative mb-lg h-16 w-16">
                <div className="absolute inset-0 rounded-full border-4 border-primary/20" />
                <div className="absolute inset-0 animate-spin rounded-full border-4 border-primary border-t-transparent" />
              </div>
              <h2 className="font-headline text-headline-md text-on-surface">題目準備中</h2>
              <p className="mt-sm max-w-md text-body-sm text-on-surface-variant">
                AI 導師正在根據您的背景與職位需求，為您量身定制面試題目。
              </p>
            </Card>
          ) : null}

          {interview.status === 'questions_ready' ? (
            <Card className="p-lg md:p-xl">
              <div className="flex flex-col gap-lg md:flex-row md:items-center md:justify-between">
                <div>
                  <h2 className="font-headline text-headline-md text-on-surface">題目已準備完成</h2>
                  <p className="mt-sm text-body-sm text-on-surface-variant">
                    準備好後即可開始。題目會在面試過程中逐題朗讀。
                  </p>
                </div>
                <Button
                  type="button"
                  onClick={handleStartInterview}
                  disabled={isStarting}
                  icon="play_arrow"
                  className="w-full md:w-auto"
                >
                  {isStarting ? '開始中...' : '開始模擬面試'}
                </Button>
              </div>
            </Card>
          ) : null}

          {interview.status === 'failed' ? (
            <Card className="border-error/20 bg-error-container p-lg text-on-error-container">
              <h2 className="font-headline text-headline-sm">題目產生失敗</h2>
              <p className="mt-sm text-body-sm">請建立另一場面試。</p>
            </Card>
          ) : null}

          {interview.status === 'in_progress' ? (
            <Card className="p-lg md:p-xl">
              <h2 className="font-headline text-headline-md text-on-surface">面試進行中</h2>
              <p className="mt-sm text-body-sm text-on-surface-variant">
                您可以回到目前的面試流程繼續作答。
              </p>
              <LinkButton href={`/interviews/${interview.id}/session`} icon="play_arrow" className="mt-lg">
                繼續面試
              </LinkButton>
            </Card>
          ) : null}

          {interview.status === 'completed' ? (
            <Card className="p-lg md:p-xl">
              <h2 className="font-headline text-headline-md text-on-surface">面試已完成</h2>
              <p className="mt-sm text-body-sm text-on-surface-variant">
                查看每一題題目與您上傳的回答音檔。
              </p>
              <LinkButton href={`/interviews/${interview.id}/result`} icon="query_stats" className="mt-lg">
                查看結果
              </LinkButton>
            </Card>
          ) : null}
        </div>
      ) : null}
      </PageShell>
    </>
  )
}
