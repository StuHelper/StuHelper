#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
# shellcheck source=../lib/common.sh
source "${REPO_ROOT}/infra/ops/lib/common.sh"

fail() {
  printf '[environment-loader-security-contract][error] %s\n' "$*" >&2
  exit 1
}

tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT

for key in \
  BASH_ENV \
  ENV \
  PATH \
  IFS \
  PYTHONPATH \
  LD_PRELOAD \
  DYLD_INSERT_LIBRARIES \
  GCONV_PATH \
  NODE_OPTIONS \
  PERL5OPT \
  RUBYOPT \
  JAVA_TOOL_OPTIONS \
  GIT_EXEC_PATH \
  SSH_ASKPASS \
  DOCKER_HOST \
  DOCKER_CONTEXT \
  DOCKER_CERT_PATH \
  DOCKER_TLS \
  DOCKER_TLS_VERIFY \
  PRODUCTION_DEPLOY_LOCK_FD; do
  startup_env="${tmpdir}/${key}.env"
  printf '%s=/dev/null\n' "${key}" >"${startup_env}"
  if (source_env_file "${startup_env}") >"${tmpdir}/${key}.log" 2>&1; then
    fail "source_env_file accepted forbidden process-control variable ${key}"
  fi
  grep -q "process-control variable ${key} is not allowed" "${tmpdir}/${key}.log" ||
    fail "${key} rejection did not explain the protected boundary"
done

printf 'STACK_NAME=source-env-contract\n' >"${tmpdir}/safe.env"
mkdir -p "${tmpdir}/python-hook"
cat >"${tmpdir}/python-hook/sitecustomize.py" <<EOF
from pathlib import Path
Path(r"${tmpdir}/python-hook-executed").write_text("unsafe")
EOF
original_path="${PATH}"
export BASH_ENV=/dev/null
export ENV=/dev/null
export PYTHONPATH="${tmpdir}/python-hook"
export LD_LIBRARY_PATH="${tmpdir}/dynamic-loader-hook"
export NODE_OPTIONS=--require=/dev/null
export DOCKER_HOST=tcp://127.0.0.1:2375
export DOCKER_CONTEXT=untrusted-context
export DOCKER_CERT_PATH="${tmpdir}/docker-certs"
export DOCKER_TLS=1
export DOCKER_TLS_VERIFY=1
source_env_file "${tmpdir}/safe.env"
[[ "${STACK_NAME}" == "source-env-contract" ]] ||
  fail "source_env_file did not export a validated assignment"
[[ "${PATH}" == "${original_path}" ]] ||
  fail "source_env_file changed the caller-owned executable search path"
[[ ! -v BASH_ENV && ! -v ENV && ! -v PYTHONPATH && ! -v LD_LIBRARY_PATH && ! -v NODE_OPTIONS && \
  ! -v DOCKER_HOST && ! -v DOCKER_CONTEXT && ! -v DOCKER_CERT_PATH && ! -v DOCKER_TLS && \
  ! -v DOCKER_TLS_VERIFY ]] ||
  fail "source_env_file allowed inherited process-control hooks to survive"
[[ ! -e "${tmpdir}/python-hook-executed" ]] ||
  fail "source_env_file started its Python parser before clearing inherited import hooks"

cat >"${tmpdir}/bootstrap-safe.env" <<'EOF'
CASDOOR_BOOTSTRAP_CLIENT_ID=contract-client
CASDOOR_BOOTSTRAP_ORGANIZATION=built-in
EOF
source_casdoor_bootstrap_env_file "${tmpdir}/bootstrap-safe.env"
[[ "${CASDOOR_BOOTSTRAP_CLIENT_ID}" == "contract-client" ]] ||
  fail "Casdoor bootstrap loader rejected or changed an allowed key"
[[ "${CASDOOR_BOOTSTRAP_ORGANIZATION}" == "built-in" ]] ||
  fail "Casdoor bootstrap loader rejected the generated organization key"

cat >"${tmpdir}/bootstrap.env" <<'EOF'
CASDOOR_BOOTSTRAP_CLIENT_ID=contract-client
SCRIPT_DIR=/tmp/payload
EOF
if (source_casdoor_bootstrap_env_file "${tmpdir}/bootstrap.env") >"${tmpdir}/bootstrap.log" 2>&1; then
  fail "Casdoor bootstrap loader accepted a non-bootstrap key"
fi
grep -q 'environment key SCRIPT_DIR is not allowed in this file' "${tmpdir}/bootstrap.log" ||
  fail "Casdoor bootstrap allowlist rejection was not explicit"

cat >"${tmpdir}/bootstrap-secret-before-error.env" <<'EOF'
CASDOOR_BOOTSTRAP_CLIENT_SECRET=contract-secret-must-not-leak
PATH=/tmp/payload
EOF
if (source_casdoor_bootstrap_env_file "${tmpdir}/bootstrap-secret-before-error.env") >"${tmpdir}/bootstrap-secret-error.log" 2>&1; then
  fail "Casdoor bootstrap loader accepted an unexpected key after a credential"
fi
if grep -q 'contract-secret-must-not-leak' "${tmpdir}/bootstrap-secret-error.log"; then
  fail "source_env_file exposed an already parsed credential in its error output"
fi
grep -q 'process-control variable PATH is not allowed in StuHelper environment files' "${tmpdir}/bootstrap-secret-error.log" ||
  fail "credential-file rejection lost its non-sensitive diagnostic"

cat >"${tmpdir}/release.env" <<'EOF'
TAG=contract-release
PATH=/tmp/payload
EOF
if (source_release_record_env_file "${tmpdir}/release.env") >"${tmpdir}/release.log" 2>&1; then
  fail "release record loader accepted a process-control key"
fi
grep -q 'environment key PATH is not allowed in this release record' "${tmpdir}/release.log" ||
  fail "release record allowlist rejection was not explicit"

release_backend_ref='ghcr.io/stuhelper/backend@sha256:1111111111111111111111111111111111111111111111111111111111111111'
release_frontend_ref='ghcr.io/stuhelper/frontend@sha256:2222222222222222222222222222222222222222222222222222222222222222'
release_admin_ref='ghcr.io/stuhelper/admin@sha256:3333333333333333333333333333333333333333333333333333333333333333'

require_safe_release_tag release_2026.08-03
deploy_lock_state="${tmpdir}/deploy-lock-state"
deploy_lock_ready_fifo="${tmpdir}/deploy-lock-ready.fifo"
deploy_lock_release_fifo="${tmpdir}/deploy-lock-release.fifo"
mkfifo "${deploy_lock_ready_fifo}" "${deploy_lock_release_fifo}"
(
  export DEPLOY_STATE_DIR="${deploy_lock_state}"
  acquire_production_deploy_lock
  printf 'ready\n' >"${deploy_lock_ready_fifo}"
  IFS= read -r _ <"${deploy_lock_release_fifo}"
) &
deploy_lock_holder_pid=$!
IFS= read -r _ <"${deploy_lock_ready_fifo}"
if (
  export DEPLOY_STATE_DIR="${deploy_lock_state}"
  acquire_production_deploy_lock
) >"${tmpdir}/deploy-lock-contention.out" 2>"${tmpdir}/deploy-lock-contention.err"; then
  fail "host deployment lock allowed two concurrent production controllers"
fi
grep -q 'another production deploy or rollback already holds' "${tmpdir}/deploy-lock-contention.err" ||
  fail "host deployment lock contention was not reported"
