#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
SMOKE_SCRIPT="${REPO_ROOT}/infra/ops/public-web-auth-browser-smoke.mjs"
PROD_DEPLOY_SCRIPT="${REPO_ROOT}/infra/ops/prod-deploy.sh"
REMOTE_PREFLIGHT_SCRIPT="${REPO_ROOT}/infra/ops/remote-preflight.sh"
PROD_ENV_EXAMPLE="${REPO_ROOT}/.env.prod.example"
RELEASE_RUNBOOK="${REPO_ROOT}/docs/guides/release-runbook.md"

fail() {
  echo "[public-web-auth-browser-smoke-contract][error] $*" >&2
  exit 1
}

cleanup() {
  if [[ -n "${server_pid:-}" ]]; then
    kill "${server_pid}" >/dev/null 2>&1 || true
    wait "${server_pid}" >/dev/null 2>&1 || true
  fi
  [[ -n "${tmpdir:-}" ]] && rm -rf "${tmpdir}"
}
trap cleanup EXIT

assert_contains() {
  local file="$1"
  local pattern="$2"
  if ! grep -Eq -- "${pattern}" "${file}"; then
    fail "expected ${file} to contain pattern: ${pattern}"
  fi
}

assert_evidence() {
  local file="$1"
  python3 - "${file}" <<'PY' || fail "browser smoke evidence assertion failed"
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as fh:
    evidence = json.load(fh)

def require(condition, message):
    if not condition:
        raise SystemExit(message)

checks = evidence.get("checks", [])
names = {item.get("name"): item for item in checks}
require(evidence.get("passed") is True, "smoke did not pass")
require(evidence.get("summary", {}).get("passed") == 9, "passed count")
require(evidence.get("summary", {}).get("failed") == 0, "failed count")
for name in (
    "web-login-page-renders",
    "developer-apps-route-redirects-to-login",
    "identity-route-redirects-to-login",
    "header-login-click-starts-sso",
    "login-signup-click-starts-sso-signup",
    "join-root-route-returns-404",
    "join-main-web-route-returns-404",
    "join-verify-route-renders-spa",
    "join-mobile-camera-route-allows-camera",
):
    require(name in names, f"missing {name}")
    require(names[name].get("passed") is True, f"{name} did not pass")

targets = evidence.get("targets", {})
require(targets.get("webBaseURL", "").endswith("/web"), "web target")
require(targets.get("joinBaseURL", "").endswith("/join"), "join target")
require(targets.get("ssoBaseURL", "").endswith("/sso"), "sso target")

mobile_camera = names["join-mobile-camera-route-allows-camera"]
camera_permissions = {
    item.get("feature"): item
    for item in mobile_camera.get("browserPermissions", [])
}
media_captures = {
    item.get("name"): item
    for item in mobile_camera.get("mediaCaptures", [])
}
require("camera" in camera_permissions, "mobile camera browser permission evidence")
require(camera_permissions["camera"].get("supported") is True, "camera policy API supported")
require(camera_permissions["camera"].get("allowed") is True, "camera allowed by browser policy")
require("camera" in media_captures, "mobile camera media capture evidence")
require(media_captures["camera"].get("supported") is True, "camera media capture API supported")
require(media_captures["camera"].get("success") is True, "camera media capture succeeds")
require(media_captures["camera"].get("videoTrackCount", 0) >= 1, "camera media capture video track")
PY
}

for file in \
  "${SMOKE_SCRIPT}" \
  "${PROD_DEPLOY_SCRIPT}" \
  "${REMOTE_PREFLIGHT_SCRIPT}" \
  "${PROD_ENV_EXAMPLE}" \
  "${RELEASE_RUNBOOK}"; do
  [[ -f "${file}" ]] || fail "missing file: ${file}"
done

node --check "${SMOKE_SCRIPT}"
[[ -x "${SMOKE_SCRIPT}" ]] || fail "browser smoke script must be executable"

