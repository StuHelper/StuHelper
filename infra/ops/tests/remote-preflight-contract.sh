#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
PREFLIGHT_FILE="${REPO_ROOT}/infra/ops/remote-preflight.sh"
COMMON_LIB_FILE="${REPO_ROOT}/infra/ops/lib/common.sh"

fail() {
  echo "[remote-preflight-contract][error] $*" >&2
  exit 1
}

assert_contains() {
  local file="$1"
  local pattern="$2"
  if ! grep -Eq "${pattern}" "${file}"; then
    fail "expected ${file} to contain pattern: ${pattern}"
  fi
}

assert_not_contains() {
  local file="$1"
  local pattern="$2"
  if grep -Eq "${pattern}" "${file}"; then
    fail "expected ${file} to not contain pattern: ${pattern}"
  fi
}

line_number() {
  local file="$1"
  local pattern="$2"
  local line
  line="$(grep -nF -- "${pattern}" "${file}" | head -n1 | cut -d: -f1)"
  [[ -n "${line}" ]] || fail "expected ${file} to contain pattern: ${pattern}"
  printf '%s\n' "${line}"
}

assert_contains "${PREFLIGHT_FILE}" 'compose run --rm --no-deps -T postgres'
assert_contains "${PREFLIGHT_FILE}" 'die "备份数据库连通性检查失败'
assert_not_contains "${PREFLIGHT_FILE}" 'docker run --rm --network host "\$\{pg_image\}"'
assert_contains "${COMMON_LIB_FILE}" 'failed to read generated secret env from \$\{SECRET_BACKEND\}'
assert_not_contains "${COMMON_LIB_FILE}" 'secret_backend_read_to_stdout "\$\{GENERATED_ENV_SECRET_REF\}" 2>/dev/null'
assert_contains "${COMMON_LIB_FILE}" 'source_env_file\(\)'
assert_contains "${COMMON_LIB_FILE}" 'shlex\.quote\(value\)'
assert_contains "${COMMON_LIB_FILE}" 'require_production_postgres_ssl\(\)'
assert_contains "${COMMON_LIB_FILE}" 'require_public_identity_ingress_preflight\(\)'
assert_contains "${COMMON_LIB_FILE}" 'require_public_ingress_config_preflight\(\)'
assert_contains "${COMMON_LIB_FILE}" 'nginx-public-ingress-preflight\.sh'
assert_contains "${COMMON_LIB_FILE}" 'require_public_http_reachable\(\)'
assert_contains "${COMMON_LIB_FILE}" 'require_public_oidc_discovery\(\)'
assert_contains "${COMMON_LIB_FILE}" 'require_public_dns_resolved\(\)'
assert_contains "${COMMON_LIB_FILE}" 'dns\.google/resolve'
assert_contains "${COMMON_LIB_FILE}" 'PUBLIC_INGRESS_CONFIG_PREFLIGHT_ENABLED'
assert_contains "${COMMON_LIB_FILE}" 'PUBLIC_INGRESS_PREFLIGHT_ENABLED'
assert_contains "${COMMON_LIB_FILE}" 'PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_PUBLIC_DNS_ENABLED'
assert_contains "${COMMON_LIB_FILE}" 'public DNS preflight failed'
assert_contains "${COMMON_LIB_FILE}" 'public OIDC discovery ready'
assert_contains "${COMMON_LIB_FILE}" 'require_verified_postgres_ssl_mode "POSTGRES_INTERNAL_SSL_MODE"'
assert_contains "${COMMON_LIB_FILE}" 'DB_SSL_MODE must be verify-full for production'
assert_contains "${PREFLIGHT_FILE}" 'require_production_postgres_ssl'
assert_contains "${PREFLIGHT_FILE}" 'require_public_ingress_config_preflight'
assert_contains "${PREFLIGHT_FILE}" 'require_public_identity_ingress_preflight'

