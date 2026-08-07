#!/usr/bin/env python3
"""Render the fail-closed Baota/Nginx stream ingress for Campus Connector."""

from __future__ import annotations

import argparse
import ipaddress
import os
from pathlib import Path
import re
import sys
from typing import NoReturn


PLACEHOLDERS = {
    "__PUBLIC_PORT__",
    "__UPSTREAM_PORT__",
    "__ALLOW_DIRECTIVES__",
}


def fail(message: str) -> NoReturn:
    raise ValueError(message)


def parse_port(raw: str, name: str) -> int:
    value = raw.strip()
    if not re.fullmatch(r"[1-9][0-9]{0,4}", value):
        fail(f"{name} must be a decimal port between 1 and 65535")
    port = int(value)
    if port > 65535:
        fail(f"{name} must be a decimal port between 1 and 65535")
    return port


def parse_allowed_cidrs(raw: str) -> list[str]:
    values = [part.strip() for part in raw.split(",")]
    if not values or any(not value for value in values):
        fail("CAMPUS_CONNECTOR_ALLOWED_SOURCE_CIDRS must contain one or more explicit IPv4 CIDRs")
    if len(values) > 64:
        fail("CAMPUS_CONNECTOR_ALLOWED_SOURCE_CIDRS must contain at most 64 CIDRs")

    result: list[str] = []
    seen: set[str] = set()
    for value in values:
        try:
            network = ipaddress.ip_network(value, strict=True)
        except ValueError as exc:
            fail(f"invalid source CIDR {value!r}: {exc}")
        if network.version != 4:
            fail(f"source CIDR {value!r} must be IPv4 for the current IPv4 stream listener")
        if network.prefixlen == 0:
            fail("CAMPUS_CONNECTOR_ALLOWED_SOURCE_CIDRS must not contain 0.0.0.0/0")
        if network.is_multicast or network.is_unspecified:
            fail(f"source CIDR {value!r} is not an admissible client source network")
        canonical = str(network)
        if canonical in seen:
            fail(f"duplicate source CIDR: {canonical}")
        seen.add(canonical)
        result.append(canonical)
    return result


def render(template: str, public_port: int, upstream_port: int, allowed_cidrs: list[str]) -> str:
    for placeholder in PLACEHOLDERS:
        count = template.count(placeholder)
        if count != 1:
            fail(f"template must contain {placeholder} exactly once (found {count})")
    unknown = sorted(set(re.findall(r"__[A-Z0-9_]+__", template)) - PLACEHOLDERS)
    if unknown:
        fail(f"template contains unknown placeholders: {', '.join(unknown)}")

    allow_directives = "\n".join(f"    allow {cidr};" for cidr in allowed_cidrs)
    rendered = (
        template.replace("__PUBLIC_PORT__", str(public_port))
        .replace("__UPSTREAM_PORT__", str(upstream_port))
        .replace("__ALLOW_DIRECTIVES__", allow_directives)
    )
    if re.search(r"__[A-Z0-9_]+__", rendered):
        fail("rendered stream config still contains a placeholder")
    return rendered


def write_private(path: Path, content: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content, encoding="utf-8")
    os.chmod(path, 0o600)


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--template", required=True, type=Path)
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--public-port", required=True)
    parser.add_argument("--upstream-port", required=True)
    parser.add_argument("--allowed-cidrs", required=True)
    args = parser.parse_args()

    try:
        public_port = parse_port(args.public_port, "CAMPUS_CONNECTOR_GATEWAY_PUBLIC_PORT")
        upstream_port = parse_port(args.upstream_port, "CAMPUS_CONNECTOR_GATEWAY_EXTERNAL_PORT")
        if public_port == upstream_port:
            fail("public and loopback upstream ports must differ to prevent a stream proxy loop")
        allowed_cidrs = parse_allowed_cidrs(args.allowed_cidrs)
        template = args.template.read_text(encoding="utf-8")
        write_private(args.output, render(template, public_port, upstream_port, allowed_cidrs))
    except (OSError, ValueError) as exc:
        print(f"[campus-connector-nginx-render][error] {exc}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
