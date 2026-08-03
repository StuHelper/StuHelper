#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
PROD_DEPLOY_FILE="${REPO_ROOT}/infra/ops/prod-deploy.sh"
PROD_ENV_EXAMPLE_FILE="${REPO_ROOT}/.env.prod.example"
ADMISSION_READINESS_FILE="${REPO_ROOT}/infra/ops/admission-production-readiness.sh"
AUTHORIZATION_CUTOVER_FILE="${REPO_ROOT}/infra/ops/authorization-ledger-cutover.sh"
LEGACY_SUPER_ADMIN_BOOTSTRAP_FILE="${REPO_ROOT}/infra/ops/authorization-bootstrap-super-admin.sh"

fail() {
  echo "[prod-deploy-contract][error] $*" >&2
  exit 1
}

assert_file_not_contains() {
  local file="$1"
  local pattern="$2"
  if grep -qF -- "${pattern}" "${file}"; then
    fail "legacy provider-managed super-admin bootstrap must stay removed from ${file}: ${pattern}"
  fi
}

line_number() {
  local pattern="$1"
  local line
  line="$(grep -nF -- "${pattern}" "${PROD_DEPLOY_FILE}" | head -n1 | cut -d: -f1)"
  [[ -n "${line}" ]] || fail "expected pattern in ${PROD_DEPLOY_FILE}: ${pattern}"
  printf '%s\n' "${line}"
}

load_env_line="$(line_number 'load_env')"
remote_config_load_line="$(line_number 'load_remote_deploy_config')"
generated_secret_ref_require_line="$(line_number 'GENERATED_ENV_SECRET_REF must be configured for production deploy')"
secret_backend_require_line="$(line_number 'production deploy requires a non-file secret backend for generated secrets')"
bootstrap_validation_line="$(line_number 'validate_casdoor_bootstrap_env # validate bootstrap credential env in isolated process')"
bootstrap_parent_clear_line="$(line_number 'clear_casdoor_bootstrap_env # remove any caller-provided bootstrap values from unrelated children')"
bootstrap_validator_clear_line="$(line_number '  clear_casdoor_bootstrap_env')"
bootstrap_validator_source_line="$(line_number 'source_casdoor_bootstrap_env # load the file only after inherited values are cleared')"
postgres_ssl_line="$(line_number 'require_production_postgres_ssl')"
external_student_source_security_line="$(line_number 'require_production_external_student_source_security')"
object_storage_gate_line="$(line_number 'require_production_object_storage')"
public_ingress_config_preflight_line="$(line_number 'require_public_ingress_config_preflight')"
public_ingress_preflight_line="$(line_number 'require_public_identity_ingress_preflight')"
render_postgres_tls_line="$(line_number 'render-postgres-tls.sh')"
render_redis_acl_line="$(line_number 'render-redis-acl.sh')"
start_infra_line="$(line_number 'compose --profile prod up -d --wait "${infra_services[@]}"')"
predeploy_backup_line="$(line_number '"${SCRIPT_DIR}/backup-postgres.sh" "${predeploy_backup_path}"')"
sync_backup_line="$(line_number '"${SCRIPT_DIR}/sync-postgres-backups.sh"')"
postgres_backup_evidence_line="$(line_number '"${SCRIPT_DIR}/postgres-backup-evidence.sh"')"
migrate_line="$(line_number 'compose --profile prod up --no-deps migrate')"
admission_readiness_line="$(line_number '"${SCRIPT_DIR}/admission-production-readiness.sh"')"
start_authz_line="$(line_number 'compose --profile prod up -d --wait "${authz_services[@]}"')"
bootstrap_platform_line="$(line_number '"${SCRIPT_DIR}/bootstrap-platform.sh" prod')"
authorization_cutover_line="$(line_number '"${SCRIPT_DIR}/authorization-ledger-cutover.sh" prod')"
open_platform_evidence_line="$(line_number '"${SCRIPT_DIR}/open-platform-production-evidence.sh"')"
start_app_line="$(line_number 'compose --profile prod up -d --wait app frontend admin')"
sso_public_smoke_line="$(line_number '"${SCRIPT_DIR}/sso-public-smoke.sh"')"
admission_public_smoke_line="$(line_number '"${SCRIPT_DIR}/admission-public-smoke.sh"')"
public_web_auth_browser_smoke_line="$(line_number 'node "${SCRIPT_DIR}/public-web-auth-browser-smoke.mjs"')"
smoke_check_line="$(line_number '"${SCRIPT_DIR}/smoke-check.sh"')"
observability_smoke_line="$(line_number 'OBS_SMOKE_STRICT=true "${SCRIPT_DIR}/observability-smoke-check.sh"')"
bootstrap_require_line="$(line_number 'require_nonempty CASDOOR_BOOTSTRAP_CLIENT_SECRET')"
grep -qF 'validate_casdoor_bootstrap_env() (' "${PROD_DEPLOY_FILE}" ||
  fail "production deploy must validate Casdoor bootstrap credentials in a subshell"
