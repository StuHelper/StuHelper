#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
INIT_SCRIPT="${REPO_ROOT}/infra/ops/init-prod-env.sh"

fail() {
  echo "[init-prod-env-contract][error] $*" >&2
  exit 1
}

assert_file_contains() {
  local file="$1"
  local pattern="$2"
  if ! grep -Eq "${pattern}" "${file}"; then
    fail "expected ${file} to contain pattern: ${pattern}"
  fi
}

assert_file_not_contains() {
  local file="$1"
  local pattern="$2"
  if grep -Eq "${pattern}" "${file}"; then
    fail "expected ${file} to not contain pattern: ${pattern}"
  fi
}

assert_file_exists() {
  local file="$1"
  if [[ ! -f "${file}" ]]; then
    fail "expected file to exist: ${file}"
  fi
}

env_value() {
  local file="$1"
  local key="$2"
  python3 - "$file" "$key" <<'PY'
from pathlib import Path
import sys

path = Path(sys.argv[1])
key = sys.argv[2]

for line in path.read_text().splitlines():
    if line.startswith(f"{key}="):
        print(line.split("=", 1)[1])
        raise SystemExit(0)

raise SystemExit(f"missing env key: {key}")
PY
}

assert_env_value() {
  local file="$1"
  local key="$2"
  local expected="$3"
  local actual
  actual="$(env_value "${file}" "${key}")"
  if [[ "${actual}" != "${expected}" ]]; then
    fail "expected ${key}=${expected}, got ${actual} in ${file}"
  fi
}

run_init_prod_env() {
  local tmpdir
  tmpdir="$(mktemp -d)"
  ENV_FILE="${tmpdir}/.env.prod.shared" \
  SECRETS_ENV_FILE="${tmpdir}/.env.prod.secrets.local" \
  GENERATED_ENV_FILE="${tmpdir}/.env.prod.generated" \
  GENERATED_SECRET_ENV_FILE="${tmpdir}/.env.prod.generated.secrets" \
  GENERATED_OBS_DIR="${tmpdir}/generated/observability" \
  DEPLOY_STATE_DIR="${tmpdir}/.deploy" \
  bash "${INIT_SCRIPT}" >"${tmpdir}/stdout.log" 2>"${tmpdir}/stderr.log"
  printf '%s\n' "${tmpdir}"
}

cleanup_dirs=()
cleanup() {
  local dir
  for dir in "${cleanup_dirs[@]:-}"; do
    rm -rf "${dir}"
  done
}
trap cleanup EXIT

fresh_dir="$(run_init_prod_env)"
cleanup_dirs+=("${fresh_dir}")
fresh_env="${fresh_dir}/.env.prod.shared"
fresh_secrets="${fresh_dir}/.env.prod.secrets.local"
fresh_bootstrap_env="${fresh_dir}/.env.casdoor-bootstrap.local"

