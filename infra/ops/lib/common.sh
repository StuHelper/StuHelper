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
GENERATED_ZITADEL_DIR="${GENERATED_ZITADEL_DIR:-${REPO_ROOT}/infra/generated/zitadel}"
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

require_backup_object_storage_config() {
  local missing=()
  local key

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

ensure_env_file() {
  local template_file="${1:-${ENV_TEMPLATE_FILE}}"
  [[ -n "${template_file}" ]] || die "ENV_TEMPLATE_FILE must not be empty"
  [[ -f "${template_file}" ]] || die "missing env template: ${template_file}"
  if [[ ! -f "${ENV_FILE}" ]]; then
    cp "${template_file}" "${ENV_FILE}"
    log "created ${ENV_FILE} from $(basename "${template_file}")"
  fi
}

ensure_generated_files() {
  mkdir -p "${GENERATED_OBS_DIR}/prometheus" "${GENERATED_OBS_DIR}/alertmanager" "${GENERATED_ZITADEL_DIR}"
  touch "${GENERATED_ENV_FILE}"
  touch "${GENERATED_SECRET_ENV_FILE}"
}

materialize_secret_env_inputs() {
  if [[ -n "${SHARED_ENV_SECRET_REF}" ]]; then
    materialize_secret_env_file "${SHARED_ENV_SECRET_REF}" "${ENV_FILE}"
  fi
  if [[ -n "${SECRETS_ENV_SECRET_REF}" && -n "${SECRETS_ENV_FILE}" ]]; then
    materialize_secret_env_file "${SECRETS_ENV_SECRET_REF}" "${SECRETS_ENV_FILE}"
  fi
  if [[ -n "${GENERATED_ENV_SECRET_REF}" ]]; then
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
  # shellcheck disable=SC1090
  source "${ENV_FILE}"
  if [[ -n "${SECRETS_ENV_FILE}" && -f "${SECRETS_ENV_FILE}" ]]; then
    # shellcheck disable=SC1090
    source "${SECRETS_ENV_FILE}"
  fi
  # shellcheck disable=SC1090
  source "${GENERATED_ENV_FILE}"
  if [[ -f "${GENERATED_SECRET_ENV_FILE}" ]]; then
    # shellcheck disable=SC1090
    source "${GENERATED_SECRET_ENV_FILE}"
  fi
  if [[ "${preserved_tag}" != "__STUHELPER_UNSET__" ]]; then export TAG="${preserved_tag}"; fi
  if [[ "${preserved_rollback_tag}" != "__STUHELPER_UNSET__" ]]; then export ROLLBACK_TAG="${preserved_rollback_tag}"; fi
  if [[ "${preserved_backend_image_ref}" != "__STUHELPER_UNSET__" ]]; then export BACKEND_IMAGE_REF="${preserved_backend_image_ref}"; fi
  if [[ "${preserved_frontend_image_ref}" != "__STUHELPER_UNSET__" ]]; then export FRONTEND_IMAGE_REF="${preserved_frontend_image_ref}"; fi
  if [[ "${preserved_admin_image_ref}" != "__STUHELPER_UNSET__" ]]; then export ADMIN_IMAGE_REF="${preserved_admin_image_ref}"; fi
  set +a
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
  # shellcheck disable=SC1090
  source "${config_file}"
  if [[ "${preserved_tag}" != "__STUHELPER_UNSET__" ]]; then export TAG="${preserved_tag}"; fi
  if [[ "${preserved_rollback_tag}" != "__STUHELPER_UNSET__" ]]; then export ROLLBACK_TAG="${preserved_rollback_tag}"; fi
  if [[ "${preserved_backend_image_ref}" != "__STUHELPER_UNSET__" ]]; then export BACKEND_IMAGE_REF="${preserved_backend_image_ref}"; fi
  if [[ "${preserved_frontend_image_ref}" != "__STUHELPER_UNSET__" ]]; then export FRONTEND_IMAGE_REF="${preserved_frontend_image_ref}"; fi
  if [[ "${preserved_admin_image_ref}" != "__STUHELPER_UNSET__" ]]; then export ADMIN_IMAGE_REF="${preserved_admin_image_ref}"; fi
  if [[ "${preserved_registry_username}" != "__STUHELPER_UNSET__" ]]; then export REGISTRY_USERNAME="${preserved_registry_username}"; fi
  if [[ "${preserved_registry_password}" != "__STUHELPER_UNSET__" ]]; then export REGISTRY_PASSWORD="${preserved_registry_password}"; fi
  set +a
}

compose() {
  (
    cd "${REPO_ROOT}" && \
    preserved_tag="${TAG-__STUHELPER_UNSET__}" && \
    preserved_rollback_tag="${ROLLBACK_TAG-__STUHELPER_UNSET__}" && \
    preserved_backend_image_ref="${BACKEND_IMAGE_REF-__STUHELPER_UNSET__}" && \
    preserved_frontend_image_ref="${FRONTEND_IMAGE_REF-__STUHELPER_UNSET__}" && \
    preserved_admin_image_ref="${ADMIN_IMAGE_REF-__STUHELPER_UNSET__}" && \
    set -a && \
    source "${ENV_FILE}" && \
    if [[ -n "${SECRETS_ENV_FILE}" && -f "${SECRETS_ENV_FILE}" ]]; then source "${SECRETS_ENV_FILE}"; fi && \
    if [[ -f "${GENERATED_ENV_FILE}" ]]; then source "${GENERATED_ENV_FILE}"; fi && \
    if [[ -f "${GENERATED_SECRET_ENV_FILE}" ]]; then source "${GENERATED_SECRET_ENV_FILE}"; fi && \
    if [[ "${preserved_tag}" != "__STUHELPER_UNSET__" ]]; then export TAG="${preserved_tag}"; fi && \
    if [[ "${preserved_rollback_tag}" != "__STUHELPER_UNSET__" ]]; then export ROLLBACK_TAG="${preserved_rollback_tag}"; fi && \
    if [[ "${preserved_backend_image_ref}" != "__STUHELPER_UNSET__" ]]; then export BACKEND_IMAGE_REF="${preserved_backend_image_ref}"; fi && \
    if [[ "${preserved_frontend_image_ref}" != "__STUHELPER_UNSET__" ]]; then export FRONTEND_IMAGE_REF="${preserved_frontend_image_ref}"; fi && \
    if [[ "${preserved_admin_image_ref}" != "__STUHELPER_UNSET__" ]]; then export ADMIN_IMAGE_REF="${preserved_admin_image_ref}"; fi && \
    set +a && \
    ENV_FILE_PATH="${ENV_FILE}" \
    GENERATED_ENV_FILE_PATH="${GENERATED_ENV_FILE}" \
    BACKEND_IMAGE_REF="${BACKEND_IMAGE_REF:-}" \
    FRONTEND_IMAGE_REF="${FRONTEND_IMAGE_REF:-}" \
    ADMIN_IMAGE_REF="${ADMIN_IMAGE_REF:-}" \
    docker compose --env-file "${ENV_FILE}" "$@"
  )
}

build_zitadel_runtime_images() {
  local version="${ZITADEL_VERSION:-v4.13.0}"
  local base_image="ghcr.io/zitadel/zitadel:${version}"

  docker build \
    --build-arg "ZITADEL_IMAGE=${base_image}" \
    -f "${REPO_ROOT}/infra/zitadel/Dockerfile.init" \
    -t "stuhelper/zitadel-init:${version}" \
    "${REPO_ROOT}"

  docker build \
    --build-arg "ZITADEL_IMAGE=${base_image}" \
    -f "${REPO_ROOT}/infra/zitadel/Dockerfile.runtime" \
    -t "stuhelper/zitadel-api:${version}" \
    "${REPO_ROOT}"
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

  python3 - "$file" "$key" "$value" <<'PY'
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
  [[ -n "${REGISTRY:-}" ]] || die "REGISTRY is required"
  resolve_registry_credentials
  [[ -n "${REGISTRY_USERNAME:-}" ]] || die "REGISTRY_USERNAME or REGISTRY_USERNAME_SECRET_REF is required"
  [[ -n "${REGISTRY_PASSWORD:-}" ]] || die "REGISTRY_PASSWORD or REGISTRY_PASSWORD_SECRET_REF is required"

  echo "${REGISTRY_PASSWORD}" | docker login "${REGISTRY}" --username "${REGISTRY_USERNAME}" --password-stdin >/dev/null
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
