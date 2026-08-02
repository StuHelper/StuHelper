#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
RENDER_SCRIPT="${REPO_ROOT}/infra/ops/render-redis-acl.sh"

fail() {
  echo "[redis-acl-contract][error] $*" >&2
  exit 1
}

tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT

run_renderer() {
  local env_file="$1"
  local acl_dir="$2"
  ENV_FILE="${env_file}" \
  ENV_TEMPLATE_FILE="${env_file}" \
  SECRETS_ENV_FILE="" \
  GENERATED_ENV_FILE="${tmpdir}/generated.env" \
  GENERATED_SECRET_ENV_FILE="${tmpdir}/generated.secrets.env" \
  REDIS_ACL_DIR="${acl_dir}" \
    bash "${RENDER_SCRIPT}"
}

write_env() {
  local path="$1"
  local app_user="$2"
  local app_password="$3"
  local metrics_user="$4"
  local metrics_password="$5"
  {
    printf 'REDIS_USERNAME=%s\n' "${app_user}"
    printf 'REDIS_PASSWORD=%s\n' "${app_password}"
    printf 'REDIS_EXPORTER_USERNAME=%s\n' "${metrics_user}"
    printf 'REDIS_EXPORTER_PASSWORD=%s\n' "${metrics_password}"
  } >"${path}"
}

append_parity_maintenance_env() {
  local path="$1"
  local maintenance_user="$2"
  local maintenance_password="$3"
  {
    printf 'APP_ENV=prod-parity\n'
    printf 'REDIS_PROD_PARITY_MAINTENANCE_USERNAME=%s\n' "${maintenance_user}"
    printf 'REDIS_PROD_PARITY_MAINTENANCE_PASSWORD=%s\n' "${maintenance_password}"
  } >>"${path}"
}

valid_env="${tmpdir}/valid.env"
valid_acl_dir="${tmpdir}/valid-acl"
app_password="app-secret-with-special-safe-text"
metrics_password="metrics-secret-with-different-text"
write_env "${valid_env}" "stuhelper_app" "${app_password}" "stuhelper_metrics" "${metrics_password}"
run_renderer "${valid_env}" "${valid_acl_dir}" >"${tmpdir}/valid.stdout" 2>"${tmpdir}/valid.stderr"

acl_file="${valid_acl_dir}/users.acl"
[[ -f "${acl_file}" ]] || fail "renderer did not create users.acl"
[[ "$(stat -c '%a' "${acl_file}")" == "600" ]] || fail "users.acl must use mode 600"
grep -Eq '^user default off$' "${acl_file}" || fail "default Redis user must be disabled"
grep -Eq '^user stuhelper_app reset on #[0-9a-f]{64} .* -@all ' "${acl_file}" ||
  fail "application user must use a hash and explicit command allowlist"
grep -Eq '^user stuhelper_metrics reset on #[0-9a-f]{64} resetkeys resetchannels -@all ' "${acl_file}" ||
  fail "metrics user must use a hash, no key patterns, and explicit commands"
grep -Fq '+config|get' "${acl_file}" || fail "metrics user must be able to collect safe config metrics"
grep -Fq '+info' "${acl_file}" || fail "metrics user must be able to collect INFO metrics"
metrics_rule="$(grep '^user stuhelper_metrics ' "${acl_file}")"
for forbidden_rule in '+get' '+set' '+scan' '+eval' '+client' '+slowlog' '+latency'; do
  if [[ " ${metrics_rule} " == *" ${forbidden_rule} "* ]]; then
    fail "metrics user contains an overbroad command rule: ${forbidden_rule}"
  fi
done
for required_rule in '+client|setname' '+slowlog|len' '+slowlog|get' '+latency|latest' '+latency|histogram'; do
  if [[ " ${metrics_rule} " != *" ${required_rule} "* ]]; then
    fail "metrics user is missing required command rule: ${required_rule}"
  fi
done
if grep -Fq '+@all' "${acl_file}"; then
  fail "no Redis runtime user may receive +@all"
fi
if grep -Fq "${app_password}" "${acl_file}" || grep -Fq "${metrics_password}" "${acl_file}"; then
  fail "Redis ACL must not persist plaintext passwords"
fi
if grep -Fq "${app_password}" "${tmpdir}/valid.stdout" ||
   grep -Fq "${metrics_password}" "${tmpdir}/valid.stdout" ||
   grep -Fq "${app_password}" "${tmpdir}/valid.stderr" ||
   grep -Fq "${metrics_password}" "${tmpdir}/valid.stderr"; then
  fail "renderer output must not disclose Redis passwords"
fi

app_hash="$(printf '%s' "${app_password}" | openssl dgst -sha256 -r | awk '{print $1}')"
metrics_hash="$(printf '%s' "${metrics_password}" | openssl dgst -sha256 -r | awk '{print $1}')"
grep -Fq "#${app_hash}" "${acl_file}" || fail "application password verifier is incorrect"
grep -Fq "#${metrics_hash}" "${acl_file}" || fail "metrics password verifier is incorrect"

parity_env="${tmpdir}/parity.env"
parity_acl_dir="${tmpdir}/parity-acl"
maintenance_password="maintenance-secret-with-third-text"
write_env "${parity_env}" "stuhelper_app" "${app_password}" "stuhelper_metrics" "${metrics_password}"
append_parity_maintenance_env "${parity_env}" "stuhelper_parity_maintenance" "${maintenance_password}"
run_renderer "${parity_env}" "${parity_acl_dir}" >"${tmpdir}/parity.stdout" 2>"${tmpdir}/parity.stderr"

