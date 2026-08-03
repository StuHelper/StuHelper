#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
PREFLIGHT_FILE="${REPO_ROOT}/infra/ops/remote-preflight.sh"
COMMON_LIB_FILE="${REPO_ROOT}/infra/ops/lib/common.sh"
SYSTEMD_EXEC_VALIDATOR="${REPO_ROOT}/infra/ops/validate-systemd-unit-execution.py"
SYSTEMD_TIMER_VALIDATOR="${REPO_ROOT}/infra/ops/validate-systemd-timer.py"
SYSTEMD_CONDITION_VALIDATOR="${REPO_ROOT}/infra/ops/validate-systemd-unit-conditions.py"

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
assert_contains "${COMMON_LIB_FILE}" 'require_production_postgres_archiving\(\)'
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
assert_contains "${COMMON_LIB_FILE}" 'require_systemd_unit_exact_identity\(\)'
assert_contains "${COMMON_LIB_FILE}" 'require_systemd_unit_hardened_lifecycle\(\)'
assert_contains "${COMMON_LIB_FILE}" 'require_systemd_unit_without_filesystem_overrides\(\)'
assert_contains "${COMMON_LIB_FILE}" 'BindReadOnlyPaths'
assert_contains "${COMMON_LIB_FILE}" 'ReadOnlyPaths'
assert_contains "${COMMON_LIB_FILE}" 'ReadWritePaths'
assert_contains "${COMMON_LIB_FILE}" 'InaccessiblePaths'
assert_contains "${COMMON_LIB_FILE}" 'ExecPaths'
assert_contains "${COMMON_LIB_FILE}" 'NoExecPaths'
assert_contains "${COMMON_LIB_FILE}" 'TemporaryFileSystem'
assert_contains "${COMMON_LIB_FILE}" 'MountImages'
assert_contains "${COMMON_LIB_FILE}" 'ExtensionImages'
assert_contains "${COMMON_LIB_FILE}" 'require_systemd_unit_without_conditions\(\)'
assert_contains "${COMMON_LIB_FILE}" 'org\.freedesktop\.systemd1\.Unit Conditions'
assert_contains "${COMMON_LIB_FILE}" 'org\.freedesktop\.systemd1\.Unit Asserts'
assert_contains "${COMMON_LIB_FILE}" '"RemainAfterExit=no"'
assert_contains "${COMMON_LIB_FILE}" 'TimeoutStartUSec=\$\{expected_start_timeout\}'
assert_contains "${COMMON_LIB_FILE}" '"TimeoutStopUSec=2min"'
assert_contains "${COMMON_LIB_FILE}" '"KillMode=control-group"'
assert_contains "${COMMON_LIB_FILE}" '"SendSIGKILL=yes"'
assert_contains "${COMMON_LIB_FILE}" '"StartLimitIntervalUSec=0"'
assert_contains "${COMMON_LIB_FILE}" '"StartLimitBurst=5"'
assert_contains "${COMMON_LIB_FILE}" '"Result=success"'
assert_contains "${PREFLIGHT_FILE}" 'backup_service_start_timeouts='
assert_contains "${PREFLIGHT_FILE}" '"18h"'
assert_contains "${PREFLIGHT_FILE}" '"1d 2h"'
assert_contains "${PREFLIGHT_FILE}" '"12h"'
assert_contains "${COMMON_LIB_FILE}" 'SuccessExitStatus'
assert_contains "${COMMON_LIB_FILE}" 'ExecReload'
assert_contains "${COMMON_LIB_FILE}" 'ExecStopPost'
assert_contains "${COMMON_LIB_FILE}" 'require_systemd_unit_hardened_execution\(\)'
assert_contains "${COMMON_LIB_FILE}" 'require_systemd_timer_schedule\(\)'
assert_contains "${COMMON_LIB_FILE}" 'validate-systemd-timer\.py'
assert_contains "${COMMON_LIB_FILE}" 'AccuracyUSec'
assert_contains "${COMMON_LIB_FILE}" 'RandomizedDelayUSec'
assert_contains "${COMMON_LIB_FILE}" 'FixedRandomDelay'
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
assert_contains "${PREFLIGHT_FILE}" 'backup_service_user="\$\(id -un\)"'
assert_contains "${PREFLIGHT_FILE}" 'backup_service_group="\$\{BACKUP_SERVICE_GROUP:-\}"'
assert_contains "${PREFLIGHT_FILE}" 'backup_service_uid="\$\(id -u\)"'
assert_contains "${PREFLIGHT_FILE}" 'remote production preflight must run as the non-root deploy user'
assert_contains "${PREFLIGHT_FILE}" 'BACKUP_SERVICE_GROUP to name the configured non-root service group'
assert_contains "${PREFLIGHT_FILE}" 'configured backup service group does not exist'
assert_contains "${PREFLIGHT_FILE}" 'require_systemd_unit_exact_identity'
assert_contains "${PREFLIGHT_FILE}" 'require_systemd_unit_hardened_lifecycle'
assert_contains "${PREFLIGHT_FILE}" 'require_systemd_unit_without_filesystem_overrides'
assert_contains "${PREFLIGHT_FILE}" 'require_systemd_unit_without_conditions'
assert_contains "${PREFLIGHT_FILE}" 'require_systemd_timer_schedule'
assert_contains "${PREFLIGHT_FILE}" 'systemctl is-active --quiet'
assert_contains "${PREFLIGHT_FILE}" 'require_production_postgres_ssl'
assert_contains "${PREFLIGHT_FILE}" 'require_production_postgres_archiving'
assert_contains "${PREFLIGHT_FILE}" 'require_external_postgres_pitr_evidence'
assert_contains "${PREFLIGHT_FILE}" 'require_production_external_student_source_security'
assert_contains "${COMMON_LIB_FILE}" 'EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_MODE must be verify-full in production'
assert_contains "${PREFLIGHT_FILE}" 'require_public_ingress_config_preflight'
assert_contains "${PREFLIGHT_FILE}" 'require_public_identity_ingress_preflight'
assert_contains "${PREFLIGHT_FILE}" 'ADMISSION_PUBLIC_SMOKE_ENABLED'
assert_contains "${PREFLIGHT_FILE}" 'ADMISSION_PUBLIC_SMOKE_PREFLIGHT_RETRIES'
assert_contains "${PREFLIGHT_FILE}" 'admission-public-smoke\.sh'
assert_contains "${PREFLIGHT_FILE}" 'pre-deploy.*post-deploy.*full'
assert_contains "${COMMON_LIB_FILE}" 'configure_production_preflight_runtime_checks\(\)'
assert_contains "${PREFLIGHT_FILE}" 'configure_production_preflight_runtime_checks "\$\{preflight_phase\}"'
assert_contains "${COMMON_LIB_FILE}" 'run_database_runtime_checks=false'
assert_contains "${COMMON_LIB_FILE}" 'run_public_runtime_checks=false'
assert_contains "${PREFLIGHT_FILE}" 'mandatory post-deploy preflight'

