#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
RENDER_CONFIG="${REPO_ROOT}/infra/ops/render-local-object-storage-config.sh"
RENDER_TLS="${REPO_ROOT}/infra/ops/render-object-storage-tls.sh"
PREPARE_CLIENT_CA="${REPO_ROOT}/infra/ops/prepare-object-storage-client-ca.sh"
RCLONE_HELPER="${REPO_ROOT}/infra/ops/lib/rclone-object-storage.sh"
COMMON_LIB="${REPO_ROOT}/infra/ops/lib/common.sh"
SYNC_BACKUPS="${REPO_ROOT}/infra/ops/sync-postgres-backups.sh"
FETCH_BACKUPS="${REPO_ROOT}/infra/ops/fetch-postgres-backups.sh"
SCHEDULED_BACKUP="${REPO_ROOT}/infra/ops/run-scheduled-backup.sh"
BASE_COMPOSE="${REPO_ROOT}/docker-compose.yml"
PROD_COMPOSE="${REPO_ROOT}/docker-compose.prod.yml"
PROD_DEPLOY="${REPO_ROOT}/infra/ops/prod-deploy.sh"
WAL_ARCHIVER_VALIDATOR="${REPO_ROOT}/infra/ops/validate-postgres-wal-archiver.py"

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
  [[ "${RCLONE_CONFIG_TARGET_ENDPOINT:-}" == "https://backup-off-host.example.test" ]] ||
    fail "rclone endpoint hostname was not normalized to match its validated address pin"
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
  printf '[{"Name":"contract-backup-network","Driver":"%s","IPAM":{"Config":[{"Subnet":"172.31.0.0/16"}]},"Containers":{"local":{"IPv4Address":"172.31.0.9/16","IPv6Address":""}}}]\n' "${RESOLVER_NETWORK_DRIVER:-bridge}"
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
  ahostsv4:backup-shared-network.example.test)
    printf '172.31.0.50 STREAM %s\n' "${host}"
    ;;
  ahostsv4:backup-off-host.example.test)
    printf '198.51.100.42 STREAM %s\n' "${host}"
    ;;
  ahostsv4:contract-backup.backup-off-host.example.test)
    printf '198.51.100.43 STREAM %s\n' "${host}"
    ;;
  ahostsv4:cos.ap-beijing.myqcloud.com)
    printf '169.254.0.49 STREAM %s\n' "${host}"
    ;;
  ahostsv4:stuhelper-1370411270.cos.ap-beijing.myqcloud.com)
    printf '169.254.0.49 STREAM %s\n' "${host}"
    ;;
  ahostsv4:backup-virtual-local.example.test)
    printf '198.51.100.42 STREAM %s\n' "${host}"
    ;;
  ahostsv4:contract-backup.backup-virtual-local.example.test)
    printf '203.0.113.77 STREAM %s\n' "${host}"
    ;;
  ahostsv6:backup-mapped-identity.example.test)
    printf '::ffff:198.51.100.42 STREAM %s\n' "${host}"
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
export BACKUP_OBJECT_STORAGE_LOCAL_IDENTITY_CIDRS="none"
export BACKUP_OBJECT_STORAGE_FORCE_PATH_STYLE="true"
export BACKUP_OBJECT_STORAGE_PROVIDER_PRIVATE_ENDPOINT="none"

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
  [[ -z "${BACKUP_OBJECT_STORAGE_PINNED_HOSTS:-}" ]]
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
  unset BACKUP_OBJECT_STORAGE_LOCAL_IDENTITY_CIDRS
  export BACKUP_OBJECT_STORAGE_ENDPOINT="https://192.0.2.25"
  export BACKUP_OBJECT_STORAGE_OFF_HOST_CONFIRMED="true"
  require_off_host_backup_object_storage
) >"${tmpdir}/off-host-local-identities-missing.log" 2>&1; then
  fail "an off-host assertion without a production public/NAT identity inventory must be rejected"
fi
assert_contains "${tmpdir}/off-host-local-identities-missing.log" 'BACKUP_OBJECT_STORAGE_LOCAL_IDENTITY_CIDRS is required'

if (
  export BACKUP_OBJECT_STORAGE_LOCAL_IDENTITY_CIDRS=" , , "
  export BACKUP_OBJECT_STORAGE_ENDPOINT="https://192.0.2.25"
  export BACKUP_OBJECT_STORAGE_OFF_HOST_CONFIRMED="true"
  require_off_host_backup_object_storage
) >"${tmpdir}/off-host-local-identities-empty.log" 2>&1; then
  fail "a separator-only production public/NAT identity inventory must be rejected"
fi
assert_contains "${tmpdir}/off-host-local-identities-empty.log" 'must contain at least one address or CIDR, or be exactly none'

if (
  export BACKUP_OBJECT_STORAGE_ENDPOINT="https://minio:9000"
  export BACKUP_OBJECT_STORAGE_OFF_HOST_CONFIRMED="true"
  require_off_host_backup_object_storage
) >"${tmpdir}/off-host-compose-service.log" 2>&1; then
  fail "a Compose-local backup endpoint must be rejected even when it is asserted as off-host"
fi
assert_contains "${tmpdir}/off-host-compose-service.log" 'must use an off-host fully-qualified hostname'

if (
  export BACKUP_OBJECT_STORAGE_ENDPOINT="https://backup-off-host.example.test."
  export BACKUP_OBJECT_STORAGE_OFF_HOST_CONFIRMED="true"
  require_off_host_backup_object_storage
) >"${tmpdir}/off-host-trailing-dot.log" 2>&1; then
  fail "a trailing-dot backup hostname that can bypass Docker host pinning must be rejected"
fi
assert_contains "${tmpdir}/off-host-trailing-dot.log" 'must not use a trailing-dot hostname'

if (
  export BACKUP_OBJECT_STORAGE_ENDPOINT="https://[2001:4860::1%25eth0]"
  export BACKUP_OBJECT_STORAGE_OFF_HOST_CONFIRMED="true"
  require_off_host_backup_object_storage
) >"${tmpdir}/off-host-scoped-ipv6.log" 2>&1; then
  fail "a scoped IPv6 backup endpoint that can name a local interface must be rejected"
fi
assert_contains "${tmpdir}/off-host-scoped-ipv6.log" 'must not use an IPv6 zone identifier'

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
  export BACKUP_OBJECT_STORAGE_LOCAL_IDENTITY_CIDRS="198.51.100.0/24"
  export BACKUP_OBJECT_STORAGE_ENDPOINT="https://backup-off-host.example.test"
  export BACKUP_OBJECT_STORAGE_OFF_HOST_CONFIRMED="true"
  export BACKUP_OBJECT_STORAGE_DOCKER_NETWORK="contract-backup-network"
  require_off_host_backup_object_storage
) >"${tmpdir}/off-host-dns-nat-identity.log" 2>&1; then
  fail "a backup FQDN resolving to a configured production NAT identity must be rejected"
