#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
SERVICES_FILE="${REPO_ROOT}/infra/traefik/services.dynamic.yaml"
BACKEND_MODULES_FILE="${REPO_ROOT}/server/internal/app/modules.go"

fail() {
  echo "[traefik-services-contract][error] $*" >&2
  exit 1
}

assert_contains() {
  local file="$1"
  local pattern="$2"
  if ! grep -Eq "${pattern}" "${file}"; then
    fail "expected ${file} to contain pattern: ${pattern}"
  fi
}

assert_not_contains() {
  local file="$1"
  local pattern="$2"
  if grep -Eq "${pattern}" "${file}"; then
    fail "expected ${file} to not contain pattern: ${pattern}"
  fi
}

router_block="$(
  awk '
    /^[[:space:]]*backend-api:/ { in_block=1; next }
    /^[[:space:]]*admin-web:/ { in_block=0 }
    in_block { print }
  ' "${SERVICES_FILE}"
)"

assert_contains "${BACKEND_MODULES_FILE}" 'api := r\.Group\("/api/v1"\)'
[[ -n "${router_block}" ]] || fail "expected backend-api router block in ${SERVICES_FILE}"

if printf '%s\n' "${router_block}" | grep -Eq 'strip-api-prefix'; then
  fail "backend-api router must preserve /api prefix to match backend /api/v1 routes"
fi

assert_not_contains "${SERVICES_FILE}" '^[[:space:]]*strip-api-prefix:'

echo "[traefik-services-contract] all assertions passed"