app_env_gate_line="$(line_number "${PREFLIGHT_FILE}" 'APP_ENV must be production for remote preflight')"
ssl_gate_line="$(line_number "${PREFLIGHT_FILE}" 'require_production_postgres_ssl')"
archive_gate_line="$(line_number "${PREFLIGHT_FILE}" 'require_production_postgres_archiving')"
public_ingress_config_preflight_line="$(line_number "${PREFLIGHT_FILE}" 'require_public_ingress_config_preflight')"
public_ingress_preflight_line="$(line_number "${PREFLIGHT_FILE}" 'require_public_identity_ingress_preflight')"
admission_public_smoke_line="$(line_number "${PREFLIGHT_FILE}" 'admission-public-smoke.sh')"
backup_staging_default_line="$(line_number "${PREFLIGHT_FILE}" 'BACKUP_STAGING_DIR="${BACKUP_STAGING_DIR:-/var/lib/stuhelper/postgres/backup-staging}"')"
backup_staging_use_line="$(line_number "${PREFLIGHT_FILE}" 'expected_service_environment+=("BACKUP_STAGING_DIR=${BACKUP_STAGING_DIR}")')"
docker_info_line="$(line_number "${PREFLIGHT_FILE}" 'docker info >/dev/null')"
container_runtime_detection_line="$(line_number "${PREFLIGHT_FILE}" 'configure_production_preflight_runtime_checks "${preflight_phase}"')"
pg_isready_line="$(line_number "${PREFLIGHT_FILE}" 'pg_isready -d "${BACKUP_DATABASE_URL}" -t 5')"
external_pitr_evidence_line="$(line_number "${PREFLIGHT_FILE}" 'require_external_postgres_pitr_evidence')"
preflight_complete_line="$(line_number "${PREFLIGHT_FILE}" 'remote preflight checks passed')"
public_dns_web_line="$(line_number "${COMMON_LIB_FILE}" 'require_public_dns_resolved "Web"')"
public_dns_admission_line="$(line_number "${COMMON_LIB_FILE}" 'require_public_dns_resolved "Admission"')"
public_http_web_line="$(line_number "${COMMON_LIB_FILE}" 'require_public_http_reachable "Web"')"
public_http_admission_line="$(line_number "${COMMON_LIB_FILE}" 'require_public_http_reachable "Admission"')"
if (( app_env_gate_line >= docker_info_line || docker_info_line >= container_runtime_detection_line )); then
  fail "remote preflight must validate Docker before classifying live first-deploy prerequisites"