assert_file_contains "${fresh_dir}/stdout.log" 'from \.env\.prod\.example'
assert_file_contains "${fresh_env}" '^# StuHelper 生产环境配置样板$'
assert_env_value "${fresh_env}" "CORS_ORIGINS" "https://stuhelper.com,https://join.stuhelper.com,https://sso.stuhelper.com"
assert_env_value "${fresh_env}" "ADMISSION_PUBLIC_BASE_URL" "https://join.stuhelper.com"
assert_env_value "${fresh_env}" "ADMISSION_PRODUCTION_READINESS_ENABLED" "true"
assert_env_value "${fresh_env}" "ADMISSION_READINESS_REQUIRED_PLATFORM" "qq"
assert_env_value "${fresh_env}" "ADMISSION_READINESS_REQUIRED_GUILD_IDS" "178037297"
assert_env_value "${fresh_env}" "ADMISSION_READINESS_REQUIRED_SCHOOL_CODES" "4111010006"
assert_env_value "${fresh_env}" "ADMISSION_READINESS_REQUIRED_SCHOOL_IDS" ""
assert_env_value "${fresh_env}" "ADMISSION_PUBLIC_SMOKE_ENABLED" "true"
assert_env_value "${fresh_env}" "ADMISSION_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS" "false"
assert_env_value "${fresh_env}" "ADMISSION_PUBLIC_SMOKE_CURL_INSECURE" "false"
assert_env_value "${fresh_env}" "ADMISSION_PUBLIC_SMOKE_CURL_NO_PROXY" "*"
assert_env_value "${fresh_env}" "ADMISSION_PUBLIC_SMOKE_PROBE_TOKEN" "__stuhelper_public_smoke__"
assert_env_value "${fresh_env}" "ADMISSION_PUBLIC_SMOKE_PROBE_QQ" "10000"
assert_env_value "${fresh_env}" "PUBLIC_WEB_AUTH_BROWSER_SMOKE_ENABLED" "true"
assert_env_value "${fresh_env}" "PUBLIC_WEB_AUTH_BROWSER_SMOKE_ALLOW_LOCAL_TARGETS" "false"
assert_env_value "${fresh_env}" "PUBLIC_WEB_AUTH_BROWSER_SMOKE_PROBE_TOKEN" "__stuhelper_browser_smoke__"
assert_env_value "${fresh_env}" "PUBLIC_WEB_AUTH_BROWSER_SMOKE_PROBE_QQ" "10000"
assert_env_value "${fresh_env}" "PUBLIC_WEB_AUTH_BROWSER_SMOKE_HEADLESS" "true"
assert_env_value "${fresh_env}" "PUBLIC_WEB_AUTH_BROWSER_SMOKE_TIMEOUT_MS" "30000"
assert_env_value "${fresh_env}" "PUBLIC_WEB_AUTH_BROWSER_EXECUTABLE_PATH" ""
assert_env_value "${fresh_env}" "STUHELPER_FRESHMAN_MATERIAL_HOSTS" "stuhelper.com,join.stuhelper.com"
assert_env_value "${fresh_env}" "TOKEN_COOKIE_DOMAIN" ".stuhelper.com"
assert_env_value "${fresh_env}" "CASDOOR_ISSUER" "https://sso.stuhelper.com"
assert_env_value "${fresh_env}" "CASDOOR_PUBLIC_AUTH_BASE_URL" "https://sso.stuhelper.com"
assert_env_value "${fresh_env}" "CASDOOR_REDIRECT_URI" "https://stuhelper.com/api/v1/auth/callback"
assert_env_value "${fresh_env}" "CASDOOR_CLIENT_ID" "REPLACE_WITH_CASDOOR_CLIENT_ID"
assert_env_value "${fresh_env}" "DATABASE_URL" "postgres://stuhelper_app:REPLACE_WITH_STUHELPER_APP_DB_PASSWORD@postgres:5432/stuhelper?sslmode=verify-full&sslrootcert=/tls/ca.crt"
assert_env_value "${fresh_env}" "BACKUP_DATABASE_URL" "postgres://stuhelper_backup:REPLACE_WITH_STUHELPER_BACKUP_DB_PASSWORD@postgres:5432/stuhelper?sslmode=verify-full&sslrootcert=/tls/ca.crt"
assert_env_value "${fresh_env}" "REPLICATION_DATABASE_URL" "postgres://stuhelper_replication:REPLACE_WITH_STUHELPER_REPLICATION_DB_PASSWORD@postgres:5432/stuhelper?sslmode=verify-full&sslrootcert=/tls/ca.crt"
assert_env_value "${fresh_env}" "DB_SSL_MODE" "verify-full"
assert_env_value "${fresh_env}" "DB_SSL_ROOT_CERT" "/tls/ca.crt"
assert_env_value "${fresh_env}" "POSTGRES_ENABLE_SSL" "on"
assert_env_value "${fresh_env}" "POSTGRES_INTERNAL_SSL_MODE" "verify-full"
assert_env_value "${fresh_env}" "EXTERNAL_POSTGRES_ENABLED" "false"
assert_env_value "${fresh_env}" "EXTERNAL_POSTGRES_ALLOW_PLAINTEXT" "false"
assert_env_value "${fresh_env}" "EXTERNAL_DATASTORE_NETWORK" ""
assert_env_value "${fresh_env}" "REDIS_USERNAME" "stuhelper_app"
assert_env_value "${fresh_env}" "CASDOOR_BOOTSTRAP_ENABLED" "true"
assert_env_value "${fresh_env}" "CASDOOR_BOOTSTRAP_ENV_FILE" ".env.casdoor-bootstrap.local"
assert_env_value "${fresh_env}" "CASDOOR_ADMIN_CLIENT_ID" "stuhelper-admin"
assert_env_value "${fresh_env}" "CASDOOR_ADMIN_REDIRECT_URI" "https://stuhelper.com/api/v1/auth/callback"
assert_env_value "${fresh_env}" "CASDOOR_UNIAPP_CLIENT_ID" "stuhelper-uniapp"
assert_env_value "${fresh_env}" "CASDOOR_UNIAPP_REDIRECT_URI" "https://stuhelper.com/api/v1/auth/callback"
assert_env_value "${fresh_env}" "CASDOOR_SMS_PROVIDER_ENABLED" "true"
assert_env_value "${fresh_env}" "CASDOOR_SMS_PROVIDER_ENDPOINT" "http://app:8080/internal/sms/send"
assert_env_value "${fresh_env}" "CASDOOR_SMS_PROVIDER_TYPE" "CustomHTTP"
assert_env_value "${fresh_env}" "CASDOOR_SMS_PROVIDER_TITLE" "content"
assert_env_value "${fresh_env}" "SMS_ENABLED" "true"
assert_env_value "${fresh_env}" "SMS_APP_ID" "REPLACE_WITH_SMS_APP_ID"
assert_env_value "${fresh_env}" "SMS_SIGN_NAME" "REPLACE_WITH_SMS_SIGN_NAME"
assert_env_value "${fresh_env}" "SMS_TEMPLATE_ID" "REPLACE_WITH_SMS_TEMPLATE_ID"
assert_env_value "${fresh_env}" "SMS_REGION" "ap-beijing"
assert_env_value "${fresh_env}" "EMAIL_ENABLED" "true"
assert_env_value "${fresh_env}" "EMAIL_DRIVER" "multi"
assert_env_value "${fresh_env}" "EMAIL_STUDENT_VERIFICATION_SUBJECT" "学生认证验证码"
assert_env_value "${fresh_env}" "EMAIL_FROM" "noreply@notify.stuhelper.com"
assert_env_value "${fresh_env}" "EMAIL_FROM_NAME" "StuHelper 系统邮件"
assert_env_value "${fresh_env}" "EMAIL_TENCENT_REGION" "ap-guangzhou"
assert_env_value "${fresh_env}" "EMAIL_TENCENT_ENDPOINT" "ses.tencentcloudapi.com"
assert_env_value "${fresh_env}" "EMAIL_TENCENT_TEMPLATE_ID" "49779"
assert_env_value "${fresh_env}" "EMAIL_TENCENT_TEMPLATE_PURPOSE" "学校邮箱认证"
assert_env_value "${fresh_env}" "EMAIL_TENCENT_TEMPLATE_SCHOOL_NAME" "北京航空航天大学"
assert_env_value "${fresh_env}" "EMAIL_TENCENT_TEMPLATE_EXPIRE_MINUTES" "5"
assert_env_value "${fresh_env}" "EMAIL_RESEND_ENDPOINT" "https://api.resend.com/emails"
assert_env_value "${fresh_env}" "EXTERNAL_STUDENT_SOURCE_ENABLED" "false"
assert_env_value "${fresh_env}" "EXTERNAL_STUDENT_SOURCE_NAME" "buaa-academic-oracle"
assert_env_value "${fresh_env}" "EXTERNAL_STUDENT_SOURCE_PROVIDER" "oracle"
assert_env_value "${fresh_env}" "EXTERNAL_STUDENT_SOURCE_SCHOOL_CODE" "4111010006"
assert_env_value "${fresh_env}" "EXTERNAL_STUDENT_SOURCE_ORACLE_PORT" "1521"
assert_env_value "${fresh_env}" "EXTERNAL_STUDENT_SOURCE_ORACLE_SERVICE_NAME" "ORCLPDB1"
assert_env_value "${fresh_env}" "EXTERNAL_STUDENT_SOURCE_ORACLE_SCHEMA" "USR_JWBIZ"
assert_env_value "${fresh_env}" "EXTERNAL_STUDENT_SOURCE_ORACLE_TABLE" "T_XS_JBXX"
assert_env_value "${fresh_env}" "EXTERNAL_STUDENT_SOURCE_ORACLE_STUDENT_ID_COLUMN" "XH"
assert_env_value "${fresh_env}" "EXTERNAL_STUDENT_SOURCE_ORACLE_STUDENT_NAME_COLUMN" "XM"
assert_env_value "${fresh_env}" "EXTERNAL_STUDENT_SOURCE_ORACLE_CONNECT_TIMEOUT_SECONDS" "5"
assert_env_value "${fresh_env}" "EXTERNAL_STUDENT_SOURCE_ORACLE_QUERY_TIMEOUT_SECONDS" "3"
assert_env_value "${fresh_env}" "EXTERNAL_STUDENT_SOURCE_ORACLE_MAX_OPEN_CONNS" "4"
assert_env_value "${fresh_env}" "EXTERNAL_STUDENT_SOURCE_ORACLE_MAX_IDLE_CONNS" "1"
assert_env_value "${fresh_env}" "EXTERNAL_STUDENT_SOURCE_SMOKE_MODE" "container"
assert_env_value "${fresh_env}" "EXTERNAL_STUDENT_SOURCE_SMOKE_COMMAND" "/app/external-student-source-smoke"
assert_env_value "${fresh_env}" "EXTERNAL_STUDENT_SOURCE_SMOKE_TIMEOUT_SECONDS" "15"
assert_env_value "${fresh_env}" "EXTERNAL_STUDENT_SOURCE_SMOKE_REQUIRE_READABLE_RECORD" "true"
assert_env_value "${fresh_env}" "EXTERNAL_STUDENT_SOURCE_SMOKE_REQUIRE_SAMPLE" "false"
assert_env_value "${fresh_env}" "EXTERNAL_STUDENT_SOURCE_SMOKE_EVIDENCE_FILE" "infra/generated/external-student-source-smoke.json"
assert_env_value "${fresh_secrets}" "EMAIL_TENCENT_SECRET_ID" "REPLACE_WITH_EMAIL_TENCENT_SECRET_ID"
assert_env_value "${fresh_secrets}" "EMAIL_TENCENT_SECRET_KEY" "REPLACE_WITH_EMAIL_TENCENT_SECRET_KEY"
assert_env_value "${fresh_secrets}" "EMAIL_RESEND_API_KEY" "REPLACE_WITH_EMAIL_RESEND_API_KEY"
assert_env_value "${fresh_secrets}" "EXTERNAL_STUDENT_SOURCE_ORACLE_HOST" "REPLACE_WITH_EXTERNAL_STUDENT_SOURCE_ORACLE_HOST"
assert_env_value "${fresh_secrets}" "EXTERNAL_STUDENT_SOURCE_ORACLE_USERNAME" "REPLACE_WITH_EXTERNAL_STUDENT_SOURCE_ORACLE_USERNAME"
assert_env_value "${fresh_secrets}" "EXTERNAL_STUDENT_SOURCE_ORACLE_PASSWORD" "REPLACE_WITH_EXTERNAL_STUDENT_SOURCE_ORACLE_PASSWORD"
assert_env_value "${fresh_env}" "CASDOOR_APP_PROVISIONING_CLIENT_ID" "casdoor-admin-app-provisioning"
assert_env_value "${fresh_env}" "CASDOOR_APP_PROVISIONING_APPLICATION" "casdoor-admin-app-provisioning"
assert_env_value "${fresh_env}" "CASDOOR_USER_PROFILE_CLIENT_ID" "casdoor-admin-user-profile"
assert_env_value "${fresh_env}" "CASDOOR_USER_PROFILE_APPLICATION" "casdoor-admin-user-profile"
assert_env_value "${fresh_env}" "CASDOOR_INTROSPECTION_CLIENT_ID" "casdoor-token-introspection"
assert_env_value "${fresh_env}" "CASDOOR_INTROSPECTION_APPLICATION" "casdoor-token-introspection"
assert_env_value "${fresh_env}" "CASDOOR_ROLE_SYNC_CLIENT_ID" "casdoor-admin-role-sync"
assert_env_value "${fresh_env}" "CASDOOR_ROLE_SYNC_APPLICATION" "casdoor-admin-role-sync"
assert_env_value "${fresh_env}" "CASDOOR_USER_LOOKUP_CLIENT_ID" "casdoor-admin-user-lookup"
assert_env_value "${fresh_env}" "CASDOOR_USER_LOOKUP_APPLICATION" "casdoor-admin-user-lookup"
assert_env_value "${fresh_env}" "CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_ID" "casdoor-token-probe-smoke"
assert_env_value "${fresh_env}" "CASDOOR_TOKEN_PROBE_SMOKE_APPLICATION" "casdoor-token-probe-smoke"
assert_env_value "${fresh_env}" "CASDOOR_TOKEN_PROBE_SMOKE_REDIRECT_URI" "https://stuhelper.com/open-platform/token-probe/callback"
assert_env_value "${fresh_env}" "OPEN_PLATFORM_DISCLOSURE_APP_LIMIT" "600"
assert_env_value "${fresh_env}" "OPEN_PLATFORM_DISCLOSURE_APP_USER_LIMIT" "120"
assert_env_value "${fresh_env}" "OPEN_PLATFORM_DISCLOSURE_ENDPOINT_LIMIT" "1200"
assert_env_value "${fresh_env}" "OPEN_PLATFORM_DISCLOSURE_CONSENT_LIMIT" "20"
assert_env_value "${fresh_env}" "OPEN_PLATFORM_DISCLOSURE_REPLAY_LIMIT" "8"
assert_env_value "${fresh_env}" "OPEN_PLATFORM_DISCLOSURE_WINDOW_SECONDS" "60"
assert_env_value "${fresh_env}" "OPEN_PLATFORM_DISCLOSURE_REPLAY_WINDOW_SECONDS" "300"
assert_env_value "${fresh_env}" "OPEN_PLATFORM_DISCLOSURE_REPLAY_AUDIT_COOLDOWN_SECONDS" "600"
assert_env_value "${fresh_env}" "OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_REQUIRED" "true"
assert_env_value "${fresh_env}" "OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND" "/app/casdoor-runtime-token-probe-runner.mjs"
assert_env_value "${fresh_env}" "OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_TIMEOUT_SECONDS" "30"
assert_env_value "${fresh_env}" "OPEN_PLATFORM_PRODUCTION_EVIDENCE_ALLOW_LOCAL_TARGETS" "false"
assert_env_value "${fresh_env}" "CASDOOR_TOKEN_PROBE_BROWSER_HEADLESS" "true"
assert_env_value "${fresh_env}" "CASDOOR_TOKEN_PROBE_BROWSER_EXECUTABLE_PATH" "/usr/bin/chromium-browser"
assert_env_value "${fresh_env}" "CASDOOR_TOKEN_PROBE_BROWSER_NO_SANDBOX" "true"
assert_env_value "${fresh_env}" "CASDOOR_TOKEN_PROBE_LOGIN_TIMEOUT_SECONDS" "30"
assert_env_value "${fresh_env}" "WEB_PUBLIC_URL" "https://stuhelper.com"
assert_env_value "${fresh_env}" "ADMIN_PUBLIC_URL" "https://stuhelper.com/admin/"
assert_env_value "${fresh_env}" "PUBLIC_INGRESS_CONFIG_PREFLIGHT_ENABLED" "true"
assert_env_value "${fresh_env}" "NGINX_PUBLIC_INGRESS_PROFILE" "stuhelper"
assert_env_value "${fresh_env}" "NGINX_PUBLIC_INGRESS_CONFIG_FILE" ""
assert_env_value "${fresh_env}" "PUBLIC_INGRESS_PREFLIGHT_ENABLED" "true"
assert_env_value "${fresh_env}" "PUBLIC_INGRESS_CASDOOR_UPSTREAM_PREFLIGHT_ENABLED" "true"
assert_env_value "${fresh_env}" "PUBLIC_INGRESS_PUBLIC_DNS_ENABLED" "true"
assert_env_value "${fresh_env}" "PUBLIC_INGRESS_PREFLIGHT_TIMEOUT_SECONDS" "10"
assert_env_value "${fresh_env}" "SSO_PUBLIC_SMOKE_ENABLED" "true"
assert_env_value "${fresh_env}" "SSO_PUBLIC_BASE_URL" "https://sso.stuhelper.com"
assert_env_value "${fresh_env}" "SSO_PUBLIC_SMOKE_EXPECTED_ISSUER" "https://sso.stuhelper.com"
assert_env_value "${fresh_env}" "SSO_PUBLIC_SMOKE_CLIENT_ID" "stuhelper-web"
assert_env_value "${fresh_env}" "SSO_PUBLIC_SMOKE_APPLICATION_ID" "admin/stuhelper-web"
assert_env_value "${fresh_env}" "SSO_PUBLIC_SMOKE_REDIRECT_URI" "https://stuhelper.com/api/v1/auth/callback"
assert_env_value "${fresh_env}" "SSO_PUBLIC_SMOKE_SCOPE" "openid"
assert_env_value "${fresh_env}" "SSO_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS" "false"
assert_env_value "${fresh_env}" "SSO_PUBLIC_SMOKE_CURL_INSECURE" "false"
assert_env_value "${fresh_env}" "SSO_PUBLIC_SMOKE_CURL_NO_PROXY" "*"
assert_env_value "${fresh_env}" "WEB_VITE_API_URL" "/api"
assert_env_value "${fresh_env}" "WEB_VITE_SSO_URL" "https://sso.stuhelper.com"
assert_env_value "${fresh_env}" "WEB_VITE_IDENTITY_URL" ""
assert_env_value "${fresh_env}" "WEB_VITE_WEB_URL" "https://stuhelper.com"
assert_env_value "${fresh_env}" "BACKEND_EXTERNAL_PORT" "18080"
assert_env_value "${fresh_env}" "WEB_EXTERNAL_PORT" "18000"
assert_env_value "${fresh_env}" "ADMIN_EXTERNAL_PORT" "18001"
assert_env_value "${fresh_env}" "OPENFGA_API_URL" "http://openfga:8080"
assert_env_value "${fresh_env}" "OPENFGA_RESOURCE_SMOKE_MODE" "container"
assert_env_value "${fresh_env}" "OBJECT_STORAGE_ENDPOINT" "REPLACE_WITH_OBJECT_STORAGE_ENDPOINT"
assert_env_value "${fresh_env}" "OBJECT_STORAGE_USE_SSL" "true"
assert_env_value "${fresh_env}" "OBJECT_STORAGE_FORCE_PATH_STYLE" "false"
assert_env_value "${fresh_env}" "OBJECT_STORAGE_TLS_CA" "/minio-tls/ca.crt"
assert_env_value "${fresh_env}" "GRAFANA_ROOT_URL" "REPLACE_WITH_GRAFANA_ROOT_URL"
assert_env_value "${fresh_env}" "ALERTMANAGER_WEBHOOK_URL" "REPLACE_WITH_ALERTMANAGER_WEBHOOK_URL"
assert_env_value "${fresh_env}" "ALLOW_LOCAL_ALERT_SINK" "false"
assert_env_value "${fresh_env}" "TAG" ""
assert_env_value "${fresh_env}" "BACKEND_IMAGE_REF" "REPLACE_WITH_BACKEND_IMAGE_REF"
assert_env_value "${fresh_env}" "FRONTEND_IMAGE_REF" "REPLACE_WITH_FRONTEND_IMAGE_REF"
assert_env_value "${fresh_env}" "ADMIN_IMAGE_REF" "REPLACE_WITH_ADMIN_IMAGE_REF"
assert_file_exists "${fresh_dir}/.env.prod.generated.secrets"
assert_file_exists "${fresh_bootstrap_env}"
assert_env_value "${fresh_bootstrap_env}" "CASDOOR_BOOTSTRAP_CLIENT_ID" "REPLACE_WITH_CASDOOR_BOOTSTRAP_CLIENT_ID"
assert_env_value "${fresh_bootstrap_env}" "CASDOOR_BOOTSTRAP_CLIENT_SECRET" "REPLACE_WITH_CASDOOR_BOOTSTRAP_CLIENT_SECRET"
assert_env_value "${fresh_bootstrap_env}" "CASDOOR_BOOTSTRAP_APPLICATION" "REPLACE_WITH_CASDOOR_BOOTSTRAP_APPLICATION"
assert_file_contains "${fresh_secrets}" '^CASDOOR_CLIENT_SECRET=prod-casdoor-web-[0-9a-f]+$'
assert_file_contains "${fresh_secrets}" '^CASDOOR_ADMIN_CLIENT_SECRET=prod-casdoor-admin-[0-9a-f]+$'
assert_file_contains "${fresh_secrets}" '^CASDOOR_UNIAPP_CLIENT_SECRET=prod-casdoor-uniapp-[0-9a-f]+$'
assert_file_contains "${fresh_secrets}" '^CASDOOR_APP_PROVISIONING_CLIENT_SECRET=prod-casdoor-app-provisioning-[0-9a-f]+$'
assert_file_contains "${fresh_secrets}" '^CASDOOR_USER_PROFILE_CLIENT_SECRET=prod-casdoor-user-profile-[0-9a-f]+$'
assert_file_contains "${fresh_secrets}" '^CASDOOR_INTROSPECTION_CLIENT_SECRET=prod-casdoor-introspection-[0-9a-f]+$'
assert_file_contains "${fresh_secrets}" '^CASDOOR_ROLE_SYNC_CLIENT_SECRET=prod-casdoor-role-sync-[0-9a-f]+$'
assert_file_contains "${fresh_secrets}" '^CASDOOR_USER_LOOKUP_CLIENT_SECRET=prod-casdoor-user-lookup-[0-9a-f]+$'
assert_file_contains "${fresh_secrets}" '^CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_SECRET=prod-casdoor-token-probe-smoke-[0-9a-f]+$'
assert_env_value "${fresh_secrets}" "CASDOOR_TOKEN_PROBE_USERNAME" "REPLACE_WITH_CASDOOR_TOKEN_PROBE_USERNAME"
assert_env_value "${fresh_secrets}" "CASDOOR_TOKEN_PROBE_PASSWORD" "REPLACE_WITH_CASDOOR_TOKEN_PROBE_PASSWORD"
assert_file_not_contains "${fresh_secrets}" '^CASDOOR_DB_PASSWORD='
assert_env_value "${fresh_secrets}" "SMS_SECRET_ID" "REPLACE_WITH_SMS_SECRET_ID"
assert_env_value "${fresh_secrets}" "SMS_SECRET_KEY" "REPLACE_WITH_SMS_SECRET_KEY"
assert_file_contains "${fresh_secrets}" '^SMS_INTERNAL_KEY=[0-9a-f]+$'
assert_file_not_contains "${fresh_env}" '^DATABASE_URL=.*@localhost:5432/.*sslmode=disable$'
assert_file_not_contains "${fresh_env}" '^CASDOOR_INTERNAL_ADDRESS=host\.docker\.internal:8085$'
assert_file_not_contains "${fresh_env}" '^ALERTMANAGER_WEBHOOK_URL=http://alert-webhook-sink:8080/alerts$'
assert_file_not_contains "${fresh_env}" '^BACKEND_IMAGE_REF=stuhelper/backend:dev-placeholder$'
assert_file_not_contains "${fresh_env}" '^FRONTEND_IMAGE_REF=stuhelper/frontend:dev-placeholder$'
assert_file_not_contains "${fresh_env}" '^ADMIN_IMAGE_REF=stuhelper/admin:dev-placeholder$'

