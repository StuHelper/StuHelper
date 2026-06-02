#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
RECLAIM_SCRIPT="${REPO_ROOT}/infra/ops/reclaim-production-disk-space.sh"
RELEASE_RUNBOOK="${REPO_ROOT}/docs/guides/release-runbook.md"

fail() {
  echo "[reclaim-production-disk-space-contract][error] $*" >&2
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

for file in "${RECLAIM_SCRIPT}" "${RELEASE_RUNBOOK}"; do
  [[ -f "${file}" ]] || fail "missing file: ${file}"
done

bash -n "${RECLAIM_SCRIPT}"

assert_contains "${RECLAIM_SCRIPT}" 'dry-run'
assert_contains "${RECLAIM_SCRIPT}" '--apply'
assert_contains "${RECLAIM_SCRIPT}" 'journalctl "--vacuum-size=\$\{journal_vacuum_size\}"'
assert_contains "${RECLAIM_SCRIPT}" 'apt-get clean'
assert_contains "${RECLAIM_SCRIPT}" 'docker builder prune -af'
assert_contains "${RECLAIM_SCRIPT}" 'docker image prune -f'
assert_contains "${RECLAIM_SCRIPT}" 'Never performed by default:'
assert_contains "${RECLAIM_SCRIPT}" 'docker volume prune'
assert_contains "${RECLAIM_SCRIPT}" 'PostgreSQL PGDATA or wal-archive'
assert_contains "${RECLAIM_SCRIPT}" 'ALLOW_POSTGRES_WAL_ARCHIVE_PRUNE'
assert_contains "${RECLAIM_SCRIPT}" 'PRUNE_POSTGRES_WAL_ARCHIVE_KEEP_HOURS'
assert_contains "${RECLAIM_SCRIPT}" 'stuhelper-wal-archive-prune-'
assert_contains "${RECLAIM_SCRIPT}" 'resolve_postgres_wal_archive_dir'
assert_contains "${RECLAIM_SCRIPT}" 'docker volume ls --format'
assert_contains "${RECLAIM_SCRIPT}" 'set POSTGRES_WAL_ARCHIVE_VOLUME_NAME explicitly'
assert_contains "${RECLAIM_SCRIPT}" 'PRUNE_STUHELPER_TMP_ARTIFACTS'
assert_contains "${RECLAIM_SCRIPT}" '/tmp/stuhelper-release'
assert_contains "${RECLAIM_SCRIPT}" 'stuhelper-\*\.tar\.sha256'
assert_contains "${RECLAIM_SCRIPT}" 'deploy-\*\.sh'
assert_contains "${RECLAIM_SCRIPT}" 'PRUNE_BAOTA_PANEL_BACKUPS_KEEP'
assert_contains "${RECLAIM_SCRIPT}" 'df -h / /var/lib/docker /www'
assert_contains "${RECLAIM_SCRIPT}" '\[\[ "\$\{prune_stuhelper_tmp_artifacts\}" == "true" \]\] \|\| return 0'
assert_contains "${RECLAIM_SCRIPT}" '\[\[ -n "\$\{prune_baota_panel_backups_keep\}" \]\] \|\| return 0'
assert_contains "${RECLAIM_SCRIPT}" '\[\[ "\$\{truncate_var_log_messages\}" == "true" \]\] \|\| return 0'
assert_contains "${RECLAIM_SCRIPT}" '\[\[ "\$\{prune_stuhelper_old_images\}" == "true" \]\] \|\| return 0'
assert_contains "${RECLAIM_SCRIPT}" '\[\[ -n "\$\{prune_postgres_wal_archive_keep_hours\}" \]\] \|\| return 0'
assert_not_contains "${RECLAIM_SCRIPT}" 'rm -rf /var/lib/docker/volumes'
assert_not_contains "${RECLAIM_SCRIPT}" 'run_or_print docker volume prune'

dry_run_output="$(mktemp)"
trap 'rm -f "${dry_run_output}"' EXIT
"${RECLAIM_SCRIPT}" --dry-run >"${dry_run_output}"
assert_contains "${dry_run_output}" 'post-cleanup usage'

assert_contains "${RELEASE_RUNBOOK}" 'reclaim-production-disk-space.sh --apply'
assert_contains "${RELEASE_RUNBOOK}" 'No space left on device'
assert_contains "${RELEASE_RUNBOOK}" 'docker volume prune'
assert_contains "${RELEASE_RUNBOOK}" 'wal-archive'
assert_contains "${RELEASE_RUNBOOK}" 'quarantine-incomplete'
assert_contains "${RELEASE_RUNBOOK}" 'PITR'

echo "[reclaim-production-disk-space-contract] all assertions passed"
