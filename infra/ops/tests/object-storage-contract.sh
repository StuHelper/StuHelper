#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
RENDER_CONFIG="${REPO_ROOT}/infra/ops/render-local-object-storage-config.sh"
RENDER_TLS="${REPO_ROOT}/infra/ops/render-object-storage-tls.sh"
PREPARE_CLIENT_CA="${REPO_ROOT}/infra/ops/prepare-object-storage-client-ca.sh"
RCLONE_HELPER="${REPO_ROOT}/infra/ops/lib/rclone-object-storage.sh"
SYNC_BACKUPS="${REPO_ROOT}/infra/ops/sync-postgres-backups.sh"
FETCH_BACKUPS="${REPO_ROOT}/infra/ops/fetch-postgres-backups.sh"
SCHEDULED_BACKUP="${REPO_ROOT}/infra/ops/run-scheduled-backup.sh"
BASE_COMPOSE="${REPO_ROOT}/docker-compose.yml"
PROD_COMPOSE="${REPO_ROOT}/docker-compose.prod.yml"
PROD_DEPLOY="${REPO_ROOT}/infra/ops/prod-deploy.sh"

fail() {
  printf '[object-storage-contract][error] %s\n' "$*" >&2
  exit 1
}

assert_contains() {
  local file="$1"
  local pattern="$2"
  grep -Eq -- "${pattern}" "${file}" ||
    fail "expected ${file} to contain pattern: ${pattern}"
}

assert_not_contains() {
  local file="$1"
  local pattern="$2"
  if grep -Eq -- "${pattern}" "${file}"; then
    fail "expected ${file} not to contain pattern: ${pattern}"
  fi
}

assert_mode() {
  local expected="$1"
  local file="$2"
  local actual
  actual="$(stat -c '%a' "${file}")"
  [[ "${actual}" == "${expected}" ]] ||
    fail "expected ${file} mode ${expected}, got ${actual}"
}

for file in \
  "${RENDER_CONFIG}" \
  "${RENDER_TLS}" \
  "${PREPARE_CLIENT_CA}" \
  "${RCLONE_HELPER}" \
  "${SYNC_BACKUPS}" \
  "${FETCH_BACKUPS}" \
  "${SCHEDULED_BACKUP}"; do
  [[ -f "${file}" ]] || fail "missing file: ${file}"
  bash -n "${file}"
done

tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT
touch "${tmpdir}/empty.env" "${tmpdir}/empty.generated" "${tmpdir}/empty.generated.secrets"

render_env=(
  ENV_FILE="${tmpdir}/empty.env"
  GENERATED_ENV_FILE="${tmpdir}/empty.generated"
  GENERATED_SECRET_ENV_FILE="${tmpdir}/empty.generated.secrets"
)

distinct_dir="${tmpdir}/distinct"
env \
  "${render_env[@]}" \
  LOCAL_OBJECT_STORAGE_CONFIG_DIR="${distinct_dir}" \
  OBJECT_STORAGE_BUCKET=contract-app \
  OBJECT_STORAGE_ACCESS_KEY_ID=contract-app-key \
  OBJECT_STORAGE_SECRET_ACCESS_KEY=contract-app-secret-do-not-log \
  BACKUP_OBJECT_STORAGE_BUCKET=contract-backup \
  BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID=contract-backup-key \
  BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY=contract-backup-secret-do-not-log \
  bash "${RENDER_CONFIG}" >"${tmpdir}/distinct.log" 2>&1

assert_mode 600 "${distinct_dir}/s3.json"
assert_not_contains "${tmpdir}/distinct.log" 'contract-app-secret-do-not-log'
assert_not_contains "${tmpdir}/distinct.log" 'contract-backup-secret-do-not-log'
if find "${distinct_dir}" -maxdepth 1 -type f -name '.s3.json.*' | grep -q .; then
  fail "atomic identity rendering left a temporary file behind"
fi

