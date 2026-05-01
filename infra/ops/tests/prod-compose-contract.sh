#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
COMPOSE_PROD_FILE="${REPO_ROOT}/docker-compose.prod.yml"
COMMON_LIB_FILE="${REPO_ROOT}/infra/ops/lib/common.sh"

fail() {
  echo "[prod-compose-contract][error] $*" >&2
  exit 1
}

assert_contains() {
  local file="$1"
  local pattern="$2"
  if ! grep -Eq "${pattern}" "${file}"; then
    fail "expected ${file} to contain pattern: ${pattern}"
  fi
}

app_block="$(
  awk '
    /^  app:/ { in_block=1; next }
    /^  frontend:/ { in_block=0 }
    in_block { print }
  ' "${COMPOSE_PROD_FILE}"
)"

[[ -n "${app_block}" ]] || fail "expected app service block in ${COMPOSE_PROD_FILE}"

if ! printf '%s\n' "${app_block}" | grep -Eq '\$\{SECRETS_ENV_FILE_PATH:-\.env\.prod\.secrets\.local\}'; then
  fail "app env_file must inject the production secrets env file"
fi

if ! printf '%s\n' "${app_block}" | grep -Eq '\$\{GENERATED_SECRET_ENV_FILE_PATH:-\.env\.prod\.generated\.secrets\}'; then
  fail "app env_file must inject the generated secret env file placeholder"
fi

assert_contains "${COMMON_LIB_FILE}" 'SECRETS_ENV_FILE_PATH="\$\{SECRETS_ENV_FILE\}"'
assert_contains "${COMMON_LIB_FILE}" 'GENERATED_SECRET_ENV_FILE_PATH="\$\{GENERATED_SECRET_ENV_FILE\}"'
assert_contains "${COMPOSE_PROD_FILE}" 'CASDOOR_CLIENT_SECRET: \$\{CASDOOR_CLIENT_SECRET:\?CASDOOR_CLIENT_SECRET is required\}'
assert_contains "${COMPOSE_PROD_FILE}" 'CASDOOR_APP_PROVISIONING_CLIENT_SECRET: \$\{CASDOOR_APP_PROVISIONING_CLIENT_SECRET:\?CASDOOR_APP_PROVISIONING_CLIENT_SECRET is required\}'
assert_contains "${COMPOSE_PROD_FILE}" 'CASDOOR_ROLE_SYNC_CLIENT_SECRET: \$\{CASDOOR_ROLE_SYNC_CLIENT_SECRET:\?CASDOOR_ROLE_SYNC_CLIENT_SECRET is required\}'
assert_contains "${COMPOSE_PROD_FILE}" 'CASDOOR_USER_LOOKUP_CLIENT_SECRET: \$\{CASDOOR_USER_LOOKUP_CLIENT_SECRET:\?CASDOOR_USER_LOOKUP_CLIENT_SECRET is required\}'

echo "[prod-compose-contract] all assertions passed"
