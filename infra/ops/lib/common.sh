#!/usr/bin/env bash
set -euo pipefail

COMMON_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${COMMON_LIB_DIR}/../../.." && pwd)"
ENV_FILE="${ENV_FILE:-${REPO_ROOT}/.env}"
ENV_TEMPLATE_FILE="${ENV_TEMPLATE_FILE:-${REPO_ROOT}/.env.example}"
SECRETS_ENV_FILE="${SECRETS_ENV_FILE:-}"
GENERATED_ENV_FILE="${GENERATED_ENV_FILE:-${REPO_ROOT}/.env.generated}"
if [[ -z "${GENERATED_SECRET_ENV_FILE:-}" ]]; then
  case "${ENV_FILE}" in
    *.env.prod.shared|*.env.prod.example|*/.env.prod.shared|*/.env.prod.example)
      GENERATED_SECRET_ENV_FILE="${REPO_ROOT}/.env.prod.generated.secrets"
      ;;
    *)
      GENERATED_SECRET_ENV_FILE="${REPO_ROOT}/.env.generated.secrets"
      ;;
  esac
fi
SHARED_ENV_SECRET_REF="${SHARED_ENV_SECRET_REF:-}"
SECRETS_ENV_SECRET_REF="${SECRETS_ENV_SECRET_REF:-}"
GENERATED_ENV_SECRET_REF="${GENERATED_ENV_SECRET_REF:-}"
GENERATED_OBS_DIR="${GENERATED_OBS_DIR:-${REPO_ROOT}/infra/generated/observability}"
DEPLOY_STATE_DIR="${DEPLOY_STATE_DIR:-${REPO_ROOT}/.deploy}"
REMOTE_DEPLOY_CONFIG_FILE="${REMOTE_DEPLOY_CONFIG_FILE:-${DEPLOY_STATE_DIR}/remote.env}"
# shellcheck source=secrets.sh
source "${COMMON_LIB_DIR}/secrets.sh"

log() {
  echo "[stuhelper] $*"
}

warn() {
  echo "[stuhelper][warn] $*" >&2
}

die() {
  echo "[stuhelper][error] $*" >&2
  exit 1
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "missing required command: $1"
}

source_env_file() {
  local file="$1"
  [[ -f "${file}" ]] || return 0

  local rendered
  if ! rendered="$(python3 - "${file}" 2>&1 <<'PY'
import re
import shlex
import sys
from pathlib import Path

path = Path(sys.argv[1])
key_pattern = re.compile(r"[A-Za-z_][A-Za-z0-9_]*")

for lineno, line in enumerate(path.read_text().splitlines(), 1):
    stripped = line.strip()
    if not stripped or stripped.startswith("#"):
        continue
    if stripped.startswith("export "):
        stripped = stripped[len("export "):].lstrip()
    if "=" not in stripped:
        raise SystemExit(f"{path}:{lineno}: expected KEY=VALUE")
    key, raw_value = stripped.split("=", 1)
    key = key.strip()
    if not key_pattern.fullmatch(key):
        raise SystemExit(f"{path}:{lineno}: invalid env key: {key}")

    raw_value = raw_value.strip()
    if raw_value:
        lexer = shlex.shlex(raw_value, posix=True)
        lexer.whitespace_split = True
        lexer.commenters = ""
        try:
            parts = list(lexer)
        except ValueError as exc:
            raise SystemExit(f"{path}:{lineno}: invalid value for {key}: {exc}") from exc
        value = parts[0] if len(parts) == 1 else raw_value
    else:
        value = ""

    print(f"export {key}={shlex.quote(value)}")
PY
)"; then
    die "${rendered}"
  fi

  # shellcheck disable=SC1091
  source /dev/stdin <<<"${rendered}"
}

default_local_state_dir() {
  if [[ "$(uname -s)" == "Darwin" ]]; then
    printf '%s/Library/Application Support/StuHelper' "${HOME}"
  else
    printf '%s/stuhelper' "${XDG_STATE_HOME:-${HOME}/.local/state}"
  fi
}

normalize_backup_object_storage_env() {
  if [[ -z "${BACKUP_OBJECT_STORAGE_ENDPOINT:-}" && -n "${OBJECT_STORAGE_ENDPOINT:-}" ]]; then
    export BACKUP_OBJECT_STORAGE_ENDPOINT="${OBJECT_STORAGE_ENDPOINT}"
  fi
  export BACKUP_OBJECT_STORAGE_BUCKET="${BACKUP_OBJECT_STORAGE_BUCKET:-stuhelper-postgres-backup}"
  export BACKUP_OBJECT_STORAGE_PREFIX="${BACKUP_OBJECT_STORAGE_PREFIX:-postgres}"
  if [[ -z "${BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID:-}" && -n "${OBJECT_STORAGE_ACCESS_KEY_ID:-}" ]]; then
    export BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID="${OBJECT_STORAGE_ACCESS_KEY_ID}"
  fi
  if [[ -z "${BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY:-}" && -n "${OBJECT_STORAGE_SECRET_ACCESS_KEY:-}" ]]; then
    export BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY="${OBJECT_STORAGE_SECRET_ACCESS_KEY}"
  fi
  if [[ -z "${BACKUP_OBJECT_STORAGE_TLS_INSECURE:-}" ]]; then
    if [[ "${OBJECT_STORAGE_USE_SSL:-false}" == "true" ]]; then
      export BACKUP_OBJECT_STORAGE_TLS_INSECURE="false"
    else
      export BACKUP_OBJECT_STORAGE_TLS_INSECURE="true"
    fi
  fi
}

