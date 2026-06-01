#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
EVIDENCE_SCRIPT="${REPO_ROOT}/infra/ops/koishi-admission-production-evidence.sh"
PROD_GO_LIVE="${REPO_ROOT}/docs/guides/production-go-live.md"
RELEASE_RUNBOOK="${REPO_ROOT}/docs/guides/release-runbook.md"

fail() {
  echo "[koishi-admission-production-evidence-contract][error] $*" >&2
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
    fail "expected ${file} not to contain pattern: ${pattern}"
  fi
}

[[ -f "${EVIDENCE_SCRIPT}" ]] || fail "missing evidence script: ${EVIDENCE_SCRIPT}"
[[ -f "${PROD_GO_LIVE}" ]] || fail "missing production go-live guide: ${PROD_GO_LIVE}"
[[ -f "${RELEASE_RUNBOOK}" ]] || fail "missing release runbook: ${RELEASE_RUNBOOK}"

bash -n "${EVIDENCE_SCRIPT}"

assert_contains "${EVIDENCE_SCRIPT}" 'KOISHI_COMPOSE_DIR'
assert_contains "${EVIDENCE_SCRIPT}" 'KOISHI_CONFIG_FILE'
assert_contains "${EVIDENCE_SCRIPT}" 'KOISHI_CONTAINER_NAME'
assert_contains "${EVIDENCE_SCRIPT}" 'KOISHI_ADMISSION_EXPECTED_GROUP_IDS'
assert_contains "${EVIDENCE_SCRIPT}" 'KOISHI_ADMISSION_BOT_SELF_ID'
assert_contains "${EVIDENCE_SCRIPT}" 'STUHELPER_PLATFORM_BASE_URL'
assert_contains "${EVIDENCE_SCRIPT}" 'STUHELPER_PLATFORM_SERVICE_TOKEN'
assert_contains "${EVIDENCE_SCRIPT}" 'STUHELPER_FRESHMAN_MATERIAL_HOSTS'
assert_contains "${EVIDENCE_SCRIPT}" '/api/v1/bot/admission/sessions/pending'
assert_contains "${EVIDENCE_SCRIPT}" 'platform=qq'
assert_contains "${EVIDENCE_SCRIPT}" 'duplicate command names: 举报'
assert_contains "${EVIDENCE_SCRIPT}" 'pending-forward'
assert_contains "${EVIDENCE_SCRIPT}" 'B0000001'
assert_contains "${EVIDENCE_SCRIPT}" 'enableGroupVerify'
assert_contains "${EVIDENCE_SCRIPT}" 'commands'
assert_contains "${EVIDENCE_SCRIPT}" 'moderation'
assert_contains "${EVIDENCE_SCRIPT}" 'freshmanForward'
assert_contains "${EVIDENCE_SCRIPT}" 'Koishi admission config semantics'

assert_not_contains "${EVIDENCE_SCRIPT}" 'root@'
assert_not_contains "${EVIDENCE_SCRIPT}" '65022|2222'
assert_not_contains "${EVIDENCE_SCRIPT}" 'sshpass'
assert_not_contains "${EVIDENCE_SCRIPT}" 'EV~|20050626|STUHELPER_PLATFORM_SERVICE_TOKEN='

tmpdir="$(mktemp -d)"
cleanup() {
  rm -rf "${tmpdir}"
}
trap cleanup EXIT

compose_dir="${tmpdir}/koishi-napcat"
mkdir -p "${compose_dir}/koishi" "${tmpdir}/bin"
cat >"${compose_dir}/koishi/koishi.yml" <<'YAML'
plugins:
  group:stuhelper:
    stuhelper-group-guard:admission:
      platform:
        baseUrl: ${{ env.STUHELPER_PLATFORM_BASE_URL }}
        serviceToken: ${{ env.STUHELPER_PLATFORM_SERVICE_TOKEN }}
      guard:
        targetGroups:
          - '178037297'
      commands:
        enabled: false
      admissionCommands:
        enabled: true
        minAuthority: 4
      moderation:
        enabled: false
      freshmanForward:
        enabled: false
    student-query:uciuxr:
      enableGroupVerify: true
