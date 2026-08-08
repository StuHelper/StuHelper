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

usage() {
  cat <<'USAGE'
Usage: infra/ops/admission-bootstrap-production-data.sh

Idempotently prepares the minimal production data required by the admission MVP:

  - BUAA admission config with school_code=4111010006
  - buaa.edu.cn as the only BUAA school email domain
  - qq group admission policy for 178037297
  - forward_raw_material_to_qq=false for the MVP

Input:
  ADMISSION_BOOTSTRAP_DATABASE_URL or DATABASE_URL is required.
  ENV_FILE defaults to .env.prod.shared when this script is run from a Baota source bundle.
  SECRETS_ENV_FILE defaults to .env.prod.secrets.local or .env.prod.secrets when present.
  GENERATED_ENV_FILE defaults to .env.prod.generated when present.
  GENERATED_SECRET_ENV_FILE defaults to .env.prod.generated.secrets when present.
  ADMISSION_BOOTSTRAP_PLATFORM defaults to qq.
  ADMISSION_BOOTSTRAP_SCHOOL_CODE defaults to 4111010006.
  ADMISSION_BOOTSTRAP_GROUP_IDS defaults to 178037297.
  ADMISSION_BOOTSTRAP_MANAGEMENT_GUILD_IDS defaults to ADMISSION_BOOTSTRAP_GROUP_IDS.
  ADMISSION_BOOTSTRAP_EMAIL_DOMAINS defaults to buaa.edu.cn.
  ADMISSION_BOOTSTRAP_DISABLE_OTHER_SCHOOLS defaults to true for the current BUAA-only launch.
  ADMISSION_BOOTSTRAP_PRUNE_OTHER_GROUP_POLICIES defaults to true for the current single-group launch.

No secret values are written to the repository or printed.
USAGE
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
  exit 0
fi

require_cmd docker
require_cmd python3

load_env
export COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-${STACK_NAME:-stuhelper}}"

database_url="${ADMISSION_BOOTSTRAP_DATABASE_URL:-${DATABASE_URL:-}}"
[[ -n "${database_url}" ]] || die "ADMISSION_BOOTSTRAP_DATABASE_URL or DATABASE_URL is required"

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

platform="${ADMISSION_BOOTSTRAP_PLATFORM:-qq}"
school_code="${ADMISSION_BOOTSTRAP_SCHOOL_CODE:-4111010006}"
school_name="${ADMISSION_BOOTSTRAP_SCHOOL_NAME:-北京航空航天大学}"
group_ids="${ADMISSION_BOOTSTRAP_GROUP_IDS:-178037297}"
management_guild_ids="${ADMISSION_BOOTSTRAP_MANAGEMENT_GUILD_IDS:-${group_ids}}"
email_domains="${ADMISSION_BOOTSTRAP_EMAIL_DOMAINS:-buaa.edu.cn}"
disable_other_schools="${ADMISSION_BOOTSTRAP_DISABLE_OTHER_SCHOOLS:-true}"
prune_other_group_policies="${ADMISSION_BOOTSTRAP_PRUNE_OTHER_GROUP_POLICIES:-true}"
freshman_channel_closes_at="${ADMISSION_BOOTSTRAP_FRESHMAN_CHANNEL_CLOSES_AT:-2026-12-31T23:59:59+08:00}"
freshman_default_expires_at="${ADMISSION_BOOTSTRAP_FRESHMAN_DEFAULT_EXPIRES_AT:-2026-10-31T23:59:59+08:00}"

