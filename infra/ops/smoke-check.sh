#!/usr/bin/env bash
# 部署后业务冒烟检查脚本
# 两阶段：1) 等待基础设施就绪  2) 验证业务端点
#
# 用法:
#   ./smoke-check.sh                           # 本地默认地址
#   API_BASE_URL=https://stuhelper.com WEB_BASE_URL=https://stuhelper.com ADMIN_BASE_URL=https://stuhelper.com ./smoke-check.sh
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
load_env

API_BASE_URL="${API_BASE_URL:-http://127.0.0.1:${BACKEND_EXTERNAL_PORT:-18080}}"
WEB_BASE_URL="${WEB_BASE_URL:-http://127.0.0.1:${WEB_EXTERNAL_PORT:-18000}}"
ADMIN_BASE_URL="${ADMIN_BASE_URL:-http://127.0.0.1:${ADMIN_EXTERNAL_PORT:-18001}}"
ADMIN_SMOKE_PATH="${ADMIN_SMOKE_PATH:-/admin/}"
CHECK_ADMIN="${CHECK_ADMIN:-true}"
SMOKE_RETRIES="${SMOKE_RETRIES:-30}"
SMOKE_SLEEP_SECONDS="${SMOKE_SLEEP_SECONDS:-2}"

curl() {
  if [[ "${SMOKE_CHECK_CURL_INSECURE:-false}" == "true" ]]; then
    command curl --insecure "$@"
    return
  fi
  command curl "$@"
}

PASS=0
FAIL=0
WARN=0

# ── 阶段 1：等待基础设施就绪 ──

curl_wait() {
  local name="$1" url="$2"
  local retries="${3:-$SMOKE_RETRIES}" sleep_seconds="${4:-$SMOKE_SLEEP_SECONDS}"
  echo "[smoke] 等待 ${name}: ${url} (最多 ${retries} 次)"
  for ((attempt = 1; attempt <= retries; attempt++)); do
    if curl --fail --silent --show-error --location --max-time 10 "$url" >/dev/null 2>&1; then
      echo "  ✅ ${name} 就绪"
      PASS=$((PASS + 1))
      return 0
    fi
    sleep "${sleep_seconds}"
  done
  echo "  ❌ ${name} 超时未就绪: ${url}" >&2
  FAIL=$((FAIL + 1))
  return 1
}

echo "━━━ 阶段 1：基础设施就绪 ━━━"
curl_wait "API live" "${API_BASE_URL}/health/live"
curl_wait "API ready" "${API_BASE_URL}/health/ready"
curl_wait "Web 前端" "${WEB_BASE_URL}/"
if [[ "$CHECK_ADMIN" == "true" ]]; then
  curl_wait "Admin 前端" "${ADMIN_BASE_URL}${ADMIN_SMOKE_PATH}"
fi

# ── 阶段 2：业务端点验证 ──

check_status() {
  local name="$1" url="$2" expect="${3:-200}"
  local method="${4:-GET}"
  local status
  status=$(curl -s -o /dev/null -w '%{http_code}' --max-time 10 -X "${method}" "$url" 2>/dev/null || echo "000")
  if [ "$status" = "$expect" ]; then
    echo "  ✅ $name (HTTP $status)"
    PASS=$((PASS + 1))
  else
    echo "  ❌ $name (期望 $expect, 实际 $status)"
    FAIL=$((FAIL + 1))
  fi
}

check_body() {
  local name="$1" url="$2" pattern="$3"
  local body
  body=$(curl -s --max-time 10 "$url" 2>/dev/null || echo "")
  if echo "$body" | grep -q "$pattern"; then
    echo "  ✅ $name"
    PASS=$((PASS + 1))
  else
    echo "  ❌ $name (响应中未包含 '$pattern')"
    FAIL=$((FAIL + 1))
  fi
}

check_body_regex() {
  local name="$1" url="$2" pattern="$3"
  local body
  body=$(curl -s --max-time 10 "$url" 2>/dev/null || echo "")
  if echo "$body" | grep -Eq "$pattern"; then
    echo "  ✅ $name"
    PASS=$((PASS + 1))
  else
    echo "  ❌ $name (响应中未匹配 '$pattern')"
    FAIL=$((FAIL + 1))
  fi
}

