#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
SCRIPT="${ROOT_DIR}/infra/ops/admission-mvp-local-test.sh"
MAKEFILE="${ROOT_DIR}/Makefile"
GO_LIVE_DOC="${ROOT_DIR}/docs/guides/production-go-live.md"
RELEASE_RUNBOOK="${ROOT_DIR}/docs/guides/release-runbook.md"

fail() {
  printf '[admission-mvp-local-test-contract] %s\n' "$*" >&2
  exit 1
}

assert_file_contains() {
  local file="$1"
  local pattern="$2"
  grep -Eq -- "${pattern}" "${file}" || fail "${file#"${ROOT_DIR}"/} missing pattern: ${pattern}"
}

assert_file_not_contains() {
  local file="$1"
  local pattern="$2"
  if grep -Eq -- "${pattern}" "${file}"; then
    fail "${file#"${ROOT_DIR}"/} unexpectedly contains stale pattern: ${pattern}"
  fi
}

[[ -f "${SCRIPT}" ]] || fail "missing infra/ops/admission-mvp-local-test.sh"
[[ -x "${SCRIPT}" ]] || fail "infra/ops/admission-mvp-local-test.sh must be executable"

assert_file_contains "${SCRIPT}" '^#!/usr/bin/env bash$'
assert_file_contains "${SCRIPT}" 'go test -count=1 -p 1 \./internal/modules/admission'
assert_file_contains "${SCRIPT}" 'go test -count=1 -p 1 \./internal/modules/auth \./internal/modules/user'
assert_file_contains "${SCRIPT}" 'admissionPageStates\.test\.ts'
assert_file_contains "${SCRIPT}" 'admissionToken\.test\.ts'
assert_file_contains "${SCRIPT}" 'api\.test\.ts'
assert_file_contains "${SCRIPT}" 'joinStartPage\.test\.ts'
assert_file_contains "${SCRIPT}" 'projectionRefresh\.test\.ts'
assert_file_contains "${SCRIPT}" 'authAuthorizeFlow\.test\.ts'
assert_file_contains "${SCRIPT}" 'verification\.test\.ts'
assert_file_contains "${SCRIPT}" 'accountProfilePage\.test\.ts'
assert_file_contains "${SCRIPT}" 'studentVerificationPage\.test\.ts'
assert_file_contains "${SCRIPT}" 'mainlandID\.test\.ts'
assert_file_not_contains "${SCRIPT}" 'cameraCapture\.test\.ts'
assert_file_not_contains "${SCRIPT}" 'freshmanMobileCameraPage\.test\.ts'
assert_file_not_contains "${SCRIPT}" 'oldStudentFlow\.test\.ts'
assert_file_not_contains "${SCRIPT}" 'identityVerificationPage\.test\.ts'
assert_file_contains "${SCRIPT}" 'auth-callback-and-admission\.spec\.ts'
assert_file_contains "${SCRIPT}" 'auth-flow\.spec\.ts'
assert_file_contains "${SCRIPT}" 'user-verification\.spec\.ts'
assert_file_contains "${SCRIPT}" 'ADMISSION_MVP_PLAYWRIGHT_REUSE_SERVER'
assert_file_contains "${SCRIPT}" 'ADMISSION_MVP_PLAYWRIGHT_RETRIES="\$\{ADMISSION_MVP_PLAYWRIGHT_RETRIES:-1\}"'
assert_file_contains "${SCRIPT}" 'ADMISSION_MVP_TESTCONTAINERS_QUIESCE_TIMEOUT_SECONDS'
assert_file_contains "${SCRIPT}" 'capture_testcontainers_baseline'
assert_file_contains "${SCRIPT}" 'wait_for_testcontainers_network_quiescence'
assert_file_contains "${SCRIPT}" "label=org\.testcontainers=true"
assert_file_contains "${SCRIPT}" 'PLAYWRIGHT_REUSE_SERVER="\$\{ADMISSION_MVP_PLAYWRIGHT_REUSE_SERVER\}"'
assert_file_contains "${SCRIPT}" '--workers=1'
assert_file_contains "${SCRIPT}" '--retries="\$\{ADMISSION_MVP_PLAYWRIGHT_RETRIES\}"'
assert_file_contains "${SCRIPT}" 'koishi-plugin-stuhelper-group-guard test:unit'
assert_file_contains "${SCRIPT}" 'corepack yarn build'
assert_file_contains "${SCRIPT}" 'admission-public-smoke-contract\.sh'
assert_file_contains "${SCRIPT}" 'admission-production-readiness-contract\.sh'
assert_file_contains "${SCRIPT}" 'admission-reviewer-readiness-contract\.sh'
assert_file_contains "${SCRIPT}" 'admission-mvp-production-evidence-contract\.sh'
assert_file_contains "${SCRIPT}" 'admission-mvp-final-evidence-verify-contract\.sh'
assert_file_contains "${SCRIPT}" 'admission-join-e2e-evidence-contract\.sh'
assert_file_contains "${SCRIPT}" 'koishi-admission-production-evidence-contract\.sh'
assert_file_contains "${SCRIPT}" 'prod-parity-contract\.sh'
assert_file_contains "${SCRIPT}" 'public-web-auth-browser-smoke-contract\.sh'
assert_file_contains "${SCRIPT}" 'ADMISSION_MVP_SKIP_PLAYWRIGHT'
assert_file_contains "${SCRIPT}" 'ADMISSION_MVP_SKIP_BUILD'
assert_file_contains "${SCRIPT}" 'ADMISSION_MVP_SKIP_USER_CENTER'
assert_file_contains "${SCRIPT}" 'ADMISSION_MVP_SKIP_KOISHI'
assert_file_contains "${SCRIPT}" 'ADMISSION_MVP_SKIP_INFRA_CONTRACTS'

assert_file_contains "${MAKEFILE}" '^check-admission-mvp:$'
assert_file_contains "${MAKEFILE}" '\./infra/ops/admission-mvp-local-test\.sh'
assert_file_contains "${MAKEFILE}" 'make check-admission-mvp - run the local admission MVP regression suite'
assert_file_contains "${GO_LIVE_DOC}" 'make check-admission-mvp'
assert_file_contains "${RELEASE_RUNBOOK}" 'make check-admission-mvp'

printf '[admission-mvp-local-test-contract] all assertions passed\n'
