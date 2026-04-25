#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
BACKUP_FILE="${REPO_ROOT}/infra/ops/backup-postgres.sh"

fail() {
  echo "[backup-postgres-contract][error] $*" >&2
  exit 1
}

line_number() {
  local pattern="$1"
  local line
  line="$(grep -nF -- "${pattern}" "${BACKUP_FILE}" | head -n1 | cut -d: -f1)"
  [[ -n "${line}" ]] || fail "expected pattern in ${BACKUP_FILE}: ${pattern}"
  printf '%s\n' "${line}"
}

load_env_line="$(line_number 'load_env')"
logical_url_line="$(line_number 'logical_url="${BACKUP_DATABASE_URL:-}"')"
replication_url_line="$(line_number 'replication_url="${REPLICATION_DATABASE_URL:-}"')"

if (( load_env_line >= logical_url_line )); then
  fail "backup-postgres.sh must load env before reading BACKUP_DATABASE_URL"
fi

if (( load_env_line >= replication_url_line )); then
  fail "backup-postgres.sh must load env before reading REPLICATION_DATABASE_URL"
fi

echo "[backup-postgres-contract] all assertions passed"
