#!/usr/bin/env bash
set -euo pipefail

COMMON_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${COMMON_LIB_DIR}/../../.." && pwd)"
ENV_FILE="${ENV_FILE:-${REPO_ROOT}/.env}"
ENV_TEMPLATE_FILE="${ENV_TEMPLATE_FILE:-${REPO_ROOT}/.env.example}"
SECRETS_ENV_FILE="${SECRETS_ENV_FILE:-}"
GENERATED_ENV_FILE="${GENERATED_ENV_FILE:-${REPO_ROOT}/.env.generated}"
if [[ -z "${GENERATED_SECRET_ENV_FILE:-}" ]]; then
  case "${ENV_FILE}" in
    *.env.prod.shared|*.env.prod.example|*/.env.prod.shared|*/.env.prod.example)
      GENERATED_SECRET_ENV_FILE="${REPO_ROOT}/.env.prod.generated.secrets"
      ;;
    *)
      GENERATED_SECRET_ENV_FILE="${REPO_ROOT}/.env.generated.secrets"
      ;;
  esac
fi
SHARED_ENV_SECRET_REF="${SHARED_ENV_SECRET_REF:-}"
SECRETS_ENV_SECRET_REF="${SECRETS_ENV_SECRET_REF:-}"
GENERATED_ENV_SECRET_REF="${GENERATED_ENV_SECRET_REF:-}"
GENERATED_OBS_DIR="${GENERATED_OBS_DIR:-${REPO_ROOT}/infra/generated/observability}"
DEPLOY_STATE_DIR="${DEPLOY_STATE_DIR:-${REPO_ROOT}/.deploy}"
REMOTE_DEPLOY_CONFIG_FILE="${REMOTE_DEPLOY_CONFIG_FILE:-${DEPLOY_STATE_DIR}/remote.env}"
# shellcheck source=secrets.sh
source "${COMMON_LIB_DIR}/secrets.sh"

log() {
  echo "[stuhelper] $*"
}

warn() {
  echo "[stuhelper][warn] $*" >&2
}

die() {
  echo "[stuhelper][error] $*" >&2
  exit 1
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "missing required command: $1"
}

