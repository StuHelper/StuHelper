#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT_GUESS="$(cd "${SCRIPT_DIR}/../.." && pwd)"

if [[ -z "${ENV_FILE+x}" && -f "${REPO_ROOT_GUESS}/.env.prod.shared" ]]; then
  export ENV_FILE="${REPO_ROOT_GUESS}/.env.prod.shared"
fi
if [[ -z "${SECRETS_ENV_FILE+x}" ]]; then
  if [[ -f "${REPO_ROOT_GUESS}/.env.prod.secrets.local" ]]; then
    export SECRETS_ENV_FILE="${REPO_ROOT_GUESS}/.env.prod.secrets.local"
  elif [[ -f "${REPO_ROOT_GUESS}/.env.prod.secrets" ]]; then
    export SECRETS_ENV_FILE="${REPO_ROOT_GUESS}/.env.prod.secrets"
  fi
fi
if [[ -z "${GENERATED_ENV_FILE+x}" && -f "${REPO_ROOT_GUESS}/.env.prod.generated" ]]; then
  export GENERATED_ENV_FILE="${REPO_ROOT_GUESS}/.env.prod.generated"
fi
if [[ -z "${GENERATED_SECRET_ENV_FILE+x}" && -f "${REPO_ROOT_GUESS}/.env.prod.generated.secrets" ]]; then
  export GENERATED_SECRET_ENV_FILE="${REPO_ROOT_GUESS}/.env.prod.generated.secrets"
fi

# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

require_cmd docker
require_cmd python3

load_env
export COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-${STACK_NAME:-stuhelper}}"

case "${ADMISSION_PRODUCTION_READINESS_ENABLED:-true}" in
  true|TRUE|1|yes|YES) ;;
  false|FALSE|0|no|NO)
    warn "admission production readiness skipped because ADMISSION_PRODUCTION_READINESS_ENABLED is not true"
    exit 0
    ;;
  *) die "ADMISSION_PRODUCTION_READINESS_ENABLED must be true or false" ;;
esac

admission_public_base_url="${ADMISSION_PUBLIC_BASE_URL:-}"
[[ "${admission_public_base_url}" == "https://join.stuhelper.com" ]] || \
  die "ADMISSION_PUBLIC_BASE_URL must be exactly https://join.stuhelper.com for production admission links"

database_url="${ADMISSION_READINESS_DATABASE_URL:-${DATABASE_URL:-}}"
[[ -n "${database_url}" ]] || die "ADMISSION_READINESS_DATABASE_URL or DATABASE_URL is required"

materialize_database_url() {
  local value="$1"
  if [[ "${value}" == *"REPLACE_WITH_STUHELPER_APP_DB_PASSWORD"* ]]; then
    [[ -n "${STUHELPER_APP_DB_PASSWORD:-}" ]] || \
      die "DATABASE_URL contains REPLACE_WITH_STUHELPER_APP_DB_PASSWORD but STUHELPER_APP_DB_PASSWORD is not set"
    local encoded_password
    encoded_password="$(
      STUHELPER_APP_DB_PASSWORD="${STUHELPER_APP_DB_PASSWORD}" python3 - <<'PY'
import os
import urllib.parse

print(urllib.parse.quote(os.environ["STUHELPER_APP_DB_PASSWORD"], safe=""))
PY
    )"
    value="${value//REPLACE_WITH_STUHELPER_APP_DB_PASSWORD/${encoded_password}}"
  fi
  [[ "${value}" != *"REPLACE_WITH"* ]] || die "DATABASE_URL contains unresolved placeholder"
  printf '%s\n' "${value}"
}

database_url="$(materialize_database_url "${database_url}")"

