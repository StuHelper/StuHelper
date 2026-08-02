#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

operation="${1:-}"
case "${operation}" in
  deploy | rollback) ;;
  *) die "usage: $0 deploy|rollback" ;;
esac

load_remote_deploy_config
require_cmd docker

[[ "${REGISTRY_AUTH_MODE:-}" == "workflow-token" ]] ||
  die "remote CI releases require REGISTRY_AUTH_MODE=workflow-token"
[[ "${REGISTRY:-}" == "ghcr.io" ]] ||
  die "workflow-token registry authentication is restricted to ghcr.io"

registry_username="${CI_REGISTRY_USERNAME:-}"
[[ "${registry_username}" =~ ^([A-Za-z0-9][A-Za-z0-9_.-]{0,127}|[A-Za-z0-9][A-Za-z0-9_.-]{0,121}\[bot\])$ ]] ||
  die "CI_REGISTRY_USERNAME is invalid"

registry_token=""
if ! IFS= read -r registry_token; then
  die "a short-lived registry token must be provided on standard input"
fi
[[ -n "${registry_token}" && ${#registry_token} -le 4096 ]] ||
  die "the short-lived registry token is empty or unexpectedly large"

umask 077
mkdir -p "${DEPLOY_STATE_DIR}"
registry_config_dir="$(mktemp -d "${DEPLOY_STATE_DIR%/}/registry-auth.XXXXXX")"
cleanup() {
  registry_token=""
  unset registry_token
  if [[ -n "${registry_config_dir:-}" && -d "${registry_config_dir}" ]]; then
    rm -rf -- "${registry_config_dir}"
  fi
}
trap cleanup EXIT

export DOCKER_CONFIG="${registry_config_dir}"
printf '%s\n' "${registry_token}" |
  docker login "${REGISTRY}" --username "${registry_username}" --password-stdin >/dev/null
registry_token=""
unset registry_token
export CI_REGISTRY_LOGIN_READY=true

case "${operation}" in
  deploy)
    "${SCRIPT_DIR}/remote-preflight.sh"
    "${SCRIPT_DIR}/remote-prod-deploy.sh"
    ;;
  rollback)
    "${SCRIPT_DIR}/remote-prod-rollback.sh"
    ;;
esac

"${SCRIPT_DIR}/smoke-check.sh"
OBS_SMOKE_STRICT=true "${SCRIPT_DIR}/observability-smoke-check.sh"
log "remote CI ${operation} and post-deployment verification finished"
