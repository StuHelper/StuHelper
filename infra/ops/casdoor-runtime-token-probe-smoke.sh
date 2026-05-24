#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

usage() {
  cat <<'USAGE'
Usage: infra/ops/casdoor-runtime-token-probe-smoke.sh

Runs the automatic Casdoor authorization-code runtime token minimization probe
against the dedicated smoke application created by bootstrap-platform.sh.
The script fails if returned ID/access token claims include business claims.

Required env:
  CASDOOR_ISSUER
  CASDOOR_TOKEN_PROBE_USERNAME
  CASDOOR_TOKEN_PROBE_PASSWORD
  CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_ID
  CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_SECRET
  CASDOOR_TOKEN_PROBE_SMOKE_APPLICATION
  CASDOOR_TOKEN_PROBE_SMOKE_REDIRECT_URI

Optional env:
  ENV_FILE / SECRETS_ENV_FILE / GENERATED_ENV_FILE
  OPEN_PLATFORM_TOKEN_PROBE_SMOKE_MODE              host or container; defaults to container in production
  OPEN_PLATFORM_TOKEN_PROBE_SMOKE_COMMAND           host-mode runner; defaults to infra/ops runner
  OPEN_PLATFORM_TOKEN_PROBE_SMOKE_CONTAINER_ENTRYPOINT
  CASDOOR_TOKEN_PROBE_BROWSER_EXECUTABLE_PATH
  CASDOOR_TOKEN_PROBE_BROWSER_HEADLESS
  CASDOOR_TOKEN_PROBE_BROWSER_NO_SANDBOX
  CASDOOR_TOKEN_PROBE_LOGIN_TIMEOUT_SECONDS
  CASDOOR_TOKEN_PROBE_USERNAME_SELECTOR
  CASDOOR_TOKEN_PROBE_PASSWORD_SELECTOR
  CASDOOR_TOKEN_PROBE_SUBMIT_SELECTOR
  CASDOOR_TOKEN_PROBE_CONSENT_SELECTOR
USAGE
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
  exit 0
fi

require_cmd jq

load_env

require_env() {
  local missing=()
  local key
  for key in "$@"; do
    if [[ -z "${!key:-}" ]]; then
      missing+=("${key}")
    fi
  done
  if (( ${#missing[@]} > 0 )); then
    die "missing required Casdoor runtime token probe smoke env: ${missing[*]}"
  fi
}

require_env \
  CASDOOR_ISSUER \
  CASDOOR_TOKEN_PROBE_USERNAME \
  CASDOOR_TOKEN_PROBE_PASSWORD \
  CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_ID \
  CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_SECRET \
  CASDOOR_TOKEN_PROBE_SMOKE_APPLICATION \
  CASDOOR_TOKEN_PROBE_SMOKE_REDIRECT_URI

if [[ "${APP_ENV:-}" == "production" ]]; then
  [[ "${OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_REQUIRED:-false}" == "true" ]] || \
    die "OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_REQUIRED must be true before running production token probe smoke"
fi

build_payload() {
  jq -cn \
    --arg issuer "${CASDOOR_ISSUER}" \
    --arg app "${CASDOOR_TOKEN_PROBE_SMOKE_APPLICATION}" \
    --arg client_id "${CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_ID}" \
    --arg client_secret "${CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_SECRET}" \
    --arg redirect_uri "${CASDOOR_TOKEN_PROBE_SMOKE_REDIRECT_URI}" \
    '{
      issuer: $issuer,
      casdoorApplicationName: $app,
      clientID: $client_id,
      clientSecret: $client_secret,
      redirectURIs: [$redirect_uri],
      scope: "openid"
    }'
}

smoke_mode() {
  if [[ -n "${OPEN_PLATFORM_TOKEN_PROBE_SMOKE_MODE:-}" ]]; then
    printf '%s\n' "${OPEN_PLATFORM_TOKEN_PROBE_SMOKE_MODE}"
    return
  fi
  if [[ "${APP_ENV:-}" == "production" ]]; then
    printf '%s\n' "container"
    return
  fi
  printf '%s\n' "host"
}

resolve_host_runner() {
  local runner="${OPEN_PLATFORM_TOKEN_PROBE_SMOKE_COMMAND:-}"
  if [[ -z "${runner}" ]]; then
    if [[ -n "${OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND:-}" && "${OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND}" != /app/* ]]; then
      runner="${OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND}"
    else
      runner="${SCRIPT_DIR}/casdoor-runtime-token-probe-runner.mjs"
    fi
  fi
  if [[ "${runner}" != */* ]]; then
    command -v "${runner}" || return 1
    return 0
  fi
  if [[ "${runner}" != /* ]]; then
    runner="${REPO_ROOT}/${runner}"
  fi
  printf '%s\n' "${runner}"
}

run_host_smoke() {
  local runner
  runner="$(resolve_host_runner)" || die "host token probe smoke runner was not found"
  [[ -x "${runner}" ]] || die "host token probe smoke runner is not executable: ${runner}"
  env \
    -u CASDOOR_TOKEN_PROBE_CLIENT_ID \
    -u CASDOOR_TOKEN_PROBE_CLIENT_SECRET \
    -u CASDOOR_TOKEN_PROBE_REDIRECT_URI \
    -u CASDOOR_TOKEN_PROBE_AUTH_CODE \
    -u CASDOOR_TOKEN_PROBE_CODE_VERIFIER \
    CASDOOR_ISSUER="${CASDOOR_ISSUER}" \
    CASDOOR_TOKEN_PROBE_USERNAME="${CASDOOR_TOKEN_PROBE_USERNAME}" \
    CASDOOR_TOKEN_PROBE_PASSWORD="${CASDOOR_TOKEN_PROBE_PASSWORD}" \
    CASDOOR_TOKEN_PROBE_SCOPE="openid" \
    CASDOOR_TOKEN_PROBE_OUTPUT="json" \
    CASDOOR_TOKEN_PROBE_BROWSER_HEADLESS="${CASDOOR_TOKEN_PROBE_BROWSER_HEADLESS:-true}" \
    CASDOOR_TOKEN_PROBE_BROWSER_EXECUTABLE_PATH="${CASDOOR_TOKEN_PROBE_BROWSER_EXECUTABLE_PATH:-}" \
    CASDOOR_TOKEN_PROBE_BROWSER_NO_SANDBOX="${CASDOOR_TOKEN_PROBE_BROWSER_NO_SANDBOX:-true}" \
    CASDOOR_TOKEN_PROBE_LOGIN_TIMEOUT_SECONDS="${CASDOOR_TOKEN_PROBE_LOGIN_TIMEOUT_SECONDS:-30}" \
    CASDOOR_TOKEN_PROBE_USERNAME_SELECTOR="${CASDOOR_TOKEN_PROBE_USERNAME_SELECTOR:-}" \
    CASDOOR_TOKEN_PROBE_PASSWORD_SELECTOR="${CASDOOR_TOKEN_PROBE_PASSWORD_SELECTOR:-}" \
    CASDOOR_TOKEN_PROBE_SUBMIT_SELECTOR="${CASDOOR_TOKEN_PROBE_SUBMIT_SELECTOR:-}" \
    CASDOOR_TOKEN_PROBE_CONSENT_SELECTOR="${CASDOOR_TOKEN_PROBE_CONSENT_SELECTOR:-}" \
    "${runner}"
}

run_container_smoke() {
  require_cmd docker
  local entrypoint="${OPEN_PLATFORM_TOKEN_PROBE_SMOKE_CONTAINER_ENTRYPOINT:-${OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND:-/app/casdoor-runtime-token-probe-runner.mjs}}"
  compose --profile prod run --rm --no-deps -T \
    --entrypoint "${entrypoint}" \
    -e CASDOOR_ISSUER="${CASDOOR_ISSUER}" \
    -e CASDOOR_TOKEN_PROBE_CLIENT_ID= \
    -e CASDOOR_TOKEN_PROBE_CLIENT_SECRET= \
    -e CASDOOR_TOKEN_PROBE_REDIRECT_URI= \
    -e CASDOOR_TOKEN_PROBE_AUTH_CODE= \
    -e CASDOOR_TOKEN_PROBE_CODE_VERIFIER= \
    -e CASDOOR_TOKEN_PROBE_USERNAME="${CASDOOR_TOKEN_PROBE_USERNAME}" \
    -e CASDOOR_TOKEN_PROBE_PASSWORD="${CASDOOR_TOKEN_PROBE_PASSWORD}" \
    -e CASDOOR_TOKEN_PROBE_SCOPE="openid" \
    -e CASDOOR_TOKEN_PROBE_OUTPUT="json" \
    -e CASDOOR_TOKEN_PROBE_BROWSER_HEADLESS="${CASDOOR_TOKEN_PROBE_BROWSER_HEADLESS:-true}" \
    -e CASDOOR_TOKEN_PROBE_BROWSER_EXECUTABLE_PATH="${CASDOOR_TOKEN_PROBE_BROWSER_EXECUTABLE_PATH:-}" \
    -e CASDOOR_TOKEN_PROBE_BROWSER_NO_SANDBOX="${CASDOOR_TOKEN_PROBE_BROWSER_NO_SANDBOX:-true}" \
    -e CASDOOR_TOKEN_PROBE_LOGIN_TIMEOUT_SECONDS="${CASDOOR_TOKEN_PROBE_LOGIN_TIMEOUT_SECONDS:-30}" \
    -e CASDOOR_TOKEN_PROBE_USERNAME_SELECTOR="${CASDOOR_TOKEN_PROBE_USERNAME_SELECTOR:-}" \
    -e CASDOOR_TOKEN_PROBE_PASSWORD_SELECTOR="${CASDOOR_TOKEN_PROBE_PASSWORD_SELECTOR:-}" \
    -e CASDOOR_TOKEN_PROBE_SUBMIT_SELECTOR="${CASDOOR_TOKEN_PROBE_SUBMIT_SELECTOR:-}" \
    -e CASDOOR_TOKEN_PROBE_CONSENT_SELECTOR="${CASDOOR_TOKEN_PROBE_CONSENT_SELECTOR:-}" \
    app
}

validate_evidence() {
  jq -e '
    .method == "authorization_code"
    and (.inspectedClaims | type == "array" and length > 0)
    and (.businessClaims | type == "array" and length == 0)
    and (.tokenClaims | type == "object")
    and (.metadata.source == "casdoor-runtime-token-probe-runner.mjs")
    and (.metadata.nonceVerified == true)
  ' >/dev/null
}

payload="$(build_payload)"
mode="$(smoke_mode)"

log "running Casdoor runtime token probe smoke in ${mode} mode against ${CASDOOR_ISSUER}" >&2
case "${mode}" in
  host)
    if ! evidence="$(printf '%s\n' "${payload}" | run_host_smoke)"; then
      die "Casdoor runtime token probe smoke failed in host mode"
    fi
    ;;
  container)
    if ! evidence="$(printf '%s\n' "${payload}" | run_container_smoke)"; then
      die "Casdoor runtime token probe smoke failed in container mode"
    fi
    ;;
  *)
    die "OPEN_PLATFORM_TOKEN_PROBE_SMOKE_MODE must be host or container"
    ;;
esac

if ! printf '%s\n' "${evidence}" | validate_evidence; then
  printf '%s\n' "${evidence}" | jq . >&2 || true
  die "Casdoor runtime token probe smoke evidence did not pass minimization checks"
fi

printf '%s\n' "${evidence}" | jq .
