#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

usage() {
  cat <<'USAGE'
Usage: infra/ops/koishi-admission-production-evidence.sh

Collects redacted, repeatable production evidence for the Koishi admission MVP.
Run this on the Koishi/NapCat production node, not on the main StuHelper host,
unless the Koishi Compose directory is mounted locally.

Environment:
  KOISHI_COMPOSE_DIR                         defaults to current working directory
  KOISHI_CONFIG_FILE                         defaults to $KOISHI_COMPOSE_DIR/koishi/koishi.yml
  KOISHI_CONTAINER_NAME                      defaults to koishi
  KOISHI_ADMISSION_EXPECTED_GROUP_IDS        defaults to 178037297
  KOISHI_ADMISSION_BOT_SELF_ID               defaults to 2118785781
  KOISHI_ADMISSION_LOG_SINCE                 defaults to 2h
  KOISHI_ADMISSION_LOG_TAIL                  defaults to 2000
  KOISHI_ADMISSION_REQUIRE_LOAD_LOG          defaults to false
  KOISHI_ADMISSION_EVIDENCE_FILE             defaults to infra/generated/koishi-admission-production-evidence.json

The script never prints STUHELPER_PLATFORM_SERVICE_TOKEN. It only records
whether the token is present and whether the bot API probe returns HTTP 200.
USAGE
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
  exit 0
fi

log() {
  echo "[stuhelper] $*"
}

warn() {
  echo "[stuhelper][warn] $*" >&2
}

die() {
  echo "[stuhelper][error] $*" >&2
  exit 1
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "missing required command: $1"
}

trim_trailing_slash() {
  local value="${1:-}"
  value="${value%/}"
  printf '%s\n' "${value}"
}

require_cmd docker
require_cmd python3

compose_dir="${KOISHI_COMPOSE_DIR:-$(pwd)}"
config_file="${KOISHI_CONFIG_FILE:-${compose_dir}/koishi/koishi.yml}"
container_name="${KOISHI_CONTAINER_NAME:-koishi}"
expected_group_ids="${KOISHI_ADMISSION_EXPECTED_GROUP_IDS:-178037297}"
bot_self_id="${KOISHI_ADMISSION_BOT_SELF_ID:-2118785781}"
log_since="${KOISHI_ADMISSION_LOG_SINCE:-2h}"
log_tail="${KOISHI_ADMISSION_LOG_TAIL:-2000}"
require_load_log="${KOISHI_ADMISSION_REQUIRE_LOAD_LOG:-false}"
evidence_file="${KOISHI_ADMISSION_EVIDENCE_FILE:-${REPO_ROOT}/infra/generated/koishi-admission-production-evidence.json}"

[[ -d "${compose_dir}" ]] || die "KOISHI_COMPOSE_DIR does not exist: ${compose_dir}"
[[ -f "${config_file}" ]] || die "KOISHI_CONFIG_FILE does not exist: ${config_file}"
[[ -n "${container_name}" ]] || die "KOISHI_CONTAINER_NAME must not be empty"
[[ -n "${expected_group_ids//,/}" ]] || die "KOISHI_ADMISSION_EXPECTED_GROUP_IDS must not be empty"
[[ -n "${bot_self_id}" ]] || die "KOISHI_ADMISSION_BOT_SELF_ID must not be empty"

evidence_lines="$(mktemp)"
trap 'rm -f "${evidence_lines}"' EXIT
PASS=0
FAIL=0

record_check() {
  local name="$1"
  local kind="$2"
  local passed="$3"
  local detail="${4:-}"

  if [[ "${passed}" == "true" ]]; then
    PASS=$((PASS + 1))
  else
    FAIL=$((FAIL + 1))
  fi

  python3 - "${evidence_lines}" "${name}" "${kind}" "${passed}" "${detail}" <<'PY'
import json
import sys

path, name, kind, passed, detail = sys.argv[1:6]
with open(path, "a", encoding="utf-8") as handle:
    handle.write(json.dumps({
        "name": name,
        "kind": kind,
        "passed": passed == "true",
        "detail": detail,
    }, ensure_ascii=True, separators=(",", ":")) + "\n")
PY
}

config_contains() {
  local name="$1"
  local pattern="$2"
  if grep -Eq -- "${pattern}" "${config_file}"; then
    record_check "${name}" "koishi_config" "true" "${pattern}"
  else
    record_check "${name}" "koishi_config" "false" "missing pattern: ${pattern}"
  fi
}

config_section_has_enabled() {
  local section="$1"
  local expected="$2"

  if python3 - "${config_file}" "${section}" "${expected}" <<'PY'
import re
import sys
from pathlib import Path

path = Path(sys.argv[1])
section = sys.argv[2]
expected = sys.argv[3]
lines = path.read_text(encoding="utf-8").splitlines()

for index, line in enumerate(lines):
    if not re.match(rf"^\s*{re.escape(section)}:\s*(?:#.*)?$", line):
        continue
    section_indent = len(line) - len(line.lstrip())
    for nested in lines[index + 1:index + 12]:
        stripped = nested.strip()
        if not stripped or stripped.startswith("#"):
            continue
        nested_indent = len(nested) - len(nested.lstrip())
        if nested_indent <= section_indent:
            break
        if re.match(rf"^\s*enabled:\s*{re.escape(expected)}\s*(?:#.*)?$", nested):
            raise SystemExit(0)
raise SystemExit(1)
PY
  then
    record_check "${section}.enabled=${expected}" "koishi_config" "true" "${section}.enabled=${expected}"
  else
    record_check "${section}.enabled=${expected}" "koishi_config" "false" "missing ${section}.enabled=${expected}"
  fi
}

check_koishi_config_semantics() {
  local output

  if output="$(python3 - "${config_file}" "${expected_group_ids}" <<'PY'
import json
import re
import sys
from pathlib import Path

path = Path(sys.argv[1])
expected_groups = [item.strip() for item in sys.argv[2].split(",") if item.strip()]
lines = path.read_text(encoding="utf-8").splitlines()
errors = []


def indent_of(line):
    return len(line) - len(line.lstrip())


def collect_blocks(pattern):
    blocks = []
    for index, line in enumerate(lines):
        if not pattern.match(line):
            continue
        section_indent = indent_of(line)
        end = len(lines)
        for nested_index in range(index + 1, len(lines)):
            nested = lines[nested_index]
            stripped = nested.strip()
            if not stripped or stripped.startswith("#"):
                continue
            if indent_of(nested) <= section_indent:
                end = nested_index
                break
        blocks.append(lines[index:end])
    return blocks


def find_section(block, section):
    section_pattern = re.compile(rf"^\s*{re.escape(section)}:\s*(?:#.*)?$")
    for index, line in enumerate(block):
        if not section_pattern.match(line):
            continue
        section_indent = indent_of(line)
        end = len(block)
        for nested_index in range(index + 1, len(block)):
            nested = block[nested_index]
            stripped = nested.strip()
            if not stripped or stripped.startswith("#"):
                continue
            if indent_of(nested) <= section_indent:
                end = nested_index
                break
        return block[index:end]
    return []


def section_enabled(block, section, expected):
    section_block = find_section(block, section)
    if not section_block:
        errors.append(f"missing {section} section in stuhelper-group-guard admission block")
        return
    text = "\n".join(section_block)
    if not re.search(rf"^\s*enabled:\s*{re.escape(expected)}\s*(?:#.*)?$", text, re.MULTILINE):
        errors.append(f"{section}.enabled is not {expected} in stuhelper-group-guard admission block")


def parse_group_values(value):
    return [
        item.strip().strip("'\"")
        for item in value.split(",")
        if item.strip().strip("'\"")
    ]


def extract_target_groups(block):
    section = find_section(block, "targetGroups")
    if not section:
        errors.append("missing targetGroups in stuhelper-group-guard admission block")
        return []

    first_line = section[0]
    inline = re.search(r"targetGroups:\s*\[(.*)\]\s*(?:#.*)?$", first_line)
    if inline:
        return parse_group_values(inline.group(1))

    values = []
    for line in section[1:]:
        stripped = line.strip()
        if not stripped or stripped.startswith("#"):
            continue
        match = re.match(r"^-\s*['\"]?([^'\"\s,#\]]+)['\"]?\s*(?:#.*)?$", stripped)
        if match:
            values.append(match.group(1))
    return values


guard_pattern = re.compile(r"^\s*stuhelper-group-guard(?::[A-Za-z0-9_-]+)?:\s*(?:#.*)?$")
student_query_pattern = re.compile(r"^\s*student-query(?::[A-Za-z0-9_-]+)?:\s*(?:#.*)?$")

guard_blocks = collect_blocks(guard_pattern)
if not guard_blocks:
    errors.append("missing stuhelper-group-guard admission block")
    guard_block = []
else:
    guard_block = guard_blocks[0]

actual_groups = []
if guard_block:
    guard_text = "\n".join(guard_block)
    if "STUHELPER_PLATFORM_BASE_URL" not in guard_text:
        errors.append("admission platform.baseUrl does not use STUHELPER_PLATFORM_BASE_URL")
    if "STUHELPER_PLATFORM_SERVICE_TOKEN" not in guard_text:
        errors.append("admission platform.serviceToken does not use STUHELPER_PLATFORM_SERVICE_TOKEN")

    actual_groups = extract_target_groups(guard_block)
    if actual_groups != sorted(set(actual_groups), key=actual_groups.index):
        errors.append(f"admission targetGroups contains duplicate values: {actual_groups}")
    if sorted(actual_groups) != sorted(expected_groups):
        errors.append(f"admission targetGroups {actual_groups} do not match expected {expected_groups}")

    section_enabled(guard_block, "commands", "false")
    section_enabled(guard_block, "admissionCommands", "true")
    section_enabled(guard_block, "moderation", "false")
    section_enabled(guard_block, "freshmanForward", "false")

student_blocks = collect_blocks(student_query_pattern)
if not student_blocks:
    errors.append("missing student-query plugin block")
elif not any(
    re.search(r"^\s*enableGroupVerify:\s*true\s*(?:#.*)?$", "\n".join(block), re.MULTILINE)
    for block in student_blocks
):
    errors.append("student-query.enableGroupVerify is not true")

print(json.dumps({
    "expectedGroupIDs": expected_groups,
    "actualTargetGroups": actual_groups,
    "errors": errors,
}, ensure_ascii=True, separators=(",", ":")))
raise SystemExit(1 if errors else 0)
PY
  )"; then
    record_check "Koishi admission config semantics" "koishi_config" "true" "${output}"
  else
    record_check "Koishi admission config semantics" "koishi_config" "false" "${output:-semantic config check failed}"
  fi
}

