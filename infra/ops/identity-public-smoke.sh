#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

usage() {
  cat <<'USAGE'
Usage: infra/ops/identity-public-smoke.sh

Verifies the public production identity ingress:

  - WEB_PUBLIC_URL /health/ready
  - IDENTITY_ISSUER OIDC discovery, OAuth authorization server metadata, and JWKS
  - IDENTITY_ISSUER web app assets served on the identity host without falling
    through to the main site
  - IDENTITY_ISSUER /oauth2/authorize unauthenticated redirect
  - IDENTITY_ISSUER /oauth2/authorize prompt=login / max_age=0 reauth redirect
  - Optional registered-client prompt=none login_required redirect with iss
  - Optional client_credentials app-only token and resource-access API check
  - IDENTITY_ISSUER /oauth2/token, /introspect, /revoke route-level OAuth errors
  - IDENTITY_ISSUER /oauth2/logout GET/POST no-session success
  - IDENTITY_ISSUER /oauth2/logout POST query/body rejection
  - IDENTITY_ISSUER /oidc/userinfo GET/POST missing bearer failure
  - IDENTITY_ISSUER /oidc/userinfo query/body token-source rejection
  - Optional CASDOOR_ISSUER discovery and JWKS when enabled

Required env:
  IDENTITY_ISSUER

Optional env:
  WEB_PUBLIC_URL                         defaults to https://stuhelper.com
  CASDOOR_ISSUER                         required only when
                                         IDENTITY_PUBLIC_SMOKE_CASDOOR_UPSTREAM_ENABLED=true
  IDENTITY_PUBLIC_SMOKE_RETRIES          defaults to 30
  IDENTITY_PUBLIC_SMOKE_SLEEP_SECONDS    defaults to 2
  IDENTITY_PUBLIC_SMOKE_EVIDENCE_FILE    defaults to infra/generated/identity-public-smoke-evidence.json
                                        set to "-" to only print the JSON bundle
  IDENTITY_PUBLIC_SMOKE_CLIENT_ID        optional approved Identity client for richer OIDC checks
  IDENTITY_PUBLIC_SMOKE_REDIRECT_URI     required when IDENTITY_PUBLIC_SMOKE_CLIENT_ID is set
  IDENTITY_PUBLIC_SMOKE_SCOPE            defaults to "openid" for registered-client checks
  IDENTITY_PUBLIC_SMOKE_CLIENT_SECRET    optional client secret; enables client_credentials grant checks
  IDENTITY_PUBLIC_SMOKE_CLIENT_CREDENTIALS_SCOPE
                                         defaults to "resource.read" for client_credentials checks
  IDENTITY_PUBLIC_SMOKE_RESOURCE_ACCESS_RESOURCE_TYPE
                                         defaults to "resource_item"
  IDENTITY_PUBLIC_SMOKE_RESOURCE_ACCESS_RESOURCE_ID
                                         defaults to a per-run generated resource id
  IDENTITY_PUBLIC_SMOKE_RESOURCE_ACCESS_ACTION
                                         defaults to read when scope includes resource.read, otherwise write
  IDENTITY_PUBLIC_SMOKE_RESOURCE_ACCESS_EXPECT_ALLOWED
                                         defaults to false; set true only for a pre-granted smoke resource
  IDENTITY_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS
                                         defaults to false; set true only for local contract tests or
                                         intentional local production validation
  IDENTITY_PUBLIC_SMOKE_CASDOOR_UPSTREAM_ENABLED
                                         defaults to false; set true only to audit a public
                                         browser-facing Casdoor upstream
USAGE
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
  exit 0
fi

require_cmd curl
require_cmd jq
require_cmd python3

preserved_identity_issuer="${IDENTITY_ISSUER-__STUHELPER_UNSET__}"
preserved_casdoor_issuer="${CASDOOR_ISSUER-__STUHELPER_UNSET__}"
preserved_web_public_url="${WEB_PUBLIC_URL-__STUHELPER_UNSET__}"
preserved_retries="${IDENTITY_PUBLIC_SMOKE_RETRIES-__STUHELPER_UNSET__}"
preserved_sleep_seconds="${IDENTITY_PUBLIC_SMOKE_SLEEP_SECONDS-__STUHELPER_UNSET__}"
preserved_evidence_file="${IDENTITY_PUBLIC_SMOKE_EVIDENCE_FILE-__STUHELPER_UNSET__}"
preserved_smoke_client_id="${IDENTITY_PUBLIC_SMOKE_CLIENT_ID-__STUHELPER_UNSET__}"
preserved_smoke_redirect_uri="${IDENTITY_PUBLIC_SMOKE_REDIRECT_URI-__STUHELPER_UNSET__}"
preserved_smoke_scope="${IDENTITY_PUBLIC_SMOKE_SCOPE-__STUHELPER_UNSET__}"
preserved_smoke_client_secret="${IDENTITY_PUBLIC_SMOKE_CLIENT_SECRET-__STUHELPER_UNSET__}"
preserved_smoke_client_credentials_scope="${IDENTITY_PUBLIC_SMOKE_CLIENT_CREDENTIALS_SCOPE-__STUHELPER_UNSET__}"
preserved_smoke_resource_access_resource_type="${IDENTITY_PUBLIC_SMOKE_RESOURCE_ACCESS_RESOURCE_TYPE-__STUHELPER_UNSET__}"
preserved_smoke_resource_access_resource_id="${IDENTITY_PUBLIC_SMOKE_RESOURCE_ACCESS_RESOURCE_ID-__STUHELPER_UNSET__}"
preserved_smoke_resource_access_action="${IDENTITY_PUBLIC_SMOKE_RESOURCE_ACCESS_ACTION-__STUHELPER_UNSET__}"
preserved_smoke_resource_access_expect_allowed="${IDENTITY_PUBLIC_SMOKE_RESOURCE_ACCESS_EXPECT_ALLOWED-__STUHELPER_UNSET__}"
preserved_allow_local_targets="${IDENTITY_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS-__STUHELPER_UNSET__}"
preserved_casdoor_upstream_enabled="${IDENTITY_PUBLIC_SMOKE_CASDOOR_UPSTREAM_ENABLED-__STUHELPER_UNSET__}"
preserved_curl_insecure="${IDENTITY_PUBLIC_SMOKE_CURL_INSECURE-__STUHELPER_UNSET__}"

load_env

if [[ "${preserved_identity_issuer}" != "__STUHELPER_UNSET__" ]]; then IDENTITY_ISSUER="${preserved_identity_issuer}"; fi
if [[ "${preserved_casdoor_issuer}" != "__STUHELPER_UNSET__" ]]; then CASDOOR_ISSUER="${preserved_casdoor_issuer}"; fi
if [[ "${preserved_web_public_url}" != "__STUHELPER_UNSET__" ]]; then WEB_PUBLIC_URL="${preserved_web_public_url}"; fi
if [[ "${preserved_retries}" != "__STUHELPER_UNSET__" ]]; then IDENTITY_PUBLIC_SMOKE_RETRIES="${preserved_retries}"; fi
if [[ "${preserved_sleep_seconds}" != "__STUHELPER_UNSET__" ]]; then IDENTITY_PUBLIC_SMOKE_SLEEP_SECONDS="${preserved_sleep_seconds}"; fi
if [[ "${preserved_evidence_file}" != "__STUHELPER_UNSET__" ]]; then IDENTITY_PUBLIC_SMOKE_EVIDENCE_FILE="${preserved_evidence_file}"; fi
if [[ "${preserved_smoke_client_id}" != "__STUHELPER_UNSET__" ]]; then IDENTITY_PUBLIC_SMOKE_CLIENT_ID="${preserved_smoke_client_id}"; fi
if [[ "${preserved_smoke_redirect_uri}" != "__STUHELPER_UNSET__" ]]; then IDENTITY_PUBLIC_SMOKE_REDIRECT_URI="${preserved_smoke_redirect_uri}"; fi
if [[ "${preserved_smoke_scope}" != "__STUHELPER_UNSET__" ]]; then IDENTITY_PUBLIC_SMOKE_SCOPE="${preserved_smoke_scope}"; fi
if [[ "${preserved_smoke_client_secret}" != "__STUHELPER_UNSET__" ]]; then IDENTITY_PUBLIC_SMOKE_CLIENT_SECRET="${preserved_smoke_client_secret}"; fi
if [[ "${preserved_smoke_client_credentials_scope}" != "__STUHELPER_UNSET__" ]]; then IDENTITY_PUBLIC_SMOKE_CLIENT_CREDENTIALS_SCOPE="${preserved_smoke_client_credentials_scope}"; fi
if [[ "${preserved_smoke_resource_access_resource_type}" != "__STUHELPER_UNSET__" ]]; then IDENTITY_PUBLIC_SMOKE_RESOURCE_ACCESS_RESOURCE_TYPE="${preserved_smoke_resource_access_resource_type}"; fi
if [[ "${preserved_smoke_resource_access_resource_id}" != "__STUHELPER_UNSET__" ]]; then IDENTITY_PUBLIC_SMOKE_RESOURCE_ACCESS_RESOURCE_ID="${preserved_smoke_resource_access_resource_id}"; fi
if [[ "${preserved_smoke_resource_access_action}" != "__STUHELPER_UNSET__" ]]; then IDENTITY_PUBLIC_SMOKE_RESOURCE_ACCESS_ACTION="${preserved_smoke_resource_access_action}"; fi
if [[ "${preserved_smoke_resource_access_expect_allowed}" != "__STUHELPER_UNSET__" ]]; then IDENTITY_PUBLIC_SMOKE_RESOURCE_ACCESS_EXPECT_ALLOWED="${preserved_smoke_resource_access_expect_allowed}"; fi
if [[ "${preserved_allow_local_targets}" != "__STUHELPER_UNSET__" ]]; then IDENTITY_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS="${preserved_allow_local_targets}"; fi
if [[ "${preserved_casdoor_upstream_enabled}" != "__STUHELPER_UNSET__" ]]; then IDENTITY_PUBLIC_SMOKE_CASDOOR_UPSTREAM_ENABLED="${preserved_casdoor_upstream_enabled}"; fi
if [[ "${preserved_curl_insecure}" != "__STUHELPER_UNSET__" ]]; then IDENTITY_PUBLIC_SMOKE_CURL_INSECURE="${preserved_curl_insecure}"; fi

curl() {
  if [[ "${IDENTITY_PUBLIC_SMOKE_CURL_INSECURE:-false}" == "true" ]]; then
    command curl --insecure "$@"
    return
  fi
  command curl "$@"
}

trim_trailing_slash() {
  local value="$1"
  printf '%s\n' "${value%/}"
}

reject_local_smoke_target() {
  local name="$1"
  local value="$2"
  case "${value}" in
    *localhost*|*127.0.0.1*|*::1*|*host.docker.internal*)
      die "${name} points to a local target (${value}); identity-public-smoke verifies public production ingress. Set IDENTITY_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS=true only for local contract tests or intentional local validation."
      ;;
  esac
}

