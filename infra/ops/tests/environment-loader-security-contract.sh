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
  DOCKER_TLS_VERIFY; do
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

release_state_dir="${tmpdir}/release-state"
fresh_release_guard_state="${tmpdir}/fresh-release-guard-state"
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
  export DEPLOY_STATE_DIR="${release_state_dir}"
  export BACKEND_IMAGE_REF="${release_backend_ref}"
  export FRONTEND_IMAGE_REF="${release_frontend_ref}"
  export ADMIN_IMAGE_REF="${release_admin_ref}"
  record_release contract-release
)
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
