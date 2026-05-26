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
const admissionToken = process.env.PROD_PARITY_ADMISSION_TOKEN || 'PROD-PARITY-ADMIT-LOGIN';
const admissionQQ = process.env.PROD_PARITY_ADMISSION_QQ || '990001';
const evidenceFile =
  process.env.PROD_PARITY_BROWSER_SMOKE_EVIDENCE_FILE ||
  resolve(repoRoot, '.run/prod-parity/browser-smoke-evidence.json');
const screenshotDir =
  process.env.PROD_PARITY_BROWSER_SMOKE_SCREENSHOT_DIR || dirname(evidenceFile);
const criticalResourceTypes = new Set(['document', 'font', 'image', 'script', 'stylesheet']);
const telemetryRoutePattern = /\/api\/v1\/metrics\/(?:frontend-errors|vitals)(?:\?|$)/;
const viewportVariants = [
  {
    name: 'desktop',
    viewport: { width: 1365, height: 900 },
    deviceScaleFactor: 1,
    isMobile: false,
    hasTouch: false,
  },
  {
    name: 'mobile',
    viewport: { width: 390, height: 844 },
    deviceScaleFactor: 2,
    isMobile: true,
    hasTouch: true,
  },
];

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
    name: 'web-auth-callback-missing-code',
    url: joinURL(webBaseURL, '/auth/callback?state=prod-parity-smoke'),
    expectedTexts: ['登录失败', 'Login Failed'],
    requiredTexts: ['缺少授权码'],
  },
  {
    name: 'web-admission-login',
    url: joinURL(
      webBaseURL,
      `/admission/a/${encodeURIComponent(admissionToken)}?qq=${encodeURIComponent(admissionQQ)}`,
    ),
    expectedTexts: ['登录 StuHelper'],
    requiredTexts: ['入群身份认证', `QQ：${admissionQQ}`],
    allowedAPIResponses: [
      {
        urlIncludes: '/api/v1/auth/me',
        statuses: [401],
      },
      {
        urlIncludes: '/api/v1/auth/refresh',
        statuses: [401],
      },
    ],
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
    requiredTexts: ['生产等价课程'],
  },
  {
    name: 'web-course-about',
    url: joinURL(webBaseURL, '/courses/about'),
    expectedTexts: ['关于评课社区@BUAA'],
  },
  {
    name: 'web-course-detail',
    url: joinURL(webBaseURL, '/courses/900001'),
    expectedTexts: ['生产等价课程'],
    requiredTexts: ['生产等价课程', '生产等价学院'],
  },
  {
    name: 'web-course-detail-reviews',
    url: joinURL(webBaseURL, '/courses/900001/reviews'),
    expectedTexts: ['生产等价课程'],
    requiredTexts: ['生产等价课程', '生产等价评课'],
  },
  {
    name: 'web-review-feed',
    url: joinURL(webBaseURL, '/courses/reviews'),
    expectedTexts: ['最新', 'Latest'],
    requiredTexts: ['生产等价评课'],
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
    requiredTexts: ['生产等价教师'],
  },
  {
    name: 'web-teacher-profile',
    url: joinURL(webBaseURL, '/teachers/900001'),
    expectedTexts: ['生产等价教师'],
    requiredTexts: ['生产等价教师', '生产等价课程'],
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
    name: 'web-protected-user-votes',
    url: joinURL(webBaseURL, '/user/votes'),
    expectedTexts: ['登录', 'Login'],
    expectedURLIncludes: ['/login', 'redirect=/user/votes'],
  },
  {
    name: 'web-protected-user-favorites',
    url: joinURL(webBaseURL, '/user/favorites'),
    expectedTexts: ['登录', 'Login'],
    expectedURLIncludes: ['/login', 'redirect=/user/favorites'],
  },
  {
    name: 'web-protected-user-authorized-apps',
    url: joinURL(webBaseURL, '/user/authorized-apps'),
    expectedTexts: ['登录', 'Login'],
    expectedURLIncludes: ['/login', 'redirect=/user/authorized-apps'],
  },
  {
    name: 'web-protected-identity-verification',
    url: joinURL(webBaseURL, '/user/identity-verification'),
    expectedTexts: ['登录', 'Login'],
    expectedURLIncludes: ['/login', 'redirect=/user/identity-verification'],
  },
  {
    name: 'web-protected-student-verification',
    url: joinURL(webBaseURL, '/user/student-verification'),
    expectedTexts: ['登录', 'Login'],
    expectedURLIncludes: ['/login', 'redirect=/user/student-verification'],
  },
  {
    name: 'web-protected-phone-binding',
    url: joinURL(webBaseURL, '/user/phone-binding'),
    expectedTexts: ['登录', 'Login'],
    expectedURLIncludes: ['/login', 'redirect=/user/phone-binding'],
  },
  {
    name: 'web-protected-qq-binding',
    url: joinURL(webBaseURL, '/user/qq-binding'),
    expectedTexts: ['登录', 'Login'],
    expectedURLIncludes: ['/login', 'redirect=/user/qq-binding'],
  },
  {
    name: 'web-protected-academic-info',
    url: joinURL(webBaseURL, '/user/academic-info'),
    expectedTexts: ['登录', 'Login'],
    expectedURLIncludes: ['/login', 'redirect=/user/academic-info'],
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
    stubbedResources: [
      {
        url: 'https://fonts.googleapis.com/**',
        contentType: 'text/css',
        body: '/* prod-parity smoke uses system fonts for the Casdoor login page. */\n',
      },
    ],
    allowedAPIResponses: [
      {
        urlIncludes: '/api/v1/auth/me',
        statuses: [401],
      },
    ],
  },
];