if ! python3 - "${distinct_dir}/s3.json" <<'PY'
import json
from pathlib import Path
import sys

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
identities = {identity["name"]: identity for identity in payload["identities"]}
assert set(identities) == {"stuhelper-application", "stuhelper-backup"}
assert identities["stuhelper-application"]["credentials"] == [{
    "accessKey": "contract-app-key",
    "secretKey": "contract-app-secret-do-not-log",
}]
assert identities["stuhelper-application"]["actions"] == [
    "Read:contract-app",
    "List:contract-app",
    "Tagging:contract-app",
    "Write:contract-app",
]
assert identities["stuhelper-backup"]["credentials"] == [{
    "accessKey": "contract-backup-key",
    "secretKey": "contract-backup-secret-do-not-log",
}]
assert identities["stuhelper-backup"]["actions"] == [
    "Read:contract-backup",
    "List:contract-backup",
    "Tagging:contract-backup",
    "Write:contract-backup",
]
PY
then
  fail "distinct object-storage identities do not enforce bucket-scoped actions"
fi

shared_dir="${tmpdir}/shared"
env \
  "${render_env[@]}" \
  LOCAL_OBJECT_STORAGE_CONFIG_DIR="${shared_dir}" \
  OBJECT_STORAGE_BUCKET=contract-app \
  OBJECT_STORAGE_ACCESS_KEY_ID=contract-shared-key \
  OBJECT_STORAGE_SECRET_ACCESS_KEY=contract-shared-secret \
  BACKUP_OBJECT_STORAGE_BUCKET=contract-backup \
  BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID=contract-shared-key \
  BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY=contract-shared-secret \
  bash "${RENDER_CONFIG}" >"${tmpdir}/shared.log" 2>&1

if ! python3 - "${shared_dir}/s3.json" <<'PY'
import json
from pathlib import Path
import sys

payload = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
assert len(payload["identities"]) == 1
identity = payload["identities"][0]
assert identity["name"] == "stuhelper-local"
assert set(identity["actions"]) == {
    "Read:contract-app",
    "List:contract-app",
    "Tagging:contract-app",
    "Write:contract-app",
    "Read:contract-backup",
    "List:contract-backup",
    "Tagging:contract-backup",
    "Write:contract-backup",
}
PY
then
  fail "shared local object-storage identity does not cover both explicit buckets"
fi

if env \
  "${render_env[@]}" \
  LOCAL_OBJECT_STORAGE_CONFIG_DIR="${tmpdir}/invalid-same-key" \
  OBJECT_STORAGE_BUCKET=contract-app \
  OBJECT_STORAGE_ACCESS_KEY_ID=contract-key \
  OBJECT_STORAGE_SECRET_ACCESS_KEY=first-secret \
  BACKUP_OBJECT_STORAGE_BUCKET=contract-backup \
  BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID=contract-key \
  BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY=second-secret \
  bash "${RENDER_CONFIG}" >"${tmpdir}/invalid-same-key.log" 2>&1; then
  fail "the same access key with two secrets must be rejected"
fi

if env \
  "${render_env[@]}" \
  LOCAL_OBJECT_STORAGE_CONFIG_DIR="${tmpdir}/invalid-shared-secret" \
  OBJECT_STORAGE_BUCKET=contract-app \
  OBJECT_STORAGE_ACCESS_KEY_ID=contract-app-key \
  OBJECT_STORAGE_SECRET_ACCESS_KEY=shared-secret \
  BACKUP_OBJECT_STORAGE_BUCKET=contract-backup \
  BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID=contract-backup-key \
  BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY=shared-secret \
  bash "${RENDER_CONFIG}" >"${tmpdir}/invalid-shared-secret.log" 2>&1; then
  fail "distinct access keys sharing one secret must be rejected"
fi

env \
  "${render_env[@]}" \
  OBJECT_STORAGE_TLS_DIR="${distinct_dir}" \
  bash "${RENDER_TLS}" >"${tmpdir}/tls-first.log" 2>&1
certificate_before="$(sha256sum "${distinct_dir}/public.crt" | cut -d ' ' -f 1)"
env \
  "${render_env[@]}" \
  OBJECT_STORAGE_TLS_DIR="${distinct_dir}" \
  bash "${RENDER_TLS}" >"${tmpdir}/tls-second.log" 2>&1
