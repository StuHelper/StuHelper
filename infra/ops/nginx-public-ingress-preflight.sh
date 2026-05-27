#!/usr/bin/env bash
# Validate the effective Baota/Nginx public ingress config before deployment.
#
# By default this audits the StuHelper app host (`stuhelper.com`, `www`, and
# `id`). Run with NGINX_PUBLIC_INGRESS_PROFILE=sso on the external Casdoor host,
# or NGINX_PUBLIC_INGRESS_PROFILE=all against a combined nginx -T dump.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

if [[ "${PUBLIC_INGRESS_CONFIG_PREFLIGHT_ENABLED:-true}" != "true" ]]; then
  warn "public Nginx ingress config preflight skipped because PUBLIC_INGRESS_CONFIG_PREFLIGHT_ENABLED is not true"
  exit 0
fi

require_cmd python3

config_file="${NGINX_PUBLIC_INGRESS_CONFIG_FILE:-}"
tmp_config_file=""
tmp_error_file=""
source_label=""

cleanup() {
  if [[ -n "${tmp_config_file}" ]]; then
    rm -f "${tmp_config_file}"
  fi
  if [[ -n "${tmp_error_file}" ]]; then
    rm -f "${tmp_error_file}"
  fi
}
trap cleanup EXIT

if [[ -n "${config_file}" ]]; then
  [[ -f "${config_file}" ]] || die "NGINX_PUBLIC_INGRESS_CONFIG_FILE does not exist: ${config_file}"
  source_label="${config_file}"
else
  nginx_bin="${NGINX_PUBLIC_INGRESS_NGINX_BIN:-nginx}"
  require_cmd "${nginx_bin}"
  tmp_config_file="$(mktemp)"
  tmp_error_file="$(mktemp)"
  if ! "${nginx_bin}" -T >"${tmp_config_file}" 2>"${tmp_error_file}"; then
    die "nginx public ingress config preflight failed because nginx -T failed: $(_public_ingress_body_snippet "${tmp_error_file}")"
  fi
  config_file="${tmp_config_file}"
  source_label="$("${nginx_bin}" -v 2>&1 | sed 's/^nginx version: //')"
fi

python3 - "${config_file}" <<'PY'
from __future__ import annotations

from dataclasses import dataclass
import os
import re
import sys
from pathlib import Path


class CheckError(Exception):
    pass


@dataclass
class Node:
    name: str
    args: list[str]
    children: list["Node"]


def strip_comments(text: str) -> str:
    out: list[str] = []
    quote = ""
    escaped = False
    i = 0
    while i < len(text):
        ch = text[i]
        if quote:
            out.append(ch)
            if escaped:
                escaped = False
            elif ch == "\\":
                escaped = True
            elif ch == quote:
                quote = ""
            i += 1
            continue
        if ch in {"'", '"'}:
            quote = ch
            out.append(ch)
            i += 1
            continue
        if ch == "#":
            while i < len(text) and text[i] not in "\r\n":
                i += 1
            continue
        out.append(ch)
        i += 1
    return "".join(out)


def tokenize(text: str) -> list[str]:
    tokens: list[str] = []
    current: list[str] = []
    quote = ""
    escaped = False

    def flush() -> None:
        if current:
            tokens.append("".join(current))
            current.clear()

    for ch in strip_comments(text):
        if quote:
            if escaped:
                current.append(ch)
                escaped = False
            elif ch == "\\":
                escaped = True
            elif ch == quote:
                quote = ""
            else:
                current.append(ch)
            continue
        if ch in {"'", '"'}:
            quote = ch
        elif ch.isspace():
            flush()
        elif ch in "{};":
            flush()
            tokens.append(ch)
        else:
            current.append(ch)

    if quote:
        raise CheckError("unterminated quote in Nginx config")
    flush()
    return tokens


def skip_block(tokens: list[str], index: int) -> int:
    depth = 1
    while index < len(tokens):
        token = tokens[index]
        if token == "{":
            depth += 1
        elif token == "}":
            depth -= 1
            if depth == 0:
                return index + 1
        index += 1
    raise CheckError("unterminated block in Nginx config")