printf 'release\n' >"${deploy_lock_release_fifo}"
wait "${deploy_lock_holder_pid}"
(
  export DEPLOY_STATE_DIR="${deploy_lock_state}"
  acquire_production_deploy_lock
)
inherited_lock_observed="${tmpdir}/inherited-deploy-lock-observed"
(
  export DEPLOY_STATE_DIR="${deploy_lock_state}"
  acquire_production_deploy_lock
  bash -c '
    set -euo pipefail
    source "$1"
    acquire_production_deploy_lock
    printf "reused\n" >"$2"
  ' _ "${REPO_ROOT}/infra/ops/lib/common.sh" "${inherited_lock_observed}"
)
[[ "$(cat "${inherited_lock_observed}")" == "reused" ]] ||
  fail "nested rollback/deploy controller did not reuse its inherited host lock"
deployment_attempt_one="$(new_deployment_attempt_id)"
deployment_attempt_two="$(new_deployment_attempt_id)"
[[ "${deployment_attempt_one}" =~ ^[0-9]{8}T[0-9]{6}Z-[0-9a-f]{32}$ ]] ||
  fail "deployment attempt identifier did not use the canonical safe format"
[[ "${deployment_attempt_one}" != "${deployment_attempt_two}" ]] ||
  fail "two deployment attempts unexpectedly reused the same identifier"
for unsafe_release_tag in \
  '' \
  '../escape' \
  'nested/release' \
  'release with spaces' \
  $'release\nnewline' \
  "$(printf 'a%.0s' {1..129})"; do
  if (require_safe_release_tag "${unsafe_release_tag}") >"${tmpdir}/unsafe-release-tag.log" 2>&1; then
    fail "release-tag validator accepted an unsafe or overlong identifier"
  fi
  grep -q 'release tag must be 1-128 characters' "${tmpdir}/unsafe-release-tag.log" ||
    fail "unsafe release-tag rejection did not report the canonical constraint"
done

cat >"${tmpdir}/release-mutable-image.env" <<EOF
TAG=contract-release
DEPLOYED_AT=2026-08-03T00:00:00Z
BACKEND_IMAGE_REF=ghcr.io/stuhelper/backend:sha-contract
FRONTEND_IMAGE_REF=${release_frontend_ref}
ADMIN_IMAGE_REF=${release_admin_ref}
EOF
if (source_release_record_env_file "${tmpdir}/release-mutable-image.env" contract-release) >"${tmpdir}/release-mutable-image.log" 2>&1; then
  fail "release record loader accepted a mutable application image tag"
fi
grep -q 'BACKEND_IMAGE_REF must be a complete image@sha256 digest reference' "${tmpdir}/release-mutable-image.log" ||
  fail "mutable release image rejection did not identify the digest requirement"

source_release_record_env_file \
  "${tmpdir}/release-mutable-image.env" \
  contract-release \
  true
[[ "${BACKEND_IMAGE_REF}" == "ghcr.io/stuhelper/backend:sha-contract" ]] ||
  fail "explicit legacy release parsing changed the historical backend reference before override"
if (source_release_record_env_file "${tmpdir}/release-mutable-image.env" contract-release maybe) \
  >"${tmpdir}/release-invalid-legacy-mode.log" 2>&1; then
  fail "release record loader accepted an invalid legacy parsing mode"
fi
grep -q 'allow_legacy_image_refs must be true or false' "${tmpdir}/release-invalid-legacy-mode.log" ||
  fail "invalid legacy release parsing mode was not reported"

cat >"${tmpdir}/release-missing.env" <<EOF
TAG=contract-release
DEPLOYED_AT=2026-08-03T00:00:00Z
BACKEND_IMAGE_REF=${release_backend_ref}
FRONTEND_IMAGE_REF=${release_frontend_ref}
EOF
if (source_release_record_env_file "${tmpdir}/release-missing.env" contract-release) >"${tmpdir}/release-missing.log" 2>&1; then
  fail "release record loader accepted a truncated record"
fi
grep -q 'missing required keys: ADMIN_IMAGE_REF' "${tmpdir}/release-missing.log" ||
  fail "truncated release record rejection did not identify the missing field"

cat >"${tmpdir}/release-duplicate.env" <<EOF
TAG=contract-release
DEPLOYED_AT=2026-08-03T00:00:00Z
BACKEND_IMAGE_REF=${release_backend_ref}
FRONTEND_IMAGE_REF=${release_frontend_ref}
ADMIN_IMAGE_REF=${release_admin_ref}
TAG=contract-release
EOF
if (source_release_record_env_file "${tmpdir}/release-duplicate.env" contract-release) >"${tmpdir}/release-duplicate.log" 2>&1; then
  fail "release record loader accepted a duplicate field"
fi
grep -q 'duplicate release record key: TAG' "${tmpdir}/release-duplicate.log" ||
  fail "duplicate release record rejection did not identify the repeated field"

cat >"${tmpdir}/release-safe.env" <<EOF
TAG=contract-release
DEPLOYED_AT=2026-08-03T00:00:00Z
BACKEND_IMAGE_REF=${release_backend_ref}
FRONTEND_IMAGE_REF=${release_frontend_ref}
ADMIN_IMAGE_REF=${release_admin_ref}
EOF
if (source_release_record_env_file "${tmpdir}/release-safe.env" wrong-release) >"${tmpdir}/release-tag-mismatch.log" 2>&1; then
  fail "release record loader accepted a mismatched target tag"
fi
grep -q 'TAG does not match rollback target wrong-release' "${tmpdir}/release-tag-mismatch.log" ||
  fail "release record tag mismatch did not report the expected target"

export TAG=stale-release
export DEPLOYED_AT=2026-01-01T00:00:00Z
export BACKEND_IMAGE_REF=stale-backend
export FRONTEND_IMAGE_REF=stale-frontend
export ADMIN_IMAGE_REF=stale-admin
source_release_record_env_file "${tmpdir}/release-safe.env" contract-release
[[ "${TAG}" == "contract-release" ]] ||
  fail "release record loader rejected or changed the target tag"
[[ "${BACKEND_IMAGE_REF}" == "${release_backend_ref}" ]] ||
  fail "release record loader retained an inherited backend image"
[[ "${FRONTEND_IMAGE_REF}" == "${release_frontend_ref}" ]] ||
  fail "release record loader retained an inherited frontend image"
[[ "${ADMIN_IMAGE_REF}" == "${release_admin_ref}" ]] ||
  fail "release record loader retained an inherited admin image"

legacy_permission_state="${tmpdir}/legacy-permission-state"
mkdir -p "${legacy_permission_state}/releases"
cp "${tmpdir}/release-safe.env" "${legacy_permission_state}/current-release.env"
cp "${tmpdir}/release-safe.env" "${legacy_permission_state}/releases/contract-release.env"
printf '2026-08-03T00:00:00Z\tcontract-release\n' >"${legacy_permission_state}/releases.log"
chmod 0644 \
  "${legacy_permission_state}/current-release.env" \
  "${legacy_permission_state}/releases/contract-release.env" \
  "${legacy_permission_state}/releases.log"
chmod 0775 "${legacy_permission_state}/releases"
(
  export DEPLOY_STATE_DIR="${legacy_permission_state}"
  migrate_legacy_release_state_permissions
)
for migrated_release_state_file in \
  "${legacy_permission_state}/current-release.env" \
  "${legacy_permission_state}/releases/contract-release.env" \
  "${legacy_permission_state}/releases.log"; do
  [[ "$(stat -c '%a' "${migrated_release_state_file}")" == "600" ]] ||
    fail "legacy release-state permission migration did not normalize ${migrated_release_state_file}"
done
[[ "$(stat -c '%a' "${legacy_permission_state}/releases")" == "700" ]] ||
  fail "legacy release-state permission migration did not protect the immutable-record directory"

