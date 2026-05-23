import { defineConfig } from '@playwright/test'

const pnpmCommand = process.env.PLAYWRIGHT_PNPM_COMMAND ?? 'corepack pnpm'

export default defineConfig({
  testDir: './tests/e2e',
  timeout: 30_000,
  use: {
    baseURL: 'http://127.0.0.1:3000',
    trace: 'on-first-retry'
  },
  webServer: {
    command: `${pnpmCommand} dev --host 127.0.0.1 --port 3000`,
    url: 'http://127.0.0.1:3000',
    reuseExistingServer: !process.env.CI,
    env: {
      ...process.env,
      VITE_SSO_URL: process.env.VITE_SSO_URL ?? 'http://localhost:8085',
      VITE_E2E_API_STUB: '1',
    },
  }
})
