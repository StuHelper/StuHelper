#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
SMOKE_SCRIPT="${REPO_ROOT}/infra/ops/sso-public-smoke.sh"
PROD_DEPLOY_SCRIPT="${REPO_ROOT}/infra/ops/prod-deploy.sh"
PROD_PARITY_SMOKE_SCRIPT="${REPO_ROOT}/infra/ops/prod-parity-smoke.sh"
PROD_ENV_EXAMPLE="${REPO_ROOT}/.env.prod.example"

fail() {
  echo "[sso-public-smoke-contract][error] $*" >&2
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
  python3 - "${file}" <<'PY' || fail "SSO public smoke evidence assertion failed"
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as fh:
    evidence = json.load(fh)

checks = evidence.get("checks", [])

def require(condition, message):
    if not condition:
        raise SystemExit(message)

require(evidence.get("passed") is True, "smoke did not pass")
require(evidence.get("summary", {}).get("passed") == 4, "passed count")
require(evidence.get("summary", {}).get("failed") == 0, "failed count")
require(len(checks) == 4, "check count")
for name in (
    "SSO discovery metadata",
    "SSO JWKS",
    "SSO authorize route reachable",
    "SSO web application exposes password signup controls",
):
    require(len([item for item in checks if item.get("name") == name and item.get("passed") is True]) == 1, name)
for item in checks:
    require(item.get("details", {}).get("remoteIP") == "127.0.0.1", f"remoteIP detail for {item.get('name')}")

issuer = evidence.get("expectedIssuer", "")
endpoints = evidence.get("endpoints", {})
require(evidence.get("ssoPublicBaseURL", "").endswith("/sso"), "sso public base")
require(issuer.endswith("/sso"), "expected issuer")
require(endpoints.get("discovery", "") == evidence.get("ssoPublicBaseURL") + "/.well-known/openid-configuration", "discovery endpoint")
require(endpoints.get("jwks", "") == issuer + "/.well-known/jwks", "jwks endpoint")
require(endpoints.get("authorization", "") == issuer + "/login/oauth/authorize", "authorization endpoint")
require(endpoints.get("token", "") == issuer + "/api/login/oauth/access_token", "token endpoint")
PY
}

for file in \
  "${SMOKE_SCRIPT}" \
  "${PROD_DEPLOY_SCRIPT}" \
  "${PROD_PARITY_SMOKE_SCRIPT}" \
  "${PROD_ENV_EXAMPLE}"; do
  [[ -f "${file}" ]] || fail "missing file: ${file}"
done

bash -n "${SMOKE_SCRIPT}"
[[ -x "${SMOKE_SCRIPT}" ]] || fail "SSO public smoke script must be executable"

