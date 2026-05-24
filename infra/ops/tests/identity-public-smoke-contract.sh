#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
SMOKE_SCRIPT="${REPO_ROOT}/infra/ops/identity-public-smoke.sh"

fail() {
  echo "[identity-public-smoke-contract][error] $*" >&2
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

assert_evidence() {
  local file="$1"
  local mode="$2"
  python3 - "${file}" "${mode}" <<'PY' || fail "identity smoke ${mode} evidence assertion failed"
import json
import sys

path, mode = sys.argv[1], sys.argv[2]
with open(path, "r", encoding="utf-8") as fh:
    evidence = json.load(fh)
checks = evidence.get("checks", [])

def require(condition, message):
    if not condition:
        raise SystemExit(message)

def named(name):
    return [item for item in checks if item.get("name") == name]

def require_no_store(name):
    matches = [item for item in named(name) if item.get("passed") is True]
    require(len(matches) == 1, name)
    details = matches[0].get("details", {})
    require("no-store" in details.get("cacheControl", ""), f"{name} cache-control")
    require("no-cache" in details.get("pragma", ""), f"{name} pragma")

def require_www_authenticate(name, expected):
    matches = [item for item in named(name) if item.get("passed") is True]
    require(len(matches) == 1, name)
    actual = matches[0].get("details", {}).get("wwwAuthenticate", "")
    require(expected in actual, f"{name} www-authenticate")

if mode == "default":
    require(evidence.get("passed") is True, "default smoke did not pass")
    require(evidence.get("summary", {}).get("passed") == 26, "default passed count")
    require(evidence.get("summary", {}).get("failed") == 0, "default failed count")
    require(len(checks) == 26, "default check count")
    require(sum(1 for item in checks if "details" in item) >= 12, "default details count")
    for name in (
        "Identity JWKS",
        "Casdoor JWKS",
        "Identity authorize 重新认证跳转",
        "Identity token 缺少 grant_type 返回 invalid_request",
        "Identity introspect 缺少 client 返回 invalid_client",
        "Identity revoke 缺少 client 返回 invalid_client",
        "Identity token JSON Content-Type 返回 invalid_request",
        "Identity introspect text/plain Content-Type 返回 invalid_request",
        "Identity revoke 缺少 Content-Type 返回 invalid_request",
        "Identity token URL query 参数返回 invalid_request",
        "Identity introspect URL query 参数返回 invalid_request",
        "Identity revoke URL query 参数返回 invalid_request",
        "Identity logout GET 无会话返回 204",
        "Identity logout POST 无会话返回 204",
        "Identity logout GET 无回跳拒绝非 ID Token hint",
        "Identity logout POST URL query 参数返回 invalid logout request",
        "Identity logout POST JSON body 返回 invalid logout request",
    ):
        require(len([item for item in named(name) if item.get("passed") is True]) == 1, name)
    for name in (
        "Identity token 缺少 grant_type 返回 invalid_request",
        "Identity introspect 缺少 client 返回 invalid_client",
        "Identity revoke 缺少 client 返回 invalid_client",
        "Identity token JSON Content-Type 返回 invalid_request",
        "Identity introspect text/plain Content-Type 返回 invalid_request",
        "Identity revoke 缺少 Content-Type 返回 invalid_request",
        "Identity token URL query 参数返回 invalid_request",
        "Identity introspect URL query 参数返回 invalid_request",
        "Identity revoke URL query 参数返回 invalid_request",
        "Identity UserInfo GET 无 bearer 返回 invalid_token",
        "Identity UserInfo POST 无 bearer 返回 invalid_token",
        "Identity UserInfo URL query token 返回 invalid_token",
        "Identity UserInfo body token 返回 invalid_token",
    ):
        require_no_store(name)
    for name in (
        "Identity introspect 缺少 client 返回 invalid_client",
        "Identity revoke 缺少 client 返回 invalid_client",
    ):
        require_www_authenticate(name, 'Basic realm="StuHelper Identity"')
    require_www_authenticate("Identity UserInfo GET 无 bearer 返回 invalid_token", 'Bearer realm="StuHelper Identity", error="invalid_token"')
    require_www_authenticate("Identity UserInfo POST 无 bearer 返回 invalid_token", 'Bearer realm="StuHelper Identity", error="invalid_token"')
    require_www_authenticate("Identity UserInfo URL query token 返回 invalid_token", 'Bearer realm="StuHelper Identity", error="invalid_token"')
    require_www_authenticate("Identity UserInfo body token 返回 invalid_token", 'Bearer realm="StuHelper Identity", error="invalid_token"')
    endpoints = evidence.get("endpoints", {})
    issuer = evidence.get("identityIssuer", "")
    casdoor_issuer = evidence.get("casdoorIssuer", "")
    require(endpoints.get("identityDiscovery") == issuer + "/.well-known/openid-configuration", "identityDiscovery endpoint")
    require(endpoints.get("identityAuthorizationServerMetadata") == issuer + "/.well-known/oauth-authorization-server", "identityAuthorizationServerMetadata endpoint")
    require(endpoints.get("identityToken") == issuer + "/oauth2/token", "identityToken endpoint")
    require(endpoints.get("identityIntrospection") == issuer + "/oauth2/introspect", "identityIntrospection endpoint")
    require(endpoints.get("identityRevocation") == issuer + "/oauth2/revoke", "identityRevocation endpoint")
    require(endpoints.get("identityLogout") == issuer + "/oauth2/logout", "identityLogout endpoint")
    require(endpoints.get("casdoorDiscovery") == casdoor_issuer + "/.well-known/openid-configuration", "casdoorDiscovery endpoint")
    require(endpoints.get("casdoorJWKS") == casdoor_issuer + "/.well-known/jwks", "casdoorJWKS endpoint")
elif mode == "registered":
    require(evidence.get("passed") is True, "registered smoke did not pass")
    require(evidence.get("summary", {}).get("passed") == 27, "registered passed count")
    require(evidence.get("summary", {}).get("failed") == 0, "registered failed count")
    require(len(checks) == 27, "registered check count")
    matches = [
        item for item in named("Identity prompt=none 未登录错误回调")
        if item.get("passed") is True
        and item.get("details", {}).get("clientID") == "registered-client"
        and "error=login_required" in item.get("details", {}).get("location", "")
        and "iss=" in item.get("details", {}).get("location", "")
    ]
    require(len(matches) == 1, "registered prompt=none check")
elif mode == "client_credentials":
    require(evidence.get("passed") is True, "client credentials smoke did not pass")
    require(evidence.get("summary", {}).get("passed") == 40, "client credentials passed count")
    require(evidence.get("summary", {}).get("failed") == 0, "client credentials failed count")
    require(len(checks) == 40, "client credentials check count")
    for name in (
        "Identity prompt=none 未登录错误回调",
        "Identity token 混用 client authentication 返回 invalid_client",
        "Identity introspect 混用 client authentication 返回 invalid_client",
        "Identity revoke 混用 client authentication 返回 invalid_client",
        "Identity token JSON Content-Type 返回 invalid_request",
        "Identity introspect text/plain Content-Type 返回 invalid_request",
        "Identity revoke 缺少 Content-Type 返回 invalid_request",
        "Identity token URL query 参数返回 invalid_request",
        "Identity introspect URL query 参数返回 invalid_request",
        "Identity revoke URL query 参数返回 invalid_request",
        "Identity logout GET 无回跳拒绝非 ID Token hint",
        "Identity logout POST URL query 参数返回 invalid logout request",
        "Identity logout POST JSON body 返回 invalid logout request",
        "Identity UserInfo URL query token 返回 invalid_token",
        "Identity UserInfo body token 返回 invalid_token",
        "Identity token 已认证但缺少 grant_type 返回 invalid_request",
        "Identity introspect 已认证但空 token 返回 invalid_request",
        "Identity revoke 已认证但空 token 返回 invalid_request",
        "Identity client_credentials token 签发",
        "Identity client_credentials introspection active",
        "Identity client_credentials UserInfo 拒绝 app-only token",
        "Identity logout 拒绝 access token id_token_hint",
        "Open Platform resource access API 拒绝未授权随机资源",
        "Identity client_credentials revoke 成功",
        "Identity client_credentials revoke 后 inactive",
    ):
        require(len([item for item in named(name) if item.get("passed") is True]) == 1, name)
    for name in (
        "Identity token 混用 client authentication 返回 invalid_client",
        "Identity introspect 混用 client authentication 返回 invalid_client",
        "Identity revoke 混用 client authentication 返回 invalid_client",
        "Identity token JSON Content-Type 返回 invalid_request",
        "Identity introspect text/plain Content-Type 返回 invalid_request",
        "Identity revoke 缺少 Content-Type 返回 invalid_request",
        "Identity token URL query 参数返回 invalid_request",
        "Identity introspect URL query 参数返回 invalid_request",
        "Identity revoke URL query 参数返回 invalid_request",
        "Identity UserInfo URL query token 返回 invalid_token",
        "Identity UserInfo body token 返回 invalid_token",
        "Identity token 已认证但缺少 grant_type 返回 invalid_request",
        "Identity introspect 已认证但空 token 返回 invalid_request",
        "Identity revoke 已认证但空 token 返回 invalid_request",
        "Identity client_credentials token 签发",
        "Identity client_credentials introspection active",
        "Identity client_credentials UserInfo 拒绝 app-only token",
        "Identity client_credentials revoke 成功",
        "Identity client_credentials revoke 后 inactive",
    ):
        require_no_store(name)
    for name in (
        "Identity token 混用 client authentication 返回 invalid_client",
        "Identity introspect 混用 client authentication 返回 invalid_client",
        "Identity revoke 混用 client authentication 返回 invalid_client",
    ):
        require_www_authenticate(name, 'Basic realm="StuHelper Identity"')
    require_www_authenticate("Identity client_credentials UserInfo 拒绝 app-only token", 'Bearer realm="StuHelper Identity", error="invalid_token"')
    require_www_authenticate("Identity UserInfo URL query token 返回 invalid_token", 'Bearer realm="StuHelper Identity", error="invalid_token"')
    require_www_authenticate("Identity UserInfo body token 返回 invalid_token", 'Bearer realm="StuHelper Identity", error="invalid_token"')
    resource_access = named("Open Platform resource access API 拒绝未授权随机资源")[0]
    require(resource_access.get("details", {}).get("expectedReason") == "fga_denied", "resource access reason")
    require(resource_access.get("details", {}).get("action") == "read", "resource access action")
    endpoints = evidence.get("endpoints", {})
    require(endpoints.get("openPlatformResourceAccessCheck", "").endswith("/api/v1/open-platform/resources/access/check"), "resource access endpoint")
    serialized = json.dumps(evidence, ensure_ascii=False)
    require("registered-secret" not in serialized, "client secret leaked into evidence")
    require("app-only-token" not in serialized, "access token leaked into evidence")
elif mode == "client_credentials_allowed":
    require(evidence.get("passed") is True, "allowed resource smoke did not pass")
    require(evidence.get("summary", {}).get("passed") == 40, "allowed resource passed count")
    require(evidence.get("summary", {}).get("failed") == 0, "allowed resource failed count")
    require(len(checks) == 40, "allowed resource check count")
    for name in (
        "Identity token 混用 client authentication 返回 invalid_client",
        "Identity introspect 混用 client authentication 返回 invalid_client",
        "Identity revoke 混用 client authentication 返回 invalid_client",
        "Identity token JSON Content-Type 返回 invalid_request",
        "Identity introspect text/plain Content-Type 返回 invalid_request",
        "Identity revoke 缺少 Content-Type 返回 invalid_request",
        "Identity token URL query 参数返回 invalid_request",
        "Identity introspect URL query 参数返回 invalid_request",
        "Identity revoke URL query 参数返回 invalid_request",
        "Identity UserInfo URL query token 返回 invalid_token",
        "Identity UserInfo body token 返回 invalid_token",
        "Identity token 已认证但缺少 grant_type 返回 invalid_request",
        "Identity introspect 已认证但空 token 返回 invalid_request",
        "Identity revoke 已认证但空 token 返回 invalid_request",
        "Identity client_credentials token 签发",
        "Identity client_credentials introspection active",
        "Identity client_credentials UserInfo 拒绝 app-only token",
        "Identity client_credentials revoke 成功",
        "Identity client_credentials revoke 后 inactive",
    ):
        require_no_store(name)
    for name in (
        "Identity token 混用 client authentication 返回 invalid_client",
        "Identity introspect 混用 client authentication 返回 invalid_client",
        "Identity revoke 混用 client authentication 返回 invalid_client",
    ):
        require_www_authenticate(name, 'Basic realm="StuHelper Identity"')
    require_www_authenticate("Identity client_credentials UserInfo 拒绝 app-only token", 'Bearer realm="StuHelper Identity", error="invalid_token"')
    require_www_authenticate("Identity UserInfo URL query token 返回 invalid_token", 'Bearer realm="StuHelper Identity", error="invalid_token"')
    require_www_authenticate("Identity UserInfo body token 返回 invalid_token", 'Bearer realm="StuHelper Identity", error="invalid_token"')
    require(len([item for item in named("Identity logout 拒绝 access token id_token_hint") if item.get("passed") is True]) == 1, "logout access-token id_token_hint check")
    require(len([item for item in named("Identity logout GET 无回跳拒绝非 ID Token hint") if item.get("passed") is True]) == 1, "logout invalid id_token_hint without redirect check")
    matches = [
        item for item in named("Open Platform resource access API 允许已授权资源")
        if item.get("passed") is True
        and item.get("details", {}).get("expectedAllowed") == "true"
        and item.get("details", {}).get("expectedReason") == "allowed"
        and item.get("details", {}).get("resourceID") == "granted-resource"
    ]
    require(len(matches) == 1, "allowed resource access check")
    serialized = json.dumps(evidence, ensure_ascii=False)
    require("registered-secret" not in serialized, "client secret leaked into allowed evidence")
    require("app-only-token" not in serialized, "access token leaked into allowed evidence")
elif mode == "failure":
    require(evidence.get("passed") is False, "failure smoke passed unexpectedly")
    require(evidence.get("summary", {}).get("passed") == 0, "failure passed count")
    require(evidence.get("summary", {}).get("failed") == 25, "failure failed count")
    require(len(checks) == 25, "failure check count")
    require(sum(1 for item in checks if item.get("details", {}).get("httpStatus") == "404") >= 24, "failure 404 count")
    web_health = checks[0].get("details", {})
    require(web_health.get("httpStatus") == "404", "failure web health status")
    require("not_found" in web_health.get("bodySnippet", ""), "failure web health body snippet")
elif mode == "unreachable":
    require(evidence.get("passed") is False, "unreachable smoke passed unexpectedly")
    require(evidence.get("summary", {}).get("failed", 0) > 0, "unreachable failed count")
    require(len(checks) == 26, "unreachable check count")
    unreachable = [item for item in checks if item.get("details", {}).get("httpStatus") == "000"]
    require(len(unreachable) >= 15, "unreachable 000 count")
    require(all(item.get("details", {}).get("curlError") for item in unreachable), "unreachable curl errors")
elif mode == "override":
    require(evidence.get("passed") is True, "override smoke did not pass")
    require(evidence.get("summary", {}).get("failed") == 0, "override failed count")
else:
    raise SystemExit(f"unknown evidence assertion mode: {mode}")
PY
}

[[ -f "${SMOKE_SCRIPT}" ]] || fail "missing smoke script: ${SMOKE_SCRIPT}"
[[ -x "${SMOKE_SCRIPT}" ]] || fail "smoke script must be executable: ${SMOKE_SCRIPT}"
command -v jq >/dev/null 2>&1 || fail "missing jq"

bash -n "${SMOKE_SCRIPT}"

assert_contains "${SMOKE_SCRIPT}" 'IDENTITY_ISSUER is required'
assert_contains "${SMOKE_SCRIPT}" 'CASDOOR_ISSUER is required'
assert_contains "${SMOKE_SCRIPT}" 'IDENTITY_PUBLIC_SMOKE_EVIDENCE_FILE'
assert_contains "${SMOKE_SCRIPT}" 'IDENTITY_PUBLIC_SMOKE_CLIENT_ID'
assert_contains "${SMOKE_SCRIPT}" 'IDENTITY_PUBLIC_SMOKE_REDIRECT_URI'
assert_contains "${SMOKE_SCRIPT}" 'IDENTITY_PUBLIC_SMOKE_CLIENT_SECRET'
assert_contains "${SMOKE_SCRIPT}" 'IDENTITY_PUBLIC_SMOKE_CLIENT_CREDENTIALS_SCOPE'
assert_contains "${SMOKE_SCRIPT}" 'IDENTITY_PUBLIC_SMOKE_RESOURCE_ACCESS_RESOURCE_TYPE'
assert_contains "${SMOKE_SCRIPT}" 'IDENTITY_PUBLIC_SMOKE_RESOURCE_ACCESS_RESOURCE_ID'
assert_contains "${SMOKE_SCRIPT}" 'IDENTITY_PUBLIC_SMOKE_RESOURCE_ACCESS_ACTION'
assert_contains "${SMOKE_SCRIPT}" 'IDENTITY_PUBLIC_SMOKE_RESOURCE_ACCESS_EXPECT_ALLOWED'
assert_contains "${SMOKE_SCRIPT}" 'IDENTITY_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS'
assert_contains "${SMOKE_SCRIPT}" '/.well-known/openid-configuration'
assert_contains "${SMOKE_SCRIPT}" '/.well-known/oauth-authorization-server'
assert_contains "${SMOKE_SCRIPT}" '/.well-known/jwks.json'
assert_contains "${SMOKE_SCRIPT}" 'Casdoor JWKS'
assert_contains "${SMOKE_SCRIPT}" '/oauth2/authorize'
assert_contains "${SMOKE_SCRIPT}" '/oauth2/logout'
assert_contains "${SMOKE_SCRIPT}" 'id_token_hint'
assert_contains "${SMOKE_SCRIPT}" '无回跳拒绝非 ID Token hint'
assert_contains "${SMOKE_SCRIPT}" '/oauth2/token'
assert_contains "${SMOKE_SCRIPT}" '/oauth2/introspect'
assert_contains "${SMOKE_SCRIPT}" '/oauth2/revoke'
assert_contains "${SMOKE_SCRIPT}" '/oidc/userinfo'
assert_contains "${SMOKE_SCRIPT}" '/api/v1/open-platform/resources/access/check'
assert_contains "${SMOKE_SCRIPT}" 'code_challenge_methods_supported'
assert_contains "${SMOKE_SCRIPT}" 'response_modes_supported'
assert_contains "${SMOKE_SCRIPT}" '"query"'
assert_contains "${SMOKE_SCRIPT}" 'subject_types_supported'
assert_contains "${SMOKE_SCRIPT}" 'id_token_signing_alg_values_supported'
assert_contains "${SMOKE_SCRIPT}" 'claims_supported'
assert_contains "${SMOKE_SCRIPT}" 'prompt_values_supported'
assert_contains "${SMOKE_SCRIPT}" '"login"'
assert_contains "${SMOKE_SCRIPT}" '"consent"'
assert_contains "${SMOKE_SCRIPT}" 'reauth=1'
assert_contains "${SMOKE_SCRIPT}" 'max_age=0'
assert_contains "${SMOKE_SCRIPT}" 'end_session_endpoint'
assert_contains "${SMOKE_SCRIPT}" 'offline_access'
assert_contains "${SMOKE_SCRIPT}" 'authorization_response_iss_parameter_supported'
assert_contains "${SMOKE_SCRIPT}" 'login_required'
assert_contains "${SMOKE_SCRIPT}" 'invalid_request'
assert_contains "${SMOKE_SCRIPT}" 'invalid_client'
assert_contains "${SMOKE_SCRIPT}" 'Content-Type'
assert_contains "${SMOKE_SCRIPT}" 'application/json'
assert_contains "${SMOKE_SCRIPT}" 'text/plain'
assert_contains "${SMOKE_SCRIPT}" 'URL query 参数'
assert_contains "${SMOKE_SCRIPT}" '混用 client authentication'
assert_contains "${SMOKE_SCRIPT}" 'client_secret@'
assert_contains "${SMOKE_SCRIPT}" 'invalid_token'
assert_contains "${SMOKE_SCRIPT}" 'token_endpoint_auth_methods_supported'
assert_contains "${SMOKE_SCRIPT}" 'revocation_endpoint_auth_methods_supported'
assert_contains "${SMOKE_SCRIPT}" 'introspection_endpoint_auth_methods_supported'
assert_contains "${SMOKE_SCRIPT}" 'client_secret_post'
assert_contains "${SMOKE_SCRIPT}" 'token_type_hint=access_token'
assert_contains "${SMOKE_SCRIPT}" 'token_type == "Bearer"'
assert_contains "${SMOKE_SCRIPT}" 'token_kind == "access_token"'
assert_contains "${SMOKE_SCRIPT}" 'Cache-Control'
assert_contains "${SMOKE_SCRIPT}" 'no-store'
assert_contains "${SMOKE_SCRIPT}" 'Pragma'
assert_contains "${SMOKE_SCRIPT}" 'no-cache'
assert_contains "${SMOKE_SCRIPT}" 'WWW-Authenticate'
assert_contains "${SMOKE_SCRIPT}" 'StuHelper Identity'
assert_contains "${SMOKE_SCRIPT}" 'httpStatus'
assert_contains "${SMOKE_SCRIPT}" 'curlError'
assert_contains "${SMOKE_SCRIPT}" 'bodySnippet'
assert_contains "${SMOKE_SCRIPT}" 'client_credentials'
assert_contains "${SMOKE_SCRIPT}" 'grant_type=client_credentials'
assert_contains "${SMOKE_SCRIPT}" 'fga_denied'
assert_contains "${SMOKE_SCRIPT}" 'expectedAllowed'

tmpdir="$(mktemp -d)"
port_file="${tmpdir}/port"
env_file="${tmpdir}/.env"
bad_env_file="${tmpdir}/bad.env"
generated_env_file="${tmpdir}/.env.generated"
generated_secret_env_file="${tmpdir}/.env.generated.secrets"
generated_obs_dir="${tmpdir}/generated/observability"
identity_evidence_file="${tmpdir}/identity-public-smoke-evidence.json"
registered_evidence_file="${tmpdir}/identity-public-smoke-registered-evidence.json"
client_credentials_evidence_file="${tmpdir}/identity-public-smoke-client-credentials-evidence.json"
client_credentials_allowed_evidence_file="${tmpdir}/identity-public-smoke-client-credentials-allowed-evidence.json"
override_evidence_file="${tmpdir}/identity-public-smoke-override-evidence.json"
failure_evidence_file="${tmpdir}/identity-public-smoke-failure-evidence.json"
unreachable_evidence_file="${tmpdir}/identity-public-smoke-unreachable-evidence.json"
local_refused_evidence_file="${tmpdir}/identity-public-smoke-local-refused-evidence.json"

cat >"${tmpdir}/fake-identity-server.py" <<'PY'
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
import base64
import json
import sys
from urllib.parse import parse_qs, quote, urlencode, urlparse

port_file = Path(sys.argv[1])

class Handler(BaseHTTPRequestHandler):
    revoked_tokens = set()

    def log_message(self, *_):
        return

    def write_json(self, status, payload):
        body = json.dumps(payload).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Cache-Control", "no-store")
        self.send_header("Pragma", "no-cache")
        if status == 401 and payload.get("error") == "invalid_client":
            self.send_header("WWW-Authenticate", 'Basic realm="StuHelper Identity"')
        if status == 401 and payload.get("error") == "invalid_token":
            self.send_header("WWW-Authenticate", 'Bearer realm="StuHelper Identity", error="invalid_token"')
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def write_empty(self, status):
        self.send_response(status)
        self.send_header("Cache-Control", "no-store")
        self.send_header("Pragma", "no-cache")
        self.send_header("Content-Length", "0")
        self.end_headers()

    def write_text(self, status, body):
        raw = body.encode()
        self.send_response(status)
        self.send_header("Content-Type", "text/plain; charset=utf-8")
        self.send_header("Content-Length", str(len(raw)))
        self.end_headers()
        self.wfile.write(raw)

    def basic_auth(self):
        header = self.headers.get("Authorization", "")
        if not header.startswith("Basic "):
            return "", ""
        try:
            decoded = base64.b64decode(header.removeprefix("Basic ")).decode()
        except Exception:
            return "", ""
        if ":" not in decoded:
            return "", ""
        return decoded.split(":", 1)

    def read_form(self):
        length = int(self.headers.get("Content-Length", "0") or "0")
        raw = self.rfile.read(length).decode() if length else ""
        return parse_qs(raw)

    def form_content_type_valid(self):
        content_type = self.headers.get("Content-Type", "")
        return content_type.lower().split(";", 1)[0].strip() == "application/x-www-form-urlencoded"

    def client_authenticated(self, form=None):
        username, password = self.basic_auth()
        has_post_credentials = bool(form and ("client_id" in form or "client_secret" in form))
        if username or password:
            return username == "registered-client" and password == "registered-secret" and not has_post_credentials
        if form:
            return form.get("client_id") == ["registered-client"] and form.get("client_secret") == ["registered-secret"]
        return False

    def do_GET(self):
        base = f"http://127.0.0.1:{self.server.server_port}"
        if self.path == "/health/ready":
            self.write_json(200, {"status": "ok"})
            return
        if self.path in ("/id/.well-known/openid-configuration", "/id/.well-known/oauth-authorization-server"):
            issuer = base + "/id"
            self.write_json(200, {
                "issuer": issuer,
                "authorization_endpoint": issuer + "/oauth2/authorize",
                "token_endpoint": issuer + "/oauth2/token",
                "userinfo_endpoint": issuer + "/oidc/userinfo",
                "jwks_uri": issuer + "/.well-known/jwks.json",
                "revocation_endpoint": issuer + "/oauth2/revoke",
                "introspection_endpoint": issuer + "/oauth2/introspect",
                "end_session_endpoint": issuer + "/oauth2/logout",
                "authorization_response_iss_parameter_supported": True,
                "response_types_supported": ["code"],
                "response_modes_supported": ["query"],
                "grant_types_supported": ["authorization_code", "refresh_token", "client_credentials"],
                "code_challenge_methods_supported": ["S256"],
                "prompt_values_supported": ["none", "login", "consent"],
                "subject_types_supported": ["public"],
                "id_token_signing_alg_values_supported": ["RS256"],
                "token_endpoint_auth_methods_supported": ["client_secret_basic", "client_secret_post"],
                "revocation_endpoint_auth_methods_supported": ["client_secret_basic", "client_secret_post"],
                "introspection_endpoint_auth_methods_supported": ["client_secret_basic", "client_secret_post"],
                "scopes_supported": ["openid", "offline_access"],
                "claims_supported": [
                    "sub",
                    "preferred_username",
                    "name",
                    "email",
                    "email_verified",
                    "phone_number",
                    "phone_number_verified",
                    "identityVerified",
                    "identityType",
                    "studentVerified",
                    "school",
                ],
            })
            return
        if self.path == "/id/.well-known/jwks.json":
            self.write_json(200, {"keys": [{"kid": "identity-test-key", "kty": "RSA", "use": "sig"}]})
            return
        if self.path.startswith("/id/oauth2/authorize"):
            issuer = base + "/id"
            parsed = urlparse(self.path)
            query = parse_qs(parsed.query)
            if query.get("prompt") == ["none"]:
                redirect_uri = query.get("redirect_uri", [""])[0]
                if query.get("client_id") == ["registered-client"] and redirect_uri == base + "/client/callback":
                    self.send_response(302)
                    self.send_header("Location", redirect_uri + "?" + urlencode({
                        "error": "login_required",
                        "state": query.get("state", [""])[0],
                        "iss": issuer,
                    }))
                    self.end_headers()
                    return
                self.write_json(400, {"error": "invalid_request"})
                return
            self.send_response(302)
            if "prompt=login" in self.path or "max_age=0" in self.path:
                target = self.path
                for needle in ("&prompt=login", "prompt=login&", "&max_age=0", "max_age=0&"):
                    target = target.replace(needle, "")
                self.send_header("Location", base + "/id/login?reauth=1&redirect=" + quote(target, safe=""))
                self.end_headers()
                return
            self.send_header("Location", base + "/id/login?redirect=" + quote(self.path, safe=""))
            self.end_headers()
            return
        if urlparse(self.path).path == "/id/oauth2/logout":
            query = parse_qs(urlparse(self.path).query)
            if query.get("id_token_hint") == ["app-only-token"]:
                self.write_text(400, "invalid logout request")
                return
            if query:
                self.write_text(400, "invalid logout request")
                return
            self.write_empty(204)
            return
        if urlparse(self.path).path == "/id/oidc/userinfo":
            self.write_json(401, {"error": "invalid_token"})
            return
        if self.path == "/sso/.well-known/openid-configuration":
            issuer = base + "/sso"
            self.write_json(200, {
                "issuer": issuer,
                "authorization_endpoint": issuer + "/login/oauth/authorize",
                "token_endpoint": issuer + "/api/login/oauth/access_token",
                "jwks_uri": issuer + "/.well-known/jwks",
            })
            return
        if self.path == "/sso/.well-known/jwks":
            self.write_json(200, {"keys": [{"kid": "casdoor-test-key", "kty": "RSA", "use": "sig"}]})
            return
        self.write_json(404, {"error": "not_found", "path": self.path})

    def do_POST(self):
        parsed = urlparse(self.path)
        path = parsed.path
        if path == "/id/oauth2/logout":
            if parsed.query:
                self.write_text(400, "invalid logout request")
                return
            if int(self.headers.get("Content-Length", "0") or "0") > 0 and not self.form_content_type_valid():
                self.write_text(400, "invalid logout request")
                return
            self.write_empty(204)
            return
        if path == "/api/v1/open-platform/resources/access/check":
            if self.headers.get("Authorization") != "Bearer app-only-token":
                self.write_json(401, {
                    "success": False,
                    "error": {
                        "code": "A0010002",
                        "message": "open platform resource access token is invalid",
                    },
                })
                return
            try:
                payload = json.loads(self.rfile.read(int(self.headers.get("Content-Length", "0") or "0")).decode())
            except Exception:
                self.write_json(400, {
                    "success": False,
                    "error": {
                        "code": "A0000001",
                        "message": "invalid request body",
                    },
                })
                return
            if payload.get("clientID") or payload.get("clientSecret"):
                self.write_json(400, {
                    "success": False,
                    "error": {
                        "code": "A0000001",
                        "message": "resource access bearer and body client credentials are mutually exclusive",
                    },
                })
                return
            action = payload.get("action", "")
            relation = "can_write_by_app" if action == "write" else "can_read_by_app"
            allowed = payload.get("resourceID") == "granted-resource"
            self.write_json(200, {
                "success": True,
                "data": {
                    "allowed": allowed,
                    "appID": 1,
                    "clientID": "registered-client",
                    "resourceType": payload.get("resourceType", ""),
                    "resourceID": payload.get("resourceID", ""),
                    "action": action,
                    "relation": relation,
                    "reason": "allowed" if allowed else "fga_denied",
                },
            })
            return
        if path == "/id/oauth2/token":
            if parsed.query:
                self.write_json(400, {"error": "invalid_request"})
                return
            if not self.form_content_type_valid():
                self.write_json(400, {"error": "invalid_request"})
                return
            form = self.read_form()
            if not form.get("grant_type") or form.get("grant_type") == [""]:
                self.write_json(400, {"error": "invalid_request"})
                return
            if form.get("grant_type") == ["client_credentials"]:
                if not self.client_authenticated(form):
                    self.write_json(401, {"error": "invalid_client"})
                    return
                if form.get("scope") != ["resource.read"]:
                    self.write_json(400, {"error": "invalid_scope"})
                    return
                self.revoked_tokens.discard("app-only-token")
                self.write_json(200, {
                    "access_token": "app-only-token",
                    "token_type": "Bearer",
                    "expires_in": 900,
                    "scope": "resource.read",
                })
                return
            self.write_json(400, {"error": "unsupported_grant_type"})
            return
        if path == "/id/oauth2/introspect":
            if parsed.query:
                self.write_json(400, {"error": "invalid_request"})
                return
            if not self.form_content_type_valid():
                self.write_json(400, {"error": "invalid_request"})
                return
            form = self.read_form()
            if self.client_authenticated(form):
                token = form.get("token", [""])[0]
                if not token:
                    self.write_json(400, {"error": "invalid_request"})
                    return
                if token == "app-only-token" and token not in self.revoked_tokens:
                    self.write_json(200, {
                        "active": True,
                        "client_id": "registered-client",
                        "sub": "client:registered-client",
                        "scope": "resource.read",
                        "token_type": "Bearer",
                        "token_kind": "access_token",
                        "grant_type": "client_credentials",
                    })
                    return
                self.write_json(200, {"active": False})
                return
            self.write_json(401, {"error": "invalid_client"})
            return
        if path == "/id/oauth2/revoke":
            if parsed.query:
                self.write_json(400, {"error": "invalid_request"})
                return
            if not self.form_content_type_valid():
                self.write_json(400, {"error": "invalid_request"})
                return
            form = self.read_form()
            if self.client_authenticated(form):
                token = form.get("token", [""])[0]
                if not token:
                    self.write_json(400, {"error": "invalid_request"})
                    return
                self.revoked_tokens.add(token)
                self.write_empty(200)
                return
            self.write_json(401, {"error": "invalid_client"})
            return
        if path == "/id/oidc/userinfo":
            self.write_json(401, {"error": "invalid_token"})
            return
        self.write_json(404, {"error": "not_found", "path": self.path})

server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
port_file.write_text(str(server.server_port))
server.serve_forever()
PY

python3 "${tmpdir}/fake-identity-server.py" "${port_file}" &
server_pid=$!
for _ in {1..50}; do
  [[ -s "${port_file}" ]] && break
  sleep 0.1
done
[[ -s "${port_file}" ]] || fail "fake identity server did not start"

port="$(cat "${port_file}")"
base_url="http://127.0.0.1:${port}"
cat >"${env_file}" <<ENV
APP_ENV=production
WEB_PUBLIC_URL=${base_url}
IDENTITY_ISSUER=${base_url}/id
CASDOOR_ISSUER=${base_url}/sso
ENV

cat >"${bad_env_file}" <<'ENV'
APP_ENV=production
WEB_PUBLIC_URL=http://127.0.0.1:9
IDENTITY_ISSUER=http://127.0.0.1:9/id-from-env-file
CASDOOR_ISSUER=http://127.0.0.1:9/sso-from-env-file
ENV

local_refused_output="$(
  ENV_FILE="${env_file}" \
  GENERATED_ENV_FILE="${generated_env_file}" \
  GENERATED_SECRET_ENV_FILE="${generated_secret_env_file}" \
  GENERATED_OBS_DIR="${generated_obs_dir}" \
  IDENTITY_PUBLIC_SMOKE_EVIDENCE_FILE="${local_refused_evidence_file}" \
  IDENTITY_PUBLIC_SMOKE_RETRIES=1 \
  "${SMOKE_SCRIPT}" 2>&1
)" && fail "smoke unexpectedly allowed local targets without an explicit override"

