#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
BUNDLE_SCRIPT="${REPO_ROOT}/infra/ops/build-deploy-bundle.sh"

fail() {
  echo "[deploy-bundle-contract][error] $*" >&2
  exit 1
}

assert_contains() {
  local file="$1"
  local pattern="$2"
  if ! grep -Eq -- "${pattern}" "${file}"; then
    fail "expected ${file} to contain pattern: ${pattern}"
  fi
}

line_number() {
  local pattern="$1"
  local line
  line="$(grep -nE "${pattern}" "${BUNDLE_SCRIPT}" | head -n1 | cut -d: -f1)"
  [[ -n "${line}" ]] || fail "missing expected pattern in build-deploy-bundle.sh: ${pattern}"
  printf '%s\n' "${line}"
}

bash -n "${BUNDLE_SCRIPT}"

assert_contains "${BUNDLE_SCRIPT}" '^require_cmd git$'
assert_contains "${BUNDLE_SCRIPT}" 'rev-parse --is-inside-work-tree'
assert_contains "${BUNDLE_SCRIPT}" 'status --porcelain --untracked-files=all'
assert_contains "${BUNDLE_SCRIPT}" 'deployment bundle requires a clean git worktree'
assert_contains "${BUNDLE_SCRIPT}" "--exclude='\\.git'"
assert_contains "${BUNDLE_SCRIPT}" "--exclude='\\.run'"
assert_contains "${BUNDLE_SCRIPT}" "--exclude='\\.deploy'"
assert_contains "${BUNDLE_SCRIPT}" "--exclude='\\.env'"
assert_contains "${BUNDLE_SCRIPT}" "--exclude='\\.env\\.prod\\.shared'"
assert_contains "${BUNDLE_SCRIPT}" "--exclude='infra/generated/\\*'"
assert_contains "${BUNDLE_SCRIPT}" "--exclude='\\*\\*/node_modules'"
assert_contains "${BUNDLE_SCRIPT}" "--exclude='\\*\\*/dist'"

clean_check_line="$(line_number 'status --porcelain --untracked-files=all')"
mkdir_line="$(line_number 'mkdir -p "\$\{OUTPUT_DIR\}"')"
tar_line="$(line_number '^[[:space:]]*tar[[:space:]]+\\$')"

if (( clean_check_line >= mkdir_line )); then
  fail "clean worktree check must run before creating generated bundle output"
fi
if (( clean_check_line >= tar_line )); then
  fail "clean worktree check must run before tar packaging"
fi

echo "[deploy-bundle-contract] all assertions passed"