identity_issuer="$(trim_trailing_slash "${IDENTITY_ISSUER:-}")"
casdoor_issuer="$(trim_trailing_slash "${CASDOOR_ISSUER:-}")"
web_public_url="$(trim_trailing_slash "${WEB_PUBLIC_URL:-https://stuhelper.com}")"
retries="${IDENTITY_PUBLIC_SMOKE_RETRIES:-30}"
sleep_seconds="${IDENTITY_PUBLIC_SMOKE_SLEEP_SECONDS:-2}"
evidence_file="${IDENTITY_PUBLIC_SMOKE_EVIDENCE_FILE:-${REPO_ROOT}/infra/generated/identity-public-smoke-evidence.json}"
smoke_client_id="${IDENTITY_PUBLIC_SMOKE_CLIENT_ID:-}"
smoke_redirect_uri="${IDENTITY_PUBLIC_SMOKE_REDIRECT_URI:-}"
smoke_scope="${IDENTITY_PUBLIC_SMOKE_SCOPE:-openid}"
smoke_client_secret="${IDENTITY_PUBLIC_SMOKE_CLIENT_SECRET:-}"
smoke_client_credentials_scope="${IDENTITY_PUBLIC_SMOKE_CLIENT_CREDENTIALS_SCOPE:-resource.read}"
smoke_resource_access_resource_type="${IDENTITY_PUBLIC_SMOKE_RESOURCE_ACCESS_RESOURCE_TYPE:-resource_item}"
smoke_resource_access_resource_id="${IDENTITY_PUBLIC_SMOKE_RESOURCE_ACCESS_RESOURCE_ID:-identity-public-smoke-$(date +%s%N)}"
smoke_resource_access_action="${IDENTITY_PUBLIC_SMOKE_RESOURCE_ACCESS_ACTION:-}"
smoke_resource_access_expect_allowed="${IDENTITY_PUBLIC_SMOKE_RESOURCE_ACCESS_EXPECT_ALLOWED:-false}"
allow_local_targets="${IDENTITY_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS:-false}"
casdoor_upstream_enabled="${IDENTITY_PUBLIC_SMOKE_CASDOOR_UPSTREAM_ENABLED:-false}"
pkce_s256_challenge="E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
casdoor_jwks_url=""

[[ -n "${identity_issuer}" ]] || die "IDENTITY_ISSUER is required"
case "${allow_local_targets}" in
  true | TRUE | 1 | yes | YES) allow_local_targets="true" ;;
  false | FALSE | 0 | no | NO | "") allow_local_targets="false" ;;
  *) die "IDENTITY_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS must be true or false" ;;
esac
case "${casdoor_upstream_enabled}" in
  true | TRUE | 1 | yes | YES) casdoor_upstream_enabled="true" ;;
  false | FALSE | 0 | no | NO | "") casdoor_upstream_enabled="false" ;;
  *) die "IDENTITY_PUBLIC_SMOKE_CASDOOR_UPSTREAM_ENABLED must be true or false" ;;
esac
if [[ "${casdoor_upstream_enabled}" == "true" ]]; then
  [[ -n "${casdoor_issuer}" ]] || die "CASDOOR_ISSUER is required when IDENTITY_PUBLIC_SMOKE_CASDOOR_UPSTREAM_ENABLED=true"
  casdoor_jwks_url="${casdoor_issuer}/.well-known/jwks"
fi
if [[ "${allow_local_targets}" != "true" ]]; then
  reject_local_smoke_target "WEB_PUBLIC_URL" "${web_public_url}"
  reject_local_smoke_target "IDENTITY_ISSUER" "${identity_issuer}"
  if [[ "${casdoor_upstream_enabled}" == "true" ]]; then
    reject_local_smoke_target "CASDOOR_ISSUER" "${casdoor_issuer}"
  fi
fi
if [[ -n "${smoke_client_id}" && -z "${smoke_redirect_uri}" ]]; then
  die "IDENTITY_PUBLIC_SMOKE_REDIRECT_URI is required when IDENTITY_PUBLIC_SMOKE_CLIENT_ID is set"
fi
if [[ -z "${smoke_client_id}" && -n "${smoke_redirect_uri}" ]]; then
  die "IDENTITY_PUBLIC_SMOKE_CLIENT_ID is required when IDENTITY_PUBLIC_SMOKE_REDIRECT_URI is set"
fi
if [[ -z "${smoke_client_id}" && -n "${smoke_client_secret}" ]]; then
  die "IDENTITY_PUBLIC_SMOKE_CLIENT_ID is required when IDENTITY_PUBLIC_SMOKE_CLIENT_SECRET is set"
fi
if [[ -z "${smoke_resource_access_action}" ]]; then
  if [[ " ${smoke_client_credentials_scope} " == *" resource.read "* ]]; then
    smoke_resource_access_action="read"
  else
    smoke_resource_access_action="write"
  fi
fi
case "${smoke_resource_access_expect_allowed}" in
  true | TRUE | 1 | yes | YES) smoke_resource_access_expect_allowed="true" ;;
  false | FALSE | 0 | no | NO | "") smoke_resource_access_expect_allowed="false" ;;
  *) die "IDENTITY_PUBLIC_SMOKE_RESOURCE_ACCESS_EXPECT_ALLOWED must be true or false" ;;
esac

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
  printf '  ✅ %s\n' "$1"
  pass=$((pass + 1))
  record_check "true" "$1" "${details}"
}

record_fail() {
  local details="${2-}"
  [[ -n "${details}" ]] || details="{}"
  printf '  ❌ %s\n' "$1" >&2
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

oauth_no_store_headers_present() {
  local headers_file="$1"
  awk '
    {
      line = tolower($0)
      sub(/\r$/, "", line)
      if (line ~ /^cache-control:/ && line ~ /(^|[,[:space:]])no-store([,[:space:]]|$)/) {
        cache_control = 1
      }
      if (line ~ /^pragma:/ && line ~ /(^|[,[:space:]])no-cache([,[:space:]]|$)/) {
        pragma = 1
      }
    }
    END { exit !(cache_control && pragma) }
  ' "${headers_file}"
}

basic_invalid_client_challenge_present() {
  local headers_file="$1"
  awk '
    {
      line = tolower($0)
      sub(/\r$/, "", line)
      if (line ~ /^www-authenticate:/ && line ~ /basic/ && line ~ /realm="stuhelper identity"/) {
        found = 1
      }
    }
    END { exit !found }
  ' "${headers_file}"
}

bearer_invalid_token_challenge_present() {
  local headers_file="$1"
  awk '
    {
      line = tolower($0)
      sub(/\r$/, "", line)
      if (line ~ /^www-authenticate:/ && line ~ /bearer/ && line ~ /realm="stuhelper identity"/ && line ~ /error="invalid_token"/) {
        found = 1
      }
    }
    END { exit !found }
  ' "${headers_file}"
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
      "${web_public_url}" \
      "${identity_issuer}" \
      "${casdoor_issuer}" \
      "${casdoor_jwks_url}" \
      "${pass}" \
      "${fail}" \
      "${passed}" \
      "${casdoor_upstream_enabled}" \
      "${check_jsonl}" <<'PY'
import json
import sys
from pathlib import Path

checks_path = Path(sys.argv[11])
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
    "passed": sys.argv[9] == "true",
    "webPublicURL": sys.argv[3],
    "identityIssuer": sys.argv[4],
    "casdoorIssuer": sys.argv[5],
    "casdoorUpstreamChecked": sys.argv[10] == "true",
    "endpoints": {
        "webHealth": sys.argv[3] + "/health/ready",
        "identityDiscovery": sys.argv[4] + "/.well-known/openid-configuration",
        "identityAuthorizationServerMetadata": sys.argv[4] + "/.well-known/oauth-authorization-server",
        "identityJWKS": sys.argv[4] + "/.well-known/jwks.json",
        "identityFavicon": sys.argv[4] + "/favicon.ico",
        "identityWebManifest": sys.argv[4] + "/site.webmanifest",
        "identityAuthorize": sys.argv[4] + "/oauth2/authorize",
        "identityToken": sys.argv[4] + "/oauth2/token",
        "identityIntrospection": sys.argv[4] + "/oauth2/introspect",
        "identityRevocation": sys.argv[4] + "/oauth2/revoke",
        "identityLogout": sys.argv[4] + "/oauth2/logout",
        "identityUserInfo": sys.argv[4] + "/oidc/userinfo",
        "openPlatformResourceAccessCheck": sys.argv[3] + "/api/v1/open-platform/resources/access/check",
    },
    "summary": {
        "passed": int(sys.argv[7]),
        "failed": int(sys.argv[8]),
    },
    "checks": checks,
}
if sys.argv[10] == "true":
    bundle["endpoints"]["casdoorDiscovery"] = sys.argv[5] + "/.well-known/openid-configuration"
    bundle["endpoints"]["casdoorJWKS"] = sys.argv[6]
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
    log "wrote public identity smoke evidence to ${evidence_file}" >&2
  else
    printf '%s\n' "${bundle}" | jq .
  fi
}

urlencode() {
  python3 - "$1" <<'PY'
from urllib.parse import quote
import sys
print(quote(sys.argv[1], safe=""))
PY
}

