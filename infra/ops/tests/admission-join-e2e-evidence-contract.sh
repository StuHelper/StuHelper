#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
EVIDENCE_SCRIPT="${REPO_ROOT}/infra/ops/admission-join-e2e-evidence.sh"
PROD_GO_LIVE="${REPO_ROOT}/docs/guides/production-go-live.md"
RELEASE_RUNBOOK="${REPO_ROOT}/docs/guides/release-runbook.md"

fail() {
  echo "[admission-join-e2e-evidence-contract][error] $*" >&2
  exit 1
}

assert_contains() {
  local file="$1"
  local pattern="$2"
  if ! grep -Eq -- "${pattern}" "${file}"; then
    fail "expected ${file} to contain pattern: ${pattern}"
  fi
}

assert_not_contains() {
  local file="$1"
  local pattern="$2"
  if grep -Eq -- "${pattern}" "${file}"; then
    fail "expected ${file} not to contain pattern: ${pattern}"
  fi
}

[[ -f "${EVIDENCE_SCRIPT}" ]] || fail "missing E2E evidence script: ${EVIDENCE_SCRIPT}"
[[ -f "${PROD_GO_LIVE}" ]] || fail "missing production go-live guide: ${PROD_GO_LIVE}"
[[ -f "${RELEASE_RUNBOOK}" ]] || fail "missing release runbook: ${RELEASE_RUNBOOK}"

bash -n "${EVIDENCE_SCRIPT}"

assert_contains "${EVIDENCE_SCRIPT}" 'ADMISSION_E2E_QQ_ID'
assert_contains "${EVIDENCE_SCRIPT}" 'ADMISSION_E2E_GUILD_ID'
assert_contains "${EVIDENCE_SCRIPT}" 'ADMISSION_E2E_EXPECTED_STAGE'
assert_contains "${EVIDENCE_SCRIPT}" 'ADMISSION_E2E_MAX_SESSION_AGE_MINUTES'
assert_contains "${EVIDENCE_SCRIPT}" 'join-created'
assert_contains "${EVIDENCE_SCRIPT}" 'flow-completed'
assert_contains "${EVIDENCE_SCRIPT}" 'bot-released'
assert_contains "${EVIDENCE_SCRIPT}" 'group_admission_sessions'
assert_contains "${EVIDENCE_SCRIPT}" 'user_qq_bindings'
assert_contains "${EVIDENCE_SCRIPT}" 'user_verification_credentials'
assert_contains "${EVIDENCE_SCRIPT}" 'freshman_verification_applications'
assert_contains "${EVIDENCE_SCRIPT}" 'member_blacklist_entries'
assert_contains "${EVIDENCE_SCRIPT}" 'released_at IS NULL'
assert_contains "${EVIDENCE_SCRIPT}" 'authURLCanonicalPrefix'
assert_contains "${EVIDENCE_SCRIPT}" 'join\.stuhelper\.com'
assert_contains "${EVIDENCE_SCRIPT}" '/verify/'
assert_contains "${EVIDENCE_SCRIPT}" '/verify/redacted'
assert_contains "${EVIDENCE_SCRIPT}" 'tokenHashPresent'
assert_contains "${EVIDENCE_SCRIPT}" 'sessionAgeSeconds'
assert_contains "${EVIDENCE_SCRIPT}" 'updatedAgeSeconds'
assert_contains "${EVIDENCE_SCRIPT}" 'latest session is fresh enough for this E2E run'
assert_contains "${EVIDENCE_SCRIPT}" 'bot release evidence is fresh enough for this E2E run'
assert_contains "${EVIDENCE_SCRIPT}" 'token was consumed by authenticated user'
assert_contains "${EVIDENCE_SCRIPT}" 'student verification credential or submitted freshman material recorded'
assert_contains "${EVIDENCE_SCRIPT}" 'has_submitted_freshman_material'
assert_contains "${EVIDENCE_SCRIPT}" 'release requires active student verification credential'
assert_contains "${EVIDENCE_SCRIPT}" 'backend recorded successful bot release'
assert_contains "${EVIDENCE_SCRIPT}" 'cancelledAtPresent'
assert_contains "${EVIDENCE_SCRIPT}" 'botReleaseRecorded'
assert_contains "${EVIDENCE_SCRIPT}" 'compose --profile prod run --rm --no-deps -T'
assert_contains "${EVIDENCE_SCRIPT}" 'postgres'
assert_contains "${EVIDENCE_SCRIPT}" 'psql'
assert_contains "${EVIDENCE_SCRIPT}" 'REPLACE_WITH_STUHELPER_APP_DB_PASSWORD'
assert_contains "${EVIDENCE_SCRIPT}" 'ADMISSION_PUBLIC_BASE_URL must be exactly https://join.stuhelper.com'

