#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
SMOKE_SCRIPT="${REPO_ROOT}/infra/ops/casdoor-runtime-token-probe-smoke.sh"
BOOTSTRAP_SCRIPT="${REPO_ROOT}/infra/ops/bootstrap-platform.sh"

fail() {
  echo "[casdoor-runtime-token-probe-smoke-contract][error] $*" >&2
  exit 1
}

cleanup_dirs=()
cleanup() {
  local dir
  for dir in "${cleanup_dirs[@]:-}"; do
    rm -rf "${dir}"
  done
}
trap cleanup EXIT

assert_contains() {
  local file="$1"
  local pattern="$2"
  if ! grep -Eq -- "${pattern}" "${file}"; then
    fail "expected ${file} to contain pattern: ${pattern}"
  fi
}

[[ -f "${SMOKE_SCRIPT}" ]] || fail "missing smoke script: ${SMOKE_SCRIPT}"
[[ -x "${SMOKE_SCRIPT}" ]] || fail "smoke script must be executable: ${SMOKE_SCRIPT}"
[[ -f "${BOOTSTRAP_SCRIPT}" ]] || fail "missing bootstrap script: ${BOOTSTRAP_SCRIPT}"

bash -n "${SMOKE_SCRIPT}"

assert_contains "${SMOKE_SCRIPT}" 'source "\$\{SCRIPT_DIR\}/lib/common\.sh"'
assert_contains "${SMOKE_SCRIPT}" '^load_env$'
assert_contains "${SMOKE_SCRIPT}" 'CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_ID'
assert_contains "${SMOKE_SCRIPT}" 'CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_SECRET'
assert_contains "${SMOKE_SCRIPT}" 'CASDOOR_TOKEN_PROBE_SMOKE_APPLICATION'
assert_contains "${SMOKE_SCRIPT}" 'CASDOOR_TOKEN_PROBE_SMOKE_REDIRECT_URI'
assert_contains "${SMOKE_SCRIPT}" 'OPEN_PLATFORM_TOKEN_PROBE_SMOKE_MODE'
assert_contains "${SMOKE_SCRIPT}" 'OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_REQUIRED must be true'
assert_contains "${SMOKE_SCRIPT}" 'CASDOOR_TOKEN_PROBE_CLIENT_ID='
assert_contains "${SMOKE_SCRIPT}" 'CASDOOR_TOKEN_PROBE_CLIENT_SECRET='
assert_contains "${SMOKE_SCRIPT}" 'CASDOOR_TOKEN_PROBE_REDIRECT_URI='
assert_contains "${SMOKE_SCRIPT}" 'compose --profile prod run --rm --no-deps -T'
assert_contains "${SMOKE_SCRIPT}" '--entrypoint "\$\{entrypoint\}"'
assert_contains "${SMOKE_SCRIPT}" 'businessClaims'
assert_contains "${SMOKE_SCRIPT}" 'length == 0'
assert_contains "${SMOKE_SCRIPT}" 'inspectedClaims'
assert_contains "${SMOKE_SCRIPT}" 'casdoor-runtime-token-probe-runner\.mjs'
assert_contains "${SMOKE_SCRIPT}" 'nonceVerified'
assert_contains "${SMOKE_SCRIPT}" '\| jq \.'

assert_contains "${BOOTSTRAP_SCRIPT}" 'CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_ID'
assert_contains "${BOOTSTRAP_SCRIPT}" 'CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_SECRET'
assert_contains "${BOOTSTRAP_SCRIPT}" 'CASDOOR_TOKEN_PROBE_SMOKE_APPLICATION'
assert_contains "${BOOTSTRAP_SCRIPT}" 'CASDOOR_TOKEN_PROBE_SMOKE_REDIRECT_URI'

