---
type: design
audience: maintainers, backend-dev, frontend-dev, product
status: current
authoritative-source: this file for open-platform v1 architecture; server/api/openapi.yaml for API contract
created: 2026-05-01
last-verified: 2026-05-24
related:
  - iam-v2-casdoor.md
  - auth-and-session.md
  - authorization-model.md
  - security-model.md
scope: 当前 Open Platform v1 baseline 与目标架构；第三方应用接入、用户授权、最小化披露
---

# StuHelper Open Platform / Identity v1

## 状态

当前 v1 baseline 已进入主线实现，不再是 deferred design。

已落地的能力：

- 第三方应用注册 API：`POST /api/v1/open-platform/apps`
- 管理员审批 API：`POST /api/v1/admin/open-platform/apps/{appID}/scopes/{scope}/approve`、`POST /api/v1/admin/open-platform/apps/{appID}/scopes/{scope}/reject`、`POST /api/v1/admin/open-platform/apps/{appID}/approve`
- Legacy Casdoor 应用导入 API：`POST /api/v1/admin/open-platform/apps/import-casdoor`
- 对外 OIDC issuer：`https://id.stuhelper.com`
- OIDC / OAuth authorization server metadata / JWKS：`/.well-known/openid-configuration`、`/.well-known/oauth-authorization-server`、`/.well-known/jwks.json`
- OIDC 授权码链路：`/oauth2/authorize`、`/oauth2/token`、`/oauth2/continue`、`/oauth2/introspect`、`/oauth2/revoke`
- OIDC authorization code flow 强制 S256 PKCE；缺失 `code_challenge`、非 `S256` 方法或无效 `code_verifier` 必须拒绝；token 交换必须提交与授权请求完全一致的 `redirect_uri`；discovery 暴露 `response_modes_supported=["query"]`，授权请求显式携带 `response_mode=query` 时按默认 query 回调处理，其他 response mode 返回 invalid authorization request。
- OIDC ID token 最小化：授权码或 refresh token grant 只有在已授予 OAuth scope 包含 `openid` 时才返回 `id_token`；纯 `resource.read` / `resource.write` 授权码流只返回 user-delegated access token，不能调用 UserInfo。
- OIDC 授权响应 issuer：discovery 暴露 `authorization_response_iss_parameter_supported=true`；授权成功回调和可回调的授权错误响应都会携带 RFC 9207 `iss=https://id.stuhelper.com`，便于客户端在多 issuer 或迁移期并存接入时防护 authorization code mix-up。
- OIDC `state` 透传：授权成功、可回调授权错误、用户拒绝 consent 和 RP-Initiated Logout 回调都会保留客户端提交的非空 `state` 原值，不做空白裁剪，避免破坏客户端 CSRF/state 校验。
- OIDC 静默登录、重新认证与强制同意：discovery 暴露 `prompt_values_supported=["none","login","consent"]`；`prompt=none` 在未登录时经 app 与 redirect URI 校验后回调 `error=login_required`，在已登录但需要资料补全或用户同意时回调 `error=interaction_required` / `error=consent_required`，且不创建临时同意或资料补全 challenge；`prompt=login` 或 `max_age=0` 会转接到上游 Casdoor `prompt=login&max_age=0` 重新认证后继续授权码流程；`prompt=consent` 会对已有 disclosure consent 的授权请求重新展示 StuHelper consent 页。
- OIDC RP-Initiated Logout：discovery 暴露 `end_session_endpoint=https://id.stuhelper.com/oauth2/logout`；登出回跳必须通过 `client_id` 或 `id_token_hint` 绑定 approved app，并精确匹配该 app 已注册 redirect URI；请求一旦携带 `id_token_hint`，即使没有回跳 URI，也必须能验证为 StuHelper Identity ID token，access token 不能作为 logout hint；存在当前 StuHelper 浏览器会话时会撤销 session 并清理认证 cookie。
- OIDC refresh token：`offline_access` 是需开发者申请、管理员审批、用户同意并可撤回的 scope，且必须与 `openid` 同时申请；授权码同时包含 `openid offline_access` 时 token endpoint 才返回 refresh token；`refresh_token` grant 执行 token rotation，并在每次刷新时重新校验 app approved、scope approved 和用户 active consent；刷新请求可用 `scope` 收窄到原 grant 子集，但必须保留 `openid offline_access`，且下一代 refresh token 继承收窄后的 scope，任何扩展都会返回 `invalid_scope`；同一 client 重放已消费 refresh token 时撤销当前 refresh token family，并写入 `iam.token.revoked` 审计事件；同一 client 调用 `/oauth2/revoke` 撤销已被 rotation 消费的 refresh token 时，也会撤销当前 family，用于覆盖登出与刷新并发的旧 token 提交。
- OIDC client credentials：`client_credentials` grant 仅签发 app-only access token，不返回 `id_token` 或 `refresh_token`，只允许请求 `resource.read` / `resource.write` 这类应用级资源 scope；签发和 introspection 都会重新校验 app approved 与 scope approved，UserInfo 必须拒绝 app-only token。access token introspection 遵循 RFC 7662：`token_type` 表示 OAuth token type（`Bearer`），StuHelper 扩展字段 `token_kind` 才表示 `access_token` / `refresh_token` 分类。
- OIDC 敏感响应缓存控制：`/oauth2/token`、`/oidc/userinfo`、`/oauth2/introspect` 和 `/oauth2/revoke` 的成功或错误响应都会返回 `Cache-Control: no-store` 与 `Pragma: no-cache`，避免浏览器、代理或客户端调试缓存保存 token、UserInfo 或 introspection 结果。
- OIDC 认证挑战头：token / introspection / revoke 的 `invalid_client` 401 响应返回 `WWW-Authenticate: Basic realm="StuHelper Identity"`；UserInfo 的 `invalid_token` 401 响应返回 `WWW-Authenticate: Bearer realm="StuHelper Identity", error="invalid_token"`，便于标准 OAuth / OIDC 客户端正确识别认证失败类型。
- OIDC UserInfo：`/oidc/userinfo`
- 用户授权页：`/consent?token=...` 前端页调用 `GET /api/v1/open-platform/consent`
- 资料补全页：`/complete-profile?token=...` 前端页调用 `GET /api/v1/open-platform/profile-completion`
- 用户同意 / 拒绝：`POST /api/v1/open-platform/consent/accept`、`POST /api/v1/open-platform/consent/deny`
- 资料补全后继续：`POST /api/v1/open-platform/profile-completion/continue`
- 用户授权管理：主站 `/user/authorized-apps` 调用 `GET /api/v1/open-platform/consents`、`DELETE /api/v1/open-platform/consents/{appID}`，支持按 scope 或整应用撤销；授权列表会从 disclosure granted 审计派生每个 scope 的 `lastUsedAt`，让用户撤权前能看到最近一次成功披露时间
- 用户授权活动记录：主站 `/user/authorized-apps` 调用 `GET /api/v1/open-platform/consents/audit-events`，只返回当前登录用户自己的 consent grant/deny/revoke、disclosure granted/denied 和 replay_detected 审计摘要
- 披露 API：`GET /api/v1/open-platform/userinfo`、`/verification`、`/student`、`/phone`
- 开发者应用列表 / 创建 UI：主站 `/developers/apps` 调用 `GET /api/v1/open-platform/apps`、`POST /api/v1/open-platform/apps`，可查看 app 与 scope 审核状态并提交新应用；新应用每个 scope 必须提供非空用途说明，pending 应用可由 owner 调用 `POST /api/v1/open-platform/apps/{appID}/withdraw` 撤回为 revoked 终态
- 开发者应用资料维护：主站 `/developers/apps` 调用 `PATCH /api/v1/open-platform/apps/{appID}`，允许 owner 更新非 revoked 应用的展示名、描述、主页和隐私政策 URL；redirect URI 仍走独立审核，资料变更写入 `open_platform.app.profile_updated`
- 开发者 scope 变更申请：主站 `/developers/apps` 调用 `POST /api/v1/open-platform/apps/{appID}/scopes`，可为非 revoked 应用新增 scope，或在 scope 被 rejected / withdrawn 后更新用途说明并重提审核；已 approved 或 pending scope 不能重复提交；pending scope 可由 owner 调用 `POST /api/v1/open-platform/apps/{appID}/scopes/{scope}/withdraw` 撤回
- 开发者 redirect URI 变更申请：主站 `/developers/apps` 调用 `POST /api/v1/open-platform/apps/{appID}/redirect-uris`，对 approved / suspended 应用提交新的完整 redirect URI 列表，管理员批准后整体替换；pending redirect URI 申请可由 owner 调用 `POST /api/v1/open-platform/apps/{appID}/redirect-uri-requests/{requestID}/withdraw` 撤回
- 开发者自有应用审计摘要：主站 `/developers/apps` 调用 `GET /api/v1/open-platform/apps/{appID}/audit-events`，只允许 app owner 查看该应用生命周期、审批、授权、披露、资源授权和 token 探针摘要；响应隐藏用户 ID、原始 token claims 和内部错误细节
- Client secret 生命周期：开发者可在 `/developers/apps` 调用 `POST /api/v1/open-platform/apps/{appID}/secret/rotate` 轮换自己的 approved 应用 secret；管理员可调用 `POST /api/v1/admin/open-platform/apps/{appID}/secret/rotate` 处置 approved / suspended 应用 secret，旧 secret 立即失效，新 secret 只返回一次
- 应用暂停 / 恢复 / 吊销处置：管理员可暂停 approved 应用、将 suspended 应用恢复为 approved，或吊销 pending / approved / suspended 应用；吊销会撤回 pending 子申请，若应用已有 active user consent，则同事务撤销这些 consent，并为受影响用户写入 `open_platform.consent.revoked` 审计，用户授权列表不再显示该已吊销应用
- 管理员应用审核列表 API/UI：`GET /api/v1/admin/open-platform/apps` 与管理后台开放平台应用审核页，可批准 / 驳回 scope、批准 / 驳回 redirect URI 变更、导入 legacy Casdoor 应用、签发 / 轮换 client secret、暂停、恢复和吊销应用
- 管理员审计事件查看 API/UI：`GET /api/v1/admin/open-platform/audit-events` 与管理后台开放平台审计事件页，可按 app、用户、事件类型和 scope 检索审批、consent、密钥、生命周期、资源授权和 token 探针事件；前端使用共享 Open Platform audit event taxonomy 生成筛选项，并通过测试防止新增事件缺少可读标签
- 管理员用户授权处置 API/UI：`GET /api/v1/admin/open-platform/consents` 可按 appID 或 userID 查看 active consent；`POST /api/v1/admin/open-platform/apps/{appID}/consents/revoke` 可按用户和 scope 定向撤销授权，并写入 `open_platform.consent.revoked` 审计 metadata
- 管理员 token 探针证据查看 API/UI：`GET /api/v1/admin/open-platform/token-probe-evidence` 与管理后台开放平台 Token 探针证据页，可按 app、审核人、结果和 client ID 检索 runtime code-flow evidence
- 管理员 disclosure 运营报表 API/UI：`GET /api/v1/admin/open-platform/disclosure-report` 与管理后台开放平台 Disclosure 运营报表页，可查看窗口内总请求、成功、拒绝、超限、异常重放、endpoint 分布、拒绝原因和限流维度
- 资源授权 API/UI v1.1：管理员可在管理后台应用审核页查看、授予和撤销 app 到具体资源的 OpenFGA tuple；后端通过 `GET/POST /api/v1/admin/open-platform/apps/{appID}/resource-grants` 和 `POST /api/v1/admin/open-platform/apps/{appID}/resource-grants/revoke` 管理 tuple；第三方应用可用 `POST /api/v1/open-platform/resources/access/check` 检查资源访问决策
- 数据模型：`open_platform_apps`、`open_platform_scope_requests`、`open_platform_approved_scopes`、`open_platform_user_consents`、`open_platform_audit_events`、`open_platform_token_probe_evidence`
- Casdoor 上游登录、账号资料同步与 StuHelper 本地加密手机号投影披露
- 审批 app 前 Casdoor third-party application 最小化创建 / 更新、legacy 导入 token fields 静态门禁、审批前 command-backed authorization-code runtime token 探针门禁，以及 authorization-code token payload 运维探针脚本
- Disclosure / UserInfo 的 app、app+user、endpoint、consent 维度可配置 Redis 限流、低基数 Prometheus 指标、成功 / 拒绝 / 超限审计、异常重放检测、告警规则和管理端运营报表

