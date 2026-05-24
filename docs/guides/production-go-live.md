---
type: guide
audience: ops
status: current
authoritative-source: docker-compose.prod.yml + infra/ops/*.sh + infra/nginx/baota-stuhelper.conf + infra/nginx/baota-casdoor-sso.conf
last-verified: 2026-05-24
---

# 生产上线缺漏清单与执行指导

本文回答一个问题：现在要把 `stuhelper.com` 主站先落地到生产，哪些还缺、哪些仓库已经补齐、每一步怎么做。

当前生产入口约定：

- 公网 `80/443` 只由宝塔 Nginx 监听。
- Docker Compose 只把业务服务绑定到宿主机回环地址：
  - backend：`127.0.0.1:18080`
  - web：`127.0.0.1:18000`
  - admin：`127.0.0.1:18001`
- `stuhelper.com` 承载主站、后台和 API；`id.stuhelper.com` 承载 StuHelper Identity/OIDC；`sso.stuhelper.com` 是外部 Casdoor SSO。
- Traefik 保留给开发或可选内部网关，不作为当前生产公网入口。

## 缺漏清单

| 项目 | 当前状态 | 上线前动作 | 是否阻断 |
|------|----------|------------|----------|
| 域名 DNS | 仓库不能代配 | `stuhelper.com`、`www.stuhelper.com`、`id.stuhelper.com` A/AAAA 记录指向生产机；`sso.stuhelper.com` 指向 SSO 机器 | 是 |
| 宝塔 Nginx 反代 | 仓库已提供 `infra/nginx/baota-stuhelper.conf`、`infra/nginx/baota-casdoor-sso.conf` 和 `infra/ops/nginx-public-ingress-preflight.sh` | 主站机器合并主站/id 配置并跑 `NGINX_PUBLIC_INGRESS_PROFILE=stuhelper` 审计；SSO 机器合并 Casdoor SSO 配置并跑 `NGINX_PUBLIC_INGRESS_PROFILE=sso` 审计；证书生效后 reload Nginx | 是 |
| Docker 生产端口 | 仓库已在 `docker-compose.prod.yml` 绑定 `127.0.0.1:18080/18000/18001` | 保持默认端口，确认防火墙没有把这些端口开放公网 | 是 |
| 远端 secret backend | 脚本要求生产使用非 file secret backend | 配置 `.deploy/remote.env` 中的 `SECRET_BACKEND=vault-kv-v2`、`VAULT_ADDR`、`VAULT_TOKEN_FILE`、`*_SECRET_REF` | 是 |
| 生产环境变量 | `.env.prod.example` 已按 `stuhelper.com` 预设主站 URL | 替换所有 `REPLACE_WITH_*` 和镜像占位符；不得提交 `.env.prod.*` | 是 |
| 公网身份入口 smoke | 发布脚本会执行 `infra/ops/identity-public-smoke.sh` 并写入 `infra/generated/identity-public-smoke-evidence.json` | 验证 `stuhelper.com` health、`id.stuhelper.com` OIDC discovery、OAuth authorization server metadata、JWKS、OAuth/UserInfo GET/POST 路由、`authorization_code` / `refresh_token` / `client_credentials` grant metadata、`response_modes_supported=query`、token / introspect / revoke endpoint 的 `client_secret_basic` / `client_secret_post` auth metadata、authorize 未登录跳转、`prompt=login&max_age=0` 重新认证跳转、token / introspect / revoke 路由级错误、GET/POST logout、POST logout URL query / JSON body 拒绝、UserInfo URL query / body token 来源拒绝、token/UserInfo/introspection/revoke 响应 `Cache-Control: no-store` 与 `Pragma: no-cache`、401 `WWW-Authenticate` challenge、`sso.stuhelper.com` discovery；配置专用 `IDENTITY_PUBLIC_SMOKE_CLIENT_ID` 后还会验证 `prompt=none` 的 `login_required` + `iss` 错误回调；同时配置 client secret 时会实测 `client_credentials` token 签发、携带 `token_type_hint=access_token` 的 introspection / revoke、混用 Basic 与 body client credential 时返回 `invalid_client`、UserInfo 拒绝、Open Platform resource access API 使用 Bearer app-only token 对未授权随机资源返回 `fga_denied` 或对预授权 smoke 资源返回 `allowed`、revoke 后 inactive | 是 |
| 公网身份入口诊断 | 仓库提供 `infra/ops/public-identity-ingress-diagnostic.sh` | 当 smoke 或 preflight 失败时，生成脱敏 `infra/generated/public-identity-ingress-diagnostic.json`，区分 DNS、SNI TLS、`id.stuhelper.com` OIDC discovery / OAuth authorization server metadata / JWKS `.well-known` 反代、`sso.stuhelper.com` Casdoor discovery/JWKS 被 SPA/404 覆盖等问题 | 否 |
| 不可变镜像 | 脚本会拒绝 `latest` / 浮动 tag | 准备 `BACKEND_IMAGE_REF`、`FRONTEND_IMAGE_REF`、`ADMIN_IMAGE_REF`，使用明确 tag 或 digest | 是 |
| Casdoor SSO | 主站 Compose 不启动本地 Casdoor | 确认 `https://sso.stuhelper.com` 可达，准备 bootstrap 管理应用凭据和 `stuhelper-identity` 一方应用 client secret | 是 |
| SMS | 生产强制 `SMS_ENABLED=true` | 配置短信厂商 `SMS_SECRET_ID`、`SMS_SECRET_KEY`、`SMS_APP_ID`、签名、模板 | 是 |
| Open Platform runtime token 探针 | backend 镜像内置 `/app/casdoor-runtime-token-probe-runner.mjs`，发布 evidence 脚本会执行 Casdoor smoke | 配置低权限 Casdoor 探针账号，保持 `OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_REQUIRED=true`，审批第三方 app 前自动实测 authorization-code token claims | 是 |
| 对象存储 | 后端生产要求 `OBJECT_STORAGE_USE_SSL=true` | 配置 HTTPS S3 兼容 endpoint、bucket、access key；本仓库未把内置 MinIO 暴露成生产 HTTPS 对象存储入口 | 是 |
| 备份对象存储 | 发布脚本会拒绝备份对象存储占位符，并在迁移前同步 / 取回验证 pre-deploy backup | 配置 `BACKUP_OBJECT_STORAGE_*`，并确认 `sync-postgres-backups.sh` 能写入 | 是 |
| PostgreSQL 备份 timer | bootstrap 脚本可安装 | 启用 dump、basebackup、backup sync timer，并完成一次恢复演练 | 是 |
| 观测与告警 | Compose 已包含 LGTM、Alertmanager、cAdvisor | 配置真实 `ALERTMANAGER_WEBHOOK_URL`，不要用本地 sink；确认 Grafana dashboard 有数据 | 是 |
| OpenFGA | 发布脚本会执行 migrate + bootstrap | 不手填模型；让 `bootstrap-platform.sh prod` 写入生成配置 | 是 |
| OpenFGA 资源授权 smoke | 发布 evidence 脚本会执行 `infra/ops/openfga-resource-access-smoke.sh` | 验证 Open Platform app -> resource_item tuple 的 grant / check / list / list-after-revoke / revoke 在真实 OpenFGA store/model 中生效 | 是 |
| Koishi/NapCat | 不纳入主站 Compose | 如机器人也要一起上线，按 `docs/guides/koishi-development.md` 单独部署并配置 service token | 视上线范围 |
| 真实教务连接器 / 第三方网盘驱动 | 当前是扩展骨架，不是主站基础链路 | 若本次主站只交付认证、课程评课、用户系统、后台和机器人入口，可暂缓 | 否 |

## 1. 准备生产机器

在部署机上准备仓库目录，例如：

```bash
cd /opt
git clone <repo-url> StuHelper
cd StuHelper
```

首次初始化 Ubuntu 生产机：

```bash
sudo bash infra/ops/bootstrap-ubuntu2404.sh
```

完成后确认：

```bash
docker version
docker compose version
ss -lntp | grep -E ':(80|443|18080|18000|18001|5432|6379)\b' || true
```

要求：

- 公网安全组只开放 `80`、`443`，SSH 只允许可信 IP。
- `18080`、`18000`、`18001`、`5432`、`6379`、`9000`、`9001` 只能绑定或访问本机/内网。
- 承载 Docker volume 的磁盘启用云盘 KMS、主机 LUKS 或等价静态加密。

## 2. 配置 DNS 和证书

DNS：

```text
stuhelper.com      A/AAAA -> 主站生产机
www.stuhelper.com  A/AAAA -> 主站生产机
id.stuhelper.com   A/AAAA -> 主站生产机
sso.stuhelper.com  A/AAAA -> Casdoor SSO 机器
```

宝塔中为 `stuhelper.com` 建站，并把 `www.stuhelper.com` 加入同一站点。证书可以用宝塔 Let's Encrypt，也可以上传已有证书。

证书生效后先确认：

```bash
curl -I https://stuhelper.com
curl -fsS https://id.stuhelper.com/.well-known/openid-configuration | head
curl -fsS https://sso.stuhelper.com/.well-known/openid-configuration | head
```

此时主站应用还没启动，`stuhelper.com` 可以暂时不是 200；但 TLS 握手和证书链必须正常。

## 3. 应用宝塔 Nginx 反代

把 `infra/nginx/baota-stuhelper.conf` 的三个 server block 合并到宝塔站点配置。反代目标必须保持：

```text
/api/*      -> http://127.0.0.1:18080
/health/*   -> http://127.0.0.1:18080
/metrics    -> http://127.0.0.1:18080
/docs/*     -> http://127.0.0.1:18080
/admin/*    -> http://127.0.0.1:18001
/           -> http://127.0.0.1:18000

id.stuhelper.com /.well-known/* -> http://127.0.0.1:18080
id.stuhelper.com /oauth2/*      -> http://127.0.0.1:18080
id.stuhelper.com /oidc/*        -> http://127.0.0.1:18080
id.stuhelper.com /api/*         -> http://127.0.0.1:18080
id.stuhelper.com /login /consent /complete-profile /assets/* -> http://127.0.0.1:18000
id.stuhelper.com /              -> 302 https://stuhelper.com/developers/apps
id.stuhelper.com 其他浏览器路径 -> 302 https://stuhelper.com$request_uri
```

宝塔面板保存后执行 Nginx 配置测试和 reload。命令路径随宝塔安装方式可能不同；至少要在面板里看到 Nginx 测试通过。

在外部 Casdoor SSO 机器上，合并 `infra/nginx/baota-casdoor-sso.conf` 或等价规则。关键要求：

```text
sso.stuhelper.com /.well-known/* -> http://127.0.0.1:8087
sso.stuhelper.com /api/*         -> http://127.0.0.1:8087
sso.stuhelper.com /              -> http://127.0.0.1:8087
```

`/.well-known/openid-configuration` 不能落到宝塔静态站点根目录；否则会返回 Casdoor SPA HTML / 404，`remote-preflight.sh` 和 `identity-public-smoke.sh` 都会阻断发布。

当前生产 SSO 现场端口是 `127.0.0.1:8087`，仓库模板也按该端口给出 upstream；如果外部 SSO 机器实际监听其他端口，需要在合并模板时同步替换三处 `proxy_pass`，并用 `NGINX_PUBLIC_INGRESS_CASDOOR_UPSTREAM=http://127.0.0.1:<port>` 执行 SSO 侧 Nginx preflight。

保存配置并 reload 前，先在对应机器审计实际 Nginx 配置。默认读取 `nginx -T` 输出；如果宝塔机器只能导出单个配置文件，可用 `NGINX_PUBLIC_INGRESS_CONFIG_FILE=/path/to/nginx.conf` 指定离线文件。

主站生产机：

```bash
NGINX_PUBLIC_INGRESS_PROFILE=stuhelper ./infra/ops/nginx-public-ingress-preflight.sh
```

外部 Casdoor SSO 机器：

```bash
NGINX_PUBLIC_INGRESS_PROFILE=sso ./infra/ops/nginx-public-ingress-preflight.sh
```

应用未启动前，`127.0.0.1:18080/18000/18001` 连接失败是正常的；Nginx 本身不能报配置语法错误。

## 4. 准备远端部署控制面

在生产机仓库目录执行：

```bash
./infra/ops/init-remote-deploy-config.sh
chmod 600 .deploy/remote.env
```

编辑 `.deploy/remote.env`，至少填实：

```bash
REGISTRY=<registry-host>
REGISTRY_USERNAME_SECRET_REF=secret/stuhelper/prod/registry-username
REGISTRY_PASSWORD_SECRET_REF=secret/stuhelper/prod/registry-password

ENV_FILE=/opt/StuHelper/.env.prod.shared
SECRETS_ENV_FILE=/opt/StuHelper/.env.prod.secrets
GENERATED_ENV_FILE=/opt/StuHelper/.env.prod.generated
GENERATED_SECRET_ENV_FILE=/opt/StuHelper/.env.prod.generated.secrets

SECRET_BACKEND=vault-kv-v2
SHARED_ENV_SECRET_REF=secret/stuhelper/prod/shared-env
SECRETS_ENV_SECRET_REF=secret/stuhelper/prod/secrets-env
GENERATED_ENV_SECRET_REF=secret/stuhelper/prod/generated-secrets-env
VAULT_ADDR=https://<vault-host>
VAULT_TOKEN_FILE=/opt/StuHelper/.secrets/vault/token
VAULT_KV_MOUNT=secret
```

生产发布脚本会拒绝 `SECRET_BACKEND=none`、空 secret backend、以及 `SECRET_BACKEND=file`。这是硬门禁：运行时派生 secret 不能靠本地明文文件长期保存。

## 5. 生成并填写生产环境变量

先生成 skeleton：

```bash
ENV_FILE=.env.prod.shared \
SECRETS_ENV_FILE=.env.prod.secrets \
GENERATED_ENV_FILE=.env.prod.generated \
GENERATED_SECRET_ENV_FILE=.env.prod.generated.secrets \
./infra/ops/init-prod-env.sh
```

然后填写 `.env.prod.shared` 和 `.env.prod.secrets` 中的占位符。核心项：

```bash
CORS_ORIGINS=https://stuhelper.com,https://id.stuhelper.com
WEB_PUBLIC_URL=https://stuhelper.com
ADMIN_PUBLIC_URL=https://stuhelper.com/admin/
PUBLIC_INGRESS_CONFIG_PREFLIGHT_ENABLED=true
NGINX_PUBLIC_INGRESS_PROFILE=stuhelper
NGINX_PUBLIC_INGRESS_CONFIG_FILE=
PUBLIC_INGRESS_PREFLIGHT_ENABLED=true
PUBLIC_INGRESS_PREFLIGHT_TIMEOUT_SECONDS=10
PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_TIMEOUT=10
PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_FILE=infra/generated/public-identity-ingress-diagnostic.json
PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_STRICT=false
PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_PUBLIC_DNS_ENABLED=true
PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_USE_ENV_TARGETS=false
IDENTITY_PUBLIC_SMOKE_ENABLED=true
IDENTITY_PUBLIC_SMOKE_BOOTSTRAP_ENABLED=false
IDENTITY_PUBLIC_SMOKE_BOOTSTRAP_MODE=container
IDENTITY_PUBLIC_SMOKE_OWNER_USER_ID=
IDENTITY_PUBLIC_SMOKE_REVIEWER_USER_ID=
IDENTITY_PUBLIC_SMOKE_HOMEPAGE_URL=https://stuhelper.com
IDENTITY_PUBLIC_SMOKE_PRIVACY_POLICY_URL=https://stuhelper.com/privacy
IDENTITY_PUBLIC_SMOKE_CLIENT_ID=
IDENTITY_PUBLIC_SMOKE_REDIRECT_URI=
IDENTITY_PUBLIC_SMOKE_SCOPE=openid
IDENTITY_PUBLIC_SMOKE_CLIENT_SECRET=
IDENTITY_PUBLIC_SMOKE_CLIENT_CREDENTIALS_SCOPE=resource.read
IDENTITY_PUBLIC_SMOKE_RESOURCE_ACCESS_RESOURCE_TYPE=resource_item
IDENTITY_PUBLIC_SMOKE_RESOURCE_ACCESS_RESOURCE_ID=
IDENTITY_PUBLIC_SMOKE_RESOURCE_ACCESS_ACTION=
IDENTITY_PUBLIC_SMOKE_RESOURCE_ACCESS_EXPECT_ALLOWED=false
IDENTITY_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS=false
WEB_VITE_API_URL=/api
WEB_VITE_SSO_URL=https://sso.stuhelper.com
ADMIN_VITE_API_URL=/api/v1
ADMIN_VITE_BASE=/admin/
IDENTITY_ISSUER=https://id.stuhelper.com
IDENTITY_REFRESH_TOKEN_TTL=2592000
TOKEN_COOKIE_SECURE=true
TOKEN_COOKIE_DOMAIN=.stuhelper.com

CASDOOR_ISSUER=https://sso.stuhelper.com
CASDOOR_REDIRECT_URI=https://stuhelper.com/api/v1/auth/callback
CASDOOR_ADMIN_REDIRECT_URI=https://stuhelper.com/api/v1/auth/callback
CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_ID=casdoor-token-probe-smoke
CASDOOR_TOKEN_PROBE_SMOKE_CLIENT_SECRET=<generated-or-secret-backend-value>
CASDOOR_TOKEN_PROBE_SMOKE_APPLICATION=casdoor-token-probe-smoke
CASDOOR_TOKEN_PROBE_SMOKE_REDIRECT_URI=https://stuhelper.com/open-platform/token-probe/callback

OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_REQUIRED=true
OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND=/app/casdoor-runtime-token-probe-runner.mjs
OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_TIMEOUT_SECONDS=30
OPEN_PLATFORM_PRODUCTION_EVIDENCE_ALLOW_LOCAL_TARGETS=false
CASDOOR_TOKEN_PROBE_USERNAME=<low-privilege-casdoor-probe-user>
CASDOOR_TOKEN_PROBE_PASSWORD=<probe-user-password>
CASDOOR_TOKEN_PROBE_BROWSER_EXECUTABLE_PATH=/usr/bin/chromium-browser
CASDOOR_TOKEN_PROBE_BROWSER_NO_SANDBOX=true

OPENFGA_RESOURCE_SMOKE_MODE=container

BACKEND_EXTERNAL_PORT=18080
WEB_EXTERNAL_PORT=18000
ADMIN_EXTERNAL_PORT=18001
```

`id.stuhelper.com` 的 `/login` 页面会复用主站后端会话。`CASDOOR_REDIRECT_URI` 固定回到 `stuhelper.com/api/v1/auth/callback` 时，生产必须设置 `TOKEN_COOKIE_DOMAIN=.stuhelper.com`，否则回到 `id.stuhelper.com/oauth2/authorize` 时浏览器不会携带登录 cookie，第三方授权会进入重复登录。

必须替换的外部依赖：

- `CASDOOR_CLIENT_ID` / `CASDOOR_CLIENT_SECRET`
- `CASDOOR_ADMIN_CLIENT_SECRET`
- `CASDOOR_UNIAPP_CLIENT_SECRET` / `CASDOOR_UNIAPP_REDIRECT_URI`
- `CASDOOR_APP_PROVISIONING_*`
- `CASDOOR_INTROSPECTION_*`
- `CASDOOR_ROLE_SYNC_*`
- `CASDOOR_USER_LOOKUP_*`
- `CASDOOR_TOKEN_PROBE_SMOKE_*`
- `OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND`
- `CASDOOR_TOKEN_PROBE_USERNAME` / `CASDOOR_TOKEN_PROBE_PASSWORD`
- `IDENTITY_SIGNING_PRIVATE_KEY_PEM`
- `SMS_*`
- `OBJECT_STORAGE_*`
- `BACKUP_OBJECT_STORAGE_*`
- `GRAFANA_ROOT_URL`
- `ALERTMANAGER_WEBHOOK_URL`
- `BACKEND_IMAGE_REF` / `FRONTEND_IMAGE_REF` / `ADMIN_IMAGE_REF`

不要手动填：

- `OPENFGA_STORE_ID`
- `OPENFGA_MODEL_ID`

它们由 `bootstrap-platform.sh prod` 在发布时写入生成配置。

Open Platform runtime token 探针要求：

- 探针账号必须是专用低权限 Casdoor 用户，不授予后台管理员、平台运维或业务管理角色。
- 探针账号需要能完成普通网页登录；如果生产 Casdoor 对所有用户强制 MFA，要为该账号配置可自动化的专用测试认证策略，或改用等价的自定义 `OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND`。
- `bootstrap-platform.sh prod` 会创建 / 更新专用 `CASDOOR_TOKEN_PROBE_SMOKE_APPLICATION`，显式设置空 `TokenFields`；`prod-deploy.sh` 在启动 backend/frontend/admin 前会运行 `infra/ops/open-platform-production-evidence.sh`，用该 app 实跑 authorization-code token 最小化验证，并同步验证 OpenFGA resource smoke。
- 内置 runner 只请求 `scope=openid`，通过 Playwright 完成 authorization-code + PKCE 登录，解码 `id_token` 和 JWT `access_token` 的 claim key，发现手机号、学生认证、学校、身份类型等业务 claim 时审批失败。
- backend 镜像已经包含 Node、Playwright Core、Chromium 和 `/app/casdoor-runtime-token-probe-runner.mjs`；生产样例默认 `CASDOOR_TOKEN_PROBE_BROWSER_EXECUTABLE_PATH=/usr/bin/chromium-browser`。
- 如 Casdoor 登录页定制导致默认 selector 失效，可设置 `CASDOOR_TOKEN_PROBE_USERNAME_SELECTOR`、`CASDOOR_TOKEN_PROBE_PASSWORD_SELECTOR`、`CASDOOR_TOKEN_PROBE_SUBMIT_SELECTOR`、`CASDOOR_TOKEN_PROBE_CONSENT_SELECTOR`。
- 生产 OpenFGA resource smoke 默认用 `OPENFGA_RESOURCE_SMOKE_MODE=container`，确保 `OPENFGA_API_URL=http://openfga:8080` 在 Docker backend 网络内解析；开发环境默认 `host`，使用 `http://localhost:8081`。

## 6. 写入 secret backend

如果使用 Vault KV v2，建议把 shared env、secrets env、registry 凭据分别写入独立 secret：

```bash
vault kv put secret/stuhelper/prod/shared-env value=@.env.prod.shared
vault kv put secret/stuhelper/prod/secrets-env value=@.env.prod.secrets
vault kv put secret/stuhelper/prod/generated-secrets-env value=''
vault kv put secret/stuhelper/prod/registry-username value='<registry-user>'
vault kv put secret/stuhelper/prod/registry-password value='<registry-password>'
```

运行时生成的 `generated-secrets-env` 初始可以为空，但 `GENERATED_ENV_SECRET_REF` 必须能被当前 token 读写。发布脚本会在 bootstrap 后写入真实派生配置。

生产机本地的 `.env.prod.generated.secrets` 只保留空占位文件，不作为真实 secret 来源。

## 7. 配置对象存储与备份

后端生产校验要求：

```bash
OBJECT_STORAGE_USE_SSL=true
```

因此生产对象存储必须是 HTTPS S3 兼容 endpoint，例如云厂商 OSS/COS/S3，或你自己暴露了 HTTPS 的 MinIO endpoint。

示例：

```bash
OBJECT_STORAGE_ENDPOINT=https://s3.example.com
OBJECT_STORAGE_BUCKET=stuhelper-identity
OBJECT_STORAGE_ACCESS_KEY_ID=<access-key>
OBJECT_STORAGE_SECRET_ACCESS_KEY=<secret-key>
OBJECT_STORAGE_USE_SSL=true
OBJECT_STORAGE_FORCE_PATH_STYLE=false

BACKUP_OBJECT_STORAGE_ENDPOINT=https://s3.example.com
BACKUP_OBJECT_STORAGE_BUCKET=stuhelper-postgres-backup
BACKUP_OBJECT_STORAGE_PREFIX=postgres
BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID=<backup-access-key>
BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY=<backup-secret-key>
BACKUP_OBJECT_STORAGE_TLS_INSECURE=false
```

注意：仓库内 Compose 的 `minio` 服务默认只绑定本机 HTTP 端口，不能直接满足生产 `OBJECT_STORAGE_USE_SSL=true` 的外部访问要求。除非你额外给 MinIO 配好 HTTPS 公网或内网入口，否则不要把 `OBJECT_STORAGE_ENDPOINT` 指向 `http://minio:9000`。

安装并启用备份 timer：

```bash
sudo ./infra/ops/install-backup-timers.sh
systemctl list-timers | grep stuhelper-postgres
```

上线前至少做一次恢复链路验证：

```bash
./infra/ops/fetch-postgres-backups.sh all
```

如果目标环境还没有历史备份，第一次发布前 `remote-preflight.sh` 会允许目录为空，但备份 timer、连接串和对象存储配置必须齐。

## 8. 准备不可变镜像

生产镜像不能使用 `latest`、`main`、`master`、`develop-latest`、`ci-*` 这类浮动 tag。

推荐使用 digest：

```bash
BACKEND_IMAGE_REF=registry.example.com/stuhelper/backend@sha256:<digest>
FRONTEND_IMAGE_REF=registry.example.com/stuhelper/frontend@sha256:<digest>
ADMIN_IMAGE_REF=registry.example.com/stuhelper/admin@sha256:<digest>
```

也可以使用明确发布 tag：

```bash
BACKEND_IMAGE_REF=registry.example.com/stuhelper/backend:2026-05-09-<sha>
FRONTEND_IMAGE_REF=registry.example.com/stuhelper/frontend:2026-05-09-<sha>
ADMIN_IMAGE_REF=registry.example.com/stuhelper/admin:2026-05-09-<sha>
```

## 9. 执行发布前预检

在生产机执行：

```bash
./infra/ops/remote-preflight.sh
```

预检会检查：

- Docker / Compose 可用
- 生产 PostgreSQL TLS 配置默认强制启用：`POSTGRES_ENABLE_SSL=on`、`POSTGRES_INTERNAL_SSL_MODE=verify-full`（最低必须为 `verify-ca`）、`DB_SSL_MODE=verify-full`，并且三个 PostgreSQL URL 都带 `sslrootcert`。如果目标机器复用已有宝塔 Postgres 且该服务未启用 TLS，必须显式设置 `EXTERNAL_POSTGRES_ENABLED=true`、`EXTERNAL_DATASTORE_NETWORK=baota_net`、`EXTERNAL_POSTGRES_ALLOW_PLAINTEXT=true`，发布前先在外部 Postgres 中为 StuHelper / OpenFGA 创建独立数据库和独立账号，并把旧 StuHelper 专用 Postgres 中的 `stuhelper` / `openfga` 数据迁移到外部 Postgres。Redis 不复用全局实例，仍由 StuHelper Compose 以独立实例运行。
- 本机宝塔 Nginx 主站/id 入口配置满足反代契约：`stuhelper.com`、`www.stuhelper.com`、`id.stuhelper.com` 均有 HTTPS server block，主站 `/.well-known/`、`/oauth2/`、`/oidc/`、`/api/`、`/health/`、`/admin/` 和 `/` 均代理到约定的回环端口；`id.stuhelper.com/` 302 到开放平台开发者应用页，授权页 `/login`、`/consent`、`/complete-profile` 及 `/assets/` 仍代理到 web 前端
- 公网身份入口公共 DNS / TLS / OIDC 可用：`stuhelper.com`、`id.stuhelper.com`、`sso.stuhelper.com` 在公共 DNS-over-HTTPS 中有公网 A/AAAA，`stuhelper.com`、`id.stuhelper.com` TLS 可达，`sso.stuhelper.com/.well-known/openid-configuration` 返回有效 Casdoor OIDC discovery
- PostgreSQL 备份工具可用
- secret backend 配置可用
- 备份 timer 已安装且启用
- `BACKUP_DATABASE_URL` / `REPLICATION_DATABASE_URL` 可连接
- 恢复工具 `pg_restore` 可用
- 备份目录可写

预检失败时不要继续发布。先修具体错误，再重跑。

## 10. 发布

远端生产发布入口：

```bash
./infra/ops/remote-prod-deploy.sh
```

如果你是在生产机上手工执行且已经本地登录 registry，也可以：

```bash
set -a
source .deploy/remote.env
set +a
make prod-deploy
```

不要在没有加载 `.deploy/remote.env` 的情况下直接执行 `make prod-deploy`，否则它会按本地生产演练路径读取 `.env.prod.secrets.local`。

发布脚本会按顺序执行：

1. 读取远端部署控制面和 secret backend
2. 校验生产必填配置、占位符、不可变镜像、PostgreSQL TLS/Redis TLS（或显式外部明文 PostgreSQL 例外）/SMS/OTEL/Open Platform runtime token 探针门禁，并在拉镜像前审计本机宝塔 Nginx 主站/id 配置、验证 `stuhelper.com` / `id.stuhelper.com` / `sso.stuhelper.com` 公共 DNS、`stuhelper.com` / `id.stuhelper.com` TLS 可达与 `sso.stuhelper.com` OIDC discovery 元数据
3. 渲染 PostgreSQL TLS、Redis ACL、观测配置；启用外部明文 PostgreSQL 例外时只跳过 StuHelper 内部 PostgreSQL TLS 渲染，Redis ACL 仍始终渲染
4. 拉取 backend / frontend / admin 镜像
5. 启动 StuHelper 独立 Redis、MinIO、观测栈，并在未启用外部 PostgreSQL 时启动 StuHelper 内部 PostgreSQL
6. 创建发布前逻辑备份
7. 同步 pre-deploy 逻辑备份到对象存储，并执行 `postgres-backup-evidence.sh` 证明本地备份和取回备份 sha256 均匹配
8. 执行数据库迁移和 OpenFGA migrate
9. bootstrap Casdoor 应用、角色、provider 与 OpenFGA 配置
10. 执行 Open Platform 生产准入 evidence smokes，验证 Casdoor token `businessClaims=[]` 和 OpenFGA app -> resource tuple 的 grant / check / list / list-after-revoke / revoke
11. 启动 app、frontend、admin
12. 执行公网身份入口 smoke、业务 smoke check 和 strict observability smoke check
13. 写入 `.deploy/releases.log`

## 11. 验证

容器和本机端口：

```bash
docker compose -f docker-compose.yml -f docker-compose.observability.yml -f docker-compose.prod.yml --profile prod ps
curl -fsS http://127.0.0.1:18080/health/live
curl -fsS http://127.0.0.1:18080/health/ready
curl -fsSI http://127.0.0.1:18000/
curl -fsSI http://127.0.0.1:18001/admin/
```

宝塔公网入口：

```bash
curl -fsSI https://stuhelper.com/
curl -fsS https://stuhelper.com/health/ready
curl -fsSI https://stuhelper.com/admin/
curl -fsS https://stuhelper.com/api/v1/course/departments
curl -fsS https://id.stuhelper.com/.well-known/openid-configuration | head
curl -fsS https://sso.stuhelper.com/.well-known/openid-configuration | head
```

如果上述公网入口或 `identity-public-smoke.sh` 失败，先生成脱敏诊断 evidence：

```bash
./infra/ops/public-identity-ingress-diagnostic.sh
```

诊断脚本默认固定检查 `https://stuhelper.com`、`https://id.stuhelper.com` 和 `https://sso.stuhelper.com`，即使本地 `.env` 是开发环境 localhost 也不会覆盖公网目标。需要临时诊断其他目标时，应在当前命令显式传入 `WEB_PUBLIC_URL` / `IDENTITY_ISSUER` / `CASDOOR_ISSUER`；只有确实要使用 `ENV_FILE` 里的目标时，才设置 `PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_USE_ENV_TARGETS=true`。诊断输出会同时保留本机 resolver 和 `dns.google` 公共 DNS-over-HTTPS 视角，标记 `dns_resolution_failed`、`dns_non_public_address`、`public_dns_nxdomain`、`public_dns_non_public_address`、`tls_handshake_failed`、`identity_well_known_not_proxied`、`identity_oauth_as_metadata_not_proxied`、`casdoor_well_known_served_by_spa` 等分类，并把 `Set-Cookie` 等响应头值替换成 `<redacted>`。如果生产机无法访问公共 DoH，可设置 `PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_PUBLIC_DNS_ENABLED=false` 只保留本机 resolver 视角。

Open Platform token 探针链路：

```bash
docker compose -f docker-compose.yml -f docker-compose.observability.yml -f docker-compose.prod.yml --profile prod exec app \
  test -x /app/casdoor-runtime-token-probe-runner.mjs

./infra/ops/casdoor-runtime-token-probe-smoke.sh
```

发布脚本会在 `bootstrap-platform.sh prod` 之后通过 `open-platform-production-evidence.sh` 自动运行同一个 smoke；输出 JSON evidence 时 `businessClaims` 必须为空，且 `metadata.nonceVerified` 必须为 `true`，证明返回的 ID Token 绑定到本次授权请求。随后可在管理后台批准一个只请求基础 `openid` 的测试第三方应用，确认审批成功且 `/open-platform/token-probe-evidence` 中出现 `passed` evidence。不要在终端输出探针账号密码；runner 所需凭据应来自生产 secret backend 注入的环境变量。

完整 smoke：

```bash
API_BASE_URL=https://stuhelper.com \
WEB_BASE_URL=https://stuhelper.com \
ADMIN_BASE_URL=https://stuhelper.com \
CASDOOR_ISSUER=https://sso.stuhelper.com \
./infra/ops/smoke-check.sh

./infra/ops/open-platform-production-evidence.sh

./infra/ops/postgres-backup-evidence.sh

./infra/ops/identity-public-smoke.sh

./infra/ops/observability-smoke-check.sh
```

公网身份入口 smoke 成功或失败都会把脱敏 evidence 写入
`infra/generated/identity-public-smoke-evidence.json`，其中包含本次检查的 public URL、
issuer、各 endpoint、通过 / 失败计数，以及每个检查可用的 `httpStatus`、`curlError`
和 `bodySnippet` 诊断字段。发布留档时确认 `.passed == true` 且 `.summary.failed == 0`。
脚本默认拒绝 `localhost`、`127.0.0.1`、`::1` 和 `host.docker.internal` 目标，防止把开发环境结果误写成公网 smoke evidence；只有本地合同测试或明确的本地生产验证才允许设置 `IDENTITY_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS=true`。

如果已经在 Open Platform 中准备了一个 approved 的专用 Identity smoke client，可以在
生产环境中设置：

```bash
IDENTITY_PUBLIC_SMOKE_BOOTSTRAP_ENABLED=false
IDENTITY_PUBLIC_SMOKE_CLIENT_ID=<approved-open-platform-client-id>
IDENTITY_PUBLIC_SMOKE_REDIRECT_URI=<registered-exact-redirect-uri>
IDENTITY_PUBLIC_SMOKE_SCOPE=openid
IDENTITY_PUBLIC_SMOKE_CLIENT_SECRET=<approved-open-platform-client-secret>
IDENTITY_PUBLIC_SMOKE_CLIENT_CREDENTIALS_SCOPE=resource.read
IDENTITY_PUBLIC_SMOKE_RESOURCE_ACCESS_RESOURCE_TYPE=resource_item
IDENTITY_PUBLIC_SMOKE_RESOURCE_ACCESS_RESOURCE_ID=
IDENTITY_PUBLIC_SMOKE_RESOURCE_ACCESS_ACTION=
IDENTITY_PUBLIC_SMOKE_RESOURCE_ACCESS_EXPECT_ALLOWED=false
```

也可以让发布脚本在公网 smoke 前自动准备这个专用 client。此模式需要指定一个
已存在的普通 owner 用户和一个已存在的 reviewer/admin 用户 ID；脚本只会把生成的
client secret 写入 secret env 文件，日志和 evidence 不输出 secret：

```bash
IDENTITY_PUBLIC_SMOKE_BOOTSTRAP_ENABLED=true
IDENTITY_PUBLIC_SMOKE_BOOTSTRAP_MODE=container
IDENTITY_PUBLIC_SMOKE_OWNER_USER_ID=<existing-owner-user-id>
IDENTITY_PUBLIC_SMOKE_REVIEWER_USER_ID=<existing-reviewer-user-id>
IDENTITY_PUBLIC_SMOKE_CLIENT_ID=identity-public-smoke
IDENTITY_PUBLIC_SMOKE_REDIRECT_URI=https://stuhelper.com/open-platform/identity-public-smoke/callback
IDENTITY_PUBLIC_SMOKE_HOMEPAGE_URL=https://stuhelper.com
IDENTITY_PUBLIC_SMOKE_PRIVACY_POLICY_URL=https://stuhelper.com/privacy
IDENTITY_PUBLIC_SMOKE_CLIENT_CREDENTIALS_SCOPE=resource.read
```

默认 smoke 还会用无凭据请求验证 `/oauth2/token`、`/oauth2/introspect`、`/oauth2/revoke`
会返回 OAuth JSON 错误，验证无会话 GET/POST `/oauth2/logout` 返回 204，并验证 logout
POST URL query / JSON body 以及 UserInfo URL query / body token 来源会被拒绝，避免公网 Nginx
只放行 discovery 但漏转 token、撤销、登出、UserInfo 等方法路由。配置专用 client secret 后，smoke 还会对
token、introspection 和 revoke 路由发送同时包含 Basic 头与 body `client_id` / `client_secret`
的请求，并要求返回 `invalid_client`，证明公网链路没有绕过 Identity Server 的 client
authentication hardening。
token、UserInfo、introspection 和 revoke 相关检查还会要求 `Cache-Control: no-store` 与
`Pragma: no-cache`，避免反向代理或浏览器缓存保存 token、用户资料或 introspection 结果。
其中 `invalid_client` 401 必须带 Basic `WWW-Authenticate` challenge，UserInfo 的
`invalid_token` 401 必须带 Bearer `WWW-Authenticate` challenge，确保标准 OAuth/OIDC
客户端能按认证类型处理失败。

设置专用 client 后，`identity-public-smoke.sh` 会额外发送注册客户端的 `prompt=none` 授权请求，
并要求未登录浏览器态通过已登记 redirect URI 回调 `error=login_required`、原始
`state` 和 `iss=https://id.stuhelper.com`。这能同时证明 app registry、redirect URI
精确匹配、静默登录错误语义和 RFC 9207 错误回调 issuer 都在公网链路上可用。
如果同时设置 `IDENTITY_PUBLIC_SMOKE_CLIENT_SECRET`，脚本还会执行真实 `client_credentials`
grant，要求只返回 app-only access token、introspection 为 `active=true` 且
`grant_type=client_credentials`；introspection 和 revoke 请求会携带
`token_type_hint=access_token`，验证标准客户端 hint 不影响 token 查找和撤销语义。UserInfo
对该 token 返回 `invalid_token`，并用该 app-only token 调用
`POST /api/v1/open-platform/resources/access/check`。默认检查一个每次运行不同的
`resource_item`，预期返回 `allowed=false` / `reason=fga_denied`，证明公开 API 能完成 token
校验、scope 准入和 OpenFGA 决策；随后 revoke 后再次 introspection 必须为 `active=false`。
evidence 只记录 client ID、scope、资源类型 / ID、动作和 HTTP 状态，不记录 client secret 或
access token。

如果要把公网 smoke 升级为正向资源授权证据，先在管理后台给该 smoke app 授权一个稳定的
`resource_item` 或 `user_profile` 资源，再设置：

```bash
IDENTITY_PUBLIC_SMOKE_RESOURCE_ACCESS_RESOURCE_TYPE=resource_item
IDENTITY_PUBLIC_SMOKE_RESOURCE_ACCESS_RESOURCE_ID=<pre-granted-resource-id>
IDENTITY_PUBLIC_SMOKE_RESOURCE_ACCESS_ACTION=read
IDENTITY_PUBLIC_SMOKE_RESOURCE_ACCESS_EXPECT_ALLOWED=true
```

此时 evidence 必须包含 `Open Platform resource access API 允许已授权资源`，且
`.details.expectedReason == "allowed"`。不要把业务生产用户资料 ID 当作长期公开 smoke
资源；建议准备一个专用测试资源或专用测试用户投影。

Open Platform 生产准入证据也可单独用聚合脚本生成：

```bash
./infra/ops/open-platform-production-evidence.sh
```

脚本会顺序执行 `casdoor-runtime-token-probe-smoke.sh` 和 `openfga-resource-access-smoke.sh`，
并把脱敏后的 JSON evidence 写入 `infra/generated/open-platform-production-evidence.json`。
聚合脚本会先校验 `OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_REQUIRED=true` 且 runtime command 已配置并非模板占位符，证明审批路径的强制 token 探针门禁处于开启状态；随后校验子 smoke evidence 指向当前 `CASDOOR_ISSUER`、`OPENFGA_API_URL`、
`OPENFGA_STORE_ID` 和 `OPENFGA_MODEL_ID`，避免把其他环境的证据误归档到本次发布。失败诊断只输出
claim 名称和 OpenFGA 布尔断言等白名单字段，不输出 raw token、client secret 或 probe 密码。
脚本默认拒绝 `CASDOOR_ISSUER`、`CASDOOR_TOKEN_PROBE_SMOKE_REDIRECT_URI` 或 `OPENFGA_API_URL`
指向 `localhost`、`127.0.0.1`、`::1`、`host.docker.internal`，避免把开发环境 Casdoor/OpenFGA
结果写成生产准入 evidence；只有本地合同测试或明确的本地生产验证才允许设置
`OPEN_PLATFORM_PRODUCTION_EVIDENCE_ALLOW_LOCAL_TARGETS=true`。
完成证据必须同时满足：

- `.passed == true`
- `.configuration.runtimeTokenProbeRequired == true`
- `.casdoorRuntimeTokenProbe.businessClaims == []`
- `.casdoorRuntimeTokenProbe.metadata.nonceVerified == true`
- `.openfgaResourceAccessSmoke.listedReadGrant == true`
- `.openfgaResourceAccessSmoke.readAfterGrant == true`
- `.openfgaResourceAccessSmoke.writeAfterGrant == true`
- `.openfgaResourceAccessSmoke.listedReadAfterRevoke == false`
- `.openfgaResourceAccessSmoke.readAfterRevoke == false`
- `.openfgaResourceAccessSmoke.writeAfterRevoke == false`

Grafana 中至少确认：

- `StuHelper Overview` 有最近 5 分钟数据
- `StuHelper Application` 有 API 延迟、状态码、错误率
- `StuHelper Infrastructure` 有 node-exporter 与 cAdvisor 数据
- Alertmanager receiver 指向真实值班系统

生产发布会以 `OBS_SMOKE_STRICT=true` 执行 `observability-smoke-check.sh`，并写入
`infra/generated/observability-smoke-evidence.json`。该 evidence 会证明核心服务 ready、
Prometheus 已抓到 app/Grafana/Alertmanager/cAdvisor，黑盒探针已覆盖 `id.stuhelper.com`、
`sso.stuhelper.com` 和 OpenFGA TCP，且 Alertmanager webhook 不是本地 sink。

## 12. 回滚

应用层回滚：

```bash
export ROLLBACK_TAG=<previous-stable-tag>
make prod-rollback
```

如果不设置 `ROLLBACK_TAG`，脚本会尝试使用 `.deploy/releases.log` 中上一条成功发布。

如果本次包含破坏性数据库变更：

1. 先切回上一版镜像。
2. 用 `docs/guides/backup-and-restore.md` 的流程恢复数据库。
3. 重跑 smoke check。
4. 记录恢复耗时、备份文件、验证结果。

## 上线完成定义

同时满足下面条件，才算主站生产落地完成：

- `https://stuhelper.com/`、`/admin/`、`/api/v1/*`、`/health/ready` 可通过宝塔 Nginx 访问。
- `https://id.stuhelper.com/.well-known/openid-configuration`、`/oauth2/authorize`、`/oidc/userinfo` 路由到主站后端。
- `https://sso.stuhelper.com/.well-known/openid-configuration` 可访问，登录 URL 能生成。
- `make prod-deploy` 或 `remote-prod-deploy.sh` 完整通过。
- `identity-public-smoke.sh` 完整通过，并生成 `infra/generated/identity-public-smoke-evidence.json`，证明 `stuhelper.com`、`id.stuhelper.com`、`sso.stuhelper.com` 公网身份入口可达，Identity OIDC discovery、OAuth authorization server metadata、JWKS 以及 Casdoor discovery/JWKS 正确，普通 authorize、`prompt=login&max_age=0` 重新认证跳转、`response_modes_supported=query`、token / introspect / revoke 路由级行为、GET/POST logout、POST logout URL query / JSON body 拒绝、GET/POST UserInfo 缺 bearer、UserInfo URL query / body token 来源拒绝、敏感 OAuth 响应 no-store/no-cache 缓存控制和 401 `WWW-Authenticate` challenge 符合预期；如已配置专用 `IDENTITY_PUBLIC_SMOKE_CLIENT_ID`，还要证明 `prompt=none` 会按注册 redirect URI 回调 `login_required`、`state` 和 `iss`；如同时配置 `IDENTITY_PUBLIC_SMOKE_CLIENT_SECRET`，还要证明 `client_credentials` token 签发、携带 `token_type_hint=access_token` 的 introspection / revoke、UserInfo 拒绝、Open Platform resource access API 使用 Bearer app-only token 且对未授权随机资源返回 `fga_denied` 或对预授权 smoke 资源返回 `allowed`、revoke 后 inactive 均通过。
- `smoke-check.sh`、`open-platform-production-evidence.sh` 和 `observability-smoke-check.sh` 完整通过；其中 Open Platform evidence 已包含 `openfga-resource-access-smoke.sh`。
- `open-platform-production-evidence.sh` 生成脱敏 evidence，且 runtime token probe command 不是占位符、`.configuration.runtimeTokenProbeRequired=true`、`businessClaims=[]`、OpenFGA grant/list/list-after-revoke/revoke 断言通过。
- 备份 timer 已启用，`postgres-backup-evidence.sh` 生成脱敏 evidence，证明至少一个本地逻辑备份存在且能从对象存储取回并通过 sha256 校验。
- `observability-smoke-check.sh` 生成脱敏 evidence，证明 Prometheus / Grafana / Alertmanager / blackbox 探针有数据，且 Alertmanager 已接真实值班渠道。
- 回滚命令和上一版镜像可用。
