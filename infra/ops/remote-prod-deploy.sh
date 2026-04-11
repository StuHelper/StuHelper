#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

require_cmd docker
require_cmd curl
require_cmd jq
require_cmd python3
require_cmd openssl

resolve_registry_credentials() {
  if [[ -z "${REGISTRY_USERNAME:-}" && -n "${REGISTRY_USERNAME_SECRET_REF:-}" ]]; then
    REGISTRY_USERNAME="$(materialize_secret_value "${REGISTRY_USERNAME_SECRET_REF}")"
    export REGISTRY_USERNAME
  fi
  if [[ -z "${REGISTRY_PASSWORD:-}" && -n "${REGISTRY_PASSWORD_SECRET_REF:-}" ]]; then
    REGISTRY_PASSWORD="$(materialize_secret_value "${REGISTRY_PASSWORD_SECRET_REF}")"
    export REGISTRY_PASSWORD
  fi
}

[[ -n "${REGISTRY:-}" ]] || die "REGISTRY is required"
resolve_registry_credentials
[[ -n "${REGISTRY_USERNAME:-}" ]] || die "REGISTRY_USERNAME or REGISTRY_USERNAME_SECRET_REF is required"
[[ -n "${REGISTRY_PASSWORD:-}" ]] || die "REGISTRY_PASSWORD or REGISTRY_PASSWORD_SECRET_REF is required"

echo "${REGISTRY_PASSWORD}" | docker login "${REGISTRY}" --username "${REGISTRY_USERNAME}" --password-stdin >/dev/null

"${SCRIPT_DIR}/prod-deploy.sh"

docker image prune -f >/dev/null 2>&1 || true
log "remote production deploy finished"