wait_for_http() {
  local name="$1"
  local url="$2"
  local attempt
  local response_file
  local error_file
  local status
  local curl_error
  local snippet
  response_file="$(mktemp)"
  error_file="$(mktemp)"
  printf '[identity-smoke] 等待 %s: %s\n' "${name}" "${url}"
  for ((attempt = 1; attempt <= retries; attempt++)); do
    : >"${response_file}"
    : >"${error_file}"
    if ! status="$(curl -sS --location --max-time 10 -o "${response_file}" -w '%{http_code}' "${url}" 2>"${error_file}")"; then
      status="${status:-000}"
    fi
    curl_error="$(body_snippet "${error_file}")"
    if [[ "${status}" =~ ^[0-9][0-9][0-9]$ ]] && (( status >= 200 && status < 300 )); then
      rm -f "${response_file}" "${error_file}"
      record_pass "${name} 就绪" "$(json_detail url "${url}" attempts "${attempt}" httpStatus "${status}" curlError "${curl_error}")"
      return 0
    fi
    if (( attempt < retries )); then
      sleep "${sleep_seconds}"
    fi
  done
  snippet="$(body_snippet "${response_file}")"
  rm -f "${response_file}" "${error_file}"
  record_fail "${name} 超时未就绪: ${url}" "$(json_detail url "${url}" attempts "${retries}" httpStatus "${status:-000}" curlError "${curl_error:-}" bodySnippet "${snippet}")"
  return 1
}

fetch_json() {
  local name="$1"
  local url="$2"
  local output_file="$3"
  local error_file
  local status
  local curl_error
  local snippet
  error_file="$(mktemp)"
  if ! status="$(curl -sS --max-time 10 -w '%{http_code}' -o "${output_file}" "${url}" 2>"${error_file}")"; then
    status="${status:-000}"
    curl_error="$(body_snippet "${error_file}")"
    rm -f "${error_file}"
    record_fail "${name} 请求失败: ${url}" "$(json_detail url "${url}" httpStatus "${status}" curlError "${curl_error}")"
    return 1
  fi
  curl_error="$(body_snippet "${error_file}")"
  rm -f "${error_file}"
  if [[ ! "${status}" =~ ^[0-9][0-9][0-9]$ ]] || (( status < 200 || status >= 300 )); then
    snippet="$(body_snippet "${output_file}")"
    record_fail "${name} 请求失败: ${url}" "$(json_detail url "${url}" httpStatus "${status}" curlError "${curl_error}" bodySnippet "${snippet}")"
    return 1
  fi
  if ! jq -e 'type == "object"' "${output_file}" >/dev/null; then
    snippet="$(body_snippet "${output_file}")"
    record_fail "${name} 响应不是 JSON object: ${url}" "$(json_detail url "${url}" httpStatus "${status}" bodySnippet "${snippet}")"
    return 1
  fi
}

check_identity_web_asset() {
  local name="$1"
  local url="$2"
  local expected_text="${3:-}"
  local min_bytes="${4:-1}"
  local response_file
  local headers_file
  local error_file
  local status
  local location
  local content_type
  local bytes
  local curl_error
  local snippet
  response_file="$(mktemp)"
  headers_file="$(mktemp)"
  error_file="$(mktemp)"

  if ! status="$(curl -sS --max-time 10 -D "${headers_file}" -w '%{http_code}' -o "${response_file}" "${url}" 2>"${error_file}")"; then
    status="${status:-000}"
  fi
  location="$(response_header "${headers_file}" "Location")"
  content_type="$(response_header "${headers_file}" "Content-Type")"
  bytes="$(wc -c <"${response_file}" | tr -d '[:space:]')"
  curl_error="$(body_snippet "${error_file}")"

  if [[ "${status}" =~ ^[0-9][0-9][0-9]$ ]] && \
    (( status >= 200 && status < 300 )) && \
    (( bytes >= min_bytes )) && \
    { [[ -z "${expected_text}" ]] || grep -Fq -- "${expected_text}" "${response_file}"; }; then
    rm -f "${response_file}" "${headers_file}" "${error_file}"
    record_pass "${name}" "$(json_detail url "${url}" httpStatus "${status}" bytes "${bytes}" contentType "${content_type}" curlError "${curl_error}")"
    return
  fi

  snippet="$(body_snippet "${response_file}")"
  rm -f "${response_file}" "${headers_file}" "${error_file}"
  record_fail "${name} 响应异常，status=${status}" "$(json_detail url "${url}" httpStatus "${status}" expectedText "${expected_text}" minBytes "${min_bytes}" bytes "${bytes}" contentType "${content_type}" location "${location}" bodySnippet "${snippet}" curlError "${curl_error}")"
}

check_identity_discovery() {
  local metadata_file="$1"
  local check_name="${2:-Identity discovery 元数据}"
  local fail_name="${3:-Identity discovery 元数据不匹配 ${identity_issuer}}"
  if python3 - "${metadata_file}" "${identity_issuer}" <<'PY' >/dev/null; then
import json
import sys

metadata_path, issuer = sys.argv[1], sys.argv[2].rstrip("/")
with open(metadata_path, "r", encoding="utf-8") as fh:
    metadata = json.load(fh)

expected = {
    "issuer": issuer,
    "authorization_endpoint": issuer + "/oauth2/authorize",
    "token_endpoint": issuer + "/oauth2/token",
    "userinfo_endpoint": issuer + "/oidc/userinfo",
    "jwks_uri": issuer + "/.well-known/jwks.json",
    "revocation_endpoint": issuer + "/oauth2/revoke",
    "introspection_endpoint": issuer + "/oauth2/introspect",
    "end_session_endpoint": issuer + "/oauth2/logout",
}
for key, value in expected.items():
    if metadata.get(key) != value:
        raise SystemExit(1)

if metadata.get("authorization_response_iss_parameter_supported") is not True:
    raise SystemExit(1)

required_values = {
    "response_types_supported": ["code"],
    "response_modes_supported": ["query"],
    "code_challenge_methods_supported": ["S256"],
    "prompt_values_supported": ["none", "login", "consent"],
    "scopes_supported": ["offline_access"],
    "subject_types_supported": ["public"],
    "id_token_signing_alg_values_supported": ["RS256"],
    "token_endpoint_auth_methods_supported": ["client_secret_basic", "client_secret_post"],
    "revocation_endpoint_auth_methods_supported": ["client_secret_basic", "client_secret_post"],
    "introspection_endpoint_auth_methods_supported": ["client_secret_basic", "client_secret_post"],
    "claims_supported": [
        "sub",
        "preferred_username",
        "name",
        "email",
        "email_verified",
        "phone_number",
        "phone_number_verified",
        "identityVerified",
        "identityType",
        "studentVerified",
        "school",
    ],
}
for key, expected_values in required_values.items():
    values = metadata.get(key)
    if not isinstance(values, list):
        raise SystemExit(1)
    for value in expected_values:
        if value not in values:
            raise SystemExit(1)

grant_types = metadata.get("grant_types_supported")
if not isinstance(grant_types, list):
    raise SystemExit(1)
for value in ("authorization_code", "refresh_token", "client_credentials"):
    if value not in grant_types:
        raise SystemExit(1)
PY
    record_pass "${check_name}" "$(json_detail issuer "${identity_issuer}")"
  else
    record_fail "${fail_name}" "$(json_detail expectedIssuer "${identity_issuer}")"
  fi
}

check_jwks() {
  local label="${1:-JWKS}"
  shift || true
  local body="$1"
  if printf '%s\n' "${body}" | jq -e '.keys | type == "array" and length > 0' >/dev/null; then
    record_pass "${label} JWKS"
  else
    record_fail "${label} JWKS 未返回可用 keys" "$(json_detail expected "non-empty keys array")"
  fi
}

check_authorize_redirect() {
  local redirect_uri="https://client.example.com/callback"
  local encoded_redirect
  local headers_file
  local error_file
  local location
  local status
  local curl_error
  encoded_redirect="$(urlencode "${redirect_uri}")"
  local authorize_url="${identity_issuer}/oauth2/authorize?response_type=code&client_id=smoke-client&redirect_uri=${encoded_redirect}&scope=openid&state=identity-public-smoke&code_challenge=${pkce_s256_challenge}&code_challenge_method=S256"

  headers_file="$(mktemp)"
  error_file="$(mktemp)"
  if ! status="$(curl -sS -o /dev/null -D "${headers_file}" -w '%{http_code}' --max-time 10 "${authorize_url}" 2>"${error_file}")"; then
    status="${status:-000}"
  fi
  location="$(awk 'BEGIN{IGNORECASE=1} /^location:/ {sub(/\r$/, ""); print substr($0, index($0, ":") + 2); exit}' "${headers_file}")"
  curl_error="$(body_snippet "${error_file}")"
  rm -f "${headers_file}" "${error_file}"

  if [[ "${status}" == "302" && "${location}" == "${identity_issuer}/login"* ]]; then
    record_pass "Identity authorize 未认证跳转登录" "$(json_detail url "${authorize_url}" httpStatus "${status}" location "${location}" curlError "${curl_error}")"
  else
    record_fail "Identity authorize 未按预期 302 到 ${identity_issuer}/login，status=${status}, location=${location:-<empty>}" "$(json_detail url "${authorize_url}" httpStatus "${status}" expectedLocationPrefix "${identity_issuer}/login" location "${location:-}" curlError "${curl_error}")"
  fi
}

