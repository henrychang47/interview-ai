import react from '@vitejs/plugin-react'
import { defineConfig } from 'vitest/config'

declare const process: {
  env: Record<string, string | undefined>
}

const apiProxyTarget = process.env.VITE_API_PROXY_TARGET ?? 'http://localhost:8080'
const deployDomain = process.env.DEPLOY_DOMAIN?.trim()
const allowedHosts = deployDomain ? [deployDomain] : []

export default defineConfig({
  plugins: [react()],
  server: {
    host: '0.0.0.0',
    allowedHosts,
    port: 5173,
    proxy: {
      '/api': {
        target: apiProxyTarget,
        changeOrigin: true,
      },
      '/audio': {
        target: apiProxyTarget,
        changeOrigin: true,
      },
    },
  },
  test: {
    environment: 'jsdom',
    globals: false,
  },
})
