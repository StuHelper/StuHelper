#!/usr/bin/env bash
set -euo pipefail

COMMON_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${COMMON_LIB_DIR}/../../.." && pwd)"
ENV_FILE="${ENV_FILE:-${REPO_ROOT}/.env}"
SECRETS_ENV_FILE="${SECRETS_ENV_FILE:-}"
GENERATED_ENV_FILE="${GENERATED_ENV_FILE:-${REPO_ROOT}/.env.generated}"
GENERATED_OBS_DIR="${GENERATED_OBS_DIR:-${REPO_ROOT}/infra/generated/observability}"
DEPLOY_STATE_DIR="${DEPLOY_STATE_DIR:-${REPO_ROOT}/.deploy}"

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

ensure_env_file() {
  if [[ ! -f "${ENV_FILE}" ]]; then
    cp "${REPO_ROOT}/.env.example" "${ENV_FILE}"
    log "created ${ENV_FILE} from .env.example"
  fi
}

ensure_generated_files() {
  mkdir -p "${GENERATED_OBS_DIR}/prometheus" "${GENERATED_OBS_DIR}/alertmanager"
  touch "${GENERATED_ENV_FILE}"
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
  set -a
  # shellcheck disable=SC1090
  source "${ENV_FILE}"
  if [[ -n "${SECRETS_ENV_FILE}" && -f "${SECRETS_ENV_FILE}" ]]; then
    # shellcheck disable=SC1090
    source "${SECRETS_ENV_FILE}"
  fi
  # shellcheck disable=SC1090
  source "${GENERATED_ENV_FILE}"
  set +a
}

compose() {
  (
    cd "${REPO_ROOT}" && \
    set -a && \
    source "${ENV_FILE}" && \
    if [[ -n "${SECRETS_ENV_FILE}" && -f "${SECRETS_ENV_FILE}" ]]; then source "${SECRETS_ENV_FILE}"; fi && \
    if [[ -f "${GENERATED_ENV_FILE}" ]]; then source "${GENERATED_ENV_FILE}"; fi && \
    set +a && \
    ENV_FILE_PATH="${ENV_FILE}" \
    GENERATED_ENV_FILE_PATH="${GENERATED_ENV_FILE}" \
    docker compose --env-file "${ENV_FILE}" "$@"
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

record_release() {
  local tag="$1"
  mkdir -p "${DEPLOY_STATE_DIR}"
  local now
  now="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  printf '%s\t%s\n' "${now}" "${tag}" >> "${DEPLOY_STATE_DIR}/releases.log"
  printf 'TAG=%s\nDEPLOYED_AT=%s\n' "${tag}" "${now}" > "${DEPLOY_STATE_DIR}/current-release.env"
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
