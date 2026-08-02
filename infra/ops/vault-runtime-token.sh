#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

operation="${1:-}"
case "${operation}" in
  check | renew | configure) ;;
  *) die "usage: $0 check|renew|configure" ;;
esac

load_remote_deploy_config
require_cmd jq
require_cmd python3

[[ "${SECRET_BACKEND:-}" == "vault-kv-v2" ]] ||
  die "Vault runtime token management requires SECRET_BACKEND=vault-kv-v2"
[[ -n "${VAULT_ADDR:-}" ]] || die "VAULT_ADDR is required"
[[ -n "${VAULT_TOKEN_FILE:-}" ]] || die "VAULT_TOKEN_FILE is required"
[[ -n "${SHARED_ENV_SECRET_REF:-}" ]] || die "SHARED_ENV_SECRET_REF is required"
[[ -n "${SECRETS_ENV_SECRET_REF:-}" ]] || die "SECRETS_ENV_SECRET_REF is required"
[[ -n "${GENERATED_ENV_SECRET_REF:-}" ]] || die "GENERATED_ENV_SECRET_REF is required"

runtime_policy="${VAULT_RUNTIME_TOKEN_POLICY:-stuhelper-production-deploy}"
period_seconds="${VAULT_RUNTIME_TOKEN_PERIOD_SECONDS:-259200}"
minimum_ttl_seconds="${VAULT_RUNTIME_TOKEN_MIN_TTL_SECONDS:-43200}"
vault_http="${SCRIPT_DIR}/lib/vault-http.py"

[[ "${runtime_policy}" =~ ^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$ ]] ||
  die "VAULT_RUNTIME_TOKEN_POLICY must be a safe Vault policy name"
require_integer_range "VAULT_RUNTIME_TOKEN_PERIOD_SECONDS" "${period_seconds}" 86400 2678400
require_integer_range "VAULT_RUNTIME_TOKEN_MIN_TTL_SECONDS" "${minimum_ttl_seconds}" 3600 "${period_seconds}"
[[ -x "${vault_http}" || -f "${vault_http}" ]] || die "missing Vault HTTP helper: ${vault_http}"

tmpdir="$(mktemp -d)"
cleanup() {
  rm -rf -- "${tmpdir}"
}
trap cleanup EXIT

vault_api() {
  local token_file="$1"
  local method="$2"
  local path="$3"
  local data_file="$4"
  local output_file="$5"
  local args=(
    "${vault_http}"
    --address "${VAULT_ADDR}"
    --token-file "${token_file}"
    --method "${method}"
    --path "${path}"
    --output-file "${output_file}"
  )
  if [[ -n "${VAULT_NAMESPACE:-}" ]]; then
    args+=(--namespace "${VAULT_NAMESPACE}")
  fi
  if [[ -n "${data_file}" ]]; then
    args+=(--data-file "${data_file}")
  fi
  python3 "${args[@]}"
}

secret_data_path() {
  local ref="$1"
  local parts mount path
  mapfile -t parts < <(parse_secret_ref "${ref}")
  mount="${parts[0]:-}"
  path="${parts[1]:-}"
  [[ "${mount}" =~ ^[A-Za-z0-9][A-Za-z0-9._/-]*$ &&
     "${path}" =~ ^[A-Za-z0-9][A-Za-z0-9._/-]*$ ]] ||
    die "Vault secret refs used for production deployment may contain only safe exact path characters"
  [[ "${mount}" != *".."* && "${path}" != *".."* &&
     "${mount}" != *"*"* && "${path}" != *"*"* &&
     "${mount}" != *"+"* && "${path}" != *"+"* ]] ||
    die "Vault runtime policy refuses parent traversal and wildcard secret refs"
  printf '%s/data/%s\n' "${mount%/}" "${path#/}"
}

