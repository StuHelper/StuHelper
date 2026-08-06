#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

require_cmd docker
require_cmd curl
require_cmd jq
require_cmd python3

PARITY_DIR="${PROD_PARITY_DIR:-${REPO_ROOT}/.run/prod-parity}"
mkdir -p "${PARITY_DIR}"
chmod 700 "${PARITY_DIR}" 2>/dev/null || true

parity_default_path() {
  local current="$1"
  local common_default="$2"
  local parity_default="$3"
  if repo_default_path_matches "${current}" "${common_default}"; then
    printf '%s\n' "${parity_default}"
    return
  fi
  printf '%s\n' "${current}"
}

export ENV_TEMPLATE_FILE="${REPO_ROOT}/.env.prod.example"
export ENV_FILE="$(parity_default_path "${ENV_FILE:-}" "${REPO_ROOT}/.env" "${PARITY_DIR}/.env.prod.shared")"
export SECRETS_ENV_FILE="$(parity_default_path "${SECRETS_ENV_FILE:-}" "" "${PARITY_DIR}/.env.prod.secrets.local")"
export GENERATED_ENV_FILE="$(parity_default_path "${GENERATED_ENV_FILE:-}" "${REPO_ROOT}/.env.generated" "${PARITY_DIR}/.env.prod.generated")"
export GENERATED_SECRET_ENV_FILE="$(parity_default_path "${GENERATED_SECRET_ENV_FILE:-}" "${REPO_ROOT}/.env.generated.secrets" "${PARITY_DIR}/.env.prod.generated.secrets")"
export DEPLOY_STATE_DIR="$(parity_default_path "${DEPLOY_STATE_DIR:-}" "${REPO_ROOT}/.deploy" "${PARITY_DIR}/deploy-state")"
export CASDOOR_BOOTSTRAP_ENV_FILE="$(parity_default_path "${CASDOOR_BOOTSTRAP_ENV_FILE:-}" "" "${PARITY_DIR}/.env.casdoor-bootstrap.local")"

touch "${ENV_FILE}" "${SECRETS_ENV_FILE}" "${GENERATED_ENV_FILE}" "${GENERATED_SECRET_ENV_FILE}" "${CASDOOR_BOOTSTRAP_ENV_FILE}"
chmod 600 "${ENV_FILE}" "${SECRETS_ENV_FILE}" "${GENERATED_ENV_FILE}" "${GENERATED_SECRET_ENV_FILE}" "${CASDOOR_BOOTSTRAP_ENV_FILE}" 2>/dev/null || true

ensure_file_value() {
  local file="$1"
  local key="$2"
  local value="$3"
  upsert_env_file "${file}" "${key}" "${value}"
}

ensure_file_secret() {
  local file="$1"
  local key="$2"
  local prefix="$3"
  if ! grep -Eq "^${key}=" "${file}"; then
    upsert_env_file "${file}" "${key}" "${prefix}-$(random_hex 16)"
  fi
}

ensure_bootstrap_value() {
  local key="$1"
  local value="$2"
  upsert_env_file "${CASDOOR_BOOTSTRAP_ENV_FILE}" "${key}" "${value}"
}

append_local_no_proxy() {
  local hosts="127.0.0.1,localhost,::1,stuhelper.com,www.stuhelper.com,join.stuhelper.com,sso.stuhelper.com,.stuhelper.com"
  if [[ -n "${NO_PROXY:-}" ]]; then
    export NO_PROXY="${NO_PROXY},${hosts}"
  else
    export NO_PROXY="${hosts}"
  fi
  if [[ -n "${no_proxy:-}" ]]; then
    export no_proxy="${no_proxy},${hosts}"
  else
    export no_proxy="${hosts}"
  fi
}

append_local_no_proxy

sync_casdoor_builtin_bootstrap_credentials() {
  local postgres_container="${SHARED_POSTGRES_CONTAINER:-${PROD_PARITY_POSTGRES_CONTAINER:-stuhelper-prod-parity-postgres}}"
  local superuser="${SHARED_POSTGRES_SUPERUSER:-postgres}"
  local casdoor_db="${CASDOOR_DB_NAME:-casdoor}"
  local credentials
  local client_id
  local client_secret

  credentials="$(
    docker exec -i "${postgres_container}" \
      psql -At -F $'\t' \
        -U "${superuser}" \
        -d "${casdoor_db}" \
        -c "SELECT client_id, client_secret FROM application WHERE name = 'app-built-in' AND organization = 'built-in' LIMIT 1"
  )"
  IFS=$'\t' read -r client_id client_secret <<<"${credentials}"

  [[ -n "${client_id}" ]] || die "failed to read Casdoor built-in application client_id from ${casdoor_db}"
  [[ -n "${client_secret}" ]] || die "failed to read Casdoor built-in application client_secret from ${casdoor_db}"

  ensure_bootstrap_value "CASDOOR_BOOTSTRAP_CLIENT_ID" "${client_id}"
  ensure_bootstrap_value "CASDOOR_BOOTSTRAP_CLIENT_SECRET" "${client_secret}"
  ensure_bootstrap_value "CASDOOR_BOOTSTRAP_APPLICATION" "app-built-in"
  ensure_bootstrap_value "CASDOOR_BOOTSTRAP_CERTIFICATE" "cert-built-in"
  ensure_bootstrap_value "CASDOOR_BOOTSTRAP_ORGANIZATION" "built-in"
}

