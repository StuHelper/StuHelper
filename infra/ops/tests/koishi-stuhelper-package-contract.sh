#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
PACKAGE_SCRIPT="${REPO_ROOT}/infra/ops/package-koishi-stuhelper-packages.sh"
RELEASE_RUNBOOK="${REPO_ROOT}/docs/guides/release-runbook.md"
KOISHI_README="${REPO_ROOT}/bots/koishi/README.md"

fail() {
  echo "[koishi-stuhelper-package-contract][error] $*" >&2
  exit 1
}

assert_contains() {
  local file="$1"
  local pattern="$2"
  if ! grep -Eq "${pattern}" "${file}"; then
    fail "expected ${file} to contain pattern: ${pattern}"
  fi
}

assert_not_contains() {
  local file="$1"
  local pattern="$2"
  if grep -Eq "${pattern}" "${file}"; then
    fail "expected ${file} to not contain pattern: ${pattern}"
  fi
}

for file in "${PACKAGE_SCRIPT}" "${RELEASE_RUNBOOK}" "${KOISHI_README}"; do
  [[ -f "${file}" ]] || fail "missing file: ${file}"
done

bash -n "${PACKAGE_SCRIPT}"

assert_contains "${PACKAGE_SCRIPT}" '@stuhelper/koishi-shared'
assert_contains "${PACKAGE_SCRIPT}" '@stuhelper/koishi-moderation-core'
assert_contains "${PACKAGE_SCRIPT}" 'koishi-plugin-stuhelper-core'
assert_contains "${PACKAGE_SCRIPT}" 'koishi-plugin-stuhelper-binding'
assert_contains "${PACKAGE_SCRIPT}" 'koishi-plugin-stuhelper-group-guard'
assert_contains "${PACKAGE_SCRIPT}" 'koishi/node_modules/@stuhelper/koishi-shared'
assert_contains "${PACKAGE_SCRIPT}" 'koishi/node_modules/@stuhelper/koishi-moderation-core'
assert_contains "${PACKAGE_SCRIPT}" 'koishi/node_modules/koishi-plugin-stuhelper-core'
assert_contains "${PACKAGE_SCRIPT}" 'koishi/node_modules/koishi-plugin-stuhelper-binding'
assert_contains "${PACKAGE_SCRIPT}" 'koishi/node_modules/koishi-plugin-stuhelper-group-guard'
assert_contains "${PACKAGE_SCRIPT}" 'koishi/local-workspaces/packages/koishi-shared'
assert_contains "${PACKAGE_SCRIPT}" 'koishi/local-workspaces/packages/koishi-moderation-core'
assert_contains "${PACKAGE_SCRIPT}" 'koishi/local-workspaces/plugins/stuhelper-core'
assert_contains "${PACKAGE_SCRIPT}" 'koishi/local-workspaces/plugins/stuhelper-binding'
assert_contains "${PACKAGE_SCRIPT}" 'koishi/local-workspaces/plugins/stuhelper-group-guard'
assert_contains "${PACKAGE_SCRIPT}" 'STUHELPER_KOISHI_APPLY_LOCAL_WORKSPACES\.cjs'
assert_contains "${PACKAGE_SCRIPT}" 'workspace:\*'
assert_contains "${PACKAGE_SCRIPT}" 'package.json lib'
assert_contains "${PACKAGE_SCRIPT}" 'dist/index\.js'
assert_contains "${PACKAGE_SCRIPT}" 'browser dist is older'
assert_contains "${PACKAGE_SCRIPT}" 'sha256sum'
assert_contains "${PACKAGE_SCRIPT}" 'shasum -a 256'
assert_contains "${PACKAGE_SCRIPT}" 'corepack yarn build'
assert_contains "${PACKAGE_SCRIPT}" 'build output is older'
assert_contains "${PACKAGE_SCRIPT}" 'KOISHI_STUHELPER_PACKAGE_OUTPUT'
assert_contains "${PACKAGE_SCRIPT}" '/tmp/stuhelper-koishi-packages.tar.gz'
assert_not_contains "${PACKAGE_SCRIPT}" 'root@'
assert_not_contains "${PACKAGE_SCRIPT}" '65022'
assert_not_contains "${PACKAGE_SCRIPT}" '2222'
assert_not_contains "${PACKAGE_SCRIPT}" 'STUHELPER_PLATFORM_SERVICE_TOKEN='

tmp_dir="$(mktemp -d)"
trap 'rm -rf "${tmp_dir}"' EXIT

output_file="${tmp_dir}/stuhelper-koishi-packages.tar.gz"
"${PACKAGE_SCRIPT}" "${output_file}" >/tmp/koishi-stuhelper-package-contract.log
[[ -s "${output_file}" ]] || fail "expected archive to be created"

