# 发布运行手册

## 适用范围

- GitLab CI/CD 自动发布
- 手工 SSH 到部署机执行的应急发布

## 发布前检查

- [ ] 本次变更已通过 CI（web / admin / backend）
- [ ] production 发布已由发布人手工审批（`deploy_production`）
- [ ] 如果包含数据库变更，已完成备份（注：`prod-deploy.sh` 现已自动在迁移前执行 `backup-postgres.sh`）
- [ ] 生产机上的逻辑备份 / base backup / backup sync timer 已启用
- [ ] 远端部署控制面已核对：`.deploy/remote.env`
- [ ] 共享配置已核对：`.env.prod.shared`
- [ ] secrets 已核对：`.env.prod.secrets`（本地演练可用 `.env.prod.secrets.local`）；运行时派生 secrets 由 `.env.prod.generated.secrets` 管理
- [ ] registry 凭据已核对：`.secrets/registry/username`、`.secrets/registry/password` 或等价 secret ref
- [ ] 关键变量已核对：`POSTGRES_PASSWORD`、`REDIS_PASSWORD`、`TAG`、`OBJECT_STORAGE_*`、`WEB_VITE_SSO_URL`
- [ ] 观测配置已核对：`METRICS_PASSWORD`、`GRAFANA_ADMIN_PASSWORD`、`OTEL_ENABLED=true`
- [ ] staging 已验证通过（如有 staging）

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

- API：`http://127.0.0.1:8080/health/live`
- API：`http://127.0.0.1:8080/health/ready`
- Web：首页 200
- Admin：首页 200
- Grafana：`http://127.0.0.1:3003`
- Prometheus：`http://127.0.0.1:9090/-/ready`
- Loki：`http://127.0.0.1:3100/ready`
- Tempo：`http://127.0.0.1:3200/ready`
- `docker compose --profile prod ps` 中 `app` / `frontend` / `admin` 为 healthy/running

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
5. 重新跑 `./infra/ops/observability-smoke-check.sh`

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
