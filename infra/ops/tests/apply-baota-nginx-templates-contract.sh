#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
APPLY_SCRIPT="${REPO_ROOT}/infra/ops/apply-baota-nginx-templates.sh"
MAIN_NGINX_FILE="${REPO_ROOT}/infra/nginx/baota-stuhelper.conf"
SSO_NGINX_FILE="${REPO_ROOT}/infra/nginx/baota-casdoor-sso.conf"
SSO_WELL_KNOWN_EXTENSION_FILE="${REPO_ROOT}/infra/nginx/baota-casdoor-sso-well-known-extension.conf"
CONNECTOR_NGINX_TEMPLATE="${REPO_ROOT}/infra/nginx/baota-campus-connector-stream.conf.template"
CONNECTOR_RENDERER="${REPO_ROOT}/infra/ops/render-campus-connector-nginx-stream.py"

fail() {
  echo "[apply-baota-nginx-templates-contract][error] $*" >&2
  exit 1
}

assert_contains() {
  local file="$1"
  local pattern="$2"
  if ! grep -Eq -- "${pattern}" "${file}"; then
    fail "expected ${file} to contain pattern: ${pattern}"
  fi
}

assert_not_contains() {
  local file="$1"
  local pattern="$2"
  if grep -Eq -- "${pattern}" "${file}"; then
    fail "expected ${file} not to contain pattern: ${pattern}"
  fi
}

[[ -f "${APPLY_SCRIPT}" ]] || fail "missing apply script: ${APPLY_SCRIPT}"

assert_contains "${APPLY_SCRIPT}" 'baota-stuhelper\.conf'
assert_contains "${APPLY_SCRIPT}" 'baota-casdoor-sso\.conf'
assert_contains "${APPLY_SCRIPT}" 'baota-casdoor-sso-well-known-extension\.conf'
assert_contains "${APPLY_SCRIPT}" 'baota-campus-connector-stream\.conf\.template'
assert_contains "${APPLY_SCRIPT}" '/www/server/panel/vhost/nginx/stuhelper\.com\.conf'
assert_contains "${APPLY_SCRIPT}" '/www/server/panel/vhost/nginx/sso\.stuhelper\.com\.conf'
assert_contains "${APPLY_SCRIPT}" '/www/server/panel/vhost/nginx/extension/sso\.stuhelper\.com/stuhelper-sso-well-known\.conf'
assert_contains "${APPLY_SCRIPT}" '/www/server/panel/vhost/nginx/tcp/connector\.stuhelper\.com\.conf'
assert_contains "${APPLY_SCRIPT}" '--apply'
assert_contains "${APPLY_SCRIPT}" 'dry-run'
assert_contains "${APPLY_SCRIPT}" 'date -u \+%Y%m%dT%H%M%SZ'
assert_contains "${APPLY_SCRIPT}" '\.bak\.\$\{timestamp\}'
assert_contains "${APPLY_SCRIPT}" 'install -m 0644'
assert_contains "${APPLY_SCRIPT}" 'nginx -t'
assert_contains "${APPLY_SCRIPT}" 'nginx -s reload'
assert_contains "${APPLY_SCRIPT}" 'nginx-public-ingress-preflight\.sh'
assert_contains "${APPLY_SCRIPT}" 'NGINX_PUBLIC_INGRESS_NGINX_BIN'
assert_contains "${APPLY_SCRIPT}" 'NGINX_PUBLIC_INGRESS_CONNECTOR_CONFIG_FILE'

[[ -f "${CONNECTOR_NGINX_TEMPLATE}" ]] || fail "missing Connector Nginx stream template"
[[ -f "${CONNECTOR_RENDERER}" ]] || fail "missing Connector Nginx stream renderer"
assert_contains "${CONNECTOR_NGINX_TEMPLATE}" '^server \{$'
assert_contains "${CONNECTOR_NGINX_TEMPLATE}" 'listen __PUBLIC_PORT__;'
assert_contains "${CONNECTOR_NGINX_TEMPLATE}" 'proxy_pass 127\.0\.0\.1:__UPSTREAM_PORT__;'
assert_contains "${CONNECTOR_NGINX_TEMPLATE}" '__ALLOW_DIRECTIVES__'
assert_contains "${CONNECTOR_NGINX_TEMPLATE}" 'deny all;'
assert_not_contains "${CONNECTOR_NGINX_TEMPLATE}" 'listen .*ssl|ssl_certificate|ssl_certificate_key|proxy_ssl|http://|https://'

