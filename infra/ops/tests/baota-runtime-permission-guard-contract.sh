#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
GUARD="${REPO_ROOT}/infra/ops/guard-baota-files-set-mode.sh"
INSTALLER="${REPO_ROOT}/infra/ops/install-baota-runtime-permission-guard.sh"

fail() {
  echo "[baota-runtime-permission-guard-contract][error] $*" >&2
  exit 1
}

assert_contains() {
  local file="$1"
  local pattern="$2"
  grep -Eq -- "${pattern}" "${file}" || fail "expected ${file} to contain: ${pattern}"
}

bash -n "${GUARD}"
bash -n "${INSTALLER}"

assert_contains "${INSTALLER}" 'Before=docker\.service bt\.service'
assert_contains "${INSTALLER}" 'ExecStartPre=\$\{guard_bin\}'
assert_contains "${INSTALLER}" 'Environment=GENERATED_OBSERVABILITY_CONFIG_OWNER_UID=\$\{generated_observability_owner_uid\}'
assert_contains "${INSTALLER}" 'Environment=ALERTMANAGER_CONFIG_GID=\$\{generated_observability_group_gid\}'
assert_contains "${INSTALLER}" 'cmp --silent "\$\{permission_source\}" "\$\{permission_bin\}"'
assert_contains "${INSTALLER}" 'cmp --silent "\$\{guard_source\}" "\$\{guard_bin\}"'
assert_contains "${INSTALLER}" '--verify-installed'
assert_contains "${INSTALLER}" 'installed guard matches the current release and persistent ownership settings'
assert_contains "${INSTALLER}" 'OnUnitActiveSec=6h'
assert_contains "${INSTALLER}" 'bt\.service\.d'
assert_contains "${INSTALLER}" 'docker\.service\.d'
assert_contains "${INSTALLER}" 'tmpfiles\.d/stuhelper-baota-permission-guard\.conf'
assert_contains "${INSTALLER}" 'systemd-tmpfiles --create'
assert_contains "${INSTALLER}" 'systemctl enable "\$\{UNIT_NAME\}\.service" "\$\{UNIT_NAME\}\.timer"'
assert_contains "${INSTALLER}" 'Docker and Baota were not restarted'

tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT
marker="${tmpdir}/last_files_set_mode.pl"
jobs_file="${tmpdir}/jobs.py"

cat >"${jobs_file}" <<EOF
import os
import time

tips_file = '${marker}'
if os.path.exists(tips_file):
    if time.time() - os.path.getmtime(tips_file) < 86400:
        pass
EOF

BAOTA_JOBS_FILE="${jobs_file}" \
BAOTA_FILES_SET_MODE_MARKER="${marker}" \
BAOTA_GUARD_ALLOW_NON_ROOT_FOR_TESTS=true \
  "${GUARD}" >"${tmpdir}/guard.out"

[[ -f "${marker}" ]] || fail "guard did not create marker"
[[ "$(stat -c '%a' "${marker}")" == "600" ]] || fail "marker mode is not 600"
assert_contains "${tmpdir}/guard.out" 'refreshed marker'

printf 'tips_file = "%s"\n' "${marker}" >"${jobs_file}"
if BAOTA_JOBS_FILE="${jobs_file}" \
  BAOTA_FILES_SET_MODE_MARKER="${marker}" \
  BAOTA_GUARD_ALLOW_NON_ROOT_FOR_TESTS=true \
  "${GUARD}" >"${tmpdir}/invalid.out" 2>&1; then
  fail "expected guard to reject a changed Baota marker contract"
fi
assert_contains "${tmpdir}/invalid.out" 'no longer honors the 24-hour'

echo "[baota-runtime-permission-guard-contract] all assertions passed"
