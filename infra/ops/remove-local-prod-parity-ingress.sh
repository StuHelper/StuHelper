#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

require_cmd awk
require_cmd nginx
require_cmd pgrep
if [[ "${EUID}" -ne 0 ]]; then
  require_cmd sudo
fi

HOSTS_START="# StuHelper prod-parity local ingress BEGIN"
HOSTS_END="# StuHelper prod-parity local ingress END"
PROXY_BYPASS_HOSTS=(stuhelper.com www.stuhelper.com join.stuhelper.com sso.stuhelper.com "*.stuhelper.com")
PROXY_BYPASS_STATE_FILE="${PROD_PARITY_LOCAL_PROXY_STATE_FILE:-${REPO_ROOT}/.run/prod-parity/local-ingress-proxy-ignore-hosts.before}"
NGINX_TARGETS=(
  /www/server/panel/vhost/nginx/stuhelper-prod-parity-local.conf
  /etc/nginx/conf.d/stuhelper-prod-parity-local.conf
)

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

validate_hosts_markers() {
  awk -v start="${HOSTS_START}" -v end="${HOSTS_END}" '
    $0 == start {
      starts++
      if (starts > 1 || open) exit 1
      open = 1
      next
    }
    $0 == end {
      ends++
      if (ends > 1 || !open) exit 1
      open = 0
      next
    }
    END {
      if (open || starts != ends) exit 1
    }
  ' /etc/hosts || die "malformed prod-parity marker block in /etc/hosts"
}

remove_hosts() {
  local tmp
  validate_hosts_markers
  tmp="$(mktemp)"
  awk -v start="${HOSTS_START}" -v end="${HOSTS_END}" '
    $0 == start { skipping = 1; next }
    $0 == end { skipping = 0; next }
    !skipping { print }
  ' /etc/hosts >"${tmp}"
  run_root install -m 0644 "${tmp}" /etc/hosts
  rm -f "${tmp}"
}

remove_proxy_bypass() {
  if ! command -v gsettings >/dev/null 2>&1 || ! command -v python3 >/dev/null 2>&1; then
    if [[ -e "${PROXY_BYPASS_STATE_FILE}" ]]; then
      warn "cannot restore GNOME proxy bypass without gsettings and python3; retained ${PROXY_BYPASS_STATE_FILE}"
    fi
    return
  fi
  if ! gsettings get org.gnome.system.proxy ignore-hosts >/dev/null 2>&1; then
    if [[ -e "${PROXY_BYPASS_STATE_FILE}" ]]; then
      warn "cannot access GNOME proxy settings; retained ${PROXY_BYPASS_STATE_FILE}"
    fi
    return
  fi

  python3 - "${PROXY_BYPASS_STATE_FILE}" "${PROXY_BYPASS_HOSTS[@]}" <<'PY'
import ast
from pathlib import Path
import subprocess
import sys

state_path = Path(sys.argv[1])
parity_hosts = set(sys.argv[2:])


def parse(raw: str) -> list[str]:
    raw = raw.strip()
    if raw.startswith("@as "):
        raw = raw[4:]
    try:
        values = ast.literal_eval(raw)
    except Exception as exc:
        raise SystemExit(f"cannot parse GNOME proxy ignore-hosts: {exc}") from exc
    if not isinstance(values, list) or not all(isinstance(value, str) for value in values):
        raise SystemExit("GNOME proxy ignore-hosts is not a string list")
    return values


current = parse(
    subprocess.check_output(
        ["gsettings", "get", "org.gnome.system.proxy", "ignore-hosts"],
        text=True,
    ),
)
original = parse(state_path.read_text()) if state_path.exists() else []
original_hosts = set(original)

# Only remove parity entries that the installer added. When an older installer
# left no state file, exact host removal is the safest recoverable fallback.
removable = parity_hosts - original_hosts if state_path.exists() else parity_hosts
filtered = [value for value in current if value not in removable]

if filtered != current:
    rendered = (
        "[" + ", ".join(repr(value) for value in filtered) + "]"
        if filtered
        else "@as []"
    )
    subprocess.check_call(
        ["gsettings", "set", "org.gnome.system.proxy", "ignore-hosts", rendered],
    )
PY

  if [[ -e "${PROXY_BYPASS_STATE_FILE}" ]]; then
    rm -f "${PROXY_BYPASS_STATE_FILE}"
  fi
}

remove_nginx_config() {
  local target
  local changed=false
  for target in "${NGINX_TARGETS[@]}"; do
    if root_test -f "${target}"; then
      run_root rm -f "${target}"
      changed=true
    fi
  done

  run_root nginx -t
  if [[ "${changed}" == "true" ]] && pgrep -x nginx >/dev/null 2>&1; then
    run_root nginx -s reload
  fi
}

remove_hosts
remove_proxy_bypass
remove_nginx_config
log "removed local prod-parity hosts, proxy bypass, and Nginx ingress"
