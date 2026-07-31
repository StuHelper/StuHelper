#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
# shellcheck source=../lib/common.sh
source "${REPO_ROOT}/infra/ops/lib/common.sh"

fail() {
  echo "[postgres-runtime-url-contract][error] $*" >&2
  exit 1
}

export DATABASE_URL='postgres://stuhelper_app:REPLACE_WITH_STUHELPER_APP_DB_PASSWORD@external-postgres:5432/stuhelper?sslmode=verify-full&sslrootcert=/tls/ca.crt'
export BACKUP_DATABASE_URL='postgres://stuhelper_backup:REPLACE_WITH_STUHELPER_BACKUP_DB_PASSWORD@external-postgres:5432/stuhelper?sslmode=verify-full&sslrootcert=/tls/ca.crt'
export REPLICATION_DATABASE_URL='postgres://stuhelper_replication:REPLACE_WITH_STUHELPER_REPLICATION_DB_PASSWORD@external-postgres:5432/stuhelper?sslmode=verify-full&sslrootcert=/tls/ca.crt'
export STUHELPER_APP_DB_PASSWORD='app@password/path'
export STUHELPER_BACKUP_DB_PASSWORD='backup@password/path'
export STUHELPER_REPLICATION_DB_PASSWORD='replication@password/path'

materialize_postgres_runtime_urls

[[ "${DATABASE_URL}" == *'app%40password%2Fpath'* ]] ||
  fail "application database password was not URL-encoded and materialized"
[[ "${BACKUP_DATABASE_URL}" == *'backup%40password%2Fpath'* ]] ||
  fail "backup database password was not URL-encoded and materialized"
[[ "${REPLICATION_DATABASE_URL}" == *'replication%40password%2Fpath'* ]] ||
  fail "replication database password was not URL-encoded and materialized"
[[ "${DATABASE_URL}${BACKUP_DATABASE_URL}${REPLICATION_DATABASE_URL}" != *'REPLACE_WITH_'* ]] ||
  fail "a PostgreSQL runtime URL still contains a secret placeholder"

require_production_postgres_url DATABASE_URL "${DATABASE_URL}"
require_production_postgres_url BACKUP_DATABASE_URL "${BACKUP_DATABASE_URL}"
require_production_postgres_url REPLICATION_DATABASE_URL "${REPLICATION_DATABASE_URL}"

rejection_log="$(mktemp)"
trap 'rm -f "${rejection_log}"' EXIT
if (
  require_production_postgres_url \
    DATABASE_URL \
    'postgres://stuhelper_app:REPLACE_WITH_STUHELPER_APP_DB_PASSWORD@external-postgres:5432/stuhelper?sslmode=verify-full&sslrootcert=/tls/ca.crt'
) 2>"${rejection_log}"; then
  fail "production PostgreSQL validation accepted an unresolved password placeholder"
fi
grep -q 'unresolved secret placeholder' "${rejection_log}" ||
  fail "unresolved PostgreSQL password rejection was not explicit"

if (
  require_production_postgres_url \
    DATABASE_URL \
    'postgres://stuhelper_app:example@external-postgres:5432/stuhelper?sslmode=disable'
) 2>"${rejection_log}"; then
  fail "production PostgreSQL validation accepted plaintext transport"
fi
grep -q 'must include sslmode=verify-ca or sslmode=verify-full' "${rejection_log}" ||
  fail "plaintext PostgreSQL rejection was not explicit"

preserve_dir="$(mktemp -d)"
trap 'rm -f "${rejection_log}"; rm -rf "${preserve_dir}"' EXIT
cp "${REPO_ROOT}/.env.example" "${preserve_dir}/shared.env"
: >"${preserve_dir}/generated.env"
: >"${preserve_dir}/generated-secrets.env"
export ENV_FILE="${preserve_dir}/shared.env"
export GENERATED_ENV_FILE="${preserve_dir}/generated.env"
export GENERATED_SECRET_ENV_FILE="${preserve_dir}/generated-secrets.env"
export DATABASE_URL='postgres://explicit:explicit-password@explicit-postgres:5432/stuhelper?sslmode=verify-full&sslrootcert=/tls/ca.crt'
load_env_preserving DATABASE_URL
[[ "${DATABASE_URL}" == *'@explicit-postgres:5432/'* ]] ||
  fail "an explicit runtime DATABASE_URL was overwritten by the shared env file"

compose_env=(
  EXTERNAL_POSTGRES_ENABLED=true
  EXTERNAL_DATASTORE_NETWORK=stuhelper-external-contract
  ENV_FILE_PATH=/dev/null
  SECRETS_ENV_FILE_PATH=/dev/null
  GENERATED_ENV_FILE_PATH=/dev/null
  GENERATED_SECRET_ENV_FILE_PATH=/dev/null
  DATABASE_URL=postgres://stuhelper_app:dummy@external-postgres:5432/stuhelper?sslmode=verify-full\&sslrootcert=/tls/ca.crt
  REDIS_PASSWORD=redis-dummy
  REDIS_EXPORTER_PASSWORD=redis-metrics-dummy
  GRAFANA_ADMIN_PASSWORD=grafana-dummy
  POSTGRES_EXPORTER_DB_PASSWORD=postgres-metrics-dummy
  OPENFGA_DB_PASSWORD=openfga-dummy
  STUHELPER_APP_DB_PASSWORD=app-dummy
  STUHELPER_BACKUP_DB_PASSWORD=backup-dummy
  STUHELPER_REPLICATION_DB_PASSWORD=replication-dummy
  OBJECT_STORAGE_ACCESS_KEY_ID=object-storage-dummy
  OBJECT_STORAGE_SECRET_ACCESS_KEY=object-storage-secret-dummy
  OBJECT_STORAGE_BUCKET=object-storage-bucket-dummy
  CASDOOR_CLIENT_SECRET=casdoor-client-dummy
  CASDOOR_APP_PROVISIONING_CLIENT_SECRET=casdoor-provisioning-dummy
  CASDOOR_USER_PROFILE_CLIENT_SECRET=casdoor-profile-dummy
  CASDOOR_INTROSPECTION_CLIENT_SECRET=casdoor-introspection-dummy
  CASDOOR_USER_LOOKUP_CLIENT_SECRET=casdoor-user-lookup-dummy
  BACKEND_IMAGE_REF=example.invalid/stuhelper/backend:test
  FRONTEND_IMAGE_REF=example.invalid/stuhelper/frontend:test
  ADMIN_IMAGE_REF=example.invalid/stuhelper/admin:test
)

env -u POSTGRES_PASSWORD "${compose_env[@]}" \
  docker compose \
    -f "${REPO_ROOT}/docker-compose.yml" \
    -f "${REPO_ROOT}/docker-compose.observability.yml" \
    -f "${REPO_ROOT}/docker-compose.prod.yml" \
    -f "${REPO_ROOT}/docker-compose.external-datastore.yml" \
    --env-file /dev/null \
    --profile prod \
    config --quiet ||
  fail "external PostgreSQL compose rendering still requires POSTGRES_PASSWORD"

echo "[postgres-runtime-url-contract] all assertions passed"
