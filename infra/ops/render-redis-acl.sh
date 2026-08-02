#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

load_env

ACL_DIR="${REDIS_ACL_DIR:-${REPO_ROOT}/infra/generated/redis}"
ACL_FILE="${REDIS_ACL_FILE:-${ACL_DIR}/users.acl}"
umask 077
[[ ! -L "${ACL_DIR}" ]] || die "Redis ACL directory must not be a symlink: ${ACL_DIR}"
mkdir -p "${ACL_DIR}"
[[ ! -L "${ACL_FILE}" ]] || die "Redis ACL file must not be a symlink: ${ACL_FILE}"

[[ -n "${REDIS_PASSWORD:-}" ]] || die "REDIS_PASSWORD is required to render Redis ACL"
[[ -n "${REDIS_EXPORTER_PASSWORD:-}" ]] || die "REDIS_EXPORTER_PASSWORD is required to render Redis ACL"
REDIS_USERNAME="${REDIS_USERNAME:-stuhelper_app}"
REDIS_EXPORTER_USERNAME="${REDIS_EXPORTER_USERNAME:-stuhelper_metrics}"
REDIS_PROD_PARITY_MAINTENANCE_USERNAME="${REDIS_PROD_PARITY_MAINTENANCE_USERNAME:-}"
REDIS_PROD_PARITY_MAINTENANCE_PASSWORD="${REDIS_PROD_PARITY_MAINTENANCE_PASSWORD:-}"

validate_acl_username() {
  local key="$1"
  local value="$2"
  [[ "${value}" =~ ^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$ ]] ||
    die "${key} must match [A-Za-z0-9][A-Za-z0-9._-]{0,63}"
}

password_sha256() {
  printf '%s' "$1" | openssl dgst -sha256 -r | awk '{print $1}'
}

require_cmd openssl
require_cmd awk
validate_acl_username "REDIS_USERNAME" "${REDIS_USERNAME}"
validate_acl_username "REDIS_EXPORTER_USERNAME" "${REDIS_EXPORTER_USERNAME}"
[[ "${REDIS_USERNAME}" != "${REDIS_EXPORTER_USERNAME}" ]] ||
  die "REDIS_USERNAME and REDIS_EXPORTER_USERNAME must be different"
[[ "${REDIS_PASSWORD}" != "${REDIS_EXPORTER_PASSWORD}" ]] ||
  die "REDIS_PASSWORD and REDIS_EXPORTER_PASSWORD must be different"

maintenance_enabled=false
if [[ -n "${REDIS_PROD_PARITY_MAINTENANCE_USERNAME}" || -n "${REDIS_PROD_PARITY_MAINTENANCE_PASSWORD}" ]]; then
  [[ -n "${REDIS_PROD_PARITY_MAINTENANCE_USERNAME}" ]] ||
    die "REDIS_PROD_PARITY_MAINTENANCE_USERNAME is required when the prod-parity maintenance identity is enabled"
  [[ -n "${REDIS_PROD_PARITY_MAINTENANCE_PASSWORD}" ]] ||
    die "REDIS_PROD_PARITY_MAINTENANCE_PASSWORD is required when the prod-parity maintenance identity is enabled"
  [[ "${APP_ENV:-}" == "prod-parity" ]] ||
    die "the Redis maintenance identity is only allowed in APP_ENV=prod-parity"
  validate_acl_username "REDIS_PROD_PARITY_MAINTENANCE_USERNAME" "${REDIS_PROD_PARITY_MAINTENANCE_USERNAME}"
  [[ "${REDIS_PROD_PARITY_MAINTENANCE_USERNAME}" != "${REDIS_USERNAME}" ]] ||
    die "REDIS_PROD_PARITY_MAINTENANCE_USERNAME must differ from REDIS_USERNAME"
  [[ "${REDIS_PROD_PARITY_MAINTENANCE_USERNAME}" != "${REDIS_EXPORTER_USERNAME}" ]] ||
    die "REDIS_PROD_PARITY_MAINTENANCE_USERNAME must differ from REDIS_EXPORTER_USERNAME"
  [[ "${REDIS_PROD_PARITY_MAINTENANCE_PASSWORD}" != "${REDIS_PASSWORD}" ]] ||
    die "REDIS_PROD_PARITY_MAINTENANCE_PASSWORD must differ from REDIS_PASSWORD"
  [[ "${REDIS_PROD_PARITY_MAINTENANCE_PASSWORD}" != "${REDIS_EXPORTER_PASSWORD}" ]] ||
    die "REDIS_PROD_PARITY_MAINTENANCE_PASSWORD must differ from REDIS_EXPORTER_PASSWORD"
  maintenance_enabled=true
fi

app_password_hash="$(password_sha256 "${REDIS_PASSWORD}")"
exporter_password_hash="$(password_sha256 "${REDIS_EXPORTER_PASSWORD}")"
[[ "${app_password_hash}" =~ ^[0-9a-f]{64}$ ]] || die "failed to hash REDIS_PASSWORD"
[[ "${exporter_password_hash}" =~ ^[0-9a-f]{64}$ ]] || die "failed to hash REDIS_EXPORTER_PASSWORD"

maintenance_acl_line=""
if [[ "${maintenance_enabled}" == "true" ]]; then
  maintenance_password_hash="$(password_sha256 "${REDIS_PROD_PARITY_MAINTENANCE_PASSWORD}")"
  [[ "${maintenance_password_hash}" =~ ^[0-9a-f]{64}$ ]] ||
    die "failed to hash REDIS_PROD_PARITY_MAINTENANCE_PASSWORD"
  maintenance_acl_line="user ${REDIS_PROD_PARITY_MAINTENANCE_USERNAME} reset on #${maintenance_password_hash} ~course:* ~review:* ~cache:version:course* ~cache:version:review* ~rl:* resetchannels -@all +auth +select +ping +scan +del +client|setname"
fi

acl_tmp="$(mktemp "${ACL_DIR}/.users.acl.XXXXXX")"
trap 'rm -f "${acl_tmp}"' EXIT
cat >"${acl_tmp}" <<ACL
user default off
user ${REDIS_USERNAME} reset on #${app_password_hash} ~* &notify:* &notification:v2:* -@all +hello +auth +select +ping +get +set +getdel +del +exists +incr +decr +expire +pexpire +sadd +srem +smembers +zadd +zcard +zremrangebyscore +publish +subscribe +unsubscribe +psubscribe +punsubscribe +eval +evalsha +client|setinfo +client|setname
user ${REDIS_EXPORTER_USERNAME} reset on #${exporter_password_hash} resetkeys resetchannels -@all +auth +select +ping +client|setname +info +config|get +slowlog|len +slowlog|get +latency|latest +latency|histogram
${maintenance_acl_line}
ACL
chmod 600 "${acl_tmp}"
mv -f "${acl_tmp}" "${ACL_FILE}"
trap - EXIT

log "rendered Redis ACL file at ${ACL_FILE}"
