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
  --skip-casdoor             Do not touch the external Casdoor Compose root.
  --apply                    Write permissions. Without this flag, dry-run only.
  -h, --help                 Show this help.

Environment:
  POSTGRES_TLS_SERVER_KEY_OWNER  Host owner for PostgreSQL server.key when run
                                 as root. Default: 70:70 for postgres:*-alpine.
  CASDOOR_CONTAINER_OWNER        Host owner for Casdoor conf/logs when run as
                                 root. Default: 1000:1000 for casbin/casdoor.

The script does not read or print secret file contents.
USAGE
}

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

source_dir="${BAOTA_SOURCE_DIR:-${REPO_ROOT}}"
casdoor_compose_root="${CASDOOR_COMPOSE_ROOT:-/www/server/panel/data/compose/casdoor}"
skip_casdoor=false
apply=false

postgres_key_owner="${POSTGRES_TLS_SERVER_KEY_OWNER:-70:70}"
casdoor_owner="${CASDOOR_CONTAINER_OWNER:-1000:1000}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --source-dir) source_dir="${2:-}"; shift 2 ;;
    --casdoor-compose-root) casdoor_compose_root="${2:-}"; shift 2 ;;
    --skip-casdoor) skip_casdoor=true; shift ;;
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

maybe_chown_recursive() {
  local owner="$1"
  shift
  [[ -n "${owner}" ]] || return 0
  if [[ "$(id -u)" != "0" ]]; then
    log "skip recursive chown ${owner}; not running as root"
    return 0
  fi
  run_or_print chown -R "${owner}" "$@"
}

chmod_if_exists() {
  local mode="$1"
  local path="$2"
  [[ -e "${path}" ]] || return 0
  run_or_print chmod "${mode}" "${path}"
}

postgres_tls_dir="${source_dir}/infra/generated/postgres"
redis_tls_dir="${source_dir}/infra/generated/redis"

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
  chmod_if_exists 644 "${redis_tls_dir}/server.key"
  chmod_if_exists 644 "${redis_tls_dir}/users.acl"
fi

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
    maybe_chown_recursive "${casdoor_owner}" "${logs_dir}"
    chmod_if_exists 750 "${logs_dir}"
  fi
fi

if [[ "${apply}" == "true" ]]; then
  log "runtime bind-mount permissions normalized"
else
  log "dry-run complete; rerun with --apply to write changes"
fi
