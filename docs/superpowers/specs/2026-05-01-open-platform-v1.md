---
type: design
audience: maintainers, backend-dev, product
status: draft
authoritative-source: this file for open-platform v1 target
created: 2026-05-01
last-verified: 2026-05-02
prerequisites:
  - 2026-05-01-casdoor-iam-v2.md (must land first)
related:
  - 2026-05-01-casdoor-iam-v2.md
scope: 第三方应用接入与披露网关；IAM v2 落地前不启动实施
---

# StuHelper Open Platform v1

## 0. 状态

**deferred-design** — 设计已起草，**实施延后到 IAM v2 全部验证策略通过、Zitadel 完全退役之后**。

不要在 IAM v2 实施期间并行启动本文档的工程任务。原因：
- 开放平台依赖 IAM v2 的 `AuthorizationService` 单一入口、Casdoor Application 模型、OpenFGA `open_platform_app` 关系；
- 并行实施会让 IAM v2 范围继续膨胀，重蹈旧 spec（已删除）的覆辙。

## 1. 决策记录

- **业务来源**：项目 owner 明确要求第三方应用接入 sso.stuhelper.com，scope 审批后通过 API 获取最小化身份信息。
- **范围拆分**：从原 `2026-05-01-casdoor-open-platform-iam-design.md` 中剥离的开放平台部分独立成本文档；旧 spec 已被 [`casdoor-iam-v2.md`](./2026-05-01-casdoor-iam-v2.md) 取代。
- **网关定位**：开放平台是 StuHelper 业务侧的披露网关，Casdoor 仅承载应用身份和 OAuth token 签发。

## 2. 目标

1. 第三方应用通过 sso.stuhelper.com 完成 OIDC authorization code + PKCE 登录；
2. 开发者在 StuHelper 开放平台门户提交应用，申请 scope，由管理员审批；
3. 用户通过 consent 流程显式授权应用访问哪些 scope；
4. 第三方应用通过 StuHelper 开放平台 API 获取已审批且已授权的最小化身份信息；
5. 全链路审计、限流、用户 / 管理员吊销；
6. fail-closed：scope 未审批、用户未授权、token 无效、依赖不可用 → 拒绝。

## 3. 非目标

1. 不允许第三方应用直接读 Casdoor 原始用户 API；
2. 不在 ID Token 中暴露业务事实（手机号 / 学校 / 学籍状态 / 实名状态 / QQ）；
3. 不开放需要"管理员级"权限的资源（评课删除、用户封禁等）；
4. 不实现内部一方应用之间的 SSO 互通（一方应用直接信任 IAM v2，无需 consent）；
5. **不开放 `qq.binding.read` scope**（高熵关联标识符，除非有专门隐私评估，默认目录中不出现）。

## 4. 与 IAM v2 的依赖关系

| IAM v2 提供 | Open Platform v1 消费 |
|-------------|------------------------|
| Casdoor Application 模型（`third-party-*`） | 第三方应用注册、provisioning |
| `AuthorizationService.Authorize` 的 `Subject.AppID` 维度 | 应用调用披露 API 时识别调用方 |
| OpenFGA `open_platform_app` 关系 | 应用所属开发者、审批人 |
| 业务 DB 表 `open_platform_user_consent` | 用户对应用的 scope 授权记录（不在 OpenFGA 中建模，理由见 §13） |
| Casdoor SDK 的 `CreateApplication` / `UpdateApplication` 出口 | 审批通过后自动创建 Casdoor app |
| 业务事实 DB | scope 披露 API 的事实源 |
| 审计基础设施 | 披露 / consent / 撤销事件 |

## 5. 开发者应用生命周期

```text
1. 开发者通过 Casdoor 登录 StuHelper 主站
2. 进入开放平台门户提交应用：
   - 名称、描述、主页 URL、隐私政策 URL、回调 URL（精确）
   - 请求 scope 列表 + 数据用途说明
3. StuHelper 校验：redirect URI 精确白名单、scope 在白名单内、隐私政策 URL 可达
4. 管理员审核应用整体 + 每个敏感 scope 单独审批
5. 审批通过 → StuHelper 调 Casdoor SDK CreateApplication
6. StuHelper 保存：app 元数据、已批准 scope 列表、状态
7. 开发者通过一次性安全流程获取 client_id / client_secret
8. 应用上线后可被：暂停、密钥轮换、吊销
```

