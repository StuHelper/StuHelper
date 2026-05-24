#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

if [[ -z "${ENV_TEMPLATE_FILE:-}" || "${ENV_TEMPLATE_FILE}" == "${REPO_ROOT}/.env.example" ]]; then
  export ENV_TEMPLATE_FILE="${REPO_ROOT}/.env.prod.example"
fi
if [[ "${ENV_FILE:-${REPO_ROOT}/.env}" == "${REPO_ROOT}/.env" || "${ENV_FILE:-.env}" == ".env" ]]; then
  ENV_FILE="${REPO_ROOT}/.env.prod.shared"
fi
if [[ -z "${SECRETS_ENV_FILE:-}" ]]; then
  SECRETS_ENV_FILE="${REPO_ROOT}/.env.prod.secrets.local"
fi
if [[ "${GENERATED_ENV_FILE:-${REPO_ROOT}/.env.generated}" == "${REPO_ROOT}/.env.generated" || "${GENERATED_ENV_FILE:-.env.generated}" == ".env.generated" ]]; then
  GENERATED_ENV_FILE="${REPO_ROOT}/.env.prod.generated"
fi
if [[ -z "${GENERATED_SECRET_ENV_FILE:-}" || "${GENERATED_SECRET_ENV_FILE:-}" == "${REPO_ROOT}/.env.generated.secrets" ]]; then
  GENERATED_SECRET_ENV_FILE="${REPO_ROOT}/.env.prod.generated.secrets"
fi

require_cmd python3

load_env

STACK_NAME_VALUE="${STACK_NAME:-${COMPOSE_PROJECT_NAME:-stuhelper}}"
DOCKER_NETWORK_NAME="${DOCKER_NETWORK_NAME:-${STACK_NAME_VALUE}-backend}"
bootstrap_mode="${IDENTITY_PUBLIC_SMOKE_BOOTSTRAP_MODE:-container}"
go_image_ref="${IDENTITY_PUBLIC_SMOKE_BOOTSTRAP_GO_IMAGE:-${GOLANG_IMAGE_REF:-golang:1.26-bookworm}}"

default_public_url="${WEB_PUBLIC_URL:-https://stuhelper.com}"
default_redirect_uri="$(trim_trailing_slash "${default_public_url}")/open-platform/identity-public-smoke/callback"
default_privacy_url="$(trim_trailing_slash "${default_public_url}")/privacy"

export IDENTITY_PUBLIC_SMOKE_CLIENT_ID="${IDENTITY_PUBLIC_SMOKE_CLIENT_ID:-identity-public-smoke}"
export IDENTITY_PUBLIC_SMOKE_REDIRECT_URI="${IDENTITY_PUBLIC_SMOKE_REDIRECT_URI:-${default_redirect_uri}}"
export IDENTITY_PUBLIC_SMOKE_CLIENT_CREDENTIALS_SCOPE="${IDENTITY_PUBLIC_SMOKE_CLIENT_CREDENTIALS_SCOPE:-resource.read}"
export IDENTITY_PUBLIC_SMOKE_HOMEPAGE_URL="${IDENTITY_PUBLIC_SMOKE_HOMEPAGE_URL:-${default_public_url}}"
export IDENTITY_PUBLIC_SMOKE_PRIVACY_POLICY_URL="${IDENTITY_PUBLIC_SMOKE_PRIVACY_POLICY_URL:-${default_privacy_url}}"

[[ -n "${IDENTITY_PUBLIC_SMOKE_OWNER_USER_ID:-}" ]] || die "IDENTITY_PUBLIC_SMOKE_OWNER_USER_ID is required"
[[ -n "${IDENTITY_PUBLIC_SMOKE_REVIEWER_USER_ID:-}" ]] || die "IDENTITY_PUBLIC_SMOKE_REVIEWER_USER_ID is required"
[[ -n "${DATABASE_URL:-}" ]] || die "DATABASE_URL is required"

run_bootstrap_host() {
  require_cmd go
  (
    cd "${REPO_ROOT}/server"
    go run ./cmd/identity-public-smoke-client-bootstrap
  )
}

