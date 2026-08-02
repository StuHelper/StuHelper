#!/usr/bin/env bash

run_backup_object_storage_rclone() {
  local container_user="$1"
  local mount_spec="$2"
  shift 2

  local image_ref="${RCLONE_IMAGE_REF:-rclone/rclone:beta@sha256:f52965eba611ba8984117638b2a0539dcce170731937f93fbace66897d102698}"
  local endpoint="${BACKUP_OBJECT_STORAGE_ENDPOINT:-}"
  local region="${BACKUP_OBJECT_STORAGE_REGION:-${OBJECT_STORAGE_REGION:-us-east-1}}"
  local force_path_style="${BACKUP_OBJECT_STORAGE_FORCE_PATH_STYLE:-${OBJECT_STORAGE_FORCE_PATH_STYLE:-true}}"
  local tls_insecure="${BACKUP_OBJECT_STORAGE_TLS_INSECURE:-false}"
  local ca_file="${BACKUP_OBJECT_STORAGE_TLS_CA:-}"
  local pinned_hosts="${BACKUP_OBJECT_STORAGE_PINNED_HOSTS:-}"
  local bucket="${BACKUP_OBJECT_STORAGE_BUCKET:-}"
  local -x RCLONE_CONFIG_TARGET_TYPE="s3"
  local -x RCLONE_CONFIG_TARGET_PROVIDER="${BACKUP_OBJECT_STORAGE_PROVIDER:-Other}"
  local -x RCLONE_CONFIG_TARGET_ACCESS_KEY_ID="${BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID:-}"
  local -x RCLONE_CONFIG_TARGET_SECRET_ACCESS_KEY="${BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY:-}"
  local -x RCLONE_CONFIG_TARGET_ENDPOINT="${endpoint}"
  local -x RCLONE_CONFIG_TARGET_REGION="${region}"
  local -x RCLONE_CONFIG_TARGET_FORCE_PATH_STYLE="${force_path_style}"
  local -x RCLONE_CONFIG_TARGET_NO_CHECK_BUCKET="true"
  local -a docker_args=(
    run
    --rm
    --read-only
    --cap-drop ALL
    --security-opt no-new-privileges
    --tmpfs "/tmp:rw,noexec,nosuid,size=64m"
    --user "${container_user}"
    --mount "${mount_spec}"
    --env RCLONE_CONFIG_TARGET_TYPE
    --env RCLONE_CONFIG_TARGET_PROVIDER
    --env RCLONE_CONFIG_TARGET_ACCESS_KEY_ID
    --env RCLONE_CONFIG_TARGET_SECRET_ACCESS_KEY
    --env RCLONE_CONFIG_TARGET_ENDPOINT
    --env RCLONE_CONFIG_TARGET_REGION
    --env RCLONE_CONFIG_TARGET_FORCE_PATH_STYLE
    --env RCLONE_CONFIG_TARGET_NO_CHECK_BUCKET
  )
  local -a rclone_args=(
    --config=/dev/null
    --cache-dir=/tmp/rclone
    --check-first
    --checksum
    --metadata
    --transfers=4
    --checkers=8
    --retries=5
    --low-level-retries=10
    --stats=30s
    --stats-one-line
  )

  [[ "${image_ref}" =~ ^.+@sha256:[0-9a-f]{64}$ ]] ||
    die "RCLONE_IMAGE_REF must be a complete image@sha256 reference"
  [[ -n "${endpoint}" ]] || die "BACKUP_OBJECT_STORAGE_ENDPOINT is required"
  [[ -n "${BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID:-}" ]] ||
    die "BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID is required"
  [[ -n "${BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY:-}" ]] ||
    die "BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY is required"
  case "${force_path_style}" in
    true|false) ;;
    *) die "BACKUP_OBJECT_STORAGE_FORCE_PATH_STYLE must be true or false" ;;
  esac
  case "${tls_insecure}" in
    true)
      [[ -z "${ca_file}" ]] ||
        die "BACKUP_OBJECT_STORAGE_TLS_CA and BACKUP_OBJECT_STORAGE_TLS_INSECURE=true are mutually exclusive"
      rclone_args+=(--no-check-certificate)
      ;;
    false) ;;
    *) die "BACKUP_OBJECT_STORAGE_TLS_INSECURE must be true or false" ;;
  esac

  if [[ -n "${ca_file}" ]]; then
    [[ -f "${ca_file}" && -r "${ca_file}" ]] ||
      die "BACKUP_OBJECT_STORAGE_TLS_CA must be a readable regular file: ${ca_file}"
    docker_args+=(--mount "type=bind,src=${ca_file},dst=/tls/object-storage-ca.crt,readonly")
    rclone_args+=(--ca-cert=/tls/object-storage-ca.crt)
  fi
  if [[ -n "${BACKUP_OBJECT_STORAGE_DOCKER_NETWORK:-}" ]]; then
    [[ "${BACKUP_OBJECT_STORAGE_DOCKER_NETWORK}" =~ ^[A-Za-z0-9_.-]+$ ]] ||
      die "BACKUP_OBJECT_STORAGE_DOCKER_NETWORK contains unsupported characters"
    docker_args+=(--network "${BACKUP_OBJECT_STORAGE_DOCKER_NETWORK}")
  fi
  if [[ -n "${pinned_hosts}" ]]; then
    local pinned_payload
    if ! pinned_payload="$(python3 - \
      "${endpoint}" \
      "${pinned_hosts}" \
      "${bucket}" \
      "${force_path_style}" 2>&1 <<'PY'
