#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
PREFLIGHT_FILE="${REPO_ROOT}/infra/ops/remote-preflight.sh"
COMMON_LIB_FILE="${REPO_ROOT}/infra/ops/lib/common.sh"
SYSTEMD_EXEC_VALIDATOR="${REPO_ROOT}/infra/ops/validate-systemd-unit-execution.py"

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

assert_contains "${PREFLIGHT_FILE}" 'compose run --rm --no-deps -T postgres-client'
assert_not_contains "${PREFLIGHT_FILE}" 'compose run --rm --no-deps -T postgres \\'
assert_contains "${PREFLIGHT_FILE}" 'die "备份数据库连通性检查失败'
assert_not_contains "${PREFLIGHT_FILE}" 'docker run --rm --network host "\$\{pg_image\}"'
assert_contains "${COMMON_LIB_FILE}" 'failed to read generated secret env from \$\{SECRET_BACKEND\}'
assert_not_contains "${COMMON_LIB_FILE}" 'secret_backend_read_to_stdout "\$\{GENERATED_ENV_SECRET_REF\}" 2>/dev/null'
assert_contains "${COMMON_LIB_FILE}" '\$\{GENERATED_ENV_SECRET_REF:-\}'
assert_contains "${COMMON_LIB_FILE}" 'source_env_file\(\)'
assert_contains "${COMMON_LIB_FILE}" 'shlex\.quote\(value\)'
assert_contains "${COMMON_LIB_FILE}" 'require_production_postgres_ssl\(\)'
assert_contains "${COMMON_LIB_FILE}" 'EXTERNAL_POSTGRES_ALLOW_PLAINTEXT'
assert_contains "${COMMON_LIB_FILE}" 'EXTERNAL_POSTGRES_ALLOW_PLAINTEXT is only allowed in prod-parity'
assert_contains "${COMMON_LIB_FILE}" 'EXTERNAL_DATASTORE_NETWORK'
assert_contains "${COMMON_LIB_FILE}" 'docker-compose\.external-datastore\.yml'
assert_contains "${COMMON_LIB_FILE}" 'require_public_identity_ingress_preflight\(\)'
assert_contains "${COMMON_LIB_FILE}" 'require_public_ingress_config_preflight\(\)'
assert_contains "${COMMON_LIB_FILE}" 'nginx-public-ingress-preflight\.sh'
assert_contains "${COMMON_LIB_FILE}" 'require_public_http_reachable\(\)'
assert_contains "${COMMON_LIB_FILE}" 'require_public_oidc_discovery\(\)'
assert_contains "${COMMON_LIB_FILE}" 'require_public_jwks\(\)'
assert_contains "${COMMON_LIB_FILE}" 'public_oidc_jwks_uri\(\)'
assert_contains "${COMMON_LIB_FILE}" 'require_public_dns_resolved\(\)'
assert_contains "${COMMON_LIB_FILE}" 'dns\.google/resolve'
assert_contains "${COMMON_LIB_FILE}" 'require_public_dns_resolved "Admission"'
assert_contains "${COMMON_LIB_FILE}" 'require_public_http_reachable "Admission"'
assert_contains "${COMMON_LIB_FILE}" 'ADMISSION_PUBLIC_BASE_URL'
assert_contains "${COMMON_LIB_FILE}" '/verify/__stuhelper_public_ingress_probe__'
assert_contains "${COMMON_LIB_FILE}" 'PUBLIC_INGRESS_CONFIG_PREFLIGHT_ENABLED'
assert_contains "${COMMON_LIB_FILE}" 'PUBLIC_INGRESS_PREFLIGHT_ENABLED'
assert_contains "${COMMON_LIB_FILE}" 'PUBLIC_INGRESS_CASDOOR_UPSTREAM_PREFLIGHT_ENABLED'
assert_contains "${COMMON_LIB_FILE}" 'PUBLIC_INGRESS_PUBLIC_DNS_ENABLED'
assert_contains "${COMMON_LIB_FILE}" 'public DNS preflight failed'
assert_contains "${COMMON_LIB_FILE}" 'public OIDC discovery ready'
assert_contains "${COMMON_LIB_FILE}" 'public JWKS ready'
assert_contains "${COMMON_LIB_FILE}" 'discovery did not expose jwks_uri'
assert_contains "${COMMON_LIB_FILE}" 'require_verified_postgres_ssl_mode "POSTGRES_INTERNAL_SSL_MODE"'
assert_contains "${COMMON_LIB_FILE}" 'DB_SSL_MODE must be verify-full for production'
assert_contains "${PREFLIGHT_FILE}" 'APP_ENV must be production for remote preflight'
assert_contains "${PREFLIGHT_FILE}" 'vault-runtime-token\.sh" check'
assert_contains "${PREFLIGHT_FILE}" 'stuhelper-vault-token-renewal\.timer'
assert_contains "${PREFLIGHT_FILE}" 'Vault runtime token renewal timer is not active'
assert_contains "${COMMON_LIB_FILE}" 'require_systemd_unit_exact_environment\(\)'
assert_contains "${COMMON_LIB_FILE}" 'require_systemd_unit_hardened_lifecycle\(\)'
assert_contains "${COMMON_LIB_FILE}" '"RemainAfterExit=no"'
assert_contains "${COMMON_LIB_FILE}" 'SuccessExitStatus'
assert_contains "${COMMON_LIB_FILE}" 'require_systemd_unit_hardened_execution\(\)'
assert_contains "${COMMON_LIB_FILE}" 'validate-systemd-unit-environment\.py'
assert_contains "${COMMON_LIB_FILE}" 'property=ExecStartEx'
assert_contains "${COMMON_LIB_FILE}" 'validate-systemd-unit-execution\.py'
assert_contains "${SYSTEMD_EXEC_VALIDATOR}" 'exec_fields\.get\("ignore_errors"\) == "no"'
assert_contains "${SYSTEMD_EXEC_VALIDATOR}" 'exec_ex_fields\.get\("flags"\) == ""'
assert_contains "${PREFLIGHT_FILE}" 'stuhelper-postgres-dump-backup\.service'
assert_contains "${PREFLIGHT_FILE}" 'stuhelper-postgres-basebackup\.service'
assert_contains "${PREFLIGHT_FILE}" 'stuhelper-postgres-backup-sync\.service'
assert_contains "${PREFLIGHT_FILE}" 'backup_service_common_environment=\('
assert_contains "${PREFLIGHT_FILE}" '"ENV_FILE=\$\{REPO_ROOT\}/\.env\.prod\.shared"'
assert_contains "${PREFLIGHT_FILE}" '"LOCAL_STATE_DIR=/var/lib/stuhelper"'
assert_contains "${PREFLIGHT_FILE}" 'BACKUP_STAGING_DIR="\$\{BACKUP_STAGING_DIR:-/var/lib/stuhelper/postgres/backup-staging\}"'
assert_contains "${PREFLIGHT_FILE}" 'expected_service_environment\+=\("BACKUP_STAGING_DIR=\$\{BACKUP_STAGING_DIR\}"\)'
[[ "$(grep -Fc '"${expected_service_environment[@]}"' "${PREFLIGHT_FILE}")" == "2" ]] || \
  fail "remote preflight must validate the same environment allowlist in systemd properties and ExecStart argv"
