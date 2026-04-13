#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

require_cmd python3

if [[ -z "${ENV_TEMPLATE_FILE:-}" || "${ENV_TEMPLATE_FILE}" == "${REPO_ROOT}/.env.example" ]]; then
  export ENV_TEMPLATE_FILE="${REPO_ROOT}/.env.prod.example"
fi

if [[ "${ENV_FILE:-}" == "${REPO_ROOT}/.env" || "${ENV_FILE:-.env}" == ".env" ]]; then
  ENV_FILE="${REPO_ROOT}/.env.prod.shared"
fi
if [[ -z "${GENERATED_SECRET_ENV_FILE:-}" || "${GENERATED_SECRET_ENV_FILE:-}" == "${REPO_ROOT}/.env.generated.secrets" ]]; then
  GENERATED_SECRET_ENV_FILE="${REPO_ROOT}/.env.prod.generated.secrets"
fi
if [[ -z "${SECRETS_ENV_FILE:-}" ]]; then
  SECRETS_ENV_FILE="${REPO_ROOT}/.env.prod.secrets.local"
fi
if [[ "${GENERATED_ENV_FILE:-}" == "${REPO_ROOT}/.env.generated" || "${GENERATED_ENV_FILE:-.env.generated}" == ".env.generated" ]]; then
  GENERATED_ENV_FILE="${REPO_ROOT}/.env.prod.generated"
fi

ensure_env_file
ensure_secrets_env_file
ensure_generated_files

ensure_value() {
  local key="$1"
  local current="${2:-}"
  local desired="$3"
  if [[ -z "${current}" ]]; then
    upsert_env_file "${ENV_FILE}" "${key}" "${desired}"
  fi
}

ensure_secret_value() {
  local key="$1"
  local current="${2:-}"
  local desired="$3"
  if [[ -z "${current}" ]]; then
    upsert_env_file "${SECRETS_ENV_FILE}" "${key}" "${desired}"
  fi
}

ensure_prod_default() {
  local key="$1"
  local current="${2:-}"
  local desired="$3"
  shift 3 || true

  if [[ -z "${current}" ]]; then
    upsert_env_file "${ENV_FILE}" "${key}" "${desired}"
    return
  fi

  local legacy
  for legacy in "$@"; do
    if [[ "${current}" == "${legacy}" ]]; then
      upsert_env_file "${ENV_FILE}" "${key}" "${desired}"
      return
    fi
  done
}

placeholder_or_empty() {
  local value="${1:-}"
  [[ -z "${value}" || "${value}" == *"REPLACE_WITH_"* || "${value}" == "ChangeMeBeforeProduction" ]]
}

future_iso_timestamp() {
  python3 - <<'PY'
from datetime import datetime, timedelta, timezone
print((datetime.now(timezone.utc) + timedelta(days=180)).replace(microsecond=0).isoformat().replace("+00:00", "Z"))
PY
}

load_env

if placeholder_or_empty "${POSTGRES_PASSWORD:-}" || [[ "${POSTGRES_PASSWORD:-}" == "dev123" ]]; then
  upsert_env_file "${SECRETS_ENV_FILE}" "POSTGRES_PASSWORD" "prod-pg-$(random_hex 16)"
fi
if placeholder_or_empty "${REDIS_PASSWORD:-}" || [[ "${REDIS_PASSWORD:-}" == "dev123" ]]; then
  upsert_env_file "${SECRETS_ENV_FILE}" "REDIS_PASSWORD" "prod-redis-$(random_hex 16)"
fi
if placeholder_or_empty "${STUHELPER_APP_DB_PASSWORD:-}"; then
  upsert_env_file "${SECRETS_ENV_FILE}" "STUHELPER_APP_DB_PASSWORD" "prod-app-$(random_hex 16)"
fi
if placeholder_or_empty "${ZITADEL_DB_PASSWORD:-}"; then
  upsert_env_file "${SECRETS_ENV_FILE}" "ZITADEL_DB_PASSWORD" "prod-zitadel-$(random_hex 16)"
