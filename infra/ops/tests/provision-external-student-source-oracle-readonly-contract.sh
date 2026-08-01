#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
PROVISION_SCRIPT="${REPO_ROOT}/infra/ops/provision-external-student-source-oracle-readonly.sh"
GO_LIVE_DOC="${REPO_ROOT}/docs/guides/production-go-live.md"
RELEASE_RUNBOOK="${REPO_ROOT}/docs/guides/release-runbook.md"

fail() {
  echo "[provision-external-student-source-oracle-readonly-contract][error] $*" >&2
  exit 1
}

assert_contains() {
  local file="$1"
  local pattern="$2"
  if ! grep -Eq -- "${pattern}" "${file}"; then
    fail "expected ${file} to contain pattern: ${pattern}"
  fi
}

cleanup() {
  [[ -n "${tmpdir:-}" ]] && rm -rf "${tmpdir}"
}
trap cleanup EXIT

[[ -f "${PROVISION_SCRIPT}" ]] || fail "missing provision script: ${PROVISION_SCRIPT}"
bash -n "${PROVISION_SCRIPT}"
[[ -x "${PROVISION_SCRIPT}" ]] || fail "provision script must be executable"

assert_contains "${PROVISION_SCRIPT}" 'EXTERNAL_STUDENT_SOURCE_ORACLE_READONLY_PASSWORD'
assert_contains "${PROVISION_SCRIPT}" 'STUHELPER_ACADEMIC_RO'
assert_contains "${PROVISION_SCRIPT}" 'ORCLPDB1'
assert_contains "${PROVISION_SCRIPT}" 'USR_JWBIZ'
assert_contains "${PROVISION_SCRIPT}" 'T_XS_JBXX'
assert_contains "${PROVISION_SCRIPT}" 'grant create session'
assert_contains "${PROVISION_SCRIPT}" 'grant select on'
assert_contains "${PROVISION_SCRIPT}" 'READONLY_HAS_SELECT'
assert_contains "${PROVISION_SCRIPT}" 'READONLY_NONEMPTY_COLUMNS'
assert_contains "${PROVISION_SCRIPT}" 'readonly username must not own the source schema'
assert_contains "${PROVISION_SCRIPT}" 'dba_role_privs'
assert_contains "${PROVISION_SCRIPT}" 'dba_sys_privs'
assert_contains "${PROVISION_SCRIPT}" 'dba_tab_privs'
assert_contains "${PROVISION_SCRIPT}" 'dba_col_privs'
assert_contains "${PROVISION_SCRIPT}" "hierarchy = 'NO'"
assert_contains "${PROVISION_SCRIPT}" 'privileges outside the approved CREATE SESSION and target-table SELECT boundary'
assert_contains "${PROVISION_SCRIPT}" 'unset EXTERNAL_STUDENT_SOURCE_ORACLE_READONLY_PASSWORD'
assert_contains "${PROVISION_SCRIPT}" 'must be 30 characters or fewer'
assert_contains "${PROVISION_SCRIPT}" 'never prints the password or raw'

tmpdir="$(mktemp -d)"
fake_bin="${tmpdir}/bin"
mkdir -p "${fake_bin}"
cat >"${fake_bin}/sqlplus" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
[[ -z "${EXTERNAL_STUDENT_SOURCE_ORACLE_READONLY_PASSWORD:-}" ]]
sql_file="${@: -1}"
sql_file="${sql_file#@}"
grep -q 'grant create session to STUHELPER_ACADEMIC_RO' "${sql_file}"
grep -q 'grant select on USR_JWBIZ.T_XS_JBXX to STUHELPER_ACADEMIC_RO' "${sql_file}"
grep -q 'READONLY_NONEMPTY_COLUMNS=' "${sql_file}"
printf 'READONLY_USER_EXISTS=1\nREADONLY_HAS_SELECT=1\nREADONLY_TABLE_COUNT=132806\nREADONLY_NONEMPTY_COLUMNS=132806\n'
SH
chmod +x "${fake_bin}/sqlplus"

output="$(
  PATH="${fake_bin}:$PATH" \
  EXTERNAL_STUDENT_SOURCE_ORACLE_READONLY_PASSWORD=SuperSecret123 \
  "${PROVISION_SCRIPT}"
)"
grep -q 'READONLY_HAS_SELECT=1' <<<"${output}" || fail "fake sqlplus output missing grant status"
grep -q 'external student source Oracle readonly user provisioned' <<<"${output}" || fail "provision success message missing"
if grep -q 'SuperSecret123' <<<"${output}"; then
  fail "provision output leaked password"
fi

if PATH="${fake_bin}:$PATH" \
  EXTERNAL_STUDENT_SOURCE_ORACLE_READONLY_PASSWORD=ThisPasswordIsTooLongForOracle123 \
  "${PROVISION_SCRIPT}" >/tmp/provision-oracle-ro-long-password.stdout 2>/tmp/provision-oracle-ro-long-password.stderr; then
  fail "provision unexpectedly accepted an overlong Oracle password"
fi
assert_contains /tmp/provision-oracle-ro-long-password.stderr '30 characters or fewer'

if PATH="${fake_bin}:$PATH" \
  EXTERNAL_STUDENT_SOURCE_ORACLE_READONLY_USERNAME=USR_JWBIZ \
  EXTERNAL_STUDENT_SOURCE_ORACLE_READONLY_PASSWORD=SuperSecret123 \
  "${PROVISION_SCRIPT}" >/tmp/provision-oracle-ro-owner.stdout 2>/tmp/provision-oracle-ro-owner.stderr; then
  fail "provision unexpectedly accepted the source schema owner"
fi
assert_contains /tmp/provision-oracle-ro-owner.stderr 'must not own the source schema'

assert_contains "${GO_LIVE_DOC}" 'provision-external-student-source-oracle-readonly\.sh'
assert_contains "${RELEASE_RUNBOOK}" 'provision-external-student-source-oracle-readonly\.sh'

echo "[provision-external-student-source-oracle-readonly-contract] all assertions passed"