[[ "$(grep -c 'BACKUP_OBJECT_STORAGE_OFF_HOST_REQUIRED=true' "${PREFLIGHT_FILE}")" == "1" ]] || \
  fail "remote preflight must require the off-host marker for every backup service"
assert_contains "${PREFLIGHT_FILE}" 'require_systemd_unit_hardened_execution'
assert_contains "${PREFLIGHT_FILE}" 'require_systemd_unit_hardened_lifecycle'
assert_contains "${PREFLIGHT_FILE}" 'require_production_postgres_ssl'
assert_contains "${PREFLIGHT_FILE}" 'require_production_external_student_source_security'
assert_contains "${COMMON_LIB_FILE}" 'EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_MODE must be verify-full in production'
assert_contains "${PREFLIGHT_FILE}" 'require_public_ingress_config_preflight'
assert_contains "${PREFLIGHT_FILE}" 'require_public_identity_ingress_preflight'
assert_contains "${PREFLIGHT_FILE}" 'ADMISSION_PUBLIC_SMOKE_ENABLED'
assert_contains "${PREFLIGHT_FILE}" 'ADMISSION_PUBLIC_SMOKE_PREFLIGHT_RETRIES'
assert_contains "${PREFLIGHT_FILE}" 'admission-public-smoke\.sh'

