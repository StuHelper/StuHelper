#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CONFIG="tools/semgrep/stuhelper-security.yml"
TEST_DIR="tools/semgrep/tests"
TEST_FIXTURE="${TEST_DIR}/no_raw_phone_log.go"
SCAN_TARGET="server/internal"
SEMGREP_IMAGE="${SEMGREP_IMAGE:-semgrep/semgrep:1.128.0}"

if command -v semgrep >/dev/null 2>&1; then
  semgrep_cmd=(semgrep)
elif command -v docker >/dev/null 2>&1; then
  semgrep_cmd=(docker run --rm -v "${ROOT_DIR}:/src" -w /src "${SEMGREP_IMAGE}" semgrep)
else
  echo "ERROR: semgrep or docker is required to run custom Semgrep rules" >&2
  exit 1
fi

cd "${ROOT_DIR}"

tmp_json="$(mktemp)"
trap 'rm -f "${tmp_json}"' EXIT

"${semgrep_cmd[@]}" scan --config="${CONFIG}" --json --quiet --no-git-ignore "${TEST_FIXTURE}" >"${tmp_json}"
python3 - "${tmp_json}" <<'PY'
import json
import pathlib
import sys

report = json.loads(pathlib.Path(sys.argv[1]).read_text())
results = [
    item
    for item in report.get("results", [])
    if str(item.get("check_id", "")).endswith("stuhelper.go.raw-phone-log-field")
]
actual_lines = sorted(item.get("start", {}).get("line") for item in results)
expected_lines = [14, 16, 18]
if actual_lines != expected_lines:
    print(
        "ERROR: custom Semgrep phone-log rule test mismatch; "
        f"expected lines {expected_lines}, got {actual_lines}",
        file=sys.stderr,
    )
    sys.exit(1)
PY

"${semgrep_cmd[@]}" scan --config="${CONFIG}" --severity ERROR --error --no-git-ignore "${SCAN_TARGET}"
