#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
VALIDATOR="${REPO_ROOT}/infra/ops/validate-ci-deploy-inputs.sh"
RESOLVER="${REPO_ROOT}/infra/ops/resolve-attested-release-images.sh"
RELEASE_VERIFIER="${REPO_ROOT}/infra/ops/verify-github-release.sh"
REMOTE_CI_RELEASE="${REPO_ROOT}/infra/ops/remote-ci-release.sh"
DEPLOY_WORKFLOW="${REPO_ROOT}/.github/workflows/deploy.yml"
ROLLBACK_WORKFLOW="${REPO_ROOT}/.github/workflows/rollback.yml"
PUBLISH_WORKFLOW="${REPO_ROOT}/.github/workflows/publish-images.yml"
PUBLISHER="${REPO_ROOT}/infra/ops/publish-scanned-image.sh"
VALID_SHA="0123456789abcdef0123456789abcdef01234567"
DIGEST="sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

fail() {
  printf '[ci-deploy-security-contract][error] %s\n' "$*" >&2
  exit 1
}

assert_contains() {
  local file="$1"
  local pattern="$2"
  grep -Eq -- "${pattern}" "${file}" ||
    fail "expected ${file} to contain pattern: ${pattern}"
}

assert_not_contains() {
  local file="$1"
  local pattern="$2"
  if grep -Eq -- "${pattern}" "${file}"; then
    fail "expected ${file} not to contain pattern: ${pattern}"
  fi
}

run_validator() {
  DEPLOY_ENVIRONMENT="${DEPLOY_ENVIRONMENT_OVERRIDE:-production}" \
    DEPLOY_TARGET_HOST="${DEPLOY_TARGET_HOST_OVERRIDE:-deploy.example.com}" \
    DEPLOY_TARGET_PORT="${DEPLOY_TARGET_PORT_OVERRIDE:-22}" \
    DEPLOY_TARGET_USER="${DEPLOY_TARGET_USER_OVERRIDE:-stuhelper}" \
    DEPLOY_TARGET_APP_DIR="${DEPLOY_TARGET_APP_DIR_OVERRIDE:-/srv/stuhelper}" \
    DEPLOY_TARGET_SSH_KEY="${DEPLOY_TARGET_SSH_KEY_OVERRIDE:-private-key-sentinel}" \
    DEPLOY_TARGET_SSH_KNOWN_HOSTS="${DEPLOY_TARGET_SSH_KNOWN_HOSTS_OVERRIDE:-known-host-sentinel}" \
    TARGET_SHA="${TARGET_SHA_OVERRIDE:-${VALID_SHA}}" \
    "${VALIDATOR}"
}

expect_validator_failure() {
  local name="$1"
  shift
  if env \
    DEPLOY_ENVIRONMENT=production \
    DEPLOY_TARGET_HOST=deploy.example.com \
    DEPLOY_TARGET_PORT=22 \
    DEPLOY_TARGET_USER=stuhelper \
    DEPLOY_TARGET_APP_DIR=/srv/stuhelper \
    DEPLOY_TARGET_SSH_KEY=private-key-sentinel \
    DEPLOY_TARGET_SSH_KNOWN_HOSTS=known-host-sentinel \
    TARGET_SHA="${VALID_SHA}" \
    "$@" \
    "${VALIDATOR}" >/dev/null 2>&1; then
    fail "validator accepted invalid case: ${name}"
  fi
}

bash -n "${VALIDATOR}"
bash -n "${RESOLVER}"
bash -n "${RELEASE_VERIFIER}"
bash -n "${REMOTE_CI_RELEASE}"

validator_output="$(run_validator)"
[[ "${validator_output}" == *"deployment inputs validated"* ]] ||
  fail "validator did not confirm valid inputs"
[[ "${validator_output}" != *"private-key-sentinel"* ]] ||
  fail "validator leaked the SSH private key"
[[ "${validator_output}" != *"known-host-sentinel"* ]] ||
  fail "validator leaked known_hosts"

DEPLOY_TARGET_HOST_OVERRIDE=192.0.2.10 run_validator >/dev/null
DEPLOY_TARGET_PORT_OVERRIDE=65535 run_validator >/dev/null

expect_validator_failure "shell metacharacter in host" \
  DEPLOY_TARGET_HOST='deploy.example.com;id'
expect_validator_failure "invalid IPv4 octet" \
  DEPLOY_TARGET_HOST='256.0.0.1'
expect_validator_failure "non-canonical IPv4" \
  DEPLOY_TARGET_HOST='192.168.001.1'