printf '%s\n' "${local_refused_output}" | grep -q 'identity-public-smoke verifies public production ingress' || \
  fail "local target refusal did not explain the public-ingress guard"

output="$(
  ENV_FILE="${env_file}" \
  GENERATED_ENV_FILE="${generated_env_file}" \
  GENERATED_SECRET_ENV_FILE="${generated_secret_env_file}" \
  GENERATED_OBS_DIR="${generated_obs_dir}" \
  IDENTITY_PUBLIC_SMOKE_EVIDENCE_FILE="${identity_evidence_file}" \
  IDENTITY_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS=true \
  IDENTITY_PUBLIC_SMOKE_RETRIES=1 \
  "${SMOKE_SCRIPT}"
)"

printf '%s\n' "${output}" | grep -q 'public identity smoke passed' || fail "smoke did not pass against fake identity server"
assert_evidence "${identity_evidence_file}" default

registered_output="$(
  ENV_FILE="${env_file}" \
  GENERATED_ENV_FILE="${generated_env_file}" \
  GENERATED_SECRET_ENV_FILE="${generated_secret_env_file}" \
  GENERATED_OBS_DIR="${generated_obs_dir}" \
  IDENTITY_PUBLIC_SMOKE_EVIDENCE_FILE="${registered_evidence_file}" \
  IDENTITY_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS=true \
  IDENTITY_PUBLIC_SMOKE_CLIENT_ID="registered-client" \
  IDENTITY_PUBLIC_SMOKE_REDIRECT_URI="${base_url}/client/callback" \
  IDENTITY_PUBLIC_SMOKE_RETRIES=1 \
  "${SMOKE_SCRIPT}"
)"

