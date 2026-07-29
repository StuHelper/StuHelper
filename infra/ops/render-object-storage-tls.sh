#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

require_cmd openssl

openssl_cmd() {
  MSYS2_ARG_CONV_EXCL="/CN=" openssl "$@"
}

remove_path() {
  local path="$1"
  if rm -rf "${path}" 2>/dev/null; then
    return
  fi
  if command -v sudo >/dev/null 2>&1 && sudo -n true 2>/dev/null; then
    sudo rm -rf "${path}"
    return
  fi
  die "failed to remove ${path}; remove it manually and rerun"
}

ensure_dir_owner() {
  local path="$1"
  if [[ -w "${path}" ]]; then
    return
  fi
  if command -v sudo >/dev/null 2>&1 && sudo -n true 2>/dev/null; then
    sudo chown "$(id -u):$(id -g)" "${path}"
    return
  fi
  die "${path} is not writable; fix ownership and rerun"
}

load_env

OBJECT_STORAGE_TLS_DIR="${OBJECT_STORAGE_TLS_DIR:-${REPO_ROOT}/infra/generated/object-storage}"
CA_KEY="${OBJECT_STORAGE_TLS_DIR}/ca.key"
CA_CERT="${OBJECT_STORAGE_TLS_DIR}/ca.crt"
SERVER_KEY="${OBJECT_STORAGE_TLS_DIR}/private.key"
SERVER_CERT="${OBJECT_STORAGE_TLS_DIR}/public.crt"
COMMON_NAME="${LOCAL_OBJECT_STORAGE_TLS_COMMON_NAME:-object-storage}"

if [[ -d "${CA_CERT}" ]]; then
  warn "${CA_CERT} is a directory; removing stale Docker-created bind source"
  remove_path "${CA_CERT}"
fi
if [[ -d "${CA_KEY}" ]]; then
  warn "${CA_KEY} is a directory; removing stale Docker-created bind source"
  remove_path "${CA_KEY}"
fi
if [[ -d "${SERVER_KEY}" ]]; then
  warn "${SERVER_KEY} is a directory; removing stale Docker-created bind source"
  remove_path "${SERVER_KEY}"
fi
if [[ -d "${SERVER_CERT}" ]]; then
  warn "${SERVER_CERT} is a directory; removing stale Docker-created bind source"
  remove_path "${SERVER_CERT}"
fi

mkdir -p "${OBJECT_STORAGE_TLS_DIR}"
ensure_dir_owner "${OBJECT_STORAGE_TLS_DIR}"

ensure_object_storage_tls_permissions() {
  chmod 755 "${OBJECT_STORAGE_TLS_DIR}"
  if [[ -f "${CA_KEY}" ]]; then chmod 600 "${CA_KEY}"; fi
  if [[ -f "${CA_CERT}" ]]; then chmod 644 "${CA_CERT}"; fi
  if [[ -f "${SERVER_KEY}" ]]; then chmod 600 "${SERVER_KEY}"; fi
  if [[ -f "${SERVER_CERT}" ]]; then chmod 644 "${SERVER_CERT}"; fi
}

ensure_object_storage_tls_permissions

if [[ -f "${CA_KEY}" && -f "${CA_CERT}" && -f "${SERVER_KEY}" && -f "${SERVER_CERT}" ]]; then
  ensure_object_storage_tls_permissions
  log "local object-storage TLS material already exists: ${SERVER_CERT}"
  exit 0
fi

if [[ ! -f "${CA_KEY}" || ! -f "${CA_CERT}" ]]; then
  openssl_cmd genrsa -out "${CA_KEY}" 4096 >/dev/null 2>&1
  openssl_cmd req \
    -x509 \
    -new \
    -nodes \
    -key "${CA_KEY}" \
    -sha256 \
    -days 3650 \
    -subj "/CN=${COMMON_NAME}-ca" \
    -out "${CA_CERT}" >/dev/null 2>&1
fi

openssl_cmd genrsa -out "${SERVER_KEY}" 3072 >/dev/null 2>&1
csr_file="${OBJECT_STORAGE_TLS_DIR}/server.csr"
openssl_cmd req \
  -new \
  -key "${SERVER_KEY}" \
  -sha256 \
  -subj "/CN=${COMMON_NAME}" \
  -addext "subjectAltName=DNS:${COMMON_NAME},DNS:object-storage,DNS:localhost,DNS:host.docker.internal,IP:127.0.0.1" \
  -out "${csr_file}" >/dev/null 2>&1
openssl_cmd x509 \
  -req \
  -in "${csr_file}" \
  -CA "${CA_CERT}" \
  -CAkey "${CA_KEY}" \
  -CAcreateserial \
  -days 825 \
  -sha256 \
  -copy_extensions copy \
  -out "${SERVER_CERT}" >/dev/null 2>&1
rm -f "${csr_file}" "${OBJECT_STORAGE_TLS_DIR}/ca.srl"

ensure_object_storage_tls_permissions

openssl_cmd verify -CAfile "${CA_CERT}" "${SERVER_CERT}" >/dev/null
log "generated local object-storage TLS material at ${OBJECT_STORAGE_TLS_DIR}"
