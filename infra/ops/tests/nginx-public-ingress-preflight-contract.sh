#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
PREFLIGHT_SCRIPT="${REPO_ROOT}/infra/ops/nginx-public-ingress-preflight.sh"
MAIN_NGINX_FILE="${REPO_ROOT}/infra/nginx/baota-stuhelper.conf"
SSO_NGINX_FILE="${REPO_ROOT}/infra/nginx/baota-casdoor-sso.conf"

fail() {
  echo "[nginx-public-ingress-preflight-contract][error] $*" >&2
  exit 1
}

assert_file_contains() {
  local file="$1"
  local pattern="$2"
  if ! grep -Eq "${pattern}" "${file}"; then
    fail "expected ${file} to contain pattern: ${pattern}"
  fi
}

run_preflight_pass() {
  local profile="$1"
  local config_file="$2"
  local tmpdir="$3"
  local label="$4"

  if ! NGINX_PUBLIC_INGRESS_PROFILE="${profile}" \
    NGINX_PUBLIC_INGRESS_CONFIG_FILE="${config_file}" \
    bash "${PREFLIGHT_SCRIPT}" >"${tmpdir}/${label}.stdout" 2>"${tmpdir}/${label}.stderr"; then
    cat "${tmpdir}/${label}.stdout" >&2 || true
    cat "${tmpdir}/${label}.stderr" >&2 || true
    fail "expected Nginx ingress preflight to pass for ${label}"
  fi
  assert_file_contains "${tmpdir}/${label}.stdout" 'public Nginx ingress config preflight passed'
}

run_preflight_fail() {
  local profile="$1"
  local config_file="$2"
  local tmpdir="$3"
  local label="$4"
  local expected_error="$5"

  if NGINX_PUBLIC_INGRESS_PROFILE="${profile}" \
    NGINX_PUBLIC_INGRESS_CONFIG_FILE="${config_file}" \
    bash "${PREFLIGHT_SCRIPT}" >"${tmpdir}/${label}.stdout" 2>"${tmpdir}/${label}.stderr"; then
    fail "expected Nginx ingress preflight to fail for ${label}"
  fi
  assert_file_contains "${tmpdir}/${label}.stderr" "${expected_error}"
}

tmpdir="$(mktemp -d)"
cleanup() {
  rm -rf "${tmpdir}"
}
trap cleanup EXIT

combined_good="${tmpdir}/combined-good.conf"
cat "${MAIN_NGINX_FILE}" "${SSO_NGINX_FILE}" >"${combined_good}"

baota_dump_with_json_logs="${tmpdir}/baota-dump-with-json-logs.conf"
{
  cat <<'NGINX'
http {
    baota_log_payload {
        "server_addr": "$server_addr",
        "request_uri": "$request_uri",
        "nested": {
            "host": "$host"
        }
    };
NGINX
  cat "${MAIN_NGINX_FILE}" "${SSO_NGINX_FILE}"
  cat <<'NGINX'
}
NGINX
} >"${baota_dump_with_json_logs}"

missing_id="${tmpdir}/missing-id.conf"
cat >"${missing_id}" <<'NGINX'
server {
    listen 443 ssl http2;
    server_name www.stuhelper.com;
    ssl_certificate /tmp/fullchain.pem;
    ssl_certificate_key /tmp/privkey.pem;
    return 301 https://stuhelper.com$request_uri;
}

server {
    listen 443 ssl http2;
    server_name stuhelper.com;
    ssl_certificate /tmp/fullchain.pem;
    ssl_certificate_key /tmp/privkey.pem;
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-Proto https;
    proxy_set_header X-Forwarded-Host $host;
    location ^~ /api/ { proxy_pass http://127.0.0.1:18080; }
    location ^~ /health/ { proxy_pass http://127.0.0.1:18080; }
    location ^~ /admin/ { proxy_pass http://127.0.0.1:18001; }
    location / { proxy_pass http://127.0.0.1:18000; }
}
NGINX

missing_id_redirect_cache_headers="${tmpdir}/missing-id-redirect-cache-headers.conf"
sed '/add_header Cache-Control "no-store, no-cache, must-revalidate, private" always;/d' \
  "${MAIN_NGINX_FILE}" >"${missing_id_redirect_cache_headers}"

bad_sso_static_root="${tmpdir}/bad-sso-static-root.conf"
cat >"${bad_sso_static_root}" <<'NGINX'
server {
    listen 443 ssl http2;
    server_name sso.stuhelper.com;
    ssl_certificate /tmp/fullchain.pem;
    ssl_certificate_key /tmp/privkey.pem;
    root /www/wwwroot/sso.stuhelper.com;
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-Proto https;
    proxy_set_header X-Forwarded-Host $host;
    location / { try_files $uri /index.html; }
    location ^~ /api/ { proxy_pass http://127.0.0.1:8087; }
}
NGINX

sso_custom_upstream="${tmpdir}/sso-custom-upstream.conf"
sed 's/127[.]0[.]0[.]1:8087/127.0.0.1:8085/g' "${SSO_NGINX_FILE}" >"${sso_custom_upstream}"

run_preflight_pass "stuhelper" "${MAIN_NGINX_FILE}" "${tmpdir}" "main-template"
run_preflight_pass "sso" "${SSO_NGINX_FILE}" "${tmpdir}" "sso-template"
if ! NGINX_PUBLIC_INGRESS_PROFILE="sso" \
  NGINX_PUBLIC_INGRESS_CONFIG_FILE="${sso_custom_upstream}" \
  NGINX_PUBLIC_INGRESS_CASDOOR_UPSTREAM="http://127.0.0.1:8085" \
  bash "${PREFLIGHT_SCRIPT}" >"${tmpdir}/sso-custom-upstream.stdout" 2>"${tmpdir}/sso-custom-upstream.stderr"; then
  cat "${tmpdir}/sso-custom-upstream.stdout" >&2 || true
  cat "${tmpdir}/sso-custom-upstream.stderr" >&2 || true
  fail "expected Nginx ingress preflight to pass for custom SSO upstream"
fi
assert_file_contains "${tmpdir}/sso-custom-upstream.stdout" 'public Nginx ingress config preflight passed'
run_preflight_pass "all" "${combined_good}" "${tmpdir}" "combined-template"
run_preflight_pass "all" "${baota_dump_with_json_logs}" "${tmpdir}" "baota-json-log-dump"
run_preflight_fail "stuhelper" "${missing_id}" "${tmpdir}" "missing-id" 'id\.stuhelper\.com: missing HTTPS server block'
run_preflight_fail "stuhelper" "${missing_id_redirect_cache_headers}" "${tmpdir}" "missing-id-redirect-cache-headers" 'location = / must add_header Cache-Control'
run_preflight_fail "sso" "${bad_sso_static_root}" "${tmpdir}" "bad-sso-static-root" 'root or try_files'
run_preflight_fail "unknown" "${combined_good}" "${tmpdir}" "unknown-profile" 'unknown NGINX_PUBLIC_INGRESS_PROFILE'

if ! PUBLIC_INGRESS_CONFIG_PREFLIGHT_ENABLED=false \
  NGINX_PUBLIC_INGRESS_CONFIG_FILE="${tmpdir}/does-not-exist.conf" \
  bash "${PREFLIGHT_SCRIPT}" >"${tmpdir}/skip.stdout" 2>"${tmpdir}/skip.stderr"; then
  fail "expected disabled Nginx ingress preflight to skip even when config file is missing"
fi
assert_file_contains "${tmpdir}/skip.stderr" 'public Nginx ingress config preflight skipped'

echo "[nginx-public-ingress-preflight-contract] all assertions passed"
