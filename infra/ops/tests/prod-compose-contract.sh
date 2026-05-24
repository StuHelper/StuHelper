#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
COMPOSE_PROD_FILE="${REPO_ROOT}/docker-compose.prod.yml"
COMPOSE_EXTERNAL_DATASTORE_FILE="${REPO_ROOT}/docker-compose.external-datastore.yml"
COMPOSE_FILE="${REPO_ROOT}/docker-compose.yml"
COMMON_LIB_FILE="${REPO_ROOT}/infra/ops/lib/common.sh"
REDIS_ACL_RENDER_FILE="${REPO_ROOT}/infra/ops/render-redis-acl.sh"
REDIS_TLS_RENDER_FILE="${REPO_ROOT}/infra/ops/render-redis-tls.sh"
PG_HBA_PROD_FILE="${REPO_ROOT}/infra/postgres/pg_hba.prod.conf"
BAOTA_NGINX_FILE="${REPO_ROOT}/infra/nginx/baota-stuhelper.conf"
SSO_NGINX_FILE="${REPO_ROOT}/infra/nginx/baota-casdoor-sso.conf"

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

assert_not_contains() {
  local file="$1"
  local pattern="$2"
  if grep -Eq "${pattern}" "${file}"; then
    fail "expected ${file} to not contain pattern: ${pattern}"
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
[[ -f "${COMPOSE_EXTERNAL_DATASTORE_FILE}" ]] || fail "missing external datastore compose overlay"

minio_block="$(
  awk '
    /^  minio:/ { in_block=1; next }
    /^  minio-init:/ { in_block=0 }
    in_block { print }
  ' "${COMPOSE_FILE}"
)"

minio_init_block="$(
  awk '
    /^  minio-init:/ { in_block=1; next }
    /^  migrate-dev:/ { in_block=0 }
    in_block { print }
  ' "${COMPOSE_FILE}"
)"

[[ -n "${minio_block}" ]] || fail "expected minio service block in ${COMPOSE_FILE}"
[[ -n "${minio_init_block}" ]] || fail "expected minio-init service block in ${COMPOSE_FILE}"

if ! printf '%s\n' "${app_block}" | grep -Eq '\$\{SECRETS_ENV_FILE_PATH:-\.env\.prod\.secrets\.local\}'; then
  fail "app env_file must inject the production secrets env file"
fi

if ! printf '%s\n' "${app_block}" | grep -Eq '\$\{GENERATED_SECRET_ENV_FILE_PATH:-\.env\.prod\.generated\.secrets\}'; then
  fail "app env_file must inject the generated secret env file placeholder"
fi
if printf '%s\n' "${app_block}" | grep -Eq 'CASDOOR_BOOTSTRAP_CLIENT_SECRET'; then
  fail "app service must not receive one-shot Casdoor bootstrap credentials"
fi

