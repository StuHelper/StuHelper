---
type: guide
audience: ops
status: current
authoritative-source: docker-compose.prod.yml + infra/ops/*.sh + infra/nginx/baota-stuhelper.conf
last-verified: 2026-05-22
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
| 宝塔 Nginx 反代 | 仓库已提供 `infra/nginx/baota-stuhelper.conf` | 在宝塔站点配置中合并该文件，证书生效后 reload Nginx | 是 |
| Docker 生产端口 | 仓库已在 `docker-compose.prod.yml` 绑定 `127.0.0.1:18080/18000/18001` | 保持默认端口，确认防火墙没有把这些端口开放公网 | 是 |
| 远端 secret backend | 脚本要求生产使用非 file secret backend | 配置 `.deploy/remote.env` 中的 `SECRET_BACKEND=vault-kv-v2`、`VAULT_ADDR`、`VAULT_TOKEN_FILE`、`*_SECRET_REF` | 是 |
| 生产环境变量 | `.env.prod.example` 已按 `stuhelper.com` 预设主站 URL | 替换所有 `REPLACE_WITH_*` 和镜像占位符；不得提交 `.env.prod.*` | 是 |
| 不可变镜像 | 脚本会拒绝 `latest` / 浮动 tag | 准备 `BACKEND_IMAGE_REF`、`FRONTEND_IMAGE_REF`、`ADMIN_IMAGE_REF`，使用明确 tag 或 digest | 是 |
| Casdoor SSO | 主站 Compose 不启动本地 Casdoor | 确认 `https://sso.stuhelper.com` 可达，准备 bootstrap 管理应用凭据和 `stuhelper-identity` 一方应用 client secret | 是 |
| SMS | 生产强制 `SMS_ENABLED=true` | 配置短信厂商 `SMS_SECRET_ID`、`SMS_SECRET_KEY`、`SMS_APP_ID`、签名、模板 | 是 |
| 对象存储 | 后端生产要求 `OBJECT_STORAGE_USE_SSL=true` | 配置 HTTPS S3 兼容 endpoint、bucket、access key；本仓库未把内置 MinIO 暴露成生产 HTTPS 对象存储入口 | 是 |
| 备份对象存储 | 发布脚本会拒绝备份对象存储占位符 | 配置 `BACKUP_OBJECT_STORAGE_*`，并确认 `sync-postgres-backups.sh` 能写入 | 是 |
| PostgreSQL 备份 timer | bootstrap 脚本可安装 | 启用 dump、basebackup、backup sync timer，并完成一次恢复演练 | 是 |
| 观测与告警 | Compose 已包含 LGTM、Alertmanager、cAdvisor | 配置真实 `ALERTMANAGER_WEBHOOK_URL`，不要用本地 sink；确认 Grafana dashboard 有数据 | 是 |
| OpenFGA | 发布脚本会执行 migrate + bootstrap | 不手填模型；让 `bootstrap-platform.sh prod` 写入生成配置 | 是 |
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
id.stuhelper.com /              -> http://127.0.0.1:18000
```

宝塔面板保存后执行 Nginx 配置测试和 reload。命令路径随宝塔安装方式可能不同；至少要在面板里看到 Nginx 测试通过。

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
WEB_VITE_API_URL=/api
WEB_VITE_SSO_URL=https://sso.stuhelper.com
ADMIN_VITE_API_URL=/api/v1
ADMIN_VITE_BASE=/admin/
IDENTITY_ISSUER=https://id.stuhelper.com
TOKEN_COOKIE_SECURE=true
TOKEN_COOKIE_DOMAIN=.stuhelper.com

CASDOOR_ISSUER=https://sso.stuhelper.com
CASDOOR_REDIRECT_URI=https://stuhelper.com/api/v1/auth/callback
CASDOOR_ADMIN_REDIRECT_URI=https://stuhelper.com/api/v1/auth/callback

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
2. 校验生产必填配置、占位符、不可变镜像、TLS/SSL/SMS/OTEL 门禁
3. 渲染 PostgreSQL TLS、Redis ACL、观测配置
4. 拉取 backend / frontend / admin 镜像
5. 启动 PostgreSQL、Redis、MinIO、观测栈
6. 创建发布前逻辑备份
7. 执行数据库迁移和 OpenFGA migrate
8. bootstrap Casdoor 应用、角色、provider 与 OpenFGA 配置
9. 启动 app、frontend、admin
10. 执行业务 smoke check 和 observability smoke check
11. 写入 `.deploy/releases.log`

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

完整 smoke：

```bash
API_BASE_URL=https://stuhelper.com \
WEB_BASE_URL=https://stuhelper.com \
ADMIN_BASE_URL=https://stuhelper.com \
CASDOOR_ISSUER=https://sso.stuhelper.com \
./infra/ops/smoke-check.sh

./infra/ops/observability-smoke-check.sh
```

Grafana 中至少确认：

- `StuHelper Overview` 有最近 5 分钟数据
- `StuHelper Application` 有 API 延迟、状态码、错误率
- `StuHelper Infrastructure` 有 node-exporter 与 cAdvisor 数据
- Alertmanager receiver 指向真实值班系统

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
- `smoke-check.sh` 和 `observability-smoke-check.sh` 完整通过。
- 备份 timer 已启用，至少生成过一次逻辑备份，并能从对象存储取回。
- Grafana dashboard 有数据，Alertmanager 已接真实值班渠道。
- 回滚命令和上一版镜像可用。