assert_contains "${SMOKE_SCRIPT}" '@playwright/test'
assert_contains "${SMOKE_SCRIPT}" 'WEB_PUBLIC_URL.*https://stuhelper\.com'
assert_contains "${SMOKE_SCRIPT}" 'ADMISSION_PUBLIC_BASE_URL.*https://join\.stuhelper\.com'
assert_contains "${SMOKE_SCRIPT}" 'SSO_PUBLIC_BASE_URL.*https://sso\.stuhelper\.com'
assert_contains "${SMOKE_SCRIPT}" 'developer-apps-route-redirects-to-login'
assert_contains "${SMOKE_SCRIPT}" 'identity-route-redirects-to-login'
assert_contains "${SMOKE_SCRIPT}" "redirect.*identity"
assert_contains "${SMOKE_SCRIPT}" 'header-login-click-starts-sso'
assert_contains "${SMOKE_SCRIPT}" 'login-signup-click-starts-sso-signup'
assert_contains "${SMOKE_SCRIPT}" 'join-root-route-returns-404'
assert_contains "${SMOKE_SCRIPT}" 'join-main-web-route-returns-404'
assert_contains "${SMOKE_SCRIPT}" 'join-verify-route-renders-spa'
assert_contains "${SMOKE_SCRIPT}" 'join-mobile-camera-route-allows-camera'
if grep -Eq 'probeQQ|PUBLIC_WEB_AUTH_BROWSER_SMOKE_PROBE_QQ' "${SMOKE_SCRIPT}" "${PROD_ENV_EXAMPLE}"; then
  fail "browser smoke must not carry qq query probe configuration"
fi
assert_contains "${SMOKE_SCRIPT}" 'expectedResponseHeaders'
assert_contains "${SMOKE_SCRIPT}" 'expectedBrowserPermissions'
assert_contains "${SMOKE_SCRIPT}" 'expectedMediaCaptures'
assert_contains "${SMOKE_SCRIPT}" 'browserPermissions'
assert_contains "${SMOKE_SCRIPT}" 'mediaCaptures'
assert_contains "${SMOKE_SCRIPT}" 'isUnexpectedFailedRequest'
assert_contains "${SMOKE_SCRIPT}" 'isAPIURL'
assert_contains "${SMOKE_SCRIPT}" 'getUserMedia'
assert_contains "${SMOKE_SCRIPT}" 'use-fake-device-for-media-stream'
assert_contains "${SMOKE_SCRIPT}" 'permissions-policy'
assert_contains "${SMOKE_SCRIPT}" 'camera=\(\)'
assert_contains "${SMOKE_SCRIPT}" 'camera=\(self\)'
assert_contains "${SMOKE_SCRIPT}" 'Permissions policy violation'
assert_contains "${SMOKE_SCRIPT}" 'manual-camera-handoffs'
assert_contains "${SMOKE_SCRIPT}" 'PUBLIC_WEB_AUTH_BROWSER_SMOKE_ALLOW_LOCAL_TARGETS'
assert_contains "${SMOKE_SCRIPT}" 'PUBLIC_WEB_AUTH_BROWSER_EXECUTABLE_PATH'
assert_contains "${SMOKE_SCRIPT}" 'rejectLoopbackResolvedTarget'
assert_contains "${SMOKE_SCRIPT}" 'resolves to loopback'
assert_contains "${SMOKE_SCRIPT}" 'client_id.*stuhelper-web'
assert_contains "${SMOKE_SCRIPT}" '/login/oauth/authorize'
assert_contains "${SMOKE_SCRIPT}" '/signup/oauth/authorize'
assert_contains "${SMOKE_SCRIPT}" 'SSO login page does not expose password login'
assert_contains "${SMOKE_SCRIPT}" 'SSO signup page does not expose username/password signup fields'
assert_contains "${PROD_DEPLOY_SCRIPT}" 'PUBLIC_WEB_AUTH_BROWSER_SMOKE_ENABLED'
assert_contains "${PROD_DEPLOY_SCRIPT}" 'public-web-auth-browser-smoke\.mjs'
assert_contains "${REMOTE_PREFLIGHT_SCRIPT}" 'PUBLIC_WEB_AUTH_BROWSER_SMOKE_PREFLIGHT_ENABLED'
assert_contains "${REMOTE_PREFLIGHT_SCRIPT}" 'public-web-auth-browser-smoke\.mjs'
assert_contains "${PROD_ENV_EXAMPLE}" '^PUBLIC_WEB_AUTH_BROWSER_SMOKE_ENABLED=true$'
assert_contains "${PROD_ENV_EXAMPLE}" '^PUBLIC_WEB_AUTH_BROWSER_SMOKE_ALLOW_LOCAL_TARGETS=false$'
assert_contains "${RELEASE_RUNBOOK}" 'public-web-auth-browser-smoke\.mjs'

