// @vitest-environment node

import { describe, expect, it } from 'vitest'

import packageJSON from '../package.json'

describe('frontend scripts', () => {
  it('loads Vite tools from the TypeScript config file', () => {
    expect(packageJSON.scripts.dev).toContain('--config vite.config.ts')
    expect(packageJSON.scripts.build).toContain('vite build --config vite.config.ts')
    expect(packageJSON.scripts.test).toContain('--config vite.config.ts')
  })
})
