#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: infra/ops/ensure-baota-runtime-permissions.sh [options]

Normalizes host-side bind-mount permissions used by the Baota production
Compose deployment. Run this after replacing the source directory and before
recreating containers that mount generated TLS, Redis ACL, or Casdoor files.

Options:
  --source-dir PATH          StuHelper source directory. Default: current repo.
  --casdoor-compose-root PATH
                             External Casdoor Compose root. Default:
                             /www/server/panel/data/compose/casdoor when present.
  --openlist-data-dir PATH   External OpenList data directory. Default:
                             /opt/openlist/data when present.
  --skip-casdoor             Do not touch the external Casdoor Compose root.
  --skip-openlist            Do not touch the external OpenList data directory.
  --apply                    Write permissions. Without this flag, dry-run only.
  -h, --help                 Show this help.

Environment:
  POSTGRES_TLS_SERVER_KEY_OWNER  Host owner for PostgreSQL server.key when run
                                 as root. Default: 70:70 for postgres:*-alpine.
  REDIS_TLS_SERVER_KEY_OWNER     Host owner for Redis server.key and users.acl.
                                 Default: 999:1000 for redis:*-alpine.
  CASDOOR_CONTAINER_OWNER        Host owner for Casdoor conf/logs when run as
                                 root. Default: 1000:1000 for casbin/casdoor.
  OPENLIST_CONTAINER_OWNER       Host owner for OpenList runtime data when run
                                 as root. Default: 1001:1001.

The script does not read or print secret file contents.
USAGE
}

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

source_dir="${BAOTA_SOURCE_DIR:-${REPO_ROOT}}"
casdoor_compose_root="${CASDOOR_COMPOSE_ROOT:-/www/server/panel/data/compose/casdoor}"
openlist_data_dir="${OPENLIST_DATA_DIR:-/opt/openlist/data}"
skip_casdoor=false
skip_openlist=false
apply=false

postgres_key_owner="${POSTGRES_TLS_SERVER_KEY_OWNER:-70:70}"
redis_key_owner="${REDIS_TLS_SERVER_KEY_OWNER:-999:1000}"
casdoor_owner="${CASDOOR_CONTAINER_OWNER:-1000:1000}"
openlist_owner="${OPENLIST_CONTAINER_OWNER:-1001:1001}"
validation_failed=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    --source-dir) source_dir="${2:-}"; shift 2 ;;
    --casdoor-compose-root) casdoor_compose_root="${2:-}"; shift 2 ;;
    --openlist-data-dir) openlist_data_dir="${2:-}"; shift 2 ;;
    --skip-casdoor) skip_casdoor=true; shift ;;
    --skip-openlist) skip_openlist=true; shift ;;
    --apply) apply=true; shift ;;
    -h|--help) usage; exit 0 ;;
    *)
      echo "[ensure-baota-runtime-permissions][error] unknown option: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

die() {
  echo "[ensure-baota-runtime-permissions][error] $*" >&2
  exit 1
}

log() {
  echo "[ensure-baota-runtime-permissions] $*"
}

warn() {
  echo "[ensure-baota-runtime-permissions][warning] $*" >&2
}

[[ -d "${source_dir}" ]] || die "source dir not found: ${source_dir}"
source_dir="$(cd "${source_dir}" && pwd)"

run_or_print() {
  if [[ "${apply}" == "true" ]]; then
    "$@"
  else
    printf '[ensure-baota-runtime-permissions] dry-run:'
    printf ' %q' "$@"
    printf '\n'
  fi
}

maybe_chown() {
  local owner="$1"
  shift
  [[ -n "${owner}" ]] || return 0
  if [[ "$(id -u)" != "0" ]]; then
    log "skip chown ${owner}; not running as root"
    return 0
  fi
  run_or_print chown "${owner}" "$@"
}

chmod_if_exists() {
  local mode="$1"
  local path="$2"
  [[ -e "${path}" ]] || return 0
  run_or_print chmod "${mode}" "${path}"
}

normalize_readonly_tree() {
  local path="$1"
  [[ -d "${path}" ]] || return 0

  while IFS= read -r -d '' entry; do
    run_or_print chmod 755 "${entry}"
  done < <(find "${path}" -type d -print0)
  while IFS= read -r -d '' entry; do
    run_or_print chmod 644 "${entry}"
  done < <(find "${path}" -type f -print0)
}

normalize_owned_tree() {
  local path="$1"
  local owner="$2"
  local directory_mode="$3"
  local file_mode="$4"
  [[ -d "${path}" ]] || return 0

  while IFS= read -r -d '' entry; do
    maybe_chown "${owner}" "${entry}"
    run_or_print chmod "${directory_mode}" "${entry}"
  done < <(find "${path}" -type d -print0)
  while IFS= read -r -d '' entry; do
    maybe_chown "${owner}" "${entry}"
    run_or_print chmod "${file_mode}" "${entry}"
  done < <(find "${path}" -type f -print0)
}

postgres_tls_dir="${source_dir}/infra/generated/postgres"
redis_tls_dir="${source_dir}/infra/generated/redis"
minio_tls_dir="${source_dir}/infra/generated/minio"
postgres_config_dir="${source_dir}/infra/postgres"
observability_dir="${source_dir}/infra/observability"
generated_observability_dir="${source_dir}/infra/generated/observability"

chmod_if_exists 755 "${source_dir}/infra"
chmod_if_exists 755 "${source_dir}/infra/generated"