assert_not_contains "${EVIDENCE_SCRIPT}" 'root@'
assert_not_contains "${EVIDENCE_SCRIPT}" 'sshpass'
assert_not_contains "${EVIDENCE_SCRIPT}" '65022|2222'
assert_not_contains "${EVIDENCE_SCRIPT}" 'STUHELPER_PLATFORM_SERVICE_TOKEN='
assert_not_contains "${EVIDENCE_SCRIPT}" '"tokenHash"'
assert_not_contains "${EVIDENCE_SCRIPT}" '^session_user AS'

tmpdir="$(mktemp -d)"
cleanup() {
  rm -rf "${tmpdir}"
}
trap cleanup EXIT

cat >"${tmpdir}/.env" <<'ENV'
APP_ENV=production
ADMISSION_PUBLIC_BASE_URL=https://join.stuhelper.com
DATABASE_URL=postgres://stuhelper:REPLACE_WITH_STUHELPER_APP_DB_PASSWORD@postgres/stuhelper
STUHELPER_APP_DB_PASSWORD=fake-password
ENV
touch "${tmpdir}/.env.generated" "${tmpdir}/.env.generated.secrets"

mkdir -p "${tmpdir}/bin"
cat >"${tmpdir}/bin/docker" <<'SH'
#!/usr/bin/env bash
set -euo pipefail

while (($# > 0)); do
  case "$1" in
    -v)
      shift 2
      ;;
    --profile|run|--rm|--no-deps|-T|postgres|psql|-X|-At)
      shift
      ;;
    -v)
      shift 2
      ;;
    *)
      shift
      ;;
  esac
done

