#!/usr/bin/env python3
"""Validate the runtime-image inventory, scan evidence, exceptions, and VEX records."""

from __future__ import annotations

import argparse
import json
import os
import re
import sys
from dataclasses import dataclass
from datetime import date, datetime
from pathlib import Path
from typing import Any


DIGEST_REF_RE = re.compile(
    r"^[^\s@]+:[^\s/@]+@sha256:[0-9a-f]{64}$",
)
APPLICATION_DIGEST_REF_RE = re.compile(r"^[^\s@]+@sha256:[0-9a-f]{64}$")
LEGACY_APPLICATION_REF_RE = re.compile(
    r"^[A-Za-z0-9][A-Za-z0-9._-]*(?::[0-9]+)?"
    r"(?:/[A-Za-z0-9][A-Za-z0-9._-]*)*"
    r":[A-Za-z0-9_][A-Za-z0-9._-]{0,127}$",
)
MOVING_TAG_RE = re.compile(r":(?:latest|latest-dev|beta|master|nightly(?:-slim)?)@sha256:")
ROLLBACK_TAG_RE = re.compile(r"^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$")
ROLLBACK_ACTOR_RE = re.compile(r"^[A-Za-z0-9][A-Za-z0-9_.@-]{0,127}$")
AUDIT_ID_RE = re.compile(
    r"^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$",
)
ALLOWED_SCAN_SEVERITIES = {"HIGH", "CRITICAL", "UNKNOWN"}
ALLOWED_EXCEPTION_SEVERITIES = {"HIGH", "UNKNOWN"}
ALLOWED_VEX_JUSTIFICATIONS = {
    "component_not_present",
    "vulnerable_code_not_present",
    "vulnerable_code_not_in_execute_path",
    "vulnerable_code_cannot_be_controlled_by_adversary",
}


class PolicyError(Exception):
    """Raised when the policy or scan evidence violates a mandatory control."""


@dataclass(frozen=True, order=True)
class FindingKey:
    image_id: str
    vulnerability_id: str
    severity: str
    package: str
    installed_version: str


@dataclass(frozen=True)
class Finding:
    key: FindingKey
    fixed_version: str


@dataclass(frozen=True)
class RollbackReleaseEvidence:
    tag: str
    deployed_at: str
    review_date: date


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--policy",
        type=Path,
        default=Path("infra/security/runtime-images.json"),
    )
    parser.add_argument("--scan-dir", type=Path)
    parser.add_argument("--repo-root", type=Path, default=Path("."))
    parser.add_argument("--today", type=date.fromisoformat, default=date.today())
    parser.add_argument(
        "--minimum-review-days-remaining",
        type=int,
        default=0,
        help="fail when a review window has fewer than this many full days remaining",
    )
    parser.add_argument("--policy-only", action="store_true")
    parser.add_argument("--print-plan", action="store_true")
    parser.add_argument(
        "--rollback-release-record",
        type=Path,
        help=(
            "validate review windows at a previously successful deployment date; "
            "reserved for the audited production rollback path"
        ),
    )
    parser.add_argument(
        "--allow-legacy-rollback-record",
        action="store_true",
        help=(
            "validate an explicit digest transition from a legacy tag-based rollback "
            "record without mutating that record; reserved for prod-rollback.sh"
        ),
    )
    parser.add_argument(
        "--effective-environment",
        choices=("production",),
        help="also require the current process environment to match the scanned policy",
    )
    return parser.parse_args()


def require(condition: bool, message: str) -> None:
    if not condition:
        raise PolicyError(message)


def load_json(path: Path) -> Any:
    try:
        with path.open(encoding="utf-8") as handle:
            return json.load(handle)
    except (OSError, json.JSONDecodeError) as exc:
        raise PolicyError(f"cannot read valid JSON from {path}: {exc}") from exc


def parse_iso_date(value: Any, field: str) -> date:
    require(isinstance(value, str), f"{field} must be an ISO date string")
    try:
        return date.fromisoformat(value)
    except ValueError as exc:
        raise PolicyError(f"{field} must use YYYY-MM-DD: {value}") from exc


def require_text(value: Any, field: str, minimum: int = 1) -> str:
    require(isinstance(value, str), f"{field} must be a string")
    value = value.strip()
    require(len(value) >= minimum, f"{field} must contain at least {minimum} characters")
    return value


