#!/usr/bin/env python3
"""Publish and validate an audited activation for a pre-existing PostgreSQL backup control plane."""

from __future__ import annotations

import argparse
import datetime as dt
import hashlib
import json
import os
import re
import stat
import tempfile
from pathlib import Path
from typing import Any


SCHEMA_VERSION = 2
EVENT = "existing_postgres_backup_control_activated"
CURRENT_NAME = "postgres-backup-activation.json"
RECORDS_DIR_NAME = "postgres-backup-activations"
SAFE_ID = re.compile(r"[A-Za-z0-9][A-Za-z0-9._-]{0,127}")
SAFE_DOCKER_NAME = re.compile(r"[A-Za-z0-9][A-Za-z0-9_.-]{0,127}")
DIGEST_REF = re.compile(r"[^\s@]+@sha256:[0-9a-f]{64}")
SYSTEM_IDENTIFIER = re.compile(r"[0-9]{10,20}")
SHA256 = re.compile(r"[0-9a-f]{64}")
BACKUP_FILE = re.compile(r"[A-Za-z0-9][A-Za-z0-9._-]{0,254}")
EXPECTED_FIELDS = {
    "schemaVersion",
    "event",
    "activationId",
    "activatedAt",
    "stackName",
    "postgresContainerName",
    "postgresImageRef",
    "postgresSystemIdentifier",
    "postgresDataVolume",
    "postgresWalArchiveVolume",
    "previousActivation",
    "recoveryEvidence",
}


class ActivationError(RuntimeError):
    pass


def canonical_payload(document: dict[str, Any]) -> bytes:
    return (json.dumps(document, sort_keys=True, separators=(",", ":")) + "\n").encode()


def fsync_directory(path: Path) -> None:
    fd = os.open(path, os.O_RDONLY | os.O_DIRECTORY)
    try:
        os.fsync(fd)
    finally:
        os.close(fd)


def require_state_directory(path: Path) -> None:
    try:
        metadata = path.lstat()
    except FileNotFoundError as exc:
        raise ActivationError(f"deployment state directory does not exist: {path}") from exc
    if not stat.S_ISDIR(metadata.st_mode) or path.is_symlink():
        raise ActivationError(f"deployment state path must be a regular directory: {path}")
    if metadata.st_uid != os.geteuid():
        raise ActivationError(f"deployment state directory must be owned by the deploy user: {path}")
    if stat.S_IMODE(metadata.st_mode) & 0o022:
        raise ActivationError(f"deployment state directory must not be group- or world-writable: {path}")


def require_records_directory(path: Path) -> None:
    metadata = path.lstat()
    if not stat.S_ISDIR(metadata.st_mode) or path.is_symlink():
        raise ActivationError(f"PostgreSQL backup activation record path must be a regular directory: {path}")
    if metadata.st_uid != os.geteuid():
        raise ActivationError(f"PostgreSQL backup activation record directory must be owned by the deploy user: {path}")
    if stat.S_IMODE(metadata.st_mode) != 0o700:
        raise ActivationError(f"PostgreSQL backup activation record directory must use mode 0700: {path}")


def read_protected(path: Path) -> bytes:
    flags = os.O_RDONLY
    if hasattr(os, "O_NOFOLLOW"):
        flags |= os.O_NOFOLLOW
    try:
        fd = os.open(path, flags)
    except FileNotFoundError:
        raise
    except OSError as exc:
        raise ActivationError(f"cannot safely open PostgreSQL backup activation record: {path}") from exc
    try:
        metadata = os.fstat(fd)
        if not stat.S_ISREG(metadata.st_mode):
            raise ActivationError(f"PostgreSQL backup activation record must be a regular file: {path}")
        if metadata.st_uid != os.geteuid():
            raise ActivationError(f"PostgreSQL backup activation record must be owned by the deploy user: {path}")
        if stat.S_IMODE(metadata.st_mode) != 0o600:
            raise ActivationError(f"PostgreSQL backup activation record must use mode 0600: {path}")
        if metadata.st_size > 16 * 1024:
            raise ActivationError(f"PostgreSQL backup activation record is unexpectedly large: {path}")
        with os.fdopen(fd, "rb", closefd=False) as stream:
            return stream.read()
    finally:
        os.close(fd)


