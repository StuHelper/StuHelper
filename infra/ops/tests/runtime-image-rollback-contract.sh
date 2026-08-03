#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
ROLLBACK_SCRIPT="${REPO_ROOT}/infra/ops/prod-rollback.sh"

fail() {
  printf '[runtime-image-rollback-contract][error] %s\n' "$*" >&2
  exit 1
}

bash -n "${ROLLBACK_SCRIPT}"

fixture_root="$(mktemp -d)"
trap 'rm -rf "${fixture_root}"' EXIT
fixture_repo="${fixture_root}/repo"
fixture_state="${fixture_root}/deploy-state"
mkdir -p \
  "${fixture_repo}/infra/ops/lib" \
  "${fixture_repo}/infra/security" \
  "${fixture_state}/releases"

cp "${ROLLBACK_SCRIPT}" "${fixture_repo}/infra/ops/prod-rollback.sh"
chmod +x "${fixture_repo}/infra/ops/prod-rollback.sh"

cat >"${fixture_repo}/infra/ops/lib/common.sh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

COMMON_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${COMMON_LIB_DIR}/../../.." && pwd)"
DEPLOY_STATE_DIR="${DEPLOY_STATE_DIR:-${REPO_ROOT}/.deploy}"

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || exit 90
}
die() {
  printf '[fixture][error] %s\n' "$*" >&2
  exit 1
}
log() {
  printf '[fixture] %s\n' "$*"
}
warn() {
  printf '[fixture][warn] %s\n' "$*" >&2
}
require_safe_release_tag() {
  [[ "${1:-}" =~ ^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$ ]] || die "release tag must be 1-128 characters"
}
require_digest_image_ref() {
  local key="$1"
  local value="${2:-}"
  [[ "${value}" =~ ^[^[:space:]@]+@sha256:[0-9a-f]{64}$ ]] ||
    die "${key} must be a complete image@sha256 digest reference"
}
acquire_production_deploy_lock() { :; }
migrate_legacy_release_state_permissions() {
  if [[ -d "${DEPLOY_STATE_DIR}/releases" ]]; then
    find "${DEPLOY_STATE_DIR}/releases" -maxdepth 1 -type f -name '*.env' -exec chmod 0600 {} +
  fi
  [[ ! -f "${DEPLOY_STATE_DIR}/current-release.env" ]] || chmod 0600 "${DEPLOY_STATE_DIR}/current-release.env"
  [[ ! -f "${DEPLOY_STATE_DIR}/releases.log" ]] || chmod 0600 "${DEPLOY_STATE_DIR}/releases.log"
}
migrate_explicit_legacy_release_identity() {
  local tag="$1"
  local backend_ref="$2"
  local frontend_ref="$3"
  local admin_ref="$4"
  local actor="$5"
  local reason="$6"
  local file="${DEPLOY_STATE_DIR}/releases/${tag}.env"
  local deployed_at
  if grep -q '@sha256:' "${file}"; then
    return 0
  fi
  [[ -n "${actor}" && ${#reason} -ge 12 ]] || die "legacy rollback migration audit context is required"
  deployed_at="$(sed -n 's/^DEPLOYED_AT=//p' "${file}")"
  cat >"${file}" <<RECORD
TAG=${tag}
DEPLOYED_AT=${deployed_at}
BACKEND_IMAGE_REF=${backend_ref}
FRONTEND_IMAGE_REF=${frontend_ref}
ADMIN_IMAGE_REF=${admin_ref}
RECORD
  chmod 0600 "${file}"
  if [[ -n "${ROLLBACK_MIGRATION_OBSERVED_FILE:-}" ]]; then
    printf '%s\n' "${tag}" >"${ROLLBACK_MIGRATION_OBSERVED_FILE}"
  fi
}
load_env() {
  if [[ -n "${ROLLBACK_LOAD_ENV_OBSERVED_FILE:-}" ]]; then
    printf 'loaded\n' >"${ROLLBACK_LOAD_ENV_OBSERVED_FILE}"
  fi
  export TAG=current-release
  export BACKEND_IMAGE_REF=ghcr.io/stuhelper/backend@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
  export FRONTEND_IMAGE_REF=ghcr.io/stuhelper/frontend@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
  export ADMIN_IMAGE_REF=ghcr.io/stuhelper/admin@sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc
}
source_release_record_env_file() {
  local file="$1"
  local expected_tag="$2"
  local allow_legacy_image_refs="${3:-false}"
  local key value
  local -A seen=()
  unset TAG DEPLOYED_AT BACKEND_IMAGE_REF FRONTEND_IMAGE_REF ADMIN_IMAGE_REF
  while IFS='=' read -r key value; do
    [[ -n "${key}" ]] || continue
    [[ -z "${seen[${key}]:-}" ]] || die "duplicate release record key ${key}"
    seen["${key}"]=true
    printf -v "${key}" '%s' "${value}"
    export "${key}"
  done <"${file}"
  for key in TAG DEPLOYED_AT BACKEND_IMAGE_REF FRONTEND_IMAGE_REF ADMIN_IMAGE_REF; do
    [[ -n "${!key:-}" ]] || die "missing release record key ${key}"
  done
  [[ "${TAG}" == "${expected_tag}" ]] || die "release record TAG does not match rollback target"
  if [[ "${allow_legacy_image_refs}" != "true" ]]; then
    [[ "${BACKEND_IMAGE_REF}" =~ @sha256:[0-9a-f]{64}$ ]] || die "BACKEND_IMAGE_REF must be a complete image@sha256 digest reference"
    [[ "${FRONTEND_IMAGE_REF}" =~ @sha256:[0-9a-f]{64}$ ]] || die "FRONTEND_IMAGE_REF must be a complete image@sha256 digest reference"
    [[ "${ADMIN_IMAGE_REF}" =~ @sha256:[0-9a-f]{64}$ ]] || die "ADMIN_IMAGE_REF must be a complete image@sha256 digest reference"
  fi
}
resolve_previous_release_tag() {
  return 1
}
EOF

grep -qF '"${allow_legacy_record_images}"' "${ROLLBACK_SCRIPT}" ||
  fail "rollback release records must use their exact target-bound loader"
if grep -qF 'source "${release_file}"' "${ROLLBACK_SCRIPT}"; then
  fail "rollback release records must not be raw-sourced"
fi
python3 - "${ROLLBACK_SCRIPT}" <<'PY'
import sys
from pathlib import Path

source = Path(sys.argv[1]).read_text(encoding="utf-8")
lock = source.index("acquire_production_deploy_lock")
load = source.index("load_env", lock)
permission_migration = source.index("migrate_legacy_release_state_permissions", load)
record_read = source.index("source_release_record_env_file", permission_migration)
identity_migration = source.index("migrate_explicit_legacy_release_identity", record_read)
deploy = source.index('"${SCRIPT_DIR}/prod-deploy.sh"', identity_migration)
if not lock < load < permission_migration < record_read < identity_migration < deploy:
    raise SystemExit("rollback lock and legacy migration ordering is unsafe")
PY

cat >"${fixture_repo}/infra/ops/validate-runtime-image-scan.py" <<'PY'
import os
import sys
from pathlib import Path

record = os.environ.get("RUNTIME_IMAGE_ROLLBACK_RELEASE_RECORD", "")
if not record:
    if os.environ.get("VALIDATOR_CURRENT_OK") == "true":
        print("[runtime-image-policy] current fixture accepted")
        raise SystemExit(0)
    print("[runtime-image-scan][error] review window expired", file=sys.stderr)
    raise SystemExit(1)
if not Path(record).is_file():
    print("[runtime-image-scan][error] missing release record", file=sys.stderr)
    raise SystemExit(2)
if not os.environ.get("ROLLBACK_REVIEW_AUDIT_ID"):
    print("[runtime-image-scan][error] missing audit id", file=sys.stderr)
    raise SystemExit(3)
print("[runtime-image-policy] audited rollback fixture accepted")
PY

cat >"${fixture_repo}/infra/ops/prod-deploy.sh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
if [[ -n "${RUNTIME_IMAGE_ROLLBACK_RELEASE_RECORD:-}" ]]; then
  : "${ROLLBACK_REVIEW_AUDIT_ID:?missing rollback audit id}"
  printf '%s\n' "${ROLLBACK_REVIEW_AUDIT_ID}" >"${ROLLBACK_DEPLOY_OBSERVED_FILE}"
else
  printf 'none\n' >"${ROLLBACK_DEPLOY_OBSERVED_FILE}"
fi
if [[ -n "${ROLLBACK_DEPLOY_IMAGE_REFS_FILE:-}" ]]; then
  printf '%s\n%s\n%s\n' \
    "${BACKEND_IMAGE_REF}" \
    "${FRONTEND_IMAGE_REF}" \
    "${ADMIN_IMAGE_REF}" \
    >"${ROLLBACK_DEPLOY_IMAGE_REFS_FILE}"
fi
EOF
chmod +x "${fixture_repo}/infra/ops/prod-deploy.sh"
printf '{}\n' >"${fixture_repo}/infra/security/runtime-images.json"

target_tag="0123456789abcdef0123456789abcdef01234567"
backend_ref="ghcr.io/stuhelper/backend@sha256:1111111111111111111111111111111111111111111111111111111111111111"
frontend_ref="ghcr.io/stuhelper/frontend@sha256:2222222222222222222222222222222222222222222222222222222222222222"
admin_ref="ghcr.io/stuhelper/admin@sha256:3333333333333333333333333333333333333333333333333333333333333333"
unsafe_observed_file="${fixture_root}/unsafe-deploy-observed"
unsafe_load_env_observed_file="${fixture_root}/unsafe-load-env-observed"
if VALIDATOR_CURRENT_OK=true \
  DEPLOY_STATE_DIR="${fixture_state}" \
  ROLLBACK_TAG='../escape' \
  ROLLBACK_LOAD_ENV_OBSERVED_FILE="${unsafe_load_env_observed_file}" \
  ROLLBACK_DEPLOY_OBSERVED_FILE="${unsafe_observed_file}" \
  "${fixture_repo}/infra/ops/prod-rollback.sh" >"${fixture_root}/unsafe.out" 2>"${fixture_root}/unsafe.err"; then
  fail "rollback accepted a path-traversing target tag"
fi
grep -q 'release tag must be 1-128 characters' "${fixture_root}/unsafe.err" ||
  fail "rollback did not report the canonical release-tag constraint"
[[ ! -e "${unsafe_observed_file}" ]] ||
  fail "unsafe rollback target reached prod-deploy"
[[ ! -e "${unsafe_load_env_observed_file}" ]] ||
  fail "unsafe rollback target loaded secret-backed environment state"

unsafe_inherited_tag_observed_file="${fixture_root}/unsafe-inherited-tag-load-env-observed"
if VALIDATOR_CURRENT_OK=true \
  DEPLOY_STATE_DIR="${fixture_state}" \
  TAG='../inherited-escape' \
  ROLLBACK_LOAD_ENV_OBSERVED_FILE="${unsafe_inherited_tag_observed_file}" \
  ROLLBACK_DEPLOY_OBSERVED_FILE="${fixture_root}/unsafe-inherited-tag-deploy-observed" \
  "${fixture_repo}/infra/ops/prod-rollback.sh" \
  >"${fixture_root}/unsafe-inherited-tag.out" 2>"${fixture_root}/unsafe-inherited-tag.err"; then
  fail "rollback accepted a path-traversing inherited TAG"
fi
grep -q 'release tag must be 1-128 characters' "${fixture_root}/unsafe-inherited-tag.err" ||
  fail "rollback did not report the canonical inherited TAG constraint"
[[ ! -e "${unsafe_inherited_tag_observed_file}" ]] ||
  fail "unsafe inherited TAG loaded secret-backed environment state"

partial_override_load_observed="${fixture_root}/partial-override-load-observed"
if VALIDATOR_CURRENT_OK=true \
  DEPLOY_STATE_DIR="${fixture_state}" \
  ROLLBACK_TAG=contract-partial \
  ROLLBACK_BACKEND_IMAGE_REF="${backend_ref}" \
  ROLLBACK_LOAD_ENV_OBSERVED_FILE="${partial_override_load_observed}" \
  ROLLBACK_DEPLOY_OBSERVED_FILE="${fixture_root}/partial-override-deploy-observed" \
  "${fixture_repo}/infra/ops/prod-rollback.sh" \
  >"${fixture_root}/partial-override.out" 2>"${fixture_root}/partial-override.err"; then
  fail "rollback accepted an incomplete image override tuple"
fi
grep -q 'rollback image override requires backend, frontend, and admin digest references together' \
  "${fixture_root}/partial-override.err" ||
  fail "partial rollback override did not report the complete-tuple requirement"
[[ ! -e "${partial_override_load_observed}" ]] ||
  fail "partial rollback override loaded secret-backed environment state"

cat >"${fixture_state}/releases/${target_tag}.env" <<EOF
TAG=${target_tag}
DEPLOYED_AT=2026-07-30T12:00:00Z
BACKEND_IMAGE_REF=${backend_ref}
FRONTEND_IMAGE_REF=${frontend_ref}
ADMIN_IMAGE_REF=${admin_ref}
EOF

truncated_tag="1111111111111111111111111111111111111111"
cat >"${fixture_state}/releases/${truncated_tag}.env" <<EOF
TAG=${truncated_tag}
DEPLOYED_AT=2026-07-30T12:00:00Z
BACKEND_IMAGE_REF=${backend_ref}
FRONTEND_IMAGE_REF=${frontend_ref}
EOF
truncated_observed_file="${fixture_root}/truncated-deploy-observed"
if VALIDATOR_CURRENT_OK=true \
  DEPLOY_STATE_DIR="${fixture_state}" \
  ROLLBACK_TAG="${truncated_tag}" \
  ROLLBACK_DEPLOY_OBSERVED_FILE="${truncated_observed_file}" \
  "${fixture_repo}/infra/ops/prod-rollback.sh" >"${fixture_root}/truncated.out" 2>"${fixture_root}/truncated.err"; then
  fail "rollback accepted a release record missing one image reference"
fi
grep -q 'missing release record key ADMIN_IMAGE_REF' "${fixture_root}/truncated.err" ||
  fail "truncated rollback record failure did not identify the missing image reference"
[[ ! -e "${truncated_observed_file}" ]] ||
  fail "truncated rollback record reached prod-deploy"

mismatched_tag="2222222222222222222222222222222222222222"
cat >"${fixture_state}/releases/${mismatched_tag}.env" <<EOF
TAG=${target_tag}
DEPLOYED_AT=2026-07-30T12:00:00Z
BACKEND_IMAGE_REF=${backend_ref}
FRONTEND_IMAGE_REF=${frontend_ref}
ADMIN_IMAGE_REF=${admin_ref}
EOF
if VALIDATOR_CURRENT_OK=true \
  DEPLOY_STATE_DIR="${fixture_state}" \
  ROLLBACK_TAG="${mismatched_tag}" \
  ROLLBACK_DEPLOY_OBSERVED_FILE="${fixture_root}/mismatched-deploy-observed" \
  "${fixture_repo}/infra/ops/prod-rollback.sh" >"${fixture_root}/mismatched.out" 2>"${fixture_root}/mismatched.err"; then
  fail "rollback accepted a release record for a different tag"
fi
grep -q 'TAG does not match rollback target' "${fixture_root}/mismatched.err" ||
  fail "rollback record tag mismatch did not report the target binding"

legacy_tag="legacy-release"
cat >"${fixture_state}/releases/${legacy_tag}.env" <<EOF
TAG=${legacy_tag}
DEPLOYED_AT=2026-07-30T12:00:00Z
BACKEND_IMAGE_REF=ghcr.io/stuhelper/backend:legacy
FRONTEND_IMAGE_REF=ghcr.io/stuhelper/frontend:legacy
ADMIN_IMAGE_REF=ghcr.io/stuhelper/admin:legacy
EOF
if VALIDATOR_CURRENT_OK=true \
  DEPLOY_STATE_DIR="${fixture_state}" \
  ROLLBACK_TAG="${legacy_tag}" \
  ROLLBACK_DEPLOY_OBSERVED_FILE="${fixture_root}/legacy-without-overrides-observed" \
  "${fixture_repo}/infra/ops/prod-rollback.sh" \
  >"${fixture_root}/legacy-without-overrides.out" 2>"${fixture_root}/legacy-without-overrides.err"; then
  fail "rollback accepted legacy mutable image refs without a verified override tuple"
fi
grep -q 'BACKEND_IMAGE_REF must be a complete image@sha256 digest reference' \
  "${fixture_root}/legacy-without-overrides.err" ||
  fail "legacy rollback without overrides did not retain the immutable-image gate"

if VALIDATOR_CURRENT_OK=true \
  DEPLOY_STATE_DIR="${fixture_state}" \
  ROLLBACK_TAG="${legacy_tag}" \
  ROLLBACK_BACKEND_IMAGE_REF="${backend_ref}" \
  ROLLBACK_FRONTEND_IMAGE_REF="${frontend_ref}" \
  ROLLBACK_ADMIN_IMAGE_REF="${admin_ref}" \
  ROLLBACK_DEPLOY_OBSERVED_FILE="${fixture_root}/legacy-without-audit-observed" \
  "${fixture_repo}/infra/ops/prod-rollback.sh" \
  >"${fixture_root}/legacy-without-audit.out" 2>"${fixture_root}/legacy-without-audit.err"; then
  fail "rollback migrated a legacy release without operator audit context"
fi
grep -q 'legacy rollback migration audit context is required' \
  "${fixture_root}/legacy-without-audit.err" ||
  fail "legacy migration without audit context did not fail explicitly"
[[ ! -e "${fixture_root}/legacy-without-audit-observed" ]] ||
  fail "legacy migration without audit context reached prod-deploy"

legacy_refs_observed="${fixture_root}/legacy-refs-observed"
legacy_migration_observed="${fixture_root}/legacy-migration-observed"
VALIDATOR_CURRENT_OK=true \
DEPLOY_STATE_DIR="${fixture_state}" \
ROLLBACK_TAG="${legacy_tag}" \
ROLLBACK_BACKEND_IMAGE_REF="${backend_ref}" \
ROLLBACK_FRONTEND_IMAGE_REF="${frontend_ref}" \
ROLLBACK_ADMIN_IMAGE_REF="${admin_ref}" \
ROLLBACK_REVIEW_ACTOR=contract-operator \
ROLLBACK_REVIEW_REASON='restore a provenance-verified legacy release' \
ROLLBACK_MIGRATION_OBSERVED_FILE="${legacy_migration_observed}" \
ROLLBACK_DEPLOY_OBSERVED_FILE="${fixture_root}/legacy-deploy-observed" \
ROLLBACK_DEPLOY_IMAGE_REFS_FILE="${legacy_refs_observed}" \
  "${fixture_repo}/infra/ops/prod-rollback.sh" >/dev/null
cat >"${fixture_root}/expected-legacy-refs" <<EOF
${backend_ref}
${frontend_ref}
${admin_ref}
EOF
cmp -s "${fixture_root}/expected-legacy-refs" "${legacy_refs_observed}" ||
  fail "verified rollback overrides did not replace all legacy image refs before deployment"
[[ "$(cat "${legacy_migration_observed}")" == "${legacy_tag}" ]] ||
  fail "verified rollback overrides did not migrate the legacy per-tag record before deployment"
grep -qF "BACKEND_IMAGE_REF=${backend_ref}" "${fixture_state}/releases/${legacy_tag}.env" ||
  fail "legacy rollback migration did not persist the verified backend digest"

reason="incident rollback to a previously successful immutable release"
reason_b64="$(python3 - "${reason}" <<'PY'
import base64
import sys

print(base64.b64encode(sys.argv[1].encode()).decode())
PY
)"
observed_file="${fixture_root}/deploy-observed"

current_observed_file="${fixture_root}/current-deploy-observed"
VALIDATOR_CURRENT_OK=true \
DEPLOY_STATE_DIR="${fixture_state}" \
ROLLBACK_TAG="${target_tag}" \
ROLLBACK_BACKEND_IMAGE_REF="${backend_ref}" \
ROLLBACK_FRONTEND_IMAGE_REF="${frontend_ref}" \
ROLLBACK_ADMIN_IMAGE_REF="${admin_ref}" \
ROLLBACK_DEPLOY_OBSERVED_FILE="${current_observed_file}" \
  "${fixture_repo}/infra/ops/prod-rollback.sh" >/dev/null
[[ "$(cat "${current_observed_file}")" == "none" ]] ||
  fail "a current policy rollback must not receive historical review evidence"
[[ ! -e "${fixture_state}/rollback-review-exceptions.jsonl" ]] ||
  fail "a current policy rollback must not create an exception audit event"

DEPLOY_STATE_DIR="${fixture_state}" \
ROLLBACK_TAG="${target_tag}" \
ROLLBACK_BACKEND_IMAGE_REF="${backend_ref}" \
ROLLBACK_FRONTEND_IMAGE_REF="${frontend_ref}" \
ROLLBACK_ADMIN_IMAGE_REF="${admin_ref}" \
ROLLBACK_REVIEW_ACTOR="contract-operator" \
ROLLBACK_REVIEW_REASON_B64="${reason_b64}" \
ROLLBACK_DEPLOY_OBSERVED_FILE="${observed_file}" \
  "${fixture_repo}/infra/ops/prod-rollback.sh" >/dev/null

[[ -s "${observed_file}" ]] ||
  fail "prod-deploy must receive the generated rollback audit id"
audit_file="${fixture_state}/rollback-review-exceptions.jsonl"
[[ -s "${audit_file}" ]] ||
  fail "an expired review-window rollback must append an audit event"
[[ "$(stat -c '%a' "${audit_file}")" == "600" ]] ||
  fail "rollback audit log must use mode 0600"

python3 - "${audit_file}" "${observed_file}" "${target_tag}" "${reason}" <<'PY'
import json
import sys
from pathlib import Path

audit_path = Path(sys.argv[1])
observed_path = Path(sys.argv[2])
target_tag = sys.argv[3]
reason = sys.argv[4]
lines = audit_path.read_text(encoding="utf-8").splitlines()
assert len(lines) == 1
event = json.loads(lines[0])
assert event["event"] == "runtime_image_review_window_exception_authorized"
assert event["target_tag"] == target_tag
assert event["actor"] == "contract-operator"
assert event["reason"] == reason
assert event["current_policy_error"].endswith("review window expired")
assert len(event["policy_sha256"]) == 64
assert event["audit_id"] == observed_path.read_text(encoding="utf-8").strip()
PY

printf '[runtime-image-rollback-contract] all assertions passed\n'
