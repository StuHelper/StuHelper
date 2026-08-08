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
require(evidence.get("summary", {}).get("passed") == 14, "passed count")
require(evidence.get("summary", {}).get("failed") == 0, "failed count")
require(len(checks) == 14, "check count")
for name in (
    "Admission join verify token serves Web SPA",
    "Admission join start serves Web SPA",
    "Admission join verify token keeps camera disabled",
    "Admission join manual camera handoff allows camera capture",
    "Admission join metrics vitals beacon returns 204",
    "Admission join metrics frontend error beacon returns 204",
    "Admission join retired freshman camera route returns 404",
    "Admission join retired camera handoff SSE route returns 404",
    "Admission join root returns 404",
    "Admission join main Web route returns 404",
    "Admission join bare verify returns 404",
    "Web host join start returns 404",
    "Web host verify token returns 404",
    "Web host bare verify returns 404",
):
    require(len([item for item in checks if item.get("name") == name and item.get("passed") is True]) == 1, name)

endpoints = evidence.get("endpoints", {})
require(endpoints.get("admissionVerify", "").endswith("/join/verify/__stuhelper_public_smoke__"), "admissionVerify endpoint")
require(endpoints.get("admissionStart", "").endswith("/join/start"), "admissionStart endpoint")
require(endpoints.get("admissionRoot", "").endswith("/join/"), "admissionRoot endpoint")
require(endpoints.get("admissionMainRouteProbe", "").endswith("/join/developers/apps"), "admissionMainRouteProbe endpoint")
require(endpoints.get("admissionBareVerify", "").endswith("/join/verify"), "admissionBareVerify endpoint")
require(endpoints.get("webStart", "").endswith("/web/start"), "webStart endpoint")
require(endpoints.get("webVerify", "").endswith("/web/verify/__stuhelper_public_smoke__"), "webVerify endpoint")
require(endpoints.get("webBareVerify", "").endswith("/web/verify"), "webBareVerify endpoint")
require(endpoints.get("admissionMetricsVitals", "").endswith("/join/api/v1/metrics/vitals"), "admissionMetricsVitals endpoint")
require(endpoints.get("admissionMetricsFrontendErrors", "").endswith("/join/api/v1/metrics/frontend-errors"), "admissionMetricsFrontendErrors endpoint")
require(endpoints.get("admissionManualCamera", "").endswith("/join/student-verification/manual-camera/__stuhelper_public_smoke__"), "admissionManualCamera endpoint")
require(endpoints.get("admissionRetiredCamera", "").endswith("/join/admission/freshman/camera/__stuhelper_public_smoke__"), "admissionRetiredCamera endpoint")
require(endpoints.get("admissionRetiredCameraHandoffEvents", "").endswith("/join/api/v1/admission/freshman/camera-handoffs/__stuhelper_public_smoke__/events"), "admissionRetiredCameraHandoffEvents endpoint")
require("qq" not in evidence.get("probe", {}), "probe qq must not be emitted")
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

