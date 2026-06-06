#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
INIT_SCRIPT="${REPO_ROOT}/infra/ops/init-dev-env.sh"

fail() {
  echo "[init-dev-env-contract][error] $*" >&2
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

cleanup_dirs=()
cleanup() {
  local dir
  for dir in "${cleanup_dirs[@]:-}"; do
    rm -rf "${dir}"
  done
}
trap cleanup EXIT

tmpdir="$(mktemp -d)"
cleanup_dirs+=("${tmpdir}")

ENV_FILE="${tmpdir}/.env" \
GENERATED_ENV_FILE="${tmpdir}/.env.generated" \
GENERATED_SECRET_ENV_FILE="${tmpdir}/.env.generated.secrets" \
GENERATED_OBS_DIR="${tmpdir}/generated/observability" \
bash "${INIT_SCRIPT}" >"${tmpdir}/stdout.log" 2>"${tmpdir}/stderr.log"

env_file="${tmpdir}/.env"
assert_file_not_contains "${env_file}" '^TRAEFIK_'
assert_env_value "${env_file}" "STACK_NAME" "stuhelper-dev"
assert_env_value "${env_file}" "COMPOSE_PROJECT_NAME" "stuhelper-dev"
assert_env_value "${env_file}" "APP_ENV" "development"
assert_env_value "${env_file}" "CORS_ORIGINS" "http://localhost:3000,http://127.0.0.1:3000,http://localhost:3001,http://127.0.0.1:3001"
assert_env_value "${env_file}" "ADMISSION_PUBLIC_BASE_URL" "http://localhost:3000"
assert_env_value "${env_file}" "BACKEND_EXTERNAL_PORT" "8080"
assert_env_value "${env_file}" "WEB_EXTERNAL_PORT" "3000"
assert_env_value "${env_file}" "ADMIN_EXTERNAL_PORT" "3001"
assert_env_value "${env_file}" "POSTGRES_EXTERNAL_PORT" "5432"
assert_env_value "${env_file}" "REDIS_EXTERNAL_PORT" "6379"
assert_env_value "${env_file}" "OPENFGA_HTTP_EXTERNAL_PORT" "8081"
assert_env_value "${env_file}" "OPENFGA_GRPC_EXTERNAL_PORT" "8082"
assert_env_value "${env_file}" "OPENFGA_PLAYGROUND_EXTERNAL_PORT" "3002"
assert_env_value "${env_file}" "SMS_ENABLED" "false"
assert_env_value "${env_file}" "EMAIL_ENABLED" "false"
assert_env_value "${env_file}" "EMAIL_DRIVER" "smtp"
assert_env_value "${env_file}" "EMAIL_STUDENT_VERIFICATION_SUBJECT" "学生认证验证码"
assert_env_value "${env_file}" "EMAIL_FROM_NAME" "StuHelper 系统邮件"
assert_env_value "${env_file}" "EMAIL_TENCENT_REGION" "ap-guangzhou"
assert_env_value "${env_file}" "EMAIL_TENCENT_ENDPOINT" "ses.tencentcloudapi.com"
assert_env_value "${env_file}" "EMAIL_TENCENT_TEMPLATE_PURPOSE" "学校邮箱认证"
assert_env_value "${env_file}" "EMAIL_TENCENT_TEMPLATE_SCHOOL_NAME" "北京航空航天大学"
assert_env_value "${env_file}" "EMAIL_TENCENT_TEMPLATE_EXPIRE_MINUTES" "5"
assert_env_value "${env_file}" "EMAIL_RESEND_ENDPOINT" "https://api.resend.com/emails"
assert_env_value "${env_file}" "WEB_VITE_API_URL" "/api"
assert_env_value "${env_file}" "WEB_VITE_WEB_URL" "http://localhost:3000"
assert_env_value "${env_file}" "API_IP_RATE_LIMIT" "5000"
assert_env_value "${env_file}" "API_GLOBAL_RATE_LIMIT" "50000"
assert_env_value "${env_file}" "REVIEW_RATE_POST_LIMIT" "500"
assert_env_value "${env_file}" "REVIEW_RATE_SEARCH_USER_LIMIT" "500"
assert_env_value "${env_file}" "OPEN_PLATFORM_CONSENT_BASE_URL" ""
assert_env_value "${env_file}" "OPEN_PLATFORM_ACCOUNT_BASE_URL" ""
assert_env_value "${env_file}" "OPEN_PLATFORM_DISCLOSURE_APP_LIMIT" "600"
assert_env_value "${env_file}" "OPEN_PLATFORM_DISCLOSURE_APP_USER_LIMIT" "120"
assert_env_value "${env_file}" "OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_REQUIRED" "false"
assert_env_value "${env_file}" "OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND" ""
assert_env_value "${env_file}" "OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_TIMEOUT_SECONDS" "30"
assert_env_value "${env_file}" "CASDOOR_TOKEN_PROBE_BROWSER_HEADLESS" "true"
assert_env_value "${env_file}" "CASDOOR_TOKEN_PROBE_BROWSER_NO_SANDBOX" "true"
assert_env_value "${env_file}" "MINIO_API_EXTERNAL_PORT" "9000"
assert_env_value "${env_file}" "MINIO_CONSOLE_EXTERNAL_PORT" "9001"
assert_env_value "${env_file}" "CASDOOR_USER_PROFILE_CLIENT_ID" "casdoor-admin-user-profile"
assert_env_value "${env_file}" "CASDOOR_USER_PROFILE_APPLICATION" "casdoor-admin-user-profile"
assert_env_value "${env_file}" "CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_ID" "casdoor-token-probe-smoke"
assert_file_contains "${env_file}" '^CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_SECRET=dev-casdoor-token-probe-smoke-[0-9a-f]+$'
assert_env_value "${env_file}" "CASDOOR_TOKEN_PROBE_SMOKE_APPLICATION" "casdoor-token-probe-smoke"
assert_env_value "${env_file}" "CASDOOR_TOKEN_PROBE_SMOKE_REDIRECT_URI" "http://localhost:3000/open-platform/token-probe/callback"
assert_env_value "${env_file}" "OPENFGA_RESOURCE_SMOKE_MODE" "host"

legacy_dir="$(mktemp -d)"
cleanup_dirs+=("${legacy_dir}")
cp "${REPO_ROOT}/.env.example" "${legacy_dir}/.env"
python3 - "${legacy_dir}/.env" <<'PY'
from pathlib import Path
import sys

path = Path(sys.argv[1])
lines = []
for line in path.read_text().splitlines():
    if line.startswith("CORS_ORIGINS="):
        lines.append("CORS_ORIGINS=http://localhost:3000,http://localhost:3001")
    elif line.startswith("API_IP_RATE_LIMIT="):
        lines.append("API_IP_RATE_LIMIT=100")
    elif line.startswith("API_GLOBAL_RATE_LIMIT="):
        lines.append("API_GLOBAL_RATE_LIMIT=10000")
    elif line.startswith("REVIEW_RATE_POST_LIMIT="):
        lines.append("REVIEW_RATE_POST_LIMIT=5")
    elif line.startswith("REVIEW_RATE_VOTE_LIMIT="):
        lines.append("REVIEW_RATE_VOTE_LIMIT=30")
    elif line.startswith("REVIEW_RATE_REPORT_LIMIT="):
        lines.append("REVIEW_RATE_REPORT_LIMIT=10")
    elif line.startswith("REVIEW_RATE_REPLY_LIMIT="):
        lines.append("REVIEW_RATE_REPLY_LIMIT=10")
    elif line.startswith("REVIEW_RATE_WRITE_LIMIT="):
        lines.append("REVIEW_RATE_WRITE_LIMIT=10")
    elif line.startswith("WEB_VITE_API_URL="):
        lines.append("WEB_VITE_API_URL=")
    else:
        lines.append(line)
path.write_text("\n".join(lines) + "\n")
PY

ENV_FILE="${legacy_dir}/.env" \
GENERATED_ENV_FILE="${legacy_dir}/.env.generated" \
GENERATED_SECRET_ENV_FILE="${legacy_dir}/.env.generated.secrets" \
GENERATED_OBS_DIR="${legacy_dir}/generated/observability" \
bash "${INIT_SCRIPT}" >"${legacy_dir}/stdout.log" 2>"${legacy_dir}/stderr.log"

legacy_env="${legacy_dir}/.env"
assert_file_not_contains "${legacy_env}" '^TRAEFIK_'
assert_env_value "${legacy_env}" "STACK_NAME" "stuhelper-dev"
assert_env_value "${legacy_env}" "COMPOSE_PROJECT_NAME" "stuhelper-dev"
assert_env_value "${legacy_env}" "APP_ENV" "development"
assert_env_value "${legacy_env}" "CORS_ORIGINS" "http://localhost:3000,http://127.0.0.1:3000,http://localhost:3001,http://127.0.0.1:3001"
assert_env_value "${legacy_env}" "ADMISSION_PUBLIC_BASE_URL" "http://localhost:3000"
assert_env_value "${legacy_env}" "BACKEND_EXTERNAL_PORT" "8080"
assert_env_value "${legacy_env}" "WEB_EXTERNAL_PORT" "3000"
assert_env_value "${legacy_env}" "ADMIN_EXTERNAL_PORT" "3001"
assert_env_value "${legacy_env}" "POSTGRES_EXTERNAL_PORT" "5432"
assert_env_value "${legacy_env}" "REDIS_EXTERNAL_PORT" "6379"
assert_env_value "${legacy_env}" "OPENFGA_HTTP_EXTERNAL_PORT" "8081"
assert_env_value "${legacy_env}" "OPENFGA_GRPC_EXTERNAL_PORT" "8082"
assert_env_value "${legacy_env}" "OPENFGA_PLAYGROUND_EXTERNAL_PORT" "3002"
assert_env_value "${legacy_env}" "SMS_ENABLED" "false"
assert_env_value "${legacy_env}" "EMAIL_ENABLED" "false"
assert_env_value "${legacy_env}" "EMAIL_DRIVER" "smtp"
assert_env_value "${legacy_env}" "EMAIL_STUDENT_VERIFICATION_SUBJECT" "学生认证验证码"
assert_env_value "${legacy_env}" "EMAIL_FROM_NAME" "StuHelper 系统邮件"
assert_env_value "${legacy_env}" "EMAIL_TENCENT_REGION" "ap-guangzhou"
assert_env_value "${legacy_env}" "EMAIL_TENCENT_ENDPOINT" "ses.tencentcloudapi.com"
assert_env_value "${legacy_env}" "EMAIL_TENCENT_TEMPLATE_PURPOSE" "学校邮箱认证"
assert_env_value "${legacy_env}" "EMAIL_TENCENT_TEMPLATE_SCHOOL_NAME" "北京航空航天大学"
assert_env_value "${legacy_env}" "EMAIL_TENCENT_TEMPLATE_EXPIRE_MINUTES" "5"
assert_env_value "${legacy_env}" "EMAIL_RESEND_ENDPOINT" "https://api.resend.com/emails"
assert_env_value "${legacy_env}" "WEB_VITE_API_URL" "/api"
assert_env_value "${legacy_env}" "API_IP_RATE_LIMIT" "5000"
assert_env_value "${legacy_env}" "API_GLOBAL_RATE_LIMIT" "50000"
assert_env_value "${legacy_env}" "REVIEW_RATE_POST_LIMIT" "500"
assert_env_value "${legacy_env}" "REVIEW_RATE_SEARCH_USER_LIMIT" "500"
assert_env_value "${legacy_env}" "OPEN_PLATFORM_CONSENT_BASE_URL" ""
assert_env_value "${legacy_env}" "OPEN_PLATFORM_ACCOUNT_BASE_URL" ""
assert_env_value "${legacy_env}" "OPEN_PLATFORM_DISCLOSURE_APP_LIMIT" "600"
assert_env_value "${legacy_env}" "OPEN_PLATFORM_DISCLOSURE_APP_USER_LIMIT" "120"
assert_env_value "${legacy_env}" "OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_REQUIRED" "false"
assert_env_value "${legacy_env}" "OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND" ""
assert_env_value "${legacy_env}" "OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_TIMEOUT_SECONDS" "30"
assert_env_value "${legacy_env}" "CASDOOR_TOKEN_PROBE_BROWSER_HEADLESS" "true"
assert_env_value "${legacy_env}" "CASDOOR_TOKEN_PROBE_BROWSER_NO_SANDBOX" "true"
assert_env_value "${legacy_env}" "MINIO_API_EXTERNAL_PORT" "9000"
assert_env_value "${legacy_env}" "MINIO_CONSOLE_EXTERNAL_PORT" "9001"
assert_env_value "${legacy_env}" "CASDOOR_USER_PROFILE_CLIENT_ID" "casdoor-admin-user-profile"
assert_env_value "${legacy_env}" "CASDOOR_USER_PROFILE_APPLICATION" "casdoor-admin-user-profile"
assert_env_value "${legacy_env}" "CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_ID" "casdoor-token-probe-smoke"
assert_file_contains "${legacy_env}" '^CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_SECRET=dev-casdoor-token-probe-smoke-[0-9a-f]+$'
assert_env_value "${legacy_env}" "CASDOOR_TOKEN_PROBE_SMOKE_APPLICATION" "casdoor-token-probe-smoke"
assert_env_value "${legacy_env}" "CASDOOR_TOKEN_PROBE_SMOKE_REDIRECT_URI" "http://localhost:3000/open-platform/token-probe/callback"
assert_env_value "${legacy_env}" "OPENFGA_RESOURCE_SMOKE_MODE" "host"

polluted_dir="$(mktemp -d)"
cleanup_dirs+=("${polluted_dir}")
cp "${REPO_ROOT}/.env.example" "${polluted_dir}/.env"
python3 - "${polluted_dir}/.env" <<'PY'
from pathlib import Path
import sys

path = Path(sys.argv[1])
overrides = {
    "STACK_NAME": "stuhelper-prod-parity",
    "COMPOSE_PROJECT_NAME": "stuhelper-prod-parity",
    "APP_ENV": "prod-parity",
    "TRAEFIK_HTTP_PORT": "8088",
    "TRAEFIK_HTTPS_PORT": "443",
    "DATABASE_URL": "postgres://stuhelper_app:prod-parity-app@postgres:5432/stuhelper?sslmode=disable",
    "BACKUP_DATABASE_URL": "postgres://stuhelper_backup:prod-parity-backup@postgres:5432/stuhelper?sslmode=disable",
    "REPLICATION_DATABASE_URL": "postgres://stuhelper_replication:prod-parity-repl@postgres:5432/stuhelper?sslmode=disable",
    "EXTERNAL_POSTGRES_ENABLED": "true",
    "EXTERNAL_POSTGRES_ALLOW_PLAINTEXT": "true",
    "EXTERNAL_DATASTORE_NETWORK": "stuhelper-prod-parity-baota-net",
    "SHARED_POSTGRES_CONTAINER": "stuhelper-prod-parity-postgres",
    "PROD_PARITY_POSTGRES_CONTAINER": "stuhelper-prod-parity-postgres",
    "PROD_PARITY_POSTGRES_PORT": "15432",
    "SHARED_POSTGRES_SUPERUSER": "postgres",
    "SHARED_POSTGRES_DB": "postgres",
    "POSTGRES_HOST": "postgres",
    "POSTGRES_EXTERNAL_PORT": "15432",
    "REDIS_HOST": "redis",
    "REDIS_EXTERNAL_PORT": "26379",
    "REDIS_TLS_CA": "/redis-tls/ca.crt",
    "CORS_ORIGINS": "http://stuhelper.com,http://join.stuhelper.com,http://sso.stuhelper.com",
    "CASDOOR_EXTERNALPORT": "28085",
    "CASDOOR_ISSUER": "http://sso.stuhelper.com",
    "CASDOOR_INTERNAL_ADDRESS": "host.docker.internal:80",
    "CASDOOR_PUBLIC_AUTH_BASE_URL": "http://sso.stuhelper.com",
    "CASDOOR_REDIRECT_URI": "http://stuhelper.com/api/v1/auth/callback",
    "CASDOOR_BOOTSTRAP_ENABLED": "true",
    "CASDOOR_BOOTSTRAP_ENV_FILE": "/workspace/.run/prod-parity/.env.casdoor-bootstrap.local",
    "CASDOOR_ADMIN_REDIRECT_URI": "http://stuhelper.com/api/v1/auth/callback",
    "CASDOOR_UNIAPP_REDIRECT_URI": "http://stuhelper.com/api/v1/auth/callback",
    "CASDOOR_SMS_PROVIDER_ENABLED": "true",
    "CASDOOR_SMS_PROVIDER_ENDPOINT": "http://app:8080/internal/sms/send",
    "SMS_ENABLED": "true",
    "SMS_APP_ID": "prod-parity-sms-app",
    "SMS_SIGN_NAME": "StuHelper",
    "SMS_TEMPLATE_ID": "prod-parity-template",
    "BACKEND_EXTERNAL_PORT": "28080",
    "WEB_EXTERNAL_PORT": "28000",
    "ADMIN_EXTERNAL_PORT": "28001",
    "WEB_PUBLIC_URL": "http://stuhelper.com",
    "ADMIN_PUBLIC_URL": "http://stuhelper.com/admin/",
    "ADMISSION_PUBLIC_BASE_URL": "http://join.stuhelper.com",
    "PUBLIC_INGRESS_CASDOOR_UPSTREAM_PREFLIGHT_ENABLED": "true",
    "STUHELPER_PLATFORM_BASE_URL": "http://stuhelper.com",
    "WEB_VITE_SSO_URL": "http://sso.stuhelper.com",
    "WEB_VITE_WEB_URL": "http://stuhelper.com",
    "OPENFGA_API_URL": "http://openfga:8080",
    "OPENFGA_HTTP_EXTERNAL_PORT": "8081",
    "OPENFGA_GRPC_EXTERNAL_PORT": "8082",
    "OPENFGA_PLAYGROUND_EXTERNAL_PORT": "3002",
    "OPENFGA_RESOURCE_SMOKE_MODE": "container",
    "OPEN_PLATFORM_PRODUCTION_EVIDENCE_ALLOW_LOCAL_TARGETS": "true",
    "OTEL_EXPORTER_OTLP_ENDPOINT": "http://alloy:4318",
    "OBJECT_STORAGE_ENDPOINT": "http://minio:9000",
    "OBJECT_STORAGE_ACCESS_KEY_ID": "stuhelper-prod-parity",
    "OBJECT_STORAGE_USE_SSL": "true",
    "MINIO_API_EXTERNAL_PORT": "29000",
    "MINIO_CONSOLE_EXTERNAL_PORT": "29001",
    "TOKEN_COOKIE_SECURE": "true",
    "TOKEN_COOKIE_DOMAIN": ".stuhelper.com",
    "ALLOW_LOCAL_ALERT_SINK": "true",
    "ALERTMANAGER_WEBHOOK_URL": "http://alert-webhook-sink:8080/alerts",
    "BACKEND_IMAGE_REF": "stuhelper/backend:prod-parity-deadbee",
    "FRONTEND_IMAGE_REF": "stuhelper/frontend:prod-parity-deadbee",
    "ADMIN_IMAGE_REF": "stuhelper/admin:prod-parity-deadbee",
}
seen = set()
lines = []
for line in path.read_text().splitlines():
    if "=" not in line or line.startswith("#"):
        lines.append(line)
        continue
    key = line.split("=", 1)[0]
    if key in overrides:
        lines.append(f"{key}={overrides[key]}")
        seen.add(key)
    else:
        lines.append(line)
for key, value in overrides.items():
    if key not in seen:
        lines.append(f"{key}={value}")
path.write_text("\n".join(lines) + "\n")
PY

ENV_FILE="${polluted_dir}/.env" \
GENERATED_ENV_FILE="${polluted_dir}/.env.generated" \
GENERATED_SECRET_ENV_FILE="${polluted_dir}/.env.generated.secrets" \
GENERATED_OBS_DIR="${polluted_dir}/generated/observability" \
bash "${INIT_SCRIPT}" >"${polluted_dir}/stdout.log" 2>"${polluted_dir}/stderr.log"

polluted_env="${polluted_dir}/.env"
assert_file_not_contains "${polluted_env}" '^TRAEFIK_'
assert_env_value "${polluted_env}" "STACK_NAME" "stuhelper-dev"
assert_env_value "${polluted_env}" "COMPOSE_PROJECT_NAME" "stuhelper-dev"
assert_env_value "${polluted_env}" "APP_ENV" "development"
assert_file_contains "${polluted_env}" '^DATABASE_URL=postgres://stuhelper_app:dev-app-[0-9a-f]+@localhost:5432/stuhelper\?sslmode=disable$'
assert_env_value "${polluted_env}" "BACKUP_DATABASE_URL" ""
assert_env_value "${polluted_env}" "REPLICATION_DATABASE_URL" ""
assert_env_value "${polluted_env}" "EXTERNAL_POSTGRES_ENABLED" "false"
assert_env_value "${polluted_env}" "EXTERNAL_POSTGRES_ALLOW_PLAINTEXT" "false"
assert_env_value "${polluted_env}" "EXTERNAL_DATASTORE_NETWORK" ""
assert_env_value "${polluted_env}" "SHARED_POSTGRES_CONTAINER" ""
assert_env_value "${polluted_env}" "PROD_PARITY_POSTGRES_CONTAINER" ""
assert_env_value "${polluted_env}" "PROD_PARITY_POSTGRES_PORT" ""
assert_env_value "${polluted_env}" "SHARED_POSTGRES_SUPERUSER" ""
assert_env_value "${polluted_env}" "SHARED_POSTGRES_DB" ""
assert_env_value "${polluted_env}" "POSTGRES_HOST" "localhost"
assert_env_value "${polluted_env}" "POSTGRES_EXTERNAL_PORT" "5432"
assert_env_value "${polluted_env}" "REDIS_HOST" "localhost"
assert_env_value "${polluted_env}" "REDIS_EXTERNAL_PORT" "6379"
assert_env_value "${polluted_env}" "REDIS_TLS_CA" "/tls/ca.crt"
assert_env_value "${polluted_env}" "CORS_ORIGINS" "http://localhost:3000,http://127.0.0.1:3000,http://localhost:3001,http://127.0.0.1:3001"
assert_env_value "${polluted_env}" "CASDOOR_EXTERNALPORT" "8085"
assert_env_value "${polluted_env}" "CASDOOR_ISSUER" "http://localhost:8085"
assert_env_value "${polluted_env}" "CASDOOR_INTERNAL_ADDRESS" "casdoor:8000"
assert_env_value "${polluted_env}" "CASDOOR_PUBLIC_AUTH_BASE_URL" ""
assert_env_value "${polluted_env}" "CASDOOR_REDIRECT_URI" "http://localhost:8080/api/v1/auth/callback"
assert_env_value "${polluted_env}" "CASDOOR_BOOTSTRAP_ENABLED" "false"
assert_env_value "${polluted_env}" "CASDOOR_BOOTSTRAP_ENV_FILE" ".env.casdoor-bootstrap.local"
assert_env_value "${polluted_env}" "CASDOOR_ADMIN_REDIRECT_URI" "http://localhost:8080/api/v1/auth/callback"
assert_env_value "${polluted_env}" "CASDOOR_UNIAPP_REDIRECT_URI" "http://localhost:8080/api/v1/auth/callback"
assert_env_value "${polluted_env}" "CASDOOR_SMS_PROVIDER_ENABLED" "false"
assert_env_value "${polluted_env}" "CASDOOR_SMS_PROVIDER_ENDPOINT" "http://host.docker.internal:8080/internal/sms/send"
assert_env_value "${polluted_env}" "BACKEND_EXTERNAL_PORT" "8080"
assert_env_value "${polluted_env}" "WEB_EXTERNAL_PORT" "3000"
assert_env_value "${polluted_env}" "ADMIN_EXTERNAL_PORT" "3001"
assert_env_value "${polluted_env}" "WEB_PUBLIC_URL" "http://localhost:3000"
assert_env_value "${polluted_env}" "ADMIN_PUBLIC_URL" "http://localhost:3001"
assert_env_value "${polluted_env}" "ADMISSION_PUBLIC_BASE_URL" "http://localhost:3000"
assert_env_value "${polluted_env}" "PUBLIC_INGRESS_CASDOOR_UPSTREAM_PREFLIGHT_ENABLED" "false"
assert_file_not_contains "${polluted_env}" '^SSO_PUBLIC_'
assert_env_value "${polluted_env}" "STUHELPER_PLATFORM_BASE_URL" "http://localhost:8080"
assert_env_value "${polluted_env}" "WEB_VITE_SSO_URL" "http://localhost:8085"
assert_env_value "${polluted_env}" "WEB_VITE_WEB_URL" "http://localhost:3000"
assert_env_value "${polluted_env}" "OPENFGA_API_URL" "http://localhost:8081"
assert_env_value "${polluted_env}" "OPENFGA_HTTP_EXTERNAL_PORT" "8081"
assert_env_value "${polluted_env}" "OPENFGA_GRPC_EXTERNAL_PORT" "8082"
assert_env_value "${polluted_env}" "OPENFGA_PLAYGROUND_EXTERNAL_PORT" "3002"
assert_env_value "${polluted_env}" "OPENFGA_RESOURCE_SMOKE_MODE" "host"
assert_env_value "${polluted_env}" "SMS_ENABLED" "false"
assert_env_value "${polluted_env}" "SMS_APP_ID" ""
assert_env_value "${polluted_env}" "SMS_SIGN_NAME" ""
assert_env_value "${polluted_env}" "SMS_TEMPLATE_ID" ""
assert_env_value "${polluted_env}" "OPEN_PLATFORM_PRODUCTION_EVIDENCE_ALLOW_LOCAL_TARGETS" "false"
assert_env_value "${polluted_env}" "OTEL_EXPORTER_OTLP_ENDPOINT" "http://localhost:4318"
assert_env_value "${polluted_env}" "OBJECT_STORAGE_ENDPOINT" "http://localhost:9000"
assert_env_value "${polluted_env}" "OBJECT_STORAGE_ACCESS_KEY_ID" "stuhelper"
assert_env_value "${polluted_env}" "OBJECT_STORAGE_USE_SSL" "false"
assert_env_value "${polluted_env}" "MINIO_API_EXTERNAL_PORT" "9000"
assert_env_value "${polluted_env}" "MINIO_CONSOLE_EXTERNAL_PORT" "9001"
assert_env_value "${polluted_env}" "TOKEN_COOKIE_SECURE" "false"
assert_env_value "${polluted_env}" "TOKEN_COOKIE_DOMAIN" ""
assert_env_value "${polluted_env}" "ALLOW_LOCAL_ALERT_SINK" "false"
assert_env_value "${polluted_env}" "ALERTMANAGER_WEBHOOK_URL" ""
assert_env_value "${polluted_env}" "BACKEND_IMAGE_REF" "stuhelper/backend:dev-placeholder"
assert_env_value "${polluted_env}" "FRONTEND_IMAGE_REF" "stuhelper/frontend:dev-placeholder"
assert_env_value "${polluted_env}" "ADMIN_IMAGE_REF" "stuhelper/admin:dev-placeholder"

echo "[init-dev-env-contract] ok"
