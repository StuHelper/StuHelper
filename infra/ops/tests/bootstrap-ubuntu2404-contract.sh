#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
BOOTSTRAP_SCRIPT="${REPO_ROOT}/infra/ops/bootstrap-ubuntu2404.sh"
BACKUP_TIMER_INSTALLER="${REPO_ROOT}/infra/ops/install-backup-timers.sh"

fail() {
  echo "[bootstrap-ubuntu2404-contract][error] $*" >&2
  exit 1
}

assert_contains() {
  local file="$1"
  local pattern="$2"
  if ! grep -Eq -- "${pattern}" "${file}"; then
    fail "expected ${file} to contain pattern: ${pattern}"
  fi
}

assert_not_contains() {
  local file="$1"
  local pattern="$2"
  if grep -Eq -- "${pattern}" "${file}"; then
    fail "expected ${file} not to contain pattern: ${pattern}"
  fi
}

[[ -f "${BOOTSTRAP_SCRIPT}" ]] || fail "missing bootstrap script: ${BOOTSTRAP_SCRIPT}"

bash -n "${BOOTSTRAP_SCRIPT}"

assert_contains "${BOOTSTRAP_SCRIPT}" 'DEPLOY_USER="\$\{DEPLOY_USER:-stuhelper\}"'
assert_contains "${BOOTSTRAP_SCRIPT}" 'DEPLOY_APP_DIR="\$\{DEPLOY_APP_DIR:-/opt/stuhelper\}"'
assert_contains "${BOOTSTRAP_SCRIPT}" 'CONFIGURE_UFW="\$\{CONFIGURE_UFW:-true\}"'
assert_contains "${BOOTSTRAP_SCRIPT}" 'INSTALL_BACKUP_TIMERS="\$\{INSTALL_BACKUP_TIMERS:-true\}"'
assert_contains "${BOOTSTRAP_SCRIPT}" 'BACKUP_STAGING_DIR="\$\{BACKUP_STAGING_DIR:-/var/lib/stuhelper/postgres/backup-staging\}"'
assert_contains "${BOOTSTRAP_SCRIPT}" 'INSTALL_GO="\$\{INSTALL_GO:-true\}"'
assert_contains "${BOOTSTRAP_SCRIPT}" 'GO_VERSION="\$\{GO_VERSION:-1\.26\.5\}"'
assert_contains "${BOOTSTRAP_SCRIPT}" 'run as root \(sudo bash infra/ops/bootstrap-ubuntu2404\.sh\)'
assert_contains "${BOOTSTRAP_SCRIPT}" 'require_non_root_deploy_identity'
assert_contains "${BOOTSTRAP_SCRIPT}" 'DEPLOY_USER must be an explicit non-root account'
assert_contains "${BOOTSTRAP_SCRIPT}" 'DEPLOY_GROUP must be an explicit non-root group'
assert_contains "${BOOTSTRAP_SCRIPT}" 'DEPLOY_USER must not resolve to uid 0'
assert_contains "${BOOTSTRAP_SCRIPT}" 'DEPLOY_GROUP must not resolve to gid 0'

assert_contains "${BOOTSTRAP_SCRIPT}" 'apt-get update -y'
assert_contains "${BOOTSTRAP_SCRIPT}" 'ca-certificates curl gnupg iproute2 jq openssl git bash python3'
assert_contains "${BOOTSTRAP_SCRIPT}" 'https://download\.docker\.com/linux/ubuntu/gpg'
assert_contains "${BOOTSTRAP_SCRIPT}" '/etc/apt/keyrings/docker\.asc'
assert_contains "${BOOTSTRAP_SCRIPT}" '/etc/apt/sources\.list\.d/docker\.list'
assert_contains "${BOOTSTRAP_SCRIPT}" 'docker-ce docker-ce-cli containerd\.io docker-buildx-plugin docker-compose-plugin'
assert_contains "${BOOTSTRAP_SCRIPT}" 'systemctl enable --now docker'

