#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
PARITY_COMPOSE="${REPO_ROOT}/docker-compose.prod-parity-postgres.yml"
INIT_SHARED_PG="${REPO_ROOT}/infra/ops/init-shared-postgres.sh"
PARITY_UP="${REPO_ROOT}/infra/ops/prod-parity-up.sh"
PARITY_DOWN="${REPO_ROOT}/infra/ops/prod-parity-down.sh"
PARITY_SMOKE="${REPO_ROOT}/infra/ops/prod-parity-smoke.sh"
SMOKE_CHECK="${REPO_ROOT}/infra/ops/smoke-check.sh"
SSO_PUBLIC_SMOKE="${REPO_ROOT}/infra/ops/sso-public-smoke.sh"
ADMISSION_PUBLIC_SMOKE="${REPO_ROOT}/infra/ops/admission-public-smoke.sh"
PARITY_SMOKE_DATA="${REPO_ROOT}/infra/ops/prod-parity-smoke-data.sh"
OBJECT_STORAGE_TLS_RENDER="${REPO_ROOT}/infra/ops/render-object-storage-tls.sh"
OBJECT_STORAGE_CONFIG_RENDER="${REPO_ROOT}/infra/ops/render-local-object-storage-config.sh"
ADMISSION_READINESS="${REPO_ROOT}/infra/ops/admission-production-readiness.sh"
PARITY_DATASTORE_SMOKE="${REPO_ROOT}/infra/ops/prod-parity-datastore-smoke.sh"
PARITY_BROWSER_SMOKE="${REPO_ROOT}/infra/ops/prod-parity-browser-smoke.sh"
PARITY_BROWSER_SMOKE_NODE="${REPO_ROOT}/infra/ops/prod-parity-browser-smoke.mjs"
ADMISSION_PROD_SIM_E2E="${REPO_ROOT}/infra/ops/admission-prod-sim-e2e.sh"
ADMISSION_PROD_SIM_E2E_NODE="${REPO_ROOT}/infra/ops/admission-prod-sim-e2e.mjs"
PARITY_LOCAL_INGRESS="${REPO_ROOT}/infra/ops/install-local-prod-parity-ingress.sh"
PARITY_LOCAL_INGRESS_NGINX="${REPO_ROOT}/infra/nginx/prod-parity-local-ingress.conf"
COMMON_LIB="${REPO_ROOT}/infra/ops/lib/common.sh"
ADMIN_INDEX_HTML="${REPO_ROOT}/clients/admin/apps/web-ele/index.html"
WEB_NGINX="${REPO_ROOT}/clients/web/nginx.conf"
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

for file in "${PARITY_COMPOSE}" "${INIT_SHARED_PG}" "${PARITY_UP}" "${PARITY_DOWN}" "${PARITY_SMOKE}" "${SMOKE_CHECK}" "${SSO_PUBLIC_SMOKE}" "${ADMISSION_PUBLIC_SMOKE}" "${PARITY_SMOKE_DATA}" "${OBJECT_STORAGE_TLS_RENDER}" "${OBJECT_STORAGE_CONFIG_RENDER}" "${ADMISSION_READINESS}" "${PARITY_DATASTORE_SMOKE}" "${PARITY_BROWSER_SMOKE}" "${PARITY_BROWSER_SMOKE_NODE}" "${ADMISSION_PROD_SIM_E2E}" "${ADMISSION_PROD_SIM_E2E_NODE}" "${PARITY_LOCAL_INGRESS}" "${PARITY_LOCAL_INGRESS_NGINX}" "${COMMON_LIB}" "${ADMIN_INDEX_HTML}" "${WEB_NGINX}"; do
  [[ -f "${file}" ]] || fail "missing file: ${file}"
done

bash -n "${INIT_SHARED_PG}" "${PARITY_UP}" "${PARITY_DOWN}" "${PARITY_SMOKE}" "${SMOKE_CHECK}" "${SSO_PUBLIC_SMOKE}" "${ADMISSION_PUBLIC_SMOKE}" "${PARITY_SMOKE_DATA}" "${OBJECT_STORAGE_TLS_RENDER}" "${OBJECT_STORAGE_CONFIG_RENDER}" "${ADMISSION_READINESS}" "${PARITY_DATASTORE_SMOKE}" "${PARITY_BROWSER_SMOKE}" "${ADMISSION_PROD_SIM_E2E}" "${PARITY_LOCAL_INGRESS}"

