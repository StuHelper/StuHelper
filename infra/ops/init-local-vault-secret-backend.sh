#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
deploy_state_dir_was_explicit="false"
remote_config_file_was_explicit="false"
secret_file_root_was_explicit="false"
[[ -n "${DEPLOY_STATE_DIR+x}" ]] && deploy_state_dir_was_explicit="true"
[[ -n "${REMOTE_DEPLOY_CONFIG_FILE+x}" ]] && remote_config_file_was_explicit="true"
[[ -n "${SECRET_FILE_ROOT+x}" ]] && secret_file_root_was_explicit="true"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

require_cmd docker
require_cmd curl
require_cmd jq
require_cmd python3
require_cmd install

vault_image="${LOCAL_VAULT_IMAGE:-hashicorp/vault:1.17.6}"
vault_container="${LOCAL_VAULT_CONTAINER:-stuhelper-vault}"
vault_host="${LOCAL_VAULT_HOST:-127.0.0.1}"
vault_port="${LOCAL_VAULT_PORT:-18200}"
vault_logs_tmpfs_options="${LOCAL_VAULT_LOGS_TMPFS_OPTIONS:-rw,noexec,nosuid,nodev,size=16m}"
vault_addr="${VAULT_ADDR:-http://${vault_host}:${vault_port}}"
if [[ -n "${LOCAL_VAULT_STATE_DIR:-}" ]]; then
  vault_state_dir="${LOCAL_VAULT_STATE_DIR}"
elif [[ "${deploy_state_dir_was_explicit}" == "true" ]]; then
  vault_state_dir="${DEPLOY_STATE_DIR}/vault"
else
  vault_state_dir="${LOCAL_STATE_DIR}/vault"
fi
if [[ -n "${LOCAL_VAULT_CREDENTIALS_DIR:-}" ]]; then
  vault_credentials_dir="${LOCAL_VAULT_CREDENTIALS_DIR}"
elif [[ "${deploy_state_dir_was_explicit}" == "true" ]]; then
  vault_credentials_dir="${DEPLOY_STATE_DIR}/vault-credentials"
else
  vault_credentials_dir="${LOCAL_STATE_DIR}/vault-credentials"
fi
vault_config_file="${LOCAL_VAULT_CONFIG_FILE:-${vault_state_dir}/config.hcl}"
vault_data_dir="${LOCAL_VAULT_DATA_DIR:-${vault_state_dir}/file}"
vault_init_file="${LOCAL_VAULT_INIT_FILE:-${vault_credentials_dir}/init.json}"
vault_unseal_key_file="${LOCAL_VAULT_UNSEAL_KEY_FILE:-${vault_credentials_dir}/unseal-key}"

if [[ "${secret_file_root_was_explicit}" == "true" && -n "${SECRET_FILE_ROOT:-}" ]]; then
  secret_file_root="${SECRET_FILE_ROOT}"
elif [[ "${deploy_state_dir_was_explicit}" == "true" ]]; then
  secret_file_root="${DEPLOY_STATE_DIR}/secrets"
else
  secret_file_root="${LOCAL_STATE_DIR}/secrets"
fi
vault_token_file="${VAULT_TOKEN_FILE:-${secret_file_root}/vault/token}"
vault_kv_mount="${VAULT_KV_MOUNT:-secret}"
generated_secret_ref="${GENERATED_ENV_SECRET_REF:-${vault_kv_mount}/stuhelper/prod/generated-secrets-env}"
if [[ "${remote_config_file_was_explicit}" == "true" && -n "${REMOTE_DEPLOY_CONFIG_FILE:-}" ]]; then
  remote_config_file="${REMOTE_DEPLOY_CONFIG_FILE}"
elif [[ "${deploy_state_dir_was_explicit}" == "true" ]]; then
  remote_config_file="${DEPLOY_STATE_DIR}/remote.env"
else
  remote_config_file="${LOCAL_STATE_DIR}/deploy/remote.env"
fi
remote_config_compat_file=""
if [[ "${remote_config_file_was_explicit}" != "true" && "${deploy_state_dir_was_explicit}" != "true" ]]; then
  remote_config_compat_file="${REPO_ROOT}/.deploy/remote.env"
fi

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

if [[ "${LOCAL_VAULT_PRINT_PATHS_ONLY:-false}" == "true" ]]; then
  printf 'LOCAL_VAULT_STATE_DIR=%s\n' "${vault_state_dir}"
  printf 'LOCAL_VAULT_CONFIG_FILE=%s\n' "${vault_config_file}"
  printf 'LOCAL_VAULT_DATA_DIR=%s\n' "${vault_data_dir}"
  printf 'LOCAL_VAULT_LOGS_TMPFS_OPTIONS=%s\n' "${vault_logs_tmpfs_options}"
  printf 'LOCAL_VAULT_CREDENTIALS_DIR=%s\n' "${vault_credentials_dir}"
  printf 'LOCAL_VAULT_INIT_FILE=%s\n' "${vault_init_file}"
  printf 'LOCAL_VAULT_UNSEAL_KEY_FILE=%s\n' "${vault_unseal_key_file}"
  printf 'SECRET_FILE_ROOT=%s\n' "${secret_file_root}"
  printf 'VAULT_TOKEN_FILE=%s\n' "${vault_token_file}"
  printf 'REMOTE_DEPLOY_CONFIG_FILE=%s\n' "${remote_config_file}"
  printf 'REMOTE_DEPLOY_COMPAT_FILE=%s\n' "${remote_config_compat_file}"
  exit 0
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

