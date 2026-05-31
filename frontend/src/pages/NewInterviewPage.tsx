import { FormEvent, MouseEvent, useState } from 'react'

import { createInterview } from '../api/interviews'
import { Button, Card, Icon, StatusBadge, StepProgress, TopBar } from '../components/ui'
import type { CreateInterviewRequest } from '../types/interview'

type NewInterviewPageProps = {
  onCreated: (interviewID: string) => void
}

const initialForm: CreateInterviewRequest = {
  job_title: '',
  job_description: '',
  user_profile: '',
  question_count: 3,
  question_language: 'zh-TW',
}

type SetupStep = 'profile' | 'settings'

function getMicrophoneErrorMessage(error: unknown) {
  if (error instanceof DOMException || error instanceof Error) {
    if (
      error.name === 'NotFoundError' ||
      error.name === 'DevicesNotFoundError' ||
      error.message.toLowerCase().includes('device not found')
    ) {
      return '找不到可用的麥克風，請確認裝置已連接，並在瀏覽器或系統設定中允許麥克風。'
    }

    if (error.name === 'NotAllowedError' || error.name === 'PermissionDeniedError') {
      return '麥克風權限已被拒絕，請在瀏覽器設定中允許此網站使用麥克風。'
    }

    return error.message
  }

  return '無法測試麥克風'
}

