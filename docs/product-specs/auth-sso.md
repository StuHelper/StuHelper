# 认证与 SSO

> 状态：现行。支持 Zitadel OIDC 登录和手机号验证码登录。

## 登录方式

### Zitadel OIDC（主要链路）

```
前端请求 /api/v1/auth/login?redirect=...
→ 后端生成 state，写 Redis + HttpOnly state cookie
→ 跳转 Zitadel
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
- 只授予 `user` 角色，管理角色仍需 Zitadel SSO

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

## 会话

| 令牌 | TTL | 存储 | 用途 |
|------|-----|------|------|
| Access Token | 5 分钟 | HttpOnly Cookie | API 访问 |
| Refresh Token | 7 天 | Path 受限 HttpOnly Cookie | 续期 |
| CSRF Token | 随 refresh | 普通 Cookie | 写请求防 CSRF |

access / refresh token 区分 `typ`，refresh 不会被当作 access 验证。

## Shadow User

OIDC 用户同步到本地 `users` 表：`external_id`、`username`、`email`、`avatar_url`。

手机号登录通过 `UpsertByPhone` 处理，保证业务侧始终有稳定锚点。

## 角色与能力

- Zitadel Token 提供粗粒度角色
- 中间件静态展开为 capabilities
- `/api/v1/auth/me` 返回：`roles`、`capabilities`、`globalCapabilities`、`canAccessAdmin`

## 代码入口

| 组件 | 位置 |
|------|------|
| Auth Handler | `server/internal/modules/auth/` |
| OIDC 客户端 | `server/internal/pkg/oidc/` |
| Token 服务 | `server/internal/pkg/token/` |
| 用户同步 | `server/internal/modules/auth/user_sync.go` |
