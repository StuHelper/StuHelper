#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
SMOKE_SCRIPT="${REPO_ROOT}/infra/ops/resend-email-channel-smoke.sh"

fail() {
  echo "[resend-email-channel-smoke-contract][error] $*" >&2
  exit 1
}

grep -q '.env.prod.secrets.local' "${SMOKE_SCRIPT}" ||
  fail "script should auto-detect production secrets env"
grep -q '.env.prod.shared' "${SMOKE_SCRIPT}" ||
  fail "script should auto-detect production shared env"

tmpdir="$(mktemp -d)"
cleanup() {
  if [[ -n "${server_pid:-}" ]]; then
    kill "${server_pid}" >/dev/null 2>&1 || true
    wait "${server_pid}" >/dev/null 2>&1 || true
  fi
  rm -rf "${tmpdir}"
}
trap cleanup EXIT

request_file="${tmpdir}/request.json"
port_file="${tmpdir}/port"

python3 - "${request_file}" "${port_file}" <<'PY' &
from http.server import BaseHTTPRequestHandler, HTTPServer
from pathlib import Path
import json
import sys

request_file = Path(sys.argv[1])
port_file = Path(sys.argv[2])


class Handler(BaseHTTPRequestHandler):
    def do_POST(self):
        body = self.rfile.read(int(self.headers.get("Content-Length", "0")))
        request_file.write_text(json.dumps({
            "path": self.path,
            "headers": {key: self.headers.get(key) for key in [
            "Authorization",
            "Content-Type",
            "Idempotency-Key",
            "User-Agent",
            ]},
            "body": body.decode("utf-8"),
        }, ensure_ascii=False), encoding="utf-8")
        payload = json.dumps({"id": "email-contract"}).encode("utf-8")
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(payload)))
        self.end_headers()
        self.wfile.write(payload)

    def log_message(self, *_):
        pass


server = HTTPServer(("127.0.0.1", 0), Handler)
port_file.write_text(str(server.server_address[1]), encoding="utf-8")
server.handle_request()
PY
server_pid=$!

for _ in {1..50}; do
  [[ -s "${port_file}" ]] && break
  sleep 0.1
done
[[ -s "${port_file}" ]] || fail "fake Resend server did not start"
port="$(cat "${port_file}")"

cat >"${tmpdir}/.env.prod.shared" <<ENV
EMAIL_FROM=noreply@notify.stuhelper.com
EMAIL_FROM_NAME=StuHelper 系统邮件
EMAIL_STUDENT_VERIFICATION_SUBJECT=学生认证验证码
EMAIL_TENCENT_TEMPLATE_PURPOSE=学校邮箱认证
EMAIL_TENCENT_TEMPLATE_SCHOOL_NAME=北京航空航天大学
EMAIL_TENCENT_TEMPLATE_EXPIRE_MINUTES=5
EMAIL_RESEND_ENDPOINT=http://127.0.0.1:${port}/emails
ENV
cat >"${tmpdir}/.env.prod.secrets.local" <<'ENV'
EMAIL_RESEND_API_KEY=re_contract_secret
ENV
: >"${tmpdir}/.env.prod.generated"
: >"${tmpdir}/.env.prod.generated.secrets"

output="$(
  ENV_FILE="${tmpdir}/.env.prod.shared" \
  SECRETS_ENV_FILE="${tmpdir}/.env.prod.secrets.local" \
  GENERATED_ENV_FILE="${tmpdir}/.env.prod.generated" \
  GENERATED_SECRET_ENV_FILE="${tmpdir}/.env.prod.generated.secrets" \
  RESEND_EMAIL_SMOKE_TO="student@buaa.edu.cn" \
  RESEND_EMAIL_CHANNEL_SMOKE_EVIDENCE_FILE="${tmpdir}/evidence.json" \
  bash "${SMOKE_SCRIPT}"
)"

[[ "${output}" == *"[resend-email-channel-smoke] ok"* ]] || fail "expected ok output"
[[ "${output}" != *"re_contract_secret"* ]] || fail "output leaked api key"
[[ "${output}" != *"student@buaa.edu.cn"* ]] || fail "output leaked recipient"
[[ -f "${tmpdir}/evidence.json" ]] || fail "missing evidence file"
if grep -qE 're_contract_secret|student@buaa.edu.cn' "${tmpdir}/evidence.json"; then
  fail "evidence leaked api key or full recipient"
fi

python3 - "${request_file}" "${tmpdir}/evidence.json" "${REPO_ROOT}" <<'PY'
import json
import sys
from pathlib import Path

request = json.loads(Path(sys.argv[1]).read_text())
evidence = json.loads(Path(sys.argv[2]).read_text())
repo_root = Path(sys.argv[3])
body = json.loads(request["body"])
html_template = (repo_root / "infra/email-templates/tencent-ses/stuhelper-school-email-otp.html").read_text(encoding="utf-8")
text_template = (repo_root / "infra/email-templates/tencent-ses/stuhelper-school-email-otp.txt").read_text(encoding="utf-8")
expected_html = (html_template
    .replace("{{code}}", "123456")
    .replace("{{purpose}}", "学校邮箱认证")
    .replace("{{school_name}}", "北京航空航天大学")
    .replace("{{expire_minutes}}", "5"))
expected_text = (text_template
    .replace("{{code}}", "123456")
    .replace("{{purpose}}", "学校邮箱认证")
    .replace("{{school_name}}", "北京航空航天大学")
    .replace("{{expire_minutes}}", "5"))

assert request["path"] == "/emails", request
assert request["headers"]["Authorization"] == "Bearer re_contract_secret", request
assert request["headers"]["Content-Type"] == "application/json", request
assert request["headers"]["Idempotency-Key"].startswith("resend-email-channel-smoke-"), request
assert request["headers"]["User-Agent"] == "StuHelper", request
assert body["from"] == "StuHelper 系统邮件 <noreply@notify.stuhelper.com>", body
assert body["to"] == ["student@buaa.edu.cn"], body
assert body["subject"] == "学生认证验证码", body
assert body["html"] == expected_html, body["html"]
assert body["text"] == expected_text, body["text"]
assert evidence["sent"] is True, evidence
assert evidence["emailID"] == "email-contract", evidence
assert evidence["recipientDomain"] == "buaa.edu.cn", evidence
assert evidence["recipientHashPrefix"], evidence
assert evidence["userAgent"] == "StuHelper", evidence
assert evidence["htmlTemplate"].endswith("stuhelper-school-email-otp.html"), evidence
PY

wait "${server_pid}"
server_pid=""

echo "[resend-email-channel-smoke-contract] all assertions passed"
