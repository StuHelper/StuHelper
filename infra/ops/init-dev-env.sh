#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"
# shellcheck source=lib/retired-idp-env.sh
source "${SCRIPT_DIR}/lib/retired-idp-env.sh"

require_cmd python3

ensure_env_file
ensure_generated_files
remove_retired_idp_env_files "${ENV_FILE}" "${GENERATED_ENV_FILE}" "${GENERATED_SECRET_ENV_FILE}"

ensure_value() {
  local key="$1"
  local current="${2:-}"
  local desired="$3"
  if [[ -z "${current}" ]]; then
    upsert_env_file "${ENV_FILE}" "${key}" "${desired}"
  fi
}

ensure_dev_default() {
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
  [[ -z "${value}" || "${value}" == *"REPLACE_WITH_"* || "${value}" == "ChangeMeBeforeProduction" || "${value}" == "RUN_MAKE_DEV_INIT" ]]
}

load_env

if placeholder_or_empty "${POSTGRES_PASSWORD:-}" || [[ "${POSTGRES_PASSWORD:-}" == "dev123" ]]; then
  upsert_env_file "${ENV_FILE}" "POSTGRES_PASSWORD" "dev-postgres-$(random_hex 12)"
fi
if placeholder_or_empty "${REDIS_PASSWORD:-}" || [[ "${REDIS_PASSWORD:-}" == "dev123" ]]; then
  upsert_env_file "${ENV_FILE}" "REDIS_PASSWORD" "dev-redis-$(random_hex 12)"
fi
if placeholder_or_empty "${STUHELPER_APP_DB_PASSWORD:-}"; then
  upsert_env_file "${ENV_FILE}" "STUHELPER_APP_DB_PASSWORD" "dev-app-$(random_hex 12)"
fi
if placeholder_or_empty "${CASDOOR_DB_PASSWORD:-}"; then
  upsert_env_file "${ENV_FILE}" "CASDOOR_DB_PASSWORD" "dev-casdoor-$(random_hex 12)"
fi
if placeholder_or_empty "${OPENFGA_DB_PASSWORD:-}"; then
  upsert_env_file "${ENV_FILE}" "OPENFGA_DB_PASSWORD" "dev-openfga-$(random_hex 12)"
fi
if placeholder_or_empty "${STUHELPER_BACKUP_DB_PASSWORD:-}"; then
  upsert_env_file "${ENV_FILE}" "STUHELPER_BACKUP_DB_PASSWORD" "dev-backup-$(random_hex 12)"
fi
if placeholder_or_empty "${STUHELPER_REPLICATION_DB_PASSWORD:-}"; then
  upsert_env_file "${ENV_FILE}" "STUHELPER_REPLICATION_DB_PASSWORD" "dev-repl-$(random_hex 12)"
fi
if placeholder_or_empty "${HMAC_SECRET:-}"; then
  upsert_env_file "${ENV_FILE}" "HMAC_SECRET" "$(random_hex 32)"
fi
if placeholder_or_empty "${DOC_AES_KEYS:-}"; then
  upsert_env_file "${ENV_FILE}" "DOC_AES_ACTIVE_KEY_ID" "1"
  upsert_env_file "${ENV_FILE}" "DOC_AES_KEYS" "1:$(random_hex 32)"
fi
if [[ "${SMS_ENABLED:-false}" == "true" ]] && placeholder_or_empty "${SMS_INTERNAL_KEY:-}"; then
  upsert_env_file "${ENV_FILE}" "SMS_INTERNAL_KEY" "$(random_hex 16)"
fi
if placeholder_or_empty "${METRICS_PASSWORD:-}"; then
  upsert_env_file "${ENV_FILE}" "METRICS_PASSWORD" "dev-metrics-$(random_hex 8)"
fi
if placeholder_or_empty "${GRAFANA_ADMIN_PASSWORD:-}"; then
  upsert_env_file "${ENV_FILE}" "GRAFANA_ADMIN_PASSWORD" "dev-grafana-$(random_hex 8)"
fi
if placeholder_or_empty "${MINIO_ROOT_USER:-}"; then
  upsert_env_file "${ENV_FILE}" "MINIO_ROOT_USER" "dev-minio-root"
fi
if placeholder_or_empty "${MINIO_ROOT_PASSWORD:-}"; then
  upsert_env_file "${ENV_FILE}" "MINIO_ROOT_PASSWORD" "dev-minio-root-$(random_hex 12)"
