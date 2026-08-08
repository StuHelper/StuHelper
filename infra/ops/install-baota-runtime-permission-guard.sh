#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: infra/ops/install-baota-runtime-permission-guard.sh [--verify-installed]

With no arguments, install or refresh the root-owned permission guard. This
mode must run as root. --verify-installed is read-only and may run as the
non-root deploy user; it fails when the installed guard or its persistent
systemd configuration differs from the current release.
USAGE
}

die() {
  echo "[install-baota-runtime-permission-guard][error] $*" >&2
  exit 1
}

verify_installed=false
case "${1:-}" in
  "") ;;
  --verify-installed) verify_installed=true ;;
  -h | --help) usage; exit 0 ;;
  *) usage >&2; die "unknown option: $1" ;;
esac
[[ "$#" -le 1 ]] || {
  usage >&2
  die "too many arguments"
}

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SOURCE_DIR="${BAOTA_SOURCE_DIR:-/www/server/panel/data/compose/stuhelper/source}"
CASDOOR_ROOT="${CASDOOR_COMPOSE_ROOT:-/www/server/panel/data/compose/casdoor}"
OPENLIST_DIR="${OPENLIST_DATA_DIR:-/opt/openlist/data}"
UNIT_NAME="${RUNTIME_PERMISSION_UNIT_NAME:-stuhelper-runtime-permissions}"

permission_source="${SCRIPT_DIR}/ensure-baota-runtime-permissions.sh"
guard_source="${SCRIPT_DIR}/guard-baota-files-set-mode.sh"
permission_bin="/usr/local/sbin/stuhelper-runtime-permissions"
guard_bin="/usr/local/sbin/stuhelper-baota-files-set-mode-guard"
service_file="/etc/systemd/system/${UNIT_NAME}.service"
timer_file="/etc/systemd/system/${UNIT_NAME}.timer"
bt_dropin_dir="/etc/systemd/system/bt.service.d"
docker_dropin_dir="/etc/systemd/system/docker.service.d"
tmpfiles_file="/etc/tmpfiles.d/stuhelper-baota-permission-guard.conf"

[[ -f "${permission_source}" ]] || {
  die "missing ${permission_source}"
}
[[ -f "${guard_source}" ]] || {
  die "missing ${guard_source}"
}
[[ -d "${SOURCE_DIR}" ]] || {
  die "source dir not found: ${SOURCE_DIR}"
}
SOURCE_DIR="$(cd "${SOURCE_DIR}" && pwd)"

generated_observability_owner_uid="${GENERATED_OBSERVABILITY_CONFIG_OWNER_UID:-$(stat -c '%u' "${SOURCE_DIR}")}"
generated_observability_group_gid="${ALERTMANAGER_CONFIG_GID:-65534}"
[[ "${generated_observability_owner_uid}" =~ ^[0-9]+$ ]] ||
  die "GENERATED_OBSERVABILITY_CONFIG_OWNER_UID must be a numeric UID"
[[ "${generated_observability_group_gid}" =~ ^[0-9]+$ ]] ||
  die "ALERTMANAGER_CONFIG_GID must be a numeric GID"

