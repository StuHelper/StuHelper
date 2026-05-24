#!/usr/bin/env node
import { createRequire } from 'node:module';
import { webcrypto } from 'node:crypto';
import { URL } from 'node:url';

const DEFAULT_USERNAME_SELECTOR =
  'input[name="username"], input[name="email"], input[type="email"], input[type="text"]';
const DEFAULT_PASSWORD_SELECTOR = 'input[name="password"], input[type="password"]';
const DEFAULT_SUBMIT_SELECTOR =
  'button[type="submit"], input[type="submit"], button:has-text("Sign in"), button:has-text("Log in"), button:has-text("登录")';
const DEFAULT_CONSENT_SELECTOR =
  'button:has-text("Authorize"), button:has-text("Allow"), button:has-text("Accept"), button:has-text("同意"), button:has-text("授权")';

const FORBIDDEN_CLAIMS = new Set([
  'phone',
  'phone_number',
  'phone_verified',
  'phone_number_verified',
  'identity_verified',
  'identity_type',
  'student_verified',
  'school',
  'school_id',
  'school_name',
  'qq',
  'qq_binding',
  'stuhelper_identity_verified',
  'stuhelper_identity_type',
  'stuhelper_student_verified',
  'stuhelper_student_school',
  'stuhelper_student_school_id',
  'stuhelper_student_school_name',
]);

function log(message) {
  process.stderr.write(`[casdoor-runtime-token-probe-runner] ${message}\n`);
}

function fail(message) {
  process.stderr.write(`[casdoor-runtime-token-probe-runner][error] ${message}\n`);
  process.exit(1);
}

function notConfigured(message) {
  process.stderr.write(`[casdoor-runtime-token-probe-runner][not-configured] ${message}\n`);
  process.exit(78);
}

function env(name, fallback = '') {
  return (process.env[name] ?? fallback).trim();
}

function envBool(name, fallback) {
  const value = env(name);
  if (value === '') return fallback;
  return ['1', 'true', 'yes', 'on'].includes(value.toLowerCase());
}

function envInt(name, fallback) {
  const value = env(name);
  if (value === '') return fallback;
  const parsed = Number.parseInt(value, 10);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback;
}

function canonicalClaim(raw) {
  let value = String(raw ?? '').trim();
  value = value.replace(/([a-z0-9])([A-Z])/g, '$1_$2');
  value = value.toLowerCase().replaceAll('-', '_').replaceAll('.', '_').replaceAll(' ', '_');
  while (value.includes('__')) value = value.replaceAll('__', '_');
  return value.replace(/^_+|_+$/g, '');
}

function uniqueSorted(values) {
  return [...new Set(values.filter(Boolean))].sort();
}

function base64URLDecode(value) {
  return Buffer.from(value, 'base64url').toString('utf8');
}

function decodeJWTClaims(raw) {
  if (typeof raw !== 'string' || raw.split('.').length < 2) return undefined;
  const [, payload] = raw.split('.');
  return JSON.parse(base64URLDecode(payload));
}

function inspectTokenResponse(tokenResponse, issuer, expectedNonce = '') {
  if (!tokenResponse || typeof tokenResponse !== 'object') {
    throw new Error('token endpoint returned an invalid JSON object');
  }
  if (tokenResponse.error) {
    throw new Error(`token endpoint returned error: ${tokenResponse.error}`);
  }
  if (typeof tokenResponse.id_token !== 'string' || tokenResponse.id_token === '') {
    throw new Error('token response missing id_token');
  }
  const idTokenClaims = decodeJWTClaims(tokenResponse.id_token);
  if (!idTokenClaims || typeof idTokenClaims !== 'object') {
    throw new Error('id_token payload is not a JSON object');
  }
  if (expectedNonce && idTokenClaims.nonce !== expectedNonce) {
    throw new Error('id_token nonce mismatch');
  }

  const tokens = {
    id_token: tokenResponse.id_token,
    access_token: tokenResponse.access_token,
  };
  const tokenClaims = {};
  const inspectedClaims = [];
  const businessClaims = [];
  for (const [label, token] of Object.entries(tokens)) {
    if (typeof token !== 'string' || token === '' || token.split('.').length < 2) {
      continue;
    }
    const claims = decodeJWTClaims(token);
    const keys = uniqueSorted(Object.keys(claims).map(canonicalClaim));
    tokenClaims[label] = keys;
    inspectedClaims.push(...keys);
    businessClaims.push(...keys.filter((key) => FORBIDDEN_CLAIMS.has(key)));
    log(`${label} inspected claims: ${keys.join(', ')}`);
  }

  return {
    method: 'authorization_code',
    issuer,
    inspectedClaims: uniqueSorted(inspectedClaims),
    businessClaims: uniqueSorted(businessClaims),
    tokenClaims,
    metadata: {
      source: 'casdoor-runtime-token-probe-runner.mjs',
      capture: 'playwright',
      nonceVerified: Boolean(expectedNonce),
    },
  };
}

