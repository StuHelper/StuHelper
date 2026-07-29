#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
PREPARE="${REPO_ROOT}/infra/ops/prepare-datastore-client-cas.sh"

fail() {
  printf '[datastore-client-ca-contract][error] %s\n' "$*" >&2
  exit 1
}

tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT

pg_source="${tmpdir}/postgres-source"
redis_source="${tmpdir}/redis-source"
oracle_source="${tmpdir}/oracle-source"
pg_client="${tmpdir}/postgres-client"
redis_client="${tmpdir}/redis-client"
oracle_client="${tmpdir}/oracle-client"
mkdir -p "${pg_source}" "${redis_source}" "${oracle_source}"

openssl req -x509 -newkey rsa:2048 -nodes \
  -keyout "${pg_source}/ca.key" \
  -out "${pg_source}/ca.crt" \
  -days 1 \
  -subj "/CN=postgres-client-ca-contract" >/dev/null 2>&1
openssl req -x509 -newkey rsa:2048 -nodes \
  -keyout "${redis_source}/ca.key" \
  -out "${redis_source}/ca.crt" \
  -days 1 \
  -subj "/CN=redis-client-ca-contract" >/dev/null 2>&1
openssl req -x509 -newkey rsa:2048 -nodes \
  -keyout "${oracle_source}/ca.key" \
  -out "${oracle_source}/ca.crt" \
  -days 1 \
  -subj "/CN=oracle-client-ca-contract" >/dev/null 2>&1
printf 'server-private-key-must-not-be-copied\n' >"${pg_source}/server.key"
printf 'server-private-key-must-not-be-copied\n' >"${redis_source}/server.key"
printf 'user app on >password\n' >"${redis_source}/users.acl"

env_file="${tmpdir}/internal.env"
cat >"${env_file}" <<EOF
POSTGRES_INTERNAL_SSL_MODE=verify-full
EXTERNAL_POSTGRES_ENABLED=false
POSTGRES_TLS_DIR=${pg_source}
POSTGRES_CLIENT_CA_DIR=${pg_client}
REDIS_TLS_ENABLED=true
REDIS_TLS_DIR=${redis_source}
REDIS_CLIENT_CA_DIR=${redis_client}
EXTERNAL_STUDENT_SOURCE_ENABLED=false
EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_CA_DIR=${oracle_client}
EOF

ENV_FILE="${env_file}" \
GENERATED_ENV_FILE="${tmpdir}/missing.generated" \
GENERATED_SECRET_ENV_FILE="${tmpdir}/missing.generated.secrets" \
"${PREPARE}" >/dev/null

cmp -s "${pg_source}/ca.crt" "${pg_client}/ca.crt" ||
  fail "PostgreSQL client CA copy differs from source"
cmp -s "${redis_source}/ca.crt" "${redis_client}/ca.crt" ||
  fail "Redis client CA copy differs from source"
[[ "$(find "${pg_client}" -mindepth 1 -maxdepth 1 -printf '%f\n')" == "ca.crt" ]] ||
  fail "PostgreSQL client directory must contain ca.crt only"
[[ "$(find "${redis_client}" -mindepth 1 -maxdepth 1 -printf '%f\n')" == "ca.crt" ]] ||
  fail "Redis client directory must contain ca.crt only"
[[ -d "${oracle_client}" && ! -e "${oracle_client}/ca.crt" ]] ||
  fail "disabled Oracle student source must prepare an empty client CA mount"
[[ "$(stat -c '%a' "${pg_client}/ca.crt")" == "644" ]] ||
  fail "PostgreSQL client CA must use mode 644"
[[ "$(stat -c '%a' "${redis_client}/ca.crt")" == "644" ]] ||
  fail "Redis client CA must use mode 644"

printf 'unexpected\n' >"${pg_client}/server.key"
if ENV_FILE="${env_file}" \
  GENERATED_ENV_FILE="${tmpdir}/missing.generated" \
  GENERATED_SECRET_ENV_FILE="${tmpdir}/missing.generated.secrets" \
  "${PREPARE}" >"${tmpdir}/unexpected.out" 2>"${tmpdir}/unexpected.err"; then
  fail "preparer accepted an unexpected client-directory entry"
fi
grep -Fq 'client CA directory contains an unexpected entry' "${tmpdir}/unexpected.err" ||
  fail "unexpected-entry rejection did not explain the failure"
rm -f "${pg_client}/server.key"