import ipaddress
import re
import sys
from urllib.parse import urlsplit, urlunsplit

endpoint, pinned_hosts, bucket, force_path_style = sys.argv[1:]
parsed = urlsplit(endpoint)
raw_host = (parsed.hostname or "").lower()
if parsed.username is not None or parsed.password is not None:
    raise SystemExit(
        "BACKUP_OBJECT_STORAGE_PINNED_HOSTS does not allow endpoint credentials"
    )
if "%" in raw_host:
    raise SystemExit(
        "BACKUP_OBJECT_STORAGE_PINNED_HOSTS does not allow an IPv6 zone identifier"
    )
if raw_host.endswith("."):
    raise SystemExit(
        "BACKUP_OBJECT_STORAGE_PINNED_HOSTS does not allow a trailing-dot endpoint hostname"
    )
host = raw_host
labels = host.split(".")
if (
    not host
    or len(host) > 253
    or any(
        not re.fullmatch(r"[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?", label)
        for label in labels
    )
):
    raise SystemExit(
        "BACKUP_OBJECT_STORAGE_PINNED_HOSTS requires a valid ASCII DNS endpoint hostname"
    )

expected_hosts = {host}
if force_path_style == "false":
    virtual_host = f"{bucket}.{host}"
    virtual_labels = virtual_host.split(".")
    if not bucket or len(virtual_host) > 253 or any(
        not re.fullmatch(r"[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?", label)
        for label in virtual_labels
    ):
        raise SystemExit(
            "BACKUP_OBJECT_STORAGE_BUCKET must form a valid lowercase ASCII virtual-hosted S3 hostname"
        )
    expected_hosts.add(virtual_host)

addresses_by_host = {}
for raw_value in pinned_hosts.splitlines():
    value = raw_value.strip()
    if not value:
        continue
    pinned_host, separator, raw_address = value.partition("=")
    pinned_host = pinned_host.lower()
    if separator != "=" or pinned_host not in expected_hosts:
        raise SystemExit(
            f"BACKUP_OBJECT_STORAGE_PINNED_HOSTS contains an unexpected transfer hostname: {value}"
        )
    try:
        address = ipaddress.ip_address(raw_address)
    except ValueError:
        raise SystemExit(
            f"BACKUP_OBJECT_STORAGE_PINNED_HOSTS contains an invalid IP address: {value}"
        ) from None
    if isinstance(address, ipaddress.IPv6Address) and address.ipv4_mapped is not None:
        address = address.ipv4_mapped
    addresses_by_host.setdefault(pinned_host, set()).add(address)
missing_hosts = expected_hosts - addresses_by_host.keys()
if missing_hosts:
    raise SystemExit(
        "BACKUP_OBJECT_STORAGE_PINNED_HOSTS is missing validated addresses for: "
        + ", ".join(sorted(missing_hosts))
    )

try:
    port = parsed.port
except ValueError as error:
    raise SystemExit(f"BACKUP_OBJECT_STORAGE_ENDPOINT has an invalid port: {error}") from None
normalized_authority = host if port is None else f"{host}:{port}"
print(
    urlunsplit(
        (
            parsed.scheme,
            normalized_authority,
            parsed.path,
            parsed.query,
            parsed.fragment,
        )
    )
)
for pinned_host in sorted(addresses_by_host):
    for address in sorted(
        addresses_by_host[pinned_host],
        key=lambda value: (value.version, int(value)),
    ):
        print(f"{pinned_host}={address.compressed}")
PY
    )"; then
      die "${pinned_payload}"
    fi
    local -a pinned_lines=()
    mapfile -t pinned_lines <<<"${pinned_payload}"
    (( ${#pinned_lines[@]} >= 2 )) ||
      die "failed to render the validated backup endpoint address pins"
    endpoint="${pinned_lines[0]}"
    RCLONE_CONFIG_TARGET_ENDPOINT="${endpoint}"
    local pinned_index
    for ((pinned_index = 1; pinned_index < ${#pinned_lines[@]}; pinned_index++)); do
      docker_args+=(--add-host "${pinned_lines[${pinned_index}]}")
    done
  fi

  docker "${docker_args[@]}" "${image_ref}" "$@" "${rclone_args[@]}"
}