assert_contains "${BOOTSTRAP_SCRIPT}" 'useradd --system --create-home'
assert_contains "${BOOTSTRAP_SCRIPT}" 'usermod -aG docker'
assert_contains "${BOOTSTRAP_SCRIPT}" '\$\{DEPLOY_APP_DIR\}/\.deploy'
assert_contains "${BOOTSTRAP_SCRIPT}" '\$\{DEPLOY_APP_DIR\}/\.secrets/vault'
assert_contains "${BOOTSTRAP_SCRIPT}" '\$\{DEPLOY_APP_DIR\}/infra/generated'
assert_contains "${BOOTSTRAP_SCRIPT}" '\$\{DEPLOY_APP_DIR\}/backups/postgres/logical'
assert_contains "${BOOTSTRAP_SCRIPT}" '\$\{DEPLOY_APP_DIR\}/backups/postgres/base'
assert_contains "${BOOTSTRAP_SCRIPT}" 'install -d -o "\$\{DEPLOY_USER\}".*-m 0700 "\$\{BACKUP_STAGING_DIR\}"'
assert_contains "${BOOTSTRAP_SCRIPT}" 'install -d -o "\$\{DEPLOY_USER\}".*-m 0755 "/var/lib/stuhelper/postgres/wal-restore"'
assert_contains "${BOOTSTRAP_SCRIPT}" 'chmod 0600'
assert_contains "${BOOTSTRAP_SCRIPT}" 'DEPLOY_SSH_PUBKEY'
assert_contains "${BOOTSTRAP_SCRIPT}" 'authorized_keys'

assert_contains "${BOOTSTRAP_SCRIPT}" 'SECRET_BACKEND=vault-kv-v2'
assert_contains "${BOOTSTRAP_SCRIPT}" 'SHARED_ENV_SECRET_REF=secret/stuhelper/prod/shared-env'
assert_contains "${BOOTSTRAP_SCRIPT}" 'SECRETS_ENV_SECRET_REF=secret/stuhelper/prod/secrets-env'
assert_contains "${BOOTSTRAP_SCRIPT}" 'GENERATED_ENV_SECRET_REF=secret/stuhelper/prod/generated-secrets-env'
assert_contains "${BOOTSTRAP_SCRIPT}" 'VAULT_TOKEN_FILE=\$\{DEPLOY_APP_DIR\}/\.secrets/vault/token'
assert_contains "${BOOTSTRAP_SCRIPT}" 'VAULT_RUNTIME_TOKEN_POLICY=stuhelper-production-deploy'
assert_contains "${BOOTSTRAP_SCRIPT}" 'VAULT_RUNTIME_TOKEN_PERIOD_SECONDS=259200'
assert_contains "${BOOTSTRAP_SCRIPT}" 'VAULT_RUNTIME_TOKEN_MIN_TTL_SECONDS=43200'
assert_contains "${BOOTSTRAP_SCRIPT}" 'vault-runtime-token\.sh configure'
assert_contains "${BOOTSTRAP_SCRIPT}" 'do not persist the Vault root token'
assert_contains "${BOOTSTRAP_SCRIPT}" 'REGISTRY_AUTH_MODE=.workflow-token.'
assert_contains "${BOOTSTRAP_SCRIPT}" 'REGISTRY_USERNAME_SECRET_REF=secret/stuhelper/prod/registry-username'
assert_contains "${BOOTSTRAP_SCRIPT}" 'REGISTRY_PASSWORD_SECRET_REF=secret/stuhelper/prod/registry-password'

assert_contains "${BOOTSTRAP_SCRIPT}" 'ufw --force enable'
assert_contains "${BOOTSTRAP_SCRIPT}" 'ufw allow OpenSSH'
assert_contains "${BOOTSTRAP_SCRIPT}" 'ALLOW_HTTP_PORTS="\$\{ALLOW_HTTP_PORTS:-80,443\}"'
assert_contains "${BOOTSTRAP_SCRIPT}" 'ufw allow "\$\{port\}"/tcp'

assert_contains "${BOOTSTRAP_SCRIPT}" 'https://go\.dev/dl/go\$\{GO_VERSION\}\.linux-\$\{arch\}\.tar\.gz'
assert_contains "${BOOTSTRAP_SCRIPT}" 'ln -sf /usr/local/go/bin/go /usr/local/bin/go'
assert_contains "${BOOTSTRAP_SCRIPT}" 'ln -sf /usr/local/go/bin/gofmt /usr/local/bin/gofmt'

