#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

if [[ "${ENV_FILE}" == "${REPO_ROOT}/.env" && -f "${REPO_ROOT}/.env.prod.shared" ]]; then
  ENV_FILE="${REPO_ROOT}/.env.prod.shared"
fi
if [[ -z "${SECRETS_ENV_FILE:-}" && -f "${REPO_ROOT}/.env.prod.secrets.local" ]]; then
  SECRETS_ENV_FILE="${REPO_ROOT}/.env.prod.secrets.local"
fi
if [[ "${GENERATED_ENV_FILE}" == "${REPO_ROOT}/.env.generated" && -f "${REPO_ROOT}/.env.prod.generated" ]]; then
  GENERATED_ENV_FILE="${REPO_ROOT}/.env.prod.generated"
fi
if [[ "${GENERATED_SECRET_ENV_FILE}" == "${REPO_ROOT}/.env.generated.secrets" && -f "${REPO_ROOT}/.env.prod.generated.secrets" ]]; then
  GENERATED_SECRET_ENV_FILE="${REPO_ROOT}/.env.prod.generated.secrets"
fi

require_cmd python3
load_env

EVIDENCE_FILE="${TENCENT_SES_TEMPLATE_SMOKE_EVIDENCE_FILE:-${REPO_ROOT}/infra/generated/tencent-ses-template-smoke.json}"
mkdir -p "$(dirname "${EVIDENCE_FILE}")"

TENCENT_SES_TEMPLATE_SMOKE_EVIDENCE_FILE="${EVIDENCE_FILE}" python3 <<'PY'
import datetime as dt
import hashlib
import hmac
import json
import os
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path

SERVICE = "ses"
ACTION = "GetEmailTemplate"
VERSION = "2020-10-02"


def require_env(name: str) -> str:
    value = os.environ.get(name, "").strip()
    if not value:
        raise SystemExit(f"[tencent-ses-template-smoke][error] missing env: {name}")
    return value


def sha256_hex(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def hmac_sha256(key: bytes, data: str) -> bytes:
    return hmac.new(key, data.encode("utf-8"), hashlib.sha256).digest()


secret_id = require_env("EMAIL_TENCENT_SECRET_ID")
secret_key = require_env("EMAIL_TENCENT_SECRET_KEY")
region = os.environ.get("EMAIL_TENCENT_REGION", "ap-guangzhou").strip() or "ap-guangzhou"
endpoint = os.environ.get("EMAIL_TENCENT_ENDPOINT", "ses.tencentcloudapi.com").strip() or "ses.tencentcloudapi.com"
template_id = int(require_env("EMAIL_TENCENT_TEMPLATE_ID"))

if "://" not in endpoint:
    endpoint = "https://" + endpoint
parsed = urllib.parse.urlparse(endpoint)
if parsed.scheme not in {"http", "https"} or not parsed.netloc:
    raise SystemExit("[tencent-ses-template-smoke][error] EMAIL_TENCENT_ENDPOINT must be an http(s) endpoint")
path = parsed.path or "/"
url = urllib.parse.urlunparse((parsed.scheme, parsed.netloc, path, "", "", ""))

payload = json.dumps({"TemplateID": template_id}, separators=(",", ":")).encode("utf-8")
timestamp = int(time.time())
timestamp_text = str(timestamp)
date = dt.datetime.fromtimestamp(timestamp, dt.timezone.utc).strftime("%Y-%m-%d")
content_type = "application/json"

canonical_request = "\n".join([
    "POST",
    path,
    "",
    f"content-type:{content_type}",
    f"host:{parsed.netloc}",
    "",
    "content-type;host",
    sha256_hex(payload),
])
credential_scope = f"{date}/{SERVICE}/tc3_request"
string_to_sign = "\n".join([
    "TC3-HMAC-SHA256",
    timestamp_text,
    credential_scope,
    sha256_hex(canonical_request.encode("utf-8")),
])
secret_date = hmac_sha256(("TC3" + secret_key).encode("utf-8"), date)
secret_service = hmac_sha256(secret_date, SERVICE)
secret_signing = hmac_sha256(secret_service, "tc3_request")
signature = hmac.new(secret_signing, string_to_sign.encode("utf-8"), hashlib.sha256).hexdigest()
authorization = (
    f"TC3-HMAC-SHA256 Credential={secret_id}/{credential_scope}, "
    f"SignedHeaders=content-type;host, Signature={signature}"
)

request = urllib.request.Request(
    url,
    data=payload,
    method="POST",
    headers={
        "Content-Type": content_type,
        "X-TC-Action": ACTION,
        "X-TC-Version": VERSION,
        "X-TC-Region": region,
        "X-TC-Timestamp": timestamp_text,
        "Authorization": authorization,
    },
)

try:
    with urllib.request.urlopen(request, timeout=15) as response:
        body = response.read()
        http_status = response.status
except urllib.error.HTTPError as exc:
    body = exc.read()
    http_status = exc.code
except urllib.error.URLError as exc:
    raise SystemExit(f"[tencent-ses-template-smoke][error] request failed: {exc.reason}") from exc

try:
    result = json.loads(body.decode("utf-8"))
except json.JSONDecodeError as exc:
    raise SystemExit(f"[tencent-ses-template-smoke][error] invalid JSON response: {exc}") from exc

response = result.get("Response", {})
api_error = response.get("Error")
evidence = {
    "checkedAt": dt.datetime.now(dt.timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z"),
    "endpoint": parsed.netloc,
    "region": region,
    "templateID": template_id,
    "httpStatus": http_status,
    "requestID": response.get("RequestId", ""),
    "templateName": response.get("TemplateName", ""),
    "templateStatus": response.get("TemplateStatus"),
    "templateApproved": response.get("TemplateStatus") == 0,
}
Path(os.environ["TENCENT_SES_TEMPLATE_SMOKE_EVIDENCE_FILE"]).write_text(
    json.dumps(evidence, ensure_ascii=False, indent=2) + "\n",
    encoding="utf-8",
)

if api_error:
    code = api_error.get("Code", "Unknown")
    message = api_error.get("Message", "")
    raise SystemExit(f"[tencent-ses-template-smoke][error] Tencent API error {code}: {message}")
if http_status < 200 or http_status >= 300:
    raise SystemExit(f"[tencent-ses-template-smoke][error] unexpected HTTP status: {http_status}")
if response.get("TemplateStatus") != 0:
    raise SystemExit(
        "[tencent-ses-template-smoke][error] template is not approved: "
        f"status={response.get('TemplateStatus')}"
    )

print(
    "[tencent-ses-template-smoke] ok "
    f"templateID={template_id} status={response.get('TemplateStatus')} "
    f"evidence={os.environ['TENCENT_SES_TEMPLATE_SMOKE_EVIDENCE_FILE']}"
)
PY
