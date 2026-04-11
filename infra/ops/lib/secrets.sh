#!/usr/bin/env bash

SECRET_BACKEND="${SECRET_BACKEND:-none}"
SECRET_FILE_ROOT="${SECRET_FILE_ROOT:-${REPO_ROOT}/infra/generated/secrets}"
VAULT_ADDR="${VAULT_ADDR:-}"
VAULT_NAMESPACE="${VAULT_NAMESPACE:-}"
VAULT_TOKEN_FILE="${VAULT_TOKEN_FILE:-}"
VAULT_KV_MOUNT="${VAULT_KV_MOUNT:-secret}"

parse_secret_ref() {
  local ref="$1"
  python3 - "$ref" "$VAULT_KV_MOUNT" <<'PY'
import sys
ref = sys.argv[1].strip()
default_mount = sys.argv[2].strip() or 'secret'
field = 'value'
if '#' in ref:
    ref, field = ref.rsplit('#', 1)
ref = ref.strip('/')
parts = [segment for segment in ref.split('/') if segment]
if not parts:
    raise SystemExit('secret ref must not be empty')
if len(parts) == 1:
    mount = default_mount
    path = parts[0]
else:
    mount = parts[0]
    path = '/'.join(parts[1:])
print(mount)
print(path)
print(field or 'value')
PY
}

secret_backend_enabled() {
  [[ -n "${SECRET_BACKEND}" && "${SECRET_BACKEND}" != "none" ]]
}

vault_token() {
  if [[ -n "${VAULT_TOKEN:-}" ]]; then
    printf '%s' "${VAULT_TOKEN}"
    return 0
  fi
  [[ -n "${VAULT_TOKEN_FILE}" ]] || die "VAULT_TOKEN or VAULT_TOKEN_FILE is required for SECRET_BACKEND=vault-kv-v2"
  [[ -f "${VAULT_TOKEN_FILE}" ]] || die "vault token file not found: ${VAULT_TOKEN_FILE}"
  tr -d '\r\n' < "${VAULT_TOKEN_FILE}"
}

secret_file_path() {
  local ref="$1"
  if [[ "${ref}" = /* ]]; then
    printf '%s\n' "${ref}"
    return 0
  fi
  printf '%s/%s\n' "${SECRET_FILE_ROOT%/}" "${ref}"
}

secret_backend_read_to_file() {
  local ref="$1"
  local dest="$2"
  mkdir -p "$(dirname "${dest}")"

  case "${SECRET_BACKEND}" in
    none|"")
      die "secret backend is disabled; cannot resolve secret ref ${ref}"
      ;;
    file)
      local src
      src="$(secret_file_path "${ref}")"
      [[ -f "${src}" ]] || die "secret ref not found: ${src}"
      if [[ "$(cd "$(dirname "${src}")" && pwd)/$(basename "${src}")" = "$(cd "$(dirname "${dest}")" && pwd)/$(basename "${dest}")" ]]; then
        chmod 600 "${dest}" 2>/dev/null || true
      else
        install -m 600 "${src}" "${dest}"
      fi
      ;;
    vault-kv-v2)
      require_cmd curl
      require_cmd jq
      require_cmd python3
      [[ -n "${VAULT_ADDR}" ]] || die "VAULT_ADDR is required for SECRET_BACKEND=vault-kv-v2"
      local mount path field token response
      mapfile -t __secret_ref_parts < <(parse_secret_ref "${ref}")
      mount="${__secret_ref_parts[0]}"
      path="${__secret_ref_parts[1]}"
      field="${__secret_ref_parts[2]}"
      token="$(vault_token)"
      response="$(mktemp)"
      local headers=(-H "X-Vault-Token: ${token}")
      if [[ -n "${VAULT_NAMESPACE}" ]]; then
        headers+=( -H "X-Vault-Namespace: ${VAULT_NAMESPACE}" )
      fi
      curl --fail --silent --show-error \
        "${headers[@]}" \
        "${VAULT_ADDR%/}/v1/${mount}/data/${path}" > "${response}"
      python3 - "${response}" "${field}" "${dest}" <<'PY'
import json
import pathlib
import sys

payload = json.loads(pathlib.Path(sys.argv[1]).read_text())
field = sys.argv[2]
dest = pathlib.Path(sys.argv[3])
try:
    value = payload['data']['data'][field]
except Exception as exc:
    raise SystemExit(f'vault secret field not found: {field} ({exc})')
if not isinstance(value, str):
    raise SystemExit(f'vault secret field must be a string: {field}')
dest.write_text(value if value.endswith('\n') else value + '\n')
dest.chmod(0o600)
PY
      rm -f "${response}"
      ;;
    *)
      die "unsupported SECRET_BACKEND=${SECRET_BACKEND}"
      ;;
  esac
}

secret_backend_write_from_file() {
  local ref="$1"
  local src="$2"
  [[ -f "${src}" ]] || die "secret source file not found: ${src}"

  case "${SECRET_BACKEND}" in
    none|"")
      die "secret backend is disabled; cannot persist secret ref ${ref}"
      ;;
    file)
      local dest
      dest="$(secret_file_path "${ref}")"
      mkdir -p "$(dirname "${dest}")"
      if [[ "$(cd "$(dirname "${src}")" && pwd)/$(basename "${src}")" = "$(cd "$(dirname "${dest}")" && pwd)/$(basename "${dest}")" ]]; then
        chmod 600 "${dest}" 2>/dev/null || true
      else
        install -m 600 "${src}" "${dest}"
      fi
      ;;
    vault-kv-v2)
      require_cmd curl
      require_cmd jq
      require_cmd python3
      [[ -n "${VAULT_ADDR}" ]] || die "VAULT_ADDR is required for SECRET_BACKEND=vault-kv-v2"
      local mount path field token payload_file
      mapfile -t __secret_ref_parts < <(parse_secret_ref "${ref}")
      mount="${__secret_ref_parts[0]}"
      path="${__secret_ref_parts[1]}"
      field="${__secret_ref_parts[2]}"
      token="$(vault_token)"
      payload_file="$(mktemp)"
      python3 - "${src}" "${field}" "${payload_file}" <<'PY'
import json
import pathlib
import sys
src = pathlib.Path(sys.argv[1]).read_text()
field = sys.argv[2]
payload = {'data': {field: src}}
pathlib.Path(sys.argv[3]).write_text(json.dumps(payload))
PY
      local headers=(-H "X-Vault-Token: ${token}" -H 'Content-Type: application/json')
      if [[ -n "${VAULT_NAMESPACE}" ]]; then
        headers+=( -H "X-Vault-Namespace: ${VAULT_NAMESPACE}" )
      fi
      curl --fail --silent --show-error \
        -X POST \
        "${headers[@]}" \
        --data @"${payload_file}" \
        "${VAULT_ADDR%/}/v1/${mount}/data/${path}" >/dev/null
      rm -f "${payload_file}"
      ;;
    *)
      die "unsupported SECRET_BACKEND=${SECRET_BACKEND}"
      ;;
  esac
}

materialize_secret_env_file() {
  local ref="$1"
  local dest="$2"
  [[ -n "${ref}" ]] || return 0
  secret_backend_read_to_file "${ref}" "${dest}"
}


materialize_secret_value() {
  local ref="$1"
  local tmp
  tmp="$(mktemp)"
  secret_backend_read_to_file "${ref}" "${tmp}"
  tr -d '
' < "${tmp}"
  rm -f "${tmp}"
}
