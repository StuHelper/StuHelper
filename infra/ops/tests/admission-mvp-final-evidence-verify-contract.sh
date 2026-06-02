#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
VERIFY_SCRIPT="${REPO_ROOT}/infra/ops/admission-mvp-final-evidence-verify.sh"
LOCAL_TEST_SCRIPT="${REPO_ROOT}/infra/ops/admission-mvp-local-test.sh"
GO_LIVE_DOC="${REPO_ROOT}/docs/guides/production-go-live.md"
RELEASE_RUNBOOK="${REPO_ROOT}/docs/guides/release-runbook.md"
MAKEFILE="${REPO_ROOT}/Makefile"
PROD_ENV_EXAMPLE="${REPO_ROOT}/.env.prod.example"

fail() {
  echo "[admission-mvp-final-evidence-verify-contract][error] $*" >&2
  exit 1
}

assert_contains() {
  local file="$1"
  local pattern="$2"
  if ! grep -Eq -- "${pattern}" "${file}"; then
    fail "expected ${file} to contain pattern: ${pattern}"
  fi
}

cleanup() {
  [[ -n "${tmpdir:-}" ]] && rm -rf "${tmpdir}"
}
trap cleanup EXIT

[[ -f "${VERIFY_SCRIPT}" ]] || fail "missing final evidence verifier: ${VERIFY_SCRIPT}"
[[ -f "${LOCAL_TEST_SCRIPT}" ]] || fail "missing local MVP test script: ${LOCAL_TEST_SCRIPT}"
[[ -f "${GO_LIVE_DOC}" ]] || fail "missing go-live doc: ${GO_LIVE_DOC}"
[[ -f "${RELEASE_RUNBOOK}" ]] || fail "missing release runbook: ${RELEASE_RUNBOOK}"
[[ -f "${MAKEFILE}" ]] || fail "missing Makefile: ${MAKEFILE}"
[[ -f "${PROD_ENV_EXAMPLE}" ]] || fail "missing prod env example: ${PROD_ENV_EXAMPLE}"

bash -n "${VERIFY_SCRIPT}"
[[ -x "${VERIFY_SCRIPT}" ]] || fail "final evidence verifier must be executable"

assert_contains "${VERIFY_SCRIPT}" 'ADMISSION_MVP_FINAL_MAIN_EVIDENCE_FILE'
assert_contains "${VERIFY_SCRIPT}" 'ADMISSION_MVP_FINAL_KOISHI_EVIDENCE_FILE'
assert_contains "${VERIFY_SCRIPT}" 'ADMISSION_MVP_FINAL_JOIN_E2E_EVIDENCE_FILE'
assert_contains "${VERIFY_SCRIPT}" 'ADMISSION_MVP_FINAL_VERIFY_FILE'
assert_contains "${VERIFY_SCRIPT}" 'ADMISSION_MVP_FINAL_MAX_EVIDENCE_AGE_MINUTES'
assert_contains "${VERIFY_SCRIPT}" 'ADMISSION_MVP_FINAL_EXPECTED_E2E_STAGE'
assert_contains "${VERIFY_SCRIPT}" 'real QQ bot-released E2E'
assert_contains "${VERIFY_SCRIPT}" 'koishi evidence does not contain real QQ E2E placeholders'
assert_contains "${VERIFY_SCRIPT}" 'public Web auth browser smoke'
assert_contains "${VERIFY_SCRIPT}" 'admission production readiness'
assert_contains "${VERIFY_SCRIPT}" 'Koishi admission production evidence'
assert_contains "${VERIFY_SCRIPT}" 'release requires active student verification credential'

tmpdir="$(mktemp -d)"
fresh_main="${tmpdir}/main.json"
fresh_koishi="${tmpdir}/koishi.json"
fresh_join="${tmpdir}/join-e2e.json"
verify_file="${tmpdir}/verify.json"

python3 - "${fresh_main}" "${fresh_koishi}" "${fresh_join}" <<'PY'
import json
import sys
from datetime import datetime, timezone
from pathlib import Path

main_path = Path(sys.argv[1])
koishi_path = Path(sys.argv[2])
join_path = Path(sys.argv[3])
now = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")

