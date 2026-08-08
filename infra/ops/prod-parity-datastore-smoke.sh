#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

require_cmd docker
require_cmd jq
require_cmd python3

PARITY_DIR="${PROD_PARITY_DIR:-${REPO_ROOT}/.run/prod-parity}"

parity_default_path() {
  local current="$1"
  local common_default="$2"
  local parity_default="$3"
  if repo_default_path_matches "${current}" "${common_default}"; then
    printf '%s\n' "${parity_default}"
    return
  fi
  printf '%s\n' "${current}"
}

export ENV_TEMPLATE_FILE="${REPO_ROOT}/.env.prod.example"
export ENV_FILE="$(parity_default_path "${ENV_FILE:-}" "${REPO_ROOT}/.env" "${PARITY_DIR}/.env.prod.shared")"
export SECRETS_ENV_FILE="$(parity_default_path "${SECRETS_ENV_FILE:-}" "" "${PARITY_DIR}/.env.prod.secrets.local")"
export GENERATED_ENV_FILE="$(parity_default_path "${GENERATED_ENV_FILE:-}" "${REPO_ROOT}/.env.generated" "${PARITY_DIR}/.env.prod.generated")"
export GENERATED_SECRET_ENV_FILE="$(parity_default_path "${GENERATED_SECRET_ENV_FILE:-}" "${REPO_ROOT}/.env.generated.secrets" "${PARITY_DIR}/.env.prod.generated.secrets")"
export DEPLOY_STATE_DIR="$(parity_default_path "${DEPLOY_STATE_DIR:-}" "${REPO_ROOT}/.deploy" "${PARITY_DIR}/deploy-state")"

load_env

evidence_file="${PROD_PARITY_DATASTORE_SMOKE_EVIDENCE_FILE:-${PARITY_DIR}/datastore-smoke-evidence.json}"
checks_file="$(mktemp)"
trap 'rm -f "${checks_file}"' EXIT

record_check() {
  printf '%s\n' "$1" >>"${checks_file}"
}

postgres_container="${SHARED_POSTGRES_CONTAINER:-${PROD_PARITY_POSTGRES_CONTAINER:-postgres}}"
postgres_superuser="${SHARED_POSTGRES_SUPERUSER:-${POSTGRES_USER:-postgres}}"
postgres_superdb="${SHARED_POSTGRES_DB:-postgres}"
stuhelper_db="${STUHELPER_APP_DB_NAME:-${POSTGRES_DB:-stuhelper}}"
openfga_db="${OPENFGA_DB_NAME:-openfga}"
casdoor_db="${CASDOOR_DB_NAME:-casdoor}"
app_user="${STUHELPER_APP_DB_USER:-stuhelper_app}"
backup_user="${STUHELPER_BACKUP_DB_USER:-stuhelper_backup}"
replication_user="${STUHELPER_REPLICATION_DB_USER:-stuhelper_replication}"
metrics_user="stuhelper_metrics"
openfga_user="${OPENFGA_DB_USER:-openfga}"
casdoor_user="${CASDOOR_DB_USER:-casdoor}"
redis_container="${REDIS_CONTAINER_NAME:-${STACK_NAME:-stuhelper}-redis}"
redis_user="${REDIS_USERNAME:-stuhelper_app}"
redis_metrics_user="${REDIS_EXPORTER_USERNAME:-stuhelper_metrics}"
external_datastore_network="${EXTERNAL_DATASTORE_NETWORK:-}"
compose_project="${COMPOSE_PROJECT_NAME:-${STACK_NAME:-stuhelper}}"

for key in \
  STUHELPER_APP_DB_PASSWORD \
  STUHELPER_BACKUP_DB_PASSWORD \
  STUHELPER_REPLICATION_DB_PASSWORD \
  POSTGRES_EXPORTER_DB_PASSWORD \
  OPENFGA_DB_PASSWORD \
  REDIS_PASSWORD \
  REDIS_EXPORTER_PASSWORD; do
  [[ -n "${!key:-}" ]] || die "${key} is required for prod-parity datastore smoke"
done
[[ "${redis_user}" != "${redis_metrics_user}" ]] ||
  die "REDIS_USERNAME and REDIS_EXPORTER_USERNAME must be different"
