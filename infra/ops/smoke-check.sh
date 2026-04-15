#!/usr/bin/env bash
# 部署后业务冒烟检查脚本
# 两阶段：1) 等待基础设施就绪  2) 验证业务端点
#
# 用法:
#   ./smoke-check.sh                           # 本地默认地址
#   API_BASE_URL=https://api.example.com ./smoke-check.sh  # 远端
set -euo pipefail

API_BASE_URL="${API_BASE_URL:-http://127.0.0.1:8080}"
WEB_BASE_URL="${WEB_BASE_URL:-http://127.0.0.1:3000}"
ADMIN_BASE_URL="${ADMIN_BASE_URL:-http://127.0.0.1:${ADMIN_EXTERNAL_PORT:-3001}}"
ADMIN_SMOKE_PATH="${ADMIN_SMOKE_PATH:-/admin/}"
CHECK_ADMIN="${CHECK_ADMIN:-true}"
SMOKE_RETRIES="${SMOKE_RETRIES:-30}"
SMOKE_SLEEP_SECONDS="${SMOKE_SLEEP_SECONDS:-2}"

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
  local status
  status=$(curl -s -o /dev/null -w '%{http_code}' --max-time 10 "$url" 2>/dev/null || echo "000")
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
check_body   "登录 URL" "${API_BASE_URL}/api/v1/auth/login-url" '"url":'
check_status "未认证保护端点" "${API_BASE_URL}/api/v1/auth/me" "401"

echo ""
echo "── 指标与观测 ──"
check_status "指标端点需认证" "${API_BASE_URL}/metrics" "401"

# OIDC（可选）
ZITADEL_ISSUER="${ZITADEL_ISSUER:-}"
if [ -n "$ZITADEL_ISSUER" ]; then
  echo ""
  echo "── OIDC ──"
  check_body "Zitadel well-known" "${ZITADEL_ISSUER}/.well-known/openid-configuration" '"issuer":'
else
  echo "  ⚠️  ZITADEL_ISSUER 未设置，跳过 OIDC 检查"
  WARN=$((WARN + 1))
fi

# Grafana（可选）
GRAFANA_URL="${GRAFANA_URL:-}"
if [ -n "$GRAFANA_URL" ]; then
  echo ""
  echo "── Grafana ──"
  check_body "Grafana 健康" "${GRAFANA_URL}/api/health" '"database":"ok"'
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
