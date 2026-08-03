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

require_systemd_unit_exact_environment() {
  local unit="$1"
  shift
  (($# > 0)) || die "at least one expected environment assignment is required for systemd unit ${unit}"
  local effective_environment
  local effective_unset_environment
  local protected_property
  local protected_property_value
  local -a validator_args

  for protected_property in EnvironmentFiles PassEnvironment; do
    if ! protected_property_value="$(systemctl show "${unit}" --property="${protected_property}" --value 2>/dev/null)"; then
      die "failed to inspect ${protected_property} for systemd unit ${unit}"
    fi
    if [[ -n "${protected_property_value//[[:space:]]/}" ]]; then
      die "systemd unit ${unit} must not set ${protected_property} for the protected backup gate; reinstall the production backup timers and remove overriding drop-ins"
    fi
  done

  if ! effective_environment="$(systemctl show "${unit}" --property=Environment --value 2>/dev/null)"; then
    die "failed to inspect effective environment for systemd unit ${unit}"
  fi
  if ! effective_unset_environment="$(systemctl show "${unit}" --property=UnsetEnvironment --value 2>/dev/null)"; then
    die "failed to inspect UnsetEnvironment for systemd unit ${unit}"
  fi

  validator_args=(
    --environment "${effective_environment}"
    --unset-environment "${effective_unset_environment}"
  )
  for protected_property_value in "$@"; do
    validator_args+=(--expected-environment "${protected_property_value}")
  done
  if ! python3 "${COMMON_LIB_DIR}/../validate-systemd-unit-environment.py" "${validator_args[@]}"
  then
    die "systemd unit ${unit} must use the exact protected environment and pre-exec unset list; reinstall the production backup timers and remove overriding drop-ins"
  fi
}

require_no_legacy_backup_timer_units() {
  local listing unit _state
  local -a legacy_units=()
  listing="$(systemctl list-unit-files --type=timer --all --no-legend --no-pager)" ||
    die "failed to enumerate installed systemd timer units"
  while read -r unit _state _; do
    [[ -n "${unit}" ]] || continue
    case "${unit}" in
      stuhelper-postgres-dump-backup.timer | \
      stuhelper-postgres-basebackup.timer | \
      stuhelper-postgres-backup-sync.timer)
        ;;
      *-postgres-dump-backup.timer | \
      *-postgres-basebackup.timer | \
      *-postgres-backup-sync.timer)
        legacy_units+=("${unit}")
        ;;
    esac
  done <<<"${listing}"
  ((${#legacy_units[@]} == 0)) ||
    die "legacy prefixed PostgreSQL backup timer units must be disabled and removed before canonical activation: ${legacy_units[*]}"
}

require_systemd_unit_exact_identity() {
  local unit="$1"
  local expected_user="$2"
  local expected_group="$3"
  local effective_user
  local effective_group

  [[ -n "${expected_user}" && "${expected_user}" != "root" && "${expected_user}" != "0" ]] ||
    die "systemd unit ${unit} requires an explicit non-root deploy user"
  [[ -n "${expected_group}" && "${expected_group}" != "root" && "${expected_group}" != "0" ]] ||
    die "systemd unit ${unit} requires an explicit non-root deploy group"

  if ! effective_user="$(systemctl show "${unit}" --property=User --value 2>/dev/null)"; then
    die "failed to inspect User for systemd unit ${unit}"
  fi
  if ! effective_group="$(systemctl show "${unit}" --property=Group --value 2>/dev/null)"; then
    die "failed to inspect Group for systemd unit ${unit}"
  fi

  [[ "${effective_user}" == "${expected_user}" ]] ||
    die "systemd unit ${unit} must run as deploy user ${expected_user}, got ${effective_user:-<implicit-root>}; reinstall the production backup timers and remove overriding drop-ins"
  [[ "${effective_group}" == "${expected_group}" ]] ||
    die "systemd unit ${unit} must run as deploy group ${expected_group}, got ${effective_group:-<unset>}; reinstall the production backup timers and remove overriding drop-ins"
}

require_systemd_unit_hardened_lifecycle() {
  local unit="$1"
  local expected_start_timeout="$2"
  local actual_value
  local expected_value
  local property
  local property_spec
  local -a exact_properties=(
    "Type=oneshot"
    "RemainAfterExit=no"
    "Restart=no"
    "TimeoutStartUSec=${expected_start_timeout}"
    "TimeoutStopUSec=2min"
    "KillMode=control-group"
    "SendSIGKILL=yes"
    "StartLimitIntervalUSec=0"
    "StartLimitBurst=5"
    "Result=success"
  )
  local -a empty_properties=(
    DropInPaths
    ExecCondition
    ExecReload
    ExecStartPre
    ExecStartPost
    ExecStop
    ExecStopPost
    SuccessExitStatus
  )

  for property_spec in "${exact_properties[@]}"; do
    property="${property_spec%%=*}"
    expected_value="${property_spec#*=}"
    if ! actual_value="$(systemctl show "${unit}" --property="${property}" --value 2>/dev/null)"; then
      die "failed to inspect ${property} for systemd unit ${unit}"
    fi
    if [[ "${actual_value}" != "${expected_value}" ]]; then
      die "systemd unit ${unit} must set ${property}=${expected_value}; reinstall the production backup timers and remove overriding drop-ins"
    fi
  done

  for property in "${empty_properties[@]}"; do
    if ! actual_value="$(systemctl show "${unit}" --property="${property}" --value 2>/dev/null)"; then
      die "failed to inspect ${property} for systemd unit ${unit}"
    fi
    if [[ -n "${actual_value//[[:space:]]/}" ]]; then
      die "systemd unit ${unit} must not set ${property}; reinstall the production backup timers and remove overriding drop-ins"
    fi
  done
}

require_systemd_unit_without_filesystem_overrides() {
  local unit="$1"
  local actual_value
  local expected_value
  local property
  local property_spec
  local -a empty_properties=(
    RootDirectory
    RootImage
    BindPaths
    BindReadOnlyPaths
    ReadOnlyPaths
    ReadWritePaths
    InaccessiblePaths
    ExecPaths
    NoExecPaths
    TemporaryFileSystem
    MountImages
    ExtensionImages
    ExtensionDirectories
  )
  local -a exact_properties=(
    "RootEphemeral=no"
    "RootDirectoryStartOnly=no"
    "ProtectSystem=no"
    "ProtectHome=no"
    "PrivateTmp=no"
    "PrivateMounts=no"
  )

  for property in "${empty_properties[@]}"; do
    if ! actual_value="$(systemctl show "${unit}" --property="${property}" --value 2>/dev/null)"; then
      die "failed to inspect ${property} for systemd unit ${unit}"
    fi
    if [[ -n "${actual_value//[[:space:]]/}" ]]; then
      die "systemd unit ${unit} must not set ${property}; filesystem namespace overrides can replace protected backup code"
    fi
  done

  for property_spec in "${exact_properties[@]}"; do
    property="${property_spec%%=*}"
    expected_value="${property_spec#*=}"
    if ! actual_value="$(systemctl show "${unit}" --property="${property}" --value 2>/dev/null)"; then
      die "failed to inspect ${property} for systemd unit ${unit}"
    fi
    if [[ "${actual_value}" != "${expected_value}" ]]; then
      die "systemd unit ${unit} must set ${property}=${expected_value}; filesystem namespace overrides can replace protected backup code"
    fi
  done
}

require_systemd_unit_without_conditions() {
  local unit="$1"
  local asserts_json
  local conditions_json
  local unit_path
  local unit_path_json

  require_cmd busctl
  if ! unit_path_json="$(busctl --json=short call \
    org.freedesktop.systemd1 \
    /org/freedesktop/systemd1 \
    org.freedesktop.systemd1.Manager \
    GetUnit s "${unit}" 2>/dev/null)"; then
    die "failed to resolve the systemd D-Bus object for ${unit}"
  fi
  if ! unit_path="$(python3 - "${unit_path_json}" <<'PY'
import json
import sys

try:
    document = json.loads(sys.argv[1])
    values = document["data"]
except (json.JSONDecodeError, KeyError, TypeError):
    raise SystemExit(1)
if document.get("type") != "o" or len(values) != 1:
    raise SystemExit(1)
path = values[0]
if not isinstance(path, str) or not path.startswith("/org/freedesktop/systemd1/unit/"):
    raise SystemExit(1)
print(path)
PY
)"; then
    die "systemd returned an invalid D-Bus object for ${unit}"
  fi
  if ! conditions_json="$(busctl --json=short get-property \
    org.freedesktop.systemd1 "${unit_path}" \
    org.freedesktop.systemd1.Unit Conditions 2>/dev/null)"; then
    die "failed to inspect effective Conditions for systemd unit ${unit}"
  fi
  if ! asserts_json="$(busctl --json=short get-property \
    org.freedesktop.systemd1 "${unit_path}" \
    org.freedesktop.systemd1.Unit Asserts 2>/dev/null)"; then
    die "failed to inspect effective Asserts for systemd unit ${unit}"
  fi
  if ! python3 "${COMMON_LIB_DIR}/../validate-systemd-unit-conditions.py" \
    --conditions-json "${conditions_json}" \
    --asserts-json "${asserts_json}"
  then
    die "systemd unit ${unit} must not define Conditions or Asserts that can skip protected backups; reinstall the production backup timers and remove overriding drop-ins"
  fi
}

require_systemd_timer_schedule() {
  local timer_unit="$1"
  local expected_target="$2"
  local expected_calendar="$3"
  local effective_accuracy
  local effective_calendar
  local effective_fixed_random_delay
  local effective_monotonic
  local effective_persistent
  local effective_randomized_delay
  local effective_target

  if ! effective_target="$(systemctl show "${timer_unit}" --property=Unit --value 2>/dev/null)"; then
    die "failed to inspect Unit for systemd timer ${timer_unit}"
  fi
  if ! effective_persistent="$(systemctl show "${timer_unit}" --property=Persistent --value 2>/dev/null)"; then
    die "failed to inspect Persistent for systemd timer ${timer_unit}"
  fi
  if ! effective_calendar="$(systemctl show "${timer_unit}" --property=TimersCalendar --value 2>/dev/null)"; then
    die "failed to inspect TimersCalendar for systemd timer ${timer_unit}"
  fi
  if ! effective_monotonic="$(systemctl show "${timer_unit}" --property=TimersMonotonic --value 2>/dev/null)"; then
    die "failed to inspect TimersMonotonic for systemd timer ${timer_unit}"
  fi
  if ! effective_accuracy="$(systemctl show "${timer_unit}" --property=AccuracyUSec --value 2>/dev/null)"; then
    die "failed to inspect AccuracyUSec for systemd timer ${timer_unit}"
  fi
  if ! effective_randomized_delay="$(systemctl show "${timer_unit}" --property=RandomizedDelayUSec --value 2>/dev/null)"; then
    die "failed to inspect RandomizedDelayUSec for systemd timer ${timer_unit}"
  fi
  if ! effective_fixed_random_delay="$(systemctl show "${timer_unit}" --property=FixedRandomDelay --value 2>/dev/null)"; then
    die "failed to inspect FixedRandomDelay for systemd timer ${timer_unit}"
  fi

  if ! python3 "${COMMON_LIB_DIR}/../validate-systemd-timer.py" \
    --target "${effective_target}" \
    --persistent "${effective_persistent}" \
    --timers-calendar "${effective_calendar}" \
    --timers-monotonic "${effective_monotonic}" \
    --accuracy "${effective_accuracy}" \
    --randomized-delay "${effective_randomized_delay}" \
    --fixed-random-delay "${effective_fixed_random_delay}" \
    --expected-target "${expected_target}" \
    --expected-calendar "${expected_calendar}"
  then
    die "systemd timer ${timer_unit} must target ${expected_target} with the exact protected calendar ${expected_calendar}, one-minute accuracy, and no randomized delay; reinstall the production backup timers and remove overriding drop-ins"
  fi
}

require_systemd_unit_hardened_execution() {
  local unit="$1"
  local expected_working_directory="$2"
  local expected_command="$3"
  shift 3
  local effective_exec_records
  local effective_exec_start
  local effective_exec_start_ex
  local effective_working_directory
  local expected_environment
  local -a validator_args

  if ! effective_working_directory="$(systemctl show "${unit}" --property=WorkingDirectory --value 2>/dev/null)"; then
    die "failed to inspect WorkingDirectory for systemd unit ${unit}"
  fi
  if ! effective_exec_records="$(systemctl show "${unit}" --property=ExecStart --property=ExecStartEx --value 2>/dev/null)"; then
    die "failed to inspect ExecStart/ExecStartEx for systemd unit ${unit}"
  fi
  effective_exec_start="${effective_exec_records%%$'\n'*}"
  effective_exec_start_ex="${effective_exec_records#*$'\n'}"
  [[ "${effective_exec_start}" != "${effective_exec_start_ex}" ]] ||
    die "systemd unit ${unit} did not expose distinct ExecStart and ExecStartEx records"

  validator_args=(
    --expected-working-directory "${expected_working_directory}"
    --expected-command "${expected_command}"
    --actual-working-directory "${effective_working_directory}"
    --exec-start "${effective_exec_start}"
    --exec-start-ex "${effective_exec_start_ex}"
  )
  for expected_environment in "$@"; do
    validator_args+=(--expected-environment "${expected_environment}")
  done
  if ! python3 "${COMMON_LIB_DIR}/../validate-systemd-unit-execution.py" "${validator_args[@]}"
  then
    die "systemd unit ${unit} must use the protected non-login Bash execution path in ${expected_working_directory}; reinstall the production backup timers"
  fi
}

require_integer_range() {
  local key="$1"
  local value="$2"
  local minimum="$3"
  local maximum="$4"

  if [[ ! "${value}" =~ ^[0-9]+$ ]] ||
    ((10#${value} < minimum || 10#${value} > maximum)); then
    die "${key} must be an integer between ${minimum} and ${maximum}"
  fi
}

is_process_control_environment_key() {
  local key="$1"

  case "${key}" in
    PATH | IFS | CDPATH | BASH_ENV | ENV | BASHOPTS | SHELLOPTS | GLOBIGNORE | \
      GCONV_PATH | LOCPATH | NLSPATH | \
      NODE_OPTIONS | NODE_PATH | \
      PERL5LIB | PERLLIB | PERL5OPT | \
      RUBYLIB | RUBYOPT | \
      JAVA_TOOL_OPTIONS | _JAVA_OPTIONS | JDK_JAVA_OPTIONS | \
      GIT_EXEC_PATH | GIT_SSH | GIT_SSH_COMMAND | \
      SSH_ASKPASS | SSH_ASKPASS_REQUIRE | \
      DOCKER_HOST | DOCKER_CONTEXT | DOCKER_CERT_PATH | DOCKER_TLS | DOCKER_TLS_VERIFY | \
      LD_* | DYLD_* | PYTHON*)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

clear_process_control_environment() {
  local key

  # PATH is a caller-owned control-plane input and production systemd units pin
  # it explicitly. Environment files still cannot replace it. BASHOPTS and
  # SHELLOPTS are readonly Bash state, so they are rejected in files but cannot
  # be unset after the shell has started.
  while IFS= read -r key; do
    case "${key}" in
      PATH | BASHOPTS | SHELLOPTS)
        continue
        ;;
    esac
    if is_process_control_environment_key "${key}"; then
      unset "${key}"
    fi
  done < <(compgen -A variable)
  unset IFS
}

source_env_file() {
  local file="$1"
  shift
  # Clear inherited interpreter and dynamic-loader hooks before starting the
  # Python parser. The validated file is then prevented from reintroducing any
  # process-control variable before credential-bearing child processes run.
  clear_process_control_environment
  [[ -f "${file}" ]] || return 0

  local rendered
  if ! rendered="$(python3 - "${file}" "$@" 2>&1 <<'PY'
import re
import shlex
import sys
from pathlib import Path

path = Path(sys.argv[1])
allowed_keys = set(sys.argv[2:])
key_pattern = re.compile(r"[A-Za-z_][A-Za-z0-9_]*")
forbidden_exact_keys = {
    "PATH", "IFS", "CDPATH", "BASH_ENV", "ENV", "BASHOPTS", "SHELLOPTS",
    "GLOBIGNORE", "GCONV_PATH", "LOCPATH", "NLSPATH", "NODE_OPTIONS",
    "NODE_PATH", "PERL5LIB", "PERLLIB", "PERL5OPT", "RUBYLIB", "RUBYOPT",
    "JAVA_TOOL_OPTIONS", "_JAVA_OPTIONS", "JDK_JAVA_OPTIONS", "GIT_EXEC_PATH",
    "GIT_SSH", "GIT_SSH_COMMAND", "SSH_ASKPASS", "SSH_ASKPASS_REQUIRE",
    "DOCKER_HOST", "DOCKER_CONTEXT", "DOCKER_CERT_PATH", "DOCKER_TLS",
    "DOCKER_TLS_VERIFY", "PRODUCTION_DEPLOY_LOCK_FD",
}
forbidden_prefixes = ("LD_", "DYLD_", "PYTHON")
rendered_assignments = []

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
    if key in forbidden_exact_keys or key.startswith(forbidden_prefixes):
        raise SystemExit(
            f"{path}:{lineno}: process-control variable {key} is not allowed in StuHelper environment files"
        )
    if allowed_keys and key not in allowed_keys:
        raise SystemExit(
            f"{path}:{lineno}: environment key {key} is not allowed in this file"
        )

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

    rendered_assignments.append(f"export {key}={shlex.quote(value)}")

print("\n".join(rendered_assignments))
PY
)"; then
    die "${rendered}"
  fi

  # shellcheck disable=SC1091
  source /dev/stdin <<<"${rendered}"
  clear_process_control_environment
}

source_casdoor_bootstrap_env_file() {
  source_env_file \
    "$1" \
    CASDOOR_BOOTSTRAP_CLIENT_ID \
    CASDOOR_BOOTSTRAP_CLIENT_SECRET \
    CASDOOR_BOOTSTRAP_APPLICATION \
    CASDOOR_BOOTSTRAP_CERTIFICATE \
    CASDOOR_BOOTSTRAP_ORGANIZATION
}

source_remote_deploy_config_env_file() {
  source_env_file \
    "$1" \
    REGISTRY \
    REGISTRY_AUTH_MODE \
    REGISTRY_USERNAME_SECRET_REF \
    REGISTRY_PASSWORD_SECRET_REF \
    BACKUP_SERVICE_GROUP \
    ENV_FILE \
    SECRETS_ENV_FILE \
    GENERATED_ENV_FILE \
    GENERATED_SECRET_ENV_FILE \
    SECRET_BACKEND \
    SHARED_ENV_SECRET_REF \
    SECRETS_ENV_SECRET_REF \
    GENERATED_ENV_SECRET_REF \
    SECRET_FILE_ROOT \
    VAULT_ADDR \
    VAULT_NAMESPACE \
    VAULT_TOKEN_FILE \
    VAULT_KV_MOUNT \
    VAULT_RUNTIME_TOKEN_POLICY \
    VAULT_RUNTIME_TOKEN_PERIOD_SECONDS \
    VAULT_RUNTIME_TOKEN_MIN_TTL_SECONDS
}

require_protected_backup_environment_paths() {
  local index key actual expected
  local -a keys=(
    ENV_FILE
    SECRETS_ENV_FILE
    GENERATED_ENV_FILE
    GENERATED_SECRET_ENV_FILE
  )
  local -a expected_paths=(
    "${REPO_ROOT}/.env.prod.shared"
    "${REPO_ROOT}/.env.prod.secrets"
    "${REPO_ROOT}/.env.prod.generated"
    "${REPO_ROOT}/.env.prod.generated.secrets"
  )
  for index in "${!keys[@]}"; do
    key="${keys[${index}]}"
    actual="${!key:-}"
    expected="${expected_paths[${index}]}"
    [[ "${actual}" == "${expected}" ]] ||
      die "${key} must be exactly ${expected} for the protected production backup services"
  done
}

require_digest_image_ref() {
  local key="$1"
  local value="${2:-}"

  if [[ ! "${value}" =~ ^[^[:space:]@]+@sha256:[0-9a-f]{64}$ ]]; then
    die "${key} must be a complete image@sha256 digest reference"
  fi
}

require_safe_release_tag() {
  local tag="${1:-}"

  if [[ ! "${tag}" =~ ^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$ ]]; then
    die "release tag must be 1-128 characters and contain only ASCII letters, digits, dot, underscore, or hyphen, starting with a letter or digit"
  fi
}

acquire_production_deploy_lock() {
  local state_mode path_identity fd_identity
  require_cmd flock
  mkdir -p -m 0700 "${DEPLOY_STATE_DIR}"
  [[ -d "${DEPLOY_STATE_DIR}" && ! -L "${DEPLOY_STATE_DIR}" ]] ||
    die "deployment state path must be a regular non-symlink directory: ${DEPLOY_STATE_DIR}"
  [[ -O "${DEPLOY_STATE_DIR}" ]] ||
    die "deployment state directory must be owned by the deploy user: ${DEPLOY_STATE_DIR}"
  state_mode="$(stat -c '%a' "${DEPLOY_STATE_DIR}")"
  [[ "${state_mode}" =~ ^[0-7]{3,4}$ ]] ||
    die "unable to validate deployment state directory mode: ${DEPLOY_STATE_DIR}"
  (((8#${state_mode} & 8#022) == 0)) ||
    die "deployment state directory must not be group- or world-writable: ${DEPLOY_STATE_DIR}"

  if [[ -n "${PRODUCTION_DEPLOY_LOCK_FD:-}" ]]; then
    [[ "${PRODUCTION_DEPLOY_LOCK_FD}" =~ ^[0-9]+$ ]] ||
      die "inherited production deployment lock descriptor is invalid"
    [[ -e "/proc/${BASHPID}/fd/${PRODUCTION_DEPLOY_LOCK_FD}" ]] ||
      die "inherited production deployment lock descriptor is not open"
    path_identity="$(stat -Lc '%d:%i' "${DEPLOY_STATE_DIR}")"
    fd_identity="$(stat -Lc '%d:%i' "/proc/${BASHPID}/fd/${PRODUCTION_DEPLOY_LOCK_FD}")"
    [[ "${path_identity}" == "${fd_identity}" ]] ||
      die "inherited production deployment lock targets a different state directory"
    flock --exclusive --nonblock "${PRODUCTION_DEPLOY_LOCK_FD}" ||
      die "inherited production deployment lock is not held by this controller"
    log "reusing the inherited host production deployment lock"
    return 0
  fi

  exec {PRODUCTION_DEPLOY_LOCK_FD}<"${DEPLOY_STATE_DIR}"
  path_identity="$(stat -Lc '%d:%i' "${DEPLOY_STATE_DIR}")"
  fd_identity="$(stat -Lc '%d:%i' "/proc/${BASHPID}/fd/${PRODUCTION_DEPLOY_LOCK_FD}")"
  [[ "${path_identity}" == "${fd_identity}" ]] ||
    die "deployment state directory changed while acquiring the production lock"
  flock --exclusive --nonblock "${PRODUCTION_DEPLOY_LOCK_FD}" ||
    die "another production deploy or rollback already holds the host deployment lock"
  export PRODUCTION_DEPLOY_LOCK_FD
  log "acquired the host production deployment lock"
}

new_deployment_attempt_id() {
  local attempt_id
  attempt_id="$(python3 - <<'PY'
from datetime import datetime, timezone
import uuid

timestamp = datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ")
print(f"{timestamp}-{uuid.uuid4().hex}")
PY
)"
  if [[ ! "${attempt_id}" =~ ^[0-9]{8}T[0-9]{6}Z-[0-9a-f]{32}$ ]]; then
    die "failed to generate a safe deployment attempt identifier"
  fi
  printf '%s\n' "${attempt_id}"
}

migrate_legacy_release_state_permissions() {
  local migrated_count
  migrated_count="$(python3 - "${DEPLOY_STATE_DIR}" <<'PY'
import errno
import os
import re
import stat
import sys
from pathlib import Path

state_dir = Path(sys.argv[1])
try:
    state_metadata = state_dir.lstat()
except FileNotFoundError:
    print(0)
    raise SystemExit(0)
if not stat.S_ISDIR(state_metadata.st_mode) or state_dir.is_symlink():
    raise SystemExit(f"deployment state path must be a regular directory: {state_dir}")
if state_metadata.st_uid != os.geteuid():
    raise SystemExit(f"deployment state directory must be owned by the deploy user: {state_dir}")

candidates = [state_dir / "current-release.env", state_dir / "releases.log"]
releases_dir = state_dir / "releases"
release_directory_migrated = False
try:
    releases_metadata = releases_dir.lstat()
except FileNotFoundError:
    releases_metadata = None
if releases_metadata is not None:
    if not stat.S_ISDIR(releases_metadata.st_mode) or releases_dir.is_symlink():
        raise SystemExit(f"release record path must be a regular directory: {releases_dir}")
    if releases_metadata.st_uid != os.geteuid():
        raise SystemExit(f"release record directory must be owned by the deploy user: {releases_dir}")
    if stat.S_IMODE(releases_metadata.st_mode) != 0o700:
        os.chmod(releases_dir, 0o700)
        release_directory_migrated = True
    tag_record = re.compile(r"[A-Za-z0-9][A-Za-z0-9._-]{0,127}\.env")
    with os.scandir(releases_dir) as entries:
        for entry in entries:
            if entry.name.endswith(".env") and not tag_record.fullmatch(entry.name):
                raise SystemExit(f"release record has an unsafe filename: {entry.path}")
            if tag_record.fullmatch(entry.name):
                candidates.append(Path(entry.path))

migrated = 1 if release_directory_migrated else 0
directory_fsync = {state_dir, releases_dir} if release_directory_migrated else set()
for path in candidates:
    flags = os.O_RDONLY
    if hasattr(os, "O_NOFOLLOW"):
        flags |= os.O_NOFOLLOW
    try:
        fd = os.open(path, flags)
    except FileNotFoundError:
        continue
    except OSError as exc:
        if exc.errno == errno.ELOOP:
            raise SystemExit(f"release state entry must not be a symlink: {path}") from exc
        raise
    try:
        metadata = os.fstat(fd)
        if not stat.S_ISREG(metadata.st_mode):
            raise SystemExit(f"release state entry is not a regular file: {path}")
        if metadata.st_uid != os.geteuid():
            raise SystemExit(f"release state entry must be owned by the deploy user: {path}")
        mode = stat.S_IMODE(metadata.st_mode)
        if mode == 0o600:
            continue
        if mode != 0o644:
            raise SystemExit(
                f"release state entry has an unsafe mode {mode:04o}; expected legacy 0644 or canonical 0600: {path}",
            )
        os.fchmod(fd, 0o600)
        os.fsync(fd)
        migrated += 1
        directory_fsync.add(path.parent)
    finally:
        os.close(fd)

for directory in directory_fsync:
    directory_fd = os.open(directory, os.O_RDONLY | os.O_DIRECTORY)
    try:
        os.fsync(directory_fd)
    finally:
        os.close(directory_fd)
print(migrated)
PY
)" || die "failed to normalize legacy release-state permissions"
  if [[ "${migrated_count}" != "0" ]]; then
    log "normalized ${migrated_count} legacy release-state path(s) to protected modes"
  fi
}

migrate_verified_legacy_current_release_identity() {
  local migrated_count
  migrated_count="$(python3 - "${DEPLOY_STATE_DIR}" "${STACK_NAME:-${COMPOSE_PROJECT_NAME:-stuhelper}}" <<'PY'
import hashlib
import json
import os
import re
import stat
import subprocess
import sys
import tempfile
from datetime import datetime, timezone
from pathlib import Path

state_dir = Path(sys.argv[1])
stack_name = sys.argv[2]
current_path = state_dir / "current-release.env"
fields = (
    "TAG",
    "DEPLOYED_AT",
    "BACKEND_IMAGE_REF",
    "FRONTEND_IMAGE_REF",
    "ADMIN_IMAGE_REF",
)
image_fields = fields[2:]
digest_ref_pattern = re.compile(r"^[^\s@]+@sha256:[0-9a-f]{64}$")
legacy_ref_pattern = re.compile(
    r"^[A-Za-z0-9][A-Za-z0-9._-]*(?::[0-9]+)?"
    r"(?:/[A-Za-z0-9][A-Za-z0-9._-]*)*"
    r":[A-Za-z0-9_][A-Za-z0-9._-]{0,127}$",
)
tag_pattern = re.compile(r"[A-Za-z0-9][A-Za-z0-9._-]{0,127}")
timestamp_pattern = re.compile(r"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z")


def fsync_directory(path: Path) -> None:
    directory_fd = os.open(path, os.O_RDONLY | os.O_DIRECTORY)
    try:
        os.fsync(directory_fd)
    finally:
        os.close(directory_fd)


def read_record(path: Path) -> tuple[bytes, dict[str, str]]:
    flags = os.O_RDONLY
    if hasattr(os, "O_NOFOLLOW"):
        flags |= os.O_NOFOLLOW
    fd = os.open(path, flags)
    try:
        metadata = os.fstat(fd)
        if not stat.S_ISREG(metadata.st_mode):
            raise SystemExit(f"legacy release record is not a regular file: {path}")
        if metadata.st_uid != os.geteuid():
            raise SystemExit(f"legacy release record must be owned by the deploy user: {path}")
        if stat.S_IMODE(metadata.st_mode) != 0o600:
            raise SystemExit(f"legacy release record must use mode 0600 after permission migration: {path}")
        with os.fdopen(fd, "rb", closefd=False) as stream:
            payload = stream.read()
    finally:
        os.close(fd)

    try:
        text = payload.decode("utf-8")
    except UnicodeDecodeError as exc:
        raise SystemExit(f"legacy release record is not UTF-8: {path}") from exc
    if not text.endswith("\n"):
        raise SystemExit(f"legacy release record is truncated: {path}")
    lines = text.splitlines()
    if len(lines) != len(fields):
        raise SystemExit(f"legacy release record is incomplete: {path}")

    values: dict[str, str] = {}
    for expected_key, line in zip(fields, lines, strict=True):
        if "=" not in line:
            raise SystemExit(f"legacy release record is malformed: {path}")
        key, value = line.split("=", 1)
        if key != expected_key or not value:
            raise SystemExit(f"legacy release record is not canonical: {path}")
        values[key] = value
    if not tag_pattern.fullmatch(values["TAG"]):
        raise SystemExit(f"legacy release record has an unsafe TAG: {path}")
    if not timestamp_pattern.fullmatch(values["DEPLOYED_AT"]):
        raise SystemExit(f"legacy release record has an invalid DEPLOYED_AT: {path}")
    for key in image_fields:
        value = values[key]
        if not digest_ref_pattern.fullmatch(value) and not legacy_ref_pattern.fullmatch(value):
            raise SystemExit(f"legacy release record has an invalid {key}: {path}")
    return payload, values


def stage_payload(path: Path, payload: bytes) -> Path:
    fd, temporary_name = tempfile.mkstemp(prefix=f".{path.name}.", dir=path.parent)
    temporary_path = Path(temporary_name)
    try:
        os.fchmod(fd, 0o600)
        with os.fdopen(fd, "wb") as stream:
            stream.write(payload)
            stream.flush()
            os.fsync(stream.fileno())
        return temporary_path
    except BaseException:
        try:
            os.close(fd)
        except OSError:
            pass
        temporary_path.unlink(missing_ok=True)
        raise


def atomic_write(path: Path, payload: bytes) -> None:
    temporary_path = stage_payload(path, payload)
    try:
        os.replace(temporary_path, path)
        fsync_directory(path.parent)
    except BaseException:
        temporary_path.unlink(missing_ok=True)
        raise


def read_protected_evidence(path: Path) -> bytes:
    flags = os.O_RDONLY
    if hasattr(os, "O_NOFOLLOW"):
        flags |= os.O_NOFOLLOW
    fd = os.open(path, flags)
    try:
        metadata = os.fstat(fd)
        if not stat.S_ISREG(metadata.st_mode):
            raise SystemExit(f"release migration evidence is not a regular file: {path}")
        if metadata.st_uid != os.geteuid() or stat.S_IMODE(metadata.st_mode) != 0o600:
            raise SystemExit(f"release migration evidence must be deploy-user-owned mode 0600: {path}")
        with os.fdopen(fd, "rb", closefd=False) as stream:
            return stream.read()
    finally:
        os.close(fd)


def docker_json(arguments: list[str], subject: str):
    completed = subprocess.run(
        ["docker", *arguments],
        check=False,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )
    if completed.returncode != 0:
        raise SystemExit(f"cannot verify legacy release identity from Docker {subject}")
    try:
        return json.loads(completed.stdout)
    except json.JSONDecodeError as exc:
        raise SystemExit(f"Docker returned malformed identity metadata for {subject}") from exc


def image_repository(reference: str) -> str:
    if "@" in reference:
        return reference.split("@", 1)[0]
    last_slash = reference.rfind("/")
    last_colon = reference.rfind(":")
    if last_colon <= last_slash:
        raise SystemExit(f"legacy image reference must contain an explicit tag: {reference}")
    return reference[:last_colon]


try:
    current_payload, current = read_record(current_path)
except FileNotFoundError:
    print(0)
    raise SystemExit(0)

if all(digest_ref_pattern.fullmatch(current[key]) for key in image_fields):
    print(0)
    raise SystemExit(0)

release_path = state_dir / "releases" / f"{current['TAG']}.env"
try:
    release_payload, release = read_record(release_path)
except FileNotFoundError as exc:
    raise SystemExit(
        f"legacy current release is missing its per-tag record: {release_path}",
    ) from exc

service_map = {
    "BACKEND_IMAGE_REF": ("app", f"{stack_name}-app"),
    "FRONTEND_IMAGE_REF": ("frontend", f"{stack_name}-frontend"),
    "ADMIN_IMAGE_REF": ("admin", f"{stack_name}-admin"),
}
canonical_refs: dict[str, str] = {}
verification: dict[str, dict[str, str]] = {}
container_format = (
    '{"imageId":{{json .Image}},"configuredImage":{{json .Config.Image}},'
    '"state":{{json .State.Status}},'
    '"project":{{json (index .Config.Labels "com.docker.compose.project")}},'
    '"service":{{json (index .Config.Labels "com.docker.compose.service")}}}'
)

for key in image_fields:
    recorded_ref = current[key]
    if digest_ref_pattern.fullmatch(recorded_ref):
        canonical_refs[key] = recorded_ref
        continue

    service, container_name = service_map[key]
    container = docker_json(
        ["inspect", "--type", "container", "--format", container_format, container_name],
        f"container {container_name}",
    )
    if not isinstance(container, dict):
        raise SystemExit(f"Docker returned invalid container identity metadata for {container_name}")
    if container.get("configuredImage") != recorded_ref:
        raise SystemExit(
            f"legacy {key} does not match the configured image of container {container_name}",
        )
    if container.get("project") != stack_name or container.get("service") != service:
        raise SystemExit(f"container {container_name} is not the expected Compose service")
    image_id = container.get("imageId")
    if not isinstance(image_id, str) or not re.fullmatch(r"sha256:[0-9a-f]{64}", image_id):
        raise SystemExit(f"container {container_name} has an invalid Docker image identity")
    state = container.get("state")
    if state not in {"created", "running", "paused", "restarting", "exited"}:
        raise SystemExit(f"container {container_name} has an unusable state for identity migration")

    repo_digests = docker_json(
        ["image", "inspect", "--format", "{{json .RepoDigests}}", image_id],
        f"image {image_id}",
    )
    if not isinstance(repo_digests, list) or not all(isinstance(item, str) for item in repo_digests):
        raise SystemExit(f"Docker image {image_id} has invalid RepoDigests metadata")
    repository = image_repository(recorded_ref)
    matches = sorted(
        {
            item
            for item in repo_digests
            if item.startswith(f"{repository}@sha256:") and digest_ref_pattern.fullmatch(item)
        },
    )
    if len(matches) != 1:
        raise SystemExit(
            f"container {container_name} does not provide exactly one verified digest for {repository}",
        )
    canonical_refs[key] = matches[0]
    verification[key] = {
        "legacyRef": recorded_ref,
        "digestRef": matches[0],
        "container": container_name,
        "containerState": state,
        "imageId": image_id,
    }

canonical = dict(current)
canonical.update(canonical_refs)
canonical_payload = "".join(f"{key}={canonical[key]}\n" for key in fields).encode()
if release_payload not in {current_payload, canonical_payload}:
    raise SystemExit(
        f"legacy current release does not match its per-tag record: {release_path}",
    )
if release["TAG"] != current["TAG"] or release["DEPLOYED_AT"] != current["DEPLOYED_AT"]:
    raise SystemExit(f"legacy current and per-tag release metadata differ: {release_path}")

evidence_dir = state_dir / "release-migrations"
try:
    evidence_metadata = evidence_dir.lstat()
except FileNotFoundError:
    evidence_dir.mkdir(mode=0o700)
    os.chmod(evidence_dir, 0o700)
    fsync_directory(state_dir)
else:
    if not stat.S_ISDIR(evidence_metadata.st_mode) or evidence_dir.is_symlink():
        raise SystemExit(f"release migration evidence path must be a regular directory: {evidence_dir}")
    if evidence_metadata.st_uid != os.geteuid() or stat.S_IMODE(evidence_metadata.st_mode) != 0o700:
        raise SystemExit(f"release migration evidence directory must be deploy-user-owned mode 0700: {evidence_dir}")

legacy_sha256 = hashlib.sha256(current_payload).hexdigest()
evidence_path = evidence_dir / f"{current['TAG']}.json"
evidence = {
    "schemaVersion": 1,
    "event": "legacy_release_identity_migrated",
    "tag": current["TAG"],
    "deployedAt": current["DEPLOYED_AT"],
    "migratedAt": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
    "legacyRecordSha256": legacy_sha256,
    "verificationSource": "running_compose_container_image_identity",
    "images": verification,
}
evidence_payload = (json.dumps(evidence, sort_keys=True, separators=(",", ":")) + "\n").encode()
try:
    existing_evidence = read_protected_evidence(evidence_path)
except FileNotFoundError:
    temporary_evidence = stage_payload(evidence_path, evidence_payload)
    try:
        try:
            os.link(temporary_evidence, evidence_path)
        except FileExistsError:
            existing_evidence = read_protected_evidence(evidence_path)
        else:
            fsync_directory(evidence_dir)
            existing_evidence = evidence_payload
    finally:
        temporary_evidence.unlink(missing_ok=True)

try:
    existing_document = json.loads(existing_evidence)
except (UnicodeDecodeError, json.JSONDecodeError) as exc:
    raise SystemExit(f"release migration evidence is malformed: {evidence_path}") from exc
if (
    existing_document.get("schemaVersion") != 1
    or existing_document.get("event") != "legacy_release_identity_migrated"
    or existing_document.get("tag") != current["TAG"]
    or existing_document.get("legacyRecordSha256") != legacy_sha256
    or {
        key: value.get("digestRef")
        for key, value in existing_document.get("images", {}).items()
        if isinstance(value, dict)
    }
    != {key: value["digestRef"] for key, value in verification.items()}
):
    raise SystemExit(f"release migration evidence conflicts with the verified transition: {evidence_path}")

# Replace the per-tag record first. A crash between these two writes is
# recoverable because a subsequent run accepts the already-canonical per-tag
# payload only when it exactly matches the independently re-verified result.
if release_payload != canonical_payload:
    atomic_write(release_path, canonical_payload)
atomic_write(current_path, canonical_payload)
print(1)
PY
)" || die "failed to migrate a verified legacy current release identity"
  if [[ "${migrated_count}" != "0" ]]; then
    log "migrated the legacy current release identity to verified image digests"
  fi
}

migrate_explicit_legacy_release_identity() {
  local tag="$1"
  local backend_ref="$2"
  local frontend_ref="$3"
  local admin_ref="$4"
  local actor="$5"
  local reason="$6"
  local migrated_count

  require_safe_release_tag "${tag}"
  require_digest_image_ref ROLLBACK_BACKEND_IMAGE_REF "${backend_ref}"
  require_digest_image_ref ROLLBACK_FRONTEND_IMAGE_REF "${frontend_ref}"
  require_digest_image_ref ROLLBACK_ADMIN_IMAGE_REF "${admin_ref}"
  migrated_count="$(python3 - \
    "${DEPLOY_STATE_DIR}" \
    "${tag}" \
    "${backend_ref}" \
    "${frontend_ref}" \
    "${admin_ref}" \
    "${actor}" \
    "${reason}" <<'PY'
import hashlib
import json
import os
import re
import stat
import sys
import tempfile
from datetime import datetime, timezone
from pathlib import Path

state_dir = Path(sys.argv[1])
tag, backend_ref, frontend_ref, admin_ref, actor, reason = sys.argv[2:]
fields = (
    "TAG",
    "DEPLOYED_AT",
    "BACKEND_IMAGE_REF",
    "FRONTEND_IMAGE_REF",
    "ADMIN_IMAGE_REF",
)
image_fields = fields[2:]
requested_refs = {
    "BACKEND_IMAGE_REF": backend_ref,
    "FRONTEND_IMAGE_REF": frontend_ref,
    "ADMIN_IMAGE_REF": admin_ref,
}
digest_ref_pattern = re.compile(r"^[^\s@]+@sha256:[0-9a-f]{64}$")
legacy_ref_pattern = re.compile(
    r"^[A-Za-z0-9][A-Za-z0-9._-]*(?::[0-9]+)?"
    r"(?:/[A-Za-z0-9][A-Za-z0-9._-]*)*"
    r":[A-Za-z0-9_][A-Za-z0-9._-]{0,127}$",
)
tag_pattern = re.compile(r"[A-Za-z0-9][A-Za-z0-9._-]{0,127}")
timestamp_pattern = re.compile(r"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z")
actor_pattern = re.compile(r"[A-Za-z0-9][A-Za-z0-9_.@-]{0,127}")


def fsync_directory(path: Path) -> None:
    directory_fd = os.open(path, os.O_RDONLY | os.O_DIRECTORY)
    try:
        os.fsync(directory_fd)
    finally:
        os.close(directory_fd)


def read_protected(path: Path, label: str) -> bytes:
    flags = os.O_RDONLY
    if hasattr(os, "O_NOFOLLOW"):
        flags |= os.O_NOFOLLOW
    fd = os.open(path, flags)
    try:
        metadata = os.fstat(fd)
        if not stat.S_ISREG(metadata.st_mode):
            raise SystemExit(f"{label} is not a regular file: {path}")
        if metadata.st_uid != os.geteuid() or stat.S_IMODE(metadata.st_mode) != 0o600:
            raise SystemExit(f"{label} must be deploy-user-owned mode 0600: {path}")
        with os.fdopen(fd, "rb", closefd=False) as stream:
            return stream.read()
    finally:
        os.close(fd)


def read_record(path: Path) -> tuple[bytes, dict[str, str]]:
    payload = read_protected(path, "legacy release record")
    try:
        text = payload.decode("utf-8")
    except UnicodeDecodeError as exc:
        raise SystemExit(f"legacy release record is not UTF-8: {path}") from exc
    if not text.endswith("\n"):
        raise SystemExit(f"legacy release record is truncated: {path}")
    lines = text.splitlines()
    if len(lines) != len(fields):
        raise SystemExit(f"legacy release record is incomplete: {path}")
    values: dict[str, str] = {}
    for expected_key, line in zip(fields, lines, strict=True):
        if "=" not in line:
            raise SystemExit(f"legacy release record is malformed: {path}")
        key, value = line.split("=", 1)
        if key != expected_key or not value:
            raise SystemExit(f"legacy release record is not canonical: {path}")
        values[key] = value
    if not tag_pattern.fullmatch(values["TAG"]):
        raise SystemExit(f"legacy release record has an unsafe TAG: {path}")
    if not timestamp_pattern.fullmatch(values["DEPLOYED_AT"]):
        raise SystemExit(f"legacy release record has an invalid DEPLOYED_AT: {path}")
    for key in image_fields:
        value = values[key]
        if not digest_ref_pattern.fullmatch(value) and not legacy_ref_pattern.fullmatch(value):
            raise SystemExit(f"legacy release record has an invalid {key}: {path}")
    return payload, values


def stage_payload(path: Path, payload: bytes) -> Path:
    fd, temporary_name = tempfile.mkstemp(prefix=f".{path.name}.", dir=path.parent)
    temporary_path = Path(temporary_name)
    try:
        os.fchmod(fd, 0o600)
        with os.fdopen(fd, "wb") as stream:
            stream.write(payload)
            stream.flush()
            os.fsync(stream.fileno())
        return temporary_path
    except BaseException:
        try:
            os.close(fd)
        except OSError:
            pass
        temporary_path.unlink(missing_ok=True)
        raise


def atomic_write(path: Path, payload: bytes) -> None:
    temporary_path = stage_payload(path, payload)
    try:
        os.replace(temporary_path, path)
        fsync_directory(path.parent)
    except BaseException:
        temporary_path.unlink(missing_ok=True)
        raise


def image_repository(reference: str) -> str:
    if "@" in reference:
        return reference.split("@", 1)[0]
    last_slash = reference.rfind("/")
    last_colon = reference.rfind(":")
    if last_colon <= last_slash:
        raise SystemExit(f"legacy image reference must contain an explicit tag: {reference}")
    return reference[:last_colon]


def validate_transition(
    source: dict[str, str],
    source_path: Path,
) -> tuple[bytes, dict[str, dict[str, str]]]:
    canonical = dict(source)
    migrated_images: dict[str, dict[str, str]] = {}
    for key in image_fields:
        recorded_ref = source[key]
        requested_ref = requested_refs[key]
        if digest_ref_pattern.fullmatch(recorded_ref):
            if recorded_ref != requested_ref:
                raise SystemExit(
                    f"explicit rollback {key} does not match canonical release record: {source_path}",
                )
            continue
        if image_repository(recorded_ref) != image_repository(requested_ref):
            raise SystemExit(
                f"explicit rollback {key} changes the legacy image repository: {source_path}",
            )
        canonical[key] = requested_ref
        migrated_images[key] = {
            "legacyRef": recorded_ref,
            "digestRef": requested_ref,
        }
    payload = "".join(f"{key}={canonical[key]}\n" for key in fields).encode()
    return payload, migrated_images


def validate_original_identity_pair(
    release: dict[str, str],
    current: dict[str, str],
) -> bool:
    release_identity = tuple(release[key] for key in image_fields)
    current_identity = tuple(current[key] for key in image_fields)
    if release_identity == current_identity:
        return False

    def is_requested_canonical(record: dict[str, str]) -> bool:
        return all(record[key] == requested_refs[key] for key in image_fields)

    def is_fully_legacy(record: dict[str, str]) -> bool:
        return all(legacy_ref_pattern.fullmatch(record[key]) is not None for key in image_fields)

    # Both migration paths publish evidence, then the immutable per-tag record,
    # then the current pointer. The only valid asymmetric retry state is
    # therefore release=canonical/current=legacy; the reverse order cannot be
    # produced by this controller and represents divergent history.
    if is_requested_canonical(release) and is_fully_legacy(current):
        for key in image_fields:
            if image_repository(release[key]) != image_repository(current[key]):
                break
        else:
            return True

    raise SystemExit("legacy current and per-tag release identities differ before migration")


release_path = state_dir / "releases" / f"{tag}.env"
release_payload, release = read_record(release_path)
if release["TAG"] != tag:
    raise SystemExit(f"legacy release record TAG does not match rollback target: {release_path}")
canonical_payload, release_migrations = validate_transition(release, release_path)

current_path = state_dir / "current-release.env"
current_payload = None
current = None
current_migrations: dict[str, dict[str, str]] = {}
requires_existing_migration_evidence = False
try:
    current_payload, current = read_record(current_path)
except FileNotFoundError:
    pass
if current is not None and current["TAG"] == tag:
    if current["DEPLOYED_AT"] != release["DEPLOYED_AT"]:
        raise SystemExit("legacy current and per-tag release timestamps differ")
    requires_existing_migration_evidence = validate_original_identity_pair(release, current)
    current_canonical_payload, current_migrations = validate_transition(current, current_path)
    if current_canonical_payload != canonical_payload:
        raise SystemExit("legacy current and per-tag release identities differ")

needs_release_write = release_payload != canonical_payload
needs_current_write = current is not None and current["TAG"] == tag and current_payload != canonical_payload
if not needs_release_write and not needs_current_write:
    print(0)
    raise SystemExit(0)

if not actor_pattern.fullmatch(actor):
    raise SystemExit("legacy rollback record migration requires a valid ROLLBACK_REVIEW_ACTOR")
if not 12 <= len(reason) <= 500 or any(ord(character) < 32 or ord(character) == 127 for character in reason):
    raise SystemExit("legacy rollback record migration requires a 12-500 character printable reason")

migrations = release_migrations or current_migrations
legacy_payload = release_payload if release_migrations else current_payload
if not migrations or legacy_payload is None:
    raise SystemExit("legacy rollback migration has no mutable identity to migrate")

evidence_dir = state_dir / "release-migrations"
try:
    evidence_metadata = evidence_dir.lstat()
except FileNotFoundError:
    evidence_dir.mkdir(mode=0o700)
    os.chmod(evidence_dir, 0o700)
    fsync_directory(state_dir)
else:
    if not stat.S_ISDIR(evidence_metadata.st_mode) or evidence_dir.is_symlink():
        raise SystemExit(f"release migration evidence path must be a regular directory: {evidence_dir}")
    if evidence_metadata.st_uid != os.geteuid() or stat.S_IMODE(evidence_metadata.st_mode) != 0o700:
        raise SystemExit(f"release migration evidence directory must be deploy-user-owned mode 0700: {evidence_dir}")

legacy_sha256 = hashlib.sha256(legacy_payload).hexdigest()
evidence_path = evidence_dir / f"{tag}.json"
evidence = {
    "schemaVersion": 1,
    "event": "legacy_release_identity_migrated",
    "tag": tag,
    "deployedAt": release["DEPLOYED_AT"],
    "migratedAt": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
    "legacyRecordSha256": legacy_sha256,
    "verificationSource": "explicit_provenance_verified_rollback_digest_override",
    "actor": actor,
    "reason": reason,
    "images": migrations,
}
evidence_payload = (json.dumps(evidence, sort_keys=True, separators=(",", ":")) + "\n").encode()
try:
    existing_evidence = read_protected(evidence_path, "release migration evidence")
except FileNotFoundError:
    if requires_existing_migration_evidence:
        raise SystemExit(
            f"release-first partial migration requires matching preexisting evidence: {evidence_path}",
        )
    temporary_evidence = stage_payload(evidence_path, evidence_payload)
    try:
        try:
            os.link(temporary_evidence, evidence_path)
        except FileExistsError:
            existing_evidence = read_protected(evidence_path, "release migration evidence")
        else:
            fsync_directory(evidence_dir)
            existing_evidence = evidence_payload
    finally:
        temporary_evidence.unlink(missing_ok=True)

try:
    existing_document = json.loads(existing_evidence)
except (UnicodeDecodeError, json.JSONDecodeError) as exc:
    raise SystemExit(f"release migration evidence is malformed: {evidence_path}") from exc
existing_digests = {
    key: value.get("digestRef")
    for key, value in existing_document.get("images", {}).items()
    if isinstance(value, dict)
}
expected_digests = {key: value["digestRef"] for key, value in migrations.items()}
if (
    existing_document.get("schemaVersion") != 1
    or existing_document.get("event") != "legacy_release_identity_migrated"
    or existing_document.get("tag") != tag
    or existing_document.get("legacyRecordSha256") != legacy_sha256
    or existing_digests != expected_digests
):
    raise SystemExit(f"release migration evidence conflicts with the explicit transition: {evidence_path}")

if needs_release_write:
    atomic_write(release_path, canonical_payload)
if needs_current_write:
    atomic_write(current_path, canonical_payload)
print(1)
PY
)" || die "failed to migrate the explicit legacy rollback identity"
  if [[ "${migrated_count}" != "0" ]]; then
    log "migrated the explicit legacy rollback identity to provenance-verified digests"
  fi
}

source_release_record_env_file() {
  local file="$1"
  local expected_tag="${2:-}"
  local allow_legacy_image_refs="${3:-false}"
  local diagnostic
  local key
  local -a required_keys=(
    TAG
    DEPLOYED_AT
    BACKEND_IMAGE_REF
    FRONTEND_IMAGE_REF
    ADMIN_IMAGE_REF
  )

  if [[ -n "${expected_tag}" ]]; then
    require_safe_release_tag "${expected_tag}"
  fi
  case "${allow_legacy_image_refs}" in
    true | false) ;;
    *) die "allow_legacy_image_refs must be true or false" ;;
  esac
  clear_process_control_environment
  if ! diagnostic="$(python3 - "${file}" "${required_keys[@]}" 2>&1 <<'PY'
