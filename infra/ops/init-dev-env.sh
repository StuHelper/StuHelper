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
remove_env_key_prefixes_from_file "${ENV_FILE}" "TRAEFIK_"
remove_env_key_prefixes_from_file "${ENV_FILE}" "SSO_PUBLIC_SMOKE_"
remove_env_key_prefixes_from_file "${ENV_FILE}" "SSO_PUBLIC_BASE_URL"

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
  return 0
}

ensure_dev_pattern_default() {
  local key="$1"
  local current="${2:-}"
  local desired="$3"
  shift 3 || true

  if [[ -z "${current}" ]]; then
    upsert_env_file "${ENV_FILE}" "${key}" "${desired}"
    return
  fi

  local legacy_pattern
  for legacy_pattern in "$@"; do
    case "${current}" in
      ${legacy_pattern})
        upsert_env_file "${ENV_FILE}" "${key}" "${desired}"
        return
        ;;
    esac
  done
  return 0
}

replace_legacy_env_value() {
  local key="$1"
  local current="${2:-}"
  local desired="$3"
  shift 3 || true

  [[ -n "${current}" ]] || return 0

  local legacy
  for legacy in "$@"; do
    if [[ "${current}" == "${legacy}" ]]; then
      upsert_env_file "${ENV_FILE}" "${key}" "${desired}"
      return
    fi
  done
  return 0
}

replace_legacy_env_pattern() {
  local key="$1"
  local current="${2:-}"
  local desired="$3"
  shift 3 || true

  [[ -n "${current}" ]] || return 0

  local legacy_pattern
  for legacy_pattern in "$@"; do
    case "${current}" in
      ${legacy_pattern})
        upsert_env_file "${ENV_FILE}" "${key}" "${desired}"
        return
        ;;
    esac
  done
  return 0
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
if placeholder_or_empty "${CASDOOR_USER_PROFILE_CLIENT_ID:-}"; then
  upsert_env_file "${ENV_FILE}" "CASDOOR_USER_PROFILE_CLIENT_ID" "casdoor-admin-user-profile"
fi
if placeholder_or_empty "${CASDOOR_USER_PROFILE_CLIENT_SECRET:-}"; then
  upsert_env_file "${ENV_FILE}" "CASDOOR_USER_PROFILE_CLIENT_SECRET" "dev-casdoor-user-profile-$(random_hex 16)"
fi
if placeholder_or_empty "${CASDOOR_USER_PROFILE_APPLICATION:-}"; then
  upsert_env_file "${ENV_FILE}" "CASDOOR_USER_PROFILE_APPLICATION" "casdoor-admin-user-profile"
fi
if placeholder_or_empty "${CASDOOR_INTROSPECTION_CLIENT_ID:-}"; then
  upsert_env_file "${ENV_FILE}" "CASDOOR_INTROSPECTION_CLIENT_ID" "casdoor-token-introspection"
fi
if placeholder_or_empty "${CASDOOR_INTROSPECTION_CLIENT_SECRET:-}"; then
  upsert_env_file "${ENV_FILE}" "CASDOOR_INTROSPECTION_CLIENT_SECRET" "dev-casdoor-introspection-$(random_hex 16)"
fi
if placeholder_or_empty "${CASDOOR_INTROSPECTION_APPLICATION:-}"; then
  upsert_env_file "${ENV_FILE}" "CASDOOR_INTROSPECTION_APPLICATION" "casdoor-token-introspection"
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
if placeholder_or_empty "${CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_ID:-}"; then
  upsert_env_file "${ENV_FILE}" "CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_ID" "casdoor-token-probe-smoke"
fi
if placeholder_or_empty "${CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_SECRET:-}"; then
  upsert_env_file "${ENV_FILE}" "CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_SECRET" "dev-casdoor-token-probe-smoke-$(random_hex 16)"
fi
if placeholder_or_empty "${CASDOOR_TOKEN_PROBE_SMOKE_APPLICATION:-}"; then
  upsert_env_file "${ENV_FILE}" "CASDOOR_TOKEN_PROBE_SMOKE_APPLICATION" "casdoor-token-probe-smoke"
