---
type: guide
audience: all
status: current
authoritative-source: this file
last-verified: 2026-08-06
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
- 把会参与数据库派生值和 PII 解密的本地开发 HMAC / AES 密钥以 `0600` 权限持久化到 `${XDG_STATE_HOME:-~/.local/state}/stuhelper/dev/crypto.env`；重克隆仓库但继续使用同一 Docker 数据卷时会自动恢复这些密钥，避免健康检查通过但用户资料因密钥漂移而不可读
- 启动 PostgreSQL / Redis / Casdoor / OpenFGA / SeaweedFS mini（Docker，仅本地 S3 同构验证）
- 验证 Casdoor OIDC metadata；本地开发会从 Casdoor 内置应用读取一次性 bootstrap 凭据，并幂等创建 StuHelper 的 Web / Admin / UniApp first-party applications 和启用的 providers（不创建普通 StuHelper role catalog）；生产路径会使用独立 bootstrap 凭据执行同一套身份对象收敛。目标 StuHelper organization 的 `IsAdmin` 是 `super_admin` 权威，并在用户登录或 refresh 时投影到授权账本
- 初始化 OpenFGA Store 和 Model
- 按桶隔离身份并预创建应用 / 备份 bucket，上传开发资源 seed
- 数据库迁移和开发 seed
- 启动热重载：后端 `air`、前端 `Vite`
- 自动选择可用端口；若 PostgreSQL / Redis / OpenFGA / 本地对象存储、Web/Admin 或启用观测栈时的 Prometheus / Grafana / Alloy / exporter 默认宿主机端口已被占用，会顺延到下一个空闲端口；默认开发链路不启动 Traefik，也不占用本机 `80/443`