fi
if placeholder_or_empty "${OPENFGA_DB_PASSWORD:-}"; then
  upsert_env_file "${SECRETS_ENV_FILE}" "OPENFGA_DB_PASSWORD" "prod-openfga-$(random_hex 16)"
fi
if placeholder_or_empty "${STUHELPER_BACKUP_DB_PASSWORD:-}"; then
  upsert_env_file "${SECRETS_ENV_FILE}" "STUHELPER_BACKUP_DB_PASSWORD" "prod-backup-$(random_hex 16)"
fi
if placeholder_or_empty "${STUHELPER_REPLICATION_DB_PASSWORD:-}"; then
  upsert_env_file "${SECRETS_ENV_FILE}" "STUHELPER_REPLICATION_DB_PASSWORD" "prod-repl-$(random_hex 16)"
fi
if placeholder_or_empty "${HMAC_SECRET:-}" || [[ "${HMAC_SECRET:-}" == "dev_hmac_secret_change_in_production_32ch" ]]; then
  upsert_env_file "${SECRETS_ENV_FILE}" "HMAC_SECRET" "$(random_hex 32)"
fi
if placeholder_or_empty "${DOC_AES_KEYS:-}"; then
  upsert_env_file "${SECRETS_ENV_FILE}" "DOC_AES_ACTIVE_KEY_ID" "1"
  upsert_env_file "${SECRETS_ENV_FILE}" "DOC_AES_KEYS" "1:$(random_hex 32)"
fi
if placeholder_or_empty "${SMS_INTERNAL_KEY:-}"; then
  upsert_env_file "${SECRETS_ENV_FILE}" "SMS_INTERNAL_KEY" "$(random_hex 16)"
fi
if placeholder_or_empty "${METRICS_PASSWORD:-}"; then
  upsert_env_file "${SECRETS_ENV_FILE}" "METRICS_PASSWORD" "prod-metrics-$(random_hex 12)"
fi
if placeholder_or_empty "${GRAFANA_ADMIN_PASSWORD:-}"; then
  upsert_env_file "${SECRETS_ENV_FILE}" "GRAFANA_ADMIN_PASSWORD" "prod-grafana-$(random_hex 12)"
fi
if placeholder_or_empty "${MINIO_ROOT_USER:-}"; then
  upsert_env_file "${SECRETS_ENV_FILE}" "MINIO_ROOT_USER" "minio-root-$(random_hex 8)"
fi
if placeholder_or_empty "${MINIO_ROOT_PASSWORD:-}"; then
  upsert_env_file "${SECRETS_ENV_FILE}" "MINIO_ROOT_PASSWORD" "prod-minio-root-$(random_hex 24)"
fi
if placeholder_or_empty "${OBJECT_STORAGE_SECRET_ACCESS_KEY:-}"; then
  upsert_env_file "${SECRETS_ENV_FILE}" "OBJECT_STORAGE_SECRET_ACCESS_KEY" "prod-minio-app-$(random_hex 16)"
fi
if placeholder_or_empty "${BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY:-}"; then
  upsert_env_file "${SECRETS_ENV_FILE}" "BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY" "prod-minio-backup-$(random_hex 16)"
fi
if [[ -z "${ZITADEL_MASTERKEY:-}" || "${ZITADEL_MASTERKEY:-}" == "StuHelperDevMasterKey123456789AB" ]]; then
  upsert_env_file "${SECRETS_ENV_FILE}" "ZITADEL_MASTERKEY" "$(random_hex 16)"
fi
if [[ -z "${ZITADEL_ADMIN_PASSWORD:-}" || "${ZITADEL_ADMIN_PASSWORD:-}" == "Admin1234!" ]]; then
  upsert_env_file "${SECRETS_ENV_FILE}" "ZITADEL_ADMIN_PASSWORD" "ProdAdmin!$(random_hex 6)"
fi
if placeholder_or_empty "${LOGIN_CLIENT_PAT_EXPIRATION:-}" || [[ "${LOGIN_CLIENT_PAT_EXPIRATION:-}" == "2040-01-01T00:00:00Z" ]]; then
  upsert_env_file "${ENV_FILE}" "LOGIN_CLIENT_PAT_EXPIRATION" "$(future_iso_timestamp)"