def validate_review_window(
    *,
    start_value: Any,
    end_value: Any,
    start_field: str,
    end_field: str,
    today: date,
    maximum_days: int,
    minimum_days_remaining: int,
) -> None:
    start = parse_iso_date(start_value, start_field)
    end = parse_iso_date(end_value, end_field)
    require(start <= today, f"{start_field} cannot be in the future")
    require(end >= today, f"{end_field} expired on {end.isoformat()}")
    require(
        (end - today).days >= minimum_days_remaining,
        f"{end_field} has fewer than {minimum_days_remaining} days remaining "
        f"(expires on {end.isoformat()})",
    )
    require(end >= start, f"{end_field} cannot precede {start_field}")
    require(
        (end - start).days <= maximum_days,
        f"{end_field} exceeds the {maximum_days}-day review window",
    )


def parse_env_file(path: Path) -> dict[str, list[str]]:
    values: dict[str, list[str]] = {}
    try:
        lines = path.read_text(encoding="utf-8").splitlines()
    except OSError as exc:
        raise PolicyError(f"cannot read {path}: {exc}") from exc
    for raw_line in lines:
        line = raw_line.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        key, value = line.split("=", 1)
        values.setdefault(key.strip(), []).append(value.strip())
    return values


def parse_release_record(path: Path) -> dict[str, str]:
    values: dict[str, str] = {}
    try:
        lines = path.read_text(encoding="utf-8").splitlines()
    except OSError as exc:
        raise PolicyError(f"cannot read rollback release record {path}: {exc}") from exc
    for line_number, raw_line in enumerate(lines, start=1):
        if not raw_line:
            continue
        require(
            "=" in raw_line,
            f"rollback release record {path}:{line_number} must use KEY=value",
        )
        key, value = raw_line.split("=", 1)
        require(
            re.fullmatch(r"[A-Z][A-Z0-9_]*", key) is not None,
            f"rollback release record {path}:{line_number} has an invalid key",
        )
        require(
            key not in values,
            f"rollback release record {path} contains duplicate key {key}",
        )
        require(
            "\x00" not in value and "\r" not in value and "\n" not in value,
            f"rollback release record {path}:{line_number} has an invalid value",
        )
        values[key] = value
    return values


def image_repository(reference: str) -> str:
    if "@" in reference:
        return reference.split("@", 1)[0]
    last_slash = reference.rfind("/")
    last_colon = reference.rfind(":")
    require(
        last_colon > last_slash,
        f"legacy rollback image reference must contain an explicit tag: {reference}",
    )
    return reference[:last_colon]