canonical_path() {
  python3 - "$1" <<'PY'
from pathlib import Path
import sys

print(Path(sys.argv[1]).expanduser().resolve(strict=False))
PY
}

container_mount_source() {
  local destination="$1"
  docker inspect "${vault_container}" |
    jq -r --arg destination "${destination}" \
      '.[0].Mounts[] | select(.Type == "bind" and .Destination == $destination) | .Source' |
    head -n 1
}

container_layout_issues() {
  local actual_config_source actual_data_source actual_host actual_image actual_logs_tmpfs actual_port actual_restart
  local expected_config_source expected_data_source

  expected_config_source="$(canonical_path "${vault_config_file}")"
  expected_data_source="$(canonical_path "${vault_data_dir}")"
  actual_config_source="$(container_mount_source /vault/config/config.hcl)"
  actual_data_source="$(container_mount_source /vault/file)"
  actual_image="$(docker inspect -f '{{.Config.Image}}' "${vault_container}")"
  actual_restart="$(docker inspect -f '{{.HostConfig.RestartPolicy.Name}}' "${vault_container}")"
  actual_logs_tmpfs="$(docker inspect "${vault_container}" | jq -r '.[0].HostConfig.Tmpfs["/vault/logs"] // ""')"
  actual_host="$(docker inspect "${vault_container}" | jq -r --arg port "${vault_port}/tcp" '.[0].HostConfig.PortBindings[$port][0].HostIp // ""')"
  actual_port="$(docker inspect "${vault_container}" | jq -r --arg port "${vault_port}/tcp" '.[0].HostConfig.PortBindings[$port][0].HostPort // ""')"

  [[ "$(canonical_path "${actual_config_source}")" == "${expected_config_source}" ]] ||
    printf 'config bind source is %s, expected %s\n' "${actual_config_source:-<missing>}" "${expected_config_source}"
  [[ "$(canonical_path "${actual_data_source}")" == "${expected_data_source}" ]] ||
    printf 'data bind source is %s, expected %s\n' "${actual_data_source:-<missing>}" "${expected_data_source}"
  [[ "${actual_image}" == "${vault_image}" ]] ||
    printf 'image is %s, expected %s\n' "${actual_image:-<missing>}" "${vault_image}"
  [[ "${actual_host}" == "${vault_host}" ]] ||
    printf 'published host is %s, expected %s\n' "${actual_host:-<missing>}" "${vault_host}"
  [[ "${actual_port}" == "${vault_port}" ]] ||
    printf 'published port is %s, expected %s\n' "${actual_port:-<missing>}" "${vault_port}"
  [[ "${actual_restart}" == "unless-stopped" ]] ||
    printf 'restart policy is %s, expected unless-stopped\n' "${actual_restart:-<missing>}"
  [[ "${actual_logs_tmpfs}" == "${vault_logs_tmpfs_options}" ]] ||
    printf 'logs tmpfs is %s, expected %s\n' "${actual_logs_tmpfs:-<missing>}" "${vault_logs_tmpfs_options}"
}

validate_initialized_recreate_target() {
  [[ -f "${vault_config_file}" ]] ||
    die "refusing to recreate initialized Vault without staged config: ${vault_config_file}"
  [[ -d "${vault_data_dir}" ]] ||
    die "refusing to recreate initialized Vault without staged data directory: ${vault_data_dir}"
  [[ -n "$(find "${vault_data_dir}" -type f -print -quit 2>/dev/null)" ]] ||
    die "refusing to recreate initialized Vault from an empty data directory: ${vault_data_dir}"
  [[ -f "${vault_unseal_key_file}" ]] ||
    die "refusing to recreate initialized Vault without unseal key file: ${vault_unseal_key_file}"
  [[ -f "${vault_token_file}" || -f "${vault_init_file}" ]] ||
    die "refusing to recreate initialized Vault without token or init material"
}

reconcile_existing_container() {
  local initialized issues
  issues="$(container_layout_issues)"
  if [[ -z "${issues}" ]]; then
    return 0
  fi

  if [[ "${LOCAL_VAULT_RECREATE_CONTAINER:-false}" != "true" ]]; then
    die "existing ${vault_container} layout does not match the stable local state paths: ${issues}. Stage the existing data and credentials first, then rerun with LOCAL_VAULT_RECREATE_CONTAINER=true"
  fi

  container_running || die "refusing offline recreation of mismatched ${vault_container}; start and inspect the existing container first"
  initialized="$(vault_health_json | jq -r '.initialized')"
  if [[ "${initialized}" == "true" ]]; then
    validate_initialized_recreate_target
  fi

  log "stopping mismatched local Vault container after stable state validation"
  docker stop --time 30 "${vault_container}" >/dev/null
  docker rm "${vault_container}" >/dev/null
}

