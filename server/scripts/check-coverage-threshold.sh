#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVER_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${SERVER_ROOT}"

packages=(
  "./internal/modules/auth:70"
  "./internal/modules/course:80"
  "./internal/modules/course/review:70"
  "./internal/pkg/middleware:75"
  "./internal/pkg/oidc:80"
  "./internal/pkg/fga:80"
)

failures=0

for entry in "${packages[@]}"; do
  pkg="${entry%%:*}"
  threshold="${entry##*:}"

  output="$(go test "${pkg}" -cover -count=1)"
  echo "${output}"

  coverage="$(printf '%s\n' "${output}" | awk '/coverage:/ {gsub("%","",$5); print $5}' | tail -n1)"
  if [[ -z "${coverage}" ]]; then
    echo "[check-coverage-threshold] failed to parse coverage for ${pkg}" >&2
    failures=$((failures + 1))
    continue
  fi

  if ! awk -v actual="${coverage}" -v want="${threshold}" 'BEGIN { exit !(actual + 0 >= want + 0) }'; then
    echo "[check-coverage-threshold] ${pkg} coverage ${coverage}% is below required ${threshold}%" >&2
    failures=$((failures + 1))
  else
    echo "[check-coverage-threshold] ${pkg} coverage ${coverage}% >= ${threshold}%"
  fi
done

if [[ "${failures}" -ne 0 ]]; then
  echo "[check-coverage-threshold] ${failures} package(s) below threshold" >&2
  exit 1
fi

echo "[check-coverage-threshold] all package thresholds satisfied"