2026-05-24 生产现场已完成 token 最小化 code-flow 探针、Identity public smoke 专用客户端和真实 OpenFGA 资源授权 smoke 的脱敏 evidence 留档；后续重点转为持续发布门禁、配置漂移检查和运营报表复核。执行记录见 [`docs/internal/exec-plans/active/current-project-open-items.md`](../internal/exec-plans/active/current-project-open-items.md)。

## 架构定位

Casdoor 与 StuHelper 的职责边界固定如下：

| 层 | 职责 | 不承担 |
|----|------|--------|
| Casdoor (`sso.stuhelper.com`) | 身份、注册、登录、SMS / Email provider、用户手机号真相源、StuHelper 一方登录应用 | StuHelper 业务 scope、学生认证事实、业务授权决策、第三方应用对外 issuer、第三方披露审计 |
| StuHelper Identity (`id.stuhelper.com`) | 对一方 / 三方应用暴露 OIDC issuer、授权码、JWKS、UserInfo、token introspection/revoke | 原始账号密码登录、手机号真相源 |
| StuHelper Open Platform | 第三方应用注册、scope 申请、管理员审批、用户 consent、profile completion gate、disclosure API、审计、撤销与限流 | 原始账号密码登录、直接暴露 Casdoor 用户 API |
| StuHelper DB | 实名认证、学生认证、学校归属、业务用户锚点、open_platform_* 状态 | Casdoor 账号主数据 |
| OpenFGA | 资源级关系授权；v1.1 起承载 app 到具体资源的关系 | scope consent、应用元数据、登录决策 |

第三方应用不能直接读取 Casdoor 用户 API，也不应再直接接入 `sso.stuhelper.com`。对外稳定 issuer 是 `https://id.stuhelper.com`；Casdoor 只作为 `id` 的上游登录源。手机号、用户名、头像、学生认证状态等业务字段只能由 `id` 在用户授权后通过 UserInfo / disclosure payload 返回。

## 完整目标

