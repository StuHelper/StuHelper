#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 2 ]]; then
  echo "Usage: ALLOW_DESTRUCTIVE=1 $0 <basebackup-file.tar.gz> <pgdata-dir>" >&2
  exit 1
fi

if [[ "${ALLOW_DESTRUCTIVE:-0}" != "1" ]]; then
  echo "ERROR: set ALLOW_DESTRUCTIVE=1 to acknowledge base backup restore is destructive" >&2
  exit 1
fi

backup_file="$1"
pgdata_dir="$2"
wal_archive_dir="${WAL_ARCHIVE_DIR:-}"

if [[ ! -f "${backup_file}" ]]; then
  echo "ERROR: backup file not found: ${backup_file}" >&2
  exit 1
fi

if [[ -f "${backup_file}.sha256" ]]; then
  expected_sha256="$(awk 'NR == 1 { print $1 }' "${backup_file}.sha256")"
  actual_sha256="$(sha256sum "${backup_file}" | awk '{ print $1 }')"
  if [[ ! "${expected_sha256}" =~ ^[0-9a-fA-F]{64}$ ||
        "${actual_sha256}" != "${expected_sha256,,}" ]]; then
    echo "ERROR: base backup SHA256 verification failed: ${backup_file}" >&2
    exit 1
  fi
fi

tar -tzf "${backup_file}" >/dev/null

if [[ -L "${pgdata_dir}" ]]; then
  echo "ERROR: refusing to restore through a symlinked PGDATA directory: ${pgdata_dir}" >&2
  exit 1
fi
mkdir -p "${pgdata_dir}"
rm -rf "${pgdata_dir:?}/"*
tar -xzf "${backup_file}" -C "${pgdata_dir}"

if [[ ! -s "${pgdata_dir}/PG_VERSION" || ! -s "${pgdata_dir}/backup_manifest" ]]; then
  echo "ERROR: extracted archive is not a verified pg_basebackup layout" >&2
  exit 1
fi

if [[ -n "${wal_archive_dir}" ]]; then
  cat >>"${pgdata_dir}/postgresql.auto.conf" <<EOF
restore_command = 'cp ${wal_archive_dir}/%f %p'
EOF
  touch "${pgdata_dir}/recovery.signal"
fi

echo "[restore] extracted base backup into ${pgdata_dir}"
if [[ -n "${wal_archive_dir}" ]]; then
  echo "[restore] configured restore_command against ${wal_archive_dir}"
fi
