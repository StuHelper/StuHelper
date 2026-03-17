# 会话与安全

安全层覆盖 Cookie 会话、令牌黑名单、PII 加密和审计日志。

## 敏感数据保护

| 数据 | 存储方式 |
| --- | --- |
| 证件号码 | AES-256-GCM 加密存储在 `user_identities.doc_number_enc`（BYTEA） |
| 证件派生标识 | HMAC-SHA256 派生后存储在 `user_identities.person_uid` |
| 学号 | 明文存储在 `user_profiles.student_ids`（JSON 数组）和 `active_student_id` |
| 手机号 | 明文存储在 `user_profiles.phone` |

## PII 加密格式

证件号码使用版本化信封格式加密：

```text
version(1 字节) | keyID(1 字节) | nonce(12 字节) | ciphertext + GCM tag
```

- `version`：当前固定为 `0x01`
- `keyID`：标识使用的加密密钥（支持 key rotation）
- `nonce`：每次加密随机生成（`crypto/rand`），同一明文多次加密结果不同
- `ciphertext + tag`：AES-256-GCM 密文和认证标签

`person_uid` 通过 `HMAC-SHA256(doc_type + ":" + doc_number)` 计算，用于跨记录匹配同一自然人，同时避免暴露原始证件号。

## 环境变量配置

```bash
DOC_AES_ACTIVE_KEY_ID=1
DOC_AES_KEYS=1:<64-char-hex-key>
HMAC_SECRET=<至少 32 个字符>
```

- `DOC_AES_KEYS` 支持多密钥格式（如 `1:<key1>,2:<key2>`），实现密钥轮换
- `HMAC_SECRET` 生产环境必须配置，开发环境未配置时自动生成随机密钥

## 会话架构

| 组件 | 实现方式 |
| --- | --- |
| Access Token | HttpOnly Cookie（`access_token`），默认 15 分钟，Path `/` |
| Refresh Token | HttpOnly Cookie（`refresh_token`），默认 7 天，Path 限定 `/api/v1/auth/refresh` |
| CSRF 防护 | `csrf_token` 普通 Cookie（HttpOnly=false），前端在请求头中回传 |
| 会话撤销 | Redis 黑名单（`token:blacklist:` 前缀）+ 用户令牌集合追踪（`token:user:` 前缀） |

Cookie 安全属性：`SameSite=Strict`，`Secure` 通过 `TOKEN_COOKIE_SECURE` 环境变量控制。

黑名单服务内置熔断器（5 次失败阈值、30 秒超时）和本地缓存降级机制。熔断器打开或 Redis 不可用时，已缓存的黑名单记录仍然生效；无缓存时采用 fail-closed 策略拒绝请求。

## 令牌刷新流程

1. Access token 过期（15 分钟）
2. 前端收到 401 响应
3. 前端调用 `POST /api/v1/auth/refresh`（限流：每分钟 10 次）
4. 后端校验 refresh token 是否在黑名单中
5. 后端将旧 refresh token 加入黑名单
6. 后端向 Casdoor 请求新令牌对
7. 后端写入新的 HttpOnly Cookie
8. 后端更新令牌追踪集合（移除旧令牌哈希，添加新令牌哈希）
9. 前端重试原始请求

刷新失败（令牌过期或被撤销）时，后端清除 Cookie，前端清除本地会话并跳转登录页。

## 审计事件

`server/internal/pkg/audit` 通过结构化日志记录认证和关键业务事件：

| 事件 | 说明 |
| --- | --- |
| `user.login` | 登录成功 |
| `user.login_failed` | 登录失败（无效 state、OAuth 错误、JWT 解析失败、组织不匹配） |
| `user.logout` | 单设备登出 |
| `user.logout_all` | 全设备登出 |
| `token.refresh` | 令牌刷新 |
| `token.revoked` | 令牌被撤销 |

审计日志包含 `user_id`、`username`（脱敏）、`ip`（脱敏）、`user_agent`、`request_id` 等字段。

## 安全响应头

### 所有环境

- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `Referrer-Policy: strict-origin-when-cross-origin`
- `X-Permitted-Cross-Domain-Policies: none`
- `Content-Security-Policy: default-src 'none'; frame-ancestors 'none'`（API 路径）

### 生产环境额外添加

- `Strict-Transport-Security: max-age=31536000; includeSubDomains`
- `Permissions-Policy: geolocation=(), microphone=(), camera=()`
- `Cross-Origin-Resource-Policy: same-origin`
- `Cross-Origin-Opener-Policy: same-origin`
