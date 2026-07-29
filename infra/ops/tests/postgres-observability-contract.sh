#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
BASE_COMPOSE="${REPO_ROOT}/docker-compose.yml"
OBS_COMPOSE="${REPO_ROOT}/docker-compose.observability.yml"
PROD_COMPOSE="${REPO_ROOT}/docker-compose.prod.yml"
EXTERNAL_COMPOSE="${REPO_ROOT}/docker-compose.external-datastore.yml"
INIT_SQL="${REPO_ROOT}/infra/postgres/init-extra-dbs.sh"
DEV_HBA="${REPO_ROOT}/infra/postgres/pg_hba.conf"
PROD_HBA="${REPO_ROOT}/infra/postgres/pg_hba.prod.conf"
ENSURE_ROLE="${REPO_ROOT}/infra/ops/ensure-postgres-monitoring-role.sh"
SHARED_INIT="${REPO_ROOT}/infra/ops/init-shared-postgres.sh"
OBS_SMOKE="${REPO_ROOT}/infra/ops/observability-smoke-check.sh"
TLS_ENTRYPOINT="${REPO_ROOT}/infra/postgres/docker-entrypoint-with-tls.sh"
REDIS_ENTRYPOINT="${REPO_ROOT}/infra/redis/docker-entrypoint-with-secrets.sh"
CLIENT_CA_PREPARE="${REPO_ROOT}/infra/ops/prepare-datastore-client-cas.sh"

fail() {
  printf '[postgres-observability-contract][error] %s\n' "$*" >&2
  exit 1
}

assert_contains() {
  local file="$1"
  local expected="$2"
  grep -Fq -- "${expected}" "${file}" ||
    fail "expected ${file} to contain: ${expected}"
}

assert_not_contains() {
  local file="$1"
  local forbidden="$2"
  if grep -Fq -- "${forbidden}" "${file}"; then
    fail "expected ${file} to not contain: ${forbidden}"
  fi
}

for script in "${INIT_SQL}" "${ENSURE_ROLE}" "${SHARED_INIT}" "${OBS_SMOKE}"; do
  bash -n "${script}"
done
sh -n "${TLS_ENTRYPOINT}"
sh -n "${REDIS_ENTRYPOINT}"
bash -n "${CLIENT_CA_PREPARE}"
[[ -x "${ENSURE_ROLE}" ]] || fail "ensure-postgres-monitoring-role.sh must be executable"
[[ -x "${TLS_ENTRYPOINT}" ]] || fail "PostgreSQL TLS entrypoint must be executable"
[[ -x "${REDIS_ENTRYPOINT}" ]] || fail "Redis secret entrypoint must be executable"
[[ -x "${CLIENT_CA_PREPARE}" ]] || fail "datastore client CA preparer must be executable"

assert_contains "${BASE_COMPOSE}" 'POSTGRES_EXPORTER_DB_PASSWORD: ${POSTGRES_EXPORTER_DB_PASSWORD:?POSTGRES_EXPORTER_DB_PASSWORD is required}'
assert_contains "${BASE_COMPOSE}" 'entrypoint: ["/usr/local/bin/stuhelper-postgres-entrypoint"]'
assert_contains "${BASE_COMPOSE}" './infra/generated/postgres:/tls-source:ro'
assert_contains "${BASE_COMPOSE}" '/tls:mode=0700,uid=70,gid=70'
assert_contains "${TLS_ENTRYPOINT}" 'install -o "${postgres_uid}" -g "${postgres_gid}" -m 0600'
assert_contains "${TLS_ENTRYPOINT}" 'exec "${upstream_entrypoint}" "$@"'
assert_contains "${BASE_COMPOSE}" 'entrypoint: ["/usr/local/bin/stuhelper-redis-entrypoint"]'
assert_contains "${BASE_COMPOSE}" './infra/generated/redis:/redis-source:ro'
assert_contains "${BASE_COMPOSE}" '/redis-runtime:mode=0700,uid=999,gid=1000'
assert_contains "${REDIS_ENTRYPOINT}" 'install -o "${redis_uid}" -g "${redis_gid}" -m 0600'
assert_contains "${REPO_ROOT}/infra/ops/render-redis-tls.sh" 'chmod 600 "${SERVER_KEY}"'
assert_contains "${REPO_ROOT}/infra/ops/render-redis-acl.sh" 'chmod 600 "${acl_tmp}"'
assert_contains "${REPO_ROOT}/infra/ops/render-redis-acl.sh" 'mv -f "${acl_tmp}" "${ACL_FILE}"'
assert_contains "${CLIENT_CA_PREPARE}" 'POSTGRES_CLIENT_CA_HOST_PATH is required for external PostgreSQL TLS'
assert_contains "${CLIENT_CA_PREPARE}" 'client CA source must contain certificates only, not a private key'

[[ "$(grep -Fc -- './infra/generated/postgres:/tls-source:ro' "${BASE_COMPOSE}")" -eq 1 ]] ||
  fail "only the PostgreSQL service may mount the PostgreSQL server TLS source"
