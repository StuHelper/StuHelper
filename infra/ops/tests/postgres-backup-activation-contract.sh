#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
MANAGER="${REPO_ROOT}/infra/ops/manage-postgres-backup-activation.py"
ACTIVATOR="${REPO_ROOT}/infra/ops/activate-existing-postgres-backups.sh"

fail() {
  printf '[postgres-backup-activation-contract][error] %s\n' "$*" >&2
  exit 1
}

tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT
state_dir="${tmpdir}/state"
install -d -m 0700 "${state_dir}"

python3 - "${ACTIVATOR}" <<'PY' || fail "existing-datastore activation ordering is unsafe"
from pathlib import Path
import sys

source = Path(sys.argv[1]).read_text(encoding="utf-8")
lock = source.index("acquire_production_deploy_lock")
environment = source.index("load_env")
release_guard = source.index("current_release_file=")
off_host = source.index("require_off_host_backup_object_storage")
wal_probe = source.index("require_live_postgres_wal_archiving")
sync = source.index('"${SCRIPT_DIR}/sync-postgres-backups.sh"', wal_probe)
publish = source.index("manage-postgres-backup-activation.py\" publish")
final_validation = source.rindex("require_live_postgres_backup_activation")
if not lock < environment < release_guard < off_host < wal_probe < sync < publish < final_validation:
    raise SystemExit("activation must serialize before config, reject release evidence, protect off-host data, then publish and revalidate")
if "record_release" in source or "current-release.env\" >" in source:
    raise SystemExit("backup activation must not publish or rewrite an application release")
PY

identity_args=(
  --state-dir "${state_dir}"
  --stack-name contract-prod
  --postgres-container-name contract-prod-postgres
  --postgres-image-ref registry.example.test/stuhelper/postgres@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
  --postgres-system-identifier 1234567890123456789
  --postgres-data-volume contract-prod-postgres-data
  --postgres-wal-archive-volume contract-prod-postgres-wal-archive
)

python3 "${MANAGER}" publish \
  "${identity_args[@]}" \
  --activation-id postgres-20260804T000000Z-contract >"${tmpdir}/publish.json"
python3 "${MANAGER}" validate "${identity_args[@]}" >"${tmpdir}/validate.json"

current="${state_dir}/postgres-backup-activation.json"
immutable="${state_dir}/postgres-backup-activations/postgres-20260804T000000Z-contract.json"
[[ -f "${current}" && -f "${immutable}" ]] || fail "activation records were not published"
[[ "$(stat -c '%a' "${current}")" == "600" ]] || fail "current activation must use mode 0600"
[[ "$(stat -c '%a' "${immutable}")" == "600" ]] || fail "immutable activation must use mode 0600"
[[ "$(stat -c '%a' "${state_dir}/postgres-backup-activations")" == "700" ]] ||
  fail "activation record directory must use mode 0700"
cmp -s "${current}" "${immutable}" || fail "current activation does not match its immutable record"
grep -q '"validated": true' "${tmpdir}/validate.json" || fail "activation validation did not report success"

python3 "${MANAGER}" publish \
  "${identity_args[@]}" \
  --activation-id postgres-20260804T000100Z-idempotent >"${tmpdir}/idempotent.json"
[[ "$(find "${state_dir}/postgres-backup-activations" -maxdepth 1 -type f | wc -l)" == "1" ]] ||
  fail "idempotent activation created a second immutable record"

printf '{}\n' >"${state_dir}/postgres-backup-activations/unexpected.json"
chmod 0600 "${state_dir}/postgres-backup-activations/unexpected.json"
if python3 "${MANAGER}" validate "${identity_args[@]}" \
  >"${tmpdir}/unexpected-record.out" 2>"${tmpdir}/unexpected-record.err"; then
  fail "activation validation accepted an unexpected immutable record"
fi
grep -q 'unexpected or orphaned records' "${tmpdir}/unexpected-record.err" ||
  fail "unexpected activation record did not produce the expected diagnostic"
rm -f "${state_dir}/postgres-backup-activations/unexpected.json"

if python3 "${MANAGER}" validate \
  "${identity_args[@]/1234567890123456789/2234567890123456789}" \
  >"${tmpdir}/mismatch.out" 2>"${tmpdir}/mismatch.err"; then
  fail "activation validation accepted a different PostgreSQL system identifier"
fi
grep -q 'does not match the live datastore' "${tmpdir}/mismatch.err" ||
  fail "system-identifier mismatch did not produce the expected diagnostic"

cp "${current}" "${tmpdir}/canonical.json"
printf '\n' >>"${current}"
if python3 "${MANAGER}" validate "${identity_args[@]}" \
  >"${tmpdir}/noncanonical.out" 2>"${tmpdir}/noncanonical.err"; then
  fail "activation validation accepted a non-canonical current pointer"
fi
grep -q 'is not canonical' "${tmpdir}/noncanonical.err" ||
  fail "non-canonical activation did not produce the expected diagnostic"
install -m 0600 "${tmpdir}/canonical.json" "${current}"

rm -f "${current}"
if python3 "${MANAGER}" publish \
  "${identity_args[@]}" \
  --activation-id postgres-20260804T000200Z-orphaned \
  >"${tmpdir}/orphaned.out" 2>"${tmpdir}/orphaned.err"; then
  fail "activation publication accepted orphaned immutable evidence"
fi
grep -q 'evidence survives without a current pointer' "${tmpdir}/orphaned.err" ||
  fail "orphaned activation evidence did not produce the expected diagnostic"

rm -rf "${state_dir}/postgres-backup-activations"
ln -s "${tmpdir}/canonical.json" "${current}"
if python3 "${MANAGER}" validate "${identity_args[@]}" \
  >"${tmpdir}/symlink.out" 2>"${tmpdir}/symlink.err"; then
  fail "activation validation followed a symlink current pointer"
fi
grep -Eq 'cannot safely open|regular file' "${tmpdir}/symlink.err" ||
  fail "symlink activation did not produce the expected diagnostic"

printf '[postgres-backup-activation-contract] all assertions passed\n'
