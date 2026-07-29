# StuHelper

北航校园信息平台：主站、管理后台、认证、课程评课、教务数据导入展示、资源共享。

当前主路线明确不包含：

- 完整教务写侧
- 实验 / 作业 / 提交 / 批改 / 评分系统
- 第三方教务系统的模拟登录或写操作连接器
- 第三方网盘驱动

教务只读数据通过 `academics` 标准化并落库；北航学籍即时校验由
`externaldata` 的只读 Oracle TCPS 连接器提供。两类能力都不取得或代管第三方
教务系统的用户登录凭据。

## 本地开发

```bash
make dev-init
make dev-up
```

自动启动 PostgreSQL / Redis / Casdoor / OpenFGA / SeaweedFS（仅本地 S3 同构验证），执行迁移和 seed，启动热重载（后端 `air`，前端 `Vite`）。

前置工具：Docker / Docker Compose、Go 1.26.5+、Node.js 24+ / pnpm 10+、Python 3（供环境渲染与运维脚本使用）。

```bash
make dev-status   # 端口和进程
make dev-logs     # 热重载日志
make e2e          # Web/Admin Playwright
make e2e-koishi   # Koishi Console Playwright
make dev-down     # 停止
```

## 机器人开发

```bash
cd bots/koishi
corepack yarn install
corepack yarn dev
corepack yarn test:ui
corepack yarn workspaces list
```

该目录是 StuHelper 的 Koishi 插件工作区，用于承载 QQ 机器人与群管相关能力；NapCat 保持外部独立部署。

默认地址（以 `make dev-status` 为准）：

| 服务 | 地址 |
|------|------|
| Web | http://127.0.0.1:3000 |
| Admin | http://127.0.0.1:3001/admin/ |
| API | http://127.0.0.1:8080 |
| Casdoor | http://127.0.0.1:8085 |

## 全 Docker 开发

```bash
make dev-docker-up
```

## 本地生产验证

```bash
make prod-init    # 准备本地生产演练用 .env.prod.* 文件
make prod-deploy  # 配置校验 → 镜像构建 → 启动 → Smoke Check
```

## CI/CD

GitHub Actions 是仓库唯一的 CI/CD 通道：PR 和 `develop` / `main` push 运行按路径裁剪的质量门禁；受信任分支的质量门禁通过后发布带完整 commit SHA 的 GHCR 镜像。staging 与 production 部署使用受保护 GitHub environment 和手工作流，production 必须经过审批。

质量门禁：Go lint/test/build、OpenAPI lint/drift、gosec、govulncheck、pnpm audit、Trivy、Web/Admin lint/type-check/test/build/Playwright、Koishi 单元 / 启动 / Console Playwright smoke。

Actions 权限、branch ruleset、environment secrets 和回滚约束见 [GitHub 仓库与 Actions 治理](docs/guides/github-migration.md)。仓库不保存真实部署 secrets；真实 staging / production 发布仍须按该文档配置环境秘密并单独验收。

```bash
make prod-rollback              # 本地回滚
make ansible-deploy-staging     # Ansible 发布
make ansible-rollback-prod      # Ansible 回滚
```

生产主机首次 bootstrap 会安装 PostgreSQL 逻辑备份 / base backup / backup sync 定时任务，并配好 WAL 归档目录、`.deploy/remote.env` 与远端 registry 凭据占位文件。

## 技术栈

| 组件 | 技术 |
|------|------|
| 后端 | Go 1.26.5+ / Gin / pgx |
| 前端 | Vue 3.5+ / TypeScript 5+ / Vite 6+ / Pinia |
| 管理后台 | Vben Admin（Element Plus 变体） |
| 数据库 | PostgreSQL 18 / Redis 8 |
| 对象存储 | 生产外部 HTTPS S3；开发 / prod-parity 使用 SeaweedFS mini |
| 认证 | Casdoor OIDC |
| 资源授权 | OpenFGA |
| 契约 | OpenAPI 3.1 |
| 部署 | Docker Compose / GitHub Actions / GHCR |
| 观测 | Grafana LGTM + Alertmanager + OpenTelemetry |
| 机器人 | Koishi 工作区 / NapCat（外部部署） |

## 仓库结构

```
StuHelper/
├── AGENTS.md           # Agent 入口
├── Makefile            # 一键命令
├── docker-compose.yml
├── server/             # Go 后端
├── clients/            # Web / Admin / Shared / uniappx
├── bots/               # Koishi 机器人工作区
├── infra/              # 部署、观测、运维脚本
└── docs/               # 项目文档
```

## 文档入口

| 文档 | 内容 |
|------|------|
| [docs/QUICKSTART.md](docs/QUICKSTART.md) | 快速开始 |
| [docs/guides/backend-development.md](docs/guides/backend-development.md) | 后端规范 |
| [docs/guides/frontend-development.md](docs/guides/frontend-development.md) | 前端规范 |
| [docs/design/product-overview.md](docs/design/product-overview.md) | 产品形态与边界 |
| [docs/design/target-scope.md](docs/design/target-scope.md) | 目标范围与模块边界 |
| [docs/guides/production-go-live.md](docs/guides/production-go-live.md) | 生产上线缺漏清单与执行步骤 |
| [docs/guides/](docs/guides/) | 运维与发布 |
| [docs/README.md](docs/README.md) | 文档总索引 |
| [CONTRIBUTING.md](CONTRIBUTING.md) | 贡献流程与提交前验证 |
| [SECURITY.md](SECURITY.md) | 私密漏洞报告与安全响应 |

## 约定

- API 契约权威来源：`server/api/openapi.yaml`
- 数据库权威来源：`server/migrations/`
- 生成代码禁止手改：`server/internal/api/gen/`、`clients/shared/src/types/api.gen.ts`

## 许可状态

仓库根目录目前没有统一的开源许可证。除子目录中明确附带许可证的第三方或派生组件外，仓库公开可见不代表授予复制、修改、分发或再许可权利；在项目所有者完成 monorepo 许可证决策前，不应把整个仓库称为开源项目。

`clients/admin/` 保留其目录内 Vben 的 MIT 许可证与版权声明；`bots/koishi/package.json` 当前声明 `AGPL-3.0`。这两个既有边界不自动扩展到仓库其他目录。
