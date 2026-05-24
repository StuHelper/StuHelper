#!/usr/bin/env bash
# Collect sanitized diagnostics for the public StuHelper identity ingress.
#
# This script is intentionally diagnostic rather than a deployment gate by
# default. Set PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_STRICT=true to fail the
# command when any public ingress check fails.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

usage() {
  cat <<'USAGE'
Usage: infra/ops/public-identity-ingress-diagnostic.sh

Collects DNS, SNI TLS, and public OIDC endpoint diagnostics for:

  - WEB_PUBLIC_URL /health/ready
  - IDENTITY_ISSUER /.well-known/openid-configuration
  - IDENTITY_ISSUER /.well-known/oauth-authorization-server
  - IDENTITY_ISSUER /.well-known/jwks.json
  - CASDOOR_ISSUER /.well-known/openid-configuration

Required env:
  none

Optional env:
  WEB_PUBLIC_URL                                  defaults to https://stuhelper.com
  IDENTITY_ISSUER                                 defaults to https://id.stuhelper.com
  CASDOOR_ISSUER                                  defaults to https://sso.stuhelper.com
  PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_TIMEOUT     defaults to PUBLIC_INGRESS_PREFLIGHT_TIMEOUT_SECONDS or 10
  PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_FILE        defaults to infra/generated/public-identity-ingress-diagnostic.json
                                                   set to "-" to only print the JSON bundle
  PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_STRICT      defaults to false; true exits non-zero when checks fail
  PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_PUBLIC_DNS_ENABLED
                                                   defaults to true; queries dns.google DoH for public A/AAAA evidence.
  PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_USE_ENV_TARGETS
                                                   defaults to false; true allows WEB_PUBLIC_URL /
                                                   IDENTITY_ISSUER / CASDOOR_ISSUER loaded from ENV_FILE.
                                                   Inline env always overrides.
USAGE
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
  exit 0
fi

require_cmd curl
require_cmd jq
require_cmd openssl
require_cmd python3

preserved_web_public_url="${WEB_PUBLIC_URL-__STUHELPER_UNSET__}"
preserved_identity_issuer="${IDENTITY_ISSUER-__STUHELPER_UNSET__}"
preserved_casdoor_issuer="${CASDOOR_ISSUER-__STUHELPER_UNSET__}"
preserved_timeout="${PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_TIMEOUT-__STUHELPER_UNSET__}"
preserved_file="${PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_FILE-__STUHELPER_UNSET__}"
preserved_strict="${PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_STRICT-__STUHELPER_UNSET__}"
preserved_public_dns_enabled="${PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_PUBLIC_DNS_ENABLED-__STUHELPER_UNSET__}"
preserved_use_env_targets="${PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_USE_ENV_TARGETS-__STUHELPER_UNSET__}"

load_env

if [[ "${preserved_timeout}" != "__STUHELPER_UNSET__" ]]; then PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_TIMEOUT="${preserved_timeout}"; fi
if [[ "${preserved_file}" != "__STUHELPER_UNSET__" ]]; then PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_FILE="${preserved_file}"; fi
if [[ "${preserved_strict}" != "__STUHELPER_UNSET__" ]]; then PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_STRICT="${preserved_strict}"; fi
if [[ "${preserved_public_dns_enabled}" != "__STUHELPER_UNSET__" ]]; then PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_PUBLIC_DNS_ENABLED="${preserved_public_dns_enabled}"; fi
if [[ "${preserved_use_env_targets}" != "__STUHELPER_UNSET__" ]]; then PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_USE_ENV_TARGETS="${preserved_use_env_targets}"; fi

use_env_targets="${PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_USE_ENV_TARGETS:-false}"
case "${use_env_targets}" in
  true | TRUE | 1 | yes | YES) use_env_targets="true" ;;
  false | FALSE | 0 | no | NO | "") use_env_targets="false" ;;
  *) die "PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_USE_ENV_TARGETS must be true or false" ;;
