#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
TARGET_SCRIPT="${REPO_ROOT}/infra/ops/init-remote-deploy-config.sh"

grep -qF 'source_remote_deploy_config_env_file "${config_file}"' "${TARGET_SCRIPT}" || {
  echo "[init-remote-deploy-config-contract][error] remote deploy config must use its allowlisted loader" >&2
  exit 1
}
if grep -qF 'source "${config_file}"' "${TARGET_SCRIPT}"; then
  echo "[init-remote-deploy-config-contract][error] remote deploy config must not be raw-sourced" >&2
  exit 1
fi

assert_contains() {
  local file="$1"
  local expected="$2"
  grep -Fqx "${expected}" "${file}" || {
    echo "[init-remote-deploy-config-contract][error] missing line: ${expected}" >&2
    echo "--- ${file} ---" >&2
    cat "${file}" >&2
    exit 1
  }
}

tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT

default_cfg="${tmpdir}/default.remote.env"
SECRET_FILE_ROOT="${tmpdir}/.secrets" \
REMOTE_DEPLOY_CONFIG_FILE="${default_cfg}" \
bash "${TARGET_SCRIPT}" >/dev/null

assert_contains "${default_cfg}" "ENV_FILE=${REPO_ROOT}/.env.prod.shared"
assert_contains "${default_cfg}" "REGISTRY_AUTH_MODE=workflow-token"
assert_contains "${default_cfg}" "BACKUP_SERVICE_GROUP=$(id -gn)"
assert_contains "${default_cfg}" "SECRETS_ENV_FILE=${REPO_ROOT}/.env.prod.secrets"
assert_contains "${default_cfg}" "GENERATED_ENV_FILE=${REPO_ROOT}/.env.prod.generated"
assert_contains "${default_cfg}" "GENERATED_SECRET_ENV_FILE=${REPO_ROOT}/.env.prod.generated.secrets"
assert_contains "${default_cfg}" "SECRET_BACKEND=vault-kv-v2"
assert_contains "${default_cfg}" "SHARED_ENV_SECRET_REF=secret/stuhelper/prod/shared-env"
assert_contains "${default_cfg}" "SECRETS_ENV_SECRET_REF=secret/stuhelper/prod/secrets-env"
assert_contains "${default_cfg}" "GENERATED_ENV_SECRET_REF=secret/stuhelper/prod/generated-secrets-env"
assert_contains "${default_cfg}" "SECRET_FILE_ROOT=${tmpdir}/.secrets"
assert_contains "${default_cfg}" "VAULT_TOKEN_FILE=${tmpdir}/.secrets/vault/token"
assert_contains "${default_cfg}" "VAULT_RUNTIME_TOKEN_POLICY=stuhelper-production-deploy"
assert_contains "${default_cfg}" "VAULT_RUNTIME_TOKEN_PERIOD_SECONDS=259200"
assert_contains "${default_cfg}" "VAULT_RUNTIME_TOKEN_MIN_TTL_SECONDS=43200"

override_cfg="${tmpdir}/override.remote.env"
REMOTE_DEPLOY_CONFIG_FILE="${override_cfg}" \
REGISTRY_AUTH_MODE="persistent-secret" \
BACKUP_SERVICE_GROUP="deployment" \
ENV_FILE="/srv/stuhelper/shared.env" \
SECRETS_ENV_FILE="/srv/stuhelper/secrets.env" \
GENERATED_ENV_FILE="/srv/stuhelper/generated.env" \
GENERATED_SECRET_ENV_FILE="/srv/stuhelper/generated.secrets.env" \
SECRET_BACKEND="vault-kv-v2" \
SECRET_FILE_ROOT="/srv/stuhelper/.secrets" \
SHARED_ENV_SECRET_REF="secret/custom/shared" \
SECRETS_ENV_SECRET_REF="secret/custom/secrets" \
GENERATED_ENV_SECRET_REF="secret/custom/generated" \
VAULT_ADDR="https://vault.example.com" \
VAULT_NAMESPACE="platform/prod" \
VAULT_TOKEN_FILE="/srv/stuhelper/.secrets/vault/token" \
VAULT_KV_MOUNT="platform" \
VAULT_RUNTIME_TOKEN_POLICY="custom-prod-deploy" \
VAULT_RUNTIME_TOKEN_PERIOD_SECONDS="345600" \
VAULT_RUNTIME_TOKEN_MIN_TTL_SECONDS="86400" \
bash "${TARGET_SCRIPT}" >/dev/null

