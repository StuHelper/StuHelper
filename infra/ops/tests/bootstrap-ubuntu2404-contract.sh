#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
BOOTSTRAP_SCRIPT="${REPO_ROOT}/infra/ops/bootstrap-ubuntu2404.sh"

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

[[ -f "${BOOTSTRAP_SCRIPT}" ]] || fail "missing bootstrap script: ${BOOTSTRAP_SCRIPT}"

bash -n "${BOOTSTRAP_SCRIPT}"

assert_contains "${BOOTSTRAP_SCRIPT}" 'DEPLOY_USER="\$\{DEPLOY_USER:-stuhelper\}"'
assert_contains "${BOOTSTRAP_SCRIPT}" 'DEPLOY_APP_DIR="\$\{DEPLOY_APP_DIR:-/opt/stuhelper\}"'
assert_contains "${BOOTSTRAP_SCRIPT}" 'CONFIGURE_UFW="\$\{CONFIGURE_UFW:-true\}"'
assert_contains "${BOOTSTRAP_SCRIPT}" 'INSTALL_BACKUP_TIMERS="\$\{INSTALL_BACKUP_TIMERS:-true\}"'
assert_contains "${BOOTSTRAP_SCRIPT}" 'INSTALL_GO="\$\{INSTALL_GO:-true\}"'
assert_contains "${BOOTSTRAP_SCRIPT}" 'GO_VERSION="\$\{GO_VERSION:-1\.26\.0\}"'
assert_contains "${BOOTSTRAP_SCRIPT}" 'run as root \(sudo bash infra/ops/bootstrap-ubuntu2404\.sh\)'

assert_contains "${BOOTSTRAP_SCRIPT}" 'apt-get update -y'
assert_contains "${BOOTSTRAP_SCRIPT}" 'ca-certificates curl gnupg jq openssl git bash python3'
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
assert_contains "${BOOTSTRAP_SCRIPT}" 'chmod 0600'
assert_contains "${BOOTSTRAP_SCRIPT}" 'DEPLOY_SSH_PUBKEY'
assert_contains "${BOOTSTRAP_SCRIPT}" 'authorized_keys'

assert_contains "${BOOTSTRAP_SCRIPT}" 'SECRET_BACKEND=vault-kv-v2'
assert_contains "${BOOTSTRAP_SCRIPT}" 'SHARED_ENV_SECRET_REF=secret/stuhelper/prod/shared-env'
assert_contains "${BOOTSTRAP_SCRIPT}" 'SECRETS_ENV_SECRET_REF=secret/stuhelper/prod/secrets-env'
assert_contains "${BOOTSTRAP_SCRIPT}" 'GENERATED_ENV_SECRET_REF=secret/stuhelper/prod/generated-secrets-env'
assert_contains "${BOOTSTRAP_SCRIPT}" 'VAULT_TOKEN_FILE=\$\{DEPLOY_APP_DIR\}/\.secrets/vault/token'
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
assert_contains "${BOOTSTRAP_SCRIPT}" 'systemctl daemon-reload'
assert_contains "${BOOTSTRAP_SCRIPT}" 'systemctl enable --now stuhelper-postgres-dump-backup\.timer stuhelper-postgres-basebackup\.timer stuhelper-postgres-backup-sync\.timer'

echo "[bootstrap-ubuntu2404-contract] all assertions passed"
