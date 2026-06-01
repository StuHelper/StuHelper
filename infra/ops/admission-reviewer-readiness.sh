#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

usage() {
  cat <<'USAGE'
Usage: infra/ops/admission-reviewer-readiness.sh

Read-only production readiness check for freshman material reviewers. It calls
the bot "view freshman application" endpoint, which exercises the same operator
QQ binding, management guild, and admission:freshman:review capability checks as
the mutating approve/reject endpoint, without changing application state.

Environment:
  ADMISSION_REVIEWER_READINESS_APPLICATION_ID   required pending freshman application id
  ADMISSION_REVIEWER_READINESS_OPERATOR_QQ_IDS  required comma-separated operator QQ ids
  ADMISSION_REVIEWER_READINESS_GUILD_ID         defaults to first ADMISSION_READINESS_REQUIRED_GUILD_IDS
  ADMISSION_REVIEWER_READINESS_CHANNEL_ID       optional
  ADMISSION_REVIEWER_READINESS_BASE_URL         default http://127.0.0.1:18080
  ADMISSION_REVIEWER_READINESS_SERVICE_TOKEN    optional; if empty, reads BOT_SERVICE_TOKEN from app container
  ADMISSION_REVIEWER_READINESS_APP_CONTAINER    default stuhelper-prod-app
  ADMISSION_REVIEWER_READINESS_EVIDENCE_FILE    default infra/generated/admission-reviewer-readiness.json
  ADMISSION_REVIEWER_READINESS_REQUIRE_ALL      default false; false means at least one operator must pass
USAGE
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
  exit 0
fi

require_cmd curl
require_cmd python3

application_id="${ADMISSION_REVIEWER_READINESS_APPLICATION_ID:-}"
operator_qq_ids="${ADMISSION_REVIEWER_READINESS_OPERATOR_QQ_IDS:-}"
guild_id="${ADMISSION_REVIEWER_READINESS_GUILD_ID:-}"
channel_id="${ADMISSION_REVIEWER_READINESS_CHANNEL_ID:-}"
base_url="${ADMISSION_REVIEWER_READINESS_BASE_URL:-http://127.0.0.1:18080}"
service_token="${ADMISSION_REVIEWER_READINESS_SERVICE_TOKEN:-}"
app_container="${ADMISSION_REVIEWER_READINESS_APP_CONTAINER:-stuhelper-prod-app}"
evidence_file="${ADMISSION_REVIEWER_READINESS_EVIDENCE_FILE:-${REPO_ROOT}/infra/generated/admission-reviewer-readiness.json}"
require_all="${ADMISSION_REVIEWER_READINESS_REQUIRE_ALL:-false}"

if [[ -z "${application_id}" ]]; then
  die "ADMISSION_REVIEWER_READINESS_APPLICATION_ID is required"
fi
if [[ -z "${operator_qq_ids}" ]]; then
  die "ADMISSION_REVIEWER_READINESS_OPERATOR_QQ_IDS is required"
fi
if [[ -z "${guild_id}" ]]; then
  first_required_guild="${ADMISSION_READINESS_REQUIRED_GUILD_IDS%%,*}"
  guild_id="${first_required_guild//[[:space:]]/}"
fi
if [[ -z "${guild_id}" ]]; then
  die "ADMISSION_REVIEWER_READINESS_GUILD_ID or ADMISSION_READINESS_REQUIRED_GUILD_IDS is required"
fi

case "${require_all}" in
  true|TRUE|1|yes|YES) require_all="true" ;;
  false|FALSE|0|no|NO|"") require_all="false" ;;
  *) die "ADMISSION_REVIEWER_READINESS_REQUIRE_ALL must be true or false" ;;
esac

if [[ -z "${service_token}" ]]; then
  require_cmd docker
  service_token="$(docker exec "${app_container}" sh -lc 'printf %s "$BOT_SERVICE_TOKEN"')"
fi
if [[ -z "${service_token}" ]]; then
  die "bot service token is empty"
fi

tmpdir="$(mktemp -d)"
checks_jsonl="${tmpdir}/checks.jsonl"
trap 'rm -rf "${tmpdir}"' EXIT