printf '%s\n' "${registered_output}" | grep -q 'public identity smoke passed' || fail "registered-client smoke did not pass"
assert_evidence "${registered_evidence_file}" registered

client_credentials_output="$(
  ENV_FILE="${env_file}" \
  GENERATED_ENV_FILE="${generated_env_file}" \
  GENERATED_SECRET_ENV_FILE="${generated_secret_env_file}" \
  GENERATED_OBS_DIR="${generated_obs_dir}" \
  IDENTITY_PUBLIC_SMOKE_EVIDENCE_FILE="${client_credentials_evidence_file}" \
  IDENTITY_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS=true \
  IDENTITY_PUBLIC_SMOKE_CLIENT_ID="registered-client" \
  IDENTITY_PUBLIC_SMOKE_REDIRECT_URI="${base_url}/client/callback" \
  IDENTITY_PUBLIC_SMOKE_CLIENT_SECRET="registered-secret" \
  IDENTITY_PUBLIC_SMOKE_CLIENT_CREDENTIALS_SCOPE="resource.read" \
  IDENTITY_PUBLIC_SMOKE_RETRIES=1 \
  "${SMOKE_SCRIPT}"
)"

printf '%s\n' "${client_credentials_output}" | grep -q 'public identity smoke passed' || fail "client_credentials smoke did not pass"
assert_evidence "${client_credentials_evidence_file}" client_credentials