LOCAL_STATE_DIR="${LOCAL_STATE_DIR:-$(default_local_state_dir)}"
POSTGRES_WAL_RESTORE_DIR="${POSTGRES_WAL_RESTORE_DIR:-${LOCAL_STATE_DIR}/postgres/wal-restore}"

require_backup_object_storage_config() {
  local missing=()
  local key

  normalize_backup_object_storage_env

  for key in \
    BACKUP_OBJECT_STORAGE_ENDPOINT \
    BACKUP_OBJECT_STORAGE_BUCKET \
    BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID \
    BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY; do
    if [[ -z "${!key:-}" ]]; then
      missing+=("${key}")
    fi
  done

  if (( ${#missing[@]} > 0 )); then
    die "backup object storage configuration is incomplete; missing: ${missing[*]}"
  fi
}

verified_postgres_ssl_mode() {
  case "${1:-}" in
    verify-ca|verify-full) return 0 ;;
    *) return 1 ;;
  esac
}

require_verified_postgres_ssl_mode() {
  local key="$1"
  local value="${2:-}"
  if ! verified_postgres_ssl_mode "${value}"; then
    die "${key} must be verify-ca or verify-full for production PostgreSQL TLS (got ${value:-<empty>})"
  fi
}

require_production_postgres_url() {
  local key="$1"
  local value="${2:-}"
  local allow_plaintext="${3:-false}"
  local output

  if ! output="$(python3 - "${key}" "${value}" "${allow_plaintext}" 2>&1 <<'PY'
import sys
from urllib.parse import parse_qs, urlsplit

key = sys.argv[1]
value = sys.argv[2].strip()
allow_plaintext = sys.argv[3] == "true"

if not value:
    raise SystemExit(f"{key} must be configured for production PostgreSQL")

parsed = urlsplit(value)
if parsed.scheme not in {"postgres", "postgresql"}:
    raise SystemExit(f"{key} must be a postgres/postgresql URL")

host = (parsed.hostname or "").lower()
if host in {"localhost", "127.0.0.1", "::1"}:
    raise SystemExit(f"{key} must not point to a local/development PostgreSQL endpoint ({host})")

query = parse_qs(parsed.query, keep_blank_values=True)
ssl_modes = query.get("sslmode", [])
ssl_mode = ssl_modes[0] if ssl_modes else ""
if allow_plaintext and ssl_mode == "disable":
    raise SystemExit(0)
if ssl_mode not in {"verify-ca", "verify-full"}:
    raise SystemExit(
        f"{key} must include sslmode=verify-ca or sslmode=verify-full for production (got {ssl_mode or '<missing>'})"
    )

root_certs = query.get("sslrootcert", [])
if not root_certs or not root_certs[0].strip():
    raise SystemExit(f"{key} must include sslrootcert for production PostgreSQL TLS")
PY
)"; then
    die "${output}"
  fi
}

require_production_postgres_ssl() {
  local allow_plaintext="false"
  if [[ "${EXTERNAL_POSTGRES_ALLOW_PLAINTEXT:-false}" == "true" ]]; then
    allow_plaintext="true"
    [[ "${EXTERNAL_POSTGRES_ENABLED:-false}" == "true" ]] || die "EXTERNAL_POSTGRES_ENABLED must be true when EXTERNAL_POSTGRES_ALLOW_PLAINTEXT=true"
    [[ -n "${EXTERNAL_DATASTORE_NETWORK:-}" ]] || die "EXTERNAL_DATASTORE_NETWORK is required when EXTERNAL_POSTGRES_ALLOW_PLAINTEXT=true"
    [[ "${POSTGRES_INTERNAL_SSL_MODE:-}" == "disable" ]] || die "POSTGRES_INTERNAL_SSL_MODE must be disable when EXTERNAL_POSTGRES_ALLOW_PLAINTEXT=true"
    [[ "${DB_SSL_MODE:-}" == "disable" ]] || die "DB_SSL_MODE must be disable when EXTERNAL_POSTGRES_ALLOW_PLAINTEXT=true"
  else
    [[ "${POSTGRES_ENABLE_SSL:-}" == "on" ]] || die "POSTGRES_ENABLE_SSL must be on for production"
    require_verified_postgres_ssl_mode "POSTGRES_INTERNAL_SSL_MODE" "${POSTGRES_INTERNAL_SSL_MODE:-}"
    [[ "${DB_SSL_MODE:-}" == "verify-full" ]] || die "DB_SSL_MODE must be verify-full for production"
    [[ -n "${DB_SSL_ROOT_CERT:-}" ]] || die "DB_SSL_ROOT_CERT is required for production"
  fi

  require_production_postgres_url "DATABASE_URL" "${DATABASE_URL:-}" "${allow_plaintext}"
  require_production_postgres_url "BACKUP_DATABASE_URL" "${BACKUP_DATABASE_URL:-}" "${allow_plaintext}"
  require_production_postgres_url "REPLICATION_DATABASE_URL" "${REPLICATION_DATABASE_URL:-}" "${allow_plaintext}"
}

