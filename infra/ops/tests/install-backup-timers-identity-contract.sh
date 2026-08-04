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
fake_systemd_run="${tmpdir}/systemd-run"
argument_log="${tmpdir}/arguments"

cat >"${fake_systemd_run}" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
: "${SYSTEMD_RUN_ARGUMENT_LOG:?}"
printf '%s\0' "$@" >"${SYSTEMD_RUN_ARGUMENT_LOG}"
exit "${SYSTEMD_RUN_EXIT_STATUS:-0}"
SH
chmod +x "${fake_systemd_run}"

# shellcheck source=infra/ops/install-backup-timers.sh
source "${INSTALLER}"

activation_environment=(
  PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
  ENV_FILE=/opt/contract/.env.prod.shared
  SECRETS_ENV_FILE=/opt/contract/.env.prod.secrets
  LOCAL_STATE_DIR=/var/lib/contract
)
export SYSTEMD_RUN_ARGUMENT_LOG="${argument_log}"
run_activation_preflight \
  "${fake_systemd_run}" \
  contract-user \
  backup-service \
  /opt/contract \
  activation_environment

mapfile -d '' -t actual <"${argument_log}"
expected=(
  --quiet
  --wait
  --pipe
  --collect
  --service-type=exec
  --property=User=contract-user
  --property=Group=backup-service
  --property=WorkingDirectory=/opt/contract
  '--property=UnsetEnvironment=LD_PRELOAD LD_LIBRARY_PATH LD_AUDIT GCONV_PATH LOCPATH'
  /usr/bin/env
  -i
  "${activation_environment[@]}"
  /bin/bash
  --noprofile
  --norc
  /opt/contract/infra/ops/remote-preflight.sh
  --timer-activation
)
[[ "${#actual[@]}" == "${#expected[@]}" ]] ||
  fail "systemd-run activation argument count mismatch"
for index in "${!expected[@]}"; do
  [[ "${actual[${index}]}" == "${expected[${index}]}" ]] ||
    fail "systemd-run activation argument mismatch at index ${index}"
done

export SYSTEMD_RUN_EXIT_STATUS=23
if run_activation_preflight \
  "${fake_systemd_run}" \
  contract-user \
  backup-service \
  /opt/contract \
  activation_environment 2>/dev/null; then
  fail "a failed transient preflight unit must fail closed"
fi
unset SYSTEMD_RUN_EXIT_STATUS

if run_activation_preflight \
  "${tmpdir}/missing-systemd-run" \
  contract-user \
  backup-service \
  /opt/contract \
  activation_environment 2>/dev/null; then
  fail "a missing systemd-run binary must fail closed"
fi

echo "[install-backup-timers-identity-contract] all assertions passed"
