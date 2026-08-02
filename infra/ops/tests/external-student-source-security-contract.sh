#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
COMMON_LIB="${REPO_ROOT}/infra/ops/lib/common.sh"

fail() {
  echo "[external-student-source-security-contract][error] $*" >&2
  exit 1
}

valid_env=(
  EXTERNAL_STUDENT_SOURCE_ENABLED=true
  EXTERNAL_STUDENT_SOURCE_PROVIDER=oracle
  EXTERNAL_STUDENT_SOURCE_SCHOOL_CODE=4111010006
  EXTERNAL_STUDENT_SOURCE_ORACLE_HOST=oracle.example.test
  EXTERNAL_STUDENT_SOURCE_ORACLE_PORT=2484
  EXTERNAL_STUDENT_SOURCE_ORACLE_SERVICE_NAME=ORCLPDB1
  EXTERNAL_STUDENT_SOURCE_ORACLE_USERNAME=stuhelper_ro
  EXTERNAL_STUDENT_SOURCE_ORACLE_READONLY_USERNAME=stuhelper_ro
  EXTERNAL_STUDENT_SOURCE_ORACLE_PASSWORD=contract-only-password
  EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_MODE=verify-full
  EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_CA_FILE=/external-student-source-tls/ca.crt
  EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_CA_HOST_PATH=/run/secrets/oracle-ca.crt
  EXTERNAL_STUDENT_SOURCE_ORACLE_SCHEMA=USR_JWBIZ
  EXTERNAL_STUDENT_SOURCE_ORACLE_TABLE=T_XS_JBXX
  EXTERNAL_STUDENT_SOURCE_ORACLE_STUDENT_ID_COLUMN=XH
  EXTERNAL_STUDENT_SOURCE_ORACLE_STUDENT_NAME_COLUMN=XM
  EXTERNAL_STUDENT_SOURCE_ORACLE_CONNECT_TIMEOUT_SECONDS=5
  EXTERNAL_STUDENT_SOURCE_ORACLE_QUERY_TIMEOUT_SECONDS=3
  EXTERNAL_STUDENT_SOURCE_ORACLE_MAX_OPEN_CONNS=4
  EXTERNAL_STUDENT_SOURCE_ORACLE_MAX_IDLE_CONNS=1
  EXTERNAL_STUDENT_SOURCE_ORACLE_CONN_MAX_LIFETIME_SECONDS=300
  EXTERNAL_STUDENT_SOURCE_ORACLE_CONN_MAX_IDLE_TIME_SECONDS=60
  EXTERNAL_STUDENT_SOURCE_ORACLE_BREAKER_FAILURE_THRESHOLD=5
  EXTERNAL_STUDENT_SOURCE_ORACLE_BREAKER_SUCCESS_THRESHOLD=2
  EXTERNAL_STUDENT_SOURCE_ORACLE_BREAKER_OPEN_SECONDS=30
)

run_gate() {
  env "$@" bash -c 'source "$1"; require_production_external_student_source_security' _ "${COMMON_LIB}"
}

run_gate "${valid_env[@]}" >/dev/null 2>&1 ||
  fail "valid verified Oracle configuration must pass"
run_gate EXTERNAL_STUDENT_SOURCE_ENABLED=false >/dev/null 2>&1 ||
  fail "disabled external source must not require Oracle settings"

if run_gate "${valid_env[@]}" EXTERNAL_STUDENT_SOURCE_ORACLE_PASSWORD=REPLACE_WITH_ORACLE_PASSWORD >/dev/null 2>&1; then
  fail "unresolved Oracle password placeholder must fail"
fi
if run_gate "${valid_env[@]}" EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_MODE=disable >/dev/null 2>&1; then
  fail "plaintext Oracle mode must fail"
fi
if run_gate "${valid_env[@]}" EXTERNAL_STUDENT_SOURCE_ORACLE_TABLE='T_XS_JBXX;DROP' >/dev/null 2>&1; then
  fail "unsafe Oracle identifier must fail"
fi
if run_gate "${valid_env[@]}" EXTERNAL_STUDENT_SOURCE_ORACLE_USERNAME=usr_jwbiz >/dev/null 2>&1; then
  fail "Oracle source schema owner must not be accepted as the runtime account"
fi
if run_gate "${valid_env[@]}" EXTERNAL_STUDENT_SOURCE_ORACLE_USERNAME=another_ro >/dev/null 2>&1; then
  fail "runtime Oracle account must match the provisioned readonly account"
fi
if run_gate "${valid_env[@]}" EXTERNAL_STUDENT_SOURCE_ORACLE_MAX_IDLE_CONNS=5 >/dev/null 2>&1; then
  fail "Oracle idle connections above max open connections must fail"
fi
if run_gate "${valid_env[@]}" EXTERNAL_STUDENT_SOURCE_ORACLE_BREAKER_FAILURE_THRESHOLD=0 >/dev/null 2>&1; then
  fail "invalid Oracle circuit breaker threshold must fail"
fi
if run_gate "${valid_env[@]}" EXTERNAL_STUDENT_SOURCE_ORACLE_BREAKER_OPEN_SECONDS=601 >/dev/null 2>&1; then
  fail "invalid Oracle circuit breaker open duration must fail"
fi

echo "[external-student-source-security-contract] all assertions passed"
