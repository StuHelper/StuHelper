#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
PREFLIGHT_FILE="${REPO_ROOT}/infra/ops/remote-preflight.sh"
COMMON_LIB_FILE="${REPO_ROOT}/infra/ops/lib/common.sh"

fail() {
  echo "[remote-preflight-contract][error] $*" >&2
  exit 1
}

assert_contains() {
  local file="$1"
  local pattern="$2"
  if ! grep -Eq "${pattern}" "${file}"; then
    fail "expected ${file} to contain pattern: ${pattern}"
  fi
}

assert_not_contains() {
  local file="$1"
  local pattern="$2"
  if grep -Eq "${pattern}" "${file}"; then
    fail "expected ${file} to not contain pattern: ${pattern}"
  fi
}

assert_contains "${PREFLIGHT_FILE}" 'compose run --rm --no-deps -T postgres'
assert_contains "${PREFLIGHT_FILE}" 'die "备份数据库连通性检查失败'
assert_not_contains "${PREFLIGHT_FILE}" 'docker run --rm --network host "\$\{pg_image\}"'
assert_contains "${COMMON_LIB_FILE}" 'failed to read generated secret env from \$\{SECRET_BACKEND\}'
assert_not_contains "${COMMON_LIB_FILE}" 'secret_backend_read_to_stdout "\$\{GENERATED_ENV_SECRET_REF\}" 2>/dev/null'

echo "[remote-preflight-contract] all assertions passed"
