#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

usage() {
  cat <<'USAGE'
Usage: infra/ops/open-platform-production-evidence.sh

Runs the two Open Platform production evidence smokes and writes a sanitized
JSON evidence bundle:

  1. Casdoor authorization-code runtime token minimization smoke
  2. OpenFGA app -> resource_item grant/check/list/revoke smoke

Required env is inherited from the two underlying smoke scripts.

Optional env:
  OPEN_PLATFORM_PRODUCTION_EVIDENCE_FILE
      Output path. Defaults to infra/generated/open-platform-production-evidence.json.
      Set to "-" to only print the JSON bundle.
  OPEN_PLATFORM_EVIDENCE_CASDOOR_SMOKE_COMMAND
      Test/custom override for the Casdoor smoke command.
  OPEN_PLATFORM_EVIDENCE_OPENFGA_SMOKE_COMMAND
      Test/custom override for the OpenFGA smoke command.
  OPEN_PLATFORM_PRODUCTION_EVIDENCE_ALLOW_LOCAL_TARGETS
      Defaults to false. When false, rejects localhost-style Casdoor/OpenFGA
      targets so local validation cannot be archived as production evidence.
USAGE
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
  exit 0
fi

require_cmd jq
require_cmd python3

preserved_allow_local_targets="${OPEN_PLATFORM_PRODUCTION_EVIDENCE_ALLOW_LOCAL_TARGETS-__STUHELPER_UNSET__}"
load_env
if [[ "${preserved_allow_local_targets}" != "__STUHELPER_UNSET__" ]]; then
  OPEN_PLATFORM_PRODUCTION_EVIDENCE_ALLOW_LOCAL_TARGETS="${preserved_allow_local_targets}"
fi

evidence_file="${OPEN_PLATFORM_PRODUCTION_EVIDENCE_FILE:-${REPO_ROOT}/infra/generated/open-platform-production-evidence.json}"
casdoor_smoke_command="${OPEN_PLATFORM_EVIDENCE_CASDOOR_SMOKE_COMMAND:-${SCRIPT_DIR}/casdoor-runtime-token-probe-smoke.sh}"
openfga_smoke_command="${OPEN_PLATFORM_EVIDENCE_OPENFGA_SMOKE_COMMAND:-${SCRIPT_DIR}/openfga-resource-access-smoke.sh}"
expected_casdoor_issuer="$(trim_trailing_slash "${CASDOOR_ISSUER:-}")"
expected_openfga_api_url="$(trim_trailing_slash "${OPENFGA_API_URL:-}")"
expected_openfga_store_id="${OPENFGA_STORE_ID:-}"
expected_openfga_model_id="${OPENFGA_MODEL_ID:-}"
expected_runtime_probe_required="${OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_REQUIRED:-}"
expected_runtime_probe_command="${OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND:-}"
allow_local_targets="${OPEN_PLATFORM_PRODUCTION_EVIDENCE_ALLOW_LOCAL_TARGETS:-false}"

[[ -n "${expected_casdoor_issuer}" ]] || die "CASDOOR_ISSUER is required"
[[ -n "${expected_openfga_api_url}" ]] || die "OPENFGA_API_URL is required"

case "${allow_local_targets}" in
  true|false) ;;
  *) die "OPEN_PLATFORM_PRODUCTION_EVIDENCE_ALLOW_LOCAL_TARGETS must be true or false" ;;
esac

reject_local_evidence_target() {
  local name="$1"
  local value="${2:-}"
  local normalized="${value,,}"
  [[ -n "${value}" ]] || return 0
  case "${normalized}" in
    *localhost*|*127.0.0.1*|*::1*|*host.docker.internal*)
      die "${name} points to a local target (${value}); open-platform-production-evidence verifies production Casdoor/OpenFGA evidence. Set OPEN_PLATFORM_PRODUCTION_EVIDENCE_ALLOW_LOCAL_TARGETS=true only for local contract tests or intentional local production validation."
      ;;
  esac
}

if [[ "${allow_local_targets}" != "true" ]]; then
  reject_local_evidence_target "CASDOOR_ISSUER" "${expected_casdoor_issuer}"
  reject_local_evidence_target "CASDOOR_TOKEN_PROBE_SMOKE_REDIRECT_URI" "${CASDOOR_TOKEN_PROBE_SMOKE_REDIRECT_URI:-}"
  reject_local_evidence_target "OPENFGA_API_URL" "${expected_openfga_api_url}"
fi

[[ -n "${expected_openfga_store_id}" ]] || die "OPENFGA_STORE_ID is required"
[[ -n "${expected_openfga_model_id}" ]] || die "OPENFGA_MODEL_ID is required"
[[ "${expected_runtime_probe_required}" == "true" ]] || die "OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_REQUIRED must be true for Open Platform production evidence"
[[ -n "${expected_runtime_probe_command}" ]] || die "OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND is required for Open Platform production evidence"
[[ "${expected_runtime_probe_command}" != "REPLACE_WITH_OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND" ]] || die "OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND is still the placeholder value (REPLACE_WITH_OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND); set the real production runtime token probe runner before generating Open Platform production evidence"