def validate_timestamp(value: object) -> str:
    if not isinstance(value, str) or not re.fullmatch(r"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z", value):
        raise ActivationError("activatedAt must use UTC YYYY-MM-DDTHH:MM:SSZ")
    parsed = dt.datetime.strptime(value, "%Y-%m-%dT%H:%M:%SZ").replace(tzinfo=dt.timezone.utc)
    if parsed > dt.datetime.now(dt.timezone.utc) + dt.timedelta(minutes=5):
        raise ActivationError("activatedAt cannot be in the future")
    return value


def validate_document(payload: bytes, path: Path) -> dict[str, Any]:
    try:
        document = json.loads(payload)
    except (UnicodeDecodeError, json.JSONDecodeError) as exc:
        raise ActivationError(f"PostgreSQL backup activation record is not valid UTF-8 JSON: {path}") from exc
    if not isinstance(document, dict) or set(document) != EXPECTED_FIELDS:
        raise ActivationError(f"PostgreSQL backup activation record has unexpected fields: {path}")
    if document.get("schemaVersion") != SCHEMA_VERSION or document.get("event") != EVENT:
        raise ActivationError(f"PostgreSQL backup activation record has an unsupported schema: {path}")
    if canonical_payload(document) != payload:
        raise ActivationError(f"PostgreSQL backup activation record is not canonical: {path}")

    activation_id = document.get("activationId")
    if not isinstance(activation_id, str) or not SAFE_ID.fullmatch(activation_id):
        raise ActivationError(f"PostgreSQL backup activation ID is invalid: {path}")
    validate_timestamp(document.get("activatedAt"))
    for key in ("stackName", "postgresContainerName", "postgresDataVolume", "postgresWalArchiveVolume"):
        value = document.get(key)
        if not isinstance(value, str) or not SAFE_DOCKER_NAME.fullmatch(value):
            raise ActivationError(f"PostgreSQL backup activation field {key} is invalid: {path}")
    image_ref = document.get("postgresImageRef")
    if not isinstance(image_ref, str) or not DIGEST_REF.fullmatch(image_ref):
        raise ActivationError(f"PostgreSQL backup activation image is not an immutable digest: {path}")
    system_identifier = document.get("postgresSystemIdentifier")
    if not isinstance(system_identifier, str) or not SYSTEM_IDENTIFIER.fullmatch(system_identifier):
        raise ActivationError(f"PostgreSQL backup activation system identifier is invalid: {path}")
    previous = document.get("previousActivation")
    if previous is not None:
        if not isinstance(previous, dict) or set(previous) != {"activationId", "sha256"}:
            raise ActivationError(f"PostgreSQL backup activation predecessor is invalid: {path}")
        previous_id = previous.get("activationId")
        previous_sha256 = previous.get("sha256")
        if not isinstance(previous_id, str) or not SAFE_ID.fullmatch(previous_id):
            raise ActivationError(f"PostgreSQL backup activation predecessor ID is invalid: {path}")
        if not isinstance(previous_sha256, str) or not SHA256.fullmatch(previous_sha256):
            raise ActivationError(f"PostgreSQL backup activation predecessor digest is invalid: {path}")
    evidence = document.get("recoveryEvidence")
    if not isinstance(evidence, dict) or set(evidence) != {"logicalBackup", "physicalBaseBackup"}:
        raise ActivationError(f"PostgreSQL backup activation recovery evidence is invalid: {path}")
    for key, suffix in (("logicalBackup", ".dump"), ("physicalBaseBackup", ".tar.gz")):
        artifact = evidence.get(key)
        if not isinstance(artifact, dict) or set(artifact) != {"file", "sha256"}:
            raise ActivationError(f"PostgreSQL backup activation {key} evidence is invalid: {path}")
        filename = artifact.get("file")
        digest = artifact.get("sha256")
        if (
            not isinstance(filename, str)
            or not BACKUP_FILE.fullmatch(filename)
            or not filename.endswith(suffix)
        ):
            raise ActivationError(f"PostgreSQL backup activation {key} filename is invalid: {path}")
        if not isinstance(digest, str) or not SHA256.fullmatch(digest):
            raise ActivationError(f"PostgreSQL backup activation {key} digest is invalid: {path}")
    return document


