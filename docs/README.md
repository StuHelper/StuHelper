# StuHelper 文档中心

这份索引只保留当前代码还在兑现的文档。旧规划稿、重复说明和已经失效的结构都已经移除。

如果文档和实现冲突，按下面顺序判断：

1. HTTP 契约看 `server/api/openapi.yaml`
2. 数据库结构看 `server/scripts/init.sql`
3. 运行时行为看当前代码与测试
4. `docs/reference/`、`docs/guides/`、`docs/modules/` 负责把这些事实整理清楚

## 先看哪里

| 你要做什么 | 去哪里看 |
| --- | --- |
| 第一次把项目跑起来 | [tutorials/quick-start.md](tutorials/quick-start.md) |
| 在后端加接口或改实现 | [guides/backend-quickstart.md](guides/backend-quickstart.md) |
| 走 OpenAPI Spec-First 流程 | [guides/openapi-development-guide.md](guides/openapi-development-guide.md) |
| 继续前端开发 | [guides/frontend-development.md](guides/frontend-development.md) |
| 看 API、数据库、错误码 | [reference/](reference/) |
| 看业务模块边界 | [modules/](modules/) |
| 看系统为什么这样分层 | [architecture/](architecture/) |
| 看 StuHelper 生态身份与应用授权边界 | [architecture/ecosystem-identity-and-authorization.md](architecture/ecosystem-identity-and-authorization.md) |

## 文档分层

| 目录 | 作用 |
| --- | --- |
| [tutorials/](tutorials/) | 新人顺序阅读 |
| [guides/](guides/) | 开发任务手册 |
| [reference/](reference/) | 当前契约和技术事实 |
| [modules/](modules/) | 业务域说明 |
| [architecture/](architecture/) | 高层设计和边界说明 |

## 当前模块

| 模块 | 代码入口 | 文档 |
| --- | --- | --- |
| 身份认证与 SSO | `server/internal/modules/auth` | [modules/auth/](modules/auth/) |
| 评课社区 | `server/internal/modules/course`、`server/internal/modules/course/review` | [modules/course/](modules/course/) |
| 用户系统 | `server/internal/modules/user`、`server/internal/modules/ldap` | [modules/user-system/](modules/user-system/) |
| 应用内 RBAC | `server/internal/modules/rbac` | [modules/rbac/](modules/rbac/) |
| 授权策略 | `server/internal/modules/rbac`、`server/internal/modules/course/review/access.go` | [modules/policy/](modules/policy/) |
| 日志与审计 | `server/internal/pkg/logger`、`server/internal/pkg/middleware`、`server/internal/modules/course/review/*log*` | [modules/logging/](modules/logging/) |

## 规则入口

- 项目工作流在 `.trellis/workflow.md`
- 后端规则入口在 `.trellis/spec/backend/index.md`
- 前端规则入口在 `.trellis/spec/frontend/index.md`
- 项目归档在 `.trellis/workspace/wztxy/journal-1.md`
