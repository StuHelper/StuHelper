#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
EVIDENCE_SCRIPT="${REPO_ROOT}/infra/ops/open-platform-production-evidence.sh"

fail() {
  echo "[open-platform-production-evidence-contract][error] $*" >&2
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

[[ -f "${EVIDENCE_SCRIPT}" ]] || fail "missing evidence script: ${EVIDENCE_SCRIPT}"
[[ -x "${EVIDENCE_SCRIPT}" ]] || fail "evidence script must be executable: ${EVIDENCE_SCRIPT}"

bash -n "${EVIDENCE_SCRIPT}"

assert_contains "${EVIDENCE_SCRIPT}" 'casdoor-runtime-token-probe-smoke\.sh'
assert_contains "${EVIDENCE_SCRIPT}" 'openfga-resource-access-smoke\.sh'
assert_contains "${EVIDENCE_SCRIPT}" 'OPEN_PLATFORM_PRODUCTION_EVIDENCE_FILE'
assert_contains "${EVIDENCE_SCRIPT}" 'OPEN_PLATFORM_PRODUCTION_EVIDENCE_ALLOW_LOCAL_TARGETS'
assert_contains "${EVIDENCE_SCRIPT}" 'CASDOOR_ISSUER is required'
assert_contains "${EVIDENCE_SCRIPT}" 'OPENFGA_API_URL is required'
assert_contains "${EVIDENCE_SCRIPT}" 'OPENFGA_STORE_ID is required'
assert_contains "${EVIDENCE_SCRIPT}" 'OPENFGA_MODEL_ID is required'
assert_contains "${EVIDENCE_SCRIPT}" 'OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_REQUIRED must be true for Open Platform production evidence'
assert_contains "${EVIDENCE_SCRIPT}" 'OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND is required for Open Platform production evidence'
assert_contains "${EVIDENCE_SCRIPT}" 'OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND is still the placeholder value'
assert_contains "${EVIDENCE_SCRIPT}" 'open-platform-production-evidence verifies production Casdoor/OpenFGA evidence'
assert_contains "${EVIDENCE_SCRIPT}" 'businessClaims'
assert_contains "${EVIDENCE_SCRIPT}" 'length == 0'
assert_contains "${EVIDENCE_SCRIPT}" 'readAfterGrant == true'
assert_contains "${EVIDENCE_SCRIPT}" 'listedReadAfterRevoke == false'
assert_contains "${EVIDENCE_SCRIPT}" 'writeAfterRevoke == false'
assert_contains "${EVIDENCE_SCRIPT}" '"passed": True'
assert_contains "${EVIDENCE_SCRIPT}" '"summary"'
assert_contains "${EVIDENCE_SCRIPT}" 'print_casdoor_diagnostic'
assert_contains "${EVIDENCE_SCRIPT}" 'print_openfga_diagnostic'
assert_contains "${EVIDENCE_SCRIPT}" 'tokenClaimTypes'
assert_contains "${EVIDENCE_SCRIPT}" 'nonceVerified'
assert_contains "${EVIDENCE_SCRIPT}" 'infra/generated/open-platform-production-evidence\.json'

tmpdir="$(mktemp -d)"
cleanup_dirs+=("${tmpdir}")
fake_casdoor="${tmpdir}/fake-casdoor-smoke"
fake_casdoor_leaky="${tmpdir}/fake-casdoor-leaky-smoke"
fake_casdoor_local="${tmpdir}/fake-casdoor-local-smoke"
fake_openfga="${tmpdir}/fake-openfga-smoke"
fake_openfga_broken="${tmpdir}/fake-openfga-broken-smoke"
fake_openfga_local="${tmpdir}/fake-openfga-local-smoke"
env_file="${tmpdir}/.env"
local_env_file="${tmpdir}/.env.local-targets"
runtime_probe_disabled_env_file="${tmpdir}/.env.runtime-probe-disabled"
runtime_probe_placeholder_env_file="${tmpdir}/.env.runtime-probe-placeholder"
generated_env_file="${tmpdir}/.env.generated"
generated_secret_env_file="${tmpdir}/.env.generated.secrets"
generated_obs_dir="${tmpdir}/generated/observability"
evidence_file="${tmpdir}/evidence/open-platform-production-evidence.json"
leaky_evidence_file="${tmpdir}/evidence/open-platform-production-evidence-leaky.json"
broken_openfga_evidence_file="${tmpdir}/evidence/open-platform-production-evidence-broken-openfga.json"
local_refused_evidence_file="${tmpdir}/evidence/open-platform-production-evidence-local-refused.json"
local_allowed_evidence_file="${tmpdir}/evidence/open-platform-production-evidence-local-allowed.json"

cat >"${fake_casdoor}" <<'CASDOOR'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' '{
  "method": "authorization_code",
  "issuer": "https://sso.stuhelper.com",
  "inspectedClaims": ["iss", "sub"],
  "businessClaims": [],
  "tokenClaims": {"id_token": ["iss", "sub"]},
  "clientSecret": "should-not-leak",
	  "metadata": {
	    "source": "casdoor-runtime-token-probe-runner.mjs",
	    "capture": "playwright",
	    "nonceVerified": true,
	    "rawToken": "should-not-leak"
	  }
	}'
CASDOOR
chmod +x "${fake_casdoor}"

cat >"${fake_casdoor_leaky}" <<'CASDOOR'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' '{
  "method": "authorization_code",
  "issuer": "https://sso.stuhelper.com",
  "inspectedClaims": ["iss", "phone", "sub"],
  "businessClaims": ["phone"],
  "tokenClaims": {"id_token": ["iss", "phone", "sub"]},
  "clientSecret": "should-not-leak",
  "accessToken": "should-not-leak",
	  "metadata": {
	    "source": "casdoor-runtime-token-probe-runner.mjs",
	    "capture": "playwright",
	    "nonceVerified": true,
	    "rawToken": "should-not-leak"
	  }
	}'
CASDOOR
chmod +x "${fake_casdoor_leaky}"

cat >"${fake_casdoor_local}" <<'CASDOOR'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' '{
  "method": "authorization_code",
  "issuer": "http://localhost:8085",
  "inspectedClaims": ["iss", "sub"],
  "businessClaims": [],
  "tokenClaims": {"id_token": ["iss", "sub"]},
	  "metadata": {
	    "source": "casdoor-runtime-token-probe-runner.mjs",
	    "capture": "playwright",
	    "nonceVerified": true
	  }
	}'
CASDOOR
chmod +x "${fake_casdoor_local}"

cat >"${fake_openfga}" <<'OPENFGA'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' '{
  "apiURL": "http://openfga:8080",
  "appObject": "open_platform_app:smoke",
  "listedReadAfterRevoke": false,
  "listedReadGrant": true,
  "modelID": "model-1",
  "readAfterGrant": true,
  "readAfterRevoke": false,
  "resourceObject": "resource_item:resource",
  "storeID": "store-1",
  "writeAfterGrant": true,
  "writeAfterRevoke": false,
  "clientSecret": "should-not-leak"
}'
OPENFGA
chmod +x "${fake_openfga}"

cat >"${fake_openfga_broken}" <<'OPENFGA'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' '{
  "apiURL": "http://openfga:8080",
  "appObject": "open_platform_app:smoke",
  "listedReadAfterRevoke": true,
  "listedReadGrant": true,
  "modelID": "model-1",
  "readAfterGrant": true,
  "readAfterRevoke": false,
  "resourceObject": "resource_item:resource",
  "storeID": "store-1",
  "writeAfterGrant": true,
  "writeAfterRevoke": false,
  "clientSecret": "should-not-leak"
}'
OPENFGA
chmod +x "${fake_openfga_broken}"

cat >"${fake_openfga_local}" <<'OPENFGA'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' '{
  "apiURL": "http://127.0.0.1:8081",
  "appObject": "open_platform_app:smoke",
  "listedReadAfterRevoke": false,
  "listedReadGrant": true,
  "modelID": "model-1",
  "readAfterGrant": true,
  "readAfterRevoke": false,
  "resourceObject": "resource_item:resource",
  "storeID": "store-1",
  "writeAfterGrant": true,
  "writeAfterRevoke": false
}'
OPENFGA
chmod +x "${fake_openfga_local}"

cat >"${env_file}" <<'ENV'
APP_ENV=production
CASDOOR_ISSUER=https://sso.stuhelper.com
OPENFGA_API_URL=http://openfga:8080
OPENFGA_STORE_ID=store-1
OPENFGA_MODEL_ID=model-1
OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_REQUIRED=true
OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND=/app/casdoor-runtime-token-probe-runner.mjs
ENV

cat >"${local_env_file}" <<'ENV'
APP_ENV=production
CASDOOR_ISSUER=http://localhost:8085
CASDOOR_TOKEN_PROBE_SMOKE_REDIRECT_URI=http://localhost:3000/open-platform/token-probe/callback
OPENFGA_API_URL=http://127.0.0.1:8081
OPENFGA_STORE_ID=store-1
OPENFGA_MODEL_ID=model-1
OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_REQUIRED=true
OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND=/app/casdoor-runtime-token-probe-runner.mjs
OPEN_PLATFORM_PRODUCTION_EVIDENCE_ALLOW_LOCAL_TARGETS=false
ENV

cat >"${runtime_probe_disabled_env_file}" <<'ENV'
APP_ENV=production
CASDOOR_ISSUER=https://sso.stuhelper.com
OPENFGA_API_URL=http://openfga:8080
OPENFGA_STORE_ID=store-1
OPENFGA_MODEL_ID=model-1
OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_REQUIRED=false
OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND=/app/casdoor-runtime-token-probe-runner.mjs
ENV

cat >"${runtime_probe_placeholder_env_file}" <<'ENV'
APP_ENV=production
CASDOOR_ISSUER=https://sso.stuhelper.com
OPENFGA_API_URL=http://openfga:8080
OPENFGA_STORE_ID=store-1
OPENFGA_MODEL_ID=model-1
OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_REQUIRED=true
OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND=REPLACE_WITH_OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND
ENV

output="$(
  ENV_FILE="${env_file}" \
  GENERATED_ENV_FILE="${generated_env_file}" \
  GENERATED_SECRET_ENV_FILE="${generated_secret_env_file}" \
  GENERATED_OBS_DIR="${generated_obs_dir}" \
  OPEN_PLATFORM_EVIDENCE_CASDOOR_SMOKE_COMMAND="${fake_casdoor}" \
  OPEN_PLATFORM_EVIDENCE_OPENFGA_SMOKE_COMMAND="${fake_openfga}" \
  OPEN_PLATFORM_PRODUCTION_EVIDENCE_FILE="${evidence_file}" \
  "${EVIDENCE_SCRIPT}"
)"