fi
if (( archive_gate_line <= ssl_gate_line )); then
  fail "remote preflight must validate internal WAL archiving after PostgreSQL TLS config"
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
if (( backup_staging_default_line >= backup_staging_use_line )); then
  fail "remote preflight must define the installer-compatible backup staging default before comparing systemd environments"
fi
if (( admission_public_smoke_line <= public_ingress_preflight_line )); then
  fail "remote preflight admission public smoke must run after public SSO/admission ingress preflight"
fi
if (( ssl_gate_line >= pg_isready_line )); then
  fail "remote preflight must validate production PostgreSQL SSL before pg_isready"
fi
if (( external_pitr_evidence_line <= pg_isready_line || external_pitr_evidence_line >= preflight_complete_line )); then
  fail "remote preflight must bind external PITR evidence to the reachable live cluster before passing"
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

source "${COMMON_LIB_FILE}"
runtime_postgres_running=false
docker() {
  [[ "${1:-}" == "inspect" ]] || return 90
  local target="${!#}"
  case "${target}" in
    contract-postgres) printf '%s\n' "${runtime_postgres_running}" ;;
    *) return 1 ;;
  esac
}
warn() { :; }
assert_runtime_check_state() {
  local phase="$1"
  local expected_database="$2"
  local expected_public="$3"
  configure_production_preflight_runtime_checks "${phase}"
  [[ "${run_database_runtime_checks}" == "${expected_database}" ]] ||
    fail "${phase} database runtime policy was ${run_database_runtime_checks}, expected ${expected_database}"
  [[ "${run_public_runtime_checks}" == "${expected_public}" ]] ||
    fail "${phase} public runtime policy was ${run_public_runtime_checks}, expected ${expected_public}"
}

STACK_NAME=contract
EXTERNAL_POSTGRES_ENABLED=false
assert_runtime_check_state --pre-deploy false false

runtime_postgres_running=true
assert_runtime_check_state --pre-deploy true false

runtime_postgres_running=false
EXTERNAL_POSTGRES_ENABLED=true
assert_runtime_check_state --pre-deploy true false

EXTERNAL_POSTGRES_ENABLED=false
assert_runtime_check_state --post-deploy true true
assert_runtime_check_state --full true true

unset -f docker warn assert_runtime_check_state
unset runtime_postgres_running STACK_NAME EXTERNAL_POSTGRES_ENABLED
unset run_database_runtime_checks run_public_runtime_checks
source "${COMMON_LIB_FILE}"

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

source "${COMMON_LIB_FILE}"
identity_user="stuhelper"
identity_group="stuhelper"
systemctl() {
  case "$*" in
    *--property=User*) printf '%s\n' "${identity_user}" ;;
    *--property=Group*) printf '%s\n' "${identity_group}" ;;
    *) return 90 ;;
  esac
}

if ! require_systemd_unit_exact_identity \
  stuhelper-postgres-backup-sync.service stuhelper stuhelper; then
  fail "the systemd identity validator rejected the exact non-root deploy identity"
fi

identity_user="root"
if (require_systemd_unit_exact_identity \
  stuhelper-postgres-backup-sync.service stuhelper stuhelper) \
  >"${tmpdir}/backup-unit-root-user.out" 2>"${tmpdir}/backup-unit-root-user.err"; then
  fail "the systemd identity validator accepted a root service override"
fi
grep -q 'must run as deploy user stuhelper' "${tmpdir}/backup-unit-root-user.err" ||
  fail "the root service identity failure did not report the expected deploy user"
identity_user="stuhelper"

