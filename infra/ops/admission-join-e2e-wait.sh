#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
EVIDENCE_SCRIPT="${SCRIPT_DIR}/admission-join-e2e-evidence.sh"

usage() {
  cat <<'USAGE'
Usage: infra/ops/admission-join-e2e-wait.sh

Waits for a real QQ small-account admission E2E evidence stage to pass.
This script is read-only: it never creates admission data. It repeatedly runs
admission-join-e2e-evidence.sh until the requested stage is proven or the
timeout expires.

Required:
  ADMISSION_E2E_QQ_ID                    QQ number used by the real E2E.

Optional:
  ADMISSION_E2E_GUILD_ID                 defaults to 178037297.
  ADMISSION_E2E_EXPECTED_STAGE           join-created | flow-completed | bot-released,
                                         defaults to flow-completed.
  ADMISSION_E2E_WAIT_TIMEOUT_SECONDS     defaults to 900.
  ADMISSION_E2E_WAIT_INTERVAL_SECONDS    defaults to 10.
  ADMISSION_E2E_FINAL_EVIDENCE_FILE      defaults to ADMISSION_E2E_EVIDENCE_FILE
                                          or infra/generated/admission-join-e2e-evidence.json.
  ADMISSION_E2E_WAIT_EVIDENCE_FILE       defaults to infra/generated/admission-join-e2e-wait-evidence.json.

Typical use:
  1. Start this script with ADMISSION_E2E_EXPECTED_STAGE=join-created.
  2. Let the QQ small account apply/join the target group.
  3. After join-created passes, complete the join.stuhelper.com flow.
  4. Run it again with ADMISSION_E2E_EXPECTED_STAGE=flow-completed.
  5. Run it again with ADMISSION_E2E_EXPECTED_STAGE=bot-released to prove
     Koishi has executed and reported the release action.
USAGE
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
  exit 0
fi

log() {
  echo "[stuhelper] $*" >&2
}

die() {
  echo "[stuhelper][error] $*" >&2
  exit 1
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "missing required command: $1"
}

require_positive_int() {
  local name="$1"
  local value="$2"
  [[ "${value}" =~ ^[0-9]+$ && "${value}" -gt 0 ]] || die "${name} must be a positive integer"
}

require_cmd date
require_cmd python3

[[ -x "${EVIDENCE_SCRIPT}" ]] || die "missing executable evidence script: ${EVIDENCE_SCRIPT}"

expected_stage="${ADMISSION_E2E_EXPECTED_STAGE:-flow-completed}"
timeout_seconds="${ADMISSION_E2E_WAIT_TIMEOUT_SECONDS:-900}"
interval_seconds="${ADMISSION_E2E_WAIT_INTERVAL_SECONDS:-10}"
final_evidence_file="${ADMISSION_E2E_FINAL_EVIDENCE_FILE:-${ADMISSION_E2E_EVIDENCE_FILE:-${REPO_ROOT}/infra/generated/admission-join-e2e-evidence.json}}"
wait_evidence_file="${ADMISSION_E2E_WAIT_EVIDENCE_FILE:-${REPO_ROOT}/infra/generated/admission-join-e2e-wait-evidence.json}"

case "${expected_stage}" in
  join-created|flow-completed|bot-released) ;;
  *) die "ADMISSION_E2E_EXPECTED_STAGE must be join-created, flow-completed, or bot-released" ;;
esac
require_positive_int "ADMISSION_E2E_WAIT_TIMEOUT_SECONDS" "${timeout_seconds}"
require_positive_int "ADMISSION_E2E_WAIT_INTERVAL_SECONDS" "${interval_seconds}"

tmpdir="$(mktemp -d)"
cleanup() {
  rm -rf "${tmpdir}"
}
trap cleanup EXIT

attempts_file="${tmpdir}/attempts.jsonl"
: >"${attempts_file}"

started_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
start_epoch="$(date +%s)"
deadline_epoch=$((start_epoch + timeout_seconds))
attempt=0
last_status=1
last_attempt_file=""
last_stderr_file=""

record_attempt() {
  local attempt_no="$1"
  local status="$2"
  local evidence_path="$3"
  local stderr_path="$4"
  python3 - "${attempts_file}" "${attempt_no}" "${status}" "${evidence_path}" "${stderr_path}" <<'PY'
import json
import sys
from pathlib import Path

attempts_path = Path(sys.argv[1])
attempt_no = int(sys.argv[2])
status = int(sys.argv[3])
evidence_path = Path(sys.argv[4])
stderr_path = Path(sys.argv[5])

payload = {
    "attempt": attempt_no,
    "exitStatus": status,
    "passed": False,
    "summary": None,
    "sessionStatus": None,
    "failedChecks": [],
    "errorTail": "",
}

if evidence_path.exists() and evidence_path.stat().st_size > 0:
    try:
        evidence = json.loads(evidence_path.read_text(encoding="utf-8"))
        payload["passed"] = bool(evidence.get("passed"))
        payload["summary"] = evidence.get("summary")
        session = evidence.get("session") or {}
        payload["sessionStatus"] = session.get("status")
        payload["failedChecks"] = [
            item.get("name")
            for item in evidence.get("checks", [])
            if item.get("passed") is not True
        ][:8]
    except Exception as exc:
        payload["failedChecks"] = [f"invalid evidence json: {exc}"]

if stderr_path.exists() and stderr_path.stat().st_size > 0:
    lines = stderr_path.read_text(encoding="utf-8", errors="replace").splitlines()
    payload["errorTail"] = "\n".join(lines[-4:])

with attempts_path.open("a", encoding="utf-8") as handle:
    handle.write(json.dumps(payload, ensure_ascii=True, separators=(",", ":")) + "\n")
PY
}

