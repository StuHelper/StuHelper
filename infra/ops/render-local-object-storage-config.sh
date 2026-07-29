#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

require_cmd python3
load_env

config_dir="${LOCAL_OBJECT_STORAGE_CONFIG_DIR:-${REPO_ROOT}/infra/generated/object-storage}"
config_file="${LOCAL_OBJECT_STORAGE_CONFIG_FILE:-${config_dir}/s3.json}"

app_bucket="${OBJECT_STORAGE_BUCKET:-}"
app_access_key="${OBJECT_STORAGE_ACCESS_KEY_ID:-}"
app_secret_key="${OBJECT_STORAGE_SECRET_ACCESS_KEY:-}"
backup_bucket="${BACKUP_OBJECT_STORAGE_BUCKET:-stuhelper-postgres-backup}"
backup_access_key="${BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID:-${app_access_key}}"
backup_secret_key="${BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY:-${app_secret_key}}"

[[ -n "${app_bucket}" ]] || die "OBJECT_STORAGE_BUCKET is required"
[[ -n "${app_access_key}" ]] || die "OBJECT_STORAGE_ACCESS_KEY_ID is required"
[[ -n "${app_secret_key}" ]] || die "OBJECT_STORAGE_SECRET_ACCESS_KEY is required"
[[ -n "${backup_bucket}" ]] || die "BACKUP_OBJECT_STORAGE_BUCKET is required"
[[ -n "${backup_access_key}" ]] || die "BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID is required"
[[ -n "${backup_secret_key}" ]] || die "BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY is required"

if [[ "${app_access_key}" == "${backup_access_key}" && "${app_secret_key}" != "${backup_secret_key}" ]]; then
  die "the same object-storage access key cannot map to different secrets"
fi
if [[ "${app_access_key}" != "${backup_access_key}" && "${app_secret_key}" == "${backup_secret_key}" ]]; then
  die "application and backup object-storage identities must not share a secret"
fi

mkdir -p "${config_dir}"
umask 077

APP_BUCKET="${app_bucket}" \
APP_ACCESS_KEY="${app_access_key}" \
APP_SECRET_KEY="${app_secret_key}" \
BACKUP_BUCKET="${backup_bucket}" \
BACKUP_ACCESS_KEY="${backup_access_key}" \
BACKUP_SECRET_KEY="${backup_secret_key}" \
python3 - "${config_file}" <<'PY'
import json
import os
from pathlib import Path
import tempfile
import sys


def actions(bucket: str) -> list[str]:
    return [f"{verb}:{bucket}" for verb in ("Read", "List", "Tagging", "Write")]


target = Path(sys.argv[1])
app_bucket = os.environ["APP_BUCKET"]
app_access_key = os.environ["APP_ACCESS_KEY"]
app_secret_key = os.environ["APP_SECRET_KEY"]
backup_bucket = os.environ["BACKUP_BUCKET"]
backup_access_key = os.environ["BACKUP_ACCESS_KEY"]
backup_secret_key = os.environ["BACKUP_SECRET_KEY"]

if app_access_key == backup_access_key:
    identities = [{
        "name": "stuhelper-local",
        "credentials": [{
            "accessKey": app_access_key,
            "secretKey": app_secret_key,
        }],
        "actions": actions(app_bucket) + actions(backup_bucket),
    }]
else:
    identities = [
        {
            "name": "stuhelper-application",
            "credentials": [{
                "accessKey": app_access_key,
                "secretKey": app_secret_key,
            }],
            "actions": actions(app_bucket),
        },
        {
            "name": "stuhelper-backup",
            "credentials": [{
                "accessKey": backup_access_key,
                "secretKey": backup_secret_key,
            }],
            "actions": actions(backup_bucket),
        },
    ]

payload = {"identities": identities}
if not identities or any(
    not identity["name"]
    or len(identity["credentials"]) != 1
    or len(identity["actions"]) < 4
    for identity in identities
):
    raise SystemExit("invalid local object-storage identity configuration")

target.parent.mkdir(parents=True, exist_ok=True)
fd, temporary_name = tempfile.mkstemp(
    dir=target.parent,
    prefix=f".{target.name}.",
)
temporary = Path(temporary_name)
try:
    os.fchmod(fd, 0o600)
    with os.fdopen(fd, "w", encoding="utf-8") as handle:
        json.dump(payload, handle, ensure_ascii=False, indent=2)
        handle.write("\n")
        handle.flush()
        os.fsync(handle.fileno())
    os.replace(temporary, target)
finally:
    temporary.unlink(missing_ok=True)
PY

chmod 600 "${config_file}"

log "rendered local object-storage identities at ${config_file}"
