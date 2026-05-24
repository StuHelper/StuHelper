#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

require_cmd docker
require_cmd node

PARITY_DIR="${PROD_PARITY_DIR:-${REPO_ROOT}/.run/prod-parity}"
parity_default_path() {
  local current="$1"
  local common_default="$2"
  local parity_default="$3"
  if [[ -z "${current}" || "${current}" == "${common_default}" ]]; then
    printf '%s\n' "${parity_default}"
    return
  fi
  printf '%s\n' "${current}"
}

export ENV_TEMPLATE_FILE="${REPO_ROOT}/.env.prod.example"
export ENV_FILE="$(parity_default_path "${ENV_FILE:-}" "${REPO_ROOT}/.env" "${PARITY_DIR}/.env.prod.shared")"
export SECRETS_ENV_FILE="$(parity_default_path "${SECRETS_ENV_FILE:-}" "" "${PARITY_DIR}/.env.prod.secrets.local")"
export GENERATED_ENV_FILE="$(parity_default_path "${GENERATED_ENV_FILE:-}" "${REPO_ROOT}/.env.generated" "${PARITY_DIR}/.env.prod.generated")"
export GENERATED_SECRET_ENV_FILE="$(parity_default_path "${GENERATED_SECRET_ENV_FILE:-}" "${REPO_ROOT}/.env.generated.secrets" "${PARITY_DIR}/.env.prod.generated.secrets")"
export DEPLOY_STATE_DIR="$(parity_default_path "${DEPLOY_STATE_DIR:-}" "${REPO_ROOT}/.deploy" "${PARITY_DIR}/deploy-state")"

load_env

clear_rate_limit_keys() {
  local redis_container="${REDIS_CONTAINER_NAME:-${STACK_NAME:-stuhelper-prod-parity}-redis}"
  local redis_username="${REDIS_USERNAME:-stuhelper_app}"
  local key_output
  local keys=()
  local redis_cli=(
    docker exec
    -e "REDISCLI_AUTH=${REDIS_PASSWORD:?REDIS_PASSWORD is required}"
    "${redis_container}"
    redis-cli
    --user "${redis_username}"
  )

  if [[ "${REDIS_TLS_ENABLED:-true}" == "true" ]]; then
    redis_cli+=(--tls --cacert /tls/ca.crt)
  fi

  if ! docker inspect "${redis_container}" >/dev/null 2>&1; then
    die "Redis container not found for prod-parity browser smoke: ${redis_container}"
  fi

  key_output="$("${redis_cli[@]}" --scan --pattern 'rl:*')"
  while IFS= read -r key; do
    [[ -n "${key}" ]] && keys+=("${key}")
  done <<<"${key_output}"

  if (( ${#keys[@]} == 0 )); then
    return
  fi

  "${redis_cli[@]}" DEL "${keys[@]}" >/dev/null
  log "cleared ${#keys[@]} prod-parity rate-limit keys before browser smoke"
}

clear_rate_limit_keys

export API_BASE_URL="${API_BASE_URL:-http://127.0.0.1:${BACKEND_EXTERNAL_PORT:-28080}}"
export WEB_BASE_URL="${WEB_BASE_URL:-http://127.0.0.1:${WEB_EXTERNAL_PORT:-28000}}"
export ADMIN_BASE_URL="${ADMIN_BASE_URL:-http://127.0.0.1:${ADMIN_EXTERNAL_PORT:-28001}}"
export PROD_PARITY_BROWSER_SMOKE_EVIDENCE_FILE="${PROD_PARITY_BROWSER_SMOKE_EVIDENCE_FILE:-${PARITY_DIR}/browser-smoke-evidence.json}"
export PROD_PARITY_BROWSER_SMOKE_SCREENSHOT_DIR="${PROD_PARITY_BROWSER_SMOKE_SCREENSHOT_DIR:-${PARITY_DIR}/browser-smoke-screenshots}"

node "${SCRIPT_DIR}/prod-parity-browser-smoke.mjs"
