#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

usage() {
  cat <<'USAGE'
Usage: infra/ops/admission-mvp-production-evidence.sh

Collects a redacted production evidence bundle for the admission MVP. This is
an orchestration entrypoint; the source-of-truth checks remain the individual
smoke/evidence scripts.

Modes:
  main    Run main StuHelper host checks only. This is the default.
  koishi  Run Koishi/NapCat node checks only.
  all     Run both sets of checks when the current host can access both.

Main-host checks:
  - sso-public-smoke.sh
  - admission-public-smoke.sh
  - public-web-auth-browser-smoke.mjs
  - admission-production-readiness.sh

Koishi-node checks:
  - koishi-admission-production-evidence.sh

Real QQ E2E:
  Set ADMISSION_MVP_PRODUCTION_E2E_REQUIRED=true and ADMISSION_E2E_QQ_ID to
  require a real QQ small-account evidence check. By default the script checks
  ADMISSION_MVP_PRODUCTION_E2E_EXPECTED_STAGE=bot-released with the read-only
  admission-join-e2e-evidence.sh script. Set
  ADMISSION_MVP_PRODUCTION_E2E_WAIT=true to wait via admission-join-e2e-wait.sh.
  The final release gate should require ADMISSION_MVP_PRODUCTION_E2E_REQUIRED=true,
  ADMISSION_MVP_PRODUCTION_E2E_WAIT=true, and bot-released.

Environment:
  ADMISSION_MVP_PRODUCTION_EVIDENCE_MODE       main | koishi | all, default main
  ADMISSION_MVP_PRODUCTION_EVIDENCE_FILE       default infra/generated/admission-mvp-production-evidence.json
  ADMISSION_MVP_PRODUCTION_RUN_SSO_SMOKE       default true
  ADMISSION_MVP_PRODUCTION_RUN_ADMISSION_SMOKE default true
  ADMISSION_MVP_PRODUCTION_RUN_BROWSER_SMOKE   default true
  ADMISSION_MVP_PRODUCTION_BROWSER_SMOKE_EVIDENCE_FILE optional pre-collected
                                               public-web-auth-browser-smoke evidence;
                                               defaults to
                                               infra/generated/public-web-auth-browser-smoke-evidence-current.json
                                               when that file exists
  ADMISSION_MVP_PRODUCTION_BROWSER_SMOKE_MAX_AGE_MINUTES default 180
  ADMISSION_MVP_PRODUCTION_RUN_READINESS       default true
  ADMISSION_MVP_PRODUCTION_RUN_KOISHI          default true for mode koishi/all
  ADMISSION_MVP_PRODUCTION_E2E_REQUIRED        default false
  ADMISSION_MVP_PRODUCTION_E2E_WAIT            default false
  ADMISSION_MVP_PRODUCTION_E2E_EXPECTED_STAGE  default bot-released
  ADMISSION_MVP_PRODUCTION_E2E_MAX_SESSION_AGE_MINUTES default 180

The generated JSON records command names, exit codes and durations. It does not
embed service tokens or raw admission tokens.
USAGE
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
  exit 0
fi

require_cmd python3
require_cmd date

mode="${ADMISSION_MVP_PRODUCTION_EVIDENCE_MODE:-main}"
evidence_file="${ADMISSION_MVP_PRODUCTION_EVIDENCE_FILE:-${REPO_ROOT}/infra/generated/admission-mvp-production-evidence.json}"
run_sso_smoke="${ADMISSION_MVP_PRODUCTION_RUN_SSO_SMOKE:-true}"
run_admission_smoke="${ADMISSION_MVP_PRODUCTION_RUN_ADMISSION_SMOKE:-true}"
run_browser_smoke="${ADMISSION_MVP_PRODUCTION_RUN_BROWSER_SMOKE:-true}"
browser_smoke_default_evidence_file="${REPO_ROOT}/infra/generated/public-web-auth-browser-smoke-evidence-current.json"
browser_smoke_evidence_file="${ADMISSION_MVP_PRODUCTION_BROWSER_SMOKE_EVIDENCE_FILE:-}"
if [[ -z "${browser_smoke_evidence_file}" && -f "${browser_smoke_default_evidence_file}" ]]; then
  browser_smoke_evidence_file="${browser_smoke_default_evidence_file}"
fi
browser_smoke_max_age_minutes="${ADMISSION_MVP_PRODUCTION_BROWSER_SMOKE_MAX_AGE_MINUTES:-180}"
run_readiness="${ADMISSION_MVP_PRODUCTION_RUN_READINESS:-true}"
run_koishi="${ADMISSION_MVP_PRODUCTION_RUN_KOISHI:-true}"
e2e_required="${ADMISSION_MVP_PRODUCTION_E2E_REQUIRED:-false}"
e2e_wait="${ADMISSION_MVP_PRODUCTION_E2E_WAIT:-false}"
e2e_expected_stage="${ADMISSION_MVP_PRODUCTION_E2E_EXPECTED_STAGE:-bot-released}"
e2e_max_session_age_minutes="${ADMISSION_MVP_PRODUCTION_E2E_MAX_SESSION_AGE_MINUTES:-180}"

case "${mode}" in
  main|koishi|all) ;;
  *) die "ADMISSION_MVP_PRODUCTION_EVIDENCE_MODE must be main, koishi, or all" ;;
