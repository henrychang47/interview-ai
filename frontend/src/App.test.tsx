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
        answers: [],
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
        answers: [],
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
    expect(screen.getByRole('button', { name: '下一題' })).toBeDisabled()

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