tag="${PROD_PARITY_TAG:-prod-parity-$(git_tag_default)}"
commit="$(git -C "${REPO_ROOT}" rev-parse --short HEAD 2>/dev/null || echo "local")"
build_time="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

ensure_file_value "${ENV_FILE}" "STACK_NAME" "stuhelper-prod-parity"
ensure_file_value "${ENV_FILE}" "COMPOSE_PROJECT_NAME" "stuhelper-prod-parity"
ensure_file_value "${ENV_FILE}" "APP_ENV" "prod-parity"
ensure_file_value "${ENV_FILE}" "LOG_LEVEL" "info"
ensure_file_value "${ENV_FILE}" "LOG_FORMAT" "json"
ensure_file_value "${ENV_FILE}" "LOG_OUTPUT" "stdout"
ensure_file_value "${ENV_FILE}" "TAG" "${tag}"
ensure_file_value "${ENV_FILE}" "BACKEND_IMAGE_REF" "stuhelper/backend:${tag}"
ensure_file_value "${ENV_FILE}" "FRONTEND_IMAGE_REF" "stuhelper/frontend:${tag}"
ensure_file_value "${ENV_FILE}" "ADMIN_IMAGE_REF" "stuhelper/admin:${tag}"
ensure_file_value "${ENV_FILE}" "EXTERNAL_POSTGRES_ENABLED" "true"
ensure_file_value "${ENV_FILE}" "EXTERNAL_POSTGRES_ALLOW_PLAINTEXT" "true"
ensure_file_value "${ENV_FILE}" "EXTERNAL_DATASTORE_NETWORK" "stuhelper-prod-parity-baota-net"
ensure_file_value "${ENV_FILE}" "SHARED_POSTGRES_CONTAINER" "stuhelper-prod-parity-postgres"
ensure_file_value "${ENV_FILE}" "PROD_PARITY_POSTGRES_CONTAINER" "stuhelper-prod-parity-postgres"
ensure_file_value "${ENV_FILE}" "PROD_PARITY_POSTGRES_PORT" "15432"
ensure_file_value "${ENV_FILE}" "SHARED_POSTGRES_SUPERUSER" "postgres"
ensure_file_value "${ENV_FILE}" "SHARED_POSTGRES_DB" "postgres"
ensure_file_value "${ENV_FILE}" "POSTGRES_DB" "stuhelper"
ensure_file_value "${ENV_FILE}" "POSTGRES_HOST" "postgres"
ensure_file_value "${ENV_FILE}" "POSTGRES_INTERNAL_SSL_MODE" "disable"
ensure_file_value "${ENV_FILE}" "POSTGRES_ENABLE_SSL" "off"
ensure_file_value "${ENV_FILE}" "DB_SSL_MODE" "disable"
ensure_file_value "${ENV_FILE}" "POSTGRES_EXPORTER_DATA_SOURCE_URI" "postgres:5432/postgres?sslmode=disable"
ensure_file_value "${ENV_FILE}" "DATABASE_URL" "postgres://stuhelper_app:${STUHELPER_APP_DB_PASSWORD:-REPLACE_WITH_STUHELPER_APP_DB_PASSWORD}@postgres:5432/stuhelper?sslmode=disable"
ensure_file_value "${ENV_FILE}" "BACKUP_DATABASE_URL" "postgres://stuhelper_backup:${STUHELPER_BACKUP_DB_PASSWORD:-REPLACE_WITH_STUHELPER_BACKUP_DB_PASSWORD}@postgres:5432/stuhelper?sslmode=disable"
ensure_file_value "${ENV_FILE}" "REPLICATION_DATABASE_URL" "postgres://stuhelper_replication:${STUHELPER_REPLICATION_DB_PASSWORD:-REPLACE_WITH_STUHELPER_REPLICATION_DB_PASSWORD}@postgres:5432/stuhelper?sslmode=disable"
ensure_file_value "${ENV_FILE}" "REDIS_HOST" "redis"
ensure_file_value "${ENV_FILE}" "REDIS_PORT" "6379"
ensure_file_value "${ENV_FILE}" "REDIS_EXTERNAL_PORT" "26379"
ensure_file_value "${ENV_FILE}" "REDIS_USERNAME" "stuhelper_app"
ensure_file_value "${ENV_FILE}" "REDIS_EXPORTER_USERNAME" "stuhelper_metrics"
ensure_file_value "${ENV_FILE}" "REDIS_PROD_PARITY_MAINTENANCE_USERNAME" "stuhelper_parity_maintenance"
ensure_file_value "${ENV_FILE}" "REDIS_TLS_ENABLED" "true"
ensure_file_value "${ENV_FILE}" "REDIS_TLS_CA" "/redis-tls/ca.crt"
ensure_file_value "${ENV_FILE}" "WEB_PUBLIC_URL" "https://stuhelper.com"
ensure_file_value "${ENV_FILE}" "ADMIN_PUBLIC_URL" "https://stuhelper.com/admin/"
ensure_file_value "${ENV_FILE}" "ADMISSION_PUBLIC_BASE_URL" "https://join.stuhelper.com"
ensure_file_value "${ENV_FILE}" "ADMISSION_PRODUCTION_READINESS_ENABLED" "true"
ensure_file_value "${ENV_FILE}" "ADMISSION_READINESS_REQUIRED_PLATFORM" "qq"
ensure_file_value "${ENV_FILE}" "ADMISSION_READINESS_REQUIRED_GUILD_IDS" "prod-parity-guild"
ensure_file_value "${ENV_FILE}" "ADMISSION_READINESS_REQUIRED_SCHOOL_CODES" "4111010006"
ensure_file_value "${ENV_FILE}" "ADMISSION_READINESS_REQUIRED_SCHOOL_IDS" ""
ensure_file_value "${ENV_FILE}" "WEB_VITE_API_URL" "/api"
ensure_file_value "${ENV_FILE}" "WEB_VITE_SSO_URL" "https://sso.stuhelper.com"
ensure_file_value "${ENV_FILE}" "WEB_VITE_WEB_URL" "https://stuhelper.com"
ensure_file_value "${ENV_FILE}" "WEB_VITE_API_TIMEOUT_MS" "15000"
ensure_file_value "${ENV_FILE}" "WEB_VITE_QQ_BOT_ENTRY" ""
ensure_file_value "${ENV_FILE}" "WEB_VITE_QQ_BIND_COMMAND" "绑定"
ensure_file_value "${ENV_FILE}" "ADMIN_VITE_API_URL" "/api/v1"
ensure_file_value "${ENV_FILE}" "ADMIN_VITE_BASE" "/admin/"
ensure_file_value "${ENV_FILE}" "BACKEND_EXTERNAL_PORT" "28080"
ensure_file_value "${ENV_FILE}" "WEB_EXTERNAL_PORT" "28000"
ensure_file_value "${ENV_FILE}" "ADMIN_EXTERNAL_PORT" "28001"
ensure_file_value "${ENV_FILE}" "OPENFGA_API_URL" "http://openfga:8080"
ensure_file_value "${ENV_FILE}" "OPENFGA_HTTP_EXTERNAL_PORT" "8081"
ensure_file_value "${ENV_FILE}" "OPENFGA_GRPC_EXTERNAL_PORT" "8082"
ensure_file_value "${ENV_FILE}" "OPENFGA_PLAYGROUND_EXTERNAL_PORT" "3002"
ensure_file_value "${ENV_FILE}" "OPENFGA_RESOURCE_SMOKE_MODE" "container"
ensure_file_value "${ENV_FILE}" "CASDOOR_EXTERNALPORT" "28085"
ensure_file_value "${ENV_FILE}" "CASDOOR_ISSUER" "http://sso.stuhelper.com"
ensure_file_value "${ENV_FILE}" "CASDOOR_PUBLIC_AUTH_BASE_URL" "https://sso.stuhelper.com"
ensure_file_value "${ENV_FILE}" "CASDOOR_INTERNAL_ADDRESS" "host.docker.internal:80"
ensure_file_value "${ENV_FILE}" "CASDOOR_REDIRECT_URI" "https://stuhelper.com/api/v1/auth/callback"
ensure_file_value "${ENV_FILE}" "CASDOOR_CLIENT_ID" "stuhelper-web"
ensure_file_value "${ENV_FILE}" "CASDOOR_ORGANIZATION" "stuhelper"
ensure_file_value "${ENV_FILE}" "CASDOOR_BOOTSTRAP_ENABLED" "true"
ensure_file_value "${ENV_FILE}" "CASDOOR_BOOTSTRAP_ENV_FILE" "${CASDOOR_BOOTSTRAP_ENV_FILE}"
ensure_file_value "${ENV_FILE}" "CASDOOR_ADMIN_CLIENT_ID" "stuhelper-admin"
ensure_file_value "${ENV_FILE}" "CASDOOR_ADMIN_REDIRECT_URI" "https://stuhelper.com/api/v1/auth/callback"
ensure_file_value "${ENV_FILE}" "CASDOOR_UNIAPP_CLIENT_ID" "stuhelper-uniapp"
ensure_file_value "${ENV_FILE}" "CASDOOR_UNIAPP_REDIRECT_URI" "https://stuhelper.com/api/v1/auth/callback"
ensure_file_value "${ENV_FILE}" "CASDOOR_APP_PROVISIONING_CLIENT_ID" "casdoor-admin-app-provisioning"
ensure_file_value "${ENV_FILE}" "CASDOOR_APP_PROVISIONING_APPLICATION" "casdoor-admin-app-provisioning"
ensure_file_value "${ENV_FILE}" "CASDOOR_USER_PROFILE_CLIENT_ID" "casdoor-admin-user-profile"
ensure_file_value "${ENV_FILE}" "CASDOOR_USER_PROFILE_APPLICATION" "casdoor-admin-user-profile"
ensure_file_value "${ENV_FILE}" "CASDOOR_INTROSPECTION_CLIENT_ID" "casdoor-token-introspection"
ensure_file_value "${ENV_FILE}" "CASDOOR_INTROSPECTION_APPLICATION" "casdoor-token-introspection"
ensure_file_value "${ENV_FILE}" "CASDOOR_USER_LOOKUP_CLIENT_ID" "casdoor-admin-user-lookup"
ensure_file_value "${ENV_FILE}" "CASDOOR_USER_LOOKUP_APPLICATION" "casdoor-admin-user-lookup"
ensure_file_value "${ENV_FILE}" "CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_ID" "casdoor-token-probe-smoke"
ensure_file_value "${ENV_FILE}" "CASDOOR_TOKEN_PROBE_SMOKE_APPLICATION" "casdoor-token-probe-smoke"
ensure_file_value "${ENV_FILE}" "CASDOOR_TOKEN_PROBE_SMOKE_REDIRECT_URI" "https://stuhelper.com/open-platform/token-probe/callback"
ensure_file_value "${ENV_FILE}" "CASDOOR_SMS_PROVIDER_ENABLED" "true"
ensure_file_value "${ENV_FILE}" "CASDOOR_SMS_PROVIDER_NAME" "stuhelper-sms"
ensure_file_value "${ENV_FILE}" "CASDOOR_SMS_PROVIDER_DISPLAY_NAME" "StuHelper-SMS"
ensure_file_value "${ENV_FILE}" "CASDOOR_SMS_PROVIDER_CATEGORY" "SMS"
ensure_file_value "${ENV_FILE}" "CASDOOR_SMS_PROVIDER_TYPE" "CustomHTTP"
ensure_file_value "${ENV_FILE}" "CASDOOR_SMS_PROVIDER_METHOD" "POST"
ensure_file_value "${ENV_FILE}" "CASDOOR_SMS_PROVIDER_TITLE" "content"
ensure_file_value "${ENV_FILE}" "CASDOOR_SMS_PROVIDER_ENDPOINT" "http://app:8080/internal/sms/send"
ensure_file_value "${ENV_FILE}" "CASDOOR_EMAIL_PROVIDER_ENABLED" "false"
ensure_file_value "${ENV_FILE}" "SMS_ENABLED" "true"
ensure_file_value "${ENV_FILE}" "SMS_APP_ID" "prod-parity-sms-app"
ensure_file_value "${ENV_FILE}" "SMS_SIGN_NAME" "StuHelper"
ensure_file_value "${ENV_FILE}" "SMS_TEMPLATE_ID" "prod-parity-template"
ensure_file_value "${ENV_FILE}" "SMS_REGION" "ap-beijing"
ensure_file_value "${ENV_FILE}" "EMAIL_ENABLED" "true"
ensure_file_value "${ENV_FILE}" "EMAIL_DRIVER" "blackhole"
ensure_file_value "${ENV_FILE}" "EMAIL_FROM" "no-reply@stuhelper.local"
ensure_file_value "${ENV_FILE}" "OPEN_PLATFORM_CONSENT_BASE_URL" "https://stuhelper.com"
ensure_file_value "${ENV_FILE}" "OPEN_PLATFORM_ACCOUNT_BASE_URL" "https://stuhelper.com"
ensure_file_value "${ENV_FILE}" "OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_REQUIRED" "true"
ensure_file_value "${ENV_FILE}" "OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND" "/app/casdoor-runtime-token-probe-runner.mjs"
ensure_file_value "${ENV_FILE}" "OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_TIMEOUT_SECONDS" "30"
ensure_file_value "${ENV_FILE}" "OPEN_PLATFORM_PRODUCTION_EVIDENCE_ALLOW_LOCAL_TARGETS" "true"
ensure_file_value "${ENV_FILE}" "API_IP_RATE_LIMIT" "5000"
ensure_file_value "${ENV_FILE}" "API_GLOBAL_RATE_LIMIT" "50000"
ensure_file_value "${ENV_FILE}" "PUBLIC_INGRESS_CASDOOR_UPSTREAM_PREFLIGHT_ENABLED" "true"
ensure_file_value "${ENV_FILE}" "SSO_PUBLIC_SMOKE_ENABLED" "true"
ensure_file_value "${ENV_FILE}" "SSO_PUBLIC_BASE_URL" "https://sso.stuhelper.com"
ensure_file_value "${ENV_FILE}" "SSO_PUBLIC_SMOKE_EXPECTED_ISSUER" "http://sso.stuhelper.com"
ensure_file_value "${ENV_FILE}" "SSO_PUBLIC_SMOKE_CLIENT_ID" "stuhelper-web"
ensure_file_value "${ENV_FILE}" "SSO_PUBLIC_SMOKE_REDIRECT_URI" "https://stuhelper.com/api/v1/auth/callback"
ensure_file_value "${ENV_FILE}" "SSO_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS" "true"
ensure_file_value "${ENV_FILE}" "SSO_PUBLIC_SMOKE_CURL_INSECURE" "true"
ensure_file_value "${ENV_FILE}" "CORS_ORIGINS" "https://stuhelper.com,https://join.stuhelper.com,https://sso.stuhelper.com"
ensure_file_value "${ENV_FILE}" "STUHELPER_FRESHMAN_MATERIAL_HOSTS" "stuhelper.com,join.stuhelper.com"
ensure_file_value "${ENV_FILE}" "TRUSTED_PROXIES" "127.0.0.1/32,172.16.0.0/12,192.168.0.0/16"
ensure_file_value "${ENV_FILE}" "TOKEN_COOKIE_SECURE" "true"
ensure_file_value "${ENV_FILE}" "TOKEN_COOKIE_DOMAIN" ".stuhelper.com"
ensure_file_value "${ENV_FILE}" "OTEL_ENABLED" "true"
ensure_file_value "${ENV_FILE}" "OTEL_SERVICE_NAME" "stuhelper-backend"
ensure_file_value "${ENV_FILE}" "OTEL_SERVICE_NAMESPACE" "stuhelper"
ensure_file_value "${ENV_FILE}" "FRONTEND_METRICS_ALLOWED_ORIGINS" "https://stuhelper.com,https://join.stuhelper.com"
ensure_file_value "${ENV_FILE}" "OTEL_EXPORTER_OTLP_ENDPOINT" "http://alloy:4318"
ensure_file_value "${ENV_FILE}" "OTEL_EXPORTER_OTLP_INSECURE" "true"
ensure_file_value "${ENV_FILE}" "OBJECT_STORAGE_ENDPOINT" "https://object-storage:8334"
ensure_file_value "${ENV_FILE}" "OBJECT_STORAGE_REGION" "us-east-1"
ensure_file_value "${ENV_FILE}" "OBJECT_STORAGE_BUCKET" "stuhelper-identity"
ensure_file_value "${ENV_FILE}" "OBJECT_STORAGE_ACCESS_KEY_ID" "stuhelper-prod-parity"
ensure_file_value "${ENV_FILE}" "OBJECT_STORAGE_USE_SSL" "true"
ensure_file_value "${ENV_FILE}" "OBJECT_STORAGE_FORCE_PATH_STYLE" "true"
ensure_file_value "${ENV_FILE}" "OBJECT_STORAGE_PRESIGN_TTL" "600"
ensure_file_value "${ENV_FILE}" "OBJECT_STORAGE_TLS_CA" "/object-storage-tls/ca.crt"
ensure_file_value "${ENV_FILE}" "OBJECT_STORAGE_TLS_CA_HOST_PATH" "${REPO_ROOT}/infra/generated/object-storage/ca.crt"
ensure_file_value "${ENV_FILE}" "BACKUP_OBJECT_STORAGE_ENDPOINT" "https://object-storage:8334"
ensure_file_value "${ENV_FILE}" "BACKUP_OBJECT_STORAGE_BUCKET" "stuhelper-postgres-backup"
ensure_file_value "${ENV_FILE}" "BACKUP_OBJECT_STORAGE_PREFIX" "postgres"
ensure_file_value "${ENV_FILE}" "BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID" "stuhelper-prod-parity-backup"
ensure_file_value "${ENV_FILE}" "BACKUP_OBJECT_STORAGE_TLS_CA" "${REPO_ROOT}/infra/generated/object-storage/ca.crt"
ensure_file_value "${ENV_FILE}" "BACKUP_OBJECT_STORAGE_TLS_INSECURE" "false"
ensure_file_value "${ENV_FILE}" "BACKUP_OBJECT_STORAGE_DOCKER_NETWORK" "stuhelper-prod-parity-backend"
ensure_file_value "${ENV_FILE}" "DEV_OBJECT_STORAGE_EXTERNAL_PORT" "29000"
ensure_file_value "${ENV_FILE}" "LOCAL_OBJECT_STORAGE_TLS_EXTERNAL_PORT" "29001"
ensure_file_value "${ENV_FILE}" "GRAFANA_ROOT_URL" "http://127.0.0.1:23003"
ensure_file_value "${ENV_FILE}" "ALLOW_LOCAL_ALERT_SINK" "true"
ensure_file_value "${ENV_FILE}" "ALERTMANAGER_WEBHOOK_URL" "http://alert-webhook-sink:8080/alerts"
ensure_file_value "${ENV_FILE}" "ALERTMANAGER_CONFIG_GID" "$(id -g)"
ensure_file_value "${ENV_FILE}" "ALLOY_HTTP_PORT" "22345"
ensure_file_value "${ENV_FILE}" "OTEL_GRPC_PORT" "24317"
ensure_file_value "${ENV_FILE}" "OTEL_HTTP_PORT" "24318"
ensure_file_value "${ENV_FILE}" "PROMETHEUS_PORT" "29090"
ensure_file_value "${ENV_FILE}" "ALERTMANAGER_PORT" "29093"
ensure_file_value "${ENV_FILE}" "LOKI_PORT" "23100"
ensure_file_value "${ENV_FILE}" "TEMPO_HTTP_PORT" "23200"
ensure_file_value "${ENV_FILE}" "GRAFANA_PORT" "23003"
ensure_file_value "${ENV_FILE}" "CADVISOR_PORT" "28088"
ensure_file_value "${ENV_FILE}" "POSTGRES_EXPORTER_PORT" "29187"
ensure_file_value "${ENV_FILE}" "REDIS_EXPORTER_PORT" "29121"
ensure_file_value "${ENV_FILE}" "BLACKBOX_EXPORTER_PORT" "29115"