Open Platform 的完整目标是让校内外第三方应用以可审计、可撤销、最小化披露的方式接入 StuHelper 身份与认证事实：

1. 开发者提交应用、redirect URI、隐私政策、申请 scope 与用途说明；注册、legacy 导入、新增或重提 scope 时都会拒绝空用途说明。
2. 管理员按应用和敏感 scope 审批；通过后由 StuHelper Identity 签发 client secret。
3. 历史上直接接入 Casdoor 的应用可由管理员导入 id registry，优先保留原 `client_id` / `client_secret` / `redirect_uri`，再由应用侧把 issuer 切到 `id.stuhelper.com`。
4. 用户在 StuHelper 授权页看到应用信息、当前登录身份、每个 scope 实际读取的字段和开发者提交的用途说明，并选择允许或拒绝。
5. 用户资料不满足本次授权 scope 时，先进入 `/complete-profile` gate，补全页同样展示相关 scope 的字段清单和用途说明，补全后才能继续 consent / 登录。
6. 第三方应用完成 `id.stuhelper.com` OIDC 登录后，只能获取已审批且已授权的字段。
7. 用户可以查看和撤销已授权应用，并看到每个已授权 scope 的用途说明、授权时间、最近成功披露时间和自己的授权活动记录；开发者可以撤回 pending 应用 / scope / redirect URI 申请、新增或重提 scope、查看自有应用的审计摘要用于自助排障；管理员可以驳回 scope、暂停、吊销、轮换密钥，并查看 Open Platform 审计事件。
8. 任何未审批、未授权、资料未补全、token 无效、redirect 不匹配、依赖不可用、审计失败的路径都 fail-closed。

## 非目标

- 不把学生认证、实名认证、学校归属、QQ 绑定等业务事实同步成第三方可直接读取的 Casdoor claim。
- 不让第三方应用直接读 Casdoor user API。
- 不在第三方 access token 中塞入业务字段；ID token 不包含未审批、未授权或未请求的手机号、邮箱、学校、认证状态等业务字段。
- 不把 `qq.binding.read` 放入默认 scope 目录。
- 不在 v1 开放第三方直接操作评课删除、用户封禁等管理员级资源。
- 不把 scope consent 建模进 OpenFGA；scope 是业务字符串属性，权威状态在 `open_platform_user_consents`。

## 登录与授权流程

```text
第三方应用
  │
  │ GET https://id.stuhelper.com/oauth2/authorize
  │   ?response_type=code
  │   &client_id=...
  │   &redirect_uri=...
  │   &scope=openid profile email phone stu.student.status.read ...
  │   &state=...
  ▼
StuHelper Identity /oauth2/authorize
  │
  ├─ 未登录: 跳转 /login?redirect=/oauth2/authorize...
  ▼
StuHelper Open Platform
  ├─ 校验 app approved
  ├─ 校验 redirect_uri 精确匹配
  ├─ 校验 scope 合法且已审批
  ├─ 校验用户资料是否满足 scope
  │
  ├─ 资料缺失: 返回 /complete-profile?token=...
  │             用户补全后继续同一 OAuth 请求
  │
  ├─ 校验当前 StuHelper 用户是否已 consent
  │
  ├─ 未 consent: 返回 /consent?token=...
  │             用户允许后写 open_platform_user_consents
  │
  └─ 已 consent: 签发一次性 authorization code
                  token endpoint 换取 id_token / access_token
```

`/consent` 与 `/complete-profile` 是 StuHelper 页面，不依赖 Casdoor 内置 consent 表达业务 scope。Casdoor 仍负责账号注册、登录和上游会话；`id.stuhelper.com` 负责对第三方签发授权码和 token。

## OIDC Scope 与业务 Scope

第三方应用对 `id.stuhelper.com` 可请求标准 OIDC scope 和 StuHelper 业务 scope。标准 scope 会映射到业务 scope：

- `openid`：只表示 OIDC 登录请求，不单独披露字段。
- `profile` → `profile.basic.read`
- `email` → `email.read`
- `phone` → `phone.read`

授权链路会同时保留两套 scope 表达：第三方请求并被授予的 OAuth scope 保存在 authorization code、access token、token response 的 `scope` 字段和 introspection 响应中，例如客户端请求 `openid profile email resource.read` 时仍看到这些值；Open Platform 内部审批、资料补全、用户 consent、UserInfo 字段披露和审计判断使用映射后的业务 scope，例如 `profile.basic.read`、`email.read`、`resource.read`。这样外部 OIDC 客户端不需要理解 StuHelper 的内部字段 scope，服务端仍能用规范化 scope 做最小化授权。只有已授予 scope 包含 `openid` 的 OIDC 登录授权才会签发 `id_token` 并允许访问 UserInfo；纯资源 scope 授权码流仍可得到 access token 和 introspection 能力，但不会触发身份 token 或资料披露。

业务 scope 由 StuHelper 自己维护：

| Scope | 字段 | 敏感度 | 当前策略 |
|-------|------|--------|----------|
| `profile.basic.read` | `username`、`displayName`、`avatar` | low | 需用户授权 |
| `email.read` | `email` | medium | 需用户授权 |
| `phone.read` | `phone`、`phoneMasked`、`phoneVerified` | high | 需应用审批 + 用户授权；手机号从 StuHelper 本地加密投影解密披露 |
| `stu.identity.status.read` | `identityVerified` | high | 需应用审批 + 用户授权 |
| `stu.identity.type.read` | `identityType` | high | 需应用审批 + 用户授权 |
| `stu.student.status.read` | `studentVerified` | high | 需应用审批 + 用户授权 |
| `stu.student.school.read` | `school.id`、`school.name` | high | 需应用审批 + 用户授权 |
| `resource.read` | 资源类读取能力准入 | high | 需应用审批；可用于授权码 flow 和 `client_credentials`；具体资源由 OpenFGA tuple 决定 |
| `resource.write` | 资源类写入能力准入 | very_high | 需应用审批；可用于授权码 flow 和 `client_credentials`；具体资源由 OpenFGA tuple 决定 |
| `offline_access` | refresh token 换取后续登录 token 的长期授权 | high | 必须与 `openid` 同时申请；需应用审批 + 用户授权；token 绑定发行时的 consent 指纹，用户撤回或重新授权后旧 refresh token 不能继续换取 token；有效期由 `IDENTITY_REFRESH_TOKEN_TTL` 控制，默认 30 天；Redis 仅使用 token 哈希索引；重放已消费 token 会撤销当前 family 并写入 `iam.token.revoked` |

默认目录不包含身份证号、学号、详细地址、出生日期和 QQ 绑定。

授权码、access token 和 introspection 返回的 `scope` 是客户端被授予的 OAuth scope 原文去重结果；内部业务 scope 只用于服务端审批、consent 和 disclosure。`openid` only token 不产生业务字段，但仍会在 token scope 中保留 `openid`，token response 会返回只含最小身份声明的 `id_token`，并在 introspection 时校验应用当前仍为 `approved`。不含 `openid` 的 OAuth token 不返回 `id_token`，UserInfo 必须拒绝。

## Disclosure API

当前 disclosure API：

```text
GET /oidc/userinfo
GET /api/v1/open-platform/userinfo
GET /api/v1/open-platform/verification
GET /api/v1/open-platform/student
GET /api/v1/open-platform/phone
```

