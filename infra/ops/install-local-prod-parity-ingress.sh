#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

require_cmd awk
require_cmd nginx
require_cmd sed
if [[ "${EUID}" -ne 0 ]]; then
  require_cmd sudo
fi

TEMPLATE="${REPO_ROOT}/infra/nginx/prod-parity-local-ingress.conf"
HOSTS_START="# StuHelper prod-parity local ingress BEGIN"
HOSTS_END="# StuHelper prod-parity local ingress END"
HOSTS_LINE="127.0.0.1 stuhelper.com www.stuhelper.com id.stuhelper.com sso.stuhelper.com"

run_root() {
  if [[ "${EUID}" -eq 0 ]]; then
    "$@"
    return
  fi
  sudo -n "$@"
}

root_test() {
  if [[ "${EUID}" -eq 0 ]]; then
    test "$@"
    return
  fi
  sudo -n test "$@"
}

install_hosts() {
  local tmp
  tmp="$(mktemp)"
  {
    printf '%s\n' "${HOSTS_START}"
    printf '%s\n' "${HOSTS_LINE}"
    printf '%s\n' "${HOSTS_END}"
    awk -v start="${HOSTS_START}" -v end="${HOSTS_END}" '
      $0 == start { skipping = 1; next }
      $0 == end { skipping = 0; next }
      !skipping { print }
    ' /etc/hosts
  } >"${tmp}"
  run_root install -m 0644 "${tmp}" /etc/hosts
  rm -f "${tmp}"
}

nginx_target() {
  if root_test -d /www/server/panel/vhost/nginx && root_test -f /www/server/nginx/conf/nginx.conf; then
    printf '%s\n' "/www/server/panel/vhost/nginx/stuhelper-prod-parity-local.conf"
    return
  fi
  printf '%s\n' "/etc/nginx/conf.d/stuhelper-prod-parity-local.conf"
}

render_config() {
  sed \
    -e "s/__WEB_PORT__/${WEB_EXTERNAL_PORT:-28000}/g" \
    -e "s/__ADMIN_PORT__/${ADMIN_EXTERNAL_PORT:-28001}/g" \
    -e "s/__BACKEND_PORT__/${BACKEND_EXTERNAL_PORT:-28080}/g" \
    -e "s/__CASDOOR_PORT__/${CASDOOR_EXTERNALPORT:-28085}/g" \
    "${TEMPLATE}"
}

install_nginx_config() {
  local target tmp
  target="$(nginx_target)"
  tmp="$(mktemp)"
  render_config >"${tmp}"
  run_root install -d -m 0755 "$(dirname "${target}")"
  run_root install -m 0644 "${tmp}" "${target}"
  if [[ "${target}" != "/etc/nginx/conf.d/stuhelper-prod-parity-local.conf" ]] && root_test -f /etc/nginx/conf.d/stuhelper-prod-parity-local.conf; then
    run_root rm -f /etc/nginx/conf.d/stuhelper-prod-parity-local.conf
  fi
  rm -f "${tmp}"
  run_root nginx -t
  run_root nginx -s reload
  log "installed local prod-parity ingress: ${target}"
}

install_hosts
install_nginx_config
log "local prod-parity hosts: stuhelper.com, id.stuhelper.com, sso.stuhelper.com -> 127.0.0.1"
