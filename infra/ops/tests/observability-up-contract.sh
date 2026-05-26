#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
OBS_UP="${REPO_ROOT}/infra/ops/observability-up.sh"
DEV_LOCAL="${REPO_ROOT}/infra/ops/lib/dev-local.sh"
OBS_SMOKE="${REPO_ROOT}/infra/ops/observability-smoke-check.sh"

fail() {
  echo "[observability-up-contract][error] $*" >&2
  exit 1
}

assert_contains() {
  local file="$1"
  local pattern="$2"
  if ! grep -Eq -- "${pattern}" "${file}"; then
    fail "expected ${file} to contain pattern: ${pattern}"
  fi
}

[[ -f "${OBS_UP}" ]] || fail "missing observability up script: ${OBS_UP}"
[[ -f "${DEV_LOCAL}" ]] || fail "missing dev-local library: ${DEV_LOCAL}"
[[ -f "${OBS_SMOKE}" ]] || fail "missing observability smoke script: ${OBS_SMOKE}"

bash -n "${OBS_UP}"
bash -n "${DEV_LOCAL}"
bash -n "${OBS_SMOKE}"

assert_contains "${OBS_UP}" 'source "\$\{SCRIPT_DIR\}/lib/dev-local\.sh"'
assert_contains "${OBS_UP}" '"\$\{SCRIPT_DIR\}/init-dev-env\.sh"'
assert_contains "${OBS_UP}" 'sync_dev_observability_ports'
assert_contains "${DEV_LOCAL}" 'ALLOY_HTTP_PORT_SELECTED="\$\(pick_available_or_current_container_port "\$\{ALLOY_HTTP_PORT:-12345\}" 30 "\$\{stack\}-alloy"'
assert_contains "${DEV_LOCAL}" 'CADVISOR_PORT_SELECTED="\$\(pick_available_or_current_container_port "\$\{CADVISOR_PORT:-8088\}" 30 "\$\{stack\}-cadvisor"'
assert_contains "${DEV_LOCAL}" 'POSTGRES_EXPORTER_PORT_SELECTED="\$\(pick_available_or_current_container_port "\$\{POSTGRES_EXPORTER_PORT:-9187\}" 30 "\$\{stack\}-postgres-exporter"'
assert_contains "${DEV_LOCAL}" 'REDIS_EXPORTER_PORT_SELECTED="\$\(pick_available_or_current_container_port "\$\{REDIS_EXPORTER_PORT:-9121\}" 30 "\$\{stack\}-redis-exporter"'
assert_contains "${DEV_LOCAL}" 'BLACKBOX_EXPORTER_PORT_SELECTED="\$\(pick_available_or_current_container_port "\$\{BLACKBOX_EXPORTER_PORT:-9115\}" 30 "\$\{stack\}-blackbox-exporter"'
assert_contains "${DEV_LOCAL}" 'upsert_env_file "\$\{ENV_FILE\}" "GRAFANA_ROOT_URL" "http://127\.0\.0\.1:\$\{GRAFANA_PORT_SELECTED\}"'
assert_contains "${OBS_SMOKE}" 'load_env'
assert_contains "${OBS_SMOKE}" 'PROM_READY_URL="\$\{PROMETHEUS_URL:-http://127\.0\.0\.1:\$\{PROMETHEUS_PORT:-9090\}/-/ready\}"'
assert_contains "${OBS_SMOKE}" 'GRAFANA_HEALTH_URL="\$\{GRAFANA_URL:-http://127\.0\.0\.1:\$\{GRAFANA_PORT:-3003\}/api/health\}"'
assert_contains "${OBS_SMOKE}" 'ALLOY_READY_URL="\$\{ALLOY_URL:-http://127\.0\.0\.1:\$\{ALLOY_HTTP_PORT:-12345\}/-/ready\}"'

echo "[observability-up-contract] all assertions passed"
