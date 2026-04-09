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
      die "${key} is using placeholder/default value (${placeholder}); set a real production secret first"
    fi
  done
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
require_nonempty ZITADEL_REDIRECT_URI "${ZITADEL_REDIRECT_URI:-}"
require_nonempty WEB_PUBLIC_URL "${WEB_PUBLIC_URL:-}"
require_nonempty WEB_VITE_SSO_URL "${WEB_VITE_SSO_URL:-}"
require_nonempty OPENFGA_API_URL "${OPENFGA_API_URL:-}"
require_nonempty OBJECT_STORAGE_ENDPOINT "${OBJECT_STORAGE_ENDPOINT:-}"
require_nonempty OBJECT_STORAGE_BUCKET "${OBJECT_STORAGE_BUCKET:-}"
require_nonempty OBJECT_STORAGE_ACCESS_KEY_ID "${OBJECT_STORAGE_ACCESS_KEY_ID:-}"
require_nonempty OBJECT_STORAGE_SECRET_ACCESS_KEY "${OBJECT_STORAGE_SECRET_ACCESS_KEY:-}"

reject_placeholder POSTGRES_PASSWORD "${POSTGRES_PASSWORD:-}" "dev123"
reject_placeholder REDIS_PASSWORD "${REDIS_PASSWORD:-}" "dev123"
reject_placeholder ZITADEL_ADMIN_PASSWORD "${ZITADEL_ADMIN_PASSWORD:-}" "Admin1234!" "REPLACE_WITH_ZITADEL_ADMIN_PASSWORD"
reject_placeholder ZITADEL_MASTERKEY "${ZITADEL_MASTERKEY:-}" "StuHelperDevMasterKey123456789AB" "REPLACE_WITH_ZITADEL_MASTERKEY_32_CHARS"
reject_placeholder LOGIN_CLIENT_PAT_EXPIRATION "${LOGIN_CLIENT_PAT_EXPIRATION:-}" "REPLACE_WITH_LOGIN_CLIENT_PAT_EXPIRATION_ISO8601"
reject_placeholder GRAFANA_ADMIN_PASSWORD "${GRAFANA_ADMIN_PASSWORD:-}" "ChangeMeBeforeProduction"

if [[ "${ALERTMANAGER_WEBHOOK_URL:-}" == "http://alert-webhook-sink:8080/alerts" && "${ALLOW_LOCAL_ALERT_SINK:-false}" != "true" ]]; then
  die "ALERTMANAGER_WEBHOOK_URL points to the local sink; set ALLOW_LOCAL_ALERT_SINK=true only for local production validation"
fi

[[ "${TOKEN_COOKIE_SECURE:-false}" == "true" ]] || die "TOKEN_COOKIE_SECURE must be true for production deploy"
[[ "${OTEL_ENABLED:-false}" == "true" ]] || die "OTEL_ENABLED must be true for production deploy"
[[ "${DB_SSL_MODE:-disable}" != "disable" ]] || die "DB_SSL_MODE must not be disable for production deploy"

export TAG="${TAG:-$(git_tag_default)}"
export BUILD_TIME="${BUILD_TIME:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"

"${SCRIPT_DIR}/render-postgres-tls.sh"
"${SCRIPT_DIR}/render-zitadel-secrets.sh"
"${SCRIPT_DIR}/render-observability.sh" prod

if [[ "${SKIP_BUILD:-false}" == "true" ]]; then
  log "pulling production images with TAG=${TAG}"
  compose --profile prod pull app frontend admin
else
  log "building production images with TAG=${TAG}"
  compose --profile prod build app frontend admin
fi

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
echo "  Admin:   ${ADMIN_PUBLIC_URL:-http://127.0.0.1:${ADMIN_EXTERNAL_PORT:-3001}}"
echo "  SSO:     ${WEB_VITE_SSO_URL}"
echo "  Tag:     ${TAG}"
