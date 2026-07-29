#!/usr/bin/env node
import assert from 'node:assert/strict';
import { Buffer } from 'node:buffer';
import { spawn } from 'node:child_process';
import { access, readFile, stat } from 'node:fs/promises';
import http from 'node:http';
import { constants as fsConstants } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const repoRoot = resolve(__dirname, '../../..');
const runner = resolve(repoRoot, 'infra/ops/casdoor-runtime-token-probe-runner.mjs');
const serverRunner = resolve(repoRoot, 'server/scripts/casdoor-runtime-token-probe-runner.mjs');
let lastAuthorizationNonce = '';
let lastAuthorizationRedirectURI = '';
let lastAuthorizationState = '';

await assertStaticContract();
const browserExecutable = await findBrowserExecutable();
if (!browserExecutable) {
  console.log('[casdoor-runtime-token-probe-runner-contract] browser not found; static assertions passed');
  process.exit(0);
}
await assertBrowserFlow(browserExecutable);
console.log('[casdoor-runtime-token-probe-runner-contract] all assertions passed');

async function assertStaticContract() {
  const mode = (await stat(runner)).mode;
  assert.ok(mode & 0o111, 'runner must be executable');
  const source = await readFile(runner, 'utf8');
  const serverMode = (await stat(serverRunner)).mode;
  assert.ok(serverMode & 0o111, 'server image runner copy must be executable');
  const serverSource = await readFile(serverRunner, 'utf8');
  assert.equal(serverSource, source, 'server image runner copy must stay in sync');
  for (const pattern of [
    'CASDOOR_TOKEN_PROBE_USERNAME',
    'CASDOOR_TOKEN_PROBE_PASSWORD',
    'CASDOOR_TOKEN_PROBE_BROWSER_HEADLESS',
    'CASDOOR_TOKEN_PROBE_BROWSER_EXECUTABLE_PATH',
    'CASDOOR_TOKEN_PROBE_BROWSER_NO_SANDBOX',
    'code_challenge_method',
    'authorization_code',
    'phone_verified',
    'stuhelper_student_verified',
    'businessClaims',
    'tokenClaims',
    'nonceVerified',
    'id_token nonce mismatch',
    'playwright',
  ]) {
    assert.ok(source.includes(pattern), `runner must contain ${pattern}`);
  }
  assert.ok(
    source.includes("../../clients/web/package.json") &&
      source.includes("requireFromWeb('@playwright/test')"),
    'host runner fallback must use the Web workspace installed by infrastructure CI',
  );
  assert.ok(
    !source.includes('../../clients/admin/package.json'),
    'host runner must not depend on the separately installed Admin workspace',
  );
  const retiredIDPPattern = Buffer.from('5a49544144454c', 'hex').toString('utf8');
  assert.ok(!source.includes(retiredIDPPattern), 'runner must not reference retired IDP identifiers');
}

async function findBrowserExecutable() {
  const candidates = [
    process.env.CASDOOR_TOKEN_PROBE_BROWSER_EXECUTABLE_PATH,
    '/usr/bin/google-chrome',
    '/usr/bin/google-chrome-stable',
    '/usr/bin/chromium',
    '/usr/bin/chromium-browser',
  ].filter(Boolean);
  for (const candidate of candidates) {
    try {
      await access(candidate, fsConstants.X_OK);
      return candidate;
    } catch {
      // try next candidate
    }
  }
  return '';
}

async function assertBrowserFlow(browserExecutable) {
  const server = http.createServer(mockCasdoorHandler);
  await new Promise((resolveServer) => server.listen(0, '127.0.0.1', resolveServer));
  const address = server.address();
  const issuer = `http://127.0.0.1:${address.port}`;
  const redirectURI = `${issuer}/callback`;
  try {
    const evidence = await runRunner({
      issuer,
      redirectURI,
      browserExecutable,
    });
    assert.equal(evidence.method, 'authorization_code');
    assert.equal(evidence.issuer, issuer);
    assert.deepEqual(evidence.businessClaims, []);
    assert.deepEqual(evidence.tokenClaims.id_token, ['aud', 'iss', 'nonce', 'sub']);
    assert.deepEqual(evidence.tokenClaims.access_token, ['iss', 'scope', 'sub']);
    assert.deepEqual(evidence.inspectedClaims, ['aud', 'iss', 'nonce', 'scope', 'sub']);
    assert.equal(evidence.metadata.source, 'casdoor-runtime-token-probe-runner.mjs');
    assert.equal(evidence.metadata.capture, 'playwright');
    assert.equal(evidence.metadata.nonceVerified, true);
  } finally {
    await new Promise((resolveClose) => server.close(resolveClose));
  }
}