check_authorize_reauth_redirect() {
  local redirect_uri="https://client.example.com/callback"
  local encoded_redirect
  local headers_file
  local error_file
  local location
  local status
  local curl_error
  encoded_redirect="$(urlencode "${redirect_uri}")"
  local authorize_url="${identity_issuer}/oauth2/authorize?response_type=code&client_id=smoke-client&redirect_uri=${encoded_redirect}&scope=openid&state=identity-public-reauth-smoke&code_challenge=${pkce_s256_challenge}&code_challenge_method=S256&prompt=login&max_age=0"

  headers_file="$(mktemp)"
  error_file="$(mktemp)"
  if ! status="$(curl -sS -o /dev/null -D "${headers_file}" -w '%{http_code}' --max-time 10 "${authorize_url}" 2>"${error_file}")"; then
    status="${status:-000}"
  fi
  location="$(awk 'BEGIN{IGNORECASE=1} /^location:/ {sub(/\r$/, ""); print substr($0, index($0, ":") + 2); exit}' "${headers_file}")"
  curl_error="$(body_snippet "${error_file}")"
  rm -f "${headers_file}" "${error_file}"

  if [[ "${status}" == "302" ]] && python3 - "${location}" "${identity_issuer}" <<'PY' >/dev/null; then
from urllib.parse import parse_qs, urlparse
import sys

location = sys.argv[1]
issuer = sys.argv[2].rstrip("/")
parsed = urlparse(location)
if parsed.scheme + "://" + parsed.netloc + parsed.path != issuer + "/login":
    raise SystemExit(1)
query = parse_qs(parsed.query)
if query.get("reauth") != ["1"]:
    raise SystemExit(1)
redirect_values = query.get("redirect")
if not redirect_values:
    raise SystemExit(1)
target = urlparse(redirect_values[0])
issuer_path = urlparse(issuer).path.rstrip("/")
expected_authorize_path = issuer_path + "/oauth2/authorize"
if target.path != expected_authorize_path:
    raise SystemExit(1)
target_query = parse_qs(target.query)
if "login" in " ".join(target_query.get("prompt", [])):
    raise SystemExit(1)
if target_query.get("max_age") == ["0"]:
    raise SystemExit(1)
PY
    record_pass "Identity authorize 重新认证跳转" "$(json_detail url "${authorize_url}" httpStatus "${status}" location "${location}" curlError "${curl_error}")"
  else
    record_fail "Identity authorize 未按预期 302 到 reauth 登录，status=${status}, location=${location:-<empty>}" "$(json_detail url "${authorize_url}" httpStatus "${status}" expectedLocationPrefix "${identity_issuer}/login?reauth=1" location "${location:-}" curlError "${curl_error}")"
  fi
}

check_registered_prompt_none_login_required() {
  if [[ -z "${smoke_client_id}" ]]; then
    log "skipping registered-client prompt=none check; set IDENTITY_PUBLIC_SMOKE_CLIENT_ID and IDENTITY_PUBLIC_SMOKE_REDIRECT_URI to enable" >&2
    return
  fi

  local encoded_redirect
  local encoded_scope
  local encoded_client_id
  local headers_file
  local error_file
  local location
  local status
  local curl_error
  encoded_redirect="$(urlencode "${smoke_redirect_uri}")"
  encoded_scope="$(urlencode "${smoke_scope}")"
  encoded_client_id="$(urlencode "${smoke_client_id}")"
  local authorize_url="${identity_issuer}/oauth2/authorize?response_type=code&client_id=${encoded_client_id}&redirect_uri=${encoded_redirect}&scope=${encoded_scope}&state=identity-public-prompt-none-smoke&code_challenge=${pkce_s256_challenge}&code_challenge_method=S256&prompt=none"

  headers_file="$(mktemp)"
  error_file="$(mktemp)"
  if ! status="$(curl -sS -o /dev/null -D "${headers_file}" -w '%{http_code}' --max-time 10 "${authorize_url}" 2>"${error_file}")"; then
    status="${status:-000}"
  fi
  location="$(awk 'BEGIN{IGNORECASE=1} /^location:/ {sub(/\r$/, ""); print substr($0, index($0, ":") + 2); exit}' "${headers_file}")"
  curl_error="$(body_snippet "${error_file}")"
  rm -f "${headers_file}" "${error_file}"

  if [[ "${status}" == "302" ]] && python3 - "${location}" "${smoke_redirect_uri}" "${identity_issuer}" <<'PY' >/dev/null; then
from urllib.parse import parse_qs, urlparse
import sys

location = sys.argv[1]
redirect_uri = sys.argv[2]
issuer = sys.argv[3].rstrip("/")
parsed = urlparse(location)
expected = urlparse(redirect_uri)
if (parsed.scheme, parsed.netloc, parsed.path) != (expected.scheme, expected.netloc, expected.path):
    raise SystemExit(1)
query = parse_qs(parsed.query)
if query.get("error") != ["login_required"]:
    raise SystemExit(1)
if query.get("state") != ["identity-public-prompt-none-smoke"]:
    raise SystemExit(1)
if query.get("iss") != [issuer]:
    raise SystemExit(1)
PY
    record_pass "Identity prompt=none 未登录错误回调" "$(json_detail url "${authorize_url}" httpStatus "${status}" location "${location}" clientID "${smoke_client_id}" curlError "${curl_error}")"
  else
    record_fail "Identity prompt=none 未按预期回调 login_required，status=${status}, location=${location:-<empty>}" "$(json_detail url "${authorize_url}" httpStatus "${status}" expectedRedirectURI "${smoke_redirect_uri}" expectedError "login_required" expectedIssuer "${identity_issuer}" location "${location:-}" clientID "${smoke_client_id}" curlError "${curl_error}")"
  fi
}

write_smoke_client_netrc() {
  local output_file="$1"
  python3 - "${identity_issuer}" "${smoke_client_id}" "${smoke_client_secret}" "${output_file}" <<'PY'
from pathlib import Path
from urllib.parse import urlparse
import sys

issuer, client_id, client_secret, output_file = sys.argv[1:]
host = urlparse(issuer).hostname
if not host:
    raise SystemExit("identity issuer host is required")
Path(output_file).write_text(
    f"machine {host}\n  login {client_id}\n  password {client_secret}\n",
    encoding="utf-8",
)
PY
  chmod 600 "${output_file}"
}

check_client_credentials_resource_access_api() {
  local access_token="$1"
  local required_scope
  local response_file
  local request_file
  local curl_config_file
  local error_file
  local status
  local curl_error
  local snippet
  local url="${web_public_url}/api/v1/open-platform/resources/access/check"
  local expected_allowed="${smoke_resource_access_expect_allowed}"
  local expected_reason="fga_denied"
  local check_name="Open Platform resource access API 拒绝未授权随机资源"
  local fail_name="Open Platform resource access API 未按预期拒绝未授权随机资源"
  case "${smoke_resource_access_action}" in
    read) required_scope="resource.read" ;;
    write) required_scope="resource.write" ;;
    *)
      record_fail "Open Platform resource access API smoke action 配置无效" "$(json_detail action "${smoke_resource_access_action}" expected "read|write")"
      return
      ;;
  esac
  if [[ " ${smoke_client_credentials_scope} " != *" ${required_scope} "* ]]; then
    record_fail "Open Platform resource access API smoke scope 配置不足" "$(json_detail action "${smoke_resource_access_action}" requiredScope "${required_scope}" scope "${smoke_client_credentials_scope}")"
    return
  fi
  if [[ "${expected_allowed}" == "true" ]]; then
    expected_reason="allowed"
    check_name="Open Platform resource access API 允许已授权资源"
    fail_name="Open Platform resource access API 未按预期允许已授权资源"
  fi

  response_file="$(mktemp)"
  request_file="$(mktemp)"
  curl_config_file="$(mktemp)"
  error_file="$(mktemp)"
  python3 - \
    "${smoke_resource_access_resource_type}" \
    "${smoke_resource_access_resource_id}" \
    "${smoke_resource_access_action}" \
    "${request_file}" <<'PY'
import json
import sys

resource_type, resource_id, action, output_file = sys.argv[1:]
payload = {
    "resourceType": resource_type,
    "resourceID": resource_id,
    "action": action,
}
with open(output_file, "w", encoding="utf-8") as fh:
    json.dump(payload, fh, ensure_ascii=True, separators=(",", ":"))
PY
  printf 'header = "Authorization: Bearer %s"\n' "${access_token}" >"${curl_config_file}"
  chmod 600 "${curl_config_file}"

  if ! status="$(curl -sS -X POST \
    --config "${curl_config_file}" \
    -H 'Content-Type: application/json' \
    -o "${response_file}" \
    -w '%{http_code}' \
    --max-time 10 \
    --data-binary "@${request_file}" \
    "${url}" 2>"${error_file}")"; then
    status="${status:-000}"
  fi
  curl_error="$(body_snippet "${error_file}")"
  if [[ "${status}" == "200" ]] && jq -e \
      --arg client_id "${smoke_client_id}" \
      --arg resource_type "${smoke_resource_access_resource_type}" \
      --arg resource_id "${smoke_resource_access_resource_id}" \
      --arg action "${smoke_resource_access_action}" \
      --argjson allowed "${expected_allowed}" \
      --arg reason "${expected_reason}" '
      .success == true
      and .data.allowed == $allowed
      and .data.clientID == $client_id
      and .data.resourceType == $resource_type
      and .data.resourceID == $resource_id
      and .data.action == $action
      and .data.reason == $reason
    ' "${response_file}" >/dev/null 2>&1; then
    record_pass "${check_name}" "$(json_detail url "${url}" httpStatus "${status}" clientID "${smoke_client_id}" resourceType "${smoke_resource_access_resource_type}" resourceID "${smoke_resource_access_resource_id}" action "${smoke_resource_access_action}" expectedAllowed "${expected_allowed}" expectedReason "${expected_reason}" curlError "${curl_error}")"
  else
    snippet="$(body_snippet "${response_file}")"
    record_fail "${fail_name}，status=${status}" "$(json_detail url "${url}" httpStatus "${status}" clientID "${smoke_client_id}" resourceType "${smoke_resource_access_resource_type}" resourceID "${smoke_resource_access_resource_id}" action "${smoke_resource_access_action}" expectedAllowed "${expected_allowed}" expectedReason "${expected_reason}" bodySnippet "${snippet}" curlError "${curl_error}")"
  fi
  rm -f "${response_file}" "${request_file}" "${curl_config_file}" "${error_file}"
}

