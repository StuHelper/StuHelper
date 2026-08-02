#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"
# shellcheck source=lib/rclone-object-storage.sh
source "${SCRIPT_DIR}/lib/rclone-object-storage.sh"

MODE="${1:-all}"

require_cmd docker
load_env
unset BACKUP_OBJECT_STORAGE_PINNED_HOSTS

[[ -n "${BACKUP_OBJECT_STORAGE_ENDPOINT:-}" ]] || die "BACKUP_OBJECT_STORAGE_ENDPOINT is required"
[[ -n "${BACKUP_OBJECT_STORAGE_BUCKET:-}" ]] || die "BACKUP_OBJECT_STORAGE_BUCKET is required"
[[ -n "${BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID:-}" ]] || die "BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID is required"
[[ -n "${BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY:-}" ]] || die "BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY is required"

logical_dir="${BACKUP_LOGICAL_DIR:-${REPO_ROOT}/backups/postgres/logical}"
base_dir="${BACKUP_BASE_DIR:-${REPO_ROOT}/backups/postgres/base}"
wal_archive_dir="${POSTGRES_WAL_RESTORE_DIR:-${LOCAL_STATE_DIR}/postgres/wal-restore}"
prefix="${BACKUP_OBJECT_STORAGE_PREFIX:-postgres}"

mkdir -p "${logical_dir}" "${base_dir}" "${wal_archive_dir}"

case "${MODE}" in
  all|logical|base|wal)
    ;;
  *)
    die "unsupported fetch mode: ${MODE} (expected all, logical, base, or wal)"
    ;;
esac

host_user="$(id -u):$(id -g)"

if [[ "${MODE}" == "all" || "${MODE}" == "logical" ]]; then
  run_backup_object_storage_rclone \
    "${host_user}" \
    "type=bind,src=${logical_dir},dst=/restore" \
    copy "target:${BACKUP_OBJECT_STORAGE_BUCKET}/${prefix}/logical" /restore
fi
if [[ "${MODE}" == "all" || "${MODE}" == "base" ]]; then
  run_backup_object_storage_rclone \
    "${host_user}" \
    "type=bind,src=${base_dir},dst=/restore" \
    copy "target:${BACKUP_OBJECT_STORAGE_BUCKET}/${prefix}/base" /restore
fi
if [[ "${MODE}" == "all" || "${MODE}" == "wal" ]]; then
  run_backup_object_storage_rclone \
    "${host_user}" \
    "type=bind,src=${wal_archive_dir},dst=/restore" \
    copy "target:${BACKUP_OBJECT_STORAGE_BUCKET}/${prefix}/wal" /restore
fi

log "fetched PostgreSQL backup artifacts from object storage (${MODE})"
