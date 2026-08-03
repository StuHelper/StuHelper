#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

MODE="${1:-prod}"
if [[ "${MODE}" != "dev" && "${MODE}" != "prod" ]]; then
  die "usage: $0 [dev|prod]"
fi

require_cmd jq

CALLER_CASDOOR_CUTOVER_ENDPOINT="${CASDOOR_CUTOVER_ENDPOINT-}"
CALLER_OPENFGA_CUTOVER_API_URL="${OPENFGA_CUTOVER_API_URL-}"
CALLER_AUTHORIZATION_CUTOVER_DATABASE_URL="${AUTHORIZATION_CUTOVER_DATABASE_URL-}"
CALLER_CASDOOR_BOOTSTRAP_ENV_FILE="${CASDOOR_BOOTSTRAP_ENV_FILE-}"

load_env

if [[ -n "${CALLER_CASDOOR_CUTOVER_ENDPOINT}" ]]; then
  CASDOOR_CUTOVER_ENDPOINT="${CALLER_CASDOOR_CUTOVER_ENDPOINT}"
fi
if [[ -n "${CALLER_OPENFGA_CUTOVER_API_URL}" ]]; then
  OPENFGA_CUTOVER_API_URL="${CALLER_OPENFGA_CUTOVER_API_URL}"
fi
if [[ -n "${CALLER_AUTHORIZATION_CUTOVER_DATABASE_URL}" ]]; then
  AUTHORIZATION_CUTOVER_DATABASE_URL="${CALLER_AUTHORIZATION_CUTOVER_DATABASE_URL}"
fi
if [[ -n "${CALLER_CASDOOR_BOOTSTRAP_ENV_FILE}" ]]; then
  CASDOOR_BOOTSTRAP_ENV_FILE="${CALLER_CASDOOR_BOOTSTRAP_ENV_FILE}"
fi