def expected_identity(args: argparse.Namespace) -> dict[str, str]:
    return {
        "stackName": args.stack_name,
        "postgresContainerName": args.postgres_container_name,
        "postgresImageRef": args.postgres_image_ref,
        "postgresSystemIdentifier": args.postgres_system_identifier,
        "postgresDataVolume": args.postgres_data_volume,
        "postgresWalArchiveVolume": args.postgres_wal_archive_volume,
    }


def validate_identity(document: dict[str, Any], expected: dict[str, str], path: Path) -> None:
    for key, value in expected.items():
        if document.get(key) != value:
            raise ActivationError(f"PostgreSQL backup activation field {key} does not match the live datastore: {path}")


def load_chain(state_dir: Path) -> tuple[dict[str, Any], bytes]:
    current_path = state_dir / CURRENT_NAME
    payload = read_protected(current_path)
    document = validate_document(payload, current_path)
    records_dir = state_dir / RECORDS_DIR_NAME
    require_records_directory(records_dir)

    records: dict[str, tuple[dict[str, Any], bytes]] = {}
    for path in records_dir.iterdir():
        if not path.name.endswith(".json"):
            raise ActivationError(
                "PostgreSQL backup activation history contains unexpected or orphaned records"
            )
        record_payload = read_protected(path)
        record = validate_document(record_payload, path)
        activation_id = record["activationId"]
        if path.name != f"{activation_id}.json" or activation_id in records:
            raise ActivationError(
                "PostgreSQL backup activation history contains unexpected or orphaned records"
            )
        records[activation_id] = (record, record_payload)

    current_id = document["activationId"]
    immutable = records.get(current_id)
    if immutable is None:
        raise ActivationError(
            "PostgreSQL backup activation history contains unexpected or orphaned records"
        )
    immutable_document, immutable_payload = immutable
    if payload != immutable_payload or document != immutable_document:
        raise ActivationError("current PostgreSQL backup activation does not match its immutable record")

    seen: set[str] = set()
    cursor = document
    while True:
        cursor_id = cursor["activationId"]
        if cursor_id in seen:
            raise ActivationError("PostgreSQL backup activation history contains a cycle")
        seen.add(cursor_id)
        previous = cursor["previousActivation"]
        if previous is None:
            break
        previous_id = previous["activationId"]
        predecessor = records.get(previous_id)
        if predecessor is None:
            raise ActivationError(
                "PostgreSQL backup activation history contains unexpected or orphaned records"
            )
        predecessor_document, predecessor_payload = predecessor
        if hashlib.sha256(predecessor_payload).hexdigest() != previous["sha256"]:
            raise ActivationError("PostgreSQL backup activation predecessor digest does not match")
        if cursor["activatedAt"] < predecessor_document["activatedAt"]:
            raise ActivationError("PostgreSQL backup activation history is not chronological")
        cursor = predecessor_document

    if seen != set(records):
        raise ActivationError(
            "PostgreSQL backup activation history contains unexpected or orphaned records"
        )
    return document, payload


def load_current(state_dir: Path, expected: dict[str, str]) -> tuple[dict[str, Any], bytes]:
    document, payload = load_chain(state_dir)
    validate_identity(document, expected, state_dir / CURRENT_NAME)
    return document, payload


def stage_payload(path: Path, payload: bytes) -> Path:
    fd, temporary_name = tempfile.mkstemp(prefix=f".{path.name}.", dir=path.parent)
    temporary_path = Path(temporary_name)
    try:
        os.fchmod(fd, 0o600)
        with os.fdopen(fd, "wb") as stream:
            stream.write(payload)
            stream.flush()
            os.fsync(stream.fileno())
        return temporary_path
    except BaseException:
        try:
            os.close(fd)
        except OSError:
            pass
        temporary_path.unlink(missing_ok=True)
        raise


def atomic_write(path: Path, payload: bytes) -> None:
    temporary_path = stage_payload(path, payload)
    try:
        os.replace(temporary_path, path)
        fsync_directory(path.parent)
    except BaseException:
        temporary_path.unlink(missing_ok=True)
        raise


