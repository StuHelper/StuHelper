#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

usage() {
  cat <<'USAGE'
Usage:
  infra/ops/authorization-bootstrap-super-admin.sh [USER[,USER...]]
  STUHELPER_INITIAL_SUPER_ADMINS=USER[,USER...] \
    infra/ops/authorization-bootstrap-super-admin.sh

One-time, fail-closed bootstrap for the PostgreSQL authorization control plane.

The command resolves existing StuHelper users by username, creates audited
super_admin desired-state grants and queues their OpenFGA projection. It never
writes Casdoor role membership and never writes OpenFGA tuples directly. Once
any desired super_admin grant exists, it exits without changing grants, so a
normal deployment cannot silently restore a deliberately revoked principal.

The affected users must already have logged in once so the StuHelper users table
contains their Casdoor identities. Keep at least two independently controlled
initial administrators in production.
USAGE
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
  exit 0
fi

require_cmd docker
load_env

if [[ -n "${1:-}" ]]; then
  export STUHELPER_INITIAL_SUPER_ADMINS="$1"
fi
[[ -n "${STUHELPER_INITIAL_SUPER_ADMINS:-}" ]] ||
  die "STUHELPER_INITIAL_SUPER_ADMINS or positional USER[,USER...] is required"
case "${STUHELPER_INITIAL_SUPER_ADMINS}" in
  REPLACE_WITH_*) die "STUHELPER_INITIAL_SUPER_ADMINS contains a placeholder" ;;
esac

log "bootstrapping PostgreSQL-managed initial super administrators"
compose --profile prod run --rm --no-deps \
  --env STUHELPER_INITIAL_SUPER_ADMINS \
  --entrypoint /app/authorization-bootstrap \
  app \
  --apply
