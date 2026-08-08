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

readiness_enabled_override_set=false
gateway_enabled_override_set=false
legacy_source_override_set=false
public_base_url_override_set=false
readiness_database_url_override_set=false
if [[ -n "${STUDENT_VERIFICATION_PRODUCTION_READINESS_ENABLED+x}" ]]; then
  readiness_enabled_override_set=true
  readiness_enabled_override="${STUDENT_VERIFICATION_PRODUCTION_READINESS_ENABLED}"
fi
if [[ -n "${CAMPUS_CONNECTOR_GATEWAY_ENABLED+x}" ]]; then
  gateway_enabled_override_set=true
  gateway_enabled_override="${CAMPUS_CONNECTOR_GATEWAY_ENABLED}"
fi
if [[ -n "${EXTERNAL_STUDENT_SOURCE_ENABLED+x}" ]]; then
  legacy_source_override_set=true
  legacy_source_override="${EXTERNAL_STUDENT_SOURCE_ENABLED}"
fi
if [[ -n "${STUDENT_VERIFICATION_PUBLIC_BASE_URL+x}" ]]; then
  public_base_url_override_set=true
  public_base_url_override="${STUDENT_VERIFICATION_PUBLIC_BASE_URL}"
fi
if [[ -n "${STUDENT_VERIFICATION_READINESS_DATABASE_URL+x}" ]]; then
  readiness_database_url_override_set=true
  readiness_database_url_override="${STUDENT_VERIFICATION_READINESS_DATABASE_URL}"
fi

load_env

if [[ "${readiness_enabled_override_set}" == "true" ]]; then
  export STUDENT_VERIFICATION_PRODUCTION_READINESS_ENABLED="${readiness_enabled_override}"
fi
if [[ "${gateway_enabled_override_set}" == "true" ]]; then
  export CAMPUS_CONNECTOR_GATEWAY_ENABLED="${gateway_enabled_override}"
fi
if [[ "${legacy_source_override_set}" == "true" ]]; then
  export EXTERNAL_STUDENT_SOURCE_ENABLED="${legacy_source_override}"
fi
if [[ "${public_base_url_override_set}" == "true" ]]; then
  export STUDENT_VERIFICATION_PUBLIC_BASE_URL="${public_base_url_override}"
fi
if [[ "${readiness_database_url_override_set}" == "true" ]]; then
  export STUDENT_VERIFICATION_READINESS_DATABASE_URL="${readiness_database_url_override}"
fi

case "${STUDENT_VERIFICATION_PRODUCTION_READINESS_ENABLED:-true}" in
  true|TRUE|1|yes|YES) ;;
  false|FALSE|0|no|NO)
    warn "student verification production readiness skipped because STUDENT_VERIFICATION_PRODUCTION_READINESS_ENABLED is not true"
    exit 0
    ;;
  *) die "STUDENT_VERIFICATION_PRODUCTION_READINESS_ENABLED must be true or false" ;;
esac

require_cmd docker
require_cmd python3
export COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-${STACK_NAME:-stuhelper}}"

student_verification_public_base_url="${STUDENT_VERIFICATION_PUBLIC_BASE_URL:-}"
[[ "${student_verification_public_base_url}" == "https://stuhelper.com" ]] || \
  die "STUDENT_VERIFICATION_PUBLIC_BASE_URL must be exactly https://stuhelper.com for production student verification"

case "${CAMPUS_CONNECTOR_GATEWAY_ENABLED:-false}" in
  true|TRUE|1|yes|YES) ;;
  false|FALSE|0|no|NO|"")
    die "CAMPUS_CONNECTOR_GATEWAY_ENABLED must be true for production student verification"
    ;;
  *) die "CAMPUS_CONNECTOR_GATEWAY_ENABLED must be true or false" ;;
esac

case "${EXTERNAL_STUDENT_SOURCE_ENABLED:-false}" in
  false|FALSE|0|no|NO|"") ;;
  true|TRUE|1|yes|YES)
    die "EXTERNAL_STUDENT_SOURCE_ENABLED must remain false; the online API may not connect to Oracle"
    ;;
  *) die "EXTERNAL_STUDENT_SOURCE_ENABLED must be true or false" ;;
esac