assert_contains "${override_cfg}" "ENV_FILE=/srv/stuhelper/shared.env"
assert_contains "${override_cfg}" "REGISTRY_AUTH_MODE=persistent-secret"
assert_contains "${override_cfg}" "BACKUP_SERVICE_GROUP=deployment"
assert_contains "${override_cfg}" "SECRETS_ENV_FILE=/srv/stuhelper/secrets.env"
assert_contains "${override_cfg}" "GENERATED_ENV_FILE=/srv/stuhelper/generated.env"
assert_contains "${override_cfg}" "GENERATED_SECRET_ENV_FILE=/srv/stuhelper/generated.secrets.env"
assert_contains "${override_cfg}" "SECRET_BACKEND=vault-kv-v2"
assert_contains "${override_cfg}" "SHARED_ENV_SECRET_REF=secret/custom/shared"
assert_contains "${override_cfg}" "SECRETS_ENV_SECRET_REF=secret/custom/secrets"
assert_contains "${override_cfg}" "GENERATED_ENV_SECRET_REF=secret/custom/generated"
assert_contains "${override_cfg}" "SECRET_FILE_ROOT=/srv/stuhelper/.secrets"
assert_contains "${override_cfg}" "VAULT_ADDR=https://vault.example.com"
assert_contains "${override_cfg}" "VAULT_NAMESPACE=platform/prod"
assert_contains "${override_cfg}" "VAULT_TOKEN_FILE=/srv/stuhelper/.secrets/vault/token"
assert_contains "${override_cfg}" "VAULT_KV_MOUNT=platform"
assert_contains "${override_cfg}" "VAULT_RUNTIME_TOKEN_POLICY=custom-prod-deploy"
assert_contains "${override_cfg}" "VAULT_RUNTIME_TOKEN_PERIOD_SECONDS=345600"
assert_contains "${override_cfg}" "VAULT_RUNTIME_TOKEN_MIN_TTL_SECONDS=86400"

preserved_cfg="${tmpdir}/preserved.remote.env"
cp "${override_cfg}" "${preserved_cfg}"
REMOTE_DEPLOY_CONFIG_FILE="${preserved_cfg}" \
REMOTE_DEPLOY_CONFIG_PRESERVE_EXISTING=true \
REGISTRY="REPLACE_WITH_REGISTRY_HOST" \
BACKUP_SERVICE_GROUP="replacement-group" \
VAULT_ADDR="REPLACE_WITH_VAULT_ADDR" \
SHARED_ENV_SECRET_REF="secret/stuhelper/prod/shared-env" \
bash "${TARGET_SCRIPT}" >/dev/null

assert_contains "${preserved_cfg}" "REGISTRY_AUTH_MODE=persistent-secret"
assert_contains "${preserved_cfg}" "BACKUP_SERVICE_GROUP=deployment"
assert_contains "${preserved_cfg}" "VAULT_ADDR=https://vault.example.com"
assert_contains "${preserved_cfg}" "VAULT_NAMESPACE=platform/prod"
assert_contains "${preserved_cfg}" "SHARED_ENV_SECRET_REF=secret/custom/shared"

if REMOTE_DEPLOY_CONFIG_FILE="${tmpdir}/invalid-preserve.remote.env" \
  REMOTE_DEPLOY_CONFIG_PRESERVE_EXISTING=maybe \
  bash "${TARGET_SCRIPT}" >"${tmpdir}/invalid-preserve.out" 2>"${tmpdir}/invalid-preserve.err"; then
  echo "[init-remote-deploy-config-contract][error] initializer accepted an invalid preservation mode" >&2
  exit 1
fi
grep -q 'REMOTE_DEPLOY_CONFIG_PRESERVE_EXISTING must be true or false' "${tmpdir}/invalid-preserve.err" || {
  echo "[init-remote-deploy-config-contract][error] invalid preservation mode was not reported" >&2
  exit 1
}

invalid_cfg="${tmpdir}/invalid.remote.env"
printf 'REGISTRY=ghcr.io\nPATH=/tmp/payload\n' >"${invalid_cfg}"
invalid_before="$(<"${invalid_cfg}")"
if REMOTE_DEPLOY_CONFIG_FILE="${invalid_cfg}" bash "${TARGET_SCRIPT}" >"${tmpdir}/invalid.out" 2>"${tmpdir}/invalid.err"; then
  echo "[init-remote-deploy-config-contract][error] initializer accepted an unknown existing key" >&2
  exit 1
fi
grep -q 'process-control variable PATH is not allowed in StuHelper environment files' "${tmpdir}/invalid.err" || {
  echo "[init-remote-deploy-config-contract][error] existing-state rejection was not explicit" >&2
  exit 1
}
[[ "$(<"${invalid_cfg}")" == "${invalid_before}" ]] || {
  echo "[init-remote-deploy-config-contract][error] initializer rewrote invalid existing state before rejecting it" >&2
  exit 1
}

echo "[init-remote-deploy-config-contract] all assertions passed"
