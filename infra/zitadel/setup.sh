#!/usr/bin/env bash
# infra/zitadel/setup.sh — 自动化初始化 Zitadel Project、OIDC Application 和 Project Roles
#
# 依赖：
#   - jq
#   - curl
#   - Zitadel API 已启动且可访问
#   - Admin PAT（通过 docker volume 或环境变量获取）
#
# 使用方式：
#   ./infra/zitadel/setup.sh
#   ZITADEL_URL=https://sso.stuhelper.com ./infra/zitadel/setup.sh
#
# 输出：
#   .env 片段，包含 ZITADEL_CLIENT_ID、ZITADEL_CLIENT_SECRET、ZITADEL_PROJECT_ID
set -euo pipefail

# ── 配置 ─────────────────────────────────────────────────
ZITADEL_URL="${ZITADEL_URL:-http://localhost:8085}"
PAT_FILE="${PAT_FILE:-}"
PAT="${ZITADEL_MANAGEMENT_PAT:-}"
STACK_NAME_VALUE="${STACK_NAME:-${COMPOSE_PROJECT_NAME:-stuhelper}}"
BOOTSTRAP_VOLUME="${BOOTSTRAP_VOLUME:-${STACK_NAME_VALUE}-zitadel-bootstrap}"

# OIDC Application 配置
APP_NAME="stuhelper-web"
PROJECT_NAME="hangxiaoban"
REDIRECT_URIS="${REDIRECT_URIS:-http://localhost:8080/api/v1/auth/callback}"
POST_LOGOUT_URIS="${POST_LOGOUT_URIS:-http://localhost:3000}"
BOOTSTRAP_ORG_NAME="${BOOTSTRAP_ORG_NAME:-stuhelper-platform}"
ADMIN_LOGIN_NAME="${ADMIN_LOGIN_NAME:-admin@${BOOTSTRAP_ORG_NAME}.${ZITADEL_DOMAIN:-localhost}}"

# Project Roles
ROLES=("super_admin" "school_admin" "moderator" "verified_student" "user")
ROLE_DISPLAY_NAMES=("超级管理员" "学校管理员" "内容管理员" "已认证学生" "普通用户")

# ── 获取 PAT ─────────────────────────────────────────────
if [[ -n "$PAT_FILE" && -f "$PAT_FILE" ]]; then
  PAT=$(cat "$PAT_FILE")
else
  # 优先读取当前运行环境的 bootstrap volume，避免复用 .env.generated 中陈旧 PAT。
  PAT=$(docker run --rm -v "${BOOTSTRAP_VOLUME}:/bootstrap:ro" alpine:3.22 cat /bootstrap/admin.pat 2>/dev/null || true)
  if [[ -z "$PAT" ]]; then
    PAT="${ZITADEL_MANAGEMENT_PAT:-}"
  fi
fi

if [[ -z "$PAT" ]]; then
  echo "ERROR: 无法获取 Admin PAT。请设置 ZITADEL_MANAGEMENT_PAT 或 PAT_FILE 环境变量。" >&2
  exit 1
fi

AUTH_HEADER="Authorization: Bearer $PAT"

# ── 工具函数 ─────────────────────────────────────────────
zitadel_api() {
  local method="$1" path="$2" body="${3:-}"
  local url="${ZITADEL_URL}${path}"

  if [[ -n "$body" ]]; then
    curl -sS -X "$method" "$url" \
      -H "$AUTH_HEADER" \
      -H "Content-Type: application/json" \
      -d "$body"
  else
    curl -sS -X "$method" "$url" \
      -H "$AUTH_HEADER" \
      -H "Content-Type: application/json"
  fi
}

zitadel_connect() {
  local service_method="$1" body="${2:-}"
  local url="${ZITADEL_URL}/${service_method}"

  if [[ -z "$body" ]]; then
    body='{}'
  fi

  curl -sS -X POST "$url" \
    -H "$AUTH_HEADER" \
    -H "Connect-Protocol-Version: 1" \
    -H "Content-Type: application/json" \
    -d "$body"
}

wait_for_zitadel() {
  echo "等待 Zitadel API 就绪..."
  local retries=30
  while ! curl -sf "${ZITADEL_URL}/debug/healthz" >/dev/null 2>&1; do
    retries=$((retries - 1))
    if [[ $retries -le 0 ]]; then
      echo "ERROR: Zitadel API 未在超时内就绪" >&2
      exit 1
    fi
    sleep 2
  done
  echo "Zitadel API 已就绪"
}

