#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
CI_FILE="${REPO_ROOT}/.gitlab-ci.yml"
SERVER_CI="${REPO_ROOT}/.gitlab/server-ci.yml"
PACKAGE_CI="${REPO_ROOT}/.gitlab/package-ci.yml"
CD_FILE="${REPO_ROOT}/.gitlab/cd.yml"
RESOLVER="${REPO_ROOT}/infra/ops/resolve-registry-release-images.sh"
REMOTE_OPERATION="${REPO_ROOT}/infra/ops/run-ci-remote-operation.sh"
VALID_SHA="0123456789abcdef0123456789abcdef01234567"
DIGEST="sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

fail() {
  printf '[gitlab-ci-supply-chain-contract][error] %s\n' "$*" >&2
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

bash -n "${RESOLVER}"
bash -n "${REMOTE_OPERATION}"

assert_contains "${CI_FILE}" '^  TRIVY_IMAGE: ghcr\.io/aquasecurity/trivy:0\.72\.0@sha256:[0-9a-f]{64}$'
assert_contains "${CI_FILE}" '^  SYFT_IMAGE: ghcr\.io/anchore/syft:v1\.44\.0@sha256:[0-9a-f]{64}$'
assert_contains "${CI_FILE}" 'image: docker:27\.5\.1-cli@sha256:[0-9a-f]{64}$'
assert_contains "${CI_FILE}" 'name: docker:27\.5\.1-dind@sha256:[0-9a-f]{64}$'
assert_contains "${CI_FILE}" 'image: node:24\.18\.0-bookworm@sha256:[0-9a-f]{64}$'
assert_contains "${CI_FILE}" 'image: mcr\.microsoft\.com/playwright:v1\.58\.2-noble@sha256:[0-9a-f]{64}$'
assert_contains "${CI_FILE}" 'image: semgrep/semgrep:1\.128\.0@sha256:[0-9a-f]{64}$'
assert_contains "${CI_FILE}" 'name: ghcr\.io/gitleaks/gitleaks:v8\.24\.3@sha256:[0-9a-f]{64}$'
assert_contains "${SERVER_CI}" 'image: golang:1\.26\.5-bookworm@sha256:[0-9a-f]{64}$'
assert_contains "${SERVER_CI}" 'name: postgres:18\.3@sha256:[0-9a-f]{64}$'
assert_contains "${SERVER_CI}" 'name: redis:8\.6\.2@sha256:[0-9a-f]{64}$'
assert_contains "${CD_FILE}" 'image: alpine:3\.24@sha256:[0-9a-f]{64}$'

assert_not_contains "${CI_FILE}" 'curl .*\|[[:space:]]*tar'
assert_not_contains "${CI_FILE}" 'raw\.githubusercontent\.com/.*/install\.sh'
assert_not_contains "${SERVER_CI}" 'deb\.nodesource\.com'
assert_contains "${SERVER_CI}" 'node-v24\.18\.0-linux-x64'
assert_contains "${SERVER_CI}" 'sha256sum --check --strict'

[[ "$(grep -c -- '--metadata-file .*image-metadata\.json' "${CI_FILE}")" -eq 3 ]] ||
  fail "all three image builds must emit BuildKit metadata"
# The literal GitLab variable expression is the contract under test.
# shellcheck disable=SC2016
[[ "$(grep -c -- 'candidate-\${CI_COMMIT_SHA}' "${CI_FILE}")" -eq 3 ]] ||
  fail "all three candidate tags must use the full commit SHA"
[[ "$(grep -c -- '"containerimage.digest"' "${CI_FILE}")" -eq 3 ]] ||
  fail "all three image builds must export the registry digest"
assert_not_contains "${CI_FILE}" ':ci-\$\{CI_COMMIT_SHORT_SHA\}'
assert_contains "${CI_FILE}" '"\$\{BACKEND_CANDIDATE_REF\}"'
assert_contains "${CI_FILE}" '"\$\{FRONTEND_CANDIDATE_REF\}"'
assert_contains "${CI_FILE}" '"\$\{ADMIN_CANDIDATE_REF\}"'
[[ "$(grep -c -- 'cyclonedx:' "${CI_FILE}")" -eq 3 ]] ||
  fail "all three SBOM jobs must publish CycloneDX reports"

assert_not_contains "${PACKAGE_CI}" 'CI_COMMIT_SHORT_SHA'
assert_contains "${PACKAGE_CI}" 'immutable_ref="\$\{BACKEND_IMAGE\}:\$\{CI_COMMIT_SHA\}"'
assert_contains "${PACKAGE_CI}" 'BACKEND_IMAGE_REF=%s@%s'
assert_contains "${PACKAGE_CI}" 'FRONTEND_IMAGE_REF=%s@%s'
assert_contains "${PACKAGE_CI}" 'ADMIN_IMAGE_REF=%s@%s'
assert_contains "${PACKAGE_CI}" '^  - sbom_backend$'
assert_contains "${PACKAGE_CI}" '^  - sbom_frontend$'
assert_contains "${PACKAGE_CI}" '^  - sbom_admin$'

assert_contains "${CD_FILE}" 'run-ci-remote-operation\.sh deploy'
assert_contains "${CD_FILE}" 'run-ci-remote-operation\.sh verify'
assert_contains "${CD_FILE}" 'run-ci-remote-operation\.sh rollback'
assert_contains "${CD_FILE}" 'resolve-registry-release-images\.sh'
assert_contains "${REMOTE_OPERATION}" 'StrictHostKeyChecking=yes'
assert_contains "${REMOTE_OPERATION}" 'validate-ci-deploy-inputs\.sh'
assert_contains "${REMOTE_OPERATION}" 'must be an immutable OCI image digest reference'

tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT
mkdir -p "${tmpdir}/bin"

cat >"${tmpdir}/bin/docker" <<EOF
#!/usr/bin/env bash
set -euo pipefail
[[ "\$1 \$2 \$3" == "buildx imagetools inspect" ]] || exit 91
printf 'Name: test\\nMediaType: application/vnd.oci.image.manifest.v1+json\\nDigest:    ${DIGEST}\\n'
EOF
chmod +x "${tmpdir}/bin/docker"

output_file="${tmpdir}/rollback-images.env"
PATH="${tmpdir}/bin:${PATH}" \
  TARGET_SHA="${VALID_SHA}" \
  BACKEND_IMAGE=registry.example.com/stuhelper/backend \
  FRONTEND_IMAGE=registry.example.com/stuhelper/frontend \
  ADMIN_IMAGE=registry.example.com/stuhelper/admin \
  "${RESOLVER}" "${output_file}" >/dev/null

assert_contains "${output_file}" "^ROLLBACK_BACKEND_IMAGE_REF=registry\\.example\\.com/stuhelper/backend@${DIGEST}$"
assert_contains "${output_file}" "^ROLLBACK_FRONTEND_IMAGE_REF=registry\\.example\\.com/stuhelper/frontend@${DIGEST}$"
assert_contains "${output_file}" "^ROLLBACK_ADMIN_IMAGE_REF=registry\\.example\\.com/stuhelper/admin@${DIGEST}$"
[[ "$(wc -l <"${output_file}")" -eq 3 ]] ||
  fail "resolver must emit exactly three rollback image references"

printf '[gitlab-ci-supply-chain-contract] all assertions passed\n'
