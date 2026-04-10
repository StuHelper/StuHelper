#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

require_cmd docker
require_cmd curl
require_cmd jq
require_cmd python3

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

log "starting development base services"
compose up -d --wait "${base_services[@]}"
compose up --no-deps minio-init

log "bootstrapping platform identities and authorization model"
"${SCRIPT_DIR}/bootstrap-platform.sh" dev

if [[ "${WITH_OBSERVABILITY:-false}" == "true" ]]; then
  log "starting observability stack for development"
  compose --profile observability up -d --wait alloy prometheus alertmanager loki tempo grafana node-exporter cadvisor postgres-exporter redis-exporter blackbox-exporter
fi

log "starting full dockerized development stack"
compose --profile dev-full up -d --wait app-dev frontend-dev admin-dev

wait_for_http "backend" "http://127.0.0.1:8080/health/ready" 120 2
wait_for_http "frontend" "http://127.0.0.1:3000/" 180 2
wait_for_http "admin" "http://127.0.0.1:${ADMIN_EXTERNAL_PORT:-3001}/admin/" 240 2

log "dockerized development stack is ready"
echo "  Web:        http://127.0.0.1:3000"
echo "  Admin:      http://127.0.0.1:${ADMIN_EXTERNAL_PORT:-3001}/admin/"
echo "  Backend:    http://127.0.0.1:8080"
echo "  Zitadel:    http://127.0.0.1:${ZITADEL_EXTERNALPORT:-8085}"
echo "  Generated:  ${GENERATED_ENV_FILE}"