app_env_gate_line="$(line_number "${PREFLIGHT_FILE}" 'APP_ENV must be production for remote preflight')"
ssl_gate_line="$(line_number "${PREFLIGHT_FILE}" 'require_production_postgres_ssl')"
public_ingress_config_preflight_line="$(line_number "${PREFLIGHT_FILE}" 'require_public_ingress_config_preflight')"
public_ingress_preflight_line="$(line_number "${PREFLIGHT_FILE}" 'require_public_identity_ingress_preflight')"
admission_public_smoke_line="$(line_number "${PREFLIGHT_FILE}" 'admission-public-smoke.sh')"
backup_staging_default_line="$(line_number "${PREFLIGHT_FILE}" 'BACKUP_STAGING_DIR="${BACKUP_STAGING_DIR:-/var/lib/stuhelper/postgres/backup-staging}"')"
backup_staging_use_line="$(line_number "${PREFLIGHT_FILE}" 'expected_service_environment+=("BACKUP_STAGING_DIR=${BACKUP_STAGING_DIR}")')"
docker_info_line="$(line_number "${PREFLIGHT_FILE}" 'docker info >/dev/null')"
pg_isready_line="$(line_number "${PREFLIGHT_FILE}" 'pg_isready -d "${BACKUP_DATABASE_URL}" -t 5')"
public_dns_web_line="$(line_number "${COMMON_LIB_FILE}" 'require_public_dns_resolved "Web"')"
public_dns_admission_line="$(line_number "${COMMON_LIB_FILE}" 'require_public_dns_resolved "Admission"')"
public_http_web_line="$(line_number "${COMMON_LIB_FILE}" 'require_public_http_reachable "Web"')"
public_http_admission_line="$(line_number "${COMMON_LIB_FILE}" 'require_public_http_reachable "Admission"')"
if (( ssl_gate_line >= docker_info_line )); then
  fail "remote preflight must validate production PostgreSQL SSL before Docker checks"
fi
if (( app_env_gate_line >= ssl_gate_line )); then
  fail "remote preflight must enforce APP_ENV=production before PostgreSQL SSL config validation"
fi
if (( public_ingress_preflight_line <= ssl_gate_line )); then
  fail "remote preflight must validate public SSO/admission ingress after PostgreSQL SSL config validation"
fi
if (( public_ingress_config_preflight_line <= ssl_gate_line )); then
  fail "remote preflight must validate local Nginx public ingress config after PostgreSQL SSL config validation"
fi
if (( public_ingress_preflight_line <= public_ingress_config_preflight_line )); then
  fail "remote preflight must validate public SSO/admission ingress after local Nginx config validation"
fi
if (( public_ingress_preflight_line >= docker_info_line )); then
  fail "remote preflight must validate public SSO/admission ingress before Docker checks"
fi
if (( backup_staging_default_line >= backup_staging_use_line )); then
  fail "remote preflight must define the installer-compatible backup staging default before comparing systemd environments"
fi
if (( admission_public_smoke_line <= public_ingress_preflight_line )); then
  fail "remote preflight admission public smoke must run after public SSO/admission ingress preflight"
fi
if (( admission_public_smoke_line >= docker_info_line )); then
  fail "remote preflight admission public smoke must run before Docker checks"
fi
if (( ssl_gate_line >= pg_isready_line )); then
  fail "remote preflight must validate production PostgreSQL SSL before pg_isready"
fi
if (( public_dns_web_line >= public_http_web_line )); then
  fail "public DNS preflight must run before public HTTP/TLS reachability checks"
fi
if (( public_dns_admission_line >= public_http_admission_line )); then
  fail "admission public DNS preflight must run before admission public HTTP/TLS reachability checks"