ensure_file_secret "${SECRETS_ENV_FILE}" "SHARED_POSTGRES_PASSWORD" "prod-parity-shared-pg"
ensure_file_secret "${SECRETS_ENV_FILE}" "POSTGRES_PASSWORD" "prod-parity-internal-pg"
ensure_file_secret "${SECRETS_ENV_FILE}" "STUHELPER_APP_DB_PASSWORD" "prod-parity-app"
ensure_file_secret "${SECRETS_ENV_FILE}" "STUHELPER_BACKUP_DB_PASSWORD" "prod-parity-backup"
ensure_file_secret "${SECRETS_ENV_FILE}" "STUHELPER_REPLICATION_DB_PASSWORD" "prod-parity-repl"
ensure_file_secret "${SECRETS_ENV_FILE}" "POSTGRES_EXPORTER_DB_PASSWORD" "prod-parity-pg-metrics"
ensure_file_secret "${SECRETS_ENV_FILE}" "OPENFGA_DB_PASSWORD" "prod-parity-openfga"
ensure_file_secret "${SECRETS_ENV_FILE}" "CASDOOR_DB_PASSWORD" "prod-parity-casdoor-db"
ensure_file_secret "${SECRETS_ENV_FILE}" "REDIS_PASSWORD" "prod-parity-redis"
ensure_file_secret "${SECRETS_ENV_FILE}" "REDIS_EXPORTER_PASSWORD" "prod-parity-redis-metrics"
ensure_file_secret "${SECRETS_ENV_FILE}" "REDIS_PROD_PARITY_MAINTENANCE_PASSWORD" "prod-parity-redis-maintenance"
ensure_file_secret "${SECRETS_ENV_FILE}" "METRICS_PASSWORD" "prod-parity-metrics"
ensure_file_secret "${SECRETS_ENV_FILE}" "ALERTMANAGER_WEBHOOK_TOKEN" "prod-parity-alertmanager-token-0123456789abcdef"
ensure_file_secret "${SECRETS_ENV_FILE}" "HMAC_SECRET" "prod-parity-hmac"
if ! grep -Eq '^DOC_AES_KEYS=' "${SECRETS_ENV_FILE}"; then
  upsert_env_file "${SECRETS_ENV_FILE}" "DOC_AES_ACTIVE_KEY_ID" "1"
  upsert_env_file "${SECRETS_ENV_FILE}" "DOC_AES_KEYS" "1:$(random_hex 32)"