require_systemd_unit_exact_environment() {
  local unit="$1"
  shift
  (($# > 0)) || die "at least one expected environment assignment is required for systemd unit ${unit}"
  local effective_environment
  local protected_property
  local protected_property_value

  for protected_property in EnvironmentFiles UnsetEnvironment PassEnvironment; do
    if ! protected_property_value="$(systemctl show "${unit}" --property="${protected_property}" --value 2>/dev/null)"; then
      die "failed to inspect ${protected_property} for systemd unit ${unit}"
    fi
    if [[ -n "${protected_property_value//[[:space:]]/}" ]]; then
      die "systemd unit ${unit} must not set ${protected_property} for the protected backup gate; reinstall the production backup timers and remove overriding drop-ins"
    fi
  done

  if ! effective_environment="$(systemctl show "${unit}" --property=Environment --value 2>/dev/null)"; then
    die "failed to inspect effective environment for systemd unit ${unit}"
  fi
  if ! python3 - "${effective_environment}" "$@" <<'PY'
import shlex
import sys

effective = sys.argv[1]
expected_assignments = sys.argv[2:]

def assignments_to_environment(assignments):
    environment = {}
    for assignment in assignments:
        if "=" not in assignment:
            raise ValueError
        key, value = assignment.split("=", 1)
        if not key or key in environment:
            raise ValueError
        environment[key] = value
    return environment

try:
    actual = assignments_to_environment(shlex.split(effective))
    expected = assignments_to_environment(expected_assignments)
except ValueError:
    raise SystemExit(1) from None
raise SystemExit(0 if actual == expected else 1)
PY
  then
    die "systemd unit ${unit} must use the exact protected environment; reinstall the production backup timers and remove overriding drop-ins"
  fi
}

require_systemd_unit_hardened_execution() {
  local unit="$1"
  local expected_working_directory="$2"
  local expected_command="$3"
  local effective_exec_start
  local effective_working_directory

  if ! effective_working_directory="$(systemctl show "${unit}" --property=WorkingDirectory --value 2>/dev/null)"; then
    die "failed to inspect WorkingDirectory for systemd unit ${unit}"
  fi
  if ! effective_exec_start="$(systemctl show "${unit}" --property=ExecStart --value 2>/dev/null)"; then
    die "failed to inspect ExecStart for systemd unit ${unit}"
  fi

  if ! python3 - \
    "${expected_working_directory}" \
    "${expected_command}" \
    "${effective_working_directory}" \
    "${effective_exec_start}" <<'PY'
import shlex
import sys

expected_working_directory, expected_command, actual_working_directory, exec_start = sys.argv[1:]
if actual_working_directory != expected_working_directory:
    raise SystemExit(1)

marker = " argv[]="
if marker not in exec_start:
    raise SystemExit(1)
argv_text = exec_start.split(marker, 1)[1].split(" ;", 1)[0]
actual_argv = argv_text.split()
expected_argv = [
    "/usr/bin/env",
    "--unset=BASH_ENV",
    "--unset=ENV",
    "/bin/bash",
    "--noprofile",
    "--norc",
    *shlex.split(expected_command),
]
raise SystemExit(0 if actual_argv == expected_argv else 1)
PY
  then
    die "systemd unit ${unit} must use the protected non-login Bash execution path in ${expected_working_directory}; reinstall the production backup timers"
  fi
}

require_integer_range() {
  local key="$1"
  local value="$2"
  local minimum="$3"
  local maximum="$4"

  if [[ ! "${value}" =~ ^[0-9]+$ ]] ||
    ((10#${value} < minimum || 10#${value} > maximum)); then
    die "${key} must be an integer between ${minimum} and ${maximum}"
  fi
}

source_env_file() {
  local file="$1"
  shift
  [[ -f "${file}" ]] || return 0

  local rendered
  if ! rendered="$(python3 - "${file}" "$@" 2>&1 <<'PY'
import re
import shlex
import sys
from pathlib import Path

path = Path(sys.argv[1])
allowed_keys = set(sys.argv[2:])
key_pattern = re.compile(r"[A-Za-z_][A-Za-z0-9_]*")

for lineno, line in enumerate(path.read_text().splitlines(), 1):
    stripped = line.strip()
    if not stripped or stripped.startswith("#"):
        continue
    if stripped.startswith("export "):
        stripped = stripped[len("export "):].lstrip()
    if "=" not in stripped:
        raise SystemExit(f"{path}:{lineno}: expected KEY=VALUE")
    key, raw_value = stripped.split("=", 1)
    key = key.strip()
    if not key_pattern.fullmatch(key):
        raise SystemExit(f"{path}:{lineno}: invalid env key: {key}")
    if key in {"BASH_ENV", "ENV"}:
        raise SystemExit(
            f"{path}:{lineno}: shell startup variable {key} is not allowed in StuHelper environment files"
        )
    if allowed_keys and key not in allowed_keys:
        raise SystemExit(
            f"{path}:{lineno}: environment key {key} is not allowed in this file"
        )

    raw_value = raw_value.strip()
    if raw_value:
        lexer = shlex.shlex(raw_value, posix=True)
        lexer.whitespace_split = True
        lexer.commenters = ""
        try:
            parts = list(lexer)
        except ValueError as exc:
            raise SystemExit(f"{path}:{lineno}: invalid value for {key}: {exc}") from exc
        value = parts[0] if len(parts) == 1 else raw_value
    else:
        value = ""

    print(f"export {key}={shlex.quote(value)}")
PY
)"; then
    die "${rendered}"
  fi

  # shellcheck disable=SC1091
  source /dev/stdin <<<"${rendered}"
  unset BASH_ENV ENV
}

source_casdoor_bootstrap_env_file() {
  source_env_file \
    "$1" \
    CASDOOR_BOOTSTRAP_CLIENT_ID \
    CASDOOR_BOOTSTRAP_CLIENT_SECRET \
    CASDOOR_BOOTSTRAP_APPLICATION \
    CASDOOR_BOOTSTRAP_CERTIFICATE \
    CASDOOR_BOOTSTRAP_ORGANIZATION
}

source_remote_deploy_config_env_file() {
  source_env_file \
    "$1" \
    REGISTRY \
    REGISTRY_AUTH_MODE \
    REGISTRY_USERNAME_SECRET_REF \
    REGISTRY_PASSWORD_SECRET_REF \
    ENV_FILE \
    SECRETS_ENV_FILE \
    GENERATED_ENV_FILE \
    GENERATED_SECRET_ENV_FILE \
    SECRET_BACKEND \
    SHARED_ENV_SECRET_REF \
    SECRETS_ENV_SECRET_REF \
    GENERATED_ENV_SECRET_REF \
    SECRET_FILE_ROOT \
    VAULT_ADDR \
    VAULT_NAMESPACE \
    VAULT_TOKEN_FILE \
    VAULT_KV_MOUNT \
    VAULT_RUNTIME_TOKEN_POLICY \
    VAULT_RUNTIME_TOKEN_PERIOD_SECONDS \
    VAULT_RUNTIME_TOKEN_MIN_TTL_SECONDS
}

source_release_record_env_file() {
  source_env_file \
    "$1" \
    TAG \
    DEPLOYED_AT \
    BACKEND_IMAGE_REF \
    FRONTEND_IMAGE_REF \
    ADMIN_IMAGE_REF
}

default_local_state_dir() {
  if [[ "$(uname -s)" == "Darwin" ]]; then
    printf '%s/Library/Application Support/StuHelper' "${HOME}"
  else
    printf '%s/stuhelper' "${XDG_STATE_HOME:-${HOME}/.local/state}"
  fi
}

normalize_backup_object_storage_env() {
  if [[ -z "${BACKUP_OBJECT_STORAGE_ENDPOINT:-}" && -n "${OBJECT_STORAGE_ENDPOINT:-}" ]]; then
    export BACKUP_OBJECT_STORAGE_ENDPOINT="${OBJECT_STORAGE_ENDPOINT}"
  fi
  export BACKUP_OBJECT_STORAGE_BUCKET="${BACKUP_OBJECT_STORAGE_BUCKET:-stuhelper-postgres-backup}"
  export BACKUP_OBJECT_STORAGE_PREFIX="${BACKUP_OBJECT_STORAGE_PREFIX:-postgres}"
  if [[ -z "${BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID:-}" && -n "${OBJECT_STORAGE_ACCESS_KEY_ID:-}" ]]; then
    export BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID="${OBJECT_STORAGE_ACCESS_KEY_ID}"
  fi
  if [[ -z "${BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY:-}" && -n "${OBJECT_STORAGE_SECRET_ACCESS_KEY:-}" ]]; then
    export BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY="${OBJECT_STORAGE_SECRET_ACCESS_KEY}"
  fi
  if [[ -z "${BACKUP_OBJECT_STORAGE_TLS_INSECURE:-}" ]]; then
    if [[ "${OBJECT_STORAGE_USE_SSL:-false}" == "true" ]]; then
      export BACKUP_OBJECT_STORAGE_TLS_INSECURE="false"
    else
      export BACKUP_OBJECT_STORAGE_TLS_INSECURE="true"
    fi
  fi
}

materialize_postgres_runtime_urls() {
  local rendered
  rendered="$(python3 - <<'PY'
import os
import shlex
from urllib.parse import quote

mappings = (
    (
        "DATABASE_URL",
        "REPLACE_WITH_STUHELPER_APP_DB_PASSWORD",
        "STUHELPER_APP_DB_PASSWORD",
    ),
    (
        "BACKUP_DATABASE_URL",
        "REPLACE_WITH_STUHELPER_BACKUP_DB_PASSWORD",
        "STUHELPER_BACKUP_DB_PASSWORD",
    ),
    (
        "REPLICATION_DATABASE_URL",
        "REPLACE_WITH_STUHELPER_REPLICATION_DB_PASSWORD",
        "STUHELPER_REPLICATION_DB_PASSWORD",
    ),
)

for url_key, placeholder, secret_key in mappings:
    if url_key not in os.environ:
        continue
    value = os.environ[url_key]
    secret = os.environ.get(secret_key, "")
    # Environment initialization loads the shared template before generating
    # its secret file. Leave the placeholder intact during that first pass;
    # strict production validation rejects it if no later source resolves it.
    if placeholder in value and secret:
        value = value.replace(placeholder, quote(secret, safe=""))
    print(f"export {url_key}={shlex.quote(value)}")
PY
)"
  # shellcheck disable=SC1091
  source /dev/stdin <<<"${rendered}"
}

LOCAL_STATE_DIR="${LOCAL_STATE_DIR:-$(default_local_state_dir)}"
POSTGRES_WAL_RESTORE_DIR="${POSTGRES_WAL_RESTORE_DIR:-${LOCAL_STATE_DIR}/postgres/wal-restore}"

require_backup_object_storage_config() {
  local missing=()
  local key

  normalize_backup_object_storage_env

  for key in \
    BACKUP_OBJECT_STORAGE_ENDPOINT \
    BACKUP_OBJECT_STORAGE_BUCKET \
    BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID \
    BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY; do
    if [[ -z "${!key:-}" ]]; then
      missing+=("${key}")
    fi
  done

  if (( ${#missing[@]} > 0 )); then
    die "backup object storage configuration is incomplete; missing: ${missing[*]}"
  fi
}

require_https_object_storage_endpoint() {
  local key="$1"
  local value="${2:-}"
  local output

  if ! output="$(python3 - "${key}" "${value}" 2>&1 <<'PY'
import sys
from urllib.parse import urlsplit

key, value = sys.argv[1:3]
value = value.strip()
if not value:
    raise SystemExit(f"{key} is required")

parsed = urlsplit(value)
if parsed.scheme.lower() != "https":
    raise SystemExit(f"{key} must use https")
if not parsed.hostname:
    raise SystemExit(f"{key} must include a hostname")
if parsed.username is not None or parsed.password is not None:
    raise SystemExit(f"{key} must not contain embedded credentials")
if parsed.query or parsed.fragment:
    raise SystemExit(f"{key} must not contain query parameters or a fragment")
if parsed.path not in {"", "/"}:
    raise SystemExit(f"{key} must not contain an object path")
PY
  )"; then
    die "${output}"
  fi
}

require_off_host_backup_object_storage() {
  local confirmation="${BACKUP_OBJECT_STORAGE_OFF_HOST_CONFIRMED:-false}"
  local output

  case "${confirmation}" in
    true) ;;
    false|"")
      die "BACKUP_OBJECT_STORAGE_OFF_HOST_CONFIRMED must be true only after the backup target has been verified to survive loss of the production host"
      ;;
    *) die "BACKUP_OBJECT_STORAGE_OFF_HOST_CONFIRMED must be true or false" ;;
  esac

  require_https_object_storage_endpoint \
    "BACKUP_OBJECT_STORAGE_ENDPOINT" \
    "${BACKUP_OBJECT_STORAGE_ENDPOINT:-}"
  unset BACKUP_OBJECT_STORAGE_PINNED_HOSTS
  if ! output="$(python3 - "${BACKUP_OBJECT_STORAGE_ENDPOINT:-}" 2>&1 <<'PY'
import ipaddress
import json
import os
import re
import socket
import subprocess
import sys
from urllib.parse import urlsplit

endpoint = sys.argv[1].strip()
raw_host = (urlsplit(endpoint).hostname or "").lower()
if not raw_host:
    raise SystemExit("BACKUP_OBJECT_STORAGE_ENDPOINT must include a hostname")
if "%" in raw_host:
    raise SystemExit(
        "BACKUP_OBJECT_STORAGE_ENDPOINT must not use an IPv6 zone identifier"
    )
if raw_host.endswith("."):
    raise SystemExit(
        "BACKUP_OBJECT_STORAGE_ENDPOINT must not use a trailing-dot hostname"
    )
host = raw_host

local_identity_spec = os.environ.get(
    "BACKUP_OBJECT_STORAGE_LOCAL_IDENTITY_CIDRS", ""
).strip()
if not local_identity_spec:
    raise SystemExit(
        "BACKUP_OBJECT_STORAGE_LOCAL_IDENTITY_CIDRS is required; list every public/NAT/LB address or CIDR that can route back to this production host, or set none only after verifying no such identity exists"
    )
local_identity_networks = set()
if local_identity_spec.lower() != "none":
    for item in re.split(r"[\s,]+", local_identity_spec):
        if not item:
            continue
        try:
            local_identity_networks.add(ipaddress.ip_network(item, strict=False))
        except ValueError:
            raise SystemExit(
                f"BACKUP_OBJECT_STORAGE_LOCAL_IDENTITY_CIDRS contains an invalid address or CIDR: {item}"
            ) from None
    if not local_identity_networks:
        raise SystemExit(
            "BACKUP_OBJECT_STORAGE_LOCAL_IDENTITY_CIDRS must contain at least one address or CIDR, or be exactly none"
        )

try:
    address = ipaddress.ip_address(host)
except ValueError:
    address = None

if address is None:
    try:
        socket.inet_aton(host)
    except OSError:
        pass
    else:
        raise SystemExit(
            "BACKUP_OBJECT_STORAGE_ENDPOINT must not use a legacy or abbreviated numeric IPv4 address"
        )

if address is None and (
    "." not in host
    or host == "host.docker.internal"
    or host.endswith(".localhost")
    or host.endswith(".local")
):
    raise SystemExit(
        "BACKUP_OBJECT_STORAGE_ENDPOINT must use an off-host fully-qualified hostname or a non-local IP address"
    )
if address is None:
    labels = host.split(".")
    if len(host) > 253 or any(
        not re.fullmatch(r"[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?", label)
        for label in labels
    ):
        raise SystemExit(
            "BACKUP_OBJECT_STORAGE_ENDPOINT must use a valid ASCII DNS hostname"
        )

force_path_style = os.environ.get(
    "BACKUP_OBJECT_STORAGE_FORCE_PATH_STYLE",
    os.environ.get("OBJECT_STORAGE_FORCE_PATH_STYLE", "true"),
)
if force_path_style not in {"true", "false"}:
    raise SystemExit(
        "BACKUP_OBJECT_STORAGE_FORCE_PATH_STYLE must be true or false"
    )
transfer_hosts = [host]
if address is None and force_path_style == "false":
    bucket = os.environ.get("BACKUP_OBJECT_STORAGE_BUCKET", "").strip()
    if not bucket:
        raise SystemExit(
            "BACKUP_OBJECT_STORAGE_BUCKET is required for virtual-hosted backup transfers"
        )
    virtual_host = f"{bucket}.{host}"
    virtual_labels = virtual_host.split(".")
    if len(virtual_host) > 253 or any(
        not re.fullmatch(r"[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?", label)
        for label in virtual_labels
    ):
        raise SystemExit(
            "BACKUP_OBJECT_STORAGE_BUCKET must form a valid lowercase ASCII virtual-hosted S3 hostname"
        )
    transfer_hosts.append(virtual_host)

image_ref = os.environ.get(
    "RCLONE_IMAGE_REF",
    "rclone/rclone:beta@sha256:f52965eba611ba8984117638b2a0539dcce170731937f93fbace66897d102698",
)
if not re.fullmatch(r".+@sha256:[0-9a-f]{64}", image_ref):
    raise SystemExit("RCLONE_IMAGE_REF must be a complete image@sha256 reference")
docker_network = os.environ.get("BACKUP_OBJECT_STORAGE_DOCKER_NETWORK", "")
if docker_network and not re.fullmatch(r"[A-Za-z0-9_.-]+", docker_network):
    raise SystemExit(
        "BACKUP_OBJECT_STORAGE_DOCKER_NETWORK contains unsupported characters"
    )
if docker_network in {"host", "none"}:
    raise SystemExit(
        "BACKUP_OBJECT_STORAGE_DOCKER_NETWORK must not use host or none for off-host production backups"
    )
effective_docker_network = docker_network or "bridge"

if address is not None:
    resolved_addresses_by_host = {host: {address}}
else:
    resolved_addresses_by_host = {}
    resolver_command = [
        "docker",
        "run",
        "--rm",
        "--read-only",
        "--cap-drop",
        "ALL",
        "--security-opt",
        "no-new-privileges",
    ]
    if docker_network:
        resolver_command.extend(["--network", docker_network])
    resolver_command.extend(["--entrypoint", "/usr/bin/getent", image_ref])
    for transfer_host in transfer_hosts:
        resolved_addresses = set()
        for database in ("ahostsv4", "ahostsv6"):
            try:
                result = subprocess.run(
                    [*resolver_command, database, transfer_host],
                    check=False,
                    capture_output=True,
                    text=True,
                    timeout=30,
                )
            except FileNotFoundError:
                raise SystemExit(
                    "docker is required to resolve BACKUP_OBJECT_STORAGE_ENDPOINT in the rclone network namespace"
                ) from None
            except subprocess.TimeoutExpired:
                raise SystemExit(
                    f"backup transfer hostname {transfer_host} resolution timed out in the rclone network namespace"
                ) from None
            for line in result.stdout.splitlines():
                fields = line.split()
                if not fields:
                    continue
                candidate = fields[0].split("%", 1)[0]
                try:
                    resolved_addresses.add(ipaddress.ip_address(candidate))
                except ValueError:
                    continue
        if not resolved_addresses:
            raise SystemExit(
                f"backup transfer hostname {transfer_host} must resolve to at least one A or AAAA address in the rclone network namespace"
            )
        resolved_addresses_by_host[transfer_host] = resolved_addresses

try:
    docker_network_result = subprocess.run(
        ["docker", "network", "inspect", effective_docker_network],
        check=True,
        capture_output=True,
        text=True,
        timeout=10,
    )
    docker_networks = json.loads(docker_network_result.stdout)
except FileNotFoundError:
    raise SystemExit(
        "docker is required to inspect the rclone network while verifying BACKUP_OBJECT_STORAGE_ENDPOINT"
    ) from None
except subprocess.TimeoutExpired:
    raise SystemExit(
        "rclone Docker network inspection timed out while verifying BACKUP_OBJECT_STORAGE_ENDPOINT"
    ) from None
except (subprocess.CalledProcessError, json.JSONDecodeError) as error:
    raise SystemExit(
        f"failed to inspect the rclone Docker network while verifying BACKUP_OBJECT_STORAGE_ENDPOINT: {error}"
    ) from None

local_docker_subnets = set()
local_docker_addresses = set()
for network in docker_networks:
    driver = str(network.get("Driver", ""))
    if driver not in {"bridge", "macvlan", "ipvlan", "overlay"}:
        raise SystemExit(
            f"unsupported rclone Docker network driver for off-host verification: {driver or 'unknown'}"
        )
    if driver == "bridge":
        for config in network.get("IPAM", {}).get("Config", []) or []:
            subnet = str(config.get("Subnet", "")).split("%", 1)[0]
            try:
                local_docker_subnets.add(ipaddress.ip_network(subnet, strict=False))
            except ValueError:
                continue
    for container in (network.get("Containers") or {}).values():
        for key in ("IPv4Address", "IPv6Address"):
            candidate = str(container.get(key, "")).split("/", 1)[0].split("%", 1)[0]
            try:
                local_docker_addresses.add(ipaddress.ip_address(candidate))
            except ValueError:
                continue

try:
    local_result = subprocess.run(
        ["ip", "-j", "address", "show"],
        check=True,
        capture_output=True,
        text=True,
    )
    local_interfaces = json.loads(local_result.stdout)
except FileNotFoundError:
    raise SystemExit(
        "iproute2 is required to verify that BACKUP_OBJECT_STORAGE_ENDPOINT is off-host"
    ) from None
except (subprocess.CalledProcessError, json.JSONDecodeError) as error:
    raise SystemExit(
        f"failed to enumerate local addresses while verifying BACKUP_OBJECT_STORAGE_ENDPOINT: {error}"
    ) from None

local_addresses = set()
for interface in local_interfaces:
    for info in interface.get("addr_info", []):
        candidate = str(info.get("local", "")).split("%", 1)[0]
        try:
            local_addresses.add(ipaddress.ip_address(candidate))
        except ValueError:
            continue

def normalize(value):
    if isinstance(value, ipaddress.IPv6Address) and value.ipv4_mapped is not None:
        return value.ipv4_mapped
    return value

def equivalent_addresses(value):
    normalized = normalize(value)
    values = {value, normalized}
    if isinstance(normalized, ipaddress.IPv4Address):
        values.add(ipaddress.IPv6Address(f"::ffff:{normalized}"))
    return values

normalized_local_addresses = {normalize(value) for value in local_addresses}
normalized_local_docker_addresses = {
    normalize(value) for value in local_docker_addresses
}
for resolved_address in {
    value
    for values in resolved_addresses_by_host.values()
    for value in values
}:
    normalized_address = normalize(resolved_address)
    comparable_addresses = equivalent_addresses(resolved_address)
    if (
        normalized_address.is_loopback
        or normalized_address.is_unspecified
        or normalized_address.is_link_local
        or normalized_address.is_multicast
    ):
        raise SystemExit(
            "BACKUP_OBJECT_STORAGE_ENDPOINT must not resolve to a loopback, unspecified, link-local, or multicast address"
        )
    if normalized_address in normalized_local_addresses:
        raise SystemExit(
            "BACKUP_OBJECT_STORAGE_ENDPOINT must not resolve to an address assigned to the production host"
        )
    if any(
        candidate.version == identity.version and candidate in identity
        for identity in local_identity_networks
        for candidate in comparable_addresses
    ):
        raise SystemExit(
            "BACKUP_OBJECT_STORAGE_ENDPOINT must not resolve to a configured public/NAT/LB identity of the production host"
        )
    if normalized_address in normalized_local_docker_addresses or any(
        normalized_address.version == subnet.version and normalized_address in subnet
        for subnet in local_docker_subnets
    ):
        raise SystemExit(
            "BACKUP_OBJECT_STORAGE_ENDPOINT must not resolve into a Docker network hosted on the production host"
        )

if address is None:
    for transfer_host in transfer_hosts:
        pinned_addresses = {
            normalize(value)
            for value in resolved_addresses_by_host[transfer_host]
        }
        for pinned_address in sorted(
            pinned_addresses, key=lambda value: (value.version, int(value))
        ):
            print(f"{transfer_host}={pinned_address.compressed}")
PY
  )"; then
    die "${output}"
  fi
  export BACKUP_OBJECT_STORAGE_PINNED_HOSTS="${output}"
}

