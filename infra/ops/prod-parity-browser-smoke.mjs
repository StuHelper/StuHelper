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
const criticalResourceTypes = new Set(['document', 'font', 'image', 'script', 'stylesheet']);

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
    name: 'web-about',
    url: joinURL(webBaseURL, '/about'),
    expectedTexts: ['关于 StuHelper', 'About StuHelper'],
  },
  {
    name: 'web-privacy',
    url: joinURL(webBaseURL, '/privacy'),
    expectedTexts: ['隐私政策', 'Privacy Policy'],
  },
  {
    name: 'web-terms',
    url: joinURL(webBaseURL, '/terms'),
    expectedTexts: ['服务条款', 'Terms of Service'],
  },
  {
    name: 'web-course-hub',
    url: joinURL(webBaseURL, '/courses'),
    expectedTexts: ['评课社区@BUAA', 'Browse Courses'],
  },
  {
    name: 'web-course-list',
    url: joinURL(webBaseURL, '/courses/list'),
    expectedTexts: ['课程列表', 'Course List'],
  },
  {
    name: 'web-course-about',
    url: joinURL(webBaseURL, '/courses/about'),
    expectedTexts: ['关于评课社区@BUAA'],
  },
  {
    name: 'web-review-feed',
    url: joinURL(webBaseURL, '/courses/reviews'),
    expectedTexts: ['最新', 'Latest'],
  },
  {
    name: 'web-search',
    url: joinURL(webBaseURL, '/search'),
    expectedTexts: ['高级搜索', 'Advanced Search'],
  },
  {
    name: 'web-teacher-hub',
    url: joinURL(webBaseURL, '/teachers'),
    expectedTexts: ['教师主页', 'Teacher'],
  },
  {
    name: 'web-protected-review-post',
    url: joinURL(webBaseURL, '/courses/reviews/post'),
    expectedTexts: ['登录', 'Login'],
    expectedURLIncludes: ['/login', 'redirect=/courses/reviews/post'],
  },
  {
    name: 'web-protected-user-reviews',
    url: joinURL(webBaseURL, '/user/reviews'),
    expectedTexts: ['登录', 'Login'],
    expectedURLIncludes: ['/login', 'redirect=/user/reviews'],
  },
  {
    name: 'web-protected-notifications',
    url: joinURL(webBaseURL, '/notifications'),
    expectedTexts: ['登录', 'Login'],
    expectedURLIncludes: ['/login', 'redirect=/notifications'],
  },
  {
    name: 'web-protected-developer-apps',
    url: joinURL(webBaseURL, '/developers/apps'),
    expectedTexts: ['登录', 'Login'],
    expectedURLIncludes: ['/login', 'redirect=/developers/apps'],
  },
  {
    name: 'web-protected-open-platform-consent',
    url: joinURL(webBaseURL, '/consent?token=smoke'),
    expectedTexts: ['登录', 'Login'],
    expectedURLIncludes: ['/login', 'redirect=/consent?token=smoke'],
  },
  {
    name: 'web-protected-profile-completion',
    url: joinURL(webBaseURL, '/complete-profile?token=smoke'),
    expectedTexts: ['登录', 'Login'],
    expectedURLIncludes: ['/login', 'redirect=/complete-profile?token=smoke'],
  },
  {
    name: 'web-not-found',
    url: joinURL(webBaseURL, '/this-route-does-not-exist'),
    expectedTexts: ['页面不存在', 'Page Not Found'],
  },
  {
    name: 'admin-login-redirect',
    url: joinURL(adminBaseURL, '/admin/'),
    expectedTexts: ['Sign In', 'Password', '登录'],
    expectedURLIncludes: '/login/oauth/authorize',
  },
];

const results = [];
let passed = false;
let browser;

try {
  await mkdir(screenshotDir, { recursive: true });
  browser = await chromium.launch({ headless: process.env.PLAYWRIGHT_HEADLESS !== '0' });

  for (const check of checks) {
    results.push(await runCheck(browser, check));
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

async function runCheck(browser, check) {
  const context = await browser.newContext();
  const page = await context.newPage();
  const assetFailures = [];
  const apiFailures = [];
  const pageErrors = [];
  const screenshotFile = resolve(screenshotDir, `${check.name}.png`);

  page.on('pageerror', (error) => {
    pageErrors.push(error.message);
  });
  page.on('requestfailed', (request) => {
    if (criticalResourceTypes.has(request.resourceType())) {
      assetFailures.push({
        url: request.url(),
        resourceType: request.resourceType(),
        failure: request.failure()?.errorText || 'unknown',
      });
    }
  });
  page.on('response', (response) => {
    const request = response.request();
    const resourceType = request.resourceType();
    if (
      ['fetch', 'xhr'].includes(resourceType) &&
      response.status() >= 500
    ) {
      apiFailures.push({
        url: response.url(),
        resourceType,
        status: response.status(),
        statusText: response.statusText(),
      });
    }
    if (
      criticalResourceTypes.has(resourceType) &&
      response.status() >= 400
    ) {
      assetFailures.push({
        url: response.url(),
        resourceType,
        status: response.status(),
        statusText: response.statusText(),
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

    const expectedURLIncludes = toArray(check.expectedURLIncludes);
    if (expectedURLIncludes.length > 0) {
      await page.waitForURL(
        (url) => expectedURLIncludes.every((text) => url.href.includes(text)),
        { timeout: timeoutMs },
      );
    }

    await page.waitForLoadState('networkidle', { timeout: 5000 }).catch(() => undefined);
    const title = await page.title();
    const bodyText = await page.locator('body').innerText({ timeout: timeoutMs });
    const matchedText = check.expectedTexts.find((text) => bodyText.includes(text));

    if (!matchedText) {
      throw new Error(
        `missing expected text; expected one of: ${check.expectedTexts.join(', ')}`,
      );
    }
    if (
      expectedURLIncludes.length > 0 &&
      expectedURLIncludes.some((text) => !page.url().includes(text))
    ) {
      throw new Error(
        `final URL ${page.url()} does not include ${expectedURLIncludes.join(', ')}`,
      );
    }
    if (pageErrors.length > 0) {
      throw new Error(`page errors: ${pageErrors.join(' | ')}`);
    }
    if (assetFailures.length > 0) {
      throw new Error(`asset failures: ${JSON.stringify(assetFailures)}`);
    }
    if (apiFailures.length > 0) {
      throw new Error(`api failures: ${JSON.stringify(apiFailures)}`);
    }

    await page.screenshot({ path: screenshotFile, fullPage: true });

    return {
      name: check.name,
      passed: true,
      url: check.url,
      finalURL: page.url(),
      status: response.status(),
      title,
      matchedText,
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
      apiFailures,
      screenshot: relative(repoRoot, screenshotFile),
    };
  } finally {
    await context.close();
  }
}

function normalizeBaseURL(value) {
  return value.replace(/\/+$/, '');
}

function joinURL(base, path) {
  return `${normalizeBaseURL(base)}${path}`;
}

function toArray(value) {
  if (Array.isArray(value)) return value;
  if (typeof value === 'string') return [value];
  return [];
}
