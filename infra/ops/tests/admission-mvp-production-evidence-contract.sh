#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
EVIDENCE_SCRIPT="${REPO_ROOT}/infra/ops/admission-mvp-production-evidence.sh"
GO_LIVE_DOC="${REPO_ROOT}/docs/guides/production-go-live.md"
RELEASE_RUNBOOK="${REPO_ROOT}/docs/guides/release-runbook.md"
MAKEFILE="${REPO_ROOT}/Makefile"
PROD_ENV_EXAMPLE="${REPO_ROOT}/.env.prod.example"

fail() {
  echo "[admission-mvp-production-evidence-contract][error] $*" >&2
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

[[ -f "${EVIDENCE_SCRIPT}" ]] || fail "missing evidence script: ${EVIDENCE_SCRIPT}"
[[ -f "${GO_LIVE_DOC}" ]] || fail "missing go-live doc: ${GO_LIVE_DOC}"
[[ -f "${RELEASE_RUNBOOK}" ]] || fail "missing release runbook: ${RELEASE_RUNBOOK}"
[[ -f "${MAKEFILE}" ]] || fail "missing Makefile: ${MAKEFILE}"
[[ -f "${PROD_ENV_EXAMPLE}" ]] || fail "missing prod env example: ${PROD_ENV_EXAMPLE}"

bash -n "${EVIDENCE_SCRIPT}"
[[ -x "${EVIDENCE_SCRIPT}" ]] || fail "admission MVP production evidence script must be executable"

assert_contains "${EVIDENCE_SCRIPT}" 'ADMISSION_MVP_PRODUCTION_EVIDENCE_MODE'
assert_contains "${EVIDENCE_SCRIPT}" 'ADMISSION_MVP_PRODUCTION_EVIDENCE_FILE'
assert_contains "${EVIDENCE_SCRIPT}" 'sso-public-smoke\.sh'
assert_contains "${EVIDENCE_SCRIPT}" 'admission-public-smoke\.sh'
assert_contains "${EVIDENCE_SCRIPT}" 'public-web-auth-browser-smoke\.mjs'
assert_contains "${EVIDENCE_SCRIPT}" 'ADMISSION_MVP_PRODUCTION_BROWSER_SMOKE_EVIDENCE_FILE'
assert_contains "${EVIDENCE_SCRIPT}" 'ADMISSION_MVP_PRODUCTION_BROWSER_SMOKE_MAX_AGE_MINUTES'
assert_contains "${EVIDENCE_SCRIPT}" 'public-web-auth-browser-smoke-evidence-current\.json'
assert_contains "${EVIDENCE_SCRIPT}" 'validate_public_browser_smoke_evidence'
assert_contains "${EVIDENCE_SCRIPT}" 'join-root-route-returns-404'
assert_contains "${EVIDENCE_SCRIPT}" 'join-main-web-route-returns-404'
assert_contains "${EVIDENCE_SCRIPT}" 'admission-production-readiness\.sh'
assert_contains "${EVIDENCE_SCRIPT}" 'external-student-source-smoke\.sh'
assert_contains "${EVIDENCE_SCRIPT}" 'ADMISSION_MVP_PRODUCTION_RUN_EXTERNAL_STUDENT_SOURCE_SMOKE'
assert_contains "${EVIDENCE_SCRIPT}" 'EXTERNAL_STUDENT_SOURCE_ENABLED=true requires external-student-source-smoke\.sh'
assert_contains "${EVIDENCE_SCRIPT}" 'koishi-admission-production-evidence\.sh'
assert_contains "${EVIDENCE_SCRIPT}" 'admission-join-e2e-evidence\.sh'
assert_contains "${EVIDENCE_SCRIPT}" 'admission-join-e2e-wait\.sh'
assert_contains "${EVIDENCE_SCRIPT}" 'ADMISSION_MVP_PRODUCTION_E2E_REQUIRED'
assert_contains "${EVIDENCE_SCRIPT}" 'ADMISSION_MVP_PRODUCTION_E2E_WAIT'
assert_contains "${EVIDENCE_SCRIPT}" 'ADMISSION_MVP_PRODUCTION_E2E_MAX_SESSION_AGE_MINUTES'
assert_contains "${EVIDENCE_SCRIPT}" 'ADMISSION_E2E_MAX_SESSION_AGE_MINUTES'
assert_contains "${EVIDENCE_SCRIPT}" 'ADMISSION_E2E_QQ_ID'
assert_contains "${EVIDENCE_SCRIPT}" 'should_consider_e2e'
assert_contains "${EVIDENCE_SCRIPT}" 'bot-released'
assert_contains "${EVIDENCE_SCRIPT}" 'service tokens or raw admission tokens'

tmpdir="$(mktemp -d)"
fake_repo="${tmpdir}/repo"
mkdir -p "${fake_repo}/infra/ops/lib" "${fake_repo}/infra/generated"
cp "${EVIDENCE_SCRIPT}" "${fake_repo}/infra/ops/admission-mvp-production-evidence.sh"

cat >"${fake_repo}/infra/ops/lib/common.sh" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
COMMON_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${COMMON_LIB_DIR}/../../.." && pwd)"
log() { echo "[fake] $*"; }
warn() { echo "[fake][warn] $*" >&2; }
die() { echo "[fake][error] $*" >&2; exit 1; }
require_cmd() { command -v "$1" >/dev/null 2>&1 || die "missing required command: $1"; }
load_env() { :; }
SH
chmod +x "${fake_repo}/infra/ops/lib/common.sh"

for script in \
  sso-public-smoke.sh \
  admission-public-smoke.sh \
  admission-production-readiness.sh \
  external-student-source-smoke.sh \
  koishi-admission-production-evidence.sh \
  admission-join-e2e-evidence.sh \
  admission-join-e2e-wait.sh; do
  cat >"${fake_repo}/infra/ops/${script}" <<SH
#!/usr/bin/env bash
set -euo pipefail
case "\$(basename "\$0")" in
  admission-join-e2e-evidence.sh|admission-join-e2e-wait.sh)
    [[ "\${ADMISSION_E2E_MAX_SESSION_AGE_MINUTES:-}" == "180" ]] || {
      echo "ADMISSION_E2E_MAX_SESSION_AGE_MINUTES was not forwarded" >&2
      exit 42
    }
    ;;