[[ -f "${evidence_file}" ]] || fail "evidence bundle file was not written"
if [[ "${output}" == *"should-not-leak"* ]] || grep -q 'should-not-leak' "${evidence_file}"; then
  fail "production evidence bundle must not include unapproved child smoke fields"
fi

OUTPUT_JSON="${output}" python3 - <<'PY'
import json
import os

data = json.loads(os.environ["OUTPUT_JSON"])
assert data["appEnv"] == "production"
assert data["passed"] is True
assert data["summary"]["casdoorRuntimeTokenProbe"] is True
assert data["summary"]["openfgaResourceAccessSmoke"] is True
assert data["summary"]["runtimeTokenProbeRequired"] is True
assert data["configuration"]["runtimeTokenProbeRequired"] is True
assert data["configuration"]["runtimeTokenProbeCommandConfigured"] is True
assert data["casdoorRuntimeTokenProbe"]["passed"] is True
assert data["casdoorRuntimeTokenProbe"]["businessClaims"] == []
assert data["casdoorRuntimeTokenProbe"]["tokenClaimTypes"] == ["id_token"]
assert data["casdoorRuntimeTokenProbe"]["metadata"]["source"] == "casdoor-runtime-token-probe-runner.mjs"
assert data["casdoorRuntimeTokenProbe"]["metadata"]["nonceVerified"] is True
assert data["openfgaResourceAccessSmoke"]["passed"] is True
assert data["openfgaResourceAccessSmoke"]["listedReadAfterRevoke"] is False
assert data["openfgaResourceAccessSmoke"]["readAfterGrant"] is True
assert data["openfgaResourceAccessSmoke"]["writeAfterGrant"] is True
assert data["openfgaResourceAccessSmoke"]["readAfterRevoke"] is False
assert data["openfgaResourceAccessSmoke"]["writeAfterRevoke"] is False
PY

