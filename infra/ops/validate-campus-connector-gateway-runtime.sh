#!/usr/bin/env bash
set -euo pipefail

runtime_dir="${CAMPUS_CONNECTOR_GATEWAY_SECRET_DIR:-}"
gateway_host="${CAMPUS_CONNECTOR_GATEWAY_PUBLIC_HOST:-}"
expected_snapshot_key_id="${CAMPUS_CONNECTOR_SNAPSHOT_KEY_ID:-}"
expected_uid="${BACKEND_RUNTIME_UID:-$(id -u)}"
expected_gid="${BACKEND_RUNTIME_GID:-$(id -g)}"
print_snapshot_key_id=false

usage() {
  cat <<'EOF'
Usage: infra/ops/validate-campus-connector-gateway-runtime.sh [options]

Validate the minimal Campus Connector Gateway runtime bundle. The directory
must contain exactly the two private runtime keys and three public certificates;
offline CA private keys and campus-node private material are rejected.

Options:
  --dir DIR                    Gateway runtime bundle directory
  --gateway-host HOST          Exact DNS name or IPv4 address in gateway.crt
  --snapshot-key-id ID         Expected cc-snapshot-x25519-* key identifier
  --expected-uid UID           Required directory/private-key owner UID
  --expected-gid GID           Required directory/private-key owner GID
  --print-snapshot-key-id      Print the validated snapshot key ID to stdout
  -h, --help                   Show this help
EOF
}

die() {
  printf '[campus-connector-gateway-runtime][error] %s\n' "$*" >&2
  exit 1
}

log() {
  printf '[campus-connector-gateway-runtime] %s\n' "$*" >&2
}

