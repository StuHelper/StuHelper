#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
ENV_FILE="${ENV_FILE:-${REPO_ROOT}/.env}"
CURL_TIMEOUT_SECONDS="${CASDOOR_TOKEN_PROBE_CURL_TIMEOUT_SECONDS:-10}"
STATE="${CASDOOR_TOKEN_PROBE_STATE:-stuhelper-token-minimization-probe}"
NONCE="${CASDOOR_TOKEN_PROBE_NONCE:-stuhelper-token-minimization-probe}"
SCOPE="${CASDOOR_TOKEN_PROBE_SCOPE:-openid}"
OUTPUT="${CASDOOR_TOKEN_PROBE_OUTPUT:-text}"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

usage() {
  cat <<'USAGE'
Usage: infra/ops/casdoor-token-minimization-probe.sh

Runs a real OIDC authorization-code token minimization probe for a third-party
Casdoor application. The probe always requests scope=openid and rejects any
business claim in the returned ID token or JWT access token.

Required env:
  CASDOOR_ISSUER
  CASDOOR_TOKEN_PROBE_CLIENT_ID
  CASDOOR_TOKEN_PROBE_REDIRECT_URI

Optional env:
  ENV_FILE
  CASDOOR_TOKEN_PROBE_CLIENT_SECRET       required for confidential clients
  CASDOOR_TOKEN_PROBE_AUTH_CODE           authorization code captured from the printed URL
  CASDOOR_TOKEN_PROBE_CODE_VERIFIER       PKCE verifier; generated when absent
  CASDOOR_TOKEN_PROBE_STATE               defaults to stuhelper-token-minimization-probe
  CASDOOR_TOKEN_PROBE_NONCE               defaults to stuhelper-token-minimization-probe
  CASDOOR_TOKEN_PROBE_CURL_TIMEOUT_SECONDS
  CASDOOR_TOKEN_PROBE_OUTPUT              text or json; json emits probe evidence on stdout

When CASDOOR_TOKEN_PROBE_AUTH_CODE is absent, the script prints the authorize
URL and exits 78. After authenticating a probe user and capturing the callback
code, rerun with CASDOOR_TOKEN_PROBE_AUTH_CODE and the printed verifier.
USAGE
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
  exit 0
fi

log() {
  if [[ "${OUTPUT}" == "json" ]]; then
    echo "[casdoor-token-minimization-probe] $*" >&2
  else
    echo "[casdoor-token-minimization-probe] $*"
  fi
}

warn() {
  echo "[casdoor-token-minimization-probe][warn] $*" >&2
}

fail() {
  echo "[casdoor-token-minimization-probe][error] $*" >&2
  exit 1
}

not_configured() {
  echo "[casdoor-token-minimization-probe][not-configured] $*" >&2
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
  for key in CASDOOR_ISSUER CASDOOR_TOKEN_PROBE_CLIENT_ID CASDOOR_TOKEN_PROBE_REDIRECT_URI; do
    if [[ -z "${!key:-}" ]]; then
      missing+=("${key}")
    fi
  done
  if (( ${#missing[@]} > 0 )); then
    not_configured "missing required env: ${missing[*]}"
  fi
  if [[ "${SCOPE}" != "openid" ]]; then
    fail "CASDOOR_TOKEN_PROBE_SCOPE must be exactly openid"
  fi
  if [[ "${OUTPUT}" != "text" && "${OUTPUT}" != "json" ]]; then
    fail "CASDOOR_TOKEN_PROBE_OUTPUT must be text or json"
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

code_verifier() {
  if [[ -n "${CASDOOR_TOKEN_PROBE_CODE_VERIFIER:-}" ]]; then
    printf '%s' "${CASDOOR_TOKEN_PROBE_CODE_VERIFIER}"
    return 0
  fi
  python3 - <<'PY'
import secrets
print("stuhelper-token-probe-" + secrets.token_urlsafe(48))
PY
}

build_authorize_url() {
  python3 - "$1" "${CASDOOR_TOKEN_PROBE_CLIENT_ID}" "${CASDOOR_TOKEN_PROBE_REDIRECT_URI}" "$2" "${STATE}" "${NONCE}" <<'PY'
import base64
import hashlib
import sys
import urllib.parse

endpoint, client_id, redirect_uri, verifier, state, nonce = sys.argv[1:7]
challenge = base64.urlsafe_b64encode(hashlib.sha256(verifier.encode()).digest()).decode().rstrip("=")
params = {
    "response_type": "code",
    "client_id": client_id,
    "redirect_uri": redirect_uri,
    "scope": "openid",
    "state": state,
    "nonce": nonce,
    "code_challenge": challenge,
    "code_challenge_method": "S256",
}
separator = "&" if "?" in endpoint else "?"
print(endpoint + separator + urllib.parse.urlencode(params))
PY
}

exchange_code() {
  local token_endpoint="$1"
  local verifier="$2"
  local args=(
    -sS --max-time "${CURL_TIMEOUT_SECONDS}"
    -X POST "${token_endpoint}"
    -H 'Content-Type: application/x-www-form-urlencoded'
    --data-urlencode 'grant_type=authorization_code'
    --data-urlencode "code=${CASDOOR_TOKEN_PROBE_AUTH_CODE}"
    --data-urlencode "redirect_uri=${CASDOOR_TOKEN_PROBE_REDIRECT_URI}"
    --data-urlencode "client_id=${CASDOOR_TOKEN_PROBE_CLIENT_ID}"
    --data-urlencode "code_verifier=${verifier}"
  )
  if [[ -n "${CASDOOR_TOKEN_PROBE_CLIENT_SECRET:-}" ]]; then
    args+=(--data-urlencode "client_secret=${CASDOOR_TOKEN_PROBE_CLIENT_SECRET}")
  fi
  curl "${args[@]}"
}

inspect_token_response() {
  python3 - "$1" "${OUTPUT}" "${CASDOOR_ISSUER}" <<'PY'
import base64
import json
import sys

FORBIDDEN = {
    "phone",
    "phone_number",
    "phone_verified",
    "phone_number_verified",
    "identity_verified",
    "identity_type",
    "student_verified",
    "school",
    "school_id",
    "school_name",
    "qq",
    "qq_binding",
    "stuhelper_identity_verified",
    "stuhelper_identity_type",
    "stuhelper_student_verified",
    "stuhelper_student_school",
    "stuhelper_student_school_id",
    "stuhelper_student_school_name",
}

def canonical(key: str) -> str:
    out = []
    for i, ch in enumerate(key.strip()):
        if i > 0 and ch.isupper():
            out.append("_")
        out.append(ch.lower())
    value = "".join(out).replace("-", "_").replace(".", "_").replace(" ", "_")
    while "__" in value:
        value = value.replace("__", "_")
    return value.strip("_")

def decode_jwt(raw: str):
    parts = raw.split(".")
    if len(parts) < 2:
        return None
    padded = parts[1] + "=" * (-len(parts[1]) % 4)
    return json.loads(base64.urlsafe_b64decode(padded))

def unique_sorted(values):
    return sorted({value for value in values if value})

def log(message: str):
    stream = sys.stderr if output == "json" else sys.stdout
    print(f"[casdoor-token-minimization-probe] {message}", file=stream)

response = json.loads(sys.argv[1])
output = sys.argv[2]
issuer = sys.argv[3]
if "error" in response:
    raise SystemExit(f"token endpoint returned error: {response.get('error')}")

tokens = {
    "id_token": response.get("id_token", ""),
    "access_token": response.get("access_token", ""),
}
if not isinstance(tokens["id_token"], str) or not tokens["id_token"]:
    raise SystemExit("token response missing id_token")

violations = {}
token_claims = {}
inspected_claims = []
for label, token in tokens.items():
    if not isinstance(token, str) or not token or token.count(".") < 2:
        continue
    claims = decode_jwt(token)
    keys = sorted(canonical(key) for key in claims.keys())
    token_claims[label] = keys
    inspected_claims.extend(keys)
    bad = [key for key in keys if key in FORBIDDEN]
    log(f"{label} inspected claims: {', '.join(keys)}")
    if bad:
        violations[label] = bad

business_claims = unique_sorted(key for keys in violations.values() for key in keys)
evidence = {
    "method": "authorization_code",
    "issuer": issuer,
    "inspectedClaims": unique_sorted(inspected_claims),
    "businessClaims": business_claims,
    "tokenClaims": token_claims,
    "metadata": {
        "source": "casdoor-token-minimization-probe.sh",
    },
}
if output == "json":
    print(json.dumps(evidence, sort_keys=True, separators=(",", ":")))
    raise SystemExit(0)

if violations:
    details = "; ".join(f"{label}={','.join(keys)}" for label, keys in violations.items())
    raise SystemExit("forbidden business claims found: " + details)

log("token minimization verdict: passed")
PY
}

main() {
  require_cmd curl
  require_cmd python3
  load_probe_env
  require_base_config

  local metadata_url metadata authorization_endpoint token_endpoint verifier authorize_url
  metadata_url="${CASDOOR_ISSUER%/}/.well-known/openid-configuration"
  log "Fetching OIDC metadata: ${metadata_url}"
  metadata="$(curl -fsS --max-time "${CURL_TIMEOUT_SECONDS}" "${metadata_url}")"
  authorization_endpoint="$(json_field authorization_endpoint "${metadata}")"
  token_endpoint="$(json_field token_endpoint "${metadata}")"
  [[ -n "${authorization_endpoint}" ]] || fail "OIDC metadata missing authorization_endpoint"
  [[ -n "${token_endpoint}" ]] || fail "OIDC metadata missing token_endpoint"

  verifier="$(code_verifier)"
  authorize_url="$(build_authorize_url "${authorization_endpoint}" "${verifier}")"
  log "Authorize URL for scope=openid:"
  if [[ "${OUTPUT}" == "json" ]]; then
    printf '%s\n' "${authorize_url}" >&2
  else
    printf '%s\n' "${authorize_url}"
  fi
  log "CASDOOR_TOKEN_PROBE_CODE_VERIFIER=${verifier}"

  if [[ -z "${CASDOOR_TOKEN_PROBE_AUTH_CODE:-}" ]]; then
    not_configured "set CASDOOR_TOKEN_PROBE_AUTH_CODE after completing the authorize URL"
  fi

  local token_response
  token_response="$(exchange_code "${token_endpoint}" "${verifier}")"
  inspect_token_response "${token_response}"
}

main
