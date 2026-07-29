#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"
# shellcheck source=lib/rclone-object-storage.sh
source "${SCRIPT_DIR}/lib/rclone-object-storage.sh"

require_cmd docker
load_env

[[ -n "${BACKUP_OBJECT_STORAGE_ENDPOINT:-}" ]] || die "BACKUP_OBJECT_STORAGE_ENDPOINT is required"
[[ -n "${BACKUP_OBJECT_STORAGE_BUCKET:-}" ]] || die "BACKUP_OBJECT_STORAGE_BUCKET is required"
[[ -n "${BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID:-}" ]] || die "BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID is required"
[[ -n "${BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY:-}" ]] || die "BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY is required"

logical_dir="${BACKUP_LOGICAL_DIR:-${REPO_ROOT}/backups/postgres/logical}"
base_dir="${BACKUP_BASE_DIR:-${REPO_ROOT}/backups/postgres/base}"
wal_archive_volume="${POSTGRES_WAL_ARCHIVE_VOLUME_NAME:-${STACK_NAME:-stuhelper}-postgres-wal-archive}"
prefix="${BACKUP_OBJECT_STORAGE_PREFIX:-postgres}"

mkdir -p "${logical_dir}" "${base_dir}"

run_backup_object_storage_rclone \
  "0:0" \
  "type=bind,src=${logical_dir},dst=/source,readonly" \
  copy /source "target:${BACKUP_OBJECT_STORAGE_BUCKET}/${prefix}/logical"
run_backup_object_storage_rclone \
  "0:0" \
  "type=bind,src=${base_dir},dst=/source,readonly" \
  copy /source "target:${BACKUP_OBJECT_STORAGE_BUCKET}/${prefix}/base"
run_backup_object_storage_rclone \
  "0:0" \
  "type=volume,src=${wal_archive_volume},dst=/source,readonly" \
  copy /source "target:${BACKUP_OBJECT_STORAGE_BUCKET}/${prefix}/wal"

log "synchronized PostgreSQL backup artifacts to object storage"
