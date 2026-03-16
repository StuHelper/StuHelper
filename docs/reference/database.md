# 数据库设计

当前后端使用 PostgreSQL 存业务数据，Redis 存会话和缓存，Casdoor 只负责身份平面。

## 权威来源

- 初始化结构在 `server/scripts/init.sql`
- 种子数据在 `server/scripts/seed.sql`
- 运行时查询约束以 `server/internal/modules/**/repository*.go` 为准

## 基础组件

| 组件 | 作用 |
| --- | --- |
| PostgreSQL 18+ | 主业务库 |
| Redis 8+ | Token、限流、缓存 |
| Casdoor | 登录、OAuth/OIDC、平台管理员 |

## 数据分层

### 身份事实

放在 Casdoor：

- 账号
- 登录态
- OAuth client
- scope
- 平台管理员身份

### 业务事实

放在本地 PostgreSQL：

- 课程、教师、院系、评分维度
- 实名认证、学生认证、学校配置
- 测评、回复、举报、通知、草稿

### 授权事实

当前也放在本地 PostgreSQL：

- `roles`
- `permissions`
- `role_permissions`
- `user_roles`
- `user_group_*`
- `user_permissions`

这意味着 Casdoor 的平台管理员身份不是航小伴业务授权的真相源。

## 重点表域

| 域 | 代表表 |
| --- | --- |
| 用户系统 | `users`、`user_identities`、`user_profiles`、`school_configs`、`system_configs` |
| RBAC | `roles`、`permissions`、`role_permissions`、`user_roles`、`user_groups`、`user_group_members`、`user_group_permissions`、`user_permissions` |
| 课程与评课 | `departments`、`courses`、`teachers`、`rating_dimensions`、`reviews`、`review_votes`、`review_reports`、`review_replies`、`course_favorites`、`review_drafts`、`notifications` |
| 审计 | `admin_operation_logs` |

## 实名信息保护

实名认证相关字段在 `user_identities`：

- `doc_number_enc` 保存证件号密文
- `person_uid` 保存 `doc_type + ":" + doc_number` 的 HMAC 稳定标识

这样做是为了：

- 数据库不落证件号明文
- 还能做同人匹配和去重
- 日常查询和审核默认不需要读取密文

## 相关模块文档

- [评课社区](../modules/course/)
- [用户系统](../modules/user-system/)
- [应用内 RBAC](../modules/rbac/)
