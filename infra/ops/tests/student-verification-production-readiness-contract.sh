#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
READINESS_SCRIPT="${REPO_ROOT}/infra/ops/student-verification-production-readiness.sh"
PROD_DEPLOY_SCRIPT="${REPO_ROOT}/infra/ops/prod-deploy.sh"
PROD_ENV_EXAMPLE="${REPO_ROOT}/.env.prod.example"
DEV_ENV_EXAMPLE="${REPO_ROOT}/.env.example"
MAKEFILE="${REPO_ROOT}/Makefile"
GO_LIVE_DOC="${REPO_ROOT}/docs/guides/production-go-live.md"
RELEASE_RUNBOOK="${REPO_ROOT}/docs/guides/release-runbook.md"

fail() {
  echo "[student-verification-production-readiness-contract][error] $*" >&2
  exit 1
}

assert_contains() {
  local file="$1"
  local pattern="$2"
  if ! grep -Eq -- "${pattern}" "${file}"; then
    fail "expected ${file} to contain pattern: ${pattern}"
  fi
}

assert_not_contains() {
  local file="$1"
  local pattern="$2"
  if grep -Eq -- "${pattern}" "${file}"; then
    fail "expected ${file} not to contain pattern: ${pattern}"
  fi
}

for file in \
  "${READINESS_SCRIPT}" \
  "${PROD_DEPLOY_SCRIPT}" \
  "${PROD_ENV_EXAMPLE}" \
  "${DEV_ENV_EXAMPLE}" \
  "${MAKEFILE}" \
  "${GO_LIVE_DOC}" \
  "${RELEASE_RUNBOOK}"; do
  [[ -f "${file}" ]] || fail "missing file: ${file}"
done

bash -n "${READINESS_SCRIPT}"
[[ -x "${READINESS_SCRIPT}" ]] || fail "readiness script must be executable"

assert_contains "${PROD_ENV_EXAMPLE}" '^STUDENT_VERIFICATION_PRODUCTION_READINESS_ENABLED=true$'
assert_contains "${PROD_ENV_EXAMPLE}" '^STUDENT_VERIFICATION_READINESS_REQUIRED_SCHOOL_CODES=4111010006$'
assert_contains "${PROD_ENV_EXAMPLE}" '^STUDENT_VERIFICATION_READINESS_REQUIRED_METHODS=real_name_identity_check,school_sso,manual_material_review$'
assert_contains "${PROD_ENV_EXAMPLE}" '^STUDENT_VERIFICATION_READINESS_EXPECTED_SYNC_SECONDS=604800$'
assert_contains "${PROD_ENV_EXAMPLE}" '^STUDENT_VERIFICATION_READINESS_EXPECTED_WARNING_SECONDS=691200$'
assert_contains "${PROD_ENV_EXAMPLE}" '^STUDENT_VERIFICATION_READINESS_EXPECTED_HARD_EXPIRY_SECONDS=1209600$'
assert_contains "${PROD_ENV_EXAMPLE}" '^STUDENT_VERIFICATION_READINESS_EVIDENCE_FILE=infra/generated/student-verification-production-readiness\.json$'
assert_contains "${DEV_ENV_EXAMPLE}" '^STUDENT_VERIFICATION_PRODUCTION_READINESS_ENABLED=false$'

assert_contains "${READINESS_SCRIPT}" 'STUDENT_VERIFICATION_PRODUCTION_READINESS_ENABLED'
assert_contains "${READINESS_SCRIPT}" 'STUDENT_VERIFICATION_PUBLIC_BASE_URL must be exactly https://stuhelper\.com'
assert_contains "${READINESS_SCRIPT}" 'CAMPUS_CONNECTOR_GATEWAY_ENABLED must be true'
assert_contains "${READINESS_SCRIPT}" 'EXTERNAL_STUDENT_SOURCE_ENABLED must remain false'
assert_contains "${READINESS_SCRIPT}" 'STUDENT_VERIFICATION_READINESS_DATABASE_URL'
assert_contains "${READINESS_SCRIPT}" 'REPLACE_WITH_STUHELPER_APP_DB_PASSWORD'
assert_contains "${READINESS_SCRIPT}" 'urllib\.parse\.quote'
assert_contains "${READINESS_SCRIPT}" 'compose --profile ops run --rm --no-deps -T'
assert_contains "${READINESS_SCRIPT}" 'school_verification_profiles'
assert_contains "${READINESS_SCRIPT}" 'school_verification_methods'
assert_contains "${READINESS_SCRIPT}" 'academic\.student_roster_active'
assert_contains "${READINESS_SCRIPT}" 'academic\.student_roster_snapshots'
assert_contains "${READINESS_SCRIPT}" 'academic\.student_roster_records'
assert_contains "${READINESS_SCRIPT}" 'academic\.student_roster_quality_checks'
assert_contains "${READINESS_SCRIPT}" 'campus_connector_school_operations'
assert_contains "${READINESS_SCRIPT}" 'campus_connector_nodes'
assert_contains "${READINESS_SCRIPT}" 'certificate_not_after > now\(\) \+ interval '\''30 days'\'''
assert_contains "${READINESS_SCRIPT}" 'last_heartbeat_at >= now\(\) - make_interval'
assert_contains "${READINESS_SCRIPT}" "source_kind <> 'campus_connector'"
assert_contains "${READINESS_SCRIPT}" "import_mode NOT IN \('full', 'reconciled_full'\)"
assert_contains "${READINESS_SCRIPT}" 'actual_row_count <> row_count'
assert_contains "${READINESS_SCRIPT}" 'blocking_quality_check_count > 0'
assert_contains "${READINESS_SCRIPT}" 'source_cutoff_at \+ make_interval.*snapshot_hard_expiry_seconds'
assert_contains "${READINESS_SCRIPT}" 'campus_connector_requests_manual_inflight_uidx'
assert_contains "${READINESS_SCRIPT}" 'student-verification-production-readiness\.json'
assert_contains "${READINESS_SCRIPT}" 'chmod 0600'
assert_contains "${READINESS_SCRIPT}" 'mv -f'
assert_not_contains "${READINESS_SCRIPT}" 'academic\.buaa_students'
assert_not_contains "${READINESS_SCRIPT}" 'EXTERNAL_STUDENT_SOURCE_ORACLE_HOST'
assert_not_contains "${READINESS_SCRIPT}" 'secret_reference'
assert_not_contains "${READINESS_SCRIPT}" 'command -v psql'
assert_not_contains "${READINESS_SCRIPT}" 'require_cmd psql'

assert_contains "${PROD_DEPLOY_SCRIPT}" 'require_nonempty STUDENT_VERIFICATION_PRODUCTION_READINESS_ENABLED'
assert_contains "${PROD_DEPLOY_SCRIPT}" 'checking student verification production readiness'
assert_contains "${PROD_DEPLOY_SCRIPT}" 'student-verification-production-readiness\.sh'
assert_contains "${MAKEFILE}" '^prod-student-verification-readiness:'
assert_contains "${MAKEFILE}" 'prod-student-verification-readiness.*verify profiles, methods, connector, and active roster'
assert_contains "${GO_LIVE_DOC}" 'make prod-student-verification-readiness'
assert_contains "${GO_LIVE_DOC}" 'student-verification-production-readiness\.json'
assert_contains "${RELEASE_RUNBOOK}" 'make prod-student-verification-readiness'

echo "[student-verification-production-readiness-contract] all assertions passed"
