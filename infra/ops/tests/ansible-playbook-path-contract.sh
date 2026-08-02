#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
PLAYBOOK_DIR="${REPO_ROOT}/infra/ansible/playbooks"
DEPLOY_PLAYBOOK="${PLAYBOOK_DIR}/deploy.yml"
ROLLBACK_PLAYBOOK="${PLAYBOOK_DIR}/rollback.yml"
ANSIBLE_CONFIG="${REPO_ROOT}/infra/ansible/ansible.cfg"
ANSIBLE_REQUIREMENTS="${REPO_ROOT}/infra/ansible/requirements.txt"
CI_WORKFLOW="${REPO_ROOT}/.github/workflows/ci.yml"

fail() {
  echo "[ansible-playbook-path-contract][error] $*" >&2
  exit 1
}

assert_contains() {
  local file="$1"
  local pattern="$2"
  if ! grep -Eq -- "${pattern}" "${file}"; then
    fail "expected ${file} to contain pattern: ${pattern}"
  fi
}

assert_not_contains() {
  local file="$1"
  local pattern="$2"
  if grep -Eq -- "${pattern}" "${file}"; then
    fail "expected ${file} to not contain pattern: ${pattern}"
  fi
}

build_task="$(
  sed -n \
    '/^[[:space:]]*- name: Build deploy bundle on control node$/,/^[[:space:]]*- name: Ensure deploy directory exists$/p' \
    "${DEPLOY_PLAYBOOK}"
)"
upload_task="$(
  sed -n \
    '/^[[:space:]]*- name: Upload deploy bundle$/,/^[[:space:]]*- name: Run remote deploy$/p' \
    "${DEPLOY_PLAYBOOK}"
)"

[[ -n "${build_task}" ]] || fail "missing deploy bundle build task"
[[ -n "${upload_task}" ]] || fail "missing deploy bundle upload task"

assert_contains "${DEPLOY_PLAYBOOK}" '^  gather_facts: false$'
assert_contains "${ROLLBACK_PLAYBOOK}" '^  gather_facts: false$'
grep -Eq '^[[:space:]]+argv:$' <<<"${build_task}" ||
  fail "deploy bundle build must use argv"
grep -Eq '^[[:space:]]+- "\{\{ playbook_dir \}\}/../../ops/build-deploy-bundle\.sh"$' <<<"${build_task}" ||
  fail "deploy bundle script must be anchored to playbook_dir"
grep -Eq '^[[:space:]]+- deploy-bundle$' <<<"${build_task}" ||
  fail "deploy bundle build task must expose the CI smoke tag"
if grep -Eq 'generated/deploy|^[[:space:]]+cmd:' <<<"${build_task}"; then
  fail "deploy bundle build must use the script's repository-anchored default output"
fi

grep -Eq \
  '^[[:space:]]+src: "\{\{ playbook_dir \}\}/../../generated/deploy/stuhelper-deploy-bundle\.tar\.gz"$' \
  <<<"${upload_task}" ||
  fail "deploy bundle upload source must be anchored to playbook_dir"

resolved_script="$(realpath -m "${PLAYBOOK_DIR}/../../ops/build-deploy-bundle.sh")"
resolved_bundle="$(realpath -m "${PLAYBOOK_DIR}/../../generated/deploy/stuhelper-deploy-bundle.tar.gz")"
[[ "${resolved_script}" == "${REPO_ROOT}/infra/ops/build-deploy-bundle.sh" ]] ||
  fail "playbook script path resolves outside infra/ops: ${resolved_script}"
[[ -x "${resolved_script}" ]] || fail "resolved deploy bundle script is not executable"
[[ "${resolved_bundle}" == "${REPO_ROOT}/infra/generated/deploy/stuhelper-deploy-bundle.tar.gz" ]] ||
  fail "playbook upload path does not match the bundle script default: ${resolved_bundle}"

for release_playbook in "${DEPLOY_PLAYBOOK}" "${ROLLBACK_PLAYBOOK}"; do
  assert_contains "${release_playbook}" "lookup\('env', 'REGISTRY_PULL_TOKEN'\)"
  assert_contains "${release_playbook}" "lookup\('env', 'REGISTRY_USERNAME'\)"
  assert_contains "${release_playbook}" 'stdin: "\{\{ registry_pull_token \}\}\\n"'
  assert_contains "${release_playbook}" '^      no_log: true$'
  assert_contains "${release_playbook}" 'CI_REGISTRY_USERNAME: "\{\{ registry_username \}\}"'
  assert_contains "${release_playbook}" 'remote-ci-release\.sh (deploy|rollback)'
  assert_not_contains "${release_playbook}" 'remote-prod-(deploy|rollback)\.sh'
done

assert_contains "${ANSIBLE_REQUIREMENTS}" '^ansible-core==2\.20\.7$'
assert_contains "${ANSIBLE_CONFIG}" '^stdout_callback = default$'
assert_contains "${ANSIBLE_CONFIG}" '^callback_result_format = yaml$'
assert_not_contains "${ANSIBLE_CONFIG}" '^stdout_callback = yaml$'

assert_contains "${CI_WORKFLOW}" '"\$\{RUNNER_TEMP\}/stuhelper-ansible/bin/pip" install'
assert_contains "${CI_WORKFLOW}" '^[[:space:]]+--requirement infra/ansible/requirements\.txt$'
assert_contains "${CI_WORKFLOW}" 'ansible-playbook .*--syntax-check'
assert_contains "${CI_WORKFLOW}" 'ansible-playbook .*--tags deploy-bundle'

echo "[ansible-playbook-path-contract] all assertions passed"
