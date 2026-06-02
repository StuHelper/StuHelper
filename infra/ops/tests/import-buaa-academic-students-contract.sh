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
assert_contains "${IMPORT_SCRIPT}" 'BUAA_ACADEMIC_VALIDATE_ONLY'
assert_contains "${IMPORT_SCRIPT}" 'BUAA_ACADEMIC_DRY_RUN'
assert_contains "${IMPORT_SCRIPT}" 'validated_buaa_academic_students'
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

tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT

valid_tsv="${tmpdir}/valid.tsv"
cat >"${valid_tsv}" <<'TSV'
学号	姓名	学院代码
20250001	张三	BUAA01
20250002	李四	BUAA02
TSV

validate_output="$(
  BUAA_ACADEMIC_VALIDATE_ONLY=true \
  BUAA_ACADEMIC_STUDENTS_TSV="${valid_tsv}" \
  "${IMPORT_SCRIPT}"
)"
[[ "${validate_output}" == "validated_buaa_academic_students=2" ]] || \
  fail "unexpected validate-only output: ${validate_output}"

missing_column_tsv="${tmpdir}/missing-column.tsv"
cat >"${missing_column_tsv}" <<'TSV'
学号
20250001
TSV

if BUAA_ACADEMIC_VALIDATE_ONLY=true BUAA_ACADEMIC_STUDENTS_TSV="${missing_column_tsv}" "${IMPORT_SCRIPT}" >"${tmpdir}/missing.out" 2>"${tmpdir}/missing.err"; then
  fail "validate-only should reject TSV missing required xm column"
fi
grep -Eq 'missing required column\(s\): xm' "${tmpdir}/missing.err" || \
  fail "missing-column error did not explain xm requirement"

duplicate_tsv="${tmpdir}/duplicate.tsv"
cat >"${duplicate_tsv}" <<'TSV'
xh	xm
20250001	张三
20250001	李四
TSV

if BUAA_ACADEMIC_DRY_RUN=true BUAA_ACADEMIC_STUDENTS_TSV="${duplicate_tsv}" "${IMPORT_SCRIPT}" >"${tmpdir}/duplicate.out" 2>"${tmpdir}/duplicate.err"; then
  fail "validate-only should reject duplicate xh rows"
fi
grep -Eq 'duplicate xh 20250001' "${tmpdir}/duplicate.err" || \
  fail "duplicate-row error did not explain duplicate xh"

echo "[import-buaa-academic-students-contract] all assertions passed"
