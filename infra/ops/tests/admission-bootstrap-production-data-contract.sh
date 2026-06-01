#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
BOOTSTRAP_SCRIPT="${REPO_ROOT}/infra/ops/admission-bootstrap-production-data.sh"
PROD_ENV_EXAMPLE="${REPO_ROOT}/.env.prod.example"
PROD_GO_LIVE="${REPO_ROOT}/docs/guides/production-go-live.md"
RELEASE_RUNBOOK="${REPO_ROOT}/docs/guides/release-runbook.md"

fail() {
  echo "[admission-bootstrap-production-data-contract][error] $*" >&2
  exit 1
}

assert_contains() {
  local file="$1"
  local pattern="$2"
  if ! grep -Eq "${pattern}" "${file}"; then
    fail "expected ${file} to contain pattern: ${pattern}"
  fi
}

for file in \
  "${BOOTSTRAP_SCRIPT}" \
  "${PROD_ENV_EXAMPLE}" \
  "${PROD_GO_LIVE}" \
  "${RELEASE_RUNBOOK}"; do
  [[ -f "${file}" ]] || fail "missing file: ${file}"
done

bash -n "${BOOTSTRAP_SCRIPT}"

assert_contains "${BOOTSTRAP_SCRIPT}" 'ADMISSION_BOOTSTRAP_DATABASE_URL'
assert_contains "${BOOTSTRAP_SCRIPT}" 'REPO_ROOT_GUESS'
assert_contains "${BOOTSTRAP_SCRIPT}" '.env.prod.shared'
assert_contains "${BOOTSTRAP_SCRIPT}" '.env.prod.secrets.local'
assert_contains "${BOOTSTRAP_SCRIPT}" '.env.prod.generated.secrets'
assert_contains "${BOOTSTRAP_SCRIPT}" 'COMPOSE_PROJECT_NAME'
assert_contains "${BOOTSTRAP_SCRIPT}" 'REPLACE_WITH_STUHELPER_APP_DB_PASSWORD'
assert_contains "${BOOTSTRAP_SCRIPT}" 'urllib.parse.quote'
assert_contains "${BOOTSTRAP_SCRIPT}" 'ADMISSION_BOOTSTRAP_PLATFORM.*qq'
assert_contains "${BOOTSTRAP_SCRIPT}" 'ADMISSION_BOOTSTRAP_SCHOOL_ID.*10006'
assert_contains "${BOOTSTRAP_SCRIPT}" 'ADMISSION_BOOTSTRAP_GROUP_IDS.*178037297'
assert_contains "${BOOTSTRAP_SCRIPT}" 'ADMISSION_BOOTSTRAP_EMAIL_DOMAINS.*buaa.edu.cn'
assert_contains "${BOOTSTRAP_SCRIPT}" 'ADMISSION_BOOTSTRAP_DISABLE_OTHER_SCHOOLS.*true'
assert_contains "${BOOTSTRAP_SCRIPT}" 'ADMISSION_BOOTSTRAP_PRUNE_OTHER_GROUP_POLICIES.*true'
assert_contains "${BOOTSTRAP_SCRIPT}" 'group_admission_policies'
assert_contains "${BOOTSTRAP_SCRIPT}" 'school_configs'
assert_contains "${BOOTSTRAP_SCRIPT}" "emailDomains"
assert_contains "${BOOTSTRAP_SCRIPT}" "emailIdentityPolicy"
assert_contains "${BOOTSTRAP_SCRIPT}" "academic_student_email"
assert_contains "${BOOTSTRAP_SCRIPT}" "academic_db_table = EXCLUDED.academic_db_table"
assert_contains "${BOOTSTRAP_SCRIPT}" "disabled_other_school_configs"
assert_contains "${BOOTSTRAP_SCRIPT}" "sc.school_id <> input.school_id"
assert_contains "${BOOTSTRAP_SCRIPT}" "prune_other_group_policies"
assert_contains "${BOOTSTRAP_SCRIPT}" "DELETE FROM public.group_admission_policies"
assert_contains "${BOOTSTRAP_SCRIPT}" "forward_raw_material_to_qq = false"
assert_contains "${BOOTSTRAP_SCRIPT}" 'ON CONFLICT \(platform, guild_id\) DO UPDATE'
assert_contains "${BOOTSTRAP_SCRIPT}" 'compose --profile prod run --rm --no-deps -T'
assert_contains "${BOOTSTRAP_SCRIPT}" 'No secret values are written'

assert_contains "${PROD_ENV_EXAMPLE}" '^ADMISSION_READINESS_REQUIRED_GUILD_IDS=178037297$'
assert_contains "${PROD_ENV_EXAMPLE}" '^ADMISSION_READINESS_REQUIRED_SCHOOL_CODES=4111010006$'

assert_contains "${PROD_GO_LIVE}" 'admission-bootstrap-production-data.sh'
assert_contains "${RELEASE_RUNBOOK}" 'admission-bootstrap-production-data.sh'
assert_contains "${RELEASE_RUNBOOK}" 'buaa.edu.cn'
assert_contains "${RELEASE_RUNBOOK}" '178037297'

echo "[admission-bootstrap-production-data-contract] all assertions passed"
