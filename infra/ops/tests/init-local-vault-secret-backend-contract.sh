#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
TARGET_SCRIPT="${REPO_ROOT}/infra/ops/init-local-vault-secret-backend.sh"
MAKEFILE="${REPO_ROOT}/Makefile"

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

[[ -f "${TARGET_SCRIPT}" ]] || fail "missing ${TARGET_SCRIPT}"

assert_contains "${TARGET_SCRIPT}" 'LOCAL_VAULT_IMAGE:-hashicorp/vault:1\.17\.6'
assert_contains "${TARGET_SCRIPT}" 'LOCAL_VAULT_CONTAINER:-stuhelper-vault'
assert_contains "${TARGET_SCRIPT}" 'LOCAL_VAULT_PORT:-18200'
assert_contains "${TARGET_SCRIPT}" 'VAULT_TOKEN_FILE:-\$\{secret_file_root\}/vault/token'
assert_contains "${TARGET_SCRIPT}" 'GENERATED_ENV_SECRET_REF:-\$\{vault_kv_mount\}/stuhelper/prod/generated-secrets-env'
assert_contains "${TARGET_SCRIPT}" 'docker run -d'
assert_contains "${TARGET_SCRIPT}" '--entrypoint vault'
assert_contains "${TARGET_SCRIPT}" 'server -config=/vault/config/config\.hcl'
assert_contains "${TARGET_SCRIPT}" 'vault operator init -key-shares=1 -key-threshold=1 -format=json'
assert_contains "${TARGET_SCRIPT}" 'vault operator unseal'
assert_contains "${TARGET_SCRIPT}" 'vault secrets enable -path="\$\{vault_kv_mount\}" kv-v2'
assert_contains "${TARGET_SCRIPT}" 'secret_backend_write_from_file "\$\{generated_secret_ref\}" "\$\{tmp_file\}"'
assert_contains "${TARGET_SCRIPT}" 'init-remote-deploy-config\.sh'
assert_contains "${TARGET_SCRIPT}" 'upsert_env_file "\$\{remote_config_file\}" "SHARED_ENV_SECRET_REF" "\$\{LOCAL_VAULT_SHARED_ENV_SECRET_REF:-\}"'
assert_contains "${TARGET_SCRIPT}" 'upsert_env_file "\$\{remote_config_file\}" "SECRETS_ENV_SECRET_REF" "\$\{LOCAL_VAULT_SECRETS_ENV_SECRET_REF:-\}"'
assert_contains "${TARGET_SCRIPT}" 'warn "keep \$\{vault_init_file\}, \$\{vault_unseal_key_file\}, and \$\{vault_token_file\} secret and backed up"'
assert_not_contains "${TARGET_SCRIPT}" 'echo .*\$\{VAULT_TOKEN'
assert_not_contains "${TARGET_SCRIPT}" 'echo .*\$\{token\}'

assert_contains "${MAKEFILE}" '^prod-vault-init:'
assert_contains "${MAKEFILE}" 'init-local-vault-secret-backend\.sh'

echo "[init-local-vault-secret-backend-contract] all assertions passed"