legacy_dir="$(mktemp -d)"
cleanup_dirs+=("${legacy_dir}")
cp "${REPO_ROOT}/.env.example" "${legacy_dir}/.env.prod.shared"
retired_idp_prefix='ZITA''DEL_'
retired_pat_key='LOGIN_CLIENT_''PAT_EXPIRATION'
printf '%s\n' \
  "${retired_idp_prefix}ISSUER=http://localhost:8085" \
  "${retired_idp_prefix}CLIENT_SECRET=old-client-secret" \
  "${retired_pat_key}=2040-01-01T00:00:00Z" \
  >>"${legacy_dir}/.env.prod.shared"
printf '%s\n' "${retired_idp_prefix}MASTERKEY=old-masterkey" >"${legacy_dir}/.env.prod.secrets.local"
printf '%s\n' "${retired_idp_prefix}PROJECT_ID=old-project" >"${legacy_dir}/.env.prod.generated"
printf '%s\n' "${retired_idp_prefix}MANAGEMENT_PAT=old-pat" >"${legacy_dir}/.env.prod.generated.secrets"
ENV_FILE="${legacy_dir}/.env.prod.shared" \
SECRETS_ENV_FILE="${legacy_dir}/.env.prod.secrets.local" \
GENERATED_ENV_FILE="${legacy_dir}/.env.prod.generated" \
GENERATED_SECRET_ENV_FILE="${legacy_dir}/.env.prod.generated.secrets" \
GENERATED_OBS_DIR="${legacy_dir}/generated/observability" \
DEPLOY_STATE_DIR="${legacy_dir}/.deploy" \
bash "${INIT_SCRIPT}" >"${legacy_dir}/stdout.log" 2>"${legacy_dir}/stderr.log"

