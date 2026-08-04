#!/usr/bin/env bash
set -euo pipefail
umask 077

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

require_cmd docker
require_cmd python3
require_cmd sha256sum
acquire_production_deploy_lock
if [[ -f "${REMOTE_DEPLOY_CONFIG_FILE}" ]]; then
  load_remote_deploy_config
fi
load_env
export BACKUP_OBJECT_STORAGE_OFF_HOST_REQUIRED=true
unset BACKUP_OBJECT_STORAGE_PINNED_HOSTS

[[ "${APP_ENV:-}" == "production" ]] ||
  die "existing PostgreSQL backup activation is only allowed in production"
[[ "${EXTERNAL_POSTGRES_ENABLED:-false}" != "true" ]] ||
  die "existing PostgreSQL backup activation currently supports only the Compose-managed internal PostgreSQL service"

current_release_file="${DEPLOY_STATE_DIR}/current-release.env"
release_log_file="${DEPLOY_STATE_DIR}/releases.log"
release_records_dir="${DEPLOY_STATE_DIR}/releases"
activation_file="${DEPLOY_STATE_DIR}/postgres-backup-activation.json"

[[ ! -e "${current_release_file}" && ! -L "${current_release_file}" ]] ||
  die "a committed application release already exists; a separate PostgreSQL backup activation is neither required nor allowed"
[[ ! -e "${release_log_file}" && ! -L "${release_log_file}" ]] ||
  die "release-log evidence survives without a current release; reconcile the application release ledger first"
require_no_surviving_release_records "${release_records_dir}"

require_backup_object_storage_config
require_off_host_backup_object_storage
require_production_postgres_archiving

stack_name="${STACK_NAME:-stuhelper}"
postgres_container="${POSTGRES_CONTAINER_NAME:-${stack_name}-postgres}"
postgres_data_volume="${stack_name}-postgres-data"
postgres_wal_volume="${POSTGRES_WAL_ARCHIVE_VOLUME_NAME:-${stack_name}-postgres-wal-archive}"

require_live_canonical_postgres_datastore
require_internal_postgres_backup_sources_match_live_datastore
require_live_postgres_wal_archiving \
  "${postgres_container}" "${POSTGRES_USER:-stuhelper}" "${POSTGRES_DB:-stuhelper}"
system_identifier="$(live_postgres_system_identifier \
  "${postgres_container}" "${POSTGRES_USER:-stuhelper}" "${POSTGRES_DB:-stuhelper}")"
require_postgres_backup_object_storage_namespace \
  "${BACKUP_OBJECT_STORAGE_PREFIX:-postgres}" "${system_identifier}"
activation_identity_args=(
  --state-dir "${DEPLOY_STATE_DIR}"
  --stack-name "${stack_name}"
  --postgres-container-name "${postgres_container}"
  --postgres-image-ref "${POSTGRES_IMAGE_REF}"
  --postgres-system-identifier "${system_identifier}"
  --backup-object-storage-prefix "${BACKUP_OBJECT_STORAGE_PREFIX:-postgres}"
  --postgres-data-volume "${postgres_data_volume}"
  --postgres-wal-archive-volume "${postgres_wal_volume}"
)
supersede_args=()

if [[ -e "${activation_file}" || -L "${activation_file}" ]]; then
  if python3 "${SCRIPT_DIR}/manage-postgres-backup-activation.py" validate \
    "${activation_identity_args[@]}" >/dev/null 2>&1; then
    require_live_postgres_backup_activation
    "${SCRIPT_DIR}/sync-postgres-backups.sh"
    log "existing PostgreSQL backup activation remains valid and an off-host synchronization completed"
    exit 0
  fi
  python3 "${SCRIPT_DIR}/manage-postgres-backup-activation.py" validate-chain \
    --state-dir "${DEPLOY_STATE_DIR}" >/dev/null ||
    die "existing PostgreSQL backup activation history is not a valid supersession chain"
  supersede_args=(--supersede)
  log "live PostgreSQL identity changed; preparing fresh recovery evidence for an audited superseding activation"
fi

activation_id="postgres-$(new_deployment_attempt_id)"
logical_dir="${BACKUP_LOGICAL_DIR:-${REPO_ROOT}/backups/postgres/logical}"
base_dir="${BACKUP_BASE_DIR:-${REPO_ROOT}/backups/postgres/base}"
logical_file="${logical_dir}/stuhelper-${activation_id}.dump"
base_file="${base_dir}/stuhelper-${activation_id}.tar.gz"
evidence_file="$(mktemp "${DEPLOY_STATE_DIR}/.postgres-backup-activation-evidence.XXXXXX.json")"
trap 'rm -f -- "${evidence_file}"' EXIT
activation_started_epoch="$(date -u +%s)"

