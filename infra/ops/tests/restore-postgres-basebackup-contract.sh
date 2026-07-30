#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
RESTORE_SCRIPT="${REPO_ROOT}/infra/ops/restore-postgres-basebackup.sh"

fail() {
  echo "[restore-postgres-basebackup-contract][error] $*" >&2
  exit 1
}

cleanup() {
  [[ -n "${tmpdir:-}" ]] && rm -rf "${tmpdir}"
}
trap cleanup EXIT

tmpdir="$(mktemp -d)"
fixture_dir="${tmpdir}/fixture"
artifact_dir="${tmpdir}/artifacts"
restore_dir="${tmpdir}/restored"
mkdir -p "${fixture_dir}/pg_wal" "${artifact_dir}"

printf '18\n' >"${fixture_dir}/PG_VERSION"
printf '{"PostgreSQL-Backup-Manifest-Version": 2}\n' \
  >"${fixture_dir}/backup_manifest"
printf 'probe\n' >"${fixture_dir}/restore-probe"

archive="${artifact_dir}/basebackup.tar.gz"
tar -C "${fixture_dir}" -czf "${archive}" .
(
  cd "${artifact_dir}"
  sha256sum "$(basename "${archive}")" >"$(basename "${archive}").sha256"
)

ALLOW_DESTRUCTIVE=1 "${RESTORE_SCRIPT}" "${archive}" "${restore_dir}" >/dev/null
[[ "$(cat "${restore_dir}/PG_VERSION")" == "18" ]] ||
  fail "restored PG_VERSION is missing or incorrect"
[[ "$(cat "${restore_dir}/restore-probe")" == "probe" ]] ||
  fail "restored probe is missing or incorrect"

invalid_archive="${artifact_dir}/invalid.tar.gz"
cp "${archive}" "${invalid_archive}"
printf '%064d  invalid.tar.gz\n' 0 >"${invalid_archive}.sha256"
invalid_target="${tmpdir}/invalid-target"
mkdir -p "${invalid_target}"
printf 'must-survive\n' >"${invalid_target}/preexisting"

if ALLOW_DESTRUCTIVE=1 \
  "${RESTORE_SCRIPT}" "${invalid_archive}" "${invalid_target}" >/dev/null 2>&1; then
  fail "restore must reject an invalid SHA256 sidecar"
fi
[[ "$(cat "${invalid_target}/preexisting")" == "must-survive" ]] ||
  fail "checksum failure modified the restore target"

symlink_target="${tmpdir}/symlink-target"
mkdir -p "${symlink_target}"
ln -s "${symlink_target}" "${tmpdir}/pgdata-link"
if ALLOW_DESTRUCTIVE=1 \
  "${RESTORE_SCRIPT}" "${archive}" "${tmpdir}/pgdata-link" >/dev/null 2>&1; then
  fail "restore must reject a symlinked PGDATA target"
fi

echo "[restore-postgres-basebackup-contract] all assertions passed"
