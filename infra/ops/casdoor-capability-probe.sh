#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
ENV_FILE="${ENV_FILE:-${REPO_ROOT}/.env}"
CURL_TIMEOUT_SECONDS="${CASDOOR_PROBE_CURL_TIMEOUT_SECONDS:-10}"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

usage() {
  cat <<'USAGE'
Usage: infra/ops/casdoor-capability-probe.sh

Runs the IAM v2 Casdoor capability probe against a real Casdoor test instance.

Required env:
  CASDOOR_ISSUER
  CASDOOR_CLIENT_ID
  CASDOOR_REDIRECT_URI

Optional env:
  ENV_FILE                                  source this env file first when present
  CASDOOR_CLIENT_SECRET                     required only for refresh rotation probe
  CASDOOR_PROBE_ID_TOKEN                    JWT to inspect for amr/auth_time/acr
  CASDOOR_PROBE_ID_TOKEN_FILE               file containing a JWT to inspect
  CASDOOR_PROBE_REFRESH_TOKEN               refresh token consumed by rotation probe
  CASDOOR_PROBE_RUN_REFRESH_ROTATION=true   enable the destructive refresh probe

The refresh probe consumes the provided refresh token. It is disabled unless
CASDOOR_PROBE_RUN_REFRESH_ROTATION=true is set explicitly.
USAGE
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
  exit 0
fi

log() {
  echo "[casdoor-capability-probe] $*"
}

warn() {
  echo "[casdoor-capability-probe][warn] $*" >&2
}

fail() {
  echo "[casdoor-capability-probe][error] $*" >&2
  exit 1
}

not_configured() {
  echo "[casdoor-capability-probe][not-configured] $*" >&2
  exit 78
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "missing required command: $1"
}

load_probe_env() {
  if [[ -f "${ENV_FILE}" ]]; then
    set -a
    source_env_file "${ENV_FILE}"
    set +a
  else
    warn "env file not found: ${ENV_FILE}; relying on process environment"
  fi
}

require_base_config() {
  local missing=()
  for key in CASDOOR_ISSUER CASDOOR_CLIENT_ID CASDOOR_REDIRECT_URI; do
    if [[ -z "${!key:-}" ]]; then
      missing+=("${key}")
    fi
  done
  if (( ${#missing[@]} > 0 )); then
    not_configured "missing required env: ${missing[*]}"
  fi
}

json_field() {
  python3 - "$1" "$2" <<'PY'
import json
import sys

field = sys.argv[1]
payload = sys.argv[2]
data = json.loads(payload)
value = data.get(field, "")
if not isinstance(value, str):
    value = ""
print(value)
PY
}

build_step_up_url() {
  python3 - "$1" "${CASDOOR_CLIENT_ID}" "${CASDOOR_REDIRECT_URI}" <<'PY'
import sys
import base64
import hashlib
import urllib.parse

endpoint, client_id, redirect_uri = sys.argv[1:4]
code_verifier = "stuhelper-capability-probe-verifier-20260502"
code_challenge = base64.urlsafe_b64encode(
    hashlib.sha256(code_verifier.encode()).digest()
).decode().rstrip("=")
params = {
    "response_type": "code",
    "client_id": client_id,
    "redirect_uri": redirect_uri,
    "scope": "openid offline_access",
    "state": "stuhelper-capability-probe",
    "code_challenge": code_challenge,
    "code_challenge_method": "S256",
    "prompt": "login",
    "max_age": "0",
    "acr_values": "mfa",
}
separator = "&" if "?" in endpoint else "?"
print(endpoint + separator + urllib.parse.urlencode(params))
PY
}

load_probe_id_token() {
  if [[ -n "${CASDOOR_PROBE_ID_TOKEN:-}" ]]; then
    printf '%s' "${CASDOOR_PROBE_ID_TOKEN}"
    return 0
  fi
  if [[ -n "${CASDOOR_PROBE_ID_TOKEN_FILE:-}" ]]; then
    [[ -f "${CASDOOR_PROBE_ID_TOKEN_FILE}" ]] || fail "missing CASDOOR_PROBE_ID_TOKEN_FILE"
    tr -d '\n\r ' <"${CASDOOR_PROBE_ID_TOKEN_FILE}"
  fi
}

inspect_id_token_claims() {
  local token="$1"
  if [[ -z "${token}" ]]; then
    log "C2/C3/C6 JWT claim probe: not run; provide CASDOOR_PROBE_ID_TOKEN or CASDOOR_PROBE_ID_TOKEN_FILE"
    return 0
  fi

  python3 - "${token}" <<'PY'
import base64
import json
import sys

token = sys.argv[1]
parts = token.split(".")
if len(parts) < 2:
    raise SystemExit("invalid JWT: expected at least two segments")
payload = parts[1] + "=" * (-len(parts[1]) % 4)
claims = json.loads(base64.urlsafe_b64decode(payload))
print("[casdoor-capability-probe] C2 amr claim present: " + str("amr" in claims).lower())
print("[casdoor-capability-probe] C3 auth_time claim present: " + str("auth_time" in claims).lower())
print("[casdoor-capability-probe] C6 acr claim present: " + str("acr" in claims).lower())
PY
}

curl_form() {
  local url="$1"
  local refresh_token="$2"
  curl -sS --max-time "${CURL_TIMEOUT_SECONDS}" \
    -w '\n%{http_code}' \
    -X POST "${url}" \
    -H 'Content-Type: application/x-www-form-urlencoded' \
    --data-urlencode 'grant_type=refresh_token' \
    --data-urlencode "refresh_token=${refresh_token}" \
    --data-urlencode "client_id=${CASDOOR_CLIENT_ID}" \
    --data-urlencode "client_secret=${CASDOOR_CLIENT_SECRET}"
}

parse_refresh_token() {
  python3 - "$1" <<'PY'
import json
import sys

try:
    data = json.loads(sys.argv[1])
except json.JSONDecodeError:
    print("")
else:
    token = data.get("refresh_token", "")
    print(token if isinstance(token, str) else "")
PY
}

run_refresh_rotation_probe() {
  local token_endpoint="$1"
  if [[ "${CASDOOR_PROBE_RUN_REFRESH_ROTATION:-false}" != "true" ]]; then
    log "Refresh rotation probe: not run; set CASDOOR_PROBE_RUN_REFRESH_ROTATION=true to consume a probe refresh token"
    return 0
  fi
  [[ -n "${CASDOOR_CLIENT_SECRET:-}" ]] || not_configured "CASDOOR_CLIENT_SECRET is required for refresh rotation probe"
  [[ -n "${CASDOOR_PROBE_REFRESH_TOKEN:-}" ]] || not_configured "CASDOOR_PROBE_REFRESH_TOKEN is required for refresh rotation probe"

  log "Refresh rotation probe: consuming provided probe refresh token"
  local first raw_body first_status second second_status rotated
  first="$(curl_form "${token_endpoint}" "${CASDOOR_PROBE_REFRESH_TOKEN}")"
  first_status="${first##*$'\n'}"
  raw_body="${first%$'\n'*}"
  rotated="$(parse_refresh_token "${raw_body}")"
  log "Refresh rotation first exchange status: ${first_status}; returned new refresh token: $([[ -n "${rotated}" ]] && echo true || echo false)"

  second="$(curl_form "${token_endpoint}" "${CASDOOR_PROBE_REFRESH_TOKEN}")"
  second_status="${second##*$'\n'}"
  log "Refresh rotation reused original token status: ${second_status}"
  if [[ "${first_status}" == "200" && -n "${rotated}" && "${second_status}" =~ ^(400|401|403)$ ]]; then
    log "Refresh rotation verdict: provider appears single-use for this probe token"
  elif [[ "${second_status}" == "200" ]]; then
    fail "refresh token reuse was accepted; provider rotation/reuse detection is not sufficient"
  else
    fail "refresh rotation probe inconclusive; inspect Casdoor token endpoint logs"
  fi
}

main() {
  require_cmd curl
  require_cmd python3
  load_probe_env
  require_base_config

  local metadata_url metadata authorization_endpoint token_endpoint step_up_url id_token
  id_token=""
  metadata_url="${CASDOOR_ISSUER%/}/.well-known/openid-configuration"
  log "Fetching OIDC metadata: ${metadata_url}"
  metadata="$(curl -fsS --max-time "${CURL_TIMEOUT_SECONDS}" "${metadata_url}")"
  authorization_endpoint="$(json_field authorization_endpoint "${metadata}")"
  token_endpoint="$(json_field token_endpoint "${metadata}")"
  [[ -n "${authorization_endpoint}" ]] || fail "OIDC metadata missing authorization_endpoint"
  [[ -n "${token_endpoint}" ]] || fail "OIDC metadata missing token_endpoint"

  step_up_url="$(build_step_up_url "${authorization_endpoint}")"
  log "C4/C5/C6 step-up authorize URL contains prompt=login, max_age=0, acr_values=mfa:"
  printf '%s\n' "${step_up_url}"
  if [[ -n "${CASDOOR_PROBE_ID_TOKEN:-}" || -n "${CASDOOR_PROBE_ID_TOKEN_FILE:-}" ]]; then
    id_token="$(load_probe_id_token)"
  fi
  inspect_id_token_claims "${id_token}"
  log "C1 role/user-level MFA enforcement remains a manual Casdoor admin-console observation"
  run_refresh_rotation_probe "${token_endpoint}"
}

main