wait_for_management_pat() {
  echo "等待 Zitadel Management PAT 生效..."
  local retries=120
  local response
  while true; do
    response=$(curl -sS -o /dev/null -w '%{http_code}' \
      -X POST "${ZITADEL_URL}/management/v1/projects/_search" \
      -H "$AUTH_HEADER" \
      -H "Content-Type: application/json" \
      -d '{}' || true)
    if [[ "$response" == "200" ]]; then
      echo "Zitadel Management PAT 已就绪"
      return 0
    fi
    retries=$((retries - 1))
    if [[ $retries -le 0 ]]; then
      echo "ERROR: Zitadel Management PAT 未在超时内生效（最近 HTTP 状态: ${response:-n/a}）" >&2
      exit 1
    fi
    sleep 2
  done
}

# ── 主流程 ─────────────────────────────────────────────
wait_for_zitadel
wait_for_management_pat

# 1. 创建 Project
echo "创建 Project: ${PROJECT_NAME}..."
PROJECT_RESPONSE=$(zitadel_api POST "/management/v1/projects" \
  "{\"name\": \"${PROJECT_NAME}\", \"projectRoleAssertion\": true, \"projectRoleCheck\": true}")
PROJECT_ID=$(echo "$PROJECT_RESPONSE" | jq -r '.id // empty')

if [[ -z "$PROJECT_ID" ]]; then
  # Project 可能已存在，尝试查找
  echo "Project 创建失败或已存在，尝试查找..."
  SEARCH_RESPONSE=$(zitadel_api POST "/management/v1/projects/_search" \
    "{\"queries\": [{\"nameQuery\": {\"name\": \"${PROJECT_NAME}\", \"method\": \"TEXT_QUERY_METHOD_EQUALS\"}}]}")
  PROJECT_ID=$(echo "$SEARCH_RESPONSE" | jq -r '.result[0].id // empty')
  if [[ -z "$PROJECT_ID" ]]; then
    echo "ERROR: 无法创建或找到 Project ${PROJECT_NAME}" >&2
    echo "API 响应: $PROJECT_RESPONSE" >&2
    exit 1
  fi
  echo "找到已有 Project: ${PROJECT_ID}"
else
  echo "Project 创建成功: ${PROJECT_ID}"
fi

# 2. 创建 Project Roles
echo "创建 Project Roles..."
for i in "${!ROLES[@]}"; do
  ROLE_KEY="${ROLES[$i]}"
  ROLE_DISPLAY="${ROLE_DISPLAY_NAMES[$i]}"
  ROLE_RESPONSE=$(zitadel_api POST "/management/v1/projects/${PROJECT_ID}/roles" \
    "{\"roleKey\": \"${ROLE_KEY}\", \"displayName\": \"${ROLE_DISPLAY}\"}" 2>&1 || true)
  if echo "$ROLE_RESPONSE" | jq -e '.code' >/dev/null 2>&1; then
    ERROR_CODE=$(echo "$ROLE_RESPONSE" | jq -r '.code')
    if [[ "$ERROR_CODE" == "6" ]]; then
      echo "  角色 ${ROLE_KEY} 已存在，跳过"
    else
      echo "  警告: 创建角色 ${ROLE_KEY} 失败: ${ROLE_RESPONSE}"
    fi
  else
    echo "  角色 ${ROLE_KEY} 创建成功"
  fi
done

# 3. 创建 OIDC Application
echo "创建 OIDC Application: ${APP_NAME}..."

# 构建 redirect URIs JSON 数组
IFS=',' read -ra REDIRECT_ARRAY <<< "$REDIRECT_URIS"
REDIRECT_JSON=$(printf '%s\n' "${REDIRECT_ARRAY[@]}" | jq -R . | jq -s .)

IFS=',' read -ra LOGOUT_ARRAY <<< "$POST_LOGOUT_URIS"
LOGOUT_JSON=$(printf '%s\n' "${LOGOUT_ARRAY[@]}" | jq -R . | jq -s .)

APP_BODY=$(jq -n \
  --arg name "$APP_NAME" \
  --argjson redirectUris "$REDIRECT_JSON" \
  --argjson postLogoutRedirectUris "$LOGOUT_JSON" \
  '{
    "name": $name,
    "redirectUris": $redirectUris,
    "responseTypes": ["OIDC_RESPONSE_TYPE_CODE"],
    "grantTypes": ["OIDC_GRANT_TYPE_AUTHORIZATION_CODE", "OIDC_GRANT_TYPE_REFRESH_TOKEN"],
    "appType": "OIDC_APP_TYPE_WEB",
    "authMethodType": "OIDC_AUTH_METHOD_TYPE_BASIC",
    "postLogoutRedirectUris": $postLogoutRedirectUris,
    "devMode": true,
    "accessTokenType": "OIDC_TOKEN_TYPE_JWT",
    "idTokenRoleAssertion": true,
    "idTokenUserinfoAssertion": true
  }')

