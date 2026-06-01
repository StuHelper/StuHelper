#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

usage() {
  cat <<'USAGE'
Usage: infra/ops/admission-public-smoke.sh

Verifies the public production admission ingress:

  - ADMISSION_PUBLIC_BASE_URL /verify/<token>?qq=<qq> serves the Web SPA
  - ADMISSION_PUBLIC_BASE_URL /verify/<token>?qq=<qq> allows camera permission for
    desktop material capture
  - ADMISSION_PUBLIC_BASE_URL /api/v1/metrics/vitals accepts same-origin Web Vitals beacons
  - ADMISSION_PUBLIC_BASE_URL /api/v1/metrics/frontend-errors accepts same-origin frontend error beacons
  - ADMISSION_PUBLIC_BASE_URL /api/v1/admission/freshman/camera-handoffs/<id>/events reaches the backend
    as an SSE endpoint with buffering disabled
  - ADMISSION_PUBLIC_BASE_URL /verify returns 404
  - WEB_PUBLIC_URL /verify and /verify/<token>?qq=<qq> return 404
  - id.stuhelper.com /verify and /verify/<token>?qq=<qq> return 404

Required production env:
  ADMISSION_PUBLIC_BASE_URL must be https://join.stuhelper.com

Optional env:
  WEB_PUBLIC_URL                              defaults to https://stuhelper.com
  ADMISSION_PUBLIC_SMOKE_DISABLED_ID_URL     defaults to https://id.stuhelper.com
  ADMISSION_PUBLIC_SMOKE_RETRIES             defaults to 3
  ADMISSION_PUBLIC_SMOKE_SLEEP_SECONDS       defaults to 2
  ADMISSION_PUBLIC_SMOKE_EVIDENCE_FILE       defaults to infra/generated/admission-public-smoke-evidence.json
                                             set to "-" to only print the JSON bundle
  ADMISSION_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS defaults to false; set true only for local contract tests or
                                             intentional local validation
  ADMISSION_PUBLIC_SMOKE_CURL_INSECURE       defaults to false; set true only for local self-signed TLS
  ADMISSION_PUBLIC_SMOKE_CURL_NO_PROXY       defaults to "*"; set empty to honor proxy env vars
  ADMISSION_PUBLIC_SMOKE_RESOLVE_IP          optional diagnostic override for stuhelper.com/join/id
  ADMISSION_PUBLIC_SMOKE_PROBE_TOKEN         defaults to __stuhelper_public_smoke__
  ADMISSION_PUBLIC_SMOKE_PROBE_QQ            defaults to 10000
USAGE
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
  exit 0
fi

require_cmd curl
require_cmd jq
require_cmd python3

preserved_admission_public_base_url="${ADMISSION_PUBLIC_BASE_URL-__STUHELPER_UNSET__}"
preserved_web_public_url="${WEB_PUBLIC_URL-__STUHELPER_UNSET__}"
preserved_disabled_id_url="${ADMISSION_PUBLIC_SMOKE_DISABLED_ID_URL-__STUHELPER_UNSET__}"
preserved_retries="${ADMISSION_PUBLIC_SMOKE_RETRIES-__STUHELPER_UNSET__}"
preserved_sleep_seconds="${ADMISSION_PUBLIC_SMOKE_SLEEP_SECONDS-__STUHELPER_UNSET__}"
preserved_evidence_file="${ADMISSION_PUBLIC_SMOKE_EVIDENCE_FILE-__STUHELPER_UNSET__}"
preserved_allow_local_targets="${ADMISSION_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS-__STUHELPER_UNSET__}"
preserved_curl_insecure="${ADMISSION_PUBLIC_SMOKE_CURL_INSECURE-__STUHELPER_UNSET__}"
preserved_curl_no_proxy="${ADMISSION_PUBLIC_SMOKE_CURL_NO_PROXY-__STUHELPER_UNSET__}"
preserved_resolve_ip="${ADMISSION_PUBLIC_SMOKE_RESOLVE_IP-__STUHELPER_UNSET__}"
preserved_probe_token="${ADMISSION_PUBLIC_SMOKE_PROBE_TOKEN-__STUHELPER_UNSET__}"
preserved_probe_qq="${ADMISSION_PUBLIC_SMOKE_PROBE_QQ-__STUHELPER_UNSET__}"

load_env

if [[ "${preserved_admission_public_base_url}" != "__STUHELPER_UNSET__" ]]; then ADMISSION_PUBLIC_BASE_URL="${preserved_admission_public_base_url}"; fi
if [[ "${preserved_web_public_url}" != "__STUHELPER_UNSET__" ]]; then WEB_PUBLIC_URL="${preserved_web_public_url}"; fi
if [[ "${preserved_disabled_id_url}" != "__STUHELPER_UNSET__" ]]; then ADMISSION_PUBLIC_SMOKE_DISABLED_ID_URL="${preserved_disabled_id_url}"; fi
if [[ "${preserved_retries}" != "__STUHELPER_UNSET__" ]]; then ADMISSION_PUBLIC_SMOKE_RETRIES="${preserved_retries}"; fi
if [[ "${preserved_sleep_seconds}" != "__STUHELPER_UNSET__" ]]; then ADMISSION_PUBLIC_SMOKE_SLEEP_SECONDS="${preserved_sleep_seconds}"; fi
if [[ "${preserved_evidence_file}" != "__STUHELPER_UNSET__" ]]; then ADMISSION_PUBLIC_SMOKE_EVIDENCE_FILE="${preserved_evidence_file}"; fi
if [[ "${preserved_allow_local_targets}" != "__STUHELPER_UNSET__" ]]; then ADMISSION_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS="${preserved_allow_local_targets}"; fi
if [[ "${preserved_curl_insecure}" != "__STUHELPER_UNSET__" ]]; then ADMISSION_PUBLIC_SMOKE_CURL_INSECURE="${preserved_curl_insecure}"; fi
if [[ "${preserved_curl_no_proxy}" != "__STUHELPER_UNSET__" ]]; then ADMISSION_PUBLIC_SMOKE_CURL_NO_PROXY="${preserved_curl_no_proxy}"; fi
if [[ "${preserved_resolve_ip}" != "__STUHELPER_UNSET__" ]]; then ADMISSION_PUBLIC_SMOKE_RESOLVE_IP="${preserved_resolve_ip}"; fi
if [[ "${preserved_probe_token}" != "__STUHELPER_UNSET__" ]]; then ADMISSION_PUBLIC_SMOKE_PROBE_TOKEN="${preserved_probe_token}"; fi
if [[ "${preserved_probe_qq}" != "__STUHELPER_UNSET__" ]]; then ADMISSION_PUBLIC_SMOKE_PROBE_QQ="${preserved_probe_qq}"; fi

curl() {
  local args=()
  if [[ "${ADMISSION_PUBLIC_SMOKE_CURL_INSECURE:-false}" == "true" ]]; then
    args+=(--insecure)
  fi
  if [[ -n "${ADMISSION_PUBLIC_SMOKE_CURL_NO_PROXY:-*}" ]]; then
    args+=(--noproxy "${ADMISSION_PUBLIC_SMOKE_CURL_NO_PROXY:-*}")
  fi
  if [[ -n "${ADMISSION_PUBLIC_SMOKE_RESOLVE_IP:-}" ]]; then
    args+=(
      --resolve "stuhelper.com:443:${ADMISSION_PUBLIC_SMOKE_RESOLVE_IP}"
      --resolve "join.stuhelper.com:443:${ADMISSION_PUBLIC_SMOKE_RESOLVE_IP}"
      --resolve "id.stuhelper.com:443:${ADMISSION_PUBLIC_SMOKE_RESOLVE_IP}"
    )
  fi
  command curl "${args[@]}" "$@"
}

trim_trailing_slash() {
  local value="$1"
  printf '%s\n' "${value%/}"
}

url_origin() {
  python3 - "$1" <<'PY'
from urllib.parse import urlparse
import sys

parsed = urlparse(sys.argv[1])
if not parsed.scheme or not parsed.netloc:
    raise SystemExit(1)
print(f"{parsed.scheme}://{parsed.netloc}")
PY
}

reject_local_smoke_target() {
  local name="$1"
  local value="$2"
  case "${value}" in
    *localhost*|*127.0.0.1*|*::1*|*host.docker.internal*)
      die "${name} points to a local target (${value}); admission-public-smoke verifies public production ingress. Set ADMISSION_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS=true only for local contract tests or intentional local validation."
      ;;
  esac
}

