#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
PROBE_SCRIPT="${REPO_ROOT}/infra/ops/casdoor-token-minimization-probe.sh"

fail() {
  echo "[casdoor-token-minimization-probe-contract][error] $*" >&2
  exit 1
}

assert_contains() {
  local pattern="$1"
  if ! grep -Fq -- "${pattern}" "${PROBE_SCRIPT}"; then
    fail "expected probe script to contain: ${pattern}"
  fi
}

[[ -x "${PROBE_SCRIPT}" ]] || fail "probe script must be executable"
assert_contains "CASDOOR_TOKEN_PROBE_AUTH_CODE"
assert_contains "CASDOOR_TOKEN_PROBE_OUTPUT"
assert_contains "json emits probe evidence on stdout"
assert_contains "scope=openid"
assert_contains "code_challenge_method"
assert_contains "authorization_code"
assert_contains "phone_verified"
assert_contains "stuhelper_student_verified"
assert_contains '"businessClaims"'
assert_contains '"tokenClaims"'
assert_contains "sort_keys=True"
assert_contains "token minimization verdict: passed"
assert_contains "exit 78"

retired_idp_pattern="$(printf '\x5a\x49\x54\x41\x44\x45\x4c')"
if grep -Eq "${retired_idp_pattern}" "${PROBE_SCRIPT}"; then
  fail "probe script must not reference retired IDP identifiers"
fi

echo "[casdoor-token-minimization-probe-contract] all assertions passed"
