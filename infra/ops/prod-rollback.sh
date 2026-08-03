#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

require_cmd python3

# Reject an explicitly requested target before loading any secret-backed
# environment. The later validation remains necessary for targets derived from
# loaded deployment state.
if [[ -n "${ROLLBACK_TAG:-}" ]]; then
  require_safe_release_tag "${ROLLBACK_TAG}"
elif [[ -n "${TAG:-}" ]]; then
  require_safe_release_tag "${TAG}"
fi

requested_backend_image_ref="${ROLLBACK_BACKEND_IMAGE_REF:-}"
requested_frontend_image_ref="${ROLLBACK_FRONTEND_IMAGE_REF:-}"
requested_admin_image_ref="${ROLLBACK_ADMIN_IMAGE_REF:-}"

override_count=0
[[ -n "${requested_backend_image_ref}" ]] && ((override_count += 1))
[[ -n "${requested_frontend_image_ref}" ]] && ((override_count += 1))
[[ -n "${requested_admin_image_ref}" ]] && ((override_count += 1))
if ((override_count != 0 && override_count != 3)); then
  die "rollback image override requires backend, frontend, and admin digest references together"
fi
if ((override_count == 3)); then
  require_digest_image_ref ROLLBACK_BACKEND_IMAGE_REF "${requested_backend_image_ref}"
  require_digest_image_ref ROLLBACK_FRONTEND_IMAGE_REF "${requested_frontend_image_ref}"
  require_digest_image_ref ROLLBACK_ADMIN_IMAGE_REF "${requested_admin_image_ref}"
fi

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
require_safe_release_tag "${target_tag}"

release_file="${DEPLOY_STATE_DIR}/releases/${target_tag}.env"
if [[ -f "${release_file}" ]]; then
  allow_legacy_record_images=false
  ((override_count == 3)) && allow_legacy_record_images=true
  source_release_record_env_file \
    "${release_file}" \
    "${target_tag}" \
    "${allow_legacy_record_images}"
else
  BACKEND_IMAGE_REF="$(derive_tagged_image_ref "${BACKEND_IMAGE_REF:-}" "${target_tag}" || true)"
  FRONTEND_IMAGE_REF="$(derive_tagged_image_ref "${FRONTEND_IMAGE_REF:-}" "${target_tag}" || true)"
  ADMIN_IMAGE_REF="$(derive_tagged_image_ref "${ADMIN_IMAGE_REF:-}" "${target_tag}" || true)"
fi

if ((override_count == 3)); then
  BACKEND_IMAGE_REF="${requested_backend_image_ref}"
  FRONTEND_IMAGE_REF="${requested_frontend_image_ref}"
  ADMIN_IMAGE_REF="${requested_admin_image_ref}"
fi

[[ -n "${BACKEND_IMAGE_REF:-}" ]] || die "missing BACKEND_IMAGE_REF for rollback target ${target_tag}; deploy that release once with the new immutable-image flow or set it explicitly"
[[ -n "${FRONTEND_IMAGE_REF:-}" ]] || die "missing FRONTEND_IMAGE_REF for rollback target ${target_tag}; deploy that release once with the new immutable-image flow or set it explicitly"
[[ -n "${ADMIN_IMAGE_REF:-}" ]] || die "missing ADMIN_IMAGE_REF for rollback target ${target_tag}; deploy that release once with the new immutable-image flow or set it explicitly"

export TAG="${target_tag}"
export ROLLBACK_TAG="${target_tag}"
export DEPLOY_STATE_DIR
export BACKEND_IMAGE_REF FRONTEND_IMAGE_REF ADMIN_IMAGE_REF

validator=(
  python3
  "${REPO_ROOT}/infra/ops/validate-runtime-image-scan.py"
  --repo-root "${REPO_ROOT}"
  --policy-only
  --effective-environment production
)

current_policy_output=""
if current_policy_output="$("${validator[@]}" 2>&1)"; then
  log "runtime-image review windows are current; no rollback exception is needed"
