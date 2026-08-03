---
type: guide
audience: ops
status: current
authoritative-source: this file
last-verified: 2026-08-03
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
2. 启动 PostgreSQL / Redis / Casdoor / OpenFGA / SeaweedFS mini / migration / seed（Docker；SeaweedFS 仅用于本地 S3 同构验证）
3. 验证 Casdoor OIDC metadata，从本地 Casdoor 内置应用读取一次性 bootstrap 凭据，并幂等创建 StuHelper 的 Web / Admin / UniApp first-party applications 和启用的 providers；Casdoor bootstrap 不创建普通 StuHelper role catalog，目标 organization 的 `IsAdmin` 由登录/refresh 链路投影为 `super_admin`
4. 自动初始化 OpenFGA Store、Model、基础 tuples
5. 生成桶级本地身份配置，预创建应用 / 备份 bucket，并上传开发资源 seed
6. 生成 `.env.generated`
7. 启动本机热重载进程：
   - 后端：`air`
   - Web：`Vite`
   - Admin：`Vite`
8. 自动选择可用端口；若 `3000/3001`、PostgreSQL / Redis / OpenFGA / 本地对象存储默认宿主机端口，或启用观测栈时的 Prometheus / Grafana / Alloy / exporter 等宿主机端口已被占用，会顺延到下一个空闲端口

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

## 运行时镜像安全策略

`infra/security/runtime-images.json` 是第三方运行时、运维工具和本地开发基础镜像的扫描清单。清单中的 registry 镜像必须使用完整 `tag@sha256`，带 `latest`、`beta`、`master` 或 `nightly` 的标签即使固定了 digest，也必须在 30 天内重新核对上游；生产使用的引用还必须同时与 `.env.example`、`.env.prod.example` 一致。

完整扫描入口：

```bash
mkdir -p .cache/trivy runtime-image-scan-evidence
TRIVY_CACHE_DIR="$PWD/.cache/trivy" \
RUNTIME_IMAGE_SCAN_OUTPUT_DIR="$PWD/runtime-image-scan-evidence" \
bash infra/ops/scan-runtime-images.sh
```

扫描器使用固定 digest 的 Trivy，刷新漏洞库后检查 `HIGH`、`CRITICAL` 和 `UNKNOWN`，并把每个镜像的 JSON evidence 单独落盘。规则如下：

- 未列入策略的发现立即失败；
- `CRITICAL` 不能使用普通例外，只能提交 `not_affected` VEX，并提供可复核的上游或仓库证据；
- `HIGH` 和 `UNKNOWN` 只能使用带 owner、缓解措施和最长 30 天有效期的逐包、逐版本例外；
- 已修复但仍留在策略中的例外/VEX 会作为 stale record 失败；
- 生产部署与远端 preflight 会校验当前进程中的基础设施镜像引用，任何未经过该策略扫描的覆盖值都会阻断部署。

GitHub Actions 的 `Runtime image security` 运行完整扫描并保留 JSON evidence；`make check-infra-contracts` 负责校验策略结构、CI 接线和防降级约束，不重复拉取全部镜像。

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

生成 `.env.prod.shared` 时会直接以 `.env.prod.example` 为模板，并把旧版由开发模板带入的 `localhost` / `http` / 本地对象存储 / 本地告警 sink 默认值重写回生产占位符或正式生产默认值。仓库已知的旧 PostgreSQL、Redis、OpenFGA、运维工具和可观测性镜像默认值也会升级到已扫描的 digest；不等于旧默认值的运维自定义覆盖不会被静默改写，但若不匹配当前扫描策略，生产部署会明确失败。

其中：

- `.env.prod.shared`：共享配置
- `.env.prod.secrets.local`：本机或临时环境使用的 secrets 文件
- `.env.prod.generated`：运行时派生配置
- `.env.prod.generated.secrets`：生产保留空占位；真实运行时派生 secrets 写入远端 secret backend，避免本地明文落盘
- `.env.casdoor-bootstrap.local`：一次性 Casdoor bootstrap admin credential；只按受限 `KEY=VALUE` 环境文件解析，不执行 shell 语法，只接受 `CASDOOR_BOOTSTRAP_CLIENT_ID`、`CASDOOR_BOOTSTRAP_CLIENT_SECRET`、`CASDOOR_BOOTSTRAP_APPLICATION`、`CASDOOR_BOOTSTRAP_CERTIFICATE`、`CASDOOR_BOOTSTRAP_ORGANIZATION`，拒绝其他字段及 `BASH_ENV` / `ENV`，且不挂载到运行时 app 容器。解析器会先验证完整文件再输出赋值；失败诊断只含文件、行号和字段名，不会连带输出此前已解析的 credential。生产父部署进程只在隔离子 Shell 中验证该文件并立即清除可能继承的同名变量；真正需要凭据的 bootstrap/cutover 子进程各自重新读取，备份、渲染、迁移、Docker 和 smoke 子进程不会继承 bootstrap credential