esac

if [[ "${preserved_web_public_url}" != "__STUHELPER_UNSET__" ]]; then
  WEB_PUBLIC_URL="${preserved_web_public_url}"
elif [[ "${use_env_targets}" != "true" ]]; then
  unset WEB_PUBLIC_URL
fi
if [[ "${preserved_identity_issuer}" != "__STUHELPER_UNSET__" ]]; then
  IDENTITY_ISSUER="${preserved_identity_issuer}"
elif [[ "${use_env_targets}" != "true" ]]; then
  unset IDENTITY_ISSUER
fi
if [[ "${preserved_casdoor_issuer}" != "__STUHELPER_UNSET__" ]]; then
  CASDOOR_ISSUER="${preserved_casdoor_issuer}"
elif [[ "${use_env_targets}" != "true" ]]; then
  unset CASDOOR_ISSUER
fi

web_public_url="$(trim_trailing_slash "${WEB_PUBLIC_URL:-https://stuhelper.com}")"
identity_issuer="$(trim_trailing_slash "${IDENTITY_ISSUER:-https://id.stuhelper.com}")"
casdoor_issuer="$(trim_trailing_slash "${CASDOOR_ISSUER:-https://sso.stuhelper.com}")"
timeout_seconds="${PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_TIMEOUT:-${PUBLIC_INGRESS_PREFLIGHT_TIMEOUT_SECONDS:-10}}"
evidence_file="${PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_FILE:-${REPO_ROOT}/infra/generated/public-identity-ingress-diagnostic.json}"
strict="${PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_STRICT:-false}"
public_dns_enabled="${PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_PUBLIC_DNS_ENABLED:-true}"

case "${strict}" in
  true | TRUE | 1 | yes | YES) strict="true" ;;
  false | FALSE | 0 | no | NO | "") strict="false" ;;
  *) die "PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_STRICT must be true or false" ;;
esac

case "${public_dns_enabled}" in
  true | TRUE | 1 | yes | YES) public_dns_enabled="true" ;;
  false | FALSE | 0 | no | NO | "") public_dns_enabled="false" ;;
  *) die "PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_PUBLIC_DNS_ENABLED must be true or false" ;;
esac

bundle="$(
  python3 - \
    "${web_public_url}" \
    "${identity_issuer}" \
    "${casdoor_issuer}" \
    "${timeout_seconds}" \
    "${public_dns_enabled}" <<'PY'
from __future__ import annotations

from datetime import datetime, timezone
import ipaddress
import json
import os
import socket
import subprocess
import sys
import tempfile
from pathlib import Path
from urllib.parse import quote, urlparse


web_public_url, identity_issuer, casdoor_issuer, timeout_raw, public_dns_enabled_raw = sys.argv[1:6]
public_dns_enabled = public_dns_enabled_raw == "true"

try:
    timeout = max(1.0, float(timeout_raw))
except ValueError:
    timeout = 10.0


def trim(value: str) -> str:
    return value.rstrip("/")


def snippet(value: str, limit: int = 320) -> str:
    collapsed = " ".join(value.replace("\r", " ").replace("\n", " ").split())
    if len(collapsed) <= limit:
        return collapsed
    return collapsed[: limit - 3] + "..."


def sanitize_headers(value: str) -> str:
    sensitive = {
        "authorization",
        "cookie",
        "proxy-authorization",
        "set-cookie",
        "x-csrf-token",
    }
    safe_lines: list[str] = []
    for raw_line in value.splitlines():
        line = raw_line.strip()
        if not line:
            continue
        name = line.split(":", 1)[0].strip().lower()
        if name in sensitive:
            safe_lines.append(f"{name}: <redacted>")
        else:
            safe_lines.append(line)
    return "\n".join(safe_lines)