> 生产环境第三方应用**禁止**在 Casdoor 控制台手工创建。任何手工变更视为配置漂移，由 reconciler 检测告警。
>
> **Redirect 安全 gate**：Casdoor 前置网关必须对 `/login/oauth/authorize` 的 `client_id + redirect_uri` 做 StuHelper DB 精确白名单校验；拒绝 wildcard、通配子路径、通配子域、regex、scheme/userinfo 混淆、URL 编码绕过、双重编码绕过。若当前 Casdoor 版本存在未修复 open redirect advisory，则没有该网关校验不得开放第三方 OAuth。

## 6. Scope 目录

> 这是初始设计；正式实施前必须经隐私评估。

### 6.1 OIDC scope vs 业务 scope 边界（避免双轨泄漏）

**关键事实**：Casdoor 的 access token 与 ID token 共用同一份 JWT payload（参考 [Casdoor Token Overview](https://casdoor.org/docs/token/overview/)）。第三方 OIDC client 拿到 token 后即可读取 token 内的全部 claim——所以**任何写进 OIDC token 的字段就是已对该应用披露的字段**，不存在"OIDC 写了 email 但 email.read 还能控制披露"这种二次门槛。

为避免双轨泄漏，本 spec 采用**严格分离模型**：

| 应用类型 | 允许的 OIDC scope | token 实际包含 | 业务数据访问 |
|----------|-------------------|----------------|--------------|
| **一方应用**（web / admin / uniapp）| `openid profile email` | `sub / iss / aud / exp / iat / preferred_username / name / picture / email` | 受信内部应用，无 consent 流程；直接走 IAM v2 §5 Authorization Service |
| **第三方应用** | **仅 `openid`**（默认拒绝其它 OIDC scope）| `sub / iss / aud / exp / iat`（pseudonymous，无任何业务字段）| 必须通过 StuHelper disclosure API + 应用 scope 审批 + user consent；token 不暴露任何业务信息 |

**强制规则**：

1. 第三方应用注册时，Casdoor Application 配置**禁止**勾选 `profile` / `email` scope；
2. 第三方应用 token 中**绝不**出现 `preferred_username` / `name` / `picture` / `email`——即使应用尝试请求；
3. 业务 scope（下方目录）全部由 StuHelper disclosure API 控制，**不**注册到 Casdoor；
4. 应用必须显式调用 disclosure API 才能获得任何业务数据；token 本身只是 bearer credential。

**上线 gate（不可跳过）**：

- 每个第三方 Casdoor Application 创建后，必须用 `openid`-only authorization code + PKCE 实测并解码 access token / ID token；
- 若 token payload 出现 `preferred_username` / `name` / `picture` / `avatar` / `email` / `email_verified` / `phone` / `phone_verified` 任一字段，该应用不得进入 `approved` 状态；
- 若 Casdoor 当前版本无法对第三方应用做到上述 JWT claim 最小化，则第三方应用**不得**直接把 Casdoor JWT 作为 disclosure API bearer；必须改走 StuHelper token exchange / opaque disclosure session，由 StuHelper 签发只含 `sub` / `app_id` / `aud` / `exp` 的最小 token。

**第三方 `sub` 语义**：

- v1 默认只承诺 `sub` 是 opaque stable subject，不承诺跨应用不可关联的 pairwise subject；
- 若隐私评估要求 pairwise subject，StuHelper token exchange 层生成 `pairwise_sub = HMAC(server_secret, casdoor_subject || app_id)`，不把原始 Casdoor subject 暴露给第三方。

### 6.2 业务 scope 目录

| Scope | 暴露字段 | 敏感等级 | 默认状态 |
|-------|----------|----------|----------|
| `profile.basic.read` | 披露给第三方应用：昵称、头像 | 低 | 默认开放 |
| `email.read` | 披露给第三方应用：邮箱 | 中 | 需用户授权 |
| `phone.read` | 已绑定手机号 + 验证时间 | 高 | 需应用审批 + 用户授权 |
| `stu.identity.status.read` | 实名认证状态布尔 | 高 | 需应用审批 + 用户授权 |
| `stu.identity.type.read` | 身份类型（student / staff / other） | 高 | 需应用审批 + 用户授权 |
| `stu.student.status.read` | 学生认证状态布尔 | 高 | 需应用审批 + 用户授权 |
| `stu.student.school.read` | 学校 ID + 名称 | 高 | 需应用审批 + 用户授权 |
| `resource.read` | 用户授权资源读取 | 高 | 按资源类型细化 |
| `resource.write` | 用户授权资源写入 | 很高 | 仅特殊审批 |

> 上述业务 scope 与 OIDC scope **正交**——第三方应用拿到 OIDC `openid`-only token 后，需要再次走 §7 consent flow 获取业务 scope 的用户授权，才能调 disclosure API 获取对应字段。

**默认目录中不包含**：
- `qq.binding.read` — 高关联标识符，除非有明确业务场景与隐私评估
- 任何暴露身份证号、学号、详细地址、出生日期的 scope

## 7. 用户授权（Consent）

> **设计原则**：StuHelper 不在 OIDC authorization code flow 内部插入自己的 consent 页（这要求 Casdoor 提供非标准的授权扩展点，不是开箱能力）。改为 Google "incremental authorization" 模式：Casdoor 只处理应用类型允许的 OIDC 标准 scope（一方应用可 `openid profile email`，第三方应用仅 `openid`）；StuHelper-specific scope（`profile.basic.read`、`email.read`、`stu.*`、`phone.read` 等）由披露 API 在 token 之外二次校验，未 consent 时返回 403 + `consent_url`，前端跳转 StuHelper consent UI 完成授权后重试。

### 7.1 流程

```text
[OIDC 阶段]
应用发起 OIDC 授权（**第三方应用仅请求 `openid`**；一方应用可请求 `openid profile email`，详见 §6.1 严格分离模型）
    │
    ▼
Casdoor 完成登录 + 内置 consent 页（按 §6.1 应用类型限制 OIDC scope；第三方仅 `openid`）
    │
    ▼
Casdoor 颁发 access token（**第三方应用 token 仅 pseudonymous：sub/iss/aud/exp/iat，无任何业务字段**；不含 `stu.*` 业务 scope；任何业务数据访问必须走下方 disclosure API）
    │
    ▼
[披露阶段]
应用调用 StuHelper 开放平台披露 API
（如 GET /api/v1/open-platform/me/student）
    │
    ▼
StuHelper 校验：调用方应用是否已审批该 scope
    │   未审批 → 403 + body { error: "scope_not_approved" }
    ▼
StuHelper 校验：用户是否已对该应用 consent 该 scope
    │   未 consent → 403 + body { error: "consent_required",
    │                               consent_url: "https://stuhelper.com/open-platform/consent?token={consent_token}" }
    ▼
[首次 consent 阶段，按需触发]
应用前端跳转 consent_url
    │
    ▼
StuHelper consent UI：列出 scope + 数据用途 + 已绑定/未绑定状态
    │
    ▼
用户勾选 + 确认 → StuHelper 写 open_platform_user_consent 行
    │
    ▼
跳回应用预设的 return_url
    │
    ▼
应用重试披露 API → 返回最小化字段
```

### 7.2 关键设计点

1. **Casdoor 内置 consent 仅处理应用类型允许的 OIDC 标准 scope**：一方应用可使用 `openid` / `profile` / `email`；第三方应用只允许 `openid`，禁止 `profile` / `email`，任何业务字段必须通过 disclosure API + StuHelper consent 获取。StuHelper 不向 Casdoor 注册 `profile.basic.read` / `email.read` / `stu.*` 等业务 scope。
2. **StuHelper 特有 scope 是 StuHelper 业务概念**，不进 OAuth scope 协议层。披露 API 在 token 之外查询 `open_platform_user_consent` 表决定是否放行。
3. **应用 scope 审批**（管理员侧）与**用户 scope consent**（用户侧）是两个独立检查，缺一不可。
4. **Consent UI 必须列出每个 scope 实际暴露的数据字段**（参考 §14.7），不能用模糊措辞。
5. **撤销立即生效**：用户撤销 consent 后，下一次披露 API 调用即返回 403。

### 7.3 Consent 数据

```text
app_id
user_id
scope
granted_at
revoked_at      (NULL when active)
grant_source    (web / mobile / api)
request_id      (审计关联)
```

**用户管理 UI**：必须能查看所有授权应用、按 scope 撤销、按整应用撤销。撤销在 StuHelper 网关层立即生效（无 token 缓存延迟）；同时调 Casdoor token 吊销端点（针对该 app 的 refresh token）以阻止应用继续刷新 token 重试。

### 7.4 Consent flow 安全约束

`consent_url` 与 `return_url` 跳转链路必须满足以下约束（防伪造、防 replay、防 open redirect）：

1. **服务端 consent_token**：
   - 披露 API 在返回 403 `consent_required` 时，由服务端生成一次性不可猜测 `consent_token`（>=128 bit 随机熵）；
   - token 绑定 `(app_id, user_id, requested_scopes, redirect_target, expires_at)`，存 Redis；
   - TTL ≤ 5 分钟；过期或消耗后立即销毁；
   - `consent_url` 形如 `https://stuhelper.com/open-platform/consent?token={consent_token}`，不在 URL 直接暴露 app_id / scope 等敏感参数。

2. **redirect_target 严格白名单**：
   - 应用**不得**通过 query / body 自由传入 `return_url`；
   - 实际跳转目标由服务端从 `consent_token` 读取；
   - 必须**精确匹配**该 app 在 StuHelper `open_platform_app.redirect_uris` 注册的 redirect URI 之一；
   - 不信任 Casdoor 自身的宽松 redirect 匹配；Casdoor 前置网关同样执行 `client_id + redirect_uri` 精确白名单；
   - 拒绝任何 wildcard、通配子路径或子域名匹配。

3. **一次性使用**：
   - consent UI 提交后立即销毁 `consent_token`；
   - 同一 token 第二次提交一律拒绝；
   - 防止 replay 攻击。

4. **CSRF + nonce 双层保护**：
   - consent UI 表单含 CSRF token（与会话绑定）；
   - 表单中嵌入 nonce，提交时与 Redis 中绑定值比对；
   - 双层失败任一立即 403。

5. **同源限定**：
   - consent UI 与披露 API 同根域（`stuhelper.com`），cookie 自然隔离第三方 origin；
   - consent UI 设 `X-Frame-Options: DENY` + `Content-Security-Policy: frame-ancestors 'none'`，防 clickjacking。

6. **登录态校验**：
   - consent UI 入口必须已登录到 Casdoor session；
   - 未登录时跳转标准登录链路并在登录后回跳 consent；
   - 不允许匿名用户触发 consent。

7. **审计**：
   - `consent_token` 颁发、消耗、过期均落 `open_platform_audit_event`（含 request_id、app_id、user_id、scope、decision）；
   - 异常事件（多次提交、过期重试、非法 return_url）单独打 metric 触发告警。

8. **限流**：
   - 同一 user × app 的 consent 触发频率受限（建议 ≤ 5 次/分钟）；
   - 防止恶意应用通过反复触发 consent 制造骚扰。

## 8. 披露 API

```text
GET  /api/v1/open-platform/userinfo
GET  /api/v1/open-platform/me/verification
GET  /api/v1/open-platform/me/student
GET  /api/v1/open-platform/me/phone
POST /api/v1/open-platform/consents/{appID}/revoke
```

**每个端点的强制流程**：

1. 校验 Casdoor access token；
2. 解析调用方 `app_id`；
3. 查询应用已批准 scope 列表；
4. 查询用户对该应用的 consent；
5. 查询业务事实 DB（按 scope 决定字段）；
6. 写审计事件（含 request_id / app_id / user_id / scope / decision）；
7. **只返回 scope 覆盖的字段**，最小化披露。

**响应示例 — `stu.student.status.read` 单 scope**：

```json
{
  "studentVerified": true
}
```

**同时持有 `stu.student.status.read` + `stu.student.school.read`**：

```json
{
  "studentVerified": true,
  "school": {
    "id": 10006,
    "name": "北京航空航天大学"
  }
}
```

**`phone.read`**：

```json
{
  "phone": "+8613800000000",
  "phoneVerified": true,
  "verifiedAt": "2026-05-01T00:00:00Z"
}
```

## 9. 应用访问资源（前瞻设计）

资源 API 通过 OpenFGA 表达"应用 + 用户 + 资源"三方授权关系：

```text
app:calendar       can_read  user_profile:123
app:course-tool    can_read  resource:456
app:writer-tool    can_write review_draft:789
```

Casdoor 决定**应用身份**（第三方应用 OIDC token 仅 pseudonymous `sub`，**不含** `profile` / `email` 等任何业务字段，参见 §6.1 严格分离模型）；**StuHelper DB 决定 app scope approval（管理员审批）与 user consent**（见 §6 / §7）；**StuHelper Authorization Service** 组合判断；OpenFGA 决定**该应用是否被该用户授权访问该具体资源**。

详细资源 API 设计推迟到 v1.1。

## 10. 限流

| 维度 | 限流策略（建议初值） |
|------|---------------------|
| 应用全局 | 1000 req/min（按 app 信誉调整） |
| 应用 + 用户 | 60 req/min |
| 应用 + 端点 | 按端点敏感度细化 |
| OAuth token endpoint | 30 req/min per app |
| Consent endpoint | 10 req/min per user |

超限返回 429 + `Retry-After` header。审计记录被限流的应用 / 用户 / 端点。

## 11. 审计

每次披露 / consent / 撤销事件落 `open_platform_audit_event` 表：

```text
id
app_id
user_id
scope
endpoint
decision (allow / deny / rate_limited)
request_id
client_ip
created_at
```

保留期建议 1 年；敏感 scope（phone / identity）的审计记录 3 年。

## 12. 失败语义

| 失败 | 决策 |
|------|------|
| Casdoor token 校验失败 | 401 |
| 应用 scope 未审批 | 403 |
| 用户未 consent | 403 + 引导 consent flow |
| 应用被暂停 / 吊销 | 403 |
| 审计写入失败 | 拒绝披露（fail-closed） |
| 业务 DB 不可用 | 503 |
| OpenFGA 不可用（资源 API） | 503 |
| 限流命中 | 429 |
| Casdoor 不可用（token introspection） | 503（按 IAM v2 §10 规则） |

## 13. 数据模型

```text
open_platform_app
  id
  casdoor_application_name
  owner_user_id
  display_name
  description
  homepage_url
  privacy_policy_url
  redirect_uris (1..N)
  status  (pending / approved / suspended / revoked)
  created_at, updated_at

open_platform_scope_request
  id
  app_id
  scope
  reason
  status  (pending / approved / rejected)
  reviewer_user_id
  reviewed_at
  decision_note

open_platform_approved_scope
  app_id
  scope
  approved_at
  approved_by

open_platform_user_consent
  app_id
  user_id
  scope
  granted_at
  revoked_at
  grant_source
  request_id

open_platform_audit_event
  (见 §11)
```

OpenFGA tuple（仅 `open_platform_app` 类型；scope consent **不**进 OpenFGA）：

```text
open_platform_app:{app_id}#developer    @ user:{owner_user_id}
open_platform_app:{app_id}#approved_by  @ user:{admin_user_id}
```

**为什么 scope consent 不进 OpenFGA**：

- OpenFGA tuple 的 object 必须是声明了 type 的实体；scope（如 `phone.read`）是字符串属性，不是实体；
- 强行用 `string:{scope_name}` 这类伪类型会撞 OpenFGA DSL 语法；
- 用 `scope_consent:{app_id}_{user_id}_{scope}` 这种合成 ID 反 ReBAC 设计本意；
- Scope consent 由 DB 表 `open_platform_user_consent` 承载（见 §13 数据模型），由 `AuthorizationService` 在决策时查询。

**应用 → 资源**关系（v1.1，不在 v1 范围）才进 OpenFGA：

```text
# 形如：
review_draft:{id}#can_write_by_app  @ open_platform_app:{app_id}
user_profile:{id}#can_read_by_app   @ open_platform_app:{app_id}
```

具体 type 与 relation 在 IAM v2 [`§7.1`](./2026-05-01-casdoor-iam-v2.md) 已为 `open_platform_app` 预留 type 定义。

## 14. 安全与隐私基线

承袭 IAM v2 §14，并加：

1. 第三方应用最小权限默认值；
2. 敏感 scope 审批必须管理员复核 + 留痕；
3. 应用密钥泄漏的处置流程（自动检测 + 强制轮换）；
4. 用户撤销 30 天内仍保留审计记录（用于争议处理）；
5. 应用 redirect URI 变更必须重新审批；
6. 即使开发者本人是已认证学生，其应用仍按外部应用处理；
7. consent 页面文案必须明确列出每个 scope 的实际数据字段（不能用模糊措辞）。

## 15. 验证策略

实施时至少覆盖：

1. 应用注册 + 审批 happy path；
2. 未审批应用调披露 API → 403；
3. 已审批 scope 但用户未 consent → 403 + 引导 consent；
4. 已审批 + 已 consent → 返回最小化字段；
5. 用户撤销 consent 后立即生效（5 秒内）；
6. 应用被吊销后立即拒绝；
7. 限流触发 429；
8. 审计写入失败时 fail-closed；
9. consent 页面文案与实际披露字段一致；
10. `qq.binding.read` 不在默认 scope 目录；
11. `phone.read` 同时检查 scope 审批 / 用户 consent / 手机号绑定 / 审计；
12. 资源 API（v1.1）：app 必须经 OpenFGA `can_read user_profile:{id}` tuple 才能访问。

## 16. 实施前置条件（强制）

启动 v1 工程任务前必须满足：

- [ ] IAM v2 §15 全部 15 条验证策略通过；
- [ ] Zitadel 容器、infra、env、code 完全退役；
- [ ] `AuthorizationService.Authorize` 的 `Subject.AppID` 维度已上线并被一方应用消费；
- [ ] OpenFGA 模型包含 `open_platform_app` 类型定义（IAM v2 已写入定义但未消费）；
- [ ] 业务 DB 已迁移 `open_platform_user_consent` 表 schema；
- [ ] 隐私评估覆盖 §6 scope 目录；
- [ ] 限流基础设施可承载新增端点。

## 17. 后续实施阶段（设计阶段，不立即执行）

1. 数据模型 schema migration；
2. 开放平台门户前端（开发者侧）；
3. 管理员审批后台（接入现有 admin）；
4. Consent UI（用户侧）；
5. 披露 API 实现 + 单元 / 集成测试；
6. 审计基础设施集成；
7. 限流策略接入；
8. 资源 API（v1.1，独立后续）；
9. 文档与开发者门户上线。

## 18. 参考

- IAM v2 主架构：[`2026-05-01-casdoor-iam-v2.md`](./2026-05-01-casdoor-iam-v2.md)
- Casdoor Application：<https://casdoor.org/docs/application/overview>
- Casdoor OAuth：<https://casdoor.org/docs/how-to-connect/oidc-client/>
- OAuth 2.0 Threat Model and Security Considerations：<https://datatracker.ietf.org/doc/html/rfc6819>
- OpenID Connect Core：<https://openid.net/specs/openid-connect-core-1_0.html>
