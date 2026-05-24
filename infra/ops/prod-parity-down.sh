#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

require_cmd docker

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

if [[ -f "${ENV_FILE}" ]]; then
  load_env
  args=(--profile prod down --remove-orphans)
  if [[ "${REMOVE_VOLUMES:-false}" == "true" ]]; then
    args+=(-v)
  fi
  compose "${args[@]}"
else
  warn "local production parity app env file not found, skipping app compose down: ${ENV_FILE}"
fi

datastore_args=(--env-file "${ENV_FILE}" -f "${REPO_ROOT}/docker-compose.prod-parity-postgres.yml" down --remove-orphans)
if [[ "${REMOVE_VOLUMES:-false}" == "true" ]]; then
  datastore_args+=(-v)
fi
(
  cd "${REPO_ROOT}" && \
  docker compose "${datastore_args[@]}"
) || true

log "local production parity stack stopped"
