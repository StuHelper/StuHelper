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

[[ -f "${ENV_FILE}" ]] || die "missing ${ENV_FILE}"
if [[ -n "${SECRETS_ENV_FILE:-}" ]]; then
  [[ -f "${SECRETS_ENV_FILE}" ]] || die "missing ${SECRETS_ENV_FILE}"
fi

load_env

docker info >/dev/null
docker compose version >/dev/null

mkdir -p \
  "${REPO_ROOT}/backups/postgres/logical" \
  "${REPO_ROOT}/backups/postgres/base" \
  "${REPO_ROOT}/infra/generated/postgres/wal-archive" \
  "${DEPLOY_STATE_DIR}"

if command -v systemctl >/dev/null 2>&1; then
  if ! systemctl list-unit-files | grep -q '^stuhelper-postgres-dump-backup.timer'; then
    die "backup timer stuhelper-postgres-dump-backup.timer is not installed on the target host"
  fi
  if ! systemctl is-enabled --quiet stuhelper-postgres-dump-backup.timer; then
    die "backup timer stuhelper-postgres-dump-backup.timer is not enabled"
  fi
  if ! systemctl is-enabled --quiet stuhelper-postgres-basebackup.timer; then
    die "backup timer stuhelper-postgres-basebackup.timer is not enabled"
  fi
fi

log "remote preflight checks passed"
