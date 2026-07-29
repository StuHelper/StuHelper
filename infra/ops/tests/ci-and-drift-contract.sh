#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
GITHUB_CI_FILE="${REPO_ROOT}/.github/workflows/ci.yml"
GITHUB_PUBLISH_FILE="${REPO_ROOT}/.github/workflows/publish-images.yml"
SECRET_SCAN_SCRIPT="${REPO_ROOT}/scripts/check-secrets.sh"
ROOT_MAKEFILE="${REPO_ROOT}/Makefile"
SERVER_MAKEFILE="${REPO_ROOT}/server/Makefile"
WEB_DOCKERFILE="${REPO_ROOT}/clients/web/Dockerfile"
ADMIN_DOCKERFILE="${REPO_ROOT}/clients/admin/scripts/deploy/Dockerfile"
ADMIN_NGINX="${REPO_ROOT}/clients/admin/scripts/deploy/nginx.conf"
ADMIN_ENV_LOADER="${REPO_ROOT}/clients/admin/internal/vite-config/src/utils/env.ts"
ADMIN_TURBO="${REPO_ROOT}/clients/admin/turbo.json"
CLIENTS_DOCKERIGNORE="${REPO_ROOT}/clients/.dockerignore"
CLIENTS_PACKAGE="${REPO_ROOT}/clients/package.json"
CLIENTS_WORKSPACE="${REPO_ROOT}/clients/pnpm-workspace.yaml"
ADMIN_WORKSPACE="${REPO_ROOT}/clients/admin/pnpm-workspace.yaml"
BRACE_EXPANSION_PATCH="${REPO_ROOT}/clients/patches/npm/brace-expansion@5.0.8.patch"