```bash
make dev-status   # 查看实际地址
make dev-logs     # 热重载日志
make dev-smoke    # 验收 API / Web / Admin，并用浏览器检查 Web/Admin 是否真实渲染；若观测栈运行中，也检查 Grafana
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

PostgreSQL、Redis 和本地对象存储的宿主机端口如果已被本机服务占用，`make dev-up`
会顺延到可用端口并写回 `.env`；实际值以 `make dev-status` 与 `.env` 为准。

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

说明：`pnpm test:e2e` 会同时运行 Web、Admin 与 UniAppX H5 的 Playwright 用例，并在桌面和移动视口下执行。UniAppX H5 用例覆盖首页、课程列表、课程详情、评课广场、教师主页、写评课草稿、个人中心、我的评课 / 投票 / 收藏、通知和认证页，并检查关键资源、API 4xx/5xx、`pageerror` 与 `console.error`；H5 游客态不得通过预期内的 `/api/v1/auth/me` 401 掩盖控制台噪声，只有浏览器已有 CSRF/session hint 时才探测当前用户。

如果使用 Codex / MCP 调试本地页面，Playwright MCP 使用自己安装目录下的 `playwright-core`，可能与项目
`@playwright/test` 使用的浏览器缓存版本不同。出现工具列表可见但调用时报 `Transport closed` 时，先确认 MCP
对应浏览器已安装，再重启当前 Codex 会话让 MCP server 重新挂载：

```bash
codex mcp get playwright
node ~/.codex/mcp-packages/playwright-mcp-*/node_modules/@playwright/mcp/cli.js --version
node ~/.codex/mcp-packages/playwright-mcp-*/node_modules/@playwright/mcp/cli.js install-browser chromium
```

如果浏览器已安装但调用 `browser_navigate` 仍返回 `Transport closed`，优先让 MCP 使用隔离浏览器
profile，避免复用已损坏或跨版本的持久 profile。`codex mcp get playwright` 中的 args 应包含
`--headless --no-sandbox --isolated`；修改 MCP 配置后需要重启当前 Codex 会话，让 server 按新参数重新挂载。

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

- `make dev-up` 会随主站开发环境启动 Koishi Console 和 StuHelper 机器人插件；以下命令用于单独调试 Koishi 工作区或运行 Koishi 专项测试。
- 本地 `koishi.yml` 固定监听 `5140`；启动烟雾验证会先释放已占用的 `5140` 端口。
- `STUHELPER_CONSOLE_ADMIN_PASSWORD` 必须在启动前提供非空值；`bots/koishi/koishi.yml` 会依赖它作为 Console 管理员密码。
- `STUHELPER_PLATFORM_BASE_URL` 和 `STUHELPER_PLATFORM_SERVICE_TOKEN` 是 Koishi 插件读取的后端连接配置；其中 service token 应与后端 `BOT_SERVICE_TOKEN` 一致。
- NapCat 保持外部部署；本地单元测试不依赖真实 OneBot。
- Koishi Console 已挂载 StuHelper 自定义群管页面，访问路径为 `/stuhelper`。
- QQ 绑定页面的机器人入口必须通过 `WEB_VITE_QQ_BOT_ENTRY` / `VITE_QQ_BOT_ENTRY` 配置真实可联系入口；未配置时页面会明确显示“未配置机器人入口”，不会把旧占位名称当作真实 bot。
- `test:ui` 会临时拉起 Koishi Console 并通过 Playwright 覆盖群管中心 NavRail、11 个业务视图、ChatDock、配置治理二级工作区和 guard template 保存动作，并检查 `pageerror`、未放行的 console error/warning、关键资源加载失败和关键资源 HTTP 4xx/5xx；根目录也可直接运行 `make e2e-koishi`。
- 机器人开发说明见 [guides/koishi-development.md](guides/koishi-development.md)。

## 手动拆分启动

```bash
docker compose up -d     # 基础设施
cd server && air          # 后端
cd clients && pnpm dev:web   # 前端
```

手动 `docker compose up -d` 只用于基础设施容器。公网 / 单入口反代以生产文档中的宝塔 Nginx 配置为准；本地热更新开发不需要 Traefik。

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
其中 `POSTGRES_EXPORTER_DB_PASSWORD` 会作为独立 secret 生成，用于只有
`pg_monitor` 权限的 `stuhelper_metrics`；不要把 `STUHELPER_BACKUP_DB_PASSWORD`
或应用数据库密码复用给 exporter。Redis exporter 使用独立
`REDIS_EXPORTER_PASSWORD` 和无应用 key 访问权的 `stuhelper_metrics` ACL
用户，不得复用应用的 `REDIS_PASSWORD`。
初始化还会生成 `postgres-client-ca` / `redis-client-ca` 两个公开证书目录。
应用、迁移、OpenFGA 和 exporter 只挂载这些目录；服务端私钥和仅含密码哈希的 Redis ACL
保留在各自服务的私有源目录，并在容器启动时复制到 0600 的 tmpfs。

## 本机生产等价环境

在生产服务器落地前，优先用本机生产等价环境验证完整 Compose 链路：

```bash
make prod-parity-up      # 构建并启动本机生产等价栈
make prod-parity-ingress # 只安装本机 stuhelper/join/sso 域名入口
make prod-parity-ingress-down # 只移除本机 hosts、代理绕过和 Nginx 域名入口
make prod-parity-smoke   # 验收 Web / Admin / API / SSO / 观测入口，并用浏览器检查生产镜像渲染
make prod-parity-datastore-smoke # 只复跑共享 PG / 独立 Redis 隔离检查
make prod-parity-browser-smoke # 只复跑 Web / Admin 浏览器渲染检查
make prod-parity-down    # 停止本机生产等价栈并移除本地域名入口
make prod-parity-reset   # 停止并清理本机生产等价入口、volume 和 network
```

该模式使用仓库内生产 Compose 和 `.run/prod-parity/` 下的本地 env/secrets；`make prod-parity-up` 会安装本机 Nginx/hosts 入口，把 `stuhelper.com`、`join.stuhelper.com`、`sso.stuhelper.com` 指向本机，并在缺少本机证书时生成 `.run/prod-parity/local-tls/` 下的本地 TLS 证书。浏览器可见的账号中心、授权应用、开发者应用、学生认证和 QQ 绑定都由 `https://stuhelper.com` 主站承载；`https://join.stuhelper.com/verify/<code>` 承载入群验证，`join.stuhelper.com/` 和主站业务页面路径返回 404，避免把 join 当成主站别名；`https://sso.stuhelper.com` 承载 Casdoor 登录和 OIDC issuer。本机浏览器默认访问 Web `https://stuhelper.com`、Admin `https://stuhelper.com/admin/`、Join 验证入口 `https://join.stuhelper.com/verify/<code>`、SSO `https://sso.stuhelper.com`；直连排错地址仍保留 API `http://127.0.0.1:28080`、Grafana `http://127.0.0.1:23003`。它用于在 Ubuntu 24.04 本机先跑通“构建 → 启动 → migration/bootstrap → datastore isolation → API/Casdoor/OpenFGA/观测 smoke → Web/Admin 浏览器渲染 smoke → 真实上游登录后刷新保持会话”的生产等价流程，再把同一套发布脚本用于真实生产。datastore smoke 会验证共享 PostgreSQL 中 StuHelper / OpenFGA / 本地 SSO Casdoor 使用独立数据库、独立登录账号和跨库拒绝连接，并验证 Redis 是 StuHelper Compose 内的独立 TLS/ACL 实例、没有加入外部 datastore 网络；脱敏 evidence 写入 `.run/prod-parity/datastore-smoke-evidence.json`。浏览器 smoke 会先写入本机 prod-parity 专用的最小课程 / 教师 / 评课数据、入群认证会话，并刷新评分统计与教师物化视图，脱敏 evidence 写入 `.run/prod-parity/smoke-data-evidence.json`；随后用桌面和移动视口访问首页、登录、SSO admin/123 登录刷新、认证回调错误态、入群认证链接、静态说明页、课程入口、课程列表、课程详情、课程评课详情、评课聚合、搜索、教师主页、教师详情、写评课、用户中心业务 tab、账号中心、个人资料、账号安全、Connect 端点、授权应用、实名/学生认证、手机/QQ 绑定、学籍信息、开发者应用、通知、Open Platform 授权与资料补全保护跳转、404 页面和 Admin 登录跳转，并验证保护入口保留 redirect；evidence 和截图写入 `.run/prod-parity/`，截图文件名带视口后缀。每次浏览器 smoke 前会使用仅在 `APP_ENV=prod-parity` 下渲染的专用 Redis 维护身份，清理本机课程 / 评课缓存和 `rl:*` 限流键，避免上一轮验收污染本轮结果；该身份只能枚举并删除这些前缀的键，不能读取值、写入值或执行管理命令，生产默认 ACL 不创建该身份。浏览器运行时会拦截 Web Vitals / 前端错误上报，防止烟测自身消耗业务 API 限流额度；Admin 跳转到本机 Casdoor 登录页时，会把 Casdoor 打包 CSS 中的 Google Fonts 请求替换为空 CSS 并写入 `stubbedExternalResources` 证据，避免本机准入依赖第三方字体 CDN。该检查会覆盖 `document`、`script`、`stylesheet`、`font`、`image` 关键资源加载失败、关键资源 HTTP 4xx/5xx、页面触发的未声明允许 `fetch` / `xhr` HTTP 4xx/5xx、前端 `pageerror` 和非网络状态类浏览器 `console.error`，用于区分“curl 200 但前端实际白屏/资源加载失败”的问题。