tmpdir="$(mktemp -d)"
evidence_file="${tmpdir}/public-web-auth-browser-smoke-evidence.json"
bad_evidence_file="${tmpdir}/public-web-auth-browser-smoke-bad-camera-evidence.json"
bad_api_evidence_file="${tmpdir}/public-web-auth-browser-smoke-bad-api-evidence.json"
port_file="${tmpdir}/port"
camera_policy_file="${tmpdir}/camera-policy"
api_failure_file="${tmpdir}/api-failure"
printf '%s\n' 'camera=(self), microphone=(), geolocation=(), payment=()' >"${camera_policy_file}"
printf '%s\n' 'ok' >"${api_failure_file}"

cat >"${tmpdir}/fake-public-web-server.mjs" <<'JS'
import http from 'node:http';
import { readFileSync, writeFileSync } from 'node:fs';

const portFile = process.argv[2];
const cameraPolicyFile = process.argv[3];
const apiFailureFile = process.argv[4];
const defaultCameraPolicy = 'camera=(self), microphone=(), geolocation=(), payment=()';
const verifyCameraPolicy = 'camera=(), microphone=(), geolocation=(), payment=()';

function html(body, status = 200) {
  return {
    status,
    headers: { 'content-type': 'text/html; charset=utf-8' },
    body: `<!doctype html><html><head><title>StuHelper</title></head><body>${body}</body></html>`,
  };
}

function currentCameraPolicy() {
  try {
    return readFileSync(cameraPolicyFile, 'utf8').trim() || defaultCameraPolicy;
  } catch {
    return defaultCameraPolicy;
  }
}

function currentAPIFailureMode() {
  try {
    return readFileSync(apiFailureFile, 'utf8').trim() || 'ok';
  } catch {
    return 'ok';
  }
}

function withCameraPolicy(result, policy = currentCameraPolicy()) {
  result.headers['permissions-policy'] = policy;
  return result;
}

