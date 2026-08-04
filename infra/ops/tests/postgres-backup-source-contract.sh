#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
COMMON_LIB="${REPO_ROOT}/infra/ops/lib/common.sh"
BACKUP_SCRIPT="${REPO_ROOT}/infra/ops/backup-postgres.sh"
ACTIVATION_SCRIPT="${REPO_ROOT}/infra/ops/activate-existing-postgres-backups.sh"

fail() {
  printf '[postgres-backup-source-contract][error] %s\n' "$*" >&2
  exit 1
}

tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT
fake_bin="${tmpdir}/bin"
mkdir -p "${fake_bin}"

cat >"${fake_bin}/docker" <<'DOCKER'
#!/usr/bin/env bash
set -euo pipefail
case " $* " in
  *' com.docker.compose.service '* | *'com.docker.compose.service"}}'*)
    printf 'postgres\n'
    ;;
  *' NetworkSettings.Networks '* | *'NetworkSettings.Networks}}'*)
    printf '{"contract":{"IPAddress":"172.30.0.2","GlobalIPv6Address":""}}\n'
    ;;
  *)
    printf 'unexpected fake docker invocation: %s\n' "$*" >&2
    exit 41
    ;;
esac
DOCKER
chmod +x "${fake_bin}/docker"

# shellcheck source=../lib/common.sh
source "${COMMON_LIB}"

compose() {
  printf '%s\n' "${FAKE_POSTGRES_SOURCE_OBSERVATION:-172.30.0.2/32|stuhelper|f}"
}

export PATH="${fake_bin}:${PATH}"
export STACK_NAME=contract
export POSTGRES_CONTAINER_NAME=contract-postgres
export POSTGRES_DB=stuhelper
export EXTERNAL_POSTGRES_ENABLED=false
export BACKUP_DATABASE_URL='postgres://stuhelper_backup:contract@postgres:5432/stuhelper?sslmode=verify-full&sslrootcert=/tls/ca.crt'
export REPLICATION_DATABASE_URL='postgresql://stuhelper_replication:contract@postgres/stuhelper?sslmode=verify-full&sslrootcert=/tls/ca.crt'

require_internal_postgres_backup_sources_match_live_datastore

if (
  BACKUP_DATABASE_URL='postgres://stuhelper_backup:contract@other-postgres:5432/stuhelper?sslmode=verify-full'
  require_internal_postgres_backup_sources_match_live_datastore
) >"${tmpdir}/wrong-host.out" 2>"${tmpdir}/wrong-host.err"; then
  fail "backup source validation accepted another PostgreSQL host"
fi
grep -q 'do not target the canonical Compose datastore' "${tmpdir}/wrong-host.err" ||
  fail "wrong-host rejection did not produce a sanitized diagnostic"

if (
  BACKUP_DATABASE_URL='postgres://stuhelper_backup:contract@postgres:5432/stuhelper?sslmode=verify-full&host=other-postgres'
  require_internal_postgres_backup_sources_match_live_datastore
) >"${tmpdir}/routing-override.out" 2>"${tmpdir}/routing-override.err"; then
  fail "backup source validation accepted a query-level routing override"
fi
grep -q 'do not target the canonical Compose datastore' "${tmpdir}/routing-override.err" ||
  fail "routing-override rejection did not produce a sanitized diagnostic"

if (
  FAKE_POSTGRES_SOURCE_OBSERVATION='172.30.0.99/32|stuhelper|f'
  export FAKE_POSTGRES_SOURCE_OBSERVATION
  require_internal_postgres_backup_sources_match_live_datastore
) >"${tmpdir}/wrong-address.out" 2>"${tmpdir}/wrong-address.err"; then
  fail "backup source validation accepted an address outside the canonical container"
fi
grep -q 'do not resolve to the canonical live PostgreSQL container' "${tmpdir}/wrong-address.err" ||
  fail "wrong-address rejection did not produce a sanitized diagnostic"

if (
  FAKE_POSTGRES_SOURCE_OBSERVATION='172.30.0.2/32|stuhelper|t'
  export FAKE_POSTGRES_SOURCE_OBSERVATION
  require_internal_postgres_backup_sources_match_live_datastore
) >"${tmpdir}/recovery.out" 2>"${tmpdir}/recovery.err"; then
  fail "backup source validation accepted a standby"
fi
grep -q 'do not resolve to the canonical live PostgreSQL container' "${tmpdir}/recovery.err" ||
  fail "standby rejection did not produce a sanitized diagnostic"

if ! python3 - "${BACKUP_SCRIPT}" "${ACTIVATION_SCRIPT}" "${COMMON_LIB}" <<'PY'
from pathlib import Path
import sys

backup = Path(sys.argv[1]).read_text(encoding="utf-8")
activation = Path(sys.argv[2]).read_text(encoding="utf-8")
common = Path(sys.argv[3]).read_text(encoding="utf-8")
gate = "require_internal_postgres_backup_sources_match_live_datastore"
if not backup.index("load_env_preserving") < backup.index(gate) < backup.index("run_dump()"):
    raise SystemExit("backup script must validate the source after loading final env and before any dump")
if not activation.index("require_live_canonical_postgres_datastore") < activation.index(gate) < activation.index("BACKUP_MODE=dump"):
    raise SystemExit("activation must bind source URLs before creating recovery artifacts")
activation_validator = common.index("require_live_postgres_backup_activation()")
if common.index(gate, activation_validator) > common.index("manage-postgres-backup-activation.py", activation_validator):
    raise SystemExit("scheduled activation must validate source URLs before trusting its record")
PY
then
  fail "backup source identity gate is not wired into every protected path"
fi

printf '[postgres-backup-source-contract] all assertions passed\n'
