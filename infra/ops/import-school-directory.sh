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
Usage: infra/ops/import-school-directory.sh

Imports server/data/school_directory_2025.tsv into public.schools.

The imported directory is not an admission whitelist. Only rows in
school_configs with enabled=true are exposed for student verification.

Input:
  SCHOOL_DIRECTORY_DATABASE_URL or DATABASE_URL is required.
  SCHOOL_DIRECTORY_TSV defaults to server/data/school_directory_2025.tsv.

No secret values are written to the repository or printed.
USAGE
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
  exit 0
fi

require_cmd docker
require_cmd python3

load_env
export COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-${STACK_NAME:-stuhelper}}"

database_url="${SCHOOL_DIRECTORY_DATABASE_URL:-${DATABASE_URL:-}}"
[[ -n "${database_url}" ]] || die "SCHOOL_DIRECTORY_DATABASE_URL or DATABASE_URL is required"

directory_tsv="${SCHOOL_DIRECTORY_TSV:-${REPO_ROOT_GUESS}/server/data/school_directory_2025.tsv}"
[[ -f "${directory_tsv}" ]] || die "school directory TSV not found: ${directory_tsv}"

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

compose --profile prod run --rm --no-deps -T \
  -v "${directory_tsv}:/tmp/school_directory_2025.tsv:ro" \
  postgres \
  psql \
    -X \
    -v ON_ERROR_STOP=1 \
    "${database_url}" <<'SQL'
CREATE TEMP TABLE tmp_school_directory_2025 (
  code text NOT NULL,
  name text NOT NULL,
  authority text,
  location text,
  education_level text,
  remark text
);

\copy tmp_school_directory_2025 (code, name, authority, location, education_level, remark) FROM '/tmp/school_directory_2025.tsv' WITH (FORMAT csv, HEADER true, DELIMITER E'\t');

INSERT INTO public.schools (code, name, authority, location, education_level, remark, source_name, source_as_of)
SELECT
  btrim(code),
  btrim(name),
  NULLIF(btrim(authority), ''),
  NULLIF(btrim(location), ''),
  NULLIF(btrim(education_level), ''),
  NULLIF(btrim(remark), ''),
  '全国普通高等学校名单',
  DATE '2025-06-20'
FROM tmp_school_directory_2025
WHERE btrim(code) ~ '^[0-9]{10}$'
ON CONFLICT (code) DO UPDATE
SET name = EXCLUDED.name,
    authority = EXCLUDED.authority,
    location = EXCLUDED.location,
    education_level = EXCLUDED.education_level,
    remark = EXCLUDED.remark,
    source_name = EXCLUDED.source_name,
    source_as_of = EXCLUDED.source_as_of;

UPDATE public.schools
SET code = '4111010006',
    name = '北京航空航天大学',
    authority = '工业和信息化部',
    location = '北京市',
    education_level = '本科',
    remark = NULL,
    source_name = '全国普通高等学校名单',
    source_as_of = DATE '2025-06-20'
WHERE id = 10006;

SELECT pg_catalog.setval(
  'public.schools_id_seq',
  GREATEST((SELECT COALESCE(MAX(id), 1) FROM public.schools), 1),
  true
);

SELECT count(*) AS imported_school_count
FROM public.schools
WHERE source_name = '全国普通高等学校名单';
SQL
