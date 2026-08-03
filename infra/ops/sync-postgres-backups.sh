#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"
# shellcheck source=lib/rclone-object-storage.sh
source "${SCRIPT_DIR}/lib/rclone-object-storage.sh"

require_cmd docker
load_env_preserving BACKUP_OBJECT_STORAGE_OFF_HOST_REQUIRED LOCAL_STATE_DIR
unset BACKUP_OBJECT_STORAGE_PINNED_HOSTS

case "${BACKUP_OBJECT_STORAGE_OFF_HOST_REQUIRED:-false}" in
  true) require_off_host_backup_object_storage ;;
  false|"")
    if [[ "${APP_ENV:-}" == "production" ]]; then
      require_off_host_backup_object_storage
    fi
    ;;
  *) die "BACKUP_OBJECT_STORAGE_OFF_HOST_REQUIRED must be true or false" ;;
esac

[[ -n "${BACKUP_OBJECT_STORAGE_ENDPOINT:-}" ]] || die "BACKUP_OBJECT_STORAGE_ENDPOINT is required"
[[ -n "${BACKUP_OBJECT_STORAGE_BUCKET:-}" ]] || die "BACKUP_OBJECT_STORAGE_BUCKET is required"
[[ -n "${BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID:-}" ]] || die "BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID is required"
[[ -n "${BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY:-}" ]] || die "BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY is required"

logical_dir="${BACKUP_LOGICAL_DIR:-${REPO_ROOT}/backups/postgres/logical}"
base_dir="${BACKUP_BASE_DIR:-${REPO_ROOT}/backups/postgres/base}"
wal_archive_volume="${POSTGRES_WAL_ARCHIVE_VOLUME_NAME:-${STACK_NAME:-stuhelper}-postgres-wal-archive}"
prefix="${BACKUP_OBJECT_STORAGE_PREFIX:-postgres}"
host_user="$(id -u):$(id -g)"
wal_archive_user="${POSTGRES_WAL_ARCHIVE_CONTAINER_USER:-70:70}"
external_postgres_enabled="${EXTERNAL_POSTGRES_ENABLED:-false}"

case "${external_postgres_enabled}" in
  true|false) ;;
  *) die "EXTERNAL_POSTGRES_ENABLED must be true or false" ;;
esac

mkdir -p "${logical_dir}" "${base_dir}"
if [[ "${external_postgres_enabled}" != "true" ]]; then
  [[ "${wal_archive_user}" =~ ^[0-9]+:[0-9]+$ ]] ||
    die "POSTGRES_WAL_ARCHIVE_CONTAINER_USER must be a numeric uid:gid pair"
  require_live_postgres_wal_archive_volume \
    "${wal_archive_volume}" \
    "${STACK_NAME:-stuhelper}-postgres"
  require_live_postgres_wal_archiving \
    "${STACK_NAME:-stuhelper}-postgres" \
    "${POSTGRES_USER:-stuhelper}" \
    "${POSTGRES_DB:-stuhelper}"
else
  require_external_postgres_pitr_evidence >/dev/null
fi

sync_excludes=(
  --exclude '*.partial'
  --exclude '*.partial.*'
  --exclude '*.tmp'
  --exclude '*.tmp.*'
  --exclude '.stuhelper-postgres-backup-staging/**'
  --exclude 'backup-staging/**'
  --exclude 'quarantine-incomplete-*/**'
)

run_backup_object_storage_rclone \
  "${host_user}" \
  "type=bind,src=${logical_dir},dst=/source,readonly" \
  copy /source "target:${BACKUP_OBJECT_STORAGE_BUCKET}/${prefix}/logical" \
  "${sync_excludes[@]}"
run_backup_object_storage_rclone \
  "${host_user}" \
  "type=bind,src=${base_dir},dst=/source,readonly" \
  copy /source "target:${BACKUP_OBJECT_STORAGE_BUCKET}/${prefix}/base" \
  "${sync_excludes[@]}"
if [[ "${external_postgres_enabled}" == "true" ]]; then
  log "external PostgreSQL selected; fresh cluster-bound continuous WAL/PITR evidence was verified"
else
  run_backup_object_storage_rclone \
    "${wal_archive_user}" \
    "type=volume,src=${wal_archive_volume},dst=/source,readonly" \
    copy /source "target:${BACKUP_OBJECT_STORAGE_BUCKET}/${prefix}/wal" \
    "${sync_excludes[@]}"
fi

log "synchronized PostgreSQL backup artifacts to object storage"