expect_validator_failure "invalid Linux user" \
  DEPLOY_TARGET_USER='root;id'
expect_validator_failure "port zero" \
  DEPLOY_TARGET_PORT='0'
expect_validator_failure "port overflow" \
  DEPLOY_TARGET_PORT='65536'
expect_validator_failure "relative app directory" \
  DEPLOY_TARGET_APP_DIR='srv/stuhelper'
expect_validator_failure "root app directory" \
  DEPLOY_TARGET_APP_DIR='/'
expect_validator_failure "parent app directory segment" \
  DEPLOY_TARGET_APP_DIR='/srv/../root'
expect_validator_failure "quoted app directory" \
  DEPLOY_TARGET_APP_DIR="/srv/stuhelper'bad"
expect_validator_failure "mutable release identifier" \
  TARGET_SHA='latest'
expect_validator_failure "unknown environment" \
  DEPLOY_ENVIRONMENT='preview'

tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT
mkdir -p "${tmpdir}/bin"

cat >"${tmpdir}/bin/docker" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
[[ "$1 $2 $3" == "buildx imagetools inspect" ]] || exit 91
printf '{"digest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}\n'
EOF

cat >"${tmpdir}/bin/gh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >>"${GH_CALLS_FILE}"
[[ "${GH_FORCE_FAILURE:-false}" != "true" ]]
EOF

chmod +x "${tmpdir}/bin/docker" "${tmpdir}/bin/gh"
resolver_output="${tmpdir}/resolver-output"
gh_calls="${tmpdir}/gh-calls"

PATH="${tmpdir}/bin:${PATH}" \
  TARGET_SHA="${VALID_SHA}" \
  DEPLOY_ENVIRONMENT=production \
  GITHUB_REPOSITORY=StuHelper/StuHelper \
  IMAGE_NAMESPACE=ghcr.io/stuhelper \
  GITHUB_OUTPUT="${resolver_output}" \
  GH_CALLS_FILE="${gh_calls}" \
  "${RESOLVER}" >/dev/null

assert_contains "${resolver_output}" "^backend_image_ref=ghcr\\.io/stuhelper/backend@${DIGEST}$"
assert_contains "${resolver_output}" "^frontend_image_ref=ghcr\\.io/stuhelper/frontend@${DIGEST}$"
assert_contains "${resolver_output}" "^admin_image_ref=ghcr\\.io/stuhelper/admin@${DIGEST}$"
[[ "$(wc -l <"${resolver_output}")" -eq 3 ]] ||
  fail "resolver must emit exactly three image references"
assert_contains "${gh_calls}" '--repo StuHelper/StuHelper'
assert_contains "${gh_calls}" '--signer-workflow StuHelper/StuHelper/\.github/workflows/publish-images\.yml'
assert_contains "${gh_calls}" "--source-digest ${VALID_SHA}"
assert_contains "${gh_calls}" '--source-ref refs/heads/main'
assert_contains "${gh_calls}" '--deny-self-hosted-runners'
assert_contains "${gh_calls}" "oci://ghcr\\.io/stuhelper/backend@${DIGEST}"

if PATH="${tmpdir}/bin:${PATH}" \
  TARGET_SHA="${VALID_SHA}" \
  DEPLOY_ENVIRONMENT=production \
  GITHUB_REPOSITORY=StuHelper/StuHelper \
  IMAGE_NAMESPACE=ghcr.io/stuhelper \
  GITHUB_OUTPUT="${tmpdir}/failed-output" \
  GH_CALLS_FILE="${tmpdir}/failed-gh-calls" \
  GH_FORCE_FAILURE=true \
  "${RESOLVER}" >/dev/null 2>&1; then
  fail "resolver accepted an image without trusted provenance"
fi

assert_contains "${DEPLOY_WORKFLOW}" 'attestations: read'
assert_contains "${ROLLBACK_WORKFLOW}" 'attestations: read'
assert_contains "${DEPLOY_WORKFLOW}" 'checks: read'
assert_contains "${ROLLBACK_WORKFLOW}" 'checks: read'
assert_contains "${DEPLOY_WORKFLOW}" 'workflow_call:'
assert_contains "${DEPLOY_WORKFLOW}" 'promotion_mode:'
assert_contains "${DEPLOY_WORKFLOW}" 'PRODUCTION_PROMOTION_MODE: \$\{\{ inputs\.promotion_mode \}\}'
assert_contains "${DEPLOY_WORKFLOW}" 'RELEASE_OPERATION: forward'
assert_not_contains "${DEPLOY_WORKFLOW}" 'skip_staging_gate'
assert_contains "${DEPLOY_WORKFLOW}" 'verify-github-release\.sh'
assert_contains "${ROLLBACK_WORKFLOW}" 'verify-github-release\.sh'
[[ "$(grep -c 'verify-github-release\.sh' "${DEPLOY_WORKFLOW}")" -eq 2 ]] ||
  fail "deploy workflow must verify before approval and revalidate before SSH"
