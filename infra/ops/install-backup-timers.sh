#!/usr/bin/env bash
set -euo pipefail

DEPLOY_USER="${DEPLOY_USER:-stuhelper}"
DEPLOY_GROUP="${DEPLOY_GROUP:-${DEPLOY_USER}}"
DEPLOY_APP_DIR="${DEPLOY_APP_DIR:-/opt/stuhelper}"
SYSTEMD_PREFIX="${SYSTEMD_PREFIX:-stuhelper}"
BACKUP_STAGING_DIR="${BACKUP_STAGING_DIR:-/var/lib/stuhelper/postgres/backup-staging}"

dump_service="/etc/systemd/system/${SYSTEMD_PREFIX}-postgres-dump-backup.service"
dump_timer="/etc/systemd/system/${SYSTEMD_PREFIX}-postgres-dump-backup.timer"
base_service="/etc/systemd/system/${SYSTEMD_PREFIX}-postgres-basebackup.service"
base_timer="/etc/systemd/system/${SYSTEMD_PREFIX}-postgres-basebackup.timer"
sync_service="/etc/systemd/system/${SYSTEMD_PREFIX}-postgres-backup-sync.service"
sync_timer="/etc/systemd/system/${SYSTEMD_PREFIX}-postgres-backup-sync.timer"

require_root() {
  if [[ "${EUID}" -ne 0 ]]; then
    echo "[install-backup-timers][error] run as root" >&2
    exit 1
  fi
}