fi
ensure_file_secret "${SECRETS_ENV_FILE}" "CASDOOR_CLIENT_SECRET" "prod-parity-casdoor-web"
ensure_file_secret "${SECRETS_ENV_FILE}" "CASDOOR_ADMIN_CLIENT_SECRET" "prod-parity-casdoor-admin"
ensure_file_secret "${SECRETS_ENV_FILE}" "CASDOOR_UNIAPP_CLIENT_SECRET" "prod-parity-casdoor-uniapp"
ensure_file_secret "${SECRETS_ENV_FILE}" "CASDOOR_APP_PROVISIONING_CLIENT_SECRET" "prod-parity-casdoor-app-provisioning"
ensure_file_secret "${SECRETS_ENV_FILE}" "CASDOOR_USER_PROFILE_CLIENT_SECRET" "prod-parity-casdoor-user-profile"
ensure_file_secret "${SECRETS_ENV_FILE}" "CASDOOR_INTROSPECTION_CLIENT_SECRET" "prod-parity-casdoor-introspection"
ensure_file_secret "${SECRETS_ENV_FILE}" "CASDOOR_USER_LOOKUP_CLIENT_SECRET" "prod-parity-casdoor-user-lookup"
ensure_file_secret "${SECRETS_ENV_FILE}" "CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_SECRET" "prod-parity-casdoor-token-probe-smoke"
ensure_file_secret "${SECRETS_ENV_FILE}" "SMS_SECRET_ID" "prod-parity-sms-secret-id"
ensure_file_secret "${SECRETS_ENV_FILE}" "SMS_SECRET_KEY" "prod-parity-sms-secret-key"
ensure_file_secret "${SECRETS_ENV_FILE}" "SMS_INTERNAL_KEY" "prod-parity-sms-internal"
ensure_file_secret "${SECRETS_ENV_FILE}" "OBJECT_STORAGE_SECRET_ACCESS_KEY" "prod-parity-object-storage-app"
ensure_file_secret "${SECRETS_ENV_FILE}" "BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY" "prod-parity-object-storage-backup"
ensure_file_secret "${SECRETS_ENV_FILE}" "GRAFANA_ADMIN_PASSWORD" "prod-parity-grafana"
ensure_file_secret "${SECRETS_ENV_FILE}" "BOT_SERVICE_TOKEN" "prod-parity-bot"

