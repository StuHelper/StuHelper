#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
SMOKE_SCRIPT="${REPO_ROOT}/infra/ops/openfga-resource-access-smoke.sh"
SMOKE_CMD="${REPO_ROOT}/server/cmd/openfga-resource-smoke/main.go"

fail() {
  echo "[openfga-resource-access-smoke-contract][error] $*" >&2
  exit 1
}

assert_contains() {
  local file="$1"
  local pattern="$2"
  if ! grep -Eq -- "${pattern}" "${file}"; then
    fail "expected ${file} to contain pattern: ${pattern}"
  fi
}

[[ -f "${SMOKE_SCRIPT}" ]] || fail "missing smoke script: ${SMOKE_SCRIPT}"
[[ -f "${SMOKE_CMD}" ]] || fail "missing smoke command: ${SMOKE_CMD}"

bash -n "${SMOKE_SCRIPT}"

assert_contains "${SMOKE_SCRIPT}" 'source "\$\{SCRIPT_DIR\}/lib/common\.sh"'
assert_contains "${SMOKE_SCRIPT}" '^load_env$'
assert_contains "${SMOKE_SCRIPT}" 'OPENFGA_API_URL is required'
assert_contains "${SMOKE_SCRIPT}" 'reject_placeholder_if_set OPENFGA_STORE_ID "\$\{OPENFGA_STORE_ID:-\}" "REPLACE_WITH_OPENFGA_STORE_ID"'
assert_contains "${SMOKE_SCRIPT}" 'reject_placeholder_if_set OPENFGA_MODEL_ID "\$\{OPENFGA_MODEL_ID:-\}" "REPLACE_WITH_OPENFGA_MODEL_ID"'
assert_contains "${SMOKE_SCRIPT}" 'run_smoke_with_go'
assert_contains "${SMOKE_SCRIPT}" 'run_smoke_with_docker'
assert_contains "${SMOKE_SCRIPT}" 'OPENFGA_RESOURCE_SMOKE_MODE'
assert_contains "${SMOKE_SCRIPT}" 'OPENFGA_RESOURCE_SMOKE_GO_IMAGE'
assert_contains "${SMOKE_SCRIPT}" 'OPENFGA_RESOURCE_SMOKE_EVIDENCE_FILE'
assert_contains "${SMOKE_SCRIPT}" 'GOLANG_IMAGE_REF'
assert_contains "${SMOKE_SCRIPT}" 'APP_ENV:-.*production'
assert_contains "${SMOKE_SCRIPT}" 'OPENFGA_RESOURCE_SMOKE_MODE=host'
assert_contains "${SMOKE_SCRIPT}" 'OPENFGA_RESOURCE_SMOKE_MODE must be host or container'
assert_contains "${SMOKE_SCRIPT}" 'golang:1\.26-bookworm'
assert_contains "${SMOKE_SCRIPT}" '--network "\$\{docker_network_name\}"'
assert_contains "${SMOKE_SCRIPT}" 'go run \./cmd/openfga-resource-smoke'
assert_contains "${SMOKE_SCRIPT}" 'mkdir -p "\$\(dirname "\$\{OPENFGA_RESOURCE_SMOKE_EVIDENCE_FILE\}"\)"'
assert_contains "${SMOKE_SCRIPT}" '\| jq \.'

assert_contains "${SMOKE_CMD}" 'relationReadByApp[[:space:]]+= "can_read_by_app"'
assert_contains "${SMOKE_CMD}" 'relationWriteByApp[[:space:]]+= "can_write_by_app"'
assert_contains "${SMOKE_CMD}" 'open_platform_app:'
assert_contains "${SMOKE_CMD}" 'resource_item:'
assert_contains "${SMOKE_CMD}" 'WriteMissingTuples'
assert_contains "${SMOKE_CMD}" 'ListObjects'
assert_contains "${SMOKE_CMD}" 'DeleteTuples'
assert_contains "${SMOKE_CMD}" 'ListedReadAfterRevoke'
assert_contains "${SMOKE_CMD}" 'ReadAfterRevoke'
assert_contains "${SMOKE_CMD}" 'WriteAfterRevoke'

echo "[openfga-resource-access-smoke-contract] all assertions passed"