def validate_rollback_release_evidence(
    *,
    path: Path,
    repo_root: Path,
    today: date,
    policy_only: bool,
    effective_environment: str | None,
    minimum_days_remaining: int,
    allow_legacy_record: bool,
) -> RollbackReleaseEvidence:
    require(policy_only, "--rollback-release-record requires --policy-only")
    require(
        effective_environment == "production",
        "--rollback-release-record requires --effective-environment production",
    )
    require(
        minimum_days_remaining == 0,
        "--rollback-release-record cannot be combined with a future review-window horizon",
    )

    tag = os.environ.get("TAG", "").strip()
    rollback_tag = os.environ.get("ROLLBACK_TAG", "").strip()
    require(ROLLBACK_TAG_RE.fullmatch(tag) is not None, "TAG is invalid for audited rollback")
    require(rollback_tag == tag, "ROLLBACK_TAG must exactly match TAG for audited rollback")

    state_dir_value = os.environ.get("DEPLOY_STATE_DIR", "").strip()
    require(state_dir_value, "DEPLOY_STATE_DIR is required for audited rollback")
    state_dir = Path(state_dir_value)
    if not state_dir.is_absolute():
        state_dir = repo_root / state_dir
    expected_record = (state_dir / "releases" / f"{tag}.env").resolve()
    record_path = path if path.is_absolute() else repo_root / path
    record_path = record_path.resolve()
    require(
        record_path == expected_record,
        "rollback release record must be the target environment's exact successful-release record",
    )
    require(record_path.is_file(), f"rollback release record does not exist: {record_path}")

    values = parse_release_record(record_path)
    required_fields = {
        "TAG",
        "DEPLOYED_AT",
        "BACKEND_IMAGE_REF",
        "FRONTEND_IMAGE_REF",
        "ADMIN_IMAGE_REF",
    }
    missing_fields = sorted(required_fields - values.keys())
    require(
        not missing_fields,
        f"rollback release record is missing fields: {missing_fields}",
    )
    require(values["TAG"] == tag, "rollback release record TAG does not match target")

    deployed_at = values["DEPLOYED_AT"]
    try:
        deployed_time = datetime.strptime(deployed_at, "%Y-%m-%dT%H:%M:%SZ")
    except ValueError as exc:
        raise PolicyError(
            "rollback release record DEPLOYED_AT must use UTC YYYY-MM-DDTHH:MM:SSZ",
        ) from exc
    review_date = deployed_time.date()
    require(
        review_date <= today,
        "rollback release record DEPLOYED_AT cannot be in the future",
    )

    legacy_fields: list[str] = []
    for field in ("BACKEND_IMAGE_REF", "FRONTEND_IMAGE_REF", "ADMIN_IMAGE_REF"):
        recorded_ref = values[field].strip()
        effective_ref = os.environ.get(field, "").strip()
        if APPLICATION_DIGEST_REF_RE.fullmatch(recorded_ref) is not None:
            require(
                effective_ref == recorded_ref,
                f"{field} does not exactly match the successful rollback release record",
            )
            continue
        require(
            allow_legacy_record
            and LEGACY_APPLICATION_REF_RE.fullmatch(recorded_ref) is not None,
            f"rollback release record {field} must be an immutable digest reference",
        )
        require(
            APPLICATION_DIGEST_REF_RE.fullmatch(effective_ref) is not None,
            f"explicit legacy rollback {field} must transition to an immutable digest reference",
        )
        require(
            image_repository(recorded_ref) == image_repository(effective_ref),
            f"explicit legacy rollback {field} changes the successful release repository",
        )
        legacy_fields.append(field)

    if allow_legacy_record:
        require(
            bool(legacy_fields),
            "--allow-legacy-rollback-record requires at least one legacy image reference",
        )

    actor = os.environ.get("ROLLBACK_REVIEW_ACTOR", "").strip()
    reason = os.environ.get("ROLLBACK_REVIEW_REASON", "").strip()
    audit_id = os.environ.get("ROLLBACK_REVIEW_AUDIT_ID", "").strip()
    require(
        ROLLBACK_ACTOR_RE.fullmatch(actor) is not None,
        "ROLLBACK_REVIEW_ACTOR is required and invalid",
    )
    require(
        12 <= len(reason) <= 500 and not any(ord(char) < 32 for char in reason),
        "ROLLBACK_REVIEW_REASON must be 12-500 printable characters",
    )
    require(
        AUDIT_ID_RE.fullmatch(audit_id) is not None,
        "ROLLBACK_REVIEW_AUDIT_ID must be a UUID generated by the rollback controller",
    )

    return RollbackReleaseEvidence(
        tag=tag,
        deployed_at=deployed_at,
        review_date=review_date,
    )