check_container_running() {
  local running
  if running="$(docker inspect -f '{{.State.Running}}' "${container_name}" 2>/dev/null)" && [[ "${running}" == "true" ]]; then
    record_check "Koishi container running" "docker" "true" "${container_name}"
  else
    record_check "Koishi container running" "docker" "false" "${container_name}"
  fi
}

check_container_env() {
  local output
  if output="$(docker exec "${container_name}" node -e '
const payload = {
  baseURL: process.env.STUHELPER_PLATFORM_BASE_URL || "",
  hasServiceToken: Boolean(process.env.STUHELPER_PLATFORM_SERVICE_TOKEN),
  hasFreshmanHosts: Boolean(process.env.STUHELPER_FRESHMAN_MATERIAL_HOSTS),
  freshmanHosts: process.env.STUHELPER_FRESHMAN_MATERIAL_HOSTS || "",
}
console.log(JSON.stringify(payload))
')" && python3 - "${output}" <<'PY'
import json
import sys

payload = json.loads(sys.argv[1])
if payload.get("baseURL") != "https://stuhelper.com":
    raise SystemExit(1)
if not payload.get("hasServiceToken"):
    raise SystemExit(1)
hosts = {item.strip() for item in payload.get("freshmanHosts", "").split(",") if item.strip()}
if not {"stuhelper.com", "join.stuhelper.com"}.issubset(hosts):
    raise SystemExit(1)
PY
  then
    record_check "Koishi container StuHelper env" "docker_env" "true" "${output}"
  else
    record_check "Koishi container StuHelper env" "docker_env" "false" "${output:-docker exec failed}"
  fi
}

