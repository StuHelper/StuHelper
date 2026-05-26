#!/usr/bin/env node
import { createRequire } from 'node:module';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const scriptDir = dirname(fileURLToPath(import.meta.url));
const repoRoot = resolve(scriptDir, '../..');
const requireFromWeb = createRequire(resolve(repoRoot, 'clients/web/package.json'));

let chromium;
try {
  ({ chromium } = requireFromWeb('@playwright/test'));
} catch (error) {
  console.error('[dev-browser-smoke] failed to load @playwright/test.');
  console.error('Run make dev-up or pnpm --dir clients install first.');
  console.error(error instanceof Error ? error.message : String(error));
  process.exit(1);
}

const webBaseURL = normalizeBaseURL(process.env.WEB_BASE_URL || 'http://localhost:3000');
const adminBaseURL = normalizeBaseURL(process.env.ADMIN_BASE_URL || 'http://localhost:3001');
const adminPath = process.env.ADMIN_SMOKE_PATH || '/admin/';
const timeoutMs = Number(process.env.DEV_BROWSER_SMOKE_TIMEOUT_MS || 15000);
const criticalResourceTypes = new Set(['document', 'font', 'image', 'script', 'stylesheet']);

const checks = [
  {
    name: 'web-home',
    url: joinURL(webBaseURL, '/'),
    expectedTexts: ['StuHelper'],
  },
  {
    name: 'web-courses',
    url: joinURL(webBaseURL, '/courses'),
    expectedTexts: ['评课社区@BUAA', 'Browse Courses'],
  },
  {
    name: 'admin-entry',
    url: joinURL(adminBaseURL, adminPath),
    expectedTexts: ['Sign In', '登录', 'StuHelper Admin'],
    allowNavigationAbortedResourceRequests: true,
    allowedAPIResponses: [
      {
        path: '/api/v1/auth/me',
        statuses: [401],
      },
    ],
  },
];

const browser = await chromium.launch({ headless: true });
const results = [];

try {
  for (const check of checks) {
    results.push(await runCheck(browser, check));
  }
} finally {
  await browser.close();
}

const failures = results.filter((result) => result.status === 'failed');
for (const result of results) {
  const icon = result.status === 'passed' ? '✅' : '❌';
  console.log(`${icon} ${result.name} ${result.url}`);
  if (result.status === 'failed') {
    for (const failure of result.failures) {
      console.error(`   - ${failure}`);
    }
  }
}

if (failures.length > 0) {
  console.error(`[dev-browser-smoke] ${failures.length}/${results.length} checks failed`);
  process.exit(1);
}

console.log(`[dev-browser-smoke] ${results.length} browser checks passed`);

async function runCheck(browserInstance, check) {
  const context = await browserInstance.newContext({
    viewport: { width: 1366, height: 900 },
  });
  const page = await context.newPage();
  page.setDefaultTimeout(timeoutMs);

  const failures = [];
  const allowedAPIResponses = check.allowedAPIResponses || [];

  page.on('pageerror', (error) => {
    failures.push(`pageerror: ${error.stack || error.message}`);
  });
  page.on('requestfailed', (request) => {
    if (!criticalResourceTypes.has(request.resourceType())) {
      return;
    }
    if (isAllowedFailedCriticalResource(request, check)) {
      return;
    }
    failures.push(
      `${request.resourceType()} ${request.method()} ${request.url()} ${request.failure()?.errorText || 'failed'}`,
    );
  });
  page.on('response', (response) => {
    const request = response.request();
    if (isAllowedAPIResponse(response, allowedAPIResponses)) {
      return;
    }
    if (criticalResourceTypes.has(request.resourceType()) && response.status() >= 400) {
      failures.push(
        `${request.resourceType()} ${request.method()} ${response.url()} HTTP ${response.status()}`,
      );
    }
    if (isAPIRequest(request) && response.status() >= 400) {
      failures.push(
        `api ${request.method()} ${response.url()} HTTP ${response.status()}`,
      );
    }
  });
  page.on('console', (message) => {
    if (message.type() !== 'error') {
      return;
    }
    if (isAllowedNetworkStatusConsoleError(message.text(), allowedAPIResponses)) {
      return;
    }
    failures.push(`console.error: ${message.text()}`);
  });

  try {
    await page.goto(check.url, { waitUntil: 'domcontentloaded', timeout: timeoutMs });
    await page.waitForLoadState('networkidle', { timeout: timeoutMs }).catch(() => undefined);
    await page.waitForTimeout(500);

    const bodyText = await page.locator('body').innerText({ timeout: timeoutMs });
    if (!bodyText.trim()) {
      failures.push('body text is empty');
    }
    if (!check.expectedTexts.some((text) => bodyText.includes(text))) {
      failures.push(`body text did not include any expected marker: ${check.expectedTexts.join(' | ')}`);
    }
  } catch (error) {
    failures.push(error instanceof Error ? error.message : String(error));
  } finally {
    await context.close();
  }

  return {
    failures,
    name: check.name,
    status: failures.length === 0 ? 'passed' : 'failed',
    url: check.url,
  };
}

function isAPIRequest(request) {
  const resourceType = request.resourceType();
  if (resourceType !== 'fetch' && resourceType !== 'xhr' && resourceType !== 'eventsource') {
    return false;
  }
  return new URL(request.url()).pathname.startsWith('/api/v1/');
}

function isAllowedAPIResponse(response, allowedResponses) {
  const url = new URL(response.url());
  return allowedResponses.some((allowed) =>
    url.pathname === allowed.path && allowed.statuses.includes(response.status()),
  );
}

function isAllowedNetworkStatusConsoleError(text, allowedResponses) {
  if (!/^Failed to load resource: the server responded with a status of [45]\d\d \([^)]+\)$/.test(text)) {
    return false;
  }
  return allowedResponses.length > 0;
}

function isAllowedFailedCriticalResource(request, check) {
  if (!check.allowNavigationAbortedResourceRequests) {
    return false;
  }
  if (request.resourceType() === 'document') {
    return false;
  }
  if (request.failure()?.errorText !== 'net::ERR_ABORTED') {
    return false;
  }
  return new URL(request.url()).origin === new URL(check.url).origin;
}

function normalizeBaseURL(value) {
  return value.replace(/\/+$/, '');
}

function joinURL(base, path) {
  const normalizedPath = path.startsWith('/') ? path : `/${path}`;
  return `${normalizeBaseURL(base)}${normalizedPath}`;
}