fi
if placeholder_or_empty "${CASDOOR_TOKEN_PROBE_SMOKE_REDIRECT_URI:-}"; then
  upsert_env_file "${ENV_FILE}" "CASDOOR_TOKEN_PROBE_SMOKE_REDIRECT_URI" "http://localhost:3000/open-platform/token-probe/callback"
fi
if placeholder_or_empty "${STUHELPER_CONSOLE_ADMIN_PASSWORD:-}"; then
  upsert_env_file "${ENV_FILE}" "STUHELPER_CONSOLE_ADMIN_PASSWORD" "dev-koishi-admin-$(random_hex 12)"
fi
if placeholder_or_empty "${BOT_SERVICE_TOKEN:-}"; then
  bot_service_token="dev-bot-$(random_hex 24)"
  upsert_env_file "${ENV_FILE}" "BOT_SERVICE_TOKEN" "${bot_service_token}"
  if placeholder_or_empty "${STUHELPER_PLATFORM_SERVICE_TOKEN:-}"; then
    upsert_env_file "${ENV_FILE}" "STUHELPER_PLATFORM_SERVICE_TOKEN" "${bot_service_token}"
  fi
elif placeholder_or_empty "${STUHELPER_PLATFORM_SERVICE_TOKEN:-}"; then
  upsert_env_file "${ENV_FILE}" "STUHELPER_PLATFORM_SERVICE_TOKEN" "${BOT_SERVICE_TOKEN}"
fi