`make prod-deploy` 会自动完成：

1. 校验生产共享配置、secret backend、应用不可变镜像引用、基础设施镜像扫描策略及例外有效期、外部 HTTPS S3 与 PostgreSQL/Redis TLS 配置
2. 渲染 Prometheus / Alertmanager 生成配置并启动 Redis、OpenFGA 和可观测性组件；生产 Compose 不启动本地对象存储
3. 在迁移前生成 PostgreSQL 备份，通过 rclone 上传到已预配的外部备份 bucket，并执行取回 evidence
4. 执行数据库迁移，幂等创建 / 校验 Casdoor organization、first-party applications、启用的 provider，初始化 OpenFGA 派生配置；PostgreSQL `authorization_grants` 管理 scoped admin，并保存 Casdoor organization administrator 到 `super_admin` 的 serving projection
5. 启动 `app` / `frontend` / `admin`，绑定宿主机 `127.0.0.1:18080` / `18000` / `18001`
6. 执行业务、公网浏览器和 Observability Smoke Check

生产对象存储 bucket、访问身份、服务端加密和生命周期策略必须由外部 S3 控制面预先创建；部署身份没有建桶或管理 IAM 的权限。公共 CA 场景下 `OBJECT_STORAGE_TLS_CA` 与 `OBJECT_STORAGE_TLS_CA_HOST_PATH` 都留空。私有 CA 场景下，前者固定为容器路径 `/object-storage-tls/ca.crt`，后者指向宿主机上经核验的只读 PEM CA bundle；部署脚本只把公开证书原子复制到 `infra/generated/object-storage-client-ca/ca.crt`，应用容器不会挂载本地 SeaweedFS 的私钥或身份配置。备份 rclone 的 `BACKUP_OBJECT_STORAGE_TLS_CA` 仍使用宿主机可读路径，可与应用 CA 不同。

备份目标还必须与生产主机处于独立故障域。完成异机/云对象存储配置并验证生产主机完全丢失后仍可取回工件，才可把 `BACKUP_OBJECT_STORAGE_OFF_HOST_CONFIRMED` 设为 `true`。同时必须用 `BACKUP_OBJECT_STORAGE_LOCAL_IDENTITY_CIDRS` 列出所有可路由回本机的公网、1:1 NAT 或负载均衡 IP/CIDR；确认不存在额外身份时也要显式填写 `none`。生产预检会拒绝缺少这份清单、只有逗号/空白等分隔符的空清单或命中清单的端点，也会拒绝 `minio`、`object-storage` 等单标签 Compose 主机名、loopback、link-local、旧式缩写数字 IPv4、带 zone identifier 的 IPv6、尾随点 FQDN 和 `.local` 端点。

FQDN 必须在 rclone 实际使用的 Docker 网络命名空间内解析到 A/AAAA 地址，且任何解析结果都不能是本机接口地址、本机 Docker 容器地址或列入生产公网/NAT/LB 身份清单的等价 IPv4、IPv4-mapped IPv6 地址。使用 virtual-hosted S3 时，基础 endpoint 与实际请求使用的 `bucket.endpoint` 会被分别解析和校验，bucket 必须能组成合法的小写 ASCII DNS 主机名。通过验证的每个传输主机及完整地址集合都会用容器 `--add-host` 固定给本轮 rclone；URL、HTTP Host 与 TLS SNI 仍保留原 FQDN，避免验证和传输之间的 DNS 轮换或重绑定。生产取回 evidence 也会在加载最终环境后重新执行该门禁，而不是复用父进程地址或回到普通 DNS。

