#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

python3 - "$REPO_ROOT" <<'PY'
from __future__ import annotations

import json
import re
import sys
from pathlib import Path


def fail(message: str) -> None:
    print(f"[openfga-model-contract][error] {message}", file=sys.stderr)
    raise SystemExit(1)


def strip_comment(line: str) -> str:
    return line.split("#", 1)[0].strip()


def parse_fga(path: Path) -> dict[str, set[str]]:
    if not path.is_file():
        fail(f"missing file: {path}")

    current_type: str | None = None
    relations_by_type: dict[str, set[str]] = {}
    for line_no, raw_line in enumerate(path.read_text(encoding="utf-8").splitlines(), start=1):
        line = strip_comment(raw_line)
        if not line:
            continue
        if re.search(r"\bfrom\s+\w+\s+from\b", line):
            fail(f"{path}:{line_no} contains chained tuple-to-userset: {line}")
        type_match = re.fullmatch(r"type\s+(\w+)", line)
        if type_match:
            current_type = type_match.group(1)
            relations_by_type.setdefault(current_type, set())
            continue
        relation_match = re.fullmatch(r"define\s+(\w+):\s+(.+)", line)
        if relation_match:
            if current_type is None:
                fail(f"{path}:{line_no} defines relation outside a type")
            relations_by_type[current_type].add(relation_match.group(1))

    return relations_by_type


def load_json_types(path: Path) -> dict[str, set[str]]:
    if not path.is_file():
        fail(f"missing file: {path}")

    data = json.loads(path.read_text(encoding="utf-8"))
    if data.get("schema_version") != "1.1":
        fail(f"{path} must use schema_version 1.1")

    result: dict[str, set[str]] = {}
    for type_def in data.get("type_definitions", []):
        type_name = type_def.get("type")
        if not type_name:
            fail(f"{path} contains a type definition without type")
        result[type_name] = set(type_def.get("relations", {}).keys())
    return result


def load_json(path: Path) -> dict:
    return json.loads(path.read_text(encoding="utf-8"))


def assert_relation_sets_match(label: str, left: dict[str, set[str]], right: dict[str, set[str]]) -> None:
    if left == right:
        return
    left_only = {key: sorted(left[key] - right.get(key, set())) for key in left}
    right_only = {key: sorted(right[key] - left.get(key, set())) for key in right}
    left_only = {key: value for key, value in left_only.items() if value or key not in right}
    right_only = {key: value for key, value in right_only.items() if value or key not in left}
    fail(f"{label} relation set drift: only_left={left_only}; only_right={right_only}")


def assert_direct_tuplesets(model_path: Path, model: dict) -> None:
    for type_def in model.get("type_definitions", []):
        type_name = type_def["type"]
        relations = type_def.get("relations", {})
        direct_relations = {
            name for name, definition in relations.items()
            if isinstance(definition, dict) and "this" in definition
        }
        for relation_name, definition in relations.items():
            for tupleset in find_tuplesets(definition):
                if tupleset not in direct_relations:
                    fail(
                        f"{model_path}: type {type_name}.{relation_name} uses tupleset "
                        f"{tupleset!r}, but it is not a direct relation on {type_name}"
                    )


def find_tuplesets(value: object) -> list[str]:
    result: list[str] = []
    if isinstance(value, dict):
        tuple_to_userset = value.get("tupleToUserset")
        if isinstance(tuple_to_userset, dict):
            tupleset = tuple_to_userset.get("tupleset", {})
            relation = tupleset.get("relation")
            if isinstance(relation, str):
                result.append(relation)
        for child in value.values():
            result.extend(find_tuplesets(child))
    elif isinstance(value, list):
        for child in value:
            result.extend(find_tuplesets(child))
    return result


repo_root = Path(sys.argv[1])
infra_fga = repo_root / "infra/openfga/model.fga"
design_fga = repo_root / "docs/design/openfga-model.fga"
model_json = repo_root / "infra/openfga/model.json"

infra_relations = parse_fga(infra_fga)
design_relations = parse_fga(design_fga)
json_relations = load_json_types(model_json)

assert_relation_sets_match("infra/openfga/model.fga vs docs/design/openfga-model.fga", infra_relations, design_relations)
assert_relation_sets_match("infra/openfga/model.fga vs infra/openfga/model.json", infra_relations, json_relations)
assert_direct_tuplesets(model_json, load_json(model_json))

print("[openfga-model-contract] all assertions passed")
PY
