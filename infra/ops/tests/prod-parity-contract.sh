#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
PARITY_COMPOSE="${REPO_ROOT}/docker-compose.prod-parity-postgres.yml"
INIT_SHARED_PG="${REPO_ROOT}/infra/ops/init-shared-postgres.sh"
PARITY_UP="${REPO_ROOT}/infra/ops/prod-parity-up.sh"
PARITY_DOWN="${REPO_ROOT}/infra/ops/prod-parity-down.sh"
PARITY_SMOKE="${REPO_ROOT}/infra/ops/prod-parity-smoke.sh"
MAKEFILE="${REPO_ROOT}/Makefile"

fail() {
  echo "[prod-parity-contract][error] $*" >&2
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

for file in "${PARITY_COMPOSE}" "${INIT_SHARED_PG}" "${PARITY_UP}" "${PARITY_DOWN}" "${PARITY_SMOKE}"; do
  [[ -f "${file}" ]] || fail "missing file: ${file}"
done

bash -n "${INIT_SHARED_PG}" "${PARITY_UP}" "${PARITY_DOWN}" "${PARITY_SMOKE}"

assert_contains "${PARITY_COMPOSE}" '^  postgres:'
assert_contains "${PARITY_COMPOSE}" 'POSTGRES_PASSWORD: \$\{SHARED_POSTGRES_PASSWORD:\?SHARED_POSTGRES_PASSWORD is required\}'
assert_contains "${PARITY_COMPOSE}" 'PROD_PARITY_POSTGRES_PORT:-15432'
assert_contains "${PARITY_COMPOSE}" 'name: \$\{EXTERNAL_DATASTORE_NETWORK:-stuhelper-prod-parity-baota-net\}'
assert_contains "${PARITY_COMPOSE}" 'aliases:'
assert_contains "${PARITY_COMPOSE}" 'postgres'
assert_not_contains "${PARITY_COMPOSE}" '^  redis:'

assert_contains "${INIT_SHARED_PG}" 'STUHELPER_APP_DB_PASSWORD'
assert_contains "${INIT_SHARED_PG}" 'OPENFGA_DB_PASSWORD'
assert_contains "${INIT_SHARED_PG}" 'CREATE DATABASE %I OWNER %I'
assert_contains "${INIT_SHARED_PG}" 'ALTER ROLE %I WITH LOGIN PASSWORD %L'
assert_contains "${INIT_SHARED_PG}" 'GRANT pg_read_all_data, pg_read_all_settings, pg_read_all_stats'
assert_contains "${INIT_SHARED_PG}" 'GRANT USAGE, CREATE ON SCHEMA public'
assert_contains "${INIT_SHARED_PG}" 'openfga_database'

assert_contains "${PARITY_UP}" 'EXTERNAL_POSTGRES_ENABLED.*true'
assert_contains "${PARITY_UP}" 'EXTERNAL_POSTGRES_ALLOW_PLAINTEXT.*true'
assert_contains "${PARITY_UP}" 'EXTERNAL_DATASTORE_NETWORK.*stuhelper-prod-parity-baota-net'
assert_contains "${PARITY_UP}" 'parity_default_path'
assert_contains "${PARITY_UP}" '\.run/prod-parity'
assert_contains "${PARITY_UP}" 'init-shared-postgres.sh'
assert_contains "${PARITY_UP}" 'render-redis-tls.sh'
assert_contains "${PARITY_UP}" 'render-redis-acl.sh'
assert_contains "${PARITY_UP}" 'render-observability.sh.*prod'
assert_contains "${PARITY_UP}" 'docker build'
assert_contains "${PARITY_UP}" 'compose --profile prod up -d --wait'
assert_contains "${PARITY_UP}" 'bootstrap-platform.sh" dev'
assert_contains "${PARITY_UP}" 'prod-parity-smoke.sh'
assert_contains "${PARITY_UP}" 'IDENTITY_SIGNING_PRIVATE_KEY_PEM=%s'
assert_not_contains "${PARITY_UP}" 'prod-deploy.sh'

assert_contains "${PARITY_DOWN}" 'parity_default_path'
assert_contains "${PARITY_SMOKE}" 'identity-public-smoke.sh'
assert_contains "${PARITY_SMOKE}" 'parity_default_path'
assert_contains "${PARITY_SMOKE}" 'IDENTITY_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS=true'
assert_contains "${PARITY_SMOKE}" 'openfga-resource-access-smoke.sh'
assert_contains "${PARITY_SMOKE}" 'observability-smoke-check.sh'
assert_contains "${PARITY_SMOKE}" 'OBS_SMOKE_STRICT=false'

assert_contains "${MAKEFILE}" 'prod-parity-up'
assert_contains "${MAKEFILE}" 'prod-parity-down'
assert_contains "${MAKEFILE}" 'prod-parity-smoke'

echo "[prod-parity-contract] all assertions passed"