esac
echo "[fake] ${script}"
SH
  chmod +x "${fake_repo}/infra/ops/${script}"
done

cat >"${fake_repo}/infra/ops/public-web-auth-browser-smoke.mjs" <<'JS'
#!/usr/bin/env node
console.log('[fake] public-web-auth-browser-smoke.mjs')
JS
chmod +x "${fake_repo}/infra/ops/public-web-auth-browser-smoke.mjs"

browser_evidence_file="${tmpdir}/browser-smoke.json"
python3 - "${browser_evidence_file}" <<'PY'
import json
import sys
from datetime import datetime, timezone
from pathlib import Path

path = Path(sys.argv[1])
now = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
checks = [
    {"name": "web-login-page-renders", "passed": True},
    {"name": "developer-apps-route-redirects-to-login", "passed": True},
    {"name": "identity-route-redirects-to-login", "passed": True},
    {"name": "header-login-click-starts-sso", "passed": True},
    {"name": "login-signup-click-starts-sso-signup", "passed": True},
    {"name": "join-root-route-returns-404", "passed": True},
    {"name": "join-main-web-route-returns-404", "passed": True},
    {"name": "join-verify-route-renders-spa", "passed": True},
    {
        "name": "join-mobile-camera-route-allows-camera",
        "passed": True,
        "browserPermissions": [{"feature": "camera", "supported": True, "allowed": True}],
        "mediaCaptures": [{"name": "camera", "supported": True, "success": True, "videoTrackCount": 1}],
    },
]
payload = {
    "generatedAt": now,
    "passed": True,
    "targets": {
        "webBaseURL": "https://stuhelper.com",
        "joinBaseURL": "https://join.stuhelper.com",
        "ssoBaseURL": "https://sso.stuhelper.com",
    },
    "summary": {"passed": len(checks), "failed": 0},
    "checks": checks,
}
path.write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")
PY

