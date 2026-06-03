#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: infra/ops/baota-compose-import-image-tars.sh [options]

Verify and import locally built production image tarballs on a Baota Compose
host, update source/.env.prod.shared with immutable image refs, then refresh
the generated Baota root docker-compose.yml image lines.

Required options:
  --backend-ref REF       Immutable backend image ref to write to env.
  --backend-tar PATH      Backend image tar or tar.gz.
  --backend-sha256 SHA    Expected backend tar sha256.
  --frontend-ref REF      Immutable frontend image ref to write to env.
  --frontend-tar PATH     Frontend image tar or tar.gz.
  --frontend-sha256 SHA   Expected frontend tar sha256.
  --admin-ref REF         Immutable admin image ref to write to env.
  --admin-tar PATH        Admin image tar or tar.gz.
  --admin-sha256 SHA      Expected admin tar sha256.

Options:
  --compose-root PATH     Baota Compose root. Default: current directory.
  --source-dir PATH       Source directory relative to compose root, or absolute.
                          Default: source
  --env-file PATH         Env file relative to compose root, or absolute.
                          Default: source/.env.prod.shared
  --backup-dir PATH       Backup directory for compose refresh. Default: backups
  --apply                 Write changes and docker load images. Without this
                          flag, only verify inputs and print the planned refs.
  -h, --help              Show this help.

The script never accepts or prints secrets. It only touches the non-secret
shared env image refs and delegates root compose rewriting to
baota-compose-refresh-image-refs.sh.
USAGE
}

compose_root="${BAOTA_COMPOSE_ROOT:-.}"
source_dir="${BAOTA_COMPOSE_SOURCE_DIR:-source}"
env_file="${BAOTA_COMPOSE_ENV_FILE:-source/.env.prod.shared}"
backup_dir="${BAOTA_COMPOSE_BACKUP_DIR:-backups}"
apply=false

backend_ref=""
backend_tar=""
backend_sha256=""
frontend_ref=""
frontend_tar=""
frontend_sha256=""
admin_ref=""
admin_tar=""
admin_sha256=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --backend-ref) backend_ref="${2:-}"; shift 2 ;;
    --backend-tar) backend_tar="${2:-}"; shift 2 ;;
    --backend-sha256) backend_sha256="${2:-}"; shift 2 ;;
    --frontend-ref) frontend_ref="${2:-}"; shift 2 ;;
    --frontend-tar) frontend_tar="${2:-}"; shift 2 ;;
    --frontend-sha256) frontend_sha256="${2:-}"; shift 2 ;;
    --admin-ref) admin_ref="${2:-}"; shift 2 ;;
    --admin-tar) admin_tar="${2:-}"; shift 2 ;;
    --admin-sha256) admin_sha256="${2:-}"; shift 2 ;;
    --compose-root) compose_root="${2:-}"; shift 2 ;;
    --source-dir) source_dir="${2:-}"; shift 2 ;;
    --env-file) env_file="${2:-}"; shift 2 ;;
    --backup-dir) backup_dir="${2:-}"; shift 2 ;;
    --apply) apply=true; shift ;;
    -h|--help) usage; exit 0 ;;
    *)
      echo "[baota-compose-import-image-tars][error] unknown option: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

die() {
  echo "[baota-compose-import-image-tars][error] $*" >&2
  exit 1
}

require_nonempty() {
  local key="$1"
  local value="$2"
  [[ -n "${value}" ]] || die "${key} is required"
}

require_image_ref() {
  local key="$1"
  local value="$2"
  require_nonempty "${key}" "${value}"
  [[ "${value}" != REPLACE_WITH_* ]] || die "${key} still uses placeholder"
  [[ "${value}" != *":latest" ]] || die "${key} must not use :latest"
  [[ "${value}" == *":"* || "${value}" == *@sha256:* ]] || die "${key} must include a tag or digest"
}

require_sha256() {
  local key="$1"
  local value="$2"
  [[ "${value}" =~ ^[0-9a-f]{64}$ ]] || die "${key} must be a lowercase sha256 hex digest"
}

