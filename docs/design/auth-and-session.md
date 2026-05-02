---
type: design
audience: backend-dev, frontend-dev
status: current
authoritative-source: server/internal/modules/auth/
last-verified: 2026-05-02
---

# 认证与 SSO

> 状态：现行。默认支持 Casdoor OIDC 登录；手机号验证码登录仅在 `SMS_ENABLED=true` 且短信凭据完整时启用。

## 登录方式

### Casdoor OIDC（主要链路）

```
前端请求 /api/v1/auth/login?redirect=...
→ 后端生成 state，写 Redis + HttpOnly state cookie
→ 跳转 Casdoor
→ 回调 /api/v1/auth/callback?code=...&state=...
→ 校验 state（一次性验证后销毁）→ 交换 token → 写入 Cookie → 同步 shadow user
→ 302 回前端
→ 前端请求 /api/v1/auth/me
```

### 手机号验证码（补充）

```
POST /api/v1/auth/phone/request-otp → 发送验证码
POST /api/v1/auth/phone/verify-otp  → 验证 → 签发本地 JWT Cookie
```

- 仅限中国大陆手机号
- 有冷却和频率限制
- 只授予 `user` 角色，管理角色仍需 Casdoor SSO
- 默认关闭；需要同时配置 `SMS_ENABLED=true`、`SMS_SECRET_ID`、`SMS_SECRET_KEY`、`SMS_APP_ID`、`SMS_SIGN_NAME`、`SMS_TEMPLATE_ID`、`SMS_INTERNAL_KEY`

## 端点

| 路径 | 方法 | 说明 |
|------|------|------|
| `/api/v1/auth/login` | GET | 获取登录跳转地址 |
| `/api/v1/auth/signup` | GET | 获取注册跳转地址 |
| `/api/v1/auth/callback` | GET | OIDC 回调 |
| `/api/v1/auth/phone/request-otp` | POST | 发送验证码 |
| `/api/v1/auth/phone/verify-otp` | POST | 验证码登录 |
| `/api/v1/auth/refresh` | POST | 续期会话 |
| `/api/v1/auth/me` | GET | 当前用户 + 角色 + 能力 |
| `/api/v1/auth/logout` | POST | 登出当前设备 |
| `/api/v1/auth/logout-all` | POST | 全设备登出 |
| `/api/v1/auth/exchange-native` | POST | native app 令牌交换（deep-link 回调后用 code+state 换取 token pair） |

## 会话

| 令牌 | TTL | 存储 | 用途 |
|------|-----|------|------|
| Access Token | 5 分钟（默认 300 s） | HttpOnly Cookie | API 访问 |
| Refresh Token | 7 天 | Path `/api/v1/auth` HttpOnly Cookie | 续期 |
| CSRF Token | 随 refresh | 普通 Cookie | 写请求防 CSRF |

access / refresh token 区分 `typ`，refresh 不会被当作 access 验证。

### Refresh 行为

- **浏览器**：`POST /api/v1/auth/refresh`
  - refresh token 来自 HttpOnly cookie
  - 必须同时携带 CSRF header + CSRF cookie
  - 响应体只返回 `message` / `expiresIn`，新 token 通过 cookie 下发
- **原生 App**：`POST /api/v1/auth/refresh`
  - 请求体传 `{ "refreshToken": "..." }`
  - 不走 cookie / 不做 CSRF 校验
  - 响应体返回 `accessToken`、`refreshToken`、`expiresIn`
- OIDC refresh token 由 StuHelper session store 代持：
  - `oidc` / `oidc-native` session 会把 provider refresh token 加密后写入 Redis session；
  - refresh 轮换时先吊销旧 provider refresh token，再保存新 provider refresh token；
  - `logout` / `logout-all` 必须先调用 Casdoor revocation endpoint 吊销 provider refresh token，失败时返回错误，不清理本地 session 假装成功；
  - `phone` 登录使用 StuHelper 自签 refresh token，不进入 provider revoke 流程。

### 浏览器 access token 校验模型

- 浏览器 Cookie access token 走 **本地 JWKS 验证**，不做每请求 session store lookup
- 即时吊销依赖 Redis blacklist；自然过期依赖 5 分钟 `TOKEN_ACCESS_TTL`
- `refresh` / `logout` / `logout-all` 仍会命中 session store 做轮换或撤销
- 这是当前有意的性能/安全边界：不把浏览器读请求重新拉回每请求 Redis RTT

### Native exchange / refresh 当前口径

- `POST /api/v1/auth/exchange-native` 请求体：`{ code, state }`
- 成功响应：`{ accessToken, refreshToken, sessionID, expiresIn }`
- 原生 OIDC refresh 必须通过 `X-Stuhelper-Session-ID` 回传 `sessionID`；缺失或不匹配时拒绝 refresh。
- refresh 会对旧 refresh token 做 blacklist，并在 session store 内更新新 token hash 和加密后的 provider refresh token。
- 旧 refresh token 再次提交会触发 reuse detection：吊销该用户全部 session 并记录审计。

## Shadow User

OIDC 用户同步到本地 `users` 表：`casdoor_subject`、`username`、`email`、`avatar_url`。

手机号登录通过 `UpsertByPhone` 处理，保证业务侧始终有稳定锚点。

## 角色与能力

- Casdoor JWT 提供粗粒度角色
- 中间件静态展开为 capabilities
- `/api/v1/auth/me` 返回：`roles`、`capabilities`、`globalCapabilities`、`canAccessAdmin`、`displayName`、`isPlatformAdmin`、`capabilityGrants`、`accountSettingsUrl`

## Native App OIDC 登录

原生客户端（iOS / Android / uni-app）无法直接接收 HttpOnly Cookie，因此使用独立的令牌交换端点：

1. 前端发起 `/api/v1/auth/login?platform=native&redirect=...`
2. 后端生成 state + PKCE code_verifier，state 标记 `native=true`
3. 用户在系统浏览器完成 Casdoor 登录
4. 回调时后端识别 native state，将 `code` + `state` 通过 deep link（`stuhelper://auth/callback?code=...&state=...`）回传给 App
5. App 调用 `POST /api/v1/auth/exchange-native`（body: `{code, state}`），后端用保存的 code_verifier 完成 OIDC 交换，返回 JSON token pair + `sessionID`

## 代码入口

| 组件 | 位置 |
|------|------|
| Auth Handler | `server/internal/modules/auth/` |
| OIDC 客户端 | `server/internal/pkg/oidc/` |
| Token 服务 | `server/internal/pkg/token/` |
| Provider refresh token revoke | `server/internal/modules/auth/service_provider_tokens.go` + `server/internal/pkg/oidc/revoke.go` |
| 用户同步 | `server/internal/modules/auth/user_sync.go` |