legacy_identity_backend_ref='ghcr.io/stuhelper/backend:legacy-release'
legacy_identity_frontend_ref='ghcr.io/stuhelper/frontend:legacy-release'
legacy_identity_admin_ref='ghcr.io/stuhelper/admin:legacy-release'
legacy_identity_state="${tmpdir}/legacy-identity-state"
mkdir -p "${legacy_identity_state}/releases"
cat >"${legacy_identity_state}/current-release.env" <<EOF
TAG=legacy-release
DEPLOYED_AT=2026-08-02T00:00:00Z
BACKEND_IMAGE_REF=${legacy_identity_backend_ref}
FRONTEND_IMAGE_REF=${legacy_identity_frontend_ref}
ADMIN_IMAGE_REF=${legacy_identity_admin_ref}
EOF
cp \
  "${legacy_identity_state}/current-release.env" \
  "${legacy_identity_state}/releases/legacy-release.env"
printf '2026-08-02T00:00:00Z\tlegacy-release\n' >"${legacy_identity_state}/releases.log"
chmod 0600 \
  "${legacy_identity_state}/current-release.env" \
  "${legacy_identity_state}/releases/legacy-release.env" \
  "${legacy_identity_state}/releases.log"

legacy_docker_bin="${tmpdir}/legacy-docker-bin"
mkdir -p "${legacy_docker_bin}"
cat >"${legacy_docker_bin}/docker" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

last_argument="${!#}"
backend_image_id="sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
frontend_image_id="sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
admin_image_id="sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"

if [[ "$1" == "inspect" && "$2" == "--type" ]]; then
  case "${last_argument}" in
    legacy-contract-app)
      configured_image="ghcr.io/stuhelper/backend:legacy-release"
      [[ "${LEGACY_FAKE_CONFIG_MISMATCH:-false}" != "true" ]] || configured_image="ghcr.io/stuhelper/backend:moved"
      printf '{"imageId":"%s","configuredImage":"%s","state":"running","project":"legacy-contract","service":"app"}\n' \
        "${backend_image_id}" "${configured_image}"
      ;;
    legacy-contract-frontend)
      printf '{"imageId":"%s","configuredImage":"ghcr.io/stuhelper/frontend:legacy-release","state":"running","project":"legacy-contract","service":"frontend"}\n' \
        "${frontend_image_id}"
      ;;
    legacy-contract-admin)
      printf '{"imageId":"%s","configuredImage":"ghcr.io/stuhelper/admin:legacy-release","state":"running","project":"legacy-contract","service":"admin"}\n' \
        "${admin_image_id}"
      ;;
    *) exit 1 ;;
  esac
  exit 0
fi

if [[ "$1" == "image" && "$2" == "inspect" ]]; then
  case "${last_argument}" in
    "${backend_image_id}")
      printf '["ghcr.io/stuhelper/backend@sha256:1111111111111111111111111111111111111111111111111111111111111111"]\n'
      ;;
    "${frontend_image_id}")
      printf '["ghcr.io/stuhelper/frontend@sha256:2222222222222222222222222222222222222222222222222222222222222222"]\n'
      ;;
    "${admin_image_id}")
      printf '["ghcr.io/stuhelper/admin@sha256:3333333333333333333333333333333333333333333333333333333333333333"]\n'
      ;;
    *) exit 1 ;;
  esac
  exit 0
fi

exit 1
EOF
chmod +x "${legacy_docker_bin}/docker"

(
  export PATH="${legacy_docker_bin}:${original_path}"
  export DEPLOY_STATE_DIR="${legacy_identity_state}"
  export STACK_NAME=legacy-contract
  migrate_verified_legacy_current_release_identity
)
assert_file_contains() {
  local file="$1"
  local expected="$2"
  grep -qF "${expected}" "${file}" || fail "${file} does not contain expected text: ${expected}"
}
assert_file_contains "${legacy_identity_state}/current-release.env" "BACKEND_IMAGE_REF=${release_backend_ref}"
assert_file_contains "${legacy_identity_state}/current-release.env" "FRONTEND_IMAGE_REF=${release_frontend_ref}"
assert_file_contains "${legacy_identity_state}/current-release.env" "ADMIN_IMAGE_REF=${release_admin_ref}"
cmp -s \
  "${legacy_identity_state}/current-release.env" \
  "${legacy_identity_state}/releases/legacy-release.env" ||
  fail "verified legacy migration left current and per-tag records inconsistent"
legacy_identity_evidence="${legacy_identity_state}/release-migrations/legacy-release.json"
[[ -s "${legacy_identity_evidence}" && "$(stat -c '%a' "${legacy_identity_evidence}")" == "600" ]] ||
  fail "verified legacy migration did not publish protected audit evidence"
python3 - "${legacy_identity_evidence}" <<'PY'
import json
import sys
from pathlib import Path

document = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
assert document["event"] == "legacy_release_identity_migrated"
assert document["verificationSource"] == "running_compose_container_image_identity"
assert set(document["images"]) == {
    "BACKEND_IMAGE_REF",
    "FRONTEND_IMAGE_REF",
    "ADMIN_IMAGE_REF",
}
PY

legacy_identity_mismatch_state="${tmpdir}/legacy-identity-mismatch-state"
mkdir -p "${legacy_identity_mismatch_state}/releases"
cat >"${legacy_identity_mismatch_state}/current-release.env" <<EOF
TAG=legacy-release
DEPLOYED_AT=2026-08-02T00:00:00Z
BACKEND_IMAGE_REF=${legacy_identity_backend_ref}
FRONTEND_IMAGE_REF=${legacy_identity_frontend_ref}
ADMIN_IMAGE_REF=${legacy_identity_admin_ref}
EOF
cp \
  "${legacy_identity_mismatch_state}/current-release.env" \
  "${legacy_identity_mismatch_state}/releases/legacy-release.env"
chmod 0600 \
  "${legacy_identity_mismatch_state}/current-release.env" \
  "${legacy_identity_mismatch_state}/releases/legacy-release.env"
legacy_identity_checksum_before="$(sha256sum "${legacy_identity_mismatch_state}/current-release.env" | cut -d ' ' -f 1)"
if (
  export PATH="${legacy_docker_bin}:${original_path}"
  export DEPLOY_STATE_DIR="${legacy_identity_mismatch_state}"
  export STACK_NAME=legacy-contract
  export LEGACY_FAKE_CONFIG_MISMATCH=true
  migrate_verified_legacy_current_release_identity
) >"${tmpdir}/legacy-identity-mismatch.out" 2>"${tmpdir}/legacy-identity-mismatch.err"; then
  fail "legacy identity migration accepted a container whose configured image did not match the release record"
fi
grep -q 'does not match the configured image' "${tmpdir}/legacy-identity-mismatch.err" ||
  fail "legacy identity mismatch did not report the verified-container boundary"
[[ "$(sha256sum "${legacy_identity_mismatch_state}/current-release.env" | cut -d ' ' -f 1)" == "${legacy_identity_checksum_before}" ]] ||
  fail "failed legacy identity migration changed the current release record"
[[ ! -e "${legacy_identity_mismatch_state}/release-migrations/legacy-release.json" ]] ||
  fail "failed legacy identity migration published audit evidence before verification"

explicit_legacy_state="${tmpdir}/explicit-legacy-state"
mkdir -p "${explicit_legacy_state}/releases"
cat >"${explicit_legacy_state}/releases/explicit-legacy.env" <<'EOF'
TAG=explicit-legacy
DEPLOYED_AT=2026-08-01T12:00:00Z
BACKEND_IMAGE_REF=ghcr.io/stuhelper/backend:explicit-legacy
FRONTEND_IMAGE_REF=ghcr.io/stuhelper/frontend:explicit-legacy
ADMIN_IMAGE_REF=ghcr.io/stuhelper/admin:explicit-legacy
EOF
chmod 0600 "${explicit_legacy_state}/releases/explicit-legacy.env"
(
  export DEPLOY_STATE_DIR="${explicit_legacy_state}"
  migrate_explicit_legacy_release_identity \
    explicit-legacy \
    "${release_backend_ref}" \
    "${release_frontend_ref}" \
    "${release_admin_ref}" \
    contract-operator \
    'restore a provenance-verified historical release'
)
assert_file_contains \
  "${explicit_legacy_state}/releases/explicit-legacy.env" \
  "BACKEND_IMAGE_REF=${release_backend_ref}"