legacy_env="${legacy_dir}/.env.prod.shared"
assert_env_value "${legacy_env}" "DATABASE_URL" "postgres://stuhelper_app:REPLACE_WITH_STUHELPER_APP_DB_PASSWORD@postgres:5432/stuhelper?sslmode=verify-full&sslrootcert=/tls/ca.crt"
assert_env_value "${legacy_env}" "BACKUP_DATABASE_URL" "postgres://stuhelper_backup:REPLACE_WITH_STUHELPER_BACKUP_DB_PASSWORD@postgres:5432/stuhelper?sslmode=verify-full&sslrootcert=/tls/ca.crt"
assert_env_value "${legacy_env}" "REPLICATION_DATABASE_URL" "postgres://stuhelper_replication:REPLACE_WITH_STUHELPER_REPLICATION_DB_PASSWORD@postgres:5432/stuhelper?sslmode=verify-full&sslrootcert=/tls/ca.crt"
assert_env_value "${legacy_env}" "DB_SSL_MODE" "verify-full"
assert_env_value "${legacy_env}" "DB_SSL_ROOT_CERT" "/tls/ca.crt"
assert_env_value "${legacy_env}" "POSTGRES_ENABLE_SSL" "on"
assert_env_value "${legacy_env}" "POSTGRES_INTERNAL_SSL_MODE" "verify-full"
assert_env_value "${legacy_env}" "EXTERNAL_POSTGRES_ENABLED" "false"
assert_env_value "${legacy_env}" "EXTERNAL_POSTGRES_ALLOW_PLAINTEXT" "false"
assert_env_value "${legacy_env}" "CORS_ORIGINS" "https://stuhelper.com,https://join.stuhelper.com,https://sso.stuhelper.com"
assert_env_value "${legacy_env}" "ADMISSION_PUBLIC_BASE_URL" "https://join.stuhelper.com"
assert_env_value "${legacy_env}" "ADMISSION_PRODUCTION_READINESS_ENABLED" "true"
assert_env_value "${legacy_env}" "ADMISSION_READINESS_REQUIRED_PLATFORM" "qq"
assert_env_value "${legacy_env}" "ADMISSION_READINESS_REQUIRED_GUILD_IDS" ""
assert_env_value "${legacy_env}" "ADMISSION_READINESS_REQUIRED_SCHOOL_CODES" ""
assert_env_value "${legacy_env}" "ADMISSION_READINESS_REQUIRED_SCHOOL_IDS" ""
assert_env_value "${legacy_env}" "STUHELPER_FRESHMAN_MATERIAL_HOSTS" "stuhelper.com,join.stuhelper.com"
assert_env_value "${legacy_env}" "TOKEN_COOKIE_DOMAIN" ".stuhelper.com"
assert_env_value "${legacy_env}" "CASDOOR_ISSUER" "https://sso.stuhelper.com"
assert_env_value "${legacy_env}" "CASDOOR_INTERNAL_ADDRESS" ""
assert_env_value "${legacy_env}" "CASDOOR_PUBLIC_AUTH_BASE_URL" "https://sso.stuhelper.com"
assert_env_value "${legacy_env}" "CASDOOR_REDIRECT_URI" "https://stuhelper.com/api/v1/auth/callback"
assert_env_value "${legacy_env}" "CASDOOR_CLIENT_ID" "REPLACE_WITH_CASDOOR_CLIENT_ID"
assert_env_value "${legacy_env}" "CASDOOR_BOOTSTRAP_ENABLED" "true"
assert_env_value "${legacy_env}" "CASDOOR_BOOTSTRAP_ENV_FILE" ".env.casdoor-bootstrap.local"
assert_env_value "${legacy_env}" "CASDOOR_ADMIN_CLIENT_ID" "stuhelper-admin"
assert_env_value "${legacy_env}" "CASDOOR_ADMIN_REDIRECT_URI" "https://stuhelper.com/api/v1/auth/callback"
assert_env_value "${legacy_env}" "CASDOOR_UNIAPP_CLIENT_ID" "stuhelper-uniapp"
assert_env_value "${legacy_env}" "CASDOOR_UNIAPP_REDIRECT_URI" "https://stuhelper.com/api/v1/auth/callback"
assert_env_value "${legacy_env}" "CASDOOR_SMS_PROVIDER_TYPE" "CustomHTTP"
assert_env_value "${legacy_env}" "CASDOOR_SMS_PROVIDER_TITLE" "content"
assert_env_value "${legacy_env}" "SMS_ENABLED" "true"
assert_env_value "${legacy_env}" "SMS_APP_ID" "REPLACE_WITH_SMS_APP_ID"
assert_env_value "${legacy_env}" "SMS_SIGN_NAME" "REPLACE_WITH_SMS_SIGN_NAME"
assert_env_value "${legacy_env}" "SMS_TEMPLATE_ID" "REPLACE_WITH_SMS_TEMPLATE_ID"
assert_env_value "${legacy_env}" "SMS_REGION" "ap-beijing"
assert_env_value "${legacy_env}" "CASDOOR_APP_PROVISIONING_CLIENT_ID" "casdoor-admin-app-provisioning"
assert_env_value "${legacy_env}" "CASDOOR_APP_PROVISIONING_APPLICATION" "casdoor-admin-app-provisioning"
assert_env_value "${legacy_env}" "CASDOOR_USER_PROFILE_CLIENT_ID" "casdoor-admin-user-profile"
assert_env_value "${legacy_env}" "CASDOOR_USER_PROFILE_APPLICATION" "casdoor-admin-user-profile"
assert_env_value "${legacy_env}" "CASDOOR_INTROSPECTION_CLIENT_ID" "casdoor-token-introspection"
assert_env_value "${legacy_env}" "CASDOOR_INTROSPECTION_APPLICATION" "casdoor-token-introspection"
assert_env_value "${legacy_env}" "CASDOOR_ROLE_SYNC_CLIENT_ID" "casdoor-admin-role-sync"
assert_env_value "${legacy_env}" "CASDOOR_ROLE_SYNC_APPLICATION" "casdoor-admin-role-sync"
assert_env_value "${legacy_env}" "CASDOOR_USER_LOOKUP_CLIENT_ID" "casdoor-admin-user-lookup"
assert_env_value "${legacy_env}" "CASDOOR_USER_LOOKUP_APPLICATION" "casdoor-admin-user-lookup"
assert_env_value "${legacy_env}" "CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_ID" "casdoor-token-probe-smoke"
assert_env_value "${legacy_env}" "CASDOOR_TOKEN_PROBE_SMOKE_APPLICATION" "casdoor-token-probe-smoke"
assert_env_value "${legacy_env}" "CASDOOR_TOKEN_PROBE_SMOKE_REDIRECT_URI" "https://stuhelper.com/open-platform/token-probe/callback"
assert_env_value "${legacy_env}" "OPEN_PLATFORM_DISCLOSURE_APP_LIMIT" "600"
assert_env_value "${legacy_env}" "OPEN_PLATFORM_DISCLOSURE_APP_USER_LIMIT" "120"
assert_env_value "${legacy_env}" "OPEN_PLATFORM_DISCLOSURE_ENDPOINT_LIMIT" "1200"
assert_env_value "${legacy_env}" "OPEN_PLATFORM_DISCLOSURE_CONSENT_LIMIT" "20"
assert_env_value "${legacy_env}" "OPEN_PLATFORM_DISCLOSURE_REPLAY_LIMIT" "8"
assert_env_value "${legacy_env}" "OPEN_PLATFORM_DISCLOSURE_WINDOW_SECONDS" "60"
assert_env_value "${legacy_env}" "OPEN_PLATFORM_DISCLOSURE_REPLAY_WINDOW_SECONDS" "300"
assert_env_value "${legacy_env}" "OPEN_PLATFORM_DISCLOSURE_REPLAY_AUDIT_COOLDOWN_SECONDS" "600"
assert_env_value "${legacy_env}" "OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_REQUIRED" "true"
assert_env_value "${legacy_env}" "OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND" "/app/casdoor-runtime-token-probe-runner.mjs"
assert_env_value "${legacy_env}" "OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_TIMEOUT_SECONDS" "30"
assert_env_value "${legacy_env}" "OPEN_PLATFORM_PRODUCTION_EVIDENCE_ALLOW_LOCAL_TARGETS" "false"
assert_env_value "${legacy_env}" "CASDOOR_TOKEN_PROBE_BROWSER_HEADLESS" "true"
assert_env_value "${legacy_env}" "CASDOOR_TOKEN_PROBE_BROWSER_EXECUTABLE_PATH" "/usr/bin/chromium-browser"
assert_env_value "${legacy_env}" "CASDOOR_TOKEN_PROBE_BROWSER_NO_SANDBOX" "true"
assert_env_value "${legacy_env}" "CASDOOR_TOKEN_PROBE_LOGIN_TIMEOUT_SECONDS" "30"
assert_env_value "${legacy_env}" "WEB_PUBLIC_URL" "https://stuhelper.com"
assert_env_value "${legacy_env}" "ADMIN_PUBLIC_URL" "https://stuhelper.com/admin/"
assert_env_value "${legacy_env}" "PUBLIC_INGRESS_CONFIG_PREFLIGHT_ENABLED" "true"
assert_env_value "${legacy_env}" "NGINX_PUBLIC_INGRESS_PROFILE" "stuhelper"
assert_env_value "${legacy_env}" "NGINX_PUBLIC_INGRESS_CONFIG_FILE" ""
assert_env_value "${legacy_env}" "PUBLIC_INGRESS_PREFLIGHT_ENABLED" "true"
assert_env_value "${legacy_env}" "PUBLIC_INGRESS_CASDOOR_UPSTREAM_PREFLIGHT_ENABLED" "true"
assert_env_value "${legacy_env}" "PUBLIC_INGRESS_PUBLIC_DNS_ENABLED" "true"
assert_env_value "${legacy_env}" "PUBLIC_INGRESS_PREFLIGHT_TIMEOUT_SECONDS" "10"
assert_env_value "${legacy_env}" "SSO_PUBLIC_SMOKE_ENABLED" "true"
assert_env_value "${legacy_env}" "SSO_PUBLIC_BASE_URL" "https://sso.stuhelper.com"
assert_env_value "${legacy_env}" "SSO_PUBLIC_SMOKE_EXPECTED_ISSUER" "https://sso.stuhelper.com"
assert_env_value "${legacy_env}" "SSO_PUBLIC_SMOKE_CLIENT_ID" "stuhelper-web"
assert_env_value "${legacy_env}" "SSO_PUBLIC_SMOKE_APPLICATION_ID" "admin/stuhelper-web"
assert_env_value "${legacy_env}" "SSO_PUBLIC_SMOKE_REDIRECT_URI" "https://stuhelper.com/api/v1/auth/callback"
assert_env_value "${legacy_env}" "SSO_PUBLIC_SMOKE_SCOPE" "openid"
assert_env_value "${legacy_env}" "SSO_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS" "false"
assert_env_value "${legacy_env}" "SSO_PUBLIC_SMOKE_CURL_INSECURE" "false"
assert_env_value "${legacy_env}" "SSO_PUBLIC_SMOKE_CURL_NO_PROXY" "*"
assert_env_value "${legacy_env}" "ADMISSION_PUBLIC_SMOKE_ENABLED" "true"
assert_env_value "${legacy_env}" "ADMISSION_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS" "false"
assert_env_value "${legacy_env}" "ADMISSION_PUBLIC_SMOKE_CURL_INSECURE" "false"
assert_env_value "${legacy_env}" "ADMISSION_PUBLIC_SMOKE_CURL_NO_PROXY" "*"
assert_env_value "${legacy_env}" "ADMISSION_PUBLIC_SMOKE_PROBE_TOKEN" "__stuhelper_public_smoke__"
assert_env_value "${legacy_env}" "ADMISSION_PUBLIC_SMOKE_PROBE_QQ" "10000"
assert_env_value "${legacy_env}" "PUBLIC_WEB_AUTH_BROWSER_SMOKE_ENABLED" "true"
assert_env_value "${legacy_env}" "PUBLIC_WEB_AUTH_BROWSER_SMOKE_ALLOW_LOCAL_TARGETS" "false"
assert_env_value "${legacy_env}" "PUBLIC_WEB_AUTH_BROWSER_SMOKE_PROBE_TOKEN" "__stuhelper_browser_smoke__"
assert_env_value "${legacy_env}" "PUBLIC_WEB_AUTH_BROWSER_SMOKE_PROBE_QQ" "10000"
assert_env_value "${legacy_env}" "PUBLIC_WEB_AUTH_BROWSER_SMOKE_HEADLESS" "true"
assert_env_value "${legacy_env}" "PUBLIC_WEB_AUTH_BROWSER_SMOKE_TIMEOUT_MS" "30000"
assert_env_value "${legacy_env}" "PUBLIC_WEB_AUTH_BROWSER_EXECUTABLE_PATH" ""
assert_env_value "${legacy_env}" "WEB_VITE_API_URL" "/api"
assert_env_value "${legacy_env}" "WEB_VITE_SSO_URL" "https://sso.stuhelper.com"
assert_env_value "${legacy_env}" "WEB_VITE_IDENTITY_URL" ""
assert_env_value "${legacy_env}" "WEB_VITE_WEB_URL" "https://stuhelper.com"
assert_env_value "${legacy_env}" "BACKEND_EXTERNAL_PORT" "18080"
assert_env_value "${legacy_env}" "WEB_EXTERNAL_PORT" "18000"
assert_env_value "${legacy_env}" "ADMIN_EXTERNAL_PORT" "18001"
assert_env_value "${legacy_env}" "OPENFGA_API_URL" "http://openfga:8080"
assert_env_value "${legacy_env}" "OPENFGA_RESOURCE_SMOKE_MODE" "container"
assert_env_value "${legacy_env}" "OBJECT_STORAGE_ENDPOINT" "REPLACE_WITH_OBJECT_STORAGE_ENDPOINT"
assert_env_value "${legacy_env}" "OBJECT_STORAGE_USE_SSL" "true"
assert_env_value "${legacy_env}" "OBJECT_STORAGE_FORCE_PATH_STYLE" "false"
assert_env_value "${legacy_env}" "OBJECT_STORAGE_TLS_CA" "/minio-tls/ca.crt"
assert_env_value "${legacy_env}" "GRAFANA_ROOT_URL" "REPLACE_WITH_GRAFANA_ROOT_URL"
assert_env_value "${legacy_env}" "ALERTMANAGER_WEBHOOK_URL" "REPLACE_WITH_ALERTMANAGER_WEBHOOK_URL"
assert_env_value "${legacy_env}" "ALLOW_LOCAL_ALERT_SINK" "false"
assert_env_value "${legacy_env}" "TAG" ""
assert_env_value "${legacy_env}" "BACKEND_IMAGE_REF" "REPLACE_WITH_BACKEND_IMAGE_REF"
assert_env_value "${legacy_env}" "FRONTEND_IMAGE_REF" "REPLACE_WITH_FRONTEND_IMAGE_REF"
assert_env_value "${legacy_env}" "ADMIN_IMAGE_REF" "REPLACE_WITH_ADMIN_IMAGE_REF"
assert_file_exists "${legacy_dir}/.env.prod.generated.secrets"
assert_file_exists "${legacy_dir}/.env.casdoor-bootstrap.local"
assert_env_value "${legacy_dir}/.env.casdoor-bootstrap.local" "CASDOOR_BOOTSTRAP_CLIENT_ID" "REPLACE_WITH_CASDOOR_BOOTSTRAP_CLIENT_ID"
assert_env_value "${legacy_dir}/.env.casdoor-bootstrap.local" "CASDOOR_BOOTSTRAP_CLIENT_SECRET" "REPLACE_WITH_CASDOOR_BOOTSTRAP_CLIENT_SECRET"
assert_env_value "${legacy_dir}/.env.casdoor-bootstrap.local" "CASDOOR_BOOTSTRAP_APPLICATION" "REPLACE_WITH_CASDOOR_BOOTSTRAP_APPLICATION"
assert_file_not_contains "${legacy_env}" "^${retired_idp_prefix}"
assert_file_not_contains "${legacy_env}" "^${retired_pat_key}="
assert_file_not_contains "${legacy_dir}/.env.prod.secrets.local" "^${retired_idp_prefix}"
assert_file_not_contains "${legacy_dir}/.env.prod.generated" "^${retired_idp_prefix}"
assert_file_not_contains "${legacy_dir}/.env.prod.generated.secrets" "^${retired_idp_prefix}"