[[ "$(grep -c 'verify-github-release\.sh' "${ROLLBACK_WORKFLOW}")" -eq 2 ]] ||
  fail "rollback workflow must verify before approval and revalidate before SSH"
assert_contains "${DEPLOY_WORKFLOW}" 'Revalidate the release after environment approval'
assert_contains "${ROLLBACK_WORKFLOW}" 'Revalidate the rollback controller after environment approval'
assert_contains "${DEPLOY_WORKFLOW}" 'validate-ci-deploy-inputs\.sh'
assert_contains "${ROLLBACK_WORKFLOW}" 'validate-ci-deploy-inputs\.sh'
assert_contains "${DEPLOY_WORKFLOW}" 'resolve-attested-release-images\.sh'
assert_contains "${ROLLBACK_WORKFLOW}" 'resolve-attested-release-images\.sh'
assert_contains "${DEPLOY_WORKFLOW}" 'steps\.release-images\.outputs\.backend_image_ref'
assert_contains "${ROLLBACK_WORKFLOW}" 'steps\.release-images\.outputs\.backend_image_ref'
assert_contains "${ROLLBACK_WORKFLOW}" 'commit_sha:'
assert_contains "${ROLLBACK_WORKFLOW}" 'reason:'
assert_not_contains "${DEPLOY_WORKFLOW}" 'commit_sha:'
assert_not_contains "${ROLLBACK_WORKFLOW}" 'release_tag:'
assert_not_contains "${ROLLBACK_WORKFLOW}" 'TARGET_TAG'
assert_contains "${DEPLOY_WORKFLOW}" 'ref: \$\{\{ fromJSON\(toJSON\(job\)\)\.workflow_sha \}\}'
assert_contains "${ROLLBACK_WORKFLOW}" 'ref: \$\{\{ fromJSON\(toJSON\(job\)\)\.workflow_sha \}\}'
assert_not_contains "${DEPLOY_WORKFLOW}" 'github\.workflow_sha'
assert_not_contains "${ROLLBACK_WORKFLOW}" 'github\.workflow_sha'
assert_not_contains "${ROLLBACK_WORKFLOW}" 'path: rollback-release'
assert_not_contains "${ROLLBACK_WORKFLOW}" '"\$\{GITHUB_WORKSPACE\}/rollback-release"'
assert_contains "${ROLLBACK_WORKFLOW}" 'Build the trusted rollback-controller bundle'
assert_contains "${DEPLOY_WORKFLOW}" 'remote-ci-release\.sh deploy'
assert_contains "${ROLLBACK_WORKFLOW}" 'remote-ci-release\.sh rollback'
assert_contains "${DEPLOY_WORKFLOW}" 'GHCR_PULL_TOKEN: \$\{\{ github\.token \}\}'
assert_contains "${ROLLBACK_WORKFLOW}" 'GHCR_PULL_TOKEN: \$\{\{ github\.token \}\}'
assert_contains "${DEPLOY_WORKFLOW}" 'printf .%s\\n. "\$\{GHCR_PULL_TOKEN\}" \| ssh'
assert_contains "${ROLLBACK_WORKFLOW}" 'printf .%s\\n. "\$\{GHCR_PULL_TOKEN\}" \| ssh'
[[ "$(grep -c 'printf .%s\\n. "\${GHCR_PULL_TOKEN}" | ssh' "${DEPLOY_WORKFLOW}")" -eq 1 ]] ||
  fail "deploy workflow must pass the short-lived registry token to exactly one SSH process"
[[ "$(grep -c 'printf .%s\\n. "\${GHCR_PULL_TOKEN}" | ssh' "${ROLLBACK_WORKFLOW}")" -eq 1 ]] ||
  fail "rollback workflow must pass the short-lived registry token to exactly one SSH process"
deploy_execution_block="$(sed -n '/# Release identifiers, digest references/,/name: Record deployment result/p' "${DEPLOY_WORKFLOW}")"
rollback_execution_block="$(sed -n '/# The current controller applies/,/name: Record rollback result/p' "${ROLLBACK_WORKFLOW}")"
grep -Eq 'printf .%s\\n. "\$\{GHCR_PULL_TOKEN\}" \| ssh' <<<"${deploy_execution_block}" ||
  fail "deploy must pipe the short-lived registry token to the SSH process that runs remote-ci-release"
