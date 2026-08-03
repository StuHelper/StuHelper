#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
POLICY_SCRIPT="${REPO_ROOT}/scripts/check-dependabot-policy.sh"

fail() {
  echo "[dependabot-policy-contract][error] $*" >&2
  exit 1
}

tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT

bash "${POLICY_SCRIPT}" >/dev/null ||
  fail "repository Dependabot configuration violated the develop-first policy"

cat >"${tmpdir}/valid.yml" <<'EOF'
version: 2
updates:
  - package-ecosystem: gomod
    directory: /server
    target-branch: develop
  - package-ecosystem: npm
    directory: /clients
    target-branch: develop
EOF
bash "${POLICY_SCRIPT}" "${tmpdir}/valid.yml" >/dev/null ||
  fail "policy validator rejected complete develop-targeted update blocks"

cat >"${tmpdir}/missing-target.yml" <<'EOF'
version: 2
updates:
  - package-ecosystem: gomod
    directory: /server
  - package-ecosystem: npm
    directory: /clients
    target-branch: develop
EOF
if bash "${POLICY_SCRIPT}" "${tmpdir}/missing-target.yml" >"${tmpdir}/missing.out" 2>"${tmpdir}/missing.err"; then
  fail "policy validator accepted a version-update block without target-branch: develop"
fi
grep -q 'gomod must set target-branch: develop' "${tmpdir}/missing.err" ||
  fail "missing target-branch rejection did not identify the affected ecosystem"

cat >"${tmpdir}/wrong-target.yml" <<'EOF'
version: 2
updates:
  - package-ecosystem: docker
    directory: /
    target-branch: main
EOF
if bash "${POLICY_SCRIPT}" "${tmpdir}/wrong-target.yml" >"${tmpdir}/wrong.out" 2>"${tmpdir}/wrong.err"; then
  fail "policy validator accepted a routine version update targeting main"
fi
grep -q 'docker must set target-branch: develop' "${tmpdir}/wrong.err" ||
  fail "wrong target-branch rejection did not identify the affected ecosystem"

echo "[dependabot-policy-contract] all assertions passed"
