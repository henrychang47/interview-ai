// @vitest-environment node

import { describe, expect, it } from 'vitest'

import config from '../vite.config.ts'

describe('vite dev proxy', () => {
  it('proxies uploaded answer audio requests to the backend', () => {
    const proxy = config.server?.proxy

    expect(proxy).toMatchObject({
      '/audio': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    })
  })
})