`/oidc/userinfo` 是推荐给新 OIDC 应用使用的标准端点。`/api/v1/open-platform/*` disclosure 端点保留给已有内部调用或后续资源 API 分层。

每个请求必须带：

- `/oidc/userinfo`：`Authorization: Bearer <access_token>`；
- legacy disclosure API：bearer/cookie 身份凭据，由 auth middleware 解析当前用户；显式 Bearer 凭据携带 app identity 时必须与 `client_id` 一致，浏览器 cookie 会话使用请求中的 `client_id` 识别目标应用；
- `client_id` 对应已批准应用；
- `redirect_uri` 精确匹配该应用登记值；
- `scope` 空格分隔业务 scope；
- 可选 `consent_base_url`，用于 legacy disclosure API 返回 `/consent?token=...`。

服务端决策顺序：

1. 标准化 scope，非法 scope 直接拒绝。
2. 按 `client_id` 找应用，应用必须是 `approved`。
3. 显式 Bearer 凭据存在 app identity 时必须与请求 `client_id` 一致，防止某个 client 的 user token 借用另一个 client 的 consent。
4. `redirect_uri` 必须精确匹配该应用登记值，即使用户已存在 active consent 也不能跳过。
5. 读取 auth middleware 在 handler 边界解析出的内部 `users.id`；service 层不再把 Casdoor subject 作为业务用户身份。
6. 对 app、app + user、endpoint 执行 Redis 滑动窗口限流；Redis 不可用时 fail-closed。
7. 检查应用已审批全部请求 scope。
8. 检查用户对该应用仍有 active consent。
9. 未授权时对 consent challenge 维度限流，通过后返回 consent required，并生成 5 分钟 Redis consent challenge。
10. 已授权时按 scope 查询最小字段并返回。

Disclosure 路径会写入 `open_platform_audit_events`：

- `open_platform.disclosure.granted`：成功返回最小披露 payload。
- `open_platform.disclosure.denied`：记录 app 未批准、scope 未批准、consent required、rate limited、资料投影不可用、手机号解密失败等拒绝原因。
- `open_platform.disclosure.replay_detected`：同一 app、用户、endpoint、结果和 scope 组合在重放窗口内重复拒绝达到阈值，写入 signature hash、count 和限流维度；审计写入失败继续 fail-closed。

Prometheus 指标使用低基数标签：

- `open_platform_disclosure_requests_total{endpoint,result}`：endpoint 限定为 `userinfo`、`verification`、`student`、`phone`、`identity_token`。
- `open_platform_disclosure_rate_limit_total{dimension,outcome}`：dimension 限定为 `app`、`app_user`、`endpoint`、`consent`。
- `open_platform_disclosure_replay_total{endpoint,outcome}`：outcome 限定为 `detected`、`suppressed`、`error`。

阈值通过环境变量配置，默认值适合生产起步：

| 环境变量 | 默认 | 含义 |
|----------|------|------|
| `OPEN_PLATFORM_DISCLOSURE_APP_LIMIT` | `600` | 单 app 每窗口 disclosure 请求数 |
| `OPEN_PLATFORM_DISCLOSURE_APP_USER_LIMIT` | `120` | 单 app + 用户每窗口 disclosure 请求数 |
| `OPEN_PLATFORM_DISCLOSURE_ENDPOINT_LIMIT` | `1200` | 单 endpoint 每窗口 disclosure 请求数 |
| `OPEN_PLATFORM_DISCLOSURE_CONSENT_LIMIT` | `20` | 单 app + 用户每窗口 consent challenge 次数 |
| `OPEN_PLATFORM_DISCLOSURE_WINDOW_SECONDS` | `60` | disclosure 限流窗口 |
| `OPEN_PLATFORM_DISCLOSURE_REPLAY_LIMIT` | `8` | 重放窗口内同签名拒绝达到该次数后记录 replay 事件 |
| `OPEN_PLATFORM_DISCLOSURE_REPLAY_WINDOW_SECONDS` | `300` | 异常重放检测窗口 |
| `OPEN_PLATFORM_DISCLOSURE_REPLAY_AUDIT_COOLDOWN_SECONDS` | `600` | 同一签名 replay 审计冷却时间 |

管理端运营报表：

```text
管理员调用 GET /api/v1/admin/open-platform/disclosure-report?windowHours=24
  → 统计 open_platform_audit_events 中 disclosure granted / denied / replay_detected 事件
  → 返回 summary、endpoint stats、拒绝原因、rate limit 维度和最近 replay 事件
  → windowHours 默认 24，最大 168
```

Prometheus 告警规则覆盖异常重放、拒绝峰值和 disclosure rate limited。告警规则契约由 `infra/ops/tests/observability-alert-contract.sh` 锁定。

`phone.read` 是特殊路径：Casdoor 仍是账号手机号上游来源，但 disclosure 不直接开放 Casdoor user API。auth / user 同步链路把已验证手机号写入 StuHelper 本地加密投影；披露时仅在应用审批、用户授权、资料完整且解密成功后标准化返回。

## Resource Access API

资源授权 API v1.1 把应用级能力准入和具体资源授权分开：

- `resource.read` / `resource.write` 是应用可申请资源类能力的 scope，权威状态仍在 `open_platform_approved_scopes`。
- 授权码 flow 可以携带 `resource.read` / `resource.write` 以便第三方在 token introspection 中确认应用能力，但它们不是用户资料 disclosure scope，不进入 consent 页，也不写入 `open_platform_user_consents`；未同时包含 `openid` 时，token response 不返回 `id_token`，UserInfo 也会拒绝该 access token。
- `client_credentials` grant 也只能请求 `resource.read` / `resource.write`，签发的 app-only access token 使用 `sub=client:<client_id>`、`grant_type=client_credentials`，并且不能调用 `/oidc/userinfo`。
- 具体资源能否被某个 app 读取或写入，只看 OpenFGA tuple，不读取 `open_platform_user_consents`。
- 资源授权 tuple 当前覆盖 `user_profile:{id}` 和 `resource_item:{id}`；`user_profile` 仅支持 `can_read_by_app`，`resource_item` 支持 `can_read_by_app` / `can_write_by_app`，主体为 `open_platform_app:{app_id}`。
- 管理员授予资源授权时要求 app 已 `approved`，且已批准对应 `resource.read` 或 `resource.write` scope；授予和撤销都必须携带非空审计原因。撤销资源授权不要求 app 仍为 approved，便于处置 suspended / revoked app 的历史 tuple。
- 管理员吊销 approved / suspended app 时，服务层会列出并删除该 app 在 OpenFGA 中的 `resource_item` / `user_profile` 授权 tuple，并以 `source=app_lifecycle` 写入 `open_platform.resource_access.revoked` 审计；其他 app 的 tuple 不受影响。
- 第三方应用检查资源访问时首选使用 `Authorization: Bearer <client_credentials access_token>` 认证应用身份；兼容模式仍支持 body 内 `clientID` + `clientSecret`，但 Bearer token 与 body client credential 互斥，混用时返回 invalid request，避免凭据歧义和应用密钥进入请求体；携带非 Bearer 或重复 Authorization 头会被拒绝，使用 body credential 时应省略 Authorization 头。Bearer token 缺少本次动作所需的 `resource.read` / `resource.write` 时返回 `allowed=false` 且 `reason=token_scope_missing`；缺少 app 已审批 scope 或 OpenFGA tuple 时同样返回 `allowed=false`，OpenFGA / 审计不可用时 fail-closed。
- 管理后台应用审核页提供资源授权对话框，可按资源类型列出 OpenFGA grant，给 approved app 新增 grant，并撤销 suspended / revoked app 的历史 grant；grant / revoke 表单都会强制填写审计原因。

