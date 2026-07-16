#!/usr/bin/env bash
set -euo pipefail

jobs_file="${BAOTA_JOBS_FILE:-/www/server/panel/class/jobs.py}"
marker="${BAOTA_FILES_SET_MODE_MARKER:-/tmp/last_files_set_mode.pl}"

die() {
  echo "[guard-baota-files-set-mode][error] $*" >&2
  exit 1
}

allow_non_root_for_tests="${BAOTA_GUARD_ALLOW_NON_ROOT_FOR_TESTS:-false}"
if [[ "$(id -u)" != "0" && "${allow_non_root_for_tests}" != "true" ]]; then
  die "run as root"
fi
[[ -f "${jobs_file}" ]] || die "Baota jobs file not found: ${jobs_file}"

# Refuse to claim protection after a panel update changes the marker contract.
grep -Fq "${marker}" "${jobs_file}" || die "Baota jobs file no longer references marker: ${marker}"
grep -Eq 'time\.time\(\)[[:space:]]*-[[:space:]]*os\.path\.getmtime\(tips_file\)[[:space:]]*<[[:space:]]*86400' "${jobs_file}" \
  || die "Baota jobs file no longer honors the 24-hour files_set_mode guard"

mkdir -p "$(dirname "${marker}")"
touch "${marker}"
if [[ "$(id -u)" == "0" ]]; then
  chown root:root "${marker}"
fi
chmod 600 "${marker}"

echo "[guard-baota-files-set-mode] refreshed marker: ${marker}"
