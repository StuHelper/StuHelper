# AGENTS.md

Agent 入口文件。只包含导航和速查表，详细内容见各文档。

## 命令速查

```bash
# 一键入口
make dev-init && make dev-up   # 开发环境
make dev-status                # 端口和进程
make dev-down                  # 停止
make e2e                       # Web/Admin/UniAppX H5 Playwright
make e2e-koishi                # Koishi Console Playwright
make obs-up                    # 可观测性
make prod-init && make prod-deploy  # 生产

# 后端 (server/)
make fmt && make lint && make test && make build
make generate          # OpenAPI → Go 类型
make lint-spec         # 校验 OpenAPI
make check-drift       # 生成代码是否过期

# 前端 (clients/)
pnpm install
pnpm type-check:all && pnpm lint:all
pnpm test:web && pnpm test:e2e
pnpm build:web && pnpm build:admin && pnpm build:uni:h5

# 机器人 (bots/koishi)
cd bots/koishi
corepack yarn install
corepack yarn dev
corepack yarn test:ui
corepack yarn workspaces list
```

## 技术栈

| 层 | 技术 |
|----|------|
| 后端 | Go 1.26.5+ / Gin / pgx / PostgreSQL 18 / Redis 8 |
| 前端 | Vue 3.5+ / TypeScript 5+ / Vite 6+ / Element Plus / Pinia |
| 管理后台 | Vben Admin 5（Element Plus 变体） |
| 机器人 | Koishi 工作区 / NapCat（外部部署） |
| 认证 | Casdoor OIDC |
| 资源授权 | OpenFGA |
| 契约 | OpenAPI 3.1 → Go + TypeScript 生成 |
| 部署 | Docker Compose / GitHub Actions / GHCR |
| 观测 | Grafana LGTM + Alertmanager + OpenTelemetry |

## 真实来源

| 事实 | 权威来源 |
|------|----------|
| API 契约 | `server/api/openapi.yaml` |
| 数据库 schema | `server/migrations/` |
| 能力常量 | `server/internal/pkg/capability/` |
| 运行时行为 | 源代码和测试 |
| 文档 | `docs/` |

## 系统拓扑

```
Web ──┐             ┌── PostgreSQL
Admin ─┼── Go API ──┼── Redis
SSO ───┘      │     └── 对象存储 (MinIO/S3)
              ├── Casdoor (OIDC)
              ├── OpenFGA (资源授权)
              └── Grafana LGTM (观测)
```

## 业务域

| 域 | 代码入口 |
|----|----------|
| 认证 | `modules/auth` + `pkg/oidc` + `pkg/token` |
| 课程与评课 | `modules/course` + `course/review` |
| 用户系统 | `modules/user` + `modules/ldap` |
| 通知 | `modules/notification` |
| 授权 | `modules/authorization` + `platform/authorization` + `pkg/capability` + `pkg/fga` |

## 后端分层

```
Handler   → HTTP 绑定、响应、错误映射
Service   → 业务规则、事务编排
Repository → SQL、结果扫描
```

SQL 只写在 Repository，业务判断只放在 Service，响应统一通过 `response.*` 返回。

## 授权模型

1. **Casdoor** — 证明身份、会话与登录层 MFA；目标 StuHelper 组织用户对象的 `IsAdmin` 是 `super_admin` 的唯一管理权威，普通 provider `roles` claim 不参与授权
2. **PostgreSQL 授权账本** — `authorization_grants` 是 `school_admin` / `section_*` 的管理真源，同时保存 Casdoor 组织管理员到 `super_admin` 的可审计 serving projection
3. **Capability** — 从 DB-derived access snapshot 静态展开功能权限
4. **OpenFGA** — DB 可重建的资源关系运行时投影（谁能操作哪条 review/report）

## 共享契约链路

```
server/api/openapi.yaml
  ├── server/internal/api/gen/        (Go)
  └── clients/shared/src/types/api.gen.ts (TS source)
        ↓
      clients/shared/dist/*           (package exports)
        ↓
      web / admin / uniappx 各自封装
```

改接口：先改 OpenAPI → `make generate` → 再改实现。

## 数据分层

| 数据 | 存储 |
|------|------|
| 身份与登录 | Casdoor |
| 业务数据 / 授权期望状态 | PostgreSQL |
| 缓存 / 黑名单 / 限流 | Redis |
| 资源关系 | OpenFGA |
| 指标 / 日志 / Trace | Prometheus / Loki / Tempo |

## 项目结构