trim_trailing_slash() {
  local value="$1"
  printf '%s\n' "${value%/}"
}

_public_ingress_body_snippet() {
  python3 - "$1" <<'PY'
from pathlib import Path
import sys

path = Path(sys.argv[1])
if not path.exists():
    print("")
    raise SystemExit
raw = path.read_bytes()[:240]
text = raw.decode("utf-8", "replace").replace("\r", " ").replace("\n", " ")
print(" ".join(text.split()))
PY
}

_public_ingress_host() {
  python3 - "$1" <<'PY'
import sys
from urllib.parse import urlsplit

value = sys.argv[1].strip()
parsed = urlsplit(value)
print(parsed.hostname or "")
PY
}

require_public_dns_resolved() {
  local name="$1"
  local url="$2"
  local timeout="${PUBLIC_INGRESS_PREFLIGHT_TIMEOUT_SECONDS:-10}"
  local enabled="${PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_PUBLIC_DNS_ENABLED:-true}"
  local host
  local a_file
  local aaaa_file
  local error_file

  case "${enabled}" in
    true|TRUE|1|yes|YES) ;;
    false|FALSE|0|no|NO|"")
      warn "${name} public DNS preflight skipped because PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_PUBLIC_DNS_ENABLED is not true"
      return 0
      ;;
    *) die "PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_PUBLIC_DNS_ENABLED must be true or false" ;;
  esac

  host="$(_public_ingress_host "${url}")"
  [[ -n "${host}" ]] || die "${name} public DNS preflight requires a URL with hostname: ${url:-<empty>}"

  local host_kind
  if ! host_kind="$(python3 - "${name}" "${host}" 2>&1 <<'PY'
import ipaddress
import sys

name, host = sys.argv[1:3]
try:
    literal = ipaddress.ip_address(host.strip("[]").split("%", 1)[0])
except ValueError:
    print("domain")
else:
    if not literal.is_global:
        raise SystemExit(f"{name} public DNS preflight resolved to non-public IP literal for {host}")
    print("global_ip")
PY
  )"; then
    die "${host_kind}"
  fi
  if [[ "${host_kind}" == "global_ip" ]]; then
    log "${name} public DNS ready: ${host}"
    return 0
  fi

  a_file="$(mktemp)"
  aaaa_file="$(mktemp)"
  error_file="$(mktemp)"
  if ! curl -fsS --max-time "${timeout}" -o "${a_file}" "https://dns.google/resolve?name=${host}&type=A" 2>"${error_file}"; then
    local curl_error
    curl_error="$(_public_ingress_body_snippet "${error_file}")"
    rm -f "${a_file}" "${aaaa_file}" "${error_file}"
    die "${name} public DNS preflight failed for ${host} A: ${curl_error:-curl failed}"
  fi
  if ! curl -fsS --max-time "${timeout}" -o "${aaaa_file}" "https://dns.google/resolve?name=${host}&type=AAAA" 2>"${error_file}"; then
    local curl_error
    curl_error="$(_public_ingress_body_snippet "${error_file}")"
    rm -f "${a_file}" "${aaaa_file}" "${error_file}"
    die "${name} public DNS preflight failed for ${host} AAAA: ${curl_error:-curl failed}"
  fi
  rm -f "${error_file}"

  local dns_validation_output
  if ! dns_validation_output="$(python3 - "${name}" "${host}" "${a_file}" "${aaaa_file}" 2>&1 <<'PY'
import ipaddress
import json
import sys
from pathlib import Path

name, host, a_path, aaaa_path = sys.argv[1:5]

def fail(message: str) -> None:
    raise SystemExit(message)

def load(path: str) -> dict:
    try:
        payload = json.loads(Path(path).read_text())
    except json.JSONDecodeError as exc:
        fail(f"{name} public DNS preflight got invalid JSON for {host}: {exc}")
    if not isinstance(payload, dict):
        fail(f"{name} public DNS preflight got non-object JSON for {host}")
    return payload

def is_global(value: str) -> bool:
    try:
        return ipaddress.ip_address(value.split("%", 1)[0]).is_global
    except ValueError:
        return False

def ip_literal(value: str):
    try:
        return ipaddress.ip_address(value.strip("[]").split("%", 1)[0])
    except ValueError:
        return None

literal = ip_literal(host)
if literal is not None:
    if not literal.is_global:
        fail(f"{name} public DNS preflight resolved to non-public IP literal for {host}")
    raise SystemExit(0)

payloads = {"A": load(a_path), "AAAA": load(aaaa_path)}
addresses: list[str] = []
statuses: dict[str, object] = {}
for rrtype, rrnumber in (("A", 1), ("AAAA", 28)):
    payload = payloads[rrtype]
    statuses[rrtype] = payload.get("Status")
    for answer in payload.get("Answer") or []:
        if isinstance(answer, dict) and answer.get("type") == rrnumber and isinstance(answer.get("data"), str):
            addresses.append(answer["data"])

