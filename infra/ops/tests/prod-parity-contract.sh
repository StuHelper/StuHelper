#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
PARITY_COMPOSE="${REPO_ROOT}/docker-compose.prod-parity-postgres.yml"
INIT_SHARED_PG="${REPO_ROOT}/infra/ops/init-shared-postgres.sh"
PARITY_UP="${REPO_ROOT}/infra/ops/prod-parity-up.sh"
PARITY_DOWN="${REPO_ROOT}/infra/ops/prod-parity-down.sh"
PARITY_SMOKE="${REPO_ROOT}/infra/ops/prod-parity-smoke.sh"
PARITY_SMOKE_DATA="${REPO_ROOT}/infra/ops/prod-parity-smoke-data.sh"
PARITY_DATASTORE_SMOKE="${REPO_ROOT}/infra/ops/prod-parity-datastore-smoke.sh"
PARITY_BROWSER_SMOKE="${REPO_ROOT}/infra/ops/prod-parity-browser-smoke.sh"
PARITY_BROWSER_SMOKE_NODE="${REPO_ROOT}/infra/ops/prod-parity-browser-smoke.mjs"
PARITY_LOCAL_INGRESS="${REPO_ROOT}/infra/ops/install-local-prod-parity-ingress.sh"
PARITY_LOCAL_INGRESS_NGINX="${REPO_ROOT}/infra/nginx/prod-parity-local-ingress.conf"
COMMON_LIB="${REPO_ROOT}/infra/ops/lib/common.sh"
ADMIN_INDEX_HTML="${REPO_ROOT}/clients/admin/apps/web-ele/index.html"
WEB_NGINX="${REPO_ROOT}/clients/web/nginx.conf"
MAKEFILE="${REPO_ROOT}/Makefile"