evidence_file="${tmpdir}/evidence.json"
ADMISSION_MVP_PRODUCTION_EVIDENCE_MODE=all \
ADMISSION_MVP_PRODUCTION_EVIDENCE_FILE="${evidence_file}" \
ADMISSION_MVP_PRODUCTION_E2E_REQUIRED=true \
ADMISSION_MVP_PRODUCTION_E2E_WAIT=true \
ADMISSION_MVP_PRODUCTION_E2E_EXPECTED_STAGE=bot-released \
ADMISSION_MVP_PRODUCTION_E2E_MAX_SESSION_AGE_MINUTES=180 \
ADMISSION_E2E_QQ_ID=123456789 \
"${fake_repo}/infra/ops/admission-mvp-production-evidence.sh" >/tmp/admission-mvp-production-evidence-contract.stdout

python3 - "${evidence_file}" <<'PY' || fail "production evidence JSON assertion failed"
import json
import sys
from pathlib import Path

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))

def require(condition, message):
    if not condition:
        raise SystemExit(message)

require(payload.get("passed") is True, "bundle did not pass")
require(payload.get("mode") == "all", "mode")
require(payload.get("e2eExpectedStage") == "bot-released", "e2e stage")
require(payload.get("e2eMaxSessionAgeMinutes") == 180, "e2e max session age")
summary = payload.get("summary", {})
require(summary.get("passed") == 6, f"passed count: {summary}")
require(summary.get("failed") == 0, f"failed count: {summary}")
require(summary.get("skipped") == 0, f"skipped count: {summary}")
names = [step.get("name") for step in payload.get("steps", [])]
for name in (
    "public SSO smoke",
    "public admission smoke",
    "public Web auth browser smoke",
    "admission production readiness",
    "Koishi admission production evidence",
    "real QQ admission E2E bot-released wait",
):
    require(name in names, f"missing step {name}")
PY

external_student_source_evidence_file="${tmpdir}/external-student-source-evidence.json"
EXTERNAL_STUDENT_SOURCE_ENABLED=true \
ADMISSION_MVP_PRODUCTION_EVIDENCE_MODE=main \
ADMISSION_MVP_PRODUCTION_EVIDENCE_FILE="${external_student_source_evidence_file}" \
ADMISSION_MVP_PRODUCTION_RUN_SSO_SMOKE=false \
ADMISSION_MVP_PRODUCTION_RUN_ADMISSION_SMOKE=false \
ADMISSION_MVP_PRODUCTION_RUN_BROWSER_SMOKE=false \
ADMISSION_MVP_PRODUCTION_RUN_READINESS=false \
"${fake_repo}/infra/ops/admission-mvp-production-evidence.sh" >/tmp/admission-mvp-production-evidence-contract-external-source.stdout

python3 - "${external_student_source_evidence_file}" <<'PY' || fail "external student source evidence JSON assertion failed"
import json
import sys
from pathlib import Path

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
summary = payload.get("summary", {})
steps = {step.get("name"): step for step in payload.get("steps", [])}
external = steps.get("external student source smoke") or {}
if payload.get("passed") is not True:
    raise SystemExit("bundle did not pass")
if summary.get("passed") != 1:
    raise SystemExit(f"expected only external smoke to pass: {summary}")
if summary.get("failed") != 0:
    raise SystemExit(f"failed count: {summary}")
if external.get("status") != "passed":
    raise SystemExit(f"external source smoke did not pass: {external}")
PY

