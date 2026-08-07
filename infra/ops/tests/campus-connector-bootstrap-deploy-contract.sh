#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
PROD_DEPLOY="${REPO_ROOT}/infra/ops/prod-deploy.sh"
tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT

fail() {
  printf '[campus-connector-bootstrap-deploy-contract][error] %s\n' "$*" >&2
  exit 1
}

# Load only the gateway ownership/bootstrap functions. Sourcing prod-deploy.sh
# itself would intentionally execute a real production deployment.
awk '
  /^campus_connector_gateway_owner=/ { capture=1 }
  /^resolve_env_path\(\)/ { capture=0 }
  capture { print }
' "${PROD_DEPLOY}" >"${tmpdir}/gateway-functions.sh"
[[ -s "${tmpdir}/gateway-functions.sh" ]] || fail "could not extract gateway deployment functions"
# shellcheck source=/dev/null
source "${tmpdir}/gateway-functions.sh"

APP_CONTAINER="stuhelper-test-app"
BOOTSTRAP_CONTAINER="stuhelper-test-campus-connector-bootstrap"
STACK_NAME="stuhelper-test"
CAMPUS_CONNECTOR_GATEWAY_ENABLED=true
CAMPUS_CONNECTOR_GATEWAY_EXTERNAL_PORT=19444

mock_app_running=false
mock_app_bound=false
mock_app_listening=false
mock_bootstrap_running=false
mock_bootstrap_bound=false
mock_bootstrap_listening=false
mock_published_container=""
mock_non_docker_listener=false
compose_calls=""
log_messages=""

reset_mocks() {
  mock_app_running=false
  mock_app_bound=false
  mock_app_listening=false
  mock_bootstrap_running=false
  mock_bootstrap_bound=false
  mock_bootstrap_listening=false
  mock_published_container=""
  mock_non_docker_listener=false
  compose_calls=""
  log_messages=""
  campus_connector_gateway_owner="disabled"
}

log() {
  log_messages+="$*"$'\n'
}

die() {
  printf '%s\n' "$*" >&2
  exit 97
}

docker() {
  local command="${1:-}"
  shift || true
  case "${command}" in
    inspect)
      if [[ "${1:-}" == "--format" ]]; then
        local container="${3:-}"
        case "${container}" in
          "${APP_CONTAINER}") printf '%s\n' "${mock_app_running}" ;;
          "${BOOTSTRAP_CONTAINER}") printf '%s\n' "${mock_bootstrap_running}" ;;
          *) return 1 ;;
        esac
        return 0
      fi
      local container="${1:-}"
      local bound=false
      case "${container}" in
        "${APP_CONTAINER}") bound="${mock_app_bound}" ;;
        "${BOOTSTRAP_CONTAINER}") bound="${mock_bootstrap_bound}" ;;
        *) return 1 ;;
      esac
      if [[ "${bound}" == "true" ]]; then
        printf '[{"HostConfig":{"PortBindings":{"9444/tcp":[{"HostIp":"127.0.0.1","HostPort":"19444"}]}}}]\n'
      else
        printf '[{"HostConfig":{"PortBindings":{}}}]\n'
      fi
      ;;
    exec)
      local container="${1:-}"
      case "${container}" in
        "${APP_CONTAINER}") [[ "${mock_app_listening}" == "true" ]] ;;
        "${BOOTSTRAP_CONTAINER}") [[ "${mock_bootstrap_listening}" == "true" ]] ;;
        *) return 1 ;;
      esac
      ;;
    ps)
      [[ -z "${mock_published_container}" ]] || printf '%s\n' "${mock_published_container}"
      ;;
    *)
      fail "unexpected docker command in contract: ${command} $*"
      ;;
  esac
}

