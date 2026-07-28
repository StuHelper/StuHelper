#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
DEV_STATUS="${REPO_ROOT}/infra/ops/dev-status.sh"

fail() {
  echo "[dev-status-contract][error] $*" >&2
  exit 1
}

assert_contains() {
  local pattern="$1"
  if ! grep -Eq "${pattern}" "${DEV_STATUS}"; then
    fail "expected dev-status.sh to contain pattern: ${pattern}"
  fi
}

assert_not_contains() {
  local pattern="$1"
  if grep -Eq "${pattern}" "${DEV_STATUS}"; then
    fail "expected dev-status.sh to omit pattern: ${pattern}"
  fi
}

[[ -f "${DEV_STATUS}" ]] || fail "missing file: ${DEV_STATUS}"
bash -n "${DEV_STATUS}"

assert_not_contains 'init-dev-env[.]sh'
assert_not_contains 'ensure_dev_runtime_dirs'
assert_not_contains 'ensure_env_file'
assert_not_contains 'ensure_generated_files'
assert_not_contains 'render-'
assert_not_contains 'compose ps'
assert_contains 'source_env_file "\$\{ENV_FILE\}"'
assert_contains 'source_env_file "\$\{GENERATED_ENV_FILE\}"'
assert_contains 'source_env_file "\$\{DEV_RUNTIME_ENV\}"'
assert_contains 'docker ps -a'
assert_contains 'label=com[.]docker[.]compose[.]project='

echo "[dev-status-contract] all assertions passed"
