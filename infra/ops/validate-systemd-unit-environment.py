#!/usr/bin/env python3
"""Validate the explicit and pre-exec-cleared environment of a backup unit."""

from __future__ import annotations

import argparse
import shlex


PRE_EXEC_UNSET_KEYS = (
    "LD_PRELOAD",
    "LD_LIBRARY_PATH",
    "LD_AUDIT",
    "GCONV_PATH",
    "LOCPATH",
)


def assignments_to_environment(assignments: list[str]) -> dict[str, str]:
    environment: dict[str, str] = {}
    for assignment in assignments:
        if "=" not in assignment:
            raise ValueError("invalid environment assignment")
        key, value = assignment.split("=", 1)
        if not key or key in environment:
            raise ValueError("invalid or duplicate environment key")
        environment[key] = value
    return environment


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--environment", required=True)
    parser.add_argument("--unset-environment", required=True)
    parser.add_argument("--expected-environment", action="append", default=[])
    args = parser.parse_args()

    try:
        actual = assignments_to_environment(shlex.split(args.environment))
        expected = assignments_to_environment(args.expected_environment)
        actual_unset = shlex.split(args.unset_environment)
    except ValueError:
        return 1

    unset_is_exact = (
        len(actual_unset) == len(PRE_EXEC_UNSET_KEYS)
        and set(actual_unset) == set(PRE_EXEC_UNSET_KEYS)
    )
    return 0 if actual == expected and unset_is_exact else 1


if __name__ == "__main__":
    raise SystemExit(main())
