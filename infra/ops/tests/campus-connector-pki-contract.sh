#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
GENERATOR="${REPO_ROOT}/infra/ops/generate-campus-connector-pki.sh"
PREPARE_RUNTIME="${REPO_ROOT}/infra/ops/prepare-campus-connector-gateway-runtime.sh"
VALIDATE_RUNTIME="${REPO_ROOT}/infra/ops/validate-campus-connector-gateway-runtime.sh"

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
runtime_dir="${tmp_dir}/gateway-runtime"
owner_uid="$(id -u)"
owner_gid="$(id -g)"
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

"${PREPARE_RUNTIME}" \
  --source "${output_dir}" \
  --output "${runtime_dir}" \
  --gateway-host connector.example.test \
  --owner-uid "${owner_uid}" \
  --owner-gid "${owner_gid}" >/dev/null
"${PREPARE_RUNTIME}" \
  --source "${output_dir}" \
  --output "${runtime_dir}" \
  --gateway-host connector.example.test \
  --owner-uid "${owner_uid}" \
  --owner-gid "${owner_gid}" >/dev/null

expected_runtime_files="$({
  printf '%s\n' client-ca.crt gateway-ca.crt gateway.crt gateway.key snapshot-x25519.key
} | sort)"
actual_runtime_files="$(find "${runtime_dir}" -mindepth 1 -maxdepth 1 -printf '%f\n' | sort)"
[[ "${actual_runtime_files}" == "${expected_runtime_files}" ]] ||
  fail "runtime bundle does not contain exactly the five approved files"
[[ "$(stat -c '%a' "${runtime_dir}")" == "700" ]] ||
  fail "runtime bundle directory mode is not 0700"
[[ "$(stat -c '%a' "${runtime_dir}/gateway.key")" == "600" ]] ||
  fail "runtime gateway private key mode is not 0600"
[[ "$(stat -c '%u:%g' "${runtime_dir}/gateway.key")" == "${owner_uid}:${owner_gid}" ]] ||
  fail "runtime gateway private key owner mismatch"
[[ ! -e "${runtime_dir}/authority" && ! -e "${runtime_dir}/node" ]] ||
  fail "offline authority or node material leaked into the runtime bundle"

snapshot_key_id="$(
  "${VALIDATE_RUNTIME}" \
    --dir "${runtime_dir}" \
    --gateway-host connector.example.test \
    --expected-uid "${owner_uid}" \
    --expected-gid "${owner_gid}" \
    --print-snapshot-key-id 2>/dev/null
)"
[[ "${snapshot_key_id}" == "$(jq -r '.snapshotKeyID' "${output_dir}/public-metadata.json")" ]] ||
  fail "runtime snapshot key ID does not match the full PKI metadata"

if "${VALIDATE_RUNTIME}" \
  --dir "${runtime_dir}" \
  --gateway-host another.example.test \
  --snapshot-key-id "${snapshot_key_id}" \
  --expected-uid "${owner_uid}" \
  --expected-gid "${owner_gid}" >/dev/null 2>&1; then
  fail "runtime validator accepted a gateway host mismatch"
fi

chmod 0644 "${runtime_dir}/gateway.key"
if "${VALIDATE_RUNTIME}" \
  --dir "${runtime_dir}" \
  --gateway-host connector.example.test \
  --snapshot-key-id "${snapshot_key_id}" \
  --expected-uid "${owner_uid}" \
  --expected-gid "${owner_gid}" >/dev/null 2>&1; then
  fail "runtime validator accepted a group/world-readable private key"
fi
chmod 0600 "${runtime_dir}/gateway.key"

mv -- "${runtime_dir}/snapshot-x25519.key" "${tmp_dir}/snapshot-x25519.key.valid"
openssl genpkey \
  -algorithm RSA \
  -pkeyopt rsa_keygen_bits:2048 \
  -out "${runtime_dir}/snapshot-x25519.key" >/dev/null 2>&1
chmod 0600 "${runtime_dir}/snapshot-x25519.key"
if "${VALIDATE_RUNTIME}" \
  --dir "${runtime_dir}" \
  --gateway-host connector.example.test \
  --expected-uid "${owner_uid}" \
  --expected-gid "${owner_gid}" >/dev/null 2>&1; then
  fail "runtime validator accepted a non-X25519 snapshot key"
fi
mv -- "${tmp_dir}/snapshot-x25519.key.valid" "${runtime_dir}/snapshot-x25519.key"

mv -- "${runtime_dir}/client-ca.crt" "${tmp_dir}/client-ca.crt.valid"
openssl req \
  -x509 \
  -newkey rsa:2048 \
  -nodes \
  -days 365 \
  -subj /CN=not-a-client-ca \
  -addext basicConstraints=critical,CA:FALSE \
  -addext keyUsage=critical,digitalSignature \
  -keyout "${tmp_dir}/not-a-client-ca.key" \
  -out "${runtime_dir}/client-ca.crt" >/dev/null 2>&1
chmod 0644 "${runtime_dir}/client-ca.crt"
if "${VALIDATE_RUNTIME}" \
  --dir "${runtime_dir}" \
  --gateway-host connector.example.test \
  --expected-uid "${owner_uid}" \
  --expected-gid "${owner_gid}" >/dev/null 2>&1; then
  fail "runtime validator accepted client-ca.crt with CA:FALSE"
fi

openssl req \
  -x509 \
  -newkey rsa:2048 \
  -nodes \
  -days 365 \
  -subj /CN=client-ca-without-key-cert-sign \
  -addext basicConstraints=critical,CA:TRUE \
  -addext keyUsage=critical,digitalSignature \
  -keyout "${tmp_dir}/client-ca-without-key-cert-sign.key" \
  -out "${runtime_dir}/client-ca.crt" >/dev/null 2>&1
chmod 0644 "${runtime_dir}/client-ca.crt"
if "${VALIDATE_RUNTIME}" \
  --dir "${runtime_dir}" \
  --gateway-host connector.example.test \
  --expected-uid "${owner_uid}" \
  --expected-gid "${owner_gid}" >/dev/null 2>&1; then
  fail "runtime validator accepted a client CA without certificate-signing key usage"
fi
mv -- "${tmp_dir}/client-ca.crt.valid" "${runtime_dir}/client-ca.crt"

touch "${runtime_dir}/unexpected.key"
if "${VALIDATE_RUNTIME}" \
  --dir "${runtime_dir}" \
  --gateway-host connector.example.test \
  --snapshot-key-id "${snapshot_key_id}" \
  --expected-uid "${owner_uid}" \
  --expected-gid "${owner_gid}" >/dev/null 2>&1; then
  fail "runtime validator accepted unexpected material"
fi
rm -f -- "${runtime_dir}/unexpected.key"

printf '[campus-connector-pki-contract] all assertions passed\n'
