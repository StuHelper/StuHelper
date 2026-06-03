#!/usr/bin/env node
import { createRequire } from 'node:module';
import { lookup } from 'node:dns/promises';
import { mkdir, writeFile } from 'node:fs/promises';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const scriptDir = dirname(fileURLToPath(import.meta.url));
const repoRoot = resolve(scriptDir, '../..');
const requireFromWeb = createRequire(resolve(repoRoot, 'clients/web/package.json'));

let chromium;
try {
  ({ chromium } = requireFromWeb('@playwright/test'));
} catch (error) {
  console.error('[public-web-auth-browser-smoke] failed to load @playwright/test.');
  console.error('Install clients dependencies first, for example: pnpm --dir clients install.');
  console.error(error instanceof Error ? error.message : String(error));
  process.exit(1);
}

const timeoutMs = Number(process.env.PUBLIC_WEB_AUTH_BROWSER_SMOKE_TIMEOUT_MS || 30000);
const webBaseURL = normalizeBaseURL(process.env.WEB_PUBLIC_URL || 'https://stuhelper.com');
const joinBaseURL = normalizeBaseURL(
  process.env.ADMISSION_PUBLIC_BASE_URL || 'https://join.stuhelper.com',
);
const ssoBaseURL = normalizeBaseURL(
  process.env.SSO_PUBLIC_BASE_URL ||
    process.env.CASDOOR_PUBLIC_AUTH_BASE_URL ||
    'https://sso.stuhelper.com',
);
const probeToken = process.env.PUBLIC_WEB_AUTH_BROWSER_SMOKE_PROBE_TOKEN || '__stuhelper_browser_smoke__';
const probeQQ = process.env.PUBLIC_WEB_AUTH_BROWSER_SMOKE_PROBE_QQ || '10000';
const evidenceFile =
  process.env.PUBLIC_WEB_AUTH_BROWSER_SMOKE_EVIDENCE_FILE ||
  resolve(repoRoot, 'infra/generated/public-web-auth-browser-smoke-evidence.json');
const headless = normalizeBool(process.env.PUBLIC_WEB_AUTH_BROWSER_SMOKE_HEADLESS ?? 'true');
const browserExecutablePath = process.env.PUBLIC_WEB_AUTH_BROWSER_EXECUTABLE_PATH || undefined;
const allowLocalTargets = normalizeBool(
  process.env.PUBLIC_WEB_AUTH_BROWSER_SMOKE_ALLOW_LOCAL_TARGETS || 'false',
);
const resolvedTargets = {};

if (!allowLocalTargets) {
  requireExactURL('WEB_PUBLIC_URL', webBaseURL, 'https://stuhelper.com');
  requireExactURL('ADMISSION_PUBLIC_BASE_URL', joinBaseURL, 'https://join.stuhelper.com');
  requireExactURL('SSO_PUBLIC_BASE_URL', ssoBaseURL, 'https://sso.stuhelper.com');
  for (const [name, value] of [
    ['WEB_PUBLIC_URL', webBaseURL],
    ['ADMISSION_PUBLIC_BASE_URL', joinBaseURL],
    ['SSO_PUBLIC_BASE_URL', ssoBaseURL],
  ]) {
    rejectLocalTarget(name, value);
    resolvedTargets[name] = await rejectLoopbackResolvedTarget(name, value);
  }
}

const browser = await chromium.launch({
  headless,
  executablePath: browserExecutablePath,
  args: ['--use-fake-ui-for-media-stream', '--use-fake-device-for-media-stream'],
});
const checks = [];

try {
  await checkLoginPageDirect(browser);
  await checkProtectedDeveloperRoute(browser);
  await checkIdentityRoute(browser);
  await checkHeaderLoginEntry(browser);
  await checkLoginSignupEntry(browser);
  await checkJoinVerifyRoute(browser);
  await checkJoinLoginEntry(browser);
  await checkJoinSignupEntry(browser);
  await checkJoinMobileCameraRoute(browser);
} finally {
  await browser.close();
}

