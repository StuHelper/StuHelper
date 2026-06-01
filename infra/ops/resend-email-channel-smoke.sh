#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

require_cmd python3
load_env

EVIDENCE_FILE="${RESEND_EMAIL_CHANNEL_SMOKE_EVIDENCE_FILE:-${REPO_ROOT}/infra/generated/resend-email-channel-smoke.json}"
mkdir -p "$(dirname "${EVIDENCE_FILE}")"

export REPO_ROOT
RESEND_EMAIL_CHANNEL_SMOKE_EVIDENCE_FILE="${EVIDENCE_FILE}" python3 <<'PY'
import datetime as dt
import hashlib
import html
import json
import os
import sys
import urllib.error
import urllib.parse
import urllib.request
import uuid
from pathlib import Path


def require_env(name: str) -> str:
    value = os.environ.get(name, "").strip()
    if not value:
        raise SystemExit(f"[resend-email-channel-smoke][error] missing env: {name}")
    return value


def optional_env(name: str, default: str = "") -> str:
    return os.environ.get(name, default).strip() or default


def normalize_endpoint(raw: str) -> str:
    endpoint = raw.strip() or "https://api.resend.com/emails"
    if "://" not in endpoint:
        endpoint = "https://" + endpoint
    parsed = urllib.parse.urlparse(endpoint)
    if parsed.scheme not in {"http", "https"} or not parsed.netloc:
        raise SystemExit("[resend-email-channel-smoke][error] EMAIL_RESEND_ENDPOINT must be an http(s) endpoint")
    path = parsed.path if parsed.path and parsed.path != "/" else "/emails"
    return urllib.parse.urlunparse((parsed.scheme, parsed.netloc, path, "", "", ""))


def render_templates(repo_root: Path, code: str, purpose: str, school_name: str, expire_minutes: int) -> tuple[str, str]:
    template_dir = repo_root / "infra" / "email-templates" / "tencent-ses"
    html_template = (template_dir / "stuhelper-school-email-otp.html").read_text(encoding="utf-8")
    text_template = (template_dir / "stuhelper-school-email-otp.txt").read_text(encoding="utf-8")
    html_replacements = {
        "{{code}}": html.escape(code),
        "{{purpose}}": html.escape(purpose),
        "{{school_name}}": html.escape(school_name),
        "{{expire_minutes}}": html.escape(str(expire_minutes)),
    }
    text_replacements = {
        "{{code}}": code,
        "{{purpose}}": purpose,
        "{{school_name}}": school_name,
        "{{expire_minutes}}": str(expire_minutes),
    }
    rendered_html = html_template
    for key, value in html_replacements.items():
        rendered_html = rendered_html.replace(key, value)
    rendered_text = text_template
    for key, value in text_replacements.items():
        rendered_text = rendered_text.replace(key, value)
    return rendered_html, rendered_text