else
  warn "current runtime-image review policy rejected the rollback target: ${current_policy_output}"
  [[ -f "${release_file}" ]] ||
    die "expired review windows may only be reused for a release previously deployed in this environment"

  if [[ -n "${ROLLBACK_REVIEW_REASON:-}" && -n "${ROLLBACK_REVIEW_REASON_B64:-}" ]]; then
    die "set only one of ROLLBACK_REVIEW_REASON or ROLLBACK_REVIEW_REASON_B64"
  fi
  if [[ -n "${ROLLBACK_REVIEW_REASON_B64:-}" ]]; then
    ROLLBACK_REVIEW_REASON="$(
      python3 - "${ROLLBACK_REVIEW_REASON_B64}" <<'PY'
import base64
import binascii
import sys

try:
    decoded = base64.b64decode(sys.argv[1], validate=True).decode("utf-8")
except (binascii.Error, UnicodeDecodeError) as exc:
    raise SystemExit(f"invalid ROLLBACK_REVIEW_REASON_B64: {exc}") from exc
print(decoded, end="")
PY
    )"
  fi

  export ROLLBACK_REVIEW_ACTOR="${ROLLBACK_REVIEW_ACTOR:-}"
  export ROLLBACK_REVIEW_REASON="${ROLLBACK_REVIEW_REASON:-}"
  ROLLBACK_REVIEW_AUDIT_ID="$(
    python3 - <<'PY'
import uuid

print(uuid.uuid4())
PY
  )"
  export ROLLBACK_REVIEW_AUDIT_ID
  export RUNTIME_IMAGE_ROLLBACK_RELEASE_RECORD="${release_file}"

  "${validator[@]}"

  audit_file="${DEPLOY_STATE_DIR}/rollback-review-exceptions.jsonl"
  export ROLLBACK_REVIEW_AUDIT_FILE="${audit_file}"
  export ROLLBACK_REVIEW_CURRENT_POLICY_ERROR="${current_policy_output}"
  export ROLLBACK_REVIEW_POLICY_FILE="${REPO_ROOT}/infra/security/runtime-images.json"
  python3 - <<'PY'
import hashlib
import json
import os
from datetime import datetime, timezone
from pathlib import Path

audit_path = Path(os.environ["ROLLBACK_REVIEW_AUDIT_FILE"])
policy_path = Path(os.environ["ROLLBACK_REVIEW_POLICY_FILE"])
record_path = Path(os.environ["RUNTIME_IMAGE_ROLLBACK_RELEASE_RECORD"])
audit_path.parent.mkdir(parents=True, exist_ok=True)

event = {
    "event": "runtime_image_review_window_exception_authorized",
    "audit_id": os.environ["ROLLBACK_REVIEW_AUDIT_ID"],
    "authorized_at": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
    "actor": os.environ["ROLLBACK_REVIEW_ACTOR"],
    "reason": os.environ["ROLLBACK_REVIEW_REASON"],
    "target_tag": os.environ["ROLLBACK_TAG"],
    "release_record": str(record_path),
    "policy_sha256": hashlib.sha256(policy_path.read_bytes()).hexdigest(),
    "current_policy_error": os.environ["ROLLBACK_REVIEW_CURRENT_POLICY_ERROR"],
    "backend_image_ref": os.environ["BACKEND_IMAGE_REF"],
    "frontend_image_ref": os.environ["FRONTEND_IMAGE_REF"],
    "admin_image_ref": os.environ["ADMIN_IMAGE_REF"],
}
payload = (json.dumps(event, ensure_ascii=False, sort_keys=True) + "\n").encode()
fd = os.open(audit_path, os.O_WRONLY | os.O_CREAT | os.O_APPEND, 0o600)
try:
    os.write(fd, payload)
    os.fsync(fd)
finally:
    os.close(fd)
os.chmod(audit_path, 0o600)
PY
  log "authorized audited rollback review-window exception ${ROLLBACK_REVIEW_AUDIT_ID}"
fi

log "rolling back production stack to tag ${target_tag}"
TAG="${target_tag}" "${SCRIPT_DIR}/prod-deploy.sh"
