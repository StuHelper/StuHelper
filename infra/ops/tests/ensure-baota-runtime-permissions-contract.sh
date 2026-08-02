#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
SCRIPT="${REPO_ROOT}/infra/ops/ensure-baota-runtime-permissions.sh"
POSTGRES_TLS_SCRIPT="${REPO_ROOT}/infra/ops/render-postgres-tls.sh"

fail() {
  echo "[ensure-baota-runtime-permissions-contract][error] $*" >&2
  exit 1
}

assert_mode() {
  local expected="$1"
  local path="$2"
  local actual
  actual="$(stat -c '%a' "${path}")"
  [[ "${actual}" == "${expected}" ]] || fail "expected ${path} mode ${expected}, got ${actual}"
}

assert_contains() {
  local file="$1"
  local pattern="$2"
  if ! grep -Eq -- "${pattern}" "${file}"; then
    fail "expected ${file} to contain pattern: ${pattern}"
  fi
}

[[ -f "${SCRIPT}" ]] || fail "missing script: ${SCRIPT}"
bash -n "${SCRIPT}"

assert_contains "${POSTGRES_TLS_SCRIPT}" '\[\[ -f "\$\{SERVER_KEY\}" \]\] && chmod 600 "\$\{SERVER_KEY\}"'

tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT

source_dir="${tmpdir}/source"
casdoor_root="${tmpdir}/casdoor"
openlist_root="${tmpdir}/openlist"
mkdir -p \
  "${source_dir}/infra/generated/postgres" \
  "${source_dir}/infra/generated/postgres-client-ca" \
  "${source_dir}/infra/generated/redis" \
  "${source_dir}/infra/generated/redis-client-ca" \
  "${source_dir}/infra/generated/external-student-source-client-ca" \
  "${source_dir}/infra/generated/object-storage" \
  "${source_dir}/infra/generated/object-storage-client-ca" \
  "${source_dir}/infra/generated/observability/prometheus" \
  "${source_dir}/infra/generated/observability/alertmanager" \
  "${source_dir}/infra/postgres" \
  "${source_dir}/infra/redis" \
  "${source_dir}/infra/observability/alloy" \
  "${source_dir}/infra/observability/loki" \
  "${source_dir}/infra/observability/tempo" \
  "${source_dir}/infra/observability/grafana/dashboards" \
  "${casdoor_root}/conf" \
  "${casdoor_root}/logs/archive" \
  "${openlist_root}"

touch \
  "${source_dir}/infra/generated/postgres/ca.key" \
  "${source_dir}/infra/generated/postgres/ca.crt" \
  "${source_dir}/infra/generated/postgres/server.key" \
  "${source_dir}/infra/generated/postgres/server.crt" \
  "${source_dir}/infra/generated/postgres-client-ca/ca.crt" \
  "${source_dir}/infra/generated/redis/ca.key" \
  "${source_dir}/infra/generated/redis/ca.crt" \
  "${source_dir}/infra/generated/redis/server.key" \
  "${source_dir}/infra/generated/redis/server.crt" \
  "${source_dir}/infra/generated/redis/users.acl" \
  "${source_dir}/infra/generated/redis-client-ca/ca.crt" \
  "${source_dir}/infra/generated/external-student-source-client-ca/ca.crt" \
  "${source_dir}/infra/generated/object-storage/ca.key" \
  "${source_dir}/infra/generated/object-storage/ca.crt" \
  "${source_dir}/infra/generated/object-storage/private.key" \
  "${source_dir}/infra/generated/object-storage/public.crt" \
  "${source_dir}/infra/generated/object-storage/s3.json" \
  "${source_dir}/infra/generated/object-storage-client-ca/ca.crt" \
  "${source_dir}/infra/generated/observability/prometheus/prometheus.yml" \
  "${source_dir}/infra/generated/observability/alertmanager/alertmanager.yml" \
  "${source_dir}/infra/postgres/init-extra-dbs.sh" \
  "${source_dir}/infra/postgres/docker-entrypoint-with-tls.sh" \
  "${source_dir}/infra/postgres/pg_hba.prod.conf" \
  "${source_dir}/infra/redis/docker-entrypoint-with-secrets.sh" \
  "${source_dir}/infra/observability/alloy/config.alloy" \
  "${source_dir}/infra/observability/loki/loki.yaml" \
  "${source_dir}/infra/observability/tempo/tempo.yaml" \
  "${source_dir}/infra/observability/grafana/dashboards/overview.json" \
  "${casdoor_root}/conf/app.conf" \
  "${casdoor_root}/logs/casdoor.log" \
  "${casdoor_root}/logs/archive/casdoor.log.1" \
  "${openlist_root}/config.json" \
  "${openlist_root}/data.db"

