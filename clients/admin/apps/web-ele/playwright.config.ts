import path from 'node:path';
import { fileURLToPath } from 'node:url';

import { defineConfig } from '@playwright/test';

const workspaceRoot = path.resolve(
  fileURLToPath(new URL('../..', import.meta.url)),
);
const host = process.env.ADMIN_E2E_HOST || '127.0.0.1';
const port = Number(process.env.ADMIN_E2E_PORT || '4174');
const baseURL = `http://${host}:${port}/admin`;

export default defineConfig({
  testDir: './tests/e2e',
  timeout: 30_000,
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
  webServer: {
    command: `cd ${JSON.stringify(workspaceRoot)} && pnpm -F @vben/web-ele exec vite --mode development --host ${host} --port ${port}`,
    env: {
      ...process.env,
      VITE_DEV_PROXY_TARGET:
        process.env.VITE_DEV_PROXY_TARGET ?? 'http://127.0.0.1:8080',
    },
    reuseExistingServer: process.env.PLAYWRIGHT_REUSE_SERVER === '1',
    url: `${baseURL}/auth/login`,
  },
});
