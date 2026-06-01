#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
SMOKE_SCRIPT="${REPO_ROOT}/infra/ops/admission-public-smoke.sh"
PROD_DEPLOY_SCRIPT="${REPO_ROOT}/infra/ops/prod-deploy.sh"
PROD_PARITY_SMOKE_SCRIPT="${REPO_ROOT}/infra/ops/prod-parity-smoke.sh"
PROD_ENV_EXAMPLE="${REPO_ROOT}/.env.prod.example"

fail() {
  echo "[admission-public-smoke-contract][error] $*" >&2
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
  python3 - "${file}" <<'PY' || fail "admission public smoke evidence assertion failed"
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as fh:
    evidence = json.load(fh)

checks = evidence.get("checks", [])

def require(condition, message):
    if not condition:
        raise SystemExit(message)

require(evidence.get("passed") is True, "smoke did not pass")
require(evidence.get("summary", {}).get("passed") == 10, "passed count")
require(evidence.get("summary", {}).get("failed") == 0, "failed count")
require(len(checks) == 10, "check count")
for name in (
    "Admission join verify token serves Web SPA",
    "Admission join verify token allows camera capture",
    "Admission join metrics vitals beacon returns 204",
    "Admission join metrics frontend error beacon returns 204",
    "Admission join camera handoff SSE ingress returns 401 with buffering disabled",
    "Admission join bare verify returns 404",
    "Web host verify token returns 404",
    "Web host bare verify returns 404",
    "Identity host verify token returns 404",
    "Identity host bare verify returns 404",
):
    require(len([item for item in checks if item.get("name") == name and item.get("passed") is True]) == 1, name)

endpoints = evidence.get("endpoints", {})
require(endpoints.get("admissionVerify", "").endswith("/join/verify/__stuhelper_public_smoke__?qq=10000"), "admissionVerify endpoint")
require(endpoints.get("admissionBareVerify", "").endswith("/join/verify"), "admissionBareVerify endpoint")
require(endpoints.get("webVerify", "").endswith("/web/verify/__stuhelper_public_smoke__?qq=10000"), "webVerify endpoint")
require(endpoints.get("webBareVerify", "").endswith("/web/verify"), "webBareVerify endpoint")
require(endpoints.get("identityVerify", "").endswith("/id/verify/__stuhelper_public_smoke__?qq=10000"), "identityVerify endpoint")
require(endpoints.get("identityBareVerify", "").endswith("/id/verify"), "identityBareVerify endpoint")
require(endpoints.get("admissionMetricsVitals", "").endswith("/join/api/v1/metrics/vitals"), "admissionMetricsVitals endpoint")
require(endpoints.get("admissionMetricsFrontendErrors", "").endswith("/join/api/v1/metrics/frontend-errors"), "admissionMetricsFrontendErrors endpoint")
require(endpoints.get("admissionCameraHandoffEvents", "").endswith("/join/api/v1/admission/freshman/camera-handoffs/__stuhelper_public_smoke__/events"), "admissionCameraHandoffEvents endpoint")
require(evidence.get("probe", {}).get("qq") == "10000", "probe qq")
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
[[ -x "${SMOKE_SCRIPT}" ]] || fail "admission public smoke script must be executable"

assert_contains "${SMOKE_SCRIPT}" 'ADMISSION_PUBLIC_BASE_URL must be exactly https://join\.stuhelper\.com'
assert_contains "${SMOKE_SCRIPT}" '/verify/\$\{probe_token\}\?qq=\$\{probe_qq\}'
assert_contains "${SMOKE_SCRIPT}" '/api/v1/metrics/vitals'
assert_contains "${SMOKE_SCRIPT}" '/api/v1/metrics/frontend-errors'
assert_contains "${SMOKE_SCRIPT}" '/api/v1/admission/freshman/camera-handoffs/'
assert_contains "${SMOKE_SCRIPT}" 'Admission join verify token allows camera capture'
assert_contains "${SMOKE_SCRIPT}" 'Permissions-Policy'
assert_contains "${SMOKE_SCRIPT}" 'camera=\(self\)'
assert_contains "${SMOKE_SCRIPT}" 'Accept: text/event-stream'
assert_contains "${SMOKE_SCRIPT}" 'X-Accel-Buffering'
assert_contains "${SMOKE_SCRIPT}" 'Admission join camera handoff SSE ingress returns 401 with buffering disabled'
assert_contains "${SMOKE_SCRIPT}" 'Admission join metrics vitals beacon returns 204'
assert_contains "${SMOKE_SCRIPT}" 'Admission join metrics frontend error beacon returns 204'
assert_contains "${SMOKE_SCRIPT}" 'Origin: \$\{origin\}'
assert_contains "${SMOKE_SCRIPT}" 'Referer: \$\{referer\}'
assert_contains "${SMOKE_SCRIPT}" 'Web host verify token returns 404'
assert_contains "${SMOKE_SCRIPT}" 'WEB_PUBLIC_URL /verify'
assert_contains "${SMOKE_SCRIPT}" 'id\.stuhelper\.com /verify'
assert_contains "${SMOKE_SCRIPT}" 'Identity host verify token returns 404'
assert_contains "${SMOKE_SCRIPT}" 'Identity host bare verify returns 404'
assert_contains "${SMOKE_SCRIPT}" 'Admission join bare verify returns 404'
assert_contains "${SMOKE_SCRIPT}" 'ADMISSION_PUBLIC_SMOKE_CURL_NO_PROXY'
assert_contains "${SMOKE_SCRIPT}" 'ADMISSION_PUBLIC_SMOKE_RESOLVE_IP'
assert_contains "${SMOKE_SCRIPT}" 'join\.stuhelper\.com:443'
assert_contains "${SMOKE_SCRIPT}" 'id\.stuhelper\.com:443'
assert_contains "${PROD_DEPLOY_SCRIPT}" 'ADMISSION_PUBLIC_SMOKE_ENABLED'
assert_contains "${PROD_DEPLOY_SCRIPT}" 'admission-public-smoke\.sh'
assert_contains "${PROD_PARITY_SMOKE_SCRIPT}" 'ADMISSION_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS=true'
assert_contains "${PROD_PARITY_SMOKE_SCRIPT}" 'ADMISSION_PUBLIC_SMOKE_CURL_INSECURE=true'
assert_contains "${PROD_PARITY_SMOKE_SCRIPT}" 'admission-public-smoke\.sh'
assert_contains "${PROD_ENV_EXAMPLE}" '^ADMISSION_PUBLIC_SMOKE_ENABLED=true$'
assert_contains "${PROD_ENV_EXAMPLE}" '^ADMISSION_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS=false$'
assert_contains "${PROD_ENV_EXAMPLE}" '^ADMISSION_PUBLIC_SMOKE_CURL_NO_PROXY=\*$'
assert_contains "${PROD_ENV_EXAMPLE}" '^ADMISSION_PUBLIC_SMOKE_DISABLED_ID_URL=https://id\.stuhelper\.com$'
assert_contains "${PROD_ENV_EXAMPLE}" '^ADMISSION_PUBLIC_SMOKE_PROBE_TOKEN=__stuhelper_public_smoke__$'
assert_contains "${PROD_ENV_EXAMPLE}" '^ADMISSION_PUBLIC_SMOKE_PROBE_QQ=10000$'

tmpdir="$(mktemp -d)"
env_file="${tmpdir}/.env"
generated_env_file="${tmpdir}/.env.generated"
generated_secret_env_file="${tmpdir}/.env.generated.secrets"
generated_obs_dir="${tmpdir}/obs"
evidence_file="${tmpdir}/admission-public-smoke-evidence.json"
bad_sse_evidence_file="${tmpdir}/admission-public-smoke-bad-sse-evidence.json"
port_file="${tmpdir}/port"
sse_buffering_file="${tmpdir}/sse-buffering-header"
touch "${generated_env_file}" "${generated_secret_env_file}"
mkdir -p "${generated_obs_dir}"
printf '%s\n' 'no' >"${sse_buffering_file}"

cat >"${tmpdir}/fake-admission-server.py" <<'PY'
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from urllib.parse import parse_qs, urlparse
import sys

port_file = Path(sys.argv[1])
sse_buffering_file = Path(sys.argv[2])

def sse_buffering_header():
    try:
        return sse_buffering_file.read_text(encoding="utf-8").strip() or "no"
    except FileNotFoundError:
        return "no"

class Handler(BaseHTTPRequestHandler):
    def log_message(self, format, *args):
        return

    def write_text(self, status, body, content_type="text/plain"):
        encoded = body.encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", content_type)
        self.send_header("Content-Length", str(len(encoded)))
        self.end_headers()
        self.wfile.write(encoded)

    def do_GET(self):
        parsed = urlparse(self.path)
        query = parse_qs(parsed.query)
        path = parsed.path
        if path == "/join/verify/__stuhelper_public_smoke__" and query.get("qq") == ["10000"]:
            encoded = b'<!doctype html><title>StuHelper</title><div id="app"></div>'
            self.send_response(200)
            self.send_header("Content-Type", "text/html")
            self.send_header("Permissions-Policy", "camera=(self), microphone=(), geolocation=(), payment=()")
            self.send_header("Content-Length", str(len(encoded)))
            self.end_headers()
            self.wfile.write(encoded)
            return
        if path == "/join/api/v1/admission/freshman/camera-handoffs/__stuhelper_public_smoke__/events":
            encoded = b'{"success":false,"error":{"code":"A0010100","message":"login required"}}'
            self.send_response(401)
            self.send_header("Content-Type", "application/json")
            self.send_header("X-Accel-Buffering", sse_buffering_header())
            self.send_header("Content-Length", str(len(encoded)))
            self.end_headers()
            self.wfile.write(encoded)
            return
        if path in {
            "/join/verify",
            "/web/verify",
            "/web/verify/__stuhelper_public_smoke__",
            "/id/verify",
            "/id/verify/__stuhelper_public_smoke__",
        }:
            self.write_text(404, "not found")
            return
        self.write_text(500, f"unexpected path: {self.path}")

    def do_POST(self):
        parsed = urlparse(self.path)
        path = parsed.path
        expected_origin = f"http://127.0.0.1:{self.server.server_port}"
        origin = self.headers.get("Origin", "")
        referer = self.headers.get("Referer", "")
        if path in {
            "/join/api/v1/metrics/vitals",
            "/join/api/v1/metrics/frontend-errors",
        }:
            if origin != expected_origin:
                self.write_text(403, f"unexpected origin: {origin}")
                return
            if not referer.startswith(expected_origin + "/join/verify/__stuhelper_public_smoke__?qq=10000"):
                self.write_text(403, f"unexpected referer: {referer}")
                return
            self.send_response(204)
            self.send_header("Content-Length", "0")
            self.end_headers()
            return
        self.write_text(500, f"unexpected POST path: {self.path}")

server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
port_file.write_text(str(server.server_port))
server.serve_forever()
PY

python3 "${tmpdir}/fake-admission-server.py" "${port_file}" "${sse_buffering_file}" &
server_pid=$!
for _ in {1..50}; do
  [[ -s "${port_file}" ]] && break
  sleep 0.1
done
[[ -s "${port_file}" ]] || fail "fake admission server did not start"

port="$(cat "${port_file}")"
base_url="http://127.0.0.1:${port}"
cat >"${env_file}" <<ENV
APP_ENV=production
ADMISSION_PUBLIC_BASE_URL=${base_url}/join
WEB_PUBLIC_URL=${base_url}/web
ADMISSION_PUBLIC_SMOKE_DISABLED_ID_URL=${base_url}/id
ENV

local_refused_output="$(
  ENV_FILE="${env_file}" \
  GENERATED_ENV_FILE="${generated_env_file}" \
  GENERATED_SECRET_ENV_FILE="${generated_secret_env_file}" \
  GENERATED_OBS_DIR="${generated_obs_dir}" \
  ADMISSION_PUBLIC_BASE_URL=https://join.stuhelper.com \
  ADMISSION_PUBLIC_SMOKE_EVIDENCE_FILE="${evidence_file}" \
  ADMISSION_PUBLIC_SMOKE_RETRIES=1 \
  "${SMOKE_SCRIPT}" 2>&1
)" && fail "smoke unexpectedly allowed local targets without an explicit override"

