---
type: design
audience: maintainers, backend-dev, infra-dev, platform-dev, security-review
status: proposed
authoritative-source: this file for the target IAM and open-platform architecture
created: 2026-05-01
scope: greenfield target architecture; no Zitadel compatibility; no requirement to reuse current package layout
---

# Casdoor 身份平台与开放平台设计

## 1. 决策摘要

StuHelper 应迁移到以 Casdoor 为中心的身份架构，但不能把 Casdoor 强行当成所有授权问题的唯一解。

目标架构分为四层：

| 层 | 权威来源 | 职责 |
|----|----------|------|
| Casdoor | 身份与应用级授权权威 | SSO、OIDC/OAuth 客户端、登录方式、短信、邮件、MFA、用户、组织、应用、角色、权限、开发者应用身份 |
| StuHelper 业务数据库 | 业务事实权威 | 实名认证、学生/教职工身份类型、学校归属、手机号绑定、QQ 绑定、课程/评课/资源归属与状态 |
| OpenFGA | 资源关系授权权威 | 用户/应用/资源之间的关系，如 owner、author、具体资源的学校管理员、举报处理人、应用经用户授权访问资源 |
| StuHelper 开放平台 | 策略决策与数据披露网关 | 应用审批、scope 审批、用户授权、身份信息 API、审计、限流、吊销、字段最小化 |

这是一次绿地目标架构设计，不是兼容迁移。现有 Zitadel 用户、Zitadel 专用 claims、Zitadel 基础设施和 `ZITADEL_*` 配置全部退出。`sso.stuhelper.com` 由 Casdoor 承载。

## 2. 目标

1. 为 StuHelper 一方应用和第三方开发者应用提供统一 SSO。
2. 优先使用 Casdoor 内置 IAM 能力：用户、组织、应用、Provider、角色、权限、OAuth/OIDC、短信、SMTP、token 管理、SDK 管理 API。
3. 支持细颗粒度管理权限：全局管理员、学校管理员、学校某板块管理员、内容管理员、审核员以及未来的角色变体。
4. 支持业务访问规则，例如“只有完成实名、学生认证且学校为 10006 的学生才能查看某板块完整内容”。
5. 支持“本人或管理员”资源操作：内容作者可以修改自己的内容，具备对应学校或板块权限的管理员可以管理资源。
6. 支持开放平台：第三方应用申请 scope，审批通过后通过受控 API 获取身份信息。
7. 对敏感身份事实实施最小化披露、应用审批、用户授权、审计、吊销和限流。
8. 失败必须显式暴露并 fail closed。Casdoor、OpenFGA 或业务事实存储无法回答受保护授权问题时，不允许静默放行。

## 3. 非目标

1. 不保留 Zitadel 兼容。
2. 不做双 IdP 模式。
3. 不迁移旧 Zitadel 用户或旧 `users.external_id` 映射。
4. 不把旧静态角色展开作为授权权威回退路径。
5. 不允许第三方应用直接读取 Casdoor 原始用户详情来获取 StuHelper 业务事实。
6. 默认不把手机号、学生状态、学校 ID、身份类型、QQ、实名状态或证件派生事实写入 ID Token。
7. 不把 Casdoor 当成每条评课、回复、举报、资源文件的高基数业务关系图数据库。

## 4. 总体架构

```text
一方应用
  web / admin / uniapp / Koishi console
        |
        | OIDC authorization code + PKCE
        v
Casdoor at sso.stuhelper.com
        |
        | ID token / access token
        v
StuHelper API
        |
        | 验证 token、解析主体、检查 Casdoor 权限
        | 检查业务事实、检查 OpenFGA 资源关系
        v
业务 API

第三方应用
        |
        | OIDC authorization code + PKCE
        v
Casdoor at sso.stuhelper.com
        |
        | access token
        v
StuHelper 开放平台 API
        |
        | 校验应用、审批 scope、用户授权、
        | 业务事实、OpenFGA 资源关系
        v
最小化身份或资源响应
```