当前 API：

```text
GET  /api/v1/admin/open-platform/apps/{appID}/resource-grants?resourceType=resource_item
POST /api/v1/admin/open-platform/apps/{appID}/resource-grants
POST /api/v1/admin/open-platform/apps/{appID}/resource-grants/revoke
POST /api/v1/open-platform/resources/access/check
```

资源授权审计事件：

- `open_platform.resource_access.granted`
- `open_platform.resource_access.revoked`
- `open_platform.resource_access.checked`

## Consent Challenge

Consent challenge 存在 Redis：

```text
key: open_platform:consent:{token}
ttl: 5 minutes
payload:
  app_id
  user_id
  scopes
  redirect_uri
  state
  flow
  code_challenge
  code_challenge_method
  nonce
  created_at
  expires_at
```

安全约束：

- token 为服务端生成的高熵随机值，URL 中不直接暴露 app_id / user_id / scope。
- token 绑定当前内部用户；其他用户打开会被拒绝。
- `redirect_uri` 只能来自应用登记的精确 URI，不接受通配符、fragment 或运行时自由 return URL。
- consent / profile completion challenge 创建、consent page、consent accept / deny、profile completion page / continue 和 Identity `/oauth2/continue` 都必须重新加载当前 app 状态，并在创建或展示授权上下文、写入用户 consent、写入用户拒绝审计、删除 challenge、创建后续 consent challenge、生成 OIDC redirect 或签发 authorization code 前，重新校验 app 仍为 `approved`、`redirect_uri` 仍精确匹配当前登记列表、请求 scope 仍在 approved scope 集合内。任一状态在 challenge 有效期内漂移时 fail-closed，保留原 challenge 供用户重新发起授权，不写入过期授权或拒绝事实；创建阶段即使调用方传入旧的 app 快照，也必须按数据库当前 app 状态判断。
- 用户同意后写入 `open_platform_user_consents` 与 `open_platform.consent.granted` 审计，并删除 Redis token。
- 用户拒绝后先写入 `open_platform.consent.denied` 审计，再删除 Redis token，并返回带 `error=access_denied` 的第三方 redirect URL；审计写入失败时不删除 challenge，避免产生无审计的拒绝事实。
- Identity flow 中用户同意后的 `/oauth2/continue` 只校验 consent / scope / app 当前仍 active，并签发 authorization code；不得在此阶段构建 UserInfo / ID token profile 或写入 `open_platform.disclosure.granted`。披露成功审计只能发生在 token exchange 实际签发含资料 claim 的 ID token，或客户端调用 UserInfo / disclosure API 时。

当前 baseline 已提供用户侧 consent revoke API/UI。用户授权列表从 `open_platform_scope_requests.reason` 带出每个 scope 的用途说明，让用户在撤销前能对照开发者申请理由、授权时间和最近成功披露时间。用户授权活动记录从 `open_platform_audit_events` 读取当前登录用户自己的 consent grant/deny/revoke、disclosure granted/denied 和 replay_detected 摘要，可按 app、eventType、scope 分页过滤；接口不接受任意 userID，避免复用管理员级全量审计能力。撤销按 `app_id + user_id + scope` 更新 `open_platform_user_consents.revoked_at`；不传 scope 时撤销该应用下当前用户全部 active consent。撤销写入 `open_platform_audit_events`，下一次 disclosure / UserInfo 会立即回到 consent required。

管理员侧提供单独的用户授权运营面。管理后台 `/open-platform/consents` 调用 `GET /api/v1/admin/open-platform/consents`，要求至少提供 appID 或 userID，避免误扫全量 consent；结果按 `userID + app` 分组并带出 scope、授权时间、最近成功披露时间和开发者用途说明。管理员通过 `POST /api/v1/admin/open-platform/apps/{appID}/consents/revoke` 按 userID 撤销整 app 授权，或只撤销指定 scopes。撤销审计继续使用 `open_platform.consent.revoked`，metadata 包含 `actor=admin`、`actorUserID`、`reason`、`source=admin_console`，用于事故响应、隐私请求和合规追踪。

## 应用生命周期

```text
开发者提交应用
  → open_platform_apps.status = pending
  → open_platform_scope_requests.status = pending

管理员审批 scope
  → open_platform_scope_requests.status = approved
  → open_platform_approved_scopes 写入 app_id + scope

管理员驳回 scope
  → open_platform_scope_requests.status = rejected
  → 写入 open_platform_audit_events: open_platform.scope.rejected

开发者调用 POST /api/v1/open-platform/apps/{appID}/scopes/{scope}/withdraw
  → 校验当前用户是 owner 且 app.status != revoked
  → 仅允许撤回 pending scope
  → open_platform_scope_requests.status = withdrawn
  → 写入 open_platform_audit_events: open_platform.scope.withdrawn

开发者调用 POST /api/v1/open-platform/apps/{appID}/scopes
  → 校验当前用户是 owner 且 app.status != revoked
  → 新增 scope 或将 rejected / withdrawn scope 更新用途说明后置回 pending
  → 已 approved / pending scope 不能重复申请
  → 写入 open_platform_audit_events: open_platform.scope.requested

开发者调用 POST /api/v1/open-platform/apps/{appID}/withdraw
  → 校验当前用户是 owner 且 app.status = pending
  → open_platform_apps.status = revoked
  → 写入 open_platform_audit_events: open_platform.app.withdrawn

开发者调用 PATCH /api/v1/open-platform/apps/{appID}
  → 校验当前用户是 owner 且 app.status != revoked
  → 仅更新 display_name / description / homepage_url / privacy_policy_url
  → redirect URI 不在此接口变更，仍需独立审核
  → 写入 open_platform_audit_events: open_platform.app.profile_updated

管理员审批应用
  → 只允许 open_platform_apps.status = pending
  → 生成一次性 client_secret
  → open_platform_apps.status = approved
```

Redirect URI 变更：

```text
开发者调用 POST /api/v1/open-platform/apps/{appID}/redirect-uris
  → 校验当前用户是 owner 且 app.status ∈ {approved, suspended}
  → 写入 / 更新 open_platform_redirect_uri_requests pending 申请
  → 写入 open_platform_audit_events: open_platform.app.redirect_uris.requested

管理员调用 POST /api/v1/admin/open-platform/apps/{appID}/redirect-uri-requests/{requestID}/approve
  → 将申请置为 approved
  → 用申请中的 redirect URI 列表整体替换 open_platform_apps.redirect_uris
  → 写入 open_platform_audit_events: open_platform.app.redirect_uris.approved

管理员调用 POST /api/v1/admin/open-platform/apps/{appID}/redirect-uri-requests/{requestID}/reject
  → 将申请置为 rejected，不修改应用当前 redirect URI
  → 写入 open_platform_audit_events: open_platform.app.redirect_uris.rejected

开发者调用 POST /api/v1/open-platform/apps/{appID}/redirect-uri-requests/{requestID}/withdraw
  → 校验当前用户是 owner 且 app.status ∈ {approved, suspended}
  → 仅允许撤回 pending redirect URI 申请
  → 将申请置为 withdrawn，不修改应用当前 redirect URI
  → 写入 open_platform_audit_events: open_platform.app.redirect_uris.withdrawn
```

