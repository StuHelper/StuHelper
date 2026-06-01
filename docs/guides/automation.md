---
type: guide
audience: ops
status: current
authoritative-source: this file
last-verified: 2026-05-25
---

# 一键启动与部署

StuHelper 提供统一的自动化入口，目标是把开发与生产都统一为**一条命令**。

## 开发环境

Ubuntu 24.04 本地开发机首次准备：

```bash
# 项目根目录下运行
make bootstrap-dev-ubuntu2404
```

该入口会调用 `infra/ops/bootstrap-dev-ubuntu2404.sh`，安装 Docker Engine / Compose plugin、
Go 1.26、Node.js 24、pnpm 10、`air`、Playwright Chromium 运行依赖和浏览器缓存，并把当前
sudo 用户加入 `docker` 组。执行后重新打开终端，让 docker 组和 PATH 变更生效。

```bash
# 项目根目录下运行
make dev-init
make dev-up
```

`make dev-up` 会自动完成：

1. 初始化本地 `.env`（补齐可运行的开发密钥与默认值）
2. 启动 PostgreSQL / Redis / Casdoor / OpenFGA / MinIO / migration / seed（Docker）
3. 验证 Casdoor OIDC metadata，从本地 Casdoor 内置应用读取一次性 bootstrap 凭据，并幂等创建 StuHelper 的 Web / Admin / UniApp first-party applications、flat roles 和启用的 providers
4. 自动初始化 OpenFGA Store、Model、基础 tuples
5. 自动初始化对象存储 bucket
6. 生成 `.env.generated`
7. 启动本机热重载进程：
   - 后端：`air`
   - Web：`Vite`
   - Admin：`Vite`
8. 自动选择可用端口；若 `3000/3001`、PostgreSQL / Redis / OpenFGA / MinIO 默认宿主机端口，或启用观测栈时的 Prometheus / Grafana / Alloy / exporter 等宿主机端口已被占用，会顺延到下一个空闲端口

默认开发链路不启动 Traefik，也不监听本机 `80/443`。生产公网入口以宝塔 Nginx 配置和
`infra/ops/nginx-public-ingress-preflight.sh` 契约为准。

查看实际运行地址与进程状态：

```bash
make dev-status
make dev-logs
make dev-smoke
```

`make dev-smoke` 默认验收 API / Web / Admin，并在 HTTP 检查后运行浏览器 smoke：用 Playwright 打开
Web 首页、课程入口和 Admin 入口，阻断关键资源 4xx/5xx、页面运行时错误和空白渲染。这样可以发现
“HTML 返回 200，但前端脚本缺失或页面白屏”的问题。如果本地 Grafana 健康端点在 `.env` 的
`GRAFANA_PORT` 上可达，会自动把它纳入同一次 smoke，避免观测栈已运行却被跳过。

如需保留旧的全 Docker 开发模式：

```bash
make dev-docker-up
```

停止开发环境：

```bash
make dev-down
```

彻底清理（含 volume）：

```bash
make dev-reset
```

运维脚本、生产 evidence、Nginx preflight、Ubuntu bootstrap 和 CI 漂移类合同测试统一入口：

```bash
make check-infra-contracts
```

该入口会执行 `infra/ops/tests/run-infra-contracts.sh`，同时覆盖 `*.sh` 和 `*.mjs`
合同测试，避免 runtime token probe runner 这类 Node 合同漏出 CI 门禁。

## 仅启动可观测性

```bash
# 项目根目录下运行
make obs-up
```

`make obs-up` 会复用开发环境的端口选择逻辑，若本机同时运行 prod-parity 或其他服务占用了观测端口，会把实际端口写回 `.env`，并按这些端口运行 `obs-smoke`。

停止：

```bash
make obs-down
```

## 本地生产演练

```bash
# 项目根目录下运行
make prod-init
make prod-deploy
```

`make prod-init` 会准备四份**本地生产演练**文件：

- `.env.prod.shared`
- `.env.prod.secrets.local`
- `.env.prod.generated`
- `.env.prod.generated.secrets`

生成 `.env.prod.shared` 时会直接以 `.env.prod.example` 为模板，并把旧版由开发模板带入的 `localhost` / `http` / 本地对象存储 / 本地告警 sink 默认值重写回生产占位符或正式生产默认值。

其中：

- `.env.prod.shared`：共享配置
- `.env.prod.secrets.local`：本机或临时环境使用的 secrets 文件
- `.env.prod.generated`：运行时派生配置
- `.env.prod.generated.secrets`：生产保留空占位；真实运行时派生 secrets 写入远端 secret backend，避免本地明文落盘
- `.env.casdoor-bootstrap.local`：一次性 Casdoor bootstrap admin credential；只由部署脚本读取，不挂载到运行时 app 容器

`make prod-deploy` 会自动完成：

