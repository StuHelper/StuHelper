#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

require_cmd git
require_cmd tar

OUTPUT_FILE="${1:-${REPO_ROOT}/infra/generated/deploy/stuhelper-deploy-bundle.tar.gz}"
OUTPUT_DIR="$(dirname "${OUTPUT_FILE}")"

if ! git -C "${REPO_ROOT}" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  die "deployment bundle must be built from a Git worktree"
fi
if [[ -n "$(git -C "${REPO_ROOT}" status --porcelain --untracked-files=all)" ]]; then
  die "deployment bundle requires a clean git worktree; commit or remove local changes before packaging"
fi

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
    --exclude='.env.generated.secrets' \
    --exclude='.env.prod.local' \
    --exclude='.env.prod.shared' \
    --exclude='.env.prod.secrets' \
    --exclude='.env.prod.secrets.local' \
    --exclude='.env.prod.generated' \
    --exclude='.env.prod.generated.secrets' \
    --exclude='.secrets' \
    --exclude='.secrets/*' \
    --exclude='.deploy' \
    --exclude='infra/generated/*' \
    --exclude='**/node_modules' \
    --exclude='**/dist' \
    --exclude='**/.turbo' \
    --exclude='**/.vite' \
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