def parse_endpoint(base_url: str) -> dict[str, object]:
    parsed = urlparse(base_url)
    scheme = parsed.scheme or "https"
    host = parsed.hostname or ""
    if not host:
        return {
            "url": base_url,
            "scheme": scheme,
            "host": "",
            "port": None,
            "valid": False,
            "error": "URL host is empty",
        }
    port = parsed.port or (443 if scheme == "https" else 80)
    return {
        "url": base_url,
        "scheme": scheme,
        "host": host,
        "port": port,
        "valid": True,
    }


def endpoint_url(base_url: str, path: str) -> str:
    return trim(base_url) + path


def dns_probe(host: str, port: int) -> dict[str, object]:
    if not host:
        return {
            "passed": False,
            "diagnosis": "dns_resolution_failed",
            "recommendation": "Set a non-empty public hostname for this ingress target.",
            "error": "host is empty",
            "addresses": [],
        }
    try:
        infos = socket.getaddrinfo(host, port, type=socket.SOCK_STREAM)
        addresses = sorted({info[4][0] for info in infos})
        result: dict[str, object] = {"passed": bool(addresses), "addresses": addresses}
        if not addresses:
            result["diagnosis"] = "dns_resolution_failed"
            result["recommendation"] = "Create public A/AAAA records for this hostname and wait for DNS propagation."
            return result
        if not is_localhost(host):
            non_public = [address for address in addresses if not is_global_ip(address)]
            if non_public:
                result["passed"] = False
                result["diagnosis"] = "dns_non_public_address"
                result["nonPublicAddresses"] = non_public
                result["recommendation"] = "Public ingress hostnames must resolve to public A/AAAA records; check authoritative DNS, split-horizon DNS, and local fake-IP/proxy DNS."
        return result
    except Exception as exc:  # noqa: BLE001 - diagnostic command must report, not crash.
        return {
            "passed": False,
            "diagnosis": "dns_resolution_failed",
            "recommendation": "Create public A/AAAA records for this hostname and verify the resolver can reach authoritative DNS.",
            "addresses": [],
            "error": snippet(str(exc)),
        }


def is_localhost(host: str) -> bool:
    normalized = host.strip().lower().rstrip(".")
    return normalized in {"localhost", "127.0.0.1", "::1"}


def is_ip_literal(host: str) -> bool:
    try:
        ipaddress.ip_address(host.strip().strip("[]").split("%", 1)[0])
        return True
    except ValueError:
        return False


def is_global_ip(address: str) -> bool:
    try:
        return ipaddress.ip_address(address.split("%", 1)[0]).is_global
    except ValueError:
        return False