已批准应用的 secret 轮换：

```text
开发者调用 POST /api/v1/open-platform/apps/{appID}/secret/rotate
  → 校验当前用户是 owner 且 app.status = approved
  → 更新 open_platform_apps.client_secret_hash
  → 写入 open_platform_audit_events: open_platform.app.secret_rotated
  → 只在本次响应返回新的 client_secret

开发者调用 GET /api/v1/open-platform/apps/{appID}/audit-events
  → 校验当前用户是 owner，只读取该 app 的审计事件
  → 可按 eventType / scope 分页过滤
  → 返回生命周期、审批、redirect URI、secret、consent/disclosure、资源授权和 token 探针摘要
  → 响应不返回 user_id、原始 tokenClaims、内部 error / stack 等管理员级 metadata

管理员调用 POST /api/v1/admin/open-platform/apps/{appID}/secret/rotate
  → app.status ∈ {approved, suspended}
  → 更新 open_platform_apps.client_secret_hash
  → 写入 open_platform_audit_events: open_platform.app.secret_rotated
  → 只在本次响应返回新的 client_secret
```

暂停 / 恢复 / 吊销：

```text
管理员调用 POST /api/v1/admin/open-platform/apps/{appID}/suspend
  → app.status 从 approved 变为 suspended
  → client secret 校验、token、disclosure 路径 fail-closed
  → 写入 open_platform_audit_events: open_platform.app.suspended

管理员调用 POST /api/v1/admin/open-platform/apps/{appID}/resume
  → app.status 从 suspended 变为 approved
  → client secret 校验、token、introspection、disclosure 路径重新按 approved app 处理
  → 已被用户撤销的 consent 不自动恢复，仍需用户重新授权
  → 写入 open_platform_audit_events: open_platform.app.resumed

管理员调用 POST /api/v1/admin/open-platform/apps/{appID}/revoke
  → app.status 从 pending / approved / suspended 变为 revoked
  → 撤回 pending scope / redirect URI 子申请
  → 同事务撤销该 app 全部 active open_platform_user_consents
  → 为每个受影响 user + scope 写入 open_platform_audit_events: open_platform.consent.revoked
  → 删除该 app 的 OpenFGA 资源授权 tuple，并写入 open_platform.resource_access.revoked(source=app_lifecycle)
  → 后续不能继续轮换 secret
  → 写入 open_platform_audit_events: open_platform.app.revoked
```

审计查看：

```text
管理员调用 GET /api/v1/admin/open-platform/audit-events
  → 可按 appID / userID / eventType / scope 过滤
  → 返回 open_platform_audit_events 的 created_at DESC, id DESC 分页列表
  → metadata 作为 JSON object 原样返回，便于追踪原因、requestID 和 redirect URI 变更明细
```

历史 Casdoor 直连应用迁移：

```text
管理员调用 POST /api/v1/admin/open-platform/apps/import-casdoor
  → 读取 Casdoor application 元数据
  → 导入 client_id / redirect_uri / secret hash / scopes
  → open_platform_apps.status = approved
  → 管理后台应用审核页提供表单入口，隐私政策 URL 必填，可覆盖展示名、主页、redirect URI、client_secret 与 scope 用途说明
  → 应用把 discovery 从 https://sso.stuhelper.com 切到 https://id.stuhelper.com
```

迁移时优先保留原 `client_id` 和 `client_secret`，这样应用通常只需要改 issuer/discovery。若 Casdoor 不返回原 secret 且管理员也未提供，导入接口会生成新的 `client_secret`，应用需要同步更新配置。应用侧迁移步骤见 [Casdoor 直连应用迁移到 StuHelper Identity](../guides/migrate-sso-app-to-id.md)。

应用状态：

- `pending`：等待审核，不可授权。
- `approved`：可以进入授权和 disclosure。
- `suspended`：临时停用，可由管理员恢复为 `approved`。
- `revoked`：永久吊销。

`OpenPlatformManage` capability（字符串：`open_platform:manage`）控制管理员审批接口。

## 数据模型

| 表 | 作用 |
|----|------|
| `open_platform_apps` | 应用元数据、client_id、client_secret_hash、redirect_uris、状态 |
| `open_platform_scope_requests` | 开发者申请的 scope 与用途说明、审核状态 |
| `open_platform_approved_scopes` | 已批准 scope 的当前集合 |
| `open_platform_user_consents` | 用户对 app + scope 的 active/revoked 授权事实 |
| `open_platform_audit_events` | 披露、consent、审批、资源授权等事件留痕 |

用户授权管理的规模化查询路径有显式索引保障：`idx_open_platform_user_consents_active_user` 支撑当前用户 active consent 列表，`idx_open_platform_user_consents_active_app` 支撑管理员按 app 定位 active consent，`idx_open_platform_audit_events_disclosure_usage` 支撑从 `open_platform.disclosure.granted` 审计按 app / user 派生每个 scope 的最近成功披露时间。

Scope consent 不进 OpenFGA。资源 API v1.1 把具体资源关系写成 OpenFGA tuple，例如：

```text
resource_item:{id}#can_write_by_app @ open_platform_app:{app_id}
resource_item:{id}#can_read_by_app  @ open_platform_app:{app_id}
user_profile:{id}#can_read_by_app   @ open_platform_app:{app_id}
```

## 安全基线

