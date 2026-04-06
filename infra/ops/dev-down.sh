#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"
# shellcheck source=lib/dev-local.sh
source "${SCRIPT_DIR}/lib/dev-local.sh"

require_cmd docker
"${SCRIPT_DIR}/init-dev-env.sh"
ensure_dev_runtime_dirs

if [[ -f "${DEV_RUNTIME_ENV}" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "${DEV_RUNTIME_ENV}"
  set +a
fi

kill_all_dev_processes
kill_listener_if_matches 8080 "${REPO_ROOT}/server/tmp/stuhelper"
for port in "${WEB_DEV_PORT:-3000}" "3000"; do
  kill_listener_if_matches "${port}" "@stuhelper/web exec vite"
  kill_listener_if_matches "${port}" "${REPO_ROOT}/clients/web/node_modules/.bin/../vite/bin/vite.js"
done
for port in "${ADMIN_EXTERNAL_PORT:-3001}" "3001"; do
  kill_listener_if_matches "${port}" "@vben/web-ele exec vite"
  kill_listener_if_matches "${port}" "${REPO_ROOT}/clients/admin/node_modules/.bin/../vite/bin/vite.js"
done
rm -f "${DEV_RUNTIME_ENV}"

args=(down --remove-orphans)
if [[ "${REMOVE_VOLUMES:-false}" == "true" ]]; then
  args+=(-v)
fi

compose "${args[@]}"
extra_profile_args=(--remove-orphans)
if [[ "${REMOVE_VOLUMES:-false}" == "true" ]]; then
  extra_profile_args+=(-v)
fi
compose --profile dev-full down "${extra_profile_args[@]}" >/dev/null 2>&1 || true
compose --profile observability down "${extra_profile_args[@]}" >/dev/null 2>&1 || true

log "development stack stopped"
