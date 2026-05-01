#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
BOOTSTRAP_SCRIPT="${REPO_ROOT}/infra/ops/bootstrap-platform.sh"
ZITADEL_SETUP_SCRIPT="${REPO_ROOT}/infra/zitadel/setup.sh"
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

assert_contains "${BOOTSTRAP_SCRIPT}" 'ZITADEL_SECRET_OUTPUT_FILE="\$\{zitadel_secret_tmp\}"'
if grep -Fqx "printf '%s\\n' \"\${ZITADEL_OUTPUT}\"" "${BOOTSTRAP_SCRIPT}"; then
  fail "expected ${BOOTSTRAP_SCRIPT} to not print raw ZITADEL_OUTPUT"
fi
if grep -Fqx "printf '%s\\n' \"\${FGA_OUTPUT}\"" "${BOOTSTRAP_SCRIPT}"; then
  fail "expected ${BOOTSTRAP_SCRIPT} to not print raw FGA_OUTPUT"
fi

assert_contains "${ZITADEL_SETUP_SCRIPT}" 'ZITADEL_SECRET_OUTPUT_FILE='
assert_not_contains "${ZITADEL_SETUP_SCRIPT}" '^echo "ZITADEL_CLIENT_SECRET='

assert_contains "${COMPOSE_PROD_FILE}" 'CASDOOR_CLIENT_SECRET: \$\{CASDOOR_CLIENT_SECRET:\?CASDOOR_CLIENT_SECRET is required\}'
assert_contains "${COMPOSE_PROD_FILE}" 'CASDOOR_ROLE_SYNC_CLIENT_SECRET: \$\{CASDOOR_ROLE_SYNC_CLIENT_SECRET:\?CASDOOR_ROLE_SYNC_CLIENT_SECRET is required\}'
assert_contains "${COMPOSE_PROD_FILE}" 'CASDOOR_USER_LOOKUP_CLIENT_SECRET: \$\{CASDOOR_USER_LOOKUP_CLIENT_SECRET:\?CASDOOR_USER_LOOKUP_CLIENT_SECRET is required\}'

echo "[bootstrap-platform-contract] all assertions passed"
