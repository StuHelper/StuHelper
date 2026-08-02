#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

config_file="${REMOTE_DEPLOY_CONFIG_FILE:-${DEPLOY_STATE_DIR}/remote.env}"
common_default_env_file="${REPO_ROOT}/.env"
common_default_generated_env_file="${REPO_ROOT}/.env.generated"
common_default_generated_secret_env_file="${REPO_ROOT}/.env.generated.secrets"
common_default_secret_file_root="${REPO_ROOT}/infra/generated/secrets"

default_env_file="${ENV_FILE:-}"
if [[ -z "${default_env_file}" || "${default_env_file}" == "${common_default_env_file}" ]]; then
  default_env_file="${REPO_ROOT}/.env.prod.shared"
fi

default_secrets_env_file="${SECRETS_ENV_FILE:-}"
if [[ -z "${default_secrets_env_file}" ]]; then
  default_secrets_env_file="${REPO_ROOT}/.env.prod.secrets"
fi

default_generated_env_file="${GENERATED_ENV_FILE:-}"
if [[ -z "${default_generated_env_file}" || "${default_generated_env_file}" == "${common_default_generated_env_file}" ]]; then
  default_generated_env_file="${REPO_ROOT}/.env.prod.generated"
fi

default_generated_secret_env_file="${GENERATED_SECRET_ENV_FILE:-}"
if [[ -z "${default_generated_secret_env_file}" || "${default_generated_secret_env_file}" == "${common_default_generated_secret_env_file}" ]]; then
  default_generated_secret_env_file="${REPO_ROOT}/.env.prod.generated.secrets"
fi

default_secret_file_root="${SECRET_FILE_ROOT:-}"
if [[ -z "${default_secret_file_root}" || "${default_secret_file_root}" == "${common_default_secret_file_root}" ]]; then
  default_secret_file_root="${REPO_ROOT}/.secrets"
fi

default_secret_backend="${SECRET_BACKEND:-}"
if [[ -z "${default_secret_backend}" || "${default_secret_backend}" == "none" ]]; then
  default_secret_backend="vault-kv-v2"
fi

mkdir -p "$(dirname "${config_file}")"

DEFAULT_REGISTRY="${REGISTRY:-REPLACE_WITH_REGISTRY_HOST}" \
DEFAULT_REGISTRY_AUTH_MODE="${REGISTRY_AUTH_MODE:-workflow-token}" \
DEFAULT_REGISTRY_USERNAME_SECRET_REF="${REGISTRY_USERNAME_SECRET_REF:-secret/stuhelper/prod/registry-username}" \
DEFAULT_REGISTRY_PASSWORD_SECRET_REF="${REGISTRY_PASSWORD_SECRET_REF:-secret/stuhelper/prod/registry-password}" \
DEFAULT_ENV_FILE="${default_env_file}" \
DEFAULT_SECRETS_ENV_FILE="${default_secrets_env_file}" \
DEFAULT_GENERATED_ENV_FILE="${default_generated_env_file}" \
DEFAULT_GENERATED_SECRET_ENV_FILE="${default_generated_secret_env_file}" \
DEFAULT_SECRET_BACKEND="${default_secret_backend}" \
DEFAULT_SHARED_ENV_SECRET_REF="${SHARED_ENV_SECRET_REF:-secret/stuhelper/prod/shared-env}" \
DEFAULT_SECRETS_ENV_SECRET_REF="${SECRETS_ENV_SECRET_REF:-secret/stuhelper/prod/secrets-env}" \
DEFAULT_GENERATED_ENV_SECRET_REF="${GENERATED_ENV_SECRET_REF:-secret/stuhelper/prod/generated-secrets-env}" \
DEFAULT_SECRET_FILE_ROOT="${default_secret_file_root}" \
DEFAULT_VAULT_ADDR="${VAULT_ADDR:-REPLACE_WITH_VAULT_ADDR}" \
DEFAULT_VAULT_NAMESPACE="${VAULT_NAMESPACE:-}" \
DEFAULT_VAULT_TOKEN_FILE="${VAULT_TOKEN_FILE:-${default_secret_file_root}/vault/token}" \
DEFAULT_VAULT_KV_MOUNT="${VAULT_KV_MOUNT:-secret}" \
DEFAULT_VAULT_RUNTIME_TOKEN_POLICY="${VAULT_RUNTIME_TOKEN_POLICY:-stuhelper-production-deploy}" \
DEFAULT_VAULT_RUNTIME_TOKEN_PERIOD_SECONDS="${VAULT_RUNTIME_TOKEN_PERIOD_SECONDS:-259200}" \
DEFAULT_VAULT_RUNTIME_TOKEN_MIN_TTL_SECONDS="${VAULT_RUNTIME_TOKEN_MIN_TTL_SECONDS:-43200}" \
python3 - "${config_file}" <<'PY'
from pathlib import Path
import os
import sys