ensure_bootstrap_value "CASDOOR_BOOTSTRAP_APPLICATION" "app-built-in"
ensure_bootstrap_value "CASDOOR_BOOTSTRAP_CERTIFICATE" "cert-built-in"
ensure_bootstrap_value "CASDOOR_BOOTSTRAP_ORGANIZATION" "built-in"

load_env

app_password="${STUHELPER_APP_DB_PASSWORD}"
backup_password="${STUHELPER_BACKUP_DB_PASSWORD}"
replication_password="${STUHELPER_REPLICATION_DB_PASSWORD}"
ensure_file_value "${SECRETS_ENV_FILE}" "OPENFGA_DATASTORE_URI" "postgresql://openfga:${OPENFGA_DB_PASSWORD}@postgres:5432/openfga?sslmode=disable"
ensure_file_value "${ENV_FILE}" "DATABASE_URL" "postgres://stuhelper_app:${app_password}@postgres:5432/stuhelper?sslmode=disable"
ensure_file_value "${ENV_FILE}" "BACKUP_DATABASE_URL" "postgres://stuhelper_backup:${backup_password}@postgres:5432/stuhelper?sslmode=disable"
ensure_file_value "${ENV_FILE}" "REPLICATION_DATABASE_URL" "postgres://stuhelper_replication:${replication_password}@postgres:5432/stuhelper?sslmode=disable"
load_env