[[ "${REDIS_PASSWORD}" != "${REDIS_EXPORTER_PASSWORD}" ]] ||
  die "REDIS_PASSWORD and REDIS_EXPORTER_PASSWORD must be different"

docker inspect "${postgres_container}" >/dev/null 2>&1 || die "PostgreSQL container not found: ${postgres_container}"
docker inspect "${redis_container}" >/dev/null 2>&1 || die "Redis container not found: ${redis_container}"

postgres_health="$(docker inspect "${postgres_container}" | jq -r '.[0].State.Health.Status // "none"')"
[[ "${postgres_health}" == "healthy" ]] || die "PostgreSQL container is not healthy: ${postgres_container} (${postgres_health})"
record_check "postgres container health is healthy"

redis_health="$(docker inspect "${redis_container}" | jq -r '.[0].State.Health.Status // "none"')"
[[ "${redis_health}" == "healthy" ]] || die "Redis container is not healthy: ${redis_container} (${redis_health})"
record_check "redis container health is healthy"

metadata_failures="$(
  docker exec -i "${postgres_container}" \
    psql \
      -v ON_ERROR_STOP=1 \
      -At \
      -U "${postgres_superuser}" \
      -d "${postgres_superdb}" \
      -v stuhelper_db="${stuhelper_db}" \
      -v openfga_db="${openfga_db}" \
      -v app_user="${app_user}" \
      -v backup_user="${backup_user}" \
      -v replication_user="${replication_user}" \
      -v metrics_user="${metrics_user}" \
      -v openfga_user="${openfga_user}" <<'SQL'
WITH failures(message) AS (
  SELECT 'stuhelper app role must be login-only'
  WHERE NOT EXISTS (
    SELECT 1
      FROM pg_roles
     WHERE rolname = :'app_user'
       AND rolcanlogin
       AND NOT rolsuper
       AND NOT rolcreatedb
       AND NOT rolcreaterole
       AND NOT rolreplication
  )
  UNION ALL
  SELECT 'stuhelper backup role must be login-only'
  WHERE NOT EXISTS (
    SELECT 1
      FROM pg_roles
     WHERE rolname = :'backup_user'
       AND rolcanlogin
       AND NOT rolsuper
       AND NOT rolcreatedb
       AND NOT rolcreaterole
       AND NOT rolreplication
  )
  UNION ALL
  SELECT 'stuhelper replication role must be replication-only'
  WHERE NOT EXISTS (
    SELECT 1
      FROM pg_roles
     WHERE rolname = :'replication_user'
       AND rolcanlogin
       AND rolreplication
       AND NOT rolsuper
       AND NOT rolcreatedb
       AND NOT rolcreaterole
  )
  UNION ALL
  SELECT 'stuhelper metrics role must be a constrained login role'
  WHERE NOT EXISTS (
    SELECT 1
      FROM pg_roles
     WHERE rolname = :'metrics_user'
       AND rolcanlogin
       AND NOT rolsuper
       AND NOT rolcreatedb
       AND NOT rolcreaterole
       AND NOT rolreplication
       AND rolconnlimit = 5
  )
  UNION ALL
  SELECT 'stuhelper metrics role must inherit pg_monitor'
  WHERE NOT pg_has_role(:'metrics_user', 'pg_monitor', 'member')
  UNION ALL
  SELECT 'stuhelper metrics role has unexpected direct role memberships'
  WHERE EXISTS (
    SELECT 1
      FROM pg_auth_members membership
      JOIN pg_roles granted_role ON granted_role.oid = membership.roleid
      JOIN pg_roles member_role ON member_role.oid = membership.member
     WHERE member_role.rolname = :'metrics_user'
       AND granted_role.rolname <> 'pg_monitor'
  )
  UNION ALL
  SELECT 'openfga role must be login-only'
  WHERE NOT EXISTS (
    SELECT 1
      FROM pg_roles
     WHERE rolname = :'openfga_user'
       AND rolcanlogin
       AND NOT rolsuper
       AND NOT rolcreatedb
       AND NOT rolcreaterole
       AND NOT rolreplication
  )
  UNION ALL
  SELECT 'stuhelper database must be owned by stuhelper app role'
  WHERE NOT EXISTS (
    SELECT 1
      FROM pg_database d
      JOIN pg_roles r ON r.oid = d.datdba
     WHERE d.datname = :'stuhelper_db'
       AND r.rolname = :'app_user'
  )
  UNION ALL
  SELECT 'openfga database must be owned by openfga role'
  WHERE NOT EXISTS (
    SELECT 1
      FROM pg_database d
      JOIN pg_roles r ON r.oid = d.datdba
     WHERE d.datname = :'openfga_db'
       AND r.rolname = :'openfga_user'
  )
  UNION ALL
  SELECT 'stuhelper app role must not have CONNECT on openfga database'
  WHERE has_database_privilege(:'app_user', :'openfga_db', 'CONNECT')
  UNION ALL
  SELECT 'openfga role must not have CONNECT on stuhelper database'
  WHERE has_database_privilege(:'openfga_user', :'stuhelper_db', 'CONNECT')
  UNION ALL
  SELECT 'stuhelper backup role must not have CONNECT on openfga database'
  WHERE has_database_privilege(:'backup_user', :'openfga_db', 'CONNECT')
  UNION ALL
  SELECT 'stuhelper metrics role must not have CONNECT on stuhelper database'
  WHERE has_database_privilege(:'metrics_user', :'stuhelper_db', 'CONNECT')
  UNION ALL
  SELECT 'stuhelper metrics role must not have CONNECT on openfga database'
  WHERE has_database_privilege(:'metrics_user', :'openfga_db', 'CONNECT')
)
SELECT message FROM failures;
SQL
)"
if [[ -n "${metadata_failures}" ]]; then
  die "PostgreSQL metadata isolation failed: ${metadata_failures//$'\n'/; }"
