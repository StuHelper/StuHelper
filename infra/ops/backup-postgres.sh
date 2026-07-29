#!/usr/bin/env bash
# PostgreSQL 备份脚本（容器化执行）
#
# 所有 pg_dump / pg_basebackup 命令均通过只挂载客户端 CA 的
# postgres-client 一次性容器执行，不依赖宿主机安装 PostgreSQL 客户端，也不让
# 备份任务继承数据库数据卷、服务端私钥或初始化凭据。
#
# 用法: BACKUP_MODE=dump ./backup-postgres.sh <output-file>
#       BACKUP_MODE=basebackup ./backup-postgres.sh <output-file>
#
# 备份产物写入 <output-file>，同时生成 <output-file>.sha256 校验文件。
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

if [[ $# -lt 1 ]]; then
  echo "用法: $0 <output-file>" >&2
  exit 1
fi

mode="${BACKUP_MODE:-dump}"
load_env_preserving BACKUP_DATABASE_URL REPLICATION_DATABASE_URL
logical_url="${BACKUP_DATABASE_URL:-}"
replication_url="${REPLICATION_DATABASE_URL:-}"

if [[ "${mode}" == "dump" && -z "${logical_url}" ]]; then
  die "BACKUP_DATABASE_URL 未配置"
fi

if [[ "${mode}" == "basebackup" && -z "${replication_url}" ]]; then
  die "REPLICATION_DATABASE_URL 未配置"
fi

output_file="$1"
mkdir -p "$(dirname "$output_file")"

# 使用 docker compose run 在最小权限 postgres-client 容器内执行备份命令
# --rm：执行完成后自动清理容器
# --no-deps：不启动依赖服务（postgres 本身已在运行，新容器通过 Docker 网络访问）
# -T：不分配伪终端（支持管道输出）
run_dump() {
  require_cmd docker
  log "正在通过容器化 pg_dump 导出数据库..."

  compose run --rm --no-deps -T \
    postgres-client \
    pg_dump --format=custom --no-owner --no-privileges "${logical_url}" \
    >"$output_file"
}

run_basebackup() {
  require_cmd docker
  log "正在通过容器化 pg_basebackup 创建物理备份..."

  compose run --rm --no-deps -T \
    postgres-client \
    pg_basebackup \
      --dbname "${replication_url}" \
      --format=tar \
      --gzip \
      --wal-method=stream \
      --checkpoint=fast \
      --pgdata=- \
    >"$output_file"
}

case "${mode}" in
  dump)
    log "开始逻辑备份: ${output_file}"
    run_dump
    ;;
  basebackup)
    log "开始物理基础备份: ${output_file}"
    run_basebackup
    ;;
  *)
    die "不支持的 BACKUP_MODE=${mode}（可选: dump, basebackup）"
    ;;
esac

# 验证备份文件非空
if [[ ! -s "$output_file" ]]; then
  die "备份文件为空，可能存在异常: ${output_file}"
fi

# 生成 SHA256 校验和
sha256sum "$output_file" > "$output_file.sha256"
log "备份完成: ${output_file}"
log "校验和已写入: ${output_file}.sha256"