require_production_object_storage() {
  require_backup_object_storage_config
  require_https_object_storage_endpoint "OBJECT_STORAGE_ENDPOINT" "${OBJECT_STORAGE_ENDPOINT:-}"
  require_https_object_storage_endpoint "BACKUP_OBJECT_STORAGE_ENDPOINT" "${BACKUP_OBJECT_STORAGE_ENDPOINT:-}"
  require_off_host_backup_object_storage

  [[ "${OBJECT_STORAGE_USE_SSL:-false}" == "true" ]] ||
    die "OBJECT_STORAGE_USE_SSL must be true for production"
  [[ "${BACKUP_OBJECT_STORAGE_TLS_INSECURE:-false}" == "false" ]] ||
    die "BACKUP_OBJECT_STORAGE_TLS_INSECURE must be false for production"

  case "${OBJECT_STORAGE_FORCE_PATH_STYLE:-false}" in
    true|false) ;;
    *) die "OBJECT_STORAGE_FORCE_PATH_STYLE must be true or false" ;;
  esac
  case "${BACKUP_OBJECT_STORAGE_FORCE_PATH_STYLE:-${OBJECT_STORAGE_FORCE_PATH_STYLE:-false}}" in
    true|false) ;;
    *) die "BACKUP_OBJECT_STORAGE_FORCE_PATH_STYLE must be true or false" ;;
  esac

  if [[ -n "${OBJECT_STORAGE_TLS_CA:-}" ]]; then
    [[ "${OBJECT_STORAGE_TLS_CA}" == "/object-storage-tls/ca.crt" ]] ||
      die "OBJECT_STORAGE_TLS_CA must be /object-storage-tls/ca.crt"
    [[ -n "${OBJECT_STORAGE_TLS_CA_HOST_PATH:-}" ]] ||
      die "OBJECT_STORAGE_TLS_CA_HOST_PATH is required when OBJECT_STORAGE_TLS_CA is configured"
    [[ -f "${OBJECT_STORAGE_TLS_CA_HOST_PATH}" &&
       -r "${OBJECT_STORAGE_TLS_CA_HOST_PATH}" ]] ||
      die "OBJECT_STORAGE_TLS_CA_HOST_PATH must be a readable regular file"
  elif [[ -n "${OBJECT_STORAGE_TLS_CA_HOST_PATH:-}" ]]; then
    die "OBJECT_STORAGE_TLS_CA_HOST_PATH must be empty when OBJECT_STORAGE_TLS_CA is empty"
  fi

  if [[ -n "${BACKUP_OBJECT_STORAGE_TLS_CA:-}" ]]; then
    [[ -f "${BACKUP_OBJECT_STORAGE_TLS_CA}" && -r "${BACKUP_OBJECT_STORAGE_TLS_CA}" ]] ||
      die "BACKUP_OBJECT_STORAGE_TLS_CA must be a readable regular file"
  fi

  if [[ "${OBJECT_STORAGE_ACCESS_KEY_ID:-}" == "${BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID:-}" &&
        "${OBJECT_STORAGE_SECRET_ACCESS_KEY:-}" != "${BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY:-}" ]]; then
    die "the same object-storage access key cannot map to different secrets"
  fi
  if [[ "${OBJECT_STORAGE_ACCESS_KEY_ID:-}" != "${BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID:-}" &&
        "${OBJECT_STORAGE_SECRET_ACCESS_KEY:-}" == "${BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY:-}" ]]; then
    die "application and backup object-storage identities must not share a secret"
  fi
}

