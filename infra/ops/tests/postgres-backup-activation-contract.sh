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
release_records_guard = source.index("require_no_surviving_release_records", release_guard)
off_host = source.index("require_off_host_backup_object_storage")
canonical = source.index("require_live_canonical_postgres_datastore")
backup_sources = source.index("require_internal_postgres_backup_sources_match_live_datastore", canonical)
wal_probe = source.index("require_live_postgres_wal_archiving")
namespace = source.index("require_postgres_backup_object_storage_namespace", wal_probe)
chain_validation = source.index("validate-chain", wal_probe)
logical = source.index("BACKUP_MODE=dump", wal_probe)
base = source.index("BACKUP_MODE=basebackup", logical)
sync = source.index('"${SCRIPT_DIR}/sync-postgres-backups.sh"', base)
fetch_pin = source.index('POSTGRES_BACKUP_EVIDENCE_FETCH_COMMAND="${SCRIPT_DIR}/fetch-postgres-backups.sh"', sync)
evidence = source.index("postgres-backup-evidence.sh", sync)
publish = source.index("manage-postgres-backup-activation.py\" publish")
final_validation = source.rindex("require_live_postgres_backup_activation")
if not lock < environment < release_guard < release_records_guard < off_host < canonical < backup_sources < wal_probe < namespace < chain_validation < logical < base < sync < fetch_pin < evidence < publish < final_validation:
    raise SystemExit("activation must serialize before config, reject release evidence, bind the cluster namespace, then publish and revalidate")
if "--supersede" not in source:
    raise SystemExit("activation must support an audited superseding record after live identity changes")
if "POSTGRES_BACKUP_EVIDENCE_MAX_LOGICAL_AGE_SECONDS=259200" not in source or "POSTGRES_BACKUP_EVIDENCE_MAX_BASE_AGE_SECONDS=259200" not in source:
    raise SystemExit("activation evidence window must cover the documented backup and transfer capacity envelope")
if "activation_started_epoch" not in source or "backup evidence predates the current activation attempt" not in source:
    raise SystemExit("activation must bind evidence timestamps to the current attempt")
if source.count('POSTGRES_BACKUP_EVIDENCE_FETCH_COMMAND="${SCRIPT_DIR}/fetch-postgres-backups.sh"') != 1:
    raise SystemExit("activation must pin its off-host recovery proof to the production fetcher")
if "POSTGRES_BACKUP_EVIDENCE_MAX_LOGICAL_AGE_SECONDS=3600" in source or "POSTGRES_BACKUP_EVIDENCE_MAX_BASE_AGE_SECONDS=3600" in source:
    raise SystemExit("activation must not use a one-hour age window for a multi-stage backup workflow")
if "record_release" in source or "current-release.env\" >" in source:
    raise SystemExit("backup activation must not publish or rewrite an application release")
PY

identity_args=(
  --state-dir "${state_dir}"
  --stack-name contract-prod
  --postgres-container-name contract-prod-postgres
  --postgres-image-ref registry.example.test/stuhelper/postgres@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
  --postgres-system-identifier 1234567890123456789
  --backup-object-storage-prefix postgres/1234567890123456789
  --postgres-data-volume contract-prod-postgres-data
  --postgres-wal-archive-volume contract-prod-postgres-wal-archive
)
evidence_args=(
  --logical-backup-file stuhelper-postgres-contract.dump
  --logical-backup-sha256 bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
  --base-backup-file stuhelper-postgres-contract.tar.gz
  --base-backup-sha256 cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc
)
upgraded_identity_args=(
  --state-dir "${state_dir}"
  --stack-name contract-prod
  --postgres-container-name contract-prod-postgres
  --postgres-image-ref registry.example.test/stuhelper/postgres@sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd
  --postgres-system-identifier 1234567890123456789
  --backup-object-storage-prefix postgres/1234567890123456789
  --postgres-data-volume contract-prod-postgres-data
  --postgres-wal-archive-volume contract-prod-postgres-wal-archive
)
upgraded_evidence_args=(
  --logical-backup-file stuhelper-postgres-upgraded.dump
  --logical-backup-sha256 eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee
  --base-backup-file stuhelper-postgres-upgraded.tar.gz
  --base-backup-sha256 ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff
)
changed_cluster_identity_args=(
  --state-dir "${state_dir}"
  --stack-name contract-prod
  --postgres-container-name contract-prod-postgres
  --postgres-image-ref registry.example.test/stuhelper/postgres@sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd
  --postgres-system-identifier 9876543210987654321
  --backup-object-storage-prefix postgres/1234567890123456789
  --postgres-data-volume contract-prod-postgres-data
  --postgres-wal-archive-volume contract-prod-postgres-wal-archive
)

