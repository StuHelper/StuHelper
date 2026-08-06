#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
GENERATOR="${REPO_ROOT}/infra/ops/generate-campus-connector-pki.sh"

fail() {
  printf '[campus-connector-pki-contract][error] %s\n' "$*" >&2
  exit 1
}

tmp_dir="$(mktemp -d)"
cleanup() {
  rm -rf -- "${tmp_dir}"
}
trap cleanup EXIT

node_id="10000000-0000-4000-8000-000000000001"
output_dir="${tmp_dir}/pki"
"${GENERATOR}" \
  --output "${output_dir}" \
  --gateway-host connector.example.test \
  --node-id "${node_id}" >/dev/null
"${GENERATOR}" \
  --check \
  --output "${output_dir}" \
  --gateway-host connector.example.test \
  --node-id "${node_id}" >/dev/null

[[ "$(jq -r '.nodeID' "${output_dir}/public-metadata.json")" == "${node_id}" ]] ||
  fail "metadata node ID mismatch"
[[ "$(jq -r '.gatewayHost' "${output_dir}/public-metadata.json")" == "connector.example.test" ]] ||
  fail "metadata gateway host mismatch"
[[ "$(stat -c '%a' "${output_dir}/gateway/gateway.key")" == "600" ]] ||
  fail "gateway private key mode is not 0600"
[[ "$(stat -c '%a' "${output_dir}/node/campus-connector-signing.key")" == "600" ]] ||
  fail "signing private key mode is not 0600"

if "${GENERATOR}" \
  --check \
  --output "${output_dir}" \
  --gateway-host another.example.test \
  --node-id "${node_id}" >/dev/null 2>&1; then
  fail "host mismatch was accepted"
fi

printf '[campus-connector-pki-contract] all assertions passed\n'
