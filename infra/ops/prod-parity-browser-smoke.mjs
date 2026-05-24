#!/usr/bin/env node
import { mkdir, writeFile } from 'node:fs/promises';
import { dirname, relative, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { createRequire } from 'node:module';

const scriptDir = dirname(fileURLToPath(import.meta.url));
const repoRoot = resolve(scriptDir, '../..');
const requireFromWeb = createRequire(resolve(repoRoot, 'clients/web/package.json'));

let chromium;
try {
  ({ chromium } = requireFromWeb('@playwright/test'));
} catch (error) {
  console.error('[prod-parity-browser-smoke] failed to load @playwright/test.');
  console.error('Run infra/ops/bootstrap-dev-ubuntu2404.sh or install clients dependencies first.');
  console.error(error instanceof Error ? error.message : String(error));
  process.exit(1);
}

const timeoutMs = Number(process.env.PROD_PARITY_BROWSER_SMOKE_TIMEOUT_MS || 30000);
const webBaseURL = normalizeBaseURL(process.env.WEB_BASE_URL || 'http://127.0.0.1:28000');
const adminBaseURL = normalizeBaseURL(process.env.ADMIN_BASE_URL || 'http://127.0.0.1:28001');
const evidenceFile =
  process.env.PROD_PARITY_BROWSER_SMOKE_EVIDENCE_FILE ||
  resolve(repoRoot, '.run/prod-parity/browser-smoke-evidence.json');
const screenshotDir =
  process.env.PROD_PARITY_BROWSER_SMOKE_SCREENSHOT_DIR || dirname(evidenceFile);

const checks = [
  {
    name: 'web-home',
    url: joinURL(webBaseURL, '/'),
    expectedTexts: ['StuHelper'],
  },
  {
    name: 'web-login',
    url: joinURL(webBaseURL, '/login'),
    expectedTexts: ['StuHelper'],
  },
  {
    name: 'admin-login-redirect',
    url: joinURL(adminBaseURL, '/admin/'),
    expectedTexts: ['Sign In', 'Password', '登录'],
  },
];

const results = [];
let passed = false;
let browser;

try {
  await mkdir(screenshotDir, { recursive: true });
  browser = await chromium.launch({ headless: process.env.PLAYWRIGHT_HEADLESS !== '0' });
  const context = await browser.newContext();

  for (const check of checks) {
    results.push(await runCheck(context, check));
  }

  passed = results.every((result) => result.passed);
} catch (error) {
  results.push({
    name: 'fatal',
    passed: false,
    error: error instanceof Error ? error.message : String(error),
  });
} finally {
  if (browser) {
    await browser.close();
  }

  const evidence = {
    generatedAt: new Date().toISOString(),
    passed,
    webBaseURL,
    adminBaseURL,
    checks: results,
  };
  await writeFile(evidenceFile, `${JSON.stringify(evidence, null, 2)}\n`);
}

if (!passed) {
  console.error(`[prod-parity-browser-smoke] failed; evidence: ${evidenceFile}`);
  process.exit(1);
}

console.log(`[prod-parity-browser-smoke] passed; evidence: ${evidenceFile}`);

async function runCheck(context, check) {
  const page = await context.newPage();
  const assetFailures = [];
  const pageErrors = [];
  const screenshotFile = resolve(screenshotDir, `${check.name}.png`);

  page.on('pageerror', (error) => {
    pageErrors.push(error.message);
  });
  page.on('requestfailed', (request) => {
    if (['document', 'script', 'stylesheet'].includes(request.resourceType())) {
      assetFailures.push({
        url: request.url(),
        resourceType: request.resourceType(),
        failure: request.failure()?.errorText || 'unknown',
      });
    }
  });

  try {
    const response = await page.goto(check.url, {
      timeout: timeoutMs,
      waitUntil: 'domcontentloaded',
    });

    if (!response) {
      throw new Error(`no response for ${check.url}`);
    }
    if (response.status() >= 400) {
      throw new Error(`unexpected HTTP ${response.status()} for ${check.url}`);
    }

    await page.waitForLoadState('networkidle', { timeout: 5000 }).catch(() => undefined);
    const title = await page.title();
    const bodyText = await page.locator('body').innerText({ timeout: timeoutMs });
    const hasExpectedText = check.expectedTexts.some((text) => bodyText.includes(text));

    if (!hasExpectedText) {
      throw new Error(
        `missing expected text; expected one of: ${check.expectedTexts.join(', ')}`,
      );
    }
    if (pageErrors.length > 0) {
      throw new Error(`page errors: ${pageErrors.join(' | ')}`);
    }
    if (assetFailures.length > 0) {
      throw new Error(`asset failures: ${JSON.stringify(assetFailures)}`);
    }

    await page.screenshot({ path: screenshotFile, fullPage: true });

    return {
      name: check.name,
      passed: true,
      url: check.url,
      finalURL: page.url(),
      status: response.status(),
      title,
      screenshot: relative(repoRoot, screenshotFile),
    };
  } catch (error) {
    await page.screenshot({ path: screenshotFile, fullPage: true }).catch(() => undefined);
    return {
      name: check.name,
      passed: false,
      url: check.url,
      finalURL: page.url(),
      error: error instanceof Error ? error.message : String(error),
      pageErrors,
      assetFailures,
      screenshot: relative(repoRoot, screenshotFile),
    };
  } finally {
    await page.close();
  }
}

function normalizeBaseURL(value) {
  return value.replace(/\/+$/, '');
}

function joinURL(base, path) {
  return `${normalizeBaseURL(base)}${path}`;
}
