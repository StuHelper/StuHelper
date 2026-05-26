#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
SMOKE_CHECK="${REPO_ROOT}/infra/ops/smoke-check.sh"

fail() {
  echo "[smoke-check-contract][error] $*" >&2
  exit 1
}

assert_contains() {
  local file="$1"
  local pattern="$2"
  if ! grep -Eq -- "${pattern}" "${file}"; then
    fail "expected ${file} to contain pattern: ${pattern}"
  fi
}

[[ -x "${SMOKE_CHECK}" ]] || fail "missing executable file: ${SMOKE_CHECK}"

bash -n "${SMOKE_CHECK}"

assert_contains "${SMOKE_CHECK}" 'check_body_regex\(\)'
assert_contains "${SMOKE_CHECK}" 'grep -Eq "\$pattern"'
assert_contains "${SMOKE_CHECK}" 'check_body_regex "Grafana 健康" "\$\{GRAFANA_URL\}/api/health"'
assert_contains "${SMOKE_CHECK}" '"database"\[\[:space:\]\]\*:\[\[:space:\]\]\*"ok"'

echo "[smoke-check-contract] all assertions passed"
