#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

usage() {
  cat <<'USAGE'
Usage: infra/ops/admission-mvp-final-evidence-verify.sh

Verifies the two final admission MVP production evidence bundles:
  1. main StuHelper host evidence from prod-admission-mvp-final-evidence
  2. Koishi/NapCat node evidence from prod-admission-mvp-final-koishi-evidence

This script is intentionally read-only. It does not contact production, does
not print secrets, and only validates already collected redacted evidence JSON.

Environment:
  ADMISSION_MVP_FINAL_MAIN_EVIDENCE_FILE
      default infra/generated/admission-mvp-final-evidence.json
  ADMISSION_MVP_FINAL_KOISHI_EVIDENCE_FILE
      default infra/generated/admission-mvp-final-koishi-evidence.json
  ADMISSION_MVP_FINAL_JOIN_E2E_EVIDENCE_FILE
      default infra/generated/admission-join-e2e-evidence.json
  ADMISSION_MVP_FINAL_VERIFY_FILE
      default infra/generated/admission-mvp-final-evidence-verify.json
  ADMISSION_MVP_FINAL_MAX_EVIDENCE_AGE_MINUTES
      default 180
  ADMISSION_MVP_FINAL_EXPECTED_E2E_STAGE
      default bot-released
USAGE
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
  exit 0
fi

require_cmd python3

main_evidence_file="${ADMISSION_MVP_FINAL_MAIN_EVIDENCE_FILE:-${REPO_ROOT}/infra/generated/admission-mvp-final-evidence.json}"
koishi_evidence_file="${ADMISSION_MVP_FINAL_KOISHI_EVIDENCE_FILE:-${REPO_ROOT}/infra/generated/admission-mvp-final-koishi-evidence.json}"
join_e2e_evidence_file="${ADMISSION_MVP_FINAL_JOIN_E2E_EVIDENCE_FILE:-${REPO_ROOT}/infra/generated/admission-join-e2e-evidence.json}"
verify_file="${ADMISSION_MVP_FINAL_VERIFY_FILE:-${REPO_ROOT}/infra/generated/admission-mvp-final-evidence-verify.json}"
max_age_minutes="${ADMISSION_MVP_FINAL_MAX_EVIDENCE_AGE_MINUTES:-180}"
expected_stage="${ADMISSION_MVP_FINAL_EXPECTED_E2E_STAGE:-bot-released}"

[[ "${max_age_minutes}" =~ ^[0-9]+$ && "${max_age_minutes}" -gt 0 ]] || die "ADMISSION_MVP_FINAL_MAX_EVIDENCE_AGE_MINUTES must be a positive integer"
case "${expected_stage}" in
  bot-released) ;;
  *) die "ADMISSION_MVP_FINAL_EXPECTED_E2E_STAGE must be bot-released for final MVP acceptance" ;;
esac

generated_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
mkdir -p "$(dirname "${verify_file}")"

if python3 - \
  "${main_evidence_file}" \
  "${koishi_evidence_file}" \
  "${join_e2e_evidence_file}" \
  "${verify_file}" \
  "${max_age_minutes}" \
  "${expected_stage}" \
  "${generated_at}" <<'PY'
import json
import sys
from datetime import datetime, timezone
from pathlib import Path

main_path = Path(sys.argv[1])
koishi_path = Path(sys.argv[2])
join_e2e_path = Path(sys.argv[3])
verify_path = Path(sys.argv[4])
max_age_minutes = int(sys.argv[5])
expected_stage = sys.argv[6]
generated_at = sys.argv[7]

checks = []


def add_check(name, passed, detail=""):
    item = {"name": name, "passed": bool(passed)}
    if detail:
        item["detail"] = str(detail)
    checks.append(item)
    return bool(passed)


def parse_generated_at(value):
    if not isinstance(value, str) or not value:
        raise ValueError("generatedAt is missing")
    normalized = value[:-1] + "+00:00" if value.endswith("Z") else value
    parsed = datetime.fromisoformat(normalized)
    if parsed.tzinfo is None:
        parsed = parsed.replace(tzinfo=timezone.utc)
    return parsed.astimezone(timezone.utc)