def public_dns_probe(host: str) -> dict[str, object]:
    if not public_dns_enabled:
        return {"passed": True, "skipped": True, "reason": "public DNS probe disabled"}
    if not host:
        return {
            "passed": False,
            "provider": "dns.google",
            "diagnosis": "public_dns_resolution_failed",
            "recommendation": "Set a non-empty public hostname before checking public DNS.",
            "addresses": [],
        }
    if is_localhost(host) or is_ip_literal(host):
        return {
            "passed": True,
            "provider": "dns.google",
            "skipped": True,
            "reason": "public DNS probe only applies to hostname targets",
        }

    statuses: dict[str, object] = {}
    addresses: list[str] = []
    cname_records: list[str] = []
    for rrtype, rrnumber in (("A", 1), ("AAAA", 28)):
        url = f"https://dns.google/resolve?name={quote(host.rstrip('.'))}&type={rrtype}"
        proc = subprocess.run(
            ["curl", "-fsS", "--max-time", str(timeout), url],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            check=False,
        )
        if proc.returncode != 0:
            return {
                "passed": False,
                "provider": "dns.google",
                "diagnosis": "public_dns_lookup_failed",
                "recommendation": "Check outbound HTTPS/DNS-over-HTTPS reachability from this runner, or disable PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_PUBLIC_DNS_ENABLED for air-gapped diagnostics.",
                "returnCode": proc.returncode,
                "error": snippet(proc.stderr.decode("utf-8", "replace")),
                "addresses": addresses,
            }
        try:
            payload = json.loads(proc.stdout.decode("utf-8", "replace"))
        except json.JSONDecodeError:
            return {
                "passed": False,
                "provider": "dns.google",
                "diagnosis": "public_dns_invalid_response",
                "recommendation": "Public DNS-over-HTTPS response must be valid JSON.",
                "bodySnippet": snippet(proc.stdout.decode("utf-8", "replace")),
                "addresses": addresses,
            }
        statuses[rrtype] = payload.get("Status")
        for answer in payload.get("Answer") or []:
            if not isinstance(answer, dict):
                continue
            data = answer.get("data")
            answer_type = answer.get("type")
            if not isinstance(data, str):
                continue
            if answer_type == rrnumber:
                addresses.append(data)
            elif answer_type == 5:
                cname_records.append(data)

    addresses = sorted(set(addresses))
    cname_records = sorted(set(cname_records))
    result: dict[str, object] = {
        "passed": bool(addresses),
        "provider": "dns.google",
        "statuses": statuses,
        "addresses": addresses,
    }
    if cname_records:
        result["cnameRecords"] = cname_records
    if not addresses:
        if any(status == 3 for status in statuses.values()):
            result["diagnosis"] = "public_dns_nxdomain"
            result["recommendation"] = "Create public DNS records for this hostname; public resolvers currently return NXDOMAIN."
        else:
            result["diagnosis"] = "public_dns_resolution_failed"
            result["recommendation"] = "Create public A/AAAA records for this hostname and wait for public DNS propagation."
        return result
    non_public = [address for address in addresses if not is_global_ip(address)]
    if non_public:
        result["passed"] = False
        result["diagnosis"] = "public_dns_non_public_address"
        result["nonPublicAddresses"] = non_public
        result["recommendation"] = "Public DNS A/AAAA records must point to globally routable addresses, not private, loopback, reserved, or fake-IP ranges."
    return result


def tls_probe(host: str, port: int, scheme: str) -> dict[str, object]:
    if scheme != "https":
        return {"passed": True, "skipped": True, "reason": f"scheme is {scheme}"}
    if not host:
        return {"passed": False, "error": "host is empty"}
    cmd = [
        "openssl",
        "s_client",
        "-connect",
        f"{host}:{port}",
        "-servername",
        host,
        "-verify_hostname",
        host,
        "-verify_return_error",
        "-brief",
    ]
    try:
        proc = subprocess.run(
            cmd,
            input=b"",
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            timeout=timeout,
            check=False,
        )
    except subprocess.TimeoutExpired:
        return {
            "passed": False,
            "diagnosis": "tls_timeout",
            "error": f"TLS handshake timed out after {timeout:g}s",
            "command": "openssl s_client -servername <host> -connect <host>:<port>",
        }
    output = (proc.stdout + proc.stderr).decode("utf-8", "replace")
    result: dict[str, object] = {
        "passed": proc.returncode == 0,
        "returnCode": proc.returncode,
        "command": "openssl s_client -servername <host> -connect <host>:<port>",
        "outputSnippet": snippet(output, 640),
    }
    if proc.returncode != 0:
        lowered = output.lower()
        if "ssl_error_syscall" in lowered or "unexpected eof" in lowered:
            result["diagnosis"] = "tls_handshake_failed"
            result["recommendation"] = "Check DNS A/AAAA, firewall/CDN, Nginx 443 ssl listener, SNI server_name, and certificate binding."
        elif "hostname mismatch" in lowered or "verify error" in lowered:
            result["diagnosis"] = "tls_certificate_invalid"
            result["recommendation"] = "Check the certificate SAN/CN and trust chain for this exact hostname."
        else:
            result["diagnosis"] = "tls_probe_failed"
            result["recommendation"] = "Run openssl s_client with the same SNI on the target host and inspect Nginx error logs."
    return result


