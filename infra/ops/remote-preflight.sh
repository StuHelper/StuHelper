#!/usr/bin/env bash
# 远端部署预检脚本
#
# 在部署前验证目标机器的运行环境满足要求：
# - 必要命令行工具存在
# - Docker Compose 环境可用
# - 容器内 pg_dump/pg_basebackup 可用（备份容器化执行依赖）
# - Secret 后端配置正确
# - 生成文件和目录就绪
# - 备份 timer 已安装并启用
# - 备份数据库连接已配置
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

load_remote_deploy_config
require_cmd docker
require_cmd curl
require_cmd jq
require_cmd python3
require_cmd openssl

if [[ -n "${SHARED_ENV_SECRET_REF:-}" && -z "${SECRET_BACKEND:-}" ]]; then
  die "SECRET_BACKEND must be set when SHARED_ENV_SECRET_REF is provided"
fi
if [[ -n "${SECRETS_ENV_SECRET_REF:-}" && -z "${SECRET_BACKEND:-}" ]]; then
  die "SECRET_BACKEND must be set when SECRETS_ENV_SECRET_REF is provided"
fi

ensure_generated_files
if [[ -n "${SHARED_ENV_SECRET_REF:-}" ]]; then
  mkdir -p "$(dirname "${ENV_FILE}")"
  touch "${ENV_FILE}"
fi
if [[ -n "${SECRETS_ENV_SECRET_REF:-}" && -n "${SECRETS_ENV_FILE:-}" ]]; then
  mkdir -p "$(dirname "${SECRETS_ENV_FILE}")"
  touch "${SECRETS_ENV_FILE}"
fi

pending_generated_secret_ref="${GENERATED_ENV_SECRET_REF:-}"
unset GENERATED_ENV_SECRET_REF
load_env
if [[ -n "${pending_generated_secret_ref}" ]]; then
  export GENERATED_ENV_SECRET_REF="${pending_generated_secret_ref}"
fi

# 校验 Docker 和 Docker Compose 运行环境
docker info >/dev/null
docker compose version >/dev/null

# 校验 postgres 容器内 pg_dump 和 pg_basebackup 可用性
# 备份脚本已容器化，不再依赖宿主机 pg_dump，但需要确保容器镜像中包含这些工具
log "正在校验 postgres 容器内备份工具可用性..."

pg_image="postgres:${POSTGRES_VERSION:-18.3-alpine}"

# 检查 pg_dump 存在且可执行
if ! docker run --rm "${pg_image}" which pg_dump >/dev/null 2>&1; then
  die "postgres 容器镜像 (${pg_image}) 中 pg_dump 不可用"
fi

# 检查 pg_basebackup 存在且可执行
if ! docker run --rm "${pg_image}" which pg_basebackup >/dev/null 2>&1; then
  die "postgres 容器镜像 (${pg_image}) 中 pg_basebackup 不可用"
fi

# 获取容器内 pg_dump 版本，与运行中的 PostgreSQL 版本做兼容性日志
pg_dump_version="$(docker run --rm "${pg_image}" pg_dump --version 2>/dev/null | head -1 || echo "unknown")"
log "容器内 pg_dump 版本: ${pg_dump_version}"

require_backup_object_storage_config

mkdir -p \
  "${REPO_ROOT}/backups/postgres/logical" \
  "${REPO_ROOT}/backups/postgres/base" \
  "${REPO_ROOT}/infra/generated/postgres/wal-archive" \
  "${DEPLOY_STATE_DIR}"

if command -v systemctl >/dev/null 2>&1; then
  for unit in     stuhelper-postgres-dump-backup.timer     stuhelper-postgres-basebackup.timer     stuhelper-postgres-backup-sync.timer; do
    if ! systemctl list-unit-files | grep -q "^${unit}"; then
      die "backup timer ${unit} is not installed on the target host"
    fi
    if ! systemctl is-enabled --quiet "${unit}"; then
      die "backup timer ${unit} is not enabled"
    fi
  done
fi
[[ -n "${BACKUP_DATABASE_URL:-}" ]] || die "BACKUP_DATABASE_URL must be configured"
[[ -n "${REPLICATION_DATABASE_URL:-}" ]] || die "REPLICATION_DATABASE_URL must be configured"

log "remote preflight checks passed"