[[ "${platform}" == "qq" ]] || die "ADMISSION_BOOTSTRAP_PLATFORM must be qq for the current admission MVP"
[[ "${school_code}" == "4111010006" ]] || die "ADMISSION_BOOTSTRAP_SCHOOL_CODE must be 4111010006 for BUAA"
[[ -n "${group_ids//,/}" ]] || die "ADMISSION_BOOTSTRAP_GROUP_IDS must not be empty"
[[ -n "${management_guild_ids//,/}" ]] || die "ADMISSION_BOOTSTRAP_MANAGEMENT_GUILD_IDS must not be empty"
[[ "${email_domains}" == "buaa.edu.cn" ]] || die "ADMISSION_BOOTSTRAP_EMAIL_DOMAINS must be exactly buaa.edu.cn for BUAA"
case "${disable_other_schools}" in
  true|TRUE|1|yes|YES) disable_other_schools="true" ;;
  false|FALSE|0|no|NO) disable_other_schools="false" ;;
  *) die "ADMISSION_BOOTSTRAP_DISABLE_OTHER_SCHOOLS must be true or false" ;;
esac
case "${prune_other_group_policies}" in
  true|TRUE|1|yes|YES) prune_other_group_policies="true" ;;
  false|FALSE|0|no|NO) prune_other_group_policies="false" ;;
  *) die "ADMISSION_BOOTSTRAP_PRUNE_OTHER_GROUP_POLICIES must be true or false" ;;
esac

run_sql() {
  compose --profile prod run --rm --no-deps -T \
    postgres-client \
    psql \
      -X \
      -v ON_ERROR_STOP=1 \
      -v platform="${platform}" \
      -v school_code="${school_code}" \
      -v school_name="${school_name}" \
      -v group_ids="${group_ids}" \
      -v management_guild_ids="${management_guild_ids}" \
      -v email_domains="${email_domains}" \
      -v disable_other_schools="${disable_other_schools}" \
      -v prune_other_group_policies="${prune_other_group_policies}" \
      -v freshman_channel_closes_at="${freshman_channel_closes_at}" \
      -v freshman_default_expires_at="${freshman_default_expires_at}" \
      "${database_url}" "$@"
}