esac
case "${e2e_expected_stage}" in
  join-created|flow-completed|bot-released) ;;
  *) die "ADMISSION_MVP_PRODUCTION_E2E_EXPECTED_STAGE must be join-created, flow-completed, or bot-released" ;;
esac
[[ "${e2e_max_session_age_minutes}" =~ ^[0-9]+$ && "${e2e_max_session_age_minutes}" -gt 0 ]] || die "ADMISSION_MVP_PRODUCTION_E2E_MAX_SESSION_AGE_MINUTES must be a positive integer"
[[ "${browser_smoke_max_age_minutes}" =~ ^[0-9]+$ && "${browser_smoke_max_age_minutes}" -gt 0 ]] || die "ADMISSION_MVP_PRODUCTION_BROWSER_SMOKE_MAX_AGE_MINUTES must be a positive integer"

normalize_bool() {
  local name="$1"
  local value="$2"
  case "${value}" in
    true|TRUE|1|yes|YES) printf 'true\n' ;;
    false|FALSE|0|no|NO|"") printf 'false\n' ;;
    *) die "${name} must be true or false" ;;
  esac
}

run_sso_smoke="$(normalize_bool ADMISSION_MVP_PRODUCTION_RUN_SSO_SMOKE "${run_sso_smoke}")"
run_admission_smoke="$(normalize_bool ADMISSION_MVP_PRODUCTION_RUN_ADMISSION_SMOKE "${run_admission_smoke}")"
run_browser_smoke="$(normalize_bool ADMISSION_MVP_PRODUCTION_RUN_BROWSER_SMOKE "${run_browser_smoke}")"
run_readiness="$(normalize_bool ADMISSION_MVP_PRODUCTION_RUN_READINESS "${run_readiness}")"
run_koishi="$(normalize_bool ADMISSION_MVP_PRODUCTION_RUN_KOISHI "${run_koishi}")"
e2e_required="$(normalize_bool ADMISSION_MVP_PRODUCTION_E2E_REQUIRED "${e2e_required}")"
e2e_wait="$(normalize_bool ADMISSION_MVP_PRODUCTION_E2E_WAIT "${e2e_wait}")"

steps_jsonl="$(mktemp)"
trap 'rm -f "${steps_jsonl}"' EXIT
pass=0
fail=0
skip=0

record_step() {
  local name="$1"
  local status="$2"
  local exit_code="$3"
  local started_at="$4"
  local finished_at="$5"
  local duration_seconds="$6"
  local detail="${7:-}"

  case "${status}" in
    passed) pass=$((pass + 1)) ;;
    failed) fail=$((fail + 1)) ;;
    skipped) skip=$((skip + 1)) ;;
    *) die "invalid step status: ${status}" ;;
  esac

  python3 - "${steps_jsonl}" "${name}" "${status}" "${exit_code}" "${started_at}" "${finished_at}" "${duration_seconds}" "${detail}" <<'PY'
import json
import sys

path, name, status, exit_code, started_at, finished_at, duration_seconds, detail = sys.argv[1:9]
item = {
    "name": name,
    "status": status,
    "exitCode": int(exit_code),
    "startedAt": started_at,
    "finishedAt": finished_at,
    "durationSeconds": float(duration_seconds),
}
if detail:
    item["detail"] = detail
with open(path, "a", encoding="utf-8") as handle:
    handle.write(json.dumps(item, ensure_ascii=True, separators=(",", ":")) + "\n")
PY
}

record_skip() {
  local name="$1"
  local detail="$2"
  local now
  now="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  warn "skipping ${name}: ${detail}"
  record_step "${name}" "skipped" "0" "${now}" "${now}" "0" "${detail}"
}

record_failure() {
  local name="$1"
  local detail="$2"
  local now
  now="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  record_step "${name}" "failed" "1" "${now}" "${now}" "0" "${detail}"
}

