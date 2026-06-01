#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

usage() {
  cat <<'USAGE'
Usage: infra/ops/reclaim-production-disk-space.sh [--apply]

Safely reclaims production host disk space without deleting persistent Docker
volumes or database data. By default the script runs in dry-run mode and only
prints disk usage plus the actions it would take.

Actions with --apply:
  - journalctl --vacuum-size=${JOURNAL_VACUUM_SIZE:-256M}
  - apt-get clean
  - docker builder prune -af
  - docker image prune -f
  - when PRUNE_STUHELPER_OLD_IMAGES=true, remove inactive
    stuhelper/backend:* and stuhelper/frontend:* images
  - when PRUNE_STUHELPER_TMP_ARTIFACTS=true, remove known StuHelper /tmp
    transfer/build tarballs
  - when PRUNE_BAOTA_PANEL_BACKUPS_KEEP is a positive integer, keep only that
    many newest /www/backup/panel backup entries, preserving panel/db
  - when TRUNCATE_VAR_LOG_MESSAGES=true, truncate /var/log/messages
  - when ALLOW_POSTGRES_WAL_ARCHIVE_PRUNE=true and
    PRUNE_POSTGRES_WAL_ARCHIVE_KEEP_HOURS is positive, delete archived WAL
    segment files older than the keep window and write a manifest to /var/log

Never performed by default:
  - docker volume prune
  - deleting /var/lib/docker/volumes
  - deleting PostgreSQL PGDATA or wal-archive contents

Environment:
  JOURNAL_VACUUM_SIZE defaults to 256M.
  PRUNE_STUHELPER_OLD_IMAGES defaults to false.
  PRUNE_STUHELPER_TMP_ARTIFACTS defaults to false.
  PRUNE_BAOTA_PANEL_BACKUPS_KEEP defaults to empty.
  TRUNCATE_VAR_LOG_MESSAGES defaults to false.
  PRUNE_POSTGRES_WAL_ARCHIVE_KEEP_HOURS defaults to empty.
  ALLOW_POSTGRES_WAL_ARCHIVE_PRUNE defaults to false.
USAGE
}

apply=false
case "${1:-}" in
  --apply)
    apply=true
    ;;
  ""|--dry-run)
    ;;
  --help|-h)
    usage
    exit 0
    ;;
  *)
    usage >&2
    die "unknown argument: ${1}"
    ;;
esac

require_cmd df
require_cmd du

journal_vacuum_size="${JOURNAL_VACUUM_SIZE:-256M}"
prune_stuhelper_old_images="${PRUNE_STUHELPER_OLD_IMAGES:-false}"
prune_stuhelper_tmp_artifacts="${PRUNE_STUHELPER_TMP_ARTIFACTS:-false}"
prune_baota_panel_backups_keep="${PRUNE_BAOTA_PANEL_BACKUPS_KEEP:-}"
truncate_var_log_messages="${TRUNCATE_VAR_LOG_MESSAGES:-false}"
prune_postgres_wal_archive_keep_hours="${PRUNE_POSTGRES_WAL_ARCHIVE_KEEP_HOURS:-}"
allow_postgres_wal_archive_prune="${ALLOW_POSTGRES_WAL_ARCHIVE_PRUNE:-false}"
postgres_wal_archive_volume="${POSTGRES_WAL_ARCHIVE_VOLUME_NAME:-${STACK_NAME:-stuhelper}-postgres-wal-archive}"

show_usage() {
  log "filesystem usage"
  df -h / /var/lib/docker /www 2>/dev/null || df -h /

  log "largest top-level directories on root filesystem"
  du -xhd1 / 2>/dev/null | sort -h | tail -20 || true

  if [[ -d /var/lib/docker ]]; then
    log "largest Docker storage directories"
    du -xhd1 /var/lib/docker 2>/dev/null | sort -h | tail -20 || true
  fi

  if command -v docker >/dev/null 2>&1; then
    log "docker system df"
    docker system df || true
  fi
}

