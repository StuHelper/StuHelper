#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"
# shellcheck source=lib/dev-local.sh
source "${SCRIPT_DIR}/lib/dev-local.sh"

"${SCRIPT_DIR}/init-dev-env.sh"
ensure_dev_runtime_dirs
if [[ -f "${DEV_RUNTIME_ENV}" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "${DEV_RUNTIME_ENV}"
  set +a
fi

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
if [[ -n "${WEB_BASE_URL:-}" ]]; then
  printf '\nweb url:    %s\nadmin url:  %s%s\nbackend:    %s\n' "${WEB_BASE_URL}" "${ADMIN_BASE_URL:-}" "${ADMIN_SMOKE_PATH:-/admin/}" "${API_BASE_URL:-http://127.0.0.1:8080}"
fi
echo
echo "[stuhelper] docker services"
compose ps