verified_postgres_ssl_mode() {
  case "${1:-}" in
    verify-ca|verify-full) return 0 ;;
    *) return 1 ;;
  esac
}

require_verified_postgres_ssl_mode() {
  local key="$1"
  local value="${2:-}"
  if ! verified_postgres_ssl_mode "${value}"; then
    die "${key} must be verify-ca or verify-full for production PostgreSQL TLS (got ${value:-<empty>})"
  fi
}

require_production_postgres_url() {
  local key="$1"
  local value="${2:-}"
  local output

  if ! output="$(python3 - "${key}" "${value}" 2>&1 <<'PY'
import sys
from urllib.parse import parse_qs, urlsplit

key = sys.argv[1]
value = sys.argv[2].strip()

if not value:
    raise SystemExit(f"{key} must be configured for production PostgreSQL")
if "REPLACE_WITH_" in value:
    raise SystemExit(f"{key} contains an unresolved secret placeholder")

parsed = urlsplit(value)
if parsed.scheme not in {"postgres", "postgresql"}:
    raise SystemExit(f"{key} must be a postgres/postgresql URL")

host = (parsed.hostname or "").lower()
if host in {"localhost", "127.0.0.1", "::1"}:
    raise SystemExit(f"{key} must not point to a local/development PostgreSQL endpoint ({host})")

query = parse_qs(parsed.query, keep_blank_values=True)
ssl_modes = query.get("sslmode", [])
ssl_mode = ssl_modes[0] if ssl_modes else ""
if ssl_mode not in {"verify-ca", "verify-full"}:
    raise SystemExit(
        f"{key} must include sslmode=verify-ca or sslmode=verify-full for production (got {ssl_mode or '<missing>'})"
    )

root_certs = query.get("sslrootcert", [])
if not root_certs or not root_certs[0].strip():
    raise SystemExit(f"{key} must include sslrootcert for production PostgreSQL TLS")
PY
)"; then
    die "${output}"
  fi
}

require_production_postgres_ssl() {
  [[ "${EXTERNAL_POSTGRES_ALLOW_PLAINTEXT:-false}" != "true" ]] ||
    die "EXTERNAL_POSTGRES_ALLOW_PLAINTEXT is only allowed in prod-parity; production PostgreSQL must use verified TLS"

  if [[ "${EXTERNAL_POSTGRES_ENABLED:-false}" == "true" ]]; then
    require_verified_postgres_ssl_mode "POSTGRES_INTERNAL_SSL_MODE" "${POSTGRES_INTERNAL_SSL_MODE:-}"
    [[ "${DB_SSL_MODE:-}" == "verify-full" ]] || die "DB_SSL_MODE must be verify-full for external production PostgreSQL"
    [[ "${DB_SSL_ROOT_CERT:-}" == "/tls/ca.crt" ]] ||
      die "DB_SSL_ROOT_CERT must be /tls/ca.crt for external production PostgreSQL"
    [[ -n "${POSTGRES_CLIENT_CA_HOST_PATH:-}" ]] ||
      die "POSTGRES_CLIENT_CA_HOST_PATH is required for external production PostgreSQL TLS"
  else
    [[ "${POSTGRES_ENABLE_SSL:-}" == "on" ]] || die "POSTGRES_ENABLE_SSL must be on for production"
    require_verified_postgres_ssl_mode "POSTGRES_INTERNAL_SSL_MODE" "${POSTGRES_INTERNAL_SSL_MODE:-}"
    [[ "${DB_SSL_MODE:-}" == "verify-full" ]] || die "DB_SSL_MODE must be verify-full for production"
    [[ "${DB_SSL_ROOT_CERT:-}" == "/tls/ca.crt" ]] ||
      die "DB_SSL_ROOT_CERT must be /tls/ca.crt for production"
  fi

  require_production_postgres_url "DATABASE_URL" "${DATABASE_URL:-}"
  require_production_postgres_url "BACKUP_DATABASE_URL" "${BACKUP_DATABASE_URL:-}"
  require_production_postgres_url "REPLICATION_DATABASE_URL" "${REPLICATION_DATABASE_URL:-}"
}

require_production_external_student_source_security() {
  [[ "${EXTERNAL_STUDENT_SOURCE_ENABLED:-false}" == "true" ]] || return 0
  [[ "${EXTERNAL_STUDENT_SOURCE_PROVIDER:-}" == "oracle" ]] ||
    die "EXTERNAL_STUDENT_SOURCE_PROVIDER must be oracle when the external student source is enabled"

  local key
  local value
  for key in \
    EXTERNAL_STUDENT_SOURCE_SCHOOL_CODE \
    EXTERNAL_STUDENT_SOURCE_ORACLE_HOST \
    EXTERNAL_STUDENT_SOURCE_ORACLE_SERVICE_NAME \
    EXTERNAL_STUDENT_SOURCE_ORACLE_USERNAME \
    EXTERNAL_STUDENT_SOURCE_ORACLE_READONLY_USERNAME \
    EXTERNAL_STUDENT_SOURCE_ORACLE_PASSWORD \
    EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_CA_HOST_PATH \
    EXTERNAL_STUDENT_SOURCE_ORACLE_SCHEMA \
    EXTERNAL_STUDENT_SOURCE_ORACLE_TABLE \
    EXTERNAL_STUDENT_SOURCE_ORACLE_STUDENT_ID_COLUMN \
    EXTERNAL_STUDENT_SOURCE_ORACLE_STUDENT_NAME_COLUMN; do
    value="${!key:-}"
    [[ -n "${value}" ]] || die "${key} is required when the external student source is enabled"
    case "${value}" in
      REPLACE_WITH_* | RUN_*)
        die "${key} contains an unresolved placeholder"
        ;;
    esac
  done

  [[ "${EXTERNAL_STUDENT_SOURCE_SCHOOL_CODE}" =~ ^[0-9]{10}$ ]] ||
    die "EXTERNAL_STUDENT_SOURCE_SCHOOL_CODE must be a 10-digit school code"
  for key in \
    EXTERNAL_STUDENT_SOURCE_ORACLE_USERNAME \
    EXTERNAL_STUDENT_SOURCE_ORACLE_READONLY_USERNAME \
    EXTERNAL_STUDENT_SOURCE_ORACLE_SCHEMA \
    EXTERNAL_STUDENT_SOURCE_ORACLE_TABLE \
    EXTERNAL_STUDENT_SOURCE_ORACLE_STUDENT_ID_COLUMN \
    EXTERNAL_STUDENT_SOURCE_ORACLE_STUDENT_NAME_COLUMN; do
    value="${!key}"
    [[ "${value}" =~ ^[A-Za-z][A-Za-z0-9_]{0,127}$ ]] ||
      die "${key} must be a safe Oracle identifier"
  done
  [[ "${EXTERNAL_STUDENT_SOURCE_ORACLE_USERNAME^^}" == "${EXTERNAL_STUDENT_SOURCE_ORACLE_READONLY_USERNAME^^}" ]] ||
    die "EXTERNAL_STUDENT_SOURCE_ORACLE_USERNAME must match EXTERNAL_STUDENT_SOURCE_ORACLE_READONLY_USERNAME"
  [[ "${EXTERNAL_STUDENT_SOURCE_ORACLE_USERNAME^^}" != "${EXTERNAL_STUDENT_SOURCE_ORACLE_SCHEMA^^}" ]] ||
    die "EXTERNAL_STUDENT_SOURCE_ORACLE_USERNAME must not own the source schema"

  [[ "${EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_MODE:-}" == "verify-full" ]] ||
    die "EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_MODE must be verify-full in production"
  [[ "${EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_CA_FILE:-}" == "/external-student-source-tls/ca.crt" ]] ||
    die "EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_CA_FILE must be /external-student-source-tls/ca.crt in production"

  require_integer_range "EXTERNAL_STUDENT_SOURCE_ORACLE_PORT" "${EXTERNAL_STUDENT_SOURCE_ORACLE_PORT:-}" 1 65535
  require_integer_range "EXTERNAL_STUDENT_SOURCE_ORACLE_CONNECT_TIMEOUT_SECONDS" "${EXTERNAL_STUDENT_SOURCE_ORACLE_CONNECT_TIMEOUT_SECONDS:-}" 1 60
  require_integer_range "EXTERNAL_STUDENT_SOURCE_ORACLE_QUERY_TIMEOUT_SECONDS" "${EXTERNAL_STUDENT_SOURCE_ORACLE_QUERY_TIMEOUT_SECONDS:-}" 1 60
  require_integer_range "EXTERNAL_STUDENT_SOURCE_ORACLE_MAX_OPEN_CONNS" "${EXTERNAL_STUDENT_SOURCE_ORACLE_MAX_OPEN_CONNS:-}" 1 100
  require_integer_range "EXTERNAL_STUDENT_SOURCE_ORACLE_MAX_IDLE_CONNS" "${EXTERNAL_STUDENT_SOURCE_ORACLE_MAX_IDLE_CONNS:-}" 0 "${EXTERNAL_STUDENT_SOURCE_ORACLE_MAX_OPEN_CONNS}"
  require_integer_range "EXTERNAL_STUDENT_SOURCE_ORACLE_CONN_MAX_LIFETIME_SECONDS" "${EXTERNAL_STUDENT_SOURCE_ORACLE_CONN_MAX_LIFETIME_SECONDS:-}" 30 3600
  require_integer_range "EXTERNAL_STUDENT_SOURCE_ORACLE_CONN_MAX_IDLE_TIME_SECONDS" "${EXTERNAL_STUDENT_SOURCE_ORACLE_CONN_MAX_IDLE_TIME_SECONDS:-}" 30 "${EXTERNAL_STUDENT_SOURCE_ORACLE_CONN_MAX_LIFETIME_SECONDS}"
  require_integer_range "EXTERNAL_STUDENT_SOURCE_ORACLE_BREAKER_FAILURE_THRESHOLD" "${EXTERNAL_STUDENT_SOURCE_ORACLE_BREAKER_FAILURE_THRESHOLD:-}" 1 100
  require_integer_range "EXTERNAL_STUDENT_SOURCE_ORACLE_BREAKER_SUCCESS_THRESHOLD" "${EXTERNAL_STUDENT_SOURCE_ORACLE_BREAKER_SUCCESS_THRESHOLD:-}" 1 20
  require_integer_range "EXTERNAL_STUDENT_SOURCE_ORACLE_BREAKER_OPEN_SECONDS" "${EXTERNAL_STUDENT_SOURCE_ORACLE_BREAKER_OPEN_SECONDS:-}" 1 600
}

