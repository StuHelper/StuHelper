#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

MODE="${1:-dev}"
if [[ "${MODE}" != "dev" && "${MODE}" != "prod" ]]; then
  die "usage: $0 [dev|prod]"
fi
require_cmd docker
require_cmd curl
require_cmd jq
require_cmd python3

CALLER_CASDOOR_BOOTSTRAP_ENABLED="${CASDOOR_BOOTSTRAP_ENABLED-}"
CALLER_CASDOOR_BOOTSTRAP_ENV_FILE="${CASDOOR_BOOTSTRAP_ENV_FILE-}"
CALLER_CASDOOR_BOOTSTRAP_ENDPOINT="${CASDOOR_BOOTSTRAP_ENDPOINT-}"
CALLER_OPENFGA_BOOTSTRAP_API_URL="${OPENFGA_BOOTSTRAP_API_URL-}"
CALLER_OPENFGA_BOOTSTRAP_DATABASE_URL="${OPENFGA_BOOTSTRAP_DATABASE_URL-}"

load_env
if [[ -n "${CALLER_CASDOOR_BOOTSTRAP_ENABLED}" ]]; then
  CASDOOR_BOOTSTRAP_ENABLED="${CALLER_CASDOOR_BOOTSTRAP_ENABLED}"
fi
if [[ -n "${CALLER_CASDOOR_BOOTSTRAP_ENV_FILE}" ]]; then
  CASDOOR_BOOTSTRAP_ENV_FILE="${CALLER_CASDOOR_BOOTSTRAP_ENV_FILE}"
fi
if [[ -n "${CALLER_CASDOOR_BOOTSTRAP_ENDPOINT}" ]]; then
  CASDOOR_BOOTSTRAP_ENDPOINT="${CALLER_CASDOOR_BOOTSTRAP_ENDPOINT}"
fi
if [[ -n "${CALLER_OPENFGA_BOOTSTRAP_API_URL}" ]]; then
  OPENFGA_BOOTSTRAP_API_URL="${CALLER_OPENFGA_BOOTSTRAP_API_URL}"
fi
if [[ -n "${CALLER_OPENFGA_BOOTSTRAP_DATABASE_URL}" ]]; then
  OPENFGA_BOOTSTRAP_DATABASE_URL="${CALLER_OPENFGA_BOOTSTRAP_DATABASE_URL}"
fi
STACK_NAME_VALUE="${STACK_NAME:-${COMPOSE_PROJECT_NAME:-stuhelper}}"
DOCKER_NETWORK_NAME="${DOCKER_NETWORK_NAME:-${STACK_NAME_VALUE}-backend}"
CASDOOR_BOOTSTRAP_ENV_FILE="${CASDOOR_BOOTSTRAP_ENV_FILE:-${REPO_ROOT}/.env.casdoor-bootstrap.local}"
CASDOOR_BOOTSTRAP_ENABLED="${CASDOOR_BOOTSTRAP_ENABLED:-false}"

wait_for_casdoor() {
  local url="${CASDOOR_ISSUER:-http://localhost:8085}/.well-known/openid-configuration"
  local retries=90
  local i

  for ((i = 1; i <= retries; i++)); do
    if curl -fsS --max-time 5 "${url}" >/dev/null 2>&1; then
      log "Casdoor OIDC metadata is ready: ${url}"
      return 0
    fi
    sleep 2
  done

  die "Casdoor OIDC metadata did not become ready in time: ${url}"
}