fi
record_check "postgres role and database ownership metadata is isolated"

check_schema_owner() {
  local database="$1"
  local owner="$2"
  local failures

  failures="$(
    docker exec -i "${postgres_container}" \
      psql \
        -v ON_ERROR_STOP=1 \
        -At \
        -U "${postgres_superuser}" \
        -d "${database}" \
        -v expected_owner="${owner}" <<'SQL'
SELECT 'public schema owner mismatch'
WHERE NOT EXISTS (
  SELECT 1
    FROM pg_namespace n
    JOIN pg_roles r ON r.oid = n.nspowner
   WHERE n.nspname = 'public'
     AND r.rolname = :'expected_owner'
);
SQL
  )"
  if [[ -n "${failures}" ]]; then
    die "PostgreSQL schema isolation failed for ${database}: ${failures//$'\n'/; }"
  fi
  record_check "postgres ${database} public schema is owned by ${owner}"
}

assert_pg_connect_allowed() {
  local user="$1"
  local password="$2"
  local database="$3"

  docker exec \
    -e PGPASSWORD="${password}" \
    -i "${postgres_container}" \
    psql \
      -v ON_ERROR_STOP=1 \
      -h 127.0.0.1 \
      -U "${user}" \
      -d "${database}" \
      -At \
      -c 'SELECT current_database(), current_user' >/dev/null
  record_check "postgres ${user} can connect to ${database}"
}

assert_pg_connect_denied() {
  local user="$1"
  local password="$2"
  local database="$3"
  local output

  if output="$(
    docker exec \
      -e PGPASSWORD="${password}" \
      -i "${postgres_container}" \
      psql \
        -v ON_ERROR_STOP=1 \
        -h 127.0.0.1 \
        -U "${user}" \
        -d "${database}" \
        -At \
        -c 'SELECT 1' 2>&1 >/dev/null
  )"; then
    die "PostgreSQL role ${user} unexpectedly connected to database ${database}"
  fi

  case "${output}" in
    *"permission denied for database"*|*"does not exist"*|*"password authentication failed"*)
      ;;
    *)
      die "PostgreSQL role ${user} failed against ${database} with unexpected error: ${output}"
      ;;
  esac
  record_check "postgres ${user} cannot connect to ${database}"
}