log "installing local prod-parity Nginx ingress"
"${SCRIPT_DIR}/install-local-prod-parity-ingress.sh"

log "starting local Baota-equivalent shared PostgreSQL"
(
  cd "${REPO_ROOT}" && \
  docker compose --env-file "${ENV_FILE}" -f "${REPO_ROOT}/docker-compose.prod-parity-postgres.yml" up -d --wait
)

"${SCRIPT_DIR}/init-shared-postgres.sh"

log "starting local production-parity Casdoor SSO"
compose --profile prod --profile local-sso up -d --wait casdoor
sync_casdoor_builtin_bootstrap_credentials

log "rendering local production-parity Redis and observability configs"
"${SCRIPT_DIR}/render-redis-tls.sh"
"${SCRIPT_DIR}/render-redis-acl.sh"
"${SCRIPT_DIR}/prepare-datastore-client-cas.sh"
"${SCRIPT_DIR}/render-local-object-storage-config.sh"
"${SCRIPT_DIR}/render-object-storage-tls.sh"
"${SCRIPT_DIR}/prepare-object-storage-client-ca.sh"
"${SCRIPT_DIR}/render-observability.sh" prod

log "building production images locally for ${tag}"
docker build \
  --build-arg VERSION="${tag}" \
  --build-arg GIT_COMMIT="${commit}" \
  --build-arg BUILD_TIME="${build_time}" \
  -t "${BACKEND_IMAGE_REF}" \
  "${REPO_ROOT}/server"
