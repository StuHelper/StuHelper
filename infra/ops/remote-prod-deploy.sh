#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

load_remote_deploy_config
require_cmd docker
require_cmd curl
require_cmd jq
require_cmd python3
require_cmd openssl

docker_registry_login

"${SCRIPT_DIR}/prod-deploy.sh"

docker image prune -f >/dev/null 2>&1 || true
log "remote production deploy finished"