fi

load_env

ensure_prod_default "STACK_NAME" "${STACK_NAME:-}" "stuhelper-prod" "stuhelper-dev" "stuhelper"
ensure_prod_default "APP_ENV" "${APP_ENV:-}" "production" "development"
ensure_value "LOG_LEVEL" "${LOG_LEVEL:-}" "info"
ensure_prod_default "LOG_FORMAT" "${LOG_FORMAT:-}" "json" "console"
ensure_value "LOG_OUTPUT" "${LOG_OUTPUT:-}" "stdout"
ensure_prod_default "DATABASE_URL" "${DATABASE_URL:-}" "postgres://stuhelper_app:REPLACE_WITH_STUHELPER_APP_DB_PASSWORD@postgres:5432/stuhelper?sslmode=verify-full&sslrootcert=/tls/ca.crt" "postgres://stuhelper:dev123@localhost:5432/stuhelper?sslmode=disable" "postgres://stuhelper_app:REPLACE_WITH_STUHELPER_APP_DB_PASSWORD@localhost:5432/stuhelper?sslmode=disable"
ensure_prod_default "BACKUP_DATABASE_URL" "${BACKUP_DATABASE_URL:-}" "postgres://stuhelper_backup:REPLACE_WITH_STUHELPER_BACKUP_DB_PASSWORD@postgres:5432/stuhelper?sslmode=verify-full&sslrootcert=/tls/ca.crt"
ensure_prod_default "REPLICATION_DATABASE_URL" "${REPLICATION_DATABASE_URL:-}" "postgres://stuhelper_replication:REPLACE_WITH_STUHELPER_REPLICATION_DB_PASSWORD@postgres:5432/stuhelper?sslmode=verify-full&sslrootcert=/tls/ca.crt"
ensure_prod_default "DB_SSL_MODE" "${DB_SSL_MODE:-}" "verify-full" "require" "disable"
ensure_prod_default "DB_SSL_ROOT_CERT" "${DB_SSL_ROOT_CERT:-}" "/tls/ca.crt"
ensure_prod_default "POSTGRES_ENABLE_SSL" "${POSTGRES_ENABLE_SSL:-}" "on" "off"
ensure_prod_default "POSTGRES_INTERNAL_SSL_MODE" "${POSTGRES_INTERNAL_SSL_MODE:-}" "verify-full" "require" "disable"
ensure_prod_default "POSTGRES_PGDATA" "${POSTGRES_PGDATA:-}" "/var/lib/postgresql/data" "/var/lib/postgresql/18/docker"
ensure_prod_default "POSTGRES_ARCHIVE_MODE" "${POSTGRES_ARCHIVE_MODE:-}" "on" "off"
ensure_value "POSTGRES_ARCHIVE_TIMEOUT" "${POSTGRES_ARCHIVE_TIMEOUT:-}" "60s"
ensure_prod_default "REDIS_HOST" "${REDIS_HOST:-}" "redis" "localhost"
ensure_value "REDIS_PORT" "${REDIS_PORT:-}" "6379"
ensure_prod_default "REDIS_TLS_ENABLED" "${REDIS_TLS_ENABLED:-}" "true" "false"
ensure_prod_default "REDIS_TLS_CA" "${REDIS_TLS_CA:-}" "/tls/ca.crt"
ensure_prod_default "CORS_ORIGINS" "${CORS_ORIGINS:-}" "REPLACE_WITH_PRODUCTION_CORS_ORIGINS" "http://localhost:3000,http://localhost:3001"
ensure_value "TRUSTED_PROXIES" "${TRUSTED_PROXIES:-}" "127.0.0.1/32,172.16.0.0/12,192.168.0.0/16"
ensure_prod_default "OTEL_ENABLED" "${OTEL_ENABLED:-}" "true" "false"
ensure_value "OTEL_SERVICE_NAME" "${OTEL_SERVICE_NAME:-}" "stuhelper-backend"
ensure_value "OTEL_SERVICE_NAMESPACE" "${OTEL_SERVICE_NAMESPACE:-}" "stuhelper"
ensure_prod_default "OTEL_EXPORTER_OTLP_ENDPOINT" "${OTEL_EXPORTER_OTLP_ENDPOINT:-}" "http://alloy:4318" "http://localhost:4318"
ensure_value "OTEL_EXPORTER_OTLP_INSECURE" "${OTEL_EXPORTER_OTLP_INSECURE:-}" "true"
ensure_prod_default "TOKEN_COOKIE_SECURE" "${TOKEN_COOKIE_SECURE:-}" "true" "false"
ensure_value "ZITADEL_EXTERNALPORT" "${ZITADEL_EXTERNALPORT:-}" "8085"
ensure_prod_default "ZITADEL_DOMAIN" "${ZITADEL_DOMAIN:-}" "REPLACE_WITH_ZITADEL_DOMAIN" "localhost"
ensure_prod_default "ZITADEL_PUBLIC_SCHEME" "${ZITADEL_PUBLIC_SCHEME:-}" "https" "http"
ensure_prod_default "ZITADEL_EXTERNALSECURE" "${ZITADEL_EXTERNALSECURE:-}" "true" "false"
ensure_prod_default "ZITADEL_ISSUER" "${ZITADEL_ISSUER:-}" "REPLACE_WITH_ZITADEL_ISSUER" "http://localhost:8085"
ensure_prod_default "ZITADEL_INTERNAL_ADDRESS" "${ZITADEL_INTERNAL_ADDRESS:-}" "" "host.docker.internal:8085"
ensure_prod_default "ZITADEL_REDIRECT_URI" "${ZITADEL_REDIRECT_URI:-}" "REPLACE_WITH_ZITADEL_REDIRECT_URI" "http://localhost:8080/api/v1/auth/callback"
ensure_prod_default "WEB_PUBLIC_URL" "${WEB_PUBLIC_URL:-}" "REPLACE_WITH_WEB_PUBLIC_URL" "http://localhost:3000"
ensure_prod_default "ADMIN_PUBLIC_URL" "${ADMIN_PUBLIC_URL:-}" "REPLACE_WITH_ADMIN_PUBLIC_URL" "http://localhost:3001"
ensure_prod_default "WEB_VITE_API_URL" "${WEB_VITE_API_URL:-}" "/api" ""
ensure_prod_default "WEB_VITE_SSO_URL" "${WEB_VITE_SSO_URL:-}" "REPLACE_WITH_WEB_VITE_SSO_URL" "http://localhost:8085"
ensure_value "WEB_VITE_API_TIMEOUT_MS" "${WEB_VITE_API_TIMEOUT_MS:-}" "15000"
ensure_value "ADMIN_VITE_API_URL" "${ADMIN_VITE_API_URL:-}" "/api/v1"
ensure_value "ADMIN_VITE_BASE" "${ADMIN_VITE_BASE:-}" "/admin/"
ensure_prod_default "OPENFGA_API_URL" "${OPENFGA_API_URL:-}" "http://openfga:8081" "http://localhost:8081"
ensure_prod_default "OBJECT_STORAGE_ENDPOINT" "${OBJECT_STORAGE_ENDPOINT:-}" "REPLACE_WITH_OBJECT_STORAGE_ENDPOINT" "http://localhost:9000" "http://minio:9000"
ensure_value "OBJECT_STORAGE_REGION" "${OBJECT_STORAGE_REGION:-}" "us-east-1"
ensure_value "OBJECT_STORAGE_BUCKET" "${OBJECT_STORAGE_BUCKET:-}" "stuhelper-identity"
ensure_prod_default "MINIO_ROOT_USER" "${MINIO_ROOT_USER:-}" "REPLACE_WITH_MINIO_ROOT_USER" "stuhelper"
ensure_prod_default "OBJECT_STORAGE_ACCESS_KEY_ID" "${OBJECT_STORAGE_ACCESS_KEY_ID:-}" "REPLACE_WITH_OBJECT_STORAGE_ACCESS_KEY_ID" "stuhelper"
ensure_prod_default "BACKUP_OBJECT_STORAGE_ENDPOINT" "${BACKUP_OBJECT_STORAGE_ENDPOINT:-}" "REPLACE_WITH_BACKUP_OBJECT_STORAGE_ENDPOINT" "http://localhost:9000" "http://minio:9000"
ensure_value "BACKUP_OBJECT_STORAGE_BUCKET" "${BACKUP_OBJECT_STORAGE_BUCKET:-}" "stuhelper-postgres-backup"
ensure_value "BACKUP_OBJECT_STORAGE_PREFIX" "${BACKUP_OBJECT_STORAGE_PREFIX:-}" "postgres"
ensure_prod_default "BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID" "${BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID:-}" "REPLACE_WITH_BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID" "stuhelper-backup"
ensure_prod_default "BACKUP_OBJECT_STORAGE_TLS_INSECURE" "${BACKUP_OBJECT_STORAGE_TLS_INSECURE:-}" "false" "true"
ensure_prod_default "OBJECT_STORAGE_USE_SSL" "${OBJECT_STORAGE_USE_SSL:-}" "true" "false"
ensure_prod_default "OBJECT_STORAGE_FORCE_PATH_STYLE" "${OBJECT_STORAGE_FORCE_PATH_STYLE:-}" "false" "true"
ensure_value "OBJECT_STORAGE_PRESIGN_TTL" "${OBJECT_STORAGE_PRESIGN_TTL:-}" "600"
ensure_value "PROMETHEUS_RETENTION_TIME" "${PROMETHEUS_RETENTION_TIME:-}" "15d"
ensure_value "PROMETHEUS_RETENTION_SIZE" "${PROMETHEUS_RETENTION_SIZE:-}" "20GB"
ensure_value "BACKUP_LOGICAL_RETENTION_DAYS" "${BACKUP_LOGICAL_RETENTION_DAYS:-}" "7"
ensure_value "BACKUP_BASE_RETENTION_DAYS" "${BACKUP_BASE_RETENTION_DAYS:-}" "14"
ensure_value "WAL_ARCHIVE_RETENTION_DAYS" "${WAL_ARCHIVE_RETENTION_DAYS:-}" "7"
ensure_prod_default "GRAFANA_ROOT_URL" "${GRAFANA_ROOT_URL:-}" "REPLACE_WITH_GRAFANA_ROOT_URL" "http://localhost:3003"
ensure_prod_default "ALLOW_LOCAL_ALERT_SINK" "${ALLOW_LOCAL_ALERT_SINK:-}" "false" "true"
ensure_prod_default "ALERTMANAGER_WEBHOOK_URL" "${ALERTMANAGER_WEBHOOK_URL:-}" "REPLACE_WITH_ALERTMANAGER_WEBHOOK_URL" "http://alert-webhook-sink:8080/alerts"
ensure_prod_default "TAG" "${TAG:-}" "" "latest"
ensure_prod_default "BACKEND_IMAGE_REF" "${BACKEND_IMAGE_REF:-}" "REPLACE_WITH_BACKEND_IMAGE_REF" "registry.stuhelper.com/stuhelper/backend:latest" "stuhelper/backend:dev-placeholder"
ensure_prod_default "FRONTEND_IMAGE_REF" "${FRONTEND_IMAGE_REF:-}" "REPLACE_WITH_FRONTEND_IMAGE_REF" "registry.stuhelper.com/stuhelper/frontend:latest" "stuhelper/frontend:dev-placeholder"
ensure_prod_default "ADMIN_IMAGE_REF" "${ADMIN_IMAGE_REF:-}" "REPLACE_WITH_ADMIN_IMAGE_REF" "registry.stuhelper.com/stuhelper/admin:latest" "stuhelper/admin:dev-placeholder"

"${SCRIPT_DIR}/render-postgres-tls.sh"
"${SCRIPT_DIR}/render-redis-tls.sh"
"${SCRIPT_DIR}/render-redis-acl.sh"
"${SCRIPT_DIR}/render-zitadel-secrets.sh"

log "production environment file is ready: ${ENV_FILE}"
log "generated runtime file path: ${GENERATED_ENV_FILE}"