import re
import sys
from pathlib import Path

path = Path(sys.argv[1])
required_keys = tuple(sys.argv[2:])
required = set(required_keys)
key_pattern = re.compile(r"[A-Za-z_][A-Za-z0-9_]*")
seen = set()

for lineno, line in enumerate(path.read_text().splitlines(), 1):
    stripped = line.strip()
    if not stripped or stripped.startswith("#"):
        continue
    if stripped.startswith("export "):
        stripped = stripped[len("export "):].lstrip()
    if "=" not in stripped:
        raise SystemExit(f"{path}:{lineno}: expected KEY=VALUE")
    key = stripped.split("=", 1)[0].strip()
    if not key_pattern.fullmatch(key):
        raise SystemExit(f"{path}:{lineno}: invalid env key: {key}")
    if key not in required:
        raise SystemExit(f"{path}:{lineno}: environment key {key} is not allowed in this release record")
    if key in seen:
        raise SystemExit(f"{path}:{lineno}: duplicate release record key: {key}")
    seen.add(key)

missing = [key for key in required_keys if key not in seen]
if missing:
    raise SystemExit(f"{path}: release record is missing required keys: {', '.join(missing)}")
PY
)"; then
    die "${diagnostic}"
  fi

  unset TAG DEPLOYED_AT BACKEND_IMAGE_REF FRONTEND_IMAGE_REF ADMIN_IMAGE_REF
  source_env_file "${file}" "${required_keys[@]}"

  for key in "${required_keys[@]}"; do
    if [[ -z "${!key:-}" ]]; then
      die "${file}: release record key ${key} must not be empty"
    fi
  done
  require_safe_release_tag "${TAG}"
  if [[ "${allow_legacy_image_refs}" != "true" ]]; then
    require_digest_image_ref BACKEND_IMAGE_REF "${BACKEND_IMAGE_REF}"
    require_digest_image_ref FRONTEND_IMAGE_REF "${FRONTEND_IMAGE_REF}"
    require_digest_image_ref ADMIN_IMAGE_REF "${ADMIN_IMAGE_REF}"
  fi
  if [[ -n "${expected_tag}" && "${TAG}" != "${expected_tag}" ]]; then
    die "${file}: release record TAG does not match rollback target ${expected_tag}"
  fi
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

