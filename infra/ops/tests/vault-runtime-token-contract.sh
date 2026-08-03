#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
TOKEN_SCRIPT="${REPO_ROOT}/infra/ops/vault-runtime-token.sh"
INSTALLER="${REPO_ROOT}/infra/ops/install-vault-token-renewal-timer.sh"
HTTP_HELPER="${REPO_ROOT}/infra/ops/lib/vault-http.py"
MAKEFILE="${REPO_ROOT}/Makefile"

fail() {
  echo "[vault-runtime-token-contract][error] $*" >&2
  exit 1
}

assert_contains() {
  local file="$1"
  local pattern="$2"
  grep -Eq -- "${pattern}" "${file}" || fail "expected ${file} to contain: ${pattern}"
}

for file in "${TOKEN_SCRIPT}" "${INSTALLER}" "${HTTP_HELPER}"; do
  [[ -f "${file}" ]] || fail "missing ${file}"
done
bash -n "${TOKEN_SCRIPT}"
bash -n "${INSTALLER}"
python3 - "${HTTP_HELPER}" <<'PY'
import ast
from pathlib import Path
import sys

ast.parse(Path(sys.argv[1]).read_text())
PY

assert_contains "${TOKEN_SCRIPT}" 'auth/token/create-orphan'
assert_contains "${TOKEN_SCRIPT}" 'no_default_policy: true'
assert_contains "${TOKEN_SCRIPT}" 'auth/token/renew-self'
assert_contains "${TOKEN_SCRIPT}" 'sys/capabilities-self'
assert_contains "${TOKEN_SCRIPT}" '\["create", "read", "update"\]'
assert_contains "${TOKEN_SCRIPT}" '\["deny"\]'
assert_contains "${TOKEN_SCRIPT}" 'VAULT_RUNTIME_TOKEN_PERIOD_SECONDS:-259200'
assert_contains "${TOKEN_SCRIPT}" 'VAULT_RUNTIME_TOKEN_MIN_TTL_SECONDS:-43200'
assert_contains "${TOKEN_SCRIPT}" 'runuser -u "\$\{owner\}"'
assert_contains "${HTTP_HELPER}" 'tokens out of process arguments'
assert_contains "${HTTP_HELPER}" 'NoRedirectHandler'
assert_contains "${HTTP_HELPER}" 'ProxyHandler\(\{\}\)'
assert_contains "${HTTP_HELPER}" 'plaintext Vault HTTP is allowed only on loopback'
assert_contains "${INSTALLER}" 'OnUnitInactiveSec=12h'
assert_contains "${INSTALLER}" 'NoNewPrivileges=true'
assert_contains "${INSTALLER}" 'ProtectSystem=strict'
assert_contains "${INSTALLER}" 'CapabilityBoundingSet='
assert_contains "${INSTALLER}" 'service_unit="stuhelper-vault-token-renewal\.service"'
assert_contains "${INSTALLER}" 'timer_unit="stuhelper-vault-token-renewal\.timer"'
if grep -q 'SYSTEMD_PREFIX' "${INSTALLER}"; then
  fail "Vault renewal installer must use the fixed unit names required by production preflight"
fi
assert_contains "${MAKEFILE}" '^prod-vault-runtime-token:'

tmpdir="$(mktemp -d)"
server_pid=""
cleanup() {
  if [[ -n "${server_pid}" ]]; then
    kill "${server_pid}" >/dev/null 2>&1 || true
    wait "${server_pid}" >/dev/null 2>&1 || true
  fi
  rm -rf -- "${tmpdir}"
}
trap cleanup EXIT

runtime_token='contract-runtime-token-must-not-be-printed'
printf '%s\n' "${runtime_token}" >"${tmpdir}/token"
chmod 0600 "${tmpdir}/token"

python3 - "${tmpdir}/port" "${runtime_token}" <<'PY' &
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
from pathlib import Path
import sys

port_file = Path(sys.argv[1])
expected_token = sys.argv[2]
shared = "secret/data/stuhelper/prod/shared-env"
secrets = "secret/data/stuhelper/prod/secrets-env"
generated = "secret/data/stuhelper/prod/generated-secrets-env"
denied = "secret/data/__stuhelper_runtime_token_denied_probe__"


