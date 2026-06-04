#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

usage() {
  cat <<'USAGE'
Usage:
  infra/ops/casdoor-grant-super-admin.sh USER[,USER...]
  CASDOOR_GRANT_SUPER_ADMIN_USERS=USER[,USER...] infra/ops/casdoor-grant-super-admin.sh
  STUHELPER_INITIAL_SUPER_ADMINS=USER[,USER...] infra/ops/casdoor-grant-super-admin.sh

Break-glass operation for production lockout recovery.

This script idempotently adds existing users in the Casdoor organization to the
Casdoor flat role "super_admin". Casdoor role membership must use the
"organization/username" member format, for example "stuhelper/alice". StuHelper
Admin reads this role from OIDC claims and expands it into admin capabilities.
Casdoor organization admin is_admin=true is not enough to access StuHelper
Admin.

Inputs:
  CASDOOR_GRANT_SUPER_ADMIN_USERS         Comma-separated Casdoor usernames.
  STUHELPER_INITIAL_SUPER_ADMINS          Fallback comma-separated bootstrap usernames.
  CASDOOR_GRANT_SUPER_ADMIN_ORGANIZATION Defaults to CASDOOR_ORGANIZATION or stuhelper.
  CASDOOR_DB_CONTAINER                   Defaults to postgres.
  CASDOOR_DB_USER                        Defaults to casdoor.
  CASDOOR_DB_NAME                        Defaults to casdoor.

The script does not read or print secrets. It expects to run on a host where the
Casdoor PostgreSQL container can execute psql as the Casdoor database user.

After granting, the affected user must sign out and sign in again so Casdoor
issues a fresh ID token containing the new role.
USAGE
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
  exit 0
fi

require_cmd docker

users="${CASDOOR_GRANT_SUPER_ADMIN_USERS:-${STUHELPER_INITIAL_SUPER_ADMINS:-${1:-}}}"
organization="${CASDOOR_GRANT_SUPER_ADMIN_ORGANIZATION:-${CASDOOR_ORGANIZATION:-stuhelper}}"
role_name="super_admin"
db_container="${CASDOOR_DB_CONTAINER:-postgres}"
db_user="${CASDOOR_DB_USER:-casdoor}"
db_name="${CASDOOR_DB_NAME:-casdoor}"

[[ -n "${users//,/}" ]] || die "CASDOOR_GRANT_SUPER_ADMIN_USERS or positional USER[,USER...] is required"
[[ "${organization}" =~ ^[A-Za-z0-9_.-]+$ ]] || die "Casdoor organization contains unsupported characters"
[[ "${role_name}" == "super_admin" ]] || die "this script only grants the super_admin role"
[[ "${db_container}" =~ ^[A-Za-z0-9_.-]+$ ]] || die "CASDOOR_DB_CONTAINER contains unsupported characters"
[[ "${db_user}" =~ ^[A-Za-z0-9_.-]+$ ]] || die "CASDOOR_DB_USER contains unsupported characters"
[[ "${db_name}" =~ ^[A-Za-z0-9_.-]+$ ]] || die "CASDOOR_DB_NAME contains unsupported characters"

run_sql() {
  docker exec -i "${db_container}" \
    psql \
      -X \
      -v ON_ERROR_STOP=1 \
      -At \
      -F $'\t' \
      -U "${db_user}" \
      -d "${db_name}" \
      -v organization="${organization}" \
      -v role_name="${role_name}" \
      -v users="${users}" \
      "$@"
}

requested_count="$(
  run_sql <<'SQL'
SELECT count(*)
FROM (
  SELECT DISTINCT trim(value) AS name
  FROM regexp_split_to_table(:'users', ',') AS value
  WHERE trim(value) <> ''
) requested;
SQL
)"
[[ "${requested_count}" -gt 0 ]] || die "no non-empty Casdoor usernames were provided"

role_count="$(
  run_sql <<'SQL'
SELECT count(*)
FROM public.role
WHERE owner = :'organization'
  AND name = :'role_name'
  AND COALESCE(is_enabled, true);
SQL
)"
[[ "${role_count}" == "1" ]] || die "enabled Casdoor role ${organization}/${role_name} was not found"

missing_users="$(
  run_sql <<'SQL'
WITH requested AS (
  SELECT DISTINCT trim(value) AS name
  FROM regexp_split_to_table(:'users', ',') AS value
  WHERE trim(value) <> ''
)
SELECT requested.name
FROM requested
WHERE NOT EXISTS (
  SELECT 1
  FROM public."user" u
  WHERE u.owner = :'organization'
    AND u.name = requested.name
    AND COALESCE(u.is_deleted, false) = false
    AND COALESCE(u.is_forbidden, false) = false
)
ORDER BY requested.name;
SQL
)"
if [[ -n "${missing_users}" ]]; then
  die "Casdoor users are missing, deleted, or forbidden in organization ${organization}: ${missing_users//$'\n'/, }"
fi

before_users="$(
  run_sql <<'SQL'
SELECT COALESCE(users, 'null')
FROM public.role
WHERE owner = :'organization'
  AND name = :'role_name';
SQL
)"

updated_rows="$(
  run_sql <<'SQL'
WITH requested AS (
  SELECT DISTINCT trim(value) AS name
  FROM regexp_split_to_table(:'users', ',') AS value
  WHERE trim(value) <> ''
),
requested_members AS (
  SELECT name, :'organization' || '/' || name AS member
  FROM requested
),
target_role AS (
  SELECT
    owner,
    name,
    CASE
      WHEN users IS NULL OR trim(users) = '' OR lower(trim(users)) = 'null' THEN '[]'::jsonb
      ELSE users::jsonb
    END AS current_users
  FROM public.role
  WHERE owner = :'organization'
    AND name = :'role_name'
  FOR UPDATE
),
merged AS (
  SELECT DISTINCT value AS name
  FROM target_role, jsonb_array_elements_text(target_role.current_users) AS value
  WHERE trim(value) <> ''
    AND value NOT IN (SELECT name FROM requested_members)
    AND value NOT IN (SELECT member FROM requested_members)
  UNION
  SELECT member FROM requested_members
),
next_users AS (
  SELECT jsonb_agg(name ORDER BY name)::text AS users
  FROM merged
)
UPDATE public.role r
SET users = COALESCE(next_users.users, '[]')
FROM next_users
WHERE r.owner = :'organization'
  AND r.name = :'role_name'
RETURNING r.owner, r.name, r.users;
SQL
)"

log "granted ${organization}/${role_name} to requested users"
log "before users: ${before_users}"
log "after role: ${updated_rows}"
log "affected users must sign out and sign in again to refresh OIDC role claims"
