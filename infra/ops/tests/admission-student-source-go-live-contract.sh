#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
GATE_SCRIPT="${REPO_ROOT}/infra/ops/admission-student-source-go-live.sh"
GO_LIVE_DOC="${REPO_ROOT}/docs/guides/production-go-live.md"
RELEASE_RUNBOOK="${REPO_ROOT}/docs/guides/release-runbook.md"

fail() {
  echo "[admission-student-source-go-live-contract][error] $*" >&2
  exit 1
}

assert_contains() {
  local file="$1"
  local pattern="$2"
  if ! grep -Eq -- "${pattern}" "${file}"; then
    fail "expected ${file} to contain pattern: ${pattern}"
  fi
}

cleanup() {
  [[ -n "${tmpdir:-}" ]] && rm -rf "${tmpdir}"
}
trap cleanup EXIT

[[ -f "${GATE_SCRIPT}" ]] || fail "missing gate script: ${GATE_SCRIPT}"
bash -n "${GATE_SCRIPT}"
[[ -x "${GATE_SCRIPT}" ]] || fail "gate script must be executable"

assert_contains "${GATE_SCRIPT}" 'ADMISSION_STUDENT_SOURCE_MODE'
assert_contains "${GATE_SCRIPT}" 'EXTERNAL_STUDENT_SOURCE_ENABLED'
assert_contains "${GATE_SCRIPT}" 'BUAA_ACADEMIC_STUDENTS_TSV'
assert_contains "${GATE_SCRIPT}" 'external-student-source-smoke\.sh'
assert_contains "${GATE_SCRIPT}" 'import-buaa-academic-students\.sh'
assert_contains "${GATE_SCRIPT}" 'admission-production-readiness\.sh'
assert_contains "${GATE_SCRIPT}" 'BUAA_ACADEMIC_VALIDATE_ONLY=true'
assert_contains "${GATE_SCRIPT}" 'no student source selected'
assert_contains "${GATE_SCRIPT}" 'admission student source go-live gate passed'

tmpdir="$(mktemp -d)"
fake_repo="${tmpdir}/repo"
mkdir -p "${fake_repo}/infra/ops/lib"
cp "${GATE_SCRIPT}" "${fake_repo}/infra/ops/admission-student-source-go-live.sh"

cat >"${fake_repo}/infra/ops/lib/common.sh" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
COMMON_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${COMMON_LIB_DIR}/../../.." && pwd)"
log() { echo "[fake] $*"; }
warn() { echo "[fake][warn] $*" >&2; }
die() { echo "[fake][error] $*" >&2; exit 1; }
load_env() { :; }
SH
chmod +x "${fake_repo}/infra/ops/lib/common.sh"

for script in external-student-source-smoke.sh import-buaa-academic-students.sh admission-production-readiness.sh; do
  cat >"${fake_repo}/infra/ops/${script}" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
printf '%s validate=%s tsv=%s\n' "$(basename "$0")" "${BUAA_ACADEMIC_VALIDATE_ONLY:-}" "${BUAA_ACADEMIC_STUDENTS_TSV:-}"
SH
  chmod +x "${fake_repo}/infra/ops/${script}"
done

external_output="$(
  EXTERNAL_STUDENT_SOURCE_ENABLED=true \
  "${fake_repo}/infra/ops/admission-student-source-go-live.sh"
)"
grep -q 'external-student-source-smoke.sh' <<<"${external_output}" || fail "external mode did not run smoke"
grep -q 'admission-production-readiness.sh' <<<"${external_output}" || fail "external mode did not run readiness"
if grep -q 'import-buaa-academic-students.sh' <<<"${external_output}"; then
  fail "external mode must not run local TSV import"
fi

local_tsv="${tmpdir}/buaa.tsv"
printf 'xh\txm\n100001\tTest User\n' >"${local_tsv}"
local_output="$(
  BUAA_ACADEMIC_STUDENTS_TSV="${local_tsv}" \
  "${fake_repo}/infra/ops/admission-student-source-go-live.sh"
)"
grep -q 'import-buaa-academic-students.sh validate=true' <<<"${local_output}" || fail "local mode did not validate TSV first"
grep -q 'import-buaa-academic-students.sh validate= ' <<<"${local_output}" || fail "local mode did not import TSV after validation"
grep -q 'admission-production-readiness.sh' <<<"${local_output}" || fail "local mode did not run readiness"
if grep -q 'external-student-source-smoke.sh' <<<"${local_output}"; then
  fail "local mode must not run external smoke"
fi

if ADMISSION_STUDENT_SOURCE_MODE=local \
  "${fake_repo}/infra/ops/admission-student-source-go-live.sh" >/tmp/admission-student-source-local-missing.stdout 2>/tmp/admission-student-source-local-missing.stderr; then
  fail "local mode unexpectedly passed without BUAA_ACADEMIC_STUDENTS_TSV"
fi
assert_contains /tmp/admission-student-source-local-missing.stderr 'BUAA_ACADEMIC_STUDENTS_TSV is required'

if "${fake_repo}/infra/ops/admission-student-source-go-live.sh" >/tmp/admission-student-source-auto-missing.stdout 2>/tmp/admission-student-source-auto-missing.stderr; then
  fail "auto mode unexpectedly passed without any student source"
fi
assert_contains /tmp/admission-student-source-auto-missing.stderr 'no student source selected'

assert_contains "${GO_LIVE_DOC}" 'admission-student-source-go-live\.sh'
assert_contains "${RELEASE_RUNBOOK}" 'admission-student-source-go-live\.sh'

echo "[admission-student-source-go-live-contract] all assertions passed"
