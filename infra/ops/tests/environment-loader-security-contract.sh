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

for key in BASH_ENV ENV; do
  startup_env="${tmpdir}/${key}.env"
  printf '%s=/dev/null\n' "${key}" >"${startup_env}"
  if (source_env_file "${startup_env}") >"${tmpdir}/${key}.log" 2>&1; then
    fail "source_env_file accepted forbidden shell startup variable ${key}"
  fi
  grep -q "shell startup variable ${key} is not allowed" "${tmpdir}/${key}.log" ||
    fail "${key} rejection did not explain the protected boundary"
done

printf 'STACK_NAME=source-env-contract\n' >"${tmpdir}/safe.env"
export BASH_ENV=/dev/null
export ENV=/dev/null
source_env_file "${tmpdir}/safe.env"
[[ "${STACK_NAME}" == "source-env-contract" ]] ||
  fail "source_env_file did not export a validated assignment"
[[ ! -v BASH_ENV && ! -v ENV ]] ||
  fail "source_env_file allowed inherited shell startup hooks to survive"

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
grep -q 'environment key PATH is not allowed in this file' "${tmpdir}/bootstrap-secret-error.log" ||
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