addresses = sorted(set(addresses))
if not addresses:
    if any(status == 3 for status in statuses.values()):
        fail(f"{name} public DNS preflight failed for {host}: NXDOMAIN from public resolver")
    fail(f"{name} public DNS preflight failed for {host}: no public A/AAAA records from public resolver")

non_public = [address for address in addresses if not is_global(address)]
if non_public:
    fail(f"{name} public DNS preflight failed for {host}: non-public A/AAAA records: {', '.join(non_public)}")
PY
  )"; then
    rm -f "${a_file}" "${aaaa_file}"
    die "${dns_validation_output}"
  fi
  rm -f "${a_file}" "${aaaa_file}"
  log "${name} public DNS ready: ${host}"
}

require_public_http_reachable() {
  local name="$1"
  local url="$2"
  local timeout="${PUBLIC_INGRESS_PREFLIGHT_TIMEOUT_SECONDS:-10}"
  local error_file
  local status
  local curl_error

  [[ -n "${url}" ]] || die "${name} URL is required for public ingress preflight"
  error_file="$(mktemp)"
  if ! status="$(curl -sS --max-time "${timeout}" -o /dev/null -w '%{http_code}' "${url}" 2>"${error_file}")"; then
    curl_error="$(_public_ingress_body_snippet "${error_file}")"
    rm -f "${error_file}"
    die "${name} public TLS/HTTP preflight failed for ${url}: ${curl_error:-curl failed}"
  fi
  rm -f "${error_file}"
  if [[ ! "${status}" =~ ^[0-9][0-9][0-9]$ ]] || [[ "${status}" == "000" ]]; then
    die "${name} public TLS/HTTP preflight returned invalid HTTP status ${status:-<empty>} for ${url}"
  fi
  log "${name} public ingress TLS reachable: ${url} (HTTP ${status})"
}

require_public_oidc_discovery() {
  local name="$1"
  local issuer
  issuer="$(trim_trailing_slash "$2")"
  local timeout="${PUBLIC_INGRESS_PREFLIGHT_TIMEOUT_SECONDS:-10}"
  local url="${issuer}/.well-known/openid-configuration"
  local output_file
  local error_file
  local status
  local curl_error
  local body

  [[ -n "${issuer}" ]] || die "${name} issuer is required for public OIDC discovery preflight"
  output_file="$(mktemp)"
  error_file="$(mktemp)"
  if ! status="$(curl -sS --max-time "${timeout}" -o "${output_file}" -w '%{http_code}' "${url}" 2>"${error_file}")"; then
    curl_error="$(_public_ingress_body_snippet "${error_file}")"
    rm -f "${output_file}" "${error_file}"
    die "${name} discovery preflight failed for ${url}: ${curl_error:-curl failed}"
  fi
  curl_error="$(_public_ingress_body_snippet "${error_file}")"
  rm -f "${error_file}"
  if [[ ! "${status}" =~ ^[0-9][0-9][0-9]$ ]] || (( status < 200 || status >= 300 )); then
    body="$(_public_ingress_body_snippet "${output_file}")"
    rm -f "${output_file}"
    die "${name} discovery preflight returned HTTP ${status} for ${url}: ${body:-${curl_error:-<empty body>}}"
  fi
  if ! jq -e --arg issuer "${issuer}" '
    type == "object"
    and .issuer == $issuer
    and (.authorization_endpoint | type == "string" and length > 0)
    and (.token_endpoint | type == "string" and length > 0)
    and (.jwks_uri | type == "string" and length > 0)
  ' "${output_file}" >/dev/null; then
    body="$(_public_ingress_body_snippet "${output_file}")"
    rm -f "${output_file}"
    die "${name} discovery metadata is invalid for ${url}: ${body:-<empty body>}"
  fi
  rm -f "${output_file}"
  log "${name} public OIDC discovery ready: ${url}"
}

require_public_jwks() {
  local name="$1"
  local url="$2"
  local timeout="${PUBLIC_INGRESS_PREFLIGHT_TIMEOUT_SECONDS:-10}"
  local output_file
  local error_file
  local status
  local curl_error
  local body

  [[ -n "${url}" ]] || die "${name} JWKS URL is required for public identity ingress preflight"
  output_file="$(mktemp)"
  error_file="$(mktemp)"
  if ! status="$(curl -sS --max-time "${timeout}" -o "${output_file}" -w '%{http_code}' "${url}" 2>"${error_file}")"; then
    curl_error="$(_public_ingress_body_snippet "${error_file}")"
    rm -f "${output_file}" "${error_file}"
    die "${name} JWKS preflight failed for ${url}: ${curl_error:-curl failed}"
  fi
  curl_error="$(_public_ingress_body_snippet "${error_file}")"
  rm -f "${error_file}"
  if [[ ! "${status}" =~ ^[0-9][0-9][0-9]$ ]] || (( status < 200 || status >= 300 )); then
    body="$(_public_ingress_body_snippet "${output_file}")"
    rm -f "${output_file}"
    die "${name} JWKS preflight returned HTTP ${status} for ${url}: ${body:-${curl_error:-<empty body>}}"
  fi
  if ! jq -e 'type == "object" and (.keys | type == "array")' "${output_file}" >/dev/null; then
    body="$(_public_ingress_body_snippet "${output_file}")"
    rm -f "${output_file}"
    die "${name} JWKS metadata is invalid for ${url}: ${body:-<empty body>}"
  fi
  rm -f "${output_file}"
  log "${name} public JWKS ready: ${url}"
}