def parse_nodes(tokens: list[str], index: int = 0, nested: bool = False, mode: str = "scan") -> tuple[list[Node], int]:
    children: list[Node] = []
    while index < len(tokens):
        token = tokens[index]
        if token == "}":
            if not nested:
                raise CheckError("unexpected closing brace in Nginx config")
            return children, index + 1
        if token in {"{", ";"}:
            if mode == "scan":
                index += 1
                continue
            raise CheckError(f"unexpected token in Nginx config: {token}")

        name = token
        index += 1
        args: list[str] = []
        while index < len(tokens) and tokens[index] not in {"{", "}", ";"}:
            args.append(tokens[index])
            index += 1
        if index >= len(tokens):
            raise CheckError(f"directive {name} is missing ';' or block")
        terminator = tokens[index]
        if terminator == ";":
            if mode in {"server", "location"}:
                children.append(Node(name, args, []))
            index += 1
        elif terminator == "{":
            if name in {"http", "stream"} and mode == "scan":
                block_children, index = parse_nodes(tokens, index + 1, nested=True, mode="scan")
                children.extend(block_children)
            elif name == "server" and mode == "scan":
                block_children, index = parse_nodes(tokens, index + 1, nested=True, mode="server")
                children.append(Node(name, args, block_children))
            elif name == "location" and mode == "server":
                block_children, index = parse_nodes(tokens, index + 1, nested=True, mode="location")
                children.append(Node(name, args, block_children))
            else:
                index = skip_block(tokens, index + 1)
        else:
            if mode == "scan":
                index += 1
                continue
            raise CheckError(f"directive {name} ended with unexpected '}}'")

    if nested:
        raise CheckError("unterminated block in Nginx config")
    return children, index


def walk(nodes: list[Node]):
    for node in nodes:
        yield node
        yield from walk(node.children)


def direct(block: Node, name: str) -> list[Node]:
    return [child for child in block.children if child.name == name]


def recursive_has(block: Node, names: set[str]) -> bool:
    return any(node.name in names for node in walk(block.children))


def server_names(block: Node) -> list[str]:
    names: list[str] = []
    for directive in direct(block, "server_name"):
        names.extend(directive.args)
    return names


def has_https_listen(block: Node) -> bool:
    for directive in direct(block, "listen"):
        has_443 = any(token == "443" or token.endswith(":443") or ":443" in token for token in directive.args)
        has_ssl = any(token == "ssl" for token in directive.args)
        if has_443 and has_ssl:
            return True
    return False


def proxy_pass_values(block: Node) -> list[str]:
    values: list[str] = []
    for directive in direct(block, "proxy_pass"):
        if directive.args:
            values.append(directive.args[0])
    return values


def return_values(block: Node) -> list[tuple[str, str]]:
    values: list[tuple[str, str]] = []
    for directive in direct(block, "return"):
        if len(directive.args) >= 2:
            values.append((directive.args[0], directive.args[1]))
    return values


def add_header_values(block: Node) -> list[tuple[str, str, bool]]:
    values: list[tuple[str, str, bool]] = []
    for directive in direct(block, "add_header"):
        if len(directive.args) >= 2:
            header_name = directive.args[0]
            header_value = directive.args[1]
            always = any(arg.lower() == "always" for arg in directive.args[2:])
            values.append((header_name, header_value, always))
    return values


def location(block: Node, modifier: str | None, path: str) -> Node | None:
    for child in direct(block, "location"):
        if modifier is None and child.args == [path]:
            return child
        if modifier is not None and len(child.args) >= 2 and child.args[0] == modifier and child.args[1] == path:
            return child
    return None


def require_location_proxy(block: Node, label: str, modifier: str | None, path: str, upstream: str) -> None:
    loc = location(block, modifier, path)
    rendered = f"location {modifier + ' ' if modifier else ''}{path}"
    if loc is None:
        raise CheckError(f"{label}: missing {rendered}")
    values = proxy_pass_values(loc)
    if upstream not in values:
        raise CheckError(
            f"{label}: {rendered} must proxy_pass {upstream}; found {values or '<none>'}"
        )


def location_proxy_matches(block: Node, modifier: str | None, path: str, upstream: str) -> bool:
    loc = location(block, modifier, path)
    return loc is not None and upstream in proxy_pass_values(loc)


