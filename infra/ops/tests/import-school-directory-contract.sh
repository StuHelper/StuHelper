#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

IMPORT_SCRIPT="${REPO_ROOT}/infra/ops/import-school-directory.sh"
DIRECTORY_TSV="${REPO_ROOT}/server/data/school_directory_2025.tsv"
BOOTSTRAP_SCRIPT="${REPO_ROOT}/infra/ops/admission-bootstrap-production-data.sh"
SCHOOL_DIRECTORY_MIGRATION="${REPO_ROOT}/server/migrations/000003_school_directory_columns.up.sql"

assert_file_contains() {
  local file="$1"
  local pattern="$2"
  if ! grep -Eq -- "${pattern}" "${file}"; then
    echo "expected ${file} to contain pattern: ${pattern}" >&2
    exit 1
  fi
}

assert_file_not_contains() {
  local file="$1"
  local pattern="$2"
  if grep -Eq -- "${pattern}" "${file}"; then
    echo "expected ${file} to not contain pattern: ${pattern}" >&2
    exit 1
  fi
}

[[ -f "${IMPORT_SCRIPT}" ]] || { echo "missing import script: ${IMPORT_SCRIPT}" >&2; exit 1; }
[[ -f "${DIRECTORY_TSV}" ]] || { echo "missing school directory TSV: ${DIRECTORY_TSV}" >&2; exit 1; }
[[ -f "${BOOTSTRAP_SCRIPT}" ]] || { echo "missing admission bootstrap script: ${BOOTSTRAP_SCRIPT}" >&2; exit 1; }
[[ -f "${SCHOOL_DIRECTORY_MIGRATION}" ]] || { echo "missing school directory migration: ${SCHOOL_DIRECTORY_MIGRATION}" >&2; exit 1; }

head -n 1 "${DIRECTORY_TSV}" | grep -Fx $'code\tname\tauthority\tlocation\teducation_level\tremark' >/dev/null || {
  echo "school directory TSV header is invalid" >&2
  exit 1
}

assert_file_contains "${DIRECTORY_TSV}" $'^4111010006\t北京航空航天大学\t工业和信息化部\t北京市\t本科\t""$'
assert_file_contains "${DIRECTORY_TSV}" $'^4111010001\t北京大学\t教育部\t北京市\t本科\t""$'

school_count="$(tail -n +2 "${DIRECTORY_TSV}" | wc -l | tr -d ' ')"
if [[ "${school_count}" -lt 2500 ]]; then
  echo "school directory TSV looks incomplete: ${school_count} schools" >&2
  exit 1
fi

assert_file_contains "${IMPORT_SCRIPT}" 'The imported directory is not an admission whitelist'
assert_file_contains "${IMPORT_SCRIPT}" 'INSERT INTO public\.schools'
assert_file_not_contains "${IMPORT_SCRIPT}" 'INSERT INTO public\.school_configs'
assert_file_not_contains "${IMPORT_SCRIPT}" 'UPDATE public\.school_configs'

assert_file_contains "${BOOTSTRAP_SCRIPT}" 'ADMISSION_BOOTSTRAP_SCHOOL_CODE.*4111010006'
assert_file_contains "${BOOTSTRAP_SCRIPT}" 'ADMISSION_BOOTSTRAP_SCHOOL_ID.*10006'
assert_file_contains "${BOOTSTRAP_SCRIPT}" 'enabled = true'

assert_file_contains "${SCHOOL_DIRECTORY_MIGRATION}" 'It is not an admission whitelist'
assert_file_contains "${SCHOOL_DIRECTORY_MIGRATION}" "code = '4111010006'"
assert_file_contains "${SCHOOL_DIRECTORY_MIGRATION}" "academic_db_table = 'academic.buaa_students'"
assert_file_contains "${SCHOOL_DIRECTORY_MIGRATION}" "enabled = true"
assert_file_contains "${SCHOOL_DIRECTORY_MIGRATION}" "school_id <> 10006"
assert_file_contains "${SCHOOL_DIRECTORY_MIGRATION}" "studentIDEmailDomain"
