#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"
# shellcheck source=lib/dev-local.sh
source "${SCRIPT_DIR}/lib/dev-local.sh"

"${SCRIPT_DIR}/init-dev-env.sh"
load_env
if [[ -f "${DEV_RUNTIME_ENV}" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "${DEV_RUNTIME_ENV}"
  set +a
fi

if [[ -z "${GRAFANA_URL:-}" ]]; then
  dev_grafana_url="http://127.0.0.1:${GRAFANA_PORT:-3003}"
  if curl --fail --silent --show-error --max-time 2 "${dev_grafana_url}/api/health" >/dev/null 2>&1; then
    export GRAFANA_URL="${dev_grafana_url}"
  fi
fi

"${SCRIPT_DIR}/smoke-check.sh"

if [[ "${DEV_BROWSER_SMOKE:-true}" == "true" ]]; then
  node "${SCRIPT_DIR}/dev-browser-smoke.mjs"
fi