`make prod-parity-down` 会同时撤销脚本管理的 hosts 标记块、GNOME 代理绕过项和本机 Nginx 配置；`make prod-parity-reset` 还会删除标签精确匹配当前 `prod-parity` Compose 项目名的卷，并重试删除相同项目的网络。清理脚本在仍有该项目容器时会拒绝删除资源，也不会操作其他 Compose 项目。

本地和 CI 的 Playwright 默认都不复用已有服务，而是按各自配置启动隔离的测试服务；只有显式设置
`PLAYWRIGHT_REUSE_SERVER=1` 时才会复用目标端口上的现有服务。这样可以避免把脏运行态误当成测试结果。
如果本机已有其他进程占用默认 Playwright 端口，可改用备用端口运行 E2E。Web E2E 默认同时执行 `desktop-chromium` 和
`mobile-chromium` 两个 Playwright project，覆盖桌面与移动视口交互；Admin E2E 会先构建后台 SPA，再用
`vite preview` 测试发布形态的静态产物，并覆盖桌面与移动 project。Admin 默认串行执行；需要压测测试并发时，
可显式设置 `ADMIN_E2E_WORKERS`：

```bash
make e2e
PLAYWRIGHT_WEB_PORT=3300 make e2e-web
make e2e-admin
ADMIN_E2E_WORKERS=2 ADMIN_E2E_PORT=4178 make e2e-admin
make e2e-uni
make e2e-koishi
```

