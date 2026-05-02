---
type: design
audience: backend-dev
status: current
authoritative-source: server/internal/pkg/capability/ + design/openfga-model.fga
last-verified: 2026-05-02
---

# 授权模型

> 状态：现行

## 三层授权

1. **角色** — Casdoor JWT claims：`super_admin` / `school_admin` / `section_admin` / `section_moderator` / `section_reviewer` / `verified_student` / `user`
2. **能力（Capability）** — 后端把角色静态展开为能力字符串，零 DB 查询
3. **业务事实 + OpenFGA** — 结合应用数据库状态和资源关系做最终判断

## 角色到能力

代码：`server/internal/pkg/capability/capability.go`

Casdoor JWT 只提供扁平角色名，不携带学校 ID 或资源 ID。`school_admin` 展开为 school-scoped grants，`section_*` 展开为 section-scoped grants；缺少 scope 时不授予对应 capability，避免把 scoped admin 误放大为全局权限。

当前运行时已通过 `server/internal/platform/authorization.RoleScopeResolver` 补全 scoped admin 范围：先把 Casdoor subject 映射到内部 `users.id`，再从 OpenFGA 查询 `school#effective_admin` 生成 `school_admin` scope；`section_admin` / `section_moderator` 则反查可管理的 `section` 并保留 section ID，同时校验每个 section 必须有唯一 `section#school` 归属。OpenFGA tuple 中的 `user:<id>` 必须使用内部 `users.id`，不能使用 Casdoor subject。

典型能力：
- 后台入口：`admin:dashboard:view`
- 评课运营：`admin:reviews:manage` / `admin:reports:manage`
- 教师与敏感词：`admin:teachers:manage` / `admin:sensitive_words:manage`
- 操作日志：`admin:logs:view`
- 用户系统：`user:identity:read` / `user:identity:review` / `user:student:read` / `user:student:review` / `user:school:read` / `user:school:update` / `user:system:read` / `user:system:update`
- 主站：`review:list:brief` / `review:list:full` / `review:create` / `review:edit:own` / `review:delete:own`

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

## 后台接口

**评课后台**：评课管理、举报处理、内容标记、敏感词、教师管理、操作日志。

**用户系统后台**：实名审核、学生审核、学校配置、系统配置。

## 历史兼容

代码中保留 `A005xxxx` 旧 RBAC 错误码，不代表仍有完整本地 RBAC 管理面。

## 代码入口

| 组件 | 位置 |
|------|------|
| 能力常量与展开 | `server/internal/pkg/capability/capability.go` |
| 认证中间件 | `server/internal/pkg/middleware/auth.go` |
| Capability 中间件 | `server/internal/modules/rbac/middleware.go` |
| OpenFGA Client | `server/internal/pkg/fga/client.go` |
| 评课访问事实 | `server/internal/modules/course/review/access.go` |
