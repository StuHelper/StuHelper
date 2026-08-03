#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
CONFIG_FILE="${1:-${REPO_ROOT}/.github/dependabot.yml}"

[[ -f "${CONFIG_FILE}" ]] || {
  echo "[dependabot-policy][error] missing ${CONFIG_FILE}" >&2
  exit 1
}
command -v go >/dev/null 2>&1 || {
  echo "[dependabot-policy][error] Go is required to parse Dependabot YAML" >&2
  exit 1
}

(
  cd "${REPO_ROOT}/server"
  go run ./tools/dependabotpolicy --config "${CONFIG_FILE}"
)