def validate_policy(
    policy: Any,
    *,
    repo_root: Path,
    today: date,
    minimum_days_remaining: int,
) -> tuple[
    dict[str, dict[str, Any]],
    dict[FindingKey, Finding],
    dict[FindingKey, Finding],
]:
    require(isinstance(policy, dict), "policy root must be an object")
    require(policy.get("schema_version") == 1, "schema_version must be 1")

    controls = policy.get("controls")
    require(isinstance(controls, dict), "controls must be an object")
    max_exception_days = controls.get("max_exception_days")
    max_pin_review_days = controls.get("max_pin_review_days")
    require(
        isinstance(max_exception_days, int) and 1 <= max_exception_days <= 30,
        "max_exception_days must be between 1 and 30",
    )
    require(
        isinstance(max_pin_review_days, int) and 1 <= max_pin_review_days <= 30,
        "max_pin_review_days must be between 1 and 30",
    )
    require(
        controls.get("critical_exception_policy") == "vex-only",
        "critical_exception_policy must be vex-only",
    )

    scanner = policy.get("scanner")
    require(isinstance(scanner, dict), "scanner must be an object")
    require(
        DIGEST_REF_RE.fullmatch(require_text(scanner.get("image"), "scanner.image"))
        is not None,
        "scanner.image must be an immutable tag@sha256 reference",
    )
    severities = scanner.get("severities")
    require(
        isinstance(severities, list)
        and set(severities) == ALLOWED_SCAN_SEVERITIES
        and len(severities) == len(ALLOWED_SCAN_SEVERITIES),
        "scanner.severities must contain HIGH, CRITICAL, and UNKNOWN exactly once",
    )

    raw_images = policy.get("images")
    require(isinstance(raw_images, list) and raw_images, "images must be a non-empty array")
    images: dict[str, dict[str, Any]] = {}
    image_refs: set[str] = set()
    env_file_values: dict[str, dict[str, list[str]]] = {}

    for index, image in enumerate(raw_images):
        prefix = f"images[{index}]"
        require(isinstance(image, dict), f"{prefix} must be an object")
        image_id = require_text(image.get("id"), f"{prefix}.id")
        require(
            re.fullmatch(r"[A-Z][A-Z0-9_]*", image_id) is not None,
            f"{prefix}.id must use uppercase snake case",
        )
        require(image_id not in images, f"duplicate image id: {image_id}")
        kind = image.get("kind")
        require(kind in {"registry", "build"}, f"{prefix}.kind must be registry or build")
        image_ref = require_text(image.get("image"), f"{prefix}.image")
        require(image_ref not in image_refs, f"duplicate managed image reference: {image_ref}")
        require_text(image.get("scope"), f"{prefix}.scope")
        require_text(image.get("owner"), f"{prefix}.owner")

        if kind == "registry":
            require(
                DIGEST_REF_RE.fullmatch(image_ref) is not None,
                f"{prefix}.image must be an immutable tag@sha256 reference",
            )
            env_var = require_text(image.get("env_var"), f"{prefix}.env_var")
            env_files = image.get("env_files", [".env.example"])
            require(
                isinstance(env_files, list) and env_files,
                f"{prefix}.env_files must be a non-empty array",
            )
            require(
                len(env_files) == len(set(env_files)),
                f"{prefix}.env_files cannot contain duplicates",
            )
            require(
                ".env.example" in env_files,
                f"{prefix}.env_files must include .env.example",
            )
            for env_file_index, raw_env_file in enumerate(env_files):
                env_file = require_text(
                    raw_env_file,
                    f"{prefix}.env_files[{env_file_index}]",
                )
                env_path = Path(env_file)
                require(
                    not env_path.is_absolute() and ".." not in env_path.parts,
                    f"{prefix}.env_files[{env_file_index}] must stay inside the repository",
                )
                if env_file not in env_file_values:
                    env_file_values[env_file] = parse_env_file(repo_root / env_path)
                configured_values = env_file_values[env_file].get(env_var, [])
                require(
                    configured_values == [image_ref],
                    f"{env_var} in {env_file} must exactly match {image_ref}",
                )
            if MOVING_TAG_RE.search(image_ref):
                review = image.get("pin_review")
                require(isinstance(review, dict), f"{prefix}.pin_review is required")
                validate_review_window(
                    start_value=review.get("verified_on"),
                    end_value=review.get("review_by"),
                    start_field=f"{prefix}.pin_review.verified_on",
                    end_field=f"{prefix}.pin_review.review_by",
                    today=today,
                    maximum_days=max_pin_review_days,
                    minimum_days_remaining=minimum_days_remaining,
                )
                require_text(
                    review.get("upstream_evidence"),
                    f"{prefix}.pin_review.upstream_evidence",
                    12,
                )
        else:
            require(
                re.fullmatch(r"[a-z0-9./_-]+:[a-z0-9._-]+", image_ref) is not None,
                f"{prefix}.image must be a local scan tag",
            )
            context = repo_root / require_text(image.get("context"), f"{prefix}.context")
            dockerfile = repo_root / require_text(
                image.get("dockerfile"),
                f"{prefix}.dockerfile",
            )
            require(context.is_dir(), f"{prefix}.context does not exist: {context}")
            require(dockerfile.is_file(), f"{prefix}.dockerfile does not exist: {dockerfile}")
            build_args = image.get("build_args")
            require(
                isinstance(build_args, list) and build_args,
                f"{prefix}.build_args must be a non-empty array",
            )
            for arg_index, build_arg in enumerate(build_args):
                build_arg = require_text(
                    build_arg,
                    f"{prefix}.build_args[{arg_index}]",
                )
                require("=" in build_arg, f"{prefix}.build_args[{arg_index}] needs KEY=VALUE")
                key, value = build_arg.split("=", 1)
                require(
                    re.fullmatch(r"[A-Z][A-Z0-9_]*", key) is not None,
                    f"{prefix}.build_args[{arg_index}] has an invalid key",
                )
                if key.endswith("_IMAGE_REF"):
                    require(
                        DIGEST_REF_RE.fullmatch(value) is not None,
                        f"{prefix}.build_args[{arg_index}] must pin a digest",
                    )

        images[image_id] = image
        image_refs.add(image_ref)

    approved: dict[FindingKey, Finding] = {}
    raw_exceptions = policy.get("exceptions")
    require(isinstance(raw_exceptions, list), "exceptions must be an array")
    for group_index, group in enumerate(raw_exceptions):
        prefix = f"exceptions[{group_index}]"
        require(isinstance(group, dict), f"{prefix} must be an object")
        image_id = require_text(group.get("image_id"), f"{prefix}.image_id")
        require(image_id in images, f"{prefix} references unknown image id {image_id}")
        require_text(group.get("owner"), f"{prefix}.owner")
        require_text(group.get("rationale"), f"{prefix}.rationale", 24)
        mitigations = group.get("mitigations")
        require(
            isinstance(mitigations, list) and mitigations,
            f"{prefix}.mitigations must be non-empty",
        )
        for mitigation_index, mitigation in enumerate(mitigations):
            require_text(
                mitigation,
                f"{prefix}.mitigations[{mitigation_index}]",
                12,
            )
        validate_review_window(
            start_value=group.get("approved_on"),
            end_value=group.get("expires_on"),
            start_field=f"{prefix}.approved_on",
            end_field=f"{prefix}.expires_on",
            today=today,
            maximum_days=max_exception_days,
            minimum_days_remaining=minimum_days_remaining,
        )
        findings = group.get("findings")
        require(isinstance(findings, list) and findings, f"{prefix}.findings cannot be empty")
        for finding_index, raw_finding in enumerate(findings):
            finding_prefix = f"{prefix}.findings[{finding_index}]"
            finding = parse_policy_finding(raw_finding, image_id, finding_prefix)
            require(
                finding.key.severity in ALLOWED_EXCEPTION_SEVERITIES,
                f"{finding_prefix} cannot except severity {finding.key.severity}",
            )
            require(finding.key not in approved, f"duplicate approved finding: {finding.key}")
            approved[finding.key] = finding

    vex: dict[FindingKey, Finding] = {}
    raw_vex = policy.get("vex")
    require(isinstance(raw_vex, list), "vex must be an array")
    for vex_index, record in enumerate(raw_vex):
        prefix = f"vex[{vex_index}]"
        require(isinstance(record, dict), f"{prefix} must be an object")
        image_id = require_text(record.get("image_id"), f"{prefix}.image_id")
        require(image_id in images, f"{prefix} references unknown image id {image_id}")
        require(record.get("status") == "not_affected", f"{prefix}.status must be not_affected")
        require(
            record.get("justification") in ALLOWED_VEX_JUSTIFICATIONS,
            f"{prefix}.justification is unsupported",
        )
        require_text(record.get("owner"), f"{prefix}.owner")
        require_text(record.get("impact_statement"), f"{prefix}.impact_statement", 32)
        evidence = record.get("evidence")
        require(isinstance(evidence, list) and evidence, f"{prefix}.evidence cannot be empty")
        for evidence_index, item in enumerate(evidence):
            item = require_text(item, f"{prefix}.evidence[{evidence_index}]", 8)
            if not item.startswith("https://"):
                require(
                    (repo_root / item).exists(),
                    f"{prefix}.evidence[{evidence_index}] does not exist: {item}",
                )
        validate_review_window(
            start_value=record.get("assessed_on"),
            end_value=record.get("review_by"),
            start_field=f"{prefix}.assessed_on",
            end_field=f"{prefix}.review_by",
            today=today,
            maximum_days=max_exception_days,
            minimum_days_remaining=minimum_days_remaining,
        )
        finding = parse_policy_finding(record.get("finding"), image_id, f"{prefix}.finding")
        require(finding.key not in vex, f"duplicate VEX finding: {finding.key}")
        require(finding.key not in approved, f"finding is both excepted and VEX: {finding.key}")
        vex[finding.key] = finding

    return images, approved, vex


