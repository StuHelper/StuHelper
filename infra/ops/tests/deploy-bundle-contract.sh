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
assert_contains "${BUNDLE_SCRIPT}" 'SOURCE_REPO_ROOT="\$\{2:-\$\{REPO_ROOT\}\}"'
assert_contains "${BUNDLE_SCRIPT}" 'git -C "\$\{SOURCE_REPO_ROOT\}" archive --format=tar\.gz'
refute_contains "${BUNDLE_SCRIPT}" "--exclude='\\.env"

# The packaging path must prove the env templates survived, not assume it.
assert_contains "${BUNDLE_SCRIPT}" 'missing required env template'
assert_contains "${BUNDLE_SCRIPT}" 'unexpected env files'

clean_check_line="$(line_number 'status --porcelain --untracked-files=all')"
mkdir_line="$(line_number 'mkdir -p "\$\{OUTPUT_DIR\}"')"
archive_line="$(line_number 'git -C "\$\{SOURCE_REPO_ROOT\}" archive')"

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
fixture_root="$(mktemp -d)"
trap 'rm -f "${archive_listing}"; rm -rf "${fixture_root}"' EXIT
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

source_repo="${fixture_root}/rollback-release"
fixture_bundle="${fixture_root}/rollback-release.tgz"
mkdir -p "${source_repo}"
git -C "${source_repo}" init -q
printf 'shared-template\n' >"${source_repo}/.env.example"
printf 'production-template\n' >"${source_repo}/.env.prod.example"
printf 'historical-release\n' >"${source_repo}/release-marker.txt"
git -C "${source_repo}" add .env.example .env.prod.example release-marker.txt
git -C "${source_repo}" \
  -c user.name=contract \
  -c user.email=contract@example.test \
  commit -qm "fixture"

"${BUNDLE_SCRIPT}" "${fixture_bundle}" "${source_repo}" >/dev/null
tar -tzf "${fixture_bundle}" | grep -qxF 'release-marker.txt' ||
  fail "an explicit source worktree must be archived instead of the controller worktree"

printf 'dirty\n' >"${source_repo}/untracked.txt"
if "${BUNDLE_SCRIPT}" "${fixture_root}/dirty.tgz" "${source_repo}" >/dev/null 2>&1; then
  fail "an explicit source worktree must still fail closed when dirty"
fi
[[ ! -e "${fixture_root}/dirty.tgz" ]] ||
  fail "a rejected explicit source worktree must not publish a bundle"

echo "[deploy-bundle-contract] all assertions passed"