fail() {
  echo "[ci-and-drift-contract][error] $*" >&2
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

[[ ! -e "${REPO_ROOT}/.gitlab-ci.yml" ]] ||
  fail "retired GitLab pipeline must not remain in the GitHub repository"
[[ ! -d "${REPO_ROOT}/.gitlab" ]] ||
  [[ -z "$(find "${REPO_ROOT}/.gitlab" -type f -print -quit)" ]] ||
  fail "retired GitLab pipeline fragments must not remain in the GitHub repository"
assert_contains "${GITHUB_CI_FILE}" 'DATABASE_URL: postgres://stuhelper_app:test@127\.0\.0\.1:5432/test'
assert_contains "${GITHUB_CI_FILE}" 'STUHELPER_TEST_POSTGRES_URL: postgres://test:test@127\.0\.0\.1:5432/postgres'
assert_contains "${GITHUB_CI_FILE}" 'CREATE ROLE stuhelper_app;'
assert_contains "${GITHUB_CI_FILE}" 'WITH LOGIN PASSWORD .* NOSUPERUSER NOCREATEDB NOCREATEROLE CONNECTION LIMIT 30;'
assert_contains "${GITHUB_CI_FILE}" 'ALTER DATABASE test OWNER TO stuhelper_app;'
assert_contains "${GITHUB_CI_FILE}" 'ALTER SCHEMA public OWNER TO stuhelper_app;'
assert_contains "${GITHUB_CI_FILE}" "^[[:space:]]+- '\\.github/\\*\\*'$"
assert_contains "${GITHUB_CI_FILE}" 'INSTALL_ADMIN: \$\{\{ matrix\.install_admin \}\}'
assert_contains "${GITHUB_CI_FILE}" 'if \[ "\$\{INSTALL_ADMIN\}" = "true" \]; then'
assert_not_contains "${GITHUB_CI_FILE}" 'if \[\[ "\$\{INSTALL_ADMIN\}"'
assert_not_contains "${GITHUB_CI_FILE}" 'if \[\[ "\$\{\{ matrix\.install_admin \}\}"'
infra_job_block="$(sed -n '/^  infra:$/,/^  runtime-image-security:$/p' "${GITHUB_CI_FILE}")"
if grep -Eq '^    container:' <<<"${infra_job_block}"; then
  fail "GitHub infrastructure contracts require the hosted runner Docker CLI and must not run in a job container"
fi
if ! grep -Eq '^[[:space:]]+sudo apt-get install --yes curl jq openssl$' <<<"${infra_job_block}"; then
  fail "GitHub infrastructure contracts must install host dependencies with sudo"
fi
if ! grep -Eq '^[[:space:]]+run: pnpm --filter @stuhelper/web exec playwright install --with-deps chromium$' <<<"${infra_job_block}"; then
  fail "GitHub infrastructure contracts must install the Chromium binary used by browser smoke contracts"
fi
publish_job_block="$(sed -n '/^  publish-images:$/,$p' "${GITHUB_CI_FILE}")"
if ! grep -Eq '^[[:space:]]+always\(\) &&$' <<<"${publish_job_block}"; then
  fail "GitHub image publishing must evaluate after upstream jobs are skipped"
fi
if ! grep -Eq '^[[:space:]]+!cancelled\(\) &&$' <<<"${publish_job_block}"; then
  fail "GitHub image publishing must stop when the workflow is cancelled"
fi
if ! grep -Eq "^[[:space:]]+needs\\.required\\.result == 'success' &&$" <<<"${publish_job_block}"; then
  fail "GitHub image publishing must require the aggregate CI gate to succeed"
fi
if ! grep -Eq '^[[:space:]]+artifact-metadata: write ' <<<"${publish_job_block}"; then
  fail "GitHub image publishing caller must grant artifact metadata write access"
fi
assert_contains "${GITHUB_PUBLISH_FILE}" '^[[:space:]]+artifact-metadata: write # Link each published digest to the organization artifact inventory$'
assert_contains "${GITHUB_CI_FILE}" 'pnpm audit --registry=https://registry\.npmjs\.org --audit-level=moderate$'
assert_contains "${GITHUB_CI_FILE}" 'pnpm --dir admin audit --registry=https://registry\.npmjs\.org --audit-level=moderate$'
assert_not_contains "${GITHUB_CI_FILE}" 'pnpm audit .* --prod'
assert_not_contains "${GITHUB_CI_FILE}" 'pnpm --dir admin audit .* --prod'
assert_contains "${GITHUB_CI_FILE}" 'YARN_NPM_REGISTRY_SERVER: https://registry\.npmjs\.org'
assert_contains "${GITHUB_CI_FILE}" 'corepack yarn npm audit --all --severity moderate$'
assert_contains "${GITHUB_CI_FILE}" 'pnpm run test:all$'
assert_contains "${GITHUB_CI_FILE}" 'git /repo$'
assert_contains "${GITHUB_CI_FILE}" '--gitleaks-ignore-path /repo/\.gitleaksignore'
assert_contains "${GITHUB_CI_FILE}" '--platform github'
assert_contains "${GITHUB_CI_FILE}" '^  dependency-review:$'
assert_contains "${GITHUB_CI_FILE}" 'actions/dependency-review-action@a1d282b36b6f3519aa1f3fc636f609c47dddb294 # v5\.0\.0'
assert_contains "${GITHUB_CI_FILE}" 'fail-on-severity: moderate'
assert_contains "${GITHUB_CI_FILE}" '^      - dependency-review$'
assert_contains "${SECRET_SCAN_SCRIPT}" '^gitleaks git "\$\{source_path\}"'
assert_contains "${SECRET_SCAN_SCRIPT}" '--gitleaks-ignore-path "\$\{source_path%/\}/\.gitleaksignore"'
assert_contains "${SECRET_SCAN_SCRIPT}" '--platform github'
assert_contains "${SECRET_SCAN_SCRIPT}" '--redact=100'
assert_not_contains "${SECRET_SCAN_SCRIPT}" 'gitleaks detect'
assert_contains "${CLIENTS_PACKAGE}" '"test:all": "pnpm run check:dependency-compat .*pnpm run test:admin"'
assert_contains "${CLIENTS_WORKSPACE}" 'brace-expansion@<5\.0\.8: 5\.0\.8'
assert_contains "${CLIENTS_WORKSPACE}" 'brace-expansion@5\.0\.8: patches/npm/brace-expansion@5\.0\.8\.patch'
assert_contains "${ADMIN_WORKSPACE}" 'brace-expansion@<5\.0\.8: 5\.0\.8'
assert_contains "${ADMIN_WORKSPACE}" 'brace-expansion@5\.0\.8: \.\./patches/npm/brace-expansion@5\.0\.8\.patch'
assert_contains "${BRACE_EXPANSION_PATCH}" 'module\.exports = Object\.assign\(expand, exports, \{ default: expand \}\);'
assert_contains "${BRACE_EXPANSION_PATCH}" '^\+export default expand;$'
assert_contains "${ROOT_MAKEFILE}" '^check-infra-contracts:$'
assert_contains "${ROOT_MAKEFILE}" 'bash infra/ops/tests/run-infra-contracts\.sh'
assert_contains "${GITHUB_PUBLISH_FILE}" '^[[:space:]]+context: \.$'
assert_contains "${GITHUB_PUBLISH_FILE}" '^[[:space:]]+file: clients/web/Dockerfile$'
assert_contains "${GITHUB_PUBLISH_FILE}" '^[[:space:]]+context: clients$'
assert_contains "${GITHUB_PUBLISH_FILE}" '^[[:space:]]+file: clients/admin/scripts/deploy/Dockerfile$'
assert_contains "${GITHUB_PUBLISH_FILE}" 'VITE_QQ_BOT_ENTRY=\$\{\{ vars\.WEB_VITE_QQ_BOT_ENTRY \}\}'
assert_contains "${GITHUB_PUBLISH_FILE}" "VITE_QQ_BIND_COMMAND=\\$\\{\\{ vars\\.WEB_VITE_QQ_BIND_COMMAND \\|\\| '绑定' \\}\\}"
assert_contains "${WEB_DOCKERFILE}" '^COPY clients/patches \./patches$'
assert_contains "${ADMIN_DOCKERFILE}" '^RUN corepack enable && corepack prepare pnpm@10\.32\.1 --activate$'
assert_contains "${ADMIN_DOCKERFILE}" '^ARG VITE_BASE=/admin/$'
assert_contains "${ADMIN_DOCKERFILE}" '^COPY patches /app/patches$'
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
assert_contains "${CLIENTS_DOCKERIGNORE}" '^\*\*/\.env$'
assert_contains "${CLIENTS_DOCKERIGNORE}" '^\*\*/\.env\.\*$'
assert_contains "${SERVER_MAKEFILE}" '^check-drift-ts: bundle-spec$'
assert_contains "${SERVER_MAKEFILE}" '^check-drift-capabilities:$'
assert_contains "${SERVER_MAKEFILE}" '^check-drift-all: check-bundled-drift check-drift-go check-drift-ts check-drift-capabilities$'
assert_contains "${SERVER_MAKEFILE}" '^[[:space:]]+@\$\(MIGRATE\) -path migrations -database "\$\(DATABASE_URL\)"'
assert_contains "${SERVER_MAKEFILE}" '^[[:space:]]+@psql "\$\(DATABASE_URL\)" -v ON_ERROR_STOP=1 -f scripts/seed\.sql$'
assert_not_contains "${SERVER_MAKEFILE}" '^[[:space:]]+\$\(MIGRATE\) -path migrations -database "\$\(DATABASE_URL\)"'
assert_not_contains "${SERVER_MAKEFILE}" '^[[:space:]]+psql "\$\(DATABASE_URL\)" -v ON_ERROR_STOP=1 -f scripts/seed\.sql$'

echo "[ci-and-drift-contract] all assertions passed"