grep -qF '    CASDOOR_BOOTSTRAP_ORGANIZATION' "${PROD_DEPLOY_FILE}" ||
  fail "production deploy must clear every bootstrap-file key from the parent environment"
grep -qF 'source_casdoor_bootstrap_env_file "${file}"' "${PROD_DEPLOY_FILE}" ||
  fail "production deploy must parse the Casdoor bootstrap credential file through its allowlisted loader"
if grep -qF 'source "${file}"' "${PROD_DEPLOY_FILE}"; then
  fail "production deploy must not raw-source the Casdoor bootstrap credential file"
fi
casdoor_public_auth_require_line="$(line_number 'require_nonempty CASDOOR_PUBLIC_AUTH_BASE_URL')"
casdoor_public_auth_reject_line="$(line_number 'reject_placeholder CASDOOR_PUBLIC_AUTH_BASE_URL')"
casdoor_public_auth_local_reject_line="$(line_number 'reject_local_value CASDOOR_PUBLIC_AUTH_BASE_URL')"
web_url_require_line="$(line_number 'require_nonempty WEB_VITE_WEB_URL')"
web_url_reject_line="$(line_number 'reject_placeholder WEB_VITE_WEB_URL')"
web_url_local_reject_line="$(line_number 'reject_local_value WEB_VITE_WEB_URL')"
admission_public_base_require_line="$(line_number 'require_nonempty ADMISSION_PUBLIC_BASE_URL')"
freshman_material_hosts_require_line="$(line_number 'require_nonempty STUHELPER_FRESHMAN_MATERIAL_HOSTS')"
bootstrap_reject_line="$(line_number 'reject_placeholder CASDOOR_BOOTSTRAP_CLIENT_SECRET')"
app_provisioning_require_line="$(line_number 'require_nonempty CASDOOR_APP_PROVISIONING_CLIENT_SECRET')"
app_provisioning_reject_line="$(line_number 'reject_placeholder CASDOOR_APP_PROVISIONING_CLIENT_SECRET')"
backup_endpoint_reject_line="$(line_number 'reject_placeholder BACKUP_OBJECT_STORAGE_ENDPOINT')"
object_storage_access_reject_line="$(line_number 'reject_placeholder OBJECT_STORAGE_ACCESS_KEY_ID')"
object_storage_secret_reject_line="$(line_number 'reject_placeholder OBJECT_STORAGE_SECRET_ACCESS_KEY')"
object_storage_local_reject_line="$(line_number 'reject_local_value OBJECT_STORAGE_ENDPOINT')"
user_profile_require_line="$(line_number 'require_nonempty CASDOOR_USER_PROFILE_CLIENT_SECRET')"
introspection_require_line="$(line_number 'require_nonempty CASDOOR_INTROSPECTION_CLIENT_SECRET')"
user_lookup_require_line="$(line_number 'require_nonempty CASDOOR_USER_LOOKUP_CLIENT_SECRET')"
token_probe_smoke_client_require_line="$(line_number 'require_nonempty CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_ID')"
token_probe_smoke_secret_require_line="$(line_number 'require_nonempty CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_SECRET')"
token_probe_smoke_redirect_require_line="$(line_number 'require_nonempty CASDOOR_TOKEN_PROBE_SMOKE_REDIRECT_URI')"
token_probe_command_require_line="$(line_number 'require_nonempty OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND')"
open_platform_consent_require_line="$(line_number 'require_nonempty OPEN_PLATFORM_CONSENT_BASE_URL')"
open_platform_account_require_line="$(line_number 'require_nonempty OPEN_PLATFORM_ACCOUNT_BASE_URL')"
token_probe_username_require_line="$(line_number 'require_nonempty CASDOOR_TOKEN_PROBE_USERNAME')"
token_probe_browser_require_line="$(line_number 'require_nonempty CASDOOR_TOKEN_PROBE_BROWSER_EXECUTABLE_PATH')"
token_probe_smoke_client_reject_line="$(line_number 'reject_placeholder CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_ID')"
token_probe_smoke_secret_reject_line="$(line_number 'reject_placeholder CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_SECRET')"
token_probe_command_reject_line="$(line_number 'reject_placeholder OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND')"
open_platform_consent_reject_line="$(line_number 'reject_placeholder OPEN_PLATFORM_CONSENT_BASE_URL')"
open_platform_account_reject_line="$(line_number 'reject_placeholder OPEN_PLATFORM_ACCOUNT_BASE_URL')"
open_platform_consent_local_reject_line="$(line_number 'reject_local_value OPEN_PLATFORM_CONSENT_BASE_URL')"
open_platform_account_local_reject_line="$(line_number 'reject_local_value OPEN_PLATFORM_ACCOUNT_BASE_URL')"
token_probe_username_reject_line="$(line_number 'reject_placeholder CASDOOR_TOKEN_PROBE_USERNAME')"
sms_secret_require_line="$(line_number 'require_nonempty SMS_SECRET_ID')"
bot_service_token_require_line="$(line_number 'require_nonempty BOT_SERVICE_TOKEN')"
openfga_store_placeholder_reject_line="$(line_number 'reject_placeholder_if_set OPENFGA_STORE_ID')"
openfga_model_placeholder_reject_line="$(line_number 'reject_placeholder_if_set OPENFGA_MODEL_ID')"
bot_service_token_reject_line="$(line_number 'reject_placeholder BOT_SERVICE_TOKEN')"
admission_public_base_exact_line="$(line_number 'ADMISSION_PUBLIC_BASE_URL must be exactly https://join.stuhelper.com for production deploy')"
casdoor_sms_enabled_line="$(line_number 'CASDOOR_SMS_PROVIDER_ENABLED must be true for production deploy')"
casdoor_sms_type_line="$(line_number 'CASDOOR_SMS_PROVIDER_TYPE must be CustomHTTP for production deploy')"
casdoor_sms_title_line="$(line_number 'CASDOOR_SMS_PROVIDER_TITLE must be content for production deploy')"
sms_enabled_line="$(line_number 'SMS_ENABLED must be true for production deploy')"
token_probe_required_line="$(line_number 'OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_REQUIRED must be true for production deploy')"
admission_readiness_require_line="$(line_number 'require_nonempty ADMISSION_PRODUCTION_READINESS_ENABLED')"

