#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
DEV_UP="${REPO_ROOT}/infra/ops/dev-up.sh"
DEV_BACKEND_RUN="${REPO_ROOT}/infra/ops/dev-backend-run.sh"
DEV_LOCAL="${REPO_ROOT}/infra/ops/lib/dev-local.sh"

fail() {
  echo "[dev-up-contract][error] $*" >&2
  exit 1
}

assert_contains() {
  local file="$1"
  local pattern="$2"
  if ! grep -Eq "${pattern}" "${file}"; then
    fail "expected ${file} to contain pattern: ${pattern}"
  fi
}

assert_not_contains() {
  local file="$1"
  local pattern="$2"
  if grep -Eq "${pattern}" "${file}"; then
    fail "expected ${file} to not contain pattern: ${pattern}"
  fi
}

[[ -f "${DEV_UP}" ]] || fail "missing file: ${DEV_UP}"
[[ -x "${DEV_BACKEND_RUN}" ]] || fail "missing executable file: ${DEV_BACKEND_RUN}"
[[ -f "${DEV_LOCAL}" ]] || fail "missing file: ${DEV_LOCAL}"

bash -n "${DEV_UP}"
bash -n "${DEV_BACKEND_RUN}"
bash -n "${DEV_LOCAL}"

assert_contains "${DEV_LOCAL}" 'port_is_published_by_container'
assert_contains "${DEV_LOCAL}" 'pick_available_or_current_container_port'
assert_contains "${DEV_LOCAL}" 'sync_dev_observability_ports'
assert_contains "${DEV_UP}" 'sync_dev_casdoor_builtin_bootstrap_credentials'
assert_contains "${DEV_UP}" 'ensure_dev_casdoor_web_login_user'
assert_contains "${DEV_UP}" 'sync_dev_browser_public_urls'
assert_contains "${DEV_UP}" "SELECT client_id, client_secret FROM application WHERE name = 'app-built-in' AND organization = 'built-in' LIMIT 1"
assert_contains "${DEV_UP}" 'built-in/admin Casdoor user is required before dev web login bootstrap'
assert_contains "${DEV_UP}" 'INSERT INTO public\."user" AS target'
assert_contains "${DEV_UP}" 'ON CONFLICT \(owner, name\) DO UPDATE'
assert_contains "${DEV_UP}" 'signup_application = EXCLUDED\.signup_application'
assert_contains "${DEV_UP}" 'CASDOOR_BOOTSTRAP_ENABLED=true "\$\{SCRIPT_DIR\}/bootstrap-platform\.sh" dev'
assert_contains "${DEV_UP}" 'POSTGRES_EXTERNAL_PORT_SELECTED="\$\(pick_available_or_current_container_port "\$\{POSTGRES_EXTERNAL_PORT:-5432\}" 30 "\$\{STACK_NAME:-stuhelper-dev\}-postgres"\)"'
assert_contains "${DEV_UP}" 'REDIS_EXTERNAL_PORT_SELECTED="\$\(pick_available_or_current_container_port "\$\{REDIS_EXTERNAL_PORT:-6379\}" 30 "\$\{STACK_NAME:-stuhelper-dev\}-redis" "\$\{POSTGRES_EXTERNAL_PORT_SELECTED\}"\)"'
assert_contains "${DEV_UP}" 'OPENFGA_HTTP_EXTERNAL_PORT_SELECTED="\$\(pick_available_or_current_container_port "\$\{OPENFGA_HTTP_EXTERNAL_PORT:-8081\}" 30 "\$\{STACK_NAME:-stuhelper-dev\}-openfga" "\$\{POSTGRES_EXTERNAL_PORT_SELECTED\}" "\$\{REDIS_EXTERNAL_PORT_SELECTED\}"\)"'
assert_contains "${DEV_UP}" 'OPENFGA_GRPC_EXTERNAL_PORT_SELECTED="\$\(pick_available_or_current_container_port "\$\{OPENFGA_GRPC_EXTERNAL_PORT:-8082\}" 30 "\$\{STACK_NAME:-stuhelper-dev\}-openfga" "\$\{POSTGRES_EXTERNAL_PORT_SELECTED\}" "\$\{REDIS_EXTERNAL_PORT_SELECTED\}" "\$\{OPENFGA_HTTP_EXTERNAL_PORT_SELECTED\}"\)"'
assert_contains "${DEV_UP}" 'OPENFGA_PLAYGROUND_EXTERNAL_PORT_SELECTED="\$\(pick_available_or_current_container_port "\$\{OPENFGA_PLAYGROUND_EXTERNAL_PORT:-3002\}" 30 "\$\{STACK_NAME:-stuhelper-dev\}-openfga" "\$\{POSTGRES_EXTERNAL_PORT_SELECTED\}" "\$\{REDIS_EXTERNAL_PORT_SELECTED\}" "\$\{OPENFGA_HTTP_EXTERNAL_PORT_SELECTED\}" "\$\{OPENFGA_GRPC_EXTERNAL_PORT_SELECTED\}"\)"'
assert_contains "${DEV_UP}" 'MINIO_API_EXTERNAL_PORT_SELECTED="\$\(pick_available_or_current_container_port "\$\{MINIO_API_EXTERNAL_PORT:-9000\}" 30 "\$\{STACK_NAME:-stuhelper-dev\}-minio" "\$\{POSTGRES_EXTERNAL_PORT_SELECTED\}" "\$\{REDIS_EXTERNAL_PORT_SELECTED\}" "\$\{OPENFGA_HTTP_EXTERNAL_PORT_SELECTED\}" "\$\{OPENFGA_GRPC_EXTERNAL_PORT_SELECTED\}" "\$\{OPENFGA_PLAYGROUND_EXTERNAL_PORT_SELECTED\}"\)"'
assert_contains "${DEV_UP}" 'MINIO_CONSOLE_EXTERNAL_PORT_SELECTED="\$\(pick_available_or_current_container_port "\$\{MINIO_CONSOLE_EXTERNAL_PORT:-9001\}" 30 "\$\{STACK_NAME:-stuhelper-dev\}-minio" "\$\{POSTGRES_EXTERNAL_PORT_SELECTED\}" "\$\{REDIS_EXTERNAL_PORT_SELECTED\}" "\$\{OPENFGA_HTTP_EXTERNAL_PORT_SELECTED\}" "\$\{OPENFGA_GRPC_EXTERNAL_PORT_SELECTED\}" "\$\{OPENFGA_PLAYGROUND_EXTERNAL_PORT_SELECTED\}" "\$\{MINIO_API_EXTERNAL_PORT_SELECTED\}"\)"'
assert_contains "${DEV_UP}" 'MAILPIT_SMTP_EXTERNAL_PORT_SELECTED="\$\(pick_available_or_current_container_port "\$\{MAILPIT_SMTP_EXTERNAL_PORT:-1025\}" 30 "\$\{STACK_NAME:-stuhelper-dev\}-mailpit"'
assert_contains "${DEV_UP}" 'MAILPIT_WEB_EXTERNAL_PORT_SELECTED="\$\(pick_available_or_current_container_port "\$\{MAILPIT_WEB_EXTERNAL_PORT:-8025\}" 30 "\$\{STACK_NAME:-stuhelper-dev\}-mailpit"'
assert_contains "${DEV_UP}" 'upsert_env_file "\$\{ENV_FILE\}" "POSTGRES_EXTERNAL_PORT" "\$\{POSTGRES_EXTERNAL_PORT_SELECTED\}"'
assert_contains "${DEV_UP}" 'upsert_env_file "\$\{ENV_FILE\}" "REDIS_EXTERNAL_PORT" "\$\{REDIS_EXTERNAL_PORT_SELECTED\}"'
assert_contains "${DEV_UP}" 'upsert_env_file "\$\{ENV_FILE\}" "OPENFGA_HTTP_EXTERNAL_PORT" "\$\{OPENFGA_HTTP_EXTERNAL_PORT_SELECTED\}"'
assert_contains "${DEV_UP}" 'upsert_env_file "\$\{ENV_FILE\}" "OPENFGA_GRPC_EXTERNAL_PORT" "\$\{OPENFGA_GRPC_EXTERNAL_PORT_SELECTED\}"'
assert_contains "${DEV_UP}" 'upsert_env_file "\$\{ENV_FILE\}" "OPENFGA_PLAYGROUND_EXTERNAL_PORT" "\$\{OPENFGA_PLAYGROUND_EXTERNAL_PORT_SELECTED\}"'
assert_contains "${DEV_UP}" 'OPENFGA_API_URL'
assert_contains "${DEV_UP}" 'http://localhost:\$\{OPENFGA_HTTP_EXTERNAL_PORT_SELECTED\}'
assert_contains "${DEV_UP}" 'upsert_env_file "\$\{ENV_FILE\}" "MINIO_API_EXTERNAL_PORT" "\$\{MINIO_API_EXTERNAL_PORT_SELECTED\}"'
assert_contains "${DEV_UP}" 'upsert_env_file "\$\{ENV_FILE\}" "MINIO_CONSOLE_EXTERNAL_PORT" "\$\{MINIO_CONSOLE_EXTERNAL_PORT_SELECTED\}"'
assert_contains "${DEV_UP}" 'upsert_env_file "\$\{ENV_FILE\}" "MAILPIT_SMTP_EXTERNAL_PORT" "\$\{MAILPIT_SMTP_EXTERNAL_PORT_SELECTED\}"'
assert_contains "${DEV_UP}" 'upsert_env_file "\$\{ENV_FILE\}" "MAILPIT_WEB_EXTERNAL_PORT" "\$\{MAILPIT_WEB_EXTERNAL_PORT_SELECTED\}"'
assert_contains "${DEV_UP}" 'upsert_env_file "\$\{ENV_FILE\}" "EMAIL_SMTP_HOST" "localhost"'
assert_contains "${DEV_UP}" 'upsert_env_file "\$\{ENV_FILE\}" "EMAIL_SMTP_PORT" "\$\{MAILPIT_SMTP_EXTERNAL_PORT_SELECTED\}"'
assert_contains "${DEV_UP}" 'sync_dev_observability_ports'
assert_contains "${DEV_UP}" 'observability_reserved_ports'
assert_contains "${DEV_UP}" 'WEB_DEV_PORT_SELECTED="\$\(pick_available_port "\$\{WEB_DEV_PORT:-3000\}" 30 .*"\$\{MAILPIT_SMTP_EXTERNAL_PORT_SELECTED\}" "\$\{MAILPIT_WEB_EXTERNAL_PORT_SELECTED\}" "\$\{observability_reserved_ports\[@\]\}"\)"'
assert_contains "${DEV_UP}" 'ADMIN_DEV_PORT_SELECTED="\$\(pick_available_port "\$\{ADMIN_EXTERNAL_PORT:-3001\}" 30 .*"\$\{MAILPIT_SMTP_EXTERNAL_PORT_SELECTED\}" "\$\{MAILPIT_WEB_EXTERNAL_PORT_SELECTED\}" "\$\{observability_reserved_ports\[@\]\}"\)"'
assert_contains "${DEV_UP}" 'mailpit'
assert_contains "${DEV_UP}" 'Mailpit:'
assert_contains "${DEV_UP}" 'sync_dev_browser_public_urls "\$\{WEB_DEV_PORT_SELECTED\}" "\$\{ADMIN_DEV_PORT_SELECTED\}"'
assert_contains "${DEV_UP}" 'upsert_env_file "\$\{ENV_FILE\}" "WEB_PUBLIC_URL" "http://localhost:\$\{web_port\}"'
assert_contains "${DEV_UP}" 'upsert_env_file "\$\{ENV_FILE\}" "ADMISSION_PUBLIC_BASE_URL" "http://localhost:\$\{web_port\}"'
assert_contains "${DEV_UP}" 'upsert_env_file "\$\{ENV_FILE\}" "ADMIN_PUBLIC_URL" "http://localhost:\$\{admin_port\}/admin/"'
assert_contains "${DEV_UP}" 'upsert_env_file "\$\{ENV_FILE\}" "CASDOOR_TOKEN_PROBE_SMOKE_REDIRECT_URI" "http://localhost:\$\{web_port\}/open-platform/token-probe/callback"'
assert_contains "${DEV_UP}" 'upsert_env_file "\$\{ENV_FILE\}" "CORS_ORIGINS" "http://localhost:\$\{web_port\},http://127\.0\.0\.1:\$\{web_port\},http://localhost:\$\{admin_port\},http://127\.0\.0\.1:\$\{admin_port\}"'
assert_contains "${DEV_UP}" 'export VITE_QQ_BOT_ENTRY=.*WEB_VITE_QQ_BOT_ENTRY'
assert_contains "${DEV_UP}" 'export VITE_QQ_BIND_COMMAND=.*WEB_VITE_QQ_BIND_COMMAND'
assert_contains "${DEV_UP}" 'dev-backend-run\.sh'
assert_contains "${DEV_BACKEND_RUN}" 'localhost:\$\{POSTGRES_EXTERNAL_PORT:-5432\}'
assert_contains "${DEV_BACKEND_RUN}" 'REDIS_PORT="\$\{REDIS_EXTERNAL_PORT:-6379\}"'
assert_contains "${DEV_BACKEND_RUN}" 'OPENFGA_API_URL="http://localhost:\$\{OPENFGA_HTTP_EXTERNAL_PORT:-8081\}"'
assert_contains "${DEV_UP}" 'OBJECT_STORAGE_ENDPOINT'
assert_contains "${DEV_UP}" 'http://localhost:\$\{MINIO_API_EXTERNAL_PORT_SELECTED\}'
assert_not_contains "${DEV_UP}" '^  proxy$'
assert_not_contains "${DEV_UP}" 'traefik'

echo "[dev-up-contract] all assertions passed"
