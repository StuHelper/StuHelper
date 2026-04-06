#!/usr/bin/env bash
set -euo pipefail

API_BASE_URL="${API_BASE_URL:-http://127.0.0.1:8080}"
WEB_BASE_URL="${WEB_BASE_URL:-http://127.0.0.1:3000}"
ADMIN_BASE_URL="${ADMIN_BASE_URL:-http://127.0.0.1:${ADMIN_EXTERNAL_PORT:-3001}}"
ADMIN_SMOKE_PATH="${ADMIN_SMOKE_PATH:-/admin/}"
CHECK_ADMIN="${CHECK_ADMIN:-true}"
SMOKE_RETRIES="${SMOKE_RETRIES:-30}"
SMOKE_SLEEP_SECONDS="${SMOKE_SLEEP_SECONDS:-2}"

curl_check() {
  local name="$1"
  local url="$2"
  local retries="${3:-$SMOKE_RETRIES}"
  local sleep_seconds="${4:-$SMOKE_SLEEP_SECONDS}"
  local attempt

  echo "[smoke] checking ${name}: ${url} (retries=${retries})"
  for ((attempt = 1; attempt <= retries; attempt++)); do
    if curl --fail --silent --show-error --location --max-time 10 "$url" >/dev/null; then
      return 0
    fi
    sleep "${sleep_seconds}"
  done

  echo "[smoke][error] ${name} did not become ready in time: ${url}" >&2
  return 1
}

curl_check "api live" "${API_BASE_URL}/health/live"
curl_check "api ready" "${API_BASE_URL}/health/ready"
curl_check "web" "${WEB_BASE_URL}/"

if [[ "$CHECK_ADMIN" == "true" ]]; then
  curl_check "admin" "${ADMIN_BASE_URL}${ADMIN_SMOKE_PATH}"
fi

echo "[smoke] all checks passed"
