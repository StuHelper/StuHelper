#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

require_cmd docker
load_env

[[ -n "${BACKUP_OBJECT_STORAGE_ENDPOINT:-}" ]] || die "BACKUP_OBJECT_STORAGE_ENDPOINT is required"
[[ -n "${BACKUP_OBJECT_STORAGE_BUCKET:-}" ]] || die "BACKUP_OBJECT_STORAGE_BUCKET is required"
[[ -n "${BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID:-}" ]] || die "BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID is required"
[[ -n "${BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY:-}" ]] || die "BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY is required"

logical_dir="${BACKUP_LOGICAL_DIR:-${REPO_ROOT}/backups/postgres/logical}"
base_dir="${BACKUP_BASE_DIR:-${REPO_ROOT}/backups/postgres/base}"
wal_archive_dir="${POSTGRES_WAL_ARCHIVE_DIR:-${REPO_ROOT}/infra/generated/postgres/wal-archive}"
prefix="${BACKUP_OBJECT_STORAGE_PREFIX:-postgres}"
secure_flag=()
if [[ "${BACKUP_OBJECT_STORAGE_TLS_INSECURE:-false}" == "true" ]]; then
  secure_flag+=(--insecure)
fi

mkdir -p "${logical_dir}" "${base_dir}" "${wal_archive_dir}"

docker run --rm \
  -e MC_HOST_target="${BACKUP_OBJECT_STORAGE_ENDPOINT}" \
  -e MC_ACCESS_KEY="${BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID}" \
  -e MC_SECRET_KEY="${BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY}" \
  -e BACKUP_BUCKET="${BACKUP_OBJECT_STORAGE_BUCKET}" \
  -e BACKUP_PREFIX="${prefix}" \
  -v "${logical_dir}:/backup/logical:ro" \
  -v "${base_dir}:/backup/base:ro" \
  -v "${wal_archive_dir}:/backup/wal:ro" \
  minio/mc:RELEASE.2025-03-12T17-29-24Z /bin/sh -ec "
    mc alias set target \"\${MC_HOST_target}\" \"\${MC_ACCESS_KEY}\" \"\${MC_SECRET_KEY}\" ${secure_flag[*]} >/dev/null
    mc mb --ignore-existing \"target/\${BACKUP_BUCKET}\" >/dev/null
    mc mirror --overwrite /backup/logical \"target/\${BACKUP_BUCKET}/\${BACKUP_PREFIX}/logical\" >/dev/null
    mc mirror --overwrite /backup/base \"target/\${BACKUP_BUCKET}/\${BACKUP_PREFIX}/base\" >/dev/null
    mc mirror --overwrite /backup/wal \"target/\${BACKUP_BUCKET}/\${BACKUP_PREFIX}/wal\" >/dev/null
  "

log "synchronized PostgreSQL backup artifacts to object storage"