case "${FAKE_E2E_MODE:-completed}" in
  completed)
    cat <<'JSON'
{"input":{"platform":"qq","guildID":"178037297","qqID":"123456789","botSelfID":"2118785781","lookbackHours":24},"sessionCount":1,"latestSession":{"id":"sess-1","platform":"qq","botSelfID":"2118785781","guildID":"178037297","channelIDPresent":true,"qqID":"123456789","userIDPresent":true,"authURLHost":"join.stuhelper.com","authURLPath":"/verify/redacted","authURLHasQQQuery":true,"authURLCanonicalPrefix":true,"tokenHashPresent":true,"tokenConsumed":true,"status":"verified","verified":true,"cancelledAtPresent":true,"botReleaseRecorded":true,"lastBotErrorPresent":false,"createdAt":"2026-05-30T12:00:00Z","updatedAt":"2026-05-30T12:10:00Z","sessionAgeSeconds":600,"updatedAgeSeconds":60,"tokenExpiresAt":"2026-05-30T13:00:00Z","tokenExpired":false,"linkWaitDeadlineAt":"2026-05-30T13:00:00Z","submissionWaitDeadlineAt":"2026-05-31T12:00:00Z","initialMuteUntil":"2026-06-29T12:00:00Z"},"user":{"idPresent":true,"casdoorSubjectPresent":true,"usernamePresent":true,"emailPresent":true},"qqBinding":{"bound":true,"boundAt":"2026-05-30T12:05:00Z"},"studentVerification":{"activeCredentialCount":1,"kinds":["school_email_otp"],"schoolIDs":[4111010006]},"freshmanApplications":{"count":0,"statuses":[]},"failure":{"failureCount":0,"lastFailureAt":null},"activeBlacklistCount":0}
JSON
    ;;
  joined)
    cat <<'JSON'
{"input":{"platform":"qq","guildID":"178037297","qqID":"123456789","botSelfID":"2118785781","lookbackHours":24},"sessionCount":1,"latestSession":{"id":"sess-1","platform":"qq","botSelfID":"2118785781","guildID":"178037297","channelIDPresent":true,"qqID":"123456789","userIDPresent":false,"authURLHost":"join.stuhelper.com","authURLPath":"/verify/redacted","authURLHasQQQuery":true,"authURLCanonicalPrefix":true,"tokenHashPresent":true,"tokenConsumed":false,"status":"joined_muted","verified":false,"cancelledAtPresent":false,"botReleaseRecorded":false,"lastBotErrorPresent":false,"createdAt":"2026-05-30T12:00:00Z","updatedAt":"2026-05-30T12:00:00Z","sessionAgeSeconds":60,"updatedAgeSeconds":60,"tokenExpiresAt":"2026-05-30T13:00:00Z","tokenExpired":false,"linkWaitDeadlineAt":"2026-05-30T13:00:00Z","submissionWaitDeadlineAt":"2026-05-31T12:00:00Z","initialMuteUntil":"2026-06-29T12:00:00Z"},"user":null,"qqBinding":{"bound":false,"boundAt":null},"studentVerification":{"activeCredentialCount":0,"kinds":[],"schoolIDs":[]},"freshmanApplications":{"count":0,"statuses":[]},"failure":{"failureCount":0,"lastFailureAt":null},"activeBlacklistCount":0}
JSON
    ;;
  stale-released)
    cat <<'JSON'
{"input":{"platform":"qq","guildID":"178037297","qqID":"123456789","botSelfID":"2118785781","lookbackHours":24},"sessionCount":1,"latestSession":{"id":"sess-stale","platform":"qq","botSelfID":"2118785781","guildID":"178037297","channelIDPresent":true,"qqID":"123456789","userIDPresent":true,"authURLHost":"join.stuhelper.com","authURLPath":"/verify/redacted","authURLHasQQQuery":true,"authURLCanonicalPrefix":true,"tokenHashPresent":true,"tokenConsumed":true,"status":"verified","verified":true,"cancelledAtPresent":true,"botReleaseRecorded":true,"lastBotErrorPresent":false,"createdAt":"2026-05-30T12:00:00Z","updatedAt":"2026-05-30T12:10:00Z","sessionAgeSeconds":90000,"updatedAgeSeconds":89000,"tokenExpiresAt":"2026-05-30T13:00:00Z","tokenExpired":true,"linkWaitDeadlineAt":"2026-05-30T13:00:00Z","submissionWaitDeadlineAt":"2026-05-31T12:00:00Z","initialMuteUntil":"2026-06-29T12:00:00Z"},"user":{"idPresent":true,"casdoorSubjectPresent":true,"usernamePresent":true,"emailPresent":true},"qqBinding":{"bound":true,"boundAt":"2026-05-30T12:05:00Z"},"studentVerification":{"activeCredentialCount":1,"kinds":["school_email_otp"],"schoolIDs":[4111010006]},"freshmanApplications":{"count":0,"statuses":[]},"failure":{"failureCount":0,"lastFailureAt":null},"activeBlacklistCount":0}
JSON
    ;;
  released-no-credential)
    cat <<'JSON'
{"input":{"platform":"qq","guildID":"178037297","qqID":"123456789","botSelfID":"2118785781","lookbackHours":24},"sessionCount":1,"latestSession":{"id":"sess-no-credential","platform":"qq","botSelfID":"2118785781","guildID":"178037297","channelIDPresent":true,"qqID":"123456789","userIDPresent":true,"authURLHost":"join.stuhelper.com","authURLPath":"/verify/redacted","authURLHasQQQuery":true,"authURLCanonicalPrefix":true,"tokenHashPresent":true,"tokenConsumed":true,"status":"verified","verified":true,"cancelledAtPresent":true,"botReleaseRecorded":true,"lastBotErrorPresent":false,"createdAt":"2026-05-30T12:00:00Z","updatedAt":"2026-05-30T12:10:00Z","sessionAgeSeconds":600,"updatedAgeSeconds":60,"tokenExpiresAt":"2026-05-30T13:00:00Z","tokenExpired":false,"linkWaitDeadlineAt":"2026-05-30T13:00:00Z","submissionWaitDeadlineAt":"2026-05-31T12:00:00Z","initialMuteUntil":"2026-06-29T12:00:00Z"},"user":{"idPresent":true,"casdoorSubjectPresent":true,"usernamePresent":true,"emailPresent":true},"qqBinding":{"bound":true,"boundAt":"2026-05-30T12:05:00Z"},"studentVerification":{"activeCredentialCount":0,"kinds":[],"schoolIDs":[]},"freshmanApplications":{"count":1,"statuses":["pending"]},"failure":{"failureCount":0,"lastFailureAt":null},"activeBlacklistCount":0}
JSON
    ;;
  linked-freshman-created)
    cat <<'JSON'
{"input":{"platform":"qq","guildID":"178037297","qqID":"123456789","botSelfID":"2118785781","lookbackHours":24},"sessionCount":1,"latestSession":{"id":"sess-linked-freshman-created","platform":"qq","botSelfID":"2118785781","guildID":"178037297","channelIDPresent":true,"qqID":"123456789","userIDPresent":true,"authURLHost":"join.stuhelper.com","authURLPath":"/verify/redacted","authURLHasQQQuery":true,"authURLCanonicalPrefix":true,"tokenHashPresent":true,"tokenConsumed":true,"status":"linked","verified":false,"cancelledAtPresent":false,"botReleaseRecorded":false,"lastBotErrorPresent":false,"createdAt":"2026-05-30T12:00:00Z","updatedAt":"2026-05-30T12:05:00Z","sessionAgeSeconds":600,"updatedAgeSeconds":300,"tokenExpiresAt":"2026-05-30T13:00:00Z","tokenExpired":false,"linkWaitDeadlineAt":"2026-05-30T13:00:00Z","submissionWaitDeadlineAt":"2026-05-31T12:00:00Z","initialMuteUntil":"2026-06-29T12:00:00Z"},"user":{"idPresent":true,"casdoorSubjectPresent":true,"usernamePresent":true,"emailPresent":true},"qqBinding":{"bound":true,"boundAt":"2026-05-30T12:05:00Z"},"studentVerification":{"activeCredentialCount":0,"kinds":[],"schoolIDs":[]},"freshmanApplications":{"count":1,"statuses":["pending"]},"failure":{"failureCount":0,"lastFailureAt":null},"activeBlacklistCount":0}
JSON
    ;;
  none)
    cat <<'JSON'
{"input":{"platform":"qq","guildID":"178037297","qqID":"123456789","botSelfID":"2118785781","lookbackHours":24},"sessionCount":0,"latestSession":null,"user":null,"qqBinding":{"bound":false,"boundAt":null},"studentVerification":{"activeCredentialCount":0,"kinds":[],"schoolIDs":[]},"freshmanApplications":{"count":0,"statuses":[]},"failure":{"failureCount":0,"lastFailureAt":null},"activeBlacklistCount":0}
JSON
    ;;
