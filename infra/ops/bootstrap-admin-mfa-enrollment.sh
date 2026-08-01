#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

usage() {
  cat <<'USAGE'
Usage:
  infra/ops/bootstrap-admin-mfa-enrollment.sh USER[,USER...]
  STUHELPER_ADMIN_MFA_BOOTSTRAP_USERS=USER[,USER...] infra/ops/bootstrap-admin-mfa-enrollment.sh

Break-glass operation for the initial privileged admin MFA bootstrap.

StuHelper Admin requires two facts for super_admin/school_admin users:
  1. StuHelper authorization_grants contains a desired privileged grant.
  2. StuHelper DB user_mfa_enrollment has active=true with an allowed method.

This script idempotently writes the StuHelper DB enrollment row only after it
confirms the StuHelper user has a PostgreSQL-managed super_admin or school_admin
grant and the corresponding Casdoor identity already has a
SMS/App/WebAuthn/TOTP MFA factor. It does not disable MFA, create a Casdoor MFA
factor, or modify Casdoor role membership.

Inputs:
  STUHELPER_ADMIN_MFA_BOOTSTRAP_USERS      Comma-separated Casdoor usernames.
  STUHELPER_ADMIN_MFA_BOOTSTRAP_ORG        Defaults to CASDOOR_ORGANIZATION or stuhelper.
  STUHELPER_ADMIN_MFA_BOOTSTRAP_METHOD     auto, sms, app, webauthn, or totp. Defaults to auto.
  STUHELPER_ADMIN_MFA_REQUIRE_CASDOOR_MFA  Defaults to true.
  STUHELPER_APP_CONTAINER                  Defaults to stuhelper-prod-app.
  STUHELPER_DB_CONTAINER                   Defaults to stuhelper-prod-postgres.
  CASDOOR_DB_CONTAINER                     Defaults to postgres.
  CASDOOR_DB_USER                          Defaults to casdoor.
  CASDOOR_DB_NAME                          Defaults to casdoor.

The script does not read or print secrets. It expects to run on a host where
both the StuHelper app/PostgreSQL containers and Casdoor PostgreSQL container
are available. After bootstrapping, the affected user must sign out and sign in
again through SSO with MFA so the current token carries a fresh MFA proof.
USAGE
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
  exit 0
fi

require_cmd docker

trim() {
  local value="$1"
  value="${value#"${value%%[![:space:]]*}"}"
  value="${value%"${value##*[![:space:]]}"}"
  printf '%s' "${value}"
}

validate_name() {
  local label="$1"
  local value="$2"
  [[ "${value}" =~ ^[A-Za-z0-9_.@-]+$ ]] || die "${label} contains unsupported characters"
}

users="${STUHELPER_ADMIN_MFA_BOOTSTRAP_USERS:-${1:-}}"
organization="${STUHELPER_ADMIN_MFA_BOOTSTRAP_ORG:-${CASDOOR_ORGANIZATION:-stuhelper}}"
method="${STUHELPER_ADMIN_MFA_BOOTSTRAP_METHOD:-auto}"
require_casdoor_mfa="${STUHELPER_ADMIN_MFA_REQUIRE_CASDOOR_MFA:-true}"
stuhelper_app_container="${STUHELPER_APP_CONTAINER:-stuhelper-prod-app}"
stuhelper_db_container="${STUHELPER_DB_CONTAINER:-stuhelper-prod-postgres}"
casdoor_db_container="${CASDOOR_DB_CONTAINER:-postgres}"
casdoor_db_user="${CASDOOR_DB_USER:-casdoor}"
casdoor_db_name="${CASDOOR_DB_NAME:-casdoor}"

[[ -n "${users//,/}" ]] || die "STUHELPER_ADMIN_MFA_BOOTSTRAP_USERS or positional USER[,USER...] is required"
validate_name "STUHELPER_ADMIN_MFA_BOOTSTRAP_ORG" "${organization}"
validate_name "STUHELPER_APP_CONTAINER" "${stuhelper_app_container}"
validate_name "STUHELPER_DB_CONTAINER" "${stuhelper_db_container}"
validate_name "CASDOOR_DB_CONTAINER" "${casdoor_db_container}"
validate_name "CASDOOR_DB_USER" "${casdoor_db_user}"
validate_name "CASDOOR_DB_NAME" "${casdoor_db_name}"

case "${method}" in
  auto|sms|app|webauthn|totp) ;;
  *) die "STUHELPER_ADMIN_MFA_BOOTSTRAP_METHOD must be auto, sms, app, webauthn, or totp" ;;
