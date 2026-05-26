import '@testing-library/jest-dom/vitest'
import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import App from './App'

describe('App', () => {
  it('renders the interview practice homepage', () => {
    render(<App />)

    expect(screen.getByRole('heading', { name: '模擬面試應用' })).toBeInTheDocument()
    expect(screen.getByText('建立面試、產生題目、錄音回答，逐步打通 MVP 主流程。')).toBeInTheDocument()
  })
})
