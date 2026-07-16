#!/usr/bin/env bash
set -euo pipefail

if [[ "$(id -u)" != "0" ]]; then
  echo "[install-baota-runtime-permission-guard][error] run as root" >&2
  exit 1
fi

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
  echo "[install-baota-runtime-permission-guard][error] missing ${permission_source}" >&2
  exit 1
}
[[ -f "${guard_source}" ]] || {
  echo "[install-baota-runtime-permission-guard][error] missing ${guard_source}" >&2
  exit 1
}
[[ -d "${SOURCE_DIR}" ]] || {
  echo "[install-baota-runtime-permission-guard][error] source dir not found: ${SOURCE_DIR}" >&2
  exit 1
}

install -o root -g root -m 0755 "${permission_source}" "${permission_bin}"
install -o root -g root -m 0755 "${guard_source}" "${guard_bin}"

cat >"${service_file}" <<EOF
[Unit]
Description=Protect and normalize StuHelper Baota bind-mount permissions
After=local-fs.target
Before=docker.service bt.service

[Service]
Type=oneshot
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

echo "[install-baota-runtime-permission-guard] installed ${UNIT_NAME}.service and ${UNIT_NAME}.timer"
echo "[install-baota-runtime-permission-guard] Docker and Baota were not restarted"
