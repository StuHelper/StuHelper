#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

require_cmd openssl

openssl_cmd() {
  MSYS2_ARG_CONV_EXCL="/CN=" openssl "$@"
}

load_env

REDIS_TLS_DIR="${REDIS_TLS_DIR:-${REPO_ROOT}/infra/generated/redis}"
CA_KEY="${REDIS_TLS_DIR}/ca.key"
CA_CERT="${REDIS_TLS_DIR}/ca.crt"
SERVER_KEY="${REDIS_TLS_DIR}/server.key"
SERVER_CERT="${REDIS_TLS_DIR}/server.crt"
SERVER_CSR="${REDIS_TLS_DIR}/server.csr"
COMMON_NAME="${REDIS_SSL_COMMON_NAME:-redis}"
SAN_LIST="${REDIS_SSL_SAN_LIST:-DNS:redis,DNS:localhost,IP:127.0.0.1}"

mkdir -p "${REDIS_TLS_DIR}"

if [[ "${REDIS_TLS_ENABLED:-false}" != "true" ]]; then
  log "REDIS_TLS_ENABLED=${REDIS_TLS_ENABLED:-false}; skipping Redis TLS material generation"
  exit 0
fi

if [[ -f "${CA_KEY}" && -f "${CA_CERT}" && -f "${SERVER_KEY}" && -f "${SERVER_CERT}" ]]; then
  log "Redis TLS material already exists: ${REDIS_TLS_DIR}"
  exit 0
fi

extfile="$(mktemp)"
trap 'rm -f "${extfile}" "${SERVER_CSR}"' EXIT
printf 'subjectAltName=%s\n' "${SAN_LIST}" >"${extfile}"

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

openssl_cmd genrsa -out "${SERVER_KEY}" 4096 >/dev/null 2>&1
openssl_cmd req \
  -new \
  -key "${SERVER_KEY}" \
  -subj "/CN=${COMMON_NAME}" \
  -out "${SERVER_CSR}" >/dev/null 2>&1
openssl_cmd x509 \
  -req \
  -in "${SERVER_CSR}" \
  -CA "${CA_CERT}" \
  -CAkey "${CA_KEY}" \
  -CAcreateserial \
  -out "${SERVER_CERT}" \
  -days 825 \
  -sha256 \
  -extfile "${extfile}" >/dev/null 2>&1

chmod 600 "${CA_KEY}" "${SERVER_KEY}"
chmod 644 "${CA_CERT}" "${SERVER_CERT}"

log "generated Redis CA/server TLS material at ${REDIS_TLS_DIR}"
