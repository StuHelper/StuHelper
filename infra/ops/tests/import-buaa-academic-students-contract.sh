#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

IMPORT_SCRIPT="${REPO_ROOT}/infra/ops/import-buaa-academic-students.sh"
READINESS_SCRIPT="${REPO_ROOT}/infra/ops/admission-production-readiness.sh"
GO_LIVE_DOC="${REPO_ROOT}/docs/guides/production-go-live.md"
RELEASE_RUNBOOK="${REPO_ROOT}/docs/guides/release-runbook.md"

fail() {
  echo "[import-buaa-academic-students-contract][error] $*" >&2
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
    fail "expected ${file} to not contain pattern: ${pattern}"
  fi
}

for file in "${IMPORT_SCRIPT}" "${READINESS_SCRIPT}" "${GO_LIVE_DOC}" "${RELEASE_RUNBOOK}"; do
  [[ -f "${file}" ]] || fail "missing file: ${file}"
done

bash -n "${IMPORT_SCRIPT}"
[[ -x "${IMPORT_SCRIPT}" ]] || fail "import script must be executable"

assert_contains "${IMPORT_SCRIPT}" 'BUAA_ACADEMIC_STUDENTS_TSV is required'
assert_contains "${IMPORT_SCRIPT}" 'BUAA_ACADEMIC_DATABASE_URL'
assert_contains "${IMPORT_SCRIPT}" 'BUAA_ACADEMIC_MIN_ROWS'
assert_contains "${IMPORT_SCRIPT}" 'studentID'
assert_contains "${IMPORT_SCRIPT}" '学号'
assert_contains "${IMPORT_SCRIPT}" '姓名'
assert_contains "${IMPORT_SCRIPT}" 'duplicate xh'
assert_contains "${IMPORT_SCRIPT}" 'ON CONFLICT \(xh\) DO UPDATE'
assert_contains "${IMPORT_SCRIPT}" 'ANALYZE academic\.buaa_students'
assert_contains "${IMPORT_SCRIPT}" 'imported_buaa_academic_students'
assert_contains "${IMPORT_SCRIPT}" 'total_buaa_academic_students'
assert_contains "${IMPORT_SCRIPT}" 'sfzjh_enc'
assert_not_contains "${IMPORT_SCRIPT}" 'TRUNCATE academic\.buaa_students'
assert_not_contains "${IMPORT_SCRIPT}" 'DELETE FROM academic\.buaa_students'

assert_contains "${READINESS_SCRIPT}" 'BUAA academic table academic\.buaa_students has no rows'
assert_contains "${GO_LIVE_DOC}" 'import-buaa-academic-students\.sh'
assert_contains "${RELEASE_RUNBOOK}" 'import-buaa-academic-students\.sh'

echo "[import-buaa-academic-students-contract] all assertions passed"
