#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

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

output_file="$1"
mkdir -p "$(dirname "$output_file")"

run_dump() {
  if command -v pg_dump >/dev/null 2>&1; then
    pg_dump --format=custom --no-owner --no-privileges --file "$output_file" "$logical_url"
    return 0
  fi

  require_cmd docker
  load_env
  compose exec -T -e BACKUP_DATABASE_URL="${logical_url}" postgres \
    sh -lc 'pg_dump --format=custom --no-owner --no-privileges "$BACKUP_DATABASE_URL"' >"$output_file"
}

run_basebackup() {
  if command -v pg_basebackup >/dev/null 2>&1; then
    pg_basebackup \
      --dbname "$replication_url" \
      --format=tar \
      --gzip \
      --wal-method=stream \
      --checkpoint=fast \
      --pgdata=- >"$output_file"
    return 0
  fi

  require_cmd docker
  load_env
  compose exec -T -e REPLICATION_DATABASE_URL="${replication_url}" postgres \
    sh -lc 'pg_basebackup --dbname "$REPLICATION_DATABASE_URL" --format=tar --gzip --wal-method=stream --checkpoint=fast --pgdata=-' >"$output_file"
}

case "${mode}" in
  dump)
    echo "[backup] dumping PostgreSQL to $output_file"
    run_dump
    ;;
  basebackup)
    echo "[backup] creating PostgreSQL base backup to $output_file"
    run_basebackup
    ;;
  *)
    echo "ERROR: unsupported BACKUP_MODE=${mode}" >&2
    exit 1
    ;;
esac

sha256sum "$output_file" > "$output_file.sha256"
echo "[backup] wrote checksum $output_file.sha256"
