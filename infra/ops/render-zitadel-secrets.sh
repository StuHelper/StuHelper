#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

load_env

[[ -n "${ZITADEL_MASTERKEY:-}" ]] || die "ZITADEL_MASTERKEY is required to render Zitadel runtime secrets"

ZITADEL_SECRET_DIR="${ZITADEL_SECRET_DIR:-${GENERATED_ZITADEL_DIR}}"
MASTERKEY_FILE="${ZITADEL_MASTERKEY_PATH:-${ZITADEL_SECRET_DIR}/masterkey}"

mkdir -p "${ZITADEL_SECRET_DIR}"
printf '%s' "${ZITADEL_MASTERKEY}" > "${MASTERKEY_FILE}"
chmod 600 "${MASTERKEY_FILE}"

log "rendered Zitadel runtime secrets to ${ZITADEL_SECRET_DIR}"
