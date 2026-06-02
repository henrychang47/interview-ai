import { useCallback, useEffect, useMemo, useRef, useState } from 'react'

import { getInterview, playQuestionAudio, uploadAnswerAudio } from '../api/interviews'
import { Button, Card, Icon, StatusBadge } from '../components/ui'
import {
  getQuestionAudio,
  hasQuestionAudioPrefetchCompleted,
  markQuestionAudioPrefetchComplete,
  storeQuestionAudio,
} from '../lib/questionAudioCache'
import type { InterviewDetail } from '../types/interview'

type InterviewSessionPageProps = {
  interviewID: string
}

type SessionPhase =
  | 'loading'
  | 'playing_question'
  | 'recording_answer'
  | 'advancing'
  | 'finishing_uploads'
  | 'blocked'

type UploadQueueItem = {
  questionID: string
  audio: Blob
  attempts: number
  status: 'queued' | 'uploading' | 'uploaded' | 'failed'
  error: string | null
}

function getRecordingErrorMessage(error: unknown) {
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

  return '無法開始錄音'
}

function navigateTo(path: string) {
  window.history.pushState({}, '', path)
  window.dispatchEvent(new PopStateEvent('popstate'))
}

const maxRecordingSeconds = Number(import.meta.env.VITE_MAX_ANSWER_RECORDING_SECONDS ?? 180)
const safeMaxRecordingSeconds =
  Number.isFinite(maxRecordingSeconds) && maxRecordingSeconds > 0 ? maxRecordingSeconds : 180
const preferredChineseVoiceName = 'Google 國語（臺灣）'
const preferredChineseSpeechRate = 1.1
const preferredChineseSpeechPitch = 0.8

function normalizeLanguageTag(language: string) {
  return language.trim().toLowerCase().replace(/_/g, '-')
}

function isChineseLanguage(language: string) {
  return normalizeLanguageTag(language).split('-')[0] === 'zh'
}

function findSpeechVoice(voices: SpeechSynthesisVoice[], preferredLanguage: string) {
  const normalizedPreferredLanguage = normalizeLanguageTag(preferredLanguage || 'zh-TW')
  const preferredLanguageFamily = normalizedPreferredLanguage.split('-')[0]

  return (
    (preferredLanguageFamily === 'zh'
      ? voices.find((voice) => voice.name === preferredChineseVoiceName)
      : undefined) ??
    voices.find((voice) => normalizeLanguageTag(voice.lang) === normalizedPreferredLanguage) ??
    voices.find((voice) => normalizeLanguageTag(voice.lang).split('-')[0] === preferredLanguageFamily) ??
    null
  )
}

