#!/usr/bin/env python3
"""Validate a root-managed external PostgreSQL PITR attestation."""

from __future__ import annotations

import argparse
from datetime import datetime, timedelta, timezone
import json
import os
from pathlib import Path
import re
import stat
from typing import Any
from urllib.parse import urlparse


MAX_EVIDENCE_BYTES = 64 * 1024
MAX_OBSERVATION_AGE = timedelta(minutes=30)
MAX_ARCHIVE_LAG = timedelta(minutes=30)
MAX_EVIDENCE_LIFETIME = timedelta(hours=1)
MAX_RESTORE_DRILL_AGE = timedelta(days=90)
MIN_RETENTION_HOURS = 168
MAX_RPO_SECONDS = 900
EVIDENCE_ID = re.compile(r"[A-Za-z0-9][A-Za-z0-9._:-]{0,127}")
SYSTEM_IDENTIFIER = re.compile(r"[0-9]{10,20}")


def require_exact_keys(document: dict[str, Any], expected: set[str], label: str) -> None:
    actual = set(document)
    if actual != expected:
        missing = sorted(expected - actual)
        unknown = sorted(actual - expected)
        details: list[str] = []
        if missing:
            details.append(f"missing={','.join(missing)}")
        if unknown:
            details.append(f"unknown={','.join(unknown)}")
        raise ValueError(f"{label} keys are invalid ({'; '.join(details)})")


def require_boolean(document: dict[str, Any], key: str, label: str) -> bool:
    value = document.get(key)
    if not isinstance(value, bool):
        raise ValueError(f"{label}.{key} must be a boolean")
    return value


def require_integer(document: dict[str, Any], key: str, label: str) -> int:
    value = document.get(key)
    if isinstance(value, bool) or not isinstance(value, int):
        raise ValueError(f"{label}.{key} must be an integer")
    return value


def require_text(document: dict[str, Any], key: str, label: str) -> str:
    value = document.get(key)
    if not isinstance(value, str) or not value.strip() or len(value) > 256:
        raise ValueError(f"{label}.{key} must be a non-empty string of at most 256 characters")
    if any(ord(character) < 32 or ord(character) == 127 for character in value):
        raise ValueError(f"{label}.{key} contains control characters")
    return value.strip()


def parse_utc_timestamp(value: Any, label: str) -> datetime:
    if not isinstance(value, str) or not value.endswith("Z"):
        raise ValueError(f"{label} must be an RFC3339 UTC timestamp ending in Z")
    try:
        parsed = datetime.fromisoformat(value[:-1] + "+00:00")
    except ValueError as exc:
        raise ValueError(f"{label} is not a valid RFC3339 timestamp") from exc
    if parsed.tzinfo is None or parsed.utcoffset() != timedelta(0):
        raise ValueError(f"{label} must use UTC")
    return parsed.astimezone(timezone.utc)


def open_protected_evidence(path: Path, expected_owner_uid: int) -> dict[str, Any]:
    parent = path.parent
    parent_status = parent.lstat()
    if not stat.S_ISDIR(parent_status.st_mode):
        raise ValueError("external PostgreSQL PITR evidence parent is not a directory")
    if parent_status.st_uid != expected_owner_uid:
        raise ValueError("external PostgreSQL PITR evidence parent has the wrong owner")
    if parent_status.st_mode & (stat.S_IWGRP | stat.S_IWOTH):
        raise ValueError("external PostgreSQL PITR evidence parent must not be group/other writable")

    flags = os.O_RDONLY | getattr(os, "O_NOFOLLOW", 0)
    try:
        descriptor = os.open(path, flags)
    except OSError as exc:
        raise ValueError(f"cannot open protected external PostgreSQL PITR evidence: {exc}") from exc
    try:
        status = os.fstat(descriptor)
        if not stat.S_ISREG(status.st_mode):
            raise ValueError("external PostgreSQL PITR evidence must be a regular file")
        if status.st_uid != expected_owner_uid:
            raise ValueError("external PostgreSQL PITR evidence has the wrong owner")
        if status.st_mode & (stat.S_IWGRP | stat.S_IWOTH):
            raise ValueError("external PostgreSQL PITR evidence must not be group/other writable")
        if status.st_size <= 0 or status.st_size > MAX_EVIDENCE_BYTES:
            raise ValueError("external PostgreSQL PITR evidence size is invalid")
        with os.fdopen(descriptor, encoding="utf-8") as handle:
            descriptor = -1
            document = json.load(handle)
    except json.JSONDecodeError as exc:
        raise ValueError(f"external PostgreSQL PITR evidence is invalid JSON: {exc}") from exc
    finally:
        if descriptor >= 0:
            os.close(descriptor)
    if not isinstance(document, dict):
        raise ValueError("external PostgreSQL PITR evidence root must be an object")
    return document