assert_contains "${SMOKE_SCRIPT}" 'prefer_production_env_files_if_default'
assert_contains "${SMOKE_SCRIPT}" 'ADMISSION_PUBLIC_BASE_URL must be exactly https://join\.stuhelper\.com'
assert_contains "${SMOKE_SCRIPT}" '/verify/\$\{probe_token\}'
assert_contains "${SMOKE_SCRIPT}" '/start'
assert_contains "${SMOKE_SCRIPT}" '/api/v1/metrics/vitals'
assert_contains "${SMOKE_SCRIPT}" '/api/v1/metrics/frontend-errors'
assert_contains "${SMOKE_SCRIPT}" '/student-verification/manual-camera/'
assert_contains "${SMOKE_SCRIPT}" '/admission/freshman/camera/'
assert_contains "${SMOKE_SCRIPT}" 'Admission join verify token keeps camera disabled'
assert_contains "${SMOKE_SCRIPT}" 'Admission join manual camera handoff allows camera capture'
assert_contains "${SMOKE_SCRIPT}" 'Admission join start serves Web SPA'
assert_contains "${SMOKE_SCRIPT}" 'Permissions-Policy'
assert_contains "${SMOKE_SCRIPT}" 'camera=\(self\)'
assert_contains "${SMOKE_SCRIPT}" 'Admission join retired camera handoff SSE route returns 404'
assert_contains "${SMOKE_SCRIPT}" 'Admission join metrics vitals beacon returns 204'
assert_contains "${SMOKE_SCRIPT}" 'Admission join metrics frontend error beacon returns 204'
assert_contains "${SMOKE_SCRIPT}" 'Origin: \$\{origin\}'
assert_contains "${SMOKE_SCRIPT}" 'Referer: \$\{referer\}'
assert_contains "${SMOKE_SCRIPT}" 'Web host verify token returns 404'
assert_contains "${SMOKE_SCRIPT}" 'Web host join start returns 404'
assert_contains "${SMOKE_SCRIPT}" 'WEB_PUBLIC_URL /verify'
assert_contains "${SMOKE_SCRIPT}" 'Admission join bare verify returns 404'
assert_contains "${SMOKE_SCRIPT}" 'ADMISSION_PUBLIC_SMOKE_CURL_NO_PROXY'
assert_contains "${SMOKE_SCRIPT}" 'ADMISSION_PUBLIC_SMOKE_RESOLVE_IP'
assert_contains "${SMOKE_SCRIPT}" 'join\.stuhelper\.com:443'
assert_contains "${SMOKE_SCRIPT}" 'getent ahosts'
assert_contains "${SMOKE_SCRIPT}" 'resolves to loopback'
assert_contains "${SMOKE_SCRIPT}" 'ADMISSION_PUBLIC_SMOKE_RESOLVE_IP=<public-ip>'
assert_contains "${PROD_DEPLOY_SCRIPT}" 'ADMISSION_PUBLIC_SMOKE_ENABLED'
assert_contains "${PROD_DEPLOY_SCRIPT}" 'admission-public-smoke\.sh'
assert_contains "${PROD_PARITY_SMOKE_SCRIPT}" 'ADMISSION_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS=true'
assert_contains "${PROD_PARITY_SMOKE_SCRIPT}" 'ADMISSION_PUBLIC_SMOKE_CURL_INSECURE=true'
assert_contains "${PROD_PARITY_SMOKE_SCRIPT}" 'admission-public-smoke\.sh'
assert_contains "${PROD_ENV_EXAMPLE}" '^ADMISSION_PUBLIC_SMOKE_ENABLED=true$'
assert_contains "${PROD_ENV_EXAMPLE}" '^ADMISSION_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS=false$'
assert_contains "${PROD_ENV_EXAMPLE}" '^ADMISSION_PUBLIC_SMOKE_CURL_NO_PROXY=\*$'
assert_contains "${PROD_ENV_EXAMPLE}" '^ADMISSION_PUBLIC_SMOKE_PROBE_TOKEN=__stuhelper_public_smoke__$'
if grep -Eq '^ADMISSION_PUBLIC_SMOKE_PROBE_QQ=' "${PROD_ENV_EXAMPLE}"; then
  fail "ADMISSION_PUBLIC_SMOKE_PROBE_QQ must not be configured after short admission links removed qq query"
fi

tmpdir="$(mktemp -d)"
env_file="${tmpdir}/.env"
generated_env_file="${tmpdir}/.env.generated"
generated_secret_env_file="${tmpdir}/.env.generated.secrets"
generated_obs_dir="${tmpdir}/obs"
evidence_file="${tmpdir}/admission-public-smoke-evidence.json"
bad_camera_evidence_file="${tmpdir}/admission-public-smoke-bad-camera-evidence.json"
port_file="${tmpdir}/port"
camera_policy_file="${tmpdir}/manual-camera-policy"
touch "${generated_env_file}" "${generated_secret_env_file}"
mkdir -p "${generated_obs_dir}"
printf '%s\n' 'camera=(self), microphone=(), geolocation=(), payment=()' >"${camera_policy_file}"

cat >"${tmpdir}/fake-admission-server.py" <<'PY'
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from urllib.parse import urlparse
import sys

port_file = Path(sys.argv[1])
camera_policy_file = Path(sys.argv[2])

