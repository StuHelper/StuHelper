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

refute_contains() {
  local file="$1"
  local pattern="$2"
  if grep -Eq -- "${pattern}" "${file}"; then
    fail "${file} must not contain pattern: ${pattern}"
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

# The bundle is built from the Git index so ignored secrets cannot reach a
# deployment target. A working-tree tar with an exclude list is not acceptable:
# an over-broad pattern silently drops the env templates that deploy requires,
# and an under-broad one ships secrets.
assert_contains "${BUNDLE_SCRIPT}" 'git -C "\$\{REPO_ROOT\}" archive --format=tar\.gz'
refute_contains "${BUNDLE_SCRIPT}" "--exclude='\\.env"

# The packaging path must prove the env templates survived, not assume it.
assert_contains "${BUNDLE_SCRIPT}" 'missing required env template'
assert_contains "${BUNDLE_SCRIPT}" 'unexpected env files'

clean_check_line="$(line_number 'status --porcelain --untracked-files=all')"
mkdir_line="$(line_number 'mkdir -p "\$\{OUTPUT_DIR\}"')"
archive_line="$(line_number 'git -C "\$\{REPO_ROOT\}" archive')"

if (( clean_check_line >= mkdir_line )); then
  fail "clean worktree check must run before creating generated bundle output"
fi
if (( clean_check_line >= archive_line )); then
  fail "clean worktree check must run before archiving"
fi

# Behavioural check: whatever the packaging path is, the archive it produces
# must carry both env templates and no other env file. ensure_env_file() in
# lib/common.sh fails hard when a template is absent, so a bundle without them
# breaks every remote deploy and rollback.
archive_listing="$(mktemp)"
trap 'rm -f "${archive_listing}"' EXIT
git -C "${REPO_ROOT}" archive --format=tar.gz HEAD | tar -tz | sed 's|^\./||' >"${archive_listing}"

bundled_env_files="$(grep -E '^\.env' "${archive_listing}" || true)"
for required in .env.example .env.prod.example; do
  grep -qxF "${required}" <<<"${bundled_env_files}" ||
    fail "bundle must include env template: ${required}"
done

unexpected="$(grep -vxE '\.env\.example|\.env\.prod\.example' <<<"${bundled_env_files}" | grep -v '^$' || true)"
if [[ -n "${unexpected}" ]]; then
  fail "bundle must not include env files beyond the templates: ${unexpected//$'\n'/ }"
fi

if grep -qE '(^|/)node_modules/' "${archive_listing}"; then
  fail "bundle must not include node_modules"
fi

echo "[deploy-bundle-contract] all assertions passed"
