import { useEffect, useMemo, useRef, useState } from 'react'

import { getInterview, uploadAnswerAudio } from '../api/interviews'
import type { Answer, InterviewDetail } from '../types/interview'

type InterviewSessionPageProps = {
  interviewID: string
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

export default function InterviewSessionPage({ interviewID }: InterviewSessionPageProps) {
  const [interview, setInterview] = useState<InterviewDetail | null>(null)
  const [currentQuestionIndex, setCurrentQuestionIndex] = useState(0)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [isPlayingQuestion, setIsPlayingQuestion] = useState(false)
  const [isRecordingAnswer, setIsRecordingAnswer] = useState(false)
  const [recordedAnswerBlob, setRecordedAnswerBlob] = useState<Blob | null>(null)
  const [recordedAnswerURL, setRecordedAnswerURL] = useState<string | null>(null)
  const [uploadedAnswersByQuestionID, setUploadedAnswersByQuestionID] = useState<
    Record<string, Answer>
  >({})
  const [isUploadingAnswer, setIsUploadingAnswer] = useState(false)
  const [uploadError, setUploadError] = useState<string | null>(null)
  const [recordingError, setRecordingError] = useState<string | null>(null)
  const mediaRecorderRef = useRef<MediaRecorder | null>(null)
  const mediaStreamRef = useRef<MediaStream | null>(null)
  const recordedChunksRef = useRef<Blob[]>([])
  const recordedAnswerURLRef = useRef<string | null>(null)
  const shouldCreateRecordedPreviewRef = useRef(true)

  useEffect(() => {
    let isMounted = true

    async function loadInterview() {
      setIsLoading(true)
      setError(null)

      try {
        const detail = await getInterview(interviewID)
        if (isMounted) {
          const uploadedAnswers = detail.answers.reduce<Record<string, Answer>>(
            (answersByQuestionID, answer) => {
              answersByQuestionID[answer.question_id] = answer
              return answersByQuestionID
            },
            {},
          )

          setInterview(detail)
          setUploadedAnswersByQuestionID(uploadedAnswers)
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
  const currentUploadedAnswer = currentQuestion
    ? uploadedAnswersByQuestionID[currentQuestion.id]
    : undefined
  const canUploadCurrentAnswer = Boolean(currentQuestion && recordedAnswerBlob && !isUploadingAnswer)
  const canMoveToNextQuestion = Boolean(currentQuestion && currentUploadedAnswer && !isUploadingAnswer)

  const progressPercent = useMemo(() => {
    if (questions.length === 0) {
      return 0
    }
    return ((currentQuestionIndex + 1) / questions.length) * 100
  }, [currentQuestionIndex, questions.length])

  const canSpeakQuestion =
    typeof window !== 'undefined' &&
    'speechSynthesis' in window &&
    typeof SpeechSynthesisUtterance !== 'undefined'

  const canRecordAnswer =
    typeof navigator !== 'undefined' &&
    Boolean(navigator.mediaDevices?.getUserMedia) &&
    typeof MediaRecorder !== 'undefined'

  function playCurrentQuestion() {
    if (!currentQuestion || !canSpeakQuestion) {
      return
    }

    window.speechSynthesis.cancel()

    const utterance = new SpeechSynthesisUtterance(currentQuestion.text)
    utterance.lang = 'zh-TW'
    utterance.onend = () => setIsPlayingQuestion(false)
    utterance.onerror = () => setIsPlayingQuestion(false)

    setIsPlayingQuestion(true)
    window.speechSynthesis.speak(utterance)
  }

  function stopQuestionPlayback() {
    if (canSpeakQuestion) {
      window.speechSynthesis.cancel()
    }
    setIsPlayingQuestion(false)
  }

  function revokeRecordedAnswerURL() {
    if (recordedAnswerURLRef.current) {
      URL.revokeObjectURL(recordedAnswerURLRef.current)
      recordedAnswerURLRef.current = null
    }
    setRecordedAnswerURL(null)
    setRecordedAnswerBlob(null)
    setUploadError(null)
  }

  function revokeRecordedAnswerURLOnUnmount() {
    if (recordedAnswerURLRef.current) {
      URL.revokeObjectURL(recordedAnswerURLRef.current)
      recordedAnswerURLRef.current = null
    }
  }

  function stopMediaStream() {
    mediaStreamRef.current?.getTracks().forEach((track) => track.stop())
    mediaStreamRef.current = null
  }

  async function startAnswerRecording() {
    if (!canRecordAnswer || isRecordingAnswer) {
      return
    }

    stopQuestionPlayback()
    revokeRecordedAnswerURL()
    setRecordingError(null)
    recordedChunksRef.current = []
    shouldCreateRecordedPreviewRef.current = true

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
        if (shouldCreateRecordedPreviewRef.current) {
          const recordedBlob = new Blob(recordedChunksRef.current, { type: 'audio/webm' })
          const recordedURL = URL.createObjectURL(recordedBlob)
          recordedAnswerURLRef.current = recordedURL
          setRecordedAnswerBlob(recordedBlob)
          setRecordedAnswerURL(recordedURL)
          setUploadError(null)
        }
        setIsRecordingAnswer(false)
        stopMediaStream()
      }

      mediaRecorderRef.current = recorder
      recorder.start()
      setIsRecordingAnswer(true)
    } catch (error) {
      setIsRecordingAnswer(false)
      stopMediaStream()
      setRecordingError(getRecordingErrorMessage(error))
    }
  }

  async function uploadCurrentAnswer() {
    if (!currentQuestion || !recordedAnswerBlob || isUploadingAnswer) {
      return
    }

    setIsUploadingAnswer(true)
    setUploadError(null)

    try {
      const uploadedAnswer = await uploadAnswerAudio(interviewID, currentQuestion.id, recordedAnswerBlob)
      setUploadedAnswersByQuestionID((answersByQuestionID) => ({
        ...answersByQuestionID,
        [uploadedAnswer.question_id]: {
          id: uploadedAnswer.id,
          question_id: uploadedAnswer.question_id,
          audio_path: uploadedAnswer.audio_path,
          transcript_text: uploadedAnswer.transcript_text,
          created_at: new Date().toISOString(),
        },
      }))
    } catch (error) {
      setUploadError(error instanceof Error ? error.message : '上傳回答失敗')
    } finally {
      setIsUploadingAnswer(false)
    }
  }

  function stopAnswerRecording() {
    const recorder = mediaRecorderRef.current
    if (!recorder || recorder.state === 'inactive') {
      return
    }

    shouldCreateRecordedPreviewRef.current = true
    recorder.stop()
  }

  function resetAnswerRecording() {
    shouldCreateRecordedPreviewRef.current = false
    const recorder = mediaRecorderRef.current
    if (recorder && recorder.state !== 'inactive') {
      recorder.stop()
    } else {
      stopMediaStream()
    }

    mediaRecorderRef.current = null
    recordedChunksRef.current = []
    setIsRecordingAnswer(false)
    setRecordingError(null)
    revokeRecordedAnswerURL()
  }

  useEffect(() => {
    return () => {
      shouldCreateRecordedPreviewRef.current = false
      const recorder = mediaRecorderRef.current
      if (recorder && recorder.state !== 'inactive') {
        recorder.stop()
      } else {
        mediaStreamRef.current?.getTracks().forEach((track) => track.stop())
      }
      revokeRecordedAnswerURLOnUnmount()
    }
  }, [])

  useEffect(() => {
    return () => {
      if (
        typeof window !== 'undefined' &&
        'speechSynthesis' in window &&
        typeof SpeechSynthesisUtterance !== 'undefined'
      ) {
        window.speechSynthesis.cancel()
      }
    }
  }, [])

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

                <div className="mt-5">
                  <button
                    type="button"
                    onClick={playCurrentQuestion}
                    disabled={!canSpeakQuestion || isPlayingQuestion}
                    className="min-h-11 rounded-md border border-teal-700 bg-white px-5 py-2 text-sm font-semibold text-teal-800 hover:bg-teal-50 disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    {isPlayingQuestion ? '朗讀中' : '朗讀題目'}
                  </button>
                  {!canSpeakQuestion ? (
                    <p className="mt-2 text-sm text-slate-600">此瀏覽器不支援題目朗讀。</p>
                  ) : null}
                </div>

                <div className="mt-5 rounded-md border border-slate-200 bg-white p-5">
                  <div className="flex flex-col gap-3 sm:flex-row">
                    <button
                      type="button"
                      onClick={startAnswerRecording}
                      disabled={!canRecordAnswer || isRecordingAnswer}
                      className="min-h-11 rounded-md bg-teal-700 px-5 py-2 text-sm font-semibold text-white hover:bg-teal-800 disabled:cursor-not-allowed disabled:opacity-50"
                    >
                      {isRecordingAnswer ? '錄音中' : '開始錄音'}
                    </button>
                    <button
                      type="button"
                      onClick={stopAnswerRecording}
                      disabled={!isRecordingAnswer}
                      className="min-h-11 rounded-md border border-slate-300 bg-white px-5 py-2 text-sm font-semibold text-slate-800 hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-50"
                    >
                      停止錄音
                    </button>
                  </div>
                  {!canRecordAnswer ? (
                    <p className="mt-3 text-sm text-slate-600">此瀏覽器不支援錄音。</p>
                  ) : null}
                  {recordingError ? (
                    <p className="mt-3 text-sm text-red-700">{recordingError}</p>
                  ) : null}
                  {recordedAnswerURL ? (
                    <div className="mt-4">
                      <p className="mb-2 text-sm font-medium text-slate-700">回答錄音預覽</p>
                      <audio aria-label="回答錄音預覽" controls src={recordedAnswerURL} />
                    </div>
                  ) : null}
                  {recordedAnswerURL ? (
                    <div className="mt-4 flex flex-col gap-3 sm:flex-row sm:items-center">
                      <button
                        type="button"
                        onClick={uploadCurrentAnswer}
                        disabled={!canUploadCurrentAnswer || Boolean(currentUploadedAnswer)}
                        className="min-h-11 rounded-md bg-teal-700 px-5 py-2 text-sm font-semibold text-white hover:bg-teal-800 disabled:cursor-not-allowed disabled:opacity-50"
                      >
                        {isUploadingAnswer ? '上傳中' : '上傳本題回答'}
                      </button>
                    </div>
                  ) : null}
                  {currentUploadedAnswer ? (
                    <p className="mt-3 text-sm font-medium text-teal-700">本題回答已上傳</p>
                  ) : null}
                  {uploadError ? <p className="mt-3 text-sm text-red-700">{uploadError}</p> : null}
                </div>

                <div className="mt-6 flex flex-col gap-3 sm:flex-row sm:justify-between">
                  <button
                    type="button"
                    onClick={() => {
                      stopQuestionPlayback()
                      resetAnswerRecording()
                      setCurrentQuestionIndex((index) => Math.max(index - 1, 0))
                    }}
                    disabled={isFirstQuestion}
                    className="min-h-11 rounded-md border border-slate-300 bg-white px-5 py-2 text-sm font-semibold text-slate-800 hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    上一題
                  </button>
                  <button
                    type="button"
                    onClick={() => {
                      if (!canMoveToNextQuestion) {
                        return
                      }
                      stopQuestionPlayback()
                      resetAnswerRecording()
                      if (isLastQuestion) {
                        window.history.pushState({}, '', `/interviews/${interviewID}/result`)
                        window.dispatchEvent(new PopStateEvent('popstate'))
                        return
                      }
                      setCurrentQuestionIndex((index) => Math.min(index + 1, questions.length - 1))
                    }}
                    disabled={!canMoveToNextQuestion}
                    className="min-h-11 rounded-md bg-teal-700 px-5 py-2 text-sm font-semibold text-white hover:bg-teal-800 disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    {isLastQuestion ? '完成面試' : '下一題'}
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
