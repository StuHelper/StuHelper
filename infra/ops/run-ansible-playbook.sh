#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
ANSIBLE_DIR="${REPO_ROOT}/infra/ansible"
REQUIREMENTS_FILE="${ANSIBLE_DIR}/requirements.txt"

die() {
  echo "[stuhelper][error] $*" >&2
  exit 1
}

usage() {
  cat >&2 <<'USAGE'
Usage: run-ansible-playbook.sh <bootstrap|deploy|rollback> <staging|production> [ansible-playbook options]
USAGE
  exit 2
}

[[ $# -ge 2 ]] || usage
action="$1"
environment="$2"
shift 2

case "${action}" in
  bootstrap|deploy|rollback) ;;
  *) usage ;;
esac
case "${environment}" in
  staging|production) ;;
  *) usage ;;
esac

inventory_file="${ANSIBLE_DIR}/inventory/${environment}.ini"
playbook_file="${ANSIBLE_DIR}/playbooks/${action}.yml"
[[ -f "${inventory_file}" && -s "${inventory_file}" ]] || \
  die "missing non-empty Ansible inventory: ${inventory_file}; copy the matching .example.ini to this ignored path and replace every placeholder"
[[ -f "${playbook_file}" ]] || die "missing Ansible playbook: ${playbook_file}"
[[ -f "${REQUIREMENTS_FILE}" ]] || die "missing Ansible requirements: ${REQUIREMENTS_FILE}"

if grep -Eiq \
  'your\.(prod|staging)\.host|your[_-]?[a-z]*host|replace_with|example\.(com|test|invalid)' \
  "${inventory_file}"; then
  die "Ansible inventory still contains an example or replacement placeholder: ${inventory_file}"
fi

ansible_package="$(sed -n '/^ansible-core==[0-9][0-9.]*$/p' "${REQUIREMENTS_FILE}" | head -n 1)"
[[ "${ansible_package}" =~ ^ansible-core==[0-9]+\.[0-9]+\.[0-9]+$ ]] || \
  die "requirements.txt must pin exactly one ansible-core X.Y.Z version"

if command -v ansible-playbook >/dev/null 2>&1 && command -v ansible-inventory >/dev/null 2>&1; then
  ansible_playbook=(ansible-playbook)
  ansible_inventory=(ansible-inventory)
elif command -v uvx >/dev/null 2>&1; then
  ansible_playbook=(uvx --from "${ansible_package}" ansible-playbook)
  ansible_inventory=(uvx --from "${ansible_package}" ansible-inventory)
else
  die "ansible-playbook is unavailable and uvx is not installed; install ${ansible_package} from infra/ansible/requirements.txt"
fi

inventory_json="$(
  cd "${ANSIBLE_DIR}"
  "${ansible_inventory[@]}" -i "${inventory_file}" --list
)" || die "failed to parse Ansible inventory: ${inventory_file}"

printf '%s' "${inventory_json}" | python3 -c '
import json
import sys

payload = json.load(sys.stdin)
group = payload.get("stuhelper")
if not isinstance(group, dict):
    raise SystemExit("inventory is missing the stuhelper group")
hosts = group.get("hosts")
if not isinstance(hosts, list) or not hosts:
    raise SystemExit("inventory stuhelper group has no hosts")
hostvars = payload.get("_meta", {}).get("hostvars", {})
for host in hosts:
    if not isinstance(host, str) or not host.strip():
        raise SystemExit("inventory contains an empty host alias")
    variables = hostvars.get(host, {})
    address = str(variables.get("ansible_host", host)).strip().lower()
    if address in {"localhost", "127.0.0.1", "::1"}:
        raise SystemExit(f"inventory host {host} resolves to a local address")
' || die "Ansible inventory did not produce a non-local stuhelper host set: ${inventory_file}"

cd "${ANSIBLE_DIR}"
if [[ "${action}" == "bootstrap" ]]; then
  exec "${ansible_playbook[@]}" -i "${inventory_file}" "${playbook_file}" "$@"
fi
exec "${ansible_playbook[@]}" -i "${inventory_file}" "${playbook_file}" -e "env_name=${environment}" "$@"