assert_contains "${COMMON_LIB_FILE}" 'SECRETS_ENV_FILE_PATH="\$\{SECRETS_ENV_FILE\}"'
assert_contains "${COMMON_LIB_FILE}" 'GENERATED_SECRET_ENV_FILE_PATH="\$\{GENERATED_SECRET_ENV_FILE\}"'
assert_contains "${REPO_ROOT}/docker-compose.yml" 'CASDOOR_DB_PASSWORD: \$\{CASDOOR_DB_PASSWORD:-\}'
assert_contains "${COMPOSE_PROD_FILE}" 'CASDOOR_CLIENT_SECRET: \$\{CASDOOR_CLIENT_SECRET:\?CASDOOR_CLIENT_SECRET is required\}'
assert_contains "${COMPOSE_PROD_FILE}" 'CASDOOR_APP_PROVISIONING_CLIENT_SECRET: \$\{CASDOOR_APP_PROVISIONING_CLIENT_SECRET:\?CASDOOR_APP_PROVISIONING_CLIENT_SECRET is required\}'
assert_contains "${COMPOSE_PROD_FILE}" 'CASDOOR_USER_PROFILE_CLIENT_SECRET: \$\{CASDOOR_USER_PROFILE_CLIENT_SECRET:\?CASDOOR_USER_PROFILE_CLIENT_SECRET is required\}'
assert_contains "${COMPOSE_PROD_FILE}" 'CASDOOR_INTROSPECTION_CLIENT_SECRET: \$\{CASDOOR_INTROSPECTION_CLIENT_SECRET:\?CASDOOR_INTROSPECTION_CLIENT_SECRET is required\}'
assert_contains "${COMPOSE_PROD_FILE}" 'CASDOOR_ROLE_SYNC_CLIENT_SECRET: \$\{CASDOOR_ROLE_SYNC_CLIENT_SECRET:\?CASDOOR_ROLE_SYNC_CLIENT_SECRET is required\}'
assert_contains "${COMPOSE_PROD_FILE}" 'CASDOOR_USER_LOOKUP_CLIENT_SECRET: \$\{CASDOOR_USER_LOOKUP_CLIENT_SECRET:\?CASDOOR_USER_LOOKUP_CLIENT_SECRET is required\}'
assert_contains "${COMPOSE_PROD_FILE}" 'sslmode=\$\{DB_SSL_MODE:-verify-full\}&sslrootcert=/tls/ca\.crt'
assert_contains "${COMPOSE_PROD_FILE}" 'sslmode=\$\{POSTGRES_INTERNAL_SSL_MODE:-verify-full\}&sslrootcert=/tls/ca\.crt'
assert_contains "${COMPOSE_EXTERNAL_DATASTORE_FILE}" 'profiles: !override \[internal-datastore\]'
assert_contains "${COMPOSE_EXTERNAL_DATASTORE_FILE}" 'profiles: !override \[local-sso\]'
assert_contains "${COMPOSE_EXTERNAL_DATASTORE_FILE}" 'casdoor:'
assert_contains "${COMPOSE_EXTERNAL_DATASTORE_FILE}" 'external-datastore'
assert_contains "${COMPOSE_EXTERNAL_DATASTORE_FILE}" 'name: \$\{EXTERNAL_DATASTORE_NETWORK:\?EXTERNAL_DATASTORE_NETWORK is required\}'
assert_contains "${COMPOSE_EXTERNAL_DATASTORE_FILE}" 'external: true'
assert_contains "${COMPOSE_EXTERNAL_DATASTORE_FILE}" 'depends_on: !reset \[\]'
assert_contains "${COMPOSE_EXTERNAL_DATASTORE_FILE}" 'depends_on: !override'
assert_contains "${COMPOSE_EXTERNAL_DATASTORE_FILE}" 'DATABASE_URL: \$\{DATABASE_URL:\?DATABASE_URL is required\}'
assert_contains "${COMPOSE_EXTERNAL_DATASTORE_FILE}" '^      redis:$'
assert_not_contains "${COMPOSE_EXTERNAL_DATASTORE_FILE}" '^  redis:'
assert_not_contains "${COMPOSE_EXTERNAL_DATASTORE_FILE}" '^  redis-exporter:'
assert_not_contains "${COMPOSE_EXTERNAL_DATASTORE_FILE}" 'REDIS_TLS_ENABLED: \$\{REDIS_TLS_ENABLED:-false\}'
assert_not_contains "${COMPOSE_EXTERNAL_DATASTORE_FILE}" 'REDIS_ADDR: \$\{REDIS_EXPORTER_ADDR:-redis://'
assert_contains "${REDIS_ACL_RENDER_FILE}" '^chmod 644 "\$\{ACL_FILE\}"$'
assert_contains "${REDIS_TLS_RENDER_FILE}" '^ensure_redis_tls_permissions\(\) \{$'
assert_contains "${REDIS_TLS_RENDER_FILE}" 'chmod 644 "\$\{SERVER_KEY\}"'
if grep -Eq 'sslmode=\$\{(DB_SSL_MODE|POSTGRES_INTERNAL_SSL_MODE):-disable\}' "${COMPOSE_PROD_FILE}"; then
  fail "production compose overlay must not default PostgreSQL clients to sslmode=disable"
