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

cat >"${ACL_FILE}" <<ACL
user default on >${REDIS_PASSWORD} ~* +@all
ACL
chmod 600 "${ACL_FILE}"

log "rendered Redis ACL file at ${ACL_FILE}"