python3 "${MANAGER}" publish \
  "${identity_args[@]}" \
  "${evidence_args[@]}" \
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
python3 - "${current}" <<'PY' || fail "activation did not bind the reviewed recovery artifacts"
import json
import sys
from pathlib import Path

document = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
assert document["schemaVersion"] == 3
assert document["backupObjectStoragePrefix"] == "postgres/1234567890123456789"
assert document["previousActivation"] is None
assert document["recoveryEvidence"] == {
    "logicalBackup": {
        "file": "stuhelper-postgres-contract.dump",
        "sha256": "b" * 64,
    },
    "physicalBaseBackup": {
        "file": "stuhelper-postgres-contract.tar.gz",
        "sha256": "c" * 64,
    },
}
PY

python3 "${MANAGER}" publish \
  "${identity_args[@]}" \
  "${evidence_args[@]}" \
  --activation-id postgres-20260804T000100Z-idempotent >"${tmpdir}/idempotent.json"
[[ "$(find "${state_dir}/postgres-backup-activations" -maxdepth 1 -type f | wc -l)" == "1" ]] ||
  fail "idempotent activation created a second immutable record"

if python3 "${MANAGER}" publish \
  "${upgraded_identity_args[@]}" \
  "${upgraded_evidence_args[@]}" \
  --activation-id postgres-20260804T000200Z-upgraded \
  >"${tmpdir}/upgrade-without-supersede.out" 2>"${tmpdir}/upgrade-without-supersede.err"; then
  fail "activation publication replaced a changed datastore identity without explicit supersession"
fi
grep -q 'requires --supersede' "${tmpdir}/upgrade-without-supersede.err" ||
  fail "changed datastore identity did not require explicit supersession"

if python3 "${MANAGER}" publish \
  "${changed_cluster_identity_args[@]}" \
  "${upgraded_evidence_args[@]}" \
  --supersede \
  --activation-id postgres-20260804T000200Z-mixed-wal \
  >"${tmpdir}/mixed-wal.out" 2>"${tmpdir}/mixed-wal.err"; then
  fail "activation publication accepted a predecessor cluster's WAL namespace"
fi
grep -q 'must end in the live system identifier' "${tmpdir}/mixed-wal.err" ||
  fail "mixed-cluster WAL namespace rejection did not report the identity boundary"

python3 "${MANAGER}" publish \
  "${upgraded_identity_args[@]}" \
  "${upgraded_evidence_args[@]}" \
  --supersede \
  --activation-id postgres-20260804T000200Z-upgraded >"${tmpdir}/supersede.json"
python3 "${MANAGER}" validate "${upgraded_identity_args[@]}" >"${tmpdir}/validate-upgraded.json"
python3 "${MANAGER}" validate-chain --state-dir "${state_dir}" >"${tmpdir}/validate-chain.json"
upgraded_immutable="${state_dir}/postgres-backup-activations/postgres-20260804T000200Z-upgraded.json"
[[ -f "${upgraded_immutable}" ]] || fail "superseding immutable activation was not published"
cmp -s "${current}" "${upgraded_immutable}" ||
  fail "current activation does not point at the superseding immutable record"
[[ "$(find "${state_dir}/postgres-backup-activations" -maxdepth 1 -type f | wc -l)" == "2" ]] ||
  fail "superseding activation did not retain exactly one predecessor"
python3 - "${current}" "${immutable}" <<'PY' || fail "superseding activation chain is not digest-linked"
import hashlib
import json
import sys
from pathlib import Path

current = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
previous_payload = Path(sys.argv[2]).read_bytes()
assert current["previousActivation"] == {
    "activationId": "postgres-20260804T000000Z-contract",
    "sha256": hashlib.sha256(previous_payload).hexdigest(),
}
assert current["recoveryEvidence"]["logicalBackup"]["file"] == "stuhelper-postgres-upgraded.dump"
assert current["recoveryEvidence"]["physicalBaseBackup"]["file"] == "stuhelper-postgres-upgraded.tar.gz"
PY

if python3 "${MANAGER}" validate "${identity_args[@]}" \
  >"${tmpdir}/stale-identity.out" 2>"${tmpdir}/stale-identity.err"; then
  fail "activation validation accepted the predecessor as the live identity"
