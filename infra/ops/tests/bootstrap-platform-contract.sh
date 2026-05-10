#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
BOOTSTRAP_SCRIPT="${REPO_ROOT}/infra/ops/bootstrap-platform.sh"
COMPOSE_PROD_FILE="${REPO_ROOT}/docker-compose.prod.yml"

fail() {
  echo "[bootstrap-platform-contract][error] $*" >&2
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

assert_contains "${BOOTSTRAP_SCRIPT}" 'CASDOOR_CLIENT_ID must be configured before platform bootstrap'
assert_contains "${BOOTSTRAP_SCRIPT}" 'go run \./cmd/casdoor-bootstrap'
assert_contains "${BOOTSTRAP_SCRIPT}" 'CASDOOR_BOOTSTRAP_CLIENT_SECRET'
assert_contains "${BOOTSTRAP_SCRIPT}" 'SMS_INTERNAL_KEY'
assert_contains "${BOOTSTRAP_SCRIPT}" 'CASDOOR_ADMIN_CLIENT_SECRET'
assert_contains "${BOOTSTRAP_SCRIPT}" 'CASDOOR_INTROSPECTION_CLIENT_SECRET'
assert_contains "${BOOTSTRAP_SCRIPT}" 'Casdoor bootstrap skipped because CASDOOR_BOOTSTRAP_ENABLED is not true'
assert_contains "${BOOTSTRAP_SCRIPT}" 'casdoor_bootstrap_endpoint'
assert_contains "${BOOTSTRAP_SCRIPT}" 'CASDOOR_ISSUER:-http://localhost:8085'
assert_contains "${BOOTSTRAP_SCRIPT}" 'CALLER_CASDOOR_BOOTSTRAP_ENABLED'
assert_contains "${BOOTSTRAP_SCRIPT}" 'set -a'
retired_idp_prefix='ZITA''DEL_'
assert_not_contains "${BOOTSTRAP_SCRIPT}" "${retired_idp_prefix}"
if grep -Fqx "printf '%s\\n' \"\${FGA_OUTPUT}\"" "${BOOTSTRAP_SCRIPT}"; then
  fail "expected ${BOOTSTRAP_SCRIPT} to not print raw FGA_OUTPUT"
fi

assert_contains "${COMPOSE_PROD_FILE}" 'CASDOOR_CLIENT_SECRET: \$\{CASDOOR_CLIENT_SECRET:\?CASDOOR_CLIENT_SECRET is required\}'
assert_contains "${COMPOSE_PROD_FILE}" 'CASDOOR_APP_PROVISIONING_CLIENT_SECRET: \$\{CASDOOR_APP_PROVISIONING_CLIENT_SECRET:\?CASDOOR_APP_PROVISIONING_CLIENT_SECRET is required\}'
assert_contains "${COMPOSE_PROD_FILE}" 'CASDOOR_INTROSPECTION_CLIENT_SECRET: \$\{CASDOOR_INTROSPECTION_CLIENT_SECRET:\?CASDOOR_INTROSPECTION_CLIENT_SECRET is required\}'
assert_contains "${COMPOSE_PROD_FILE}" 'CASDOOR_ROLE_SYNC_CLIENT_SECRET: \$\{CASDOOR_ROLE_SYNC_CLIENT_SECRET:\?CASDOOR_ROLE_SYNC_CLIENT_SECRET is required\}'
assert_contains "${COMPOSE_PROD_FILE}" 'CASDOOR_USER_LOOKUP_CLIENT_SECRET: \$\{CASDOOR_USER_LOOKUP_CLIENT_SECRET:\?CASDOOR_USER_LOOKUP_CLIENT_SECRET is required\}'

echo "[bootstrap-platform-contract] all assertions passed"
