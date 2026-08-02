#!/usr/bin/env node
import { mkdir, writeFile } from 'node:fs/promises';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { createRequire } from 'node:module';
import { spawnSync } from 'node:child_process';

const scriptDir = dirname(fileURLToPath(import.meta.url));
const repoRoot = resolve(scriptDir, '../..');
const requireFromWeb = createRequire(resolve(repoRoot, 'clients/web/package.json'));

let chromium;
try {
  ({ chromium } = requireFromWeb('@playwright/test'));
} catch (error) {
  console.error('[admission-prod-sim-e2e] failed to load @playwright/test.');
  console.error('Run infra/ops/bootstrap-dev-ubuntu2404.sh or install clients dependencies first.');
  console.error(error instanceof Error ? error.message : String(error));
  process.exit(1);
}

const timeoutMs = Number(process.env.ADMISSION_PROD_SIM_E2E_TIMEOUT_MS || 45000);
const apiBaseURL = normalizeBaseURL(process.env.API_BASE_URL || 'http://127.0.0.1:28080');
const webBaseURL = normalizeBaseURL(process.env.WEB_BASE_URL || 'https://stuhelper.com');
const admissionBaseURL = normalizeBaseURL(
  process.env.ADMISSION_BASE_URL ||
  process.env.ADMISSION_PUBLIC_BASE_URL ||
  'https://join.stuhelper.com',
);
const ssoBaseURL = normalizeBaseURL(
  process.env.ADMISSION_PROD_SIM_SSO_BASE_URL ||
  process.env.SSO_BASE_URL ||
  process.env.SSO_PUBLIC_BASE_URL ||
  'https://sso.stuhelper.com',
);
const botServiceToken = process.env.BOT_SERVICE_TOKEN || '';
const casdoorLoginUsername = process.env.PROD_PARITY_CASDOOR_LOGIN_USERNAME || 'admission-e2e';
const casdoorLoginPassword = process.env.PROD_PARITY_CASDOOR_LOGIN_PASSWORD || 'ProdParityAdmission1!';
const qqID = process.env.ADMISSION_PROD_SIM_QQ_ID || '990002';
const botSelfID = process.env.ADMISSION_PROD_SIM_BOT_SELF_ID || 'prod-sim-bot';
const guildID = process.env.ADMISSION_PROD_SIM_GUILD_ID || 'prod-parity-guild';
const channelID = process.env.ADMISSION_PROD_SIM_CHANNEL_ID || 'prod-parity-channel';
const schoolCode = process.env.ADMISSION_PROD_SIM_SCHOOL_CODE || '4111010006';
const schoolID = Number(process.env.ADMISSION_PROD_SIM_SCHOOL_ID || '4111010006');
const studentID = process.env.ADMISSION_PROD_SIM_STUDENT_ID || '20259901';
const studentName = process.env.ADMISSION_PROD_SIM_STUDENT_NAME || '张三';
const studentEmail = process.env.ADMISSION_PROD_SIM_STUDENT_EMAIL || `${studentID}@buaa.edu.cn`;
const evidenceFile = process.env.ADMISSION_PROD_SIM_E2E_EVIDENCE_FILE ||
  resolve(repoRoot, '.run/prod-parity/admission-prod-sim-e2e-evidence.json');
const screenshotDir = process.env.ADMISSION_PROD_SIM_E2E_SCREENSHOT_DIR ||
  resolve(dirname(evidenceFile), 'admission-prod-sim-e2e-screenshots');

if (!botServiceToken) {
  fail('BOT_SERVICE_TOKEN is required for prod-parity admission E2E');
}
if (admissionBaseURL !== 'https://join.stuhelper.com') {
  fail('ADMISSION_BASE_URL must be https://join.stuhelper.com for production-parity admission E2E');
}
if (ssoBaseURL !== 'https://sso.stuhelper.com') {
  fail('SSO_BASE_URL must be https://sso.stuhelper.com for production-parity admission E2E');
}

const browserEvents = {
  consoleErrors: [],
  pageErrors: [],
  requestFailures: [],
  apiFailures: [],
};
const steps = [];
const startedAt = new Date();

let browser;
let context;
let page;