def parse_policy_finding(raw: Any, image_id: str, prefix: str) -> Finding:
    require(isinstance(raw, dict), f"{prefix} must be an object")
    vulnerability_id = require_text(raw.get("id"), f"{prefix}.id")
    severity = require_text(raw.get("severity"), f"{prefix}.severity")
    require(severity in ALLOWED_SCAN_SEVERITIES, f"{prefix}.severity is unsupported")
    package = require_text(raw.get("package"), f"{prefix}.package")
    installed_version = require_text(
        raw.get("installed_version"),
        f"{prefix}.installed_version",
    )
    fixed_version = raw.get("fixed_version", "")
    require(isinstance(fixed_version, str), f"{prefix}.fixed_version must be a string")
    return Finding(
        key=FindingKey(
            image_id=image_id,
            vulnerability_id=vulnerability_id,
            severity=severity,
            package=package,
            installed_version=installed_version,
        ),
        fixed_version=fixed_version,
    )


def parse_scan_findings(
    *,
    images: dict[str, dict[str, Any]],
    scan_dir: Path,
) -> dict[FindingKey, Finding]:
    require(scan_dir.is_dir(), f"scan directory does not exist: {scan_dir}")
    expected_files = {f"{image_id}.json" for image_id in images}
    actual_files = {path.name for path in scan_dir.glob("*.json")}
    require(
        actual_files == expected_files,
        "scan evidence set mismatch; "
        f"missing={sorted(expected_files - actual_files)}, "
        f"unexpected={sorted(actual_files - expected_files)}",
    )

    observed: dict[FindingKey, Finding] = {}
    for image_id, image in images.items():
        report_path = scan_dir / f"{image_id}.json"
        report = load_json(report_path)
        require(isinstance(report, dict), f"{report_path} root must be an object")
        require(
            report.get("SchemaVersion") == 2,
            f"{report_path} must use Trivy schema version 2",
        )
        if image["kind"] == "registry":
            require(
                report.get("ArtifactName") == image["image"],
                f"{report_path} scanned {report.get('ArtifactName')}, expected {image['image']}",
            )
        results = report.get("Results", [])
        require(isinstance(results, list), f"{report_path}.Results must be an array")
        for result in results:
            require(isinstance(result, dict), f"{report_path} contains an invalid result")
            vulnerabilities = result.get("Vulnerabilities", [])
            if vulnerabilities is None:
                continue
            require(
                isinstance(vulnerabilities, list),
                f"{report_path} contains invalid vulnerability data",
            )
            for raw in vulnerabilities:
                require(isinstance(raw, dict), f"{report_path} contains an invalid finding")
                severity = require_text(raw.get("Severity"), f"{report_path} Severity")
                require(
                    severity in ALLOWED_SCAN_SEVERITIES,
                    f"{report_path} contains unexpected severity {severity}",
                )
                finding = Finding(
                    key=FindingKey(
                        image_id=image_id,
                        vulnerability_id=require_text(
                            raw.get("VulnerabilityID"),
                            f"{report_path} VulnerabilityID",
                        ),
                        severity=severity,
                        package=require_text(raw.get("PkgName"), f"{report_path} PkgName"),
                        installed_version=require_text(
                            raw.get("InstalledVersion"),
                            f"{report_path} InstalledVersion",
                        ),
                    ),
                    fixed_version=str(raw.get("FixedVersion") or ""),
                )
                previous = observed.get(finding.key)
                if previous is not None:
                    require(
                        previous.fixed_version == finding.fixed_version,
                        f"{report_path} reports conflicting fixed versions for {finding.key}",
                    )
                observed[finding.key] = finding
    return observed


