#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

source_dir="${CAMPUS_CONNECTOR_PKI_DIR:-${REPO_ROOT}/infra/generated/campus-connector-pki}"
output_dir="${CAMPUS_CONNECTOR_GATEWAY_SECRET_DIR:-${REPO_ROOT}/infra/generated/campus-connector-gateway-runtime}"
gateway_host="${CAMPUS_CONNECTOR_GATEWAY_PUBLIC_HOST:-connector.stuhelper.com}"
owner_uid="${BACKEND_RUNTIME_UID:-$(id -u)}"
owner_gid="${BACKEND_RUNTIME_GID:-$(id -g)}"

usage() {
  cat <<'EOF'
Usage: infra/ops/prepare-campus-connector-gateway-runtime.sh [options]

Atomically export the minimal center-side Gateway runtime bundle from a fully
validated, offline-managed Campus Connector PKI hierarchy. Existing output is
validated and never overwritten.

Options:
  --source DIR          Full PKI hierarchy containing authority/node/registry
  --output DIR          Separate Gateway runtime bundle directory
  --gateway-host HOST   Exact DNS name or IPv4 address in gateway.crt
  --owner-uid UID       Runtime directory/private-key owner UID
  --owner-gid GID       Runtime directory/private-key owner GID
  -h, --help            Show this help
EOF
}

die() {
  printf '[campus-connector-gateway-prepare][error] %s\n' "$*" >&2
  exit 1
}

log() {
  printf '[campus-connector-gateway-prepare] %s\n' "$*"
}

while (($# > 0)); do
  case "$1" in
    --source)
      (($# >= 2)) || die "--source requires a value"
      source_dir="$2"
      shift 2
      ;;
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
    --owner-uid)
      (($# >= 2)) || die "--owner-uid requires a value"
      owner_uid="$2"
      shift 2
      ;;
    --owner-gid)
      (($# >= 2)) || die "--owner-gid requires a value"
      owner_gid="$2"
      shift 2
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

for command in install jq mktemp mv chown readlink; do
  command -v "${command}" >/dev/null 2>&1 || die "required command is missing: ${command}"
done
[[ "${owner_uid}" =~ ^[0-9]+$ ]] || die "owner UID must be numeric"
[[ "${owner_gid}" =~ ^[0-9]+$ ]] || die "owner GID must be numeric"

source_dir="$(readlink -m -- "${source_dir}")"
output_dir="$(readlink -m -- "${output_dir}")"
[[ "${output_dir}" != "${source_dir}" && "${output_dir}" != "${source_dir}/"* ]] ||
  die "runtime output must be separate from the full offline PKI hierarchy"

"${SCRIPT_DIR}/generate-campus-connector-pki.sh" \
  --check \
  --output "${source_dir}" \
  --gateway-host "${gateway_host}"
snapshot_key_id="$(jq -er '.snapshotKeyID' "${source_dir}/public-metadata.json")" ||
  die "full PKI metadata does not contain a valid snapshotKeyID"

validator=(
  "${SCRIPT_DIR}/validate-campus-connector-gateway-runtime.sh"
  --dir "${output_dir}"
  --gateway-host "${gateway_host}"
  --snapshot-key-id "${snapshot_key_id}"
  --expected-uid "${owner_uid}"
  --expected-gid "${owner_gid}"
)
if [[ -e "${output_dir}" ]]; then
  "${validator[@]}"
  log "validated existing Gateway runtime bundle at ${output_dir}"
  exit 0
fi

parent_dir="$(dirname "${output_dir}")"
mkdir -p -- "${parent_dir}"
tmp_dir="$(mktemp -d "${parent_dir}/.campus-connector-gateway-runtime.XXXXXX")"
# shellcheck disable=SC2317 # invoked by the EXIT trap
cleanup() {
  if [[ -n "${tmp_dir:-}" && -d "${tmp_dir}" ]]; then
    rm -rf -- "${tmp_dir}"
  fi
}
trap cleanup EXIT
chmod 0700 "${tmp_dir}"

install -m 0644 "${source_dir}/authority/gateway-ca.crt" "${tmp_dir}/gateway-ca.crt"
install -m 0644 "${source_dir}/gateway/gateway.crt" "${tmp_dir}/gateway.crt"
install -m 0600 "${source_dir}/gateway/gateway.key" "${tmp_dir}/gateway.key"
install -m 0644 "${source_dir}/gateway/client-ca.crt" "${tmp_dir}/client-ca.crt"
install -m 0600 "${source_dir}/gateway/snapshot-x25519.key" "${tmp_dir}/snapshot-x25519.key"
chown -R -- "${owner_uid}:${owner_gid}" "${tmp_dir}"

runtime_dir_before_move="${output_dir}"
output_dir="${tmp_dir}"
validator=(
  "${SCRIPT_DIR}/validate-campus-connector-gateway-runtime.sh"
  --dir "${output_dir}"
  --gateway-host "${gateway_host}"
  --snapshot-key-id "${snapshot_key_id}"
  --expected-uid "${owner_uid}"
  --expected-gid "${owner_gid}"
)
"${validator[@]}"
mv -T -- "${tmp_dir}" "${runtime_dir_before_move}"
tmp_dir=""
trap - EXIT
log "prepared minimal Gateway runtime bundle at ${runtime_dir_before_move}"

exit 0