path = Path(sys.argv[1])
existing = {}
if path.exists():
    for raw in path.read_text().splitlines():
        line = raw.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        key, value = line.split("=", 1)
        existing[key.strip()] = value.strip()

defaults = {
    "REGISTRY": os.environ["DEFAULT_REGISTRY"],
    "REGISTRY_AUTH_MODE": os.environ["DEFAULT_REGISTRY_AUTH_MODE"],
    "REGISTRY_USERNAME_SECRET_REF": os.environ["DEFAULT_REGISTRY_USERNAME_SECRET_REF"],
    "REGISTRY_PASSWORD_SECRET_REF": os.environ["DEFAULT_REGISTRY_PASSWORD_SECRET_REF"],
    "ENV_FILE": os.environ["DEFAULT_ENV_FILE"],
    "SECRETS_ENV_FILE": os.environ["DEFAULT_SECRETS_ENV_FILE"],
    "GENERATED_ENV_FILE": os.environ["DEFAULT_GENERATED_ENV_FILE"],
    "GENERATED_SECRET_ENV_FILE": os.environ["DEFAULT_GENERATED_SECRET_ENV_FILE"],
    "SECRET_BACKEND": os.environ["DEFAULT_SECRET_BACKEND"],
    "SHARED_ENV_SECRET_REF": os.environ["DEFAULT_SHARED_ENV_SECRET_REF"],
    "SECRETS_ENV_SECRET_REF": os.environ["DEFAULT_SECRETS_ENV_SECRET_REF"],
    "GENERATED_ENV_SECRET_REF": os.environ["DEFAULT_GENERATED_ENV_SECRET_REF"],
    "SECRET_FILE_ROOT": os.environ["DEFAULT_SECRET_FILE_ROOT"],
    "VAULT_ADDR": os.environ["DEFAULT_VAULT_ADDR"],
    "VAULT_NAMESPACE": os.environ["DEFAULT_VAULT_NAMESPACE"],
    "VAULT_TOKEN_FILE": os.environ["DEFAULT_VAULT_TOKEN_FILE"],
    "VAULT_KV_MOUNT": os.environ["DEFAULT_VAULT_KV_MOUNT"],
    "VAULT_RUNTIME_TOKEN_POLICY": os.environ["DEFAULT_VAULT_RUNTIME_TOKEN_POLICY"],
    "VAULT_RUNTIME_TOKEN_PERIOD_SECONDS": os.environ["DEFAULT_VAULT_RUNTIME_TOKEN_PERIOD_SECONDS"],
    "VAULT_RUNTIME_TOKEN_MIN_TTL_SECONDS": os.environ["DEFAULT_VAULT_RUNTIME_TOKEN_MIN_TTL_SECONDS"],
}

keys = list(defaults.keys())
result = {}
for key in keys:
    result[key] = os.environ.get(key, "") or existing.get(key, "") or defaults[key]

path.write_text(
    "# Remote-owned deploy control plane for StuHelper.\n"
    "# Manage this file on the target host; CI/Ansible no longer rewrite it per release.\n"
    + "\n".join(f"{key}={result[key]}" for key in keys)
    + "\n"
)
path.chmod(0o600)
PY

source_remote_deploy_config_env_file "${config_file}"
if [[ "${SECRET_BACKEND:-}" == "file" && "${REGISTRY_AUTH_MODE:-persistent-secret}" == "persistent-secret" ]]; then
  registry_username_path="$(secret_file_path "${REGISTRY_USERNAME_SECRET_REF}")"
  registry_password_path="$(secret_file_path "${REGISTRY_PASSWORD_SECRET_REF}")"
  install -d -m 0700 "${SECRET_FILE_ROOT}" "$(dirname "${registry_username_path}")" "$(dirname "${registry_password_path}")"
  touch "${registry_username_path}" "${registry_password_path}"
  chmod 0600 "${registry_username_path}" "${registry_password_path}"
fi

log "remote deploy config is ready: ${config_file}"
