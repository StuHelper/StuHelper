#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT_GUESS="$(cd "${SCRIPT_DIR}/../.." && pwd)"

if [[ -z "${ENV_FILE+x}" && -f "${REPO_ROOT_GUESS}/.env.prod.shared" ]]; then
  export ENV_FILE="${REPO_ROOT_GUESS}/.env.prod.shared"
fi
if [[ -z "${SECRETS_ENV_FILE+x}" ]]; then
  if [[ -f "${REPO_ROOT_GUESS}/.env.prod.secrets.local" ]]; then
    export SECRETS_ENV_FILE="${REPO_ROOT_GUESS}/.env.prod.secrets.local"
  elif [[ -f "${REPO_ROOT_GUESS}/.env.prod.secrets" ]]; then
    export SECRETS_ENV_FILE="${REPO_ROOT_GUESS}/.env.prod.secrets"
  fi
fi
if [[ -z "${GENERATED_ENV_FILE+x}" && -f "${REPO_ROOT_GUESS}/.env.prod.generated" ]]; then
  export GENERATED_ENV_FILE="${REPO_ROOT_GUESS}/.env.prod.generated"
fi
if [[ -z "${GENERATED_SECRET_ENV_FILE+x}" && -f "${REPO_ROOT_GUESS}/.env.prod.generated.secrets" ]]; then
  export GENERATED_SECRET_ENV_FILE="${REPO_ROOT_GUESS}/.env.prod.generated.secrets"
fi

# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

usage() {
  cat <<'USAGE'
Usage: infra/ops/external-student-source-smoke.sh

Verifies the configured external student source without printing credentials,
raw student IDs, or raw student names.

Environment:
  EXTERNAL_STUDENT_SOURCE_SMOKE_MODE defaults to container.
    container: run /app/external-student-source-smoke through docker compose app.
    host: run go run ./cmd/external-student-source-smoke from server/.
  EXTERNAL_STUDENT_SOURCE_SMOKE_EVIDENCE_FILE defaults to
    infra/generated/external-student-source-smoke.json.
  EXTERNAL_STUDENT_SOURCE_SMOKE_REQUIRE_SAMPLE defaults to false.
    When true, configure EXTERNAL_STUDENT_SOURCE_SMOKE_STUDENT_ID and optionally
    EXTERNAL_STUDENT_SOURCE_SMOKE_EXPECTED_NAME in a local secret env file.
USAGE
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
  exit 0
fi

require_cmd python3
load_env

mode="${EXTERNAL_STUDENT_SOURCE_SMOKE_MODE:-container}"
entrypoint="${EXTERNAL_STUDENT_SOURCE_SMOKE_COMMAND:-/app/external-student-source-smoke}"
evidence_file="${EXTERNAL_STUDENT_SOURCE_SMOKE_EVIDENCE_FILE:-${REPO_ROOT}/infra/generated/external-student-source-smoke.json}"
mkdir -p "$(dirname "${evidence_file}")"

redact_file() {
  python3 - "$1" <<'PY'
import os
import sys
from pathlib import Path

path = Path(sys.argv[1])
text = path.read_text(encoding="utf-8", errors="replace")
for key in (
    "EXTERNAL_STUDENT_SOURCE_ORACLE_PASSWORD",
    "EXTERNAL_STUDENT_SOURCE_SMOKE_STUDENT_ID",
    "EXTERNAL_STUDENT_SOURCE_SMOKE_EXPECTED_NAME",
):
    value = os.environ.get(key, "").strip()
    if value:
        text = text.replace(value, "<redacted>")
print(text, end="" if text.endswith("\n") else "\n")
PY
}

run_host_smoke() {
  require_cmd go
  if [[ "${EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_MODE:-verify-full}" == "verify-full" ]]; then
    [[ -n "${EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_CA_HOST_PATH:-}" ]] ||
      die "EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_CA_HOST_PATH is required for host-mode verified Oracle TLS"
    export EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_CA_FILE="${EXTERNAL_STUDENT_SOURCE_ORACLE_TLS_CA_HOST_PATH}"
  fi
  (
    cd "${REPO_ROOT}/server"
    go run ./cmd/external-student-source-smoke
  )
}

run_container_smoke() {
  require_cmd docker
  export COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-${STACK_NAME:-stuhelper}}"
  compose --profile prod run --rm --no-deps -T \
    --entrypoint "${entrypoint}" \
    app
}

