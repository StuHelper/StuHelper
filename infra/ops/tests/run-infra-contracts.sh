#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

for test_script in "${SCRIPT_DIR}"/*.sh; do
  [[ -e "${test_script}" ]] || continue
  if [[ "$(basename "${test_script}")" == "run-infra-contracts.sh" ]]; then
    continue
  fi
  echo "[infra-contract] ${test_script}"
  bash "${test_script}"
done

for test_script in "${SCRIPT_DIR}"/*.mjs; do
  [[ -e "${test_script}" ]] || continue
  echo "[infra-contract] ${test_script}"
  node "${test_script}"
done
