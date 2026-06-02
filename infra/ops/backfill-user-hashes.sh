#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

usage() {
  cat <<'USAGE'
Usage: infra/ops/backfill-user-hashes.sh [--dry-run|--apply]

Runs the user_hash backfill inside the already configured backend container.
The command never prints DATABASE_URL or HMAC_SECRET. Dry-run is the default
and only reports the number of users still missing user_hash.

Environment:
  USER_HASH_BACKFILL_APP_CONTAINER defaults to stuhelper-prod-app.
USAGE
}

apply=false
case "${1:-}" in
  ""|--dry-run)
    ;;
  --apply)
    apply=true
    ;;
  --help|-h)
    usage
    exit 0
    ;;
  *)
    usage >&2
    die "unknown argument: ${1}"
    ;;
esac

require_cmd docker

app_container="${USER_HASH_BACKFILL_APP_CONTAINER:-stuhelper-prod-app}"
if ! docker inspect "${app_container}" >/dev/null 2>&1; then
  die "backend container not found: ${app_container}"
fi

args=(/app/backfill-user-hashes --dry-run)
if [[ "${apply}" == "true" ]]; then
  args=(/app/backfill-user-hashes --apply)
fi

log "running user_hash backfill via ${app_container} (apply=${apply})"
docker exec "${app_container}" "${args[@]}"