try {
  const created = await step('bot creates admission session', async () => {
    const payload = await botFetch('/api/v1/bot/admission/sessions', {
      method: 'POST',
      body: {
        platform: 'qq',
        botSelfID,
        guildID,
        channelID,
        qqID,
      },
    });
    const data = requireData(payload, 'bot session create response');
    if (!data.session?.id) throw new Error('created admission session id is missing');
    if (!data.authURL?.startsWith(`${admissionBaseURL}/verify/`)) {
      throw new Error(`created authURL is not canonical: ${redactAdmissionURL(data.authURL)}`);
    }
    const createdURL = new URL(data.authURL);
    if (createdURL.searchParams.has('qq')) {
      throw new Error('created authURL must not contain a qq query');
    }
	    return {
	      sessionID: data.session.id,
	      status: data.session.status,
	      authURL: data.authURL,
	      token: data.token,
	      tokenPresent: typeof data.token === 'string' && data.token.length > 0,
	    };
	  }, (result) => ({
	    sessionID: result.sessionID,
	    status: result.status,
	    authURLShape: redactAdmissionURL(result.authURL),
	    tokenPresent: result.tokenPresent,
	  }));
	  if (!created.tokenPresent) {
	    throw new Error('created admission token is missing');
	  }

	  await step('backend previews just-created admission token', async () => {
	    const payload = await apiFetch(
	      `/api/v1/admission/sessions/${encodeURIComponent(created.token)}`,
	    );
	    const data = requireData(payload, 'admission preview response');
	    if (data.id !== created.sessionID) {
	      throw new Error(`preview session id mismatch: ${data.id}`);
	    }
	    if (data.qqID !== qqID) {
	      throw new Error(`preview qqID mismatch: ${data.qqID}`);
	    }
	    if (data.status !== created.status) {
	      throw new Error(`preview status mismatch: ${data.status}`);
	    }
	    return {
	      sessionID: data.id,
	      status: data.status,
	      qqID: data.qqID,
	      authURLShape: redactAdmissionURL(data.authURL || created.authURL),
	    };
	  });

	  browser = await chromium.launch({
	    headless: true,
	    args: ['--no-proxy-server'],
	  });
  context = await browser.newContext({
    ignoreHTTPSErrors: true,
    viewport: { width: 1365, height: 900 },
  });
  context.setDefaultTimeout(timeoutMs);
  page = await context.newPage();
  attachBrowserDiagnostics(page);

  await step('browser logs in and links admission session', async () => {
    await page.goto(created.authURL, {
      waitUntil: 'domcontentloaded',
      timeout: timeoutMs,
    });
    await expectText(page, '入群身份认证');
    await expectText(page, `QQ：${qqID}`);
    await page.getByRole('button', { name: /^登录$/ }).click();
    const loginEntry = await waitForAnyLocationPrefix(page, [
      `${webBaseURL}/login`,
      `${ssoBaseURL}/login/oauth/authorize`,
    ]);
    if (loginEntry.startsWith(`${webBaseURL}/login`)) {
      await page.getByRole('button', { name: /SSO|统一身份/i }).click();
      await waitForLocationPrefix(page, `${ssoBaseURL}/login/oauth/authorize`);
    }
    await fillCasdoorPasswordLogin(page);
    await waitForLocationPrefix(page, `${admissionBaseURL}/verify/`);
    await expectText(page, '开始认证');
    await page.getByRole('button', { name: '开始认证' }).click();
    await page.locator('[data-admission-bind-confirmation-input]').fill(qqID);
    await page.locator('[data-admission-bind-confirmation-submit]').click();
    await expectText(page, '选择认证方式');
    const userID = await waitForSessionUserID(created.sessionID);
    return { linked: true, userIDPresent: userID > 0 };
  });

  const userID = await waitForSessionUserID(created.sessionID);

  await step('browser completes BUAA academic email OTP verification', async () => {
    await page.getByRole('tab', { name: '老生认证' }).click();
    await page.locator('[data-school-select]').selectOption(schoolCode);
    await page.locator('[data-academic-student-id-input]').fill(studentID);
    await page.locator('[data-academic-student-name-input]').fill(studentName);
    await page.locator('[data-school-email-otp-request]').click();
    await expectText(page, '学号和姓名已匹配，验证码已发送到学号邮箱。');
    const emailValue = await page.locator('[data-academic-email-input]').inputValue();
    if (emailValue !== studentEmail) {
      throw new Error(`derived school email mismatch: ${emailValue}`);
    }
    const otp = await readAdmissionOTP(userID, schoolID);
    await page.locator('input[inputmode="numeric"]').fill(otp);
    await page.getByRole('button', { name: '验证邮箱' }).click();
    await expectText(page, '认证已通过');
    return { email: studentEmail, credentialKind: 'school_email_otp' };
  });

  const releaseAction = await step('bot polls pending release action', async () => {
    const action = await waitForReleaseAction(created.sessionID);
    return {
      sessionID: action.sessionID,
      action: action.action,
      platform: action.platform,
      botSelfID: action.botSelfID,
      guildID: action.guildID,
      qqID: action.qqID,
      authURLShape: redactAdmissionURL(action.authURL),
    };
  });

  await step('bot records release event', async () => {
    await botFetch(`/api/v1/bot/admission/sessions/${encodeURIComponent(created.sessionID)}/events`, {
      method: 'POST',
      body: {
        action: releaseAction.action,
        success: true,
        messageID: 'prod-sim-release',
      },
    });
    const state = await querySessionState(created.sessionID);
    if (state.status !== 'verified' || !state.cancelledAtPresent) {
      throw new Error(`session was not released: ${JSON.stringify(state)}`);
    }
    return state;
  });

  assertNoBrowserDiagnostics();
  await writeEvidence(true, {
    input: { platform: 'qq', botSelfID, guildID, channelID, qqID, schoolCode, studentID },
    steps,
    browserEvents,
  });
  console.log(`[admission-prod-sim-e2e] passed; evidence: ${evidenceFile}`);
} catch (error) {
  if (page) {
    await mkdir(screenshotDir, { recursive: true }).catch(() => undefined);
    await page.screenshot({
      path: resolve(screenshotDir, `failure-${Date.now()}.png`),
      fullPage: true,
    }).catch(() => undefined);
  }
  await writeEvidence(false, {
    input: { platform: 'qq', botSelfID, guildID, channelID, qqID, schoolCode, studentID },
    steps,
    browserEvents,
    page: await describePage(page),
    error: sanitizeErrorMessage(error instanceof Error ? error.message : String(error)),
  }).catch(() => undefined);
  fail(sanitizeErrorMessage(error instanceof Error ? error.message : String(error)));
} finally {
  await context?.close().catch(() => undefined);
  await browser?.close().catch(() => undefined);
}