docker build \
  --build-arg VITE_API_URL="${WEB_VITE_API_URL}" \
  --build-arg VITE_SSO_URL="${WEB_VITE_SSO_URL}" \
  --build-arg VITE_WEB_URL="${WEB_VITE_WEB_URL}" \
  --build-arg VITE_ADMIN_URL="${ADMIN_PUBLIC_URL}" \
  --build-arg VITE_API_TIMEOUT_MS="${WEB_VITE_API_TIMEOUT_MS}" \
  --build-arg VITE_QQ_BOT_ENTRY="${WEB_VITE_QQ_BOT_ENTRY}" \
  --build-arg VITE_QQ_BIND_COMMAND="${WEB_VITE_QQ_BIND_COMMAND}" \
  -f "${REPO_ROOT}/clients/web/Dockerfile" \
  -t "${FRONTEND_IMAGE_REF}" \
  "${REPO_ROOT}"
docker build \
  --build-arg VITE_GLOB_API_URL="${ADMIN_VITE_API_URL}" \
  --build-arg VITE_BASE="${ADMIN_VITE_BASE}" \
  -f "${REPO_ROOT}/clients/admin/scripts/deploy/Dockerfile" \
  -t "${ADMIN_IMAGE_REF}" \
  "${REPO_ROOT}/clients"