external_dir="$(mktemp -d)"
cleanup_dirs+=("${external_dir}")
cp "${REPO_ROOT}/.env.prod.example" "${external_dir}/.env.prod.shared"
python3 - "${external_dir}/.env.prod.shared" <<'PY'
from pathlib import Path
import sys

path = Path(sys.argv[1])
replacements = {
    "DATABASE_URL": "postgres://stuhelper_app:REPLACE_WITH_STUHELPER_APP_DB_PASSWORD@postgres:5432/stuhelper?sslmode=disable",
    "BACKUP_DATABASE_URL": "postgres://stuhelper_backup:REPLACE_WITH_STUHELPER_BACKUP_DB_PASSWORD@postgres:5432/stuhelper?sslmode=disable",
    "REPLICATION_DATABASE_URL": "postgres://stuhelper_replication:REPLACE_WITH_STUHELPER_REPLICATION_DB_PASSWORD@postgres:5432/stuhelper?sslmode=disable",
    "DB_SSL_MODE": "disable",
    "DB_SSL_ROOT_CERT": "",
    "POSTGRES_ENABLE_SSL": "off",
    "POSTGRES_INTERNAL_SSL_MODE": "disable",
    "EXTERNAL_POSTGRES_ENABLED": "true",
    "EXTERNAL_POSTGRES_ALLOW_PLAINTEXT": "true",
    "EXTERNAL_DATASTORE_NETWORK": "baota_net",
}
lines = []
for line in path.read_text().splitlines():
    if "=" in line and not line.startswith("#"):
        key = line.split("=", 1)[0]
        if key in replacements:
            line = f"{key}={replacements[key]}"
    lines.append(line)