nginx_test_line="$(grep -n '"${nginx_bin}" -t' "${APPLY_SCRIPT}" | head -n1 | cut -d: -f1)"
nginx_reload_line="$(grep -n '"${nginx_bin}" -s reload' "${APPLY_SCRIPT}" | head -n1 | cut -d: -f1)"
[[ -n "${nginx_test_line}" && -n "${nginx_reload_line}" ]] || fail "script must execute nginx -t and nginx -s reload through BAOTA_NGINX_BIN"
[[ "${nginx_test_line}" -lt "${nginx_reload_line}" ]] || fail "nginx -t must execute before nginx -s reload"

assert_not_contains "${APPLY_SCRIPT}" 'root@'
assert_not_contains "${APPLY_SCRIPT}" 'sshpass'
assert_not_contains "${APPLY_SCRIPT}" '2222|65022'
assert_not_contains "${APPLY_SCRIPT}" 'EV~|20050626|BOT_SERVICE_TOKEN|STUHELPER_PLATFORM_SERVICE_TOKEN'

tmpdir="$(mktemp -d)"
cleanup() {
  rm -rf "${tmpdir}"
}
trap cleanup EXIT

target_dir="${tmpdir}/vhost/nginx"
mkdir -p "${target_dir}"

rendered_connector="${tmpdir}/rendered-connector.conf"
python3 "${CONNECTOR_RENDERER}" \
  --template "${CONNECTOR_NGINX_TEMPLATE}" \
  --output "${rendered_connector}" \
  --public-port 9444 \
  --upstream-port 19444 \
  --allowed-cidrs "192.0.2.10/32,10.0.0.0/8"
assert_contains "${rendered_connector}" '^    listen 9444;$'
assert_contains "${rendered_connector}" '^    proxy_pass 127\.0\.0\.1:19444;$'
assert_contains "${rendered_connector}" '^    allow 192\.0\.2\.10/32;$'
assert_contains "${rendered_connector}" '^    allow 10\.0\.0\.0/8;$'
assert_contains "${rendered_connector}" '^    deny all;$'
[[ "$(stat -c '%a' "${rendered_connector}")" == "600" ]] ||
  fail "Connector renderer output must be mode 0600"

assert_renderer_rejects() {
  local label="$1"
  local public_port="$2"
  local upstream_port="$3"
  local allowed_cidrs="$4"
  local output="${tmpdir}/renderer-rejected-${label}.conf"

  if python3 "${CONNECTOR_RENDERER}" \
    --template "${CONNECTOR_NGINX_TEMPLATE}" \
    --output "${output}" \
    --public-port "${public_port}" \
    --upstream-port "${upstream_port}" \
    --allowed-cidrs "${allowed_cidrs}" \
    >"${tmpdir}/renderer-rejected-${label}.stdout" \
    2>"${tmpdir}/renderer-rejected-${label}.stderr"; then
    fail "Connector renderer must reject ${label}"
  fi
}

assert_renderer_rejects "empty-allowlist" "9444" "19444" ""
assert_renderer_rejects "world-allowlist" "9444" "19444" "0.0.0.0/0"
assert_renderer_rejects "ipv6-allowlist" "9444" "19444" "2001:db8::/32"
assert_renderer_rejects "host-bits" "9444" "19444" "10.0.0.1/24"
assert_renderer_rejects "duplicate-cidr" "9444" "19444" "10.0.0.0/8,10.0.0.0/8"
assert_renderer_rejects "multicast" "9444" "19444" "224.0.0.0/4"
assert_renderer_rejects "cidr-injection" "9444" "19444" "10.0.0.0/8; deny all"
assert_renderer_rejects "proxy-loop" "9444" "9444" "10.0.0.0/8"
assert_renderer_rejects "zero-port" "0" "19444" "10.0.0.0/8"
assert_renderer_rejects "oversized-port" "65536" "19444" "10.0.0.0/8"
assert_renderer_rejects "nonnumeric-port" "not-a-port" "19444" "10.0.0.0/8"

fake_nginx="${tmpdir}/nginx"
cat >"${fake_nginx}" <<'SH'
#!/usr/bin/env bash
set -euo pipefail

