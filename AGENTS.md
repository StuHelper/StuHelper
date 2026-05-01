# AGENTS.md

Agent 入口文件。只包含导航和速查表，详细内容见各文档。

## 命令速查

```bash
# 一键入口
make dev-init && make dev-up   # 开发环境
make dev-status                # 端口和进程
make dev-down                  # 停止
make e2e                       # Playwright
make obs-up                    # 可观测性
make prod-init && make prod-deploy  # 生产

# 后端 (server/)
make fmt && make lint && make test && make build
make generate          # OpenAPI → Go 类型
make lint-spec         # 校验 OpenAPI
make check-drift       # 生成代码是否过期

# 前端 (clients/)
pnpm install
pnpm type-check && pnpm lint
pnpm test:web && pnpm test:e2e
pnpm build:web && pnpm build:admin && pnpm build:uni:h5

# 机器人 (bots/koishi)
cd bots/koishi
corepack yarn install
corepack yarn dev
corepack yarn workspaces list
```

## 技术栈

| 层 | 技术 |
|----|------|
| 后端 | Go 1.26+ / Gin / pgx / PostgreSQL 18 / Redis 8 |
| 前端 | Vue 3.5+ / TypeScript 5+ / Vite 6+ / Element Plus / Pinia |
| 管理后台 | Vben Admin 5（Element Plus 变体） |
| 机器人 | Koishi 工作区 / NapCat（外部部署） |
| 认证 | Casdoor OIDC |
| 资源授权 | OpenFGA |
| 契约 | OpenAPI 3.1 → Go + TypeScript 生成 |
| 部署 | Docker Compose / GitLab CI/CD |
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
| 授权 | `pkg/capability` + `modules/rbac/middleware.go` + `pkg/fga` |

## 后端分层

```
Handler   → HTTP 绑定、响应、错误映射
Service   → 业务规则、事务编排
Repository → SQL、结果扫描
```

SQL 只写在 Repository，业务判断只放在 Service，响应统一通过 `response.*` 返回。

## 授权模型

1. **Casdoor 角色 claim** — Token claims 提供身份侧扁平角色输入
2. **Capability** — 后端静态展开，零 DB 查询
3. **OpenFGA** — 资源级关系判断（谁能操作哪条 review/report）

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
| 业务数据 | PostgreSQL |
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
| 运维发布 | [docs/guides/](docs/guides/) |
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

工程原则详见 [docs/design/core-beliefs.md](docs/design/core-beliefs.md)。
