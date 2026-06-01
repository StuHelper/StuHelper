#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
APPLY_SCRIPT="${REPO_ROOT}/infra/ops/apply-baota-nginx-templates.sh"
MAIN_NGINX_FILE="${REPO_ROOT}/infra/nginx/baota-stuhelper.conf"
SSO_NGINX_FILE="${REPO_ROOT}/infra/nginx/baota-casdoor-sso.conf"
SSO_WELL_KNOWN_EXTENSION_FILE="${REPO_ROOT}/infra/nginx/baota-casdoor-sso-well-known-extension.conf"

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
assert_contains "${APPLY_SCRIPT}" '/www/server/panel/vhost/nginx/stuhelper\.com\.conf'
assert_contains "${APPLY_SCRIPT}" '/www/server/panel/vhost/nginx/sso\.stuhelper\.com\.conf'
assert_contains "${APPLY_SCRIPT}" '/www/server/panel/vhost/nginx/extension/sso\.stuhelper\.com/stuhelper-sso-well-known\.conf'
assert_contains "${APPLY_SCRIPT}" '--apply'
assert_contains "${APPLY_SCRIPT}" 'dry-run'
assert_contains "${APPLY_SCRIPT}" 'date -u \+%Y%m%dT%H%M%SZ'
assert_contains "${APPLY_SCRIPT}" '\.bak\.\$\{timestamp\}'
assert_contains "${APPLY_SCRIPT}" 'install -m 0644'
assert_contains "${APPLY_SCRIPT}" 'nginx -t'
assert_contains "${APPLY_SCRIPT}" 'nginx -s reload'
assert_contains "${APPLY_SCRIPT}" 'nginx-public-ingress-preflight\.sh'
assert_contains "${APPLY_SCRIPT}" 'NGINX_PUBLIC_INGRESS_NGINX_BIN'

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
    for file in "${FAKE_NGINX_DUMP_DIR}"/*.conf; do
      [[ -f "${file}" ]] || continue
      cat "${file}"
      printf '\n'
    done
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