explicit_legacy_evidence="${explicit_legacy_state}/release-migrations/explicit-legacy.json"
python3 - "${explicit_legacy_evidence}" <<'PY'
import json
import sys
from pathlib import Path

document = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
assert document["event"] == "legacy_release_identity_migrated"
assert document["verificationSource"] == "explicit_provenance_verified_rollback_digest_override"
assert document["actor"] == "contract-operator"
assert document["reason"] == "restore a provenance-verified historical release"
assert set(document["images"]) == {
    "BACKEND_IMAGE_REF",
    "FRONTEND_IMAGE_REF",
    "ADMIN_IMAGE_REF",
}
PY
explicit_legacy_record_checksum="$(sha256sum "${explicit_legacy_state}/releases/explicit-legacy.env" | cut -d ' ' -f 1)"
explicit_legacy_evidence_checksum="$(sha256sum "${explicit_legacy_evidence}" | cut -d ' ' -f 1)"
(
  export DEPLOY_STATE_DIR="${explicit_legacy_state}"
  migrate_explicit_legacy_release_identity \
    explicit-legacy \
    "${release_backend_ref}" \
    "${release_frontend_ref}" \
    "${release_admin_ref}" \
    contract-operator \
    'restore a provenance-verified historical release'
)
[[ "$(sha256sum "${explicit_legacy_state}/releases/explicit-legacy.env" | cut -d ' ' -f 1)" == "${explicit_legacy_record_checksum}" ]] ||
  fail "idempotent explicit legacy migration changed its canonical record"
[[ "$(sha256sum "${explicit_legacy_evidence}" | cut -d ' ' -f 1)" == "${explicit_legacy_evidence_checksum}" ]] ||
  fail "idempotent explicit legacy migration changed its audit evidence"

explicit_wrong_repository_state="${tmpdir}/explicit-wrong-repository-state"
mkdir -p "${explicit_wrong_repository_state}/releases"
cat >"${explicit_wrong_repository_state}/releases/explicit-legacy.env" <<'EOF'
TAG=explicit-legacy
DEPLOYED_AT=2026-08-01T12:00:00Z
BACKEND_IMAGE_REF=ghcr.io/different/backend:explicit-legacy
FRONTEND_IMAGE_REF=ghcr.io/stuhelper/frontend:explicit-legacy
ADMIN_IMAGE_REF=ghcr.io/stuhelper/admin:explicit-legacy
EOF
chmod 0600 "${explicit_wrong_repository_state}/releases/explicit-legacy.env"
explicit_wrong_repository_checksum="$(sha256sum "${explicit_wrong_repository_state}/releases/explicit-legacy.env" | cut -d ' ' -f 1)"
if (
  export DEPLOY_STATE_DIR="${explicit_wrong_repository_state}"
  migrate_explicit_legacy_release_identity \
    explicit-legacy \
    "${release_backend_ref}" \
    "${release_frontend_ref}" \
    "${release_admin_ref}" \
    contract-operator \
    'restore a provenance-verified historical release'
) >"${tmpdir}/explicit-wrong-repository.out" 2>"${tmpdir}/explicit-wrong-repository.err"; then
  fail "explicit legacy migration accepted a digest from a different image repository"
fi
grep -q 'changes the legacy image repository' "${tmpdir}/explicit-wrong-repository.err" ||
  fail "explicit legacy migration did not report repository drift"
[[ "$(sha256sum "${explicit_wrong_repository_state}/releases/explicit-legacy.env" | cut -d ' ' -f 1)" == "${explicit_wrong_repository_checksum}" ]] ||
  fail "failed explicit legacy migration changed the historical record"
[[ ! -e "${explicit_wrong_repository_state}/release-migrations/explicit-legacy.json" ]] ||
  fail "failed explicit legacy migration published audit evidence"

divergent_legacy_state="${tmpdir}/divergent-legacy-state"
mkdir -p "${divergent_legacy_state}/releases"
cat >"${divergent_legacy_state}/releases/divergent-legacy.env" <<'EOF'
TAG=divergent-legacy
DEPLOYED_AT=2026-08-01T12:00:00Z
BACKEND_IMAGE_REF=ghcr.io/stuhelper/backend:per-tag-version
FRONTEND_IMAGE_REF=ghcr.io/stuhelper/frontend:per-tag-version
ADMIN_IMAGE_REF=ghcr.io/stuhelper/admin:per-tag-version
EOF
cat >"${divergent_legacy_state}/current-release.env" <<'EOF'
TAG=divergent-legacy
DEPLOYED_AT=2026-08-01T12:00:00Z
BACKEND_IMAGE_REF=ghcr.io/stuhelper/backend:current-version
FRONTEND_IMAGE_REF=ghcr.io/stuhelper/frontend:current-version
ADMIN_IMAGE_REF=ghcr.io/stuhelper/admin:current-version
EOF
chmod 0600 \
  "${divergent_legacy_state}/releases/divergent-legacy.env" \
  "${divergent_legacy_state}/current-release.env"
divergent_release_checksum="$(sha256sum "${divergent_legacy_state}/releases/divergent-legacy.env" | cut -d ' ' -f 1)"
divergent_current_checksum="$(sha256sum "${divergent_legacy_state}/current-release.env" | cut -d ' ' -f 1)"
if (
  export DEPLOY_STATE_DIR="${divergent_legacy_state}"
  migrate_explicit_legacy_release_identity \
    divergent-legacy \
    "${release_backend_ref}" \
    "${release_frontend_ref}" \
    "${release_admin_ref}" \
    contract-operator \
    'reject conflicting original legacy release identities'
) >"${tmpdir}/divergent-legacy.out" 2>"${tmpdir}/divergent-legacy.err"; then
  fail "explicit legacy migration erased divergent current and per-tag identities"
fi
grep -q 'identities differ before migration' "${tmpdir}/divergent-legacy.err" ||
  fail "divergent legacy identity rejection did not identify the original-ledger conflict"
[[ "$(sha256sum "${divergent_legacy_state}/releases/divergent-legacy.env" | cut -d ' ' -f 1)" == "${divergent_release_checksum}" ]] ||
  fail "divergent legacy rejection changed the per-tag record"
[[ "$(sha256sum "${divergent_legacy_state}/current-release.env" | cut -d ' ' -f 1)" == "${divergent_current_checksum}" ]] ||
  fail "divergent legacy rejection changed the current record"
[[ ! -e "${divergent_legacy_state}/release-migrations/divergent-legacy.json" ]] ||
  fail "divergent legacy rejection published migration evidence"

partial_legacy_state="${tmpdir}/partial-legacy-state"
mkdir -p "${partial_legacy_state}/releases"
cat >"${partial_legacy_state}/releases/partial-legacy.env" <<'EOF'
TAG=partial-legacy
DEPLOYED_AT=2026-08-01T12:00:00Z
BACKEND_IMAGE_REF=ghcr.io/stuhelper/backend:partial-legacy
FRONTEND_IMAGE_REF=ghcr.io/stuhelper/frontend:partial-legacy
ADMIN_IMAGE_REF=ghcr.io/stuhelper/admin:partial-legacy
EOF
cat >"${partial_legacy_state}/current-release.env" <<EOF
TAG=partial-legacy
DEPLOYED_AT=2026-08-01T12:00:00Z
BACKEND_IMAGE_REF=${release_backend_ref}
FRONTEND_IMAGE_REF=${release_frontend_ref}
ADMIN_IMAGE_REF=${release_admin_ref}
EOF
chmod 0600 \
  "${partial_legacy_state}/releases/partial-legacy.env" \
  "${partial_legacy_state}/current-release.env"