ensure_dev_default "STACK_NAME" "${STACK_NAME:-}" "stuhelper-dev" "stuhelper-prod" "stuhelper-prod-parity"
ensure_dev_default "COMPOSE_PROJECT_NAME" "${COMPOSE_PROJECT_NAME:-}" "stuhelper-dev" "stuhelper-prod" "stuhelper-prod-parity"
ensure_dev_default "APP_ENV" "${APP_ENV:-}" "development" "production" "prod-parity"
ensure_dev_default "API_IP_RATE_LIMIT" "${API_IP_RATE_LIMIT:-}" "5000" "100"
ensure_dev_default "API_GLOBAL_RATE_LIMIT" "${API_GLOBAL_RATE_LIMIT:-}" "50000" "10000"
ensure_dev_default "REVIEW_RATE_POST_LIMIT" "${REVIEW_RATE_POST_LIMIT:-}" "500" "5"
ensure_dev_default "REVIEW_RATE_VOTE_LIMIT" "${REVIEW_RATE_VOTE_LIMIT:-}" "500" "30"
ensure_dev_default "REVIEW_RATE_REPORT_LIMIT" "${REVIEW_RATE_REPORT_LIMIT:-}" "500" "10"
ensure_dev_default "REVIEW_RATE_REPLY_LIMIT" "${REVIEW_RATE_REPLY_LIMIT:-}" "500" "10"
ensure_dev_default "REVIEW_RATE_WRITE_LIMIT" "${REVIEW_RATE_WRITE_LIMIT:-}" "500" "10"
ensure_dev_default "REVIEW_RATE_SEARCH_ANON_LIMIT" "${REVIEW_RATE_SEARCH_ANON_LIMIT:-}" "500" "5"
ensure_dev_default "REVIEW_RATE_SEARCH_USER_LIMIT" "${REVIEW_RATE_SEARCH_USER_LIMIT:-}" "500" "60"
ensure_dev_default "REVIEW_RATE_BATCH_ANON_LIMIT" "${REVIEW_RATE_BATCH_ANON_LIMIT:-}" "500" "5"
ensure_dev_default "REVIEW_RATE_BATCH_USER_LIMIT" "${REVIEW_RATE_BATCH_USER_LIMIT:-}" "500" "60"
ensure_value "OPEN_PLATFORM_CONSENT_BASE_URL" "${OPEN_PLATFORM_CONSENT_BASE_URL:-}" ""
ensure_value "OPEN_PLATFORM_ACCOUNT_BASE_URL" "${OPEN_PLATFORM_ACCOUNT_BASE_URL:-}" ""
ensure_value "OPEN_PLATFORM_DISCLOSURE_APP_LIMIT" "${OPEN_PLATFORM_DISCLOSURE_APP_LIMIT:-}" "600"
ensure_value "OPEN_PLATFORM_DISCLOSURE_APP_USER_LIMIT" "${OPEN_PLATFORM_DISCLOSURE_APP_USER_LIMIT:-}" "120"
ensure_value "OPEN_PLATFORM_DISCLOSURE_ENDPOINT_LIMIT" "${OPEN_PLATFORM_DISCLOSURE_ENDPOINT_LIMIT:-}" "1200"
ensure_value "OPEN_PLATFORM_DISCLOSURE_CONSENT_LIMIT" "${OPEN_PLATFORM_DISCLOSURE_CONSENT_LIMIT:-}" "20"
ensure_value "OPEN_PLATFORM_DISCLOSURE_REPLAY_LIMIT" "${OPEN_PLATFORM_DISCLOSURE_REPLAY_LIMIT:-}" "8"
ensure_value "OPEN_PLATFORM_DISCLOSURE_WINDOW_SECONDS" "${OPEN_PLATFORM_DISCLOSURE_WINDOW_SECONDS:-}" "60"
ensure_value "OPEN_PLATFORM_DISCLOSURE_REPLAY_WINDOW_SECONDS" "${OPEN_PLATFORM_DISCLOSURE_REPLAY_WINDOW_SECONDS:-}" "300"
ensure_value "OPEN_PLATFORM_DISCLOSURE_REPLAY_AUDIT_COOLDOWN_SECONDS" "${OPEN_PLATFORM_DISCLOSURE_REPLAY_AUDIT_COOLDOWN_SECONDS:-}" "600"
ensure_value "OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_REQUIRED" "${OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_REQUIRED:-}" "false"
ensure_value "OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND" "${OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND:-}" ""
ensure_value "OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_TIMEOUT_SECONDS" "${OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_TIMEOUT_SECONDS:-}" "30"
ensure_value "CASDOOR_TOKEN_PROBE_USERNAME" "${CASDOOR_TOKEN_PROBE_USERNAME:-}" ""
ensure_value "CASDOOR_TOKEN_PROBE_PASSWORD" "${CASDOOR_TOKEN_PROBE_PASSWORD:-}" ""
ensure_value "CASDOOR_TOKEN_PROBE_BROWSER_HEADLESS" "${CASDOOR_TOKEN_PROBE_BROWSER_HEADLESS:-}" "true"
ensure_value "CASDOOR_TOKEN_PROBE_BROWSER_EXECUTABLE_PATH" "${CASDOOR_TOKEN_PROBE_BROWSER_EXECUTABLE_PATH:-}" ""
ensure_value "CASDOOR_TOKEN_PROBE_BROWSER_NO_SANDBOX" "${CASDOOR_TOKEN_PROBE_BROWSER_NO_SANDBOX:-}" "true"
ensure_value "CASDOOR_TOKEN_PROBE_LOGIN_TIMEOUT_SECONDS" "${CASDOOR_TOKEN_PROBE_LOGIN_TIMEOUT_SECONDS:-}" "30"
ensure_dev_default \
  "CORS_ORIGINS" \
  "${CORS_ORIGINS:-}" \
  "http://localhost:3000,http://127.0.0.1:3000,http://join.localhost:3000,http://localhost:3001,http://127.0.0.1:3001" \
  "http://localhost:3000,http://localhost:3001" \
  "http://localhost:5173,http://localhost:3001" \
  "http://127.0.0.1:28000,http://127.0.0.1:28001" \
  "http://stuhelper.com,http://join.stuhelper.com,http://sso.stuhelper.com" \
  "https://stuhelper.com,https://join.stuhelper.com,https://sso.stuhelper.com"