required_platform="${ADMISSION_READINESS_REQUIRED_PLATFORM:-qq}"
required_guild_ids="${ADMISSION_READINESS_REQUIRED_GUILD_IDS:-}"
required_school_codes="${ADMISSION_READINESS_REQUIRED_SCHOOL_CODES:-}"
required_school_ids="${ADMISSION_READINESS_REQUIRED_SCHOOL_IDS:-}"
required_bot_credential_name="${ADMISSION_READINESS_REQUIRED_BOT_CREDENTIAL_NAME:-koishi-runtime}"
required_bot_credential_audience="${ADMISSION_READINESS_REQUIRED_BOT_CREDENTIAL_AUDIENCE:-/api/v1/bot/*}"
required_bot_credential_scopes="${ADMISSION_READINESS_REQUIRED_BOT_CREDENTIAL_SCOPES:-bot.qq_binding.consume,bot.qq_verification.read,bot.admission.session,bot.admission.event,bot.admission.review,bot.admission.forward,bot.member_blacklist.read,bot.member_blacklist.manage}"
external_student_source_enabled="${EXTERNAL_STUDENT_SOURCE_ENABLED:-false}"
external_student_source_provider="${EXTERNAL_STUDENT_SOURCE_PROVIDER:-}"
external_student_source_school_code="${EXTERNAL_STUDENT_SOURCE_SCHOOL_CODE:-}"
buaa_external_student_source_ready="false"

case "${external_student_source_enabled}" in
  true|TRUE|1|yes|YES)
    [[ "${external_student_source_provider}" == "oracle" ]] || \
      die "EXTERNAL_STUDENT_SOURCE_PROVIDER must be oracle when EXTERNAL_STUDENT_SOURCE_ENABLED=true"
    [[ "${external_student_source_school_code}" =~ ^[0-9]{10}$ ]] || \
      die "EXTERNAL_STUDENT_SOURCE_SCHOOL_CODE must be a 10-digit school code"
    required_external_oracle_keys=(
      EXTERNAL_STUDENT_SOURCE_ORACLE_HOST
      EXTERNAL_STUDENT_SOURCE_ORACLE_SERVICE_NAME
      EXTERNAL_STUDENT_SOURCE_ORACLE_USERNAME
      EXTERNAL_STUDENT_SOURCE_ORACLE_READONLY_USERNAME
      EXTERNAL_STUDENT_SOURCE_ORACLE_PASSWORD
      EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_CA_HOST_PATH
      EXTERNAL_STUDENT_SOURCE_ORACLE_SCHEMA
      EXTERNAL_STUDENT_SOURCE_ORACLE_TABLE
      EXTERNAL_STUDENT_SOURCE_ORACLE_STUDENT_ID_COLUMN
      EXTERNAL_STUDENT_SOURCE_ORACLE_STUDENT_NAME_COLUMN
    )
    for key in "${required_external_oracle_keys[@]}"; do
      [[ -n "${!key:-}" && "${!key}" != REPLACE_WITH_* ]] || \
        die "${key} is required when EXTERNAL_STUDENT_SOURCE_ENABLED=true"
    done
    [[ "${EXTERNAL_STUDENT_SOURCE_ORACLE_USERNAME^^}" == "${EXTERNAL_STUDENT_SOURCE_ORACLE_READONLY_USERNAME^^}" ]] ||
      die "EXTERNAL_STUDENT_SOURCE_ORACLE_USERNAME must match EXTERNAL_STUDENT_SOURCE_ORACLE_READONLY_USERNAME"
    [[ "${EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_MODE:-}" == "verify-full" ]] ||
      die "EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_MODE must be verify-full in production"
    [[ "${EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_CA_FILE:-}" == "/external-student-source-tls/ca.crt" ]] ||
      die "EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_CA_FILE must be /external-student-source-tls/ca.crt in production"
    if [[ "${external_student_source_school_code}" == "4111010006" ]]; then
      buaa_external_student_source_ready="true"
    fi
    ;;
  false|FALSE|0|no|NO|"") ;;
  *) die "EXTERNAL_STUDENT_SOURCE_ENABLED must be true or false" ;;
esac