关键边界是身份管理、业务事实和资源关系分离。Casdoor 要深度集成，但只承载它适合承载的 IAM 域。

## 5. Casdoor 职责

Casdoor 是身份平台权威。

### 5.1 用户与登录方式

Casdoor 管理账号和登录方式：

- 密码登录；
- 腾讯云短信验证码登录；
- SMTP 邮件验证、找回和通知；
- Casdoor 支持的 MFA；
- 未来接入的社交登录或校园身份源；
- OIDC/OAuth 会话和 token。

StuHelper 不再实现并行的 `/auth/phone/*` 手机号登录链路。手机号绑定仍属于 StuHelper 业务功能，但手机号验证码登录归 Casdoor。

### 5.2 组织与应用

Casdoor 组织和应用需要显式建模：

| Casdoor 对象 | 用途 |
|--------------|------|
| Organization `stuhelper` | StuHelper 用户命名空间 |
| Application `stuhelper-web` | 主站 Web/H5 登录 |
| Application `stuhelper-admin` | 管理后台登录 |
| Application `stuhelper-uniapp` | Native/mobile OIDC 流程 |
| Application `stuhelper-koishi-console` | 机器人控制台登录，如需独立 |
| Application `third-party-*` | 审批通过后的开发者应用 |

每个应用必须配置精确 redirect URI、允许的 grant type、token TTL 和启用 Provider。生产环境第三方应用不得手工创建；必须通过 StuHelper 开放平台审批工作流和 Casdoor Go SDK 创建。

### 5.3 角色

Casdoor 角色是用户角色分配的权威来源。建议角色语义如下：

```text
stuhelper/user
stuhelper/verified_student
stuhelper/super_admin
stuhelper/school_admin/10006
stuhelper/school_section_admin/10006/course_review
stuhelper/school_section_moderator/10006/course_review
stuhelper/school_section_reviewer/10006/student_verification
```

实际对象名称可以按 Casdoor 约束规范化，但语义必须保持显式。学校和板块出现在角色语义中是有意设计，用于表达管理层级，避免依赖隐藏应用状态。

### 5.4 权限与 Casbin Enforcement

Casdoor permissions 承载 IAM 级授权，覆盖：

- 管理后台入口；
- 后台功能权限；
- 学校级功能权限；
- 板块级功能权限；
- 开发者应用管理；
- 开放平台应用审批；
- token 和 consent 管理。

示例：

| Permission object | Action | 授权角色 |
|-------------------|--------|----------|
| `admin/dashboard` | `view` | `super_admin`，选定学校角色 |
| `school/10006/course-review` | `moderate` | `school_admin/10006`、`school_section_moderator/10006/course_review` |
| `school/10006/student-verification` | `review` | `school_admin/10006`、`school_section_reviewer/10006/student_verification` |
| `open-platform/apps` | `approve` | `super_admin` |
| `open-platform/scopes/phone.read` | `approve` | `super_admin` |

StuHelper 通过 Casdoor SDK 的 `Enforce` 或 `BatchEnforce` 做检查。列表页和批量操作应使用批量检查；缓存只能是请求级或短 TTL 的派生结果，不能成为授权事实源。

### 5.5 Provider

Bootstrap 必须创建并校验：

- 腾讯云 SMS Provider，用于手机号验证码登录；
- SMTP Email Provider，用于邮件验证、找回和通知；
- 未来配置的社交或校园 Provider。

生产环境缺少必要 Provider 时启动失败。开发环境可显式关闭 Provider，但必须通过环境变量和日志明确暴露。

## 6. StuHelper 业务事实职责

StuHelper 仍然是业务事实权威。这些事实不能只塞进 Casdoor，也不能只靠角色推断。

