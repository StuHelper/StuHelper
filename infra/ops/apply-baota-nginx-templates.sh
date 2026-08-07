#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

usage() {
  cat <<'USAGE'
Usage: infra/ops/apply-baota-nginx-templates.sh [options]

Applies the repository-owned Baota/Nginx vhost templates to the production
vhost files in a repeatable way. The default mode is a dry run; pass --apply to
write files.

Options:
  --profile stuhelper|sso|connector|app-all|all
                                Template set to apply. `all` preserves the
                                existing HTTP stuhelper+sso behavior; `app-all`
                                selects stuhelper+connector. Defaults to all.
  --apply                       Replace target files. Without this, no files are changed.
  --reload                      Run nginx -s reload after a successful nginx -t.
  --preflight                   Run nginx-public-ingress-preflight.sh after apply/dry-run.
  -h, --help                    Show this help.

Environment overrides:
  BAOTA_NGINX_STUHELPER_TEMPLATE
  BAOTA_NGINX_SSO_TEMPLATE
  BAOTA_NGINX_SSO_WELL_KNOWN_EXTENSION_TEMPLATE
  BAOTA_NGINX_CONNECTOR_TEMPLATE
  BAOTA_NGINX_STUHELPER_TARGET
  BAOTA_NGINX_SSO_TARGET
  BAOTA_NGINX_SSO_WELL_KNOWN_EXTENSION_TARGET
  BAOTA_NGINX_CONNECTOR_TARGET
  BAOTA_NGINX_BIN
  CAMPUS_CONNECTOR_GATEWAY_PUBLIC_PORT
  CAMPUS_CONNECTOR_GATEWAY_EXTERNAL_PORT
  CAMPUS_CONNECTOR_ALLOWED_SOURCE_CIDRS

This script intentionally contains no SSH host, credential, token, or secret.
USAGE
}

profile="all"
apply="false"
reload="false"
preflight="false"

while (($# > 0)); do
  case "$1" in
    --profile)
      [[ $# -ge 2 ]] || die "--profile requires stuhelper, sso, connector, app-all, or all"
      profile="$2"
      shift 2
      ;;
    --apply)
      apply="true"
      shift
      ;;
    --reload)
      reload="true"
      shift
      ;;
    --preflight)
      preflight="true"
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      die "unknown option: $1"
      ;;
  esac
done

case "${profile}" in
  stuhelper|sso|connector|app-all|all) ;;
  *) die "--profile must be stuhelper, sso, connector, app-all, or all" ;;
esac

nginx_bin="${BAOTA_NGINX_BIN:-nginx}"
stuhelper_template="${BAOTA_NGINX_STUHELPER_TEMPLATE:-${REPO_ROOT}/infra/nginx/baota-stuhelper.conf}"
sso_template="${BAOTA_NGINX_SSO_TEMPLATE:-${REPO_ROOT}/infra/nginx/baota-casdoor-sso.conf}"
sso_extension_template="${BAOTA_NGINX_SSO_WELL_KNOWN_EXTENSION_TEMPLATE:-${REPO_ROOT}/infra/nginx/baota-casdoor-sso-well-known-extension.conf}"
connector_template="${BAOTA_NGINX_CONNECTOR_TEMPLATE:-${REPO_ROOT}/infra/nginx/baota-campus-connector-stream.conf.template}"
connector_renderer="${REPO_ROOT}/infra/ops/render-campus-connector-nginx-stream.py"
stuhelper_target="${BAOTA_NGINX_STUHELPER_TARGET:-/www/server/panel/vhost/nginx/stuhelper.com.conf}"
sso_target="${BAOTA_NGINX_SSO_TARGET:-/www/server/panel/vhost/nginx/sso.stuhelper.com.conf}"
sso_extension_target="${BAOTA_NGINX_SSO_WELL_KNOWN_EXTENSION_TARGET:-/www/server/panel/vhost/nginx/extension/sso.stuhelper.com/stuhelper-sso-well-known.conf}"
connector_target="${BAOTA_NGINX_CONNECTOR_TARGET:-/www/server/panel/vhost/nginx/tcp/connector.stuhelper.com.conf}"
timestamp="${BAOTA_NGINX_BACKUP_TIMESTAMP:-$(date -u +%Y%m%dT%H%M%SZ)}"
rendered_connector_template=""

cleanup() {
  if [[ -n "${rendered_connector_template}" ]]; then
    rm -f "${rendered_connector_template}"
  fi
}
trap cleanup EXIT

selected_names=()
selected_templates=()
selected_targets=()

add_entry() {
  selected_names+=("$1")
  selected_templates+=("$2")
  selected_targets+=("$3")
}

add_connector_entry() {
  require_cmd python3
  [[ -f "${connector_renderer}" ]] || die "missing campus connector stream renderer: ${connector_renderer}"
  rendered_connector_template="$(mktemp)"
  if ! python3 "${connector_renderer}" \
    --template "${connector_template}" \
    --output "${rendered_connector_template}" \
    --public-port "${CAMPUS_CONNECTOR_GATEWAY_PUBLIC_PORT:-9444}" \
    --upstream-port "${CAMPUS_CONNECTOR_GATEWAY_EXTERNAL_PORT:-19444}" \
    --allowed-cidrs "${CAMPUS_CONNECTOR_ALLOWED_SOURCE_CIDRS:-}"; then
    die "failed to render fail-closed campus connector stream ingress"
  fi
  add_entry "connector" "${rendered_connector_template}" "${connector_target}"
}