fi
if (( public_dns_admission_line >= public_http_web_line )); then
  fail "admission public DNS preflight must run before public HTTP/TLS reachability checks"
fi

tmpdir="$(mktemp -d)"
cleanup() {
  rm -rf "${tmpdir}"
}
trap cleanup EXIT

# Variables in the command string are intentionally expanded by the isolated child shell.
# shellcheck disable=SC2016
if ! env -i \
  PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin \
  LOCAL_STATE_DIR=/var/lib/stuhelper \
  /bin/bash --noprofile --norc -c '
    set -euo pipefail
    [[ -z "${HOME+x}" ]]
    source "$1"
    [[ "${LOCAL_STATE_DIR}" == "/var/lib/stuhelper" ]]
    [[ "${POSTGRES_WAL_RESTORE_DIR}" == "/var/lib/stuhelper/postgres/wal-restore" ]]
  ' bash "${COMMON_LIB_FILE}"; then
  fail "the isolated backup-service environment could not initialize common.sh without HOME"
fi

if ! bash -c '
  set -euo pipefail
  source "$1"
  systemctl() {
    case "$*" in
      *--property=UnsetEnvironment*) printf "%s\n" "LD_PRELOAD LD_LIBRARY_PATH LD_AUDIT GCONV_PATH LOCPATH" ;;
      *--property=EnvironmentFiles*|*--property=PassEnvironment*) printf "\n" ;;
      *) printf "%s\n" "ENV_FILE=/opt/stuhelper/.env.prod.shared BACKUP_OBJECT_STORAGE_OFF_HOST_REQUIRED=true" ;;
    esac
  }
  require_systemd_unit_exact_environment \
    stuhelper-postgres-backup-sync.service \
    ENV_FILE=/opt/stuhelper/.env.prod.shared \
    BACKUP_OBJECT_STORAGE_OFF_HOST_REQUIRED=true
' bash "${COMMON_LIB_FILE}"; then
  fail "the systemd environment validator rejected the exact protected environment"
fi

if bash -c '
  set -euo pipefail
  source "$1"
  systemctl() {
    case "$*" in
      *--property=UnsetEnvironment*) printf "%s\n" "LD_PRELOAD LD_LIBRARY_PATH LD_AUDIT GCONV_PATH LOCPATH" ;;
      *--property=EnvironmentFiles*|*--property=PassEnvironment*) printf "\n" ;;
      *) printf "%s\n" "ENV_FILE=/opt/stuhelper/.env.prod.shared BACKUP_OBJECT_STORAGE_OFF_HOST_REQUIRED=false" ;;
    esac
  }
  require_systemd_unit_exact_environment \
    stuhelper-postgres-backup-sync.service \
    ENV_FILE=/opt/stuhelper/.env.prod.shared \
    BACKUP_OBJECT_STORAGE_OFF_HOST_REQUIRED=true
' bash "${COMMON_LIB_FILE}" >"${tmpdir}/stale-backup-unit.out" 2>"${tmpdir}/stale-backup-unit.err"; then
  fail "the systemd environment validator accepted a stale backup unit"
fi
grep -q 'reinstall the production backup timers' "${tmpdir}/stale-backup-unit.err" || \
  fail "the stale backup unit failure did not report the remediation"

if bash -c '
  set -euo pipefail
  source "$1"
  systemctl() {
    case "$*" in
      *--property=UnsetEnvironment*) printf "%s\n" "LD_PRELOAD LD_LIBRARY_PATH LD_AUDIT GCONV_PATH LOCPATH" ;;
      *--property=EnvironmentFiles*|*--property=PassEnvironment*) printf "\n" ;;
      *) printf "%s\n" "ENV_FILE=/opt/stuhelper/.env.prod.shared BACKUP_OBJECT_STORAGE_OFF_HOST_REQUIRED=true PYTHONPATH=/tmp/payload LD_PRELOAD=/tmp/payload.so" ;;
    esac
  }
  require_systemd_unit_exact_environment \
    stuhelper-postgres-backup-sync.service \
    ENV_FILE=/opt/stuhelper/.env.prod.shared \
    BACKUP_OBJECT_STORAGE_OFF_HOST_REQUIRED=true
