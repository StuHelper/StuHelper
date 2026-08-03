#!/usr/bin/env python3
"""Validate the effective target and trigger set of a protected systemd timer."""

from __future__ import annotations

import argparse
import re


CALENDAR_RECORD = re.compile(
    r"\{\s*OnCalendar=(.*?)\s*;\s*next_elapse=[^}]*\}",
)


def parse_calendar_records(value: str) -> list[str]:
    records = [match.strip() for match in CALENDAR_RECORD.findall(value)]
    remainder = CALENDAR_RECORD.sub("", value).strip()
    if remainder:
        raise ValueError("invalid TimersCalendar record")
    return records


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--target", required=True)
    parser.add_argument("--persistent", required=True)
    parser.add_argument("--timers-calendar", required=True)
    parser.add_argument("--timers-monotonic", required=True)
    parser.add_argument("--accuracy", required=True)
    parser.add_argument("--randomized-delay", required=True)
    parser.add_argument("--fixed-random-delay", required=True)
    parser.add_argument("--expected-target", required=True)
    parser.add_argument("--expected-calendar", required=True)
    args = parser.parse_args()

    try:
        calendars = parse_calendar_records(args.timers_calendar)
    except ValueError:
        return 1

    valid = (
        args.target == args.expected_target
        and args.persistent == "yes"
        and calendars == [args.expected_calendar]
        and not args.timers_monotonic.strip()
        and args.accuracy == "1min"
        and args.randomized_delay == "0"
        and args.fixed_random_delay == "no"
    )
    return 0 if valid else 1


if __name__ == "__main__":
    raise SystemExit(main())