identity_group="root"
if (require_systemd_unit_exact_identity \
  stuhelper-postgres-backup-sync.service stuhelper stuhelper) \
  >"${tmpdir}/backup-unit-root-group.out" 2>"${tmpdir}/backup-unit-root-group.err"; then
  fail "the systemd identity validator accepted a root group override"
fi
grep -q 'must run as deploy group stuhelper' "${tmpdir}/backup-unit-root-group.err" ||
  fail "the root group identity failure did not report the expected deploy group"
unset -f systemctl

source "${COMMON_LIB_FILE}"
lifecycle_type="oneshot"
lifecycle_remain_after_exit="no"
lifecycle_restart="no"
lifecycle_timeout_start="12h"
lifecycle_timeout_stop="2min"
lifecycle_kill_mode="control-group"
lifecycle_send_sigkill="yes"
lifecycle_start_limit_interval="0"
lifecycle_start_limit_burst="5"
lifecycle_result="success"
lifecycle_exec_condition=""
lifecycle_drop_in_paths=""
lifecycle_success_exit_status=""
lifecycle_exec_stop_post=""
systemctl() {
  case "$*" in
    *--property=Type*) printf '%s\n' "${lifecycle_type}" ;;
    *--property=RemainAfterExit*) printf '%s\n' "${lifecycle_remain_after_exit}" ;;
    *--property=Restart*) printf '%s\n' "${lifecycle_restart}" ;;
    *--property=TimeoutStartUSec*) printf '%s\n' "${lifecycle_timeout_start}" ;;
    *--property=TimeoutStopUSec*) printf '%s\n' "${lifecycle_timeout_stop}" ;;
    *--property=KillMode*) printf '%s\n' "${lifecycle_kill_mode}" ;;
    *--property=SendSIGKILL*) printf '%s\n' "${lifecycle_send_sigkill}" ;;
    *--property=StartLimitIntervalUSec*) printf '%s\n' "${lifecycle_start_limit_interval}" ;;
    *--property=StartLimitBurst*) printf '%s\n' "${lifecycle_start_limit_burst}" ;;
    *--property=Result*) printf '%s\n' "${lifecycle_result}" ;;
    *--property=DropInPaths*) printf '%s\n' "${lifecycle_drop_in_paths}" ;;
    *--property=ExecCondition*) printf '%s\n' "${lifecycle_exec_condition}" ;;
    *--property=SuccessExitStatus*) printf '%s\n' "${lifecycle_success_exit_status}" ;;
    *--property=ExecStopPost*) printf '%s\n' "${lifecycle_exec_stop_post}" ;;
    *--property=ExecReload*|*--property=ExecStartPre*|*--property=ExecStartPost*|*--property=ExecStop*) printf '\n' ;;
    *) return 90 ;;
  esac
}

if ! require_systemd_unit_hardened_lifecycle \
  stuhelper-postgres-backup-sync.service 12h; then
  fail "the systemd lifecycle validator rejected the protected recurring oneshot service"
fi

lifecycle_remain_after_exit="yes"
if (require_systemd_unit_hardened_lifecycle \
  stuhelper-postgres-backup-sync.service 12h) \
  >"${tmpdir}/backup-unit-remain-after-exit.out" 2>"${tmpdir}/backup-unit-remain-after-exit.err"; then
  fail "the systemd lifecycle validator accepted RemainAfterExit=yes"
fi
grep -q 'RemainAfterExit=no' "${tmpdir}/backup-unit-remain-after-exit.err" || \
  fail "the RemainAfterExit failure did not report the timer-safe lifecycle requirement"
lifecycle_remain_after_exit="no"

lifecycle_timeout_start="30s"
if (require_systemd_unit_hardened_lifecycle \
  stuhelper-postgres-backup-sync.service 12h) \
  >"${tmpdir}/backup-unit-timeout.out" 2>"${tmpdir}/backup-unit-timeout.err"; then
  fail "the systemd lifecycle validator accepted a non-canonical backup start timeout"
fi
grep -q 'TimeoutStartUSec=12h' "${tmpdir}/backup-unit-timeout.err" || \
  fail "the start-timeout failure did not report the service-specific finite deadline"
lifecycle_timeout_start="12h"

lifecycle_timeout_stop="infinity"
if (require_systemd_unit_hardened_lifecycle \
  stuhelper-postgres-backup-sync.service 12h) \
  >"${tmpdir}/backup-unit-stop-timeout.out" 2>"${tmpdir}/backup-unit-stop-timeout.err"; then
  fail "the systemd lifecycle validator accepted an unbounded stop timeout"
