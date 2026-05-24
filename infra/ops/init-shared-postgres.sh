#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

require_cmd docker
require_cmd python3

load_env

postgres_container="${SHARED_POSTGRES_CONTAINER:-${PROD_PARITY_POSTGRES_CONTAINER:-postgres}}"
superuser="${SHARED_POSTGRES_SUPERUSER:-${POSTGRES_USER:-postgres}}"
superdb="${SHARED_POSTGRES_DB:-postgres}"

stuhelper_db="${STUHELPER_APP_DB_NAME:-${POSTGRES_DB:-stuhelper}}"
openfga_db="${OPENFGA_DB_NAME:-openfga}"
app_user="${STUHELPER_APP_DB_USER:-stuhelper_app}"
backup_user="${STUHELPER_BACKUP_DB_USER:-stuhelper_backup}"
replication_user="${STUHELPER_REPLICATION_DB_USER:-stuhelper_replication}"
openfga_user="${OPENFGA_DB_USER:-openfga}"

required=(
  STUHELPER_APP_DB_PASSWORD
  STUHELPER_BACKUP_DB_PASSWORD
  STUHELPER_REPLICATION_DB_PASSWORD
  OPENFGA_DB_PASSWORD
)
for key in "${required[@]}"; do
  [[ -n "${!key:-}" ]] || die "${key} is required to initialize shared PostgreSQL"
done

docker inspect "${postgres_container}" >/dev/null 2>&1 || die "PostgreSQL container not found: ${postgres_container}"

log "initializing shared PostgreSQL databases and roles in container ${postgres_container}"
docker exec -i "${postgres_container}" \
  psql \
    -v ON_ERROR_STOP=1 \
    -U "${superuser}" \
    -d "${superdb}" \
    -v stuhelper_db="${stuhelper_db}" \
    -v openfga_db="${openfga_db}" \
    -v app_user="${app_user}" \
    -v backup_user="${backup_user}" \
    -v replication_user="${replication_user}" \
    -v openfga_user="${openfga_user}" \
    -v app_password="${STUHELPER_APP_DB_PASSWORD}" \
    -v backup_password="${STUHELPER_BACKUP_DB_PASSWORD}" \
    -v replication_password="${STUHELPER_REPLICATION_DB_PASSWORD}" \
    -v openfga_password="${OPENFGA_DB_PASSWORD}" <<'SQL'
\set QUIET on

SELECT format(
  'CREATE ROLE %I LOGIN PASSWORD %L NOSUPERUSER NOCREATEDB NOCREATEROLE CONNECTION LIMIT 30',
  :'app_user',
  :'app_password'
)
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = :'app_user') \gexec
SELECT format(
  'ALTER ROLE %I WITH LOGIN PASSWORD %L NOSUPERUSER NOCREATEDB NOCREATEROLE CONNECTION LIMIT 30',
  :'app_user',
  :'app_password'
) \gexec

SELECT format(
  'CREATE ROLE %I LOGIN PASSWORD %L NOSUPERUSER NOCREATEDB NOCREATEROLE CONNECTION LIMIT 5',
  :'backup_user',
  :'backup_password'
)
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = :'backup_user') \gexec
SELECT format(
  'ALTER ROLE %I WITH LOGIN PASSWORD %L NOSUPERUSER NOCREATEDB NOCREATEROLE CONNECTION LIMIT 5',
  :'backup_user',
  :'backup_password'
) \gexec
SELECT format('GRANT pg_read_all_data, pg_read_all_settings, pg_read_all_stats TO %I', :'backup_user') \gexec

SELECT format(
  'CREATE ROLE %I LOGIN REPLICATION PASSWORD %L NOSUPERUSER NOCREATEDB NOCREATEROLE CONNECTION LIMIT 5',
  :'replication_user',
  :'replication_password'
)
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = :'replication_user') \gexec
SELECT format(
  'ALTER ROLE %I WITH LOGIN REPLICATION PASSWORD %L NOSUPERUSER NOCREATEDB NOCREATEROLE CONNECTION LIMIT 5',
  :'replication_user',
  :'replication_password'
) \gexec

SELECT format(
  'CREATE ROLE %I LOGIN PASSWORD %L NOSUPERUSER NOCREATEDB NOCREATEROLE CONNECTION LIMIT 20',
  :'openfga_user',
  :'openfga_password'
)
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = :'openfga_user') \gexec
SELECT format(
  'ALTER ROLE %I WITH LOGIN PASSWORD %L NOSUPERUSER NOCREATEDB NOCREATEROLE CONNECTION LIMIT 20',
  :'openfga_user',
  :'openfga_password'
) \gexec

SELECT format('CREATE DATABASE %I OWNER %I', :'stuhelper_db', :'app_user')
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = :'stuhelper_db') \gexec
SELECT format('ALTER DATABASE %I OWNER TO %I', :'stuhelper_db', :'app_user') \gexec
SELECT format('REVOKE ALL ON DATABASE %I FROM PUBLIC', :'stuhelper_db') \gexec
SELECT format('GRANT CONNECT, TEMPORARY ON DATABASE %I TO %I', :'stuhelper_db', :'app_user') \gexec
SELECT format('GRANT CONNECT ON DATABASE %I TO %I', :'stuhelper_db', :'backup_user') \gexec
SELECT format('GRANT CONNECT ON DATABASE %I TO %I', :'stuhelper_db', :'replication_user') \gexec

SELECT format('CREATE DATABASE %I OWNER %I', :'openfga_db', :'openfga_user')
WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = :'openfga_db') \gexec
SELECT format('ALTER DATABASE %I OWNER TO %I', :'openfga_db', :'openfga_user') \gexec
SELECT format('REVOKE ALL ON DATABASE %I FROM PUBLIC', :'openfga_db') \gexec
SELECT format('GRANT CONNECT, TEMPORARY ON DATABASE %I TO %I', :'openfga_db', :'openfga_user') \gexec

\connect :stuhelper_db
SELECT format('ALTER SCHEMA public OWNER TO %I', :'app_user') \gexec
REVOKE CREATE ON SCHEMA public FROM PUBLIC;
SELECT format('GRANT USAGE, CREATE ON SCHEMA public TO %I', :'app_user') \gexec
SELECT format('GRANT USAGE ON SCHEMA public TO %I', :'backup_user') \gexec
SELECT format('GRANT SELECT ON ALL TABLES IN SCHEMA public TO %I', :'backup_user') \gexec
SELECT format('GRANT SELECT ON ALL SEQUENCES IN SCHEMA public TO %I', :'backup_user') \gexec
SELECT format('ALTER DEFAULT PRIVILEGES FOR ROLE %I IN SCHEMA public GRANT SELECT ON TABLES TO %I', :'app_user', :'backup_user') \gexec
SELECT format('ALTER DEFAULT PRIVILEGES FOR ROLE %I IN SCHEMA public GRANT SELECT ON SEQUENCES TO %I', :'app_user', :'backup_user') \gexec

\connect :openfga_db
SELECT format('ALTER SCHEMA public OWNER TO %I', :'openfga_user') \gexec
REVOKE CREATE ON SCHEMA public FROM PUBLIC;
SELECT format('GRANT USAGE, CREATE ON SCHEMA public TO %I', :'openfga_user') \gexec
SQL

log "shared PostgreSQL is ready: database=${stuhelper_db}, app_role=${app_user}, openfga_database=${openfga_db}"