fi
assert_contains "${tmpdir}/off-host-dns-nat-identity.log" 'must not resolve to a configured public/NAT/LB identity'

if (
  export BACKUP_OBJECT_STORAGE_LOCAL_IDENTITY_CIDRS="::ffff:198.51.100.42/128"
  export BACKUP_OBJECT_STORAGE_ENDPOINT="https://backup-mapped-identity.example.test"
  export BACKUP_OBJECT_STORAGE_OFF_HOST_CONFIRMED="true"
  export BACKUP_OBJECT_STORAGE_DOCKER_NETWORK="contract-backup-network"
  require_off_host_backup_object_storage
) >"${tmpdir}/off-host-dns-mapped-nat-identity.log" 2>&1; then
  fail "an IPv4-mapped production identity must reject the equivalent resolved endpoint address"
fi
assert_contains "${tmpdir}/off-host-dns-mapped-nat-identity.log" 'must not resolve to a configured public/NAT/LB identity'

if (
  export BACKUP_OBJECT_STORAGE_ENDPOINT="https://cos.ap-beijing.myqcloud.com"
  export BACKUP_OBJECT_STORAGE_BUCKET="stuhelper-1370411270"
  export BACKUP_OBJECT_STORAGE_OFF_HOST_CONFIRMED="true"
  export BACKUP_OBJECT_STORAGE_FORCE_PATH_STYLE="false"
  export BACKUP_OBJECT_STORAGE_DOCKER_NETWORK="contract-backup-network"
  export BACKUP_OBJECT_STORAGE_PROVIDER="TencentCOS"
  export BACKUP_OBJECT_STORAGE_REGION="ap-beijing"
  export BACKUP_OBJECT_STORAGE_TLS_CA=""
  export BACKUP_OBJECT_STORAGE_TLS_INSECURE="false"
  export BACKUP_OBJECT_STORAGE_PROVIDER_PRIVATE_ENDPOINT="none"
  require_off_host_backup_object_storage
) >"${tmpdir}/off-host-cos-private-unconfirmed.log" 2>&1; then
  fail "a provider-private COS VIP must remain rejected without an explicit provider mode"
fi
assert_contains "${tmpdir}/off-host-cos-private-unconfirmed.log" 'must not resolve to a loopback, unspecified, link-local, or multicast address'

if (
  export BACKUP_OBJECT_STORAGE_ENDPOINT="https://cos.ap-beijing.myqcloud.com"
  export BACKUP_OBJECT_STORAGE_BUCKET="stuhelper-1370411270"
  export BACKUP_OBJECT_STORAGE_OFF_HOST_CONFIRMED="true"
  export BACKUP_OBJECT_STORAGE_FORCE_PATH_STYLE="false"
  export BACKUP_OBJECT_STORAGE_DOCKER_NETWORK="contract-backup-network"
  export BACKUP_OBJECT_STORAGE_PROVIDER="Other"
  export BACKUP_OBJECT_STORAGE_REGION="ap-beijing"
  export BACKUP_OBJECT_STORAGE_TLS_CA=""
  export BACKUP_OBJECT_STORAGE_TLS_INSECURE="false"
  export BACKUP_OBJECT_STORAGE_PROVIDER_PRIVATE_ENDPOINT="tencent-cos"
  require_off_host_backup_object_storage
) >"${tmpdir}/off-host-cos-private-wrong-provider.log" 2>&1; then
  fail "the Tencent COS private-endpoint mode must reject another rclone provider"
fi
assert_contains "${tmpdir}/off-host-cos-private-wrong-provider.log" 'requires BACKUP_OBJECT_STORAGE_PROVIDER=TencentCOS'

if (
  export BACKUP_OBJECT_STORAGE_ENDPOINT="https://cos.ap-beijing.myqcloud.com"
  export BACKUP_OBJECT_STORAGE_BUCKET="stuhelper-1370411270"
  export BACKUP_OBJECT_STORAGE_OFF_HOST_CONFIRMED="true"
  export BACKUP_OBJECT_STORAGE_FORCE_PATH_STYLE="false"
  export BACKUP_OBJECT_STORAGE_DOCKER_NETWORK="contract-backup-network"
  export BACKUP_OBJECT_STORAGE_PROVIDER="TencentCOS"
  export BACKUP_OBJECT_STORAGE_REGION="ap-beijing"
  export BACKUP_OBJECT_STORAGE_TLS_CA=""
  export BACKUP_OBJECT_STORAGE_TLS_INSECURE="true"
  export BACKUP_OBJECT_STORAGE_PROVIDER_PRIVATE_ENDPOINT="tencent-cos"
  require_off_host_backup_object_storage
) >"${tmpdir}/off-host-cos-private-insecure.log" 2>&1; then
  fail "the Tencent COS private-endpoint mode must reject disabled certificate verification"
fi
assert_contains "${tmpdir}/off-host-cos-private-insecure.log" 'requires public-CA TLS verification'

if ! (
  export BACKUP_OBJECT_STORAGE_ENDPOINT="https://cos.ap-beijing.myqcloud.com"
  export BACKUP_OBJECT_STORAGE_BUCKET="stuhelper-1370411270"
  export BACKUP_OBJECT_STORAGE_OFF_HOST_CONFIRMED="true"
  export BACKUP_OBJECT_STORAGE_FORCE_PATH_STYLE="false"
  export BACKUP_OBJECT_STORAGE_DOCKER_NETWORK="contract-backup-network"
  export BACKUP_OBJECT_STORAGE_PROVIDER="TencentCOS"
  export BACKUP_OBJECT_STORAGE_REGION="ap-beijing"
  export BACKUP_OBJECT_STORAGE_TLS_CA=""
  export BACKUP_OBJECT_STORAGE_TLS_INSECURE="false"
  export BACKUP_OBJECT_STORAGE_PROVIDER_PRIVATE_ENDPOINT="tencent-cos"
  require_off_host_backup_object_storage
  [[ "${BACKUP_OBJECT_STORAGE_PINNED_HOSTS:-}" == $'cos.ap-beijing.myqcloud.com=169.254.0.49\nstuhelper-1370411270.cos.ap-beijing.myqcloud.com=169.254.0.49' ]]
); then
  fail "an explicitly constrained Tencent COS provider-private endpoint was rejected"
fi

