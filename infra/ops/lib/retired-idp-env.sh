#!/usr/bin/env bash

remove_env_key_prefixes_from_file() {
  local file="$1"
  shift

  [[ -f "${file}" ]] || return 0
  python3 - "$file" "$@" <<'PY'
from pathlib import Path
import sys

path = Path(sys.argv[1])
prefixes = tuple(sys.argv[2:])
lines = path.read_text().splitlines()
kept = []

for line in lines:
    key = line.split("=", 1)[0]
    if prefixes and key.startswith(prefixes):
        continue
    kept.append(line)

path.write_text("\n".join(kept) + ("\n" if kept else ""))
PY
}

remove_retired_idp_env_files() {
  local retired_prefix="ZITA""DEL_"
  local login_client_pat="LOGIN_CLIENT_""PAT_EXPIRATION"
  local file

  for file in "$@"; do
    [[ -n "${file}" ]] || continue
    remove_env_key_prefixes_from_file "${file}" "${retired_prefix}" "${login_client_pat}"
  done
}