public_oidc_jwks_uri() {
  local issuer
  issuer="$(trim_trailing_slash "$1")"
  local timeout="${PUBLIC_INGRESS_PREFLIGHT_TIMEOUT_SECONDS:-10}"
  local metadata

  [[ -n "${issuer}" ]] || return 1
  if ! metadata="$(curl -fsS --max-time "${timeout}" "${issuer}/.well-known/openid-configuration")"; then
    return 1
  fi
  printf '%s\n' "${metadata}" | jq -r '.jwks_uri // empty'
}

require_public_identity_ingress_preflight() {
  if [[ "${PUBLIC_INGRESS_PREFLIGHT_ENABLED:-true}" != "true" ]]; then
    warn "public identity ingress preflight skipped because PUBLIC_INGRESS_PREFLIGHT_ENABLED is not true"
    return 0
  fi

  require_public_dns_resolved "Web" "$(trim_trailing_slash "${WEB_PUBLIC_URL:-}")"
  require_public_dns_resolved "Identity" "$(trim_trailing_slash "${IDENTITY_ISSUER:-}")"
  require_public_dns_resolved "Casdoor" "$(trim_trailing_slash "${CASDOOR_ISSUER:-}")"
  require_public_http_reachable "Web" "$(trim_trailing_slash "${WEB_PUBLIC_URL:-}")"
  require_public_http_reachable "Identity" "$(trim_trailing_slash "${IDENTITY_ISSUER:-}")"
  require_public_oidc_discovery "Casdoor" "${CASDOOR_ISSUER:-}"
  local casdoor_jwks_uri
  if ! casdoor_jwks_uri="$(public_oidc_jwks_uri "${CASDOOR_ISSUER:-}")"; then
    die "Casdoor JWKS URI preflight failed for ${CASDOOR_ISSUER:-}: discovery did not expose jwks_uri"
  fi
  require_public_jwks "Casdoor" "${casdoor_jwks_uri}"
}

require_public_ingress_config_preflight() {
  if [[ "${PUBLIC_INGRESS_CONFIG_PREFLIGHT_ENABLED:-true}" != "true" ]]; then
    warn "public Nginx ingress config preflight skipped because PUBLIC_INGRESS_CONFIG_PREFLIGHT_ENABLED is not true"
    return 0
  fi

  "${COMMON_LIB_DIR}/../nginx-public-ingress-preflight.sh"
}

ensure_env_file() {
  local template_file="${1:-${ENV_TEMPLATE_FILE}}"
  [[ -n "${template_file}" ]] || die "ENV_TEMPLATE_FILE must not be empty"
  [[ -f "${template_file}" ]] || die "missing env template: ${template_file}"
  if [[ ! -f "${ENV_FILE}" ]]; then
    cp "${template_file}" "${ENV_FILE}"
    log "created ${ENV_FILE} from $(basename "${template_file}")"
  fi
}

ensure_generated_files() {
  mkdir -p "${GENERATED_OBS_DIR}/prometheus" "${GENERATED_OBS_DIR}/alertmanager"
  touch "${GENERATED_ENV_FILE}"
  touch "${GENERATED_SECRET_ENV_FILE}"
}

should_source_generated_secret_from_backend() {
  [[ -n "${GENERATED_ENV_SECRET_REF:-}" ]] && [[ -n "${SECRET_BACKEND:-}" ]] && [[ "${SECRET_BACKEND:-}" != "none" ]] && [[ "${SECRET_BACKEND:-}" != "file" ]]
}

source_generated_secret_env() {
  if should_source_generated_secret_from_backend; then
    local rendered
    if ! rendered="$(secret_backend_read_to_stdout "${GENERATED_ENV_SECRET_REF}")"; then
      die "failed to read generated secret env from ${SECRET_BACKEND}: ${GENERATED_ENV_SECRET_REF}"
    fi
    if [[ -n "${rendered}" ]]; then
      local tmp_env
      tmp_env="$(mktemp)"
      printf '%s\n' "${rendered}" >"${tmp_env}"
      source_env_file "${tmp_env}"
      rm -f "${tmp_env}"
    fi
    return 0
  fi

  if [[ -f "${GENERATED_SECRET_ENV_FILE}" ]]; then
    source_env_file "${GENERATED_SECRET_ENV_FILE}"
  fi
}

materialize_secret_env_inputs() {
  if [[ -n "${SHARED_ENV_SECRET_REF}" ]]; then
    materialize_secret_env_file "${SHARED_ENV_SECRET_REF}" "${ENV_FILE}"
  fi
  if [[ -n "${SECRETS_ENV_SECRET_REF}" && -n "${SECRETS_ENV_FILE}" ]]; then
    materialize_secret_env_file "${SECRETS_ENV_SECRET_REF}" "${SECRETS_ENV_FILE}"
  fi
  if [[ -n "${GENERATED_ENV_SECRET_REF}" ]] && ! should_source_generated_secret_from_backend; then
    materialize_secret_env_file "${GENERATED_ENV_SECRET_REF}" "${GENERATED_SECRET_ENV_FILE}"
  fi
}