run_or_print() {
  if [[ "${apply}" == "true" ]]; then
    log "run: $*"
    "$@"
    return
  fi
  log "dry-run: $*"
}

run_shell_or_print() {
  local description="$1"
  shift
  if [[ "${apply}" == "true" ]]; then
    log "run: ${description}"
    "$@"
    return
  fi
  log "dry-run: ${description}"
}

is_positive_integer() {
  [[ "${1:-}" =~ ^[1-9][0-9]*$ ]]
}

prune_tmp_artifacts() {
  [[ "${prune_stuhelper_tmp_artifacts}" == "true" ]] || return 0
  run_shell_or_print "remove known StuHelper /tmp transfer artifacts" bash -lc '
    set -euo pipefail
    find /tmp -xdev -maxdepth 1 -type f \( \
      -name "stuhelper-*.tar" -o \
      -name "stuhelper-*.tar.gz" -o \
      -name "stuhelper-*.tgz" -o \
      -name "stuhelper-*.out" -o \
      -name "stuhelper-*.log" -o \
      -name "go*.linux-amd64.tar.gz" \
    \) -print -delete
  '
}

prune_baota_panel_backups() {
  [[ -n "${prune_baota_panel_backups_keep}" ]] || return 0
  if ! is_positive_integer "${prune_baota_panel_backups_keep}"; then
    die "PRUNE_BAOTA_PANEL_BACKUPS_KEEP must be a positive integer"
  fi
  run_shell_or_print "keep newest ${prune_baota_panel_backups_keep} Baota panel backups" bash -lc '
    set -euo pipefail
    keep="$1"
    backup_dir="/www/backup/panel"
    [[ -d "${backup_dir}" ]] || exit 0
    mapfile -t victims < <(
      find "${backup_dir}" -mindepth 1 -maxdepth 1 ! -name db -printf "%T@ %p\n" |
        sort -rn |
        tail -n +"$((keep + 1))" |
        cut -d" " -f2-
    )
    if (( ${#victims[@]} == 0 )); then
      echo "no old Baota panel backups to remove"
      exit 0
    fi
    printf "%s\n" "${victims[@]}"
    rm -rf -- "${victims[@]}"
  ' _ "${prune_baota_panel_backups_keep}"
}

truncate_messages_log() {
  [[ "${truncate_var_log_messages}" == "true" ]] || return 0
  run_shell_or_print "truncate /var/log/messages" bash -lc '
    set -euo pipefail
    [[ -f /var/log/messages ]] || exit 0
    : > /var/log/messages
  '
}

prune_inactive_stuhelper_images() {
  local active_ids image_lines ref image_id remove_refs=()
  [[ "${prune_stuhelper_old_images}" == "true" ]] || return 0
  command -v docker >/dev/null 2>&1 || return

  active_ids="$(
    docker ps --format '{{.Image}}' |
      while IFS= read -r ref; do
        [[ -n "${ref}" ]] || continue
        docker image inspect --format '{{.Id}}' "${ref}" 2>/dev/null || true
      done |
      sort -u
  )"
  image_lines="$(
    docker image ls --format '{{.Repository}}:{{.Tag}} {{.ID}}' |
      awk '$1 ~ /^stuhelper\/(backend|frontend):/ { print }'
  )"
  while read -r ref image_id; do
    [[ -n "${ref}" && -n "${image_id}" ]] || continue
    if printf '%s\n' "${active_ids}" | grep -q "^sha256:${image_id}"; then
      log "keep active StuHelper image: ${ref}"
      continue
    fi
    remove_refs+=("${ref}")
  done <<<"${image_lines}"

  if (( ${#remove_refs[@]} == 0 )); then
    log "no inactive StuHelper backend/frontend images to remove"
    return
  fi
  run_or_print docker image rm "${remove_refs[@]}"
}

resolve_postgres_wal_archive_dir() {
  local configured="${POSTGRES_WAL_ARCHIVE_VOLUME_NAME:-}"
  local wal_dir="/var/lib/docker/volumes/${postgres_wal_archive_volume}/_data"
  [[ -d "${wal_dir}" ]] && {
    printf '%s\n' "${wal_dir}"
    return 0
  }

  if [[ -n "${configured}" ]]; then
    die "WAL archive directory not found: ${wal_dir}"
  fi

  command -v docker >/dev/null 2>&1 || die "WAL archive directory not found: ${wal_dir}"

  local matches=()
  mapfile -t matches < <(
    docker volume ls --format '{{.Name}}' |
      awk '/-postgres-wal-archive$/ { print }'
  )
  case "${#matches[@]}" in
    0)
      die "WAL archive directory not found: ${wal_dir}"
      ;;
    1)
      wal_dir="/var/lib/docker/volumes/${matches[0]}/_data"
      [[ -d "${wal_dir}" ]] || die "detected WAL archive volume has no data directory: ${wal_dir}"
      warn "detected PostgreSQL WAL archive volume: ${matches[0]}"
      printf '%s\n' "${wal_dir}"
      ;;
    *)
      die "multiple PostgreSQL WAL archive volumes found; set POSTGRES_WAL_ARCHIVE_VOLUME_NAME explicitly: ${matches[*]}"
      ;;
  esac
}

prune_postgres_wal_archive() {
  [[ -n "${prune_postgres_wal_archive_keep_hours}" ]] || return 0
  [[ "${allow_postgres_wal_archive_prune}" == "true" ]] || \
    die "set ALLOW_POSTGRES_WAL_ARCHIVE_PRUNE=true to prune PostgreSQL WAL archive"
  if ! is_positive_integer "${prune_postgres_wal_archive_keep_hours}"; then
    die "PRUNE_POSTGRES_WAL_ARCHIVE_KEEP_HOURS must be a positive integer"
  fi

  local wal_dir
  wal_dir="$(resolve_postgres_wal_archive_dir)"
  local keep_minutes=$((prune_postgres_wal_archive_keep_hours * 60))
  local manifest="/var/log/stuhelper-wal-archive-prune-$(date -u +%Y%m%dT%H%M%SZ).manifest"
  [[ -d "${wal_dir}" ]] || die "WAL archive directory not found: ${wal_dir}"

  run_shell_or_print "prune PostgreSQL WAL archive older than ${prune_postgres_wal_archive_keep_hours}h; manifest=${manifest}" \
    bash -lc '
      set -euo pipefail
      wal_dir="$1"
      keep_minutes="$2"
      manifest="$3"
      find "${wal_dir}" -xdev -maxdepth 1 -type f -mmin +"${keep_minutes}" \
        -regextype posix-extended -regex ".*/[0-9A-F]{24}" \
        -printf "%s %TY-%Tm-%TdT%TH:%TM:%TS %p\n" |
        sort > "${manifest}"
      if [[ ! -s "${manifest}" ]]; then
        echo "no archived WAL files matched prune window" | tee -a "${manifest}"
        exit 0
      fi
      awk "{sum+=\$1; count++} END {printf \"files=%d bytes=%d\\n\", count, sum}" "${manifest}" | tee -a "${manifest}"
      awk "{print \$3}" "${manifest}" | xargs -r rm -f --
    ' _ "${wal_dir}" "${keep_minutes}" "${manifest}"
}

show_usage

prune_tmp_artifacts
prune_baota_panel_backups
truncate_messages_log

run_or_print journalctl "--vacuum-size=${journal_vacuum_size}"
run_or_print apt-get clean

if command -v docker >/dev/null 2>&1; then
  run_or_print docker builder prune -af
  run_or_print docker image prune -f
  prune_inactive_stuhelper_images
else
  warn "docker command is not available; skipping Docker cache cleanup"
fi

prune_postgres_wal_archive

log "post-cleanup usage"
df -h / /var/lib/docker /www 2>/dev/null || df -h /