external_missing_env="${tmpdir}/external-missing.env"
cat >"${external_missing_env}" <<EOF
POSTGRES_INTERNAL_SSL_MODE=verify-full
EXTERNAL_POSTGRES_ENABLED=true
POSTGRES_CLIENT_CA_DIR=${pg_client}
REDIS_TLS_ENABLED=false
REDIS_CLIENT_CA_DIR=${redis_client}
EOF
if ENV_FILE="${external_missing_env}" \
  GENERATED_ENV_FILE="${tmpdir}/missing.generated" \
  GENERATED_SECRET_ENV_FILE="${tmpdir}/missing.generated.secrets" \
  "${PREPARE}" >"${tmpdir}/external-missing.out" 2>"${tmpdir}/external-missing.err"; then
  fail "external PostgreSQL TLS was accepted without a host CA path"
fi
grep -Fq 'POSTGRES_CLIENT_CA_HOST_PATH is required' "${tmpdir}/external-missing.err" ||
  fail "missing external PostgreSQL CA rejection did not explain the failure"

external_env="${tmpdir}/external.env"
cat >"${external_env}" <<EOF
POSTGRES_INTERNAL_SSL_MODE=verify-full
EXTERNAL_POSTGRES_ENABLED=true
POSTGRES_CLIENT_CA_HOST_PATH=${pg_source}/ca.crt
POSTGRES_CLIENT_CA_DIR=${pg_client}
REDIS_TLS_ENABLED=false
REDIS_CLIENT_CA_DIR=${redis_client}
EOF
ENV_FILE="${external_env}" \
GENERATED_ENV_FILE="${tmpdir}/missing.generated" \
GENERATED_SECRET_ENV_FILE="${tmpdir}/missing.generated.secrets" \
"${PREPARE}" >/dev/null
cmp -s "${pg_source}/ca.crt" "${pg_client}/ca.crt" ||
  fail "external PostgreSQL CA was not copied"
[[ ! -e "${redis_client}/ca.crt" ]] ||
  fail "disabled Redis TLS must prepare an empty client CA mount"

oracle_missing_env="${tmpdir}/oracle-missing.env"
cat >"${oracle_missing_env}" <<EOF
APP_ENV=production
POSTGRES_INTERNAL_SSL_MODE=disable
POSTGRES_CLIENT_CA_DIR=${pg_client}
REDIS_TLS_ENABLED=false
REDIS_CLIENT_CA_DIR=${redis_client}
EXTERNAL_STUDENT_SOURCE_ENABLED=true
EXTERNAL_STUDENT_SOURCE_PROVIDER=oracle
EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_MODE=verify-full
EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_CA_DIR=${oracle_client}
EOF
if ENV_FILE="${oracle_missing_env}" \
  GENERATED_ENV_FILE="${tmpdir}/missing.generated" \
  GENERATED_SECRET_ENV_FILE="${tmpdir}/missing.generated.secrets" \
  "${PREPARE}" >"${tmpdir}/oracle-missing.out" 2>"${tmpdir}/oracle-missing.err"; then
  fail "verified Oracle TLS was accepted without a host CA path"
fi
grep -Fq 'EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_CA_HOST_PATH is required' "${tmpdir}/oracle-missing.err" ||
  fail "missing Oracle CA rejection did not explain the failure"

oracle_env="${tmpdir}/oracle.env"
cat >"${oracle_env}" <<EOF
APP_ENV=production
POSTGRES_INTERNAL_SSL_MODE=disable
POSTGRES_CLIENT_CA_DIR=${pg_client}
REDIS_TLS_ENABLED=false
REDIS_CLIENT_CA_DIR=${redis_client}
EXTERNAL_STUDENT_SOURCE_ENABLED=true
EXTERNAL_STUDENT_SOURCE_PROVIDER=oracle
EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_MODE=verify-full
EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_CA_HOST_PATH=${oracle_source}/ca.crt
EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_CA_DIR=${oracle_client}
EOF
ENV_FILE="${oracle_env}" \
GENERATED_ENV_FILE="${tmpdir}/missing.generated" \
GENERATED_SECRET_ENV_FILE="${tmpdir}/missing.generated.secrets" \
"${PREPARE}" >/dev/null
cmp -s "${oracle_source}/ca.crt" "${oracle_client}/ca.crt" ||
  fail "Oracle student source CA was not copied"
[[ "$(find "${oracle_client}" -mindepth 1 -maxdepth 1 -printf '%f\n')" == "ca.crt" ]] ||
  fail "Oracle student source client directory must contain ca.crt only"

printf '[datastore-client-ca-contract] all assertions passed\n'