external_student_source_disabled_smoke_file="${tmpdir}/external-student-source-disabled-smoke.json"
if EXTERNAL_STUDENT_SOURCE_ENABLED=true \
  ADMISSION_MVP_PRODUCTION_RUN_EXTERNAL_STUDENT_SOURCE_SMOKE=false \
  ADMISSION_MVP_PRODUCTION_EVIDENCE_MODE=main \
  ADMISSION_MVP_PRODUCTION_EVIDENCE_FILE="${external_student_source_disabled_smoke_file}" \
  ADMISSION_MVP_PRODUCTION_RUN_SSO_SMOKE=false \
  ADMISSION_MVP_PRODUCTION_RUN_ADMISSION_SMOKE=false \
  ADMISSION_MVP_PRODUCTION_RUN_BROWSER_SMOKE=false \
  ADMISSION_MVP_PRODUCTION_RUN_READINESS=false \
  "${fake_repo}/infra/ops/admission-mvp-production-evidence.sh" >/tmp/admission-mvp-production-evidence-contract-external-disabled.stdout 2>/tmp/admission-mvp-production-evidence-contract-external-disabled.stderr; then
  fail "production evidence unexpectedly passed when external student source smoke was disabled"
fi
assert_contains "${external_student_source_disabled_smoke_file}" 'EXTERNAL_STUDENT_SOURCE_ENABLED=true requires external-student-source-smoke\.sh'

cat >"${fake_repo}/infra/ops/public-web-auth-browser-smoke.mjs" <<'JS'
#!/usr/bin/env node
console.error('browser smoke script should not run when pre-collected evidence is configured')
process.exit(77)
JS
chmod +x "${fake_repo}/infra/ops/public-web-auth-browser-smoke.mjs"

precollected_evidence_file="${tmpdir}/precollected-evidence.json"
ADMISSION_MVP_PRODUCTION_EVIDENCE_MODE=main \
ADMISSION_MVP_PRODUCTION_EVIDENCE_FILE="${precollected_evidence_file}" \
ADMISSION_MVP_PRODUCTION_RUN_SSO_SMOKE=false \
ADMISSION_MVP_PRODUCTION_RUN_ADMISSION_SMOKE=false \
ADMISSION_MVP_PRODUCTION_RUN_READINESS=false \
ADMISSION_MVP_PRODUCTION_BROWSER_SMOKE_EVIDENCE_FILE="${browser_evidence_file}" \
ADMISSION_MVP_PRODUCTION_BROWSER_SMOKE_MAX_AGE_MINUTES=180 \
"${fake_repo}/infra/ops/admission-mvp-production-evidence.sh" >/tmp/admission-mvp-production-evidence-contract-precollected.stdout

python3 - "${precollected_evidence_file}" <<'PY' || fail "pre-collected browser evidence JSON assertion failed"
import json
import sys
from pathlib import Path

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
steps = {step.get("name"): step for step in payload.get("steps", [])}
browser = steps.get("public Web auth browser smoke") or {}
if payload.get("passed") is not True:
    raise SystemExit("bundle did not pass")
if browser.get("status") != "passed":
    raise SystemExit(f"browser step did not pass: {browser}")
if "validate_public_browser_smoke_evidence" not in browser.get("detail", ""):
    raise SystemExit(f"browser step did not use evidence validator: {browser}")
PY

default_browser_evidence_file="${fake_repo}/infra/generated/public-web-auth-browser-smoke-evidence-current.json"
cp "${browser_evidence_file}" "${default_browser_evidence_file}"
default_browser_bundle_file="${tmpdir}/default-browser-evidence.json"
ADMISSION_MVP_PRODUCTION_EVIDENCE_MODE=main \
ADMISSION_MVP_PRODUCTION_EVIDENCE_FILE="${default_browser_bundle_file}" \
ADMISSION_MVP_PRODUCTION_RUN_SSO_SMOKE=false \
ADMISSION_MVP_PRODUCTION_RUN_ADMISSION_SMOKE=false \
ADMISSION_MVP_PRODUCTION_RUN_READINESS=false \
ADMISSION_MVP_PRODUCTION_BROWSER_SMOKE_MAX_AGE_MINUTES=180 \
"${fake_repo}/infra/ops/admission-mvp-production-evidence.sh" >/tmp/admission-mvp-production-evidence-contract-default-browser.stdout

python3 - "${default_browser_bundle_file}" <<'PY' || fail "default browser evidence JSON assertion failed"
import json
import sys
from pathlib import Path

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
steps = {step.get("name"): step for step in payload.get("steps", [])}
browser = steps.get("public Web auth browser smoke") or {}
if payload.get("passed") is not True:
    raise SystemExit("bundle did not pass")