' bash "${COMMON_LIB_FILE}" >"${tmpdir}/backup-unit-injected-env.out" 2>"${tmpdir}/backup-unit-injected-env.err"; then
  fail "the systemd environment validator accepted injected child-runtime variables"
fi
grep -q 'exact protected environment' "${tmpdir}/backup-unit-injected-env.err" || \
  fail "the injected systemd environment failure did not report the exact-environment policy"

if bash -c '
  set -euo pipefail
  source "$1"
  systemctl() {
    case "$*" in
      *--property=EnvironmentFiles*) printf "%s\n" "/opt/stuhelper/.env.prod.shared (ignore_errors=no)" ;;
      *--property=UnsetEnvironment*) printf "%s\n" "LD_PRELOAD LD_LIBRARY_PATH LD_AUDIT GCONV_PATH LOCPATH" ;;
      *--property=PassEnvironment*) printf "\n" ;;
      *) printf "%s\n" "BACKUP_OBJECT_STORAGE_OFF_HOST_REQUIRED=true" ;;
    esac
  }
  require_systemd_unit_exact_environment \
    stuhelper-postgres-backup-sync.service \
    BACKUP_OBJECT_STORAGE_OFF_HOST_REQUIRED=true
' bash "${COMMON_LIB_FILE}" >"${tmpdir}/backup-unit-env-file.out" 2>"${tmpdir}/backup-unit-env-file.err"; then
  fail "the systemd environment validator accepted a backup unit with EnvironmentFile overrides"
fi
grep -q 'must not set EnvironmentFiles' "${tmpdir}/backup-unit-env-file.err" || \
  fail "the backup unit EnvironmentFile failure did not report the protected marker policy"

if bash -c '
  set -euo pipefail
  source "$1"
  systemctl() {
    case "$*" in
      *--property=UnsetEnvironment*) printf "%s\n" "LD_PRELOAD LD_LIBRARY_PATH LD_AUDIT GCONV_PATH" ;;
      *--property=EnvironmentFiles*|*--property=PassEnvironment*) printf "\n" ;;
      *) printf "%s\n" "BACKUP_OBJECT_STORAGE_OFF_HOST_REQUIRED=true" ;;
    esac
  }
  require_systemd_unit_exact_environment \
    stuhelper-postgres-backup-sync.service \
    BACKUP_OBJECT_STORAGE_OFF_HOST_REQUIRED=true
' bash "${COMMON_LIB_FILE}" >"${tmpdir}/backup-unit-unset-env.out" 2>"${tmpdir}/backup-unit-unset-env.err"; then
  fail "the systemd environment validator accepted an incomplete pre-exec unset list"
fi
grep -q 'pre-exec unset list' "${tmpdir}/backup-unit-unset-env.err" || \
  fail "the backup unit UnsetEnvironment failure did not report the protected boundary"

if ! bash -c '
  set -euo pipefail
  source "$1"
  systemctl() {
    case "$*" in
      *--property=Type*) printf "%s\n" oneshot ;;
      *--property=RemainAfterExit*) printf "%s\n" no ;;
      *--property=Restart*) printf "%s\n" no ;;
      *--property=ExecCondition*|*--property=ExecStartPre*|*--property=ExecStartPost*|*--property=SuccessExitStatus*) printf "\n" ;;
      *) return 90 ;;
    esac
  }
  require_systemd_unit_hardened_lifecycle stuhelper-postgres-backup-sync.service
' bash "${COMMON_LIB_FILE}"; then
  fail "the systemd lifecycle validator rejected the protected recurring oneshot service"
fi

if bash -c '
  set -euo pipefail
  source "$1"
  systemctl() {
    case "$*" in
      *--property=Type*) printf "%s\n" oneshot ;;
      *--property=RemainAfterExit*) printf "%s\n" yes ;;
      *--property=Restart*) printf "%s\n" no ;;
      *--property=ExecCondition*|*--property=ExecStartPre*|*--property=ExecStartPost*|*--property=SuccessExitStatus*) printf "\n" ;;
      *) return 90 ;;
    esac
  }
  require_systemd_unit_hardened_lifecycle stuhelper-postgres-backup-sync.service
