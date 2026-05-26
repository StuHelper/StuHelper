import path from 'node:path';
import { fileURLToPath } from 'node:url';

import { defineConfig, devices } from '@playwright/test';

delete process.env.NO_COLOR;

const workspaceRoot = path.resolve(
  fileURLToPath(new URL('../..', import.meta.url)),
);
const host = process.env.ADMIN_E2E_HOST || '127.0.0.1';
const port = Number(process.env.ADMIN_E2E_PORT || '4174');
const baseURL = `http://${host}:${port}/admin`;
// Admin E2E targets the built SPA instead of Vite's dev server. The Vben
// workspace has many lazy chunks; testing the preview build avoids dev-time
// transform/HMR network churn while keeping the browser path production-like.
const workers = Number(process.env.ADMIN_E2E_WORKERS || '1');
const webServerCommand = [
  `cd ${JSON.stringify(workspaceRoot)}`,
  'pnpm -F @vben/web-ele exec vite build --mode production',
  `pnpm -F @vben/web-ele exec vite preview --host ${host} --port ${port}`,
].join(' && ');
const webServerEnv = withoutNoColorEnv({
  VITE_E2E_API_STUB: '1',
  VITE_DEV_PROXY_TARGET:
    process.env.VITE_DEV_PROXY_TARGET ?? 'http://127.0.0.1:8080',
});

function withoutNoColorEnv(overrides: Record<string, string>) {
  const env: Record<string, string> = {};
  for (const [key, value] of Object.entries(process.env)) {
    if (key !== 'NO_COLOR' && typeof value === 'string') {
      env[key] = value;
    }
  }
  return { ...env, ...overrides };
}

export default defineConfig({
  testDir: './tests/e2e',
  timeout: 30_000,
  workers,
  outputDir: './test-results',
  reporter: [
    ['list'],
    ['html', { outputFolder: './playwright-report', open: 'never' }],
  ],
  use: {
    baseURL,
    headless: !!process.env.CI,
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
    command: webServerCommand,
    env: webServerEnv,
    reuseExistingServer: process.env.PLAYWRIGHT_REUSE_SERVER === '1',
    timeout: 180_000,
    url: `${baseURL}/auth/login`,
  },
});