fi
grep -q 'TimeoutStopUSec=2min' "${tmpdir}/backup-unit-stop-timeout.err" ||
  fail "the stop-timeout failure did not report the enforced termination deadline"
lifecycle_timeout_stop="2min"

lifecycle_kill_mode="process"
if (require_systemd_unit_hardened_lifecycle \
  stuhelper-postgres-backup-sync.service 12h) \
  >"${tmpdir}/backup-unit-kill-mode.out" 2>"${tmpdir}/backup-unit-kill-mode.err"; then
  fail "the systemd lifecycle validator accepted process-only timeout termination"
fi
grep -q 'KillMode=control-group' "${tmpdir}/backup-unit-kill-mode.err" ||
  fail "the kill-mode failure did not report the complete-cgroup boundary"
lifecycle_kill_mode="control-group"

lifecycle_send_sigkill="no"
if (require_systemd_unit_hardened_lifecycle \
  stuhelper-postgres-backup-sync.service 12h) \
  >"${tmpdir}/backup-unit-sigkill.out" 2>"${tmpdir}/backup-unit-sigkill.err"; then
  fail "the systemd lifecycle validator accepted a non-enforceable finite deadline"
fi
grep -q 'SendSIGKILL=yes' "${tmpdir}/backup-unit-sigkill.err" ||
  fail "the SIGKILL failure did not report the enforceable-timeout boundary"
lifecycle_send_sigkill="yes"

lifecycle_start_limit_interval="1h"
if (require_systemd_unit_hardened_lifecycle \
  stuhelper-postgres-backup-sync.service 12h) \
  >"${tmpdir}/backup-unit-start-limit.out" 2>"${tmpdir}/backup-unit-start-limit.err"; then
  fail "the systemd lifecycle validator accepted a rate-limited recurring backup service"
fi
grep -q 'StartLimitIntervalUSec=0' "${tmpdir}/backup-unit-start-limit.err" ||
  fail "the start-limit failure did not report the disabled rate-limit requirement"
lifecycle_start_limit_interval="0"

lifecycle_result="start-limit-hit"
if (require_systemd_unit_hardened_lifecycle \
  stuhelper-postgres-backup-sync.service 12h) \
  >"${tmpdir}/backup-unit-rate-limited-result.out" 2>"${tmpdir}/backup-unit-rate-limited-result.err"; then
  fail "the systemd lifecycle validator accepted a currently rate-limited backup service"
fi
grep -q 'Result=success' "${tmpdir}/backup-unit-rate-limited-result.err" ||
  fail "the rate-limited result failure did not report the healthy-result requirement"
lifecycle_result="success"

lifecycle_exec_condition="{ path=/bin/false ; argv[]=/bin/false ; ignore_errors=no ; }"
if (require_systemd_unit_hardened_lifecycle \
  stuhelper-postgres-backup-sync.service 12h) \
  >"${tmpdir}/backup-unit-exec-condition.out" 2>"${tmpdir}/backup-unit-exec-condition.err"; then
  fail "the systemd lifecycle validator accepted a backup-skipping ExecCondition"
fi
grep -q 'must not set ExecCondition' "${tmpdir}/backup-unit-exec-condition.err" || \
  fail "the ExecCondition failure did not report the lifecycle override"
lifecycle_exec_condition=""

lifecycle_success_exit_status="1"
if (require_systemd_unit_hardened_lifecycle \
  stuhelper-postgres-backup-sync.service 12h) \
  >"${tmpdir}/backup-unit-success-status.out" 2>"${tmpdir}/backup-unit-success-status.err"; then
  fail "the systemd lifecycle validator accepted an extended success exit status"
fi
grep -q 'must not set SuccessExitStatus' "${tmpdir}/backup-unit-success-status.err" || \
  fail "the SuccessExitStatus failure did not report the failure-masking override"
lifecycle_success_exit_status=""

lifecycle_exec_stop_post="{ path=/bin/sh ; argv[]=/bin/sh -c true ; ignore_errors=no ; }"
if (require_systemd_unit_hardened_lifecycle \
  stuhelper-postgres-backup-sync.service 12h) \
  >"${tmpdir}/backup-unit-stop-post.out" 2>"${tmpdir}/backup-unit-stop-post.err"; then
  fail "the systemd lifecycle validator accepted an extra ExecStopPost hook"
