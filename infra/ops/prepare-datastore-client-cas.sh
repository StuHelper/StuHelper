#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

require_cmd openssl

load_env

prepare_ca_bundle() {
  local label="$1"
  local source_file="$2"
  local destination_dir="$3"
  local destination_file="${destination_dir}/ca.crt"
  local temporary_file
  local unexpected_entry

  [[ ! -L "${destination_dir}" ]] ||
    die "${label} client CA directory must not be a symlink: ${destination_dir}"
  mkdir -p "${destination_dir}"
  [[ -d "${destination_dir}" && -w "${destination_dir}" ]] ||
    die "${label} client CA directory is not writable: ${destination_dir}"
  chmod 755 "${destination_dir}"
  unexpected_entry="$(find "${destination_dir}" -mindepth 1 -maxdepth 1 ! -name ca.crt -print -quit)"
  [[ -z "${unexpected_entry}" ]] ||
    die "${label} client CA directory contains an unexpected entry: ${unexpected_entry}"
  [[ ! -L "${destination_file}" ]] ||
    die "${label} client CA destination must not be a symlink: ${destination_file}"
  [[ ! -d "${destination_file}" ]] ||
    die "${label} client CA destination is unexpectedly a directory: ${destination_file}"

  if [[ -z "${source_file}" ]]; then
    rm -f -- "${destination_file}"
    log "${label} TLS is disabled; prepared an empty client CA mount"
    return
  fi

  [[ -f "${source_file}" && -r "${source_file}" ]] ||
    die "${label} client CA source must be a readable regular file: ${source_file}"
  if grep -Eq -- '-----BEGIN ([A-Z0-9 ]+ )?PRIVATE KEY-----' "${source_file}"; then
    die "${label} client CA source must contain certificates only, not a private key"
  fi
  if ! openssl crl2pkcs7 -nocrl -certfile "${source_file}" |
    openssl pkcs7 -print_certs -noout >/dev/null 2>&1; then
    die "${label} client CA source is not a valid PEM certificate bundle"
  fi

  temporary_file="$(mktemp "${destination_dir}/.ca.crt.XXXXXX")"
  cp -- "${source_file}" "${temporary_file}"
  chmod 644 "${temporary_file}"
  mv -f -- "${temporary_file}" "${destination_file}"

  cmp -s "${source_file}" "${destination_file}" ||
    die "${label} client CA copy verification failed"
  log "prepared ${label} client CA bundle: ${destination_file}"
}

postgres_client_ca_dir="${POSTGRES_CLIENT_CA_DIR:-${REPO_ROOT}/infra/generated/postgres-client-ca}"
redis_client_ca_dir="${REDIS_CLIENT_CA_DIR:-${REPO_ROOT}/infra/generated/redis-client-ca}"
oracle_student_source_client_ca_dir="${EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_CA_DIR:-${REPO_ROOT}/infra/generated/external-student-source-client-ca}"

postgres_ca_source=""
if [[ "${POSTGRES_INTERNAL_SSL_MODE:-disable}" != "disable" ]]; then
  if [[ "${EXTERNAL_POSTGRES_ENABLED:-false}" == "true" ]]; then
    postgres_ca_source="${POSTGRES_CLIENT_CA_HOST_PATH:-}"
    [[ -n "${postgres_ca_source}" ]] ||
      die "POSTGRES_CLIENT_CA_HOST_PATH is required for external PostgreSQL TLS"
  else
    postgres_ca_source="${POSTGRES_CLIENT_CA_HOST_PATH:-${POSTGRES_TLS_DIR:-${REPO_ROOT}/infra/generated/postgres}/ca.crt}"
  fi
fi
prepare_ca_bundle "PostgreSQL" "${postgres_ca_source}" "${postgres_client_ca_dir}"

redis_ca_source=""
if [[ "${REDIS_TLS_ENABLED:-false}" == "true" ]]; then
  redis_ca_source="${REDIS_CLIENT_CA_HOST_PATH:-${REDIS_TLS_DIR:-${REPO_ROOT}/infra/generated/redis}/ca.crt}"
fi
prepare_ca_bundle "Redis" "${redis_ca_source}" "${redis_client_ca_dir}"

oracle_student_source_ca=""
if [[ "${EXTERNAL_STUDENT_SOURCE_ENABLED:-false}" == "true" ]]; then
  [[ "${EXTERNAL_STUDENT_SOURCE_PROVIDER:-}" == "oracle" ]] ||
    die "EXTERNAL_STUDENT_SOURCE_PROVIDER must be oracle when the external student source is enabled"
  case "${EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_MODE:-verify-full}" in
    verify-full)
      oracle_student_source_ca="${EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_CA_HOST_PATH:-}"
      [[ -n "${oracle_student_source_ca}" ]] ||
        die "EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_CA_HOST_PATH is required for verified Oracle TLS"
      ;;
    disable)
      if [[ "${APP_ENV:-development}" == "production" || "${APP_ENV:-development}" == "prod-parity" ]]; then
        die "EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_MODE must be verify-full in production"
      fi
      ;;
    *)
      die "EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_MODE must be verify-full or disable"
      ;;
  esac
fi
prepare_ca_bundle \
  "Oracle student source" \
  "${oracle_student_source_ca}" \
  "${oracle_student_source_client_ca_dir}"