assert_contains "${SMOKE_SCRIPT}" 'SSO_PUBLIC_BASE_URL must be exactly https://sso\.stuhelper\.com'
assert_contains "${SMOKE_SCRIPT}" 'SSO discovery metadata'
assert_contains "${SMOKE_SCRIPT}" 'actualIssuer'
assert_contains "${SMOKE_SCRIPT}" 'actualAuthorizationEndpoint'
assert_contains "${SMOKE_SCRIPT}" 'actualJWKSURI'
assert_contains "${SMOKE_SCRIPT}" 'SSO JWKS'
assert_contains "${SMOKE_SCRIPT}" 'SSO authorize route reachable'
assert_contains "${SMOKE_SCRIPT}" 'SSO web application exposes password signup controls'
assert_contains "${SMOKE_SCRIPT}" 'SSO_PUBLIC_SMOKE_APPLICATION_ID'
assert_contains "${SMOKE_SCRIPT}" 'SSO_PUBLIC_SMOKE_CURL_NO_PROXY'
assert_contains "${SMOKE_SCRIPT}" 'SSO_PUBLIC_SMOKE_RESOLVE_IP'
assert_contains "${SMOKE_SCRIPT}" '%\{remote_ip\}'
assert_contains "${SMOKE_SCRIPT}" 'remoteIP'
assert_contains "${SMOKE_SCRIPT}" 'not a public/global IP address'
assert_contains "${PROD_DEPLOY_SCRIPT}" 'SSO_PUBLIC_SMOKE_ENABLED'
assert_contains "${PROD_DEPLOY_SCRIPT}" 'sso-public-smoke\.sh'
assert_contains "${PROD_PARITY_SMOKE_SCRIPT}" 'SSO_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS=true'
assert_contains "${PROD_PARITY_SMOKE_SCRIPT}" 'SSO_PUBLIC_SMOKE_CURL_INSECURE=true'
assert_contains "${PROD_PARITY_SMOKE_SCRIPT}" 'sso-public-smoke\.sh'
assert_contains "${PROD_ENV_EXAMPLE}" '^SSO_PUBLIC_SMOKE_ENABLED=true$'
assert_contains "${PROD_ENV_EXAMPLE}" '^SSO_PUBLIC_BASE_URL=https://sso\.stuhelper\.com$'
assert_contains "${PROD_ENV_EXAMPLE}" '^SSO_PUBLIC_SMOKE_EXPECTED_ISSUER=https://sso\.stuhelper\.com$'
assert_contains "${PROD_ENV_EXAMPLE}" '^SSO_PUBLIC_SMOKE_APPLICATION_ID=admin/stuhelper-web$'
assert_contains "${PROD_ENV_EXAMPLE}" '^SSO_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS=false$'
assert_contains "${PROD_ENV_EXAMPLE}" '^SSO_PUBLIC_SMOKE_CURL_NO_PROXY=\*$'

tmpdir="$(mktemp -d)"
env_file="${tmpdir}/.env"
generated_env_file="${tmpdir}/.env.generated"
generated_secret_env_file="${tmpdir}/.env.generated.secrets"
generated_obs_dir="${tmpdir}/obs"
evidence_file="${tmpdir}/sso-public-smoke-evidence.json"
port_file="${tmpdir}/port"
touch "${generated_env_file}" "${generated_secret_env_file}"
mkdir -p "${generated_obs_dir}"

cat >"${tmpdir}/fake-sso-server.py" <<'PY'
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from urllib.parse import urlparse
import json
import sys

port_file = Path(sys.argv[1])

class Handler(BaseHTTPRequestHandler):
    def log_message(self, format, *args):
        return

    def write_json(self, status, payload):
        encoded = json.dumps(payload).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(encoded)))
        self.end_headers()
        self.wfile.write(encoded)

    def write_text(self, status, body):
        encoded = body.encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "text/html")
        self.send_header("Content-Length", str(len(encoded)))
        self.end_headers()
        self.wfile.write(encoded)

    def do_GET(self):
        origin = f"http://127.0.0.1:{self.server.server_port}/sso"
        parsed = urlparse(self.path)
        if parsed.path == "/sso/.well-known/openid-configuration":
            self.write_json(200, {
                "issuer": origin,
                "authorization_endpoint": origin + "/login/oauth/authorize",
                "token_endpoint": origin + "/api/login/oauth/access_token",
                "userinfo_endpoint": origin + "/api/userinfo",
                "jwks_uri": origin + "/.well-known/jwks",
            })
            return
        if parsed.path == "/sso/.well-known/jwks":
            self.write_json(200, {"keys": [{"kid": "fake-sso-key"}]})
            return
        if parsed.path == "/sso/login/oauth/authorize":
            self.write_text(200, "<!doctype html><title>Casdoor</title>")
            return
        if parsed.path == "/sso/api/get-application":
            self.write_json(200, {
                "status": "ok",
                "msg": "",
                "data": {
                    "owner": "admin",
                    "name": "stuhelper-web",
                    "organization": "stuhelper",
                    "enablePassword": False,
                    "enableSignUp": True,
                    "enableSigninSession": False,
                    "signinMethods": [
                        {"name": "Password", "displayName": "Password", "rule": "All"},
                        {"name": "Face ID", "displayName": "Face ID", "rule": "None"},
                    ],
                    "signupItems": [
                        {"name": "Password", "visible": True, "required": True, "rule": "None"},
                        {"name": "Confirm password", "visible": True, "required": True, "rule": "None"},
                    ],
                },
            })
            return
        self.write_text(404, "not found")

