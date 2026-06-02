#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

PLAYWRIGHT_WEB_PORT="${PLAYWRIGHT_WEB_PORT:-3018}"
ADMISSION_MVP_PLAYWRIGHT_REUSE_SERVER="${ADMISSION_MVP_PLAYWRIGHT_REUSE_SERVER:-0}"
ADMISSION_MVP_SKIP_BUILD="${ADMISSION_MVP_SKIP_BUILD:-false}"
ADMISSION_MVP_SKIP_PLAYWRIGHT="${ADMISSION_MVP_SKIP_PLAYWRIGHT:-false}"
ADMISSION_MVP_SKIP_USER_CENTER="${ADMISSION_MVP_SKIP_USER_CENTER:-false}"
ADMISSION_MVP_SKIP_KOISHI="${ADMISSION_MVP_SKIP_KOISHI:-false}"
ADMISSION_MVP_SKIP_INFRA_CONTRACTS="${ADMISSION_MVP_SKIP_INFRA_CONTRACTS:-false}"
ADMISSION_MVP_PLAYWRIGHT_RETRIES="${ADMISSION_MVP_PLAYWRIGHT_RETRIES:-1}"

log() {
  printf '[admission-mvp-test] %s\n' "$*"
}

run_in() {
  local dir="$1"
  shift
  log "running in ${dir#${ROOT_DIR}/}: $*"
  (
    cd "${dir}"
    "$@"
  )
}

run_server_tests() {
  run_in "${ROOT_DIR}/server" go test -count=1 -p 1 ./internal/modules/admission
}

run_identity_dependency_server_tests() {
  if [[ "${ADMISSION_MVP_SKIP_USER_CENTER}" == "true" ]]; then
    log "skipping auth/user dependency server tests"
    return
  fi
  run_in "${ROOT_DIR}/server" go test -count=1 -p 1 ./internal/modules/auth ./internal/modules/user
}

run_web_unit_tests() {
  run_in "${ROOT_DIR}/clients" corepack pnpm --filter @stuhelper/web exec vitest run \
    src/modules/admission/__tests__/admissionPageStates.test.ts \
    src/modules/admission/__tests__/admissionToken.test.ts \
    src/modules/admission/__tests__/api.test.ts \
    src/modules/admission/__tests__/cameraCapture.test.ts \
    src/modules/admission/__tests__/freshmanMobileCameraPage.test.ts \
    src/modules/admission/__tests__/oldStudentFlow.test.ts \
    src/modules/admission/__tests__/projectionRefresh.test.ts
}

run_identity_dependency_web_unit_tests() {
  if [[ "${ADMISSION_MVP_SKIP_USER_CENTER}" == "true" ]]; then
    log "skipping auth/user dependency Web unit tests"
    return
  fi
  run_in "${ROOT_DIR}/clients" corepack pnpm --filter @stuhelper/web exec vitest run \
    src/stores/__tests__/authAuthorizeFlow.test.ts \
    src/modules/user/__tests__/identityVerificationPage.test.ts \
    src/modules/user/__tests__/studentVerificationPage.test.ts \
    src/modules/user/utils/mainlandID.test.ts
}

run_web_browser_tests() {
  if [[ "${ADMISSION_MVP_SKIP_PLAYWRIGHT}" == "true" ]]; then
    log "skipping Playwright admission browser tests"
    return
  fi
  log "running in clients: PLAYWRIGHT_WEB_PORT=${PLAYWRIGHT_WEB_PORT} PLAYWRIGHT_REUSE_SERVER=${ADMISSION_MVP_PLAYWRIGHT_REUSE_SERVER} playwright admission flow"
  (
    cd "${ROOT_DIR}/clients"
    PLAYWRIGHT_WEB_PORT="${PLAYWRIGHT_WEB_PORT}" \
    PLAYWRIGHT_REUSE_SERVER="${ADMISSION_MVP_PLAYWRIGHT_REUSE_SERVER}" \
      corepack pnpm --filter @stuhelper/web exec playwright test \
        tests/e2e/auth-callback-and-admission.spec.ts \
        --workers=1 \
        --retries="${ADMISSION_MVP_PLAYWRIGHT_RETRIES}"
  )
}

run_identity_dependency_browser_tests() {
  if [[ "${ADMISSION_MVP_SKIP_PLAYWRIGHT}" == "true" || "${ADMISSION_MVP_SKIP_USER_CENTER}" == "true" ]]; then
    log "skipping Playwright user-center dependency tests"
    return
  fi
  log "running in clients: PLAYWRIGHT_WEB_PORT=${PLAYWRIGHT_WEB_PORT} PLAYWRIGHT_REUSE_SERVER=${ADMISSION_MVP_PLAYWRIGHT_REUSE_SERVER} auth and user verification flows"
  (
    cd "${ROOT_DIR}/clients"
    PLAYWRIGHT_WEB_PORT="${PLAYWRIGHT_WEB_PORT}" \
    PLAYWRIGHT_REUSE_SERVER="${ADMISSION_MVP_PLAYWRIGHT_REUSE_SERVER}" \
      corepack pnpm --filter @stuhelper/web exec playwright test \
        tests/e2e/auth-flow.spec.ts \
        tests/e2e/user-verification.spec.ts \
        --workers=1 \
        --retries="${ADMISSION_MVP_PLAYWRIGHT_RETRIES}"
  )
}

run_web_build() {
  if [[ "${ADMISSION_MVP_SKIP_BUILD}" == "true" ]]; then
    log "skipping Web production build"
    return
  fi
  run_in "${ROOT_DIR}/clients" corepack pnpm --filter @stuhelper/web build
}

run_koishi_tests() {
  if [[ "${ADMISSION_MVP_SKIP_KOISHI}" == "true" ]]; then
    log "skipping Koishi group guard tests"
    return
  fi
  run_in "${ROOT_DIR}/bots/koishi" corepack yarn workspace koishi-plugin-stuhelper-group-guard test:unit
  run_in "${ROOT_DIR}/bots/koishi" corepack yarn build
}

run_infra_contracts() {
  if [[ "${ADMISSION_MVP_SKIP_INFRA_CONTRACTS}" == "true" ]]; then
    log "skipping admission infra contracts"
    return
  fi

  local contracts=(
    admission-public-smoke-contract.sh
    admission-production-readiness-contract.sh
    admission-reviewer-readiness-contract.sh
    admission-mvp-production-evidence-contract.sh
    admission-mvp-final-evidence-verify-contract.sh
    admission-join-e2e-evidence-contract.sh
    admission-join-e2e-wait-contract.sh
    koishi-admission-production-evidence-contract.sh
    prod-parity-contract.sh
    public-web-auth-browser-smoke-contract.sh
  )

  local contract
  for contract in "${contracts[@]}"; do
    run_in "${ROOT_DIR}" bash "infra/ops/tests/${contract}"
  done
}

run_server_tests
run_identity_dependency_server_tests
run_web_unit_tests
run_identity_dependency_web_unit_tests
run_web_browser_tests
run_identity_dependency_browser_tests
run_web_build
run_koishi_tests
run_infra_contracts

log "admission MVP local test suite passed"
