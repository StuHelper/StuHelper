#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

if [[ -n "${ROLLBACK_TAG:-}" ]]; then
  require_safe_release_tag "${ROLLBACK_TAG}"
fi

load_remote_deploy_config
require_cmd docker

docker_registry_login

"${SCRIPT_DIR}/prod-rollback.sh"

docker image prune -f >/dev/null 2>&1 || true
log "remote production rollback finished"