path.write_text("\n".join(lines) + "\n")
PY
touch "${external_dir}/.env.prod.secrets.local" "${external_dir}/.env.prod.generated" "${external_dir}/.env.prod.generated.secrets"
ENV_FILE="${external_dir}/.env.prod.shared" \
SECRETS_ENV_FILE="${external_dir}/.env.prod.secrets.local" \
GENERATED_ENV_FILE="${external_dir}/.env.prod.generated" \
GENERATED_SECRET_ENV_FILE="${external_dir}/.env.prod.generated.secrets" \
GENERATED_OBS_DIR="${external_dir}/generated/observability" \
DEPLOY_STATE_DIR="${external_dir}/.deploy" \
bash "${INIT_SCRIPT}" >"${external_dir}/stdout.log" 2>"${external_dir}/stderr.log"
external_env="${external_dir}/.env.prod.shared"
assert_env_value "${external_env}" "DATABASE_URL" "postgres://stuhelper_app:REPLACE_WITH_STUHELPER_APP_DB_PASSWORD@postgres:5432/stuhelper?sslmode=disable"
assert_env_value "${external_env}" "DB_SSL_MODE" "disable"
assert_env_value "${external_env}" "DB_SSL_ROOT_CERT" ""
assert_env_value "${external_env}" "POSTGRES_ENABLE_SSL" "off"
assert_env_value "${external_env}" "POSTGRES_INTERNAL_SSL_MODE" "disable"
assert_env_value "${external_env}" "EXTERNAL_DATASTORE_NETWORK" "baota_net"
assert_env_value "${external_env}" "REDIS_USERNAME" "stuhelper_app"
assert_env_value "${external_env}" "REDIS_TLS_ENABLED" "true"
assert_env_value "${external_env}" "REDIS_TLS_CA" "/tls/ca.crt"

