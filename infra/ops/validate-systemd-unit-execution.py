#!/usr/bin/env python3
"""Validate the effective execution record of a protected systemd unit."""

from __future__ import annotations

import argparse
import shlex


SAFE_PATH = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"


def parse_record(record: str) -> dict[str, str]:
    if not record.startswith("{ ") or not record.endswith(" }"):
        raise ValueError("invalid systemd execution record")

    inner = record[2:-2]
    if inner.endswith(" ;"):
        inner = inner[:-2]

    fields: dict[str, str] = {}
    for segment in inner.split(" ; "):
        if "=" not in segment:
            raise ValueError("invalid systemd execution field")
        key, value = segment.split("=", 1)
        if key in fields:
            raise ValueError("duplicate systemd execution field")
        fields[key] = value
    return fields


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--expected-working-directory", required=True)
    parser.add_argument("--expected-command", required=True)
    parser.add_argument("--actual-working-directory", required=True)
    parser.add_argument("--exec-start", required=True)
    parser.add_argument("--exec-start-ex", required=True)
    parser.add_argument("--expected-environment", action="append", default=[])
    args = parser.parse_args()

    if args.actual_working_directory != args.expected_working_directory:
        return 1

    try:
        exec_fields = parse_record(args.exec_start)
        exec_ex_fields = parse_record(args.exec_start_ex)
        command = shlex.split(args.expected_command)
        environment_keys = [assignment.split("=", 1)[0] for assignment in args.expected_environment]
    except ValueError:
        return 1
    if any("=" not in assignment for assignment in args.expected_environment):
        return 1
    if "PATH" in environment_keys or len(environment_keys) != len(set(environment_keys)):
        return 1

    expected_argv = [
        "/usr/bin/env",
        "-i",
        f"PATH={SAFE_PATH}",
        *args.expected_environment,
        "/bin/bash",
        "--noprofile",
        "--norc",
        *command,
    ]
    valid = (
        exec_fields.get("path") == "/usr/bin/env"
        and exec_fields.get("argv[]", "").split() == expected_argv
        and exec_fields.get("ignore_errors") == "no"
        and exec_ex_fields.get("path") == "/usr/bin/env"
        and exec_ex_fields.get("argv[]", "").split() == expected_argv
        and exec_ex_fields.get("flags") == ""
    )
    return 0 if valid else 1


if __name__ == "__main__":
    raise SystemExit(main())
