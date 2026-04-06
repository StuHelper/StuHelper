# StuHelper 文档

`docs/` 记录当前代码库的实际状态。接口以 `server/api/openapi.yaml` 为准，schema 以 `server/migrations/` 为准。

## 目录

| 文件 / 目录 | 内容 |
|-------------|------|
| `QUICKSTART.md` | 环境搭建与首次启动 |
| `BACKEND.md` | 后端开发规范 |
| `FRONTEND.md` | 前端开发规范 |
| `PRODUCT.md` | 产品形态概览 |
| `SECURITY.md` | 安全措施 |
| `QUALITY_SCORE.md` | 质量评估 |
| `product-specs/` | 按业务域拆分的功能规格 |
| `design-docs/` | 架构设计与工程原则 |
| `operations/` | 启动、部署、观测、回滚、备份 |
| `references/` | API、数据库、错误码速查 |
| `exec-plans/` | 执行计划与技术债 |
| `adr/` | 架构决策记录 |

## 导航

- [QUICKSTART.md](QUICKSTART.md)
- [BACKEND.md](BACKEND.md) / [FRONTEND.md](FRONTEND.md)
- [product-specs/index.md](product-specs/index.md)
- [operations/README.md](operations/README.md)
- [references/api-overview.md](references/api-overview.md) / [references/database.md](references/database.md)

## 业务域索引

| 域 | 后端入口 | 规格文档 |
|----|----------|----------|
| 认证 | `modules/auth` + `pkg/oidc` + `pkg/token` | [auth-sso.md](product-specs/auth-sso.md) |
| 课程与评课 | `modules/course` + `course/review` | [course-review.md](product-specs/course-review.md) |
| 用户系统 | `modules/user` + `modules/ldap` | [user-system.md](product-specs/user-system.md) |
| 通知 | `modules/notification` | [notification.md](product-specs/notification.md) |
| 授权 | `pkg/capability` + `modules/rbac/middleware.go` + `pkg/fga` | [rbac-authorization.md](product-specs/rbac-authorization.md) |
| 审计 | `pkg/audit` + `pkg/logger` | [audit-logging.md](product-specs/audit-logging.md) |
