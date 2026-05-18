---
type: design
audience: maintainers, backend-dev, frontend-dev, product
status: current
authoritative-source: this file for open-platform v1 architecture; server/api/openapi.yaml for API contract
created: 2026-05-01
last-verified: 2026-05-18
related:
  - iam-v2-casdoor.md
  - auth-and-session.md
  - authorization-model.md
  - security-model.md
scope: 当前 Open Platform v1 baseline 与目标架构；第三方应用接入、用户授权、最小化披露
---

# StuHelper Open Platform v1

## 状态

当前 v1 baseline 已进入主线实现，不再是 deferred design。

已落地的能力：

- 第三方应用注册 API：`POST /api/v1/open-platform/apps`
- 管理员审批 API：`POST /api/v1/admin/open-platform/apps/{appID}/scopes/{scope}/approve`、`POST /api/v1/admin/open-platform/apps/{appID}/approve`
- 授权入口：`/connect/authorize` 前端页调用 `GET /api/v1/open-platform/authorize`
- 用户授权页：`/connect/consent?token=...` 前端页调用 `GET /api/v1/open-platform/consent`
- 用户同意 / 拒绝：`POST /api/v1/open-platform/consent/accept`、`POST /api/v1/open-platform/consent/deny`
- 披露 API：`GET /api/v1/open-platform/userinfo`、`/verification`、`/student`、`/phone`
- 数据模型：`open_platform_apps`、`open_platform_scope_requests`、`open_platform_approved_scopes`、`open_platform_user_consents`、`open_platform_audit_events`
- Casdoor 第三方 Application provisioning、OIDC authorize URL 生成、手机号实时读取 client

仍未宣称完成的目标：完整开发者门户、用户授权管理 / 撤销 UI、管理后台审核 UI、生产级限流与指标、密钥轮换 UI、第三方 token claim 最小化探针与生产准入门禁。下一步执行项记录在 [`docs/internal/exec-plans/active/current-project-open-items.md`](../internal/exec-plans/active/current-project-open-items.md)。

## 架构定位

Casdoor 与 StuHelper 的职责边界固定如下：

| 层 | 职责 | 不承担 |
|----|------|--------|
| Casdoor | 身份、注册、登录、OIDC/OAuth client registry、SMS / Email provider、用户手机号真相源、第三方 Application 创建 | StuHelper 业务 scope、学生认证事实、业务授权决策、第三方披露审计 |
| StuHelper Open Platform | 第三方应用注册、scope 申请、管理员审批、用户 consent、disclosure API、审计、撤销与限流 | 原始账号密码登录、直接替代 Casdoor OAuth |
| StuHelper DB | 实名认证、学生认证、学校归属、业务用户锚点、open_platform_* 状态 | Casdoor 账号主数据 |
| OpenFGA | 资源级关系授权；v1.1 起承载 app 到具体资源的关系 | scope consent、应用元数据、登录决策 |

第三方应用不能直接读取 Casdoor 用户 API。第三方获得的 Casdoor token 只作为最小身份凭据；手机号、用户名、头像、学生认证状态等业务字段只能通过 StuHelper disclosure API 返回。

## 完整目标

Open Platform 的完整目标是让校内外第三方应用以可审计、可撤销、最小化披露的方式接入 StuHelper 身份与认证事实：

1. 开发者提交应用、redirect URI、隐私政策、申请 scope 与用途说明。
2. 管理员按应用和敏感 scope 审批；通过后由 StuHelper 自动创建 Casdoor Application。
3. 用户在 StuHelper 授权页看到应用信息、当前登录身份、每个 scope 实际读取的字段，并选择允许或拒绝。
4. 第三方应用完成 Casdoor OIDC 登录后，只能通过已审批且已授权的 disclosure API 获取字段。
5. 用户可以查看和撤销已授权应用；管理员可以暂停、吊销、轮换密钥、审计异常访问。
6. 任何未审批、未授权、token 无效、redirect 不匹配、依赖不可用、审计失败的路径都 fail-closed。

## 非目标

- 不把学生认证、实名认证、学校归属、QQ 绑定等业务事实同步成第三方可直接读取的 Casdoor claim。
- 不让第三方应用直接读 Casdoor user API。
- 不在第三方 ID token / access token 中塞入手机号、邮箱、学校、认证状态等业务字段。
- 不把 `qq.binding.read` 放入默认 scope 目录。
- 不在 v1 开放第三方直接操作评课删除、用户封禁等管理员级资源。
- 不把 scope consent 建模进 OpenFGA；scope 是业务字符串属性，权威状态在 `open_platform_user_consents`。

