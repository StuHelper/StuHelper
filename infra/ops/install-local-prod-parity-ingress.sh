#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

require_cmd awk
require_cmd nginx
require_cmd openssl
require_cmd sed
if [[ "${EUID}" -ne 0 ]]; then
  require_cmd sudo
fi

TEMPLATE="${REPO_ROOT}/infra/nginx/prod-parity-local-ingress.conf"
HOSTS_START="# StuHelper prod-parity local ingress BEGIN"
HOSTS_END="# StuHelper prod-parity local ingress END"
HOSTS_LINE="127.0.0.1 stuhelper.com www.stuhelper.com join.stuhelper.com sso.stuhelper.com"
PROXY_BYPASS_HOSTS=(stuhelper.com www.stuhelper.com join.stuhelper.com sso.stuhelper.com "*.stuhelper.com")
DEFAULT_BAOTA_TLS_CERT="/www/server/panel/vhost/cert/panel212.stuhelper.com/fullchain.pem"
DEFAULT_BAOTA_TLS_KEY="/www/server/panel/vhost/cert/panel212.stuhelper.com/privkey.pem"
DEFAULT_GENERATED_TLS_DIR="${PROD_PARITY_LOCAL_TLS_DIR:-${REPO_ROOT}/.run/prod-parity/local-tls}"
DEFAULT_GENERATED_TLS_CERT="${DEFAULT_GENERATED_TLS_DIR}/stuhelper-local.crt"
DEFAULT_GENERATED_TLS_KEY="${DEFAULT_GENERATED_TLS_DIR}/stuhelper-local.key"

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

install_proxy_bypass() {
  if ! command -v gsettings >/dev/null 2>&1 || ! command -v python3 >/dev/null 2>&1; then
    return
  fi
  if ! gsettings get org.gnome.system.proxy ignore-hosts >/dev/null 2>&1; then
    return
  fi

  python3 - "${PROXY_BYPASS_HOSTS[@]}" <<'PY'
import ast
import subprocess
import sys

hosts = sys.argv[1:]
raw = subprocess.check_output(
    ["gsettings", "get", "org.gnome.system.proxy", "ignore-hosts"],
    text=True,
).strip()
try:
    values = ast.literal_eval(raw)
except Exception:
    values = []
if not isinstance(values, list):
    values = []

changed = False
for host in hosts:
    if host not in values:
        values.append(host)
        changed = True

if changed:
    rendered = "[" + ", ".join(repr(value) for value in values) + "]"
    subprocess.check_call(
        ["gsettings", "set", "org.gnome.system.proxy", "ignore-hosts", rendered],
    )
PY
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

  if local_tls_available; then
    render_tls_config
  fi
}

local_tls_cert() {
  if [[ -n "${PROD_PARITY_LOCAL_TLS_CERT:-}" ]]; then
    printf '%s\n' "${PROD_PARITY_LOCAL_TLS_CERT}"
    return
  fi
  if root_test -f "${DEFAULT_BAOTA_TLS_CERT}" && root_test -f "${DEFAULT_BAOTA_TLS_KEY}"; then
    printf '%s\n' "${DEFAULT_BAOTA_TLS_CERT}"
    return
  fi
  printf '%s\n' "${DEFAULT_GENERATED_TLS_CERT}"
}

local_tls_key() {
  if [[ -n "${PROD_PARITY_LOCAL_TLS_KEY:-}" ]]; then
    printf '%s\n' "${PROD_PARITY_LOCAL_TLS_KEY}"
    return
  fi
  if root_test -f "${DEFAULT_BAOTA_TLS_CERT}" && root_test -f "${DEFAULT_BAOTA_TLS_KEY}"; then
    printf '%s\n' "${DEFAULT_BAOTA_TLS_KEY}"
    return
  fi
  printf '%s\n' "${DEFAULT_GENERATED_TLS_KEY}"
}

ensure_local_tls() {
  local cert key
  cert="$(local_tls_cert)"
  key="$(local_tls_key)"
  if root_test -f "${cert}" && root_test -f "${key}"; then
    return
  fi
  mkdir -p "$(dirname "${cert}")"
  openssl req \
    -x509 \
    -nodes \
    -newkey rsa:2048 \
    -sha256 \
    -days 825 \
    -subj "/CN=stuhelper.com" \
    -addext "subjectAltName=DNS:stuhelper.com,DNS:www.stuhelper.com,DNS:join.stuhelper.com,DNS:sso.stuhelper.com" \
    -keyout "${key}" \
    -out "${cert}" \
    >/dev/null 2>&1
  chmod 0644 "${cert}"
  chmod 0600 "${key}"
  log "generated local prod-parity TLS certificate: ${cert}"
}

local_tls_available() {
  local cert key
  cert="$(local_tls_cert)"
  key="$(local_tls_key)"
  root_test -f "${cert}" && root_test -f "${key}"
}