assert_contains "${BOOTSTRAP_SCRIPT}" 'stuhelper-postgres-dump-backup\.service'
assert_contains "${BOOTSTRAP_SCRIPT}" 'stuhelper-postgres-basebackup\.service'
assert_contains "${BOOTSTRAP_SCRIPT}" 'stuhelper-postgres-backup-sync\.service'
assert_contains "${BOOTSTRAP_SCRIPT}" 'BACKUP_SERVICE_GROUP=.\$\{DEPLOY_GROUP\}.'
assert_contains "${BOOTSTRAP_SCRIPT}" '^BACKUP_SERVICE_GROUP=\$\{DEPLOY_GROUP\}$'
assert_contains "${BOOTSTRAP_SCRIPT}" '^Unit=stuhelper-postgres-dump-backup\.service$'
assert_contains "${BOOTSTRAP_SCRIPT}" '^Unit=stuhelper-postgres-basebackup\.service$'
assert_contains "${BOOTSTRAP_SCRIPT}" '^Unit=stuhelper-postgres-backup-sync\.service$'
assert_contains "${BOOTSTRAP_SCRIPT}" '^OnCalendar=\*-\*-\* \*:00/15:00$'
[[ "$(grep -c '^AccuracySec=1min$' "${BOOTSTRAP_SCRIPT}")" == "3" ]] ||
  fail "bootstrap fallback must bound timer coalescing accuracy to one minute"
[[ "$(grep -c '^RandomizedDelaySec=0$' "${BOOTSTRAP_SCRIPT}")" == "3" ]] ||
  fail "bootstrap fallback must disable randomized timer delays"
[[ "$(grep -c '^FixedRandomDelay=false$' "${BOOTSTRAP_SCRIPT}")" == "3" ]] ||
  fail "bootstrap fallback must explicitly disable fixed randomized delays"
assert_contains "${BOOTSTRAP_SCRIPT}" 'Environment=BACKUP_STAGING_DIR=\$\{BACKUP_STAGING_DIR\}'
[[ "$(grep -c '^Environment=LOCAL_STATE_DIR=/var/lib/stuhelper$' "${BOOTSTRAP_SCRIPT}")" == "3" ]] ||
  fail "bootstrap fallback must provide a HOME-independent local state path to every isolated backup service"
[[ "$(grep -c '^Environment=BACKUP_OBJECT_STORAGE_OFF_HOST_REQUIRED=true$' "${BOOTSTRAP_SCRIPT}")" == "3" ]] ||
  fail "bootstrap fallback must require off-host storage in all three backup services"
[[ "$(grep -c '^RemainAfterExit=no$' "${BOOTSTRAP_SCRIPT}")" == "3" ]] ||
  fail "bootstrap fallback must keep all backup oneshot services restartable by their timers"
[[ "$(grep -c '^Restart=no$' "${BOOTSTRAP_SCRIPT}")" == "3" ]] ||
  fail "bootstrap fallback must leave retries under timer control"
[[ "$(grep -c '^StartLimitIntervalSec=0$' "${BOOTSTRAP_SCRIPT}")" == "3" ]] ||
  fail "bootstrap fallback must disable service start-rate limiting"
[[ "$(grep -c '^StartLimitBurst=5$' "${BOOTSTRAP_SCRIPT}")" == "3" ]] ||
  fail "bootstrap fallback must pin the canonical service start-limit burst"
assert_contains "${BOOTSTRAP_SCRIPT}" '^TimeoutStartSec=18h$'
assert_contains "${BOOTSTRAP_SCRIPT}" '^TimeoutStartSec=1d 2h$'
assert_contains "${BOOTSTRAP_SCRIPT}" '^TimeoutStartSec=12h$'
[[ "$(grep -c '^TimeoutStopSec=2min$' "${BOOTSTRAP_SCRIPT}")" == "3" ]] ||
  fail "bootstrap fallback must bound service termination after a backup timeout"
[[ "$(grep -c '^KillMode=control-group$' "${BOOTSTRAP_SCRIPT}")" == "3" ]] ||
  fail "bootstrap fallback must terminate the complete backup service cgroup"
[[ "$(grep -c '^SendSIGKILL=yes$' "${BOOTSTRAP_SCRIPT}")" == "3" ]] ||
  fail "bootstrap fallback must make the finite backup deadline enforceable"
