#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

load_env

ACL_DIR="${REDIS_ACL_DIR:-${REPO_ROOT}/infra/generated/redis}"
ACL_FILE="${REDIS_ACL_FILE:-${ACL_DIR}/users.acl}"
mkdir -p "${ACL_DIR}"

[[ -n "${REDIS_PASSWORD:-}" ]] || die "REDIS_PASSWORD is required to render Redis ACL"
REDIS_USERNAME="${REDIS_USERNAME:-stuhelper_app}"
REDIS_EXPORTER_USERNAME="${REDIS_EXPORTER_USERNAME:-stuhelper_metrics}"

cat >"${ACL_FILE}" <<ACL
user default off
user ${REDIS_USERNAME} on >${REDIS_PASSWORD} ~* +@all
user ${REDIS_EXPORTER_USERNAME} on >${REDIS_PASSWORD} ~* +@all
ACL
chmod 600 "${ACL_FILE}"

log "rendered Redis ACL file at ${ACL_FILE}"
