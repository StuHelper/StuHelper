#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

require_cmd docker
require_cmd curl
require_cmd jq
require_cmd python3

vault_image="${LOCAL_VAULT_IMAGE:-hashicorp/vault:1.17.6}"
vault_container="${LOCAL_VAULT_CONTAINER:-stuhelper-vault}"
vault_host="${LOCAL_VAULT_HOST:-127.0.0.1}"
vault_port="${LOCAL_VAULT_PORT:-18200}"
vault_addr="${VAULT_ADDR:-http://${vault_host}:${vault_port}}"
vault_state_dir="${LOCAL_VAULT_STATE_DIR:-${DEPLOY_STATE_DIR}/vault}"
vault_config_file="${LOCAL_VAULT_CONFIG_FILE:-${vault_state_dir}/config.hcl}"
vault_data_dir="${LOCAL_VAULT_DATA_DIR:-${vault_state_dir}/file}"
vault_init_file="${LOCAL_VAULT_INIT_FILE:-${vault_state_dir}/init.json}"
vault_unseal_key_file="${LOCAL_VAULT_UNSEAL_KEY_FILE:-${vault_state_dir}/unseal-key}"

default_secret_file_root="${SECRET_FILE_ROOT:-}"
if [[ -z "${default_secret_file_root}" || "${default_secret_file_root}" == "${REPO_ROOT}/infra/generated/secrets" ]]; then
  default_secret_file_root="${REPO_ROOT}/.secrets"
fi
secret_file_root="${default_secret_file_root}"
vault_token_file="${VAULT_TOKEN_FILE:-${secret_file_root}/vault/token}"
vault_kv_mount="${VAULT_KV_MOUNT:-secret}"
generated_secret_ref="${GENERATED_ENV_SECRET_REF:-${vault_kv_mount}/stuhelper/prod/generated-secrets-env}"
remote_config_file="${REMOTE_DEPLOY_CONFIG_FILE:-${DEPLOY_STATE_DIR}/remote.env}"

default_env_file="${ENV_FILE:-}"
if [[ -z "${default_env_file}" || "${default_env_file}" == "${REPO_ROOT}/.env" ]]; then
  default_env_file="${REPO_ROOT}/.env.prod.shared"
fi
default_secrets_env_file="${SECRETS_ENV_FILE:-}"
if [[ -z "${default_secrets_env_file}" ]]; then
  default_secrets_env_file="${REPO_ROOT}/.env.prod.secrets.local"
fi
default_generated_env_file="${GENERATED_ENV_FILE:-}"
if [[ -z "${default_generated_env_file}" || "${default_generated_env_file}" == "${REPO_ROOT}/.env.generated" ]]; then
  default_generated_env_file="${REPO_ROOT}/.env.prod.generated"
fi
default_generated_secret_env_file="${GENERATED_SECRET_ENV_FILE:-}"
if [[ -z "${default_generated_secret_env_file}" || "${default_generated_secret_env_file}" == "${REPO_ROOT}/.env.generated.secrets" ]]; then
  default_generated_secret_env_file="${REPO_ROOT}/.env.prod.generated.secrets"
fi

write_vault_config() {
  mkdir -p "$(dirname "${vault_config_file}")" "${vault_data_dir}"
  chmod 700 "$(dirname "${vault_config_file}")" "${vault_data_dir}" 2>/dev/null || true
  if [[ -f "${vault_config_file}" ]]; then
    return 0
  fi

  cat >"${vault_config_file}" <<EOF
ui = true
disable_mlock = true

storage "file" {
  path = "/vault/file"
}

listener "tcp" {
  address = "0.0.0.0:${vault_port}"
  tls_disable = true
}

api_addr = "${vault_addr}"
EOF
  chmod 600 "${vault_config_file}" 2>/dev/null || true
}

container_exists() {
  docker ps -a --format '{{.Names}}' | grep -Fxq "${vault_container}"
}

container_running() {
  docker ps --format '{{.Names}}' | grep -Fxq "${vault_container}"
}

start_vault_container() {
  write_vault_config
  if container_running; then
    return 0
  fi
  if container_exists; then
    docker start "${vault_container}" >/dev/null
    return 0
  fi

  docker run -d \
    --name "${vault_container}" \
    --restart unless-stopped \
    --entrypoint vault \
    -p "${vault_host}:${vault_port}:${vault_port}" \
    -v "${vault_config_file}:/vault/config/config.hcl:ro" \
    -v "${vault_data_dir}:/vault/file" \
    "${vault_image}" \
    server -config=/vault/config/config.hcl >/dev/null
}

wait_for_vault_api() {
  local health_url="${vault_addr%/}/v1/sys/health?standbyok=true&sealedcode=200&uninitcode=200"
  local i
  for ((i = 1; i <= 60; i++)); do
    if curl -fsS --max-time 2 "${health_url}" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  die "local Vault API did not become reachable: ${vault_addr}"
}

vault_health_json() {
  curl -fsS --max-time 5 "${vault_addr%/}/v1/sys/health?standbyok=true&sealedcode=200&uninitcode=200"
}

write_unseal_and_token_files_from_init_json() {
  [[ -f "${vault_init_file}" ]] || return 0
  install -d -m 0700 "$(dirname "${vault_unseal_key_file}")" "$(dirname "${vault_token_file}")"
  jq -r '.unseal_keys_b64[0]' "${vault_init_file}" >"${vault_unseal_key_file}"
  jq -r '.root_token' "${vault_init_file}" >"${vault_token_file}"
  chmod 600 "${vault_init_file}" "${vault_unseal_key_file}" "${vault_token_file}"
}