fi
grep -q 'must not set ExecStopPost' "${tmpdir}/backup-unit-stop-post.err" || \
  fail "the ExecStopPost failure did not report the lifecycle override"
lifecycle_exec_stop_post=""

lifecycle_drop_in_paths="/etc/systemd/system/stuhelper-postgres-backup-sync.service.d/override.conf"
if (require_systemd_unit_hardened_lifecycle \
  stuhelper-postgres-backup-sync.service 12h) \
  >"${tmpdir}/backup-unit-drop-in.out" 2>"${tmpdir}/backup-unit-drop-in.err"; then
  fail "the systemd lifecycle validator accepted an ad-hoc service drop-in"
fi
grep -q 'must not set DropInPaths' "${tmpdir}/backup-unit-drop-in.err" || \
  fail "the drop-in failure did not report the immutable-unit boundary"
unset -f systemctl

source "${COMMON_LIB_FILE}"
namespace_root_directory=""
namespace_root_image=""
namespace_bind_paths=""
namespace_bind_read_only_paths=""
namespace_read_only_paths=""
namespace_read_write_paths=""
namespace_inaccessible_paths=""
namespace_exec_paths=""
namespace_no_exec_paths=""
namespace_temporary_file_system=""
namespace_mount_images=""
namespace_extension_images=""
namespace_extension_directories=""
namespace_root_ephemeral="no"
namespace_root_directory_start_only="no"
namespace_protect_system="no"
namespace_protect_home="no"
namespace_private_tmp="no"
namespace_private_mounts="no"
systemctl() {
  case "$*" in
    *--property=RootDirectoryStartOnly*) printf '%s\n' "${namespace_root_directory_start_only}" ;;
    *--property=RootDirectory*) printf '%s\n' "${namespace_root_directory}" ;;
    *--property=RootImage*) printf '%s\n' "${namespace_root_image}" ;;
    *--property=BindPaths*) printf '%s\n' "${namespace_bind_paths}" ;;
    *--property=BindReadOnlyPaths*) printf '%s\n' "${namespace_bind_read_only_paths}" ;;
    *--property=ReadOnlyPaths*) printf '%s\n' "${namespace_read_only_paths}" ;;
    *--property=ReadWritePaths*) printf '%s\n' "${namespace_read_write_paths}" ;;
    *--property=InaccessiblePaths*) printf '%s\n' "${namespace_inaccessible_paths}" ;;
    *--property=NoExecPaths*) printf '%s\n' "${namespace_no_exec_paths}" ;;
    *--property=ExecPaths*) printf '%s\n' "${namespace_exec_paths}" ;;
    *--property=TemporaryFileSystem*) printf '%s\n' "${namespace_temporary_file_system}" ;;
    *--property=MountImages*) printf '%s\n' "${namespace_mount_images}" ;;
    *--property=ExtensionImages*) printf '%s\n' "${namespace_extension_images}" ;;
    *--property=ExtensionDirectories*) printf '%s\n' "${namespace_extension_directories}" ;;
    *--property=RootEphemeral*) printf '%s\n' "${namespace_root_ephemeral}" ;;
    *--property=ProtectSystem*) printf '%s\n' "${namespace_protect_system}" ;;
    *--property=ProtectHome*) printf '%s\n' "${namespace_protect_home}" ;;
    *--property=PrivateTmp*) printf '%s\n' "${namespace_private_tmp}" ;;
    *--property=PrivateMounts*) printf '%s\n' "${namespace_private_mounts}" ;;
    *) return 90 ;;
  esac
}

if ! require_systemd_unit_without_filesystem_overrides \
  stuhelper-postgres-backup-sync.service; then
  fail "the systemd filesystem validator rejected an unmodified service namespace"
fi

namespace_bind_read_only_paths="/tmp/noop:/opt/stuhelper/infra/ops/sync-postgres-backups.sh"
if (require_systemd_unit_without_filesystem_overrides \
  stuhelper-postgres-backup-sync.service) \
  >"${tmpdir}/backup-unit-bind-path.out" 2>"${tmpdir}/backup-unit-bind-path.err"; then
  fail "the systemd filesystem validator accepted a protected-script bind replacement"
fi
grep -q 'must not set BindReadOnlyPaths' "${tmpdir}/backup-unit-bind-path.err" || \
  fail "the bind-path failure did not report the filesystem namespace override"
namespace_bind_read_only_paths=""

