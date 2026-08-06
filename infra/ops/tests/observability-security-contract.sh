#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
BASE_COMPOSE="${REPO_ROOT}/docker-compose.yml"
OBS_COMPOSE="${REPO_ROOT}/docker-compose.observability.yml"
ALLOY_CONFIG="${REPO_ROOT}/infra/observability/alloy/config.alloy"
TEMPO_CONFIG="${REPO_ROOT}/infra/observability/tempo/tempo.yaml"

fail() {
  printf '[observability-security-contract][error] %s\n' "$*" >&2
  exit 1
}

service_block() {
  local file="$1"
  local service="$2"
  awk -v service="${service}" '
    $0 == "  " service ":" { in_service=1 }
    in_service && $0 ~ /^  [A-Za-z0-9_-]+:$/ && $0 != "  " service ":" { exit }
    in_service { print }
  ' "${file}"
}

assert_block_contains() {
  local block="$1"
  local expected="$2"
  grep -Fq -- "${expected}" <<<"${block}" ||
    fail "service block is missing: ${expected}"
}

assert_block_not_contains() {
  local block="$1"
  local forbidden="$2"
  if grep -Fq -- "${forbidden}" <<<"${block}"; then
    fail "service block must not contain: ${forbidden}"
  fi
}

proxy_block="$(service_block "${OBS_COMPOSE}" docker-socket-proxy)"
alloy_block="$(service_block "${OBS_COMPOSE}" alloy)"
prometheus_block="$(service_block "${OBS_COMPOSE}" prometheus)"
alertmanager_block="$(service_block "${OBS_COMPOSE}" alertmanager)"
blackbox_block="$(service_block "${OBS_COMPOSE}" blackbox-exporter)"
tempo_block="$(service_block "${OBS_COMPOSE}" tempo)"
grafana_block="$(service_block "${OBS_COMPOSE}" grafana)"

assert_block_contains "${proxy_block}" 'POST: "0"'
assert_block_contains "${proxy_block}" '/var/run/docker.sock:/var/run/docker.sock:ro'
assert_block_contains "${proxy_block}" '- docker_api'
assert_block_contains "${proxy_block}" 'read_only: true'
assert_block_contains "${proxy_block}" 'cap_drop:'
assert_block_not_contains "${proxy_block}" '    ports:'

grep -A5 '^  docker_api:' "${BASE_COMPOSE}" | grep -Fq 'internal: true' ||
  fail "docker_api network must remain internal"

assert_block_contains "${alloy_block}" 'user: "473:473"'
assert_block_contains "${alloy_block}" 'docker-socket-proxy:'
assert_block_contains "${alloy_block}" 'ALLOY_DOCKER_LOG_MAX_AGE:'
assert_block_contains "${alloy_block}" 'ALLOY_DOCKER_STREAM_TIMEOUT:'
assert_block_contains "${alloy_block}" '--disable-reporting'
assert_block_contains "${alloy_block}" 'read_only: true'
assert_block_contains "${alloy_block}" 'cap_drop:'
assert_block_not_contains "${alloy_block}" '/var/run/docker.sock'
assert_block_not_contains "${alloy_block}" '/var/lib/docker/containers'

# Prometheus scrapes app:8080 directly, so it must share the backend network
# with the application while retaining the isolated observability network for
# exporters and the rest of the telemetry stack.
assert_block_contains "${prometheus_block}" '- backend'
assert_block_contains "${prometheus_block}" '- observability'

assert_block_contains "${alertmanager_block}" 'webhook-token:/etc/alertmanager/secrets/webhook-token:ro'
assert_block_contains "${alertmanager_block}" '"${ALERTMANAGER_CONFIG_GID:-65534}"'
assert_block_not_contains "${alertmanager_block}" 'ALERTMANAGER_WEBHOOK_TOKEN:'

# Blackbox executes the probes for app/frontend/admin/OpenFGA itself; reaching
# only the Prometheus-facing observability network is insufficient.
assert_block_contains "${blackbox_block}" '- backend'
assert_block_contains "${blackbox_block}" '- observability'

grep -Fq '__meta_docker_container_label_com_docker_compose_project' "${ALLOY_CONFIG}" ||
  fail "Alloy must filter discovery by Compose project"
grep -Fq 'regex         = "(alloy|loki)"' "${ALLOY_CONFIG}" ||
  fail "Alloy must exclude its recursive log path"
grep -Fq 'host             = "http://docker-socket-proxy:2375"' "${ALLOY_CONFIG}" ||
  fail "Alloy log tailing must use the restricted Docker proxy"
grep -Fq 'older_than          = sys.env("ALLOY_DOCKER_LOG_MAX_AGE")' "${ALLOY_CONFIG}" ||
  fail "Alloy must bound Docker backlog replay"
grep -Fq 'refresh_interval = sys.env("ALLOY_DOCKER_STREAM_TIMEOUT")' "${ALLOY_CONFIG}" ||
  fail "Alloy Docker stream timeout must be explicit"

assert_block_contains "${tempo_block}" 'test: ["CMD", "/tempo", "-health"'
if grep -Eq '^(ingester|compactor):' "${TEMPO_CONFIG}"; then
  fail "Tempo 3 config must not restore removed top-level ingester/compactor blocks"
fi
grep -Fq 'backend_scheduler:' "${TEMPO_CONFIG}" ||
  fail "Tempo 3 retention scheduler configuration is missing"
grep -Fq 'backend_worker:' "${TEMPO_CONFIG}" ||
  fail "Tempo 3 retention worker configuration is missing"

for setting in \
  'GF_SECURITY_DISABLE_GRAVATAR: "true"' \
  'GF_ANALYTICS_REPORTING_ENABLED: "false"' \
  'GF_ANALYTICS_CHECK_FOR_UPDATES: "false"' \
  'GF_ANALYTICS_CHECK_FOR_PLUGIN_UPDATES: "false"' \
  'GF_NEWS_NEWS_FEED_ENABLED: "false"' \
  'GF_SERVER_SERVE_FROM_SUB_PATH: "true"'; do
  assert_block_contains "${grafana_block}" "${setting}"
done

for service in alloy prometheus alertmanager loki tempo grafana; do
  block="$(service_block "${OBS_COMPOSE}" "${service}")"
  assert_block_contains "${block}" 'read_only: true'
  assert_block_contains "${block}" 'cap_drop:'
  assert_block_contains "${block}" 'no-new-privileges:true'
done

printf '[observability-security-contract] all assertions passed\n'