database_url="${STUDENT_VERIFICATION_READINESS_DATABASE_URL:-${DATABASE_URL:-}}"
[[ -n "${database_url}" ]] || \
  die "STUDENT_VERIFICATION_READINESS_DATABASE_URL or DATABASE_URL is required"

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
required_school_codes="${STUDENT_VERIFICATION_READINESS_REQUIRED_SCHOOL_CODES:-4111010006}"
required_methods="${STUDENT_VERIFICATION_READINESS_REQUIRED_METHODS:-real_name_identity_check,school_sso,manual_material_review}"
expected_sync_seconds="${STUDENT_VERIFICATION_READINESS_EXPECTED_SYNC_SECONDS:-604800}"
expected_warning_seconds="${STUDENT_VERIFICATION_READINESS_EXPECTED_WARNING_SECONDS:-691200}"
expected_hard_expiry_seconds="${STUDENT_VERIFICATION_READINESS_EXPECTED_HARD_EXPIRY_SECONDS:-1209600}"

for value_name in \
  expected_sync_seconds \
  expected_warning_seconds \
  expected_hard_expiry_seconds; do
  value="${!value_name}"
  [[ "${value}" =~ ^[1-9][0-9]*$ ]] || die "${value_name} must be a positive integer"
done
((expected_warning_seconds >= expected_sync_seconds)) || \
  die "expected warning threshold must not be lower than the sync interval"
((expected_hard_expiry_seconds > expected_warning_seconds)) || \
  die "expected hard expiry must be greater than the warning threshold"

run_readiness_sql() {
  compose --profile ops run --rm --no-deps -T \
    postgres-client \
    psql \
      -X \
      -v ON_ERROR_STOP=1 \
      -At \
      -v required_school_codes="${required_school_codes}" \
      -v required_methods="${required_methods}" \
      -v expected_sync_seconds="${expected_sync_seconds}" \
      -v expected_warning_seconds="${expected_warning_seconds}" \
      -v expected_hard_expiry_seconds="${expected_hard_expiry_seconds}" \
      "${database_url}" "$@"
}

failures="$(
  run_readiness_sql <<'SQL'