run_readiness_sql() {
  compose --profile prod run --rm --no-deps -T \
    postgres-client \
    psql \
      -X \
      -v ON_ERROR_STOP=1 \
      -At \
      -v admission_public_base_url="${admission_public_base_url}" \
      -v required_platform="${required_platform}" \
      -v required_guild_ids="${required_guild_ids}" \
      -v required_school_codes="${required_school_codes}" \
      -v required_school_ids="${required_school_ids}" \
      -v buaa_external_student_source_ready="${buaa_external_student_source_ready}" \
      -v required_bot_credential_name="${required_bot_credential_name}" \
      -v required_bot_credential_audience="${required_bot_credential_audience}" \
      -v required_bot_credential_scopes="${required_bot_credential_scopes}" \
      "${database_url}" "$@"
}

failures="$(
  run_readiness_sql <<'SQL'
WITH
input AS (
  SELECT
    :'admission_public_base_url'::text AS admission_public_base_url,
    :'required_platform'::text AS required_platform,
    :'required_guild_ids'::text AS required_guild_ids,
    :'required_school_codes'::text AS required_school_codes,
    :'required_school_ids'::text AS required_school_ids,
    :'buaa_external_student_source_ready'::boolean AS buaa_external_student_source_ready,
    :'required_bot_credential_name'::text AS required_bot_credential_name,
    :'required_bot_credential_audience'::text AS required_bot_credential_audience,
    :'required_bot_credential_scopes'::text AS required_bot_credential_scopes
),
required_guilds AS (
  SELECT DISTINCT trim(value) AS guild_id
  FROM input, regexp_split_to_table(input.required_guild_ids, ',') AS value
  WHERE trim(value) <> ''
),
required_schools AS (
  SELECT DISTINCT trim(value)::bigint AS school_id
  FROM input, regexp_split_to_table(input.required_school_ids, ',') AS value
  WHERE trim(value) <> ''
),
required_school_codes AS (
  SELECT DISTINCT trim(value) AS school_code
  FROM input, regexp_split_to_table(input.required_school_codes, ',') AS value
  WHERE trim(value) <> ''
),
required_bot_scopes AS (
  SELECT DISTINCT trim(value) AS scope
  FROM input, regexp_split_to_table(input.required_bot_credential_scopes, ',') AS value
  WHERE trim(value) <> ''
),
admission_schools AS (
  SELECT
    sc.school_id,
    COALESCE(s.code, sc.school_id::text) AS school_code,
    sc.school_name,
    sc.enabled,
    COALESCE(NULLIF(trim(sc.academic_db_table), ''), '') AS academic_db_table,
    COALESCE(NULLIF(split_part(NULLIF(trim(sc.academic_db_table), ''), '.', 1), ''), 'public') AS academic_table_schema,
    CASE
      WHEN position('.' in COALESCE(NULLIF(trim(sc.academic_db_table), ''), '')) > 0
      THEN split_part(NULLIF(trim(sc.academic_db_table), ''), '.', 2)
      ELSE COALESCE(NULLIF(trim(sc.academic_db_table), ''), '')
    END AS academic_table_name,
    CASE
      WHEN NULLIF(trim(sc.academic_db_table), '') IS NULL THEN false
      ELSE to_regclass(trim(sc.academic_db_table)) IS NOT NULL
    END AS academic_table_exists,
    EXISTS (
      SELECT 1
      FROM information_schema.columns c
      WHERE c.table_schema = COALESCE(NULLIF(split_part(NULLIF(trim(sc.academic_db_table), ''), '.', 1), ''), 'public')
        AND c.table_name = CASE
          WHEN position('.' in COALESCE(NULLIF(trim(sc.academic_db_table), ''), '')) > 0
          THEN split_part(NULLIF(trim(sc.academic_db_table), ''), '.', 2)
          ELSE COALESCE(NULLIF(trim(sc.academic_db_table), ''), '')
        END
        AND c.column_name = 'xh'
    ) AS academic_table_has_student_id,
    EXISTS (
      SELECT 1
      FROM information_schema.columns c
      WHERE c.table_schema = COALESCE(NULLIF(split_part(NULLIF(trim(sc.academic_db_table), ''), '.', 1), ''), 'public')
        AND c.table_name = CASE
          WHEN position('.' in COALESCE(NULLIF(trim(sc.academic_db_table), ''), '')) > 0
          THEN split_part(NULLIF(trim(sc.academic_db_table), ''), '.', 2)
          ELSE COALESCE(NULLIF(trim(sc.academic_db_table), ''), '')
        END
        AND c.column_name = 'xm'
    ) AS academic_table_has_student_name,
    COALESCE(NULLIF(trim(COALESCE(sc.manual_form_fields, '{}'::jsonb) #>> '{admission,ssoLoginURL}'), ''), '') AS sso_login_url,
    ARRAY(
      SELECT DISTINCT lower(trim(domain.value))
      FROM jsonb_array_elements_text(
        CASE
          WHEN jsonb_typeof(COALESCE(sc.manual_form_fields, '{}'::jsonb) #> '{admission,emailDomains}') = 'array'
          THEN COALESCE(sc.manual_form_fields, '{}'::jsonb) #> '{admission,emailDomains}'
          ELSE '[]'::jsonb
        END
      ) AS domain(value)
      WHERE trim(domain.value) <> ''
      ORDER BY lower(trim(domain.value))
    ) AS email_domains,
    COALESCE(NULLIF(trim(COALESCE(sc.manual_form_fields, '{}'::jsonb) #>> '{admission,emailIdentityPolicy,type}'), ''), '') AS email_identity_policy_type,
    COALESCE(NULLIF(trim(COALESCE(sc.manual_form_fields, '{}'::jsonb) #>> '{admission,emailIdentityPolicy,studentIDEmailDomain}'), ''), '') AS student_id_email_domain,
    CASE
      WHEN jsonb_typeof(COALESCE(sc.manual_form_fields, '{}'::jsonb) #> '{admission,emailIdentityPolicy,requireStudentName}') = 'boolean'
      THEN (COALESCE(sc.manual_form_fields, '{}'::jsonb) #>> '{admission,emailIdentityPolicy,requireStudentName}')::boolean
      ELSE false
    END AS require_student_name,
    EXISTS (
      SELECT 1
      FROM jsonb_array_elements_text(
        CASE
          WHEN jsonb_typeof(COALESCE(sc.manual_form_fields, '{}'::jsonb) #> '{admission,emailDomains}') = 'array'
          THEN COALESCE(sc.manual_form_fields, '{}'::jsonb) #> '{admission,emailDomains}'
          ELSE '[]'::jsonb
        END
      ) AS domain(value)
      WHERE trim(domain.value) <> ''
    ) AS has_email_domain
  FROM public.school_configs sc
  LEFT JOIN public.schools s ON s.id = sc.school_id
),
policy_readiness AS (
  SELECT
    p.*,
    s.school_name,
    s.enabled AS school_enabled,
    COALESCE(s.sso_login_url, '') AS sso_login_url,
    COALESCE(s.has_email_domain, false) AS has_email_domain
  FROM public.group_admission_policies p
  LEFT JOIN admission_schools s ON s.school_id = p.school_id
  WHERE p.platform = (SELECT required_platform FROM input)
),
bot_credential_readiness AS (
  SELECT c.*
  FROM public.bot_service_credentials c
  WHERE c.name = (SELECT required_bot_credential_name FROM input)
),
buaa_academic_row_stats AS (
  SELECT
    GREATEST(
      CASE WHEN c.reltuples < 0 THEN 0 ELSE c.reltuples::bigint END,
      COALESCE(s.n_live_tup, 0)
    ) AS estimated_rows
  FROM pg_catalog.pg_class c
  JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
  LEFT JOIN pg_catalog.pg_stat_all_tables s ON s.relid = c.oid
  WHERE n.nspname = 'academic'
    AND c.relname = 'buaa_students'
    AND c.relkind IN ('r', 'p')
),
failures(message) AS (
  SELECT 'ADMISSION_PUBLIC_BASE_URL must be exactly https://join.stuhelper.com'
  WHERE (SELECT admission_public_base_url FROM input) <> 'https://join.stuhelper.com'

  UNION ALL
  SELECT format('bot service credential %s is missing', (SELECT required_bot_credential_name FROM input))
  WHERE NOT EXISTS (SELECT 1 FROM bot_credential_readiness)

  UNION ALL
  SELECT format('bot service credential %s is revoked', name)
  FROM bot_credential_readiness
  WHERE revoked_at IS NOT NULL

  UNION ALL
  SELECT format('bot service credential %s is expired', name)
  FROM bot_credential_readiness
  WHERE expires_at IS NOT NULL AND expires_at <= now()

  UNION ALL
  SELECT format('bot service credential %s must allow audience %s', name, (SELECT required_bot_credential_audience FROM input))
  FROM bot_credential_readiness
  WHERE NOT ((SELECT required_bot_credential_audience FROM input) = ANY(audience))

  UNION ALL
  SELECT format('bot service credential %s missing scope %s', (SELECT required_bot_credential_name FROM input), rs.scope)
  FROM required_bot_scopes rs
  WHERE NOT EXISTS (
    SELECT 1
    FROM bot_credential_readiness c
    WHERE rs.scope = ANY(c.scopes)
  )

  UNION ALL
  SELECT 'no enabled admission school config with school SSO or email OTP'
  WHERE NOT EXISTS (
    SELECT 1
    FROM admission_schools
    WHERE enabled AND (sso_login_url <> '' OR has_email_domain)
  )

  UNION ALL
  SELECT 'no qq group admission policy configured'
  WHERE NOT EXISTS (SELECT 1 FROM policy_readiness)

  UNION ALL
  SELECT format('required %s guild %s has no group admission policy', (SELECT required_platform FROM input), rg.guild_id)
  FROM required_guilds rg
  WHERE NOT EXISTS (
    SELECT 1 FROM policy_readiness p WHERE p.guild_id = rg.guild_id
  )

  UNION ALL
  SELECT format('required school code %s has no enabled admission config with SSO or email OTP', rsc.school_code)
  FROM required_school_codes rsc
  WHERE NOT EXISTS (
    SELECT 1
    FROM admission_schools s
    WHERE s.school_code = rsc.school_code
      AND s.enabled
      AND (s.sso_login_url <> '' OR s.has_email_domain)
  )

  UNION ALL
  SELECT format('required school %s has no enabled admission config with SSO or email OTP', rs.school_id)
  FROM required_schools rs
  WHERE NOT EXISTS (
    SELECT 1
    FROM admission_schools s
    WHERE s.school_id = rs.school_id
      AND s.enabled
      AND (s.sso_login_url <> '' OR s.has_email_domain)
  )

  UNION ALL
  SELECT format('enabled school code %s is not in ADMISSION_READINESS_REQUIRED_SCHOOL_CODES', s.school_code)
  FROM admission_schools s
  WHERE s.enabled
    AND EXISTS (SELECT 1 FROM required_school_codes)
    AND NOT EXISTS (
      SELECT 1
      FROM required_school_codes rsc
      WHERE rsc.school_code = s.school_code
    )

  UNION ALL
  SELECT 'BUAA school directory row code 4111010006 is missing'
  WHERE EXISTS (SELECT 1 FROM required_school_codes WHERE school_code = '4111010006')
    AND NOT EXISTS (
      SELECT 1
      FROM public.schools
      WHERE code = '4111010006'
    )

  UNION ALL
  SELECT 'BUAA admission config must use academic.buaa_students as academic_db_table'
  WHERE EXISTS (SELECT 1 FROM required_school_codes WHERE school_code = '4111010006')
    AND NOT (SELECT buaa_external_student_source_ready FROM input)
    AND NOT EXISTS (
      SELECT 1
      FROM admission_schools
      WHERE school_code = '4111010006'
        AND academic_db_table = 'academic.buaa_students'
    )

  UNION ALL
  SELECT 'BUAA admission emailDomains must be exactly buaa.edu.cn'
  WHERE EXISTS (SELECT 1 FROM required_school_codes WHERE school_code = '4111010006')
    AND NOT EXISTS (
      SELECT 1
      FROM admission_schools
      WHERE school_code = '4111010006'
        AND email_domains = ARRAY['buaa.edu.cn']::text[]
    )

  UNION ALL
  SELECT 'BUAA admission emailIdentityPolicy.type must be academic_student_email'
  WHERE EXISTS (SELECT 1 FROM required_school_codes WHERE school_code = '4111010006')
    AND NOT EXISTS (
      SELECT 1
      FROM admission_schools
      WHERE school_code = '4111010006'
        AND email_identity_policy_type = 'academic_student_email'
    )

  UNION ALL
  SELECT 'BUAA admission studentIDEmailDomain must be buaa.edu.cn'
  WHERE EXISTS (SELECT 1 FROM required_school_codes WHERE school_code = '4111010006')
    AND NOT EXISTS (
      SELECT 1
      FROM admission_schools
      WHERE school_code = '4111010006'
        AND student_id_email_domain = 'buaa.edu.cn'
    )

  UNION ALL
  SELECT 'BUAA admission must require student name before sending school email OTP'
  WHERE EXISTS (SELECT 1 FROM required_school_codes WHERE school_code = '4111010006')
    AND NOT EXISTS (
      SELECT 1
      FROM admission_schools
      WHERE school_code = '4111010006'
        AND require_student_name
    )

  UNION ALL
  SELECT 'BUAA academic table academic.buaa_students is missing'
  WHERE EXISTS (SELECT 1 FROM required_school_codes WHERE school_code = '4111010006')
    AND NOT (SELECT buaa_external_student_source_ready FROM input)
    AND NOT EXISTS (
      SELECT 1
      FROM admission_schools
      WHERE school_code = '4111010006'
        AND academic_table_exists
    )

  UNION ALL
  SELECT 'BUAA academic table academic.buaa_students must expose xh and xm columns'
  WHERE EXISTS (SELECT 1 FROM required_school_codes WHERE school_code = '4111010006')
    AND NOT (SELECT buaa_external_student_source_ready FROM input)
    AND NOT EXISTS (
      SELECT 1
      FROM admission_schools
      WHERE school_code = '4111010006'
        AND academic_table_has_student_id
        AND academic_table_has_student_name
    )

  UNION ALL
  SELECT 'BUAA academic table academic.buaa_students has no rows; import real BUAA student records before admission go-live'
  WHERE EXISTS (SELECT 1 FROM required_school_codes WHERE school_code = '4111010006')
    AND NOT (SELECT buaa_external_student_source_ready FROM input)
    AND EXISTS (SELECT 1 FROM buaa_academic_row_stats)
    AND NOT EXISTS (
      SELECT 1
      FROM buaa_academic_row_stats
      WHERE estimated_rows > 0
    )

  UNION ALL
  SELECT 'freshman camera handoff table is missing'
  WHERE NOT EXISTS (
    SELECT 1
    FROM information_schema.tables
    WHERE table_schema = 'public'
      AND table_name = 'freshman_camera_handoffs'
  )

  UNION ALL
  SELECT 'freshman camera handoff table must expose token_hash, status and continue_on columns'
  WHERE NOT EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = 'public'
      AND table_name = 'freshman_camera_handoffs'
      AND column_name = 'token_hash'
  )
    OR NOT EXISTS (
      SELECT 1
      FROM information_schema.columns
      WHERE table_schema = 'public'
        AND table_name = 'freshman_camera_handoffs'
        AND column_name = 'status'
    )
    OR NOT EXISTS (
      SELECT 1
      FROM information_schema.columns
      WHERE table_schema = 'public'
        AND table_name = 'freshman_camera_handoffs'
        AND column_name = 'continue_on'
    )

  UNION ALL
  SELECT 'freshman camera handoff table must enforce one active handoff per application'
  WHERE to_regclass('public.freshman_camera_handoffs_active_application_idx') IS NULL

  UNION ALL
  SELECT format('policy %s/%s references missing or disabled admission school %s', platform, guild_id, school_id)
  FROM policy_readiness
  WHERE school_enabled IS DISTINCT FROM true

  UNION ALL
  SELECT format('policy %s/%s school %s has no admission SSO or email OTP capability', platform, guild_id, school_id)
  FROM policy_readiness
  WHERE school_enabled AND sso_login_url = '' AND NOT has_email_domain

  UNION ALL
  SELECT format('policy %s/%s must keep auto_approve_verified_join=true and auto_approve_unverified_join=true for admission MVP', platform, guild_id)
  FROM policy_readiness
  WHERE auto_approve_verified_join IS DISTINCT FROM true
     OR auto_approve_unverified_join IS DISTINCT FROM true

  UNION ALL
  SELECT format('policy %s/%s management_guild_ids must not be empty', platform, guild_id)
  FROM policy_readiness
  WHERE COALESCE(cardinality(management_guild_ids), 0) = 0

  UNION ALL
  SELECT format('policy %s/%s freshman_channel_enabled must be true', platform, guild_id)
  FROM policy_readiness
  WHERE freshman_channel_enabled IS DISTINCT FROM true

  UNION ALL
  SELECT format('policy %s/%s freshman_channel_closes_at must be in the future', platform, guild_id)
  FROM policy_readiness
  WHERE freshman_channel_enabled AND freshman_channel_closes_at <= now()

  UNION ALL
  SELECT format('policy %s/%s freshman_default_expires_at must be in the future', platform, guild_id)
  FROM policy_readiness
  WHERE freshman_channel_enabled AND freshman_default_expires_at <= now()

  UNION ALL
  SELECT format('policy %s/%s link_wait_seconds must be > 0', platform, guild_id)
  FROM policy_readiness
  WHERE link_wait_seconds <= 0

  UNION ALL
  SELECT format('policy %s/%s submission_wait_seconds must be > 0', platform, guild_id)
  FROM policy_readiness
  WHERE submission_wait_seconds <= 0

  UNION ALL
  SELECT format('policy %s/%s failed_join_limit must be > 0', platform, guild_id)
  FROM policy_readiness
  WHERE failed_join_limit <= 0

  UNION ALL
  SELECT format('policy %s/%s max_material_bytes must be > 0', platform, guild_id)
  FROM policy_readiness
  WHERE max_material_bytes <= 0
)
SELECT message
FROM failures
ORDER BY message;
SQL
)" || die "admission production readiness query failed: ${failures//$'\n'/; }"

if [[ -n "${failures}" ]]; then
  die "admission production readiness failed: ${failures//$'\n'/; }"
fi

summary="$(
  run_readiness_sql <<'SQL'
WITH
admission_schools AS (
  SELECT
    sc.school_id,
    COALESCE(s.code, sc.school_id::text) AS school_code,
    sc.enabled,
    COALESCE(NULLIF(trim(COALESCE(sc.manual_form_fields, '{}'::jsonb) #>> '{admission,ssoLoginURL}'), ''), '') AS sso_login_url,
    EXISTS (
      SELECT 1
      FROM jsonb_array_elements_text(
        CASE
          WHEN jsonb_typeof(COALESCE(sc.manual_form_fields, '{}'::jsonb) #> '{admission,emailDomains}') = 'array'
          THEN COALESCE(sc.manual_form_fields, '{}'::jsonb) #> '{admission,emailDomains}'
          ELSE '[]'::jsonb
        END
      ) AS domain(value)
      WHERE trim(domain.value) <> ''
    ) AS has_email_domain
  FROM public.school_configs sc
  LEFT JOIN public.schools s ON s.id = sc.school_id
)
SELECT format(
  'admission production readiness passed: schools=%s policies=%s bot_credential=%s buaa_student_source=%s',
  (
    SELECT count(*)
    FROM admission_schools
    WHERE enabled AND (sso_login_url <> '' OR has_email_domain)
  ),
  (
    SELECT count(*)
    FROM public.group_admission_policies
    WHERE platform = :'required_platform'
  ),
  :'required_bot_credential_name',
  CASE WHEN :'buaa_external_student_source_ready'::boolean THEN 'external_oracle' ELSE 'local_academic_table' END
);
SQL
)" || die "admission production readiness summary query failed: ${summary//$'\n'/; }"

log "${summary}"
