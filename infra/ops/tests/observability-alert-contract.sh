#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
APPLICATION_RULES="${REPO_ROOT}/infra/observability/prometheus/rules/application.yml"

fail() {
  echo "[observability-alert-contract][error] $*" >&2
  exit 1
}

assert_contains() {
  local pattern="$1"
  if ! grep -Fq -- "${pattern}" "${APPLICATION_RULES}"; then
    fail "expected application.yml to contain: ${pattern}"
  fi
}

assert_contains "StuHelperOutboxTerminalFailures"
assert_contains 'increase(outbox_job_failures_total{terminal="true"}[5m]) > 0'
assert_contains "Outbox job reached terminal retry threshold"
assert_contains "StuHelperRefreshTokenReuseDetected"
assert_contains "increase(auth_refresh_token_reuse_total[5m]) > 0"
assert_contains "Refresh token reuse detected"
assert_contains "StuHelperIAMDriftReconciliationThresholdExceeded"
assert_contains "increase(iam_drift_reconciliation_threshold_exceeded_total[10m]) > 0"
assert_contains "IAM drift reconciliation exceeded automatic repair threshold"

echo "[observability-alert-contract] all assertions passed"
