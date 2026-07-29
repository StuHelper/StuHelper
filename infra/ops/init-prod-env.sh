#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"
# shellcheck source=lib/retired-idp-env.sh
source "${SCRIPT_DIR}/lib/retired-idp-env.sh"

require_cmd python3
export STUHELPER_PRESERVE_POSTGRES_URL_PLACEHOLDERS=true

if [[ -z "${ENV_TEMPLATE_FILE:-}" || "${ENV_TEMPLATE_FILE}" == ".env.example" || "${ENV_TEMPLATE_FILE}" == "${REPO_ROOT}/.env.example" ]]; then
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
remove_retired_idp_env_files "${ENV_FILE}" "${SECRETS_ENV_FILE}" "${GENERATED_ENV_FILE}" "${GENERATED_SECRET_ENV_FILE}"

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

ensure_managed_runtime_image() {
  local key="$1"
  local desired="$2"
  shift 2
  local current="${!key-}"

  ensure_prod_default "${key}" "${current}" "${desired}" "$@"
}

placeholder_or_empty() {
  local value="${1:-}"
  [[ -z "${value}" || "${value}" == *"REPLACE_WITH_"* || "${value}" == "ChangeMeBeforeProduction" ]]
}

resolve_env_path() {
  local raw="$1"
  case "${raw}" in
    /*) printf '%s\n' "${raw}" ;;
    *) printf '%s/%s\n' "$(dirname "${ENV_FILE}")" "${raw}" ;;
  esac
}

ensure_bootstrap_env_value() {
  local file="$1"
  local key="$2"
  local desired="$3"
  if ! grep -Eq "^${key}=" "${file}"; then
    upsert_env_file "${file}" "${key}" "${desired}"
  fi
}

load_env

if [[ "${EXTERNAL_POSTGRES_ENABLED:-false}" != "true" ]]; then
  if placeholder_or_empty "${POSTGRES_PASSWORD:-}" || [[ "${POSTGRES_PASSWORD:-}" == "dev123" ]]; then
    upsert_env_file "${SECRETS_ENV_FILE}" "POSTGRES_PASSWORD" "prod-pg-$(random_hex 16)"
  fi
fi
if placeholder_or_empty "${REDIS_PASSWORD:-}" || [[ "${REDIS_PASSWORD:-}" == "dev123" ]]; then
  upsert_env_file "${SECRETS_ENV_FILE}" "REDIS_PASSWORD" "prod-redis-$(random_hex 16)"
fi
if placeholder_or_empty "${REDIS_EXPORTER_PASSWORD:-}" ||
   [[ "${REDIS_EXPORTER_PASSWORD:-}" == "${REDIS_PASSWORD:-}" ]]; then
  upsert_env_file "${SECRETS_ENV_FILE}" "REDIS_EXPORTER_PASSWORD" "prod-redis-metrics-$(random_hex 16)"
fi
if placeholder_or_empty "${STUHELPER_APP_DB_PASSWORD:-}"; then
  upsert_env_file "${SECRETS_ENV_FILE}" "STUHELPER_APP_DB_PASSWORD" "prod-app-$(random_hex 16)"
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
if placeholder_or_empty "${POSTGRES_EXPORTER_DB_PASSWORD:-}"; then
  upsert_env_file "${SECRETS_ENV_FILE}" "POSTGRES_EXPORTER_DB_PASSWORD" "prod-pg-metrics-$(random_hex 16)"
fi
if placeholder_or_empty "${HMAC_SECRET:-}" || [[ "${HMAC_SECRET:-}" == "dev_hmac_secret_change_in_production_32ch" ]]; then
  upsert_env_file "${SECRETS_ENV_FILE}" "HMAC_SECRET" "$(random_hex 32)"
fi
if placeholder_or_empty "${DOC_AES_KEYS:-}"; then
  upsert_env_file "${SECRETS_ENV_FILE}" "DOC_AES_ACTIVE_KEY_ID" "1"
  upsert_env_file "${SECRETS_ENV_FILE}" "DOC_AES_KEYS" "1:$(random_hex 32)"
fi
if placeholder_or_empty "${SMS_SECRET_ID:-}"; then
  upsert_env_file "${SECRETS_ENV_FILE}" "SMS_SECRET_ID" "REPLACE_WITH_SMS_SECRET_ID"
fi
if placeholder_or_empty "${SMS_SECRET_KEY:-}"; then
  upsert_env_file "${SECRETS_ENV_FILE}" "SMS_SECRET_KEY" "REPLACE_WITH_SMS_SECRET_KEY"
fi
if placeholder_or_empty "${EMAIL_TENCENT_SECRET_ID:-}"; then
  upsert_env_file "${SECRETS_ENV_FILE}" "EMAIL_TENCENT_SECRET_ID" "REPLACE_WITH_EMAIL_TENCENT_SECRET_ID"
fi
if placeholder_or_empty "${EMAIL_TENCENT_SECRET_KEY:-}"; then
  upsert_env_file "${SECRETS_ENV_FILE}" "EMAIL_TENCENT_SECRET_KEY" "REPLACE_WITH_EMAIL_TENCENT_SECRET_KEY"
fi
if placeholder_or_empty "${EMAIL_RESEND_API_KEY:-}"; then
  upsert_env_file "${SECRETS_ENV_FILE}" "EMAIL_RESEND_API_KEY" "REPLACE_WITH_EMAIL_RESEND_API_KEY"
fi
if placeholder_or_empty "${EXTERNAL_STUDENT_SOURCE_ORACLE_HOST:-}"; then
  upsert_env_file "${SECRETS_ENV_FILE}" "EXTERNAL_STUDENT_SOURCE_ORACLE_HOST" "REPLACE_WITH_EXTERNAL_STUDENT_SOURCE_ORACLE_HOST"
fi
if placeholder_or_empty "${EXTERNAL_STUDENT_SOURCE_ORACLE_USERNAME:-}"; then
  upsert_env_file "${SECRETS_ENV_FILE}" "EXTERNAL_STUDENT_SOURCE_ORACLE_USERNAME" "REPLACE_WITH_EXTERNAL_STUDENT_SOURCE_ORACLE_USERNAME"
fi
if placeholder_or_empty "${EXTERNAL_STUDENT_SOURCE_ORACLE_PASSWORD:-}"; then
  upsert_env_file "${SECRETS_ENV_FILE}" "EXTERNAL_STUDENT_SOURCE_ORACLE_PASSWORD" "REPLACE_WITH_EXTERNAL_STUDENT_SOURCE_ORACLE_PASSWORD"
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
if placeholder_or_empty "${OBJECT_STORAGE_SECRET_ACCESS_KEY:-}"; then
  upsert_env_file "${SECRETS_ENV_FILE}" "OBJECT_STORAGE_SECRET_ACCESS_KEY" "REPLACE_WITH_OBJECT_STORAGE_SECRET_ACCESS_KEY"
fi
if placeholder_or_empty "${BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY:-}"; then
  upsert_env_file "${SECRETS_ENV_FILE}" "BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY" "REPLACE_WITH_BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY"
fi
if placeholder_or_empty "${CASDOOR_CLIENT_SECRET:-}"; then
  upsert_env_file "${SECRETS_ENV_FILE}" "CASDOOR_CLIENT_SECRET" "prod-casdoor-web-$(random_hex 24)"
fi
if placeholder_or_empty "${CASDOOR_ADMIN_CLIENT_SECRET:-}"; then
  upsert_env_file "${SECRETS_ENV_FILE}" "CASDOOR_ADMIN_CLIENT_SECRET" "prod-casdoor-admin-$(random_hex 24)"
fi
if placeholder_or_empty "${CASDOOR_UNIAPP_CLIENT_SECRET:-}"; then
  upsert_env_file "${SECRETS_ENV_FILE}" "CASDOOR_UNIAPP_CLIENT_SECRET" "prod-casdoor-uniapp-$(random_hex 24)"
fi
if placeholder_or_empty "${CASDOOR_APP_PROVISIONING_CLIENT_SECRET:-}"; then
  upsert_env_file "${SECRETS_ENV_FILE}" "CASDOOR_APP_PROVISIONING_CLIENT_SECRET" "prod-casdoor-app-provisioning-$(random_hex 24)"
fi
if placeholder_or_empty "${CASDOOR_USER_PROFILE_CLIENT_SECRET:-}"; then
  upsert_env_file "${SECRETS_ENV_FILE}" "CASDOOR_USER_PROFILE_CLIENT_SECRET" "prod-casdoor-user-profile-$(random_hex 24)"
fi
if placeholder_or_empty "${CASDOOR_INTROSPECTION_CLIENT_SECRET:-}"; then
  upsert_env_file "${SECRETS_ENV_FILE}" "CASDOOR_INTROSPECTION_CLIENT_SECRET" "prod-casdoor-introspection-$(random_hex 24)"
fi
if placeholder_or_empty "${CASDOOR_ROLE_SYNC_CLIENT_SECRET:-}"; then
  upsert_env_file "${SECRETS_ENV_FILE}" "CASDOOR_ROLE_SYNC_CLIENT_SECRET" "prod-casdoor-role-sync-$(random_hex 24)"
fi
if placeholder_or_empty "${CASDOOR_USER_LOOKUP_CLIENT_SECRET:-}"; then
  upsert_env_file "${SECRETS_ENV_FILE}" "CASDOOR_USER_LOOKUP_CLIENT_SECRET" "prod-casdoor-user-lookup-$(random_hex 24)"
fi
if placeholder_or_empty "${CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_SECRET:-}"; then
  upsert_env_file "${SECRETS_ENV_FILE}" "CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_SECRET" "prod-casdoor-token-probe-smoke-$(random_hex 24)"
fi
if placeholder_or_empty "${CASDOOR_TOKEN_PROBE_USERNAME:-}"; then
  upsert_env_file "${SECRETS_ENV_FILE}" "CASDOOR_TOKEN_PROBE_USERNAME" "REPLACE_WITH_CASDOOR_TOKEN_PROBE_USERNAME"
fi
if placeholder_or_empty "${CASDOOR_TOKEN_PROBE_PASSWORD:-}"; then
  upsert_env_file "${SECRETS_ENV_FILE}" "CASDOOR_TOKEN_PROBE_PASSWORD" "REPLACE_WITH_CASDOOR_TOKEN_PROBE_PASSWORD"
fi
load_env

ensure_prod_default "STACK_NAME" "${STACK_NAME:-}" "stuhelper-prod" "stuhelper-dev" "stuhelper"
ensure_prod_default "APP_ENV" "${APP_ENV:-}" "production" "development" "prod-parity"
ensure_value "LOG_LEVEL" "${LOG_LEVEL:-}" "info"
ensure_prod_default "LOG_FORMAT" "${LOG_FORMAT:-}" "json" "console"
ensure_value "LOG_OUTPUT" "${LOG_OUTPUT:-}" "stdout"
ensure_value "EXTERNAL_POSTGRES_ENABLED" "${EXTERNAL_POSTGRES_ENABLED:-}" "false"
ensure_value "EXTERNAL_POSTGRES_ALLOW_PLAINTEXT" "${EXTERNAL_POSTGRES_ALLOW_PLAINTEXT:-}" "false"
ensure_value "EXTERNAL_DATASTORE_NETWORK" "${EXTERNAL_DATASTORE_NETWORK:-}" ""
if [[ "${EXTERNAL_POSTGRES_ALLOW_PLAINTEXT:-false}" == "true" ]]; then
  ensure_value "DATABASE_URL" "${DATABASE_URL:-}" "postgres://stuhelper_app:REPLACE_WITH_STUHELPER_APP_DB_PASSWORD@postgres:5432/stuhelper?sslmode=disable"
  ensure_value "BACKUP_DATABASE_URL" "${BACKUP_DATABASE_URL:-}" "postgres://stuhelper_backup:REPLACE_WITH_STUHELPER_BACKUP_DB_PASSWORD@postgres:5432/stuhelper?sslmode=disable"
  ensure_value "REPLICATION_DATABASE_URL" "${REPLICATION_DATABASE_URL:-}" "postgres://stuhelper_replication:REPLACE_WITH_STUHELPER_REPLICATION_DB_PASSWORD@postgres:5432/stuhelper?sslmode=disable"
  ensure_value "DB_SSL_MODE" "${DB_SSL_MODE:-}" "disable"
  ensure_value "POSTGRES_ENABLE_SSL" "${POSTGRES_ENABLE_SSL:-}" "off"
  ensure_value "POSTGRES_INTERNAL_SSL_MODE" "${POSTGRES_INTERNAL_SSL_MODE:-}" "disable"
  ensure_value "POSTGRES_CLIENT_CA_HOST_PATH" "${POSTGRES_CLIENT_CA_HOST_PATH:-}" ""
else
  ensure_prod_default "DATABASE_URL" "${DATABASE_URL:-}" "postgres://stuhelper_app:REPLACE_WITH_STUHELPER_APP_DB_PASSWORD@postgres:5432/stuhelper?sslmode=verify-full&sslrootcert=/tls/ca.crt" "postgres://stuhelper:dev123@localhost:5432/stuhelper?sslmode=disable" "postgres://stuhelper_app:REPLACE_WITH_STUHELPER_APP_DB_PASSWORD@localhost:5432/stuhelper?sslmode=disable"
  ensure_prod_default "BACKUP_DATABASE_URL" "${BACKUP_DATABASE_URL:-}" "postgres://stuhelper_backup:REPLACE_WITH_STUHELPER_BACKUP_DB_PASSWORD@postgres:5432/stuhelper?sslmode=verify-full&sslrootcert=/tls/ca.crt"
  ensure_prod_default "REPLICATION_DATABASE_URL" "${REPLICATION_DATABASE_URL:-}" "postgres://stuhelper_replication:REPLACE_WITH_STUHELPER_REPLICATION_DB_PASSWORD@postgres:5432/stuhelper?sslmode=verify-full&sslrootcert=/tls/ca.crt"
  ensure_prod_default "DB_SSL_MODE" "${DB_SSL_MODE:-}" "verify-full" "require" "disable"
  ensure_prod_default "DB_SSL_ROOT_CERT" "${DB_SSL_ROOT_CERT:-}" "/tls/ca.crt"
  ensure_prod_default "POSTGRES_ENABLE_SSL" "${POSTGRES_ENABLE_SSL:-}" "on" "off"
  ensure_prod_default "POSTGRES_INTERNAL_SSL_MODE" "${POSTGRES_INTERNAL_SSL_MODE:-}" "verify-full" "require" "disable"
  ensure_value "POSTGRES_CLIENT_CA_HOST_PATH" "${POSTGRES_CLIENT_CA_HOST_PATH:-}" ""
fi
ensure_prod_default "POSTGRES_PGDATA" "${POSTGRES_PGDATA:-}" "/var/lib/postgresql/data" "/var/lib/postgresql/18/docker"
ensure_value "POSTGRES_ARCHIVE_MODE" "${POSTGRES_ARCHIVE_MODE:-}" "off"
ensure_value "POSTGRES_ARCHIVE_TIMEOUT" "${POSTGRES_ARCHIVE_TIMEOUT:-}" "15min"
ensure_prod_default "REDIS_HOST" "${REDIS_HOST:-}" "redis" "localhost"
ensure_value "REDIS_PORT" "${REDIS_PORT:-}" "6379"
ensure_value "REDIS_USERNAME" "${REDIS_USERNAME+x}" "stuhelper_app"
ensure_value "REDIS_EXPORTER_USERNAME" "${REDIS_EXPORTER_USERNAME+x}" "stuhelper_metrics"
ensure_prod_default "REDIS_TLS_ENABLED" "${REDIS_TLS_ENABLED:-}" "true" "false"
ensure_prod_default "REDIS_TLS_CA" "${REDIS_TLS_CA:-}" "/tls/ca.crt"
ensure_prod_default "CORS_ORIGINS" "${CORS_ORIGINS:-}" "https://stuhelper.com,https://join.stuhelper.com,https://sso.stuhelper.com" "https://stuhelper.com" "REPLACE_WITH_PRODUCTION_CORS_ORIGINS" "http://localhost:3000,http://localhost:3001" "http://localhost:3000,http://127.0.0.1:3000,http://localhost:3001,http://127.0.0.1:3001" "http://localhost:3000,http://127.0.0.1:3000,http://join.localhost:3000,http://localhost:3001,http://127.0.0.1:3001"
ensure_prod_default "ADMISSION_PUBLIC_BASE_URL" "${ADMISSION_PUBLIC_BASE_URL:-}" "https://join.stuhelper.com" "REPLACE_WITH_ADMISSION_PUBLIC_BASE_URL" "http://localhost:3000" "http://join.localhost:3000" "http://join.stuhelper.com"
ensure_value "ADMISSION_PRODUCTION_READINESS_ENABLED" "${ADMISSION_PRODUCTION_READINESS_ENABLED:-}" "true"
ensure_value "ADMISSION_READINESS_REQUIRED_PLATFORM" "${ADMISSION_READINESS_REQUIRED_PLATFORM:-}" "qq"
ensure_value "ADMISSION_READINESS_REQUIRED_GUILD_IDS" "${ADMISSION_READINESS_REQUIRED_GUILD_IDS:-}" ""
ensure_value "ADMISSION_READINESS_REQUIRED_SCHOOL_CODES" "${ADMISSION_READINESS_REQUIRED_SCHOOL_CODES:-}" ""
ensure_value "ADMISSION_READINESS_REQUIRED_SCHOOL_IDS" "${ADMISSION_READINESS_REQUIRED_SCHOOL_IDS:-}" ""
ensure_prod_default "STUHELPER_FRESHMAN_MATERIAL_HOSTS" "${STUHELPER_FRESHMAN_MATERIAL_HOSTS:-}" "stuhelper.com,join.stuhelper.com" "cdn.example.test" "stuhelper.com"
ensure_value "TRUSTED_PROXIES" "${TRUSTED_PROXIES:-}" "127.0.0.1/32,172.16.0.0/12,192.168.0.0/16"
ensure_prod_default "OTEL_ENABLED" "${OTEL_ENABLED:-}" "true" "false"
ensure_value "OTEL_SERVICE_NAME" "${OTEL_SERVICE_NAME:-}" "stuhelper-backend"
ensure_value "OTEL_SERVICE_NAMESPACE" "${OTEL_SERVICE_NAMESPACE:-}" "stuhelper"
ensure_prod_default "OTEL_EXPORTER_OTLP_ENDPOINT" "${OTEL_EXPORTER_OTLP_ENDPOINT:-}" "http://alloy:4318" "http://localhost:4318"
ensure_value "OTEL_EXPORTER_OTLP_INSECURE" "${OTEL_EXPORTER_OTLP_INSECURE:-}" "true"
ensure_prod_default "FRONTEND_METRICS_ALLOWED_ORIGINS" "${FRONTEND_METRICS_ALLOWED_ORIGINS:-}" "https://stuhelper.com" "http://localhost:3000" "REPLACE_WITH_FRONTEND_METRICS_ALLOWED_ORIGINS"
ensure_prod_default "TOKEN_COOKIE_SECURE" "${TOKEN_COOKIE_SECURE:-}" "true" "false"
ensure_prod_default "TOKEN_COOKIE_DOMAIN" "${TOKEN_COOKIE_DOMAIN:-}" ".stuhelper.com" "localhost" "127.0.0.1"
ensure_prod_default "CASDOOR_ISSUER" "${CASDOOR_ISSUER:-}" "https://sso.stuhelper.com" "REPLACE_WITH_CASDOOR_ISSUER" "http://localhost:8085" "http://localhost"
ensure_prod_default "CASDOOR_INTERNAL_ADDRESS" "${CASDOOR_INTERNAL_ADDRESS:-}" "" "host.docker.internal:8085" "casdoor:8000"
ensure_prod_default "CASDOOR_PUBLIC_AUTH_BASE_URL" "${CASDOOR_PUBLIC_AUTH_BASE_URL:-}" "https://sso.stuhelper.com" "REPLACE_WITH_CASDOOR_PUBLIC_AUTH_BASE_URL" "http://localhost:8085"
ensure_prod_default "CASDOOR_REDIRECT_URI" "${CASDOOR_REDIRECT_URI:-}" "https://stuhelper.com/api/v1/auth/callback" "REPLACE_WITH_CASDOOR_REDIRECT_URI" "http://localhost:8080/api/v1/auth/callback"
ensure_prod_default "CASDOOR_CLIENT_ID" "${CASDOOR_CLIENT_ID:-}" "REPLACE_WITH_CASDOOR_CLIENT_ID" "stuhelper-web"
ensure_value "CASDOOR_ORGANIZATION" "${CASDOOR_ORGANIZATION:-}" "stuhelper"
ensure_value "CASDOOR_ROLES_CLAIM" "${CASDOOR_ROLES_CLAIM:-}" "roles"
ensure_prod_default "CASDOOR_BOOTSTRAP_ENABLED" "${CASDOOR_BOOTSTRAP_ENABLED:-}" "true" "false"
ensure_value "CASDOOR_BOOTSTRAP_ENV_FILE" "${CASDOOR_BOOTSTRAP_ENV_FILE:-}" ".env.casdoor-bootstrap.local"
ensure_value "CASDOOR_ADMIN_CLIENT_ID" "${CASDOOR_ADMIN_CLIENT_ID:-}" "stuhelper-admin"
ensure_prod_default "CASDOOR_ADMIN_REDIRECT_URI" "${CASDOOR_ADMIN_REDIRECT_URI:-}" "https://stuhelper.com/api/v1/auth/callback" "REPLACE_WITH_CASDOOR_ADMIN_REDIRECT_URI" "http://localhost:8080/api/v1/auth/callback"
ensure_value "CASDOOR_UNIAPP_CLIENT_ID" "${CASDOOR_UNIAPP_CLIENT_ID:-}" "stuhelper-uniapp"
ensure_prod_default "CASDOOR_UNIAPP_REDIRECT_URI" "${CASDOOR_UNIAPP_REDIRECT_URI:-}" "https://stuhelper.com/api/v1/auth/callback" "REPLACE_WITH_CASDOOR_UNIAPP_REDIRECT_URI" "http://localhost:8080/api/v1/auth/callback"
ensure_prod_default "CASDOOR_SMS_PROVIDER_ENABLED" "${CASDOOR_SMS_PROVIDER_ENABLED:-}" "true" "false"
ensure_value "CASDOOR_SMS_PROVIDER_NAME" "${CASDOOR_SMS_PROVIDER_NAME:-}" "stuhelper-sms"
ensure_value "CASDOOR_SMS_PROVIDER_DISPLAY_NAME" "${CASDOOR_SMS_PROVIDER_DISPLAY_NAME:-}" "StuHelper-SMS"
ensure_value "CASDOOR_SMS_PROVIDER_CATEGORY" "${CASDOOR_SMS_PROVIDER_CATEGORY:-}" "SMS"
ensure_value "CASDOOR_SMS_PROVIDER_TYPE" "${CASDOOR_SMS_PROVIDER_TYPE:-}" "CustomHTTP"
ensure_value "CASDOOR_SMS_PROVIDER_METHOD" "${CASDOOR_SMS_PROVIDER_METHOD:-}" "POST"
ensure_prod_default "CASDOOR_SMS_PROVIDER_TITLE" "${CASDOOR_SMS_PROVIDER_TITLE:-}" "content"
ensure_prod_default "CASDOOR_SMS_PROVIDER_ENDPOINT" "${CASDOOR_SMS_PROVIDER_ENDPOINT:-}" "http://app:8080/internal/sms/send" "http://host.docker.internal:8080/internal/sms/send"
ensure_value "CASDOOR_EMAIL_PROVIDER_ENABLED" "${CASDOOR_EMAIL_PROVIDER_ENABLED:-}" "false"
ensure_prod_default "SMS_ENABLED" "${SMS_ENABLED:-}" "true" "false"
ensure_prod_default "SMS_APP_ID" "${SMS_APP_ID:-}" "REPLACE_WITH_SMS_APP_ID"
ensure_prod_default "SMS_SIGN_NAME" "${SMS_SIGN_NAME:-}" "REPLACE_WITH_SMS_SIGN_NAME"
ensure_prod_default "SMS_TEMPLATE_ID" "${SMS_TEMPLATE_ID:-}" "REPLACE_WITH_SMS_TEMPLATE_ID"
ensure_value "SMS_REGION" "${SMS_REGION:-}" "ap-beijing"
ensure_prod_default "EMAIL_ENABLED" "${EMAIL_ENABLED:-}" "true" "false"
ensure_prod_default "EMAIL_DRIVER" "${EMAIL_DRIVER:-}" "multi" "tencent_ses" "resend" "smtp" "blackhole"
ensure_value "EMAIL_STUDENT_VERIFICATION_SUBJECT" "${EMAIL_STUDENT_VERIFICATION_SUBJECT:-}" "学生认证验证码"
ensure_prod_default "EMAIL_SMTP_HOST" "${EMAIL_SMTP_HOST:-}" "" "REPLACE_WITH_EMAIL_SMTP_HOST"
ensure_value "EMAIL_SMTP_PORT" "${EMAIL_SMTP_PORT:-}" "587"
ensure_prod_default "EMAIL_SMTP_USERNAME" "${EMAIL_SMTP_USERNAME:-}" "" "REPLACE_WITH_EMAIL_SMTP_USERNAME"
ensure_prod_default "EMAIL_SMTP_PASSWORD" "${EMAIL_SMTP_PASSWORD:-}" "" "REPLACE_WITH_EMAIL_SMTP_PASSWORD"
ensure_prod_default "EMAIL_FROM" "${EMAIL_FROM:-}" "noreply@notify.stuhelper.com" "REPLACE_WITH_EMAIL_FROM"
ensure_value "EMAIL_FROM_NAME" "${EMAIL_FROM_NAME:-}" "StuHelper 系统邮件"
ensure_value "EMAIL_SMTP_USE_TLS" "${EMAIL_SMTP_USE_TLS:-}" "false"
ensure_value "EMAIL_SMTP_STARTTLS" "${EMAIL_SMTP_STARTTLS:-}" "true"
ensure_value "EMAIL_TENCENT_REGION" "${EMAIL_TENCENT_REGION:-}" "ap-guangzhou"
ensure_value "EMAIL_TENCENT_ENDPOINT" "${EMAIL_TENCENT_ENDPOINT:-}" "ses.tencentcloudapi.com"
ensure_prod_default "EMAIL_TENCENT_TEMPLATE_ID" "${EMAIL_TENCENT_TEMPLATE_ID:-}" "49779" "REPLACE_WITH_EMAIL_TENCENT_TEMPLATE_ID"
ensure_value "EMAIL_TENCENT_REPLY_TO" "${EMAIL_TENCENT_REPLY_TO:-}" ""
ensure_value "EMAIL_TENCENT_TEMPLATE_PURPOSE" "${EMAIL_TENCENT_TEMPLATE_PURPOSE:-}" "学校邮箱认证"
ensure_value "EMAIL_TENCENT_TEMPLATE_SCHOOL_NAME" "${EMAIL_TENCENT_TEMPLATE_SCHOOL_NAME:-}" "北京航空航天大学"
ensure_value "EMAIL_TENCENT_TEMPLATE_EXPIRE_MINUTES" "${EMAIL_TENCENT_TEMPLATE_EXPIRE_MINUTES:-}" "5"
ensure_value "EMAIL_RESEND_ENDPOINT" "${EMAIL_RESEND_ENDPOINT:-}" "https://api.resend.com/emails"
ensure_value "EMAIL_RESEND_REPLY_TO" "${EMAIL_RESEND_REPLY_TO:-}" ""
ensure_value "EXTERNAL_STUDENT_SOURCE_ENABLED" "${EXTERNAL_STUDENT_SOURCE_ENABLED:-}" "false"
ensure_value "EXTERNAL_STUDENT_SOURCE_NAME" "${EXTERNAL_STUDENT_SOURCE_NAME:-}" "buaa-academic-oracle"
ensure_value "EXTERNAL_STUDENT_SOURCE_PROVIDER" "${EXTERNAL_STUDENT_SOURCE_PROVIDER:-}" "oracle"
ensure_value "EXTERNAL_STUDENT_SOURCE_SCHOOL_CODE" "${EXTERNAL_STUDENT_SOURCE_SCHOOL_CODE:-}" "4111010006"
ensure_value "EXTERNAL_STUDENT_SOURCE_ORACLE_PORT" "${EXTERNAL_STUDENT_SOURCE_ORACLE_PORT:-}" "2484"
ensure_value "EXTERNAL_STUDENT_SOURCE_ORACLE_SERVICE_NAME" "${EXTERNAL_STUDENT_SOURCE_ORACLE_SERVICE_NAME:-}" "ORCLPDB1"
ensure_value "EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_MODE" "${EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_MODE:-}" "verify-full"
ensure_value "EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_CA_FILE" "${EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_CA_FILE:-}" "/external-student-source-tls/ca.crt"
ensure_value "EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_CA_HOST_PATH" "${EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_CA_HOST_PATH:-}" ""
ensure_value "EXTERNAL_STUDENT_SOURCE_ORACLE_SCHEMA" "${EXTERNAL_STUDENT_SOURCE_ORACLE_SCHEMA:-}" "USR_JWBIZ"
ensure_value "EXTERNAL_STUDENT_SOURCE_ORACLE_TABLE" "${EXTERNAL_STUDENT_SOURCE_ORACLE_TABLE:-}" "T_XS_JBXX"
ensure_value "EXTERNAL_STUDENT_SOURCE_ORACLE_STUDENT_ID_COLUMN" "${EXTERNAL_STUDENT_SOURCE_ORACLE_STUDENT_ID_COLUMN:-}" "XH"
ensure_value "EXTERNAL_STUDENT_SOURCE_ORACLE_STUDENT_NAME_COLUMN" "${EXTERNAL_STUDENT_SOURCE_ORACLE_STUDENT_NAME_COLUMN:-}" "XM"
ensure_value "EXTERNAL_STUDENT_SOURCE_ORACLE_CONNECT_TIMEOUT_SECONDS" "${EXTERNAL_STUDENT_SOURCE_ORACLE_CONNECT_TIMEOUT_SECONDS:-}" "5"
ensure_value "EXTERNAL_STUDENT_SOURCE_ORACLE_QUERY_TIMEOUT_SECONDS" "${EXTERNAL_STUDENT_SOURCE_ORACLE_QUERY_TIMEOUT_SECONDS:-}" "3"
ensure_value "EXTERNAL_STUDENT_SOURCE_ORACLE_MAX_OPEN_CONNS" "${EXTERNAL_STUDENT_SOURCE_ORACLE_MAX_OPEN_CONNS:-}" "4"
ensure_value "EXTERNAL_STUDENT_SOURCE_ORACLE_MAX_IDLE_CONNS" "${EXTERNAL_STUDENT_SOURCE_ORACLE_MAX_IDLE_CONNS:-}" "1"
ensure_value "EXTERNAL_STUDENT_SOURCE_ORACLE_CONN_MAX_LIFETIME_SECONDS" "${EXTERNAL_STUDENT_SOURCE_ORACLE_CONN_MAX_LIFETIME_SECONDS:-}" "300"
ensure_value "EXTERNAL_STUDENT_SOURCE_ORACLE_CONN_MAX_IDLE_TIME_SECONDS" "${EXTERNAL_STUDENT_SOURCE_ORACLE_CONN_MAX_IDLE_TIME_SECONDS:-}" "60"
ensure_value "EXTERNAL_STUDENT_SOURCE_ORACLE_BREAKER_FAILURE_THRESHOLD" "${EXTERNAL_STUDENT_SOURCE_ORACLE_BREAKER_FAILURE_THRESHOLD:-}" "5"
ensure_value "EXTERNAL_STUDENT_SOURCE_ORACLE_BREAKER_SUCCESS_THRESHOLD" "${EXTERNAL_STUDENT_SOURCE_ORACLE_BREAKER_SUCCESS_THRESHOLD:-}" "2"
ensure_value "EXTERNAL_STUDENT_SOURCE_ORACLE_BREAKER_OPEN_SECONDS" "${EXTERNAL_STUDENT_SOURCE_ORACLE_BREAKER_OPEN_SECONDS:-}" "30"
ensure_value "EXTERNAL_STUDENT_SOURCE_SMOKE_MODE" "${EXTERNAL_STUDENT_SOURCE_SMOKE_MODE:-}" "container"
ensure_value "EXTERNAL_STUDENT_SOURCE_SMOKE_COMMAND" "${EXTERNAL_STUDENT_SOURCE_SMOKE_COMMAND:-}" "/app/external-student-source-smoke"
ensure_value "EXTERNAL_STUDENT_SOURCE_SMOKE_TIMEOUT_SECONDS" "${EXTERNAL_STUDENT_SOURCE_SMOKE_TIMEOUT_SECONDS:-}" "15"
ensure_value "EXTERNAL_STUDENT_SOURCE_SMOKE_REQUIRE_READABLE_RECORD" "${EXTERNAL_STUDENT_SOURCE_SMOKE_REQUIRE_READABLE_RECORD:-}" "true"
ensure_value "EXTERNAL_STUDENT_SOURCE_SMOKE_REQUIRE_SAMPLE" "${EXTERNAL_STUDENT_SOURCE_SMOKE_REQUIRE_SAMPLE:-}" "false"
ensure_value "EXTERNAL_STUDENT_SOURCE_SMOKE_STUDENT_ID" "${EXTERNAL_STUDENT_SOURCE_SMOKE_STUDENT_ID:-}" ""
ensure_value "EXTERNAL_STUDENT_SOURCE_SMOKE_EXPECTED_NAME" "${EXTERNAL_STUDENT_SOURCE_SMOKE_EXPECTED_NAME:-}" ""
ensure_value "EXTERNAL_STUDENT_SOURCE_SMOKE_EVIDENCE_FILE" "${EXTERNAL_STUDENT_SOURCE_SMOKE_EVIDENCE_FILE:-}" "infra/generated/external-student-source-smoke.json"
ensure_value "ADMISSION_MVP_PRODUCTION_RUN_EXTERNAL_STUDENT_SOURCE_SMOKE" "${ADMISSION_MVP_PRODUCTION_RUN_EXTERNAL_STUDENT_SOURCE_SMOKE:-}" "auto"
ensure_prod_default "CASDOOR_APP_PROVISIONING_CLIENT_ID" "${CASDOOR_APP_PROVISIONING_CLIENT_ID:-}" "casdoor-admin-app-provisioning" "REPLACE_WITH_CASDOOR_APP_PROVISIONING_CLIENT_ID"
ensure_prod_default "CASDOOR_APP_PROVISIONING_APPLICATION" "${CASDOOR_APP_PROVISIONING_APPLICATION:-}" "casdoor-admin-app-provisioning" "REPLACE_WITH_CASDOOR_APP_PROVISIONING_APPLICATION"
ensure_prod_default "CASDOOR_USER_PROFILE_CLIENT_ID" "${CASDOOR_USER_PROFILE_CLIENT_ID:-}" "casdoor-admin-user-profile" "REPLACE_WITH_CASDOOR_USER_PROFILE_CLIENT_ID"
ensure_prod_default "CASDOOR_USER_PROFILE_APPLICATION" "${CASDOOR_USER_PROFILE_APPLICATION:-}" "casdoor-admin-user-profile" "REPLACE_WITH_CASDOOR_USER_PROFILE_APPLICATION"
ensure_prod_default "CASDOOR_INTROSPECTION_CLIENT_ID" "${CASDOOR_INTROSPECTION_CLIENT_ID:-}" "casdoor-token-introspection" "REPLACE_WITH_CASDOOR_INTROSPECTION_CLIENT_ID"
ensure_prod_default "CASDOOR_INTROSPECTION_APPLICATION" "${CASDOOR_INTROSPECTION_APPLICATION:-}" "casdoor-token-introspection" "REPLACE_WITH_CASDOOR_INTROSPECTION_APPLICATION"
ensure_prod_default "CASDOOR_ROLE_SYNC_CLIENT_ID" "${CASDOOR_ROLE_SYNC_CLIENT_ID:-}" "casdoor-admin-role-sync" "REPLACE_WITH_CASDOOR_ROLE_SYNC_CLIENT_ID"
ensure_prod_default "CASDOOR_ROLE_SYNC_APPLICATION" "${CASDOOR_ROLE_SYNC_APPLICATION:-}" "casdoor-admin-role-sync" "REPLACE_WITH_CASDOOR_ROLE_SYNC_APPLICATION"
ensure_prod_default "CASDOOR_USER_LOOKUP_CLIENT_ID" "${CASDOOR_USER_LOOKUP_CLIENT_ID:-}" "casdoor-admin-user-lookup" "REPLACE_WITH_CASDOOR_USER_LOOKUP_CLIENT_ID"
ensure_prod_default "CASDOOR_USER_LOOKUP_APPLICATION" "${CASDOOR_USER_LOOKUP_APPLICATION:-}" "casdoor-admin-user-lookup" "REPLACE_WITH_CASDOOR_USER_LOOKUP_APPLICATION"
ensure_prod_default "CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_ID" "${CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_ID:-}" "casdoor-token-probe-smoke" "REPLACE_WITH_CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_ID"
ensure_prod_default "CASDOOR_TOKEN_PROBE_SMOKE_APPLICATION" "${CASDOOR_TOKEN_PROBE_SMOKE_APPLICATION:-}" "casdoor-token-probe-smoke" "REPLACE_WITH_CASDOOR_TOKEN_PROBE_SMOKE_APPLICATION"
ensure_prod_default "CASDOOR_TOKEN_PROBE_SMOKE_REDIRECT_URI" "${CASDOOR_TOKEN_PROBE_SMOKE_REDIRECT_URI:-}" "https://stuhelper.com/open-platform/token-probe/callback" "REPLACE_WITH_CASDOOR_TOKEN_PROBE_SMOKE_REDIRECT_URI" "http://localhost:3000/open-platform/token-probe/callback"
ensure_prod_default "OPEN_PLATFORM_CONSENT_BASE_URL" "${OPEN_PLATFORM_CONSENT_BASE_URL:-}" "https://stuhelper.com" "http://localhost:3000" "REPLACE_WITH_OPEN_PLATFORM_CONSENT_BASE_URL"
ensure_prod_default "OPEN_PLATFORM_ACCOUNT_BASE_URL" "${OPEN_PLATFORM_ACCOUNT_BASE_URL:-}" "https://stuhelper.com" "http://localhost:3000" "REPLACE_WITH_OPEN_PLATFORM_ACCOUNT_BASE_URL"
ensure_value "OPEN_PLATFORM_DISCLOSURE_APP_LIMIT" "${OPEN_PLATFORM_DISCLOSURE_APP_LIMIT:-}" "600"
ensure_value "OPEN_PLATFORM_DISCLOSURE_APP_USER_LIMIT" "${OPEN_PLATFORM_DISCLOSURE_APP_USER_LIMIT:-}" "120"
ensure_value "OPEN_PLATFORM_DISCLOSURE_ENDPOINT_LIMIT" "${OPEN_PLATFORM_DISCLOSURE_ENDPOINT_LIMIT:-}" "1200"
ensure_value "OPEN_PLATFORM_DISCLOSURE_CONSENT_LIMIT" "${OPEN_PLATFORM_DISCLOSURE_CONSENT_LIMIT:-}" "20"
ensure_value "OPEN_PLATFORM_DISCLOSURE_REPLAY_LIMIT" "${OPEN_PLATFORM_DISCLOSURE_REPLAY_LIMIT:-}" "8"
ensure_value "OPEN_PLATFORM_DISCLOSURE_WINDOW_SECONDS" "${OPEN_PLATFORM_DISCLOSURE_WINDOW_SECONDS:-}" "60"
ensure_value "OPEN_PLATFORM_DISCLOSURE_REPLAY_WINDOW_SECONDS" "${OPEN_PLATFORM_DISCLOSURE_REPLAY_WINDOW_SECONDS:-}" "300"
ensure_value "OPEN_PLATFORM_DISCLOSURE_REPLAY_AUDIT_COOLDOWN_SECONDS" "${OPEN_PLATFORM_DISCLOSURE_REPLAY_AUDIT_COOLDOWN_SECONDS:-}" "600"
ensure_prod_default "OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_REQUIRED" "${OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_REQUIRED:-}" "true" "false"
ensure_prod_default "OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND" "${OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND:-}" "/app/casdoor-runtime-token-probe-runner.mjs" "REPLACE_WITH_OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND"
ensure_value "OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_TIMEOUT_SECONDS" "${OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_TIMEOUT_SECONDS:-}" "30"
ensure_prod_default "OPEN_PLATFORM_PRODUCTION_EVIDENCE_ALLOW_LOCAL_TARGETS" "${OPEN_PLATFORM_PRODUCTION_EVIDENCE_ALLOW_LOCAL_TARGETS:-}" "false" "true"
ensure_value "CASDOOR_TOKEN_PROBE_BROWSER_HEADLESS" "${CASDOOR_TOKEN_PROBE_BROWSER_HEADLESS:-}" "true"
ensure_prod_default "CASDOOR_TOKEN_PROBE_BROWSER_EXECUTABLE_PATH" "${CASDOOR_TOKEN_PROBE_BROWSER_EXECUTABLE_PATH:-}" "/usr/bin/chromium-browser" "/usr/bin/google-chrome"
ensure_value "CASDOOR_TOKEN_PROBE_BROWSER_NO_SANDBOX" "${CASDOOR_TOKEN_PROBE_BROWSER_NO_SANDBOX:-}" "true"
ensure_value "CASDOOR_TOKEN_PROBE_LOGIN_TIMEOUT_SECONDS" "${CASDOOR_TOKEN_PROBE_LOGIN_TIMEOUT_SECONDS:-}" "30"
casdoor_bootstrap_env_file="$(resolve_env_path "${CASDOOR_BOOTSTRAP_ENV_FILE:-.env.casdoor-bootstrap.local}")"
mkdir -p "$(dirname "${casdoor_bootstrap_env_file}")"
touch "${casdoor_bootstrap_env_file}"
ensure_bootstrap_env_value "${casdoor_bootstrap_env_file}" "CASDOOR_BOOTSTRAP_CLIENT_ID" "REPLACE_WITH_CASDOOR_BOOTSTRAP_CLIENT_ID"
ensure_bootstrap_env_value "${casdoor_bootstrap_env_file}" "CASDOOR_BOOTSTRAP_CLIENT_SECRET" "REPLACE_WITH_CASDOOR_BOOTSTRAP_CLIENT_SECRET"
ensure_bootstrap_env_value "${casdoor_bootstrap_env_file}" "CASDOOR_BOOTSTRAP_APPLICATION" "REPLACE_WITH_CASDOOR_BOOTSTRAP_APPLICATION"
ensure_bootstrap_env_value "${casdoor_bootstrap_env_file}" "CASDOOR_BOOTSTRAP_CERTIFICATE" ""
ensure_prod_default "WEB_PUBLIC_URL" "${WEB_PUBLIC_URL:-}" "https://stuhelper.com" "REPLACE_WITH_WEB_PUBLIC_URL" "http://localhost:3000"
ensure_prod_default "ADMIN_PUBLIC_URL" "${ADMIN_PUBLIC_URL:-}" "https://stuhelper.com/admin/" "REPLACE_WITH_ADMIN_PUBLIC_URL" "http://localhost:3001"
ensure_value "PUBLIC_INGRESS_CONFIG_PREFLIGHT_ENABLED" "${PUBLIC_INGRESS_CONFIG_PREFLIGHT_ENABLED:-}" "true"
ensure_value "NGINX_PUBLIC_INGRESS_PROFILE" "${NGINX_PUBLIC_INGRESS_PROFILE:-}" "stuhelper"
ensure_value "NGINX_PUBLIC_INGRESS_CONFIG_FILE" "${NGINX_PUBLIC_INGRESS_CONFIG_FILE:-}" ""
ensure_value "PUBLIC_INGRESS_PREFLIGHT_ENABLED" "${PUBLIC_INGRESS_PREFLIGHT_ENABLED:-}" "true"
ensure_prod_default "PUBLIC_INGRESS_CASDOOR_UPSTREAM_PREFLIGHT_ENABLED" "${PUBLIC_INGRESS_CASDOOR_UPSTREAM_PREFLIGHT_ENABLED:-}" "true" "false"
ensure_value "PUBLIC_INGRESS_PUBLIC_DNS_ENABLED" "${PUBLIC_INGRESS_PUBLIC_DNS_ENABLED:-}" "true"
ensure_value "PUBLIC_INGRESS_PREFLIGHT_TIMEOUT_SECONDS" "${PUBLIC_INGRESS_PREFLIGHT_TIMEOUT_SECONDS:-}" "10"
ensure_value "SSO_PUBLIC_SMOKE_ENABLED" "${SSO_PUBLIC_SMOKE_ENABLED:-}" "true"
ensure_prod_default "SSO_PUBLIC_BASE_URL" "${SSO_PUBLIC_BASE_URL:-}" "https://sso.stuhelper.com" "http://localhost:8085"
ensure_prod_default "SSO_PUBLIC_SMOKE_EXPECTED_ISSUER" "${SSO_PUBLIC_SMOKE_EXPECTED_ISSUER:-}" "https://sso.stuhelper.com" "http://localhost:8085"
ensure_value "SSO_PUBLIC_SMOKE_CLIENT_ID" "${SSO_PUBLIC_SMOKE_CLIENT_ID:-}" "stuhelper-web"
ensure_value "SSO_PUBLIC_SMOKE_APPLICATION_ID" "${SSO_PUBLIC_SMOKE_APPLICATION_ID:-}" "admin/stuhelper-web"
ensure_prod_default "SSO_PUBLIC_SMOKE_REDIRECT_URI" "${SSO_PUBLIC_SMOKE_REDIRECT_URI:-}" "https://stuhelper.com/api/v1/auth/callback" "http://localhost:8080/api/v1/auth/callback"
ensure_value "SSO_PUBLIC_SMOKE_SCOPE" "${SSO_PUBLIC_SMOKE_SCOPE:-}" "openid"
ensure_prod_default "SSO_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS" "${SSO_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS:-}" "false" "true"
ensure_prod_default "SSO_PUBLIC_SMOKE_CURL_INSECURE" "${SSO_PUBLIC_SMOKE_CURL_INSECURE:-}" "false" "true"
ensure_value "SSO_PUBLIC_SMOKE_CURL_NO_PROXY" "${SSO_PUBLIC_SMOKE_CURL_NO_PROXY:-}" "*"
ensure_value "ADMISSION_PUBLIC_SMOKE_ENABLED" "${ADMISSION_PUBLIC_SMOKE_ENABLED:-}" "true"
ensure_prod_default "ADMISSION_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS" "${ADMISSION_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS:-}" "false" "true"
ensure_prod_default "ADMISSION_PUBLIC_SMOKE_CURL_INSECURE" "${ADMISSION_PUBLIC_SMOKE_CURL_INSECURE:-}" "false" "true"
ensure_value "ADMISSION_PUBLIC_SMOKE_CURL_NO_PROXY" "${ADMISSION_PUBLIC_SMOKE_CURL_NO_PROXY:-}" "*"
ensure_value "ADMISSION_PUBLIC_SMOKE_PROBE_TOKEN" "${ADMISSION_PUBLIC_SMOKE_PROBE_TOKEN:-}" "__stuhelper_public_smoke__"
ensure_value "PUBLIC_WEB_AUTH_BROWSER_SMOKE_ENABLED" "${PUBLIC_WEB_AUTH_BROWSER_SMOKE_ENABLED:-}" "true"
ensure_prod_default "PUBLIC_WEB_AUTH_BROWSER_SMOKE_ALLOW_LOCAL_TARGETS" "${PUBLIC_WEB_AUTH_BROWSER_SMOKE_ALLOW_LOCAL_TARGETS:-}" "false" "true"
ensure_value "PUBLIC_WEB_AUTH_BROWSER_SMOKE_PROBE_TOKEN" "${PUBLIC_WEB_AUTH_BROWSER_SMOKE_PROBE_TOKEN:-}" "__stuhelper_browser_smoke__"
ensure_value "PUBLIC_WEB_AUTH_BROWSER_SMOKE_HEADLESS" "${PUBLIC_WEB_AUTH_BROWSER_SMOKE_HEADLESS:-}" "true"
ensure_value "PUBLIC_WEB_AUTH_BROWSER_SMOKE_TIMEOUT_MS" "${PUBLIC_WEB_AUTH_BROWSER_SMOKE_TIMEOUT_MS:-}" "30000"
ensure_prod_default "PUBLIC_WEB_AUTH_BROWSER_EXECUTABLE_PATH" "${PUBLIC_WEB_AUTH_BROWSER_EXECUTABLE_PATH:-}" "" "/usr/bin/chromium-browser" "/usr/bin/google-chrome"
ensure_prod_default "WEB_VITE_API_URL" "${WEB_VITE_API_URL:-}" "/api" ""
ensure_prod_default "WEB_VITE_SSO_URL" "${WEB_VITE_SSO_URL:-}" "https://sso.stuhelper.com" "REPLACE_WITH_WEB_VITE_SSO_URL" "http://localhost:8085" "http://localhost"
ensure_prod_default "WEB_VITE_WEB_URL" "${WEB_VITE_WEB_URL:-}" "https://stuhelper.com" "REPLACE_WITH_WEB_VITE_WEB_URL" "http://localhost:3000"
ensure_value "WEB_VITE_API_TIMEOUT_MS" "${WEB_VITE_API_TIMEOUT_MS:-}" "15000"
ensure_prod_default "WEB_VITE_QQ_BOT_ENTRY" "${WEB_VITE_QQ_BOT_ENTRY:-}" "" "StuHelper QQ Bot"
ensure_value "WEB_VITE_QQ_BIND_COMMAND" "${WEB_VITE_QQ_BIND_COMMAND:-}" "绑定"
ensure_value "ADMIN_VITE_API_URL" "${ADMIN_VITE_API_URL:-}" "/api/v1"
ensure_value "ADMIN_VITE_BASE" "${ADMIN_VITE_BASE:-}" "/admin/"
ensure_prod_default "BACKEND_EXTERNAL_PORT" "${BACKEND_EXTERNAL_PORT:-}" "18080" "8080"
ensure_prod_default "WEB_EXTERNAL_PORT" "${WEB_EXTERNAL_PORT:-}" "18000" "3000"
ensure_prod_default "ADMIN_EXTERNAL_PORT" "${ADMIN_EXTERNAL_PORT:-}" "18001" "3001"
ensure_prod_default "OPENFGA_API_URL" "${OPENFGA_API_URL:-}" "http://openfga:8080" "http://localhost:8081"
ensure_prod_default "OPENFGA_RESOURCE_SMOKE_MODE" "${OPENFGA_RESOURCE_SMOKE_MODE:-}" "container" "host"
ensure_prod_default "OBJECT_STORAGE_ENDPOINT" "${OBJECT_STORAGE_ENDPOINT:-}" "REPLACE_WITH_OBJECT_STORAGE_ENDPOINT" "http://localhost:9000" "http://object-storage:8333"
ensure_value "OBJECT_STORAGE_REGION" "${OBJECT_STORAGE_REGION:-}" "us-east-1"
ensure_value "OBJECT_STORAGE_BUCKET" "${OBJECT_STORAGE_BUCKET:-}" "stuhelper-identity"
ensure_prod_default "OBJECT_STORAGE_ACCESS_KEY_ID" "${OBJECT_STORAGE_ACCESS_KEY_ID:-}" "REPLACE_WITH_OBJECT_STORAGE_ACCESS_KEY_ID" "stuhelper"
ensure_prod_default "BACKUP_OBJECT_STORAGE_ENDPOINT" "${BACKUP_OBJECT_STORAGE_ENDPOINT:-}" "REPLACE_WITH_BACKUP_OBJECT_STORAGE_ENDPOINT" "http://localhost:9000" "http://object-storage:8333"
ensure_value "BACKUP_OBJECT_STORAGE_BUCKET" "${BACKUP_OBJECT_STORAGE_BUCKET:-}" "stuhelper-postgres-backup"
ensure_value "BACKUP_OBJECT_STORAGE_PREFIX" "${BACKUP_OBJECT_STORAGE_PREFIX:-}" "postgres"
ensure_prod_default "BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID" "${BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID:-}" "REPLACE_WITH_BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID" "stuhelper-backup"
ensure_prod_default "BACKUP_OBJECT_STORAGE_TLS_INSECURE" "${BACKUP_OBJECT_STORAGE_TLS_INSECURE:-}" "false" "true"
ensure_prod_default "OBJECT_STORAGE_USE_SSL" "${OBJECT_STORAGE_USE_SSL:-}" "true" "false"
ensure_prod_default "OBJECT_STORAGE_FORCE_PATH_STYLE" "${OBJECT_STORAGE_FORCE_PATH_STYLE:-}" "false" "true"
ensure_value "OBJECT_STORAGE_PRESIGN_TTL" "${OBJECT_STORAGE_PRESIGN_TTL:-}" "600"
ensure_value "OBJECT_STORAGE_TLS_CA_HOST_PATH" "${OBJECT_STORAGE_TLS_CA_HOST_PATH:-}" ""
ensure_value "PROMETHEUS_RETENTION_TIME" "${PROMETHEUS_RETENTION_TIME:-}" "15d"
ensure_value "PROMETHEUS_RETENTION_SIZE" "${PROMETHEUS_RETENTION_SIZE:-}" "20GB"
ensure_value "BACKUP_LOGICAL_RETENTION_DAYS" "${BACKUP_LOGICAL_RETENTION_DAYS:-}" "14"
ensure_value "BACKUP_BASE_RETENTION_DAYS" "${BACKUP_BASE_RETENTION_DAYS:-}" "30"
ensure_value "WAL_ARCHIVE_RETENTION_DAYS" "${WAL_ARCHIVE_RETENTION_DAYS:-}" "14"
ensure_prod_default "GRAFANA_ROOT_URL" "${GRAFANA_ROOT_URL:-}" "REPLACE_WITH_GRAFANA_ROOT_URL" "http://localhost:3003"
ensure_prod_default "ALLOW_LOCAL_ALERT_SINK" "${ALLOW_LOCAL_ALERT_SINK:-}" "false" "true"
ensure_prod_default "ALERTMANAGER_WEBHOOK_URL" "${ALERTMANAGER_WEBHOOK_URL:-}" "REPLACE_WITH_ALERTMANAGER_WEBHOOK_URL" "http://alert-webhook-sink:8080/alerts"
ensure_prod_default "TAG" "${TAG:-}" "" "latest"
ensure_managed_runtime_image "POSTGRES_IMAGE_REF" "cgr.dev/chainguard/postgres:latest@sha256:dc2f04037c1044a22af76cee4de70b9111885b17c561b939d7ed70103d100759" "postgres:18.3-alpine@sha256:54451ecb8ab38c24c3ec123f2fd501303a3a1856a5c66e98cecf2460d5e1e9d7"
ensure_managed_runtime_image "REDIS_IMAGE_REF" "redis:8.8.1-alpine@sha256:8096655e437712b07503796fb64d81359256cfcff0ab29d95a7da72863786efb" "redis:8.6.2-alpine@sha256:c5e375abb885e6b2021c0377879e4890bf76f9065b8922ffc113f2b226b9fc17"
ensure_managed_runtime_image "RCLONE_IMAGE_REF" "rclone/rclone:beta@sha256:f52965eba611ba8984117638b2a0539dcce170731937f93fbace66897d102698"
ensure_managed_runtime_image "GOLANG_IMAGE_REF" "cgr.dev/chainguard/go:latest@sha256:b116b5f2d3f5e7556b66252f9ee7ef9988b84c2139c89d824efcebd6cadbf436" "golang:1.26.5-bookworm@sha256:1ecb7edf62a0408027bd5729dfd6b1b8766e578e8df93995b225dfd0944eb651"
ensure_managed_runtime_image "OPENFGA_IMAGE_REF" "openfga/openfga:v1.18.1@sha256:efde89d24487da1a8bc37d85b61341f1fb7024943a1ded65f4b7d51a75666688" "openfga/openfga:v1.8.12@sha256:e2c06000981774a7a02d8aa2292df800ea80a8bc4df61a7ed5f098738b53b1ff"
ensure_managed_runtime_image "DOCKER_SOCKET_PROXY_IMAGE_REF" "ghcr.io/tecnativa/docker-socket-proxy:latest@sha256:1f5038b54f06c3e18422902cf00ba21803d1c97805aae032e5e6673d532d3459"
ensure_managed_runtime_image "GRAFANA_ALLOY_IMAGE_REF" "grafana/alloy:v1.18.0@sha256:491b0578c04983fd54fe99b587b6fab4404dc46d0dc16677bd6b00cc1140b308" "grafana/alloy:v1.10.2@sha256:bcf27f18c4402869af112fb39e35e1db3804a404686f4caa20bdf77814219223"
ensure_managed_runtime_image "PROMETHEUS_IMAGE_REF" "prom/prometheus:v3.13.1@sha256:3c42b892cf723fa54d2f262c37a0e1f80aa8c8ddb1da7b9b0df9455a35a7f893" "prom/prometheus:v3.5.0@sha256:63805ebb8d2b3920190daf1cb14a60871b16fd38bed42b857a3182bc621f4996"
ensure_managed_runtime_image "ALERTMANAGER_IMAGE_REF" "prom/alertmanager:v0.33.1@sha256:9e082985f56f4c8c9f724e18f2288c6708f472e56a5286b8863d080434ea065d" "prom/alertmanager:v0.28.1@sha256:27c475db5fb156cab31d5c18a4251ac7ed567746a2483ff264516437a39b15ba"
ensure_managed_runtime_image "LOKI_IMAGE_REF" "grafana/loki:3.7.4@sha256:87f0a067673756a3cede1bcbf0c74875f7df9b09fddb53e399d0c576f756cfcc" "grafana/loki:3.5.5@sha256:31628519045e7f28692a7ae73b4a3fd293dccb425585ed5d16ceea3b5c9592e6"
ensure_managed_runtime_image "TEMPO_IMAGE_REF" "grafana/tempo:3.0.2@sha256:cda87c212d8c584dc0b89e337e7ed648a5100feb657e5d528480ee4fa03dbbe3" "grafana/tempo:2.8.2@sha256:0ef775495967cd5d7a6b2e146b6ea695d624803c8db8349fb8ce4164f719f9b7"
ensure_managed_runtime_image "GRAFANA_IMAGE_REF" "grafana/grafana:nightly-slim@sha256:5909f8f4123b9ff3efcd701e23c0b5310b6ae0ea12fd3ee906f2bc91831e5363" "grafana/grafana:12.1.1@sha256:a1701c2180249361737a99a01bc770db39381640e4d631825d38ff4535efa47d"
ensure_managed_runtime_image "NODE_EXPORTER_IMAGE_REF" "prom/node-exporter:v1.12.1@sha256:1b4e4438faca4dd7e001dd445d161a4a2091b0fededa84093b3a8dfeae1f1be0" "prom/node-exporter:v1.9.1@sha256:d00a542e409ee618a4edc67da14dd48c5da66726bbd5537ab2af9c1dfc442c8a"
ensure_managed_runtime_image "CADVISOR_IMAGE_REF" "ghcr.io/google/cadvisor:v0.60.5@sha256:763aecf1c32c2be8a1a75f9abfc2fc461005c9dbbaa39cb356b354aac1296dbe" "gcr.io/cadvisor/cadvisor:v0.52.1@sha256:f40e65878e25c2e78ea037f73a449527a0fb994e303dc3e34cb6b187b4b91435"
ensure_managed_runtime_image "POSTGRES_EXPORTER_IMAGE_REF" "quay.io/prometheuscommunity/postgres-exporter:v0.20.1@sha256:ac5ec343104fae0e2d84a27bb8d69b38430a11910c5382cad85d478d2bab713e" "quay.io/prometheuscommunity/postgres-exporter:v0.17.1@sha256:38606faa38c54787525fb0ff2fd6b41b4cfb75d455c1df294927c5f611699b17"
ensure_managed_runtime_image "REDIS_EXPORTER_IMAGE_REF" "oliver006/redis_exporter:v1.88.0@sha256:2c8c55c63ce4d915389f03d337b8acef56aaaca9fab8728291287e612d4d6398" "oliver006/redis_exporter:v1.76.0@sha256:1542bc6a88decfc16db6603045accd502cc3a46c46659d7cfd568e1f6965fe59"
ensure_managed_runtime_image "BLACKBOX_EXPORTER_IMAGE_REF" "quay.io/prometheus/blackbox-exporter:master@sha256:9a7db82eecc48c8f226a24ca72c7b367b749b7994881824aa9b6a05b24ff4579" "quay.io/prometheus/blackbox-exporter:v0.27.0@sha256:a50c4c0eda297baa1678cd4dc4712a67fdea713b832d43ce7fcc5f9bea05094d"
ensure_prod_default "BACKEND_IMAGE_REF" "${BACKEND_IMAGE_REF:-}" "REPLACE_WITH_BACKEND_IMAGE_REF" "registry.stuhelper.com/stuhelper/backend:latest" "stuhelper/backend:dev-placeholder"
ensure_prod_default "FRONTEND_IMAGE_REF" "${FRONTEND_IMAGE_REF:-}" "REPLACE_WITH_FRONTEND_IMAGE_REF" "registry.stuhelper.com/stuhelper/frontend:latest" "stuhelper/frontend:dev-placeholder"
ensure_prod_default "ADMIN_IMAGE_REF" "${ADMIN_IMAGE_REF:-}" "REPLACE_WITH_ADMIN_IMAGE_REF" "registry.stuhelper.com/stuhelper/admin:latest" "stuhelper/admin:dev-placeholder"

load_env
materialize_postgres_runtime_urls
require_production_postgres_ssl
if [[ "${EXTERNAL_POSTGRES_ENABLED:-false}" != "true" ]]; then
  "${SCRIPT_DIR}/render-postgres-tls.sh"
else
  log "external PostgreSQL selected; skipping local PostgreSQL server certificate generation"
fi
"${SCRIPT_DIR}/render-redis-tls.sh"
"${SCRIPT_DIR}/render-redis-acl.sh"
"${SCRIPT_DIR}/prepare-datastore-client-cas.sh"
"${SCRIPT_DIR}/prepare-object-storage-client-ca.sh"
log "production environment file is ready: ${ENV_FILE}"
log "generated runtime file path: ${GENERATED_ENV_FILE}"
