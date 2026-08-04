#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
INSTALLER="${REPO_ROOT}/infra/ops/install-backup-timers.sh"

fail() {
  echo "[install-backup-timers-identity-contract][error] $*" >&2
  exit 1
}

tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT
mkdir -p "${tmpdir}/bin"

cat >"${tmpdir}/bin/id" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
[[ "$#" == 2 && "$1" == "-G" && "$2" == "contract-user" ]] || exit 64
case "${FAKE_ID_MODE:-valid}" in
  valid) printf '%s\n' '2100 2101 2102' ;;
  invalid) printf '%s\n' '2100 not-a-gid' ;;
  empty) printf '\n' ;;
  *) exit 65 ;;
esac
SH

cat >"${tmpdir}/bin/getent" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
[[ "$#" == 2 && "$1" == "group" ]] || exit 64
[[ "${FAIL_GROUP_ID:-}" != "$2" ]] || exit 2
case "$2" in
  2100) printf '%s\n' 'contract-primary:x:2100:' ;;
  2101) printf '%s\n' 'docker:x:2101:contract-user' ;;
  2102) printf '%s\n' 'private-ca:x:2102:contract-user' ;;
  *) exit 2 ;;
esac
SH

chmod +x "${tmpdir}/bin/id" "${tmpdir}/bin/getent"

# shellcheck source=infra/ops/install-backup-timers.sh
source "${INSTALLER}"

PATH="${tmpdir}/bin:/usr/bin:/bin"
declare -a identity=()
build_runuser_identity contract-user backup-service identity

expected=(
  runuser -u contract-user -g backup-service
  -G contract-primary
  -G docker
  -G private-ca
)
[[ "${#identity[@]}" == "${#expected[@]}" ]] ||
  fail "runuser identity argument count mismatch"
for index in "${!expected[@]}"; do
  [[ "${identity[${index}]}" == "${expected[${index}]}" ]] ||
    fail "runuser identity mismatch at index ${index}"
done

if FAKE_ID_MODE=invalid build_runuser_identity contract-user backup-service identity 2>/dev/null; then
  fail "invalid group IDs must fail closed"
fi
if FAKE_ID_MODE=empty build_runuser_identity contract-user backup-service identity 2>/dev/null; then
  fail "an empty account group list must fail closed"
fi
if FAIL_GROUP_ID=2102 build_runuser_identity contract-user backup-service identity 2>/dev/null; then
  fail "an unresolvable account group must fail closed"
fi

echo "[install-backup-timers-identity-contract] all assertions passed"
