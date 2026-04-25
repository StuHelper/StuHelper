#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLIENTS_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

if rg -n --pcre2 'catch\s*\{' \
  "${CLIENTS_ROOT}/web" \
  "${CLIENTS_ROOT}/uniappx" \
  "${CLIENTS_ROOT}/shared" \
  "${CLIENTS_ROOT}/admin/apps/web-ele" \
  --glob '!**/node_modules/**' \
  --glob '!**/dist/**' \
  --glob '!**/storybook-static/**' \
  --glob '!**/_archived/**'; then
  echo "[check-no-empty-catch] empty catch block detected" >&2
  exit 1
fi

echo "[check-no-empty-catch] OK: no empty catch blocks found"