check_client_credentials_grant() {
  if [[ -z "${smoke_client_secret}" ]]; then
    log "skipping client_credentials token check; set IDENTITY_PUBLIC_SMOKE_CLIENT_SECRET to enable" >&2
    return
  fi

  local netrc_file
  local token_file
  local headers_file
  local error_file
  local status
  local curl_error
  local snippet
  local access_token
  local token_value_file
  local encoded_scope
  local cache_control
  local pragma
  local www_authenticate
  netrc_file="$(mktemp)"
  token_file="$(mktemp)"
  headers_file="$(mktemp)"
  error_file="$(mktemp)"
  token_value_file="$(mktemp)"
  chmod 600 "${token_value_file}"
  write_smoke_client_netrc "${netrc_file}"
  encoded_scope="$(urlencode "${smoke_client_credentials_scope}")"

  if ! status="$(curl -sS -X POST \
    --netrc-file "${netrc_file}" \
    -H 'Content-Type: application/x-www-form-urlencoded' \
    -D "${headers_file}" \
    -o "${token_file}" \
    -w '%{http_code}' \
    --max-time 10 \
    --data "grant_type=client_credentials&scope=${encoded_scope}" \
    "${identity_issuer}/oauth2/token" 2>"${error_file}")"; then
    status="${status:-000}"
  fi
  curl_error="$(body_snippet "${error_file}")"
  cache_control="$(response_header "${headers_file}" "Cache-Control")"
  pragma="$(response_header "${headers_file}" "Pragma")"
  if [[ "${status}" == "200" ]] && jq -e --arg scope "${smoke_client_credentials_scope}" '
      (.access_token | type == "string" and length > 0)
      and .token_type == "Bearer"
      and .scope == $scope
      and (has("id_token") | not)
      and (has("refresh_token") | not)
    ' "${token_file}" >/dev/null 2>&1 && oauth_no_store_headers_present "${headers_file}"; then
    record_pass "Identity client_credentials token 签发" "$(json_detail url "${identity_issuer}/oauth2/token" httpStatus "${status}" clientID "${smoke_client_id}" scope "${smoke_client_credentials_scope}" cacheControl "${cache_control}" pragma "${pragma}" curlError "${curl_error}")"
  else
    snippet="$(body_snippet "${token_file}")"
    rm -f "${netrc_file}" "${token_file}" "${headers_file}" "${error_file}" "${token_value_file}"
    record_fail "Identity client_credentials token 签发异常，status=${status}" "$(json_detail url "${identity_issuer}/oauth2/token" httpStatus "${status}" clientID "${smoke_client_id}" scope "${smoke_client_credentials_scope}" expectedCacheControl "no-store" expectedPragma "no-cache" cacheControl "${cache_control}" pragma "${pragma}" bodySnippet "${snippet}" curlError "${curl_error}")"
    return
  fi

  access_token="$(jq -r '.access_token' "${token_file}")"
  printf '%s' "${access_token}" >"${token_value_file}"
  rm -f "${token_file}" "${headers_file}" "${error_file}"

  local introspection_file
  introspection_file="$(mktemp)"
  headers_file="$(mktemp)"
  error_file="$(mktemp)"
  if ! status="$(curl -sS -X POST \
    --netrc-file "${netrc_file}" \
    -H 'Content-Type: application/x-www-form-urlencoded' \
	    -D "${headers_file}" \
	    -o "${introspection_file}" \
	    -w '%{http_code}' \
	    --max-time 10 \
	    --data-urlencode "token@${token_value_file}" \
	    --data-urlencode "token_type_hint=access_token" \
	    "${identity_issuer}/oauth2/introspect" 2>"${error_file}")"; then
    status="${status:-000}"
  fi
  curl_error="$(body_snippet "${error_file}")"
  cache_control="$(response_header "${headers_file}" "Cache-Control")"
  pragma="$(response_header "${headers_file}" "Pragma")"
  if [[ "${status}" == "200" ]] && jq -e --arg client_id "${smoke_client_id}" --arg scope "${smoke_client_credentials_scope}" '
      .active == true
      and .client_id == $client_id
      and .grant_type == "client_credentials"
      and .token_type == "Bearer"
      and .token_kind == "access_token"
      and .scope == $scope
    ' "${introspection_file}" >/dev/null 2>&1 && oauth_no_store_headers_present "${headers_file}"; then
    record_pass "Identity client_credentials introspection active" "$(json_detail url "${identity_issuer}/oauth2/introspect" httpStatus "${status}" clientID "${smoke_client_id}" scope "${smoke_client_credentials_scope}" cacheControl "${cache_control}" pragma "${pragma}" curlError "${curl_error}")"
  else
    snippet="$(body_snippet "${introspection_file}")"
    rm -f "${netrc_file}" "${headers_file}" "${error_file}" "${introspection_file}" "${token_value_file}"
    record_fail "Identity client_credentials introspection active 异常，status=${status}" "$(json_detail url "${identity_issuer}/oauth2/introspect" httpStatus "${status}" clientID "${smoke_client_id}" scope "${smoke_client_credentials_scope}" expectedCacheControl "no-store" expectedPragma "no-cache" cacheControl "${cache_control}" pragma "${pragma}" bodySnippet "${snippet}" curlError "${curl_error}")"
    return
  fi
  rm -f "${headers_file}" "${error_file}" "${introspection_file}"

  local userinfo_file
  local userinfo_config_file
  userinfo_file="$(mktemp)"
  userinfo_config_file="$(mktemp)"
  headers_file="$(mktemp)"
  error_file="$(mktemp)"
  printf 'header = "Authorization: Bearer %s"\n' "${access_token}" >"${userinfo_config_file}"
  chmod 600 "${userinfo_config_file}"
  if ! status="$(curl -sS \
    --config "${userinfo_config_file}" \
    -D "${headers_file}" \
    -o "${userinfo_file}" \
    -w '%{http_code}' \
    --max-time 10 \
    "${identity_issuer}/oidc/userinfo" 2>"${error_file}")"; then
    status="${status:-000}"
  fi
  curl_error="$(body_snippet "${error_file}")"
  cache_control="$(response_header "${headers_file}" "Cache-Control")"
  pragma="$(response_header "${headers_file}" "Pragma")"
  www_authenticate="$(response_header "${headers_file}" "WWW-Authenticate")"
  if [[ "${status}" == "401" ]] && \
    jq -e '.error == "invalid_token"' "${userinfo_file}" >/dev/null 2>&1 && \
    oauth_no_store_headers_present "${headers_file}" && \
    bearer_invalid_token_challenge_present "${headers_file}"; then
    record_pass "Identity client_credentials UserInfo 拒绝 app-only token" "$(json_detail url "${identity_issuer}/oidc/userinfo" httpStatus "${status}" clientID "${smoke_client_id}" cacheControl "${cache_control}" pragma "${pragma}" wwwAuthenticate "${www_authenticate}" curlError "${curl_error}")"
  else
    snippet="$(body_snippet "${userinfo_file}")"
    rm -f "${netrc_file}" "${headers_file}" "${error_file}" "${userinfo_file}" "${userinfo_config_file}" "${token_value_file}"
    record_fail "Identity client_credentials UserInfo 未拒绝 app-only token，status=${status}" "$(json_detail url "${identity_issuer}/oidc/userinfo" httpStatus "${status}" clientID "${smoke_client_id}" expectedCacheControl "no-store" expectedPragma "no-cache" expectedWWWAuthenticate "Bearer realm=\"StuHelper Identity\", error=\"invalid_token\"" cacheControl "${cache_control}" pragma "${pragma}" wwwAuthenticate "${www_authenticate}" bodySnippet "${snippet}" curlError "${curl_error}")"
    return
  fi
  rm -f "${headers_file}" "${error_file}" "${userinfo_file}" "${userinfo_config_file}"

  local logout_hint_file
  logout_hint_file="$(mktemp)"
  headers_file="$(mktemp)"
  error_file="$(mktemp)"
  if ! status="$(curl -sS -G \
    -D "${headers_file}" \
    -o "${logout_hint_file}" \
    -w '%{http_code}' \
    --max-time 10 \
    --data-urlencode "id_token_hint@${token_value_file}" \
    --data-urlencode "post_logout_redirect_uri=${smoke_redirect_uri}" \
    "${identity_issuer}/oauth2/logout" 2>"${error_file}")"; then
    status="${status:-000}"
  fi
  curl_error="$(body_snippet "${error_file}")"
  if [[ "${status}" == "400" ]] && grep -qi 'invalid logout request' "${logout_hint_file}"; then
    record_pass "Identity logout 拒绝 access token id_token_hint" "$(json_detail url "${identity_issuer}/oauth2/logout" httpStatus "${status}" clientID "${smoke_client_id}" curlError "${curl_error}")"
  else
    snippet="$(body_snippet "${logout_hint_file}")"
    rm -f "${netrc_file}" "${headers_file}" "${error_file}" "${logout_hint_file}" "${token_value_file}"
    record_fail "Identity logout 未拒绝 access token id_token_hint，status=${status}" "$(json_detail url "${identity_issuer}/oauth2/logout" httpStatus "${status}" clientID "${smoke_client_id}" expectedStatus "400" expectedBody "invalid logout request" bodySnippet "${snippet}" curlError "${curl_error}")"
    return
  fi
  rm -f "${headers_file}" "${error_file}" "${logout_hint_file}"

  check_client_credentials_resource_access_api "${access_token}"

  local revoke_file
  revoke_file="$(mktemp)"
  headers_file="$(mktemp)"
  error_file="$(mktemp)"
  if ! status="$(curl -sS -X POST \
    --netrc-file "${netrc_file}" \
    -H 'Content-Type: application/x-www-form-urlencoded' \
    -D "${headers_file}" \
	    -o "${revoke_file}" \
	    -w '%{http_code}' \
	    --max-time 10 \
	    --data-urlencode "token@${token_value_file}" \
	    --data-urlencode "token_type_hint=access_token" \
	    "${identity_issuer}/oauth2/revoke" 2>"${error_file}")"; then
    status="${status:-000}"
  fi
  curl_error="$(body_snippet "${error_file}")"
  cache_control="$(response_header "${headers_file}" "Cache-Control")"
  pragma="$(response_header "${headers_file}" "Pragma")"
  if [[ "${status}" == "200" ]] && oauth_no_store_headers_present "${headers_file}"; then
    record_pass "Identity client_credentials revoke 成功" "$(json_detail url "${identity_issuer}/oauth2/revoke" httpStatus "${status}" clientID "${smoke_client_id}" cacheControl "${cache_control}" pragma "${pragma}" curlError "${curl_error}")"
  else
    snippet="$(body_snippet "${revoke_file}")"
    rm -f "${netrc_file}" "${headers_file}" "${error_file}" "${revoke_file}" "${token_value_file}"
    record_fail "Identity client_credentials revoke 异常，status=${status}" "$(json_detail url "${identity_issuer}/oauth2/revoke" httpStatus "${status}" clientID "${smoke_client_id}" expectedCacheControl "no-store" expectedPragma "no-cache" cacheControl "${cache_control}" pragma "${pragma}" bodySnippet "${snippet}" curlError "${curl_error}")"
    return
  fi
  rm -f "${headers_file}" "${error_file}" "${revoke_file}"

  introspection_file="$(mktemp)"
  headers_file="$(mktemp)"
  error_file="$(mktemp)"
  if ! status="$(curl -sS -X POST \
    --netrc-file "${netrc_file}" \
    -H 'Content-Type: application/x-www-form-urlencoded' \
    -D "${headers_file}" \
	    -o "${introspection_file}" \
	    -w '%{http_code}' \
	    --max-time 10 \
	    --data-urlencode "token@${token_value_file}" \
	    --data-urlencode "token_type_hint=access_token" \
	    "${identity_issuer}/oauth2/introspect" 2>"${error_file}")"; then
    status="${status:-000}"
  fi
  curl_error="$(body_snippet "${error_file}")"
  cache_control="$(response_header "${headers_file}" "Cache-Control")"
  pragma="$(response_header "${headers_file}" "Pragma")"
  if [[ "${status}" == "200" ]] && \
    jq -e '.active == false' "${introspection_file}" >/dev/null 2>&1 && \
    oauth_no_store_headers_present "${headers_file}"; then
    record_pass "Identity client_credentials revoke 后 inactive" "$(json_detail url "${identity_issuer}/oauth2/introspect" httpStatus "${status}" clientID "${smoke_client_id}" cacheControl "${cache_control}" pragma "${pragma}" curlError "${curl_error}")"
  else
    snippet="$(body_snippet "${introspection_file}")"
    record_fail "Identity client_credentials revoke 后仍 active 或响应异常，status=${status}" "$(json_detail url "${identity_issuer}/oauth2/introspect" httpStatus "${status}" clientID "${smoke_client_id}" expectedCacheControl "no-store" expectedPragma "no-cache" cacheControl "${cache_control}" pragma "${pragma}" bodySnippet "${snippet}" curlError "${curl_error}")"
  fi
  rm -f "${netrc_file}" "${headers_file}" "${error_file}" "${introspection_file}" "${token_value_file}"
}

