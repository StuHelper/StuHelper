#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

fail() {
  printf '[production-env-permissions-contract][error] %s\n' "$*" >&2
  exit 1
}

tmp_dir="$(mktemp -d)"
cleanup() {
  rm -rf -- "${tmp_dir}"
}
trap cleanup EXIT

template="${tmp_dir}/template.env"
env_file="${tmp_dir}/shared.env"
secrets_file="${tmp_dir}/secrets.env"
generated_file="${tmp_dir}/generated.env"
generated_secrets_file="${tmp_dir}/generated-secrets.env"
printf 'APP_ENV=production\n' >"${template}"
chmod 0664 "${template}"

ENV_FILE="${env_file}" \
ENV_TEMPLATE_FILE="${template}" \
SECRETS_ENV_FILE="${secrets_file}" \
GENERATED_ENV_FILE="${generated_file}" \
GENERATED_SECRET_ENV_FILE="${generated_secrets_file}" \
GENERATED_OBS_DIR="${tmp_dir}/observability" \
bash -c '
  set -euo pipefail
  source "$1"
  ensure_env_file
  ensure_secrets_env_file
  ensure_generated_files
' _ "${REPO_ROOT}/infra/ops/lib/common.sh"

for path in \
  "${env_file}" "${secrets_file}" \
  "${generated_file}" "${generated_secrets_file}"; do
  [[ "$(stat -c '%a' "${path}")" == "600" ]] ||
    fail "expected mode 0600 for ${path}"
done

printf '[production-env-permissions-contract] all assertions passed\n'