| 事实 | 权威来源 |
|------|----------|
| 实名认证是否通过 | StuHelper 认证域 |
| 身份类型是学生、教职工或其他 | StuHelper 认证域 |
| 学生认证是否通过 | StuHelper 认证域 |
| 学生学校 ID 是否为 10006 | StuHelper 认证域 |
| 手机号绑定值和验证时间 | StuHelper 用户域 |
| QQ 绑定 | StuHelper 用户域 |
| 课程属于哪个学校 | StuHelper 课程域 |
| 评课、举报、资源的 owner | StuHelper 业务表与 OpenFGA |
| 学校是否开放某板块访问 | StuHelper 系统配置 |

Casdoor 可以保存同步投影，例如 `verified_student` 或 `school_admin/10006`，但投影不是原始证据。业务认证状态变更后，StuHelper 通过同步任务更新 Casdoor 角色和权限。

## 7. OpenFGA 职责

OpenFGA 应保留。它负责资源关系授权。

Casdoor 回答：

```text
这个人是否具备管理 10006 学校评课板块的资格？
这个应用是否被批准申请 phone.read？
这个用户是否有管理后台入口权限？
```

OpenFGA 回答：

```text
这个用户是否是 review r123 的作者？
这个用户是否能删除 review r123，因为他是作者或所属学校内容管理员？
这个应用是否能读取某个用户授权过的具体资源？
这个管理员是否能查看 profile p456 的实名信息？
```

建议资源关系模型：

```text
ecosystem
  super_admin

school
  parent ecosystem
  admin
  section_admin:{section}
  section_moderator:{section}

course
  school
  owner
  teaching_assistant

review
  course
  school
  section
  author
  can_edit = author
  can_delete = author or section_moderator from section or admin from school
  can_hide = section_moderator from section or admin from school

user_profile
  owner
  school
  can_view_identity = owner or admin from school or section_reviewer from section

open_platform_app
  owner developer
  approved_by admin

app_consent
  app
  user
  scope
```

这可以从当前 OpenFGA 模型演进，但最终实现不受现有文件布局限制。

## 8. 授权决策模式

### 8.1 仅实名北航学生可查看某板块

需求：

```text
只有完成实名认证、
学生认证通过、
身份类型为学生、
学校为 10006
的人才能查看某板块完整内容。
```

决策流程：

```text
1. 验证 Casdoor token。
2. 通过 Casdoor subject 解析 StuHelper 用户。
3. 用 Casdoor permission 检查用户是否有该板块 view 权限。
4. 读取 StuHelper 业务事实：
   identity_verified = true
   identity_type = student
   student_verified = true
   school_id = 10006
5. 读取板块策略：
   section 允许学校 10006 访问
6. 全部通过才放行。
```

Casdoor 不能单独完成这个判断，因为实名和学生认证证据的权威来源是 StuHelper。

### 8.2 本人或管理员可修改资源

需求：

```text
只有资源 owner 和被授权管理员能修改内容资源。
```

决策流程：

```text
1. 验证 Casdoor token。
2. 从 StuHelper 加载资源元数据：
   resource_id, owner_user_id, school_id, section_id。
3. 检查 OpenFGA：
   user 是资源 owner/author。
4. 如果不是 owner，检查 Casdoor permission：
   user 具备 school/section/action 的管理权限。
5. 检查 OpenFGA：
   admin 关系适用于这个具体资源。
6. owner 路径或 admin 路径成功则放行。
```

简单 owner-only 操作可以在同一事务内直接查数据库；涉及跨域资源关系或共享资源时优先使用 OpenFGA。

### 8.3 全局、学校、板块管理员

| 管理类型 | Casdoor | OpenFGA | StuHelper DB |
|----------|---------|---------|--------------|
| 全局管理员 | 全局 role 和 permission | ecosystem `super_admin` 投影 | 审计元数据 |
| 学校管理员 | `school_admin/10006` role 和 permissions | school `admin` 投影 | 学校存在且启用 |
| 板块管理员 | `school_section_admin/10006/course_review` role 和 permissions | section relation 投影 | 板块配置 |
| 学生认证审核员 | section reviewer permission | profile relation check | profile 状态和学校 |