client_credentials_allowed_output="$(
  ENV_FILE="${env_file}" \
  GENERATED_ENV_FILE="${generated_env_file}" \
  GENERATED_SECRET_ENV_FILE="${generated_secret_env_file}" \
  GENERATED_OBS_DIR="${generated_obs_dir}" \
  IDENTITY_PUBLIC_SMOKE_EVIDENCE_FILE="${client_credentials_allowed_evidence_file}" \
  IDENTITY_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS=true \
  IDENTITY_PUBLIC_SMOKE_CLIENT_ID="registered-client" \
  IDENTITY_PUBLIC_SMOKE_REDIRECT_URI="${base_url}/client/callback" \
  IDENTITY_PUBLIC_SMOKE_CLIENT_SECRET="registered-secret" \
  IDENTITY_PUBLIC_SMOKE_CLIENT_CREDENTIALS_SCOPE="resource.read" \
  IDENTITY_PUBLIC_SMOKE_RESOURCE_ACCESS_RESOURCE_ID="granted-resource" \
  IDENTITY_PUBLIC_SMOKE_RESOURCE_ACCESS_EXPECT_ALLOWED=true \
  IDENTITY_PUBLIC_SMOKE_RETRIES=1 \
  "${SMOKE_SCRIPT}"
)"

printf '%s\n' "${client_credentials_allowed_output}" | grep -q 'public identity smoke passed' || fail "client_credentials allowed resource smoke did not pass"
assert_evidence "${client_credentials_allowed_evidence_file}" client_credentials_allowed

