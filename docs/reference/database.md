# 数据库设计

当前后端使用 PostgreSQL 存业务数据，Redis 存会话与缓存，Casdoor 提供身份平面。

## 权威来源

- 初始化结构在 `server/scripts/init.sql`
- 种子数据在 `server/scripts/seed.sql`
- 运行时查询约束在 `server/internal/modules/**/repository*.go`

## 基础组件

| 组件 | 作用 |
| --- | --- |
| PostgreSQL | 主业务库 |
| Redis | Token、限流、缓存 |
| Casdoor | 登录、OAuth/OIDC、平台管理员字段 |

## 当前数据分层

### 身份事实

Casdoor 保存账号、登录态、OAuth client 和平台管理员字段。

### 业务事实

PostgreSQL 保存课程、教师、评分维度、实名认证、学生认证、学校配置、测评、回复、举报、通知和草稿。

### 授权事实

PostgreSQL 同时保存 `roles`、`permissions`、`role_permissions`、`user_roles`、`user_group_*`、`user_permissions`。航小伴后台授权读取这组本地关系和 capability。

## 重点表域

| 域 | 代表表 |
| --- | --- |
| 用户系统 | `users`、`user_identities`、`user_profiles`、`school_configs`、`system_configs` |
| RBAC | `roles`、`permissions`、`role_permissions`、`user_roles`、`user_groups`、`user_group_members`、`user_group_permissions`、`user_permissions` |
| 课程与评课 | `departments`、`courses`、`teachers`、`rating_dimensions`、`reviews`、`review_votes`、`review_reports`、`review_replies`、`course_favorites`、`review_drafts`、`notifications` |
| 审计 | `admin_operation_logs` |

## 实名信息保护

`user_identities` 保存两类核心字段：

- `doc_number_enc` 存证件号密文
- `person_uid` 存 `doc_type + ":" + doc_number` 的 HMAC 稳定标识

这组字段支持同人匹配、去重和学籍比对，同时保持数据库中的证件号密文形态。

## 相关文档

- [../modules/course/](../modules/course/)
- [../modules/user-system/](../modules/user-system/)
- [../modules/rbac/](../modules/rbac/)