partial_release_checksum="$(sha256sum "${partial_legacy_state}/releases/partial-legacy.env" | cut -d ' ' -f 1)"
partial_current_checksum="$(sha256sum "${partial_legacy_state}/current-release.env" | cut -d ' ' -f 1)"
if (
  export DEPLOY_STATE_DIR="${partial_legacy_state}"
  migrate_explicit_legacy_release_identity \
    partial-legacy \
    "${release_backend_ref}" \
    "${release_frontend_ref}" \
    "${release_admin_ref}" \
    contract-operator \
    'reject an impossible current-first partial migration'
) >"${tmpdir}/partial-current-first.out" 2>"${tmpdir}/partial-current-first.err"; then
  fail "explicit legacy migration accepted an impossible current-first partial state"
fi
grep -q 'identities differ before migration' "${tmpdir}/partial-current-first.err" ||
  fail "current-first partial rejection did not identify divergent history"
[[ "$(sha256sum "${partial_legacy_state}/releases/partial-legacy.env" | cut -d ' ' -f 1)" == "${partial_release_checksum}" ]] ||
  fail "current-first partial rejection changed the per-tag record"
[[ "$(sha256sum "${partial_legacy_state}/current-release.env" | cut -d ' ' -f 1)" == "${partial_current_checksum}" ]] ||
  fail "current-first partial rejection changed the current record"

release_first_partial_state="${tmpdir}/release-first-partial-state"
mkdir -p \
  "${release_first_partial_state}/releases" \
  "${release_first_partial_state}/release-migrations"
cat >"${release_first_partial_state}/releases/partial-legacy.env" <<EOF
TAG=partial-legacy
DEPLOYED_AT=2026-08-01T12:00:00Z
BACKEND_IMAGE_REF=${release_backend_ref}
FRONTEND_IMAGE_REF=${release_frontend_ref}
ADMIN_IMAGE_REF=${release_admin_ref}
EOF
cat >"${release_first_partial_state}/current-release.env" <<'EOF'
TAG=partial-legacy
DEPLOYED_AT=2026-08-01T12:00:00Z
BACKEND_IMAGE_REF=ghcr.io/stuhelper/backend:partial-legacy
FRONTEND_IMAGE_REF=ghcr.io/stuhelper/frontend:partial-legacy
ADMIN_IMAGE_REF=ghcr.io/stuhelper/admin:partial-legacy
EOF
chmod 0700 "${release_first_partial_state}/release-migrations"
chmod 0600 \
  "${release_first_partial_state}/releases/partial-legacy.env" \
  "${release_first_partial_state}/current-release.env"
if (
  export DEPLOY_STATE_DIR="${release_first_partial_state}"
  migrate_explicit_legacy_release_identity \
    partial-legacy \
    "${release_backend_ref}" \
    "${release_frontend_ref}" \
    "${release_admin_ref}" \
    contract-operator \
    'complete an interrupted release-first migration'
) >"${tmpdir}/partial-missing-evidence.out" 2>"${tmpdir}/partial-missing-evidence.err"; then
  fail "release-first partial migration recreated missing audit evidence"
fi
grep -q 'requires matching preexisting evidence' "${tmpdir}/partial-missing-evidence.err" ||
  fail "release-first partial state did not require its durable pre-write evidence"

python3 - \
  "${release_first_partial_state}/current-release.env" \
  "${release_first_partial_state}/release-migrations/partial-legacy.json" \
  "${release_backend_ref}" \
  "${release_frontend_ref}" \
  "${release_admin_ref}" <<'PY'
import hashlib
import json
import sys
from pathlib import Path

legacy_path = Path(sys.argv[1])
evidence_path = Path(sys.argv[2])
digests = dict(zip(
    ("BACKEND_IMAGE_REF", "FRONTEND_IMAGE_REF", "ADMIN_IMAGE_REF"),
    sys.argv[3:],
    strict=True,
))
legacy_payload = legacy_path.read_bytes()
legacy_values = dict(
    line.split("=", 1)
    for line in legacy_payload.decode("utf-8").splitlines()
)
document = {
    "schemaVersion": 1,
    "event": "legacy_release_identity_migrated",
    "tag": "partial-legacy",
    "deployedAt": "2026-08-01T12:00:00Z",
    "migratedAt": "2026-08-03T00:00:00Z",
    "legacyRecordSha256": hashlib.sha256(legacy_payload).hexdigest(),
    "verificationSource": "explicit_provenance_verified_rollback_digest_override",
    "actor": "contract-operator",
    "reason": "complete an interrupted release-first migration",
    "images": {
        key: {"legacyRef": legacy_values[key], "digestRef": digest}
        for key, digest in digests.items()
    },
}
evidence_path.write_text(
    json.dumps(document, sort_keys=True, separators=(",", ":")) + "\n",
    encoding="utf-8",
)
PY
chmod 0600 "${release_first_partial_state}/release-migrations/partial-legacy.json"
(
  export DEPLOY_STATE_DIR="${release_first_partial_state}"
  migrate_explicit_legacy_release_identity \
    partial-legacy \
    "${release_backend_ref}" \
    "${release_frontend_ref}" \
    "${release_admin_ref}" \
    contract-operator \
    'complete an interrupted release-first migration'
)
cmp -s \
  "${release_first_partial_state}/current-release.env" \
  "${release_first_partial_state}/releases/partial-legacy.env" ||
  fail "evidence-backed release-first recovery did not converge both records"

unsafe_permission_state="${tmpdir}/unsafe-permission-state"
mkdir -p "${unsafe_permission_state}"
cp "${tmpdir}/release-safe.env" "${unsafe_permission_state}/current-release.env"
chmod 0664 "${unsafe_permission_state}/current-release.env"
if (
  export DEPLOY_STATE_DIR="${unsafe_permission_state}"
  migrate_legacy_release_state_permissions
) >"${tmpdir}/unsafe-permission.out" 2>"${tmpdir}/unsafe-permission.err"; then
  fail "legacy release-state permission migration accepted a group-writable file"
fi
grep -q 'unsafe mode 0664' "${tmpdir}/unsafe-permission.err" ||
  fail "unsafe legacy release-state mode rejection was not explicit"

symlink_permission_state="${tmpdir}/symlink-permission-state"
mkdir -p "${symlink_permission_state}"
ln -s "${tmpdir}/release-safe.env" "${symlink_permission_state}/current-release.env"
if (
  export DEPLOY_STATE_DIR="${symlink_permission_state}"
  migrate_legacy_release_state_permissions
) >"${tmpdir}/symlink-permission.out" 2>"${tmpdir}/symlink-permission.err"; then
  fail "legacy release-state permission migration followed a symlink"
fi
grep -q 'must not be a symlink' "${tmpdir}/symlink-permission.err" ||
  fail "legacy release-state symlink rejection was not explicit"

release_state_dir="${tmpdir}/release-state"
fresh_release_guard_state="${tmpdir}/fresh-release-guard-state"
python3 - "${REPO_ROOT}/infra/ops/lib/common.sh" <<'PY'
import sys
from pathlib import Path

source = Path(sys.argv[1]).read_text(encoding="utf-8")
function_start = source.index("_release_record_operation()")
immutable = source.index("release_payload = publish_immutable_release", function_start)
activation_log = source.index('log_path = state_dir / "releases.log"', immutable)
current_pointer = source.index(
    'atomic_write(state_dir / "current-release.env", release_payload)',
    activation_log,
)
if not immutable < activation_log < current_pointer:
    raise SystemExit("release publication must durably log activation before replacing current")
PY
(
  export DEPLOY_STATE_DIR="${fresh_release_guard_state}"
  export BACKEND_IMAGE_REF="${release_backend_ref}"
  export FRONTEND_IMAGE_REF="${release_frontend_ref}"
  export ADMIN_IMAGE_REF="${release_admin_ref}"
  require_release_tag_identity_available contract-release
)
[[ ! -e "${fresh_release_guard_state}" ]] ||
  fail "checking a previously unused release tag created deployment state"