[[ -f "${ADMISSION_READINESS_FILE}" ]] || fail "missing admission production readiness script"
bash -n "${ADMISSION_READINESS_FILE}"
if [[ ! -x "${ADMISSION_READINESS_FILE}" ]]; then
  fail "admission production readiness script must be executable"
fi
[[ -f "${AUTHORIZATION_CUTOVER_FILE}" ]] || fail "missing authorization ledger cutover script"
bash -n "${AUTHORIZATION_CUTOVER_FILE}"
if [[ ! -x "${AUTHORIZATION_CUTOVER_FILE}" ]]; then
  fail "authorization ledger cutover script must be executable"
fi

[[ -f "${PROD_ENV_EXAMPLE_FILE}" ]] || fail "missing production environment example"
[[ ! -e "${LEGACY_SUPER_ADMIN_BOOTSTRAP_FILE}" ]] ||
  fail "legacy manual super-admin bootstrap script must stay removed"
assert_file_not_contains "${PROD_DEPLOY_FILE}" 'STUHELPER_INITIAL_SUPER_ADMINS'
assert_file_not_contains "${PROD_DEPLOY_FILE}" 'authorization-bootstrap-super-admin.sh'
assert_file_not_contains "${PROD_ENV_EXAMPLE_FILE}" 'STUHELPER_INITIAL_SUPER_ADMINS'