tmpdir="$(mktemp -d)"
cleanup_dirs+=("${tmpdir}")
fake_runner="${tmpdir}/fake-runtime-token-probe-runner"
env_file="${tmpdir}/.env"
generated_env_file="${tmpdir}/.env.generated"
generated_secret_env_file="${tmpdir}/.env.generated.secrets"
generated_obs_dir="${tmpdir}/generated/observability"

cat >"${fake_runner}" <<'RUNNER'
#!/usr/bin/env bash
set -euo pipefail

[[ "${CASDOOR_ISSUER}" == "https://sso.stuhelper.com" ]]
[[ "${CASDOOR_TOKEN_PROBE_USERNAME}" == "probe-user" ]]
[[ "${CASDOOR_TOKEN_PROBE_PASSWORD}" == "probe-password" ]]
[[ "${CASDOOR_TOKEN_PROBE_CLIENT_ID:-}" == "" ]]
[[ "${CASDOOR_TOKEN_PROBE_CLIENT_SECRET:-}" == "" ]]
[[ "${CASDOOR_TOKEN_PROBE_REDIRECT_URI:-}" == "" ]]
[[ "${CASDOOR_TOKEN_PROBE_SCOPE}" == "openid" ]]

payload="$(cat)"
jq -e '
  .issuer == "https://sso.stuhelper.com"
  and .casdoorApplicationName == "casdoor-token-probe-smoke"
  and .clientID == "smoke-client"
  and .clientSecret == "smoke-secret"
  and .redirectURIs == ["https://stuhelper.com/open-platform/token-probe/callback"]
  and .scope == "openid"
' <<<"${payload}" >/dev/null

printf '%s\n' '{"method":"authorization_code","issuer":"https://sso.stuhelper.com","inspectedClaims":["iss","nonce","sub"],"businessClaims":[],"tokenClaims":{"id_token":["iss","nonce","sub"]},"metadata":{"source":"casdoor-runtime-token-probe-runner.mjs","nonceVerified":true}}'
RUNNER
chmod +x "${fake_runner}"

cat >"${env_file}" <<'ENV'
APP_ENV=production
OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_REQUIRED=true
OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND=/app/casdoor-runtime-token-probe-runner.mjs
CASDOOR_ISSUER=https://sso.stuhelper.com
CASDOOR_TOKEN_PROBE_USERNAME=probe-user
CASDOOR_TOKEN_PROBE_PASSWORD=probe-password
CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_ID=smoke-client
CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_SECRET=smoke-secret
CASDOOR_TOKEN_PROBE_SMOKE_APPLICATION=casdoor-token-probe-smoke
CASDOOR_TOKEN_PROBE_SMOKE_REDIRECT_URI=https://stuhelper.com/open-platform/token-probe/callback
CASDOOR_TOKEN_PROBE_CLIENT_ID=ambient-wrong-client
CASDOOR_TOKEN_PROBE_CLIENT_SECRET=ambient-wrong-secret
CASDOOR_TOKEN_PROBE_REDIRECT_URI=https://wrong.example.com/callback
ENV

output="$(
  ENV_FILE="${env_file}" \
  GENERATED_ENV_FILE="${generated_env_file}" \
  GENERATED_SECRET_ENV_FILE="${generated_secret_env_file}" \
  GENERATED_OBS_DIR="${generated_obs_dir}" \
  OPEN_PLATFORM_TOKEN_PROBE_SMOKE_MODE=host \
  OPEN_PLATFORM_TOKEN_PROBE_SMOKE_COMMAND="${fake_runner}" \
  "${SMOKE_SCRIPT}"
)"

if [[ "${output}" == *"probe-password"* || "${output}" == *"smoke-secret"* ]]; then
  fail "smoke output must not contain token probe secrets"
fi
printf '%s\n' "${output}" | jq -e '
  .businessClaims == []
  and .inspectedClaims == ["iss","nonce","sub"]
  and .metadata.nonceVerified == true
' >/dev/null

echo "[casdoor-runtime-token-probe-smoke-contract] all assertions passed"