shared_data_path="$(secret_data_path "${SHARED_ENV_SECRET_REF}")"
secrets_data_path="$(secret_data_path "${SECRETS_ENV_SECRET_REF}")"
generated_data_path="$(secret_data_path "${GENERATED_ENV_SECRET_REF}")"
[[ "${shared_data_path}" != "${secrets_data_path}" &&
   "${shared_data_path}" != "${generated_data_path}" &&
   "${secrets_data_path}" != "${generated_data_path}" ]] ||
  die "shared, secrets, and generated Vault refs must resolve to three distinct exact paths"
denied_probe_path="${VAULT_KV_MOUNT%/}/data/__stuhelper_runtime_token_denied_probe__"

write_capability_payloads() {
  jq -n \
    --arg shared "${shared_data_path}" \
    --arg secrets "${secrets_data_path}" \
    --arg generated "${generated_data_path}" \
    --arg denied "${denied_probe_path}" \
    '{paths: [$shared, $secrets, $generated, "auth/token/lookup-self", "auth/token/renew-self", "sys/capabilities-self", "sys/mounts", $denied]}' \
    >"${tmpdir}/capabilities-request.json"

  jq -n \
    --arg shared "${shared_data_path}" \
    --arg secrets "${secrets_data_path}" \
    --arg generated "${generated_data_path}" \
    --arg denied "${denied_probe_path}" \
    '{
      ($shared): ["read"],
      ($secrets): ["read"],
      ($generated): ["create", "read", "update"],
      "auth/token/lookup-self": ["read"],
      "auth/token/renew-self": ["update"],
      "sys/capabilities-self": ["update"],
      "sys/mounts": ["read"],
      ($denied): ["deny"]
    }' >"${tmpdir}/capabilities-expected.json"
}

validate_runtime_token() {
  local token_file="$1"
  local lookup_file="${tmpdir}/lookup.json"
  local capabilities_file="${tmpdir}/capabilities.json"
  local read_output="${tmpdir}/secret-read.json"

  [[ -s "${token_file}" ]] || return 1
  vault_api "${token_file}" GET auth/token/lookup-self "" "${lookup_file}" || return 1
  python3 - "${lookup_file}" "${runtime_policy}" "${period_seconds}" "${minimum_ttl_seconds}" <<'PY' || return 1
import json
import sys
from pathlib import Path

payload = json.loads(Path(sys.argv[1]).read_text())
policy = sys.argv[2]
period = int(sys.argv[3])
minimum_ttl = int(sys.argv[4])
data = payload.get("data") or {}
token_policies = set(data.get("token_policies") or data.get("policies") or [])
if token_policies != {policy}:
    raise SystemExit("runtime token has unexpected policies")
if data.get("renewable") is not True or data.get("orphan") is not True:
    raise SystemExit("runtime token must be renewable and orphaned")
if int(data.get("period") or 0) != period:
    raise SystemExit("runtime token period does not match the configured period")
if int(data.get("ttl") or 0) < minimum_ttl:
    raise SystemExit("runtime token TTL is below the configured safety floor")
PY

  write_capability_payloads
  vault_api "${token_file}" POST sys/capabilities-self \
    "${tmpdir}/capabilities-request.json" "${capabilities_file}" || return 1
  python3 - "${capabilities_file}" "${tmpdir}/capabilities-expected.json" <<'PY' || return 1
import json
import sys
from pathlib import Path

actual = json.loads(Path(sys.argv[1]).read_text())
expected = json.loads(Path(sys.argv[2]).read_text())
for path, capabilities in expected.items():
    if sorted(actual.get(path) or []) != sorted(capabilities):
        raise SystemExit(f"unexpected Vault capabilities for {path}")
PY

  vault_api "${token_file}" GET "${shared_data_path}" "" "${read_output}" || return 1
  vault_api "${token_file}" GET "${secrets_data_path}" "" "${read_output}" || return 1
  vault_api "${token_file}" GET "${generated_data_path}" "" "${read_output}" || return 1
}

