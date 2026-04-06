#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

load_env

current_tag="${TAG:-}"
target_tag="${ROLLBACK_TAG:-${TAG:-}}"

if [[ -z "${target_tag}" ]]; then
  target_tag="$(resolve_previous_release_tag "${current_tag:-}")" || die "unable to resolve previous release tag; set ROLLBACK_TAG manually"
fi

log "rolling back production stack to tag ${target_tag}"
TAG="${target_tag}" SKIP_BUILD=true "${SCRIPT_DIR}/prod-deploy.sh"