host-local `bridge` 网络还会拒绝整个 IPAM 子网；共享的 `macvlan`、`ipvlan`、`overlay` 只拒绝本机容器精确地址，避免误伤异机节点。`BACKUP_OBJECT_STORAGE_DOCKER_NETWORK=host|none` 不允许用于生产异机备份。root 管理的三个生产备份 systemd service 还会固定注入 `BACKUP_OBJECT_STORAGE_OFF_HOST_REQUIRED=true`；共享配置无法把它覆盖为 false，因此发布后发生配置漂移时，定时同步同样失败关闭。StuHelper 环境加载器会拒绝配置文件中的 `BASH_ENV` / `ENV`，并在加载结束后清除父进程继承的同名变量。服务本身使用非登录、禁用 profile 的 Bash；定时备份调用备份、同步子脚本时也使用同样的隔离启动路径。systemd 在启动 `/usr/bin/env` 前通过受检的 `UnsetEnvironment=` 清除 `LD_PRELOAD`、`LD_LIBRARY_PATH`、`LD_AUDIT`、`GCONV_PATH`、`LOCPATH`，随后 `env -i` 丢弃包括 manager `DefaultEnvironment=` / `systemd.setenv=` 在内的其余继承环境，只重新加入固定 `PATH`、四个环境文件路径、`LOCAL_STATE_DIR=/var/lib/stuhelper`、可选 staging 路径和异机门禁标记；显式 local-state 路径使脚本不依赖已清除的 `HOME`。生产预检会把三个 service 的完整显式 `Environment`、上述 pre-exec unset 集合与安装器定义精确比对；同时要求 timer 可重复触发的 `Type=oneshot`、`RemainAfterExit=no`、`Restart=no`，拒绝 `ExecCondition`、额外 pre/post 命令与扩展成功退出码，并校验有效 `WorkingDirectory`、`ExecStart` 与 `ExecStartEx`。`ignore_errors` 必须为 `no` 且不能使用 `-`、`+`、`!` 等执行前缀，异机门禁或同步失败不能被记作成功。预检拒绝任何 `EnvironmentFile=`、`PassEnvironment=`，也拒绝缺少或增加字段的 `UnsetEnvironment=` 及对应 drop-in。升级已有节点时必须以 root 重新运行 `./infra/ops/install-backup-timers.sh`，旧单元不会被当作合格配置继续发布。该变量是运维确认而不是自动证明，不能替代隔离恢复与 PITR 演练。

## 远端部署控制面

远端部署配置由目标机持有，CI / Ansible 不在每次发布时下发 `deploy.remote.env`：

- 目标机自持 `${DEPLOY_APP_DIR}/.deploy/remote.env`
- 目标机自持 `${DEPLOY_APP_DIR}/.env.prod.shared` / `${DEPLOY_APP_DIR}/.env.prod.secrets`
- 运行时派生 secrets 通过 `GENERATED_ENV_SECRET_REF` 写入远端 secret backend；`${DEPLOY_APP_DIR}/.env.prod.generated.secrets` 仅保留空占位文件
- 目标机自持 Vault **最小权限运行 token** 文件：
  - `${DEPLOY_APP_DIR}/.secrets/vault/token`
- 该文件禁止保存初始化 root token。`vault-runtime-token.sh configure` 用 root-only init material
  创建无 `default` policy 的孤儿 periodic token：shared/secrets 两条 KV v2 路径只能读取，generated
  路径只能创建、读取和更新，并只额外拥有 `lookup-self`、`renew-self`、`capabilities-self` 与为幂等
  Vault bootstrap 核对 mount 所需的 `sys/mounts` 只读权限；其他路径默认拒绝。默认 period 为 72
  小时，systemd 每 12 小时续期，部署前要求 TTL 至少还有 12 小时。
- shared env / generated env secrets 由 `${DEPLOY_APP_DIR}/.deploy/remote.env` 中的 secret ref 决定
  （默认 `SECRET_BACKEND=vault-kv-v2`）
- `.deploy/remote.env` 只接受注册表、环境文件路径、secret backend 和 Vault 运行 token 参数；发布回滚读取的 `releases/*.env` 只接受 `TAG`、`DEPLOYED_AT` 和三个不可变镜像引用。三类低权限状态文件分别使用独立字段白名单，不能注入 `PATH`、`SCRIPT_DIR`、`PYTHONPATH`、`LD_PRELOAD` 等进程控制字段。初始化器会在重写已有 `remote.env` 前先验证原文件；未知或拼错字段会保留现场文件并使初始化、部署或回滚立即失败，新增控制面字段时必须同步代码白名单与契约测试。
- GitHub Actions 远端发布使用 `REGISTRY_AUTH_MODE=workflow-token`：每个 job 的短期
  `github.token` 经 SSH 标准输入传递，只写入远端临时 `DOCKER_CONFIG` 并在结束时删除；目标机不保存
  个人 PAT 或长期 GHCR pull token。`persistent-secret` 只用于明确管理的非 GitHub 兼容链路
