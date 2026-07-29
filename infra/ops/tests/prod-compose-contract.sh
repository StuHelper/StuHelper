#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
COMPOSE_PROD_FILE="${REPO_ROOT}/docker-compose.prod.yml"
COMPOSE_EXTERNAL_DATASTORE_FILE="${REPO_ROOT}/docker-compose.external-datastore.yml"
COMPOSE_FILE="${REPO_ROOT}/docker-compose.yml"
OBS_COMPOSE_FILE="${REPO_ROOT}/docker-compose.observability.yml"
COMMON_LIB_FILE="${REPO_ROOT}/infra/ops/lib/common.sh"
REDIS_ACL_RENDER_FILE="${REPO_ROOT}/infra/ops/render-redis-acl.sh"
REDIS_TLS_RENDER_FILE="${REPO_ROOT}/infra/ops/render-redis-tls.sh"
OBJECT_STORAGE_TLS_RENDER_FILE="${REPO_ROOT}/infra/ops/render-object-storage-tls.sh"
OBJECT_STORAGE_CLIENT_CA_PREPARE_FILE="${REPO_ROOT}/infra/ops/prepare-object-storage-client-ca.sh"
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
external_app_block="$(
  awk '
    /^  app:/ { in_block=1; next }
    /^networks:/ { in_block=0 }
    in_block { print }
  ' "${COMPOSE_EXTERNAL_DATASTORE_FILE}"
)"
[[ -n "${external_app_block}" ]] || fail "expected app service block in ${COMPOSE_EXTERNAL_DATASTORE_FILE}"

object_storage_block="$(
  awk '
    /^  object-storage:/ { in_block=1; next }
    /^  migrate-dev:/ { in_block=0 }
    in_block { print }
  ' "${COMPOSE_FILE}"
)"

[[ -n "${object_storage_block}" ]] || fail "expected local object-storage service block in ${COMPOSE_FILE}"

migrate_dev_block="$(
  awk '
    /^  migrate-dev:/ { in_block=1; next }
    /^  migrate:/ { in_block=0 }
    in_block { print }
  ' "${COMPOSE_FILE}"
)"
[[ -n "${migrate_dev_block}" ]] || fail "expected migrate-dev service block in ${COMPOSE_FILE}"

openfga_migrate_block="$(
  awk '
    /^  openfga-migrate:/ { in_block=1; next }
    /^  openfga:/ { in_block=0 }
    in_block { print }
  ' "${COMPOSE_FILE}"
)"
[[ -n "${openfga_migrate_block}" ]] || fail "expected openfga-migrate service block in ${COMPOSE_FILE}"

postgres_client_block="$(
  awk '
    /^  postgres-client:/ { in_block=1; next }
    /^  redis:/ { in_block=0 }
    in_block { print }
  ' "${COMPOSE_FILE}"
)"
[[ -n "${postgres_client_block}" ]] || fail "expected postgres-client service block in ${COMPOSE_FILE}"

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
assert_contains "${REPO_ROOT}/docker-compose.yml" 'password=\$\{CASDOOR_DB_PASSWORD:-\}'
assert_not_contains "${REPO_ROOT}/docker-compose.yml" 'password=\$\{CASDOOR_DB_PASSWORD\}'
assert_contains "${REPO_ROOT}/docker-compose.yml" 'POSTGRES_PASSWORD: \$\{POSTGRES_PASSWORD:-\}'
assert_not_contains "${REPO_ROOT}/docker-compose.yml" 'POSTGRES_PASSWORD: \$\{POSTGRES_PASSWORD:\?'
if printf '%s\n' "${postgres_client_block}" | grep -Eq 'postgres_(data|wal_archive)|/tls-source|server\.key|STUHELPER_.*DB_PASSWORD|POSTGRES_PASSWORD'; then
  fail "postgres-client must not inherit database volumes, server keys, or database credentials"
fi
if ! printf '%s\n' "${postgres_client_block}" | grep -Eq '\./infra/generated/postgres-client-ca:/tls:ro'; then
  fail "postgres-client must mount only the public PostgreSQL client CA bundle"
fi
if ! printf '%s\n' "${postgres_client_block}" | grep -Eq '^    user: "70:70"$'; then
  fail "postgres-client must run as the PostgreSQL image non-root user"
