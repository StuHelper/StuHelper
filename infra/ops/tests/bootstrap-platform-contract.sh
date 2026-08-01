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

line_number() {
  local pattern="$1"
  local line
  line="$(grep -nE "${pattern}" "${BOOTSTRAP_SCRIPT}" | head -n1 | cut -d: -f1)"
  [[ -n "${line}" ]] || fail "missing bootstrap-platform pattern: ${pattern}"
  printf '%s\n' "${line}"
}

assert_contains "${BOOTSTRAP_SCRIPT}" 'CASDOOR_CLIENT_ID must be configured before platform bootstrap'
assert_contains "${BOOTSTRAP_SCRIPT}" 'go run \./cmd/casdoor-bootstrap'
assert_contains "${BOOTSTRAP_SCRIPT}" 'CASDOOR_BOOTSTRAP_MODE=applications-only'
assert_contains "${BOOTSTRAP_SCRIPT}" 'retrying applications-only bootstrap with app provisioning credentials'
assert_contains "${BOOTSTRAP_SCRIPT}" 'CASDOOR_BOOTSTRAP_CLIENT_SECRET'
assert_contains "${BOOTSTRAP_SCRIPT}" 'CASDOOR_BOOTSTRAP_ORGANIZATION'
assert_contains "${BOOTSTRAP_SCRIPT}" 'SMS_INTERNAL_KEY'
assert_contains "${BOOTSTRAP_SCRIPT}" 'CASDOOR_ADMIN_CLIENT_SECRET'
assert_contains "${BOOTSTRAP_SCRIPT}" 'CASDOOR_INTROSPECTION_CLIENT_SECRET'
assert_contains "${BOOTSTRAP_SCRIPT}" 'CASDOOR_USER_PROFILE_CLIENT_SECRET'
assert_contains "${BOOTSTRAP_SCRIPT}" 'Casdoor bootstrap skipped because CASDOOR_BOOTSTRAP_ENABLED is not true'
assert_contains "${BOOTSTRAP_SCRIPT}" 'casdoor_bootstrap_endpoint'
assert_contains "${BOOTSTRAP_SCRIPT}" 'CASDOOR_ISSUER:-http://localhost:8085'
assert_contains "${BOOTSTRAP_SCRIPT}" 'CALLER_CASDOOR_BOOTSTRAP_ENABLED'
assert_contains "${BOOTSTRAP_SCRIPT}" 'CALLER_OPENFGA_BOOTSTRAP_API_URL'
assert_contains "${BOOTSTRAP_SCRIPT}" 'CALLER_OPENFGA_BOOTSTRAP_DATABASE_URL'
assert_contains "${BOOTSTRAP_SCRIPT}" 'OPENFGA_BOOTSTRAP_API_URL:-\$\{OPENFGA_API_URL:-http://localhost:8081\}'
assert_contains "${BOOTSTRAP_SCRIPT}" 'DATABASE_URL="\$\{OPENFGA_BOOTSTRAP_DATABASE_URL:-\$\{DATABASE_URL:-\}\}"'
assert_contains "${BOOTSTRAP_SCRIPT}" 'reject_placeholder_if_set OPENFGA_STORE_ID "\$\{OPENFGA_STORE_ID:-\}" "REPLACE_WITH_OPENFGA_STORE_ID"'
assert_contains "${BOOTSTRAP_SCRIPT}" 'reject_placeholder_if_set OPENFGA_MODEL_ID "\$\{OPENFGA_MODEL_ID:-\}" "REPLACE_WITH_OPENFGA_MODEL_ID"'
assert_contains "${BOOTSTRAP_SCRIPT}" 'set -a'
casdoor_required_line="$(line_number '^if casdoor_bootstrap_required; then$')"
casdoor_wait_line="$(line_number '^[[:space:]]+wait_for_casdoor$')"
openfga_store_placeholder_line="$(line_number 'reject_placeholder_if_set OPENFGA_STORE_ID')"
openfga_bootstrap_line="$(line_number '^log "bootstrapping OpenFGA store and model"$')"
if (( casdoor_wait_line <= casdoor_required_line )); then
  fail "wait_for_casdoor must only run inside the Casdoor bootstrap-required branch"
fi
if (( openfga_store_placeholder_line >= openfga_bootstrap_line )); then
  fail "OpenFGA placeholder IDs must be rejected before fga-setup runs"
fi
retired_idp_prefix='ZITA''DEL_'
assert_not_contains "${BOOTSTRAP_SCRIPT}" "${retired_idp_prefix}"
if grep -Fqx "printf '%s\\n' \"\${FGA_OUTPUT}\"" "${BOOTSTRAP_SCRIPT}"; then
  fail "expected ${BOOTSTRAP_SCRIPT} to not print raw FGA_OUTPUT"
fi

assert_contains "${COMPOSE_PROD_FILE}" 'CASDOOR_CLIENT_SECRET: \$\{CASDOOR_CLIENT_SECRET:\?CASDOOR_CLIENT_SECRET is required\}'
assert_contains "${COMPOSE_PROD_FILE}" 'CASDOOR_APP_PROVISIONING_CLIENT_SECRET: \$\{CASDOOR_APP_PROVISIONING_CLIENT_SECRET:\?CASDOOR_APP_PROVISIONING_CLIENT_SECRET is required\}'
assert_contains "${COMPOSE_PROD_FILE}" 'CASDOOR_USER_PROFILE_CLIENT_SECRET: \$\{CASDOOR_USER_PROFILE_CLIENT_SECRET:\?CASDOOR_USER_PROFILE_CLIENT_SECRET is required\}'
assert_contains "${COMPOSE_PROD_FILE}" 'CASDOOR_INTROSPECTION_CLIENT_SECRET: \$\{CASDOOR_INTROSPECTION_CLIENT_SECRET:\?CASDOOR_INTROSPECTION_CLIENT_SECRET is required\}'
assert_not_contains "${BOOTSTRAP_SCRIPT}" 'CASDOOR_ROLE_SYNC'
assert_not_contains "${COMPOSE_PROD_FILE}" 'CASDOOR_ROLE_SYNC'
assert_contains "${COMPOSE_PROD_FILE}" 'CASDOOR_USER_LOOKUP_CLIENT_SECRET: \$\{CASDOOR_USER_LOOKUP_CLIENT_SECRET:\?CASDOOR_USER_LOOKUP_CLIENT_SECRET is required\}'

echo "[bootstrap-platform-contract] all assertions passed"