def run_curl(url: str) -> dict[str, object]:
    marker = "__STUHELPER_CURL_META__:"
    with tempfile.TemporaryDirectory() as tmp:
        tmpdir = Path(tmp)
        body_path = tmpdir / "body"
        header_path = tmpdir / "headers"
        write_out = (
            "\n"
            + marker
            + "%{http_code}\t%{content_type}\t%{ssl_verify_result}\t"
            + "%{remote_ip}\t%{http_version}\t%{time_connect}\t%{time_appconnect}"
        )
        cmd = [
            "curl",
            "-sS",
            "--max-time",
            str(timeout),
            "-D",
            str(header_path),
            "-o",
            str(body_path),
            "-w",
            write_out,
            url,
        ]
        proc = subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=False)
        stdout = proc.stdout.decode("utf-8", "replace")
        stderr = proc.stderr.decode("utf-8", "replace")
        meta_line = ""
        for line in stdout.splitlines():
            if line.startswith(marker):
                meta_line = line[len(marker) :]
        parts = meta_line.split("\t") if meta_line else []
        body_text = body_path.read_text(encoding="utf-8", errors="replace") if body_path.exists() else ""
        headers_text = header_path.read_text(encoding="utf-8", errors="replace") if header_path.exists() else ""
    status = parts[0] if len(parts) >= 1 else "000"
    result: dict[str, object] = {
        "url": url,
        "passed": proc.returncode == 0 and status.isdigit() and 200 <= int(status) < 300,
        "curlReturnCode": proc.returncode,
        "httpStatus": status,
        "contentType": parts[1] if len(parts) >= 2 else "",
        "sslVerifyResult": parts[2] if len(parts) >= 3 else "",
        "remoteIP": parts[3] if len(parts) >= 4 else "",
        "httpVersion": parts[4] if len(parts) >= 5 else "",
        "timeConnect": parts[5] if len(parts) >= 6 else "",
        "timeAppConnect": parts[6] if len(parts) >= 7 else "",
        "bodySnippet": snippet(body_text),
        "headersSnippet": snippet(sanitize_headers(headers_text)),
    }
    if stderr.strip():
        result["curlError"] = snippet(stderr)
    if proc.returncode != 0 or status == "000":
        lowered = stderr.lower()
        if "ssl_error_syscall" in lowered or "ssl connect" in lowered:
            result["diagnosis"] = "tls_handshake_failed"
            result["recommendation"] = "Check public 443 reachability, SNI server block, TLS certificate binding, firewall/CDN, and Nginx error logs."
        else:
            result["diagnosis"] = "http_request_failed"
            result["recommendation"] = "Check network reachability and upstream public ingress logs."
    return result