ssl_gate_line="$(line_number "${PREFLIGHT_FILE}" 'require_production_postgres_ssl')"
public_ingress_config_preflight_line="$(line_number "${PREFLIGHT_FILE}" 'require_public_ingress_config_preflight')"
public_ingress_preflight_line="$(line_number "${PREFLIGHT_FILE}" 'require_public_identity_ingress_preflight')"
docker_info_line="$(line_number "${PREFLIGHT_FILE}" 'docker info >/dev/null')"
pg_isready_line="$(line_number "${PREFLIGHT_FILE}" 'pg_isready -d "${BACKUP_DATABASE_URL}" -t 5')"
public_dns_web_line="$(line_number "${COMMON_LIB_FILE}" 'require_public_dns_resolved "Web"')"
public_http_web_line="$(line_number "${COMMON_LIB_FILE}" 'require_public_http_reachable "Web"')"
if (( ssl_gate_line >= docker_info_line )); then
  fail "remote preflight must validate production PostgreSQL SSL before Docker checks"
fi
if (( public_ingress_preflight_line <= ssl_gate_line )); then
  fail "remote preflight must validate public identity ingress after PostgreSQL SSL config validation"
fi
if (( public_ingress_config_preflight_line <= ssl_gate_line )); then
  fail "remote preflight must validate local Nginx public ingress config after PostgreSQL SSL config validation"
fi
if (( public_ingress_preflight_line <= public_ingress_config_preflight_line )); then
  fail "remote preflight must validate public identity ingress after local Nginx config validation"
fi
if (( public_ingress_preflight_line >= docker_info_line )); then
  fail "remote preflight must validate public identity ingress before Docker checks"
fi
if (( ssl_gate_line >= pg_isready_line )); then
  fail "remote preflight must validate production PostgreSQL SSL before pg_isready"
fi
if (( public_dns_web_line >= public_http_web_line )); then
  fail "public DNS preflight must run before public HTTP/TLS reachability checks"
fi

tmpdir="$(mktemp -d)"
cleanup() {
  rm -rf "${tmpdir}"
}
trap cleanup EXIT

fake_bin="${tmpdir}/bin"
mkdir -p "${fake_bin}"
cat >"${fake_bin}/curl" <<'CURL'
#!/usr/bin/env bash
set -euo pipefail

output_file=""
url=""
args=("$@")
index=0
while [[ "${index}" -lt "${#args[@]}" ]]; do
  case "${args[${index}]}" in
    -o)
      index=$((index + 1))
      output_file="${args[${index}]}"
      ;;
    http://* | https://*)
      url="${args[${index}]}"
      ;;
  esac
  index=$((index + 1))
done

body='{"Status":0,"Answer":[]}'
case "${url}" in
  *name=missing.example.com*)
    body='{"Status":3}'
    ;;
  *name=private.example.com*type=A*)
    body='{"Status":0,"Answer":[{"type":1,"data":"10.0.0.8"}]}'
    ;;
  *type=A*)
    body='{"Status":0,"Answer":[{"type":1,"data":"93.184.216.34"}]}'
    ;;
esac

if [[ -n "${output_file}" ]]; then
  printf '%s\n' "${body}" >"${output_file}"
else
  printf '%s\n' "${body}"
fi
CURL
chmod +x "${fake_bin}/curl"

PATH="${fake_bin}:${PATH}" bash -c '
  set -euo pipefail
  source "$1"
  require_public_dns_resolved "Web" "https://example.com"
' bash "${COMMON_LIB_FILE}" >/dev/null

if PATH="${fake_bin}:${PATH}" bash -c '
  set -euo pipefail
  source "$1"
  require_public_dns_resolved "Identity" "https://missing.example.com"
' bash "${COMMON_LIB_FILE}" >"${tmpdir}/missing.out" 2>"${tmpdir}/missing.err"; then
  fail "public DNS preflight should fail when public resolver returns NXDOMAIN"
fi
grep -q 'NXDOMAIN from public resolver' "${tmpdir}/missing.err" || \
  fail "public DNS NXDOMAIN failure did not report the expected diagnostic"

if PATH="${fake_bin}:${PATH}" bash -c '
  set -euo pipefail
  source "$1"
  require_public_dns_resolved "Web" "https://private.example.com"
' bash "${COMMON_LIB_FILE}" >"${tmpdir}/private.out" 2>"${tmpdir}/private.err"; then
  fail "public DNS preflight should fail when public resolver returns private addresses"
fi
grep -q 'non-public A/AAAA records: 10.0.0.8' "${tmpdir}/private.err" || \
  fail "public DNS private-address failure did not report the expected diagnostic"

echo "[remote-preflight-contract] all assertions passed"
