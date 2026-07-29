#!/bin/sh
set -eu

upstream_entrypoint="/usr/bin/docker-entrypoint.sh"
tls_source_dir="${POSTGRES_TLS_SOURCE_DIR:-/tls-source}"
client_tls_source_dir="${POSTGRES_CLIENT_TLS_SOURCE_DIR:-/client-tls-source}"
tls_runtime_dir="${POSTGRES_TLS_RUNTIME_DIR:-/tls}"
postgres_uid="${POSTGRES_RUNTIME_UID:-70}"
postgres_gid="${POSTGRES_RUNTIME_GID:-70}"

mkdir -p "${tls_runtime_dir}"

if [ "${1:-}" = "postgres" ]; then
  : "${POSTGRES_PASSWORD:?POSTGRES_PASSWORD is required for the internal PostgreSQL service}"
fi

if [ -f "${client_tls_source_dir}/ca.crt" ]; then
  install -o "${postgres_uid}" -g "${postgres_gid}" -m 0644 \
    "${client_tls_source_dir}/ca.crt" "${tls_runtime_dir}/ca.crt"
fi

if [ "${1:-}" = "postgres" ] && [ "${POSTGRES_ENABLE_SSL:-off}" = "on" ]; then
  for filename in ca.crt server.crt server.key; do
    if [ ! -f "${tls_source_dir}/${filename}" ]; then
      echo "[postgres-entrypoint][error] missing PostgreSQL TLS file: ${tls_source_dir}/${filename}" >&2
      exit 1
    fi
  done

  install -o "${postgres_uid}" -g "${postgres_gid}" -m 0644 \
    "${tls_source_dir}/ca.crt" "${tls_runtime_dir}/ca.crt"
  install -o "${postgres_uid}" -g "${postgres_gid}" -m 0644 \
    "${tls_source_dir}/server.crt" "${tls_runtime_dir}/server.crt"
  install -o "${postgres_uid}" -g "${postgres_gid}" -m 0600 \
    "${tls_source_dir}/server.key" "${tls_runtime_dir}/server.key"
fi

exec "${upstream_entrypoint}" "$@"
