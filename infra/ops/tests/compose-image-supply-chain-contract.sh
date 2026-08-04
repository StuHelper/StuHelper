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
NODE_DEV_DOCKERFILE="${REPO_ROOT}/infra/images/node-dev/Dockerfile"

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
assert_contains "${BASE_COMPOSE}" 'POSTGRES_IMAGE_REF:-cgr\.dev/chainguard/postgres:latest@sha256:[0-9a-f]{64}'
assert_contains "${PARITY_COMPOSE}" 'POSTGRES_IMAGE_REF:-cgr\.dev/chainguard/postgres:latest@sha256:[0-9a-f]{64}'
assert_contains "${REMOTE_PREFLIGHT}" 'POSTGRES_IMAGE_REF:-cgr\.dev/chainguard/postgres:latest@sha256:[0-9a-f]{64}'
assert_contains "${BASE_COMPOSE}" 'entrypoint: \["/usr/local/bin/stuhelper-postgres-entrypoint"\]'
assert_contains "${BASE_COMPOSE}" 'docker-entrypoint-with-tls\.sh:/usr/local/bin/stuhelper-postgres-entrypoint:ro'
assert_contains "${BASE_COMPOSE}" '/var/lib/postgres/initdb/00-init-extra-dbs\.sh:ro'
assert_contains "${BASE_COMPOSE}" '^x-node-dev-build: &node-dev-build$'
assert_contains "${BASE_COMPOSE}" 'NODE_BASE_IMAGE_REF: \$\{NODE_DEV_BASE_IMAGE_REF:-node:24\.18\.0-alpine@sha256:[0-9a-f]{64}\}'
assert_contains "${BASE_COMPOSE}" 'BRACE_EXPANSION_VERSION: "5\.0\.9"'
assert_contains "${BASE_COMPOSE}" 'IP_ADDRESS_VERSION: "10\.3\.1"'
assert_contains "${NODE_DEV_DOCKERFILE}" '^ARG NODE_BASE_IMAGE_REF=node:24\.18\.0-alpine@sha256:[0-9a-f]{64}$'
assert_contains "${NODE_DEV_DOCKERFILE}" '^ARG NPM_TARBALL_SHA512=[A-Za-z0-9+/]+=*$'
assert_contains "${NODE_DEV_DOCKERFILE}" '^ARG BRACE_EXPANSION_VERSION=5\.0\.9$'
assert_contains "${NODE_DEV_DOCKERFILE}" '^ARG BRACE_EXPANSION_TARBALL_SHA512=[A-Za-z0-9+/]+=*$'
assert_contains "${NODE_DEV_DOCKERFILE}" '^ARG IP_ADDRESS_VERSION=10\.3\.1$'
assert_contains "${NODE_DEV_DOCKERFILE}" '^ARG IP_ADDRESS_TARBALL_SHA512=[A-Za-z0-9+/]+=*$'
assert_contains "${NODE_DEV_DOCKERFILE}" 'sha512sum "\$\{ip_address_tarball\}"'

mailpit_block="$(service_block "${BASE_COMPOSE}" mailpit)"
casdoor_block="$(service_block "${BASE_COMPOSE}" casdoor)"
[[ "${mailpit_block}" == *"profiles: [dev-full]"* ]] ||
  fail "Mailpit must not start in the production profile"
[[ "${casdoor_block}" == *"profiles: [dev-full]"* ]] ||
  fail "the repository-local Casdoor must not start in the production profile"
[[ "${casdoor_block}" == *'image: ${CASDOOR_IMAGE_REF:-casbin/casdoor:latest@sha256:53f00e1e69190c8629925a179cd59a10573f7be2764038571906196908cd6579}'* ]] ||
  fail "the repository-local Casdoor fallback must match the reviewed runtime-image policy"

for image_var in \
  POSTGRES_IMAGE_REF \
  REDIS_IMAGE_REF \
  MAILPIT_IMAGE_REF \
  DEV_OBJECT_STORAGE_IMAGE_REF \
  RCLONE_IMAGE_REF \
  GOLANG_IMAGE_REF \
  NODE_DEV_BASE_IMAGE_REF \
  CASDOOR_IMAGE_REF \
  OPENFGA_IMAGE_REF \
  DOCKER_SOCKET_PROXY_IMAGE_REF \
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
