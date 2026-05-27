#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

require_cmd curl
require_cmd python3

PROMETHEUS_URL_OVERRIDE="${PROMETHEUS_URL-__STUHELPER_UNSET__}"
GRAFANA_URL_OVERRIDE="${GRAFANA_URL-__STUHELPER_UNSET__}"
LOKI_URL_OVERRIDE="${LOKI_URL-__STUHELPER_UNSET__}"
TEMPO_URL_OVERRIDE="${TEMPO_URL-__STUHELPER_UNSET__}"
ALERTMANAGER_URL_OVERRIDE="${ALERTMANAGER_URL-__STUHELPER_UNSET__}"
ALLOY_URL_OVERRIDE="${ALLOY_URL-__STUHELPER_UNSET__}"
PROMETHEUS_PORT_OVERRIDE="${PROMETHEUS_PORT-__STUHELPER_UNSET__}"
GRAFANA_PORT_OVERRIDE="${GRAFANA_PORT-__STUHELPER_UNSET__}"
LOKI_PORT_OVERRIDE="${LOKI_PORT-__STUHELPER_UNSET__}"
TEMPO_HTTP_PORT_OVERRIDE="${TEMPO_HTTP_PORT-__STUHELPER_UNSET__}"
ALERTMANAGER_PORT_OVERRIDE="${ALERTMANAGER_PORT-__STUHELPER_UNSET__}"
ALLOY_HTTP_PORT_OVERRIDE="${ALLOY_HTTP_PORT-__STUHELPER_UNSET__}"
ALERTMANAGER_WEBHOOK_URL_OVERRIDE="${ALERTMANAGER_WEBHOOK_URL-__STUHELPER_UNSET__}"
GENERATED_OBS_DIR_OVERRIDE="${GENERATED_OBS_DIR-__STUHELPER_UNSET__}"
OBSERVABILITY_SMOKE_EVIDENCE_FILE_OVERRIDE="${OBSERVABILITY_SMOKE_EVIDENCE_FILE-__STUHELPER_UNSET__}"

restore_env_override() {
  local key="$1"
  local value="$2"
  if [[ "${value}" != "__STUHELPER_UNSET__" ]]; then
    export "${key}=${value}"
  fi
}

load_env

restore_env_override "PROMETHEUS_URL" "${PROMETHEUS_URL_OVERRIDE}"
restore_env_override "GRAFANA_URL" "${GRAFANA_URL_OVERRIDE}"
restore_env_override "LOKI_URL" "${LOKI_URL_OVERRIDE}"
restore_env_override "TEMPO_URL" "${TEMPO_URL_OVERRIDE}"
restore_env_override "ALERTMANAGER_URL" "${ALERTMANAGER_URL_OVERRIDE}"
restore_env_override "ALLOY_URL" "${ALLOY_URL_OVERRIDE}"
restore_env_override "PROMETHEUS_PORT" "${PROMETHEUS_PORT_OVERRIDE}"
restore_env_override "GRAFANA_PORT" "${GRAFANA_PORT_OVERRIDE}"
restore_env_override "LOKI_PORT" "${LOKI_PORT_OVERRIDE}"
restore_env_override "TEMPO_HTTP_PORT" "${TEMPO_HTTP_PORT_OVERRIDE}"
restore_env_override "ALERTMANAGER_PORT" "${ALERTMANAGER_PORT_OVERRIDE}"
restore_env_override "ALLOY_HTTP_PORT" "${ALLOY_HTTP_PORT_OVERRIDE}"
restore_env_override "ALERTMANAGER_WEBHOOK_URL" "${ALERTMANAGER_WEBHOOK_URL_OVERRIDE}"
restore_env_override "GENERATED_OBS_DIR" "${GENERATED_OBS_DIR_OVERRIDE}"
restore_env_override "OBSERVABILITY_SMOKE_EVIDENCE_FILE" "${OBSERVABILITY_SMOKE_EVIDENCE_FILE_OVERRIDE}"

PROM_READY_URL="${PROMETHEUS_URL:-http://127.0.0.1:${PROMETHEUS_PORT:-9090}/-/ready}"
GRAFANA_HEALTH_URL="${GRAFANA_URL:-http://127.0.0.1:${GRAFANA_PORT:-3003}/api/health}"
LOKI_READY_URL="${LOKI_URL:-http://127.0.0.1:${LOKI_PORT:-3100}/ready}"
TEMPO_READY_URL="${TEMPO_URL:-http://127.0.0.1:${TEMPO_HTTP_PORT:-3200}/ready}"
ALERT_READY_URL="${ALERTMANAGER_URL:-http://127.0.0.1:${ALERTMANAGER_PORT:-9093}/-/ready}"
ALLOY_READY_URL="${ALLOY_URL:-http://127.0.0.1:${ALLOY_HTTP_PORT:-12345}/-/ready}"
OBS_SMOKE_RETRIES="${OBS_SMOKE_RETRIES:-30}"
OBS_SMOKE_SLEEP_SECONDS="${OBS_SMOKE_SLEEP_SECONDS:-2}"
OBS_SMOKE_STRICT="${OBS_SMOKE_STRICT:-false}"
OBSERVABILITY_SMOKE_EVIDENCE_FILE="${OBSERVABILITY_SMOKE_EVIDENCE_FILE:-${REPO_ROOT}/infra/generated/observability-smoke-evidence.json}"