if (( bootstrap_validation_line <= load_env_line )); then
  fail "Casdoor bootstrap env must be validated after load_env"
fi
if (( generated_secret_ref_require_line <= remote_config_load_line )); then
  fail "production deploy must load remote.env before requiring GENERATED_ENV_SECRET_REF"
fi
if (( secret_backend_require_line <= remote_config_load_line )); then
  fail "production deploy must load remote.env before requiring SECRET_BACKEND"
fi
if (( bootstrap_require_line >= bootstrap_validation_line )); then
  fail "Casdoor bootstrap credential requirements must execute inside the isolated validator"
fi
if (( bootstrap_validator_clear_line >= bootstrap_validator_source_line )); then
  fail "Casdoor bootstrap validation must clear inherited values before reading the credential file"
fi
if (( bootstrap_parent_clear_line <= bootstrap_validation_line )); then
  fail "Casdoor bootstrap credentials must be removed from the parent after validation"
fi
if (( casdoor_public_auth_require_line <= bootstrap_validation_line )); then
  fail "Casdoor public auth base URL must be validated with identity ingress settings"
fi
if (( casdoor_public_auth_reject_line <= casdoor_public_auth_require_line )); then
  fail "Casdoor public auth base URL placeholder rejection must run after non-empty validation"
fi
if (( casdoor_public_auth_local_reject_line <= casdoor_public_auth_reject_line )); then
  fail "Casdoor public auth base URL local endpoint rejection must run after placeholder rejection"
fi
if (( web_url_require_line <= casdoor_public_auth_require_line )); then
  fail "Web public frontend URL validation should be grouped after SSO public auth validation"
fi
if (( web_url_reject_line <= web_url_require_line )); then
  fail "Web public frontend URL placeholder rejection must run after non-empty validation"
fi
if (( web_url_local_reject_line <= web_url_reject_line )); then
  fail "Web public frontend URL local endpoint rejection must run after placeholder rejection"
fi
if (( admission_readiness_require_line <= admission_public_base_require_line )); then
  fail "admission production readiness flag must be validated with admission public URL settings"
fi
if (( freshman_material_hosts_require_line <= admission_readiness_require_line )); then
  fail "freshman material host validation must stay grouped after admission readiness flag validation"
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
if (( object_storage_secret_reject_line <= object_storage_access_reject_line ||
      object_storage_secret_reject_line >= backup_endpoint_reject_line )); then
  fail "application object-storage secret placeholder rejection must follow its access key and precede backup validation"
fi
if (( object_storage_local_reject_line <= object_storage_secret_reject_line )); then
  fail "object-storage local endpoint rejection must run after placeholder rejection"
fi
if (( object_storage_gate_line <= object_storage_local_reject_line )); then
  fail "HTTPS and TLS object-storage gates must run after endpoint placeholder/local checks"
fi
if (( introspection_require_line <= user_profile_require_line )); then
  fail "Casdoor introspection validation should be grouped after user-profile validation"
fi
if (( user_lookup_require_line <= introspection_require_line )); then
  fail "Casdoor user-lookup validation should be grouped after introspection validation"
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
if (( open_platform_consent_require_line <= token_probe_command_require_line )); then
  fail "Open Platform consent base URL validation should run after token probe command validation"
fi
if (( open_platform_account_require_line <= open_platform_consent_require_line )); then
  fail "Open Platform account base URL validation should run after consent base URL validation"
