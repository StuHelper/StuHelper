#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

MODE="${1:-dump}"

load_env_preserving \
  BACKUP_OBJECT_STORAGE_OFF_HOST_REQUIRED \
  LOCAL_STATE_DIR \
  BACKUP_STAGING_DIR \
  EXTERNAL_POSTGRES_ENABLED \
  STACK_NAME \
  POSTGRES_CONTAINER_NAME
require_cmd docker
protected_bash=(
  /usr/bin/env
  --unset=BASH_ENV
  --unset=ENV
  /bin/bash
  --noprofile
  --norc
)

case "${MODE}" in
  dump | basebackup | sync) ;;
  *) die "unsupported backup mode: ${MODE} (expected dump, basebackup, or sync)" ;;
esac

# Timers may be activated after configuration is ready but before the first
# datastore deployment starts. A missing marker is a successful no-op only
# while the host is demonstrably empty: no surviving release evidence, no
# external PostgreSQL selection, and no local PostgreSQL container or data
# volume. Once any datastore can contain production data, losing the marker is
# a control-plane inconsistency and must make systemd/alerting fail closed.
current_release_file="${DEPLOY_STATE_DIR}/current-release.env"
if [[ ! -e "${current_release_file}" && ! -L "${current_release_file}" ]]; then
  releases_dir="${DEPLOY_STATE_DIR}/releases"
  releases_log="${DEPLOY_STATE_DIR}/releases.log"
  if [[ -e "${releases_log}" || -L "${releases_log}" ]]; then
    die "committed release marker is missing while release-log evidence survives: ${releases_log}"
  fi
  if [[ -L "${releases_dir}" || ( -e "${releases_dir}" && ! -d "${releases_dir}" ) ]]; then
    die "release record path is not a regular directory: ${releases_dir}"
  fi
  shopt -s nullglob
  surviving_release_records=("${releases_dir}"/*.env)
  shopt -u nullglob
  if ((${#surviving_release_records[@]} != 0)); then
    die "committed release marker is missing while immutable per-tag evidence survives"
  fi
  if [[ "${EXTERNAL_POSTGRES_ENABLED:-false}" == "true" ]]; then
    die "committed release marker is missing while external PostgreSQL is selected"
  fi
  docker info --format '{{json .ServerVersion}}' >/dev/null 2>&1 ||
    die "cannot verify an empty bootstrap host because the Docker daemon is unavailable"
  stack_name="${STACK_NAME:-stuhelper}"
  postgres_container="${POSTGRES_CONTAINER_NAME:-${stack_name}-postgres}"
  postgres_data_volume="${stack_name}-postgres-data"
  if docker container inspect "${postgres_container}" >/dev/null 2>&1; then
    die "committed release marker is missing while PostgreSQL container evidence survives: ${postgres_container}"
  fi
  if docker volume inspect "${postgres_data_volume}" >/dev/null 2>&1; then
    die "committed release marker is missing while PostgreSQL data-volume evidence survives: ${postgres_data_volume}"
  fi
  log "scheduled PostgreSQL ${MODE} deferred: no committed release or datastore evidence exists yet"
  exit 0
fi
if [[ ! -f "${current_release_file}" || -L "${current_release_file}" ]]; then
  die "committed release marker must be a regular non-symlink file: ${current_release_file}"
fi
(
  source_release_record_env_file "${current_release_file}"
  immutable_release_file="${DEPLOY_STATE_DIR}/releases/${TAG}.env"
  [[ -f "${immutable_release_file}" && ! -L "${immutable_release_file}" ]] ||
    die "committed release is missing its immutable per-tag record: ${immutable_release_file}"
  cmp -s "${current_release_file}" "${immutable_release_file}" ||
    die "committed release marker does not match its immutable per-tag record"
)

case "${BACKUP_OBJECT_STORAGE_OFF_HOST_REQUIRED:-false}" in
  true) require_off_host_backup_object_storage ;;
  false|"") ;;
  *) die "BACKUP_OBJECT_STORAGE_OFF_HOST_REQUIRED must be true or false" ;;
esac

if [[ "${MODE}" == "sync" ]]; then
  "${protected_bash[@]}" "${SCRIPT_DIR}/sync-postgres-backups.sh"
  log "scheduled PostgreSQL backup sync completed"
  exit 0
fi

[[ -n "${BACKUP_DATABASE_URL:-}" ]] || die "BACKUP_DATABASE_URL is required"
[[ -n "${REPLICATION_DATABASE_URL:-}" ]] || die "REPLICATION_DATABASE_URL is required"

timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
wal_archive_volume="${POSTGRES_WAL_ARCHIVE_VOLUME_NAME:-${STACK_NAME:-stuhelper}-postgres-wal-archive}"
postgres_image_ref="${POSTGRES_IMAGE_REF:-cgr.dev/chainguard/postgres:latest@sha256:dc2f04037c1044a22af76cee4de70b9111885b17c561b939d7ed70103d100759}"
[[ "${postgres_image_ref}" =~ ^.+@sha256:[0-9a-f]{64}$ ]] ||
  die "POSTGRES_IMAGE_REF must be a complete image@sha256 reference"

prune_old_backups() {
  local dir="$1"
  local retention_days="$2"
  [[ -d "${dir}" ]] || return 0
  find "${dir}" -type f -mtime +"${retention_days}" -print -delete
}

case "${MODE}" in
  dump)
    backup_dir="${BACKUP_LOGICAL_DIR:-${REPO_ROOT}/backups/postgres/logical}"
    backup_extension="dump"
    retention_days="${BACKUP_LOGICAL_RETENTION_DAYS:-14}"
    ;;
  basebackup)
    backup_dir="${BACKUP_BASE_DIR:-${REPO_ROOT}/backups/postgres/base}"
    backup_extension="tar.gz"
    retention_days="${BACKUP_BASE_RETENTION_DAYS:-30}"
    ;;
esac

mkdir -p "${backup_dir}"
BACKUP_MODE="${MODE}" \
  "${protected_bash[@]}" \
  "${SCRIPT_DIR}/backup-postgres.sh" \
  "${backup_dir}/stuhelper-${timestamp}.${backup_extension}"

# Never delete a local recovery artifact until rclone has confirmed that the
# complete logical/base/WAL set is present in the independent backup target.
"${protected_bash[@]}" "${SCRIPT_DIR}/sync-postgres-backups.sh"

prune_old_backups "${backup_dir}" "${retention_days}"

if [[ "${EXTERNAL_POSTGRES_ENABLED:-false}" == "true" ]]; then
  log "external PostgreSQL selected; skipping local WAL archive pruning"
else
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
fi

log "scheduled PostgreSQL ${MODE} backup completed"
