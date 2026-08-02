#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
CUTOVER_SCRIPT="${REPO_ROOT}/infra/ops/authorization-ledger-cutover.sh"
CUTOVER_COMMAND="${REPO_ROOT}/server/cmd/authorization-cutover/main.go"
UP_MIGRATION="${REPO_ROOT}/server/migrations/000024_authorization_authority_cutover.up.sql"
DOWN_MIGRATION="${REPO_ROOT}/server/migrations/000024_authorization_authority_cutover.down.sql"

fail() {
  echo "[authorization-ledger-cutover-contract][error] $*" >&2
  exit 1
}

assert_contains() {
  local file="$1"
  local pattern="$2"
  grep -Eq -- "${pattern}" "${file}" || fail "expected ${file} to contain: ${pattern}"
}

for file in "${CUTOVER_SCRIPT}" "${CUTOVER_COMMAND}" "${UP_MIGRATION}" "${DOWN_MIGRATION}"; do
  [[ -f "${file}" ]] || fail "missing required cutover artifact: ${file}"
done

bash -n "${CUTOVER_SCRIPT}"
[[ -x "${CUTOVER_SCRIPT}" ]] || fail "cutover script must be executable"

assert_contains "${CUTOVER_SCRIPT}" '^load_env$'
assert_contains "${CUTOVER_SCRIPT}" '^source_casdoor_bootstrap_env$'
assert_contains "${CUTOVER_SCRIPT}" 'source_casdoor_bootstrap_env_file "\$\{file\}"'
if grep -qF 'source "${file}"' "${CUTOVER_SCRIPT}"; then
  fail "authorization cutover must not raw-source the Casdoor bootstrap credential file"
fi
assert_contains "${CUTOVER_SCRIPT}" 'go run ./cmd/authorization-cutover'
assert_contains "${CUTOVER_SCRIPT}" 'OPENFGA_CUTOVER_API_URL'
assert_contains "${CUTOVER_SCRIPT}" 'AUTHORIZATION_CUTOVER_DATABASE_URL'
assert_contains "${CUTOVER_SCRIPT}" 'AUTHORIZATION_CUTOVER_GO_IMAGE'
assert_contains "${CUTOVER_SCRIPT}" 'env_args\+=\(-e "\$\{key\}"\)'
assert_contains "${CUTOVER_SCRIPT}" 'cgr\.dev/chainguard/go:latest@sha256:[0-9a-f]{64}'
assert_contains "${CUTOVER_SCRIPT}" '^    --read-only \\'
assert_contains "${CUTOVER_SCRIPT}" '^    --cap-drop ALL \\'
assert_contains "${CUTOVER_SCRIPT}" '^    --security-opt no-new-privileges \\'
assert_contains "${CUTOVER_SCRIPT}" 'test\("\^\[0-9a-f\]\{64\}\$"\)'

if grep -Eq -- '-e "CASDOOR_BOOTSTRAP_CLIENT_SECRET=' "${CUTOVER_SCRIPT}"; then
  fail "Docker invocation must inherit secrets by variable name instead of embedding values"
fi

assert_contains "${CUTOVER_COMMAND}" 'AuthorityCutoverStatus\(ctx\)'
assert_contains "${CUTOVER_COMMAND}" 'NewAuthoritySnapshotClient'
assert_contains "${CUTOVER_COMMAND}" 'ImportLegacyAuthority'
assert_contains "${UP_MIGRATION}" "status IN \('pending', 'completed'\)"
assert_contains "${UP_MIGRATION}" 'source_digest ~ '\''\^\[0-9a-f\]\{64\}\$'\'''
assert_contains "${DOWN_MIGRATION}" 'DROP TABLE IF EXISTS public.authorization_authority_cutover'

echo "[authorization-ledger-cutover-contract] all assertions passed"
