#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"
# shellcheck source=lib/dev-local.sh
source "${SCRIPT_DIR}/lib/dev-local.sh"

"${SCRIPT_DIR}/init-dev-env.sh"
if [[ -f "${DEV_RUNTIME_ENV}" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "${DEV_RUNTIME_ENV}"
  set +a
fi

"${SCRIPT_DIR}/smoke-check.sh"
