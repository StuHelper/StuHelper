#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
EVIDENCE_SCRIPT="${REPO_ROOT}/infra/ops/postgres-backup-evidence.sh"

fail() {
  echo "[postgres-backup-evidence-contract][error] $*" >&2
  exit 1
}

cleanup() {
  [[ -n "${tmpdir:-}" ]] && rm -rf "${tmpdir}"
}
trap cleanup EXIT

assert_contains() {
  local file="$1"
  local pattern="$2"
  if ! grep -Eq -- "${pattern}" "${file}"; then
    fail "expected ${file} to contain pattern: ${pattern}"
  fi
}

[[ -f "${EVIDENCE_SCRIPT}" ]] || fail "missing evidence script: ${EVIDENCE_SCRIPT}"
[[ -x "${EVIDENCE_SCRIPT}" ]] || fail "evidence script must be executable: ${EVIDENCE_SCRIPT}"

bash -n "${EVIDENCE_SCRIPT}"

assert_contains "${EVIDENCE_SCRIPT}" 'fetch-postgres-backups\.sh'
assert_contains "${EVIDENCE_SCRIPT}" 'POSTGRES_BACKUP_EVIDENCE_FILE'
assert_contains "${EVIDENCE_SCRIPT}" 'POSTGRES_BACKUP_EVIDENCE_FETCH_COMMAND'
assert_contains "${EVIDENCE_SCRIPT}" 'stuhelper-postgres-dump-backup\.timer'
assert_contains "${EVIDENCE_SCRIPT}" 'sha256Verified'
assert_contains "${EVIDENCE_SCRIPT}" 'infra/generated/postgres-backup-evidence\.json'

tmpdir="$(mktemp -d)"
env_file="${tmpdir}/.env"
generated_env_file="${tmpdir}/.env.generated"
generated_secret_env_file="${tmpdir}/.env.generated.secrets"
generated_obs_dir="${tmpdir}/generated/observability"
logical_dir="${tmpdir}/local-logical"
evidence_file="${tmpdir}/evidence/postgres-backup-evidence.json"
fake_fetch="${tmpdir}/fake-fetch-postgres-backups"
mkdir -p "${logical_dir}"

printf 'local logical backup\n' >"${logical_dir}/predeploy-test.dump"
sha256sum "${logical_dir}/predeploy-test.dump" >"${logical_dir}/predeploy-test.dump.sha256"

cat >"${fake_fetch}" <<'FETCH'
#!/usr/bin/env bash
set -euo pipefail
[[ "${1:-}" == "logical" ]]
mkdir -p "${BACKUP_LOGICAL_DIR}"
printf 'fetched logical backup\n' >"${BACKUP_LOGICAL_DIR}/fetched-test.dump"
sha256sum "${BACKUP_LOGICAL_DIR}/fetched-test.dump" >"${BACKUP_LOGICAL_DIR}/fetched-test.dump.sha256"
FETCH
chmod +x "${fake_fetch}"

cat >"${env_file}" <<ENV
APP_ENV=production
BACKUP_LOGICAL_DIR=${logical_dir}
ENV

output="$(
  ENV_FILE="${env_file}" \
  GENERATED_ENV_FILE="${generated_env_file}" \
  GENERATED_SECRET_ENV_FILE="${generated_secret_env_file}" \
  GENERATED_OBS_DIR="${generated_obs_dir}" \
  POSTGRES_BACKUP_EVIDENCE_SKIP_TIMERS=true \
  POSTGRES_BACKUP_EVIDENCE_FETCH_COMMAND="${fake_fetch}" \
  POSTGRES_BACKUP_EVIDENCE_FILE="${evidence_file}" \
  "${EVIDENCE_SCRIPT}"
)"

[[ -f "${evidence_file}" ]] || fail "backup evidence file was not written"
printf '%s\n' "${output}" | jq -e '
  .appEnv == "production"
  and .timers.checked == false
  and .localLogicalBackup.file == "predeploy-test.dump"
  and .localLogicalBackup.sha256Verified == true
  and .fetchedLogicalBackup.file == "fetched-test.dump"
  and .fetchedLogicalBackup.sha256Verified == true
' >/dev/null
jq -e '.localLogicalBackup and .fetchedLogicalBackup' "${evidence_file}" >/dev/null

echo "[postgres-backup-evidence-contract] all assertions passed"
