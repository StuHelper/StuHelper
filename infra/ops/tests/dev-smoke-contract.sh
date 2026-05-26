#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
DEV_SMOKE="${REPO_ROOT}/infra/ops/dev-smoke.sh"

fail() {
  echo "[dev-smoke-contract][error] $*" >&2
  exit 1
}

assert_contains() {
  local file="$1"
  local pattern="$2"
  if ! grep -Eq -- "${pattern}" "${file}"; then
    fail "expected ${file} to contain pattern: ${pattern}"
  fi
}

line_number() {
  local pattern="$1"
  local line
  line="$(grep -En -- "${pattern}" "${DEV_SMOKE}" | head -n 1 | cut -d: -f1 || true)"
  [[ -n "${line}" ]] || fail "missing pattern in ${DEV_SMOKE}: ${pattern}"
  printf '%s\n' "${line}"
}

[[ -x "${DEV_SMOKE}" ]] || fail "missing executable file: ${DEV_SMOKE}"

bash -n "${DEV_SMOKE}"

assert_contains "${DEV_SMOKE}" 'source "\$\{SCRIPT_DIR\}/lib/common\.sh"'
assert_contains "${DEV_SMOKE}" 'source "\$\{SCRIPT_DIR\}/lib/dev-local\.sh"'
assert_contains "${DEV_SMOKE}" '"\$\{SCRIPT_DIR\}/init-dev-env\.sh"'
assert_contains "${DEV_SMOKE}" 'load_env'
assert_contains "${DEV_SMOKE}" 'source "\$\{DEV_RUNTIME_ENV\}"'
assert_contains "${DEV_SMOKE}" 'if \[\[ -z "\$\{GRAFANA_URL:-\}" \]\]; then'
assert_contains "${DEV_SMOKE}" 'dev_grafana_url="http://127\.0\.0\.1:\$\{GRAFANA_PORT:-3003\}"'
assert_contains "${DEV_SMOKE}" 'curl --fail --silent --show-error --max-time 2 "\$\{dev_grafana_url\}/api/health"'
assert_contains "${DEV_SMOKE}" 'export GRAFANA_URL="\$\{dev_grafana_url\}"'
assert_contains "${DEV_SMOKE}" '"\$\{SCRIPT_DIR\}/smoke-check\.sh"'
assert_contains "${DEV_SMOKE}" 'if \[\[ "\$\{DEV_BROWSER_SMOKE:-true\}" == "true" \]\]; then'
assert_contains "${DEV_SMOKE}" 'node "\$\{SCRIPT_DIR\}/dev-browser-smoke\.mjs"'

probe_line="$(line_number 'curl --fail --silent --show-error --max-time 2 "\$\{dev_grafana_url\}/api/health"')"
export_line="$(line_number 'export GRAFANA_URL="\$\{dev_grafana_url\}"')"
smoke_line="$(line_number '"\$\{SCRIPT_DIR\}/smoke-check\.sh"')"
browser_smoke_line="$(line_number 'node "\$\{SCRIPT_DIR\}/dev-browser-smoke\.mjs"')"

if (( probe_line >= export_line )); then
  fail "Grafana URL must only be exported after a successful health probe"
fi

if (( export_line >= smoke_line )); then
  fail "Grafana URL detection must run before smoke-check"
fi

if (( smoke_line >= browser_smoke_line )); then
  fail "browser smoke must run after HTTP smoke-check"
fi

echo "[dev-smoke-contract] all assertions passed"
