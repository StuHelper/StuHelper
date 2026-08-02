#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

require_cmd docker

cleanup_project_resources() {
  local project_name="$1"
  local containers=()
  local networks=()
  local volumes=()

  [[ "${project_name}" =~ ^[A-Za-z0-9][A-Za-z0-9_.-]*$ ]] ||
    die "invalid prod-parity Compose project name: ${project_name}"
  [[ "${project_name}" == *prod-parity* ]] ||
    die "refusing to clean non-parity Compose project: ${project_name}"

  mapfile -t containers < <(
    docker ps -aq --filter "label=com.docker.compose.project=${project_name}"
  )
  if ((${#containers[@]} > 0)); then
    die "refusing to remove prod-parity resources while project containers remain: ${project_name}"
  fi

  if [[ "${REMOVE_VOLUMES:-false}" == "true" ]]; then
    mapfile -t volumes < <(
      docker volume ls -q --filter "label=com.docker.compose.project=${project_name}"
    )
    if ((${#volumes[@]} > 0)); then
      docker volume rm "${volumes[@]}"
    fi
  fi

  mapfile -t networks < <(
    docker network ls -q --filter "label=com.docker.compose.project=${project_name}"
  )
  if ((${#networks[@]} > 0)); then
    docker network rm "${networks[@]}"
  fi
}

cleanup_remaining_parity_resources() {
  local app_project="${COMPOSE_PROJECT_NAME:-stuhelper-prod-parity}"
  local datastore_project="${PROD_PARITY_DATASTORE_PROJECT:-stuhelper-prod-parity-datastore}"

  cleanup_project_resources "${app_project}"
  if [[ "${datastore_project}" != "${app_project}" ]]; then
    cleanup_project_resources "${datastore_project}"
  fi
}

PARITY_DIR="${PROD_PARITY_DIR:-${REPO_ROOT}/.run/prod-parity}"
parity_default_path() {
  local current="$1"
  local common_default="$2"
  local parity_default="$3"
  if repo_default_path_matches "${current}" "${common_default}"; then
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

if [[ -f "${ENV_FILE}" ]]; then
  datastore_args=(--env-file "${ENV_FILE}" -f "${REPO_ROOT}/docker-compose.prod-parity-postgres.yml" down --remove-orphans)
  if [[ "${REMOVE_VOLUMES:-false}" == "true" ]]; then
    datastore_args+=(-v)
  fi
  (
    cd "${REPO_ROOT}" && \
    docker compose "${datastore_args[@]}"
  ) || true
fi

cleanup_remaining_parity_resources

if [[ "${REMOVE_LOCAL_INGRESS:-true}" == "true" ]]; then
  "${SCRIPT_DIR}/remove-local-prod-parity-ingress.sh"
fi

log "local production parity stack stopped"