[[ "$(grep -c '^UnsetEnvironment=LD_PRELOAD LD_LIBRARY_PATH LD_AUDIT GCONV_PATH LOCPATH$' "${BOOTSTRAP_SCRIPT}")" == "3" ]] ||
  fail "bootstrap fallback must clear dynamic-loader inputs before all three backup services start"
[[ "$(grep -Ec '^ExecStart=/usr/bin/env -i PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin .* BACKUP_OBJECT_STORAGE_OFF_HOST_REQUIRED=true /bin/bash --noprofile --norc \./infra/ops/' "${BOOTSTRAP_SCRIPT}")" == "3" ]] ||
  fail "bootstrap fallback must start all backup services with an allowlisted empty environment"
assert_not_contains "${BOOTSTRAP_SCRIPT}" 'ExecStart=/bin/bash -lc'
assert_contains "${BOOTSTRAP_SCRIPT}" 'systemctl daemon-reload'
assert_contains "${BOOTSTRAP_SCRIPT}" 'systemctl disable --now stuhelper-postgres-dump-backup\.timer stuhelper-postgres-basebackup\.timer stuhelper-postgres-backup-sync\.timer'
assert_contains "${BOOTSTRAP_SCRIPT}" 'systemctl reset-failed stuhelper-postgres-dump-backup\.service stuhelper-postgres-basebackup\.service stuhelper-postgres-backup-sync\.service'
assert_contains "${BOOTSTRAP_SCRIPT}" 'backup timer units remain disabled until the deploy bundle and production configuration exist'
assert_not_contains "${BOOTSTRAP_SCRIPT}" 'systemctl enable --now stuhelper-postgres-dump-backup\.timer stuhelper-postgres-basebackup\.timer stuhelper-postgres-backup-sync\.timer'

[[ -f "${BACKUP_TIMER_INSTALLER}" ]] ||
  fail "missing backup timer installer: ${BACKUP_TIMER_INSTALLER}"
bash -n "${BACKUP_TIMER_INSTALLER}"
assert_contains "${BACKUP_TIMER_INSTALLER}" 'require_service_identity'
assert_contains "${BACKUP_TIMER_INSTALLER}" 'deploy user does not exist'
assert_contains "${BACKUP_TIMER_INSTALLER}" 'deploy group does not exist'
assert_contains "${BACKUP_TIMER_INSTALLER}" 'DEPLOY_USER must be an explicit non-root account'
assert_contains "${BACKUP_TIMER_INSTALLER}" 'DEPLOY_GROUP must be an explicit non-root group'
assert_contains "${BACKUP_TIMER_INSTALLER}" 'DEPLOY_USER must not resolve to uid 0'
assert_contains "${BACKUP_TIMER_INSTALLER}" 'DEPLOY_GROUP must not resolve to gid 0'
assert_contains "${BACKUP_TIMER_INSTALLER}" 'BACKUP_STAGING_DIR="\$\{BACKUP_STAGING_DIR:-/var/lib/stuhelper/postgres/backup-staging\}"'
assert_contains "${BACKUP_TIMER_INSTALLER}" 'install -d -o "\$\{DEPLOY_USER\}".*-m 0700 "\$\{BACKUP_STAGING_DIR\}"'
assert_contains "${BACKUP_TIMER_INSTALLER}" 'Environment=BACKUP_STAGING_DIR=\$\{BACKUP_STAGING_DIR\}'
assert_contains "${BACKUP_TIMER_INSTALLER}" '^Unit=\$\{SYSTEMD_PREFIX\}-postgres-dump-backup\.service$'
assert_contains "${BACKUP_TIMER_INSTALLER}" '^Unit=\$\{SYSTEMD_PREFIX\}-postgres-basebackup\.service$'
assert_contains "${BACKUP_TIMER_INSTALLER}" '^Unit=\$\{SYSTEMD_PREFIX\}-postgres-backup-sync\.service$'
assert_contains "${BACKUP_TIMER_INSTALLER}" '^OnCalendar=\*-\*-\* \*:00/15:00$'
[[ "$(grep -c '^AccuracySec=1min$' "${BACKUP_TIMER_INSTALLER}")" == "3" ]] ||
  fail "backup timer installer must bound timer coalescing accuracy to one minute"