YAML

cat >"${tmpdir}/bin/docker" <<'SH'
#!/usr/bin/env bash
set -euo pipefail

case "${1:-}" in
  inspect)
    [[ "${2:-}" == "-f" ]] || exit 2
    [[ "${4:-}" == "koishi" ]] || exit 3
    echo "true"
    ;;
  exec)
    shift
    interactive=false
    if [[ "${1:-}" == "-i" ]]; then
      interactive=true
      shift
    fi
    container="${1:-}"
    shift
    [[ "${container}" == "koishi" ]] || exit 4
    if [[ "${interactive}" == "true" ]]; then
      cat >/dev/null
      if [[ "${FAKE_DOCKER_MODE:-ok}" == "bad_probe" ]]; then
        echo '{"status":500,"bodyBytes":70}'
        exit 1
      fi
      echo '{"status":200,"bodyBytes":26}'
      exit 0
    fi
    if [[ "${1:-}" == "node" && "${2:-}" == "-e" ]]; then
      if [[ "${FAKE_DOCKER_MODE:-ok}" == "missing_env" ]]; then
        echo '{"baseURL":"https://stuhelper.com","hasServiceToken":false,"hasFreshmanHosts":false,"freshmanHosts":""}'
        exit 0
      fi
      echo '{"baseURL":"https://stuhelper.com","hasServiceToken":true,"hasFreshmanHosts":true,"freshmanHosts":"stuhelper.com,join.stuhelper.com"}'
      exit 0
    fi
    exit 5
    ;;
  logs)
    case "${FAKE_DOCKER_MODE:-ok}" in
      bad_log)
        echo '2026-05-30 09:05:26 [E] stuhelper:group-guard group guard scheduled scan failed PlatformAPIError: failed to verify bot service credential'
        exit 0
        ;;
      bad_b0000001)
        echo "2026-05-30 09:05:26 [E] stuhelper:group-guard PlatformAPIError { status: 500, code: 'B0000001' }"
        exit 0
        ;;
      bad_pending_forward)
        echo '2026-05-30 09:05:26 [E] stuhelper:group-guard pending-forward returned 500'
        exit 0
        ;;
      bad_duplicate_command)
        echo '2026-05-30 09:05:26 [E] app duplicate command names: 举报'
        exit 0
        ;;
    esac
    echo '2026-05-30 19:26:15 [I] loader apply plugin stuhelper-group-guard:admission'
    echo '2026-05-30 19:26:17 [I] stuhelper:group-guard 群管插件已加载，目标群数量：1，扫描间隔：60 秒'
    echo '2026-05-30 19:26:19 [E] chatluna Error: INVALID_API_KEY'
    ;;
  *)
    exit 9
    ;;
esac
SH
chmod +x "${tmpdir}/bin/docker"

evidence_file="${tmpdir}/evidence.json"
PATH="${tmpdir}/bin:${PATH}" \
KOISHI_COMPOSE_DIR="${compose_dir}" \
KOISHI_CONTAINER_NAME="koishi" \
KOISHI_ADMISSION_EVIDENCE_FILE="${evidence_file}" \
KOISHI_ADMISSION_LOG_SINCE="2h" \
"${EVIDENCE_SCRIPT}" >"${tmpdir}/ok.stdout" 2>"${tmpdir}/ok.stderr"

grep -q 'Koishi admission production evidence passed' "${tmpdir}/ok.stdout" || fail "evidence script did not report pass"
[[ -f "${evidence_file}" ]] || fail "expected evidence file to be written"
jq -e '
  .passed == true
  and .summary.failed == 0
  and .targets.expectedGroupIDs == ["178037297"]
  and ([.checks[] | select(.name == "Koishi admission config semantics" and .passed == true)] | length == 1)
  and ([.checks[] | select(.name == "Koishi bot admission API probe" and .passed == true)] | length == 1)
  and ([.checks[] | select(.name == "Koishi admission logs clean" and .passed == true)] | length == 1)
  and ([.checks[] | select(.name == "Koishi non-admission ChatLuna signal" and .detail == "chatluna_error_count=1")] | length == 1)
