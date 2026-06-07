#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/dev-local.sh
source "${SCRIPT_DIR}/lib/dev-local.sh"

ensure_dev_runtime_dirs

tail -n "${TAIL_LINES:-80}" -f \
  "$(log_file backend)" \
  "$(log_file frontend)" \
  "$(log_file admin)" \
  "$(log_file koishi)"
