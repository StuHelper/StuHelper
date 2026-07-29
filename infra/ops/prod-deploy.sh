#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

require_cmd docker
require_cmd curl
require_cmd jq
require_cmd python3
require_cmd openssl

if [[ -f "${REMOTE_DEPLOY_CONFIG_FILE}" ]]; then
  load_remote_deploy_config
fi

[[ -n "${GENERATED_ENV_SECRET_REF:-}" ]] || die "GENERATED_ENV_SECRET_REF must be configured for production deploy"
[[ -n "${SECRET_BACKEND:-}" && "${SECRET_BACKEND:-}" != "none" && "${SECRET_BACKEND:-}" != "file" ]] || \
  die "production deploy requires a non-file secret backend for generated secrets"

ensure_generated_files
if [[ -n "${SHARED_ENV_SECRET_REF:-}" ]]; then
  mkdir -p "$(dirname "${ENV_FILE}")"
  touch "${ENV_FILE}"
fi
if [[ -n "${SECRETS_ENV_SECRET_REF:-}" && -n "${SECRETS_ENV_FILE:-}" ]]; then
  mkdir -p "$(dirname "${SECRETS_ENV_FILE}")"
  touch "${SECRETS_ENV_FILE}"
fi
pending_generated_secret_ref="${GENERATED_ENV_SECRET_REF:-}"
unset GENERATED_ENV_SECRET_REF
load_env
if [[ -n "${pending_generated_secret_ref}" ]]; then
  export GENERATED_ENV_SECRET_REF="${pending_generated_secret_ref}"
fi

python3 "${REPO_ROOT}/infra/ops/validate-runtime-image-scan.py" \
  --repo-root "${REPO_ROOT}" \
  --policy-only \
  --effective-environment production

require_nonempty() {
  local key="$1"
  local value="${2:-}"
  [[ -n "${value}" ]] || die "${key} is required for production deploy"
}

reject_placeholder() {
  local key="$1"
  local value="${2:-}"
  shift 2 || true
  for placeholder in "$@"; do
    if [[ "${value}" == "${placeholder}" ]]; then
      die "${key} is using placeholder/default value (${placeholder}); set a real production value first"
    fi
  done
}

reject_placeholder_if_set() {
  local key="$1"
  local value="${2:-}"
  shift 2 || true
  [[ -n "${value}" ]] || return 0
  reject_placeholder "${key}" "${value}" "$@"
}

reject_local_value() {
  local key="$1"
  local value="${2:-}"
  local normalized="${value,,}"
  case "${normalized}" in
    *localhost*|*127.0.0.1*|*::1*|*host.docker.internal*|*alert-webhook-sink*)
      die "${key} must not point to a local/development endpoint (${value})"
      ;;
  esac
}

