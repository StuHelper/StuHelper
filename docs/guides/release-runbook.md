---
type: guide
audience: ops
status: current
authoritative-source: infra/ops/*.sh + infra/ansible/
last-verified: 2026-08-02
---

# 发布运行手册

## 适用范围

- GitHub Actions、GHCR、受保护 environment 的发布与回滚。
- 手工 SSH 到部署机执行的应急发布。

首次生产落地先按 [production-go-live.md](production-go-live.md) 完成域名、宝塔 Nginx、secret backend、对象存储、备份与告警准备；本文只描述发布与回滚运行流程。

## 发布前检查

- [ ] 本次变更已通过 CI（web / admin / backend / koishi）。
- [ ] admission MVP 相关变更已在本地执行 `make check-admission-mvp`；该入口覆盖 admission 后端、auth/user 依赖后端、Web admission 和用户认证 Vitest、认证/admission 与用户中心 Playwright、Web build、Koishi group guard、生产入口和 evidence 契约。
- [ ] 涉及运维脚本、部署配置、生产 evidence、Nginx preflight 或 CI 漂移门禁时，本地已执行 `make check-infra-contracts`；该入口同时覆盖 `infra/ops/tests/*.sh` 和 `infra/ops/tests/*.mjs`。
- [ ] GitHub Actions `Runtime image security` 已通过并保留本次 JSON evidence；`infra/security/runtime-images.json` 中没有过期的 pin review、漏洞例外或 VEX，生产 `.env.prod.shared` 中的基础设施镜像引用与已扫描策略一致。
- [ ] production `Deploy` 作业已通过受保护 GitHub environment 的人工审批。
- [ ] 如果包含数据库变更，已完成备份；`prod-deploy.sh` 在迁移前执行 `backup-postgres.sh`。
- [ ] 生产机上的逻辑备份 / base backup / backup sync timer 已启用。
- [ ] 承载 `postgres_data` / `redis_data` 的宿主机块设备已启用静态加密；外部 S3 已启用服务端加密、版本/保留和生命周期策略。
- [ ] 远端部署控制面已核对：`.deploy/remote.env`。
- [ ] 共享配置已核对：`.env.prod.shared`。
- [ ] secrets 已核对：`.env.prod.secrets`（本地演练可用 `.env.prod.secrets.local`）；运行时派生 secrets 必须通过 `GENERATED_ENV_SECRET_REF` 写入远端 secret backend，`.env.prod.generated.secrets` 仅保留空占位。
- [ ] secret backend 已核对：`.deploy/remote.env` 中的 `SECRET_BACKEND` / `*_SECRET_REF` / `GENERATED_ENV_SECRET_REF` / `VAULT_ADDR` / `VAULT_TOKEN_FILE`。
- [ ] 关键变量已核对：内置 PostgreSQL 模式的 `POSTGRES_PASSWORD`、所有模式的 `POSTGRES_EXPORTER_DB_PASSWORD`、`REDIS_PASSWORD`、`REDIS_EXPORTER_PASSWORD`、`TAG`、`OBJECT_STORAGE_*`、`BACKUP_OBJECT_STORAGE_*`、`ADMISSION_PUBLIC_BASE_URL=https://join.stuhelper.com`、`WEB_VITE_SSO_URL=https://sso.stuhelper.com`、`WEB_VITE_WEB_URL=https://stuhelper.com`。外部 PostgreSQL 模式不生成、不保存也不要求 StuHelper 自建数据库的超级用户密码。`POSTGRES_EXPORTER_DB_PASSWORD` 必须是 PostgreSQL `stuhelper_metrics` 专用值，不与应用、备份或超级用户复用；`REDIS_EXPORTER_PASSWORD` 必须是 Redis `stuhelper_metrics` 专用值，不与 `REDIS_PASSWORD` 复用。生产对象存储与备份端点必须为 HTTPS；公共 CA 场景下 `OBJECT_STORAGE_TLS_CA` / `OBJECT_STORAGE_TLS_CA_HOST_PATH` 都留空。私有 CA 场景下前者固定为 `/object-storage-tls/ca.crt`，后者指向宿主机可读 PEM CA bundle；`BACKUP_OBJECT_STORAGE_TLS_CA` 独立使用宿主机可读路径。
- [ ] admission 最小生产数据已通过 `./infra/ops/import-school-directory.sh` 和 `./infra/ops/admission-bootstrap-production-data.sh` 幂等准备：学校目录包含 `school_code=4111010006` 的北京航空航天大学，当前只启用该校，公开学生认证/admission 表单以 `schoolCode=4111010006` 为主识别字段，邮箱域仅 `buaa.edu.cn`，`platform=qq` 的 `178037297` 策略存在，`auto_approve_verified_join=true`、`auto_approve_unverified_join=true`、`forward_raw_material_to_qq=false`。
- [ ] Koishi/NapCat 独立节点已确认：Koishi service 使用 `env_file` 或等价机制注入 `STUHELPER_PLATFORM_BASE_URL=https://stuhelper.com`、`STUHELPER_PLATFORM_SERVICE_TOKEN`、`STUHELPER_FRESHMAN_MATERIAL_HOSTS=stuhelper.com,join.stuhelper.com`；真实 token 不写入仓库或 runbook。
- [ ] 生产 PostgreSQL TLS 已核对：默认 `POSTGRES_ENABLE_SSL=on`、`POSTGRES_INTERNAL_SSL_MODE=verify-full`（最低必须为 `verify-ca`）、`DB_SSL_MODE=verify-full`、`DB_SSL_ROOT_CERT=/tls/ca.crt`，且 `DATABASE_URL` / `BACKUP_DATABASE_URL` / `REPLICATION_DATABASE_URL` 都包含 `sslmode=verify-full&sslrootcert=/tls/ca.crt`。若生产机复用宝塔已有 PostgreSQL，必须设置 `EXTERNAL_POSTGRES_ENABLED=true`、`EXTERNAL_DATASTORE_NETWORK=baota_net` 和 `POSTGRES_CLIENT_CA_HOST_PATH`，并先为 StuHelper / OpenFGA 创建独立数据库和独立账号、完成数据迁移；外部实例没有可验证 TLS 时不得上线。`EXTERNAL_POSTGRES_ALLOW_PLAINTEXT` 只允许本地 `prod-parity`，生产必须为 `false`。本地 `render-postgres-tls.sh` 不会为外部数据库生成伪 CA。外部数据库管理员还须预置仅有 `pg_monitor`、连接数上限 5、只能连接 `postgres` 维护库的 `stuhelper_metrics`；部署后的严格观测 smoke 必须显示 `up{job="postgres-exporter"}=1` 和 `pg_up=1`。Redis 不复用全局实例，仍由 StuHelper Compose 以独立实例运行。
- [ ] Open Platform runtime token 探针已核对：`OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_REQUIRED=true`、`OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND=/app/casdoor-runtime-token-probe-runner.mjs` 且不是 `REPLACE_WITH_OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_COMMAND` 占位符；`OPEN_PLATFORM_PRODUCTION_EVIDENCE_ALLOW_LOCAL_TARGETS=false`；专用低权限 `CASDOOR_TOKEN_PROBE_USERNAME` / `CASDOOR_TOKEN_PROBE_PASSWORD` 已通过 secret backend 注入；`CASDOOR_TOKEN_PROBE_SMOKE_*` 专用 smoke app 已配置，发布时会通过 `open-platform-production-evidence.sh` 自动运行 `casdoor-runtime-token-probe-smoke.sh`，且聚合 evidence 会在子 smoke 前验证强制探针门禁开启并默认拒绝 localhost Casdoor/OpenFGA 目标。
- [ ] OpenFGA 派生配置已核对：`OPENFGA_STORE_ID` / `OPENFGA_MODEL_ID` 由 `bootstrap-platform.sh` 生成，`OPENFGA_RESOURCE_SMOKE_MODE=container`，发布时会通过 `open-platform-production-evidence.sh` 自动运行 `openfga-resource-access-smoke.sh`。container 模式直接执行 `BACKEND_IMAGE_REF` 内构建期固化的 `/app/openfga-resource-smoke`，不在生产节点下载 Go 模块或现场编译。
- [ ] Casdoor token/session lease 与撤销契约已核对：发布流程中的 `bootstrap-platform.sh prod` 必须成功把托管 Web/Admin/UniApp application 收敛到 `ExpireInHours=1`，且运行时 `TOKEN_REFRESH_TTL` 不得小于 provider access-token 寿命（默认 7 天，大于 1 小时）。新登录/refresh 会用已验证 `exp` 再次拒绝“access 剩余寿命大于 session lease”或超过 30 天 hard cap 的漂移；不要通过提高 hard cap 或向下截断绕过。滚动升级期间旧 session 没有 `accessTokenExpiresAt`，其 logout-all 仅在上述约束成立时使用真实 Redis PTTL 作为保守黑名单 TTL。`TOKEN_ACCESS_TTL=300` 只是 cookie/`expiresIn` 策略，不是 Casdoor token 自然失效时间。生产 discovery 还必须满足：没有 `revocation_endpoint` 时，`end_session_endpoint` 精确为同 issuer 的 `/api/logout`；受控测试账号执行 login → logout 与 login → logout-all 后，旧 access introspection 均为 inactive、旧 refresh grant 均返回 `invalid_grant`，且应用日志没有 Casdoor `status=error`、畸形 JSON 或 provider revoke partial failure。滚动升级旧 session 的 logout-all 还要留证“refresh rotation 后替代 family 立即失效”；不能只以 HTTP 200、客户端 cookie 清除或本地 Redis session 删除代替 provider 验收。
- [ ] 公网 SSO 和入群验证入口门禁已核对：默认 `PUBLIC_INGRESS_CONFIG_PREFLIGHT_ENABLED=true`，远端 preflight / prod deploy 会先审计本机宝塔 Nginx 主站和 join 配置；默认 `PUBLIC_INGRESS_PREFLIGHT_ENABLED=true`，随后验证 `stuhelper.com`、`join.stuhelper.com`、`sso.stuhelper.com` 公共 DNS-over-HTTPS 有公网 A/AAAA 且 TLS 可达。`sso.stuhelper.com` discovery/JWKS/authorize 路由必须反代到 Casdoor。旧 identity smoke 已删除；默认 `ADMISSION_PUBLIC_SMOKE_ENABLED=true`，远端 preflight / prod deploy 都会运行 `admission-public-smoke.sh` 并写入 `infra/generated/admission-public-smoke-evidence.json`，要求 `join.stuhelper.com/verify/<probe>` 由 Web SPA 承载，`join.stuhelper.com/api/v1/metrics/vitals` 和 `/api/v1/metrics/frontend-errors` 接受同源 beacon 并返回 204，`join.stuhelper.com/api/v1/admission/freshman/camera-handoffs/<probe>/events` 无登录探测返回 401 且 `X-Accel-Buffering: no`，`join.stuhelper.com/`、`join.stuhelper.com/developers/apps`、`join.stuhelper.com/verify`、主站 `/verify`、主站 `/verify/*` 全部返回 404，且 `ADMISSION_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS=false`、`ADMISSION_PUBLIC_SMOKE_CURL_INSECURE=false`。默认 `PUBLIC_WEB_AUTH_BROWSER_SMOKE_ENABLED=true`，prod deploy 会运行 `public-web-auth-browser-smoke.mjs`，用 Playwright 真浏览器验证主站登录页非空、开发者入口未登录跳回 `/login?redirect=/developers/apps`、点击登录进入 `sso.stuhelper.com/login/oauth/authorize` 并看到账号密码登录和 `/signup/oauth/authorize` 注册入口、点击注册进入 `sso.stuhelper.com/signup/oauth/authorize` 并看到账号密码注册表单、join 根路径/主站业务路径不渲染主站内容、join verify SPA 可加载、join 手机拍照页允许 camera。
- [ ] 观测配置已核对：`METRICS_PASSWORD`、`POSTGRES_EXPORTER_DB_PASSWORD`、`REDIS_EXPORTER_PASSWORD`、`GRAFANA_ADMIN_PASSWORD`、`OTEL_ENABLED=true`；严格 smoke 中 PostgreSQL/Redis exporter 的 `up` 与 `pg_up`/`redis_up`、node-exporter 和 cAdvisor 均为 1。
- [ ] staging 已验证通过（如有 staging）。
- [ ] 发布 bundle 从干净 Git 工作区打包；`git status --short` 为空，所有待发布改动已经提交并签名。

## GitHub 自动发布链路

### 构建与发布镜像

1. Pull Request、`develop` 或 `main` push 运行按路径选择的 `CI`。
2. `CI / Required` 汇总后端、契约、客户端、E2E、Koishi、Infra、依赖和安全扫描。
3. 只有受信任的 `develop` / `main` push、聚合门禁与两项 CodeQL 成功，且提交仍是实时 branch head，
   才调用 `Publish images`。
4. backend / frontend / admin 使用完整 commit SHA 构建并在本地接受 Trivy 扫描。
5. 同一候选镜像推送到 GHCR，记录 manifest digest，并签发 provenance 与 CycloneDX SBOM
   attestation；SBOM JSON 作为 Actions evidence 保留 30 天。

### staging / production

1. `main` CI 成功发布三个镜像后，如果仓库变量 `STAGING_AUTO_DEPLOY_ENABLED=true`，自动调用
   `Deploy` 把同一 SHA 晋级到 staging；未启用时可手工运行 `Deploy`。
2. 如果同时设置 `PRODUCTION_AUTO_PROMOTION_ENABLED=true`，staging 成功后自动创建同 SHA
   production deployment；工作流会停在 protected environment，直到 `Xauryan` 审批，批准后才部署。
3. 手工运行时选择 workflow ref、`staging` 或 `production`，并填写变更/事故原因。Forward Deploy
   不再接受 commit SHA；候选固定为当前 workflow ref 的 `github.sha`。
4. environment secrets 暴露和 production 审批之前，工作流先验证目标仍是当前 branch head、
   `Required`、Go/JavaScript-TypeScript CodeQL、三个 GHCR digest 的来源与 provenance。
5. production 默认还要求同一 SHA 的最新 staging deployment 为 success。事故中确需绕过时，手工
   运行 `Deploy` 并显式
   选择 `skip_staging_gate=true`、填写至少 24 字符上下文，并留下 production approval 审计。
6. 验证完成后 environment 再校验允许分支、审批者和 secrets；production 当前唯一 reviewer 为
   `Xauryan` 且允许该用户自批。审批后、任何 SSH 前再次校验实时 branch head、checks 和 staging
   gate，等待期间已经过期的候选会失败关闭。
7. 远端实际执行 `./infra/ops/remote-preflight.sh` 和
   `./infra/ops/remote-prod-deploy.sh`。
8. 自动运行业务与严格可观测性 smoke；任何 smoke 失败，本次发布即失败并进入回滚判断。

只要 Smoke Check 失败，本次发布就视为失败，需要立刻进入回滚判断。
不要把失败处理改成无条件自动回滚：migration 已成功时，旧应用可能不兼容新 schema。先核对备份、
迁移兼容性和当前 release evidence，再启动有原因、有 environment 审批的 `Rollback`。

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

### 本机 Vault 控制面与稳定状态目录

`make prod-vault-init` 是维护者工作站上的本地 Vault 辅助入口，不改变下文生产目标机自持
`${DEPLOY_APP_DIR}/.deploy` / `${DEPLOY_APP_DIR}/.secrets` 的规则。未显式覆盖路径时，本机入口把
可持久状态放在操作系统 state 目录，而不是 Git worktree：

| 内容 | Linux 默认路径 |
|------|----------------|
| Vault 配置与 file storage | `${XDG_STATE_HOME:-$HOME/.local/state}/stuhelper/vault/` |
| init / unseal 材料 | `${XDG_STATE_HOME:-$HOME/.local/state}/stuhelper/vault-credentials/` |
| Vault token | `${XDG_STATE_HOME:-$HOME/.local/state}/stuhelper/secrets/vault/token` |
| 本机 remote deploy config | `${XDG_STATE_HOME:-$HOME/.local/state}/stuhelper/deploy/remote.env` |

macOS 使用 `$HOME/Library/Application Support/StuHelper/`。这些目录必须保持 `0700`，凭据与配置
文件保持 `0600`。仓库内 `.deploy/remote.env` 只是在默认本机场景下指向稳定配置的兼容符号链接；
Vault 容器的 bind mount 不得指向该链接或 worktree 内的 `.deploy/vault`。删除、移动或重新克隆
仓库不能改变正在运行的 Vault 数据源。未启用 file audit sink 的 `/vault/logs` 使用受限的 16 MiB
tmpfs，避免每次受控重建遗留新的匿名 Docker volume；正式审计日志应输出到明确配置的持久审计
后端，而不是依赖镜像声明的匿名 volume。

可用 `LOCAL_VAULT_PRINT_PATHS_ONLY=true ./infra/ops/init-local-vault-secret-backend.sh` 只打印将要
使用的路径，不启动或修改容器。`LOCAL_VAULT_STATE_DIR`、`LOCAL_VAULT_CREDENTIALS_DIR`、
`SECRET_FILE_ROOT`、`VAULT_TOKEN_FILE`、`REMOTE_DEPLOY_CONFIG_FILE` 可用于受控覆盖；显式传入
`DEPLOY_STATE_DIR` 时仍使用该隔离目录，供测试或专用环境使用。

脚本会核对现有 `stuhelper-vault` 的配置/data bind source、镜像、仅本机端口和 restart policy。
任何漂移都默认失败关闭，不会启动、重启或自动初始化空 Vault。只有在旧 Vault API 可访问、目标
配置和非空数据已经按字节核验、unseal/token 材料已经安全落盘后，运维人员才能显式设置
`LOCAL_VAULT_RECREATE_CONTAINER=true` 做一次容器重建。恢复隔离目录不是长期运行路径；迁移成功后
同时设置 `LOCAL_VAULT_REQUIRE_EXISTING_DATA=true`，会在任何 KV enable/seed 操作之前要求原 token、
KV mount 和既有 `GENERATED_ENV_SECRET_REF` 全部可读；如果 staged storage 报告为 uninitialized，
脚本会在 `vault operator init` 和任何凭据文件覆盖之前失败。恢复隔离目录应保留为只读恢复锚点，
直到完成重启、解封和已知 KV 读取验证；不能让 Docker 继续依赖已移动目录的存活 inode。

### 宝塔 Compose 源码包应急发布

如果当前生产机不能直接从 registry 拉取镜像，且宝塔 Compose 实际运行目录采用“Compose 根目录 + `source/` 源码副本”的形态，可以执行一次源码包 + 本地镜像 tar 的应急发布。该流程只接受由本地当前仓库生成的产物；不得在生产 `source/`、容器文件系统或 `node_modules` 中手工改代码作为最终状态。

要求：本地打包前记录当前 Git ref、`git status --short` 和源码包 `sha256sum`。源码包不得包含真实 `.env*`、`.deploy/`、`node_modules`、`dist`、临时 SSH 脚本或本地 secret；`infra/ops/build-deploy-bundle.sh` 会先拒绝脏工作树，再只打包 Git `HEAD` 中的跟踪文件，并在发布前断言根目录恰好包含 `.env.example`、`.env.prod.example` 两个 env 模板、没有其他根 env 文件。生成后仍需用 `tar -tzf` 抽查。生产 `source/.env.prod.shared`、`source/.env.prod.secrets.local`、`source/.env.prod.generated`、`source/.env.prod.generated.secrets` 只从旧生产目录保留或由 secret backend 重新生成，不从源码包覆盖。

如果旧生产目录已有 `source/infra/generated`，恢复到新 `source/infra/generated` 时复制整个 `generated` 目录本身，目标必须是 `source/infra/generated`，不能变成 `source/infra/generated/generated`。生产仅在仓库内生成 PostgreSQL / Redis TLS；应分别执行 `source/infra/ops/render-postgres-tls.sh`、`source/infra/ops/render-redis-tls.sh`、`source/infra/ops/render-redis-acl.sh` 和 `source/infra/ops/prepare-datastore-client-cas.sh`。PostgreSQL 容器启动时会把只读服务端源目录中的证书和私钥复制到容器内 `/tls` tmpfs，把 `server.key` 设为 PostgreSQL UID/GID 70、模式 0600 后再降权启动；Redis 同样把服务端 key 与仅含密码哈希的 ACL 复制到 UID/GID 999:1000、模式 0600 的 `/redis-runtime` tmpfs。宿主私钥和 ACL 不得放宽为 group/world-readable。应用、迁移、OpenFGA、exporter 以及备份/恢复工具只挂载分别含一个公开 `ca.crt` 的 `postgres-client-ca` / `redis-client-ca`，不得挂载服务端源目录；`postgres-client` 还必须保持非 root、只读根文件系统、无 Linux capability，并且不得挂载 PostgreSQL 数据卷。外部 S3 使用公共 CA 时不恢复对象存储 CA 目录；使用私有 CA 时把经运维核验的 CA bundle 放在 `OBJECT_STORAGE_TLS_CA_HOST_PATH` 指向的位置，再运行 `source/infra/ops/prepare-object-storage-client-ca.sh`。不要把本地 SeaweedFS 的 `s3.json`、`ca.key` 或 `private.key` 恢复到生产应用挂载目录；应用只挂载独立的 `infra/generated/object-storage-client-ca`。宝塔源码目录替换后还必须执行 `source/infra/ops/ensure-baota-runtime-permissions.sh --apply`，它会归一化数据库 TLS/ACL 源文件、客户端 CA 和对象存储客户端 CA 权限，并在同机存在独立 Casdoor Compose 时修复 `conf/app.conf` 与 `logs/` 的 UID 1000 权限。镜像必须在本地从当前代码构建，上传 tar 后在生产执行 `sha256sum -c`，再 `docker load`；记录 backend / web / admin 镜像 ID 和 tar sha256。数据库 bootstrap、migration、readiness、public smoke 仍运行仓库脚本。`admission-bootstrap-production-data.sh` 和 `admission-production-readiness.sh` 在宝塔 `source/` 目录下会自动识别 `.env.prod.shared`、`.env.prod.secrets.local`、`.env.prod.generated`、`.env.prod.generated.secrets`。

重建容器必须用宝塔实际 Compose 根目录和实际 env file，例如：

```bash
cd /www/server/panel/data/compose/stuhelper

# 如果宝塔根 docker-compose.yml 是生成产物，里面的 image: 行可能已经固化旧 tag。
# 先从 source/.env.prod.shared 中的非敏感 *_IMAGE_REF 刷新根 compose，并保留备份。
./source/infra/ops/baota-compose-refresh-image-refs.sh --apply
./source/infra/ops/ensure-baota-runtime-permissions.sh --apply

docker compose \
  --env-file source/.env.prod.shared \
  --env-file source/.env.prod.secrets.local \
  --env-file source/.env.prod.generated \
  --env-file source/.env.prod.generated.secrets \
  up -d --no-deps --force-recreate app frontend admin
```

`baota-compose-refresh-image-refs.sh` 默认 dry-run，只会读取 `source/.env.prod.shared` 中的 `BACKEND_IMAGE_REF`、`FRONTEND_IMAGE_REF`、`ADMIN_IMAGE_REF`，并更新根 compose 的 `migrate` / `app` / `frontend` / `admin` 镜像行。生产不应手工只改根 compose 后结束；同一个 image ref 必须同时存在于 `source/.env.prod.shared`，并记录镜像 ID、tar sha256 和根 compose 备份路径。

不要用 shell `source source/.env.prod.shared` 作为 Compose 重建方式。生产共享 env 允许出现带空格的非敏感值，例如中文发件人别名；仓库脚本通过 `infra/ops/lib/common.sh` 解析 env，Compose 重建应使用上面的 `docker compose --env-file ...` 形式，避免 shell 把值拆成命令。

如果这一路径中发现必须修改生产 Nginx、env 或 DB 数据，修改后的非敏感结构必须同步回仓库模板、脚本或本文档；真实 secret 只保留在生产 secret backend / env 文件中。

### user_hash 回填运维任务

后端启动日志如果出现 `users with missing user_hash detected; run backfill ops task`，不要手工拼 HMAC 或直接改表。该检查是有意保留的启动提醒；实际修复应通过仓库脚本调用后端镜像内的运维命令，复用生产容器已有 `DATABASE_URL` 和 `HMAC_SECRET`，并且不输出 secret。

先 dry-run 确认缺失数量，再显式 apply：

```bash
./infra/ops/backfill-user-hashes.sh --dry-run
./infra/ops/backfill-user-hashes.sh --apply
./infra/ops/backfill-user-hashes.sh --dry-run
```

预期最后一次 dry-run 输出 `remaining=0`。如果生产 backend 容器名不是默认的 `stuhelper-prod-app`，通过 `USER_HASH_BACKFILL_APP_CONTAINER=<container>` 指定。该任务只填充 `users.user_hash IS NULL` 的记录，可重复执行；不修改已有 `user_hash`。

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
- 公网 SSO 入口：`./infra/ops/sso-public-smoke.sh`，留档 `infra/generated/sso-public-smoke-evidence.json`。该检查会断言 `admin/stuhelper-web` 的公开 Casdoor application 元数据仍启用密码登录、注册开关和必填密码注册项，避免生产漂移成只剩 Face ID；同时断言 `https://sso.stuhelper.com/.well-known/openid-configuration` 的 `issuer`、authorize、token 和 JWKS endpoint 都使用公开 HTTPS origin，并把每个请求的 `remoteIP` 写入 evidence。生产模式会拒绝本机、私网、链路本地和保留网段解析；如果运维机 hosts、代理或 `SSO_PUBLIC_SMOKE_RESOLVE_IP` 把 SSO 指到非公网地址，应修正解析或显式指定公网 Edge/source IP 后复跑。若 Casdoor 独立宝塔 Compose 的 `conf/app.conf` 中 `origin = https://sso.stuhelper.com` 已正确但 discovery 仍返回 `http://sso.stuhelper.com`，说明容器尚未重启并仍使用旧内存配置；应通过该 Casdoor Compose 项目重启 `casdoor` 容器，然后重新运行本 smoke，不要只手工修改配置文件后结束。
- StuHelper Admin 最高管理员初始化 / 恢复：不再使用 `STUHELPER_INITIAL_SUPER_ADMINS` 或 `authorization-bootstrap-super-admin.sh`，也不要求两个账号。在 Casdoor 的目标 `stuhelper` organization 中把预期用户（默认一个）设置为 organization administrator（用户对象 `IsAdmin=true`）；`built-in/admin`、其他 organization 的管理员和普通 Casdoor role membership 不会自动映射。目标用户随后完成一次正常 StuHelper 登录或 refresh，系统会用 `casdoor-org-admin-sync` system actor 在同一 PostgreSQL 事务写 `source=casdoor_org_admin` grant、审计和 OpenFGA outbox。新 grant 在 projection verified/applied 前保持 fail-closed。完成后在 `/authorization/grants` 确认来源为 Casdoor 组织管理员、`activatedAt` 非空且 projection 为 `applied`，再验收 Admin 登录和 step-up mutation。降权也必须在 Casdoor 取消 `IsAdmin`；下一次 login/refresh 或该候选 super-admin 的受保护请求会先提交 DB revoke 围栏。允许撤销最后一名管理员；恢复时重新设置 `IsAdmin=true` 并登录/refresh。发布记录应写明 Casdoor 操作者、目标用户名、原因、grant ID 和投影验证结果，但不得记录 token/secret。
- StuHelper Admin 初始 MFA bootstrap：生产后台对 `super_admin` / `school_admin` 还要求 StuHelper 本地 `user_mfa_enrollment.active=true` 和当前登录 token 的 MFA proof。若登录后台出现 `A0010204`，说明 PostgreSQL 授权 grant 已通过但本地 MFA enrollment 缺失。先确认用户已在 `sso.stuhelper.com/account` 绑定 SMS/App/WebAuthn/TOTP 类 MFA，然后在主站生产节点执行 `STUHELPER_ADMIN_MFA_BOOTSTRAP_USERS=<casdoor-username> ./infra/ops/bootstrap-admin-mfa-enrollment.sh`。脚本会确认 StuHelper `authorization_grants` 中存在 desired `super_admin` 或 `school_admin`、Casdoor 身份仍有效且已有 SMS/App/WebAuthn/TOTP MFA 证据，再幂等 upsert `user_mfa_enrollment` 并写入 `iam.mfa.bootstrap` 审计事件；普通 Casdoor role membership 在整个流程中不参与授权，目标 organization `IsAdmin` 是 `super_admin` 的独立权威。`super_admin` MFA reset 不要求另一名 `super_admin` 复核，但仍必须满足能力/step-up 门禁并落审计；自行 disable 自己 MFA 仍被拒绝。执行后受影响用户必须退出 StuHelper 和 SSO 并重新登录，让新 token 带上 MFA proof；如果随后返回 `A0010205`，按前端 step-up 跳转重新完成一次 SSO MFA。
- 公网入群验证入口：`./infra/ops/admission-public-smoke.sh`，留档 `infra/generated/admission-public-smoke-evidence.json`，同时验证 join 域根路径和主站业务路径返回 404，验证 `/api/v1/metrics/vitals` 和 `/api/v1/metrics/frontend-errors` 同源 beacon 返回 204，并验证手机拍照接力 SSE 入口 `/api/v1/admission/freshman/camera-handoffs/<probe>/events` 无登录返回 401 且禁用 Nginx buffering。
- 公网 Web 登录浏览器链路：`./infra/ops/public-web-auth-browser-smoke.mjs`，留档 `infra/generated/public-web-auth-browser-smoke-evidence.json`。该检查会用真实浏览器确认登录按钮进入 `sso.stuhelper.com/login/oauth/authorize` 后仍有账号密码登录和注册入口，确认主站“注册账号”进入 `sso.stuhelper.com/signup/oauth/authorize` 的账号密码注册表单，并确认 join 根路径/主站业务路径不串站、join verify SPA 可加载、join 手机拍照页允许 camera。生产模式会拒绝 `stuhelper.com`、`join.stuhelper.com` 或 `sso.stuhelper.com` 解析到 loopback；如果运维机 `/etc/hosts`、浏览器代理或 Playwright 运行环境把生产域名指到本地开发环境，先修正解析再生成 evidence。
- 新生材料审核员只读准入：`./infra/ops/admission-reviewer-readiness.sh` 或 `make prod-admission-reviewer-readiness`，留档 `infra/generated/admission-reviewer-readiness.json`。该检查只调用 bot view 接口，不会批准或驳回申请，用来确认至少一个管理群 QQ 已绑定 StuHelper 用户且拥有 `admission:freshman:review` 能力。
- 学校邮箱 OTP 邮件准入：生产应使用 `EMAIL_DRIVER=multi`、`EMAIL_FROM_NAME=StuHelper 系统邮件`、`EMAIL_STUDENT_VERIFICATION_SUBJECT=学生认证验证码`。`./infra/ops/tencent-ses-template-smoke.sh` 留档 `infra/generated/tencent-ses-template-smoke.json`；该检查不发送邮件，只用生产 secret 调腾讯云 SES `GetEmailTemplate`，要求 `EMAIL_TENCENT_TEMPLATE_ID=49779` 且模板状态为已审核，输出不包含 Secret。`EMAIL_RESEND_API_KEY` 只能在 secret env/secret store 中配置；Resend 兜底发送不使用腾讯云模板 ID，但必须复用仓库内已审核的 `infra/email-templates/tencent-ses/stuhelper-school-email-otp.html` 和 `.txt` 渲染 Resend `html`/`text` 字段。真实发送 smoke 使用 `RESEND_EMAIL_SMOKE_TO=<recipient-email> ./infra/ops/resend-email-channel-smoke.sh`，留档 `infra/generated/resend-email-channel-smoke.json`，输出和 evidence 不得包含 API key 或完整收件地址。管理后台“系统配置”中的 `email.delivery_policy` 用于调整 provider 启用状态、优先级、权重和 `priority`/`weighted` 策略；如果腾讯云 `GetSendEmailStatus` 返回 `SendStatus=0`、`DeliverStatus=1`，但收件人仍无法在学校邮箱看到邮件，应按收件侧隔离/可见性故障处理，临时或长期把 Resend 设为 `priority=10`，腾讯云设为 `priority=20`，因为这种“腾讯云已递送成功”的结果不会触发自动兜底。

- 北航学籍数据准入：学校目录导入只写 `schools`，不是白名单；只有 `school_configs.enabled=true` 才进入学生认证和 admission 白名单。生产学籍源使用外部只读 Oracle，secret backend 配置 `EXTERNAL_STUDENT_SOURCE_ENABLED=true`、`EXTERNAL_STUDENT_SOURCE_PROVIDER=oracle`、`EXTERNAL_STUDENT_SOURCE_SCHOOL_CODE=4111010006` 及 Oracle 连接、表、列、连接池和熔断参数。Oracle 必须启用 TCPS，应用固定使用 `verify-full`、默认端口 `2484` 和独立只读 CA 挂载 `/external-student-source-tls/ca.crt`；证书 SAN 必须覆盖配置 host。`EXTERNAL_STUDENT_SOURCE_ORACLE_USERNAME`、DBA provisioning 使用的 `EXTERNAL_STUDENT_SOURCE_ORACLE_READONLY_USERNAME` 与实际 `SESSION_USER` 必须是同一账号。该账号必须与源 schema owner 不同，不得使用 `SYS`、`SYSTEM`、`SYSBACKUP`、`SYSDG`、`SYSKM` 或 `SYSRAC`；只允许直接授予无 `ADMIN OPTION` 的 `CREATE SESSION`，以及 `USR_JWBIZ.T_XS_JBXX` 上无 `GRANT OPTION`、无 `HIERARCHY OPTION` 的 `SELECT`，不得有 role、列级授权或其他系统/对象权限。账号创建或轮换使用 `EXTERNAL_STUDENT_SOURCE_ORACLE_READONLY_PASSWORD=<secret> ./infra/ops/provision-external-student-source-oracle-readonly.sh`。首选执行 `./infra/ops/admission-student-source-go-live.sh`，并要求 readiness summary 为 `buaa_student_source=external_oracle`、`infra/generated/external-student-source-smoke.json` 中 `tlsVerified=true`、`runtimeIdentityMatched=true`、`leastPrivilegeGrantsVerified=true` 且授权计数严格为 0/1/1/0（role/system/object/column），并存在可读记录。身份 evidence 只保留账号哈希前缀，抽样 evidence 只记录学号哈希前缀和匹配布尔值。`external_requests_total{client="oracle_student_directory"}` 和 `circuit_breaker_state{name="external_student_source_oracle_4111010006"}` 必须进入 Prometheus；依赖故障对 User/Admission 返回 503。如果没有可用 Oracle，使用 `BUAA_ACADEMIC_STUDENTS_TSV=/path/to/buaa-students.tsv ./infra/ops/admission-student-source-go-live.sh` 校验并幂等导入本地 `academic.buaa_students`；需要单独离线校验或定位 TSV 格式时运行 `BUAA_ACADEMIC_VALIDATE_ONLY=true BUAA_ACADEMIC_STUDENTS_TSV=/path/to/buaa-students.tsv ./infra/ops/import-buaa-academic-students.sh`，不得清空旧数据或打印学生明细。

- admission MVP 聚合生产证据：`./infra/ops/admission-mvp-production-evidence.sh`，留档 `infra/generated/admission-mvp-production-evidence.json`。主站节点默认聚合 SSO public smoke、admission public smoke、Web auth browser smoke 和 admission DB readiness；如果 `EXTERNAL_STUDENT_SOURCE_ENABLED=true`，聚合入口会在 `ADMISSION_MVP_PRODUCTION_RUN_EXTERNAL_STUDENT_SOURCE_SMOKE=auto` 默认模式下强制运行 `external-student-source-smoke.sh`，确认外部学籍源可连接、可读取；未启用外部源时，本地 fallback 表仍由 readiness 检查非空。Koishi 节点使用 `ADMISSION_MVP_PRODUCTION_EVIDENCE_MODE=koishi` 聚合 Koishi admission evidence。该普通入口允许真实 QQ E2E 被记录为 skipped，只能作为生产 smoke。最终上线验收必须在主站节点使用 `make prod-admission-mvp-final-evidence`，并在 Koishi 节点使用 `make prod-admission-mvp-final-koishi-evidence`。主站 final evidence 等价于显式设置 `ADMISSION_MVP_PRODUCTION_E2E_REQUIRED=true`、`ADMISSION_MVP_PRODUCTION_E2E_WAIT=true`、`ADMISSION_E2E_QQ_ID=<small-account-qq>`、`ADMISSION_MVP_PRODUCTION_E2E_EXPECTED_STAGE=bot-released` 和 `ADMISSION_MVP_PRODUCTION_E2E_MAX_SESSION_AGE_MINUTES=180`。
- 如果主站生产机没有 Node/Playwright，先在有 Playwright 的运维机或 CI 上生成 `infra/generated/public-web-auth-browser-smoke-evidence-current.json`，复制到主站源码目录的同一路径，再运行聚合入口；聚合入口会默认读取该 evidence。需要使用其他文件名时，再设置 `ADMISSION_MVP_PRODUCTION_BROWSER_SMOKE_EVIDENCE_FILE=infra/generated/<evidence>.json`。聚合入口会校验该 evidence 新鲜、目标域名正确、浏览器检查全部通过，以及 `/identity` 直接入口、join 防串站 404、camera permission/media capture 成功。预采集 evidence 时不得设置本地 hosts 把生产域名指向 `127.0.0.1`；脚本会直接失败，避免把本地开发环境误判为生产。
- admission MVP 最终证据校验：`./infra/ops/admission-mvp-final-evidence-verify.sh`，留档 `infra/generated/admission-mvp-final-evidence-verify.json`。该脚本只读取已采集的脱敏 evidence，不访问生产；它要求主站聚合 evidence、join E2E 子证据和 Koishi evidence 都新鲜，主站/Koishi 聚合 evidence 无 failed/skipped，并要求主站包含真实 QQ `bot-released`、Koishi evidence 不包含真实 QQ E2E placeholder、join E2E 子证据显示 token 已消费、QQ 已绑定、存在 active student verification credential、后端记录 bot release 和 cancelled marker，且包含通过的 `release requires active student verification credential` 检查。
- admission 最小数据初始化：`./infra/ops/admission-bootstrap-production-data.sh`
- 公网 SSO 入口诊断：`./infra/ops/sso-public-smoke.sh`，失败时留档 `infra/generated/sso-public-smoke-evidence.json`
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

`postgres-backup-evidence.sh` 会验证本地与从对象存储取回的最近逻辑备份、物理 base backup
均带有效 `.sha256` sidecar且未超过新鲜度阈值，物理压缩包还必须可完整遍历；结果写入
`infra/generated/postgres-backup-evidence.json`。这份 evidence 不替代隔离恢复启动和 WAL
连续性/PITR 演练，未执行后两者时不得宣称生产 PITR 已验收。

`OBS_SMOKE_STRICT=true ./infra/ops/observability-smoke-check.sh` 会写入
`infra/generated/observability-smoke-evidence.json`，证明 Prometheus targets、数据库 exporter
真实连接（`pg_up` / `redis_up`）、blackbox probes 和 Alertmanager receiver 配置可用。

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

### GitHub 手工回滚

优先在 GitHub Actions 运行 `Rollback`，选择 `staging` 或 `production` environment，并输入：

```text
commit_sha=<previous-release-full-40-character-commit-sha>
reason=<incident-or-change-record-and-rollback-rationale>
```

回滚 Job 会：

1. 在 environment secrets/审批之前，校验当前 rollback controller 仍是实时 branch head，且其
   `Required` 与双语言 CodeQL 都成功
2. 校验目标环境、完整历史 commit SHA 和回滚原因，把 GHCR 中三个 `<full-sha>` tag 解析成
   digest 并验证发布 provenance
3. environment 审批后校验 SSH endpoint，上传当前可信 controller bundle，并只传递已校验的历史
   digest 引用
4. 读取目标机 `.deploy/remote.env`，用当前 `prod-rollback.sh` / `prod-deploy.sh` 执行应用镜像回滚
5. 自动再次运行业务与严格可观测性 smoke checks

GitHub `Rollback` 不接受 `latest`、短 SHA 或任意业务 tag。仓库本地的 `make prod-rollback`
仍可按上节规则读取上一条成功发布记录，二者不要混淆。

回滚 workflow 不 checkout 或执行历史发布中的 workflow/运维脚本。上传目标机的是当前
`job.workflow_sha` 的完整、干净 controller bundle；历史 SHA 只用于解析 backend/frontend/admin
镜像 digest。这样即使目标版本早于当前安全修复，也不会让旧控制代码重新获得 environment
secrets。当前 controller 与历史应用镜像仍可能存在兼容边界，因此 production rollback 必须保留
人工原因、审批和 migration expand/contract 判断。

正常部署和回滚默认都按执行当天硬校验 runtime-image review window。只有当前日期校验失败时，
回滚才会尝试以下窄范围例外，而且条件必须全部成立：

1. 目标环境存在 `.deploy/releases/<target-tag>.env`，证明该版本曾在**同一环境**成功部署；
2. `TAG`、`ROLLBACK_TAG` 和发布记录完全相同；
3. backend、frontend、admin 三个引用都是 digest，且与成功发布记录逐字相同；
4. 目标提交的完整 runtime-image policy 在原 `DEPLOYED_AT` 日期确实有效，当前生产基础镜像
   仍与该 policy 完全一致；
5. 操作人和 12–500 字符的回滚原因已明确提供。

满足这些条件时，脚本只复用该版本原成功部署日期的审核窗口，并把授权尝试写到权限为 0600 的
`.deploy/rollback-review-exceptions.jsonl`，其中包括 audit ID、操作人、原因、目标 tag、
policy SHA256 和三个应用镜像 digest。镜像漂移、没有同环境成功记录、policy 结构错误或缺少
审计上下文仍会 fail-closed；该例外不会放宽普通 `prod-deploy`。

本地在审核窗口已经过期时回滚，需显式提供审计上下文：

```bash
export ROLLBACK_TAG=<previous-stable-tag>
export ROLLBACK_REVIEW_ACTOR=<operator-or-change-owner>
export ROLLBACK_REVIEW_REASON='关联事故或变更单，并说明为何必须回到该成功版本'
make prod-rollback
```

GitHub `Rollback` workflow 的 `reason` 输入承担同一作用。另有每日
`Runtime image review deadlines` workflow，在任一审核窗口剩余不足 3 天时失败告警；
它只是提前告警，不替代按期更新镜像扫描证据和 policy。

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

当前发布链路不提供 Traefik 入口模式。不要在同一生产机上再启动 Traefik 监听公网 `80/443`，也不要把 `stuhelper.com` / `join.stuhelper.com` 同时分散到 Traefik 和宝塔 Nginx；如确实需要外部负载均衡，只允许放在宝塔 Nginx 前面，并保持宝塔 Nginx 作为应用反代层。

### 方案 A：宝塔 Nginx（当前默认）

宝塔 Nginx 是唯一公网入口，负责 `stuhelper.com` 的 `80/443`、证书、HTTP 到 HTTPS 跳转和反向代理。仓库提供反代示例：

```text
infra/nginx/baota-stuhelper.conf
```

`baota-stuhelper.conf` 用于主站机器的 `stuhelper.com` / `join.stuhelper.com`。`join.stuhelper.com` 只承载加群验证业务入口，公开验证链接固定为 `https://join.stuhelper.com/verify/<code>`；join 根路径和主站业务页面路径必须返回 404，主站上的 `/verify/*` 也必须返回 404。Casdoor 通过 `sso.stuhelper.com` 公开，登录回调固定回到 `https://stuhelper.com/api/v1/auth/callback`。

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

远端发布主机默认使用 `NGINX_PUBLIC_INGRESS_PROFILE=stuhelper`，审计本机拥有的 `stuhelper.com` / `join.stuhelper.com` server block。`NGINX_PUBLIC_INGRESS_PROFILE=sso` 用于审计独立 `sso.stuhelper.com` Casdoor 公网入口。

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
- [ ] `ADMISSION_PUBLIC_BASE_URL=https://join.stuhelper.com`
- [ ] `CORS_ORIGINS=https://stuhelper.com,https://join.stuhelper.com,https://sso.stuhelper.com`
- [ ] `FRONTEND_METRICS_ALLOWED_ORIGINS=https://stuhelper.com,https://join.stuhelper.com`
- [ ] `WEB_VITE_SSO_URL=https://sso.stuhelper.com`
- [ ] `WEB_VITE_WEB_URL=https://stuhelper.com`
- [ ] 主站生产机 `NGINX_PUBLIC_INGRESS_PROFILE=stuhelper ./infra/ops/nginx-public-ingress-preflight.sh` 通过。
- [ ] `https://sso.stuhelper.com/.well-known/openid-configuration` 可达并返回 Casdoor discovery。
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

这表示新插件只接入 admission 入群后验证，不抢“举报 / 骰子 / 抽禁言”等生产既有命令，不接管消息风控监听，也不扫描新生材料原图转发；但保留“查询入群认证 / 重发认证链接 / 重新生成认证链接 / 跳过入群认证 / 清空入群未认证次数 / 解除入群拉黑”等 admission 管理命令。“跳过入群认证”只跳过本群审核并解除禁言，不代表 StuHelper 学生认证通过；“解除入群拉黑”不隐式清空失败次数，需要重新计数时单独执行“清空入群未认证次数”。旧 `student-query` 插件不要整体关闭；如它也对 admission 目标群做同一阶段入群验证，应在旧插件自己的目标群或功能范围里排除 `178037297`。

这些 YAML 值是启动默认值；Koishi 群管中心 WebUI 的“入群认证”页面会把 `actionStream.enabled`、`scheduler.fallbackScanEnabled`、`commands.enabled`、`admissionCommands.enabled`、`moderation.enabled` 和 `freshmanForward.enabled` 持久化到 `stuhelper_admission_runtime_settings`。`actionStream.enabled`、兜底扫描、消息风控和材料转发保存后立即生效；公开命令和 admission 管理命令只能关闭已注册命令，若启动时未注册，需要调整 `koishi.yml` 并重启后才能启用。`platform.baseUrl`、`platform.serviceToken`、`scheduler.scanIntervalSeconds`、`actionStream.reconnectDelaySeconds`、`admissionCommands.minAuthority` 和 `admissionCommands.operatorQQIDs` 仍是启动/安全配置，只在 WebUI 脱敏或汇总展示。NapCat 的账号级反向 OneBot 配置也属于生产可复现状态，`napcat/config/onebot11_<qq>.json` 里的 `reconnectInterval` 应保持 1000ms 级、`heartInterval` 应保持 10000ms 级；默认 30000ms 重连会在 Koishi 重启或临时断线时扩大消息丢失窗口，入群验证码这类一次性群消息会直接表现为“没反应”。

若同机启用 `stuhelper-core` 提供 Koishi 群管中心 WebUI，生产必须设置 `runtimeModules.enabled: false`。这会保留 Console 入口、WebUI 和 Console API，但不初始化 core 旧运行时模块，避免注册 `report`、`sub`、`config`、`ai` 等命令并与生产既有插件冲突。

如果 Koishi 日志出现 `TypeError: Invalid URL`，并且错误里 `base` 是原样的 `${{ env.STUHELPER_PLATFORM_BASE_URL }}`，说明当前运行时没有把 Koishi 配置占位符插值成真实环境变量。不要在生产 `node_modules` 里手改源码；应确认 Compose 已通过 `.env` 或 `env_file` 注入 `STUHELPER_PLATFORM_BASE_URL` 和 `STUHELPER_PLATFORM_SERVICE_TOKEN`，然后从本地仓库构建并部署包含 `@stuhelper/koishi-shared` 平台配置环境变量回退逻辑的 StuHelper 插件包。

如果当前 Koishi 生产仍通过宝塔 Compose 挂载本机 `node_modules` 运行 StuHelper 插件，插件发布包必须来自本地仓库当前构建产物，不允许在生产容器内或生产 `node_modules` 中直接改源码作为最终状态：

```bash
cd bots/koishi
corepack yarn build
cd ../..
./infra/ops/package-koishi-stuhelper-packages.sh /tmp/stuhelper-koishi-packages.tar.gz
sha256sum /tmp/stuhelper-koishi-packages.tar.gz
```

上传 `/tmp/stuhelper-koishi-packages.tar.gz` 到 Koishi 生产 Compose 目录后，生产侧先校验 sha256，再备份现有 StuHelper 包目录、`koishi/local-workspaces`、`koishi/package.json`、`koishi/yarn.lock` 与 `koishi.yml`，最后在 Compose 目录根部解包。归档内部路径已经固定为 `koishi/node_modules/...` 与 `koishi/local-workspaces/...`，所以解包目标必须是 Koishi Compose 目录，不是 `koishi/node_modules`。归档必须包含 `koishi-plugin-stuhelper-core/lib` 与 `koishi-plugin-stuhelper-core/dist`，否则群管中心 admission WebUI 不会随后端 Console API 一起发布。解包后必须运行 `koishi/STUHELPER_KOISHI_APPLY_LOCAL_WORKSPACES.cjs`，把 `koishi-plugin-stuhelper-core` 与 `koishi-plugin-stuhelper-group-guard` 固定为本地 `workspace:*` 依赖；否则 Koishi Market 更新普通插件时会把私有 StuHelper 包当作 npm 包解析，并请求 `https://registry.npmmirror.com/koishi-plugin-stuhelper-core` 导致 `Package not found`。

```bash
sha256sum stuhelper-koishi-packages.tar.gz
backup_dir="backups/admission-$(date -u +%Y%m%dT%H%M%SZ)"
mkdir -p "${backup_dir}"
cp -a koishi.yml koishi/package.json koishi/yarn.lock koishi/local-workspaces koishi/node_modules/@stuhelper/koishi-shared koishi/node_modules/@stuhelper/koishi-moderation-core koishi/node_modules/koishi-plugin-stuhelper-core koishi/node_modules/koishi-plugin-stuhelper-binding koishi/node_modules/koishi-plugin-stuhelper-group-guard "${backup_dir}/"
tar -xzf stuhelper-koishi-packages.tar.gz
node koishi/STUHELPER_KOISHI_APPLY_LOCAL_WORKSPACES.cjs
docker compose exec koishi sh -lc 'cd /koishi && corepack yarn install --immutable'
docker compose restart koishi
```

这个 tar 只包含运行时 `package.json`、`lib/`、`stuhelper-core` 浏览器 `dist/`、本地 workspace 镜像和无 secret 的 workspace guard，不包含源码树、嵌套 `node_modules`、环境变量文件、SSH 辅助脚本或 secret。临时 SSH 命令和上传脚本可以放在本机未跟踪路径，但不得提交到 git。

生产 `student-query.enableGroupVerify` 保持 `true`；admission 上线不能靠关闭旧插件本身来规避冲突，冲突应通过新插件的 `commands.enabled=false`、`moderation.enabled=false`，以及旧插件自己的目标群/功能范围收敛处理。

Koishi 重启后用仓库脚本采集 admission 生产证据，不要临时拼带 token 的命令：

```bash
KOISHI_COMPOSE_DIR=/www/server/panel/data/compose/koishi-napcat \
KOISHI_ADMISSION_BOT_SELF_ID=<botSelfID> \
./infra/ops/koishi-admission-production-evidence.sh
```

该脚本检查 `stuhelper-group-guard:admission` 配置目标群 `178037297`、`commands.enabled=false`、`moderation.enabled=false`、`freshmanForward.enabled=false`、`student-query.enableGroupVerify=true`、NapCat 账号级 OneBot 反连配置、Koishi 容器环境、bot admission API 200，以及最近日志中没有 `duplicate command names: 举报`、`admission 401/unauthorized`、`B0000001`、`pending-forward` 循环报错。ChatLuna API key invalid 之类日志不属于 admission 上线验收范围。
默认日志窗口是最近 2 小时，避免上午救火前的历史错误影响当前验收；需要回溯时可显式设置 `KOISHI_ADMISSION_LOG_SINCE=24h`。长期运行的 Koishi 容器可能已经没有启动加载日志，脚本默认不要求加载日志必须出现在当前窗口；如果刚重启后要强制检查加载日志，可设置 `KOISHI_ADMISSION_REQUIRE_LOAD_LOG=true`。

也可以在 Koishi 节点用聚合入口采集同一份证据：

```bash
ADMISSION_MVP_PRODUCTION_EVIDENCE_MODE=koishi \
./infra/ops/admission-mvp-production-evidence.sh
```

真实 QQ 小号入群 E2E 分两段留证。第一段在小号入群后立即执行，证明真实入群事件已经让 Koishi/后端创建 `https://join.stuhelper.com/verify/<code>` canonical session：

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