render_tls_config() {
  local cert key web_port admin_port backend_port casdoor_port
  cert="$(local_tls_cert)"
  key="$(local_tls_key)"
  web_port="${WEB_EXTERNAL_PORT:-28000}"
  admin_port="${ADMIN_EXTERNAL_PORT:-28001}"
  backend_port="${BACKEND_EXTERNAL_PORT:-28080}"
  casdoor_port="${CASDOOR_EXTERNALPORT:-28085}"

  cat <<NGINX

server {
    listen 443 ssl;
    listen [::]:443 ssl;
    http2 on;
    server_name stuhelper.com www.stuhelper.com;

    ssl_certificate ${cert};
    ssl_certificate_key ${key};
    ssl_protocols TLSv1.2 TLSv1.3;

    client_max_body_size 50m;

    proxy_set_header Host \$host;
    proxy_set_header X-Real-IP \$remote_addr;
    proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto \$scheme;
    proxy_set_header X-Forwarded-Host \$host;

    location = /verify {
        return 404;
    }

    location ^~ /verify/ {
        return 404;
    }

    location ^~ /api/v1/admission/freshman/camera-handoffs/ {
        proxy_pass http://127.0.0.1:${backend_port};
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        proxy_buffering off;
        proxy_cache off;
        proxy_read_timeout 3600s;
        proxy_send_timeout 3600s;
        add_header X-Accel-Buffering no always;
    }

    location = /api/v1/bot/admission/actions/stream {
        proxy_pass http://127.0.0.1:${backend_port};
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        proxy_buffering off;
        proxy_cache off;
        proxy_read_timeout 3600s;
        proxy_send_timeout 3600s;
        add_header X-Accel-Buffering no always;
    }

    location ^~ /api/ {
        proxy_pass http://127.0.0.1:${backend_port};
        proxy_http_version 1.1;
        proxy_buffering off;
        proxy_read_timeout 300s;
        proxy_send_timeout 300s;
    }

    location = /health {
        proxy_pass http://127.0.0.1:${backend_port};
        proxy_http_version 1.1;
    }

    location ^~ /health/ {
        proxy_pass http://127.0.0.1:${backend_port};
        proxy_http_version 1.1;
    }

    location = /metrics {
        proxy_pass http://127.0.0.1:${backend_port};
        proxy_http_version 1.1;
    }

    location ^~ /docs/ {
        proxy_pass http://127.0.0.1:${backend_port};
        proxy_http_version 1.1;
    }

    location = /admin {
        return 301 /admin/;
    }

    location = /_app.config.js {
        return 404;
    }

    location ^~ /css/ {
        return 404;
    }

    location ^~ /js/ {
        return 404;
    }

    location ^~ /jse/ {
        return 404;
    }

    location ^~ /admin/ {
        proxy_pass http://127.0.0.1:${admin_port};
        proxy_http_version 1.1;
    }

    location / {
        proxy_pass http://127.0.0.1:${web_port};
        proxy_http_version 1.1;
    }
}

server {
    listen 443 ssl;
    listen [::]:443 ssl;
    http2 on;
    server_name join.stuhelper.com;

    ssl_certificate ${cert};
    ssl_certificate_key ${key};
    ssl_protocols TLSv1.2 TLSv1.3;

    client_max_body_size 50m;

    proxy_set_header Host \$host;
    proxy_set_header X-Real-IP \$remote_addr;
    proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto \$scheme;
    proxy_set_header X-Forwarded-Host \$host;

    location = /verify {
        return 404;
    }

    location ^~ /verify/ {
        proxy_pass http://127.0.0.1:${web_port};
        proxy_http_version 1.1;
    }

    location ^~ /admission/freshman/camera/ {
        proxy_pass http://127.0.0.1:${web_port};
        proxy_http_version 1.1;
    }

    location ^~ /api/v1/admission/freshman/camera-handoffs/ {
        proxy_pass http://127.0.0.1:${backend_port};
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        proxy_buffering off;
        proxy_cache off;
        proxy_read_timeout 3600s;
        proxy_send_timeout 3600s;
        add_header X-Accel-Buffering no always;
    }

    location ^~ /api/ {
        proxy_pass http://127.0.0.1:${backend_port};
        proxy_http_version 1.1;
        proxy_buffering off;
        proxy_read_timeout 300s;
        proxy_send_timeout 300s;
    }

    location = /health {
        proxy_pass http://127.0.0.1:${backend_port};
        proxy_http_version 1.1;
    }

    location ^~ /health/ {
        proxy_pass http://127.0.0.1:${backend_port};
        proxy_http_version 1.1;
    }

    location ^~ /assets/ {
        proxy_pass http://127.0.0.1:${web_port};
        proxy_http_version 1.1;
    }

    location / {
        return 404;
    }
}

server {
    listen 443 ssl;
    listen [::]:443 ssl;
    http2 on;
    server_name sso.stuhelper.com;

    ssl_certificate ${cert};
    ssl_certificate_key ${key};
    ssl_protocols TLSv1.2 TLSv1.3;

    client_max_body_size 50m;

    proxy_set_header Host \$host;
    proxy_set_header X-Real-IP \$remote_addr;
    proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto \$scheme;
    proxy_set_header X-Forwarded-Host \$host;

    location ^~ /.well-known/ {
        proxy_pass http://127.0.0.1:${casdoor_port};
        proxy_http_version 1.1;
        proxy_read_timeout 300s;
        proxy_send_timeout 300s;
    }

    location ^~ /api/ {
        proxy_pass http://127.0.0.1:${casdoor_port};
        proxy_http_version 1.1;
        proxy_buffering off;
        proxy_read_timeout 300s;
        proxy_send_timeout 300s;
    }

    location / {
        proxy_pass http://127.0.0.1:${casdoor_port};
        proxy_http_version 1.1;
        proxy_read_timeout 300s;
        proxy_send_timeout 300s;
    }
}
NGINX
}

install_nginx_config() {
  local target tmp
  target="$(nginx_target)"
  tmp="$(mktemp)"
  ensure_local_tls
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
install_proxy_bypass
install_nginx_config
log "local prod-parity hosts: stuhelper.com, join.stuhelper.com, sso.stuhelper.com -> 127.0.0.1"
