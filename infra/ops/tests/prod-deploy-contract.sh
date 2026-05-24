#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
PROD_DEPLOY_FILE="${REPO_ROOT}/infra/ops/prod-deploy.sh"

fail() {
  echo "[prod-deploy-contract][error] $*" >&2
  exit 1
}

line_number() {
  local pattern="$1"
  local line
  line="$(grep -nF -- "${pattern}" "${PROD_DEPLOY_FILE}" | head -n1 | cut -d: -f1)"
  [[ -n "${line}" ]] || fail "expected pattern in ${PROD_DEPLOY_FILE}: ${pattern}"
  printf '%s\n' "${line}"
}

load_env_line="$(line_number 'load_env')"
source_bootstrap_line="$(line_number 'source_casdoor_bootstrap_env # load bootstrap credential env')"
postgres_ssl_line="$(line_number 'require_production_postgres_ssl')"
public_ingress_config_preflight_line="$(line_number 'require_public_ingress_config_preflight')"
public_ingress_preflight_line="$(line_number 'require_public_identity_ingress_preflight')"
render_postgres_tls_line="$(line_number 'render-postgres-tls.sh')"
render_redis_acl_line="$(line_number 'render-redis-acl.sh')"
start_infra_line="$(line_number 'compose --profile prod up -d --wait "${infra_services[@]}"')"
predeploy_backup_line="$(line_number '"${SCRIPT_DIR}/backup-postgres.sh" "${predeploy_backup_path}"')"
sync_backup_line="$(line_number '"${SCRIPT_DIR}/sync-postgres-backups.sh"')"
postgres_backup_evidence_line="$(line_number '"${SCRIPT_DIR}/postgres-backup-evidence.sh"')"
migrate_line="$(line_number 'compose --profile prod up --no-deps migrate')"
start_authz_line="$(line_number 'compose --profile prod up -d --wait "${authz_services[@]}"')"
bootstrap_platform_line="$(line_number '"${SCRIPT_DIR}/bootstrap-platform.sh" prod')"
open_platform_evidence_line="$(line_number '"${SCRIPT_DIR}/open-platform-production-evidence.sh"')"
start_app_line="$(line_number 'compose --profile prod up -d --wait app frontend admin')"
identity_public_smoke_bootstrap_line="$(line_number '"${SCRIPT_DIR}/bootstrap-identity-public-smoke-client.sh"')"
identity_public_smoke_reload_line="$(line_number 'load_env # reload Identity public smoke credentials')"
identity_public_smoke_line="$(line_number '"${SCRIPT_DIR}/identity-public-smoke.sh"')"
smoke_check_line="$(line_number '"${SCRIPT_DIR}/smoke-check.sh"')"
observability_smoke_line="$(line_number 'OBS_SMOKE_STRICT=true "${SCRIPT_DIR}/observability-smoke-check.sh"')"
bootstrap_require_line="$(line_number 'require_nonempty CASDOOR_BOOTSTRAP_CLIENT_SECRET')"
identity_issuer_require_line="$(line_number 'require_nonempty IDENTITY_ISSUER')"
identity_issuer_reject_line="$(line_number 'reject_placeholder IDENTITY_ISSUER')"
identity_issuer_local_reject_line="$(line_number 'reject_local_value IDENTITY_ISSUER')"
bootstrap_reject_line="$(line_number 'reject_placeholder CASDOOR_BOOTSTRAP_CLIENT_SECRET')"
app_provisioning_require_line="$(line_number 'require_nonempty CASDOOR_APP_PROVISIONING_CLIENT_SECRET')"
app_provisioning_reject_line="$(line_number 'reject_placeholder CASDOOR_APP_PROVISIONING_CLIENT_SECRET')"
backup_endpoint_reject_line="$(line_number 'reject_placeholder BACKUP_OBJECT_STORAGE_ENDPOINT')"
user_profile_require_line="$(line_number 'require_nonempty CASDOOR_USER_PROFILE_CLIENT_SECRET')"
introspection_require_line="$(line_number 'require_nonempty CASDOOR_INTROSPECTION_CLIENT_SECRET')"
role_sync_require_line="$(line_number 'require_nonempty CASDOOR_ROLE_SYNC_CLIENT_SECRET')"
user_lookup_require_line="$(line_number 'require_nonempty CASDOOR_USER_LOOKUP_CLIENT_SECRET')"
token_probe_smoke_client_require_line="$(line_number 'require_nonempty CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_ID')"
token_probe_smoke_secret_require_line="$(line_number 'require_nonempty CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_SECRET')"
token_probe_smoke_redirect_require_line="$(line_number 'require_nonempty CASDOOR_TOKEN_PROBE_SMOKE_REDIRECT_URI')"
token_probe_command_require_line="$(line_number 'require_nonempty OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND')"
token_probe_username_require_line="$(line_number 'require_nonempty CASDOOR_TOKEN_PROBE_USERNAME')"
token_probe_browser_require_line="$(line_number 'require_nonempty CASDOOR_TOKEN_PROBE_BROWSER_EXECUTABLE_PATH')"
token_probe_smoke_client_reject_line="$(line_number 'reject_placeholder CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_ID')"
token_probe_smoke_secret_reject_line="$(line_number 'reject_placeholder CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_SECRET')"
token_probe_command_reject_line="$(line_number 'reject_placeholder OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND')"
token_probe_username_reject_line="$(line_number 'reject_placeholder CASDOOR_TOKEN_PROBE_USERNAME')"
sms_secret_require_line="$(line_number 'require_nonempty SMS_SECRET_ID')"
casdoor_sms_enabled_line="$(line_number 'CASDOOR_SMS_PROVIDER_ENABLED must be true for production deploy')"
casdoor_sms_type_line="$(line_number 'CASDOOR_SMS_PROVIDER_TYPE must be CustomHTTP for production deploy')"
casdoor_sms_title_line="$(line_number 'CASDOOR_SMS_PROVIDER_TITLE must be content for production deploy')"
sms_enabled_line="$(line_number 'SMS_ENABLED must be true for production deploy')"
token_probe_required_line="$(line_number 'OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_REQUIRED must be true for production deploy')"

