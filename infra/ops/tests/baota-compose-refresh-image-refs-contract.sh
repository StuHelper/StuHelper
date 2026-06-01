#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
REFRESH_SCRIPT="${REPO_ROOT}/infra/ops/baota-compose-refresh-image-refs.sh"

fail() {
  echo "[baota-compose-refresh-image-refs-contract][error] $*" >&2
  exit 1
}

assert_contains() {
  local file="$1"
  local pattern="$2"
  if ! grep -Eq -- "${pattern}" "${file}"; then
    fail "expected ${file} to contain pattern: ${pattern}"
  fi
}

assert_not_contains() {
  local file="$1"
  local pattern="$2"
  if grep -Eq -- "${pattern}" "${file}"; then
    fail "expected ${file} to not contain pattern: ${pattern}"
  fi
}

[[ -f "${REFRESH_SCRIPT}" ]] || fail "missing script: ${REFRESH_SCRIPT}"
bash -n "${REFRESH_SCRIPT}"

tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT

compose_file="${tmpdir}/docker-compose.yml"
env_file="${tmpdir}/.env.prod.shared"
backup_dir="${tmpdir}/backups"

cat >"${compose_file}" <<'YAML'
services:
  postgres:
    image: postgres:18-alpine
  migrate:
    image: stuhelper/backend:old
  app:
    image: stuhelper/backend:old
  frontend:
    image: stuhelper/frontend:old
  admin:
    image: stuhelper/admin:old
YAML

cat >"${env_file}" <<'ENV'
BACKEND_IMAGE_REF=registry.example.com/stuhelper/backend:sha-1111111
FRONTEND_IMAGE_REF=registry.example.com/stuhelper/frontend:sha-2222222
ADMIN_IMAGE_REF=registry.example.com/stuhelper/admin:sha-3333333
ENV

dry_run_output="${tmpdir}/dry-run.out"
"${REFRESH_SCRIPT}" --compose-file "${compose_file}" --env-file "${env_file}" --backup-dir "${backup_dir}" >"${dry_run_output}"
assert_contains "${dry_run_output}" 'dry-run diff'
assert_contains "${dry_run_output}" 'registry\.example\.com/stuhelper/backend:sha-1111111'
assert_contains "${compose_file}" 'stuhelper/backend:old'

"${REFRESH_SCRIPT}" --compose-file "${compose_file}" --env-file "${env_file}" --backup-dir "${backup_dir}" --apply >"${tmpdir}/apply.out"

assert_contains "${compose_file}" 'migrate:'
assert_contains "${compose_file}" 'registry\.example\.com/stuhelper/backend:sha-1111111'
assert_contains "${compose_file}" 'registry\.example\.com/stuhelper/frontend:sha-2222222'
assert_contains "${compose_file}" 'registry\.example\.com/stuhelper/admin:sha-3333333'
assert_not_contains "${compose_file}" 'stuhelper/backend:old'
assert_not_contains "${compose_file}" 'stuhelper/frontend:old'
assert_not_contains "${compose_file}" 'stuhelper/admin:old'

backup_count="$(find "${backup_dir}" -type f -name 'docker-compose.yml.before-image-refresh-*' | wc -l | tr -d ' ')"
[[ "${backup_count}" == "1" ]] || fail "expected exactly one backup, got ${backup_count}"

if "${REFRESH_SCRIPT}" --compose-file "${compose_file}" --env-file "${env_file}" --backup-dir "${backup_dir}" --apply >"${tmpdir}/second-apply.out"; then
  assert_contains "${tmpdir}/second-apply.out" 'no changes needed'
else
  fail "second apply should be idempotent"
fi

echo "[baota-compose-refresh-image-refs-contract] all assertions passed"