fi
if placeholder_or_empty "${OBJECT_STORAGE_SECRET_ACCESS_KEY:-}"; then
  upsert_env_file "${ENV_FILE}" "OBJECT_STORAGE_SECRET_ACCESS_KEY" "dev-minio-$(random_hex 12)"
fi
if placeholder_or_empty "${CASDOOR_CLIENT_ID:-}"; then
  upsert_env_file "${ENV_FILE}" "CASDOOR_CLIENT_ID" "stuhelper-web"
fi
if placeholder_or_empty "${CASDOOR_CLIENT_SECRET:-}"; then
  upsert_env_file "${ENV_FILE}" "CASDOOR_CLIENT_SECRET" "dev-casdoor-client-$(random_hex 16)"
fi
if placeholder_or_empty "${CASDOOR_ADMIN_CLIENT_ID:-}"; then
  upsert_env_file "${ENV_FILE}" "CASDOOR_ADMIN_CLIENT_ID" "stuhelper-admin"
fi
if placeholder_or_empty "${CASDOOR_ADMIN_CLIENT_SECRET:-}"; then
  upsert_env_file "${ENV_FILE}" "CASDOOR_ADMIN_CLIENT_SECRET" "dev-casdoor-admin-$(random_hex 16)"
fi
if placeholder_or_empty "${CASDOOR_UNIAPP_CLIENT_ID:-}"; then
  upsert_env_file "${ENV_FILE}" "CASDOOR_UNIAPP_CLIENT_ID" "stuhelper-uniapp"
fi
if placeholder_or_empty "${CASDOOR_UNIAPP_CLIENT_SECRET:-}"; then
  upsert_env_file "${ENV_FILE}" "CASDOOR_UNIAPP_CLIENT_SECRET" "dev-casdoor-uniapp-$(random_hex 16)"
fi
if placeholder_or_empty "${CASDOOR_APP_PROVISIONING_CLIENT_ID:-}"; then
  upsert_env_file "${ENV_FILE}" "CASDOOR_APP_PROVISIONING_CLIENT_ID" "casdoor-admin-app-provisioning"
fi
if placeholder_or_empty "${CASDOOR_APP_PROVISIONING_CLIENT_SECRET:-}"; then
  upsert_env_file "${ENV_FILE}" "CASDOOR_APP_PROVISIONING_CLIENT_SECRET" "dev-casdoor-app-provisioning-$(random_hex 16)"
fi
if placeholder_or_empty "${CASDOOR_APP_PROVISIONING_APPLICATION:-}"; then
  upsert_env_file "${ENV_FILE}" "CASDOOR_APP_PROVISIONING_APPLICATION" "casdoor-admin-app-provisioning"
fi
if placeholder_or_empty "${CASDOOR_ROLE_SYNC_CLIENT_ID:-}"; then
  upsert_env_file "${ENV_FILE}" "CASDOOR_ROLE_SYNC_CLIENT_ID" "casdoor-admin-role-sync"
fi
if placeholder_or_empty "${CASDOOR_ROLE_SYNC_CLIENT_SECRET:-}"; then
  upsert_env_file "${ENV_FILE}" "CASDOOR_ROLE_SYNC_CLIENT_SECRET" "dev-casdoor-role-sync-$(random_hex 16)"
fi
if placeholder_or_empty "${CASDOOR_ROLE_SYNC_APPLICATION:-}"; then
  upsert_env_file "${ENV_FILE}" "CASDOOR_ROLE_SYNC_APPLICATION" "casdoor-admin-role-sync"
fi
if placeholder_or_empty "${CASDOOR_USER_LOOKUP_CLIENT_ID:-}"; then
  upsert_env_file "${ENV_FILE}" "CASDOOR_USER_LOOKUP_CLIENT_ID" "casdoor-admin-user-lookup"
fi
if placeholder_or_empty "${CASDOOR_USER_LOOKUP_CLIENT_SECRET:-}"; then
  upsert_env_file "${ENV_FILE}" "CASDOOR_USER_LOOKUP_CLIENT_SECRET" "dev-casdoor-user-lookup-$(random_hex 16)"
fi
if placeholder_or_empty "${CASDOOR_USER_LOOKUP_APPLICATION:-}"; then
  upsert_env_file "${ENV_FILE}" "CASDOOR_USER_LOOKUP_APPLICATION" "casdoor-admin-user-lookup"