namespace_root_image="/var/lib/machines/replacement.raw"
if (require_systemd_unit_without_filesystem_overrides \
  stuhelper-postgres-backup-sync.service) \
  >"${tmpdir}/backup-unit-root-image.out" 2>"${tmpdir}/backup-unit-root-image.err"; then
  fail "the systemd filesystem validator accepted an alternate root image"
fi
grep -q 'must not set RootImage' "${tmpdir}/backup-unit-root-image.err" || \
  fail "the root-image failure did not report the filesystem namespace override"
namespace_root_image=""

namespace_temporary_file_system="/opt/stuhelper:ro"
if (require_systemd_unit_without_filesystem_overrides \
  stuhelper-postgres-backup-sync.service) \
  >"${tmpdir}/backup-unit-tmpfs.out" 2>"${tmpdir}/backup-unit-tmpfs.err"; then
  fail "the systemd filesystem validator accepted a temporary filesystem override"
fi
grep -q 'must not set TemporaryFileSystem' "${tmpdir}/backup-unit-tmpfs.err" || \
  fail "the temporary-filesystem failure did not report the namespace override"
namespace_temporary_file_system=""

namespace_read_only_paths="/opt/stuhelper"
if (require_systemd_unit_without_filesystem_overrides \
  stuhelper-postgres-backup-sync.service) \
  >"${tmpdir}/backup-unit-read-only.out" 2>"${tmpdir}/backup-unit-read-only.err"; then
  fail "the systemd filesystem validator accepted a read-only application tree"
fi
grep -q 'must not set ReadOnlyPaths' "${tmpdir}/backup-unit-read-only.err" || \
  fail "the read-only path failure did not report the namespace override"
namespace_read_only_paths=""

namespace_inaccessible_paths="/opt/stuhelper"
if (require_systemd_unit_without_filesystem_overrides \
  stuhelper-postgres-backup-sync.service) \
  >"${tmpdir}/backup-unit-inaccessible.out" 2>"${tmpdir}/backup-unit-inaccessible.err"; then
  fail "the systemd filesystem validator accepted an inaccessible application tree"
fi
grep -q 'must not set InaccessiblePaths' "${tmpdir}/backup-unit-inaccessible.err" || \
  fail "the inaccessible path failure did not report the namespace override"
namespace_inaccessible_paths=""

namespace_root_ephemeral="yes"
if (require_systemd_unit_without_filesystem_overrides \
  stuhelper-postgres-backup-sync.service) \
  >"${tmpdir}/backup-unit-root-ephemeral.out" 2>"${tmpdir}/backup-unit-root-ephemeral.err"; then
  fail "the systemd filesystem validator accepted an ephemeral root override"
fi
grep -q 'RootEphemeral=no' "${tmpdir}/backup-unit-root-ephemeral.err" || \
  fail "the ephemeral-root failure did not report the expected disabled value"
unset -f systemctl

empty_conditions='{"type":"a(sbbsi)","data":[]}'
if ! python3 "${SYSTEMD_CONDITION_VALIDATOR}" \
  --conditions-json "${empty_conditions}" \
  --asserts-json "${empty_conditions}"; then
  fail "the systemd condition validator rejected empty effective conditions"
fi
if python3 "${SYSTEMD_CONDITION_VALIDATOR}" \
  --conditions-json '{"type":"a(sbbsi)","data":[["ConditionPathExists",false,false,"/nonexistent",0]]}' \
  --asserts-json "${empty_conditions}"; then
  fail "the systemd condition validator accepted a backup-skipping condition"
fi
if ! bash -c '
  set -euo pipefail
  source "$1"
  busctl() {
    case "$*" in
      *Manager\ GetUnit*) printf "%s\n" "{\"type\":\"o\",\"data\":[\"/org/freedesktop/systemd1/unit/stuhelper_2dpostgres_2dbackup_2dsync_2eservice\"]}" ;;
      *Conditions|*Asserts) printf "%s\n" "{\"type\":\"a(sbbsi)\",\"data\":[]}" ;;
      *) return 90 ;;
    esac
  }
  require_systemd_unit_without_conditions stuhelper-postgres-backup-sync.service
' bash "${COMMON_LIB_FILE}"; then
  fail "the systemd condition gate rejected a service without conditions"