jq -e '.generatedAt and .casdoorRuntimeTokenProbe and .openfgaResourceAccessSmoke' "${evidence_file}" >/dev/null

runtime_probe_disabled_output="$(
  ENV_FILE="${runtime_probe_disabled_env_file}" \
  GENERATED_ENV_FILE="${generated_env_file}" \
  GENERATED_SECRET_ENV_FILE="${generated_secret_env_file}" \
  GENERATED_OBS_DIR="${generated_obs_dir}" \
  OPEN_PLATFORM_EVIDENCE_CASDOOR_SMOKE_COMMAND="${fake_casdoor}" \
  OPEN_PLATFORM_EVIDENCE_OPENFGA_SMOKE_COMMAND="${fake_openfga}" \
  OPEN_PLATFORM_PRODUCTION_EVIDENCE_FILE="${tmpdir}/evidence/open-platform-production-evidence-runtime-probe-disabled.json" \
  "${EVIDENCE_SCRIPT}" 2>&1
)" && fail "production evidence script passed despite disabled runtime token probe gate"

printf '%s\n' "${runtime_probe_disabled_output}" | grep -q 'OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_REQUIRED must be true for Open Platform production evidence' || \
  fail "disabled runtime token probe evidence did not fail with the expected gate"
if [[ "${runtime_probe_disabled_output}" == *"running Casdoor runtime token minimization smoke"* ]]; then
  fail "runtime token probe gate should fail before child smokes run"
