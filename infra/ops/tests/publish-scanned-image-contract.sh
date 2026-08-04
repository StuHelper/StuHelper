#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
PUBLISHER="${REPO_ROOT}/infra/ops/publish-scanned-image.sh"
VALID_SHA="0123456789abcdef0123456789abcdef01234567"
DIGEST="sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

fail() {
  printf '[publish-scanned-image-contract][error] %s\n' "$*" >&2
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

printf '%s\n' "$*" >>"${DOCKER_CALLS_FILE}"
if [[ "$1" == "push" ]]; then
  count=0
  if [[ -s "${DOCKER_PUSH_COUNT_FILE}" ]]; then
    count="$(<"${DOCKER_PUSH_COUNT_FILE}")"
  fi
  count=$((count + 1))
  printf '%d\n' "${count}" >"${DOCKER_PUSH_COUNT_FILE}"
  case "${DOCKER_MODE:-success}" in
    success | invalid-digest)
      exit 0
      ;;
    secondary-once)
      if (( count == 1 )); then
        printf 'denied: permission_denied: HTTP status code 403: secondary rate limit\n' >&2
        exit 1
      fi
      exit 0
      ;;
    secondary-always)
      printf 'denied: permission_denied: HTTP status code 403: secondary rate limit\n' >&2
      exit 1
      ;;
    permanent)
      printf 'denied: permission_denied: authentication required\n' >&2
      exit 1
      ;;
    *)
      exit 92
      ;;
  esac
fi

[[ "$1 $2 $3" == "buildx imagetools inspect" ]] || exit 91
if [[ "${DOCKER_MODE:-success}" == "invalid-digest" ]]; then
  printf '{"digest":"latest"}\n'
else
  printf '{"digest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}\n'
fi
EOF

cat >"${tmpdir}/bin/sleep" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$1" >>"${SLEEP_CALLS_FILE}"
EOF

chmod +x "${tmpdir}/bin/docker" "${tmpdir}/bin/sleep"
bash -n "${PUBLISHER}"

run_publisher() {
  local output_file="$1"
  shift
  : >"${tmpdir}/docker-calls"
  : >"${tmpdir}/push-count"
  : >"${tmpdir}/sleep-calls"
  (
    export PATH="${tmpdir}/bin:${PATH}"
    export COMMIT_SHA="${VALID_SHA}"
    export IMAGE_NAME=ghcr.io/stuhelper/backend
    export GITHUB_OUTPUT="${output_file}"
    export DOCKER_CALLS_FILE="${tmpdir}/docker-calls"
    export DOCKER_PUSH_COUNT_FILE="${tmpdir}/push-count"
    export SLEEP_CALLS_FILE="${tmpdir}/sleep-calls"
    export PUBLISH_RETRY_BASE_DELAY_SECONDS=1
    if (( $# > 0 )); then
      export "${@?}"
    fi
    "${PUBLISHER}"
  )
}

success_output="${tmpdir}/success-output"
run_publisher "${success_output}"
assert_contains "${success_output}" "^digest=${DIGEST}$"
[[ "$(<"${tmpdir}/push-count")" == "1" ]] ||
  fail "a successful publish should push exactly once"
[[ ! -s "${tmpdir}/sleep-calls" ]] ||
  fail "a successful publish must not sleep"

transient_output="${tmpdir}/transient-output"
run_publisher "${transient_output}" DOCKER_MODE=secondary-once
assert_contains "${transient_output}" "^digest=${DIGEST}$"
[[ "$(<"${tmpdir}/push-count")" == "2" ]] ||
  fail "a rate-limited publish should retry"
[[ "$(<"${tmpdir}/sleep-calls")" == "1" ]] ||
  fail "the first rate-limit retry did not use the configured base delay"

if run_publisher "${tmpdir}/permanent-output" DOCKER_MODE=permanent >/dev/null 2>&1; then
  fail "a permanent registry error was retried as success"
fi
[[ "$(<"${tmpdir}/push-count")" == "1" ]] ||
  fail "a permanent registry error should fail immediately"
[[ ! -s "${tmpdir}/sleep-calls" ]] ||
  fail "a permanent registry error must not sleep"

if run_publisher \
  "${tmpdir}/exhausted-output" \
  DOCKER_MODE=secondary-always \
  PUBLISH_MAX_ATTEMPTS=4 >/dev/null 2>&1; then
  fail "an indefinitely rate-limited publish exceeded its retry budget"
fi
[[ "$(<"${tmpdir}/push-count")" == "4" ]] ||
  fail "the publisher did not enforce the configured attempt budget"
[[ "$(paste -sd, "${tmpdir}/sleep-calls")" == "1,2,4" ]] ||
  fail "the publisher did not use bounded exponential backoff"

if run_publisher "${tmpdir}/invalid-digest-output" DOCKER_MODE=invalid-digest >/dev/null 2>&1; then
  fail "the publisher accepted an invalid manifest digest"
fi

if COMMIT_SHA=latest \
  IMAGE_NAME=ghcr.io/stuhelper/backend \
  GITHUB_OUTPUT="${tmpdir}/invalid-input-output" \
  "${PUBLISHER}" >/dev/null 2>&1; then
  fail "the publisher accepted a mutable release identifier"
fi

if COMMIT_SHA="${VALID_SHA}" \
  IMAGE_NAME=ghcr.io/stuhelper/backend \
  GITHUB_OUTPUT="${tmpdir}/invalid-delay-output" \
  PUBLISH_RETRY_BASE_DELAY_SECONDS=121 \
  "${PUBLISHER}" >/dev/null 2>&1; then
  fail "the publisher accepted a retry delay that can exceed the job budget"
fi

printf '[publish-scanned-image-contract] all assertions passed\n'
