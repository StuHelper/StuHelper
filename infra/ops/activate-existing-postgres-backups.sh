#!/usr/bin/env bash
set -euo pipefail
umask 077

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

require_cmd docker
require_cmd python3
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
if [[ -L "${release_records_dir}" || ( -e "${release_records_dir}" && ! -d "${release_records_dir}" ) ]]; then
  die "release record path is not a regular directory: ${release_records_dir}"
fi
shopt -s nullglob
surviving_release_records=("${release_records_dir}"/*.env)
shopt -u nullglob
((${#surviving_release_records[@]} == 0)) ||
  die "immutable application release evidence survives without a current release; reconcile the application release ledger first"

require_backup_object_storage_config
require_off_host_backup_object_storage
require_production_postgres_archiving

stack_name="${STACK_NAME:-stuhelper}"
postgres_container="${POSTGRES_CONTAINER_NAME:-${stack_name}-postgres}"
postgres_data_volume="${stack_name}-postgres-data"
postgres_wal_volume="${POSTGRES_WAL_ARCHIVE_VOLUME_NAME:-${stack_name}-postgres-wal-archive}"

require_live_canonical_postgres_datastore

if [[ -e "${activation_file}" || -L "${activation_file}" ]]; then
  require_live_postgres_backup_activation
  "${SCRIPT_DIR}/sync-postgres-backups.sh"
  log "existing PostgreSQL backup activation remains valid and an off-host synchronization completed"
  exit 0
fi

require_live_postgres_wal_archiving \
  "${postgres_container}" "${POSTGRES_USER:-stuhelper}" "${POSTGRES_DB:-stuhelper}"
system_identifier="$(live_postgres_system_identifier \
  "${postgres_container}" "${POSTGRES_USER:-stuhelper}" "${POSTGRES_DB:-stuhelper}")"

# Protect the existing datastore first. The activation pointer is published
# only after all current logical/base/WAL artifacts have reached the verified
# off-host target.
"${SCRIPT_DIR}/sync-postgres-backups.sh"

activation_id="postgres-$(new_deployment_attempt_id)"
python3 "${SCRIPT_DIR}/manage-postgres-backup-activation.py" publish \
  --state-dir "${DEPLOY_STATE_DIR}" \
  --activation-id "${activation_id}" \
  --stack-name "${stack_name}" \
  --postgres-container-name "${postgres_container}" \
  --postgres-image-ref "${POSTGRES_IMAGE_REF}" \
  --postgres-system-identifier "${system_identifier}" \
  --postgres-data-volume "${postgres_data_volume}" \
  --postgres-wal-archive-volume "${postgres_wal_volume}" >/dev/null

require_live_postgres_backup_activation
log "activated the audited PostgreSQL backup control plane for the existing datastore"
