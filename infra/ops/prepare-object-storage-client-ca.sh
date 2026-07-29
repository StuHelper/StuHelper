#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

require_cmd openssl

load_env

client_ca_dir="${OBJECT_STORAGE_CLIENT_CA_DIR:-${REPO_ROOT}/infra/generated/object-storage-client-ca}"
client_ca_file="${client_ca_dir}/ca.crt"
container_ca="${OBJECT_STORAGE_TLS_CA:-}"
host_ca="${OBJECT_STORAGE_TLS_CA_HOST_PATH:-}"

[[ ! -L "${client_ca_dir}" ]] ||
  die "object-storage client CA directory must not be a symlink: ${client_ca_dir}"
mkdir -p "${client_ca_dir}"
[[ -d "${client_ca_dir}" && -w "${client_ca_dir}" ]] ||
  die "object-storage client CA directory is not writable: ${client_ca_dir}"
chmod 755 "${client_ca_dir}"
unexpected_entry="$(find "${client_ca_dir}" -mindepth 1 -maxdepth 1 ! -name ca.crt -print -quit)"
[[ -z "${unexpected_entry}" ]] ||
  die "object-storage client CA directory contains an unexpected entry: ${unexpected_entry}"
[[ ! -L "${client_ca_file}" ]] ||
  die "object-storage client CA destination must not be a symlink: ${client_ca_file}"

if [[ -z "${container_ca}" ]]; then
  [[ -z "${host_ca}" ]] ||
    die "OBJECT_STORAGE_TLS_CA_HOST_PATH must be empty when OBJECT_STORAGE_TLS_CA is empty"
  [[ ! -d "${client_ca_file}" ]] ||
    die "object-storage client CA destination is unexpectedly a directory: ${client_ca_file}"
  rm -f -- "${client_ca_file}"
  log "object-storage uses the system CA bundle; prepared an empty client CA mount"
  exit 0
fi

[[ "${container_ca}" == "/object-storage-tls/ca.crt" ]] ||
  die "OBJECT_STORAGE_TLS_CA must be /object-storage-tls/ca.crt when a private CA is configured"
[[ -n "${host_ca}" ]] ||
  die "OBJECT_STORAGE_TLS_CA_HOST_PATH is required when OBJECT_STORAGE_TLS_CA is configured"
[[ -f "${host_ca}" && -r "${host_ca}" ]] ||
  die "OBJECT_STORAGE_TLS_CA_HOST_PATH must be a readable regular file"
[[ ! -d "${client_ca_file}" ]] ||
  die "object-storage client CA destination is unexpectedly a directory: ${client_ca_file}"

if grep -Eq -- '-----BEGIN ([A-Z0-9 ]+ )?PRIVATE KEY-----' "${host_ca}"; then
  die "OBJECT_STORAGE_TLS_CA_HOST_PATH must contain certificates only, not a private key"
fi
if ! openssl crl2pkcs7 -nocrl -certfile "${host_ca}" |
  openssl pkcs7 -print_certs -noout >/dev/null 2>&1; then
  die "OBJECT_STORAGE_TLS_CA_HOST_PATH is not a valid PEM certificate bundle"
fi

temporary_ca="$(mktemp "${client_ca_dir}/.ca.crt.XXXXXX")"
cleanup() {
  rm -f -- "${temporary_ca}"
}
trap cleanup EXIT

cp -- "${host_ca}" "${temporary_ca}"
chmod 644 "${temporary_ca}"
mv -f -- "${temporary_ca}" "${client_ca_file}"
trap - EXIT

cmp -s "${host_ca}" "${client_ca_file}" ||
  die "object-storage client CA copy verification failed"
log "prepared object-storage client CA bundle: ${client_ca_file}"