grep -Eq 'printf .%s\\n. "\$\{GHCR_PULL_TOKEN\}" \| ssh' <<<"${rollback_execution_block}" ||
  fail "rollback must pipe the short-lived registry token to the SSH process that runs remote-ci-release"
assert_not_contains "${DEPLOY_WORKFLOW}" "GHCR_PULL_TOKEN='"
assert_not_contains "${ROLLBACK_WORKFLOW}" "GHCR_PULL_TOKEN='"
assert_contains "${REMOTE_CI_RELEASE}" 'REGISTRY_AUTH_MODE:-.*workflow-token'
assert_contains "${REMOTE_CI_RELEASE}" 'docker login "\$\{REGISTRY\}" --username "\$\{registry_username\}" --password-stdin'
assert_not_contains "${REMOTE_CI_RELEASE}" 'password[^-].*registry_token'
assert_contains "${ROLLBACK_WORKFLOW}" 'ROLLBACK_REVIEW_ACTOR'
assert_contains "${ROLLBACK_WORKFLOW}" 'ROLLBACK_REVIEW_REASON_B64'
assert_contains "${PUBLISH_WORKFLOW}" '^  verify-release:$'
assert_contains "${PUBLISH_WORKFLOW}" 'name: Verify trusted release checks'
assert_contains "${PUBLISH_WORKFLOW}" 'verify-github-release\.sh'
assert_contains "${PUBLISH_WORKFLOW}" 'RELEASE_OPERATION: publish'
assert_contains "${PUBLISH_WORKFLOW}" 'PRODUCTION_PROMOTION_MODE: not-applicable'
assert_contains "${PUBLISH_WORKFLOW}" '^    needs: verify-release$'
assert_contains "${PUBLISH_WORKFLOW}" 'checks: read'
assert_contains "${PUBLISH_WORKFLOW}" 'name: Scan the candidate image'
assert_contains "${PUBLISH_WORKFLOW}" 'name: Generate the candidate SBOM'
assert_contains "${PUBLISH_WORKFLOW}" 'format: cyclonedx'
assert_contains "${PUBLISH_WORKFLOW}" "steps\.existing\.outputs\.found == 'true'"
assert_contains "${PUBLISH_WORKFLOW}" "format\('\{0\}@\{1\}', matrix\.image, steps\.existing\.outputs\.digest\)"
assert_contains "${PUBLISH_WORKFLOW}" 'sbom-path: sbom-\$\{\{ matrix\.name \}\}\.cdx\.json'
assert_contains "${PUBLISH_WORKFLOW}" 'name: Upload the SBOM evidence'
assert_contains "${PUBLISH_WORKFLOW}" '^  group: publish-images$'
assert_contains "${PUBLISH_WORKFLOW}" 'cancel-in-progress: false'
assert_contains "${PUBLISH_WORKFLOW}" 'max-parallel: 1'
assert_contains "${PUBLISH_WORKFLOW}" 'SOURCE_DATE_EPOCH: \$\{\{ steps\.build-meta\.outputs\.commit_epoch \}\}'
assert_contains "${PUBLISH_WORKFLOW}" 'org\.opencontainers\.image\.created=\$\{\{ steps\.build-meta\.outputs\.build_time \}\}'
assert_contains "${PUBLISH_WORKFLOW}" 'name: Resolve a trusted immutable image'
assert_contains "${PUBLISH_WORKFLOW}" 'resolve-existing-published-image\.sh'
assert_contains "${PUBLISH_WORKFLOW}" 'steps\.existing\.outputs\.found != '\''true'\'''
assert_contains "${PUBLISH_WORKFLOW}" 'publish-scanned-image\.sh'
assert_contains "${PUBLISHER}" 'docker push "\$\{immutable_ref\}"'
assert_contains "${PUBLISHER}" 'secondary rate limit'
assert_contains "${PUBLISHER}" 'PUBLISH_MAX_ATTEMPTS:-4'
assert_contains "${PUBLISHER}" 'PUBLISH_RETRY_BASE_DELAY_SECONDS:-30'
assert_contains "${PUBLISH_WORKFLOW}" 'docker buildx imagetools create'
assert_not_contains "${PUBLISH_WORKFLOW}" 'docker/metadata-action@'
[[ "$(grep -c 'docker/build-push-action@' "${PUBLISH_WORKFLOW}")" -eq 1 ]] ||
  fail "the publish workflow must build the scanned artifact exactly once"

printf '[ci-deploy-security-contract] all assertions passed\n'
