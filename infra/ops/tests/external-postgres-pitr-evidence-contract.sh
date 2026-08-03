#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
VALIDATOR="${REPO_ROOT}/infra/ops/validate-external-postgres-pitr-evidence.py"
COMMON_LIB="${REPO_ROOT}/infra/ops/lib/common.sh"
SYNC_BACKUPS="${REPO_ROOT}/infra/ops/sync-postgres-backups.sh"
BACKUP_EVIDENCE="${REPO_ROOT}/infra/ops/postgres-backup-evidence.sh"
REMOTE_PREFLIGHT="${REPO_ROOT}/infra/ops/remote-preflight.sh"
PROD_DEPLOY="${REPO_ROOT}/infra/ops/prod-deploy.sh"

fail() {
  printf '[external-postgres-pitr-evidence-contract][error] %s\n' "$*" >&2
  exit 1
}

assert_contains() {
  local file="$1"
  local pattern="$2"
  grep -Eq -- "${pattern}" "${file}" ||
    fail "expected ${file} to contain pattern: ${pattern}"
}

for file in \
  "${VALIDATOR}" \
  "${COMMON_LIB}" \
  "${SYNC_BACKUPS}" \
  "${BACKUP_EVIDENCE}" \
  "${REMOTE_PREFLIGHT}" \
  "${PROD_DEPLOY}"; do
  [[ -f "${file}" ]] || fail "missing file: ${file}"
done
python3 -m py_compile "${VALIDATOR}"

assert_contains "${COMMON_LIB}" 'require_external_postgres_pitr_evidence\(\)'
assert_contains "${COMMON_LIB}" "SELECT system_identifier::text FROM pg_control_system\\(\\)"
assert_contains "${COMMON_LIB}" '--expected-owner-uid 0'
assert_contains "${COMMON_LIB}" 'EXTERNAL_POSTGRES_PITR_EVIDENCE_FILE must be /etc/stuhelper/external-postgres-pitr-evidence\.json'
assert_contains "${SYNC_BACKUPS}" 'require_external_postgres_pitr_evidence'
assert_contains "${BACKUP_EVIDENCE}" '"externalPITR"'
assert_contains "${REMOTE_PREFLIGHT}" 'require_external_postgres_pitr_evidence'
assert_contains "${PROD_DEPLOY}" 'require_external_postgres_pitr_evidence'
assert_contains "${REPO_ROOT}/.env.prod.example" '^EXTERNAL_POSTGRES_PITR_EVIDENCE_FILE=/etc/stuhelper/external-postgres-pitr-evidence\.json$'

tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT
chmod 0700 "${tmpdir}"
owner_uid="$(id -u)"
system_identifier="1234567890123456789"
valid_evidence="${tmpdir}/valid.json"

python3 - "${valid_evidence}" <<'PY'
import json
import sys
from pathlib import Path

document = {
    "schemaVersion": 1,
    "evidenceId": "provider-check-20260803T025000Z",
    "provider": "contract-postgres-platform",
    "evidenceUri": "https://evidence.example.test/postgres/check-20260803T025000Z",
    "clusterSystemIdentifier": "1234567890123456789",
    "observedAt": "2026-08-03T02:50:00Z",
    "expiresAt": "2026-08-03T03:30:00Z",
    "continuousArchiving": {
        "enabled": True,
        "offHost": True,
        "rpoSeconds": 900,
        "retentionHours": 168,
        "lastArchivedAt": "2026-08-03T02:45:00Z",
    },
    "restoreDrill": {
        "status": "passed",
        "completedAt": "2026-07-15T12:00:00Z",
        "isolatedTarget": True,
        "baseBackupVerified": True,
        "walReplayVerified": True,
    },
}
Path(sys.argv[1]).write_text(json.dumps(document), encoding="utf-8")
PY
chmod 0644 "${valid_evidence}"

summary="$(python3 "${VALIDATOR}" \
  --evidence-file "${valid_evidence}" \
  --expected-system-identifier "${system_identifier}" \
  --expected-owner-uid "${owner_uid}" \
  --now 2026-08-03T03:00:00Z)" ||
  fail "validator rejected fresh cluster-bound external PITR evidence"
SUMMARY="${summary}" python3 - <<'PY'
import json
import os

summary = json.loads(os.environ["SUMMARY"])
assert summary["verified"] is True
assert summary["clusterSystemIdentifier"] == "1234567890123456789"
assert summary["continuousArchiving"]["offHost"] is True
assert summary["restoreDrill"]["walReplayVerified"] is True
PY

python3 - "${valid_evidence}" "${tmpdir}" <<'PY'
import json
import sys
from pathlib import Path

source = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
output = Path(sys.argv[2])

different_cluster = dict(source)
different_cluster["clusterSystemIdentifier"] = "9999999999999999999"
(output / "different-cluster.json").write_text(json.dumps(different_cluster), encoding="utf-8")

stale = dict(source)
stale["observedAt"] = "2026-08-03T01:00:00Z"
stale["expiresAt"] = "2026-08-03T03:30:00Z"
(output / "stale.json").write_text(json.dumps(stale), encoding="utf-8")

same_host = json.loads(json.dumps(source))
same_host["continuousArchiving"]["offHost"] = False
(output / "same-host.json").write_text(json.dumps(same_host), encoding="utf-8")

old_drill = json.loads(json.dumps(source))
old_drill["restoreDrill"]["completedAt"] = "2026-01-01T00:00:00Z"
(output / "old-drill.json").write_text(json.dumps(old_drill), encoding="utf-8")
PY
chmod 0644 \
  "${tmpdir}/different-cluster.json" \
  "${tmpdir}/stale.json" \
  "${tmpdir}/same-host.json" \
  "${tmpdir}/old-drill.json"

for invalid in different-cluster stale same-host old-drill; do
  if python3 "${VALIDATOR}" \
    --evidence-file "${tmpdir}/${invalid}.json" \
    --expected-system-identifier "${system_identifier}" \
    --expected-owner-uid "${owner_uid}" \
    --now 2026-08-03T03:00:00Z \
    >"${tmpdir}/${invalid}.out" 2>&1; then
    fail "validator accepted invalid external PITR evidence: ${invalid}"
  fi
done

cp "${valid_evidence}" "${tmpdir}/writable.json"
chmod 0664 "${tmpdir}/writable.json"
if python3 "${VALIDATOR}" \
  --evidence-file "${tmpdir}/writable.json" \
  --expected-system-identifier "${system_identifier}" \
  --expected-owner-uid "${owner_uid}" \
  --now 2026-08-03T03:00:00Z \
  >"${tmpdir}/writable.out" 2>&1; then
  fail "validator accepted group-writable external PITR evidence"
fi
grep -q 'must not be group/other writable' "${tmpdir}/writable.out" ||
  fail "writable evidence rejection did not report its trust boundary"

printf '[external-postgres-pitr-evidence-contract] all assertions passed\n'