fi

runtime_probe_placeholder_evidence_file="${tmpdir}/evidence/open-platform-production-evidence-runtime-probe-placeholder.json"
runtime_probe_placeholder_output="$(
  ENV_FILE="${runtime_probe_placeholder_env_file}" \
  GENERATED_ENV_FILE="${generated_env_file}" \
  GENERATED_SECRET_ENV_FILE="${generated_secret_env_file}" \
  GENERATED_OBS_DIR="${generated_obs_dir}" \
  OPEN_PLATFORM_EVIDENCE_CASDOOR_SMOKE_COMMAND="${fake_casdoor}" \
  OPEN_PLATFORM_EVIDENCE_OPENFGA_SMOKE_COMMAND="${fake_openfga}" \
  OPEN_PLATFORM_PRODUCTION_EVIDENCE_FILE="${runtime_probe_placeholder_evidence_file}" \
  "${EVIDENCE_SCRIPT}" 2>&1
)" && fail "production evidence script passed despite placeholder runtime token probe command"

[[ ! -f "${runtime_probe_placeholder_evidence_file}" ]] || fail "placeholder runtime token probe evidence must not be written"
printf '%s\n' "${runtime_probe_placeholder_output}" | grep -q 'OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND is still the placeholder value' || \
  fail "placeholder runtime token probe evidence did not fail with the expected gate"
if [[ "${runtime_probe_placeholder_output}" == *"running Casdoor runtime token minimization smoke"* ]]; then
  fail "placeholder runtime token probe gate should fail before child smokes run"
fi

local_refused_output="$(
  ENV_FILE="${local_env_file}" \
  GENERATED_ENV_FILE="${generated_env_file}" \
  GENERATED_SECRET_ENV_FILE="${generated_secret_env_file}" \
  GENERATED_OBS_DIR="${generated_obs_dir}" \
  OPEN_PLATFORM_EVIDENCE_CASDOOR_SMOKE_COMMAND="${fake_casdoor_local}" \
  OPEN_PLATFORM_EVIDENCE_OPENFGA_SMOKE_COMMAND="${fake_openfga_local}" \
  OPEN_PLATFORM_PRODUCTION_EVIDENCE_FILE="${local_refused_evidence_file}" \
  "${EVIDENCE_SCRIPT}" 2>&1
)" && fail "production evidence script passed despite local Casdoor/OpenFGA targets"

[[ ! -f "${local_refused_evidence_file}" ]] || fail "local-target production evidence must not be written by default"
printf '%s\n' "${local_refused_output}" | grep -q 'CASDOOR_ISSUER points to a local target' || \
  fail "local-target evidence did not fail with the expected CASDOOR_ISSUER guard"
if [[ "${local_refused_output}" == *"running Casdoor runtime token minimization smoke"* ]]; then
  fail "local-target guard should fail before child smokes run"
fi