def print_plan(images: dict[str, dict[str, Any]]) -> None:
    for image_id, image in images.items():
        build_args = json.dumps(image.get("build_args", []), separators=(",", ":"))
        print(
            "\t".join(
                (
                    image_id,
                    image["kind"],
                    image["image"],
                    image.get("context", ""),
                    image.get("dockerfile", ""),
                    build_args,
                ),
            ),
        )


def validate_effective_environment(
    *,
    images: dict[str, dict[str, Any]],
    environment: str,
) -> None:
    template = {
        "production": ".env.prod.example",
    }[environment]
    checked = 0
    for image_id, image in images.items():
        if image["kind"] != "registry" or template not in image.get("env_files", []):
            continue
        env_var = image["env_var"]
        configured_ref = os.environ.get(env_var, "").strip()
        require(configured_ref, f"{env_var} is required in the {environment} environment")
        require(
            configured_ref == image["image"],
            f"{env_var} in the {environment} environment does not match "
            f"the scanned {image_id} policy",
        )
        checked += 1
    require(checked > 0, f"no managed images are declared for {environment}")


def validate_scan(
    *,
    observed: dict[FindingKey, Finding],
    approved: dict[FindingKey, Finding],
    vex: dict[FindingKey, Finding],
) -> None:
    failures: list[str] = []
    for key, finding in sorted(observed.items()):
        policy_finding = vex.get(key) or approved.get(key)
        if policy_finding is None:
            failures.append(f"unapproved finding: {key}")
            continue
        if policy_finding.fixed_version != finding.fixed_version:
            failures.append(
                "fixed-version drift for "
                f"{key}: policy={policy_finding.fixed_version!r}, "
                f"scan={finding.fixed_version!r}",
            )
    for key in sorted(set(approved) - set(observed)):
        failures.append(f"stale exception no longer observed: {key}")
    for key in sorted(set(vex) - set(observed)):
        failures.append(f"stale VEX record no longer observed: {key}")
    if failures:
        raise PolicyError("\n".join(failures))


