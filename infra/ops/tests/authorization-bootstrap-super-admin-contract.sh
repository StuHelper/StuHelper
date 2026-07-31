#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
BOOTSTRAP_SCRIPT="${REPO_ROOT}/infra/ops/authorization-bootstrap-super-admin.sh"
PROD_ENV_EXAMPLE="${REPO_ROOT}/.env.prod.example"
RUNBOOK="${REPO_ROOT}/docs/guides/release-runbook.md"
DOCKERFILE="${REPO_ROOT}/server/Dockerfile"
DEPLOY_SCRIPT="${REPO_ROOT}/infra/ops/prod-deploy.sh"

fail() {
  echo "[authorization-bootstrap-super-admin-contract][error] $*" >&2
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
    fail "expected ${file} not to contain pattern: ${pattern}"
  fi
}

for file in "${BOOTSTRAP_SCRIPT}" "${PROD_ENV_EXAMPLE}" "${RUNBOOK}" "${DOCKERFILE}" "${DEPLOY_SCRIPT}"; do
  [[ -f "${file}" ]] || fail "missing file: ${file}"
done

bash -n "${BOOTSTRAP_SCRIPT}"
[[ -x "${BOOTSTRAP_SCRIPT}" ]] || fail "bootstrap script must be executable"

assert_contains "${BOOTSTRAP_SCRIPT}" 'PostgreSQL authorization control plane'
assert_contains "${BOOTSTRAP_SCRIPT}" 'STUHELPER_INITIAL_SUPER_ADMINS'
assert_contains "${BOOTSTRAP_SCRIPT}" '/app/authorization-bootstrap'
assert_contains "${BOOTSTRAP_SCRIPT}" '--apply'
assert_contains "${BOOTSTRAP_SCRIPT}" '--env STUHELPER_INITIAL_SUPER_ADMINS'
assert_not_contains "${BOOTSTRAP_SCRIPT}" 'public\.role'
assert_not_contains "${BOOTSTRAP_SCRIPT}" 'CASDOOR_ROLE'

assert_contains "${DOCKERFILE}" '/app/bin/authorization-bootstrap'
assert_contains "${DOCKERFILE}" '/app/authorization-bootstrap'
assert_contains "${DEPLOY_SCRIPT}" 'authorization-bootstrap-super-admin\.sh'
assert_contains "${PROD_ENV_EXAMPLE}" '^STUHELPER_INITIAL_SUPER_ADMINS=REPLACE_WITH_INITIAL_SUPER_ADMIN_USERNAMES$'

assert_contains "${RUNBOOK}" 'authorization-bootstrap-super-admin\.sh'
assert_contains "${RUNBOOK}" 'authorization_grants'
assert_contains "${RUNBOOK}" 'Casdoor.*不'

echo "[authorization-bootstrap-super-admin-contract] all assertions passed"
