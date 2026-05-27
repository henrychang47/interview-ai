import { useCallback, useEffect, useMemo, useRef, useState } from 'react'

import { getInterview, uploadAnswerAudio } from '../api/interviews'
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

export default function InterviewSessionPage({ interviewID }: InterviewSessionPageProps) {
  const [interview, setInterview] = useState<InterviewDetail | null>(null)
  const [currentQuestionIndex, setCurrentQuestionIndex] = useState(0)
  const [phase, setPhase] = useState<SessionPhase>('loading')
  const [error, setError] = useState<string | null>(null)
  const [recordingError, setRecordingError] = useState<string | null>(null)
  const [secondsRemaining, setSecondsRemaining] = useState(safeMaxRecordingSeconds)
  const [uploadQueue, setUploadQueue] = useState<UploadQueueItem[]>([])
  const mediaRecorderRef = useRef<MediaRecorder | null>(null)
  const mediaStreamRef = useRef<MediaStream | null>(null)
  const recordedChunksRef = useRef<Blob[]>([])
  const discardRecordingRef = useRef(false)
  const autoPlayKeyRef = useRef<string | null>(null)

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

  const playCurrentQuestion = useCallback(() => {
    if (!currentQuestion || !interview || !canSpeakQuestion) {
      setPhase('blocked')
      return
    }

    stopQuestionPlayback()
    setPhase('playing_question')

    const utterance = new SpeechSynthesisUtterance(currentQuestion.text)
    utterance.lang = interview.question_language || 'zh-TW'
    utterance.onend = () => {
      void startAnswerRecording()
    }
    utterance.onerror = () => {
      setPhase('blocked')
    }

    window.speechSynthesis.speak(utterance)
  }, [canSpeakQuestion, currentQuestion, interview, startAnswerRecording, stopQuestionPlayback])

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

  return (
    <main className="min-h-screen bg-slate-50 text-slate-950">
      <section className="mx-auto w-full max-w-4xl px-6 py-10">
        <a
          href={`/interviews/${interviewID}`}
          className="text-sm font-medium text-teal-700 hover:text-teal-800"
        >
          返回面試詳情
        </a>

        {phase === 'loading' ? <p className="mt-8 text-slate-600">載入面試中...</p> : null}

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

                <div className="mt-8 rounded-md border border-slate-200 bg-white p-6">
                  {phase === 'playing_question' ? (
                    <h2 className="text-xl font-semibold text-slate-900">正在播放題目</h2>
                  ) : null}
                  {phase === 'recording_answer' ? (
                    <>
                      <h2 className="text-xl font-semibold text-slate-900">正在錄音</h2>
                      <p className="mt-2 text-sm text-slate-600">
                        剩餘 {secondsRemaining} 秒
                      </p>
                    </>
                  ) : null}
                  {phase === 'advancing' ? (
                    <h2 className="text-xl font-semibold text-slate-900">準備下一題</h2>
                  ) : null}
                  {phase === 'finishing_uploads' ? (
                    <h2 className="text-xl font-semibold text-slate-900">正在完成面試</h2>
                  ) : null}
                  {phase === 'blocked' && !error ? (
                    <h2 className="text-xl font-semibold text-slate-900">面試暫停</h2>
                  ) : null}

                  {recordingError ? (
                    <p className="mt-3 text-sm text-red-700">{recordingError}</p>
                  ) : null}
                  {!canSpeakQuestion ? (
                    <p className="mt-3 text-sm text-slate-600">此瀏覽器不支援題目朗讀。</p>
                  ) : null}
                  {!canRecordAnswer ? (
                    <p className="mt-3 text-sm text-slate-600">此瀏覽器不支援錄音。</p>
                  ) : null}
                  {failedUpload ? (
                    <p className="mt-3 text-sm text-red-700">{failedUpload.error}</p>
                  ) : null}
                </div>

                <div className="mt-6 flex flex-wrap gap-3">
                  <button
                    type="button"
                    onClick={() => stopAnswerRecording({ discard: false })}
                    disabled={phase !== 'recording_answer'}
                    className="min-h-11 rounded-md bg-teal-700 px-5 py-2 text-sm font-semibold text-white hover:bg-teal-800 disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    回答結束
                  </button>
                  <button
                    type="button"
                    onClick={() => stopAnswerRecording({ discard: true })}
                    disabled={phase !== 'recording_answer'}
                    className="min-h-11 rounded-md border border-slate-300 bg-white px-5 py-2 text-sm font-semibold text-slate-800 hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    重新播放題目
                  </button>
                  {failedUpload ? (
                    <>
                      <button
                        type="button"
                        onClick={() =>
                          setUploadQueue((items) =>
                            items.map((item) =>
                              item.status === 'failed'
                                ? { ...item, status: 'queued', error: null }
                                : item,
                            ),
                          )
                        }
                        className="min-h-11 rounded-md border border-teal-700 px-5 py-2 text-sm font-semibold text-teal-800 hover:bg-teal-50"
                      >
                        重試上傳
                      </button>
                      <button
                        type="button"
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
                        className="min-h-11 rounded-md border border-slate-300 px-5 py-2 text-sm font-semibold text-slate-800 hover:bg-slate-100"
                      >
                        重新回答本題
                      </button>
                    </>
                  ) : null}
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