renew_runtime_token() {
  local token_file="$1"
  printf '{}\n' >"${tmpdir}/renew-request.json"
  vault_api "${token_file}" POST auth/token/renew-self \
    "${tmpdir}/renew-request.json" "${tmpdir}/renew-response.json"
  validate_runtime_token "${token_file}"
}

write_runtime_policy() {
  python3 - "${tmpdir}/runtime-policy.hcl" \
    "${shared_data_path}" "${secrets_data_path}" "${generated_data_path}" <<'PY'
import sys
from pathlib import Path

output, shared, secrets, generated = sys.argv[1:5]
rules = (
    (shared, ("read",)),
    (secrets, ("read",)),
    (generated, ("create", "read", "update")),
    ("auth/token/lookup-self", ("read",)),
    ("auth/token/renew-self", ("update",)),
    ("sys/capabilities-self", ("update",)),
    ("sys/mounts", ("read",)),
)
rendered = []
for path, capabilities in rules:
    caps = ", ".join(f'"{capability}"' for capability in capabilities)
    rendered.append(f'path "{path}" {{\n  capabilities = [{caps}]\n}}')
Path(output).write_text("\n\n".join(rendered) + "\n")
PY
  jq -Rs '{policy: .}' <"${tmpdir}/runtime-policy.hcl" >"${tmpdir}/policy-request.json"
}

