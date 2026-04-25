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
  install -d -o "${DEPLOY_USER}" -g "${DEPLOY_GROUP}" -m 0700 "${DEPLOY_APP_DIR}/.deploy"
  install -d -o "${DEPLOY_USER}" -g "${DEPLOY_GROUP}" -m 0700 "${DEPLOY_APP_DIR}/.secrets"
  install -d -o "${DEPLOY_USER}" -g "${DEPLOY_GROUP}" -m 0700 "${DEPLOY_APP_DIR}/.secrets/vault"
  install -d -o "${DEPLOY_USER}" -g "${DEPLOY_GROUP}" -m 0755 "${DEPLOY_APP_DIR}/infra/generated"
  install -d -o "${DEPLOY_USER}" -g "${DEPLOY_GROUP}" -m 0755 "${DEPLOY_APP_DIR}/infra/generated/postgres"
  install -d -o "${DEPLOY_USER}" -g "${DEPLOY_GROUP}" -m 0755 "${DEPLOY_APP_DIR}/backups/postgres/logical"
  install -d -o "${DEPLOY_USER}" -g "${DEPLOY_GROUP}" -m 0755 "${DEPLOY_APP_DIR}/backups/postgres/base"
  touch \
    "${DEPLOY_APP_DIR}/.env.prod.shared" \
    "${DEPLOY_APP_DIR}/.env.prod.secrets" \
    "${DEPLOY_APP_DIR}/.env.prod.generated" \
    "${DEPLOY_APP_DIR}/.env.prod.generated.secrets" \
    "${DEPLOY_APP_DIR}/.secrets/vault/token"
  chown \
    "${DEPLOY_USER}:${DEPLOY_GROUP}" \
    "${DEPLOY_APP_DIR}/.env.prod.shared" \
    "${DEPLOY_APP_DIR}/.env.prod.secrets" \
    "${DEPLOY_APP_DIR}/.env.prod.generated" \
    "${DEPLOY_APP_DIR}/.env.prod.generated.secrets" \
    "${DEPLOY_APP_DIR}/.secrets/vault/token"
  chmod 0600 \
    "${DEPLOY_APP_DIR}/.env.prod.shared" \
    "${DEPLOY_APP_DIR}/.env.prod.secrets" \
    "${DEPLOY_APP_DIR}/.env.prod.generated" \
    "${DEPLOY_APP_DIR}/.env.prod.generated.secrets" \
    "${DEPLOY_APP_DIR}/.secrets/vault/token"

  if [[ -n "${DEPLOY_SSH_PUBKEY}" ]]; then
    install -d -o "${DEPLOY_USER}" -g "${DEPLOY_GROUP}" -m 0700 "/home/${DEPLOY_USER}/.ssh"
    touch "/home/${DEPLOY_USER}/.ssh/authorized_keys"
    grep -qxF "${DEPLOY_SSH_PUBKEY}" "/home/${DEPLOY_USER}/.ssh/authorized_keys" || \
      echo "${DEPLOY_SSH_PUBKEY}" >>"/home/${DEPLOY_USER}/.ssh/authorized_keys"
    chown "${DEPLOY_USER}:${DEPLOY_GROUP}" "/home/${DEPLOY_USER}/.ssh/authorized_keys"
    chmod 0600 "/home/${DEPLOY_USER}/.ssh/authorized_keys"
  fi
}

