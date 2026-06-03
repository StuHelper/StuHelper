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
Usage: infra/ops/admission-join-e2e-evidence.sh

Collects read-only evidence for a real QQ small-account admission E2E.
This script does not create or mutate production admission data; run it after a
real QQ account joins an admission target group and, for full closure, after the
user completes the join.stuhelper.com verification flow.

Required:
  ADMISSION_E2E_QQ_ID                 QQ number used by the real small-account E2E.

Optional:
  ADMISSION_E2E_GUILD_ID              defaults to 178037297
  ADMISSION_E2E_PLATFORM              defaults to qq
  ADMISSION_E2E_BOT_SELF_ID           defaults to 2118785781
  ADMISSION_E2E_EXPECTED_STAGE        bot-released | flow-completed | join-created, defaults to flow-completed
  ADMISSION_E2E_LOOKBACK_HOURS        defaults to 24
  ADMISSION_E2E_MAX_SESSION_AGE_MINUTES defaults to 180
  ADMISSION_E2E_DATABASE_URL          defaults to DATABASE_URL
  ADMISSION_E2E_EVIDENCE_FILE         defaults to infra/generated/admission-join-e2e-evidence.json
  ADMISSION_E2E_PUBLIC_PREVIEW_PROBE_ENABLED defaults to true. When the latest
                                   session is joined_muted and unconsumed, the
                                   script verifies the raw auth URL through the
                                   public join API without printing the token.
  ADMISSION_E2E_CURL_NO_PROXY         defaults to "*"; set empty to honor proxy env vars
  ADMISSION_E2E_CURL_INSECURE         defaults to false; local/self-signed diagnostics only
  ADMISSION_E2E_PUBLIC_RESOLVE_IP     optional diagnostic --resolve override for join.stuhelper.com

Evidence is redacted: token_hash and raw token are never printed. auth_url is
validated by shape and then reduced to host/path/query-key booleans.
USAGE
}

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  usage
  exit 0
fi

require_cmd docker
require_cmd curl
require_cmd python3

load_env
export COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-${STACK_NAME:-stuhelper}}"

qq_id="${ADMISSION_E2E_QQ_ID:-}"
guild_id="${ADMISSION_E2E_GUILD_ID:-178037297}"
platform="${ADMISSION_E2E_PLATFORM:-qq}"
bot_self_id="${ADMISSION_E2E_BOT_SELF_ID:-2118785781}"
expected_stage="${ADMISSION_E2E_EXPECTED_STAGE:-flow-completed}"
lookback_hours="${ADMISSION_E2E_LOOKBACK_HOURS:-24}"
max_session_age_minutes="${ADMISSION_E2E_MAX_SESSION_AGE_MINUTES:-180}"
evidence_file="${ADMISSION_E2E_EVIDENCE_FILE:-${REPO_ROOT}/infra/generated/admission-join-e2e-evidence.json}"
database_url="${ADMISSION_E2E_DATABASE_URL:-${DATABASE_URL:-}}"
admission_public_base_url="${ADMISSION_PUBLIC_BASE_URL:-https://join.stuhelper.com}"
public_preview_probe_enabled="${ADMISSION_E2E_PUBLIC_PREVIEW_PROBE_ENABLED:-true}"
curl_no_proxy="${ADMISSION_E2E_CURL_NO_PROXY:-*}"
curl_insecure="${ADMISSION_E2E_CURL_INSECURE:-false}"
public_resolve_ip="${ADMISSION_E2E_PUBLIC_RESOLVE_IP:-}"