server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
port_file.write_text(str(server.server_port))
server.serve_forever()
PY

python3 "${tmpdir}/fake-sso-server.py" "${port_file}" &
server_pid=$!
for _ in {1..50}; do
  [[ -s "${port_file}" ]] && break
  sleep 0.1
done
[[ -s "${port_file}" ]] || fail "fake SSO server did not start"

port="$(cat "${port_file}")"
base_url="http://127.0.0.1:${port}/sso"
cat >"${env_file}" <<ENV
APP_ENV=production
CASDOOR_PUBLIC_AUTH_BASE_URL=${base_url}
CASDOOR_ISSUER=${base_url}
CASDOOR_CLIENT_ID=stuhelper-web
CASDOOR_REDIRECT_URI=http://127.0.0.1:${port}/callback
ENV

local_refused_output="$(
  ENV_FILE="${env_file}" \
  GENERATED_ENV_FILE="${generated_env_file}" \
  GENERATED_SECRET_ENV_FILE="${generated_secret_env_file}" \
  GENERATED_OBS_DIR="${generated_obs_dir}" \
  SSO_PUBLIC_SMOKE_EVIDENCE_FILE="${evidence_file}" \
  SSO_PUBLIC_SMOKE_RETRIES=1 \
  "${SMOKE_SCRIPT}" 2>&1
)" && fail "smoke unexpectedly allowed local targets without an explicit override"

printf '%s\n' "${local_refused_output}" | grep -q 'SSO_PUBLIC_BASE_URL must be exactly https://sso.stuhelper.com' || \
  fail "local target refusal did not enforce the production SSO origin guard"

prod_env_file="${tmpdir}/.env.prod"
cat >"${prod_env_file}" <<ENV
APP_ENV=production
CASDOOR_PUBLIC_AUTH_BASE_URL=https://sso.stuhelper.com
CASDOOR_ISSUER=https://sso.stuhelper.com
CASDOOR_CLIENT_ID=stuhelper-web
CASDOOR_REDIRECT_URI=https://stuhelper.com/api/v1/auth/callback
ENV

local_resolve_refused_output="$(
  ENV_FILE="${prod_env_file}" \
  GENERATED_ENV_FILE="${generated_env_file}" \
  GENERATED_SECRET_ENV_FILE="${generated_secret_env_file}" \
  GENERATED_OBS_DIR="${generated_obs_dir}" \
  SSO_PUBLIC_SMOKE_EVIDENCE_FILE="${evidence_file}" \
  SSO_PUBLIC_SMOKE_RESOLVE_IP=127.0.0.1 \
  SSO_PUBLIC_SMOKE_RETRIES=1 \
  "${SMOKE_SCRIPT}" 2>&1
)" && fail "smoke unexpectedly allowed a local SSO_PUBLIC_SMOKE_RESOLVE_IP"

printf '%s\n' "${local_resolve_refused_output}" | grep -q 'SSO_PUBLIC_SMOKE_RESOLVE_IP resolved to a non-public target' || \
  fail "local resolve override refusal did not mention the resolved target"

output="$(
  ENV_FILE="${env_file}" \
  GENERATED_ENV_FILE="${generated_env_file}" \
  GENERATED_SECRET_ENV_FILE="${generated_secret_env_file}" \
  GENERATED_OBS_DIR="${generated_obs_dir}" \
  SSO_PUBLIC_SMOKE_EVIDENCE_FILE="${evidence_file}" \
  SSO_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS=true \
  SSO_PUBLIC_SMOKE_RETRIES=1 \
  "${SMOKE_SCRIPT}"
)"

printf '%s\n' "${output}" | grep -q 'public SSO smoke passed' || fail "smoke did not pass against fake SSO server"
assert_evidence "${evidence_file}"

echo "[sso-public-smoke-contract] all assertions passed"