evidence_lines="$(mktemp)"
trap 'rm -f "${evidence_lines}"' EXIT
PASS=0
FAIL=0

base_from_ready_url() {
  local url="$1"
  case "${url}" in
    */-/ready) printf '%s\n' "${url%/-/ready}" ;;
    */ready) printf '%s\n' "${url%/ready}" ;;
    */api/health) printf '%s\n' "${url%/api/health}" ;;
    *) printf '%s\n' "${url%/}" ;;
  esac
}

PROM_BASE_URL="${PROMETHEUS_BASE_URL:-$(base_from_ready_url "${PROM_READY_URL}")}"

record_check() {
  local name="$1"
  local kind="$2"
  local passed="$3"
  local detail="${4:-}"
  if [[ "${passed}" == "true" ]]; then
    PASS=$((PASS + 1))
  else
    FAIL=$((FAIL + 1))
  fi
  python3 - "${evidence_lines}" "${name}" "${kind}" "${passed}" "${detail}" <<'PY'
import json
import sys

path, name, kind, passed, detail = sys.argv[1:6]
with open(path, "a", encoding="utf-8") as handle:
    handle.write(json.dumps({
        "name": name,
        "kind": kind,
        "passed": passed == "true",
        "detail": detail,
    }, ensure_ascii=True, separators=(",", ":")) + "\n")
PY
}

check_ready() {
  local name="$1"
  local url="$2"
  local attempt

  echo "[obs-smoke] checking ${name}: ${url} (retries=${OBS_SMOKE_RETRIES})"
  for ((attempt = 1; attempt <= OBS_SMOKE_RETRIES; attempt++)); do
    if curl -fsS --max-time 10 "${url}" >/dev/null; then
      record_check "${name}" "http_ready" "true" "${url}"
      return 0
    fi
    sleep "${OBS_SMOKE_SLEEP_SECONDS}"
  done

  echo "[obs-smoke][error] ${name} did not become ready in time: ${url}" >&2
  record_check "${name}" "http_ready" "false" "${url}"
  return 1
}

urlencode() {
  python3 - "$1" <<'PY'
from urllib.parse import quote
import sys
print(quote(sys.argv[1], safe=""))
PY
}

prom_query_has_positive_sample() {
  local query="$1"
  local url="${PROM_BASE_URL}/api/v1/query?query=$(urlencode "${query}")"
  local body
  body="$(curl -fsS --max-time 10 "${url}")" || return 1
  PROM_RESPONSE="${body}" python3 <<'PY'
import json
import os
import sys

payload = json.loads(os.environ["PROM_RESPONSE"])
if payload.get("status") != "success":
    raise SystemExit(1)
for result in payload.get("data", {}).get("result", []):
    value = result.get("value", [])
    if len(value) >= 2:
        try:
            if float(value[1]) > 0:
                raise SystemExit(0)
        except ValueError:
            pass
raise SystemExit(1)
PY
}

check_prometheus_query() {
  local name="$1"
  local query="$2"
  local attempt

  echo "[obs-smoke] checking Prometheus query ${name}: ${query}"
  for ((attempt = 1; attempt <= OBS_SMOKE_RETRIES; attempt++)); do
    if prom_query_has_positive_sample "${query}"; then
      record_check "${name}" "prometheus_query" "true" "${query}"
      return 0
    fi
    sleep "${OBS_SMOKE_SLEEP_SECONDS}"
  done

  echo "[obs-smoke][error] Prometheus query did not return a positive sample: ${query}" >&2
  record_check "${name}" "prometheus_query" "false" "${query}"
  return 1
}

check_grafana_health_body() {
  local body
  if ! body="$(curl -fsS --max-time 10 "${GRAFANA_HEALTH_URL}")"; then
    record_check "Grafana health body" "json_health" "false" "${GRAFANA_HEALTH_URL}"
    return 1
  fi
  if GRAFANA_HEALTH_RESPONSE="${body}" python3 <<'PY'
import json
import os

payload = json.loads(os.environ["GRAFANA_HEALTH_RESPONSE"])
if payload.get("database") != "ok":
    raise SystemExit(1)
PY
  then
    record_check "Grafana health body" "json_health" "true" "${GRAFANA_HEALTH_URL}"
  else
    record_check "Grafana health body" "json_health" "false" "${GRAFANA_HEALTH_URL}"
    return 1
  fi
}