fi

ensure_value "STACK_NAME" "${STACK_NAME:-}" "stuhelper-dev"
ensure_value "APP_ENV" "${APP_ENV:-}" "development"
load_env

ensure_value "DATABASE_URL" "${DATABASE_URL:-}" "postgres://stuhelper_app:${STUHELPER_APP_DB_PASSWORD:-}@localhost:5432/stuhelper?sslmode=disable"
ensure_value "POSTGRES_INTERNAL_SSL_MODE" "${POSTGRES_INTERNAL_SSL_MODE:-}" "disable"
ensure_dev_default "POSTGRES_PGDATA" "${POSTGRES_PGDATA:-}" "/var/lib/postgresql/data" "/var/lib/postgresql/18/docker"
ensure_dev_default "POSTGRES_ARCHIVE_MODE" "${POSTGRES_ARCHIVE_MODE:-}" "off" "on"
ensure_value "POSTGRES_ARCHIVE_TIMEOUT" "${POSTGRES_ARCHIVE_TIMEOUT:-}" "60s"
ensure_value "REDIS_HOST" "${REDIS_HOST:-}" "localhost"
ensure_value "REDIS_PORT" "${REDIS_PORT:-}" "6379"
ensure_value "REDIS_TLS_ENABLED" "${REDIS_TLS_ENABLED:-}" "true"
ensure_value "REDIS_TLS_CA" "${REDIS_TLS_CA:-}" "/tls/ca.crt"
ensure_value "CASDOOR_EXTERNALPORT" "${CASDOOR_EXTERNALPORT:-}" "8085"
ensure_value "CASDOOR_VERSION" "${CASDOOR_VERSION:-}" "3.31.1"
ensure_dev_default "TRAEFIK_HTTP_PORT" "${TRAEFIK_HTTP_PORT:-}" "8088" "8085"
ensure_dev_default "CASDOOR_ISSUER" "${CASDOOR_ISSUER:-}" "http://localhost:8085" "http://localhost" "http://host.docker.internal:8085"
ensure_value "CASDOOR_INTERNAL_ADDRESS" "${CASDOOR_INTERNAL_ADDRESS:-}" "casdoor:8000"
ensure_value "CASDOOR_REDIRECT_URI" "${CASDOOR_REDIRECT_URI:-}" "http://localhost:8080/api/v1/auth/callback"
ensure_value "CASDOOR_ORGANIZATION" "${CASDOOR_ORGANIZATION:-}" "stuhelper"
ensure_value "CASDOOR_ROLES_CLAIM" "${CASDOOR_ROLES_CLAIM:-}" "roles"
ensure_value "CASDOOR_BOOTSTRAP_ENABLED" "${CASDOOR_BOOTSTRAP_ENABLED:-}" "false"
ensure_value "CASDOOR_BOOTSTRAP_ENV_FILE" "${CASDOOR_BOOTSTRAP_ENV_FILE:-}" ".env.casdoor-bootstrap.local"
ensure_value "CASDOOR_ADMIN_REDIRECT_URI" "${CASDOOR_ADMIN_REDIRECT_URI:-}" "${CASDOOR_REDIRECT_URI:-http://localhost:8080/api/v1/auth/callback}"
ensure_value "CASDOOR_UNIAPP_REDIRECT_URI" "${CASDOOR_UNIAPP_REDIRECT_URI:-}" "${CASDOOR_REDIRECT_URI:-http://localhost:8080/api/v1/auth/callback}"
ensure_value "CASDOOR_SMS_PROVIDER_ENABLED" "${CASDOOR_SMS_PROVIDER_ENABLED:-}" "false"
ensure_value "CASDOOR_SMS_PROVIDER_NAME" "${CASDOOR_SMS_PROVIDER_NAME:-}" "stuhelper-sms"
ensure_value "CASDOOR_SMS_PROVIDER_DISPLAY_NAME" "${CASDOOR_SMS_PROVIDER_DISPLAY_NAME:-}" "StuHelper-SMS"
ensure_value "CASDOOR_SMS_PROVIDER_CATEGORY" "${CASDOOR_SMS_PROVIDER_CATEGORY:-}" "SMS"
ensure_value "CASDOOR_SMS_PROVIDER_TYPE" "${CASDOOR_SMS_PROVIDER_TYPE:-}" "CustomHTTP"
ensure_value "CASDOOR_SMS_PROVIDER_METHOD" "${CASDOOR_SMS_PROVIDER_METHOD:-}" "POST"
ensure_value "CASDOOR_SMS_PROVIDER_TITLE" "${CASDOOR_SMS_PROVIDER_TITLE:-}" "content"
ensure_value "CASDOOR_SMS_PROVIDER_ENDPOINT" "${CASDOOR_SMS_PROVIDER_ENDPOINT:-}" "http://host.docker.internal:8080/internal/sms/send"
ensure_value "CASDOOR_EMAIL_PROVIDER_ENABLED" "${CASDOOR_EMAIL_PROVIDER_ENABLED:-}" "false"
ensure_value "WEB_PUBLIC_URL" "${WEB_PUBLIC_URL:-}" "http://localhost:3000"
ensure_value "ADMIN_PUBLIC_URL" "${ADMIN_PUBLIC_URL:-}" "http://localhost:${ADMIN_EXTERNAL_PORT:-3001}"
ensure_value "WEB_VITE_API_URL" "${WEB_VITE_API_URL:-}" ""
ensure_dev_default "WEB_VITE_SSO_URL" "${WEB_VITE_SSO_URL:-}" "http://localhost:8085" "http://host.docker.internal:8085"
ensure_value "WEB_VITE_API_TIMEOUT_MS" "${WEB_VITE_API_TIMEOUT_MS:-}" "15000"
ensure_value "ADMIN_VITE_API_URL" "${ADMIN_VITE_API_URL:-}" "/api/v1"
ensure_value "ADMIN_VITE_BASE" "${ADMIN_VITE_BASE:-}" "/admin/"
ensure_value "OPENFGA_API_URL" "${OPENFGA_API_URL:-}" "http://localhost:8081"
ensure_value "OBJECT_STORAGE_ENDPOINT" "${OBJECT_STORAGE_ENDPOINT:-}" "http://localhost:9000"
ensure_value "OBJECT_STORAGE_REGION" "${OBJECT_STORAGE_REGION:-}" "us-east-1"
ensure_value "OBJECT_STORAGE_BUCKET" "${OBJECT_STORAGE_BUCKET:-}" "stuhelper-identity"
ensure_value "OBJECT_STORAGE_ACCESS_KEY_ID" "${OBJECT_STORAGE_ACCESS_KEY_ID:-}" "stuhelper"
ensure_value "OBJECT_STORAGE_USE_SSL" "${OBJECT_STORAGE_USE_SSL:-}" "false"
ensure_value "OBJECT_STORAGE_FORCE_PATH_STYLE" "${OBJECT_STORAGE_FORCE_PATH_STYLE:-}" "true"
ensure_value "OBJECT_STORAGE_PRESIGN_TTL" "${OBJECT_STORAGE_PRESIGN_TTL:-}" "600"
ensure_value "PROMETHEUS_RETENTION_TIME" "${PROMETHEUS_RETENTION_TIME:-}" "15d"
ensure_value "PROMETHEUS_RETENTION_SIZE" "${PROMETHEUS_RETENTION_SIZE:-}" "20GB"
ensure_value "BACKUP_LOGICAL_RETENTION_DAYS" "${BACKUP_LOGICAL_RETENTION_DAYS:-}" "14"
ensure_value "BACKUP_BASE_RETENTION_DAYS" "${BACKUP_BASE_RETENTION_DAYS:-}" "30"
ensure_value "WAL_ARCHIVE_RETENTION_DAYS" "${WAL_ARCHIVE_RETENTION_DAYS:-}" "14"
ensure_value "BACKEND_IMAGE_REF" "${BACKEND_IMAGE_REF:-}" "stuhelper/backend:dev-placeholder"
ensure_value "FRONTEND_IMAGE_REF" "${FRONTEND_IMAGE_REF:-}" "stuhelper/frontend:dev-placeholder"
ensure_value "ADMIN_IMAGE_REF" "${ADMIN_IMAGE_REF:-}" "stuhelper/admin:dev-placeholder"

"${SCRIPT_DIR}/render-redis-tls.sh"
"${SCRIPT_DIR}/render-redis-acl.sh"
log "development environment file is ready: ${ENV_FILE}"
