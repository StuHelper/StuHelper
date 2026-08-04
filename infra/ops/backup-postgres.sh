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
# 备份先写入同步目录之外的 staging，再以 .partial 名称复制到目标文件系统并原子发布；
# 成功后同时生成 <output-file>.sha256 校验文件。
set -euo pipefail
umask 077

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

if [[ $# -lt 1 ]]; then
  echo "用法: $0 <output-file>" >&2
  exit 1
fi

mode="${BACKUP_MODE:-dump}"
load_env_preserving \
  BACKUP_DATABASE_URL \
  REPLICATION_DATABASE_URL \
  LOCAL_STATE_DIR \
  BACKUP_STAGING_DIR
logical_url="${BACKUP_DATABASE_URL:-}"
replication_url="${REPLICATION_DATABASE_URL:-}"

if [[ "${APP_ENV:-}" == "production" && "${EXTERNAL_POSTGRES_ENABLED:-false}" != "true" ]]; then
  require_live_canonical_postgres_datastore
  require_internal_postgres_backup_sources_match_live_datastore
fi

if [[ "${mode}" == "dump" && -z "${logical_url}" ]]; then
  die "BACKUP_DATABASE_URL 未配置"
fi

if [[ "${mode}" == "basebackup" && -z "${replication_url}" ]]; then
  die "REPLICATION_DATABASE_URL 未配置"
fi

requested_output_file="$1"
output_dir="$(dirname "${requested_output_file}")"
output_name="$(basename "${requested_output_file}")"
[[ "${output_name}" != "." && "${output_name}" != ".." ]] ||
  die "invalid backup output file: ${requested_output_file}"

mkdir -p "${output_dir}"
output_dir="$(cd "${output_dir}" && pwd -P)"
output_file="${output_dir}/${output_name}"
checksum_file="${output_file}.sha256"

if [[ -e "${output_file}" || -L "${output_file}" || -e "${checksum_file}" || -L "${checksum_file}" ]]; then
  die "refusing to overwrite an existing backup artifact: ${output_file}"
fi

# Keep staging outside both rclone source directories. The systemd installer
# pins this to /var/lib/stuhelper/postgres/backup-staging; direct invocations
# use the current user's local state directory unless explicitly overridden.
staging_root="${BACKUP_STAGING_DIR:-${LOCAL_STATE_DIR}/postgres/backup-staging}"
mkdir -p "${staging_root}"
staging_root="$(cd "${staging_root}" && pwd -P)"
work_dir="$(mktemp -d "${staging_root}/backup.XXXXXX")"
staged_file="${work_dir}/${output_name}"
staged_checksum="${staged_file}.sha256"
partial_file="${output_file}.partial.${BASHPID}"
partial_checksum="${checksum_file}.partial.${BASHPID}"
backup_published=false
checksum_published=false

cleanup() {
  rm -f -- "${partial_file}" "${partial_checksum}"
  if [[ "${checksum_published}" == "true" && "${backup_published}" != "true" ]]; then
    rm -f -- "${checksum_file}"
  fi
  rm -rf -- "${work_dir}"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

# 使用 docker compose run 在最小权限 postgres-client 容器内执行备份命令
# --rm：执行完成后自动清理容器
# --no-deps：不启动依赖服务（postgres 本身已在运行，新容器通过 Docker 网络访问）
# -T：不分配伪终端
run_dump() {
  require_cmd docker
  log "正在通过容器化 pg_dump 导出数据库..."

  compose run --rm --no-deps -T \
    postgres-client \
    pg_dump --format=custom --no-owner --no-privileges "${logical_url}" \
    >"${staged_file}"
}

run_basebackup() {
  require_cmd docker
  require_cmd tar
  log "正在通过容器化 pg_basebackup 创建物理备份..."

  local host_uid host_gid container_user pgdata_dir
  host_uid="$(id -u)"
  host_gid="$(id -g)"
  container_user="${host_uid}:${host_gid}"

  # systemd timers normally run as the non-root deploy user. When an operator
  # invokes the script as root, keep the database client at the image's
  # dedicated uid/gid instead of overriding it to root.
  if [[ "${host_uid}" == "0" ]]; then
    container_user="70:70"
    chown 70:70 "${work_dir}"
  fi

  pgdata_dir="${work_dir}/pgdata"
  compose run --rm --no-deps -T \
    --user "${container_user}" \
    --workdir /backup \
    --volume "${work_dir}:/backup" \
    postgres-client \
    pg_basebackup \
      --dbname "${replication_url}" \
      --format=plain \
      --wal-method=stream \
      --checkpoint=fast \
      --pgdata=/backup/pgdata

  # pg_verifybackup validates the manifest and every file before the archive is
  # published. WAL streaming uses pg_basebackup's temporary replication slot;
  # no persistent slot lifecycle is introduced here.
  compose run --rm --no-deps -T \
    --user "${container_user}" \
    --workdir /backup \
    --volume "${work_dir}:/backup:ro" \
    postgres-client \
    pg_verifybackup /backup/pgdata

  tar -C "${pgdata_dir}" -czf "${staged_file}" .
  tar -tzf "${staged_file}" >/dev/null
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
if [[ ! -s "${staged_file}" ]]; then
  die "备份文件为空，可能存在异常: ${output_file}"
fi

# sidecar 只记录 basename，取回到其他目录后仍可验证。先发布 sidecar，
# 再原子发布完整备份；若后一步失败，EXIT trap 会移除孤立 sidecar。
digest="$(sha256sum "${staged_file}" | awk '{print $1}')"
printf '%s  %s\n' "${digest}" "${output_name}" >"${staged_checksum}"

mv -- "${staged_checksum}" "${partial_checksum}"
mv -- "${partial_checksum}" "${checksum_file}"
checksum_published=true

mv -- "${staged_file}" "${partial_file}"
mv -- "${partial_file}" "${output_file}"
backup_published=true

log "备份完成: ${output_file}"
log "校验和已写入: ${checksum_file}"