certificate_after="$(sha256sum "${distinct_dir}/public.crt" | cut -d ' ' -f 1)"

[[ "${certificate_before}" == "${certificate_after}" ]] ||
  fail "an existing valid local object-storage certificate must be reused"
assert_mode 755 "${distinct_dir}"
assert_mode 600 "${distinct_dir}/ca.key"
assert_mode 600 "${distinct_dir}/private.key"
assert_mode 644 "${distinct_dir}/ca.crt"
assert_mode 644 "${distinct_dir}/public.crt"
openssl verify -CAfile "${distinct_dir}/ca.crt" "${distinct_dir}/public.crt" >/dev/null
openssl x509 -in "${distinct_dir}/public.crt" -noout -checkhost object-storage >/dev/null
openssl x509 -in "${distinct_dir}/public.crt" -noout -checkhost localhost >/dev/null
openssl x509 -in "${distinct_dir}/public.crt" -noout -checkip 127.0.0.1 >/dev/null

public_client_ca_dir="${tmpdir}/public-client-ca"
env \
  "${render_env[@]}" \
  OBJECT_STORAGE_CLIENT_CA_DIR="${public_client_ca_dir}" \
  OBJECT_STORAGE_TLS_CA="" \
  OBJECT_STORAGE_TLS_CA_HOST_PATH="" \
  bash "${PREPARE_CLIENT_CA}" >"${tmpdir}/public-client-ca.log" 2>&1
assert_mode 755 "${public_client_ca_dir}"
[[ ! -e "${public_client_ca_dir}/ca.crt" ]] ||
  fail "system-CA mode must leave the dedicated client CA mount empty"

private_client_ca_dir="${tmpdir}/private-client-ca"
env \
  "${render_env[@]}" \
  OBJECT_STORAGE_CLIENT_CA_DIR="${private_client_ca_dir}" \
  OBJECT_STORAGE_TLS_CA="/object-storage-tls/ca.crt" \
  OBJECT_STORAGE_TLS_CA_HOST_PATH="${distinct_dir}/ca.crt" \
  bash "${PREPARE_CLIENT_CA}" >"${tmpdir}/private-client-ca.log" 2>&1
assert_mode 755 "${private_client_ca_dir}"
assert_mode 644 "${private_client_ca_dir}/ca.crt"
cmp -s "${distinct_dir}/ca.crt" "${private_client_ca_dir}/ca.crt" ||
  fail "the staged client CA must exactly match the configured host certificate"
[[ "$(find "${private_client_ca_dir}" -maxdepth 1 -type f | wc -l)" -eq 1 ]] ||
  fail "the application client CA mount must contain only the public CA bundle"

if env \
  "${render_env[@]}" \
  OBJECT_STORAGE_CLIENT_CA_DIR="${tmpdir}/invalid-private-key-client-ca" \
  OBJECT_STORAGE_TLS_CA="/object-storage-tls/ca.crt" \
  OBJECT_STORAGE_TLS_CA_HOST_PATH="${distinct_dir}/private.key" \
  bash "${PREPARE_CLIENT_CA}" >"${tmpdir}/invalid-private-key.log" 2>&1; then
  fail "a private key must never be accepted as an application CA bundle"
fi

capture_file="${tmpdir}/docker-argv"
docker() {
  [[ "${RCLONE_CONFIG_TARGET_SECRET_ACCESS_KEY:-}" == "contract-runtime-secret" ]] ||
    fail "rclone secret was not passed through the child environment"
  printf '%s\n' "$@" >"${capture_file}"
}

# shellcheck disable=SC1091
source "${REPO_ROOT}/infra/ops/lib/common.sh"
# shellcheck disable=SC1090,SC1091
source "${RCLONE_HELPER}"

resolver_bin="${tmpdir}/resolver-bin"
mkdir -p "${resolver_bin}"
cat >"${resolver_bin}/docker" <<'DOCKER'
#!/usr/bin/env bash
set -euo pipefail