def oidc_discovery_probe(label: str, issuer: str) -> dict[str, object]:
    url = endpoint_url(issuer, "/.well-known/openid-configuration")
    result = run_curl(url)
    if not (str(result.get("httpStatus", "")).isdigit() and 200 <= int(str(result["httpStatus"])) < 300):
        body = str(result.get("bodySnippet", "")).lower()
        if "casdoor" in body and "<html" in body:
            result["diagnosis"] = "casdoor_well_known_served_by_spa"
            result["recommendation"] = "On the SSO host, proxy location ^~ /.well-known/ to Casdoor before any static root/try_files rule."
        elif str(result.get("httpStatus")) == "404" and label == "Identity":
            result["diagnosis"] = "identity_well_known_not_proxied"
            result["recommendation"] = "On the main host, proxy id.stuhelper.com location ^~ /.well-known/ to the backend."
        return result
    try:
        metadata = json.loads(str(result.get("bodySnippet", "")))
    except json.JSONDecodeError:
        # bodySnippet may be truncated. Re-run from the saved curl body is not
        # available here, so do a small direct fetch without headers for metadata.
        direct = subprocess.run(
            ["curl", "-sS", "--max-time", str(timeout), url],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            check=False,
        )
        try:
            metadata = json.loads(direct.stdout.decode("utf-8", "replace"))
        except json.JSONDecodeError:
            result["passed"] = False
            result["diagnosis"] = "oidc_discovery_invalid_json"
            result["recommendation"] = "Discovery must return a JSON object, not an SPA/HTML/static fallback."
            return result
    expected = trim(issuer)
    if metadata.get("issuer") != expected:
        result["passed"] = False
        result["diagnosis"] = "oidc_issuer_mismatch"
        result["expectedIssuer"] = expected
        result["actualIssuer"] = metadata.get("issuer")
        result["recommendation"] = "Fix issuer/base URL config so discovery issuer exactly matches the public issuer."
        return result
    required = ["authorization_endpoint", "token_endpoint", "jwks_uri"]
    missing = [key for key in required if not isinstance(metadata.get(key), str) or not metadata.get(key)]
    if missing:
        result["passed"] = False
        result["diagnosis"] = "oidc_discovery_missing_fields"
        result["missingFields"] = missing
        result["recommendation"] = "Ensure the upstream OIDC service is serving complete discovery metadata."
    else:
        result["passed"] = True
        result["issuer"] = metadata.get("issuer")
    return result


def oauth_authorization_server_metadata_probe(issuer: str) -> dict[str, object]:
    url = endpoint_url(issuer, "/.well-known/oauth-authorization-server")
    result = run_curl(url)
    if not (str(result.get("httpStatus", "")).isdigit() and 200 <= int(str(result["httpStatus"])) < 300):
        if str(result.get("httpStatus")) == "404":
            result["diagnosis"] = "identity_oauth_as_metadata_not_proxied"
            result["recommendation"] = "On the main host, proxy id.stuhelper.com location ^~ /.well-known/ to the backend, including /.well-known/oauth-authorization-server."
        return result
    try:
        metadata = json.loads(str(result.get("bodySnippet", "")))
    except json.JSONDecodeError:
        direct = subprocess.run(
            ["curl", "-sS", "--max-time", str(timeout), url],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            check=False,
        )
        try:
            metadata = json.loads(direct.stdout.decode("utf-8", "replace"))
        except json.JSONDecodeError:
            result["passed"] = False
            result["diagnosis"] = "oauth_as_metadata_invalid_json"
            result["recommendation"] = "OAuth authorization server metadata must return a JSON object, not an SPA/HTML/static fallback."
            return result
    expected = trim(issuer)
    if metadata.get("issuer") != expected:
        result["passed"] = False
        result["diagnosis"] = "oauth_as_issuer_mismatch"
        result["expectedIssuer"] = expected
        result["actualIssuer"] = metadata.get("issuer")
        result["recommendation"] = "Fix issuer/base URL config so OAuth AS metadata issuer exactly matches the public issuer."
        return result
    required = [
        "authorization_endpoint",
        "token_endpoint",
        "jwks_uri",
        "revocation_endpoint",
        "introspection_endpoint",
    ]
    missing = [key for key in required if not isinstance(metadata.get(key), str) or not metadata.get(key)]
    if missing:
        result["passed"] = False
        result["diagnosis"] = "oauth_as_metadata_missing_fields"
        result["missingFields"] = missing
        result["recommendation"] = "Ensure StuHelper Identity serves complete RFC 8414 metadata for OAuth2 gateways and resource servers."
    else:
        result["passed"] = True
        result["issuer"] = metadata.get("issuer")
    return result