check_oauth_post_error() {
  local name="$1"
  local url="$2"
  local form_body="$3"
  local expected_status="$4"
  local expected_error="$5"
  local response_file
  local headers_file
  local error_file
  local status
  local curl_error
  local snippet
  local cache_control
  local pragma
  local www_authenticate
  local auth_challenge_ok="true"
  local expected_www_authenticate=""
  response_file="$(mktemp)"
  headers_file="$(mktemp)"
  error_file="$(mktemp)"
  if ! status="$(curl -sS -X POST \
    -H 'Content-Type: application/x-www-form-urlencoded' \
    -D "${headers_file}" \
    -o "${response_file}" \
    -w '%{http_code}' \
    --max-time 10 \
    --data "${form_body}" \
    "${url}" 2>"${error_file}")"; then
    status="${status:-000}"
  fi
  curl_error="$(body_snippet "${error_file}")"
  cache_control="$(response_header "${headers_file}" "Cache-Control")"
  pragma="$(response_header "${headers_file}" "Pragma")"
  www_authenticate="$(response_header "${headers_file}" "WWW-Authenticate")"
  if [[ "${expected_error}" == "invalid_client" ]]; then
    expected_www_authenticate='Basic realm="StuHelper Identity"'
    if ! basic_invalid_client_challenge_present "${headers_file}"; then
      auth_challenge_ok="false"
    fi
  fi
  if [[ "${status}" == "${expected_status}" ]] && \
    jq -e --arg error "${expected_error}" '.error == $error' "${response_file}" >/dev/null 2>&1 && \
    oauth_no_store_headers_present "${headers_file}" && \
    [[ "${auth_challenge_ok}" == "true" ]]; then
    rm -f "${response_file}" "${headers_file}" "${error_file}"
    record_pass "${name}" "$(json_detail url "${url}" httpStatus "${status}" expectedError "${expected_error}" cacheControl "${cache_control}" pragma "${pragma}" wwwAuthenticate "${www_authenticate}" curlError "${curl_error}")"
    return
  fi
  snippet="$(body_snippet "${response_file}")"
  rm -f "${response_file}" "${headers_file}" "${error_file}"
  record_fail "${name} 响应异常，status=${status}" "$(json_detail url "${url}" httpStatus "${status}" expectedStatus "${expected_status}" expectedError "${expected_error}" expectedCacheControl "no-store" expectedPragma "no-cache" expectedWWWAuthenticate "${expected_www_authenticate}" cacheControl "${cache_control}" pragma "${pragma}" wwwAuthenticate "${www_authenticate}" bodySnippet "${snippet}" curlError "${curl_error}")"
}

check_oauth_post_invalid_content_type() {
  local name="$1"
  local url="$2"
  local content_type="$3"
  local body="$4"
  local response_file
  local headers_file
  local error_file
  local status
  local curl_error
  local snippet
  local cache_control
  local pragma
  response_file="$(mktemp)"
  headers_file="$(mktemp)"
  error_file="$(mktemp)"
  local -a curl_args=(
    -sS
    -X POST
    -D "${headers_file}"
    -o "${response_file}"
    -w '%{http_code}'
    --max-time 10
  )
  if [[ -n "${content_type}" ]]; then
    curl_args+=(-H "Content-Type: ${content_type}")
  else
    curl_args+=(-H "Content-Type:")
  fi
  if ! status="$(curl "${curl_args[@]}" --data-binary "${body}" "${url}" 2>"${error_file}")"; then
    status="${status:-000}"
  fi
  curl_error="$(body_snippet "${error_file}")"
  cache_control="$(response_header "${headers_file}" "Cache-Control")"
  pragma="$(response_header "${headers_file}" "Pragma")"
  if [[ "${status}" == "400" ]] && \
    jq -e '.error == "invalid_request"' "${response_file}" >/dev/null 2>&1 && \
    oauth_no_store_headers_present "${headers_file}"; then
    rm -f "${response_file}" "${headers_file}" "${error_file}"
    record_pass "${name}" "$(json_detail url "${url}" httpStatus "${status}" expectedError "invalid_request" contentType "${content_type:-<missing>}" cacheControl "${cache_control}" pragma "${pragma}" curlError "${curl_error}")"
    return
  fi
  snippet="$(body_snippet "${response_file}")"
  rm -f "${response_file}" "${headers_file}" "${error_file}"
  record_fail "${name} 响应异常，status=${status}" "$(json_detail url "${url}" httpStatus "${status}" expectedStatus "400" expectedError "invalid_request" expectedCacheControl "no-store" expectedPragma "no-cache" contentType "${content_type:-<missing>}" cacheControl "${cache_control}" pragma "${pragma}" bodySnippet "${snippet}" curlError "${curl_error}")"
}

check_oauth_post_query_rejected() {
  local name="$1"
  local url="$2"
  local form_body="$3"
  local response_file
  local headers_file
  local error_file
  local status
  local curl_error
  local snippet
  local cache_control
  local pragma
  response_file="$(mktemp)"
  headers_file="$(mktemp)"
  error_file="$(mktemp)"
  if ! status="$(curl -sS -X POST \
    -H 'Content-Type: application/x-www-form-urlencoded' \
    -D "${headers_file}" \
    -o "${response_file}" \
    -w '%{http_code}' \
    --max-time 10 \
    --data "${form_body}" \
    "${url}" 2>"${error_file}")"; then
    status="${status:-000}"
  fi
  curl_error="$(body_snippet "${error_file}")"
  cache_control="$(response_header "${headers_file}" "Cache-Control")"
  pragma="$(response_header "${headers_file}" "Pragma")"
  if [[ "${status}" == "400" ]] && \
    jq -e '.error == "invalid_request"' "${response_file}" >/dev/null 2>&1 && \
    oauth_no_store_headers_present "${headers_file}"; then
    rm -f "${response_file}" "${headers_file}" "${error_file}"
    record_pass "${name}" "$(json_detail url "${url}" httpStatus "${status}" expectedError "invalid_request" cacheControl "${cache_control}" pragma "${pragma}" curlError "${curl_error}")"
    return
  fi
  snippet="$(body_snippet "${response_file}")"
  rm -f "${response_file}" "${headers_file}" "${error_file}"
  record_fail "${name} 响应异常，status=${status}" "$(json_detail url "${url}" httpStatus "${status}" expectedStatus "400" expectedError "invalid_request" expectedCacheControl "no-store" expectedPragma "no-cache" cacheControl "${cache_control}" pragma "${pragma}" bodySnippet "${snippet}" curlError "${curl_error}")"
}

check_mixed_client_auth_error() {
  local name="$1"
  local url="$2"
  shift 2
  local netrc_file
  local secret_file
  local response_file
  local headers_file
  local error_file
  local status
  local curl_error
  local snippet
  local cache_control
  local pragma
  local www_authenticate
  netrc_file="$(mktemp)"
  secret_file="$(mktemp)"
  response_file="$(mktemp)"
  headers_file="$(mktemp)"
  error_file="$(mktemp)"
  write_smoke_client_netrc "${netrc_file}"
  printf '%s' "${smoke_client_secret}" >"${secret_file}"
  chmod 600 "${secret_file}"
  if ! status="$(curl -sS -X POST \
    --netrc-file "${netrc_file}" \
    -H 'Content-Type: application/x-www-form-urlencoded' \
    -D "${headers_file}" \
    -o "${response_file}" \
    -w '%{http_code}' \
    --max-time 10 \
    "$@" \
    --data-urlencode "client_id=${smoke_client_id}" \
    --data-urlencode "client_secret@${secret_file}" \
    "${url}" 2>"${error_file}")"; then
    status="${status:-000}"
  fi
  curl_error="$(body_snippet "${error_file}")"
  cache_control="$(response_header "${headers_file}" "Cache-Control")"
  pragma="$(response_header "${headers_file}" "Pragma")"
  www_authenticate="$(response_header "${headers_file}" "WWW-Authenticate")"
  if [[ "${status}" == "401" ]] && \
    jq -e '.error == "invalid_client"' "${response_file}" >/dev/null 2>&1 && \
    oauth_no_store_headers_present "${headers_file}" && \
    basic_invalid_client_challenge_present "${headers_file}"; then
    rm -f "${netrc_file}" "${secret_file}" "${response_file}" "${headers_file}" "${error_file}"
    record_pass "${name}" "$(json_detail url "${url}" httpStatus "${status}" clientID "${smoke_client_id}" expectedError "invalid_client" cacheControl "${cache_control}" pragma "${pragma}" wwwAuthenticate "${www_authenticate}" curlError "${curl_error}")"
    return
  fi
  snippet="$(body_snippet "${response_file}")"
  rm -f "${netrc_file}" "${secret_file}" "${response_file}" "${headers_file}" "${error_file}"
  record_fail "${name} 响应异常，status=${status}" "$(json_detail url "${url}" httpStatus "${status}" expectedStatus "401" expectedError "invalid_client" expectedCacheControl "no-store" expectedPragma "no-cache" expectedWWWAuthenticate "Basic realm=\"StuHelper Identity\"" cacheControl "${cache_control}" pragma "${pragma}" wwwAuthenticate "${www_authenticate}" clientID "${smoke_client_id}" bodySnippet "${snippet}" curlError "${curl_error}")"
}

