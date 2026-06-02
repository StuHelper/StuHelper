#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
CMD_FILE="${REPO_ROOT}/server/cmd/backfill-user-hashes/main.go"
DOCKERFILE="${REPO_ROOT}/server/Dockerfile"
SCRIPT_FILE="${REPO_ROOT}/infra/ops/backfill-user-hashes.sh"
RELEASE_RUNBOOK="${REPO_ROOT}/docs/guides/release-runbook.md"

fail() {
  echo "[backfill-user-hashes-contract][error] $*" >&2
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

for file in "${CMD_FILE}" "${DOCKERFILE}" "${SCRIPT_FILE}" "${RELEASE_RUNBOOK}"; do
  [[ -f "${file}" ]] || fail "missing file: ${file}"
done

bash -n "${SCRIPT_FILE}"

assert_contains "${CMD_FILE}" 'Usage: backfill-user-hashes'
assert_contains "${CMD_FILE}" 'CountMissingUserHashes'
assert_contains "${CMD_FILE}" 'BackfillUserHashes'
assert_contains "${CMD_FILE}" 'DATABASE_URL is required'
assert_contains "${CMD_FILE}" 'HMAC_SECRET'
assert_contains "${CMD_FILE}" 'apply=false pending_user_hashes='
assert_contains "${CMD_FILE}" 'apply=true pending_user_hashes='
assert_not_contains "${CMD_FILE}" 'fmt[.]Printf.*DATABASE_URL'
assert_not_contains "${CMD_FILE}" 'fmt[.]Printf.*HMAC_SECRET'

assert_contains "${DOCKERFILE}" './cmd/backfill-user-hashes'
assert_contains "${DOCKERFILE}" '/app/backfill-user-hashes'

assert_contains "${SCRIPT_FILE}" '--dry-run'
assert_contains "${SCRIPT_FILE}" '--apply'
assert_contains "${SCRIPT_FILE}" 'USER_HASH_BACKFILL_APP_CONTAINER'
assert_contains "${SCRIPT_FILE}" 'docker exec "\$\{app_container\}"'
assert_not_contains "${SCRIPT_FILE}" 'echo .*DATABASE_URL'
assert_not_contains "${SCRIPT_FILE}" 'echo .*HMAC_SECRET'
assert_not_contains "${SCRIPT_FILE}" 'printf .*DATABASE_URL'
assert_not_contains "${SCRIPT_FILE}" 'printf .*HMAC_SECRET'

assert_contains "${RELEASE_RUNBOOK}" 'backfill-user-hashes.sh --dry-run'
assert_contains "${RELEASE_RUNBOOK}" 'backfill-user-hashes.sh --apply'
assert_contains "${RELEASE_RUNBOOK}" 'users with missing user_hash detected'

echo "[backfill-user-hashes-contract] all assertions passed"
