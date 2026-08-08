#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
UP_MIGRATION="${REPO_ROOT}/server/migrations/000037_drop_noncanonical_admission_policy_backups.up.sql"
DOWN_MIGRATION="${REPO_ROOT}/server/migrations/000037_drop_noncanonical_admission_policy_backups.down.sql"

fail() {
  echo "[noncanonical-admission-policy-backup-cleanup-contract][error] $*" >&2
  exit 1
}

for file in "${UP_MIGRATION}" "${DOWN_MIGRATION}"; do
  [[ -f "${file}" ]] || fail "missing cleanup migration: ${file}"
done

mapfile -t dropped_tables < <(
  sed -nE 's/^[[:space:]]*DROP[[:space:]]+TABLE[[:space:]]+IF[[:space:]]+EXISTS[[:space:]]+public\.([a-zA-Z0-9_]+);[[:space:]]*$/\1/p' "${UP_MIGRATION}"
)

expected_tables=(
  group_admission_policies_backup_20260608_743policy
  group_admission_policies_backup_20260608_targets
)

[[ "${#dropped_tables[@]}" -eq "${#expected_tables[@]}" ]] ||
  fail "cleanup migration must drop exactly two explicitly named tables"

for index in "${!expected_tables[@]}"; do
  [[ "${dropped_tables[${index}]}" == "${expected_tables[${index}]}" ]] ||
    fail "unexpected cleanup target at index ${index}: ${dropped_tables[${index}]}"
done

if grep -Eiq -- '\bCASCADE\b|EXECUTE[[:space:]]|format\(|LIKE[[:space:]]|SIMILAR[[:space:]]+TO' "${UP_MIGRATION}"; then
  fail "cleanup migration must not use CASCADE, patterns or dynamic SQL"
fi

if grep -Eiq -- 'CREATE[[:space:]]+TABLE|INSERT[[:space:]]+INTO|UPDATE[[:space:]]|DELETE[[:space:]]+FROM' "${DOWN_MIGRATION}"; then
  fail "down migration must not fabricate replacement backup data"
fi

grep -Eq -- '^[[:space:]]*SELECT[[:space:]]+1;[[:space:]]*$' "${DOWN_MIGRATION}" ||
  fail "down migration must be an explicit no-op"

echo "[noncanonical-admission-policy-backup-cleanup-contract] all assertions passed"