const passed = checks.every((check) => check.passed);
const evidence = {
  generatedAt: new Date().toISOString(),
  passed,
  targets: {
    webBaseURL,
    joinBaseURL,
    ssoBaseURL,
    probeToken,
    probeQQ,
    resolvedTargets,
  },
  summary: {
    passed: checks.filter((check) => check.passed).length,
    failed: checks.filter((check) => !check.passed).length,
  },
  checks,
};

if (evidenceFile !== '-') {
  await mkdir(dirname(evidenceFile), { recursive: true });
  await writeFile(evidenceFile, `${JSON.stringify(evidence, null, 2)}\n`, { mode: 0o600 });
  console.error(`[public-web-auth-browser-smoke] wrote evidence to ${evidenceFile}`);
} else {
  console.log(JSON.stringify(evidence, null, 2));
}

for (const check of checks) {
  const status = check.passed ? 'PASS' : 'FAIL';
  console.log(`${status} ${check.name}`);
  if (!check.passed) {
    for (const failure of check.failures) {
      console.error(`  - ${failure}`);
    }
  }
}

if (!passed) {
  console.error('[public-web-auth-browser-smoke] public web auth browser smoke failed');
  process.exit(1);
}

console.log(`[public-web-auth-browser-smoke] ${checks.length} browser checks passed`);

async function checkLoginPageDirect(browserInstance) {
  await runCheck(browserInstance, {
    name: 'web-login-page-renders',
    url: joinURL(webBaseURL, '/login?redirect=/developers/apps'),
    expectedURL: (url) =>
      isURLAtBasePath(url, webBaseURL, '/login') &&
      url.searchParams.get('redirect') === '/developers/apps',
    expectedText: /StuHelper|统一登录|Sign-in|统一身份认证/,
    action: async (page) => {
      await expectVisibleText(page, /使用统一身份认证登录|Continue with unified sign-in/);
    },
  });
}

async function checkProtectedDeveloperRoute(browserInstance) {
  await runCheck(browserInstance, {
    name: 'developer-apps-route-redirects-to-login',
    url: joinURL(webBaseURL, '/developers/apps'),
    expectedURL: (url) =>
      isURLAtBasePath(url, webBaseURL, '/login') &&
      url.searchParams.get('redirect') === '/developers/apps',
    expectedText: /StuHelper|统一登录|Sign-in|统一身份认证/,
    action: async (page) => {
      await expectVisibleText(page, /使用统一身份认证登录|Continue with unified sign-in/);
    },
  });
}

async function checkIdentityRoute(browserInstance) {
  await runCheck(browserInstance, {
    name: 'identity-route-redirects-to-login',
    url: joinURL(webBaseURL, '/identity'),
    expectedURL: (url) =>
      isURLAtBasePath(url, webBaseURL, '/login') &&
      url.searchParams.get('redirect') === '/identity',
    expectedText: /StuHelper|统一登录|Sign-in|统一身份认证/,
    action: async (page) => {
      await expectVisibleText(page, /使用统一身份认证登录|Continue with unified sign-in/);
    },
  });
}

async function checkHeaderLoginEntry(browserInstance) {
  await runCheck(browserInstance, {
    name: 'header-login-click-starts-sso',
    url: joinURL(webBaseURL, '/'),
    expectedURL: (url) => isURLAtBasePath(url, webBaseURL, '/'),
    expectedText: /StuHelper/,
    action: async (page) => {
      const loginLink = page.getByRole('link', { name: /登录|Login/ }).first();
      await loginLink.waitFor({ state: 'visible', timeout: timeoutMs });
      await loginLink.click();
      await page.waitForURL((url) => isURLAtBasePath(url, webBaseURL, '/login'), {
        timeout: timeoutMs,
      });
      await expectVisibleText(page, /使用统一身份认证登录|Continue with unified sign-in/);
      const loginButton = page
        .getByRole('button', { name: /使用统一身份认证登录|Continue with unified sign-in/ })
        .first();
      await Promise.all([
        page.waitForURL(
          (url) =>
            isURLAtBasePath(url, ssoBaseURL, '/login/oauth/authorize') &&
            url.searchParams.get('client_id') === 'stuhelper-web',
          { timeout: timeoutMs },
        ),
        loginButton.click(),
      ]);
      await expectSSOLoginAuthorizePageReady(page);
    },
  });
}