fi
if (( token_probe_username_require_line <= open_platform_account_require_line )); then
  fail "bundled token probe credentials must be validated after command validation"
fi
if (( token_probe_browser_require_line <= token_probe_username_require_line )); then
  fail "bundled token probe browser path must be validated after bundled credentials"
fi
if (( token_probe_command_reject_line <= token_probe_command_require_line )); then
  fail "Open Platform token probe command placeholder rejection must run after non-empty validation"
fi
if (( open_platform_consent_reject_line <= open_platform_consent_require_line )); then
  fail "Open Platform consent base URL placeholder rejection must run after non-empty validation"
fi
if (( open_platform_account_reject_line <= open_platform_account_require_line )); then
  fail "Open Platform account base URL placeholder rejection must run after non-empty validation"
fi
if (( open_platform_consent_local_reject_line <= open_platform_consent_reject_line )); then
  fail "Open Platform consent base URL local endpoint rejection must run after placeholder rejection"
fi
if (( open_platform_account_local_reject_line <= open_platform_account_reject_line )); then
  fail "Open Platform account base URL local endpoint rejection must run after placeholder rejection"
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
if (( bot_service_token_require_line <= sms_secret_require_line )); then
  fail "BOT_SERVICE_TOKEN validation must be grouped after SMS runtime credentials"
fi
if (( openfga_store_placeholder_reject_line <= bot_service_token_require_line )); then
  fail "OpenFGA store ID placeholder rejection must run after required runtime credentials"
fi
if (( openfga_model_placeholder_reject_line <= openfga_store_placeholder_reject_line )); then
  fail "OpenFGA model ID placeholder rejection must run after store ID placeholder rejection"
fi
if (( bot_service_token_reject_line <= bot_service_token_require_line )); then
  fail "BOT_SERVICE_TOKEN placeholder rejection must run after non-empty validation"
fi
if (( casdoor_sms_enabled_line <= sms_secret_require_line )); then
  fail "Casdoor SMS provider production gate must run after SMS credentials are validated"
fi
if (( admission_public_base_exact_line <= sms_secret_require_line )); then
  fail "Admission public base URL exact production gate must run after required credentials are validated"
fi
if (( casdoor_sms_enabled_line <= admission_public_base_exact_line )); then
  fail "Casdoor SMS provider production gate must run after admission public URL exact gate"
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
if (( postgres_ssl_line <= object_storage_gate_line )); then
  fail "production PostgreSQL SSL gate must run after production object-storage TLS validation"
fi
if (( postgres_ssl_line <= token_probe_required_line )); then
  fail "production PostgreSQL SSL gate must run after production runtime feature gates"
fi
if (( external_student_source_security_line <= postgres_ssl_line )); then
  fail "external student source TLS validation must run after PostgreSQL validation"
fi
if (( public_ingress_config_preflight_line <= external_student_source_security_line )); then
  fail "external student source TLS validation must run before public ingress checks"
fi
if (( public_ingress_config_preflight_line <= postgres_ssl_line )); then
  fail "public Nginx ingress config preflight must run after production PostgreSQL SSL config validation"
fi
if (( public_ingress_preflight_line <= public_ingress_config_preflight_line )); then
  fail "public SSO/admission ingress preflight must run after local Nginx config preflight"
fi
if (( admission_public_smoke_line <= sso_public_smoke_line )); then
  fail "admission public smoke must run after SSO public smoke"
fi
if (( public_web_auth_browser_smoke_line <= admission_public_smoke_line )); then
  fail "public Web auth browser smoke must run after admission public smoke"
fi
if (( smoke_check_line <= public_web_auth_browser_smoke_line )); then
  fail "smoke-check must run after public Web auth browser smoke"
fi
if (( public_ingress_preflight_line <= postgres_ssl_line )); then
  fail "public SSO/admission ingress preflight must run after production PostgreSQL SSL config validation"
fi
if (( render_postgres_tls_line <= public_ingress_preflight_line )); then
  fail "render-postgres-tls.sh must run after public SSO/admission ingress preflight"
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
if grep -qF 'external PostgreSQL plaintext transport is explicitly enabled' "${PROD_DEPLOY_FILE}"; then
  fail "production deploy must not retain a plaintext PostgreSQL bypass"
