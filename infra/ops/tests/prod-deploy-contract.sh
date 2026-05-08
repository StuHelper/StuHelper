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
render_redis_acl_line="$(line_number 'render-redis-acl.sh')"
start_infra_line="$(line_number 'compose --profile prod up -d --wait "${infra_services[@]}"')"
start_authz_line="$(line_number 'compose --profile prod up -d --wait "${authz_services[@]}"')"
bootstrap_require_line="$(line_number 'require_nonempty CASDOOR_BOOTSTRAP_CLIENT_SECRET')"
bootstrap_reject_line="$(line_number 'reject_placeholder CASDOOR_BOOTSTRAP_CLIENT_SECRET')"
app_provisioning_require_line="$(line_number 'require_nonempty CASDOOR_APP_PROVISIONING_CLIENT_SECRET')"
app_provisioning_reject_line="$(line_number 'reject_placeholder CASDOOR_APP_PROVISIONING_CLIENT_SECRET')"
backup_endpoint_reject_line="$(line_number 'reject_placeholder BACKUP_OBJECT_STORAGE_ENDPOINT')"
introspection_require_line="$(line_number 'require_nonempty CASDOOR_INTROSPECTION_CLIENT_SECRET')"
role_sync_require_line="$(line_number 'require_nonempty CASDOOR_ROLE_SYNC_CLIENT_SECRET')"
user_lookup_require_line="$(line_number 'require_nonempty CASDOOR_USER_LOOKUP_CLIENT_SECRET')"
sms_secret_require_line="$(line_number 'require_nonempty SMS_SECRET_ID')"
casdoor_sms_enabled_line="$(line_number 'CASDOOR_SMS_PROVIDER_ENABLED must be true for production deploy')"
casdoor_sms_type_line="$(line_number 'CASDOOR_SMS_PROVIDER_TYPE must be CustomHTTP for production deploy')"
casdoor_sms_title_line="$(line_number 'CASDOOR_SMS_PROVIDER_TITLE must be content for production deploy')"
sms_enabled_line="$(line_number 'SMS_ENABLED must be true for production deploy')"

if (( source_bootstrap_line <= load_env_line )); then
  fail "Casdoor bootstrap env must be sourced after load_env"
fi
if (( bootstrap_require_line <= source_bootstrap_line )); then
  fail "Casdoor bootstrap credentials must be validated after sourcing bootstrap env"
fi
if (( bootstrap_reject_line <= bootstrap_require_line )); then
  fail "Casdoor bootstrap placeholder rejection must run after non-empty validation"
fi
if (( app_provisioning_reject_line <= app_provisioning_require_line )); then
  fail "Casdoor app-provisioning placeholder rejection must run after non-empty validation"
fi
if (( backup_endpoint_reject_line <= app_provisioning_reject_line )); then
  fail "backup object storage placeholder rejection must be part of production deploy validation"
fi
if (( introspection_require_line <= app_provisioning_require_line )); then
  fail "Casdoor introspection validation should be grouped after app-provisioning validation"
fi
if (( role_sync_require_line <= introspection_require_line )); then
  fail "Casdoor role-sync validation should be grouped after introspection validation"
fi
if (( user_lookup_require_line <= role_sync_require_line )); then
  fail "Casdoor user-lookup validation should be grouped after role-sync validation"
fi
if (( sms_secret_require_line <= user_lookup_require_line )); then
  fail "SMS runtime credentials must be validated after Casdoor service credentials"
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

if ! grep -qF 'reject_local_value BACKUP_OBJECT_STORAGE_ENDPOINT' "${PROD_DEPLOY_FILE}"; then
  fail "production deploy must reject local backup object storage endpoints"
fi

if (( render_redis_acl_line <= load_env_line )); then
  fail "render-redis-acl.sh must run after load_env so the latest REDIS_PASSWORD is available"
fi

if (( render_redis_acl_line >= start_infra_line )); then
  fail "render-redis-acl.sh must run before production infrastructure starts"
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
