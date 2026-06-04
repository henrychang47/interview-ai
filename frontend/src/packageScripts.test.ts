// @vitest-environment node

import { describe, expect, it } from 'vitest'

import packageJSON from '../package.json'
import tsconfig from '../tsconfig.json'

describe('frontend scripts', () => {
  it('loads Vite tools from the TypeScript config file', () => {
    expect(packageJSON.scripts.dev).toContain('--config vite.config.ts')
    expect(packageJSON.scripts.build).toContain('vite build --config vite.config.ts')
    expect(packageJSON.scripts.test).toContain('--config vite.config.ts')
  })
})

describe('frontend TypeScript config', () => {
  it('keeps Vitest files out of the production typecheck', () => {
    expect(tsconfig.exclude).toEqual(
      expect.arrayContaining(['src/**/*.test.ts', 'src/**/*.test.tsx']),
    )
  })
})