def require_location_return(block: Node, label: str, modifier: str | None, path: str, code: str, target: str) -> None:
    loc = location(block, modifier, path)
    rendered = f"location {modifier + ' ' if modifier else ''}{path}"
    if loc is None:
        raise CheckError(f"{label}: missing {rendered}")
    values = return_values(loc)
    if (code, target) not in values:
        raise CheckError(
            f"{label}: {rendered} must return {code} {target}; found {values or '<none>'}"
        )


def require_location_add_header(
    block: Node,
    label: str,
    modifier: str | None,
    path: str,
    header: str,
    value: str,
    require_always: bool = True,
) -> None:
    loc = location(block, modifier, path)
    rendered = f"location {modifier + ' ' if modifier else ''}{path}"
    if loc is None:
        raise CheckError(f"{label}: missing {rendered}")
    values = add_header_values(loc)
    for found_header, found_value, found_always in values:
        if (
            found_header.lower() == header.lower()
            and found_value == value
            and (not require_always or found_always)
        ):
            return
    suffix = " always" if require_always else ""
    raise CheckError(
        f"{label}: {rendered} must add_header {header} {value}{suffix}; found {values or '<none>'}"
    )


def has_proxy_header(block: Node, header: str, values: set[str]) -> bool:
    for directive in direct(block, "proxy_set_header"):
        if len(directive.args) >= 2 and directive.args[0].lower() == header.lower() and directive.args[1] in values:
            return True
    return False


def require_proxy_header(block: Node, label: str, header: str, value: str) -> None:
    if has_proxy_header(block, header, {value}):
        return
    raise CheckError(f"{label}: missing proxy_set_header {header} {value}")


def require_proxy_header_on_server_or_location(
    server: Node,
    loc: Node,
    label: str,
    rendered_location: str,
    header: str,
    values: set[str],
) -> None:
    if has_proxy_header(server, header, values) or has_proxy_header(loc, header, values):
        return
    rendered_values = ", ".join(sorted(values))
    raise CheckError(
        f"{label}: {rendered_location} or server block must set proxy_set_header {header} to one of {rendered_values}"
    )


def require_tls(block: Node, label: str) -> None:
    if not has_https_listen(block):
        raise CheckError(f"{label}: server block must listen on 443 ssl")
    if not direct(block, "ssl_certificate"):
        raise CheckError(f"{label}: missing ssl_certificate")
    if not direct(block, "ssl_certificate_key"):
        raise CheckError(f"{label}: missing ssl_certificate_key")


def require_common_proxy_server(block: Node, label: str) -> None:
    require_tls(block, label)
    require_proxy_header(block, label, "Host", "$host")
    require_proxy_header(block, label, "X-Forwarded-Proto", "https")
    require_proxy_header(block, label, "X-Forwarded-Host", "$host")


def validate_main(block: Node, upstreams: dict[str, str]) -> None:
    label = "stuhelper.com"
    require_common_proxy_server(block, label)
    for path in [
        "/login",
        "/auth/callback",
        "/consent",
        "/complete-profile",
        "/user/authorized-apps",
        "/user/identity-verification",
        "/user/student-verification",
        "/user/phone-binding",
        "/user/qq-binding",
        "/user/academic-info",
    ]:
        require_location_return(block, label, "=", path, "302", "https://id.stuhelper.com$request_uri")
    require_location_return(block, label, "^~", "/developers/", "302", "https://id.stuhelper.com$request_uri")
    require_location_proxy(block, label, "^~", "/api/", upstreams["backend"])
    require_location_proxy(block, label, "^~", "/health/", upstreams["backend"])
    require_location_proxy(block, label, "^~", "/admin/", upstreams["admin"])
    require_location_proxy(block, label, None, "/", upstreams["web"])


def has_static_well_known_location(block: Node, upstream: str) -> bool:
    for loc in direct(block, "location"):
        if not loc.args:
            continue
        path = loc.args[-1]
        if path not in {"/.well-known", "/.well-known/"}:
            continue
        if upstream not in proxy_pass_values(loc):
            return True
    return False


def validate_www(block: Node) -> None:
    require_tls(block, "www.stuhelper.com")


