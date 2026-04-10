#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"
# shellcheck source=lib/dev-local.sh
source "${SCRIPT_DIR}/lib/dev-local.sh"

require_cmd docker
require_cmd curl
require_cmd jq
require_cmd python3
require_cmd go

ensure_dev_runtime_dirs
ensure_node_toolchain
"${SCRIPT_DIR}/init-dev-env.sh"
load_env
"${SCRIPT_DIR}/render-observability.sh" dev

log "building Zitadel runtime images"
build_zitadel_runtime_images

base_services=(
  postgres
  redis
  minio
  migrate
  seed-dev
  zitadel-api
  zitadel-login
  proxy
  openfga-migrate
  openfga
)

log "starting development infrastructure (Docker)"
compose up -d "${base_services[@]}"
compose up --no-deps minio-init

log "bootstrapping platform identities and authorization model"
"${SCRIPT_DIR}/bootstrap-platform.sh" dev

log "ensuring dockerized dev app containers are stopped"
compose --profile dev-full stop app-dev frontend-dev admin-dev >/dev/null 2>&1 || true
compose --profile dev-full rm -f app-dev frontend-dev admin-dev >/dev/null 2>&1 || true
kill_all_dev_processes

if [[ "${WITH_OBSERVABILITY:-false}" == "true" ]]; then
  log "starting observability stack for development"
  compose --profile observability up -d --wait alloy prometheus alertmanager loki tempo grafana node-exporter cadvisor postgres-exporter redis-exporter blackbox-exporter
fi

ensure_pnpm_workspace \
  "${REPO_ROOT}/clients" \
  "${REPO_ROOT}/clients/pnpm-lock.yaml" \
  "clients-root" \
  "corepack enable >/dev/null 2>&1 || true && corepack prepare pnpm@10 --activate >/dev/null 2>&1 && pnpm install --frozen-lockfile"

ensure_pnpm_workspace \
  "${REPO_ROOT}/clients/admin" \
  "${REPO_ROOT}/clients/admin/pnpm-lock.yaml" \
  "clients-admin" \
  "corepack enable >/dev/null 2>&1 || true && corepack prepare pnpm@10 --activate >/dev/null 2>&1 && CI=1 pnpm install --frozen-lockfile"

AIR_BIN="$(ensure_air)"
WEB_DEV_PORT_SELECTED="$(pick_available_port "${WEB_DEV_PORT:-3000}" 30)"
ADMIN_DEV_PORT_SELECTED="$(pick_available_port "${ADMIN_EXTERNAL_PORT:-3001}" 30 "${WEB_DEV_PORT_SELECTED}")"
write_dev_runtime_env "${WEB_DEV_PORT_SELECTED}" "${ADMIN_DEV_PORT_SELECTED}"

wait_for_http "Zitadel OIDC metadata" "http://localhost:${ZITADEL_EXTERNALPORT:-8085}/.well-known/openid-configuration" 90 2

backend_cmd="
  cd '${REPO_ROOT}/server' && \
  set -a && source '${ENV_FILE}' && source '${GENERATED_ENV_FILE}' && set +a && \
  export APP_ENV=development && \
  export DATABASE_URL='postgresql://${STUHELPER_APP_DB_USER:-stuhelper_app}:${STUHELPER_APP_DB_PASSWORD}@localhost:5432/${POSTGRES_DB:-stuhelper}?sslmode=disable' && \
  export REDIS_HOST='localhost' && \
  export REDIS_PORT='6379' && \
  export REDIS_PASSWORD='${REDIS_PASSWORD}' && \
  export ZITADEL_INTERNAL_ADDRESS='' && \
  export OPENFGA_API_URL='http://localhost:8081' && \
  exec '${AIR_BIN}' -c .air.toml
"

frontend_cmd="
  cd '${REPO_ROOT}/clients' && \
  export NODE_ENV=development && \
  export VITE_API_URL='' && \
  export VITE_SSO_URL='${WEB_VITE_SSO_URL:-http://localhost:8085}' && \
  export VITE_API_TIMEOUT_MS='${WEB_VITE_API_TIMEOUT_MS:-15000}' && \
  export VITE_DEV_PROXY_TARGET='http://127.0.0.1:8080' && \
  exec pnpm --filter @stuhelper/web exec vite --host 127.0.0.1 --strictPort --port ${WEB_DEV_PORT_SELECTED}
"

admin_cmd="
  cd '${REPO_ROOT}/clients/admin' && \
  export NODE_ENV=development && \
  export VITE_DEV_PROXY_TARGET='http://127.0.0.1:8080' && \
  export VITE_GLOB_API_URL='${ADMIN_VITE_API_URL:-/api/v1}' && \
  export VITE_BASE='${ADMIN_VITE_BASE:-/admin/}' && \
  exec pnpm -F @vben/web-ele exec vite --mode development --host 127.0.0.1 --strictPort --port ${ADMIN_DEV_PORT_SELECTED}
"

log "starting hot-reload backend (air)"
start_managed_process backend "${backend_cmd}"
wait_for_http "backend" "http://127.0.0.1:8080/health/ready" 120 2

log "starting hot-reload frontend (Vite)"
start_managed_process frontend "${frontend_cmd}"
wait_for_http "frontend" "http://127.0.0.1:${WEB_DEV_PORT_SELECTED}/" 180 2

log "starting hot-reload admin (Vite)"
start_managed_process admin "${admin_cmd}"
wait_for_http "admin" "http://127.0.0.1:${ADMIN_DEV_PORT_SELECTED}/admin/" 240 2

log "development stack is ready"
echo "  Web:        http://localhost:${WEB_DEV_PORT_SELECTED}"
echo "  Admin:      http://localhost:${ADMIN_DEV_PORT_SELECTED}/admin/"
echo "  Backend:    http://localhost:8080"
echo "  Zitadel:    http://localhost:${ZITADEL_EXTERNALPORT:-8085}"
echo "  Logs:       ${DEV_LOG_DIR}"
echo "  Generated:  ${GENERATED_ENV_FILE}"