1. 校验生产共享配置 + 本机 secrets + 运行时派生 secrets 文件
2. 渲染 Prometheus / Alertmanager 生成配置
3. 拉取 / 启动 backend / frontend / admin 生产镜像
4. 启动基础设施、授权、对象存储、可观测性组件
5. 幂等创建 / 校验 Casdoor organization、first-party applications、7 个 flat roles、启用的 provider，并初始化 OpenFGA 派生配置
6. 自动初始化对象存储 bucket
7. 启动 `app` / `frontend` / `admin`，并将它们绑定到宿主机 `127.0.0.1:18080` / `18000` / `18001`
8. 执行业务 Smoke Check + Observability Smoke Check

## 远端部署控制面

远端机器不再由 CI / Ansible 在每次发布时下发 `deploy.remote.env`。现在改为：

- 目标机自持 `${DEPLOY_APP_DIR}/.deploy/remote.env`
- 目标机自持 `${DEPLOY_APP_DIR}/.env.prod.shared` / `${DEPLOY_APP_DIR}/.env.prod.secrets`
- 运行时派生 secrets 通过 `GENERATED_ENV_SECRET_REF` 写入远端 secret backend；`${DEPLOY_APP_DIR}/.env.prod.generated.secrets` 仅保留空占位文件
- 目标机自持 Vault token 文件：
  - `${DEPLOY_APP_DIR}/.secrets/vault/token`
- registry / shared env / generated env secrets 都由 `${DEPLOY_APP_DIR}/.deploy/remote.env` 中的 secret ref 决定（默认 `SECRET_BACKEND=vault-kv-v2`）
- CI / Ansible 仅传：`TAG`、`BACKEND_IMAGE_REF`、`FRONTEND_IMAGE_REF`、`ADMIN_IMAGE_REF`、`ROLLBACK_TAG`

如果远端部署控制面变更，直接在目标机执行：

```bash
cd /opt/stuhelper
./infra/ops/init-remote-deploy-config.sh
```

如果是远端部署，实际链路里会先执行 `infra/ops/remote-preflight.sh`，检查：

- `.deploy/remote.env` 是否就位
- 共享配置 / secrets 文件是否就位
- 备份目录是否存在
- 生产 PostgreSQL TLS 是否强制开启并使用 `verify-ca` 或更严格的 `verify-full`
- 本机宝塔 Nginx 主站/join/id legacy disabled 入口配置是否满足 `infra/ops/nginx-public-ingress-preflight.sh` 契约
- 公网 SSO 入口是否已经具备可用 TLS，并且 `sso.stuhelper.com/.well-known/openid-configuration` 返回有效 Casdoor OIDC discovery；`id.stuhelper.com` 必须保持 404
- PostgreSQL 逻辑备份 / base backup / backup sync timer 是否已启用
- `BACKUP_DATABASE_URL` / `REPLICATION_DATABASE_URL` / `BACKUP_OBJECT_STORAGE_ENDPOINT` / `BACKUP_OBJECT_STORAGE_BUCKET` / `BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID` / `BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY`

停止生产环境：

```bash
make prod-down
```

彻底清理（含 volume）：

```bash
make prod-reset
```

## 生成文件

自动化会生成以下文件：

- `.env.generated`
- `.env.prod.generated`
- `.env.prod.generated.secrets`（生产为空占位）
- `infra/generated/observability/prometheus/prometheus.yml`
- `infra/generated/observability/alertmanager/alertmanager.yml`
- `.deploy/releases.log`
- `.deploy/current-release.env`

这些文件都属于**运行时派生产物**，不应手工维护，也不应提交到 Git。

## GitLab 自动构建与远端部署

符合分支规则的 push 会触发 GitLab CI/CD：

1. 先跑质量门禁与安全门禁
   - Go lint / test / build
   - OpenAPI lint / drift
   - `gosec`
   - `govulncheck`
   - `pnpm audit`
   - `Trivy`
   - Web / Admin unit test + Playwright
   - Koishi unit / startup / Console Playwright smoke
   - 前端构建要求显式提供 `WEB_VITE_SSO_URL`；生产值应指向 `https://sso.stuhelper.com`，用于主站发起 Casdoor 登录和展示 Connect 端点，缺失即失败，不再使用构建期 fallback
2. 构建 backend / frontend / admin 镜像
3. 推送到自建镜像仓库
4. 在 Git 工作区干净时打包部署 bundle（脚本、compose、配置模板、文档）
5. 通过 SSH 上传到远端 Ubuntu 24.04 服务器
6. 在远端执行 `infra/ops/remote-preflight.sh` + `infra/ops/remote-prod-deploy.sh`

生产分支真正部署到线上之前，打包阶段和 `remote-preflight.sh` 会共同避免：

- 远端缺少 `.deploy/remote.env`
- 远端缺少共享配置 / secrets
- backup timer 或 backup sync timer 没启
- WAL 归档目录没准备好
- 部署 bundle 不是从已提交的干净工作区打包
- 主站生产机的宝塔 Nginx `stuhelper.com` / `id.stuhelper.com` / `join.stuhelper.com` server block 漂移
- `id.stuhelper.com` TLS 或 OIDC discovery 漂移
- `join.stuhelper.com` DNS/TLS 或 `/verify/<token>?qq=<qq>` 入口漂移
- 主站和 `id.stuhelper.com` 上的 `/verify` / `/verify/*` 被错误兼容或兜底到旧流程
- `id.stuhelper.com/login/oauth/*` 或 `/signup/oauth/*` 被错误反代到 Casdoor，而不是返回 404

