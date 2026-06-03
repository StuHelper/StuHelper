#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
IMPORT_SCRIPT="${REPO_ROOT}/infra/ops/baota-compose-import-image-tars.sh"
REFRESH_SCRIPT="${REPO_ROOT}/infra/ops/baota-compose-refresh-image-refs.sh"

fail() {
  echo "[baota-compose-import-image-tars-contract][error] $*" >&2
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

[[ -f "${IMPORT_SCRIPT}" ]] || fail "missing script: ${IMPORT_SCRIPT}"
[[ -f "${REFRESH_SCRIPT}" ]] || fail "missing script: ${REFRESH_SCRIPT}"
bash -n "${IMPORT_SCRIPT}"

tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT

compose_root="${tmpdir}/compose"
source_dir="${compose_root}/source"
mkdir -p "${source_dir}/infra/ops" "${compose_root}/backups" "${tmpdir}/bin"
cp "${REFRESH_SCRIPT}" "${source_dir}/infra/ops/baota-compose-refresh-image-refs.sh"
chmod +x "${source_dir}/infra/ops/baota-compose-refresh-image-refs.sh"

cat >"${compose_root}/docker-compose.yml" <<'YAML'
services:
  migrate:
    image: stuhelper/backend:old
  app:
    image: stuhelper/backend:old
  frontend:
    image: stuhelper/frontend:old
  admin:
    image: stuhelper/admin:old
YAML

cat >"${source_dir}/.env.prod.shared" <<'ENV'
BACKEND_IMAGE_REF=stuhelper/backend:old
FRONTEND_IMAGE_REF=stuhelper/frontend:old
ADMIN_IMAGE_REF=stuhelper/admin:old
ENV

make_tar() {
  local name="$1"
  local path="${tmpdir}/${name}.tar.gz"
  printf '%s\n' "${name}" | gzip -1 >"${path}"
  printf '%s\n' "${path}"
}

backend_tar="$(make_tar backend)"
frontend_tar="$(make_tar frontend)"
admin_tar="$(make_tar admin)"
backend_sha="$(sha256sum "${backend_tar}" | awk '{print $1}')"
frontend_sha="$(sha256sum "${frontend_tar}" | awk '{print $1}')"
admin_sha="$(sha256sum "${admin_tar}" | awk '{print $1}')"

common_args=(
  --compose-root "${compose_root}"
  --backend-ref registry.example.com/stuhelper/backend:sha-1111111
  --backend-tar "${backend_tar}"
  --backend-sha256 "${backend_sha}"
  --frontend-ref registry.example.com/stuhelper/frontend:sha-2222222
  --frontend-tar "${frontend_tar}"
  --frontend-sha256 "${frontend_sha}"
  --admin-ref registry.example.com/stuhelper/admin:sha-3333333
  --admin-tar "${admin_tar}"
  --admin-sha256 "${admin_sha}"
)

"${IMPORT_SCRIPT}" "${common_args[@]}" >"${tmpdir}/dry-run.out"
assert_contains "${tmpdir}/dry-run.out" 'dry-run'
assert_contains "${tmpdir}/dry-run.out" 'registry\.example\.com/stuhelper/backend:sha-1111111'
assert_contains "${source_dir}/.env.prod.shared" 'stuhelper/backend:old'
assert_contains "${compose_root}/docker-compose.yml" 'stuhelper/frontend:old'

cat >"${tmpdir}/bin/docker" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
echo "docker $*" >>"${FAKE_DOCKER_LOG}"
if [[ "${1:-}" == "load" && "${2:-}" == "-i" ]]; then
  cat "${3}" >/dev/null
else
  cat >/dev/null
fi
SH
chmod +x "${tmpdir}/bin/docker"

FAKE_DOCKER_LOG="${tmpdir}/docker.log" \
DOCKER_BIN="${tmpdir}/bin/docker" \
"${IMPORT_SCRIPT}" "${common_args[@]}" --apply >"${tmpdir}/apply.out"

assert_contains "${tmpdir}/docker.log" 'docker load'
assert_contains "${source_dir}/.env.prod.shared" 'BACKEND_IMAGE_REF=registry\.example\.com/stuhelper/backend:sha-1111111'
assert_contains "${source_dir}/.env.prod.shared" 'FRONTEND_IMAGE_REF=registry\.example\.com/stuhelper/frontend:sha-2222222'
assert_contains "${source_dir}/.env.prod.shared" 'ADMIN_IMAGE_REF=registry\.example\.com/stuhelper/admin:sha-3333333'
assert_contains "${compose_root}/docker-compose.yml" 'registry\.example\.com/stuhelper/backend:sha-1111111'
assert_contains "${compose_root}/docker-compose.yml" 'registry\.example\.com/stuhelper/frontend:sha-2222222'
assert_contains "${compose_root}/docker-compose.yml" 'registry\.example\.com/stuhelper/admin:sha-3333333'
assert_not_contains "${compose_root}/docker-compose.yml" 'stuhelper/backend:old'

backup_count="$(find "${compose_root}/backups" -type f -name 'docker-compose.yml.before-image-refresh-*' | wc -l | tr -d ' ')"
[[ "${backup_count}" == "1" ]] || fail "expected one compose backup, got ${backup_count}"

bad_args=("${common_args[@]}")
bad_args[7]="0000000000000000000000000000000000000000000000000000000000000000"
if "${IMPORT_SCRIPT}" "${bad_args[@]}" >"${tmpdir}/bad.out" 2>&1; then
  fail "bad sha256 should fail"
fi
assert_contains "${tmpdir}/bad.out" 'FAILED|did NOT match|sha256'

echo "[baota-compose-import-image-tars-contract] all assertions passed"
