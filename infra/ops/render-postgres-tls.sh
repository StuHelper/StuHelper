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

POSTGRES_TLS_DIR="${POSTGRES_TLS_DIR:-${REPO_ROOT}/infra/generated/postgres}"
CA_KEY="${POSTGRES_TLS_DIR}/ca.key"
CA_CERT="${POSTGRES_TLS_DIR}/ca.crt"
SERVER_KEY="${POSTGRES_TLS_DIR}/server.key"
SERVER_CERT="${POSTGRES_TLS_DIR}/server.crt"
SERVER_CSR="${POSTGRES_TLS_DIR}/server.csr"
COMMON_NAME="${POSTGRES_SSL_COMMON_NAME:-postgres}"
SAN_LIST="${POSTGRES_SSL_SAN_LIST:-DNS:postgres,DNS:localhost,IP:127.0.0.1}"
POSTGRES_TLS_SERVER_KEY_OWNER="${POSTGRES_TLS_SERVER_KEY_OWNER:-70:70}"

mkdir -p "${POSTGRES_TLS_DIR}"

ensure_postgres_tls_permissions() {
  chmod 755 "${POSTGRES_TLS_DIR}"
  [[ -f "${CA_KEY}" ]] && chmod 600 "${CA_KEY}"
  if [[ -f "${SERVER_KEY}" ]]; then
    if [[ -n "${POSTGRES_TLS_SERVER_KEY_OWNER}" && "$(id -u)" == "0" ]]; then
      chown "${POSTGRES_TLS_SERVER_KEY_OWNER}" "${SERVER_KEY}"
    fi
    chmod 600 "${SERVER_KEY}"
  fi
  [[ -f "${CA_CERT}" ]] && chmod 644 "${CA_CERT}"
  [[ -f "${SERVER_CERT}" ]] && chmod 644 "${SERVER_CERT}"
}

ensure_postgres_tls_permissions

if [[ "${POSTGRES_ENABLE_SSL:-off}" != "on" ]]; then
  log "POSTGRES_ENABLE_SSL=${POSTGRES_ENABLE_SSL:-off}; skipping PostgreSQL TLS material generation"
  exit 0
fi

if [[ -f "${CA_KEY}" && -f "${CA_CERT}" && -f "${SERVER_KEY}" && -f "${SERVER_CERT}" ]]; then
  ensure_postgres_tls_permissions
  log "PostgreSQL TLS material already exists: ${POSTGRES_TLS_DIR}"
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

ensure_postgres_tls_permissions

log "generated PostgreSQL CA/server TLS material at ${POSTGRES_TLS_DIR}"