def manual_camera_policy():
    try:
        return camera_policy_file.read_text(encoding="utf-8").strip() or "camera=()"
    except FileNotFoundError:
        return "camera=()"

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
        path = parsed.path
        if path in {"/join/verify/__stuhelper_public_smoke__", "/join/start"} and parsed.query == "":
            encoded = b'<!doctype html><title>StuHelper</title><div id="app"></div>'
            self.send_response(200)
            self.send_header("Content-Type", "text/html")
            self.send_header("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")
            self.send_header("Content-Length", str(len(encoded)))
            self.end_headers()
            self.wfile.write(encoded)
            return
        if path == "/join/student-verification/manual-camera/__stuhelper_public_smoke__":
            encoded = b'<!doctype html><title>StuHelper</title><div id="app"></div>'
            self.send_response(200)
            self.send_header("Content-Type", "text/html")
            self.send_header("Permissions-Policy", manual_camera_policy())
            self.send_header("Content-Length", str(len(encoded)))
            self.end_headers()
            self.wfile.write(encoded)
            return
        if path in {
            "/join/admission/freshman/camera/__stuhelper_public_smoke__",
            "/join/api/v1/admission/freshman/camera-handoffs/__stuhelper_public_smoke__/events",
            "/join/",
            "/join/developers/apps",
            "/join/verify",
            "/web/start",
            "/web/verify",
            "/web/verify/__stuhelper_public_smoke__",
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
            if referer != expected_origin + "/join/verify/__stuhelper_public_smoke__":
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

python3 "${tmpdir}/fake-admission-server.py" "${port_file}" "${camera_policy_file}" &
server_pid=$!
for _ in {1..50}; do
  [[ -s "${port_file}" ]] && break
  sleep 0.1
done
[[ -s "${port_file}" ]] || fail "fake admission server did not start"

port="$(cat "${port_file}")"
base_url="http://127.0.0.1:${port}"
mkdir -p "${tmpdir}/bin"
cat >"${tmpdir}/bin/getent" <<'SH'
#!/usr/bin/env bash
if [[ "${1:-}" == "ahosts" ]]; then
  case "${2:-}" in
    join.stuhelper.com|stuhelper.com)
      printf '127.0.0.1 STREAM %s\n' "${2}"
      exit 0
      ;;
  esac
fi
exec /usr/bin/getent "$@"
SH
chmod +x "${tmpdir}/bin/getent"
cat >"${env_file}" <<ENV
APP_ENV=production
ADMISSION_PUBLIC_BASE_URL=${base_url}/join
WEB_PUBLIC_URL=${base_url}/web
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

printf '%s\n' "${local_refused_output}" | grep -Eq 'admission-public-smoke verifies public production ingress' || \
  fail "local target refusal did not explain the public-ingress guard"

loopback_refused_output="$(
  ENV_FILE="${env_file}" \
  GENERATED_ENV_FILE="${generated_env_file}" \
  GENERATED_SECRET_ENV_FILE="${generated_secret_env_file}" \
  GENERATED_OBS_DIR="${generated_obs_dir}" \
  ADMISSION_PUBLIC_BASE_URL=https://join.stuhelper.com \
  WEB_PUBLIC_URL=https://stuhelper.com \
  ADMISSION_PUBLIC_SMOKE_EVIDENCE_FILE="${evidence_file}" \
  ADMISSION_PUBLIC_SMOKE_RETRIES=1 \
  PATH="${tmpdir}/bin:${PATH}" \
  "${SMOKE_SCRIPT}" 2>&1
)" && fail "smoke unexpectedly allowed production hostnames that resolve to loopback"

printf '%s\n' "${loopback_refused_output}" | grep -q 'ADMISSION_PUBLIC_BASE_URL host join.stuhelper.com resolves to loopback' || \
  fail "loopback hostname refusal did not explain the local hosts override"

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

printf '%s\n' 'camera=()' >"${camera_policy_file}"
bad_camera_output="$(
  ENV_FILE="${env_file}" \
  GENERATED_ENV_FILE="${generated_env_file}" \
  GENERATED_SECRET_ENV_FILE="${generated_secret_env_file}" \
  GENERATED_OBS_DIR="${generated_obs_dir}" \
  ADMISSION_PUBLIC_SMOKE_EVIDENCE_FILE="${bad_camera_evidence_file}" \
  ADMISSION_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS=true \
  ADMISSION_PUBLIC_SMOKE_RETRIES=1 \
  "${SMOKE_SCRIPT}" 2>&1
)" && fail "smoke unexpectedly passed when manual camera permission was disabled"

printf '%s\n' "${bad_camera_output}" | grep -q 'Admission join manual camera handoff allows camera capture' || \
  fail "bad camera permission run did not report the manual camera check"

jq -e '
  .passed == false
  and .summary.failed == 1
  and ([.checks[] | select(.name | test("Admission join manual camera handoff allows camera capture")) | select(.passed == false)] | length == 1)
' "${bad_camera_evidence_file}" >/dev/null || fail "bad camera permission evidence did not fail the expected check"

echo "[admission-public-smoke-contract] all assertions passed"
