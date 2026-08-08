#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
CONFIG="${REPO_ROOT}/infra/campus-connector/node-config.example.json"
COMPOSE="${REPO_ROOT}/infra/campus-connector/docker-compose.example.yml"

fail() {
  printf '[campus-connector-node-secret-contract][error] %s\n' "$*" >&2
  exit 1
}

for command in git jq grep; do
  command -v "${command}" >/dev/null 2>&1 || fail "missing command: ${command}"
done

ldap_password_file="$(jq -er '.operations[] | select(.type == "school_account_authenticate") | .ldap.systemBindPasswordFile' "${CONFIG}")" ||
  fail "LDAP password file reference is missing"
oracle_username_file="$(jq -er '.operations[] | select(.type == "roster_snapshot_upload") | .oracleRoster.usernameFile' "${CONFIG}")" ||
  fail "Oracle username file reference is missing"
oracle_password_file="$(jq -er '.operations[] | select(.type == "roster_snapshot_upload") | .oracleRoster.passwordFile' "${CONFIG}")" ||
  fail "Oracle password file reference is missing"

for path in "${ldap_password_file}" "${oracle_username_file}" "${oracle_password_file}"; do
  [[ "${path}" == /run/secrets/* ]] || fail "secret reference escapes /run/secrets: ${path}"
done
[[ "${oracle_username_file}" != "${oracle_password_file}" ]] || fail "Oracle username/password must use separate files"

if grep -Eiq 'systemBindPasswordEnv|usernameEnv|passwordEnv|SCHOOL_LDAP_READER_PASSWORD|SCHOOL_ORACLE_EXISTING_(USERNAME|PASSWORD)' "${CONFIG}" "${COMPOSE}"; then
  fail "retired environment-backed connector secret reference remains"
fi
grep -Eq '^      - \./secrets:/run/secrets:ro$' "${COMPOSE}" ||
  fail "connector secret directory is not mounted read-only"
git -C "${REPO_ROOT}" check-ignore -q \
  'infra/campus-connector/secrets/upstream/school-ldap-reader-password' ||
  fail "connector node-local secret directory is not ignored by Git"

printf '[campus-connector-node-secret-contract] all assertions passed\n'