class Handler(BaseHTTPRequestHandler):
    def log_message(self, format, *args):  # noqa: A003, ANN001
        return

    def respond(self, status, payload):
        body = json.dumps(payload).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def authenticated(self):
        return self.headers.get("X-Vault-Token") == expected_token

    def do_GET(self):  # noqa: N802
        if not self.authenticated():
            self.respond(403, {"errors": ["permission denied"]})
            return
        if self.path == "/v1/auth/token/lookup-self":
            self.respond(200, {"data": {
                "token_policies": ["stuhelper-production-deploy"],
                "renewable": True,
                "orphan": True,
                "period": 259200,
                "ttl": 250000,
            }})
            return
        if self.path.removeprefix("/v1/") in {shared, secrets, generated}:
            self.respond(200, {"data": {"data": {"value": "redacted-test-value"}}})
            return
        self.respond(404, {"errors": ["not found"]})

    def do_POST(self):  # noqa: N802
        if not self.authenticated():
            self.respond(403, {"errors": ["permission denied"]})
            return
        length = int(self.headers.get("Content-Length", "0"))
        payload = json.loads(self.rfile.read(length) or b"{}")
        if self.path == "/v1/auth/token/renew-self":
            self.respond(200, {"auth": {"renewable": True, "lease_duration": 259200}})
            return
        if self.path == "/v1/sys/capabilities-self":
            capabilities = {
                shared: ["read"],
                secrets: ["read"],
                generated: ["create", "read", "update"],
                "auth/token/lookup-self": ["read"],
                "auth/token/renew-self": ["update"],
                "sys/capabilities-self": ["update"],
                "sys/mounts": ["read"],
                denied: ["deny"],
            }
            self.respond(200, {path: capabilities.get(path, ["deny"]) for path in payload["paths"]})
            return
        self.respond(404, {"errors": ["not found"]})


server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
port_file.write_text(str(server.server_port))
server.serve_forever()
PY
server_pid="$!"

for _ in $(seq 1 50); do
  [[ -s "${tmpdir}/port" ]] && break
  sleep 0.1
done
[[ -s "${tmpdir}/port" ]] || fail "fake Vault server did not start"
port="$(<"${tmpdir}/port")"

cat >"${tmpdir}/remote.env" <<EOF
SECRET_BACKEND=vault-kv-v2
SHARED_ENV_SECRET_REF=secret/stuhelper/prod/shared-env
SECRETS_ENV_SECRET_REF=secret/stuhelper/prod/secrets-env
GENERATED_ENV_SECRET_REF=secret/stuhelper/prod/generated-secrets-env
VAULT_ADDR=http://127.0.0.1:${port}
VAULT_NAMESPACE=
VAULT_TOKEN_FILE=${tmpdir}/token
VAULT_KV_MOUNT=secret
VAULT_RUNTIME_TOKEN_POLICY=stuhelper-production-deploy
VAULT_RUNTIME_TOKEN_PERIOD_SECONDS=259200
VAULT_RUNTIME_TOKEN_MIN_TTL_SECONDS=43200
EOF

REMOTE_DEPLOY_CONFIG_FILE="${tmpdir}/remote.env" \
  "${TOKEN_SCRIPT}" check >"${tmpdir}/check.out" 2>"${tmpdir}/check.err"
REMOTE_DEPLOY_CONFIG_FILE="${tmpdir}/remote.env" \
  "${TOKEN_SCRIPT}" renew >"${tmpdir}/renew.out" 2>"${tmpdir}/renew.err"

grep -q 'policy, TTL, and exact secret reads are valid' "${tmpdir}/check.out" ||
  fail "check mode did not report success"
grep -q 'renewed and revalidated' "${tmpdir}/renew.out" ||
  fail "renew mode did not report success"
if grep -R -Fq -- "${runtime_token}" \
  "${tmpdir}/check.out" "${tmpdir}/check.err" "${tmpdir}/renew.out" "${tmpdir}/renew.err"; then
  fail "runtime token leaked into command output"
fi

if python3 "${HTTP_HELPER}" \
  --address http://vault.example.com \
  --token-file "${tmpdir}/token" \
  --method GET \
  --path auth/token/lookup-self \
  --output-file "${tmpdir}/unexpected.json" \
  >"${tmpdir}/remote-http.out" 2>"${tmpdir}/remote-http.err"; then
  fail "Vault HTTP helper accepted non-loopback plaintext transport"
fi
grep -q 'plaintext Vault HTTP is allowed only on loopback' "${tmpdir}/remote-http.err" ||
  fail "plaintext transport rejection did not explain the failure"

echo "[vault-runtime-token-contract] all assertions passed"
