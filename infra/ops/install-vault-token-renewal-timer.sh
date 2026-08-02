#!/usr/bin/env bash
set -euo pipefail

DEPLOY_USER="${VAULT_RUNTIME_TOKEN_OWNER:-${DEPLOY_USER:-stuhelper}}"
DEPLOY_GROUP="${VAULT_RUNTIME_TOKEN_GROUP:-${DEPLOY_GROUP:-${DEPLOY_USER}}}"
DEPLOY_APP_DIR="${DEPLOY_APP_DIR:-/opt/stuhelper}"
REMOTE_DEPLOY_CONFIG_FILE="${REMOTE_DEPLOY_CONFIG_FILE:-${DEPLOY_APP_DIR}/.deploy/remote.env}"
SYSTEMD_PREFIX="${SYSTEMD_PREFIX:-stuhelper}"

service_unit="${SYSTEMD_PREFIX}-vault-token-renewal.service"
timer_unit="${SYSTEMD_PREFIX}-vault-token-renewal.timer"
service_file="/etc/systemd/system/${service_unit}"
timer_file="/etc/systemd/system/${timer_unit}"

die() {
  echo "[install-vault-token-renewal-timer][error] $*" >&2
  exit 1
}

[[ "${EUID}" -eq 0 ]] || die "run as root"
command -v systemctl >/dev/null 2>&1 || die "systemctl is required"
id -u "${DEPLOY_USER}" >/dev/null 2>&1 || die "deploy user does not exist: ${DEPLOY_USER}"
getent group "${DEPLOY_GROUP}" >/dev/null 2>&1 || die "deploy group does not exist: ${DEPLOY_GROUP}"
[[ -x "${DEPLOY_APP_DIR}/infra/ops/vault-runtime-token.sh" ]] ||
  die "Vault runtime token script is not executable under ${DEPLOY_APP_DIR}"
[[ -f "${REMOTE_DEPLOY_CONFIG_FILE}" ]] ||
  die "remote deploy config does not exist: ${REMOTE_DEPLOY_CONFIG_FILE}"

cat >"${service_file}" <<EOF
[Unit]
Description=Renew and verify the StuHelper production Vault runtime token
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
User=${DEPLOY_USER}
Group=${DEPLOY_GROUP}
UMask=0077
WorkingDirectory=${DEPLOY_APP_DIR}
Environment=REMOTE_DEPLOY_CONFIG_FILE=${REMOTE_DEPLOY_CONFIG_FILE}
ExecStart=${DEPLOY_APP_DIR}/infra/ops/vault-runtime-token.sh renew
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectKernelLogs=true
ProtectControlGroups=true
RestrictSUIDSGID=true
RestrictRealtime=true
LockPersonality=true
MemoryDenyWriteExecute=true
CapabilityBoundingSet=
RestrictAddressFamilies=AF_INET AF_INET6

[Install]
WantedBy=multi-user.target
EOF

cat >"${timer_file}" <<EOF
[Unit]
Description=Renew the StuHelper production Vault runtime token every 12 hours

[Timer]
OnBootSec=10min
OnUnitInactiveSec=12h
RandomizedDelaySec=10min
AccuracySec=1min
Unit=${service_unit}

[Install]
WantedBy=timers.target
EOF

chmod 0644 "${service_file}" "${timer_file}"
systemctl daemon-reload
systemctl enable --now "${timer_unit}"
systemctl start "${service_unit}"

echo "[install-vault-token-renewal-timer] installed and verified:"
echo "  - ${service_file}"
echo "  - ${timer_file}"
