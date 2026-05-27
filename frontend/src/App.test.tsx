import '@testing-library/jest-dom/vitest'
import { act, cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import App from './App'

function mockPathname(pathname: string) {
  window.history.pushState({}, '', pathname)
}

function mockFetchOnce(response: unknown, init: ResponseInit = {}) {
  return vi.fn().mockResolvedValue(
    new Response(JSON.stringify(response), {
      status: init.status ?? 200,
      headers: { 'Content-Type': 'application/json' },
    }),
  )
}

type MockUtterance = {
  text: string
  lang: string
  onend: (() => void) | null
  onerror: (() => void) | null
}

function installSpeechSynthesisMock() {
  const speak = vi.fn()
  const cancel = vi.fn()
  const utterances: MockUtterance[] = []

  class MockSpeechSynthesisUtterance {
    text: string
    lang = ''
    onend: (() => void) | null = null
    onerror: (() => void) | null = null

    constructor(text: string) {
      this.text = text
      utterances.push(this)
    }
  }

  vi.stubGlobal('speechSynthesis', {
    speak,
    cancel,
  })
  vi.stubGlobal('SpeechSynthesisUtterance', MockSpeechSynthesisUtterance)

  return { speak, cancel, utterances }
}

type MockMediaRecorderInstance = {
  stream: MediaStream
  state: 'inactive' | 'recording'
  ondataavailable: ((event: BlobEvent) => void) | null
  onstop: (() => void) | null
  start: ReturnType<typeof vi.fn>
  stop: ReturnType<typeof vi.fn>
}

function installMediaRecorderMock() {
  const trackStop = vi.fn()
  const stream = {
    getTracks: () => [{ stop: trackStop }],
  } as unknown as MediaStream
  const getUserMedia = vi.fn().mockResolvedValue(stream)
  const recorders: MockMediaRecorderInstance[] = []

  class MockMediaRecorder {
    stream: MediaStream
    state: 'inactive' | 'recording' = 'inactive'
    ondataavailable: ((event: BlobEvent) => void) | null = null
    onstop: (() => void) | null = null

    constructor(stream: MediaStream) {
      this.stream = stream
      recorders.push(this as unknown as MockMediaRecorderInstance)
    }

    start = vi.fn(() => {
      this.state = 'recording'
    })

    stop = vi.fn(() => {
      this.state = 'inactive'
      this.ondataavailable?.({
        data: new Blob(['recorded-answer'], { type: 'audio/webm' }),
      } as BlobEvent)
      this.onstop?.()
    })

    static isTypeSupported(type: string) {
      return type === 'audio/webm'
    }
  }

  Object.defineProperty(navigator, 'mediaDevices', {
    configurable: true,
    value: { getUserMedia },
  })
  vi.stubGlobal('MediaRecorder', MockMediaRecorder)

  return { getUserMedia, recorders, stream, trackStop }
}

function installObjectURLMock() {
  const createObjectURL = vi.fn(() => 'blob:recorded-answer')
  const revokeObjectURL = vi.fn()

  Object.defineProperty(URL, 'createObjectURL', {
    configurable: true,
    value: createObjectURL,
  })
  Object.defineProperty(URL, 'revokeObjectURL', {
    configurable: true,
    value: revokeObjectURL,
  })

  return { createObjectURL, revokeObjectURL }
}

describe('App', () => {
  beforeEach(() => {
    mockPathname('/')
    vi.restoreAllMocks()
  })

  afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
    window.history.pushState({}, '', '/')
  })

  it('renders the interview practice homepage', () => {
    render(<App />)

    expect(screen.getByRole('heading', { name: '模擬面試應用' })).toBeInTheDocument()
    expect(screen.getByText('建立面試、產生題目、錄音回答，逐步打通 MVP 主流程。')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: '建立新的模擬面試' })).toHaveAttribute(
      'href',
      '/interviews/new',
    )
  })

  it('renders the create interview form at /interviews/new', () => {
    mockPathname('/interviews/new')

    render(<App />)

    expect(screen.getByRole('heading', { name: '建立模擬面試' })).toBeInTheDocument()
    expect(screen.getByLabelText('職位名稱')).toBeInTheDocument()
    expect(screen.getByLabelText('職位要求及說明')).toBeInTheDocument()
    expect(screen.getByLabelText('個人資訊')).toBeInTheDocument()
    expect(screen.getByLabelText('題目數量')).toHaveValue(3)
  })

  it('submits the create interview form and navigates to detail route', async () => {
    mockPathname('/interviews/new')
    const fetchMock = mockFetchOnce({ id: 'interview-123', status: 'questions_ready' })
    vi.stubGlobal('fetch', fetchMock)
    const pushState = vi.spyOn(window.history, 'pushState')

    render(<App />)

    fireEvent.change(screen.getByLabelText('職位名稱'), { target: { value: '後端工程師' } })
    fireEvent.change(screen.getByLabelText('職位要求及說明'), {
      target: { value: '需要熟悉 Go、PostgreSQL、REST API' },
    })
    fireEvent.change(screen.getByLabelText('個人資訊'), {
      target: { value: '有 Java 和 Go 學習經驗' },
    })
    fireEvent.change(screen.getByLabelText('題目數量'), { target: { value: '3' } })
    fireEvent.click(screen.getByRole('button', { name: '建立面試' }))

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith('/api/interviews', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          job_title: '後端工程師',
          job_description: '需要熟悉 Go、PostgreSQL、REST API',
          user_profile: '有 Java 和 Go 學習經驗',
          question_count: 3,
        }),
      })
    })
    expect(pushState).toHaveBeenCalledWith({}, '', '/interviews/interview-123')
  })

  it('shows an API error when create interview fails', async () => {
    mockPathname('/interviews/new')
    vi.stubGlobal('fetch', mockFetchOnce({ error: 'job_title is required' }, { status: 400 }))

    render(<App />)

    fireEvent.click(screen.getByRole('button', { name: '建立面試' }))

    expect(await screen.findByText('job_title is required')).toBeInTheDocument()
  })

  it('loads interview details and displays questions at /interviews/:id', async () => {
    mockPathname('/interviews/interview-123')
    vi.stubGlobal(
      'fetch',
      mockFetchOnce({
        id: 'interview-123',
        job_title: '後端工程師',
        job_description: '需要熟悉 Go、PostgreSQL、REST API',
        user_profile: '有 Java 和 Go 學習經驗',
        question_count: 2,
        status: 'questions_ready',
        questions: [
          { id: 'question-1', order: 1, text: '請介紹你過去與後端開發相關的經驗。' },
          { id: 'question-2', order: 2, text: '你如何設計一個 REST API？' },
        ],
        answers: [],
      }),
    )

    render(<App />)

    expect(await screen.findByRole('heading', { name: '後端工程師' })).toBeInTheDocument()
    expect(screen.getByText('questions_ready')).toBeInTheDocument()
    expect(screen.getByText('請介紹你過去與後端開發相關的經驗。')).toBeInTheDocument()
    expect(screen.getByText('你如何設計一個 REST API？')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: '開始模擬面試' })).toHaveAttribute(
      'href',
      '/interviews/interview-123/session',
    )
  })

  it('loads the interview session page at /interviews/:id/session', async () => {
    mockPathname('/interviews/interview-123/session')
    vi.stubGlobal(
      'fetch',
      mockFetchOnce({
        id: 'interview-123',
        job_title: '後端工程師',
        job_description: '需要熟悉 Go、PostgreSQL、REST API',
        user_profile: '有 Java 和 Go 學習經驗',
        question_count: 2,
        status: 'questions_ready',
        questions: [
          { id: 'question-1', order: 1, text: '請介紹你過去與後端開發相關的經驗。' },
          { id: 'question-2', order: 2, text: '你如何設計一個 REST API？' },
        ],
        answers: [],
      }),
    )

    render(<App />)

    expect(await screen.findByRole('heading', { name: '模擬面試進行中' })).toBeInTheDocument()
    expect(screen.getByText('後端工程師')).toBeInTheDocument()
    expect(screen.getByText('第 1 題 / 共 2 題')).toBeInTheDocument()
    expect(screen.getByText('請介紹你過去與後端開發相關的經驗。')).toBeInTheDocument()
  })

  it('loads the completed interview result page with playable answers', async () => {
    mockPathname('/interviews/interview-123/result')
    vi.stubGlobal(
      'fetch',
      mockFetchOnce({
        id: 'interview-123',
        job_title: '後端工程師',
        job_description: '需要熟悉 Go、PostgreSQL、REST API',
        user_profile: '有 Java 和 Go 學習經驗',
        question_count: 2,
        status: 'completed',
        questions: [
          { id: 'question-1', order: 1, text: '請介紹你過去與後端開發相關的經驗。' },
          { id: 'question-2', order: 2, text: '你如何設計一個 REST API？' },
        ],
        answers: [
          {
            id: 'answer-1',
            question_id: 'question-1',
            audio_path: 'storage/audio/interview-123/question-1.webm',
            transcript_text: null,
            created_at: '2026-05-27T06:30:04Z',
          },
          {
            id: 'answer-2',
            question_id: 'question-2',
            audio_path: 'storage/audio/interview-123/question-2.webm',
            transcript_text: '我會先確認需求，再設計 resource 與錯誤格式。',
            created_at: '2026-05-27T06:31:04Z',
          },
        ],
      }),
    )

    render(<App />)

    expect(await screen.findByRole('heading', { name: '面試結果' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: '後端工程師' })).toBeInTheDocument()
    expect(screen.getByText('需要熟悉 Go、PostgreSQL、REST API')).toBeInTheDocument()
    expect(screen.getByText('有 Java 和 Go 學習經驗')).toBeInTheDocument()
    expect(screen.getByText('completed')).toBeInTheDocument()
    expect(screen.getByText('請介紹你過去與後端開發相關的經驗。')).toBeInTheDocument()
    expect(screen.getByText('你如何設計一個 REST API？')).toBeInTheDocument()
    expect(screen.getByText('尚未轉錄')).toBeInTheDocument()
    expect(screen.getByText('我會先確認需求，再設計 resource 與錯誤格式。')).toBeInTheDocument()
    expect(screen.getByLabelText('問題 1 回答音檔')).toHaveAttribute(
      'src',
      '/audio/interview-123/question-1.webm',
    )
    expect(screen.getByLabelText('問題 2 回答音檔')).toHaveAttribute(
      'src',
      '/audio/interview-123/question-2.webm',
    )
  })

  it('shows a missing-answer state on the result page', async () => {
    mockPathname('/interviews/interview-123/result')
    vi.stubGlobal(
      'fetch',
      mockFetchOnce({
        id: 'interview-123',
        job_title: '後端工程師',
        job_description: '需要熟悉 Go',
        user_profile: '有 Go 學習經驗',
        question_count: 1,
        status: 'completed',
        questions: [{ id: 'question-1', order: 1, text: '第一題' }],
        answers: [],
      }),
    )

    render(<App />)

    expect(await screen.findByText('第一題')).toBeInTheDocument()
    expect(screen.getByText('尚未上傳回答')).toBeInTheDocument()
    expect(screen.queryByLabelText('問題 1 回答音檔')).not.toBeInTheDocument()
  })

  it('shows an API error when loading a result page fails', async () => {
    mockPathname('/interviews/missing/result')
    vi.stubGlobal('fetch', mockFetchOnce({ error: 'interview not found' }, { status: 404 }))

    render(<App />)

    expect(await screen.findByText('interview not found')).toBeInTheDocument()
  })

  it('speaks the current session question when the play button is clicked', async () => {
    const speech = installSpeechSynthesisMock()
    mockPathname('/interviews/interview-123/session')
    vi.stubGlobal(
      'fetch',
      mockFetchOnce({
        id: 'interview-123',
        job_title: '後端工程師',
        job_description: '需要熟悉 Go、PostgreSQL、REST API',
        user_profile: '有 Java 和 Go 學習經驗',
        question_count: 2,
        status: 'questions_ready',
        questions: [
          { id: 'question-1', order: 1, text: '請介紹你過去與後端開發相關的經驗。' },
          { id: 'question-2', order: 2, text: '你如何設計一個 REST API？' },
        ],
        answers: [],
      }),
    )

    render(<App />)

    expect(await screen.findByText('請介紹你過去與後端開發相關的經驗。')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: '朗讀題目' }))

    expect(speech.cancel).toHaveBeenCalledTimes(1)
    expect(speech.speak).toHaveBeenCalledTimes(1)
    expect(speech.utterances[0].text).toBe('請介紹你過去與後端開發相關的經驗。')
    expect(speech.utterances[0].lang).toBe('zh-TW')
    expect(screen.getByRole('button', { name: '朗讀中' })).toBeDisabled()

    act(() => {
      speech.utterances[0].onend?.()
    })

    expect(screen.getByRole('button', { name: '朗讀題目' })).toBeEnabled()
  })

  it('cancels existing speech before speaking the question again', async () => {
    const speech = installSpeechSynthesisMock()
    mockPathname('/interviews/interview-123/session')
    vi.stubGlobal(
      'fetch',
      mockFetchOnce({
        id: 'interview-123',
        job_title: '後端工程師',
        job_description: '需要熟悉 Go、PostgreSQL、REST API',
        user_profile: '有 Java 和 Go 學習經驗',
        question_count: 1,
        status: 'questions_ready',
        questions: [
          { id: 'question-1', order: 1, text: '請介紹你過去與後端開發相關的經驗。' },
        ],
        answers: [],
      }),
    )

    render(<App />)

    expect(await screen.findByText('請介紹你過去與後端開發相關的經驗。')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: '朗讀題目' }))
    act(() => {
      speech.utterances[0].onend?.()
    })
    fireEvent.click(screen.getByRole('button', { name: '朗讀題目' }))

    expect(speech.cancel).toHaveBeenCalledTimes(2)
    expect(speech.speak).toHaveBeenCalledTimes(2)
  })

  it('stops speech when moving to another question', async () => {
    const speech = installSpeechSynthesisMock()
    mockPathname('/interviews/interview-123/session')
    vi.stubGlobal(
      'fetch',
      mockFetchOnce({
        id: 'interview-123',
        job_title: '後端工程師',
        job_description: '需要熟悉 Go、PostgreSQL、REST API',
        user_profile: '有 Java 和 Go 學習經驗',
        question_count: 2,
        status: 'questions_ready',
        questions: [
          { id: 'question-1', order: 1, text: '請介紹你過去與後端開發相關的經驗。' },
          { id: 'question-2', order: 2, text: '你如何設計一個 REST API？' },
        ],
        answers: [
          {
            id: 'answer-1',
            question_id: 'question-1',
            audio_path: 'storage/audio/interview-123/question-1.webm',
            transcript_text: null,
            created_at: '2026-05-27T06:30:04Z',
          },
        ],
      }),
    )

    render(<App />)

    expect(await screen.findByText('請介紹你過去與後端開發相關的經驗。')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: '朗讀題目' }))
    fireEvent.click(screen.getByRole('button', { name: '下一題' }))

    expect(speech.cancel).toHaveBeenCalledTimes(2)
    expect(screen.getByRole('button', { name: '朗讀題目' })).toBeEnabled()
  })

  it('stops speech when leaving the session page', async () => {
    const speech = installSpeechSynthesisMock()
    mockPathname('/interviews/interview-123/session')
    vi.stubGlobal(
      'fetch',
      mockFetchOnce({
        id: 'interview-123',
        job_title: '後端工程師',
        job_description: '需要熟悉 Go、PostgreSQL、REST API',
        user_profile: '有 Java 和 Go 學習經驗',
        question_count: 1,
        status: 'questions_ready',
        questions: [
          { id: 'question-1', order: 1, text: '請介紹你過去與後端開發相關的經驗。' },
        ],
        answers: [],
      }),
    )

    const { unmount } = render(<App />)

    expect(await screen.findByText('請介紹你過去與後端開發相關的經驗。')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: '朗讀題目' }))
    unmount()

    expect(speech.cancel).toHaveBeenCalledTimes(2)
  })

  it('disables question playback when speech synthesis is unavailable', async () => {
    vi.stubGlobal('speechSynthesis', undefined)
    vi.stubGlobal('SpeechSynthesisUtterance', undefined)
    mockPathname('/interviews/interview-123/session')
    vi.stubGlobal(
      'fetch',
      mockFetchOnce({
        id: 'interview-123',
        job_title: '後端工程師',
        job_description: '需要熟悉 Go、PostgreSQL、REST API',
        user_profile: '有 Java 和 Go 學習經驗',
        question_count: 1,
        status: 'questions_ready',
        questions: [
          { id: 'question-1', order: 1, text: '請介紹你過去與後端開發相關的經驗。' },
        ],
        answers: [],
      }),
    )

    render(<App />)

    expect(await screen.findByText('請介紹你過去與後端開發相關的經驗。')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '朗讀題目' })).toBeDisabled()
    expect(screen.getByText('此瀏覽器不支援題目朗讀。')).toBeInTheDocument()
  })

  it('starts recording an answer for the current session question', async () => {
    const media = installMediaRecorderMock()
    installObjectURLMock()
    mockPathname('/interviews/interview-123/session')
    vi.stubGlobal(
      'fetch',
      mockFetchOnce({
        id: 'interview-123',
        job_title: '後端工程師',
        job_description: '需要熟悉 Go、PostgreSQL、REST API',
        user_profile: '有 Java 和 Go 學習經驗',
        question_count: 1,
        status: 'questions_ready',
        questions: [
          { id: 'question-1', order: 1, text: '請介紹你過去與後端開發相關的經驗。' },
        ],
        answers: [
          {
            id: 'answer-1',
            question_id: 'question-1',
            audio_path: 'storage/audio/interview-123/question-1.webm',
            transcript_text: null,
            created_at: '2026-05-27T06:30:04Z',
          },
        ],
      }),
    )

    render(<App />)

    expect(await screen.findByText('請介紹你過去與後端開發相關的經驗。')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: '開始錄音' }))

    await waitFor(() => {
      expect(media.getUserMedia).toHaveBeenCalledWith({ audio: true })
    })
    expect(media.recorders).toHaveLength(1)
    expect(media.recorders[0].start).toHaveBeenCalledTimes(1)
    expect(screen.getByRole('button', { name: '錄音中' })).toBeDisabled()
    expect(screen.getByRole('button', { name: '停止錄音' })).toBeEnabled()
  })

  it('stops recording and shows an audio preview for the recorded answer', async () => {
    const media = installMediaRecorderMock()
    const objectURL = installObjectURLMock()
    mockPathname('/interviews/interview-123/session')
    vi.stubGlobal(
      'fetch',
      mockFetchOnce({
        id: 'interview-123',
        job_title: '後端工程師',
        job_description: '需要熟悉 Go、PostgreSQL、REST API',
        user_profile: '有 Java 和 Go 學習經驗',
        question_count: 1,
        status: 'questions_ready',
        questions: [
          { id: 'question-1', order: 1, text: '請介紹你過去與後端開發相關的經驗。' },
        ],
        answers: [
          {
            id: 'answer-1',
            question_id: 'question-1',
            audio_path: 'storage/audio/interview-123/question-1.webm',
            transcript_text: null,
            created_at: '2026-05-27T06:30:04Z',
          },
        ],
      }),
    )

    render(<App />)

    expect(await screen.findByText('請介紹你過去與後端開發相關的經驗。')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: '開始錄音' }))

    await waitFor(() => {
      expect(media.recorders).toHaveLength(1)
    })
    fireEvent.click(screen.getByRole('button', { name: '停止錄音' }))

    expect(media.recorders[0].stop).toHaveBeenCalledTimes(1)
    expect(media.trackStop).toHaveBeenCalledTimes(1)
    expect(objectURL.createObjectURL).toHaveBeenCalledTimes(1)
    expect(screen.getByLabelText('回答錄音預覽')).toHaveAttribute('src', 'blob:recorded-answer')
    expect(screen.getByRole('button', { name: '開始錄音' })).toBeEnabled()
  })

  it('shows an error when microphone permission is denied', async () => {
    installObjectURLMock()
    Object.defineProperty(navigator, 'mediaDevices', {
      configurable: true,
      value: {
        getUserMedia: vi.fn().mockRejectedValue(new Error('Permission denied')),
      },
    })
    vi.stubGlobal('MediaRecorder', class MockMediaRecorder {})
    mockPathname('/interviews/interview-123/session')
    vi.stubGlobal(
      'fetch',
      mockFetchOnce({
        id: 'interview-123',
        job_title: '後端工程師',
        job_description: '需要熟悉 Go、PostgreSQL、REST API',
        user_profile: '有 Java 和 Go 學習經驗',
        question_count: 1,
        status: 'questions_ready',
        questions: [
          { id: 'question-1', order: 1, text: '請介紹你過去與後端開發相關的經驗。' },
        ],
        answers: [
          {
            id: 'answer-1',
            question_id: 'question-1',
            audio_path: 'storage/audio/interview-123/question-1.webm',
            transcript_text: null,
            created_at: '2026-05-27T06:30:04Z',
          },
        ],
      }),
    )

    render(<App />)

    expect(await screen.findByText('請介紹你過去與後端開發相關的經驗。')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: '開始錄音' }))

    expect(await screen.findByText('Permission denied')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '開始錄音' })).toBeEnabled()
  })

  it('shows a helpful message when no microphone device is available', async () => {
    installObjectURLMock()
    const deviceError = new Error('Requested device not found')
    deviceError.name = 'NotFoundError'
    Object.defineProperty(navigator, 'mediaDevices', {
      configurable: true,
      value: {
        getUserMedia: vi.fn().mockRejectedValue(deviceError),
      },
    })
    vi.stubGlobal('MediaRecorder', class MockMediaRecorder {})
    mockPathname('/interviews/interview-123/session')
    vi.stubGlobal(
      'fetch',
      mockFetchOnce({
        id: 'interview-123',
        job_title: '後端工程師',
        job_description: '需要熟悉 Go、PostgreSQL、REST API',
        user_profile: '有 Java 和 Go 學習經驗',
        question_count: 1,
        status: 'questions_ready',
        questions: [
          { id: 'question-1', order: 1, text: '請介紹你過去與後端開發相關的經驗。' },
        ],
        answers: [
          {
            id: 'answer-1',
            question_id: 'question-1',
            audio_path: 'storage/audio/interview-123/question-1.webm',
            transcript_text: null,
            created_at: '2026-05-27T06:30:04Z',
          },
        ],
      }),
    )

    render(<App />)

    expect(await screen.findByText('請介紹你過去與後端開發相關的經驗。')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: '開始錄音' }))

    expect(
      await screen.findByText('找不到可用的麥克風，請確認裝置已連接，並在瀏覽器或系統設定中允許麥克風。'),
    ).toBeInTheDocument()
    expect(screen.queryByText('Requested device not found')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: '開始錄音' })).toBeEnabled()
  })

  it('disables answer recording when MediaRecorder is unavailable', async () => {
    installObjectURLMock()
    Object.defineProperty(navigator, 'mediaDevices', {
      configurable: true,
      value: undefined,
    })
    vi.stubGlobal('MediaRecorder', undefined)
    mockPathname('/interviews/interview-123/session')
    vi.stubGlobal(
      'fetch',
      mockFetchOnce({
        id: 'interview-123',
        job_title: '後端工程師',
        job_description: '需要熟悉 Go、PostgreSQL、REST API',
        user_profile: '有 Java 和 Go 學習經驗',
        question_count: 1,
        status: 'questions_ready',
        questions: [
          { id: 'question-1', order: 1, text: '請介紹你過去與後端開發相關的經驗。' },
        ],
        answers: [],
      }),
    )

    render(<App />)

    expect(await screen.findByText('請介紹你過去與後端開發相關的經驗。')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '開始錄音' })).toBeDisabled()
    expect(screen.getByRole('button', { name: '停止錄音' })).toBeDisabled()
    expect(screen.getByText('此瀏覽器不支援錄音。')).toBeInTheDocument()
  })

  it('stops active recording and clears preview when moving to another question', async () => {
    const media = installMediaRecorderMock()
    const objectURL = installObjectURLMock()
    mockPathname('/interviews/interview-123/session')
    vi.stubGlobal(
      'fetch',
      mockFetchOnce({
        id: 'interview-123',
        job_title: '後端工程師',
        job_description: '需要熟悉 Go、PostgreSQL、REST API',
        user_profile: '有 Java 和 Go 學習經驗',
        question_count: 2,
        status: 'questions_ready',
        questions: [
          { id: 'question-1', order: 1, text: '請介紹你過去與後端開發相關的經驗。' },
          { id: 'question-2', order: 2, text: '你如何設計一個 REST API？' },
        ],
        answers: [
          {
            id: 'answer-1',
            question_id: 'question-1',
            audio_path: 'storage/audio/interview-123/question-1.webm',
            transcript_text: null,
            created_at: '2026-05-27T06:30:04Z',
          },
        ],
      }),
    )

    render(<App />)

    expect(await screen.findByText('請介紹你過去與後端開發相關的經驗。')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: '開始錄音' }))

    await waitFor(() => {
      expect(media.recorders).toHaveLength(1)
    })
    fireEvent.click(screen.getByRole('button', { name: '停止錄音' }))
    expect(screen.getByLabelText('回答錄音預覽')).toHaveAttribute('src', 'blob:recorded-answer')

    fireEvent.click(screen.getByRole('button', { name: '下一題' }))

    expect(objectURL.revokeObjectURL).toHaveBeenCalledWith('blob:recorded-answer')
    expect(screen.queryByLabelText('回答錄音預覽')).not.toBeInTheDocument()
  })

  it('stops media tracks when leaving the session page during recording', async () => {
    const media = installMediaRecorderMock()
    installObjectURLMock()
    mockPathname('/interviews/interview-123/session')
    vi.stubGlobal(
      'fetch',
      mockFetchOnce({
        id: 'interview-123',
        job_title: '後端工程師',
        job_description: '需要熟悉 Go、PostgreSQL、REST API',
        user_profile: '有 Java 和 Go 學習經驗',
        question_count: 1,
        status: 'questions_ready',
        questions: [
          { id: 'question-1', order: 1, text: '請介紹你過去與後端開發相關的經驗。' },
        ],
        answers: [],
      }),
    )

    const { unmount } = render(<App />)

    expect(await screen.findByText('請介紹你過去與後端開發相關的經驗。')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: '開始錄音' }))

    await waitFor(() => {
      expect(media.recorders).toHaveLength(1)
    })
    unmount()

    expect(media.trackStop).toHaveBeenCalledTimes(1)
  })

  it('revokes the recorded preview URL when leaving the session page', async () => {
    const media = installMediaRecorderMock()
    const objectURL = installObjectURLMock()
    mockPathname('/interviews/interview-123/session')
    vi.stubGlobal(
      'fetch',
      mockFetchOnce({
        id: 'interview-123',
        job_title: '後端工程師',
        job_description: '需要熟悉 Go、PostgreSQL、REST API',
        user_profile: '有 Java 和 Go 學習經驗',
        question_count: 1,
        status: 'questions_ready',
        questions: [
          { id: 'question-1', order: 1, text: '請介紹你過去與後端開發相關的經驗。' },
        ],
        answers: [],
      }),
    )

    const { unmount } = render(<App />)

    expect(await screen.findByText('請介紹你過去與後端開發相關的經驗。')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: '開始錄音' }))

    await waitFor(() => {
      expect(media.recorders).toHaveLength(1)
    })
    fireEvent.click(screen.getByRole('button', { name: '停止錄音' }))
    expect(screen.getByLabelText('回答錄音預覽')).toHaveAttribute('src', 'blob:recorded-answer')

    unmount()

    expect(objectURL.revokeObjectURL).toHaveBeenCalledWith('blob:recorded-answer')
  })

  it('shows uploaded answer state when resuming an answered question', async () => {
    mockPathname('/interviews/interview-123/session')
    vi.stubGlobal(
      'fetch',
      mockFetchOnce({
        id: 'interview-123',
        job_title: '後端工程師',
        job_description: '需要熟悉 Go、PostgreSQL、REST API',
        user_profile: '有 Java 和 Go 學習經驗',
        question_count: 1,
        status: 'questions_ready',
        questions: [
          { id: 'question-1', order: 1, text: '請介紹你過去與後端開發相關的經驗。' },
        ],
        answers: [
          {
            id: 'answer-1',
            question_id: 'question-1',
            audio_path: 'storage/audio/interview-123/question-1.webm',
            transcript_text: null,
            created_at: '2026-05-27T06:30:04Z',
          },
        ],
      }),
    )

    render(<App />)

    expect(await screen.findByText('本題回答已上傳')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '完成面試' })).toBeEnabled()
  })

  it('uploads the recorded answer for the current question', async () => {
    const media = installMediaRecorderMock()
    installObjectURLMock()
    mockPathname('/interviews/interview-123/session')
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            id: 'interview-123',
            job_title: '後端工程師',
            job_description: '需要熟悉 Go、PostgreSQL、REST API',
            user_profile: '有 Java 和 Go 學習經驗',
            question_count: 1,
            status: 'questions_ready',
            questions: [
              { id: 'question-1', order: 1, text: '請介紹你過去與後端開發相關的經驗。' },
            ],
            answers: [],
          }),
          { headers: { 'Content-Type': 'application/json' } },
        ),
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            id: 'answer-1',
            interview_id: 'interview-123',
            question_id: 'question-1',
            audio_path: 'storage/audio/interview-123/question-1.webm',
            transcript_text: null,
          }),
          { status: 201, headers: { 'Content-Type': 'application/json' } },
        ),
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            id: 'interview-123',
            job_title: '後端工程師',
            job_description: '需要熟悉 Go、PostgreSQL、REST API',
            user_profile: '有 Java 和 Go 學習經驗',
            question_count: 1,
            status: 'completed',
            questions: [
              { id: 'question-1', order: 1, text: '請介紹你過去與後端開發相關的經驗。' },
            ],
            answers: [
              {
                id: 'answer-1',
                question_id: 'question-1',
                audio_path: 'storage/audio/interview-123/question-1.webm',
                transcript_text: null,
                created_at: '2026-05-27T06:30:04Z',
              },
            ],
          }),
          { headers: { 'Content-Type': 'application/json' } },
        ),
      )
    vi.stubGlobal('fetch', fetchMock)

    render(<App />)

    expect(await screen.findByText('請介紹你過去與後端開發相關的經驗。')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: '開始錄音' }))

    await waitFor(() => {
      expect(media.recorders).toHaveLength(1)
    })
    fireEvent.click(screen.getByRole('button', { name: '停止錄音' }))
    fireEvent.click(await screen.findByRole('button', { name: '上傳本題回答' }))

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledTimes(2)
    })
    expect(fetchMock).toHaveBeenLastCalledWith(
      '/api/interviews/interview-123/questions/question-1/answer',
      expect.objectContaining({
        method: 'POST',
        body: expect.any(FormData),
      }),
    )
    expect(await screen.findByText('本題回答已上傳')).toBeInTheDocument()
  })

  it('finishes the session after uploading the final answer', async () => {
    const media = installMediaRecorderMock()
    installObjectURLMock()
    mockPathname('/interviews/interview-123/session')
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            id: 'interview-123',
            job_title: '後端工程師',
            job_description: '需要熟悉 Go、PostgreSQL、REST API',
            user_profile: '有 Java 和 Go 學習經驗',
            question_count: 1,
            status: 'questions_ready',
            questions: [
              { id: 'question-1', order: 1, text: '請介紹你過去與後端開發相關的經驗。' },
            ],
            answers: [],
          }),
          { headers: { 'Content-Type': 'application/json' } },
        ),
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            id: 'answer-1',
            interview_id: 'interview-123',
            question_id: 'question-1',
            audio_path: 'storage/audio/interview-123/question-1.webm',
            transcript_text: null,
          }),
          { status: 201, headers: { 'Content-Type': 'application/json' } },
        ),
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            id: 'interview-123',
            job_title: '後端工程師',
            job_description: '需要熟悉 Go、PostgreSQL、REST API',
            user_profile: '有 Java 和 Go 學習經驗',
            question_count: 1,
            status: 'completed',
            questions: [
              { id: 'question-1', order: 1, text: '請介紹你過去與後端開發相關的經驗。' },
            ],
            answers: [
              {
                id: 'answer-1',
                question_id: 'question-1',
                audio_path: 'storage/audio/interview-123/question-1.webm',
                transcript_text: null,
                created_at: '2026-05-27T06:30:04Z',
              },
            ],
          }),
          { headers: { 'Content-Type': 'application/json' } },
        ),
      )
    vi.stubGlobal('fetch', fetchMock)

    render(<App />)

    expect(await screen.findByText('請介紹你過去與後端開發相關的經驗。')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '完成面試' })).toBeDisabled()
    fireEvent.click(screen.getByRole('button', { name: '開始錄音' }))

    await waitFor(() => {
      expect(media.recorders).toHaveLength(1)
    })
    fireEvent.click(screen.getByRole('button', { name: '停止錄音' }))
    fireEvent.click(await screen.findByRole('button', { name: '上傳本題回答' }))

    expect(await screen.findByText('本題回答已上傳')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: '完成面試' }))

    expect(await screen.findByRole('heading', { name: '面試結果' })).toBeInTheDocument()
    expect(screen.getByLabelText('問題 1 回答音檔')).toHaveAttribute(
      'src',
      '/audio/interview-123/question-1.webm',
    )
    expect(window.location.pathname).toBe('/interviews/interview-123/result')
  })

  it('moves between session questions with previous and next buttons', async () => {
    mockPathname('/interviews/interview-123/session')
    vi.stubGlobal(
      'fetch',
      mockFetchOnce({
        id: 'interview-123',
        job_title: '後端工程師',
        job_description: '需要熟悉 Go、PostgreSQL、REST API',
        user_profile: '有 Java 和 Go 學習經驗',
        question_count: 2,
        status: 'questions_ready',
        questions: [
          { id: 'question-1', order: 1, text: '請介紹你過去與後端開發相關的經驗。' },
          { id: 'question-2', order: 2, text: '你如何設計一個 REST API？' },
        ],
        answers: [
          {
            id: 'answer-1',
            question_id: 'question-1',
            audio_path: 'storage/audio/interview-123/question-1.webm',
            transcript_text: null,
            created_at: '2026-05-27T06:30:04Z',
          },
        ],
      }),
    )

    render(<App />)

    expect(await screen.findByText('請介紹你過去與後端開發相關的經驗。')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '上一題' })).toBeDisabled()
    expect(screen.getByRole('button', { name: '下一題' })).toBeEnabled()

    fireEvent.click(screen.getByRole('button', { name: '下一題' }))

    expect(screen.getByText('第 2 題 / 共 2 題')).toBeInTheDocument()
    expect(screen.getByText('你如何設計一個 REST API？')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '上一題' })).toBeEnabled()
    expect(screen.getByRole('button', { name: '完成面試' })).toBeDisabled()

    fireEvent.click(screen.getByRole('button', { name: '上一題' }))

    expect(screen.getByText('第 1 題 / 共 2 題')).toBeInTheDocument()
    expect(screen.getByText('請介紹你過去與後端開發相關的經驗。')).toBeInTheDocument()
  })

  it('shows an empty state when a session has no questions', async () => {
    mockPathname('/interviews/interview-empty/session')
    vi.stubGlobal(
      'fetch',
      mockFetchOnce({
        id: 'interview-empty',
        job_title: '後端工程師',
        job_description: '需要熟悉 Go',
        user_profile: '有 Go 學習經驗',
        question_count: 0,
        status: 'questions_ready',
        questions: [],
        answers: [],
      }),
    )

    render(<App />)

    expect(await screen.findByText('這場面試目前沒有題目。')).toBeInTheDocument()
  })

  it('shows an API error when loading a session fails', async () => {
    mockPathname('/interviews/missing/session')
    vi.stubGlobal('fetch', mockFetchOnce({ error: 'interview not found' }, { status: 404 }))

    render(<App />)

    expect(await screen.findByText('interview not found')).toBeInTheDocument()
  })
})