if (
  export BACKUP_OBJECT_STORAGE_ENDPOINT="https://backup-virtual-local.example.test"
  export BACKUP_OBJECT_STORAGE_BUCKET="contract-backup"
  export BACKUP_OBJECT_STORAGE_OFF_HOST_CONFIRMED="true"
  export BACKUP_OBJECT_STORAGE_FORCE_PATH_STYLE="false"
  export BACKUP_OBJECT_STORAGE_DOCKER_NETWORK="contract-backup-network"
  require_off_host_backup_object_storage
) >"${tmpdir}/off-host-virtual-local.log" 2>&1; then
  fail "a virtual-hosted S3 transfer hostname resolving to the production host must be rejected"
fi
assert_contains "${tmpdir}/off-host-virtual-local.log" 'must not resolve to an address assigned to the production host'

if (
  export BACKUP_OBJECT_STORAGE_ENDPOINT="https://backup-local-container.example.test"
  export BACKUP_OBJECT_STORAGE_OFF_HOST_CONFIRMED="true"
  export BACKUP_OBJECT_STORAGE_DOCKER_NETWORK="contract-backup-network"
  require_off_host_backup_object_storage
) >"${tmpdir}/off-host-dns-local-container.log" 2>&1; then
  fail "a backup FQDN resolving into a same-host Docker network must be rejected"
fi
assert_contains "${tmpdir}/off-host-dns-local-container.log" 'must not resolve into a Docker network hosted on the production host'

if (
  export RESOLVER_NETWORK_DRIVER="macvlan"
  export BACKUP_OBJECT_STORAGE_ENDPOINT="https://backup-local-container.example.test"
  export BACKUP_OBJECT_STORAGE_OFF_HOST_CONFIRMED="true"
  export BACKUP_OBJECT_STORAGE_DOCKER_NETWORK="contract-backup-network"
  require_off_host_backup_object_storage
) >"${tmpdir}/off-host-dns-macvlan-local-container.log" 2>&1; then
  fail "a backup FQDN resolving to a local container on a shared Docker network must be rejected"
fi
assert_contains "${tmpdir}/off-host-dns-macvlan-local-container.log" 'must not resolve into a Docker network hosted on the production host'

if ! (
  export RESOLVER_NETWORK_DRIVER="macvlan"
  export BACKUP_OBJECT_STORAGE_ENDPOINT="https://backup-shared-network.example.test"
  export BACKUP_OBJECT_STORAGE_OFF_HOST_CONFIRMED="true"
  export BACKUP_OBJECT_STORAGE_DOCKER_NETWORK="contract-backup-network"
  require_off_host_backup_object_storage
); then
  fail "an off-host address in a shared macvlan subnet was incorrectly treated as host-local"
fi

if ! (
  export BACKUP_OBJECT_STORAGE_ENDPOINT="https://backup-off-host.example.test"
  export BACKUP_OBJECT_STORAGE_BUCKET="contract-backup"
  export BACKUP_OBJECT_STORAGE_OFF_HOST_CONFIRMED="true"
  export BACKUP_OBJECT_STORAGE_FORCE_PATH_STYLE="false"
  export BACKUP_OBJECT_STORAGE_DOCKER_NETWORK="contract-backup-network"
  require_off_host_backup_object_storage
  [[ "${BACKUP_OBJECT_STORAGE_PINNED_HOSTS:-}" == $'backup-off-host.example.test=198.51.100.42\ncontract-backup.backup-off-host.example.test=198.51.100.43' ]]
); then
  fail "an asserted virtual-hosted backup FQDN did not export every validated transfer hostname and address"
fi
assert_contains "${resolver_docker_capture}" '^--network$'
assert_contains "${resolver_docker_capture}" '^contract-backup-network$'
assert_contains "${resolver_docker_capture}" '^/usr/bin/getent$'

export BACKUP_OBJECT_STORAGE_ENDPOINT="https://BACKUP-OFF-HOST.EXAMPLE.TEST"
export BACKUP_OBJECT_STORAGE_BUCKET="contract-backup"
export BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID="contract-runtime-key"
export BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY="contract-runtime-secret"
export BACKUP_OBJECT_STORAGE_FORCE_PATH_STYLE="false"
export BACKUP_OBJECT_STORAGE_TLS_INSECURE="false"
export RCLONE_IMAGE_REF="rclone/rclone:beta@sha256:f52965eba611ba8984117638b2a0539dcce170731937f93fbace66897d102698"
if (
  export BACKUP_OBJECT_STORAGE_PINNED_HOSTS="backup-off-host.example.test=198.51.100.42"
  run_backup_object_storage_rclone \
    "1000:1000" \
    "type=bind,src=${tmpdir},dst=/source,readonly" \
    copy /source target:contract-backup/audit
) >"${tmpdir}/missing-virtual-host-pin.log" 2>&1; then
  fail "a virtual-hosted transfer without a validated bucket hostname pin must be rejected"
fi
assert_contains "${tmpdir}/missing-virtual-host-pin.log" 'is missing validated addresses for: contract-backup\.backup-off-host\.example\.test'

export BACKUP_OBJECT_STORAGE_PINNED_HOSTS=$'backup-off-host.example.test=198.51.100.42\nbackup-off-host.example.test=2001:db8::42\ncontract-backup.backup-off-host.example.test=198.51.100.43\ncontract-backup.backup-off-host.example.test=2001:db8::43'
run_backup_object_storage_rclone \
  "1000:1000" \
  "type=bind,src=${tmpdir},dst=/source,readonly" \
  copy /source target:contract-backup/audit

assert_not_contains "${capture_file}" 'contract-runtime-secret'
assert_not_contains "${capture_file}" 'RCLONE_CONFIG_TARGET_SECRET_ACCESS_KEY='
assert_contains "${capture_file}" '^RCLONE_CONFIG_TARGET_SECRET_ACCESS_KEY$'
for proxy_environment_key in \
  HTTP_PROXY HTTPS_PROXY FTP_PROXY ALL_PROXY NO_PROXY \
  http_proxy https_proxy ftp_proxy all_proxy no_proxy; do
  assert_contains "${capture_file}" "^${proxy_environment_key}=$"
