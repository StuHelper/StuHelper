---
type: reference
audience: all
status: current
authoritative-source: this file (index) + linked docs
last-verified: 2026-05-09
---

# StuHelper 文档

`docs/` 只放**长期文档**：反映当前代码库真实状态的规范、设计、规格、参考。临时工件（执行计划、历史设计快照、审计、评估）在 [internal/README.md](internal/README.md)。

**真源永远是代码**：API → `server/api/openapi.yaml`；Schema → `server/migrations/`；能力常量 → `server/internal/pkg/capability/`。本目录只做解释与导航。

---

## 按读者分流

### 我是**新人**，第一次跑这个项目

1. [QUICKSTART.md](QUICKSTART.md) — 环境搭建与一键启动
2. [design/product-overview.md](design/product-overview.md) — 这是什么产品、给谁用
3. [design/target-scope.md](design/target-scope.md) — 做什么、不做什么

### 我要**写代码**（后端 / 前端）

1. [guides/backend-development.md](guides/backend-development.md) — 后端规范、分层、OpenAPI 工作流
2. [guides/frontend-development.md](guides/frontend-development.md) — 前端规范、共享契约、路由
3. [guides/koishi-development.md](guides/koishi-development.md) — Koishi 工作区、绑定/群管能力、固定端口与机器人测试
4. [design/core-beliefs.md](design/core-beliefs.md) — 工程原则（契约驱动、不可变、小文件）
5. [product-specs/](product-specs/) — 要改的业务域规格
6. [reference/api-overview.md](reference/api-overview.md) — 接口模块分组

### 我要**维护文档系统**

1. [design/documentation-governance.md](design/documentation-governance.md) — 文档架构、元数据、生命周期
2. [guides/documentation-maintenance.md](guides/documentation-maintenance.md) — 新增 / 修改文档时的操作步骤
3. [design/core-beliefs.md](design/core-beliefs.md) — 工程原则与真源约束

### 我要**理解系统**（架构 / 设计）

1. [design/target-scope.md](design/target-scope.md) — 主路线边界
2. [design/layered-architecture.md](design/layered-architecture.md) — 后端分层为什么这么切
3. [design/frontend-architecture.md](design/frontend-architecture.md) — 前端 Monorepo 与共享契约链路
4. [design/auth-and-session.md](design/auth-and-session.md) — 认证与会话机制
5. [design/authorization-model.md](design/authorization-model.md) — 三层授权（角色 → 能力 → FGA）
6. [design/open-platform-v1.md](design/open-platform-v1.md) — 第三方应用接入、用户授权与最小化披露
7. [design/storage-architecture.md](design/storage-architecture.md) — 存储抽象与驱动
8. [design/security-model.md](design/security-model.md) — 安全措施
9. [guides/koishi-development.md](guides/koishi-development.md) — 机器人子系统边界与开发入口
10. [adr/](adr/) — 单项架构决策

### 我要**运维 / 发布 / 排障**

1. [guides/production-go-live.md](guides/production-go-live.md) — 生产上线缺漏清单与执行步骤
2. [guides/release-runbook.md](guides/release-runbook.md) — 发布流程、verify、回滚
3. [guides/database-migrations.md](guides/database-migrations.md) — 迁移、seed、回滚
4. [guides/observability.md](guides/observability.md) — Grafana LGTM、告警、排障
5. [guides/backup-and-restore.md](guides/backup-and-restore.md) — PostgreSQL 备份恢复
6. [guides/automation.md](guides/automation.md) — 一键启动、GitLab、Ansible
7. [guides/production-topology.md](guides/production-topology.md) — 生产拓扑

### 我想查**事实**

| 我想查 | 去哪里 |
|--------|--------|
| 某个接口的字段 / schema | `server/api/openapi.yaml` |
| 某张表的列 / 索引 | `server/migrations/000001_initial_schema.up.sql` |
| 能力常量 | `server/internal/pkg/capability/capability.go` |
| API 模块分组 | [reference/api-overview.md](reference/api-overview.md) |
| 数据库模块分组 | [reference/database.md](reference/database.md) |
| 错误码规则 | [reference/error-codes.md](reference/error-codes.md) |

---

## 目录图

```
docs/
├── README.md              本页（角色分流入口）
├── QUICKSTART.md          新人上手
├── guides/                How-to:怎么做
├── design/                Explanation:为什么这样设计
├── product-specs/         业务域规格
├── reference/             查资料(只做导航摘要)
├── adr/                   单项架构决策
└── internal/              临时工件:exec-plans + design-snapshots + 阶段性评估
```

## 目录职责

- **一目录一承诺**：进入任一目录前，读者应能预期里面是什么类型的文档。
- **一文档一类型**：每份文档只回答一种问题，不混写。
- **一事实一真源**：OpenAPI、migrations、capability 常量只能有一份权威来源。
- **一切规则可执行**：长期文档结构与元数据由 `make check-docs` 守卫。

完整规则见 [design/core-beliefs.md](design/core-beliefs.md)、[design/documentation-governance.md](design/documentation-governance.md) 与 [guides/documentation-maintenance.md](guides/documentation-maintenance.md)。