if browser.get("status") != "passed":
    raise SystemExit(f"browser step did not pass: {browser}")
if "validate_public_browser_smoke_evidence" not in browser.get("detail", ""):
    raise SystemExit(f"browser step did not use default evidence validator: {browser}")
PY

stale_browser_evidence_file="${tmpdir}/stale-browser-smoke.json"
cp "${browser_evidence_file}" "${stale_browser_evidence_file}"
python3 - "${stale_browser_evidence_file}" <<'PY'
import json
import sys
from datetime import datetime, timedelta, timezone
from pathlib import Path

path = Path(sys.argv[1])
payload = json.loads(path.read_text(encoding="utf-8"))
payload["generatedAt"] = (datetime.now(timezone.utc) - timedelta(minutes=181)).strftime("%Y-%m-%dT%H:%M:%SZ")
path.write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")
PY

stale_browser_bundle_file="${tmpdir}/stale-browser-bundle.json"
if ADMISSION_MVP_PRODUCTION_EVIDENCE_MODE=main \
  ADMISSION_MVP_PRODUCTION_EVIDENCE_FILE="${stale_browser_bundle_file}" \
  ADMISSION_MVP_PRODUCTION_RUN_SSO_SMOKE=false \
  ADMISSION_MVP_PRODUCTION_RUN_ADMISSION_SMOKE=false \
  ADMISSION_MVP_PRODUCTION_RUN_READINESS=false \
  ADMISSION_MVP_PRODUCTION_BROWSER_SMOKE_EVIDENCE_FILE="${stale_browser_evidence_file}" \
  ADMISSION_MVP_PRODUCTION_BROWSER_SMOKE_MAX_AGE_MINUTES=180 \
  "${fake_repo}/infra/ops/admission-mvp-production-evidence.sh" >/tmp/admission-mvp-production-evidence-contract-stale-browser.stdout 2>/tmp/admission-mvp-production-evidence-contract-stale-browser.stderr; then
  fail "production evidence unexpectedly passed with stale pre-collected browser evidence"
fi
assert_contains "${stale_browser_bundle_file}" 'public Web auth browser smoke'

missing_identity_browser_evidence_file="${tmpdir}/missing-identity-browser-smoke.json"
cp "${browser_evidence_file}" "${missing_identity_browser_evidence_file}"
python3 - "${missing_identity_browser_evidence_file}" <<'PY'
import json
import sys
from pathlib import Path

path = Path(sys.argv[1])
payload = json.loads(path.read_text(encoding="utf-8"))
payload["checks"] = [
    item
    for item in payload.get("checks", [])
    if item.get("name") != "identity-route-redirects-to-login"
]
payload["summary"] = {"passed": len(payload["checks"]), "failed": 0}
path.write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")
PY

missing_identity_bundle_file="${tmpdir}/missing-identity-browser-bundle.json"
if ADMISSION_MVP_PRODUCTION_EVIDENCE_MODE=main \
  ADMISSION_MVP_PRODUCTION_EVIDENCE_FILE="${missing_identity_bundle_file}" \
  ADMISSION_MVP_PRODUCTION_RUN_SSO_SMOKE=false \
  ADMISSION_MVP_PRODUCTION_RUN_ADMISSION_SMOKE=false \
  ADMISSION_MVP_PRODUCTION_RUN_READINESS=false \
  ADMISSION_MVP_PRODUCTION_BROWSER_SMOKE_EVIDENCE_FILE="${missing_identity_browser_evidence_file}" \
  ADMISSION_MVP_PRODUCTION_BROWSER_SMOKE_MAX_AGE_MINUTES=180 \
  "${fake_repo}/infra/ops/admission-mvp-production-evidence.sh" >/tmp/admission-mvp-production-evidence-contract-missing-identity-browser.stdout 2>/tmp/admission-mvp-production-evidence-contract-missing-identity-browser.stderr; then
  fail "production evidence unexpectedly passed with pre-collected browser evidence missing the /identity route check"
