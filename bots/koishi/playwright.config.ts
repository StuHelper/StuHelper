import { defineConfig, devices } from '@playwright/test'

const PORT = Number.parseInt(process.env.STUHELPER_UI_SMOKE_PORT ?? '5140', 10)

export default defineConfig({
  testDir: './e2e',
  fullyParallel: false,
  forbidOnly: Boolean(process.env.CI),
  retries: 0,
  workers: 1,
  reporter: process.env.CI ? 'github' : 'list',
  timeout: 30_000,
  expect: {
    timeout: 10_000,
  },
  use: {
    baseURL: `http://127.0.0.1:${PORT}`,
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },
  projects: [
    {
      name: 'chromium',
      use: {
        ...devices['Desktop Chrome'],
        // viewport 写在 project use 块、放在 spread 之后，
        // 才能覆盖 devices['Desktop Chrome'] 默认的 1280×720
        viewport: { width: 1920, height: 1080 },
      },
    },
  ],
})
