#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
POLICY="${REPO_ROOT}/infra/security/runtime-images.json"
VALIDATOR="${REPO_ROOT}/infra/ops/validate-runtime-image-scan.py"
SCANNER="${REPO_ROOT}/infra/ops/scan-runtime-images.sh"
NODE_DOCKERFILE="${REPO_ROOT}/infra/images/node-dev/Dockerfile"
BASE_COMPOSE="${REPO_ROOT}/docker-compose.yml"
POSTGRES_INIT="${REPO_ROOT}/infra/postgres/init-extra-dbs.sh"
GITHUB_CI="${REPO_ROOT}/.github/workflows/ci.yml"
PROD_ENV_EXAMPLE="${REPO_ROOT}/.env.prod.example"
PROD_DEPLOY="${REPO_ROOT}/infra/ops/prod-deploy.sh"
REMOTE_PREFLIGHT="${REPO_ROOT}/infra/ops/remote-preflight.sh"

fail() {
  printf '[runtime-image-security-contract][error] %s\n' "$*" >&2
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

github_job_block() {
  local job="$1"
  awk -v job="${job}" '
    $0 == "  " job ":" { in_job=1 }
    in_job && $0 ~ /^  [A-Za-z0-9_-]+:$/ && $0 != "  " job ":" { exit }
    in_job { print }
  ' "${GITHUB_CI}"
}

python3 "${VALIDATOR}" \
  --repo-root "${REPO_ROOT}" \
  --policy "${POLICY}" \
  --policy-only

while IFS='=' read -r key value; do
  case "${key}" in
    *_IMAGE_REF) export "${key}=${value}" ;;
  esac
done < "${PROD_ENV_EXAMPLE}"

python3 "${VALIDATOR}" \
  --repo-root "${REPO_ROOT}" \
  --policy "${POLICY}" \
  --policy-only \
  --effective-environment production

if POSTGRES_IMAGE_REF="postgres:unscanned@sha256:0000000000000000000000000000000000000000000000000000000000000000" \
  python3 "${VALIDATOR}" \
    --repo-root "${REPO_ROOT}" \
    --policy "${POLICY}" \
    --policy-only \
    --effective-environment production >/dev/null 2>&1; then
  fail "the effective production environment must reject an unscanned image override"
fi

assert_contains "${SCANNER}" '^set -euo pipefail$'
assert_contains "${SCANNER}" '--download-db-only'
assert_contains "${SCANNER}" '--skip-db-update'
assert_contains "${SCANNER}" '--scanners vuln'
assert_contains "${SCANNER}" '--severity HIGH,CRITICAL,UNKNOWN'
assert_contains "${SCANNER}" '--ignore-unfixed=false'
assert_contains "${SCANNER}" '--format json'
assert_contains "${SCANNER}" 'mktemp -d /tmp/stuhelper-trivy-cache\.XXXXXX'
assert_contains "${SCANNER}" 'Trivy cache is not writable by uid'
assert_not_contains "${SCANNER}" '/var/run/docker\.sock'
assert_not_contains "${SCANNER}" '--ignore-unfixed=true'

assert_contains "${NODE_DOCKERFILE}" '^ARG NODE_BASE_IMAGE_REF=node:24\.18\.0-alpine@sha256:[0-9a-f]{64}$'
assert_contains "${NODE_DOCKERFILE}" '^ARG NPM_VERSION=11\.18\.0$'
assert_contains "${NODE_DOCKERFILE}" '^ARG NPM_TARBALL_SHA512=[A-Za-z0-9+/]+=*$'
assert_contains "${NODE_DOCKERFILE}" '^ARG BRACE_EXPANSION_VERSION=5\.0\.8$'
assert_contains "${NODE_DOCKERFILE}" '^ARG BRACE_EXPANSION_TARBALL_SHA512=[A-Za-z0-9+/]+=*$'
assert_contains "${NODE_DOCKERFILE}" 'sha512sum "\$\{npm_tarball\}"'
assert_contains "${NODE_DOCKERFILE}" 'sha512sum "\$\{brace_expansion_tarball\}"'
assert_contains "${BASE_COMPOSE}" '^x-node-dev-build: &node-dev-build$'
assert_contains "${BASE_COMPOSE}" 'NODE_BASE_IMAGE_REF: \$\{NODE_DEV_BASE_IMAGE_REF:-node:24\.18\.0-alpine@sha256:[0-9a-f]{64}\}'
[[ "$(grep -Ec '^[[:space:]]+build: \*node-dev-build$' "${BASE_COMPOSE}")" -eq 2 ]] ||
  fail "frontend-dev and admin-dev must both use the hardened node-dev build"

assert_contains "${POSTGRES_INIT}" '^set -euo pipefail$'
assert_contains "${POSTGRES_INIT}" "<<-'EOSQL'"
assert_contains "${POSTGRES_INIT}" '\\getenv stuhelper_app_password STUHELPER_APP_DB_PASSWORD'
assert_contains "${POSTGRES_INIT}" "PASSWORD :'stuhelper_app_password'"
assert_contains "${POSTGRES_INIT}" "format\\('GRANT CONNECT ON DATABASE %I TO stuhelper_app'"
assert_not_contains "${POSTGRES_INIT}" 'PASSWORD[[:space:]]+["'\'']?\$\{'
assert_contains "${BASE_COMPOSE}" 'entrypoint: \["/usr/local/bin/stuhelper-postgres-entrypoint"\]'
assert_contains "${BASE_COMPOSE}" 'docker-entrypoint-with-tls\.sh:/usr/local/bin/stuhelper-postgres-entrypoint:ro'
assert_contains "${BASE_COMPOSE}" '/var/lib/postgres/initdb/00-init-extra-dbs\.sh:ro'

for production_image_var in \
  POSTGRES_IMAGE_REF \
  REDIS_IMAGE_REF \
  RCLONE_IMAGE_REF \
  GOLANG_IMAGE_REF \
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
  BLACKBOX_EXPORTER_IMAGE_REF; do
  assert_contains "${PROD_ENV_EXAMPLE}" "^${production_image_var}=.*@sha256:[0-9a-f]{64}$"
done
assert_contains "${PROD_DEPLOY}" '--effective-environment production'
assert_contains "${REMOTE_PREFLIGHT}" '--effective-environment production'

github_runtime_block="$(github_job_block runtime-image-security)"
[[ "${github_runtime_block}" == *"actions/cache@55cc8345863c7cc4c66a329aec7e433d2d1c52a9"* ]] ||
  fail "GitHub runtime-image job must use the pinned cache action"
[[ "${github_runtime_block}" == *"bash infra/ops/scan-runtime-images.sh"* ]] ||
  fail "GitHub runtime-image job must invoke the repository scanner"
[[ "${github_runtime_block}" == *"runtime-image-scan-evidence/*.json"* ]] ||
  fail "GitHub runtime-image job must retain JSON evidence"

github_required_block="$(github_job_block required)"
[[ "${github_required_block}" == *"- runtime-image-security"* ]] ||
  fail "GitHub required gate must depend on runtime-image-security"

assert_contains "${GITHUB_CI}" 'image: cgr\.dev/chainguard/postgres:latest@sha256:[0-9a-f]{64}$'
assert_contains "${GITHUB_CI}" 'image: redis:8\.8\.1-alpine@sha256:[0-9a-f]{64}$'

printf '[runtime-image-security-contract] all assertions passed\n'