printf '%s\n' "$@" >>"${RESOLVER_DOCKER_CAPTURE:?}"
if [[ "${1:-}" == "network" && "${2:-}" == "inspect" ]]; then
  printf '%s\n' '[{"Name":"contract-backup-network","IPAM":{"Config":[{"Subnet":"172.31.0.0/16"}]},"Containers":{}}]'
  exit 0
fi

args=("$@")
(( ${#args[@]} >= 2 )) || exit 2
database="${args[${#args[@]} - 2]}"
host="${args[${#args[@]} - 1]}"
[[ "${database}" == "ahostsv4" || "${database}" == "ahostsv6" ]] || exit 2
case "${database}:${host}" in
  ahostsv4:backup-on-host.example.test)
    printf '203.0.113.77 STREAM %s\n' "${host}"
    ;;
  ahostsv4:backup-local-container.example.test)
    printf '172.31.0.9 STREAM %s\n' "${host}"
    ;;
  ahostsv4:backup-off-host.example.test)
    printf '198.51.100.42 STREAM %s\n' "${host}"
    ;;
  *) exit 2 ;;
esac
DOCKER
cat >"${resolver_bin}/ip" <<'IP'
#!/usr/bin/env bash
set -euo pipefail

[[ "$*" == "-j address show" ]] || exit 2
printf '%s\n' '[{"ifname":"eth0","addr_info":[{"family":"inet","local":"203.0.113.77","prefixlen":24}]}]'
IP
chmod +x "${resolver_bin}/docker" "${resolver_bin}/ip"
resolver_docker_capture="${tmpdir}/resolver-docker-argv"
: >"${resolver_docker_capture}"
export PATH="${resolver_bin}:${PATH}"
export RESOLVER_DOCKER_CAPTURE="${resolver_docker_capture}"

if ! (
  export OBJECT_STORAGE_ENDPOINT="https://objects.example.test"
  export OBJECT_STORAGE_USE_SSL="true"
  export OBJECT_STORAGE_FORCE_PATH_STYLE="false"
  export OBJECT_STORAGE_ACCESS_KEY_ID="contract-app-key"
  export OBJECT_STORAGE_SECRET_ACCESS_KEY="contract-app-secret"
  export BACKUP_OBJECT_STORAGE_ENDPOINT="https://192.0.2.25"
  export BACKUP_OBJECT_STORAGE_BUCKET="contract-backup"
  export BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID="contract-backup-key"
  export BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY="contract-backup-secret"
  export BACKUP_OBJECT_STORAGE_OFF_HOST_CONFIRMED="true"
  export BACKUP_OBJECT_STORAGE_FORCE_PATH_STYLE="false"
  export BACKUP_OBJECT_STORAGE_TLS_CA="${distinct_dir}/ca.crt"
  export BACKUP_OBJECT_STORAGE_TLS_INSECURE="false"
  require_production_object_storage
); then
  fail "valid HTTPS production object-storage configuration was rejected"
fi

if ! (
  export OBJECT_STORAGE_ENDPOINT="https://objects.example.test"
  export OBJECT_STORAGE_USE_SSL="true"
  export OBJECT_STORAGE_FORCE_PATH_STYLE="false"
  export OBJECT_STORAGE_ACCESS_KEY_ID="contract-app-key"
  export OBJECT_STORAGE_SECRET_ACCESS_KEY="contract-app-secret"
  export OBJECT_STORAGE_TLS_CA="/object-storage-tls/ca.crt"
  export OBJECT_STORAGE_TLS_CA_HOST_PATH="${distinct_dir}/ca.crt"
  export BACKUP_OBJECT_STORAGE_ENDPOINT="https://192.0.2.25"
  export BACKUP_OBJECT_STORAGE_BUCKET="contract-backup"
  export BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID="contract-backup-key"
  export BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY="contract-backup-secret"
  export BACKUP_OBJECT_STORAGE_OFF_HOST_CONFIRMED="true"
  export BACKUP_OBJECT_STORAGE_FORCE_PATH_STYLE="false"
  export BACKUP_OBJECT_STORAGE_TLS_INSECURE="false"
  require_production_object_storage
); then
  fail "valid private-CA production object-storage configuration was rejected"
fi