if (( source_bootstrap_line <= load_env_line )); then
  fail "Casdoor bootstrap env must be sourced after load_env"
fi
if (( bootstrap_require_line <= source_bootstrap_line )); then
  fail "Casdoor bootstrap credentials must be validated after sourcing bootstrap env"
fi
if (( identity_issuer_require_line <= source_bootstrap_line )); then
  fail "Identity issuer must be validated after env files and bootstrap env are loaded"
fi
if (( identity_issuer_reject_line <= identity_issuer_require_line )); then
  fail "Identity issuer placeholder rejection must run after non-empty validation"
fi
if (( identity_issuer_local_reject_line <= identity_issuer_reject_line )); then
  fail "Identity issuer local endpoint rejection must run after placeholder rejection"
fi
if (( bootstrap_reject_line <= bootstrap_require_line )); then
  fail "Casdoor bootstrap placeholder rejection must run after non-empty validation"
fi
if (( app_provisioning_reject_line <= app_provisioning_require_line )); then
  fail "Casdoor app-provisioning placeholder rejection must run after non-empty validation"
fi
if (( user_profile_require_line <= app_provisioning_require_line )); then
  fail "Casdoor user-profile validation should be grouped after app-provisioning validation"
fi
if (( backup_endpoint_reject_line <= app_provisioning_reject_line )); then
  fail "backup object storage placeholder rejection must be part of production deploy validation"
fi
if (( introspection_require_line <= user_profile_require_line )); then
  fail "Casdoor introspection validation should be grouped after user-profile validation"
fi
if (( role_sync_require_line <= introspection_require_line )); then
  fail "Casdoor role-sync validation should be grouped after introspection validation"
fi
if (( user_lookup_require_line <= role_sync_require_line )); then
  fail "Casdoor user-lookup validation should be grouped after role-sync validation"
fi
if (( token_probe_smoke_client_require_line <= user_lookup_require_line )); then
  fail "Casdoor token probe smoke app validation should be grouped after user-lookup validation"
fi
if (( token_probe_smoke_secret_require_line <= token_probe_smoke_client_require_line )); then
  fail "Casdoor token probe smoke secret validation should run after smoke client validation"