materialize_postgres_runtime_urls() {
  local rendered
  rendered="$(python3 - <<'PY'
import os
import shlex
from urllib.parse import quote

mappings = (
    (
        "DATABASE_URL",
        "REPLACE_WITH_STUHELPER_APP_DB_PASSWORD",
        "STUHELPER_APP_DB_PASSWORD",
    ),
    (
        "BACKUP_DATABASE_URL",
        "REPLACE_WITH_STUHELPER_BACKUP_DB_PASSWORD",
        "STUHELPER_BACKUP_DB_PASSWORD",
    ),
    (
        "REPLICATION_DATABASE_URL",
        "REPLACE_WITH_STUHELPER_REPLICATION_DB_PASSWORD",
        "STUHELPER_REPLICATION_DB_PASSWORD",
    ),
)

for url_key, placeholder, secret_key in mappings:
    if url_key not in os.environ:
        continue
    value = os.environ[url_key]
    secret = os.environ.get(secret_key, "")
    # Environment initialization loads the shared template before generating
    # its secret file. Leave the placeholder intact during that first pass;
    # strict production validation rejects it if no later source resolves it.
    if placeholder in value and secret:
        value = value.replace(placeholder, quote(secret, safe=""))
    print(f"export {url_key}={shlex.quote(value)}")
PY
)"
  # shellcheck disable=SC1091
  source /dev/stdin <<<"${rendered}"
}