def validate_identity(block: Node, upstreams: dict[str, str]) -> None:
    label = "id.stuhelper.com"
    require_common_proxy_server(block, label)
    require_location_proxy(block, label, "^~", "/.well-known/", upstreams["backend"])
    require_location_proxy(block, label, "^~", "/oauth2/", upstreams["backend"])
    require_location_proxy(block, label, "^~", "/oidc/", upstreams["backend"])
    require_location_proxy(block, label, "^~", "/api/v1/", upstreams["backend"])
    require_location_proxy(block, label, "^~", "/api/", upstreams["casdoor"])
    require_location_return(block, label, "=", "/", "302", "/developers/apps")
    require_location_add_header(block, label, "=", "/", "Cache-Control", "no-store, no-cache, must-revalidate, private")
    require_location_add_header(block, label, "=", "/", "Pragma", "no-cache")
    require_location_add_header(block, label, "=", "/", "Expires", "0")
    require_location_proxy(block, label, "^~", "/login/oauth/", upstreams["casdoor"])
    require_location_proxy(block, label, "^~", "/signup/oauth/", upstreams["casdoor"])
    require_location_proxy(block, label, "=", "/login", upstreams["web"])
    require_location_proxy(block, label, "=", "/auth/callback", upstreams["web"])
    require_location_proxy(block, label, "=", "/consent", upstreams["web"])
    require_location_proxy(block, label, "=", "/complete-profile", upstreams["web"])
    require_location_proxy(block, label, "^~", "/developers/", upstreams["web"])
    for path in [
        "/user/authorized-apps",
        "/user/identity-verification",
        "/user/student-verification",
        "/user/phone-binding",
        "/user/qq-binding",
        "/user/academic-info",
    ]:
        require_location_proxy(block, label, "=", path, upstreams["web"])
    require_location_proxy(block, label, "^~", "/assets/", upstreams["web"])
    for path in ["/static/", "/img/", "/buttons/", "/flag-icons/", "/web/", "/mfa/"]:
        require_location_proxy(block, label, "^~", path, upstreams["casdoor"])
    require_location_proxy(block, label, "=", "/account", upstreams["casdoor"])
    require_location_proxy(block, label, "^~", "/signup", upstreams["casdoor"])
    require_location_proxy(block, label, "^~", "/forget", upstreams["casdoor"])
    require_location_return(block, label, None, "/", "302", "https://stuhelper.com$request_uri")


def validate_sso(block: Node, upstreams: dict[str, str]) -> None:
    label = "sso.stuhelper.com"
    require_tls(block, label)
    casdoor_upstream = upstreams["casdoor"]

    exact_discovery = location_proxy_matches(block, "=", "/.well-known/openid-configuration", casdoor_upstream)
    exact_jwks = location_proxy_matches(block, "=", "/.well-known/jwks", casdoor_upstream)
    well_known_prefix = location_proxy_matches(block, "^~", "/.well-known/", casdoor_upstream)
    has_static_well_known_risk = recursive_has(block, {"root", "try_files"}) or has_static_well_known_location(block, casdoor_upstream)
    if has_static_well_known_risk and not (exact_discovery and exact_jwks):
        raise CheckError(
            f"{label}: Baota static /.well-known handling requires exact openid-configuration and jwks proxy_pass {casdoor_upstream}"
        )
    if not well_known_prefix and not (exact_discovery and exact_jwks):
        raise CheckError(
            f"{label}: missing location ^~ /.well-known/ or exact openid-configuration and jwks proxy_pass {casdoor_upstream}"
        )

    header_locations: list[tuple[str, Node]] = []
    for modifier, path in [
        ("=", "/.well-known/openid-configuration"),
        ("=", "/.well-known/jwks"),
        ("^~", "/.well-known/"),
        ("^~", "/api/"),
        ("^~", "/"),
        (None, "/"),
    ]:
        loc = location(block, modifier, path)
        if loc is not None and casdoor_upstream in proxy_pass_values(loc):
            rendered = f"location {modifier + ' ' if modifier else ''}{path}"
            header_locations.append((rendered, loc))

    for rendered, loc in header_locations:
        require_proxy_header_on_server_or_location(block, loc, label, rendered, "Host", {"$host", "$http_host"})
        require_proxy_header_on_server_or_location(block, loc, label, rendered, "X-Forwarded-Proto", {"https", "$scheme"})
        require_proxy_header_on_server_or_location(block, loc, label, rendered, "X-Forwarded-Host", {"$host"})

    require_location_proxy(block, label, "^~", "/api/", casdoor_upstream)
    if not (
        location_proxy_matches(block, None, "/", casdoor_upstream)
        or location_proxy_matches(block, "^~", "/", casdoor_upstream)
    ):
        raise CheckError(
            f"{label}: missing location / or location ^~ / proxy_pass {casdoor_upstream}"
        )


