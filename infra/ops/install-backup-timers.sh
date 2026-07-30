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
User=${DEPLOY_USER}
Group=${DEPLOY_GROUP}
WorkingDirectory=${DEPLOY_APP_DIR}
Environment=ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.shared
Environment=SECRETS_ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.secrets
Environment=GENERATED_ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.generated
Environment=GENERATED_SECRET_ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.generated.secrets
Environment=BACKUP_STAGING_DIR=${BACKUP_STAGING_DIR}
ExecStart=/bin/bash -lc 'cd "${DEPLOY_APP_DIR}" && ./infra/ops/run-scheduled-backup.sh dump'
EOF

  cat >"${dump_timer}" <<EOF
[Unit]
Description=StuHelper PostgreSQL logical backup timer

[Timer]
OnCalendar=*-*-* 03:15:00
Persistent=true

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
User=${DEPLOY_USER}
Group=${DEPLOY_GROUP}
WorkingDirectory=${DEPLOY_APP_DIR}
Environment=ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.shared
Environment=SECRETS_ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.secrets
Environment=GENERATED_ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.generated
Environment=GENERATED_SECRET_ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.generated.secrets
Environment=BACKUP_STAGING_DIR=${BACKUP_STAGING_DIR}
ExecStart=/bin/bash -lc 'cd "${DEPLOY_APP_DIR}" && ./infra/ops/run-scheduled-backup.sh basebackup'
EOF

  cat >"${base_timer}" <<EOF
[Unit]
Description=StuHelper PostgreSQL base backup timer

[Timer]
OnCalendar=Sun *-*-* 03:45:00
Persistent=true

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
User=${DEPLOY_USER}
Group=${DEPLOY_GROUP}
WorkingDirectory=${DEPLOY_APP_DIR}
Environment=ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.shared
Environment=SECRETS_ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.secrets
Environment=GENERATED_ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.generated
Environment=GENERATED_SECRET_ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.generated.secrets
ExecStart=/bin/bash -lc 'cd "${DEPLOY_APP_DIR}" && ./infra/ops/sync-postgres-backups.sh'
EOF

  cat >"${sync_timer}" <<EOF
[Unit]
Description=StuHelper PostgreSQL backup artifact sync timer

[Timer]
OnCalendar=*:0/15
Persistent=true

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
