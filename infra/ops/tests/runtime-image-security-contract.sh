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
PROD_ROLLBACK="${REPO_ROOT}/infra/ops/prod-rollback.sh"
REMOTE_PREFLIGHT="${REPO_ROOT}/infra/ops/remote-preflight.sh"
REVIEW_WORKFLOW="${REPO_ROOT}/.github/workflows/runtime-image-review.yml"

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

if python3 "${VALIDATOR}" \
  --repo-root "${REPO_ROOT}" \
  --policy "${POLICY}" \
  --policy-only \
  --today 2026-08-13 >/dev/null 2>&1; then
  fail "expired runtime-image review windows must fail normal validation"
fi

python3 "${VALIDATOR}" \
  --repo-root "${REPO_ROOT}" \
  --policy "${POLICY}" \
  --policy-only \
  --today 2026-07-30 \
  --minimum-review-days-remaining 6 >/dev/null
if python3 "${VALIDATOR}" \
  --repo-root "${REPO_ROOT}" \
  --policy "${POLICY}" \
  --policy-only \
  --today 2026-07-30 \
  --minimum-review-days-remaining 7 >/dev/null 2>&1; then
  fail "review-deadline validation must fail before fewer than the requested days remain"
fi

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

rollback_fixture="$(mktemp -d)"
trap 'rm -rf "${rollback_fixture}"' EXIT
rollback_tag="0123456789abcdef0123456789abcdef01234567"
rollback_state="${rollback_fixture}/deploy-state"
rollback_record="${rollback_state}/releases/${rollback_tag}.env"
mkdir -p "$(dirname "${rollback_record}")"
backend_ref="ghcr.io/stuhelper/backend@sha256:1111111111111111111111111111111111111111111111111111111111111111"
frontend_ref="ghcr.io/stuhelper/frontend@sha256:2222222222222222222222222222222222222222222222222222222222222222"
admin_ref="ghcr.io/stuhelper/admin@sha256:3333333333333333333333333333333333333333333333333333333333333333"
cat >"${rollback_record}" <<EOF
TAG=${rollback_tag}
DEPLOYED_AT=2026-07-30T12:00:00Z
BACKEND_IMAGE_REF=${backend_ref}
FRONTEND_IMAGE_REF=${frontend_ref}
ADMIN_IMAGE_REF=${admin_ref}
EOF

rollback_output="$(
  TAG="${rollback_tag}" \
  ROLLBACK_TAG="${rollback_tag}" \
  DEPLOY_STATE_DIR="${rollback_state}" \
  BACKEND_IMAGE_REF="${backend_ref}" \
  FRONTEND_IMAGE_REF="${frontend_ref}" \
  ADMIN_IMAGE_REF="${admin_ref}" \
  ROLLBACK_REVIEW_ACTOR="contract-test" \
  ROLLBACK_REVIEW_REASON="restore a previously successful immutable release" \
  ROLLBACK_REVIEW_AUDIT_ID="11111111-1111-4111-8111-111111111111" \
  RUNTIME_IMAGE_ROLLBACK_RELEASE_RECORD="${rollback_record}" \
    python3 "${VALIDATOR}" \
      --repo-root "${REPO_ROOT}" \
      --policy "${POLICY}" \
      --policy-only \
      --effective-environment production \
      --today 2026-08-13 2>&1
)"
[[ "${rollback_output}" == *"audited rollback is reusing review windows valid at 2026-07-30T12:00:00Z"* ]] ||
  fail "audited rollback validation must report the reused historical review date"

legacy_rollback_record="${rollback_state}/releases/legacy-release.env"
cat >"${legacy_rollback_record}" <<'EOF'
TAG=legacy-release
DEPLOYED_AT=2026-07-30T12:00:00Z
BACKEND_IMAGE_REF=ghcr.io/stuhelper/backend:legacy-release
FRONTEND_IMAGE_REF=ghcr.io/stuhelper/frontend:legacy-release
ADMIN_IMAGE_REF=ghcr.io/stuhelper/admin:legacy-release
EOF
if TAG=legacy-release \
  ROLLBACK_TAG=legacy-release \
  DEPLOY_STATE_DIR="${rollback_state}" \
  BACKEND_IMAGE_REF="${backend_ref}" \
  FRONTEND_IMAGE_REF="${frontend_ref}" \
  ADMIN_IMAGE_REF="${admin_ref}" \
  ROLLBACK_REVIEW_ACTOR="contract-test" \
  ROLLBACK_REVIEW_REASON="validate a provenance-verified legacy digest transition" \
  ROLLBACK_REVIEW_AUDIT_ID="11111111-1111-4111-8111-111111111111" \
    python3 "${VALIDATOR}" \
      --repo-root "${REPO_ROOT}" \
      --policy "${POLICY}" \
      --policy-only \
      --effective-environment production \
      --rollback-release-record "${legacy_rollback_record}" \
      --today 2026-08-13 >/dev/null 2>&1; then
  fail "audited rollback accepted a legacy record without the explicit transition gate"