trim_trailing_slash() {
  local value="$1"
  printf '%s\n' "${value%/}"
}

repo_default_path_matches() {
  local current="$1"
  local common_default="$2"
  local normalized

  if [[ -z "${current}" ]]; then
    return 0
  fi
  if [[ -z "${common_default}" ]]; then
    return 1
  fi
  if [[ "${current}" == "${common_default}" ]]; then
    return 0
  fi

  case "${current}" in
    /*) normalized="${current}" ;;
    *) normalized="${REPO_ROOT}/${current#./}" ;;
  esac

  [[ "${normalized}" == "${common_default}" ]]
}

_public_ingress_body_snippet() {
  python3 - "$1" <<'PY'
from pathlib import Path
import sys

path = Path(sys.argv[1])
if not path.exists():
    print("")
    raise SystemExit
raw = path.read_bytes()[:240]
text = raw.decode("utf-8", "replace").replace("\r", " ").replace("\n", " ")
print(" ".join(text.split()))
PY
}

_public_ingress_host() {
  python3 - "$1" <<'PY'
import sys
from urllib.parse import urlsplit

value = sys.argv[1].strip()
parsed = urlsplit(value)
print(parsed.hostname or "")
PY
}

require_public_dns_resolved() {
  local name="$1"
  local url="$2"
  local timeout="${PUBLIC_INGRESS_PREFLIGHT_TIMEOUT_SECONDS:-10}"
  local enabled="${PUBLIC_INGRESS_PUBLIC_DNS_ENABLED:-true}"
  local host
  local a_file
  local aaaa_file
  local error_file

  case "${enabled}" in
    true|TRUE|1|yes|YES) ;;
    false|FALSE|0|no|NO|"")
      warn "${name} public DNS preflight skipped because PUBLIC_INGRESS_PUBLIC_DNS_ENABLED is not true"
      return 0
      ;;
    *) die "PUBLIC_INGRESS_PUBLIC_DNS_ENABLED must be true or false" ;;
  esac

  host="$(_public_ingress_host "${url}")"
  [[ -n "${host}" ]] || die "${name} public DNS preflight requires a URL with hostname: ${url:-<empty>}"

  local host_kind
  if ! host_kind="$(python3 - "${name}" "${host}" 2>&1 <<'PY'
import ipaddress
import sys

name, host = sys.argv[1:3]
try:
    literal = ipaddress.ip_address(host.strip("[]").split("%", 1)[0])
except ValueError:
    print("domain")
else:
    if not literal.is_global:
        raise SystemExit(f"{name} public DNS preflight resolved to non-public IP literal for {host}")
    print("global_ip")
PY
  )"; then
    die "${host_kind}"
  fi
  if [[ "${host_kind}" == "global_ip" ]]; then
    log "${name} public DNS ready: ${host}"
    return 0
  fi

  a_file="$(mktemp)"
  aaaa_file="$(mktemp)"
  error_file="$(mktemp)"
  if ! curl -fsS --max-time "${timeout}" -o "${a_file}" "https://dns.google/resolve?name=${host}&type=A" 2>"${error_file}"; then
    local curl_error
    curl_error="$(_public_ingress_body_snippet "${error_file}")"
    rm -f "${a_file}" "${aaaa_file}" "${error_file}"
    die "${name} public DNS preflight failed for ${host} A: ${curl_error:-curl failed}"
  fi
  if ! curl -fsS --max-time "${timeout}" -o "${aaaa_file}" "https://dns.google/resolve?name=${host}&type=AAAA" 2>"${error_file}"; then
    local curl_error
    curl_error="$(_public_ingress_body_snippet "${error_file}")"
    rm -f "${a_file}" "${aaaa_file}" "${error_file}"
    die "${name} public DNS preflight failed for ${host} AAAA: ${curl_error:-curl failed}"
  fi
  rm -f "${error_file}"

  local dns_validation_output
  if ! dns_validation_output="$(python3 - "${name}" "${host}" "${a_file}" "${aaaa_file}" 2>&1 <<'PY'
import ipaddress
import json
import sys
from pathlib import Path

name, host, a_path, aaaa_path = sys.argv[1:5]

def fail(message: str) -> None:
    raise SystemExit(message)

def load(path: str) -> dict:
    try:
        payload = json.loads(Path(path).read_text())
    except json.JSONDecodeError as exc:
        fail(f"{name} public DNS preflight got invalid JSON for {host}: {exc}")
    if not isinstance(payload, dict):
        fail(f"{name} public DNS preflight got non-object JSON for {host}")
    return payload

def is_global(value: str) -> bool:
    try:
        return ipaddress.ip_address(value.split("%", 1)[0]).is_global
    except ValueError:
        return False

def ip_literal(value: str):
    try:
        return ipaddress.ip_address(value.strip("[]").split("%", 1)[0])
    except ValueError:
        return None

literal = ip_literal(host)
if literal is not None:
    if not literal.is_global:
        fail(f"{name} public DNS preflight resolved to non-public IP literal for {host}")
    raise SystemExit(0)

payloads = {"A": load(a_path), "AAAA": load(aaaa_path)}
addresses: list[str] = []
statuses: dict[str, object] = {}
for rrtype, rrnumber in (("A", 1), ("AAAA", 28)):
    payload = payloads[rrtype]
    statuses[rrtype] = payload.get("Status")
    for answer in payload.get("Answer") or []:
        if isinstance(answer, dict) and answer.get("type") == rrnumber and isinstance(answer.get("data"), str):
            addresses.append(answer["data"])

addresses = sorted(set(addresses))
if not addresses:
    if any(status == 3 for status in statuses.values()):
        fail(f"{name} public DNS preflight failed for {host}: NXDOMAIN from public resolver")
    fail(f"{name} public DNS preflight failed for {host}: no public A/AAAA records from public resolver")

non_public = [address for address in addresses if not is_global(address)]
if non_public:
    fail(f"{name} public DNS preflight failed for {host}: non-public A/AAAA records: {', '.join(non_public)}")
PY
  )"; then
    rm -f "${a_file}" "${aaaa_file}"
    die "${dns_validation_output}"
  fi
  rm -f "${a_file}" "${aaaa_file}"
  log "${name} public DNS ready: ${host}"
}

require_public_http_reachable() {
  local name="$1"
  local url="$2"
  local timeout="${PUBLIC_INGRESS_PREFLIGHT_TIMEOUT_SECONDS:-10}"
  local error_file
  local status
  local curl_error

  [[ -n "${url}" ]] || die "${name} URL is required for public ingress preflight"
  error_file="$(mktemp)"
  if ! status="$(curl -sS --max-time "${timeout}" -o /dev/null -w '%{http_code}' "${url}" 2>"${error_file}")"; then
    curl_error="$(_public_ingress_body_snippet "${error_file}")"
    rm -f "${error_file}"
    die "${name} public TLS/HTTP preflight failed for ${url}: ${curl_error:-curl failed}"
  fi
  rm -f "${error_file}"
  if [[ ! "${status}" =~ ^[0-9][0-9][0-9]$ ]] || [[ "${status}" == "000" ]]; then
    die "${name} public TLS/HTTP preflight returned invalid HTTP status ${status:-<empty>} for ${url}"
  fi
  log "${name} public ingress TLS reachable: ${url} (HTTP ${status})"
}

require_public_oidc_discovery() {
  local name="$1"
  local issuer
  issuer="$(trim_trailing_slash "$2")"
  local timeout="${PUBLIC_INGRESS_PREFLIGHT_TIMEOUT_SECONDS:-10}"
  local url="${issuer}/.well-known/openid-configuration"
  local output_file
  local error_file
  local status
  local curl_error
  local body

  [[ -n "${issuer}" ]] || die "${name} issuer is required for public OIDC discovery preflight"
  output_file="$(mktemp)"
  error_file="$(mktemp)"
  if ! status="$(curl -sS --max-time "${timeout}" -o "${output_file}" -w '%{http_code}' "${url}" 2>"${error_file}")"; then
    curl_error="$(_public_ingress_body_snippet "${error_file}")"
    rm -f "${output_file}" "${error_file}"
    die "${name} discovery preflight failed for ${url}: ${curl_error:-curl failed}"
  fi
  curl_error="$(_public_ingress_body_snippet "${error_file}")"
  rm -f "${error_file}"
  if [[ ! "${status}" =~ ^[0-9][0-9][0-9]$ ]] || (( status < 200 || status >= 300 )); then
    body="$(_public_ingress_body_snippet "${output_file}")"
    rm -f "${output_file}"
    die "${name} discovery preflight returned HTTP ${status} for ${url}: ${body:-${curl_error:-<empty body>}}"
  fi
  if ! jq -e --arg issuer "${issuer}" '
    type == "object"
    and .issuer == $issuer
    and (.authorization_endpoint | type == "string" and length > 0)
    and (.token_endpoint | type == "string" and length > 0)
    and (.jwks_uri | type == "string" and length > 0)
  ' "${output_file}" >/dev/null; then
    body="$(_public_ingress_body_snippet "${output_file}")"
    rm -f "${output_file}"
    die "${name} discovery metadata is invalid for ${url}: ${body:-<empty body>}"
  fi
  rm -f "${output_file}"
  log "${name} public OIDC discovery ready: ${url}"
}

require_public_jwks() {
  local name="$1"
  local url="$2"
  local timeout="${PUBLIC_INGRESS_PREFLIGHT_TIMEOUT_SECONDS:-10}"
  local output_file
  local error_file
  local status
  local curl_error
  local body

  [[ -n "${url}" ]] || die "${name} JWKS URL is required for public identity ingress preflight"
  output_file="$(mktemp)"
  error_file="$(mktemp)"
  if ! status="$(curl -sS --max-time "${timeout}" -o "${output_file}" -w '%{http_code}' "${url}" 2>"${error_file}")"; then
    curl_error="$(_public_ingress_body_snippet "${error_file}")"
    rm -f "${output_file}" "${error_file}"
    die "${name} JWKS preflight failed for ${url}: ${curl_error:-curl failed}"
  fi
  curl_error="$(_public_ingress_body_snippet "${error_file}")"
  rm -f "${error_file}"
  if [[ ! "${status}" =~ ^[0-9][0-9][0-9]$ ]] || (( status < 200 || status >= 300 )); then
    body="$(_public_ingress_body_snippet "${output_file}")"
    rm -f "${output_file}"
    die "${name} JWKS preflight returned HTTP ${status} for ${url}: ${body:-${curl_error:-<empty body>}}"
  fi
  if ! jq -e 'type == "object" and (.keys | type == "array")' "${output_file}" >/dev/null; then
    body="$(_public_ingress_body_snippet "${output_file}")"
    rm -f "${output_file}"
    die "${name} JWKS metadata is invalid for ${url}: ${body:-<empty body>}"
  fi
  rm -f "${output_file}"
  log "${name} public JWKS ready: ${url}"
}

public_oidc_jwks_uri() {
  local issuer
  issuer="$(trim_trailing_slash "$1")"
  local timeout="${PUBLIC_INGRESS_PREFLIGHT_TIMEOUT_SECONDS:-10}"
  local metadata

  [[ -n "${issuer}" ]] || return 1
  if ! metadata="$(curl -fsS --max-time "${timeout}" "${issuer}/.well-known/openid-configuration")"; then
    return 1
  fi
  printf '%s\n' "${metadata}" | jq -r '.jwks_uri // empty'
}

require_public_identity_ingress_preflight() {
  if [[ "${PUBLIC_INGRESS_PREFLIGHT_ENABLED:-true}" != "true" ]]; then
    warn "public SSO/admission ingress preflight skipped because PUBLIC_INGRESS_PREFLIGHT_ENABLED is not true"
    return 0
  fi

  local admission_public_base_url
  admission_public_base_url="$(trim_trailing_slash "${ADMISSION_PUBLIC_BASE_URL:-}")"
  local sso_public_base_url
  sso_public_base_url="$(trim_trailing_slash "${CASDOOR_PUBLIC_AUTH_BASE_URL:-${WEB_VITE_SSO_URL:-${CASDOOR_ISSUER:-}}}")"

  require_public_dns_resolved "Web" "$(trim_trailing_slash "${WEB_PUBLIC_URL:-}")"
  require_public_dns_resolved "SSO" "${sso_public_base_url}"
  require_public_dns_resolved "Admission" "${admission_public_base_url}"
  require_public_http_reachable "Web" "$(trim_trailing_slash "${WEB_PUBLIC_URL:-}")"
  require_public_http_reachable "SSO" "${sso_public_base_url}"
  require_public_http_reachable "Admission" "${admission_public_base_url}/verify/__stuhelper_public_ingress_probe__"

  if [[ "${PUBLIC_INGRESS_CASDOOR_UPSTREAM_PREFLIGHT_ENABLED:-true}" == "true" ]]; then
    require_public_oidc_discovery "SSO" "${sso_public_base_url}"
    local sso_jwks_uri
    if ! sso_jwks_uri="$(public_oidc_jwks_uri "${sso_public_base_url}")"; then
      die "SSO JWKS URI preflight failed for ${sso_public_base_url}: discovery did not expose jwks_uri"
    fi
    require_public_jwks "SSO" "${sso_jwks_uri}"
  else
    log "SSO public OIDC metadata preflight skipped because PUBLIC_INGRESS_CASDOOR_UPSTREAM_PREFLIGHT_ENABLED is not true"
  fi
}

require_public_ingress_config_preflight() {
  if [[ "${PUBLIC_INGRESS_CONFIG_PREFLIGHT_ENABLED:-true}" != "true" ]]; then
    warn "public Nginx ingress config preflight skipped because PUBLIC_INGRESS_CONFIG_PREFLIGHT_ENABLED is not true"
    return 0
  fi

  "${COMMON_LIB_DIR}/../nginx-public-ingress-preflight.sh"
}

ensure_env_file() {
  local template_file="${1:-${ENV_TEMPLATE_FILE}}"
  [[ -n "${template_file}" ]] || die "ENV_TEMPLATE_FILE must not be empty"
  [[ -f "${template_file}" ]] || die "missing env template: ${template_file}"
  if [[ ! -f "${ENV_FILE}" ]]; then
    cp "${template_file}" "${ENV_FILE}"
    log "created ${ENV_FILE} from $(basename "${template_file}")"
  fi
}

prefer_production_env_files_if_default() {
  local default_env_file="${REPO_ROOT}/.env"
  [[ "${ENV_FILE}" == "${default_env_file}" ]] || return 0
  [[ ! -f "${ENV_FILE}" ]] || return 0
  [[ -f "${REPO_ROOT}/.env.prod.shared" ]] || return 0

  export ENV_FILE="${REPO_ROOT}/.env.prod.shared"
  if [[ -z "${SECRETS_ENV_FILE:-}" ]]; then
    if [[ -f "${REPO_ROOT}/.env.prod.secrets.local" ]]; then
      export SECRETS_ENV_FILE="${REPO_ROOT}/.env.prod.secrets.local"
    elif [[ -f "${REPO_ROOT}/.env.prod.secrets" ]]; then
      export SECRETS_ENV_FILE="${REPO_ROOT}/.env.prod.secrets"
    fi
  fi
  if [[ "${GENERATED_ENV_FILE}" == "${REPO_ROOT}/.env.generated" ]]; then
    export GENERATED_ENV_FILE="${REPO_ROOT}/.env.prod.generated"
  fi
  if [[ "${GENERATED_SECRET_ENV_FILE}" == "${REPO_ROOT}/.env.generated.secrets" ]]; then
    export GENERATED_SECRET_ENV_FILE="${REPO_ROOT}/.env.prod.generated.secrets"
  fi
}

ensure_generated_files() {
  mkdir -p "${GENERATED_OBS_DIR}/prometheus" "${GENERATED_OBS_DIR}/alertmanager"
  touch "${GENERATED_ENV_FILE}"
  touch "${GENERATED_SECRET_ENV_FILE}"
}

should_source_generated_secret_from_backend() {
  [[ -n "${GENERATED_ENV_SECRET_REF:-}" ]] && [[ -n "${SECRET_BACKEND:-}" ]] && [[ "${SECRET_BACKEND:-}" != "none" ]] && [[ "${SECRET_BACKEND:-}" != "file" ]]
}

source_generated_secret_env() {
  if should_source_generated_secret_from_backend; then
    local rendered
    if ! rendered="$(secret_backend_read_to_stdout "${GENERATED_ENV_SECRET_REF}")"; then
      die "failed to read generated secret env from ${SECRET_BACKEND}: ${GENERATED_ENV_SECRET_REF}"
    fi
    if [[ -n "${rendered}" ]]; then
      local tmp_env
      tmp_env="$(mktemp)"
      printf '%s\n' "${rendered}" >"${tmp_env}"
      source_env_file "${tmp_env}"
      rm -f "${tmp_env}"
    fi
    return 0
  fi

  if [[ -f "${GENERATED_SECRET_ENV_FILE}" ]]; then
    source_env_file "${GENERATED_SECRET_ENV_FILE}"
  fi
}

materialize_secret_env_inputs() {
  if [[ -n "${SHARED_ENV_SECRET_REF}" ]]; then
    materialize_secret_env_file "${SHARED_ENV_SECRET_REF}" "${ENV_FILE}"
  fi
  if [[ -n "${SECRETS_ENV_SECRET_REF}" && -n "${SECRETS_ENV_FILE}" ]]; then
    materialize_secret_env_file "${SECRETS_ENV_SECRET_REF}" "${SECRETS_ENV_FILE}"
  fi
  if [[ -n "${GENERATED_ENV_SECRET_REF:-}" ]] && ! should_source_generated_secret_from_backend; then
    materialize_secret_env_file "${GENERATED_ENV_SECRET_REF}" "${GENERATED_SECRET_ENV_FILE}"
  fi
}

ensure_secrets_env_file() {
  if [[ -n "${SECRETS_ENV_FILE}" ]]; then
    mkdir -p "$(dirname "${SECRETS_ENV_FILE}")"
    touch "${SECRETS_ENV_FILE}"
  fi
}

load_env() {
  ensure_env_file
  ensure_secrets_env_file
  ensure_generated_files
  materialize_secret_env_inputs
  local preserved_tag="${TAG-__STUHELPER_UNSET__}"
  local preserved_rollback_tag="${ROLLBACK_TAG-__STUHELPER_UNSET__}"
  local preserved_backend_image_ref="${BACKEND_IMAGE_REF-__STUHELPER_UNSET__}"
  local preserved_frontend_image_ref="${FRONTEND_IMAGE_REF-__STUHELPER_UNSET__}"
  local preserved_admin_image_ref="${ADMIN_IMAGE_REF-__STUHELPER_UNSET__}"
  set -a
  source_env_file "${ENV_FILE}"
  if [[ -n "${SECRETS_ENV_FILE}" && -f "${SECRETS_ENV_FILE}" ]]; then
    source_env_file "${SECRETS_ENV_FILE}"
  fi
  source_env_file "${GENERATED_ENV_FILE}"
  source_generated_secret_env
  if [[ "${STUHELPER_PRESERVE_POSTGRES_URL_PLACEHOLDERS:-false}" != "true" ]]; then
    materialize_postgres_runtime_urls
  fi
  normalize_backup_object_storage_env
  export POSTGRES_WAL_ARCHIVE_VOLUME_NAME="${POSTGRES_WAL_ARCHIVE_VOLUME_NAME:-${STACK_NAME:-stuhelper}-postgres-wal-archive}"
  if [[ "${preserved_tag}" != "__STUHELPER_UNSET__" ]]; then export TAG="${preserved_tag}"; fi
  if [[ "${preserved_rollback_tag}" != "__STUHELPER_UNSET__" ]]; then export ROLLBACK_TAG="${preserved_rollback_tag}"; fi
  if [[ "${preserved_backend_image_ref}" != "__STUHELPER_UNSET__" ]]; then export BACKEND_IMAGE_REF="${preserved_backend_image_ref}"; fi
  if [[ "${preserved_frontend_image_ref}" != "__STUHELPER_UNSET__" ]]; then export FRONTEND_IMAGE_REF="${preserved_frontend_image_ref}"; fi
  if [[ "${preserved_admin_image_ref}" != "__STUHELPER_UNSET__" ]]; then export ADMIN_IMAGE_REF="${preserved_admin_image_ref}"; fi
  set +a
  unset BASH_ENV ENV
}

load_env_preserving() {
  local -A preserved_values=()
  local key

  for key in "$@"; do
    [[ "${key}" =~ ^[A-Za-z_][A-Za-z0-9_]*$ ]] ||
      die "invalid environment key requested for preservation: ${key}"
    if [[ -v "${key}" ]]; then
      preserved_values["${key}"]="${!key}"
    fi
  done

  load_env

  for key in "${!preserved_values[@]}"; do
    printf -v "${key}" '%s' "${preserved_values[${key}]}"
    export "${key}"
  done
  materialize_postgres_runtime_urls
  unset BASH_ENV ENV
}

load_remote_deploy_config() {
  local config_file="${1:-${REMOTE_DEPLOY_CONFIG_FILE}}"
  [[ -n "${config_file}" ]] || die "REMOTE_DEPLOY_CONFIG_FILE must not be empty"
  [[ -f "${config_file}" ]] || die "missing remote deploy config: ${config_file} (run ./infra/ops/init-remote-deploy-config.sh on the target host first)"

  local preserved_tag="${TAG-__STUHELPER_UNSET__}"
  local preserved_rollback_tag="${ROLLBACK_TAG-__STUHELPER_UNSET__}"
  local preserved_backend_image_ref="${BACKEND_IMAGE_REF-__STUHELPER_UNSET__}"
  local preserved_frontend_image_ref="${FRONTEND_IMAGE_REF-__STUHELPER_UNSET__}"
  local preserved_admin_image_ref="${ADMIN_IMAGE_REF-__STUHELPER_UNSET__}"
  local preserved_registry_username="${REGISTRY_USERNAME-__STUHELPER_UNSET__}"
  local preserved_registry_password="${REGISTRY_PASSWORD-__STUHELPER_UNSET__}"

  set -a
  source_remote_deploy_config_env_file "${config_file}"
  if [[ "${preserved_tag}" != "__STUHELPER_UNSET__" ]]; then export TAG="${preserved_tag}"; fi
  if [[ "${preserved_rollback_tag}" != "__STUHELPER_UNSET__" ]]; then export ROLLBACK_TAG="${preserved_rollback_tag}"; fi
  if [[ "${preserved_backend_image_ref}" != "__STUHELPER_UNSET__" ]]; then export BACKEND_IMAGE_REF="${preserved_backend_image_ref}"; fi
  if [[ "${preserved_frontend_image_ref}" != "__STUHELPER_UNSET__" ]]; then export FRONTEND_IMAGE_REF="${preserved_frontend_image_ref}"; fi
  if [[ "${preserved_admin_image_ref}" != "__STUHELPER_UNSET__" ]]; then export ADMIN_IMAGE_REF="${preserved_admin_image_ref}"; fi
  if [[ "${preserved_registry_username}" != "__STUHELPER_UNSET__" ]]; then export REGISTRY_USERNAME="${preserved_registry_username}"; fi
  if [[ "${preserved_registry_password}" != "__STUHELPER_UNSET__" ]]; then export REGISTRY_PASSWORD="${preserved_registry_password}"; fi
  set +a
  unset BASH_ENV ENV
}

compose() {
  (
    cd "${REPO_ROOT}" && \
    compose_files=(-f "${REPO_ROOT}/docker-compose.yml") && \
    if [[ -f "${REPO_ROOT}/docker-compose.observability.yml" ]]; then compose_files+=(-f "${REPO_ROOT}/docker-compose.observability.yml"); fi && \
    if [[ " $* " == *" --profile prod "* && -f "${REPO_ROOT}/docker-compose.prod.yml" ]]; then compose_files+=(-f "${REPO_ROOT}/docker-compose.prod.yml"); fi && \
    if [[ -n "${EXTERNAL_DATASTORE_NETWORK:-}" && "${EXTERNAL_POSTGRES_ENABLED:-false}" == "true" && -f "${REPO_ROOT}/docker-compose.external-datastore.yml" ]]; then compose_files+=(-f "${REPO_ROOT}/docker-compose.external-datastore.yml"); fi && \
    preserved_tag="${TAG-__STUHELPER_UNSET__}" && \
    preserved_rollback_tag="${ROLLBACK_TAG-__STUHELPER_UNSET__}" && \
    preserved_backend_image_ref="${BACKEND_IMAGE_REF-__STUHELPER_UNSET__}" && \
    preserved_frontend_image_ref="${FRONTEND_IMAGE_REF-__STUHELPER_UNSET__}" && \
    preserved_admin_image_ref="${ADMIN_IMAGE_REF-__STUHELPER_UNSET__}" && \
    set -a && \
    source_env_file "${ENV_FILE}" && \
    if [[ -n "${SECRETS_ENV_FILE}" && -f "${SECRETS_ENV_FILE}" ]]; then source_env_file "${SECRETS_ENV_FILE}"; fi && \
    if [[ -f "${GENERATED_ENV_FILE}" ]]; then source_env_file "${GENERATED_ENV_FILE}"; fi && \
    source_generated_secret_env && \
    materialize_postgres_runtime_urls && \
    if [[ "${preserved_tag}" != "__STUHELPER_UNSET__" ]]; then export TAG="${preserved_tag}"; fi && \
    if [[ "${preserved_rollback_tag}" != "__STUHELPER_UNSET__" ]]; then export ROLLBACK_TAG="${preserved_rollback_tag}"; fi && \
    if [[ "${preserved_backend_image_ref}" != "__STUHELPER_UNSET__" ]]; then export BACKEND_IMAGE_REF="${preserved_backend_image_ref}"; fi && \
    if [[ "${preserved_frontend_image_ref}" != "__STUHELPER_UNSET__" ]]; then export FRONTEND_IMAGE_REF="${preserved_frontend_image_ref}"; fi && \
    if [[ "${preserved_admin_image_ref}" != "__STUHELPER_UNSET__" ]]; then export ADMIN_IMAGE_REF="${preserved_admin_image_ref}"; fi && \
    set +a && \
    if [[ -z "${BACKUP_OBJECT_STORAGE_ENDPOINT:-}" && -n "${OBJECT_STORAGE_ENDPOINT:-}" ]]; then export BACKUP_OBJECT_STORAGE_ENDPOINT="${OBJECT_STORAGE_ENDPOINT}"; fi && \
    export BACKUP_OBJECT_STORAGE_BUCKET="${BACKUP_OBJECT_STORAGE_BUCKET:-stuhelper-postgres-backup}" && \
    export BACKUP_OBJECT_STORAGE_PREFIX="${BACKUP_OBJECT_STORAGE_PREFIX:-postgres}" && \
    if [[ -z "${BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID:-}" && -n "${OBJECT_STORAGE_ACCESS_KEY_ID:-}" ]]; then export BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID="${OBJECT_STORAGE_ACCESS_KEY_ID}"; fi && \
    if [[ -z "${BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY:-}" && -n "${OBJECT_STORAGE_SECRET_ACCESS_KEY:-}" ]]; then export BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY="${OBJECT_STORAGE_SECRET_ACCESS_KEY}"; fi && \
    if [[ -z "${BACKUP_OBJECT_STORAGE_TLS_INSECURE:-}" ]]; then \
      if [[ "${OBJECT_STORAGE_USE_SSL:-false}" == "true" ]]; then export BACKUP_OBJECT_STORAGE_TLS_INSECURE="false"; else export BACKUP_OBJECT_STORAGE_TLS_INSECURE="true"; fi; \
    fi && \
    export POSTGRES_WAL_ARCHIVE_VOLUME_NAME="${POSTGRES_WAL_ARCHIVE_VOLUME_NAME:-${STACK_NAME:-stuhelper}-postgres-wal-archive}" && \
    ENV_FILE_PATH="${ENV_FILE}" \
    SECRETS_ENV_FILE_PATH="${SECRETS_ENV_FILE}" \
    GENERATED_ENV_FILE_PATH="${GENERATED_ENV_FILE}" \
    GENERATED_SECRET_ENV_FILE_PATH="${GENERATED_SECRET_ENV_FILE}" \
    BACKEND_IMAGE_REF="${BACKEND_IMAGE_REF:-}" \
    FRONTEND_IMAGE_REF="${FRONTEND_IMAGE_REF:-}" \
    ADMIN_IMAGE_REF="${ADMIN_IMAGE_REF:-}" \
    docker compose "${compose_files[@]}" --env-file "${ENV_FILE}" "$@"
  )
}

wait_for_http() {
  local name="$1"
  local url="$2"
  local retries="${3:-60}"
  local sleep_seconds="${4:-2}"
  local i

  for ((i = 1; i <= retries; i++)); do
    if curl -fsS --max-time 5 "${url}" >/dev/null 2>&1; then
      log "${name} is ready: ${url}"
      return 0
    fi
    sleep "${sleep_seconds}"
  done

  die "${name} did not become ready in time: ${url}"
}

upsert_env_file() {
  local file="$1"
  local key="$2"
  local value="$3"
  local python_file="${file}"

  if [[ "$(uname -s)" =~ ^(MINGW|MSYS|CYGWIN) ]] && command -v cygpath >/dev/null 2>&1; then
    python_file="$(cygpath -w "${file}")"
  fi

  MSYS2_ARG_CONV_EXCL="*" python3 - "$python_file" "$key" "$value" <<'PY'
from pathlib import Path
import sys

path = Path(sys.argv[1])
key = sys.argv[2]
value = sys.argv[3]

lines = path.read_text().splitlines() if path.exists() else []
updated = False
for idx, line in enumerate(lines):
    if line.startswith(f"{key}="):
        lines[idx] = f"{key}={value}"
        updated = True
        break
if not updated:
    lines.append(f"{key}={value}")
path.write_text("\n".join(lines) + "\n")
PY
}

random_hex() {
  local nbytes="${1:-32}"
  python3 - "$nbytes" <<'PY'
import secrets
import sys
print(secrets.token_hex(int(sys.argv[1])))
PY
}

git_tag_default() {
  git -C "${REPO_ROOT}" rev-parse --short HEAD 2>/dev/null || echo "local"
}

derive_release_id_from_image_ref() {
  local image_ref="${1:-}"
  python3 - "${image_ref}" <<'PY'
import sys

ref = sys.argv[1].strip()
if not ref:
    raise SystemExit(1)

if "@sha256:" in ref:
    print(ref.split("@sha256:", 1)[1][:12])
    raise SystemExit(0)

image, sep, tag = ref.rpartition(":")
if not sep or "/" not in image:
    raise SystemExit(1)

print(tag)
PY
}

resolve_registry_credentials() {
  if [[ -z "${REGISTRY_USERNAME:-}" && -n "${REGISTRY_USERNAME_SECRET_REF:-}" ]]; then
    REGISTRY_USERNAME="$(materialize_secret_value "${REGISTRY_USERNAME_SECRET_REF}")"
    export REGISTRY_USERNAME
  fi
  if [[ -z "${REGISTRY_PASSWORD:-}" && -n "${REGISTRY_PASSWORD_SECRET_REF:-}" ]]; then
    REGISTRY_PASSWORD="$(materialize_secret_value "${REGISTRY_PASSWORD_SECRET_REF}")"
    export REGISTRY_PASSWORD
  fi
}

docker_registry_login() {
  local auth_mode="${REGISTRY_AUTH_MODE:-persistent-secret}"
  [[ -n "${REGISTRY:-}" ]] || die "REGISTRY is required"

  case "${auth_mode}" in
    workflow-token)
      [[ "${CI_REGISTRY_LOGIN_READY:-false}" == "true" ]] ||
        die "workflow-token registry auth must be established by remote-ci-release.sh"
      ;;
    persistent-secret)
      resolve_registry_credentials
      [[ -n "${REGISTRY_USERNAME:-}" ]] || die "REGISTRY_USERNAME or REGISTRY_USERNAME_SECRET_REF is required"
      [[ -n "${REGISTRY_PASSWORD:-}" ]] || die "REGISTRY_PASSWORD or REGISTRY_PASSWORD_SECRET_REF is required"
      printf '%s\n' "${REGISTRY_PASSWORD}" |
        docker login "${REGISTRY}" --username "${REGISTRY_USERNAME}" --password-stdin >/dev/null
      ;;
    *)
      die "REGISTRY_AUTH_MODE must be workflow-token or persistent-secret"
      ;;
  esac
}

record_release() {
  local tag="$1"
  mkdir -p "${DEPLOY_STATE_DIR}"
  mkdir -p "${DEPLOY_STATE_DIR}/releases"
  local now
  now="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  printf '%s\t%s\n' "${now}" "${tag}" >> "${DEPLOY_STATE_DIR}/releases.log"
  cat > "${DEPLOY_STATE_DIR}/current-release.env" <<EOF
TAG=${tag}
DEPLOYED_AT=${now}
BACKEND_IMAGE_REF=${BACKEND_IMAGE_REF:-}
FRONTEND_IMAGE_REF=${FRONTEND_IMAGE_REF:-}
ADMIN_IMAGE_REF=${ADMIN_IMAGE_REF:-}
EOF
  cat > "${DEPLOY_STATE_DIR}/releases/${tag}.env" <<EOF
TAG=${tag}
DEPLOYED_AT=${now}
BACKEND_IMAGE_REF=${BACKEND_IMAGE_REF:-}
FRONTEND_IMAGE_REF=${FRONTEND_IMAGE_REF:-}
ADMIN_IMAGE_REF=${ADMIN_IMAGE_REF:-}
EOF
}

resolve_previous_release_tag() {
  local current_tag="${1:-}"
  local releases_file="${DEPLOY_STATE_DIR}/releases.log"
  [[ -f "${releases_file}" ]] || return 1
  python3 - "${releases_file}" "${current_tag}" <<'PY'
from pathlib import Path
import sys

path = Path(sys.argv[1])
current = sys.argv[2]
lines = path.read_text().splitlines()
for line in reversed(lines):
    parts = line.split("\t")
    if len(parts) >= 2 and parts[1] and parts[1] != current:
        print(parts[1])
        raise SystemExit(0)
raise SystemExit(1)
PY
}