normalize_bool() {
  local name="$1"
  local value="$2"
  case "${value}" in
    true | TRUE | 1 | yes | YES) printf 'true\n' ;;
    false | FALSE | 0 | no | NO | "") printf 'false\n' ;;
    *) die "${name} must be true or false" ;;
  esac
}

admission_public_base_url="$(trim_trailing_slash "${ADMISSION_PUBLIC_BASE_URL:-https://join.stuhelper.com}")"
web_public_url="$(trim_trailing_slash "${WEB_PUBLIC_URL:-https://stuhelper.com}")"
disabled_id_url="$(trim_trailing_slash "${ADMISSION_PUBLIC_SMOKE_DISABLED_ID_URL:-https://id.stuhelper.com}")"
retries="${ADMISSION_PUBLIC_SMOKE_RETRIES:-3}"
sleep_seconds="${ADMISSION_PUBLIC_SMOKE_SLEEP_SECONDS:-2}"
evidence_file="${ADMISSION_PUBLIC_SMOKE_EVIDENCE_FILE:-${REPO_ROOT}/infra/generated/admission-public-smoke-evidence.json}"
allow_local_targets="$(normalize_bool "ADMISSION_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS" "${ADMISSION_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS:-false}")"
resolve_ip="${ADMISSION_PUBLIC_SMOKE_RESOLVE_IP:-}"
probe_token="${ADMISSION_PUBLIC_SMOKE_PROBE_TOKEN:-__stuhelper_public_smoke__}"
probe_qq="${ADMISSION_PUBLIC_SMOKE_PROBE_QQ:-10000}"

[[ -n "${admission_public_base_url}" ]] || die "ADMISSION_PUBLIC_BASE_URL is required"
[[ -n "${web_public_url}" ]] || die "WEB_PUBLIC_URL is required"
[[ -n "${disabled_id_url}" ]] || die "ADMISSION_PUBLIC_SMOKE_DISABLED_ID_URL is required"

case "${probe_token}" in
  ""|*/*|*\?*|*#*|*"&"*|*=*)
    die "ADMISSION_PUBLIC_SMOKE_PROBE_TOKEN must be a non-empty single URL path segment"
    ;;
esac
case "${probe_qq}" in
  ""|*/*|*\?*|*#*|*"&"*|*=*)
    die "ADMISSION_PUBLIC_SMOKE_PROBE_QQ must not be empty or contain URL separators"
    ;;
esac
if [[ ! "${retries}" =~ ^[0-9]+$ ]] || (( retries < 1 )); then
  die "ADMISSION_PUBLIC_SMOKE_RETRIES must be a positive integer"
fi
if [[ ! "${sleep_seconds}" =~ ^[0-9]+$ ]]; then
  die "ADMISSION_PUBLIC_SMOKE_SLEEP_SECONDS must be a non-negative integer"
fi

if [[ "${allow_local_targets}" != "true" ]]; then
  [[ "${admission_public_base_url}" == "https://join.stuhelper.com" ]] || \
    die "ADMISSION_PUBLIC_BASE_URL must be exactly https://join.stuhelper.com for production admission public smoke"
  [[ "${disabled_id_url}" == "https://id.stuhelper.com" ]] || \
    die "ADMISSION_PUBLIC_SMOKE_DISABLED_ID_URL must be exactly https://id.stuhelper.com for production admission public smoke"
  reject_local_smoke_target "ADMISSION_PUBLIC_BASE_URL" "${admission_public_base_url}"
  reject_local_smoke_target "WEB_PUBLIC_URL" "${web_public_url}"
  reject_local_smoke_target "ADMISSION_PUBLIC_SMOKE_DISABLED_ID_URL" "${disabled_id_url}"
fi

admission_verify_url="${admission_public_base_url}/verify/${probe_token}?qq=${probe_qq}"
admission_bare_verify_url="${admission_public_base_url}/verify"
web_verify_url="${web_public_url}/verify/${probe_token}?qq=${probe_qq}"
web_bare_verify_url="${web_public_url}/verify"
disabled_id_verify_url="${disabled_id_url}/verify/${probe_token}?qq=${probe_qq}"
disabled_id_bare_verify_url="${disabled_id_url}/verify"
admission_metrics_vitals_url="${admission_public_base_url}/api/v1/metrics/vitals"
admission_metrics_frontend_errors_url="${admission_public_base_url}/api/v1/metrics/frontend-errors"
admission_camera_handoff_events_url="${admission_public_base_url}/api/v1/admission/freshman/camera-handoffs/${probe_token}/events"
admission_origin="$(url_origin "${admission_public_base_url}")"

pass=0
fail=0
check_jsonl=""

record_check() {
  local passed="$1"
  local name="$2"
  local details="${3-}"
  [[ -n "${details}" ]] || details="{}"
  [[ -n "${check_jsonl}" ]] || return 0
  python3 - "${passed}" "${name}" "${details}" >>"${check_jsonl}" <<'PY'
import json
import sys

item = {
    "name": sys.argv[2],
    "passed": sys.argv[1] == "true",
}
try:
    details = json.loads(sys.argv[3])
except Exception:
    details = {}
if isinstance(details, dict) and details:
    item["details"] = details
print(json.dumps(item, ensure_ascii=True, separators=(",", ":")))
PY
}

record_pass() {
  local details="${2-}"
  [[ -n "${details}" ]] || details="{}"
  printf '  PASS %s\n' "$1"
  pass=$((pass + 1))
  record_check "true" "$1" "${details}"
}

record_fail() {
  local details="${2-}"
  [[ -n "${details}" ]] || details="{}"
  printf '  FAIL %s\n' "$1" >&2
  fail=$((fail + 1))
  record_check "false" "$1" "${details}"
}

json_detail() {
  python3 - "$@" <<'PY'
import json
import sys

pairs = sys.argv[1:]
details = {}
for index in range(0, len(pairs) - 1, 2):
    key = pairs[index]
    value = pairs[index + 1]
    if value != "":
        details[key] = value
print(json.dumps(details, ensure_ascii=True, separators=(",", ":")))
PY
}

body_snippet() {
  python3 - "$1" <<'PY'
from pathlib import Path
import sys

path = Path(sys.argv[1])
if not path.exists():
    print("")
    raise SystemExit
raw = path.read_bytes()[:240]
text = raw.decode("utf-8", "replace").replace("\r", " ").replace("\n", " ")
print(" ".join(text.split()))
PY
}

response_header() {
  local headers_file="$1"
  local header_name="$2"
  awk -v name="${header_name}" '
    index(tolower($0), tolower(name) ":") == 1 {
      sub(/\r$/, "")
      print substr($0, index($0, ":") + 2)
      exit
    }
  ' "${headers_file}"
}

request_url() {
  local url="$1"
  local response_file="$2"
  local headers_file="$3"
  local error_file="$4"
  local meta
  local status
  local effective_url
  local ssl_verify_result

  if ! meta="$(curl -sS --max-time 10 -D "${headers_file}" -w $'%{http_code}\n%{url_effective}\n%{ssl_verify_result}' -o "${response_file}" "${url}" 2>"${error_file}")"; then
    :
  fi

  status="$(printf '%s\n' "${meta:-}" | sed -n '1p')"
  effective_url="$(printf '%s\n' "${meta:-}" | sed -n '2p')"
  ssl_verify_result="$(printf '%s\n' "${meta:-}" | sed -n '3p')"
  printf '%s\n%s\n%s\n' "${status:-000}" "${effective_url:-}" "${ssl_verify_result:-}"
}

check_http_status() {
  local name="$1"
  local url="$2"
  local expected_status="$3"
  local require_spa="${4:-false}"
  local attempt
  local response_file
  local headers_file
  local error_file
  local status
  local effective_url
  local ssl_verify_result
  local content_type
  local location
  local bytes
  local curl_error
  local snippet
  local meta

  response_file="$(mktemp)"
  headers_file="$(mktemp)"
  error_file="$(mktemp)"

  for ((attempt = 1; attempt <= retries; attempt++)); do
    : >"${response_file}"
    : >"${headers_file}"
    : >"${error_file}"
    meta="$(request_url "${url}" "${response_file}" "${headers_file}" "${error_file}")"
    status="$(printf '%s\n' "${meta}" | sed -n '1p')"
    effective_url="$(printf '%s\n' "${meta}" | sed -n '2p')"
    ssl_verify_result="$(printf '%s\n' "${meta}" | sed -n '3p')"
    content_type="$(response_header "${headers_file}" "Content-Type")"
    location="$(response_header "${headers_file}" "Location")"
    bytes="$(wc -c <"${response_file}" | tr -d '[:space:]')"
    curl_error="$(body_snippet "${error_file}")"

    if [[ "${status}" == "${expected_status}" ]]; then
      if [[ "${require_spa}" == "true" ]]; then
        if grep -Fq '<title>StuHelper</title>' "${response_file}" && grep -Fq 'id="app"' "${response_file}"; then
          rm -f "${response_file}" "${headers_file}" "${error_file}"
          record_pass "${name}" "$(json_detail url "${url}" httpStatus "${status}" attempts "${attempt}" bytes "${bytes}" contentType "${content_type}" sslVerifyResult "${ssl_verify_result}" curlError "${curl_error}")"
          return
        fi
      else
        rm -f "${response_file}" "${headers_file}" "${error_file}"
        record_pass "${name}" "$(json_detail url "${url}" httpStatus "${status}" attempts "${attempt}" location "${location}" contentType "${content_type}" sslVerifyResult "${ssl_verify_result}" curlError "${curl_error}")"
        return
      fi
    fi

    if (( attempt < retries )); then
      sleep "${sleep_seconds}"
    fi
  done

  snippet="$(body_snippet "${response_file}")"
  rm -f "${response_file}" "${headers_file}" "${error_file}"
  record_fail "${name} expected ${expected_status}, got ${status:-000}" "$(json_detail url "${url}" expectedStatus "${expected_status}" httpStatus "${status:-000}" attempts "${retries}" location "${location:-}" contentType "${content_type:-}" sslVerifyResult "${ssl_verify_result:-}" bodySnippet "${snippet}" curlError "${curl_error:-}")"
}

check_post_json_status() {
  local name="$1"
  local url="$2"
  local payload="$3"
  local expected_status="$4"
  local origin="$5"
  local referer="$6"
  local attempt
  local response_file
  local headers_file
  local error_file
  local status
  local effective_url
  local ssl_verify_result
  local content_type
  local bytes
  local curl_error
  local snippet
  local meta

  response_file="$(mktemp)"
  headers_file="$(mktemp)"
  error_file="$(mktemp)"

  for ((attempt = 1; attempt <= retries; attempt++)); do
    : >"${response_file}"
    : >"${headers_file}"
    : >"${error_file}"
    if ! meta="$(
      curl \
        -sS \
        --max-time 10 \
        -D "${headers_file}" \
        -w $'%{http_code}\n%{url_effective}\n%{ssl_verify_result}' \
        -o "${response_file}" \
        -X POST \
        -H 'Content-Type: application/json' \
        -H "Origin: ${origin}" \
        -H "Referer: ${referer}" \
        --data "${payload}" \
        "${url}" \
        2>"${error_file}"
    )"; then
      :
    fi
    status="$(printf '%s\n' "${meta:-}" | sed -n '1p')"
    effective_url="$(printf '%s\n' "${meta:-}" | sed -n '2p')"
    ssl_verify_result="$(printf '%s\n' "${meta:-}" | sed -n '3p')"
    content_type="$(response_header "${headers_file}" "Content-Type")"
    bytes="$(wc -c <"${response_file}" | tr -d '[:space:]')"
    curl_error="$(body_snippet "${error_file}")"

    if [[ "${status}" == "${expected_status}" ]]; then
      rm -f "${response_file}" "${headers_file}" "${error_file}"
      record_pass "${name}" "$(json_detail url "${url}" httpStatus "${status}" attempts "${attempt}" bytes "${bytes}" contentType "${content_type}" sslVerifyResult "${ssl_verify_result}" origin "${origin}" referer "${referer}" curlError "${curl_error}")"
      return
    fi

    if (( attempt < retries )); then
      sleep "${sleep_seconds}"
    fi
  done

  snippet="$(body_snippet "${response_file}")"
  rm -f "${response_file}" "${headers_file}" "${error_file}"
  record_fail "${name} expected ${expected_status}, got ${status:-000}" "$(json_detail url "${url}" expectedStatus "${expected_status}" httpStatus "${status:-000}" attempts "${retries}" contentType "${content_type:-}" sslVerifyResult "${ssl_verify_result:-}" bodySnippet "${snippet}" origin "${origin}" referer "${referer}" curlError "${curl_error:-}" effectiveURL "${effective_url:-}")"
}

check_get_status_header_contains() {
  local name="$1"
  local url="$2"
  local expected_status="$3"
  local header_name="$4"
  local expected_header_value="$5"
  local attempt
  local response_file
  local headers_file
  local error_file
  local status
  local effective_url
  local ssl_verify_result
  local content_type
  local actual_header
  local bytes
  local curl_error
  local snippet
  local meta

  response_file="$(mktemp)"
  headers_file="$(mktemp)"
  error_file="$(mktemp)"

  for ((attempt = 1; attempt <= retries; attempt++)); do
    : >"${response_file}"
    : >"${headers_file}"
    : >"${error_file}"
    if ! meta="$(
      curl \
        -sS \
        --max-time 10 \
        -D "${headers_file}" \
        -w $'%{http_code}\n%{url_effective}\n%{ssl_verify_result}' \
        -o "${response_file}" \
        -H 'Accept: text/event-stream' \
        -H "Origin: ${admission_origin}" \
        -H "Referer: ${admission_verify_url}" \
        "${url}" \
        2>"${error_file}"
    )"; then
      :
    fi
    status="$(printf '%s\n' "${meta:-}" | sed -n '1p')"
    effective_url="$(printf '%s\n' "${meta:-}" | sed -n '2p')"
    ssl_verify_result="$(printf '%s\n' "${meta:-}" | sed -n '3p')"
    content_type="$(response_header "${headers_file}" "Content-Type")"
    actual_header="$(response_header "${headers_file}" "${header_name}")"
    bytes="$(wc -c <"${response_file}" | tr -d '[:space:]')"
    curl_error="$(body_snippet "${error_file}")"

    if [[ "${status}" == "${expected_status}" && "${actual_header}" == *"${expected_header_value}"* ]]; then
      rm -f "${response_file}" "${headers_file}" "${error_file}"
      record_pass "${name}" "$(json_detail url "${url}" httpStatus "${status}" attempts "${attempt}" bytes "${bytes}" contentType "${content_type}" sslVerifyResult "${ssl_verify_result}" headerName "${header_name}" headerValue "${actual_header}" curlError "${curl_error}")"
      return
    fi

    if (( attempt < retries )); then
      sleep "${sleep_seconds}"
    fi
  done

  snippet="$(body_snippet "${response_file}")"
  rm -f "${response_file}" "${headers_file}" "${error_file}"
  record_fail "${name} expected ${expected_status} and ${header_name} containing ${expected_header_value}, got ${status:-000} and ${actual_header:-<missing>}" "$(json_detail url "${url}" expectedStatus "${expected_status}" httpStatus "${status:-000}" attempts "${retries}" contentType "${content_type:-}" sslVerifyResult "${ssl_verify_result:-}" bodySnippet "${snippet}" headerName "${header_name}" headerValue "${actual_header:-}" curlError "${curl_error:-}" effectiveURL "${effective_url:-}")"
}

write_evidence() {
  local passed="$1"
  local generated_at
  local bundle
  generated_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  bundle="$(
    python3 - \
      "${generated_at}" \
      "${APP_ENV:-}" \
      "${admission_public_base_url}" \
      "${web_public_url}" \
      "${disabled_id_url}" \
      "${probe_token}" \
      "${probe_qq}" \
      "${admission_verify_url}" \
      "${admission_bare_verify_url}" \
      "${web_verify_url}" \
      "${web_bare_verify_url}" \
      "${disabled_id_verify_url}" \
      "${disabled_id_bare_verify_url}" \
      "${admission_metrics_vitals_url}" \
      "${admission_metrics_frontend_errors_url}" \
      "${admission_camera_handoff_events_url}" \
      "${resolve_ip}" \
      "${pass}" \
      "${fail}" \
      "${passed}" \
      "${check_jsonl}" <<'PY'
import json
import sys
from pathlib import Path

checks_path = Path(sys.argv[21])
checks = []
if checks_path.exists():
    checks = [
        json.loads(line)
        for line in checks_path.read_text().splitlines()
        if line.strip()
    ]

bundle = {
    "generatedAt": sys.argv[1],
    "appEnv": sys.argv[2],
    "admissionPublicBaseURL": sys.argv[3],
    "webPublicURL": sys.argv[4],
    "disabledIDURL": sys.argv[5],
    "probe": {
        "token": sys.argv[6],
        "qq": sys.argv[7],
    },
    "endpoints": {
        "admissionVerify": sys.argv[8],
        "admissionBareVerify": sys.argv[9],
        "webVerify": sys.argv[10],
        "webBareVerify": sys.argv[11],
        "identityVerify": sys.argv[12],
        "identityBareVerify": sys.argv[13],
        "admissionMetricsVitals": sys.argv[14],
        "admissionMetricsFrontendErrors": sys.argv[15],
        "admissionCameraHandoffEvents": sys.argv[16],
    },
    "resolveIP": sys.argv[17],
    "summary": {
        "passed": int(sys.argv[18]),
        "failed": int(sys.argv[19]),
    },
    "checks": checks,
    "passed": sys.argv[20] == "true",
}
print(json.dumps(bundle, ensure_ascii=True, indent=2))
PY
  )"

  if [[ "${evidence_file}" != "-" ]]; then
    mkdir -p "$(dirname "${evidence_file}")"
    local tmp_file
    tmp_file="$(mktemp)"
    printf '%s\n' "${bundle}" >"${tmp_file}"
    install -m 600 "${tmp_file}" "${evidence_file}"
    rm -f "${tmp_file}"
    log "wrote public admission smoke evidence to ${evidence_file}" >&2
  else
    printf '%s\n' "${bundle}" | jq .
  fi
}