check_bot_api_probe() {
  local output
  if output="$(docker exec -i "${container_name}" node - "${bot_self_id}" <<'NODE'
const botSelfID = process.argv[2] || '2118785781'
const base = (process.env.STUHELPER_PLATFORM_BASE_URL || '').replace(/\/+$/, '')
const token = process.env.STUHELPER_PLATFORM_SERVICE_TOKEN || ''

if (!base || !token) {
  console.log(JSON.stringify({ error: 'missing_env', hasBase: Boolean(base), hasToken: Boolean(token) }))
  process.exit(1)
}

const headers = new Headers()
headers.set('Authorization', `Bearer ${token}`)
const url = `${base}/api/v1/bot/admission/sessions/pending?platform=qq&botSelfID=${encodeURIComponent(botSelfID)}&limit=5`

fetch(url, { headers })
  .then(async (response) => {
    const body = await response.text()
    console.log(JSON.stringify({ status: response.status, bodyBytes: body.length }))
    process.exit(response.ok ? 0 : 1)
  })
  .catch((error) => {
    console.log(JSON.stringify({ error: error && error.message ? error.message : String(error) }))
    process.exit(1)
  })
NODE
)" && python3 - "${output}" <<'PY'
import json
import sys

payload = json.loads(sys.argv[1])
if payload.get("status") != 200:
    raise SystemExit(1)