```
StuHelper/
├── server/
│   ├── cmd/stuhelper/       # 入口
│   ├── api/                 # OpenAPI 源文件
│   ├── internal/
│   │   ├── api/gen/         # 生成代码（禁止手改）
│   │   ├── modules/         # 业务模块
│   │   └── pkg/             # 内部公共包
│   ├── migrations/          # 数据库迁移
│   └── scripts/             # 快照、seed、初始化
├── clients/
│   ├── web/                 # 主站 SPA
│   ├── admin/               # 独立管理后台
│   ├── shared/              # 共享 API / 类型
│   └── uniappx/             # 实验性跨端
├── bots/
│   └── koishi/              # Koishi 机器人插件工作区
├── infra/                   # Docker、观测、运维脚本
└── docs/                    # 项目文档
```

## 文档导航

| 目标 | 入口 |
|------|------|
| 快速开始 | [docs/QUICKSTART.md](docs/QUICKSTART.md) |
| 后端规范 | [docs/guides/backend-development.md](docs/guides/backend-development.md) |
| 前端规范 | [docs/guides/frontend-development.md](docs/guides/frontend-development.md) |
| 产品规格 | [docs/product-specs/index.md](docs/product-specs/index.md) |
| 设计文档 | [docs/design/](docs/design/) |
| 文档治理 | [docs/design/documentation-governance.md](docs/design/documentation-governance.md) |
| 文档维护 | [docs/guides/documentation-maintenance.md](docs/guides/documentation-maintenance.md) |
| 运维发布 | [docs/guides/production-go-live.md](docs/guides/production-go-live.md) / [docs/guides/](docs/guides/) |
| GitHub 治理 | [docs/guides/github-migration.md](docs/guides/github-migration.md) |
| API 参考 | [docs/reference/api-overview.md](docs/reference/api-overview.md) |
| 数据库参考 | [docs/reference/database.md](docs/reference/database.md) |
| 错误码参考 | [docs/reference/error-codes.md](docs/reference/error-codes.md) |

## 开发铁律

- 不手改生成代码（`api/gen/`、`api.gen.ts`）
- 改接口先改 OpenAPI，再 `make generate`
- 后端严格遵循 Handler → Service → Repository
- SQL 只写在 Repository
- HTTP 响应统一通过 `response.*` 返回
- 前端 API 统一使用 `clients/shared`
- 配置通过环境变量读取，不硬编码
- IAM、认证、session、outbox、审计 retention 和后台任务变更必须遵守 [docs/design/iam-implementation-guardrails.md](docs/design/iam-implementation-guardrails.md)

## 北航 Oracle 操作边界

- 本仓库中的所有代理、脚本和自动化对北航 Oracle 只允许建立经过批准的登录会话，以及执行代码中明确固定的 `SELECT` 业务数据查询和只读数据字典查询。
- 连接北航 Oracle 只能使用用户明确指定的既有账号；不得创建、申请、更换或建议自动创建所谓“专用只读”或“更小权限”账号，也不得对既有账号执行任何授权、回收权限或账号变更。若该既有账号自身权限较宽，只能在应用侧继续执行固定 `SELECT` 并将权限风险如实报告，不得因此对北航 Oracle 做任何整改操作。
- `SELECT` 不得使用 `FOR UPDATE`，不得调用存储过程、包、序列、用户自定义函数或其他可能产生副作用的数据库功能；不得接受调用方传入任意 SQL、表名、字段名或查询片段。
- 严禁执行任何写入或可改变数据库、会话、权限或外部系统状态的操作，包括 `INSERT`、`UPDATE`、`DELETE`、`MERGE`、`COMMIT`、`ROLLBACK`、`SAVEPOINT`、`SET TRANSACTION`、`ALTER SESSION`、DDL、PL/SQL、存储过程、作业、锁表、数据库链接、用户/权限变更和文件/外部过程操作。
- 除登录和固定 `SELECT` 自然产生且必须保留的正常审计、连接与网络日志外，不得留下其他操作痕迹；不得为了探测、修复、清理或验证而写入临时对象、临时表、审计记录、测试数据、配置或会话状态。
- 不得删除、规避、篡改、伪造或要求关闭 Oracle、跳板机或网络设备的正常审计日志；“不留下其他操作痕迹”绝不表示规避正常审计。
- Oracle 账号、连接信息和查询结果只在受控进程内存中使用；不得把密码、私钥、完整 DSN、学生个人字段或原始行写入聊天、日志、截图、Git、文档或 memory。
- 本项目代理不得以追加授权、调试、验证、修复、提权或紧急处置为由越过上述边界；任何任务一旦依赖“登录 + 固定 `SELECT`”以外的动作，必须立即停止并向用户报告，不得执行或试探。

工程原则详见 [docs/design/core-beliefs.md](docs/design/core-beliefs.md)。
