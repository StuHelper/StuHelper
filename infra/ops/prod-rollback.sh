#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

load_env

current_tag="${TAG:-}"
target_tag="${ROLLBACK_TAG:-${TAG:-}}"
release_file=""

derive_tagged_image_ref() {
  local image_ref="${1:-}"
  local new_tag="${2:-}"
  python3 - "${image_ref}" "${new_tag}" <<'PY'
import sys

ref = sys.argv[1].strip()
tag = sys.argv[2].strip()
if not ref or not tag or "@sha256:" in ref:
    raise SystemExit(1)

image, sep, _ = ref.rpartition(":")
if not sep or "/" not in image:
    raise SystemExit(1)

print(f"{image}:{tag}")
PY
}

if [[ -z "${target_tag}" ]]; then
  target_tag="$(resolve_previous_release_tag "${current_tag:-}")" || die "unable to resolve previous release tag; set ROLLBACK_TAG manually"
fi

release_file="${DEPLOY_STATE_DIR}/releases/${target_tag}.env"
if [[ -f "${release_file}" ]]; then
  # shellcheck disable=SC1090
  source "${release_file}"
else
  BACKEND_IMAGE_REF="$(derive_tagged_image_ref "${BACKEND_IMAGE_REF:-}" "${target_tag}" || true)"
  FRONTEND_IMAGE_REF="$(derive_tagged_image_ref "${FRONTEND_IMAGE_REF:-}" "${target_tag}" || true)"
  ADMIN_IMAGE_REF="$(derive_tagged_image_ref "${ADMIN_IMAGE_REF:-}" "${target_tag}" || true)"
fi

[[ -n "${BACKEND_IMAGE_REF:-}" ]] || die "missing BACKEND_IMAGE_REF for rollback target ${target_tag}; deploy that release once with the new immutable-image flow or set it explicitly"
[[ -n "${FRONTEND_IMAGE_REF:-}" ]] || die "missing FRONTEND_IMAGE_REF for rollback target ${target_tag}; deploy that release once with the new immutable-image flow or set it explicitly"
[[ -n "${ADMIN_IMAGE_REF:-}" ]] || die "missing ADMIN_IMAGE_REF for rollback target ${target_tag}; deploy that release once with the new immutable-image flow or set it explicitly"

log "rolling back production stack to tag ${target_tag}"
TAG="${target_tag}" "${SCRIPT_DIR}/prod-deploy.sh"