parity_acl_file="${parity_acl_dir}/users.acl"
maintenance_rule="$(grep '^user stuhelper_parity_maintenance ' "${parity_acl_file}")"
[[ -n "${maintenance_rule}" ]] || fail "prod-parity maintenance user was not rendered"
for required_rule in '~course:*' '~review:*' '~cache:version:course*' '~cache:version:review*' '~rl:*' '-@all' '+scan' '+del' '+client|setname'; do
  if [[ " ${maintenance_rule} " != *" ${required_rule} "* ]]; then
    fail "prod-parity maintenance user is missing required rule: ${required_rule}"
  fi
done
for forbidden_rule in '~*' '+get' '+set' '+eval' '+info' '+config|get' '+@all'; do
  if [[ " ${maintenance_rule} " == *" ${forbidden_rule} "* ]]; then
    fail "prod-parity maintenance user contains an overbroad rule: ${forbidden_rule}"
  fi
done
maintenance_hash="$(printf '%s' "${maintenance_password}" | openssl dgst -sha256 -r | awk '{print $1}')"
grep -Fq "#${maintenance_hash}" "${parity_acl_file}" || fail "maintenance password verifier is incorrect"
if grep -Fq "${maintenance_password}" "${parity_acl_file}" ||
   grep -Fq "${maintenance_password}" "${tmpdir}/parity.stdout" ||
   grep -Fq "${maintenance_password}" "${tmpdir}/parity.stderr"; then
  fail "renderer must not disclose the prod-parity maintenance password"
fi

production_maintenance_env="${tmpdir}/production-maintenance.env"
write_env "${production_maintenance_env}" "stuhelper_app" "${app_password}" "stuhelper_metrics" "${metrics_password}"
append_parity_maintenance_env "${production_maintenance_env}" "stuhelper_parity_maintenance" "${maintenance_password}"
sed -i 's/^APP_ENV=prod-parity$/APP_ENV=production/' "${production_maintenance_env}"
if run_renderer "${production_maintenance_env}" "${tmpdir}/production-maintenance-acl" >"${tmpdir}/production-maintenance.stdout" 2>"${tmpdir}/production-maintenance.stderr"; then
  fail "renderer enabled the local maintenance identity in production"
fi
grep -Fq 'only allowed in APP_ENV=prod-parity' "${tmpdir}/production-maintenance.stderr" ||
  fail "production maintenance identity rejection must be explicit"

incomplete_maintenance_env="${tmpdir}/incomplete-maintenance.env"
write_env "${incomplete_maintenance_env}" "stuhelper_app" "${app_password}" "stuhelper_metrics" "${metrics_password}"
printf 'APP_ENV=prod-parity\nREDIS_PROD_PARITY_MAINTENANCE_USERNAME=stuhelper_parity_maintenance\n' >>"${incomplete_maintenance_env}"
if run_renderer "${incomplete_maintenance_env}" "${tmpdir}/incomplete-maintenance-acl" >"${tmpdir}/incomplete-maintenance.stdout" 2>"${tmpdir}/incomplete-maintenance.stderr"; then
  fail "renderer accepted an incomplete prod-parity maintenance identity"
fi
grep -Fq 'REDIS_PROD_PARITY_MAINTENANCE_PASSWORD is required' "${tmpdir}/incomplete-maintenance.stderr" ||
  fail "incomplete maintenance identity rejection must be explicit"

same_password_env="${tmpdir}/same-password.env"
write_env "${same_password_env}" "stuhelper_app" "shared-secret" "stuhelper_metrics" "shared-secret"
if run_renderer "${same_password_env}" "${tmpdir}/same-password-acl" >"${tmpdir}/same-password.stdout" 2>"${tmpdir}/same-password.stderr"; then
  fail "renderer accepted application/exporter password reuse"
fi
grep -Fq 'REDIS_PASSWORD and REDIS_EXPORTER_PASSWORD must be different' "${tmpdir}/same-password.stderr" ||
  fail "password reuse rejection must be explicit"

same_user_env="${tmpdir}/same-user.env"
write_env "${same_user_env}" "stuhelper" "app-secret" "stuhelper" "metrics-secret"
if run_renderer "${same_user_env}" "${tmpdir}/same-user-acl" >"${tmpdir}/same-user.stdout" 2>"${tmpdir}/same-user.stderr"; then
  fail "renderer accepted application/exporter username reuse"
fi
grep -Fq 'REDIS_USERNAME and REDIS_EXPORTER_USERNAME must be different' "${tmpdir}/same-user.stderr" ||
  fail "username reuse rejection must be explicit"

unsafe_user_env="${tmpdir}/unsafe-user.env"
write_env "${unsafe_user_env}" 'unsafe user' "app-secret" "stuhelper_metrics" "metrics-secret"
if run_renderer "${unsafe_user_env}" "${tmpdir}/unsafe-user-acl" >"${tmpdir}/unsafe-user.stdout" 2>"${tmpdir}/unsafe-user.stderr"; then
  fail "renderer accepted an unsafe ACL username"
fi
grep -Fq 'REDIS_USERNAME must match' "${tmpdir}/unsafe-user.stderr" ||
  fail "unsafe username rejection must be explicit"

symlink_dir="${tmpdir}/symlink-acl"
mkdir -p "${symlink_dir}"
ln -s "${tmpdir}/symlink-target" "${symlink_dir}/users.acl"
if run_renderer "${valid_env}" "${symlink_dir}" >"${tmpdir}/symlink.stdout" 2>"${tmpdir}/symlink.stderr"; then
  fail "renderer followed a users.acl symlink"
fi
grep -Fq 'Redis ACL file must not be a symlink' "${tmpdir}/symlink.stderr" ||
  fail "ACL symlink rejection must be explicit"

echo "[redis-acl-contract] all assertions passed"