ensure_remote_deploy_config() {
  local remote_config="${DEPLOY_APP_DIR}/.deploy/remote.env"

  if [[ -x "${DEPLOY_APP_DIR}/infra/ops/init-remote-deploy-config.sh" ]]; then
    log "rendering remote deploy control-plane config from repository script"
    su -s /bin/bash "${DEPLOY_USER}" -c "
      cd '${DEPLOY_APP_DIR}' && \
      REMOTE_DEPLOY_CONFIG_FILE='${remote_config}' \
      ENV_FILE='${DEPLOY_APP_DIR}/.env.prod.shared' \
      SECRETS_ENV_FILE='${DEPLOY_APP_DIR}/.env.prod.secrets' \
      GENERATED_ENV_FILE='${DEPLOY_APP_DIR}/.env.prod.generated' \
      GENERATED_SECRET_ENV_FILE='${DEPLOY_APP_DIR}/.env.prod.generated.secrets' \
      SECRET_BACKEND='vault-kv-v2' \
      SECRET_FILE_ROOT='${DEPLOY_APP_DIR}/.secrets' \
      SHARED_ENV_SECRET_REF='secret/stuhelper/prod/shared-env' \
      SECRETS_ENV_SECRET_REF='secret/stuhelper/prod/secrets-env' \
      GENERATED_ENV_SECRET_REF='secret/stuhelper/prod/generated-secrets-env' \
      REGISTRY='REPLACE_WITH_REGISTRY_HOST' \
      REGISTRY_USERNAME_SECRET_REF='secret/stuhelper/prod/registry-username' \
      REGISTRY_PASSWORD_SECRET_REF='secret/stuhelper/prod/registry-password' \
      VAULT_ADDR='REPLACE_WITH_VAULT_ADDR' \
      VAULT_TOKEN_FILE='${DEPLOY_APP_DIR}/.secrets/vault/token' \
      ./infra/ops/init-remote-deploy-config.sh
    "
    return 0
  fi

  cat >"${remote_config}" <<EOF
# Remote-owned deploy control plane for StuHelper.
# Manage this file on the target host; CI/Ansible no longer rewrite it per release.
REGISTRY=REPLACE_WITH_REGISTRY_HOST
REGISTRY_USERNAME_SECRET_REF=secret/stuhelper/prod/registry-username
REGISTRY_PASSWORD_SECRET_REF=secret/stuhelper/prod/registry-password
ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.shared
SECRETS_ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.secrets
GENERATED_ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.generated
GENERATED_SECRET_ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.generated.secrets
SECRET_BACKEND=vault-kv-v2
SHARED_ENV_SECRET_REF=secret/stuhelper/prod/shared-env
SECRETS_ENV_SECRET_REF=secret/stuhelper/prod/secrets-env
GENERATED_ENV_SECRET_REF=secret/stuhelper/prod/generated-secrets-env
SECRET_FILE_ROOT=${DEPLOY_APP_DIR}/.secrets
VAULT_ADDR=REPLACE_WITH_VAULT_ADDR
VAULT_NAMESPACE=
VAULT_TOKEN_FILE=${DEPLOY_APP_DIR}/.secrets/vault/token
VAULT_KV_MOUNT=secret
EOF
  chown "${DEPLOY_USER}:${DEPLOY_GROUP}" "${remote_config}"
  chmod 0600 "${remote_config}"
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
Environment=GENERATED_SECRET_ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.generated.secrets
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
Environment=GENERATED_SECRET_ENV_FILE=${DEPLOY_APP_DIR}/.env.prod.generated.secrets
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

  cat >/etc/systemd/system/stuhelper-postgres-backup-sync.service <<EOF
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

  cat >/etc/systemd/system/stuhelper-postgres-backup-sync.timer <<EOF
[Unit]
Description=StuHelper PostgreSQL backup artifact sync timer

[Timer]
OnCalendar=*:0/15
Persistent=true

[Install]
WantedBy=timers.target
EOF

  systemctl daemon-reload
  systemctl enable --now stuhelper-postgres-dump-backup.timer stuhelper-postgres-basebackup.timer stuhelper-postgres-backup-sync.timer
}

main() {
  require_root

  log "installing base packages"
  apt_install ca-certificates curl gnupg jq openssl git bash python3

  log "installing Docker Engine and Compose plugin"
  ensure_docker_repo
  apt-get update -y
  apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
  systemctl enable --now docker

  log "preparing deploy user and application directory"
  ensure_deploy_user
  ensure_remote_deploy_config

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
3. Put the Vault token into:
   - ${DEPLOY_APP_DIR}/.secrets/vault/token
4. Review the remote deploy control plane in ${DEPLOY_APP_DIR}/.deploy/remote.env
   - registry/shared/generated secret refs should point to your remote secret backend
5. Ensure the deploy bundle is synced to ${DEPLOY_APP_DIR}; re-run bootstrap or install-backup-timers.sh afterwards if you want systemd timers installed from the repo
6. Ensure the GitLab CI variables are set:
   - DEPLOY_HOST / DEPLOY_PORT / DEPLOY_USER / DEPLOY_APP_DIR / DEPLOY_SSH_KEY
7. Push to main and approve the production deploy job to trigger build -> push -> remote deploy
EOF
}

main "$@"