async function step(name, fn, detailFn = (value) => value) {
  const started = new Date();
  try {
    const detail = await fn();
    const finished = new Date();
    steps.push({
      name,
      status: 'passed',
      startedAt: started.toISOString(),
      finishedAt: finished.toISOString(),
      durationMs: finished.getTime() - started.getTime(),
      detail: detailFn(detail),
    });
    return detail;
  } catch (error) {
    const finished = new Date();
    steps.push({
      name,
      status: 'failed',
      startedAt: started.toISOString(),
      finishedAt: finished.toISOString(),
      durationMs: finished.getTime() - started.getTime(),
      error: sanitizeErrorMessage(error instanceof Error ? error.message : String(error)),
    });
    throw error;
  }
}

async function fillCasdoorPasswordLogin(targetPage) {
  await fillRoleTextboxOrVisibleInput(
    targetPage,
    /username|email|phone/i,
    [
      '.login-username-input input',
      'input[autocomplete="username"]',
      'input[name="username"]',
      'input[id*="username" i]',
      'input[placeholder*="username" i]',
      'input[placeholder*="email" i]',
      'input[type="text"]',
    ].join(', '),
    casdoorLoginUsername,
    'Casdoor username',
  );

  await fillRoleTextboxOrVisibleInput(
    targetPage,
    /^password$/i,
    [
      '.login-password-input input',
      'input[autocomplete="current-password"]',
      'input[name="password"]',
      'input[id*="password" i]',
      'input[type="password"]',
    ].join(', '),
    casdoorLoginPassword,
    'Casdoor password',
  );

  const filled = await describeCasdoorLoginForm(targetPage);
  if (filled.usernameValueLength !== casdoorLoginUsername.length || filled.passwordValueLength !== casdoorLoginPassword.length) {
    throw new Error(`Casdoor login form did not retain filled values: ${JSON.stringify(filled)}`);
  }

  const loginResponse = targetPage.waitForResponse(
    (response) => new URL(response.url()).origin === ssoBaseURL && new URL(response.url()).pathname === '/api/login',
    { timeout: 10000 },
  ).catch(() => null);
  await clickRoleButtonOrVisible(targetPage, /^sign in$|^登录$/i, [
    'button:has-text("Sign In")',
    'button:has-text("登录")',
    '.login-button',
    'button[type="submit"]',
  ].join(', '), 'Casdoor login button');
  const response = await loginResponse;
  if (!response) {
    throw new Error(`Casdoor login button did not submit /api/login: ${JSON.stringify(await describeCasdoorLoginForm(targetPage))}`);
  }
  if (response.status() >= 400) {
    throw new Error(`Casdoor /api/login returned ${response.status()}`);
  }
}