LOCAL_STATE_DIR="${LOCAL_STATE_DIR:-$(default_local_state_dir)}"
POSTGRES_WAL_RESTORE_DIR="${POSTGRES_WAL_RESTORE_DIR:-${LOCAL_STATE_DIR}/postgres/wal-restore}"

require_live_postgres_wal_archive_volume() {
  local volume_name="$1"
  local postgres_container="${2:-${STACK_NAME:-stuhelper}-postgres}"
  local container_running
  local mounts_json

  require_cmd docker
  require_cmd python3
  [[ -n "${volume_name}" ]] || die "PostgreSQL WAL archive volume name must not be empty"
  [[ -n "${postgres_container}" ]] || die "PostgreSQL container name must not be empty"

  if ! docker volume inspect "${volume_name}" >/dev/null 2>&1; then
    die "PostgreSQL WAL archive volume ${volume_name} does not exist; refusing to let Docker create an empty backup source"
  fi
  if ! container_running="$(docker container inspect --format '{{.State.Running}}' "${postgres_container}" 2>/dev/null)"; then
    die "failed to inspect production PostgreSQL container ${postgres_container}"
  fi
  [[ "${container_running}" == "true" ]] ||
    die "production PostgreSQL container ${postgres_container} is not running"
  if ! mounts_json="$(docker container inspect --format '{{json .Mounts}}' "${postgres_container}" 2>/dev/null)"; then
    die "failed to inspect mounts for production PostgreSQL container ${postgres_container}"
  fi

  if ! python3 - "${volume_name}" "${mounts_json}" <<'PY'
import json
import sys

volume_name = sys.argv[1]
try:
    mounts = json.loads(sys.argv[2])
except (json.JSONDecodeError, TypeError):
    raise SystemExit(1)

matches = [
    mount
    for mount in mounts
    if mount.get("Type") == "volume"
    and mount.get("Name") == volume_name
    and mount.get("Destination") == "/var/lib/postgresql/wal-archive"
    and mount.get("RW") is True
]
raise SystemExit(0 if len(matches) == 1 else 1)
PY
  then
    die "volume ${volume_name} is not the writable WAL archive mounted by ${postgres_container}; refusing backup sync"
  fi
}

postgres_wal_archiver_status() {
  local postgres_container="$1"
  local postgres_user="$2"
  local postgres_database="$3"
  local minimum_archived_count="${4:-}"
  local status_json
  local -a validator_args

  if ! status_json="$(docker exec "${postgres_container}" \
    psql \
    --no-psqlrc \
    --set=ON_ERROR_STOP=1 \
    --username="${postgres_user}" \
    --dbname="${postgres_database}" \
    --tuples-only \
    --no-align \
    --command="SELECT json_build_object(
      'archive_mode', current_setting('archive_mode'),
      'archive_command', current_setting('archive_command'),
      'archive_timeout', current_setting('archive_timeout'),
      'archived_count', archived_count,
      'failed_count', failed_count,
      'last_archived_wal', last_archived_wal,
      'last_archived_epoch', EXTRACT(EPOCH FROM last_archived_time),
      'last_failed_epoch', EXTRACT(EPOCH FROM last_failed_time),
      'postmaster_started_epoch', EXTRACT(EPOCH FROM pg_postmaster_start_time())
    )::text FROM pg_stat_archiver" 2>/dev/null)"; then
    return 1
  fi
  validator_args=(--status-json "${status_json}")
  if [[ -n "${minimum_archived_count}" ]]; then
    validator_args+=(--minimum-archived-count "${minimum_archived_count}")
  fi
  python3 "${COMMON_LIB_DIR}/../validate-postgres-wal-archiver.py" \
    "${validator_args[@]}"
}

