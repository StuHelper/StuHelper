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
assert_env_value "${fresh_env}" "CORS_ORIGINS" "REPLACE_WITH_PRODUCTION_CORS_ORIGINS"
assert_env_value "${fresh_env}" "CASDOOR_ISSUER" "REPLACE_WITH_CASDOOR_ISSUER"
assert_env_value "${fresh_env}" "CASDOOR_REDIRECT_URI" "REPLACE_WITH_CASDOOR_REDIRECT_URI"
assert_env_value "${fresh_env}" "CASDOOR_CLIENT_ID" "REPLACE_WITH_CASDOOR_CLIENT_ID"
assert_env_value "${fresh_env}" "CASDOOR_BOOTSTRAP_ENABLED" "true"
assert_env_value "${fresh_env}" "CASDOOR_BOOTSTRAP_ENV_FILE" ".env.casdoor-bootstrap.local"
assert_env_value "${fresh_env}" "CASDOOR_ADMIN_CLIENT_ID" "stuhelper-admin"
assert_env_value "${fresh_env}" "CASDOOR_ADMIN_REDIRECT_URI" "REPLACE_WITH_CASDOOR_ADMIN_REDIRECT_URI"
assert_env_value "${fresh_env}" "CASDOOR_UNIAPP_CLIENT_ID" "stuhelper-uniapp"
assert_env_value "${fresh_env}" "CASDOOR_UNIAPP_REDIRECT_URI" "REPLACE_WITH_CASDOOR_UNIAPP_REDIRECT_URI"
assert_env_value "${fresh_env}" "CASDOOR_SMS_PROVIDER_ENABLED" "true"
assert_env_value "${fresh_env}" "CASDOOR_SMS_PROVIDER_ENDPOINT" "http://app:8080/internal/sms/send"
assert_env_value "${fresh_env}" "CASDOOR_APP_PROVISIONING_CLIENT_ID" "REPLACE_WITH_CASDOOR_APP_PROVISIONING_CLIENT_ID"
assert_env_value "${fresh_env}" "CASDOOR_APP_PROVISIONING_APPLICATION" "REPLACE_WITH_CASDOOR_APP_PROVISIONING_APPLICATION"
assert_env_value "${fresh_env}" "CASDOOR_ROLE_SYNC_CLIENT_ID" "REPLACE_WITH_CASDOOR_ROLE_SYNC_CLIENT_ID"
assert_env_value "${fresh_env}" "CASDOOR_ROLE_SYNC_APPLICATION" "REPLACE_WITH_CASDOOR_ROLE_SYNC_APPLICATION"
assert_env_value "${fresh_env}" "CASDOOR_USER_LOOKUP_CLIENT_ID" "REPLACE_WITH_CASDOOR_USER_LOOKUP_CLIENT_ID"
assert_env_value "${fresh_env}" "CASDOOR_USER_LOOKUP_APPLICATION" "REPLACE_WITH_CASDOOR_USER_LOOKUP_APPLICATION"
assert_env_value "${fresh_env}" "WEB_PUBLIC_URL" "REPLACE_WITH_WEB_PUBLIC_URL"
assert_env_value "${fresh_env}" "ADMIN_PUBLIC_URL" "REPLACE_WITH_ADMIN_PUBLIC_URL"
assert_env_value "${fresh_env}" "WEB_VITE_API_URL" "/api"
assert_env_value "${fresh_env}" "WEB_VITE_SSO_URL" "REPLACE_WITH_WEB_VITE_SSO_URL"
assert_env_value "${fresh_env}" "OPENFGA_API_URL" "http://openfga:8080"
assert_env_value "${fresh_env}" "OBJECT_STORAGE_ENDPOINT" "REPLACE_WITH_OBJECT_STORAGE_ENDPOINT"
assert_env_value "${fresh_env}" "OBJECT_STORAGE_USE_SSL" "true"
assert_env_value "${fresh_env}" "OBJECT_STORAGE_FORCE_PATH_STYLE" "false"
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
assert_env_value "${legacy_env}" "CORS_ORIGINS" "REPLACE_WITH_PRODUCTION_CORS_ORIGINS"
assert_env_value "${legacy_env}" "CASDOOR_ISSUER" "REPLACE_WITH_CASDOOR_ISSUER"
assert_env_value "${legacy_env}" "CASDOOR_INTERNAL_ADDRESS" ""
assert_env_value "${legacy_env}" "CASDOOR_REDIRECT_URI" "REPLACE_WITH_CASDOOR_REDIRECT_URI"
assert_env_value "${legacy_env}" "CASDOOR_CLIENT_ID" "REPLACE_WITH_CASDOOR_CLIENT_ID"
assert_env_value "${legacy_env}" "CASDOOR_BOOTSTRAP_ENABLED" "true"
assert_env_value "${legacy_env}" "CASDOOR_BOOTSTRAP_ENV_FILE" ".env.casdoor-bootstrap.local"
assert_env_value "${legacy_env}" "CASDOOR_ADMIN_CLIENT_ID" "stuhelper-admin"
assert_env_value "${legacy_env}" "CASDOOR_ADMIN_REDIRECT_URI" "REPLACE_WITH_CASDOOR_ADMIN_REDIRECT_URI"
assert_env_value "${legacy_env}" "CASDOOR_UNIAPP_CLIENT_ID" "stuhelper-uniapp"
assert_env_value "${legacy_env}" "CASDOOR_UNIAPP_REDIRECT_URI" "REPLACE_WITH_CASDOOR_UNIAPP_REDIRECT_URI"
assert_env_value "${legacy_env}" "CASDOOR_APP_PROVISIONING_CLIENT_ID" "REPLACE_WITH_CASDOOR_APP_PROVISIONING_CLIENT_ID"
assert_env_value "${legacy_env}" "CASDOOR_APP_PROVISIONING_APPLICATION" "REPLACE_WITH_CASDOOR_APP_PROVISIONING_APPLICATION"
assert_env_value "${legacy_env}" "WEB_PUBLIC_URL" "REPLACE_WITH_WEB_PUBLIC_URL"
assert_env_value "${legacy_env}" "ADMIN_PUBLIC_URL" "REPLACE_WITH_ADMIN_PUBLIC_URL"
assert_env_value "${legacy_env}" "WEB_VITE_API_URL" "/api"
assert_env_value "${legacy_env}" "WEB_VITE_SSO_URL" "REPLACE_WITH_WEB_VITE_SSO_URL"
assert_env_value "${legacy_env}" "OPENFGA_API_URL" "http://openfga:8080"
assert_env_value "${legacy_env}" "OBJECT_STORAGE_ENDPOINT" "REPLACE_WITH_OBJECT_STORAGE_ENDPOINT"
assert_env_value "${legacy_env}" "OBJECT_STORAGE_USE_SSL" "true"
assert_env_value "${legacy_env}" "OBJECT_STORAGE_FORCE_PATH_STYLE" "false"
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

echo "[init-prod-env-contract] all assertions passed"