source_casdoor_bootstrap_env() {
  local file
  case "${CASDOOR_BOOTSTRAP_ENV_FILE}" in
    /*) file="${CASDOOR_BOOTSTRAP_ENV_FILE}" ;;
    *) file="$(dirname "${ENV_FILE}")/${CASDOOR_BOOTSTRAP_ENV_FILE}" ;;
  esac
  if [[ -f "${file}" ]]; then
    # shellcheck disable=SC1090
    set -a
    source "${file}"
    set +a
  fi
}

casdoor_bootstrap_required() {
  [[ "${MODE}" == "prod" || "${CASDOOR_BOOTSTRAP_ENABLED}" == "true" ]]
}

require_casdoor_bootstrap_config() {
  local key
  for key in \
    CASDOOR_BOOTSTRAP_CLIENT_ID \
    CASDOOR_BOOTSTRAP_CLIENT_SECRET \
    CASDOOR_BOOTSTRAP_APPLICATION \
    CASDOOR_CLIENT_ID \
    CASDOOR_CLIENT_SECRET \
    CASDOOR_REDIRECT_URI \
    CASDOOR_ADMIN_CLIENT_ID \
    CASDOOR_ADMIN_CLIENT_SECRET \
    CASDOOR_ADMIN_REDIRECT_URI \
    CASDOOR_UNIAPP_CLIENT_ID \
    CASDOOR_UNIAPP_CLIENT_SECRET \
    CASDOOR_UNIAPP_REDIRECT_URI \
    CASDOOR_APP_PROVISIONING_CLIENT_ID \
    CASDOOR_APP_PROVISIONING_CLIENT_SECRET \
    CASDOOR_APP_PROVISIONING_APPLICATION \
    CASDOOR_USER_PROFILE_CLIENT_ID \
    CASDOOR_USER_PROFILE_CLIENT_SECRET \
    CASDOOR_USER_PROFILE_APPLICATION \
    CASDOOR_INTROSPECTION_CLIENT_ID \
    CASDOOR_INTROSPECTION_CLIENT_SECRET \
    CASDOOR_INTROSPECTION_APPLICATION \
    CASDOOR_USER_LOOKUP_CLIENT_ID \
    CASDOOR_USER_LOOKUP_CLIENT_SECRET \
    CASDOOR_USER_LOOKUP_APPLICATION \
    CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_ID \
    CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_SECRET \
    CASDOOR_TOKEN_PROBE_SMOKE_APPLICATION \
    CASDOOR_TOKEN_PROBE_SMOKE_REDIRECT_URI \
    CASDOOR_ORGANIZATION; do
    [[ -n "${!key:-}" ]] || die "${key} must be configured before Casdoor bootstrap"
  done
}

reject_placeholder_if_set() {
  local key="$1"
  local value="${2:-}"
  shift 2 || true
  [[ -n "${value}" ]] || return 0
  local placeholder
  for placeholder in "$@"; do
    if [[ "${value}" == "${placeholder}" ]]; then
      die "${key} is using placeholder/default value (${placeholder}); set a real value first"
    fi
  done
}

casdoor_internal_endpoint() {
  local address="${CASDOOR_INTERNAL_ADDRESS:-}"
  [[ -n "${address}" ]] || return 1
  case "${address}" in
    http://*|https://*) printf '%s\n' "${address}" ;;
    *) printf 'http://%s\n' "${address}" ;;
  esac
}

casdoor_bootstrap_endpoint() {
  local prefer_internal="${1:-false}"
  local endpoint
  if [[ -n "${CASDOOR_BOOTSTRAP_ENDPOINT:-}" ]]; then
    printf '%s\n' "${CASDOOR_BOOTSTRAP_ENDPOINT}"
    return
  fi
  if [[ "${prefer_internal}" == "true" ]] && endpoint="$(casdoor_internal_endpoint)"; then
    printf '%s\n' "${endpoint}"
    return
  fi
  printf '%s\n' "${CASDOOR_ISSUER:-http://localhost:8085}"
}

run_casdoor_bootstrap_with_go() {
  local mode="${1:-full}"
  local -a env_args=()
  if [[ "${mode}" == "applications-only" ]]; then
    env_args+=(CASDOOR_BOOTSTRAP_MODE=applications-only)
  fi
  (
    cd "${REPO_ROOT}/server" && \
    env "${env_args[@]}" \
    CASDOOR_BOOTSTRAP_ENDPOINT="$(casdoor_bootstrap_endpoint false)" \
    go run ./cmd/casdoor-bootstrap
  )
}

run_casdoor_bootstrap_with_docker() {
  local mode="${1:-full}"
  local endpoint
  local -a env_args
  local key
  endpoint="$(casdoor_bootstrap_endpoint true)"
  env_args=(-e "CASDOOR_BOOTSTRAP_ENDPOINT=${endpoint}")
  if [[ "${mode}" == "applications-only" ]]; then
    env_args+=(-e "CASDOOR_BOOTSTRAP_MODE=applications-only")
  fi
  for key in "${CASDOOR_BOOTSTRAP_ENV_KEYS[@]}"; do
    env_args+=(-e "${key}=${!key:-}")
  done
  docker run --rm \
    --network "${DOCKER_NETWORK_NAME}" \
    -v "${REPO_ROOT}:/workspace" \
    -w /workspace/server \
    -e "PATH=/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin" \
    "${env_args[@]}" \
    golang:1.26.5-bookworm \
    go run ./cmd/casdoor-bootstrap
}

CASDOOR_BOOTSTRAP_ENV_KEYS=(
  CASDOOR_BOOTSTRAP_CLIENT_ID
  CASDOOR_BOOTSTRAP_CLIENT_SECRET
  CASDOOR_BOOTSTRAP_APPLICATION
  CASDOOR_BOOTSTRAP_CERTIFICATE
  CASDOOR_BOOTSTRAP_ORGANIZATION
  CASDOOR_ORGANIZATION
  CASDOOR_ORGANIZATION_DISPLAY_NAME
  WEB_PUBLIC_URL
  CASDOOR_LOGO
  CASDOOR_CLIENT_ID
  CASDOOR_CLIENT_SECRET
  CASDOOR_REDIRECT_URI
  CASDOOR_ADDITIONAL_REDIRECT_URIS
  CASDOOR_ADMIN_LOGO
  CASDOOR_ADMIN_CLIENT_ID
  CASDOOR_ADMIN_CLIENT_SECRET
  CASDOOR_ADMIN_REDIRECT_URI
  CASDOOR_ADMIN_ADDITIONAL_REDIRECT_URIS
  CASDOOR_UNIAPP_LOGO
  CASDOOR_UNIAPP_CLIENT_ID
  CASDOOR_UNIAPP_CLIENT_SECRET
  CASDOOR_UNIAPP_REDIRECT_URI
  CASDOOR_UNIAPP_ADDITIONAL_REDIRECT_URIS
  CASDOOR_APP_PROVISIONING_LOGO
  CASDOOR_APP_PROVISIONING_CLIENT_ID
  CASDOOR_APP_PROVISIONING_CLIENT_SECRET
  CASDOOR_APP_PROVISIONING_APPLICATION
  CASDOOR_USER_PROFILE_LOGO
  CASDOOR_USER_PROFILE_CLIENT_ID
  CASDOOR_USER_PROFILE_CLIENT_SECRET
  CASDOOR_USER_PROFILE_APPLICATION
  CASDOOR_INTROSPECTION_LOGO
  CASDOOR_INTROSPECTION_CLIENT_ID
  CASDOOR_INTROSPECTION_CLIENT_SECRET
  CASDOOR_INTROSPECTION_APPLICATION
  CASDOOR_USER_LOOKUP_LOGO
  CASDOOR_USER_LOOKUP_CLIENT_ID
  CASDOOR_USER_LOOKUP_CLIENT_SECRET
  CASDOOR_USER_LOOKUP_APPLICATION
  CASDOOR_TOKEN_PROBE_SMOKE_LOGO
  CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_ID
  CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_SECRET
  CASDOOR_TOKEN_PROBE_SMOKE_APPLICATION
  CASDOOR_TOKEN_PROBE_SMOKE_REDIRECT_URI
  CASDOOR_TOKEN_PROBE_SMOKE_ADDITIONAL_REDIRECT_URIS
  CASDOOR_SMS_PROVIDER_ENABLED
  CASDOOR_SMS_PROVIDER_NAME
  CASDOOR_SMS_PROVIDER_DISPLAY_NAME
  CASDOOR_SMS_PROVIDER_CATEGORY
  CASDOOR_SMS_PROVIDER_TYPE
  CASDOOR_SMS_PROVIDER_SUB_TYPE
  CASDOOR_SMS_PROVIDER_METHOD
  CASDOOR_SMS_PROVIDER_PROVIDER_URL
  CASDOOR_SMS_PROVIDER_ENDPOINT
  CASDOOR_SMS_PROVIDER_HOST
  CASDOOR_SMS_PROVIDER_PORT
  CASDOOR_SMS_PROVIDER_DISABLE_SSL
  CASDOOR_SMS_PROVIDER_TITLE
  CASDOOR_SMS_PROVIDER_CONTENT
  CASDOOR_SMS_PROVIDER_METADATA
  SMS_INTERNAL_KEY
  CASDOOR_EMAIL_PROVIDER_ENABLED
  CASDOOR_EMAIL_PROVIDER_NAME
  CASDOOR_EMAIL_PROVIDER_DISPLAY_NAME
  CASDOOR_EMAIL_PROVIDER_CATEGORY
  CASDOOR_EMAIL_PROVIDER_TYPE
  CASDOOR_EMAIL_PROVIDER_SUB_TYPE
  CASDOOR_EMAIL_PROVIDER_METHOD
  CASDOOR_EMAIL_PROVIDER_PROVIDER_URL
  CASDOOR_EMAIL_PROVIDER_ENDPOINT
  CASDOOR_EMAIL_PROVIDER_HOST
  CASDOOR_EMAIL_PROVIDER_PORT
  CASDOOR_EMAIL_PROVIDER_DISABLE_SSL
  CASDOOR_EMAIL_PROVIDER_TITLE
  CASDOOR_EMAIL_PROVIDER_CONTENT
  CASDOOR_EMAIL_PROVIDER_METADATA
)

source_casdoor_bootstrap_env
if casdoor_bootstrap_required; then
  wait_for_casdoor
  [[ -n "${CASDOOR_CLIENT_ID:-}" ]] || die "CASDOOR_CLIENT_ID must be configured before platform bootstrap"
  [[ -n "${CASDOOR_CLIENT_SECRET:-}" ]] || die "CASDOOR_CLIENT_SECRET must be configured before platform bootstrap"
  log "using configured Casdoor OIDC application ${CASDOOR_CLIENT_ID}"
  require_casdoor_bootstrap_config
  log "bootstrapping Casdoor identity organization, applications, and providers"
  if command -v go >/dev/null 2>&1; then
    if ! run_casdoor_bootstrap_with_go; then
      warn "full Casdoor bootstrap failed; retrying applications-only bootstrap with app provisioning credentials"
      run_casdoor_bootstrap_with_go applications-only
    fi
  else
    if ! run_casdoor_bootstrap_with_docker; then
      warn "full Casdoor bootstrap failed; retrying applications-only bootstrap with app provisioning credentials"
      run_casdoor_bootstrap_with_docker applications-only
    fi
  fi
else
  warn "Casdoor bootstrap skipped because CASDOOR_BOOTSTRAP_ENABLED is not true"
fi

reject_placeholder_if_set OPENFGA_STORE_ID "${OPENFGA_STORE_ID:-}" "REPLACE_WITH_OPENFGA_STORE_ID"
reject_placeholder_if_set OPENFGA_MODEL_ID "${OPENFGA_MODEL_ID:-}" "REPLACE_WITH_OPENFGA_MODEL_ID"
log "bootstrapping OpenFGA store and model"
if command -v go >/dev/null 2>&1; then
  FGA_OUTPUT="$(
    cd "${REPO_ROOT}/server" && \
    OPENFGA_API_URL="${OPENFGA_BOOTSTRAP_API_URL:-${OPENFGA_API_URL:-http://localhost:8081}}" \
    DATABASE_URL="${OPENFGA_BOOTSTRAP_DATABASE_URL:-${DATABASE_URL:-}}" \
    OPENFGA_STORE_ID="${OPENFGA_STORE_ID:-}" \
    FGA_MODEL_PATH="${FGA_MODEL_PATH:-${REPO_ROOT}/infra/openfga/model.fga}" \
    go run ./cmd/fga-setup
  )"
else
  FGA_OUTPUT="$(
    docker run --rm \
      --network "${DOCKER_NETWORK_NAME}" \
      -v "${REPO_ROOT}:/workspace" \
      -w /workspace/server \
      -e "PATH=/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin" \
      -e OPENFGA_API_URL="http://openfga:8080" \
      -e OPENFGA_STORE_ID="${OPENFGA_STORE_ID:-}" \
      -e FGA_MODEL_PATH="${FGA_MODEL_PATH:-/workspace/infra/openfga/model.fga}" \
      golang:1.26.5-bookworm \
      go run ./cmd/fga-setup
  )"
fi

OPENFGA_STORE_ID_VALUE="$(printf '%s\n' "${FGA_OUTPUT}" | awk -F= '/^OPENFGA_STORE_ID=/{print $2}' | tail -n1)"
OPENFGA_MODEL_ID_VALUE="$(printf '%s\n' "${FGA_OUTPUT}" | awk -F= '/^OPENFGA_MODEL_ID=/{print $2}' | tail -n1)"

[[ -n "${OPENFGA_STORE_ID_VALUE}" ]] || die "failed to extract OPENFGA_STORE_ID from setup output"
[[ -n "${OPENFGA_MODEL_ID_VALUE}" ]] || die "failed to extract OPENFGA_MODEL_ID from setup output"
log "verified configured Casdoor OIDC application ${CASDOOR_CLIENT_ID}"
log "bootstrapped OpenFGA store ${OPENFGA_STORE_ID_VALUE} with model ${OPENFGA_MODEL_ID_VALUE}"

runtime_env_tmp="$(mktemp)"
secret_env_tmp="$(mktemp)"
trap 'rm -f "${runtime_env_tmp}" "${secret_env_tmp}"' EXIT

cat >"${runtime_env_tmp}" <<EOF
# Generated by infra/ops/bootstrap-platform.sh (${MODE})
OPENFGA_STORE_ID=${OPENFGA_STORE_ID_VALUE}
OPENFGA_MODEL_ID=${OPENFGA_MODEL_ID_VALUE}
EOF

cat >"${secret_env_tmp}" <<EOF
# Generated secrets by infra/ops/bootstrap-platform.sh (${MODE})
EOF

install -m 600 "${runtime_env_tmp}" "${GENERATED_ENV_FILE}"
log "wrote derived runtime configuration to ${GENERATED_ENV_FILE}"

if [[ "${MODE}" == "prod" ]]; then
  [[ -n "${GENERATED_ENV_SECRET_REF:-}" ]] || die "GENERATED_ENV_SECRET_REF is required in production"
  [[ -n "${SECRET_BACKEND:-}" && "${SECRET_BACKEND}" != "none" && "${SECRET_BACKEND}" != "file" ]] || \
    die "production generated secrets must use a non-file secret backend"
  secret_backend_write_from_file "${GENERATED_ENV_SECRET_REF}" "${secret_env_tmp}"
  : > "${GENERATED_SECRET_ENV_FILE}"
  log "persisted generated secrets to remote secret backend ref ${GENERATED_ENV_SECRET_REF}"
else
  cat "${secret_env_tmp}" >> "${GENERATED_ENV_FILE}"
  : > "${GENERATED_SECRET_ENV_FILE}"
  log "in development mode appended generated secrets to ${GENERATED_ENV_FILE}"
fi