postgres_wal_archived_count() {
  local postgres_container="$1"
  local postgres_user="$2"
  local postgres_database="$3"
  local archived_count

  if ! archived_count="$(docker exec "${postgres_container}" \
    psql \
    --no-psqlrc \
    --set=ON_ERROR_STOP=1 \
    --username="${postgres_user}" \
    --dbname="${postgres_database}" \
    --tuples-only \
    --no-align \
    --command='SELECT archived_count::text FROM pg_stat_archiver' 2>/dev/null)"; then
    return 1
  fi
  [[ "${archived_count}" =~ ^[0-9]+$ ]] || return 1
  printf '%s\n' "${archived_count}"
}

require_live_postgres_wal_archiving() {
  local postgres_container="$1"
  local postgres_user="${2:-${POSTGRES_USER:-stuhelper}}"
  local postgres_database="${3:-${POSTGRES_DB:-stuhelper}}"
  local archived_wal
  local attempt
  local baseline_archived_count
  local status

  [[ -n "${postgres_container}" ]] || die "PostgreSQL container name must not be empty"
  [[ -n "${postgres_user}" ]] || die "PostgreSQL archive probe user must not be empty"
  [[ -n "${postgres_database}" ]] || die "PostgreSQL archive probe database must not be empty"

  if postgres_wal_archiver_status \
    "${postgres_container}" "${postgres_user}" "${postgres_database}" >/dev/null; then
    :
  else
    status=$?
    [[ "${status}" -eq 2 ]] ||
      die "live PostgreSQL WAL archiver settings or status are invalid"
  fi

  baseline_archived_count="$(postgres_wal_archived_count \
    "${postgres_container}" "${postgres_user}" "${postgres_database}")" ||
    die "failed to read the live PostgreSQL WAL archive counter"

  log "forcing a fresh PostgreSQL WAL archive probe for this production gate"
  docker exec "${postgres_container}" \
    psql \
    --no-psqlrc \
    --set=ON_ERROR_STOP=1 \
    --username="${postgres_user}" \
    --dbname="${postgres_database}" \
    --tuples-only \
    --no-align \
    --command='SELECT pg_switch_wal()' >/dev/null 2>&1 ||
    die "failed to request a PostgreSQL WAL archive probe"

  archived_wal=""
  for ((attempt = 1; attempt <= 30; attempt++)); do
    if archived_wal="$(postgres_wal_archiver_status \
      "${postgres_container}" "${postgres_user}" "${postgres_database}" \
      "${baseline_archived_count}")"; then
      break
    else
      status=$?
    fi
    [[ "${status}" -eq 2 ]] || die "PostgreSQL WAL archiver became invalid during the live probe"
    sleep 1
  done
  [[ -n "${archived_wal}" ]] ||
    die "PostgreSQL did not archive a new WAL segment after the live probe"

  docker exec "${postgres_container}" /bin/sh -c \
    'test -f "/var/lib/postgresql/wal-archive/$1"' sh "${archived_wal}" >/dev/null 2>&1 ||
    die "PostgreSQL reported archived WAL ${archived_wal}, but it is missing from the protected volume"
}

require_external_postgres_pitr_evidence() {
  local expected_evidence_file="/etc/stuhelper/external-postgres-pitr-evidence.json"
  local evidence_file="${EXTERNAL_POSTGRES_PITR_EVIDENCE_FILE:-}"
  local system_identifier

  [[ "${EXTERNAL_POSTGRES_ENABLED:-false}" == "true" ]] || return 0
  [[ "${evidence_file}" == "${expected_evidence_file}" ]] ||
    die "EXTERNAL_POSTGRES_PITR_EVIDENCE_FILE must be ${expected_evidence_file} in production"
  [[ -n "${BACKUP_DATABASE_URL:-}" ]] ||
    die "BACKUP_DATABASE_URL is required to bind external PostgreSQL PITR evidence to the live cluster"
  require_cmd docker
  require_cmd python3

  if ! system_identifier="$(compose run --rm --no-deps -T \
    postgres-client \
    psql \
    --no-psqlrc \
    --set=ON_ERROR_STOP=1 \
    --tuples-only \
    --no-align \
    --quiet \
    --dbname="${BACKUP_DATABASE_URL}" \
    --command='SELECT system_identifier::text FROM pg_control_system()' 2>/dev/null)"; then
    die "failed to read the live external PostgreSQL system identifier"
  fi
  system_identifier="${system_identifier//[[:space:]]/}"
  [[ "${system_identifier}" =~ ^[0-9]{10,20}$ ]] ||
    die "live external PostgreSQL returned an invalid system identifier"

  python3 "${COMMON_LIB_DIR}/../validate-external-postgres-pitr-evidence.py" \
    --evidence-file "${evidence_file}" \
    --expected-system-identifier "${system_identifier}" \
    --expected-owner-uid 0 ||
    die "external PostgreSQL continuous WAL/PITR evidence is invalid"
}

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

require_https_object_storage_endpoint() {
  local key="$1"
  local value="${2:-}"
  local output

  if ! output="$(python3 - "${key}" "${value}" 2>&1 <<'PY'
import sys
from urllib.parse import urlsplit

key, value = sys.argv[1:3]
value = value.strip()
if not value:
    raise SystemExit(f"{key} is required")

parsed = urlsplit(value)
if parsed.scheme.lower() != "https":
    raise SystemExit(f"{key} must use https")
if not parsed.hostname:
    raise SystemExit(f"{key} must include a hostname")
if parsed.username is not None or parsed.password is not None:
    raise SystemExit(f"{key} must not contain embedded credentials")
if parsed.query or parsed.fragment:
    raise SystemExit(f"{key} must not contain query parameters or a fragment")
if parsed.path not in {"", "/"}:
    raise SystemExit(f"{key} must not contain an object path")
PY
  )"; then
    die "${output}"
  fi
}

require_off_host_backup_object_storage() {
  local confirmation="${BACKUP_OBJECT_STORAGE_OFF_HOST_CONFIRMED:-false}"
  local output

  case "${confirmation}" in
    true) ;;
    false|"")
      die "BACKUP_OBJECT_STORAGE_OFF_HOST_CONFIRMED must be true only after the backup target has been verified to survive loss of the production host"
      ;;
    *) die "BACKUP_OBJECT_STORAGE_OFF_HOST_CONFIRMED must be true or false" ;;
  esac

  require_https_object_storage_endpoint \
    "BACKUP_OBJECT_STORAGE_ENDPOINT" \
    "${BACKUP_OBJECT_STORAGE_ENDPOINT:-}"
  unset BACKUP_OBJECT_STORAGE_PINNED_HOSTS
  if ! output="$(python3 - "${BACKUP_OBJECT_STORAGE_ENDPOINT:-}" 2>&1 <<'PY'
import ipaddress
import json
import os
import re
import socket
import subprocess
import sys
from urllib.parse import urlsplit

endpoint = sys.argv[1].strip()
parsed_endpoint = urlsplit(endpoint)
raw_host = (parsed_endpoint.hostname or "").lower()
if not raw_host:
    raise SystemExit("BACKUP_OBJECT_STORAGE_ENDPOINT must include a hostname")
if "%" in raw_host:
    raise SystemExit(
        "BACKUP_OBJECT_STORAGE_ENDPOINT must not use an IPv6 zone identifier"
    )
if raw_host.endswith("."):
    raise SystemExit(
        "BACKUP_OBJECT_STORAGE_ENDPOINT must not use a trailing-dot hostname"
    )
host = raw_host

local_identity_spec = os.environ.get(
    "BACKUP_OBJECT_STORAGE_LOCAL_IDENTITY_CIDRS", ""
).strip()
if not local_identity_spec:
    raise SystemExit(
        "BACKUP_OBJECT_STORAGE_LOCAL_IDENTITY_CIDRS is required; list every public/NAT/LB address or CIDR that can route back to this production host, or set none only after verifying no such identity exists"
    )
local_identity_networks = set()
if local_identity_spec.lower() != "none":
    for item in re.split(r"[\s,]+", local_identity_spec):
        if not item:
            continue
        try:
            local_identity_networks.add(ipaddress.ip_network(item, strict=False))
        except ValueError:
            raise SystemExit(
                f"BACKUP_OBJECT_STORAGE_LOCAL_IDENTITY_CIDRS contains an invalid address or CIDR: {item}"
            ) from None
    if not local_identity_networks:
        raise SystemExit(
            "BACKUP_OBJECT_STORAGE_LOCAL_IDENTITY_CIDRS must contain at least one address or CIDR, or be exactly none"
        )

try:
    address = ipaddress.ip_address(host)
except ValueError:
    address = None

if address is None:
    try:
        socket.inet_aton(host)
    except OSError:
        pass
    else:
        raise SystemExit(
            "BACKUP_OBJECT_STORAGE_ENDPOINT must not use a legacy or abbreviated numeric IPv4 address"
        )

if address is None and (
    "." not in host
    or host == "host.docker.internal"
    or host.endswith(".localhost")
    or host.endswith(".local")
):
    raise SystemExit(
        "BACKUP_OBJECT_STORAGE_ENDPOINT must use an off-host fully-qualified hostname or a non-local IP address"
    )
if address is None:
    labels = host.split(".")
    if len(host) > 253 or any(
        not re.fullmatch(r"[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?", label)
        for label in labels
    ):
        raise SystemExit(
            "BACKUP_OBJECT_STORAGE_ENDPOINT must use a valid ASCII DNS hostname"
        )

force_path_style = os.environ.get(
    "BACKUP_OBJECT_STORAGE_FORCE_PATH_STYLE",
    os.environ.get("OBJECT_STORAGE_FORCE_PATH_STYLE", "true"),
)
if force_path_style not in {"true", "false"}:
    raise SystemExit(
        "BACKUP_OBJECT_STORAGE_FORCE_PATH_STYLE must be true or false"
    )

bucket = os.environ.get("BACKUP_OBJECT_STORAGE_BUCKET", "").strip()
provider_private_endpoint = os.environ.get(
    "BACKUP_OBJECT_STORAGE_PROVIDER_PRIVATE_ENDPOINT", "none"
).strip() or "none"
if provider_private_endpoint not in {"none", "tencent-cos"}:
    raise SystemExit(
        "BACKUP_OBJECT_STORAGE_PROVIDER_PRIVATE_ENDPOINT must be none or tencent-cos"
    )

allow_provider_link_local = False
if provider_private_endpoint == "tencent-cos":
    provider = os.environ.get("BACKUP_OBJECT_STORAGE_PROVIDER", "").strip()
    region = os.environ.get("BACKUP_OBJECT_STORAGE_REGION", "").strip()
    tls_insecure = os.environ.get(
        "BACKUP_OBJECT_STORAGE_TLS_INSECURE", "false"
    ).strip()
    tls_ca = os.environ.get("BACKUP_OBJECT_STORAGE_TLS_CA", "").strip()
    if provider != "TencentCOS":
        raise SystemExit(
            "tencent-cos provider-private endpoint requires BACKUP_OBJECT_STORAGE_PROVIDER=TencentCOS"
        )
    if force_path_style != "false":
        raise SystemExit(
            "tencent-cos provider-private endpoint requires virtual-hosted S3 addressing"
        )
    try:
        endpoint_port = parsed_endpoint.port
    except ValueError as error:
        raise SystemExit(
            f"tencent-cos provider-private endpoint has an invalid port: {error}"
        ) from None
    if endpoint_port not in {None, 443}:
        raise SystemExit(
            "tencent-cos provider-private endpoint must use the default HTTPS port"
        )
    endpoint_match = re.fullmatch(
        r"cos\.([a-z]{2,}(?:-[a-z0-9]+)+)\.(?:myqcloud\.com|tencentcos\.cn)",
        host,
    )
    if endpoint_match is None:
        raise SystemExit(
            "tencent-cos provider-private endpoint must use an official regional COS service hostname"
        )
    if region != endpoint_match.group(1):
        raise SystemExit(
            "tencent-cos provider-private endpoint region must match BACKUP_OBJECT_STORAGE_REGION"
        )
    if tls_insecure != "false" or tls_ca:
        raise SystemExit(
            "tencent-cos provider-private endpoint requires public-CA TLS verification"
        )
    if not re.fullmatch(
        r"(?=.{3,63}$)[a-z0-9][a-z0-9.-]*-[1-9][0-9]{4,}", bucket
    ):
        raise SystemExit(
            "tencent-cos provider-private endpoint requires a full BucketName-APPID bucket name"
        )
    allow_provider_link_local = True

transfer_hosts = [host]
if address is None and force_path_style == "false":
    if not bucket:
        raise SystemExit(
            "BACKUP_OBJECT_STORAGE_BUCKET is required for virtual-hosted backup transfers"
        )
    virtual_host = f"{bucket}.{host}"
    virtual_labels = virtual_host.split(".")
    if len(virtual_host) > 253 or any(
        not re.fullmatch(r"[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?", label)
        for label in virtual_labels
    ):
        raise SystemExit(
            "BACKUP_OBJECT_STORAGE_BUCKET must form a valid lowercase ASCII virtual-hosted S3 hostname"
        )
    transfer_hosts.append(virtual_host)

image_ref = os.environ.get(
    "RCLONE_IMAGE_REF",
    "rclone/rclone:beta@sha256:f52965eba611ba8984117638b2a0539dcce170731937f93fbace66897d102698",
)
if not re.fullmatch(r".+@sha256:[0-9a-f]{64}", image_ref):
    raise SystemExit("RCLONE_IMAGE_REF must be a complete image@sha256 reference")
docker_network = os.environ.get("BACKUP_OBJECT_STORAGE_DOCKER_NETWORK", "")
if docker_network and not re.fullmatch(r"[A-Za-z0-9_.-]+", docker_network):
    raise SystemExit(
        "BACKUP_OBJECT_STORAGE_DOCKER_NETWORK contains unsupported characters"
    )
if docker_network in {"host", "none"}:
    raise SystemExit(
        "BACKUP_OBJECT_STORAGE_DOCKER_NETWORK must not use host or none for off-host production backups"
    )
effective_docker_network = docker_network or "bridge"

if address is not None:
    resolved_addresses_by_host = {host: {address}}
