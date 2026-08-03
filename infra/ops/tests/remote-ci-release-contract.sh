#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
TARGET_SCRIPT="${REPO_ROOT}/infra/ops/remote-ci-release.sh"

fail() {
  printf '[remote-ci-release-contract][error] %s\n' "$*" >&2
  exit 1
}

tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT
mkdir -p "${tmpdir}/ops/lib" "${tmpdir}/bin" "${tmpdir}/state"
cp "${TARGET_SCRIPT}" "${tmpdir}/ops/remote-ci-release.sh"

cat >"${tmpdir}/ops/lib/common.sh" <<'EOF'
#!/usr/bin/env bash
die() {
  printf '[test-common][error] %s\n' "$*" >&2
  exit 1
}
log() {
  printf '[test-common] %s\n' "$*"
}
require_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "missing command: $1"
}
require_safe_release_tag() {
  [[ "${1:-}" =~ ^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$ ]] || die "release tag must be 1-128 characters"
}
load_remote_deploy_config() {
  REGISTRY="${TEST_REGISTRY:-ghcr.io}"
  REGISTRY_AUTH_MODE="${TEST_REGISTRY_AUTH_MODE:-workflow-token}"
  DEPLOY_STATE_DIR="${TEST_DEPLOY_STATE_DIR:?}"
  export REGISTRY REGISTRY_AUTH_MODE DEPLOY_STATE_DIR
}
EOF

cat >"${tmpdir}/bin/docker" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
[[ "$*" == "login ghcr.io --username ${CI_REGISTRY_USERNAME} --password-stdin" ]] || exit 91
printf '%s\n' "$*" >"${TEST_DOCKER_ARGS_FILE}"
printf '%s\n' "${DOCKER_CONFIG}" >"${TEST_DOCKER_CONFIG_FILE}"
IFS= read -r token
printf '%s' "${token}" >"${TEST_DOCKER_STDIN_FILE}"
printf '{}' >"${DOCKER_CONFIG}/config.json"
EOF
chmod +x "${tmpdir}/bin/docker"

cat >"${tmpdir}/ops/operation-stub.sh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
[[ "${CI_REGISTRY_LOGIN_READY:-false}" == "true" ]]
[[ -f "${DOCKER_CONFIG}/config.json" ]]
basename "$0" >>"${TEST_OPERATION_LOG}"
EOF
chmod +x "${tmpdir}/ops/operation-stub.sh"
for name in remote-preflight.sh remote-prod-deploy.sh remote-prod-rollback.sh smoke-check.sh observability-smoke-check.sh; do
  cp "${tmpdir}/ops/operation-stub.sh" "${tmpdir}/ops/${name}"
done

export TEST_DEPLOY_STATE_DIR="${tmpdir}/state"
export TEST_DOCKER_ARGS_FILE="${tmpdir}/docker-args"
export TEST_DOCKER_CONFIG_FILE="${tmpdir}/docker-config-path"
export TEST_DOCKER_STDIN_FILE="${tmpdir}/docker-stdin"
export TEST_OPERATION_LOG="${tmpdir}/operation-log"

token_sentinel='short-lived-token-sentinel'
output_file="${tmpdir}/output"
if ! printf '%s\n' "${token_sentinel}" | \
  PATH="${tmpdir}/bin:${PATH}" \
  CI_REGISTRY_USERNAME=Xauryan \
  TAG=0123456789abcdef0123456789abcdef01234567 \
  /bin/bash "${tmpdir}/ops/remote-ci-release.sh" deploy >"${output_file}" 2>&1; then
  fail "valid deploy operation failed"
fi

[[ "$(<"${TEST_DOCKER_STDIN_FILE}")" == "${token_sentinel}" ]] ||
  fail "short-lived token was not delivered through Docker standard input"
[[ "$(<"${TEST_DOCKER_ARGS_FILE}")" == "login ghcr.io --username Xauryan --password-stdin" ]] ||
  fail "Docker login arguments are not constrained"