async function checkLoginSignupEntry(browserInstance) {
  await runCheck(browserInstance, {
    name: 'login-signup-click-starts-sso-signup',
    url: joinURL(webBaseURL, '/login?redirect=/developers/apps'),
    expectedURL: (url) =>
      isURLAtBasePath(url, webBaseURL, '/login') &&
      url.searchParams.get('redirect') === '/developers/apps',
    expectedText: /StuHelper|统一登录|Sign-in|统一身份认证/,
    action: async (page) => {
      await expectVisibleText(page, /注册账号|Create account|Sign up/i);
      const signupButton = page
        .getByRole('button', { name: /注册账号|Create account|Sign up/i })
        .first();
      await Promise.all([
        page.waitForURL(
          (url) =>
            isURLAtBasePath(url, ssoBaseURL, '/signup/oauth/authorize') &&
            url.searchParams.get('client_id') === 'stuhelper-web',
          { timeout: timeoutMs },
        ),
        signupButton.click(),
      ]);
      await expectSSOSignupAuthorizePageReady(page);
    },
  });
}

async function checkJoinVerifyRoute(browserInstance) {
  await runCheck(browserInstance, {
    name: 'join-verify-route-renders-spa',
    url: joinURL(joinBaseURL, `/verify/${encodeURIComponent(probeToken)}?qq=${encodeURIComponent(probeQQ)}`),
    expectedURL: (url) =>
      isURLAtBasePath(url, joinBaseURL, `/verify/${probeToken}`) &&
      url.searchParams.get('qq') === probeQQ,
    expectedText: /StuHelper|加群|认证|验证|统一身份认证|登录|admission|verify/i,
    expectedResponseHeaders: [
      {
        name: 'permissions-policy',
        includes: 'camera=(self)',
      },
    ],
  });
}

async function checkJoinLoginEntry(browserInstance) {
  const admissionRedirect = `/verify/${encodeURIComponent(probeToken)}?qq=${encodeURIComponent(probeQQ)}`;
  await runCheck(browserInstance, {
    name: 'join-login-click-starts-sso',
    url: joinURL(joinBaseURL, `/login?redirect=${encodeURIComponent(admissionRedirect)}`),
    expectedURL: (url) =>
      isURLAtBasePath(url, joinBaseURL, '/login') &&
      url.searchParams.get('redirect') === admissionRedirect,
    expectedText: /StuHelper|统一登录|Sign-in|统一身份认证/,
    action: async (page) => {
      await expectVisibleText(page, /使用统一身份认证登录|Continue with unified sign-in/);
      const loginButton = page
        .getByRole('button', { name: /使用统一身份认证登录|Continue with unified sign-in/ })
        .first();
      await Promise.all([
        page.waitForURL(
          (url) =>
            isURLAtBasePath(url, ssoBaseURL, '/login/oauth/authorize') &&
            url.searchParams.get('client_id') === 'stuhelper-web' &&
            url.searchParams.get('redirect_uri') === `${webBaseURL}/api/v1/auth/callback`,
          { timeout: timeoutMs },
        ),
        loginButton.click(),
      ]);
      await expectSSOLoginAuthorizePageReady(page);
    },
  });
}