[[ -n "${qq_id}" ]] || die "ADMISSION_E2E_QQ_ID is required; run this only after a real QQ small account joins 178037297, for example: ADMISSION_E2E_QQ_ID=<small-account-qq> ADMISSION_E2E_EXPECTED_STAGE=join-created ./infra/ops/admission-join-e2e-evidence.sh"
[[ "${qq_id}" =~ ^[0-9]{5,20}$ ]] || die "ADMISSION_E2E_QQ_ID must be a QQ number"
[[ -n "${guild_id}" ]] || die "ADMISSION_E2E_GUILD_ID must not be empty"
[[ "${platform}" == "qq" ]] || die "ADMISSION_E2E_PLATFORM must be qq for the current admission MVP"
[[ -n "${bot_self_id}" ]] || die "ADMISSION_E2E_BOT_SELF_ID must not be empty"
case "${expected_stage}" in
  join-created|flow-completed|bot-released) ;;
  *) die "ADMISSION_E2E_EXPECTED_STAGE must be join-created, flow-completed, or bot-released" ;;
esac
case "${public_preview_probe_enabled}" in
  true|TRUE|1|yes|YES) public_preview_probe_enabled="true" ;;
  false|FALSE|0|no|NO|"") public_preview_probe_enabled="false" ;;
  *) die "ADMISSION_E2E_PUBLIC_PREVIEW_PROBE_ENABLED must be true or false" ;;
esac
case "${curl_insecure}" in
  true|TRUE|1|yes|YES) curl_insecure="true" ;;
  false|FALSE|0|no|NO|"") curl_insecure="false" ;;
  *) die "ADMISSION_E2E_CURL_INSECURE must be true or false" ;;
esac
[[ "${lookback_hours}" =~ ^[0-9]+$ && "${lookback_hours}" -gt 0 ]] || die "ADMISSION_E2E_LOOKBACK_HOURS must be a positive integer"
[[ "${max_session_age_minutes}" =~ ^[0-9]+$ && "${max_session_age_minutes}" -gt 0 ]] || die "ADMISSION_E2E_MAX_SESSION_AGE_MINUTES must be a positive integer"
[[ "${admission_public_base_url}" == "https://join.stuhelper.com" ]] || \
  die "ADMISSION_PUBLIC_BASE_URL must be exactly https://join.stuhelper.com for production admission E2E evidence"
[[ -n "${database_url}" ]] || die "ADMISSION_E2E_DATABASE_URL or DATABASE_URL is required"

materialize_database_url() {
  local value="$1"
  if [[ "${value}" == *"REPLACE_WITH_STUHELPER_APP_DB_PASSWORD"* ]]; then
    [[ -n "${STUHELPER_APP_DB_PASSWORD:-}" ]] || \
      die "DATABASE_URL contains REPLACE_WITH_STUHELPER_APP_DB_PASSWORD but STUHELPER_APP_DB_PASSWORD is not set"
    local encoded_password
    encoded_password="$(
      STUHELPER_APP_DB_PASSWORD="${STUHELPER_APP_DB_PASSWORD}" python3 - <<'PY'
import os
import urllib.parse

print(urllib.parse.quote(os.environ["STUHELPER_APP_DB_PASSWORD"], safe=""))
PY
    )"
    value="${value//REPLACE_WITH_STUHELPER_APP_DB_PASSWORD/${encoded_password}}"
  fi
  [[ "${value}" != *"REPLACE_WITH"* ]] || die "DATABASE_URL contains unresolved placeholder"
  printf '%s\n' "${value}"
}

database_url="$(materialize_database_url "${database_url}")"

curl_public_preview() {
  local args=()
  if [[ "${curl_insecure}" == "true" ]]; then
    args+=(--insecure)
  fi
  if [[ -n "${curl_no_proxy}" ]]; then
    args+=(--noproxy "${curl_no_proxy}")
  fi
  if [[ -n "${public_resolve_ip}" ]]; then
    args+=(--resolve "join.stuhelper.com:443:${public_resolve_ip}")
  fi
  command curl "${args[@]}" "$@"
}