done
assert_contains "${capture_file}" '^--read-only$'
assert_contains "${capture_file}" '^ALL$'
assert_contains "${capture_file}" '^no-new-privileges$'
assert_contains "${capture_file}" '^--add-host$'
assert_contains "${capture_file}" '^backup-off-host\.example\.test=198\.51\.100\.42$'
assert_contains "${capture_file}" '^backup-off-host\.example\.test=2001:db8::42$'
assert_contains "${capture_file}" '^contract-backup\.backup-off-host\.example\.test=198\.51\.100\.43$'
assert_contains "${capture_file}" '^contract-backup\.backup-off-host\.example\.test=2001:db8::43$'
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
assert_contains "${SYNC_BACKUPS}" 'load_env_preserving BACKUP_OBJECT_STORAGE_OFF_HOST_REQUIRED LOCAL_STATE_DIR'
assert_contains "${SYNC_BACKUPS}" 'require_off_host_backup_object_storage'
assert_contains "${SYNC_BACKUPS}" 'APP_ENV:-.*production'
assert_contains "${SYNC_BACKUPS}" 'unset BACKUP_OBJECT_STORAGE_PINNED_HOSTS'
assert_contains "${SYNC_BACKUPS}" 'require_live_postgres_wal_archive_volume'
assert_contains "${SYNC_BACKUPS}" 'require_live_postgres_wal_archiving'
assert_contains "${COMMON_LIB}" 'pg_switch_wal'
assert_contains "${COMMON_LIB}" 'pg_postmaster_start_time'
assert_contains "${COMMON_LIB}" 'last_archived_wal'
assert_contains "${WAL_ARCHIVER_VALIDATOR}" 'archive_mode != "on"'
assert_contains "${WAL_ARCHIVER_VALIDATOR}" 'archive_timeout != "15min"'
assert_contains "${WAL_ARCHIVER_VALIDATOR}" '/var/lib/postgresql/wal-archive/%f'
assert_contains "${SYNC_BACKUPS}" 'fresh cluster-bound continuous WAL/PITR evidence was verified'
assert_contains "${FETCH_BACKUPS}" 'load_env_preserving \\'
assert_contains "${FETCH_BACKUPS}" 'BACKUP_OBJECT_STORAGE_OFF_HOST_REQUIRED \\'
assert_contains "${FETCH_BACKUPS}" 'LOCAL_STATE_DIR \\'
assert_contains "${FETCH_BACKUPS}" 'POSTGRES_WAL_RESTORE_DIR'
assert_contains "${FETCH_BACKUPS}" 'unset BACKUP_OBJECT_STORAGE_PINNED_HOSTS'
assert_contains "${FETCH_BACKUPS}" 'require_off_host_backup_object_storage'
assert_contains "${FETCH_BACKUPS}" 'APP_ENV:-.*production'
assert_contains "${SCHEDULED_BACKUP}" 'load_env_preserving \\'
assert_contains "${SCHEDULED_BACKUP}" 'BACKUP_OBJECT_STORAGE_OFF_HOST_REQUIRED \\'
assert_contains "${SCHEDULED_BACKUP}" 'LOCAL_STATE_DIR \\'
assert_contains "${SCHEDULED_BACKUP}" 'BACKUP_STAGING_DIR'
assert_contains "${SCHEDULED_BACKUP}" 'external PostgreSQL selected; skipping local WAL archive pruning'
assert_contains "${SCHEDULED_BACKUP}" 'true\) require_off_host_backup_object_storage'
assert_contains "${SCHEDULED_BACKUP}" 'protected_bash=\('
assert_contains "${SCHEDULED_BACKUP}" '--unset=BASH_ENV'
assert_contains "${SCHEDULED_BACKUP}" '--unset=ENV'
assert_contains "${SCHEDULED_BACKUP}" '/bin/bash'
assert_contains "${SCHEDULED_BACKUP}" '--noprofile'
assert_contains "${SCHEDULED_BACKUP}" '--norc'
assert_contains "${SCHEDULED_BACKUP}" '"\$\{protected_bash\[@\]\}" "\$\{SCRIPT_DIR\}/sync-postgres-backups\.sh"'
assert_contains "${SCHEDULED_BACKUP}" "-path '/wal/quarantine-incomplete-\\*' -prune"
assert_contains "${FETCH_BACKUPS}" 'copy "target:'
assert_not_contains "${SYNC_BACKUPS}" '(^|[[:space:]])sync([[:space:]]|$)'
assert_not_contains "${FETCH_BACKUPS}" '(^|[[:space:]])sync([[:space:]]|$)'
assert_not_contains "${SYNC_BACKUPS}" 'minio|MC_HOST|mc mirror'
assert_not_contains "${FETCH_BACKUPS}" 'minio|MC_HOST|mc mirror'

if ! POSTGRES_ARCHIVE_MODE=on POSTGRES_ARCHIVE_TIMEOUT=15min EXTERNAL_POSTGRES_ENABLED=false bash -c '
  set -euo pipefail
  source "$1"
  require_production_postgres_archiving
' bash "${COMMON_LIB}"; then
  fail "the production archiving config gate rejected internal archive_mode=on"
fi
if POSTGRES_ARCHIVE_MODE=off POSTGRES_ARCHIVE_TIMEOUT=15min EXTERNAL_POSTGRES_ENABLED=false bash -c '
  set -euo pipefail
  source "$1"
  require_production_postgres_archiving
' bash "${COMMON_LIB}" >"${tmpdir}/archive-mode-off.out" 2>"${tmpdir}/archive-mode-off.err"; then
  fail "the production archiving config gate accepted internal archive_mode=off"
fi
grep -q 'POSTGRES_ARCHIVE_MODE must be on' "${tmpdir}/archive-mode-off.err" ||
  fail "disabled production archive mode did not report the configuration boundary"
if POSTGRES_ARCHIVE_MODE=on POSTGRES_ARCHIVE_TIMEOUT=1d EXTERNAL_POSTGRES_ENABLED=false bash -c '
  set -euo pipefail
  source "$1"
  require_production_postgres_archiving
' bash "${COMMON_LIB}" >"${tmpdir}/archive-timeout-invalid.out" 2>"${tmpdir}/archive-timeout-invalid.err"; then
  fail "the production archiving config gate accepted a non-canonical archive timeout"
fi
grep -q 'POSTGRES_ARCHIVE_TIMEOUT must be 15min' "${tmpdir}/archive-timeout-invalid.err" ||
  fail "invalid production archive timeout did not report the configuration boundary"
if POSTGRES_ARCHIVE_MODE=off EXTERNAL_POSTGRES_ENABLED=true bash -c '
  set -euo pipefail
  source "$1"
  require_production_postgres_archiving
' bash "${COMMON_LIB}" >"${tmpdir}/external-pitr-evidence-missing.out" 2>"${tmpdir}/external-pitr-evidence-missing.err"; then
  fail "the production archiving config gate accepted external PostgreSQL without its fixed PITR evidence path"
fi
grep -q 'EXTERNAL_POSTGRES_PITR_EVIDENCE_FILE must be' "${tmpdir}/external-pitr-evidence-missing.err" ||
  fail "missing external PITR evidence path did not report the production boundary"