check_mixed_client_auth_rejections() {
  if [[ -z "${smoke_client_secret}" ]]; then
    log "skipping mixed client authentication checks; set IDENTITY_PUBLIC_SMOKE_CLIENT_SECRET to enable" >&2
    return
  fi

  check_mixed_client_auth_error \
    "Identity token 混用 client authentication 返回 invalid_client" \
    "${identity_issuer}/oauth2/token" \
    --data-urlencode "grant_type=client_credentials" \
    --data-urlencode "scope=${smoke_client_credentials_scope}"
  check_mixed_client_auth_error \
    "Identity introspect 混用 client authentication 返回 invalid_client" \
    "${identity_issuer}/oauth2/introspect" \
    --data-urlencode "token=identity-public-smoke"
  check_mixed_client_auth_error \
    "Identity revoke 混用 client authentication 返回 invalid_client" \
    "${identity_issuer}/oauth2/revoke" \
    --data-urlencode "token=identity-public-smoke"
}

check_client_authenticated_invalid_request() {
  local name="$1"
  local url="$2"
  shift 2
  local netrc_file
  local response_file
  local headers_file
  local error_file
  local status
  local curl_error
  local snippet
  local cache_control
  local pragma
  netrc_file="$(mktemp)"
  response_file="$(mktemp)"
  headers_file="$(mktemp)"
  error_file="$(mktemp)"
  write_smoke_client_netrc "${netrc_file}"
  if ! status="$(curl -sS -X POST \
    --netrc-file "${netrc_file}" \
    -H 'Content-Type: application/x-www-form-urlencoded' \
    -D "${headers_file}" \
    -o "${response_file}" \
    -w '%{http_code}' \
    --max-time 10 \
    "$@" \
    "${url}" 2>"${error_file}")"; then
    status="${status:-000}"
  fi
  curl_error="$(body_snippet "${error_file}")"
  cache_control="$(response_header "${headers_file}" "Cache-Control")"
  pragma="$(response_header "${headers_file}" "Pragma")"
  if [[ "${status}" == "400" ]] && \
    jq -e '.error == "invalid_request"' "${response_file}" >/dev/null 2>&1 && \
    oauth_no_store_headers_present "${headers_file}"; then
    rm -f "${netrc_file}" "${response_file}" "${headers_file}" "${error_file}"
    record_pass "${name}" "$(json_detail url "${url}" httpStatus "${status}" clientID "${smoke_client_id}" expectedError "invalid_request" cacheControl "${cache_control}" pragma "${pragma}" curlError "${curl_error}")"
    return
  fi
  snippet="$(body_snippet "${response_file}")"
  rm -f "${netrc_file}" "${response_file}" "${headers_file}" "${error_file}"
  record_fail "${name} 响应异常，status=${status}" "$(json_detail url "${url}" httpStatus "${status}" expectedStatus "400" expectedError "invalid_request" expectedCacheControl "no-store" expectedPragma "no-cache" cacheControl "${cache_control}" pragma "${pragma}" clientID "${smoke_client_id}" bodySnippet "${snippet}" curlError "${curl_error}")"
}

check_required_parameter_errors_with_client_auth() {
  if [[ -z "${smoke_client_secret}" ]]; then
    log "skipping authenticated required-parameter checks; set IDENTITY_PUBLIC_SMOKE_CLIENT_SECRET to enable" >&2
    return
  fi

  check_client_authenticated_invalid_request \
    "Identity token 已认证但缺少 grant_type 返回 invalid_request" \
    "${identity_issuer}/oauth2/token" \
    --data-urlencode "scope=${smoke_client_credentials_scope}"
  check_client_authenticated_invalid_request \
    "Identity introspect 已认证但空 token 返回 invalid_request" \
    "${identity_issuer}/oauth2/introspect" \
    --data-urlencode "token="
  check_client_authenticated_invalid_request \
    "Identity revoke 已认证但空 token 返回 invalid_request" \
    "${identity_issuer}/oauth2/revoke" \
    --data-urlencode "token="
}

check_logout_no_session() {
  local method="${1:-GET}"
  local name="${2:-Identity logout 无会话返回 204}"
  local response_file
  local error_file
  local status
  local curl_error
  local snippet
  local url="${identity_issuer}/oauth2/logout"
  response_file="$(mktemp)"
  error_file="$(mktemp)"
  if ! status="$(curl -sS -X "${method}" -o "${response_file}" -w '%{http_code}' --max-time 10 "${url}" 2>"${error_file}")"; then
    status="${status:-000}"
  fi
  curl_error="$(body_snippet "${error_file}")"
  if [[ "${status}" == "204" ]]; then
    rm -f "${response_file}" "${error_file}"
    record_pass "${name}" "$(json_detail url "${url}" method "${method}" httpStatus "${status}" curlError "${curl_error}")"
    return
  fi
  snippet="$(body_snippet "${response_file}")"
  rm -f "${response_file}" "${error_file}"
  record_fail "${name} 响应异常，status=${status}" "$(json_detail url "${url}" method "${method}" httpStatus "${status}" expectedStatus "204" bodySnippet "${snippet}" curlError "${curl_error}")"
}

check_logout_post_rejected() {
  local name="$1"
  local url="$2"
  local body="${3:-}"
  local content_type="${4:-}"
  local response_file
  local error_file
  local status
  local curl_error
  local snippet
  response_file="$(mktemp)"
  error_file="$(mktemp)"

  local curl_args=(-sS -X POST -o "${response_file}" -w '%{http_code}' --max-time 10)
  if [[ -n "${content_type}" ]]; then
    curl_args+=(-H "Content-Type: ${content_type}" --data "${body}")
  fi
  curl_args+=("${url}")

  if ! status="$(curl "${curl_args[@]}" 2>"${error_file}")"; then
    status="${status:-000}"
  fi
  curl_error="$(body_snippet "${error_file}")"
  snippet="$(body_snippet "${response_file}")"
  if [[ "${status}" == "400" ]] && grep -qi 'invalid logout request' "${response_file}"; then
    rm -f "${response_file}" "${error_file}"
    record_pass "${name}" "$(json_detail url "${url}" method "POST" httpStatus "${status}" bodySnippet "${snippet}" curlError "${curl_error}")"
    return
  fi
  rm -f "${response_file}" "${error_file}"
  record_fail "${name} 响应异常，status=${status}" "$(json_detail url "${url}" method "POST" httpStatus "${status}" expectedStatus "400" expectedBody "invalid logout request" bodySnippet "${snippet}" curlError "${curl_error}")"
}

check_logout_rejects_invalid_id_token_hint_without_redirect() {
  local name="${1:-Identity logout GET 无回跳拒绝非 ID Token hint}"
  local response_file
  local error_file
  local status
  local curl_error
  local snippet
  local url="${identity_issuer}/oauth2/logout"
  response_file="$(mktemp)"
  error_file="$(mktemp)"
  if ! status="$(curl -sS -G \
    -o "${response_file}" \
    -w '%{http_code}' \
    --max-time 10 \
    --data-urlencode "id_token_hint=identity-public-smoke-access-token" \
    "${url}" 2>"${error_file}")"; then
    status="${status:-000}"
  fi
  curl_error="$(body_snippet "${error_file}")"
  snippet="$(body_snippet "${response_file}")"
  if [[ "${status}" == "400" ]] && grep -qi 'invalid logout request' "${response_file}"; then
    rm -f "${response_file}" "${error_file}"
    record_pass "${name}" "$(json_detail url "${url}" method "GET" httpStatus "${status}" bodySnippet "${snippet}" curlError "${curl_error}")"
    return
  fi
  rm -f "${response_file}" "${error_file}"
  record_fail "${name} 响应异常，status=${status}" "$(json_detail url "${url}" method "GET" httpStatus "${status}" expectedStatus "400" expectedBody "invalid logout request" bodySnippet "${snippet}" curlError "${curl_error}")"
}

check_userinfo_missing_bearer() {
  local method="${1:-GET}"
  local name="${2:-Identity UserInfo 无 bearer 返回 invalid_token}"
  local response_file
  local headers_file
  local error_file
  local status
  local curl_error
  local cache_control
  local pragma
  local www_authenticate
  response_file="$(mktemp)"
  headers_file="$(mktemp)"
  error_file="$(mktemp)"
  if ! status="$(curl -sS -X "${method}" -D "${headers_file}" -o "${response_file}" -w '%{http_code}' --max-time 10 \
    "${identity_issuer}/oidc/userinfo" 2>"${error_file}")"; then
    status="000"
  fi
  curl_error="$(body_snippet "${error_file}")"
  cache_control="$(response_header "${headers_file}" "Cache-Control")"
  pragma="$(response_header "${headers_file}" "Pragma")"
  www_authenticate="$(response_header "${headers_file}" "WWW-Authenticate")"
  if [[ "${status}" == "401" ]] && \
    jq -e '.error == "invalid_token"' "${response_file}" >/dev/null 2>&1 && \
    oauth_no_store_headers_present "${headers_file}" && \
    bearer_invalid_token_challenge_present "${headers_file}"; then
    rm -f "${response_file}" "${headers_file}" "${error_file}"
    record_pass "${name}" "$(json_detail url "${identity_issuer}/oidc/userinfo" method "${method}" httpStatus "${status}" cacheControl "${cache_control}" pragma "${pragma}" wwwAuthenticate "${www_authenticate}" curlError "${curl_error}")"
    return
  fi
  local snippet
  snippet="$(body_snippet "${response_file}")"
  rm -f "${response_file}" "${headers_file}" "${error_file}"
  record_fail "${name} 响应异常，status=${status}" "$(json_detail url "${identity_issuer}/oidc/userinfo" method "${method}" httpStatus "${status}" expectedCacheControl "no-store" expectedPragma "no-cache" expectedWWWAuthenticate "Bearer realm=\"StuHelper Identity\", error=\"invalid_token\"" cacheControl "${cache_control}" pragma "${pragma}" wwwAuthenticate "${www_authenticate}" bodySnippet "${snippet}" curlError "${curl_error}")"
}

