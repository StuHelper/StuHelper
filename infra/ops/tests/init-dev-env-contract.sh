#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
INIT_SCRIPT="${REPO_ROOT}/infra/ops/init-dev-env.sh"

fail() {
  echo "[init-dev-env-contract][error] $*" >&2
  exit 1
}

env_value() {
  local file="$1"
  local key="$2"
  python3 - "$file" "$key" <<'PY'
from pathlib import Path
import sys

path = Path(sys.argv[1])
key = sys.argv[2]

for line in path.read_text().splitlines():
    if line.startswith(f"{key}="):
        print(line.split("=", 1)[1])
        raise SystemExit(0)

raise SystemExit(f"missing env key: {key}")
PY
}

assert_env_value() {
  local file="$1"
  local key="$2"
  local expected="$3"
  local actual
  actual="$(env_value "${file}" "${key}")"
  if [[ "${actual}" != "${expected}" ]]; then
    fail "expected ${key}=${expected}, got ${actual} in ${file}"
  fi
}

cleanup_dirs=()
cleanup() {
  local dir
  for dir in "${cleanup_dirs[@]:-}"; do
    rm -rf "${dir}"
  done
}
trap cleanup EXIT

tmpdir="$(mktemp -d)"
cleanup_dirs+=("${tmpdir}")

ENV_FILE="${tmpdir}/.env" \
GENERATED_ENV_FILE="${tmpdir}/.env.generated" \
GENERATED_SECRET_ENV_FILE="${tmpdir}/.env.generated.secrets" \
GENERATED_OBS_DIR="${tmpdir}/generated/observability" \
bash "${INIT_SCRIPT}" >"${tmpdir}/stdout.log" 2>"${tmpdir}/stderr.log"

env_file="${tmpdir}/.env"
assert_env_value "${env_file}" "CORS_ORIGINS" "http://localhost:3000,http://127.0.0.1:3000,http://localhost:3001,http://127.0.0.1:3001"
assert_env_value "${env_file}" "WEB_VITE_API_URL" "/api"
assert_env_value "${env_file}" "API_IP_RATE_LIMIT" "5000"
assert_env_value "${env_file}" "API_GLOBAL_RATE_LIMIT" "50000"
assert_env_value "${env_file}" "REVIEW_RATE_POST_LIMIT" "500"
assert_env_value "${env_file}" "REVIEW_RATE_SEARCH_USER_LIMIT" "500"

legacy_dir="$(mktemp -d)"
cleanup_dirs+=("${legacy_dir}")
cp "${REPO_ROOT}/.env.example" "${legacy_dir}/.env"
python3 - "${legacy_dir}/.env" <<'PY'
from pathlib import Path
import sys

path = Path(sys.argv[1])
lines = []
for line in path.read_text().splitlines():
    if line.startswith("CORS_ORIGINS="):
        lines.append("CORS_ORIGINS=http://localhost:3000,http://localhost:3001")
    elif line.startswith("API_IP_RATE_LIMIT="):
        lines.append("API_IP_RATE_LIMIT=100")
    elif line.startswith("API_GLOBAL_RATE_LIMIT="):
        lines.append("API_GLOBAL_RATE_LIMIT=10000")
    elif line.startswith("REVIEW_RATE_POST_LIMIT="):
        lines.append("REVIEW_RATE_POST_LIMIT=5")
    elif line.startswith("REVIEW_RATE_VOTE_LIMIT="):
        lines.append("REVIEW_RATE_VOTE_LIMIT=30")
    elif line.startswith("REVIEW_RATE_REPORT_LIMIT="):
        lines.append("REVIEW_RATE_REPORT_LIMIT=10")
    elif line.startswith("REVIEW_RATE_REPLY_LIMIT="):
        lines.append("REVIEW_RATE_REPLY_LIMIT=10")
    elif line.startswith("REVIEW_RATE_WRITE_LIMIT="):
        lines.append("REVIEW_RATE_WRITE_LIMIT=10")
    elif line.startswith("WEB_VITE_API_URL="):
        lines.append("WEB_VITE_API_URL=")
    else:
        lines.append(line)
path.write_text("\n".join(lines) + "\n")
PY

ENV_FILE="${legacy_dir}/.env" \
GENERATED_ENV_FILE="${legacy_dir}/.env.generated" \
GENERATED_SECRET_ENV_FILE="${legacy_dir}/.env.generated.secrets" \
GENERATED_OBS_DIR="${legacy_dir}/generated/observability" \
bash "${INIT_SCRIPT}" >"${legacy_dir}/stdout.log" 2>"${legacy_dir}/stderr.log"

legacy_env="${legacy_dir}/.env"
assert_env_value "${legacy_env}" "CORS_ORIGINS" "http://localhost:3000,http://127.0.0.1:3000,http://localhost:3001,http://127.0.0.1:3001"
assert_env_value "${legacy_env}" "WEB_VITE_API_URL" "/api"
assert_env_value "${legacy_env}" "API_IP_RATE_LIMIT" "5000"
assert_env_value "${legacy_env}" "API_GLOBAL_RATE_LIMIT" "50000"
assert_env_value "${legacy_env}" "REVIEW_RATE_POST_LIMIT" "500"
assert_env_value "${legacy_env}" "REVIEW_RATE_SEARCH_USER_LIMIT" "500"

echo "[init-dev-env-contract] ok"