fi
assert_contains /tmp/admission-mvp-production-evidence-contract-missing-identity-browser.stderr 'pre-collected browser smoke is missing checks: identity-route-redirects-to-login'
python3 - "${missing_identity_bundle_file}" <<'PY' || fail "missing identity browser evidence JSON assertion failed"
import json
import sys
from pathlib import Path

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
steps = {step.get("name"): step for step in payload.get("steps", [])}
browser = steps.get("public Web auth browser smoke") or {}
if payload.get("passed") is not False:
    raise SystemExit("bundle should fail")
if browser.get("status") != "failed":
    raise SystemExit(f"browser step should fail: {browser}")
PY

missing_join_root_browser_evidence_file="${tmpdir}/missing-join-root-browser-smoke.json"
cp "${browser_evidence_file}" "${missing_join_root_browser_evidence_file}"
python3 - "${missing_join_root_browser_evidence_file}" <<'PY'
import json
import sys
from pathlib import Path

path = Path(sys.argv[1])
payload = json.loads(path.read_text(encoding="utf-8"))
payload["checks"] = [
    item
    for item in payload.get("checks", [])
    if item.get("name") != "join-root-route-returns-404"
]
payload["summary"] = {"passed": len(payload["checks"]), "failed": 0}
path.write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")
PY

missing_join_root_bundle_file="${tmpdir}/missing-join-root-browser-bundle.json"
if ADMISSION_MVP_PRODUCTION_EVIDENCE_MODE=main \
  ADMISSION_MVP_PRODUCTION_EVIDENCE_FILE="${missing_join_root_bundle_file}" \
  ADMISSION_MVP_PRODUCTION_RUN_SSO_SMOKE=false \
  ADMISSION_MVP_PRODUCTION_RUN_ADMISSION_SMOKE=false \
  ADMISSION_MVP_PRODUCTION_RUN_READINESS=false \
  ADMISSION_MVP_PRODUCTION_BROWSER_SMOKE_EVIDENCE_FILE="${missing_join_root_browser_evidence_file}" \
  ADMISSION_MVP_PRODUCTION_BROWSER_SMOKE_MAX_AGE_MINUTES=180 \
  "${fake_repo}/infra/ops/admission-mvp-production-evidence.sh" >/tmp/admission-mvp-production-evidence-contract-missing-join-root-browser.stdout 2>/tmp/admission-mvp-production-evidence-contract-missing-join-root-browser.stderr; then
  fail "production evidence unexpectedly passed with pre-collected browser evidence missing the join root denial check"
fi
assert_contains /tmp/admission-mvp-production-evidence-contract-missing-join-root-browser.stderr 'pre-collected browser smoke is missing checks: join-root-route-returns-404'
python3 - "${missing_join_root_bundle_file}" <<'PY' || fail "missing join root browser evidence JSON assertion failed"
import json
import sys
from pathlib import Path

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
steps = {step.get("name"): step for step in payload.get("steps", [])}
browser = steps.get("public Web auth browser smoke") or {}
if payload.get("passed") is not False:
    raise SystemExit("bundle should fail")
if browser.get("status") != "failed":
    raise SystemExit(f"browser step should fail: {browser}")
PY

koishi_evidence_file="${tmpdir}/koishi-evidence.json"
ADMISSION_MVP_PRODUCTION_EVIDENCE_MODE=koishi \
ADMISSION_MVP_PRODUCTION_EVIDENCE_FILE="${koishi_evidence_file}" \
"${fake_repo}/infra/ops/admission-mvp-production-evidence.sh" >/tmp/admission-mvp-production-evidence-contract-koishi.stdout

python3 - "${koishi_evidence_file}" <<'PY' || fail "koishi production evidence JSON assertion failed"
import json
import sys
from pathlib import Path

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))

def require(condition, message):
    if not condition:
        raise SystemExit(message)

summary = payload.get("summary", {})
names = [step.get("name") for step in payload.get("steps", [])]
require(payload.get("passed") is True, "koishi bundle did not pass")
require(payload.get("mode") == "koishi", "koishi mode")
require(summary.get("passed") == 1, f"koishi passed count: {summary}")
require(summary.get("failed") == 0, f"koishi failed count: {summary}")
require(summary.get("skipped") == 0, f"koishi skipped count: {summary}")
require(names == ["Koishi admission production evidence"], f"koishi steps: {names}")
PY

