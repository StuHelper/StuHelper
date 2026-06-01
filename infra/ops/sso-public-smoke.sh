#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

usage() {
  cat <<'USAGE'
Usage: infra/ops/sso-public-smoke.sh

Verifies the public production SSO/Casdoor ingress:

  - SSO_PUBLIC_BASE_URL /.well-known/openid-configuration returns OIDC metadata
  - the discovery issuer matches SSO_PUBLIC_SMOKE_EXPECTED_ISSUER
  - the discovered JWKS endpoint returns a JSON Web Key Set
  - the discovered authorization endpoint is routed to SSO and not a static 404/5xx
  - the StuHelper web Casdoor application still enables password login and signup controls

Required production env:
  CASDOOR_ISSUER must be https://sso.stuhelper.com

Optional env:
  SSO_PUBLIC_BASE_URL                         defaults to CASDOOR_PUBLIC_AUTH_BASE_URL,
                                              WEB_VITE_SSO_URL, CASDOOR_ISSUER, or
                                              https://sso.stuhelper.com
  SSO_PUBLIC_SMOKE_EXPECTED_ISSUER            defaults to CASDOOR_ISSUER or SSO_PUBLIC_BASE_URL
  SSO_PUBLIC_SMOKE_CLIENT_ID                  defaults to CASDOOR_CLIENT_ID
  SSO_PUBLIC_SMOKE_APPLICATION_ID             defaults to admin/stuhelper-web
  SSO_PUBLIC_SMOKE_REDIRECT_URI               defaults to CASDOOR_REDIRECT_URI
  SSO_PUBLIC_SMOKE_SCOPE                      defaults to openid
  SSO_PUBLIC_SMOKE_RETRIES                    defaults to 3
  SSO_PUBLIC_SMOKE_SLEEP_SECONDS              defaults to 2
  SSO_PUBLIC_SMOKE_EVIDENCE_FILE              defaults to infra/generated/sso-public-smoke-evidence.json
                                              set to "-" to only print the JSON bundle
  SSO_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS        defaults to false; set true only for local contract tests or
                                              intentional local validation
  SSO_PUBLIC_SMOKE_CURL_INSECURE              defaults to false; set true only for local self-signed TLS
  SSO_PUBLIC_SMOKE_CURL_NO_PROXY              defaults to "*"; set empty to honor proxy env vars
  SSO_PUBLIC_SMOKE_RESOLVE_IP                 optional diagnostic override for sso.stuhelper.com
USAGE
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
  exit 0
fi

require_cmd curl
require_cmd jq
require_cmd python3

preserved_sso_public_base_url="${SSO_PUBLIC_BASE_URL-__STUHELPER_UNSET__}"
preserved_expected_issuer="${SSO_PUBLIC_SMOKE_EXPECTED_ISSUER-__STUHELPER_UNSET__}"
preserved_client_id="${SSO_PUBLIC_SMOKE_CLIENT_ID-__STUHELPER_UNSET__}"
preserved_application_id="${SSO_PUBLIC_SMOKE_APPLICATION_ID-__STUHELPER_UNSET__}"
preserved_redirect_uri="${SSO_PUBLIC_SMOKE_REDIRECT_URI-__STUHELPER_UNSET__}"
preserved_scope="${SSO_PUBLIC_SMOKE_SCOPE-__STUHELPER_UNSET__}"
preserved_retries="${SSO_PUBLIC_SMOKE_RETRIES-__STUHELPER_UNSET__}"
preserved_sleep_seconds="${SSO_PUBLIC_SMOKE_SLEEP_SECONDS-__STUHELPER_UNSET__}"
preserved_evidence_file="${SSO_PUBLIC_SMOKE_EVIDENCE_FILE-__STUHELPER_UNSET__}"
preserved_allow_local_targets="${SSO_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS-__STUHELPER_UNSET__}"
preserved_curl_insecure="${SSO_PUBLIC_SMOKE_CURL_INSECURE-__STUHELPER_UNSET__}"
preserved_curl_no_proxy="${SSO_PUBLIC_SMOKE_CURL_NO_PROXY-__STUHELPER_UNSET__}"
preserved_resolve_ip="${SSO_PUBLIC_SMOKE_RESOLVE_IP-__STUHELPER_UNSET__}"

load_env

if [[ "${preserved_sso_public_base_url}" != "__STUHELPER_UNSET__" ]]; then SSO_PUBLIC_BASE_URL="${preserved_sso_public_base_url}"; fi
if [[ "${preserved_expected_issuer}" != "__STUHELPER_UNSET__" ]]; then SSO_PUBLIC_SMOKE_EXPECTED_ISSUER="${preserved_expected_issuer}"; fi
if [[ "${preserved_client_id}" != "__STUHELPER_UNSET__" ]]; then SSO_PUBLIC_SMOKE_CLIENT_ID="${preserved_client_id}"; fi
if [[ "${preserved_application_id}" != "__STUHELPER_UNSET__" ]]; then SSO_PUBLIC_SMOKE_APPLICATION_ID="${preserved_application_id}"; fi
if [[ "${preserved_redirect_uri}" != "__STUHELPER_UNSET__" ]]; then SSO_PUBLIC_SMOKE_REDIRECT_URI="${preserved_redirect_uri}"; fi
if [[ "${preserved_scope}" != "__STUHELPER_UNSET__" ]]; then SSO_PUBLIC_SMOKE_SCOPE="${preserved_scope}"; fi
if [[ "${preserved_retries}" != "__STUHELPER_UNSET__" ]]; then SSO_PUBLIC_SMOKE_RETRIES="${preserved_retries}"; fi
if [[ "${preserved_sleep_seconds}" != "__STUHELPER_UNSET__" ]]; then SSO_PUBLIC_SMOKE_SLEEP_SECONDS="${preserved_sleep_seconds}"; fi
if [[ "${preserved_evidence_file}" != "__STUHELPER_UNSET__" ]]; then SSO_PUBLIC_SMOKE_EVIDENCE_FILE="${preserved_evidence_file}"; fi
if [[ "${preserved_allow_local_targets}" != "__STUHELPER_UNSET__" ]]; then SSO_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS="${preserved_allow_local_targets}"; fi
if [[ "${preserved_curl_insecure}" != "__STUHELPER_UNSET__" ]]; then SSO_PUBLIC_SMOKE_CURL_INSECURE="${preserved_curl_insecure}"; fi
if [[ "${preserved_curl_no_proxy}" != "__STUHELPER_UNSET__" ]]; then SSO_PUBLIC_SMOKE_CURL_NO_PROXY="${preserved_curl_no_proxy}"; fi
if [[ "${preserved_resolve_ip}" != "__STUHELPER_UNSET__" ]]; then SSO_PUBLIC_SMOKE_RESOLVE_IP="${preserved_resolve_ip}"; fi

curl() {
  local args=()
  if [[ "${SSO_PUBLIC_SMOKE_CURL_INSECURE:-false}" == "true" ]]; then
    args+=(--insecure)
  fi
  if [[ -n "${SSO_PUBLIC_SMOKE_CURL_NO_PROXY:-*}" ]]; then
    args+=(--noproxy "${SSO_PUBLIC_SMOKE_CURL_NO_PROXY:-*}")
  fi
  if [[ -n "${SSO_PUBLIC_SMOKE_RESOLVE_IP:-}" ]]; then
    args+=(--resolve "sso.stuhelper.com:443:${SSO_PUBLIC_SMOKE_RESOLVE_IP}")
  fi
  command curl "${args[@]}" "$@"
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

reject_local_smoke_target() {
  local name="$1"
  local value="$2"
  case "${value}" in
    *localhost*|*127.0.0.1*|*::1*|*host.docker.internal*)
      die "${name} points to a local target (${value}); sso-public-smoke verifies public production SSO ingress. Set SSO_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS=true only for local contract tests or intentional local validation."
      ;;
  esac
}

non_public_ip_error() {
  local value="$1"
  [[ -n "${value}" ]] || return 0
  python3 - "${value}" <<'PY'
import ipaddress
import sys

raw = sys.argv[1].strip()
candidate = raw.strip("[]").split("%", 1)[0]
try:
    ip = ipaddress.ip_address(candidate)
except ValueError:
    raise SystemExit(0)

if not ip.is_global:
    print(f"{raw} is not a public/global IP address")
    raise SystemExit(1)
PY
}

non_public_smoke_ip_message() {
  local name="$1"
  local value="$2"
  local error
  [[ "${allow_local_targets:-false}" != "true" ]] || return 1
  [[ -n "${value}" ]] || return 1
  if ! error="$(non_public_ip_error "${value}")"; then
    printf '%s resolved to a non-public target (%s); sso-public-smoke verifies public production SSO ingress. Set SSO_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS=true only for local contract tests or intentional local validation.' \
      "${name}" "${error:-${value}}"
    return 0
  fi
  return 1
}

urlencode() {
  python3 - "$1" <<'PY'
from urllib.parse import quote
import sys

print(quote(sys.argv[1], safe=""))
PY
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
  if ! meta="$(curl -sS --max-time 10 -D "${headers_file}" -w $'%{http_code}\n%{url_effective}\n%{ssl_verify_result}\n%{remote_ip}' -o "${response_file}" "${url}" 2>"${error_file}")"; then
    :
  fi
  printf '%s\n%s\n%s\n%s\n' \
    "$(printf '%s\n' "${meta:-}" | sed -n '1p')" \
    "$(printf '%s\n' "${meta:-}" | sed -n '2p')" \
    "$(printf '%s\n' "${meta:-}" | sed -n '3p')" \
    "$(printf '%s\n' "${meta:-}" | sed -n '4p')"
}

sso_public_base_url="$(trim_trailing_slash "${SSO_PUBLIC_BASE_URL:-${CASDOOR_PUBLIC_AUTH_BASE_URL:-${WEB_VITE_SSO_URL:-${CASDOOR_ISSUER:-https://sso.stuhelper.com}}}}")"
expected_issuer="$(trim_trailing_slash "${SSO_PUBLIC_SMOKE_EXPECTED_ISSUER:-${CASDOOR_ISSUER:-${sso_public_base_url}}}")"
client_id="${SSO_PUBLIC_SMOKE_CLIENT_ID:-${CASDOOR_CLIENT_ID:-}}"
application_id="${SSO_PUBLIC_SMOKE_APPLICATION_ID:-admin/stuhelper-web}"
redirect_uri="${SSO_PUBLIC_SMOKE_REDIRECT_URI:-${CASDOOR_REDIRECT_URI:-}}"
scope="${SSO_PUBLIC_SMOKE_SCOPE:-openid}"
retries="${SSO_PUBLIC_SMOKE_RETRIES:-3}"
sleep_seconds="${SSO_PUBLIC_SMOKE_SLEEP_SECONDS:-2}"
evidence_file="${SSO_PUBLIC_SMOKE_EVIDENCE_FILE:-${REPO_ROOT}/infra/generated/sso-public-smoke-evidence.json}"
allow_local_targets="$(normalize_bool "SSO_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS" "${SSO_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS:-false}")"
resolve_ip="${SSO_PUBLIC_SMOKE_RESOLVE_IP:-}"
discovery_url="${sso_public_base_url}/.well-known/openid-configuration"

[[ -n "${sso_public_base_url}" ]] || die "SSO_PUBLIC_BASE_URL or CASDOOR_PUBLIC_AUTH_BASE_URL is required"
[[ -n "${expected_issuer}" ]] || die "SSO_PUBLIC_SMOKE_EXPECTED_ISSUER or CASDOOR_ISSUER is required"
if [[ ! "${retries}" =~ ^[0-9]+$ ]] || (( retries < 1 )); then
  die "SSO_PUBLIC_SMOKE_RETRIES must be a positive integer"
fi
if [[ ! "${sleep_seconds}" =~ ^[0-9]+$ ]]; then
  die "SSO_PUBLIC_SMOKE_SLEEP_SECONDS must be a non-negative integer"
fi

if [[ "${allow_local_targets}" != "true" ]]; then
  [[ "${sso_public_base_url}" == "https://sso.stuhelper.com" ]] || \
    die "SSO_PUBLIC_BASE_URL must be exactly https://sso.stuhelper.com for production SSO public smoke"
  [[ "${expected_issuer}" == "https://sso.stuhelper.com" ]] || \
    die "SSO_PUBLIC_SMOKE_EXPECTED_ISSUER must be exactly https://sso.stuhelper.com for production SSO public smoke"
  reject_local_smoke_target "SSO_PUBLIC_BASE_URL" "${sso_public_base_url}"
  reject_local_smoke_target "SSO_PUBLIC_SMOKE_EXPECTED_ISSUER" "${expected_issuer}"
  if non_public_message="$(non_public_smoke_ip_message "SSO_PUBLIC_SMOKE_RESOLVE_IP" "${resolve_ip}")"; then
    die "${non_public_message}"
  fi
fi

pass=0
fail=0
check_jsonl=""
jwks_url=""
authorization_endpoint=""
token_endpoint=""
userinfo_endpoint=""

record_check() {
  local passed="$1"
  local name="$2"
  local details="${3-}"
  [[ -n "${details}" ]] || details="{}"
  [[ -n "${check_jsonl}" ]] || return 0
  jq -cn \
    --arg name "${name}" \
    --arg passed "${passed}" \
    --arg details "${details}" \
    '($details | fromjson? // {}) as $parsedDetails
    | {
        name: $name,
        passed: ($passed == "true")
      }
      + (
        if ($parsedDetails | type) == "object" and ($parsedDetails | length) > 0
        then {details: $parsedDetails}
        else {}
        end
      )' >>"${check_jsonl}"
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

check_discovery() {
  local attempt response_file headers_file error_file meta status remote_ip remote_ip_error content_type bytes curl_error snippet
  local actual_issuer actual_authorization_endpoint actual_jwks_uri actual_token_endpoint
  response_file="$(mktemp)"
  headers_file="$(mktemp)"
  error_file="$(mktemp)"
  for ((attempt = 1; attempt <= retries; attempt++)); do
    : >"${response_file}"
    : >"${headers_file}"
    : >"${error_file}"
    meta="$(request_url "${discovery_url}" "${response_file}" "${headers_file}" "${error_file}")"
    status="$(printf '%s\n' "${meta}" | sed -n '1p')"
    remote_ip="$(printf '%s\n' "${meta}" | sed -n '4p')"
    content_type="$(response_header "${headers_file}" "Content-Type")"
    bytes="$(wc -c <"${response_file}" | tr -d '[:space:]')"
    curl_error="$(body_snippet "${error_file}")"
    if remote_ip_error="$(non_public_smoke_ip_message "SSO discovery remote IP" "${remote_ip}")"; then
      rm -f "${response_file}" "${headers_file}" "${error_file}"
      record_fail "SSO discovery metadata resolved to non-public target" "$(json_detail url "${discovery_url}" remoteIP "${remote_ip}" resolveIP "${resolve_ip}" reason "${remote_ip_error}" curlError "${curl_error}")"
      return
    fi
    actual_issuer="$(jq -r '.issuer // ""' "${response_file}" 2>/dev/null || true)"
    actual_authorization_endpoint="$(jq -r '.authorization_endpoint // ""' "${response_file}" 2>/dev/null || true)"
    actual_jwks_uri="$(jq -r '.jwks_uri // ""' "${response_file}" 2>/dev/null || true)"
    actual_token_endpoint="$(jq -r '.token_endpoint // ""' "${response_file}" 2>/dev/null || true)"
    if [[ "${status}" == "200" ]] && jq -e --arg issuer "${expected_issuer}" '
      type == "object"
      and .issuer == $issuer
      and (.authorization_endpoint | type == "string" and length > 0)
      and (.token_endpoint | type == "string" and length > 0)
      and (.jwks_uri | type == "string" and length > 0)
    ' "${response_file}" >/dev/null; then
      authorization_endpoint="$(jq -r '.authorization_endpoint' "${response_file}")"
      token_endpoint="$(jq -r '.token_endpoint' "${response_file}")"
      userinfo_endpoint="$(jq -r '.userinfo_endpoint // empty' "${response_file}")"
      jwks_url="$(jq -r '.jwks_uri' "${response_file}")"
      rm -f "${response_file}" "${headers_file}" "${error_file}"
      record_pass "SSO discovery metadata" "$(json_detail url "${discovery_url}" httpStatus "${status}" remoteIP "${remote_ip}" issuer "${expected_issuer}" bytes "${bytes}" contentType "${content_type}" curlError "${curl_error}")"
      return
    fi
    if (( attempt < retries )); then
      sleep "${sleep_seconds}"
    fi
  done
  snippet="$(body_snippet "${response_file}")"
  rm -f "${response_file}" "${headers_file}" "${error_file}"
  record_fail "SSO discovery metadata expected issuer ${expected_issuer}, got ${actual_issuer:-<missing>} with HTTP ${status:-000}" "$(json_detail url "${discovery_url}" expectedIssuer "${expected_issuer}" actualIssuer "${actual_issuer:-}" actualAuthorizationEndpoint "${actual_authorization_endpoint:-}" actualJWKSURI "${actual_jwks_uri:-}" actualTokenEndpoint "${actual_token_endpoint:-}" httpStatus "${status:-000}" remoteIP "${remote_ip:-}" contentType "${content_type:-}" bodySnippet "${snippet}" curlError "${curl_error:-}")"
}

check_jwks() {
  local attempt response_file headers_file error_file meta status remote_ip remote_ip_error content_type bytes curl_error snippet
  response_file="$(mktemp)"
  headers_file="$(mktemp)"
  error_file="$(mktemp)"
  for ((attempt = 1; attempt <= retries; attempt++)); do
    : >"${response_file}"
    : >"${headers_file}"
    : >"${error_file}"
    meta="$(request_url "${jwks_url}" "${response_file}" "${headers_file}" "${error_file}")"
    status="$(printf '%s\n' "${meta}" | sed -n '1p')"
    remote_ip="$(printf '%s\n' "${meta}" | sed -n '4p')"
    content_type="$(response_header "${headers_file}" "Content-Type")"
    bytes="$(wc -c <"${response_file}" | tr -d '[:space:]')"
    curl_error="$(body_snippet "${error_file}")"
    if remote_ip_error="$(non_public_smoke_ip_message "SSO JWKS remote IP" "${remote_ip}")"; then
      rm -f "${response_file}" "${headers_file}" "${error_file}"
      record_fail "SSO JWKS resolved to non-public target" "$(json_detail url "${jwks_url}" remoteIP "${remote_ip}" resolveIP "${resolve_ip}" reason "${remote_ip_error}" curlError "${curl_error}")"
      return
    fi
    if [[ "${status}" == "200" ]] && jq -e 'type == "object" and (.keys | type == "array")' "${response_file}" >/dev/null; then
      rm -f "${response_file}" "${headers_file}" "${error_file}"
      record_pass "SSO JWKS" "$(json_detail url "${jwks_url}" httpStatus "${status}" remoteIP "${remote_ip}" bytes "${bytes}" contentType "${content_type}" curlError "${curl_error}")"
      return
    fi
    if (( attempt < retries )); then
      sleep "${sleep_seconds}"
    fi
  done
  snippet="$(body_snippet "${response_file}")"
  rm -f "${response_file}" "${headers_file}" "${error_file}"
  record_fail "SSO JWKS expected JSON Web Key Set, got HTTP ${status:-000}" "$(json_detail url "${jwks_url}" httpStatus "${status:-000}" remoteIP "${remote_ip:-}" contentType "${content_type:-}" bodySnippet "${snippet}" curlError "${curl_error:-}")"
}

check_authorize_route() {
  local id redirect encoded_scope state authorize_url response_file headers_file error_file meta status remote_ip remote_ip_error location content_type curl_error snippet
  id="${client_id:-sso-public-smoke}"
  redirect="${redirect_uri:-https://stuhelper.com/api/v1/auth/callback}"
  encoded_scope="$(urlencode "${scope}")"
  state="sso-public-smoke"
  authorize_url="${authorization_endpoint}?response_type=code&client_id=$(urlencode "${id}")&redirect_uri=$(urlencode "${redirect}")&scope=${encoded_scope}&state=${state}"
  response_file="$(mktemp)"
  headers_file="$(mktemp)"
  error_file="$(mktemp)"
  meta="$(request_url "${authorize_url}" "${response_file}" "${headers_file}" "${error_file}")"
  status="$(printf '%s\n' "${meta}" | sed -n '1p')"
  remote_ip="$(printf '%s\n' "${meta}" | sed -n '4p')"
  location="$(response_header "${headers_file}" "Location")"
  content_type="$(response_header "${headers_file}" "Content-Type")"
  curl_error="$(body_snippet "${error_file}")"
  if remote_ip_error="$(non_public_smoke_ip_message "SSO authorize remote IP" "${remote_ip}")"; then
    rm -f "${response_file}" "${headers_file}" "${error_file}"
    record_fail "SSO authorize route resolved to non-public target" "$(json_detail url "${authorization_endpoint}" remoteIP "${remote_ip}" resolveIP "${resolve_ip}" reason "${remote_ip_error}" clientID "${id}" curlError "${curl_error}")"
    return
  fi
  if [[ "${status}" =~ ^[0-9][0-9][0-9]$ ]] && (( status >= 200 && status < 500 )) && [[ "${status}" != "404" ]]; then
    rm -f "${response_file}" "${headers_file}" "${error_file}"
    record_pass "SSO authorize route reachable" "$(json_detail url "${authorization_endpoint}" httpStatus "${status}" remoteIP "${remote_ip}" location "${location}" contentType "${content_type}" clientID "${id}" curlError "${curl_error}")"
    return
  fi
  snippet="$(body_snippet "${response_file}")"
  rm -f "${response_file}" "${headers_file}" "${error_file}"
  record_fail "SSO authorize route returned ${status:-000}" "$(json_detail url "${authorization_endpoint}" httpStatus "${status:-000}" remoteIP "${remote_ip:-}" location "${location}" contentType "${content_type}" bodySnippet "${snippet}" curlError "${curl_error}")"
}

check_application_config() {
  local response_file headers_file error_file meta status remote_ip remote_ip_error content_type bytes curl_error summary
  local application_url
  application_url="${sso_public_base_url}/api/get-application?id=$(urlencode "${application_id}")"
  response_file="$(mktemp)"
  headers_file="$(mktemp)"
  error_file="$(mktemp)"
  meta="$(request_url "${application_url}" "${response_file}" "${headers_file}" "${error_file}")"
  status="$(printf '%s\n' "${meta}" | sed -n '1p')"
  remote_ip="$(printf '%s\n' "${meta}" | sed -n '4p')"
  content_type="$(response_header "${headers_file}" "Content-Type")"
  bytes="$(wc -c <"${response_file}" | tr -d '[:space:]')"
  curl_error="$(body_snippet "${error_file}")"
  if remote_ip_error="$(non_public_smoke_ip_message "SSO application config remote IP" "${remote_ip}")"; then
    rm -f "${response_file}" "${headers_file}" "${error_file}"
    record_fail "SSO web application config resolved to non-public target" "$(json_detail url "${application_url}" remoteIP "${remote_ip}" resolveIP "${resolve_ip}" reason "${remote_ip_error}" applicationID "${application_id}" curlError "${curl_error}")"
    return
  fi
  if [[ "${status}" == "200" ]] && jq -e --arg app_id "${application_id}" '
    def method_rule($name):
      (.data.signinMethods // [])
      | map(select(.name == $name))
      | first
      | .rule // "";
    def signup_item_required($name):
      (.data.signupItems // [])
      | map(select(.name == $name))
      | first
      | .required // false;
    type == "object"
    and .status == "ok"
    and (.data | type == "object")
    and (.data.owner + "/" + .data.name) == $app_id
    and .data.organization == "stuhelper"
    and .data.enableSignUp == true
    and method_rule("Password") == "All"
    and method_rule("Face ID") == "None"
    and signup_item_required("Password") == true
    and signup_item_required("Confirm password") == true
  ' "${response_file}" >/dev/null; then
    rm -f "${response_file}" "${headers_file}" "${error_file}"
    record_pass "SSO web application exposes password signup controls" "$(json_detail url "${application_url}" httpStatus "${status}" remoteIP "${remote_ip}" applicationID "${application_id}" bytes "${bytes}" contentType "${content_type}" curlError "${curl_error}")"
    return
  fi

  summary="$(jq -c --arg url "${application_url}" --arg http_status "${status:-000}" --arg remote_ip "${remote_ip:-}" --arg content_type "${content_type:-}" --arg bytes "${bytes:-0}" --arg curl_error "${curl_error:-}" '
    {
      url: $url,
      httpStatus: $http_status,
      remoteIP: $remote_ip,
      contentType: $content_type,
      bytes: $bytes,
      curlError: $curl_error,
      status: (.status // ""),
      msg: (.msg // ""),
      data: (
        .data // {}
        | {
            owner,
            name,
            organization,
            enablePassword,
            enableSignUp,
            enableSigninSession,
            signinMethods: ((.signinMethods // []) | map({name, displayName, rule})),
            signupItems: ((.signupItems // []) | map({name, visible, required, rule}))
          }
      )
    }
  ' "${response_file}" 2>/dev/null || true)"
  rm -f "${response_file}" "${headers_file}" "${error_file}"
  if [[ -z "${summary}" ]]; then
    summary="$(json_detail url "${application_url}" httpStatus "${status:-000}" remoteIP "${remote_ip:-}" contentType "${content_type:-}" bytes "${bytes:-0}" curlError "${curl_error:-}")"
  fi
  record_fail "SSO web application password/signup controls drift" "${summary}"
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
      "${sso_public_base_url}" \
      "${expected_issuer}" \
      "${discovery_url}" \
      "${jwks_url}" \
      "${authorization_endpoint}" \
      "${token_endpoint}" \
      "${userinfo_endpoint}" \
      "${resolve_ip}" \
      "${pass}" \
      "${fail}" \
      "${passed}" \
      "${check_jsonl}" <<'PY'
import json
import sys
from pathlib import Path

checks_path = Path(sys.argv[14])
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
    "passed": sys.argv[13] == "true",
    "ssoPublicBaseURL": sys.argv[3],
    "expectedIssuer": sys.argv[4],
    "endpoints": {
        "discovery": sys.argv[5],
        "jwks": sys.argv[6],
        "authorization": sys.argv[7],
        "token": sys.argv[8],
        "userinfo": sys.argv[9],
    },
    "resolveIP": sys.argv[10],
    "summary": {
        "passed": int(sys.argv[11]),
        "failed": int(sys.argv[12]),
    },
    "checks": checks,
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
    log "wrote public SSO smoke evidence to ${evidence_file}" >&2
  else
    printf '%s\n' "${bundle}" | jq .
  fi
}

printf '%s\n' '--- Public SSO Smoke ---'
tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT
check_jsonl="${tmpdir}/checks.jsonl"
: >"${check_jsonl}"

check_discovery
if [[ -n "${jwks_url}" ]]; then
  check_jwks
fi
if [[ -n "${authorization_endpoint}" ]]; then
  check_authorize_route
fi
if [[ -n "${application_id}" ]]; then
  check_application_config
fi

printf '%s\n' '------------------------'
printf 'Result: %s passed, %s failed\n' "${pass}" "${fail}"

smoke_passed="false"
if (( fail == 0 )); then
  smoke_passed="true"
fi
write_evidence "${smoke_passed}"

if (( fail > 0 )); then
  die "public SSO smoke failed"
fi

log "public SSO smoke passed"
