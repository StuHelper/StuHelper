#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
PROVISION_SCRIPT="${REPO_ROOT}/infra/ops/provision-external-student-source-oracle-readonly.sh"
GO_LIVE_DOC="${REPO_ROOT}/docs/guides/production-go-live.md"
RELEASE_RUNBOOK="${REPO_ROOT}/docs/guides/release-runbook.md"
AGENT_POLICY="${REPO_ROOT}/AGENTS.md"

fail() {
  echo "[provision-external-student-source-oracle-readonly-contract][error] $*" >&2
  exit 1
}

assert_contains() {
  local file="$1"
  local pattern="$2"
  if ! grep -Eiq -- "${pattern}" "${file}"; then
    fail "expected ${file} to contain pattern: ${pattern}"
  fi
}

assert_not_contains() {
  local file="$1"
  local pattern="$2"
  if grep -Eiq -- "${pattern}" "${file}"; then
    fail "expected ${file} to not contain pattern: ${pattern}"
  fi
}

cleanup() {
  [[ -n "${tmpdir:-}" ]] && rm -rf "${tmpdir}"
}
trap cleanup EXIT

[[ -f "${PROVISION_SCRIPT}" ]] || fail "missing compatibility script: ${PROVISION_SCRIPT}"
bash -n "${PROVISION_SCRIPT}"
[[ -x "${PROVISION_SCRIPT}" ]] || fail "compatibility script must remain executable for fail-closed upgrades"

assert_contains "${PROVISION_SCRIPT}" 'permanently disabled'
assert_contains "${PROVISION_SCRIPT}" 'existing account'
assert_contains "${PROVISION_SCRIPT}" 'never opens an Oracle connection'
assert_contains "${PROVISION_SCRIPT}" 'provisioning is prohibited'
assert_not_contains "${PROVISION_SCRIPT}" 'sqlplus'
assert_not_contains "${PROVISION_SCRIPT}" 'EXTERNAL_STUDENT_SOURCE_ORACLE_READONLY_PASSWORD'
assert_not_contains "${PROVISION_SCRIPT}" 'execute[[:space:]]+immediate'
assert_not_contains "${PROVISION_SCRIPT}" 'grant[[:space:]]+create[[:space:]]+session'
assert_not_contains "${PROVISION_SCRIPT}" 'grant[[:space:]]+select'
assert_not_contains "${PROVISION_SCRIPT}" 'alter[[:space:]]+session'

tmpdir="$(mktemp -d)"
fake_bin="${tmpdir}/bin"
marker="${tmpdir}/oracle-client-invoked"
mkdir -p "${fake_bin}"
cat >"${fake_bin}/sqlplus" <<SH
#!/usr/bin/env bash
touch "${marker}"
exit 99
SH
chmod +x "${fake_bin}/sqlplus"

if PATH="${fake_bin}:$PATH" "${PROVISION_SCRIPT}" >"${tmpdir}/stdout" 2>"${tmpdir}/stderr"; then
  fail "disabled compatibility script unexpectedly succeeded"
fi
[[ ! -e "${marker}" ]] || fail "disabled compatibility script invoked an Oracle client"
assert_contains "${tmpdir}/stderr" 'provisioning is prohibited'

PATH="${fake_bin}:$PATH" "${PROVISION_SCRIPT}" --help >"${tmpdir}/help"
[[ ! -e "${marker}" ]] || fail "help path invoked an Oracle client"
assert_contains "${tmpdir}/help" 'never opens an Oracle connection'

assert_contains "${AGENT_POLICY}" '只能使用用户明确指定的既有账号'
assert_contains "${AGENT_POLICY}" '不得创建、申请、更换或建议自动创建'
assert_contains "${GO_LIVE_DOC}" 'provision-external-student-source-oracle-readonly\.sh'
assert_contains "${GO_LIVE_DOC}" '明确拒绝执行'
assert_contains "${RELEASE_RUNBOOK}" 'provision-external-student-source-oracle-readonly\.sh'
assert_contains "${RELEASE_RUNBOOK}" '明确拒绝执行'

echo "[provision-external-student-source-oracle-readonly-contract] all assertions passed"
