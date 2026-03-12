# Casdoor SSO 集成指南

> 状态：已实现

StuHelper 当前使用 Casdoor 处理登录、注册、单点登录和 Refresh Token 续期。浏览器端已经切到 Cookie 会话模型，不再把 access token 暴露给业务代码。

## 当前认证流程

```text
1. 前端调用 /api/v1/auth/login 或 /api/v1/auth/signup
2. 后端生成 Casdoor 授权地址和随机 state
3. 浏览器跳转到 https://sso.stuhelper.com
4. Casdoor 登录完成后回跳到前端 /auth/callback
5. 前端回调页调用 /api/v1/auth/callback
6. 后端使用授权码向 Casdoor 换取 access token / refresh token
7. 后端把 token 写入 HttpOnly Cookie，并返回当前用户信息
8. 前端保存最小用户信息和过期时间，用于路由预检
```

关键点：

- OAuth 回调地址现在是前端页面，如 `http://localhost:3000/auth/callback`。
- 真正的 token 交换发生在后端 `/api/v1/auth/callback`。
- Access Token 和 Refresh Token 都通过 Cookie 管理。

## 环境变量

### 本地开发

```bash
CASDOOR_ENDPOINT=https://sso.stuhelper.com
CASDOOR_CLIENT_ID=<your-client-id>
CASDOOR_CLIENT_SECRET=<your-client-secret>
CASDOOR_ORGANIZATION=stuhelper
CASDOOR_APPLICATION=stuhelper
CASDOOR_REDIRECT_URI=http://localhost:3000/auth/callback
CASDOOR_CERTIFICATE=<pem-content-or-empty>

TOKEN_ACCESS_TTL=900
TOKEN_REFRESH_TTL=604800
TOKEN_COOKIE_SECURE=false
```

### 生产环境

```bash
CASDOOR_REDIRECT_URI=https://stuhelper.com/auth/callback
TOKEN_COOKIE_SECURE=true
```

## 后端接口

| 方法 | 路径                      | 说明                    | 认证 |
| ---- | ------------------------- | ----------------------- | ---- |
| GET  | `/api/v1/auth/login`      | 获取登录 URL            | 否   |
| GET  | `/api/v1/auth/signup`     | 获取注册 URL            | 否   |
| GET  | `/api/v1/auth/callback`   | OAuth 回调换取本地会话  | 否   |
| POST | `/api/v1/auth/refresh`    | 使用 Refresh Token 续期 | 否   |
| GET  | `/api/v1/auth/me`         | 获取当前用户            | 是   |
| POST | `/api/v1/auth/logout`     | 登出当前设备            | 是   |
| POST | `/api/v1/auth/logout-all` | 登出所有设备            | 是   |

## 前端集成方式

### Web 端

Web 端 API 客户端位于 `clients/web/src/api/client.ts`，已经做了这些事情：

1. 所有请求自动 `credentials: 'include'`
2. 变更型请求自动附带 `X-CSRF-Token`
3. 收到 `401` 时自动尝试调用 `/api/v1/auth/refresh`
4. 刷新失败就清理本地会话

登录页与回调页还会保留 `post_login_redirect`，确保登录后能回跳到原页面。

### 路由守卫

需要登录的页面加：

```typescript
meta: {
	requiresAuth: true;
}
```

管理员页面再加：

```typescript
meta: {
	requiresAdmin: true;
}
```

## Refresh Token 与登出

Casdoor 官方文档支持标准的 Refresh Token 流程和登出流程：

- [Refresh token](https://casdoor.org/docs/basic/server-side-auth/token/#refresh-token)
- [Logout](https://casdoor.org/docs/basic/server-side-auth/token/#logout)

StuHelper 当前实现对应关系如下：

- `/api/v1/auth/refresh` 依赖浏览器中的 refresh token Cookie 完成续期。
- `/api/v1/auth/logout` 会清理本地会话，并返回 `ssoLogoutURL`。
- 前端访问 `ssoLogoutURL` 时使用顶级导航或弹窗，确保 Casdoor 的浏览器会话也一起失效。

## 常见排查点

### 登录后回调失败

- 检查 `CASDOOR_REDIRECT_URI` 是否与 Casdoor 应用配置一致。
- 检查浏览器地址是否为 `/auth/callback?code=...&state=...`。
- 检查前端和后端是否都连接到了同一个 Casdoor 应用。

### 会话刚登录就过期

- 检查 `TOKEN_ACCESS_TTL`、`TOKEN_REFRESH_TTL`。
- 检查浏览器是否接受了 Cookie。
- 检查是否有反向代理错误地剥离了 Cookie 或 `Set-Cookie` 头。