## 登录与授权流程

```text
第三方应用
  │
  │ GET https://stuhelper.com/connect/authorize?client_id=...&redirect_uri=...&scope=...&state=...
  ▼
StuHelper Web /connect/authorize
  │
  │ GET /api/v1/open-platform/authorize
  ▼
StuHelper Open Platform
  ├─ 校验 app approved
  ├─ 校验 redirect_uri 精确匹配
  ├─ 校验 scope 合法且已审批
  ├─ 校验当前 StuHelper 用户是否已 consent
  │
  ├─ 未 consent: 返回 /connect/consent?token=...
  │             用户允许后写 open_platform_user_consents
  │
  └─ 已 consent: 生成 Casdoor OIDC authorize URL
                  Casdoor 只请求 openid
```

`/connect/consent` 是 StuHelper 页面，不依赖 Casdoor 内置 consent 表达业务 scope。Casdoor 仍负责注册、登录、会话和授权码签发。

## OIDC Scope 与业务 Scope

第三方应用的 Casdoor OIDC scope 固定为 `openid`。当前服务端 `casdoorOAuthScopes()` 会忽略业务 scope，只向 Casdoor 请求 `openid`。

业务 scope 由 StuHelper 自己维护：

| Scope | 字段 | 敏感度 | 当前策略 |
|-------|------|--------|----------|
| `profile.basic.read` | `username`、`displayName`、`avatar` | low | 需用户授权 |
| `email.read` | `email` | medium | 需用户授权 |
| `phone.read` | `phone`、`phoneMasked`、`phoneVerified` | high | 需应用审批 + 用户授权；手机号实时从 Casdoor 读取 |
| `stu.identity.status.read` | `identityVerified` | high | 需应用审批 + 用户授权 |
| `stu.identity.type.read` | `identityType` | high | 需应用审批 + 用户授权 |
| `stu.student.status.read` | `studentVerified` | high | 需应用审批 + 用户授权 |
| `stu.student.school.read` | `school.id`、`school.name` | high | 需应用审批 + 用户授权 |
| `resource.read` | 未来资源读取授权 | high | v1.1 设计 |
| `resource.write` | 未来资源写入授权 | very_high | v1.1 设计 |

默认目录不包含身份证号、学号、详细地址、出生日期和 QQ 绑定。

## Disclosure API

当前 disclosure API：

```text
GET /api/v1/open-platform/userinfo
GET /api/v1/open-platform/verification
GET /api/v1/open-platform/student
GET /api/v1/open-platform/phone
```

每个请求必须带：

- bearer/cookie 身份凭据，由 auth middleware 解析当前用户；
- `client_id` 对应已批准应用；
- `redirect_uri` 精确匹配该应用登记值；
- `scope` 空格分隔业务 scope；
- 可选 `consent_base_url`，用于返回 `/connect/consent?token=...`。

服务端决策顺序：

1. 标准化 scope，非法 scope 直接拒绝。
2. 按 `client_id` 找应用，应用必须是 `approved`。
3. 按 Casdoor subject 映射内部 `users.id`。
4. 检查应用已审批全部请求 scope。
5. 检查用户对该应用仍有 active consent。
6. 未授权时返回 consent required，并生成 5 分钟 Redis consent challenge。
7. 已授权时按 scope 查询最小字段并返回。

`phone.read` 是特殊路径：本地 `users` / profile 只保存手机号验证投影和脱敏展示需要的状态；完整手机号真相源是 Casdoor，披露时通过 Casdoor user profile client 实时读取并标准化为中国大陆手机号。

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
  created_at
  expires_at
```

安全约束：

- token 为服务端生成的高熵随机值，URL 中不直接暴露 app_id / user_id / scope。
- token 绑定当前内部用户；其他用户打开会被拒绝。
- `redirect_uri` 只能来自应用登记的精确 URI，不接受通配符、fragment 或运行时自由 return URL。
- 用户同意后写入 `open_platform_user_consents` 并删除 Redis token。
- 用户拒绝后删除 Redis token，并返回带 `error=access_denied` 的第三方 redirect URL。

当前 baseline 没有单独的 consent revoke API/UI，撤销能力属于下一步目标。

## 应用生命周期

```text
开发者提交应用
  → open_platform_apps.status = pending
  → open_platform_scope_requests.status = pending

管理员审批 scope
  → open_platform_scope_requests.status = approved
  → open_platform_approved_scopes 写入 app_id + scope