missing_e2e_file="${tmpdir}/missing-e2e.json"
if ADMISSION_MVP_PRODUCTION_EVIDENCE_MODE=main \
  ADMISSION_MVP_PRODUCTION_EVIDENCE_FILE="${missing_e2e_file}" \
  ADMISSION_MVP_PRODUCTION_E2E_REQUIRED=true \
  ADMISSION_MVP_PRODUCTION_RUN_SSO_SMOKE=false \
  ADMISSION_MVP_PRODUCTION_RUN_ADMISSION_SMOKE=false \
  ADMISSION_MVP_PRODUCTION_RUN_BROWSER_SMOKE=false \
  ADMISSION_MVP_PRODUCTION_RUN_READINESS=false \
  "${fake_repo}/infra/ops/admission-mvp-production-evidence.sh" >/tmp/admission-mvp-production-evidence-contract-missing.stdout 2>/tmp/admission-mvp-production-evidence-contract-missing.stderr; then
  fail "production evidence unexpectedly passed when required E2E QQ id is missing"
fi
assert_contains "${missing_e2e_file}" 'ADMISSION_E2E_QQ_ID is required'

assert_contains "${GO_LIVE_DOC}" 'admission-mvp-production-evidence\.sh'
assert_contains "${GO_LIVE_DOC}" 'prod-admission-mvp-final-evidence'
assert_contains "${GO_LIVE_DOC}" 'prod-admission-mvp-final-koishi-evidence'
assert_contains "${RELEASE_RUNBOOK}" 'admission-mvp-production-evidence\.sh'
assert_contains "${RELEASE_RUNBOOK}" 'prod-admission-mvp-final-evidence'
assert_contains "${RELEASE_RUNBOOK}" 'prod-admission-mvp-final-koishi-evidence'
assert_contains "${MAKEFILE}" '^prod-admission-mvp-evidence:'
assert_contains "${MAKEFILE}" '^prod-admission-mvp-final-evidence:'
assert_contains "${MAKEFILE}" '^prod-admission-mvp-final-koishi-evidence:'
assert_contains "${MAKEFILE}" 'ADMISSION_MVP_PRODUCTION_E2E_REQUIRED=true'
assert_contains "${MAKEFILE}" 'ADMISSION_MVP_PRODUCTION_E2E_WAIT=true'
assert_contains "${MAKEFILE}" 'ADMISSION_MVP_PRODUCTION_E2E_MAX_SESSION_AGE_MINUTES=180'
assert_contains "${MAKEFILE}" 'ADMISSION_MVP_PRODUCTION_EVIDENCE_MODE=koishi'
assert_contains "${MAKEFILE}" 'admission-mvp-final-evidence\.json'
assert_contains "${MAKEFILE}" 'admission-mvp-final-koishi-evidence\.json'
assert_contains "${MAKEFILE}" 'admission-mvp-production-evidence\.sh'
assert_contains "${PROD_ENV_EXAMPLE}" '^ADMISSION_MVP_PRODUCTION_E2E_EXPECTED_STAGE=bot-released$'
assert_contains "${PROD_ENV_EXAMPLE}" '^ADMISSION_MVP_PRODUCTION_E2E_MAX_SESSION_AGE_MINUTES=180$'
assert_contains "${PROD_ENV_EXAMPLE}" '^ADMISSION_MVP_PRODUCTION_BROWSER_SMOKE_EVIDENCE_FILE=$'
assert_contains "${PROD_ENV_EXAMPLE}" '^ADMISSION_MVP_PRODUCTION_BROWSER_SMOKE_MAX_AGE_MINUTES=180$'
assert_contains "${PROD_ENV_EXAMPLE}" '^ADMISSION_MVP_PRODUCTION_RUN_EXTERNAL_STUDENT_SOURCE_SMOKE=auto$'

echo "[admission-mvp-production-evidence-contract] all assertions passed"
