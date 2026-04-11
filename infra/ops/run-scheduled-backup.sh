#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

MODE="${1:-dump}"

load_env

[[ -n "${BACKUP_DATABASE_URL:-}" ]] || die "BACKUP_DATABASE_URL is required"
[[ -n "${REPLICATION_DATABASE_URL:-}" ]] || die "REPLICATION_DATABASE_URL is required"

timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
wal_archive_dir="${POSTGRES_WAL_ARCHIVE_DIR:-${REPO_ROOT}/infra/generated/postgres/wal-archive}"

prune_old_backups() {
  local dir="$1"
  local retention_days="$2"
  [[ -d "${dir}" ]] || return 0
  find "${dir}" -type f -mtime +"${retention_days}" -print -delete
}

case "${MODE}" in
  dump)
    logical_dir="${BACKUP_LOGICAL_DIR:-${REPO_ROOT}/backups/postgres/logical}"
    mkdir -p "${logical_dir}"
    BACKUP_MODE=dump "${SCRIPT_DIR}/backup-postgres.sh" "${logical_dir}/stuhelper-${timestamp}.dump"
    prune_old_backups "${logical_dir}" "${BACKUP_LOGICAL_RETENTION_DAYS:-7}"
    ;;
  basebackup)
    base_dir="${BACKUP_BASE_DIR:-${REPO_ROOT}/backups/postgres/base}"
    mkdir -p "${base_dir}"
    BACKUP_MODE=basebackup "${SCRIPT_DIR}/backup-postgres.sh" "${base_dir}/stuhelper-${timestamp}.tar.gz"
    prune_old_backups "${base_dir}" "${BACKUP_BASE_RETENTION_DAYS:-14}"
    ;;
  *)
    die "unsupported backup mode: ${MODE} (expected dump or basebackup)"
    ;;
esac

if [[ -d "${wal_archive_dir}" ]]; then
  prune_old_backups "${wal_archive_dir}" "${WAL_ARCHIVE_RETENTION_DAYS:-7}"
fi

./infra/ops/sync-postgres-backups.sh

log "scheduled PostgreSQL ${MODE} backup completed"