main() {
  require_root

  install -d -o "${DEPLOY_USER}" -g "${DEPLOY_GROUP}" -m 0755 "${DEPLOY_APP_DIR}/backups/postgres/logical"
  install -d -o "${DEPLOY_USER}" -g "${DEPLOY_GROUP}" -m 0755 "${DEPLOY_APP_DIR}/backups/postgres/base"
  install -d -o "${DEPLOY_USER}" -g "${DEPLOY_GROUP}" -m 0700 "${BACKUP_STAGING_DIR}"
  install -d -o "${DEPLOY_USER}" -g "${DEPLOY_GROUP}" -m 0755 "/var/lib/stuhelper/postgres/wal-restore"

  cat >"${dump_service}" <<EOF
[Unit]
Description=StuHelper PostgreSQL logical backup
After=docker.service network-online.target
Wants=network-online.target

[Service]
Type=oneshot
RemainAfterExit=no
Restart=no
TimeoutStartSec=4h
TimeoutStopSec=2min
KillMode=control-group
SendSIGKILL=yes
User=${DEPLOY_USER}
Group=${DEPLOY_GROUP}
WorkingDirectory=${DEPLOY_APP_DIR}
Environment=ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.shared
Environment=SECRETS_ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.secrets
Environment=GENERATED_ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.generated
Environment=GENERATED_SECRET_ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.generated.secrets
Environment=LOCAL_STATE_DIR=/var/lib/stuhelper
Environment=BACKUP_STAGING_DIR=${BACKUP_STAGING_DIR}
Environment=BACKUP_OBJECT_STORAGE_OFF_HOST_REQUIRED=true
UnsetEnvironment=LD_PRELOAD LD_LIBRARY_PATH LD_AUDIT GCONV_PATH LOCPATH
ExecStart=/usr/bin/env -i PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.shared SECRETS_ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.secrets GENERATED_ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.generated GENERATED_SECRET_ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.generated.secrets LOCAL_STATE_DIR=/var/lib/stuhelper BACKUP_STAGING_DIR=${BACKUP_STAGING_DIR} BACKUP_OBJECT_STORAGE_OFF_HOST_REQUIRED=true /bin/bash --noprofile --norc ./infra/ops/run-scheduled-backup.sh dump
EOF

  cat >"${dump_timer}" <<EOF
[Unit]
Description=StuHelper PostgreSQL logical backup timer

[Timer]
Unit=${SYSTEMD_PREFIX}-postgres-dump-backup.service
OnCalendar=*-*-* 03:15:00
Persistent=true
AccuracySec=1min
RandomizedDelaySec=0
FixedRandomDelay=false

[Install]
WantedBy=timers.target
EOF

  cat >"${base_service}" <<EOF
[Unit]
Description=StuHelper PostgreSQL base backup
After=docker.service network-online.target
Wants=network-online.target

[Service]
Type=oneshot
RemainAfterExit=no
Restart=no
TimeoutStartSec=12h
TimeoutStopSec=2min
KillMode=control-group
SendSIGKILL=yes
User=${DEPLOY_USER}
Group=${DEPLOY_GROUP}
WorkingDirectory=${DEPLOY_APP_DIR}
Environment=ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.shared
Environment=SECRETS_ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.secrets
Environment=GENERATED_ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.generated
Environment=GENERATED_SECRET_ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.generated.secrets
Environment=LOCAL_STATE_DIR=/var/lib/stuhelper
Environment=BACKUP_STAGING_DIR=${BACKUP_STAGING_DIR}
Environment=BACKUP_OBJECT_STORAGE_OFF_HOST_REQUIRED=true
UnsetEnvironment=LD_PRELOAD LD_LIBRARY_PATH LD_AUDIT GCONV_PATH LOCPATH
ExecStart=/usr/bin/env -i PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.shared SECRETS_ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.secrets GENERATED_ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.generated GENERATED_SECRET_ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.generated.secrets LOCAL_STATE_DIR=/var/lib/stuhelper BACKUP_STAGING_DIR=${BACKUP_STAGING_DIR} BACKUP_OBJECT_STORAGE_OFF_HOST_REQUIRED=true /bin/bash --noprofile --norc ./infra/ops/run-scheduled-backup.sh basebackup
EOF

  cat >"${base_timer}" <<EOF
[Unit]
Description=StuHelper PostgreSQL base backup timer

[Timer]
Unit=${SYSTEMD_PREFIX}-postgres-basebackup.service
OnCalendar=Sun *-*-* 03:45:00
Persistent=true
AccuracySec=1min
RandomizedDelaySec=0
FixedRandomDelay=false

[Install]
WantedBy=timers.target
EOF


  cat >"${sync_service}" <<EOF
[Unit]
Description=StuHelper PostgreSQL backup artifact sync
After=docker.service network-online.target
Wants=network-online.target

[Service]
Type=oneshot
RemainAfterExit=no
Restart=no
TimeoutStartSec=10min
TimeoutStopSec=2min
KillMode=control-group
SendSIGKILL=yes
User=${DEPLOY_USER}
Group=${DEPLOY_GROUP}
WorkingDirectory=${DEPLOY_APP_DIR}
Environment=ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.shared
Environment=SECRETS_ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.secrets
Environment=GENERATED_ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.generated
Environment=GENERATED_SECRET_ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.generated.secrets
Environment=LOCAL_STATE_DIR=/var/lib/stuhelper
Environment=BACKUP_OBJECT_STORAGE_OFF_HOST_REQUIRED=true
UnsetEnvironment=LD_PRELOAD LD_LIBRARY_PATH LD_AUDIT GCONV_PATH LOCPATH
ExecStart=/usr/bin/env -i PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.shared SECRETS_ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.secrets GENERATED_ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.generated GENERATED_SECRET_ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.generated.secrets LOCAL_STATE_DIR=/var/lib/stuhelper BACKUP_OBJECT_STORAGE_OFF_HOST_REQUIRED=true /bin/bash --noprofile --norc ./infra/ops/sync-postgres-backups.sh
EOF

  cat >"${sync_timer}" <<EOF
[Unit]
Description=StuHelper PostgreSQL backup artifact sync timer

[Timer]
Unit=${SYSTEMD_PREFIX}-postgres-backup-sync.service
OnCalendar=*-*-* *:00/15:00
Persistent=true
AccuracySec=1min
RandomizedDelaySec=0
FixedRandomDelay=false

[Install]
WantedBy=timers.target
EOF

  systemctl daemon-reload
  systemctl enable --now "${SYSTEMD_PREFIX}-postgres-dump-backup.timer" "${SYSTEMD_PREFIX}-postgres-basebackup.timer" "${SYSTEMD_PREFIX}-postgres-backup-sync.timer"

  echo "[install-backup-timers] installed:"
  echo "  - ${dump_service}"
  echo "  - ${dump_timer}"
  echo "  - ${base_service}"
  echo "  - ${base_timer}"
  echo "  - ${sync_service}"
  echo "  - ${sync_timer}"
}

main "$@"