else:
    resolved_addresses_by_host = {}
    resolver_command = [
        "docker",
        "run",
        "--rm",
        "--read-only",
        "--cap-drop",
        "ALL",
        "--security-opt",
        "no-new-privileges",
    ]
    if docker_network:
        resolver_command.extend(["--network", docker_network])
    resolver_command.extend(["--entrypoint", "/usr/bin/getent", image_ref])
    for transfer_host in transfer_hosts:
        resolved_addresses = set()
        for database in ("ahostsv4", "ahostsv6"):
            try:
                result = subprocess.run(
                    [*resolver_command, database, transfer_host],
                    check=False,
                    capture_output=True,
                    text=True,
                    timeout=30,
                )
            except FileNotFoundError:
                raise SystemExit(
                    "docker is required to resolve BACKUP_OBJECT_STORAGE_ENDPOINT in the rclone network namespace"
                ) from None
            except subprocess.TimeoutExpired:
                raise SystemExit(
                    f"backup transfer hostname {transfer_host} resolution timed out in the rclone network namespace"
                ) from None
            for line in result.stdout.splitlines():
                fields = line.split()
                if not fields:
                    continue
                candidate = fields[0].split("%", 1)[0]
                try:
                    resolved_addresses.add(ipaddress.ip_address(candidate))
                except ValueError:
                    continue
        if not resolved_addresses:
            raise SystemExit(
                f"backup transfer hostname {transfer_host} must resolve to at least one A or AAAA address in the rclone network namespace"
            )
        resolved_addresses_by_host[transfer_host] = resolved_addresses

try:
    docker_network_result = subprocess.run(
        ["docker", "network", "inspect", effective_docker_network],
        check=True,
        capture_output=True,
        text=True,
        timeout=10,
    )
    docker_networks = json.loads(docker_network_result.stdout)
except FileNotFoundError:
    raise SystemExit(
        "docker is required to inspect the rclone network while verifying BACKUP_OBJECT_STORAGE_ENDPOINT"
    ) from None
except subprocess.TimeoutExpired:
    raise SystemExit(
        "rclone Docker network inspection timed out while verifying BACKUP_OBJECT_STORAGE_ENDPOINT"
    ) from None
except (subprocess.CalledProcessError, json.JSONDecodeError) as error:
    raise SystemExit(
        f"failed to inspect the rclone Docker network while verifying BACKUP_OBJECT_STORAGE_ENDPOINT: {error}"
    ) from None

local_docker_subnets = set()
local_docker_addresses = set()
for network in docker_networks:
    driver = str(network.get("Driver", ""))
    if driver not in {"bridge", "macvlan", "ipvlan", "overlay"}:
        raise SystemExit(
            f"unsupported rclone Docker network driver for off-host verification: {driver or 'unknown'}"
        )
    if driver == "bridge":
        for config in network.get("IPAM", {}).get("Config", []) or []:
            subnet = str(config.get("Subnet", "")).split("%", 1)[0]
            try:
                local_docker_subnets.add(ipaddress.ip_network(subnet, strict=False))
            except ValueError:
                continue
    for container in (network.get("Containers") or {}).values():
        for key in ("IPv4Address", "IPv6Address"):
            candidate = str(container.get(key, "")).split("/", 1)[0].split("%", 1)[0]
            try:
                local_docker_addresses.add(ipaddress.ip_address(candidate))
            except ValueError:
                continue

try:
    local_result = subprocess.run(
        ["ip", "-j", "address", "show"],
        check=True,
        capture_output=True,
        text=True,
    )
    local_interfaces = json.loads(local_result.stdout)
except FileNotFoundError:
    raise SystemExit(
        "iproute2 is required to verify that BACKUP_OBJECT_STORAGE_ENDPOINT is off-host"
    ) from None
except (subprocess.CalledProcessError, json.JSONDecodeError) as error:
    raise SystemExit(
        f"failed to enumerate local addresses while verifying BACKUP_OBJECT_STORAGE_ENDPOINT: {error}"
    ) from None

local_addresses = set()
for interface in local_interfaces:
    for info in interface.get("addr_info", []):
        candidate = str(info.get("local", "")).split("%", 1)[0]
        try:
            local_addresses.add(ipaddress.ip_address(candidate))
        except ValueError:
            continue

def normalize(value):
    if isinstance(value, ipaddress.IPv6Address) and value.ipv4_mapped is not None:
        return value.ipv4_mapped
    return value

def equivalent_addresses(value):
    normalized = normalize(value)
    values = {value, normalized}
    if isinstance(normalized, ipaddress.IPv4Address):
        values.add(ipaddress.IPv6Address(f"::ffff:{normalized}"))
    return values

normalized_local_addresses = {normalize(value) for value in local_addresses}
normalized_local_docker_addresses = {
    normalize(value) for value in local_docker_addresses
}
for resolved_address in {
    value
    for values in resolved_addresses_by_host.values()
    for value in values
}:
    normalized_address = normalize(resolved_address)
    comparable_addresses = equivalent_addresses(resolved_address)
    provider_link_local = (
        allow_provider_link_local
        and isinstance(normalized_address, ipaddress.IPv4Address)
        and normalized_address.is_link_local
    )
    if (
        normalized_address.is_loopback
        or normalized_address.is_unspecified
        or (normalized_address.is_link_local and not provider_link_local)
        or normalized_address.is_multicast
    ):
        raise SystemExit(
            "BACKUP_OBJECT_STORAGE_ENDPOINT must not resolve to a loopback, unspecified, link-local, or multicast address"
        )
    if normalized_address in normalized_local_addresses:
        raise SystemExit(
            "BACKUP_OBJECT_STORAGE_ENDPOINT must not resolve to an address assigned to the production host"
        )
    if any(
        candidate.version == identity.version and candidate in identity
        for identity in local_identity_networks
        for candidate in comparable_addresses
    ):
        raise SystemExit(
            "BACKUP_OBJECT_STORAGE_ENDPOINT must not resolve to a configured public/NAT/LB identity of the production host"
        )
    if normalized_address in normalized_local_docker_addresses or any(
        normalized_address.version == subnet.version and normalized_address in subnet
        for subnet in local_docker_subnets
    ):
        raise SystemExit(
            "BACKUP_OBJECT_STORAGE_ENDPOINT must not resolve into a Docker network hosted on the production host"
        )

if address is None:
    for transfer_host in transfer_hosts:
        pinned_addresses = {
            normalize(value)
            for value in resolved_addresses_by_host[transfer_host]
        }
        for pinned_address in sorted(
            pinned_addresses, key=lambda value: (value.version, int(value))
        ):
            print(f"{transfer_host}={pinned_address.compressed}")
PY
  )"; then
    die "${output}"
  fi
  export BACKUP_OBJECT_STORAGE_PINNED_HOSTS="${output}"
}

require_production_object_storage() {
  require_backup_object_storage_config
  require_https_object_storage_endpoint "OBJECT_STORAGE_ENDPOINT" "${OBJECT_STORAGE_ENDPOINT:-}"
  require_https_object_storage_endpoint "BACKUP_OBJECT_STORAGE_ENDPOINT" "${BACKUP_OBJECT_STORAGE_ENDPOINT:-}"
  require_off_host_backup_object_storage

  [[ "${OBJECT_STORAGE_USE_SSL:-false}" == "true" ]] ||
    die "OBJECT_STORAGE_USE_SSL must be true for production"
  [[ "${BACKUP_OBJECT_STORAGE_TLS_INSECURE:-false}" == "false" ]] ||
    die "BACKUP_OBJECT_STORAGE_TLS_INSECURE must be false for production"

  case "${OBJECT_STORAGE_FORCE_PATH_STYLE:-false}" in
    true|false) ;;
    *) die "OBJECT_STORAGE_FORCE_PATH_STYLE must be true or false" ;;
  esac
  case "${BACKUP_OBJECT_STORAGE_FORCE_PATH_STYLE:-${OBJECT_STORAGE_FORCE_PATH_STYLE:-false}}" in
    true|false) ;;
    *) die "BACKUP_OBJECT_STORAGE_FORCE_PATH_STYLE must be true or false" ;;
  esac

  if [[ -n "${OBJECT_STORAGE_TLS_CA:-}" ]]; then
    [[ "${OBJECT_STORAGE_TLS_CA}" == "/object-storage-tls/ca.crt" ]] ||
      die "OBJECT_STORAGE_TLS_CA must be /object-storage-tls/ca.crt"
    [[ -n "${OBJECT_STORAGE_TLS_CA_HOST_PATH:-}" ]] ||
      die "OBJECT_STORAGE_TLS_CA_HOST_PATH is required when OBJECT_STORAGE_TLS_CA is configured"
    [[ -f "${OBJECT_STORAGE_TLS_CA_HOST_PATH}" &&
       -r "${OBJECT_STORAGE_TLS_CA_HOST_PATH}" ]] ||
      die "OBJECT_STORAGE_TLS_CA_HOST_PATH must be a readable regular file"
  elif [[ -n "${OBJECT_STORAGE_TLS_CA_HOST_PATH:-}" ]]; then
    die "OBJECT_STORAGE_TLS_CA_HOST_PATH must be empty when OBJECT_STORAGE_TLS_CA is empty"
  fi

  if [[ -n "${BACKUP_OBJECT_STORAGE_TLS_CA:-}" ]]; then
    [[ -f "${BACKUP_OBJECT_STORAGE_TLS_CA}" && -r "${BACKUP_OBJECT_STORAGE_TLS_CA}" ]] ||
      die "BACKUP_OBJECT_STORAGE_TLS_CA must be a readable regular file"
  fi

  if [[ "${OBJECT_STORAGE_ACCESS_KEY_ID:-}" == "${BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID:-}" &&
        "${OBJECT_STORAGE_SECRET_ACCESS_KEY:-}" != "${BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY:-}" ]]; then
    die "the same object-storage access key cannot map to different secrets"
  fi
  if [[ "${OBJECT_STORAGE_ACCESS_KEY_ID:-}" != "${BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID:-}" &&
        "${OBJECT_STORAGE_SECRET_ACCESS_KEY:-}" == "${BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY:-}" ]]; then
    die "application and backup object-storage identities must not share a secret"
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
  local output

  if ! output="$(python3 - "${key}" "${value}" 2>&1 <<'PY'
import sys
from urllib.parse import parse_qs, urlsplit

key = sys.argv[1]
value = sys.argv[2].strip()

if not value:
    raise SystemExit(f"{key} must be configured for production PostgreSQL")
if "REPLACE_WITH_" in value:
    raise SystemExit(f"{key} contains an unresolved secret placeholder")

parsed = urlsplit(value)
if parsed.scheme not in {"postgres", "postgresql"}:
    raise SystemExit(f"{key} must be a postgres/postgresql URL")

host = (parsed.hostname or "").lower()
if host in {"localhost", "127.0.0.1", "::1"}:
    raise SystemExit(f"{key} must not point to a local/development PostgreSQL endpoint ({host})")

query = parse_qs(parsed.query, keep_blank_values=True)
ssl_modes = query.get("sslmode", [])
ssl_mode = ssl_modes[0] if ssl_modes else ""
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
  [[ "${EXTERNAL_POSTGRES_ALLOW_PLAINTEXT:-false}" != "true" ]] ||
    die "EXTERNAL_POSTGRES_ALLOW_PLAINTEXT is only allowed in prod-parity; production PostgreSQL must use verified TLS"

  if [[ "${EXTERNAL_POSTGRES_ENABLED:-false}" == "true" ]]; then
    require_verified_postgres_ssl_mode "POSTGRES_INTERNAL_SSL_MODE" "${POSTGRES_INTERNAL_SSL_MODE:-}"
    [[ "${DB_SSL_MODE:-}" == "verify-full" ]] || die "DB_SSL_MODE must be verify-full for external production PostgreSQL"
    [[ "${DB_SSL_ROOT_CERT:-}" == "/tls/ca.crt" ]] ||
      die "DB_SSL_ROOT_CERT must be /tls/ca.crt for external production PostgreSQL"
    [[ -n "${POSTGRES_CLIENT_CA_HOST_PATH:-}" ]] ||
      die "POSTGRES_CLIENT_CA_HOST_PATH is required for external production PostgreSQL TLS"
  else
    [[ "${POSTGRES_ENABLE_SSL:-}" == "on" ]] || die "POSTGRES_ENABLE_SSL must be on for production"
    require_verified_postgres_ssl_mode "POSTGRES_INTERNAL_SSL_MODE" "${POSTGRES_INTERNAL_SSL_MODE:-}"
    [[ "${DB_SSL_MODE:-}" == "verify-full" ]] || die "DB_SSL_MODE must be verify-full for production"
    [[ "${DB_SSL_ROOT_CERT:-}" == "/tls/ca.crt" ]] ||
      die "DB_SSL_ROOT_CERT must be /tls/ca.crt for production"
  fi

  require_production_postgres_url "DATABASE_URL" "${DATABASE_URL:-}"
  require_production_postgres_url "BACKUP_DATABASE_URL" "${BACKUP_DATABASE_URL:-}"
  require_production_postgres_url "REPLICATION_DATABASE_URL" "${REPLICATION_DATABASE_URL:-}"
}

require_production_postgres_archiving() {
  if [[ "${EXTERNAL_POSTGRES_ENABLED:-false}" == "true" ]]; then
    [[ "${EXTERNAL_POSTGRES_PITR_EVIDENCE_FILE:-}" == "/etc/stuhelper/external-postgres-pitr-evidence.json" ]] ||
      die "EXTERNAL_POSTGRES_PITR_EVIDENCE_FILE must be /etc/stuhelper/external-postgres-pitr-evidence.json for external production PostgreSQL"
    return 0
  fi
  [[ "${POSTGRES_ARCHIVE_MODE:-}" == "on" ]] ||
    die "POSTGRES_ARCHIVE_MODE must be on for the internal production PostgreSQL service"
  [[ "${POSTGRES_ARCHIVE_TIMEOUT:-}" == "15min" ]] ||
    die "POSTGRES_ARCHIVE_TIMEOUT must be 15min for the internal production PostgreSQL service"
}

