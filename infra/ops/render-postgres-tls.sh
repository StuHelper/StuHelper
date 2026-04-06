#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

require_cmd openssl

load_env

POSTGRES_TLS_DIR="${POSTGRES_TLS_DIR:-${REPO_ROOT}/infra/generated/postgres}"
SERVER_KEY="${POSTGRES_TLS_DIR}/server.key"
SERVER_CERT="${POSTGRES_TLS_DIR}/server.crt"
COMMON_NAME="${POSTGRES_SSL_COMMON_NAME:-postgres}"
SAN_LIST="${POSTGRES_SSL_SAN_LIST:-DNS:postgres,DNS:localhost,IP:127.0.0.1}"

mkdir -p "${POSTGRES_TLS_DIR}"

if [[ "${POSTGRES_ENABLE_SSL:-off}" != "on" ]]; then
  log "POSTGRES_ENABLE_SSL=${POSTGRES_ENABLE_SSL:-off}; skipping PostgreSQL TLS material generation"
  exit 0
fi

if [[ -f "${SERVER_KEY}" && -f "${SERVER_CERT}" ]]; then
  log "PostgreSQL TLS material already exists: ${POSTGRES_TLS_DIR}"
  exit 0
fi

openssl req \
  -x509 \
  -nodes \
  -newkey rsa:4096 \
  -sha256 \
  -days 825 \
  -subj "/CN=${COMMON_NAME}" \
  -addext "subjectAltName=${SAN_LIST}" \
  -keyout "${SERVER_KEY}" \
  -out "${SERVER_CERT}"

chmod 600 "${SERVER_KEY}"
chmod 644 "${SERVER_CERT}"

log "generated PostgreSQL TLS certificate at ${POSTGRES_TLS_DIR}"