printf '%s\n' '--- Public Admission Smoke ---'
tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT
check_jsonl="${tmpdir}/checks.jsonl"
: >"${check_jsonl}"

check_http_status "Admission join verify token serves Web SPA" "${admission_verify_url}" "200" "true"
check_get_status_header_contains "Admission join verify token allows camera capture" "${admission_verify_url}" "200" "Permissions-Policy" "camera=(self)"
check_post_json_status "Admission join metrics vitals beacon returns 204" "${admission_metrics_vitals_url}" '{"name":"LCP","value":1234.5,"rating":"good"}' "204" "${admission_origin}" "${admission_verify_url}"
check_post_json_status "Admission join metrics frontend error beacon returns 204" "${admission_metrics_frontend_errors_url}" '{"kind":"error","message":"public admission smoke"}' "204" "${admission_origin}" "${admission_verify_url}"
check_get_status_header_contains "Admission join camera handoff SSE ingress returns 401 with buffering disabled" "${admission_camera_handoff_events_url}" "401" "X-Accel-Buffering" "no"
check_http_status "Admission join bare verify returns 404" "${admission_bare_verify_url}" "404"
check_http_status "Web host verify token returns 404" "${web_verify_url}" "404"
check_http_status "Web host bare verify returns 404" "${web_bare_verify_url}" "404"
check_http_status "Identity host verify token returns 404" "${disabled_id_verify_url}" "404"
check_http_status "Identity host bare verify returns 404" "${disabled_id_bare_verify_url}" "404"

printf '%s\n' '--------------------------------'
printf 'Result: %s passed, %s failed\n' "${pass}" "${fail}"

smoke_passed="false"
if (( fail == 0 )); then
  smoke_passed="true"
fi
write_evidence "${smoke_passed}"

if (( fail > 0 )); then
  die "public admission smoke failed"
fi

log "public admission smoke passed"