async function checkJoinSignupEntry(browserInstance) {
  const admissionRedirect = `/verify/${encodeURIComponent(probeToken)}?qq=${encodeURIComponent(probeQQ)}`;
  await runCheck(browserInstance, {
    name: 'join-signup-click-starts-sso-signup',
    url: joinURL(joinBaseURL, `/login?redirect=${encodeURIComponent(admissionRedirect)}`),
    expectedURL: (url) =>
      isURLAtBasePath(url, joinBaseURL, '/login') &&
      url.searchParams.get('redirect') === admissionRedirect,
    expectedText: /StuHelper|统一登录|Sign-in|统一身份认证/,
    action: async (page) => {
      await expectVisibleText(page, /注册账号|Create account|Sign up/i);
      const signupButton = page
        .getByRole('button', { name: /注册账号|Create account|Sign up/i })
        .first();
      await Promise.all([
        page.waitForURL(
          (url) =>
            isURLAtBasePath(url, ssoBaseURL, '/signup/oauth/authorize') &&
            url.searchParams.get('client_id') === 'stuhelper-web' &&
            url.searchParams.get('redirect_uri') === `${webBaseURL}/api/v1/auth/callback`,
          { timeout: timeoutMs },
        ),
        signupButton.click(),
      ]);
      await expectSSOSignupAuthorizePageReady(page);
    },
  });
}

async function checkJoinMobileCameraRoute(browserInstance) {
  await runCheck(browserInstance, {
    name: 'join-mobile-camera-route-allows-camera',
    url: joinURL(joinBaseURL, '/admission/freshman/camera/__stuhelper_browser_smoke__'),
    expectedURL: (url) => isURLAtBasePath(url, joinBaseURL, '/admission/freshman/camera/__stuhelper_browser_smoke__'),
    expectedText: /StuHelper|新生材料拍照|无法打开拍照链接|camera/i,
    expectedResponseHeaders: [
      {
        name: 'permissions-policy',
        includes: 'camera=(self)',
      },
    ],
    expectedBrowserPermissions: [
      {
        feature: 'camera',
        allowed: true,
      },
    ],
    expectedMediaCaptures: [
      {
        name: 'camera',
        constraints: { video: true, audio: false },
        minVideoTracks: 1,
      },
    ],
  });
}

