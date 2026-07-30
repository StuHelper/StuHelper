---
type: design
audience: backend-dev, frontend-dev
status: current
authoritative-source: server/internal/modules/auth/
last-verified: 2026-07-31
---

# 认证与 SSO

> 状态：现行。公开登录认证入口和 OIDC issuer 是 `sso.stuhelper.com` 的 Casdoor。`stuhelper.com` 承载账号中心、学生认证、QQ 绑定和开放平台业务页面；`join.stuhelper.com` 承载入群验证业务闭环。StuHelper 不再发布独立身份入口域名。

## 登录方式

### StuHelper Web + Casdoor（主要链路）

```
业务页面访问 stuhelper.com、join.stuhelper.com 或受保护页面
→ 前端进入 /login?redirect=<受保护目标>
→ LoginPage 调用 /api/v1/auth/login?redirect=<受保护目标>
→ 后端生成 upstream state，写 Redis + HttpOnly state cookie
→ 跳转 https://sso.stuhelper.com/login/oauth/authorize
→ 回调 https://stuhelper.com/api/v1/auth/callback?code=...&state=...
→ 校验 state（一次性验证后销毁）→ 交换 token → 写入 Cookie → 同步 shadow user
→ 302 回原业务目标页
→ 前端请求 /api/v1/auth/me
```

### 手机号验证码（Casdoor）

手机号注册、登录、绑定初始校验由 Casdoor 使用 SMS Provider 完成。StuHelper 不再暴露公开的 `/api/v1/auth/phone/*` 登录端点，也不签发独立的 phone-login 本地会话。

StuHelper 个人中心仍可以承载手机号补绑 / 更换 UI，但写路径调用 Casdoor user profile client；完整手机号真相源在 Casdoor。StuHelper 本地只保留业务展示和筛选需要的脱敏投影、验证状态与更新时间。

## 端点

| 路径 | 方法 | 说明 |
|------|------|------|
| `/api/v1/auth/login` | GET | 获取登录跳转地址 |
| `/api/v1/auth/signup` | GET | 获取注册跳转地址 |
| `/api/v1/auth/callback` | GET | OIDC 回调 |
| `/api/v1/auth/refresh` | POST | 续期会话 |
| `/api/v1/auth/me` | GET | 当前用户 + 角色 + 能力 |
| `/api/v1/auth/logout` | POST | 登出当前设备 |
| `/api/v1/auth/logout-all` | POST | 全设备登出 |
| `/api/v1/auth/exchange-native` | POST | native app 令牌交换（deep-link 回调后用 code+state 换取 token pair） |

## 会话

| 凭据 / 状态 | 有效期口径 | 存储 | 用途 |
|-------------|------------|------|------|
| Casdoor ID Token（StuHelper access credential） | provider `exp` 是自然失效真值；仓库托管的 Casdoor application 默认 1 小时 | HttpOnly Cookie 或 native 安全存储 | API 访问 |
| Access Cookie / `expiresIn` 策略 | `TOKEN_ACCESS_TTL` 默认 300 s；这是客户端刷新/持有策略，不会改写 provider `exp` | HttpOnly Cookie / 响应字段 | 缩短浏览器持有窗口、提示客户端刷新 |
| Provider Refresh Token | 仓库托管的 Casdoor application 默认 24 小时；本地 session / cookie lease 由 `TOKEN_REFRESH_TTL` 控制，默认 7 天 | Path `/api/v1/auth` HttpOnly Cookie；服务端 session 保存加密副本 | 续期 |
| CSRF Token | 随本地 refresh/session lease | 普通 Cookie | 写请求防 CSRF |

Casdoor token 通过 provider `tokenType` 区分 access / refresh；遗留 StuHelper 自签 token
通过 `typ` 区分，但已不属于公开登录链路。`TOKEN_ACCESS_TTL` 不能作为 provider token
黑名单 TTL，因为默认配置下它比已验证 ID token 的真实寿命短 55 分钟。

### Refresh 行为

- **浏览器**：`POST /api/v1/auth/refresh`
  - refresh token 来自 HttpOnly cookie
  - 必须同时携带 CSRF header + CSRF cookie
  - 响应体只返回 `message` / `expiresIn`，新 token 通过 cookie 下发
- **原生 App**：`POST /api/v1/auth/refresh`
  - 请求体传 `{ "refreshToken": "..." }`
  - 不走 cookie / 不做 CSRF 校验
  - 响应体返回 `accessToken`、`refreshToken`、`expiresIn`