ensure_secrets_env_file() {
  if [[ -n "${SECRETS_ENV_FILE}" ]]; then
    mkdir -p "$(dirname "${SECRETS_ENV_FILE}")"
    touch "${SECRETS_ENV_FILE}"
  fi
}

load_env() {
  ensure_env_file
  ensure_secrets_env_file
  ensure_generated_files
  materialize_secret_env_inputs
  local preserved_tag="${TAG-__STUHELPER_UNSET__}"
  local preserved_rollback_tag="${ROLLBACK_TAG-__STUHELPER_UNSET__}"
  local preserved_backend_image_ref="${BACKEND_IMAGE_REF-__STUHELPER_UNSET__}"
  local preserved_frontend_image_ref="${FRONTEND_IMAGE_REF-__STUHELPER_UNSET__}"
  local preserved_admin_image_ref="${ADMIN_IMAGE_REF-__STUHELPER_UNSET__}"
  set -a
  source_env_file "${ENV_FILE}"
  if [[ -n "${SECRETS_ENV_FILE}" && -f "${SECRETS_ENV_FILE}" ]]; then
    source_env_file "${SECRETS_ENV_FILE}"
  fi
  source_env_file "${GENERATED_ENV_FILE}"
  source_generated_secret_env
  normalize_backup_object_storage_env
  export POSTGRES_WAL_ARCHIVE_VOLUME_NAME="${POSTGRES_WAL_ARCHIVE_VOLUME_NAME:-${STACK_NAME:-stuhelper}-postgres-wal-archive}"
  if [[ "${preserved_tag}" != "__STUHELPER_UNSET__" ]]; then export TAG="${preserved_tag}"; fi
  if [[ "${preserved_rollback_tag}" != "__STUHELPER_UNSET__" ]]; then export ROLLBACK_TAG="${preserved_rollback_tag}"; fi
  if [[ "${preserved_backend_image_ref}" != "__STUHELPER_UNSET__" ]]; then export BACKEND_IMAGE_REF="${preserved_backend_image_ref}"; fi
  if [[ "${preserved_frontend_image_ref}" != "__STUHELPER_UNSET__" ]]; then export FRONTEND_IMAGE_REF="${preserved_frontend_image_ref}"; fi
  if [[ "${preserved_admin_image_ref}" != "__STUHELPER_UNSET__" ]]; then export ADMIN_IMAGE_REF="${preserved_admin_image_ref}"; fi
  set +a
}

load_remote_deploy_config() {
  local config_file="${1:-${REMOTE_DEPLOY_CONFIG_FILE}}"
  [[ -n "${config_file}" ]] || die "REMOTE_DEPLOY_CONFIG_FILE must not be empty"
  [[ -f "${config_file}" ]] || die "missing remote deploy config: ${config_file} (run ./infra/ops/init-remote-deploy-config.sh on the target host first)"

  local preserved_tag="${TAG-__STUHELPER_UNSET__}"
  local preserved_rollback_tag="${ROLLBACK_TAG-__STUHELPER_UNSET__}"
  local preserved_backend_image_ref="${BACKEND_IMAGE_REF-__STUHELPER_UNSET__}"
  local preserved_frontend_image_ref="${FRONTEND_IMAGE_REF-__STUHELPER_UNSET__}"
  local preserved_admin_image_ref="${ADMIN_IMAGE_REF-__STUHELPER_UNSET__}"
  local preserved_registry_username="${REGISTRY_USERNAME-__STUHELPER_UNSET__}"
  local preserved_registry_password="${REGISTRY_PASSWORD-__STUHELPER_UNSET__}"

  set -a
  source_env_file "${config_file}"
  if [[ "${preserved_tag}" != "__STUHELPER_UNSET__" ]]; then export TAG="${preserved_tag}"; fi
  if [[ "${preserved_rollback_tag}" != "__STUHELPER_UNSET__" ]]; then export ROLLBACK_TAG="${preserved_rollback_tag}"; fi
  if [[ "${preserved_backend_image_ref}" != "__STUHELPER_UNSET__" ]]; then export BACKEND_IMAGE_REF="${preserved_backend_image_ref}"; fi
  if [[ "${preserved_frontend_image_ref}" != "__STUHELPER_UNSET__" ]]; then export FRONTEND_IMAGE_REF="${preserved_frontend_image_ref}"; fi
  if [[ "${preserved_admin_image_ref}" != "__STUHELPER_UNSET__" ]]; then export ADMIN_IMAGE_REF="${preserved_admin_image_ref}"; fi
  if [[ "${preserved_registry_username}" != "__STUHELPER_UNSET__" ]]; then export REGISTRY_USERNAME="${preserved_registry_username}"; fi
  if [[ "${preserved_registry_password}" != "__STUHELPER_UNSET__" ]]; then export REGISTRY_PASSWORD="${preserved_registry_password}"; fi
  set +a
}

