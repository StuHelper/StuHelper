#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
VERIFIER="${REPO_ROOT}/infra/ops/verify-github-release.sh"
VALID_SHA="0123456789abcdef0123456789abcdef01234567"
OTHER_SHA="89abcdef0123456789abcdef0123456789abcdef"

fail() {
  printf '[github-release-verification-contract][error] %s\n' "$*" >&2
  exit 1
}

tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT
mkdir -p "${tmpdir}/bin"

cat >"${tmpdir}/bin/gh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

if [[ "$*" == *'/git/ref/heads/'* ]]; then
  printf '%s\n' "${GH_BRANCH_SHA}"
  exit 0
fi

if [[ "$*" == *'/check-runs?per_page=100'* ]]; then
  case "${GH_CHECK_MODE:-success}" in
    success)
      cat <<JSON
{"check_runs":[
  {"id":1,"name":"Required","status":"completed","conclusion":"success","app":{"slug":"github-actions"}},
  {"id":2,"name":"go","status":"completed","conclusion":"success","app":{"slug":"github-actions"}},
  {"id":3,"name":"javascript-typescript","status":"completed","conclusion":"success","app":{"slug":"github-actions"}}
]}
JSON
      ;;
    failed)
      cat <<JSON
{"check_runs":[
  {"id":1,"name":"Required","status":"completed","conclusion":"failure","app":{"slug":"github-actions"}},
  {"id":2,"name":"go","status":"completed","conclusion":"success","app":{"slug":"github-actions"}},
  {"id":3,"name":"javascript-typescript","status":"completed","conclusion":"success","app":{"slug":"github-actions"}}
]}
JSON
      ;;
    missing)
      printf '%s\n' '{"check_runs":[]}'
      ;;
    pending)
      cat <<JSON
{"check_runs":[
  {"id":1,"name":"Required","status":"in_progress","conclusion":null,"app":{"slug":"github-actions"}},
  {"id":2,"name":"go","status":"completed","conclusion":"success","app":{"slug":"github-actions"}},
  {"id":3,"name":"javascript-typescript","status":"completed","conclusion":"success","app":{"slug":"github-actions"}}
]}
JSON
      ;;
    *) exit 91 ;;
  esac
  exit 0
fi

if [[ "$*" == *'/deployments?sha='* ]]; then
  if [[ "${GH_STAGING_MODE:-success}" == "missing" ]]; then
    printf '%s\n' '[]'
  else
    printf '%s\n' '[{"id":101}]'
  fi
  exit 0
fi

if [[ "$*" == *'/deployments/101/statuses?per_page=100'* ]]; then
  case "${GH_STAGING_MODE:-success}" in
    success) printf '%s\n' '[{"id":201,"state":"success"}]' ;;
    failed) printf '%s\n' '[{"id":201,"state":"failure"}]' ;;
    pending) printf '%s\n' '[{"id":201,"state":"in_progress"}]' ;;
    *) exit 93 ;;
  esac
  exit 0
fi

exit 92
EOF
chmod +x "${tmpdir}/bin/gh"

run_verifier() {
  PATH="${tmpdir}/bin:${PATH}" \
    TARGET_SHA="${TARGET_SHA_OVERRIDE:-${VALID_SHA}}" \
    WORKFLOW_SHA="${WORKFLOW_SHA_OVERRIDE:-${VALID_SHA}}" \
    SOURCE_REF="${SOURCE_REF_OVERRIDE:-refs/heads/main}" \
    GITHUB_REPOSITORY=StuHelper/StuHelper \
    DEPLOY_ENVIRONMENT="${DEPLOY_ENVIRONMENT_OVERRIDE:-production}" \
    DEPLOY_REASON="${DEPLOY_REASON_OVERRIDE:-Reviewed production release}" \
    RELEASE_CHECK_MAX_ATTEMPTS=1 \
    RELEASE_CHECK_POLL_SECONDS=0 \
    REQUIRE_STAGING_SUCCESS="${REQUIRE_STAGING_SUCCESS_OVERRIDE:-false}" \
    STAGING_GATE_BYPASSED="${STAGING_GATE_BYPASSED_OVERRIDE:-false}" \
    GH_BRANCH_SHA="${GH_BRANCH_SHA_OVERRIDE:-${VALID_SHA}}" \
    GH_CHECK_MODE="${GH_CHECK_MODE_OVERRIDE:-success}" \
    GH_STAGING_MODE="${GH_STAGING_MODE_OVERRIDE:-success}" \
    "${VERIFIER}"
}

expect_failure() {
  local name="$1"
  shift
  if env "$@" bash -c 'run_verifier >/dev/null 2>&1'; then
    fail "verifier accepted invalid case: ${name}"
  fi
}

export -f run_verifier
export VERIFIER tmpdir VALID_SHA OTHER_SHA

bash -n "${VERIFIER}"
run_verifier | grep -q 'branch head, required checks, and promotion policy verified' ||
  fail "valid release was not accepted"

expect_failure "workflow and target SHA differ" WORKFLOW_SHA_OVERRIDE="${OTHER_SHA}"
expect_failure "production from develop" SOURCE_REF_OVERRIDE=refs/heads/develop
expect_failure "branch advanced" GH_BRANCH_SHA_OVERRIDE="${OTHER_SHA}"
expect_failure "required check failed" GH_CHECK_MODE_OVERRIDE=failed
expect_failure "required check missing" GH_CHECK_MODE_OVERRIDE=missing
expect_failure "required check pending" GH_CHECK_MODE_OVERRIDE=pending
expect_failure "reason too short" DEPLOY_REASON_OVERRIDE=short
expect_failure "unknown environment" DEPLOY_ENVIRONMENT_OVERRIDE=preview
REQUIRE_STAGING_SUCCESS_OVERRIDE=true run_verifier >/dev/null ||
  fail "successful staging promotion was not accepted"
expect_failure "staging deployment missing" \
  REQUIRE_STAGING_SUCCESS_OVERRIDE=true GH_STAGING_MODE_OVERRIDE=missing
expect_failure "staging deployment failed" \
  REQUIRE_STAGING_SUCCESS_OVERRIDE=true GH_STAGING_MODE_OVERRIDE=failed
expect_failure "staging deployment remained pending" \
  REQUIRE_STAGING_SUCCESS_OVERRIDE=true GH_STAGING_MODE_OVERRIDE=pending
STAGING_GATE_BYPASSED_OVERRIDE=true \
  DEPLOY_REASON_OVERRIDE='Incident INC-123 requires an urgent controlled promotion' \
  run_verifier >/dev/null || fail "audited staging bypass was not accepted"
expect_failure "short staging bypass context" \
  STAGING_GATE_BYPASSED_OVERRIDE=true DEPLOY_REASON_OVERRIDE='short bypass reason'
expect_failure "staging target cannot bypass staging" \
  STAGING_GATE_BYPASSED_OVERRIDE=true \
  DEPLOY_ENVIRONMENT_OVERRIDE=staging \
  SOURCE_REF_OVERRIDE=refs/heads/develop \
  DEPLOY_REASON_OVERRIDE='Incident INC-123 requires an urgent controlled promotion'

printf '[github-release-verification-contract] all assertions passed\n'
