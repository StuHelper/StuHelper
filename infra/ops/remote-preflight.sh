#!/usr/bin/env bash
# 远端部署预检脚本
#
# 在部署前验证目标机器的运行环境满足要求：
# - 必要命令行工具存在
# - Docker Compose 环境可用
# - postgres-client 镜像内 pg_dump/pg_basebackup 可用（备份容器化执行依赖）
# - Secret 后端配置正确
# - 生成文件和目录就绪
# - 备份 timer 已安装并启用
# - 备份数据库连接已配置
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

preflight_phase="${1:---full}"
[[ "$#" -le 1 ]] || die "usage: remote-preflight.sh [--pre-deploy|--post-deploy|--timer-activation|--full]"
case "${preflight_phase}" in
  --pre-deploy | --post-deploy | --timer-activation | --full) ;;
  *) die "usage: remote-preflight.sh [--pre-deploy|--post-deploy|--timer-activation|--full]" ;;
esac

load_remote_deploy_config
require_protected_backup_environment_paths
BACKUP_STAGING_DIR="${BACKUP_STAGING_DIR:-/var/lib/stuhelper/postgres/backup-staging}"
require_cmd docker
require_cmd curl
require_cmd jq
require_cmd python3
require_cmd openssl
require_cmd systemctl
require_cmd id
require_cmd getent
require_no_legacy_backup_timer_units

backup_service_user="$(id -un)"
backup_service_uid="$(id -u)"
backup_service_group="${BACKUP_SERVICE_GROUP:-}"
backup_service_group_record=""
backup_service_gid=""
[[ "${backup_service_user}" != "root" && "${EUID}" -ne 0 && "${backup_service_uid}" != "0" ]] ||
  die "remote production preflight must run as the non-root deploy user"
[[ -n "${backup_service_group}" && "${backup_service_group}" != "root" && \
  "${backup_service_group}" != "0" ]] ||
  die "remote production preflight requires BACKUP_SERVICE_GROUP to name the configured non-root service group"
backup_service_group_record="$(getent group "${backup_service_group}")" ||
  die "configured backup service group does not exist: ${backup_service_group}"
IFS=: read -r _ _ backup_service_gid _ <<<"${backup_service_group_record}"
[[ "${backup_service_gid}" =~ ^[0-9]+$ && "${backup_service_gid}" != "0" ]] ||
  die "configured backup service group must not resolve to gid 0"

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
if [[ "${SECRET_BACKEND}" == "vault-kv-v2" ]]; then
  require_cmd systemctl
  [[ -x "${SCRIPT_DIR}/vault-runtime-token.sh" ]] ||
    die "Vault runtime token checker is missing or not executable"
  "${SCRIPT_DIR}/vault-runtime-token.sh" check
  if ! systemd_unit_file_is_installed stuhelper-vault-token-renewal.timer; then
    die "Vault runtime token renewal timer is not installed on the target host"
  fi
  systemctl is-enabled --quiet stuhelper-vault-token-renewal.timer ||
    die "Vault runtime token renewal timer is not enabled"
  systemctl is-active --quiet stuhelper-vault-token-renewal.timer ||
    die "Vault runtime token renewal timer is not active"
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
load_env_preserving NGINX_PUBLIC_INGRESS_CONFIG_FILE
if [[ -n "${pending_generated_secret_ref}" ]]; then
  export GENERATED_ENV_SECRET_REF="${pending_generated_secret_ref}"
fi

[[ "${APP_ENV:-}" == "production" ]] || die "APP_ENV must be production for remote preflight"
docker info >/dev/null
docker compose version >/dev/null
configure_production_preflight_runtime_checks "${preflight_phase}"

python3 "${REPO_ROOT}/infra/ops/validate-runtime-image-scan.py" \
  --repo-root "${REPO_ROOT}" \
  --policy-only \
  --effective-environment production

require_production_postgres_ssl
require_production_postgres_archiving
require_production_external_student_source_security
require_production_object_storage
"${SCRIPT_DIR}/prepare-object-storage-client-ca.sh"
"${SCRIPT_DIR}/prepare-datastore-client-cas.sh"
require_public_ingress_config_preflight
if [[ "${run_public_runtime_checks}" == "true" ]]; then
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
else
  warn "live public runtime checks deferred until the mandatory post-deploy preflight"
fi

# 校验 Docker 和 Docker Compose 运行环境

# 校验 postgres-client 镜像内 pg_dump 和 pg_basebackup 可用性
# 备份脚本已容器化，不再依赖宿主机 pg_dump，但需要确保容器镜像中包含这些工具
log "正在校验 postgres-client 镜像内备份工具可用性..."

pg_image="${POSTGRES_IMAGE_REF:-cgr.dev/chainguard/postgres:latest@sha256:dc2f04037c1044a22af76cee4de70b9111885b17c561b939d7ed70103d100759}"

# 检查 pg_dump 存在且可执行
if ! docker run --rm --entrypoint /usr/bin/which "${pg_image}" pg_dump >/dev/null 2>&1; then
  die "postgres-client 镜像 (${pg_image}) 中 pg_dump 不可用"
fi

# 检查 pg_basebackup 存在且可执行
if ! docker run --rm --entrypoint /usr/bin/which "${pg_image}" pg_basebackup >/dev/null 2>&1; then
  die "postgres-client 镜像 (${pg_image}) 中 pg_basebackup 不可用"
fi

# 获取容器内 pg_dump 版本，与运行中的 PostgreSQL 版本做兼容性日志
pg_dump_version="$(docker run --rm --entrypoint /usr/bin/pg_dump "${pg_image}" --version 2>/dev/null | head -1 || echo "unknown")"
log "容器内 pg_dump 版本: ${pg_dump_version}"