管理员审批应用
  → 生成一次性 client_secret
  → 调 Casdoor CreateApplication
  → open_platform_apps.status = approved
```

应用状态：

- `pending`：等待审核，不可授权。
- `approved`：可以进入授权和 disclosure。
- `suspended`：临时停用。
- `revoked`：永久吊销。

`OpenPlatformManage` capability（字符串：`open_platform:manage`）控制管理员审批接口。

## 数据模型

| 表 | 作用 |
|----|------|
| `open_platform_apps` | 应用元数据、client_id、client_secret_hash、redirect_uris、状态 |
| `open_platform_scope_requests` | 开发者申请的 scope 与用途说明、审核状态 |
| `open_platform_approved_scopes` | 已批准 scope 的当前集合 |
| `open_platform_user_consents` | 用户对 app + scope 的 active/revoked 授权事实 |
| `open_platform_audit_events` | 披露、consent、审批等事件留痕 |

Scope consent 不进 OpenFGA。未来资源 API 才会把具体资源关系写成 OpenFGA tuple，例如：

```text
review_draft:{id}#can_write_by_app @ open_platform_app:{app_id}
user_profile:{id}#can_read_by_app  @ open_platform_app:{app_id}
```

## 安全基线

- 第三方 OIDC token 只请求 `openid`。
- 生产准入必须实测第三方 token 不含 `preferred_username`、`name`、`picture`、`email`、`phone`、`studentVerified` 等业务 claim。
- redirect URI 必须精确匹配，禁止 wildcard、fragment、regex、通配子域、通配路径。
- 应用未批准、scope 未批准、用户未授权、审计不可用、Casdoor 手机号不可读时拒绝披露。
- 敏感 scope 审批必须有管理员留痕。
- Consent UI 必须逐项展示实际字段名，不能只写“基本信息”这类模糊文案。
- 第三方应用密钥只在批准或轮换时展示一次，服务端只保存 hash。

## 验证策略

当前 v1 baseline 的最小验证集：

1. 注册应用写入 pending app 与 pending scope request。
2. 管理员审批 scope 后写入 approved scope。
3. 管理员审批 app 后创建 Casdoor application 并只返回一次 client secret。
4. 未审批 app 或 scope 调 authorize/disclosure 被拒绝。
5. 未 consent 调 authorize 返回 `/connect/consent?token=...`。
6. Consent 页显示应用信息、redirect host、scope 字段清单和当前登录身份。
7. 同意后写入 `open_platform_user_consents` 并跳回第三方 redirect。
8. 拒绝后跳回第三方 redirect 并附带 `error=access_denied`。
9. 已审批且已 consent 后 disclosure 只返回请求 scope 覆盖的字段。
10. `phone.read` 实时从 Casdoor 读取手机号；未绑定或 Casdoor 不可用时 fail-closed。
11. `qq.binding.read` 不在 OpenAPI scope enum 中。

## 下一步

Open Platform 的下一步不是重做登录，而是补齐生产运营面：

| 目标 | 内容 | 完成标准 |
|------|------|----------|
| 用户授权管理 | 用户中心查看已授权应用、按 scope 或整应用撤销 | 撤销后下一次 disclosure 立即返回 consent required |
| 开发者门户 | 应用列表、创建、查看审核状态、密钥轮换、redirect URI 变更申请 | 开发者不需要直接进入 Casdoor 控制台 |
| 管理后台审核 | app / scope 审批 UI、敏感 scope 复核、暂停 / 吊销 | 所有审核动作落审计 |
| 生产准入探针 | 创建第三方 Casdoor app 后自动解码 token payload | token 出现业务 claim 时 app 不得 approved |
| 限流与指标 | app、app+user、endpoint、consent 维度限流；Prometheus 指标 | 超限返回 429 并有可观测标签 |
| 审计硬化 | disclosure、consent、拒绝、异常重放、手机号读取失败全量审计 | 审计失败 fail-closed |
| 密钥生命周期 | client secret 轮换、吊销、泄漏处置 | 旧 secret 立即失效且可追踪 |
| 资源 API v1.1 | app 到具体资源的 OpenFGA 关系 | 与 scope consent 分层，不混用 |

## 参考

- IAM v2 主架构：[`iam-v2-casdoor.md`](iam-v2-casdoor.md)
- 认证与会话：[`auth-and-session.md`](auth-and-session.md)
- 授权模型：[`authorization-model.md`](authorization-model.md)
- 安全实践：[`security-model.md`](security-model.md)