(
  umask 0002
  export DEPLOY_STATE_DIR="${release_state_dir}"
  export BACKEND_IMAGE_REF="${release_backend_ref}"
  export FRONTEND_IMAGE_REF="${release_frontend_ref}"
  export ADMIN_IMAGE_REF="${release_admin_ref}"
  record_release contract-release
)
[[ "$(stat -c '%a' "${release_state_dir}/releases")" == "700" ]] ||
  fail "immutable release directory must use mode 0700 under a collaborative umask"
cmp -s "${release_state_dir}/current-release.env" "${release_state_dir}/releases/contract-release.env" ||
  fail "atomic current and immutable release records diverged"
for release_record_path in \
  "${release_state_dir}/current-release.env" \
  "${release_state_dir}/releases/contract-release.env" \
  "${release_state_dir}/releases.log"; do
  [[ "$(stat -c '%a' "${release_record_path}")" == "600" ]] ||
    fail "release record ${release_record_path} must use mode 0600"
done
immutable_release_path="${release_state_dir}/releases/contract-release.env"
immutable_inode_before="$(stat -c '%i' "${immutable_release_path}")"
immutable_checksum_before="$(sha256sum "${immutable_release_path}" | cut -d ' ' -f 1)"
(
  export DEPLOY_STATE_DIR="${release_state_dir}"
  export BACKEND_IMAGE_REF="${release_backend_ref}"
  export FRONTEND_IMAGE_REF="${release_frontend_ref}"
  export ADMIN_IMAGE_REF="${release_admin_ref}"
  record_release contract-release
)
[[ "$(stat -c '%i' "${immutable_release_path}")" == "${immutable_inode_before}" ]] ||
  fail "reusing an identical release replaced its immutable record"
[[ "$(sha256sum "${immutable_release_path}" | cut -d ' ' -f 1)" == "${immutable_checksum_before}" ]] ||
  fail "reusing an identical release changed its immutable payload"
cmp -s "${release_state_dir}/current-release.env" "${immutable_release_path}" ||
  fail "reusing an identical release did not restore its original immutable payload"
[[ "$(wc -l <"${release_state_dir}/releases.log")" == "2" ]] ||
  fail "release activation log did not record the repeated activation"

(
  export DEPLOY_STATE_DIR="${release_state_dir}"
  export BACKEND_IMAGE_REF="${release_backend_ref}"
  export FRONTEND_IMAGE_REF="${release_frontend_ref}"
  export ADMIN_IMAGE_REF="${release_admin_ref}"
  require_release_tag_identity_available contract-release
)
[[ "$(wc -l <"${release_state_dir}/releases.log")" == "2" ]] ||
  fail "read-only release identity validation changed the activation log"

interrupted_current_state="${tmpdir}/interrupted-current-state"
mkdir -p "${interrupted_current_state}/releases"
cp "${release_state_dir}/current-release.env" \
  "${interrupted_current_state}/current-release.env"
cp "${release_state_dir}/releases/contract-release.env" \
  "${interrupted_current_state}/releases/contract-release.env"
chmod 0600 \
  "${interrupted_current_state}/current-release.env" \
  "${interrupted_current_state}/releases/contract-release.env"
(
  export DEPLOY_STATE_DIR="${interrupted_current_state}"
  export BACKEND_IMAGE_REF="${release_backend_ref}"
  export FRONTEND_IMAGE_REF="${release_frontend_ref}"
  export ADMIN_IMAGE_REF="${release_admin_ref}"
  require_release_tag_identity_available contract-release
  record_release contract-release
)
[[ "$(wc -l <"${interrupted_current_state}/releases.log")" == "1" ]] ||
  fail "exact retry did not repair an unlogged current release"
cmp -s \
  "${interrupted_current_state}/current-release.env" \
  "${interrupted_current_state}/releases/contract-release.env" ||
  fail "unlogged-current repair changed the immutable release identity"

unlogged_other_tag_state="${tmpdir}/unlogged-other-tag-state"
cp -a "${interrupted_current_state}" "${unlogged_other_tag_state}"
: >"${unlogged_other_tag_state}/releases.log"
chmod 0600 "${unlogged_other_tag_state}/releases.log"
if (
  export DEPLOY_STATE_DIR="${unlogged_other_tag_state}"
  export BACKEND_IMAGE_REF="${release_backend_ref}"
  export FRONTEND_IMAGE_REF="${release_frontend_ref}"
  export ADMIN_IMAGE_REF="${release_admin_ref}"
  require_release_tag_identity_available future-release
) >"${tmpdir}/guard-unlogged-other-tag.out" 2>"${tmpdir}/guard-unlogged-other-tag.err"; then
  fail "unlogged current release allowed a different tag to replace its sole evidence"
fi
grep -q 'retry that exact release identity before deploying another tag' \
  "${tmpdir}/guard-unlogged-other-tag.err" ||
  fail "unlogged-current rejection did not explain the exact retry recovery path"

malformed_log_existing_state="${tmpdir}/malformed-log-existing-state"
mkdir -p "${malformed_log_existing_state}/releases"
cp "${release_state_dir}/current-release.env" "${malformed_log_existing_state}/current-release.env"
cp "${release_state_dir}/releases/contract-release.env" \
  "${malformed_log_existing_state}/releases/contract-release.env"
printf '2026-08-03T00:00:00Z\tcontract-release' >"${malformed_log_existing_state}/releases.log"
chmod 0600 \
  "${malformed_log_existing_state}/current-release.env" \
  "${malformed_log_existing_state}/releases/contract-release.env" \
  "${malformed_log_existing_state}/releases.log"
if (
  export DEPLOY_STATE_DIR="${malformed_log_existing_state}"
  export BACKEND_IMAGE_REF="${release_backend_ref}"
  export FRONTEND_IMAGE_REF="${release_frontend_ref}"
  export ADMIN_IMAGE_REF="${release_admin_ref}"
  require_release_tag_identity_available contract-release
) >"${tmpdir}/guard-malformed-existing-log.out" 2>"${tmpdir}/guard-malformed-existing-log.err"; then
  fail "release identity guard accepted a truncated activation log when the per-tag record existed"
fi
grep -q 'release activation log is truncated' "${tmpdir}/guard-malformed-existing-log.err" ||
  fail "existing-tag guard did not report the truncated activation log"
malformed_log_before="$(sha256sum "${malformed_log_existing_state}/releases.log" | cut -d ' ' -f 1)"
if (
  export DEPLOY_STATE_DIR="${malformed_log_existing_state}"
  export BACKEND_IMAGE_REF="${release_backend_ref}"
  export FRONTEND_IMAGE_REF="${release_frontend_ref}"
  export ADMIN_IMAGE_REF="${release_admin_ref}"
  record_release contract-release
) >"${tmpdir}/record-malformed-existing-log.out" 2>"${tmpdir}/record-malformed-existing-log.err"; then
  fail "release recorder appended to a malformed activation log"
fi
grep -q 'release activation log is truncated' "${tmpdir}/record-malformed-existing-log.err" ||
  fail "release recorder did not validate the existing activation log before publication"
[[ "$(sha256sum "${malformed_log_existing_state}/releases.log" | cut -d ' ' -f 1)" == "${malformed_log_before}" ]] ||
  fail "failed malformed-ledger publication changed the activation log"

legacy_history_state="${tmpdir}/legacy-history-state"
mkdir -p "${legacy_history_state}/releases"
cp "${release_state_dir}/current-release.env" "${legacy_history_state}/current-release.env"
cp \
  "${release_state_dir}/releases/contract-release.env" \
  "${legacy_history_state}/releases/contract-release.env"