echo ""
echo "━━━ 阶段 2：业务端点验证 ━━━"

echo ""
echo "── 公开 API ──"
check_status "院系列表" "${API_BASE_URL}/api/v1/course/departments"
check_status "学期列表" "${API_BASE_URL}/api/v1/course/terms"
check_status "课程分类" "${API_BASE_URL}/api/v1/course/categories"
check_status "课程分组" "${API_BASE_URL}/api/v1/course/courses/grouped"
check_body   "课程列表返回 success" "${API_BASE_URL}/api/v1/course/courses?page=1&pageSize=5" '"success":true'

echo ""
echo "── 认证端点 ──"
check_body   "登录 URL" "${API_BASE_URL}/api/v1/auth/login?platform=native" '"url":'
check_status "未认证保护端点" "${API_BASE_URL}/api/v1/auth/me" "401"

echo ""
echo "── OIDC 回调流程 ──"
# 验证登录 URL 接口返回的 JSON 中包含授权 URL（url 字段）
check_body   "login-url 返回授权地址" "${API_BASE_URL}/api/v1/auth/login?platform=native" '"url":'
# 验证 callback 端点在缺少 code 参数时返回 400
check_status "callback 无 code 返回 400" "${API_BASE_URL}/api/v1/auth/callback" "400"
# 验证 exchange-native 端点在无请求体时返回 400
check_status "exchange-native 无 body 返回 400" "${API_BASE_URL}/api/v1/auth/exchange-native" "400" "POST"
# 验证 refresh 端点在无 cookie 时返回 401（未携带刷新令牌）
check_status "refresh 无 cookie 返回 401" "${API_BASE_URL}/api/v1/auth/refresh" "401" "POST"

echo ""
echo "── 指标与观测 ──"
case "${APP_ENV:-development}" in
  production|prod-parity)
    check_status "生产指标端点需认证" "${API_BASE_URL}/metrics" "401"
    ;;
  *)
    check_status "开发指标端点可访问" "${API_BASE_URL}/metrics" "200"
    ;;
esac

# SSO OIDC（可选）
SSO_PUBLIC_BASE_URL="${SSO_PUBLIC_BASE_URL:-${CASDOOR_PUBLIC_AUTH_BASE_URL:-${WEB_VITE_SSO_URL:-${CASDOOR_ISSUER:-}}}}"
if [ -n "$SSO_PUBLIC_BASE_URL" ]; then
  echo ""
  echo "── SSO OIDC ──"
  check_body "SSO well-known" "${SSO_PUBLIC_BASE_URL%/}/.well-known/openid-configuration" '"issuer":'
else
  echo "  ⚠️  SSO_PUBLIC_BASE_URL / CASDOOR_PUBLIC_AUTH_BASE_URL / WEB_VITE_SSO_URL / CASDOOR_ISSUER 均未设置，跳过 SSO OIDC 检查"
  WARN=$((WARN + 1))
fi

CASDOOR_ISSUER="${CASDOOR_ISSUER:-}"
if [[ "${SMOKE_CHECK_CASDOOR_UPSTREAM_ENABLED:-false}" == "true" ]]; then
  if [ -n "$CASDOOR_ISSUER" ]; then
    echo ""
    echo "── Casdoor issuer ──"
    check_body "Casdoor issuer well-known" "${CASDOOR_ISSUER%/}/.well-known/openid-configuration" '"issuer":'
  else
    echo "  ⚠️  CASDOOR_ISSUER 未设置，跳过 Casdoor upstream 检查"
    WARN=$((WARN + 1))
  fi
fi

# Grafana（可选）
GRAFANA_URL="${GRAFANA_URL:-}"
if [ -n "$GRAFANA_URL" ]; then
  echo ""
  echo "── Grafana ──"
  check_body_regex "Grafana 健康" "${GRAFANA_URL}/api/health" '"database"[[:space:]]*:[[:space:]]*"ok"'
else
  echo "  ⚠️  GRAFANA_URL 未设置，跳过 Grafana 检查"
  WARN=$((WARN + 1))
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "结果: ✅ $PASS 通过  ❌ $FAIL 失败  ⚠️  $WARN 跳过"

if [ "$FAIL" -gt 0 ]; then
  echo "❌ 冒烟检查未通过"
  exit 1
fi

echo "✅ 冒烟检查全部通过"