if ! POSTGRES_ARCHIVE_MODE=off \
  EXTERNAL_POSTGRES_ENABLED=true \
  EXTERNAL_POSTGRES_PITR_EVIDENCE_FILE=/etc/stuhelper/external-postgres-pitr-evidence.json \
  bash -c '
  set -euo pipefail
  source "$1"
  require_production_postgres_archiving
' bash "${COMMON_LIB}"; then
  fail "the production archiving config gate incorrectly claimed external PostgreSQL ownership"
fi

if ! bash -c '
  set -euo pipefail
  source "$1"
  docker() {
    if [[ "$1" == volume && "$2" == inspect && "$3" == contract-postgres-wal-archive ]]; then
      return 0
    fi
    if [[ "$1" == container && "$2" == inspect && "$3" == --format && "$5" == contract-postgres ]]; then
      case "$4" in
        "{{.State.Running}}") printf "%s\n" true ;;
        "{{json .Mounts}}") printf "%s\n" "[{\"Type\":\"volume\",\"Name\":\"contract-postgres-wal-archive\",\"Destination\":\"/var/lib/postgresql/wal-archive\",\"RW\":true}]" ;;
        *) return 90 ;;
      esac
      return 0
    fi
    return 90
  }
  require_live_postgres_wal_archive_volume contract-postgres-wal-archive contract-postgres
' bash "${COMMON_LIB}"; then
  fail "the live WAL volume validator rejected the actual PostgreSQL archive mount"
fi

if bash -c '
  set -euo pipefail
  source "$1"
  docker() {
    if [[ "$1" == volume && "$2" == inspect ]]; then
      return 1
    fi
    return 90
  }
  require_live_postgres_wal_archive_volume missing-wal-volume contract-postgres
' bash "${COMMON_LIB}" >"${tmpdir}/missing-wal-volume.out" 2>"${tmpdir}/missing-wal-volume.err"; then
  fail "the live WAL volume validator accepted a missing volume"
fi
grep -q 'refusing to let Docker create an empty backup source' "${tmpdir}/missing-wal-volume.err" ||
  fail "missing WAL volume rejection did not explain the empty-volume risk"

if bash -c '
  set -euo pipefail
  source "$1"
  docker() {
    if [[ "$1" == volume && "$2" == inspect ]]; then
      return 0
    fi
    if [[ "$1" == container && "$2" == inspect && "$3" == --format ]]; then
      case "$4" in
        "{{.State.Running}}") printf "%s\n" true ;;
        "{{json .Mounts}}") printf "%s\n" "[{\"Type\":\"volume\",\"Name\":\"different-wal-volume\",\"Destination\":\"/var/lib/postgresql/wal-archive\",\"RW\":true}]" ;;
        *) return 90 ;;
      esac
      return 0
    fi
    return 90
  }
  require_live_postgres_wal_archive_volume configured-wal-volume contract-postgres
' bash "${COMMON_LIB}" >"${tmpdir}/mismatched-wal-volume.out" 2>"${tmpdir}/mismatched-wal-volume.err"; then
  fail "the live WAL volume validator accepted a volume not mounted by PostgreSQL"
fi
grep -q 'is not the writable WAL archive mounted by contract-postgres' "${tmpdir}/mismatched-wal-volume.err" ||
  fail "mismatched WAL volume rejection did not report the live-container boundary"

healthy_archiver_status="$(python3 - <<'PY'
import json

archive_command = '''sh -c 'dest=/var/lib/postgresql/wal-archive/%f; tmp="$dest.tmp.$$"; if [ -f "$dest" ]; then cmp -s %p "$dest"; else cp %p "$tmp" && mv "$tmp" "$dest"; fi\''''
print(json.dumps({
    "archive_mode": "on",
    "archive_command": archive_command,
    "archive_timeout": "15min",
    "archived_count": 2,
    "failed_count": 0,
    "last_archived_wal": "000000010000000000000001",
    "last_archived_epoch": 200,
    "last_failed_epoch": None,
    "postmaster_started_epoch": 100,
}, separators=(",", ":")))
PY
)"
WAL_ARCHIVER_STATUS="${healthy_archiver_status}"
fresh_archiver_status="$(python3 - "${healthy_archiver_status}" <<'PY'
import json
import sys

document = json.loads(sys.argv[1])
document["archived_count"] = 3
document["last_archived_wal"] = "000000010000000000000002"
document["last_archived_epoch"] = 300
print(json.dumps(document, separators=(",", ":")))
PY
)"
healthy_probe_state="${tmpdir}/healthy-wal-probe-state"
HEALTHY_PROBE_STATE="${healthy_probe_state}"
FRESH_WAL_ARCHIVER_STATUS="${fresh_archiver_status}"
docker() {
  if [[ "$1" == exec && "$2" == contract-postgres && "$*" == *"SELECT archived_count::text"* ]]; then
    printf '%s\n' 2
    return 0
  fi
  if [[ "$1" == exec && "$2" == contract-postgres && "$*" == *pg_switch_wal* ]]; then
    : >"${HEALTHY_PROBE_STATE}.switched"
    return 0
  fi
  if [[ "$1" == exec && "$2" == contract-postgres && "$*" == *json_build_object* ]]; then
    if [[ -f "${HEALTHY_PROBE_STATE}.switched" ]]; then
      printf '%s\n' "${FRESH_WAL_ARCHIVER_STATUS}"
    else
      printf '%s\n' "${WAL_ARCHIVER_STATUS}"
    fi
    return 0
  fi
  if [[ "$1" == exec && "$2" == contract-postgres && "$3" == /bin/sh ]]; then
    return 0
  fi
  return 90
}
if ! require_live_postgres_wal_archiving contract-postgres contract-admin contract-db; then
  fail "the live WAL archiver validator rejected current post-start archive progress"
fi
[[ -f "${healthy_probe_state}.switched" ]] ||
  fail "the live WAL archiver validator reused stale success instead of forcing a fresh probe"

if python3 "${WAL_ARCHIVER_VALIDATOR}" \
  --status-json "${healthy_archiver_status}" \
  --minimum-archived-count 2 \
  >"${tmpdir}/stale-archive-count.out" 2>&1; then
  fail "the live WAL archiver validator accepted an archive count that did not advance"
else
  validator_status=$?
fi
[[ "${validator_status}" -eq 2 ]] ||
  fail "an archive count that has not advanced must request another live probe"
if ! python3 "${WAL_ARCHIVER_VALIDATOR}" \
  --status-json "${healthy_archiver_status}" \
  --minimum-archived-count 1 >/dev/null; then
  fail "the live WAL archiver validator rejected a newly advanced archive count"
fi

