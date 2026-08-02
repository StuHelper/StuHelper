#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
CMD_FILE="${REPO_ROOT}/server/cmd/external-student-source-smoke/main.go"
DOCKERFILE="${REPO_ROOT}/server/Dockerfile"
SCRIPT_FILE="${REPO_ROOT}/infra/ops/external-student-source-smoke.sh"
PROD_ENV_EXAMPLE="${REPO_ROOT}/.env.prod.example"
INIT_PROD_ENV="${REPO_ROOT}/infra/ops/init-prod-env.sh"
RELEASE_RUNBOOK="${REPO_ROOT}/docs/guides/release-runbook.md"
GO_LIVE="${REPO_ROOT}/docs/guides/production-go-live.md"

fail() {
  echo "[external-student-source-smoke-contract][error] $*" >&2
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
    fail "expected ${file} to not contain pattern: ${pattern}"
  fi
}

for file in \
  "${CMD_FILE}" \
  "${DOCKERFILE}" \
  "${SCRIPT_FILE}" \
  "${PROD_ENV_EXAMPLE}" \
  "${INIT_PROD_ENV}" \
  "${RELEASE_RUNBOOK}" \
  "${GO_LIVE}"; do
  [[ -f "${file}" ]] || fail "missing file: ${file}"
done

bash -n "${SCRIPT_FILE}"
[[ -x "${SCRIPT_FILE}" ]] || fail "external student source smoke script must be executable"

assert_contains "${CMD_FILE}" 'external-student-source-smoke'
assert_contains "${CMD_FILE}" 'EXTERNAL_STUDENT_SOURCE_ENABLED must be true'
assert_contains "${CMD_FILE}" 'EXTERNAL_STUDENT_SOURCE_PROVIDER must be oracle'
assert_contains "${CMD_FILE}" 'EXTERNAL_STUDENT_SOURCE_SMOKE_REQUIRE_SAMPLE'
assert_contains "${CMD_FILE}" 'EXTERNAL_STUDENT_SOURCE_SMOKE_STUDENT_ID is required'
assert_contains "${CMD_FILE}" 'StudentIDHashPrefix'
assert_contains "${CMD_FILE}" 'redactSensitiveText'
assert_contains "${CMD_FILE}" 'EXTERNAL_STUDENT_SOURCE_ORACLE_PASSWORD'
assert_contains "${CMD_FILE}" 'ReadableRecordPresent'
assert_contains "${CMD_FILE}" 'TLSVerified'
assert_contains "${CMD_FILE}" 'RuntimeIdentityMatched'
assert_contains "${CMD_FILE}" 'LeastPrivilegeGrantsVerified'
assert_contains "${CMD_FILE}" 'ProbeRuntimeSecurity'
assert_contains "${CMD_FILE}" 'BreakerFailureThreshold'
assert_not_contains "${CMD_FILE}" 'fmt[.]Printf.*EXTERNAL_STUDENT_SOURCE_ORACLE_PASSWORD'
assert_not_contains "${CMD_FILE}" 'fmt[.]Printf.*SampleStudentID'
assert_not_contains "${CMD_FILE}" 'json:"studentID"'
assert_not_contains "${CMD_FILE}" 'json:"studentName"'

assert_contains "${DOCKERFILE}" './cmd/external-student-source-smoke'
assert_contains "${DOCKERFILE}" '/app/external-student-source-smoke'

assert_contains "${SCRIPT_FILE}" 'EXTERNAL_STUDENT_SOURCE_SMOKE_MODE'
assert_contains "${SCRIPT_FILE}" '--entrypoint "\$\{entrypoint\}"'
assert_contains "${SCRIPT_FILE}" '/app/external-student-source-smoke'
assert_contains "${SCRIPT_FILE}" 'EXTERNAL_STUDENT_SOURCE_SMOKE_EVIDENCE_FILE'
assert_contains "${SCRIPT_FILE}" 'readableRecordPresent'
assert_contains "${SCRIPT_FILE}" 'external Oracle student source did not use verified TLS'
assert_contains "${SCRIPT_FILE}" 'runtime identity did not match the provisioned readonly identity'
assert_contains "${SCRIPT_FILE}" 'leastPrivilegeGrantsVerified'
assert_contains "${SCRIPT_FILE}" 'expected_grant_counts'
assert_contains "${SCRIPT_FILE}" 'expected_oracle_tuning'
assert_contains "${SCRIPT_FILE}" 'EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_CA_HOST_PATH'
assert_contains "${SCRIPT_FILE}" 'EXTERNAL_STUDENT_SOURCE_SMOKE_REQUIRE_SAMPLE'
assert_contains "${SCRIPT_FILE}" 'redact_file'
assert_not_contains "${SCRIPT_FILE}" 'echo .*\$\{EXTERNAL_STUDENT_SOURCE_ORACLE_PASSWORD'
assert_not_contains "${SCRIPT_FILE}" 'printf .*\$\{EXTERNAL_STUDENT_SOURCE_ORACLE_PASSWORD'

assert_contains "${PROD_ENV_EXAMPLE}" '^EXTERNAL_STUDENT_SOURCE_SMOKE_MODE=container$'
assert_contains "${PROD_ENV_EXAMPLE}" '^EXTERNAL_STUDENT_SOURCE_SMOKE_COMMAND=/app/external-student-source-smoke$'
assert_contains "${PROD_ENV_EXAMPLE}" '^EXTERNAL_STUDENT_SOURCE_SMOKE_REQUIRE_READABLE_RECORD=true$'
assert_contains "${PROD_ENV_EXAMPLE}" '^EXTERNAL_STUDENT_SOURCE_SMOKE_REQUIRE_SAMPLE=false$'
assert_contains "${PROD_ENV_EXAMPLE}" '^EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_MODE=verify-full$'
assert_contains "${PROD_ENV_EXAMPLE}" '^EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_CA_FILE=/external-student-source-tls/ca\.crt$'
assert_contains "${PROD_ENV_EXAMPLE}" '^EXTERNAL_STUDENT_SOURCE_ORACLE_READONLY_USERNAME=STUHELPER_ACADEMIC_RO$'
assert_contains "${INIT_PROD_ENV}" 'EXTERNAL_STUDENT_SOURCE_SMOKE_COMMAND'
assert_contains "${INIT_PROD_ENV}" 'EXTERNAL_STUDENT_SOURCE_ORACLE_PASSWORD'
assert_contains "${INIT_PROD_ENV}" 'EXTERNAL_STUDENT_SOURCE_ORACLE_READONLY_USERNAME'

assert_contains "${RELEASE_RUNBOOK}" 'external-student-source-smoke.sh'
assert_contains "${GO_LIVE}" 'external-student-source-smoke.sh'

echo "[external-student-source-smoke-contract] all assertions passed"
