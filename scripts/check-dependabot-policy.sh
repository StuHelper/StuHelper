#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
CONFIG_FILE="${1:-${REPO_ROOT}/.github/dependabot.yml}"

[[ -f "${CONFIG_FILE}" ]] || {
  echo "[dependabot-policy][error] missing ${CONFIG_FILE}" >&2
  exit 1
}

awk '
  function finish_update() {
    if (in_update && !targets_develop) {
      printf "[dependabot-policy][error] %s must set target-branch: develop\n", ecosystem > "/dev/stderr"
      invalid = 1
    }
  }

  /^  - package-ecosystem: / {
    finish_update()
    in_update = 1
    targets_develop = 0
    ecosystem = $0
    sub(/^  - package-ecosystem: /, "", ecosystem)
    next
  }

  /^    target-branch: develop$/ && in_update {
    targets_develop = 1
  }

  END {
    finish_update()
    exit invalid
  }
' "${CONFIG_FILE}"

echo "[dependabot-policy] all version-update ecosystems target develop"