fi
if (( token_probe_smoke_redirect_require_line <= token_probe_smoke_secret_require_line )); then
  fail "Casdoor token probe smoke redirect validation should run after smoke secret validation"
fi
if (( token_probe_command_require_line <= token_probe_smoke_redirect_require_line )); then
  fail "Open Platform token probe command validation should run after Casdoor service credentials"
fi
if (( token_probe_username_require_line <= token_probe_command_require_line )); then
  fail "bundled token probe credentials must be validated after command validation"
fi
if (( token_probe_browser_require_line <= token_probe_username_require_line )); then
  fail "bundled token probe browser path must be validated after bundled credentials"
fi
if (( token_probe_command_reject_line <= token_probe_command_require_line )); then
  fail "Open Platform token probe command placeholder rejection must run after non-empty validation"
fi
if (( token_probe_username_reject_line <= token_probe_username_require_line )); then
  fail "bundled token probe credential placeholder rejection must run after non-empty validation"
fi
if (( token_probe_smoke_client_reject_line <= token_probe_smoke_client_require_line )); then
  fail "Casdoor token probe smoke client placeholder rejection must run after non-empty validation"
fi
if (( token_probe_smoke_secret_reject_line <= token_probe_smoke_secret_require_line )); then
  fail "Casdoor token probe smoke secret placeholder rejection must run after non-empty validation"
fi
if (( sms_secret_require_line <= token_probe_browser_require_line )); then
  fail "SMS runtime credentials must be validated after Open Platform token probe configuration"
fi
if (( casdoor_sms_enabled_line <= sms_secret_require_line )); then
  fail "Casdoor SMS provider production gate must run after SMS credentials are validated"
fi
if (( casdoor_sms_type_line <= casdoor_sms_enabled_line )); then
  fail "Casdoor SMS provider type gate must run after provider enabled gate"
fi
if (( casdoor_sms_title_line <= casdoor_sms_type_line )); then
  fail "Casdoor SMS provider title gate must run after provider type gate"
fi
if (( sms_enabled_line <= sms_secret_require_line )); then
  fail "SMS_ENABLED production gate must run after SMS credentials are validated"
fi
if (( token_probe_required_line <= sms_enabled_line )); then
  fail "Open Platform runtime token probe required gate must run after SMS production gate"
fi
if (( postgres_ssl_line <= token_probe_required_line )); then
  fail "production PostgreSQL SSL gate must run after production runtime feature gates"
fi
if (( public_ingress_config_preflight_line <= postgres_ssl_line )); then
  fail "public Nginx ingress config preflight must run after production PostgreSQL SSL config validation"
fi
if (( public_ingress_preflight_line <= public_ingress_config_preflight_line )); then
  fail "public identity ingress preflight must run after local Nginx config preflight"
fi
if (( public_ingress_preflight_line <= postgres_ssl_line )); then
  fail "public identity ingress preflight must run after production PostgreSQL SSL config validation"
fi
if (( render_postgres_tls_line <= public_ingress_preflight_line )); then
  fail "render-postgres-tls.sh must run after public identity ingress preflight"
fi
if (( render_postgres_tls_line <= postgres_ssl_line )); then
  fail "render-postgres-tls.sh must run after production PostgreSQL SSL config validation"
fi
if (( render_redis_acl_line <= postgres_ssl_line )); then
  fail "render-redis-acl.sh must run after production PostgreSQL SSL config validation"
fi

if ! grep -qF 'reject_local_value BACKUP_OBJECT_STORAGE_ENDPOINT' "${PROD_DEPLOY_FILE}"; then
  fail "production deploy must reject local backup object storage endpoints"
fi
if ! grep -qF 'reject_local_value CASDOOR_TOKEN_PROBE_SMOKE_REDIRECT_URI' "${PROD_DEPLOY_FILE}"; then
  fail "production deploy must reject local token probe smoke redirect URIs"
fi
if ! grep -qF 'casdoor-runtime-token-probe-runner.mjs' "${PROD_DEPLOY_FILE}"; then
  fail "production deploy must require bundled token probe credentials when the bundled runner is configured"
fi
if ! grep -qF 'require_production_postgres_ssl' "${PROD_DEPLOY_FILE}"; then
  fail "production deploy must fail fast on insecure internal PostgreSQL SSL settings"