fi
if ! printf '%s\n' "${postgres_client_block}" | grep -Eq '^    read_only: true$'; then
  fail "postgres-client must use a read-only root filesystem"
fi
if ! printf '%s\n' "${postgres_client_block}" | grep -Eq '^    - ALL$'; then
  fail "postgres-client must drop all Linux capabilities"
fi
assert_contains "${COMPOSE_EXTERNAL_DATASTORE_FILE}" '^  postgres-client:$'
assert_contains "${REPO_ROOT}/docker-compose.yml" 'VITE_QQ_BOT_ENTRY=\$\{WEB_VITE_QQ_BOT_ENTRY:-\}'
assert_contains "${REPO_ROOT}/docker-compose.yml" 'VITE_QQ_BIND_COMMAND=\$\{WEB_VITE_QQ_BIND_COMMAND:-绑定\}'
assert_contains "${COMPOSE_FILE}" 'archive_mode=\$\{POSTGRES_ARCHIVE_MODE:-off\}'
assert_contains "${COMPOSE_FILE}" 'archive_command=sh -c'
assert_contains "${COMPOSE_FILE}" 'dest=/var/lib/postgresql/wal-archive/%f'
assert_contains "${COMPOSE_FILE}" 'cmp -s %p'
assert_contains "${COMPOSE_FILE}" 'mv "\$\$tmp" "\$\$dest"'
assert_not_contains "${COMPOSE_FILE}" "archive_command=sh -c 'test ! -f /var/lib/postgresql/wal-archive/%f && cp %p /var/lib/postgresql/wal-archive/%f'"
assert_contains "${COMPOSE_PROD_FILE}" 'CASDOOR_CLIENT_SECRET: \$\{CASDOOR_CLIENT_SECRET:\?CASDOOR_CLIENT_SECRET is required\}'
assert_contains "${COMPOSE_PROD_FILE}" 'CASDOOR_APP_PROVISIONING_CLIENT_SECRET: \$\{CASDOOR_APP_PROVISIONING_CLIENT_SECRET:\?CASDOOR_APP_PROVISIONING_CLIENT_SECRET is required\}'
assert_contains "${COMPOSE_PROD_FILE}" 'CASDOOR_USER_PROFILE_CLIENT_SECRET: \$\{CASDOOR_USER_PROFILE_CLIENT_SECRET:\?CASDOOR_USER_PROFILE_CLIENT_SECRET is required\}'
assert_contains "${COMPOSE_PROD_FILE}" 'CASDOOR_INTROSPECTION_CLIENT_SECRET: \$\{CASDOOR_INTROSPECTION_CLIENT_SECRET:\?CASDOOR_INTROSPECTION_CLIENT_SECRET is required\}'
assert_contains "${COMPOSE_PROD_FILE}" 'CASDOOR_ROLE_SYNC_CLIENT_SECRET: \$\{CASDOOR_ROLE_SYNC_CLIENT_SECRET:\?CASDOOR_ROLE_SYNC_CLIENT_SECRET is required\}'
assert_contains "${COMPOSE_PROD_FILE}" 'CASDOOR_USER_LOOKUP_CLIENT_SECRET: \$\{CASDOOR_USER_LOOKUP_CLIENT_SECRET:\?CASDOOR_USER_LOOKUP_CLIENT_SECRET is required\}'
assert_contains "${COMPOSE_PROD_FILE}" 'OBJECT_STORAGE_TLS_CA: \$\{OBJECT_STORAGE_TLS_CA:-\}'
assert_contains "${COMPOSE_PROD_FILE}" 'OBJECT_STORAGE_CLIENT_CA_DIR:-\./infra/generated/object-storage-client-ca\}:/object-storage-tls:ro'
assert_contains "${COMPOSE_PROD_FILE}" '\./infra/generated/external-student-source-client-ca:/external-student-source-tls:ro'
assert_contains "${COMPOSE_FILE}" '\./infra/generated/external-student-source-client-ca:/external-student-source-tls:ro'
assert_not_contains "${COMPOSE_PROD_FILE}" '\./infra/generated/object-storage:/object-storage-tls:ro'
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
assert_contains "${OBS_COMPOSE_FILE}" 'REDIS_ADDR: rediss://redis:6379'
assert_contains "${OBS_COMPOSE_FILE}" 'REDIS_USER: \$\{REDIS_EXPORTER_USERNAME:-stuhelper_metrics\}'
assert_contains "${OBS_COMPOSE_FILE}" 'REDIS_PASSWORD: \$\{REDIS_EXPORTER_PASSWORD:\?REDIS_EXPORTER_PASSWORD is required\}'
assert_contains "${OBS_COMPOSE_FILE}" 'REDIS_EXPORTER_DISABLE_SCRAPE_ENDPOINT: "true"'
assert_not_contains "${OBS_COMPOSE_FILE}" 'REDIS_ADDR: .*REDIS_(PASSWORD|EXPORTER_PASSWORD)'
assert_contains "${REDIS_ACL_RENDER_FILE}" '^chmod 600 "\$\{acl_tmp\}"$'
assert_contains "${REDIS_ACL_RENDER_FILE}" '^mv -f "\$\{acl_tmp\}" "\$\{ACL_FILE\}"$'
assert_contains "${REDIS_ACL_RENDER_FILE}" 'password_sha256'
assert_contains "${REDIS_ACL_RENDER_FILE}" 'REDIS_PASSWORD and REDIS_EXPORTER_PASSWORD must be different'
assert_contains "${REDIS_ACL_RENDER_FILE}" 'user \$\{REDIS_USERNAME\} reset on #\$\{app_password_hash\} .* -@all '
assert_contains "${REDIS_ACL_RENDER_FILE}" 'user \$\{REDIS_EXPORTER_USERNAME\} reset on #\$\{exporter_password_hash\} resetkeys resetchannels -@all '
assert_not_contains "${REDIS_ACL_RENDER_FILE}" '>\$\{REDIS_PASSWORD\}'
assert_not_contains "${REDIS_ACL_RENDER_FILE}" '>\$\{REDIS_EXPORTER_PASSWORD\}'
assert_not_contains "${REDIS_ACL_RENDER_FILE}" '\+@all'
assert_contains "${REDIS_TLS_RENDER_FILE}" '^ensure_redis_tls_permissions\(\) \{$'
assert_contains "${REDIS_TLS_RENDER_FILE}" 'chmod 755 "\$\{REDIS_TLS_DIR\}"'
assert_contains "${REDIS_TLS_RENDER_FILE}" 'chmod 600 "\$\{SERVER_KEY\}"'
assert_contains "${REPO_ROOT}/infra/ops/render-postgres-tls.sh" '^ensure_postgres_tls_permissions\(\) \{$'
assert_contains "${REPO_ROOT}/infra/ops/render-postgres-tls.sh" '\[\[ -f "\$\{SERVER_KEY\}" \]\] && chmod 600 "\$\{SERVER_KEY\}"'
assert_contains "${REPO_ROOT}/infra/ops/render-postgres-tls.sh" 'chmod 755 "\$\{POSTGRES_TLS_DIR\}"'
assert_contains "${REPO_ROOT}/infra/ops/render-postgres-tls.sh" '\[\[ -f "\$\{CA_CERT\}" \]\] && chmod 644 "\$\{CA_CERT\}"'
assert_contains "${REPO_ROOT}/infra/ops/render-postgres-tls.sh" 'ensure_postgres_tls_permissions'
assert_contains "${OBJECT_STORAGE_TLS_RENDER_FILE}" '^ensure_object_storage_tls_permissions\(\) \{$'
assert_contains "${OBJECT_STORAGE_TLS_RENDER_FILE}" 'chmod 755 "\$\{OBJECT_STORAGE_TLS_DIR\}"'
assert_contains "${OBJECT_STORAGE_TLS_RENDER_FILE}" 'chmod 644 "\$\{CA_CERT\}"'
assert_contains "${OBJECT_STORAGE_CLIENT_CA_PREPARE_FILE}" '^set -euo pipefail$'
assert_contains "${OBJECT_STORAGE_CLIENT_CA_PREPARE_FILE}" 'OBJECT_STORAGE_TLS_CA_HOST_PATH'
assert_contains "${OBJECT_STORAGE_CLIENT_CA_PREPARE_FILE}" 'object-storage-client-ca'
if grep -Eq 'sslmode=\$\{(DB_SSL_MODE|POSTGRES_INTERNAL_SSL_MODE):-disable\}' "${COMPOSE_PROD_FILE}"; then
  fail "production compose overlay must not default PostgreSQL clients to sslmode=disable"