async function runCheck(browserInstance, check) {
  const context = await browserInstance.newContext({
    viewport: { width: 1366, height: 900 },
    ignoreHTTPSErrors: normalizeBool(process.env.PUBLIC_WEB_AUTH_BROWSER_SMOKE_TLS_INSECURE || 'false'),
  });
  const capturePermissions = expectedMediaCapturePermissions(check);
  if (capturePermissions.length > 0) {
    await context.grantPermissions(capturePermissions, { origin: new URL(check.url).origin });
  }
  const page = await context.newPage();
  page.setDefaultTimeout(timeoutMs);

  const failures = [];
  const responses = [];

  page.on('pageerror', (error) => {
    failures.push(`pageerror: ${error.stack || error.message}`);
  });
  page.on('console', (message) => {
    if (/Permissions policy violation/i.test(message.text())) {
      failures.push(`console.${message.type()}: ${message.text()}`);
      return;
    }
    if (message.type() === 'error' && !isAllowedConsoleError(message)) {
      failures.push(`console.error: ${message.text()}`);
    }
  });
  page.on('requestfailed', (request) => {
    if (isUnexpectedFailedRequest(request)) {
      const type = request.resourceType();
      failures.push(
        `${type} ${request.method()} ${request.url()} ${request.failure()?.errorText || 'failed'}`,
      );
    }
  });
  page.on('response', (response) => {
    responses.push({
      url: response.url(),
      status: response.status(),
      requestType: response.request().resourceType(),
    });
    const request = response.request();
    const type = request.resourceType();
    if (type === 'document') {
      return;
    }
    if ((type === 'script' || type === 'stylesheet' || type === 'font') && response.status() >= 400) {
      failures.push(`${type} ${request.method()} ${response.url()} HTTP ${response.status()}`);
    }
    if (isUnexpectedAPIResponse(response)) {
      failures.push(`api ${request.method()} ${response.url()} HTTP ${response.status()}`);
    }
  });

  let mainResponse = null;
  let browserPermissions = [];
  let mediaCaptures = [];
  try {
    mainResponse = await page.goto(check.url, {
      waitUntil: 'domcontentloaded',
      timeout: timeoutMs,
    });
    await page.waitForLoadState('networkidle', { timeout: timeoutMs }).catch(() => undefined);
    await page.waitForTimeout(300);

    if (check.expectedStatus !== undefined) {
      const status = mainResponse?.status();
      if (status !== check.expectedStatus) {
        failures.push(`expected document HTTP ${check.expectedStatus}, got ${status ?? 'none'}`);
      }
    } else if (mainResponse && mainResponse.status() >= 400) {
      failures.push(`document HTTP ${mainResponse.status()}`);
    }

    const currentURL = new URL(page.url());
    if (check.expectedURL && !check.expectedURL(currentURL)) {
      failures.push(`unexpected URL after load: ${currentURL.toString()}`);
    }
    if (mainResponse) {
      failures.push(...expectedResponseHeaderFailures(mainResponse, check));
    }
    browserPermissions = await readExpectedBrowserPermissions(page, check);
    failures.push(...browserPermissions.flatMap((item) => item.failures));
    mediaCaptures = await readExpectedMediaCaptures(page, check);
    failures.push(...mediaCaptures.flatMap((item) => item.failures));

    if (check.expectedText) {
      const bodyText = await page.locator('body').innerText({ timeout: timeoutMs }).catch(() => '');
      if (!bodyText.trim()) {
        failures.push('body text is empty');
      } else if (!check.expectedText.test(bodyText)) {
        failures.push(`body text did not match ${check.expectedText}`);
      }
    }

    if (check.action) {
      await check.action(page);
    }
  } catch (error) {
    failures.push(error instanceof Error ? error.message : String(error));
  } finally {
    await context.close();
  }

  checks.push({
    name: check.name,
    passed: failures.length === 0,
    failures,
    url: check.url,
    finalURL: page.url(),
    documentStatus: mainResponse?.status() ?? null,
    responseHeaders: mainResponse ? readExpectedResponseHeaders(mainResponse, check) : {},
    browserPermissions,
    mediaCaptures,
    responses: responses
      .filter((response) => response.requestType === 'document')
      .slice(0, 5)
      .map(({ requestType: _requestType, ...response }) => response),
  });
}

function expectedResponseHeaderFailures(response, check) {
  const failures = [];
  for (const expected of toArray(check.expectedResponseHeaders)) {
    const actual = response.headers()[expected.name.toLowerCase()] || '';
    if (expected.includes && !actual.includes(expected.includes)) {
      failures.push(
        `header ${expected.name}=${JSON.stringify(actual)} does not include ${JSON.stringify(expected.includes)}`,
      );
    }
  }
  return failures;
}

function readExpectedResponseHeaders(response, check) {
  const headers = {};
  for (const expected of toArray(check.expectedResponseHeaders)) {
    headers[expected.name.toLowerCase()] = response.headers()[expected.name.toLowerCase()] || '';
  }
  return headers;
}

async function readExpectedBrowserPermissions(page, check) {
  const expectedPermissions = toArray(check.expectedBrowserPermissions);
  if (expectedPermissions.length === 0) {
    return [];
  }

  return Promise.all(
    expectedPermissions.map(async (expected) => {
      const result = await page.evaluate((feature) => {
        const policy = document.permissionsPolicy || document.featurePolicy;
        if (!policy || typeof policy.allowsFeature !== 'function') {
          return { supported: false, allowed: null, error: null };
        }
        try {
          return {
            supported: true,
            allowed: Boolean(policy.allowsFeature(feature)),
            error: null,
          };
        } catch (error) {
          return {
            supported: true,
            allowed: null,
            error: error instanceof Error ? error.message : String(error),
          };
        }
      }, expected.feature);
      const failures = [];
      if (!result.supported) {
        failures.push(`browser permission policy API is not available for ${expected.feature}`);
      } else if (result.error) {
        failures.push(`browser permission policy check for ${expected.feature} failed: ${result.error}`);
      } else if (result.allowed !== expected.allowed) {
        failures.push(
          `browser permission policy ${expected.feature} allowed=${result.allowed}, expected ${expected.allowed}`,
        );
      }
      return {
        feature: expected.feature,
        expectedAllowed: expected.allowed,
        supported: result.supported,
        allowed: result.allowed,
        failures,
      };
    }),
  );
}