fi
if ! grep -qF 'require_public_identity_ingress_preflight' "${PROD_DEPLOY_FILE}"; then
  fail "production deploy must fail fast on missing public identity ingress TLS and Casdoor discovery"
fi
if ! grep -qF 'require_public_ingress_config_preflight' "${PROD_DEPLOY_FILE}"; then
  fail "production deploy must fail fast on missing local Nginx public ingress config"
fi

if (( render_redis_acl_line <= load_env_line )); then
  fail "render-redis-acl.sh must run after load_env so the latest REDIS_PASSWORD is available"
fi

if (( render_redis_acl_line >= start_infra_line )); then
  fail "render-redis-acl.sh must run before production infrastructure starts"
fi
if (( predeploy_backup_line <= start_infra_line )); then
  fail "pre-deploy database backup must run after production infrastructure starts"
fi
if (( sync_backup_line <= predeploy_backup_line )); then
  fail "pre-deploy backup artifacts must sync to object storage after backup completes"
fi
if (( postgres_backup_evidence_line <= sync_backup_line )); then
  fail "PostgreSQL backup evidence must run after backup artifacts sync"
fi
if (( migrate_line <= postgres_backup_evidence_line )); then
  fail "database migrations must wait until PostgreSQL backup evidence passes"
fi

authz_block="$(
  awk '
    /^authz_services=\(/ { in_block=1; next }
    /^\)/ && in_block { in_block=0 }
    in_block { print }
  ' "${PROD_DEPLOY_FILE}"
)"

[[ -n "${authz_block}" ]] || fail "expected authz_services block in ${PROD_DEPLOY_FILE}"
if printf '%s\n' "${authz_block}" | grep -Eq '(^|[[:space:]])casdoor($|[[:space:]])'; then
  fail "production deploy must not start the local casdoor service; SSO is external"
fi
if printf '%s\n' "${authz_block}" | grep -Eq '(^|[[:space:]])proxy($|[[:space:]])'; then
  fail "production deploy must not start Traefik when Baota/Nginx owns public ingress"
fi
if ! printf '%s\n' "${authz_block}" | grep -Eq '(^|[[:space:]])openfga($|[[:space:]])'; then
  fail "production authz services must still start OpenFGA"
fi
if (( start_authz_line <= start_infra_line )); then
  fail "authorization services must start after infrastructure services"
fi
if (( open_platform_evidence_line <= bootstrap_platform_line )); then
  fail "Open Platform production evidence smokes must run after bootstrap-platform creates Casdoor smoke app and writes OpenFGA IDs"
fi
if (( start_app_line <= open_platform_evidence_line )); then
  fail "application services must start after Open Platform production evidence smokes pass"
fi
if (( identity_public_smoke_line <= start_app_line )); then
  fail "public identity smoke must run after production application services start"
fi
if (( identity_public_smoke_bootstrap_line <= start_app_line )); then
  fail "Identity public smoke client bootstrap must run after production application services start"
fi
if (( identity_public_smoke_reload_line <= identity_public_smoke_bootstrap_line )); then
  fail "production deploy must reload env after bootstrapping Identity public smoke credentials"
fi
if (( identity_public_smoke_line <= identity_public_smoke_reload_line )); then
  fail "public identity smoke must run after optional Identity public smoke client bootstrap reloads env"
fi
if (( smoke_check_line <= identity_public_smoke_line )); then
  fail "business smoke-check must run after public identity smoke"
fi
if (( observability_smoke_line <= smoke_check_line )); then
  fail "strict observability smoke must run after business smoke-check"
fi

infra_block="$(
  awk '
    /^infra_services=\(/ { in_block=1; next }
    /^\)/ && in_block { in_block=0 }
    in_block { print }
  ' "${PROD_DEPLOY_FILE}"
)"

[[ -n "${infra_block}" ]] || fail "expected infra_services block in ${PROD_DEPLOY_FILE}"
if ! printf '%s\n' "${infra_block}" | grep -Eq '(^|[[:space:]])cadvisor($|[[:space:]])'; then
  fail "production infra services must include cadvisor because Prometheus scrapes it"
fi

echo "[prod-deploy-contract] all assertions passed"