check_schema_owner "${stuhelper_db}" "${app_user}"
check_schema_owner "${openfga_db}" "${openfga_user}"

assert_pg_connect_allowed "${app_user}" "${STUHELPER_APP_DB_PASSWORD}" "${stuhelper_db}"
assert_pg_connect_denied "${app_user}" "${STUHELPER_APP_DB_PASSWORD}" "${openfga_db}"
assert_pg_connect_allowed "${backup_user}" "${STUHELPER_BACKUP_DB_PASSWORD}" "${stuhelper_db}"
assert_pg_connect_denied "${backup_user}" "${STUHELPER_BACKUP_DB_PASSWORD}" "${openfga_db}"
assert_pg_connect_allowed "${replication_user}" "${STUHELPER_REPLICATION_DB_PASSWORD}" "${stuhelper_db}"
assert_pg_connect_allowed "${metrics_user}" "${POSTGRES_EXPORTER_DB_PASSWORD}" "postgres"
assert_pg_connect_denied "${metrics_user}" "${POSTGRES_EXPORTER_DB_PASSWORD}" "${stuhelper_db}"
assert_pg_connect_denied "${metrics_user}" "${POSTGRES_EXPORTER_DB_PASSWORD}" "${openfga_db}"
assert_pg_connect_allowed "${openfga_user}" "${OPENFGA_DB_PASSWORD}" "${openfga_db}"
assert_pg_connect_denied "${openfga_user}" "${OPENFGA_DB_PASSWORD}" "${stuhelper_db}"

casdoor_checked="false"
if [[ -n "${CASDOOR_DB_PASSWORD:-}" ]]; then
  casdoor_failures="$(
    docker exec -i "${postgres_container}" \
      psql \
        -v ON_ERROR_STOP=1 \
        -At \
        -U "${postgres_superuser}" \
        -d "${postgres_superdb}" \
        -v casdoor_db="${casdoor_db}" \
        -v casdoor_user="${casdoor_user}" \
        -v metrics_user="${metrics_user}" \
        -v stuhelper_db="${stuhelper_db}" \
        -v openfga_db="${openfga_db}" <<'SQL'
WITH failures(message) AS (
  SELECT 'casdoor role must be login-only'
  WHERE NOT EXISTS (
    SELECT 1
      FROM pg_roles
     WHERE rolname = :'casdoor_user'
       AND rolcanlogin
       AND NOT rolsuper
       AND NOT rolcreatedb
       AND NOT rolcreaterole
       AND NOT rolreplication
  )
  UNION ALL
  SELECT 'casdoor database must be owned by casdoor role'
  WHERE NOT EXISTS (
    SELECT 1
      FROM pg_database d
      JOIN pg_roles r ON r.oid = d.datdba
     WHERE d.datname = :'casdoor_db'
       AND r.rolname = :'casdoor_user'
  )
  UNION ALL
  SELECT 'casdoor role must not have CONNECT on stuhelper database'
  WHERE has_database_privilege(:'casdoor_user', :'stuhelper_db', 'CONNECT')
  UNION ALL
  SELECT 'casdoor role must not have CONNECT on openfga database'
  WHERE has_database_privilege(:'casdoor_user', :'openfga_db', 'CONNECT')
  UNION ALL
  SELECT 'stuhelper metrics role must not have CONNECT on casdoor database'
  WHERE has_database_privilege(:'metrics_user', :'casdoor_db', 'CONNECT')
)
SELECT message FROM failures;
SQL
  )"
  if [[ -n "${casdoor_failures}" ]]; then
    die "PostgreSQL Casdoor isolation failed: ${casdoor_failures//$'\n'/; }"
  fi
  check_schema_owner "${casdoor_db}" "${casdoor_user}"
  assert_pg_connect_allowed "${casdoor_user}" "${CASDOOR_DB_PASSWORD}" "${casdoor_db}"
  assert_pg_connect_denied "${casdoor_user}" "${CASDOOR_DB_PASSWORD}" "${stuhelper_db}"
  assert_pg_connect_denied "${casdoor_user}" "${CASDOOR_DB_PASSWORD}" "${openfga_db}"
  assert_pg_connect_denied "${metrics_user}" "${POSTGRES_EXPORTER_DB_PASSWORD}" "${casdoor_db}"
  casdoor_checked="true"
  record_check "postgres casdoor role and database are isolated"