require_production_external_student_source_security() {
  [[ "${EXTERNAL_STUDENT_SOURCE_ENABLED:-false}" == "true" ]] || return 0
  [[ "${EXTERNAL_STUDENT_SOURCE_PROVIDER:-}" == "oracle" ]] ||
    die "EXTERNAL_STUDENT_SOURCE_PROVIDER must be oracle when the external student source is enabled"

  local key
  local value
  for key in \
    EXTERNAL_STUDENT_SOURCE_SCHOOL_CODE \
    EXTERNAL_STUDENT_SOURCE_ORACLE_HOST \
    EXTERNAL_STUDENT_SOURCE_ORACLE_SERVICE_NAME \
    EXTERNAL_STUDENT_SOURCE_ORACLE_USERNAME \
    EXTERNAL_STUDENT_SOURCE_ORACLE_READONLY_USERNAME \
    EXTERNAL_STUDENT_SOURCE_ORACLE_PASSWORD \
    EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_CA_HOST_PATH \
    EXTERNAL_STUDENT_SOURCE_ORACLE_SCHEMA \
    EXTERNAL_STUDENT_SOURCE_ORACLE_TABLE \
    EXTERNAL_STUDENT_SOURCE_ORACLE_STUDENT_ID_COLUMN \
    EXTERNAL_STUDENT_SOURCE_ORACLE_STUDENT_NAME_COLUMN; do
    value="${!key:-}"
    [[ -n "${value}" ]] || die "${key} is required when the external student source is enabled"
    case "${value}" in
      REPLACE_WITH_* | RUN_*)
        die "${key} contains an unresolved placeholder"
        ;;
    esac
  done

  [[ "${EXTERNAL_STUDENT_SOURCE_SCHOOL_CODE}" =~ ^[0-9]{10}$ ]] ||
    die "EXTERNAL_STUDENT_SOURCE_SCHOOL_CODE must be a 10-digit school code"
  for key in \
    EXTERNAL_STUDENT_SOURCE_ORACLE_USERNAME \
    EXTERNAL_STUDENT_SOURCE_ORACLE_READONLY_USERNAME \
    EXTERNAL_STUDENT_SOURCE_ORACLE_SCHEMA \
    EXTERNAL_STUDENT_SOURCE_ORACLE_TABLE \
    EXTERNAL_STUDENT_SOURCE_ORACLE_STUDENT_ID_COLUMN \
    EXTERNAL_STUDENT_SOURCE_ORACLE_STUDENT_NAME_COLUMN; do
    value="${!key}"
    [[ "${value}" =~ ^[A-Za-z][A-Za-z0-9_]{0,127}$ ]] ||
      die "${key} must be a safe Oracle identifier"
  done
  [[ "${EXTERNAL_STUDENT_SOURCE_ORACLE_USERNAME^^}" == "${EXTERNAL_STUDENT_SOURCE_ORACLE_READONLY_USERNAME^^}" ]] ||
    die "EXTERNAL_STUDENT_SOURCE_ORACLE_USERNAME must match EXTERNAL_STUDENT_SOURCE_ORACLE_READONLY_USERNAME"
  [[ "${EXTERNAL_STUDENT_SOURCE_ORACLE_USERNAME^^}" != "${EXTERNAL_STUDENT_SOURCE_ORACLE_SCHEMA^^}" ]] ||
    die "EXTERNAL_STUDENT_SOURCE_ORACLE_USERNAME must not own the source schema"

  [[ "${EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_MODE:-}" == "verify-full" ]] ||
    die "EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_MODE must be verify-full in production"
  [[ "${EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_CA_FILE:-}" == "/external-student-source-tls/ca.crt" ]] ||
    die "EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_CA_FILE must be /external-student-source-tls/ca.crt in production"

  require_integer_range "EXTERNAL_STUDENT_SOURCE_ORACLE_PORT" "${EXTERNAL_STUDENT_SOURCE_ORACLE_PORT:-}" 1 65535
  require_integer_range "EXTERNAL_STUDENT_SOURCE_ORACLE_CONNECT_TIMEOUT_SECONDS" "${EXTERNAL_STUDENT_SOURCE_ORACLE_CONNECT_TIMEOUT_SECONDS:-}" 1 60
  require_integer_range "EXTERNAL_STUDENT_SOURCE_ORACLE_QUERY_TIMEOUT_SECONDS" "${EXTERNAL_STUDENT_SOURCE_ORACLE_QUERY_TIMEOUT_SECONDS:-}" 1 60
  require_integer_range "EXTERNAL_STUDENT_SOURCE_ORACLE_MAX_OPEN_CONNS" "${EXTERNAL_STUDENT_SOURCE_ORACLE_MAX_OPEN_CONNS:-}" 1 100
  require_integer_range "EXTERNAL_STUDENT_SOURCE_ORACLE_MAX_IDLE_CONNS" "${EXTERNAL_STUDENT_SOURCE_ORACLE_MAX_IDLE_CONNS:-}" 0 "${EXTERNAL_STUDENT_SOURCE_ORACLE_MAX_OPEN_CONNS}"
  require_integer_range "EXTERNAL_STUDENT_SOURCE_ORACLE_CONN_MAX_LIFETIME_SECONDS" "${EXTERNAL_STUDENT_SOURCE_ORACLE_CONN_MAX_LIFETIME_SECONDS:-}" 30 3600
  require_integer_range "EXTERNAL_STUDENT_SOURCE_ORACLE_CONN_MAX_IDLE_TIME_SECONDS" "${EXTERNAL_STUDENT_SOURCE_ORACLE_CONN_MAX_IDLE_TIME_SECONDS:-}" 30 "${EXTERNAL_STUDENT_SOURCE_ORACLE_CONN_MAX_LIFETIME_SECONDS}"
  require_integer_range "EXTERNAL_STUDENT_SOURCE_ORACLE_BREAKER_FAILURE_THRESHOLD" "${EXTERNAL_STUDENT_SOURCE_ORACLE_BREAKER_FAILURE_THRESHOLD:-}" 1 100
  require_integer_range "EXTERNAL_STUDENT_SOURCE_ORACLE_BREAKER_SUCCESS_THRESHOLD" "${EXTERNAL_STUDENT_SOURCE_ORACLE_BREAKER_SUCCESS_THRESHOLD:-}" 1 20
  require_integer_range "EXTERNAL_STUDENT_SOURCE_ORACLE_BREAKER_OPEN_SECONDS" "${EXTERNAL_STUDENT_SOURCE_ORACLE_BREAKER_OPEN_SECONDS:-}" 1 600
}

trim_trailing_slash() {
  local value="$1"
  printf '%s\n' "${value%/}"
}