const server = http.createServer((request, response) => {
  const url = new URL(request.url, `http://${request.headers.host}`);
  let result;

  if (
    url.pathname === '/web/api/v1/auth/me' ||
    url.pathname === '/web/api/v1/auth/refresh' ||
    url.pathname === '/join/api/v1/auth/me' ||
    url.pathname === '/join/api/v1/auth/refresh'
  ) {
    result = {
      status: 401,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ success: false, error: { code: 'A0010100', message: 'login required' } }),
    };
  } else if (url.pathname === '/web/api/v1/auth/login' || url.pathname === '/join/api/v1/auth/login') {
    result = {
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        success: true,
        data: {
          state: 'browser-smoke-state',
          url: `http://127.0.0.1:${server.address().port}/sso/login/oauth/authorize?client_id=stuhelper-web&redirect_uri=${encodeURIComponent(`http://127.0.0.1:${server.address().port}/web/api/v1/auth/callback`)}&state=browser-smoke-state`,
        },
      }),
    };
  } else if (url.pathname === '/web/api/v1/auth/signup' || url.pathname === '/join/api/v1/auth/signup') {
    result = {
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        success: true,
        data: {
          state: 'browser-smoke-signup-state',
          url: `http://127.0.0.1:${server.address().port}/sso/signup/oauth/authorize?client_id=stuhelper-web&redirect_uri=${encodeURIComponent(`http://127.0.0.1:${server.address().port}/web/api/v1/auth/callback`)}&state=browser-smoke-signup-state`,
        },
      }),
    };
  } else if (url.pathname === '/sso/login/oauth/authorize') {
    result = html(`
      <main>
        <h1>Casdoor SSO</h1>
        <label>username, Email or phone <input name="username" /></label>
        <label>Password <input name="password" type="password" /></label>
        <button>Sign In</button>
        <a href="/sso/signup/oauth/authorize?client_id=stuhelper-web">sign up now</a>
      </main>
    `);
  } else if (url.pathname === '/sso/signup/oauth/authorize') {
    result = html(`
      <main>
        <h1>Casdoor SSO Signup</h1>
        <label>Username <input name="username" /></label>
        <label>Display name <input name="displayName" /></label>
        <label>Password <input name="password" type="password" /></label>
        <label>Confirm <input name="confirm" type="password" /></label>
        <button>Sign Up</button>
      </main>
    `);
  } else if (url.pathname === '/web/' || url.pathname === '/web') {
    result = html('<main><h1>StuHelper</h1><a href="/web/login?redirect=/">登录</a></main>');
  } else if (url.pathname === '/web/login') {
    result = html(`
      <main>
        <h1>StuHelper 统一登录</h1>
        <button onclick="fetch('/web/api/v1/auth/login?app=web&redirect=' + encodeURIComponent(new URLSearchParams(location.search).get('redirect') || '/')).then(r => r.json()).then(j => location.href = j.data.url)">使用统一身份认证登录</button>
        <button onclick="fetch('/web/api/v1/auth/signup?app=web&redirect=' + encodeURIComponent(new URLSearchParams(location.search).get('redirect') || '/')).then(r => r.json()).then(j => location.href = j.data.url)">注册账号</button>
      </main>
    `);
  } else if (url.pathname === '/join/' || url.pathname === '/join/developers/apps') {
    result = html('<main>not found</main>', 404);
  } else if (url.pathname === '/web/developers/apps') {
    result = html('<script>location.replace("/web/login?redirect=/developers/apps")</script><main>redirecting</main>');
  } else if (url.pathname === '/web/identity') {
    result = html('<script>location.replace("/web/login?redirect=/identity")</script><main>redirecting</main>');
  } else if (url.pathname === '/join/api/v1/metrics/vitals') {
    if (currentAPIFailureMode() === 'drop-metrics') {
      request.socket.destroy();
      return;
    }
    result = {
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: '{"ok":true}',
    };
  } else if (url.pathname === '/join/verify/__stuhelper_browser_smoke_manual_probe__' && url.search === '') {
    result = withCameraPolicy(html(`
      <main><h1>StuHelper 加群验证</h1><button>使用统一身份认证登录</button></main>
      <script>
        fetch('/join/api/v1/metrics/vitals?browser=smoke').catch(() => {});
      </script>
    `), verifyCameraPolicy);
  } else if (url.pathname === '/join/student-verification/manual-camera/__stuhelper_browser_smoke_manual_probe__') {
    result = withCameraPolicy(html('<main><h1>手机拍摄认证材料</h1><p>无法打开拍摄链接</p></main>'));
  } else {
    result = html(`<main>unexpected ${url.pathname}</main>`, 500);
  }

  response.writeHead(result.status, result.headers);
  response.end(result.body);
});

server.listen(0, '127.0.0.1', () => {
  writeFileSync(portFile, String(server.address().port));
});
JS

node "${tmpdir}/fake-public-web-server.mjs" "${port_file}" "${camera_policy_file}" "${api_failure_file}" &
server_pid=$!
for _ in {1..50}; do
  [[ -s "${port_file}" ]] && break
  sleep 0.1
done
[[ -s "${port_file}" ]] || fail "fake browser smoke server did not start"

port="$(cat "${port_file}")"
base_url="http://127.0.0.1:${port}"

local_refused_output="$(
  WEB_PUBLIC_URL="${base_url}/web" \
  ADMISSION_PUBLIC_BASE_URL="${base_url}/join" \
  SSO_PUBLIC_BASE_URL="${base_url}/sso" \
  PUBLIC_WEB_AUTH_BROWSER_SMOKE_EVIDENCE_FILE="${evidence_file}" \
  "${SMOKE_SCRIPT}" 2>&1
)" && fail "browser smoke unexpectedly allowed local targets without explicit override"

printf '%s\n' "${local_refused_output}" | grep -Eq 'PUBLIC_WEB_AUTH_BROWSER_SMOKE_ALLOW_LOCAL_TARGETS=true|WEB_PUBLIC_URL must be exactly https://stuhelper\.com' || \
  fail "local target refusal did not explain override"

WEB_PUBLIC_URL="${base_url}/web" \
ADMISSION_PUBLIC_BASE_URL="${base_url}/join" \
SSO_PUBLIC_BASE_URL="${base_url}/sso" \
PUBLIC_WEB_AUTH_BROWSER_SMOKE_EVIDENCE_FILE="${evidence_file}" \
PUBLIC_WEB_AUTH_BROWSER_SMOKE_ALLOW_LOCAL_TARGETS=true \
PUBLIC_WEB_AUTH_BROWSER_SMOKE_TIMEOUT_MS=10000 \
"${SMOKE_SCRIPT}"

