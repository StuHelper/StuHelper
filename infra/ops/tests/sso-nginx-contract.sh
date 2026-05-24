#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
SSO_NGINX_FILE="${REPO_ROOT}/infra/nginx/baota-casdoor-sso.conf"

fail() {
  echo "[sso-nginx-contract][error] $*" >&2
  exit 1
}

assert_contains() {
  local file="$1"
  local pattern="$2"
  if ! grep -Eq "${pattern}" "${file}"; then
    fail "expected ${file} to contain pattern: ${pattern}"
  fi
}

[[ -f "${SSO_NGINX_FILE}" ]] || fail "missing SSO Nginx template: ${SSO_NGINX_FILE}"

assert_contains "${SSO_NGINX_FILE}" 'server_name sso\.stuhelper\.com;'
assert_contains "${SSO_NGINX_FILE}" 'return 301 https://sso\.stuhelper\.com\$request_uri;'
assert_contains "${SSO_NGINX_FILE}" 'location = /\.well-known/openid-configuration \{'
assert_contains "${SSO_NGINX_FILE}" 'location = /\.well-known/jwks \{'
assert_contains "${SSO_NGINX_FILE}" 'location \^~ /\.well-known/ \{'
assert_contains "${SSO_NGINX_FILE}" 'location \^~ /api/ \{'
assert_contains "${SSO_NGINX_FILE}" 'proxy_pass http://127\.0\.0\.1:8087;'
assert_contains "${SSO_NGINX_FILE}" 'NGINX_PUBLIC_INGRESS_CASDOOR_UPSTREAM'
assert_contains "${SSO_NGINX_FILE}" 'X-Forwarded-Proto https'
assert_contains "${SSO_NGINX_FILE}" 'X-Forwarded-Host \$host'
assert_contains "${SSO_NGINX_FILE}" 'OIDC discovery returns the Casdoor SPA'

if grep -Eq 'try_files|root[[:space:]]+' "${SSO_NGINX_FILE}"; then
  fail "SSO Nginx template must not serve /.well-known from a static root"
fi

echo "[sso-nginx-contract] all assertions passed"
