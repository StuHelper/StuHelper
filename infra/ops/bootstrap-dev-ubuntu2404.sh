#!/usr/bin/env bash
set -euo pipefail

DEV_USER="${DEV_USER:-${SUDO_USER:-${USER:-}}}"
GO_VERSION="${GO_VERSION:-1.26.0}"
NODE_MAJOR="${NODE_MAJOR:-24}"
PNPM_VERSION="${PNPM_VERSION:-10.32.1}"
AIR_VERSION="${AIR_VERSION:-v1.61.7}"
PLAYWRIGHT_VERSION="${PLAYWRIGHT_VERSION:-1.58.2}"
INSTALL_DOCKER="${INSTALL_DOCKER:-true}"
INSTALL_NODE="${INSTALL_NODE:-true}"
INSTALL_GO="${INSTALL_GO:-true}"
INSTALL_AIR="${INSTALL_AIR:-true}"
INSTALL_PLAYWRIGHT="${INSTALL_PLAYWRIGHT:-true}"
INSTALL_WORKSPACE_DEPS="${INSTALL_WORKSPACE_DEPS:-false}"

log() {
  echo "[bootstrap-dev-ubuntu2404] $*"
}

die() {
  echo "[bootstrap-dev-ubuntu2404][error] $*" >&2
  exit 1
}

usage() {
  cat <<'USAGE'
Usage: sudo bash infra/ops/bootstrap-dev-ubuntu2404.sh

Bootstraps an Ubuntu 24.04 workstation for StuHelper local development.

Optional env:
  DEV_USER                 Target non-root user. Defaults to SUDO_USER.
  GO_VERSION               Defaults to 1.26.0.
  NODE_MAJOR               Defaults to 24.
  PNPM_VERSION             Defaults to 10.32.1.
  AIR_VERSION              Defaults to v1.61.7.
  PLAYWRIGHT_VERSION       Defaults to 1.58.2.
  INSTALL_DOCKER           true/false, defaults to true.
  INSTALL_NODE             true/false, defaults to true.
  INSTALL_GO               true/false, defaults to true.
  INSTALL_AIR              true/false, defaults to true.
  INSTALL_PLAYWRIGHT       true/false, defaults to true.
  INSTALL_WORKSPACE_DEPS   true/false, defaults to false. When true, runs
                           pnpm install for clients and clients/admin.
USAGE
}

require_root() {
  if [[ "${EUID}" -ne 0 ]]; then
    die "run as root (sudo bash infra/ops/bootstrap-dev-ubuntu2404.sh)"
  fi
}

validate_bool() {
  local name="$1"
  local value="${2:-}"
  case "${value}" in
    true|false) ;;
    *) die "${name} must be true or false" ;;
  esac
}

require_dev_user() {
  [[ -n "${DEV_USER}" ]] || die "DEV_USER is required when SUDO_USER is not set"
  id -u "${DEV_USER}" >/dev/null 2>&1 || die "DEV_USER does not exist: ${DEV_USER}"
}

apt_install() {
  export DEBIAN_FRONTEND=noninteractive
  apt-get update -y
  apt-get install -y "$@"
}

ensure_base_packages() {
  apt_install \
    ca-certificates \
    curl \
    git \
    gnupg \
    jq \
    lsof \
    make \
    openssl \
    python3 \
    tar \
    unzip \
    xz-utils \
    build-essential
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

install_docker() {
  [[ "${INSTALL_DOCKER}" == "true" ]] || return 0

  log "installing Docker Engine and Compose plugin"
  ensure_docker_repo
  apt-get update -y
  apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
  systemctl enable --now docker
  if [[ "${DEV_USER}" != "root" ]]; then
    usermod -aG docker "${DEV_USER}"
  fi
}

ensure_nodesource_repo() {
  install -m 0755 -d /etc/apt/keyrings
  if [[ ! -f /etc/apt/keyrings/nodesource.gpg ]]; then
    curl -fsSL https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key | \
      gpg --dearmor -o /etc/apt/keyrings/nodesource.gpg
    chmod a+r /etc/apt/keyrings/nodesource.gpg
  fi

  cat >/etc/apt/sources.list.d/nodesource.list <<EOF
deb [signed-by=/etc/apt/keyrings/nodesource.gpg] https://deb.nodesource.com/node_${NODE_MAJOR}.x nodistro main
EOF
}

install_node() {
  [[ "${INSTALL_NODE}" == "true" ]] || return 0

  log "installing Node.js ${NODE_MAJOR}.x and pnpm ${PNPM_VERSION}"
  ensure_nodesource_repo
  apt-get update -y
  apt-get install -y nodejs
  corepack enable
  corepack prepare "pnpm@${PNPM_VERSION}" --activate
}

go_arch() {
  case "$(dpkg --print-architecture)" in
    amd64) printf '%s\n' amd64 ;;
    arm64) printf '%s\n' arm64 ;;
    *) die "unsupported Go architecture: $(dpkg --print-architecture)" ;;
  esac
}

