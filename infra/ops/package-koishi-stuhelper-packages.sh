#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

usage() {
  cat <<'USAGE'
Usage: infra/ops/package-koishi-stuhelper-packages.sh [output.tar.gz]

Packages the StuHelper Koishi runtime packages from the current local build:

  - @stuhelper/koishi-shared
  - @stuhelper/koishi-moderation-core
  - koishi-plugin-stuhelper-core
  - koishi-plugin-stuhelper-binding
  - koishi-plugin-stuhelper-group-guard

The archive is meant to be extracted at the Koishi production Compose directory
root, so paths are laid out as:

  koishi/node_modules/<package>/package.json
  koishi/node_modules/<package>/lib/
  koishi/local-workspaces/{packages,plugins}/<package>/package.json
  koishi/local-workspaces/{packages,plugins}/<package>/lib/
  koishi/STUHELPER_KOISHI_APPLY_LOCAL_WORKSPACES.cjs

Only package.json, lib/, and the stuhelper-core browser dist/ are included.
No source tree, nested node_modules, environment file, SSH helper, or secret is
packaged.

Input:
  output.tar.gz defaults to KOISHI_STUHELPER_PACKAGE_OUTPUT, then
  /tmp/stuhelper-koishi-packages.tar.gz.

Before running this script, build the local workspace:

  cd bots/koishi
  corepack yarn build
USAGE
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
  exit 0
fi

require_cmd tar
require_cmd find
require_cmd awk

OUTPUT_FILE="${1:-${KOISHI_STUHELPER_PACKAGE_OUTPUT:-/tmp/stuhelper-koishi-packages.tar.gz}}"
case "${OUTPUT_FILE}" in
  /*) ;;
  *) OUTPUT_FILE="${PWD}/${OUTPUT_FILE}" ;;
esac
OUTPUT_DIR="$(dirname "${OUTPUT_FILE}")"

sha256_file() {
  local file="$1"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "${file}" | awk '{print $1}'
    return
  fi
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "${file}" | awk '{print $1}'
    return
  fi
  die "missing required command: sha256sum or shasum"
}

copy_package_payload() {
  local source_dir="$1"
  local package_name="$2"
  local destination_dir="$3"

  [[ -f "${source_dir}/package.json" ]] || die "${package_name} package.json is missing"
  [[ -d "${source_dir}/lib" ]] || die "${package_name} lib/ is missing; run corepack yarn build in bots/koishi"
  [[ -f "${source_dir}/lib/index.js" ]] || die "${package_name} lib/index.js is missing; run corepack yarn build in bots/koishi"

  if [[ -d "${source_dir}/src" ]]; then
    local newer_source
    newer_source="$(
      find "${source_dir}/src" -type f -newer "${source_dir}/lib/index.js" -print -quit
    )"
    [[ -z "${newer_source}" ]] || die "${package_name} build output is older than ${newer_source}; run corepack yarn build in bots/koishi"
  fi

  mkdir -p "${destination_dir}"
  (
    cd "${source_dir}"
    tar -cf - package.json lib
  ) | (
    cd "${destination_dir}"
    tar -xf -
  )
}

copy_browser_plugin_payload() {
  local source_dir="$1"
  local package_name="$2"
  local destination_dir="$3"

  copy_package_payload "${source_dir}" "${package_name}" "${destination_dir}"

  [[ -d "${source_dir}/dist" ]] || die "${package_name} dist/ is missing; run corepack yarn build in bots/koishi"
  [[ -f "${source_dir}/dist/index.js" ]] || die "${package_name} dist/index.js is missing; run corepack yarn build in bots/koishi"

  if [[ -d "${source_dir}/client" ]]; then
    local newer_client
    newer_client="$(
      find "${source_dir}/client" -type f -newer "${source_dir}/dist/index.js" -print -quit
    )"
    [[ -z "${newer_client}" ]] || die "${package_name} browser dist is older than ${newer_client}; run corepack yarn build in bots/koishi"
  fi

  (
    cd "${source_dir}"
    tar -cf - dist
  ) | (
    cd "${destination_dir}"
    tar -xf -
  )
}

write_workspace_guard() {
  local destination_file="$1"

  cat >"${destination_file}" <<'NODE'
#!/usr/bin/env node
const fs = require('node:fs')
const path = require('node:path')

const root = __dirname
const manifestPath = path.join(root, 'package.json')
const requiredWorkspaces = [
  'local-workspaces/plugins/*',
  'local-workspaces/packages/*',
]
const requiredPrivateDependencies = [
  'koishi-plugin-stuhelper-core',
  'koishi-plugin-stuhelper-group-guard',
]
const optionalPrivateDependencies = [
  'koishi-plugin-stuhelper-binding',
  'koishi-plugin-stuhelper-admin',
]

if (!fs.existsSync(manifestPath)) {
  throw new Error(`Koishi package.json not found: ${manifestPath}`)
}

const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'))
manifest.workspaces = Array.from(new Set([
  ...(Array.isArray(manifest.workspaces) ? manifest.workspaces : []),
  ...requiredWorkspaces,
]))
manifest.dependencies ||= {}

for (const name of requiredPrivateDependencies) {
  manifest.dependencies[name] = 'workspace:*'
}
for (const name of optionalPrivateDependencies) {
  if (Object.prototype.hasOwnProperty.call(manifest.dependencies, name)) {
    manifest.dependencies[name] = 'workspace:*'
  }
}

fs.writeFileSync(manifestPath, `${JSON.stringify(manifest, null, 2)}\n`)
console.log('[stuhelper-koishi] local workspace guard applied')
NODE
  chmod +x "${destination_file}"
}

mkdir -p "${OUTPUT_DIR}"

tmp_root="$(mktemp -d "${TMPDIR:-/tmp}/stuhelper-koishi-packages.XXXXXX")"
tmp_file="$(mktemp "${OUTPUT_DIR}/stuhelper-koishi-packages.XXXXXX.tar.gz")"
trap 'rm -rf "${tmp_root}" "${tmp_file}"' EXIT

copy_package_payload \
  "${REPO_ROOT}/bots/koishi/packages/shared" \
  "@stuhelper/koishi-shared" \
  "${tmp_root}/koishi/node_modules/@stuhelper/koishi-shared"
copy_package_payload \
  "${REPO_ROOT}/bots/koishi/packages/shared" \
  "@stuhelper/koishi-shared" \
  "${tmp_root}/koishi/local-workspaces/packages/koishi-shared"

copy_package_payload \
  "${REPO_ROOT}/bots/koishi/packages/moderation-core" \
  "@stuhelper/koishi-moderation-core" \
  "${tmp_root}/koishi/node_modules/@stuhelper/koishi-moderation-core"
copy_package_payload \
  "${REPO_ROOT}/bots/koishi/packages/moderation-core" \
  "@stuhelper/koishi-moderation-core" \
  "${tmp_root}/koishi/local-workspaces/packages/koishi-moderation-core"

copy_browser_plugin_payload \
  "${REPO_ROOT}/bots/koishi/plugins/stuhelper-core" \
  "koishi-plugin-stuhelper-core" \
  "${tmp_root}/koishi/node_modules/koishi-plugin-stuhelper-core"
copy_browser_plugin_payload \
  "${REPO_ROOT}/bots/koishi/plugins/stuhelper-core" \
  "koishi-plugin-stuhelper-core" \
  "${tmp_root}/koishi/local-workspaces/plugins/stuhelper-core"

copy_package_payload \
  "${REPO_ROOT}/bots/koishi/plugins/stuhelper-binding" \
  "koishi-plugin-stuhelper-binding" \
  "${tmp_root}/koishi/node_modules/koishi-plugin-stuhelper-binding"
copy_package_payload \
  "${REPO_ROOT}/bots/koishi/plugins/stuhelper-binding" \
  "koishi-plugin-stuhelper-binding" \
  "${tmp_root}/koishi/local-workspaces/plugins/stuhelper-binding"

copy_package_payload \
  "${REPO_ROOT}/bots/koishi/plugins/stuhelper-group-guard" \
  "koishi-plugin-stuhelper-group-guard" \
  "${tmp_root}/koishi/node_modules/koishi-plugin-stuhelper-group-guard"
copy_package_payload \
  "${REPO_ROOT}/bots/koishi/plugins/stuhelper-group-guard" \
  "koishi-plugin-stuhelper-group-guard" \
  "${tmp_root}/koishi/local-workspaces/plugins/stuhelper-group-guard"

write_workspace_guard "${tmp_root}/koishi/STUHELPER_KOISHI_APPLY_LOCAL_WORKSPACES.cjs"

(
  cd "${tmp_root}"
  tar -czf "${tmp_file}" \
    koishi/STUHELPER_KOISHI_APPLY_LOCAL_WORKSPACES.cjs \
    koishi/local-workspaces/packages/koishi-shared \
    koishi/local-workspaces/packages/koishi-moderation-core \
    koishi/local-workspaces/plugins/stuhelper-core \
    koishi/local-workspaces/plugins/stuhelper-binding \
    koishi/local-workspaces/plugins/stuhelper-group-guard \
    koishi/node_modules/@stuhelper/koishi-shared \
    koishi/node_modules/@stuhelper/koishi-moderation-core \
    koishi/node_modules/koishi-plugin-stuhelper-core \
    koishi/node_modules/koishi-plugin-stuhelper-binding \
    koishi/node_modules/koishi-plugin-stuhelper-group-guard
)

mv "${tmp_file}" "${OUTPUT_FILE}"

checksum="$(sha256_file "${OUTPUT_FILE}")"
log "Koishi StuHelper package archive created: ${OUTPUT_FILE}"
log "sha256=${checksum}"