' bash "${COMMON_LIB_FILE}" >"${tmpdir}/backup-unit-remain-after-exit.out" 2>"${tmpdir}/backup-unit-remain-after-exit.err"; then
  fail "the systemd lifecycle validator accepted RemainAfterExit=yes"
fi
grep -q 'RemainAfterExit=no' "${tmpdir}/backup-unit-remain-after-exit.err" || \
  fail "the RemainAfterExit failure did not report the timer-safe lifecycle requirement"

if bash -c '
  set -euo pipefail
  source "$1"
  systemctl() {
    case "$*" in
      *--property=Type*) printf "%s\n" oneshot ;;
      *--property=RemainAfterExit*) printf "%s\n" no ;;
      *--property=Restart*) printf "%s\n" no ;;
      *--property=ExecCondition*) printf "%s\n" "{ path=/bin/false ; argv[]=/bin/false ; ignore_errors=no ; }" ;;
      *--property=ExecStartPre*|*--property=ExecStartPost*|*--property=SuccessExitStatus*) printf "\n" ;;
      *) return 90 ;;
    esac
  }
  require_systemd_unit_hardened_lifecycle stuhelper-postgres-backup-sync.service
' bash "${COMMON_LIB_FILE}" >"${tmpdir}/backup-unit-exec-condition.out" 2>"${tmpdir}/backup-unit-exec-condition.err"; then
  fail "the systemd lifecycle validator accepted a backup-skipping ExecCondition"
fi
grep -q 'must not set ExecCondition' "${tmpdir}/backup-unit-exec-condition.err" || \
  fail "the ExecCondition failure did not report the lifecycle override"

if bash -c '
  set -euo pipefail
  source "$1"
  systemctl() {
    case "$*" in
      *--property=Type*) printf "%s\n" oneshot ;;
      *--property=RemainAfterExit*) printf "%s\n" no ;;
      *--property=Restart*) printf "%s\n" no ;;
      *--property=SuccessExitStatus*) printf "%s\n" 1 ;;
      *--property=ExecCondition*|*--property=ExecStartPre*|*--property=ExecStartPost*) printf "\n" ;;
      *) return 90 ;;
    esac
  }
  require_systemd_unit_hardened_lifecycle stuhelper-postgres-backup-sync.service
' bash "${COMMON_LIB_FILE}" >"${tmpdir}/backup-unit-success-status.out" 2>"${tmpdir}/backup-unit-success-status.err"; then
  fail "the systemd lifecycle validator accepted an extended success exit status"
fi
grep -q 'must not set SuccessExitStatus' "${tmpdir}/backup-unit-success-status.err" || \
  fail "the SuccessExitStatus failure did not report the failure-masking override"

protected_exec_argv='/usr/bin/env -i PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin ENV_FILE=/opt/stuhelper/.env.prod.shared BACKUP_OBJECT_STORAGE_OFF_HOST_REQUIRED=true /bin/bash --noprofile --norc ./infra/ops/sync-postgres-backups.sh'
valid_exec_start="{ path=/usr/bin/env ; argv[]=${protected_exec_argv} ; ignore_errors=no ; }"
valid_exec_start_ex="{ path=/usr/bin/env ; argv[]=${protected_exec_argv} ; flags= ; }"
if ! python3 "${SYSTEMD_EXEC_VALIDATOR}" \
  --expected-working-directory /opt/stuhelper \
  --expected-command "./infra/ops/sync-postgres-backups.sh" \
  --actual-working-directory /opt/stuhelper \
  --exec-start "${valid_exec_start}" \
  --exec-start-ex "${valid_exec_start_ex}" \
  --expected-environment ENV_FILE=/opt/stuhelper/.env.prod.shared \
  --expected-environment BACKUP_OBJECT_STORAGE_OFF_HOST_REQUIRED=true; then
  fail "the systemd execution validator rejected the protected non-login backup command"
fi

