#!/usr/bin/env bash
set -euo pipefail

DEPLOY_USER="${DEPLOY_USER:-stuhelper}"
DEPLOY_GROUP="${DEPLOY_GROUP:-${DEPLOY_USER}}"
DEPLOY_APP_DIR="${DEPLOY_APP_DIR:-/opt/stuhelper}"
BACKUP_STAGING_DIR="${BACKUP_STAGING_DIR:-/var/lib/stuhelper/postgres/backup-staging}"
BACKUP_TIMERS_ACTIVATE="${BACKUP_TIMERS_ACTIVATE:-false}"
REMOTE_DEPLOY_CONFIG_FILE="${REMOTE_DEPLOY_CONFIG_FILE:-${DEPLOY_APP_DIR}/.deploy/remote.env}"

dump_service="/etc/systemd/system/stuhelper-postgres-dump-backup.service"
dump_timer="/etc/systemd/system/stuhelper-postgres-dump-backup.timer"
base_service="/etc/systemd/system/stuhelper-postgres-basebackup.service"
base_timer="/etc/systemd/system/stuhelper-postgres-basebackup.timer"
sync_service="/etc/systemd/system/stuhelper-postgres-backup-sync.service"
sync_timer="/etc/systemd/system/stuhelper-postgres-backup-sync.timer"

require_root() {
  if [[ "${EUID}" -ne 0 ]]; then
    echo "[install-backup-timers][error] run as root" >&2
    exit 1
  fi
}

require_service_identity() {
  local deploy_uid
  local deploy_group_record
  local deploy_gid

  [[ -n "${DEPLOY_USER}" && "${DEPLOY_USER}" != "root" && "${DEPLOY_USER}" != "0" ]] || {
    echo "[install-backup-timers][error] DEPLOY_USER must be an explicit non-root account" >&2
    exit 1
  }
  [[ -n "${DEPLOY_GROUP}" && "${DEPLOY_GROUP}" != "root" && "${DEPLOY_GROUP}" != "0" ]] || {
    echo "[install-backup-timers][error] DEPLOY_GROUP must be an explicit non-root group" >&2
    exit 1
  }
  deploy_uid="$(id -u "${DEPLOY_USER}" 2>/dev/null)" || {
    echo "[install-backup-timers][error] deploy user does not exist: ${DEPLOY_USER}" >&2
    exit 1
  }
  [[ "${deploy_uid}" != "0" ]] || {
    echo "[install-backup-timers][error] DEPLOY_USER must not resolve to uid 0" >&2
    exit 1
  }
  deploy_group_record="$(getent group "${DEPLOY_GROUP}")" || {
    echo "[install-backup-timers][error] deploy group does not exist: ${DEPLOY_GROUP}" >&2
    exit 1
  }
  IFS=: read -r _ _ deploy_gid _ <<<"${deploy_group_record}"
  [[ "${deploy_gid}" =~ ^[0-9]+$ && "${deploy_gid}" != "0" ]] || {
    echo "[install-backup-timers][error] DEPLOY_GROUP must not resolve to gid 0" >&2
    exit 1
  }
}