Casdoor 是管理员分配的操作入口和 API。OpenFGA 接收投影，用于具体资源检查。

## 9. 开放平台

开放平台必须是一等业务域。

### 9.1 开发者应用生命周期

```text
1. 开发者通过 Casdoor 登录。
2. 开发者提交应用：
   名称、描述、主页、回调地址、隐私政策、
   请求 scope、数据用途说明。
3. StuHelper 校验 redirect URI 和请求 scope。
4. 管理员审核应用和每个敏感 scope。
5. 审批通过后，StuHelper 通过 Casdoor SDK 创建或更新 Casdoor Application。
6. StuHelper 保存已批准 scope 和应用状态。
7. 开发者通过安全的一次性流程获取 client_id/client_secret。
8. 应用可以被暂停、轮换密钥或吊销。
```

第三方应用创建必须通过 StuHelper 和 Casdoor SDK 自动化。生产环境中 Casdoor 控制台手工变更视为配置漂移。

### 9.2 Scope 目录

初始 scope 建议如下：

| Scope | 暴露数据 | 敏感等级 |
|-------|----------|----------|
| `openid` | 稳定 subject | 低 |
| `profile.basic.read` | 昵称、头像 | 低 |
| `email.read` | 邮箱 | 中 |
| `phone.read` | 绑定手机号 | 高 |
| `stu.identity.status.read` | 实名认证状态 | 高 |
| `stu.identity.type.read` | 学生、教职工等身份类型 | 高 |
| `stu.student.status.read` | 学生认证状态 | 高 |
| `stu.student.school.read` | 学校 ID 和学校名 | 高 |
| `stu.qq.binding.read` | QQ 绑定状态或 QQ ID | 高 |
| `resource.read` | 用户授权资源读取 | 高 |
| `resource.write` | 用户授权资源写入 | 很高 |

敏感 scope 必须满足：

- 应用审批；
- 用户授权；
- 审计日志；
- 限流；
- 吊销能力；
- 响应字段最小化。

### 9.3 用户授权

Casdoor 提供登录和应用身份。StuHelper 负责 StuHelper 业务数据的用户授权。

授权记录：

```text
app_id
user_id
scope
granted_at
revoked_at
grant_source
request_id
```

用户必须能查看和撤销应用授权。撤销在 StuHelper API 层立即生效。应用暂停或泄露时再调用 Casdoor token/app 吊销能力。

### 9.4 身份信息披露 API

第三方应用通过 StuHelper API 获取 StuHelper 身份事实，不直接读取 Casdoor 原始用户 API。

示例端点：

```text
GET /api/v1/open-platform/userinfo
GET /api/v1/open-platform/me/verification
GET /api/v1/open-platform/me/student
GET /api/v1/open-platform/me/phone
GET /api/v1/open-platform/me/qq-binding
POST /api/v1/open-platform/consents/{appID}/revoke
```

每个端点都必须：

1. 校验 Casdoor access token；
2. 解析调用方 Casdoor application；
3. 检查应用已批准 scope；
4. 检查用户授权；
5. 检查业务事实；
6. 写审计事件；
7. 只返回 scope 覆盖的字段。

`stu.student.status.read` 示例响应：

```json
{
  "studentVerified": true
}
```

同时批准 `stu.student.school.read` 后才返回学校：

```json
{
  "studentVerified": true,
  "school": {
    "id": 10006,
    "name": "北京航空航天大学"
  }
}
```

`phone.read` 示例响应：

```json
{
  "phone": "+8613800000000",
  "phoneVerified": true,
  "verifiedAt": "2026-05-01T00:00:00Z"
}
```

### 9.5 应用访问资源

资源 API 使用 OpenFGA 管理应用、用户、资源之间的关系：

```text
app:calendar can_read user_profile:123
app:course-tool can_read resource:456
app:writer-tool can_write review_draft:789
```