verify_container_layout() {
  local issues
  issues="$(container_layout_issues)"
  [[ -z "${issues}" ]] || die "local Vault container layout verification failed: ${issues}"
}

start_vault_container() {
  if container_exists; then
    reconcile_existing_container
  fi
  if container_running; then
    verify_container_layout
    return 0
  fi
  if container_exists; then
    docker start "${vault_container}" >/dev/null
    verify_container_layout
    return 0
  fi

  write_vault_config

  docker run -d \
    --name "${vault_container}" \
    --restart unless-stopped \
    --entrypoint vault \
    -p "${vault_host}:${vault_port}:${vault_port}" \
    --tmpfs "/vault/logs:${vault_logs_tmpfs_options}" \
    -v "${vault_config_file}:/vault/config/config.hcl:ro" \
    -v "${vault_data_dir}:/vault/file" \
    "${vault_image}" \
    server -config=/vault/config/config.hcl >/dev/null
  verify_container_layout
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

  if [[ "${LOCAL_VAULT_REQUIRE_EXISTING_DATA:-false}" == "true" ]]; then
    die "refusing to initialize an uninitialized Vault while LOCAL_VAULT_REQUIRE_EXISTING_DATA=true"
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
  if ! python3 - "${vault_addr}" "${vault_unseal_key_file}" <<'PY'
import json
from pathlib import Path
import sys
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen

vault_addr = sys.argv[1].rstrip("/")
key_file = Path(sys.argv[2])
key = key_file.read_text().strip()
if not key:
    raise SystemExit("configured unseal key file is empty")

request = Request(
    f"{vault_addr}/v1/sys/unseal",
    data=json.dumps({"key": key}).encode(),
    headers={"Content-Type": "application/json"},
    method="PUT",
)
try:
    with urlopen(request, timeout=5) as response:
        result = json.load(response)
except (HTTPError, URLError, TimeoutError) as exc:
    raise SystemExit(f"Vault unseal request failed: {exc}") from exc

if result.get("sealed") is not False:
    raise SystemExit("Vault accepted the unseal request but remains sealed")
PY
  then
    die "failed to unseal local Vault with the configured key file"
  fi
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

verify_existing_vault_data_if_required() {
  local token
  [[ "${LOCAL_VAULT_REQUIRE_EXISTING_DATA:-false}" == "true" ]] || return 0

  token="$(vault_token)"
  docker exec \
    -e "VAULT_ADDR=${vault_addr}" \
    -e "VAULT_TOKEN=${token}" \
    "${vault_container}" \
    vault token lookup -format=json >/dev/null
  docker exec \
    -e "VAULT_ADDR=${vault_addr}" \
    -e "VAULT_TOKEN=${token}" \
    "${vault_container}" \
    vault secrets list -format=json |
    jq -e --arg mount "${vault_kv_mount}/" 'has($mount)' >/dev/null
  secret_backend_read_to_stdout "${generated_secret_ref}" >/dev/null
  log "verified existing local Vault token, KV mount, and generated secret reference"
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

write_remote_deploy_compat_link() {
  local existing_target
  [[ -n "${remote_config_compat_file}" ]] || return 0
  [[ "$(canonical_path "${remote_config_file}")" != "$(canonical_path "${remote_config_compat_file}")" ]] || return 0

  install -d -m 0700 "$(dirname "${remote_config_compat_file}")"
  if [[ -L "${remote_config_compat_file}" ]]; then
    existing_target="$(canonical_path "$(dirname "${remote_config_compat_file}")/$(readlink "${remote_config_compat_file}")")"
    [[ "${existing_target}" == "$(canonical_path "${remote_config_file}")" ]] ||
      die "refusing to replace unexpected remote config symlink: ${remote_config_compat_file}"
    return 0
  fi
  [[ ! -e "${remote_config_compat_file}" ]] ||
    die "refusing to replace existing remote deploy config: ${remote_config_compat_file}"
  ln -s "${remote_config_file}" "${remote_config_compat_file}"
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

verify_existing_vault_data_if_required
enable_kv_v2_if_needed
seed_generated_secret_env_if_needed
write_remote_deploy_config
write_remote_deploy_compat_link

log "local Vault secret backend is ready"
echo "  Vault: ${vault_addr}"
echo "  Remote config: ${remote_config_file}"
echo "  Token file: ${vault_token_file}"
echo "  Generated secret ref: ${generated_secret_ref}"
warn "keep ${vault_init_file}, ${vault_unseal_key_file}, and ${vault_token_file} secret and backed up"