function expectedMediaCapturePermissions(check) {
  const permissions = new Set();
  for (const expected of toArray(check.expectedMediaCaptures)) {
    const constraints = expected.constraints || {};
    if (constraints.video) {
      permissions.add('camera');
    }
    if (constraints.audio) {
      permissions.add('microphone');
    }
  }
  return Array.from(permissions);
}

async function readExpectedMediaCaptures(page, check) {
  const expectedCaptures = toArray(check.expectedMediaCaptures);
  if (expectedCaptures.length === 0) {
    return [];
  }

  return Promise.all(
    expectedCaptures.map(async (expected) => {
      const constraints = expected.constraints || { video: true, audio: false };
      const result = await page.evaluate(async (mediaConstraints) => {
        const mediaDevices = navigator.mediaDevices;
        if (!mediaDevices || typeof mediaDevices.getUserMedia !== 'function') {
          return {
            supported: false,
            success: false,
            error: null,
            trackCount: 0,
            videoTrackCount: 0,
            audioTrackCount: 0,
          };
        }

        let stream;
        try {
          stream = await mediaDevices.getUserMedia(mediaConstraints);
          const tracks = stream.getTracks();
          const videoTrackCount = stream.getVideoTracks().length;
          const audioTrackCount = stream.getAudioTracks().length;
          for (const track of tracks) {
            track.stop();
          }
          return {
            supported: true,
            success: true,
            error: null,
            trackCount: tracks.length,
            videoTrackCount,
            audioTrackCount,
          };
        } catch (error) {
          if (stream) {
            for (const track of stream.getTracks()) {
              track.stop();
            }
          }
          const name = error instanceof DOMException && error.name ? error.name : 'Error';
          const message = error instanceof Error ? error.message : String(error);
          return {
            supported: true,
            success: false,
            error: `${name}: ${message}`,
            trackCount: 0,
            videoTrackCount: 0,
            audioTrackCount: 0,
          };
        }
      }, constraints);
      const failures = [];
      const name = expected.name || 'media';
      if (!result.supported) {
        failures.push(`media capture ${name} API is not available`);
      } else if (!result.success) {
        failures.push(`media capture ${name} failed: ${result.error || 'unknown error'}`);
      } else if (expected.minVideoTracks !== undefined && result.videoTrackCount < expected.minVideoTracks) {
        failures.push(
          `media capture ${name} videoTrackCount=${result.videoTrackCount}, expected at least ${expected.minVideoTracks}`,
        );
      } else if (expected.minAudioTracks !== undefined && result.audioTrackCount < expected.minAudioTracks) {
        failures.push(
          `media capture ${name} audioTrackCount=${result.audioTrackCount}, expected at least ${expected.minAudioTracks}`,
        );
      } else if (result.trackCount < 1) {
        failures.push(`media capture ${name} returned no tracks`);
      }
      return {
        name,
        constraints,
        supported: result.supported,
        success: result.success,
        trackCount: result.trackCount,
        videoTrackCount: result.videoTrackCount,
        audioTrackCount: result.audioTrackCount,
        error: result.error,
        failures,
      };
    }),
  );
}

function toArray(value) {
  if (Array.isArray(value)) return value;
  if (value === undefined || value === null) return [];
  return [value];
}

async function expectVisibleText(page, pattern) {
  await page.getByText(pattern).first().waitFor({ state: 'visible', timeout: timeoutMs });
}