cat >"${legacy_history_state}/releases/legacy-history.env" <<'EOF'
TAG=legacy-history
DEPLOYED_AT=2026-08-01T00:00:00Z
BACKEND_IMAGE_REF=ghcr.io/stuhelper/backend:legacy-history
FRONTEND_IMAGE_REF=ghcr.io/stuhelper/frontend:legacy-history
ADMIN_IMAGE_REF=ghcr.io/stuhelper/admin:legacy-history
EOF
printf '%s\n' \
  $'2026-08-01T00:00:00Z\tlegacy-history' \
  $'2026-08-03T00:00:00Z\tcontract-release' \
  >"${legacy_history_state}/releases.log"
chmod 0600 \
  "${legacy_history_state}/current-release.env" \
  "${legacy_history_state}/releases/contract-release.env" \
  "${legacy_history_state}/releases/legacy-history.env" \
  "${legacy_history_state}/releases.log"
legacy_history_log_before="$(sha256sum "${legacy_history_state}/releases.log" | cut -d ' ' -f 1)"
(
  export DEPLOY_STATE_DIR="${legacy_history_state}"
  export BACKEND_IMAGE_REF="${release_backend_ref}"
  export FRONTEND_IMAGE_REF="${release_frontend_ref}"
  export ADMIN_IMAGE_REF="${release_admin_ref}"
  require_release_tag_identity_available future-release
)
[[ "$(sha256sum "${legacy_history_state}/releases.log" | cut -d ' ' -f 1)" == "${legacy_history_log_before}" ]] ||
  fail "semantic ledger validation changed a valid activation log"

omitted_history_state="${tmpdir}/omitted-history-state"
cp -a "${legacy_history_state}" "${omitted_history_state}"
grep -v $'\tlegacy-history$' \
  "${legacy_history_state}/releases.log" \
  >"${omitted_history_state}/releases.log"
chmod 0600 "${omitted_history_state}/releases.log"
omitted_history_log_before="$(sha256sum "${omitted_history_state}/releases.log" | cut -d ' ' -f 1)"
if (
  export DEPLOY_STATE_DIR="${omitted_history_state}"
  export BACKEND_IMAGE_REF="${release_backend_ref}"
  export FRONTEND_IMAGE_REF="${release_frontend_ref}"
  export ADMIN_IMAGE_REF="${release_admin_ref}"
  require_release_tag_identity_available future-release
) >"${tmpdir}/guard-omitted-history.out" 2>"${tmpdir}/guard-omitted-history.err"; then
  fail "release identity guard accepted immutable history omitted from the activation log"
fi
grep -q 'immutable release tag legacy-history is missing from the activation log' \
  "${tmpdir}/guard-omitted-history.err" ||
  fail "omitted activation history did not identify the surviving immutable record"
if (
  export DEPLOY_STATE_DIR="${omitted_history_state}"
  export BACKEND_IMAGE_REF="${release_backend_ref}"
  export FRONTEND_IMAGE_REF="${release_frontend_ref}"
  export ADMIN_IMAGE_REF="${release_admin_ref}"
  record_release future-release
) >"${tmpdir}/record-omitted-history.out" 2>"${tmpdir}/record-omitted-history.err"; then
  fail "release recorder appended to a ledger missing immutable history"
fi
[[ "$(sha256sum "${omitted_history_state}/releases.log" | cut -d ' ' -f 1)" == "${omitted_history_log_before}" ]] ||
  fail "failed omitted-history publication changed the activation log"
[[ ! -e "${omitted_history_state}/releases/future-release.env" ]] ||
  fail "failed omitted-history publication created a candidate record"

missing_history_state="${tmpdir}/missing-history-state"
cp -a "${legacy_history_state}" "${missing_history_state}"
rm -f "${missing_history_state}/releases/legacy-history.env"
missing_history_log_before="$(sha256sum "${missing_history_state}/releases.log" | cut -d ' ' -f 1)"
if (
  export DEPLOY_STATE_DIR="${missing_history_state}"
  export BACKEND_IMAGE_REF="${release_backend_ref}"
  export FRONTEND_IMAGE_REF="${release_frontend_ref}"
  export ADMIN_IMAGE_REF="${release_admin_ref}"
  require_release_tag_identity_available future-release
) >"${tmpdir}/guard-missing-history.out" 2>"${tmpdir}/guard-missing-history.err"; then
  fail "release identity guard accepted a log whose historical per-tag record was missing"
fi
grep -q 'release tag legacy-history was previously used but its immutable record is missing' \
  "${tmpdir}/guard-missing-history.err" ||
  fail "missing historical record rejection did not identify the logged tag"
if (
  export DEPLOY_STATE_DIR="${missing_history_state}"
  export BACKEND_IMAGE_REF="${release_backend_ref}"
  export FRONTEND_IMAGE_REF="${release_frontend_ref}"
  export ADMIN_IMAGE_REF="${release_admin_ref}"
  record_release future-release
) >"${tmpdir}/record-missing-history.out" 2>"${tmpdir}/record-missing-history.err"; then
  fail "release recorder appended to a semantically incomplete historical ledger"
fi
[[ "$(sha256sum "${missing_history_state}/releases.log" | cut -d ' ' -f 1)" == "${missing_history_log_before}" ]] ||
  fail "failed historical-ledger publication changed the activation log"
[[ ! -e "${missing_history_state}/releases/future-release.env" ]] ||
  fail "failed historical-ledger publication created a candidate record"

mismatched_history_state="${tmpdir}/mismatched-history-state"
cp -a "${legacy_history_state}" "${mismatched_history_state}"
sed -i 's/^TAG=legacy-history$/TAG=other-history/' \
  "${mismatched_history_state}/releases/legacy-history.env"
if (
  export DEPLOY_STATE_DIR="${mismatched_history_state}"
  export BACKEND_IMAGE_REF="${release_backend_ref}"
  export FRONTEND_IMAGE_REF="${release_frontend_ref}"
  export ADMIN_IMAGE_REF="${release_admin_ref}"
  require_release_tag_identity_available future-release
) >"${tmpdir}/guard-mismatched-history.out" 2>"${tmpdir}/guard-mismatched-history.err"; then
  fail "release identity guard accepted a logged tag whose per-tag TAG differed"
fi
grep -q 'release activation log tag legacy-history does not match its immutable record' \
  "${tmpdir}/guard-mismatched-history.err" ||
  fail "historical tag mismatch did not identify the semantic ledger inconsistency"

if (
  export DEPLOY_STATE_DIR="${release_state_dir}"
  export BACKEND_IMAGE_REF=ghcr.io/stuhelper/backend@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
  export FRONTEND_IMAGE_REF="${release_frontend_ref}"
  export ADMIN_IMAGE_REF="${release_admin_ref}"
  require_release_tag_identity_available contract-release
) >"${tmpdir}/guard-conflicting-release.log" 2>&1; then
  fail "pre-deploy release identity guard accepted a reused tag with a different image tuple"
fi
grep -q 'release field BACKEND_IMAGE_REF does not match existing immutable release record' "${tmpdir}/guard-conflicting-release.log" ||
  fail "pre-deploy conflicting release-tag rejection did not identify the changed image"
[[ "$(wc -l <"${release_state_dir}/releases.log")" == "2" ]] ||
  fail "failed read-only release identity validation changed the activation log"

assert_missing_immutable_record_rejected() {
  local state_dir="$1"
  local expected_evidence="$2"
  local output_file="$3"
  if (
    export DEPLOY_STATE_DIR="${state_dir}"
    export BACKEND_IMAGE_REF="${release_backend_ref}"
    export FRONTEND_IMAGE_REF="${release_frontend_ref}"
    export ADMIN_IMAGE_REF="${release_admin_ref}"
    require_release_tag_identity_available contract-release
  ) >"${output_file}" 2>&1; then
    fail "release identity guard accepted a previously used tag whose immutable record was missing"
  fi
  grep -q 'was previously used but its immutable record is missing' "${output_file}" ||
    fail "missing immutable release rejection did not report ledger inconsistency"
  grep -q "${expected_evidence}" "${output_file}" ||
    fail "missing immutable release rejection did not identify ${expected_evidence} evidence"
  [[ ! -e "${state_dir}/releases/contract-release.env" ]] ||
    fail "read-only ledger inconsistency check recreated a missing immutable record"
}

