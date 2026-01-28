# Casdoor SSO 集成指南

> 状态：🟢 已实现

本文档介绍 StuHelper 项目的 Casdoor OAuth2/OIDC 单点登录系统的配置和使用方法。

## 认证流程

```
1. 用户点击登录按钮
2. 前端调用 /api/v1/auth/login 获取 SSO 登录 URL
3. 前端跳转到 SSO 登录页面
4. 用户在 SSO 系统完成登录
5. SSO 系统重定向回应用的回调地址，携带授权码
6. 前端调用 /api/v1/auth/callback 处理回调
7. 后端使用授权码换取访问令牌和用户信息
8. 后端生成 JWT token 返回给前端
9. 前端保存 token，完成登录
```

## SSO 配置

### 1. 在 SSO 管理后台创建应用

访问 https://sso.stuhelper.com 管理后台：

1. 创建组织（Organization）：`stuhelper`
2. 创建应用（Application）：`stuhelper-main`
3. 配置回调地址：`http://localhost:8080/api/v1/auth/callback`
4. 获取 Client ID 和 Client Secret

### 2. 配置环境变量

```bash
cd server/deployments
cp .env.example .env
```

编辑 `.env` 文件：

```bash
# Casdoor SSO 配置
CASDOOR_ENDPOINT=https://sso.stuhelper.com
CASDOOR_CLIENT_ID=your_client_id_here
CASDOOR_CLIENT_SECRET=your_client_secret_here
CASDOOR_ORGANIZATION=stuhelper
CASDOOR_APPLICATION=stuhelper-main
CASDOOR_REDIRECT_URI=http://localhost:8080/api/v1/auth/callback
CASDOOR_CERTIFICATE=your_certificate_here

# Token 配置
TOKEN_ACCESS_TTL=7200
TOKEN_REFRESH_TTL=604800
TOKEN_COOKIE_DOMAIN=localhost
TOKEN_COOKIE_SECURE=false
```

## 后端 API

### 认证接口列表

| 方法 | 路径 | 说明 | 认证 |
|------|------|------|------|
| GET | `/api/v1/auth/login` | 获取登录 URL | 否 |
| GET | `/api/v1/auth/signup` | 获取注册 URL | 否 |
| GET | `/api/v1/auth/callback` | 处理 OAuth 回调 | 否 |
| POST | `/api/v1/auth/refresh` | 刷新 Access Token | 否 |
| GET | `/api/v1/auth/me` | 获取当前用户信息 | 是 |
| POST | `/api/v1/auth/logout` | 登出当前设备 | 是 |
| POST | `/api/v1/auth/logout-all` | 登出所有设备 | 是 |

### 接口详情

#### 获取登录 URL

```
GET /api/v1/auth/login
```

响应：
```json
{
  "success": true,
  "data": {
    "url": "https://sso.stuhelper.com/login/oauth/authorize?...",
    "state": "random_state_string"
  }
}
```

#### 处理 OAuth 回调

```
GET /api/v1/auth/callback?code=xxx&state=xxx
```

响应：
```json
{
  "success": true,
  "data": {
    "user": {
      "id": "user_id",
      "name": "username",
      "display_name": "显示名称",
      "email": "user@example.com",
      "avatar": "avatar_url"
    }
  }
}
```

> **注意**：Access Token 和 Refresh Token 通过 HttpOnly Cookie 设置，不在响应体中返回。

#### 刷新 Token

```
POST /api/v1/auth/refresh
```

响应：
```json
{
  "success": true,
  "data": {
    "message": "token refreshed successfully"
  }
}
```

#### 获取当前用户信息

```
GET /api/v1/auth/me
Cookie: access_token=<token>
```

响应：
```json
{
  "success": true,
  "data": {
    "id": "user_id",
    "name": "username",
    "email": "user@example.com",
    "display_name": "显示名称"
  }
}
```

#### 登出

```
POST /api/v1/auth/logout
Cookie: access_token=<token>
```

响应：
```json
{
  "success": true,
  "data": {
    "message": "logout successful"
  }
}
```

## 前端集成

### 使用 Auth Store

```typescript
import { useAuthStore } from '@/stores/auth'

const authStore = useAuthStore()

// 登录
await authStore.login()

// 登出
await authStore.logout()

// 检查登录状态
if (authStore.isAuthenticated) {
  console.log('已登录')
}
```

### 路由守卫

需要登录的页面添加 `meta.requiresAuth`：

```typescript
{
  path: '/profile',
  name: 'profile',
  component: () => import('@/views/ProfilePage.vue'),
  meta: { requiresAuth: true }
}
```

## 代码结构

```
server/internal/
├── modules/auth/
│   └── handler.go          # 认证 API 处理器
└── pkg/
    ├── middleware/
    │   └── auth.go         # 认证中间件
    ├── sso/
    │   ├── client.go       # SSO 客户端
    │   └── state.go        # State 管理
    └── token/
        ├── blacklist.go    # Token 黑名单
        └── service.go      # Token 服务
```