printf '{}\n' >"${openlist_root}/config.json"

chmod 700 \
  "${source_dir}/infra/generated/postgres" \
  "${source_dir}/infra/generated/postgres-client-ca" \
  "${source_dir}/infra/generated/redis" \
  "${source_dir}/infra/generated/redis-client-ca" \
  "${source_dir}/infra/generated/external-student-source-client-ca" \
  "${source_dir}/infra/generated/object-storage" \
  "${source_dir}/infra/generated/object-storage-client-ca" \
  "${source_dir}/infra/postgres" \
  "${source_dir}/infra/redis" \
  "${source_dir}/infra/observability" \
  "${openlist_root}"
chmod 600 "${source_dir}/infra/generated/redis/users.acl"
chmod 700 "${casdoor_root}/conf" "${casdoor_root}/logs"
chmod 600 "${casdoor_root}/conf/app.conf" "${casdoor_root}/logs/casdoor.log"

"${SCRIPT}" \
  --source-dir "${source_dir}" \
  --casdoor-compose-root "${casdoor_root}" \
  --openlist-data-dir "${openlist_root}" >"${tmpdir}/dry-run.out"
assert_contains "${tmpdir}/dry-run.out" 'dry-run complete'
assert_mode 700 "${source_dir}/infra/generated/postgres"
assert_mode 600 "${source_dir}/infra/generated/redis/users.acl"

# Recovered Baota trees can contain a non-traversable generated observability
# parent even when its child directories have usable modes. The repair must
# normalize the parent before walking or updating those children.
chmod 600 "${source_dir}/infra/generated/observability"

"${SCRIPT}" \
  --source-dir "${source_dir}" \
  --casdoor-compose-root "${casdoor_root}" \
  --openlist-data-dir "${openlist_root}" \
  --apply >"${tmpdir}/apply.out"
assert_contains "${tmpdir}/apply.out" 'runtime bind-mount permissions normalized'

assert_mode 755 "${source_dir}/infra/generated/postgres"
assert_mode 600 "${source_dir}/infra/generated/postgres/ca.key"
assert_mode 600 "${source_dir}/infra/generated/postgres/server.key"
assert_mode 644 "${source_dir}/infra/generated/postgres/ca.crt"
assert_mode 644 "${source_dir}/infra/generated/postgres/server.crt"
assert_mode 755 "${source_dir}/infra/generated/postgres-client-ca"
assert_mode 644 "${source_dir}/infra/generated/postgres-client-ca/ca.crt"

assert_mode 755 "${source_dir}/infra/generated/redis"
assert_mode 600 "${source_dir}/infra/generated/redis/ca.key"
assert_mode 644 "${source_dir}/infra/generated/redis/ca.crt"
assert_mode 600 "${source_dir}/infra/generated/redis/server.key"
assert_mode 644 "${source_dir}/infra/generated/redis/server.crt"
assert_mode 600 "${source_dir}/infra/generated/redis/users.acl"
assert_mode 755 "${source_dir}/infra/generated/redis-client-ca"
assert_mode 644 "${source_dir}/infra/generated/redis-client-ca/ca.crt"
assert_mode 755 "${source_dir}/infra/generated/external-student-source-client-ca"
assert_mode 644 "${source_dir}/infra/generated/external-student-source-client-ca/ca.crt"

