#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: infra/ops/baota-compose-refresh-image-refs.sh [options]

Refresh immutable image refs in a generated Baota root docker-compose.yml from
the non-secret production shared env file. This script is for deployments where
Baota manages a generated Compose root while the repository source lives under
source/.

Options:
  --compose-file PATH   Generated Baota root compose file. Default: docker-compose.yml
  --env-file PATH       Non-secret env file containing *_IMAGE_REF. Default: source/.env.prod.shared
  --backup-dir PATH     Backup directory for --apply. Default: backups
  --apply               Write changes. Without this flag, print a dry-run diff.
  -h, --help            Show this help.
USAGE
}

compose_file="${BAOTA_COMPOSE_FILE:-docker-compose.yml}"
env_file="${BAOTA_COMPOSE_ENV_FILE:-source/.env.prod.shared}"
backup_dir="${BAOTA_COMPOSE_BACKUP_DIR:-backups}"
apply=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    --compose-file)
      compose_file="${2:-}"
      shift 2
      ;;
    --env-file)
      env_file="${2:-}"
      shift 2
      ;;
    --backup-dir)
      backup_dir="${2:-}"
      shift 2
      ;;
    --apply)
      apply=true
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "[baota-compose-refresh-image-refs][error] unknown option: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

die() {
  echo "[baota-compose-refresh-image-refs][error] $*" >&2
  exit 1
}

read_env_value() {
  local key="$1"
  local file="$2"
  awk -v key="${key}" '
    BEGIN { prefix = key "=" }
    index($0, prefix) == 1 {
      value = substr($0, length(prefix) + 1)
      sub(/\r$/, "", value)
      print value
      found = 1
      exit
    }
    END { if (!found) exit 1 }
  ' "${file}"
}

require_image_ref() {
  local key="$1"
  local value="$2"
  [[ -n "${value}" ]] || die "${key} is empty"
  [[ "${value}" != REPLACE_WITH_* ]] || die "${key} still uses placeholder"
  [[ "${value}" != *":latest" ]] || die "${key} must be immutable, got :latest"
  [[ "${value}" == *":"* || "${value}" == *@sha256:* ]] || die "${key} must include a tag or digest"
}

[[ -f "${compose_file}" ]] || die "missing compose file: ${compose_file}"
[[ -f "${env_file}" ]] || die "missing env file: ${env_file}"

backend_image_ref="$(read_env_value BACKEND_IMAGE_REF "${env_file}")" || die "BACKEND_IMAGE_REF not found in ${env_file}"
frontend_image_ref="$(read_env_value FRONTEND_IMAGE_REF "${env_file}")" || die "FRONTEND_IMAGE_REF not found in ${env_file}"
admin_image_ref="$(read_env_value ADMIN_IMAGE_REF "${env_file}")" || die "ADMIN_IMAGE_REF not found in ${env_file}"

require_image_ref BACKEND_IMAGE_REF "${backend_image_ref}"
require_image_ref FRONTEND_IMAGE_REF "${frontend_image_ref}"
require_image_ref ADMIN_IMAGE_REF "${admin_image_ref}"

tmp_file="$(mktemp)"
cleanup() {
  rm -f "${tmp_file}"
}
trap cleanup EXIT

service=""
migrate_count=0
app_count=0
frontend_count=0
admin_count=0

while IFS= read -r line || [[ -n "${line}" ]]; do
  if [[ "${line}" =~ ^[[:space:]]{2}([A-Za-z0-9_-]+):[[:space:]]*$ ]]; then
    service="${BASH_REMATCH[1]}"
  fi

  if [[ "${line}" =~ ^[[:space:]]{4}image:[[:space:]]* ]]; then
    case "${service}" in
      migrate)
        printf '    image: %s\n' "${backend_image_ref}" >>"${tmp_file}"
        migrate_count=$((migrate_count + 1))
        continue
        ;;
      app)
        printf '    image: %s\n' "${backend_image_ref}" >>"${tmp_file}"
        app_count=$((app_count + 1))
        continue
        ;;
      frontend)
        printf '    image: %s\n' "${frontend_image_ref}" >>"${tmp_file}"
        frontend_count=$((frontend_count + 1))
        continue
        ;;
      admin)
        printf '    image: %s\n' "${admin_image_ref}" >>"${tmp_file}"
        admin_count=$((admin_count + 1))
        continue
        ;;
    esac
  fi

  printf '%s\n' "${line}" >>"${tmp_file}"
done <"${compose_file}"

if [[ "${migrate_count}" -ne 1 || "${app_count}" -ne 1 || "${frontend_count}" -ne 1 || "${admin_count}" -ne 1 ]]; then
  die "unexpected image line counts: migrate=${migrate_count} app=${app_count} frontend=${frontend_count} admin=${admin_count}"
fi

if cmp -s "${compose_file}" "${tmp_file}"; then
  echo "[baota-compose-refresh-image-refs] no changes needed"
  exit 0
fi

if [[ "${apply}" != "true" ]]; then
  echo "[baota-compose-refresh-image-refs] dry-run diff:"
  diff -u "${compose_file}" "${tmp_file}" || true
  echo "[baota-compose-refresh-image-refs] rerun with --apply to write changes"
  exit 0
fi

mkdir -p "${backup_dir}"
backup_file="${backup_dir}/$(basename "${compose_file}").before-image-refresh-$(date -u +%Y%m%dT%H%M%SZ)"
cp "${compose_file}" "${backup_file}"
mv "${tmp_file}" "${compose_file}"
trap - EXIT

echo "[baota-compose-refresh-image-refs] updated ${compose_file}"
echo "[baota-compose-refresh-image-refs] backup: ${backup_file}"
echo "[baota-compose-refresh-image-refs] backend: ${backend_image_ref}"
echo "[baota-compose-refresh-image-refs] frontend: ${frontend_image_ref}"
echo "[baota-compose-refresh-image-refs] admin: ${admin_image_ref}"