def jwks_probe(issuer: str) -> dict[str, object]:
    result = run_curl(endpoint_url(issuer, "/.well-known/jwks.json"))
    if not result.get("passed"):
        if str(result.get("httpStatus")) == "404":
            result["diagnosis"] = "identity_jwks_not_proxied"
            result["recommendation"] = "Proxy id.stuhelper.com location ^~ /.well-known/ to the backend."
        return result
    direct = subprocess.run(
        ["curl", "-sS", "--max-time", str(timeout), endpoint_url(issuer, "/.well-known/jwks.json")],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False,
    )
    try:
        jwks = json.loads(direct.stdout.decode("utf-8", "replace"))
    except json.JSONDecodeError:
        result["passed"] = False
        result["diagnosis"] = "jwks_invalid_json"
        result["recommendation"] = "JWKS must return a JSON object with a keys array."
        return result
    if not isinstance(jwks.get("keys"), list):
        result["passed"] = False
        result["diagnosis"] = "jwks_missing_keys"
        result["recommendation"] = "Identity JWKS must contain a keys array."
    return result


targets = {
    "web": parse_endpoint(web_public_url),
    "identity": parse_endpoint(identity_issuer),
    "casdoor": parse_endpoint(casdoor_issuer),
}

hosts: dict[str, object] = {}
for name, target in targets.items():
    host = str(target.get("host", ""))
    port = int(target["port"]) if target.get("port") is not None else 443
    scheme = str(target.get("scheme", "https"))
    hosts[name] = {
        "url": target.get("url"),
        "host": host,
        "port": port,
        "dns": dns_probe(host, port),
        "publicDNS": public_dns_probe(host),
        "tls": tls_probe(host, port, scheme),
    }

endpoints = {
    "webHealth": run_curl(endpoint_url(web_public_url, "/health/ready")),
    "identityDiscovery": oidc_discovery_probe("Identity", identity_issuer),
    "identityAuthorizationServerMetadata": oauth_authorization_server_metadata_probe(identity_issuer),
    "identityJWKS": jwks_probe(identity_issuer),
    "casdoorDiscovery": oidc_discovery_probe("Casdoor", casdoor_issuer),
}

diagnoses: list[dict[str, str]] = []
failed = 0
passed = 0
for group_name, group in (("hosts", hosts), ("endpoints", endpoints)):
    for name, item in group.items():
        checks: list[object]
        if group_name == "hosts":
            checks = [item.get("dns", {}), item.get("publicDNS", {}), item.get("tls", {})]  # type: ignore[union-attr]
        else:
            checks = [item]
        item_passed = all(bool(check.get("passed")) for check in checks if isinstance(check, dict))
        if item_passed:
            passed += 1
        else:
            failed += 1
        for check in checks:
            if not isinstance(check, dict):
                continue
            if check.get("passed"):
                continue
            diagnosis = check.get("diagnosis")
            if diagnosis:
                diagnoses.append(
                    {
                        "target": f"{group_name}.{name}",
                        "diagnosis": str(diagnosis),
                        "recommendation": str(check.get("recommendation", "")),
                    }
                )

bundle = {
    "generatedAt": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
    "passed": failed == 0,
    "summary": {
        "passed": passed,
        "failed": failed,
        "diagnoses": len(diagnoses),
    },
    "webPublicURL": web_public_url,
    "identityIssuer": identity_issuer,
    "casdoorIssuer": casdoor_issuer,
    "hosts": hosts,
    "endpoints": endpoints,
    "diagnoses": diagnoses,
}

print(json.dumps(bundle, ensure_ascii=True, indent=2, sort_keys=True))
PY
)"

if [[ "${evidence_file}" != "-" ]]; then
  mkdir -p "$(dirname "${evidence_file}")"
  tmp_file="$(mktemp)"
  trap 'rm -f "${tmp_file}"' EXIT
  printf '%s\n' "${bundle}" >"${tmp_file}"
  install -m 600 "${tmp_file}" "${evidence_file}"
  log "wrote public identity ingress diagnostic to ${evidence_file}" >&2
fi

printf '%s\n' "${bundle}" | jq .

if [[ "${strict}" == "true" ]] && ! printf '%s\n' "${bundle}" | jq -e '.passed == true' >/dev/null; then
  die "public identity ingress diagnostic found failing checks"
fi
