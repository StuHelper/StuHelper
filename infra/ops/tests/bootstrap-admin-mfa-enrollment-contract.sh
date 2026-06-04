#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
BOOTSTRAP_SCRIPT="${REPO_ROOT}/infra/ops/bootstrap-admin-mfa-enrollment.sh"
PROD_ENV_EXAMPLE="${REPO_ROOT}/.env.prod.example"
RUNBOOK="${REPO_ROOT}/docs/guides/release-runbook.md"

fail() {
  echo "[bootstrap-admin-mfa-enrollment-contract][error] $*" >&2
  exit 1
}

assert_contains() {
  local file="$1"
  local pattern="$2"
  if ! grep -Eq "${pattern}" "${file}"; then
    fail "expected ${file} to contain pattern: ${pattern}"
  fi
}

assert_literal() {
  local file="$1"
  local needle="$2"
  if ! grep -Fq "${needle}" "${file}"; then
    fail "expected ${file} to contain literal text: ${needle}"
  fi
}

for file in "${BOOTSTRAP_SCRIPT}" "${PROD_ENV_EXAMPLE}" "${RUNBOOK}"; do
  [[ -f "${file}" ]] || fail "missing file: ${file}"
done

bash -n "${BOOTSTRAP_SCRIPT}"
[[ -x "${BOOTSTRAP_SCRIPT}" ]] || fail "bootstrap script must be executable"

assert_contains "${BOOTSTRAP_SCRIPT}" 'Break-glass operation'
assert_contains "${BOOTSTRAP_SCRIPT}" 'user_mfa_enrollment'
assert_contains "${BOOTSTRAP_SCRIPT}" 'STUHELPER_ADMIN_MFA_BOOTSTRAP_USERS'
assert_contains "${BOOTSTRAP_SCRIPT}" 'STUHELPER_ADMIN_MFA_BOOTSTRAP_METHOD'
assert_contains "${BOOTSTRAP_SCRIPT}" 'STUHELPER_ADMIN_MFA_REQUIRE_CASDOOR_MFA'
assert_contains "${BOOTSTRAP_SCRIPT}" 'STUHELPER_APP_CONTAINER'
assert_contains "${BOOTSTRAP_SCRIPT}" 'STUHELPER_DB_CONTAINER'
assert_contains "${BOOTSTRAP_SCRIPT}" 'CASDOOR_DB_CONTAINER'
assert_contains "${BOOTSTRAP_SCRIPT}" 'public\."user"'
assert_literal "${BOOTSTRAP_SCRIPT}" "name IN ('super_admin', 'school_admin')"
assert_literal "${BOOTSTRAP_SCRIPT}" "member = :'organization' || '/' || :'target_user'"
assert_contains "${BOOTSTRAP_SCRIPT}" 'COALESCE\("webauthnCredentials"'
assert_contains "${BOOTSTRAP_SCRIPT}" 'mfa_phone_enabled'
assert_contains "${BOOTSTRAP_SCRIPT}" 'totp_secret'
assert_literal "${BOOTSTRAP_SCRIPT}" "ARRAY[:'mfa_method']::text[]"
assert_contains "${BOOTSTRAP_SCRIPT}" 'ON CONFLICT \(user_id\) DO UPDATE'
assert_contains "${BOOTSTRAP_SCRIPT}" 'iam\.mfa\.bootstrap'
assert_contains "${BOOTSTRAP_SCRIPT}" 'gen_random_uuid\(\)::text'
assert_contains "${BOOTSTRAP_SCRIPT}" 'sign out and sign in again through SSO with MFA'

assert_contains "${PROD_ENV_EXAMPLE}" '^STUHELPER_ADMIN_MFA_BOOTSTRAP_USERS=REPLACE_WITH_PRIVILEGED_ADMIN_USERNAMES$'

assert_contains "${RUNBOOK}" 'bootstrap-admin-mfa-enrollment\.sh'
assert_contains "${RUNBOOK}" 'user_mfa_enrollment'
assert_contains "${RUNBOOK}" 'A0010204'
assert_contains "${RUNBOOK}" 'SMS/WebAuthn/TOTP'
assert_contains "${RUNBOOK}" '退出 StuHelper 和 SSO 并重新登录'

echo "[bootstrap-admin-mfa-enrollment-contract] all assertions passed"
