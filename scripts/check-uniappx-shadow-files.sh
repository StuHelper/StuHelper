#!/usr/bin/env bash
# CI gate: detect .ts/.js shadow-file pairs in clients/uniappx/src/
# Exits 0 if clean, exits 1 with a list of offending pairs.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SRC_DIR="$REPO_ROOT/clients/uniappx/src"

if [ ! -d "$SRC_DIR" ]; then
  echo "ERROR: $SRC_DIR does not exist" >&2
  exit 1
fi

shadow_pairs=()

while IFS= read -r ts_file; do
  js_file="${ts_file%.ts}.js"
  if [ -f "$js_file" ]; then
    rel_path="${ts_file#"$SRC_DIR"/}"
    shadow_pairs+=("$rel_path")
  fi
done < <(find "$SRC_DIR" -type f -name '*.ts')

if [ ${#shadow_pairs[@]} -eq 0 ]; then
  echo "OK: no .ts/.js shadow-file pairs found in clients/uniappx/src/"
  exit 0
fi

echo "FAIL: found ${#shadow_pairs[@]} .ts file(s) with .js shadow siblings:" >&2
for pair in "${shadow_pairs[@]}"; do
  echo "  ${pair} -> ${pair%.ts}.js" >&2
done
exit 1