fi
if ! printf '%s\n' "${object_storage_block}" | grep -Eq '^    read_only: true$'; then
  fail "local object-storage service must run with a read-only root filesystem"
fi
if ! printf '%s\n' "${object_storage_block}" | grep -Eq '^    - /tmp$'; then
  fail "local object-storage service must provide /tmp as tmpfs when root is read-only"
fi
if ! printf '%s\n' "${object_storage_block}" | grep -Eq '^    user: "1000:1000"$'; then
  fail "local object-storage service must run as the upstream non-root user"
fi
if ! printf '%s\n' "${object_storage_block}" | grep -Eq '^    - ALL$'; then
  fail "local object-storage service must drop all Linux capabilities"
fi
if ! printf '%s\n' "${object_storage_block}" | grep -Eq '^    profiles: \[dev-full, prod-parity\]$'; then
  fail "local object-storage service must never join the production profile"
fi
if ! printf '%s\n' "${migrate_dev_block}" | grep -Eq '^      GOTMPDIR: /go-cache$'; then
  fail "migrate-dev must place go run executables on the executable Go cache volume"
fi
if ! printf '%s\n' "${migrate_dev_block}" | grep -Eq '^    - /tmp:rw,nosuid,nodev,noexec,size=64m$'; then
  fail "migrate-dev must keep its disposable /tmp mount non-executable"
