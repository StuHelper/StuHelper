import { defineConfig, devices } from '@playwright/test'

delete process.env.NO_COLOR

const pnpmCommand = process.env.PLAYWRIGHT_PNPM_COMMAND ?? 'corepack pnpm'
const webPort = Number(process.env.PLAYWRIGHT_WEB_PORT ?? 3000)
const webHost = process.env.PLAYWRIGHT_WEB_HOST ?? '127.0.0.1'
const webOrigin = `http://${webHost}:${webPort}`
const webServerEnv = withoutNoColorEnv({
  VITE_SSO_URL: process.env.VITE_SSO_URL ?? 'http://localhost:8085',
  VITE_E2E_API_STUB: '1',
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
    baseURL: webOrigin,
    trace: 'on-first-retry'
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
    command: `${pnpmCommand} dev --host ${webHost} --port ${webPort}`,
    url: webOrigin,
    reuseExistingServer: process.env.PLAYWRIGHT_REUSE_SERVER === '1',
    env: webServerEnv,
  }
})
