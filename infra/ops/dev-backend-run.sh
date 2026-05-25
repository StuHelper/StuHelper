#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"
# shellcheck source=lib/dev-local.sh
source "${SCRIPT_DIR}/lib/dev-local.sh"

load_env

AIR_BIN="${AIR_BIN:-$(ensure_air)}"

cd "${REPO_ROOT}/server"

export APP_ENV=development
export DATABASE_URL="postgresql://${STUHELPER_APP_DB_USER:-stuhelper_app}:${STUHELPER_APP_DB_PASSWORD}@localhost:${POSTGRES_EXTERNAL_PORT:-5432}/${POSTGRES_DB:-stuhelper}?sslmode=disable"
export REDIS_HOST=localhost
export REDIS_PORT="${REDIS_EXTERNAL_PORT:-6379}"
export REDIS_USERNAME="${REDIS_USERNAME:-stuhelper_app}"
export REDIS_PASSWORD="${REDIS_PASSWORD}"
export REDIS_TLS_ENABLED=true
export REDIS_TLS_CA="${REPO_ROOT}/infra/generated/redis/ca.crt"
export CASDOOR_INTERNAL_ADDRESS=""
export OPENFGA_API_URL="http://localhost:${OPENFGA_HTTP_EXTERNAL_PORT:-8081}"

exec "${AIR_BIN}" -c .air.toml