insecure_dir="$(mktemp -d)"
cleanup_dirs+=("${insecure_dir}")
cp "${REPO_ROOT}/.env.prod.example" "${insecure_dir}/.env.prod.shared"
python3 - "${insecure_dir}/.env.prod.shared" <<'PY'
from pathlib import Path
import sys

path = Path(sys.argv[1])
lines = []
for line in path.read_text().splitlines():
    if line.startswith("BACKUP_DATABASE_URL="):
        line = "BACKUP_DATABASE_URL=postgres://stuhelper_backup:REPLACE_WITH_STUHELPER_BACKUP_DB_PASSWORD@postgres:5432/stuhelper?sslmode=require&sslrootcert=/tls/ca.crt"
    lines.append(line)
path.write_text("\n".join(lines) + "\n")
PY
touch "${insecure_dir}/.env.prod.secrets.local" "${insecure_dir}/.env.prod.generated" "${insecure_dir}/.env.prod.generated.secrets"
if ENV_FILE="${insecure_dir}/.env.prod.shared" \
SECRETS_ENV_FILE="${insecure_dir}/.env.prod.secrets.local" \
GENERATED_ENV_FILE="${insecure_dir}/.env.prod.generated" \
GENERATED_SECRET_ENV_FILE="${insecure_dir}/.env.prod.generated.secrets" \
GENERATED_OBS_DIR="${insecure_dir}/generated/observability" \
DEPLOY_STATE_DIR="${insecure_dir}/.deploy" \
bash "${INIT_SCRIPT}" >"${insecure_dir}/stdout.log" 2>"${insecure_dir}/stderr.log"; then
  fail "expected init-prod-env.sh to reject production BACKUP_DATABASE_URL with sslmode=require"
fi
assert_file_contains "${insecure_dir}/stderr.log" 'BACKUP_DATABASE_URL must include sslmode=verify-ca or sslmode=verify-full'

echo "[init-prod-env-contract] all assertions passed"
