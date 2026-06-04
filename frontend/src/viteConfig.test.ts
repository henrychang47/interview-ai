// @vitest-environment node

import { afterEach, describe, expect, it, vi } from 'vitest'

async function loadConfig(name: string) {
  void name
  vi.resetModules()
  return (await import('../vite.config.ts')).default
}

afterEach(() => {
  vi.unstubAllEnvs()
})

describe('vite dev proxy', () => {
  it('proxies uploaded answer audio requests to the backend', async () => {
    const config = await loadConfig('default-proxy')
    const proxy = config.server?.proxy

    expect(proxy).toMatchObject({
      '/audio': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    })
  })

  it('does not allow extra hosts when DEPLOY_DOMAIN is unset', async () => {
    vi.stubEnv('DEPLOY_DOMAIN', '')

    const config = await loadConfig('no-deploy-domain')

    expect(config.server?.allowedHosts).toEqual([])
  })

  it('allows the configured deploy domain', async () => {
    vi.stubEnv('DEPLOY_DOMAIN', 'interview.henry.christmas')

    const config = await loadConfig('deploy-domain')

    expect(config.server?.allowedHosts).toEqual(['interview.henry.christmas'])
  })
})
