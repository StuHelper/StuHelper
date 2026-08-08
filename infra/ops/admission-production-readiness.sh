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

load_env_preserving APP_ENV ADMISSION_READINESS_REQUIRED_METHODS ADMISSION_READINESS_ALLOW_FIXTURE_ROSTER
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
required_methods="${ADMISSION_READINESS_REQUIRED_METHODS:-${STUDENT_VERIFICATION_READINESS_REQUIRED_METHODS:-real_name_identity_check,school_sso,manual_material_review}}"
allow_fixture_roster="${ADMISSION_READINESS_ALLOW_FIXTURE_ROSTER:-false}"

case "${EXTERNAL_STUDENT_SOURCE_ENABLED:-false}" in
  false|FALSE|0|no|NO|"") ;;
  true|TRUE|1|yes|YES)
    die "EXTERNAL_STUDENT_SOURCE_ENABLED must remain false; the online admission path uses the student-verification domain and Campus Connector"
    ;;
  *) die "EXTERNAL_STUDENT_SOURCE_ENABLED must be true or false" ;;
esac

case "${allow_fixture_roster}" in
  true|TRUE|1|yes|YES)
    [[ "${APP_ENV:-}" == "prod-parity" ]] || \
      die "ADMISSION_READINESS_ALLOW_FIXTURE_ROSTER is only allowed when APP_ENV=prod-parity"
    allow_fixture_roster="true"
    ;;
  false|FALSE|0|no|NO|"")
    allow_fixture_roster="false"
    ;;
  *) die "ADMISSION_READINESS_ALLOW_FIXTURE_ROSTER must be true or false" ;;
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
      -v required_methods="${required_methods}" \
      -v allow_fixture_roster="${allow_fixture_roster}" \
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
    :'required_methods'::text AS required_methods,
    :'allow_fixture_roster'::boolean AS allow_fixture_roster,
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
    NULLIF(trim(sc.academic_db_table), '') AS academic_db_table
  FROM public.school_configs sc
  LEFT JOIN public.schools s ON s.id = sc.school_id
),
policy_readiness AS (
  SELECT
    p.*,
    s.school_name,
    s.enabled AS school_enabled
  FROM public.group_admission_policies p
  LEFT JOIN admission_schools s ON s.school_id = p.school_id
  WHERE p.platform = (SELECT required_platform FROM input)
),
bot_credential_readiness AS (
  SELECT c.*
  FROM public.bot_service_credentials c
  WHERE c.name = (SELECT required_bot_credential_name FROM input)
),
required_verification_methods AS (
  SELECT DISTINCT trim(value) AS method
  FROM input, regexp_split_to_table(input.required_methods, ',') AS value
  WHERE trim(value) <> ''
),
target_schools AS (
  SELECT
    required.school_code,
    school.id AS school_id,
    school.name AS school_name,
    profile.enabled AS profile_enabled,
    profile.validation_status AS profile_validation_status,
    profile.snapshot_sync_interval_seconds,
    profile.snapshot_warning_after_seconds,
    profile.snapshot_hard_expiry_seconds,
    profile.config_revision
  FROM required_school_codes required
  LEFT JOIN public.schools school ON school.code = required.school_code
  LEFT JOIN public.school_verification_profiles profile ON profile.school_id = school.id
),
available_methods AS (
  SELECT
    school.school_code,
    method.method,
    method.roster_dependency,
    method.connector_operation_key,
    method.privacy_notice_version,
    method.privacy_notice,
    method.school_id
  FROM target_schools school
  JOIN public.school_verification_methods method ON method.school_id = school.school_id
  WHERE method.enabled
    AND method.validation_status = 'valid'
    AND method.health_status IN ('healthy', 'degraded')
),
available_connector_operations AS (
  SELECT operation.school_id, operation.operation_key, operation.operation_type
  FROM public.campus_connector_school_operations operation
  JOIN public.campus_connector_nodes node ON node.id = operation.node_id
  WHERE operation.enabled
    AND operation.validation_status = 'valid'
    AND operation.health_status IN ('healthy', 'degraded')
    AND node.status IN ('active', 'degraded')
    AND node.revoked_at IS NULL
    AND node.certificate_not_after > now() + interval '30 days'
    AND node.last_heartbeat_at >= now() - make_interval(
      secs => GREATEST(120, node.heartbeat_interval_seconds * 3)
    )
),
active_rosters AS (
  SELECT
    school.school_code,
    school.school_id,
    profile.snapshot_hard_expiry_seconds,
    snapshot.id AS snapshot_id,
    snapshot.status,
    snapshot.source_kind,
    snapshot.import_mode,
    snapshot.source_cutoff_at,
    snapshot.row_count,
    snapshot.eligible_row_count,
    snapshot.checksum,
    snapshot.signature_algorithm,
    snapshot.signature_key_id,
    snapshot.snapshot_signature,
    snapshot.connector_node_id,
    (
      SELECT count(*)
      FROM academic.student_roster_records record
      WHERE record.snapshot_id = snapshot.id
        AND record.school_id = school.school_id
    ) AS actual_row_count,
    (
      SELECT count(*)
      FROM academic.student_roster_records record
      WHERE record.snapshot_id = snapshot.id
        AND record.school_id = school.school_id
        AND record.eligibility_status = 'eligible'
    ) AS actual_eligible_row_count,
    (
      SELECT count(*)
      FROM academic.student_roster_quality_checks quality
      WHERE quality.snapshot_id = snapshot.id
    ) AS quality_check_count,
    (
      SELECT count(*)
      FROM academic.student_roster_quality_checks quality
      WHERE quality.snapshot_id = snapshot.id
        AND quality.status IN ('pending', 'failed')
    ) AS blocking_quality_check_count
  FROM target_schools school
  LEFT JOIN public.school_verification_profiles profile ON profile.school_id = school.school_id
  LEFT JOIN academic.student_roster_active active ON active.school_id = school.school_id
  LEFT JOIN academic.student_roster_snapshots snapshot
    ON snapshot.id = active.snapshot_id
   AND snapshot.school_id = school.school_id
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
  SELECT 'no enabled and healthy student verification method configured'
  WHERE NOT EXISTS (SELECT 1 FROM available_methods)

  UNION ALL
  SELECT 'ADMISSION_READINESS_REQUIRED_METHODS must not be empty'
  WHERE NOT EXISTS (SELECT 1 FROM required_verification_methods)

  UNION ALL
  SELECT format('required school code %s method %s is missing or unavailable', school.school_code, required.method)
  FROM target_schools school
  CROSS JOIN required_verification_methods required
  WHERE school.school_id IS NOT NULL
    AND NOT EXISTS (
      SELECT 1
      FROM available_methods method
      WHERE method.school_code = school.school_code
        AND method.method = required.method
    )

  UNION ALL
  SELECT format('required school code %s method %s must include a privacy notice', method.school_code, method.method)
  FROM available_methods method
  WHERE method.method IN (SELECT required.method FROM required_verification_methods required)
    AND (
      NULLIF(trim(COALESCE(method.privacy_notice_version, '')), '') IS NULL
      OR method.privacy_notice = '{}'::jsonb
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
  SELECT format('required school code %s is missing from the school directory', school.school_code)
  FROM target_schools school
  WHERE school.school_id IS NULL

  UNION ALL
  SELECT format('required school code %s has no verification profile', school.school_code)
  FROM target_schools school
  WHERE school.school_id IS NOT NULL
    AND school.profile_validation_status IS NULL

  UNION ALL
  SELECT format('required school code %s verification profile is not enabled and valid', school.school_code)
  FROM target_schools school
  WHERE school.profile_validation_status IS NOT NULL
    AND (school.profile_enabled IS DISTINCT FROM true OR school.profile_validation_status <> 'valid')

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
  SELECT format('required school code %s school config still references the retired academic table', school.school_code)
  FROM admission_schools school
  WHERE school.school_code IN (SELECT required.school_code FROM required_school_codes required)
    AND school.academic_db_table IS NOT NULL

  UNION ALL
  SELECT format('required school code %s active roster is missing', roster.school_code)
  FROM active_rosters roster
  WHERE roster.snapshot_id IS NULL

  UNION ALL
  SELECT format('required school code %s active roster pointer references a non-active snapshot', roster.school_code)
  FROM active_rosters roster
  WHERE roster.snapshot_id IS NOT NULL AND roster.status <> 'active'

  UNION ALL
  SELECT format('required school code %s active roster must be a full snapshot', roster.school_code)
  FROM active_rosters roster
  WHERE roster.snapshot_id IS NOT NULL
    AND roster.import_mode NOT IN ('full', 'reconciled_full')

  UNION ALL
  SELECT format('required school code %s active roster source must be Campus Connector', roster.school_code)
  FROM active_rosters roster
  WHERE roster.snapshot_id IS NOT NULL
    AND (
      (NOT (SELECT allow_fixture_roster FROM input) AND roster.source_kind <> 'campus_connector')
      OR (NOT (SELECT allow_fixture_roster FROM input) AND roster.connector_node_id IS NULL)
      OR (NOT (SELECT allow_fixture_roster FROM input) AND (
        roster.checksum IS NULL
        OR roster.signature_algorithm IS NULL
        OR roster.signature_key_id IS NULL
        OR roster.snapshot_signature IS NULL
      ))
      OR ((SELECT allow_fixture_roster FROM input) AND roster.source_kind NOT IN ('fixture', 'campus_connector'))
    )

  UNION ALL
  SELECT format('required school code %s active roster snapshot is past the hard freshness threshold', roster.school_code)
  FROM active_rosters roster
  WHERE roster.snapshot_id IS NOT NULL
    AND roster.source_cutoff_at + make_interval(secs => roster.snapshot_hard_expiry_seconds) <= now()

  UNION ALL
  SELECT format('required school code %s active roster snapshot has invalid row counts', roster.school_code)
  FROM active_rosters roster
  WHERE roster.snapshot_id IS NOT NULL
    AND (
      roster.row_count <= 0
      OR roster.eligible_row_count <= 0
      OR roster.eligible_row_count > roster.row_count
      OR roster.actual_row_count <> roster.row_count
      OR roster.actual_eligible_row_count <> roster.eligible_row_count
    )

  UNION ALL
  SELECT format('required school code %s active roster snapshot has incomplete or failed quality gates', roster.school_code)
  FROM active_rosters roster
  WHERE roster.snapshot_id IS NOT NULL
    AND NOT (SELECT allow_fixture_roster FROM input)
    AND (roster.quality_check_count = 0 OR roster.blocking_quality_check_count > 0)

  UNION ALL
  SELECT format('required school code %s required method has no healthy approved connector operation', method.school_code)
  FROM available_methods method
  WHERE method.method IN (SELECT required.method FROM required_verification_methods required)
    AND method.connector_operation_key IS NOT NULL
    AND NOT EXISTS (
      SELECT 1
      FROM available_connector_operations operation
      WHERE operation.school_id = method.school_id
        AND operation.operation_key = method.connector_operation_key
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
  SELECT format('policy %s/%s references missing or disabled student-verification school %s', p.platform, p.guild_id, p.school_id)
  FROM policy_readiness p
  WHERE NOT EXISTS (
    SELECT 1
    FROM target_schools school
    WHERE school.school_id = p.school_id
      AND school.profile_enabled = true
      AND school.profile_validation_status = 'valid'
  )

  UNION ALL
  SELECT format('policy %s/%s school %s has no required healthy student-verification method', p.platform, p.guild_id, p.school_id)
  FROM policy_readiness p
  WHERE NOT EXISTS (
    SELECT 1
    FROM available_methods method
    WHERE method.school_id = p.school_id
      AND method.method IN (SELECT required.method FROM required_verification_methods required)
  )

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
required_school_codes AS (
  SELECT DISTINCT trim(value) AS school_code
  FROM regexp_split_to_table(:'required_school_codes', ',') AS value
  WHERE trim(value) <> ''
),
target_schools AS (
  SELECT
    required.school_code,
    school.id AS school_id,
    profile.snapshot_hard_expiry_seconds
  FROM required_school_codes required
  LEFT JOIN public.schools school ON school.code = required.school_code
  LEFT JOIN public.school_verification_profiles profile ON profile.school_id = school.id
),
active_rosters AS (
  SELECT
    school.school_code,
    snapshot.source_kind,
    snapshot.row_count,
    snapshot.eligible_row_count,
    snapshot.source_cutoff_at,
    active.activation_revision
  FROM target_schools school
  JOIN academic.student_roster_active active ON active.school_id = school.school_id
  JOIN academic.student_roster_snapshots snapshot
    ON snapshot.id = active.snapshot_id
   AND snapshot.school_id = school.school_id
),
available_methods AS (
  SELECT method.school_id
  FROM public.school_verification_methods method
  WHERE method.enabled
    AND method.validation_status = 'valid'
    AND method.health_status IN ('healthy', 'degraded')
)
SELECT format(
  'admission production readiness passed: schools=%s policies=%s bot_credential=%s roster_source=%s',
  (
    SELECT count(*) FROM target_schools
  ),
  (
    SELECT count(*)
    FROM public.group_admission_policies
    WHERE platform = :'required_platform'
  ),
  :'required_bot_credential_name',
  COALESCE((SELECT string_agg(DISTINCT source_kind, ',') FROM active_rosters), 'none')
);
SQL
)" || die "admission production readiness summary query failed: ${summary//$'\n'/; }"

log "${summary}"