check_alertmanager_receiver() {
  local webhook="${ALERTMANAGER_WEBHOOK_URL:-}"
  local config_file="${GENERATED_OBS_DIR}/alertmanager/alertmanager.yml"
  if [[ -z "${webhook}" ]]; then
    record_check "Alertmanager webhook configured" "config" "false" "ALERTMANAGER_WEBHOOK_URL is empty"
    return 1
  fi
  case "${webhook}" in
    *localhost*|*127.0.0.1*|*::1*|*host.docker.internal*|*alert-webhook-sink*)
      record_check "Alertmanager webhook configured" "config" "false" "webhook points to a local sink"
      return 1
      ;;
  esac
  if [[ ! -f "${config_file}" ]]; then
    record_check "Alertmanager rendered config" "config" "false" "${config_file}"
    return 1
  fi
  if ! grep -Fq -- "${webhook}" "${config_file}"; then
    record_check "Alertmanager rendered config" "config" "false" "${config_file}"
    return 1
  fi
  local host
  host="$(python3 - "${webhook}" <<'PY'
from urllib.parse import urlsplit
import sys
print(urlsplit(sys.argv[1]).netloc)
PY
)"
  record_check "Alertmanager webhook configured" "config" "true" "host=${host}"
}

write_evidence() {
  local tmp_file
  tmp_file="$(mktemp)"
  GENERATED_AT="$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  OBS_SMOKE_STRICT_VALUE="${OBS_SMOKE_STRICT}" \
  OBS_PASS="${PASS}" \
  OBS_FAIL="${FAIL}" \
  python3 - "${evidence_lines}" <<'PY' >"${tmp_file}"
import json
import os
import sys
from pathlib import Path

checks = [
    json.loads(line)
    for line in Path(sys.argv[1]).read_text().splitlines()
    if line.strip()
]
bundle = {
    "generatedAt": os.environ["GENERATED_AT"],
    "strict": os.environ["OBS_SMOKE_STRICT_VALUE"] == "true",
    "passed": int(os.environ["OBS_FAIL"]) == 0,
    "summary": {
        "passed": int(os.environ["OBS_PASS"]),
        "failed": int(os.environ["OBS_FAIL"]),
    },
    "checks": checks,
}
print(json.dumps(bundle, ensure_ascii=True, indent=2))
PY
  if [[ "${OBSERVABILITY_SMOKE_EVIDENCE_FILE}" != "-" ]]; then
    mkdir -p "$(dirname "${OBSERVABILITY_SMOKE_EVIDENCE_FILE}")"
    install -m 600 "${tmp_file}" "${OBSERVABILITY_SMOKE_EVIDENCE_FILE}"
    echo "[obs-smoke] wrote evidence: ${OBSERVABILITY_SMOKE_EVIDENCE_FILE}"
  fi
  cat "${tmp_file}"
  rm -f "${tmp_file}"
}

check_ready Prometheus "${PROM_READY_URL}" || true
check_ready Grafana "${GRAFANA_HEALTH_URL}" || true
check_ready Loki "${LOKI_READY_URL}" || true
check_ready Tempo "${TEMPO_READY_URL}" || true
check_ready Alertmanager "${ALERT_READY_URL}" || true
check_ready Alloy "${ALLOY_READY_URL}" || true

if [[ "${OBS_SMOKE_STRICT}" == "true" ]]; then
  check_grafana_health_body || true
  check_alertmanager_receiver || true
  check_prometheus_query "Prometheus target app" 'up{job="app"}' || true
  check_prometheus_query "Prometheus target grafana" 'up{job="grafana"}' || true
  check_prometheus_query "Prometheus target alertmanager" 'up{job="alertmanager"}' || true
  check_prometheus_query "Prometheus target cadvisor" 'up{job="cadvisor"}' || true
  check_prometheus_query "Blackbox identity metadata" 'probe_success{job="blackbox-http",instance="https://id.stuhelper.com/.well-known/openid-configuration"}' || true
  if [[ "${OBS_SMOKE_CASDOOR_UPSTREAM_ENABLED:-false}" == "true" ]]; then
    check_prometheus_query "Blackbox Casdoor upstream metadata" 'probe_success{job="blackbox-http",instance="https://sso.stuhelper.com/.well-known/openid-configuration"}' || true
  fi
  check_prometheus_query "Blackbox OpenFGA TCP" 'probe_success{job="blackbox-tcp",instance="openfga:8081"}' || true
fi

write_evidence >/dev/null

if (( FAIL > 0 )); then
  echo "[obs-smoke][error] observability smoke failed: ${FAIL} failed checks" >&2
  exit 1
fi

echo "[obs-smoke] observability stack healthy"
