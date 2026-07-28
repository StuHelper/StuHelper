#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"
# shellcheck source=lib/dev-local.sh
source "${SCRIPT_DIR}/lib/dev-local.sh"

if [[ ! -f "${ENV_FILE}" ]]; then
  warn "development environment is not initialized; run make dev-init before starting the stack"
fi

set -a
if [[ -f "${ENV_FILE}" ]]; then
  source_env_file "${ENV_FILE}"
fi
if [[ -f "${GENERATED_ENV_FILE}" ]]; then
  source_env_file "${GENERATED_ENV_FILE}"
fi
if [[ -f "${DEV_RUNTIME_ENV}" ]]; then
  source_env_file "${DEV_RUNTIME_ENV}"
fi
set +a

report_process() {
  local name="$1"
  local pidfile
  pidfile="$(pid_file "${name}")"
  if [[ -f "${pidfile}" ]] && process_running "$(cat "${pidfile}")"; then
    printf '%-10s running (pid=%s) log=%s\n' "${name}" "$(cat "${pidfile}")" "$(log_file "${name}")"
  else
    printf '%-10s stopped\n' "${name}"
  fi
}

echo "[stuhelper] managed local dev processes"
report_process backend
report_process frontend
report_process admin
report_process koishi
if [[ -n "${WEB_BASE_URL:-}" ]]; then
  printf '\nweb url:    %s\nadmin url:  %s%s\nbackend:    %s\nkoishi:     %s\n' "${WEB_BASE_URL}" "${ADMIN_BASE_URL:-}" "${ADMIN_SMOKE_PATH:-/admin/}" "${API_BASE_URL:-http://127.0.0.1:8080}" "${KOISHI_BASE_URL:-http://127.0.0.1:5140}"
fi
echo
echo "[stuhelper] docker services"
if ! command -v docker >/dev/null 2>&1; then
  warn "docker command is unavailable"
  exit 0
fi
if ! docker info >/dev/null 2>&1; then
  warn "docker daemon is unavailable"
  exit 0
fi

compose_project="${COMPOSE_PROJECT_NAME:-${STACK_NAME:-stuhelper-dev}}"
docker ps -a \
  --filter "label=com.docker.compose.project=${compose_project}" \
  --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}'