fi
if ! printf '%s\n' "${minio_block}" | grep -Eq '^    read_only: true$'; then
  fail "minio service must run with a read-only root filesystem"
fi
if ! printf '%s\n' "${minio_block}" | grep -Eq '^    - /tmp$'; then
  fail "minio service must provide /tmp as tmpfs when root is read-only"
fi
if ! printf '%s\n' "${minio_init_block}" | grep -Eq '^      MC_CONFIG_DIR: /tmp/\.mc$'; then
  fail "minio-init must keep mc config under writable tmpfs"
fi
if ! printf '%s\n' "${minio_init_block}" | grep -Eq '^    read_only: true$'; then
  fail "minio-init service must run with a read-only root filesystem"
fi
if ! printf '%s\n' "${minio_init_block}" | grep -Eq '^    - /tmp$'; then
  fail "minio-init service must provide /tmp as tmpfs when root is read-only"
fi
assert_contains "${COMPOSE_PROD_FILE}" '127\.0\.0\.1:\$\{BACKEND_EXTERNAL_PORT:-18080\}:8080'
assert_contains "${COMPOSE_PROD_FILE}" '127\.0\.0\.1:\$\{WEB_EXTERNAL_PORT:-18000\}:80'
assert_contains "${COMPOSE_PROD_FILE}" '127\.0\.0\.1:\$\{ADMIN_EXTERNAL_PORT:-18001\}:8080'
assert_contains "${COMPOSE_FILE}" '127\.0\.0\.1:\$\{MINIO_API_EXTERNAL_PORT:-9000\}:9000'
assert_contains "${COMPOSE_FILE}" '127\.0\.0\.1:\$\{MINIO_CONSOLE_EXTERNAL_PORT:-9001\}:9001'
assert_contains "${BAOTA_NGINX_FILE}" 'server_name stuhelper\.com www\.stuhelper\.com;'
assert_contains "${BAOTA_NGINX_FILE}" 'server_name stuhelper\.com;'
assert_contains "${BAOTA_NGINX_FILE}" 'proxy_pass http://127\.0\.0\.1:18080;'
assert_contains "${BAOTA_NGINX_FILE}" 'proxy_pass http://127\.0\.0\.1:18000;'
assert_contains "${BAOTA_NGINX_FILE}" 'proxy_pass http://127\.0\.0\.1:18001;'
assert_contains "${BAOTA_NGINX_FILE}" 'location \^~ /health/ \{'
assert_contains "${BAOTA_NGINX_FILE}" 'location = /metrics \{'
assert_contains "${BAOTA_NGINX_FILE}" 'location \^~ /docs/ \{'
assert_contains "${SSO_NGINX_FILE}" 'server_name sso\.stuhelper\.com;'
assert_contains "${SSO_NGINX_FILE}" 'location = /\.well-known/openid-configuration \{'
assert_contains "${SSO_NGINX_FILE}" 'location = /\.well-known/jwks \{'
assert_contains "${SSO_NGINX_FILE}" 'location \^~ /\.well-known/ \{'
assert_contains "${SSO_NGINX_FILE}" 'proxy_pass http://127\.0\.0\.1:8087;'
if printf '%s\n' "${app_block}" | grep -Eq 'proxy:'; then
  fail "production app service must not depend on Traefik when Baota/Nginx owns public ingress"
fi
if ! printf '%s\n' "${app_block}" | grep -Eq '^    - frontend$'; then
  fail "production app service must join frontend network so web/admin nginx can resolve app upstreams"
fi
assert_contains "${COMPOSE_PROD_FILE}" '\./infra/postgres/pg_hba\.prod\.conf:/etc/postgresql/pg_hba\.conf:ro'
assert_contains "${PG_HBA_PROD_FILE}" '^hostnossl all[[:space:]]+all[[:space:]]+10\.0\.0\.0/8[[:space:]]+reject$'
assert_contains "${PG_HBA_PROD_FILE}" '^hostssl stuhelper[[:space:]]+stuhelper_app[[:space:]]+10\.0\.0\.0/8[[:space:]]+scram-sha-256$'

echo "[prod-compose-contract] all assertions passed"
