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
Usage: infra/ops/admission-student-source-go-live.sh

Runs the student-source gate needed before admission MVP go-live.

Modes:
  auto      Use external when EXTERNAL_STUDENT_SOURCE_ENABLED=true. Otherwise
            use local TSV when BUAA_ACADEMIC_STUDENTS_TSV is set. This is the
            default.
  external  Run external-student-source-smoke.sh, then admission readiness.
  local     Validate and import BUAA_ACADEMIC_STUDENTS_TSV, then readiness.

Environment:
  ADMISSION_STUDENT_SOURCE_MODE=auto|external|local
  BUAA_ACADEMIC_STUDENTS_TSV=/path/to/buaa-students.tsv  required for local
  BUAA_ACADEMIC_MIN_ROWS=1                               optional local guard

The script does not accept or print raw student records, passwords, or tokens.
It is intentionally a gate/orchestration wrapper; source-of-truth checks remain
external-student-source-smoke.sh, import-buaa-academic-students.sh, and
admission-production-readiness.sh.
USAGE
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
  exit 0
fi

normalize_mode() {
  case "${1:-auto}" in
    auto|external|local) printf '%s\n' "$1" ;;
    *) die "ADMISSION_STUDENT_SOURCE_MODE must be auto, external, or local" ;;
  esac
}

is_truthy() {
  case "${1:-}" in
    true|TRUE|1|yes|YES|on|ON) return 0 ;;
    *) return 1 ;;
  esac
}

load_env

mode="$(normalize_mode "${ADMISSION_STUDENT_SOURCE_MODE:-auto}")"
if [[ "${mode}" == "auto" ]]; then
  if is_truthy "${EXTERNAL_STUDENT_SOURCE_ENABLED:-false}"; then
    mode="external"
  elif [[ -n "${BUAA_ACADEMIC_STUDENTS_TSV:-}" ]]; then
    mode="local"
  else
    die "no student source selected: enable EXTERNAL_STUDENT_SOURCE_ENABLED=true or set BUAA_ACADEMIC_STUDENTS_TSV"
  fi
fi

case "${mode}" in
  external)
    log "running external student source smoke"
    "${SCRIPT_DIR}/external-student-source-smoke.sh"
    log "running admission production readiness"
    "${SCRIPT_DIR}/admission-production-readiness.sh"
    ;;
  local)
    [[ -n "${BUAA_ACADEMIC_STUDENTS_TSV:-}" ]] || die "BUAA_ACADEMIC_STUDENTS_TSV is required in local mode"
    log "validating BUAA academic TSV"
    BUAA_ACADEMIC_VALIDATE_ONLY=true "${SCRIPT_DIR}/import-buaa-academic-students.sh"
    log "importing BUAA academic TSV"
    "${SCRIPT_DIR}/import-buaa-academic-students.sh"
    log "running admission production readiness"
    "${SCRIPT_DIR}/admission-production-readiness.sh"
    ;;
esac

log "admission student source go-live gate passed: mode=${mode}"