# Never authorize scheduling from an empty or stale local directory. Create a
# fresh logical dump and physical base backup from the canonical live cluster,
# synchronize the full logical/base/WAL set, and then independently fetch the
# two new artifacts back before publishing durable activation evidence.
BACKUP_MODE=dump "${SCRIPT_DIR}/backup-postgres.sh" "${logical_file}"
BACKUP_MODE=basebackup "${SCRIPT_DIR}/backup-postgres.sh" "${base_file}"
"${SCRIPT_DIR}/sync-postgres-backups.sh"

POSTGRES_BACKUP_EVIDENCE_FILE="${evidence_file}" \
POSTGRES_BACKUP_EVIDENCE_FETCH_COMMAND="${SCRIPT_DIR}/fetch-postgres-backups.sh" \
POSTGRES_BACKUP_EVIDENCE_SKIP_TIMERS=true \
POSTGRES_BACKUP_EVIDENCE_MAX_LOGICAL_AGE_SECONDS=259200 \
POSTGRES_BACKUP_EVIDENCE_MAX_BASE_AGE_SECONDS=259200 \
  "${SCRIPT_DIR}/postgres-backup-evidence.sh" >/dev/null

read -r logical_sha256 base_sha256 < <(
  python3 - \
    "${evidence_file}" \
    "$(basename "${logical_file}")" \
    "$(basename "${base_file}")" \
    "${activation_started_epoch}" <<'PY'
import datetime as dt
import json
import re
import sys
from pathlib import Path

evidence_path = Path(sys.argv[1])
logical_name, base_name = sys.argv[2:4]
activation_started = dt.datetime.fromtimestamp(int(sys.argv[4]), dt.timezone.utc)
document = json.loads(evidence_path.read_text(encoding="utf-8"))
logical = document.get("localLogicalBackup")
fetched_logical = document.get("fetchedLogicalBackup")
base = document.get("localBaseBackup")
fetched_base = document.get("fetchedBaseBackup")
records = (logical, fetched_logical, base, fetched_base)
if not all(isinstance(record, dict) for record in records):
    raise SystemExit("backup evidence is missing required artifact records")
if logical.get("file") != logical_name or fetched_logical.get("file") != logical_name:
    raise SystemExit("backup evidence does not bind the newly created logical dump")
if base.get("file") != base_name or fetched_base.get("file") != base_name:
    raise SystemExit("backup evidence does not bind the newly created physical base backup")
logical_sha = logical.get("sha256")
base_sha = base.get("sha256")
if logical_sha != fetched_logical.get("sha256") or base_sha != fetched_base.get("sha256"):
    raise SystemExit("local and fetched backup evidence digests do not match")
if not all(
    record.get("sha256Verified") is True and record.get("fresh") is True
    for record in records
):
    raise SystemExit("backup evidence did not verify every fresh artifact")
for record in records:
    modified_at = record.get("modifiedAt")
    if not isinstance(modified_at, str):
        raise SystemExit("backup evidence is missing an artifact modification time")
    try:
        modified = dt.datetime.fromisoformat(modified_at.replace("Z", "+00:00"))
    except ValueError as exc:
        raise SystemExit("backup evidence contains an invalid artifact modification time") from exc
    if modified < activation_started - dt.timedelta(seconds=1):
        raise SystemExit("backup evidence predates the current activation attempt")
if not isinstance(logical_sha, str) or not re.fullmatch(r"[0-9a-f]{64}", logical_sha):
    raise SystemExit("logical backup evidence digest is invalid")
if not isinstance(base_sha, str) or not re.fullmatch(r"[0-9a-f]{64}", base_sha):
    raise SystemExit("physical base backup evidence digest is invalid")
print(logical_sha, base_sha)
PY
)

python3 "${SCRIPT_DIR}/manage-postgres-backup-activation.py" publish \
  "${activation_identity_args[@]}" \
  "${supersede_args[@]}" \
  --activation-id "${activation_id}" \
  --logical-backup-file "$(basename "${logical_file}")" \
  --logical-backup-sha256 "${logical_sha256}" \
  --base-backup-file "$(basename "${base_file}")" \
  --base-backup-sha256 "${base_sha256}" >/dev/null

require_live_postgres_backup_activation
log "activated the audited PostgreSQL backup control plane for the existing datastore"