[[ "$(grep -c '^RandomizedDelaySec=0$' "${BACKUP_TIMER_INSTALLER}")" == "3" ]] ||
  fail "backup timer installer must disable randomized timer delays"
[[ "$(grep -c '^FixedRandomDelay=false$' "${BACKUP_TIMER_INSTALLER}")" == "3" ]] ||
  fail "backup timer installer must explicitly disable fixed randomized delays"
[[ "$(grep -c '^Environment=LOCAL_STATE_DIR=/var/lib/stuhelper$' "${BACKUP_TIMER_INSTALLER}")" == "3" ]] ||
  fail "backup timer installer must provide a HOME-independent local state path to every isolated backup service"
[[ "$(grep -c '^Environment=BACKUP_OBJECT_STORAGE_OFF_HOST_REQUIRED=true$' "${BACKUP_TIMER_INSTALLER}")" == "3" ]] ||
  fail "backup timer installer must require off-host storage in all three services"
[[ "$(grep -c '^RemainAfterExit=no$' "${BACKUP_TIMER_INSTALLER}")" == "3" ]] ||
  fail "backup timer installer must keep all backup oneshot services restartable by their timers"
[[ "$(grep -c '^Restart=no$' "${BACKUP_TIMER_INSTALLER}")" == "3" ]] ||
  fail "backup timer installer must leave retries under timer control"
[[ "$(grep -c '^StartLimitIntervalSec=0$' "${BACKUP_TIMER_INSTALLER}")" == "3" ]] ||
  fail "backup timer installer must disable service start-rate limiting"
[[ "$(grep -c '^StartLimitBurst=5$' "${BACKUP_TIMER_INSTALLER}")" == "3" ]] ||
  fail "backup timer installer must pin the canonical service start-limit burst"
assert_contains "${BACKUP_TIMER_INSTALLER}" '^TimeoutStartSec=18h$'
assert_contains "${BACKUP_TIMER_INSTALLER}" '^TimeoutStartSec=1d 2h$'
assert_contains "${BACKUP_TIMER_INSTALLER}" '^TimeoutStartSec=12h$'
[[ "$(grep -c '^TimeoutStopSec=2min$' "${BACKUP_TIMER_INSTALLER}")" == "3" ]] ||
  fail "backup timer installer must bound service termination after a backup timeout"
[[ "$(grep -c '^KillMode=control-group$' "${BACKUP_TIMER_INSTALLER}")" == "3" ]] ||
  fail "backup timer installer must terminate the complete backup service cgroup"
[[ "$(grep -c '^SendSIGKILL=yes$' "${BACKUP_TIMER_INSTALLER}")" == "3" ]] ||
  fail "backup timer installer must make the finite backup deadline enforceable"
[[ "$(grep -c '^UnsetEnvironment=LD_PRELOAD LD_LIBRARY_PATH LD_AUDIT GCONV_PATH LOCPATH$' "${BACKUP_TIMER_INSTALLER}")" == "3" ]] ||
  fail "backup timer installer must clear dynamic-loader inputs before all three backup services start"
[[ "$(grep -Ec '^ExecStart=/usr/bin/env -i PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin .* BACKUP_OBJECT_STORAGE_OFF_HOST_REQUIRED=true /bin/bash --noprofile --norc \./infra/ops/' "${BACKUP_TIMER_INSTALLER}")" == "3" ]] ||
  fail "backup timer installer must start all backup services with an allowlisted empty environment"
assert_not_contains "${BACKUP_TIMER_INSTALLER}" 'ExecStart=/bin/bash -lc'
assert_contains "${BACKUP_TIMER_INSTALLER}" 'systemctl reset-failed "\$\{SYSTEMD_PREFIX\}-postgres-dump-backup\.service" "\$\{SYSTEMD_PREFIX\}-postgres-basebackup\.service" "\$\{SYSTEMD_PREFIX\}-postgres-backup-sync\.service"'
assert_contains "${BACKUP_TIMER_INSTALLER}" 'systemctl enable --now "\$\{SYSTEMD_PREFIX\}-postgres-dump-backup\.timer"'

echo "[bootstrap-ubuntu2404-contract] all assertions passed"