- CI / Ansible 仅传发布标识与镜像引用；GitHub Actions 传完整 commit SHA、三个
  `image@sha256:...` 引用和一次性 registry token，仓库本地/Ansible 兼容链路仍可使用
  `TAG` / `ROLLBACK_TAG`

如果远端部署控制面变更，直接在目标机执行：

```bash
cd /opt/stuhelper
./infra/ops/init-remote-deploy-config.sh
```

Vault 已初始化、解封并写入三条 secret ref 后，由 root 一次性收敛运行权限并安装续期 timer：

```bash
cd /opt/stuhelper
sudo VAULT_ROOT_INIT_FILE=/var/lib/stuhelper/vault-credentials/init.json \
  ./infra/ops/vault-runtime-token.sh configure
sudo systemctl status stuhelper-vault-token-renewal.timer --no-pager
sudo -u stuhelper REMOTE_DEPLOY_CONFIG_FILE=/opt/stuhelper/.deploy/remote.env \
  ./infra/ops/vault-runtime-token.sh check
```

`init.json` 与 unseal key 保持 `root:root 0600`；运行 token 文件保持 `stuhelper:stuhelper 0600`。
初始化脚本重复执行时不会再用 root token 覆盖已经安装的 scoped token。Vault 仍采用人工解封：主机
重启后必须先解封；在未解封期间续期与部署均失败关闭，不能通过延长为永久 token 绕过。

如果是远端部署，实际链路里会先执行 `infra/ops/remote-preflight.sh`，检查：

- `.deploy/remote.env` 是否就位
- Vault 运行 token 是否只有约定策略、TTL 是否高于安全下限、三条精确 KV 路径是否可读，以及续期
  timer 是否已安装、启用并处于 active
- 共享配置 / secrets 文件是否就位
- 备份目录是否存在
- 生产 PostgreSQL TLS 是否强制开启并使用 `verify-ca` 或更严格的 `verify-full`
- 本机宝塔 Nginx 主站和 join 入口配置是否满足 `infra/ops/nginx-public-ingress-preflight.sh` 契约
- 公网 SSO 入口是否已经具备可用 TLS，并且 `sso.stuhelper.com/.well-known/openid-configuration` 返回有效 Casdoor OIDC discovery
- PostgreSQL 逻辑备份 / base backup / backup sync timer 是否已启用
- `BACKUP_DATABASE_URL` / `REPLICATION_DATABASE_URL` / `BACKUP_OBJECT_STORAGE_ENDPOINT` / `BACKUP_OBJECT_STORAGE_BUCKET` / `BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID` / `BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY`，以及仅在独立故障域验证完成后设置的 `BACKUP_OBJECT_STORAGE_OFF_HOST_CONFIRMED=true`

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

## GitHub Actions 构建与远端部署

Pull Request、`develop` 和 `main` push 会触发 `.github/workflows/ci.yml`。按变更路径选择的门禁包括：

- Go lint / test / race / coverage / build、`gosec`、`govulncheck` 和可逆迁移验证；
- OpenAPI lint、Go/TypeScript/capability 生成物 drift；
- Web、Admin、UniAppX lint / type-check / unit / build，以及桌面和移动 Playwright；
- Koishi 全工作区依赖审计、单元测试、真实启动和 Console Playwright；
- npm/Yarn 依赖审计、Semgrep、CodeQL、完整 Git 历史 Gitleaks；
- 应用候选镜像的 `HIGH` / `CRITICAL` Trivy 门禁，以及 22 个受管运行时镜像额外 `UNKNOWN` 策略、限时例外和 VEX 校验。

受信任的 `develop` / `main` push 只有在 `CI / Required` 和两项 CodeQL 都成功、提交仍是实时 branch
head 后，才调用
`.github/workflows/publish-images.yml`：

1. 使用完整 commit SHA 构建 backend / frontend / admin；
2. 扫描本地候选镜像；
3. 把同一镜像推送到 `ghcr.io/stuhelper/*:<full-commit-sha>`；
4. 解析最终 manifest digest，并签发 provenance 与 CycloneDX SBOM attestation；SBOM JSON 作为
   Actions evidence 保留 30 天；