async function fillRoleTextboxOrVisibleInput(targetPage, accessibleName, selector, value, label) {
  const roleLocator = targetPage.getByRole('textbox', { name: accessibleName }).first();
  if (await isActionable(roleLocator)) {
    await roleLocator.click({ timeout: timeoutMs });
    await roleLocator.fill(value, { timeout: timeoutMs });
    const actual = await roleLocator.inputValue({ timeout: timeoutMs });
    if (actual === value) return { role: true };
  }
  return fillFirstVisibleInput(targetPage, selector, value, label);
}

async function fillFirstVisibleInput(targetPage, selector, value, label) {
  const locator = targetPage.locator(selector);
  await locator.first().waitFor({ state: 'attached', timeout: timeoutMs });
  const count = await locator.count();
  for (let index = 0; index < count; index += 1) {
    const candidate = locator.nth(index);
    if (!(await isActionable(candidate))) continue;
    await candidate.click({ timeout: timeoutMs });
    await candidate.fill(value, { timeout: timeoutMs });
    const actual = await candidate.inputValue({ timeout: timeoutMs });
    if (actual === value) {
      return { index };
    }
  }
  throw new Error(`${label} visible input was not filled; candidates=${count}`);
}

async function clickRoleButtonOrVisible(targetPage, accessibleName, selector, label) {
  const roleButton = targetPage.getByRole('button', { name: accessibleName });
  const roleButtonCount = await roleButton.count().catch(() => 0);
  for (let index = 0; index < roleButtonCount; index += 1) {
    const candidate = roleButton.nth(index);
    if (!(await isActionable(candidate))) continue;
    await candidate.click({ timeout: timeoutMs });
    return { role: true, index };
  }

  const locator = targetPage.locator(selector);
  const count = await locator.count();
  for (let index = 0; index < count; index += 1) {
    const candidate = locator.nth(index);
    if (!(await isActionable(candidate))) continue;
    await candidate.click({ timeout: timeoutMs });
    return { role: false, index };
  }
  throw new Error(`${label} was not visible; candidates=${count}`);
}

async function isActionable(locator) {
  if (!(await locator.isVisible().catch(() => false))) return false;
  if (!(await locator.isEnabled().catch(() => true))) return false;
  const box = await locator.boundingBox().catch(() => null);
  return Boolean(box && box.width > 0 && box.height > 0);
}

async function describeCasdoorLoginForm(targetPage) {
  return targetPage.evaluate(() => {
    const visible = (element) => {
      const rect = element.getBoundingClientRect();
      const style = window.getComputedStyle(element);
      return rect.width > 0 && rect.height > 0 && style.visibility !== 'hidden' && style.display !== 'none';
    };
    const inputs = Array.from(document.querySelectorAll('input')).filter(visible);
    const username = inputs.find((input) => {
      const haystack = `${input.name || ''} ${input.id || ''} ${input.placeholder || ''} ${input.type || ''}`.toLowerCase();
      return input.type !== 'password' && /(username|email|phone|text)/.test(haystack);
    });
    const password = inputs.find((input) => input.type === 'password');
    return {
      visibleInputCount: inputs.length,
      usernameValueLength: username?.value?.length || 0,
      passwordValueLength: password?.value?.length || 0,
    };
  });
}