resolve_env_path() {
  local raw="$1"
  case "${raw}" in
    /*) printf '%s\n' "${raw}" ;;
    *) printf '%s/%s\n' "$(dirname "${ENV_FILE}")" "${raw}" ;;
  esac
}

source_casdoor_bootstrap_env() {
  local file
  file="$(resolve_env_path "${CASDOOR_BOOTSTRAP_ENV_FILE:-.env.casdoor-bootstrap.local}")"
  [[ -f "${file}" ]] || die "missing Casdoor bootstrap env file: ${file}"
  # shellcheck disable=SC1090
  source "${file}"
}

require_immutable_image_ref() {
  local key="$1"
  local value="${2:-}"
  require_nonempty "${key}" "${value}"
  python3 - "${key}" "${value}" <<'PY'
import re
import sys

key = sys.argv[1]
ref = sys.argv[2].strip()

if "@sha256:" in ref:
    if not re.fullmatch(r".+@sha256:[0-9a-f]{64}", ref):
        raise SystemExit(f"{key} must use a valid pinned digest reference: {ref}")
    raise SystemExit(0)

image, sep, tag = ref.rpartition(":")
if not sep or "/" not in image or not tag:
    raise SystemExit(f"{key} must be pinned by explicit tag or digest: {ref}")

if tag in {"latest", "develop-latest", "stable", "main", "master"} or tag.startswith("ci-"):
    raise SystemExit(f"{key} uses a mutable or pre-release tag and is not allowed in production: {ref}")
PY
}

require_backup_object_storage_config
source_casdoor_bootstrap_env # load bootstrap credential env

if [[ "${EXTERNAL_POSTGRES_ENABLED:-false}" != "true" ]]; then
  require_nonempty POSTGRES_PASSWORD "${POSTGRES_PASSWORD:-}"
fi
require_nonempty REDIS_PASSWORD "${REDIS_PASSWORD:-}"
require_nonempty REDIS_EXPORTER_PASSWORD "${REDIS_EXPORTER_PASSWORD:-}"
require_nonempty POSTGRES_EXPORTER_DB_PASSWORD "${POSTGRES_EXPORTER_DB_PASSWORD:-}"
require_nonempty METRICS_PASSWORD "${METRICS_PASSWORD:-}"
require_nonempty GRAFANA_ADMIN_PASSWORD "${GRAFANA_ADMIN_PASSWORD:-}"
require_nonempty ALERTMANAGER_WEBHOOK_URL "${ALERTMANAGER_WEBHOOK_URL:-}"
require_nonempty TRUSTED_PROXIES "${TRUSTED_PROXIES:-}"
require_nonempty CORS_ORIGINS "${CORS_ORIGINS:-}"
require_nonempty HMAC_SECRET "${HMAC_SECRET:-}"
require_nonempty DOC_AES_KEYS "${DOC_AES_KEYS:-}"
require_nonempty CASDOOR_ISSUER "${CASDOOR_ISSUER:-}"
require_nonempty CASDOOR_PUBLIC_AUTH_BASE_URL "${CASDOOR_PUBLIC_AUTH_BASE_URL:-}"
require_nonempty CASDOOR_REDIRECT_URI "${CASDOOR_REDIRECT_URI:-}"
require_nonempty CASDOOR_CLIENT_ID "${CASDOOR_CLIENT_ID:-}"
require_nonempty CASDOOR_CLIENT_SECRET "${CASDOOR_CLIENT_SECRET:-}"
require_nonempty CASDOOR_BOOTSTRAP_CLIENT_ID "${CASDOOR_BOOTSTRAP_CLIENT_ID:-}"
require_nonempty CASDOOR_BOOTSTRAP_CLIENT_SECRET "${CASDOOR_BOOTSTRAP_CLIENT_SECRET:-}"
require_nonempty CASDOOR_BOOTSTRAP_APPLICATION "${CASDOOR_BOOTSTRAP_APPLICATION:-}"
require_nonempty CASDOOR_ADMIN_CLIENT_ID "${CASDOOR_ADMIN_CLIENT_ID:-}"
require_nonempty CASDOOR_ADMIN_CLIENT_SECRET "${CASDOOR_ADMIN_CLIENT_SECRET:-}"
require_nonempty CASDOOR_ADMIN_REDIRECT_URI "${CASDOOR_ADMIN_REDIRECT_URI:-}"
require_nonempty CASDOOR_UNIAPP_CLIENT_ID "${CASDOOR_UNIAPP_CLIENT_ID:-}"
require_nonempty CASDOOR_UNIAPP_CLIENT_SECRET "${CASDOOR_UNIAPP_CLIENT_SECRET:-}"
require_nonempty CASDOOR_UNIAPP_REDIRECT_URI "${CASDOOR_UNIAPP_REDIRECT_URI:-}"
require_nonempty CASDOOR_SMS_PROVIDER_NAME "${CASDOOR_SMS_PROVIDER_NAME:-}"
require_nonempty CASDOOR_SMS_PROVIDER_DISPLAY_NAME "${CASDOOR_SMS_PROVIDER_DISPLAY_NAME:-}"
require_nonempty CASDOOR_SMS_PROVIDER_CATEGORY "${CASDOOR_SMS_PROVIDER_CATEGORY:-}"
require_nonempty CASDOOR_SMS_PROVIDER_TYPE "${CASDOOR_SMS_PROVIDER_TYPE:-}"
require_nonempty CASDOOR_SMS_PROVIDER_METHOD "${CASDOOR_SMS_PROVIDER_METHOD:-}"
require_nonempty CASDOOR_SMS_PROVIDER_TITLE "${CASDOOR_SMS_PROVIDER_TITLE:-}"
require_nonempty CASDOOR_SMS_PROVIDER_ENDPOINT "${CASDOOR_SMS_PROVIDER_ENDPOINT:-}"
require_nonempty CASDOOR_APP_PROVISIONING_CLIENT_ID "${CASDOOR_APP_PROVISIONING_CLIENT_ID:-}"
require_nonempty CASDOOR_APP_PROVISIONING_CLIENT_SECRET "${CASDOOR_APP_PROVISIONING_CLIENT_SECRET:-}"
require_nonempty CASDOOR_APP_PROVISIONING_APPLICATION "${CASDOOR_APP_PROVISIONING_APPLICATION:-}"
require_nonempty CASDOOR_USER_PROFILE_CLIENT_ID "${CASDOOR_USER_PROFILE_CLIENT_ID:-}"
require_nonempty CASDOOR_USER_PROFILE_CLIENT_SECRET "${CASDOOR_USER_PROFILE_CLIENT_SECRET:-}"
require_nonempty CASDOOR_USER_PROFILE_APPLICATION "${CASDOOR_USER_PROFILE_APPLICATION:-}"
require_nonempty CASDOOR_INTROSPECTION_CLIENT_ID "${CASDOOR_INTROSPECTION_CLIENT_ID:-}"
require_nonempty CASDOOR_INTROSPECTION_CLIENT_SECRET "${CASDOOR_INTROSPECTION_CLIENT_SECRET:-}"
require_nonempty CASDOOR_INTROSPECTION_APPLICATION "${CASDOOR_INTROSPECTION_APPLICATION:-}"
require_nonempty CASDOOR_ROLE_SYNC_CLIENT_ID "${CASDOOR_ROLE_SYNC_CLIENT_ID:-}"
require_nonempty CASDOOR_ROLE_SYNC_CLIENT_SECRET "${CASDOOR_ROLE_SYNC_CLIENT_SECRET:-}"
require_nonempty CASDOOR_ROLE_SYNC_APPLICATION "${CASDOOR_ROLE_SYNC_APPLICATION:-}"
require_nonempty CASDOOR_USER_LOOKUP_CLIENT_ID "${CASDOOR_USER_LOOKUP_CLIENT_ID:-}"
require_nonempty CASDOOR_USER_LOOKUP_CLIENT_SECRET "${CASDOOR_USER_LOOKUP_CLIENT_SECRET:-}"
require_nonempty CASDOOR_USER_LOOKUP_APPLICATION "${CASDOOR_USER_LOOKUP_APPLICATION:-}"
require_nonempty CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_ID "${CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_ID:-}"
require_nonempty CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_SECRET "${CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_SECRET:-}"
require_nonempty CASDOOR_TOKEN_PROBE_SMOKE_APPLICATION "${CASDOOR_TOKEN_PROBE_SMOKE_APPLICATION:-}"
require_nonempty CASDOOR_TOKEN_PROBE_SMOKE_REDIRECT_URI "${CASDOOR_TOKEN_PROBE_SMOKE_REDIRECT_URI:-}"
require_nonempty OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_REQUIRED "${OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_REQUIRED:-}"
require_nonempty OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND "${OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND:-}"
require_nonempty OPEN_PLATFORM_CONSENT_BASE_URL "${OPEN_PLATFORM_CONSENT_BASE_URL:-}"
require_nonempty OPEN_PLATFORM_ACCOUNT_BASE_URL "${OPEN_PLATFORM_ACCOUNT_BASE_URL:-}"
if [[ "${OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND:-}" == *"casdoor-runtime-token-probe-runner.mjs"* ]]; then
  require_nonempty CASDOOR_TOKEN_PROBE_USERNAME "${CASDOOR_TOKEN_PROBE_USERNAME:-}"
  require_nonempty CASDOOR_TOKEN_PROBE_PASSWORD "${CASDOOR_TOKEN_PROBE_PASSWORD:-}"
  require_nonempty CASDOOR_TOKEN_PROBE_BROWSER_EXECUTABLE_PATH "${CASDOOR_TOKEN_PROBE_BROWSER_EXECUTABLE_PATH:-}"
fi
require_nonempty SMS_SECRET_ID "${SMS_SECRET_ID:-}"
require_nonempty SMS_SECRET_KEY "${SMS_SECRET_KEY:-}"
require_nonempty SMS_APP_ID "${SMS_APP_ID:-}"
require_nonempty SMS_SIGN_NAME "${SMS_SIGN_NAME:-}"
require_nonempty SMS_TEMPLATE_ID "${SMS_TEMPLATE_ID:-}"
require_nonempty SMS_REGION "${SMS_REGION:-}"
require_nonempty SMS_INTERNAL_KEY "${SMS_INTERNAL_KEY:-}"
require_nonempty BOT_SERVICE_TOKEN "${BOT_SERVICE_TOKEN:-}"
require_nonempty WEB_PUBLIC_URL "${WEB_PUBLIC_URL:-}"
require_nonempty ADMIN_PUBLIC_URL "${ADMIN_PUBLIC_URL:-}"
require_nonempty ADMISSION_PUBLIC_BASE_URL "${ADMISSION_PUBLIC_BASE_URL:-}"
require_nonempty ADMISSION_PRODUCTION_READINESS_ENABLED "${ADMISSION_PRODUCTION_READINESS_ENABLED:-}"
require_nonempty STUHELPER_FRESHMAN_MATERIAL_HOSTS "${STUHELPER_FRESHMAN_MATERIAL_HOSTS:-}"
require_nonempty WEB_VITE_SSO_URL "${WEB_VITE_SSO_URL:-}"
require_nonempty WEB_VITE_WEB_URL "${WEB_VITE_WEB_URL:-}"
require_nonempty OPENFGA_API_URL "${OPENFGA_API_URL:-}"
reject_placeholder_if_set OPENFGA_STORE_ID "${OPENFGA_STORE_ID:-}" "REPLACE_WITH_OPENFGA_STORE_ID"
reject_placeholder_if_set OPENFGA_MODEL_ID "${OPENFGA_MODEL_ID:-}" "REPLACE_WITH_OPENFGA_MODEL_ID"
require_nonempty OBJECT_STORAGE_ENDPOINT "${OBJECT_STORAGE_ENDPOINT:-}"
require_nonempty OBJECT_STORAGE_BUCKET "${OBJECT_STORAGE_BUCKET:-}"
require_nonempty OBJECT_STORAGE_ACCESS_KEY_ID "${OBJECT_STORAGE_ACCESS_KEY_ID:-}"
require_nonempty OBJECT_STORAGE_SECRET_ACCESS_KEY "${OBJECT_STORAGE_SECRET_ACCESS_KEY:-}"
require_nonempty GRAFANA_ROOT_URL "${GRAFANA_ROOT_URL:-}"
require_immutable_image_ref BACKEND_IMAGE_REF "${BACKEND_IMAGE_REF:-}"
require_immutable_image_ref FRONTEND_IMAGE_REF "${FRONTEND_IMAGE_REF:-}"
require_immutable_image_ref ADMIN_IMAGE_REF "${ADMIN_IMAGE_REF:-}"

if [[ "${EXTERNAL_POSTGRES_ENABLED:-false}" != "true" ]]; then
  reject_placeholder POSTGRES_PASSWORD "${POSTGRES_PASSWORD:-}" "dev123"
fi
reject_placeholder REDIS_PASSWORD "${REDIS_PASSWORD:-}" "dev123"
reject_placeholder REDIS_EXPORTER_PASSWORD "${REDIS_EXPORTER_PASSWORD:-}" "RUN_MAKE_DEV_INIT" "REPLACE_WITH_REDIS_EXPORTER_PASSWORD"
reject_placeholder POSTGRES_EXPORTER_DB_PASSWORD "${POSTGRES_EXPORTER_DB_PASSWORD:-}" "RUN_MAKE_DEV_INIT" "REPLACE_WITH_POSTGRES_EXPORTER_DB_PASSWORD"
[[ "${REDIS_EXPORTER_PASSWORD}" != "${REDIS_PASSWORD}" ]] ||
  die "REDIS_EXPORTER_PASSWORD must be independent from REDIS_PASSWORD"
reject_placeholder GRAFANA_ADMIN_PASSWORD "${GRAFANA_ADMIN_PASSWORD:-}" "ChangeMeBeforeProduction"
reject_placeholder CORS_ORIGINS "${CORS_ORIGINS:-}" "REPLACE_WITH_PRODUCTION_CORS_ORIGINS"
reject_placeholder CASDOOR_ISSUER "${CASDOOR_ISSUER:-}" "REPLACE_WITH_CASDOOR_ISSUER"
reject_placeholder CASDOOR_PUBLIC_AUTH_BASE_URL "${CASDOOR_PUBLIC_AUTH_BASE_URL:-}" "REPLACE_WITH_CASDOOR_PUBLIC_AUTH_BASE_URL"
reject_placeholder CASDOOR_REDIRECT_URI "${CASDOOR_REDIRECT_URI:-}" "REPLACE_WITH_CASDOOR_REDIRECT_URI"
reject_placeholder CASDOOR_CLIENT_ID "${CASDOOR_CLIENT_ID:-}" "REPLACE_WITH_CASDOOR_CLIENT_ID"
reject_placeholder CASDOOR_CLIENT_SECRET "${CASDOOR_CLIENT_SECRET:-}" "REPLACE_WITH_CASDOOR_CLIENT_SECRET"
reject_placeholder CASDOOR_BOOTSTRAP_CLIENT_ID "${CASDOOR_BOOTSTRAP_CLIENT_ID:-}" "REPLACE_WITH_CASDOOR_BOOTSTRAP_CLIENT_ID"
reject_placeholder CASDOOR_BOOTSTRAP_CLIENT_SECRET "${CASDOOR_BOOTSTRAP_CLIENT_SECRET:-}" "REPLACE_WITH_CASDOOR_BOOTSTRAP_CLIENT_SECRET"
reject_placeholder CASDOOR_BOOTSTRAP_APPLICATION "${CASDOOR_BOOTSTRAP_APPLICATION:-}" "REPLACE_WITH_CASDOOR_BOOTSTRAP_APPLICATION"
reject_placeholder CASDOOR_ADMIN_CLIENT_ID "${CASDOOR_ADMIN_CLIENT_ID:-}" "REPLACE_WITH_CASDOOR_ADMIN_CLIENT_ID"
reject_placeholder CASDOOR_ADMIN_CLIENT_SECRET "${CASDOOR_ADMIN_CLIENT_SECRET:-}" "REPLACE_WITH_CASDOOR_ADMIN_CLIENT_SECRET"
reject_placeholder CASDOOR_ADMIN_REDIRECT_URI "${CASDOOR_ADMIN_REDIRECT_URI:-}" "REPLACE_WITH_CASDOOR_ADMIN_REDIRECT_URI"
reject_placeholder CASDOOR_UNIAPP_CLIENT_ID "${CASDOOR_UNIAPP_CLIENT_ID:-}" "REPLACE_WITH_CASDOOR_UNIAPP_CLIENT_ID"
reject_placeholder CASDOOR_UNIAPP_CLIENT_SECRET "${CASDOOR_UNIAPP_CLIENT_SECRET:-}" "REPLACE_WITH_CASDOOR_UNIAPP_CLIENT_SECRET"
reject_placeholder CASDOOR_UNIAPP_REDIRECT_URI "${CASDOOR_UNIAPP_REDIRECT_URI:-}" "REPLACE_WITH_CASDOOR_UNIAPP_REDIRECT_URI"
reject_placeholder CASDOOR_APP_PROVISIONING_CLIENT_ID "${CASDOOR_APP_PROVISIONING_CLIENT_ID:-}" "REPLACE_WITH_CASDOOR_APP_PROVISIONING_CLIENT_ID"
reject_placeholder CASDOOR_APP_PROVISIONING_CLIENT_SECRET "${CASDOOR_APP_PROVISIONING_CLIENT_SECRET:-}" "REPLACE_WITH_CASDOOR_APP_PROVISIONING_CLIENT_SECRET"
reject_placeholder CASDOOR_APP_PROVISIONING_APPLICATION "${CASDOOR_APP_PROVISIONING_APPLICATION:-}" "REPLACE_WITH_CASDOOR_APP_PROVISIONING_APPLICATION"
reject_placeholder CASDOOR_USER_PROFILE_CLIENT_ID "${CASDOOR_USER_PROFILE_CLIENT_ID:-}" "REPLACE_WITH_CASDOOR_USER_PROFILE_CLIENT_ID"
reject_placeholder CASDOOR_USER_PROFILE_CLIENT_SECRET "${CASDOOR_USER_PROFILE_CLIENT_SECRET:-}" "REPLACE_WITH_CASDOOR_USER_PROFILE_CLIENT_SECRET"
reject_placeholder CASDOOR_USER_PROFILE_APPLICATION "${CASDOOR_USER_PROFILE_APPLICATION:-}" "REPLACE_WITH_CASDOOR_USER_PROFILE_APPLICATION"
reject_placeholder CASDOOR_INTROSPECTION_CLIENT_ID "${CASDOOR_INTROSPECTION_CLIENT_ID:-}" "REPLACE_WITH_CASDOOR_INTROSPECTION_CLIENT_ID"
reject_placeholder CASDOOR_INTROSPECTION_CLIENT_SECRET "${CASDOOR_INTROSPECTION_CLIENT_SECRET:-}" "REPLACE_WITH_CASDOOR_INTROSPECTION_CLIENT_SECRET"
reject_placeholder CASDOOR_INTROSPECTION_APPLICATION "${CASDOOR_INTROSPECTION_APPLICATION:-}" "REPLACE_WITH_CASDOOR_INTROSPECTION_APPLICATION"
reject_placeholder CASDOOR_ROLE_SYNC_CLIENT_ID "${CASDOOR_ROLE_SYNC_CLIENT_ID:-}" "REPLACE_WITH_CASDOOR_ROLE_SYNC_CLIENT_ID"
reject_placeholder CASDOOR_ROLE_SYNC_CLIENT_SECRET "${CASDOOR_ROLE_SYNC_CLIENT_SECRET:-}" "REPLACE_WITH_CASDOOR_ROLE_SYNC_CLIENT_SECRET"
reject_placeholder CASDOOR_ROLE_SYNC_APPLICATION "${CASDOOR_ROLE_SYNC_APPLICATION:-}" "REPLACE_WITH_CASDOOR_ROLE_SYNC_APPLICATION"
reject_placeholder CASDOOR_USER_LOOKUP_CLIENT_ID "${CASDOOR_USER_LOOKUP_CLIENT_ID:-}" "REPLACE_WITH_CASDOOR_USER_LOOKUP_CLIENT_ID"
reject_placeholder CASDOOR_USER_LOOKUP_CLIENT_SECRET "${CASDOOR_USER_LOOKUP_CLIENT_SECRET:-}" "REPLACE_WITH_CASDOOR_USER_LOOKUP_CLIENT_SECRET"
reject_placeholder CASDOOR_USER_LOOKUP_APPLICATION "${CASDOOR_USER_LOOKUP_APPLICATION:-}" "REPLACE_WITH_CASDOOR_USER_LOOKUP_APPLICATION"
reject_placeholder CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_ID "${CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_ID:-}" "REPLACE_WITH_CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_ID"
reject_placeholder CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_SECRET "${CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_SECRET:-}" "REPLACE_WITH_CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_SECRET"
reject_placeholder CASDOOR_TOKEN_PROBE_SMOKE_APPLICATION "${CASDOOR_TOKEN_PROBE_SMOKE_APPLICATION:-}" "REPLACE_WITH_CASDOOR_TOKEN_PROBE_SMOKE_APPLICATION"
reject_placeholder CASDOOR_TOKEN_PROBE_SMOKE_REDIRECT_URI "${CASDOOR_TOKEN_PROBE_SMOKE_REDIRECT_URI:-}" "REPLACE_WITH_CASDOOR_TOKEN_PROBE_SMOKE_REDIRECT_URI"
reject_placeholder OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND "${OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND:-}" "REPLACE_WITH_OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND"
reject_placeholder OPEN_PLATFORM_CONSENT_BASE_URL "${OPEN_PLATFORM_CONSENT_BASE_URL:-}" "REPLACE_WITH_OPEN_PLATFORM_CONSENT_BASE_URL"
reject_placeholder OPEN_PLATFORM_ACCOUNT_BASE_URL "${OPEN_PLATFORM_ACCOUNT_BASE_URL:-}" "REPLACE_WITH_OPEN_PLATFORM_ACCOUNT_BASE_URL"
if [[ "${OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND:-}" == *"casdoor-runtime-token-probe-runner.mjs"* ]]; then
  reject_placeholder CASDOOR_TOKEN_PROBE_USERNAME "${CASDOOR_TOKEN_PROBE_USERNAME:-}" "REPLACE_WITH_CASDOOR_TOKEN_PROBE_USERNAME"
  reject_placeholder CASDOOR_TOKEN_PROBE_PASSWORD "${CASDOOR_TOKEN_PROBE_PASSWORD:-}" "REPLACE_WITH_CASDOOR_TOKEN_PROBE_PASSWORD"
fi
reject_placeholder SMS_SECRET_ID "${SMS_SECRET_ID:-}" "REPLACE_WITH_SMS_SECRET_ID"
reject_placeholder SMS_SECRET_KEY "${SMS_SECRET_KEY:-}" "REPLACE_WITH_SMS_SECRET_KEY"
reject_placeholder SMS_APP_ID "${SMS_APP_ID:-}" "REPLACE_WITH_SMS_APP_ID"
reject_placeholder SMS_SIGN_NAME "${SMS_SIGN_NAME:-}" "REPLACE_WITH_SMS_SIGN_NAME"
reject_placeholder SMS_TEMPLATE_ID "${SMS_TEMPLATE_ID:-}" "REPLACE_WITH_SMS_TEMPLATE_ID"
reject_placeholder SMS_INTERNAL_KEY "${SMS_INTERNAL_KEY:-}" "REPLACE_WITH_SMS_INTERNAL_KEY"
reject_placeholder BOT_SERVICE_TOKEN "${BOT_SERVICE_TOKEN:-}" "REPLACE_WITH_BOT_SERVICE_TOKEN_BOOTSTRAP"
reject_placeholder WEB_PUBLIC_URL "${WEB_PUBLIC_URL:-}" "REPLACE_WITH_WEB_PUBLIC_URL"
reject_placeholder ADMIN_PUBLIC_URL "${ADMIN_PUBLIC_URL:-}" "REPLACE_WITH_ADMIN_PUBLIC_URL"
reject_placeholder ADMISSION_PUBLIC_BASE_URL "${ADMISSION_PUBLIC_BASE_URL:-}" "REPLACE_WITH_ADMISSION_PUBLIC_BASE_URL"
reject_placeholder WEB_VITE_SSO_URL "${WEB_VITE_SSO_URL:-}" "REPLACE_WITH_WEB_VITE_SSO_URL"
reject_placeholder WEB_VITE_WEB_URL "${WEB_VITE_WEB_URL:-}" "REPLACE_WITH_WEB_VITE_WEB_URL"
reject_placeholder OBJECT_STORAGE_ENDPOINT "${OBJECT_STORAGE_ENDPOINT:-}" "REPLACE_WITH_OBJECT_STORAGE_ENDPOINT"
reject_placeholder OBJECT_STORAGE_ACCESS_KEY_ID "${OBJECT_STORAGE_ACCESS_KEY_ID:-}" "REPLACE_WITH_OBJECT_STORAGE_ACCESS_KEY_ID"
reject_placeholder OBJECT_STORAGE_SECRET_ACCESS_KEY "${OBJECT_STORAGE_SECRET_ACCESS_KEY:-}" "REPLACE_WITH_OBJECT_STORAGE_SECRET_ACCESS_KEY"
reject_placeholder BACKUP_OBJECT_STORAGE_ENDPOINT "${BACKUP_OBJECT_STORAGE_ENDPOINT:-}" "REPLACE_WITH_BACKUP_OBJECT_STORAGE_ENDPOINT"
reject_placeholder BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID "${BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID:-}" "REPLACE_WITH_BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID"
reject_placeholder BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY "${BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY:-}" "REPLACE_WITH_BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY"
reject_placeholder GRAFANA_ROOT_URL "${GRAFANA_ROOT_URL:-}" "REPLACE_WITH_GRAFANA_ROOT_URL"
reject_placeholder ALERTMANAGER_WEBHOOK_URL "${ALERTMANAGER_WEBHOOK_URL:-}" "REPLACE_WITH_ALERTMANAGER_WEBHOOK_URL"
reject_placeholder BACKEND_IMAGE_REF "${BACKEND_IMAGE_REF:-}" "REPLACE_WITH_BACKEND_IMAGE_REF"
reject_placeholder FRONTEND_IMAGE_REF "${FRONTEND_IMAGE_REF:-}" "REPLACE_WITH_FRONTEND_IMAGE_REF"
reject_placeholder ADMIN_IMAGE_REF "${ADMIN_IMAGE_REF:-}" "REPLACE_WITH_ADMIN_IMAGE_REF"

if [[ "${ALERTMANAGER_WEBHOOK_URL:-}" == "http://alert-webhook-sink:8080/alerts" && "${ALLOW_LOCAL_ALERT_SINK:-false}" != "true" ]]; then
  die "ALERTMANAGER_WEBHOOK_URL points to the local sink; set ALLOW_LOCAL_ALERT_SINK=true only for local production validation"
fi

[[ "${APP_ENV:-production}" == "production" ]] || die "APP_ENV must be production for production deploy"

reject_local_value CORS_ORIGINS "${CORS_ORIGINS:-}"
reject_local_value CASDOOR_ISSUER "${CASDOOR_ISSUER:-}"
reject_local_value CASDOOR_PUBLIC_AUTH_BASE_URL "${CASDOOR_PUBLIC_AUTH_BASE_URL:-}"
reject_local_value CASDOOR_REDIRECT_URI "${CASDOOR_REDIRECT_URI:-}"
reject_local_value CASDOOR_ADMIN_REDIRECT_URI "${CASDOOR_ADMIN_REDIRECT_URI:-}"
reject_local_value CASDOOR_UNIAPP_REDIRECT_URI "${CASDOOR_UNIAPP_REDIRECT_URI:-}"
reject_local_value CASDOOR_TOKEN_PROBE_SMOKE_REDIRECT_URI "${CASDOOR_TOKEN_PROBE_SMOKE_REDIRECT_URI:-}"
reject_local_value WEB_PUBLIC_URL "${WEB_PUBLIC_URL:-}"
reject_local_value ADMIN_PUBLIC_URL "${ADMIN_PUBLIC_URL:-}"
reject_local_value ADMISSION_PUBLIC_BASE_URL "${ADMISSION_PUBLIC_BASE_URL:-}"
reject_local_value WEB_VITE_SSO_URL "${WEB_VITE_SSO_URL:-}"
reject_local_value WEB_VITE_WEB_URL "${WEB_VITE_WEB_URL:-}"
reject_local_value OPEN_PLATFORM_CONSENT_BASE_URL "${OPEN_PLATFORM_CONSENT_BASE_URL:-}"
reject_local_value OPEN_PLATFORM_ACCOUNT_BASE_URL "${OPEN_PLATFORM_ACCOUNT_BASE_URL:-}"
reject_local_value OPENFGA_API_URL "${OPENFGA_API_URL:-}"
reject_local_value OBJECT_STORAGE_ENDPOINT "${OBJECT_STORAGE_ENDPOINT:-}"
reject_local_value BACKUP_OBJECT_STORAGE_ENDPOINT "${BACKUP_OBJECT_STORAGE_ENDPOINT:-}"
reject_local_value GRAFANA_ROOT_URL "${GRAFANA_ROOT_URL:-}"

require_production_object_storage
"${SCRIPT_DIR}/prepare-object-storage-client-ca.sh"
[[ "${TOKEN_COOKIE_SECURE:-false}" == "true" ]] || die "TOKEN_COOKIE_SECURE must be true for production deploy"
[[ "${ADMISSION_PUBLIC_BASE_URL:-}" == "https://join.stuhelper.com" ]] || die "ADMISSION_PUBLIC_BASE_URL must be exactly https://join.stuhelper.com for production deploy"
[[ "${OTEL_ENABLED:-false}" == "true" ]] || die "OTEL_ENABLED must be true for production deploy"
[[ "${CASDOOR_BOOTSTRAP_ENABLED:-false}" == "true" ]] || die "CASDOOR_BOOTSTRAP_ENABLED must be true for production deploy"
[[ "${CASDOOR_SMS_PROVIDER_ENABLED:-false}" == "true" ]] || die "CASDOOR_SMS_PROVIDER_ENABLED must be true for production deploy"
[[ "${CASDOOR_SMS_PROVIDER_TYPE:-}" == "CustomHTTP" ]] || die "CASDOOR_SMS_PROVIDER_TYPE must be CustomHTTP for production deploy"
[[ "${CASDOOR_SMS_PROVIDER_TITLE:-}" == "content" ]] || die "CASDOOR_SMS_PROVIDER_TITLE must be content for production deploy"
[[ "${SMS_ENABLED:-false}" == "true" ]] || die "SMS_ENABLED must be true for production deploy"
[[ "${OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_REQUIRED:-false}" == "true" ]] || die "OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_REQUIRED must be true for production deploy"
if [[ "${EXTERNAL_POSTGRES_ALLOW_PLAINTEXT:-false}" == "true" ]]; then
  [[ "${EXTERNAL_POSTGRES_ENABLED:-false}" == "true" ]] ||
    die "EXTERNAL_POSTGRES_ENABLED must be true when EXTERNAL_POSTGRES_ALLOW_PLAINTEXT=true"
  warn "external PostgreSQL plaintext transport is explicitly enabled; TLS is not provided by StuHelper"
fi
require_production_postgres_ssl
require_production_external_student_source_security
[[ "${REDIS_TLS_ENABLED:-false}" == "true" ]] || die "REDIS_TLS_ENABLED must be true for production deploy"
require_public_ingress_config_preflight
require_public_identity_ingress_preflight

export TAG="${TAG:-$(derive_release_id_from_image_ref "${BACKEND_IMAGE_REF:-}" || git_tag_default)}"
export BUILD_TIME="${BUILD_TIME:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"

if [[ "${EXTERNAL_POSTGRES_ENABLED:-false}" != "true" ]]; then
  "${SCRIPT_DIR}/render-postgres-tls.sh"
else
  log "external PostgreSQL selected; skipping local PostgreSQL server certificate generation"
fi
"${SCRIPT_DIR}/render-redis-tls.sh"
"${SCRIPT_DIR}/prepare-datastore-client-cas.sh"
"${SCRIPT_DIR}/render-redis-acl.sh"
"${SCRIPT_DIR}/render-observability.sh" prod

log "pulling immutable production images for release ${TAG}"
compose --profile prod pull app frontend admin

infra_services=(
  docker-socket-proxy
  alloy
  alertmanager
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
datastore_services=(
  redis
)
if [[ "${EXTERNAL_POSTGRES_ENABLED:-false}" != "true" ]]; then
  datastore_services=(postgres "${datastore_services[@]}")
fi

authz_services=(
  openfga
)

log "starting production datastore services"
compose --profile prod up -d --wait "${datastore_services[@]}"
if [[ "${EXTERNAL_POSTGRES_ENABLED:-false}" != "true" ]]; then
  "${SCRIPT_DIR}/ensure-postgres-monitoring-role.sh"
else
  log "external PostgreSQL selected; expecting a pre-provisioned stuhelper_metrics role with pg_monitor"
fi

log "starting production observability infrastructure services"
compose --profile prod up -d --wait "${infra_services[@]}"

log "creating pre-deploy database backup"
predeploy_backup_dir="${REPO_ROOT}/backups/postgres/logical"
predeploy_backup_path="${predeploy_backup_dir}/predeploy-${TAG}.dump"
mkdir -p "${predeploy_backup_dir}"
"${SCRIPT_DIR}/backup-postgres.sh" "${predeploy_backup_path}" || die "pre-deploy backup failed; aborting deployment"

log "syncing pre-deploy PostgreSQL backup artifacts to object storage"
"${SCRIPT_DIR}/sync-postgres-backups.sh"

log "verifying PostgreSQL backup evidence"
POSTGRES_BACKUP_EVIDENCE_TIMER_REQUIRED=false "${SCRIPT_DIR}/postgres-backup-evidence.sh"

log "running production database migrations"
compose --profile prod up --no-deps migrate
compose --profile prod up --no-deps openfga-migrate

log "checking admission production readiness"
"${SCRIPT_DIR}/admission-production-readiness.sh"

log "starting production authorization services"
compose --profile prod up -d --wait "${authz_services[@]}"

log "bootstrapping runtime identities against external Casdoor and local OpenFGA"
"${SCRIPT_DIR}/bootstrap-platform.sh" prod

log "running Open Platform production evidence smokes"
"${SCRIPT_DIR}/open-platform-production-evidence.sh"

log "starting production application services"
compose --profile prod up -d --wait app frontend admin

if [[ "${SSO_PUBLIC_SMOKE_ENABLED:-true}" == "true" ]]; then
  "${SCRIPT_DIR}/sso-public-smoke.sh"
else
  warn "public SSO smoke skipped because SSO_PUBLIC_SMOKE_ENABLED is not true"
fi
if [[ "${ADMISSION_PUBLIC_SMOKE_ENABLED:-true}" == "true" ]]; then
  "${SCRIPT_DIR}/admission-public-smoke.sh"
else
  warn "public admission smoke skipped because ADMISSION_PUBLIC_SMOKE_ENABLED is not true"
fi
if [[ "${PUBLIC_WEB_AUTH_BROWSER_SMOKE_ENABLED:-true}" == "true" ]]; then
  node "${SCRIPT_DIR}/public-web-auth-browser-smoke.mjs"
else
  warn "public Web auth browser smoke skipped because PUBLIC_WEB_AUTH_BROWSER_SMOKE_ENABLED is not true"
fi
"${SCRIPT_DIR}/smoke-check.sh"
OBS_SMOKE_STRICT=true "${SCRIPT_DIR}/observability-smoke-check.sh"
record_release "${TAG}"

log "production deployment completed successfully"
echo "  Web:     ${WEB_PUBLIC_URL}"
echo "  Admin UI:${ADMIN_PUBLIC_URL:-http://127.0.0.1:${ADMIN_EXTERNAL_PORT:-18001}}"
echo "  SSO:     ${WEB_VITE_SSO_URL}"
echo "  Release: ${TAG}"
echo "  Backend: ${BACKEND_IMAGE_REF}"
echo "  Frontend:${FRONTEND_IMAGE_REF}"
echo "  AdminImg:${ADMIN_IMAGE_REF}"