5. 更新仅供人类识别的 `develop-latest` 或 `latest` alias。

`.github/workflows/deploy.yml` 同时支持手工晋级，以及由 `main` CI 调用 staging 或 production。
Forward Deploy 不接受人工 commit SHA；候选固定为当前 workflow ref 的 `github.sha`，并要求显式
选择 `staging`、`after-staging`、`direct` 或 `break-glass` promotion mode。工作流先在不绑定 environment、
不读取部署 secrets 的 job 中验证：实时 branch head、`Required`、Go 与 JavaScript/TypeScript
CodeQL、镜像所属仓库、签发工作流、源分支、源提交和 digest。验证通过后才进入 environment，
审批完成后、任何 SSH 前再次校验实时 branch head、checks 和所选 promotion policy，再通过固定 SSH
host key 上传带 SHA-256 传输校验的唯一 bundle。远端 `infra/ops/remote-ci-release.sh` 使用短期 GHCR
token 完成 registry 登录，并依次执行 preflight、deploy、业务 smoke 和严格可观测性 smoke。

仓库变量 `STAGING_AUTO_DEPLOY_ENABLED=true` 时，`main` 的 CI 和三个镜像发布成功后自动部署同一
SHA 到 staging。production 自动路径由两个变量共同控制：

- `PRODUCTION_AUTO_PROMOTION_ENABLED=true` 且 `PRODUCTION_PROMOTION_MODE=after-staging`：只在同
  SHA staging 成功后创建 production deployment；
- `PRODUCTION_AUTO_PROMOTION_ENABLED=true` 且 `PRODUCTION_PROMOTION_MODE=direct`：不依赖
  staging，三个镜像发布成功后直接创建 production deployment。

production 必须由受保护 environment 的唯一 reviewer `Xauryan` 审批，批准后才真正连接目标机。
`direct` 至少需要 24 个字符的变更上下文；`break-glass` 是独立的手工事故模式，不能配置成自动
promotion。两种模式都不绕过 checks、provenance、digest、当前 `main` head、审批后复验、远端预检、
备份或 smoke。自动开关默认保持关闭，直到 production 专用部署身份、环境 secrets、Vault、备份和
一次受控发布演练全部就绪；staging 暂缓期间采用 `direct`，未来隔离 staging 就绪后切换为
`after-staging`。

PR 的旧 run 会在新 push 后自动取消；`develop` / `main` 的可信 push 使用独立 run，不会相互取消，
也不会让旧 production approval 阻塞新 head 的 CI。registry mutation 全局串行，staging / production
mutation 分环境串行。若分支在排队或等待 environment 审批期间前移，部署前二次校验会拒绝旧候选。

生产分支真正部署到线上之前，打包阶段和 `remote-preflight.sh` 会共同避免：

- 远端缺少 `.deploy/remote.env`
- 远端缺少共享配置 / secrets
- backup timer 或 backup sync timer 没启
- WAL 归档目录没准备好
- 部署 bundle 不是从已提交的干净工作区打包
- 主站生产机的宝塔 Nginx `stuhelper.com` / `join.stuhelper.com` server block 漂移
- `join.stuhelper.com` DNS/TLS 或 `/verify/<code>` 入口漂移
- 主站上的 `/verify` / `/verify/*` 被错误兼容或兜底到旧流程

第一次准备远端服务器：

```bash
sudo bash infra/ops/bootstrap-ubuntu2404.sh
```

这个脚本除了装 Docker / Compose 和 Go 1.26 之外，还会准备：

- 部署目录
- `.deploy/remote.env`
- `.secrets/vault/token` 占位文件
- 最小权限 Vault token 配置脚本和 12 小时续期 timer 安装入口（需要在 Vault 初始化/seed 后由 root
  执行 `vault-runtime-token.sh configure`）
- PostgreSQL 逻辑备份 timer
- PostgreSQL base backup timer
- PostgreSQL backup sync timer
- WAL 归档目录

`production` GitHub environment 必须使用以下 secrets；启用 staging 时创建同名、不同值的
`staging` environment，禁止与生产复用部署身份或目标：