async function botFetch(path, options = {}) {
  return apiFetch(path, {
    ...options,
    headers: {
      Authorization: `Bearer ${botServiceToken}`,
      ...(options.headers || {}),
    },
  });
}

async function apiFetch(path, options = {}) {
  const response = await fetch(joinURL(apiBaseURL, path), {
    method: options.method || 'GET',
    headers: {
      Accept: 'application/json',
      ...(options.body ? { 'Content-Type': 'application/json' } : {}),
      ...(options.headers || {}),
    },
    body: options.body ? JSON.stringify(options.body) : undefined,
  });
  const text = await response.text();
  let payload = null;
  if (text) {
    try {
      payload = JSON.parse(text);
    } catch {
      throw new Error(`non-JSON API response ${response.status} from ${path}: ${text.slice(0, 160)}`);
    }
  }
  if (!response.ok || payload?.success === false) {
    throw new Error(`API ${path} returned ${response.status}: ${redactJSON(payload)}`);
  }
  return payload;
}

function requireData(payload, label) {
  if (!payload || payload.success !== true || payload.data == null) {
    throw new Error(`${label} did not contain success data: ${redactJSON(payload)}`);
  }
  return payload.data;
}

async function waitForSessionUserID(sessionID) {
  for (let i = 0; i < 30; i += 1) {
    const raw = queryScalar(
      `SELECT COALESCE(user_id::text, '') FROM public.group_admission_sessions WHERE id = ${sqlLiteral(sessionID)};`,
    );
    if (raw) return Number(raw);
    await sleep(500);
  }
  throw new Error(`session ${sessionID} was not linked to a user`);
}

async function waitForReleaseAction(sessionID) {
  for (let i = 0; i < 30; i += 1) {
    const payload = await botFetch(
      `/api/v1/bot/admission/sessions/pending?platform=qq&botSelfID=${encodeURIComponent(botSelfID)}&limit=20`,
    );
    const actions = Array.isArray(payload?.data) ? payload.data : [];
    const action = actions.find((item) => item.sessionID === sessionID && item.action === 'release');
    if (action) return action;
    await sleep(500);
  }
  throw new Error(`release pending action was not produced for session ${sessionID}`);
}

async function readAdmissionOTP(userID, schoolIDValue) {
  const redisContainer = process.env.REDIS_CONTAINER_NAME ||
    `${process.env.STACK_NAME || 'stuhelper-prod-parity'}-redis`;
  const redisUsername = process.env.REDIS_USERNAME || 'stuhelper_app';
  const redisPassword = process.env.REDIS_PASSWORD || '';
  if (!redisPassword) throw new Error('REDIS_PASSWORD is required to read prod-parity OTP');

  const args = [
    'exec',
    '-e',
    `REDISCLI_AUTH=${redisPassword}`,
    redisContainer,
    'redis-cli',
    '--user',
    redisUsername,
  ];
  if ((process.env.REDIS_TLS_ENABLED || 'true') === 'true') {
    args.push('--tls', '--cacert', '/redis-runtime/ca.crt');
  }
  args.push('GET', `admission:email_otp:${userID}:${schoolIDValue}`);
  const raw = run('docker', args, { secretCommand: 'docker exec redis-cli GET admission OTP' }).trim();
  if (!raw) throw new Error('admission OTP was not found in prod-parity Redis');
  const record = JSON.parse(raw);
  if (record.email !== studentEmail) {
    throw new Error(`OTP record email mismatch: ${record.email}`);
  }
  if (!/^[0-9]{6}$/.test(record.code || '')) {
    throw new Error('OTP record code is invalid');
  }
  return record.code;
}

async function querySessionState(sessionID) {
  const raw = queryScalar(`
    SELECT jsonb_build_object(
      'status', status,
      'verifiedAtPresent', verified_at IS NOT NULL,
      'cancelledAtPresent', cancelled_at IS NOT NULL,
      'lastBotErrorPresent', COALESCE(last_bot_error, '') <> ''
    )::text
    FROM public.group_admission_sessions
    WHERE id = ${sqlLiteral(sessionID)};
  `);
  if (!raw) throw new Error(`session ${sessionID} was not found`);
  return JSON.parse(raw);
}