query_e2e_json() {
  compose --profile prod run --rm --no-deps -T \
    postgres \
    psql \
      -X \
      -v ON_ERROR_STOP=1 \
      -At \
      -v platform="${platform}" \
      -v guild_id="${guild_id}" \
      -v qq_id="${qq_id}" \
      -v bot_self_id="${bot_self_id}" \
      -v lookback_hours="${lookback_hours}" \
      "${database_url}" <<'SQL'
WITH
input AS (
  SELECT
    :'platform'::text AS platform,
    :'guild_id'::text AS guild_id,
    :'qq_id'::text AS qq_id,
    :'bot_self_id'::text AS bot_self_id,
    make_interval(hours => :'lookback_hours'::int) AS lookback
),
matched_sessions AS (
  SELECT s.*
  FROM public.group_admission_sessions s, input i
  WHERE s.platform = i.platform
    AND s.guild_id = i.guild_id
    AND s.qq_id = i.qq_id
    AND s.created_at >= now() - i.lookback
  ORDER BY s.created_at DESC
),
latest_session AS (
  SELECT *
  FROM matched_sessions
  LIMIT 1
),
joined_user AS (
  SELECT u.id, u.casdoor_subject, u.username, u.email
  FROM public.users u
  JOIN latest_session s ON s.user_id = u.id
),
active_credentials AS (
  SELECT c.*
  FROM public.user_verification_credentials c
  JOIN latest_session s ON s.user_id = c.user_id
  WHERE c.revoked_at IS NULL
    AND (c.expires_at IS NULL OR c.expires_at > now())
),
freshman_applications AS (
  SELECT app.*
  FROM public.freshman_verification_applications app
  JOIN latest_session s ON app.admission_session_id = s.id
),
latest_failure AS (
  SELECT f.*
  FROM public.group_admission_failures f, input i
  WHERE f.platform = i.platform
    AND f.guild_id = i.guild_id
    AND f.qq_id = i.qq_id
),
blacklist AS (
  SELECT b.*
  FROM public.member_blacklist_entries b, input i
  WHERE b.platform = i.platform
    AND b.subject_type = 'qq_user'
    AND b.subject_id = i.qq_id
    AND b.scope_type = 'guild'
    AND b.guild_id = i.guild_id
    AND b.released_at IS NULL
    AND (b.expires_at IS NULL OR b.expires_at > now())
)
SELECT jsonb_build_object(
  'input', jsonb_build_object(
    'platform', (SELECT platform FROM input),
    'guildID', (SELECT guild_id FROM input),
    'qqID', (SELECT qq_id FROM input),
    'botSelfID', (SELECT bot_self_id FROM input),
    'lookbackHours', :'lookback_hours'::int
  ),
  'sessionCount', (SELECT count(*) FROM matched_sessions),
  'latestSession', (
    SELECT jsonb_build_object(
      'id', s.id,
      'platform', s.platform,
      'botSelfID', s.bot_self_id,
      'guildID', s.guild_id,
      'channelIDPresent', s.channel_id <> '',
      'qqID', s.qq_id,
      'userIDPresent', s.user_id IS NOT NULL,
      'authURLRaw', s.auth_url,
      'authURLHost', COALESCE(NULLIF(split_part(regexp_replace(s.auth_url, '^https?://', ''), '/', 1), ''), ''),
      'authURLPath', CASE
        WHEN s.auth_url LIKE 'https://join.stuhelper.com/verify/%' THEN '/verify/redacted'
        ELSE COALESCE(NULLIF(substring(s.auth_url from '^https?://[^/]+([^?]*)'), ''), '')
      END,
      'authURLHasQQQuery', position('qq=' || s.qq_id in s.auth_url) > 0,
      'authURLCanonicalPrefix', s.auth_url LIKE 'https://join.stuhelper.com/verify/%',
      'tokenHashPresent', s.token_hash <> '',
      'tokenConsumed', s.token_consumed_at IS NOT NULL,
      'status', s.status,
      'verified', s.verified_at IS NOT NULL OR s.status = 'verified',
      'cancelledAtPresent', s.cancelled_at IS NOT NULL,
      'botReleaseRecorded', s.status = 'verified' AND s.cancelled_at IS NOT NULL,
      'lastBotErrorPresent', s.last_bot_error IS NOT NULL AND s.last_bot_error <> '',
      'createdAt', s.created_at,
      'updatedAt', s.updated_at,
      'sessionAgeSeconds', floor(extract(epoch from now() - s.created_at))::bigint,
      'updatedAgeSeconds', floor(extract(epoch from now() - s.updated_at))::bigint,
      'tokenExpiresAt', s.token_expires_at,
      'tokenExpired', s.token_expires_at <= now(),
      'linkWaitDeadlineAt', s.link_wait_deadline_at,
      'submissionWaitDeadlineAt', s.submission_wait_deadline_at,
      'initialMuteUntil', s.initial_mute_until
    )
    FROM latest_session s
  ),
  'user', (
    SELECT jsonb_build_object(
      'idPresent', u.id IS NOT NULL,
      'casdoorSubjectPresent', u.casdoor_subject <> '',
      'usernamePresent', u.username <> '',
      'emailPresent', u.email IS NOT NULL AND u.email <> ''
    )
    FROM joined_user u
  ),
  'qqBinding', (
    SELECT jsonb_build_object(
      'bound', EXISTS (
        SELECT 1
        FROM public.user_qq_bindings q
        JOIN latest_session s ON s.user_id = q.user_id
        WHERE q.qq_id = (SELECT qq_id FROM input)
      ),
      'boundAt', (
        SELECT max(q.bound_at)
        FROM public.user_qq_bindings q
        JOIN latest_session s ON s.user_id = q.user_id
        WHERE q.qq_id = (SELECT qq_id FROM input)
      )
    )
  ),
  'studentVerification', jsonb_build_object(
    'activeCredentialCount', (SELECT count(*) FROM active_credentials),
    'kinds', COALESCE((SELECT jsonb_agg(DISTINCT kind ORDER BY kind) FROM active_credentials), '[]'::jsonb),
    'schoolIDs', COALESCE((SELECT jsonb_agg(DISTINCT school_id ORDER BY school_id) FROM active_credentials), '[]'::jsonb)
  ),
  'freshmanApplications', jsonb_build_object(
    'count', (SELECT count(*) FROM freshman_applications),
    'statuses', COALESCE((SELECT jsonb_agg(DISTINCT status ORDER BY status) FROM freshman_applications), '[]'::jsonb)
  ),
  'failure', (
    SELECT jsonb_build_object(
      'failureCount', COALESCE(f.failure_count, 0),
      'lastFailureAt', f.last_failure_at
    )
    FROM latest_failure f
  ),
  'activeBlacklistCount', (SELECT count(*) FROM blacklist)
)::text;
SQL
}

