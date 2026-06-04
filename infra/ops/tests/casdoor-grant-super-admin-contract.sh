#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
GRANT_SCRIPT="${REPO_ROOT}/infra/ops/casdoor-grant-super-admin.sh"
PROD_ENV_EXAMPLE="${REPO_ROOT}/.env.prod.example"
RUNBOOK="${REPO_ROOT}/docs/guides/release-runbook.md"

fail() {
  echo "[casdoor-grant-super-admin-contract][error] $*" >&2
  exit 1
}

assert_contains() {
  local file="$1"
  local pattern="$2"
  if ! grep -Eq "${pattern}" "${file}"; then
    fail "expected ${file} to contain pattern: ${pattern}"
  fi
}

for file in "${GRANT_SCRIPT}" "${PROD_ENV_EXAMPLE}" "${RUNBOOK}"; do
  [[ -f "${file}" ]] || fail "missing file: ${file}"
done

bash -n "${GRANT_SCRIPT}"
[[ -x "${GRANT_SCRIPT}" ]] || fail "grant script must be executable"

assert_contains "${GRANT_SCRIPT}" 'Break-glass operation'
assert_contains "${GRANT_SCRIPT}" 'CASDOOR_GRANT_SUPER_ADMIN_USERS'
assert_contains "${GRANT_SCRIPT}" 'STUHELPER_INITIAL_SUPER_ADMINS'
assert_contains "${GRANT_SCRIPT}" 'CASDOOR_GRANT_SUPER_ADMIN_ORGANIZATION'
assert_contains "${GRANT_SCRIPT}" 'role_name="super_admin"'
assert_contains "${GRANT_SCRIPT}" 'this script only grants the super_admin role'
assert_contains "${GRANT_SCRIPT}" 'CASDOOR_DB_CONTAINER'
assert_contains "${GRANT_SCRIPT}" 'CASDOOR_DB_USER'
assert_contains "${GRANT_SCRIPT}" 'CASDOOR_DB_NAME'
assert_contains "${GRANT_SCRIPT}" 'docker exec -i'
assert_contains "${GRANT_SCRIPT}" 'psql'
assert_contains "${GRANT_SCRIPT}" 'public\."user"'
assert_contains "${GRANT_SCRIPT}" 'COALESCE\(u\.is_deleted, false\) = false'
assert_contains "${GRANT_SCRIPT}" 'COALESCE\(u\.is_forbidden, false\) = false'
assert_contains "${GRANT_SCRIPT}" 'FROM public\.role'
assert_contains "${GRANT_SCRIPT}" 'FOR UPDATE'
assert_contains "${GRANT_SCRIPT}" 'jsonb_array_elements_text'
assert_contains "${GRANT_SCRIPT}" 'UPDATE public\.role'
assert_contains "${GRANT_SCRIPT}" 'SET users'
assert_contains "${GRANT_SCRIPT}" 'sign out and sign in again'

assert_contains "${PROD_ENV_EXAMPLE}" '^STUHELPER_INITIAL_SUPER_ADMINS=REPLACE_WITH_INITIAL_SUPER_ADMIN_USERNAMES$'

assert_contains "${RUNBOOK}" 'StuHelper Admin break-glass'
assert_contains "${RUNBOOK}" 'built-in/admin'
assert_contains "${RUNBOOK}" 'casdoor-grant-super-admin\.sh'
assert_contains "${RUNBOOK}" 'user\.is_admin=true'
assert_contains "${RUNBOOK}" 'OIDC `roles` claim'
assert_contains "${RUNBOOK}" '退出 StuHelper 和 SSO 并重新登录'

echo "[casdoor-grant-super-admin-contract] all assertions passed"