local_allowed_output="$(
  ENV_FILE="${local_env_file}" \
  GENERATED_ENV_FILE="${generated_env_file}" \
  GENERATED_SECRET_ENV_FILE="${generated_secret_env_file}" \
  GENERATED_OBS_DIR="${generated_obs_dir}" \
  OPEN_PLATFORM_EVIDENCE_CASDOOR_SMOKE_COMMAND="${fake_casdoor_local}" \
  OPEN_PLATFORM_EVIDENCE_OPENFGA_SMOKE_COMMAND="${fake_openfga_local}" \
  OPEN_PLATFORM_PRODUCTION_EVIDENCE_FILE="${local_allowed_evidence_file}" \
  OPEN_PLATFORM_PRODUCTION_EVIDENCE_ALLOW_LOCAL_TARGETS=true \
  "${EVIDENCE_SCRIPT}"
)"

[[ -f "${local_allowed_evidence_file}" ]] || fail "explicitly allowed local-target evidence bundle was not written"
OUTPUT_JSON="${local_allowed_output}" python3 - <<'PY'
import json
import os

data = json.loads(os.environ["OUTPUT_JSON"])
assert data["passed"] is True
assert data["casdoorRuntimeTokenProbe"]["issuer"] == "http://localhost:8085"
assert data["openfgaResourceAccessSmoke"]["apiURL"] == "http://127.0.0.1:8081"
PY

leaky_output="$(
  ENV_FILE="${env_file}" \
  GENERATED_ENV_FILE="${generated_env_file}" \
  GENERATED_SECRET_ENV_FILE="${generated_secret_env_file}" \
  GENERATED_OBS_DIR="${generated_obs_dir}" \
  OPEN_PLATFORM_EVIDENCE_CASDOOR_SMOKE_COMMAND="${fake_casdoor_leaky}" \
  OPEN_PLATFORM_EVIDENCE_OPENFGA_SMOKE_COMMAND="${fake_openfga}" \
  OPEN_PLATFORM_PRODUCTION_EVIDENCE_FILE="${leaky_evidence_file}" \
  "${EVIDENCE_SCRIPT}" 2>&1
)" && fail "production evidence script passed despite business token claims"

[[ ! -f "${leaky_evidence_file}" ]] || fail "leaky production evidence must not be written"
printf '%s\n' "${leaky_output}" | grep -q 'Casdoor runtime token minimization evidence did not pass production checks' || \
  fail "leaky token evidence did not fail with the expected message"
printf '%s\n' "${leaky_output}" | grep -q '"phone"' || fail "leaky token evidence did not print diagnostic claims"
if [[ "${leaky_output}" == *"should-not-leak"* ]]; then
  fail "leaky token diagnostic must not print raw secret fields"
fi

broken_openfga_output="$(
  ENV_FILE="${env_file}" \
  GENERATED_ENV_FILE="${generated_env_file}" \
  GENERATED_SECRET_ENV_FILE="${generated_secret_env_file}" \
  GENERATED_OBS_DIR="${generated_obs_dir}" \
  OPEN_PLATFORM_EVIDENCE_CASDOOR_SMOKE_COMMAND="${fake_casdoor}" \
  OPEN_PLATFORM_EVIDENCE_OPENFGA_SMOKE_COMMAND="${fake_openfga_broken}" \
  OPEN_PLATFORM_PRODUCTION_EVIDENCE_FILE="${broken_openfga_evidence_file}" \
  "${EVIDENCE_SCRIPT}" 2>&1
)" && fail "production evidence script passed despite broken OpenFGA list-after-revoke check"

[[ ! -f "${broken_openfga_evidence_file}" ]] || fail "broken OpenFGA production evidence must not be written"
printf '%s\n' "${broken_openfga_output}" | grep -q 'OpenFGA resource access evidence did not pass production checks' || \
  fail "broken OpenFGA evidence did not fail with the expected message"
printf '%s\n' "${broken_openfga_output}" | grep -q '"listedReadAfterRevoke": true' || \
  fail "broken OpenFGA evidence did not print list-after-revoke diagnostic"
if [[ "${broken_openfga_output}" == *"should-not-leak"* ]]; then
  fail "broken OpenFGA diagnostic must not print raw secret fields"
fi

echo "[open-platform-production-evidence-contract] all assertions passed"