trim() {
  local value="$1"
  value="${value#"${value%%[![:space:]]*}"}"
  value="${value%"${value##*[![:space:]]}"}"
  printf '%s' "${value}"
}

request_payload() {
  local operator_qq_id="$1"
  python3 - "${operator_qq_id}" "${guild_id}" "${channel_id}" "${application_id}" <<'PY'
import json
import sys

operator_qq_id, guild_id, channel_id, application_id = sys.argv[1:5]
payload = {
    "operatorQQID": operator_qq_id,
    "guildID": guild_id,
    "rawCommand": f"freshman view readiness {application_id}",
}
if channel_id:
    payload["channelID"] = channel_id
print(json.dumps(payload, ensure_ascii=True, separators=(",", ":")))
PY
}

record_check() {
  local operator_qq_id="$1"
  local status="$2"
  local body_file="$3"
  python3 - "${checks_jsonl}" "${operator_qq_id}" "${status}" "${body_file}" <<'PY'
import json
import sys
from pathlib import Path

checks_path = Path(sys.argv[1])
operator_qq_id = sys.argv[2]
status = int(sys.argv[3])
body_path = Path(sys.argv[4])
raw = body_path.read_text(encoding="utf-8", errors="replace")
passed = False
error_code = ""
error_message = ""
try:
    payload = json.loads(raw) if raw.strip() else {}
except json.JSONDecodeError as exc:
    payload = {}
    error_message = f"invalid JSON response: {exc}"
else:
    passed = status == 200 and payload.get("success") is True
    error = payload.get("error") or {}
    if isinstance(error, dict):
        error_code = str(error.get("code") or "")
        error_message = str(error.get("message") or "")

item = {
    "operatorQQID": operator_qq_id,
    "httpStatus": status,
    "passed": passed,
    "responseBytes": len(raw.encode("utf-8")),
}
if error_code:
    item["errorCode"] = error_code
if error_message:
    item["errorMessage"] = error_message
with checks_path.open("a", encoding="utf-8") as handle:
    handle.write(json.dumps(item, ensure_ascii=True, separators=(",", ":")) + "\n")
PY
}

IFS=',' read -r -a operators <<<"${operator_qq_ids}"
operator_count=0
for raw_operator in "${operators[@]}"; do
  operator_qq_id="$(trim "${raw_operator}")"
  [[ -n "${operator_qq_id}" ]] || continue
  operator_count=$((operator_count + 1))
  body_file="${tmpdir}/body-${operator_count}.json"
  payload="$(request_payload "${operator_qq_id}")"
  stderr_file="${tmpdir}/curl-${operator_count}.stderr"
  set +e
  status="$(
    curl -sS \
      -o "${body_file}" \
      -w "%{http_code}" \
      -X POST \
      "${base_url%/}/api/v1/bot/admission/freshman/applications/${application_id}/view" \
      -H "Authorization: Bearer ${service_token}" \
      -H "Content-Type: application/json" \
      --data "${payload}" \
      2>"${stderr_file}"
  )"
  curl_status=$?
  set -e
  if [[ "${curl_status}" -ne 0 ]]; then
    status="000"
    python3 - "${body_file}" "${stderr_file}" <<'PY'
import json
import sys
from pathlib import Path

body_path = Path(sys.argv[1])
stderr_path = Path(sys.argv[2])
message = stderr_path.read_text(encoding="utf-8", errors="replace").strip()
body_path.write_text(
    json.dumps(
        {
            "success": False,
            "error": {
                "code": "curl_failed",
                "message": message[:300] or "curl failed",
            },
        },
        ensure_ascii=True,
        separators=(",", ":"),
    ),
    encoding="utf-8",
)
PY
  fi
  record_check "${operator_qq_id}" "${status}" "${body_file}"
done

if [[ "${operator_count}" -eq 0 ]]; then
  die "ADMISSION_REVIEWER_READINESS_OPERATOR_QQ_IDS did not contain any operator QQ ids"
fi

generated_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
mkdir -p "$(dirname "${evidence_file}")"
if python3 - \
  "${checks_jsonl}" \
  "${evidence_file}" \
  "${generated_at}" \
  "${application_id}" \
  "${guild_id}" \
  "${base_url}" \
  "${require_all}" <<'PY'
import json
import sys
from pathlib import Path

checks_path = Path(sys.argv[1])
evidence_path = Path(sys.argv[2])
checks = [
    json.loads(line)
    for line in checks_path.read_text(encoding="utf-8").splitlines()
    if line.strip()
]
require_all = sys.argv[7] == "true"
passed_count = sum(1 for item in checks if item.get("passed") is True)
failed_count = len(checks) - passed_count
passed = passed_count == len(checks) if require_all else passed_count > 0
bundle = {
    "generatedAt": sys.argv[3],
    "passed": passed,
    "applicationID": sys.argv[4],
    "guildID": sys.argv[5],
    "baseURL": sys.argv[6],
    "requireAll": require_all,
    "summary": {
        "passed": passed_count,
        "failed": failed_count,
        "total": len(checks),
    },
    "checks": checks,
}
tmp_path = evidence_path.with_suffix(evidence_path.suffix + ".tmp")
tmp_path.write_text(json.dumps(bundle, ensure_ascii=True, indent=2) + "\n", encoding="utf-8")
tmp_path.chmod(0o600)
tmp_path.replace(evidence_path)
if not passed:
    raise SystemExit(1)
PY
then
  log "admission reviewer readiness passed; wrote ${evidence_file}"
else
  die "admission reviewer readiness failed; wrote ${evidence_file}"
fi