main_payload = {
    "generatedAt": now,
    "mode": "main",
    "e2eExpectedStage": "bot-released",
    "e2eMaxSessionAgeMinutes": 180,
    "passed": True,
    "summary": {"passed": 5, "failed": 0, "skipped": 0},
    "steps": [
        {"name": "public SSO smoke", "status": "passed", "exitCode": 0},
        {"name": "public admission smoke", "status": "passed", "exitCode": 0},
        {"name": "public Web auth browser smoke", "status": "passed", "exitCode": 0},
        {"name": "admission production readiness", "status": "passed", "exitCode": 0},
        {"name": "real QQ admission E2E bot-released wait", "status": "passed", "exitCode": 0},
    ],
}
koishi_payload = {
    "generatedAt": now,
    "mode": "koishi",
    "e2eExpectedStage": "bot-released",
    "e2eMaxSessionAgeMinutes": 180,
    "passed": True,
    "summary": {"passed": 1, "failed": 0, "skipped": 0},
    "steps": [
        {"name": "Koishi admission production evidence", "status": "passed", "exitCode": 0},
    ],
}
join_payload = {
    "generatedAt": now,
    "expectedStage": "bot-released",
    "passed": True,
    "summary": {"passed": 30, "failed": 0},
    "admissionPublicBaseURL": "https://join.stuhelper.com",
    "maxSessionAgeMinutes": 180,
    "input": {
        "platform": "qq",
        "guildID": "178037297",
        "qqID": "123456789",
        "botSelfID": "2118785781",
        "lookbackHours": 24,
    },
    "session": {
        "id": "sess-1",
        "platform": "qq",
        "botSelfID": "2118785781",
        "guildID": "178037297",
        "qqID": "123456789",
        "tokenConsumed": True,
        "status": "verified",
        "cancelledAtPresent": True,
        "botReleaseRecorded": True,
        "sessionAgeSeconds": 600,
        "updatedAgeSeconds": 60,
    },
    "qqBinding": {"bound": True, "boundAt": now},
    "studentVerification": {
        "activeCredentialCount": 1,
        "kinds": ["school_email_otp"],
        "schoolIDs": [4111010006],
    },
    "freshmanApplications": {"count": 0, "statuses": []},
    "checks": [
        {"name": "latest session is fresh enough for this E2E run", "passed": True},
        {"name": "token was consumed by authenticated user", "passed": True},
        {"name": "release requires active student verification credential", "passed": True},
        {"name": "backend recorded successful bot release", "passed": True},
        {"name": "bot release evidence is fresh enough for this E2E run", "passed": True},
    ],
}
main_path.write_text(json.dumps(main_payload, indent=2) + "\n", encoding="utf-8")
koishi_path.write_text(json.dumps(koishi_payload, indent=2) + "\n", encoding="utf-8")
join_path.write_text(json.dumps(join_payload, indent=2) + "\n", encoding="utf-8")
PY

ADMISSION_MVP_FINAL_MAIN_EVIDENCE_FILE="${fresh_main}" \
ADMISSION_MVP_FINAL_KOISHI_EVIDENCE_FILE="${fresh_koishi}" \
ADMISSION_MVP_FINAL_JOIN_E2E_EVIDENCE_FILE="${fresh_join}" \
ADMISSION_MVP_FINAL_VERIFY_FILE="${verify_file}" \
ADMISSION_MVP_FINAL_MAX_EVIDENCE_AGE_MINUTES=180 \
"${VERIFY_SCRIPT}" >/tmp/admission-mvp-final-evidence-verify-contract.stdout

python3 - "${verify_file}" <<'PY' || fail "fresh final evidence verification JSON assertion failed"
import json
import sys
from pathlib import Path

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))

def require(condition, message):
    if not condition:
        raise SystemExit(message)

require(payload.get("passed") is True, "verifier did not pass")
summary = payload.get("summary", {})
require(summary.get("failed") == 0, f"failed checks: {summary}")
names = [item.get("name") for item in payload.get("checks", [])]
for name in (
    "main evidence includes real QQ bot-released E2E step",
    "main evidence real QQ bot-released E2E step passed",
    "koishi evidence does not contain real QQ E2E placeholders",
    "join E2E evidence includes passed check: release requires active student verification credential",
):
    require(name in names, f"missing check {name}")