function mockCasdoorHandler(req, res) {
  const url = new URL(req.url, 'http://127.0.0.1');
  if (req.method === 'GET' && url.pathname === '/.well-known/openid-configuration') {
    json(res, {
      authorization_endpoint: `${origin(req)}/authorize`,
      token_endpoint: `${origin(req)}/token`,
    });
    return;
  }
  if (req.method === 'GET' && url.pathname === '/authorize') {
    lastAuthorizationNonce = url.searchParams.get('nonce') || '';
    lastAuthorizationRedirectURI = url.searchParams.get('redirect_uri') || '';
    lastAuthorizationState = url.searchParams.get('state') || '';
    const html = `<!doctype html>
      <html>
        <body>
          <form method="post" action="/login">
            <input name="username" />
            <input name="password" type="password" />
            <button type="submit">Sign in</button>
          </form>
        </body>
      </html>`;
    res.writeHead(200, { 'content-type': 'text/html; charset=utf-8' });
    res.end(html);
    return;
  }
  if (req.method === 'POST' && url.pathname === '/login') {
    collectBody(req).then((body) => {
      const form = new URLSearchParams(body);
      if (form.get('username') !== 'probe-user' || form.get('password') !== 'probe-password') {
        res.writeHead(401);
        res.end('bad credentials');
        return;
      }
      assert.ok(lastAuthorizationRedirectURI, 'authorization redirect URI must be captured');
      const redirectURI = new URL(lastAuthorizationRedirectURI);
      redirectURI.searchParams.set('code', 'mock-code');
      redirectURI.searchParams.set('state', lastAuthorizationState);
      res.writeHead(302, { location: redirectURI.toString() });
      res.end();
    });
    return;
  }
  if (req.method === 'POST' && url.pathname === '/token') {
    collectBody(req).then((body) => {
      const form = new URLSearchParams(body);
      assert.equal(form.get('grant_type'), 'authorization_code');
      assert.equal(form.get('code'), 'mock-code');
      assert.equal(form.get('client_id'), 'mock-client');
      assert.equal(form.get('client_secret'), 'mock-secret');
      assert.ok(form.get('code_verifier')?.startsWith('stuhelper-token-probe-'));
      json(res, {
        id_token: unsignedJWT({
          iss: origin(req),
          sub: 'probe-user',
          aud: 'mock-client',
          nonce: lastAuthorizationNonce,
        }),
        access_token: unsignedJWT({ iss: origin(req), sub: 'probe-user', scope: 'openid' }),
        token_type: 'Bearer',
        expires_in: 300,
      });
    });
    return;
  }
  if (req.method === 'GET' && url.pathname === '/callback') {
    res.writeHead(200, { 'content-type': 'text/plain' });
    res.end('callback captured');
    return;
  }
  res.writeHead(404);
  res.end('not found');
}

function runRunner({ issuer, redirectURI, browserExecutable }) {
  const input = JSON.stringify({
    issuer,
    clientID: 'mock-client',
    clientSecret: 'mock-secret',
    redirectURIs: [redirectURI],
    scope: 'openid',
  });
  const child = spawn(process.execPath, [runner], {
    cwd: repoRoot,
    env: {
      ...process.env,
      CASDOOR_TOKEN_PROBE_USERNAME: 'probe-user',
      CASDOOR_TOKEN_PROBE_PASSWORD: 'probe-password',
      CASDOOR_TOKEN_PROBE_BROWSER_EXECUTABLE_PATH: browserExecutable,
      CASDOOR_TOKEN_PROBE_BROWSER_HEADLESS: 'true',
      CASDOOR_TOKEN_PROBE_LOGIN_TIMEOUT_SECONDS: '10',
      ENV_FILE: resolve(repoRoot, '.env.does-not-exist'),
    },
    stdio: ['pipe', 'pipe', 'pipe'],
  });
  child.stdin.end(input);
  let stdout = '';
  let stderr = '';
  child.stdout.on('data', (chunk) => {
    stdout += chunk;
  });
  child.stderr.on('data', (chunk) => {
    stderr += chunk;
  });
  return new Promise((resolve, reject) => {
    child.on('error', reject);
    child.on('close', (code) => {
      if (code !== 0) {
        reject(new Error(`runner exited ${code}\nstdout:\n${stdout}\nstderr:\n${stderr}`));
        return;
      }
      assert.ok(!stdout.includes('probe-password'), 'stdout must not contain probe password');
      assert.ok(!stderr.includes('probe-password'), 'stderr must not contain probe password');
      resolve(JSON.parse(stdout));
    });
  });
}

function origin(req) {
  return `http://${req.headers.host}`;
}

function json(res, payload) {
  res.writeHead(200, { 'content-type': 'application/json' });
  res.end(JSON.stringify(payload));
}

function collectBody(req) {
  const chunks = [];
  req.on('data', (chunk) => chunks.push(chunk));
  return new Promise((resolve) => {
    req.on('end', () => resolve(Buffer.concat(chunks).toString('utf8')));
  });
}

function unsignedJWT(claims) {
  const header = Buffer.from(JSON.stringify({ alg: 'none', typ: 'JWT' })).toString('base64url');
  const payload = Buffer.from(JSON.stringify(claims)).toString('base64url');
  return `${header}.${payload}.`;
}
