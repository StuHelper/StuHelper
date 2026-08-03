#!/usr/bin/env python3
"""Validate live PostgreSQL WAL archiver settings and post-start progress."""

from __future__ import annotations

import argparse
import json
import re
from typing import Any


EXPECTED_ARCHIVE_COMMAND = (
    "sh -c 'dest=/var/lib/postgresql/wal-archive/%f; "
    'tmp="$dest.tmp.$$"; '
    'if [ -f "$dest" ]; then cmp -s %p "$dest"; '
    'else cp %p "$tmp" && mv "$tmp" "$dest"; fi\''
)
SAFE_ARCHIVE_NAME = re.compile(r"[A-Za-z0-9][A-Za-z0-9._-]*")


def require_nonnegative_integer(document: dict[str, Any], key: str) -> int:
    value = document.get(key)
    if isinstance(value, bool) or not isinstance(value, int) or value < 0:
        raise ValueError(f"{key} must be a non-negative integer")
    return value


def optional_epoch(document: dict[str, Any], key: str) -> float | None:
    value = document.get(key)
    if value is None:
        return None
    if isinstance(value, bool) or not isinstance(value, (int, float)):
        raise ValueError(f"{key} must be a numeric epoch or null")
    return float(value)


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--status-json", required=True)
    args = parser.parse_args()

    try:
        document = json.loads(args.status_json)
        if not isinstance(document, dict):
            raise ValueError("status must be an object")
        archive_mode = document.get("archive_mode")
        archive_command = document.get("archive_command")
        archive_timeout = document.get("archive_timeout")
        archived_count = require_nonnegative_integer(document, "archived_count")
        require_nonnegative_integer(document, "failed_count")
        last_archived_epoch = optional_epoch(document, "last_archived_epoch")
        last_failed_epoch = optional_epoch(document, "last_failed_epoch")
        postmaster_started_epoch = optional_epoch(document, "postmaster_started_epoch")
    except (json.JSONDecodeError, ValueError) as exc:
        print(f"invalid PostgreSQL WAL archiver status: {exc}", flush=True)
        return 1

    if archive_mode != "on":
        print("live PostgreSQL archive_mode must be on", flush=True)
        return 1
    if archive_timeout != "15min":
        print("live PostgreSQL archive_timeout must be 15min", flush=True)
        return 1
    if archive_command != EXPECTED_ARCHIVE_COMMAND:
        print("live PostgreSQL archive_command does not match the protected command", flush=True)
        return 1
    if postmaster_started_epoch is None:
        print("live PostgreSQL postmaster start time is missing", flush=True)
        return 1

    last_archived_wal = document.get("last_archived_wal")
    needs_probe = (
        archived_count == 0
        or last_archived_epoch is None
        or last_archived_epoch < postmaster_started_epoch
        or (last_failed_epoch is not None and last_failed_epoch >= last_archived_epoch)
    )
    if needs_probe:
        return 2
    if not isinstance(last_archived_wal, str) or not SAFE_ARCHIVE_NAME.fullmatch(
        last_archived_wal,
    ):
        print("live PostgreSQL last_archived_wal is missing or unsafe", flush=True)
        return 1

    print(last_archived_wal)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