build_runuser_identity() {
  local deploy_user="$1"
  local deploy_group="$2"
  local output_name="$3"
  local supplementary_group_listing
  local supplementary_group_id
  local supplementary_group_record
  local supplementary_group_name
  local supplementary_group_resolved_id
  local -a supplementary_group_ids=()
  local -n output_ref="${output_name}"

  output_ref=(runuser -u "${deploy_user}" -g "${deploy_group}")
  if ! supplementary_group_listing="$(id -G "${deploy_user}")"; then
    echo "[install-backup-timers][error] failed to enumerate supplementary groups for ${deploy_user}" >&2
    return 1
  fi
  read -r -a supplementary_group_ids <<<"${supplementary_group_listing}"
  ((${#supplementary_group_ids[@]} > 0)) || {
    echo "[install-backup-timers][error] deploy user has no resolvable account groups: ${deploy_user}" >&2
    return 1
  }
  for supplementary_group_id in "${supplementary_group_ids[@]}"; do
    [[ "${supplementary_group_id}" =~ ^[0-9]+$ && "${supplementary_group_id}" != "0" ]] || {
      echo "[install-backup-timers][error] invalid supplementary group id for ${deploy_user}: ${supplementary_group_id}" >&2
      return 1
    }
    supplementary_group_record="$(getent group "${supplementary_group_id}")" || {
      echo "[install-backup-timers][error] failed to resolve supplementary group ${supplementary_group_id} for ${deploy_user}" >&2
      return 1
    }
    IFS=: read -r supplementary_group_name _ supplementary_group_resolved_id _ <<<"${supplementary_group_record}"
    [[ -n "${supplementary_group_name}" && "${supplementary_group_name}" != "root" &&
      "${supplementary_group_resolved_id}" == "${supplementary_group_id}" ]] || {
      echo "[install-backup-timers][error] supplementary group ${supplementary_group_id} has an unsafe or inconsistent NSS record" >&2
      return 1
    }
    output_ref+=(-G "${supplementary_group_name}")
  done
}

main() {
  require_root
  require_service_identity
  local required_script
  case "${BACKUP_TIMERS_ACTIVATE}" in
    true | false) ;;
    *)
      echo "[install-backup-timers][error] BACKUP_TIMERS_ACTIVATE must be true or false" >&2
      exit 1
      ;;
  esac
  for required_script in \
    "${DEPLOY_APP_DIR}/infra/ops/run-scheduled-backup.sh" \
    "${DEPLOY_APP_DIR}/infra/ops/backup-postgres.sh" \
    "${DEPLOY_APP_DIR}/infra/ops/sync-postgres-backups.sh" \
    "${DEPLOY_APP_DIR}/infra/ops/fetch-postgres-backups.sh" \
    "${DEPLOY_APP_DIR}/infra/ops/postgres-backup-evidence.sh" \
    "${DEPLOY_APP_DIR}/infra/ops/activate-existing-postgres-backups.sh" \
    "${DEPLOY_APP_DIR}/infra/ops/manage-postgres-backup-activation.py" \
    "${DEPLOY_APP_DIR}/infra/ops/remote-preflight.sh"; do
    [[ -x "${required_script}" ]] || {
      echo "[install-backup-timers][error] required deploy-bundle script is not executable: ${required_script}" >&2
      exit 1
    }
  done

  install -d -o "${DEPLOY_USER}" -g "${DEPLOY_GROUP}" -m 0755 "${DEPLOY_APP_DIR}/backups/postgres/logical"
  install -d -o "${DEPLOY_USER}" -g "${DEPLOY_GROUP}" -m 0755 "${DEPLOY_APP_DIR}/backups/postgres/base"
  install -d -o "${DEPLOY_USER}" -g "${DEPLOY_GROUP}" -m 0700 "${BACKUP_STAGING_DIR}"
  install -d -o "${DEPLOY_USER}" -g "${DEPLOY_GROUP}" -m 0755 "/var/lib/stuhelper/postgres/wal-restore"

  cat >"${dump_service}" <<EOF
[Unit]
Description=StuHelper PostgreSQL logical backup
After=docker.service network-online.target
Wants=network-online.target
StartLimitIntervalSec=0
StartLimitBurst=5

[Service]
Type=oneshot
RemainAfterExit=no
Restart=no
TimeoutStartSec=18h
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
Unit=stuhelper-postgres-dump-backup.service
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
StartLimitIntervalSec=0
StartLimitBurst=5

[Service]
Type=oneshot
RemainAfterExit=no
Restart=no
TimeoutStartSec=1d 2h
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
Unit=stuhelper-postgres-basebackup.service
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
StartLimitIntervalSec=0
StartLimitBurst=5

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
Environment=BACKUP_OBJECT_STORAGE_OFF_HOST_REQUIRED=true
UnsetEnvironment=LD_PRELOAD LD_LIBRARY_PATH LD_AUDIT GCONV_PATH LOCPATH
ExecStart=/usr/bin/env -i PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.shared SECRETS_ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.secrets GENERATED_ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.generated GENERATED_SECRET_ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.generated.secrets LOCAL_STATE_DIR=/var/lib/stuhelper BACKUP_OBJECT_STORAGE_OFF_HOST_REQUIRED=true /bin/bash --noprofile --norc ./infra/ops/run-scheduled-backup.sh sync
EOF

  cat >"${sync_timer}" <<EOF
[Unit]
Description=StuHelper PostgreSQL backup artifact sync timer

[Timer]
Unit=stuhelper-postgres-backup-sync.service
OnCalendar=*-*-* *:00/15:00
Persistent=true
AccuracySec=1min
RandomizedDelaySec=0
FixedRandomDelay=false

[Install]
WantedBy=timers.target
EOF

  systemctl daemon-reload
  if [[ "${BACKUP_TIMERS_ACTIVATE}" != "true" ]]; then
    echo "[install-backup-timers] units installed; timers were not activated"
    echo "[install-backup-timers] after production configuration and Vault are ready, run again with BACKUP_TIMERS_ACTIVATE=true"
    return 0
  fi

  command -v runuser >/dev/null 2>&1 || {
    echo "[install-backup-timers][error] runuser is required for non-root activation preflight" >&2
    exit 1
  }
  local -a activation_environment=(
    PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
    "ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.shared"
    "SECRETS_ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.secrets"
    "GENERATED_ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.generated"
    "GENERATED_SECRET_ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.generated.secrets"
    LOCAL_STATE_DIR=/var/lib/stuhelper
    "BACKUP_STAGING_DIR=${BACKUP_STAGING_DIR}"
    "REMOTE_DEPLOY_CONFIG_FILE=${REMOTE_DEPLOY_CONFIG_FILE}"
  )
  if [[ -n "${NGINX_PUBLIC_INGRESS_CONFIG_FILE:-}" ]]; then
    activation_environment+=(
      "NGINX_PUBLIC_INGRESS_CONFIG_FILE=${NGINX_PUBLIC_INGRESS_CONFIG_FILE}"
    )
  fi
  local -a runuser_identity=()
  build_runuser_identity "${DEPLOY_USER}" "${DEPLOY_GROUP}" runuser_identity
  "${runuser_identity[@]}" -- env -i \
    "${activation_environment[@]}" \
    /bin/bash --noprofile --norc "${DEPLOY_APP_DIR}/infra/ops/remote-preflight.sh" --timer-activation
  systemctl enable --now stuhelper-postgres-dump-backup.timer stuhelper-postgres-basebackup.timer stuhelper-postgres-backup-sync.timer

  echo "[install-backup-timers] installed:"
  echo "  - ${dump_service}"
  echo "  - ${dump_timer}"
  echo "  - ${base_service}"
  echo "  - ${base_timer}"
  echo "  - ${sync_service}"
  echo "  - ${sync_timer}"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  main "$@"
fi