assert_mode 755 "${source_dir}/infra/generated/object-storage"
assert_mode 600 "${source_dir}/infra/generated/object-storage/ca.key"
assert_mode 600 "${source_dir}/infra/generated/object-storage/private.key"
assert_mode 600 "${source_dir}/infra/generated/object-storage/s3.json"
assert_mode 644 "${source_dir}/infra/generated/object-storage/ca.crt"
assert_mode 644 "${source_dir}/infra/generated/object-storage/public.crt"
assert_mode 755 "${source_dir}/infra/generated/object-storage-client-ca"
assert_mode 644 "${source_dir}/infra/generated/object-storage-client-ca/ca.crt"

assert_mode 755 "${source_dir}/infra/postgres"
assert_mode 755 "${source_dir}/infra/postgres/init-extra-dbs.sh"
assert_mode 755 "${source_dir}/infra/postgres/docker-entrypoint-with-tls.sh"
assert_mode 644 "${source_dir}/infra/postgres/pg_hba.prod.conf"
assert_mode 755 "${source_dir}/infra/redis"
assert_mode 755 "${source_dir}/infra/redis/docker-entrypoint-with-secrets.sh"

assert_mode 755 "${source_dir}/infra/observability/alloy"
assert_mode 644 "${source_dir}/infra/observability/alloy/config.alloy"
assert_mode 755 "${source_dir}/infra/observability/loki"
assert_mode 644 "${source_dir}/infra/observability/loki/loki.yaml"
assert_mode 755 "${source_dir}/infra/observability/tempo"
assert_mode 644 "${source_dir}/infra/observability/tempo/tempo.yaml"
assert_mode 755 "${source_dir}/infra/observability/grafana/dashboards"
assert_mode 644 "${source_dir}/infra/observability/grafana/dashboards/overview.json"

assert_mode 755 "${source_dir}/infra/generated/observability"
assert_mode 750 "${source_dir}/infra/generated/observability/prometheus"
assert_mode 640 "${source_dir}/infra/generated/observability/prometheus/prometheus.yml"
assert_mode 750 "${source_dir}/infra/generated/observability/alertmanager"
assert_mode 640 "${source_dir}/infra/generated/observability/alertmanager/alertmanager.yml"

assert_mode 750 "${casdoor_root}/conf"
assert_mode 640 "${casdoor_root}/conf/app.conf"
assert_mode 750 "${casdoor_root}/logs"
assert_mode 750 "${casdoor_root}/logs/archive"
assert_mode 640 "${casdoor_root}/logs/archive/casdoor.log.1"

assert_mode 750 "${openlist_root}"
assert_mode 640 "${openlist_root}/config.json"
assert_mode 640 "${openlist_root}/data.db"

"${SCRIPT}" \
  --source-dir "${source_dir}" \
  --casdoor-compose-root "${casdoor_root}" \
  --openlist-data-dir "${openlist_root}" \
  --skip-casdoor \
  --skip-openlist \
  --apply >"${tmpdir}/skip.out"
assert_contains "${tmpdir}/skip.out" 'runtime bind-mount permissions normalized'

: >"${openlist_root}/config.json"
if "${SCRIPT}" \
  --source-dir "${source_dir}" \
  --skip-casdoor \
  --openlist-data-dir "${openlist_root}" \
  --apply >"${tmpdir}/empty-openlist.out" 2>&1; then
  fail "expected empty OpenList config to fail validation"
fi
assert_contains "${tmpdir}/empty-openlist.out" 'OpenList config is empty'

printf '{invalid\n' >"${openlist_root}/config.json"
if command -v jq >/dev/null 2>&1; then
  if "${SCRIPT}" \
    --source-dir "${source_dir}" \
    --skip-casdoor \
    --openlist-data-dir "${openlist_root}" \
    --apply >"${tmpdir}/invalid-openlist.out" 2>&1; then
    fail "expected invalid OpenList JSON to fail validation"
  fi
  assert_contains "${tmpdir}/invalid-openlist.out" 'OpenList config is not valid JSON'
fi

echo "[ensure-baota-runtime-permissions-contract] all assertions passed"
