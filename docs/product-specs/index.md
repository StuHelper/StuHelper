# 产品规格

按业务域拆分的功能规格，描述当前代码库已实现的功能。

## 域索引

| 域 | 后端入口 | 文档 | 状态 |
|----|----------|------|------|
| 认证 | `modules/auth` + `pkg/oidc` + `pkg/token` | [auth-sso.md](auth-sso.md) | 现行 |
| 课程与评课 | `modules/course` + `course/review` | [course-review.md](course-review.md) | 现行 |
| 用户系统 | `modules/user` + `modules/ldap` | [user-system.md](user-system.md) | 现行 |
| 授权 | `pkg/capability` + `modules/rbac` + `pkg/fga` | [rbac-authorization.md](rbac-authorization.md) | 现行 |
| 通知 | `modules/notification` + `course/review` | [notification.md](notification.md) | 现行（功能已统一，SSE 在 notification 模块，CRUD 在 review 模块） |
| 审计 | `pkg/audit` + `pkg/logger` | [audit-logging.md](audit-logging.md) | 现行 |

## 用户角色

| 角色 | 值 | 场景 |
|------|----|------|
| 游客 | — | 浏览课程、教师、公开评课预览 |
| 登录用户 | `user` | 查看更多内容，管理个人资料 |
| 已认证学生 | `verified_student` | 查看完整评课、发布评课 |
| 学校管理员 / 志愿者 | `school_admin` / `moderator` | 审核内容、处理举报 |
| 平台管理员 | `super_admin` | 平台级运维 |

## 核心概念

| 概念 | 定义 |
|------|------|
| Capability | 控制功能入口的字符串权限 |
| 访问事实 | 业务状态：实名通过、学生认证通过、学校归属 |
| OpenFGA 关系 | 回答"能否操作这个具体 review / report / profile" |
| Shadow User | 本地 `users` 表，Zitadel sub → 业务外键锚点 |