第一次准备远端服务器：

```bash
sudo bash infra/ops/bootstrap-ubuntu2404.sh
```

这个脚本除了装 Docker / Compose 和 Go 1.26 之外，还会准备：

- 部署目录
- `.deploy/remote.env`
- `.secrets/vault/token` 占位文件
- PostgreSQL 逻辑备份 timer
- PostgreSQL base backup timer
- PostgreSQL backup sync timer
- WAL 归档目录

GitLab CI 至少需要以下变量：

- **打包镜像阶段**
  - `REGISTRY`
  - `REGISTRY_USERNAME`
  - `REGISTRY_PASSWORD`
  - `WEB_VITE_SSO_URL`（前端构建单一来源；生产值指向 `https://sso.stuhelper.com`，只用于上游 Casdoor 认证，缺失即失败，不再回落到默认 SSO 域名）
  - `WEB_VITE_IDENTITY_URL`（legacy identity portal 已禁用；生产值必须为空，不得指向 `id.stuhelper.com`）
  - `WEB_VITE_WEB_URL`（Web 主站浏览器 origin；生产值指向 `https://stuhelper.com`，用于登录后回到主站原路径）
- **SSH 发布阶段（staging）**
  - `STAGING_DEPLOY_HOST`
  - `STAGING_DEPLOY_PORT`
  - `STAGING_DEPLOY_USER`
  - `STAGING_DEPLOY_APP_DIR`
  - `STAGING_DEPLOY_SSH_KEY`
  - `STAGING_DEPLOY_SSH_KNOWN_HOSTS`（目标机 SSH host public key，禁止 TOFU）
- **SSH 发布阶段（production）**
  - `DEPLOY_HOST`
  - `DEPLOY_PORT`
  - `DEPLOY_USER`
  - `DEPLOY_APP_DIR`
  - `DEPLOY_SSH_KEY`
  - `DEPLOY_SSH_KNOWN_HOSTS`（目标机 SSH host public key，禁止 TOFU）

远端主机自身还必须提前准备：

- `${DEPLOY_APP_DIR}/.deploy/remote.env`
- `${DEPLOY_APP_DIR}/.env.prod.shared`
- `${DEPLOY_APP_DIR}/.env.prod.secrets`
- `${DEPLOY_APP_DIR}/.env.prod.generated.secrets`（应为空占位）
- `${DEPLOY_APP_DIR}/.secrets/vault/token`

## GitLab 环境流转

- `develop` 分支 push：
  - 构建 backend / frontend / admin 镜像
  - 推送到自建 registry
  - 自动部署到 staging
  - 自动执行 `verify_staging`
- `main` 分支 push：
  - 构建 backend / frontend / admin 镜像
  - 推送到自建 registry
  - 等待手工触发 `deploy_production`
  - 发布完成后自动执行 `verify_production`

前端质量门禁：

- `frontend_e2e`：Web Playwright
- `admin_e2e`：Admin Playwright
- `uniappx_e2e`：UniAppX H5 Playwright
- `koishi_test`：Koishi packages/plugins 单元测试、真实启动烟雾验证和 Console Playwright smoke

只有 Web/Admin/UniAppX H5 E2E 和 Koishi 测试通过后，镜像构建与远端部署才会继续。Koishi Console Playwright
失败时，GitLab 会保留 `bots/koishi/playwright-report` 和 `bots/koishi/test-results` 作为 artifact，
用于查看 trace、截图和错误上下文。

## 回滚

GitLab 提供两个手工 Job：

- `rollback_staging`
- `rollback_production`

可以传：

- `ROLLBACK_TAG=<之前已推送过的镜像 tag>`

如果不传，远端会优先回滚到 `.deploy/releases.log` 里记录的上一条成功版本。

回滚本质上是：

1. 远端读取 `.deploy/remote.env`
2. 远端重新拉取指定 tag 的 backend / frontend / admin 镜像
3. 重新执行 `infra/ops/remote-prod-rollback.sh`
4. 自动再次跑 smoke check

仓库内也保留了本地生产回滚命令：

```bash
# 项目根目录下运行
make prod-rollback
```

## Ansible 入口

如果你希望把远端机器准备、发布和回滚都纳入 playbook，可以直接用：

```bash
# 项目根目录下运行
make ansible-bootstrap
make ansible-deploy-staging
make ansible-deploy-prod
make ansible-rollback-staging
make ansible-rollback-prod
```

第一次用之前，先准备 inventory：

- `infra/ansible/inventory/staging.ini`
- `infra/ansible/inventory/production.ini`

仓库里已经给了同目录示例文件，可以直接改。
