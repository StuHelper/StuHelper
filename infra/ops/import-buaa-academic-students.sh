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
Usage: infra/ops/import-buaa-academic-students.sh

Imports or updates BUAA student records into academic.buaa_students.

Input:
  BUAA_ACADEMIC_STUDENTS_TSV is required. The file must be tab-separated and
  contain at least xh and xm columns. Chinese aliases 学号 and 姓名 are accepted.

Optional:
  BUAA_ACADEMIC_DATABASE_URL defaults to DATABASE_URL.
  BUAA_ACADEMIC_MIN_ROWS defaults to 1.
  BUAA_ACADEMIC_VALIDATE_ONLY=true validates and normalizes the TSV without
    requiring Docker, loading production env files, or writing to the database.
  BUAA_ACADEMIC_DRY_RUN=true is accepted as an alias for validate-only.

Supported optional columns:
  sfzjlxdm, sfzjh_hash, yxdm, zydm, bjdm, xznj, rxnj, pyccdm, xslbdm, sjh,
  dzxx, xjztdm, sfzx, sfzj

sfzjh_enc is intentionally not imported from this TSV. Use a dedicated encrypted
identity sync path for encrypted ID document numbers.

The script performs an idempotent upsert and does not print student details.
USAGE
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
  exit 0
fi

require_cmd python3

input_tsv="${BUAA_ACADEMIC_STUDENTS_TSV:-}"
[[ -n "${input_tsv}" ]] || die "BUAA_ACADEMIC_STUDENTS_TSV is required"
[[ -f "${input_tsv}" ]] || die "BUAA academic TSV not found: ${input_tsv}"

min_rows="${BUAA_ACADEMIC_MIN_ROWS:-1}"
[[ "${min_rows}" =~ ^[0-9]+$ && "${min_rows}" -gt 0 ]] || die "BUAA_ACADEMIC_MIN_ROWS must be a positive integer"

validate_only="${BUAA_ACADEMIC_VALIDATE_ONLY:-${BUAA_ACADEMIC_DRY_RUN:-false}}"

case "${validate_only}" in
  1|true|TRUE|True|yes|YES|Yes|on|ON|On)
    validate_only="true"
    ;;
  0|false|FALSE|False|no|NO|No|off|OFF|Off|"")
    validate_only="false"
    ;;
  *)
    die "BUAA_ACADEMIC_VALIDATE_ONLY must be true or false"
    ;;
esac

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

tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT
normalized_tsv="${tmpdir}/buaa_academic_students.normalized.tsv"

normalized_count="$(
  python3 - "${input_tsv}" "${normalized_tsv}" "${min_rows}" <<'PY'
import csv
import sys
from pathlib import Path

source = Path(sys.argv[1])
target = Path(sys.argv[2])
min_rows = int(sys.argv[3])

columns = [
    "xh",
    "xm",
    "sfzjlxdm",
    "sfzjh_hash",
    "yxdm",
    "zydm",
    "bjdm",
    "xznj",
    "rxnj",
    "pyccdm",
    "xslbdm",
    "sjh",
    "dzxx",
    "xjztdm",
    "sfzx",
    "sfzj",
]
aliases = {
    "xh": ["xh", "student_id", "studentID", "学号"],
    "xm": ["xm", "name", "student_name", "studentName", "姓名"],
}

with source.open("r", encoding="utf-8-sig", newline="") as fh:
    reader = csv.DictReader(fh, delimiter="\t")
    if not reader.fieldnames:
        raise SystemExit("BUAA academic TSV must have a header row")

    header = {name.strip(): name for name in reader.fieldnames if name is not None}

    def source_key(column: str) -> str | None:
        for candidate in aliases.get(column, [column]):
            if candidate in header:
                return header[candidate]
        return header.get(column)

    resolved = {column: source_key(column) for column in columns}
    missing = [column for column in ["xh", "xm"] if resolved[column] is None]
    if missing:
        raise SystemExit(f"BUAA academic TSV missing required column(s): {', '.join(missing)}")

    rows = []
    seen_xh: set[str] = set()
    for line_number, raw in enumerate(reader, 2):
        row = {}
        for column in columns:
            key = resolved[column]
            row[column] = (raw.get(key, "") if key else "").strip()
        if not any(row.values()):
            continue
        if not row["xh"]:
            raise SystemExit(f"{source}:{line_number}: xh must not be empty")
        if not row["xm"]:
            raise SystemExit(f"{source}:{line_number}: xm must not be empty")
        if row["xh"] in seen_xh:
            raise SystemExit(f"{source}:{line_number}: duplicate xh {row['xh']}")
        seen_xh.add(row["xh"])
        rows.append(row)