run_bootstrap_container() {
  require_cmd docker
  docker run --rm \
    --network "${DOCKER_NETWORK_NAME}" \
    -v "${REPO_ROOT}:/workspace" \
    -v "${REPO_ROOT}/infra/generated/postgres:/tls:ro" \
    -w /workspace/server \
    -e DATABASE_URL="${DATABASE_URL}" \
    -e IDENTITY_PUBLIC_SMOKE_OWNER_USER_ID="${IDENTITY_PUBLIC_SMOKE_OWNER_USER_ID}" \
    -e IDENTITY_PUBLIC_SMOKE_REVIEWER_USER_ID="${IDENTITY_PUBLIC_SMOKE_REVIEWER_USER_ID}" \
    -e IDENTITY_PUBLIC_SMOKE_CLIENT_ID="${IDENTITY_PUBLIC_SMOKE_CLIENT_ID}" \
    -e IDENTITY_PUBLIC_SMOKE_CLIENT_SECRET="${IDENTITY_PUBLIC_SMOKE_CLIENT_SECRET:-}" \
    -e IDENTITY_PUBLIC_SMOKE_DISPLAY_NAME="${IDENTITY_PUBLIC_SMOKE_DISPLAY_NAME:-}" \
    -e IDENTITY_PUBLIC_SMOKE_DESCRIPTION="${IDENTITY_PUBLIC_SMOKE_DESCRIPTION:-}" \
    -e IDENTITY_PUBLIC_SMOKE_HOMEPAGE_URL="${IDENTITY_PUBLIC_SMOKE_HOMEPAGE_URL}" \
    -e IDENTITY_PUBLIC_SMOKE_PRIVACY_POLICY_URL="${IDENTITY_PUBLIC_SMOKE_PRIVACY_POLICY_URL}" \
    -e IDENTITY_PUBLIC_SMOKE_REDIRECT_URI="${IDENTITY_PUBLIC_SMOKE_REDIRECT_URI}" \
    -e IDENTITY_PUBLIC_SMOKE_CLIENT_CREDENTIALS_SCOPE="${IDENTITY_PUBLIC_SMOKE_CLIENT_CREDENTIALS_SCOPE}" \
    -e IDENTITY_PUBLIC_SMOKE_BOOTSTRAP_REQUEST_ID="${IDENTITY_PUBLIC_SMOKE_BOOTSTRAP_REQUEST_ID:-identity-public-smoke-bootstrap}" \
    -e IDENTITY_PUBLIC_SMOKE_BOOTSTRAP_ALLOW_REVOKED_REPAIR="${IDENTITY_PUBLIC_SMOKE_BOOTSTRAP_ALLOW_REVOKED_REPAIR:-false}" \
    "${go_image_ref}" \
    go run ./cmd/identity-public-smoke-client-bootstrap
}

bootstrap_output="$(mktemp)"
chmod 600 "${bootstrap_output}"
trap 'rm -f "${bootstrap_output}"' EXIT

case "${bootstrap_mode}" in
  host)
    run_bootstrap_host >"${bootstrap_output}"
    ;;
  container)
    run_bootstrap_container >"${bootstrap_output}"
    ;;
  *)
    die "IDENTITY_PUBLIC_SMOKE_BOOTSTRAP_MODE must be host or container"
    ;;
esac

while IFS='=' read -r key value; do
  [[ -n "${key}" ]] || continue
  case "${key}" in
    IDENTITY_PUBLIC_SMOKE_CLIENT_SECRET)
      upsert_env_file "${SECRETS_ENV_FILE}" "${key}" "${value}"
      ;;
    IDENTITY_PUBLIC_SMOKE_CLIENT_ID|IDENTITY_PUBLIC_SMOKE_REDIRECT_URI|IDENTITY_PUBLIC_SMOKE_CLIENT_CREDENTIALS_SCOPE|IDENTITY_PUBLIC_SMOKE_RESOURCE_ACCESS_EXPECT_ALLOWED)
      upsert_env_file "${ENV_FILE}" "${key}" "${value}"
      if [[ "${key}" == "IDENTITY_PUBLIC_SMOKE_CLIENT_ID" ]]; then
        IDENTITY_PUBLIC_SMOKE_CLIENT_ID="${value}"
      fi
      ;;
    *)
      die "unexpected bootstrap output key: ${key}"
      ;;
  esac
done <"${bootstrap_output}"

log "ensured approved Identity public smoke client ${IDENTITY_PUBLIC_SMOKE_CLIENT_ID}"