login_exec_start='{ path=/bin/bash ; argv[]=/bin/bash -lc cd /opt/stuhelper && ./infra/ops/sync-postgres-backups.sh ; ignore_errors=no ; }'
login_exec_start_ex='{ path=/bin/bash ; argv[]=/bin/bash -lc cd /opt/stuhelper && ./infra/ops/sync-postgres-backups.sh ; flags= ; }'
if python3 "${SYSTEMD_EXEC_VALIDATOR}" \
  --expected-working-directory /opt/stuhelper \
  --expected-command "./infra/ops/sync-postgres-backups.sh" \
  --actual-working-directory /opt/stuhelper \
  --exec-start "${login_exec_start}" \
  --exec-start-ex "${login_exec_start_ex}" \
  --expected-environment ENV_FILE=/opt/stuhelper/.env.prod.shared \
  --expected-environment BACKUP_OBJECT_STORAGE_OFF_HOST_REQUIRED=true; then
  fail "the systemd execution validator accepted a login shell that can override the backup gate"
fi

ignore_exec_start="{ path=/usr/bin/env ; argv[]=${protected_exec_argv} ; ignore_errors=yes ; }"
ignore_exec_start_ex="{ path=/usr/bin/env ; argv[]=${protected_exec_argv} ; flags=ignore-failure ; }"
if python3 "${SYSTEMD_EXEC_VALIDATOR}" \
  --expected-working-directory /opt/stuhelper \
  --expected-command "./infra/ops/sync-postgres-backups.sh" \
  --actual-working-directory /opt/stuhelper \
  --exec-start "${ignore_exec_start}" \
  --exec-start-ex "${ignore_exec_start_ex}" \
  --expected-environment ENV_FILE=/opt/stuhelper/.env.prod.shared \
  --expected-environment BACKUP_OBJECT_STORAGE_OFF_HOST_REQUIRED=true; then
  fail "the systemd execution validator accepted a failure-ignoring ExecStart"
fi

fake_bin="${tmpdir}/bin"
mkdir -p "${fake_bin}"
cat >"${fake_bin}/curl" <<'CURL'
#!/usr/bin/env bash
set -euo pipefail

output_file=""
write_out=""
url=""
args=("$@")
index=0
while [[ "${index}" -lt "${#args[@]}" ]]; do
  case "${args[${index}]}" in
    -o)
      index=$((index + 1))
      output_file="${args[${index}]}"
      ;;
    -w)
      index=$((index + 1))
      write_out="${args[${index}]}"
      ;;
    http://* | https://*)
      url="${args[${index}]}"
      ;;
  esac
  index=$((index + 1))
done

status=200
body='{"Status":0,"Answer":[]}'
case "${url}" in
  https://sso.example.com/.well-known/openid-configuration)
    body='{"issuer":"https://sso.example.com","authorization_endpoint":"https://sso.example.com/login/oauth/authorize","token_endpoint":"https://sso.example.com/api/login/oauth/access_token","jwks_uri":"https://sso.example.com/.well-known/jwks"}'
    ;;
  https://sso.example.com/.well-known/jwks)
    body='{"keys":[{"kid":"casdoor-test-key"}]}'
    ;;
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
if [[ -n "${write_out}" ]]; then
  printf '%s' "${status}"
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
  require_public_dns_resolved "SSO" "https://missing.example.com"
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

jwks_uri="$(
  PATH="${fake_bin}:${PATH}" bash -c '
    set -euo pipefail
    source "$1"
    public_oidc_jwks_uri "https://sso.example.com"
  ' bash "${COMMON_LIB_FILE}"
)"
[[ "${jwks_uri}" == "https://sso.example.com/.well-known/jwks" ]] || \
  fail "public_oidc_jwks_uri did not return the discovery JWKS URI"

PATH="${fake_bin}:${PATH}" bash -c '
  set -euo pipefail
  source "$1"
  require_public_jwks "Casdoor" "https://sso.example.com/.well-known/jwks"
' bash "${COMMON_LIB_FILE}" >"${tmpdir}/jwks.out"
grep -q 'Casdoor public JWKS ready' "${tmpdir}/jwks.out" || \
  fail "public JWKS preflight did not report success"

echo "[remote-preflight-contract] all assertions passed"