esac
SH
chmod +x "${tmpdir}/bin/docker"

run_script() {
  local mode="$1"
  local stage="$2"
  local output="$3"
  PATH="${tmpdir}/bin:${PATH}" \
  ENV_FILE="${tmpdir}/.env" \
  GENERATED_ENV_FILE="${tmpdir}/.env.generated" \
  GENERATED_SECRET_ENV_FILE="${tmpdir}/.env.generated.secrets" \
  FAKE_E2E_MODE="${mode}" \
  ADMISSION_E2E_QQ_ID="123456789" \
  ADMISSION_E2E_GUILD_ID="178037297" \
  ADMISSION_E2E_BOT_SELF_ID="2118785781" \
  ADMISSION_E2E_EXPECTED_STAGE="${stage}" \
  ADMISSION_E2E_MAX_SESSION_AGE_MINUTES="180" \
  ADMISSION_E2E_EVIDENCE_FILE="${output}" \
  "${EVIDENCE_SCRIPT}" >"${output}.stdout" 2>"${output}.stderr"
}

completed_file="${tmpdir}/completed.json"
run_script completed flow-completed "${completed_file}"
jq -e '
  .passed == true
  and .expectedStage == "flow-completed"
  and .summary.failed == 0
  and .session.authURLHost == "join.stuhelper.com"
  and .session.authURLPath == "/verify/redacted"
  and (.session | has("tokenHash") | not)
  and ([.checks[] | select(.name == "token was consumed by authenticated user" and .passed == true)] | length == 1)
' "${completed_file}" >/dev/null

released_file="${tmpdir}/released.json"
run_script completed bot-released "${released_file}"
jq -e '
  .passed == true
  and .expectedStage == "bot-released"
  and .maxSessionAgeMinutes == 180
  and .session.botReleaseRecorded == true
  and ([.checks[] | select(.name == "release requires active student verification credential" and .passed == true)] | length == 1)
  and ([.checks[] | select(.name == "backend recorded successful bot release" and .passed == true)] | length == 1)
  and ([.checks[] | select(.name == "bot release evidence is fresh enough for this E2E run" and .passed == true)] | length == 1)
