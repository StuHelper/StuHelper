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

[[ -f "${ENV_FILE}" ]] || die "missing ${ENV_FILE}; production deploy requires a prepared .env"
if [[ -n "${SECRETS_ENV_FILE:-}" ]]; then
  [[ -f "${SECRETS_ENV_FILE}" ]] || die "missing ${SECRETS_ENV_FILE}; production deploy requires a prepared secrets env file"
fi
ensure_generated_files
load_env

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

require_nonempty POSTGRES_PASSWORD "${POSTGRES_PASSWORD:-}"
require_nonempty REDIS_PASSWORD "${REDIS_PASSWORD:-}"
require_nonempty METRICS_PASSWORD "${METRICS_PASSWORD:-}"
require_nonempty GRAFANA_ADMIN_PASSWORD "${GRAFANA_ADMIN_PASSWORD:-}"
require_nonempty ALERTMANAGER_WEBHOOK_URL "${ALERTMANAGER_WEBHOOK_URL:-}"
require_nonempty TRUSTED_PROXIES "${TRUSTED_PROXIES:-}"
require_nonempty CORS_ORIGINS "${CORS_ORIGINS:-}"
require_nonempty HMAC_SECRET "${HMAC_SECRET:-}"
require_nonempty DOC_AES_KEYS "${DOC_AES_KEYS:-}"
require_nonempty ZITADEL_DOMAIN "${ZITADEL_DOMAIN:-}"
require_nonempty ZITADEL_ISSUER "${ZITADEL_ISSUER:-}"
require_nonempty ZITADEL_REDIRECT_URI "${ZITADEL_REDIRECT_URI:-}"
require_nonempty WEB_PUBLIC_URL "${WEB_PUBLIC_URL:-}"
require_nonempty ADMIN_PUBLIC_URL "${ADMIN_PUBLIC_URL:-}"
require_nonempty WEB_VITE_SSO_URL "${WEB_VITE_SSO_URL:-}"
require_nonempty OPENFGA_API_URL "${OPENFGA_API_URL:-}"
require_nonempty OBJECT_STORAGE_ENDPOINT "${OBJECT_STORAGE_ENDPOINT:-}"
require_nonempty OBJECT_STORAGE_BUCKET "${OBJECT_STORAGE_BUCKET:-}"
require_nonempty OBJECT_STORAGE_ACCESS_KEY_ID "${OBJECT_STORAGE_ACCESS_KEY_ID:-}"
require_nonempty OBJECT_STORAGE_SECRET_ACCESS_KEY "${OBJECT_STORAGE_SECRET_ACCESS_KEY:-}"
require_nonempty GRAFANA_ROOT_URL "${GRAFANA_ROOT_URL:-}"
require_immutable_image_ref BACKEND_IMAGE_REF "${BACKEND_IMAGE_REF:-}"
require_immutable_image_ref FRONTEND_IMAGE_REF "${FRONTEND_IMAGE_REF:-}"
require_immutable_image_ref ADMIN_IMAGE_REF "${ADMIN_IMAGE_REF:-}"

reject_placeholder POSTGRES_PASSWORD "${POSTGRES_PASSWORD:-}" "dev123"
reject_placeholder REDIS_PASSWORD "${REDIS_PASSWORD:-}" "dev123"
reject_placeholder ZITADEL_ADMIN_PASSWORD "${ZITADEL_ADMIN_PASSWORD:-}" "Admin1234!" "REPLACE_WITH_ZITADEL_ADMIN_PASSWORD"
reject_placeholder ZITADEL_MASTERKEY "${ZITADEL_MASTERKEY:-}" "StuHelperDevMasterKey123456789AB" "REPLACE_WITH_ZITADEL_MASTERKEY_32_CHARS"
reject_placeholder LOGIN_CLIENT_PAT_EXPIRATION "${LOGIN_CLIENT_PAT_EXPIRATION:-}" "REPLACE_WITH_LOGIN_CLIENT_PAT_EXPIRATION_ISO8601"
reject_placeholder GRAFANA_ADMIN_PASSWORD "${GRAFANA_ADMIN_PASSWORD:-}" "ChangeMeBeforeProduction"
reject_placeholder CORS_ORIGINS "${CORS_ORIGINS:-}" "REPLACE_WITH_PRODUCTION_CORS_ORIGINS"
reject_placeholder ZITADEL_DOMAIN "${ZITADEL_DOMAIN:-}" "REPLACE_WITH_ZITADEL_DOMAIN"
reject_placeholder ZITADEL_ISSUER "${ZITADEL_ISSUER:-}" "REPLACE_WITH_ZITADEL_ISSUER"
reject_placeholder ZITADEL_REDIRECT_URI "${ZITADEL_REDIRECT_URI:-}" "REPLACE_WITH_ZITADEL_REDIRECT_URI"
reject_placeholder WEB_PUBLIC_URL "${WEB_PUBLIC_URL:-}" "REPLACE_WITH_WEB_PUBLIC_URL"
reject_placeholder ADMIN_PUBLIC_URL "${ADMIN_PUBLIC_URL:-}" "REPLACE_WITH_ADMIN_PUBLIC_URL"
reject_placeholder WEB_VITE_SSO_URL "${WEB_VITE_SSO_URL:-}" "REPLACE_WITH_WEB_VITE_SSO_URL"
reject_placeholder OBJECT_STORAGE_ENDPOINT "${OBJECT_STORAGE_ENDPOINT:-}" "REPLACE_WITH_OBJECT_STORAGE_ENDPOINT"
reject_placeholder OBJECT_STORAGE_ACCESS_KEY_ID "${OBJECT_STORAGE_ACCESS_KEY_ID:-}" "REPLACE_WITH_OBJECT_STORAGE_ACCESS_KEY_ID"
reject_placeholder GRAFANA_ROOT_URL "${GRAFANA_ROOT_URL:-}" "REPLACE_WITH_GRAFANA_ROOT_URL"
reject_placeholder ALERTMANAGER_WEBHOOK_URL "${ALERTMANAGER_WEBHOOK_URL:-}" "REPLACE_WITH_ALERTMANAGER_WEBHOOK_URL"
reject_placeholder BACKEND_IMAGE_REF "${BACKEND_IMAGE_REF:-}" "REPLACE_WITH_BACKEND_IMAGE_REF"
reject_placeholder FRONTEND_IMAGE_REF "${FRONTEND_IMAGE_REF:-}" "REPLACE_WITH_FRONTEND_IMAGE_REF"
reject_placeholder ADMIN_IMAGE_REF "${ADMIN_IMAGE_REF:-}" "REPLACE_WITH_ADMIN_IMAGE_REF"

