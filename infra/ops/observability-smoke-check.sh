#!/usr/bin/env bash
set -euo pipefail

PROM_URL="${PROMETHEUS_URL:-http://127.0.0.1:9090/-/ready}"
GRAFANA_URL="${GRAFANA_URL:-http://127.0.0.1:3003/api/health}"
LOKI_URL="${LOKI_URL:-http://127.0.0.1:3100/ready}"
TEMPO_URL="${TEMPO_URL:-http://127.0.0.1:3200/ready}"
ALERT_URL="${ALERTMANAGER_URL:-http://127.0.0.1:9093/-/ready}"
ALLOY_URL="${ALLOY_URL:-http://127.0.0.1:12345/-/ready}"
OBS_SMOKE_RETRIES="${OBS_SMOKE_RETRIES:-30}"
OBS_SMOKE_SLEEP_SECONDS="${OBS_SMOKE_SLEEP_SECONDS:-2}"

check() {
  local name="$1"
  local url="$2"
  local retries="${3:-$OBS_SMOKE_RETRIES}"
  local sleep_seconds="${4:-$OBS_SMOKE_SLEEP_SECONDS}"
  local attempt

  echo "[obs-smoke] checking ${name}: ${url} (retries=${retries})"
  for ((attempt = 1; attempt <= retries; attempt++)); do
    if curl -fsS "$url" >/dev/null; then
      return 0
    fi
    sleep "${sleep_seconds}"
  done

  echo "[obs-smoke][error] ${name} did not become ready in time: ${url}" >&2
  return 1
}

check Prometheus "$PROM_URL"
check Grafana "$GRAFANA_URL"
check Loki "$LOKI_URL"
check Tempo "$TEMPO_URL"
check Alertmanager "$ALERT_URL"
check Alloy "$ALLOY_URL"

echo "[obs-smoke] observability stack healthy"
