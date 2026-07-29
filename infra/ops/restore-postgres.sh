#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

if [[ $# -lt 1 ]]; then
  echo "Usage: ALLOW_DESTRUCTIVE=1 $0 <backup-file>" >&2
  exit 1
fi

if [[ "${ALLOW_DESTRUCTIVE:-0}" != "1" ]]; then
  echo "ERROR: set ALLOW_DESTRUCTIVE=1 to acknowledge restore is destructive" >&2
  exit 1
fi

load_env_preserving DATABASE_URL
require_cmd docker
[[ -n "${DATABASE_URL:-}" ]] || die "DATABASE_URL is required"

backup_file="$1"
if [[ ! -f "$backup_file" ]]; then
  echo "ERROR: backup file not found: $backup_file" >&2
  exit 1
fi

if [[ -f "$backup_file.sha256" ]]; then
  sha256sum -c "$backup_file.sha256"
fi

echo "[restore] restoring PostgreSQL from $backup_file"
compose run --rm --no-deps -T \
  postgres-client \
  pg_restore \
    --clean \
    --if-exists \
    --no-owner \
    --no-privileges \
    --dbname "${DATABASE_URL}" \
  <"${backup_file}"
echo "[restore] restore completed"