if int(payload.get("bodyBytes", 0)) <= 0:
    raise SystemExit(1)
PY
  then
    record_check "Koishi bot admission API probe" "bot_api" "true" "${output}"
  else
    record_check "Koishi bot admission API probe" "bot_api" "false" "${output:-docker exec failed}"
  fi
}

check_logs() {
  local logs admission_error_pattern load_pattern chatluna_count
  logs="$(docker logs --since "${log_since}" --tail "${log_tail}" "${container_name}" 2>&1 || true)"

  admission_error_pattern='group guard scheduled scan failed|pending-forward|B0000001|failed to verify bot service credential|admission 401|unauthorized|duplicate command names: 举报'
  if grep -Eq -- "${admission_error_pattern}" <<<"${logs}"; then
    record_check "Koishi admission logs clean" "docker_logs" "false" "matched admission error pattern"
  else
    record_check "Koishi admission logs clean" "docker_logs" "true" "no admission error pattern matched since ${log_since}"
  fi

  load_pattern='apply plugin stuhelper-group-guard|stuhelper:group-guard.*目标群数量：[0-9]+|目标群数量：[0-9]+'
  if grep -Eq -- "${load_pattern}" <<<"${logs}"; then
    record_check "Koishi group guard loaded" "docker_logs" "true" "load log found since ${log_since}"
  elif [[ "${require_load_log}" == "true" ]]; then
    record_check "Koishi group guard loaded" "docker_logs" "false" "load log not found since ${log_since}"
  else
    record_check "Koishi group guard loaded" "docker_logs" "true" "load log not required"
  fi

  chatluna_count="$(grep -Ec 'ChatLunaError|INVALID_API_KEY' <<<"${logs}" || true)"
  record_check "Koishi non-admission ChatLuna signal" "docker_logs_info" "true" "chatluna_error_count=${chatluna_count}"
}