install_go() {
  [[ "${INSTALL_GO}" == "true" ]] || return 0

  local arch tmp_tarball
  arch="$(go_arch)"
  tmp_tarball="$(mktemp)"
  log "installing Go ${GO_VERSION} for linux-${arch}"
  curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${arch}.tar.gz" -o "${tmp_tarball}"
  rm -rf /usr/local/go
  tar -C /usr/local -xzf "${tmp_tarball}"
  rm -f "${tmp_tarball}"
  ln -sf /usr/local/go/bin/go /usr/local/bin/go
  ln -sf /usr/local/go/bin/gofmt /usr/local/bin/gofmt
  cat >/etc/profile.d/stuhelper-dev.sh <<'EOF'
export PATH="/usr/local/go/bin:${HOME}/go/bin:${PATH}"
EOF
  chmod 0644 /etc/profile.d/stuhelper-dev.sh
}

go_binary() {
  if [[ -x /usr/local/go/bin/go ]]; then
    printf '%s\n' /usr/local/go/bin/go
    return 0
  fi
  if command -v go >/dev/null 2>&1; then
    command -v go
    return 0
  fi
  die "Go is required to install air; enable INSTALL_GO or install Go first"
}

install_air() {
  [[ "${INSTALL_AIR}" == "true" ]] || return 0

  local go_bin
  go_bin="$(go_binary)"
  log "installing air ${AIR_VERSION}"
  GOBIN=/usr/local/bin "${go_bin}" install "github.com/air-verse/air@${AIR_VERSION}"
}

run_as_dev_user() {
  local command="$1"
  su -s /bin/bash "${DEV_USER}" -c "export PATH=/usr/local/go/bin:/usr/local/bin:\${HOME}/go/bin:\${PATH}; ${command}"
}

install_playwright() {
  [[ "${INSTALL_PLAYWRIGHT}" == "true" ]] || return 0

  log "installing Playwright Chromium OS dependencies and browser cache"
  npx --yes "playwright@${PLAYWRIGHT_VERSION}" install-deps chromium
  run_as_dev_user "npx --yes playwright@${PLAYWRIGHT_VERSION} install chromium"
}

install_workspace_deps() {
  [[ "${INSTALL_WORKSPACE_DEPS}" == "true" ]] || return 0

  local repo_root
  repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
  log "installing StuHelper pnpm workspaces"
  run_as_dev_user "cd '${repo_root}/clients' && corepack enable && corepack prepare pnpm@${PNPM_VERSION} --activate && pnpm install --frozen-lockfile"
  run_as_dev_user "cd '${repo_root}/clients/admin' && corepack enable && corepack prepare pnpm@${PNPM_VERSION} --activate && pnpm install --frozen-lockfile"
}

print_versions() {
  log "tool versions"
  docker --version || true
  docker compose version || true
  go version || /usr/local/go/bin/go version || true
  node --version || true
  pnpm --version || true
  air -v || true
}

main() {
  if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
    usage
    exit 0
  fi

  require_root
  require_dev_user
  validate_bool INSTALL_DOCKER "${INSTALL_DOCKER}"
  validate_bool INSTALL_NODE "${INSTALL_NODE}"
  validate_bool INSTALL_GO "${INSTALL_GO}"
  validate_bool INSTALL_AIR "${INSTALL_AIR}"
  validate_bool INSTALL_PLAYWRIGHT "${INSTALL_PLAYWRIGHT}"
  validate_bool INSTALL_WORKSPACE_DEPS "${INSTALL_WORKSPACE_DEPS}"

  log "installing base packages"
  ensure_base_packages
  install_docker
  install_go
  install_node
  install_air
  install_playwright
  install_workspace_deps
  print_versions

  cat <<EOF

[bootstrap-dev-ubuntu2404] completed
Developer user: ${DEV_USER}

Next steps:
1. Open a new shell so docker group and PATH changes apply.
2. Run: make dev-init
3. Run: make dev-up
4. Run: make dev-status
EOF
}

main "$@"