async function expectSSOAuthorizePageReady(page) {
  await page.waitForLoadState('networkidle', { timeout: timeoutMs }).catch(() => undefined);
  await page.waitForTimeout(1000);
  const bodyText = await page.locator('body').innerText({ timeout: timeoutMs }).catch(() => '');
  const normalized = bodyText.replace(/\s+/g, ' ').trim();
  if (!normalized) {
    throw new Error('SSO authorize page body text is empty');
  }
  if (/invalid redirect|redirect_uri mismatch|redirect uri mismatch|回调地址错误/i.test(normalized)) {
    throw new Error('SSO authorize page shows a redirect URI error');
  }
  return normalized;
}

async function expectSSOLoginAuthorizePageReady(page) {
  const normalized = await expectSSOAuthorizePageReady(page);
  if (!/password|密码/i.test(normalized)) {
    throw new Error('SSO login page does not expose password login');
  }
  await page
    .getByRole('textbox', { name: /username|email|phone|用户名|邮箱|手机/i })
    .first()
    .waitFor({ state: 'visible', timeout: timeoutMs });
  await page
    .getByRole('textbox', { name: /password|密码/i })
    .first()
    .waitFor({ state: 'visible', timeout: timeoutMs });
  await page
    .getByRole('button', { name: /sign in|登录/i })
    .first()
    .waitFor({ state: 'visible', timeout: timeoutMs });
  const signupLink = page.getByRole('link', { name: /sign up|注册/i }).first();
  await signupLink.waitFor({ state: 'visible', timeout: timeoutMs });
  const href = await signupLink.getAttribute('href');
  if (!href || !href.includes('/signup/oauth/authorize')) {
    throw new Error(`SSO login page signup link is not /signup/oauth/authorize: ${href || '<missing>'}`);
  }
}

async function expectSSOSignupAuthorizePageReady(page) {
  const normalized = await expectSSOAuthorizePageReady(page);
  if (!/username|用户名/i.test(normalized) || !/password|密码/i.test(normalized)) {
    throw new Error('SSO signup page does not expose username/password signup fields');
  }
  await page
    .getByRole('textbox', { name: /username|用户名/i })
    .first()
    .waitFor({ state: 'visible', timeout: timeoutMs });
  await page
    .getByRole('textbox', { name: /display name|显示名称|昵称/i })
    .first()
    .waitFor({ state: 'visible', timeout: timeoutMs });
  await page
    .getByRole('textbox', { name: /password|密码/i })
    .first()
    .waitFor({ state: 'visible', timeout: timeoutMs });
  await page
    .getByRole('textbox', { name: /confirm|确认/i })
    .first()
    .waitFor({ state: 'visible', timeout: timeoutMs });
  await page
    .getByRole('button', { name: /sign up|注册/i })
    .first()
    .waitFor({ state: 'visible', timeout: timeoutMs });
}

function isUnexpectedAPIResponse(response) {
  const request = response.request();
  const type = request.resourceType();
  if (type !== 'fetch' && type !== 'xhr' && type !== 'eventsource') {
    return false;
  }
  if (response.status() < 400) {
    return false;
  }
  const url = new URL(response.url());
  if (url.pathname === '/api/v1/auth/me' && response.status() === 401) {
    return false;
  }
  if (url.pathname === '/api/v1/auth/refresh' && response.status() === 401) {
    return false;
  }
  if (/^\/api\/v1\/admission\/sessions\/[^/]+$/.test(url.pathname) && response.status() === 404) {
    return false;
  }
  if (
    /^\/api\/v1\/admission\/freshman\/mobile-camera-handoffs\/[^/]+$/.test(url.pathname) &&
    response.status() === 404
  ) {
    return false;
  }
  return true;
}