fail() {
  echo "[prod-parity-contract][error] $*" >&2
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

for file in "${PARITY_COMPOSE}" "${INIT_SHARED_PG}" "${PARITY_UP}" "${PARITY_DOWN}" "${PARITY_SMOKE}" "${PARITY_SMOKE_DATA}" "${PARITY_DATASTORE_SMOKE}" "${PARITY_BROWSER_SMOKE}" "${PARITY_BROWSER_SMOKE_NODE}" "${PARITY_LOCAL_INGRESS}" "${PARITY_LOCAL_INGRESS_NGINX}" "${COMMON_LIB}" "${ADMIN_INDEX_HTML}" "${WEB_NGINX}"; do
  [[ -f "${file}" ]] || fail "missing file: ${file}"
done

bash -n "${INIT_SHARED_PG}" "${PARITY_UP}" "${PARITY_DOWN}" "${PARITY_SMOKE}" "${PARITY_SMOKE_DATA}" "${PARITY_DATASTORE_SMOKE}" "${PARITY_BROWSER_SMOKE}" "${PARITY_LOCAL_INGRESS}"

for default_path in ".env" "./.env" ".env.generated" "./.env.generated" ".env.generated.secrets" "./.env.generated.secrets" ".deploy" "./.deploy"; do
  case "${default_path}" in
    *.generated.secrets|./*.generated.secrets) expected="${REPO_ROOT}/.env.generated.secrets" ;;
    *.generated|./*.generated) expected="${REPO_ROOT}/.env.generated" ;;
    *.env|./*.env) expected="${REPO_ROOT}/.env" ;;
    *) expected="${REPO_ROOT}/.deploy" ;;
  esac
  if ! bash -c 'source "$1"; repo_default_path_matches "$2" "$3"' bash "${COMMON_LIB}" "${default_path}" "${expected}"; then
    fail "expected repo_default_path_matches to accept ${default_path} as ${expected}"
  fi
done

if bash -c 'source "$1"; repo_default_path_matches "$2" "$3"' bash "${COMMON_LIB}" "/tmp/custom.env" "${REPO_ROOT}/.env"; then
  fail "expected repo_default_path_matches to preserve explicit custom env paths"
fi

assert_contains "${PARITY_COMPOSE}" '^  postgres:'
assert_contains "${PARITY_COMPOSE}" 'POSTGRES_PASSWORD: \$\{SHARED_POSTGRES_PASSWORD:\?SHARED_POSTGRES_PASSWORD is required\}'
assert_contains "${PARITY_COMPOSE}" 'PROD_PARITY_POSTGRES_PORT:-15432'
assert_contains "${PARITY_COMPOSE}" 'prod_parity_postgres_data:/var/lib/postgresql'
assert_contains "${PARITY_COMPOSE}" 'name: \$\{EXTERNAL_DATASTORE_NETWORK:-stuhelper-prod-parity-baota-net\}'
assert_contains "${PARITY_COMPOSE}" 'aliases:'
assert_contains "${PARITY_COMPOSE}" 'postgres'
assert_not_contains "${PARITY_COMPOSE}" '^  redis:'
assert_contains "${REPO_ROOT}/docker-compose.prod.yml" 'APP_ENV: \$\{APP_ENV:-production\}'

assert_contains "${INIT_SHARED_PG}" 'STUHELPER_APP_DB_PASSWORD'
assert_contains "${INIT_SHARED_PG}" 'OPENFGA_DB_PASSWORD'
assert_contains "${INIT_SHARED_PG}" 'CASDOOR_DB_PASSWORD'
assert_contains "${INIT_SHARED_PG}" 'CREATE DATABASE %I OWNER %I'
assert_contains "${INIT_SHARED_PG}" 'ALTER ROLE %I WITH LOGIN PASSWORD %L'
assert_contains "${INIT_SHARED_PG}" 'GRANT pg_read_all_data, pg_read_all_settings, pg_read_all_stats'
assert_contains "${INIT_SHARED_PG}" 'GRANT USAGE, CREATE ON SCHEMA public'
assert_contains "${INIT_SHARED_PG}" 'openfga_database'
assert_contains "${INIT_SHARED_PG}" 'casdoor_database'

assert_contains "${PARITY_UP}" 'EXTERNAL_POSTGRES_ENABLED.*true'
assert_contains "${PARITY_UP}" 'EXTERNAL_POSTGRES_ALLOW_PLAINTEXT.*true'
assert_contains "${PARITY_UP}" 'EXTERNAL_DATASTORE_NETWORK.*stuhelper-prod-parity-baota-net'
assert_contains "${PARITY_UP}" 'APP_ENV.*prod-parity'
assert_contains "${PARITY_UP}" 'REDIS_EXTERNAL_PORT.*26379'
assert_contains "${PARITY_UP}" 'OPENFGA_HTTP_EXTERNAL_PORT.*8081'
assert_contains "${PARITY_UP}" 'OPENFGA_GRPC_EXTERNAL_PORT.*8082'
assert_contains "${PARITY_UP}" 'OPENFGA_PLAYGROUND_EXTERNAL_PORT.*3002'
assert_contains "${PARITY_UP}" 'MINIO_API_EXTERNAL_PORT.*29000'
assert_contains "${PARITY_UP}" 'MINIO_CONSOLE_EXTERNAL_PORT.*29001'
assert_contains "${PARITY_UP}" 'WEB_PUBLIC_URL.*https://stuhelper\.com'
assert_contains "${PARITY_UP}" 'ADMIN_PUBLIC_URL.*https://stuhelper\.com/admin/'
assert_contains "${PARITY_UP}" 'IDENTITY_ISSUER.*https://id\.stuhelper\.com'
assert_contains "${PARITY_UP}" 'WEB_VITE_SSO_URL.*https://id\.stuhelper\.com'
assert_contains "${PARITY_UP}" 'WEB_VITE_IDENTITY_URL.*https://id\.stuhelper\.com'
assert_contains "${PARITY_UP}" 'WEB_VITE_WEB_URL.*https://stuhelper\.com'
assert_contains "${PARITY_UP}" 'IDENTITY_PUBLIC_SMOKE_CASDOOR_UPSTREAM_ENABLED.*false'
assert_contains "${PARITY_UP}" 'CASDOOR_EXTERNALPORT.*28085'
assert_contains "${PARITY_UP}" 'CASDOOR_ISSUER.*http://sso\.stuhelper\.com'
assert_contains "${PARITY_UP}" 'CASDOOR_PUBLIC_AUTH_BASE_URL.*https://id\.stuhelper\.com'
assert_contains "${PARITY_UP}" 'CASDOOR_INTERNAL_ADDRESS.*host\.docker\.internal:80'
assert_contains "${PARITY_UP}" 'CASDOOR_REDIRECT_URI.*https://id\.stuhelper\.com/api/v1/auth/callback'
assert_contains "${PARITY_UP}" 'TOKEN_COOKIE_SECURE.*true'
assert_contains "${PARITY_UP}" 'TOKEN_COOKIE_DOMAIN.*\.stuhelper\.com'
assert_contains "${PARITY_UP}" 'CASDOOR_BOOTSTRAP_ENABLED.*true'
assert_contains "${PARITY_UP}" 'CASDOOR_BOOTSTRAP_ENV_FILE'
assert_contains "${PARITY_UP}" 'CASDOOR_DB_PASSWORD'
assert_contains "${PARITY_UP}" 'CASDOOR_BOOTSTRAP_CLIENT_ID'
assert_contains "${PARITY_UP}" 'CASDOOR_BOOTSTRAP_CLIENT_SECRET'
assert_contains "${PARITY_UP}" 'CASDOOR_BOOTSTRAP_ORGANIZATION'
assert_contains "${PARITY_UP}" 'sync_casdoor_builtin_bootstrap_credentials'
assert_contains "${PARITY_UP}" 'app-built-in'
assert_contains "${PARITY_UP}" 'SELECT client_id, client_secret FROM application'
assert_contains "${PARITY_UP}" 'compose --profile prod --profile local-sso up -d --wait casdoor'
assert_contains "${PARITY_UP}" 'parity_default_path'
assert_contains "${PARITY_UP}" '\.run/prod-parity'
assert_contains "${PARITY_UP}" 'init-shared-postgres.sh'
assert_contains "${PARITY_UP}" 'render-redis-tls.sh'
assert_contains "${PARITY_UP}" 'render-redis-acl.sh'
assert_contains "${PARITY_UP}" 'render-observability.sh.*prod'
assert_contains "${PARITY_UP}" 'docker build'
assert_contains "${PARITY_UP}" '.*-f "\$\{REPO_ROOT\}/clients/web/Dockerfile"'
assert_contains "${PARITY_UP}" '"\$\{REPO_ROOT\}"'
assert_contains "${PARITY_UP}" 'compose --profile prod up -d --wait'
assert_contains "${PARITY_UP}" 'bootstrap-platform.sh" dev'
assert_contains "${PARITY_UP}" 'CASDOOR_BOOTSTRAP_ENABLED=true'
assert_contains "${PARITY_UP}" 'OPENFGA_BOOTSTRAP_API_URL="http://127\.0\.0\.1:8081"'
assert_contains "${PARITY_UP}" 'OPENFGA_BOOTSTRAP_DATABASE_URL="postgres://stuhelper_app:\$\{STUHELPER_APP_DB_PASSWORD\}@127\.0\.0\.1:\$\{PROD_PARITY_POSTGRES_PORT:-15432\}/stuhelper\?sslmode=disable"'
assert_contains "${PARITY_UP}" 'prod-parity-smoke.sh'
assert_contains "${PARITY_UP}" 'IDENTITY_SIGNING_PRIVATE_KEY_PEM=%s'
assert_contains "${PARITY_UP}" 'repo_default_path_matches'
assert_contains "${PARITY_UP}" 'install-local-prod-parity-ingress\.sh'
assert_not_contains "${PARITY_UP}" 'prod-deploy.sh'

assert_contains "${PARITY_LOCAL_INGRESS}" 'stuhelper\.com www\.stuhelper\.com id\.stuhelper\.com sso\.stuhelper\.com'
assert_contains "${PARITY_LOCAL_INGRESS}" 'PROXY_BYPASS_HOSTS'
assert_contains "${PARITY_LOCAL_INGRESS}" 'gsettings.*org\.gnome\.system\.proxy'
assert_contains "${PARITY_LOCAL_INGRESS}" '\*\.stuhelper\.com'
assert_contains "${PARITY_LOCAL_INGRESS}" 'DEFAULT_BAOTA_TLS_CERT'
assert_contains "${PARITY_LOCAL_INGRESS}" 'DEFAULT_GENERATED_TLS_CERT'
assert_contains "${PARITY_LOCAL_INGRESS}" 'ensure_local_tls'
assert_contains "${PARITY_LOCAL_INGRESS}" 'subjectAltName=DNS:stuhelper\.com,DNS:www\.stuhelper\.com,DNS:id\.stuhelper\.com,DNS:sso\.stuhelper\.com'
assert_contains "${PARITY_LOCAL_INGRESS}" 'listen 443 ssl'
assert_contains "${PARITY_LOCAL_INGRESS}" 'server_name sso\.stuhelper\.com'
assert_contains "${PARITY_LOCAL_INGRESS}" 'location = /account/profile'
assert_contains "${PARITY_LOCAL_INGRESS}" 'location = /connect'
assert_contains "${PARITY_LOCAL_INGRESS}" 'location = /account/security'
assert_contains "${PARITY_LOCAL_INGRESS}" 'nginx -t'
assert_contains "${PARITY_LOCAL_INGRESS}" 'nginx -s reload'
assert_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'server_name stuhelper\.com www\.stuhelper\.com'
assert_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'server_name id\.stuhelper\.com'
assert_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'server_name sso\.stuhelper\.com'
assert_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'return 301 https://stuhelper\.com\$request_uri'
assert_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'return 301 https://id\.stuhelper\.com\$request_uri'
assert_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'proxy_pass http://127\.0\.0\.1:__BACKEND_PORT__'
assert_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'proxy_pass http://127\.0\.0\.1:__CASDOOR_PORT__'
assert_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'return 302 \$scheme://id\.stuhelper\.com\$request_uri'
assert_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'location \^~ /api/v1/'
assert_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'location \^~ /login/oauth/'
assert_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'location \^~ /developers/'
assert_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'location = /identity'
assert_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'location = /account/profile'
assert_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'location = /connect'
assert_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'location = /account/security'
assert_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'location = /user/authorized-apps'
assert_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'return 302 /identity'
assert_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'return 302 \$scheme://stuhelper\.com\$request_uri'
assert_not_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'return 302 http://stuhelper\.com/developers/apps'
assert_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'location = /_app\.config\.js'
assert_contains "${PARITY_LOCAL_INGRESS_NGINX}" 'location \^~ /jse/'
assert_contains "${REPO_ROOT}/docker-compose.prod.yml" 'stuhelper\.com:host-gateway'
assert_contains "${REPO_ROOT}/docker-compose.prod.yml" 'id\.stuhelper\.com:host-gateway'
assert_contains "${REPO_ROOT}/docker-compose.prod.yml" 'sso\.stuhelper\.com:host-gateway'
assert_contains "${REPO_ROOT}/docker-compose.prod.yml" 'host\.docker\.internal:host-gateway'

assert_contains "${PARITY_DOWN}" 'parity_default_path'
assert_contains "${PARITY_DOWN}" 'repo_default_path_matches'
assert_contains "${PARITY_SMOKE}" 'prod-parity-datastore-smoke.sh'
assert_contains "${PARITY_SMOKE}" 'prod-parity-smoke-data.sh'
assert_contains "${PARITY_SMOKE}" 'identity-public-smoke.sh'
assert_contains "${PARITY_SMOKE}" 'parity_default_path'
assert_contains "${PARITY_SMOKE}" 'repo_default_path_matches'
assert_contains "${PARITY_SMOKE}" 'IDENTITY_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS=true'
assert_contains "${PARITY_SMOKE}" 'openfga-resource-access-smoke.sh'
assert_contains "${PARITY_SMOKE}" 'prod-parity-browser-smoke.sh'
assert_contains "${PARITY_SMOKE}" 'observability-smoke-check.sh'
assert_contains "${PARITY_SMOKE}" 'OBS_SMOKE_STRICT=false'

assert_contains "${PARITY_DATASTORE_SMOKE}" 'datastore-smoke-evidence\.json'
assert_contains "${PARITY_DATASTORE_SMOKE}" 'repo_default_path_matches'
assert_contains "${PARITY_DATASTORE_SMOKE}" 'has_database_privilege'
assert_contains "${PARITY_DATASTORE_SMOKE}" 'assert_pg_connect_allowed'
assert_contains "${PARITY_DATASTORE_SMOKE}" 'assert_pg_connect_denied'
assert_contains "${PARITY_DATASTORE_SMOKE}" 'Redis container must not join external datastore network'
assert_contains "${PARITY_DATASTORE_SMOKE}" 'Redis plaintext port must be disabled'
assert_contains "${PARITY_DATASTORE_SMOKE}" 'redis-cli --tls'
assert_contains "${PARITY_DATASTORE_SMOKE}" 'casdoorChecked'

assert_contains "${PARITY_SMOKE_DATA}" 'smoke-data-evidence\.json'
assert_contains "${PARITY_SMOKE_DATA}" 'repo_default_path_matches'
assert_contains "${PARITY_SMOKE_DATA}" 'refusing to seed non prod-parity PostgreSQL container'
assert_contains "${PARITY_SMOKE_DATA}" '生产等价课程'
assert_contains "${PARITY_SMOKE_DATA}" '生产等价教师'
assert_contains "${PARITY_SMOKE_DATA}" '生产等价评课'
assert_contains "${PARITY_SMOKE_DATA}" 'PROD_PARITY_ADMISSION_TOKEN'
assert_contains "${PARITY_SMOKE_DATA}" 'web_public_url'
assert_contains "${PARITY_SMOKE_DATA}" 'prod-parity-admission-session'
assert_contains "${PARITY_SMOKE_DATA}" 'admissionSessionCount'
assert_contains "${PARITY_SMOKE_DATA}" 'hmac\.new'
assert_contains "${PARITY_SMOKE_DATA}" 'REFRESH MATERIALIZED VIEW public\.mv_teacher_public_stats'
assert_contains "${PARITY_SMOKE_DATA}" "course:\\*"
assert_contains "${PARITY_SMOKE_DATA}" "review:\\*"
assert_contains "${PARITY_SMOKE_DATA}" 'courseRatingStatsCount'
assert_contains "${PARITY_SMOKE_DATA}" 'teacherPublicStatsCount'

assert_contains "${PARITY_BROWSER_SMOKE}" 'PROD_PARITY_BROWSER_SMOKE_EVIDENCE_FILE'
assert_contains "${PARITY_BROWSER_SMOKE}" 'repo_default_path_matches'
assert_contains "${PARITY_BROWSER_SMOKE}" 'browser-smoke-evidence\.json'
assert_contains "${PARITY_BROWSER_SMOKE}" 'WEB_BASE_URL'
assert_contains "${PARITY_BROWSER_SMOKE}" 'ADMIN_BASE_URL'
assert_contains "${PARITY_BROWSER_SMOKE}" 'clear_rate_limit_keys'
assert_contains "${PARITY_BROWSER_SMOKE}" 'prod-parity-smoke-data.sh'
assert_contains "${PARITY_BROWSER_SMOKE}" "scan --pattern 'rl:\*'"
assert_contains "${PARITY_BROWSER_SMOKE}" 'prod-parity-browser-smoke\.mjs'
assert_contains "${PARITY_UP}" 'API_IP_RATE_LIMIT'
assert_contains "${PARITY_UP}" 'API_GLOBAL_RATE_LIMIT'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" '@playwright/test'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'identityBaseURL'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'CASDOOR_PUBLIC_AUTH_BASE_URL'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'telemetryRoutePattern'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'viewportVariants'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" "name: 'desktop'"
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" "name: 'mobile'"
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'deviceScaleFactor'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'hasTouch'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'suppressedTelemetryRequests'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'stubbedResources'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'stubbedExternalResources'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'fonts.googleapis.com'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'cdn.casbin.org/flag-icons'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'allowedAPIResponses'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'ignoreHTTPSErrors: true'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'ignoredAPIResponses'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-home'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-login'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'identity-developer-login'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'identity-root-login'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'identity-connect-public'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-identity-connect-redirect'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'identity-account-profile-authenticated'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-protected-account-profile'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'identity-account-security-authenticated'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-protected-account-security'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'identity-home-authenticated'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'identity-authorized-apps-authenticated'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'identity-developer-apps-authenticated'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'identity-identity-verification-authenticated'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'identity-student-verification-authenticated'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'identity-phone-binding-authenticated'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'identity-qq-binding-authenticated'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'identity-academic-info-authenticated'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'identity-main-route-redirect'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-login-session-refresh'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'runWebLoginSessionRefreshFlow'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'runIdentityPortalShellFlow'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'runIdentityAuthenticatedRefreshFlow'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'expectIdentityAuthenticatedHeader'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'appShellHeader'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" "getByRole\\('banner'\\)"
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'identity header should not render the main-site notification bell'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'auth/me after browser refresh returned'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'auth/me after identity refresh returned'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-auth-callback-missing-code'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-admission-login'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-course-hub'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-course-list'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-course-detail'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-course-detail-reviews'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-review-feed'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-search'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-teacher-hub'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-teacher-profile'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-protected-review-post'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-protected-user-reviews'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-protected-user-votes'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-protected-user-favorites'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-protected-identity-home'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-protected-user-authorized-apps'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-protected-identity-verification'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-protected-student-verification'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-protected-phone-binding'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-protected-qq-binding'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-protected-academic-info'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-protected-notifications'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-protected-developer-apps'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-protected-open-platform-consent'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-protected-profile-completion'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'web-not-found'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'admin-login-redirect'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'checkName'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'viewportSize'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'finalURL'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'matchedText'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'requiredTexts'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'missingRequiredTexts'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'forbiddenTexts'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'presentForbiddenTexts'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'expectedURLIncludes'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'webRedirectQuery'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'toArray'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" "page\.on\('console'"
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'consoleErrors'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'ignoredConsoleErrors'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'describeConsoleMessage'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'isIgnoredConsoleError'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'browser\.newContext'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'pageerror'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'requestfailed'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" "page\.on\('response'"
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'response\.status\(\) >= 400'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" "'fetch', 'xhr'"
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'apiFailures'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" 'criticalResourceTypes'
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" "'image'"
assert_contains "${PARITY_BROWSER_SMOKE_NODE}" "'font'"

assert_not_contains "${ADMIN_INDEX_HTML}" 'hm\.baidu\.com'
assert_not_contains "${ADMIN_INDEX_HTML}" '_VBEN_ADMIN_PRO_APP_CONF_'

assert_contains "${WEB_NGINX}" 'location / \{'
assert_contains "${WEB_NGINX}" 'location /admission/ \{'
assert_contains "${WEB_NGINX}" 'try_files \$uri \$uri/ /index\.html'
assert_contains "${WEB_NGINX}" 'add_header Cache-Control "no-store, no-cache, must-revalidate" always;'
assert_contains "${WEB_NGINX}" 'location /assets/ \{'
assert_contains "${WEB_NGINX}" 'add_header Cache-Control "public, immutable";'

assert_contains "${MAKEFILE}" 'prod-parity-up'
assert_contains "${MAKEFILE}" 'prod-parity-ingress'
assert_contains "${MAKEFILE}" 'prod-parity-down'
assert_contains "${MAKEFILE}" 'prod-parity-smoke'
assert_contains "${MAKEFILE}" 'prod-parity-datastore-smoke'

echo "[prod-parity-contract] all assertions passed"