fi
grep -q 'does not match the live datastore' "${tmpdir}/stale-identity.err" ||
  fail "stale activation identity did not produce the expected diagnostic"

printf '{}\n' >"${state_dir}/postgres-backup-activations/unexpected.json"
chmod 0600 "${state_dir}/postgres-backup-activations/unexpected.json"
if python3 "${MANAGER}" validate "${identity_args[@]}" \
  >"${tmpdir}/unexpected-record.out" 2>"${tmpdir}/unexpected-record.err"; then
  fail "activation validation accepted an unexpected immutable record"
fi
grep -Eq 'unexpected fields|unexpected or orphaned records' "${tmpdir}/unexpected-record.err" ||
  fail "unexpected activation record did not produce the expected diagnostic"
rm -f "${state_dir}/postgres-backup-activations/unexpected.json"

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
  "${evidence_args[@]}" \
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

legacy_state_dir="${tmpdir}/legacy-state"
legacy_records_dir="${legacy_state_dir}/postgres-backup-activations"
install -d -m 0700 "${legacy_state_dir}" "${legacy_records_dir}"
legacy_id="postgres-20260803T230000Z-legacy"
legacy_current="${legacy_state_dir}/postgres-backup-activation.json"
legacy_immutable="${legacy_records_dir}/${legacy_id}.json"
python3 - "${legacy_current}" "${legacy_immutable}" <<'PY'
import json
import os
import sys
from pathlib import Path

document = {
    "schemaVersion": 1,
    "event": "existing_postgres_backup_control_activated",
    "activationId": "postgres-20260803T230000Z-legacy",
    "activatedAt": "2026-08-03T23:00:00Z",
    "stackName": "contract-prod",
    "postgresContainerName": "contract-prod-postgres",
    "postgresImageRef": "registry.example.test/stuhelper/postgres@sha256:" + "a" * 64,
    "postgresSystemIdentifier": "1234567890123456789",
    "postgresDataVolume": "contract-prod-postgres-data",
    "postgresWalArchiveVolume": "contract-prod-postgres-wal-archive",
}
payload = (json.dumps(document, sort_keys=True, separators=(",", ":")) + "\n").encode()
for name in sys.argv[1:]:
    path = Path(name)
    path.write_bytes(payload)
    os.chmod(path, 0o600)
PY

legacy_identity_args=("${identity_args[@]}")
legacy_identity_args[1]="${legacy_state_dir}"
if python3 "${MANAGER}" validate "${legacy_identity_args[@]}" \
  >"${tmpdir}/legacy-validate.out" 2>"${tmpdir}/legacy-validate.err"; then
  fail "legacy activation without recovery evidence authorized scheduled backups"
fi
grep -q 'pre-v3 PostgreSQL backup activation requires' "${tmpdir}/legacy-validate.err" ||
  fail "legacy activation did not require fresh recovery evidence"
python3 "${MANAGER}" validate-chain --state-dir "${legacy_state_dir}" \
  >"${tmpdir}/legacy-chain.json"
if python3 "${MANAGER}" publish \
  "${legacy_identity_args[@]}" \
  "${evidence_args[@]}" \
  --activation-id postgres-20260804T000300Z-migrated \
  >"${tmpdir}/legacy-without-supersede.out" 2>"${tmpdir}/legacy-without-supersede.err"; then
  fail "legacy activation was replaced without explicit supersession"
fi
grep -q 'requires fresh recovery evidence and --supersede' "${tmpdir}/legacy-without-supersede.err" ||
  fail "legacy activation did not require explicit supersession"

python3 "${MANAGER}" publish \
  "${legacy_identity_args[@]}" \
  "${evidence_args[@]}" \
  --supersede \
  --activation-id postgres-20260804T000300Z-migrated \
  >"${tmpdir}/legacy-migrated.json"
python3 "${MANAGER}" validate "${legacy_identity_args[@]}" \
  >"${tmpdir}/legacy-migrated-validate.json"
python3 "${MANAGER}" validate-chain --state-dir "${legacy_state_dir}" \
  >"${tmpdir}/legacy-migrated-chain.json"
python3 - "${legacy_current}" "${legacy_immutable}" <<'PY' || fail "legacy activation was not retained as a digest-linked root"
import hashlib
import json
import sys
from pathlib import Path

current = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
legacy_payload = Path(sys.argv[2]).read_bytes()
assert current["schemaVersion"] == 3
assert current["previousActivation"] == {
    "activationId": "postgres-20260803T230000Z-legacy",
    "sha256": hashlib.sha256(legacy_payload).hexdigest(),
}
PY

v2_state_dir="${tmpdir}/v2-state"
v2_records_dir="${v2_state_dir}/postgres-backup-activations"
install -d -m 0700 "${v2_state_dir}" "${v2_records_dir}"
v2_id="postgres-20260803T233000Z-v2"
v2_current="${v2_state_dir}/postgres-backup-activation.json"
v2_immutable="${v2_records_dir}/${v2_id}.json"
python3 - "${v2_current}" "${v2_immutable}" <<'PY'
import json
import os
import sys
from pathlib import Path

document = {
    "schemaVersion": 2,
    "event": "existing_postgres_backup_control_activated",
    "activationId": "postgres-20260803T233000Z-v2",
    "activatedAt": "2026-08-03T23:30:00Z",
    "stackName": "contract-prod",
    "postgresContainerName": "contract-prod-postgres",
    "postgresImageRef": "registry.example.test/stuhelper/postgres@sha256:" + "a" * 64,
    "postgresSystemIdentifier": "1234567890123456789",
    "postgresDataVolume": "contract-prod-postgres-data",
    "postgresWalArchiveVolume": "contract-prod-postgres-wal-archive",
    "previousActivation": None,
    "recoveryEvidence": {
        "logicalBackup": {
            "file": "stuhelper-postgres-v2.dump",
            "sha256": "b" * 64,
        },
        "physicalBaseBackup": {
            "file": "stuhelper-postgres-v2.tar.gz",
            "sha256": "c" * 64,
        },
    },
}
payload = (json.dumps(document, sort_keys=True, separators=(",", ":")) + "\n").encode()
for name in sys.argv[1:]:
    path = Path(name)
    path.write_bytes(payload)
    os.chmod(path, 0o600)
PY

v2_identity_args=("${identity_args[@]}")
v2_identity_args[1]="${v2_state_dir}"
python3 "${MANAGER}" validate-chain --state-dir "${v2_state_dir}" \
  >"${tmpdir}/v2-chain.json"
if python3 "${MANAGER}" validate "${v2_identity_args[@]}" \
  >"${tmpdir}/v2-validate.out" 2>"${tmpdir}/v2-validate.err"; then
  fail "schema-v2 activation without a cluster namespace authorized scheduled backups"
fi
grep -q 'pre-v3 PostgreSQL backup activation requires' "${tmpdir}/v2-validate.err" ||
  fail "schema-v2 activation did not require a namespaced supersession"
if python3 "${MANAGER}" publish \
  "${v2_identity_args[@]}" \
  "${evidence_args[@]}" \
  --activation-id postgres-20260804T000400Z-v3 \
  >"${tmpdir}/v2-without-supersede.out" 2>"${tmpdir}/v2-without-supersede.err"; then
  fail "schema-v2 activation was replaced without explicit supersession"
fi
grep -q 'schema-v2 PostgreSQL backup activation requires' "${tmpdir}/v2-without-supersede.err" ||
  fail "schema-v2 activation did not require explicit supersession"
python3 "${MANAGER}" publish \
  "${v2_identity_args[@]}" \
  "${evidence_args[@]}" \
  --supersede \
  --activation-id postgres-20260804T000400Z-v3 >"${tmpdir}/v2-migrated.json"
python3 "${MANAGER}" validate "${v2_identity_args[@]}" \
  >"${tmpdir}/v2-migrated-validate.json"
python3 "${MANAGER}" validate-chain --state-dir "${v2_state_dir}" \
  >"${tmpdir}/v2-migrated-chain.json"
python3 - "${v2_current}" "${v2_immutable}" <<'PY' || fail "schema-v2 activation was not retained as a digest-linked predecessor"
import hashlib
import json
import sys
from pathlib import Path

current = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
v2_payload = Path(sys.argv[2]).read_bytes()
assert current["schemaVersion"] == 3
assert current["backupObjectStoragePrefix"] == "postgres/1234567890123456789"
assert current["previousActivation"] == {
    "activationId": "postgres-20260803T233000Z-v2",
    "sha256": hashlib.sha256(v2_payload).hexdigest(),
}
PY

printf '[postgres-backup-activation-contract] all assertions passed\n'