def write_evidence(path: Path, evidence: dict) -> None:
    path.write_text(json.dumps(evidence, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")


repo_root = Path(require_env("REPO_ROOT"))
api_key = require_env("EMAIL_RESEND_API_KEY")
recipient = optional_env("RESEND_EMAIL_SMOKE_TO", optional_env("EMAIL_SMOKE_TO", ""))
if not recipient:
    raise SystemExit("[resend-email-channel-smoke][error] missing env: RESEND_EMAIL_SMOKE_TO")

endpoint = normalize_endpoint(optional_env("EMAIL_RESEND_ENDPOINT", "https://api.resend.com/emails"))
from_address = require_env("EMAIL_FROM")
from_name = optional_env("EMAIL_FROM_NAME", "StuHelper 系统邮件")
reply_to = optional_env("EMAIL_RESEND_REPLY_TO", optional_env("EMAIL_TENCENT_REPLY_TO", ""))
subject = optional_env("EMAIL_STUDENT_VERIFICATION_SUBJECT", "学生认证验证码")
code = optional_env("RESEND_EMAIL_SMOKE_CODE", "123456")
purpose = optional_env("RESEND_EMAIL_SMOKE_PURPOSE", optional_env("EMAIL_TENCENT_TEMPLATE_PURPOSE", "学校邮箱认证"))
school_name = optional_env("RESEND_EMAIL_SMOKE_SCHOOL_NAME", optional_env("EMAIL_TENCENT_TEMPLATE_SCHOOL_NAME", "北京航空航天大学"))
expire_raw = optional_env("RESEND_EMAIL_SMOKE_EXPIRE_MINUTES", optional_env("EMAIL_TENCENT_TEMPLATE_EXPIRE_MINUTES", "5"))
try:
    expire_minutes = int(expire_raw)
except ValueError as exc:
    raise SystemExit("[resend-email-channel-smoke][error] expire minutes must be an integer") from exc
if expire_minutes <= 0:
    raise SystemExit("[resend-email-channel-smoke][error] expire minutes must be positive")

rendered_html, rendered_text = render_templates(repo_root, code, purpose, school_name, expire_minutes)
sender = from_address
if from_name:
    sender = f"{from_name} <{from_address}>"

payload = {
    "from": sender,
    "to": [recipient],
    "subject": subject,
    "html": rendered_html,
    "text": rendered_text,
}
if reply_to:
    payload["reply_to"] = [reply_to]

recipient_domain = recipient.rsplit("@", 1)[-1] if "@" in recipient else ""
recipient_hash = hashlib.sha256(recipient.lower().encode("utf-8")).hexdigest()
evidence = {
    "checkedAt": dt.datetime.now(dt.timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z"),
    "endpoint": urllib.parse.urlparse(endpoint).netloc,
    "path": urllib.parse.urlparse(endpoint).path,
    "recipientDomain": recipient_domain,
    "recipientHashPrefix": recipient_hash[:12],
    "subject": subject,
    "fromAddress": from_address,
    "fromNamePresent": bool(from_name),
    "htmlTemplate": "infra/email-templates/tencent-ses/stuhelper-school-email-otp.html",
    "textTemplate": "infra/email-templates/tencent-ses/stuhelper-school-email-otp.txt",
    "htmlLength": len(rendered_html),
    "textLength": len(rendered_text),
}

request = urllib.request.Request(
    endpoint,
    data=json.dumps(payload, ensure_ascii=False, separators=(",", ":")).encode("utf-8"),
    method="POST",
    headers={
        "Authorization": "Bearer " + api_key,
        "Content-Type": "application/json",
        "Idempotency-Key": "resend-email-channel-smoke-" + uuid.uuid4().hex,
    },
)

try:
    with urllib.request.urlopen(request, timeout=20) as response:
        body = response.read()
        http_status = response.status
except urllib.error.HTTPError as exc:
    body = exc.read()
    http_status = exc.code
except urllib.error.URLError as exc:
    evidence["error"] = "request failed"
    write_evidence(Path(os.environ["RESEND_EMAIL_CHANNEL_SMOKE_EVIDENCE_FILE"]), evidence)
    raise SystemExit(f"[resend-email-channel-smoke][error] request failed: {exc.reason}") from exc

evidence["httpStatus"] = http_status
try:
    result = json.loads(body.decode("utf-8")) if body.strip() else {}
except json.JSONDecodeError as exc:
    evidence["error"] = "invalid JSON response"
    write_evidence(Path(os.environ["RESEND_EMAIL_CHANNEL_SMOKE_EVIDENCE_FILE"]), evidence)
    raise SystemExit(f"[resend-email-channel-smoke][error] invalid JSON response: {exc}") from exc

if http_status < 200 or http_status >= 300:
    evidence["error"] = result.get("name") or "resend_api_error"
    write_evidence(Path(os.environ["RESEND_EMAIL_CHANNEL_SMOKE_EVIDENCE_FILE"]), evidence)
    message = result.get("message", "")
    raise SystemExit(f"[resend-email-channel-smoke][error] Resend API error status={http_status}: {message}")

email_id = str(result.get("id", "")).strip()
if not email_id:
    evidence["error"] = "missing email id"
    write_evidence(Path(os.environ["RESEND_EMAIL_CHANNEL_SMOKE_EVIDENCE_FILE"]), evidence)
    raise SystemExit("[resend-email-channel-smoke][error] Resend response missing email id")

evidence["emailID"] = email_id
evidence["sent"] = True
write_evidence(Path(os.environ["RESEND_EMAIL_CHANNEL_SMOKE_EVIDENCE_FILE"]), evidence)

print(
    "[resend-email-channel-smoke] ok "
    f"status={http_status} id={email_id} evidence={os.environ['RESEND_EMAIL_CHANNEL_SMOKE_EVIDENCE_FILE']}"
)
PY