compose() {
  (
    cd "${REPO_ROOT}" && \
    compose_files=(-f "${REPO_ROOT}/docker-compose.yml") && \
    if [[ -f "${REPO_ROOT}/docker-compose.observability.yml" ]]; then compose_files+=(-f "${REPO_ROOT}/docker-compose.observability.yml"); fi && \
    if [[ " $* " == *" --profile prod "* && -f "${REPO_ROOT}/docker-compose.prod.yml" ]]; then compose_files+=(-f "${REPO_ROOT}/docker-compose.prod.yml"); fi && \
    if [[ -n "${EXTERNAL_DATASTORE_NETWORK:-}" && "${EXTERNAL_POSTGRES_ENABLED:-false}" == "true" && -f "${REPO_ROOT}/docker-compose.external-datastore.yml" ]]; then compose_files+=(-f "${REPO_ROOT}/docker-compose.external-datastore.yml"); fi && \
    preserved_tag="${TAG-__STUHELPER_UNSET__}" && \
    preserved_rollback_tag="${ROLLBACK_TAG-__STUHELPER_UNSET__}" && \
    preserved_backend_image_ref="${BACKEND_IMAGE_REF-__STUHELPER_UNSET__}" && \
    preserved_frontend_image_ref="${FRONTEND_IMAGE_REF-__STUHELPER_UNSET__}" && \
    preserved_admin_image_ref="${ADMIN_IMAGE_REF-__STUHELPER_UNSET__}" && \
    set -a && \
    source_env_file "${ENV_FILE}" && \
    if [[ -n "${SECRETS_ENV_FILE}" && -f "${SECRETS_ENV_FILE}" ]]; then source_env_file "${SECRETS_ENV_FILE}"; fi && \
    if [[ -f "${GENERATED_ENV_FILE}" ]]; then source_env_file "${GENERATED_ENV_FILE}"; fi && \
    source_generated_secret_env && \
    if [[ "${preserved_tag}" != "__STUHELPER_UNSET__" ]]; then export TAG="${preserved_tag}"; fi && \
    if [[ "${preserved_rollback_tag}" != "__STUHELPER_UNSET__" ]]; then export ROLLBACK_TAG="${preserved_rollback_tag}"; fi && \
    if [[ "${preserved_backend_image_ref}" != "__STUHELPER_UNSET__" ]]; then export BACKEND_IMAGE_REF="${preserved_backend_image_ref}"; fi && \
    if [[ "${preserved_frontend_image_ref}" != "__STUHELPER_UNSET__" ]]; then export FRONTEND_IMAGE_REF="${preserved_frontend_image_ref}"; fi && \
    if [[ "${preserved_admin_image_ref}" != "__STUHELPER_UNSET__" ]]; then export ADMIN_IMAGE_REF="${preserved_admin_image_ref}"; fi && \
    set +a && \
    if [[ -z "${BACKUP_OBJECT_STORAGE_ENDPOINT:-}" && -n "${OBJECT_STORAGE_ENDPOINT:-}" ]]; then export BACKUP_OBJECT_STORAGE_ENDPOINT="${OBJECT_STORAGE_ENDPOINT}"; fi && \
    export BACKUP_OBJECT_STORAGE_BUCKET="${BACKUP_OBJECT_STORAGE_BUCKET:-stuhelper-postgres-backup}" && \
    export BACKUP_OBJECT_STORAGE_PREFIX="${BACKUP_OBJECT_STORAGE_PREFIX:-postgres}" && \
    if [[ -z "${BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID:-}" && -n "${OBJECT_STORAGE_ACCESS_KEY_ID:-}" ]]; then export BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID="${OBJECT_STORAGE_ACCESS_KEY_ID}"; fi && \
    if [[ -z "${BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY:-}" && -n "${OBJECT_STORAGE_SECRET_ACCESS_KEY:-}" ]]; then export BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY="${OBJECT_STORAGE_SECRET_ACCESS_KEY}"; fi && \
    if [[ -z "${BACKUP_OBJECT_STORAGE_TLS_INSECURE:-}" ]]; then \
      if [[ "${OBJECT_STORAGE_USE_SSL:-false}" == "true" ]]; then export BACKUP_OBJECT_STORAGE_TLS_INSECURE="false"; else export BACKUP_OBJECT_STORAGE_TLS_INSECURE="true"; fi; \
    fi && \
    export POSTGRES_WAL_ARCHIVE_VOLUME_NAME="${POSTGRES_WAL_ARCHIVE_VOLUME_NAME:-${STACK_NAME:-stuhelper}-postgres-wal-archive}" && \
    ENV_FILE_PATH="${ENV_FILE}" \
    SECRETS_ENV_FILE_PATH="${SECRETS_ENV_FILE}" \
    GENERATED_ENV_FILE_PATH="${GENERATED_ENV_FILE}" \
    GENERATED_SECRET_ENV_FILE_PATH="${GENERATED_SECRET_ENV_FILE}" \
    BACKEND_IMAGE_REF="${BACKEND_IMAGE_REF:-}" \
    FRONTEND_IMAGE_REF="${FRONTEND_IMAGE_REF:-}" \
    ADMIN_IMAGE_REF="${ADMIN_IMAGE_REF:-}" \
    docker compose "${compose_files[@]}" --env-file "${ENV_FILE}" "$@"
  )
}

