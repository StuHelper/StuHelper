import path from 'node:path';
import { fileURLToPath } from 'node:url';

import { defineConfig } from '@playwright/test';

const workspaceRoot = path.resolve(
  fileURLToPath(new URL('../..', import.meta.url)),
);
const port = Number(process.env.ADMIN_E2E_PORT || '4174');
const baseURL = `http://127.0.0.1:${port}/admin`;

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
    command: `cd ${JSON.stringify(workspaceRoot)} && VITE_DEV_PROXY_TARGET=http://127.0.0.1:8080 pnpm -F @vben/web-ele exec vite --mode development --host 127.0.0.1 --port ${port}`,
    reuseExistingServer: !process.env.CI,
    url: `${baseURL}/auth/login`,
  },
});