fi
if ! grep -qF 'EXTERNAL_POSTGRES_ENABLED' "${PROD_DEPLOY_FILE}"; then
  fail "production deploy must support skipping the internal PostgreSQL service"
fi
if ! grep -qF 'if [[ "${EXTERNAL_POSTGRES_ENABLED:-false}" != "true" ]]; then' "${PROD_DEPLOY_FILE}"; then
  fail "production deploy must scope internal PostgreSQL superuser validation to internal mode"
fi
if ! grep -qF 'require_nonempty POSTGRES_PASSWORD' "${PROD_DEPLOY_FILE}"; then
  fail "production deploy must still require a superuser password for internal PostgreSQL"
fi
for image_var in BACKEND_IMAGE_REF FRONTEND_IMAGE_REF ADMIN_IMAGE_REF; do
  grep -qF "require_digest_image_ref ${image_var}" "${PROD_DEPLOY_FILE}" ||
    fail "production deploy must require a digest-pinned ${image_var}"
done
if grep -qF 'require_immutable_image_ref' "${PROD_DEPLOY_FILE}"; then
  fail "production deploy must not retain the tag-tolerant image reference validator"
fi
if ! grep -qF 'require_public_identity_ingress_preflight' "${PROD_DEPLOY_FILE}"; then
  fail "production deploy must fail fast on missing public web, SSO, and admission ingress"
fi
if ! grep -qF 'require_public_ingress_config_preflight' "${PROD_DEPLOY_FILE}"; then
  fail "production deploy must fail fast on missing local Nginx public ingress config"
fi

if (( render_redis_acl_line <= load_env_line )); then
  fail "render-redis-acl.sh must run after load_env so the latest REDIS_PASSWORD is available"
fi
grep -qF 'require_nonempty REDIS_EXPORTER_PASSWORD' "${PROD_DEPLOY_FILE}" ||
  fail "production deploy must require the dedicated Redis exporter password"
grep -qF 'REDIS_EXPORTER_PASSWORD must be independent from REDIS_PASSWORD' "${PROD_DEPLOY_FILE}" ||
  fail "production deploy must reject Redis application/exporter password reuse"
external_redis_pattern="EXTERNAL_""REDIS"
if grep -qF "${external_redis_pattern}" "${PROD_DEPLOY_FILE}"; then
  fail "production deploy must not support external Redis for the production StuHelper app"
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
if (( admission_readiness_line <= migrate_line )); then
  fail "admission production readiness must run after database migrations"
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
if (( start_authz_line <= admission_readiness_line )); then
  fail "authorization services must start after admission production readiness passes"
fi
if (( open_platform_evidence_line <= bootstrap_platform_line )); then
  fail "Open Platform production evidence smokes must run after bootstrap-platform creates Casdoor smoke app and writes OpenFGA IDs"
fi
if (( authorization_cutover_line <= bootstrap_platform_line )); then
  fail "authorization ledger cutover must run after Casdoor/OpenFGA bootstrap"
fi
if (( open_platform_evidence_line <= authorization_cutover_line )); then
  fail "Open Platform production evidence must wait until the authorization ledger cutover is sealed"
fi
if (( start_app_line <= open_platform_evidence_line )); then
  fail "application services must start after Open Platform production evidence smokes pass"
fi
if (( sso_public_smoke_line <= start_app_line )); then
  fail "public SSO smoke must run after production application services start"
fi
if (( public_web_auth_browser_smoke_line <= admission_public_smoke_line )); then
  fail "public Web auth browser smoke must run after admission public smoke"
fi
if (( smoke_check_line <= public_web_auth_browser_smoke_line )); then
  fail "business smoke-check must run after public Web auth browser smoke"
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

if (( start_app_line <= admission_readiness_line )); then
  fail "application services must start after admission production readiness passes"
fi

echo "[prod-deploy-contract] all assertions passed"
