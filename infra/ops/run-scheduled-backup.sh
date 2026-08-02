#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

MODE="${1:-dump}"

load_env_preserving BACKUP_OBJECT_STORAGE_OFF_HOST_REQUIRED
require_cmd docker

[[ -n "${BACKUP_DATABASE_URL:-}" ]] || die "BACKUP_DATABASE_URL is required"
[[ -n "${REPLICATION_DATABASE_URL:-}" ]] || die "REPLICATION_DATABASE_URL is required"

timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
wal_archive_volume="${POSTGRES_WAL_ARCHIVE_VOLUME_NAME:-${STACK_NAME:-stuhelper}-postgres-wal-archive}"

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
    prune_old_backups "${logical_dir}" "${BACKUP_LOGICAL_RETENTION_DAYS:-14}"
    ;;
  basebackup)
    base_dir="${BACKUP_BASE_DIR:-${REPO_ROOT}/backups/postgres/base}"
    mkdir -p "${base_dir}"
    BACKUP_MODE=basebackup "${SCRIPT_DIR}/backup-postgres.sh" "${base_dir}/stuhelper-${timestamp}.tar.gz"
    prune_old_backups "${base_dir}" "${BACKUP_BASE_RETENTION_DAYS:-30}"
    ;;
  *)
    die "unsupported backup mode: ${MODE} (expected dump or basebackup)"
    ;;
esac

postgres_image_ref="${POSTGRES_IMAGE_REF:-cgr.dev/chainguard/postgres:latest@sha256:dc2f04037c1044a22af76cee4de70b9111885b17c561b939d7ed70103d100759}"
[[ "${postgres_image_ref}" =~ ^.+@sha256:[0-9a-f]{64}$ ]] ||
  die "POSTGRES_IMAGE_REF must be a complete image@sha256 reference"

docker run --rm \
  --read-only \
  --cap-drop ALL \
  --security-opt no-new-privileges \
  --user 70:70 \
  --mount "type=volume,src=${wal_archive_volume},dst=/wal" \
  --entrypoint /bin/sh \
  "${postgres_image_ref}" -ec "
    if [ -d /wal ]; then
      find /wal \
        -path '/wal/quarantine-incomplete-*' -prune -o \
        -type f -mtime +${WAL_ARCHIVE_RETENTION_DAYS:-14} -print -delete
    fi
  "

./infra/ops/sync-postgres-backups.sh

log "scheduled PostgreSQL ${MODE} backup completed"