fi

TAG=legacy-release \
ROLLBACK_TAG=legacy-release \
DEPLOY_STATE_DIR="${rollback_state}" \
BACKEND_IMAGE_REF="${backend_ref}" \
FRONTEND_IMAGE_REF="${frontend_ref}" \
ADMIN_IMAGE_REF="${admin_ref}" \
ROLLBACK_REVIEW_ACTOR="contract-test" \
ROLLBACK_REVIEW_REASON="validate a provenance-verified legacy digest transition" \
ROLLBACK_REVIEW_AUDIT_ID="11111111-1111-4111-8111-111111111111" \
  python3 "${VALIDATOR}" \
    --repo-root "${REPO_ROOT}" \
    --policy "${POLICY}" \
    --policy-only \
    --effective-environment production \
    --rollback-release-record "${legacy_rollback_record}" \
    --allow-legacy-rollback-record \
    --today 2026-08-13 >/dev/null

if TAG=legacy-release \
  ROLLBACK_TAG=legacy-release \
  DEPLOY_STATE_DIR="${rollback_state}" \
  BACKEND_IMAGE_REF="ghcr.io/other/backend@sha256:1111111111111111111111111111111111111111111111111111111111111111" \
  FRONTEND_IMAGE_REF="${frontend_ref}" \
  ADMIN_IMAGE_REF="${admin_ref}" \
  ROLLBACK_REVIEW_ACTOR="contract-test" \
  ROLLBACK_REVIEW_REASON="validate a provenance-verified legacy digest transition" \
  ROLLBACK_REVIEW_AUDIT_ID="11111111-1111-4111-8111-111111111111" \
    python3 "${VALIDATOR}" \
      --repo-root "${REPO_ROOT}" \
      --policy "${POLICY}" \
      --policy-only \
      --effective-environment production \
      --rollback-release-record "${legacy_rollback_record}" \
      --allow-legacy-rollback-record \
      --today 2026-08-13 >/dev/null 2>&1; then
  fail "legacy rollback transition accepted a different application repository"
fi

if TAG="${rollback_tag}" \
  ROLLBACK_TAG="${rollback_tag}" \
  DEPLOY_STATE_DIR="${rollback_state}" \
  BACKEND_IMAGE_REF="ghcr.io/stuhelper/backend@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" \
  FRONTEND_IMAGE_REF="${frontend_ref}" \
  ADMIN_IMAGE_REF="${admin_ref}" \
  ROLLBACK_REVIEW_ACTOR="contract-test" \
  ROLLBACK_REVIEW_REASON="restore a previously successful immutable release" \
  ROLLBACK_REVIEW_AUDIT_ID="11111111-1111-4111-8111-111111111111" \
  RUNTIME_IMAGE_ROLLBACK_RELEASE_RECORD="${rollback_record}" \
    python3 "${VALIDATOR}" \
      --repo-root "${REPO_ROOT}" \
      --policy "${POLICY}" \
      --policy-only \
      --effective-environment production \
      --today 2026-08-13 >/dev/null 2>&1; then
  fail "audited rollback must reject application image drift from the successful release record"
fi

if TAG="${rollback_tag}" \
  ROLLBACK_TAG="${rollback_tag}" \
  DEPLOY_STATE_DIR="${rollback_state}" \
  BACKEND_IMAGE_REF="${backend_ref}" \
  FRONTEND_IMAGE_REF="${frontend_ref}" \
  ADMIN_IMAGE_REF="${admin_ref}" \
  ROLLBACK_REVIEW_ACTOR="contract-test" \
  ROLLBACK_REVIEW_REASON="short" \
  ROLLBACK_REVIEW_AUDIT_ID="11111111-1111-4111-8111-111111111111" \
  RUNTIME_IMAGE_ROLLBACK_RELEASE_RECORD="${rollback_record}" \
    python3 "${VALIDATOR}" \
      --repo-root "${REPO_ROOT}" \
      --policy "${POLICY}" \
      --policy-only \
      --effective-environment production \
      --today 2026-08-13 >/dev/null 2>&1; then
  fail "audited rollback must require a meaningful reason"
fi

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
assert_not_contains "${PROD_DEPLOY}" 'RUNTIME_IMAGE_ROLLBACK_RELEASE_RECORD='
assert_not_contains "${REMOTE_PREFLIGHT}" 'RUNTIME_IMAGE_ROLLBACK_RELEASE_RECORD='
assert_contains "${PROD_ROLLBACK}" 'RUNTIME_IMAGE_ROLLBACK_RELEASE_RECORD='
assert_contains "${PROD_ROLLBACK}" 'rollback-review-exceptions\.jsonl'
assert_contains "${PROD_ROLLBACK}" 'runtime_image_review_window_exception_authorized'
assert_contains "${REVIEW_WORKFLOW}" 'cron: "17 1 \* \* \*"'
assert_contains "${REVIEW_WORKFLOW}" '--minimum-review-days-remaining 3'

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
