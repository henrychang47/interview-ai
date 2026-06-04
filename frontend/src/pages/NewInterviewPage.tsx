import { FormEvent, MouseEvent, useEffect, useRef, useState } from 'react'

import { createInterview } from '../api/interviews'
import { AudioPlayer } from '../components/AudioPlayer'
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

const JOB_TITLE_MAX_LENGTH = 50
const LONG_TEXT_MAX_LENGTH = 4000

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
  const [isRecordingMicrophoneTest, setIsRecordingMicrophoneTest] = useState(false)
  const [microphoneReady, setMicrophoneReady] = useState(false)
  const [microphoneError, setMicrophoneError] = useState<string | null>(null)
  const [microphonePreviewURL, setMicrophonePreviewURL] = useState<string | null>(null)
  const microphoneRecorderRef = useRef<MediaRecorder | null>(null)
  const microphoneStreamRef = useRef<MediaStream | null>(null)
  const microphoneChunksRef = useRef<Blob[]>([])
  const isProfileComplete =
    form.job_title.trim() !== '' &&
    form.job_description.trim() !== '' &&
    form.user_profile.trim() !== ''

  function stopMicrophoneStream() {
    microphoneStreamRef.current?.getTracks().forEach((track) => track.stop())
    microphoneStreamRef.current = null
  }

  function clearMicrophonePreview() {
    setMicrophonePreviewURL((currentURL) => {
      if (currentURL) {
        URL.revokeObjectURL(currentURL)
      }
      return null
    })
  }

  useEffect(() => {
    return () => {
      if (microphoneRecorderRef.current?.state === 'recording') {
        microphoneRecorderRef.current.stop()
      }
      stopMicrophoneStream()
      if (microphonePreviewURL) {
        URL.revokeObjectURL(microphonePreviewURL)
      }
    }
  }, [microphonePreviewURL])

  async function testMicrophone() {
    if (!navigator.mediaDevices?.getUserMedia || typeof MediaRecorder === 'undefined') {
      setMicrophoneReady(false)
      setMicrophoneError('此瀏覽器不支援麥克風錄音，請改用支援 MediaRecorder 的瀏覽器。')
      return
    }

    setIsTestingMicrophone(true)
    setMicrophoneError(null)
    setMicrophoneReady(false)
    clearMicrophonePreview()
    microphoneChunksRef.current = []

    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
      microphoneStreamRef.current = stream
      const recorderOptions =
        typeof MediaRecorder.isTypeSupported === 'function' &&
          MediaRecorder.isTypeSupported('audio/webm')
          ? { mimeType: 'audio/webm' }
          : undefined
      const recorder = new MediaRecorder(stream, recorderOptions)

      recorder.ondataavailable = (event) => {
        if (event.data.size > 0) {
          microphoneChunksRef.current.push(event.data)
        }
      }

      recorder.onstop = () => {
        const recordedBlob = new Blob(microphoneChunksRef.current, { type: 'audio/webm' })
        stopMicrophoneStream()
        microphoneRecorderRef.current = null
        setIsRecordingMicrophoneTest(false)
        setIsTestingMicrophone(false)

        if (recordedBlob.size === 0) {
          setMicrophoneReady(false)
          setMicrophoneError('沒有錄到聲音，請再試一次。')
          return
        }

        setMicrophonePreviewURL(URL.createObjectURL(recordedBlob))
        setMicrophoneReady(true)
      }

      microphoneRecorderRef.current = recorder
      recorder.start()
      setIsRecordingMicrophoneTest(true)
    } catch (error) {
      setMicrophoneReady(false)
      setMicrophoneError(getMicrophoneErrorMessage(error))
      stopMicrophoneStream()
      setIsTestingMicrophone(false)
    }
  }

  function stopMicrophoneTest() {
    if (microphoneRecorderRef.current?.state === 'recording') {
      microphoneRecorderRef.current.stop()
      return
    }

    stopMicrophoneStream()
    setIsRecordingMicrophoneTest(false)
    setIsTestingMicrophone(false)
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
    stopMicrophoneTest()
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
                        maxLength={JOB_TITLE_MAX_LENGTH}
                        value={form.job_title}
                        onChange={(event) =>
                          setForm((current) => ({
                            ...current,
                            job_title: event.target.value.slice(0, JOB_TITLE_MAX_LENGTH),
                          }))
                        }
                      />
                    </label>

                    <div className="block">
                      <label
                        htmlFor="job_description"
                        className="mb-sm block text-label-md font-semibold text-on-surface"
                      >
                        職位要求及說明
                      </label>
                      <textarea
                        id="job_description"
                        className="focus-calm min-h-32 w-full rounded-lg border border-outline-variant bg-surface-container-lowest p-3 text-body-md text-on-surface placeholder:text-on-surface-variant"
                        placeholder="貼上職缺描述或條列主要技能需求..."
                        required
                        maxLength={LONG_TEXT_MAX_LENGTH}
                        value={form.job_description}
                        onChange={(event) =>
                          setForm((current) => ({
                            ...current,
                            job_description: event.target.value.slice(0, LONG_TEXT_MAX_LENGTH),
                          }))
                        }
                      />
                      <span className="mt-xs block text-right text-label-sm text-on-surface-variant">
                        {form.job_description.length}/{LONG_TEXT_MAX_LENGTH}
                      </span>
                    </div>

                    <div className="block">
                      <label
                        htmlFor="user_profile"
                        className="mb-sm block text-label-md font-semibold text-on-surface"
                      >
                        個人資訊
                      </label>
                      <textarea
                        id="user_profile"
                        className="focus-calm min-h-32 w-full rounded-lg border border-outline-variant bg-surface-container-lowest p-3 text-body-md text-on-surface placeholder:text-on-surface-variant"
                        placeholder="簡述您的相關經驗或貼上履歷摘要..."
                        required
                        maxLength={LONG_TEXT_MAX_LENGTH}
                        value={form.user_profile}
                        onChange={(event) =>
                          setForm((current) => ({
                            ...current,
                            user_profile: event.target.value.slice(0, LONG_TEXT_MAX_LENGTH),
                          }))
                        }
                      />
                      <span className="mt-xs block text-right text-label-sm text-on-surface-variant">
                        {form.user_profile.length}/{LONG_TEXT_MAX_LENGTH}
                      </span>
                    </div>
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
                          max={5}
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
                                className={`flex flex-1 cursor-pointer items-center justify-center gap-sm rounded-lg px-4 py-2.5 transition-all ${checked
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
                              錄製一小段語音並播放確認收音效果
                            </p>
                          </div>
                        </div>
                        <Button
                          type="button"
                          tone={isRecordingMicrophoneTest ? 'danger' : 'secondary'}
                          icon={isRecordingMicrophoneTest ? 'stop_circle' : 'settings_voice'}
                          disabled={isTestingMicrophone && !isRecordingMicrophoneTest}
                          onClick={isRecordingMicrophoneTest ? stopMicrophoneTest : testMicrophone}
                          className="w-full md:w-auto"
                        >
                          {isRecordingMicrophoneTest
                            ? '停止錄音'
                            : isTestingMicrophone
                              ? '準備錄音...'
                              : microphonePreviewURL
                                ? '重新測試'
                                : '測試麥克風'}
                        </Button>
                      </div>

                      {microphonePreviewURL ? (
                        <div className="mt-md rounded-lg border border-outline-variant bg-surface-container-lowest p-md">
                          <AudioPlayer
                            src={microphonePreviewURL}
                            label="麥克風測試錄音預覽"
                            title="測試錄音預覽"
                          />
                        </div>
                      ) : null}

                      <div className="mt-md flex flex-wrap gap-sm">
                        {isRecordingMicrophoneTest ? (
                          <StatusBadge tone="primary">
                            <Icon name="graphic_eq" className="text-[18px]" />
                            錄音中，請說一小段話
                          </StatusBadge>
                        ) : null}
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