infra_services=(
  redis
  object-storage
  docker-socket-proxy
  alloy
  alertmanager
  alert-webhook-sink
  loki
  tempo
  prometheus
  grafana
  node-exporter
  cadvisor
  postgres-exporter
  redis-exporter
  blackbox-exporter
)

log "starting local production-parity infrastructure services"
compose --profile prod up -d --no-deps --force-recreate --wait redis
compose --profile prod up -d --wait "${infra_services[@]}"

log "running local production-parity database migrations"
compose --profile prod up --no-deps migrate
compose --profile prod up --no-deps openfga-migrate

log "starting local production-parity OpenFGA"
compose --profile prod up -d --wait openfga

log "bootstrapping local OpenFGA store/model"
CASDOOR_BOOTSTRAP_ENABLED=true \
  OPENFGA_BOOTSTRAP_API_URL="http://127.0.0.1:8081" \
  OPENFGA_BOOTSTRAP_DATABASE_URL="postgres://stuhelper_app:${STUHELPER_APP_DB_PASSWORD}@127.0.0.1:${PROD_PARITY_POSTGRES_PORT:-15432}/stuhelper?sslmode=disable" \
  "${SCRIPT_DIR}/bootstrap-platform.sh" dev
load_env

log "importing and sealing the local production-parity authorization ledger"
CASDOOR_CUTOVER_ENDPOINT="http://sso.stuhelper.com" \
  OPENFGA_CUTOVER_API_URL="http://127.0.0.1:8081" \
  AUTHORIZATION_CUTOVER_DATABASE_URL="postgres://stuhelper_app:${STUHELPER_APP_DB_PASSWORD}@127.0.0.1:${PROD_PARITY_POSTGRES_PORT:-15432}/stuhelper?sslmode=disable" \
  "${SCRIPT_DIR}/authorization-ledger-cutover.sh" dev

log "starting local production-parity application services"
compose --profile prod up -d --wait app frontend admin

"${SCRIPT_DIR}/prod-parity-smoke.sh"

log "local production parity stack is ready"
echo "  Web:      http://stuhelper.com"
echo "  Admin:    http://stuhelper.com/admin/"
echo "  SSO:      https://sso.stuhelper.com"
echo "  Casdoor upstream(debug): http://sso.stuhelper.com"
echo "  Backend:  http://127.0.0.1:${BACKEND_EXTERNAL_PORT}"
echo "  Grafana:  http://127.0.0.1:${GRAFANA_PORT}"
echo "  Env dir:  ${PARITY_DIR}"