printf '%s\n' "${local_refused_output}" | grep -Eq 'ADMISSION_PUBLIC_SMOKE_DISABLED_ID_URL must be exactly https://id\.stuhelper\.com|admission-public-smoke verifies public production ingress' || \
  fail "local target refusal did not explain the public-ingress guard"

output="$(
  ENV_FILE="${env_file}" \
  GENERATED_ENV_FILE="${generated_env_file}" \
  GENERATED_SECRET_ENV_FILE="${generated_secret_env_file}" \
  GENERATED_OBS_DIR="${generated_obs_dir}" \
  ADMISSION_PUBLIC_SMOKE_EVIDENCE_FILE="${evidence_file}" \
  ADMISSION_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS=true \
  ADMISSION_PUBLIC_SMOKE_RETRIES=1 \
  "${SMOKE_SCRIPT}"
)"

printf '%s\n' "${output}" | grep -q 'public admission smoke passed' || fail "smoke did not pass against fake admission server"
assert_evidence "${evidence_file}"

printf '%s\n' 'yes' >"${sse_buffering_file}"
bad_sse_output="$(
  ENV_FILE="${env_file}" \
  GENERATED_ENV_FILE="${generated_env_file}" \
  GENERATED_SECRET_ENV_FILE="${generated_secret_env_file}" \
  GENERATED_OBS_DIR="${generated_obs_dir}" \
  ADMISSION_PUBLIC_SMOKE_EVIDENCE_FILE="${bad_sse_evidence_file}" \
  ADMISSION_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS=true \
  ADMISSION_PUBLIC_SMOKE_RETRIES=1 \
  "${SMOKE_SCRIPT}" 2>&1
)" && fail "smoke unexpectedly passed when SSE ingress buffering was enabled"

printf '%s\n' "${bad_sse_output}" | grep -q 'Admission join camera handoff SSE ingress returns 401 with buffering disabled' || \
  fail "bad SSE buffering run did not report the camera handoff SSE check"

jq -e '
  .passed == false
  and .summary.failed == 1
  and ([.checks[] | select(.name | test("Admission join camera handoff SSE ingress returns 401 with buffering disabled")) | select(.passed == false)] | length == 1)
' "${bad_sse_evidence_file}" >/dev/null || fail "bad SSE buffering evidence did not fail the expected check"

echo "[admission-public-smoke-contract] all assertions passed"
