#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

require_cmd python3

ensure_env_file
ensure_generated_files

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
  [[ -z "${value}" || "${value}" == *"REPLACE_WITH_"* || "${value}" == "ChangeMeBeforeProduction" ]]
}

future_iso_timestamp() {
  python3 - <<'PY'
from datetime import datetime, timedelta, timezone
print((datetime.now(timezone.utc) + timedelta(days=180)).replace(microsecond=0).isoformat().replace("+00:00", "Z"))
PY
}

load_env

if placeholder_or_empty "${POSTGRES_PASSWORD:-}"; then
  upsert_env_file "${ENV_FILE}" "POSTGRES_PASSWORD" "dev123"
fi
if placeholder_or_empty "${REDIS_PASSWORD:-}"; then
  upsert_env_file "${ENV_FILE}" "REDIS_PASSWORD" "dev123"
fi
if placeholder_or_empty "${STUHELPER_APP_DB_PASSWORD:-}"; then
  upsert_env_file "${ENV_FILE}" "STUHELPER_APP_DB_PASSWORD" "dev-app-$(random_hex 12)"
fi
if placeholder_or_empty "${ZITADEL_DB_PASSWORD:-}"; then
  upsert_env_file "${ENV_FILE}" "ZITADEL_DB_PASSWORD" "dev-zitadel-$(random_hex 12)"
fi
if placeholder_or_empty "${OPENFGA_DB_PASSWORD:-}"; then
  upsert_env_file "${ENV_FILE}" "OPENFGA_DB_PASSWORD" "dev-openfga-$(random_hex 12)"
fi
if placeholder_or_empty "${HMAC_SECRET:-}"; then
  upsert_env_file "${ENV_FILE}" "HMAC_SECRET" "$(random_hex 32)"
fi
if placeholder_or_empty "${DOC_AES_KEYS:-}"; then
  upsert_env_file "${ENV_FILE}" "DOC_AES_ACTIVE_KEY_ID" "1"
  upsert_env_file "${ENV_FILE}" "DOC_AES_KEYS" "1:$(random_hex 32)"
fi
if placeholder_or_empty "${SMS_INTERNAL_KEY:-}"; then
  upsert_env_file "${ENV_FILE}" "SMS_INTERNAL_KEY" "$(random_hex 16)"
fi
if placeholder_or_empty "${METRICS_PASSWORD:-}"; then
  upsert_env_file "${ENV_FILE}" "METRICS_PASSWORD" "dev-metrics-$(random_hex 8)"
fi
if placeholder_or_empty "${GRAFANA_ADMIN_PASSWORD:-}"; then
  upsert_env_file "${ENV_FILE}" "GRAFANA_ADMIN_PASSWORD" "dev-grafana-$(random_hex 8)"
fi
if placeholder_or_empty "${OBJECT_STORAGE_SECRET_ACCESS_KEY:-}"; then
  upsert_env_file "${ENV_FILE}" "OBJECT_STORAGE_SECRET_ACCESS_KEY" "dev-minio-$(random_hex 12)"
fi
if placeholder_or_empty "${ZITADEL_MASTERKEY:-}" || [[ "${ZITADEL_MASTERKEY:-}" == "StuHelperDevMasterKey123456789AB" ]]; then
  upsert_env_file "${ENV_FILE}" "ZITADEL_MASTERKEY" "$(random_hex 16)"
fi
if placeholder_or_empty "${ZITADEL_ADMIN_PASSWORD:-}" || [[ "${ZITADEL_ADMIN_PASSWORD:-}" == "Admin1234!" ]]; then
  upsert_env_file "${ENV_FILE}" "ZITADEL_ADMIN_PASSWORD" "DevAdmin!$(random_hex 6)"
fi
if placeholder_or_empty "${LOGIN_CLIENT_PAT_EXPIRATION:-}" || [[ "${LOGIN_CLIENT_PAT_EXPIRATION:-}" == "2040-01-01T00:00:00Z" ]]; then
  upsert_env_file "${ENV_FILE}" "LOGIN_CLIENT_PAT_EXPIRATION" "$(future_iso_timestamp)"
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
ensure_value "ZITADEL_EXTERNALPORT" "${ZITADEL_EXTERNALPORT:-}" "8085"
ensure_dev_default "ZITADEL_DOMAIN" "${ZITADEL_DOMAIN:-}" "localhost" "host.docker.internal"
ensure_value "ZITADEL_PUBLIC_SCHEME" "${ZITADEL_PUBLIC_SCHEME:-}" "http"
ensure_dev_default "ZITADEL_ISSUER" "${ZITADEL_ISSUER:-}" "http://localhost:8085" "http://host.docker.internal:8085"
ensure_value "ZITADEL_INTERNAL_ADDRESS" "${ZITADEL_INTERNAL_ADDRESS:-}" "host.docker.internal:8085"
ensure_value "ZITADEL_REDIRECT_URI" "${ZITADEL_REDIRECT_URI:-}" "http://localhost:8080/api/v1/auth/callback"
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
ensure_value "BACKUP_LOGICAL_RETENTION_DAYS" "${BACKUP_LOGICAL_RETENTION_DAYS:-}" "7"
ensure_value "BACKUP_BASE_RETENTION_DAYS" "${BACKUP_BASE_RETENTION_DAYS:-}" "14"
ensure_value "WAL_ARCHIVE_RETENTION_DAYS" "${WAL_ARCHIVE_RETENTION_DAYS:-}" "7"

"${SCRIPT_DIR}/render-zitadel-secrets.sh"

log "development environment file is ready: ${ENV_FILE}"
