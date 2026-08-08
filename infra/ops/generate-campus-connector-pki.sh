#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

output_dir="${CAMPUS_CONNECTOR_PKI_DIR:-${REPO_ROOT}/infra/generated/campus-connector-pki}"
gateway_host="${CAMPUS_CONNECTOR_GATEWAY_PUBLIC_HOST:-connector.stuhelper.com}"
node_id="${CAMPUS_CONNECTOR_NODE_ID:-}"
check_only=false

usage() {
  cat <<'EOF'
Usage: infra/ops/generate-campus-connector-pki.sh [options]

Generate the independent key hierarchy used by the StuHelper campus connector.
Existing material is validated and never overwritten.

Options:
  --output DIR          PKI root (default: infra/generated/campus-connector-pki)
  --gateway-host HOST   exact DNS name or IP in the gateway server certificate
  --node-id UUID        stable connector node UUID (generated when omitted)
  --check               validate existing material without generating anything
  -h, --help            show this help
EOF
}

die() {
  printf '[campus-connector-pki][error] %s\n' "$*" >&2
  exit 1
}

log() {
  printf '[campus-connector-pki] %s\n' "$*"
}

while (($# > 0)); do
  case "$1" in
    --output)
      (($# >= 2)) || die "--output requires a value"
      output_dir="$2"
      shift 2
      ;;
    --gateway-host)
      (($# >= 2)) || die "--gateway-host requires a value"
      gateway_host="$2"
      shift 2
      ;;
    --node-id)
      (($# >= 2)) || die "--node-id requires a value"
      node_id="$2"
      shift 2
      ;;
    --check)
      check_only=true
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      die "unknown option: $1"
      ;;
  esac
done

for command in openssl jq stat awk; do
  command -v "${command}" >/dev/null 2>&1 || die "required command is missing: ${command}"
done

[[ -n "${gateway_host}" && "${gateway_host}" != *://* && "${gateway_host}" != */* &&
   "${gateway_host}" != *:* && "${gateway_host}" != *' '* ]] ||
  die "gateway host must be one exact DNS name or IPv4 address without scheme, port, path, or wildcard"
if [[ -n "${node_id}" ]]; then
  [[ "${node_id}" =~ ^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$ ]] ||
    die "node ID must be a lowercase UUID"
fi

output_dir="$(readlink -m -- "${output_dir}")"

public_key_digest() {
  local public_key_file="$1"
  openssl pkey -pubin -in "${public_key_file}" -outform DER 2>/dev/null |
    openssl dgst -sha256 2>/dev/null |
    awk '{print $2}'
}

certificate_public_key_digest() {
  local certificate_file="$1"
  openssl x509 -in "${certificate_file}" -pubkey -noout 2>/dev/null |
    openssl pkey -pubin -outform DER 2>/dev/null |
    openssl dgst -sha256 2>/dev/null |
    awk '{print $2}'
}

private_key_public_digest() {
  local private_key_file="$1"
  openssl pkey -in "${private_key_file}" -pubout -outform DER 2>/dev/null |
    openssl dgst -sha256 2>/dev/null |
    awk '{print $2}'
}

require_private_mode() {
  local path="$1"
  local mode
  mode="$(stat -c '%a' -- "${path}")"
  case "${mode}" in
    400|600) ;;
    *) die "private key must have mode 0400 or 0600: ${path} (got ${mode})" ;;
  esac
}

validate_existing() {
  local root="$1"
  local metadata="${root}/public-metadata.json"
  local required=(
    "${metadata}"
    "${root}/authority/gateway-ca.crt"
    "${root}/authority/gateway-ca.key"
    "${root}/authority/client-ca.crt"
    "${root}/authority/client-ca.key"
    "${root}/gateway/gateway.crt"
    "${root}/gateway/gateway.key"
    "${root}/gateway/client-ca.crt"
    "${root}/gateway/snapshot-x25519.key"
    "${root}/node/campus-connector-client.crt"
    "${root}/node/campus-connector-client.key"
    "${root}/node/campus-connector-central-ca.crt"
    "${root}/node/campus-connector-signing.key"
    "${root}/node/campus-connector-snapshot-x25519.pub"
    "${root}/registry/client.crt"
    "${root}/registry/signing.pub"
  )
  local path
  for path in "${required[@]}"; do
    [[ -s "${path}" ]] || die "missing or empty PKI file: ${path}"
  done

  local recorded_host recorded_node recorded_signing_id recorded_snapshot_id
  recorded_host="$(jq -er '.gatewayHost' "${metadata}")" || die "invalid PKI metadata gatewayHost"
  recorded_node="$(jq -er '.nodeID' "${metadata}")" || die "invalid PKI metadata nodeID"
  recorded_signing_id="$(jq -er '.signingKeyID' "${metadata}")" || die "invalid PKI metadata signingKeyID"
  recorded_snapshot_id="$(jq -er '.snapshotKeyID' "${metadata}")" || die "invalid PKI metadata snapshotKeyID"
  [[ "${recorded_node}" =~ ^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$ ]] ||
    die "PKI metadata contains an invalid node ID"
  if [[ -n "${gateway_host}" && "${recorded_host}" != "${gateway_host}" ]]; then
    die "existing gateway certificate host is ${recorded_host}, not ${gateway_host}; rotate explicitly instead of overwriting"
  fi
  if [[ -n "${node_id}" && "${recorded_node}" != "${node_id}" ]]; then
    die "existing node ID is ${recorded_node}, not ${node_id}; rotate explicitly instead of overwriting"
  fi

  openssl verify -purpose sslserver \
    -CAfile "${root}/authority/gateway-ca.crt" \
    "${root}/gateway/gateway.crt" >/dev/null || die "gateway certificate chain validation failed"
  openssl verify -purpose sslclient \
    -CAfile "${root}/gateway/client-ca.crt" \
    "${root}/registry/client.crt" >/dev/null || die "client certificate chain validation failed"
  openssl x509 -in "${root}/gateway/gateway.crt" -noout -checkhost "${recorded_host}" >/dev/null ||
    die "gateway certificate does not cover ${recorded_host}"
  openssl x509 -in "${root}/gateway/gateway.crt" -noout -checkend 2592000 >/dev/null ||
    die "gateway certificate expires in less than 30 days"
  openssl x509 -in "${root}/registry/client.crt" -noout -checkend 2592000 >/dev/null ||
    die "client certificate expires in less than 30 days"

  [[ "$(certificate_public_key_digest "${root}/gateway/gateway.crt")" == \
     "$(private_key_public_digest "${root}/gateway/gateway.key")" ]] ||
    die "gateway certificate and private key do not match"
  [[ "$(certificate_public_key_digest "${root}/registry/client.crt")" == \
     "$(private_key_public_digest "${root}/node/campus-connector-client.key")" ]] ||
    die "node client certificate and private key do not match"
  [[ "$(public_key_digest "${root}/registry/signing.pub")" == \
     "$(private_key_public_digest "${root}/node/campus-connector-signing.key")" ]] ||
    die "node signing public and private keys do not match"
  [[ "$(public_key_digest "${root}/node/campus-connector-snapshot-x25519.pub")" == \
     "$(private_key_public_digest "${root}/gateway/snapshot-x25519.key")" ]] ||
    die "snapshot encryption public and private keys do not match"

  local signing_digest snapshot_digest
  signing_digest="$(public_key_digest "${root}/registry/signing.pub")"
  snapshot_digest="$(public_key_digest "${root}/node/campus-connector-snapshot-x25519.pub")"
  [[ "${recorded_signing_id}" == "cc-node-ed25519-${signing_digest:0:16}" ]] ||
    die "signing key ID does not match its public key"
  [[ "${recorded_snapshot_id}" == "cc-snapshot-x25519-${snapshot_digest:0:16}" ]] ||
    die "snapshot key ID does not match its public key"

  require_private_mode "${root}/authority/gateway-ca.key"
  require_private_mode "${root}/authority/client-ca.key"
  require_private_mode "${root}/gateway/gateway.key"
  require_private_mode "${root}/gateway/snapshot-x25519.key"
  require_private_mode "${root}/node/campus-connector-client.key"
  require_private_mode "${root}/node/campus-connector-signing.key"
  log "validated existing PKI at ${root} (node ${recorded_node}, host ${recorded_host})"
}

if [[ -e "${output_dir}" ]]; then
  validate_existing "${output_dir}"
  exit 0
fi
if [[ "${check_only}" == "true" ]]; then
  die "PKI directory does not exist: ${output_dir}"
fi

if [[ -z "${node_id}" ]]; then
  [[ -r /proc/sys/kernel/random/uuid ]] || die "cannot generate a node UUID"
  node_id="$(tr 'A-F' 'a-f' </proc/sys/kernel/random/uuid)"
fi

parent_dir="$(dirname "${output_dir}")"
mkdir -p -- "${parent_dir}"
tmp_dir="$(mktemp -d "${parent_dir}/.campus-connector-pki.XXXXXX")"
cleanup() {
  if [[ -n "${tmp_dir:-}" && -d "${tmp_dir}" ]]; then
    rm -rf -- "${tmp_dir}"
  fi
}
trap cleanup EXIT
umask 077
mkdir -m 0700 \
  "${tmp_dir}/authority" "${tmp_dir}/gateway" \
  "${tmp_dir}/node" "${tmp_dir}/registry"

openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:P-256 \
  -out "${tmp_dir}/authority/gateway-ca.key" >/dev/null 2>&1
openssl req -new -x509 -sha256 -days 3650 \
  -key "${tmp_dir}/authority/gateway-ca.key" \
  -subj "/CN=StuHelper Campus Connector Gateway CA" \
  -addext "basicConstraints=critical,CA:TRUE,pathlen:0" \
  -addext "keyUsage=critical,keyCertSign,cRLSign" \
  -out "${tmp_dir}/authority/gateway-ca.crt" >/dev/null 2>&1

openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:P-256 \
  -out "${tmp_dir}/gateway/gateway.key" >/dev/null 2>&1
openssl req -new -sha256 \
  -key "${tmp_dir}/gateway/gateway.key" \
  -subj "/CN=${gateway_host}" \
  -out "${tmp_dir}/gateway/gateway.csr" >/dev/null 2>&1
if [[ "${gateway_host}" =~ ^([0-9]{1,3}\.){3}[0-9]{1,3}$ ]]; then
  gateway_san="IP:${gateway_host}"
else
  gateway_san="DNS:${gateway_host}"
fi
printf '%s\n' \
  'basicConstraints=critical,CA:FALSE' \
  'keyUsage=critical,digitalSignature' \
  'extendedKeyUsage=serverAuth' \
  "subjectAltName=${gateway_san}" >"${tmp_dir}/gateway.ext"
openssl x509 -req -sha256 -days 397 \
  -in "${tmp_dir}/gateway/gateway.csr" \
  -CA "${tmp_dir}/authority/gateway-ca.crt" \
  -CAkey "${tmp_dir}/authority/gateway-ca.key" \
  -CAcreateserial -extfile "${tmp_dir}/gateway.ext" \
  -out "${tmp_dir}/gateway/gateway.crt" >/dev/null 2>&1

openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:P-256 \
  -out "${tmp_dir}/authority/client-ca.key" >/dev/null 2>&1
openssl req -new -x509 -sha256 -days 3650 \
  -key "${tmp_dir}/authority/client-ca.key" \
  -subj "/CN=StuHelper Campus Connector Client CA" \
  -addext "basicConstraints=critical,CA:TRUE,pathlen:0" \
  -addext "keyUsage=critical,keyCertSign,cRLSign" \
  -out "${tmp_dir}/authority/client-ca.crt" >/dev/null 2>&1

openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:P-256 \
  -out "${tmp_dir}/node/campus-connector-client.key" >/dev/null 2>&1
openssl req -new -sha256 \
  -key "${tmp_dir}/node/campus-connector-client.key" \
  -subj "/CN=${node_id}" \
  -out "${tmp_dir}/node/client.csr" >/dev/null 2>&1
printf '%s\n' \
  'basicConstraints=critical,CA:FALSE' \
  'keyUsage=critical,digitalSignature' \
  'extendedKeyUsage=clientAuth' \
  "subjectAltName=URI:spiffe://stuhelper/campus-connector/${node_id}" >"${tmp_dir}/client.ext"
openssl x509 -req -sha256 -days 397 \
  -in "${tmp_dir}/node/client.csr" \
  -CA "${tmp_dir}/authority/client-ca.crt" \
  -CAkey "${tmp_dir}/authority/client-ca.key" \
  -CAcreateserial -extfile "${tmp_dir}/client.ext" \
  -out "${tmp_dir}/node/campus-connector-client.crt" >/dev/null 2>&1

openssl genpkey -algorithm ED25519 \
  -out "${tmp_dir}/node/campus-connector-signing.key" >/dev/null 2>&1
openssl pkey -in "${tmp_dir}/node/campus-connector-signing.key" -pubout \
  -out "${tmp_dir}/registry/signing.pub" >/dev/null 2>&1
openssl genpkey -algorithm X25519 \
  -out "${tmp_dir}/gateway/snapshot-x25519.key" >/dev/null 2>&1
openssl pkey -in "${tmp_dir}/gateway/snapshot-x25519.key" -pubout \
  -out "${tmp_dir}/node/campus-connector-snapshot-x25519.pub" >/dev/null 2>&1

install -m 0644 "${tmp_dir}/authority/gateway-ca.crt" \
  "${tmp_dir}/node/campus-connector-central-ca.crt"
install -m 0644 "${tmp_dir}/authority/client-ca.crt" \
  "${tmp_dir}/gateway/client-ca.crt"
install -m 0644 "${tmp_dir}/node/campus-connector-client.crt" \
  "${tmp_dir}/registry/client.crt"

rm -f -- \
  "${tmp_dir}/gateway/gateway.csr" "${tmp_dir}/gateway.ext" \
  "${tmp_dir}/node/client.csr" "${tmp_dir}/client.ext"
chmod 0600 \
  "${tmp_dir}/authority/gateway-ca.key" \
  "${tmp_dir}/authority/client-ca.key" \
  "${tmp_dir}/gateway/gateway.key" \
  "${tmp_dir}/gateway/snapshot-x25519.key" \
  "${tmp_dir}/node/campus-connector-client.key" \
  "${tmp_dir}/node/campus-connector-signing.key"
chmod 0644 \
  "${tmp_dir}/authority/gateway-ca.crt" \
  "${tmp_dir}/authority/client-ca.crt" \
  "${tmp_dir}/gateway/gateway.crt" \
  "${tmp_dir}/gateway/client-ca.crt" \
  "${tmp_dir}/node/campus-connector-client.crt" \
  "${tmp_dir}/node/campus-connector-central-ca.crt" \
  "${tmp_dir}/node/campus-connector-snapshot-x25519.pub" \
  "${tmp_dir}/registry/client.crt" \
  "${tmp_dir}/registry/signing.pub"

signing_digest="$(public_key_digest "${tmp_dir}/registry/signing.pub")"
snapshot_digest="$(public_key_digest "${tmp_dir}/node/campus-connector-snapshot-x25519.pub")"
signing_key_id="cc-node-ed25519-${signing_digest:0:16}"
snapshot_key_id="cc-snapshot-x25519-${snapshot_digest:0:16}"
generated_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
jq -n \
  --arg schemaVersion "1" \
  --arg generatedAt "${generated_at}" \
  --arg gatewayHost "${gateway_host}" \
  --arg nodeID "${node_id}" \
  --arg signingKeyID "${signing_key_id}" \
  --arg snapshotKeyID "${snapshot_key_id}" \
  '{
    schemaVersion: ($schemaVersion | tonumber),
    generatedAt: $generatedAt,
    gatewayHost: $gatewayHost,
    nodeID: $nodeID,
    signingKeyID: $signingKeyID,
    snapshotKeyID: $snapshotKeyID,
    gatewaySecretDirectory: "gateway",
    nodeSecretDirectory: "node",
    registryPublicDirectory: "registry"
  }' >"${tmp_dir}/public-metadata.json"
chmod 0644 "${tmp_dir}/public-metadata.json"

gateway_host=""
node_id=""
validate_existing "${tmp_dir}"

mv -T -- "${tmp_dir}" "${output_dir}"
tmp_dir=""
trap - EXIT
log "generated campus connector PKI at ${output_dir}; private CA keys must be moved to an offline secret store after enrollment"
