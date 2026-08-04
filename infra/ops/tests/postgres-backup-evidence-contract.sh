#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
EVIDENCE_SCRIPT="${REPO_ROOT}/infra/ops/postgres-backup-evidence.sh"
FETCH_SCRIPT="${REPO_ROOT}/infra/ops/fetch-postgres-backups.sh"

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

assert_not_contains() {
  local file="$1"
  local pattern="$2"
  if grep -Eq -- "${pattern}" "${file}"; then
    fail "expected ${file} to not contain pattern: ${pattern}"
  fi
}

[[ -f "${EVIDENCE_SCRIPT}" ]] || fail "missing evidence script: ${EVIDENCE_SCRIPT}"
[[ -x "${EVIDENCE_SCRIPT}" ]] || fail "evidence script must be executable: ${EVIDENCE_SCRIPT}"
[[ -f "${FETCH_SCRIPT}" ]] || fail "missing fetch script: ${FETCH_SCRIPT}"
[[ -x "${FETCH_SCRIPT}" ]] || fail "fetch script must be executable: ${FETCH_SCRIPT}"

bash -n "${EVIDENCE_SCRIPT}"
bash -n "${FETCH_SCRIPT}"

assert_contains "${EVIDENCE_SCRIPT}" 'fetch-postgres-backups\.sh'
assert_contains "${EVIDENCE_SCRIPT}" 'POSTGRES_BACKUP_EVIDENCE_FILE'
assert_contains "${EVIDENCE_SCRIPT}" 'POSTGRES_BACKUP_EVIDENCE_FETCH_COMMAND'
assert_contains "${EVIDENCE_SCRIPT}" 'POSTGRES_BACKUP_EVIDENCE_MAX_BASE_AGE_SECONDS'
assert_contains "${EVIDENCE_SCRIPT}" 'stuhelper-postgres-dump-backup\.timer'
assert_contains "${EVIDENCE_SCRIPT}" 'sha256Verified'
assert_contains "${EVIDENCE_SCRIPT}" 'localBaseBackup'
assert_contains "${EVIDENCE_SCRIPT}" 'fetchedBaseBackup'
assert_contains "${EVIDENCE_SCRIPT}" 'infra/generated/postgres-backup-evidence\.json'
assert_contains "${EVIDENCE_SCRIPT}" 'systemd_unit_is_loaded "\$\{unit\}"'
assert_not_contains "${EVIDENCE_SCRIPT}" 'systemctl list-unit-files \| grep -q'

python3 - "${FETCH_SCRIPT}" <<'PY' || fail "fetch command does not preserve isolated evidence directories"
from pathlib import Path
import sys

source = Path(sys.argv[1]).read_text(encoding="utf-8")
load = source.index("load_env_preserving")
logical = source.index("BACKUP_LOGICAL_DIR", load)
base = source.index("BACKUP_BASE_DIR", load)
unset = source.index("unset BACKUP_OBJECT_STORAGE_PINNED_HOSTS", load)
if not load < logical < unset or not load < base < unset:
    raise SystemExit("fetch must preserve per-invocation logical and base directories across environment reloads")
PY

tmpdir="$(mktemp -d)"
env_file="${tmpdir}/.env"
generated_env_file="${tmpdir}/.env.generated"
generated_secret_env_file="${tmpdir}/.env.generated.secrets"
generated_obs_dir="${tmpdir}/generated/observability"
logical_dir="${tmpdir}/local-logical"
base_dir="${tmpdir}/local-base"
evidence_file="${tmpdir}/evidence/postgres-backup-evidence.json"
fake_fetch="${tmpdir}/fake-fetch-postgres-backups"
mkdir -p "${logical_dir}" "${base_dir}"

printf 'local logical backup\n' >"${logical_dir}/predeploy-test.dump"
sha256sum "${logical_dir}/predeploy-test.dump" >"${logical_dir}/predeploy-test.dump.sha256"

local_base_fixture="${tmpdir}/local-base-fixture"
mkdir -p "${local_base_fixture}/pg_wal"
printf '18\n' >"${local_base_fixture}/PG_VERSION"
printf '{"PostgreSQL-Backup-Manifest-Version": 2}\n' \
  >"${local_base_fixture}/backup_manifest"
tar -C "${local_base_fixture}" -czf "${base_dir}/predeploy-test.tar.gz" .
sha256sum "${base_dir}/predeploy-test.tar.gz" \
  >"${base_dir}/predeploy-test.tar.gz.sha256"

cat >"${fake_fetch}" <<'FETCH'
#!/usr/bin/env bash
set -euo pipefail

