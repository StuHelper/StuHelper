#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"
# shellcheck source=lib/dev-local.sh
source "${SCRIPT_DIR}/lib/dev-local.sh"

require_cmd docker
require_cmd curl
require_cmd jq
require_cmd python3
require_cmd go

dev_up_mode="${DEV_UP_MODE:-local}"

casdoor_bootstrap_env_file_path() {
  case "${CASDOOR_BOOTSTRAP_ENV_FILE:-.env.casdoor-bootstrap.local}" in
    /*) printf '%s\n' "${CASDOOR_BOOTSTRAP_ENV_FILE}" ;;
    *) printf '%s/%s\n' "$(dirname "${ENV_FILE}")" "${CASDOOR_BOOTSTRAP_ENV_FILE:-.env.casdoor-bootstrap.local}" ;;
  esac
}

sync_dev_casdoor_builtin_bootstrap_credentials() {
  local file
  local credentials
  local client_id
  local client_secret
  local i
  file="$(casdoor_bootstrap_env_file_path)"

  mkdir -p "$(dirname "${file}")"
  touch "${file}"
  chmod 600 "${file}" 2>/dev/null || true

  for ((i = 1; i <= 90; i++)); do
    credentials="$(
      docker exec -i "${STACK_NAME:-stuhelper-dev}-postgres" \
        psql -At -F $'\t' \
          -U "${POSTGRES_USER:-stuhelper}" \
          -d "${CASDOOR_DB_NAME:-casdoor}" \
          -c "SELECT client_id, client_secret FROM application WHERE name = 'app-built-in' AND organization = 'built-in' LIMIT 1" \
          2>/dev/null || true
    )"
    IFS=$'\t' read -r client_id client_secret <<<"${credentials}"
    if [[ -n "${client_id}" && -n "${client_secret}" ]]; then
      upsert_env_file "${file}" "CASDOOR_BOOTSTRAP_CLIENT_ID" "${client_id}"
      upsert_env_file "${file}" "CASDOOR_BOOTSTRAP_CLIENT_SECRET" "${client_secret}"
      upsert_env_file "${file}" "CASDOOR_BOOTSTRAP_APPLICATION" "app-built-in"
      upsert_env_file "${file}" "CASDOOR_BOOTSTRAP_CERTIFICATE" "cert-built-in"
      upsert_env_file "${file}" "CASDOOR_BOOTSTRAP_ORGANIZATION" "built-in"
      log "synced local Casdoor bootstrap credentials to ${file}"
      return 0
    fi
    sleep 2
  done

  die "failed to read local Casdoor built-in bootstrap credentials from ${STACK_NAME:-stuhelper-dev}-postgres"
}

ensure_dev_casdoor_web_login_user() {
  local target_org="${CASDOOR_ORGANIZATION:-stuhelper}"
  local target_app="${CASDOOR_CLIENT_ID:-stuhelper-web}"

  [[ "${target_org}" =~ ^[A-Za-z0-9_.-]+$ ]] || die "CASDOOR_ORGANIZATION contains unsupported characters"
  [[ "${target_app}" =~ ^[A-Za-z0-9_.-]+$ ]] || die "CASDOOR_CLIENT_ID contains unsupported characters"

  docker exec -i "${STACK_NAME:-stuhelper-dev}-postgres" \
    psql -v ON_ERROR_STOP=1 \
      -U "${POSTGRES_USER:-stuhelper}" \
      -d "${CASDOOR_DB_NAME:-casdoor}" <<SQL
DO \$\$
DECLARE
  target_org text := '${target_org}';
  target_app text := '${target_app}';
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM public."user"
    WHERE owner = 'built-in'
      AND name = 'admin'
  ) THEN
    RAISE EXCEPTION 'built-in/admin Casdoor user is required before dev web login bootstrap';
  END IF;

  INSERT INTO public."user" AS target (
    owner, name, created_time, updated_time, id, type,
    password, password_salt, password_type,
    display_name, email, email_verified, phone,
    language, is_admin, is_forbidden, is_deleted, is_verified,
    signup_application, register_type, register_source,
    score, karma, ranking, balance, balance_credit, currency, balance_currency,
    properties, roles, permissions, groups
  )
  SELECT
    target_org, name, created_time, updated_time, id, type,
    password, password_salt, password_type,
    display_name, email, true, phone,
    COALESCE(NULLIF(language, ''), 'zh'),
    is_admin, false, false, true,
    target_app, register_type, register_source,
    score, karma, ranking, balance, balance_credit, currency, balance_currency,
    COALESCE(properties, '{}'), roles, permissions, groups
  FROM public."user"
  WHERE owner = 'built-in'
    AND name = 'admin'
  ON CONFLICT (owner, name) DO UPDATE
  SET id = COALESCE(NULLIF(target.id, ''), EXCLUDED.id),
      type = COALESCE(NULLIF(target.type, ''), EXCLUDED.type),
      password = EXCLUDED.password,
      password_salt = EXCLUDED.password_salt,
      password_type = EXCLUDED.password_type,
      email_verified = true,
      language = COALESCE(NULLIF(target.language, ''), EXCLUDED.language),
      is_admin = EXCLUDED.is_admin,
      is_forbidden = false,
      is_deleted = false,
      is_verified = true,
      signup_application = EXCLUDED.signup_application,
      properties = COALESCE(target.properties, EXCLUDED.properties),
      roles = COALESCE(target.roles, EXCLUDED.roles),
      permissions = COALESCE(target.permissions, EXCLUDED.permissions),
      groups = COALESCE(target.groups, EXCLUDED.groups);
END
\$\$;
SQL

  log "ensured local Casdoor ${target_org}/admin user for web login smoke"
}

sync_dev_browser_public_urls() {
  local web_port="$1"
  local admin_port="$2"

  upsert_env_file "${ENV_FILE}" "WEB_PUBLIC_URL" "http://localhost:${web_port}"
  upsert_env_file "${ENV_FILE}" "WEB_VITE_WEB_URL" "http://localhost:${web_port}"
  upsert_env_file "${ENV_FILE}" "ADMISSION_PUBLIC_BASE_URL" "http://join.localhost:${web_port}"
  upsert_env_file "${ENV_FILE}" "ADMIN_PUBLIC_URL" "http://localhost:${admin_port}/admin/"
  upsert_env_file "${ENV_FILE}" "CASDOOR_TOKEN_PROBE_SMOKE_REDIRECT_URI" "http://localhost:${web_port}/open-platform/token-probe/callback"
  upsert_env_file "${ENV_FILE}" "CASDOOR_ADDITIONAL_REDIRECT_URIS" "http://localhost:${web_port}/api/v1/auth/callback,http://127.0.0.1:${web_port}/api/v1/auth/callback,http://join.localhost:${web_port}/api/v1/auth/callback"
  upsert_env_file "${ENV_FILE}" "CASDOOR_ADMIN_ADDITIONAL_REDIRECT_URIS" "http://localhost:${admin_port}/api/v1/auth/callback,http://127.0.0.1:${admin_port}/api/v1/auth/callback"
  upsert_env_file "${ENV_FILE}" "CORS_ORIGINS" "http://localhost:${web_port},http://127.0.0.1:${web_port},http://join.localhost:${web_port},http://localhost:${admin_port},http://127.0.0.1:${admin_port}"
}

ensure_dev_runtime_dirs
ensure_node_toolchain
"${SCRIPT_DIR}/init-dev-env.sh"
load_env

POSTGRES_EXTERNAL_PORT_SELECTED="$(pick_available_or_current_container_port "${POSTGRES_EXTERNAL_PORT:-5432}" 30 "${STACK_NAME:-stuhelper-dev}-postgres")"
REDIS_EXTERNAL_PORT_SELECTED="$(pick_available_or_current_container_port "${REDIS_EXTERNAL_PORT:-6379}" 30 "${STACK_NAME:-stuhelper-dev}-redis" "${POSTGRES_EXTERNAL_PORT_SELECTED}")"
OPENFGA_HTTP_EXTERNAL_PORT_SELECTED="$(pick_available_or_current_container_port "${OPENFGA_HTTP_EXTERNAL_PORT:-8081}" 30 "${STACK_NAME:-stuhelper-dev}-openfga" "${POSTGRES_EXTERNAL_PORT_SELECTED}" "${REDIS_EXTERNAL_PORT_SELECTED}")"
OPENFGA_GRPC_EXTERNAL_PORT_SELECTED="$(pick_available_or_current_container_port "${OPENFGA_GRPC_EXTERNAL_PORT:-8082}" 30 "${STACK_NAME:-stuhelper-dev}-openfga" "${POSTGRES_EXTERNAL_PORT_SELECTED}" "${REDIS_EXTERNAL_PORT_SELECTED}" "${OPENFGA_HTTP_EXTERNAL_PORT_SELECTED}")"
OPENFGA_PLAYGROUND_EXTERNAL_PORT_SELECTED="$(pick_available_or_current_container_port "${OPENFGA_PLAYGROUND_EXTERNAL_PORT:-3002}" 30 "${STACK_NAME:-stuhelper-dev}-openfga" "${POSTGRES_EXTERNAL_PORT_SELECTED}" "${REDIS_EXTERNAL_PORT_SELECTED}" "${OPENFGA_HTTP_EXTERNAL_PORT_SELECTED}" "${OPENFGA_GRPC_EXTERNAL_PORT_SELECTED}")"
DEV_OBJECT_STORAGE_EXTERNAL_PORT_SELECTED="$(pick_available_or_current_container_port "${DEV_OBJECT_STORAGE_EXTERNAL_PORT:-9000}" 30 "${STACK_NAME:-stuhelper-dev}-object-storage" "${POSTGRES_EXTERNAL_PORT_SELECTED}" "${REDIS_EXTERNAL_PORT_SELECTED}" "${OPENFGA_HTTP_EXTERNAL_PORT_SELECTED}" "${OPENFGA_GRPC_EXTERNAL_PORT_SELECTED}" "${OPENFGA_PLAYGROUND_EXTERNAL_PORT_SELECTED}")"
LOCAL_OBJECT_STORAGE_TLS_EXTERNAL_PORT_SELECTED="$(pick_available_or_current_container_port "${LOCAL_OBJECT_STORAGE_TLS_EXTERNAL_PORT:-9001}" 30 "${STACK_NAME:-stuhelper-dev}-object-storage" "${POSTGRES_EXTERNAL_PORT_SELECTED}" "${REDIS_EXTERNAL_PORT_SELECTED}" "${OPENFGA_HTTP_EXTERNAL_PORT_SELECTED}" "${OPENFGA_GRPC_EXTERNAL_PORT_SELECTED}" "${OPENFGA_PLAYGROUND_EXTERNAL_PORT_SELECTED}" "${DEV_OBJECT_STORAGE_EXTERNAL_PORT_SELECTED}")"
MAILPIT_SMTP_EXTERNAL_PORT_SELECTED="$(pick_available_or_current_container_port "${MAILPIT_SMTP_EXTERNAL_PORT:-1025}" 30 "${STACK_NAME:-stuhelper-dev}-mailpit" "${POSTGRES_EXTERNAL_PORT_SELECTED}" "${REDIS_EXTERNAL_PORT_SELECTED}" "${OPENFGA_HTTP_EXTERNAL_PORT_SELECTED}" "${OPENFGA_GRPC_EXTERNAL_PORT_SELECTED}" "${OPENFGA_PLAYGROUND_EXTERNAL_PORT_SELECTED}" "${DEV_OBJECT_STORAGE_EXTERNAL_PORT_SELECTED}" "${LOCAL_OBJECT_STORAGE_TLS_EXTERNAL_PORT_SELECTED}")"
MAILPIT_WEB_EXTERNAL_PORT_SELECTED="$(pick_available_or_current_container_port "${MAILPIT_WEB_EXTERNAL_PORT:-8025}" 30 "${STACK_NAME:-stuhelper-dev}-mailpit" "${POSTGRES_EXTERNAL_PORT_SELECTED}" "${REDIS_EXTERNAL_PORT_SELECTED}" "${OPENFGA_HTTP_EXTERNAL_PORT_SELECTED}" "${OPENFGA_GRPC_EXTERNAL_PORT_SELECTED}" "${OPENFGA_PLAYGROUND_EXTERNAL_PORT_SELECTED}" "${DEV_OBJECT_STORAGE_EXTERNAL_PORT_SELECTED}" "${LOCAL_OBJECT_STORAGE_TLS_EXTERNAL_PORT_SELECTED}" "${MAILPIT_SMTP_EXTERNAL_PORT_SELECTED}")"
if [[ "${POSTGRES_EXTERNAL_PORT:-}" != "${POSTGRES_EXTERNAL_PORT_SELECTED}" ]]; then
  upsert_env_file "${ENV_FILE}" "POSTGRES_EXTERNAL_PORT" "${POSTGRES_EXTERNAL_PORT_SELECTED}"
fi
case "${DATABASE_URL:-}" in
  ""|postgres://stuhelper_app:*@localhost:*"/${POSTGRES_DB:-stuhelper}?sslmode=disable"|postgresql://stuhelper_app:*@localhost:*"/${POSTGRES_DB:-stuhelper}?sslmode=disable")
    upsert_env_file "${ENV_FILE}" "DATABASE_URL" "postgres://stuhelper_app:${STUHELPER_APP_DB_PASSWORD}@localhost:${POSTGRES_EXTERNAL_PORT_SELECTED}/${POSTGRES_DB:-stuhelper}?sslmode=disable"
    ;;
esac
if [[ "${REDIS_EXTERNAL_PORT:-}" != "${REDIS_EXTERNAL_PORT_SELECTED}" ]]; then
  upsert_env_file "${ENV_FILE}" "REDIS_EXTERNAL_PORT" "${REDIS_EXTERNAL_PORT_SELECTED}"
fi
if [[ "${OPENFGA_HTTP_EXTERNAL_PORT:-}" != "${OPENFGA_HTTP_EXTERNAL_PORT_SELECTED}" ]]; then
  upsert_env_file "${ENV_FILE}" "OPENFGA_HTTP_EXTERNAL_PORT" "${OPENFGA_HTTP_EXTERNAL_PORT_SELECTED}"
fi
if [[ "${OPENFGA_GRPC_EXTERNAL_PORT:-}" != "${OPENFGA_GRPC_EXTERNAL_PORT_SELECTED}" ]]; then
  upsert_env_file "${ENV_FILE}" "OPENFGA_GRPC_EXTERNAL_PORT" "${OPENFGA_GRPC_EXTERNAL_PORT_SELECTED}"
fi
if [[ "${OPENFGA_PLAYGROUND_EXTERNAL_PORT:-}" != "${OPENFGA_PLAYGROUND_EXTERNAL_PORT_SELECTED}" ]]; then
  upsert_env_file "${ENV_FILE}" "OPENFGA_PLAYGROUND_EXTERNAL_PORT" "${OPENFGA_PLAYGROUND_EXTERNAL_PORT_SELECTED}"
fi
case "${OPENFGA_API_URL:-}" in
  ""|"http://localhost:${OPENFGA_HTTP_EXTERNAL_PORT:-8081}"|"http://127.0.0.1:${OPENFGA_HTTP_EXTERNAL_PORT:-8081}"|"http://openfga:8080")
    upsert_env_file "${ENV_FILE}" "OPENFGA_API_URL" "http://localhost:${OPENFGA_HTTP_EXTERNAL_PORT_SELECTED}"
    ;;
esac
if [[ "${DEV_OBJECT_STORAGE_EXTERNAL_PORT:-}" != "${DEV_OBJECT_STORAGE_EXTERNAL_PORT_SELECTED}" ]]; then
  upsert_env_file "${ENV_FILE}" "DEV_OBJECT_STORAGE_EXTERNAL_PORT" "${DEV_OBJECT_STORAGE_EXTERNAL_PORT_SELECTED}"
fi
if [[ "${LOCAL_OBJECT_STORAGE_TLS_EXTERNAL_PORT:-}" != "${LOCAL_OBJECT_STORAGE_TLS_EXTERNAL_PORT_SELECTED}" ]]; then
  upsert_env_file "${ENV_FILE}" "LOCAL_OBJECT_STORAGE_TLS_EXTERNAL_PORT" "${LOCAL_OBJECT_STORAGE_TLS_EXTERNAL_PORT_SELECTED}"
fi
if [[ "${MAILPIT_SMTP_EXTERNAL_PORT:-}" != "${MAILPIT_SMTP_EXTERNAL_PORT_SELECTED}" ]]; then
  upsert_env_file "${ENV_FILE}" "MAILPIT_SMTP_EXTERNAL_PORT" "${MAILPIT_SMTP_EXTERNAL_PORT_SELECTED}"
fi
if [[ "${MAILPIT_WEB_EXTERNAL_PORT:-}" != "${MAILPIT_WEB_EXTERNAL_PORT_SELECTED}" ]]; then
  upsert_env_file "${ENV_FILE}" "MAILPIT_WEB_EXTERNAL_PORT" "${MAILPIT_WEB_EXTERNAL_PORT_SELECTED}"
fi
case "${EMAIL_SMTP_HOST:-}" in
  ""|"mailhog"|"mailpit"|"127.0.0.1"|"localhost")
    upsert_env_file "${ENV_FILE}" "EMAIL_SMTP_HOST" "localhost"
    ;;
esac
case "${EMAIL_SMTP_PORT:-}" in
  ""|"587"|"1025")
    upsert_env_file "${ENV_FILE}" "EMAIL_SMTP_PORT" "${MAILPIT_SMTP_EXTERNAL_PORT_SELECTED}"
    ;;
esac
case "${OBJECT_STORAGE_ENDPOINT:-}" in
  ""|"http://localhost:${DEV_OBJECT_STORAGE_EXTERNAL_PORT:-9000}"|"http://127.0.0.1:${DEV_OBJECT_STORAGE_EXTERNAL_PORT:-9000}"|"http://object-storage:8333")
    upsert_env_file "${ENV_FILE}" "OBJECT_STORAGE_ENDPOINT" "http://localhost:${DEV_OBJECT_STORAGE_EXTERNAL_PORT_SELECTED}"
    ;;
esac
load_env

observability_reserved_ports=()
if [[ "${WITH_OBSERVABILITY:-false}" == "true" ]]; then
  sync_dev_observability_ports \
    "${POSTGRES_EXTERNAL_PORT_SELECTED}" \
    "${REDIS_EXTERNAL_PORT_SELECTED}" \
    "${OPENFGA_HTTP_EXTERNAL_PORT_SELECTED}" \
    "${OPENFGA_GRPC_EXTERNAL_PORT_SELECTED}" \
    "${OPENFGA_PLAYGROUND_EXTERNAL_PORT_SELECTED}" \
    "${DEV_OBJECT_STORAGE_EXTERNAL_PORT_SELECTED}" \
    "${LOCAL_OBJECT_STORAGE_TLS_EXTERNAL_PORT_SELECTED}"
  observability_reserved_ports=(
    "${ALLOY_HTTP_PORT_SELECTED}"
    "${OTEL_GRPC_PORT_SELECTED}"
    "${OTEL_HTTP_PORT_SELECTED}"
    "${PROMETHEUS_PORT_SELECTED}"
    "${ALERTMANAGER_PORT_SELECTED}"
    "${LOKI_PORT_SELECTED}"
    "${TEMPO_HTTP_PORT_SELECTED}"
    "${GRAFANA_PORT_SELECTED}"
    "${CADVISOR_PORT_SELECTED}"
    "${POSTGRES_EXPORTER_PORT_SELECTED}"
    "${REDIS_EXPORTER_PORT_SELECTED}"
    "${BLACKBOX_EXPORTER_PORT_SELECTED}"
  )
  load_env
fi

"${SCRIPT_DIR}/render-observability.sh" dev

base_services=(
  postgres
  redis
  object-storage
  mailpit
  migrate-dev
  seed-dev
  casdoor
  openfga-migrate
  openfga
)

log "starting development infrastructure (Docker)"
compose up -d "${base_services[@]}"
"${SCRIPT_DIR}/ensure-postgres-monitoring-role.sh"
compose up --no-deps resource-seed-dev

log "ensuring dockerized dev app containers are stopped"
compose --profile dev-full stop app-dev frontend-dev admin-dev >/dev/null 2>&1 || true
compose --profile dev-full rm -f app-dev frontend-dev admin-dev >/dev/null 2>&1 || true
kill_all_dev_processes
kill_listener_if_matches 8080 "${REPO_ROOT}/server/tmp/stuhelper"
kill_listener_if_matches 5140 "${REPO_ROOT}/bots/koishi"

WEB_DEV_PORT_SELECTED="$(pick_available_port "${WEB_DEV_PORT:-3000}" 30 "${POSTGRES_EXTERNAL_PORT_SELECTED}" "${REDIS_EXTERNAL_PORT_SELECTED}" "${OPENFGA_HTTP_EXTERNAL_PORT_SELECTED}" "${OPENFGA_GRPC_EXTERNAL_PORT_SELECTED}" "${OPENFGA_PLAYGROUND_EXTERNAL_PORT_SELECTED}" "${DEV_OBJECT_STORAGE_EXTERNAL_PORT_SELECTED}" "${LOCAL_OBJECT_STORAGE_TLS_EXTERNAL_PORT_SELECTED}" "${MAILPIT_SMTP_EXTERNAL_PORT_SELECTED}" "${MAILPIT_WEB_EXTERNAL_PORT_SELECTED}" "${observability_reserved_ports[@]}")"
ADMIN_DEV_PORT_SELECTED="$(pick_available_port "${ADMIN_EXTERNAL_PORT:-3001}" 30 "${WEB_DEV_PORT_SELECTED}" "${POSTGRES_EXTERNAL_PORT_SELECTED}" "${REDIS_EXTERNAL_PORT_SELECTED}" "${OPENFGA_HTTP_EXTERNAL_PORT_SELECTED}" "${OPENFGA_GRPC_EXTERNAL_PORT_SELECTED}" "${OPENFGA_PLAYGROUND_EXTERNAL_PORT_SELECTED}" "${DEV_OBJECT_STORAGE_EXTERNAL_PORT_SELECTED}" "${LOCAL_OBJECT_STORAGE_TLS_EXTERNAL_PORT_SELECTED}" "${MAILPIT_SMTP_EXTERNAL_PORT_SELECTED}" "${MAILPIT_WEB_EXTERNAL_PORT_SELECTED}" "${observability_reserved_ports[@]}")"
sync_dev_browser_public_urls "${WEB_DEV_PORT_SELECTED}" "${ADMIN_DEV_PORT_SELECTED}"
load_env

log "bootstrapping platform identities and authorization model"
sync_dev_casdoor_builtin_bootstrap_credentials
CASDOOR_BOOTSTRAP_ENABLED=true "${SCRIPT_DIR}/bootstrap-platform.sh" dev
ensure_dev_casdoor_web_login_user

if [[ "${WITH_OBSERVABILITY:-false}" == "true" ]]; then
  log "starting observability stack for development"
  compose --profile observability up -d --wait alloy prometheus alertmanager loki tempo grafana node-exporter cadvisor postgres-exporter redis-exporter blackbox-exporter
fi

if [[ "${dev_up_mode}" == "dockerized" ]]; then
  log "ensuring local hot-reload processes are stopped"
  kill_all_dev_processes

  log "starting full dockerized development stack"
  compose --profile dev-full up -d --wait app-dev frontend-dev admin-dev

  wait_for_http "backend" "http://127.0.0.1:8080/health/ready" 120 2
  wait_for_http "frontend" "http://127.0.0.1:3000/" 180 2
  wait_for_http "admin" "http://127.0.0.1:${ADMIN_EXTERNAL_PORT:-3001}/admin/" 240 2

  log "dockerized development stack is ready"
  echo "  Web:        http://127.0.0.1:3000"
  echo "  Admin:      http://127.0.0.1:${ADMIN_EXTERNAL_PORT:-3001}/admin/"
  echo "  Backend:    http://127.0.0.1:8080"
  echo "  Casdoor:    http://127.0.0.1:${CASDOOR_EXTERNALPORT:-8085}"
  echo "  Mailpit:    http://127.0.0.1:${MAILPIT_WEB_EXTERNAL_PORT_SELECTED}"
  echo "  Generated:  ${GENERATED_ENV_FILE}"
  exit 0
fi

ensure_pnpm_workspace \
  "${REPO_ROOT}/clients" \
  "${REPO_ROOT}/clients/pnpm-lock.yaml" \
  "clients-root" \
  "corepack enable >/dev/null 2>&1 || true && corepack prepare pnpm@10 --activate >/dev/null 2>&1 && pnpm install --frozen-lockfile"

ensure_pnpm_workspace \
  "${REPO_ROOT}/clients/admin" \
  "${REPO_ROOT}/clients/admin/pnpm-lock.yaml" \
  "clients-admin" \
  "corepack enable >/dev/null 2>&1 || true && corepack prepare pnpm@10 --activate >/dev/null 2>&1 && CI=1 pnpm install --frozen-lockfile"

ensure_yarn_workspace \
  "${REPO_ROOT}/bots/koishi" \
  "${REPO_ROOT}/bots/koishi/yarn.lock" \
  "bots-koishi" \
  "corepack enable >/dev/null 2>&1 || true && corepack yarn install --immutable"

AIR_BIN="$(ensure_air)"
write_dev_runtime_env "${WEB_DEV_PORT_SELECTED}" "${ADMIN_DEV_PORT_SELECTED}"

wait_for_http "Casdoor OIDC metadata" "http://localhost:${CASDOOR_EXTERNALPORT:-8085}/.well-known/openid-configuration" 90 2

backend_cmd="AIR_BIN='${AIR_BIN}' exec '${SCRIPT_DIR}/dev-backend-run.sh'"

frontend_cmd="
  cd '${REPO_ROOT}/clients' && \
  export NODE_ENV=development && \
  export VITE_API_URL='${WEB_VITE_API_URL:-/api}' && \
  export VITE_SSO_URL='${WEB_VITE_SSO_URL:-http://localhost:8085}' && \
  export VITE_WEB_URL='${WEB_VITE_WEB_URL:-http://localhost:3000}' && \
  export VITE_ADMIN_URL='http://localhost:${ADMIN_DEV_PORT_SELECTED}' && \
  export VITE_API_TIMEOUT_MS='${WEB_VITE_API_TIMEOUT_MS:-15000}' && \
  export VITE_QQ_BOT_ENTRY='${WEB_VITE_QQ_BOT_ENTRY:-}' && \
  export VITE_QQ_BIND_COMMAND='${WEB_VITE_QQ_BIND_COMMAND:-绑定}' && \
  export VITE_DEV_PROXY_TARGET='http://127.0.0.1:8080' && \
  exec pnpm --filter @stuhelper/web exec vite --host 127.0.0.1 --strictPort --port ${WEB_DEV_PORT_SELECTED}
"

admin_cmd="
  cd '${REPO_ROOT}/clients/admin' && \
  export NODE_ENV=development && \
  export VITE_DEV_PROXY_TARGET='http://127.0.0.1:8080' && \
  export VITE_GLOB_API_URL='${ADMIN_VITE_API_URL:-/api/v1}' && \
  export VITE_BASE='${ADMIN_VITE_BASE:-/admin/}' && \
  exec pnpm -F @vben/web-ele exec vite --mode development --host 127.0.0.1 --strictPort --port ${ADMIN_DEV_PORT_SELECTED}
"

koishi_cmd="
  cd '${REPO_ROOT}/bots/koishi' && \
  export NODE_ENV=development && \
  export KOISHI_CONFIG_FILE='' && \
  export STUHELPER_CONSOLE_ADMIN_PASSWORD='${STUHELPER_CONSOLE_ADMIN_PASSWORD}' && \
  export STUHELPER_PLATFORM_BASE_URL='${STUHELPER_PLATFORM_BASE_URL:-http://localhost:8080}' && \
  export STUHELPER_PLATFORM_SERVICE_TOKEN='${STUHELPER_PLATFORM_SERVICE_TOKEN}' && \
  export STUHELPER_GROUP_CENTER_DATA_DIR='${STUHELPER_GROUP_CENTER_DATA_DIR:-}' && \
  export STUHELPER_FRESHMAN_MATERIAL_HOSTS='${STUHELPER_FRESHMAN_MATERIAL_HOSTS:-}' && \
  exec corepack yarn dev
"

log "starting hot-reload backend (air)"
start_managed_process backend "${backend_cmd}"
wait_for_http "backend" "http://127.0.0.1:8080/health/ready" 120 2

log "starting hot-reload frontend (Vite)"
start_managed_process frontend "${frontend_cmd}"
wait_for_http "frontend" "http://127.0.0.1:${WEB_DEV_PORT_SELECTED}/" 180 2

log "starting hot-reload admin (Vite)"
start_managed_process admin "${admin_cmd}"
wait_for_http "admin" "http://127.0.0.1:${ADMIN_DEV_PORT_SELECTED}/admin/" 240 2

log "starting Koishi Console and bot plugins"
start_managed_process koishi "${koishi_cmd}"
wait_for_http "koishi" "http://127.0.0.1:5140/" 120 2

log "development stack is ready"
echo "  Web:        http://localhost:${WEB_DEV_PORT_SELECTED}"
echo "  Admin:      http://localhost:${ADMIN_DEV_PORT_SELECTED}/admin/"
echo "  Backend:    http://localhost:8080"
echo "  Koishi:     http://127.0.0.1:5140"
echo "  Casdoor:    http://localhost:${CASDOOR_EXTERNALPORT:-8085}"
echo "  Mailpit:    http://localhost:${MAILPIT_WEB_EXTERNAL_PORT_SELECTED}"
echo "  Logs:       ${DEV_LOG_DIR}"
echo "  Generated:  ${GENERATED_ENV_FILE}"
