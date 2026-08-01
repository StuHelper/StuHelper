#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: infra/ops/provision-external-student-source-oracle-readonly.sh

Creates or rotates a read-only Oracle user for the external student source.
Run this on the Oracle source host, or inside an Oracle container with sqlplus.

Required:
  EXTERNAL_STUDENT_SOURCE_ORACLE_READONLY_PASSWORD

Optional:
  EXTERNAL_STUDENT_SOURCE_ORACLE_READONLY_USERNAME default STUHELPER_ACADEMIC_RO
  EXTERNAL_STUDENT_SOURCE_ORACLE_PDB default ORCLPDB1
  EXTERNAL_STUDENT_SOURCE_ORACLE_SCHEMA default USR_JWBIZ
  EXTERNAL_STUDENT_SOURCE_ORACLE_TABLE default T_XS_JBXX
  EXTERNAL_STUDENT_SOURCE_ORACLE_STUDENT_ID_COLUMN default XH
  EXTERNAL_STUDENT_SOURCE_ORACLE_STUDENT_NAME_COLUMN default XM
  EXTERNAL_STUDENT_SOURCE_ORACLE_SQLPLUS_CONNECT default / as sysdba

The script grants only CREATE SESSION and SELECT on the configured source
table. It prints counts and grant status, but never prints the password or raw
student records.
USAGE
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
  exit 0
fi

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "[stuhelper][error] missing required command: $1" >&2
    exit 1
  }
}

require_cmd python3
require_cmd sqlplus

readonly_username="${EXTERNAL_STUDENT_SOURCE_ORACLE_READONLY_USERNAME:-STUHELPER_ACADEMIC_RO}"
readonly_password="${EXTERNAL_STUDENT_SOURCE_ORACLE_READONLY_PASSWORD:-}"
pdb="${EXTERNAL_STUDENT_SOURCE_ORACLE_PDB:-ORCLPDB1}"
schema="${EXTERNAL_STUDENT_SOURCE_ORACLE_SCHEMA:-USR_JWBIZ}"
table="${EXTERNAL_STUDENT_SOURCE_ORACLE_TABLE:-T_XS_JBXX}"
student_id_column="${EXTERNAL_STUDENT_SOURCE_ORACLE_STUDENT_ID_COLUMN:-XH}"
student_name_column="${EXTERNAL_STUDENT_SOURCE_ORACLE_STUDENT_NAME_COLUMN:-XM}"

[[ -n "${readonly_password}" ]] || {
  echo "[stuhelper][error] EXTERNAL_STUDENT_SOURCE_ORACLE_READONLY_PASSWORD is required" >&2
  exit 1
}

tmp_sql="$(mktemp)"
trap 'rm -f "${tmp_sql}"' EXIT
chmod 600 "${tmp_sql}"

python3 - \
  "${tmp_sql}" \
  "${readonly_username}" \
  "${pdb}" \
  "${schema}" \
  "${table}" \
  "${student_id_column}" \
  "${student_name_column}" <<'PY'
import os
import re
import sys
from pathlib import Path

path = Path(sys.argv[1])
readonly_username = sys.argv[2].upper()
readonly_password = os.environ["EXTERNAL_STUDENT_SOURCE_ORACLE_READONLY_PASSWORD"]
sqlplus_connect = os.environ.get(
    "EXTERNAL_STUDENT_SOURCE_ORACLE_SQLPLUS_CONNECT",
    "/ as sysdba",
)
pdb = sys.argv[3].upper()
schema = sys.argv[4].upper()
table = sys.argv[5].upper()
student_id_column = sys.argv[6].upper()
student_name_column = sys.argv[7].upper()

identifier_pattern = re.compile(r"^[A-Z][A-Z0-9_$#]{0,127}$")
for label, value in {
    "readonly username": readonly_username,
    "PDB": pdb,
    "schema": schema,
    "table": table,
    "student ID column": student_id_column,
    "student name column": student_name_column,
}.items():
    if not identifier_pattern.fullmatch(value):
        raise SystemExit(f"invalid Oracle identifier for {label}: {value!r}")