APP_RESPONSE=$(zitadel_api POST "/management/v1/projects/${PROJECT_ID}/apps/oidc" "$APP_BODY")
CLIENT_ID=$(echo "$APP_RESPONSE" | jq -r '.clientId // empty')
CLIENT_SECRET=$(echo "$APP_RESPONSE" | jq -r '.clientSecret // empty')
APP_ID=$(echo "$APP_RESPONSE" | jq -r '.appId // .id // empty')

if [[ -z "$CLIENT_ID" ]]; then
  # Application 可能已存在
  echo "OIDC App 创建失败或已存在，尝试查找..."
  APPS_RESPONSE=$(zitadel_api POST "/management/v1/projects/${PROJECT_ID}/apps/_search" "{}")
  CLIENT_ID=$(echo "$APPS_RESPONSE" | jq -r --arg name "$APP_NAME" 'first(.result[]? | select(.name == $name) | .oidcConfig.clientId) // empty')
  APP_ID=$(echo "$APPS_RESPONSE" | jq -r --arg name "$APP_NAME" 'first(.result[]? | select(.name == $name) | .id) // empty')
  if [[ -n "$CLIENT_ID" && -n "$APP_ID" ]]; then
    echo "找到已有 OIDC App: ${CLIENT_ID}"
    echo "为已有 OIDC App 重新生成客户端密钥..."
    SECRET_BODY=$(jq -cn --arg projectId "$PROJECT_ID" --arg applicationId "$APP_ID" \
      '{projectId: $projectId, applicationId: $applicationId}')
    SECRET_RESPONSE=$(zitadel_connect "zitadel.application.v2.ApplicationService/GenerateClientSecret" "$SECRET_BODY")
    CLIENT_SECRET=$(echo "$SECRET_RESPONSE" | jq -r '.clientSecret // empty')
    if [[ -z "$CLIENT_SECRET" ]]; then
      echo "ERROR: 无法为已有 OIDC App 生成 client secret" >&2
      echo "API 响应: $SECRET_RESPONSE" >&2
      exit 1
    fi
  else
    echo "ERROR: 无法创建或找到 OIDC App ${APP_NAME}" >&2
    echo "API 响应: $APP_RESPONSE" >&2
    exit 1
  fi
else
  echo "OIDC App 创建成功"
fi

# 4. 为默认管理员授予 Project Role，避免 OIDC 登录时报 GrantRequired
echo "为默认管理员授予 Project Role..."
ADMIN_SEARCH_RESPONSE=$(zitadel_api POST "/management/v1/users/_search" \
  "{\"queries\": [{\"loginNameQuery\": {\"loginName\": \"${ADMIN_LOGIN_NAME}\", \"method\": \"TEXT_QUERY_METHOD_EQUALS\"}}]}")
ADMIN_USER_ID=$(echo "$ADMIN_SEARCH_RESPONSE" | jq -r '.result[0].id // empty')

if [[ -z "$ADMIN_USER_ID" ]]; then
  echo "ERROR: 无法找到默认管理员 ${ADMIN_LOGIN_NAME}" >&2
  echo "API 响应: $ADMIN_SEARCH_RESPONSE" >&2
  exit 1
fi

ADMIN_GRANT_BODY=$(jq -n \
  --arg projectId "$PROJECT_ID" \
  '{projectId: $projectId, roleKeys: ["super_admin"]}')
ADMIN_GRANT_RESPONSE=$(zitadel_api POST "/management/v1/users/${ADMIN_USER_ID}/grants" "$ADMIN_GRANT_BODY" 2>&1 || true)

if echo "$ADMIN_GRANT_RESPONSE" | jq -e '.code' >/dev/null 2>&1; then
  ERROR_CODE=$(echo "$ADMIN_GRANT_RESPONSE" | jq -r '.code')
  if [[ "$ERROR_CODE" == "6" ]]; then
    echo "  默认管理员已具备 super_admin 角色，跳过"
  else
    echo "  警告: 默认管理员授权失败: ${ADMIN_GRANT_RESPONSE}"
  fi
else
  echo "  默认管理员已授予 super_admin 角色"
fi

# ── 输出 ─────────────────────────────────────────────────
echo ""
echo "=========================================="
echo "  Zitadel 初始化完成"
echo "=========================================="
echo ""
echo "请将以下配置添加到 .env 文件:"
echo ""
echo "ZITADEL_PROJECT_ID=${PROJECT_ID}"
echo "ZITADEL_CLIENT_ID=${CLIENT_ID}"
echo "ZITADEL_CLIENT_SECRET=${CLIENT_SECRET}"
echo ""
echo "Zitadel Console: ${ZITADEL_URL}/ui/console"
echo "登录账号: ${ADMIN_LOGIN_NAME} / ${ZITADEL_ADMIN_PASSWORD:-<configured externally>}"
