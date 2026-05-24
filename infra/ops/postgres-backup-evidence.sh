#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

usage() {
  cat <<'USAGE'
Usage: infra/ops/postgres-backup-evidence.sh

Verifies that a logical PostgreSQL backup exists locally, can be fetched back
from object storage, and has a matching SHA256 sidecar. The script writes a
sanitized JSON evidence bundle.

Required env is inherited from fetch-postgres-backups.sh.

Optional env:
  POSTGRES_BACKUP_EVIDENCE_FILE
      Output path. Defaults to infra/generated/postgres-backup-evidence.json.
      Set to "-" to only print the JSON bundle.
  POSTGRES_BACKUP_EVIDENCE_FETCH_COMMAND
      Test/custom override for the fetch command.
  POSTGRES_BACKUP_EVIDENCE_TIMER_REQUIRED
      true to fail when systemd timers cannot be checked. Defaults to false.
  POSTGRES_BACKUP_EVIDENCE_SKIP_TIMERS
      true to skip timer checks.
USAGE
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
  exit 0
fi

require_cmd python3
require_cmd sha256sum

load_env

evidence_file="${POSTGRES_BACKUP_EVIDENCE_FILE:-${REPO_ROOT}/infra/generated/postgres-backup-evidence.json}"
fetch_command="${POSTGRES_BACKUP_EVIDENCE_FETCH_COMMAND:-${SCRIPT_DIR}/fetch-postgres-backups.sh}"
logical_dir="${BACKUP_LOGICAL_DIR:-${REPO_ROOT}/backups/postgres/logical}"
timer_required="${POSTGRES_BACKUP_EVIDENCE_TIMER_REQUIRED:-false}"
skip_timers="${POSTGRES_BACKUP_EVIDENCE_SKIP_TIMERS:-false}"

resolve_command() {
  local command_path="$1"
  if [[ "${command_path}" != */* ]]; then
    command -v "${command_path}" || return 1
    return 0
  fi
  if [[ "${command_path}" != /* ]]; then
    command_path="${REPO_ROOT}/${command_path}"
  fi
  [[ -x "${command_path}" ]] || return 1
  printf '%s\n' "${command_path}"
}

latest_file() {
  local dir="$1"
  local pattern="$2"
  [[ -d "${dir}" ]] || return 1
  find "${dir}" -maxdepth 1 -type f -name "${pattern}" -printf '%T@ %p\n' 2>/dev/null | sort -rn | head -n1 | cut -d' ' -f2-
}

sha256_value() {
  local file="$1"
  sha256sum "${file}" | awk '{print $1}'
}

expected_sha256_value() {
  local sidecar="$1"
  awk '{print $1}' "${sidecar}" | head -n1
}

verify_sha256_sidecar() {
  local file="$1"
  local sidecar="${file}.sha256"
  [[ -s "${file}" ]] || die "backup file is missing or empty: ${file}"
  [[ -s "${sidecar}" ]] || die "backup sha256 sidecar is missing or empty: ${sidecar}"
  local actual expected
  actual="$(sha256_value "${file}")"
  expected="$(expected_sha256_value "${sidecar}")"
  [[ "${actual}" == "${expected}" ]] || die "backup sha256 mismatch for ${file}"
}

timer_units=(
  stuhelper-postgres-dump-backup.timer
  stuhelper-postgres-basebackup.timer
  stuhelper-postgres-backup-sync.timer
)

timer_json="[]"
timer_checked="false"
check_timers() {
  if [[ "${skip_timers}" == "true" ]]; then
    timer_checked="false"
    timer_json="[]"
    return
  fi
  if ! command -v systemctl >/dev/null 2>&1; then
    [[ "${timer_required}" != "true" ]] || die "systemctl is required to verify PostgreSQL backup timers"
    timer_checked="false"
    timer_json="[]"
    return
  fi

  local tmp_json
  tmp_json="$(mktemp)"
  printf '[' >"${tmp_json}"
  local first="true"
  local unit installed enabled
  for unit in "${timer_units[@]}"; do
    installed="false"
    enabled="false"
    if systemctl list-unit-files | grep -q "^${unit}"; then
      installed="true"
    fi
    if systemctl is-enabled --quiet "${unit}"; then
      enabled="true"
    fi
    if [[ "${timer_required}" == "true" && ( "${installed}" != "true" || "${enabled}" != "true" ) ]]; then
      rm -f "${tmp_json}"
      die "backup timer ${unit} is not installed and enabled"
    fi
    if [[ "${first}" == "true" ]]; then
      first="false"
    else
      printf ',' >>"${tmp_json}"
    fi
    python3 - "${unit}" "${installed}" "${enabled}" >>"${tmp_json}" <<'PY'
import json
import sys
print(json.dumps({
    "unit": sys.argv[1],
    "installed": sys.argv[2] == "true",
    "enabled": sys.argv[3] == "true",
}, separators=(",", ":")), end="")
PY
  done
  printf ']' >>"${tmp_json}"
  timer_checked="true"
  timer_json="$(cat "${tmp_json}")"
  rm -f "${tmp_json}"
}

backup_summary_json() {
  local file="$1"
  local sha
  sha="$(sha256_value "${file}")"
  python3 - "${file}" "${sha}" <<'PY'
import json
import os
import sys
from pathlib import Path

path = Path(sys.argv[1])
print(json.dumps({
    "file": path.name,
    "sizeBytes": path.stat().st_size,
    "sha256": sys.argv[2],
    "sha256Verified": True,
}, separators=(",", ":")))
PY
}

check_timers

local_latest="$(latest_file "${logical_dir}" '*.dump')"
[[ -n "${local_latest}" ]] || die "no local logical PostgreSQL backup was found in ${logical_dir}"
verify_sha256_sidecar "${local_latest}"
local_backup_json="$(backup_summary_json "${local_latest}")"

fetch_command_path="$(resolve_command "${fetch_command}")" || die "fetch command was not found or is not executable: ${fetch_command}"
fetch_root="$(mktemp -d)"
trap 'rm -rf "${fetch_root}"' EXIT
fetched_logical_dir="${fetch_root}/logical"
mkdir -p "${fetched_logical_dir}" "${fetch_root}/base" "${fetch_root}/wal"

log "fetching logical PostgreSQL backup artifacts from object storage" >&2
BACKUP_LOGICAL_DIR="${fetched_logical_dir}" \
BACKUP_BASE_DIR="${fetch_root}/base" \
POSTGRES_WAL_RESTORE_DIR="${fetch_root}/wal" \
  "${fetch_command_path}" logical

fetched_latest="$(latest_file "${fetched_logical_dir}" '*.dump')"
[[ -n "${fetched_latest}" ]] || die "no fetched logical PostgreSQL backup was found in ${fetched_logical_dir}"
verify_sha256_sidecar "${fetched_latest}"
fetched_backup_json="$(backup_summary_json "${fetched_latest}")"

generated_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
bundle="$(
  TIMER_JSON="${timer_json}" \
  LOCAL_BACKUP_JSON="${local_backup_json}" \
  FETCHED_BACKUP_JSON="${fetched_backup_json}" \
  python3 - "${generated_at}" "${APP_ENV:-}" "${timer_checked}" <<'PY'
import json
import os
import sys

bundle = {
    "generatedAt": sys.argv[1],
    "appEnv": sys.argv[2],
    "timers": {
        "checked": sys.argv[3] == "true",
        "units": json.loads(os.environ["TIMER_JSON"]),
    },
    "localLogicalBackup": json.loads(os.environ["LOCAL_BACKUP_JSON"]),
    "fetchedLogicalBackup": json.loads(os.environ["FETCHED_BACKUP_JSON"]),
}
print(json.dumps(bundle, ensure_ascii=True, indent=2))
PY
)"

if [[ "${evidence_file}" != "-" ]]; then
  mkdir -p "$(dirname "${evidence_file}")"
  tmp_file="$(mktemp)"
  trap 'rm -rf "${fetch_root}" "${tmp_file}"' EXIT
  printf '%s\n' "${bundle}" >"${tmp_file}"
  install -m 600 "${tmp_file}" "${evidence_file}"
  log "wrote PostgreSQL backup evidence to ${evidence_file}" >&2
fi

printf '%s\n' "${bundle}" | python3 -m json.tool
