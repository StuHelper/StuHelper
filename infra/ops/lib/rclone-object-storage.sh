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

  docker "${docker_args[@]}" "${image_ref}" "$@" "${rclone_args[@]}"
}