if [[ -d "${postgres_tls_dir}" ]]; then
  log "normalizing PostgreSQL TLS permissions: ${postgres_tls_dir}"
  chmod_if_exists 755 "${postgres_tls_dir}"
  if [[ -f "${postgres_tls_dir}/server.key" ]]; then
    maybe_chown "${postgres_key_owner}" "${postgres_tls_dir}/server.key"
    chmod_if_exists 600 "${postgres_tls_dir}/server.key"
  fi
  chmod_if_exists 600 "${postgres_tls_dir}/ca.key"
  chmod_if_exists 644 "${postgres_tls_dir}/ca.crt"
  chmod_if_exists 644 "${postgres_tls_dir}/server.crt"
fi

if [[ -d "${redis_tls_dir}" ]]; then
  log "normalizing Redis TLS/ACL permissions: ${redis_tls_dir}"
  chmod_if_exists 755 "${redis_tls_dir}"
  chmod_if_exists 600 "${redis_tls_dir}/ca.key"
  chmod_if_exists 644 "${redis_tls_dir}/ca.crt"
  chmod_if_exists 644 "${redis_tls_dir}/server.crt"
  if [[ -f "${redis_tls_dir}/server.key" ]]; then
    maybe_chown "${redis_key_owner}" "${redis_tls_dir}/server.key"
    chmod_if_exists 600 "${redis_tls_dir}/server.key"
  fi
  if [[ -f "${redis_tls_dir}/users.acl" ]]; then
    maybe_chown "${redis_key_owner}" "${redis_tls_dir}/users.acl"
    chmod_if_exists 640 "${redis_tls_dir}/users.acl"
  fi
fi

if [[ -d "${minio_tls_dir}" ]]; then
  log "normalizing MinIO TLS permissions: ${minio_tls_dir}"
  chmod_if_exists 755 "${minio_tls_dir}"
  chmod_if_exists 600 "${minio_tls_dir}/ca.key"
  chmod_if_exists 600 "${minio_tls_dir}/private.key"
  chmod_if_exists 644 "${minio_tls_dir}/ca.crt"
  chmod_if_exists 644 "${minio_tls_dir}/ca-bundle.crt"
  chmod_if_exists 644 "${minio_tls_dir}/public.crt"
fi

if [[ -d "${postgres_config_dir}" ]]; then
  log "normalizing PostgreSQL bind-mount config permissions: ${postgres_config_dir}"
  chmod_if_exists 755 "${postgres_config_dir}"
  chmod_if_exists 755 "${postgres_config_dir}/init-extra-dbs.sh"
  chmod_if_exists 644 "${postgres_config_dir}/pg_hba.prod.conf"
fi

if [[ -d "${observability_dir}" ]]; then
  log "normalizing static observability config permissions: ${observability_dir}"
  normalize_readonly_tree "${observability_dir}"
fi

for generated_config_dir in \
  "${generated_observability_dir}/prometheus" \
  "${generated_observability_dir}/alertmanager"; do
  if [[ -d "${generated_config_dir}" ]]; then
    log "normalizing generated observability config permissions: ${generated_config_dir}"
    normalize_owned_tree "${generated_config_dir}" "65534:65534" 750 640
  fi
done

if [[ "${skip_casdoor}" != "true" && -d "${casdoor_compose_root}" ]]; then
  log "normalizing external Casdoor bind-mount permissions: ${casdoor_compose_root}"
  conf_dir="${casdoor_compose_root%/}/conf"
  logs_dir="${casdoor_compose_root%/}/logs"

  if [[ -d "${conf_dir}" ]]; then
    maybe_chown "${casdoor_owner}" "${conf_dir}"
    chmod_if_exists 750 "${conf_dir}"
    if [[ -f "${conf_dir}/app.conf" ]]; then
      maybe_chown "${casdoor_owner}" "${conf_dir}/app.conf"
      chmod_if_exists 640 "${conf_dir}/app.conf"
    fi
  fi

  if [[ -d "${logs_dir}" ]]; then
    normalize_owned_tree "${logs_dir}" "${casdoor_owner}" 750 640
  fi
fi

if [[ "${skip_openlist}" != "true" && -d "${openlist_data_dir}" ]]; then
  log "normalizing external OpenList runtime permissions: ${openlist_data_dir}"
  maybe_chown "${openlist_owner}" "${openlist_data_dir}"
  chmod_if_exists 750 "${openlist_data_dir}"

  openlist_config="${openlist_data_dir%/}/config.json"
  if [[ -e "${openlist_config}" && ! -s "${openlist_config}" ]]; then
    warn "OpenList config is empty; refusing to treat it as a permission-only repair: ${openlist_config}"
    validation_failed=true
  elif [[ -f "${openlist_config}" ]] && command -v jq >/dev/null 2>&1 \
    && ! jq -e . "${openlist_config}" >/dev/null 2>&1; then
    warn "OpenList config is not valid JSON; refusing to treat it as a permission-only repair: ${openlist_config}"
    validation_failed=true
  elif [[ -f "${openlist_config}" ]]; then
    maybe_chown "${openlist_owner}" "${openlist_config}"
    chmod_if_exists 640 "${openlist_config}"
  fi

  openlist_db="${openlist_data_dir%/}/data.db"
  if [[ -f "${openlist_db}" ]]; then
    maybe_chown "${openlist_owner}" "${openlist_db}"
    chmod_if_exists 640 "${openlist_db}"
  fi
fi

if [[ "${apply}" == "true" ]]; then
  log "runtime bind-mount permissions normalized"
else
  log "dry-run complete; rerun with --apply to write changes"
fi

if [[ "${validation_failed}" == "true" ]]; then
  exit 1
fi