if [[ "${ALERTMANAGER_WEBHOOK_URL:-}" == "http://alert-webhook-sink:8080/alerts" && "${ALLOW_LOCAL_ALERT_SINK:-false}" != "true" ]]; then
  die "ALERTMANAGER_WEBHOOK_URL points to the local sink; set ALLOW_LOCAL_ALERT_SINK=true only for local production validation"
fi

reject_local_value CORS_ORIGINS "${CORS_ORIGINS:-}"
reject_local_value ZITADEL_DOMAIN "${ZITADEL_DOMAIN:-}"
reject_local_value ZITADEL_ISSUER "${ZITADEL_ISSUER:-}"
reject_local_value ZITADEL_REDIRECT_URI "${ZITADEL_REDIRECT_URI:-}"
reject_local_value WEB_PUBLIC_URL "${WEB_PUBLIC_URL:-}"
reject_local_value ADMIN_PUBLIC_URL "${ADMIN_PUBLIC_URL:-}"
reject_local_value WEB_VITE_SSO_URL "${WEB_VITE_SSO_URL:-}"
reject_local_value OPENFGA_API_URL "${OPENFGA_API_URL:-}"
reject_local_value OBJECT_STORAGE_ENDPOINT "${OBJECT_STORAGE_ENDPOINT:-}"
reject_local_value GRAFANA_ROOT_URL "${GRAFANA_ROOT_URL:-}"

[[ "${TOKEN_COOKIE_SECURE:-false}" == "true" ]] || die "TOKEN_COOKIE_SECURE must be true for production deploy"
[[ "${OTEL_ENABLED:-false}" == "true" ]] || die "OTEL_ENABLED must be true for production deploy"
[[ "${DB_SSL_MODE:-disable}" != "disable" ]] || die "DB_SSL_MODE must not be disable for production deploy"

export TAG="${TAG:-$(derive_release_id_from_image_ref "${BACKEND_IMAGE_REF:-}" || git_tag_default)}"
export BUILD_TIME="${BUILD_TIME:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"

"${SCRIPT_DIR}/render-postgres-tls.sh"
"${SCRIPT_DIR}/render-zitadel-secrets.sh"
"${SCRIPT_DIR}/render-observability.sh" prod

log "pulling immutable production images for release ${TAG}"
compose --profile prod pull app frontend admin

infra_services=(
  postgres
  redis
  minio
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

authz_services=(
  zitadel-api
  zitadel-login
  proxy
  openfga
)

log "starting production infrastructure services"
compose --profile prod up -d --wait "${infra_services[@]}"
compose --profile prod up --no-deps minio-init

log "running production database migrations"
compose --profile prod up --no-deps migrate
compose --profile prod up --no-deps openfga-migrate

log "starting production identity and authorization services"
compose --profile prod up -d --wait "${authz_services[@]}"

log "bootstrapping runtime identities and authorization"
"${SCRIPT_DIR}/bootstrap-platform.sh" prod

log "starting production application services"
compose --profile prod up -d --wait app frontend admin

"${SCRIPT_DIR}/smoke-check.sh"
"${SCRIPT_DIR}/observability-smoke-check.sh"
record_release "${TAG}"

log "production deployment completed successfully"
echo "  Web:     ${WEB_PUBLIC_URL}"
echo "  Admin UI:${ADMIN_PUBLIC_URL:-http://127.0.0.1:${ADMIN_EXTERNAL_PORT:-3001}}"
echo "  SSO:     ${WEB_VITE_SSO_URL}"
echo "  Release: ${TAG}"
echo "  Backend: ${BACKEND_IMAGE_REF}"
echo "  Frontend:${FRONTEND_IMAGE_REF}"
echo "  AdminImg:${ADMIN_IMAGE_REF}"