if (
  export OBJECT_STORAGE_ENDPOINT="https://objects.example.test"
  export OBJECT_STORAGE_USE_SSL="true"
  export OBJECT_STORAGE_FORCE_PATH_STYLE="false"
  export OBJECT_STORAGE_ACCESS_KEY_ID="contract-app-key"
  export OBJECT_STORAGE_SECRET_ACCESS_KEY="contract-app-secret"
  export OBJECT_STORAGE_TLS_CA="/object-storage-tls/ca.crt"
  export OBJECT_STORAGE_TLS_CA_HOST_PATH=""
  export BACKUP_OBJECT_STORAGE_ENDPOINT="https://192.0.2.25"
  export BACKUP_OBJECT_STORAGE_BUCKET="contract-backup"
  export BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID="contract-backup-key"
  export BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY="contract-backup-secret"
  export BACKUP_OBJECT_STORAGE_OFF_HOST_CONFIRMED="true"
  export BACKUP_OBJECT_STORAGE_TLS_INSECURE="false"
  require_production_object_storage
) >"${tmpdir}/missing-client-ca-host-path.log" 2>&1; then
  fail "a private application CA without a host source path must be rejected"
fi

if (
  export OBJECT_STORAGE_ENDPOINT="http://objects.example.test"
  export OBJECT_STORAGE_USE_SSL="true"
  export OBJECT_STORAGE_ACCESS_KEY_ID="contract-app-key"
  export OBJECT_STORAGE_SECRET_ACCESS_KEY="contract-app-secret"
  export BACKUP_OBJECT_STORAGE_ENDPOINT="https://192.0.2.25"
  export BACKUP_OBJECT_STORAGE_BUCKET="contract-backup"
  export BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID="contract-backup-key"
  export BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY="contract-backup-secret"
  export BACKUP_OBJECT_STORAGE_OFF_HOST_CONFIRMED="true"
  export BACKUP_OBJECT_STORAGE_TLS_INSECURE="false"
  require_production_object_storage
) >"${tmpdir}/invalid-http.log" 2>&1; then
  fail "plaintext production object-storage endpoint must be rejected"
fi

if (
  export OBJECT_STORAGE_ENDPOINT="https://objects.example.test"
  export OBJECT_STORAGE_USE_SSL="true"
  export OBJECT_STORAGE_ACCESS_KEY_ID="contract-app-key"
  export OBJECT_STORAGE_SECRET_ACCESS_KEY="contract-shared-secret"
  export BACKUP_OBJECT_STORAGE_ENDPOINT="https://192.0.2.25"
  export BACKUP_OBJECT_STORAGE_BUCKET="contract-backup"
  export BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID="contract-backup-key"
  export BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY="contract-shared-secret"
  export BACKUP_OBJECT_STORAGE_OFF_HOST_CONFIRMED="true"
  export BACKUP_OBJECT_STORAGE_TLS_INSECURE="false"
  require_production_object_storage
) >"${tmpdir}/invalid-shared-production-secret.log" 2>&1; then
  fail "production application and backup identities must not share one secret"
fi

if (
  export BACKUP_OBJECT_STORAGE_ENDPOINT="https://backups.example.test"
  export BACKUP_OBJECT_STORAGE_OFF_HOST_CONFIRMED="false"
  require_off_host_backup_object_storage
) >"${tmpdir}/off-host-unconfirmed.log" 2>&1; then
  fail "an unconfirmed off-host backup target must be rejected"
fi
assert_contains "${tmpdir}/off-host-unconfirmed.log" 'BACKUP_OBJECT_STORAGE_OFF_HOST_CONFIRMED must be true'

if (
  export BACKUP_OBJECT_STORAGE_ENDPOINT="https://minio:9000"
  export BACKUP_OBJECT_STORAGE_OFF_HOST_CONFIRMED="true"
  require_off_host_backup_object_storage
) >"${tmpdir}/off-host-compose-service.log" 2>&1; then
  fail "a Compose-local backup endpoint must be rejected even when it is asserted as off-host"
fi
assert_contains "${tmpdir}/off-host-compose-service.log" 'must use an off-host fully-qualified hostname'