esac

case "${require_casdoor_mfa}" in
  true|false) ;;
  *) die "STUHELPER_ADMIN_MFA_REQUIRE_CASDOOR_MFA must be true or false" ;;
esac

database_url="$(docker exec "${stuhelper_app_container}" sh -lc 'printf %s "$DATABASE_URL"')"
[[ -n "${database_url}" ]] || die "DATABASE_URL is missing from ${stuhelper_app_container}"

stuhelper_sql() {
  docker exec \
    -e DATABASE_URL="${database_url}" \
    -i "${stuhelper_db_container}" \
    sh -lc 'psql -X -v ON_ERROR_STOP=1 -At -F "\t" "$DATABASE_URL" "$@"' \
    psql \
    "$@"
}

casdoor_sql() {
  docker exec -i "${casdoor_db_container}" \
    psql \
      -X \
      -v ON_ERROR_STOP=1 \
      -At \
      -F $'\t' \
      -U "${casdoor_db_user}" \
      -d "${casdoor_db_name}" \
      -v organization="${organization}" \
      "$@"
}

casdoor_mfa_method_for_user() {
  local target_user="$1"
  casdoor_sql -v target_user="${target_user}" <<'SQL'
WITH target_user AS (
  SELECT
    COALESCE(preferred_mfa_type, '') AS preferred_mfa_type,
    octet_length(COALESCE("webauthnCredentials", ''::bytea)) > 0 AS has_webauthn,
    COALESCE(totp_secret, '') AS totp_secret,
    COALESCE(mfa_phone_enabled, false) AS has_sms
  FROM public."user"
  WHERE owner = :'organization'
    AND name = :'target_user'
    AND COALESCE(is_deleted, false) = false
    AND COALESCE(is_forbidden, false) = false
)
SELECT CASE
  WHEN NOT EXISTS (SELECT 1 FROM target_user) THEN 'missing'
  WHEN EXISTS (
    SELECT 1
    FROM target_user
    WHERE lower(preferred_mfa_type) IN ('sms', 'phone')
      AND has_sms
  ) THEN 'sms'
  WHEN EXISTS (
    SELECT 1
    FROM target_user
    WHERE lower(preferred_mfa_type) IN ('app', 'totp')
      AND length(totp_secret) > 0
  ) THEN 'totp'
  WHEN EXISTS (
    SELECT 1
    FROM target_user
    WHERE lower(preferred_mfa_type) IN ('webauthn', 'fido', 'fido2', 'faceid')
      AND has_webauthn
  ) THEN 'webauthn'
  WHEN EXISTS (SELECT 1 FROM target_user WHERE has_sms) THEN 'sms'
  WHEN EXISTS (SELECT 1 FROM target_user WHERE has_webauthn) THEN 'webauthn'
  WHEN EXISTS (
    SELECT 1
    FROM target_user
    WHERE length(totp_secret) > 0
  ) THEN 'totp'
  ELSE 'none'
END;
SQL
}

stuhelper_privileged_grant_count() {
  local target_user="$1"
  stuhelper_sql -v target_user="${target_user}" <<'SQL'
SELECT count(*)
FROM public.authorization_grants grants
JOIN public.users target ON target.id = grants.subject_user_id
WHERE lower(target.username) = lower(:'target_user')
  AND grants.role IN ('super_admin', 'school_admin')
  AND grants.desired_state = 'granted';
SQL
}

stuhelper_user_count() {
  local target_user="$1"
  stuhelper_sql -v target_user="${target_user}" <<'SQL'
SELECT count(*)
FROM public.users
WHERE lower(username) = lower(:'target_user');
SQL
}