def main() -> int:
    args = parse_args()
    repo_root = args.repo_root.resolve()
    require(
        0 <= args.minimum_review_days_remaining <= 30,
        "--minimum-review-days-remaining must be between 0 and 30",
    )
    rollback_release_record = args.rollback_release_record
    if rollback_release_record is None:
        env_record = os.environ.get("RUNTIME_IMAGE_ROLLBACK_RELEASE_RECORD", "").strip()
        if env_record:
            rollback_release_record = Path(env_record)

    policy_date = args.today
    rollback_evidence: RollbackReleaseEvidence | None = None
    if rollback_release_record is not None:
        rollback_evidence = validate_rollback_release_evidence(
            path=rollback_release_record,
            repo_root=repo_root,
            today=args.today,
            policy_only=args.policy_only,
            effective_environment=args.effective_environment,
            minimum_days_remaining=args.minimum_review_days_remaining,
            allow_legacy_record=args.allow_legacy_rollback_record,
        )
        policy_date = rollback_evidence.review_date
    else:
        require(
            not args.allow_legacy_rollback_record,
            "--allow-legacy-rollback-record requires --rollback-release-record",
        )

    policy_path = args.policy
    if not policy_path.is_absolute():
        policy_path = repo_root / policy_path
    policy = load_json(policy_path)
    images, approved, vex = validate_policy(
        policy,
        repo_root=repo_root,
        today=policy_date,
        minimum_days_remaining=args.minimum_review_days_remaining,
    )
    if args.effective_environment:
        validate_effective_environment(
            images=images,
            environment=args.effective_environment,
        )
    if args.print_plan:
        print_plan(images)
        return 0
    if args.policy_only:
        if rollback_evidence is not None:
            print(
                "[runtime-image-policy][warn] audited rollback is reusing review "
                f"windows valid at {rollback_evidence.deployed_at} for "
                f"release {rollback_evidence.tag}",
                file=sys.stderr,
            )
        print(
            f"[runtime-image-policy] valid: {len(images)} images, "
            f"{len(approved)} exceptions, {len(vex)} VEX records",
        )
        return 0
    require(args.scan_dir is not None, "--scan-dir is required unless --policy-only is used")
    scan_dir = args.scan_dir
    if not scan_dir.is_absolute():
        scan_dir = repo_root / scan_dir
    observed = parse_scan_findings(images=images, scan_dir=scan_dir)
    validate_scan(observed=observed, approved=approved, vex=vex)
    print(
        f"[runtime-image-scan] passed: {len(images)} images, "
        f"{len(observed)} unique findings, "
        f"{len(approved)} time-bound exceptions, {len(vex)} VEX records",
    )
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except PolicyError as exc:
        print(f"[runtime-image-scan][error] {exc}", file=sys.stderr)
        raise SystemExit(1) from exc