run_step() {
  local name="$1"
  shift
  local started_at finished_at start_epoch end_epoch duration status exit_code
  started_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  start_epoch="$(date +%s)"
  log "running ${name}"
  set +e
  "$@"
  exit_code=$?
  set -e
  end_epoch="$(date +%s)"
  finished_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  duration=$((end_epoch - start_epoch))
  if [[ "${exit_code}" -eq 0 ]]; then
    status="passed"
  else
    status="failed"
  fi
  record_step "${name}" "${status}" "${exit_code}" "${started_at}" "${finished_at}" "${duration}" "$*"
}

should_run_main() {
  [[ "${mode}" == "main" || "${mode}" == "all" ]]
}

should_run_koishi() {
  [[ "${mode}" == "koishi" || "${mode}" == "all" ]]
}

should_consider_e2e() {
  should_run_main || [[ "${e2e_required}" == "true" || -n "${ADMISSION_E2E_QQ_ID:-}" ]]
}

validate_public_browser_smoke_evidence() {
  [[ -n "${browser_smoke_evidence_file}" ]] || die "ADMISSION_MVP_PRODUCTION_BROWSER_SMOKE_EVIDENCE_FILE is not set"
  python3 - "${browser_smoke_evidence_file}" "${browser_smoke_max_age_minutes}" <<'PY'
import json
import sys
from datetime import datetime, timezone
from pathlib import Path

path = Path(sys.argv[1])
max_age_minutes = int(sys.argv[2])
required_checks = {
    "web-login-page-renders",
    "developer-apps-route-redirects-to-login",
    "identity-route-redirects-to-login",
    "header-login-click-starts-sso",
    "login-signup-click-starts-sso-signup",
    "join-verify-route-renders-spa",
    "join-login-click-starts-sso",
    "join-signup-click-starts-sso-signup",
    "join-mobile-camera-route-allows-camera",
    "id-host-disabled",
}


def parse_time(value):
    if not isinstance(value, str) or not value:
        raise SystemExit("generatedAt is missing")
    normalized = value[:-1] + "+00:00" if value.endswith("Z") else value
    parsed = datetime.fromisoformat(normalized)
    if parsed.tzinfo is None:
        parsed = parsed.replace(tzinfo=timezone.utc)
    return parsed.astimezone(timezone.utc)


def require(condition, message):
    if not condition:
        raise SystemExit(message)


require(path.is_file(), f"missing pre-collected browser smoke evidence: {path}")
payload = json.loads(path.read_text(encoding="utf-8"))
require(payload.get("passed") is True, "pre-collected browser smoke evidence did not pass")
summary = payload.get("summary") or {}
require(summary.get("failed") == 0, f"pre-collected browser smoke has failures: {summary}")
age_seconds = max(0.0, (datetime.now(timezone.utc) - parse_time(payload.get("generatedAt"))).total_seconds())
require(
    age_seconds <= max_age_minutes * 60,
    f"pre-collected browser smoke evidence is stale: ageSeconds={age_seconds:.0f}, maxAgeMinutes={max_age_minutes}",
)
targets = payload.get("targets") or {}
require(targets.get("webBaseURL") == "https://stuhelper.com", f"unexpected webBaseURL: {targets.get('webBaseURL')}")
require(targets.get("joinBaseURL") == "https://join.stuhelper.com", f"unexpected joinBaseURL: {targets.get('joinBaseURL')}")
require(targets.get("ssoBaseURL") == "https://sso.stuhelper.com", f"unexpected ssoBaseURL: {targets.get('ssoBaseURL')}")
require(targets.get("disabledIDBaseURL") == "https://id.stuhelper.com", f"unexpected disabledIDBaseURL: {targets.get('disabledIDBaseURL')}")
checks = {
    item.get("name"): item
    for item in payload.get("checks", [])
    if isinstance(item, dict) and isinstance(item.get("name"), str)
}
missing = sorted(required_checks.difference(checks))
require(not missing, f"pre-collected browser smoke is missing checks: {', '.join(missing)}")
for name in sorted(required_checks):
    require(checks[name].get("passed") is True, f"browser smoke check did not pass: {name}")
mobile = checks["join-mobile-camera-route-allows-camera"]
permissions = {
    item.get("name") or item.get("feature"): item
    for item in mobile.get("browserPermissions", [])
    if isinstance(item, dict)
}
camera_permission = permissions.get("camera") or {}
require(camera_permission.get("allowed") is True, "browser smoke camera permission was not allowed")
captures = {
    item.get("name"): item
    for item in mobile.get("mediaCaptures", [])
    if isinstance(item, dict)
}
camera_capture = captures.get("camera") or {}
require(camera_capture.get("success") is True, "browser smoke camera capture did not succeed")
require(int(camera_capture.get("videoTrackCount") or 0) >= 1, "browser smoke camera capture had no video track")
print(f"validated pre-collected browser smoke evidence: {path}")
PY
}