case "${1:-}" in
  logical)
    mkdir -p "${BACKUP_LOGICAL_DIR}"
    printf 'fetched logical backup\n' >"${BACKUP_LOGICAL_DIR}/fetched-test.dump"
    sha256sum "${BACKUP_LOGICAL_DIR}/fetched-test.dump" \
      >"${BACKUP_LOGICAL_DIR}/fetched-test.dump.sha256"
    ;;
  base)
    fixture="${BACKUP_BASE_DIR}/.fixture"
    mkdir -p "${fixture}/pg_wal"
    printf '18\n' >"${fixture}/PG_VERSION"
    printf '{"PostgreSQL-Backup-Manifest-Version": 2}\n' \
      >"${fixture}/backup_manifest"
    tar -C "${fixture}" -czf "${BACKUP_BASE_DIR}/fetched-test.tar.gz" .
    rm -rf "${fixture}"
    sha256sum "${BACKUP_BASE_DIR}/fetched-test.tar.gz" \
      >"${BACKUP_BASE_DIR}/fetched-test.tar.gz.sha256"
    ;;
  *)
    exit 31
    ;;
esac
FETCH
chmod +x "${fake_fetch}"

cat >"${env_file}" <<ENV
APP_ENV=production
BACKUP_LOGICAL_DIR=${logical_dir}
BACKUP_BASE_DIR=${base_dir}
POSTGRES_BACKUP_EVIDENCE_FILE=${tmpdir}/must-not-win.json
POSTGRES_BACKUP_EVIDENCE_FETCH_COMMAND=${tmpdir}/must-not-run
POSTGRES_BACKUP_EVIDENCE_TIMER_REQUIRED=true
POSTGRES_BACKUP_EVIDENCE_SKIP_TIMERS=false
POSTGRES_BACKUP_EVIDENCE_MAX_LOGICAL_AGE_SECONDS=1
POSTGRES_BACKUP_EVIDENCE_MAX_BASE_AGE_SECONDS=999999999
ENV

output="$(
  ENV_FILE="${env_file}" \
  GENERATED_ENV_FILE="${generated_env_file}" \
  GENERATED_SECRET_ENV_FILE="${generated_secret_env_file}" \
  GENERATED_OBS_DIR="${generated_obs_dir}" \
  POSTGRES_BACKUP_EVIDENCE_SKIP_TIMERS=true \
  POSTGRES_BACKUP_EVIDENCE_MAX_LOGICAL_AGE_SECONDS=129600 \
  POSTGRES_BACKUP_EVIDENCE_MAX_BASE_AGE_SECONDS=691200 \
  POSTGRES_BACKUP_EVIDENCE_FETCH_COMMAND="${fake_fetch}" \
  POSTGRES_BACKUP_EVIDENCE_FILE="${evidence_file}" \
  "${EVIDENCE_SCRIPT}"
)"

[[ -f "${evidence_file}" ]] || fail "backup evidence file was not written"
[[ ! -e "${tmpdir}/must-not-win.json" ]] ||
  fail "environment reload replaced the per-invocation evidence output path"
OUTPUT_JSON="${output}" python3 - "${evidence_file}" <<'PY'
import json
import os
import sys
from pathlib import Path

stdout_bundle = json.loads(os.environ["OUTPUT_JSON"])
file_bundle = json.loads(Path(sys.argv[1]).read_text())
assert stdout_bundle == file_bundle

assert file_bundle["appEnv"] == "production"
assert file_bundle["timers"]["checked"] is False
assert file_bundle["freshnessPolicySeconds"] == {
    "logical": 129600,
    "physicalBase": 691200,
}
assert file_bundle["externalPITR"] is None

expected = {
    "localLogicalBackup": ("predeploy-test.dump", False),
    "fetchedLogicalBackup": ("fetched-test.dump", False),
    "localBaseBackup": ("predeploy-test.tar.gz", True),
    "fetchedBaseBackup": ("fetched-test.tar.gz", True),
}
for key, (filename, physical) in expected.items():
    item = file_bundle[key]
    assert item["file"] == filename
    assert item["sha256Verified"] is True
    assert item["fresh"] is True
    if physical:
        assert item["archiveReadable"] is True
PY

touch -d '10 days ago' "${base_dir}/predeploy-test.tar.gz"
if ENV_FILE="${env_file}" \
  GENERATED_ENV_FILE="${generated_env_file}" \
  GENERATED_SECRET_ENV_FILE="${generated_secret_env_file}" \
  GENERATED_OBS_DIR="${generated_obs_dir}" \
  POSTGRES_BACKUP_EVIDENCE_SKIP_TIMERS=true \
  POSTGRES_BACKUP_EVIDENCE_MAX_LOGICAL_AGE_SECONDS=129600 \
  POSTGRES_BACKUP_EVIDENCE_MAX_BASE_AGE_SECONDS=60 \
  POSTGRES_BACKUP_EVIDENCE_FETCH_COMMAND="${fake_fetch}" \
  POSTGRES_BACKUP_EVIDENCE_FILE="${tmpdir}/evidence/stale.json" \
  "${EVIDENCE_SCRIPT}" >/dev/null 2>&1; then
  fail "stale physical backup must fail evidence generation"
fi

echo "[postgres-backup-evidence-contract] all assertions passed"
