#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
BOOTSTRAP_SCRIPT="${REPO_ROOT}/infra/ops/bootstrap-dev-ubuntu2404.sh"
MAKEFILE="${REPO_ROOT}/Makefile"
QUICKSTART="${REPO_ROOT}/docs/QUICKSTART.md"

fail() {
  echo "[bootstrap-dev-ubuntu2404-contract][error] $*" >&2
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
[[ -f "${MAKEFILE}" ]] || fail "missing Makefile"
[[ -f "${QUICKSTART}" ]] || fail "missing Quickstart"

bash -n "${BOOTSTRAP_SCRIPT}"

assert_contains "${BOOTSTRAP_SCRIPT}" 'run as root \(sudo bash infra/ops/bootstrap-dev-ubuntu2404\.sh\)'
assert_contains "${BOOTSTRAP_SCRIPT}" 'DEV_USER="\$\{DEV_USER:-\$\{SUDO_USER:-\$\{USER:-\}\}\}"'
assert_contains "${BOOTSTRAP_SCRIPT}" 'GO_VERSION="\$\{GO_VERSION:-1\.26\.0\}"'
assert_contains "${BOOTSTRAP_SCRIPT}" 'NODE_MAJOR="\$\{NODE_MAJOR:-24\}"'
assert_contains "${BOOTSTRAP_SCRIPT}" 'PNPM_VERSION="\$\{PNPM_VERSION:-10\.32\.1\}"'
assert_contains "${BOOTSTRAP_SCRIPT}" 'AIR_VERSION="\$\{AIR_VERSION:-v1\.61\.7\}"'
assert_contains "${BOOTSTRAP_SCRIPT}" 'PLAYWRIGHT_VERSION="\$\{PLAYWRIGHT_VERSION:-1\.58\.2\}"'
assert_contains "${BOOTSTRAP_SCRIPT}" 'INSTALL_WORKSPACE_DEPS="\$\{INSTALL_WORKSPACE_DEPS:-false\}"'

assert_contains "${BOOTSTRAP_SCRIPT}" 'ca-certificates'
assert_contains "${BOOTSTRAP_SCRIPT}" 'build-essential'
assert_contains "${BOOTSTRAP_SCRIPT}" 'lsof'
assert_contains "${BOOTSTRAP_SCRIPT}" 'https://download\.docker\.com/linux/ubuntu/gpg'
assert_contains "${BOOTSTRAP_SCRIPT}" 'docker-ce docker-ce-cli containerd\.io docker-buildx-plugin docker-compose-plugin'
assert_contains "${BOOTSTRAP_SCRIPT}" 'systemctl enable --now docker'
assert_contains "${BOOTSTRAP_SCRIPT}" 'usermod -aG docker "\$\{DEV_USER\}"'

assert_contains "${BOOTSTRAP_SCRIPT}" 'https://deb\.nodesource\.com/gpgkey/nodesource-repo\.gpg\.key'
assert_contains "${BOOTSTRAP_SCRIPT}" 'https://deb\.nodesource\.com/node_\$\{NODE_MAJOR\}\.x nodistro main'
assert_contains "${BOOTSTRAP_SCRIPT}" 'apt-get install -y nodejs'
assert_contains "${BOOTSTRAP_SCRIPT}" 'corepack prepare "pnpm@\$\{PNPM_VERSION\}" --activate'

assert_contains "${BOOTSTRAP_SCRIPT}" 'https://go\.dev/dl/go\$\{GO_VERSION\}\.linux-\$\{arch\}\.tar\.gz'
assert_contains "${BOOTSTRAP_SCRIPT}" 'ln -sf /usr/local/go/bin/go /usr/local/bin/go'
assert_contains "${BOOTSTRAP_SCRIPT}" 'Go is required to install air; enable INSTALL_GO or install Go first'
assert_contains "${BOOTSTRAP_SCRIPT}" 'GOBIN=/usr/local/bin "\$\{go_bin\}" install "github\.com/air-verse/air@\$\{AIR_VERSION\}"'

assert_contains "${BOOTSTRAP_SCRIPT}" 'npx --yes "playwright@\$\{PLAYWRIGHT_VERSION\}" install-deps chromium'
assert_contains "${BOOTSTRAP_SCRIPT}" 'npx --yes playwright@\$\{PLAYWRIGHT_VERSION\} install chromium'
assert_contains "${BOOTSTRAP_SCRIPT}" 'pnpm install --frozen-lockfile'

assert_contains "${MAKEFILE}" 'bootstrap-dev-ubuntu2404'
assert_contains "${MAKEFILE}" 'sudo bash infra/ops/bootstrap-dev-ubuntu2404\.sh'
assert_contains "${QUICKSTART}" 'make bootstrap-dev-ubuntu2404'

echo "[bootstrap-dev-ubuntu2404-contract] all assertions passed"
