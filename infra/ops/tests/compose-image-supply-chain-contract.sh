#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
BASE_COMPOSE="${REPO_ROOT}/docker-compose.yml"
OBS_COMPOSE="${REPO_ROOT}/docker-compose.observability.yml"
PARITY_COMPOSE="${REPO_ROOT}/docker-compose.prod-parity-postgres.yml"
PROD_COMPOSE="${REPO_ROOT}/docker-compose.prod.yml"
ENV_EXAMPLE="${REPO_ROOT}/.env.example"
REMOTE_PREFLIGHT="${REPO_ROOT}/infra/ops/remote-preflight.sh"

fail() {
  printf '[compose-image-supply-chain-contract][error] %s\n' "$*" >&2
  exit 1
}

assert_contains() {
  local file="$1"
  local pattern="$2"
  grep -Eq -- "${pattern}" "${file}" ||
    fail "expected ${file} to contain pattern: ${pattern}"
}

assert_not_contains() {
  local file="$1"
  local pattern="$2"
  if grep -Eq -- "${pattern}" "${file}"; then
    fail "expected ${file} not to contain pattern: ${pattern}"
  fi
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

for compose_file in "${BASE_COMPOSE}" "${OBS_COMPOSE}" "${PARITY_COMPOSE}" "${PROD_COMPOSE}"; do
  while IFS= read -r image_line; do
    if [[ "${image_line}" == *':?'*'IMAGE_REF is required'* ]]; then
      continue
    fi
    [[ "${image_line}" =~ @sha256:[0-9a-f]{64} ]] ||
      fail "mutable Compose image reference in ${compose_file}: ${image_line}"
  done < <(grep -E '^[[:space:]]+image:' "${compose_file}")
done

assert_not_contains "${BASE_COMPOSE}" 'image:.*_VERSION'
assert_not_contains "${OBS_COMPOSE}" 'image:.*_VERSION'
assert_contains "${BASE_COMPOSE}" 'POSTGRES_IMAGE_REF:-postgres:18\.3-alpine@sha256:[0-9a-f]{64}'
assert_contains "${PARITY_COMPOSE}" 'POSTGRES_IMAGE_REF:-postgres:18\.3-alpine@sha256:[0-9a-f]{64}'
assert_contains "${REMOTE_PREFLIGHT}" 'POSTGRES_IMAGE_REF:-postgres:18\.3-alpine@sha256:[0-9a-f]{64}'

mailpit_block="$(service_block "${BASE_COMPOSE}" mailpit)"
casdoor_block="$(service_block "${BASE_COMPOSE}" casdoor)"
[[ "${mailpit_block}" == *"profiles: [dev-full]"* ]] ||
  fail "Mailpit must not start in the production profile"
[[ "${casdoor_block}" == *"profiles: [dev-full]"* ]] ||
  fail "the repository-local Casdoor must not start in the production profile"

for image_var in \
  POSTGRES_IMAGE_REF \
  REDIS_IMAGE_REF \
  MAILPIT_IMAGE_REF \
  MINIO_IMAGE_REF \
  MINIO_MC_IMAGE_REF \
  GOLANG_IMAGE_REF \
  NODE_ALPINE_IMAGE_REF \
  NODE_BOOKWORM_IMAGE_REF \
  CASDOOR_IMAGE_REF \
  OPENFGA_IMAGE_REF \
  GRAFANA_ALLOY_IMAGE_REF \
  PROMETHEUS_IMAGE_REF \
  ALERTMANAGER_IMAGE_REF \
  LOKI_IMAGE_REF \
  TEMPO_IMAGE_REF \
  GRAFANA_IMAGE_REF \
  NODE_EXPORTER_IMAGE_REF \
  CADVISOR_IMAGE_REF \
  POSTGRES_EXPORTER_IMAGE_REF \
  REDIS_EXPORTER_IMAGE_REF \
  BLACKBOX_EXPORTER_IMAGE_REF \
  ALERT_WEBHOOK_SINK_IMAGE_REF; do
  assert_contains "${ENV_EXAMPLE}" "^${image_var}=.*@sha256:[0-9a-f]{64}$"
done

printf '[compose-image-supply-chain-contract] all assertions passed\n'