case "${1:-}" in
  -t)
    echo "test" >>"${FAKE_NGINX_LOG}"
    if [[ "${FAKE_NGINX_FAIL_TEST:-false}" == "true" ]]; then
      exit 1
    fi
    ;;
  -s)
    [[ "${2:-}" == "reload" ]] || exit 2
    echo "reload" >>"${FAKE_NGINX_LOG}"
    ;;
  -T)
    while IFS= read -r -d '' file; do
      printf '# configuration file %s:\n' "${file}"
      cat "${file}"
      printf '\n'
    done < <(find "${FAKE_NGINX_DUMP_DIR}" -type f -name '*.conf' -print0 | sort -z)
    ;;
  -v)
    echo "nginx version: fake-nginx"
    ;;
  *)
    exit 2
    ;;
esac
SH
chmod +x "${fake_nginx}"

stuhelper_target="${target_dir}/stuhelper.com.conf"
sso_target="${target_dir}/sso.stuhelper.com.conf"
sso_extension_target="${target_dir}/extension/sso.stuhelper.com/stuhelper-sso-well-known.conf"
connector_target="${target_dir}/tcp/connector.stuhelper.com.conf"
fake_log="${tmpdir}/fake-nginx.log"

BAOTA_NGINX_STUHELPER_TARGET="${stuhelper_target}" \
BAOTA_NGINX_SSO_TARGET="${sso_target}" \
BAOTA_NGINX_SSO_WELL_KNOWN_EXTENSION_TARGET="${sso_extension_target}" \
BAOTA_NGINX_BIN="${fake_nginx}" \
FAKE_NGINX_LOG="${fake_log}" \
FAKE_NGINX_DUMP_DIR="${target_dir}" \
bash "${APPLY_SCRIPT}" --profile all >"${tmpdir}/dry-run.stdout" 2>"${tmpdir}/dry-run.stderr"

[[ ! -e "${stuhelper_target}" ]] || fail "dry-run must not create stuhelper target"
[[ ! -e "${sso_target}" ]] || fail "dry-run must not create sso target"
[[ ! -e "${sso_extension_target}" ]] || fail "dry-run must not create sso well-known extension target"
[[ ! -e "${connector_target}" ]] || fail "legacy all profile must not create Connector target"
assert_contains "${tmpdir}/dry-run.stdout" 'dry-run only'

BAOTA_NGINX_STUHELPER_TARGET="${stuhelper_target}" \
BAOTA_NGINX_SSO_TARGET="${sso_target}" \
BAOTA_NGINX_SSO_WELL_KNOWN_EXTENSION_TARGET="${sso_extension_target}" \
BAOTA_NGINX_BIN="${fake_nginx}" \
FAKE_NGINX_LOG="${fake_log}" \
FAKE_NGINX_DUMP_DIR="${target_dir}" \
bash "${APPLY_SCRIPT}" --profile all --apply --reload --preflight >"${tmpdir}/apply.stdout" 2>"${tmpdir}/apply.stderr"

cmp -s "${MAIN_NGINX_FILE}" "${stuhelper_target}" || fail "stuhelper target does not match template after apply"
cmp -s "${SSO_NGINX_FILE}" "${sso_target}" || fail "sso target does not match template after apply"
cmp -s "${SSO_WELL_KNOWN_EXTENSION_FILE}" "${sso_extension_target}" || fail "sso well-known extension target does not match template after apply"
assert_contains "${fake_log}" '^test$'
assert_contains "${fake_log}" '^reload$'
assert_contains "${tmpdir}/apply.stdout" 'public Nginx ingress config preflight passed'

CAMPUS_CONNECTOR_GATEWAY_PUBLIC_PORT="9444" \
CAMPUS_CONNECTOR_GATEWAY_EXTERNAL_PORT="19444" \
CAMPUS_CONNECTOR_ALLOWED_SOURCE_CIDRS="192.0.2.10/32,10.0.0.0/8" \
BAOTA_NGINX_CONNECTOR_TARGET="${connector_target}" \
BAOTA_NGINX_BIN="${fake_nginx}" \
FAKE_NGINX_LOG="${tmpdir}/connector-dry-run-nginx.log" \
FAKE_NGINX_DUMP_DIR="${target_dir}" \
bash "${APPLY_SCRIPT}" --profile connector >"${tmpdir}/connector-dry-run.stdout" 2>"${tmpdir}/connector-dry-run.stderr"

