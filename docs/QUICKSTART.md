---
type: guide
audience: all
status: current
authoritative-source: this file
last-verified: 2026-05-25
---

# 快速开始

## 前置要求

- Docker + Docker Compose
- Go 1.26+
- Node.js 24+ / pnpm 10+
- Python 3（运维脚本、环境渲染、远程部署前置检查依赖）

Ubuntu 24.04 本地开发机可直接运行仓库内 bootstrap：

```bash
make bootstrap-dev-ubuntu2404
```

该脚本会安装 Docker Engine / Compose plugin、Go 1.26、Node.js 24、pnpm 10、
`air`、Playwright Chromium 运行依赖和浏览器缓存，并把当前 sudo 用户加入 `docker`
组。执行后重新打开终端，再继续一键启动。

## 一键启动

```bash
make dev-init
make dev-up
```

完成以下步骤：
- 生成开发环境变量
- 启动 PostgreSQL / Redis / Casdoor / OpenFGA / MinIO（Docker）
- 验证 Casdoor OIDC metadata；生产路径会用独立 bootstrap 凭据幂等创建 Casdoor organization / first-party applications / flat roles / providers
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
| Casdoor | http://127.0.0.1:8085 |
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
corepack enable && corepack prepare pnpm@10 --activate
pnpm install
pnpm dev:web        # 主站
pnpm dev:admin      # 管理后台
pnpm dev:uni        # UniApp X H5
pnpm type-check:all && pnpm lint:all
pnpm test:web && pnpm test:e2e
pnpm build:web && pnpm build:admin && pnpm build:uni:h5
```

## Koishi 工作区

```bash
cd bots/koishi
corepack yarn install
corepack yarn build
corepack yarn test:unit
corepack yarn test:startup
corepack yarn test:ui
corepack yarn test
corepack yarn dev
```

说明：

- Koishi 工作区与主站开发环境分离启动。
- 本地 `koishi.yml` 固定监听 `5140`；启动烟雾验证会先释放已占用的 `5140` 端口。
- `STUHELPER_CONSOLE_ADMIN_PASSWORD` 必须在启动前提供非空值；`bots/koishi/koishi.yml` 会依赖它作为 Console 管理员密码。
- `STUHELPER_PLATFORM_BASE_URL` 和 `STUHELPER_PLATFORM_SERVICE_TOKEN` 是 Koishi 插件读取的后端连接配置；其中 service token 应与后端 `BOT_SERVICE_TOKEN` 一致。
- NapCat 保持外部部署；本地单元测试不依赖真实 OneBot。
- Koishi Console 已挂载 StuHelper 自定义群管页面，访问路径为 `/stuhelper`。
- `test:ui` 会临时拉起 Koishi Console 并通过 Playwright 覆盖群管中心 NavRail、11 个业务视图和 ChatDock；根目录也可直接运行 `make e2e-koishi`。
- 机器人开发说明见 [guides/koishi-development.md](guides/koishi-development.md)。

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

## 本机生产等价环境

在生产服务器落地前，优先用本机生产等价环境验证完整 Compose 链路：

```bash
make prod-parity-up      # 构建并启动本机生产等价栈
make prod-parity-smoke   # 验收 Web / Admin / API / Identity / 观测入口，并用浏览器检查生产镜像渲染
make prod-parity-datastore-smoke # 只复跑共享 PG / 独立 Redis 隔离检查
make prod-parity-browser-smoke # 只复跑 Web / Admin 浏览器渲染检查
make prod-parity-down    # 停止本机生产等价栈
make prod-parity-reset   # 停止并清理本机生产等价 volume
```

该模式使用仓库内生产 Compose 和 `.run/prod-parity/` 下的本地 env/secrets，不依赖宝塔面板；默认地址以脚本输出为准，通常为 Web `http://127.0.0.1:28000`、Admin `http://127.0.0.1:28001/admin/`、API `http://127.0.0.1:28080`、Grafana `http://127.0.0.1:23003`。它用于在 Ubuntu 24.04 本机先跑通“构建 → 启动 → migration/bootstrap → datastore isolation → API/Identity/OpenFGA/观测 smoke → Web/Admin 浏览器渲染 smoke”的生产等价流程，再把同一套发布脚本用于真实生产。datastore smoke 会验证共享 PostgreSQL 中 StuHelper / OpenFGA / 本地 SSO Casdoor 使用独立数据库、独立登录账号和跨库拒绝连接，并验证 Redis 是 StuHelper Compose 内的独立 TLS/ACL 实例、没有加入外部 datastore 网络；脱敏 evidence 写入 `.run/prod-parity/datastore-smoke-evidence.json`。浏览器 smoke 会访问首页、登录、静态说明页、课程入口、课程列表、评课聚合、搜索、教师主页、写评课、用户中心各 tab、实名 / 学生认证、手机 / QQ 绑定、学籍信息、通知、开发者应用、Open Platform 授权与资料补全保护跳转、404 页面和 Admin 登录跳转，并验证保护入口保留 redirect；evidence 和截图写入 `.run/prod-parity/`，并会检查 `document`、`script`、`stylesheet`、`font`、`image` 关键资源加载失败、关键资源 HTTP 4xx/5xx、页面触发的 `fetch` / `xhr` HTTP 5xx、前端 `pageerror` 和非网络状态类浏览器 `console.error`，用于区分“curl 200 但前端实际白屏/资源加载失败”的问题。

如果本机已有 Vite 占用默认 Playwright 端口，可改用备用端口运行 Web E2E：

```bash
PLAYWRIGHT_WEB_PORT=3300 make e2e-web
make e2e-admin
make e2e-koishi
```

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

- [guides/backend-development.md](guides/backend-development.md) — 后端规范
- [guides/frontend-development.md](guides/frontend-development.md) — 前端规范
- [guides/koishi-development.md](guides/koishi-development.md) — 机器人工作区
- [product-specs/index.md](product-specs/index.md) — 业务域规格
- [guides/](guides/) — 运维文档