function isUnexpectedFailedRequest(request) {
  if (isAllowedFailedRequest(request)) {
    return false;
  }
  const type = request.resourceType();
  if (type === 'document' || type === 'script' || type === 'stylesheet' || type === 'font') {
    return true;
  }
  if (type === 'fetch' || type === 'xhr' || type === 'eventsource') {
    return true;
  }
  if (type === 'ping') {
    return false;
  }
  return isAPIURL(request.url());
}

function isAllowedFailedRequest(request) {
  return isCasdoorNativeSSOProbe(request.url());
}

function isAPIURL(rawURL) {
  try {
    const url = new URL(rawURL);
    return url.pathname.startsWith('/api/');
  } catch {
    return false;
  }
}

function isAllowedConsoleError(message) {
  const text = message.text();
  if (/^Failed to load resource: the server responded with a status of (401|404) /.test(text)) {
    return true;
  }
  if (/^Failed to load resource: net::ERR_CONNECTION_REFUSED$/.test(text)) {
    const location = message.location?.();
    if (location?.url && isCasdoorNativeSSOProbe(location.url)) {
      return true;
    }
  }
  return false;
}

function isCasdoorNativeSSOProbe(rawURL) {
  try {
    const url = new URL(rawURL);
    return (
      url.protocol === 'http:' &&
      url.hostname === '127.0.0.1' &&
      /^473[0-9][0-9]$/.test(url.port) &&
      url.pathname === '/native-sso/status'
    );
  } catch {
    return false;
  }
}

function normalizeBaseURL(value) {
  const url = new URL(value);
  url.hash = '';
  url.search = '';
  return url.toString().replace(/\/$/, '');
}

function joinURL(baseURL, path) {
  const suffix = path.startsWith('/') ? path : `/${path}`;
  return `${baseURL}${suffix}`;
}

function isURLAtBasePath(url, baseURL, path) {
  const expected = new URL(joinURL(baseURL, path));
  return url.origin === expected.origin && url.pathname === expected.pathname;
}

function normalizeBool(value) {
  switch (String(value).trim().toLowerCase()) {
    case 'true':
    case '1':
    case 'yes':
      return true;
    case 'false':
    case '0':
    case 'no':
    case '':
      return false;
    default:
      throw new Error(`invalid boolean value: ${value}`);
  }
}

function requireExactURL(name, actual, expected) {
  if (actual !== expected) {
    throw new Error(`${name} must be exactly ${expected} for production browser smoke, got ${actual}`);
  }
}

function rejectLocalTarget(name, value) {
  if (/localhost|127\.0\.0\.1|\[?::1\]?|host\.docker\.internal/.test(value)) {
    throw new Error(
      `${name} points to a local target (${value}); set PUBLIC_WEB_AUTH_BROWSER_SMOKE_ALLOW_LOCAL_TARGETS=true only for local contract tests.`,
    );
  }
}

async function rejectLoopbackResolvedTarget(name, value) {
  const host = new URL(value).hostname;
  let addresses;
  try {
    addresses = await lookup(host, { all: true, verbatim: false });
  } catch (error) {
    throw new Error(
      `${name} host ${host} could not be resolved for production browser smoke: ${
        error instanceof Error ? error.message : String(error)
      }`,
    );
  }
  if (addresses.length === 0) {
    throw new Error(`${name} host ${host} resolved to no addresses for production browser smoke`);
  }
  for (const item of addresses) {
    if (isLoopbackAddress(item.address)) {
      throw new Error(
        `${name} host ${host} resolves to loopback (${item.address}); remove local hosts overrides before running production browser smoke, or set PUBLIC_WEB_AUTH_BROWSER_SMOKE_ALLOW_LOCAL_TARGETS=true only for local contract tests.`,
      );
    }
  }
  return addresses.map((item) => ({ address: item.address, family: item.family }));
}

function isLoopbackAddress(address) {
  const normalized = String(address).trim().toLowerCase();
  return (
    normalized === '::1' ||
    normalized === '0:0:0:0:0:0:0:1' ||
    normalized === '0.0.0.0' ||
    normalized.startsWith('127.')
  );
}
