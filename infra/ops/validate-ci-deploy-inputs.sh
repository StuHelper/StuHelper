#!/usr/bin/env bash
set -euo pipefail

fail() {
  printf '[ci-deploy-inputs][error] %s\n' "$*" >&2
  exit 1
}

require_nonempty() {
  local name="$1"
  local value="${2:-}"
  [[ -n "${value}" ]] || fail "${name} is required"
}

validate_host() {
  local host="$1"
  local label
  local -a parts=()

  ((${#host} <= 253)) || fail "DEPLOY_TARGET_HOST exceeds 253 characters"
  [[ "${host}" =~ ^[A-Za-z0-9.-]+$ ]] ||
    fail "DEPLOY_TARGET_HOST must be a DNS hostname or IPv4 address"
  [[ "${host}" != .* && "${host}" != *. && "${host}" != *..* ]] ||
    fail "DEPLOY_TARGET_HOST has an invalid hostname structure"

  IFS='.' read -r -a parts <<<"${host}"
  if [[ "${host}" =~ ^[0-9.]+$ ]]; then
    ((${#parts[@]} == 4)) || fail "DEPLOY_TARGET_HOST is not a valid IPv4 address"
    for label in "${parts[@]}"; do
      [[ "${label}" =~ ^(0|[1-9][0-9]{0,2})$ ]] ||
        fail "DEPLOY_TARGET_HOST is not a canonical IPv4 address"
      ((10#${label} <= 255)) || fail "DEPLOY_TARGET_HOST contains an invalid IPv4 octet"
    done
    return
  fi

  for label in "${parts[@]}"; do
    [[ "${label}" =~ ^[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?$ ]] ||
      fail "DEPLOY_TARGET_HOST contains an invalid DNS label"
  done
}

validate_port() {
  local port="${1:-22}"
  [[ "${port}" =~ ^[0-9]{1,5}$ ]] || fail "DEPLOY_TARGET_PORT must be an integer"
  ((10#${port} >= 1 && 10#${port} <= 65535)) ||
    fail "DEPLOY_TARGET_PORT must be between 1 and 65535"
}

validate_user() {
  local user="$1"
  [[ "${user}" =~ ^[a-z_][a-z0-9_-]{0,31}$ ]] ||
    fail "DEPLOY_TARGET_USER must be a canonical Linux account name"
}

validate_app_dir() {
  local app_dir="$1"
  ((${#app_dir} <= 2048)) || fail "DEPLOY_TARGET_APP_DIR exceeds 2048 characters"
  [[ "${app_dir}" =~ ^/[A-Za-z0-9._/-]+$ ]] ||
    fail "DEPLOY_TARGET_APP_DIR must be an absolute path using safe characters"
  [[ "${app_dir}" != "/" && "${app_dir}" != */ && "${app_dir}" != *//* ]] ||
    fail "DEPLOY_TARGET_APP_DIR must identify one normalized application directory"
  [[ ! "${app_dir}" =~ (^|/)\.{1,2}(/|$) ]] ||
    fail "DEPLOY_TARGET_APP_DIR must not contain dot path segments"
}

require_nonempty DEPLOY_ENVIRONMENT "${DEPLOY_ENVIRONMENT:-}"
require_nonempty DEPLOY_TARGET_HOST "${DEPLOY_TARGET_HOST:-}"
require_nonempty DEPLOY_TARGET_USER "${DEPLOY_TARGET_USER:-}"
require_nonempty DEPLOY_TARGET_APP_DIR "${DEPLOY_TARGET_APP_DIR:-}"
require_nonempty DEPLOY_TARGET_SSH_KEY "${DEPLOY_TARGET_SSH_KEY:-}"
require_nonempty DEPLOY_TARGET_SSH_KNOWN_HOSTS "${DEPLOY_TARGET_SSH_KNOWN_HOSTS:-}"
require_nonempty TARGET_SHA "${TARGET_SHA:-}"

[[ "${DEPLOY_ENVIRONMENT}" == "staging" || "${DEPLOY_ENVIRONMENT}" == "production" ]] ||
  fail "DEPLOY_ENVIRONMENT must be staging or production"
[[ "${TARGET_SHA}" =~ ^[0-9a-f]{40}$ ]] ||
  fail "TARGET_SHA must be a full lowercase 40-character Git commit SHA"

validate_host "${DEPLOY_TARGET_HOST}"
validate_port "${DEPLOY_TARGET_PORT:-22}"
validate_user "${DEPLOY_TARGET_USER}"
validate_app_dir "${DEPLOY_TARGET_APP_DIR}"

printf '[ci-deploy-inputs] deployment inputs validated\n'
