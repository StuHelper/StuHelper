---
type: guide
audience: ops
status: current
authoritative-source: infra/ops/*.sh + infra/ansible/
last-verified: 2026-05-25
---

# 发布运行手册

## 适用范围

- GitLab CI/CD 自动发布。
- 手工 SSH 到部署机执行的应急发布。

首次生产落地先按 [production-go-live.md](production-go-live.md) 完成域名、宝塔 Nginx、secret backend、对象存储、备份与告警准备；本文只描述发布与回滚运行流程。

## 发布前检查

- [ ] 本次变更已通过 CI（web / admin / backend / koishi）。
- [ ] admission MVP 相关变更已在本地执行 `make check-admission-mvp`；该入口覆盖 admission 后端、auth/user 依赖后端、Web admission 和用户认证 Vitest、认证/admission 与用户中心 Playwright、Web build、Koishi group guard、生产入口和 evidence 契约。
- [ ] 涉及运维脚本、部署配置、生产 evidence、Nginx preflight 或 CI 漂移门禁时，本地已执行 `make check-infra-contracts`；该入口同时覆盖 `infra/ops/tests/*.sh` 和 `infra/ops/tests/*.mjs`。
- [ ] production 发布已由发布人手工审批（`deploy_production`）。
- [ ] 如果包含数据库变更，已完成备份（注：`prod-deploy.sh` 现已自动在迁移前执行 `backup-postgres.sh`）。
- [ ] 生产机上的逻辑备份 / base backup / backup sync timer 已启用。
- [ ] 承载 `postgres_data` / `redis_data` / 对象存储目录的宿主机块设备已启用静态加密（云盘 KMS/EBS/PD 或 LUKS）。
- [ ] 远端部署控制面已核对：`.deploy/remote.env`。
- [ ] 共享配置已核对：`.env.prod.shared`。
- [ ] secrets 已核对：`.env.prod.secrets`（本地演练可用 `.env.prod.secrets.local`）；运行时派生 secrets 必须通过 `GENERATED_ENV_SECRET_REF` 写入远端 secret backend，`.env.prod.generated.secrets` 仅保留空占位。
- [ ] secret backend 已核对：`.deploy/remote.env` 中的 `SECRET_BACKEND` / `*_SECRET_REF` / `GENERATED_ENV_SECRET_REF` / `VAULT_ADDR` / `VAULT_TOKEN_FILE`。
- [ ] 关键变量已核对：`POSTGRES_PASSWORD`、`REDIS_PASSWORD`、`TAG`、`OBJECT_STORAGE_*`、`ADMISSION_PUBLIC_BASE_URL=https://join.stuhelper.com`、`WEB_VITE_SSO_URL=https://sso.stuhelper.com`、`WEB_VITE_IDENTITY_URL=`、`WEB_VITE_WEB_URL=https://stuhelper.com`。
- [ ] admission 最小生产数据已通过 `./infra/ops/import-school-directory.sh` 和 `./infra/ops/admission-bootstrap-production-data.sh` 幂等准备：学校目录包含 `school_code=4111010006` 的北京航空航天大学，当前只启用该校，公开学生认证/admission 表单以 `schoolCode=4111010006` 为主识别字段，邮箱域仅 `buaa.edu.cn`，`platform=qq` 的 `178037297` 策略存在，`forward_raw_material_to_qq=false`。
- [ ] Koishi/NapCat 独立节点已确认：Koishi service 使用 `env_file` 或等价机制注入 `STUHELPER_PLATFORM_BASE_URL=https://stuhelper.com`、`STUHELPER_PLATFORM_SERVICE_TOKEN`、`STUHELPER_FRESHMAN_MATERIAL_HOSTS=stuhelper.com,join.stuhelper.com`；真实 token 不写入仓库或 runbook。
- [ ] 生产 PostgreSQL TLS 已核对：默认 `POSTGRES_ENABLE_SSL=on`、`POSTGRES_INTERNAL_SSL_MODE=verify-full`（最低必须为 `verify-ca`）、`DB_SSL_MODE=verify-full`、`DB_SSL_ROOT_CERT=/tls/ca.crt`，且 `DATABASE_URL` / `BACKUP_DATABASE_URL` / `REPLICATION_DATABASE_URL` 都包含 `sslmode=verify-full&sslrootcert=/tls/ca.crt`；若生产机复用宝塔已有明文 Postgres，必须显式设置 `EXTERNAL_POSTGRES_ENABLED=true`、`EXTERNAL_DATASTORE_NETWORK=baota_net`、`EXTERNAL_POSTGRES_ALLOW_PLAINTEXT=true`，并确认外部 Postgres 已为 StuHelper / OpenFGA 创建独立数据库和独立账号、数据已从旧 StuHelper 专用库迁移到外部 Postgres。Redis 不复用全局实例，仍由 StuHelper Compose 以独立实例运行。
- [ ] Open Platform runtime token 探针已核对：`OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_REQUIRED=true`、`OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND=/app/casdoor-runtime-token-probe-runner.mjs` 且不是 `REPLACE_WITH_OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND` 占位符；`OPEN_PLATFORM_PRODUCTION_EVIDENCE_ALLOW_LOCAL_TARGETS=false`；专用低权限 `CASDOOR_TOKEN_PROBE_USERNAME` / `CASDOOR_TOKEN_PROBE_PASSWORD` 已通过 secret backend 注入；`CASDOOR_TOKEN_PROBE_SMOKE_*` 专用 smoke app 已配置，发布时会通过 `open-platform-production-evidence.sh` 自动运行 `casdoor-runtime-token-probe-smoke.sh`，且聚合 evidence 会在子 smoke 前验证强制探针门禁开启并默认拒绝 localhost Casdoor/OpenFGA 目标。
- [ ] OpenFGA 派生配置已核对：`OPENFGA_STORE_ID` / `OPENFGA_MODEL_ID` 由 `bootstrap-platform.sh` 生成，`OPENFGA_RESOURCE_SMOKE_MODE=container`，发布时会通过 `open-platform-production-evidence.sh` 自动运行 `openfga-resource-access-smoke.sh`。
- [ ] 公网身份和入群验证入口门禁已核对：默认 `PUBLIC_INGRESS_CONFIG_PREFLIGHT_ENABLED=true`，远端 preflight / prod deploy 会先审计本机宝塔 Nginx 主站、id、join 配置；默认 `PUBLIC_INGRESS_PREFLIGHT_ENABLED=true`，随后验证 `stuhelper.com`、`join.stuhelper.com`、`sso.stuhelper.com` 公共 DNS-over-HTTPS 有公网 A/AAAA 且 TLS 可达，并确认 `id.stuhelper.com` 返回 404。`sso.stuhelper.com` discovery/JWKS/authorize 路由必须反代到 Casdoor。默认 `IDENTITY_PUBLIC_SMOKE_ENABLED=false`，发布时不再运行旧 `id` identity smoke；默认 `ADMISSION_PUBLIC_SMOKE_ENABLED=true`，远端 preflight / prod deploy 都会运行 `admission-public-smoke.sh` 并写入 `infra/generated/admission-public-smoke-evidence.json`，要求 `join.stuhelper.com/verify/<probe>?qq=<qq>` 由 Web SPA 承载，`join.stuhelper.com/api/v1/metrics/vitals` 和 `/api/v1/metrics/frontend-errors` 接受同源 beacon 并返回 204，`join.stuhelper.com/api/v1/admission/freshman/camera-handoffs/<probe>/events` 无登录探测返回 401 且 `X-Accel-Buffering: no`，`join.stuhelper.com/verify`、主站 `/verify`、主站 `/verify/*`、`id.stuhelper.com/verify`、`id.stuhelper.com/verify/*` 全部返回 404，且 `ADMISSION_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS=false`、`ADMISSION_PUBLIC_SMOKE_CURL_INSECURE=false`。默认 `PUBLIC_WEB_AUTH_BROWSER_SMOKE_ENABLED=true`，prod deploy 会运行 `public-web-auth-browser-smoke.mjs`，用 Playwright 真浏览器验证主站登录页非空、开发者入口未登录跳回 `/login?redirect=/developers/apps`、点击登录进入 `sso.stuhelper.com/login/oauth/authorize` 并看到账号密码登录和 `/signup/oauth/authorize` 注册入口、点击注册进入 `sso.stuhelper.com/signup/oauth/authorize` 并看到账号密码注册表单、join verify SPA 可加载、join 登录/注册入口进入 SSO 登录/注册授权页、`id.stuhelper.com` 禁用。
- [ ] 观测配置已核对：`METRICS_PASSWORD`、`GRAFANA_ADMIN_PASSWORD`、`OTEL_ENABLED=true`。
- [ ] staging 已验证通过（如有 staging）。
- [ ] 发布 bundle 从干净 Git 工作区打包；`git status --short` 为空，所有待发布改动已经提交并签名。

## GitLab 自动发布链路

### staging（develop）

1. `frontend_e2e`
2. `admin_e2e`
3. `uniappx_e2e`
4. `koishi_test`
5. `package_backend`
6. `package_frontend`
7. `package_admin`
8. `deploy_staging`
9. `verify_staging`
10. 远端实际执行：`./infra/ops/remote-preflight.sh` 和 `./infra/ops/remote-prod-deploy.sh`

### production（main）

1. `frontend_e2e`
2. `admin_e2e`
3. `uniappx_e2e`
4. `koishi_test`
5. `backend_security` / `backend_vulnerability_scan`
6. `frontend_dependency_scan` / `admin_dependency_scan`
7. `container_scan_backend` / `container_scan_frontend` / `container_scan_admin`
8. `package_backend`
9. `package_frontend`
10. `package_admin`
11. 手工触发 `deploy_production`
12. `verify_production`
13. 远端实际执行：`./infra/ops/remote-preflight.sh` 和 `./infra/ops/remote-prod-deploy.sh`

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

### 宝塔 Compose 源码包应急发布

如果当前生产机不能直接从 registry 拉取镜像，且宝塔 Compose 实际运行目录采用“Compose 根目录 + `source/` 源码副本”的形态，可以执行一次源码包 + 本地镜像 tar 的应急发布。该流程只接受由本地当前仓库生成的产物；不得在生产 `source/`、容器文件系统或 `node_modules` 中手工改代码作为最终状态。

要求：本地打包前记录当前 Git ref、`git status --short` 和源码包 `sha256sum`。源码包不得包含真实 `.env*`、`.deploy/`、`node_modules`、`dist`、临时 SSH 脚本或本地 secret。生产 `source/.env.prod.shared`、`source/.env.prod.secrets.local`、`source/.env.prod.generated`、`source/.env.prod.generated.secrets` 只从旧生产目录保留或由 secret backend 重新生成，不从源码包覆盖。

如果旧生产目录已有 `source/infra/generated`，恢复到新 `source/infra/generated` 时复制整个 `generated` 目录本身，目标必须是 `source/infra/generated`，不能变成 `source/infra/generated/generated`；PostgreSQL TLS 挂载依赖 `source/infra/generated/postgres/ca.crt`。镜像必须在本地从当前代码构建，上传 tar 后在生产执行 `sha256sum -c`，再 `docker load`；记录 backend / web / admin 镜像 ID 和 tar sha256。数据库 bootstrap、migration、readiness、public smoke 仍运行仓库脚本。`admission-bootstrap-production-data.sh` 和 `admission-production-readiness.sh` 在宝塔 `source/` 目录下会自动识别 `.env.prod.shared`、`.env.prod.secrets.local`、`.env.prod.generated`、`.env.prod.generated.secrets`。

重建容器必须用宝塔实际 Compose 根目录和实际 env file，例如：

```bash
docker compose \
  --env-file source/.env.prod.shared \
  --env-file source/.env.prod.secrets.local \
  --env-file source/.env.prod.generated \
  --env-file source/.env.prod.generated.secrets \
  up -d --no-deps --force-recreate app frontend admin
```

如果这一路径中发现必须修改生产 Nginx、env 或 DB 数据，修改后的非敏感结构必须同步回仓库模板、脚本或本文档；真实 secret 只保留在生产 secret backend / env 文件中。

### 生产磁盘耗尽应急处理

如果 Koishi 或后端日志出现 `failed to verify bot service credential` / `B0000001`，同时 PostgreSQL 日志出现 `No space left on device` / `could not write to file "pg_wal/xlogtemp.*"` 或容器反复 `Restarting`，优先按主站磁盘耗尽处理，而不是先旋转 bot token。

先做只读确认：

```bash
df -h /
docker logs --tail 120 stuhelper-prod-postgres
docker compose --env-file source/.env.prod.shared --env-file source/.env.prod.secrets.local --env-file source/.env.prod.generated --env-file source/.env.prod.generated.secrets ps
```

安全释放空间时使用仓库脚本，先 dry-run，再显式 apply：

```bash
./infra/ops/reclaim-production-disk-space.sh
./infra/ops/reclaim-production-disk-space.sh --apply
```

默认情况下，该脚本只清理 systemd journal、apt cache、Docker build cache 和 dangling images。可按需显式打开以下应急开关：

```bash
PRUNE_STUHELPER_TMP_ARTIFACTS=true \
PRUNE_BAOTA_PANEL_BACKUPS_KEEP=1 \
TRUNCATE_VAR_LOG_MESSAGES=true \
./infra/ops/reclaim-production-disk-space.sh --apply
```

如果根因是 PostgreSQL WAL archive 在同一根分区快速增长，先确认 `archive_status/*.ready` 不在堆积，再按“恢复在线服务优先”的应急策略清理旧归档段。该操作会破坏本机被删除时间窗口之前的 PITR 链，必须写 manifest，不得作为常规运维：

```bash
mkdir -p backups/postgres/logical
docker exec stuhelper-prod-postgres sh -lc \
  'pg_dump -U "${POSTGRES_USER:-stuhelper}" -d "${POSTGRES_DB:-stuhelper}" --format=custom --no-owner --no-privileges' \
  > "backups/postgres/logical/emergency-before-wal-prune-$(date -u +%Y%m%dT%H%M%SZ).dump"
sha256sum backups/postgres/logical/emergency-before-wal-prune-*.dump \
  > backups/postgres/logical/emergency-before-wal-prune.sha256

ALLOW_POSTGRES_WAL_ARCHIVE_PRUNE=true \
PRUNE_POSTGRES_WAL_ARCHIVE_KEEP_HOURS=6 \
./infra/ops/reclaim-production-disk-space.sh --apply
```

恢复后确认运行配置不再使用过短的 WAL 切段间隔。仓库模板默认 `POSTGRES_ARCHIVE_TIMEOUT=15min`；如果生产实际 compose 根文件仍是 `archive_timeout=60s`，应同步改为 `15min` 并只重建 PostgreSQL 服务让参数生效。

禁止执行 `docker volume prune`、删除 `/var/lib/docker/volumes`、删除 PostgreSQL `PGDATA`。`wal-archive` 只能通过上述带 manifest 的显式应急开关清理。恢复后确认：

```bash
df -h /
docker ps --filter name=stuhelper-prod-postgres --format '{{.Names}} {{.Status}}'
curl -fsS -o /dev/null -w 'ready=%{http_code}\n' http://127.0.0.1:18080/health/ready
```

如果 PostgreSQL 日志持续出现 `archive command failed`，并且命令形态是 `test ! -f ... && cp ...`，说明 WAL archive 命令不是幂等的：目标文件已存在时会返回失败。仓库 `docker-compose.yml` 的 `archive_command` 必须保持“目标不存在则原子复制；目标已存在且内容一致则成功；目标已存在但内容不一致则失败报警”的语义。修改生产 Compose 后，必须同步回仓库并用同一份 Compose 重新创建 PostgreSQL 容器；不要只在容器内 `ALTER SYSTEM` 留下漂移。

如果 WAL archive 里出现磁盘耗尽留下的残缺文件，例如归档目录中的某个 WAL 段小于 16 MiB，而 `PGDATA/pg_wal` 中同名文件为 16 MiB，不能把归档命令改成静默覆盖。应先确认文件名两边大小和 `cmp` 结果，再把残缺归档移动到带日期的 `quarantine-incomplete-*` 目录，保留证据后让 PostgreSQL archiver 重新归档完整文件。

如果本地 WAL archive 和 `PGDATA/pg_wal` 共用根分区，且 WAL archive 没有独立磁盘、对象存储或保留周期，归档失败会导致 `pg_wal` 无法回收，最终把根分区再次打满。短期应先用 `df -h /`、`du -xsh /var/lib/docker/volumes/*postgres*`、`archive_status/*.ready` 数量确认压力；长期必须把 WAL archive 迁到独立存储或启用仓库备份脚本中的保留策略。应急清理旧 WAL archive 会破坏本机历史 PITR 归档链，只能在明确“恢复在线服务优先”并记录清理 manifest 后执行，不能作为常规运维手段。

## 发布后验证

- API：`http://127.0.0.1:18080/health/live`
- API：`http://127.0.0.1:18080/health/ready`
- Web：首页 200
- Admin：首页 200
- Grafana：`http://127.0.0.1:3003`
- Prometheus：`http://127.0.0.1:9090/-/ready`
- Loki：`http://127.0.0.1:3100/ready`
- Tempo：`http://127.0.0.1:3200/ready`
- 公网 SSO 入口：`./infra/ops/sso-public-smoke.sh`，留档 `infra/generated/sso-public-smoke-evidence.json`。该检查会断言 `admin/stuhelper-web` 的公开 Casdoor application 元数据仍启用密码登录、注册和 signin session，避免生产漂移成只剩 Face ID。
- 公网入群验证入口：`./infra/ops/admission-public-smoke.sh`，留档 `infra/generated/admission-public-smoke-evidence.json`，同时验证 join 域 `/api/v1/metrics/vitals` 和 `/api/v1/metrics/frontend-errors` 同源 beacon 返回 204，并验证手机拍照接力 SSE 入口 `/api/v1/admission/freshman/camera-handoffs/<probe>/events` 无登录返回 401 且禁用 Nginx buffering。
- 公网 Web 登录浏览器链路：`./infra/ops/public-web-auth-browser-smoke.mjs`，留档 `infra/generated/public-web-auth-browser-smoke-evidence.json`。该检查会用真实浏览器确认登录按钮进入 `sso.stuhelper.com/login/oauth/authorize` 后仍有账号密码登录和注册入口，确认主站“注册账号”进入 `sso.stuhelper.com/signup/oauth/authorize` 的账号密码注册表单，并确认 join 入群登录页的登录/注册按钮也走对应 SSO 授权页。
- 新生材料审核员只读准入：`./infra/ops/admission-reviewer-readiness.sh` 或 `make prod-admission-reviewer-readiness`，留档 `infra/generated/admission-reviewer-readiness.json`。该检查只调用 bot view 接口，不会批准或驳回申请，用来确认至少一个管理群 QQ 已绑定 StuHelper 用户且拥有 `admission:freshman:review` 能力。
- 学校邮箱 OTP 邮件准入：生产应使用 `EMAIL_DRIVER=multi`、`EMAIL_FROM_NAME=StuHelper 系统邮件`、`EMAIL_STUDENT_VERIFICATION_SUBJECT=学生认证验证码`。`./infra/ops/tencent-ses-template-smoke.sh` 留档 `infra/generated/tencent-ses-template-smoke.json`；该检查不发送邮件，只用生产 secret 调腾讯云 SES `GetEmailTemplate`，要求 `EMAIL_TENCENT_TEMPLATE_ID=49779` 且模板状态为已审核，输出不包含 Secret。`EMAIL_RESEND_API_KEY` 只能在 secret env/secret store 中配置；Resend 兜底发送不使用模板，后端直接发送 HTML/text。管理后台“系统配置”中的 `email.delivery_policy` 用于调整 provider 启用状态、优先级、权重和 `priority`/`weighted` 策略。
- admission MVP 聚合生产证据：`./infra/ops/admission-mvp-production-evidence.sh`，留档 `infra/generated/admission-mvp-production-evidence.json`。主站节点默认聚合 SSO public smoke、admission public smoke、Web auth browser smoke 和 admission DB readiness；Koishi 节点使用 `ADMISSION_MVP_PRODUCTION_EVIDENCE_MODE=koishi` 聚合 Koishi admission evidence。该普通入口允许真实 QQ E2E 被记录为 skipped，只能作为生产 smoke。最终上线验收必须在主站节点使用 `make prod-admission-mvp-final-evidence`，并在 Koishi 节点使用 `make prod-admission-mvp-final-koishi-evidence`。主站 final evidence 等价于显式设置 `ADMISSION_MVP_PRODUCTION_E2E_REQUIRED=true`、`ADMISSION_MVP_PRODUCTION_E2E_WAIT=true`、`ADMISSION_E2E_QQ_ID=<small-account-qq>`、`ADMISSION_MVP_PRODUCTION_E2E_EXPECTED_STAGE=bot-released` 和 `ADMISSION_MVP_PRODUCTION_E2E_MAX_SESSION_AGE_MINUTES=180`。
- 如果主站生产机没有 Node/Playwright，先在有 Playwright 的运维机或 CI 上生成 `infra/generated/public-web-auth-browser-smoke-evidence.json`，复制到主站源码目录，再运行聚合入口并设置 `ADMISSION_MVP_PRODUCTION_BROWSER_SMOKE_EVIDENCE_FILE=infra/generated/public-web-auth-browser-smoke-evidence.json`。聚合入口会校验该 evidence 新鲜、目标域名正确、十个浏览器检查全部通过，以及 `/identity` 直接入口、join 登录/注册入口、camera permission/media capture 成功。
- admission MVP 最终证据校验：`./infra/ops/admission-mvp-final-evidence-verify.sh`，留档 `infra/generated/admission-mvp-final-evidence-verify.json`。该脚本只读取已采集的脱敏 evidence，不访问生产；它要求主站聚合 evidence、join E2E 子证据和 Koishi evidence 都新鲜，主站/Koishi 聚合 evidence 无 failed/skipped，并要求主站包含真实 QQ `bot-released`、Koishi evidence 不包含真实 QQ E2E placeholder、join E2E 子证据显示 token 已消费、QQ 已绑定、存在 active student verification credential、后端记录 bot release 和 cancelled marker，且包含通过的 `release requires active student verification credential` 检查。
- admission 最小数据初始化：`./infra/ops/admission-bootstrap-production-data.sh`
- 公网 SSO / legacy id 入口诊断：`./infra/ops/public-identity-ingress-diagnostic.sh`，失败时留档 `infra/generated/public-identity-ingress-diagnostic.json`
- OpenFGA 资源授权单项复跑：`./infra/ops/openfga-resource-access-smoke.sh`
- Open Platform 生产准入证据留档：`./infra/ops/open-platform-production-evidence.sh`
- PostgreSQL 备份证据留档：`./infra/ops/postgres-backup-evidence.sh`
- 观测证据留档：`OBS_SMOKE_STRICT=true ./infra/ops/observability-smoke-check.sh`
- `docker compose -f docker-compose.yml -f docker-compose.observability.yml -f docker-compose.prod.yml --profile prod ps` 中 `app` / `frontend` / `admin` 为 healthy/running。

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

当前发布链路不提供 Traefik 入口模式。不要在同一生产机上再启动 Traefik 监听公网 `80/443`，也不要把 `stuhelper.com` / `id.stuhelper.com` 同时分散到 Traefik 和宝塔 Nginx；如确实需要外部负载均衡，只允许放在宝塔 Nginx 前面，并保持宝塔 Nginx 作为应用反代层。

### 方案 A：宝塔 Nginx（当前默认）

宝塔 Nginx 是唯一公网入口，负责 `stuhelper.com` 的 `80/443`、证书、HTTP 到 HTTPS 跳转和反向代理。仓库提供反代示例：

```text
infra/nginx/baota-stuhelper.conf
```

`baota-stuhelper.conf` 用于主站机器的 `stuhelper.com` / `id.stuhelper.com` / `join.stuhelper.com`。`join.stuhelper.com` 只承载加群验证业务入口，公开验证链接固定为 `https://join.stuhelper.com/verify/<token>?qq=<qq>`；主站和 `id` 上的 `/verify/*` 必须返回 404。`id.stuhelper.com` 是 legacy disabled host，所有路径返回 404；Casdoor 通过 `sso.stuhelper.com` 公开，登录回调固定回到 `https://stuhelper.com/api/v1/auth/callback`。

`sso.stuhelper.com/.well-known/openid-configuration` 和 `sso.stuhelper.com/.well-known/jwks` 必须返回 Casdoor JSON，不能被宝塔证书校验目录的静态 `location /.well-known` 抢占。按 `infra/nginx/baota-casdoor-sso.conf` 把 exact `/.well-known/openid-configuration`、exact `/.well-known/jwks`、`^~ /.well-known/` 放在宝塔静态 `/.well-known` 之前，并用 `NGINX_PUBLIC_INGRESS_PROFILE=sso ./infra/ops/nginx-public-ingress-preflight.sh` 审计。

保存或 reload 前先审计实际配置：

```bash
NGINX_PUBLIC_INGRESS_PROFILE=stuhelper ./infra/ops/nginx-public-ingress-preflight.sh
```

宝塔 vhost 必须能从仓库模板重复应用。默认先 dry-run；只有确认目标机器拥有对应 vhost 后才执行 `--apply`：

```bash
./infra/ops/apply-baota-nginx-templates.sh --profile all
sudo ./infra/ops/apply-baota-nginx-templates.sh --profile all --apply --reload --preflight
```

如果宝塔面板保存站点后把 `sso.stuhelper.com.conf` 重写，导致 `/.well-known/openid-configuration` 返回 404 或 HTML，按仓库模板只恢复 SSO vhost：

```bash
sudo ./infra/ops/apply-baota-nginx-templates.sh --profile sso --apply --reload --preflight
./infra/ops/sso-public-smoke.sh
```

`--profile sso` 会同时安装主 vhost 模板和宝塔扩展片段：`infra/nginx/baota-casdoor-sso-well-known-extension.conf` 会写入 `/www/server/panel/vhost/nginx/extension/sso.stuhelper.com/stuhelper-sso-well-known.conf`。只要宝塔重写后的主 vhost 仍保留 `include /www/server/panel/vhost/nginx/extension/sso.stuhelper.com/*.conf;`，这个扩展片段就能让 discovery/JWKS 继续由 Casdoor 返回 JSON，而不是落到静态 `location /.well-known`。

远端发布主机默认使用 `NGINX_PUBLIC_INGRESS_PROFILE=stuhelper`，审计本机拥有的 `stuhelper.com` / `id.stuhelper.com` / `join.stuhelper.com` server block。`NGINX_PUBLIC_INGRESS_PROFILE=sso` 用于审计独立 `sso.stuhelper.com` Casdoor 公网入口。

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

配置要求：外部层必须保留 `X-Forwarded-Proto: https`。宝塔 Nginx 必须继续把 `Host`、`X-Forwarded-Host`、`X-Real-IP`、`X-Forwarded-For` 传给后端。`TRUSTED_PROXIES` 要包含可信代理网段。

### TLS 终止检查清单

发布前确认：

- [ ] `CASDOOR_ISSUER=https://sso.stuhelper.com`
- [ ] `CASDOOR_PUBLIC_AUTH_BASE_URL=https://sso.stuhelper.com`
- [ ] `CASDOOR_REDIRECT_URI=https://stuhelper.com/api/v1/auth/callback`
- [ ] `IDENTITY_ISSUER=`
- [ ] `ADMISSION_PUBLIC_BASE_URL=https://join.stuhelper.com`
- [ ] `IDENTITY_REFRESH_TOKEN_TTL` 符合生产离线授权策略（默认 2592000 秒，允许 3600 到 2592000 秒）。
- [ ] `CORS_ORIGINS=https://stuhelper.com,https://join.stuhelper.com,https://sso.stuhelper.com`
- [ ] `WEB_VITE_SSO_URL=https://sso.stuhelper.com`
- [ ] `WEB_VITE_IDENTITY_URL=`
- [ ] `WEB_VITE_WEB_URL=https://stuhelper.com`
- [ ] 主站生产机 `NGINX_PUBLIC_INGRESS_PROFILE=stuhelper ./infra/ops/nginx-public-ingress-preflight.sh` 通过。
- [ ] `https://sso.stuhelper.com/.well-known/openid-configuration` 可达，`https://id.stuhelper.com/.well-known/openid-configuration` 返回 404。
- [ ] 如果公网入口检查失败，已运行 `./infra/ops/nginx-public-ingress-preflight.sh` 或 `NGINX_PUBLIC_INGRESS_PROFILE=sso ./infra/ops/nginx-public-ingress-preflight.sh` 并留档脱敏 evidence。
- [ ] `TOKEN_COOKIE_SECURE=true`（生产必须）。
- [ ] `TOKEN_COOKIE_DOMAIN=.stuhelper.com`（主站与 `join.stuhelper.com` admission 流程复用登录会话）。
- [ ] 宝塔 Nginx 是唯一监听公网 `80/443` 的入口。
- [ ] `127.0.0.1:18080`、`127.0.0.1:18000`、`127.0.0.1:18001` 在宿主机可访问且未暴露公网。
- [ ] 如果使用 External LB/CDN，`TRUSTED_PROXIES` 已正确配置。

### Koishi Admission 生产配置

Koishi/NapCat 不在主站 Compose 中，生产配置仍必须可复现。Koishi 的 Compose service 应使用 `env_file: ./.env` 或等价机制，至少注入：

```env
STUHELPER_PLATFORM_BASE_URL=https://stuhelper.com
STUHELPER_PLATFORM_SERVICE_TOKEN=<redacted>
STUHELPER_FRESHMAN_MATERIAL_HOSTS=stuhelper.com,join.stuhelper.com
```

`stuhelper-group-guard` 的 admission MVP 配置应显式限制职责边界：

```yaml
stuhelper-group-guard:admission:
  platform:
    baseUrl: ${{ env.STUHELPER_PLATFORM_BASE_URL }}
    serviceToken: ${{ env.STUHELPER_PLATFORM_SERVICE_TOKEN }}
  guard:
    targetGroups:
      - '178037297'
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

这表示新插件只接入 admission 入群后验证，不抢“举报 / 骰子 / 抽禁言”等生产既有命令，不接管消息风控监听，也不扫描新生材料原图转发；但保留“查询入群认证 / 重发认证链接 / 重新生成认证链接”等 admission 管理命令。旧 `student-query` 插件不要整体关闭；如它也对 admission 目标群做同一阶段入群验证，应在旧插件自己的目标群或功能范围里排除 `178037297`。

如果 Koishi 日志出现 `TypeError: Invalid URL`，并且错误里 `base` 是原样的 `${{ env.STUHELPER_PLATFORM_BASE_URL }}`，说明当前运行时没有把 Koishi 配置占位符插值成真实环境变量。不要在生产 `node_modules` 里手改源码；应确认 Compose 已通过 `.env` 或 `env_file` 注入 `STUHELPER_PLATFORM_BASE_URL` 和 `STUHELPER_PLATFORM_SERVICE_TOKEN`，然后从本地仓库构建并部署包含 `@stuhelper/koishi-shared` 平台配置环境变量回退逻辑的 StuHelper 插件包。

如果当前 Koishi 生产仍通过宝塔 Compose 挂载本机 `node_modules` 运行 StuHelper 插件，插件发布包必须来自本地仓库当前构建产物，不允许在生产容器内或生产 `node_modules` 中直接改源码作为最终状态：

```bash
cd bots/koishi
corepack yarn build
cd ../..
./infra/ops/package-koishi-stuhelper-packages.sh /tmp/stuhelper-koishi-packages.tar.gz
sha256sum /tmp/stuhelper-koishi-packages.tar.gz
```

上传 `/tmp/stuhelper-koishi-packages.tar.gz` 到 Koishi 生产 Compose 目录后，生产侧先校验 sha256，再备份现有三个包目录和 `koishi.yml`，最后在 Compose 目录根部解包。归档内部路径已经固定为 `koishi/node_modules/...`，所以解包目标必须是 Koishi Compose 目录，不是 `koishi/node_modules`：

```bash
sha256sum stuhelper-koishi-packages.tar.gz
backup_dir="backups/admission-$(date -u +%Y%m%dT%H%M%SZ)"
mkdir -p "${backup_dir}"
cp -a koishi.yml koishi/node_modules/@stuhelper/koishi-shared koishi/node_modules/@stuhelper/koishi-moderation-core koishi/node_modules/koishi-plugin-stuhelper-binding koishi/node_modules/koishi-plugin-stuhelper-group-guard "${backup_dir}/"
tar -xzf stuhelper-koishi-packages.tar.gz
docker compose restart koishi
```

这个 tar 只包含 `package.json` 和 `lib/`，不包含源码树、`node_modules`、环境变量文件、SSH 辅助脚本或 secret。临时 SSH 命令和上传脚本可以放在本机未跟踪路径，但不得提交到 git。

生产 `student-query.enableGroupVerify` 保持 `true`；admission 上线不能靠关闭旧插件本身来规避冲突，冲突应通过新插件的 `commands.enabled=false`、`moderation.enabled=false`，以及旧插件自己的目标群/功能范围收敛处理。

Koishi 重启后用仓库脚本采集 admission 生产证据，不要临时拼带 token 的命令：

```bash
KOISHI_COMPOSE_DIR=/www/server/panel/data/compose/koishi-napcat \
KOISHI_ADMISSION_BOT_SELF_ID=<botSelfID> \
./infra/ops/koishi-admission-production-evidence.sh
```

该脚本检查 `stuhelper-group-guard:admission` 配置目标群 `178037297`、`commands.enabled=false`、`moderation.enabled=false`、`freshmanForward.enabled=false`、`student-query.enableGroupVerify=true`、Koishi 容器环境、bot admission API 200，以及最近日志中没有 `duplicate command names: 举报`、`admission 401/unauthorized`、`B0000001`、`pending-forward` 循环报错。ChatLuna API key invalid 之类日志不属于 admission 上线验收范围。
默认日志窗口是最近 2 小时，避免上午救火前的历史错误影响当前验收；需要回溯时可显式设置 `KOISHI_ADMISSION_LOG_SINCE=24h`。长期运行的 Koishi 容器可能已经没有启动加载日志，脚本默认不要求加载日志必须出现在当前窗口；如果刚重启后要强制检查加载日志，可设置 `KOISHI_ADMISSION_REQUIRE_LOAD_LOG=true`。

也可以在 Koishi 节点用聚合入口采集同一份证据：

```bash
ADMISSION_MVP_PRODUCTION_EVIDENCE_MODE=koishi \
./infra/ops/admission-mvp-production-evidence.sh
```

真实 QQ 小号入群 E2E 分两段留证。第一段在小号入群后立即执行，证明真实入群事件已经让 Koishi/后端创建 `https://join.stuhelper.com/verify/<token>?qq=<qq>` canonical session：

如果 `group_admission_sessions` 仍为 0，说明还没有真实 QQ 入群事件进入 admission 流程；这不等同于主站、SSO 或 Koishi 配置损坏。先确认小号不是已经在群内、目标群是 `178037297`、NapCat/Koishi 在线，再让小号实际申请/进入目标群。脚本缺少 `ADMISSION_E2E_QQ_ID` 时会直接失败，因为没有真实 QQ 号就无法证明端到端闭环。

实际测试时可以先启动等待脚本，再让小号申请入群；脚本会循环调用只读 evidence 检查，直到该阶段通过或超时：

```bash
ADMISSION_E2E_QQ_ID=<small-account-qq> \
ADMISSION_E2E_GUILD_ID=178037297 \
ADMISSION_E2E_EXPECTED_STAGE=join-created \
ADMISSION_E2E_WAIT_TIMEOUT_SECONDS=900 \
./infra/ops/admission-join-e2e-wait.sh
```

```bash
ADMISSION_E2E_QQ_ID=<small-account-qq> \
ADMISSION_E2E_GUILD_ID=178037297 \
ADMISSION_E2E_EXPECTED_STAGE=join-created \
./infra/ops/admission-join-e2e-evidence.sh
```

第二段在用户打开链接经 `sso.stuhelper.com` 登录回到 `join.stuhelper.com`，并完成 QQ 绑定和学生认证或新生材料流程后执行。这个阶段证明业务流程完成，但还不等同于 Koishi 已经解除禁言：

可以继续用等待脚本等待业务流程完成：

```bash
ADMISSION_E2E_QQ_ID=<small-account-qq> \
ADMISSION_E2E_GUILD_ID=178037297 \
ADMISSION_E2E_EXPECTED_STAGE=flow-completed \
ADMISSION_E2E_WAIT_TIMEOUT_SECONDS=900 \
./infra/ops/admission-join-e2e-wait.sh
```

```bash
ADMISSION_E2E_QQ_ID=<small-account-qq> \
ADMISSION_E2E_GUILD_ID=178037297 \
ADMISSION_E2E_EXPECTED_STAGE=flow-completed \
./infra/ops/admission-join-e2e-evidence.sh
```

如果 `flow-completed` 来自新生材料提交，先做只读审核员准入检查。这个步骤会提前暴露“管理员 QQ 没绑定”或“绑定用户没有 `admission:freshman:review` 能力”，避免到最后才发现 Koishi 无法审核并解除禁言：

```bash
ADMISSION_REVIEWER_READINESS_APPLICATION_ID=<freshman-application-id> \
ADMISSION_REVIEWER_READINESS_OPERATOR_QQ_IDS=<operator-qq-ids> \
ADMISSION_REVIEWER_READINESS_GUILD_ID=178037297 \
make prod-admission-reviewer-readiness
```

第三段等待 Koishi 轮询到 verified session，执行解除禁言，并把成功结果回写给后端：

```bash
ADMISSION_E2E_QQ_ID=<small-account-qq> \
ADMISSION_E2E_GUILD_ID=178037297 \
ADMISSION_E2E_EXPECTED_STAGE=bot-released \
ADMISSION_E2E_WAIT_TIMEOUT_SECONDS=900 \
./infra/ops/admission-join-e2e-wait.sh
```

```bash
ADMISSION_E2E_QQ_ID=<small-account-qq> \
ADMISSION_E2E_GUILD_ID=178037297 \
ADMISSION_E2E_EXPECTED_STAGE=bot-released \
./infra/ops/admission-join-e2e-evidence.sh
```

最终聚合验收使用同一个真实 QQ 小号，并要求 bot release evidence 在默认 180 分钟新鲜度窗口内：

```bash
ADMISSION_E2E_QQ_ID=<small-account-qq> \
make prod-admission-mvp-final-evidence
```

Koishi 生产节点必须单独留一份 final evidence，证明该节点当前配置、环境、bot API 和日志窗口都满足 admission MVP 要求：

```bash
make prod-admission-mvp-final-koishi-evidence
```

三份 final evidence 聚齐后，再运行机器校验：主站聚合 evidence、`infra/generated/admission-join-e2e-evidence.json`、Koishi evidence。该步骤会拒绝过期 evidence、skipped evidence、只跑主站未跑 Koishi、只跑 Koishi 未跑真实 QQ E2E、或 join E2E 子证据缺少 active student verification credential 的情况：

```bash
make prod-admission-mvp-final-verify
```

`flow-completed` evidence 必须显示 session 已绑定用户、token 已消费、QQ 绑定存在，并且存在有效学生认证 credential 或新生材料记录。`bot-released` evidence 必须显示后端已经记录 successful bot release 和 session cancelled marker，且 `latest session is fresh enough for this E2E run`、`release requires active student verification credential`、`bot release evidence is fresh enough for this E2E run` 通过，证明用户已有 active student verification credential，Koishi 已执行并上报当前这次 release action。脚本不输出 raw token 或 `token_hash`，只输出脱敏 URL 形状、状态和计数。