wait_for_http() {
  local name="$1"
  local url="$2"
  local retries="${3:-60}"
  local sleep_seconds="${4:-2}"
  local i

  for ((i = 1; i <= retries; i++)); do
    if curl -fsS --max-time 5 "${url}" >/dev/null 2>&1; then
      log "${name} is ready: ${url}"
      return 0
    fi
    sleep "${sleep_seconds}"
  done

  die "${name} did not become ready in time: ${url}"
}

upsert_env_file() {
  local file="$1"
  local key="$2"
  local value="$3"
  local python_file="${file}"

  if [[ "$(uname -s)" =~ ^(MINGW|MSYS|CYGWIN) ]] && command -v cygpath >/dev/null 2>&1; then
    python_file="$(cygpath -w "${file}")"
  fi

  MSYS2_ARG_CONV_EXCL="*" python3 - "$python_file" "$key" "$value" <<'PY'
from pathlib import Path
import sys

path = Path(sys.argv[1])
key = sys.argv[2]
value = sys.argv[3]

lines = path.read_text().splitlines() if path.exists() else []
updated = False
for idx, line in enumerate(lines):
    if line.startswith(f"{key}="):
        lines[idx] = f"{key}={value}"
        updated = True
        break
if not updated:
    lines.append(f"{key}={value}")
path.write_text("\n".join(lines) + "\n")
PY
}

random_hex() {
  local nbytes="${1:-32}"
  python3 - "$nbytes" <<'PY'
import secrets
import sys
print(secrets.token_hex(int(sys.argv[1])))
PY
}

git_tag_default() {
  git -C "${REPO_ROOT}" rev-parse --short HEAD 2>/dev/null || echo "local"
}

derive_release_id_from_image_ref() {
  local image_ref="${1:-}"
  python3 - "${image_ref}" <<'PY'
import sys

ref = sys.argv[1].strip()
if not ref:
    raise SystemExit(1)

if "@sha256:" in ref:
    print(ref.split("@sha256:", 1)[1][:12])
    raise SystemExit(0)

image, sep, tag = ref.rpartition(":")
if not sep or "/" not in image:
    raise SystemExit(1)

print(tag)
PY
}

resolve_registry_credentials() {
  if [[ -z "${REGISTRY_USERNAME:-}" && -n "${REGISTRY_USERNAME_SECRET_REF:-}" ]]; then
    REGISTRY_USERNAME="$(materialize_secret_value "${REGISTRY_USERNAME_SECRET_REF}")"
    export REGISTRY_USERNAME
  fi
  if [[ -z "${REGISTRY_PASSWORD:-}" && -n "${REGISTRY_PASSWORD_SECRET_REF:-}" ]]; then
    REGISTRY_PASSWORD="$(materialize_secret_value "${REGISTRY_PASSWORD_SECRET_REF}")"
    export REGISTRY_PASSWORD
  fi
}

docker_registry_login() {
  [[ -n "${REGISTRY:-}" ]] || die "REGISTRY is required"
  resolve_registry_credentials
  [[ -n "${REGISTRY_USERNAME:-}" ]] || die "REGISTRY_USERNAME or REGISTRY_USERNAME_SECRET_REF is required"
  [[ -n "${REGISTRY_PASSWORD:-}" ]] || die "REGISTRY_PASSWORD or REGISTRY_PASSWORD_SECRET_REF is required"

  echo "${REGISTRY_PASSWORD}" | docker login "${REGISTRY}" --username "${REGISTRY_USERNAME}" --password-stdin >/dev/null
}

record_release() {
  local tag="$1"
  mkdir -p "${DEPLOY_STATE_DIR}"
  mkdir -p "${DEPLOY_STATE_DIR}/releases"
  local now
  now="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  printf '%s\t%s\n' "${now}" "${tag}" >> "${DEPLOY_STATE_DIR}/releases.log"
  cat > "${DEPLOY_STATE_DIR}/current-release.env" <<EOF
TAG=${tag}
DEPLOYED_AT=${now}
BACKEND_IMAGE_REF=${BACKEND_IMAGE_REF:-}
FRONTEND_IMAGE_REF=${FRONTEND_IMAGE_REF:-}
ADMIN_IMAGE_REF=${ADMIN_IMAGE_REF:-}
EOF
  cat > "${DEPLOY_STATE_DIR}/releases/${tag}.env" <<EOF
TAG=${tag}
DEPLOYED_AT=${now}
BACKEND_IMAGE_REF=${BACKEND_IMAGE_REF:-}
FRONTEND_IMAGE_REF=${FRONTEND_IMAGE_REF:-}
ADMIN_IMAGE_REF=${ADMIN_IMAGE_REF:-}
EOF
}

resolve_previous_release_tag() {
  local current_tag="${1:-}"
  local releases_file="${DEPLOY_STATE_DIR}/releases.log"
  [[ -f "${releases_file}" ]] || return 1
  python3 - "${releases_file}" "${current_tag}" <<'PY'
from pathlib import Path
import sys

path = Path(sys.argv[1])
current = sys.argv[2]
lines = path.read_text().splitlines()
for line in reversed(lines):
    parts = line.split("\t")
    if len(parts) >= 2 and parts[1] and parts[1] != current:
        print(parts[1])
        raise SystemExit(0)
raise SystemExit(1)
PY
}