compose() {
  compose_calls+="$*"$'\n'
  if [[ " $* " == *" up "* && " $* " == *" campus-connector-bootstrap "* ]]; then
    mock_bootstrap_running=true
    mock_bootstrap_bound=true
    mock_bootstrap_listening=true
    mock_published_container="${BOOTSTRAP_CONTAINER}"
  elif [[ " $* " == *" stop "* && " $* " == *" campus-connector-bootstrap "* ]]; then
    mock_bootstrap_running=false
    mock_bootstrap_listening=false
    mock_published_container=""
  elif [[ " $* " == *" rm "* && " $* " == *" campus-connector-bootstrap "* ]]; then
    :
  else
    fail "unexpected compose command in contract: $*"
  fi
}

host_has_non_docker_campus_connector_listener() {
  [[ "${mock_non_docker_listener}" == "true" ]]
}

reset_mocks
mock_app_running=true
mock_app_bound=true
mock_app_listening=true
mock_published_container="${APP_CONTAINER}"
prepare_campus_connector_gateway_for_readiness
[[ "${campus_connector_gateway_owner}" == "app" ]] || fail "healthy running app was not reused"
[[ -z "${compose_calls}" ]] || fail "healthy running app must not start bootstrap"

reset_mocks
mock_app_running=true
mock_app_bound=true
mock_app_listening=false
mock_published_container="${APP_CONTAINER}"
if (prepare_campus_connector_gateway_for_readiness >/dev/null 2>&1); then
  fail "mapped but non-listening production app must fail closed"
fi
[[ -z "${compose_calls}" ]] || fail "unhealthy production app must not be stopped or replaced"

reset_mocks
mock_published_container="unrelated-service"
if (prepare_campus_connector_gateway_for_readiness >/dev/null 2>&1); then
  fail "unexpected Docker port owner must fail closed"
fi
[[ -z "${compose_calls}" ]] || fail "unexpected Docker port owner must not be changed"

reset_mocks
mock_non_docker_listener=true
if (prepare_campus_connector_gateway_for_readiness >/dev/null 2>&1); then
  fail "non-Compose port owner must fail closed"
fi
[[ -z "${compose_calls}" ]] || fail "non-Compose port owner must not be changed"

reset_mocks
prepare_campus_connector_gateway_for_readiness
[[ "${campus_connector_gateway_owner}" == "bootstrap" ]] || fail "first rollout did not start bootstrap"
[[ "${mock_bootstrap_running}" == "true" ]] || fail "bootstrap did not remain running for readiness"
grep -qF 'up -d --wait --no-deps campus-connector-bootstrap' <<<"${compose_calls}" ||
  fail "bootstrap must start without implicitly rerunning dependencies"

# Simulate a failed readiness by intentionally not calling handoff. The gateway
# must remain available for real heartbeats and a later signed roster upload.
[[ "${mock_bootstrap_running}" == "true" ]] || fail "readiness failure would remove bootstrap"

handoff_campus_connector_gateway_to_app
[[ "${campus_connector_gateway_owner}" == "released" ]] || fail "successful handoff did not release ownership"
[[ "${mock_bootstrap_running}" == "false" ]] || fail "successful handoff left bootstrap running"
stop_line="$(grep -nF 'stop campus-connector-bootstrap' <<<"${compose_calls}" | cut -d: -f1)"
rm_line="$(grep -nF 'rm -f campus-connector-bootstrap' <<<"${compose_calls}" | cut -d: -f1)"
[[ -n "${stop_line}" && -n "${rm_line}" && "${stop_line}" -lt "${rm_line}" ]] ||
  fail "handoff must stop bootstrap before removing it"

reset_mocks
CAMPUS_CONNECTOR_GATEWAY_ENABLED=false
prepare_campus_connector_gateway_for_readiness
[[ "${campus_connector_gateway_owner}" == "disabled" ]] || fail "disabled gateway changed ownership"
[[ -z "${compose_calls}" ]] || fail "disabled gateway must not start bootstrap"

printf '[campus-connector-bootstrap-deploy-contract] all assertions passed\n'