while (($# > 0)); do
  case "$1" in
    --dir)
      (($# >= 2)) || die "--dir requires a value"
      runtime_dir="$2"
      shift 2
      ;;
    --gateway-host)
      (($# >= 2)) || die "--gateway-host requires a value"
      gateway_host="$2"
      shift 2
      ;;
    --snapshot-key-id)
      (($# >= 2)) || die "--snapshot-key-id requires a value"
      expected_snapshot_key_id="$2"
      shift 2
      ;;
    --expected-uid)
      (($# >= 2)) || die "--expected-uid requires a value"
      expected_uid="$2"
      shift 2
      ;;
    --expected-gid)
      (($# >= 2)) || die "--expected-gid requires a value"
      expected_gid="$2"
      shift 2
      ;;
    --print-snapshot-key-id)
      print_snapshot_key_id=true
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

for command in openssl stat awk find grep readlink; do
  command -v "${command}" >/dev/null 2>&1 || die "required command is missing: ${command}"
done

[[ -n "${runtime_dir}" ]] || die "--dir is required"
[[ -n "${gateway_host}" && "${gateway_host}" != *://* && "${gateway_host}" != */* &&
   "${gateway_host}" != *:* && "${gateway_host}" != *' '* ]] ||
  die "gateway host must be one exact DNS name or IPv4 address without scheme, port, path, or wildcard"
[[ "${expected_uid}" =~ ^[0-9]+$ ]] || die "expected UID must be numeric"
[[ "${expected_gid}" =~ ^[0-9]+$ ]] || die "expected GID must be numeric"
if [[ -n "${expected_snapshot_key_id}" ]]; then
  [[ "${expected_snapshot_key_id}" =~ ^cc-snapshot-x25519-[0-9a-f]{16}$ ]] ||
    die "expected snapshot key ID has an invalid format"
fi

runtime_dir="$(readlink -m -- "${runtime_dir}")"
[[ -d "${runtime_dir}" && ! -L "${runtime_dir}" ]] ||
  die "runtime bundle must be a real directory: ${runtime_dir}"
[[ "$(stat -c '%a' -- "${runtime_dir}")" == "700" ]] ||
  die "runtime bundle directory must have mode 0700: ${runtime_dir}"
[[ "$(stat -c '%u' -- "${runtime_dir}")" == "${expected_uid}" ]] ||
  die "runtime bundle directory owner UID does not match BACKEND_RUNTIME_UID"
[[ "$(stat -c '%g' -- "${runtime_dir}")" == "${expected_gid}" ]] ||
  die "runtime bundle directory owner GID does not match BACKEND_RUNTIME_GID"

required_files=(
  gateway-ca.crt
  gateway.crt
  gateway.key
  client-ca.crt
  snapshot-x25519.key
)
for filename in "${required_files[@]}"; do
  path="${runtime_dir}/${filename}"
  [[ -f "${path}" && -s "${path}" && ! -L "${path}" ]] ||
    die "missing, empty, non-regular, or symlinked runtime file: ${path}"
done

mapfile -t unexpected_paths < <(
  find "${runtime_dir}" -mindepth 1 -maxdepth 1 \
    ! -name gateway-ca.crt \
    ! -name gateway.crt \
    ! -name gateway.key \
    ! -name client-ca.crt \
    ! -name snapshot-x25519.key \
    -print
)
if ((${#unexpected_paths[@]} > 0)); then
  die "runtime bundle contains unexpected material; expected exactly five Gateway files"
fi

for filename in gateway-ca.crt gateway.crt client-ca.crt; do
  mode="$(stat -c '%a' -- "${runtime_dir}/${filename}")"
  case "${mode}" in
    444|644) ;;
    *) die "public certificate must have mode 0444 or 0644: ${runtime_dir}/${filename}" ;;
  esac
done

for filename in gateway.key snapshot-x25519.key; do
  path="${runtime_dir}/${filename}"
  mode="$(stat -c '%a' -- "${path}")"
  case "${mode}" in
    400|600) ;;
    *) die "private key must have mode 0400 or 0600: ${path}" ;;
  esac
  [[ "$(stat -c '%u' -- "${path}")" == "${expected_uid}" ]] ||
    die "private key owner UID does not match BACKEND_RUNTIME_UID: ${path}"
  [[ "$(stat -c '%g' -- "${path}")" == "${expected_gid}" ]] ||
    die "private key owner GID does not match BACKEND_RUNTIME_GID: ${path}"
done

openssl verify -CAfile "${runtime_dir}/gateway-ca.crt" \
  "${runtime_dir}/gateway-ca.crt" >/dev/null || die "gateway CA certificate validation failed"
openssl verify -CAfile "${runtime_dir}/client-ca.crt" \
  "${runtime_dir}/client-ca.crt" >/dev/null || die "client CA certificate validation failed"
openssl verify -purpose sslserver \
  -CAfile "${runtime_dir}/gateway-ca.crt" \
  "${runtime_dir}/gateway.crt" >/dev/null || die "gateway certificate chain validation failed"
if [[ "${gateway_host}" =~ ^([0-9]{1,3}\.){3}[0-9]{1,3}$ ]]; then
  gateway_identity_check="$(
    openssl x509 -in "${runtime_dir}/gateway.crt" -noout -checkip "${gateway_host}" 2>/dev/null
  )"
else
  gateway_identity_check="$(
    openssl x509 -in "${runtime_dir}/gateway.crt" -noout -checkhost "${gateway_host}" 2>/dev/null
  )"
fi
grep -Fq 'does match certificate' <<<"${gateway_identity_check}" ||
  die "gateway certificate does not cover ${gateway_host}"
for filename in gateway-ca.crt gateway.crt client-ca.crt; do
  openssl x509 -in "${runtime_dir}/${filename}" -noout -checkend 2592000 >/dev/null ||
    die "certificate expires in less than 30 days: ${runtime_dir}/${filename}"
done

certificate_public_key_digest() {
  openssl x509 -in "$1" -pubkey -noout 2>/dev/null |
    openssl pkey -pubin -outform DER 2>/dev/null |
    openssl dgst -sha256 2>/dev/null |
    awk '{print $2}'
}

private_key_public_digest() {
  openssl pkey -in "$1" -pubout -outform DER 2>/dev/null |
    openssl dgst -sha256 2>/dev/null |
    awk '{print $2}'
}

[[ "$(certificate_public_key_digest "${runtime_dir}/gateway.crt")" == \
   "$(private_key_public_digest "${runtime_dir}/gateway.key")" ]] ||
  die "gateway certificate and private key do not match"

openssl pkey \
  -in "${runtime_dir}/snapshot-x25519.key" \
  -pubout \
  -outform DER 2>/dev/null |
  openssl asn1parse -inform DER 2>/dev/null |
  grep -Eq 'prim:[[:space:]]+OBJECT[[:space:]]*:X25519$' ||
  die "snapshot private key must use X25519"
snapshot_digest="$(private_key_public_digest "${runtime_dir}/snapshot-x25519.key")"
[[ "${snapshot_digest}" =~ ^[0-9a-f]{64}$ ]] || die "could not derive snapshot public-key digest"
snapshot_key_id="cc-snapshot-x25519-${snapshot_digest:0:16}"
if [[ -n "${expected_snapshot_key_id}" && "${snapshot_key_id}" != "${expected_snapshot_key_id}" ]]; then
  die "snapshot private key does not match CAMPUS_CONNECTOR_SNAPSHOT_KEY_ID"
fi

log "validated minimal Gateway runtime bundle for ${gateway_host}"
if [[ "${print_snapshot_key_id}" == "true" ]]; then
  printf '%s\n' "${snapshot_key_id}"
fi

exit 0