resolve_path() {
  local base="$1"
  local raw="$2"
  case "${raw}" in
    /*) printf '%s\n' "${raw}" ;;
    *) printf '%s/%s\n' "${base%/}" "${raw}" ;;
  esac
}

verify_tar() {
  local label="$1"
  local tar_path="$2"
  local expected_sha="$3"
  [[ -f "${tar_path}" ]] || die "${label} tar not found: ${tar_path}"
  printf '%s  %s\n' "${expected_sha}" "${tar_path}" | sha256sum -c -
}

docker_load_tar() {
  local tar_path="$1"
  local docker_bin="${DOCKER_BIN:-docker}"
  case "${tar_path}" in
    *.gz|*.tgz)
      gzip -dc "${tar_path}" | "${docker_bin}" load
      ;;
    *)
      "${docker_bin}" load -i "${tar_path}"
      ;;
  esac
}

require_image_ref BACKEND_IMAGE_REF "${backend_ref}"
require_image_ref FRONTEND_IMAGE_REF "${frontend_ref}"
require_image_ref ADMIN_IMAGE_REF "${admin_ref}"
require_nonempty BACKEND_TAR "${backend_tar}"
require_nonempty FRONTEND_TAR "${frontend_tar}"
require_nonempty ADMIN_TAR "${admin_tar}"
require_sha256 BACKEND_SHA256 "${backend_sha256}"
require_sha256 FRONTEND_SHA256 "${frontend_sha256}"
require_sha256 ADMIN_SHA256 "${admin_sha256}"

compose_root="$(cd "${compose_root}" && pwd)"
source_dir="$(resolve_path "${compose_root}" "${source_dir}")"
env_file="$(resolve_path "${compose_root}" "${env_file}")"
backup_dir="$(resolve_path "${compose_root}" "${backup_dir}")"
backend_tar="$(resolve_path "$(pwd)" "${backend_tar}")"
frontend_tar="$(resolve_path "$(pwd)" "${frontend_tar}")"
admin_tar="$(resolve_path "$(pwd)" "${admin_tar}")"

[[ -d "${source_dir}" ]] || die "source dir not found: ${source_dir}"
[[ -f "${env_file}" ]] || die "env file not found: ${env_file}"
refresh_script="${source_dir}/infra/ops/baota-compose-refresh-image-refs.sh"
[[ -x "${refresh_script}" ]] || die "missing executable refresh script: ${refresh_script}"

verify_tar backend "${backend_tar}" "${backend_sha256}"
verify_tar frontend "${frontend_tar}" "${frontend_sha256}"
verify_tar admin "${admin_tar}" "${admin_sha256}"

if [[ "${apply}" != "true" ]]; then
  echo "[baota-compose-import-image-tars] dry-run"
  echo "  backend:  ${backend_ref}"
  echo "  frontend: ${frontend_ref}"
  echo "  admin:    ${admin_ref}"
  echo "  env:      ${env_file}"
  echo "  compose:  ${compose_root}/docker-compose.yml"
  echo "[baota-compose-import-image-tars] rerun with --apply to load images and update refs"
  exit 0
fi

echo "[baota-compose-import-image-tars] loading backend image"
docker_load_tar "${backend_tar}"
echo "[baota-compose-import-image-tars] loading frontend image"
docker_load_tar "${frontend_tar}"
echo "[baota-compose-import-image-tars] loading admin image"
docker_load_tar "${admin_tar}"

python3 - "${env_file}" "${backend_ref}" "${frontend_ref}" "${admin_ref}" <<'PY'
from pathlib import Path
import sys

path = Path(sys.argv[1])
values = {
    "BACKEND_IMAGE_REF": sys.argv[2],
    "FRONTEND_IMAGE_REF": sys.argv[3],
    "ADMIN_IMAGE_REF": sys.argv[4],
}
lines = path.read_text().splitlines()
seen = set()
out = []
for line in lines:
    if "=" in line and not line.lstrip().startswith("#"):
        key = line.split("=", 1)[0].strip()
        if key in values:
            out.append(f"{key}={values[key]}")
            seen.add(key)
            continue
    out.append(line)
for key, value in values.items():
    if key not in seen:
        out.append(f"{key}={value}")
path.write_text("\n".join(out) + "\n")
PY

(
  cd "${compose_root}"
  "${refresh_script}" \
    --compose-file "${compose_root}/docker-compose.yml" \
    --env-file "${env_file}" \
    --backup-dir "${backup_dir}" \
    --apply
)

echo "[baota-compose-import-image-tars] imported and refreshed image refs"