fi

redis_project_label="$(docker inspect "${redis_container}" | jq -r '.[0].Config.Labels["com.docker.compose.project"] // ""')"
redis_service_label="$(docker inspect "${redis_container}" | jq -r '.[0].Config.Labels["com.docker.compose.service"] // ""')"
[[ "${redis_project_label}" == "${compose_project}" ]] || die "Redis container belongs to unexpected Compose project: ${redis_project_label}"
[[ "${redis_service_label}" == "redis" ]] || die "Redis container has unexpected Compose service label: ${redis_service_label}"
record_check "redis container belongs to StuHelper compose project"

redis_networks="$(docker inspect "${redis_container}" | jq -r '.[0].NetworkSettings.Networks | keys[]')"
if [[ -n "${external_datastore_network}" ]] && grep -Fxq "${external_datastore_network}" <<<"${redis_networks}"; then
  die "Redis container must not join external datastore network: ${external_datastore_network}"
fi
record_check "redis container is not attached to external datastore network"

docker inspect "${redis_container}" | jq -e '.[0].Mounts | any(.Destination == "/data" and .Type == "volume")' >/dev/null \
  || die "Redis container must persist data on its own Docker volume"
record_check "redis container has its own /data volume"

redis_cmd="$(docker inspect "${redis_container}" | jq -r '.[0].Config.Cmd | join(" ")')"
[[ "${redis_cmd}" == *"--port 0"* ]] || die "Redis plaintext port must be disabled"
[[ "${redis_cmd}" == *"--tls-port 6379"* ]] || die "Redis TLS port must be enabled on 6379"
[[ "${redis_cmd}" == *"--aclfile /redis-runtime/users.acl"* ]] || die "Redis ACL file must be configured"
record_check "redis command enforces TLS-only ACL access"

redis_runtime_modes="$(
  docker exec "${redis_container}" sh -eu -c \
    'stat -c "%n:%a" /redis-runtime/server.key /redis-runtime/users.acl'
)"
grep -Fxq '/redis-runtime/server.key:600' <<<"${redis_runtime_modes}" ||
  die "Redis runtime server key must use mode 600"
grep -Fxq '/redis-runtime/users.acl:600' <<<"${redis_runtime_modes}" ||
  die "Redis runtime ACL must use mode 600"
record_check "redis runtime private key and ACL use mode 600"

redis_ping="$(
  docker exec \
    -e REDISCLI_AUTH="${REDIS_PASSWORD}" \
    "${redis_container}" \
    redis-cli --no-auth-warning --tls --cacert /redis-runtime/ca.crt --user "${redis_user}" ping
)"
[[ "${redis_ping}" == "PONG" ]] || die "Redis TLS/ACL ping failed"
record_check "redis TLS ACL ping succeeds"

redis_transaction_key="stuhelper:parity:redis-transaction:$$"
redis_transaction_status=0
redis_transaction_output="$(
  {
    printf 'MULTI\n'
    printf 'SET %s 1 EX 30\n' "${redis_transaction_key}"
    printf 'EXEC\n'
  } | docker exec -i \
    -e REDISCLI_AUTH="${REDIS_PASSWORD}" \
    "${redis_container}" \
    redis-cli --no-auth-warning --raw --tls --cacert /redis-runtime/ca.crt \
      --user "${redis_user}" 2>&1
)" || redis_transaction_status=$?
docker exec \
  -e REDISCLI_AUTH="${REDIS_PASSWORD}" \
  "${redis_container}" \
  redis-cli --no-auth-warning --tls --cacert /redis-runtime/ca.crt \
    --user "${redis_user}" DEL "${redis_transaction_key}" >/dev/null 2>&1 || true
[[ "${redis_transaction_status}" -eq 0 ]] ||
  die "Redis application ACL transaction command failed: ${redis_transaction_output:-redis-cli failed without output}"
if grep -Eq 'NOPERM|ERR|[Ee]rror' <<<"${redis_transaction_output}"; then
  die "Redis application ACL transaction failed: ${redis_transaction_output}"