verify_installed_guard() {
  local effective_environment
  local effective_exec_start
  local effective_exec_start_pre
  local expected_exec_start
  local expected_entry
  local installed_entry
  local matched
  local -a expected_environment=(
    "GENERATED_OBSERVABILITY_CONFIG_OWNER_UID=${generated_observability_owner_uid}"
    "ALERTMANAGER_CONFIG_GID=${generated_observability_group_gid}"
  )
  local -a installed_environment=()

  [[ -x "${permission_bin}" ]] || die "installed permission normalizer is missing or not executable: ${permission_bin}"
  [[ -x "${guard_bin}" ]] || die "installed Baota marker guard is missing or not executable: ${guard_bin}"
  cmp --silent "${permission_source}" "${permission_bin}" ||
    die "installed permission normalizer is stale; rerun this installer as root before deploying"
  cmp --silent "${guard_source}" "${guard_bin}" ||
    die "installed Baota marker guard is stale; rerun this installer as root before deploying"

  systemctl cat "${UNIT_NAME}.service" >/dev/null 2>&1 ||
    die "${UNIT_NAME}.service is not installed"
  systemctl cat "${UNIT_NAME}.timer" >/dev/null 2>&1 ||
    die "${UNIT_NAME}.timer is not installed"

  effective_environment="$(systemctl show "${UNIT_NAME}.service" --property=Environment --value)" ||
    die "failed to inspect ${UNIT_NAME}.service environment"
  read -r -a installed_environment <<<"${effective_environment}"
  [[ "${#installed_environment[@]}" -eq "${#expected_environment[@]}" ]] ||
    die "${UNIT_NAME}.service has stale generated-observability ownership settings; rerun this installer as root before deploying"
  for expected_entry in "${expected_environment[@]}"; do
    matched=false
    for installed_entry in "${installed_environment[@]}"; do
      if [[ "${installed_entry}" == "${expected_entry}" ]]; then
        matched=true
        break
      fi
    done
    [[ "${matched}" == "true" ]] ||
      die "${UNIT_NAME}.service has stale generated-observability ownership settings; rerun this installer as root before deploying"
  done

  effective_exec_start="$(systemctl show "${UNIT_NAME}.service" --property=ExecStart --value)" ||
    die "failed to inspect ${UNIT_NAME}.service ExecStart"
  expected_exec_start="${permission_bin} --source-dir ${SOURCE_DIR} --casdoor-compose-root ${CASDOOR_ROOT} --openlist-data-dir ${OPENLIST_DIR} --apply"
  [[ "${effective_exec_start}" == *"argv[]=${expected_exec_start}"* ]] ||
    die "${UNIT_NAME}.service targets a stale source or runtime path; rerun this installer as root before deploying"

  effective_exec_start_pre="$(systemctl show "${UNIT_NAME}.service" --property=ExecStartPre --value)" ||
    die "failed to inspect ${UNIT_NAME}.service ExecStartPre"
  [[ "${effective_exec_start_pre}" == *"argv[]=${guard_bin}"* ]] ||
    die "${UNIT_NAME}.service does not use the installed Baota marker guard"

  systemctl is-enabled --quiet "${UNIT_NAME}.service" ||
    die "${UNIT_NAME}.service is not enabled"
  systemctl is-enabled --quiet "${UNIT_NAME}.timer" ||
    die "${UNIT_NAME}.timer is not enabled"
  systemctl is-active --quiet "${UNIT_NAME}.timer" ||
    die "${UNIT_NAME}.timer is not active"

  echo "[install-baota-runtime-permission-guard] installed guard matches the current release and persistent ownership settings"
}

if [[ "${verify_installed}" == "true" ]]; then
  verify_installed_guard
  exit 0
fi

if [[ "$(id -u)" != "0" ]]; then
  die "run as root"
fi

install -o root -g root -m 0755 "${permission_source}" "${permission_bin}"
install -o root -g root -m 0755 "${guard_source}" "${guard_bin}"

cat >"${service_file}" <<EOF
[Unit]
Description=Protect and normalize StuHelper Baota bind-mount permissions
After=local-fs.target
Before=docker.service bt.service

[Service]
Type=oneshot
Environment=GENERATED_OBSERVABILITY_CONFIG_OWNER_UID=${generated_observability_owner_uid}
Environment=ALERTMANAGER_CONFIG_GID=${generated_observability_group_gid}
ExecStartPre=${guard_bin}
ExecStart=${permission_bin} --source-dir ${SOURCE_DIR} --casdoor-compose-root ${CASDOOR_ROOT} --openlist-data-dir ${OPENLIST_DIR} --apply
TimeoutStartSec=5min

[Install]
WantedBy=multi-user.target
EOF

cat >"${timer_file}" <<EOF
[Unit]
Description=Periodically verify StuHelper Baota bind-mount permissions

[Timer]
OnBootSec=5min
OnUnitActiveSec=6h
AccuracySec=5min
RandomizedDelaySec=2min
Unit=${UNIT_NAME}.service

[Install]
WantedBy=timers.target
EOF

mkdir -p "${bt_dropin_dir}" "${docker_dropin_dir}"
cat >"${bt_dropin_dir}/10-stuhelper-permission-guard.conf" <<EOF
[Unit]
Wants=${UNIT_NAME}.service
After=${UNIT_NAME}.service

[Service]
ExecStartPre=${guard_bin}
EOF

cat >"${docker_dropin_dir}/10-stuhelper-runtime-permissions.conf" <<EOF
[Unit]
Wants=${UNIT_NAME}.service
After=${UNIT_NAME}.service
EOF

cat >"${tmpfiles_file}" <<'EOF'
f /tmp/last_files_set_mode.pl 0600 root root -
EOF

chmod 0644 \
  "${service_file}" \
  "${timer_file}" \
  "${bt_dropin_dir}/10-stuhelper-permission-guard.conf" \
  "${docker_dropin_dir}/10-stuhelper-runtime-permissions.conf" \
  "${tmpfiles_file}"

systemd-tmpfiles --create "${tmpfiles_file}"
systemctl daemon-reload
systemctl enable "${UNIT_NAME}.service" "${UNIT_NAME}.timer"
systemctl start "${UNIT_NAME}.service"
systemctl start "${UNIT_NAME}.timer"

verify_installed_guard

echo "[install-baota-runtime-permission-guard] installed ${UNIT_NAME}.service and ${UNIT_NAME}.timer"
echo "[install-baota-runtime-permission-guard] Docker and Baota were not restarted"