initialize_vault_if_needed() {
  local initialized
  initialized="$(vault_health_json | jq -r '.initialized')"
  if [[ "${initialized}" == "true" ]]; then
    if [[ ! -f "${vault_token_file}" && ! -f "${vault_init_file}" ]]; then
      die "Vault is already initialized but no token file exists at ${vault_token_file}; restore the token or re-create the local Vault data directory intentionally"
    fi
    write_unseal_and_token_files_from_init_json
    return 0
  fi

  install -d -m 0700 "$(dirname "${vault_init_file}")"
  docker exec \
    -e "VAULT_ADDR=${vault_addr}" \
    "${vault_container}" \
    vault operator init -key-shares=1 -key-threshold=1 -format=json >"${vault_init_file}"
  chmod 600 "${vault_init_file}"
  write_unseal_and_token_files_from_init_json
}

unseal_vault_if_needed() {
  local sealed
  sealed="$(vault_health_json | jq -r '.sealed')"
  if [[ "${sealed}" != "true" ]]; then
    return 0
  fi
  [[ -f "${vault_unseal_key_file}" ]] || die "Vault is sealed and unseal key file is missing: ${vault_unseal_key_file}"
  docker exec \
    -e "VAULT_ADDR=${vault_addr}" \
    "${vault_container}" \
    vault operator unseal "$(tr -d '\r\n' <"${vault_unseal_key_file}")" >/dev/null
}

enable_kv_v2_if_needed() {
  local token
  token="$(vault_token)"
  if docker exec \
    -e "VAULT_ADDR=${vault_addr}" \
    -e "VAULT_TOKEN=${token}" \
    "${vault_container}" \
    vault secrets list -format=json | jq -e --arg mount "${vault_kv_mount}/" 'has($mount)' >/dev/null; then
    return 0
  fi

  docker exec \
    -e "VAULT_ADDR=${vault_addr}" \
    -e "VAULT_TOKEN=${token}" \
    "${vault_container}" \
    vault secrets enable -path="${vault_kv_mount}" kv-v2 >/dev/null
}

seed_generated_secret_env_if_needed() {
  local tmp_file
  export SECRET_BACKEND="vault-kv-v2"
  export SECRET_FILE_ROOT="${secret_file_root}"
  export VAULT_ADDR="${vault_addr}"
  export VAULT_TOKEN_FILE="${vault_token_file}"
  export VAULT_KV_MOUNT="${vault_kv_mount}"
  export GENERATED_ENV_SECRET_REF="${generated_secret_ref}"

  if secret_backend_read_to_stdout "${generated_secret_ref}" >/dev/null 2>&1; then
    return 0
  fi

  tmp_file="$(mktemp)"
  : >"${tmp_file}"
  secret_backend_write_from_file "${generated_secret_ref}" "${tmp_file}"
  rm -f "${tmp_file}"
}

write_remote_deploy_config() {
  REMOTE_DEPLOY_CONFIG_FILE="${remote_config_file}" \
  ENV_FILE="${default_env_file}" \
  SECRETS_ENV_FILE="${default_secrets_env_file}" \
  GENERATED_ENV_FILE="${default_generated_env_file}" \
  GENERATED_SECRET_ENV_FILE="${default_generated_secret_env_file}" \
  SECRET_BACKEND="vault-kv-v2" \
  SECRET_FILE_ROOT="${secret_file_root}" \
  GENERATED_ENV_SECRET_REF="${generated_secret_ref}" \
  VAULT_ADDR="${vault_addr}" \
  VAULT_TOKEN_FILE="${vault_token_file}" \
  VAULT_KV_MOUNT="${vault_kv_mount}" \
    "${SCRIPT_DIR}/init-remote-deploy-config.sh" >/dev/null

  # Local bootstrap keeps hand-edited env files on disk and only stores generated
  # secret env in Vault. Operators can opt in to fully remote env materialization
  # by setting these refs explicitly later.
  upsert_env_file "${remote_config_file}" "SHARED_ENV_SECRET_REF" "${LOCAL_VAULT_SHARED_ENV_SECRET_REF:-}"
  upsert_env_file "${remote_config_file}" "SECRETS_ENV_SECRET_REF" "${LOCAL_VAULT_SECRETS_ENV_SECRET_REF:-}"
  upsert_env_file "${remote_config_file}" "GENERATED_ENV_SECRET_REF" "${generated_secret_ref}"
  chmod 600 "${remote_config_file}"
}

start_vault_container
wait_for_vault_api
initialize_vault_if_needed
unseal_vault_if_needed

export SECRET_BACKEND="vault-kv-v2"
export SECRET_FILE_ROOT="${secret_file_root}"
export VAULT_ADDR="${vault_addr}"
export VAULT_TOKEN_FILE="${vault_token_file}"
export VAULT_KV_MOUNT="${vault_kv_mount}"

enable_kv_v2_if_needed
seed_generated_secret_env_if_needed
write_remote_deploy_config

log "local Vault secret backend is ready"
echo "  Vault: ${vault_addr}"
echo "  Remote config: ${remote_config_file}"
echo "  Token file: ${vault_token_file}"
echo "  Generated secret ref: ${generated_secret_ref}"
warn "keep ${vault_init_file}, ${vault_unseal_key_file}, and ${vault_token_file} secret and backed up"