resolve_env_path() {
  local raw="$1"
  case "${raw}" in
    /*) printf '%s\n' "${raw}" ;;
    *) printf '%s/%s\n' "$(dirname "${ENV_FILE}")" "${raw}" ;;
  esac
}

source_casdoor_bootstrap_env() {
  local file
  file="$(resolve_env_path "${CASDOOR_BOOTSTRAP_ENV_FILE:-.env.casdoor-bootstrap.local}")"
  [[ -f "${file}" ]] || die "missing Casdoor bootstrap env file: ${file}"
  source_casdoor_bootstrap_env_file "${file}"
}

require_nonempty() {
  local key="$1"
  [[ -n "${!key:-}" ]] || die "${key} is required for authorization ledger cutover"
}

casdoor_internal_endpoint() {
  local address="${CASDOOR_INTERNAL_ADDRESS:-}"
  [[ -n "${address}" ]] || return 1
  case "${address}" in
    http://*|https://*) printf '%s\n' "${address}" ;;
    *) printf 'http://%s\n' "${address}" ;;
  esac
}

source_casdoor_bootstrap_env

[[ -n "${AUTHORIZATION_CUTOVER_DATABASE_URL:-${DATABASE_URL:-}}" ]] || \
  die "AUTHORIZATION_CUTOVER_DATABASE_URL or DATABASE_URL is required for authorization ledger cutover"
[[ -n "${OPENFGA_CUTOVER_API_URL:-${OPENFGA_API_URL:-}}" ]] || \
  die "OPENFGA_CUTOVER_API_URL or OPENFGA_API_URL is required for authorization ledger cutover"

for key in \
  OPENFGA_STORE_ID \
  OPENFGA_MODEL_ID \
  CASDOOR_BOOTSTRAP_CLIENT_ID \
  CASDOOR_BOOTSTRAP_CLIENT_SECRET \
  CASDOOR_BOOTSTRAP_APPLICATION; do
  require_nonempty "${key}"
done
if [[ -z "${CASDOOR_BOOTSTRAP_ORGANIZATION:-${CASDOOR_ORGANIZATION:-}}" ]]; then
  die "CASDOOR_BOOTSTRAP_ORGANIZATION or CASDOOR_ORGANIZATION is required for authorization ledger cutover"
fi

run_cutover_with_go() {
  (
    cd "${REPO_ROOT}/server"
    DATABASE_URL="${AUTHORIZATION_CUTOVER_DATABASE_URL:-${DATABASE_URL}}" \
    OPENFGA_API_URL="${OPENFGA_CUTOVER_API_URL:-${OPENFGA_API_URL}}" \
    CASDOOR_CUTOVER_ENDPOINT="${CASDOOR_CUTOVER_ENDPOINT:-${CASDOOR_BOOTSTRAP_ENDPOINT:-${CASDOOR_ISSUER:-}}}" \
    GOFLAGS="-mod=readonly" \
      go run ./cmd/authorization-cutover
  )
}

run_cutover_with_docker() {
  local endpoint
  local network_name="${DOCKER_NETWORK_NAME:-${STACK_NAME:-${COMPOSE_PROJECT_NAME:-stuhelper}}-backend}"
  local go_image_ref="${AUTHORIZATION_CUTOVER_GO_IMAGE:-${GOLANG_IMAGE_REF:-cgr.dev/chainguard/go:latest@sha256:b116b5f2d3f5e7556b66252f9ee7ef9988b84c2139c89d824efcebd6cadbf436}}"
  local key
  local -a env_args=()

  endpoint="${CASDOOR_CUTOVER_ENDPOINT:-${CASDOOR_BOOTSTRAP_ENDPOINT:-}}"
  if [[ -z "${endpoint}" ]]; then
    endpoint="$(casdoor_internal_endpoint || true)"
  fi
  [[ -n "${endpoint}" ]] || die "CASDOOR_CUTOVER_ENDPOINT or CASDOOR_INTERNAL_ADDRESS is required for Docker cutover"

  export DATABASE_URL="${AUTHORIZATION_CUTOVER_DOCKER_DATABASE_URL:-${DATABASE_URL}}"
  export OPENFGA_API_URL="${OPENFGA_CUTOVER_DOCKER_API_URL:-http://openfga:8080}"
  export CASDOOR_CUTOVER_ENDPOINT="${endpoint}"

  for key in \
    DATABASE_URL DB_MAX_CONNS DB_MIN_CONNS DB_MAX_CONN_LIFETIME DB_MAX_CONN_IDLE_TIME DB_QUERY_TIMEOUT \
    DB_SSL_MODE DB_SSL_ROOT_CERT DB_SSL_CERT DB_SSL_KEY \
    OPENFGA_API_URL OPENFGA_STORE_ID OPENFGA_MODEL_ID \
    CASDOOR_CUTOVER_ENDPOINT CASDOOR_BOOTSTRAP_CLIENT_ID CASDOOR_BOOTSTRAP_CLIENT_SECRET \
    CASDOOR_BOOTSTRAP_CERTIFICATE CASDOOR_BOOTSTRAP_ORGANIZATION CASDOOR_ORGANIZATION \
    CASDOOR_BOOTSTRAP_APPLICATION; do
    env_args+=(-e "${key}")
  done

  docker run --rm \
    --read-only \
    --cap-drop ALL \
    --security-opt no-new-privileges \
    --tmpfs /tmp:rw,nosuid,nodev,noexec,size=64m \
    --tmpfs /go-cache:rw,nosuid,nodev,size=1024m \
    --network "${network_name}" \
    -v "${REPO_ROOT}:/workspace:ro" \
    -w /workspace/server \
    -e "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin" \
    -e GOCACHE=/go-cache/build \
    -e GOMODCACHE=/go-cache/mod \
    -e "GOFLAGS=-mod=readonly" \
    "${env_args[@]}" \
    --entrypoint /usr/bin/go \
    "${go_image_ref}" \
    run ./cmd/authorization-cutover
}

log "running one-time authorization authority cutover before application startup"
if command -v go >/dev/null 2>&1; then
  cutover_output="$(run_cutover_with_go)"
else
  require_cmd docker
  cutover_output="$(run_cutover_with_docker)"
fi

printf '%s\n' "${cutover_output}" | jq -e \
  'type == "object"
   and (.changed | type == "boolean")
   and (.sourceDigest | test("^[0-9a-f]{64}$"))
   and (.importedGrantCount | type == "number" and . >= 0)
   and (.skippedTupleCount | type == "number" and . >= 0)' >/dev/null

log "authorization authority cutover marker is complete"
printf '%s\n' "${cutover_output}"