function queryScalar(sql) {
  const databaseURL = materializeDatabaseURL(process.env.DATABASE_URL || '');
  if (!databaseURL) throw new Error('DATABASE_URL is required for prod-parity admission E2E');
  const postgresContainer = process.env.SHARED_POSTGRES_CONTAINER ||
    process.env.PROD_PARITY_POSTGRES_CONTAINER ||
    'stuhelper-prod-parity-postgres';
  return run('docker', [
    'exec',
    '-i',
    '-e',
    `DATABASE_URL=${databaseURL}`,
    postgresContainer,
    'sh',
    '-lc',
    'psql -X -v ON_ERROR_STOP=1 -At "$DATABASE_URL"',
  ], {
    input: sql,
    secretCommand: 'docker exec postgres psql',
  }).trim();
}

function materializeDatabaseURL(value) {
  let result = value;
  if (result.includes('REPLACE_WITH_STUHELPER_APP_DB_PASSWORD')) {
    const password = process.env.STUHELPER_APP_DB_PASSWORD || '';
    if (!password) throw new Error('STUHELPER_APP_DB_PASSWORD is required to materialize DATABASE_URL');
    result = result.replaceAll('REPLACE_WITH_STUHELPER_APP_DB_PASSWORD', encodeURIComponent(password));
  }
  if (result.includes('REPLACE_WITH')) {
    throw new Error('DATABASE_URL contains unresolved placeholders');
  }
  return result;
}