export default function InterviewSessionPage({ interviewID }: InterviewSessionPageProps) {
  const [interview, setInterview] = useState<InterviewDetail | null>(null)
  const [currentQuestionIndex, setCurrentQuestionIndex] = useState(0)
  const [phase, setPhase] = useState<SessionPhase>('loading')
  const [error, setError] = useState<string | null>(null)
  const [recordingError, setRecordingError] = useState<string | null>(null)
  const [secondsRemaining, setSecondsRemaining] = useState(safeMaxRecordingSeconds)
  const [uploadQueue, setUploadQueue] = useState<UploadQueueItem[]>([])
  const [speechVoices, setSpeechVoices] = useState<SpeechSynthesisVoice[]>([])
  const mediaRecorderRef = useRef<MediaRecorder | null>(null)
  const mediaStreamRef = useRef<MediaStream | null>(null)
  const recordedChunksRef = useRef<Blob[]>([])
  const discardRecordingRef = useRef(false)
  const autoPlayKeyRef = useRef<string | null>(null)
  const questionAudioRef = useRef<HTMLAudioElement | null>(null)
  const questionAudioURLRef = useRef<string | null>(null)
  const playbackRunRef = useRef(0)

  const questions = interview?.questions ?? []
  const currentQuestion = questions[currentQuestionIndex]
  const canSpeakQuestion =
    typeof window !== 'undefined' &&
    'speechSynthesis' in window &&
    typeof SpeechSynthesisUtterance !== 'undefined'
  const canRecordAnswer =
    typeof navigator !== 'undefined' &&
    Boolean(navigator.mediaDevices?.getUserMedia) &&
    typeof MediaRecorder !== 'undefined'

  const progressPercent = useMemo(() => {
    if (questions.length === 0) {
      return 0
    }
    return ((currentQuestionIndex + 1) / questions.length) * 100
  }, [currentQuestionIndex, questions.length])

  const stopMediaStream = useCallback(() => {
    mediaStreamRef.current?.getTracks().forEach((track) => track.stop())
    mediaStreamRef.current = null
  }, [])

  const stopQuestionPlayback = useCallback(() => {
    playbackRunRef.current += 1
    questionAudioRef.current?.pause()
    questionAudioRef.current = null
    if (questionAudioURLRef.current) {
      URL.revokeObjectURL(questionAudioURLRef.current)
      questionAudioURLRef.current = null
    }
    if (canSpeakQuestion) {
      window.speechSynthesis.cancel()
    }
  }, [canSpeakQuestion])

  const queueAnswerUpload = useCallback(
    (questionID: string, audio: Blob) => {
      setUploadQueue((items) => [
        ...items.filter((item) => item.questionID !== questionID),
        { questionID, audio, attempts: 0, status: 'queued', error: null },
      ])

      if (currentQuestionIndex >= questions.length - 1) {
        setPhase('finishing_uploads')
      } else {
        setPhase('advancing')
        setCurrentQuestionIndex((index) => index + 1)
      }
    },
    [currentQuestionIndex, questions.length],
  )

  const stopAnswerRecording = useCallback(
    ({ discard }: { discard: boolean }) => {
      discardRecordingRef.current = discard
      const recorder = mediaRecorderRef.current
      if (recorder && recorder.state !== 'inactive') {
        recorder.stop()
        return
      }
      stopMediaStream()
      if (discard) {
        setPhase('playing_question')
      }
    },
    [stopMediaStream],
  )

  const startAnswerRecording = useCallback(async () => {
    if (!currentQuestion || !canRecordAnswer) {
      setPhase('blocked')
      setRecordingError('此瀏覽器不支援錄音。')
      return
    }

    setRecordingError(null)
    setSecondsRemaining(safeMaxRecordingSeconds)
    recordedChunksRef.current = []
    discardRecordingRef.current = false

    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
      mediaStreamRef.current = stream
      const recorderOptions =
        typeof MediaRecorder.isTypeSupported === 'function' &&
          MediaRecorder.isTypeSupported('audio/webm')
          ? { mimeType: 'audio/webm' }
          : undefined
      const recorder = new MediaRecorder(stream, recorderOptions)

      recorder.ondataavailable = (event) => {
        if (event.data.size > 0) {
          recordedChunksRef.current.push(event.data)
        }
      }
      recorder.onstop = () => {
        const shouldDiscard = discardRecordingRef.current
        const recordedBlob = new Blob(recordedChunksRef.current, { type: 'audio/webm' })
        mediaRecorderRef.current = null
        recordedChunksRef.current = []
        stopMediaStream()

        if (shouldDiscard) {
          autoPlayKeyRef.current = null
          setPhase('playing_question')
          return
        }

        queueAnswerUpload(currentQuestion.id, recordedBlob)
      }

      mediaRecorderRef.current = recorder
      recorder.start()
      setPhase('recording_answer')
    } catch (error) {
      stopMediaStream()
      setPhase('blocked')
      setRecordingError(getRecordingErrorMessage(error))
    }
  }, [canRecordAnswer, currentQuestion, queueAnswerUpload, stopMediaStream])

  const playGeneratedQuestionAudio = useCallback(
    async (questionID: string) => {
      if (typeof Audio === 'undefined') {
        throw new Error('Audio playback is not supported')
      }

      const cachedAudio = getQuestionAudio(interviewID, questionID)
      if (!cachedAudio && hasQuestionAudioPrefetchCompleted(interviewID)) {
        throw new Error('Prefetched question audio is unavailable')
      }
      const audioBlob = cachedAudio ?? (await playQuestionAudio(interviewID, questionID))
      const audioURL = URL.createObjectURL(audioBlob)
      const audio = new Audio(audioURL)
      questionAudioRef.current = audio
      questionAudioURLRef.current = audioURL

      try {
        await new Promise<void>((resolve, reject) => {
          audio.onended = () => resolve()
          audio.onerror = () => reject(new Error('Question audio playback failed'))
          const playResult = audio.play()
          if (playResult && typeof playResult.catch === 'function') {
            playResult.catch(reject)
          }
        })
      } finally {
        if (questionAudioRef.current === audio) {
          questionAudioRef.current = null
        }
        if (questionAudioURLRef.current === audioURL) {
          URL.revokeObjectURL(audioURL)
          questionAudioURLRef.current = null
        }
      }
    },
    [interviewID],
  )

  const playBrowserSpeechSynthesis = useCallback(
    async (text: string, language: string) => {
      if (!canSpeakQuestion) {
        throw new Error('Speech synthesis is not supported')
      }

      await new Promise<void>((resolve, reject) => {
        const availableVoices = speechVoices.length > 0 ? speechVoices : window.speechSynthesis.getVoices()
        const utterance = new SpeechSynthesisUtterance(text)
        utterance.lang = language
        utterance.voice = findSpeechVoice(availableVoices, language)
        if (isChineseLanguage(language)) {
          utterance.rate = preferredChineseSpeechRate
          utterance.pitch = preferredChineseSpeechPitch
        }
        utterance.onend = () => resolve()
        utterance.onerror = () => reject(new Error('Speech synthesis failed'))

        window.speechSynthesis.speak(utterance)
      })
    },
    [canSpeakQuestion, speechVoices],
  )

  const playCurrentQuestion = useCallback(async () => {
    if (!currentQuestion || !interview) {
      setPhase('blocked')
      return
    }

    stopQuestionPlayback()
    const playbackRun = ++playbackRunRef.current
    setPhase('playing_question')

    try {
      await playGeneratedQuestionAudio(currentQuestion.id)
      if (playbackRun !== playbackRunRef.current) {
        return
      }
      void startAnswerRecording()
      return
    } catch {
      // Browser TTS is the intentional fallback when Gemini TTS is unavailable.
    }

    try {
      await playBrowserSpeechSynthesis(currentQuestion.text, interview.question_language || 'zh-TW')
      if (playbackRun !== playbackRunRef.current) {
        return
      }
      void startAnswerRecording()
    } catch {
      if (playbackRun === playbackRunRef.current) {
        setPhase('blocked')
      }
    }
  }, [
    currentQuestion,
    interview,
    playBrowserSpeechSynthesis,
    playGeneratedQuestionAudio,
    startAnswerRecording,
    stopQuestionPlayback,
  ])

  useEffect(() => {
    if (!canSpeakQuestion) {
      setSpeechVoices([])
      return
    }

    const updateSpeechVoices = () => {
      setSpeechVoices(window.speechSynthesis.getVoices())
    }

    updateSpeechVoices()
    window.speechSynthesis.addEventListener('voiceschanged', updateSpeechVoices)

    return () => {
      window.speechSynthesis.removeEventListener('voiceschanged', updateSpeechVoices)
    }
  }, [canSpeakQuestion])

  useEffect(() => {
    let isMounted = true

    async function loadInterview() {
      setPhase('loading')
      setError(null)

      try {
        const detail = await getInterview(interviewID)
        if (!isMounted) {
          return
        }
        setInterview(detail)
        if (detail.status !== 'in_progress') {
          setPhase('blocked')
          setError('請先從準備頁開始面試。')
          return
        }
        await Promise.allSettled(
          detail.questions.map(async (question) => {
            if (getQuestionAudio(interviewID, question.id)) {
              return
            }
            const audio = await playQuestionAudio(interviewID, question.id)
            if (isMounted) {
              storeQuestionAudio(interviewID, question.id, audio)
            }
          }),
        )
        if (!isMounted) {
          return
        }
        markQuestionAudioPrefetchComplete(interviewID)
        setPhase('advancing')
      } catch (error) {
        if (isMounted) {
          setPhase('blocked')
          setError(error instanceof Error ? error.message : '載入面試失敗')
        }
      }
    }

    loadInterview()

    return () => {
      isMounted = false
    }
  }, [interviewID])

  useEffect(() => {
    if (!currentQuestion || phase === 'loading' || phase === 'blocked' || phase === 'finishing_uploads') {
      return
    }

    const autoPlayKey = `${currentQuestion.id}:${currentQuestionIndex}`
    if (autoPlayKeyRef.current === autoPlayKey) {
      return
    }
    autoPlayKeyRef.current = autoPlayKey
    playCurrentQuestion()
  }, [currentQuestion, currentQuestionIndex, phase, playCurrentQuestion])

  useEffect(() => {
    if (phase !== 'recording_answer') {
      return
    }

    const intervalID = window.setInterval(() => {
      setSecondsRemaining((seconds) => {
        if (seconds <= 1) {
          window.clearInterval(intervalID)
          stopAnswerRecording({ discard: false })
          return 0
        }
        return seconds - 1
      })
    }, 1000)

    return () => window.clearInterval(intervalID)
  }, [phase, stopAnswerRecording])

  useEffect(() => {
    const nextItem = uploadQueue.find((item) => item.status === 'queued')
    if (!nextItem) {
      return
    }
    const itemToUpload = nextItem

    async function upload() {
      setUploadQueue((items) =>
        items.map((item) =>
          item.questionID === itemToUpload.questionID
            ? { ...item, status: 'uploading', attempts: item.attempts + 1, error: null }
            : item,
        ),
      )

      try {
        await uploadAnswerAudio(interviewID, itemToUpload.questionID, itemToUpload.audio)
        setUploadQueue((items) =>
          items.map((item) =>
            item.questionID === itemToUpload.questionID
              ? { ...item, status: 'uploaded', error: null }
              : item,
          ),
        )
      } catch (error) {
        setUploadQueue((items) =>
          items.map((item) =>
            item.questionID === itemToUpload.questionID
              ? {
                ...item,
                status: item.attempts + 1 >= 3 ? 'failed' : 'queued',
                error: error instanceof Error ? error.message : '上傳回答失敗',
              }
              : item,
          ),
        )
      }
    }

    upload()
  }, [interviewID, uploadQueue])

  useEffect(() => {
    if (phase !== 'finishing_uploads' || uploadQueue.length === 0) {
      return
    }
    if (uploadQueue.every((item) => item.status === 'uploaded')) {
      window.setTimeout(() => navigateTo(`/interviews/${interviewID}/result`), 0)
    }
  }, [interviewID, phase, uploadQueue])

  useEffect(() => {
    return () => {
      stopQuestionPlayback()
      discardRecordingRef.current = true
      const recorder = mediaRecorderRef.current
      if (recorder && recorder.state !== 'inactive') {
        recorder.stop()
      } else {
        stopMediaStream()
      }
    }
  }, [stopMediaStream, stopQuestionPlayback])

  const failedUpload = uploadQueue.find((item) => item.status === 'failed')
  const phaseLabel =
    phase === 'playing_question'
      ? '正在播放題目'
      : phase === 'recording_answer'
        ? '正在錄音'
        : phase === 'advancing'
          ? '準備下一題'
          : phase === 'finishing_uploads'
            ? '正在完成面試'
            : phase === 'blocked'
              ? '面試暫停'
              : '載入面試中...'
  const minutes = Math.floor(secondsRemaining / 60)
  const seconds = secondsRemaining % 60
  const timerLabel = `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`

  return (
    <main className="flex min-h-screen flex-col bg-surface text-on-surface">
      <header className="border-b border-outline-variant bg-surface px-margin-mobile py-md md:px-margin-desktop md:py-lg">
        <div className="mx-auto flex w-full max-w-container-max items-center justify-between gap-md">
          <div className="min-w-0">
            <h1 className="truncate font-headline text-headline-sm text-on-surface md:text-headline-md">
              {interview?.job_title ?? '模擬面試進行中'}
            </h1>
            {currentQuestion ? (
              <p className="mt-xs text-body-sm text-on-surface-variant">
                Question {currentQuestionIndex + 1} of {questions.length}
              </p>
            ) : null}
          </div>
          <a
            href={`/interviews/${interviewID}`}
            className="flex shrink-0 items-center gap-xs rounded-lg px-sm py-xs text-label-md font-bold text-on-surface-variant hover:bg-surface-container-low hover:text-primary"
          >
            <Icon name="close" />
            <span className="hidden md:inline">結束面試</span>
          </a>
        </div>
        <div className="mx-auto mt-sm w-full max-w-container-max">
          <div className="h-1.5 w-full overflow-hidden rounded-full bg-surface-container-highest">
            <div
              className="h-full rounded-full bg-primary transition-all duration-300"
              style={{ width: `${progressPercent}%` }}
            />
          </div>
        </div>
      </header>

      <section className="relative flex flex-1 flex-col items-center justify-center px-margin-mobile py-xl md:px-margin-desktop">
        {phase === 'loading' ? <p className="text-body-md text-on-surface-variant">載入面試中...</p> : null}

        {error ? (
          <Card className="mb-lg flex w-full max-w-2xl items-start gap-sm border-error/20 bg-error-container p-md text-on-error-container">
            <Icon name="warning" className="mt-xs" />
            <p className="text-body-sm">{error}</p>
          </Card>
        ) : null}

        {interview && phase !== 'loading' ? (
          <>
            {currentQuestion ? (
              <div className="flex w-full max-w-3xl flex-col items-center text-center">
                <StatusBadge tone={phase === 'recording_answer' ? 'primary' : 'neutral'} className="mb-lg">
                  <Icon
                    name={phase === 'recording_answer' ? 'mic' : phase === 'playing_question' ? 'volume_up' : 'hourglass_empty'}
                    filled={phase === 'recording_answer'}
                  />
                  {phaseLabel}
                </StatusBadge>

                <div className="mb-xl">
                  <div className="font-headline text-headline-xl text-on-surface md:text-[64px] md:leading-[72px]">
                    {phase === 'recording_answer' ? timerLabel : `Q${currentQuestion.order}`}
                  </div>
                  <p className="mt-xs text-body-sm text-on-surface-variant">
                    {phase === 'recording_answer' ? '剩餘時間' : `第 ${currentQuestionIndex + 1} 題 / 共 ${questions.length} 題`}
                  </p>
                </div>

                <div className="flex w-full max-w-2xl flex-col items-center justify-center gap-md md:flex-row">
                  <Button
                    type="button"
                    icon="stop_circle"
                    onClick={() => stopAnswerRecording({ discard: false })}
                    disabled={phase !== 'recording_answer'}
                    className="w-full md:w-auto"
                  >
                    回答結束
                  </Button>
                  <Button
                    type="button"
                    tone="secondary"
                    icon="restart_alt"
                    onClick={() => stopAnswerRecording({ discard: true })}
                    disabled={phase !== 'recording_answer'}
                    className="w-full md:w-auto"
                  >
                    重新播放題目
                  </Button>
                </div>

                <div className="mt-lg w-full max-w-2xl space-y-sm text-left">
                  {recordingError ? (
                    <Card className="border-error/20 bg-error-container p-md text-body-sm text-on-error-container">
                      {recordingError}
                    </Card>
                  ) : null}
                  {!canSpeakQuestion ? (
                    <Card className="p-md text-body-sm text-on-surface-variant">
                      此瀏覽器不支援題目朗讀。
                    </Card>
                  ) : null}
                  {!canRecordAnswer ? (
                    <Card className="p-md text-body-sm text-on-surface-variant">
                      此瀏覽器不支援錄音。
                    </Card>
                  ) : null}
                  {failedUpload ? (
                    <Card className="border-error/20 bg-error-container p-md text-on-error-container">
                      <div className="flex flex-col gap-md md:flex-row md:items-center md:justify-between">
                        <p className="text-body-sm">{failedUpload.error}</p>
                        <div className="flex flex-wrap gap-sm">
                          <Button
                            type="button"
                            tone="danger"
                            onClick={() =>
                              setUploadQueue((items) =>
                                items.map((item) =>
                                  item.status === 'failed'
                                    ? { ...item, status: 'queued', error: null }
                                    : item,
                                ),
                              )
                            }
                          >
                            重試上傳
                          </Button>
                          <Button
                            type="button"
                            tone="secondary"
                            onClick={() => {
                              const questionIndex = questions.findIndex(
                                (question) => question.id === failedUpload.questionID,
                              )
                              if (questionIndex >= 0) {
                                setCurrentQuestionIndex(questionIndex)
                                setUploadQueue((items) =>
                                  items.filter((item) => item.questionID !== failedUpload.questionID),
                                )
                                setPhase('playing_question')
                              }
                            }}
                          >
                            重新回答本題
                          </Button>
                        </div>
                      </div>
                    </Card>
                  ) : null}
                </div>
              </div>
            ) : (
              <Card className="w-full max-w-2xl border-amber-200 bg-amber-50 p-md text-body-sm text-amber-800">
                這場面試目前沒有題目。
              </Card>
            )}
          </>
        ) : null}
      </section>
    </main>
  )
}
