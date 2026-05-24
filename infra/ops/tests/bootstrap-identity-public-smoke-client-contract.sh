#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
BOOTSTRAP_SCRIPT="${REPO_ROOT}/infra/ops/bootstrap-identity-public-smoke-client.sh"
BOOTSTRAP_CMD="${REPO_ROOT}/server/cmd/identity-public-smoke-client-bootstrap/main.go"

fail() {
  echo "[bootstrap-identity-public-smoke-client-contract][error] $*" >&2
  exit 1
}

assert_contains() {
  local file="$1"
  local pattern="$2"
  if ! grep -Eq -- "${pattern}" "${file}"; then
    fail "expected ${file} to contain pattern: ${pattern}"
  fi
}

assert_file_contains() {
  local file="$1"
  local pattern="$2"
  if ! grep -Eq -- "${pattern}" "${file}"; then
    fail "expected ${file} to contain pattern: ${pattern}"
  fi
}

[[ -f "${BOOTSTRAP_SCRIPT}" ]] || fail "missing bootstrap script: ${BOOTSTRAP_SCRIPT}"
[[ -x "${BOOTSTRAP_SCRIPT}" ]] || fail "bootstrap script must be executable: ${BOOTSTRAP_SCRIPT}"
[[ -f "${BOOTSTRAP_CMD}" ]] || fail "missing bootstrap command: ${BOOTSTRAP_CMD}"

bash -n "${BOOTSTRAP_SCRIPT}"

assert_contains "${BOOTSTRAP_SCRIPT}" 'identity-public-smoke-client-bootstrap'
assert_contains "${BOOTSTRAP_SCRIPT}" 'IDENTITY_PUBLIC_SMOKE_BOOTSTRAP_MODE'
assert_contains "${BOOTSTRAP_SCRIPT}" 'IDENTITY_PUBLIC_SMOKE_BOOTSTRAP_GO_IMAGE'
assert_contains "${BOOTSTRAP_SCRIPT}" 'GOLANG_IMAGE_REF'
assert_contains "${BOOTSTRAP_SCRIPT}" 'IDENTITY_PUBLIC_SMOKE_OWNER_USER_ID is required'
assert_contains "${BOOTSTRAP_SCRIPT}" 'IDENTITY_PUBLIC_SMOKE_REVIEWER_USER_ID is required'
assert_contains "${BOOTSTRAP_SCRIPT}" 'go run ./cmd/identity-public-smoke-client-bootstrap'
assert_contains "${BOOTSTRAP_SCRIPT}" 'upsert_env_file "\$\{SECRETS_ENV_FILE\}" "\$\{key\}" "\$\{value\}"'
assert_contains "${BOOTSTRAP_CMD}" 'BootstrapIdentityPublicSmokeClient'
assert_contains "${BOOTSTRAP_CMD}" 'IDENTITY_PUBLIC_SMOKE_CLIENT_SECRET'

tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT
mkdir -p "${tmpdir}/bin" "${tmpdir}/generated/observability"

cat >"${tmpdir}/bin/go" <<'SH'
#!/usr/bin/env bash
printf '%s\n' \
  'IDENTITY_PUBLIC_SMOKE_CLIENT_ID=identity-public-smoke-contract' \
  'IDENTITY_PUBLIC_SMOKE_REDIRECT_URI=https://stuhelper.test/open-platform/identity-public-smoke/callback' \
  'IDENTITY_PUBLIC_SMOKE_CLIENT_CREDENTIALS_SCOPE=resource.read' \
  'IDENTITY_PUBLIC_SMOKE_CLIENT_SECRET=ids_contract_secret_should_not_log' \
  'IDENTITY_PUBLIC_SMOKE_RESOURCE_ACCESS_EXPECT_ALLOWED=false'
SH
chmod +x "${tmpdir}/bin/go"

env_file="${tmpdir}/.env.prod.shared"
secrets_file="${tmpdir}/.env.prod.secrets.local"
generated_env_file="${tmpdir}/.env.prod.generated"
generated_secret_file="${tmpdir}/.env.prod.generated.secrets"

cat >"${env_file}" <<'EOF'
DATABASE_URL=postgres://stuhelper_app:pw@postgres:5432/stuhelper?sslmode=disable
WEB_PUBLIC_URL=https://stuhelper.test
IDENTITY_PUBLIC_SMOKE_BOOTSTRAP_MODE=host
IDENTITY_PUBLIC_SMOKE_OWNER_USER_ID=1
IDENTITY_PUBLIC_SMOKE_REVIEWER_USER_ID=2
EOF
touch "${secrets_file}" "${generated_env_file}" "${generated_secret_file}"

output="$(
  PATH="${tmpdir}/bin:${PATH}" \
  ENV_FILE="${env_file}" \
  SECRETS_ENV_FILE="${secrets_file}" \
  GENERATED_ENV_FILE="${generated_env_file}" \
  GENERATED_SECRET_ENV_FILE="${generated_secret_file}" \
  GENERATED_OBS_DIR="${tmpdir}/generated/observability" \
  "${BOOTSTRAP_SCRIPT}"
)"

printf '%s\n' "${output}" | grep -q 'ensured approved Identity public smoke client identity-public-smoke-contract' || \
  fail "bootstrap script did not log sanitized success"
if printf '%s\n' "${output}" | grep -q 'ids_contract_secret_should_not_log'; then
  fail "bootstrap script must not print the generated client secret"
fi

assert_file_contains "${env_file}" '^IDENTITY_PUBLIC_SMOKE_CLIENT_ID=identity-public-smoke-contract$'
assert_file_contains "${env_file}" '^IDENTITY_PUBLIC_SMOKE_REDIRECT_URI=https://stuhelper.test/open-platform/identity-public-smoke/callback$'
assert_file_contains "${env_file}" '^IDENTITY_PUBLIC_SMOKE_CLIENT_CREDENTIALS_SCOPE=resource.read$'
assert_file_contains "${env_file}" '^IDENTITY_PUBLIC_SMOKE_RESOURCE_ACCESS_EXPECT_ALLOWED=false$'
assert_file_contains "${secrets_file}" '^IDENTITY_PUBLIC_SMOKE_CLIENT_SECRET=ids_contract_secret_should_not_log$'

echo "[bootstrap-identity-public-smoke-client-contract] all assertions passed"