require_backup_object_storage_config

mkdir -p \
  "${REPO_ROOT}/backups/postgres/logical" \
  "${REPO_ROOT}/backups/postgres/base" \
  "${POSTGRES_WAL_RESTORE_DIR}" \
  "${DEPLOY_STATE_DIR}"

backup_service_units=(
  stuhelper-postgres-dump-backup.service
  stuhelper-postgres-basebackup.service
  stuhelper-postgres-backup-sync.service
)
backup_service_commands=(
  "./infra/ops/run-scheduled-backup.sh dump"
  "./infra/ops/run-scheduled-backup.sh basebackup"
  "./infra/ops/run-scheduled-backup.sh sync"
)
backup_timer_units=(
  stuhelper-postgres-dump-backup.timer
  stuhelper-postgres-basebackup.timer
  stuhelper-postgres-backup-sync.timer
)
backup_timer_calendars=(
  "*-*-* 03:15:00"
  "Sun *-*-* 03:45:00"
  "*-*-* *:00/15:00"
)
backup_service_start_timeouts=(
  "18h"
  "1d 2h"
  "12h"
)
backup_service_common_environment=(
  "ENV_FILE=${REPO_ROOT}/.env.prod.shared"
  "SECRETS_ENV_FILE=${REPO_ROOT}/.env.prod.secrets"
  "GENERATED_ENV_FILE=${REPO_ROOT}/.env.prod.generated"
  "GENERATED_SECRET_ENV_FILE=${REPO_ROOT}/.env.prod.generated.secrets"
  "LOCAL_STATE_DIR=/var/lib/stuhelper"
)
for unit in "${backup_service_units[@]}" "${backup_timer_units[@]}"; do
  if ! systemd_unit_file_is_installed "${unit}"; then
    die "backup unit ${unit} is not installed on the target host"
  fi
done
for index in "${!backup_service_units[@]}"; do
  unit="${backup_service_units[${index}]}"
  expected_service_environment=("${backup_service_common_environment[@]}")
  if ((index < 2)); then
    expected_service_environment+=("BACKUP_STAGING_DIR=${BACKUP_STAGING_DIR}")
  fi
  expected_service_environment+=("BACKUP_OBJECT_STORAGE_OFF_HOST_REQUIRED=true")
  require_systemd_unit_exact_identity \
    "${unit}" \
    "${backup_service_user}" \
    "${backup_service_group}"
  require_systemd_unit_hardened_lifecycle \
    "${unit}" \
    "${backup_service_start_timeouts[${index}]}"
  require_systemd_unit_without_filesystem_overrides "${unit}"
  require_systemd_unit_without_conditions "${unit}"
  require_systemd_unit_exact_environment \
    "${unit}" \
    "${expected_service_environment[@]}"
  require_systemd_unit_hardened_execution \
    "${unit}" \
    "${REPO_ROOT}" \
    "${backup_service_commands[${index}]}" \
    "${expected_service_environment[@]}"
done
for index in "${!backup_timer_units[@]}"; do
  unit="${backup_timer_units[${index}]}"
  require_systemd_timer_schedule \
    "${unit}" \
    "${backup_service_units[${index}]}" \
    "${backup_timer_calendars[${index}]}"
  if [[ "${preflight_phase}" != "--timer-activation" ]]; then
    if ! systemctl is-enabled --quiet "${unit}"; then
      die "backup timer ${unit} is not enabled"
    fi
    if ! systemctl is-active --quiet "${unit}"; then
      die "backup timer ${unit} is not active"
    fi
  fi
done
if [[ "${preflight_phase}" == "--timer-activation" ]]; then
  log "backup timer unit/configuration readiness passed; activation state intentionally deferred to the root installer"
fi
[[ -n "${BACKUP_DATABASE_URL:-}" ]] || die "BACKUP_DATABASE_URL must be configured"
[[ -n "${REPLICATION_DATABASE_URL:-}" ]] || die "REPLICATION_DATABASE_URL must be configured"

# ── 恢复能力校验 ──
# 不仅检查配置存在，还验证恢复链路的实际可执行性

log "正在校验恢复能力..."

# 1. pg_restore 可用（恢复 custom-format dump 的核心工具）
if ! docker run --rm --entrypoint /usr/bin/which "${pg_image}" pg_restore >/dev/null 2>&1; then
  die "postgres-client 镜像 (${pg_image}) 中 pg_restore 不可用"
fi
pg_restore_version="$(docker run --rm --entrypoint /usr/bin/pg_restore "${pg_image}" --version 2>/dev/null | head -1 || echo "unknown")"
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
if [[ "${run_database_runtime_checks}" == "true" ]]; then
  if compose run --rm --no-deps -T postgres-client \
    pg_isready -d "${BACKUP_DATABASE_URL}" -t 5 >/dev/null 2>&1; then
    log "备份数据库连通性: OK"
  else
    die "备份数据库连通性检查失败（pg_isready 超时），请确认 BACKUP_DATABASE_URL 可达"
  fi
  if [[ "${EXTERNAL_POSTGRES_ENABLED:-false}" == "true" ]]; then
    require_external_postgres_pitr_evidence >/dev/null
    log "外部 PostgreSQL 连续归档和 PITR 证据有效且与当前集群一致"
  fi
else
  warn "live database connectivity deferred until the mandatory post-deploy preflight"
fi

log "remote preflight checks passed (${preflight_phase})"
