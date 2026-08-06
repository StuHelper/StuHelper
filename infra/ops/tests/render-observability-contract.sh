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
    ALERTMANAGER_WEBHOOK_TOKEN="contract-alertmanager-token-0123456789abcdef" \
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

alertmanager_config="${tmpdir}/prod/alertmanager/alertmanager.yml"
alertmanager_token_file="${tmpdir}/prod/alertmanager/webhook-token"
grep -Fq -- 'credentials_file: /etc/alertmanager/secrets/webhook-token' "${alertmanager_config}" ||
  fail "production Alertmanager rendering lost credentials_file authorization"
if grep -Fq -- 'contract-alertmanager-token-0123456789abcdef' "${alertmanager_config}"; then
  fail "production Alertmanager config leaked the webhook token"
fi
[[ -f "${alertmanager_token_file}" ]] || fail "production Alertmanager token file was not rendered"
[[ "$(stat -c '%a' "${alertmanager_token_file}")" == "640" ]] ||
  fail "production Alertmanager token file must be mode 640 before deployment normalization"
[[ "$(wc -c <"${alertmanager_token_file}")" -ge 32 ]] ||
  fail "production Alertmanager token file is unexpectedly short"

if METRICS_PASSWORD="contract-only-metrics-password" \
  ALERTMANAGER_WEBHOOK_URL="https://alerts.example.test/stuhelper" \
  ALERTMANAGER_WEBHOOK_TOKEN="" \
  GENERATED_OBS_DIR="${tmpdir}/missing-token" \
  python3 "${RENDERER}" --mode prod >/dev/null 2>&1; then
  fail "production rendering accepted an empty Alertmanager webhook token"
fi

printf '[render-observability-contract] all assertions passed\n'