probe_public_preview_json() {
  local raw="$1"
  local meta_file body_file stderr_file
  meta_file="$(mktemp)"
  body_file="$(mktemp)"
  stderr_file="$(mktemp)"

  PUBLIC_PREVIEW_ENABLED="${public_preview_probe_enabled}" \
  RAW_JSON="${raw}" \
  ADMISSION_PUBLIC_BASE_URL_VALUE="${admission_public_base_url}" \
  python3 >"${meta_file}" <<'PY'
import json
import os
from urllib.parse import quote, urlparse

enabled = os.environ["PUBLIC_PREVIEW_ENABLED"] == "true"
raw = json.loads(os.environ["RAW_JSON"])
session = raw.get("latestSession") or {}

if not enabled:
    print(json.dumps({"applicable": False, "reason": "disabled"}, separators=(",", ":")))
    raise SystemExit(0)

if not session:
    print(json.dumps({"applicable": False, "reason": "missing_session"}, separators=(",", ":")))
    raise SystemExit(0)

if session.get("status") != "joined_muted" or session.get("tokenConsumed") is True:
    print(json.dumps({"applicable": False, "reason": "not_unconsumed_joined_session"}, separators=(",", ":")))
    raise SystemExit(0)

auth_url = str(session.get("authURLRaw") or "").strip()
if not auth_url:
    print(json.dumps({"applicable": True, "reason": "missing_auth_url"}, separators=(",", ":")))
    raise SystemExit(0)

parsed = urlparse(auth_url)
base = os.environ["ADMISSION_PUBLIC_BASE_URL_VALUE"].rstrip("/")
expected = urlparse(base)
segments = [segment for segment in parsed.path.split("/") if segment]
if parsed.scheme != expected.scheme or parsed.netloc != expected.netloc or parsed.query or len(segments) < 2 or segments[-2] != "verify":
    print(json.dumps({"applicable": True, "reason": "invalid_auth_url_shape"}, separators=(",", ":")))
    raise SystemExit(0)

token = segments[-1]
preview_path = f"/api/v1/admission/sessions/{quote(token, safe='')}"
preview_url = f"{base}{preview_path}"

print(json.dumps({
    "applicable": True,
    "previewURL": preview_url,
    "expectedSessionID": session.get("id"),
}, separators=(",", ":")))
PY

  local applicable
  applicable="$(python3 - "${meta_file}" <<'PY'
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as fh:
    data = json.load(fh)
print("true" if data.get("applicable") is True and data.get("previewURL") else "false")
PY
  )"

  if [[ "${applicable}" != "true" ]]; then
    python3 - "${meta_file}" <<'PY'
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as fh:
    data = json.load(fh)
