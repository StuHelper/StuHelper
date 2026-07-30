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

# The bundle is exactly the tracked source at HEAD. Building it from the Git
# index rather than the working tree makes it impossible for an ignored file
# (local secrets, build output, generated env) to reach a deployment target,
# and the clean-worktree gate above guarantees HEAD matches what was tested.
git -C "${REPO_ROOT}" archive --format=tar.gz --output="${tmpfile}" HEAD

# Deployment and rollback both call ensure_env_file(), which fails when the env
# templates are absent. Assert their presence rather than trusting the packaging
# path to keep including them.
bundled_env_files="$(tar -tzf "${tmpfile}" | sed 's|^\./||' | grep -E '^\.env' || true)"
for required in .env.example .env.prod.example; do
  if ! grep -qxF "${required}" <<<"${bundled_env_files}"; then
    die "deployment bundle is missing required env template: ${required}"
  fi
done
unexpected_env_files="$(grep -vxE '\.env\.example|\.env\.prod\.example' <<<"${bundled_env_files}" | grep -v '^$' || true)"
if [[ -n "${unexpected_env_files}" ]]; then
  die "deployment bundle contains unexpected env files: ${unexpected_env_files//$'\n'/ }"
fi

mv "${tmpfile}" "${OUTPUT_FILE}"
log "deployment bundle created: ${OUTPUT_FILE}"