if readonly_username in {"SYS", "SYSBACKUP", "SYSDG", "SYSKM", "SYSRAC", "SYSTEM"}:
    raise SystemExit("readonly username must be a dedicated non-administrative account")
if readonly_username == schema:
    raise SystemExit("readonly username must not own the source schema")
if len(readonly_password) > 30:
    raise SystemExit("readonly password must be 30 characters or fewer for Oracle compatibility")
if len(readonly_password) < 12:
    raise SystemExit("readonly password must be at least 12 characters")
if not re.fullmatch(r"[A-Za-z0-9_!@%+=,.:-]+", readonly_password):
    raise SystemExit("readonly password contains unsupported characters")
if "\n" in sqlplus_connect or "\r" in sqlplus_connect:
    raise SystemExit("SQL*Plus connection string must be a single line")

qualified_table = f"{schema}.{table}"
password_literal = readonly_password

sql = f"""
connect {sqlplus_connect}
set heading off feedback off verify off echo off pagesize 100 linesize 200 trimspool on
whenever sqlerror exit sql.sqlcode
alter session set container={pdb};
define ro_password='{password_literal}'
declare
  n number;
begin
  select count(*) into n from all_users where username = '{readonly_username}';
  if n = 0 then
    execute immediate 'create user {readonly_username} identified by "' || '&ro_password' || '" account unlock';
  else
    execute immediate 'alter user {readonly_username} identified by "' || '&ro_password' || '" account unlock';
  end if;
end;
/
grant create session to {readonly_username};
grant select on {qualified_table} to {readonly_username};
declare
  unexpected_roles number;
  unexpected_system_privileges number;
  unexpected_object_privileges number;
  unexpected_column_privileges number;
begin
  select count(*) into unexpected_roles
    from dba_role_privs
    where grantee = '{readonly_username}';
  select count(*) into unexpected_system_privileges
    from dba_sys_privs
    where grantee = '{readonly_username}'
      and (privilege <> 'CREATE SESSION' or admin_option <> 'NO');
  select count(*) into unexpected_object_privileges
    from dba_tab_privs
    where grantee = '{readonly_username}'
      and not (
        owner = '{schema}'
        and table_name = '{table}'
        and privilege = 'SELECT'
        and grantable = 'NO'
        and hierarchy = 'NO'
      );
  select count(*) into unexpected_column_privileges
    from dba_col_privs
    where grantee = '{readonly_username}';
  if unexpected_roles + unexpected_system_privileges +
     unexpected_object_privileges + unexpected_column_privileges > 0 then
    raise_application_error(
      -20001,
      'readonly account has privileges outside the approved CREATE SESSION and target-table SELECT boundary'
    );
  end if;
end;
/
select 'READONLY_USER_EXISTS=' || count(*) from all_users where username = '{readonly_username}';
select 'READONLY_HAS_SELECT=' || count(*) from dba_tab_privs where owner = '{schema}' and table_name = '{table}' and grantee = '{readonly_username}' and privilege = 'SELECT' and grantable = 'NO' and hierarchy = 'NO';
select 'READONLY_TABLE_COUNT=' || count(*) from {qualified_table};
select 'READONLY_NONEMPTY_COLUMNS=' || count(*) from {qualified_table} where {student_id_column} is not null and {student_name_column} is not null;
exit
"""
path.write_text(sql.lstrip(), encoding="utf-8")
PY

unset EXTERNAL_STUDENT_SOURCE_ORACLE_READONLY_PASSWORD
sqlplus -s /nolog @"${tmp_sql}"

echo "[stuhelper] external student source Oracle readonly user provisioned: username=${readonly_username} schema=${schema} table=${table}"
