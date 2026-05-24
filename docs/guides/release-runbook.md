---
type: guide
audience: ops
status: current
authoritative-source: infra/ops/*.sh + infra/ansible/
last-verified: 2026-05-24
---

# 发布运行手册

## 适用范围

- GitLab CI/CD 自动发布
- 手工 SSH 到部署机执行的应急发布

首次生产落地先按 [production-go-live.md](production-go-live.md) 完成域名、宝塔 Nginx、secret backend、对象存储、备份与告警准备；本文只描述发布与回滚运行流程。

## 发布前检查

- [ ] 本次变更已通过 CI（web / admin / backend）
- [ ] 涉及运维脚本、部署配置、生产 evidence、Nginx preflight 或 CI 漂移门禁时，本地已执行 `make check-infra-contracts`；该入口同时覆盖 `infra/ops/tests/*.sh` 和 `infra/ops/tests/*.mjs`
- [ ] production 发布已由发布人手工审批（`deploy_production`）
- [ ] 如果包含数据库变更，已完成备份（注：`prod-deploy.sh` 现已自动在迁移前执行 `backup-postgres.sh`）
- [ ] 生产机上的逻辑备份 / base backup / backup sync timer 已启用
- [ ] 承载 `postgres_data` / `redis_data` / 对象存储目录的宿主机块设备已启用静态加密（云盘 KMS/EBS/PD 或 LUKS）
- [ ] 远端部署控制面已核对：`.deploy/remote.env`
- [ ] 共享配置已核对：`.env.prod.shared`
- [ ] secrets 已核对：`.env.prod.secrets`（本地演练可用 `.env.prod.secrets.local`）；运行时派生 secrets 必须通过 `GENERATED_ENV_SECRET_REF` 写入远端 secret backend，`.env.prod.generated.secrets` 仅保留空占位
- [ ] secret backend 已核对：`.deploy/remote.env` 中的 `SECRET_BACKEND` / `*_SECRET_REF` / `GENERATED_ENV_SECRET_REF` / `VAULT_ADDR` / `VAULT_TOKEN_FILE`
- [ ] 关键变量已核对：`POSTGRES_PASSWORD`、`REDIS_PASSWORD`、`TAG`、`OBJECT_STORAGE_*`、`WEB_VITE_SSO_URL`
- [ ] 生产 PostgreSQL TLS 已核对：默认 `POSTGRES_ENABLE_SSL=on`、`POSTGRES_INTERNAL_SSL_MODE=verify-full`（最低必须为 `verify-ca`）、`DB_SSL_MODE=verify-full`、`DB_SSL_ROOT_CERT=/tls/ca.crt`，且 `DATABASE_URL` / `BACKUP_DATABASE_URL` / `REPLICATION_DATABASE_URL` 都包含 `sslmode=verify-full&sslrootcert=/tls/ca.crt`；若生产机复用宝塔已有明文 Postgres，必须显式设置 `EXTERNAL_POSTGRES_ENABLED=true`、`EXTERNAL_DATASTORE_NETWORK=baota_net`、`EXTERNAL_POSTGRES_ALLOW_PLAINTEXT=true`，并确认外部 Postgres 已为 StuHelper / OpenFGA 创建独立数据库和独立账号、数据已从旧 StuHelper 专用库迁移到外部 Postgres。Redis 不复用全局实例，仍由 StuHelper Compose 以独立实例运行。
- [ ] Open Platform runtime token 探针已核对：`OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_REQUIRED=true`、`OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND=/app/casdoor-runtime-token-probe-runner.mjs` 且不是 `REPLACE_WITH_OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND` 占位符、`OPEN_PLATFORM_PRODUCTION_EVIDENCE_ALLOW_LOCAL_TARGETS=false`、专用低权限 `CASDOOR_TOKEN_PROBE_USERNAME` / `CASDOOR_TOKEN_PROBE_PASSWORD` 已通过 secret backend 注入；`CASDOOR_TOKEN_PROBE_SMOKE_*` 专用 smoke app 已配置，发布时会通过 `open-platform-production-evidence.sh` 自动运行 `casdoor-runtime-token-probe-smoke.sh`，且聚合 evidence 会在子 smoke 前验证强制探针门禁开启并默认拒绝 localhost Casdoor/OpenFGA 目标
- [ ] OpenFGA 派生配置已核对：`OPENFGA_STORE_ID` / `OPENFGA_MODEL_ID` 由 `bootstrap-platform.sh` 生成，`OPENFGA_RESOURCE_SMOKE_MODE=container`，发布时会通过 `open-platform-production-evidence.sh` 自动运行 `openfga-resource-access-smoke.sh`
- [ ] 公网身份入口门禁已核对：默认 `PUBLIC_INGRESS_CONFIG_PREFLIGHT_ENABLED=true`，远端 preflight / prod deploy 会先审计本机宝塔 Nginx 主站/id 配置；默认 `PUBLIC_INGRESS_PREFLIGHT_ENABLED=true`，随后验证 `stuhelper.com`、`id.stuhelper.com`、`sso.stuhelper.com` 公共 DNS-over-HTTPS 有公网 A/AAAA，`stuhelper.com`、`id.stuhelper.com` TLS 可达且 `sso.stuhelper.com` OIDC discovery 正确；默认 `PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_USE_ENV_TARGETS=false` 且 `PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_PUBLIC_DNS_ENABLED=true`，诊断脚本不会被开发 `.env` localhost 目标覆盖，并会同时记录本机 resolver 与公共 DNS-over-HTTPS 视角；默认 `IDENTITY_PUBLIC_SMOKE_ENABLED=true`，发布时会自动运行 `identity-public-smoke.sh` 并写入 `infra/generated/identity-public-smoke-evidence.json`，且 `IDENTITY_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS=false`，避免 localhost 目标误写成公网 evidence；如开启 `IDENTITY_PUBLIC_SMOKE_BOOTSTRAP_ENABLED=true`，还必须配置已存在的 `IDENTITY_PUBLIC_SMOKE_OWNER_USER_ID` / `IDENTITY_PUBLIC_SMOKE_REVIEWER_USER_ID`，发布脚本会先用 `bootstrap-identity-public-smoke-client.sh` 幂等准备专用 approved client 并重新加载 smoke 凭据；smoke 覆盖 authorize、token、introspect、revoke、GET/POST logout、POST logout URL query / JSON body 拒绝、GET/POST UserInfo 路由和 UserInfo URL query / body token 来源拒绝，并核对 discovery 暴露 `authorization_code` / `refresh_token` / `client_credentials` 以及 token / introspect / revoke endpoint 支持的 `client_secret_basic` / `client_secret_post`；如已配置专用 approved `IDENTITY_PUBLIC_SMOKE_CLIENT_ID` / `IDENTITY_PUBLIC_SMOKE_REDIRECT_URI`，evidence 还必须包含 `prompt=none` 的 `login_required` + `iss` 错误回调检查；如同时配置 `IDENTITY_PUBLIC_SMOKE_CLIENT_SECRET`，evidence 还必须证明 `client_credentials` token 签发、携带 `token_type_hint=access_token` 的 introspection / revoke、UserInfo 拒绝、Open Platform resource access API 对未授权随机资源返回 `fga_denied` 或对预授权 smoke 资源返回 `allowed`、revoke 后 inactive 均通过且未记录 secret/token
- [ ] 观测配置已核对：`METRICS_PASSWORD`、`GRAFANA_ADMIN_PASSWORD`、`OTEL_ENABLED=true`
- [ ] staging 已验证通过（如有 staging）
- [ ] 发布 bundle 从干净 Git 工作区打包；`git status --short` 为空，所有待发布改动已经提交并签名

## GitLab 自动发布链路

### staging（develop）

1. `frontend_e2e`
2. `admin_e2e`
3. `package_backend`
4. `package_frontend`
5. `package_admin`
6. `deploy_staging`
7. `verify_staging`
8. 远端实际执行：
   - `./infra/ops/remote-preflight.sh`
   - `./infra/ops/remote-prod-deploy.sh`

### production（main）

1. `frontend_e2e`
2. `admin_e2e`
3. `backend_security` / `backend_vulnerability_scan`
4. `frontend_dependency_scan` / `admin_dependency_scan`
5. `container_scan_backend` / `container_scan_frontend` / `container_scan_admin`
6. `package_backend`
7. `package_frontend`
8. `package_admin`
9. 手工触发 `deploy_production`
10. `verify_production`
11. 远端实际执行：
    - `./infra/ops/remote-preflight.sh`
    - `./infra/ops/remote-prod-deploy.sh`

只要 Smoke Check 失败，本次发布就视为失败，需要立刻进入回滚判断。

## 手工发布步骤

```bash
cd /path/to/StuHelper

git fetch --all --prune
git checkout <target-ref>

# 首次或控制面变更后，准备共享配置 / 本地 secrets / 运行时派生 secrets skeleton
make prod-init

# 远端机器需要自持部署控制面
./infra/ops/init-remote-deploy-config.sh

# 可选：指定要发布的不可变镜像
export BACKEND_IMAGE_REF=<registry/backend:sha-or-tag>
export FRONTEND_IMAGE_REF=<registry/frontend:sha-or-tag>
export ADMIN_IMAGE_REF=<registry/admin:sha-or-tag>
export TAG=<release-id>

# 远端机器建议先做预检
./infra/ops/remote-preflight.sh

# 生产发布
make prod-deploy
```

## 发布后验证

- API：`http://127.0.0.1:18080/health/live`
- API：`http://127.0.0.1:18080/health/ready`
- Web：首页 200
- Admin：首页 200
- Grafana：`http://127.0.0.1:3003`
- Prometheus：`http://127.0.0.1:9090/-/ready`
- Loki：`http://127.0.0.1:3100/ready`
- Tempo：`http://127.0.0.1:3200/ready`
- 公网身份入口：`./infra/ops/identity-public-smoke.sh`，留档 `infra/generated/identity-public-smoke-evidence.json`
- 公网身份入口诊断：`./infra/ops/public-identity-ingress-diagnostic.sh`，失败时留档 `infra/generated/public-identity-ingress-diagnostic.json`
- OpenFGA 资源授权单项复跑：`./infra/ops/openfga-resource-access-smoke.sh`
- Open Platform 生产准入证据留档：`./infra/ops/open-platform-production-evidence.sh`
- PostgreSQL 备份证据留档：`./infra/ops/postgres-backup-evidence.sh`
- 观测证据留档：`OBS_SMOKE_STRICT=true ./infra/ops/observability-smoke-check.sh`
- `docker compose -f docker-compose.yml -f docker-compose.observability.yml -f docker-compose.prod.yml --profile prod ps` 中 `app` / `frontend` / `admin` 为 healthy/running

`open-platform-production-evidence.sh` 会把 Casdoor runtime token probe 和 OpenFGA resource smoke 的
关键结果汇总到 `infra/generated/open-platform-production-evidence.json`，并确认子 evidence 匹配当前
`CASDOOR_ISSUER`、`OPENFGA_API_URL`、`OPENFGA_STORE_ID` 和 `OPENFGA_MODEL_ID`。留档前确认
`.passed == true`、runtime token probe command 不是模板占位符、`.configuration.runtimeTokenProbeRequired == true`、
`.casdoorRuntimeTokenProbe.businessClaims == []`、`.casdoorRuntimeTokenProbe.metadata.nonceVerified == true`，
并确认 OpenFGA grant check/list 为 true、revoke 后 check/list 均为 false。
失败诊断只输出 claim 名称、nonce 校验布尔值和 OpenFGA 布尔断言等白名单字段，不输出 raw token、
client secret 或 probe 密码。

`postgres-backup-evidence.sh` 会验证本地最近逻辑备份和从对象存储取回的逻辑备份都带有效
`.sha256` sidecar，并写入 `infra/generated/postgres-backup-evidence.json`。

`OBS_SMOKE_STRICT=true ./infra/ops/observability-smoke-check.sh` 会写入
`infra/generated/observability-smoke-evidence.json`，证明 Prometheus targets、blackbox probes
和 Alertmanager receiver 配置可用。

## 回滚步骤

### 仅应用层回滚（数据库兼容）

```bash
cd /path/to/StuHelper

# 指定 tag 回滚
export ROLLBACK_TAG=<previous-stable-tag>
make prod-rollback
```

如果不指定 `ROLLBACK_TAG`：

```bash
cd /path/to/StuHelper
make prod-rollback
```

脚本会优先尝试回到 `.deploy/releases.log` 中记录的上一条成功版本。

### GitLab 手工回滚

如果镜像仍在 registry 中，优先使用 GitLab 手工 Job：

- staging：`rollback_staging`
- production：`rollback_production`

可选变量：

```text
ROLLBACK_TAG=<previous-stable-tag-or-sha>
```

回滚 Job 会：

1. SSH 到远端部署机
2. 读取目标机 `.deploy/remote.env`
3. 如果传了 `ROLLBACK_TAG`，按它执行；否则自动回滚到上一条成功发布记录
4. 自动再次运行 smoke checks

### 应用 + 数据库回滚（存在破坏性迁移）

1. 切回上一版 `TAG`
2. 使用备份恢复数据库（见 [backup-and-restore.md](backup-and-restore.md)）
3. 再次执行 `make prod-deploy`
4. 重新跑 `./infra/ops/smoke-check.sh`
5. 重新跑 `./infra/ops/openfga-resource-access-smoke.sh`
6. 重新跑 `./infra/ops/observability-smoke-check.sh`

## 发布记录模板

```text
时间:
执行人:
目标版本:
是否含迁移:
备份文件:
Smoke Check 结果:
回滚情况:
备注:
```

## TLS 终止策略

### 方案选择

StuHelper 当前生产入口采用宝塔 Nginx。Docker Compose 只把业务服务绑定到宿主机回环地址的非常用端口，不直接监听公网 `80/443`。

| 模式 | 适用场景 | 证书管理 |
|------|---------|---------|
| **Baota Nginx** | 当前生产默认 | 宝塔面板管理证书与站点反代 |
| **External LB/CDN + Baota Nginx** | 云厂商 LB / CDN 前置 | 外部终止 TLS，或外部转发 HTTPS 到宝塔 |

### 方案 A：宝塔 Nginx（当前默认）

宝塔 Nginx 是唯一公网入口，负责 `stuhelper.com` 的 `80/443`、证书、HTTP 到 HTTPS 跳转和反向代理。仓库提供反代示例：

```text
infra/nginx/baota-stuhelper.conf
infra/nginx/baota-casdoor-sso.conf
```

`baota-stuhelper.conf` 用于主站机器的 `stuhelper.com` / `id.stuhelper.com`；`baota-casdoor-sso.conf` 用于外部 SSO 机器的 `sso.stuhelper.com`。SSO 配置必须显式代理 `/.well-known/openid-configuration` 和 discovery 中的 JWKS 路径到 Casdoor upstream，避免 OIDC 元数据被宝塔静态站点 404 覆盖。

保存或 reload 前先审计实际配置：

```bash
NGINX_PUBLIC_INGRESS_PROFILE=stuhelper ./infra/ops/nginx-public-ingress-preflight.sh
NGINX_PUBLIC_INGRESS_PROFILE=sso ./infra/ops/nginx-public-ingress-preflight.sh
```

第一条在主站生产机执行，第二条在外部 Casdoor SSO 机器执行。远端发布主机默认使用 `NGINX_PUBLIC_INGRESS_PROFILE=stuhelper`，只审计本机拥有的 `stuhelper.com` / `id.stuhelper.com` server block；`sso.stuhelper.com` 由 SSO 机器本地审计和公网 OIDC discovery/JWKS 门禁共同覆盖。

生产端口默认值：

```bash
BACKEND_EXTERNAL_PORT=18080
WEB_EXTERNAL_PORT=18000
ADMIN_EXTERNAL_PORT=18001
```

反代目标：

- `/api/*` → `http://127.0.0.1:18080`
- `/admin/*` → `http://127.0.0.1:18001`
- `/` → `http://127.0.0.1:18000`

### 方案 B：External LB/CDN + 宝塔 Nginx

由外部负载均衡器或 CDN 作为第一层入口，宝塔 Nginx 仍然作为应用反代层。

配置要求：

- 外部层必须保留 `X-Forwarded-Proto: https`。
- 宝塔 Nginx 必须继续把 `Host`、`X-Forwarded-Host`、`X-Real-IP`、`X-Forwarded-For` 传给后端。
- `TRUSTED_PROXIES` 要包含可信代理网段。

### TLS 终止检查清单

发布前确认：

- [ ] `CASDOOR_ISSUER=https://sso.stuhelper.com`
- [ ] `IDENTITY_ISSUER=https://id.stuhelper.com`
- [ ] `IDENTITY_REFRESH_TOKEN_TTL` 符合生产离线授权策略（默认 2592000 秒，允许 3600 到 2592000 秒）
- [ ] `CORS_ORIGINS=https://stuhelper.com,https://id.stuhelper.com`
- [ ] `WEB_VITE_SSO_URL=https://sso.stuhelper.com`
- [ ] 主站生产机 `NGINX_PUBLIC_INGRESS_PROFILE=stuhelper ./infra/ops/nginx-public-ingress-preflight.sh` 通过
- [ ] SSO 机器 `NGINX_PUBLIC_INGRESS_PROFILE=sso ./infra/ops/nginx-public-ingress-preflight.sh` 通过
- [ ] `https://id.stuhelper.com/.well-known/openid-configuration` 可达
- [ ] `https://sso.stuhelper.com/.well-known/openid-configuration` 可达
- [ ] 如果公网入口检查失败，已运行 `./infra/ops/public-identity-ingress-diagnostic.sh` 并留档脱敏 `infra/generated/public-identity-ingress-diagnostic.json`
- [ ] `TOKEN_COOKIE_SECURE=true`（生产必须）
- [ ] `TOKEN_COOKIE_DOMAIN=.stuhelper.com`（`id.stuhelper.com` 授权页复用主站登录会话）
- [ ] 宝塔 Nginx 是唯一监听公网 `80/443` 的入口
- [ ] `127.0.0.1:18080`、`127.0.0.1:18000`、`127.0.0.1:18001` 在宿主机可访问且未暴露公网
- [ ] 如果使用 External LB/CDN，`TRUSTED_PROXIES` 已正确配置