drifted_archive_command_status="$(python3 - "${healthy_archiver_status}" <<'PY'
import json
import sys

document = json.loads(sys.argv[1])
document["archive_command"] = "cp %p /var/lib/postgresql/wal-archive/%f"
print(json.dumps(document, separators=(",", ":")))
PY
)"
if python3 "${WAL_ARCHIVER_VALIDATOR}" \
  --status-json "${drifted_archive_command_status}" \
  >"${tmpdir}/drifted-archive-command.out" 2>&1; then
  fail "the live WAL archiver validator accepted a drifted archive command"
fi
grep -q 'does not match the protected command' "${tmpdir}/drifted-archive-command.out" ||
  fail "drifted live archive command did not report the protected-command boundary"

drifted_archive_timeout_status="$(python3 - "${healthy_archiver_status}" <<'PY'
import json
import sys

document = json.loads(sys.argv[1])
document["archive_timeout"] = "1d"
print(json.dumps(document, separators=(",", ":")))
PY
)"
if python3 "${WAL_ARCHIVER_VALIDATOR}" \
  --status-json "${drifted_archive_timeout_status}" \
  >"${tmpdir}/drifted-archive-timeout.out" 2>&1; then
  fail "the live WAL archiver validator accepted a drifted archive timeout"
fi
grep -q 'archive_timeout must be 15min' "${tmpdir}/drifted-archive-timeout.out" ||
  fail "drifted live archive timeout did not report the canonical timeout boundary"

newer_archive_failure_status="$(python3 - "${healthy_archiver_status}" <<'PY'
import json
import sys

document = json.loads(sys.argv[1])
document["failed_count"] = 1
document["last_failed_epoch"] = document["last_archived_epoch"]
print(json.dumps(document, separators=(",", ":")))
PY
)"
if python3 "${WAL_ARCHIVER_VALIDATOR}" \
  --status-json "${newer_archive_failure_status}" \
  >"${tmpdir}/newer-archive-failure.out" 2>&1; then
  fail "the live WAL archiver validator accepted a failure not older than its last success"
else
  validator_status=$?
fi
[[ "${validator_status}" -eq 2 ]] ||
  fail "a newer archive failure must request a live probe instead of becoming a hard parse failure"

disabled_archiver_status="$(python3 - <<'PY'
import json

archive_command = '''sh -c 'dest=/var/lib/postgresql/wal-archive/%f; tmp="$dest.tmp.$$"; if [ -f "$dest" ]; then cmp -s %p "$dest"; else cp %p "$tmp" && mv "$tmp" "$dest"; fi\''''
print(json.dumps({
    "archive_mode": "off",
    "archive_command": archive_command,
    "archive_timeout": "15min",
    "archived_count": 0,
    "failed_count": 0,
    "last_archived_wal": None,
    "last_archived_epoch": None,
    "last_failed_epoch": None,
    "postmaster_started_epoch": 100,
}, separators=(",", ":")))
PY
)"
WAL_ARCHIVER_STATUS="${disabled_archiver_status}"
docker() {
  if [[ "$1" == exec && "$2" == contract-postgres && "$*" == *json_build_object* ]]; then
    printf '%s\n' "${WAL_ARCHIVER_STATUS}"
    return 0
  fi
  return 90
}
if (require_live_postgres_wal_archiving contract-postgres contract-admin contract-db) \
  >"${tmpdir}/disabled-wal-archiver.out" 2>"${tmpdir}/disabled-wal-archiver.err"; then
  fail "the live WAL archiver validator accepted archive_mode=off"
fi
grep -q 'WAL archiver settings or status are invalid' "${tmpdir}/disabled-wal-archiver.err" ||
  fail "disabled WAL archiver rejection did not report the live settings boundary"

wal_probe_state="${tmpdir}/wal-probe-state"
WAL_PROBE_STATE="${wal_probe_state}"
WAL_ARCHIVER_STATUS="${healthy_archiver_status}"
WAL_ARCHIVER_UNPROVEN_STATUS="$(python3 - <<'PY'
import json

archive_command = '''sh -c 'dest=/var/lib/postgresql/wal-archive/%f; tmp="$dest.tmp.$$"; if [ -f "$dest" ]; then cmp -s %p "$dest"; else cp %p "$tmp" && mv "$tmp" "$dest"; fi\''''
print(json.dumps({
    "archive_mode": "on",
    "archive_command": archive_command,
    "archive_timeout": "15min",
    "archived_count": 0,
    "failed_count": 0,
    "last_archived_wal": None,
    "last_archived_epoch": None,
    "last_failed_epoch": None,
    "postmaster_started_epoch": 100,
}, separators=(",", ":")))
PY
)"
docker() {
  if [[ "$1" == exec && "$2" == contract-postgres && "$*" == *"SELECT archived_count::text"* ]]; then
    printf '%s\n' 0
    return 0
  fi
  if [[ "$1" == exec && "$2" == contract-postgres && "$*" == *pg_switch_wal* ]]; then
    : >"${WAL_PROBE_STATE}.switched"
    return 0
  fi
  if [[ "$1" == exec && "$2" == contract-postgres && "$*" == *json_build_object* ]]; then
    if [[ -f "${WAL_PROBE_STATE}.switched" ]]; then
      printf '%s\n' "${WAL_ARCHIVER_STATUS}"
    else
      printf '%s\n' "${WAL_ARCHIVER_UNPROVEN_STATUS}"
    fi
    return 0
  fi
  if [[ "$1" == exec && "$2" == contract-postgres && "$3" == /bin/sh ]]; then
    return 0
  fi
  return 90
}
if ! require_live_postgres_wal_archiving contract-postgres contract-admin contract-db; then
  fail "the live WAL archiver validator did not recover by probing a post-start archive"
fi
[[ -f "${wal_probe_state}.switched" ]] ||
  fail "the live WAL archiver validator did not request pg_switch_wal for an unproven server start"

python3 - "${SYNC_BACKUPS}" <<'PY'
import sys
from pathlib import Path

source = Path(sys.argv[1]).read_text(encoding="utf-8")
volume_validation = source.index("require_live_postgres_wal_archive_volume")
archiver_validation = source.index("require_live_postgres_wal_archiving")
external_pitr_validation = source.index("require_external_postgres_pitr_evidence")
first_transfer = source.index("run_backup_object_storage_rclone", volume_validation)
if not volume_validation < archiver_validation < first_transfer:
    raise SystemExit("live WAL mount and archiver validation must precede every backup transfer")
if external_pitr_validation >= first_transfer:
    raise SystemExit("external PITR evidence validation must precede every backup transfer")
PY