' "${evidence_file}" >/dev/null

expect_log_failure() {
  local mode="$1"
  local label="$2"
  if PATH="${tmpdir}/bin:${PATH}" \
    KOISHI_COMPOSE_DIR="${compose_dir}" \
    KOISHI_CONTAINER_NAME="koishi" \
    KOISHI_ADMISSION_EVIDENCE_FILE="${tmpdir}/${label}-evidence.json" \
    FAKE_DOCKER_MODE="${mode}" \
    "${EVIDENCE_SCRIPT}" >"${tmpdir}/${label}.stdout" 2>"${tmpdir}/${label}.stderr"; then
    fail "expected evidence script to fail for ${label}"
  fi
  grep -q 'Koishi admission production evidence failed' "${tmpdir}/${label}.stderr" || fail "missing ${label} failure message"
}

expect_config_failure() {
  local mutation="$1"
  local expected_detail="$2"
  local label="bad-config-${mutation}"
  local bad_compose_dir="${tmpdir}/${label}/koishi-napcat"

  mkdir -p "${bad_compose_dir}"
  cp -R "${compose_dir}/koishi" "${bad_compose_dir}/koishi"
  python3 - "${bad_compose_dir}/koishi/koishi.yml" "${mutation}" <<'PY'
import sys
from pathlib import Path

path = Path(sys.argv[1])
mutation = sys.argv[2]
text = path.read_text(encoding="utf-8")

if mutation == "wrong_target_group":
    text = text.replace("- '178037297'", "- '699979027'")
elif mutation == "commands_enabled":
    text = text.replace("      commands:\n        enabled: false", "      commands:\n        enabled: true")
elif mutation == "moderation_enabled":
    text = text.replace("      moderation:\n        enabled: false", "      moderation:\n        enabled: true")
elif mutation == "freshman_forward_enabled":
    text = text.replace("      freshmanForward:\n        enabled: false", "      freshmanForward:\n        enabled: true")
elif mutation == "student_query_disabled":
    text = text.replace("      enableGroupVerify: true", "      enableGroupVerify: false")
else:
    raise SystemExit(f"unknown mutation: {mutation}")

path.write_text(text, encoding="utf-8")
PY

  if PATH="${tmpdir}/bin:${PATH}" \
    KOISHI_COMPOSE_DIR="${bad_compose_dir}" \
    KOISHI_CONTAINER_NAME="koishi" \
    KOISHI_ADMISSION_EVIDENCE_FILE="${tmpdir}/${label}-evidence.json" \
    "${EVIDENCE_SCRIPT}" >"${tmpdir}/${label}.stdout" 2>"${tmpdir}/${label}.stderr"; then
    fail "expected evidence script to fail for ${mutation}"
  fi
  jq -e --arg expected "${expected_detail}" '
    [.checks[] | select(.name == "Koishi admission config semantics" and .passed == false and (.detail | contains($expected)))] | length == 1
  ' "${tmpdir}/${label}-evidence.json" >/dev/null || fail "missing semantic config failure detail for ${mutation}"
}

expect_log_failure "bad_log" "bad-log"
expect_log_failure "bad_b0000001" "bad-b0000001"
expect_log_failure "bad_pending_forward" "bad-pending-forward"
expect_log_failure "bad_duplicate_command" "bad-duplicate-command"

expect_config_failure "wrong_target_group" "do not match expected"
expect_config_failure "commands_enabled" "commands.enabled is not false"
expect_config_failure "moderation_enabled" "moderation.enabled is not false"
expect_config_failure "freshman_forward_enabled" "freshmanForward.enabled is not false"
expect_config_failure "student_query_disabled" "student-query.enableGroupVerify is not true"

assert_contains "${PROD_GO_LIVE}" 'koishi-admission-production-evidence\.sh'
assert_contains "${RELEASE_RUNBOOK}" 'koishi-admission-production-evidence\.sh'

echo "[koishi-admission-production-evidence-contract] all assertions passed"
