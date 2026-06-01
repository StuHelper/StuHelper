#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
SCRIPT="${ROOT_DIR}/infra/ops/admission-reviewer-readiness.sh"
MAKEFILE="${ROOT_DIR}/Makefile"
PROD_ENV_EXAMPLE="${ROOT_DIR}/.env.prod.example"
GO_LIVE_DOC="${ROOT_DIR}/docs/guides/production-go-live.md"
RELEASE_RUNBOOK="${ROOT_DIR}/docs/guides/release-runbook.md"

fail() {
  printf '[admission-reviewer-readiness-contract][error] %s\n' "$*" >&2
  exit 1
}

assert_contains() {
  local file="$1"
  local pattern="$2"
  grep -Eq -- "${pattern}" "${file}" || fail "${file#${ROOT_DIR}/} missing pattern: ${pattern}"
}

assert_not_contains() {
  local file="$1"
  local pattern="$2"
  if grep -Eq -- "${pattern}" "${file}"; then
    fail "${file#${ROOT_DIR}/} unexpectedly contains pattern: ${pattern}"
  fi
}

for file in \
  "${SCRIPT}" \
  "${MAKEFILE}" \
  "${PROD_ENV_EXAMPLE}" \
  "${GO_LIVE_DOC}" \
  "${RELEASE_RUNBOOK}"; do
  [[ -f "${file}" ]] || fail "missing file: ${file}"
done

bash -n "${SCRIPT}"
[[ -x "${SCRIPT}" ]] || fail "infra/ops/admission-reviewer-readiness.sh must be executable"

assert_contains "${SCRIPT}" '/api/v1/bot/admission/freshman/applications/\$\{application_id\}/view'
assert_not_contains "${SCRIPT}" '/api/v1/bot/admission/freshman/applications/\$\{application_id\}/review'
assert_contains "${SCRIPT}" 'ADMISSION_REVIEWER_READINESS_OPERATOR_QQ_IDS'
assert_contains "${SCRIPT}" 'ADMISSION_REVIEWER_READINESS_REQUIRE_ALL'
assert_contains "${SCRIPT}" 'Authorization: Bearer \$\{service_token\}'

assert_contains "${PROD_ENV_EXAMPLE}" '^ADMISSION_REVIEWER_READINESS_APPLICATION_ID=$'
assert_contains "${PROD_ENV_EXAMPLE}" '^ADMISSION_REVIEWER_READINESS_OPERATOR_QQ_IDS=$'
assert_contains "${PROD_ENV_EXAMPLE}" '^ADMISSION_REVIEWER_READINESS_GUILD_ID=178037297$'
assert_contains "${PROD_ENV_EXAMPLE}" '^ADMISSION_REVIEWER_READINESS_EVIDENCE_FILE=infra/generated/admission-reviewer-readiness\.json$'
assert_contains "${PROD_ENV_EXAMPLE}" '^ADMISSION_REVIEWER_READINESS_REQUIRE_ALL=false$'

assert_contains "${MAKEFILE}" '^prod-admission-reviewer-readiness:'
assert_contains "${MAKEFILE}" 'admission-reviewer-readiness\.sh'
assert_contains "${MAKEFILE}" 'make prod-admission-reviewer-readiness - verify freshman reviewer QQ binding/capability without mutating data'
assert_contains "${GO_LIVE_DOC}" 'admission-reviewer-readiness\.sh'
assert_contains "${GO_LIVE_DOC}" 'prod-admission-reviewer-readiness'
assert_contains "${RELEASE_RUNBOOK}" 'admission-reviewer-readiness\.sh'
assert_contains "${RELEASE_RUNBOOK}" 'prod-admission-reviewer-readiness'

tmpdir="$(mktemp -d)"
server_script="${tmpdir}/server.py"
port_file="${tmpdir}/port"
log_file="${tmpdir}/server.log"
contract_token='contract-secret-token'

cleanup() {
  if [[ -n "${server_pid:-}" ]]; then
    kill "${server_pid}" >/dev/null 2>&1 || true
    wait "${server_pid}" >/dev/null 2>&1 || true
  fi
  rm -rf "${tmpdir}"
}
trap cleanup EXIT

cat >"${server_script}" <<'PY'
import json
import sys
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer


class Handler(BaseHTTPRequestHandler):
    server_version = "StuHelperAdmissionReviewerReadinessContract/1"

    def log_message(self, format, *args):
        return

    def do_POST(self):
        length = int(self.headers.get("content-length", "0"))
        raw_body = self.rfile.read(length)
        if self.headers.get("authorization") != "Bearer contract-secret-token":
            self.respond(401, {"success": False, "error": {"code": "unauthorized", "message": "unauthorized"}})
            return
        if self.path != "/api/v1/bot/admission/freshman/applications/app-1/view":
            self.respond(404, {"success": False, "error": {"code": "not_found", "message": self.path}})
            return
        try:
            body = json.loads(raw_body.decode("utf-8"))
        except json.JSONDecodeError:
            self.respond(400, {"success": False, "error": {"code": "bad_json", "message": "bad json"}})
            return
        if body.get("guildID") != "178037297":
            self.respond(403, {"success": False, "error": {"code": "admission.management_guild_forbidden", "message": "bad guild"}})
            return
        operator = body.get("operatorQQID")
        if operator == "ready-operator":
            self.respond(200, {"success": True, "data": {"id": "app-1", "status": "pending"}})
            return
        if operator == "unbound-operator":
            self.respond(403, {"success": False, "error": {"code": "admission.operator_unbound", "message": "operator QQ is not bound"}})
            return
        self.respond(403, {"success": False, "error": {"code": "admission.operator_forbidden", "message": "operator lacks review capability"}})

    def respond(self, status, payload):
        data = json.dumps(payload, ensure_ascii=True, separators=(",", ":")).encode("utf-8")
        self.send_response(status)
        self.send_header("content-type", "application/json")
        self.send_header("content-length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)


server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
print(server.server_port, flush=True)
server.serve_forever()
PY

python3 -u "${server_script}" >"${port_file}" 2>"${log_file}" &
server_pid=$!

for _ in {1..50}; do
  if [[ -s "${port_file}" ]]; then
    break
  fi
  if ! kill -0 "${server_pid}" >/dev/null 2>&1; then
    fail "fake reviewer readiness server exited: $(cat "${log_file}")"
  fi
  sleep 0.1
done

[[ -s "${port_file}" ]] || fail "fake reviewer readiness server did not publish port"
port="$(head -n 1 "${port_file}")"
base_url="http://127.0.0.1:${port}"

pass_evidence="${tmpdir}/pass.json"
pass_stdout="${tmpdir}/pass.stdout"
ADMISSION_REVIEWER_READINESS_APPLICATION_ID=app-1 \
ADMISSION_REVIEWER_READINESS_OPERATOR_QQ_IDS=forbidden-operator,ready-operator \
ADMISSION_REVIEWER_READINESS_GUILD_ID=178037297 \
ADMISSION_REVIEWER_READINESS_BASE_URL="${base_url}" \
ADMISSION_REVIEWER_READINESS_SERVICE_TOKEN="${contract_token}" \
ADMISSION_REVIEWER_READINESS_EVIDENCE_FILE="${pass_evidence}" \
  "${SCRIPT}" >"${pass_stdout}"

python3 - "${pass_evidence}" <<'PY'
import json
import sys
from pathlib import Path

evidence = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
assert evidence["passed"] is True, evidence
assert evidence["applicationID"] == "app-1", evidence
assert evidence["guildID"] == "178037297", evidence
assert evidence["requireAll"] is False, evidence
assert evidence["summary"] == {"passed": 1, "failed": 1, "total": 2}, evidence["summary"]
assert [item["operatorQQID"] for item in evidence["checks"]] == ["forbidden-operator", "ready-operator"]
assert evidence["checks"][0]["httpStatus"] == 403
assert evidence["checks"][0]["errorCode"] == "admission.operator_forbidden"
assert evidence["checks"][1]["httpStatus"] == 200
assert evidence["checks"][1]["passed"] is True
assert all(item["responseBytes"] > 0 for item in evidence["checks"])
PY

fail_evidence="${tmpdir}/fail.json"
fail_stdout="${tmpdir}/fail.stdout"
fail_stderr="${tmpdir}/fail.stderr"
if ADMISSION_REVIEWER_READINESS_APPLICATION_ID=app-1 \
  ADMISSION_REVIEWER_READINESS_OPERATOR_QQ_IDS=forbidden-operator,unbound-operator \
  ADMISSION_REVIEWER_READINESS_GUILD_ID=178037297 \
  ADMISSION_REVIEWER_READINESS_BASE_URL="${base_url}" \
  ADMISSION_REVIEWER_READINESS_SERVICE_TOKEN="${contract_token}" \
  ADMISSION_REVIEWER_READINESS_EVIDENCE_FILE="${fail_evidence}" \
    "${SCRIPT}" >"${fail_stdout}" 2>"${fail_stderr}"; then
  fail "expected readiness script to fail when every operator is forbidden"
fi

python3 - "${fail_evidence}" <<'PY'
import json
import sys
from pathlib import Path

evidence = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
assert evidence["passed"] is False, evidence
assert evidence["summary"] == {"passed": 0, "failed": 2, "total": 2}, evidence["summary"]
codes = [item.get("errorCode") for item in evidence["checks"]]
assert codes == ["admission.operator_forbidden", "admission.operator_unbound"], codes
PY

require_all_evidence="${tmpdir}/require-all.json"
if ADMISSION_REVIEWER_READINESS_APPLICATION_ID=app-1 \
  ADMISSION_REVIEWER_READINESS_OPERATOR_QQ_IDS=forbidden-operator,ready-operator \
  ADMISSION_REVIEWER_READINESS_GUILD_ID=178037297 \
  ADMISSION_REVIEWER_READINESS_BASE_URL="${base_url}" \
  ADMISSION_REVIEWER_READINESS_SERVICE_TOKEN="${contract_token}" \
  ADMISSION_REVIEWER_READINESS_EVIDENCE_FILE="${require_all_evidence}" \
  ADMISSION_REVIEWER_READINESS_REQUIRE_ALL=true \
    "${SCRIPT}" >/dev/null 2>"${tmpdir}/require-all.stderr"; then
  fail "expected require-all mode to fail when one operator is forbidden"
fi

python3 - "${require_all_evidence}" <<'PY'
import json
import sys
from pathlib import Path

evidence = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
assert evidence["passed"] is False, evidence
assert evidence["requireAll"] is True, evidence
assert evidence["summary"] == {"passed": 1, "failed": 1, "total": 2}, evidence["summary"]
PY

for output in \
  "${pass_stdout}" \
  "${pass_evidence}" \
  "${fail_stdout}" \
  "${fail_stderr}" \
  "${fail_evidence}" \
  "${require_all_evidence}"; do
  if grep -Fq "${contract_token}" "${output}"; then
    fail "secret token leaked into ${output}"
  fi
done

printf '[admission-reviewer-readiness-contract] all assertions passed\n'