- 第三方应用只信任 `iss=https://id.stuhelper.com`，不得继续信任 `iss=https://sso.stuhelper.com` 作为 StuHelper Open Platform issuer；标准 OIDC 客户端可读取 `/.well-known/openid-configuration`，OAuth2 网关或资源服务器也可读取 RFC 8414 `/.well-known/oauth-authorization-server`，两者返回同一 issuer 与 endpoint 基线。
- 支持 RFC 9207 的第三方应用必须校验授权响应 query 中的 `iss` 与 discovery `issuer` 一致；不支持该参数的 OAuth 客户端也必须忽略未知 query 参数，不能因为额外 `iss` 回调参数中断 code flow。
- `id_token` / UserInfo 中只返回本次 scope 覆盖的字段；只有包含 `openid` 的授权才返回 `id_token` 并允许 UserInfo，手机号、学校、学生认证等敏感字段必须有应用审批、用户授权和资料完整性 gate。
- `id_token` 与 UserInfo 使用同一套 StuHelper Open Platform claim 名称，例如 `identityVerified`、`identityType`、`studentVerified`、`phoneVerified` 和 `school`；同时为标准 OIDC profile / phone 字段补充 `preferred_username`、`name`、`picture`、`email_verified`、`phone_number`、`phone_number_verified` 等别名。
- 生产准入必须实测 `id` token / UserInfo 不含未请求或未授权的业务 claim；legacy Casdoor 直连迁移与审批路径还必须保证 Casdoor application `TokenFields` 显式最小化，且审批前 runtime code-flow 探针返回的 token payload 不含手机号、学生认证、学校、身份类型等业务 claim。生产环境必须设置 `OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_REQUIRED=true`，让 `OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND=/app/casdoor-runtime-token-probe-runner.mjs` 或等价 runner 可执行，并在发布时通过专用 `CASDOOR_TOKEN_PROBE_SMOKE_*` app 跑 `casdoor-runtime-token-probe-smoke.sh`。
- redirect URI 必须精确匹配，禁止 wildcard、fragment、regex、通配子域、通配路径。
- 授权码链路必须使用 S256 PKCE，`code_verifier` 长度和字符集遵守 RFC 7636；不得接受 `plain` 或无 PKCE 的 code flow；`/oauth2/token` 必须提交并精确匹配授权请求中的 `redirect_uri`。
- 授权响应只支持 query response mode；`response_mode` 为空或 `query` 时授权码、错误和 RFC 9207 `iss` 都写入 redirect URI query，`fragment`、`form_post` 或带空白的 response mode 必须拒绝。
- `/oauth2/token`、`/oauth2/introspect` 和 `/oauth2/revoke` 支持 `client_secret_basic` 与 `client_secret_post`，但同一请求只能使用一种 client authentication method；同时携带 Basic 凭据和 body `client_id` / `client_secret` 时返回 `invalid_client`，即使 body 凭据字段重复出现也优先按认证方法混用处理；重复 `Authorization` 头、任何非 Basic 或畸形 Basic 的 `Authorization` 头也会返回 `invalid_client`，不会退回 body credential，避免凭据歧义和跨客户端探测。`/oauth2/introspect` 和 `/oauth2/revoke` 接受标准 `token_type_hint`，其中 revoke 会按 `access_token` / `refresh_token` hint 优先查找，但找不到时继续查所有支持的 token 类型；未知 hint 按兼容策略忽略。
- `/oauth2/authorize`、`/oauth2/continue`、RP-Initiated Logout、`/oauth2/token`、`/oauth2/introspect` 和 `/oauth2/revoke` 的 OAuth / OIDC 单值参数重复出现时必须拒绝，避免由框架默认取第一个值造成 redirect URI、state、token 或 client authentication 参数歧义；`/oauth2/continue`、Open Platform consent 页和 profile completion 页的一次性 challenge token 必须恰好出现一次且非空。授权和登出端点重复参数按 invalid authorization/logout request 拒绝；RP-Initiated Logout 的 POST 形式不接受 URL query 参数，存在请求体时只接受 `application/x-www-form-urlencoded`，避免 `id_token_hint`、`client_id`、`post_logout_redirect_uri` 或 `state` 被放入代理 / Web 访问日志，也避免 JSON/text body 被静默当作空登出请求；`/oauth2/token`、`/oauth2/introspect` 和 `/oauth2/revoke` 只接受 `application/x-www-form-urlencoded` body 参数，缺失 Content-Type、JSON、text/plain 或任何 URL query 参数都返回 `400 {"error":"invalid_request"}`，避免 client secret、authorization code、refresh token、access token 或 introspection token 出现在代理 / Web 访问日志中，也避免非表单 body 被静默解析为空表单；`/oauth2/token` 要求非空 `grant_type`，`/oauth2/introspect` 和 `/oauth2/revoke` 通过 client authentication 后仍要求非空 `token`，缺失、重复或空白必填参数统一返回 `400 invalid_request`，不会把空 token 当作 inactive token 或 revoke no-op。UserInfo 只接受单个 `Authorization: Bearer <access_token>` 作为 token 来源；任何 URL query 参数、POST body 或重复 `Authorization` 头都按 `invalid_token` 拒绝，不尝试从 URL/body 读取 access token，也不尝试选择任意一个 Bearer token。
- `invalid_client` 与 `invalid_token` 的 401 响应都会带对应 `WWW-Authenticate` challenge；客户端应按 challenge 类型区分 client credential 失败和 Bearer token 失败，不要只靠 HTTP 状态码重试。
- `prompt=login` 和解析后为 0 的 `max_age` 必须触发真实上游 SSO 重新认证，重新认证回跳前会消费掉导致循环的 `prompt=login` / 零值 `max_age` 参数；正数 `max_age` 按当前会话 `auth_time` 校验，缺少可信 `auth_time` 时 fail-closed 到重新认证。
- `prompt=consent` 必须强制重新展示 disclosure scope consent 页；与 `prompt=login` 组合时，重新认证回跳目标必须保留 `prompt=consent`，确保用户完成上游重新认证后仍会确认授权。
- RP-Initiated Logout 的 `id_token_hint` 只接受 StuHelper Identity ID token；无论是否携带 `post_logout_redirect_uri`，access token、app-only token 或任意非 ID token hint 都必须按 invalid logout request 拒绝。
- `refresh_token` grant 如果携带 `scope`，只能请求原 refresh grant 的子集，并必须保留 `openid offline_access`；成功后 access token、ID token、token response 与新 refresh token 都使用收窄后的 scope，不能通过 refresh 静默扩展或恢复已移除的长期授权字段。服务端还会要求 refresh token 所属 family 的 `currentKey` 精确匹配当前 token hash，family 缺失、已撤销或指向其他 token 时 refresh grant 与 refresh token introspection 都 fail-closed；access token 与 refresh token 还会携带当前 disclosure consent 指纹，UserInfo、introspection 和 refresh grant 必须确认该指纹仍等于当前 active consent 指纹，用户撤回后再重新授权也不会让旧 token 复活；签发新 access token / ID token 前必须先消费当前 refresh token，避免并发重放失败请求产生 UserInfo 或 ID token 披露副作用。
- Identity Server 验证自签 access token 时除签名、`iss`、`exp`、`typ=access`、`sub` 和 `jti` 外，还要求 `aud` 包含 `client_id` 且 `azp` 与 `client_id` 一致；验证 ID token 时要求 `aud` 存在，多 audience token 必须携带属于 `aud` 的 `azp`，单 audience token 如携带 `azp` 也必须确认其属于 `aud`，避免跨 client audience 或错误签发的 token 被 UserInfo、introspection、资源访问或 logout hint 路径接受。
- `/oauth2/introspect` 和 `/oauth2/revoke` 只能由 token 所属 client 操作；其他有效 client credential 不能探测 token active 状态，也不能撤销不属于自己的 token。refresh token introspection 只对所属 client 返回 `active=true`，并在 rotation 消费、revoke、用户撤权或 app / scope 失效后返回 `active=false`。所属 client 成功 revoke access token、当前 refresh token 或已被 rotation 消费的 refresh token 时写入 `iam.token.revoked`，只记录 JTI 或 refresh token family hash，不记录原始 token；已消费 refresh token 的 revoke 会删除对应 used-token 记录，避免重复提交产生重复撤销审计。未知 token 或跨 client revoke 仍按 OAuth 兼容语义返回 `200` no-op；但 Redis 黑名单 / refresh family 查找或删除等撤销依赖失败时返回 `503 {"error":"server_error"}`，避免客户端误判 token 已撤销。
- `/oauth2/introspect` 不只校验 JWT 签名和 JTI 黑名单，还会按当前 Open Platform 状态 fail-closed：app 必须仍为 `approved`，token 中的 OAuth scope 会先映射为业务 scope，业务 scope 必须仍被批准，且当前用户对 disclosure scope 仍有 active consent。含 disclosure consent 的 access token / refresh token 还必须匹配发行时记录的 consent 指纹；用户撤销 scope / 整应用授权、`prompt=consent` 刷新授权、管理员暂停或吊销 app 后，后续 introspection 必须返回 `active=false`，即使用户随后重新授权也只能使用新授权码发行的新 token；`openid` only token 也要求 app 仍为 `approved`，`resource.read` / `resource.write` 只要求 app scope approval，不要求用户 consent；`client_credentials` token 只允许资源 scope，并同样在 introspection 时重新校验 app 和 scope 当前状态。active access token introspection 返回 `token_type=Bearer` 与 `token_kind=access_token`；refresh token introspection 返回 `token_kind=refresh_token`。
- 应用未批准、scope 未批准、用户未授权、审计不可用、手机号本地加密投影缺失或不可解密时拒绝披露。
- 敏感 scope 审批必须有管理员留痕。
- Consent UI 必须逐项展示实际字段名，不能只写“基本信息”这类模糊文案。
- 第三方应用密钥只在批准或轮换时展示一次，服务端只保存 hash。
- app 到具体资源的授权必须落在 OpenFGA；不得用用户 consent 或 scope approval 代替具体资源 tuple。