log_only_release_state="${tmpdir}/log-only-release-state"
mkdir -p "${log_only_release_state}/releases"
cp "${release_state_dir}/releases.log" "${log_only_release_state}/releases.log"
chmod 0600 "${log_only_release_state}/releases.log"
assert_missing_immutable_record_rejected \
  "${log_only_release_state}" \
  'releases.log' \
  "${tmpdir}/guard-missing-record-log-evidence.log"

current_only_release_state="${tmpdir}/current-only-release-state"
mkdir -p "${current_only_release_state}/releases"
cp "${release_state_dir}/current-release.env" "${current_only_release_state}/current-release.env"
chmod 0600 "${current_only_release_state}/current-release.env"
assert_missing_immutable_record_rejected \
  "${current_only_release_state}" \
  'current-release.env' \
  "${tmpdir}/guard-missing-record-current-evidence.log"

current_only_intervening_state="${tmpdir}/current-only-intervening-state"
mkdir -p "${current_only_intervening_state}/releases"
cp "${release_state_dir}/current-release.env" "${current_only_intervening_state}/current-release.env"
chmod 0600 "${current_only_intervening_state}/current-release.env"
if (
  export DEPLOY_STATE_DIR="${current_only_intervening_state}"
  export BACKEND_IMAGE_REF="${release_backend_ref}"
  export FRONTEND_IMAGE_REF="${release_frontend_ref}"
  export ADMIN_IMAGE_REF="${release_admin_ref}"
  require_release_tag_identity_available intervening-release
) >"${tmpdir}/guard-current-only-intervening.log" 2>&1; then
  fail "release identity guard allowed a new tag to overwrite sole current-release evidence"
fi
grep -q 'release tag contract-release was previously used but its immutable record is missing' \
  "${tmpdir}/guard-current-only-intervening.log" ||
  fail "current-only ledger corruption did not identify the historical release whose evidence would be lost"
[[ ! -e "${current_only_intervening_state}/releases/intervening-release.env" ]] ||
  fail "current-only consistency check created candidate release state"
unset -f assert_missing_immutable_record_rejected

if (
  export DEPLOY_STATE_DIR="${release_state_dir}"
  export BACKEND_IMAGE_REF=ghcr.io/stuhelper/backend@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
  export FRONTEND_IMAGE_REF="${release_frontend_ref}"
  export ADMIN_IMAGE_REF="${release_admin_ref}"
  record_release contract-release
) >"${tmpdir}/record-conflicting-release.log" 2>&1; then
  fail "release recorder accepted a reused tag with a different image tuple"
fi
grep -q 'release field BACKEND_IMAGE_REF does not match existing immutable release record' "${tmpdir}/record-conflicting-release.log" ||
  fail "conflicting release-tag rejection did not identify the changed image"
[[ "$(stat -c '%i' "${immutable_release_path}")" == "${immutable_inode_before}" ]] ||
  fail "conflicting release attempt replaced the immutable record"
[[ "$(sha256sum "${immutable_release_path}" | cut -d ' ' -f 1)" == "${immutable_checksum_before}" ]] ||
  fail "conflicting release attempt changed the immutable payload"
[[ "$(wc -l <"${release_state_dir}/releases.log")" == "2" ]] ||
  fail "failed conflicting release attempt changed the activation log"
if find "${release_state_dir}" -type f -name '.*.??????' -print -quit | grep -q .; then
  fail "atomic release recording left a temporary file behind"
fi

if (
  export DEPLOY_STATE_DIR="${tmpdir}/mutable-release-state"
  export BACKEND_IMAGE_REF=ghcr.io/stuhelper/backend:sha-contract
  export FRONTEND_IMAGE_REF="${release_frontend_ref}"
  export ADMIN_IMAGE_REF="${release_admin_ref}"
  record_release mutable-release
) >"${tmpdir}/record-mutable.log" 2>&1; then
  fail "release recorder accepted a mutable application image tag"
fi
grep -q 'BACKEND_IMAGE_REF must be a complete image@sha256 digest reference' "${tmpdir}/record-mutable.log" ||
  fail "release recorder did not report its digest-only boundary"
[[ ! -e "${tmpdir}/mutable-release-state/current-release.env" ]] ||
  fail "release recorder published state before rejecting a mutable image"

if (
  export DEPLOY_STATE_DIR="${tmpdir}/unsafe-release-state"
  export BACKEND_IMAGE_REF="${release_backend_ref}"
  export FRONTEND_IMAGE_REF="${release_frontend_ref}"
  export ADMIN_IMAGE_REF="${release_admin_ref}"
  record_release '../escape'
) >"${tmpdir}/record-unsafe-release.log" 2>&1; then
  fail "release recorder accepted a path-traversing tag"
fi
grep -q 'release tag must be 1-128 characters' "${tmpdir}/record-unsafe-release.log" ||
  fail "release recorder did not report the canonical tag constraint"
[[ ! -e "${tmpdir}/unsafe-release-state" ]] ||
  fail "release recorder created state before rejecting an unsafe tag"

: >"${tmpdir}/shared.env"
: >"${tmpdir}/generated.env"
: >"${tmpdir}/generated-secrets.env"
export ENV_FILE="${tmpdir}/shared.env"
export GENERATED_ENV_FILE="${tmpdir}/generated.env"
export GENERATED_SECRET_ENV_FILE="${tmpdir}/generated-secrets.env"
export GENERATED_OBS_DIR="${tmpdir}/observability"
export BASH_ENV=/dev/null
export ENV=/dev/null
load_env_preserving BASH_ENV ENV
[[ ! -v BASH_ENV && ! -v ENV ]] ||
  fail "load_env_preserving allowed shell startup hooks to survive"

printf 'REGISTRY=ghcr.io\n' >"${tmpdir}/remote.env"
export BASH_ENV=/dev/null
export ENV=/dev/null
load_remote_deploy_config "${tmpdir}/remote.env"
[[ "${REGISTRY}" == "ghcr.io" ]] ||
  fail "load_remote_deploy_config did not load an allowed control-plane key"
[[ ! -v BASH_ENV && ! -v ENV ]] ||
  fail "load_remote_deploy_config allowed shell startup hooks to survive"

cat >"${tmpdir}/local-state-shared.env" <<'EOF'
LOCAL_STATE_DIR=
EOF
: >"${tmpdir}/local-state-generated.env"
: >"${tmpdir}/local-state-generated-secrets.env"
# Variables in the command string are intentionally expanded by the isolated child shell.
# shellcheck disable=SC2016
if ! env -i \
  PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin \
  LOCAL_STATE_DIR=/var/lib/stuhelper \
  ENV_FILE="${tmpdir}/local-state-shared.env" \
  SECRETS_ENV_FILE= \
  GENERATED_ENV_FILE="${tmpdir}/local-state-generated.env" \
  GENERATED_SECRET_ENV_FILE="${tmpdir}/local-state-generated-secrets.env" \
  /bin/bash --noprofile --norc -c '
    set -euo pipefail
    [[ -z "${HOME+x}" ]]
    source "$1"
    load_env_preserving LOCAL_STATE_DIR
    [[ "${LOCAL_STATE_DIR}" == "/var/lib/stuhelper" ]]
    [[ "${POSTGRES_WAL_RESTORE_DIR}" == "/var/lib/stuhelper/postgres/wal-restore" ]]
  ' bash "${REPO_ROOT}/infra/ops/lib/common.sh"; then
  fail "load_env_preserving did not retain the protected local-state path without HOME"
fi

printf '[environment-loader-security-contract] all assertions passed\n'