check_userinfo_rejects_non_header_token_source() {
  local mode="${1:-query}"
  local name="$2"
  local response_file
  local headers_file
  local error_file
  local status
  local curl_error
  local cache_control
  local pragma
  local www_authenticate
  local url="${identity_issuer}/oidc/userinfo"
  response_file="$(mktemp)"
  headers_file="$(mktemp)"
  error_file="$(mktemp)"

  if [[ "${mode}" == "body" ]]; then
    if ! status="$(curl -sS -X POST -D "${headers_file}" -o "${response_file}" -w '%{http_code}' --max-time 10 \
      -H "Content-Type: application/x-www-form-urlencoded" \
      --data-urlencode "access_token=identity-public-smoke-body-token" \
      "${url}" 2>"${error_file}")"; then
      status="000"
    fi
  else
    url="${url}?access_token=identity-public-smoke-query-token"
    if ! status="$(curl -sS -D "${headers_file}" -o "${response_file}" -w '%{http_code}' --max-time 10 \
      "${url}" 2>"${error_file}")"; then
      status="000"
    fi
  fi

  curl_error="$(body_snippet "${error_file}")"
  cache_control="$(response_header "${headers_file}" "Cache-Control")"
  pragma="$(response_header "${headers_file}" "Pragma")"
  www_authenticate="$(response_header "${headers_file}" "WWW-Authenticate")"
  if [[ "${status}" == "401" ]] && \
    jq -e '.error == "invalid_token"' "${response_file}" >/dev/null 2>&1 && \
    oauth_no_store_headers_present "${headers_file}" && \
    bearer_invalid_token_challenge_present "${headers_file}"; then
    rm -f "${response_file}" "${headers_file}" "${error_file}"
    record_pass "${name}" "$(json_detail url "${url}" method "${mode}" httpStatus "${status}" cacheControl "${cache_control}" pragma "${pragma}" wwwAuthenticate "${www_authenticate}" curlError "${curl_error}")"
    return
  fi
  local snippet
  snippet="$(body_snippet "${response_file}")"
  rm -f "${response_file}" "${headers_file}" "${error_file}"
  record_fail "${name} 响应异常，status=${status}" "$(json_detail url "${url}" method "${mode}" httpStatus "${status}" expectedCacheControl "no-store" expectedPragma "no-cache" expectedWWWAuthenticate "Bearer realm=\"StuHelper Identity\", error=\"invalid_token\"" cacheControl "${cache_control}" pragma "${pragma}" wwwAuthenticate "${www_authenticate}" bodySnippet "${snippet}" curlError "${curl_error}")"
}

check_casdoor_discovery() {
  local metadata="$1"
  if printf '%s\n' "${metadata}" | jq -e \
    --arg issuer "${casdoor_issuer}" '
      .issuer == $issuer
      and (.authorization_endpoint | type == "string" and length > 0)
      and (.token_endpoint | type == "string" and length > 0)
      and (.jwks_uri | type == "string" and length > 0)
    ' >/dev/null; then
    record_pass "Casdoor discovery 元数据" "$(json_detail issuer "${casdoor_issuer}")"
  else
    record_fail "Casdoor discovery 元数据不匹配 ${casdoor_issuer}" "$(json_detail expectedIssuer "${casdoor_issuer}")"
  fi
}

printf '━━━ Public Identity Smoke ━━━\n'
tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT
check_jsonl="${tmpdir}/checks.jsonl"
: >"${check_jsonl}"

wait_for_http "Web public health" "${web_public_url}/health/ready" || true

identity_metadata_file="${tmpdir}/identity-openid-configuration.json"
if fetch_json "Identity discovery" "${identity_issuer}/.well-known/openid-configuration" "${identity_metadata_file}"; then
  check_identity_discovery "${identity_metadata_file}"
fi

identity_oauth_metadata_file="${tmpdir}/identity-oauth-authorization-server.json"
if fetch_json "Identity OAuth authorization server metadata" "${identity_issuer}/.well-known/oauth-authorization-server" "${identity_oauth_metadata_file}"; then
  check_identity_discovery \
    "${identity_oauth_metadata_file}" \
    "Identity OAuth authorization server 元数据" \
    "Identity OAuth authorization server 元数据不匹配 ${identity_issuer}"
fi

jwks_file="${tmpdir}/identity-jwks.json"
if fetch_json "Identity JWKS" "${identity_issuer}/.well-known/jwks.json" "${jwks_file}"; then
  check_jwks "Identity" "$(cat "${jwks_file}")"
fi

check_identity_web_asset "Identity favicon 留在 id 入口" "${identity_issuer}/favicon.ico" "" 128
check_identity_web_asset "Identity web manifest 留在 id 入口" "${identity_issuer}/site.webmanifest" '"name": "StuHelper"' 32
check_authorize_redirect
check_authorize_reauth_redirect
check_registered_prompt_none_login_required
check_oauth_post_error "Identity token 缺少 grant_type 返回 invalid_request" "${identity_issuer}/oauth2/token" "" "400" "invalid_request"
check_oauth_post_error "Identity introspect 缺少 client 返回 invalid_client" "${identity_issuer}/oauth2/introspect" "token=identity-public-smoke" "401" "invalid_client"
check_oauth_post_error "Identity revoke 缺少 client 返回 invalid_client" "${identity_issuer}/oauth2/revoke" "token=identity-public-smoke" "401" "invalid_client"
check_oauth_post_invalid_content_type "Identity token JSON Content-Type 返回 invalid_request" "${identity_issuer}/oauth2/token" "application/json" '{"grant_type":"client_credentials"}'
check_oauth_post_invalid_content_type "Identity introspect text/plain Content-Type 返回 invalid_request" "${identity_issuer}/oauth2/introspect" "text/plain" "token=identity-public-smoke"
check_oauth_post_invalid_content_type "Identity revoke 缺少 Content-Type 返回 invalid_request" "${identity_issuer}/oauth2/revoke" "" "token=identity-public-smoke"
check_oauth_post_query_rejected "Identity token URL query 参数返回 invalid_request" "${identity_issuer}/oauth2/token?grant_type=client_credentials" ""
check_oauth_post_query_rejected "Identity introspect URL query 参数返回 invalid_request" "${identity_issuer}/oauth2/introspect?token=identity-public-smoke" "token=identity-public-smoke"
check_oauth_post_query_rejected "Identity revoke URL query 参数返回 invalid_request" "${identity_issuer}/oauth2/revoke?token=identity-public-smoke" "token=identity-public-smoke"
check_mixed_client_auth_rejections
check_required_parameter_errors_with_client_auth
check_logout_no_session "GET" "Identity logout GET 无会话返回 204"
check_logout_no_session "POST" "Identity logout POST 无会话返回 204"
check_logout_rejects_invalid_id_token_hint_without_redirect "Identity logout GET 无回跳拒绝非 ID Token hint"
check_logout_post_rejected "Identity logout POST URL query 参数返回 invalid logout request" "${identity_issuer}/oauth2/logout?client_id=identity-public-smoke"
check_logout_post_rejected "Identity logout POST JSON body 返回 invalid logout request" "${identity_issuer}/oauth2/logout" '{"client_id":"identity-public-smoke"}' "application/json"
check_userinfo_missing_bearer "GET" "Identity UserInfo GET 无 bearer 返回 invalid_token"
check_userinfo_missing_bearer "POST" "Identity UserInfo POST 无 bearer 返回 invalid_token"
check_userinfo_rejects_non_header_token_source "query" "Identity UserInfo URL query token 返回 invalid_token"
check_userinfo_rejects_non_header_token_source "body" "Identity UserInfo body token 返回 invalid_token"
check_client_credentials_grant

if [[ "${casdoor_upstream_enabled}" == "true" ]]; then
  casdoor_metadata_file="${tmpdir}/casdoor-openid-configuration.json"
  if fetch_json "Casdoor upstream discovery" "${casdoor_issuer}/.well-known/openid-configuration" "${casdoor_metadata_file}"; then
    check_casdoor_discovery "$(cat "${casdoor_metadata_file}")"
    discovered_casdoor_jwks_url="$(jq -r '.jwks_uri // empty' "${casdoor_metadata_file}")"
    if [[ -n "${discovered_casdoor_jwks_url}" ]]; then
      casdoor_jwks_url="${discovered_casdoor_jwks_url}"
    fi
    casdoor_jwks_file="${tmpdir}/casdoor-jwks.json"
    if fetch_json "Casdoor upstream JWKS" "${casdoor_jwks_url}" "${casdoor_jwks_file}"; then
      check_jwks "Casdoor upstream" "$(cat "${casdoor_jwks_file}")"
    fi
  fi
else
  log "skipping public Casdoor upstream smoke; set IDENTITY_PUBLIC_SMOKE_CASDOOR_UPSTREAM_ENABLED=true to enable" >&2
fi

printf '━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n'
printf '结果: ✅ %s 通过  ❌ %s 失败\n' "${pass}" "${fail}"

smoke_passed="false"
if (( fail == 0 )); then
  smoke_passed="true"
fi
write_evidence "${smoke_passed}"

if (( fail > 0 )); then
  die "public identity smoke failed"
fi

log "public identity smoke passed"