const results = [];
let passed = false;
let browser;

try {
  await mkdir(screenshotDir, { recursive: true });
  browser = await chromium.launch({ headless: process.env.PLAYWRIGHT_HEADLESS !== '0' });

  for (const viewportVariant of viewportVariants) {
    for (const check of checks) {
      results.push(await runCheck(browser, check, viewportVariant));
    }
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
    viewportVariants,
    checks: results,
  };
  await writeFile(evidenceFile, `${JSON.stringify(evidence, null, 2)}\n`);
}

if (!passed) {
  console.error(`[prod-parity-browser-smoke] failed; evidence: ${evidenceFile}`);
  process.exit(1);
}

console.log(`[prod-parity-browser-smoke] passed; evidence: ${evidenceFile}`);

async function runCheck(browser, check, viewportVariant) {
  const context = await browser.newContext({
    viewport: viewportVariant.viewport,
    deviceScaleFactor: viewportVariant.deviceScaleFactor,
    isMobile: viewportVariant.isMobile,
    hasTouch: viewportVariant.hasTouch,
  });
  const page = await context.newPage();
  const assetFailures = [];
  const apiFailures = [];
  const consoleErrors = [];
  const ignoredConsoleErrors = [];
  const ignoredAPIResponses = [];
  const pageErrors = [];
  const suppressedTelemetryRequests = [];
  const stubbedExternalResources = [];
  const checkName = `${check.name}-${viewportVariant.name}`;
  const screenshotFile = resolve(screenshotDir, `${checkName}.png`);

  for (const resourceStub of resourceStubsForCheck(check)) {
    await page.route(resourceStub.url, async (route) => {
      const request = route.request();
      const status = resourceStub.status || 200;
      const contentType = resourceStub.contentType || 'text/plain';
      stubbedExternalResources.push({
        url: request.url(),
        resourceType: request.resourceType(),
        status,
        contentType,
      });
      await route.fulfill({
        status,
        contentType,
        body: resourceStub.body || '',
      });
    });
  }

  await page.route(telemetryRoutePattern, async (route) => {
    suppressedTelemetryRequests.push({
      method: route.request().method(),
      url: route.request().url(),
    });
    await route.fulfill({ status: 204, body: '' });
  });

  page.on('console', (message) => {
    if (message.type() === 'error') {
      const described = describeConsoleMessage(message);
      if (isBrowserNetworkStatusConsoleError(message.text())) {
        ignoredConsoleErrors.push(described);
      } else {
        consoleErrors.push(described);
      }
    }
  });
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
      response.status() >= 400
    ) {
      const described = {
        url: response.url(),
        resourceType,
        status: response.status(),
        statusText: response.statusText(),
      };
      if (isAllowedAPIResponse(check, response)) {
        ignoredAPIResponses.push(described);
      } else {
        apiFailures.push(described);
      }
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
    const requiredTexts = toArray(check.requiredTexts);
    const missingRequiredTexts = requiredTexts.filter((text) => !bodyText.includes(text));

    if (!matchedText) {
      throw new Error(
        `missing expected text; expected one of: ${check.expectedTexts.join(', ')}`,
      );
    }
    if (missingRequiredTexts.length > 0) {
      throw new Error(
        `missing required text: ${missingRequiredTexts.join(', ')}`,
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
    if (consoleErrors.length > 0) {
      throw new Error(`console errors: ${consoleErrors.join(' | ')}`);
    }

    await page.screenshot({ path: screenshotFile, fullPage: true });

    return {
      name: check.name,
      checkName,
      viewport: viewportVariant.name,
      viewportSize: viewportVariant.viewport,
      passed: true,
      url: check.url,
      finalURL: page.url(),
      status: response.status(),
      title,
      matchedText,
      requiredTexts,
      ignoredConsoleErrors,
      ignoredAPIResponses,
      suppressedTelemetryRequests,
      stubbedExternalResources,
      screenshot: relative(repoRoot, screenshotFile),
    };
  } catch (error) {
    await page.screenshot({ path: screenshotFile, fullPage: true }).catch(() => undefined);
    return {
      name: check.name,
      checkName,
      viewport: viewportVariant.name,
      viewportSize: viewportVariant.viewport,
      passed: false,
      url: check.url,
      finalURL: page.url(),
      error: error instanceof Error ? error.message : String(error),
      pageErrors,
      consoleErrors,
      ignoredConsoleErrors,
      ignoredAPIResponses,
      suppressedTelemetryRequests,
      stubbedExternalResources,
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

function resourceStubsForCheck(check) {
  if (!Array.isArray(check.stubbedResources)) return [];
  return check.stubbedResources.filter((resourceStub) => (
    resourceStub &&
    typeof resourceStub.url === 'string' &&
    resourceStub.url.length > 0
  ));
}

function isAllowedAPIResponse(check, response) {
  const rules = Array.isArray(check.allowedAPIResponses) ? check.allowedAPIResponses : [];
  return rules.some((rule) => {
    const statuses = Array.isArray(rule.statuses) ? rule.statuses : [];
    return (
      statuses.includes(response.status()) &&
      typeof rule.urlIncludes === 'string' &&
      response.url().includes(rule.urlIncludes)
    );
  });
}

function describeConsoleMessage(message) {
  const location = message.location();
  const locationText =
    location.url && location.lineNumber > 0
      ? ` (${location.url}:${location.lineNumber}:${location.columnNumber})`
      : '';
  return `${message.text()}${locationText}`;
}

function isBrowserNetworkStatusConsoleError(text) {
  return /^Failed to load resource: the server responded with a status of [45]\d\d \([^)]+\)$/.test(
    text,
  );
}