PY

stale_main="${tmpdir}/stale-main.json"
cp "${fresh_main}" "${stale_main}"
python3 - "${stale_main}" <<'PY'
import json
import sys
from datetime import datetime, timedelta, timezone
from pathlib import Path

path = Path(sys.argv[1])
payload = json.loads(path.read_text(encoding="utf-8"))
payload["generatedAt"] = (datetime.now(timezone.utc) - timedelta(minutes=181)).strftime("%Y-%m-%dT%H:%M:%SZ")
path.write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")
PY

stale_verify="${tmpdir}/stale-verify.json"
if ADMISSION_MVP_FINAL_MAIN_EVIDENCE_FILE="${stale_main}" \
  ADMISSION_MVP_FINAL_KOISHI_EVIDENCE_FILE="${fresh_koishi}" \
  ADMISSION_MVP_FINAL_JOIN_E2E_EVIDENCE_FILE="${fresh_join}" \
  ADMISSION_MVP_FINAL_VERIFY_FILE="${stale_verify}" \
  ADMISSION_MVP_FINAL_MAX_EVIDENCE_AGE_MINUTES=180 \
  "${VERIFY_SCRIPT}" >/tmp/admission-mvp-final-evidence-verify-contract-stale.stdout 2>/tmp/admission-mvp-final-evidence-verify-contract-stale.stderr; then
  fail "final evidence verifier unexpectedly passed with stale main evidence"
fi
assert_contains "${stale_verify}" 'main evidence is fresh'

bad_koishi="${tmpdir}/bad-koishi.json"
cp "${fresh_koishi}" "${bad_koishi}"
python3 - "${bad_koishi}" <<'PY'
import json
import sys
from pathlib import Path

path = Path(sys.argv[1])
payload = json.loads(path.read_text(encoding="utf-8"))
payload["summary"] = {"passed": 1, "failed": 0, "skipped": 1}
payload["steps"].append({
    "name": "real QQ admission E2E bot-released",
    "status": "skipped",
    "exitCode": 0,
})
path.write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")
PY

bad_koishi_verify="${tmpdir}/bad-koishi-verify.json"
if ADMISSION_MVP_FINAL_MAIN_EVIDENCE_FILE="${fresh_main}" \
  ADMISSION_MVP_FINAL_KOISHI_EVIDENCE_FILE="${bad_koishi}" \
  ADMISSION_MVP_FINAL_JOIN_E2E_EVIDENCE_FILE="${fresh_join}" \
  ADMISSION_MVP_FINAL_VERIFY_FILE="${bad_koishi_verify}" \
  ADMISSION_MVP_FINAL_MAX_EVIDENCE_AGE_MINUTES=180 \
  "${VERIFY_SCRIPT}" >/tmp/admission-mvp-final-evidence-verify-contract-koishi.stdout 2>/tmp/admission-mvp-final-evidence-verify-contract-koishi.stderr; then
  fail "final evidence verifier unexpectedly passed with Koishi E2E placeholder"
fi
assert_contains "${bad_koishi_verify}" 'koishi evidence does not contain real QQ E2E placeholders'

old_join="${tmpdir}/old-join-e2e.json"
cp "${fresh_join}" "${old_join}"
python3 - "${old_join}" <<'PY'
import json
import sys
from pathlib import Path

path = Path(sys.argv[1])
payload = json.loads(path.read_text(encoding="utf-8"))
payload["checks"] = [
    item
    for item in payload.get("checks", [])
    if item.get("name") != "release requires active student verification credential"
]
payload["summary"] = {"passed": len(payload["checks"]), "failed": 0}
path.write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")
PY

old_join_verify="${tmpdir}/old-join-verify.json"
if ADMISSION_MVP_FINAL_MAIN_EVIDENCE_FILE="${fresh_main}" \
  ADMISSION_MVP_FINAL_KOISHI_EVIDENCE_FILE="${fresh_koishi}" \
  ADMISSION_MVP_FINAL_JOIN_E2E_EVIDENCE_FILE="${old_join}" \
  ADMISSION_MVP_FINAL_VERIFY_FILE="${old_join_verify}" \
  ADMISSION_MVP_FINAL_MAX_EVIDENCE_AGE_MINUTES=180 \
  "${VERIFY_SCRIPT}" >/tmp/admission-mvp-final-evidence-verify-contract-old-join.stdout 2>/tmp/admission-mvp-final-evidence-verify-contract-old-join.stderr; then
  fail "final evidence verifier unexpectedly passed with old join E2E evidence missing the active credential release check"