override_output="$(
  ENV_FILE="${bad_env_file}" \
  GENERATED_ENV_FILE="${generated_env_file}" \
  GENERATED_SECRET_ENV_FILE="${generated_secret_env_file}" \
  GENERATED_OBS_DIR="${generated_obs_dir}" \
  IDENTITY_PUBLIC_SMOKE_EVIDENCE_FILE="${override_evidence_file}" \
  IDENTITY_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS=true \
  WEB_PUBLIC_URL="${base_url}" \
  IDENTITY_ISSUER="${base_url}/id" \
  CASDOOR_ISSUER="${base_url}/sso" \
  IDENTITY_PUBLIC_SMOKE_RETRIES=1 \
  "${SMOKE_SCRIPT}"
)"

printf '%s\n' "${override_output}" | grep -q 'public identity smoke passed' || fail "inline smoke env did not override ENV_FILE values"
assert_evidence "${override_evidence_file}" override

failure_output="$(
  ENV_FILE="${env_file}" \
  GENERATED_ENV_FILE="${generated_env_file}" \
  GENERATED_SECRET_ENV_FILE="${generated_secret_env_file}" \
  GENERATED_OBS_DIR="${generated_obs_dir}" \
  IDENTITY_PUBLIC_SMOKE_EVIDENCE_FILE="${failure_evidence_file}" \
  IDENTITY_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS=true \
  WEB_PUBLIC_URL="${base_url}/missing-web" \
  IDENTITY_ISSUER="${base_url}/missing-id" \
  CASDOOR_ISSUER="${base_url}/missing-sso" \
  IDENTITY_PUBLIC_SMOKE_RETRIES=1 \
  IDENTITY_PUBLIC_SMOKE_SLEEP_SECONDS=0 \
  "${SMOKE_SCRIPT}" 2>&1
)" && fail "smoke unexpectedly passed with missing public identity routes"