if should_run_main; then
  if [[ "${run_sso_smoke}" == "true" ]]; then
    run_step "public SSO smoke" "${SCRIPT_DIR}/sso-public-smoke.sh"
  else
    record_skip "public SSO smoke" "ADMISSION_MVP_PRODUCTION_RUN_SSO_SMOKE=false"
  fi

  if [[ "${run_admission_smoke}" == "true" ]]; then
    run_step "public admission smoke" "${SCRIPT_DIR}/admission-public-smoke.sh"
  else
    record_skip "public admission smoke" "ADMISSION_MVP_PRODUCTION_RUN_ADMISSION_SMOKE=false"
  fi

  if [[ "${run_browser_smoke}" == "true" ]]; then
    if [[ -n "${browser_smoke_evidence_file}" ]]; then
      run_step "public Web auth browser smoke" validate_public_browser_smoke_evidence
    else
      run_step "public Web auth browser smoke" node "${SCRIPT_DIR}/public-web-auth-browser-smoke.mjs"
    fi
  else
    record_skip "public Web auth browser smoke" "ADMISSION_MVP_PRODUCTION_RUN_BROWSER_SMOKE=false"
  fi

  if [[ "${run_readiness}" == "true" ]]; then
    run_step "admission production readiness" "${SCRIPT_DIR}/admission-production-readiness.sh"
  else
    record_skip "admission production readiness" "ADMISSION_MVP_PRODUCTION_RUN_READINESS=false"
  fi
fi

if should_run_koishi; then
  if [[ "${run_koishi}" == "true" ]]; then
    run_step "Koishi admission production evidence" "${SCRIPT_DIR}/koishi-admission-production-evidence.sh"
  else
    record_skip "Koishi admission production evidence" "ADMISSION_MVP_PRODUCTION_RUN_KOISHI=false"
  fi
fi

if should_consider_e2e; then
  if [[ -n "${ADMISSION_E2E_QQ_ID:-}" ]]; then
    if [[ "${e2e_wait}" == "true" ]]; then
      run_step "real QQ admission E2E ${e2e_expected_stage} wait" \
        env ADMISSION_E2E_EXPECTED_STAGE="${e2e_expected_stage}" \
        ADMISSION_E2E_MAX_SESSION_AGE_MINUTES="${e2e_max_session_age_minutes}" \
        "${SCRIPT_DIR}/admission-join-e2e-wait.sh"
    else
      run_step "real QQ admission E2E ${e2e_expected_stage} evidence" \
        env ADMISSION_E2E_EXPECTED_STAGE="${e2e_expected_stage}" \
        ADMISSION_E2E_MAX_SESSION_AGE_MINUTES="${e2e_max_session_age_minutes}" \
        "${SCRIPT_DIR}/admission-join-e2e-evidence.sh"
    fi
  elif [[ "${e2e_required}" == "true" ]]; then
    record_failure "real QQ admission E2E ${e2e_expected_stage}" "ADMISSION_E2E_QQ_ID is required when ADMISSION_MVP_PRODUCTION_E2E_REQUIRED=true"
  else
    record_skip "real QQ admission E2E ${e2e_expected_stage}" "ADMISSION_E2E_QQ_ID not set"
  fi
fi

generated_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
mkdir -p "$(dirname "${evidence_file}")"
python3 - \
  "${steps_jsonl}" \
  "${evidence_file}" \
  "${generated_at}" \
  "${mode}" \
  "${e2e_expected_stage}" \
  "${e2e_max_session_age_minutes}" \
  "${pass}" \
  "${fail}" \
  "${skip}" <<'PY'
import json
import sys
from pathlib import Path

steps_path = Path(sys.argv[1])
evidence_path = Path(sys.argv[2])
steps = [
    json.loads(line)
    for line in steps_path.read_text(encoding="utf-8").splitlines()
    if line.strip()
]
bundle = {
    "generatedAt": sys.argv[3],
    "mode": sys.argv[4],
    "e2eExpectedStage": sys.argv[5],
    "e2eMaxSessionAgeMinutes": int(sys.argv[6]),
    "passed": int(sys.argv[8]) == 0,
    "summary": {
        "passed": int(sys.argv[7]),
        "failed": int(sys.argv[8]),
        "skipped": int(sys.argv[9]),
    },
    "steps": steps,
}
tmp_path = evidence_path.with_suffix(evidence_path.suffix + ".tmp")
tmp_path.write_text(json.dumps(bundle, ensure_ascii=True, indent=2) + "\n", encoding="utf-8")
tmp_path.chmod(0o600)
tmp_path.replace(evidence_path)
PY

log "wrote admission MVP production evidence to ${evidence_file}"

if (( fail > 0 )); then
  die "admission MVP production evidence failed: ${fail} failed checks"
fi

log "admission MVP production evidence passed: checks=${pass}, skipped=${skip}"
