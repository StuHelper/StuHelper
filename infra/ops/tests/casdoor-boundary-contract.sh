#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

bash "${REPO_ROOT}/server/scripts/check-casdoor-boundary.sh"

echo "[casdoor-boundary-contract] all assertions passed"
