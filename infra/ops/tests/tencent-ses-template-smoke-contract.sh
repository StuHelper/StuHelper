#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
SMOKE_SCRIPT="${REPO_ROOT}/infra/ops/tencent-ses-template-smoke.sh"

fail() {
  echo "[tencent-ses-template-smoke-contract][error] $*" >&2
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
                "Content-Type",
                "X-TC-Action",
                "X-TC-Version",
                "X-TC-Region",
                "X-TC-Timestamp",
                "Authorization",
            ]},
            "body": body.decode("utf-8"),
        }, ensure_ascii=False), encoding="utf-8")
        payload = json.dumps({
            "Response": {
                "RequestId": "req-contract",
                "TemplateName": "stuhelper-school-email-otp",
                "TemplateStatus": 0,
                "TemplateContent": {"Html": "ignored", "Text": "ignored"},
            }
        }).encode("utf-8")
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
[[ -s "${port_file}" ]] || fail "fake Tencent SES server did not start"
port="$(cat "${port_file}")"

cat >"${tmpdir}/.env.prod.shared" <<ENV
EMAIL_TENCENT_REGION=ap-guangzhou
EMAIL_TENCENT_ENDPOINT=http://127.0.0.1:${port}
EMAIL_TENCENT_TEMPLATE_ID=49779
ENV
cat >"${tmpdir}/.env.prod.secrets.local" <<'ENV'
EMAIL_TENCENT_SECRET_ID=AKIDCONTRACT
EMAIL_TENCENT_SECRET_KEY=contract-secret
ENV
: >"${tmpdir}/.env.prod.generated"
: >"${tmpdir}/.env.prod.generated.secrets"

output="$(
  ENV_FILE="${tmpdir}/.env.prod.shared" \
  SECRETS_ENV_FILE="${tmpdir}/.env.prod.secrets.local" \
  GENERATED_ENV_FILE="${tmpdir}/.env.prod.generated" \
  GENERATED_SECRET_ENV_FILE="${tmpdir}/.env.prod.generated.secrets" \
  TENCENT_SES_TEMPLATE_SMOKE_EVIDENCE_FILE="${tmpdir}/evidence.json" \
  bash "${SMOKE_SCRIPT}"
)"

[[ "${output}" == *"[tencent-ses-template-smoke] ok"* ]] || fail "expected ok output"
[[ "${output}" != *"contract-secret"* ]] || fail "output leaked secret key"
[[ "${output}" != *"AKIDCONTRACT"* ]] || fail "output leaked secret id"
[[ -f "${tmpdir}/evidence.json" ]] || fail "missing evidence file"
if grep -qE 'contract-secret|AKIDCONTRACT' "${tmpdir}/evidence.json"; then
  fail "evidence leaked credentials"
fi

python3 - "${request_file}" "${tmpdir}/evidence.json" <<'PY'
import json
import sys
from pathlib import Path

request = json.loads(Path(sys.argv[1]).read_text())
evidence = json.loads(Path(sys.argv[2]).read_text())
body = json.loads(request["body"])

assert request["path"] == "/", request
assert request["headers"]["X-TC-Action"] == "GetEmailTemplate", request
assert request["headers"]["X-TC-Version"] == "2020-10-02", request
assert request["headers"]["X-TC-Region"] == "ap-guangzhou", request
assert "TC3-HMAC-SHA256 Credential=AKIDCONTRACT/" in request["headers"]["Authorization"], request
assert body == {"TemplateID": 49779}, body
assert evidence["templateID"] == 49779, evidence
assert evidence["templateStatus"] == 0, evidence
assert evidence["templateApproved"] is True, evidence
assert evidence["templateName"] == "stuhelper-school-email-otp", evidence
PY

wait "${server_pid}"
server_pid=""

echo "[tencent-ses-template-smoke-contract] all assertions passed"