if len(rows) < min_rows:
    raise SystemExit(f"BUAA academic TSV has {len(rows)} row(s), below BUAA_ACADEMIC_MIN_ROWS={min_rows}")

with target.open("w", encoding="utf-8", newline="") as fh:
    writer = csv.DictWriter(fh, fieldnames=columns, delimiter="\t", lineterminator="\n")
    writer.writeheader()
    writer.writerows(rows)

print(len(rows))
PY
)"

if [[ "${validate_only}" == "true" ]]; then
  printf 'validated_buaa_academic_students=%s\n' "${normalized_count}"
  exit 0
fi

require_cmd docker

load_env
export COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-${STACK_NAME:-stuhelper}}"

database_url="${BUAA_ACADEMIC_DATABASE_URL:-${DATABASE_URL:-}}"
[[ -n "${database_url}" ]] || die "BUAA_ACADEMIC_DATABASE_URL or DATABASE_URL is required"
database_url="$(materialize_database_url "${database_url}")"

compose --profile prod run --rm --no-deps -T \
  -v "${normalized_tsv}:/tmp/buaa_academic_students.normalized.tsv:ro" \
  postgres-client \
  psql \
    -X \
    -v ON_ERROR_STOP=1 \
    -v normalized_count="${normalized_count}" \
    "${database_url}" <<'SQL'
CREATE TEMP TABLE tmp_buaa_academic_students (
  xh text NOT NULL,
  xm text NOT NULL,
  sfzjlxdm text,
  sfzjh_hash text,
  yxdm text,
  zydm text,
  bjdm text,
  xznj text,
  rxnj text,
  pyccdm text,
  xslbdm text,
  sjh text,
  dzxx text,
  xjztdm text,
  sfzx text,
  sfzj text
);

\copy tmp_buaa_academic_students (xh, xm, sfzjlxdm, sfzjh_hash, yxdm, zydm, bjdm, xznj, rxnj, pyccdm, xslbdm, sjh, dzxx, xjztdm, sfzx, sfzj) FROM '/tmp/buaa_academic_students.normalized.tsv' WITH (FORMAT csv, HEADER true, DELIMITER E'\t', NULL '');

INSERT INTO academic.buaa_students (
  xh, xm, sfzjlxdm, sfzjh_hash, yxdm, zydm, bjdm, xznj, rxnj, pyccdm,
  xslbdm, sjh, dzxx, xjztdm, sfzx, sfzj, synced_at
)
SELECT
  btrim(xh),
  btrim(xm),
  NULLIF(btrim(sfzjlxdm), ''),
  NULLIF(btrim(sfzjh_hash), ''),
  NULLIF(btrim(yxdm), ''),
  NULLIF(btrim(zydm), ''),
  NULLIF(btrim(bjdm), ''),
  NULLIF(btrim(xznj), ''),
  NULLIF(btrim(rxnj), ''),
  NULLIF(btrim(pyccdm), ''),
  NULLIF(btrim(xslbdm), ''),
  NULLIF(btrim(sjh), ''),
  NULLIF(btrim(dzxx), ''),
  NULLIF(btrim(xjztdm), ''),
  NULLIF(btrim(sfzx), ''),
  NULLIF(btrim(sfzj), ''),
  now()
FROM tmp_buaa_academic_students
ON CONFLICT (xh) DO UPDATE
SET xm = EXCLUDED.xm,
    sfzjlxdm = EXCLUDED.sfzjlxdm,
    sfzjh_hash = EXCLUDED.sfzjh_hash,
    yxdm = EXCLUDED.yxdm,
    zydm = EXCLUDED.zydm,
    bjdm = EXCLUDED.bjdm,
    xznj = EXCLUDED.xznj,
    rxnj = EXCLUDED.rxnj,
    pyccdm = EXCLUDED.pyccdm,
    xslbdm = EXCLUDED.xslbdm,
    sjh = EXCLUDED.sjh,
    dzxx = EXCLUDED.dzxx,
    xjztdm = EXCLUDED.xjztdm,
    sfzx = EXCLUDED.sfzx,
    sfzj = EXCLUDED.sfzj,
    synced_at = EXCLUDED.synced_at;

ANALYZE academic.buaa_students;

SELECT format(
  'imported_buaa_academic_students=%s total_buaa_academic_students=%s',
  :'normalized_count',
  (SELECT count(*) FROM academic.buaa_students)
);
SQL