def evidence_age_seconds(payload):
    created = parse_generated_at(payload.get("generatedAt"))
    now = datetime.now(timezone.utc)
    return max(0.0, (now - created).total_seconds())


def load_payload(path, label):
    if not path.is_file():
        add_check(f"{label} evidence file exists", False, path)
        return None
    add_check(f"{label} evidence file exists", True, path)
    try:
        payload = json.loads(path.read_text(encoding="utf-8"))
    except Exception as exc:  # noqa: BLE001
        add_check(f"{label} evidence JSON is valid", False, exc)
        return None
    add_check(f"{label} evidence JSON is valid", True)
    return payload


def step_map(payload):
    result = {}
    for step in payload.get("steps", []):
        if isinstance(step, dict) and isinstance(step.get("name"), str):
            result[step["name"]] = step
    return result


def step_passed(payload, name):
    step = step_map(payload).get(name)
    return isinstance(step, dict) and step.get("status") == "passed"


def evidence_check_passed(payload, name):
    return any(
        isinstance(item, dict) and item.get("name") == name and item.get("passed") is True
        for item in payload.get("checks", [])
    )


def any_step_name(payload, predicate):
    return [
        step.get("name")
        for step in payload.get("steps", [])
        if isinstance(step, dict) and isinstance(step.get("name"), str) and predicate(step.get("name"))
    ]


def check_summary_clean(payload, label):
    summary = payload.get("summary", {})
    add_check(f"{label} summary has zero failed checks", summary.get("failed") == 0, summary)
    add_check(f"{label} summary has zero skipped checks", summary.get("skipped") == 0, summary)
    add_check(f"{label} summary has passed checks", isinstance(summary.get("passed"), int) and summary.get("passed") > 0, summary)


def check_fresh(payload, label):
    try:
        age_seconds = evidence_age_seconds(payload)
    except Exception as exc:  # noqa: BLE001
        add_check(f"{label} evidence generatedAt is parseable", False, exc)
        return
    add_check(f"{label} evidence generatedAt is parseable", True, payload.get("generatedAt"))
    add_check(
        f"{label} evidence is fresh",
        age_seconds <= max_age_minutes * 60,
        f"ageSeconds={age_seconds:.0f}, maxAgeMinutes={max_age_minutes}",
    )


main_payload = load_payload(main_path, "main")
koishi_payload = load_payload(koishi_path, "koishi")
join_e2e_payload = load_payload(join_e2e_path, "join E2E")

if main_payload is not None:
    add_check("main evidence passed flag is true", main_payload.get("passed") is True)
    add_check("main evidence mode is main", main_payload.get("mode") == "main", main_payload.get("mode"))
    add_check("main evidence expected E2E stage is bot-released", main_payload.get("e2eExpectedStage") == expected_stage, main_payload.get("e2eExpectedStage"))
    add_check("main evidence max E2E age matches final gate", main_payload.get("e2eMaxSessionAgeMinutes") == max_age_minutes, main_payload.get("e2eMaxSessionAgeMinutes"))
    check_summary_clean(main_payload, "main")
    check_fresh(main_payload, "main")
    for required_step in (
        "public SSO smoke",
        "public admission smoke",
        "public Web auth browser smoke",
        "admission production readiness",
    ):
        add_check(f"main evidence includes passed step: {required_step}", step_passed(main_payload, required_step))
    e2e_steps = any_step_name(main_payload, lambda name: name.startswith(f"real QQ admission E2E {expected_stage}"))
    add_check("main evidence includes real QQ bot-released E2E step", bool(e2e_steps), e2e_steps)
    add_check(
        "main evidence real QQ bot-released E2E step passed",
        any(step_passed(main_payload, name) for name in e2e_steps),
        e2e_steps,
    )

