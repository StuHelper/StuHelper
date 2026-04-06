#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

require_cmd tar

OUTPUT_FILE="${1:-${REPO_ROOT}/infra/generated/deploy/stuhelper-deploy-bundle.tar.gz}"
OUTPUT_DIR="$(dirname "${OUTPUT_FILE}")"
mkdir -p "${OUTPUT_DIR}"

tmpfile="$(mktemp "${OUTPUT_DIR}/bundle.XXXXXX.tar.gz")"
trap 'rm -f "${tmpfile}"' EXIT

(
  cd "${REPO_ROOT}"
  tar \
    --exclude='.git' \
    --exclude='.claude' \
    --exclude='.run' \
    --exclude='.tools' \
    --exclude='.env' \
    --exclude='.env.generated' \
    --exclude='.env.prod.local' \
    --exclude='.env.prod.shared' \
    --exclude='.env.prod.secrets.local' \
    --exclude='.env.prod.generated' \
    --exclude='.deploy' \
    --exclude='infra/generated/*' \
    --exclude='clients/**/node_modules' \
    --exclude='clients/**/dist' \
    --exclude='clients/**/.turbo' \
    --exclude='clients/**/.vite' \
    --exclude='server/tmp' \
    --exclude='server/bin' \
    --exclude='**/.DS_Store' \
    -czf "${tmpfile}" \
    .
)

mv "${tmpfile}" "${OUTPUT_FILE}"
log "deployment bundle created: ${OUTPUT_FILE}"