write_evidence() {
  local tmp_file
  tmp_file="$(mktemp)"
  GENERATED_AT="$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  KOISHI_COMPOSE_DIR_VALUE="${compose_dir}" \
  KOISHI_CONFIG_FILE_VALUE="${config_file}" \
  KOISHI_CONTAINER_VALUE="${container_name}" \
  KOISHI_EXPECTED_GROUP_IDS_VALUE="${expected_group_ids}" \
  KOISHI_BOT_SELF_ID_VALUE="${bot_self_id}" \
  KOISHI_LOG_SINCE_VALUE="${log_since}" \
  KOISHI_PASS="${PASS}" \
  KOISHI_FAIL="${FAIL}" \
  python3 - "${evidence_lines}" <<'PY' >"${tmp_file}"
import json
import os
import sys
from pathlib import Path

checks = [
    json.loads(line)
    for line in Path(sys.argv[1]).read_text(encoding="utf-8").splitlines()
    if line.strip()
]
bundle = {
    "generatedAt": os.environ["GENERATED_AT"],
    "passed": int(os.environ["KOISHI_FAIL"]) == 0,
    "summary": {
        "passed": int(os.environ["KOISHI_PASS"]),
        "failed": int(os.environ["KOISHI_FAIL"]),
    },
    "targets": {
        "composeDir": os.environ["KOISHI_COMPOSE_DIR_VALUE"],
        "configFile": os.environ["KOISHI_CONFIG_FILE_VALUE"],
        "container": os.environ["KOISHI_CONTAINER_VALUE"],
        "expectedGroupIDs": [
            item.strip()
            for item in os.environ["KOISHI_EXPECTED_GROUP_IDS_VALUE"].split(",")
            if item.strip()
        ],
        "botSelfID": os.environ["KOISHI_BOT_SELF_ID_VALUE"],
        "logSince": os.environ["KOISHI_LOG_SINCE_VALUE"],
    },
    "checks": checks,
}
print(json.dumps(bundle, ensure_ascii=True, indent=2))
PY

  if [[ "${evidence_file}" != "-" ]]; then
    mkdir -p "$(dirname "${evidence_file}")"
    install -m 600 "${tmp_file}" "${evidence_file}"
    log "wrote Koishi admission production evidence to ${evidence_file}" >&2
  fi
  cat "${tmp_file}"
  rm -f "${tmp_file}"
}

config_contains "stuhelper-group-guard configured" 'stuhelper-group-guard:[[:alnum:]_-]+:'
config_contains "platform baseUrl uses env" 'STUHELPER_PLATFORM_BASE_URL'
config_contains "platform serviceToken uses env" 'STUHELPER_PLATFORM_SERVICE_TOKEN'
config_contains "guard targetGroups configured" 'targetGroups:'

IFS=',' read -r -a expected_groups <<<"${expected_group_ids}"
for group_id in "${expected_groups[@]}"; do
  group_id="$(trim_trailing_slash "${group_id}")"
  group_id="${group_id//[[:space:]]/}"
  [[ -n "${group_id}" ]] || continue
  config_contains "target group ${group_id}" "'?${group_id}'?"
done

config_section_has_enabled "commands" "false"
config_section_has_enabled "admissionCommands" "true"
config_section_has_enabled "moderation" "false"
config_section_has_enabled "freshmanForward" "false"
config_contains "student-query plugin remains configured" 'student-query:'
config_contains "student-query group verify remains enabled" 'enableGroupVerify:[[:space:]]*true'
check_koishi_config_semantics

check_container_running
check_container_env
check_bot_api_probe
check_logs

write_evidence >/dev/null

if (( FAIL > 0 )); then
  die "Koishi admission production evidence failed: ${FAIL} failed checks"
fi

log "Koishi admission production evidence passed: checks=${PASS}"
