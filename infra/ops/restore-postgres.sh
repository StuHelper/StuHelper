#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "Usage: ALLOW_DESTRUCTIVE=1 $0 <backup-file>" >&2
  exit 1
fi

if [[ "${ALLOW_DESTRUCTIVE:-0}" != "1" ]]; then
  echo "ERROR: set ALLOW_DESTRUCTIVE=1 to acknowledge restore is destructive" >&2
  exit 1
fi

if [[ -z "${DATABASE_URL:-}" ]]; then
  echo "ERROR: DATABASE_URL is required" >&2
  exit 1
fi

if ! command -v pg_restore >/dev/null 2>&1; then
  echo "ERROR: pg_restore is required" >&2
  exit 1
fi

backup_file="$1"
if [[ ! -f "$backup_file" ]]; then
  echo "ERROR: backup file not found: $backup_file" >&2
  exit 1
fi

if [[ -f "$backup_file.sha256" ]]; then
  sha256sum -c "$backup_file.sha256"
fi

echo "[restore] restoring PostgreSQL from $backup_file"
pg_restore --clean --if-exists --no-owner --no-privileges --dbname "$DATABASE_URL" "$backup_file"
echo "[restore] restore completed"
