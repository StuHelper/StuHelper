#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
WAIT_SCRIPT="${REPO_ROOT}/infra/ops/admission-join-e2e-wait.sh"
EVIDENCE_SCRIPT="${REPO_ROOT}/infra/ops/admission-join-e2e-evidence.sh"
PROD_GO_LIVE="${REPO_ROOT}/docs/guides/production-go-live.md"
RELEASE_RUNBOOK="${REPO_ROOT}/docs/guides/release-runbook.md"

fail() {
  echo "[admission-join-e2e-wait-contract][error] $*" >&2
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

[[ -f "${WAIT_SCRIPT}" ]] || fail "missing wait script: ${WAIT_SCRIPT}"
[[ -f "${EVIDENCE_SCRIPT}" ]] || fail "missing evidence script: ${EVIDENCE_SCRIPT}"

bash -n "${WAIT_SCRIPT}"
assert_contains "${WAIT_SCRIPT}" 'ADMISSION_E2E_QQ_ID'
assert_contains "${WAIT_SCRIPT}" 'ADMISSION_E2E_WAIT_TIMEOUT_SECONDS'
assert_contains "${WAIT_SCRIPT}" 'ADMISSION_E2E_WAIT_INTERVAL_SECONDS'
assert_contains "${WAIT_SCRIPT}" 'ADMISSION_E2E_FINAL_EVIDENCE_FILE'
assert_contains "${WAIT_SCRIPT}" 'ADMISSION_E2E_WAIT_EVIDENCE_FILE'
assert_contains "${WAIT_SCRIPT}" 'admission-join-e2e-evidence\.sh'
assert_contains "${WAIT_SCRIPT}" 'join-created'
assert_contains "${WAIT_SCRIPT}" 'flow-completed'
assert_contains "${WAIT_SCRIPT}" 'bot-released'
assert_contains "${EVIDENCE_SCRIPT}" 'ADMISSION_E2E_MAX_SESSION_AGE_MINUTES'
assert_not_contains "${WAIT_SCRIPT}" 'root@'
assert_not_contains "${WAIT_SCRIPT}" 'sshpass'
assert_not_contains "${WAIT_SCRIPT}" '65022|2222'
assert_not_contains "${WAIT_SCRIPT}" 'STUHELPER_PLATFORM_SERVICE_TOKEN='

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
  shift
done

case "${FAKE_E2E_MODE:-joined}" in
  joined)
    cat <<'JSON'
{"input":{"platform":"qq","guildID":"178037297","qqID":"123456789","botSelfID":"2118785781","lookbackHours":24},"sessionCount":1,"latestSession":{"id":"sess-1","platform":"qq","botSelfID":"2118785781","guildID":"178037297","channelIDPresent":true,"qqID":"123456789","userIDPresent":false,"authURLRaw":"https://join.stuhelper.com/verify/fake-public-preview-token?qq=123456789","authURLHost":"join.stuhelper.com","authURLPath":"/verify/redacted","authURLHasQQQuery":true,"authURLCanonicalPrefix":true,"tokenHashPresent":true,"tokenConsumed":false,"status":"joined_muted","verified":false,"cancelledAtPresent":false,"botReleaseRecorded":false,"lastBotErrorPresent":false,"createdAt":"2026-05-30T12:00:00Z","updatedAt":"2026-05-30T12:00:00Z","sessionAgeSeconds":60,"updatedAgeSeconds":60,"tokenExpiresAt":"2026-05-30T13:00:00Z","tokenExpired":false,"linkWaitDeadlineAt":"2026-05-30T13:00:00Z","submissionWaitDeadlineAt":"2026-05-31T12:00:00Z","initialMuteUntil":"2026-06-29T12:00:00Z"},"user":null,"qqBinding":{"bound":false,"boundAt":null},"studentVerification":{"activeCredentialCount":0,"kinds":[],"schoolIDs":[]},"freshmanApplications":{"count":0,"statuses":[]},"failure":{"failureCount":0,"lastFailureAt":null},"activeBlacklistCount":0}
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

cat >"${tmpdir}/bin/curl" <<'SH'
#!/usr/bin/env bash
set -euo pipefail

output_file=""
write_out=""
url=""

while (($# > 0)); do
  case "$1" in
    -o)
      output_file="$2"
      shift 2
      ;;
    -w)
      write_out="$2"
      shift 2
      ;;
    --noproxy|--resolve)
      shift 2
      ;;
    --insecure|-s|-S|-sS)
      shift
      ;;
    http://*|https://*)
      url="$1"
      shift
      ;;
    *)
      shift
      ;;
  esac
done

[[ "${url}" == "https://join.stuhelper.com/api/v1/admission/sessions/fake-public-preview-token?qq=123456789" ]] || {
  echo "unexpected curl URL: ${url}" >&2
  exit 7
}

body='{"success":true,"data":{"id":"sess-1"}}'
if [[ -n "${output_file}" ]]; then
  printf '%s' "${body}" >"${output_file}"
else
  printf '%s' "${body}"
fi

if [[ -n "${write_out}" ]]; then
  printf '200\n203.0.113.10\n%s\n' "${#body}"