if (
  export BACKUP_OBJECT_STORAGE_ENDPOINT="https://127.1:9000"
  export BACKUP_OBJECT_STORAGE_OFF_HOST_CONFIRMED="true"
  require_off_host_backup_object_storage
) >"${tmpdir}/off-host-legacy-loopback.log" 2>&1; then
  fail "an abbreviated numeric loopback endpoint must be rejected"
fi
assert_contains "${tmpdir}/off-host-legacy-loopback.log" 'must not use a legacy or abbreviated numeric IPv4 address'

if (
  export BACKUP_OBJECT_STORAGE_ENDPOINT="https://192.0.2.25"
  export BACKUP_OBJECT_STORAGE_OFF_HOST_CONFIRMED="true"
  export BACKUP_OBJECT_STORAGE_DOCKER_NETWORK="host"
  require_off_host_backup_object_storage
) >"${tmpdir}/off-host-host-network.log" 2>&1; then
  fail "host networking must be rejected for an asserted off-host backup target"
fi
assert_contains "${tmpdir}/off-host-host-network.log" 'must not use host or none'

if (
  export BACKUP_OBJECT_STORAGE_ENDPOINT="https://backup-on-host.example.test"
  export BACKUP_OBJECT_STORAGE_OFF_HOST_CONFIRMED="true"
  export BACKUP_OBJECT_STORAGE_DOCKER_NETWORK="contract-backup-network"
  require_off_host_backup_object_storage
) >"${tmpdir}/off-host-dns-local.log" 2>&1; then
  fail "a backup FQDN resolving to the production host must be rejected"
fi
assert_contains "${tmpdir}/off-host-dns-local.log" 'must not resolve to an address assigned to the production host'

if (
  export BACKUP_OBJECT_STORAGE_ENDPOINT="https://backup-local-container.example.test"
  export BACKUP_OBJECT_STORAGE_OFF_HOST_CONFIRMED="true"
  export BACKUP_OBJECT_STORAGE_DOCKER_NETWORK="contract-backup-network"
  require_off_host_backup_object_storage
) >"${tmpdir}/off-host-dns-local-container.log" 2>&1; then
  fail "a backup FQDN resolving into a same-host Docker network must be rejected"
fi
assert_contains "${tmpdir}/off-host-dns-local-container.log" 'must not resolve into a Docker network hosted on the production host'

if ! (
  export BACKUP_OBJECT_STORAGE_ENDPOINT="https://backup-off-host.example.test"
  export BACKUP_OBJECT_STORAGE_OFF_HOST_CONFIRMED="true"
  export BACKUP_OBJECT_STORAGE_DOCKER_NETWORK="contract-backup-network"
  require_off_host_backup_object_storage
); then
  fail "an asserted backup FQDN resolving only to a non-local address was rejected"
fi
assert_contains "${resolver_docker_capture}" '^--network$'
assert_contains "${resolver_docker_capture}" '^contract-backup-network$'
assert_contains "${resolver_docker_capture}" '^/usr/bin/getent$'

export BACKUP_OBJECT_STORAGE_ENDPOINT="https://objects.example.test"
export BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID="contract-runtime-key"
export BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY="contract-runtime-secret"
export BACKUP_OBJECT_STORAGE_FORCE_PATH_STYLE="false"
export BACKUP_OBJECT_STORAGE_TLS_INSECURE="false"
export RCLONE_IMAGE_REF="rclone/rclone:beta@sha256:f52965eba611ba8984117638b2a0539dcce170731937f93fbace66897d102698"
run_backup_object_storage_rclone \
  "1000:1000" \
  "type=bind,src=${tmpdir},dst=/source,readonly" \
  copy /source target:contract-backup/audit

assert_not_contains "${capture_file}" 'contract-runtime-secret'
assert_not_contains "${capture_file}" 'RCLONE_CONFIG_TARGET_SECRET_ACCESS_KEY='
assert_contains "${capture_file}" '^RCLONE_CONFIG_TARGET_SECRET_ACCESS_KEY$'
assert_contains "${capture_file}" '^--read-only$'
assert_contains "${capture_file}" '^ALL$'
assert_contains "${capture_file}" '^no-new-privileges$'
assert_contains "${capture_file}" '^copy$'
assert_contains "${capture_file}" '@sha256:[0-9a-f]{64}$'

