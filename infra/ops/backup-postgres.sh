#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "Usage: $0 <output-file>" >&2
  exit 1
fi

mode="${BACKUP_MODE:-dump}"
logical_url="${BACKUP_DATABASE_URL:-}"
replication_url="${REPLICATION_DATABASE_URL:-}"

if [[ "${mode}" == "dump" && -z "${logical_url}" ]]; then
  echo "ERROR: BACKUP_DATABASE_URL is required" >&2
  exit 1
fi

if [[ "${mode}" == "basebackup" && -z "${replication_url}" ]]; then
  echo "ERROR: REPLICATION_DATABASE_URL is required" >&2
  exit 1
fi

if ! command -v pg_dump >/dev/null 2>&1; then
  if [[ "${mode}" == "dump" ]]; then
    echo "ERROR: pg_dump is required" >&2
    exit 1
  fi
fi

output_file="$1"
mkdir -p "$(dirname "$output_file")"

case "${mode}" in
  dump)
    echo "[backup] dumping PostgreSQL to $output_file"
    pg_dump --format=custom --no-owner --no-privileges --file "$output_file" "$logical_url"
    ;;
  basebackup)
    if ! command -v pg_basebackup >/dev/null 2>&1; then
      echo "ERROR: pg_basebackup is required for BACKUP_MODE=basebackup" >&2
      exit 1
    fi
    echo "[backup] creating PostgreSQL base backup to $output_file"
    pg_basebackup \
      --dbname "$replication_url" \
      --format=tar \
      --gzip \
      --wal-method=stream \
      --checkpoint=fast \
      --pgdata=- >"$output_file"
    ;;
  *)
    echo "ERROR: unsupported BACKUP_MODE=${mode}" >&2
    exit 1
    ;;
esac

sha256sum "$output_file" > "$output_file.sha256"
echo "[backup] wrote checksum $output_file.sha256"
