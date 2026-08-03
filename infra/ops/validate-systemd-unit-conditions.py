#!/usr/bin/env python3
"""Reject effective systemd Conditions/Asserts on protected backup units."""

from __future__ import annotations

import argparse
import json


def is_empty_condition_array(raw: str) -> bool:
    try:
        document = json.loads(raw)
    except json.JSONDecodeError:
        return False
    return document == {"type": "a(sbbsi)", "data": []}


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--conditions-json", required=True)
    parser.add_argument("--asserts-json", required=True)
    args = parser.parse_args()

    valid = is_empty_condition_array(args.conditions_json) and is_empty_condition_array(
        args.asserts_json,
    )
    return 0 if valid else 1


if __name__ == "__main__":
    raise SystemExit(main())
