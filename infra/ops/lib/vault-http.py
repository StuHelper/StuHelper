#!/usr/bin/env python3
"""Minimal Vault HTTP client that keeps bearer tokens out of process arguments."""

from __future__ import annotations

import argparse
import ipaddress
import json
import os
import stat
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path


MAX_RESPONSE_BYTES = 4 * 1024 * 1024


class NoRedirectHandler(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):  # noqa: ANN001
        return None


def fail(message: str) -> None:
    raise SystemExit(f"[vault-http][error] {message}")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--address", required=True)
    parser.add_argument("--token-file", required=True)
    parser.add_argument("--namespace", default="")
    parser.add_argument("--method", choices=("GET", "POST", "PUT"), required=True)
    parser.add_argument("--path", required=True)
    parser.add_argument("--data-file")
    parser.add_argument("--output-file", required=True)
    parser.add_argument("--timeout-seconds", type=int, default=10)
    return parser.parse_args()


def validate_address(raw: str) -> str:
    parsed = urllib.parse.urlsplit(raw.strip().rstrip("/"))
    if parsed.scheme not in {"http", "https"}:
        fail("Vault address must use http or https")
    if not parsed.hostname or parsed.username is not None or parsed.password is not None:
        fail("Vault address must contain a hostname and no embedded credentials")
    if parsed.query or parsed.fragment or parsed.path not in {"", "/"}:
        fail("Vault address must not contain a path, query, fragment, or embedded credentials")
    if parsed.scheme == "http":
        host = parsed.hostname.lower()
        loopback = host == "localhost"
        if not loopback:
            try:
                loopback = ipaddress.ip_address(host).is_loopback
            except ValueError:
                loopback = False
        if not loopback:
            fail("plaintext Vault HTTP is allowed only on loopback; use https for remote Vault")
    return raw.strip().rstrip("/")


def read_token(path: Path) -> str:
    if path.is_symlink():
        fail("Vault token file must not be a symbolic link")
    try:
        metadata = path.stat()
        token = path.read_text(encoding="utf-8").strip()
    except OSError as exc:
        fail(f"cannot read Vault token file: {exc}")
    if not stat.S_ISREG(metadata.st_mode):
        fail("Vault token file must be a regular file")
    if stat.S_IMODE(metadata.st_mode) & 0o077:
        fail("Vault token file must not be readable or writable by group/other")
    if not token:
        fail("Vault token file is empty")
    if "\n" in token or "\r" in token:
        fail("Vault token file must contain exactly one token")
    return token


def read_payload(path: str | None) -> bytes | None:
    if not path:
        return None
    try:
        payload = Path(path).read_bytes()
    except OSError as exc:
        fail(f"cannot read Vault request payload: {exc}")
    try:
        json.loads(payload)
    except json.JSONDecodeError as exc:
        fail(f"Vault request payload is not valid JSON: {exc}")
    return payload


def main() -> None:
    args = parse_args()
    if not 1 <= args.timeout_seconds <= 60:
        fail("timeout must be between 1 and 60 seconds")

    address = validate_address(args.address)
    token = read_token(Path(args.token_file))
    request_path = args.path.strip().lstrip("/")
    if not request_path or "?" in request_path or "#" in request_path:
        fail("Vault API path must be a non-empty path without query or fragment")

    payload = read_payload(args.data_file)
    if args.method == "GET" and payload is not None:
        fail("GET requests must not include a payload")

    headers = {
        "Accept": "application/json",
        "X-Vault-Token": token,
    }
    if payload is not None:
        headers["Content-Type"] = "application/json"
    if args.namespace:
        headers["X-Vault-Namespace"] = args.namespace

    request = urllib.request.Request(
        f"{address}/v1/{request_path}",
        data=payload,
        headers=headers,
        method=args.method,
    )
    # Vault is a directly addressed control-plane service. Ignore ambient proxy
    # variables so an operator shell cannot accidentally forward the bearer
    # token to an HTTP(S) proxy.
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}), NoRedirectHandler())
    try:
        with opener.open(request, timeout=args.timeout_seconds) as response:
            body = response.read(MAX_RESPONSE_BYTES + 1)
    except urllib.error.HTTPError as exc:
        fail(f"Vault API request failed with HTTP {exc.code} for {request_path}")
    except (urllib.error.URLError, TimeoutError, OSError) as exc:
        fail(f"Vault API request failed for {request_path}: {exc}")

    if len(body) > MAX_RESPONSE_BYTES:
        fail("Vault API response exceeded the 4 MiB safety limit")
    if not body:
        body = b"{}\n"
    try:
        json.loads(body)
    except json.JSONDecodeError as exc:
        fail(f"Vault API returned invalid JSON for {request_path}: {exc}")

    output = Path(args.output_file)
    flags = os.O_WRONLY | os.O_CREAT | os.O_TRUNC
    if hasattr(os, "O_NOFOLLOW"):
        flags |= os.O_NOFOLLOW
    try:
        descriptor = os.open(output, flags, 0o600)
        with os.fdopen(descriptor, "wb") as stream:
            stream.write(body)
        output.chmod(0o600)
    except OSError as exc:
        fail(f"cannot write Vault API response: {exc}")


if __name__ == "__main__":
    main()
