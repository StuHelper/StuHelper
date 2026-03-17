# Casdoor SSO 集成

登录流程通过 Casdoor 完成。后端发起 OAuth 流程，交换令牌，写入 Cookie，向前端返回用户信息。

## API 端点

| 端点 | 方法 | 用途 |
| --- | --- | --- |
| `/api/v1/auth/login` | GET | 生成登录 URL 和随机 `state` |
| `/api/v1/auth/signup` | GET | 生成注册 URL 和随机 `state` |
| `/api/v1/auth/callback` | GET | 用授权码交换令牌，写入 Cookie，返回 `user` 和 `expiresIn` |
| `/api/v1/auth/me` | GET | 返回当前用户信息和能力集 |
| `/api/v1/auth/refresh` | POST | 用 refresh token 刷新 access token |

## 登录流程

1. 前端请求 `GET /api/v1/auth/login`（或 `/signup`）
2. 后端通过 `sso.Client.GetSigninURL` 生成 Casdoor 授权 URL，同时在 Redis 中存储随机 state（有效期 5 分钟）
3. 浏览器跳转到 `https://sso.stuhelper.com`
4. Casdoor 回调，浏览器带 `code` 和 `state` 重定向到前端 `/auth/callback`
5. 前端调用 `GET /api/v1/auth/callback?code=xxx&state=xxx`
6. 后端验证并消费 state（Lua 原子 GET+DEL），用 code 向 Casdoor 换取 access token 和 refresh token
7. 后端解析 JWT，校验 `Owner` 是否匹配本应用组织
8. 后端调用 `UpsertUser` 同步用户到本地 `users` 表
9. 后端将令牌写入 HttpOnly Cookie，返回用户信息

## UserInfo 响应字段

回调和 `/auth/me` 都通过 `buildUserInfo` 组装，返回以下字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | string | Casdoor 用户 ID（`external_id`） |
| `name` | string | 用户名 |
| `displayName` | string | 显示名称 |
| `email` | string | 邮箱地址 |
| `avatar` | string/null | 头像 URL |
| `isPlatformAdmin` | boolean | Casdoor 平台管理员标记 |
| `capabilities` | string[] | 应用能力集（去重排序后） |
| `canAccessAdmin` | boolean | 是否持有至少一个管理端入口能力 |

## 代码路径

| 文件 | 职责 |
| --- | --- |
| `server/internal/modules/auth/handler_login.go` | 登录 URL 生成、OAuth 回调处理、Cookie 写入 |
| `server/internal/modules/auth/handler_userinfo.go` | `/auth/me` 处理器和 `buildUserInfo` 组装逻辑 |
| `server/internal/modules/auth/user_sync.go` | `UpsertUser` 同步用户到 `users` 表，接口定义 |
| `server/internal/pkg/sso/client.go` | OAuth URL 生成、state 管理、令牌交换、JWT 解析、用户查询 |

## 平台管理员语义

`isPlatformAdmin` 来自 Casdoor 用户的 `IsAdmin` 字段，表示该用户是 Casdoor 平台管理员。管理菜单和管理端点的访问控制由 `capabilities` 驱动，前端和后端使用相同的能力常量。`isPlatformAdmin` 和应用能力是两套独立的权限体系。