[[ ! -e "${connector_target}" ]] || fail "Connector dry-run must not create its target"
assert_contains "${tmpdir}/connector-dry-run.stdout" 'connector would create'

connector_fake_log="${tmpdir}/connector-fake-nginx.log"
CAMPUS_CONNECTOR_GATEWAY_PUBLIC_PORT="9444" \
CAMPUS_CONNECTOR_GATEWAY_EXTERNAL_PORT="19444" \
CAMPUS_CONNECTOR_ALLOWED_SOURCE_CIDRS="192.0.2.10/32,10.0.0.0/8" \
BAOTA_NGINX_CONNECTOR_TARGET="${connector_target}" \
BAOTA_NGINX_BIN="${fake_nginx}" \
FAKE_NGINX_LOG="${connector_fake_log}" \
FAKE_NGINX_DUMP_DIR="${target_dir}" \
bash "${APPLY_SCRIPT}" --profile connector --apply --reload --preflight >"${tmpdir}/connector-apply.stdout" 2>"${tmpdir}/connector-apply.stderr"

[[ -f "${connector_target}" ]] || fail "Connector apply must create its stream target"
[[ "$(stat -c '%a' "${connector_target}")" == "644" ]] ||
  fail "installed Connector stream target must be mode 0644"
assert_contains "${connector_target}" '^    listen 9444;$'
assert_contains "${connector_target}" '^    proxy_pass 127\.0\.0\.1:19444;$'
assert_contains "${connector_target}" '^    allow 192\.0\.2\.10/32;$'
assert_contains "${connector_target}" '^    allow 10\.0\.0\.0/8;$'
assert_contains "${connector_target}" '^    deny all;$'
assert_not_contains "${connector_target}" 'ssl_certificate|ssl_certificate_key|proxy_ssl|listen .*ssl'
[[ "$(sed -n '/^test$/=' "${connector_fake_log}" | head -n1)" -lt "$(sed -n '/^reload$/=' "${connector_fake_log}" | head -n1)" ]] ||
  fail "Connector apply must run nginx -t before reload"
assert_contains "${tmpdir}/connector-apply.stdout" 'public Nginx ingress config preflight passed'

CAMPUS_CONNECTOR_GATEWAY_PUBLIC_PORT="9444" \
CAMPUS_CONNECTOR_GATEWAY_EXTERNAL_PORT="19444" \
CAMPUS_CONNECTOR_ALLOWED_SOURCE_CIDRS="192.0.2.10/32,10.0.0.0/8" \
BAOTA_NGINX_STUHELPER_TARGET="${stuhelper_target}" \
BAOTA_NGINX_CONNECTOR_TARGET="${connector_target}" \
BAOTA_NGINX_BIN="${fake_nginx}" \
FAKE_NGINX_LOG="${tmpdir}/app-all-nginx.log" \
FAKE_NGINX_DUMP_DIR="${target_dir}" \
bash "${APPLY_SCRIPT}" --profile app-all --preflight >"${tmpdir}/app-all.stdout" 2>"${tmpdir}/app-all.stderr"
assert_contains "${tmpdir}/app-all.stdout" 'public Nginx ingress config preflight passed'

assert_connector_apply_rejects() {
  local label="$1"
  local public_port="$2"
  local upstream_port="$3"
  local allowed_cidrs="$4"

  if CAMPUS_CONNECTOR_GATEWAY_PUBLIC_PORT="${public_port}" \
    CAMPUS_CONNECTOR_GATEWAY_EXTERNAL_PORT="${upstream_port}" \
    CAMPUS_CONNECTOR_ALLOWED_SOURCE_CIDRS="${allowed_cidrs}" \
    BAOTA_NGINX_CONNECTOR_TARGET="${connector_target}" \
    BAOTA_NGINX_BIN="${fake_nginx}" \
    FAKE_NGINX_LOG="${tmpdir}/connector-rejected-${label}-nginx.log" \
    FAKE_NGINX_DUMP_DIR="${target_dir}" \
    bash "${APPLY_SCRIPT}" --profile connector \
    >"${tmpdir}/connector-rejected-${label}.stdout" \
    2>"${tmpdir}/connector-rejected-${label}.stderr"; then
    fail "Connector apply must reject ${label}"
  fi
  assert_contains "${tmpdir}/connector-rejected-${label}.stderr" 'failed to render fail-closed campus connector stream ingress'
}