external_fixture="${tmpdir}/external-postgres-sync"
mkdir -p \
  "${external_fixture}/infra/ops/lib" \
  "${external_fixture}/backups/postgres/logical" \
  "${external_fixture}/backups/postgres/base"
cp "${SYNC_BACKUPS}" "${external_fixture}/infra/ops/sync-postgres-backups.sh"
chmod +x "${external_fixture}/infra/ops/sync-postgres-backups.sh"
cat >"${external_fixture}/infra/ops/lib/common.sh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
COMMON_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${COMMON_LIB_DIR}/../../.." && pwd)"
require_cmd() { :; }
load_env_preserving() { :; }
require_off_host_backup_object_storage() { :; }
require_live_postgres_wal_archive_volume() {
  printf 'unexpected-local-wal-validation\n' >>"${SYNC_CAPTURE_FILE}"
  return 91
}
require_external_postgres_pitr_evidence() {
  printf 'external-pitr-evidence-verified\n' >>"${SYNC_CAPTURE_FILE}"
}
die() { printf '[fixture][error] %s\n' "$*" >&2; exit 1; }
log() { printf '[fixture] %s\n' "$*"; }
EOF
cat >"${external_fixture}/infra/ops/lib/rclone-object-storage.sh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
run_backup_object_storage_rclone() {
  printf '%s\n' "$*" >>"${SYNC_CAPTURE_FILE}"
}
EOF
external_sync_capture="${external_fixture}/sync-capture"
SYNC_CAPTURE_FILE="${external_sync_capture}" \
EXTERNAL_POSTGRES_ENABLED=true \
APP_ENV=production \
BACKUP_OBJECT_STORAGE_OFF_HOST_REQUIRED=true \
BACKUP_OBJECT_STORAGE_ENDPOINT=https://backup.example.test \
BACKUP_OBJECT_STORAGE_BUCKET=contract-backup \
BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID=contract-key \
BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY=contract-secret \
BACKUP_LOGICAL_DIR="${external_fixture}/backups/postgres/logical" \
BACKUP_BASE_DIR="${external_fixture}/backups/postgres/base" \
  "${external_fixture}/infra/ops/sync-postgres-backups.sh" >"${external_fixture}/sync.out"
assert_contains "${external_sync_capture}" 'target:contract-backup/postgres/logical'
assert_contains "${external_sync_capture}" 'target:contract-backup/postgres/base'
assert_contains "${external_sync_capture}" '^external-pitr-evidence-verified$'
assert_not_contains "${external_sync_capture}" 'target:contract-backup/postgres/wal|unexpected-local-wal-validation'
assert_contains "${external_fixture}/sync.out" 'fresh cluster-bound continuous WAL/PITR evidence was verified'

scheduled_fixture="${tmpdir}/scheduled-fixture"
scheduled_capture="${scheduled_fixture}/capture"
mkdir -p \
  "${scheduled_fixture}/bin" \
  "${scheduled_fixture}/infra/ops/lib" \
  "${scheduled_fixture}/.deploy/releases"
cp "${SCHEDULED_BACKUP}" "${scheduled_fixture}/infra/ops/run-scheduled-backup.sh"
chmod +x "${scheduled_fixture}/infra/ops/run-scheduled-backup.sh"
cat >"${scheduled_fixture}/infra/ops/lib/common.sh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
COMMON_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${COMMON_LIB_DIR}/../../.." && pwd)"
DEPLOY_STATE_DIR="${DEPLOY_STATE_DIR:-${REPO_ROOT}/.deploy}"
load_env_preserving() { :; }
require_cmd() { :; }
require_off_host_backup_object_storage() { printf 'off-host-gate\n' >>"${SCHEDULED_CAPTURE_FILE}"; }
source_release_record_env_file() {
  local file="$1"
  TAG="$(sed -n 's/^TAG=//p' "${file}")"
  [[ -n "${TAG}" ]] || die "missing fixture release tag"
  export TAG
}
log() { printf '[fixture] %s\n' "$*"; }
die() { printf '[fixture][error] %s\n' "$*" >&2; exit 1; }
EOF
cat >"${scheduled_fixture}/infra/ops/sync-postgres-backups.sh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
printf 'sync\n' >>"${SCHEDULED_CAPTURE_FILE}"
EOF
chmod +x "${scheduled_fixture}/infra/ops/sync-postgres-backups.sh"
cat >"${scheduled_fixture}/bin/docker" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

case "${1:-} ${2:-}" in
  "info --format")
    printf '"contract"\n'
    ;;
  "container inspect")
    [[ "${SCHEDULED_FAKE_DATABASE_STATE:-empty}" == "container" ]]
    ;;
  "volume inspect")
    [[ "${SCHEDULED_FAKE_DATABASE_STATE:-empty}" == "volume" ]]
    ;;
  *) exit 1 ;;
esac
EOF
chmod +x "${scheduled_fixture}/bin/docker"
scheduled_path="${scheduled_fixture}/bin:${PATH}"

PATH="${scheduled_path}" \
SCHEDULED_CAPTURE_FILE="${scheduled_capture}" \
BACKUP_OBJECT_STORAGE_OFF_HOST_REQUIRED=true \
  "${scheduled_fixture}/infra/ops/run-scheduled-backup.sh" sync >"${scheduled_fixture}/deferred.out"
assert_contains "${scheduled_fixture}/deferred.out" 'no committed release or datastore evidence exists yet'
[[ ! -e "${scheduled_capture}" ]] ||
  fail "scheduled sync crossed an operational gate before the first committed release"

deploy_lock_ready="${scheduled_fixture}/deploy-lock-ready.fifo"
deploy_lock_release="${scheduled_fixture}/deploy-lock-release.fifo"
mkfifo "${deploy_lock_ready}" "${deploy_lock_release}"
(
  exec {fixture_deploy_lock_fd}<"${scheduled_fixture}/.deploy"
  flock --exclusive "${fixture_deploy_lock_fd}"
  printf 'ready\n' >"${deploy_lock_ready}"
  IFS= read -r _ <"${deploy_lock_release}"
) &
fixture_deploy_lock_pid=$!
IFS= read -r _ <"${deploy_lock_ready}"
PATH="${scheduled_path}" \
SCHEDULED_CAPTURE_FILE="${scheduled_capture}" \
SCHEDULED_FAKE_DATABASE_STATE=container \
STACK_NAME=contract \
  "${scheduled_fixture}/infra/ops/run-scheduled-backup.sh" sync \
    >"${scheduled_fixture}/active-deploy.out"
assert_contains "${scheduled_fixture}/active-deploy.out" 'production deployment holds the release lock'
[[ ! -e "${scheduled_capture}" ]] ||
  fail "scheduled sync crossed an operational gate during an active first deployment"
