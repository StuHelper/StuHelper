#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
PREFLIGHT_SCRIPT="${REPO_ROOT}/infra/ops/nginx-public-ingress-preflight.sh"
MAIN_NGINX_FILE="${REPO_ROOT}/infra/nginx/baota-stuhelper.conf"
SSO_NGINX_FILE="${REPO_ROOT}/infra/nginx/baota-casdoor-sso.conf"
SSO_WELL_KNOWN_EXTENSION_FILE="${REPO_ROOT}/infra/nginx/baota-casdoor-sso-well-known-extension.conf"

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

assert_file_contains "${PREFLIGHT_SCRIPT}" 'join\.stuhelper\.com'
assert_file_contains "${SSO_WELL_KNOWN_EXTENSION_FILE}" 'location = /.well-known/openid-configuration'
assert_file_contains "${SSO_WELL_KNOWN_EXTENSION_FILE}" 'location = /.well-known/jwks'

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

missing_main_verify_reject="${tmpdir}/missing-main-verify-reject.conf"
awk '
  /^    location = \/verify \{$/ { skip=1; next }
  /^    location \^~ \/verify\/ \{$/ { skip=1; next }
  skip && /^    }$/ { skip=0; next }
  !skip { print }
' "${MAIN_NGINX_FILE}" >"${missing_main_verify_reject}"

missing_main_start_reject="${tmpdir}/missing-main-start-reject.conf"
awk '
  /^    location = \/start \{$/ { skip=1; next }
  /^    location \^~ \/start\/ \{$/ { skip=1; next }
  skip && /^    }$/ { skip=0; next }
  !skip { print }
' "${MAIN_NGINX_FILE}" >"${missing_main_start_reject}"

missing_main_grafana_proxy="${tmpdir}/missing-main-grafana-proxy.conf"
awk '
  /^    location \^~ \/admin\/observability\/ \{$/ { skip=1; next }
  skip && /^    }$/ { skip=0; next }
  !skip { print }
' "${MAIN_NGINX_FILE}" >"${missing_main_grafana_proxy}"

missing_join_verify_proxy="${tmpdir}/missing-join-verify-proxy.conf"
awk '
  /^    server_name join\.stuhelper\.com;$/ { in_join=1 }
  in_join && /^server \{$/ { in_join=0 }
  in_join && /^    location \^~ \/verify\/ \{$/ { skip=1; next }
  skip && /^    }$/ { skip=0; next }
  !skip { print }
' "${MAIN_NGINX_FILE}" >"${missing_join_verify_proxy}"

missing_join_start_proxy="${tmpdir}/missing-join-start-proxy.conf"
awk '
  /^    server_name join\.stuhelper\.com;$/ { in_join=1 }
  in_join && /^server \{$/ { in_join=0 }
  in_join && /^    location = \/start \{$/ { skip=1; next }
  in_join && /^    location \^~ \/start\/ \{$/ { skip=1; next }
  skip && /^    }$/ { skip=0; next }
  !skip { print }
' "${MAIN_NGINX_FILE}" >"${missing_join_start_proxy}"

join_root_proxies_web="${tmpdir}/join-root-proxies-web.conf"
python3 - "${MAIN_NGINX_FILE}" "${join_root_proxies_web}" <<'PY'
from pathlib import Path
import sys

lines = Path(sys.argv[1]).read_text(encoding="utf-8").splitlines(keepends=True)

def block_ranges():
    index = 0
    while index < len(lines):
        if lines[index].strip() != "server {":
            index += 1
            continue
        start = index
        depth = 0
        while index < len(lines):
            depth += lines[index].count("{") - lines[index].count("}")
            if depth == 0:
                yield start, index + 1
                break
            index += 1
        index += 1

for start, end in block_ranges():
    block = lines[start:end]
    if not any(line.strip() == "server_name join.stuhelper.com;" for line in block):
        continue
    for relative, line in enumerate(block):
        if line.strip() != "location / {":
            continue
        root_start = start + relative
        depth = 0
        root_end = root_start
        while root_end < end:
            depth += lines[root_end].count("{") - lines[root_end].count("}")
            root_end += 1
            if depth == 0:
                break
        replacement = [
            "    location / {\n",
            "        proxy_pass http://127.0.0.1:18000;\n",
            "        proxy_http_version 1.1;\n",
            "    }\n",
        ]
        Path(sys.argv[2]).write_text("".join(lines[:root_start] + replacement + lines[root_end:]), encoding="utf-8")
        raise SystemExit(0)
raise SystemExit("missing join root location")
PY

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

baota_sso_static_well_known_fixed="${tmpdir}/baota-sso-static-well-known-fixed.conf"
cat >"${baota_sso_static_well_known_fixed}" <<'NGINX'
server {
    listen 443 ssl;
    http2 on;
    server_name sso.stuhelper.com;
    root /www/dk_project/wwwroot/sso.stuhelper.com;
    ssl_certificate /tmp/fullchain.pem;
    ssl_certificate_key /tmp/privkey.pem;

    location = /.well-known/openid-configuration {
      proxy_pass http://127.0.0.1:8087;
      proxy_set_header Host $http_host;
      proxy_set_header X-Real-IP $remote_addr;
      proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
      proxy_set_header X-Forwarded-Proto $scheme;
      proxy_set_header X-Forwarded-Host $host;
      proxy_http_version 1.1;
    }

    location = /.well-known/jwks {
      proxy_pass http://127.0.0.1:8087;
      proxy_set_header Host $http_host;
      proxy_set_header X-Real-IP $remote_addr;
      proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
      proxy_set_header X-Forwarded-Proto $scheme;
      proxy_set_header X-Forwarded-Host $host;
      proxy_http_version 1.1;
    }

    location ^~ / {
      proxy_pass http://127.0.0.1:8087;
      proxy_set_header Host $http_host;
      proxy_set_header X-Forwarded-Proto $scheme;
      proxy_set_header X-Forwarded-Host $host;
    }

    location ^~ /api/ {
      proxy_pass http://127.0.0.1:8087;
      proxy_set_header Host $http_host;
      proxy_set_header X-Forwarded-Proto $scheme;
      proxy_set_header X-Forwarded-Host $host;
    }

    location /.well-known {
      allow all;
    }
}
NGINX

baota_sso_static_well_known_with_extension="${tmpdir}/baota-sso-static-well-known-with-extension.conf"
{
  cat <<'NGINX'
server {
    listen 443 ssl;
    http2 on;
    server_name sso.stuhelper.com;
    root /www/dk_project/wwwroot/sso.stuhelper.com;
    ssl_certificate /tmp/fullchain.pem;
    ssl_certificate_key /tmp/privkey.pem;

    # Baota keeps this include even when it rewrites the main vhost.
    include /www/server/panel/vhost/nginx/extension/sso.stuhelper.com/*.conf;

NGINX
  sed 's/^/    /' "${SSO_WELL_KNOWN_EXTENSION_FILE}"
  cat <<'NGINX'

    location ^~ / {
      proxy_pass http://127.0.0.1:8087;
      proxy_set_header Host $http_host;
      proxy_set_header X-Forwarded-Proto $scheme;
      proxy_set_header X-Forwarded-Host $host;
    }

    location ^~ /api/ {
      proxy_pass http://127.0.0.1:8087;
      proxy_set_header Host $http_host;
      proxy_set_header X-Forwarded-Proto $scheme;
      proxy_set_header X-Forwarded-Host $host;
    }

    location /.well-known {
      allow all;
    }
}
NGINX
} >"${baota_sso_static_well_known_with_extension}"

baota_sso_static_well_known_missing_jwks="${tmpdir}/baota-sso-static-well-known-missing-jwks.conf"
awk '
  /^    location = \/.well-known\/jwks \{/ { skip=1; next }
  skip && /^    }$/ { skip=0; next }
  !skip { print }
' "${baota_sso_static_well_known_fixed}" >"${baota_sso_static_well_known_missing_jwks}"

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
run_preflight_pass "sso" "${baota_sso_static_well_known_fixed}" "${tmpdir}" "baota-sso-static-well-known-fixed"
run_preflight_pass "sso" "${baota_sso_static_well_known_with_extension}" "${tmpdir}" "baota-sso-static-well-known-with-extension"
run_preflight_fail "stuhelper" "${missing_main_verify_reject}" "${tmpdir}" "missing-main-verify-reject" 'stuhelper\.com: no HTTPS server block satisfies the ingress contract: stuhelper\.com: missing location = /verify'
run_preflight_fail "stuhelper" "${missing_main_start_reject}" "${tmpdir}" "missing-main-start-reject" 'stuhelper\.com: no HTTPS server block satisfies the ingress contract: stuhelper\.com: missing location = /start'
run_preflight_fail "stuhelper" "${missing_main_grafana_proxy}" "${tmpdir}" "missing-main-grafana-proxy" 'stuhelper\.com: no HTTPS server block satisfies the ingress contract: stuhelper\.com: missing location \^~ /admin/observability/'
run_preflight_fail "stuhelper" "${missing_join_verify_proxy}" "${tmpdir}" "missing-join-verify-proxy" 'join\.stuhelper\.com: no HTTPS server block satisfies the ingress contract: join\.stuhelper\.com: missing location \^~ /verify/'
run_preflight_fail "stuhelper" "${missing_join_start_proxy}" "${tmpdir}" "missing-join-start-proxy" 'join\.stuhelper\.com: no HTTPS server block satisfies the ingress contract: join\.stuhelper\.com: missing location = /start'
run_preflight_fail "stuhelper" "${join_root_proxies_web}" "${tmpdir}" "join-root-proxies-web" 'join\.stuhelper\.com: no HTTPS server block satisfies the ingress contract: join\.stuhelper\.com: location / must return 404'
run_preflight_fail "sso" "${bad_sso_static_root}" "${tmpdir}" "bad-sso-static-root" 'requires exact openid-configuration and jwks'
run_preflight_fail "sso" "${baota_sso_static_well_known_missing_jwks}" "${tmpdir}" "baota-sso-static-well-known-missing-jwks" 'requires exact openid-configuration and jwks'
run_preflight_fail "unknown" "${combined_good}" "${tmpdir}" "unknown-profile" 'unknown NGINX_PUBLIC_INGRESS_PROFILE'

if ! PUBLIC_INGRESS_CONFIG_PREFLIGHT_ENABLED=false \
  NGINX_PUBLIC_INGRESS_CONFIG_FILE="${tmpdir}/does-not-exist.conf" \
  bash "${PREFLIGHT_SCRIPT}" >"${tmpdir}/skip.stdout" 2>"${tmpdir}/skip.stderr"; then
  fail "expected disabled Nginx ingress preflight to skip even when config file is missing"
fi
assert_file_contains "${tmpdir}/skip.stderr" 'public Nginx ingress config preflight skipped'

echo "[nginx-public-ingress-preflight-contract] all assertions passed"
