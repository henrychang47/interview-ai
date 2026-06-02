import '@testing-library/jest-dom/vitest'
import { act, cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import App from './App'
import { clearQuestionAudioCache, storeQuestionAudio } from './lib/questionAudioCache'

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

function findElementByExactText(text: string) {
  return screen.getByText(
    (_, element) => element?.tagName.toLowerCase() === 'p' && element.textContent === text,
  )
}

function interviewDetailResponse(overrides: Record<string, unknown> = {}) {
  return new Response(
    JSON.stringify({
      id: 'interview-123',
      job_title: '後端工程師',
      job_description: '需要熟悉 Go、PostgreSQL、REST API',
      user_profile: '有 Java 和 Go 學習經驗',
      question_count: 2,
      question_language: 'zh-TW',
      status: 'generating_questions',
      questions: [],
      answers: [],
      ...overrides,
    }),
    { headers: { 'Content-Type': 'application/json' } },
  )
}

function deferredResponse(response: Response) {
  let resolve!: (response: Response) => void
  const promise = new Promise<Response>((promiseResolve) => {
    resolve = promiseResolve
  })

  return { promise, resolve: () => resolve(response) }
}

type MockUtterance = {
  text: string
  lang: string
  voice: SpeechSynthesisVoice | null
  rate: number
  pitch: number
  onend: (() => void) | null
  onerror: (() => void) | null
}

function createSpeechVoice(language: string, name = language) {
  return {
    default: false,
    lang: language,
    localService: true,
    name,
    voiceURI: name,
  } as SpeechSynthesisVoice
}

function installSpeechSynthesisMock(initialVoices: SpeechSynthesisVoice[] = []) {
  const speak = vi.fn()
  const cancel = vi.fn()
  const utterances: MockUtterance[] = []
  const listeners = new Set<() => void>()
  let voices = initialVoices

  class MockSpeechSynthesisUtterance {
    text: string
    lang = ''
    voice: SpeechSynthesisVoice | null = null
    rate = 1
    pitch = 1
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
    getVoices: vi.fn(() => voices),
    addEventListener: vi.fn((eventName: string, listener: () => void) => {
      if (eventName === 'voiceschanged') {
        listeners.add(listener)
      }
    }),
    removeEventListener: vi.fn((eventName: string, listener: () => void) => {
      if (eventName === 'voiceschanged') {
        listeners.delete(listener)
      }
    }),
  })
  vi.stubGlobal('SpeechSynthesisUtterance', MockSpeechSynthesisUtterance)

  return {
    speak,
    cancel,
    utterances,
    setVoices: (nextVoices: SpeechSynthesisVoice[]) => {
      voices = nextVoices
      listeners.forEach((listener) => listener())
    },
  }
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

type MockAudioInstance = {
  src: string
  onended: (() => void) | null
  onerror: (() => void) | null
  play: ReturnType<typeof vi.fn>
  pause: ReturnType<typeof vi.fn>
}

function installAudioMock() {
  const instances: MockAudioInstance[] = []

  class MockAudio {
    src: string
    onended: (() => void) | null = null
    onerror: (() => void) | null = null

    constructor(src = '') {
      this.src = src
      instances.push(this as unknown as MockAudioInstance)
    }

    play = vi.fn().mockResolvedValue(undefined)
    pause = vi.fn()
  }

  vi.stubGlobal('Audio', MockAudio)

  return { instances }
}

describe('App', () => {
  beforeEach(() => {
    mockPathname('/')
    vi.restoreAllMocks()
  })

  afterEach(() => {
    cleanup()
    clearQuestionAudioCache()
    vi.restoreAllMocks()
    vi.useRealTimers()
    window.history.pushState({}, '', '/')
  })

  it('renders the interview practice homepage', () => {
    render(<App />)

    expect(screen.getByRole('link', { name: 'AI模擬面試' })).toHaveAttribute('href', '/')
    expect(
      screen.getByRole('heading', {
        name: '提升您的面試表現，隨時隨地與 AI 導師練習',
      }),
    ).toBeInTheDocument()
    expect(
      screen.getByText('四個步驟，完成從題目生成到錄音回顧的模擬面試流程。'),
    ).toBeInTheDocument()
    expect(screen.getByRole('link', { name: '建立新的模擬面試' })).toHaveAttribute(
      'href',
      '/interviews/new',
    )
  })

  it('renders the create interview form at /interviews/new', () => {
    mockPathname('/interviews/new')

    render(<App />)

    expect(screen.getByRole('link', { name: 'AI模擬面試' })).toHaveAttribute('href', '/')
    expect(screen.getByRole('link', { name: '返回首頁' })).toHaveAttribute('href', '/')
    expect(screen.getByRole('heading', { name: '建立新面試' })).toBeInTheDocument()
    expect(screen.getByLabelText('職位名稱')).toBeInTheDocument()
    expect(screen.getByLabelText('職位要求及說明')).toBeInTheDocument()
    expect(screen.getByLabelText('個人資訊')).toBeInTheDocument()
    expect(screen.queryByLabelText('題目數量')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: '下一步' })).toBeDisabled()
  })

  it('requires all profile fields before moving to setup', () => {
    mockPathname('/interviews/new')

    render(<App />)

    const nextButton = screen.getByRole('button', { name: '下一步' })
    expect(nextButton).toBeDisabled()

    fireEvent.change(screen.getByLabelText('職位名稱'), { target: { value: '後端工程師' } })
    fireEvent.change(screen.getByLabelText('職位要求及說明'), {
      target: { value: '需要熟悉 Go、PostgreSQL、REST API' },
    })
    expect(nextButton).toBeDisabled()

    fireEvent.change(screen.getByLabelText('個人資訊'), {
      target: { value: '有 Java 和 Go 學習經驗' },
    })

    expect(nextButton).toBeEnabled()
    fireEvent.click(nextButton)
    expect(screen.getByLabelText('題目數量')).toBeInTheDocument()
  })

  it('limits profile field lengths and shows counters for long text fields', () => {
    mockPathname('/interviews/new')

    render(<App />)

    const jobTitle = screen.getByLabelText('職位名稱')
    const jobDescription = screen.getByLabelText('職位要求及說明')
    const userProfile = screen.getByLabelText('個人資訊')

    expect(jobTitle).toHaveAttribute('maxLength', '50')
    expect(jobDescription).toHaveAttribute('maxLength', '4000')
    expect(userProfile).toHaveAttribute('maxLength', '4000')
    expect(screen.getAllByText('0/4000')).toHaveLength(2)
    expect(screen.queryByText('0/50')).not.toBeInTheDocument()

    fireEvent.change(jobTitle, { target: { value: '前'.repeat(51) } })
    fireEvent.change(jobDescription, { target: { value: '職'.repeat(4001) } })
    fireEvent.change(userProfile, { target: { value: '個'.repeat(123) } })

    expect(jobTitle).toHaveValue('前'.repeat(50))
    expect(jobDescription).toHaveValue('職'.repeat(4000))
    expect(userProfile).toHaveValue('個'.repeat(123))
    expect(screen.getByText('4000/4000')).toBeInTheDocument()
    expect(screen.getByText('123/4000')).toBeInTheDocument()
  })

  it('submits the two-stage create interview form after microphone test', async () => {
    const media = installMediaRecorderMock()
    installObjectURLMock()
    mockPathname('/interviews/new')
    const fetchMock = mockFetchOnce({ id: 'interview-123', status: 'generating_questions' })
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
    fireEvent.click(screen.getByRole('button', { name: '下一步' }))

    expect(await screen.findByLabelText('題目數量')).toHaveValue(3)
    fireEvent.change(screen.getByLabelText('題目數量'), { target: { value: '3' } })
    fireEvent.click(screen.getByRole('radio', { name: 'English' }))
    expect(screen.getByRole('button', { name: '建立面試' })).toBeDisabled()

    fireEvent.click(screen.getByRole('button', { name: '測試麥克風' }))
    await waitFor(() => expect(media.getUserMedia).toHaveBeenCalledWith({ audio: true }))
    await waitFor(() => expect(media.recorders).toHaveLength(1))
    expect(media.recorders[0].start).toHaveBeenCalledTimes(1)
    expect(screen.getByRole('button', { name: '停止錄音' })).toBeEnabled()
    expect(screen.getByRole('button', { name: '建立面試' })).toBeDisabled()

    fireEvent.click(screen.getByRole('button', { name: '停止錄音' }))

    expect(await screen.findByLabelText('麥克風測試錄音預覽')).toHaveAttribute(
      'src',
      'blob:recorded-answer',
    )
    expect(await screen.findByText('麥克風已就緒')).toBeInTheDocument()

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
          question_language: 'en-US',
        }),
      })
    })
    expect(pushState).toHaveBeenCalledWith({}, '', '/interviews/interview-123')
  })

  it('returns to the profile step from the setup step', async () => {
    mockPathname('/interviews/new')

    render(<App />)

    fireEvent.change(screen.getByLabelText('職位名稱'), { target: { value: '後端工程師' } })
    fireEvent.change(screen.getByLabelText('職位要求及說明'), {
      target: { value: '需要熟悉 Go、PostgreSQL、REST API' },
    })
    fireEvent.change(screen.getByLabelText('個人資訊'), {
      target: { value: '有 Java 和 Go 學習經驗' },
    })
    fireEvent.click(screen.getByRole('button', { name: '下一步' }))

    expect(await screen.findByLabelText('題目數量')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: '上一步' }))

    expect(screen.getByLabelText('職位名稱')).toHaveValue('後端工程師')
    expect(screen.queryByLabelText('題目數量')).not.toBeInTheDocument()
  })

  it('shows an API error when create interview fails', async () => {
    installMediaRecorderMock()
    installObjectURLMock()
    mockPathname('/interviews/new')
    vi.stubGlobal('fetch', mockFetchOnce({ error: 'job_title is required' }, { status: 400 }))

    render(<App />)

    fireEvent.change(screen.getByLabelText('職位名稱'), { target: { value: '後端工程師' } })
    fireEvent.change(screen.getByLabelText('職位要求及說明'), {
      target: { value: '需要熟悉 Go' },
    })
    fireEvent.change(screen.getByLabelText('個人資訊'), {
      target: { value: '有 Go 學習經驗' },
    })
    fireEvent.click(screen.getByRole('button', { name: '下一步' }))
    fireEvent.click(await screen.findByRole('button', { name: '測試麥克風' }))
    fireEvent.click(await screen.findByRole('button', { name: '停止錄音' }))
    expect(await screen.findByText('麥克風已就緒')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: '建立面試' }))

    expect(await screen.findByText('job_title is required')).toBeInTheDocument()
  })

  it('shows question preparation while questions are generating', async () => {
    vi.useFakeTimers()
    mockPathname('/interviews/interview-123')
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          id: 'interview-123',
          job_title: '後端工程師',
          job_description: '需要熟悉 Go、PostgreSQL、REST API',
          user_profile: '有 Java 和 Go 學習經驗',
          question_count: 2,
          question_language: 'zh-TW',
          status: 'generating_questions',
          questions: [],
          answers: [],
        }),
        { headers: { 'Content-Type': 'application/json' } },
      ),
    )
    vi.stubGlobal('fetch', fetchMock)

    render(<App />)
    await act(async () => {})

    expect(screen.getByText('題目準備中')).toBeInTheDocument()
    expect(screen.queryByText('問題 1')).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: '開始模擬面試' })).not.toBeInTheDocument()

    await act(async () => {
      vi.advanceTimersByTime(2000)
    })
    await act(async () => {})
    expect(fetchMock).toHaveBeenCalledTimes(2)
    vi.useRealTimers()
  })

  it('keeps the question preparation page stable during background polling', async () => {
    vi.useFakeTimers()
    mockPathname('/interviews/interview-123')
    const pollingResponse = deferredResponse(interviewDetailResponse())
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(interviewDetailResponse())
      .mockReturnValueOnce(pollingResponse.promise)
    vi.stubGlobal('fetch', fetchMock)

    render(<App />)
    await act(async () => {})

    expect(screen.getByText('題目準備中')).toBeInTheDocument()
    expect(screen.queryByText('載入面試中...')).not.toBeInTheDocument()

    act(() => {
      vi.advanceTimersByTime(2000)
    })

    expect(fetchMock).toHaveBeenCalledTimes(2)
    expect(screen.getByText('題目準備中')).toBeInTheDocument()
    expect(screen.queryByText('載入面試中...')).not.toBeInTheDocument()

    pollingResponse.resolve()
    await act(async () => {})
    vi.useRealTimers()
  })

  it('starts a ready interview from the preparation page', async () => {
    mockPathname('/interviews/interview-123')
    const jobDescription = '需要熟悉 Go、PostgreSQL、REST API\n能設計可維護的服務'
    const userProfile = '有 Java 和 Go 學習經驗\n正在準備後端工程師面試'
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            id: 'interview-123',
            job_title: '後端工程師',
            job_description: jobDescription,
            user_profile: userProfile,
            question_count: 2,
            question_language: 'zh-TW',
            status: 'questions_ready',
            questions: [],
            answers: [],
          }),
          { headers: { 'Content-Type': 'application/json' } },
        ),
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            audio: [
              { question_id: 'question-1', content_type: 'audio/wav', audio_base64: 'cXVlc3Rpb24tMS13YXY=' },
              { question_id: 'question-2', content_type: 'audio/wav', audio_base64: 'cXVlc3Rpb24tMi13YXY=' },
            ],
          }),
          { headers: { 'Content-Type': 'application/json' } },
        ),
      )
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ id: 'interview-123', status: 'in_progress' }), {
          headers: { 'Content-Type': 'application/json' },
        }),
      )
    vi.stubGlobal('fetch', fetchMock)
    const pushState = vi.spyOn(window.history, 'pushState')

    render(<App />)

    expect(await screen.findByText('題目已準備完成')).toBeInTheDocument()
    expect(screen.queryByText('請介紹你過去與後端開發相關的經驗。')).not.toBeInTheDocument()
    expect(screen.queryByText(jobDescription)).not.toBeInTheDocument()
    expect(screen.queryByText(userProfile)).not.toBeInTheDocument()
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: '職位資訊' }))

    expect(screen.getByRole('dialog', { name: '職位資訊' })).toBeInTheDocument()
    expect(findElementByExactText(jobDescription)).toHaveClass('whitespace-pre-wrap', 'break-words')
    expect(screen.queryByText(userProfile)).not.toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: '關閉' }))

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: '個人資訊' }))
    expect(screen.getByRole('dialog', { name: '個人資訊' })).toBeInTheDocument()
    expect(findElementByExactText(userProfile)).toHaveClass('whitespace-pre-wrap', 'break-words')
    fireEvent.click(screen.getByRole('button', { name: '關閉' }))
    expect(await screen.findByRole('button', { name: '開始模擬面試' })).toBeEnabled()

    await waitFor(() =>
      expect(fetchMock).toHaveBeenCalledWith('/api/interviews/interview-123/questions/tts', {
        method: 'POST',
      }),
    )
    fireEvent.click(screen.getByRole('button', { name: '開始模擬面試' }))

    await waitFor(() =>
      expect(fetchMock).toHaveBeenCalledWith('/api/interviews/interview-123/start', {
        method: 'POST',
      }),
    )
    expect(fetchMock).not.toHaveBeenCalledWith('/api/interviews/interview-123/questions/question-1/tts', {
      method: 'POST',
    })
    expect(pushState).toHaveBeenCalledWith({}, '', '/interviews/interview-123/session')
  })

  it('keeps showing question preparation until question audio is prepared', async () => {
    mockPathname('/interviews/interview-123')
    const ttsResponse = deferredResponse(
      new Response(
        JSON.stringify({
          audio: [{ question_id: 'question-1', content_type: 'audio/wav', audio_base64: 'cXVlc3Rpb24tMS13YXY=' }],
        }),
        { headers: { 'Content-Type': 'application/json' } },
      ),
    )
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        interviewDetailResponse({
          status: 'questions_ready',
          questions: [],
        }),
      )
      .mockReturnValueOnce(ttsResponse.promise)
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ id: 'interview-123', status: 'in_progress' }), {
          headers: { 'Content-Type': 'application/json' },
        }),
      )
    vi.stubGlobal('fetch', fetchMock)

    render(<App />)

    expect(await screen.findByText('題目準備中')).toBeInTheDocument()
    await waitFor(() =>
      expect(fetchMock).toHaveBeenCalledWith('/api/interviews/interview-123/questions/tts', {
        method: 'POST',
      }),
    )
    expect(screen.queryByText('題目已準備完成')).not.toBeInTheDocument()
    expect(screen.queryByText('準備題目語音...')).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: '開始模擬面試' })).not.toBeInTheDocument()
    expect(screen.queryByText('面試進行中')).not.toBeInTheDocument()

    ttsResponse.resolve()
    await act(async () => {})
    expect(screen.getByText('題目已準備完成')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '開始模擬面試' })).toBeEnabled()
  })

  it('loads interview details and displays in-progress questions at /interviews/:id', async () => {
    mockPathname('/interviews/interview-123')
    vi.stubGlobal(
      'fetch',
      mockFetchOnce({
        id: 'interview-123',
        job_title: '後端工程師',
        job_description: '需要熟悉 Go、PostgreSQL、REST API',
        user_profile: '有 Java 和 Go 學習經驗',
        question_count: 2,
        question_language: 'zh-TW',
        status: 'in_progress',
        questions: [
          { id: 'question-1', order: 1, text: '請介紹你過去與後端開發相關的經驗。' },
          { id: 'question-2', order: 2, text: '你如何設計一個 REST API？' },
        ],
        answers: [],
      }),
    )

    render(<App />)

    expect(await screen.findByRole('heading', { name: '後端工程師' })).toBeInTheDocument()
    expect(screen.getByText('in_progress')).toBeInTheDocument()
    expect(screen.getByText('面試進行中')).toBeInTheDocument()
    expect(screen.queryByText('請介紹你過去與後端開發相關的經驗。')).not.toBeInTheDocument()
    expect(screen.getByRole('link', { name: '繼續面試' })).toHaveAttribute(
      'href',
      '/interviews/interview-123/session',
    )
  })

  it('automatically plays the first question and starts recording after playback ends', async () => {
    const googleTaiwanMandarinVoice = createSpeechVoice('zh-TW', 'Google 國語（臺灣）')
    const speech = installSpeechSynthesisMock([
      createSpeechVoice('en-US', 'US English'),
      createSpeechVoice('zh-TW', 'Taiwan Mandarin'),
      googleTaiwanMandarinVoice,
    ])
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
            question_language: 'zh-TW',
            status: 'in_progress',
            questions: [{ id: 'question-1', order: 1, text: '請介紹你過去與後端開發相關的經驗。' }],
            answers: [],
          }),
          { headers: { 'Content-Type': 'application/json' } },
        ),
      )
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ error: 'question TTS is unavailable' }), {
          status: 503,
          headers: { 'Content-Type': 'application/json' },
        }),
      )
    vi.stubGlobal('fetch', fetchMock)

    render(<App />)

    expect(await screen.findByText('正在播放題目')).toBeInTheDocument()
    expect(screen.queryByText('請介紹你過去與後端開發相關的經驗。')).not.toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/interviews/interview-123/questions/question-1/tts',
      { method: 'POST' },
    )
    expect(speech.speak).toHaveBeenCalledTimes(1)
    expect(speech.utterances[0].lang).toBe('zh-TW')
    expect(speech.utterances[0].voice).toBe(googleTaiwanMandarinVoice)
    expect(speech.utterances[0].rate).toBe(1.1)
    expect(speech.utterances[0].pitch).toBe(0.8)

    act(() => speech.utterances[0].onend?.())

    await waitFor(() => expect(media.getUserMedia).toHaveBeenCalledWith({ audio: true }))
    expect(screen.getByText('正在錄音')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '回答結束' })).toBeEnabled()
  })

  it('confirms before ending an active interview session and returning home', async () => {
    const speech = installSpeechSynthesisMock()
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
            job_description: '需要熟悉 Go',
            user_profile: '有 Go 經驗',
            question_count: 1,
            question_language: 'zh-TW',
            status: 'in_progress',
            questions: [{ id: 'question-1', order: 1, text: '問題一' }],
            answers: [],
          }),
          { headers: { 'Content-Type': 'application/json' } },
        ),
      )
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ error: 'question TTS is unavailable' }), {
          status: 503,
          headers: { 'Content-Type': 'application/json' },
        }),
      )
    vi.stubGlobal('fetch', fetchMock)
    const pushState = vi.spyOn(window.history, 'pushState')

    render(<App />)

    expect(await screen.findByText('正在播放題目')).toBeInTheDocument()
    act(() => speech.utterances[0].onend?.())
    await waitFor(() => expect(media.recorders).toHaveLength(1))
    expect(screen.getByText('正在錄音')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: '結束面試' }))

    expect(screen.getByRole('dialog', { name: '結束面試' })).toBeInTheDocument()
    expect(screen.getByText('此次模擬面試將直接結束，尚未完成或尚未上傳的回答不會保留。')).toBeInTheDocument()
    expect(window.location.pathname).toBe('/interviews/interview-123/session')

    fireEvent.click(screen.getByRole('button', { name: '取消' }))

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(window.location.pathname).toBe('/interviews/interview-123/session')
    expect(media.recorders[0].stop).not.toHaveBeenCalled()

    fireEvent.click(screen.getByRole('button', { name: '結束面試' }))
    fireEvent.click(screen.getByRole('button', { name: '確認結束' }))

    expect(media.recorders[0].stop).toHaveBeenCalledTimes(1)
    expect(pushState).toHaveBeenCalledWith({}, '', '/')
    await waitFor(() => expect(window.location.pathname).toBe('/'))
    expect(await screen.findByRole('heading', { name: '提升您的面試表現，隨時隨地與 AI 導師練習' })).toBeInTheDocument()
  })

  it('uses the matching English voice without Chinese speech tuning', async () => {
    const englishVoice = createSpeechVoice('en-US', 'US English')
    const speech = installSpeechSynthesisMock([
      createSpeechVoice('zh-TW', 'Google 國語（臺灣）'),
      englishVoice,
    ])
    installMediaRecorderMock()
    installObjectURLMock()
    mockPathname('/interviews/interview-123/session')
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            id: 'interview-123',
            job_title: 'Backend Engineer',
            job_description: 'Build APIs',
            user_profile: 'Go experience',
            question_count: 1,
            question_language: 'en-US',
            status: 'in_progress',
            questions: [{ id: 'question-1', order: 1, text: 'Tell me about your backend experience.' }],
            answers: [],
          }),
          { headers: { 'Content-Type': 'application/json' } },
        ),
      )
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ error: 'question TTS is unavailable' }), {
          status: 503,
          headers: { 'Content-Type': 'application/json' },
        }),
      )
    vi.stubGlobal('fetch', fetchMock)

    render(<App />)

    expect(await screen.findByText('正在播放題目')).toBeInTheDocument()
    await waitFor(() => expect(speech.speak).toHaveBeenCalledTimes(1))
    expect(speech.utterances[0].lang).toBe('en-US')
    expect(speech.utterances[0].voice).toBe(englishVoice)
    expect(speech.utterances[0].rate).toBe(1)
    expect(speech.utterances[0].pitch).toBe(1)
  })

  it('plays generated Gemini TTS audio before recording when the endpoint succeeds', async () => {
    const speech = installSpeechSynthesisMock()
    const media = installMediaRecorderMock()
    const objectURLs = installObjectURLMock()
    objectURLs.createObjectURL.mockReturnValueOnce('blob:question-audio')
    const audio = installAudioMock()
    mockPathname('/interviews/interview-123/session')
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            id: 'interview-123',
            job_title: '後端工程師',
            job_description: '需要熟悉 Go',
            user_profile: '有 Go 經驗',
            question_count: 1,
            question_language: 'zh-TW',
            status: 'in_progress',
            questions: [{ id: 'question-1', order: 1, text: '問題一' }],
            answers: [],
          }),
          { headers: { 'Content-Type': 'application/json' } },
        ),
      )
      .mockResolvedValueOnce(
        new Response('wav-bytes', {
          headers: { 'Content-Type': 'audio/wav' },
        }),
      )
    vi.stubGlobal('fetch', fetchMock)

    render(<App />)

    expect(await screen.findByText('正在播放題目')).toBeInTheDocument()
    await waitFor(() =>
      expect(fetchMock).toHaveBeenCalledWith(
        '/api/interviews/interview-123/questions/question-1/tts',
        { method: 'POST' },
      ),
    )
    expect(speech.speak).not.toHaveBeenCalled()
    expect(audio.instances).toHaveLength(1)
    expect(audio.instances[0].src).toBe('blob:question-audio')
    expect(audio.instances[0].play).toHaveBeenCalledTimes(1)

    act(() => audio.instances[0].onended?.())

    await waitFor(() => expect(media.getUserMedia).toHaveBeenCalledWith({ audio: true }))
    expect(screen.getByText('正在錄音')).toBeInTheDocument()
    expect(objectURLs.revokeObjectURL).toHaveBeenCalledWith('blob:question-audio')
  })

  it('prefetches all question audio when opening an in-progress session before playback', async () => {
    const speech = installSpeechSynthesisMock()
    const objectURLs = installObjectURLMock()
    objectURLs.createObjectURL.mockReturnValueOnce('blob:prefetched-question-audio')
    const audio = installAudioMock()
    mockPathname('/interviews/interview-123/session')
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            id: 'interview-123',
            job_title: '後端工程師',
            job_description: '需要熟悉 Go',
            user_profile: '有 Go 經驗',
            question_count: 2,
            question_language: 'zh-TW',
            status: 'in_progress',
            questions: [
              { id: 'question-1', order: 1, text: '問題一' },
              { id: 'question-2', order: 2, text: '問題二' },
            ],
            answers: [],
          }),
          { headers: { 'Content-Type': 'application/json' } },
        ),
      )
      .mockResolvedValueOnce(new Response('question-1-wav', { headers: { 'Content-Type': 'audio/wav' } }))
      .mockResolvedValueOnce(new Response('question-2-wav', { headers: { 'Content-Type': 'audio/wav' } }))
    vi.stubGlobal('fetch', fetchMock)

    render(<App />)

    expect(await screen.findByText('正在播放題目')).toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledWith('/api/interviews/interview-123/questions/question-1/tts', {
      method: 'POST',
    })
    expect(fetchMock).toHaveBeenCalledWith('/api/interviews/interview-123/questions/question-2/tts', {
      method: 'POST',
    })
    expect(speech.speak).not.toHaveBeenCalled()
    expect(audio.instances).toHaveLength(1)
    expect(audio.instances[0].src).toBe('blob:prefetched-question-audio')
  })

  it('shows a single loading message while the session prepares cached audio', async () => {
    installSpeechSynthesisMock()
    installObjectURLMock()
    installAudioMock()
    mockPathname('/interviews/interview-123/session')
    const ttsResponse = deferredResponse(new Response('question-1-wav', { headers: { 'Content-Type': 'audio/wav' } }))
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            id: 'interview-123',
            job_title: '後端工程師',
            job_description: '需要熟悉 Go',
            user_profile: '有 Go 經驗',
            question_count: 1,
            question_language: 'zh-TW',
            status: 'in_progress',
            questions: [{ id: 'question-1', order: 1, text: '問題一' }],
            answers: [],
          }),
          { headers: { 'Content-Type': 'application/json' } },
        ),
      )
      .mockReturnValueOnce(ttsResponse.promise)
    vi.stubGlobal('fetch', fetchMock)

    render(<App />)

    await waitFor(() =>
      expect(fetchMock).toHaveBeenCalledWith('/api/interviews/interview-123/questions/question-1/tts', {
        method: 'POST',
      }),
    )
    expect(screen.getAllByText('載入面試中...')).toHaveLength(1)

    ttsResponse.resolve()
    await act(async () => {})
  })

  it('plays cached prefetched question audio in the session without calling the TTS endpoint again', async () => {
    const speech = installSpeechSynthesisMock()
    const media = installMediaRecorderMock()
    const objectURLs = installObjectURLMock()
    objectURLs.createObjectURL.mockReturnValueOnce('blob:cached-question-audio')
    const audio = installAudioMock()
    storeQuestionAudio('interview-123', 'question-1', new Blob(['cached-wav'], { type: 'audio/wav' }))
    mockPathname('/interviews/interview-123/session')
    const fetchMock = mockFetchOnce({
      id: 'interview-123',
      job_title: '後端工程師',
      job_description: '需要熟悉 Go',
      user_profile: '有 Go 經驗',
      question_count: 1,
      question_language: 'zh-TW',
      status: 'in_progress',
      questions: [{ id: 'question-1', order: 1, text: '問題一' }],
      answers: [],
    })
    vi.stubGlobal('fetch', fetchMock)

    render(<App />)

    expect(await screen.findByText('正在播放題目')).toBeInTheDocument()
    expect(fetchMock).not.toHaveBeenCalledWith(
      '/api/interviews/interview-123/questions/question-1/tts',
      { method: 'POST' },
    )
    expect(speech.speak).not.toHaveBeenCalled()
    expect(audio.instances).toHaveLength(1)
    expect(audio.instances[0].src).toBe('blob:cached-question-audio')
    expect(audio.instances[0].play).toHaveBeenCalledTimes(1)

    act(() => audio.instances[0].onended?.())

    await waitFor(() => expect(media.getUserMedia).toHaveBeenCalledWith({ audio: true }))
    expect(objectURLs.revokeObjectURL).toHaveBeenCalledWith('blob:cached-question-audio')
  })

  it('replays the question during recording and discards the current recording', async () => {
    const speech = installSpeechSynthesisMock()
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
            job_description: '需要熟悉 Go',
            user_profile: '有 Go 經驗',
            question_count: 1,
            question_language: 'zh-TW',
            status: 'in_progress',
            questions: [{ id: 'question-1', order: 1, text: '問題一' }],
            answers: [],
          }),
          { headers: { 'Content-Type': 'application/json' } },
        ),
      )
      .mockResolvedValue(
        new Response(JSON.stringify({ error: 'question TTS is unavailable' }), {
          status: 503,
          headers: { 'Content-Type': 'application/json' },
        }),
      )
    vi.stubGlobal('fetch', fetchMock)

    render(<App />)
    expect(await screen.findByText('正在播放題目')).toBeInTheDocument()
    act(() => speech.utterances[0].onend?.())
    await waitFor(() => expect(media.recorders).toHaveLength(1))

    fireEvent.click(screen.getByRole('button', { name: '重新播放題目' }))

    expect(media.recorders[0].stop).toHaveBeenCalledTimes(1)
    expect(screen.queryByLabelText('回答錄音預覽')).not.toBeInTheDocument()
    await waitFor(() => expect(speech.speak).toHaveBeenCalledTimes(2))
  })

  it('queues uploads in the background and waits before opening the result page', async () => {
    const speech = installSpeechSynthesisMock()
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
            job_description: '需要熟悉 Go',
            user_profile: '有 Go 經驗',
            question_count: 1,
            question_language: 'zh-TW',
            status: 'in_progress',
            questions: [{ id: 'question-1', order: 1, text: '問題一' }],
            answers: [],
          }),
          { headers: { 'Content-Type': 'application/json' } },
        ),
      )
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ error: 'question TTS is unavailable' }), {
          status: 503,
          headers: { 'Content-Type': 'application/json' },
        }),
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
    vi.stubGlobal('fetch', fetchMock)

    render(<App />)
    expect(await screen.findByText('正在播放題目')).toBeInTheDocument()
    await waitFor(() => expect(speech.utterances).toHaveLength(1))
    act(() => speech.utterances[0].onend?.())
    await waitFor(() => expect(media.recorders).toHaveLength(1))

    fireEvent.click(screen.getByRole('button', { name: '回答結束' }))

    expect(await screen.findByText('正在完成面試')).toBeInTheDocument()
    await waitFor(() =>
      expect(fetchMock).toHaveBeenCalledWith(
        '/api/interviews/interview-123/questions/question-1/answer',
        expect.objectContaining({ method: 'POST', body: expect.any(FormData) }),
      ),
    )
    await waitFor(() => expect(window.location.pathname).toBe('/interviews/interview-123/result'))
  })

  it('loads the completed interview result page with playable answers', async () => {
    mockPathname('/interviews/interview-123/result')
    const jobDescription = '需要熟悉 Go、PostgreSQL、REST API\n能設計可維護的服務'
    const userProfile = '有 Java 和 Go 學習經驗\n正在準備後端工程師面試'
    vi.stubGlobal(
      'fetch',
      mockFetchOnce({
        id: 'interview-123',
        job_title: '後端工程師',
        job_description: jobDescription,
        user_profile: userProfile,
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
            analysis_status: 'pending',
            improvement_suggestions: null,
            analysis_error: null,
            analyzed_at: null,
            created_at: '2026-05-27T06:30:04Z',
          },
          {
            id: 'answer-2',
            question_id: 'question-2',
            audio_path: 'storage/audio/interview-123/question-2.webm',
            transcript_text: '我會先確認需求，再設計 resource 與錯誤格式。',
            analysis_status: 'completed',
            improvement_suggestions: '可以補充具體案例與取捨理由。',
            analysis_error: null,
            analyzed_at: '2026-05-27T06:32:04Z',
            created_at: '2026-05-27T06:31:04Z',
          },
        ],
      }),
    )

    render(<App />)

    expect(await screen.findByRole('heading', { name: '面試結果' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: '後端工程師' })).toBeInTheDocument()
    expect(screen.getByText('completed')).toBeInTheDocument()
    expect(screen.queryByText(jobDescription)).not.toBeInTheDocument()
    expect(screen.queryByText(userProfile)).not.toBeInTheDocument()
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: '職位資訊' }))

    expect(screen.getByRole('dialog', { name: '職位資訊' })).toBeInTheDocument()
    expect(findElementByExactText(jobDescription)).toHaveClass('whitespace-pre-wrap', 'break-words')
    fireEvent.mouseDown(screen.getByTestId('info-modal-backdrop'))
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: '個人資訊' }))
    expect(screen.getByRole('dialog', { name: '個人資訊' })).toBeInTheDocument()
    expect(findElementByExactText(userProfile)).toHaveClass('whitespace-pre-wrap', 'break-words')
    fireEvent.mouseDown(screen.getByTestId('info-modal-backdrop'))
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(screen.getByText('請介紹你過去與後端開發相關的經驗。')).toBeInTheDocument()
    expect(screen.getByText('你如何設計一個 REST API？')).toBeInTheDocument()
    expect(screen.getByText('AI 分析中')).toBeInTheDocument()
    expect(screen.getByText('我會先確認需求，再設計 resource 與錯誤格式。')).toBeInTheDocument()
    expect(screen.getByText('可以補充具體案例與取捨理由。')).toBeInTheDocument()
    expect(screen.getByLabelText('問題 1 回答音檔')).toHaveAttribute(
      'src',
      '/audio/interview-123/question-1.webm',
    )
    expect(screen.getByLabelText('問題 2 回答音檔')).toHaveAttribute(
      'src',
      '/audio/interview-123/question-2.webm',
    )
  })

  it('renders completed improvement suggestions as markdown with preserved line breaks', async () => {
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
        answers: [
          {
            id: 'answer-1',
            question_id: 'question-1',
            audio_path: 'storage/audio/interview-123/question-1.webm',
            transcript_text: '這是逐字稿。',
            analysis_status: 'completed',
            improvement_suggestions:
              '### 回答建議\n\n- **補充具體案例**：說明你負責的 API。\n- 量化成果。\n\n下一步\n請加入 PostgreSQL schema 設計取捨。',
            analysis_error: null,
            analyzed_at: '2026-05-27T06:32:04Z',
            created_at: '2026-05-27T06:30:04Z',
          },
        ],
      }),
    )

    render(<App />)

    expect(await screen.findByRole('heading', { name: '回答建議' })).toBeInTheDocument()
    expect(screen.getByText('補充具體案例').tagName.toLowerCase()).toBe('strong')
    expect(screen.getByText(/說明你負責的 API/).closest('li')).toBeInTheDocument()

    const nextStep = screen.getByText(/下一步/)
    const nextStepParagraph = nextStep.closest('p')
    expect(nextStepParagraph).toBeInTheDocument()
    expect(nextStepParagraph?.querySelector('br')).toBeInTheDocument()
  })

  it('polls the result page while answer analysis is processing', async () => {
    vi.useFakeTimers()
    mockPathname('/interviews/interview-123/result')
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            id: 'interview-123',
            job_title: '後端工程師',
            job_description: '需要熟悉 Go',
            user_profile: '有 Go 學習經驗',
            question_count: 1,
            status: 'completed',
            questions: [{ id: 'question-1', order: 1, text: '第一題' }],
            answers: [
              {
                id: 'answer-1',
                question_id: 'question-1',
                audio_path: 'storage/audio/interview-123/question-1.webm',
                transcript_text: null,
                analysis_status: 'processing',
                improvement_suggestions: null,
                analysis_error: null,
                analyzed_at: null,
                created_at: '2026-05-27T06:30:04Z',
              },
            ],
          }),
          { headers: { 'Content-Type': 'application/json' } },
        ),
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            id: 'interview-123',
            job_title: '後端工程師',
            job_description: '需要熟悉 Go',
            user_profile: '有 Go 學習經驗',
            question_count: 1,
            status: 'completed',
            questions: [{ id: 'question-1', order: 1, text: '第一題' }],
            answers: [
              {
                id: 'answer-1',
                question_id: 'question-1',
                audio_path: 'storage/audio/interview-123/question-1.webm',
                transcript_text: '這是逐字稿。',
                analysis_status: 'completed',
                improvement_suggestions: '回答可以更具體。',
                analysis_error: null,
                analyzed_at: '2026-05-27T06:32:04Z',
                created_at: '2026-05-27T06:30:04Z',
              },
            ],
          }),
          { headers: { 'Content-Type': 'application/json' } },
        ),
      )
    vi.stubGlobal('fetch', fetchMock)

    render(<App />)
    await act(async () => {})

    expect(screen.getByText('AI 分析中')).toBeInTheDocument()

    await vi.advanceTimersByTimeAsync(3000)
    await act(async () => {})

    expect(fetchMock).toHaveBeenCalledTimes(2)
    expect(screen.getByText('這是逐字稿。')).toBeInTheDocument()
    expect(screen.getByText('回答可以更具體。')).toBeInTheDocument()
    vi.useRealTimers()
  })

  it('does not start overlapping result polling requests', async () => {
    vi.useFakeTimers()
    mockPathname('/interviews/interview-123/result')
    const pendingAnswer = {
      id: 'answer-1',
      question_id: 'question-1',
      audio_path: 'storage/audio/interview-123/question-1.webm',
      transcript_text: null,
      analysis_status: 'processing',
      improvement_suggestions: null,
      analysis_error: null,
      analyzed_at: null,
      created_at: '2026-05-27T06:30:04Z',
    }
    const slowPollingResponse = deferredResponse(
      interviewDetailResponse({
        status: 'completed',
        questions: [{ id: 'question-1', order: 1, text: '第一題' }],
        answers: [pendingAnswer],
      }),
    )
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        interviewDetailResponse({
          status: 'completed',
          questions: [{ id: 'question-1', order: 1, text: '第一題' }],
          answers: [pendingAnswer],
        }),
      )
      .mockReturnValueOnce(slowPollingResponse.promise)
    vi.stubGlobal('fetch', fetchMock)

    render(<App />)
    await act(async () => {})

    expect(screen.getByText('AI 分析中')).toBeInTheDocument()

    await act(async () => {
      vi.advanceTimersByTime(6000)
    })

    expect(fetchMock).toHaveBeenCalledTimes(2)
    expect(screen.queryByText('載入面試結果中...')).not.toBeInTheDocument()

    slowPollingResponse.resolve()
    await act(async () => {})
    vi.useRealTimers()
  })

  it('shows failed answer analysis on the result page', async () => {
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
        answers: [
          {
            id: 'answer-1',
            question_id: 'question-1',
            audio_path: 'storage/audio/interview-123/question-1.webm',
            transcript_text: null,
            analysis_status: 'failed',
            improvement_suggestions: null,
            analysis_error: 'Gemini 分析失敗',
            analyzed_at: null,
            created_at: '2026-05-27T06:30:04Z',
          },
        ],
      }),
    )

    render(<App />)

    expect(await screen.findByText('分析失敗，請稍後重新上傳回答')).toBeInTheDocument()
    expect(screen.getByText('Gemini 分析失敗')).toBeInTheDocument()
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
