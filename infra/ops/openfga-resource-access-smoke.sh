#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

usage() {
  cat <<'USAGE'
Usage: infra/ops/openfga-resource-access-smoke.sh

Runs a real OpenFGA resource-access smoke for Open Platform v1.1. The smoke
writes app -> resource_item read/write tuples, verifies check/list-objects,
revokes them, and verifies both checks become false.

Required env:
  OPENFGA_API_URL
  OPENFGA_STORE_ID
  OPENFGA_MODEL_ID

Optional env:
  ENV_FILE                         defaults to .env
  SECRETS_ENV_FILE                 optional extra env file
  GENERATED_ENV_FILE               defaults to .env.generated
  OPENFGA_RESOURCE_SMOKE_APP_ID
  OPENFGA_RESOURCE_SMOKE_RESOURCE_ID
  OPENFGA_RESOURCE_SMOKE_TIMEOUT   Go duration, defaults to 20s
  OPENFGA_RESOURCE_SMOKE_MODE      host or container; defaults to container in production, otherwise host when Go is available
  OPENFGA_RESOURCE_SMOKE_GO_IMAGE  container-mode Go image; defaults to GOLANG_IMAGE_REF or golang:1.26-bookworm
  OPENFGA_RESOURCE_SMOKE_EVIDENCE_FILE
USAGE
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
  exit 0
fi

require_cmd jq

load_env

[[ -n "${OPENFGA_API_URL:-}" ]] || die "OPENFGA_API_URL is required"
[[ -n "${OPENFGA_STORE_ID:-}" ]] || die "OPENFGA_STORE_ID is required"
[[ -n "${OPENFGA_MODEL_ID:-}" ]] || die "OPENFGA_MODEL_ID is required"

run_smoke_with_go() {
  (
    cd "${REPO_ROOT}/server"
    go run ./cmd/openfga-resource-smoke
  )
}

run_smoke_with_docker() {
  require_cmd docker
  local stack_name_value="${STACK_NAME:-${COMPOSE_PROJECT_NAME:-stuhelper}}"
  local docker_network_name="${DOCKER_NETWORK_NAME:-${stack_name_value}-backend}"
  local go_image_ref="${OPENFGA_RESOURCE_SMOKE_GO_IMAGE:-${GOLANG_IMAGE_REF:-golang:1.26-bookworm}}"
  docker run --rm \
    --network "${docker_network_name}" \
    -v "${REPO_ROOT}:/workspace" \
    -w /workspace/server \
    -e "PATH=/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin" \
    -e OPENFGA_API_URL="${OPENFGA_API_URL}" \
    -e OPENFGA_STORE_ID="${OPENFGA_STORE_ID}" \
    -e OPENFGA_MODEL_ID="${OPENFGA_MODEL_ID}" \
    -e OPENFGA_RESOURCE_SMOKE_APP_ID="${OPENFGA_RESOURCE_SMOKE_APP_ID:-}" \
    -e OPENFGA_RESOURCE_SMOKE_RESOURCE_ID="${OPENFGA_RESOURCE_SMOKE_RESOURCE_ID:-}" \
    -e OPENFGA_RESOURCE_SMOKE_TIMEOUT="${OPENFGA_RESOURCE_SMOKE_TIMEOUT:-}" \
    "${go_image_ref}" \
    go run ./cmd/openfga-resource-smoke
}

smoke_mode() {
  if [[ -n "${OPENFGA_RESOURCE_SMOKE_MODE:-}" ]]; then
    printf '%s\n' "${OPENFGA_RESOURCE_SMOKE_MODE}"
    return
  fi
  if [[ "${APP_ENV:-}" == "production" ]]; then
    printf '%s\n' "container"
    return
  fi
  if command -v go >/dev/null 2>&1; then
    printf '%s\n' "host"
    return
  fi
  printf '%s\n' "container"
}

log "running Open Platform OpenFGA resource access smoke against ${OPENFGA_API_URL}" >&2
mode="$(smoke_mode)"
case "${mode}" in
  host)
    command -v go >/dev/null 2>&1 || die "Go is required when OPENFGA_RESOURCE_SMOKE_MODE=host"
    evidence_json="$(run_smoke_with_go)"
    ;;
  container)
    evidence_json="$(run_smoke_with_docker)"
    ;;
  *)
    die "OPENFGA_RESOURCE_SMOKE_MODE must be host or container"
    ;;
esac

if [[ -n "${OPENFGA_RESOURCE_SMOKE_EVIDENCE_FILE:-}" ]]; then
  mkdir -p "$(dirname "${OPENFGA_RESOURCE_SMOKE_EVIDENCE_FILE}")"
  printf '%s\n' "${evidence_json}" | jq . >"${OPENFGA_RESOURCE_SMOKE_EVIDENCE_FILE}"
fi

printf '%s\n' "${evidence_json}" | jq .
