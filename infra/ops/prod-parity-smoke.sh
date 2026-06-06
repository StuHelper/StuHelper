#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

require_cmd curl
require_cmd jq
require_cmd python3

PARITY_DIR="${PROD_PARITY_DIR:-${REPO_ROOT}/.run/prod-parity}"
parity_default_path() {
  local current="$1"
  local common_default="$2"
  local parity_default="$3"
  if repo_default_path_matches "${current}" "${common_default}"; then
    printf '%s\n' "${parity_default}"
    return
  fi
  printf '%s\n' "${current}"
}

export ENV_TEMPLATE_FILE="${REPO_ROOT}/.env.prod.example"
export ENV_FILE="$(parity_default_path "${ENV_FILE:-}" "${REPO_ROOT}/.env" "${PARITY_DIR}/.env.prod.shared")"
export SECRETS_ENV_FILE="$(parity_default_path "${SECRETS_ENV_FILE:-}" "" "${PARITY_DIR}/.env.prod.secrets.local")"
export GENERATED_ENV_FILE="$(parity_default_path "${GENERATED_ENV_FILE:-}" "${REPO_ROOT}/.env.generated" "${PARITY_DIR}/.env.prod.generated")"
export GENERATED_SECRET_ENV_FILE="$(parity_default_path "${GENERATED_SECRET_ENV_FILE:-}" "${REPO_ROOT}/.env.generated.secrets" "${PARITY_DIR}/.env.prod.generated.secrets")"
export DEPLOY_STATE_DIR="$(parity_default_path "${DEPLOY_STATE_DIR:-}" "${REPO_ROOT}/.deploy" "${PARITY_DIR}/deploy-state")"

preserved_app_env="${APP_ENV-__STUHELPER_UNSET__}"
load_env
if [[ "${preserved_app_env}" != "__STUHELPER_UNSET__" ]]; then
  export APP_ENV="${preserved_app_env}"
elif [[ "${APP_ENV:-}" == "production" && "${ENV_FILE}" == "${PARITY_DIR}/.env.prod.shared" ]]; then
  export APP_ENV=prod-parity
else
  export APP_ENV="${APP_ENV:-prod-parity}"
fi

export API_BASE_URL="http://127.0.0.1:${BACKEND_EXTERNAL_PORT:-28080}"
export WEB_BASE_URL="${WEB_PUBLIC_URL:-https://stuhelper.com}"
export ADMIN_BASE_URL="${WEB_PUBLIC_URL:-https://stuhelper.com}"
export CHECK_ADMIN=true
export SMOKE_CHECK_CURL_INSECURE=true

"${SCRIPT_DIR}/prod-parity-datastore-smoke.sh"
"${SCRIPT_DIR}/prod-parity-smoke-data.sh"
"${SCRIPT_DIR}/admission-production-readiness.sh"

"${SCRIPT_DIR}/smoke-check.sh"

SSO_PUBLIC_BASE_URL="https://sso.stuhelper.com" \
SSO_PUBLIC_SMOKE_EXPECTED_ISSUER="${CASDOOR_ISSUER:-http://sso.stuhelper.com}" \
SSO_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS=true \
SSO_PUBLIC_SMOKE_CURL_INSECURE=true \
SSO_PUBLIC_SMOKE_EVIDENCE_FILE="${PARITY_DIR}/sso-public-smoke-evidence.json" \
"${SCRIPT_DIR}/sso-public-smoke.sh"

ADMISSION_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS=true \
ADMISSION_PUBLIC_SMOKE_CURL_INSECURE=true \
ADMISSION_PUBLIC_SMOKE_EVIDENCE_FILE="${PARITY_DIR}/admission-public-smoke-evidence.json" \
"${SCRIPT_DIR}/admission-public-smoke.sh"

OPENFGA_RESOURCE_SMOKE_MODE=container \
OPENFGA_RESOURCE_SMOKE_EVIDENCE_FILE="${PARITY_DIR}/openfga-resource-access-smoke.json" \
"${SCRIPT_DIR}/openfga-resource-access-smoke.sh" >/dev/null

"${SCRIPT_DIR}/prod-parity-browser-smoke.sh"
"${SCRIPT_DIR}/admission-prod-sim-e2e.sh"

PROMETHEUS_URL="http://127.0.0.1:${PROMETHEUS_PORT:-29090}/-/ready" \
GRAFANA_URL="http://127.0.0.1:${GRAFANA_PORT:-23003}/api/health" \
LOKI_URL="http://127.0.0.1:${LOKI_PORT:-23100}/ready" \
TEMPO_URL="http://127.0.0.1:${TEMPO_HTTP_PORT:-23200}/ready" \
ALERTMANAGER_URL="http://127.0.0.1:${ALERTMANAGER_PORT:-29093}/-/ready" \
ALLOY_URL="http://127.0.0.1:${ALLOY_HTTP_PORT:-22345}/-/ready" \
OBS_SMOKE_STRICT=false \
OBSERVABILITY_SMOKE_EVIDENCE_FILE="${PARITY_DIR}/observability-smoke-evidence.json" \
"${SCRIPT_DIR}/observability-smoke-check.sh"
