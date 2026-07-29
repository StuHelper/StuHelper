#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

require_cmd docker

load_env

[[ "${EXTERNAL_POSTGRES_ENABLED:-false}" != "true" ]] ||
  die "external PostgreSQL roles must be provisioned by its administrator; refusing to mutate an external datastore"
: "${POSTGRES_EXPORTER_DB_PASSWORD:?POSTGRES_EXPORTER_DB_PASSWORD is required}"

postgres_container="${POSTGRES_CONTAINER_NAME:-${STACK_NAME:-stuhelper}-postgres}"
postgres_superuser="${POSTGRES_USER:-stuhelper}"
max_attempts="${POSTGRES_ROLE_BOOTSTRAP_RETRIES:-30}"
sleep_seconds="${POSTGRES_ROLE_BOOTSTRAP_SLEEP_SECONDS:-2}"

for ((attempt = 1; attempt <= max_attempts; attempt++)); do
  if docker inspect "${postgres_container}" >/dev/null 2>&1 &&
     docker exec "${postgres_container}" \
       pg_isready -U "${postgres_superuser}" -d postgres >/dev/null 2>&1; then
    break
  fi
  if (( attempt == max_attempts )); then
    die "PostgreSQL container did not become ready: ${postgres_container}"
  fi
  sleep "${sleep_seconds}"
done

log "ensuring least-privilege PostgreSQL monitoring role in ${postgres_container}"
docker exec \
  -i \
  -e POSTGRES_EXPORTER_DB_PASSWORD="${POSTGRES_EXPORTER_DB_PASSWORD}" \
  "${postgres_container}" \
  psql \
    -v ON_ERROR_STOP=1 \
    -U "${postgres_superuser}" \
    -d postgres <<'SQL'
\set QUIET on
\getenv postgres_exporter_password POSTGRES_EXPORTER_DB_PASSWORD

SELECT format(
  'CREATE ROLE stuhelper_metrics LOGIN PASSWORD %L NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION INHERIT CONNECTION LIMIT 5',
  :'postgres_exporter_password'
)
WHERE NOT EXISTS (
  SELECT 1
    FROM pg_roles
   WHERE rolname = 'stuhelper_metrics'
) \gexec

ALTER ROLE stuhelper_metrics
  WITH LOGIN
       PASSWORD :'postgres_exporter_password'
       NOSUPERUSER
       NOCREATEDB
       NOCREATEROLE
       NOREPLICATION
       INHERIT
       CONNECTION LIMIT 5;
ALTER ROLE stuhelper_metrics RESET ALL;

SELECT format('REVOKE %I FROM stuhelper_metrics', granted_role.rolname)
  FROM pg_auth_members membership
  JOIN pg_roles granted_role ON granted_role.oid = membership.roleid
  JOIN pg_roles member_role ON member_role.oid = membership.member
 WHERE member_role.rolname = 'stuhelper_metrics'
   AND granted_role.rolname <> 'pg_monitor' \gexec

GRANT pg_monitor TO stuhelper_metrics;
GRANT CONNECT ON DATABASE postgres TO stuhelper_metrics;

SELECT pg_reload_conf();

DO $$
BEGIN
  IF EXISTS (
    SELECT 1
      FROM pg_hba_file_rules
     WHERE error IS NOT NULL
  ) THEN
    RAISE EXCEPTION 'loaded pg_hba.conf contains invalid rules';
  END IF;

  IF NOT EXISTS (
    SELECT 1
      FROM pg_roles
     WHERE rolname = 'stuhelper_metrics'
       AND rolcanlogin
       AND NOT rolsuper
       AND NOT rolcreatedb
       AND NOT rolcreaterole
       AND NOT rolreplication
       AND rolconnlimit = 5
  ) THEN
    RAISE EXCEPTION 'stuhelper_metrics role attributes do not match the least-privilege contract';
  END IF;

  IF NOT pg_has_role('stuhelper_metrics', 'pg_monitor', 'member') THEN
    RAISE EXCEPTION 'stuhelper_metrics is missing pg_monitor membership';
  END IF;
END
$$;
SQL

log "PostgreSQL monitoring role is ready"
