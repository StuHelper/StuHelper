#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
READINESS_SCRIPT="${REPO_ROOT}/infra/ops/admission-production-readiness.sh"
PROD_DEPLOY_SCRIPT="${REPO_ROOT}/infra/ops/prod-deploy.sh"
PROD_PARITY_SMOKE_SCRIPT="${REPO_ROOT}/infra/ops/prod-parity-smoke.sh"
PROD_PARITY_DATA_SCRIPT="${REPO_ROOT}/infra/ops/prod-parity-smoke-data.sh"
PROD_ENV_EXAMPLE="${REPO_ROOT}/.env.prod.example"

fail() {
  echo "[admission-production-readiness-contract][error] $*" >&2
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

for file in \
  "${READINESS_SCRIPT}" \
  "${PROD_DEPLOY_SCRIPT}" \
  "${PROD_PARITY_SMOKE_SCRIPT}" \
  "${PROD_PARITY_DATA_SCRIPT}" \
  "${PROD_ENV_EXAMPLE}"; do
  [[ -f "${file}" ]] || fail "missing file: ${file}"
done

bash -n "${READINESS_SCRIPT}"
[[ -x "${READINESS_SCRIPT}" ]] || fail "readiness script must be executable"

assert_contains "${PROD_ENV_EXAMPLE}" '^ADMISSION_PUBLIC_BASE_URL=https://join\.stuhelper\.com$'
assert_contains "${PROD_ENV_EXAMPLE}" '^ADMISSION_PRODUCTION_READINESS_ENABLED=true$'
assert_contains "${PROD_ENV_EXAMPLE}" '^ADMISSION_READINESS_REQUIRED_PLATFORM=qq$'
assert_contains "${PROD_ENV_EXAMPLE}" '^ADMISSION_READINESS_REQUIRED_SCHOOL_CODES=4111010006$'
assert_contains "${PROD_ENV_EXAMPLE}" '^ADMISSION_READINESS_REQUIRED_BOT_CREDENTIAL_NAME=koishi-runtime$'
assert_contains "${PROD_ENV_EXAMPLE}" '^ADMISSION_READINESS_REQUIRED_BOT_CREDENTIAL_AUDIENCE=/api/v1/bot/\*$'
assert_contains "${PROD_ENV_EXAMPLE}" '^ADMISSION_READINESS_REQUIRED_BOT_CREDENTIAL_SCOPES=.*bot\.admission\.session'
assert_contains "${PROD_ENV_EXAMPLE}" '^CORS_ORIGINS=.*https://join\.stuhelper\.com'
assert_contains "${PROD_ENV_EXAMPLE}" '^STUHELPER_FRESHMAN_MATERIAL_HOSTS=.*join\.stuhelper\.com'

assert_contains "${READINESS_SCRIPT}" 'ADMISSION_PRODUCTION_READINESS_ENABLED'
assert_contains "${READINESS_SCRIPT}" 'REPO_ROOT_GUESS'
assert_contains "${READINESS_SCRIPT}" '.env.prod.shared'
assert_contains "${READINESS_SCRIPT}" '.env.prod.secrets.local'
assert_contains "${READINESS_SCRIPT}" '.env.prod.generated.secrets'
assert_contains "${READINESS_SCRIPT}" 'COMPOSE_PROJECT_NAME'
assert_contains "${READINESS_SCRIPT}" 'REPLACE_WITH_STUHELPER_APP_DB_PASSWORD'
assert_contains "${READINESS_SCRIPT}" 'urllib.parse.quote'
assert_contains "${READINESS_SCRIPT}" 'ADMISSION_PUBLIC_BASE_URL must be exactly https://join.stuhelper.com'
assert_contains "${READINESS_SCRIPT}" 'ADMISSION_READINESS_DATABASE_URL'
assert_contains "${READINESS_SCRIPT}" 'ADMISSION_READINESS_REQUIRED_PLATFORM'
assert_contains "${READINESS_SCRIPT}" 'ADMISSION_READINESS_REQUIRED_GUILD_IDS'
assert_contains "${READINESS_SCRIPT}" 'ADMISSION_READINESS_REQUIRED_SCHOOL_CODES'
assert_contains "${READINESS_SCRIPT}" 'ADMISSION_READINESS_REQUIRED_SCHOOL_IDS'
assert_contains "${READINESS_SCRIPT}" 'ADMISSION_READINESS_REQUIRED_BOT_CREDENTIAL_NAME'
assert_contains "${READINESS_SCRIPT}" 'ADMISSION_READINESS_REQUIRED_BOT_CREDENTIAL_AUDIENCE'
assert_contains "${READINESS_SCRIPT}" 'ADMISSION_READINESS_REQUIRED_BOT_CREDENTIAL_SCOPES'
assert_contains "${READINESS_SCRIPT}" 'EXTERNAL_STUDENT_SOURCE_ENABLED'
assert_contains "${READINESS_SCRIPT}" 'EXTERNAL_STUDENT_SOURCE_ENABLED must remain false'
assert_contains "${READINESS_SCRIPT}" 'ADMISSION_READINESS_REQUIRED_METHODS'
assert_contains "${READINESS_SCRIPT}" 'ADMISSION_READINESS_ALLOW_FIXTURE_ROSTER'
assert_contains "${READINESS_SCRIPT}" 'APP_ENV=prod-parity'
assert_contains "${READINESS_SCRIPT}" 'bot_service_credentials'
assert_contains "${READINESS_SCRIPT}" 'bot service credential %s is missing'
assert_contains "${READINESS_SCRIPT}" 'bot service credential %s missing scope %s'
assert_contains "${READINESS_SCRIPT}" 'compose --profile prod run --rm --no-deps -T'
assert_contains "${READINESS_SCRIPT}" 'postgres-client'
assert_contains "${READINESS_SCRIPT}" 'psql'
assert_contains "${READINESS_SCRIPT}" 'school_verification_profiles'
assert_contains "${READINESS_SCRIPT}" 'school_verification_methods'
assert_contains "${READINESS_SCRIPT}" 'student_roster_active'
assert_contains "${READINESS_SCRIPT}" 'student_roster_snapshots'
assert_contains "${READINESS_SCRIPT}" 'student_roster_records'
assert_contains "${READINESS_SCRIPT}" 'group_admission_policies'
assert_contains "${READINESS_SCRIPT}" 'required %s guild %s has no group admission policy'
assert_contains "${READINESS_SCRIPT}" 'required school code %s is missing from the school directory'
assert_contains "${READINESS_SCRIPT}" 'required school code %s has no verification profile'
assert_contains "${READINESS_SCRIPT}" 'required school code %s method %s is missing or unavailable'
assert_contains "${READINESS_SCRIPT}" 'enabled school code %s is not in ADMISSION_READINESS_REQUIRED_SCHOOL_CODES'
assert_contains "${READINESS_SCRIPT}" 'active roster source must be Campus Connector'
assert_contains "${READINESS_SCRIPT}" 'active roster snapshot is past the hard freshness threshold'
assert_contains "${READINESS_SCRIPT}" 'active roster snapshot has invalid row counts'
assert_contains "${READINESS_SCRIPT}" 'active roster snapshot has incomplete or failed quality gates'
assert_contains "${READINESS_SCRIPT}" 'required method has no healthy approved connector operation'
assert_contains "${READINESS_SCRIPT}" 'freshman camera handoff table is missing'
assert_contains "${READINESS_SCRIPT}" 'freshman camera handoff table must expose token_hash, status and continue_on columns'
assert_contains "${READINESS_SCRIPT}" 'freshman camera handoff table must enforce one active handoff per application'
assert_contains "${READINESS_SCRIPT}" 'freshman_camera_handoffs_active_application_idx'
assert_contains "${READINESS_SCRIPT}" 'NULLIF\(trim\(sc\.academic_db_table\), '\'''\''\)'
assert_contains "${READINESS_SCRIPT}" 'references missing or disabled student-verification school'
assert_contains "${READINESS_SCRIPT}" 'has no required healthy student-verification method'
assert_contains "${READINESS_SCRIPT}" 'auto_approve_verified_join=true and auto_approve_unverified_join=true'
assert_contains "${READINESS_SCRIPT}" 'management_guild_ids must not be empty'
assert_contains "${READINESS_SCRIPT}" 'freshman_channel_enabled must be true'
assert_contains "${READINESS_SCRIPT}" 'freshman_channel_closes_at must be in the future'
assert_contains "${READINESS_SCRIPT}" 'freshman_default_expires_at must be in the future'
assert_contains "${READINESS_SCRIPT}" 'link_wait_seconds must be > 0'
assert_contains "${READINESS_SCRIPT}" 'submission_wait_seconds must be > 0'
assert_contains "${READINESS_SCRIPT}" 'failed_join_limit must be > 0'
assert_contains "${READINESS_SCRIPT}" 'max_material_bytes must be > 0'
assert_contains "${READINESS_SCRIPT}" 'admission production readiness passed: schools=%s policies=%s bot_credential=%s roster_source=%s'

assert_not_contains "${READINESS_SCRIPT}" 'command -v psql'
assert_not_contains "${READINESS_SCRIPT}" 'require_cmd psql'
assert_not_contains "${READINESS_SCRIPT}" 'localhost'
assert_not_contains "${READINESS_SCRIPT}" '127\.0\.0\.1'
assert_not_contains "${READINESS_SCRIPT}" 'FROM academic\.buaa_students'
assert_not_contains "${READINESS_SCRIPT}" 'buaa_academic_row_stats'
assert_not_contains "${READINESS_SCRIPT}" 'EXTERNAL_STUDENT_SOURCE_ORACLE_PASSWORD'

assert_contains "${PROD_DEPLOY_SCRIPT}" 'require_nonempty ADMISSION_PUBLIC_BASE_URL'
assert_contains "${PROD_DEPLOY_SCRIPT}" 'require_nonempty ADMISSION_PRODUCTION_READINESS_ENABLED'
assert_contains "${PROD_DEPLOY_SCRIPT}" 'ADMISSION_PUBLIC_BASE_URL must be exactly https://join.stuhelper.com for production deploy'
assert_contains "${PROD_DEPLOY_SCRIPT}" 'checking admission production readiness'
assert_contains "${PROD_DEPLOY_SCRIPT}" 'admission-production-readiness.sh'
assert_contains "${PROD_DEPLOY_SCRIPT}" 'admission-public-smoke.sh'
assert_contains "${PROD_PARITY_SMOKE_SCRIPT}" 'prod-parity-smoke-data.sh'
assert_contains "${PROD_PARITY_SMOKE_SCRIPT}" 'admission-production-readiness.sh'
assert_contains "${PROD_PARITY_SMOKE_SCRIPT}" 'ADMISSION_READINESS_REQUIRED_METHODS=student_email_outbound_otp'
assert_contains "${PROD_PARITY_SMOKE_SCRIPT}" 'ADMISSION_READINESS_ALLOW_FIXTURE_ROSTER=true'
assert_contains "${PROD_PARITY_SMOKE_SCRIPT}" 'admission-public-smoke.sh'
assert_contains "${PROD_PARITY_DATA_SCRIPT}" 'prod-parity-admission-policy'
assert_contains "${PROD_PARITY_DATA_SCRIPT}" 'format\('"'%s/verify/%s'"
assert_contains "${PROD_PARITY_DATA_SCRIPT}" 'auto_approve_verified_join'
assert_contains "${PROD_PARITY_DATA_SCRIPT}" 'auto_approve_unverified_join'
assert_contains "${PROD_PARITY_DATA_SCRIPT}" 'admissionSchoolConfigCount'
assert_contains "${PROD_PARITY_DATA_SCRIPT}" 'admissionPolicyCount'
assert_contains "${PROD_PARITY_DATA_SCRIPT}" 'admissionSessionCount'

echo "[admission-production-readiness-contract] all assertions passed"