Casdoor 决定应用身份和已批准 scope。OpenFGA 决定该应用是否在用户授权后能访问具体资源。

## 10. Token 与 Claim 策略

ID Token 保持最小：

```text
sub
iss
aud
exp
iat
preferred_username
name
picture
email 仅在低风险且一方应用需要时提供
```

默认不写入：

```text
phone
real-name status
student status
school ID
identity type
student ID
QQ ID
document-derived identity facts
```

Access Token 是 bearer credential。第三方应用只能用它调用 StuHelper 开放平台 API。StuHelper 根据已审批 scope 和用户授权决定披露内容。

## 11. 数据模型

最终实现可以使用新表或新 schema。以下为概念模型。

### 11.1 身份投影

```text
identity_user_projection
  id
  casdoor_subject
  casdoor_name
  display_name
  avatar_url
  email
  created_at
  updated_at
```

### 11.2 认证事实

```text
identity_verification
  user_id
  status
  identity_type
  real_name_verified_at
  document_type
  person_uid
  updated_at

student_verification
  user_id
  status
  school_id
  active_student_id
  verified_at
  verification_method
  updated_at
```

### 11.3 开放平台

```text
open_platform_app
  id
  casdoor_application_name
  owner_user_id
  display_name
  homepage_url
  privacy_policy_url
  status
  created_at
  updated_at

open_platform_redirect_uri
  app_id
  redirect_uri

open_platform_scope_request
  id
  app_id
  scope
  reason
  status
  reviewer_user_id
  reviewed_at

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

open_platform_audit_event
  id
  app_id
  user_id
  scope
  endpoint
  decision
  request_id
  created_at
```

### 11.4 授权投影任务

```text
iam_sync_job
  id
  stream
  dedupe_key
  payload
  status
  attempt_count
  next_attempt_at
  last_error
  created_at
  updated_at
```

该 outbox 同步 StuHelper 事实到 Casdoor roles/permissions 和 OpenFGA tuples。任务必须可重试，失败必须可观测。

## 12. SDK 与服务边界

建议包边界如下：

```text
server/internal/platform/casdoor/
  client.go          # SDK 初始化和健康检查
  applications.go    # 应用 CRUD 和漂移检查
  users.go           # 用户查询和投影辅助
  roles.go           # 角色分配辅助
  permissions.go     # enforce 和 batch enforce
  providers.go       # SMS/SMTP provider 检查
  bootstrap.go       # 幂等 bootstrap 校验

server/internal/platform/authorization/
  decision.go        # 组合 Casdoor、业务事实、OpenFGA
  casdoor_policy.go  # IAM 权限检查
  fga_policy.go      # 资源关系检查

server/internal/domain/openplatform/
  app_service.go
  scope_service.go
  consent_service.go
  disclosure_service.go
  audit.go
```

业务模块不直接 import `casdoorsdk`。业务代码依赖窄接口，例如：

```text
IdentityPermissionChecker
RoleAssignmentWriter
OpenPlatformAppProvisioner
ConsentReader
ResourceRelationshipChecker
```

这样既深度集成 Casdoor，又避免 Casdoor SDK 调用散落在业务模块中。

## 13. Bootstrap 与基础设施

生产 bootstrap 必须自动化且幂等。

Bootstrap 创建或校验：

- Casdoor organization；
- 一方应用；
- 初始管理员；
- roles；
- permissions；
- 如需要，Casbin model；
- 腾讯云 SMS Provider；
- SMTP Provider；
- certificate/public key 配置；
- 开放平台默认配置和默认关闭的开发者门户开关。

Casdoor 使用独立 PostgreSQL database 和 user。生产使用独立 SSO 域名。开发环境可以用 localhost，但 issuer 和 redirect URI 必须显式配置。

配置统一使用 `CASDOOR_*`，`ZITADEL_*` 全部移除。

## 14. 失败语义

受保护请求 fail closed。