fi
assert_contains "${old_join_verify}" 'join E2E evidence includes passed check: release requires active student verification credential'

bad_join="${tmpdir}/bad-join-e2e.json"
cp "${fresh_join}" "${bad_join}"
python3 - "${bad_join}" <<'PY'
import json
import sys
from pathlib import Path

path = Path(sys.argv[1])
payload = json.loads(path.read_text(encoding="utf-8"))
payload["studentVerification"] = {"activeCredentialCount": 0, "kinds": [], "schoolIDs": []}
for item in payload.get("checks", []):
    if item.get("name") == "release requires active student verification credential":
        item["passed"] = False
payload["summary"] = {
    "passed": len([item for item in payload.get("checks", []) if item.get("passed") is True]),
    "failed": len([item for item in payload.get("checks", []) if item.get("passed") is not True]),
}
payload["passed"] = False
path.write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")
PY

bad_join_verify="${tmpdir}/bad-join-verify.json"
if ADMISSION_MVP_FINAL_MAIN_EVIDENCE_FILE="${fresh_main}" \
  ADMISSION_MVP_FINAL_KOISHI_EVIDENCE_FILE="${fresh_koishi}" \
  ADMISSION_MVP_FINAL_JOIN_E2E_EVIDENCE_FILE="${bad_join}" \
  ADMISSION_MVP_FINAL_VERIFY_FILE="${bad_join_verify}" \
  ADMISSION_MVP_FINAL_MAX_EVIDENCE_AGE_MINUTES=180 \
  "${VERIFY_SCRIPT}" >/tmp/admission-mvp-final-evidence-verify-contract-bad-join.stdout 2>/tmp/admission-mvp-final-evidence-verify-contract-bad-join.stderr; then
  fail "final evidence verifier unexpectedly passed with join E2E evidence missing an active student credential"
fi
assert_contains "${bad_join_verify}" 'join E2E active student credential exists'

assert_contains "${LOCAL_TEST_SCRIPT}" 'admission-mvp-final-evidence-verify-contract\.sh'
assert_contains "${GO_LIVE_DOC}" 'admission-mvp-final-evidence-verify\.sh'
assert_contains "${GO_LIVE_DOC}" 'prod-admission-mvp-final-verify'
assert_contains "${RELEASE_RUNBOOK}" 'admission-mvp-final-evidence-verify\.sh'
assert_contains "${RELEASE_RUNBOOK}" 'prod-admission-mvp-final-verify'
assert_contains "${MAKEFILE}" '^prod-admission-mvp-final-verify:'
assert_contains "${MAKEFILE}" 'admission-mvp-final-evidence-verify\.sh'
assert_contains "${PROD_ENV_EXAMPLE}" '^ADMISSION_MVP_FINAL_MAIN_EVIDENCE_FILE=infra/generated/admission-mvp-final-evidence\.json$'
assert_contains "${PROD_ENV_EXAMPLE}" '^ADMISSION_MVP_FINAL_KOISHI_EVIDENCE_FILE=infra/generated/admission-mvp-final-koishi-evidence\.json$'
assert_contains "${PROD_ENV_EXAMPLE}" '^ADMISSION_MVP_FINAL_JOIN_E2E_EVIDENCE_FILE=infra/generated/admission-join-e2e-evidence\.json$'
assert_contains "${PROD_ENV_EXAMPLE}" '^ADMISSION_MVP_FINAL_VERIFY_FILE=infra/generated/admission-mvp-final-evidence-verify\.json$'
assert_contains "${PROD_ENV_EXAMPLE}" '^ADMISSION_MVP_FINAL_MAX_EVIDENCE_AGE_MINUTES=180$'
assert_contains "${PROD_ENV_EXAMPLE}" '^ADMISSION_MVP_FINAL_EXPECTED_E2E_STAGE=bot-released$'

echo "[admission-mvp-final-evidence-verify-contract] all assertions passed"
