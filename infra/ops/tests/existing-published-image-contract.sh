#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
RESOLVER="${REPO_ROOT}/infra/ops/resolve-existing-published-image.sh"
VALID_SHA="0123456789abcdef0123456789abcdef01234567"
DIGEST="sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

fail() {
  printf '[existing-published-image-contract][error] %s\n' "$*" >&2
  exit 1
}

assert_contains() {
  local file="$1"
  local pattern="$2"
  grep -Eq -- "${pattern}" "${file}" ||
    fail "expected ${file} to contain pattern: ${pattern}"
}

tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT
mkdir -p "${tmpdir}/bin"

cat >"${tmpdir}/bin/docker" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
[[ "$1 $2 $3" == "buildx imagetools inspect" ]] || exit 91
case "${DOCKER_MODE:-found}" in
  found)
    printf '{"digest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}\n'
    ;;
  invalid)
    printf '{"digest":"latest"}\n'
    ;;
  missing)
    printf 'manifest unknown\n' >&2
    exit 1
    ;;
  error)
    printf 'registry request failed: unauthorized\n' >&2
    exit 1
    ;;
  *)
    exit 92
    ;;
esac
EOF

cat >"${tmpdir}/bin/gh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >>"${GH_CALLS_FILE}"
[[ "${GH_FORCE_FAILURE:-false}" != "true" ]]
EOF

chmod +x "${tmpdir}/bin/docker" "${tmpdir}/bin/gh"
bash -n "${RESOLVER}"

run_resolver() {
  local output_file="$1"
  shift
  (
    export PATH="${tmpdir}/bin:${PATH}"
    export COMMIT_SHA="${VALID_SHA}"
    export IMAGE_NAME=ghcr.io/stuhelper/backend
    export GITHUB_REPOSITORY=StuHelper/StuHelper
    export GITHUB_OUTPUT="${output_file}"
    export GH_CALLS_FILE="${tmpdir}/gh-calls"
    if (( $# > 0 )); then
      export "$@"
    fi
    "${RESOLVER}"
  )
}

trusted_output="${tmpdir}/trusted-output"
run_resolver "${trusted_output}"
assert_contains "${trusted_output}" '^found=true$'
assert_contains "${trusted_output}" "^digest=${DIGEST}$"
assert_contains "${tmpdir}/gh-calls" '--repo StuHelper/StuHelper'
assert_contains "${tmpdir}/gh-calls" '--signer-workflow StuHelper/StuHelper/\.github/workflows/publish-images\.yml'
assert_contains "${tmpdir}/gh-calls" "--source-digest ${VALID_SHA}"
assert_contains "${tmpdir}/gh-calls" '--source-ref refs/heads/develop'
assert_contains "${tmpdir}/gh-calls" '--deny-self-hosted-runners'

missing_output="${tmpdir}/missing-output"
run_resolver "${missing_output}" DOCKER_MODE=missing
[[ "$(cat "${missing_output}")" == "found=false" ]] ||
  fail "a missing immutable tag must be reported without publishing"

if run_resolver "${tmpdir}/untrusted-output" GH_FORCE_FAILURE=true >/dev/null 2>&1; then
  fail "resolver accepted an existing image without trusted provenance"
fi
if run_resolver "${tmpdir}/registry-error-output" DOCKER_MODE=error >/dev/null 2>&1; then
  fail "resolver treated a registry error as a missing immutable tag"
fi
if run_resolver "${tmpdir}/invalid-digest-output" DOCKER_MODE=invalid >/dev/null 2>&1; then
  fail "resolver accepted an invalid manifest digest"
fi
if COMMIT_SHA=latest \
  IMAGE_NAME=ghcr.io/stuhelper/backend \
  GITHUB_REPOSITORY=StuHelper/StuHelper \
  GITHUB_OUTPUT="${tmpdir}/invalid-input-output" \
  "${RESOLVER}" >/dev/null 2>&1; then
  fail "resolver accepted a mutable release identifier"
fi

printf '[existing-published-image-contract] all assertions passed\n'