printf '%s\n' "${failure_output}" | grep -q '25 失败' || fail "smoke failure count did not include JSON fetch failures"
assert_evidence "${failure_evidence_file}" failure

unreachable_output="$(
  ENV_FILE="${env_file}" \
  GENERATED_ENV_FILE="${generated_env_file}" \
  GENERATED_SECRET_ENV_FILE="${generated_secret_env_file}" \
  GENERATED_OBS_DIR="${generated_obs_dir}" \
  IDENTITY_PUBLIC_SMOKE_EVIDENCE_FILE="${unreachable_evidence_file}" \
  IDENTITY_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS=true \
  WEB_PUBLIC_URL="${base_url}" \
  IDENTITY_ISSUER="http://127.0.0.1:9/unreachable-id" \
  CASDOOR_ISSUER="${base_url}/sso" \
  IDENTITY_PUBLIC_SMOKE_RETRIES=1 \
  IDENTITY_PUBLIC_SMOKE_SLEEP_SECONDS=0 \
  "${SMOKE_SCRIPT}" 2>&1
)" && fail "smoke unexpectedly passed with unreachable identity issuer"

printf '%s\n' "${unreachable_output}" | grep -q '结果:' || fail "smoke did not print a result summary for connection failures"
printf '%s\n' "${unreachable_output}" | grep -q 'public identity smoke failed' || fail "smoke did not fail cleanly for connection failures"
assert_evidence "${unreachable_evidence_file}" unreachable

echo "[identity-public-smoke-contract] all assertions passed"