bootstrap_stuhelper_mfa() {
  local target_user="$1"
  local resolved_method="$2"

  stuhelper_sql -v target_user="${target_user}" -v mfa_method="${resolved_method}" <<'SQL'
WITH target AS (
  SELECT id, username
  FROM public.users
  WHERE lower(username) = lower(:'target_user')
),
previous AS (
  SELECT e.user_id, e.active, e.methods, e.reset_required
  FROM public.user_mfa_enrollment e
  JOIN target ON target.id = e.user_id
),
upserted AS (
  INSERT INTO public.user_mfa_enrollment (
    user_id,
    active,
    methods,
    reset_required,
    last_enrolled_at,
    last_disabled_at,
    updated_at
  )
  SELECT
    target.id,
    true,
    ARRAY[:'mfa_method']::text[],
    false,
    now(),
    NULL,
    now()
  FROM target
  ON CONFLICT (user_id) DO UPDATE SET
    active = true,
    methods = EXCLUDED.methods,
    reset_required = false,
    last_enrolled_at = now(),
    last_disabled_at = NULL,
    updated_at = now()
  RETURNING user_id, active, methods, reset_required
),
audited AS (
  INSERT INTO public.audit_events (
    id,
    category,
    event_type,
    actor_type,
    actor_username,
    action,
    resource_type,
    resource_id,
    before_data,
    after_data,
    result,
    reason,
    details,
    created_at
  )
  SELECT
    gen_random_uuid()::text,
    'admin_operation',
    'iam.mfa.bootstrap',
    'system',
    'infra/ops/bootstrap-admin-mfa-enrollment.sh',
    'bootstrap',
    'iam.mfa',
    target.id::text,
    COALESCE(
      (
        SELECT jsonb_build_object(
          'active', previous.active,
          'methods', previous.methods,
          'reset_required', previous.reset_required
        )
        FROM previous
      ),
      '{}'::jsonb
    ),
    jsonb_build_object(
      'active', upserted.active,
      'methods', upserted.methods,
      'reset_required', upserted.reset_required
    ),
    'success',
    'privileged admin mfa bootstrap',
    jsonb_build_object('target_username', target.username, 'method', :'mfa_method'),
    now()
  FROM target
  JOIN upserted ON upserted.user_id = target.id
  RETURNING id
)
SELECT
  target.id,
  target.username,
  upserted.active,
  array_to_string(upserted.methods, ','),
  (SELECT count(*) FROM audited) AS audit_events
FROM target
JOIN upserted ON upserted.user_id = target.id;
SQL
}

IFS=',' read -r -a requested_users <<<"${users}"
processed=0
for raw_user in "${requested_users[@]}"; do
  target_user="$(trim "${raw_user}")"
  [[ -n "${target_user}" ]] || continue
  validate_name "Casdoor username" "${target_user}"

  if [[ "$(stuhelper_user_count "${target_user}")" != "1" ]]; then
    die "StuHelper user ${target_user} was not found or is ambiguous"
  fi

  privileged_count="$(stuhelper_privileged_grant_count "${target_user}")"
  if [[ "${privileged_count}" -lt 1 ]]; then
    die "StuHelper user ${target_user} does not have a desired super_admin or school_admin authorization grant"
  fi

  casdoor_method="$(casdoor_mfa_method_for_user "${target_user}")"
  if [[ "${casdoor_method}" == "missing" ]]; then
    die "Casdoor user ${organization}/${target_user} was not found, deleted, or forbidden"
  fi
  if [[ "${require_casdoor_mfa}" == "true" && "${casdoor_method}" == "none" ]]; then
    die "Casdoor user ${organization}/${target_user} has no SMS/App/WebAuthn/TOTP MFA evidence; bind MFA in SSO first"
  fi

  resolved_method="${method}"
  if [[ "${resolved_method}" == "app" ]]; then
    resolved_method="totp"
  fi
  if [[ "${resolved_method}" == "auto" ]]; then
    resolved_method="${casdoor_method}"
    if [[ "${resolved_method}" == "none" ]]; then
      resolved_method="sms"
    fi
  fi
  requested_method="${method}"
  if [[ "${requested_method}" == "app" ]]; then
    requested_method="totp"
  fi
  if [[ "${require_casdoor_mfa}" == "true" && "${method}" != "auto" && "${requested_method}" != "${casdoor_method}" ]]; then
    die "requested method ${method} does not match Casdoor evidence ${casdoor_method} for ${organization}/${target_user}"
  fi

  result="$(bootstrap_stuhelper_mfa "${target_user}" "${resolved_method}")"
  log "bootstrapped StuHelper MFA enrollment for ${target_user}: ${result}"
  processed=$((processed + 1))
done

[[ "${processed}" -gt 0 ]] || die "no non-empty Casdoor usernames were provided"

log "affected users must sign out and sign in again through SSO with MFA"
