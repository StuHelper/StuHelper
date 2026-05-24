#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
CI_FILE="${REPO_ROOT}/.gitlab-ci.yml"
ROOT_MAKEFILE="${REPO_ROOT}/Makefile"
SERVER_MAKEFILE="${REPO_ROOT}/server/Makefile"
ADMIN_DOCKERFILE="${REPO_ROOT}/clients/admin/scripts/deploy/Dockerfile"
CLIENTS_DOCKERIGNORE="${REPO_ROOT}/clients/.dockerignore"

fail() {
  echo "[ci-and-drift-contract][error] $*" >&2
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

stage_line() {
  local stage="$1"
  local line
  line="$(grep -nE "^- ${stage}$" "${CI_FILE}" | head -n1 | cut -d: -f1)"
  [[ -n "${line}" ]] || fail "missing GitLab stage: ${stage}"
  printf '%s\n' "${line}"
}

build_line="$(stage_line build)"
scan_line="$(stage_line package_scan)"
package_line="$(stage_line package)"

if (( scan_line <= build_line )); then
  fail "container scan stage must run after build"
fi
if (( package_line <= scan_line )); then
  fail "package stage must run after container scans"
fi

assert_contains "${CI_FILE}" '^[[:space:]]*stage: package_scan$'
assert_contains "${CI_FILE}" 'apt-get install -y curl jq nodejs openssl'
assert_contains "${CI_FILE}" 'bash infra/ops/tests/run-infra-contracts\.sh'
assert_contains "${CI_FILE}" '^koishi_test:$'
assert_contains "${CI_FILE}" 'image: mcr\.microsoft\.com/playwright:v1\.58\.2-noble'
assert_contains "${CI_FILE}" 'bots/koishi/playwright-report'
assert_contains "${CI_FILE}" 'bots/koishi/test-results'
assert_contains "${ROOT_MAKEFILE}" '^check-infra-contracts:$'
assert_contains "${ROOT_MAKEFILE}" 'bash infra/ops/tests/run-infra-contracts\.sh'
assert_contains "${CI_FILE}" 'docker buildx build .*--file clients/web/Dockerfile .* \.$'
assert_contains "${CI_FILE}" 'docker buildx build .*--file clients/admin/scripts/deploy/Dockerfile .* clients$'
assert_contains "${ADMIN_DOCKERFILE}" '^RUN corepack enable && corepack prepare pnpm@10\.32\.1 --activate$'
assert_contains "${ADMIN_DOCKERFILE}" '^COPY shared /app/shared$'
assert_contains "${ADMIN_DOCKERFILE}" '^RUN pnpm --filter @stuhelper/shared build$'
assert_not_contains "${CLIENTS_DOCKERIGNORE}" '^admin$'
assert_contains "${SERVER_MAKEFILE}" '^check-drift-ts: bundle-spec$'
assert_contains "${SERVER_MAKEFILE}" '^check-drift-capabilities:$'
assert_contains "${SERVER_MAKEFILE}" '^check-drift-all: check-drift-go check-drift-ts check-drift-capabilities$'

echo "[ci-and-drift-contract] all assertions passed"
