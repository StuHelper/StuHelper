#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
SERVER_DIR="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
REPO_DIR="$(cd -- "${SERVER_DIR}/.." && pwd)"
SCAN_DIR="${SERVER_DIR}/internal"

if [[ ! -d "${SCAN_DIR}" ]]; then
  echo "ERROR: scan dir ${SCAN_DIR} not found" >&2
  exit 1
fi

run_check() {
  local label="$1" pattern="$2" allowed="$3"
  run_check_with_find "${label}" "${pattern}" "${allowed}" "${SCAN_DIR}" -type f -name '*.go'
}

run_check_non_test() {
  local label="$1" pattern="$2" allowed="$3" scan_root="$4"
  run_check_with_find "${label}" "${pattern}" "${allowed}" "${scan_root}" -type f -name '*.go' '!' -name '*_test.go'
}

run_check_with_find() {
  local label="$1" pattern="$2" allowed="$3" scan_root="$4"
  shift 4
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
  done < <(find "${scan_root}" "$@" -print0)

  report_violations "${label}" "${hits}" "${allowed}"
}

run_git_check() {
  local label="$1" pattern="$2" allowed="$3"
  shift 3
  local hits="" rc=0

  set +e
  hits="$(
    cd "${REPO_DIR}" &&
      git -c safe.directory="${REPO_DIR}" grep -lE "${pattern}" -- "$@" 2>/dev/null
  )"
  rc=$?
  set -e

  if (( rc == 1 )); then
    return
  fi
  if (( rc > 1 )); then
    echo "ERROR: git grep failed (rc=${rc}) for ${label}" >&2
    exit 1
  fi

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

run_check_non_test \
  "business modules must not use Casdoor subject as a business identity" \
  'casdoor_subject|CasdoorSubject|casdoorSubject' \
  '^internal/modules/(auth|user)/' \
  "${SERVER_DIR}/internal/modules"

run_check_non_test \
  "backend code must not use retired moderator role literal" \
  '"moderator"|roleModerator' \
  '^$' \
  "${SERVER_DIR}/internal"

run_git_check \
  "tracked server/infra/env files must not reference retired Zitadel identifiers" \
  'Zitadel|ZITADEL|zitadel|urn:zitadel' \
  '^server/scripts/check-casdoor-boundary\.sh$' \
  server infra .env.example .env.prod.example docker-compose.yml docker-compose.prod.yml Makefile .gitlab-ci.yml
