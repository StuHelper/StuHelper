#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: infra/ops/provision-external-student-source-oracle-readonly.sh

This compatibility entry point is permanently disabled.

StuHelper may connect to the BUAA Oracle service only with the existing account
explicitly selected by the operator. Repository automation must not create,
rotate, alter, grant, revoke, unlock, or otherwise administer an Oracle account.
Only authentication and application-owned, fixed SELECT statements are allowed.

Configure the approved existing account directly in the Campus Connector secret
store. This script never opens an Oracle connection and never reads credentials.
USAGE
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
  exit 0
fi

usage >&2
echo "[stuhelper][error] Oracle account provisioning is prohibited by repository policy" >&2
exit 64