def publish(args: argparse.Namespace) -> dict[str, Any]:
    state_dir = Path(args.state_dir)
    require_state_directory(state_dir)
    expected = expected_identity(args)
    current_path = state_dir / CURRENT_NAME
    previous_activation: dict[str, str] | None = None
    if os.path.lexists(current_path):
        current, current_payload = load_chain(state_dir)
    else:
        current = None
        current_payload = b""
    if current is not None:
        try:
            validate_identity(current, expected, current_path)
        except ActivationError:
            if not args.supersede:
                raise ActivationError(
                    "live datastore identity changed; audited publication requires --supersede"
                ) from None
        else:
            return current
        previous_activation = {
            "activationId": current["activationId"],
            "sha256": hashlib.sha256(current_payload).hexdigest(),
        }

    records_dir = state_dir / RECORDS_DIR_NAME
    try:
        require_records_directory(records_dir)
    except FileNotFoundError:
        records_dir.mkdir(mode=0o700)
        fsync_directory(state_dir)
        require_records_directory(records_dir)

    surviving_records = list(records_dir.iterdir())
    if current is None and surviving_records:
        raise ActivationError(
            "immutable PostgreSQL backup activation evidence survives without a current pointer; manual reconciliation is required"
        )

    if not SAFE_ID.fullmatch(args.activation_id):
        raise ActivationError("activation ID must be a safe 1-128 character identifier")
    document: dict[str, Any] = {
        "schemaVersion": SCHEMA_VERSION,
        "event": EVENT,
        "activationId": args.activation_id,
        "activatedAt": dt.datetime.now(dt.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
        **expected,
        "previousActivation": previous_activation,
        "recoveryEvidence": {
            "logicalBackup": {
                "file": args.logical_backup_file,
                "sha256": args.logical_backup_sha256,
            },
            "physicalBaseBackup": {
                "file": args.base_backup_file,
                "sha256": args.base_backup_sha256,
            },
        },
    }
    payload = canonical_payload(document)
    validate_document(payload, current_path)

    immutable_path = records_dir / f"{args.activation_id}.json"
    temporary_path = stage_payload(immutable_path, payload)
    try:
        try:
            os.link(temporary_path, immutable_path)
        except FileExistsError as exc:
            raise ActivationError(f"PostgreSQL backup activation ID was already used: {args.activation_id}") from exc
        fsync_directory(records_dir)
    finally:
        temporary_path.unlink(missing_ok=True)
    atomic_write(current_path, payload)
    return document


def validate(args: argparse.Namespace) -> dict[str, Any]:
    state_dir = Path(args.state_dir)
    require_state_directory(state_dir)
    document, _ = load_current(state_dir, expected_identity(args))
    return document


def validate_chain(args: argparse.Namespace) -> dict[str, Any]:
    state_dir = Path(args.state_dir)
    require_state_directory(state_dir)
    document, _ = load_chain(state_dir)
    return document


def add_identity_arguments(parser: argparse.ArgumentParser) -> None:
    parser.add_argument("--state-dir", required=True)
    parser.add_argument("--stack-name", required=True)
    parser.add_argument("--postgres-container-name", required=True)
    parser.add_argument("--postgres-image-ref", required=True)
    parser.add_argument("--postgres-system-identifier", required=True)
    parser.add_argument("--postgres-data-volume", required=True)
    parser.add_argument("--postgres-wal-archive-volume", required=True)


def main() -> None:
    parser = argparse.ArgumentParser()
    subparsers = parser.add_subparsers(dest="command", required=True)
    publish_parser = subparsers.add_parser("publish")
    add_identity_arguments(publish_parser)
    publish_parser.add_argument("--activation-id", required=True)
    publish_parser.add_argument("--logical-backup-file", required=True)
    publish_parser.add_argument("--logical-backup-sha256", required=True)
    publish_parser.add_argument("--base-backup-file", required=True)
    publish_parser.add_argument("--base-backup-sha256", required=True)
    publish_parser.add_argument("--supersede", action="store_true")
    validate_parser = subparsers.add_parser("validate")
    add_identity_arguments(validate_parser)
    validate_chain_parser = subparsers.add_parser("validate-chain")
    validate_chain_parser.add_argument("--state-dir", required=True)
    args = parser.parse_args()
    try:
        if args.command == "publish":
            document = publish(args)
        elif args.command == "validate":
            document = validate(args)
        else:
            document = validate_chain(args)
    except (ActivationError, OSError) as exc:
        parser.error(str(exc))
    print(
        json.dumps(
            {
                "activationId": document["activationId"],
                "activatedAt": document["activatedAt"],
                "validated": True,
            },
            sort_keys=True,
        )
    )


if __name__ == "__main__":
    main()