for default_path in ".env" "./.env" ".env.generated" "./.env.generated" ".env.generated.secrets" "./.env.generated.secrets" ".deploy" "./.deploy"; do
  case "${default_path}" in
    *.generated.secrets|./*.generated.secrets) expected="${REPO_ROOT}/.env.generated.secrets" ;;
    *.generated|./*.generated) expected="${REPO_ROOT}/.env.generated" ;;
    *.env|./*.env) expected="${REPO_ROOT}/.env" ;;
    *) expected="${REPO_ROOT}/.deploy" ;;
  esac
  if ! bash -c 'source "$1"; repo_default_path_matches "$2" "$3"' bash "${COMMON_LIB}" "${default_path}" "${expected}"; then
    fail "expected repo_default_path_matches to accept ${default_path} as ${expected}"
  fi
done

if bash -c 'source "$1"; repo_default_path_matches "$2" "$3"' bash "${COMMON_LIB}" "/tmp/custom.env" "${REPO_ROOT}/.env"; then
  fail "expected repo_default_path_matches to preserve explicit custom env paths"
fi

assert_contains "${PARITY_COMPOSE}" '^  postgres:'
assert_contains "${PARITY_COMPOSE}" 'POSTGRES_PASSWORD: \$\{SHARED_POSTGRES_PASSWORD:\?SHARED_POSTGRES_PASSWORD is required\}'
assert_contains "${PARITY_COMPOSE}" 'PROD_PARITY_POSTGRES_PORT:-15432'
assert_contains "${PARITY_COMPOSE}" 'prod_parity_postgres_data:/var/lib/postgresql'
assert_contains "${PARITY_COMPOSE}" 'name: \$\{EXTERNAL_DATASTORE_NETWORK:-stuhelper-prod-parity-baota-net\}'
assert_contains "${PARITY_COMPOSE}" 'aliases:'
assert_contains "${PARITY_COMPOSE}" 'postgres'
assert_not_contains "${PARITY_COMPOSE}" '^  redis:'
assert_not_contains "${PARITY_COMPOSE}" '^[[:space:]]+entrypoint:'
assert_contains "${REPO_ROOT}/docker-compose.prod.yml" 'APP_ENV: \$\{APP_ENV:-production\}'

assert_contains "${INIT_SHARED_PG}" 'STUHELPER_APP_DB_PASSWORD'
assert_contains "${INIT_SHARED_PG}" 'POSTGRES_EXPORTER_DB_PASSWORD'
assert_contains "${INIT_SHARED_PG}" 'OPENFGA_DB_PASSWORD'
assert_contains "${INIT_SHARED_PG}" 'CASDOOR_DB_PASSWORD'
assert_contains "${INIT_SHARED_PG}" 'CREATE DATABASE %I OWNER %I'
assert_contains "${INIT_SHARED_PG}" 'ALTER ROLE %I WITH LOGIN PASSWORD %L'
assert_contains "${INIT_SHARED_PG}" 'GRANT pg_read_all_data, pg_read_all_settings, pg_read_all_stats'
assert_contains "${INIT_SHARED_PG}" 'GRANT pg_monitor TO %I'
assert_contains "${INIT_SHARED_PG}" 'REVOKE CONNECT ON DATABASE %I FROM %I'
assert_contains "${INIT_SHARED_PG}" 'GRANT USAGE, CREATE ON SCHEMA public'
assert_contains "${INIT_SHARED_PG}" 'openfga_database'
assert_contains "${INIT_SHARED_PG}" 'casdoor_database'

assert_contains "${PARITY_UP}" 'EXTERNAL_POSTGRES_ENABLED.*true'
assert_contains "${PARITY_UP}" 'EXTERNAL_POSTGRES_ALLOW_PLAINTEXT.*true'
assert_contains "${PARITY_UP}" 'EXTERNAL_DATASTORE_NETWORK.*stuhelper-prod-parity-baota-net'
assert_contains "${PARITY_UP}" 'APP_ENV.*prod-parity'
assert_contains "${PARITY_UP}" 'REDIS_EXTERNAL_PORT.*26379'
assert_contains "${PARITY_UP}" 'OPENFGA_HTTP_EXTERNAL_PORT.*8081'
assert_contains "${PARITY_UP}" 'OPENFGA_GRPC_EXTERNAL_PORT.*8082'
assert_contains "${PARITY_UP}" 'OPENFGA_PLAYGROUND_EXTERNAL_PORT.*3002'
assert_contains "${PARITY_UP}" 'DEV_OBJECT_STORAGE_EXTERNAL_PORT.*29000'
assert_contains "${PARITY_UP}" 'LOCAL_OBJECT_STORAGE_TLS_EXTERNAL_PORT.*29001'
assert_contains "${PARITY_UP}" 'render-local-object-storage-config\.sh'
assert_contains "${PARITY_UP}" 'render-object-storage-tls\.sh'
assert_contains "${PARITY_UP}" 'prepare-object-storage-client-ca\.sh'
assert_contains "${PARITY_UP}" 'OBJECT_STORAGE_TLS_CA_HOST_PATH.*infra/generated/object-storage/ca\.crt'
assert_contains "${PARITY_UP}" 'OBJECT_STORAGE_ENDPOINT.*https://object-storage:8334'
assert_contains "${PARITY_UP}" 'BACKUP_OBJECT_STORAGE_TLS_INSECURE.*false'
assert_contains "${PARITY_UP}" 'BACKUP_OBJECT_STORAGE_DOCKER_NETWORK.*stuhelper-prod-parity-backend'
assert_contains "${PARITY_UP}" 'WEB_PUBLIC_URL.*https://stuhelper\.com'
assert_contains "${PARITY_UP}" 'ADMIN_PUBLIC_URL.*https://stuhelper\.com/admin/'
assert_contains "${PARITY_UP}" 'ADMISSION_PUBLIC_BASE_URL.*https://join\.stuhelper\.com'
assert_contains "${PARITY_UP}" 'ADMISSION_PRODUCTION_READINESS_ENABLED.*true'
assert_contains "${PARITY_UP}" 'ADMISSION_READINESS_REQUIRED_PLATFORM.*qq'
assert_contains "${PARITY_UP}" 'ADMISSION_READINESS_REQUIRED_GUILD_IDS.*prod-parity-guild'
assert_contains "${PARITY_UP}" 'ADMISSION_READINESS_REQUIRED_SCHOOL_CODES.*4111010006'
assert_contains "${PARITY_UP}" 'CORS_ORIGINS.*https://stuhelper\.com,https://join\.stuhelper\.com,https://sso\.stuhelper\.com'
assert_contains "${PARITY_UP}" 'FRONTEND_METRICS_ALLOWED_ORIGINS.*https://stuhelper\.com'
assert_contains "${PARITY_UP}" 'OPEN_PLATFORM_CONSENT_BASE_URL.*https://stuhelper\.com'
assert_contains "${PARITY_UP}" 'OPEN_PLATFORM_ACCOUNT_BASE_URL.*https://stuhelper\.com'
assert_contains "${PARITY_UP}" 'STUHELPER_FRESHMAN_MATERIAL_HOSTS.*stuhelper\.com,join\.stuhelper\.com'
assert_contains "${PARITY_UP}" 'WEB_VITE_SSO_URL.*https://sso\.stuhelper\.com'
assert_contains "${PARITY_UP}" 'WEB_VITE_WEB_URL.*https://stuhelper\.com'
assert_contains "${PARITY_UP}" 'WEB_VITE_QQ_BOT_ENTRY.*""'
assert_contains "${PARITY_UP}" 'WEB_VITE_QQ_BIND_COMMAND.*绑定'
assert_contains "${PARITY_UP}" 'VITE_QQ_BOT_ENTRY="\$\{WEB_VITE_QQ_BOT_ENTRY\}"'
assert_contains "${PARITY_UP}" 'VITE_QQ_BIND_COMMAND="\$\{WEB_VITE_QQ_BIND_COMMAND\}"'
assert_contains "${PARITY_UP}" 'PUBLIC_INGRESS_CASDOOR_UPSTREAM_PREFLIGHT_ENABLED.*true'
assert_contains "${PARITY_UP}" 'SSO_PUBLIC_SMOKE_ENABLED.*true'
assert_contains "${PARITY_UP}" 'SSO_PUBLIC_BASE_URL.*https://sso\.stuhelper\.com'
assert_contains "${PARITY_UP}" 'SSO_PUBLIC_SMOKE_EXPECTED_ISSUER.*http://sso\.stuhelper\.com'
assert_contains "${PARITY_UP}" 'SSO_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS.*true'

assert_contains "${OBJECT_STORAGE_TLS_RENDER}" 'OBJECT_STORAGE_TLS_DIR'
assert_contains "${OBJECT_STORAGE_TLS_RENDER}" 'ca\.crt'
assert_contains "${OBJECT_STORAGE_TLS_RENDER}" 'is a directory'
assert_contains "${OBJECT_STORAGE_TLS_RENDER}" 'sudo -n true'
assert_contains "${OBJECT_STORAGE_TLS_RENDER}" 'openssl_cmd req'
assert_contains "${OBJECT_STORAGE_CONFIG_RENDER}" 'stuhelper-application'
assert_contains "${OBJECT_STORAGE_CONFIG_RENDER}" 'stuhelper-backup'
assert_contains "${OBJECT_STORAGE_CONFIG_RENDER}" 'os\.replace'
assert_contains "${PARITY_UP}" 'CASDOOR_EXTERNALPORT.*28085'
assert_contains "${PARITY_UP}" 'CASDOOR_ISSUER.*http://sso\.stuhelper\.com'
assert_contains "${PARITY_UP}" 'CASDOOR_PUBLIC_AUTH_BASE_URL.*https://sso\.stuhelper\.com'
assert_contains "${PARITY_UP}" 'CASDOOR_INTERNAL_ADDRESS.*host\.docker\.internal:80'
assert_contains "${PARITY_UP}" 'CASDOOR_REDIRECT_URI.*https://stuhelper\.com/api/v1/auth/callback'
assert_contains "${PARITY_UP}" 'TOKEN_COOKIE_SECURE.*true'
assert_contains "${PARITY_UP}" 'TOKEN_COOKIE_DOMAIN.*\.stuhelper\.com'
assert_contains "${PARITY_UP}" 'CASDOOR_BOOTSTRAP_ENABLED.*true'
assert_contains "${PARITY_UP}" 'CASDOOR_BOOTSTRAP_ENV_FILE'
assert_contains "${PARITY_UP}" 'CASDOOR_DB_PASSWORD'
assert_contains "${PARITY_UP}" 'POSTGRES_EXPORTER_DB_PASSWORD'
assert_contains "${PARITY_UP}" 'REDIS_EXPORTER_PASSWORD'
assert_contains "${PARITY_UP}" 'REDIS_EXPORTER_USERNAME'
assert_contains "${PARITY_UP}" 'CASDOOR_BOOTSTRAP_CLIENT_ID'
assert_contains "${PARITY_UP}" 'CASDOOR_BOOTSTRAP_CLIENT_SECRET'
assert_contains "${PARITY_UP}" 'CASDOOR_BOOTSTRAP_ORGANIZATION'
assert_contains "${PARITY_UP}" 'sync_casdoor_builtin_bootstrap_credentials'
assert_contains "${PARITY_UP}" 'app-built-in'
assert_contains "${PARITY_UP}" 'SELECT client_id, client_secret FROM application'
assert_contains "${PARITY_UP}" 'compose --profile prod --profile local-sso up -d --wait casdoor'
assert_contains "${PARITY_UP}" 'parity_default_path'
assert_contains "${PARITY_UP}" '\.run/prod-parity'
assert_contains "${PARITY_UP}" 'init-shared-postgres.sh'
assert_contains "${PARITY_UP}" 'render-redis-tls.sh'
assert_contains "${PARITY_UP}" 'render-redis-acl.sh'
assert_contains "${PARITY_UP}" 'render-observability.sh.*prod'
assert_contains "${PARITY_UP}" 'docker build'
assert_contains "${PARITY_UP}" '.*-f "\$\{REPO_ROOT\}/clients/web/Dockerfile"'
assert_contains "${PARITY_UP}" '"\$\{REPO_ROOT\}"'
assert_contains "${PARITY_UP}" 'compose --profile prod up -d --wait'
assert_contains "${PARITY_UP}" 'bootstrap-platform.sh" dev'
assert_contains "${PARITY_UP}" 'CASDOOR_BOOTSTRAP_ENABLED=true'
assert_contains "${PARITY_UP}" 'OPENFGA_BOOTSTRAP_API_URL="http://127\.0\.0\.1:8081"'
assert_contains "${PARITY_UP}" 'OPENFGA_BOOTSTRAP_DATABASE_URL="postgres://stuhelper_app:\$\{STUHELPER_APP_DB_PASSWORD\}@127\.0\.0\.1:\$\{PROD_PARITY_POSTGRES_PORT:-15432\}/stuhelper\?sslmode=disable"'
assert_contains "${PARITY_UP}" 'authorization-ledger-cutover.sh" dev'
assert_contains "${PARITY_UP}" 'OPENFGA_CUTOVER_API_URL="http://127\.0\.0\.1:8081"'
assert_contains "${PARITY_UP}" 'AUTHORIZATION_CUTOVER_DATABASE_URL="postgres://stuhelper_app:\$\{STUHELPER_APP_DB_PASSWORD\}@127\.0\.0\.1:\$\{PROD_PARITY_POSTGRES_PORT:-15432\}/stuhelper\?sslmode=disable"'
assert_contains "${PARITY_UP}" 'prod-parity-smoke.sh'
assert_contains "${PARITY_UP}" 'repo_default_path_matches'
assert_contains "${PARITY_UP}" 'install-local-prod-parity-ingress\.sh'
assert_not_contains "${PARITY_UP}" 'prod-deploy.sh'

assert_contains "${PARITY_LOCAL_INGRESS}" 'stuhelper\.com www\.stuhelper\.com join\.stuhelper\.com sso\.stuhelper\.com'
assert_contains "${PARITY_LOCAL_INGRESS}" 'PROXY_BYPASS_HOSTS'
assert_contains "${PARITY_LOCAL_INGRESS}" 'gsettings.*org\.gnome\.system\.proxy'
assert_contains "${PARITY_LOCAL_INGRESS}" '\*\.stuhelper\.com'
assert_contains "${PARITY_LOCAL_INGRESS}" 'DEFAULT_BAOTA_TLS_CERT'
assert_contains "${PARITY_LOCAL_INGRESS}" 'DEFAULT_GENERATED_TLS_CERT'
assert_contains "${PARITY_LOCAL_INGRESS}" 'ensure_local_tls'
assert_contains "${PARITY_LOCAL_INGRESS}" 'subjectAltName=DNS:stuhelper\.com,DNS:www\.stuhelper\.com,DNS:join\.stuhelper\.com,DNS:sso\.stuhelper\.com'
assert_contains "${PARITY_LOCAL_INGRESS}" 'listen 443 ssl'
assert_contains "${PARITY_LOCAL_INGRESS}" 'server_name sso\.stuhelper\.com'
assert_contains "${PARITY_LOCAL_INGRESS}" 'server_name join\.stuhelper\.com'
assert_contains "${PARITY_LOCAL_INGRESS}" 'location = /_app\.config\.js'
assert_contains "${PARITY_LOCAL_INGRESS}" 'location \^~ /css/'
assert_contains "${PARITY_LOCAL_INGRESS}" 'location \^~ /js/'
assert_contains "${PARITY_LOCAL_INGRESS}" 'location \^~ /jse/'
assert_contains "${PARITY_LOCAL_INGRESS}" 'location = /verify'
assert_contains "${PARITY_LOCAL_INGRESS}" 'location \^~ /verify/'
assert_contains "${PARITY_LOCAL_INGRESS}" 'location = /start'
assert_contains "${PARITY_LOCAL_INGRESS}" 'location \^~ /start/'
assert_contains "${PARITY_LOCAL_INGRESS}" 'location \^~ /admission/freshman/camera/'
assert_contains "${PARITY_LOCAL_INGRESS}" 'location \^~ /api/v1/admission/freshman/camera-handoffs/'
assert_contains "${PARITY_LOCAL_INGRESS}" 'location = /api/v1/bot/admission/actions/stream'
assert_contains "${PARITY_LOCAL_INGRESS}" 'X-Accel-Buffering no always'
assert_not_contains "${PARITY_LOCAL_INGRESS}" 'return 302 \$scheme://id\.stuhelper\.com\$request_uri'
assert_not_contains "${PARITY_LOCAL_INGRESS}" 'location = /favicon\.ico'
assert_not_contains "${PARITY_LOCAL_INGRESS}" 'location = /site\.webmanifest'
assert_contains "${PARITY_LOCAL_INGRESS}" 'nginx -t'
assert_contains "${PARITY_LOCAL_INGRESS}" 'nginx -s reload'
assert_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'server_name stuhelper\.com www\.stuhelper\.com'
assert_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'server_name join\.stuhelper\.com'
assert_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'server_name sso\.stuhelper\.com'
assert_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'location = /_app\.config\.js'
assert_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'location \^~ /css/'
assert_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'location \^~ /js/'
assert_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'location \^~ /jse/'
assert_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'location = /verify'
assert_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'location \^~ /verify/'
assert_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'location = /start'
assert_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'location \^~ /start/'
assert_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'location \^~ /admission/freshman/camera/'
assert_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'location \^~ /api/v1/admission/freshman/camera-handoffs/'
assert_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'location = /api/v1/bot/admission/actions/stream'
assert_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'X-Accel-Buffering no always'
assert_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'return 404'
assert_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'return 301 https://stuhelper\.com\$request_uri'
assert_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'return 301 https://join\.stuhelper\.com\$request_uri'
assert_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'proxy_pass http://127\.0\.0\.1:__BACKEND_PORT__'
assert_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'proxy_pass http://127\.0\.0\.1:__CASDOOR_PORT__'
assert_not_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'return 302 \$scheme://id\.stuhelper\.com\$request_uri'
assert_not_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'location \^~ /api/v1/[[:space:]]*\{'
assert_not_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'location \^~ /login/oauth/'
assert_not_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'location \^~ /developers/'
assert_not_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'location = /identity'
assert_not_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'location = /account/profile'
assert_not_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'location = /connect'
assert_not_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'location = /account/security'
assert_not_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'location = /user/authorized-apps'
assert_not_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'location = /favicon\.ico'
assert_not_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'location = /site\.webmanifest'
assert_not_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'return 302 /identity'
assert_not_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'return 302 \$scheme://stuhelper\.com\$request_uri'
assert_not_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'return 302 http://stuhelper\.com/developers/apps'
python3 - "${PARITY_LOCAL_INGRESS}" "${PARITY_LOCAL_INGRESS_NGINX}" <<'PY' || fail "prod-parity join root must return 404 instead of proxying to Web"
from pathlib import Path
import re
import sys

for filename in sys.argv[1:]:
    text = Path(filename).read_text(encoding="utf-8")
    server = re.search(r"server \{\n(?:(?!\nserver \{).)*server_name join\.stuhelper\.com;(?:(?!\nserver \{).)*\n\}", text, re.S)
    if not server:
        raise SystemExit(f"{filename}: missing join server")
    root = re.search(r"location / \{\n(?:(?!\n    \}).)*\n    \}", server.group(0), re.S)
    if not root:
        raise SystemExit(f"{filename}: missing join root location")
    block = root.group(0)
    if "return 404;" not in block or "proxy_pass" in block:
        raise SystemExit(f"{filename}: invalid join root location: {block}")
PY
assert_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'location = /_app\.config\.js'
assert_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'location \^~ /jse/'
assert_contains "${REPO_ROOT}/docker-compose.prod.yml" 'stuhelper\.com:host-gateway'
assert_not_contains "${REPO_ROOT}/docker-compose.prod.yml" 'id\.stuhelper\.com:host-gateway'
assert_contains "${REPO_ROOT}/docker-compose.prod.yml" 'join\.stuhelper\.com:host-gateway'
assert_contains "${REPO_ROOT}/docker-compose.prod.yml" 'sso\.stuhelper\.com:host-gateway'
assert_contains "${REPO_ROOT}/docker-compose.prod.yml" 'host\.docker\.internal:host-gateway'
assert_contains "${WEB_NGINX}" 'location /admission/freshman/camera/'
assert_contains "${WEB_NGINX}" 'location = /start'
assert_contains "${WEB_NGINX}" 'location /start/'
assert_contains "${WEB_NGINX}" 'location @spa_camera'
assert_contains "${WEB_NGINX}" 'Permissions-Policy "camera=\(self\), microphone=\(\), geolocation=\(\), payment=\(\)"'
assert_contains "${WEB_NGINX}" 'location @spa_default'

assert_contains "${PARITY_DOWN}" 'parity_default_path'
assert_contains "${PARITY_DOWN}" 'repo_default_path_matches'
assert_contains "${PARITY_SMOKE}" 'prod-parity-datastore-smoke.sh'
assert_contains "${PARITY_SMOKE}" 'prod-parity-smoke-data.sh'
assert_contains "${PARITY_SMOKE}" 'admission-production-readiness.sh'
assert_contains "${PARITY_SMOKE}" 'sso-public-smoke.sh'
assert_contains "${PARITY_SMOKE}" 'admission-public-smoke.sh'
assert_contains "${PARITY_SMOKE}" 'SMOKE_CHECK_CURL_INSECURE=true'
assert_contains "${PARITY_SMOKE}" 'SSO_PUBLIC_SMOKE_CURL_INSECURE=true'
assert_contains "${PARITY_SMOKE}" 'ADMISSION_PUBLIC_SMOKE_CURL_INSECURE=true'
assert_contains "${PARITY_SMOKE}" 'parity_default_path'
assert_contains "${PARITY_SMOKE}" 'repo_default_path_matches'
assert_contains "${PARITY_SMOKE}" 'APP_ENV:-prod-parity'
assert_contains "${PARITY_SMOKE}" 'preserved_app_env'
assert_not_contains "${PARITY_SMOKE}" 'APP_ENV=production'
assert_contains "${PARITY_SMOKE}" 'SSO_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS=true'
assert_contains "${PARITY_SMOKE}" 'SSO_PUBLIC_SMOKE_EXPECTED_ISSUER'
assert_contains "${PARITY_SMOKE}" 'ADMISSION_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS=true'
assert_contains "${PARITY_SMOKE}" 'openfga-resource-access-smoke.sh'
assert_contains "${PARITY_SMOKE}" 'prod-parity-browser-smoke.sh'
assert_contains "${PARITY_SMOKE}" 'admission-prod-sim-e2e.sh'
assert_contains "${PARITY_SMOKE}" 'observability-smoke-check.sh'
assert_contains "${PARITY_SMOKE}" 'OBS_SMOKE_STRICT=false'
assert_contains "${SMOKE_CHECK}" 'SMOKE_CHECK_CURL_INSECURE'
assert_contains "${SSO_PUBLIC_SMOKE}" 'SSO_PUBLIC_SMOKE_CURL_INSECURE'
assert_contains "${SSO_PUBLIC_SMOKE}" 'SSO discovery metadata'
assert_contains "${SSO_PUBLIC_SMOKE}" 'SSO JWKS'
assert_contains "${SSO_PUBLIC_SMOKE}" 'SSO authorize route reachable'
assert_contains "${ADMISSION_PUBLIC_SMOKE}" 'ADMISSION_PUBLIC_SMOKE_CURL_INSECURE'
assert_contains "${ADMISSION_PUBLIC_SMOKE}" 'https://join\.stuhelper\.com'
assert_contains "${ADMISSION_PUBLIC_SMOKE}" '/verify/\$\{probe_token\}'
assert_contains "${ADMISSION_PUBLIC_SMOKE}" '/start'
assert_contains "${ADMISSION_PUBLIC_SMOKE}" 'Web host join start returns 404'
assert_contains "${ADMISSION_PUBLIC_SMOKE}" 'Web host verify token returns 404'
assert_contains "${PARITY_UP}" 'EMAIL_DRIVER'
assert_contains "${PARITY_UP}" 'blackhole'

assert_contains "${PARITY_DATASTORE_SMOKE}" 'datastore-smoke-evidence\.json'
assert_contains "${PARITY_DATASTORE_SMOKE}" 'repo_default_path_matches'
assert_contains "${PARITY_DATASTORE_SMOKE}" 'has_database_privilege'
assert_contains "${PARITY_DATASTORE_SMOKE}" 'assert_pg_connect_allowed'
assert_contains "${PARITY_DATASTORE_SMOKE}" 'assert_pg_connect_denied'
assert_contains "${PARITY_DATASTORE_SMOKE}" 'Redis container must not join external datastore network'
assert_contains "${PARITY_DATASTORE_SMOKE}" 'Redis plaintext port must be disabled'
assert_contains "${PARITY_DATASTORE_SMOKE}" 'redis-cli --no-auth-warning --tls'
assert_contains "${PARITY_DATASTORE_SMOKE}" '/redis-runtime/users\.acl'
assert_contains "${PARITY_DATASTORE_SMOKE}" 'Redis application ACL must deny administrative CONFIG access'
assert_contains "${PARITY_DATASTORE_SMOKE}" 'Redis exporter ACL must deny writes'
assert_contains "${PARITY_DATASTORE_SMOKE}" 'casdoorChecked'

assert_contains "${PARITY_SMOKE_DATA}" 'smoke-data-evidence\.json'
assert_contains "${PARITY_SMOKE_DATA}" 'repo_default_path_matches'
assert_contains "${PARITY_SMOKE_DATA}" 'refusing to seed non prod-parity PostgreSQL container'
assert_contains "${PARITY_SMOKE_DATA}" '生产等价课程'
assert_contains "${PARITY_SMOKE_DATA}" '生产等价教师'
assert_contains "${PARITY_SMOKE_DATA}" '生产等价评课'
assert_contains "${PARITY_SMOKE_DATA}" 'PROD_PARITY_ADMISSION_TOKEN'
assert_contains "${PARITY_SMOKE_DATA}" 'admission_public_base_url'
assert_contains "${PARITY_SMOKE_DATA}" 'prod-parity-admission-policy'
assert_contains "${PARITY_SMOKE_DATA}" 'prod-parity-management'
assert_contains "${PARITY_SMOKE_DATA}" 'academic\.buaa_students'
assert_contains "${PARITY_SMOKE_DATA}" '20259901@buaa\.edu\.cn'
assert_contains "${PARITY_SMOKE_DATA}" 'emailIdentityPolicy'
assert_contains "${PARITY_SMOKE_DATA}" 'academic_student_email'
assert_contains "${PARITY_SMOKE_DATA}" 'admissionSchoolConfigCount'
assert_contains "${PARITY_SMOKE_DATA}" 'admissionAcademicStudentCount'
assert_contains "${PARITY_SMOKE_DATA}" 'admissionPolicyCount'
assert_contains "${PARITY_SMOKE_DATA}" 'prod-parity-admission-session'
assert_contains "${PARITY_SMOKE_DATA}" 'admissionSessionCount'
assert_contains "${PARITY_SMOKE_DATA}" 'PROD_PARITY_CASDOOR_LOGIN_USERNAME'
assert_contains "${PARITY_SMOKE_DATA}" 'casdoorStuhelperWebApplicationCount'
assert_contains "${PARITY_SMOKE_DATA}" 'casdoorAdmissionE2EUserCount'
assert_contains "${PARITY_SMOKE_DATA}" 'organization = '\''stuhelper'\'''
assert_contains "${ADMISSION_READINESS}" 'ADMISSION_PUBLIC_BASE_URL must be exactly https://join.stuhelper.com'
assert_contains "${ADMISSION_READINESS}" 'ADMISSION_READINESS_REQUIRED_GUILD_IDS'
assert_contains "${ADMISSION_READINESS}" 'ADMISSION_READINESS_REQUIRED_SCHOOL_CODES'
assert_contains "${ADMISSION_READINESS}" 'group_admission_policies'
assert_contains "${ADMISSION_READINESS}" 'management_guild_ids must not be empty'
assert_contains "${ADMISSION_READINESS}" 'admission production readiness passed'
assert_contains "${PARITY_SMOKE_DATA}" 'hmac\.new'
assert_contains "${PARITY_SMOKE_DATA}" 'REFRESH MATERIALIZED VIEW public\.mv_teacher_public_stats'
assert_contains "${PARITY_SMOKE_DATA}" "course:\\*"
assert_contains "${PARITY_SMOKE_DATA}" "review:\\*"
assert_contains "${PARITY_SMOKE_DATA}" 'courseRatingStatsCount'
assert_contains "${PARITY_SMOKE_DATA}" 'teacherPublicStatsCount'

assert_contains "${PARITY_BROWSER_SMOKE}" 'PROD_PARITY_BROWSER_SMOKE_EVIDENCE_FILE'
assert_contains "${PARITY_BROWSER_SMOKE}" 'repo_default_path_matches'
assert_contains "${PARITY_BROWSER_SMOKE}" 'browser-smoke-evidence\.json'
assert_contains "${PARITY_BROWSER_SMOKE}" 'WEB_BASE_URL'
assert_contains "${PARITY_BROWSER_SMOKE}" 'ADMIN_BASE_URL'
assert_contains "${PARITY_BROWSER_SMOKE}" 'clear_rate_limit_keys'
assert_contains "${PARITY_BROWSER_SMOKE}" 'append_no_proxy'
assert_contains "${PARITY_BROWSER_SMOKE}" 'prod-parity-smoke-data.sh'
assert_contains "${PARITY_BROWSER_SMOKE}" "scan --pattern 'rl:\*'"
assert_contains "${PARITY_BROWSER_SMOKE}" 'prod-parity-browser-smoke\.mjs'
assert_contains "${PARITY_UP}" 'API_IP_RATE_LIMIT'
assert_contains "${PARITY_UP}" 'API_GLOBAL_RATE_LIMIT'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" '@playwright/test'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'accountBaseURL'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'admissionBaseURL'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'ADMISSION_PUBLIC_BASE_URL'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'CASDOOR_PUBLIC_AUTH_BASE_URL'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'telemetryRoutePattern'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'viewportVariants'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" "name: 'desktop'"
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" "name: 'mobile'"
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'deviceScaleFactor'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'hasTouch'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'suppressedTelemetryRequests'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'stubbedResources'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'stubbedExternalResources'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'fonts.googleapis.com'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'cdn.casbin.org/flag-icons'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'allowedAPIResponses'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'ignoreHTTPSErrors: true'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" '\-\-no-proxy-server'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'ignoredAPIResponses'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-home'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-login'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'frontend-direct-home'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'account-developer-login'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'account-connect-public'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-account-connect-redirect'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'identity-account-profile-authenticated'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-protected-account-profile'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'identity-account-security-authenticated'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-protected-account-security'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'identity-home-authenticated'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'identity-authorized-apps-authenticated'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'account-developer-apps-authenticated'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'identity-identity-verification-authenticated'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'identity-student-verification-authenticated'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'identity-phone-binding-authenticated'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'identity-qq-binding-authenticated'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'identity-academic-info-authenticated'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'identity-main-route-redirect'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-user-center-business-tabs-authenticated'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'runWebAuthenticatedRefreshFlow'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-login-session-refresh'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'runLoginSessionRefreshFlow'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'runIdentityAuthenticatedRefreshFlow'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'appShellHeader'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" "getByRole\\('banner'\\)"
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'auth/me after browser refresh returned'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'auth/me after identity refresh returned'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-auth-callback-missing-code'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-admission-login'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'admission-mobile-camera-permission'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'admission-sse-ingress'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'runAdmissionSSEIngressFlow'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'notificationSSEPath'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'installSmokeBrowserStubs'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'PROD_PARITY_BROWSER_SMOKE_CLOSE_TIMEOUT_MS'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'closeWithTimeout'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'expectedResponseHeaders'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'permissions-policy'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'camera=\(self\)'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'x-accel-buffering'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-course-hub'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-course-list'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-course-detail'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-course-detail-reviews'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-review-feed'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-search'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-teacher-hub'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-teacher-profile'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-protected-review-post'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-protected-user-reviews'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-protected-user-votes'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-protected-user-favorites'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-protected-identity-home'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-protected-user-authorized-apps'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-protected-identity-verification'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-protected-student-verification'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-protected-phone-binding'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-protected-qq-binding'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-protected-academic-info'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-protected-notifications'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-protected-developer-apps'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-protected-open-platform-consent'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-protected-profile-completion'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-not-found'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'admin-login-redirect'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'checkName'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'viewportSize'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'finalURL'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'matchedText'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'requiredTexts'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'missingRequiredTexts'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'admission-e2e'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'fillCasdoorPasswordLogin'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'forbiddenTexts'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'presentForbiddenTexts'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'expectedURLIncludes'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'webRedirectQuery'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'toArray'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" "page\.on\('console'"
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'consoleErrors'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'ignoredConsoleErrors'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'describeConsoleMessage'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'isIgnoredConsoleError'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" '\\\(\[\^\)\]\*\\\)'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'browser\.newContext'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'pageerror'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'requestfailed'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" "page\.on\('response'"
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'response\.status\(\) >= 400'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" "'fetch', 'xhr'"
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'apiFailures'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'criticalResourceTypes'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" "'image'"
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" "'font'"

assert_contains "${ADMISSION_PROD_SIM_E2E}" 'prod-parity-smoke-data.sh'
assert_contains "${ADMISSION_PROD_SIM_E2E}" 'ADMISSION_PROD_SIM_E2E_EVIDENCE_FILE'
assert_contains "${ADMISSION_PROD_SIM_E2E}" 'ADMISSION_PROD_SIM_SSO_BASE_URL'
assert_contains "${ADMISSION_PROD_SIM_E2E}" 'append_no_proxy'
assert_contains "${ADMISSION_PROD_SIM_E2E}" 'admission-prod-sim-e2e\.mjs'
assert_contains "${ADMISSION_PROD_SIM_E2E}" "scan --pattern 'rl:\*'"
assert_contains "${ADMISSION_PROD_SIM_E2E_NODE}" '@playwright/test'
assert_contains "${ADMISSION_PROD_SIM_E2E_NODE}" 'BOT_SERVICE_TOKEN'
assert_contains "${ADMISSION_PROD_SIM_E2E_NODE}" 'backend previews just-created admission token'
assert_contains "${ADMISSION_PROD_SIM_E2E_NODE}" '/api/v1/bot/admission/sessions'
assert_contains "${ADMISSION_PROD_SIM_E2E_NODE}" '/api/v1/admission/sessions/'
assert_contains "${ADMISSION_PROD_SIM_E2E_NODE}" '/api/v1/bot/admission/sessions/pending'
assert_contains "${ADMISSION_PROD_SIM_E2E_NODE}" '/events'
assert_contains "${ADMISSION_PROD_SIM_E2E_NODE}" 'https://join\.stuhelper\.com'
assert_contains "${ADMISSION_PROD_SIM_E2E_NODE}" 'https://sso\.stuhelper\.com'
assert_contains "${ADMISSION_PROD_SIM_E2E_NODE}" 'prod-parity-guild'
assert_contains "${ADMISSION_PROD_SIM_E2E_NODE}" '20259901'
assert_contains "${ADMISSION_PROD_SIM_E2E_NODE}" '4111010006'
assert_contains "${ADMISSION_PROD_SIM_E2E_NODE}" 'admission:email_otp:'
assert_contains "${ADMISSION_PROD_SIM_E2E_NODE}" 'credentialKind'
assert_contains "${ADMISSION_PROD_SIM_E2E_NODE}" 'school_email_otp'
assert_contains "${ADMISSION_PROD_SIM_E2E_NODE}" 'release'
assert_contains "${ADMISSION_PROD_SIM_E2E_NODE}" 'cancelledAtPresent'
assert_contains "${ADMISSION_PROD_SIM_E2E_NODE}" 'redactAdmissionURL'
assert_contains "${ADMISSION_PROD_SIM_E2E_NODE}" '/api/v1/admission/sessions/redacted'
assert_contains "${ADMISSION_PROD_SIM_E2E_NODE}" 'ignoreHTTPSErrors: true'
assert_contains "${ADMISSION_PROD_SIM_E2E_NODE}" '\-\-no-proxy-server'
assert_contains "${ADMISSION_PROD_SIM_E2E_NODE}" 'browser diagnostics failed'
assert_contains "${ADMISSION_PROD_SIM_E2E_NODE}" 'fillCasdoorPasswordLogin'
assert_contains "${ADMISSION_PROD_SIM_E2E_NODE}" 'sanitizeErrorMessage'
assert_contains "${ADMISSION_PROD_SIM_E2E_NODE}" 'me\|refresh'
assert_contains "${ADMISSION_PROD_SIM_E2E_NODE}" 'response\.status\(\) === 401\) return'

assert_not_contains "${ADMIN_INDEX_HTML}" 'hm\.baidu\.com'
assert_not_contains "${ADMIN_INDEX_HTML}" '_VBEN_ADMIN_PRO_APP_CONF_'

assert_contains "${WEB_NGINX}" 'location / \{'
assert_contains "${WEB_NGINX}" 'location /verify/ \{'
assert_contains "${WEB_NGINX}" 'try_files \$uri \$uri/ @spa_default'
assert_contains "${WEB_NGINX}" 'try_files \$uri \$uri/ @spa_camera'
assert_contains "${WEB_NGINX}" 'add_header Cache-Control "no-store, no-cache, must-revalidate" always;'
assert_contains "${WEB_NGINX}" 'location /assets/ \{'
assert_contains "${WEB_NGINX}" 'add_header Cache-Control "public, immutable";'

assert_contains "${MAKEFILE}" 'prod-parity-up'
assert_contains "${MAKEFILE}" 'prod-parity-ingress'
assert_contains "${MAKEFILE}" 'prod-parity-down'
assert_contains "${MAKEFILE}" 'prod-parity-smoke'
assert_contains "${MAKEFILE}" 'prod-parity-datastore-smoke'
assert_contains "${MAKEFILE}" 'prod-parity-admission-e2e'

echo "[prod-parity-contract] all assertions passed"