export default function NewInterviewPage({ onCreated }: NewInterviewPageProps) {
  const [form, setForm] = useState<CreateInterviewRequest>(initialForm)
  const [step, setStep] = useState<SetupStep>('profile')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [isTestingMicrophone, setIsTestingMicrophone] = useState(false)
  const [microphoneReady, setMicrophoneReady] = useState(false)
  const [microphoneError, setMicrophoneError] = useState<string | null>(null)
  const isProfileComplete =
    form.job_title.trim() !== '' &&
    form.job_description.trim() !== '' &&
    form.user_profile.trim() !== ''

  async function testMicrophone() {
    setIsTestingMicrophone(true)
    setMicrophoneError(null)

    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
      stream.getTracks().forEach((track) => track.stop())
      setMicrophoneReady(true)
    } catch (error) {
      setMicrophoneReady(false)
      setMicrophoneError(getMicrophoneErrorMessage(error))
    } finally {
      setIsTestingMicrophone(false)
    }
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (step === 'profile') {
      if (!isProfileComplete) {
        return
      }
      setStep('settings')
      return
    }

    setIsSubmitting(true)
    setError(null)

    try {
      const response = await createInterview({
        job_title: form.job_title.trim(),
        job_description: form.job_description.trim(),
        user_profile: form.user_profile.trim(),
        question_count: form.question_count,
        question_language: form.question_language,
      })
      onCreated(response.id)
    } catch (error) {
      setError(error instanceof Error ? error.message : '建立面試失敗')
    } finally {
      setIsSubmitting(false)
    }
  }

  function handleBackToProfile(event: MouseEvent<HTMLButtonElement>) {
    event.preventDefault()
    event.stopPropagation()
    setStep('profile')
  }

  const currentStep = step === 'profile' ? 1 : 2

  return (
    <>
      <TopBar maxWidth="max-w-readable" action={{ href: '/', label: '返回首頁', icon: 'arrow_back' }} />
      <main className="min-h-screen bg-surface text-on-surface">
      <section className="mx-auto w-full max-w-readable px-margin-mobile py-lg md:px-margin-desktop md:py-xl">
        <div className="mb-lg">
          <p className="text-label-md font-bold uppercase text-primary">Interview Setup</p>
          <h1 className="mt-sm font-headline text-headline-lg-mobile font-bold text-on-surface md:text-headline-lg">
            建立新面試
          </h1>
          <p className="mt-sm text-body-md text-on-surface-variant">
            請提供職位資訊，AI 將為您量身打造面試情境。
          </p>
        </div>

        <div className="relative z-0 mb-xl">
          <StepProgress currentStep={currentStep} steps={['職位資訊', '設定與測試']} />
        </div>

        <form onSubmit={handleSubmit}>
          <Card className="p-lg md:p-xl">
            {error ? (
              <div className="mb-lg flex items-start gap-sm rounded-lg border border-error/20 bg-error-container p-md text-on-error-container">
                <Icon name="warning" className="mt-xs" />
                <div>
                  <p className="text-label-md font-bold">建立失敗</p>
                  <p className="mt-xs text-body-sm">{error}</p>
                </div>
              </div>
            ) : null}

            {step === 'profile' ? (
              <>
                <div className="space-y-lg">
                  <label className="block">
                    <span className="mb-sm block text-label-md font-semibold text-on-surface">
                      職位名稱
                    </span>
                    <input
                      className="focus-calm w-full rounded-lg border border-outline-variant bg-surface-container-lowest p-3 text-body-md text-on-surface placeholder:text-on-surface-variant"
                      placeholder="例如：後端工程師"
                      required
                      value={form.job_title}
                      onChange={(event) =>
                        setForm((current) => ({ ...current, job_title: event.target.value }))
                      }
                    />
                  </label>

                  <label className="block">
                    <span className="mb-sm block text-label-md font-semibold text-on-surface">
                      職位要求及說明
                    </span>
                    <textarea
                      className="focus-calm min-h-32 w-full rounded-lg border border-outline-variant bg-surface-container-lowest p-3 text-body-md text-on-surface placeholder:text-on-surface-variant"
                      placeholder="貼上職缺描述或條列主要技能需求..."
                      required
                      value={form.job_description}
                      onChange={(event) =>
                        setForm((current) => ({ ...current, job_description: event.target.value }))
                      }
                    />
                  </label>

                  <label className="block">
                    <span className="mb-sm block text-label-md font-semibold text-on-surface">
                      個人資訊
                    </span>
                    <textarea
                      className="focus-calm min-h-32 w-full rounded-lg border border-outline-variant bg-surface-container-lowest p-3 text-body-md text-on-surface placeholder:text-on-surface-variant"
                      placeholder="簡述您的相關經驗或貼上履歷摘要..."
                      required
                      value={form.user_profile}
                      onChange={(event) =>
                        setForm((current) => ({ ...current, user_profile: event.target.value }))
                      }
                    />
                  </label>
                </div>

                <div className="mt-xl flex justify-end">
                  <Button type="submit" disabled={!isProfileComplete}>
                    下一步
                  </Button>
                </div>
              </>
            ) : (
              <>
                <div className="space-y-lg">
                  <div className="grid grid-cols-1 gap-lg border-b border-outline-variant pb-lg md:grid-cols-2">
                    <label className="block">
                      <span className="mb-sm block text-label-md font-semibold text-on-surface">
                        題目數量
                      </span>
                      <input
                        type="number"
                        min={1}
                        max={10}
                        className="focus-calm w-full rounded-lg border border-outline-variant bg-surface-container-lowest p-3 text-body-md text-on-surface"
                        value={form.question_count}
                        onChange={(event) =>
                          setForm((current) => ({
                            ...current,
                            question_count: Number(event.target.value),
                          }))
                        }
                      />
                    </label>

                    <fieldset>
                      <legend className="mb-sm text-label-md font-semibold text-on-surface">
                        題目語言
                      </legend>
                      <div className="flex flex-wrap gap-md">
                        {[
                          ['zh-TW', '繁體中文'],
                          ['en-US', 'English'],
                        ].map(([value, label]) => {
                          const checked = form.question_language === value
                          return (
                            <label
                              key={value}
                              className={`flex flex-1 cursor-pointer items-center justify-center gap-sm rounded-lg px-4 py-2.5 transition-all ${
                                checked
                                  ? 'border-2 border-primary bg-primary/5 text-primary'
                                  : 'border border-outline-variant text-on-surface-variant hover:bg-surface-container-low'
                              }`}
                            >
                              <input
                                type="radio"
                                name="question_language"
                                value={value}
                                checked={checked}
                                onChange={(event) =>
                                  setForm((current) => ({
                                    ...current,
                                    question_language: event.target.value,
                                  }))
                                }
                              />
                              <span className="text-body-md font-semibold">{label}</span>
                            </label>
                          )
                        })}
                      </div>
                    </fieldset>
                  </div>

                  <div>
                    <h2 className="mb-md text-label-md font-semibold text-on-surface">裝置測試</h2>
                    <div className="flex flex-col items-center justify-between gap-md rounded-lg border border-outline-variant bg-surface-container-low p-md md:flex-row">
                      <div className="flex w-full items-center gap-md md:w-auto">
                        <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-full bg-primary-fixed text-primary">
                          <Icon name="mic" />
                        </div>
                        <div>
                          <p className="text-body-md font-bold">麥克風測試</p>
                          <p className="text-body-sm text-on-surface-variant">
                            請允許存取麥克風以進行測試
                          </p>
                        </div>
                      </div>
                      <Button
                        type="button"
                        tone="secondary"
                        icon="settings_voice"
                        disabled={isTestingMicrophone}
                        onClick={testMicrophone}
                        className="w-full md:w-auto"
                      >
                        {isTestingMicrophone ? '測試中...' : '測試麥克風'}
                      </Button>
                    </div>

                    <div className="mt-md flex flex-wrap gap-sm">
                      {microphoneReady ? (
                        <StatusBadge tone="success">
                          <Icon name="check_circle" className="text-[18px]" />
                          麥克風已就緒
                        </StatusBadge>
                      ) : null}
                      {microphoneError ? (
                        <StatusBadge tone="danger">
                          <Icon name="error" className="text-[18px]" />
                          {microphoneError}
                        </StatusBadge>
                      ) : null}
                    </div>
                  </div>
                </div>

                <div className="mt-xl flex flex-wrap items-center justify-between gap-md border-t border-outline-variant pt-md">
                  <Button type="button" tone="ghost" icon="arrow_back" onClick={handleBackToProfile}>
                    上一步
                  </Button>
                  <Button type="submit" disabled={!microphoneReady || isSubmitting}>
                    {isSubmitting ? '建立中...' : '建立面試'}
                  </Button>
                </div>
              </>
            )}
          </Card>
        </form>
      </section>
      </main>
    </>
  )
}
