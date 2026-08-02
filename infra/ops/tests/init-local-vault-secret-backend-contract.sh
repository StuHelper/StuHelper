#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
TARGET_SCRIPT="${REPO_ROOT}/infra/ops/init-local-vault-secret-backend.sh"
MAKEFILE="${REPO_ROOT}/Makefile"
tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT

fail() {
  echo "[init-local-vault-secret-backend-contract][error] $*" >&2
  exit 1
}

assert_contains() {
  local file="$1"
  local pattern="$2"
  if ! grep -Eq -- "${pattern}" "${file}"; then
    fail "expected ${file} to contain pattern: ${pattern}"
  fi
}

assert_not_contains() {
  local file="$1"
  local pattern="$2"
  if grep -Eq -- "${pattern}" "${file}"; then
    fail "expected ${file} to not contain pattern: ${pattern}"
  fi
}

assert_line() {
  local file="$1"
  local expected="$2"
  if ! grep -Fxq -- "${expected}" "${file}"; then
    fail "expected ${file} to contain exact line: ${expected}"
  fi
}

[[ -f "${TARGET_SCRIPT}" ]] || fail "missing ${TARGET_SCRIPT}"

assert_contains "${TARGET_SCRIPT}" 'LOCAL_VAULT_IMAGE:-hashicorp/vault:1\.17\.6'
assert_contains "${TARGET_SCRIPT}" 'LOCAL_VAULT_CONTAINER:-stuhelper-vault'
assert_contains "${TARGET_SCRIPT}" 'LOCAL_VAULT_PORT:-18200'
assert_contains "${TARGET_SCRIPT}" 'LOCAL_VAULT_LOGS_TMPFS_OPTIONS:-rw,noexec,nosuid,nodev,size=16m'
assert_contains "${TARGET_SCRIPT}" 'vault_state_dir="\$\{LOCAL_STATE_DIR\}/vault"'
assert_contains "${TARGET_SCRIPT}" 'vault_credentials_dir="\$\{LOCAL_STATE_DIR\}/vault-credentials"'
assert_contains "${TARGET_SCRIPT}" 'secret_file_root="\$\{LOCAL_STATE_DIR\}/secrets"'
assert_contains "${TARGET_SCRIPT}" 'remote_config_file="\$\{LOCAL_STATE_DIR\}/deploy/remote.env"'
assert_contains "${TARGET_SCRIPT}" 'VAULT_TOKEN_FILE:-\$\{secret_file_root\}/vault/token'
assert_contains "${TARGET_SCRIPT}" 'GENERATED_ENV_SECRET_REF:-\$\{vault_kv_mount\}/stuhelper/prod/generated-secrets-env'
assert_contains "${TARGET_SCRIPT}" 'LOCAL_VAULT_RECREATE_CONTAINER:-false'
assert_contains "${TARGET_SCRIPT}" 'LOCAL_VAULT_REQUIRE_EXISTING_DATA:-false'
assert_contains "${TARGET_SCRIPT}" 'refusing to recreate initialized Vault from an empty data directory'
assert_contains "${TARGET_SCRIPT}" 'local Vault container layout verification failed'
assert_contains "${TARGET_SCRIPT}" 'ln -s "\$\{remote_config_file\}" "\$\{remote_config_compat_file\}"'
assert_contains "${TARGET_SCRIPT}" 'docker run -d'
assert_contains "${TARGET_SCRIPT}" '--entrypoint vault'
assert_contains "${TARGET_SCRIPT}" '--tmpfs "/vault/logs:\$\{vault_logs_tmpfs_options\}"'
assert_contains "${TARGET_SCRIPT}" 'server -config=/vault/config/config\.hcl'
assert_contains "${TARGET_SCRIPT}" 'vault operator init -key-shares=1 -key-threshold=1 -format=json'
assert_contains "${TARGET_SCRIPT}" '/v1/sys/unseal'
assert_not_contains "${TARGET_SCRIPT}" 'vault operator unseal'
assert_contains "${TARGET_SCRIPT}" 'vault secrets enable -path="\$\{vault_kv_mount\}" kv-v2'
assert_contains "${TARGET_SCRIPT}" 'secret_backend_write_from_file "\$\{generated_secret_ref\}" "\$\{tmp_file\}"'
assert_contains "${TARGET_SCRIPT}" 'secret_backend_read_to_stdout "\$\{generated_secret_ref\}" >/dev/null'
assert_contains "${TARGET_SCRIPT}" 'init-remote-deploy-config\.sh'
assert_contains "${TARGET_SCRIPT}" 'upsert_env_file "\$\{remote_config_file\}" "SHARED_ENV_SECRET_REF" "\$\{LOCAL_VAULT_SHARED_ENV_SECRET_REF:-\}"'
assert_contains "${TARGET_SCRIPT}" 'upsert_env_file "\$\{remote_config_file\}" "SECRETS_ENV_SECRET_REF" "\$\{LOCAL_VAULT_SECRETS_ENV_SECRET_REF:-\}"'
assert_contains "${TARGET_SCRIPT}" 'warn "keep \$\{vault_init_file\}, \$\{vault_unseal_key_file\}, and \$\{vault_token_file\} secret and backed up"'
assert_not_contains "${TARGET_SCRIPT}" 'echo .*\$\{VAULT_TOKEN'
assert_not_contains "${TARGET_SCRIPT}" 'echo .*\$\{token\}'

