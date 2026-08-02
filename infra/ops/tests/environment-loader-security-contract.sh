#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
# shellcheck source=../lib/common.sh
source "${REPO_ROOT}/infra/ops/lib/common.sh"

fail() {
  printf '[environment-loader-security-contract][error] %s\n' "$*" >&2
  exit 1
}

tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT

for key in BASH_ENV ENV; do
  startup_env="${tmpdir}/${key}.env"
  printf '%s=/dev/null\n' "${key}" >"${startup_env}"
  if (source_env_file "${startup_env}") >"${tmpdir}/${key}.log" 2>&1; then
    fail "source_env_file accepted forbidden shell startup variable ${key}"
  fi
  grep -q "shell startup variable ${key} is not allowed" "${tmpdir}/${key}.log" ||
    fail "${key} rejection did not explain the protected boundary"
done

printf 'STACK_NAME=source-env-contract\n' >"${tmpdir}/safe.env"
export BASH_ENV=/dev/null
export ENV=/dev/null
source_env_file "${tmpdir}/safe.env"
[[ "${STACK_NAME}" == "source-env-contract" ]] ||
  fail "source_env_file did not export a validated assignment"
[[ ! -v BASH_ENV && ! -v ENV ]] ||
  fail "source_env_file allowed inherited shell startup hooks to survive"

: >"${tmpdir}/shared.env"
: >"${tmpdir}/generated.env"
: >"${tmpdir}/generated-secrets.env"
export ENV_FILE="${tmpdir}/shared.env"
export GENERATED_ENV_FILE="${tmpdir}/generated.env"
export GENERATED_SECRET_ENV_FILE="${tmpdir}/generated-secrets.env"
export GENERATED_OBS_DIR="${tmpdir}/observability"
export BASH_ENV=/dev/null
export ENV=/dev/null
load_env_preserving BASH_ENV ENV
[[ ! -v BASH_ENV && ! -v ENV ]] ||
  fail "load_env_preserving allowed shell startup hooks to survive"

printf 'STACK_NAME=environment-loader-contract\n' >"${tmpdir}/remote.env"
export BASH_ENV=/dev/null
export ENV=/dev/null
load_remote_deploy_config "${tmpdir}/remote.env"
[[ ! -v BASH_ENV && ! -v ENV ]] ||
  fail "load_remote_deploy_config allowed shell startup hooks to survive"

printf '[environment-loader-security-contract] all assertions passed\n'
