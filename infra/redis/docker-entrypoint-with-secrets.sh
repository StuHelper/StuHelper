#!/bin/sh
set -eu

upstream_entrypoint="/usr/local/bin/docker-entrypoint.sh"
source_dir="${REDIS_SECRET_SOURCE_DIR:-/redis-source}"
runtime_dir="${REDIS_SECRET_RUNTIME_DIR:-/redis-runtime}"
redis_uid="${REDIS_RUNTIME_UID:-999}"
redis_gid="${REDIS_RUNTIME_GID:-1000}"

if [ "${REDIS_TLS_ENABLED:-true}" = "true" ]; then
  for filename in ca.crt server.crt server.key users.acl; do
    if [ ! -f "${source_dir}/${filename}" ]; then
      echo "[redis-entrypoint][error] missing Redis runtime secret: ${source_dir}/${filename}" >&2
      exit 1
    fi
  done

  mkdir -p "${runtime_dir}"
  install -o "${redis_uid}" -g "${redis_gid}" -m 0644 \
    "${source_dir}/ca.crt" "${runtime_dir}/ca.crt"
  install -o "${redis_uid}" -g "${redis_gid}" -m 0644 \
    "${source_dir}/server.crt" "${runtime_dir}/server.crt"
  install -o "${redis_uid}" -g "${redis_gid}" -m 0600 \
    "${source_dir}/server.key" "${runtime_dir}/server.key"
  install -o "${redis_uid}" -g "${redis_gid}" -m 0600 \
    "${source_dir}/users.acl" "${runtime_dir}/users.acl"
fi

exec "${upstream_entrypoint}" "$@"