function attachBrowserDiagnostics(targetPage) {
  targetPage.on('console', (message) => {
    if (message.type() === 'error' && !isIgnoredConsoleText(message.text())) {
      browserEvents.consoleErrors.push(sanitizeErrorMessage(message.text()));
    }
  });
  targetPage.on('pageerror', (error) => {
    browserEvents.pageErrors.push(sanitizeErrorMessage(error.message));
  });
  targetPage.on('requestfailed', (request) => {
    const failure = request.failure()?.errorText || '';
    if (failure === 'net::ERR_ABORTED') return;
    browserEvents.requestFailures.push({
      url: redactURL(request.url()),
      method: request.method(),
      resourceType: request.resourceType(),
      failure,
    });
  });
  targetPage.on('response', (response) => {
    const url = response.url();
    if (!/\/api\/v1\//.test(url)) return;
    if (/\/api\/v1\/metrics\/(?:frontend-errors|vitals)(?:\?|$)/.test(url)) return;
    if (/\/api\/v1\/auth\/(?:me|refresh)(?:\?|$)/.test(url) && response.status() === 401) return;
    if (response.status() >= 400) {
      browserEvents.apiFailures.push({
        url: redactURL(url),
        status: response.status(),
      });
    }
  });
}

function assertNoBrowserDiagnostics() {
  const failures = [
    ...browserEvents.consoleErrors.map((item) => `console error: ${item}`),
    ...browserEvents.pageErrors.map((item) => `page error: ${item}`),
    ...browserEvents.requestFailures.map((item) => `request failed: ${JSON.stringify(item)}`),
    ...browserEvents.apiFailures.map((item) => `api failed: ${JSON.stringify(item)}`),
  ];
  if (failures.length > 0) {
    throw new Error(`browser diagnostics failed: ${failures.slice(0, 5).join('; ')}`);
  }
}

function isIgnoredConsoleText(text) {
  return /Failed to load resource: the server responded with a status of 4\d\d|5\d\d/.test(text);
}

async function expectText(targetPage, text) {
  await targetPage.getByText(text, { exact: false }).first().waitFor({
    state: 'visible',
    timeout: timeoutMs,
  });
}

async function waitForLocationPrefix(targetPage, prefix) {
  await targetPage.waitForFunction(
    (expectedPrefix) => window.location.href.startsWith(expectedPrefix),
    prefix,
    { timeout: timeoutMs },
  );
}

async function waitForAnyLocationPrefix(targetPage, prefixes) {
  await targetPage.waitForFunction(
    (expectedPrefixes) => expectedPrefixes.some((prefix) => window.location.href.startsWith(prefix)),
    prefixes,
    { timeout: timeoutMs },
  );
  const currentURL = targetPage.url();
  return prefixes.find((prefix) => currentURL.startsWith(prefix)) || currentURL;
}

async function writeEvidence(passed, payload) {
  const finishedAt = new Date();
  const evidence = {
    kind: 'admission_prod_sim_e2e',
    generatedAt: finishedAt.toISOString(),
    passed,
    durationMs: finishedAt.getTime() - startedAt.getTime(),
    targets: {
      apiBaseURL,
      webBaseURL,
      admissionBaseURL,
      ssoBaseURL,
    },
    ...payload,
  };
  await mkdir(dirname(evidenceFile), { recursive: true });
  await writeFile(evidenceFile, JSON.stringify(evidence, null, 2) + '\n', 'utf8');
}

async function describePage(targetPage) {
  if (!targetPage) return undefined;
  return {
    url: redactURL(targetPage.url()),
    title: await targetPage.title().catch(() => ''),
  };
}

function run(command, args, options = {}) {
  const result = spawnSync(command, args, {
    input: options.input,
    encoding: 'utf8',
    maxBuffer: 10 * 1024 * 1024,
  });
  if (result.status !== 0) {
    throw new Error(`${options.secretCommand || [command, ...args].join(' ')} failed: ${(result.stderr || result.stdout || '').slice(0, 400)}`);
  }
  return result.stdout || '';
}

function normalizeBaseURL(value) {
  return String(value || '').replace(/\/+$/, '');
}

function joinURL(base, path) {
  const normalizedPath = path.startsWith('/') ? path : `/${path}`;
  return `${normalizeBaseURL(base)}${normalizedPath}`;
}

function redactAdmissionURL(value) {
  try {
    const url = new URL(value);
    const token = url.pathname.startsWith('/verify/') ? '/verify/redacted' : url.pathname;
    return {
      host: url.host,
      path: token,
      pathAndQuery: url.search ? `${token}${url.search}` : token,
      hasQQQuery: url.searchParams.has('qq'),
    };
  } catch {
    return { host: '', path: '', pathAndQuery: '', hasQQQuery: false };
  }
}

function redactURL(value) {
  try {
	    const url = new URL(value);
	    if (url.pathname.startsWith('/verify/')) {
	      url.pathname = '/verify/redacted';
	    } else if (url.pathname.startsWith('/api/v1/admission/sessions/')) {
	      url.pathname = '/api/v1/admission/sessions/redacted';
	    }
	    for (const key of [
	      'state',
	      'code',
	      'code_challenge',
	      'code_verifier',
	      'token',
	      'access_token',
	      'refresh_token',
	      'id_token',
	      'authorization',
	    ]) {
	      if (url.searchParams.has(key)) {
	        url.searchParams.set(key, 'redacted');
	      }
	    }
	    return url.toString();
  } catch {
    return value;
  }
}

function sanitizeErrorMessage(value) {
  return String(value)
    .replace(/(\/verify\/)[A-Za-z0-9_-]+/g, '$1redacted')
    .replace(/(\/api\/v1\/admission\/sessions\/)[A-Za-z0-9_-]+/g, '$1redacted')
    .replace(/((?:state|code|code_challenge|code_verifier|token|access_token|refresh_token|id_token|authorization)=)[^&\s,)]+/gi, '$1redacted');
}

function redactJSON(value) {
  return JSON.stringify(value, (key, item) => {
    if (/token|secret|password|authorization/i.test(key)) return '[redacted]';
    if (typeof item === 'string' && item.startsWith('https://join.stuhelper.com/verify/')) {
      return redactAdmissionURL(item);
    }
    return item;
  });
}

function sqlLiteral(value) {
  return `'${String(value).replaceAll("'", "''")}'`;
}

function sleep(ms) {
  return new Promise((resolveSleep) => setTimeout(resolveSleep, ms));
}

function fail(message) {
  console.error(`[admission-prod-sim-e2e][error] ${message}`);
  process.exit(1);
}
