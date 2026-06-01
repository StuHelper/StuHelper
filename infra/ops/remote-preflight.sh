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
if [[ -z "${GENERATED_ENV_SECRET_REF:-}" ]]; then
  die "GENERATED_ENV_SECRET_REF must be set for production bootstrap"
fi
if [[ -z "${SECRET_BACKEND:-}" || "${SECRET_BACKEND:-}" == "none" || "${SECRET_BACKEND:-}" == "file" ]]; then
  die "production bootstrap requires a non-file secret backend for generated secrets"
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

require_production_postgres_ssl
require_public_ingress_config_preflight
require_public_identity_ingress_preflight
if [[ "${ADMISSION_PUBLIC_SMOKE_ENABLED:-true}" == "true" ]]; then
  ADMISSION_PUBLIC_SMOKE_RETRIES="${ADMISSION_PUBLIC_SMOKE_PREFLIGHT_RETRIES:-${ADMISSION_PUBLIC_SMOKE_RETRIES:-1}}" \
    "${SCRIPT_DIR}/admission-public-smoke.sh"
else
  warn "public admission smoke preflight skipped because ADMISSION_PUBLIC_SMOKE_ENABLED is not true"
fi
if [[ "${PUBLIC_WEB_AUTH_BROWSER_SMOKE_PREFLIGHT_ENABLED:-false}" == "true" ]]; then
  node "${SCRIPT_DIR}/public-web-auth-browser-smoke.mjs"
else
  warn "public Web auth browser smoke preflight skipped because PUBLIC_WEB_AUTH_BROWSER_SMOKE_PREFLIGHT_ENABLED is not true"
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
  "${POSTGRES_WAL_RESTORE_DIR}" \
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

# ── 恢复能力校验 ──
# 不仅检查配置存在，还验证恢复链路的实际可执行性

log "正在校验恢复能力..."

# 1. pg_restore 可用（恢复 custom-format dump 的核心工具）
if ! docker run --rm "${pg_image}" which pg_restore >/dev/null 2>&1; then
  die "postgres 容器镜像 (${pg_image}) 中 pg_restore 不可用"
fi
pg_restore_version="$(docker run --rm "${pg_image}" pg_restore --version 2>/dev/null | head -1 || echo "unknown")"
log "容器内 pg_restore 版本: ${pg_restore_version}"

# 2. 备份目录存在且可写
for dir in \
  "${REPO_ROOT}/backups/postgres/logical" \
  "${REPO_ROOT}/backups/postgres/base"; do
  if [[ ! -d "${dir}" ]]; then
    die "备份目录不存在: ${dir}"
  fi
  if [[ ! -w "${dir}" ]]; then
    die "备份目录不可写: ${dir}"
  fi
done

# 3. 最近一次逻辑备份存在（如果目录非空）
latest_dump=$(find "${REPO_ROOT}/backups/postgres/logical" -name "*.dump" -type f -printf '%T@ %p\n' 2>/dev/null | sort -rn | head -1 | cut -d' ' -f2-)
if [[ -n "${latest_dump}" ]]; then
  # 验证对应的 sha256 校验文件存在
  if [[ -f "${latest_dump}.sha256" ]]; then
    log "最近逻辑备份: $(basename "${latest_dump}"), sha256 校验文件存在"
  else
    log "⚠️  最近逻辑备份 $(basename "${latest_dump}") 缺少 .sha256 校验文件"
  fi
else
  log "⚠️  逻辑备份目录为空，尚无历史备份"
fi

# 4. BACKUP_DATABASE_URL 连通性（通过 Compose 网络快速超时检测）
if compose run --rm --no-deps -T postgres \
  pg_isready -d "${BACKUP_DATABASE_URL}" -t 5 >/dev/null 2>&1; then
  log "备份数据库连通性: OK"
else
  die "备份数据库连通性检查失败（pg_isready 超时），请确认 BACKUP_DATABASE_URL 可达"
fi

log "remote preflight checks passed"