[[ "$(grep -Fc -- './infra/generated/redis:/redis-source:ro' "${BASE_COMPOSE}")" -eq 1 ]] ||
  fail "only the Redis service may mount the Redis server secret source"
assert_not_contains "${PROD_COMPOSE}" './infra/generated/postgres:/tls:ro'
assert_not_contains "${PROD_COMPOSE}" './infra/generated/redis:/redis-tls:ro'
assert_not_contains "${OBS_COMPOSE}" './infra/generated/postgres:/tls:ro'
assert_not_contains "${OBS_COMPOSE}" './infra/generated/redis:/tls:ro'
assert_contains "${PROD_COMPOSE}" './infra/generated/postgres-client-ca:/tls:ro'
assert_contains "${PROD_COMPOSE}" './infra/generated/redis-client-ca:/redis-tls:ro'
assert_contains "${OBS_COMPOSE}" 'DATA_SOURCE_URI: postgres:5432/postgres?sslmode=${POSTGRES_INTERNAL_SSL_MODE:-disable}&sslrootcert=/tls/ca.crt'
assert_contains "${OBS_COMPOSE}" 'DATA_SOURCE_USER: stuhelper_metrics'
assert_contains "${OBS_COMPOSE}" 'DATA_SOURCE_PASS: ${POSTGRES_EXPORTER_DB_PASSWORD:?POSTGRES_EXPORTER_DB_PASSWORD is required}'
assert_not_contains "${OBS_COMPOSE}" 'DATA_SOURCE_USER: stuhelper_backup'
assert_not_contains "${OBS_COMPOSE}" 'DATA_SOURCE_PASS: ${STUHELPER_BACKUP_DB_PASSWORD}'
assert_contains "${PROD_COMPOSE}" 'DATA_SOURCE_URI: postgres:5432/postgres?sslmode=${POSTGRES_INTERNAL_SSL_MODE:-verify-full}&sslrootcert=/tls/ca.crt'
assert_contains "${EXTERNAL_COMPOSE}" 'DATA_SOURCE_URI: ${POSTGRES_EXPORTER_DATA_SOURCE_URI:-${POSTGRES_HOST:-postgres}:5432/postgres?sslmode=${POSTGRES_INTERNAL_SSL_MODE:-disable}&sslrootcert=/tls/ca.crt}'
assert_contains "${EXTERNAL_COMPOSE}" 'OPENFGA_DATASTORE_URI: ${OPENFGA_DATASTORE_URI:-postgresql://openfga:${OPENFGA_DB_PASSWORD}@${POSTGRES_HOST:-postgres}:5432/openfga?sslmode=${POSTGRES_INTERNAL_SSL_MODE:-disable}&sslrootcert=/tls/ca.crt}'

assert_contains "${INIT_SQL}" '\getenv postgres_exporter_password POSTGRES_EXPORTER_DB_PASSWORD'
assert_contains "${INIT_SQL}" 'CREATE ROLE stuhelper_metrics LOGIN PASSWORD'
assert_contains "${INIT_SQL}" 'GRANT pg_monitor TO stuhelper_metrics'
assert_contains "${SHARED_INIT}" 'POSTGRES_EXPORTER_DB_PASSWORD'
assert_contains "${SHARED_INIT}" "granted_role.rolname <> 'pg_monitor'"

for hba in "${DEV_HBA}" "${PROD_HBA}"; do
  assert_contains "${hba}" 'stuhelper,postgres stuhelper_backup'
  assert_contains "${hba}" 'postgres        stuhelper_metrics'
  assert_not_contains "${hba}" 'all             stuhelper_metrics'
done

assert_contains "${ENSURE_ROLE}" 'external PostgreSQL roles must be provisioned by its administrator'
assert_contains "${ENSURE_ROLE}" '-e POSTGRES_EXPORTER_DB_PASSWORD="${POSTGRES_EXPORTER_DB_PASSWORD}"'
assert_contains "${ENSURE_ROLE}" 'ALTER ROLE stuhelper_metrics'
assert_contains "${ENSURE_ROLE}" 'GRANT pg_monitor TO stuhelper_metrics'
assert_contains "${ENSURE_ROLE}" 'SELECT pg_reload_conf();'
assert_contains "${ENSURE_ROLE}" 'FROM pg_hba_file_rules'

assert_contains "${REPO_ROOT}/infra/ops/dev-up.sh" '"${SCRIPT_DIR}/ensure-postgres-monitoring-role.sh"'
assert_contains "${REPO_ROOT}/infra/ops/observability-up.sh" '"${SCRIPT_DIR}/ensure-postgres-monitoring-role.sh"'
assert_contains "${REPO_ROOT}/infra/ops/prod-deploy.sh" '"${SCRIPT_DIR}/ensure-postgres-monitoring-role.sh"'
assert_contains "${OBS_SMOKE}" 'pg_up{job="postgres-exporter"}'
assert_contains "${OBS_SMOKE}" 'redis_up{job="redis-exporter"}'
assert_contains "${OBS_SMOKE}" 'up{job="node-exporter"}'

printf '[postgres-observability-contract] all assertions passed\n'
