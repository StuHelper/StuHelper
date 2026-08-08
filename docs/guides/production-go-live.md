---
type: guide
audience: ops
status: current
authoritative-source: docker-compose.prod.yml + infra/ops/*.sh + infra/nginx/baota-stuhelper.conf + infra/nginx/baota-campus-connector-stream.conf.template + server/migrations/ + server/internal/app/modules.go
last-verified: 2026-08-07
---

# 生产上线缺漏清单与执行指导

本文用于当前 B/2B 架构的生产上线：本地仓库、部署脚本、配置模板和 runbook 是唯一事实来源；生产环境只做可复现部署、smoke 和日志确认，不把生产手工修改当最终状态。

## 当前入口约定

- `stuhelper.com`：主站、后台、API、账号中心、学生认证、QQ 绑定、授权应用和开发者应用。
- `join.stuhelper.com`：加群验证业务域，唯一公开验证链接是 `https://join.stuhelper.com/verify/<code>`。
- `sso.stuhelper.com`：Casdoor，唯一公开登录认证系统和 OIDC issuer。
- `connector.stuhelper.com:9444`：校园 Connector 专用原始 TCP 入口；不是 HTTP 站点，TLS 1.3 mTLS 在
  StuHelper Gateway 内终止。

主站生产 Compose 只把业务服务绑定到回环地址，公网 `80/443` 和 Connector TCP `9444` 只由宝塔
Nginx 监听。

默认回环端口：

```text
backend 127.0.0.1:18080
web     127.0.0.1:18000
admin   127.0.0.1:18001
campus connector gateway 127.0.0.1:19444
```

## 上线前阻断项

| 项目 | 要求 |
|------|------|
| DNS | `stuhelper.com`、`www.stuhelper.com`、`join.stuhelper.com`、`sso.stuhelper.com` 指向对应公网入口；启用校园 Connector 时，`connector.stuhelper.com` 也指向主站公网边缘；DNS 本身不会建立内网连接或 TCP 代理 |
| 宝塔 Nginx | 主站机合并 `infra/nginx/baota-stuhelper.conf`；Connector 使用 `infra/nginx/baota-campus-connector-stream.conf.template` 渲染出的 stream 配置做原始 TCP 透传；Casdoor 入口按 `infra/nginx/baota-casdoor-sso.conf` 或等价配置反代 |
| 生产 env | 使用 `infra/ops/init-prod-env.sh` 生成模板，替换占位符；不得提交真实 `.env.prod.*` |
| 生产目标清单 | 从 `infra/ansible/inventory/production.example.ini` 创建被 Git 忽略的 `production.ini`，填写真实非本机主机和 SSH 用户；空清单或示例占位符必须阻断部署 |
| Secret backend | 生产使用非 repo 的 secret backend；真实 token、密码、对象存储密钥不写入仓库 |
| PostgreSQL / Redis | 生产 PostgreSQL TLS 默认开启；Redis 使用 StuHelper 独立实例，不复用全局 Redis |
| 对象存储与备份 | 配置 HTTPS S3 兼容对象存储和备份对象存储，至少完成一次取回校验 |
| Admission 数据 | `admission-bootstrap-production-data.sh` 可重复执行并准备 BUAA 和目标 QQ 群策略 |
| Koishi/NapCat | 独立节点部署，插件包来自本地当前代码构建产物，不在生产 `node_modules` 手改源码 |

## 生产环境核心变量

```env
WEB_PUBLIC_URL=https://stuhelper.com
ADMIN_PUBLIC_URL=https://stuhelper.com/admin/
ADMISSION_PUBLIC_BASE_URL=https://join.stuhelper.com
WEB_VITE_WEB_URL=https://stuhelper.com
WEB_VITE_SSO_URL=https://sso.stuhelper.com

SSO_PUBLIC_SMOKE_ENABLED=true

CASDOOR_ISSUER=https://sso.stuhelper.com
CASDOOR_PUBLIC_AUTH_BASE_URL=https://sso.stuhelper.com
CASDOOR_INTERNAL_ADDRESS=
CASDOOR_REDIRECT_URI=https://stuhelper.com/api/v1/auth/callback
CASDOOR_ADMIN_REDIRECT_URI=https://stuhelper.com/api/v1/auth/callback
CASDOOR_UNIAPP_REDIRECT_URI=https://stuhelper.com/api/v1/auth/callback

CORS_ORIGINS=https://stuhelper.com,https://join.stuhelper.com,https://sso.stuhelper.com
FRONTEND_METRICS_ALLOWED_ORIGINS=https://stuhelper.com,https://join.stuhelper.com
OPEN_PLATFORM_CONSENT_BASE_URL=https://stuhelper.com
OPEN_PLATFORM_ACCOUNT_BASE_URL=https://stuhelper.com
TOKEN_COOKIE_SECURE=true
TOKEN_COOKIE_DOMAIN=.stuhelper.com

ADMISSION_PRODUCTION_READINESS_ENABLED=true
ADMISSION_READINESS_REQUIRED_PLATFORM=qq
ADMISSION_READINESS_REQUIRED_GUILD_IDS=178037297

# 仅在校园 Connector Gateway 上线时启用；CIDR 必须来自批准的稳定校园节点出口。
CAMPUS_CONNECTOR_GATEWAY_ENABLED=true
CAMPUS_CONNECTOR_GATEWAY_PUBLIC_HOST=connector.stuhelper.com
CAMPUS_CONNECTOR_GATEWAY_PUBLIC_PORT=9444
CAMPUS_CONNECTOR_GATEWAY_EXTERNAL_PORT=19444
CAMPUS_CONNECTOR_ALLOWED_SOURCE_CIDRS=REPLACE_WITH_APPROVED_CAMPUS_CONNECTOR_SOURCE_CIDRS
NGINX_PUBLIC_INGRESS_PROFILE=app-all
ADMISSION_READINESS_REQUIRED_SCHOOL_CODES=4111010006
ADMISSION_READINESS_REQUIRED_SCHOOL_IDS=
ADMISSION_READINESS_REQUIRED_BOT_CREDENTIAL_NAME=koishi-runtime
ADMISSION_READINESS_REQUIRED_BOT_CREDENTIAL_AUDIENCE=/api/v1/bot/*
ADMISSION_READINESS_REQUIRED_BOT_CREDENTIAL_SCOPES=bot.qq_binding.consume,bot.qq_verification.read,bot.admission.session,bot.admission.event,bot.admission.review,bot.admission.forward,bot.member_blacklist.read,bot.member_blacklist.manage
ADMISSION_PUBLIC_SMOKE_ENABLED=true
ADMISSION_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS=false
ADMISSION_PUBLIC_SMOKE_CURL_INSECURE=false
PUBLIC_WEB_AUTH_BROWSER_SMOKE_ENABLED=true
PUBLIC_WEB_AUTH_BROWSER_SMOKE_ALLOW_LOCAL_TARGETS=false

STUDENT_VERIFICATION_PRODUCTION_READINESS_ENABLED=true
STUDENT_VERIFICATION_READINESS_REQUIRED_SCHOOL_CODES=4111010006
STUDENT_VERIFICATION_READINESS_REQUIRED_METHODS=real_name_identity_check,school_sso,manual_material_review
STUDENT_VERIFICATION_READINESS_EXPECTED_SYNC_SECONDS=604800
STUDENT_VERIFICATION_READINESS_EXPECTED_WARNING_SECONDS=691200
STUDENT_VERIFICATION_READINESS_EXPECTED_HARD_EXPIRY_SECONDS=1209600
STUDENT_VERIFICATION_READINESS_EVIDENCE_FILE=infra/generated/student-verification-production-readiness.json

STUHELPER_FRESHMAN_MATERIAL_HOSTS=stuhelper.com,join.stuhelper.com

OPENFGA_API_URL=http://openfga:8080
OPENFGA_RESOURCE_SMOKE_MODE=container
OPENFGA_STORE_ID=
OPENFGA_MODEL_ID=
```

`make ansible-deploy-prod`、staging、bootstrap 和 rollback 都由
`run-ansible-playbook.sh` 统一验证清单。入口会先用 `ansible-inventory` 证明 `stuhelper` 组至少包含一个
非 localhost 主机；缺文件、空文件、示例占位符、解析失败或空组均非零退出。本机未全局安装 Ansible
时，脚本通过 `uvx` 使用 `infra/ansible/requirements.txt` 锁定的版本。清单本身属于部署私有状态，不能
提交到仓库。

`OPENFGA_RESOURCE_SMOKE_MODE=container` 会直接运行 `BACKEND_IMAGE_REF` 发布镜像内的
`/app/openfga-resource-smoke`。生产验收不得临时拉取 Go 工具链、下载模块或现场编译，确保
验证对象就是已经发布的固定制品。

OpenFGA 生产 `STORE_ID` / `MODEL_ID` 不在共享样例中手填占位符，必须保持为空。`bootstrap-platform.sh` / OpenFGA bootstrap 创建 store 和 authorization model 后，把真实 `OPENFGA_STORE_ID`、`OPENFGA_MODEL_ID` 写入 `GENERATED_ENV_FILE`；`GENERATED_ENV_SECRET_REF` 只承载 generated secret env，不承载这两个非 secret runtime ID。生产部署脚本允许空值等待 bootstrap 写入，但会拒绝 `REPLACE_WITH_OPENFGA_*` 这类共享样例占位符进入部署。

`CASDOOR_INTERNAL_ADDRESS` 生产默认必须为空，让后端按公开 issuer / public auth base URL 校验 OIDC 行为；不要沿用本地开发的 `host.docker.internal:8085` 或旧 Compose 内网地址。只有在有明确同机内网反代设计、并完成 SSO public smoke 验证 issuer 仍为 `https://sso.stuhelper.com` 时，才允许在非 repo secret/env 中覆盖。

Koishi 独立节点必须注入：

```env
STUHELPER_PLATFORM_BASE_URL=https://stuhelper.com
STUHELPER_PLATFORM_SERVICE_TOKEN=<redacted>
STUHELPER_FRESHMAN_MATERIAL_HOSTS=stuhelper.com,join.stuhelper.com
```

`STUHELPER_PLATFORM_SERVICE_TOKEN` 的值必须与后端生产 env 中真实 `BOT_SERVICE_TOKEN` 完全一致。该真实 token 不由 `init-prod-env.sh`、`bootstrap-platform.sh` 或 `prod-deploy.sh` 自动创建；它应来自已创建的 bot service credential、secret backend 或受控运维注入流程。不要把真实值写入 runbook、脚本或 git。

`BOT_SERVICE_TOKEN=REPLACE_WITH_BOT_SERVICE_TOKEN_BOOTSTRAP` 只允许出现在生产样例和 fresh init 生成的占位状态；legacy init 会保持 `BOT_SERVICE_TOKEN` 为空。运行 `prod-deploy.sh` 前，运维必须把真实 bot service credential 注入后端生产 env 或 secret backend，并把同一个值同步给 Koishi 节点的 `STUHELPER_PLATFORM_SERVICE_TOKEN`。`prod-deploy.sh` 会要求 `BOT_SERVICE_TOKEN` 非空并拒绝占位符，避免 Koishi admission 运行时以样例 token 调后端导致 401。

## Nginx 入口契约

主站入口：

```text
stuhelper.com /api/*      -> http://127.0.0.1:18080
stuhelper.com /health/*   -> http://127.0.0.1:18080
stuhelper.com /admin/*    -> http://127.0.0.1:18001
stuhelper.com /verify     -> 404
stuhelper.com /verify/*   -> 404
stuhelper.com /*          -> http://127.0.0.1:18000
```

Join 入口：

```text
join.stuhelper.com /verify/<code> -> http://127.0.0.1:18000
join.stuhelper.com /student-verification/manual-camera/* -> http://127.0.0.1:18000
join.stuhelper.com /admission/freshman/camera/*           -> 404
join.stuhelper.com /api/v1/admission/freshman/camera-handoffs/* -> 404
join.stuhelper.com /verify                 -> 404
join.stuhelper.com / 和主站业务页面路径   -> 404
join.stuhelper.com /api/*                  -> http://127.0.0.1:18080
join.stuhelper.com /health/*               -> http://127.0.0.1:18080
```

SSO 入口：

```text
sso.stuhelper.com /.well-known/openid-configuration -> Casdoor
sso.stuhelper.com /.well-known/jwks                 -> Casdoor
sso.stuhelper.com /login/*                          -> Casdoor
sso.stuhelper.com /api/*                            -> Casdoor
```

校园 Connector 入口：

```text
校园 Connector
  -> connector.stuhelper.com:9444
  -> 宝塔 Nginx stream 原始 TCP 透传（不终止 TLS）
  -> 127.0.0.1:19444
  -> StuHelper Gateway TLS 1.3 mTLS
```

Connector stream 配置必须只有精确的公网 `listen`、loopback `proxy_pass`、有界超时、批准来源的
`allow` 和末尾 `deny all`。不得出现 `listen ... ssl`、`ssl_certificate`、`proxy_ssl*`、HTTP 反代或
CDN TLS 终止；否则客户端证书不会到达 Gateway。宿主防火墙必须使用同一批准 CIDR 开放公网端口，不能
添加 `0.0.0.0/0` 兜底规则。

执行审计：

```bash
NGINX_PUBLIC_INGRESS_PROFILE=stuhelper ./infra/ops/nginx-public-ingress-preflight.sh
NGINX_PUBLIC_INGRESS_PROFILE=sso ./infra/ops/nginx-public-ingress-preflight.sh

# 主站启用 Connector 后，同时审计 HTTP 与 stream 配置。
CAMPUS_CONNECTOR_ALLOWED_SOURCE_CIDRS=<approved-ipv4-cidr> \
  NGINX_PUBLIC_INGRESS_PROFILE=app-all \
  ./infra/ops/nginx-public-ingress-preflight.sh
```

应用或恢复宝塔 vhost 模板：

```bash
# 默认只 dry-run，不写文件
./infra/ops/apply-baota-nginx-templates.sh --profile all

# 主站和 SSO 同机时可一次应用；脚本会先备份目标文件，再 nginx -t，最后按需 reload
sudo ./infra/ops/apply-baota-nginx-templates.sh --profile all --apply --reload --preflight

# 如果宝塔面板重写了 sso.stuhelper.com.conf，导致 discovery 变成 404，优先只恢复 SSO vhost
sudo ./infra/ops/apply-baota-nginx-templates.sh --profile sso --apply --reload --preflight
./infra/ops/sso-public-smoke.sh

# Connector 单独 dry-run；不会写配置或 reload。
CAMPUS_CONNECTOR_ALLOWED_SOURCE_CIDRS=<approved-ipv4-cidr> \
  ./infra/ops/apply-baota-nginx-templates.sh --profile connector

# 主站首次接入时只应用 Connector stream；脚本会渲染 allowlist、备份旧目标、nginx -t、reload 和 preflight。
sudo \
  CAMPUS_CONNECTOR_ALLOWED_SOURCE_CIDRS=<approved-ipv4-cidr> \
  ./infra/ops/apply-baota-nginx-templates.sh \
    --profile connector \
    --apply \
    --reload \
    --preflight

# 主站后续可用 app-all 同时收敛主站 HTTP 与 Connector stream；它不包含独立 SSO 主机配置。
sudo \
  CAMPUS_CONNECTOR_ALLOWED_SOURCE_CIDRS=<approved-ipv4-cidr> \
  ./infra/ops/apply-baota-nginx-templates.sh \
    --profile app-all \
    --apply \
    --reload \
    --preflight
```

注意：`apply-baota-nginx-templates.sh` 是仓库事实来源的一部分，生产不应直接手改 vhost 后停留在漂移状态。历史 `--profile all` 为兼容既有流程，仍只表示 `stuhelper+sso` 两组 HTTP vhost，不会隐式要求 Connector；主站 HTTP 与 Connector 应使用 `--profile app-all`，仅 Connector 使用 `--profile connector`，SSO-only 主机仍使用 `--profile sso`。`--profile sso` 会同时安装 `infra/nginx/baota-casdoor-sso.conf` 和 `infra/nginx/baota-casdoor-sso-well-known-extension.conf`。后者目标路径是宝塔扩展目录 `/www/server/panel/vhost/nginx/extension/sso.stuhelper.com/stuhelper-sso-well-known.conf`，用于在宝塔重写主 vhost 但保留 extension include 时继续让 OIDC discovery/JWKS 走 Casdoor。Connector 目标路径是 `/www/server/panel/vhost/nginx/tcp/connector.stuhelper.com.conf`，必须被宝塔主配置的 `stream { include .../tcp/*.conf; }` 实际加载。若主站和 SSO 不在同一台机器，只在对应机器执行对应 profile；不要把另一台机器的 vhost 目标路径作为临时手改。宝塔面板保存站点配置后可能重写 vhost。若 `sso-public-smoke.sh` 报 discovery 404，先用上面的 `--profile sso` 恢复，再审计 `NGINX_PUBLIC_INGRESS_PROFILE=sso ./infra/ops/nginx-public-ingress-preflight.sh`。

## Admission 生产数据

幂等准备：

```bash
./infra/ops/admission-bootstrap-production-data.sh
./infra/ops/admission-production-readiness.sh
```

验收条件：学校目录中存在 `code=4111010006` 的北京航空航天大学；管理后台学校配置页以 `schools` 目录为基表展示所有已录入学校，缺少 `school_configs` 的学校按默认停用配置展示，只有 `school_configs.enabled=true` 才进入学生认证和 admission 白名单。当前 admission 白名单只开放北航，对外、前端表单和运维检查使用学校代码 `4111010006`，不得再把旧五位学校 ID 作为业务事实或配置入口。公开学生认证和学校 SSO 路径都应以 `schoolCode` 为主识别字段。人工审核的跨设备拍照只使用 student-verification 契约：桌面端创建并查询 `/api/v1/student-verification/applications/{applicationID}/manual-review/camera-handoffs`，手机端使用 `/student-verification/manual-camera/{token}` 及 `/api/v1/student-verification/manual-camera-handoffs/{token}` 系列端点；旧 admission freshman camera 页面和 SSE 永久返回 404。`group_admission_policies` 至少包含 `platform=qq, guild_id=178037297, auto_approve_verified_join=true, auto_approve_unverified_join=true, forward_raw_material_to_qq=false`。`bot_service_credentials` 中存在 `koishi-runtime`，未吊销、未过期，audience 包含 `/api/v1/bot/*`，scopes 覆盖 QQ 绑定、admission session/event/review/forward 和 member blacklist。

北航学生认证不再允许生产 Web API 直接连接学校 Oracle 或 LDAP。`docker-compose.prod.yml` 会强制把
`EXTERNAL_STUDENT_SOURCE_ENABLED` 和 Oracle host/user/password 清空；学校内网访问只能发生在独立校园
连接器节点。该节点主动出站连接 StuHelper 的专用 mTLS 网关，只能执行注册表中已经审批的“学校账号
校验”或“完整名册快照”操作，不能提供 VPN、任意 TCP 转发、任意 SQL 或 shell 能力。完整威胁模型、
配置格式和字段映射见 `infra/campus-connector/README.md`。

首次准备中心与节点身份时执行：

```bash
CAMPUS_CONNECTOR_GATEWAY_PUBLIC_HOST=connector.stuhelper.com \
  ./infra/ops/generate-campus-connector-pki.sh

# 重复执行只校验，不覆盖或静默轮换已有身份
./infra/ops/generate-campus-connector-pki.sh \
  --check \
  --gateway-host connector.stuhelper.com

# 在可信 PKI 工作站导出与完整 PKI 分离的五文件中心运行时包
./infra/ops/prepare-campus-connector-gateway-runtime.sh \
  --source infra/generated/campus-connector-pki \
  --output infra/generated/campus-connector-gateway-runtime \
  --gateway-host connector.stuhelper.com
```

生成器创建两条相互独立的 CA、中心网关 TLS 证书、节点 mTLS 证书、节点 Ed25519 业务签名密钥和中心
X25519 快照接收密钥，并把私钥限制为 `0600`。`make prod-init` 不会生成或复制完整 PKI；启用 Gateway
时，它只校验已经安装的五文件 runtime bundle，并从该包核对 snapshot key ID。完成节点登记后必须把
`infra/generated/campus-connector-pki/authority/` 中的 CA 私钥和完整 PKI 迁入离线 secret store；生产
app 只挂载 `campus-connector-gateway-runtime/`，校园节点只安装 `node/`。严禁把整个 PKI、`authority/`
或节点私钥复制进生产 app/宿主机或提交 Git。五文件 runtime bundle 的目录必须属于生产部署用户并为
`0700`，两把运行时私钥必须属于同一用户并为 `0600`；共享 env 的 `BACKEND_RUNTIME_USER/UID/GID` 必须
分别精确等于该部署用户的账户名与数值 UID/GID，app 与 bootstrap 会以同一身份运行。

启用中心网关前必须完成以下非 secret 接线：

- 为 `CAMPUS_CONNECTOR_GATEWAY_PUBLIC_HOST` 配置真实 DNS；
- 从公网端口到宿主 `127.0.0.1:${CAMPUS_CONNECTOR_GATEWAY_EXTERNAL_PORT}` 配置原样 TCP 透传，TLS 必须
  在 StuHelper app 内终止，HTTP 反向代理或 CDN TLS 终止不能代替；
- 在防火墙只允许批准校园节点来源，并保留连接/证书到期告警；
- 设 `CAMPUS_CONNECTOR_GATEWAY_ENABLED=true` 后运行生产预检。预检会重新验证证书链、SAN、密钥配对、
  30 天到期窗口、精确五文件边界、目录/私钥所有权和 snapshot key ID，任何一项不一致都 fail closed。

Oracle 连接只能使用用户明确指定的既有账号；StuHelper 及其代理不得创建、申请、更换、授权、回收或
修改 Oracle 账号。能用 Navicat 或 DBeaver 登录只证明当前账号和网络大概率可用，不证明具备完整快照
字段权限。若既有账号权限较宽，只记录风险，不对 Oracle 做权限整改；校园节点仍只允许认证和仓库内固定
的 `SELECT`。节点本地配置固定 owner/view、字段映射、最大行数和预期 session username；Oracle 地址、
账号、密码与 CA 只存在于节点 secret store，不能进入 StuHelper 生产 env、注册表、数据库或日志。

Oracle 默认使用 `oracle_tls`（TCPS）并验证学校 CA 与证书名。北航现有接线只能经既有 SSH 跳板访问
1521 时，允许显式使用 `oracle_ssh_tunnel` 例外，但 SSH 隧道由校园节点的 systemd/sidecar 管理，不由
Connector 动态创建。必须固定跳板 host key、私钥 secret、批准的远端 Oracle 地址和 1521；本地仅监听
loopback 高端口，禁止 `GatewayPorts`、SOCKS、动态转发和任意目标。Connector operation 必须同时设置
`allowPlaintextSSHTunnel=true`，清空 Oracle CA/TLS name，并使 `allowedDialTargets` 恰好只有该 loopback
endpoint，因此 Oracle listener redirect 会被拒绝。SSH 隧道不是 Oracle TCPS，不能把公网转发端口冒充
`oracle_tls`；学校提供 TCPS 后必须迁回默认模式。

第一阶段只实施周期性完整快照：默认每 7 天一次、8 天告警、14 天停止新的名册依赖认证；失败从
15 分钟起有界退避，最长 6 小时，成功后恢复 7 天周期。管理员可以在管理后台经
`campus_connector:manage` 与近期 MFA 手动触发；任务持久化、同校同 operation 唯一、3 分钟领取租约、
最多 5 次领取、总期限 24 小时。中心收到加密签名快照后先写独立版本，经过行数、重复、空值、状态码、
来源时间和回退检测，再原子激活。旧 `import-buaa-academic-students.sh` 的 upsert 语义不满足删除/毕业
收敛要求，不得用于新认证域，也不存在“连接器不可用时自动回退旧表”的生产路径。
旧的 `admission-student-source-go-live.sh` 仅保留用于历史版本排障和回滚取证；它编排的是 Web API
直连 Oracle 或 TSV upsert，两种模式都不满足当前完整快照语义，当前生产发布与最终证据不得执行它。
旧文件名 `provision-external-student-source-oracle-readonly.sh` 只作为升级兼容占位，现已明确拒绝执行，
不会打开 Oracle 连接或读取凭据。生产只把用户明确指定的既有账号配置在校园连接器节点的 secret store，
不得用 StuHelper 脚本创建、轮换或调整账号，也不得把 Oracle 凭据重新接回 StuHelper Web API。
同样，历史 `import-buaa-academic-students.sh` 不接受 `sfzjh_enc` 或 `sfzjh_hash` 列；当前仓库不提供该入口
导入证件号，也不得把它改造成新认证域的隐式兼容路径。

北航学校账号校验适配器默认使用 LDAPS 或 LDAP StartTLS，并校验证书名与学校 CA。学校 owner 已确认
当前只开放 RFC1918 IPv4 的明文 389 时，允许只为该 operation 显式配置
`upstreamProtocol=ldap_plain_private_network` 与 `allowPlaintextPrivateNetwork=true`。运行时会拒绝域名、
公网/非 RFC1918 地址、非 389 端口、CA/TLS 名称、代理和从加密模式静默降级。该风险例外只覆盖校园
连接器到 LDAP 的最后一跳；连接器到中心仍必须使用 TLS 1.3 mTLS，EasyTier/Tailscale 等 overlay 也不能
把最后一跳视为已经加密。学校提供可验证的 TLS 端点后必须迁回 LDAPS/StartTLS。个人学号密码只能作为
一次受控验收输入，不能写入代码、JSON、env、数据库、日志或健康检查；成功验收只记录稳定结果分类和
不可逆请求引用。

注册时先把 `registry-manifest.json` 的 operation 保持 `validationStatus=pending`、`enabled=false`，执行
只读 dry-run；核对节点证书、公开签名密钥、学校代码、固定目标、TLS 名称、字段和限额后才 `--apply`。
节点稳定上报心跳且真实 Oracle 完整快照通过质量门禁后，再以新 revision 将对应 operation 标记
`valid` 并启用。不得为了绕过缺失的 Oracle/LDAP CA、服务名或只读凭据而关闭 TLS 校验。

生产部署会在迁移完成、应用启动前执行
`student-verification-production-readiness.sh`。该门禁只读检查学校 profile、要求的方法、连接器证书与
心跳、固定 operation、完整快照签名、实际行数、质量门禁、新鲜度和手动同步唯一索引；不会读取或输出
姓名、学号、证件号、手机号、Oracle 地址或 secret reference。可单独运行
`make prod-student-verification-readiness`，成功证据写入
`infra/generated/student-verification-production-readiness.json`，权限为 `0600`。任何要求学校缺少 active
snapshot、校园连接器未健康或快照超过 14 天时都必须阻断生产发布，不得用旧 `academic.buaa_students`
或 admission readiness 的通过结果替代。

首次生产发布存在一个刻意保留的两阶段启动过程。migration 完成后，`prod-deploy.sh` 会启动与正式后端
相同镜像的 `campus-connector-bootstrap` 服务，并把 `APP_RUNTIME_MODE` 固定为
`campus-connector-bootstrap`。这个进程只初始化 HMAC、PII 加密、PostgreSQL、Redis、OTLP、mTLS Gateway
和完整名册 importer；不监听 8080，不创建 Gin router，不初始化 OIDC、OpenFGA、token service、对象
存储、普通业务 worker、Web 或 Admin。真实校园节点必须通过该 Gateway 上报心跳、operation 健康状态和
签名加密完整快照，授权管理员完成质量审阅与激活后，readiness 才会通过。

若 student verification 或后续 readiness 失败，发布脚本会以非零状态退出，但故意保留
`${STACK_NAME}-campus-connector-bootstrap` 运行，供真实节点继续完成上述准备；它不会记录 release，也不会
提前启动新版 App。修复真实缺口后重新执行同一个 `make prod-deploy`。不得手工伪造 heartbeat、operation
状态或 active snapshot，也不得关闭 readiness。所有 migration/readiness/控制面 evidence 通过后，脚本
才停止并移除 bootstrap、确认 `127.0.0.1:${CAMPUS_CONNECTOR_GATEWAY_EXTERNAL_PORT}` 已释放，再启动明确
固定为 `APP_RUNTIME_MODE=app` 的正式 App/Web/Admin。

普通升级若现有 `${STACK_NAME}-app` 已映射该 loopback 端口且容器内确实监听 9444，则直接复用现有
Gateway，不并行启动 bootstrap。若 App 有映射却没有监听，或该宿主端口属于未知容器/非 Compose 进程，
发布会 fail closed，且不会停止生产 App、未知容器或未知进程。应先只读取证并修复真实端口所有权，不要
用 `docker rm -f`、`kill` 或临时改端口绕过。

当前生产范围只包括北航，要求的方法集合为实名信息校验、学校 SSO 和人工材料审核。两种学校邮件方法
保留在能力注册表中但继续停用，不发送真实邮件，也不得为了通过 readiness 伪造健康状态；待邮件收发
链路、隐私告知和真实投递验收完成后，再显式加入
`STUDENT_VERIFICATION_READINESS_REQUIRED_METHODS`。

## Koishi Admission MVP 配置

生产 `stuhelper-group-guard` 的 admission 配置必须限制职责边界：

```yaml
stuhelper-group-guard:admission:
  platform:
    baseUrl: ${{ env.STUHELPER_PLATFORM_BASE_URL }}
    serviceToken: ${{ env.STUHELPER_PLATFORM_SERVICE_TOKEN }}
  guard:
    targetGroups:
      - '178037297'
  scheduler:
    fallbackScanEnabled: true
    scanIntervalSeconds: 300
  actionStream:
    enabled: true
    reconnectDelaySeconds: 5
  commands:
    enabled: false
  admissionCommands:
    enabled: true
    minAuthority: 4
  moderation:
    enabled: false
  freshmanForward:
    enabled: false
```

如果生产启用 `stuhelper-core` 作为群管中心 WebUI 入口，必须使用 WebUI-only 模式，避免 core 旧运行时模块注册 `report`、`sub`、`config`、`ai` 等命令并与现有生产插件冲突：

```yaml
stuhelper-core:console:
  platform:
    baseUrl: ${{ env.STUHELPER_PLATFORM_BASE_URL }}
    serviceToken: ${{ env.STUHELPER_PLATFORM_SERVICE_TOKEN }}
  runtimeModules:
    enabled: false
```

约束：生产 NapCat / OneBot runtime platform 可以是 `onebot`；后端 admission policy/session/action 的 subject platform 仍是 `qq`。控制台处置 `platform=qq` 的准入 / 复核记录时会优先精确匹配同平台 bot，找不到时回退到同 `botSelfId` 的 QQ 兼容运行时 bot（生产常见为 `onebot`）。禁言、踢人、发消息使用当前 Koishi runtime bot，不切换适配器。`student-query.enableGroupVerify` 不因 admission 上线被整体关闭；冲突应通过新插件 `commands.enabled=false`、`moderation.enabled=false` 和旧插件自己的目标群范围处理。`admissionCommands.enabled=true` 只保留入群认证管理员命令，用于“查询入群认证 / 重发认证链接 / 重新生成认证链接 / 跳过入群认证 / 清空入群未认证次数 / 解除入群拉黑”。其中“跳过入群认证”只跳过本群审核并解除禁言，不代表 StuHelper 学生认证通过；“解除入群拉黑”不隐式清空失败次数，需要重新计数时单独执行“清空入群未认证次数”。当前 MVP 默认关闭 `freshmanForward.enabled`，避免未启用材料转发时每分钟请求 `/api/v1/bot/admission/freshman/applications/pending-forward` 并报错。生产 NapCat 的反向 OneBot WebSocket 账号级配置应把 `reconnectInterval` 控制在 1 秒级、`heartInterval` 控制在 10 秒级，否则 Koishi 重启或短暂断线时验证码消息会落在重连空窗中丢失。

`actionStream.enabled`、`scheduler.fallbackScanEnabled`、`commands.enabled`、`admissionCommands.enabled`、`moderation.enabled` 和 `freshmanForward.enabled` 在 `koishi.yml` 中只作为启动默认值；Koishi 群管中心 WebUI 的“入群认证”页面会把运行时开关写入 `stuhelper_admission_runtime_settings`。`actionStream.enabled`、兜底扫描、消息风控和材料转发保存后立即生效；公开命令和 admission 管理命令只能关闭已注册命令，若启动时没有注册，需要调整 `koishi.yml` 并重启后才能启用。`platform.baseUrl`、`platform.serviceToken`、`scheduler.scanIntervalSeconds`、`actionStream.reconnectDelaySeconds`、`admissionCommands.minAuthority` 和 `admissionCommands.operatorQQIDs` 仍是启动/安全配置，不在 WebUI 热改。

部署包必须从本地当前代码构建：

```bash
./infra/ops/package-koishi-stuhelper-packages.sh
```

该包必须包含 `koishi-plugin-stuhelper-core` 的 `lib/` 和 `dist/`，以及 `koishi-plugin-stuhelper-group-guard`、`koishi-plugin-stuhelper-binding`、`@stuhelper/koishi-shared`、`@stuhelper/koishi-moderation-core` 的运行时 `lib/` 产物；同时必须包含 `koishi/local-workspaces/...` 和 `koishi/STUHELPER_KOISHI_APPLY_LOCAL_WORKSPACES.cjs`，用于把 StuHelper 私有插件固定为本地 `workspace:*` 依赖，避免 Koishi Market 更新普通插件时请求 npm registry 并因 `koishi-plugin-stuhelper-core@0.0.0` 未发布而失败。否则 admission WebUI 页面或 group-guard Console API 会在生产漂移。

若宝塔 Compose 环境只能手工覆盖包，也必须记录包 sha256、覆盖路径和重启步骤；源码修复必须回写到仓库。

## 授权账本首次切换门禁

`000024_authorization_authority_cutover` 之后，production 与 prod-parity 应用不会在切换 marker
仍为 `pending` 时启动。标准 `prod-deploy.sh` 已固定以下顺序：数据库 migration → OpenFGA
启动与 Casdoor/OpenFGA bootstrap → `authorization-ledger-cutover.sh` → 应用启动。不得绕过脚本
单独启动新应用。

首次升级前先确认：

- 目标 Casdoor organization 中预期最高管理员是当前有效的 organization administrator；
- 旧 scoped operator 若确实需要保留，同时具有遗留 Casdoor `school_admin` / `section_*`
  membership 与对应 OpenFGA direct tuple；单边陈旧记录不会被导入；
- pre-deploy PostgreSQL 备份和对象存储同步 evidence 已通过；
- OpenFGA 旧 store ID 正确保留，不能误建空 store 后继续切换。

标准发布会自动执行；需要在维护窗口独立重试时使用：

```bash
./infra/ops/authorization-ledger-cutover.sh prod
```

成功只输出非敏感 JSON：`changed`、64 位 `sourceDigest`、`importedGrantCount` 和
`skippedTupleCount`。首次成功后 marker、每条 imported grant audit 和 projection outbox 在同一
PostgreSQL 事务落地；重复执行返回 `changed=false` 和原 digest/count，不重复导入。fresh install
没有 shadow user 时可以用空集合安全完成，后续 Casdoor organization administrator 在正常登录
或 refresh 时按 ADR-0009 同步。

遇到以下任一情况，脚本会中止且应用保持 fail-closed：OpenFGA 分页读取失败、tuple subject 无法
对应本地/Casdoor 身份、Casdoor legacy role 使用嵌套 group/role/domain、pending marker 下已有
无法安全归属的 grant，或来源 digest 冲突。此时应保留备份并逐项核对 Casdoor、OpenFGA 与 DB，
不得直接 `UPDATE authorization_authority_cutover SET status='completed'`，也不得清空
`authorization_grants` 强行放行。应用回滚时保留已导入 grant、audit、outbox 和 completed marker。

上线 evidence 至少记录发布版本、source digest、导入/跳过数量、marker 完成时间、projection
worker 收敛结果，以及预期管理员的实际登录/降权验证；不得记录 Casdoor token、secret 或用户
敏感字段。

## 发布流程

1. 在本地确认工作树中要发布的代码脚本模板和 runbook 已完整。
2. 生成并填写生产 env，不提交真实 secret：

   ```bash
   ENV_FILE=.env.prod.shared \
   SECRETS_ENV_FILE=.env.prod.secrets \
   GENERATED_ENV_FILE=.env.prod.generated \
   GENERATED_SECRET_ENV_FILE=.env.prod.generated.secrets \
   ./infra/ops/init-prod-env.sh
   ```

3. 本地执行契约和构建验证：

   ```bash
   make check-admission-mvp
   make check-infra-contracts
   cd server && make test && make build
   cd clients && pnpm type-check:all && pnpm build:web && pnpm build:admin
   cd bots/koishi && corepack yarn test && corepack yarn build
   ```

4. 首选 GitHub `Deploy` 受保护工作流。`main` 的可信镜像发布完成后，确认
   `PRODUCTION_PROMOTION_MODE=direct`，等待 production environment 显示候选 SHA、digest 和预验证
   结果，再由 `Xauryan` 审批。Actions 会用短期 GHCR token 调用：

   ```bash
   ./infra/ops/remote-ci-release.sh deploy
   ```

   上述脚本要求 token 从 SSH 标准输入进入，只应由工作流调用，不能把 token 写进命令、shell history
   或 `.deploy/remote.env`。必须脱离 GitHub 手工应急时，先把远端明确切换为
   `REGISTRY_AUTH_MODE=persistent-secret` 并配置独立最小读取凭据，才可在生产机执行：

   ```bash
   ./infra/ops/remote-prod-deploy.sh
   ```

5. 发布后执行 smoke 和日志确认。

## 发布后 smoke

公网入口：

```bash
curl -fsSI https://stuhelper.com/
curl -fsS https://stuhelper.com/health/live
curl -fsS https://stuhelper.com/health/ready
curl -fsSI https://stuhelper.com/admin/
curl -fsS https://sso.stuhelper.com/.well-known/openid-configuration | head
curl -fsSI 'https://join.stuhelper.com/verify/__manual_probe__'
curl -fsSI https://join.stuhelper.com/                         # 应 404
curl -fsSI https://join.stuhelper.com/developers/apps           # 应 404
curl -fsSI https://join.stuhelper.com/verify                  # 应 404
curl -fsSI 'https://stuhelper.com/verify/__manual_probe__' # 应 404
```

`sso.stuhelper.com` 的 OIDC discovery `issuer`、authorize、token 和 JWKS endpoint 必须全部是 `https://sso.stuhelper.com`。如果独立 Casdoor 宝塔 Compose 的 `conf/app.conf` 已经写入 `origin = https://sso.stuhelper.com`，但 discovery 仍返回 `http://sso.stuhelper.com`，不要只修改文件后结束；重启该 Compose 项目的 `casdoor` 容器，再运行 `./infra/ops/sso-public-smoke.sh` 留档。

如果发布时替换了宝塔 `source/` 目录或重建了基础服务容器，必须先运行 `./infra/ops/ensure-baota-runtime-permissions.sh --apply` 归一化 bind mount 权限，再重建 Postgres、Redis、app、frontend、admin。PostgreSQL 的只读 TLS 源文件会由容器入口复制到仅内存 `/tls` tmpfs，并以 UID/GID 70、0600 提供 `server.key` 后再降权启动；Redis 的服务端私钥和仅含密码哈希的 ACL 会复制到 UID/GID 999:1000、0600 的 `/redis-runtime` tmpfs。宿主私钥和 ACL 不能为了解决 UID 差异而放宽读取权限。运行 `prepare-datastore-client-cas.sh` 后，应用侧只能挂载只含公开 `ca.crt` 的 `postgres-client-ca` / `redis-client-ca`。生成的 Prometheus/Alertmanager 配置目录由源码目录 owner 持有、容器读取组使用 `ALERTMANAGER_CONFIG_GID`（默认 65534），目录必须保持 setgid `2750`、文件保持 `0640`；渲染器本身也会在写入内容前强制 `0640`，这样非 root deploy 可以重渲染，容器仍只能按组读取。若部署用户 UID 与源码目录 owner 不同，必须显式设置 `GENERATED_OBSERVABILITY_CONFIG_OWNER_UID` 后再运行权限归一化。已安装的六小时 root 权限守卫是 root-owned 副本；每次 `ensure-baota-runtime-permissions.sh` 行为发生变化或源码目录迁移后，都必须以相同的 owner/GID 参数重新运行 `install-baota-runtime-permission-guard.sh`。远端预检会核对已安装副本、unit 目标路径及持久化 owner/GID，发现漂移即拒绝发布。若同机还有独立 Casdoor 宝塔 Compose，该脚本也会修复 Casdoor `conf/app.conf` 与 `logs/` 的 UID 1000 权限，避免 SSO 502 或 Casdoor 重启循环。该步骤只改权限和 owner，不输出 secret 内容。

`sso-public-smoke.sh` 的 evidence 会记录每个公网请求的 `remoteIP`，生产模式会拒绝本机、私网、链路本地和保留网段解析结果。如果运维机 `/etc/hosts`、代理或诊断用 `SSO_PUBLIC_SMOKE_RESOLVE_IP` 把 `sso.stuhelper.com` 指向非公网地址，本 smoke 必须失败；只有本地契约测试或明确的本地验证才允许设置 `SSO_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS=true`。

自动 smoke：

```bash
./infra/ops/sso-public-smoke.sh
./infra/ops/admission-public-smoke.sh
./infra/ops/public-web-auth-browser-smoke.mjs
./infra/ops/admission-production-readiness.sh
./infra/ops/admission-reviewer-readiness.sh
./infra/ops/admission-mvp-production-evidence.sh
./infra/ops/tencent-ses-template-smoke.sh
RESEND_EMAIL_SMOKE_TO=<recipient-email> ./infra/ops/resend-email-channel-smoke.sh
./infra/ops/smoke-check.sh
./infra/ops/open-platform-production-evidence.sh
./infra/ops/observability-smoke-check.sh
```

学校邮箱 OTP 邮件使用多提供商驱动。生产配置应保持 `EMAIL_DRIVER=multi`、`EMAIL_FROM=noreply@notify.stuhelper.com`、`EMAIL_FROM_NAME=StuHelper 系统邮件`、`EMAIL_STUDENT_VERIFICATION_SUBJECT=学生认证验证码`、`EMAIL_TENCENT_TEMPLATE_ID=49779`、`EMAIL_RESEND_ENDPOINT=https://api.resend.com/emails`；`EMAIL_TENCENT_SECRET_ID`、`EMAIL_TENCENT_SECRET_KEY` 与 `EMAIL_RESEND_API_KEY` 只能放在 `.env.prod.secrets.local` 或部署 secret store。默认发送策略来自 `system_configs.email.delivery_policy`：腾讯云 SES `priority=10` 先发，Resend `priority=20` 作为故障兜底；如需负载均衡，把策略 `mode` 改为 `weighted` 并把多个 provider 设为同一优先级。当腾讯云 `GetSendEmailStatus` 显示 `SendStatus=0` 且 `DeliverStatus=1`，但目标学校邮箱仍不可见时，这属于收件侧投递可见性/隔离问题，不会触发自动 failover；生产可通过管理后台系统配置把 Resend 调整为 `priority=10`、腾讯云调整为 `priority=20`，并记录变更原因。`tencent-ses-template-smoke.sh` 不发送邮件，只调用 `GetEmailTemplate` 确认腾讯云凭据、地域、模板 ID 和审核状态，证据写入 `infra/generated/tencent-ses-template-smoke.json`。`resend-email-channel-smoke.sh` 会真实发送一封测试邮件，必须显式传入 `RESEND_EMAIL_SMOKE_TO`；它复用 `infra/email-templates/tencent-ses/stuhelper-school-email-otp.html` 和 `.txt` 渲染 Resend `html`/`text` 字段，证据只记录收件域名、收件地址哈希前缀和 Resend email ID，不输出 API key 或完整收件地址。

`sso-public-smoke.sh` 不只检查 discovery/JWKS/authorize 路由，也会读取公开 Casdoor application 元数据并断言 `admin/stuhelper-web` 仍为 `organization=stuhelper`、`enableSignUp=true`、`Password` 登录方式为 `All`、`Face ID` 为 `None`，且注册项包含必填的 `Password` 与 `Confirm password`。如果这里失败，先运行仓库内 `bootstrap-platform.sh prod` 修复 Casdoor 配置漂移，不要在 Casdoor 控制台手工改完就结束。

`admission-public-smoke.sh` 不只检查 `join.stuhelper.com/verify/<code>`，还会确认专用 `/student-verification/manual-camera/<token>` 页面允许 camera、普通 verify 页面禁用 camera，并要求旧 `/admission/freshman/camera/*` 与 `/api/v1/admission/freshman/camera-handoffs/*` 永久返回 404。脚本也会确认 `join.stuhelper.com/` 与 `join.stuhelper.com/developers/apps` 返回 404，避免 join 域串到主站首页或开发者入口；并从 join 域向 `/api/v1/metrics/vitals`、`/api/v1/metrics/frontend-errors` 发送同源 beacon，要求返回 204。

`public-web-auth-browser-smoke.mjs` 会用真实浏览器确认主站登录按钮进入 `sso.stuhelper.com/login/oauth/authorize` 后仍有账号密码登录和 `/signup/oauth/authorize` 注册入口，确认主站“注册账号”进入 `sso.stuhelper.com/signup/oauth/authorize` 的账号密码注册表单，并确认 `join.stuhelper.com/` 与 `join.stuhelper.com/developers/apps` 不渲染主站内容、`join.stuhelper.com/verify/<code>` 可加载、手机拍照页允许 camera。这样 Casdoor 配置漂移成“只剩 Face ID”、注册按钮走错授权路径、join 域串站或 camera permission 漂移时，公网浏览器 smoke 会直接失败。生产模式还会拒绝 `stuhelper.com`、`join.stuhelper.com` 或 `sso.stuhelper.com` 解析到 loopback；如果运维机 `/etc/hosts` 或浏览器代理把生产域名指向本地开发环境，先修正解析再生成 evidence。

`admission-mvp-production-evidence.sh` 是历史 admission MVP 的聚合证据入口；其中 SSO、公网入口、Web
浏览器和真实 QQ 回写检查仍可复用，但旧 `EXTERNAL_STUDENT_SOURCE_ENABLED`、
`external-student-source-smoke.sh` 与 `academic.buaa_students` 检查不是新学生认证域的上线证据。当前发布必须
额外证明：北航学校验证 profile 和所需方法均已审核启用；Campus Connector 节点/operation 已登记且健康；
存在通过质量门禁并由 `academic.student_roster_active` 原子指向的未过硬失效阈值完整快照；手动同步任务可
持久化、领取、回传并激活新版本。缺少真实 Oracle/LDAP/TCP/DNS 接线时应明确阻断相应认证方法，不能以
旧表或直连 smoke 代替。Koishi 节点仍用 `ADMISSION_MVP_PRODUCTION_EVIDENCE_MODE=koishi` 执行聚合入口；
最终真实 QQ 验收仍必须在主站运行 `make prod-admission-mvp-final-evidence`，并在 Koishi 节点运行
`make prod-admission-mvp-final-koishi-evidence`，且不得把 skipped 结果视为完成。

如果主站生产机没有 Node/Playwright，不要跳过公网浏览器 smoke。应先在有 Playwright 的运维机或 CI 上运行 `PUBLIC_WEB_AUTH_BROWSER_SMOKE_EVIDENCE_FILE=infra/generated/public-web-auth-browser-smoke-evidence-current.json ./infra/ops/public-web-auth-browser-smoke.mjs`，把生成的脱敏 evidence 复制到主站源码目录的同一路径；聚合入口会默认读取该文件。需要使用其他文件名时，再显式设置 `ADMISSION_MVP_PRODUCTION_BROWSER_SMOKE_EVIDENCE_FILE=infra/generated/<evidence>.json`。聚合入口会校验该 evidence 新鲜、目标域名正确、浏览器检查全部通过，并确认 `/identity` 直接入口、join 防串站 404、camera permission 与 fake media capture 成功。预采集 evidence 的机器不能通过 hosts 把生产域名解析到 `127.0.0.1`；这种情况应失败，而不是把本地 SPA/API 当成生产结果。

`admission-mvp-final-evidence-verify.sh` 只校验已采集的脱敏 evidence 文件，不访问生产，也不输出 secret。最终验收时，主站节点生成 `infra/generated/admission-mvp-final-evidence.json` 和 `infra/generated/admission-join-e2e-evidence.json`，Koishi 节点生成 `infra/generated/admission-mvp-final-koishi-evidence.json`；把三份文件放在同一份仓库工作目录后运行 `make prod-admission-mvp-final-verify`。该校验要求三份 evidence 都在默认 180 分钟新鲜度窗口内，主站和 Koishi 聚合 evidence 没有 failed/skipped，主站 evidence 必须包含真实 QQ `bot-released`，Koishi evidence 必须包含 Koishi admission production evidence 且不能夹带真实 QQ E2E placeholder，join E2E 子证据必须显示 token 已消费、QQ 已绑定、存在 active student verification credential、后端记录 bot release 和 cancelled marker，并包含通过的 `release requires active student verification credential` 检查。

`admission-reviewer-readiness.sh` 是只读检查，用 bot 的 `/api/v1/bot/admission/freshman/applications/{id}/view` 接口验证 QQ 管理员是否同时满足：所在群属于该申请的 `management_guild_ids`、管理员 QQ 已在 StuHelper 绑定、绑定用户拥有 `admission:freshman:review` 能力。它不调用 `/review`，不会批准或驳回申请。最终 E2E 卡在 `material_submitted` 时，应先用这条检查确认至少一个管理员可审核，而不是让用户反复试。可通过 `make prod-admission-reviewer-readiness` 调用，必填 `ADMISSION_REVIEWER_READINESS_APPLICATION_ID` 和 `ADMISSION_REVIEWER_READINESS_OPERATOR_QQ_IDS`，输出脱敏 evidence 到 `infra/generated/admission-reviewer-readiness.json`。

真实 QQ 小号入群 E2E 证据：

```bash
# 如果生产 DB 的 group_admission_sessions 为 0，说明还没有真实 QQ 入群事件触发 admission；
# 先让一个不在目标群内的小号实际申请/进入 178037297，再采集证据。

# 等待真实入群事件并自动落盘脱敏 evidence
ADMISSION_E2E_QQ_ID=<small-account-qq> \
ADMISSION_E2E_GUILD_ID=178037297 \
ADMISSION_E2E_EXPECTED_STAGE=join-created \
./infra/ops/admission-join-e2e-wait.sh

# 入群事件触发后，先验证 Koishi/后端已经创建 canonical admission session
ADMISSION_E2E_QQ_ID=<small-account-qq> \
ADMISSION_E2E_GUILD_ID=178037297 \
ADMISSION_E2E_EXPECTED_STAGE=join-created \
./infra/ops/admission-join-e2e-evidence.sh

# 等待用户完成 join.stuhelper.com 流程并自动落盘脱敏 evidence
ADMISSION_E2E_QQ_ID=<small-account-qq> \
ADMISSION_E2E_GUILD_ID=178037297 \
ADMISSION_E2E_EXPECTED_STAGE=flow-completed \
./infra/ops/admission-join-e2e-wait.sh

# 用户完成 join.stuhelper.com 登录、QQ 绑定和学生认证/新生材料流程后，验证业务流程完成
ADMISSION_E2E_QQ_ID=<small-account-qq> \
ADMISSION_E2E_GUILD_ID=178037297 \
ADMISSION_E2E_EXPECTED_STAGE=flow-completed \
./infra/ops/admission-join-e2e-evidence.sh

# 若 flow-completed 来自新生材料提交，先只读确认至少一个管理员 QQ 有审核权限
ADMISSION_REVIEWER_READINESS_APPLICATION_ID=<freshman-application-id> \
ADMISSION_REVIEWER_READINESS_OPERATOR_QQ_IDS=<operator-qq-ids> \
ADMISSION_REVIEWER_READINESS_GUILD_ID=178037297 \
make prod-admission-reviewer-readiness

# 等待 Koishi 执行解除禁言并把 release 成功回写后端
ADMISSION_E2E_QQ_ID=<small-account-qq> \
ADMISSION_E2E_GUILD_ID=178037297 \
ADMISSION_E2E_EXPECTED_STAGE=bot-released \
ADMISSION_E2E_WAIT_TIMEOUT_SECONDS=900 \
./infra/ops/admission-join-e2e-wait.sh
```

Koishi 生产证据脚本会检查配置、容器环境、bot API 探针和 admission 相关日志，只输出脱敏结果，不输出 token：

```bash
KOISHI_COMPOSE_DIR=/www/server/panel/data/compose/koishi-napcat \
KOISHI_ADMISSION_BOT_SELF_ID=<botSelfID> \
./infra/ops/koishi-admission-production-evidence.sh
```

预期：HTTP status 为 200。Koishi 日志无 `admission 401` / `unauthorized`。Koishi 日志无 `duplicate command names: 举报`。Koishi 日志无 `pending-forward` 每分钟 500 循环。`stuhelper-group-guard:admission` 已加载，目标群数量为 1。evidence 文件写入 `infra/generated/koishi-admission-production-evidence.json`，其中只记录 token 是否存在、bot API 状态码和 body 大小，不包含真实 token。默认日志窗口是最近 2 小时；如需审计更早的历史错误，可显式设置 `KOISHI_ADMISSION_LOG_SINCE=24h`。长期运行的 Koishi 容器可能已经没有启动加载日志，脚本默认不要求加载日志必须出现在当前窗口；刚重启后可设置 `KOISHI_ADMISSION_REQUIRE_LOAD_LOG=true` 强制检查。

告警路由还必须做一次独立的只读配置核对：Koishi Compose 的 `stuhelper-group-guard` 使用固定
`POST /stuhelper/internal/alertmanager` 路由，并通过受控内网/overlay 或精确反代让 Alertmanager 可达；
Koishi Console `5140` 不得作为 Alertmanager URL。Koishi 环境中的
`STUHELPER_ALERTMANAGER_WEBHOOK_ENABLED=true`、`ALERTMANAGER_WEBHOOK_TOKEN` 和可选
`STUHELPER_ALERTMANAGER_BOT_SELF_ID` 应与后端生成的 Alertmanager token/实际 bot 一致；真实 token
只能在两台主机的 secrets/env 文件中核对存在性，不能写入本 runbook 或 evidence。

随后用 Alertmanager UI/API 触发一组受控测试告警，验证 `firing` 和 `resolved` 各在唯一管理群收到一次，
并记录通知成功与失败计数。临时把 Koishi 后端地址、管理群配置或 QQ 发送置为不可用只能在获授权的
演练环境执行；预期 HTTP 503、Alertmanager failure counter 增加且恢复后自动重试，禁止用真实用户或生产
业务数据制造负向测试。

## 上线完成定义

同时满足以下条件才算完成：

- 本地仓库包含所有代码修复、配置模板、幂等脚本和 runbook；不存在只在生产手工修改的最终状态。
- `sso.stuhelper.com` 是唯一公开登录认证系统和 OIDC issuer。
- `join.stuhelper.com/verify/<code>` 是唯一公开加群验证入口；`join.stuhelper.com/verify`、`join.stuhelper.com/`、join 域主站业务页面路径和主站 `/verify*` 返回 404。
- 主站 `/health/live`、`/health/ready` 通过。
- Admission public smoke、Web auth browser smoke、DB readiness 通过。
- Koishi admission 插件在 OneBot/NapCat runtime 下使用 `qq` subject platform 调后端 admission API。
- 生产日志没有 admission 401、duplicate command、pending-forward 500 循环报错。
- 真实 QQ 小号入群 E2E 的 `admission-join-e2e-evidence.sh` 在 `flow-completed` 阶段通过，且 `admission-join-e2e-wait.sh` 在 `bot-released` 阶段通过，证明 Koishi 已执行并上报解除禁言。
- 主站 `make prod-admission-mvp-final-evidence`、Koishi 节点 `make prod-admission-mvp-final-koishi-evidence` 均通过，并且主站聚合 evidence、join E2E 子证据、Koishi evidence 三份文件经 `make prod-admission-mvp-final-verify` 校验通过。
- ChatLuna API key 等非 admission 报错不混入本次 admission 修复范围。