printf 'release\n' >"${deploy_lock_release}"
wait "${fixture_deploy_lock_pid}"

if PATH="${scheduled_path}" \
  SCHEDULED_CAPTURE_FILE="${scheduled_capture}" \
  EXTERNAL_POSTGRES_ENABLED=true \
  "${scheduled_fixture}/infra/ops/run-scheduled-backup.sh" sync \
  >"${scheduled_fixture}/external-without-marker.out" \
  2>"${scheduled_fixture}/external-without-marker.err"; then
  fail "scheduled sync silently deferred when an external database lacked a release marker"
fi
assert_contains "${scheduled_fixture}/external-without-marker.err" 'external PostgreSQL is selected'

for scheduled_database_state in container volume; do
  if PATH="${scheduled_path}" \
    SCHEDULED_CAPTURE_FILE="${scheduled_capture}" \
    SCHEDULED_FAKE_DATABASE_STATE="${scheduled_database_state}" \
    STACK_NAME=contract \
    "${scheduled_fixture}/infra/ops/run-scheduled-backup.sh" sync \
    >"${scheduled_fixture}/${scheduled_database_state}-without-marker.out" \
    2>"${scheduled_fixture}/${scheduled_database_state}-without-marker.err"; then
    fail "scheduled sync silently deferred when PostgreSQL ${scheduled_database_state} evidence survived"
  fi
  scheduled_database_diagnostic="PostgreSQL container"
  [[ "${scheduled_database_state}" != "volume" ]] || scheduled_database_diagnostic="PostgreSQL data-volume"
  assert_contains \
    "${scheduled_fixture}/${scheduled_database_state}-without-marker.err" \
    "${scheduled_database_diagnostic}"
done

printf '2026-08-03T00:00:00Z\tcontract-release\n' >"${scheduled_fixture}/.deploy/releases.log"
if PATH="${scheduled_path}" \
  SCHEDULED_CAPTURE_FILE="${scheduled_capture}" \
  "${scheduled_fixture}/infra/ops/run-scheduled-backup.sh" sync \
  >"${scheduled_fixture}/log-without-marker.out" \
  2>"${scheduled_fixture}/log-without-marker.err"; then
  fail "scheduled sync silently deferred when release-log evidence survived"
fi
assert_contains "${scheduled_fixture}/log-without-marker.err" 'release-log evidence survives'
rm -f "${scheduled_fixture}/.deploy/releases.log"

printf 'legacy\n' >"${scheduled_fixture}/.deploy/releases/contract-release.env"
if PATH="${scheduled_path}" \
  SCHEDULED_CAPTURE_FILE="${scheduled_capture}" \
  "${scheduled_fixture}/infra/ops/run-scheduled-backup.sh" sync \
  >"${scheduled_fixture}/record-without-marker.out" \
  2>"${scheduled_fixture}/record-without-marker.err"; then
  fail "scheduled sync silently deferred when immutable per-tag evidence survived"
fi
assert_contains "${scheduled_fixture}/record-without-marker.err" 'immutable per-tag evidence survives'
rm -f "${scheduled_fixture}/.deploy/releases/contract-release.env"

cat >"${scheduled_fixture}/.deploy/current-release.env" <<'EOF'
TAG=contract-release
DEPLOYED_AT=2026-08-03T00:00:00Z
BACKEND_IMAGE_REF=backend
FRONTEND_IMAGE_REF=frontend
ADMIN_IMAGE_REF=admin
EOF
cp \
  "${scheduled_fixture}/.deploy/current-release.env" \
  "${scheduled_fixture}/.deploy/releases/contract-release.env"
SCHEDULED_CAPTURE_FILE="${scheduled_capture}" \
BACKUP_OBJECT_STORAGE_OFF_HOST_REQUIRED=true \
  "${scheduled_fixture}/infra/ops/run-scheduled-backup.sh" sync >"${scheduled_fixture}/committed.out"
assert_contains "${scheduled_capture}" '^off-host-gate$'
assert_contains "${scheduled_capture}" '^sync$'

printf '\n' >>"${scheduled_fixture}/.deploy/current-release.env"
if SCHEDULED_CAPTURE_FILE="${scheduled_capture}" \
  BACKUP_OBJECT_STORAGE_OFF_HOST_REQUIRED=true \
  "${scheduled_fixture}/infra/ops/run-scheduled-backup.sh" sync \
  >"${scheduled_fixture}/mismatched.out" 2>"${scheduled_fixture}/mismatched.err"; then
  fail "scheduled sync accepted a release marker that differs from its immutable record"
fi
assert_contains "${scheduled_fixture}/mismatched.err" 'does not match its immutable per-tag record'

if ! python3 - "${FETCH_BACKUPS}" <<'PY'
from pathlib import Path
import sys

source = Path(sys.argv[1]).read_text(encoding="utf-8")
clear = source.index("unset BACKUP_OBJECT_STORAGE_PINNED_HOSTS")
gate = source.index("true) require_off_host_backup_object_storage")
transfer = source.index("run_backup_object_storage_rclone")
if not clear < gate < transfer:
    raise SystemExit("production restore must refresh address pins before rclone")
PY
then
  fail "production restore does not refresh its off-host gate after loading environment files"
fi

if ! python3 - "${SCHEDULED_BACKUP}" <<'PY'
from pathlib import Path
import sys

source = Path(sys.argv[1]).read_text(encoding="utf-8")
release_marker = source.index('current-release.env')
gate = source.index("true) require_off_host_backup_object_storage")
backup = source.index('BACKUP_MODE="${MODE}"')
sync = source.index('"${SCRIPT_DIR}/sync-postgres-backups.sh"', backup)
mutations = (
    'mkdir -p "${backup_dir}"',
    'BACKUP_MODE="${MODE}"',
)
deletions = (
    'prune_old_backups "${backup_dir}"',
    "docker run --rm",
)
if release_marker >= gate:
    raise SystemExit("scheduled work must defer before its off-host gate when no release is committed")
if 'dump | basebackup | sync' not in source:
    raise SystemExit("scheduled wrapper must cover dump, basebackup, and sync timer modes")
for mutation in mutations:
    position = source.index(mutation)
    if gate >= position:
        raise SystemExit(
            f"required off-host gate must precede scheduled-backup mutation: {mutation}"
        )
if backup >= sync:
    raise SystemExit("scheduled backup must be created before remote synchronization")
for deletion in deletions:
    position = source.index(deletion)
    if sync >= position:
        raise SystemExit(
            f"remote synchronization must succeed before local deletion: {deletion}"
        )
PY
then
  fail "scheduled backup gate, synchronization, or retention ordering is unsafe"
fi

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