data.pop("previewURL", None)
data.pop("expectedSessionID", None)
print(json.dumps(data, ensure_ascii=True, separators=(",", ":")))
PY
    rm -f "${meta_file}" "${body_file}" "${stderr_file}"
    return 0
  fi

  local preview_url expected_session_id curl_output curl_status
  preview_url="$(python3 - "${meta_file}" <<'PY'
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as fh:
    print(json.load(fh)["previewURL"])
PY
  )"
  expected_session_id="$(python3 - "${meta_file}" <<'PY'
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as fh:
    print(json.load(fh).get("expectedSessionID") or "")
PY
  )"

  set +e
  curl_output="$(
    curl_public_preview \
      -sS \
      -o "${body_file}" \
      -w '%{http_code}\n%{remote_ip}\n%{size_download}\n' \
      "${preview_url}" 2>"${stderr_file}"
  )"
  curl_status=$?
  set -e

  CURL_STATUS="${curl_status}" \
  CURL_OUTPUT="${curl_output}" \
  RESPONSE_BODY_FILE="${body_file}" \
  CURL_STDERR_FILE="${stderr_file}" \
  EXPECTED_SESSION_ID="${expected_session_id}" \
  python3 <<'PY'
import json
import os

lines = os.environ.get("CURL_OUTPUT", "").splitlines()
http_status = int(lines[0]) if len(lines) > 0 and lines[0].isdigit() else 0
remote_ip = lines[1] if len(lines) > 1 else ""
size = int(lines[2]) if len(lines) > 2 and lines[2].isdigit() else 0
curl_exit = int(os.environ.get("CURL_STATUS") or 0)
expected_session_id = os.environ.get("EXPECTED_SESSION_ID") or ""

body = {}
try:
    with open(os.environ["RESPONSE_BODY_FILE"], "r", encoding="utf-8") as fh:
        body = json.load(fh)
except Exception:
    body = {}

stderr = ""
try:
    with open(os.environ["CURL_STDERR_FILE"], "r", encoding="utf-8") as fh:
        stderr = fh.read().strip()
except Exception:
    stderr = ""

data = body.get("data") if isinstance(body, dict) else {}
error = body.get("error") if isinstance(body, dict) else {}
result = {
    "applicable": True,
    "curlExitCode": curl_exit,
    "httpStatus": http_status,
    "remoteIP": remote_ip,
    "responseBytes": size,
    "success": body.get("success") is True if isinstance(body, dict) else False,
    "sessionIDMatches": bool(expected_session_id) and isinstance(data, dict) and data.get("id") == expected_session_id,
}
if isinstance(error, dict) and error.get("code"):
    result["errorCode"] = error.get("code")
if curl_exit != 0 and stderr:
    result["curlErrorPresent"] = True
print(json.dumps(result, ensure_ascii=True, separators=(",", ":")))
PY

  rm -f "${meta_file}" "${body_file}" "${stderr_file}"
}