write_wait_evidence() {
  local passed="$1"
  local final_status="$2"
  local last_attempt="$3"
  local completed_at
  completed_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  if [[ "${wait_evidence_file}" == "-" ]]; then
    python3 - "${attempts_file}" "${passed}" "${expected_stage}" "${started_at}" "${completed_at}" "${timeout_seconds}" "${interval_seconds}" "${last_attempt}" "${final_status}" "${final_evidence_file}" <<'PY'
import json
import sys
from pathlib import Path

attempts_path = Path(sys.argv[1])
passed = sys.argv[2] == "true"
attempts = [
    json.loads(line)
    for line in attempts_path.read_text(encoding="utf-8").splitlines()
    if line.strip()
]
print(json.dumps({
    "generatedAt": sys.argv[5],
    "startedAt": sys.argv[4],
    "completedAt": sys.argv[5],
    "passed": passed,
    "expectedStage": sys.argv[3],
    "timeoutSeconds": int(sys.argv[6]),
    "intervalSeconds": int(sys.argv[7]),
    "attemptsCount": int(sys.argv[8]),
    "finalExitStatus": int(sys.argv[9]),
    "finalEvidenceFile": sys.argv[10],
    "attempts": attempts,
}, ensure_ascii=True, indent=2))
PY
    return
  fi

  mkdir -p "$(dirname "${wait_evidence_file}")"
  python3 - "${attempts_file}" "${passed}" "${expected_stage}" "${started_at}" "${completed_at}" "${timeout_seconds}" "${interval_seconds}" "${last_attempt}" "${final_status}" "${final_evidence_file}" "${wait_evidence_file}" <<'PY'
import json
import sys
from pathlib import Path

attempts_path = Path(sys.argv[1])
passed = sys.argv[2] == "true"
attempts = [
    json.loads(line)
    for line in attempts_path.read_text(encoding="utf-8").splitlines()
    if line.strip()
]
evidence = {
    "generatedAt": sys.argv[5],
    "startedAt": sys.argv[4],
    "completedAt": sys.argv[5],
    "passed": passed,
    "expectedStage": sys.argv[3],
    "timeoutSeconds": int(sys.argv[6]),
    "intervalSeconds": int(sys.argv[7]),
    "attemptsCount": int(sys.argv[8]),
    "finalExitStatus": int(sys.argv[9]),
    "finalEvidenceFile": sys.argv[10],
    "attempts": attempts,
}
Path(sys.argv[11]).write_text(json.dumps(evidence, ensure_ascii=True, indent=2) + "\n", encoding="utf-8")
PY
  log "wrote admission join E2E wait evidence to ${wait_evidence_file}"
}

while true; do
  attempt=$((attempt + 1))
  last_attempt_file="${tmpdir}/attempt-${attempt}.json"
  last_stderr_file="${tmpdir}/attempt-${attempt}.stderr"
  stdout_file="${tmpdir}/attempt-${attempt}.stdout"

  log "checking admission ${expected_stage} evidence attempt=${attempt}"
  set +e
  ADMISSION_E2E_EVIDENCE_FILE="${last_attempt_file}" \
    "${EVIDENCE_SCRIPT}" >"${stdout_file}" 2>"${last_stderr_file}"
  last_status=$?
  set -e

  record_attempt "${attempt}" "${last_status}" "${last_attempt_file}" "${last_stderr_file}"

  if [[ "${last_status}" -eq 0 ]]; then
    if [[ "${final_evidence_file}" != "-" ]]; then
      mkdir -p "$(dirname "${final_evidence_file}")"
      install -m 600 "${last_attempt_file}" "${final_evidence_file}"
      log "wrote admission join E2E evidence to ${final_evidence_file}"
    else
      cat "${last_attempt_file}"
    fi
    write_wait_evidence "true" "${last_status}" "${attempt}"
    log "admission join E2E wait passed for stage ${expected_stage} after ${attempt} attempt(s)"
    exit 0
  fi

  now_epoch="$(date +%s)"
  if (( now_epoch >= deadline_epoch )); then
    write_wait_evidence "false" "${last_status}" "${attempt}"
    echo "[stuhelper][error] admission join E2E wait timed out for stage ${expected_stage} after ${attempt} attempt(s)" >&2
    echo "[stuhelper][error] last attempt stderr:" >&2
    tail -n 8 "${last_stderr_file}" >&2 || true
    exit 1
  fi

  sleep "${interval_seconds}"
done