- `DEPLOY_HOST`
- `DEPLOY_PORT`
- `DEPLOY_USER`
- `DEPLOY_APP_DIR`
- `DEPLOY_SSH_KEY`
- `DEPLOY_SSH_KNOWN_HOSTS`（固定目标 SSH host public key，禁止 TOFU）

部署 SSH 凭据必须属于专用、可撤销的 deploy identity，不能把维护者日常 root 私钥直接上传为
GitHub secret。目标仍使用 rootful Docker 时，`docker` group 实际接近宿主 root 权限；应至少使用
独立无密码登录账号、最小文件权限和独立 key，长期再迁移到受限 sudo/gateway 或短期 SSH 证书。

浏览器构建参数使用 GitHub repository variables，现行名称见
[GitHub 仓库与 Actions 治理](github-migration.md#repository-variables)。真实密钥不得放在 repository
variables、workflow YAML 或构建参数中。

远端主机自身还必须提前准备：

- `${DEPLOY_APP_DIR}/.deploy/remote.env`
- `REGISTRY=ghcr.io` 与 `REGISTRY_AUTH_MODE=workflow-token`
- `${DEPLOY_APP_DIR}/.env.prod.shared`
- `${DEPLOY_APP_DIR}/.env.prod.secrets`
- `${DEPLOY_APP_DIR}/.env.prod.generated.secrets`（应为空占位）
- `${DEPLOY_APP_DIR}/.secrets/vault/token`（只能是 `vault-runtime-token.sh configure` 生成的 scoped
  periodic token，不能是 Vault 初始化 root token）

## 回滚

GitHub `Rollback` 手工作业选择 `staging` 或 `production` environment，并要求
`commit_sha=<previous-release-full-40-character-commit-sha>`。它不接受可变 tag、短 SHA 或空值。

回滚本质上是：

1. GitHub Actions 先验证当前 workflow controller 的 branch head、`Required` 和双语言 CodeQL
2. 把三个历史完整 SHA tag 解析为 digest，并验证 provenance
3. environment 审批后上传当前可信 controller bundle；不 checkout 或执行历史运维脚本
4. 远端 `remote-ci-release.sh rollback` 使用本次 job 的短期 GHCR token，按三个 digest 拉取
   backend / frontend / admin 镜像
5. 重新执行当前 rollback controller，并再次跑业务与严格可观测性 smoke check

本地应急入口仍保留；未传 `ROLLBACK_TAG` 时会尝试读取 `.deploy/releases.log` 的上一条成功版本：

```bash
# 项目根目录下运行
make prod-rollback
```

## Ansible 入口

如果你希望把远端机器准备、发布和回滚都纳入 playbook，可以直接用：

```bash
# 项目根目录下运行
make ansible-bootstrap

# 发布/回滚前提供一次性或短期的 GHCR 只读凭据；不得写进 inventory
export REGISTRY_USERNAME=<ghcr-user>
export REGISTRY_PULL_TOKEN=<short-lived-read-packages-token>
make ansible-deploy-staging
make ansible-deploy-prod
make ansible-rollback-staging
make ansible-rollback-prod
```

第一次用之前，先准备 inventory：

- `infra/ansible/inventory/staging.ini`
- `infra/ansible/inventory/production.ini`

仓库里已经给了同目录示例文件，可以直接改。

Ansible release playbook 与 GitHub Actions 使用同一个 `remote-ci-release.sh` 包装器，把
`REGISTRY_PULL_TOKEN` 作为 stdin 传给远端并设置 `no_log: true`；token 不进入远端 environment、
inventory、命令参数或持久 Docker config。目标机仍应配置
`REGISTRY_AUTH_MODE=workflow-token`。控制端应使用可撤销、最小 `read:packages` 且尽可能短期的凭据，
运行结束立即从当前 shell 清除；不要复用个人日常登录 token。

控制端使用仓库固定的 Ansible Core 版本；建议安装到项目内的忽略目录，避免污染系统 Python：

```bash
python3 -m venv .venv/ansible
.venv/ansible/bin/pip install --requirement infra/ansible/requirements.txt
export PATH="$PWD/.venv/ansible/bin:$PATH"
```

部署 playbook 在控制端以 `playbook_dir` 锚定 bundle 构建脚本和上传文件；构建脚本自行使用
仓库绝对路径发布 `infra/generated/deploy/stuhelper-deploy-bundle.tar.gz`。不要重新传入相对
输出路径：脚本内部的 `git -C` 会改变 Git 子进程解析该路径的基准。