expected_deploy_sequence=$'remote-preflight.sh\nremote-prod-deploy.sh\nsmoke-check.sh\nobservability-smoke-check.sh'
[[ "$(<"${TEST_OPERATION_LOG}")" == "${expected_deploy_sequence}" ]] ||
  fail "deploy did not run preflight, deploy, and both smoke checks in order"
registry_config_dir="$(<"${TEST_DOCKER_CONFIG_FILE}")"
[[ "${registry_config_dir}" == "${TEST_DEPLOY_STATE_DIR}"/registry-auth.* ]] ||
  fail "temporary Docker config was created outside the deploy state directory"
[[ ! -e "${registry_config_dir}" ]] ||
  fail "temporary Docker config was not deleted after deploy"
if grep -Fq "${token_sentinel}" "${output_file}"; then
  fail "short-lived token leaked to process output"
fi

: >"${TEST_OPERATION_LOG}"
if ! printf '%s\n' "${token_sentinel}" | \
  PATH="${tmpdir}/bin:${PATH}" \
  CI_REGISTRY_USERNAME='github-actions[bot]' \
  ROLLBACK_TAG=0123456789abcdef0123456789abcdef01234567 \
  /bin/bash "${tmpdir}/ops/remote-ci-release.sh" rollback >"${output_file}" 2>&1; then
  fail "valid rollback operation failed"
fi
expected_rollback_sequence=$'remote-prod-rollback.sh\nsmoke-check.sh\nobservability-smoke-check.sh'
[[ "$(<"${TEST_OPERATION_LOG}")" == "${expected_rollback_sequence}" ]] ||
  fail "rollback did not run rollback and both smoke checks in order"
registry_config_dir="$(<"${TEST_DOCKER_CONFIG_FILE}")"
[[ ! -e "${registry_config_dir}" ]] ||
  fail "temporary Docker config was not deleted after rollback"

rm -f "${TEST_DOCKER_ARGS_FILE}"
: >"${TEST_OPERATION_LOG}"
if printf '%s\n' "${token_sentinel}" | \
  PATH="${tmpdir}/bin:${PATH}" \
  CI_REGISTRY_USERNAME=Xauryan \
  TAG='../escape' \
  /bin/bash "${tmpdir}/ops/remote-ci-release.sh" deploy >"${tmpdir}/unsafe-tag.out" 2>"${tmpdir}/unsafe-tag.err"; then
  fail "remote CI release accepted an unsafe deployment tag"
fi
grep -q 'release tag must be 1-128 characters' "${tmpdir}/unsafe-tag.err" ||
  fail "remote CI release did not report its early release-tag gate"
[[ ! -e "${TEST_DOCKER_ARGS_FILE}" && ! -s "${TEST_OPERATION_LOG}" ]] ||
  fail "remote CI release performed registry or deployment side effects before rejecting an unsafe tag"

if printf '%s\n' "${token_sentinel}" | \
  PATH="${tmpdir}/bin:${PATH}" \
  CI_REGISTRY_USERNAME=Xauryan \
  TAG=0123456789abcdef0123456789abcdef01234567 \
  TEST_REGISTRY_AUTH_MODE=persistent-secret \
  /bin/bash "${tmpdir}/ops/remote-ci-release.sh" deploy >/dev/null 2>&1; then
  fail "persistent registry credentials were accepted by the CI release wrapper"
fi
if printf '%s\n' "${token_sentinel}" | \
  PATH="${tmpdir}/bin:${PATH}" \
  CI_REGISTRY_USERNAME='invalid;actor' \
  TAG=0123456789abcdef0123456789abcdef01234567 \
  /bin/bash "${tmpdir}/ops/remote-ci-release.sh" deploy >/dev/null 2>&1; then
  fail "an unsafe registry username was accepted"
fi
if PATH="${tmpdir}/bin:${PATH}" \
  CI_REGISTRY_USERNAME=Xauryan \
  TAG=0123456789abcdef0123456789abcdef01234567 \
  /bin/bash "${tmpdir}/ops/remote-ci-release.sh" deploy </dev/null >/dev/null 2>&1; then
  fail "an empty short-lived token was accepted"
fi

printf '[remote-ci-release-contract] all assertions passed\n'