fi
if printf '%s\n' "${openfga_migrate_block}" | grep -Eq 'sslrootcert='; then
  fail "local OpenFGA must not require a PostgreSQL CA file when sslmode defaults to disable"
fi
if ! printf '%s\n' "${openfga_migrate_block}" | grep -Eq 'sslmode=\$\{POSTGRES_INTERNAL_SSL_MODE:-disable\}$'; then
  fail "local OpenFGA must use the explicit development PostgreSQL SSL mode"
fi
assert_contains "${COMPOSE_PROD_FILE}" '127\.0\.0\.1:\$\{BACKEND_EXTERNAL_PORT:-18080\}:8080'
assert_contains "${COMPOSE_PROD_FILE}" '127\.0\.0\.1:\$\{WEB_EXTERNAL_PORT:-18000\}:80'
assert_contains "${COMPOSE_PROD_FILE}" '127\.0\.0\.1:\$\{ADMIN_EXTERNAL_PORT:-18001\}:8080'
assert_contains "${COMPOSE_FILE}" '127\.0\.0\.1:\$\{POSTGRES_EXTERNAL_PORT:-5432\}:5432'
assert_contains "${COMPOSE_FILE}" '127\.0\.0\.1:\$\{REDIS_EXTERNAL_PORT:-6379\}:6379'
assert_contains "${COMPOSE_FILE}" '127\.0\.0\.1:\$\{OPENFGA_HTTP_EXTERNAL_PORT:-8081\}:8080'
assert_contains "${COMPOSE_FILE}" '127\.0\.0\.1:\$\{OPENFGA_GRPC_EXTERNAL_PORT:-8082\}:8081'
assert_contains "${COMPOSE_FILE}" '127\.0\.0\.1:\$\{OPENFGA_PLAYGROUND_EXTERNAL_PORT:-3002\}:3000'
assert_contains "${COMPOSE_FILE}" '127\.0\.0\.1:\$\{DEV_OBJECT_STORAGE_EXTERNAL_PORT:-9000\}:8333'
assert_contains "${COMPOSE_FILE}" '127\.0\.0\.1:\$\{LOCAL_OBJECT_STORAGE_TLS_EXTERNAL_PORT:-9001\}:8334'
assert_not_contains "${COMPOSE_FILE}" '^  minio(-init)?:'
assert_not_contains "${COMPOSE_PROD_FILE}" '^  (minio|minio-init|object-storage):'
assert_not_contains "${COMPOSE_FILE}" '^  proxy:'
assert_not_contains "${COMPOSE_FILE}" 'traefik'
assert_not_contains "${COMPOSE_FILE}" 'acme_data'
assert_not_contains "${COMPOSE_FILE}" '^      proxy:'
assert_contains "${BAOTA_NGINX_FILE}" 'server_name stuhelper\.com www\.stuhelper\.com;'
assert_contains "${BAOTA_NGINX_FILE}" 'server_name stuhelper\.com;'
assert_contains "${BAOTA_NGINX_FILE}" 'proxy_pass http://127\.0\.0\.1:18080;'
assert_contains "${BAOTA_NGINX_FILE}" 'proxy_pass http://127\.0\.0\.1:18000;'
assert_contains "${BAOTA_NGINX_FILE}" 'proxy_pass http://127\.0\.0\.1:18001;'
assert_contains "${BAOTA_NGINX_FILE}" 'location = /_app\.config\.js \{'
assert_contains "${BAOTA_NGINX_FILE}" 'location \^~ /css/ \{'
assert_contains "${BAOTA_NGINX_FILE}" 'location \^~ /js/ \{'
assert_contains "${BAOTA_NGINX_FILE}" 'location \^~ /jse/ \{'
assert_contains "${BAOTA_NGINX_FILE}" 'server_name join\.stuhelper\.com;'
assert_contains "${BAOTA_NGINX_FILE}" 'location = /verify \{'
assert_contains "${BAOTA_NGINX_FILE}" 'location \^~ /verify/ \{'
assert_contains "${BAOTA_NGINX_FILE}" 'location \^~ /admission/freshman/camera/ \{'
assert_contains "${BAOTA_NGINX_FILE}" 'location \^~ /api/v1/admission/freshman/camera-handoffs/ \{'
assert_contains "${BAOTA_NGINX_FILE}" 'location = /api/v1/bot/admission/actions/stream \{'
assert_contains "${BAOTA_NGINX_FILE}" 'X-Accel-Buffering no always'
assert_contains "${BAOTA_NGINX_FILE}" 'return 404;'
python3 - "${BAOTA_NGINX_FILE}" <<'PY' || fail "join.stuhelper.com root must return 404 instead of proxying to the main Web SPA"
from pathlib import Path
import sys

