#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
# shellcheck source=../lib/common.sh
source "${REPO_ROOT}/infra/ops/lib/common.sh"

fail() {
  printf '[registry-auth-contract][error] %s\n' "$*" >&2
  exit 1
}

tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT
mkdir -p "${tmpdir}/bin"

DOCKER_CALL_FILE="${tmpdir}/docker-call"
DOCKER_STDIN_FILE="${tmpdir}/docker-stdin"
export DOCKER_CALL_FILE DOCKER_STDIN_FILE

cat >"${tmpdir}/bin/docker" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >"${DOCKER_CALL_FILE}"
IFS= read -r password
printf '%s' "${password}" >"${DOCKER_STDIN_FILE}"
EOF
chmod +x "${tmpdir}/bin/docker"

if (
  export PATH="${tmpdir}/bin:${PATH}"
  export REGISTRY=ghcr.io
  export REGISTRY_AUTH_MODE=workflow-token
  export CI_REGISTRY_LOGIN_READY=false
  docker_registry_login
) >/dev/null 2>&1; then
  fail "workflow-token mode succeeded without a CI-established login"
fi

PATH="${tmpdir}/bin:${PATH}" \
  REGISTRY=ghcr.io \
  REGISTRY_AUTH_MODE=workflow-token \
  CI_REGISTRY_LOGIN_READY=true \
  docker_registry_login
[[ ! -e "${DOCKER_CALL_FILE}" ]] ||
  fail "workflow-token mode unexpectedly performed a second Docker login"

PATH="${tmpdir}/bin:${PATH}" \
  REGISTRY=registry.example.com \
  REGISTRY_AUTH_MODE=persistent-secret \
  REGISTRY_USERNAME=deploy-user \
  REGISTRY_PASSWORD=registry-password-sentinel \
  docker_registry_login
[[ "$(<"${DOCKER_CALL_FILE}")" == "login registry.example.com --username deploy-user --password-stdin" ]] ||
  fail "persistent-secret mode did not use password-stdin"
[[ "$(<"${DOCKER_STDIN_FILE}")" == "registry-password-sentinel" ]] ||
  fail "persistent-secret mode did not deliver the credential on standard input"

if (
  export PATH="${tmpdir}/bin:${PATH}"
  export REGISTRY=ghcr.io
  export REGISTRY_AUTH_MODE=unknown
  docker_registry_login
) >/dev/null 2>&1; then
  fail "an unknown registry authentication mode was accepted"
fi

printf '[registry-auth-contract] all assertions passed\n'
