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
| Provider Access / Refresh Token | access 的 provider `exp` 默认 1 小时，refresh 默认 24 小时；本地 session / cookie lease 由 `TOKEN_REFRESH_TTL` 控制，默认 7 天 | refresh 位于 Path `/api/v1/auth` HttpOnly Cookie；服务端 session 分别保存加密的 provider access/refresh 副本 | 续期与 provider token-family 撤销 |
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
- OIDC provider token 由 StuHelper session store 加密代持：
  - `oidc` / `oidc-native` session 分别保存 provider access 与 refresh token 密文；客户端 access credential 的 hash/`exp` 仍独立保存，不能把可逆密文当认证索引；
  - refresh 在同一个 Redis Lua 操作中更新客户端 access token hash、已验证 `exp`、refresh token hash、两份 provider token 密文和 session lease，避免新旧 family 字段混合；
  - 固定 Casdoor 镜像只发布 `/api/logout` 作为 `end_session_endpoint`，没有 RFC 7009 `revocation_endpoint`。StuHelper 只对与 discovery issuer 同源且路径精确为 `/api/logout` 的 endpoint 使用 Casdoor adapter：发送 `id_token_hint=<provider access token>` 与对应 `client_id`，并要求 HTTP 2xx **且** JSON `status=ok`；跨源 endpoint、`status=error`、空响应或畸形 JSON 都算撤销失败；
  - 若未来 discovery 提供真正的 `revocation_endpoint`，该独立路径仍按 RFC 7009 发送 `token=<refresh>` 与 `token_type_hint=refresh_token`，不会把任意 end-session URL 当作 revocation endpoint；
  - Casdoor refresh grant 会删除旧 token row 并创建新 row，故正常 refresh 成功后不再对已不存在的旧 access token 重复调用 `/api/logout`；只有新 family 尚未提交到本地 session 时，才用新 provider access token 做补偿撤销；
  - 滚动升级前的 session 没有 provider access 密文。固定 Casdoor 在当前授权码/refresh 流中令 `id_token` 与 `access_token` 同值，因此当前设备 logout 可复用已经与 session hash 匹配的原始 access token；logout-all 则用加密 refresh token 先执行一次 Casdoor rotation，再立即撤销替代 family。`invalid_grant` 表示旧 row 已不存在或 `expires_in <= 0`，可视为已失效；其他 4xx、5xx、网络或业务状态错误不能伪装成成功；
  - `logout` / `logout-all` 先完成本地 session/blacklist 撤销，再调用 provider；provider 失败时请求整体返回失败且不能记录成功审计，但不会恢复本地 session。旧 session 若 provider access/refresh 凭据都缺失也必须明确失败，不能伪装成 provider 已撤销；
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
  hash、已验证 `exp`、新 refresh hash 和加密后的 provider access/refresh token。
- blacklisted refresh token 只有在 attribution 指向的 session 仍存在、当前 refresh hash
  非空且与提交 token 的 hash 不同时，才证明它已被成功轮换后再次使用：此时吊销该用户
  全部 session、增加 reuse metric 并记录 `refresh_reuse_detected` 审计。referenced session
  已删除，或仍持有相同 hash，表示 logout 已完成或正处于 blacklist→delete 窗口，只返回
  `refresh token revoked`，不能误伤其他设备或记录虚假安全事件。

## Shadow User

OIDC 用户同步到本地 `users` 表：`casdoor_subject`、`username`、`email`、`avatar_url`。

手机号不是 shadow user 的主锚点。业务侧稳定锚点是 Casdoor subject 映射出的内部 `users.id`；手机号补绑 / 更换只更新 Casdoor 真相源，并同步本地脱敏投影。

## 角色与能力

- Casdoor JWT 提供粗粒度角色
- 中间件静态展开为 capabilities；scoped admin 的学校范围不来自 token，而是由运行时 resolver 从 DB/OpenFGA 补全
- `/api/v1/auth/me` 返回：`roles`、`capabilities`、`globalCapabilities`、`canAccessAdmin`、`displayName`、`isPlatformAdmin`、`capabilityGrants`、`accountSettingsUrl`

### 前端当前用户刷新语义

- 前端通过 `fetchUser` 主动刷新 `/api/v1/auth/me` 时，只有 HTTP 401 能证明当前会话已被
  服务端拒绝，此时清除本地用户和会话提示。403 还可能来自 CSRF、MFA 或权限条件，5xx、
  网络错误和超时都不能证明用户已经登出；这些错误必须向调用方返回，但保留已有用户状态
  以便有界重试。
- 入群认证完成后的 capability 投影检查沿用 1、2、4、8、16 秒五次有界退避。单次非 401
  失败不终止整轮检查；全部尝试失败或仍未看到 `review:create` 时继续停留在
  `projectionPending` 并提供手动重试，不切换成不可恢复的通用错误页。
- Abort 表示页面离开或新一轮检查取代旧一轮，必须立即停止；HTTP 401 切换到重新登录状态。
  保留前端缓存不等于授权成功，后端 capability / OpenFGA 门禁仍是所有受保护操作的权威边界。

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
| Provider token-family revoke | `server/internal/modules/auth/service_provider_tokens.go` + `server/internal/pkg/oidc/revoke.go` |
| 用户同步 | `server/internal/modules/auth/user_sync.go` |
| 手机号资料写入 | `server/internal/modules/user/service_phone.go` + `server/internal/platform/casdoor/user_profile.go` |