text = Path(sys.argv[1]).read_text(encoding="utf-8")
lines = text.splitlines()

def server_blocks():
    index = 0
    while index < len(lines):
        if lines[index].strip() != "server {":
            index += 1
            continue
        start = index
        depth = 0
        while index < len(lines):
            depth += lines[index].count("{") - lines[index].count("}")
            if depth == 0:
                yield lines[start:index + 1]
                break
            index += 1
        index += 1

join_blocks = [block for block in server_blocks() if any(line.strip() == "server_name join.stuhelper.com;" for line in block)]
if not join_blocks:
    raise SystemExit("missing join server")
validated_root = False
for block in join_blocks:
    for offset, line in enumerate(block):
        if line.strip() != "location / {":
            continue
        depth = 0
        collected = []
        for nested in block[offset:]:
            collected.append(nested)
            depth += nested.count("{") - nested.count("}")
            if depth == 0:
                rendered = "\n".join(collected)
                if "return 404;" not in rendered or "proxy_pass" in rendered:
                    raise SystemExit(rendered)
                validated_root = True
                break
        break
if not validated_root:
    raise SystemExit("missing join root location")
PY
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
if ! printf '%s\n' "${external_app_block}" | grep -Eq '^    - frontend$'; then
  fail "external datastore app override must preserve frontend network"
fi
assert_contains "${COMPOSE_PROD_FILE}" '\./infra/postgres/pg_hba\.prod\.conf:/etc/postgresql/pg_hba\.conf:ro'
assert_contains "${PG_HBA_PROD_FILE}" '^hostnossl all[[:space:]]+all[[:space:]]+10\.0\.0\.0/8[[:space:]]+reject$'
assert_contains "${PG_HBA_PROD_FILE}" '^hostssl stuhelper[[:space:]]+stuhelper_app[[:space:]]+10\.0\.0\.0/8[[:space:]]+scram-sha-256$'

echo "[prod-compose-contract] all assertions passed"