async function readProbeInput() {
  if (process.stdin.isTTY) return {};
  const chunks = [];
  for await (const chunk of process.stdin) chunks.push(chunk);
  const raw = Buffer.concat(chunks).toString('utf8').trim();
  if (raw === '') return {};
  return JSON.parse(raw);
}

function normalizeProbeConfig(input) {
  const redirectURI = env('CASDOOR_TOKEN_PROBE_REDIRECT_URI') || input.redirectURIs?.[0] || '';
  const cfg = {
    issuer: env('CASDOOR_ISSUER') || input.issuer || '',
    clientID: env('CASDOOR_TOKEN_PROBE_CLIENT_ID') || input.clientID || '',
    clientSecret: env('CASDOOR_TOKEN_PROBE_CLIENT_SECRET') || input.clientSecret || '',
    redirectURI,
    scope: env('CASDOOR_TOKEN_PROBE_SCOPE', input.scope || 'openid'),
    username: env('CASDOOR_TOKEN_PROBE_USERNAME'),
    password: env('CASDOOR_TOKEN_PROBE_PASSWORD'),
    headless: envBool('CASDOOR_TOKEN_PROBE_BROWSER_HEADLESS', true),
    timeoutMs: envInt('CASDOOR_TOKEN_PROBE_LOGIN_TIMEOUT_SECONDS', 30) * 1000,
    usernameSelector: env('CASDOOR_TOKEN_PROBE_USERNAME_SELECTOR', DEFAULT_USERNAME_SELECTOR),
    passwordSelector: env('CASDOOR_TOKEN_PROBE_PASSWORD_SELECTOR', DEFAULT_PASSWORD_SELECTOR),
    submitSelector: env('CASDOOR_TOKEN_PROBE_SUBMIT_SELECTOR', DEFAULT_SUBMIT_SELECTOR),
    consentSelector: env('CASDOOR_TOKEN_PROBE_CONSENT_SELECTOR', DEFAULT_CONSENT_SELECTOR),
    browserExecutablePath: env('CASDOOR_TOKEN_PROBE_BROWSER_EXECUTABLE_PATH'),
    browserNoSandbox: envBool('CASDOOR_TOKEN_PROBE_BROWSER_NO_SANDBOX', true),
  };

  const missing = [];
  for (const key of ['issuer', 'clientID', 'redirectURI', 'username', 'password']) {
    if (!cfg[key]) missing.push(key);
  }
  if (missing.length > 0) {
    notConfigured(`missing required runtime probe config: ${missing.join(', ')}`);
  }
  if (cfg.scope !== 'openid') {
    fail('CASDOOR_TOKEN_PROBE_SCOPE must be exactly openid');
  }
  return cfg;
}

async function fetchJSON(url, options = {}) {
  const response = await fetch(url, options);
  const body = await response.text();
  if (!response.ok) {
    throw new Error(`${url} returned HTTP ${response.status}: ${body.slice(0, 512)}`);
  }
  return JSON.parse(body);
}

async function fetchDiscovery(issuer) {
  const metadataURL = `${issuer.replace(/\/+$/, '')}/.well-known/openid-configuration`;
  log(`fetching OIDC metadata: ${metadataURL}`);
  const metadata = await fetchJSON(metadataURL);
  if (!metadata.authorization_endpoint || !metadata.token_endpoint) {
    throw new Error('OIDC metadata missing authorization_endpoint or token_endpoint');
  }
  return metadata;
}

function randomURLSafe(bytes = 48) {
  return Buffer.from(webcrypto.getRandomValues(new Uint8Array(bytes))).toString('base64url');
}

async function sha256Base64URL(value) {
  const digest = await webcrypto.subtle.digest('SHA-256', new TextEncoder().encode(value));
  return Buffer.from(digest).toString('base64url');
}

async function buildAuthorizationRequest(cfg, authorizationEndpoint) {
  const verifier = `stuhelper-token-probe-${randomURLSafe(48)}`;
  const challenge = await sha256Base64URL(verifier);
  const state = env('CASDOOR_TOKEN_PROBE_STATE', `stuhelper-token-probe-${randomURLSafe(16)}`);
  const nonce = env('CASDOOR_TOKEN_PROBE_NONCE', `stuhelper-token-probe-${randomURLSafe(16)}`);
  const url = new URL(authorizationEndpoint);
  url.searchParams.set('response_type', 'code');
  url.searchParams.set('client_id', cfg.clientID);
  url.searchParams.set('redirect_uri', cfg.redirectURI);
  url.searchParams.set('scope', 'openid');
  url.searchParams.set('state', state);
  url.searchParams.set('nonce', nonce);
  url.searchParams.set('code_challenge', challenge);
  url.searchParams.set('code_challenge_method', 'S256');
  return { url: url.toString(), verifier, state, nonce };
}