fi
SH
chmod +x "${tmpdir}/bin/curl"

final_evidence="${tmpdir}/final.json"
wait_evidence="${tmpdir}/wait.json"
PATH="${tmpdir}/bin:${PATH}" \
ENV_FILE="${tmpdir}/.env" \
GENERATED_ENV_FILE="${tmpdir}/.env.generated" \
GENERATED_SECRET_ENV_FILE="${tmpdir}/.env.generated.secrets" \
FAKE_E2E_MODE=joined \
ADMISSION_E2E_QQ_ID=123456789 \
ADMISSION_E2E_GUILD_ID=178037297 \
ADMISSION_E2E_EXPECTED_STAGE=join-created \
ADMISSION_E2E_MAX_SESSION_AGE_MINUTES=180 \
ADMISSION_E2E_WAIT_TIMEOUT_SECONDS=2 \
ADMISSION_E2E_WAIT_INTERVAL_SECONDS=1 \
ADMISSION_E2E_FINAL_EVIDENCE_FILE="${final_evidence}" \
ADMISSION_E2E_WAIT_EVIDENCE_FILE="${wait_evidence}" \
"${WAIT_SCRIPT}" >"${tmpdir}/wait-ok.stdout" 2>"${tmpdir}/wait-ok.stderr"

[[ -f "${final_evidence}" ]] || fail "final evidence was not written"
[[ -f "${wait_evidence}" ]] || fail "wait evidence was not written"
jq -e '
  .passed == true
  and .expectedStage == "join-created"
  and .attemptsCount == 1
  and .finalEvidenceFile == "'"${final_evidence}"'"
' "${wait_evidence}" >/dev/null
jq -e '
  .passed == true
  and .expectedStage == "join-created"
  and .maxSessionAgeMinutes == 180
  and .publicPreview.httpStatus == 200
  and .publicPreview.sessionIDMatches == true
  and ([.checks[] | select(.name == "latest session is fresh enough for this E2E run" and .passed == true)] | length == 1)
  and ([.checks[] | select(.name == "unconsumed session public preview API is reachable" and .passed == true)] | length == 1)
' "${final_evidence}" >/dev/null
jq -e '.passed == true and .attemptsCount == 1 and .finalEvidenceFile == "'"${final_evidence}"'"' "${wait_evidence}" >/dev/null

PATH="${tmpdir}/bin:${PATH}" \
ENV_FILE="${tmpdir}/.env" \
GENERATED_ENV_FILE="${tmpdir}/.env.generated" \
GENERATED_SECRET_ENV_FILE="${tmpdir}/.env.generated.secrets" \
FAKE_E2E_MODE=joined \
ADMISSION_E2E_QQ_ID=123456789 \
ADMISSION_E2E_GUILD_ID=178037297 \
ADMISSION_E2E_EXPECTED_STAGE=join-created \
ADMISSION_E2E_MAX_SESSION_AGE_MINUTES=180 \
ADMISSION_E2E_WAIT_TIMEOUT_SECONDS=2 \
ADMISSION_E2E_WAIT_INTERVAL_SECONDS=1 \
ADMISSION_E2E_FINAL_EVIDENCE_FILE=- \
ADMISSION_E2E_WAIT_EVIDENCE_FILE=- \
"${WAIT_SCRIPT}" >"${tmpdir}/wait-stdout-json.out" 2>"${tmpdir}/wait-stdout-json.err"
jq -e '
  .passed == true
  and .expectedStage == "join-created"
  and .attemptsCount == 1
  and .finalEvidenceFile == "-"
' "${tmpdir}/wait-stdout-json.out" >/dev/null

failed_wait="${tmpdir}/wait-failed.json"
if PATH="${tmpdir}/bin:${PATH}" \
  ENV_FILE="${tmpdir}/.env" \
  GENERATED_ENV_FILE="${tmpdir}/.env.generated" \
  GENERATED_SECRET_ENV_FILE="${tmpdir}/.env.generated.secrets" \
  FAKE_E2E_MODE=none \
  ADMISSION_E2E_QQ_ID=123456789 \
  ADMISSION_E2E_GUILD_ID=178037297 \
ADMISSION_E2E_EXPECTED_STAGE=join-created \
ADMISSION_E2E_MAX_SESSION_AGE_MINUTES=180 \
ADMISSION_E2E_WAIT_TIMEOUT_SECONDS=1 \
  ADMISSION_E2E_WAIT_INTERVAL_SECONDS=1 \
  ADMISSION_E2E_WAIT_EVIDENCE_FILE="${failed_wait}" \
  "${WAIT_SCRIPT}" >"${tmpdir}/wait-failed.stdout" 2>"${tmpdir}/wait-failed.stderr"; then
  fail "wait script unexpectedly passed without a session"
fi
jq -e '.passed == false and .expectedStage == "join-created" and .attemptsCount >= 1' "${failed_wait}" >/dev/null

assert_contains "${PROD_GO_LIVE}" 'admission-join-e2e-wait\.sh'
assert_contains "${RELEASE_RUNBOOK}" 'admission-join-e2e-wait\.sh'

echo "[admission-join-e2e-wait-contract] all assertions passed"