replace_legacy_env_pattern "BACKUP_DATABASE_URL" "${BACKUP_DATABASE_URL:-}" "" "postgres://stuhelper_backup:*@postgres:5432/stuhelper?sslmode=disable"
replace_legacy_env_pattern "REPLICATION_DATABASE_URL" "${REPLICATION_DATABASE_URL:-}" "" "postgres://stuhelper_replication:*@postgres:5432/stuhelper?sslmode=disable"
replace_legacy_env_value "EXTERNAL_POSTGRES_ENABLED" "${EXTERNAL_POSTGRES_ENABLED:-}" "false" "true"
replace_legacy_env_value "EXTERNAL_POSTGRES_ALLOW_PLAINTEXT" "${EXTERNAL_POSTGRES_ALLOW_PLAINTEXT:-}" "false" "true"
replace_legacy_env_value "EXTERNAL_DATASTORE_NETWORK" "${EXTERNAL_DATASTORE_NETWORK:-}" "" "stuhelper-prod-parity-baota-net" "baota_net"
replace_legacy_env_value "SHARED_POSTGRES_CONTAINER" "${SHARED_POSTGRES_CONTAINER:-}" "" "stuhelper-prod-parity-postgres"
replace_legacy_env_value "PROD_PARITY_POSTGRES_CONTAINER" "${PROD_PARITY_POSTGRES_CONTAINER:-}" "" "stuhelper-prod-parity-postgres"
replace_legacy_env_value "PROD_PARITY_POSTGRES_PORT" "${PROD_PARITY_POSTGRES_PORT:-}" "" "15432"
replace_legacy_env_value "SHARED_POSTGRES_SUPERUSER" "${SHARED_POSTGRES_SUPERUSER:-}" "" "postgres"
replace_legacy_env_value "SHARED_POSTGRES_DB" "${SHARED_POSTGRES_DB:-}" "" "postgres"
replace_legacy_env_value "POSTGRES_HOST" "${POSTGRES_HOST:-}" "localhost" "postgres"
replace_legacy_env_value "PUBLIC_INGRESS_CASDOOR_UPSTREAM_PREFLIGHT_ENABLED" "${PUBLIC_INGRESS_CASDOOR_UPSTREAM_PREFLIGHT_ENABLED:-}" "false" "true"
replace_legacy_env_value "OPEN_PLATFORM_PRODUCTION_EVIDENCE_ALLOW_LOCAL_TARGETS" "${OPEN_PLATFORM_PRODUCTION_EVIDENCE_ALLOW_LOCAL_TARGETS:-}" "false" "true"
replace_legacy_env_value "OTEL_EXPORTER_OTLP_ENDPOINT" "${OTEL_EXPORTER_OTLP_ENDPOINT:-}" "http://localhost:4318" "http://alloy:4318"
replace_legacy_env_value "OBJECT_STORAGE_ACCESS_KEY_ID" "${OBJECT_STORAGE_ACCESS_KEY_ID:-}" "stuhelper" "stuhelper-prod-parity"
replace_legacy_env_value "TOKEN_COOKIE_SECURE" "${TOKEN_COOKIE_SECURE:-}" "false" "true"
replace_legacy_env_value "TOKEN_COOKIE_DOMAIN" "${TOKEN_COOKIE_DOMAIN:-}" "" ".stuhelper.com" "localhost" "127.0.0.1"
replace_legacy_env_value "ALLOW_LOCAL_ALERT_SINK" "${ALLOW_LOCAL_ALERT_SINK:-}" "false" "true"
replace_legacy_env_value "ALERTMANAGER_WEBHOOK_URL" "${ALERTMANAGER_WEBHOOK_URL:-}" "" "http://alert-webhook-sink:8080/alerts"
load_env