case "${profile}" in
  stuhelper)
    add_entry "stuhelper" "${stuhelper_template}" "${stuhelper_target}"
    ;;
  sso)
    add_entry "sso" "${sso_template}" "${sso_target}"
    add_entry "sso-well-known-extension" "${sso_extension_template}" "${sso_extension_target}"
    ;;
  connector)
    add_connector_entry
    ;;
  app-all)
    add_entry "stuhelper" "${stuhelper_template}" "${stuhelper_target}"
    add_connector_entry
    ;;
  all)
    add_entry "stuhelper" "${stuhelper_template}" "${stuhelper_target}"
    add_entry "sso" "${sso_template}" "${sso_target}"
    add_entry "sso-well-known-extension" "${sso_extension_template}" "${sso_extension_target}"
    ;;
esac

validate_inputs() {
  local i template target target_dir
  for i in "${!selected_names[@]}"; do
    template="${selected_templates[$i]}"
    target="${selected_targets[$i]}"
    target_dir="$(dirname "${target}")"
    [[ -f "${template}" ]] || die "missing template for ${selected_names[$i]}: ${template}"
    if [[ ! -d "${target_dir}" && "${apply}" != "true" ]]; then
      log "${selected_names[$i]} target directory would be created: ${target_dir}"
    fi
  done
}

run_preflight() {
  local preflight_profile="${profile}"
  [[ "${preflight_profile}" != "all" ]] || preflight_profile="all"

  log "running Nginx public ingress preflight for profile=${preflight_profile}"
  NGINX_PUBLIC_INGRESS_PROFILE="${preflight_profile}" \
    NGINX_PUBLIC_INGRESS_NGINX_BIN="${nginx_bin}" \
    NGINX_PUBLIC_INGRESS_CONNECTOR_CONFIG_FILE="${connector_target}" \
    "${SCRIPT_DIR}/nginx-public-ingress-preflight.sh"
}

dry_run() {
  local i template target
  log "dry-run only; pass --apply to replace Baota/Nginx vhost files"
  for i in "${!selected_names[@]}"; do
    template="${selected_templates[$i]}"
    target="${selected_targets[$i]}"
    if [[ -f "${target}" ]] && cmp -s "${template}" "${target}"; then
      log "${selected_names[$i]} unchanged: ${target}"
    elif [[ -f "${target}" ]]; then
      log "${selected_names[$i]} would replace ${target} with ${template}"
    else
      log "${selected_names[$i]} would create ${target} from ${template}"
    fi
  done
}

restore_changed_targets() {
  local -n names_ref=$1
  local -n targets_ref=$2
  local -n backups_ref=$3
  local i backup target

  for ((i=${#names_ref[@]} - 1; i>=0; i--)); do
    target="${targets_ref[$i]}"
    backup="${backups_ref[$i]}"
    if [[ -n "${backup}" && -f "${backup}" ]]; then
      cp -p "${backup}" "${target}" || warn "failed to restore ${target} from ${backup}"
      warn "restored ${names_ref[$i]} target from backup after nginx -t failure: ${target}"
    else
      rm -f "${target}" || warn "failed to remove newly created target after nginx -t failure: ${target}"
      warn "removed newly created ${names_ref[$i]} target after nginx -t failure: ${target}"
    fi
  done
}

apply_templates() {
  local changed_names=()
  local changed_targets=()
  local changed_backups=()
  local i template target target_dir backup tmp_target

  require_cmd install
  require_cmd cmp
  require_cmd cp
  require_cmd mv
  require_cmd "${nginx_bin}"

  for i in "${!selected_names[@]}"; do
    template="${selected_templates[$i]}"
    target="${selected_targets[$i]}"
    target_dir="$(dirname "${target}")"

    mkdir -p "${target_dir}"

    if [[ -f "${target}" ]] && cmp -s "${template}" "${target}"; then
      log "${selected_names[$i]} already matches template: ${target}"
      continue
    fi

    backup=""
    if [[ -f "${target}" ]]; then
      backup="${target}.bak.${timestamp}"
      cp -p "${target}" "${backup}"
      log "${selected_names[$i]} backup created: ${backup}"
    else
      log "${selected_names[$i]} target does not exist and will be created: ${target}"
    fi

    tmp_target="$(mktemp "$(dirname "${target}")/.${selected_names[$i]}.XXXXXX")"
    install -m 0644 "${template}" "${tmp_target}"
    mv "${tmp_target}" "${target}"
    log "${selected_names[$i]} template installed: ${target}"

    changed_names+=("${selected_names[$i]}")
    changed_targets+=("${target}")
    changed_backups+=("${backup}")
  done

  log "running nginx -t before reload"
  if ! "${nginx_bin}" -t; then
    if (( ${#changed_names[@]} > 0 )); then
      restore_changed_targets changed_names changed_targets changed_backups
    fi
    die "nginx -t failed after applying Baota/Nginx templates"
  fi

  if [[ "${reload}" == "true" ]]; then
    log "reloading Nginx"
    "${nginx_bin}" -s reload
  fi
}

validate_inputs

if [[ "${apply}" != "true" ]]; then
  dry_run
else
  apply_templates
fi

if [[ "${preflight}" == "true" ]]; then
  run_preflight
fi

if [[ "${apply}" == "true" ]]; then
  log "Baota/Nginx template application completed"
else
  log "Baota/Nginx template dry-run completed"
fi