- refresh token 来源必须唯一：原生 JSON body 与浏览器认证/session cookie 不能混用；请求体存在时必须是合法 JSON，不能静默回退到 cookie。
- session 定位来源必须唯一：`X-Stuhelper-Session-ID` 必须是单个非空 header，且不能与浏览器 `session_id` cookie 同时出现。
- OIDC refresh token 由 StuHelper session store 代持：
  - `oidc` / `oidc-native` session 会把 provider refresh token 加密后写入 Redis session；
  - refresh 在同一个 Redis Lua 操作中更新 access token hash、已验证 `exp`、refresh token hash 和 session lease，避免 hash 与 expiry 分离；
  - `logout` / `logout-all` 会撤销本地 session 和 token blacklist，并尝试执行 provider refresh-token 清理；当前固定 Casdoor 版本的 provider 撤销契约仍须按审计项 N-1 独立修复/验收，不能只因 HTTP 2xx 宣称 provider token family 已撤销；
  - 当前公开登录链路全部是 OIDC provider session，不存在 StuHelper 自签 phone-login refresh token；遗留自签 refresh token 会被 `/api/v1/auth/refresh` 拒绝，遗留自签 access cookie 也不会被认证中间件接受。

### 浏览器 access token 校验模型

- 浏览器 Cookie access token 先查 Redis blacklist，再按 `session_id` 读取 session，校验
  provider application、user 和 access token hash，最后做本地 JWKS / audience / `exp` 验证。
- 新登录把已验证的 provider `exp` 与 access token hash 一起保存；refresh 原子替换二者。
  新 token 的剩余寿命不得大于本地 session lease，也不得超过 blacklist 30 天硬上限。
- `logout` / `logout-all` 对每个 session 按其 access token 的真实剩余寿命写 blacklist。
  已自然过期的 token 不再写 key，接近一秒的剩余时间只为 Redis 最小粒度向上取整；
  超过硬上限直接失败，不能静默截断。
- 滚动升级前创建、没有 `accessTokenExpiresAt` 的旧 session 无法从 token hash 还原
  `exp`。仅在托管 Casdoor access TTL 不超过 session lease 的前提下，以该 session
  的实际 Redis PTTL 作为保守上界；session key 没有 TTL 时 fail-closed。

### Bearer access token 校验模型

- Bearer token 走 Casdoor introspection，以获得 provider 侧即时吊销状态；provider 5xx
  或网络不可用按认证后端不可用处理，不降级为匿名或仅本地解析。
- 请求发送 `token_type_hint=access_token`，但该字段按标准只是提示，不能作为安全边界。
- introspection 返回 active 后，StuHelper 还会检查同一原始 JWT 的 Casdoor
  `tokenType` claim，只有精确的 `access-token` 才能进入 Bearer 认证；`refresh-token`、
  claim 缺失、opaque 或 malformed token 均 fail-closed。
- 用户认证路径还要求 token 来自已登记的应用且 `sub` 非空。introspection response
  的 `token_type=Bearer` 只表示传输 scheme，不能用来区分 access 与 refresh。
- Bearer 路径保持 provider introspection，不为每个请求新增 session lookup 或新 session ID。
  introspection 返回的标准 `exp` 只随认证结果进入上下文，供没有 tracked session 的
  logout 兜底按真实剩余寿命吊销。

### Native exchange / refresh 当前口径

- `POST /api/v1/auth/exchange-native` 请求体：`{ code, state }`
- 成功响应：`{ accessToken, refreshToken, sessionID, expiresIn }`
- 原生 OIDC refresh 必须通过 `X-Stuhelper-Session-ID` 回传 `sessionID`；缺失或不匹配时拒绝 refresh。
- refresh 会对旧 refresh token 做 blacklist，并在 session store 内原子更新新 access
  hash、已验证 `exp`、新 refresh hash 和加密后的 provider refresh token。
- 旧 refresh token 再次提交会触发 reuse detection：吊销该用户全部 session 并记录审计。

## Shadow User

OIDC 用户同步到本地 `users` 表：`casdoor_subject`、`username`、`email`、`avatar_url`。

手机号不是 shadow user 的主锚点。业务侧稳定锚点是 Casdoor subject 映射出的内部 `users.id`；手机号补绑 / 更换只更新 Casdoor 真相源，并同步本地脱敏投影。

## 角色与能力

- Casdoor JWT 提供粗粒度角色
- 中间件静态展开为 capabilities；scoped admin 的学校范围不来自 token，而是由运行时 resolver 从 DB/OpenFGA 补全
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
| 手机号资料写入 | `server/internal/modules/user/service_phone.go` + `server/internal/platform/casdoor/user_profile.go` |