def validate(
    document: dict[str, Any],
    expected_system_identifier: str,
    now: datetime,
) -> dict[str, Any]:
    require_exact_keys(
        document,
        {
            "schemaVersion",
            "evidenceId",
            "provider",
            "evidenceUri",
            "clusterSystemIdentifier",
            "observedAt",
            "expiresAt",
            "continuousArchiving",
            "restoreDrill",
        },
        "evidence",
    )
    if require_integer(document, "schemaVersion", "evidence") != 1:
        raise ValueError("evidence.schemaVersion must be 1")
    evidence_id = require_text(document, "evidenceId", "evidence")
    if not EVIDENCE_ID.fullmatch(evidence_id):
        raise ValueError("evidence.evidenceId has an invalid format")
    provider = require_text(document, "provider", "evidence")
    evidence_uri = require_text(document, "evidenceUri", "evidence")
    parsed_uri = urlparse(evidence_uri)
    if (
        parsed_uri.scheme != "https"
        or not parsed_uri.hostname
        or parsed_uri.username is not None
        or parsed_uri.password is not None
        or parsed_uri.fragment
    ):
        raise ValueError("evidence.evidenceUri must be an HTTPS URL without credentials or a fragment")

    system_identifier = require_text(document, "clusterSystemIdentifier", "evidence")
    if not SYSTEM_IDENTIFIER.fullmatch(system_identifier):
        raise ValueError("evidence.clusterSystemIdentifier must be a PostgreSQL system identifier")
    if system_identifier != expected_system_identifier:
        raise ValueError("external PostgreSQL PITR evidence is for a different live cluster")

    observed_at = parse_utc_timestamp(document.get("observedAt"), "evidence.observedAt")
    expires_at = parse_utc_timestamp(document.get("expiresAt"), "evidence.expiresAt")
    if observed_at > now + timedelta(minutes=1):
        raise ValueError("external PostgreSQL PITR evidence observation is in the future")
    if now - observed_at > MAX_OBSERVATION_AGE:
        raise ValueError("external PostgreSQL PITR evidence observation is stale")
    if expires_at <= now:
        raise ValueError("external PostgreSQL PITR evidence has expired")
    if expires_at > observed_at + MAX_EVIDENCE_LIFETIME:
        raise ValueError("external PostgreSQL PITR evidence lifetime exceeds one hour")

    continuous = document.get("continuousArchiving")
    if not isinstance(continuous, dict):
        raise ValueError("evidence.continuousArchiving must be an object")
    require_exact_keys(
        continuous,
        {"enabled", "offHost", "rpoSeconds", "retentionHours", "lastArchivedAt"},
        "evidence.continuousArchiving",
    )
    if not require_boolean(continuous, "enabled", "evidence.continuousArchiving"):
        raise ValueError("external PostgreSQL continuous archiving is not enabled")
    if not require_boolean(continuous, "offHost", "evidence.continuousArchiving"):
        raise ValueError("external PostgreSQL WAL archive is not off-host")
    rpo_seconds = require_integer(continuous, "rpoSeconds", "evidence.continuousArchiving")
    if rpo_seconds <= 0 or rpo_seconds > MAX_RPO_SECONDS:
        raise ValueError("external PostgreSQL PITR RPO must be between 1 and 900 seconds")
    retention_hours = require_integer(
        continuous,
        "retentionHours",
        "evidence.continuousArchiving",
    )
    if retention_hours < MIN_RETENTION_HOURS:
        raise ValueError("external PostgreSQL PITR retention must be at least 168 hours")
    last_archived_at = parse_utc_timestamp(
        continuous.get("lastArchivedAt"),
        "evidence.continuousArchiving.lastArchivedAt",
    )
    if last_archived_at > observed_at + timedelta(minutes=1):
        raise ValueError("external PostgreSQL last archive time is newer than its observation")
    if now - last_archived_at > MAX_ARCHIVE_LAG:
        raise ValueError("external PostgreSQL latest archived WAL is stale")

    restore = document.get("restoreDrill")
    if not isinstance(restore, dict):
        raise ValueError("evidence.restoreDrill must be an object")
    require_exact_keys(
        restore,
        {
            "status",
            "completedAt",
            "isolatedTarget",
            "baseBackupVerified",
            "walReplayVerified",
        },
        "evidence.restoreDrill",
    )
    if require_text(restore, "status", "evidence.restoreDrill") != "passed":
        raise ValueError("external PostgreSQL PITR restore drill has not passed")
    completed_at = parse_utc_timestamp(
        restore.get("completedAt"),
        "evidence.restoreDrill.completedAt",
    )
    if completed_at > observed_at + timedelta(minutes=1):
        raise ValueError("external PostgreSQL restore drill is newer than its observation")
    if now - completed_at > MAX_RESTORE_DRILL_AGE:
        raise ValueError("external PostgreSQL PITR restore drill is older than 90 days")
    for key in ("isolatedTarget", "baseBackupVerified", "walReplayVerified"):
        if not require_boolean(restore, key, "evidence.restoreDrill"):
            raise ValueError(f"external PostgreSQL restore drill requires {key}=true")

    return {
        "verified": True,
        "schemaVersion": 1,
        "evidenceId": evidence_id,
        "provider": provider,
        "evidenceUri": evidence_uri,
        "clusterSystemIdentifier": system_identifier,
        "observedAt": document["observedAt"],
        "expiresAt": document["expiresAt"],
        "continuousArchiving": continuous,
        "restoreDrill": restore,
    }


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--evidence-file", required=True, type=Path)
    parser.add_argument("--expected-system-identifier", required=True)
    parser.add_argument("--expected-owner-uid", required=True, type=int)
    parser.add_argument("--now", help="RFC3339 UTC test clock")
    args = parser.parse_args()

    try:
        if args.expected_owner_uid < 0:
            raise ValueError("expected owner uid must be non-negative")
        expected_system_identifier = args.expected_system_identifier.strip()
        if not SYSTEM_IDENTIFIER.fullmatch(expected_system_identifier):
            raise ValueError("live PostgreSQL system identifier is invalid")
        now = (
            parse_utc_timestamp(args.now, "--now")
            if args.now
            else datetime.now(timezone.utc)
        )
        document = open_protected_evidence(args.evidence_file, args.expected_owner_uid)
        summary = validate(document, expected_system_identifier, now)
    except (OSError, ValueError) as exc:
        print(f"invalid external PostgreSQL PITR evidence: {exc}", flush=True)
        return 1

    print(json.dumps(summary, ensure_ascii=True, separators=(",", ":")))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
