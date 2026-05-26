#!/usr/bin/env bash
set -euo pipefail

# shellcheck source=common.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/common.sh"

DEV_RUNTIME_DIR="${REPO_ROOT}/.run/dev"
DEV_PID_DIR="${DEV_RUNTIME_DIR}/pids"
DEV_LOG_DIR="${DEV_RUNTIME_DIR}/logs"
DEV_STATE_DIR="${DEV_RUNTIME_DIR}/state"
DEV_RUNTIME_ENV="${DEV_RUNTIME_DIR}/runtime.env"
TOOLS_BIN_DIR="${REPO_ROOT}/.tools/bin"
AIR_VERSION="${AIR_VERSION:-v1.61.7}"

ensure_dev_runtime_dirs() {
  mkdir -p "${DEV_PID_DIR}" "${DEV_LOG_DIR}" "${DEV_STATE_DIR}" "${TOOLS_BIN_DIR}"
}

pid_file() {
  echo "${DEV_PID_DIR}/$1.pid"
}

log_file() {
  echo "${DEV_LOG_DIR}/$1.log"
}

stamp_file() {
  echo "${DEV_STATE_DIR}/$1.sha256"
}

process_running() {
  local pid="$1"
  [[ -n "${pid}" ]] && kill -0 "${pid}" >/dev/null 2>&1
}

stop_managed_process() {
  local name="$1"
  local pidfile
  pidfile="$(pid_file "${name}")"

  if [[ ! -f "${pidfile}" ]]; then
    return 0
  fi

  local pid
  pid="$(cat "${pidfile}")"
  if process_running "${pid}"; then
    kill -TERM -"${pid}" >/dev/null 2>&1 || kill "${pid}" >/dev/null 2>&1 || true
    for _ in {1..20}; do
      if ! process_running "${pid}"; then
        break
      fi
      sleep 0.5
    done
    if process_running "${pid}"; then
      kill -9 -"${pid}" >/dev/null 2>&1 || kill -9 "${pid}" >/dev/null 2>&1 || true
    fi
  fi
  rm -f "${pidfile}"
}

start_managed_process() {
  local name="$1"
  local command="$2"
  local logfile
  local pidfile
  logfile="$(log_file "${name}")"
  pidfile="$(pid_file "${name}")"

  stop_managed_process "${name}"
  : >"${logfile}"
  local pid
  pid="$(
    python3 - "${logfile}" "${command}" <<'PY'
import os
import subprocess
import sys

logfile = sys.argv[1]
command = sys.argv[2]
with open(logfile, "ab", buffering=0) as fh:
    proc = subprocess.Popen(
        ["bash", "-lc", command],
        stdin=subprocess.DEVNULL,
        stdout=fh,
        stderr=subprocess.STDOUT,
        start_new_session=True,
    )
print(proc.pid)
PY
  )"
  echo "${pid}" >"${pidfile}"
  sleep 1
  if ! process_running "${pid}"; then
    tail -n 80 "${logfile}" >&2 || true
    die "${name} failed to start; see ${logfile}"
  fi
  log "started ${name} (pid=${pid}, log=${logfile})"
}

sha256_file_portable() {
  local file="$1"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "${file}" | awk '{print $1}'
  else
    shasum -a 256 "${file}" | awk '{print $1}'
  fi
}

ensure_node_toolchain() {
  require_cmd node
  local node_version
  node_version="$(node -p 'process.versions.node')"
  if ! node - "${node_version}" <<'EOF'
const version = process.argv[2]
const [major, minor] = version.split('.').map(Number)

const supported =
  (major === 20 && minor >= 19) ||
  (major === 22 && minor >= 18) ||
  (major === 24 && minor >= 0)

process.exit(supported ? 0 : 1)
EOF
  then
    die "unsupported Node.js ${node_version}; use Node 24 (preferred) or a supported LTS release matching ^20.19.0 || ^22.18.0 || ^24.0.0"
  fi
  require_cmd corepack
  corepack enable >/dev/null 2>&1 || true
  corepack prepare pnpm@10 --activate >/dev/null 2>&1
  require_cmd pnpm
}