relative_output_file="infra/generated/koishi-stuhelper-package-contract-relative.tar.gz"
rm -f "${REPO_ROOT}/${relative_output_file}"
(
  cd "${REPO_ROOT}"
  "${PACKAGE_SCRIPT}" "${relative_output_file}" >/tmp/koishi-stuhelper-package-contract-relative.log
)
[[ -s "${REPO_ROOT}/${relative_output_file}" ]] || fail "expected relative output archive to be created"
rm -f "${REPO_ROOT}/${relative_output_file}"

tar -tzf "${output_file}" >"${tmp_dir}/manifest.txt"

assert_contains "${tmp_dir}/manifest.txt" '^koishi/STUHELPER_KOISHI_APPLY_LOCAL_WORKSPACES\.cjs$'
assert_contains "${tmp_dir}/manifest.txt" '^koishi/local-workspaces/packages/koishi-shared/package\.json$'
assert_contains "${tmp_dir}/manifest.txt" '^koishi/local-workspaces/packages/koishi-shared/lib/index\.js$'
assert_contains "${tmp_dir}/manifest.txt" '^koishi/local-workspaces/packages/koishi-moderation-core/package\.json$'
assert_contains "${tmp_dir}/manifest.txt" '^koishi/local-workspaces/packages/koishi-moderation-core/lib/index\.js$'
assert_contains "${tmp_dir}/manifest.txt" '^koishi/local-workspaces/plugins/stuhelper-core/package\.json$'
assert_contains "${tmp_dir}/manifest.txt" '^koishi/local-workspaces/plugins/stuhelper-core/lib/index\.js$'
assert_contains "${tmp_dir}/manifest.txt" '^koishi/local-workspaces/plugins/stuhelper-core/dist/index\.js$'
assert_contains "${tmp_dir}/manifest.txt" '^koishi/local-workspaces/plugins/stuhelper-binding/package\.json$'
assert_contains "${tmp_dir}/manifest.txt" '^koishi/local-workspaces/plugins/stuhelper-binding/lib/index\.js$'
assert_contains "${tmp_dir}/manifest.txt" '^koishi/local-workspaces/plugins/stuhelper-group-guard/package\.json$'
assert_contains "${tmp_dir}/manifest.txt" '^koishi/local-workspaces/plugins/stuhelper-group-guard/lib/index\.js$'
assert_contains "${tmp_dir}/manifest.txt" '^koishi/node_modules/@stuhelper/koishi-shared/package\.json$'
assert_contains "${tmp_dir}/manifest.txt" '^koishi/node_modules/@stuhelper/koishi-shared/lib/index\.js$'
assert_contains "${tmp_dir}/manifest.txt" '^koishi/node_modules/@stuhelper/koishi-moderation-core/package\.json$'
assert_contains "${tmp_dir}/manifest.txt" '^koishi/node_modules/@stuhelper/koishi-moderation-core/lib/index\.js$'
assert_contains "${tmp_dir}/manifest.txt" '^koishi/node_modules/koishi-plugin-stuhelper-core/package\.json$'
assert_contains "${tmp_dir}/manifest.txt" '^koishi/node_modules/koishi-plugin-stuhelper-core/lib/index\.js$'
assert_contains "${tmp_dir}/manifest.txt" '^koishi/node_modules/koishi-plugin-stuhelper-core/dist/index\.js$'
assert_contains "${tmp_dir}/manifest.txt" '^koishi/node_modules/koishi-plugin-stuhelper-binding/package\.json$'
assert_contains "${tmp_dir}/manifest.txt" '^koishi/node_modules/koishi-plugin-stuhelper-binding/lib/index\.js$'
assert_contains "${tmp_dir}/manifest.txt" '^koishi/node_modules/koishi-plugin-stuhelper-group-guard/package\.json$'
assert_contains "${tmp_dir}/manifest.txt" '^koishi/node_modules/koishi-plugin-stuhelper-group-guard/lib/index\.js$'
assert_not_contains "${tmp_dir}/manifest.txt" '/src/'
assert_not_contains "${tmp_dir}/manifest.txt" '/node_modules/.*/node_modules/'
assert_not_contains "${tmp_dir}/manifest.txt" '\.env'
assert_not_contains "${tmp_dir}/manifest.txt" 'ssh'

assert_contains "${RELEASE_RUNBOOK}" 'package-koishi-stuhelper-packages.sh'
assert_contains "${RELEASE_RUNBOOK}" 'sha256'
assert_contains "${RELEASE_RUNBOOK}" 'student-query'
assert_contains "${RELEASE_RUNBOOK}" 'enableGroupVerify.*true'
assert_contains "${KOISHI_README}" 'join\.stuhelper\.com/verify/<code>'

echo "[koishi-stuhelper-package-contract] all assertions passed"
