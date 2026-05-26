import { defineConfig, devices } from '@playwright/test'

delete process.env.NO_COLOR

const pnpmCommand = process.env.PLAYWRIGHT_PNPM_COMMAND ?? 'corepack pnpm'
const host = process.env.UNIAPPX_E2E_HOST ?? '127.0.0.1'
const port = Number(process.env.UNIAPPX_E2E_PORT ?? 3132)
const origin = `http://${host}:${port}`
const webServerEnv = withoutNoColorEnv({
  VITE_API_URL: '',
  VITE_WEB_URL: process.env.UNIAPPX_E2E_WEB_URL ?? 'https://web.example.test',
})

function withoutNoColorEnv(overrides: Record<string, string>) {
  const env: Record<string, string> = {}
  for (const [key, value] of Object.entries(process.env)) {
    if (key !== 'NO_COLOR' && typeof value === 'string') {
      env[key] = value
    }
  }
  return { ...env, ...overrides }
}

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
    env: webServerEnv,
    reuseExistingServer: process.env.PLAYWRIGHT_REUSE_SERVER === '1',
    url: origin,
  },
})