| 失败 | 结果 |
|------|------|
| Casdoor token 校验失败 | `401` |
| Casdoor permission check 拒绝 | `403` |
| Casdoor permission check 不可用 | `503` |
| 业务事实存储不可用 | `503` |
| 必需的 OpenFGA 资源关系检查不可用 | `503` |
| 缺少用户授权 | `403` |
| 应用 scope 未审批 | `403` |
| 敏感披露审计写入失败 | 根据端点策略拒绝或 `503` |

受保护操作不允许从已认证静默降级为匿名。不允许 mock 成功路径。

## 15. 安全与隐私基线

1. 第三方应用最小权限。
2. redirect URI 必须精确匹配。
3. 生产禁止 wildcard redirect URI。
4. public client 必须使用 PKCE。
5. client secret 加密存储或只展示一次。
6. 默认不在 token 中暴露手机号、学校、学生状态、身份类型、QQ 或实名状态。
7. 敏感 scope 必须管理员审批。
8. 第三方披露必须用户显式授权。
9. 每次敏感数据披露必须审计。
10. 支持用户和管理员吊销。
11. 开放平台 API 按 app 和 user 限流。
12. 即使开发者是已认证学生，其应用仍按外部应用处理。

## 16. 验证策略

自动化检查至少覆盖：

1. Casdoor bootstrap 创建必需 applications、providers、roles、permissions。
2. 一方应用 OIDC 登录成功。
3. 审批通过的第三方应用 authorization code flow 成功。
4. 未审批 redirect URI 被拒绝。
5. 未审批 scope 的应用无法调用敏感身份 API。
6. 有审批 scope 但无用户授权时拒绝。
7. 有审批 scope 和用户授权时只返回允许字段。
8. `phone.read` 必须同时满足 scope 审批、用户授权、手机号绑定和审计。
9. 北航学生限定板块拒绝非学生、未认证学生和其他学校学生。
10. owner 可以编辑自己的资源。
11. 学校管理员只能管理本学校资源。
12. 板块管理员只能管理被授予的板块。
13. Casdoor 不可用时，受保护管理和开放平台请求拒绝。
14. OpenFGA 不可用时，资源关系检查拒绝。
15. 同步任务失败可见且可重试。

## 17. 后续实施阶段

本文是设计文档，不是实施计划。后续实施计划应拆为：

1. Casdoor 基础设施和 bootstrap。
2. OIDC 登录和一方应用集成。
3. Casdoor SDK 管理面集成。
4. 组合 Casdoor、业务事实、OpenFGA 的授权服务。
5. 认证状态到 Casdoor 和 OpenFGA 的投影同步。
6. 开放平台应用注册、scope 审批和用户授权。
7. 开放平台身份披露 API。
8. 资源关系授权和第三方资源 API。
9. 文档、运维和安全评审。

## 18. 最终设计立场

Casdoor 应深度集成，作为 SSO、OAuth/OIDC 应用、用户、角色、权限、登录方式、Provider 和开发者应用身份的权威系统。

Casdoor 不应单独承载全部 StuHelper 业务事实或每个资源关系。StuHelper 保留实名与学生认证事实。OpenFGA 保留为资源关系授权引擎。开放平台通过已审批、已授权、可审计、字段最小化的 API 披露业务身份数据。

这个分层能同时覆盖当前 StuHelper 的业务授权需求，以及未来开放平台接入第三方应用的需求。

## 19. 参考

- Casdoor OIDC 应用与登录模型：`https://casdoor.org/docs/how-to-connect/oidc-client/`
- Casdoor SMS Provider：`https://casdoor.org/docs/provider/sms/overview/`
- Casdoor Email Provider：`https://casdoor.ai/docs/provider/email/overview/`
- Casdoor 数据初始化：`https://casdoor.org/docs/deployment/data-initialization/`
- Casdoor Go SDK：`https://github.com/casdoor/casdoor-go-sdk`