ensure_pnpm_workspace() {
  local workdir="$1"
  local lockfile="$2"
  local stamp_name="$3"
  local install_cmd="$4"

  local desired_hash
  local current_hash=""
  desired_hash="$(sha256_file_portable "${lockfile}")"
  if [[ -f "$(stamp_file "${stamp_name}")" ]]; then
    current_hash="$(cat "$(stamp_file "${stamp_name}")")"
  fi

  if [[ "${desired_hash}" == "${current_hash}" ]]; then
    log "pnpm workspace cache is current: ${workdir}"
    return 0
  fi

  log "installing pnpm dependencies in ${workdir}"
  (
    cd "${workdir}"
    bash -lc "${install_cmd}"
  )
  echo "${desired_hash}" >"$(stamp_file "${stamp_name}")"
}

ensure_air() {
  local air_bin="${TOOLS_BIN_DIR}/air"
  if [[ ! -x "${air_bin}" ]]; then
    log "installing air ${AIR_VERSION}" >&2
    (
      cd "${REPO_ROOT}/server"
      GOBIN="${TOOLS_BIN_DIR}" go install "github.com/air-verse/air@${AIR_VERSION}"
    )
  fi
  echo "${air_bin}"
}

kill_all_dev_processes() {
  stop_managed_process backend
  stop_managed_process frontend
  stop_managed_process admin
}

kill_listener_if_matches() {
  local port="$1"
  local pattern="$2"
  local pid command

  if ! command -v lsof >/dev/null 2>&1; then
    return 0
  fi

  while IFS= read -r pid; do
    [[ -n "${pid}" ]] || continue
    command="$(ps -ww -p "${pid}" -o command= 2>/dev/null || true)"
    if [[ -n "${command}" && "${command}" == *"${pattern}"* ]]; then
      kill -TERM "${pid}" >/dev/null 2>&1 || true
      sleep 1
      kill -9 "${pid}" >/dev/null 2>&1 || true
    fi
  done < <(lsof -tiTCP:"${port}" -sTCP:LISTEN 2>/dev/null || true)
}

is_port_available() {
  local port="$1"
  python3 - "$port" <<'PY'
import socket
import sys
port = int(sys.argv[1])
s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
try:
    s.bind(("127.0.0.1", port))
except OSError:
    sys.exit(1)
finally:
    s.close()
PY
}

port_is_published_by_container() {
  local port="$1"
  local container="$2"
  local published

  published="$(docker ps --filter "name=^/${container}$" --format '{{.Ports}}' 2>/dev/null || true)"
  [[ -n "${published}" ]] || return 1
  grep -Eq "(^|, )((127\\.0\\.0\\.1|0\\.0\\.0\\.0|\\[::\\]|::):)?${port}->" <<<"${published}"
}

