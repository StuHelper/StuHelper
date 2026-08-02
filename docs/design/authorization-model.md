---
type: design
audience: backend-dev
status: current
authoritative-source: server/migrations/ + server/internal/modules/authorization/ + server/internal/pkg/capability/ + design/openfga-model.fga
last-verified: 2026-08-02
---

# 授权模型

> 状态：现行

## 三层授权

1. **授权快照** — PostgreSQL 授权账本提供管理员 role/scope；其中 `super_admin` 是 Casdoor
   StuHelper organization `IsAdmin` 的持久 serving projection，`school_admin` / `section_*`
   由 StuHelper 管理面维护；业务表派生 `verified_student` / `freshman_provisional`，认证用户
   隐式拥有 `user`
2. **能力（Capability）** — 后端把该快照中的角色静态展开为能力字符串
3. **业务事实 + OpenFGA** — 结合应用数据库状态、撤权栅栏和资源关系做最终判断

Casdoor JWT `roles` claim 即使存在，也不参与以上任何一层的 allow/deny。唯一例外是服务端通过
Casdoor management API 读取目标 organization 用户对象的 `IsAdmin`：它是 `super_admin` 的
管理权威，经过组织、禁用与删除状态校验后原子同步到授权账本。

## 角色到能力

角色到能力的权威表位于 `server/internal/pkg/capability/catalog.go`；scope 归一化与展开逻辑位于
`server/internal/pkg/capability/capability.go`。

当前角色目录完整包含：`super_admin`、`school_admin`、`section_admin`、
`section_moderator`、`section_reviewer`、`verified_student`、`freshman_provisional` 与 `user`。
其中 `freshman_provisional` 是有失效时间的新生临时角色，未过期时的能力集合与
`verified_student` 相同（`review:list:full` / `review:create` / `review:edit:own` /
`review:delete:own`），凭据过期后由授权快照撤销。

`section_reviewer` 当前展开为空能力集合；即使存在 section scope 也不授予功能权限。它只作为
Casdoor/授权投影中的预留角色，未来若赋予能力，必须同时更新 capability catalog、路由门禁、
scope 测试和本文档。

`school_admin` 展开为 school-scoped grants，`section_*` 展开为 section-scoped grants；
缺少 DB active grant 时不授予对应 capability，避免把 scoped admin 误放大为全局权限。

运行时通过授权 Access Resolver 把 Casdoor subject 映射到内部 `users.id`，读取
`authorization_grants` 中 `desired=granted AND projection=applied` 的 grant，并与 DB
业务事实合成快照。OpenFGA tuple 中的 `user:<id>` 必须使用内部 `users.id`，不能使用
Casdoor subject。`school_admin` 的评课/举报管理能力是 school-scoped grant，handler 仍需
按 review/report 的学校归属做资源边界校验。

如果当前 DB 快照含 `super_admin`，受保护请求还会实时复核 Casdoor `IsAdmin`。复核失败时
fail-closed；若已被降权，先写 DB revoke 围栏并重载快照，再异步删除 OpenFGA tuple。Casdoor
晋升则在下次登录或 refresh 时建立 grant，且须等待首次 verified projection 后生效。

课程评课列表、最新评课、搜索与批量读取等 optional-auth 公共接口只把
`GlobalCapabilities` 中的 `admin:reviews:manage` 解释为平台级完整正文权限。带学校或
板块 scope 的同名 grant 不能把公共读取提升为全局管理读取；scoped admin 必须通过现有
后台审核接口及其资源归属检查读取和操作授权范围内内容。普通学生的
`review:list:full` / create / edit-own / delete-own 仍从完整 capability 集合读取，并继续
叠加数据库中的学生、实名和 owner 事实。

评课审核路由先按 capability 做功能入口判定，再按该 capability grant 自身的 school/section
范围过滤列表，并对单条 review/report 继续执行 OpenFGA 检查。范围不再从 role 名反推，避免角色目录
与路由策略漂移。正文改写比隐藏、恢复等审核操作权限更高，单独使用
`admin:reviews:edit_content`，仅由 `super_admin`（global）与 `school_admin`（school-scoped）展开；
`section_admin` / `section_moderator` 不获得正文改写能力。

典型能力：
- 后台入口：`admin:dashboard:view`
- 评课运营：`admin:reviews:manage` / `admin:reviews:edit_content` / `admin:reports:manage`
- 教师与敏感词：`admin:teachers:manage` / `admin:sensitive_words:manage`
- 操作日志：`admin:logs:view`
- 授权管理：`iam:grants:manage`
- 用户系统：`user:identity:read` / `user:identity:review` / `user:student:read` / `user:student:review` / `user:school:read` / `user:school:update` / `user:system:read` / `user:system:update`
- 开放平台管理：`open_platform:read` / `open_platform:manage`
- 主站：`review:list:brief` / `review:list:full` / `review:create` / `review:edit:own` / `review:delete:own`

`admin:teachers:manage` 当前只授予全局管理员。`teachers` 是学校级参考数据，现有教师 CRUD 尚未实现 school-scoped 资源过滤；在补齐学校范围参数、repository 过滤和对应测试前，不得把该能力授予 `school_admin` 或 `section_*` 角色。

## 业务访问事实

关键操作还需检查：
- `identityVerified` / `studentVerified`
- `schoolID` 及学校归属
- 资源 owner

代码：`server/internal/modules/course/review/access.go`

示例：发布评课要求实名 + 学生认证均通过。

## OpenFGA

资源级关系判断，用于"能否操作这个具体 review / report / profile"。

- Capability 解决"能否进入这块功能"
- 应用 DB 事实解决"业务状态是否满足"
- OpenFGA 解决"这个具体资源能否操作"

## Open Platform 授权边界

开放平台引入两类主体：人类用户与第三方应用。第三方应用不会因为用户登录而自动继承该用户的 StuHelper 业务权限。

第三方 disclosure API 的放行条件是：

1. 调用方 app 处于 `approved` 状态；
2. 请求 scope 在 `open_platform_approved_scopes` 中；
3. 当前用户对 app + scope 有 active consent；
4. 请求字段只落在 scope 覆盖范围内；
5. 对未来资源 API，额外通过 OpenFGA 检查 app 到具体资源的关系。

`open_platform:manage` 只授予管理员审批、暂停、吊销开放平台应用的能力，不等价于“读取所有用户开放平台数据”。用户对第三方应用的 scope consent 不写入 OpenFGA；它是业务 DB 中的 `open_platform_user_consents` 事实。

## 后台接口

**评课后台**：评课管理、举报处理、内容标记、敏感词、教师管理、操作日志。

**用户系统后台**：实名审核、学生审核、学校配置、系统配置。

## 历史兼容

代码中保留 `A005xxxx` 旧 RBAC 错误码，不代表仍有完整本地 RBAC 管理面。

## 代码入口

| 组件 | 位置 |
|------|------|
| 角色到能力目录 | `server/internal/pkg/capability/catalog.go` |
| Scope 归一化与能力展开 | `server/internal/pkg/capability/capability.go` |
| 认证中间件 | `server/internal/pkg/middleware/auth.go` |
| 授权账本与管理面 | `server/internal/modules/authorization/` |
| 统一授权服务 | `server/internal/platform/authorization/` |
| Capability 中间件 | `server/internal/modules/rbac/middleware.go` |
| OpenFGA Client | `server/internal/pkg/fga/client.go` |
| 评课访问事实 | `server/internal/modules/course/review/access.go` |