assert_evidence "${evidence_file}"

printf '%s\n' 'drop-metrics' >"${api_failure_file}"
bad_api_output="$(
  WEB_PUBLIC_URL="${base_url}/web" \
  ADMISSION_PUBLIC_BASE_URL="${base_url}/join" \
  SSO_PUBLIC_BASE_URL="${base_url}/sso" \
  PUBLIC_WEB_AUTH_BROWSER_SMOKE_EVIDENCE_FILE="${bad_api_evidence_file}" \
  PUBLIC_WEB_AUTH_BROWSER_SMOKE_ALLOW_LOCAL_TARGETS=true \
  PUBLIC_WEB_AUTH_BROWSER_SMOKE_TIMEOUT_MS=10000 \
  "${SMOKE_SCRIPT}" 2>&1
)" && fail "browser smoke unexpectedly passed with a failed admission API request"

printf '%s\n' "${bad_api_output}" | grep -q 'join-verify-route-renders-spa' || \
  fail "bad API run did not report the join verify route check"

python3 - "${bad_api_evidence_file}" <<'PY' || fail "bad API evidence assertion failed"
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as fh:
    evidence = json.load(fh)

checks = {item.get("name"): item for item in evidence.get("checks", [])}
join_verify = checks.get("join-verify-route-renders-spa") or {}

if evidence.get("passed") is not False:
    raise SystemExit("bad API smoke should fail")
if join_verify.get("passed") is not False:
    raise SystemExit("join verify check should fail")
if not any("/join/api/v1/metrics/vitals" in failure for failure in join_verify.get("failures", [])):
    raise SystemExit("missing failed metrics API request failure")
PY

printf '%s\n' 'ok' >"${api_failure_file}"
printf '%s\n' 'camera=(), microphone=(), geolocation=(), payment=()' >"${camera_policy_file}"
bad_camera_output="$(
  WEB_PUBLIC_URL="${base_url}/web" \
  ADMISSION_PUBLIC_BASE_URL="${base_url}/join" \
  SSO_PUBLIC_BASE_URL="${base_url}/sso" \
  PUBLIC_WEB_AUTH_BROWSER_SMOKE_EVIDENCE_FILE="${bad_evidence_file}" \
  PUBLIC_WEB_AUTH_BROWSER_SMOKE_ALLOW_LOCAL_TARGETS=true \
  PUBLIC_WEB_AUTH_BROWSER_SMOKE_TIMEOUT_MS=10000 \
  "${SMOKE_SCRIPT}" 2>&1
)" && fail "browser smoke unexpectedly passed with camera=() permissions policy"

printf '%s\n' "${bad_camera_output}" | grep -q 'join-mobile-camera-route-allows-camera' || \
  fail "bad camera policy run did not report the mobile camera check"

python3 - "${bad_evidence_file}" <<'PY' || fail "bad camera policy evidence assertion failed"
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as fh:
    evidence = json.load(fh)

checks = {item.get("name"): item for item in evidence.get("checks", [])}
mobile_camera = checks.get("join-mobile-camera-route-allows-camera") or {}
permissions = {
    item.get("feature"): item
    for item in mobile_camera.get("browserPermissions", [])
}
camera = permissions.get("camera") or {}
captures = {
    item.get("name"): item
    for item in mobile_camera.get("mediaCaptures", [])
}
camera_capture = captures.get("camera") or {}

if evidence.get("passed") is not False:
    raise SystemExit("bad camera policy smoke should fail")
if mobile_camera.get("passed") is not False:
    raise SystemExit("mobile camera check should fail")
if camera.get("supported") is not True:
    raise SystemExit("camera policy API should be supported")
if camera.get("allowed") is not False:
    raise SystemExit("camera should be denied by browser policy")
if not any("browser permission policy camera" in failure for failure in camera.get("failures", [])):
    raise SystemExit("missing browser permission policy failure")
if camera_capture.get("supported") is not True:
    raise SystemExit("camera media capture API should be supported")
if camera_capture.get("success") is not False:
    raise SystemExit("camera media capture should fail under camera=()")
if not any("media capture camera failed" in failure for failure in camera_capture.get("failures", [])):
    raise SystemExit("missing camera media capture failure")
PY

echo "[public-web-auth-browser-smoke-contract] all assertions passed"