pick_available_port() {
  local preferred="$1"
  local max="${2:-30}"
  shift 2 || true

  local -a reserved=("$@")
  local candidate reserved_port skip
  for ((candidate = preferred; candidate < preferred + max; candidate++)); do
    skip=false
    if ((${#reserved[@]} > 0)); then
      for reserved_port in "${reserved[@]}"; do
        if [[ -n "${reserved_port}" && "${candidate}" == "${reserved_port}" ]]; then
          skip=true
          break
        fi
      done
    fi
    if [[ "${skip}" == true ]]; then
      continue
    fi
    if is_port_available "${candidate}"; then
      echo "${candidate}"
      return 0
    fi
  done
  die "no free TCP port found starting at ${preferred}"
}

pick_available_or_current_container_port() {
  local preferred="$1"
  local max="$2"
  local container="$3"
  shift 3 || true
  local reserved_port

  for reserved_port in "$@"; do
    if [[ -n "${reserved_port}" && "${preferred}" == "${reserved_port}" ]]; then
      pick_available_port "${preferred}" "${max}" "$@"
      return
    fi
  done

  if is_port_available "${preferred}" || port_is_published_by_container "${preferred}" "${container}"; then
    printf '%s\n' "${preferred}"
    return
  fi

  pick_available_port "${preferred}" "${max}" "$@"
}

sync_dev_observability_ports() {
  local -a reserved=("$@")
  local stack="${STACK_NAME:-stuhelper-dev}"

  ALLOY_HTTP_PORT_SELECTED="$(pick_available_or_current_container_port "${ALLOY_HTTP_PORT:-12345}" 30 "${stack}-alloy" "${reserved[@]}")"
  reserved+=("${ALLOY_HTTP_PORT_SELECTED}")
  OTEL_GRPC_PORT_SELECTED="$(pick_available_or_current_container_port "${OTEL_GRPC_PORT:-4317}" 30 "${stack}-alloy" "${reserved[@]}")"
  reserved+=("${OTEL_GRPC_PORT_SELECTED}")
  OTEL_HTTP_PORT_SELECTED="$(pick_available_or_current_container_port "${OTEL_HTTP_PORT:-4318}" 30 "${stack}-alloy" "${reserved[@]}")"
  reserved+=("${OTEL_HTTP_PORT_SELECTED}")
  PROMETHEUS_PORT_SELECTED="$(pick_available_or_current_container_port "${PROMETHEUS_PORT:-9090}" 30 "${stack}-prometheus" "${reserved[@]}")"
  reserved+=("${PROMETHEUS_PORT_SELECTED}")
  ALERTMANAGER_PORT_SELECTED="$(pick_available_or_current_container_port "${ALERTMANAGER_PORT:-9093}" 30 "${stack}-alertmanager" "${reserved[@]}")"
  reserved+=("${ALERTMANAGER_PORT_SELECTED}")
  LOKI_PORT_SELECTED="$(pick_available_or_current_container_port "${LOKI_PORT:-3100}" 30 "${stack}-loki" "${reserved[@]}")"
  reserved+=("${LOKI_PORT_SELECTED}")
  TEMPO_HTTP_PORT_SELECTED="$(pick_available_or_current_container_port "${TEMPO_HTTP_PORT:-3200}" 30 "${stack}-tempo" "${reserved[@]}")"
  reserved+=("${TEMPO_HTTP_PORT_SELECTED}")
  GRAFANA_PORT_SELECTED="$(pick_available_or_current_container_port "${GRAFANA_PORT:-3003}" 30 "${stack}-grafana" "${reserved[@]}")"
  reserved+=("${GRAFANA_PORT_SELECTED}")
  CADVISOR_PORT_SELECTED="$(pick_available_or_current_container_port "${CADVISOR_PORT:-8088}" 30 "${stack}-cadvisor" "${reserved[@]}")"
  reserved+=("${CADVISOR_PORT_SELECTED}")
  POSTGRES_EXPORTER_PORT_SELECTED="$(pick_available_or_current_container_port "${POSTGRES_EXPORTER_PORT:-9187}" 30 "${stack}-postgres-exporter" "${reserved[@]}")"
  reserved+=("${POSTGRES_EXPORTER_PORT_SELECTED}")
  REDIS_EXPORTER_PORT_SELECTED="$(pick_available_or_current_container_port "${REDIS_EXPORTER_PORT:-9121}" 30 "${stack}-redis-exporter" "${reserved[@]}")"
  reserved+=("${REDIS_EXPORTER_PORT_SELECTED}")
  BLACKBOX_EXPORTER_PORT_SELECTED="$(pick_available_or_current_container_port "${BLACKBOX_EXPORTER_PORT:-9115}" 30 "${stack}-blackbox-exporter" "${reserved[@]}")"

  if [[ "${ALLOY_HTTP_PORT:-}" != "${ALLOY_HTTP_PORT_SELECTED}" ]]; then
    upsert_env_file "${ENV_FILE}" "ALLOY_HTTP_PORT" "${ALLOY_HTTP_PORT_SELECTED}"
  fi
  if [[ "${OTEL_GRPC_PORT:-}" != "${OTEL_GRPC_PORT_SELECTED}" ]]; then
    upsert_env_file "${ENV_FILE}" "OTEL_GRPC_PORT" "${OTEL_GRPC_PORT_SELECTED}"
  fi
  if [[ "${OTEL_HTTP_PORT:-}" != "${OTEL_HTTP_PORT_SELECTED}" ]]; then
    upsert_env_file "${ENV_FILE}" "OTEL_HTTP_PORT" "${OTEL_HTTP_PORT_SELECTED}"
  fi
  if [[ "${PROMETHEUS_PORT:-}" != "${PROMETHEUS_PORT_SELECTED}" ]]; then
    upsert_env_file "${ENV_FILE}" "PROMETHEUS_PORT" "${PROMETHEUS_PORT_SELECTED}"
  fi
  if [[ "${ALERTMANAGER_PORT:-}" != "${ALERTMANAGER_PORT_SELECTED}" ]]; then
    upsert_env_file "${ENV_FILE}" "ALERTMANAGER_PORT" "${ALERTMANAGER_PORT_SELECTED}"
  fi
  if [[ "${LOKI_PORT:-}" != "${LOKI_PORT_SELECTED}" ]]; then
    upsert_env_file "${ENV_FILE}" "LOKI_PORT" "${LOKI_PORT_SELECTED}"
  fi
  if [[ "${TEMPO_HTTP_PORT:-}" != "${TEMPO_HTTP_PORT_SELECTED}" ]]; then
    upsert_env_file "${ENV_FILE}" "TEMPO_HTTP_PORT" "${TEMPO_HTTP_PORT_SELECTED}"
  fi
  if [[ "${GRAFANA_PORT:-}" != "${GRAFANA_PORT_SELECTED}" ]]; then
    upsert_env_file "${ENV_FILE}" "GRAFANA_PORT" "${GRAFANA_PORT_SELECTED}"
  fi
  if [[ "${CADVISOR_PORT:-}" != "${CADVISOR_PORT_SELECTED}" ]]; then
    upsert_env_file "${ENV_FILE}" "CADVISOR_PORT" "${CADVISOR_PORT_SELECTED}"
  fi
  if [[ "${POSTGRES_EXPORTER_PORT:-}" != "${POSTGRES_EXPORTER_PORT_SELECTED}" ]]; then
    upsert_env_file "${ENV_FILE}" "POSTGRES_EXPORTER_PORT" "${POSTGRES_EXPORTER_PORT_SELECTED}"
  fi
  if [[ "${REDIS_EXPORTER_PORT:-}" != "${REDIS_EXPORTER_PORT_SELECTED}" ]]; then
    upsert_env_file "${ENV_FILE}" "REDIS_EXPORTER_PORT" "${REDIS_EXPORTER_PORT_SELECTED}"
  fi
  if [[ "${BLACKBOX_EXPORTER_PORT:-}" != "${BLACKBOX_EXPORTER_PORT_SELECTED}" ]]; then
    upsert_env_file "${ENV_FILE}" "BLACKBOX_EXPORTER_PORT" "${BLACKBOX_EXPORTER_PORT_SELECTED}"
  fi

  case "${GRAFANA_ROOT_URL:-}" in
    ""|http://localhost:*|http://127.0.0.1:*)
      upsert_env_file "${ENV_FILE}" "GRAFANA_ROOT_URL" "http://127.0.0.1:${GRAFANA_PORT_SELECTED}"
      ;;
  esac
}

write_dev_runtime_env() {
  local web_port="$1"
  local admin_port="$2"
  cat >"${DEV_RUNTIME_ENV}" <<EOF
API_BASE_URL=http://localhost:8080
WEB_BASE_URL=http://localhost:${web_port}
ADMIN_BASE_URL=http://localhost:${admin_port}
ADMIN_SMOKE_PATH=/admin/
WEB_DEV_PORT=${web_port}
ADMIN_EXTERNAL_PORT=${admin_port}
EOF
}