configure_runtime_token() {
  [[ "${EUID}" -eq 0 ]] || die "configure must run as root"
  require_cmd getent
  require_cmd install
  require_cmd mv
  local owner="${VAULT_RUNTIME_TOKEN_OWNER:-stuhelper}"
  local group="${VAULT_RUNTIME_TOKEN_GROUP:-${owner}}"
  id -u "${owner}" >/dev/null 2>&1 || die "Vault runtime token owner does not exist: ${owner}"
  getent group "${group}" >/dev/null 2>&1 || die "Vault runtime token group does not exist: ${group}"

  if validate_runtime_token "${VAULT_TOKEN_FILE}" >/dev/null 2>&1; then
    chown "${owner}:${group}" "${VAULT_TOKEN_FILE}"
    chmod 0600 "${VAULT_TOKEN_FILE}"
    renew_runtime_token "${VAULT_TOKEN_FILE}"
    VAULT_RUNTIME_TOKEN_OWNER="${owner}" \
    VAULT_RUNTIME_TOKEN_GROUP="${group}" \
    DEPLOY_APP_DIR="${REPO_ROOT}" \
      "${SCRIPT_DIR}/install-vault-token-renewal-timer.sh"
    log "existing least-privilege Vault runtime token is valid and renewed"
    return 0
  fi

  local init_file="${VAULT_ROOT_INIT_FILE:-/var/lib/stuhelper/vault-credentials/init.json}"
  local root_token_file="${tmpdir}/root-token"
  local old_token_file="${tmpdir}/old-runtime-token"
  local old_lookup_file="${tmpdir}/old-token-lookup.json"

  [[ -f "${init_file}" ]] || die "Vault init material not found: ${init_file}"
  jq -er '.root_token | strings | select(length > 0)' "${init_file}" >"${root_token_file}" ||
    die "Vault init material does not contain an active root token"
  chmod 0600 "${root_token_file}"
  vault_api "${root_token_file}" GET auth/token/lookup-self "" "${tmpdir}/root-lookup.json"
  jq -e '((.data.token_policies // .data.policies // []) | index("root")) != null' \
    "${tmpdir}/root-lookup.json" >/dev/null || die "configured bootstrap token is not a Vault root token"

  write_runtime_policy
  vault_api "${root_token_file}" PUT "sys/policies/acl/${runtime_policy}" \
    "${tmpdir}/policy-request.json" "${tmpdir}/policy-response.json"

  if validate_runtime_token "${VAULT_TOKEN_FILE}" >/dev/null 2>&1; then
    chown "${owner}:${group}" "${VAULT_TOKEN_FILE}"
    chmod 0600 "${VAULT_TOKEN_FILE}"
    renew_runtime_token "${VAULT_TOKEN_FILE}"
    VAULT_RUNTIME_TOKEN_OWNER="${owner}" \
    VAULT_RUNTIME_TOKEN_GROUP="${group}" \
    DEPLOY_APP_DIR="${REPO_ROOT}" \
      "${SCRIPT_DIR}/install-vault-token-renewal-timer.sh"
    log "updated the Vault runtime policy and renewed the existing token"
    return 0
  fi

  if [[ -s "${VAULT_TOKEN_FILE}" ]]; then
    install -m 0600 "${VAULT_TOKEN_FILE}" "${old_token_file}"
  fi

  jq -n \
    --arg policy "${runtime_policy}" \
    --arg period "${period_seconds}s" \
    '{
      policies: [$policy],
      period: $period,
      renewable: true,
      no_default_policy: true,
      display_name: "stuhelper-production-deploy",
      meta: {managed_by: "infra/ops/vault-runtime-token.sh"}
    }' >"${tmpdir}/token-create-request.json"
  vault_api "${root_token_file}" POST auth/token/create-orphan \
    "${tmpdir}/token-create-request.json" "${tmpdir}/token-create-response.json"
  jq -er '.auth.client_token | strings | select(length > 0)' \
    "${tmpdir}/token-create-response.json" >"${tmpdir}/new-runtime-token"
  chmod 0600 "${tmpdir}/new-runtime-token"
  validate_runtime_token "${tmpdir}/new-runtime-token" ||
    die "new Vault runtime token failed policy, TTL, or secret-read verification"

  install -d -o "${owner}" -g "${group}" -m 0700 "$(dirname "${VAULT_TOKEN_FILE}")"
  install -o "${owner}" -g "${group}" -m 0600 \
    "${tmpdir}/new-runtime-token" "${VAULT_TOKEN_FILE}.new"
  mv -f -- "${VAULT_TOKEN_FILE}.new" "${VAULT_TOKEN_FILE}"

  if [[ -s "${old_token_file}" ]] &&
     vault_api "${old_token_file}" GET auth/token/lookup-self "" "${old_lookup_file}" >/dev/null 2>&1 &&
     ! jq -e '((.data.token_policies // .data.policies // []) | index("root")) != null' \
       "${old_lookup_file}" >/dev/null; then
    jq -n --rawfile token "${old_token_file}" '{token: ($token | rtrimstr("\n"))}' \
      >"${tmpdir}/revoke-old-token.json"
    vault_api "${root_token_file}" POST auth/token/revoke \
      "${tmpdir}/revoke-old-token.json" "${tmpdir}/revoke-old-token-response.json"
  fi

  if [[ "${owner}" != "root" ]]; then
    require_cmd runuser
    runuser -u "${owner}" -- env \
      REMOTE_DEPLOY_CONFIG_FILE="${REMOTE_DEPLOY_CONFIG_FILE}" \
      "${SCRIPT_DIR}/vault-runtime-token.sh" check >/dev/null
  else
    validate_runtime_token "${VAULT_TOKEN_FILE}"
  fi

  VAULT_RUNTIME_TOKEN_OWNER="${owner}" \
  VAULT_RUNTIME_TOKEN_GROUP="${group}" \
  DEPLOY_APP_DIR="${REPO_ROOT}" \
    "${SCRIPT_DIR}/install-vault-token-renewal-timer.sh"
  log "installed and verified a least-privilege periodic Vault runtime token"
}

case "${operation}" in
  check)
    validate_runtime_token "${VAULT_TOKEN_FILE}" ||
      die "Vault runtime token failed policy, TTL, or secret-read verification"
    log "Vault runtime token policy, TTL, and exact secret reads are valid"
    ;;
  renew)
    renew_runtime_token "${VAULT_TOKEN_FILE}" || die "Vault runtime token renewal failed"
    log "Vault runtime token renewed and revalidated"
    ;;
  configure)
    configure_runtime_token
    ;;
esac
