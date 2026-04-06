#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "Usage: $0 <output-file>" >&2
  exit 1
fi

if [[ -z "${DATABASE_URL:-}" ]]; then
  echo "ERROR: DATABASE_URL is required" >&2
  exit 1
fi

if ! command -v pg_dump >/dev/null 2>&1; then
  if [[ "${BACKUP_MODE:-dump}" == "dump" ]]; then
    echo "ERROR: pg_dump is required" >&2
    exit 1
  fi
fi

output_file="$1"
mkdir -p "$(dirname "$output_file")"

case "${BACKUP_MODE:-dump}" in
  dump)
    echo "[backup] dumping PostgreSQL to $output_file"
    pg_dump --format=custom --no-owner --no-privileges --file "$output_file" "$DATABASE_URL"
    ;;
  basebackup)
    if ! command -v pg_basebackup >/dev/null 2>&1; then
      echo "ERROR: pg_basebackup is required for BACKUP_MODE=basebackup" >&2
      exit 1
    fi
    echo "[backup] creating PostgreSQL base backup to $output_file"
    pg_basebackup \
      --dbname "$DATABASE_URL" \
      --format=tar \
      --gzip \
      --wal-method=stream \
      --checkpoint=fast \
      --pgdata=- >"$output_file"
    ;;
  *)
    echo "ERROR: unsupported BACKUP_MODE=${BACKUP_MODE}" >&2
    exit 1
    ;;
esac

sha256sum "$output_file" > "$output_file.sha256"
echo "[backup] wrote checksum $output_file.sha256"
