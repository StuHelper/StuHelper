#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
SMOKE_SCRIPT="${REPO_ROOT}/infra/ops/observability-smoke-check.sh"

fail() {
  echo "[observability-smoke-contract][error] $*" >&2
  exit 1
}

cleanup() {
  if [[ -n "${server_pid:-}" ]]; then
    kill "${server_pid}" >/dev/null 2>&1 || true
    wait "${server_pid}" >/dev/null 2>&1 || true
  fi
  [[ -n "${tmpdir:-}" ]] && rm -rf "${tmpdir}"
}
trap cleanup EXIT

assert_contains() {
  local file="$1"
  local pattern="$2"
  if ! grep -Eq -- "${pattern}" "${file}"; then
    fail "expected ${file} to contain pattern: ${pattern}"
  fi
}

[[ -f "${SMOKE_SCRIPT}" ]] || fail "missing smoke script: ${SMOKE_SCRIPT}"
[[ -x "${SMOKE_SCRIPT}" ]] || fail "smoke script must be executable: ${SMOKE_SCRIPT}"

bash -n "${SMOKE_SCRIPT}"

assert_contains "${SMOKE_SCRIPT}" 'OBS_SMOKE_STRICT'
assert_contains "${SMOKE_SCRIPT}" 'OBSERVABILITY_SMOKE_EVIDENCE_FILE'
assert_contains "${SMOKE_SCRIPT}" 'up\{job="app"\}'
assert_contains "${SMOKE_SCRIPT}" 'probe_success\{job="blackbox-http",instance="https://sso\.stuhelper\.com/.well-known/openid-configuration"\}'
assert_contains "${SMOKE_SCRIPT}" 'probe_success\{job="blackbox-tcp",instance="openfga:8081"\}'
assert_contains "${SMOKE_SCRIPT}" 'alert-webhook-sink'

tmpdir="$(mktemp -d)"
port_file="${tmpdir}/port"
evidence_file="${tmpdir}/observability-smoke-evidence.json"
generated_obs_dir="${tmpdir}/generated/observability"
mkdir -p "${generated_obs_dir}/alertmanager"
cat >"${generated_obs_dir}/alertmanager/alertmanager.yml" <<'YAML'
route:
  receiver: webhook
receivers:
  - name: webhook
    webhook_configs:
      - url: "https://alerts.example.com/hook"
YAML

cat >"${tmpdir}/fake-observability-server.py" <<'PY'
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
import json
import sys
from urllib.parse import parse_qs, urlparse

port_file = Path(sys.argv[1])

class Handler(BaseHTTPRequestHandler):
    def log_message(self, *_):
        return

    def text(self, status, body):
        data = body.encode()
        self.send_response(status)
        self.send_header("Content-Type", "text/plain")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def json(self, status, payload):
        data = json.dumps(payload).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def do_GET(self):
        parsed = urlparse(self.path)
        if parsed.path.endswith("/-/ready") or parsed.path.endswith("/ready"):
            self.text(200, "ready")
            return
        if parsed.path == "/grafana/api/health":
            self.json(200, {"database": "ok"})
            return
        if parsed.path == "/prom/api/v1/query":
            query = parse_qs(parsed.query).get("query", [""])[0]
            self.json(200, {
                "status": "success",
                "data": {
                    "resultType": "vector",
                    "result": [{
                        "metric": {"query": query},
                        "value": [123, "1"],
                    }],
                },
            })
            return
        self.text(404, "not found")

server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
port_file.write_text(str(server.server_port))
server.serve_forever()
PY

python3 "${tmpdir}/fake-observability-server.py" "${port_file}" &
server_pid=$!
for _ in {1..50}; do
  [[ -s "${port_file}" ]] && break
  sleep 0.1
done
[[ -s "${port_file}" ]] || fail "fake observability server did not start"
port="$(cat "${port_file}")"
base_url="http://127.0.0.1:${port}"

output="$(
  PROMETHEUS_URL="${base_url}/prom/-/ready" \
  GRAFANA_URL="${base_url}/grafana/api/health" \
  LOKI_URL="${base_url}/loki/ready" \
  TEMPO_URL="${base_url}/tempo/ready" \
  ALERTMANAGER_URL="${base_url}/alertmanager/-/ready" \
  ALLOY_URL="${base_url}/alloy/-/ready" \
  ALERTMANAGER_WEBHOOK_URL="https://alerts.example.com/hook" \
  GENERATED_OBS_DIR="${generated_obs_dir}" \
  OBS_SMOKE_STRICT=true \
  OBS_SMOKE_RETRIES=1 \
  OBSERVABILITY_SMOKE_EVIDENCE_FILE="${evidence_file}" \
  "${SMOKE_SCRIPT}"
)"

printf '%s\n' "${output}" | grep -q 'observability stack healthy' || fail "observability smoke did not report healthy"
[[ -f "${evidence_file}" ]] || fail "observability evidence file was not written"
jq -e '
  .strict == true
  and .passed == true
  and .summary.failed == 0
  and ([.checks[] | select(.kind == "prometheus_query")] | length >= 6)
  and ([.checks[] | select(.name == "Alertmanager webhook configured" and .passed == true)] | length == 1)
' "${evidence_file}" >/dev/null

echo "[observability-smoke-contract] all assertions passed"