WITH
required_school_codes AS (
  SELECT DISTINCT trim(value) AS school_code
  FROM regexp_split_to_table(:'required_school_codes', ',') AS value
  WHERE trim(value) <> ''
),
required_methods AS (
  SELECT DISTINCT trim(value) AS method
  FROM regexp_split_to_table(:'required_methods', ',') AS value
  WHERE trim(value) <> ''
),
required_schools AS (
  SELECT
    required.school_code,
    school.id AS school_id,
    school.name AS school_name,
    profile.enabled AS profile_enabled,
    profile.validation_status AS profile_validation_status,
    profile.snapshot_sync_interval_seconds,
    profile.snapshot_warning_after_seconds,
    profile.snapshot_hard_expiry_seconds
  FROM required_school_codes required
  LEFT JOIN public.schools school ON school.code = required.school_code
  LEFT JOIN public.school_verification_profiles profile ON profile.school_id = school.id
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
    required.school_code,
    required.school_id,
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
    required.snapshot_hard_expiry_seconds,
    (
      SELECT count(*)
      FROM academic.student_roster_records record
      WHERE record.snapshot_id = snapshot.id
    ) AS actual_row_count,
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
  FROM required_schools required
  LEFT JOIN academic.student_roster_active active ON active.school_id = required.school_id
  LEFT JOIN academic.student_roster_snapshots snapshot
    ON snapshot.id = active.snapshot_id
   AND snapshot.school_id = required.school_id
),
failures(message) AS (
  SELECT 'STUDENT_VERIFICATION_READINESS_REQUIRED_SCHOOL_CODES must not be empty'
  WHERE NOT EXISTS (SELECT 1 FROM required_school_codes)

  UNION ALL
  SELECT 'STUDENT_VERIFICATION_READINESS_REQUIRED_METHODS must not be empty'
  WHERE NOT EXISTS (SELECT 1 FROM required_methods)

  UNION ALL
  SELECT format('required school code %s is missing from the school directory', school_code)
  FROM required_schools
  WHERE school_id IS NULL

  UNION ALL
  SELECT format('required school code %s has no verification profile', school_code)
  FROM required_schools
  WHERE school_id IS NOT NULL AND profile_validation_status IS NULL

  UNION ALL
  SELECT format('required school code %s verification profile is not enabled and valid', school_code)
  FROM required_schools
  WHERE profile_validation_status IS NOT NULL
    AND (profile_enabled IS DISTINCT FROM true OR profile_validation_status <> 'valid')

  UNION ALL
  SELECT format(
    'required school code %s snapshot schedule must be %s/%s/%s seconds',
    school_code,
    :'expected_sync_seconds',
    :'expected_warning_seconds',
    :'expected_hard_expiry_seconds'
  )
  FROM required_schools
  WHERE profile_validation_status IS NOT NULL
    AND (
      snapshot_sync_interval_seconds <> :'expected_sync_seconds'::integer
      OR snapshot_warning_after_seconds <> :'expected_warning_seconds'::integer
      OR snapshot_hard_expiry_seconds <> :'expected_hard_expiry_seconds'::integer
    )

  UNION ALL
  SELECT format('required school code %s method %s is missing or unavailable', school.school_code, method.method)
  FROM required_schools school
  CROSS JOIN required_methods method
  WHERE school.school_id IS NOT NULL
    AND NOT EXISTS (
      SELECT 1
      FROM public.school_verification_methods configured
      WHERE configured.school_id = school.school_id
        AND configured.method = method.method
        AND configured.enabled
        AND configured.validation_status = 'valid'
        AND configured.health_status IN ('healthy', 'degraded')
    )

  UNION ALL
  SELECT format('required school code %s method %s has no healthy approved connector operation', school.school_code, method.method)
  FROM required_schools school
  JOIN public.school_verification_methods method ON method.school_id = school.school_id
  WHERE method.enabled
    AND method.validation_status = 'valid'
    AND method.connector_operation_key IS NOT NULL
    AND NOT EXISTS (
      SELECT 1
      FROM available_connector_operations operation
      WHERE operation.school_id = school.school_id
        AND operation.operation_key = method.connector_operation_key
    )

  UNION ALL
  SELECT format('required school code %s has no healthy approved full roster connector operation', school.school_code)
  FROM required_schools school
  WHERE school.school_id IS NOT NULL
    AND NOT EXISTS (
      SELECT 1
      FROM available_connector_operations operation
      WHERE operation.school_id = school.school_id
        AND operation.operation_type = 'roster_snapshot_upload'
    )

  UNION ALL
  SELECT format('required school code %s has no active roster snapshot', school_code)
  FROM active_rosters
  WHERE snapshot_id IS NULL

  UNION ALL
  SELECT format('required school code %s active roster pointer references a non-active snapshot', school_code)
  FROM active_rosters
  WHERE snapshot_id IS NOT NULL AND status <> 'active'

  UNION ALL
  SELECT format('required school code %s active roster is not a connector full snapshot', school_code)
  FROM active_rosters
  WHERE snapshot_id IS NOT NULL
    AND (source_kind <> 'campus_connector' OR import_mode NOT IN ('full', 'reconciled_full'))

  UNION ALL
  SELECT format('required school code %s active roster snapshot is past the hard freshness threshold', school_code)
  FROM active_rosters
  WHERE snapshot_id IS NOT NULL
    AND source_cutoff_at + make_interval(secs => snapshot_hard_expiry_seconds) <= now()

  UNION ALL
  SELECT format('required school code %s active roster snapshot has invalid row counts', school_code)
  FROM active_rosters
  WHERE snapshot_id IS NOT NULL
    AND (
      row_count <= 0
      OR eligible_row_count <= 0
      OR eligible_row_count > row_count
      OR actual_row_count <> row_count
    )

  UNION ALL
  SELECT format('required school code %s active roster snapshot is unsigned or missing connector provenance', school_code)
  FROM active_rosters
  WHERE snapshot_id IS NOT NULL
    AND (
      checksum IS NULL
      OR signature_algorithm IS NULL
      OR signature_key_id IS NULL
      OR snapshot_signature IS NULL
      OR connector_node_id IS NULL
    )

  UNION ALL
  SELECT format('required school code %s active roster snapshot has incomplete or failed quality gates', school_code)
  FROM active_rosters
  WHERE snapshot_id IS NOT NULL
    AND (quality_check_count = 0 OR blocking_quality_check_count > 0)

  UNION ALL
  SELECT 'manual roster synchronization database uniqueness guard is missing'
  WHERE to_regclass('public.campus_connector_requests_manual_inflight_uidx') IS NULL
)
SELECT message
FROM failures
ORDER BY message;
SQL
)" || die "student verification production readiness query failed: ${failures//$'\n'/; }"

if [[ -n "${failures}" ]]; then
  die "student verification production readiness failed: ${failures//$'\n'/; }"
fi

summary="$(
  run_readiness_sql <<'SQL'
WITH
required_school_codes AS (
  SELECT DISTINCT trim(value) AS school_code
  FROM regexp_split_to_table(:'required_school_codes', ',') AS value
  WHERE trim(value) <> ''
),
required_schools AS (
  SELECT school.id AS school_id, school.code AS school_code
  FROM required_school_codes required
  JOIN public.schools school ON school.code = required.school_code
),
school_evidence AS (
  SELECT
    school.school_code,
    snapshot.source_cutoff_at,
    snapshot.row_count,
    snapshot.eligible_row_count,
    active.activation_revision,
    (
      SELECT count(*)
      FROM public.school_verification_methods method
      WHERE method.school_id = school.school_id
        AND method.enabled
        AND method.validation_status = 'valid'
        AND method.health_status IN ('healthy', 'degraded')
    ) AS enabled_method_count,
    (
      SELECT count(*)
      FROM public.campus_connector_school_operations operation
      JOIN public.campus_connector_nodes node ON node.id = operation.node_id
      WHERE operation.school_id = school.school_id
        AND operation.enabled
        AND operation.validation_status = 'valid'
        AND operation.health_status IN ('healthy', 'degraded')
        AND node.status IN ('active', 'degraded')
    ) AS available_connector_operation_count
  FROM required_schools school
  JOIN academic.student_roster_active active ON active.school_id = school.school_id
  JOIN academic.student_roster_snapshots snapshot
    ON snapshot.id = active.snapshot_id
   AND snapshot.school_id = school.school_id
)
SELECT json_build_object(
  'schemaVersion', 1,
  'generatedAt', to_char(clock_timestamp() AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"'),
  'status', 'passed',
  'requiredSchoolCount', (SELECT count(*) FROM required_schools),
  'schools', COALESCE((
    SELECT json_agg(
      json_build_object(
        'schoolCode', school_code,
        'sourceCutoffAt', source_cutoff_at,
        'rowCount', row_count,
        'eligibleRowCount', eligible_row_count,
        'activationRevision', activation_revision,
        'enabledMethodCount', enabled_method_count,
        'availableConnectorOperationCount', available_connector_operation_count
      ) ORDER BY school_code
    )
    FROM school_evidence
  ), '[]'::json)
)::text;
SQL
)" || die "student verification production readiness summary query failed"

printf '%s' "${summary}" | python3 -c '
import json
import sys

payload = json.load(sys.stdin)
if payload.get("status") != "passed":
    raise SystemExit("readiness summary status is not passed")
if not isinstance(payload.get("schools"), list) or not payload["schools"]:
    raise SystemExit("readiness summary contains no school evidence")
' || die "student verification production readiness summary is invalid"

evidence_file="${STUDENT_VERIFICATION_READINESS_EVIDENCE_FILE:-infra/generated/student-verification-production-readiness.json}"
if [[ "${evidence_file}" != /* ]]; then
  evidence_file="${REPO_ROOT_GUESS}/${evidence_file}"
fi
evidence_dir="$(dirname "${evidence_file}")"
mkdir -p "${evidence_dir}"
evidence_tmp="$(mktemp "${evidence_dir}/.student-verification-readiness.XXXXXX")"
cleanup() {
  [[ -n "${evidence_tmp:-}" && -e "${evidence_tmp}" ]] && rm -f "${evidence_tmp}"
}
trap cleanup EXIT
umask 077
printf '%s\n' "${summary}" >"${evidence_tmp}"
chmod 0600 "${evidence_tmp}"
mv -f "${evidence_tmp}" "${evidence_file}"
evidence_tmp=""

log "student verification production readiness passed; evidence=${evidence_file}"