resolve_command() {
  local command_path="$1"
  if [[ "${command_path}" != */* ]]; then
    command -v "${command_path}" || return 1
    return 0
  fi
  if [[ "${command_path}" != /* ]]; then
    command_path="${REPO_ROOT}/${command_path}"
  fi
  [[ -x "${command_path}" ]] || return 1
  printf '%s\n' "${command_path}"
}

run_json_smoke() {
  local label="$1"
  local configured_command="$2"
  local command_path
  command_path="$(resolve_command "${configured_command}")" || die "${label} command was not found or is not executable: ${configured_command}"
  log "running ${label}: ${command_path}" >&2
  "${command_path}"
}

validate_casdoor_evidence() {
  jq -e --arg issuer "${expected_casdoor_issuer}" '
    .method == "authorization_code"
    and (.issuer | type == "string" and rtrimstr("/") == $issuer)
    and (.businessClaims | type == "array" and length == 0)
    and (.inspectedClaims | type == "array" and length > 0)
    and (.tokenClaims | type == "object")
    and (.metadata.source == "casdoor-runtime-token-probe-runner.mjs")
    and (.metadata.nonceVerified == true)
  ' >/dev/null
}

validate_openfga_evidence() {
  jq -e \
    --arg api_url "${expected_openfga_api_url}" \
    --arg store_id "${expected_openfga_store_id}" \
    --arg model_id "${expected_openfga_model_id}" '
    .listedReadGrant == true
    and .listedReadAfterRevoke == false
    and .readAfterGrant == true
    and .writeAfterGrant == true
    and .readAfterRevoke == false
    and .writeAfterRevoke == false
    and (.apiURL | type == "string" and rtrimstr("/") == $api_url)
    and .storeID == $store_id
    and .modelID == $model_id
  ' >/dev/null
}

print_casdoor_diagnostic() {
  jq '{
    method,
    issuer,
    inspectedClaims: (.inspectedClaims // []),
    businessClaims: (.businessClaims // []),
    tokenClaimTypes: ((.tokenClaims // {}) | keys | sort),
    metadata: {
      source: (.metadata.source // null),
      capture: (.metadata.capture // null),
      nonceVerified: (.metadata.nonceVerified // false)
    }
  }' >&2
}

print_openfga_diagnostic() {
  jq '{
    apiURL,
    storeID,
    modelID,
    appObject,
    resourceObject,
    listedReadGrant,
    listedReadAfterRevoke,
    readAfterGrant,
    writeAfterGrant,
    readAfterRevoke,
    writeAfterRevoke
  }' >&2
}

casdoor_evidence="$(run_json_smoke "Casdoor runtime token minimization smoke" "${casdoor_smoke_command}")"
if ! printf '%s\n' "${casdoor_evidence}" | validate_casdoor_evidence; then
  printf '%s\n' "${casdoor_evidence}" | print_casdoor_diagnostic || true
  die "Casdoor runtime token minimization evidence did not pass production checks"
fi

openfga_evidence="$(run_json_smoke "OpenFGA resource access smoke" "${openfga_smoke_command}")"
if ! printf '%s\n' "${openfga_evidence}" | validate_openfga_evidence; then
  printf '%s\n' "${openfga_evidence}" | print_openfga_diagnostic || true
  die "OpenFGA resource access evidence did not pass production checks"
fi

generated_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
bundle="$(
  CASDOOR_EVIDENCE="${casdoor_evidence}" \
  OPENFGA_EVIDENCE="${openfga_evidence}" \
  python3 - "${generated_at}" "${APP_ENV:-}" <<'PY'
import json
import os
import sys

generated_at = sys.argv[1]
app_env = sys.argv[2]
casdoor = json.loads(os.environ["CASDOOR_EVIDENCE"])
openfga = json.loads(os.environ["OPENFGA_EVIDENCE"])

bundle = {
    "generatedAt": generated_at,
    "appEnv": app_env,
    "passed": True,
    "summary": {
        "casdoorRuntimeTokenProbe": True,
        "openfgaResourceAccessSmoke": True,
        "runtimeTokenProbeRequired": True,
    },
    "configuration": {
        "runtimeTokenProbeRequired": True,
        "runtimeTokenProbeCommandConfigured": True,
    },
    "casdoorRuntimeTokenProbe": {
        "passed": True,
        "method": casdoor.get("method"),
        "issuer": casdoor.get("issuer"),
        "inspectedClaims": casdoor.get("inspectedClaims", []),
        "businessClaims": casdoor.get("businessClaims", []),
        "tokenClaimTypes": sorted((casdoor.get("tokenClaims") or {}).keys()),
        "metadata": {
            "source": (casdoor.get("metadata") or {}).get("source"),
            "capture": (casdoor.get("metadata") or {}).get("capture"),
            "nonceVerified": (casdoor.get("metadata") or {}).get("nonceVerified"),
        },
    },
    "openfgaResourceAccessSmoke": {
        "passed": True,
        "apiURL": openfga.get("apiURL"),
        "storeID": openfga.get("storeID"),
        "modelID": openfga.get("modelID"),
        "appObject": openfga.get("appObject"),
        "resourceObject": openfga.get("resourceObject"),
        "listedReadGrant": openfga.get("listedReadGrant"),
        "listedReadAfterRevoke": openfga.get("listedReadAfterRevoke"),
        "readAfterGrant": openfga.get("readAfterGrant"),
        "writeAfterGrant": openfga.get("writeAfterGrant"),
        "readAfterRevoke": openfga.get("readAfterRevoke"),
        "writeAfterRevoke": openfga.get("writeAfterRevoke"),
    },
}
print(json.dumps(bundle, ensure_ascii=True, indent=2))
PY
)"

if [[ "${evidence_file}" != "-" ]]; then
  mkdir -p "$(dirname "${evidence_file}")"
  tmp_file="$(mktemp)"
  trap 'rm -f "${tmp_file}"' EXIT
  printf '%s\n' "${bundle}" >"${tmp_file}"
  install -m 600 "${tmp_file}" "${evidence_file}"
  log "wrote Open Platform production evidence to ${evidence_file}" >&2
fi

printf '%s\n' "${bundle}" | jq .
