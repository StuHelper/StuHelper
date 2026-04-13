#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

config_file="${REMOTE_DEPLOY_CONFIG_FILE:-${DEPLOY_STATE_DIR}/remote.env}"
default_env_file="${ENV_FILE:-${REPO_ROOT}/.env.prod.shared}"
default_secrets_env_file="${SECRETS_ENV_FILE:-${REPO_ROOT}/.env.prod.secrets}"
default_generated_env_file="${GENERATED_ENV_FILE:-${REPO_ROOT}/.env.prod.generated}"
default_generated_secret_env_file="${GENERATED_SECRET_ENV_FILE:-${REPO_ROOT}/.env.prod.generated.secrets}"
default_secret_file_root="${SECRET_FILE_ROOT:-${REPO_ROOT}/.secrets}"
default_secret_backend="${SECRET_BACKEND:-file}"

mkdir -p "$(dirname "${config_file}")"

DEFAULT_REGISTRY="${REGISTRY:-REPLACE_WITH_REGISTRY_HOST}" \
DEFAULT_REGISTRY_USERNAME_SECRET_REF="${REGISTRY_USERNAME_SECRET_REF:-${default_secret_file_root}/registry/username}" \
DEFAULT_REGISTRY_PASSWORD_SECRET_REF="${REGISTRY_PASSWORD_SECRET_REF:-${default_secret_file_root}/registry/password}" \
DEFAULT_ENV_FILE="${default_env_file}" \
DEFAULT_SECRETS_ENV_FILE="${default_secrets_env_file}" \
DEFAULT_GENERATED_ENV_FILE="${default_generated_env_file}" \
DEFAULT_GENERATED_SECRET_ENV_FILE="${default_generated_secret_env_file}" \
DEFAULT_SECRET_BACKEND="${default_secret_backend}" \
DEFAULT_SHARED_ENV_SECRET_REF="${SHARED_ENV_SECRET_REF:-${default_env_file}}" \
DEFAULT_SECRETS_ENV_SECRET_REF="${SECRETS_ENV_SECRET_REF:-${default_secrets_env_file}}" \
DEFAULT_GENERATED_ENV_SECRET_REF="${GENERATED_ENV_SECRET_REF:-${default_generated_secret_env_file}}" \
DEFAULT_SECRET_FILE_ROOT="${default_secret_file_root}" \
DEFAULT_VAULT_ADDR="${VAULT_ADDR:-}" \
DEFAULT_VAULT_NAMESPACE="${VAULT_NAMESPACE:-}" \
DEFAULT_VAULT_TOKEN_FILE="${VAULT_TOKEN_FILE:-}" \
DEFAULT_VAULT_KV_MOUNT="${VAULT_KV_MOUNT:-secret}" \
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

# shellcheck disable=SC1090
source "${config_file}"
if [[ "${SECRET_BACKEND:-}" == "file" ]]; then
  registry_username_path="$(secret_file_path "${REGISTRY_USERNAME_SECRET_REF}")"
  registry_password_path="$(secret_file_path "${REGISTRY_PASSWORD_SECRET_REF}")"
  install -d -m 0700 "${SECRET_FILE_ROOT}" "$(dirname "${registry_username_path}")" "$(dirname "${registry_password_path}")"
  touch "${registry_username_path}" "${registry_password_path}"
  chmod 0600 "${registry_username_path}" "${registry_password_path}"
fi

log "remote deploy config is ready: ${config_file}"