async function loadPlaywright() {
  try {
    const direct = await import('playwright');
    return direct;
  } catch {
    try {
      const core = await import('playwright-core');
      return core;
    } catch {
      const requireFromAdmin = createRequire(new URL('../../clients/admin/package.json', import.meta.url));
      return requireFromAdmin('playwright');
    }
  }
}

function codeFromURL(rawURL, redirectURI, expectedState) {
  if (!rawURL || !rawURL.startsWith(redirectURI)) return undefined;
  const parsed = new URL(rawURL);
  if (expectedState && parsed.searchParams.get('state') !== expectedState) {
    throw new Error('authorization response state mismatch');
  }
  const error = parsed.searchParams.get('error');
  if (error) {
    throw new Error(`authorization endpoint returned error: ${error}`);
  }
  return parsed.searchParams.get('code') || undefined;
}

async function firstVisible(page, selector, timeout = 1500) {
  const locator = page.locator(selector).first();
  try {
    await locator.waitFor({ state: 'visible', timeout });
    return locator;
  } catch {
    return undefined;
  }
}

async function captureAuthorizationCode(cfg, request) {
  const { chromium } = await loadPlaywright();
  const launchOptions = {
    headless: cfg.headless,
    args: cfg.browserNoSandbox ? ['--no-sandbox', '--disable-dev-shm-usage'] : ['--disable-dev-shm-usage'],
  };
  if (cfg.browserExecutablePath) {
    launchOptions.executablePath = cfg.browserExecutablePath;
  }
  const browser = await chromium.launch(launchOptions);
  const context = await browser.newContext();
  const page = await context.newPage();
  let capturedURL = '';

  const rememberURL = (rawURL) => {
    const code = codeFromURL(rawURL, cfg.redirectURI, request.state);
    if (code) capturedURL = rawURL;
  };
  page.on('framenavigated', (frame) => {
    try {
      rememberURL(frame.url());
    } catch (error) {
      capturedURL = `error:${error.message}`;
    }
  });
  await page.route('**/*', async (route) => {
    const rawURL = route.request().url();
    try {
      rememberURL(rawURL);
    } catch (error) {
      capturedURL = `error:${error.message}`;
    }
    if (capturedURL && rawURL.startsWith(cfg.redirectURI)) {
      await route.abort();
      return;
    }
    await route.continue();
  });

  try {
    await page.goto(request.url, { waitUntil: 'domcontentloaded', timeout: cfg.timeoutMs });
    rememberURL(page.url());
    if (!capturedURL) {
      const username = await firstVisible(page, cfg.usernameSelector, Math.min(cfg.timeoutMs, 8000));
      if (username) {
        await username.fill(cfg.username);
        const password = await firstVisible(page, cfg.passwordSelector, 5000);
        if (!password) throw new Error('password field was not visible on Casdoor login page');
        await password.fill(cfg.password);
        const submit = await firstVisible(page, cfg.submitSelector, 5000);
        if (!submit) throw new Error('submit button was not visible on Casdoor login page');
        await submit.click();
      }
    }

    const deadline = Date.now() + cfg.timeoutMs;
    while (!capturedURL && Date.now() < deadline) {
      rememberURL(page.url());
      const consent = await firstVisible(page, cfg.consentSelector, 300);
      if (consent) {
        await consent.click();
      }
      await page.waitForTimeout(250);
    }
    if (capturedURL.startsWith('error:')) {
      throw new Error(capturedURL.slice('error:'.length));
    }
    const code = codeFromURL(capturedURL || page.url(), cfg.redirectURI, request.state);
    if (!code) {
      throw new Error('authorization code was not captured before timeout');
    }
    return code;
  } finally {
    await browser.close();
  }
}

async function exchangeCode(cfg, tokenEndpoint, code, verifier) {
  const body = new URLSearchParams();
  body.set('grant_type', 'authorization_code');
  body.set('code', code);
  body.set('redirect_uri', cfg.redirectURI);
  body.set('client_id', cfg.clientID);
  body.set('code_verifier', verifier);
  if (cfg.clientSecret) {
    body.set('client_secret', cfg.clientSecret);
  }
  return fetchJSON(tokenEndpoint, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body,
  });
}

async function main() {
  const input = await readProbeInput();
  const cfg = normalizeProbeConfig(input);
  const metadata = await fetchDiscovery(cfg.issuer);
  const authorizationRequest = await buildAuthorizationRequest(cfg, metadata.authorization_endpoint);
  log('capturing authorization code with Playwright');
  const code = await captureAuthorizationCode(cfg, authorizationRequest);
  log('authorization code captured; exchanging token response');
  const tokenResponse = await exchangeCode(
    cfg,
    metadata.token_endpoint,
    code,
    authorizationRequest.verifier,
  );
  const evidence = inspectTokenResponse(tokenResponse, cfg.issuer, authorizationRequest.nonce);
  process.stdout.write(`${JSON.stringify(evidence)}\n`);
}

main().catch((error) => {
  fail(error?.message || String(error));
});
