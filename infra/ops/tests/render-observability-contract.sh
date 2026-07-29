#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
RENDERER="${REPO_ROOT}/infra/ops/render-observability-configs.py"
SSO_TARGET="https://sso.stuhelper.com/.well-known/openid-configuration"

fail() {
  printf '[render-observability-contract][error] %s\n' "$*" >&2
  exit 1
}

cleanup() {
  if [[ -n "${tmpdir:-}" && -d "${tmpdir}" ]]; then
    rm -rf "${tmpdir}"
  fi
}
trap cleanup EXIT

tmpdir="$(mktemp -d)"

render_mode() {
  local mode="$1"
  local output_dir="${tmpdir}/${mode}"
  METRICS_PASSWORD="contract-only-metrics-password" \
    ALERTMANAGER_WEBHOOK_URL="https://alerts.example.test/stuhelper" \
    GENERATED_OBS_DIR="${output_dir}" \
    python3 "${RENDERER}" --mode "${mode}" >/dev/null
  printf '%s\n' "${output_dir}/prometheus/prometheus.yml"
}

for local_mode in dev observability; do
  local_output="$(render_mode "${local_mode}")"
  if grep -Fq -- "${SSO_TARGET}" "${local_output}"; then
    fail "${local_mode} rendering must not enable the public SSO probe"
  fi
  grep -Fq -- "http://app:8080/health/live" "${local_output}" ||
    fail "${local_mode} rendering lost internal application probes"
done

prod_output="$(render_mode prod)"
grep -Fq -- "${SSO_TARGET}" "${prod_output}" ||
  fail "production rendering must retain the public SSO availability probe"

if grep -Fq -- "__BLACKBOX_PRODUCTION_HTTP_TARGETS__" "${prod_output}"; then
  fail "production rendering left an unresolved template placeholder"
fi

printf '[render-observability-contract] all assertions passed\n'
