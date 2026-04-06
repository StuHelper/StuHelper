#!/usr/bin/env bash
set -euo pipefail

DEPLOY_USER="${DEPLOY_USER:-stuhelper}"
DEPLOY_GROUP="${DEPLOY_GROUP:-${DEPLOY_USER}}"
DEPLOY_APP_DIR="${DEPLOY_APP_DIR:-/opt/stuhelper}"
DEPLOY_SSH_PUBKEY="${DEPLOY_SSH_PUBKEY:-}"
CONFIGURE_UFW="${CONFIGURE_UFW:-true}"
ALLOW_HTTP_PORTS="${ALLOW_HTTP_PORTS:-80,443}"
INSTALL_BACKUP_TIMERS="${INSTALL_BACKUP_TIMERS:-true}"

log() {
  echo "[bootstrap-ubuntu2404] $*"
}

die() {
  echo "[bootstrap-ubuntu2404][error] $*" >&2
  exit 1
}

require_root() {
  if [[ "${EUID}" -ne 0 ]]; then
    die "run as root (sudo bash infra/ops/bootstrap-ubuntu2404.sh)"
  fi
}

apt_install() {
  export DEBIAN_FRONTEND=noninteractive
  apt-get update -y
  apt-get install -y "$@"
}

ensure_docker_repo() {
  install -m 0755 -d /etc/apt/keyrings
  if [[ ! -f /etc/apt/keyrings/docker.asc ]]; then
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
    chmod a+r /etc/apt/keyrings/docker.asc
  fi

  local arch codename
  arch="$(dpkg --print-architecture)"
  codename="$(. /etc/os-release && echo "${VERSION_CODENAME}")"
  cat >/etc/apt/sources.list.d/docker.list <<EOF
deb [arch=${arch} signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu ${codename} stable
EOF
}

ensure_deploy_user() {
  if ! id -u "${DEPLOY_USER}" >/dev/null 2>&1; then
    log "creating deploy user ${DEPLOY_USER}"
    groupadd --system "${DEPLOY_GROUP}" >/dev/null 2>&1 || true
    useradd --system --create-home --gid "${DEPLOY_GROUP}" --shell /bin/bash "${DEPLOY_USER}"
  fi

  usermod -aG docker "${DEPLOY_USER}"
  install -d -o "${DEPLOY_USER}" -g "${DEPLOY_GROUP}" -m 0755 "${DEPLOY_APP_DIR}"
  install -d -o "${DEPLOY_USER}" -g "${DEPLOY_GROUP}" -m 0755 "${DEPLOY_APP_DIR}/infra/generated"
  install -d -o "${DEPLOY_USER}" -g "${DEPLOY_GROUP}" -m 0755 "${DEPLOY_APP_DIR}/infra/generated/postgres"
  install -d -o "${DEPLOY_USER}" -g "${DEPLOY_GROUP}" -m 0755 "${DEPLOY_APP_DIR}/infra/generated/postgres/wal-archive"
  install -d -o "${DEPLOY_USER}" -g "${DEPLOY_GROUP}" -m 0755 "${DEPLOY_APP_DIR}/backups/postgres/logical"
  install -d -o "${DEPLOY_USER}" -g "${DEPLOY_GROUP}" -m 0755 "${DEPLOY_APP_DIR}/backups/postgres/base"
  touch "${DEPLOY_APP_DIR}/.env.prod.shared" "${DEPLOY_APP_DIR}/.env.prod.secrets"
  chown "${DEPLOY_USER}:${DEPLOY_GROUP}" "${DEPLOY_APP_DIR}/.env.prod.shared" "${DEPLOY_APP_DIR}/.env.prod.secrets"

  if [[ -n "${DEPLOY_SSH_PUBKEY}" ]]; then
    install -d -o "${DEPLOY_USER}" -g "${DEPLOY_GROUP}" -m 0700 "/home/${DEPLOY_USER}/.ssh"
    touch "/home/${DEPLOY_USER}/.ssh/authorized_keys"
    grep -qxF "${DEPLOY_SSH_PUBKEY}" "/home/${DEPLOY_USER}/.ssh/authorized_keys" || \
      echo "${DEPLOY_SSH_PUBKEY}" >>"/home/${DEPLOY_USER}/.ssh/authorized_keys"
    chown "${DEPLOY_USER}:${DEPLOY_GROUP}" "/home/${DEPLOY_USER}/.ssh/authorized_keys"
    chmod 0600 "/home/${DEPLOY_USER}/.ssh/authorized_keys"
  fi
}

configure_ufw() {
  [[ "${CONFIGURE_UFW}" == "true" ]] || return 0

  apt_install ufw
  ufw --force enable
  ufw allow OpenSSH

  local port
  IFS=',' read -r -a ports <<<"${ALLOW_HTTP_PORTS}"
  for port in "${ports[@]}"; do
    [[ -n "${port}" ]] || continue
    ufw allow "${port}"/tcp
  done
}

install_backup_timers() {
  [[ "${INSTALL_BACKUP_TIMERS}" == "true" ]] || return 0

  if [[ -x "${DEPLOY_APP_DIR}/infra/ops/install-backup-timers.sh" ]]; then
    log "installing PostgreSQL backup timers from deploy bundle"
    DEPLOY_USER="${DEPLOY_USER}" \
    DEPLOY_GROUP="${DEPLOY_GROUP}" \
    DEPLOY_APP_DIR="${DEPLOY_APP_DIR}" \
    "${DEPLOY_APP_DIR}/infra/ops/install-backup-timers.sh"
    return 0
  fi

  log "deploy bundle not present yet, installing backup timers with bootstrap defaults"

  cat >/etc/systemd/system/stuhelper-postgres-dump-backup.service <<EOF
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
ExecStart=/bin/bash -lc 'cd "${DEPLOY_APP_DIR}" && ./infra/ops/run-scheduled-backup.sh dump'
EOF

  cat >/etc/systemd/system/stuhelper-postgres-dump-backup.timer <<EOF
[Unit]
Description=StuHelper PostgreSQL logical backup timer

[Timer]
OnCalendar=*-*-* 03:15:00
Persistent=true

[Install]
WantedBy=timers.target
EOF

  cat >/etc/systemd/system/stuhelper-postgres-basebackup.service <<EOF
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
ExecStart=/bin/bash -lc 'cd "${DEPLOY_APP_DIR}" && ./infra/ops/run-scheduled-backup.sh basebackup'
EOF

  cat >/etc/systemd/system/stuhelper-postgres-basebackup.timer <<EOF
[Unit]
Description=StuHelper PostgreSQL base backup timer

[Timer]
OnCalendar=Sun *-*-* 03:45:00
Persistent=true

[Install]
WantedBy=timers.target
EOF

  systemctl daemon-reload
  systemctl enable --now stuhelper-postgres-dump-backup.timer stuhelper-postgres-basebackup.timer
}

main() {
  require_root

  log "installing base packages"
  apt_install ca-certificates curl gnupg jq openssl git bash

  log "installing Docker Engine and Compose plugin"
  ensure_docker_repo
  apt-get update -y
  apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
  systemctl enable --now docker

  log "preparing deploy user and application directory"
  ensure_deploy_user

  log "configuring firewall"
  configure_ufw

  install_backup_timers

  cat <<EOF

[bootstrap-ubuntu2404] completed
Deploy user: ${DEPLOY_USER}
Deploy dir:  ${DEPLOY_APP_DIR}

Next steps:
1. Put production shared config into ${DEPLOY_APP_DIR}/.env.prod.shared
2. Put production secrets into ${DEPLOY_APP_DIR}/.env.prod.secrets
3. Ensure the deploy bundle is synced to ${DEPLOY_APP_DIR}; re-run bootstrap or install-backup-timers.sh afterwards if you want systemd timers installed from the repo
4. Ensure the GitLab CI variables are set:
   - REGISTRY_USERNAME / REGISTRY_PASSWORD
   - DEPLOY_HOST / DEPLOY_PORT / DEPLOY_USER / DEPLOY_APP_DIR / DEPLOY_SSH_KEY
   - optional DEPLOY_SHARED_ENV_FILE_CONTENT / DEPLOY_SECRET_ENV_FILE_CONTENT
5. Push to main and approve the production deploy job to trigger build -> push -> remote deploy
EOF
}

main "$@"