fi
if bash -c '
  set -euo pipefail
  source "$1"
  busctl() {
    case "$*" in
      *Manager\ GetUnit*) printf "%s\n" "{\"type\":\"o\",\"data\":[\"/org/freedesktop/systemd1/unit/stuhelper_2dpostgres_2dbackup_2dsync_2eservice\"]}" ;;
      *Conditions) printf "%s\n" "{\"type\":\"a(sbbsi)\",\"data\":[[\"ConditionPathExists\",false,false,\"/nonexistent\",0]]}" ;;
      *Asserts) printf "%s\n" "{\"type\":\"a(sbbsi)\",\"data\":[]}" ;;
      *) return 90 ;;
    esac
  }
  require_systemd_unit_without_conditions stuhelper-postgres-backup-sync.service
' bash "${COMMON_LIB_FILE}" >"${tmpdir}/backup-unit-condition.out" 2>"${tmpdir}/backup-unit-condition.err"; then
  fail "the systemd condition gate accepted an effective ConditionPathExists"
fi
grep -q 'must not define Conditions or Asserts' "${tmpdir}/backup-unit-condition.err" || \
  fail "the systemd condition failure did not report the skipped-backup boundary"

valid_calendar='{ OnCalendar=*-*-* *:00/15:00 ; next_elapse=Mon 2026-08-03 09:15:00 CST }'
if ! python3 "${SYSTEMD_TIMER_VALIDATOR}" \
  --target stuhelper-postgres-backup-sync.service \
  --persistent yes \
  --timers-calendar "${valid_calendar}" \
  --timers-monotonic '' \
  --accuracy 1min \
  --randomized-delay 0 \
  --fixed-random-delay no \
  --expected-target stuhelper-postgres-backup-sync.service \
  --expected-calendar '*-*-* *:00/15:00'; then
  fail "the systemd timer validator rejected the exact protected schedule"
fi
if python3 "${SYSTEMD_TIMER_VALIDATOR}" \
  --target legacy-backup.service \
  --persistent yes \
  --timers-calendar "${valid_calendar}" \
  --timers-monotonic '' \
  --accuracy 1min \
  --randomized-delay 0 \
  --fixed-random-delay no \
  --expected-target stuhelper-postgres-backup-sync.service \
  --expected-calendar '*-*-* *:00/15:00'; then
  fail "the systemd timer validator accepted a redirected target"
fi
if python3 "${SYSTEMD_TIMER_VALIDATOR}" \
  --target stuhelper-postgres-backup-sync.service \
  --persistent yes \
  --timers-calendar "${valid_calendar} { OnCalendar=*-*-* 00:00:00 ; next_elapse=Tue 2026-08-04 00:00:00 CST }" \
  --timers-monotonic '' \
  --accuracy 1min \
  --randomized-delay 0 \
  --fixed-random-delay no \
  --expected-target stuhelper-postgres-backup-sync.service \
  --expected-calendar '*-*-* *:00/15:00'; then
  fail "the systemd timer validator accepted an extra calendar"
fi
if python3 "${SYSTEMD_TIMER_VALIDATOR}" \
  --target stuhelper-postgres-backup-sync.service \
  --persistent yes \
  --timers-calendar "${valid_calendar}" \
  --timers-monotonic '{ OnBootSec=1min ; next_elapse=1min }' \
  --accuracy 1min \
  --randomized-delay 0 \
  --fixed-random-delay no \
  --expected-target stuhelper-postgres-backup-sync.service \
  --expected-calendar '*-*-* *:00/15:00'; then
  fail "the systemd timer validator accepted a monotonic trigger bypass"
fi
if python3 "${SYSTEMD_TIMER_VALIDATOR}" \
  --target stuhelper-postgres-backup-sync.service \
  --persistent yes \
  --timers-calendar "${valid_calendar}" \
  --timers-monotonic '' \
  --accuracy 30min \
  --randomized-delay 0 \
  --fixed-random-delay no \
  --expected-target stuhelper-postgres-backup-sync.service \
  --expected-calendar '*-*-* *:00/15:00'; then
  fail "the systemd timer validator accepted an excessive accuracy window"
fi
if python3 "${SYSTEMD_TIMER_VALIDATOR}" \
  --target stuhelper-postgres-backup-sync.service \
  --persistent yes \
  --timers-calendar "${valid_calendar}" \
  --timers-monotonic '' \
  --accuracy 1min \
  --randomized-delay 30min \
  --fixed-random-delay yes \
  --expected-target stuhelper-postgres-backup-sync.service \
  --expected-calendar '*-*-* *:00/15:00'; then
  fail "the systemd timer validator accepted a randomized freshness delay"
fi

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