if koishi_payload is not None:
    add_check("koishi evidence passed flag is true", koishi_payload.get("passed") is True)
    add_check("koishi evidence mode is koishi", koishi_payload.get("mode") == "koishi", koishi_payload.get("mode"))
    check_summary_clean(koishi_payload, "koishi")
    check_fresh(koishi_payload, "koishi")
    add_check(
        "koishi evidence includes passed Koishi admission production evidence step",
        step_passed(koishi_payload, "Koishi admission production evidence"),
    )
    koishi_e2e_steps = any_step_name(koishi_payload, lambda name: name.startswith("real QQ admission E2E"))
    add_check("koishi evidence does not contain real QQ E2E placeholders", not koishi_e2e_steps, koishi_e2e_steps)

if join_e2e_payload is not None:
    add_check("join E2E evidence passed flag is true", join_e2e_payload.get("passed") is True)
    add_check("join E2E evidence expected stage is bot-released", join_e2e_payload.get("expectedStage") == expected_stage, join_e2e_payload.get("expectedStage"))
    add_check("join E2E evidence max age matches final gate", join_e2e_payload.get("maxSessionAgeMinutes") == max_age_minutes, join_e2e_payload.get("maxSessionAgeMinutes"))
    summary = join_e2e_payload.get("summary", {})
    add_check("join E2E evidence summary has zero failed checks", summary.get("failed") == 0, summary)
    add_check("join E2E evidence summary has passed checks", isinstance(summary.get("passed"), int) and summary.get("passed") > 0, summary)
    check_fresh(join_e2e_payload, "join E2E")
    session = join_e2e_payload.get("session") or {}
    qq_binding = join_e2e_payload.get("qqBinding") or {}
    student = join_e2e_payload.get("studentVerification") or {}
    add_check("join E2E session reached verified state", session.get("status") == "verified", session.get("status"))
    add_check("join E2E token was consumed", session.get("tokenConsumed") is True)
    add_check("join E2E QQ binding exists", qq_binding.get("bound") is True)
    add_check("join E2E active student credential exists", int(student.get("activeCredentialCount") or 0) > 0, student)
    add_check("join E2E backend recorded bot release", session.get("botReleaseRecorded") is True)
    add_check("join E2E cancelled marker is present", session.get("cancelledAtPresent") is True)
    for required_check in (
        "latest session is fresh enough for this E2E run",
        "token was consumed by authenticated user",
        "release requires active student verification credential",
        "backend recorded successful bot release",
        "bot release evidence is fresh enough for this E2E run",
    ):
        add_check(
            f"join E2E evidence includes passed check: {required_check}",
            evidence_check_passed(join_e2e_payload, required_check),
        )

passed = all(item["passed"] for item in checks)
summary = {
    "passed": sum(1 for item in checks if item["passed"]),
    "failed": sum(1 for item in checks if not item["passed"]),
}
bundle = {
    "generatedAt": generated_at,
    "passed": passed,
    "expectedE2EStage": expected_stage,
    "maxEvidenceAgeMinutes": max_age_minutes,
    "inputs": {
        "mainEvidenceFile": str(main_path),
        "koishiEvidenceFile": str(koishi_path),
        "joinE2EEvidenceFile": str(join_e2e_path),
    },
    "summary": summary,
    "checks": checks,
}
tmp_path = verify_path.with_suffix(verify_path.suffix + ".tmp")
tmp_path.write_text(json.dumps(bundle, ensure_ascii=True, indent=2) + "\n", encoding="utf-8")
tmp_path.chmod(0o600)
tmp_path.replace(verify_path)

if not passed:
    for item in checks:
        if not item["passed"]:
            detail = item.get("detail")
            if detail:
                print(f"[admission-mvp-final-evidence-verify][failed] {item['name']}: {detail}", file=sys.stderr)
            else:
                print(f"[admission-mvp-final-evidence-verify][failed] {item['name']}", file=sys.stderr)
    raise SystemExit(1)
PY
then
  log "admission MVP final evidence verification passed; wrote ${verify_file}"
else
  die "admission MVP final evidence verification failed; wrote ${verify_file}"
fi