' "${released_file}" >/dev/null

joined_file="${tmpdir}/joined.json"
run_script joined join-created "${joined_file}"
jq -e '
  .passed == true
  and .expectedStage == "join-created"
  and .session.status == "joined_muted"
  and .qqBinding.bound == false
' "${joined_file}" >/dev/null

failed_file="${tmpdir}/failed.json"
if run_script joined flow-completed "${failed_file}"; then
  fail "flow-completed evidence unexpectedly passed for a join-created-only record"
fi
jq -e '
  .passed == false
  and ([.checks[] | select(.name == "token was consumed by authenticated user" and .passed == false)] | length == 1)
' "${failed_file}" >/dev/null

created_freshman_file="${tmpdir}/created-freshman.json"
if run_script linked-freshman-created flow-completed "${created_freshman_file}"; then
  fail "flow-completed evidence unexpectedly passed for a freshman application without submitted material"
fi
jq -e '
  .passed == false
  and .session.status == "linked"
  and .qqBinding.bound == true
  and .freshmanApplications.count == 1
  and ([.checks[] | select(.name == "student verification credential or submitted freshman material recorded" and .passed == false)] | length == 1)
  and ([.checks[] | select(.name == "session reached release-eligible or review-pending state" and .passed == false)] | length == 1)
' "${created_freshman_file}" >/dev/null

failed_release_file="${tmpdir}/failed-release.json"
if run_script joined bot-released "${failed_release_file}"; then
  fail "bot-released evidence unexpectedly passed for a join-created-only record"
fi
jq -e '
  .passed == false
  and ([.checks[] | select(.name == "backend recorded successful bot release" and .passed == false)] | length == 1)
' "${failed_release_file}" >/dev/null

stale_release_file="${tmpdir}/stale-release.json"
if run_script stale-released bot-released "${stale_release_file}"; then
  fail "bot-released evidence unexpectedly passed for a stale record"
fi
jq -e '
  .passed == false
  and ([.checks[] | select(.name == "latest session is fresh enough for this E2E run" and .passed == false)] | length == 1)
  and ([.checks[] | select(.name == "bot release evidence is fresh enough for this E2E run" and .passed == false)] | length == 1)
' "${stale_release_file}" >/dev/null

no_credential_release_file="${tmpdir}/no-credential-release.json"
if run_script released-no-credential bot-released "${no_credential_release_file}"; then
  fail "bot-released evidence unexpectedly passed without an active student verification credential"
fi
jq -e '
  .passed == false
  and .session.botReleaseRecorded == true
  and .studentVerification.activeCredentialCount == 0
  and ([.checks[] | select(.name == "release requires active student verification credential" and .passed == false)] | length == 1)
' "${no_credential_release_file}" >/dev/null

no_credential_flow_file="${tmpdir}/no-credential-flow.json"
if run_script released-no-credential flow-completed "${no_credential_flow_file}"; then
  fail "flow-completed evidence unexpectedly passed for a verified session without an active credential"
fi
jq -e '
  .passed == false
  and .session.status == "verified"
  and .studentVerification.activeCredentialCount == 0
  and ([.checks[] | select(.name == "student verification credential or submitted freshman material recorded" and .passed == false)] | length == 1)
' "${no_credential_flow_file}" >/dev/null

missing_file="${tmpdir}/missing.json"
if run_script none join-created "${missing_file}"; then
  fail "join-created evidence unexpectedly passed without a session"
fi
jq -e '
  .passed == false
  and ([.checks[] | select(.name == "real join created an admission session" and .passed == false)] | length == 1)
  and ([.checks[] | select(.name == "session is for expected qq" and .passed == false and .detail == "qqID=None, expectedQQID=123456789")] | length == 1)
' "${missing_file}" >/dev/null

assert_contains "${PROD_GO_LIVE}" 'admission-join-e2e-evidence\.sh'
assert_contains "${RELEASE_RUNBOOK}" 'admission-join-e2e-evidence\.sh'
assert_contains "${RELEASE_RUNBOOK}" 'ADMISSION_E2E_QQ_ID'
assert_contains "${RELEASE_RUNBOOK}" 'bot-released'
assert_contains "${RELEASE_RUNBOOK}" 'successful bot release'

echo "[admission-join-e2e-evidence-contract] all assertions passed"
