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

MINIO_TLS_DIR="${MINIO_TLS_DIR:-${REPO_ROOT}/infra/generated/minio}"
CA_KEY="${MINIO_TLS_DIR}/ca.key"
CA_CERT="${MINIO_TLS_DIR}/ca.crt"
COMMON_NAME="${MINIO_SSL_COMMON_NAME:-minio}"

if [[ -d "${CA_CERT}" ]]; then
  warn "${CA_CERT} is a directory; removing stale Docker-created bind source"
  remove_path "${CA_CERT}"
fi
if [[ -d "${CA_KEY}" ]]; then
  warn "${CA_KEY} is a directory; removing stale Docker-created bind source"
  remove_path "${CA_KEY}"
fi

mkdir -p "${MINIO_TLS_DIR}"
ensure_dir_owner "${MINIO_TLS_DIR}"

if [[ -f "${CA_KEY}" && -f "${CA_CERT}" ]]; then
  chmod 600 "${CA_KEY}"
  chmod 644 "${CA_CERT}"
  log "MinIO CA bundle already exists: ${CA_CERT}"
  exit 0
fi

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

chmod 600 "${CA_KEY}"
chmod 644 "${CA_CERT}"

log "generated MinIO CA bundle at ${CA_CERT}"