assert_contains "${MAKEFILE}" '^prod-vault-init:'
assert_contains "${MAKEFILE}" 'init-local-vault-secret-backend\.sh'

mkdir -p "${tmpdir}/home" "${tmpdir}/state"
HOME="${tmpdir}/home" \
XDG_STATE_HOME="${tmpdir}/state" \
LOCAL_VAULT_PRINT_PATHS_ONLY=true \
  "${TARGET_SCRIPT}" >"${tmpdir}/default-paths"

local_root="${tmpdir}/state/stuhelper"
assert_line "${tmpdir}/default-paths" "LOCAL_VAULT_STATE_DIR=${local_root}/vault"
assert_line "${tmpdir}/default-paths" "LOCAL_VAULT_CONFIG_FILE=${local_root}/vault/config.hcl"
assert_line "${tmpdir}/default-paths" "LOCAL_VAULT_DATA_DIR=${local_root}/vault/file"
assert_line "${tmpdir}/default-paths" "LOCAL_VAULT_LOGS_TMPFS_OPTIONS=rw,noexec,nosuid,nodev,size=16m"
assert_line "${tmpdir}/default-paths" "LOCAL_VAULT_CREDENTIALS_DIR=${local_root}/vault-credentials"
assert_line "${tmpdir}/default-paths" "LOCAL_VAULT_INIT_FILE=${local_root}/vault-credentials/init.json"
assert_line "${tmpdir}/default-paths" "LOCAL_VAULT_UNSEAL_KEY_FILE=${local_root}/vault-credentials/unseal-key"
assert_line "${tmpdir}/default-paths" "SECRET_FILE_ROOT=${local_root}/secrets"
assert_line "${tmpdir}/default-paths" "VAULT_TOKEN_FILE=${local_root}/secrets/vault/token"
assert_line "${tmpdir}/default-paths" "REMOTE_DEPLOY_CONFIG_FILE=${local_root}/deploy/remote.env"
assert_line "${tmpdir}/default-paths" "REMOTE_DEPLOY_COMPAT_FILE=${REPO_ROOT}/.deploy/remote.env"

DEPLOY_STATE_DIR="${tmpdir}/explicit-deploy" \
LOCAL_VAULT_PRINT_PATHS_ONLY=true \
  "${TARGET_SCRIPT}" >"${tmpdir}/override-paths"
assert_line "${tmpdir}/override-paths" "LOCAL_VAULT_STATE_DIR=${tmpdir}/explicit-deploy/vault"
assert_line "${tmpdir}/override-paths" "LOCAL_VAULT_CREDENTIALS_DIR=${tmpdir}/explicit-deploy/vault-credentials"
assert_line "${tmpdir}/override-paths" "SECRET_FILE_ROOT=${tmpdir}/explicit-deploy/secrets"
assert_line "${tmpdir}/override-paths" "REMOTE_DEPLOY_CONFIG_FILE=${tmpdir}/explicit-deploy/remote.env"
assert_line "${tmpdir}/override-paths" "REMOTE_DEPLOY_COMPAT_FILE="

echo "[init-local-vault-secret-backend-contract] all assertions passed"
