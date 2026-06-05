// @vitest-environment node

import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { resolve } from 'node:path'

import { describe, expect, it } from 'vitest'

const frontendRoot = resolve(fileURLToPath(new URL('..', import.meta.url)))
const repoRoot = resolve(frontendRoot, '..')

function readRepoFile(path: string) {
  return readFileSync(resolve(repoRoot, path), 'utf8')
}

describe('production frontend deployment config', () => {
  it('keeps the local frontend Dockerfile on the Vite dev server flow', () => {
    const dockerfile = readRepoFile('frontend/Dockerfile')

    expect(dockerfile).toContain('FROM node:20-alpine')
    expect(dockerfile).toContain('RUN npm install')
    expect(dockerfile).toContain('EXPOSE 5173')
    expect(dockerfile).toContain('"npm", "run", "dev"')
    expect(dockerfile).not.toContain('FROM nginx')
  })

  it('builds the production Caddy image with Vite production assets', () => {
    const dockerfile = readRepoFile('Dockerfile.caddy')

    expect(dockerfile).toContain('FROM node:20-alpine AS frontend-build')
    expect(dockerfile).toContain('RUN npm ci')
    expect(dockerfile).toContain('RUN npm run build')
    expect(dockerfile).toContain('FROM caddy:2-alpine')
    expect(dockerfile).toContain('COPY Caddyfile /etc/caddy/Caddyfile')
    expect(dockerfile).toContain('COPY --from=frontend-build /app/frontend/dist /srv')
  })

  it('deploys without a frontend runtime service', () => {
    const compose = readRepoFile('docker-compose.deploy.yml')

    expect(compose).toContain('dockerfile: Dockerfile.caddy')
    expect(compose).not.toContain('frontend:')
    expect(compose).not.toContain('frontend:80')
    expect(compose).not.toContain('frontend:5173')
  })

  it('serves hashed assets directly and falls back SPA routes to index.html from Caddy', () => {
    const caddyfile = readRepoFile('Caddyfile')

    expect(caddyfile).toContain('root * /srv')
    expect(caddyfile).toContain('handle /assets/*')
    expect(caddyfile).toContain('Cache-Control "public, max-age=31536000, immutable"')
    expect(caddyfile).toContain('try_files {path} /index.html')
    expect(caddyfile).not.toContain('reverse_proxy frontend')
  })
})
