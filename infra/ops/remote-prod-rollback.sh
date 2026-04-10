#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

require_cmd docker

[[ -n "${REGISTRY:-}" ]] || die "REGISTRY is required"
[[ -n "${REGISTRY_USERNAME:-}" ]] || die "REGISTRY_USERNAME is required"
[[ -n "${REGISTRY_PASSWORD:-}" ]] || die "REGISTRY_PASSWORD is required"

echo "${REGISTRY_PASSWORD}" | docker login "${REGISTRY}" --username "${REGISTRY_USERNAME}" --password-stdin >/dev/null

"${SCRIPT_DIR}/prod-rollback.sh"

docker image prune -f >/dev/null 2>&1 || true
log "remote production rollback finished"