## 验证策略

当前 v1 baseline 的最小验证集：

1. 注册应用写入 pending app 与 pending scope request。
2. 管理员审批 scope 后写入 approved scope。
3. 开发者可维护非 revoked app 展示资料，并撤回 pending app / scope / redirect URI 申请；撤回后管理员不能继续批准原申请，scope 可重新提交，pending scope 重复提交返回冲突。
4. 管理员审批 app 前先创建 / 更新对应 Casdoor third-party application，强制 `TokenFields=[]` 并写入 `open_platform.app.token_probe.passed`；runtime code-flow 探针 evidence 写入 `open_platform_token_probe_evidence` 并产生 `open_platform.app.token_probe.runtime.passed/failed` 审计；探针失败时拒绝 approved。
5. 未审批 app 或 scope 调 authorize/disclosure 被拒绝。
6. 资料缺失时 authorize 返回 `/complete-profile?token=...`，补全前不能继续登录。
7. 未 consent 调 authorize 返回 `/consent?token=...`。
8. `prompt=login` 或 `max_age=0` 触发 `/login?reauth=1` 重新认证跳转，并在回跳目标中消费会造成循环的 reauth 参数；公网 smoke 对该行为做黑盒验证。
9. `prompt=consent` 对已有 disclosure consent 的授权请求仍返回 consent challenge；`prompt=login consent` 重新认证后保留 `prompt=consent`。
10. Consent 页显示应用信息、redirect host、scope 字段清单和当前登录身份。
11. 同意后写入 `open_platform_user_consents` 并跳回第三方 redirect。
12. 拒绝后跳回第三方 redirect 并附带 `error=access_denied`。
13. 已审批且已 consent 后 UserInfo / disclosure 只返回请求 scope 覆盖的字段。
14. `phone.read` 从 StuHelper 本地加密投影解密手机号；未绑定、投影缺失或不可解密时 fail-closed。
15. `resource.read` / `resource.write` 不进入用户 consent；资源访问只由 app scope approval 和 OpenFGA tuple 决定。
16. 管理员吊销 app 后，该 app 的 active user consent 全部撤销，用户授权列表不再显示该 app，用户授权活动记录能看到 lifecycle 来源的 revoke 审计；该 app 在 OpenFGA 中的资源授权 tuple 也会被删除，并写入 resource access lifecycle revoke 审计。
17. Legacy Casdoor app 导入前先检查已有 Casdoor `TokenFields`；出现手机号、学生认证、学校、身份类型等业务 claim 时拒绝导入 approved。
18. `qq.binding.read` 不在 OpenAPI scope enum 中。
19. 管理员授予 `resource_item` / `user_profile` 资源授权写入 OpenFGA tuple，授予和撤销都要求非空审计原因；第三方应用资源检查在 scope 未批准或 tuple 缺失时返回 `allowed=false`，OpenFGA 不可用时 fail-closed。

## 下一步

Open Platform 的下一步不是重做登录，而是补齐生产运营面：

| 目标 | 内容 | 完成标准 |
|------|------|----------|
| 用户授权管理 | 用户中心查看已授权应用、按 scope 或整应用撤销；授权页、资料补全页和用户授权列表展示每个 scope 的开发者用途说明；用户中心展示当前用户自己的授权活动记录 | 撤销后下一次 disclosure 立即返回 consent required，用户授权决策前后都能看到 scope 字段清单、用途说明和授权/披露活动摘要 |
| 开发者门户 | 已有应用列表、创建、资料维护、审核状态、scope 新增 / rejected 重提、redirect URI 变更申请、secret 轮换和自有应用审计摘要 | 开发者不需要直接进入 Casdoor 控制台，能自助排查应用展示资料、审批、回调、密钥、披露和资源授权问题 |
| 管理后台审核 | 已有 app / scope 批准与驳回 UI、redirect URI 变更审核、legacy Casdoor 导入表单、敏感 scope 复核、暂停 / 恢复 / 吊销、secret 轮换和审计查看 | 所有审核和生命周期动作落审计并可检索 |
| 生产准入探针 | 审批路径已创建 / 更新第三方 Casdoor app 并强制 `TokenFields=[]`；legacy 导入会拒绝业务 claim；审批前可强制 command-backed runtime code-flow 探针并把结果写入 `open_platform_token_probe_evidence`；管理后台可检索 evidence；`infra/ops/casdoor-token-minimization-probe.sh` 可输出 JSON evidence；backend 镜像内置 `/app/casdoor-runtime-token-probe-runner.mjs` 自动 runner；`prod-deploy.sh` 会在 app 启动前运行 `casdoor-runtime-token-probe-smoke.sh`；2026-05-24 生产已用专用低权限探针账号和 smoke app 验证 `businessClaims=[]` 且 `metadata.nonceVerified=true` | 发布门禁持续生成脱敏 evidence，探针失败时拒绝上线或审批 |
| 限流、指标与运营报表 | 已有 app、app+user、endpoint、consent 维度 Redis 限流、可配置阈值、低基数 Prometheus 指标、告警规则和管理端 disclosure 运营报表 | 生产根据实际流量调整阈值，保持低基数标签 |
| 审计硬化 | 已有 disclosure 成功 / 拒绝 / 超限 / 手机号读取失败 / 异常重放审计，审计失败 fail-closed | 生产保留审计写入失败 fail-closed 语义 |
| 密钥生命周期 | 已有 client secret 轮换、暂停、恢复、吊销、泄漏处置原因审计和审计查看 | 后续补齐更细的运营报表 |
| 资源授权运营面 | 后端已有 app 到具体资源的 OpenFGA tuple grant / revoke / list / check API，管理后台应用审核页可操作 grant / revoke / list；2026-05-24 已在生产 OpenFGA store/model 完成 grant / check / revoke smoke 并留档 | 发布门禁持续证明具体资源 tuple 授权与 scope consent 分层且 fail-closed |

## 参考

- IAM v2 主架构：[`iam-v2-casdoor.md`](iam-v2-casdoor.md)
- 认证与会话：[`auth-and-session.md`](auth-and-session.md)
- 授权模型：[`authorization-model.md`](authorization-model.md)
- 安全实践：[`security-model.md`](security-model.md)
