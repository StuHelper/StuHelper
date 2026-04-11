#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

require_cmd docker
require_cmd curl
require_cmd jq
require_cmd python3
require_cmd openssl

if [[ -n "${SHARED_ENV_SECRET_REF:-}" && -z "${SECRET_BACKEND:-}" ]]; then
  die "SECRET_BACKEND must be set when SHARED_ENV_SECRET_REF is provided"
fi
if [[ -n "${SECRETS_ENV_SECRET_REF:-}" && -z "${SECRET_BACKEND:-}" ]]; then
  die "SECRET_BACKEND must be set when SECRETS_ENV_SECRET_REF is provided"
fi

ensure_generated_files
if [[ -n "${SHARED_ENV_SECRET_REF:-}" ]]; then
  mkdir -p "$(dirname "${ENV_FILE}")"
  touch "${ENV_FILE}"
fi
if [[ -n "${SECRETS_ENV_SECRET_REF:-}" && -n "${SECRETS_ENV_FILE:-}" ]]; then
  mkdir -p "$(dirname "${SECRETS_ENV_FILE}")"
  touch "${SECRETS_ENV_FILE}"
fi

pending_generated_secret_ref="${GENERATED_ENV_SECRET_REF:-}"
unset GENERATED_ENV_SECRET_REF
load_env
if [[ -n "${pending_generated_secret_ref}" ]]; then
  export GENERATED_ENV_SECRET_REF="${pending_generated_secret_ref}"
fi

docker info >/dev/null
docker compose version >/dev/null

mkdir -p \
  "${REPO_ROOT}/backups/postgres/logical" \
  "${REPO_ROOT}/backups/postgres/base" \
  "${REPO_ROOT}/infra/generated/postgres/wal-archive" \
  "${DEPLOY_STATE_DIR}"

if command -v systemctl >/dev/null 2>&1; then
  for unit in     stuhelper-postgres-dump-backup.timer     stuhelper-postgres-basebackup.timer     stuhelper-postgres-backup-sync.timer; do
    if ! systemctl list-unit-files | grep -q "^${unit}"; then
      die "backup timer ${unit} is not installed on the target host"
    fi
    if ! systemctl is-enabled --quiet "${unit}"; then
      die "backup timer ${unit} is not enabled"
    fi
  done
fi

[[ -n "${BACKUP_DATABASE_URL:-}" ]] || die "BACKUP_DATABASE_URL must be configured"
[[ -n "${REPLICATION_DATABASE_URL:-}" ]] || die "REPLICATION_DATABASE_URL must be configured"
[[ -n "${BACKUP_OBJECT_STORAGE_BUCKET:-}" ]] || die "BACKUP_OBJECT_STORAGE_BUCKET must be configured"

log "remote preflight checks passed"