repo_default_path_matches() {
  local current="$1"
  local common_default="$2"
  local normalized

  if [[ -z "${current}" ]]; then
    return 0
  fi
  if [[ -z "${common_default}" ]]; then
    return 1
  fi
  if [[ "${current}" == "${common_default}" ]]; then
    return 0
  fi

  case "${current}" in
    /*) normalized="${current}" ;;
    *) normalized="${REPO_ROOT}/${current#./}" ;;
  esac

  [[ "${normalized}" == "${common_default}" ]]
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
  local enabled="${PUBLIC_INGRESS_PUBLIC_DNS_ENABLED:-true}"
  local host
  local a_file
  local aaaa_file
  local error_file

  case "${enabled}" in
    true|TRUE|1|yes|YES) ;;
    false|FALSE|0|no|NO|"")
      warn "${name} public DNS preflight skipped because PUBLIC_INGRESS_PUBLIC_DNS_ENABLED is not true"
      return 0
      ;;
    *) die "PUBLIC_INGRESS_PUBLIC_DNS_ENABLED must be true or false" ;;
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
    warn "public SSO/admission ingress preflight skipped because PUBLIC_INGRESS_PREFLIGHT_ENABLED is not true"
    return 0
  fi

  local admission_public_base_url
  admission_public_base_url="$(trim_trailing_slash "${ADMISSION_PUBLIC_BASE_URL:-}")"
  local sso_public_base_url
  sso_public_base_url="$(trim_trailing_slash "${CASDOOR_PUBLIC_AUTH_BASE_URL:-${WEB_VITE_SSO_URL:-${CASDOOR_ISSUER:-}}}")"

  require_public_dns_resolved "Web" "$(trim_trailing_slash "${WEB_PUBLIC_URL:-}")"
  require_public_dns_resolved "SSO" "${sso_public_base_url}"
  require_public_dns_resolved "Admission" "${admission_public_base_url}"
  require_public_http_reachable "Web" "$(trim_trailing_slash "${WEB_PUBLIC_URL:-}")"
  require_public_http_reachable "SSO" "${sso_public_base_url}"
  require_public_http_reachable "Admission" "${admission_public_base_url}/verify/__stuhelper_public_ingress_probe__"

  if [[ "${PUBLIC_INGRESS_CASDOOR_UPSTREAM_PREFLIGHT_ENABLED:-true}" == "true" ]]; then
    require_public_oidc_discovery "SSO" "${sso_public_base_url}"
    local sso_jwks_uri
    if ! sso_jwks_uri="$(public_oidc_jwks_uri "${sso_public_base_url}")"; then
      die "SSO JWKS URI preflight failed for ${sso_public_base_url}: discovery did not expose jwks_uri"
    fi
    require_public_jwks "SSO" "${sso_jwks_uri}"
  else
    log "SSO public OIDC metadata preflight skipped because PUBLIC_INGRESS_CASDOOR_UPSTREAM_PREFLIGHT_ENABLED is not true"
  fi
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

prefer_production_env_files_if_default() {
  local default_env_file="${REPO_ROOT}/.env"
  [[ "${ENV_FILE}" == "${default_env_file}" ]] || return 0
  [[ ! -f "${ENV_FILE}" ]] || return 0
  [[ -f "${REPO_ROOT}/.env.prod.shared" ]] || return 0

  export ENV_FILE="${REPO_ROOT}/.env.prod.shared"
  if [[ -z "${SECRETS_ENV_FILE:-}" ]]; then
    if [[ -f "${REPO_ROOT}/.env.prod.secrets.local" ]]; then
      export SECRETS_ENV_FILE="${REPO_ROOT}/.env.prod.secrets.local"
    elif [[ -f "${REPO_ROOT}/.env.prod.secrets" ]]; then
      export SECRETS_ENV_FILE="${REPO_ROOT}/.env.prod.secrets"
    fi
  fi
  if [[ "${GENERATED_ENV_FILE}" == "${REPO_ROOT}/.env.generated" ]]; then
    export GENERATED_ENV_FILE="${REPO_ROOT}/.env.prod.generated"
  fi
  if [[ "${GENERATED_SECRET_ENV_FILE}" == "${REPO_ROOT}/.env.generated.secrets" ]]; then
    export GENERATED_SECRET_ENV_FILE="${REPO_ROOT}/.env.prod.generated.secrets"
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
  if [[ -n "${GENERATED_ENV_SECRET_REF:-}" ]] && ! should_source_generated_secret_from_backend; then
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
  if [[ "${STUHELPER_PRESERVE_POSTGRES_URL_PLACEHOLDERS:-false}" != "true" ]]; then
    materialize_postgres_runtime_urls
  fi
  normalize_backup_object_storage_env
  export POSTGRES_WAL_ARCHIVE_VOLUME_NAME="${POSTGRES_WAL_ARCHIVE_VOLUME_NAME:-${STACK_NAME:-stuhelper}-postgres-wal-archive}"
  if [[ "${preserved_tag}" != "__STUHELPER_UNSET__" ]]; then export TAG="${preserved_tag}"; fi
  if [[ "${preserved_rollback_tag}" != "__STUHELPER_UNSET__" ]]; then export ROLLBACK_TAG="${preserved_rollback_tag}"; fi
  if [[ "${preserved_backend_image_ref}" != "__STUHELPER_UNSET__" ]]; then export BACKEND_IMAGE_REF="${preserved_backend_image_ref}"; fi
  if [[ "${preserved_frontend_image_ref}" != "__STUHELPER_UNSET__" ]]; then export FRONTEND_IMAGE_REF="${preserved_frontend_image_ref}"; fi
  if [[ "${preserved_admin_image_ref}" != "__STUHELPER_UNSET__" ]]; then export ADMIN_IMAGE_REF="${preserved_admin_image_ref}"; fi
  set +a
  clear_process_control_environment
}

load_env_preserving() {
  local -A preserved_values=()
  local key

  for key in "$@"; do
    [[ "${key}" =~ ^[A-Za-z_][A-Za-z0-9_]*$ ]] ||
      die "invalid environment key requested for preservation: ${key}"
    if [[ -v "${key}" ]]; then
      preserved_values["${key}"]="${!key}"
    fi
  done

  load_env

  for key in "${!preserved_values[@]}"; do
    printf -v "${key}" '%s' "${preserved_values[${key}]}"
    export "${key}"
  done
  materialize_postgres_runtime_urls
  clear_process_control_environment
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
  source_remote_deploy_config_env_file "${config_file}"
  if [[ "${preserved_tag}" != "__STUHELPER_UNSET__" ]]; then export TAG="${preserved_tag}"; fi
  if [[ "${preserved_rollback_tag}" != "__STUHELPER_UNSET__" ]]; then export ROLLBACK_TAG="${preserved_rollback_tag}"; fi
  if [[ "${preserved_backend_image_ref}" != "__STUHELPER_UNSET__" ]]; then export BACKEND_IMAGE_REF="${preserved_backend_image_ref}"; fi
  if [[ "${preserved_frontend_image_ref}" != "__STUHELPER_UNSET__" ]]; then export FRONTEND_IMAGE_REF="${preserved_frontend_image_ref}"; fi
  if [[ "${preserved_admin_image_ref}" != "__STUHELPER_UNSET__" ]]; then export ADMIN_IMAGE_REF="${preserved_admin_image_ref}"; fi
  if [[ "${preserved_registry_username}" != "__STUHELPER_UNSET__" ]]; then export REGISTRY_USERNAME="${preserved_registry_username}"; fi
  if [[ "${preserved_registry_password}" != "__STUHELPER_UNSET__" ]]; then export REGISTRY_PASSWORD="${preserved_registry_password}"; fi
  set +a
  clear_process_control_environment
}

configure_production_preflight_runtime_checks() {
  local preflight_phase="$1"
  local stack_name="${STACK_NAME:-stuhelper}"

  run_database_runtime_checks=true
  run_public_runtime_checks=true
  case "${preflight_phase}" in
    --pre-deploy | --timer-activation) ;;
    *) return 0 ;;
  esac

  run_public_runtime_checks=false
  warn "public application runtime checks are deferred until the mandatory post-deploy preflight"
  if [[ "${EXTERNAL_POSTGRES_ENABLED:-false}" != "true" ]] && \
    [[ "$(docker inspect --format '{{.State.Running}}' "${stack_name}-postgres" 2>/dev/null || true)" != "true" ]]; then
    run_database_runtime_checks=false
    warn "local PostgreSQL is not running; deferring database connectivity to mandatory post-deploy preflight"
  fi
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
    materialize_postgres_runtime_urls && \
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
  local auth_mode="${REGISTRY_AUTH_MODE:-persistent-secret}"
  [[ -n "${REGISTRY:-}" ]] || die "REGISTRY is required"

  case "${auth_mode}" in
    workflow-token)
      [[ "${CI_REGISTRY_LOGIN_READY:-false}" == "true" ]] ||
        die "workflow-token registry auth must be established by remote-ci-release.sh"
      ;;
    persistent-secret)
      resolve_registry_credentials
      [[ -n "${REGISTRY_USERNAME:-}" ]] || die "REGISTRY_USERNAME or REGISTRY_USERNAME_SECRET_REF is required"
      [[ -n "${REGISTRY_PASSWORD:-}" ]] || die "REGISTRY_PASSWORD or REGISTRY_PASSWORD_SECRET_REF is required"
      printf '%s\n' "${REGISTRY_PASSWORD}" |
        docker login "${REGISTRY}" --username "${REGISTRY_USERNAME}" --password-stdin >/dev/null
      ;;
    *)
      die "REGISTRY_AUTH_MODE must be workflow-token or persistent-secret"
      ;;
  esac
}

_release_record_operation() {
  local operation="$1"
  local tag="$2"
  case "${operation}" in
    check | publish) ;;
    *) die "unsupported release record operation: ${operation}" ;;
  esac
  require_safe_release_tag "${tag}"
  require_digest_image_ref BACKEND_IMAGE_REF "${BACKEND_IMAGE_REF:-}"
  require_digest_image_ref FRONTEND_IMAGE_REF "${FRONTEND_IMAGE_REF:-}"
  require_digest_image_ref ADMIN_IMAGE_REF "${ADMIN_IMAGE_REF:-}"
  local now
  now="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  python3 - \
    "${DEPLOY_STATE_DIR}" \
    "${operation}" \
    "${tag}" \
    "${now}" \
    "${BACKEND_IMAGE_REF:-}" \
    "${FRONTEND_IMAGE_REF:-}" \
    "${ADMIN_IMAGE_REF:-}" <<'PY'
import os
import re
import stat
import sys
import tempfile
from pathlib import Path

state_dir = Path(sys.argv[1])
operation = sys.argv[2]
tag, deployed_at, backend_ref, frontend_ref, admin_ref = sys.argv[3:]
if operation not in {"check", "publish"}:
    raise SystemExit(f"unsupported release record operation: {operation}")
values = {
    "TAG": tag,
    "DEPLOYED_AT": deployed_at,
    "BACKEND_IMAGE_REF": backend_ref,
    "FRONTEND_IMAGE_REF": frontend_ref,
    "ADMIN_IMAGE_REF": admin_ref,
}
digest_ref_pattern = re.compile(r"^[^\s@]+@sha256:[0-9a-f]{64}$")
legacy_ref_pattern = re.compile(
    r"^[A-Za-z0-9][A-Za-z0-9._-]*(?::[0-9]+)?"
    r"(?:/[A-Za-z0-9][A-Za-z0-9._-]*)*"
    r":[A-Za-z0-9_][A-Za-z0-9._-]{0,127}$",
)

if not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9._-]{0,127}", tag):
    raise SystemExit("release tag contains unsafe path characters")
for key, value in values.items():
    if not value:
        raise SystemExit(f"release record field {key} must not be empty")
    if "\n" in value or "\r" in value:
        raise SystemExit(f"release record field {key} must be a single line")
for key in ("BACKEND_IMAGE_REF", "FRONTEND_IMAGE_REF", "ADMIN_IMAGE_REF"):
    if not digest_ref_pattern.fullmatch(values[key]):
        raise SystemExit(f"release record field {key} must be a complete image@sha256 digest reference")

releases_dir = state_dir / "releases"
candidate_payload = "".join(f"{key}={value}\n" for key, value in values.items()).encode()


def fsync_directory(path: Path) -> None:
    directory_fd = os.open(path, os.O_RDONLY | os.O_DIRECTORY)
    try:
        os.fsync(directory_fd)
    finally:
        os.close(directory_fd)


def stage_payload(path: Path, payload: bytes) -> Path:
    fd, temporary_name = tempfile.mkstemp(prefix=f".{path.name}.", dir=path.parent)
    temporary_path = Path(temporary_name)
    try:
        os.fchmod(fd, 0o600)
        with os.fdopen(fd, "wb") as stream:
            stream.write(payload)
            stream.flush()
            os.fsync(stream.fileno())
        return temporary_path
    except BaseException:
        try:
            os.close(fd)
        except OSError:
            pass
        temporary_path.unlink(missing_ok=True)
        raise


def atomic_write(path: Path, payload: bytes) -> None:
    temporary_path = stage_payload(path, payload)
    try:
        os.replace(temporary_path, path)
        fsync_directory(path.parent)
    except BaseException:
        temporary_path.unlink(missing_ok=True)
        raise


def read_canonical_release(
    path: Path,
    *,
    allow_legacy_image_refs: bool = False,
) -> tuple[bytes, dict[str, str]]:
    flags = os.O_RDONLY
    if hasattr(os, "O_NOFOLLOW"):
        flags |= os.O_NOFOLLOW
    fd = os.open(path, flags)
    try:
        metadata = os.fstat(fd)
        if not stat.S_ISREG(metadata.st_mode):
            raise SystemExit(f"existing immutable release record is not a regular file: {path}")
        if metadata.st_uid != os.geteuid():
            raise SystemExit(f"existing immutable release record must be owned by the deploy user: {path}")
        if stat.S_IMODE(metadata.st_mode) != 0o600:
            raise SystemExit(f"existing immutable release record must use mode 0600: {path}")
        with os.fdopen(fd, "rb", closefd=False) as stream:
            payload = stream.read()
    finally:
        os.close(fd)

    try:
        text = payload.decode("utf-8")
    except UnicodeDecodeError as exc:
        raise SystemExit(f"existing immutable release record is not UTF-8: {path}") from exc
    if not text.endswith("\n"):
        raise SystemExit(f"existing immutable release record is truncated: {path}")
    lines = text.splitlines()
    if len(lines) != len(values):
        raise SystemExit(f"existing immutable release record is incomplete: {path}")

    existing = {}
    for line in lines:
        if "=" not in line:
            raise SystemExit(f"existing immutable release record is malformed: {path}")
        key, value = line.split("=", 1)
        if key not in values or key in existing or not value:
            raise SystemExit(f"existing immutable release record is malformed: {path}")
        existing[key] = value
    if list(existing) != list(values):
        raise SystemExit(f"existing immutable release record has an unexpected field order: {path}")
    if not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9._-]{0,127}", existing["TAG"]):
        raise SystemExit(f"existing immutable release record has an unsafe TAG: {path}")
    if not re.fullmatch(r"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z", existing["DEPLOYED_AT"]):
        raise SystemExit(f"existing immutable release record has an invalid DEPLOYED_AT: {path}")
    for key in ("BACKEND_IMAGE_REF", "FRONTEND_IMAGE_REF", "ADMIN_IMAGE_REF"):
        is_digest = digest_ref_pattern.fullmatch(existing[key]) is not None
        is_supported_legacy = (
            allow_legacy_image_refs
            and legacy_ref_pattern.fullmatch(existing[key]) is not None
        )
        if not is_digest and not is_supported_legacy:
            raise SystemExit(f"existing immutable release record has an invalid {key}: {path}")
    canonical_payload = "".join(f"{key}={existing[key]}\n" for key in values).encode()
    if payload != canonical_payload:
        raise SystemExit(f"existing immutable release record is not canonical: {path}")
    return payload, existing


def read_existing_immutable_release(path: Path) -> bytes:
    payload, existing = read_canonical_release(path)
    for key in ("TAG", "BACKEND_IMAGE_REF", "FRONTEND_IMAGE_REF", "ADMIN_IMAGE_REF"):
        if existing[key] != values[key]:
            raise SystemExit(
                f"release field {key} does not match existing immutable release record: {path}",
            )
    return payload


def read_release_log_tags(path: Path) -> set[str]:
    flags = os.O_RDONLY
    if hasattr(os, "O_NOFOLLOW"):
        flags |= os.O_NOFOLLOW
    fd = os.open(path, flags)
    try:
        metadata = os.fstat(fd)
        if not stat.S_ISREG(metadata.st_mode):
            raise SystemExit(f"release activation log is not a regular file: {path}")
        if metadata.st_uid != os.geteuid():
            raise SystemExit(f"release activation log must be owned by the deploy user: {path}")
        if stat.S_IMODE(metadata.st_mode) != 0o600:
            raise SystemExit(f"release activation log must use mode 0600: {path}")
        with os.fdopen(fd, "rb", closefd=False) as stream:
            payload = stream.read()
    finally:
        os.close(fd)

    try:
        text = payload.decode("utf-8")
    except UnicodeDecodeError as exc:
        raise SystemExit(f"release activation log is not UTF-8: {path}") from exc
    if text and not text.endswith("\n"):
        raise SystemExit(f"release activation log is truncated: {path}")

    tags = set()
    for line in text.splitlines():
        fields = line.split("\t")
        if len(fields) != 2:
            raise SystemExit(f"release activation log is malformed: {path}")
        deployed_at, logged_tag = fields
        if not re.fullmatch(r"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z", deployed_at):
            raise SystemExit(f"release activation log has an invalid timestamp: {path}")
        if not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9._-]{0,127}", logged_tag):
            raise SystemExit(f"release activation log has an invalid tag: {path}")
        tags.add(logged_tag)
    return tags


def publish_immutable_release(path: Path, payload: bytes) -> bytes:
    temporary_path = stage_payload(path, payload)
    try:
        try:
            os.link(temporary_path, path)
        except FileExistsError:
            return read_existing_immutable_release(path)
        fsync_directory(path.parent)
        return payload
    finally:
        temporary_path.unlink(missing_ok=True)


# The pre-deploy check is read-only. A genuinely unused tag is available; an
# existing immutable record must describe exactly the requested image identity.
# Surviving current/log evidence makes a missing per-tag record ledger damage,
# not permission to reuse the tag.
release_path = releases_dir / f"{tag}.env"
current_release_path = state_dir / "current-release.env"
try:
    current_payload, current_release = read_canonical_release(current_release_path)
except FileNotFoundError:
    current_payload = None
    current_release = None
if current_release is not None:
    current_immutable_path = releases_dir / f"{current_release['TAG']}.env"
    try:
        current_immutable_payload, _ = read_canonical_release(current_immutable_path)
    except FileNotFoundError:
        raise SystemExit(
            f"release tag {current_release['TAG']} was previously used but its immutable record is missing; "
            f"evidence: {current_release_path}",
        )
    if current_payload != current_immutable_payload:
        raise SystemExit(
            f"current release pointer does not match its immutable per-tag record: {current_release_path}",
        )

release_log_path = state_dir / "releases.log"
try:
    logged_tags = read_release_log_tags(release_log_path)
except FileNotFoundError:
    logged_tags = set()

# releases.log is the activation ledger, not merely a syntactic index. Every
# referenced tag must still have a structurally canonical immutable record.
# Historical tag-only records remain readable for migration/audit purposes,
# but the current pointer and every newly published candidate stay digest-only.
for logged_tag in sorted(logged_tags):
    logged_release_path = releases_dir / f"{logged_tag}.env"
    try:
        _, logged_release = read_canonical_release(
            logged_release_path,
            allow_legacy_image_refs=True,
        )
    except FileNotFoundError as exc:
        raise SystemExit(
            f"release tag {logged_tag} was previously used but its immutable record is missing; "
            f"evidence: {release_log_path}",
        ) from exc
    if logged_release["TAG"] != logged_tag:
        raise SystemExit(
            f"release activation log tag {logged_tag} does not match its immutable record: "
            f"{logged_release_path}",
        )

# Validate the reverse edge as well: an immutable per-tag record is release
# history, so it must not silently disappear from releases.log. The only
# recoverable exception is the exact candidate being retried; publication then
# appends its missing activation before changing the current pointer.
immutable_tags = set()
try:
    releases_metadata = releases_dir.lstat()
except FileNotFoundError:
    releases_metadata = None
if releases_metadata is not None:
    if not stat.S_ISDIR(releases_metadata.st_mode) or releases_dir.is_symlink():
        raise SystemExit(f"release record path must be a regular directory: {releases_dir}")
    tag_record_pattern = re.compile(r"([A-Za-z0-9][A-Za-z0-9._-]{0,127})\.env")
    with os.scandir(releases_dir) as entries:
        for entry in entries:
            if not entry.name.endswith(".env"):
                continue
            match = tag_record_pattern.fullmatch(entry.name)
            if match is None:
                raise SystemExit(f"release record has an unsafe filename: {entry.path}")
            immutable_tag = match.group(1)
            immutable_path = Path(entry.path)
            _, immutable_release = read_canonical_release(
                immutable_path,
                allow_legacy_image_refs=True,
            )
            if immutable_release["TAG"] != immutable_tag:
                raise SystemExit(
                    f"immutable release filename tag {immutable_tag} does not match its record: "
                    f"{immutable_path}",
                )
            immutable_tags.add(immutable_tag)

unlogged_immutable_tags = sorted(immutable_tags - logged_tags)
blocking_unlogged_tags = [unlogged_tag for unlogged_tag in unlogged_immutable_tags if unlogged_tag != tag]
if blocking_unlogged_tags:
    unlogged_tag = blocking_unlogged_tags[0]
    raise SystemExit(
        f"immutable release tag {unlogged_tag} is missing from the activation log: "
        f"{release_log_path}; retry that exact release identity before deploying another tag",
    )

try:
    read_existing_immutable_release(release_path)
except FileNotFoundError:
    prior_evidence = []
    if current_release is not None and current_release["TAG"] == tag:
        prior_evidence.append(str(current_release_path))
    if tag in logged_tags:
        prior_evidence.append(str(release_log_path))

    if prior_evidence:
        evidence = ", ".join(prior_evidence)
        raise SystemExit(
            f"release tag {tag} was previously used but its immutable record is missing; evidence: {evidence}",
        )

# Older publishers wrote current-release.env before releases.log. If they were
# interrupted between those writes, only an exact retry of that same immutable
# identity may proceed and complete the activation ledger. A different tag
# remains blocked so it cannot overwrite the sole unlogged-current evidence.
if current_release is not None and current_release["TAG"] not in logged_tags:
    if current_release["TAG"] != tag:
        raise SystemExit(
            f"current release tag {current_release['TAG']} is missing from the activation log: "
            f"{release_log_path}; retry that exact release identity before deploying another tag",
        )

if operation == "check":
    raise SystemExit(0)

state_dir.mkdir(parents=True, mode=0o700, exist_ok=True)
releases_dir.mkdir(mode=0o700, exist_ok=True)
releases_metadata = releases_dir.lstat()
if not stat.S_ISDIR(releases_metadata.st_mode) or releases_dir.is_symlink():
    raise SystemExit(f"release record path must be a regular directory: {releases_dir}")
if releases_metadata.st_uid != os.geteuid():
    raise SystemExit(f"release record directory must be owned by the deploy user: {releases_dir}")
if stat.S_IMODE(releases_metadata.st_mode) != 0o700:
    os.chmod(releases_dir, 0o700)
    fsync_directory(state_dir)

# Link the per-release record into place without replacement. Reusing a tag is
# allowed only for the same complete release identity; its original timestamp
# remains immutable while releases.log records each later activation.
release_payload = publish_immutable_release(release_path, candidate_payload)

log_path = state_dir / "releases.log"
log_fd = os.open(log_path, os.O_WRONLY | os.O_CREAT | os.O_APPEND, 0o600)
os.fchmod(log_fd, 0o600)
with os.fdopen(log_fd, "ab") as stream:
    stream.write(f"{deployed_at}\t{tag}\n".encode())
    stream.flush()
    os.fsync(stream.fileno())
fsync_directory(state_dir)

# The activation ledger becomes durable before the replaceable current
# pointer. A termination after the log fsync therefore leaves a retryable
# logged release, never an unlogged current pointer.
atomic_write(state_dir / "current-release.env", release_payload)
PY
}

require_release_tag_identity_available() {
  _release_record_operation check "$1"
}

record_release() {
  _release_record_operation publish "$1"
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