## GitHub 发布

- Pull Request、`develop` 和 `main` push 运行 GitHub Actions 质量、安全、契约和 E2E 门禁。
- `develop` / `main` 的受信任 push 只有在 `CI / Required`、Go/JS CodeQL 通过且仍是实时 branch head 后，才把三个带完整 commit SHA 的不可变镜像发布到 GHCR，并附加 provenance 与 CycloneDX SBOM attestation。
- Forward `Deploy` 固定发布当前 workflow ref 的 head，不接受历史 SHA；staging 暂缓期间，`main` 可用显式 `direct` 模式创建 production approval，批准后才部署生产。未来独立 staging 就绪后切换到同 SHA `after-staging` 晋级。
- GitHub `Rollback` 手工作业由当前可信 controller 按相同的 provenance 和 digest 约束选择历史完整 SHA，不接受可变 tag，也不执行历史运维脚本。

远端服务器首次准备：

```bash
sudo bash infra/ops/bootstrap-ubuntu2404.sh
```

这个脚本会一起装好 Docker / Compose、Go 1.26、部署目录、备份目录、`.deploy/remote.env`、Vault
runtime token 占位文件，以及 PostgreSQL 逻辑备份 / base backup / backup sync unit。若
`/opt/stuhelper` 中尚无 deploy bundle，后备 timer 只写入 unit、不会启用或启动；上传首个 bundle、完成
生产配置和 Vault 准备后，必须以 root 执行
`BACKUP_TIMERS_ACTIVATE=true /opt/stuhelper/infra/ops/install-backup-timers.sh`。安装器不会预先停用已有 timer
或清除历史失败状态，而是先以非 root 部署用户运行 `remote-preflight.sh --timer-activation`；只有配置、Vault、对象
存储和 unit 契约通过后才重新启用 timer，随后生产预检才会放行。Vault 初始化、
解封并 seed 三条生产 secret ref 后，还必须由 root 执行
`VAULT_ROOT_INIT_FILE=/var/lib/stuhelper/vault-credentials/init.json ./infra/ops/vault-runtime-token.sh configure`，
把占位文件替换成专用最小权限 periodic token 并安装自动续期 timer；禁止把初始化 root token 当作
部署 token。GitHub 自动部署的 GHCR token 只在单次 job 中使用，不写入这些持久文件。

仓库、Actions 权限、GHCR、environment secrets、发布和回滚治理见
[GitHub 仓库与 Actions 治理](guides/github-migration.md)。

Ansible 入口：

先将 `infra/ansible/inventory/production.example.ini` 或 `staging.example.ini` 复制为对应的忽略文件，
填写真实非本机主机和 SSH 用户。入口会拒绝缺失、空文件、示例占位符、无法解析的清单以及空
`stuhelper` 主机组；本机没有 Ansible 时会通过 `uvx` 使用 `requirements.txt` 锁定的版本。

```bash
make ansible-bootstrap
export REGISTRY_USERNAME=<ghcr-user>
export REGISTRY_PULL_TOKEN=<short-lived-read-packages-token>
make ansible-deploy-staging
make ansible-deploy-prod
```

## 下一步

- [guides/backend-development.md](guides/backend-development.md) — 后端规范
- [guides/frontend-development.md](guides/frontend-development.md) — 前端规范
- [guides/koishi-development.md](guides/koishi-development.md) — 机器人工作区
- [product-specs/index.md](product-specs/index.md) — 业务域规格
- [guides/](guides/) — 运维文档
