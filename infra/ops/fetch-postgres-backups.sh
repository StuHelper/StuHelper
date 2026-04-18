#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

MODE="${1:-all}"

require_cmd docker
load_env

[[ -n "${BACKUP_OBJECT_STORAGE_ENDPOINT:-}" ]] || die "BACKUP_OBJECT_STORAGE_ENDPOINT is required"
[[ -n "${BACKUP_OBJECT_STORAGE_BUCKET:-}" ]] || die "BACKUP_OBJECT_STORAGE_BUCKET is required"
[[ -n "${BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID:-}" ]] || die "BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID is required"
[[ -n "${BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY:-}" ]] || die "BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY is required"

logical_dir="${BACKUP_LOGICAL_DIR:-${REPO_ROOT}/backups/postgres/logical}"
base_dir="${BACKUP_BASE_DIR:-${REPO_ROOT}/backups/postgres/base}"
wal_archive_dir="${POSTGRES_WAL_RESTORE_DIR:-${LOCAL_STATE_DIR}/postgres/wal-restore}"
prefix="${BACKUP_OBJECT_STORAGE_PREFIX:-postgres}"
secure_flag=()
if [[ "${BACKUP_OBJECT_STORAGE_TLS_INSECURE:-false}" == "true" ]]; then
  secure_flag+=(--insecure)
fi

mkdir -p "${logical_dir}" "${base_dir}" "${wal_archive_dir}"

case "${MODE}" in
  all|logical|base|wal)
    ;;
  *)
    die "unsupported fetch mode: ${MODE} (expected all, logical, base, or wal)"
    ;;
esac

docker run --rm \
  -e MC_HOST_target="${BACKUP_OBJECT_STORAGE_ENDPOINT}" \
  -e MC_ACCESS_KEY="${BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID}" \
  -e MC_SECRET_KEY="${BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY}" \
  -e BACKUP_BUCKET="${BACKUP_OBJECT_STORAGE_BUCKET}" \
  -e BACKUP_PREFIX="${prefix}" \
  -e FETCH_MODE="${MODE}" \
  -v "${logical_dir}:/restore/logical" \
  -v "${base_dir}:/restore/base" \
  -v "${wal_archive_dir}:/restore/wal" \
  minio/mc:RELEASE.2025-03-12T17-29-24Z /bin/sh -ec "
    mc alias set target \"\${MC_HOST_target}\" \"\${MC_ACCESS_KEY}\" \"\${MC_SECRET_KEY}\" ${secure_flag[*]} >/dev/null
    case \"\${FETCH_MODE}\" in
      all|logical)
        mc mirror --overwrite \"target/\${BACKUP_BUCKET}/\${BACKUP_PREFIX}/logical\" /restore/logical >/dev/null
        ;;
    esac
    case \"\${FETCH_MODE}\" in
      all|base)
        mc mirror --overwrite \"target/\${BACKUP_BUCKET}/\${BACKUP_PREFIX}/base\" /restore/base >/dev/null
        ;;
    esac
    case \"\${FETCH_MODE}\" in
      all|wal)
        mc mirror --overwrite \"target/\${BACKUP_BUCKET}/\${BACKUP_PREFIX}/wal\" /restore/wal >/dev/null
        ;;
    esac
  "

log "fetched PostgreSQL backup artifacts from object storage (${MODE})"
