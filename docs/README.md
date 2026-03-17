# StuHelper 文档

`docs/` 按用途组织内容，提供稳定的入口和清晰的主题划分。默认优先描述当前代码库、API 和数据结构；少量设计稿会在文档开头明确标注为目标方案。

## 真实来源优先级

技术事实按以下顺序读取：

1. `server/api/openapi.yaml` — API 契约
2. `server/scripts/init.sql` — 数据库 schema
3. 当前代码和测试 — 实现
4. `docs/` — 组织化文档

## 核心概念

| 概念                                  | 定义                                                                                                                  |
| ------------------------------------- | --------------------------------------------------------------------------------------------------------------------- |
| **能力（Capability）**                | 权限字符串（如 `admin:reviews:manage`），授予对特定后端功能的访问权限。能力从角色、用户组和个人覆盖三个来源计算得出。 |
| **访问事实（Access Facts）**          | 业务条件（如 `studentVerified`、`identityVerified`、`schoolID`），用于在能力之外进一步决定内容可见性和操作资格。      |
| **平台管理员（Platform Admin）**      | 在 Casdoor（SSO 提供者）中标记为管理员的用户。与应用级能力是独立的权限体系。                                          |
| **学生认证（Student Verification）**  | 验证用户是否为在校学生，通过 LDAP 验证或管理员人工审核完成。                                                          |
| **实名认证（Identity Verification）** | 使用政府证件进行真实身份验证，证件号码加密存储。                                                                      |

## 文档结构

| 目录            | 用途                               | 入口                                             |
| --------------- | ---------------------------------- | ------------------------------------------------ |
| `tutorials/`    | 入门指南                           | [tutorials/README.md](tutorials/README.md)       |
| `guides/`       | 面向任务的开发指南                 | [guides/README.md](guides/README.md)             |
| `reference/`    | API 路由、数据库 schema、错误码    | [reference/README.md](reference/README.md)       |
| `architecture/` | 系统分层、前端结构、身份与授权边界 | [architecture/README.md](architecture/README.md) |
| `modules/`      | 业务领域和模块文档                 | [modules/README.md](modules/README.md)           |

## 快速导航

| 目标             | 文档                                                                                                   |
| ---------------- | ------------------------------------------------------------------------------------------------------ |
| 本地搭建项目     | [tutorials/quick-start.md](tutorials/quick-start.md)                                                   |
| 后端开发续接     | [guides/backend-quickstart.md](guides/backend-quickstart.md)                                           |
| 前端开发续接     | [guides/frontend-development.md](guides/frontend-development.md)                                       |
| OpenAPI 工作流   | [guides/openapi-development-guide.md](guides/openapi-development-guide.md)                             |
| 查阅 API 和数据  | [reference/api-overview.md](reference/api-overview.md)、[reference/database.md](reference/database.md) |
| 了解模块边界     | [modules/README.md](modules/README.md)                                                                 |
| 了解系统架构     | [architecture/README.md](architecture/README.md)                                                       |
| 查看后续大项计划 | [architecture/follow-up-roadmap.md](architecture/follow-up-roadmap.md)                                 |

## 模块

| 模块       | 代码入口                                                                                    | 文档                                           |
| ---------- | ------------------------------------------------------------------------------------------- | ---------------------------------------------- |
| 认证与会话 | `server/internal/modules/auth`、`server/internal/pkg/sso`、`server/internal/pkg/token`      | [modules/auth/](modules/auth/)                 |
| 评课社区   | `server/internal/modules/course`、`server/internal/modules/course/review`                   | [modules/course/](modules/course/)             |
| 用户系统   | `server/internal/modules/user`、`server/internal/modules/ldap`                              | [modules/user-system/](modules/user-system/)   |
| 应用 RBAC  | `server/internal/modules/rbac`                                                              | [modules/rbac/](modules/rbac/)                 |
| 授权策略   | `server/internal/modules/rbac`、`server/internal/modules/course/review/access.go`           | [modules/policy/](modules/policy/)             |
| 应用通知   | 当前暂无独立代码模块，现有接口暂挂 `server/internal/modules/course/review`                  | [modules/notification/](modules/notification/) |
| 日志与审计 | `server/internal/pkg/logger`、`server/internal/pkg/middleware`、`server/internal/pkg/audit` | [modules/logging/](modules/logging/)           |

## 项目规范

项目工作流和文档规范维护在 `.trellis/` 下：

- `.trellis/workflow.md` — 开发工作流
- `.trellis/spec/guides/index.md` — 思维导引
- `.trellis/spec/backend/index.md` — 后端规范
- `.trellis/spec/frontend/index.md` — 前端规范