assert_connector_apply_rejects "empty-allowlist" "9444" "19444" ""
assert_connector_apply_rejects "world-allowlist" "9444" "19444" "0.0.0.0/0"
assert_connector_apply_rejects "proxy-loop" "9444" "9444" "192.0.2.10/32"

printf '\n# connector local drift\n' >>"${connector_target}"
connector_failed_log="${tmpdir}/connector-failed-nginx.log"
if CAMPUS_CONNECTOR_GATEWAY_PUBLIC_PORT="9444" \
  CAMPUS_CONNECTOR_GATEWAY_EXTERNAL_PORT="19444" \
  CAMPUS_CONNECTOR_ALLOWED_SOURCE_CIDRS="192.0.2.10/32,10.0.0.0/8" \
  BAOTA_NGINX_CONNECTOR_TARGET="${connector_target}" \
  BAOTA_NGINX_BIN="${fake_nginx}" \
  BAOTA_NGINX_BACKUP_TIMESTAMP="20260530T000002Z" \
  FAKE_NGINX_LOG="${connector_failed_log}" \
  FAKE_NGINX_DUMP_DIR="${target_dir}" \
  FAKE_NGINX_FAIL_TEST="true" \
  bash "${APPLY_SCRIPT}" --profile connector --apply --reload \
  >"${tmpdir}/connector-failed-apply.stdout" 2>"${tmpdir}/connector-failed-apply.stderr"; then
  fail "expected Connector apply to fail when nginx -t fails"
fi
assert_contains "${connector_target}" 'connector local drift'
assert_contains "${tmpdir}/connector-failed-apply.stderr" 'nginx -t failed after applying Baota/Nginx templates'
assert_not_contains "${connector_failed_log}" '^reload$'

printf '\n# local drift\n' >>"${stuhelper_target}"
BAOTA_NGINX_STUHELPER_TARGET="${stuhelper_target}" \
BAOTA_NGINX_SSO_TARGET="${sso_target}" \
BAOTA_NGINX_SSO_WELL_KNOWN_EXTENSION_TARGET="${sso_extension_target}" \
BAOTA_NGINX_BIN="${fake_nginx}" \
BAOTA_NGINX_BACKUP_TIMESTAMP="20260530T000000Z" \
FAKE_NGINX_LOG="${fake_log}" \
FAKE_NGINX_DUMP_DIR="${target_dir}" \
bash "${APPLY_SCRIPT}" --profile stuhelper --apply >"${tmpdir}/reapply.stdout" 2>"${tmpdir}/reapply.stderr"

backup_file="${stuhelper_target}.bak.20260530T000000Z"
[[ -f "${backup_file}" ]] || fail "expected backup file to be created: ${backup_file}"
assert_contains "${backup_file}" 'local drift'
cmp -s "${MAIN_NGINX_FILE}" "${stuhelper_target}" || fail "stuhelper target does not match template after reapply"

printf '\n# broken apply\n' >>"${sso_target}"
if BAOTA_NGINX_SSO_TARGET="${sso_target}" \
  BAOTA_NGINX_SSO_WELL_KNOWN_EXTENSION_TARGET="${sso_extension_target}" \
  BAOTA_NGINX_BIN="${fake_nginx}" \
  BAOTA_NGINX_BACKUP_TIMESTAMP="20260530T000001Z" \
  FAKE_NGINX_LOG="${fake_log}" \
  FAKE_NGINX_DUMP_DIR="${target_dir}" \
  FAKE_NGINX_FAIL_TEST="true" \
  bash "${APPLY_SCRIPT}" --profile sso --apply >"${tmpdir}/failed-apply.stdout" 2>"${tmpdir}/failed-apply.stderr"; then
  fail "expected apply to fail when nginx -t fails"
fi
assert_contains "${sso_target}" 'broken apply'
assert_contains "${tmpdir}/failed-apply.stderr" 'nginx -t failed after applying Baota/Nginx templates'

echo "[apply-baota-nginx-templates-contract] all assertions passed"