ensure_value "FRONTEND_METRICS_ALLOWED_ORIGINS" "${FRONTEND_METRICS_ALLOWED_ORIGINS:-}" ""
ensure_dev_pattern_default "DATABASE_URL" "${DATABASE_URL:-}" "postgres://stuhelper_app:${STUHELPER_APP_DB_PASSWORD:-}@localhost:5432/stuhelper?sslmode=disable" "postgres://stuhelper_app:*@postgres:5432/stuhelper?sslmode=disable"
ensure_dev_default "POSTGRES_EXTERNAL_PORT" "${POSTGRES_EXTERNAL_PORT:-}" "5432" "15432"
ensure_value "POSTGRES_INTERNAL_SSL_MODE" "${POSTGRES_INTERNAL_SSL_MODE:-}" "disable"
ensure_dev_default "POSTGRES_PGDATA" "${POSTGRES_PGDATA:-}" "/var/lib/postgresql/data" "/var/lib/postgresql/18/docker"
ensure_dev_default "POSTGRES_ARCHIVE_MODE" "${POSTGRES_ARCHIVE_MODE:-}" "off" "on"
ensure_value "POSTGRES_ARCHIVE_TIMEOUT" "${POSTGRES_ARCHIVE_TIMEOUT:-}" "15min"
ensure_dev_default "REDIS_HOST" "${REDIS_HOST:-}" "localhost" "redis"
ensure_value "REDIS_PORT" "${REDIS_PORT:-}" "6379"
ensure_dev_default "REDIS_EXTERNAL_PORT" "${REDIS_EXTERNAL_PORT:-}" "6379" "26379"
ensure_value "REDIS_TLS_ENABLED" "${REDIS_TLS_ENABLED:-}" "true"
ensure_dev_default "REDIS_TLS_CA" "${REDIS_TLS_CA:-}" "/tls/ca.crt" "/redis-tls/ca.crt"
ensure_dev_default "CASDOOR_EXTERNALPORT" "${CASDOOR_EXTERNALPORT:-}" "8085" "28085"
ensure_value "CASDOOR_VERSION" "${CASDOOR_VERSION:-}" "3.31.1"
ensure_dev_default "CASDOOR_ISSUER" "${CASDOOR_ISSUER:-}" "http://localhost:8085" "http://localhost" "http://host.docker.internal:8085" "http://sso.stuhelper.com" "https://sso.stuhelper.com" "http://127.0.0.1:28085"
ensure_dev_default "CASDOOR_INTERNAL_ADDRESS" "${CASDOOR_INTERNAL_ADDRESS:-}" "casdoor:8000" "host.docker.internal:80"
ensure_dev_default "CASDOOR_PUBLIC_AUTH_BASE_URL" "${CASDOOR_PUBLIC_AUTH_BASE_URL:-}" "" "http://sso.stuhelper.com" "https://sso.stuhelper.com"
ensure_dev_default "CASDOOR_REDIRECT_URI" "${CASDOOR_REDIRECT_URI:-}" "http://localhost:8080/api/v1/auth/callback" "http://127.0.0.1:28080/api/v1/auth/callback" "http://stuhelper.com/api/v1/auth/callback" "https://stuhelper.com/api/v1/auth/callback"
ensure_dev_default "CASDOOR_ADDITIONAL_REDIRECT_URIS" "${CASDOOR_ADDITIONAL_REDIRECT_URIS:-}" "http://localhost:3000/api/v1/auth/callback,http://127.0.0.1:3000/api/v1/auth/callback,http://join.localhost:3000/api/v1/auth/callback" "http://localhost:28000/api/v1/auth/callback,http://127.0.0.1:28000/api/v1/auth/callback,http://join.localhost:28000/api/v1/auth/callback"
ensure_value "CASDOOR_ORGANIZATION" "${CASDOOR_ORGANIZATION:-}" "stuhelper"
ensure_value "CASDOOR_ROLES_CLAIM" "${CASDOOR_ROLES_CLAIM:-}" "roles"
ensure_dev_default "CASDOOR_BOOTSTRAP_ENABLED" "${CASDOOR_BOOTSTRAP_ENABLED:-}" "false" "true"
ensure_dev_pattern_default "CASDOOR_BOOTSTRAP_ENV_FILE" "${CASDOOR_BOOTSTRAP_ENV_FILE:-}" ".env.casdoor-bootstrap.local" "*/.run/prod-parity/.env.casdoor-bootstrap.local"
ensure_dev_default "CASDOOR_ADMIN_REDIRECT_URI" "${CASDOOR_ADMIN_REDIRECT_URI:-}" "http://localhost:8080/api/v1/auth/callback" "http://127.0.0.1:28080/api/v1/auth/callback" "http://stuhelper.com/api/v1/auth/callback" "https://stuhelper.com/api/v1/auth/callback"
ensure_dev_default "CASDOOR_ADMIN_ADDITIONAL_REDIRECT_URIS" "${CASDOOR_ADMIN_ADDITIONAL_REDIRECT_URIS:-}" "http://localhost:3001/api/v1/auth/callback,http://127.0.0.1:3001/api/v1/auth/callback" "http://localhost:28001/api/v1/auth/callback,http://127.0.0.1:28001/api/v1/auth/callback"
ensure_dev_default "CASDOOR_UNIAPP_REDIRECT_URI" "${CASDOOR_UNIAPP_REDIRECT_URI:-}" "http://localhost:8080/api/v1/auth/callback" "http://127.0.0.1:28080/api/v1/auth/callback" "http://stuhelper.com/api/v1/auth/callback" "https://stuhelper.com/api/v1/auth/callback"
ensure_dev_default "CASDOOR_SMS_PROVIDER_ENABLED" "${CASDOOR_SMS_PROVIDER_ENABLED:-}" "false" "true"
ensure_value "CASDOOR_SMS_PROVIDER_NAME" "${CASDOOR_SMS_PROVIDER_NAME:-}" "stuhelper-sms"
ensure_value "CASDOOR_SMS_PROVIDER_DISPLAY_NAME" "${CASDOOR_SMS_PROVIDER_DISPLAY_NAME:-}" "StuHelper-SMS"
ensure_value "CASDOOR_SMS_PROVIDER_CATEGORY" "${CASDOOR_SMS_PROVIDER_CATEGORY:-}" "SMS"
ensure_value "CASDOOR_SMS_PROVIDER_TYPE" "${CASDOOR_SMS_PROVIDER_TYPE:-}" "CustomHTTP"
ensure_value "CASDOOR_SMS_PROVIDER_METHOD" "${CASDOOR_SMS_PROVIDER_METHOD:-}" "POST"
ensure_value "CASDOOR_SMS_PROVIDER_TITLE" "${CASDOOR_SMS_PROVIDER_TITLE:-}" "content"
ensure_dev_default "CASDOOR_SMS_PROVIDER_ENDPOINT" "${CASDOOR_SMS_PROVIDER_ENDPOINT:-}" "http://host.docker.internal:8080/internal/sms/send" "http://app:8080/internal/sms/send"
ensure_value "CASDOOR_EMAIL_PROVIDER_ENABLED" "${CASDOOR_EMAIL_PROVIDER_ENABLED:-}" "false"
ensure_dev_default "SMS_ENABLED" "${SMS_ENABLED:-}" "false" "true"
replace_legacy_env_value "SMS_APP_ID" "${SMS_APP_ID:-}" "" "prod-parity-sms-app" "REPLACE_WITH_SMS_APP_ID"
replace_legacy_env_value "SMS_SIGN_NAME" "${SMS_SIGN_NAME:-}" "" "StuHelper" "REPLACE_WITH_SMS_SIGN_NAME"
replace_legacy_env_value "SMS_TEMPLATE_ID" "${SMS_TEMPLATE_ID:-}" "" "prod-parity-template" "REPLACE_WITH_SMS_TEMPLATE_ID"
ensure_dev_default "EMAIL_ENABLED" "${EMAIL_ENABLED:-}" "true" "false"
ensure_dev_default "EMAIL_DRIVER" "${EMAIL_DRIVER:-}" "smtp" "blackhole" "tencent_ses" "resend" "multi"
ensure_value "EMAIL_STUDENT_VERIFICATION_SUBJECT" "${EMAIL_STUDENT_VERIFICATION_SUBJECT:-}" "学生认证验证码"
ensure_dev_default "EMAIL_SMTP_HOST" "${EMAIL_SMTP_HOST:-}" "localhost" "" "mailhog" "REPLACE_WITH_EMAIL_SMTP_HOST"
ensure_dev_default "EMAIL_SMTP_PORT" "${EMAIL_SMTP_PORT:-}" "1025" "587"
ensure_value "EMAIL_SMTP_USERNAME" "${EMAIL_SMTP_USERNAME:-}" ""
ensure_value "EMAIL_SMTP_PASSWORD" "${EMAIL_SMTP_PASSWORD:-}" ""
ensure_dev_default "EMAIL_FROM" "${EMAIL_FROM:-}" "no-reply@stuhelper.local" ""
ensure_value "EMAIL_FROM_NAME" "${EMAIL_FROM_NAME:-}" "StuHelper 系统邮件"
ensure_value "EMAIL_SMTP_USE_TLS" "${EMAIL_SMTP_USE_TLS:-}" "false"
ensure_dev_default "EMAIL_SMTP_STARTTLS" "${EMAIL_SMTP_STARTTLS:-}" "false" "true"
ensure_value "EMAIL_TENCENT_SECRET_ID" "${EMAIL_TENCENT_SECRET_ID:-}" ""
ensure_value "EMAIL_TENCENT_SECRET_KEY" "${EMAIL_TENCENT_SECRET_KEY:-}" ""
ensure_value "EMAIL_TENCENT_REGION" "${EMAIL_TENCENT_REGION:-}" "ap-guangzhou"
ensure_value "EMAIL_TENCENT_ENDPOINT" "${EMAIL_TENCENT_ENDPOINT:-}" "ses.tencentcloudapi.com"
ensure_value "EMAIL_TENCENT_TEMPLATE_ID" "${EMAIL_TENCENT_TEMPLATE_ID:-}" ""
ensure_value "EMAIL_TENCENT_REPLY_TO" "${EMAIL_TENCENT_REPLY_TO:-}" ""
ensure_value "EMAIL_TENCENT_TEMPLATE_PURPOSE" "${EMAIL_TENCENT_TEMPLATE_PURPOSE:-}" "学校邮箱认证"
ensure_value "EMAIL_TENCENT_TEMPLATE_SCHOOL_NAME" "${EMAIL_TENCENT_TEMPLATE_SCHOOL_NAME:-}" "北京航空航天大学"
ensure_value "EMAIL_TENCENT_TEMPLATE_EXPIRE_MINUTES" "${EMAIL_TENCENT_TEMPLATE_EXPIRE_MINUTES:-}" "5"
ensure_value "EMAIL_RESEND_API_KEY" "${EMAIL_RESEND_API_KEY:-}" ""
ensure_value "EMAIL_RESEND_ENDPOINT" "${EMAIL_RESEND_ENDPOINT:-}" "https://api.resend.com/emails"
ensure_value "EMAIL_RESEND_REPLY_TO" "${EMAIL_RESEND_REPLY_TO:-}" ""
ensure_dev_default "BACKEND_EXTERNAL_PORT" "${BACKEND_EXTERNAL_PORT:-}" "8080" "18080" "28080"
ensure_dev_default "WEB_EXTERNAL_PORT" "${WEB_EXTERNAL_PORT:-}" "3000" "18000" "28000"
ensure_dev_default "ADMIN_EXTERNAL_PORT" "${ADMIN_EXTERNAL_PORT:-}" "3001" "18001" "28001"
ensure_dev_default "WEB_PUBLIC_URL" "${WEB_PUBLIC_URL:-}" "http://localhost:3000" "http://127.0.0.1:28000" "http://stuhelper.com" "https://stuhelper.com"
ensure_dev_default "ADMIN_PUBLIC_URL" "${ADMIN_PUBLIC_URL:-}" "http://localhost:3001" "http://127.0.0.1:28001/admin/" "http://stuhelper.com/admin/" "https://stuhelper.com/admin/"
ensure_dev_default "STUHELPER_PLATFORM_BASE_URL" "${STUHELPER_PLATFORM_BASE_URL:-}" "http://localhost:8080" "http://127.0.0.1:28080" "http://stuhelper.com" "https://stuhelper.com"
ensure_dev_default "ADMISSION_PUBLIC_BASE_URL" "${ADMISSION_PUBLIC_BASE_URL:-}" "http://join.localhost:3000" "http://localhost:3000" "http://127.0.0.1:28000" "http://join.stuhelper.com" "https://join.stuhelper.com"
ensure_dev_default "WEB_VITE_API_URL" "${WEB_VITE_API_URL:-}" "/api" ""
ensure_dev_default "WEB_VITE_SSO_URL" "${WEB_VITE_SSO_URL:-}" "http://localhost:8085" "http://host.docker.internal:8085" "http://sso.stuhelper.com" "https://sso.stuhelper.com" "http://127.0.0.1:28085"
ensure_dev_default "WEB_VITE_WEB_URL" "${WEB_VITE_WEB_URL:-}" "http://localhost:3000" "http://stuhelper.com" "https://stuhelper.com" "http://127.0.0.1:28000"
ensure_dev_default "WEB_VITE_QQ_BOT_ENTRY" "${WEB_VITE_QQ_BOT_ENTRY:-}" "" "StuHelper QQ Bot"
ensure_value "WEB_VITE_QQ_BIND_COMMAND" "${WEB_VITE_QQ_BIND_COMMAND:-}" "绑定"
ensure_dev_default "PUBLIC_INGRESS_CASDOOR_UPSTREAM_PREFLIGHT_ENABLED" "${PUBLIC_INGRESS_CASDOOR_UPSTREAM_PREFLIGHT_ENABLED:-}" "false" "true"
ensure_value "WEB_VITE_API_TIMEOUT_MS" "${WEB_VITE_API_TIMEOUT_MS:-}" "15000"
ensure_value "ADMIN_VITE_API_URL" "${ADMIN_VITE_API_URL:-}" "/api/v1"
ensure_value "ADMIN_VITE_BASE" "${ADMIN_VITE_BASE:-}" "/admin/"
ensure_dev_default "OPENFGA_API_URL" "${OPENFGA_API_URL:-}" "http://localhost:8081" "http://openfga:8080"
ensure_dev_default "OPENFGA_HTTP_EXTERNAL_PORT" "${OPENFGA_HTTP_EXTERNAL_PORT:-}" "8081"
ensure_dev_default "OPENFGA_GRPC_EXTERNAL_PORT" "${OPENFGA_GRPC_EXTERNAL_PORT:-}" "8082"
ensure_dev_default "OPENFGA_PLAYGROUND_EXTERNAL_PORT" "${OPENFGA_PLAYGROUND_EXTERNAL_PORT:-}" "3002"
ensure_dev_default "OPENFGA_RESOURCE_SMOKE_MODE" "${OPENFGA_RESOURCE_SMOKE_MODE:-}" "host" "container"
ensure_dev_default "OBJECT_STORAGE_ENDPOINT" "${OBJECT_STORAGE_ENDPOINT:-}" "http://localhost:9000" "http://minio:9000"
ensure_value "OBJECT_STORAGE_REGION" "${OBJECT_STORAGE_REGION:-}" "us-east-1"
ensure_value "OBJECT_STORAGE_BUCKET" "${OBJECT_STORAGE_BUCKET:-}" "stuhelper-identity"
ensure_value "OBJECT_STORAGE_ACCESS_KEY_ID" "${OBJECT_STORAGE_ACCESS_KEY_ID:-}" "stuhelper"
ensure_dev_default "OBJECT_STORAGE_USE_SSL" "${OBJECT_STORAGE_USE_SSL:-}" "false" "true"
ensure_value "OBJECT_STORAGE_FORCE_PATH_STYLE" "${OBJECT_STORAGE_FORCE_PATH_STYLE:-}" "true"
ensure_value "OBJECT_STORAGE_PRESIGN_TTL" "${OBJECT_STORAGE_PRESIGN_TTL:-}" "600"
ensure_dev_default "MINIO_API_EXTERNAL_PORT" "${MINIO_API_EXTERNAL_PORT:-}" "9000" "29000"
ensure_dev_default "MINIO_CONSOLE_EXTERNAL_PORT" "${MINIO_CONSOLE_EXTERNAL_PORT:-}" "9001" "29001"
ensure_value "PROMETHEUS_RETENTION_TIME" "${PROMETHEUS_RETENTION_TIME:-}" "15d"
ensure_value "PROMETHEUS_RETENTION_SIZE" "${PROMETHEUS_RETENTION_SIZE:-}" "20GB"
ensure_value "BACKUP_LOGICAL_RETENTION_DAYS" "${BACKUP_LOGICAL_RETENTION_DAYS:-}" "14"
ensure_value "BACKUP_BASE_RETENTION_DAYS" "${BACKUP_BASE_RETENTION_DAYS:-}" "30"
ensure_value "WAL_ARCHIVE_RETENTION_DAYS" "${WAL_ARCHIVE_RETENTION_DAYS:-}" "14"
ensure_dev_pattern_default "BACKEND_IMAGE_REF" "${BACKEND_IMAGE_REF:-}" "stuhelper/backend:dev-placeholder" "stuhelper/backend:prod-parity-*"
ensure_dev_pattern_default "FRONTEND_IMAGE_REF" "${FRONTEND_IMAGE_REF:-}" "stuhelper/frontend:dev-placeholder" "stuhelper/frontend:prod-parity-*"
ensure_dev_pattern_default "ADMIN_IMAGE_REF" "${ADMIN_IMAGE_REF:-}" "stuhelper/admin:dev-placeholder" "stuhelper/admin:prod-parity-*"

"${SCRIPT_DIR}/render-redis-tls.sh"
"${SCRIPT_DIR}/render-redis-acl.sh"
log "development environment file is ready: ${ENV_FILE}"