raw_json="$(query_e2e_json)" || die "admission join E2E evidence query failed"
public_preview_json="$(probe_public_preview_json "${raw_json}")" || die "admission public preview probe failed"

tmp_file="$(mktemp)"
trap 'rm -f "${tmp_file}"' EXIT

set +e
GENERATED_AT="$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
EXPECTED_STAGE="${expected_stage}" \
MAX_SESSION_AGE_MINUTES="${max_session_age_minutes}" \
ADMISSION_PUBLIC_BASE_URL_VALUE="${admission_public_base_url}" \
RAW_JSON="${raw_json}" \
PUBLIC_PREVIEW_JSON="${public_preview_json}" \
python3 <<'PY' >"${tmp_file}"
import json
import os
import sys
from urllib.parse import urlsplit

raw = json.loads(os.environ["RAW_JSON"])
public_preview = json.loads(os.environ.get("PUBLIC_PREVIEW_JSON") or "{}")
stage = os.environ["EXPECTED_STAGE"]
max_session_age_minutes = int(os.environ["MAX_SESSION_AGE_MINUTES"])
max_session_age_seconds = max_session_age_minutes * 60
admission_base = os.environ["ADMISSION_PUBLIC_BASE_URL_VALUE"]
checks = []

def add(name, passed, detail=""):
    checks.append({
        "name": name,
        "kind": "admission_join_e2e",
        "passed": bool(passed),
        "detail": detail,
    })

session = raw.get("latestSession") or {}
if isinstance(session, dict):
    session["authURLRawPresent"] = bool(session.get("authURLRaw"))
    session.pop("authURLRaw", None)
qq_binding = raw.get("qqBinding") or {}
student = raw.get("studentVerification") or {}
freshman = raw.get("freshmanApplications") or {}
user = raw.get("user") or {}

session_count = int(raw.get("sessionCount") or 0)
add("real join created an admission session", session_count >= 1, f"sessionCount={session_count}")
session_age_seconds = session.get("sessionAgeSeconds")
updated_age_seconds = session.get("updatedAgeSeconds")
add(
    "latest session is fresh enough for this E2E run",
    isinstance(session_age_seconds, int) and session_age_seconds <= max_session_age_seconds,
    f"sessionAgeSeconds={session_age_seconds}, maxSessionAgeSeconds={max_session_age_seconds}",
)
add("session uses qq business platform", session.get("platform") == "qq", f"platform={session.get('platform')}")
add("session is for expected guild", session.get("guildID") == raw.get("input", {}).get("guildID"), f"guildID={session.get('guildID')}")
add(
    "session is for expected qq",
    session.get("qqID") == raw.get("input", {}).get("qqID"),
    f"qqID={session.get('qqID')}, expectedQQID={raw.get('input', {}).get('qqID')}",
)
add("session records bot self id", session.get("botSelfID") == raw.get("input", {}).get("botSelfID"), f"botSelfID={session.get('botSelfID')}")
add("session has channel id", session.get("channelIDPresent") is True)
add("session has token hash but no raw token", session.get("tokenHashPresent") is True)
add("session auth url uses join host", session.get("authURLHost") == "join.stuhelper.com", f"host={session.get('authURLHost')}")
add("session auth url uses /verify token path", str(session.get("authURLPath") or "").startswith("/verify/"), f"path={session.get('authURLPath')}")
add("session auth url does not carry qq query", session.get("authURLHasQQQuery") is False)
add("session auth url has canonical prefix", session.get("authURLCanonicalPrefix") is True)
add("session has no active blacklist", int(raw.get("activeBlacklistCount") or 0) == 0, f"activeBlacklistCount={raw.get('activeBlacklistCount')}")
add("session has no bot error", session.get("lastBotErrorPresent") is False)

join_created_statuses = {"joined_muted", "linked", "material_submitted", "verified"}
add("session reached admission join-created stage", session.get("status") in join_created_statuses, f"status={session.get('status')}")

