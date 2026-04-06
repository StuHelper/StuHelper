#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

require_cmd docker
require_cmd python3

"${SCRIPT_DIR}/render-observability.sh" observability

compose --profile observability up -d --wait alloy prometheus alertmanager loki tempo grafana node-exporter cadvisor postgres-exporter redis-exporter blackbox-exporter
"${SCRIPT_DIR}/observability-smoke-check.sh"

log "observability stack is ready"
