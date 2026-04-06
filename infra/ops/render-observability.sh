#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

MODE="${1:-observability}"
if [[ "${MODE}" != "dev" && "${MODE}" != "observability" && "${MODE}" != "prod" ]]; then
  die "usage: $0 [dev|observability|prod]"
fi

require_cmd python3
load_env

python3 "${SCRIPT_DIR}/render-observability-configs.py" --mode "${MODE}" >/dev/null
log "rendered observability configs for mode=${MODE}"
