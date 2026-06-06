#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
CI_FILE="${REPO_ROOT}/.gitlab-ci.yml"
ROOT_MAKEFILE="${REPO_ROOT}/Makefile"
SERVER_MAKEFILE="${REPO_ROOT}/server/Makefile"
ADMIN_DOCKERFILE="${REPO_ROOT}/clients/admin/scripts/deploy/Dockerfile"
ADMIN_NGINX="${REPO_ROOT}/clients/admin/scripts/deploy/nginx.conf"
ADMIN_ENV_LOADER="${REPO_ROOT}/clients/admin/internal/vite-config/src/utils/env.ts"
ADMIN_TURBO="${REPO_ROOT}/clients/admin/turbo.json"
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
assert_contains "${CI_FILE}" 'docker buildx build .*--build-arg "VITE_QQ_BOT_ENTRY=\$\{WEB_VITE_QQ_BOT_ENTRY:-StuHelper QQ Bot\}"'
assert_contains "${CI_FILE}" 'docker buildx build .*--build-arg "VITE_QQ_BIND_COMMAND=\$\{WEB_VITE_QQ_BIND_COMMAND:-绑定\}"'
assert_contains "${CI_FILE}" 'docker buildx build .*--file clients/admin/scripts/deploy/Dockerfile .* clients$'
assert_contains "${ADMIN_DOCKERFILE}" '^RUN corepack enable && corepack prepare pnpm@10\.32\.1 --activate$'
assert_contains "${ADMIN_DOCKERFILE}" '^ARG VITE_BASE=/admin/$'
assert_contains "${ADMIN_DOCKERFILE}" '^COPY shared /app/shared$'
assert_contains "${ADMIN_DOCKERFILE}" '^RUN pnpm --filter @stuhelper/shared build$'
assert_contains "${ADMIN_DOCKERFILE}" '^RUN pnpm --dir admin --filter @vben/vite-config stub$'
assert_contains "${ADMIN_ENV_LOADER}" 'Object\.entries\(process\.env\)'
assert_contains "${ADMIN_ENV_LOADER}" '\.\.\.envConfig,'
assert_contains "${ADMIN_TURBO}" '"globalEnv": \["NODE_ENV", "VITE_BASE", "VITE_GLOB_\*"\]'
assert_contains "${ADMIN_NGINX}" 'location = /admin/_app\.config\.js \{'
assert_contains "${ADMIN_NGINX}" 'add_header Cache-Control "no-store, no-cache, must-revalidate" always;'
assert_contains "${ADMIN_NGINX}" 'location /admin/js/ \{'
assert_contains "${ADMIN_NGINX}" 'location /admin/jse/ \{'
assert_contains "${ADMIN_NGINX}" 'location /admin/css/ \{'
assert_contains "${ADMIN_NGINX}" 'try_files \$uri =404;'
assert_not_contains "${CLIENTS_DOCKERIGNORE}" '^admin$'
assert_contains "${SERVER_MAKEFILE}" '^check-drift-ts: bundle-spec$'
assert_contains "${SERVER_MAKEFILE}" '^check-drift-capabilities:$'
assert_contains "${SERVER_MAKEFILE}" '^check-drift-all: check-bundled-drift check-drift-go check-drift-ts check-drift-capabilities$'

echo "[ci-and-drift-contract] all assertions passed"
