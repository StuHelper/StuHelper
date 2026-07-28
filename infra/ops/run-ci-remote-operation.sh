#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
VALIDATOR="${SCRIPT_DIR}/validate-ci-deploy-inputs.sh"

fail() {
  printf '[ci-remote-operation][error] %s\n' "$*" >&2
  exit 1
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "required command is unavailable: $1"
}

validate_digest_ref() {
  local name="$1"
  local value="${2:-}"

  [[ "${value}" =~ ^[A-Za-z0-9._:/-]+@sha256:[0-9a-f]{64}$ ]] ||
    fail "${name} must be an immutable OCI image digest reference"
}

operation="${1:-}"
case "${operation}" in
  deploy | rollback | verify) ;;
  *) fail "usage: $0 deploy|rollback|verify" ;;
esac

require_cmd bash
require_cmd ssh
require_cmd scp
require_cmd ssh-add
require_cmd ssh-agent
require_cmd ssh-keygen

"${VALIDATOR}"

case "${operation}" in
  deploy)
    validate_digest_ref BACKEND_IMAGE_REF "${BACKEND_IMAGE_REF:-}"
    validate_digest_ref FRONTEND_IMAGE_REF "${FRONTEND_IMAGE_REF:-}"
    validate_digest_ref ADMIN_IMAGE_REF "${ADMIN_IMAGE_REF:-}"
    ;;
  rollback)
    validate_digest_ref ROLLBACK_BACKEND_IMAGE_REF "${ROLLBACK_BACKEND_IMAGE_REF:-}"
    validate_digest_ref ROLLBACK_FRONTEND_IMAGE_REF "${ROLLBACK_FRONTEND_IMAGE_REF:-}"
    validate_digest_ref ROLLBACK_ADMIN_IMAGE_REF "${ROLLBACK_ADMIN_IMAGE_REF:-}"
    ;;
esac

port="${DEPLOY_TARGET_PORT:-22}"
ssh_dir="${HOME}/.ssh"
known_hosts_file="${ssh_dir}/known_hosts"
bundle_path="${CI_PROJECT_DIR:-${REPO_ROOT}}/deploy-bundle.tgz"

eval "$(ssh-agent -s)"
cleanup() {
  ssh-agent -k >/dev/null 2>&1 || true
}
trap cleanup EXIT

install -d -m 700 "${ssh_dir}"
printf '%s\n' "${DEPLOY_TARGET_SSH_KEY}" | tr -d '\r' | ssh-add -
printf '%s\n' "${DEPLOY_TARGET_SSH_KNOWN_HOSTS}" |
  tr -d '\r' >"${known_hosts_file}"
chmod 600 "${known_hosts_file}"

known_host_lookup="${DEPLOY_TARGET_HOST}"
if [[ "${port}" != "22" ]]; then
  known_host_lookup="[${DEPLOY_TARGET_HOST}]:${port}"
fi
if ! ssh-keygen -F "${known_host_lookup}" -f "${known_hosts_file}" >/dev/null; then
  fail "DEPLOY_TARGET_SSH_KNOWN_HOSTS does not pin the selected endpoint"
fi

ssh_args=(
  -o BatchMode=yes
  -o StrictHostKeyChecking=yes
  -p "${port}"
)
remote="${DEPLOY_TARGET_USER}@${DEPLOY_TARGET_HOST}"

if [[ "${operation}" == "verify" ]]; then
  # DEPLOY_TARGET_APP_DIR is validated before it reaches the remote shell.
  # shellcheck disable=SC2029
  ssh "${ssh_args[@]}" "${remote}" \
    "set -euo pipefail &&
     cd '${DEPLOY_TARGET_APP_DIR}' &&
     ./infra/ops/smoke-check.sh &&
     ./infra/ops/observability-smoke-check.sh"
  printf '[ci-remote-operation] remote verification completed\n'
  exit 0
fi

"${SCRIPT_DIR}/build-deploy-bundle.sh" "${bundle_path}"

# DEPLOY_TARGET_APP_DIR is validated before it reaches the remote shell.
# shellcheck disable=SC2029
ssh "${ssh_args[@]}" "${remote}" "mkdir -p '${DEPLOY_TARGET_APP_DIR}'"
scp \
  -o BatchMode=yes \
  -o StrictHostKeyChecking=yes \
  -P "${port}" \
  "${bundle_path}" \
  "${remote}:${DEPLOY_TARGET_APP_DIR}/deploy-bundle.tgz"

if [[ "${operation}" == "deploy" ]]; then
  # Every interpolated value has passed a strict allowlist validation.
  # shellcheck disable=SC2029
  ssh "${ssh_args[@]}" "${remote}" \
    "set -euo pipefail &&
     cd '${DEPLOY_TARGET_APP_DIR}' &&
     tar xzf deploy-bundle.tgz &&
     rm -f deploy-bundle.tgz &&
     TAG='${TARGET_SHA}' \
     BACKEND_IMAGE_REF='${BACKEND_IMAGE_REF}' \
     FRONTEND_IMAGE_REF='${FRONTEND_IMAGE_REF}' \
     ADMIN_IMAGE_REF='${ADMIN_IMAGE_REF}' \
     ./infra/ops/remote-preflight.sh &&
     TAG='${TARGET_SHA}' \
     BACKEND_IMAGE_REF='${BACKEND_IMAGE_REF}' \
     FRONTEND_IMAGE_REF='${FRONTEND_IMAGE_REF}' \
     ADMIN_IMAGE_REF='${ADMIN_IMAGE_REF}' \
     ./infra/ops/remote-prod-deploy.sh"
else
  # The rollback release and all image references are immutable and validated.
  # shellcheck disable=SC2029
  ssh "${ssh_args[@]}" "${remote}" \
    "set -euo pipefail &&
     cd '${DEPLOY_TARGET_APP_DIR}' &&
     tar xzf deploy-bundle.tgz &&
     rm -f deploy-bundle.tgz &&
     TAG='${TARGET_SHA}' \
     ROLLBACK_TAG='${TARGET_SHA}' \
     ROLLBACK_BACKEND_IMAGE_REF='${ROLLBACK_BACKEND_IMAGE_REF}' \
     ROLLBACK_FRONTEND_IMAGE_REF='${ROLLBACK_FRONTEND_IMAGE_REF}' \
     ROLLBACK_ADMIN_IMAGE_REF='${ROLLBACK_ADMIN_IMAGE_REF}' \
     ./infra/ops/remote-prod-rollback.sh &&
     ./infra/ops/smoke-check.sh &&
     ./infra/ops/observability-smoke-check.sh"
fi

printf '[ci-remote-operation] %s completed\n' "${operation}"