def matching_https_servers(servers: list[Node], domain: str) -> list[Node]:
    return [server for server in servers if domain in server_names(server) and has_https_listen(server)]


def require_valid_server(servers: list[Node], domain: str, validator) -> None:
    candidates = matching_https_servers(servers, domain)
    if not candidates:
        raise CheckError(f"{domain}: missing HTTPS server block")
    errors: list[str] = []
    for candidate in candidates:
        try:
            validator(candidate)
            return
        except CheckError as exc:
            errors.append(str(exc))
    joined = " | ".join(errors[:4])
    raise CheckError(f"{domain}: no HTTPS server block satisfies the ingress contract: {joined}")


def upstream(direct_env: str, port_env: str | None, default_port: str) -> str:
    direct_value = os.environ.get(direct_env, "").strip()
    if direct_value:
        return direct_value.rstrip("/")
    if port_env:
        port = os.environ.get(port_env, default_port).strip() or default_port
    else:
        port = default_port
    return f"http://127.0.0.1:{port}"


def selected_profiles() -> set[str]:
    raw = os.environ.get("NGINX_PUBLIC_INGRESS_PROFILE", "stuhelper").strip().lower()
    parts = {part for part in re.split(r"[\s,]+", raw) if part}
    aliases = {
        "all": "all",
        "app": "stuhelper",
        "main": "stuhelper",
        "identity": "stuhelper",
        "stuhelper": "stuhelper",
        "casdoor": "sso",
        "sso": "sso",
    }
    if not parts:
        parts = {"stuhelper"}
    profiles = {aliases.get(part, part) for part in parts}
    if "all" in profiles:
        profiles = {"stuhelper", "sso"}
    unknown = profiles - {"stuhelper", "sso"}
    if unknown:
        raise CheckError(f"unknown NGINX_PUBLIC_INGRESS_PROFILE value: {', '.join(sorted(unknown))}")
    return profiles


try:
    config_path = Path(sys.argv[1])
    root_nodes, _ = parse_nodes(tokenize(config_path.read_text(encoding="utf-8", errors="replace")))
    servers = [node for node in walk(root_nodes) if node.name == "server"]
    if not servers:
        raise CheckError("no server blocks found in Nginx config")

    upstreams = {
        "backend": upstream("NGINX_PUBLIC_INGRESS_BACKEND_UPSTREAM", "BACKEND_EXTERNAL_PORT", "18080"),
        "web": upstream("NGINX_PUBLIC_INGRESS_WEB_UPSTREAM", "WEB_EXTERNAL_PORT", "18000"),
        "admin": upstream("NGINX_PUBLIC_INGRESS_ADMIN_UPSTREAM", "ADMIN_EXTERNAL_PORT", "18001"),
        "casdoor": upstream("NGINX_PUBLIC_INGRESS_CASDOOR_UPSTREAM", None, "8087"),
    }

    profiles = selected_profiles()
    if "stuhelper" in profiles:
        require_valid_server(servers, "stuhelper.com", lambda block: validate_main(block, upstreams))
        require_valid_server(servers, "www.stuhelper.com", validate_www)
        require_valid_server(servers, "id.stuhelper.com", lambda block: validate_identity(block, upstreams))
    if "sso" in profiles:
        require_valid_server(servers, "sso.stuhelper.com", lambda block: validate_sso(block, upstreams))
except CheckError as exc:
    print(f"[stuhelper][error] public Nginx ingress config preflight failed: {exc}", file=sys.stderr)
    raise SystemExit(1)
PY

log "public Nginx ingress config preflight passed (${NGINX_PUBLIC_INGRESS_PROFILE:-stuhelper}; ${source_label})"
