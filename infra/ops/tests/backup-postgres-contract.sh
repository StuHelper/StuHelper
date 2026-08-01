#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
BACKUP_FILE="${REPO_ROOT}/infra/ops/backup-postgres.sh"

fail() {
  echo "[backup-postgres-contract][error] $*" >&2
  exit 1
}

cleanup() {
  [[ -n "${tmpdir:-}" ]] && rm -rf "${tmpdir}"
}
trap cleanup EXIT

assert_contains() {
  local file="$1"
  local pattern="$2"
  grep -Eq -- "${pattern}" "${file}" ||
    fail "expected ${file} to contain pattern: ${pattern}"
}

line_number() {
  local pattern="$1"
  local line
  line="$(grep -nF -- "${pattern}" "${BACKUP_FILE}" | head -n1 | cut -d: -f1)"
  [[ -n "${line}" ]] || fail "expected pattern in ${BACKUP_FILE}: ${pattern}"
  printf '%s\n' "${line}"
}

load_env_line="$(line_number 'load_env')"
logical_url_line="$(line_number 'logical_url="${BACKUP_DATABASE_URL:-}"')"
replication_url_line="$(line_number 'replication_url="${REPLICATION_DATABASE_URL:-}"')"

if (( load_env_line >= logical_url_line )); then
  fail "backup-postgres.sh must load env before reading BACKUP_DATABASE_URL"
fi

if (( load_env_line >= replication_url_line )); then
  fail "backup-postgres.sh must load env before reading REPLICATION_DATABASE_URL"
fi

assert_contains "${BACKUP_FILE}" '--format=plain'
assert_contains "${BACKUP_FILE}" '--wal-method=stream'
assert_contains "${BACKUP_FILE}" '--workdir /backup'
assert_contains "${BACKUP_FILE}" 'pg_verifybackup /backup/pgdata'
assert_contains "${BACKUP_FILE}" '\.partial\.\$\{BASHPID\}'
assert_contains "${BACKUP_FILE}" 'refusing to overwrite an existing backup artifact'

tmpdir="$(mktemp -d)"
fake_bin="${tmpdir}/bin"
output_dir="${tmpdir}/output"
staging_dir="${tmpdir}/staging"
env_file="${tmpdir}/env"
generated_env_file="${tmpdir}/generated.env"
generated_secret_env_file="${tmpdir}/generated.secrets.env"
generated_obs_dir="${tmpdir}/generated/observability"
mkdir -p "${fake_bin}" "${output_dir}"
touch \
  "${env_file}" \
  "${generated_env_file}" \
  "${generated_secret_env_file}"

cat >"${fake_bin}/docker" <<'FAKE_DOCKER'
#!/usr/bin/env bash
set -euo pipefail

host_mount=""
for arg in "$@"; do
  case "${arg}" in
    *:/backup|*:/backup:ro)
      host_mount="${arg%%:/backup*}"
      ;;
  esac
done

case " $* " in
  *" pg_basebackup "*)
    [[ -n "${host_mount}" ]]
    mkdir -p "${host_mount}/pgdata/pg_wal"
    printf '18\n' >"${host_mount}/pgdata/PG_VERSION"
    if [[ "${FAKE_BASEBACKUP_FAIL:-false}" == "true" ]]; then
      exit 23
    fi
    printf '{"PostgreSQL-Backup-Manifest-Version": 2}\n' \
      >"${host_mount}/pgdata/backup_manifest"
    ;;
  *" pg_verifybackup "*)
    [[ -s "${host_mount}/pgdata/PG_VERSION" ]]
    [[ -s "${host_mount}/pgdata/backup_manifest" ]]
    ;;
  *)
    echo "unexpected fake docker invocation" >&2
    exit 24
    ;;
esac
FAKE_DOCKER
chmod +x "${fake_bin}/docker"

run_backup() {
  local output_file="$1"
  shift
  PATH="${fake_bin}:${PATH}" \
  ENV_FILE="${env_file}" \
  ENV_TEMPLATE_FILE="${env_file}" \
  GENERATED_ENV_FILE="${generated_env_file}" \
  GENERATED_SECRET_ENV_FILE="${generated_secret_env_file}" \
  GENERATED_OBS_DIR="${generated_obs_dir}" \
  BACKUP_STAGING_DIR="${staging_dir}" \
  REPLICATION_DATABASE_URL='postgres://replication:test@postgres/stuhelper' \
  BACKUP_MODE=basebackup \
    "$@" "${BACKUP_FILE}" "${output_file}"
}

success_file="${output_dir}/success.tar.gz"
run_backup "${success_file}" env >/dev/null

[[ -s "${success_file}" ]] || fail "successful base backup was not published"
[[ -s "${success_file}.sha256" ]] || fail "successful base backup sidecar was not published"
tar -tzf "${success_file}" >/dev/null ||
  fail "published base backup archive is unreadable"
(
  cd "${output_dir}"
  sha256sum -c "$(basename "${success_file}").sha256" >/dev/null
) || fail "published sidecar must be relocatable and valid"

if find "${output_dir}" -maxdepth 1 -name '*.partial*' -print -quit | grep -q .; then
  fail "successful backup left a partial artifact"
fi
if find "${staging_dir}" -mindepth 1 -print -quit | grep -q .; then
  fail "successful backup left staging data"
fi

failed_file="${output_dir}/failed.tar.gz"
if run_backup "${failed_file}" env FAKE_BASEBACKUP_FAIL=true >/dev/null 2>&1; then
  fail "base backup failure must propagate"
fi
[[ ! -e "${failed_file}" && ! -e "${failed_file}.sha256" ]] ||
  fail "failed base backup published a final artifact"
if find "${output_dir}" -maxdepth 1 -name '*.partial*' -print -quit | grep -q .; then
  fail "failed backup left a partial artifact"
fi
if find "${staging_dir}" -mindepth 1 -print -quit | grep -q .; then
  fail "failed backup left staging data"
fi

echo "[backup-postgres-contract] all assertions passed"