fi
grep -Fxq 'OK' <<<"${redis_transaction_output}" ||
  die "Redis application ACL transaction did not start"
grep -Fq 'QUEUED' <<<"${redis_transaction_output}" ||
  die "Redis application ACL transaction did not queue SET"
[[ "$(grep -Fxc 'OK' <<<"${redis_transaction_output}")" -ge 2 ]] ||
  die "Redis application ACL transaction did not execute successfully"
record_check "redis application ACL permits the required MULTI/EXEC transaction"

redis_app_admin_attempt="$(
  docker exec \
    -e REDISCLI_AUTH="${REDIS_PASSWORD}" \
    "${redis_container}" \
    redis-cli --no-auth-warning --tls --cacert /redis-runtime/ca.crt \
      --user "${redis_user}" CONFIG GET maxmemory 2>&1 || true
)"
grep -Eq 'NOPERM|not allowed' <<<"${redis_app_admin_attempt}" ||
  die "Redis application ACL must deny administrative CONFIG access"
record_check "redis application ACL denies administrative commands"

redis_metrics_info="$(
  docker exec \
    -e REDISCLI_AUTH="${REDIS_EXPORTER_PASSWORD}" \
    "${redis_container}" \
    redis-cli --no-auth-warning --tls --cacert /redis-runtime/ca.crt \
      --user "${redis_metrics_user}" INFO server
)"
grep -q '^redis_version:' <<<"${redis_metrics_info}" ||
  die "Redis exporter ACL cannot read INFO"

redis_metrics_write_attempt="$(
  docker exec \
    -e REDISCLI_AUTH="${REDIS_EXPORTER_PASSWORD}" \
    "${redis_container}" \
    redis-cli --no-auth-warning --tls --cacert /redis-runtime/ca.crt \
      --user "${redis_metrics_user}" SET stuhelper:parity:redis-metrics-denied 1 2>&1 || true
)"
grep -Eq 'NOPERM|not allowed' <<<"${redis_metrics_write_attempt}" ||
  die "Redis exporter ACL must deny writes"
record_check "redis exporter uses an independent read-only monitoring credential"

mkdir -p "$(dirname "${evidence_file}")"
POSTGRES_CONTAINER="${postgres_container}" \
POSTGRES_STUHELPER_DB="${stuhelper_db}" \
POSTGRES_OPENFGA_DB="${openfga_db}" \
POSTGRES_CASDOOR_DB="${casdoor_db}" \
POSTGRES_CASDOOR_CHECKED="${casdoor_checked}" \
REDIS_CONTAINER="${redis_container}" \
REDIS_NETWORKS="${redis_networks}" \
EXTERNAL_DATASTORE_NETWORK="${external_datastore_network}" \
COMPOSE_PROJECT_NAME="${compose_project}" \
python3 - "${evidence_file}" "${checks_file}" <<'PY'
import json
import os
import sys
from datetime import datetime, timezone
from pathlib import Path

evidence_file = Path(sys.argv[1])
checks_file = Path(sys.argv[2])
checks = [line for line in checks_file.read_text().splitlines() if line.strip()]

payload = {
    "generatedAt": datetime.now(timezone.utc).isoformat(),
    "passed": True,
    "checks": checks,
    "postgres": {
        "container": os.environ["POSTGRES_CONTAINER"],
        "databases": {
            "stuhelper": os.environ["POSTGRES_STUHELPER_DB"],
            "openfga": os.environ["POSTGRES_OPENFGA_DB"],
            "casdoor": os.environ["POSTGRES_CASDOOR_DB"],
        },
        "casdoorChecked": os.environ["POSTGRES_CASDOOR_CHECKED"] == "true",
    },
    "redis": {
        "container": os.environ["REDIS_CONTAINER"],
        "composeProject": os.environ["COMPOSE_PROJECT_NAME"],
        "networks": os.environ["REDIS_NETWORKS"].splitlines(),
        "externalDatastoreNetwork": os.environ["EXTERNAL_DATASTORE_NETWORK"],
    },
}

evidence_file.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n")
PY

log "prod-parity datastore smoke passed; evidence: ${evidence_file}"
