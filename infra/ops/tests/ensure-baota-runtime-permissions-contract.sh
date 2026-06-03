#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
SCRIPT="${REPO_ROOT}/infra/ops/ensure-baota-runtime-permissions.sh"
POSTGRES_TLS_SCRIPT="${REPO_ROOT}/infra/ops/render-postgres-tls.sh"

fail() {
  echo "[ensure-baota-runtime-permissions-contract][error] $*" >&2
  exit 1
}

assert_mode() {
  local expected="$1"
  local path="$2"
  local actual
  actual="$(stat -c '%a' "${path}")"
  [[ "${actual}" == "${expected}" ]] || fail "expected ${path} mode ${expected}, got ${actual}"
}

assert_contains() {
  local file="$1"
  local pattern="$2"
  if ! grep -Eq -- "${pattern}" "${file}"; then
    fail "expected ${file} to contain pattern: ${pattern}"
  fi
}

[[ -f "${SCRIPT}" ]] || fail "missing script: ${SCRIPT}"
bash -n "${SCRIPT}"

assert_contains "${POSTGRES_TLS_SCRIPT}" 'POSTGRES_TLS_SERVER_KEY_OWNER="\$\{POSTGRES_TLS_SERVER_KEY_OWNER:-70:70\}"'
assert_contains "${POSTGRES_TLS_SCRIPT}" 'chown "\$\{POSTGRES_TLS_SERVER_KEY_OWNER\}" "\$\{SERVER_KEY\}"'

tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT

source_dir="${tmpdir}/source"
casdoor_root="${tmpdir}/casdoor"
mkdir -p \
  "${source_dir}/infra/generated/postgres" \
  "${source_dir}/infra/generated/redis" \
  "${casdoor_root}/conf" \
  "${casdoor_root}/logs"

touch \
  "${source_dir}/infra/generated/postgres/ca.key" \
  "${source_dir}/infra/generated/postgres/ca.crt" \
  "${source_dir}/infra/generated/postgres/server.key" \
  "${source_dir}/infra/generated/postgres/server.crt" \
  "${source_dir}/infra/generated/redis/ca.key" \
  "${source_dir}/infra/generated/redis/ca.crt" \
  "${source_dir}/infra/generated/redis/server.key" \
  "${source_dir}/infra/generated/redis/server.crt" \
  "${source_dir}/infra/generated/redis/users.acl" \
  "${casdoor_root}/conf/app.conf" \
  "${casdoor_root}/logs/casdoor.log"

chmod 700 "${source_dir}/infra/generated/postgres" "${source_dir}/infra/generated/redis"
chmod 600 "${source_dir}/infra/generated/redis/users.acl"
chmod 700 "${casdoor_root}/conf" "${casdoor_root}/logs"
chmod 600 "${casdoor_root}/conf/app.conf" "${casdoor_root}/logs/casdoor.log"

"${SCRIPT}" --source-dir "${source_dir}" --casdoor-compose-root "${casdoor_root}" >"${tmpdir}/dry-run.out"
assert_contains "${tmpdir}/dry-run.out" 'dry-run complete'
assert_mode 700 "${source_dir}/infra/generated/postgres"
assert_mode 600 "${source_dir}/infra/generated/redis/users.acl"

"${SCRIPT}" --source-dir "${source_dir}" --casdoor-compose-root "${casdoor_root}" --apply >"${tmpdir}/apply.out"
assert_contains "${tmpdir}/apply.out" 'runtime bind-mount permissions normalized'

assert_mode 755 "${source_dir}/infra/generated/postgres"
assert_mode 600 "${source_dir}/infra/generated/postgres/ca.key"
assert_mode 600 "${source_dir}/infra/generated/postgres/server.key"
assert_mode 644 "${source_dir}/infra/generated/postgres/ca.crt"
assert_mode 644 "${source_dir}/infra/generated/postgres/server.crt"

assert_mode 755 "${source_dir}/infra/generated/redis"
assert_mode 600 "${source_dir}/infra/generated/redis/ca.key"
assert_mode 644 "${source_dir}/infra/generated/redis/ca.crt"
assert_mode 644 "${source_dir}/infra/generated/redis/server.key"
assert_mode 644 "${source_dir}/infra/generated/redis/server.crt"
assert_mode 644 "${source_dir}/infra/generated/redis/users.acl"

assert_mode 750 "${casdoor_root}/conf"
assert_mode 640 "${casdoor_root}/conf/app.conf"
assert_mode 750 "${casdoor_root}/logs"

"${SCRIPT}" --source-dir "${source_dir}" --casdoor-compose-root "${casdoor_root}" --skip-casdoor --apply >"${tmpdir}/skip.out"
assert_contains "${tmpdir}/skip.out" 'runtime bind-mount permissions normalized'

echo "[ensure-baota-runtime-permissions-contract] all assertions passed"