run_sql <<'SQL'
WITH
input AS (
  SELECT
    :'platform'::text AS platform,
    :'school_code'::text AS school_code,
    :'school_name'::text AS school_name,
    ARRAY(
      SELECT DISTINCT trim(value)
      FROM regexp_split_to_table(:'group_ids', ',') AS value
      WHERE trim(value) <> ''
      ORDER BY trim(value)
    ) AS group_ids,
    ARRAY(
      SELECT DISTINCT trim(value)
      FROM regexp_split_to_table(:'management_guild_ids', ',') AS value
      WHERE trim(value) <> ''
      ORDER BY trim(value)
    ) AS management_guild_ids,
    ARRAY(
      SELECT DISTINCT lower(trim(value))
      FROM regexp_split_to_table(:'email_domains', ',') AS value
      WHERE trim(value) <> ''
      ORDER BY lower(trim(value))
    ) AS email_domains,
    :'disable_other_schools'::boolean AS disable_other_schools,
    :'freshman_channel_closes_at'::timestamptz AS freshman_channel_closes_at,
    :'freshman_default_expires_at'::timestamptz AS freshman_default_expires_at
),
validated AS (
  SELECT *
  FROM input
  WHERE cardinality(group_ids) > 0
    AND cardinality(management_guild_ids) > 0
    AND email_domains = ARRAY['buaa.edu.cn']::text[]
),
school_upsert AS (
  INSERT INTO public.schools (code, name)
  SELECT school_code, school_name
  FROM input
  ON CONFLICT (code) DO UPDATE
  SET code = EXCLUDED.code,
      name = EXCLUDED.name
  RETURNING id, code
),
school_config_upsert AS (
  INSERT INTO public.school_configs (
    school_id, school_name, verification_method, academic_db_table, consent_text,
    manual_form_fields, enabled, approval_policy, updated_at
  )
  SELECT
    school_upsert.id,
    validated.school_name,
    'manual',
    NULL,
    '本功能将使用您提供的学校账号或学校邮箱验证学生身份。认证结果用于 StuHelper 入群验证和平台服务。',
    jsonb_build_object(
      'admission',
      jsonb_build_object(
        'emailDomains', to_jsonb(email_domains),
        'emailIdentityPolicy',
        jsonb_build_object(
          'type', 'academic_student_email',
          'studentIDEmailDomain', 'buaa.edu.cn',
          'requireStudentName', true
        )
      )
    ),
    true,
    'auto',
    now()
  FROM validated
  CROSS JOIN school_upsert
  ON CONFLICT (school_id) DO UPDATE
  SET school_name = EXCLUDED.school_name,
      academic_db_table = NULL,
      manual_form_fields =
        jsonb_set(
          jsonb_set(
            COALESCE(public.school_configs.manual_form_fields, '{}'::jsonb),
            '{admission,emailDomains}',
            EXCLUDED.manual_form_fields #> '{admission,emailDomains}',
            true
          ),
          '{admission,emailIdentityPolicy}',
          EXCLUDED.manual_form_fields #> '{admission,emailIdentityPolicy}',
          true
        ),
      verification_method = 'manual',
      enabled = true,
      updated_at = now()
  RETURNING school_id
),
disabled_other_school_configs AS (
  UPDATE public.school_configs sc
  SET enabled = false,
      updated_at = now()
  FROM input
  CROSS JOIN school_upsert
  WHERE input.disable_other_schools
    AND sc.school_id <> school_upsert.id
    AND sc.enabled
  RETURNING sc.school_id
)
INSERT INTO public.group_admission_policies (
  id, platform, guild_id, school_id, auto_approve_join,
  auto_approve_verified_join, auto_approve_unverified_join,
  initial_mute_duration_seconds, link_wait_seconds, submission_wait_seconds,
  manual_review_timeout_seconds, reminder_interval_seconds, failed_join_limit,
  blacklist_duration_seconds, freshman_channel_enabled, freshman_channel_closes_at,
  freshman_default_expires_at, forward_raw_material_to_qq, management_guild_ids,
  max_material_bytes, max_extension_days, updated_at
)
SELECT
  md5(input.platform || ':' || group_id),
  input.platform,
  group_id,
  school_upsert.id,
  true,
  true,
  true,
  2592000,
  3600,
  86400,
  86400,
  900,
  3,
  NULL,
  true,
  input.freshman_channel_closes_at,
  input.freshman_default_expires_at,
  false,
  input.management_guild_ids,
  10485760,
  90,
  now()
FROM input
CROSS JOIN school_upsert
CROSS JOIN unnest(input.group_ids) AS group_id
ON CONFLICT (platform, guild_id) DO UPDATE
SET school_id = EXCLUDED.school_id,
    auto_approve_join = true,
    auto_approve_verified_join = true,
    auto_approve_unverified_join = true,
    freshman_channel_enabled = true,
    freshman_channel_closes_at = EXCLUDED.freshman_channel_closes_at,
    freshman_default_expires_at = EXCLUDED.freshman_default_expires_at,
    forward_raw_material_to_qq = false,
    management_guild_ids = EXCLUDED.management_guild_ids,
    updated_at = now();
SQL

run_sql <<'SQL'
WITH input AS (
  SELECT
    :'platform'::text AS platform,
    ARRAY(
      SELECT DISTINCT trim(value)
      FROM regexp_split_to_table(:'group_ids', ',') AS value
      WHERE trim(value) <> ''
      ORDER BY trim(value)
    ) AS group_ids,
    :'prune_other_group_policies'::boolean AS prune_other_group_policies
)
DELETE FROM public.group_admission_policies p
USING input
WHERE input.prune_other_group_policies
  AND p.platform = input.platform
  AND NOT (p.guild_id = ANY(input.group_ids));
SQL

run_sql -At <<'SQL'
SELECT format(
  'admission bootstrap data ready: school_code=%s policies=%s groups=%s',
  :'school_code',
  (SELECT count(*) FROM public.group_admission_policies WHERE platform = :'platform' AND guild_id = ANY(string_to_array(:'group_ids', ','))),
  :'group_ids'
);
SQL