assert_contains "${SYNC_BACKUPS}" 'source "\$\{SCRIPT_DIR\}/lib/rclone-object-storage\.sh"'
assert_contains "${FETCH_BACKUPS}" 'source "\$\{SCRIPT_DIR\}/lib/rclone-object-storage\.sh"'
assert_contains "${SYNC_BACKUPS}" 'copy /source'
assert_contains "${SYNC_BACKUPS}" 'host_user="\$\(id -u\):\$\(id -g\)"'
assert_contains "${SYNC_BACKUPS}" 'wal_archive_user="\$\{POSTGRES_WAL_ARCHIVE_CONTAINER_USER:-70:70\}"'
assert_contains "${SYNC_BACKUPS}" '"\$\{host_user\}"'
assert_contains "${SYNC_BACKUPS}" '"\$\{wal_archive_user\}"'
assert_not_contains "${SYNC_BACKUPS}" '"0:0"'
assert_contains "${SYNC_BACKUPS}" "--exclude '\\*\\.partial'"
assert_contains "${SYNC_BACKUPS}" "--exclude '\\*\\.partial\\.\\*'"
assert_contains "${SYNC_BACKUPS}" "--exclude '\\*\\.tmp'"
assert_contains "${SYNC_BACKUPS}" "--exclude '\\*\\.tmp\\.\\*'"
assert_contains "${SYNC_BACKUPS}" "--exclude 'quarantine-incomplete-\\*/\\*\\*'"
assert_contains "${SYNC_BACKUPS}" 'target:\$\{BACKUP_OBJECT_STORAGE_BUCKET\}/\$\{prefix\}/wal.*\\'
assert_contains "${SYNC_BACKUPS}" 'load_env_preserving BACKUP_OBJECT_STORAGE_OFF_HOST_REQUIRED'
assert_contains "${SYNC_BACKUPS}" 'require_off_host_backup_object_storage'
assert_contains "${SCHEDULED_BACKUP}" 'load_env_preserving BACKUP_OBJECT_STORAGE_OFF_HOST_REQUIRED'
assert_contains "${SCHEDULED_BACKUP}" "-path '/wal/quarantine-incomplete-\\*' -prune"
assert_contains "${FETCH_BACKUPS}" 'copy "target:'
assert_not_contains "${SYNC_BACKUPS}" '(^|[[:space:]])sync([[:space:]]|$)'
assert_not_contains "${FETCH_BACKUPS}" '(^|[[:space:]])sync([[:space:]]|$)'
assert_not_contains "${SYNC_BACKUPS}" 'minio|MC_HOST|mc mirror'
assert_not_contains "${FETCH_BACKUPS}" 'minio|MC_HOST|mc mirror'

assert_contains "${BASE_COMPOSE}" '^  object-storage:'
assert_contains "${BASE_COMPOSE}" 'profiles: \[dev-full, prod-parity\]'
assert_contains "${BASE_COMPOSE}" 'user: "1000:1000"'
assert_contains "${BASE_COMPOSE}" 'cap_drop:'
assert_not_contains "${BASE_COMPOSE}" '^  minio(-init)?:'
assert_not_contains "${PROD_COMPOSE}" '^  (object-storage|minio|minio-init):'
assert_contains "${PROD_COMPOSE}" 'OBJECT_STORAGE_CLIENT_CA_DIR:-\./infra/generated/object-storage-client-ca\}:/object-storage-tls:ro'
assert_not_contains "${PROD_COMPOSE}" '\./infra/generated/object-storage:/object-storage-tls:ro'
assert_contains "${PROD_DEPLOY}" 'require_production_object_storage'
assert_contains "${PROD_DEPLOY}" 'prepare-object-storage-client-ca\.sh'
assert_contains "${PROD_DEPLOY}" 'reject_placeholder OBJECT_STORAGE_SECRET_ACCESS_KEY'

printf '[object-storage-contract] all assertions passed\n'