tmp_output="$(mktemp)"
tmp_error="$(mktemp)"
trap 'rm -f "${tmp_output}" "${tmp_error}"' EXIT

case "${mode}" in
  container)
    if ! run_container_smoke >"${tmp_output}" 2>"${tmp_error}"; then
      redact_file "${tmp_error}" >&2
      redact_file "${tmp_output}" >&2
      die "external student source smoke failed in container mode"
    fi
    ;;
  host)
    if ! run_host_smoke >"${tmp_output}" 2>"${tmp_error}"; then
      redact_file "${tmp_error}" >&2
      redact_file "${tmp_output}" >&2
      die "external student source smoke failed in host mode"
    fi
    ;;
  *)
    die "EXTERNAL_STUDENT_SOURCE_SMOKE_MODE must be container or host"
    ;;
esac

python3 - "${tmp_output}" "${evidence_file}" <<'PY'
import json
import os
import sys
from pathlib import Path

source = Path(sys.argv[1])
target = Path(sys.argv[2])

raw = source.read_text(encoding="utf-8")
try:
    evidence = json.loads(raw)
except json.JSONDecodeError as exc:
    raise SystemExit(f"external student source smoke did not return JSON: {exc}") from exc

required = {
    "provider": "oracle",
    "schoolCode": "4111010006",
}
for key, expected in required.items():
    if evidence.get(key) != expected:
        raise SystemExit(f"unexpected {key}: {evidence.get(key)!r}")

if evidence.get("readableRecordPresent") is not True:
    raise SystemExit("external student source has no readable student record")

oracle = evidence.get("oracle") or {}
if oracle.get("tlsMode") != "verify-full" or oracle.get("tlsVerified") is not True:
    raise SystemExit("external Oracle student source did not use verified TLS")

expected_oracle_tuning = {
    "maxOpenConns": "EXTERNAL_STUDENT_SOURCE_ORACLE_MAX_OPEN_CONNS",
    "maxIdleConns": "EXTERNAL_STUDENT_SOURCE_ORACLE_MAX_IDLE_CONNS",
    "connMaxLifetimeSeconds": "EXTERNAL_STUDENT_SOURCE_ORACLE_CONN_MAX_LIFETIME_SECONDS",
    "connMaxIdleTimeSeconds": "EXTERNAL_STUDENT_SOURCE_ORACLE_CONN_MAX_IDLE_TIME_SECONDS",
    "breakerFailureThreshold": "EXTERNAL_STUDENT_SOURCE_ORACLE_BREAKER_FAILURE_THRESHOLD",
    "breakerSuccessThreshold": "EXTERNAL_STUDENT_SOURCE_ORACLE_BREAKER_SUCCESS_THRESHOLD",
    "breakerOpenSeconds": "EXTERNAL_STUDENT_SOURCE_ORACLE_BREAKER_OPEN_SECONDS",
}
for evidence_key, env_key in expected_oracle_tuning.items():
    expected = os.environ.get(env_key, "").strip()
    if not expected or oracle.get(evidence_key) != int(expected):
        raise SystemExit(f"external Oracle evidence does not match {env_key}")

sample = evidence.get("sample") or {}
require_sample = os.environ.get("EXTERNAL_STUDENT_SOURCE_SMOKE_REQUIRE_SAMPLE", "").strip().lower()
if require_sample in {"1", "true", "yes", "on"}:
    if sample.get("enabled") is not True or sample.get("found") is not True:
        raise SystemExit("required external student source sample was not found")
    if sample.get("expectedNameProvided") is True and sample.get("nameMatched") is not True:
        raise SystemExit("required external student source sample name did not match")

sensitive_values = [
    os.environ.get("EXTERNAL_STUDENT_SOURCE_ORACLE_PASSWORD", ""),
    os.environ.get("EXTERNAL_STUDENT_SOURCE_SMOKE_STUDENT_ID", ""),
    os.environ.get("EXTERNAL_STUDENT_SOURCE_SMOKE_EXPECTED_NAME", ""),
]
for value in sensitive_values:
    value = value.strip()
    if value and value in raw:
        raise SystemExit("external student source smoke evidence contains a sensitive value")

target.write_text(json.dumps(evidence, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
PY

log "external student source smoke passed; evidence=${evidence_file}"
