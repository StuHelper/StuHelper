# 快速开始

## 前置要求

- Docker + Docker Compose
- Go 1.26+
- Node.js 24+ / pnpm 10+
- Python 3（运维脚本、环境渲染、远程部署前置检查依赖）

## 一键启动

```bash
make dev-init
make dev-up
```

完成以下步骤：
- 生成开发环境变量
- 启动 PostgreSQL / Redis / Zitadel / OpenFGA / MinIO（Docker）
- 初始化 Zitadel Project、OIDC App、Project Roles
- 初始化 OpenFGA Store 和 Model
- 初始化对象存储 bucket
- 数据库迁移和开发 seed
- 启动热重载：后端 `air`、前端 `Vite`
- 自动选择可用端口

```bash
make dev-status   # 查看实际地址
make dev-logs     # 热重载日志
make dev-down     # 停止
make dev-reset    # 彻底清理（含 volume）
```

## 默认地址

以 `make dev-status` 输出为准：

| 服务 | 地址 |
|------|------|
| Web | http://127.0.0.1:3000 |
| Admin | http://127.0.0.1:3001/admin/ |
| API | http://127.0.0.1:8080 |
| Zitadel | http://127.0.0.1:8085 |
| Grafana | http://127.0.0.1:3003（需 `make obs-up`） |

## 后端命令

```bash
cd server
make fmt && make lint && make test && make build
make generate       # OpenAPI → Go 类型
make lint-spec      # 校验 OpenAPI
make check-drift    # 生成代码同步检查
make migrate-up     # 手动执行 SQL migration（需设置 DATABASE_URL）
```

## 前端命令

```bash
cd clients
pnpm install
pnpm dev:web        # 主站
pnpm dev:admin      # 管理后台
pnpm dev:uni        # UniApp X H5
pnpm type-check && pnpm lint
pnpm test:web && pnpm test:e2e
pnpm build:web && pnpm build:admin && pnpm build:uni:h5
```

## 手动拆分启动

```bash
docker compose up -d     # 基础设施
cd server && air          # 后端
cd clients && pnpm dev:web   # 前端
```

## 可观测性

```bash
make obs-up       # 启动 Grafana LGTM 栈
make obs-smoke    # 验收
make obs-down     # 停止
```

## 本地生产演练

```bash
make prod-init     # 准备 .env.prod.* 文件
make prod-deploy   # 校验 → 构建 → 启动 → Smoke Check
make prod-rollback # 回滚
make prod-down     # 停止
```

`make prod-init` 默认会准备：

- `.env.prod.shared`
- `.env.prod.secrets.local`
- `.env.prod.generated`

并且会以 `.env.prod.example` 为唯一基线生成生产 skeleton，保留生产占位符，不再从开发 `.env.example` 派生 `localhost` / `http` / 本地告警接收器等默认值。

## GitLab 发布

- `develop` → staging 自动部署 + `verify_staging`
- `main` → 先构建与安全检查，再手工触发 `deploy_production`，完成后执行 `verify_production`

远端服务器首次准备：

```bash
sudo bash infra/ops/bootstrap-ubuntu2404.sh
```

这个脚本会一起装好 Docker / Compose、部署目录、备份目录、`.deploy/remote.env`、Vault token 占位文件，以及 PostgreSQL 逻辑备份 / base backup / backup sync timer。

Ansible 入口：

```bash
make ansible-bootstrap
make ansible-deploy-staging
make ansible-deploy-prod
```

## 下一步

- [BACKEND.md](BACKEND.md) — 后端规范
- [FRONTEND.md](FRONTEND.md) — 前端规范
- [product-specs/index.md](product-specs/index.md) — 业务域规格
- [operations/README.md](operations/README.md) — 运维文档
