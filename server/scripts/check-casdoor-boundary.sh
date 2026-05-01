#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
SERVER_DIR="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
SCAN_DIR="${SERVER_DIR}/internal"

if [[ ! -d "${SCAN_DIR}" ]]; then
  echo "ERROR: scan dir ${SCAN_DIR} not found" >&2
  exit 1
fi

run_check() {
  local label="$1" pattern="$2" allowed="$3"
  local hits="" file="" rel="" rc=0

  while IFS= read -r -d '' file; do
    if grep -Eq "${pattern}" "${file}"; then
      rel="${file#${SERVER_DIR}/}"
      hits+="${rel}"$'\n'
      continue
    fi
    rc=$?
    if (( rc > 1 )); then
      echo "ERROR: grep failed (rc=${rc}) when scanning ${file}" >&2
      exit 1
    fi
  done < <(find "${SCAN_DIR}" -type f -name '*.go' -print0)

  report_violations "${label}" "${hits}" "${allowed}"
}

report_violations() {
  local label="$1" hits="$2" allowed="$3"
  local violations=""

  if [[ -n "${hits}" ]]; then
    violations="$(printf '%s' "${hits}" | sort -u | grep -vE "${allowed}" || true)"
  fi
  if [[ -n "${violations}" ]]; then
    echo "ERROR: ${label}" >&2
    printf '%s\n' "${violations}" >&2
    exit 1
  fi
}

run_check \
  "business code must not import or call Casdoor SDK directly" \
  'github\.com/casdoor/casdoor-go-sdk/casdoorsdk|casdoorsdk\.' \
  '^internal/platform/casdoor/'

run_check \
  "Casdoor Casbin decision APIs must not be used" \
  '\.(Enforce|BatchEnforce|GetPermissions)\(' \
  '^$'

run_check \
  "business code must not import OpenFGA SDK directly" \
  'github\.com/openfga/go-sdk' \
  '^internal/(pkg/fga|platform/authorization)/'

run_check \
  "backend internal Go code must not reference retired Zitadel identifiers" \
  'Zitadel|ZITADEL|zitadel|urn:zitadel' \
  '^$'
