import { defineConfig, devices } from '@playwright/test'

const pnpmCommand = process.env.PLAYWRIGHT_PNPM_COMMAND ?? 'corepack pnpm'
const host = process.env.UNIAPPX_E2E_HOST ?? '127.0.0.1'
const port = Number(process.env.UNIAPPX_E2E_PORT ?? 3132)
const origin = `http://${host}:${port}`

export default defineConfig({
  testDir: './tests/e2e',
  timeout: 30_000,
  use: {
    baseURL: origin,
    locale: 'zh-CN',
    trace: 'on-first-retry',
  },
  projects: [
    {
      name: 'desktop-chromium',
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'mobile-chromium',
      use: { ...devices['Pixel 5'] },
    },
  ],
  webServer: {
    command: `${pnpmCommand} dev:h5 --host ${host} --port ${port}`,
    env: {
      ...process.env,
      VITE_API_URL: '',
    },
    reuseExistingServer: process.env.PLAYWRIGHT_REUSE_SERVER === '1',
    url: origin,
  },
})