if session.get("status") == "joined_muted" and session.get("tokenConsumed") is False:
    add(
        "unconsumed session public preview API is reachable",
        public_preview.get("applicable") is True
        and public_preview.get("curlExitCode") == 0
        and public_preview.get("httpStatus") == 200
        and public_preview.get("success") is True
        and public_preview.get("sessionIDMatches") is True,
        f"httpStatus={public_preview.get('httpStatus')}, remoteIP={public_preview.get('remoteIP')}, responseBytes={public_preview.get('responseBytes')}, errorCode={public_preview.get('errorCode')}",
    )

if stage in {"flow-completed", "bot-released"}:
    completed_statuses = {"linked", "material_submitted", "verified"}
    add("user opened canonical link and bound session", session.get("status") in completed_statuses, f"status={session.get('status')}")
    add("token was consumed by authenticated user", session.get("tokenConsumed") is True)
    add("session has user id", session.get("userIDPresent") is True)
    add("user record exists", user.get("idPresent") is True)
    add("qq binding exists for session user", qq_binding.get("bound") is True)
    has_student_credential = int(student.get("activeCredentialCount") or 0) > 0
    has_submitted_freshman_material = (
        int(freshman.get("count") or 0) > 0
        and session.get("status") == "material_submitted"
    )
    add(
        "student verification credential or submitted freshman material recorded",
        has_student_credential or has_submitted_freshman_material,
        f"activeCredentialCount={student.get('activeCredentialCount')}, freshmanApplicationCount={freshman.get('count')}, freshmanStatuses={freshman.get('statuses')}, sessionStatus={session.get('status')}",
    )
    add(
        "session reached release-eligible or review-pending state",
        session.get("status") in {"verified", "material_submitted"},
        f"status={session.get('status')}",
    )

if stage == "bot-released":
    add(
        "release requires active student verification credential",
        int(student.get("activeCredentialCount") or 0) > 0,
        f"activeCredentialCount={student.get('activeCredentialCount')}, credentialKinds={student.get('kinds')}, freshmanStatuses={freshman.get('statuses')}",
    )
    add("session reached verified state before release", session.get("status") == "verified", f"status={session.get('status')}")
    add("backend recorded successful bot release", session.get("botReleaseRecorded") is True)
    add("session cancelled marker is present after release", session.get("cancelledAtPresent") is True)
    add(
        "bot release evidence is fresh enough for this E2E run",
        isinstance(updated_age_seconds, int) and updated_age_seconds <= max_session_age_seconds,
        f"updatedAgeSeconds={updated_age_seconds}, maxSessionAgeSeconds={max_session_age_seconds}",
    )

passed_count = sum(1 for item in checks if item["passed"])
failed_count = len(checks) - passed_count

evidence = {
    "generatedAt": os.environ["GENERATED_AT"],
    "expectedStage": stage,
    "passed": failed_count == 0,
    "summary": {"passed": passed_count, "failed": failed_count},
    "admissionPublicBaseURL": admission_base,
    "maxSessionAgeMinutes": max_session_age_minutes,
    "input": raw.get("input", {}),
    "session": session,
    "publicPreview": public_preview,
    "user": user,
    "qqBinding": qq_binding,
    "studentVerification": student,
    "freshmanApplications": freshman,
    "failure": raw.get("failure") or {"failureCount": 0, "lastFailureAt": None},
    "activeBlacklistCount": int(raw.get("activeBlacklistCount") or 0),
    "checks": checks,
}
print(json.dumps(evidence, ensure_ascii=True, indent=2))
sys.exit(0 if evidence["passed"] else 1)
PY

status=$?
set -e
if [[ "${evidence_file}" != "-" ]]; then
  mkdir -p "$(dirname "${evidence_file}")"
  install -m 600 "${tmp_file}" "${evidence_file}"
  log "wrote admission join E2E evidence to ${evidence_file}" >&2
fi
cat "${tmp_file}"

if (( status != 0 )); then
  die "admission join E2E evidence failed for stage ${expected_stage}"
fi

log "admission join E2E evidence passed for stage ${expected_stage}"
