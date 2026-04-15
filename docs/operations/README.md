# 运维文档

## 权威来源

1. `docker-compose.yml` — 容器编排
2. `server/migrations/*.sql` — 数据库 schema
3. `infra/ops/*.sh` — 发布、备份、恢复、smoke check
4. `infra/ansible/` — 远端 bootstrap / deploy / rollback
5. 本目录文档 — 执行顺序、检查项、回滚规则

## 文档索引

| 文档 | 内容 |
|------|------|
| [automation.md](automation.md) | 一键启动、GitLab 发布、Ansible |
| [database-migrations.md](database-migrations.md) | 迁移、seed、回滚 |
| [release-runbook.md](release-runbook.md) | 发布流程、verify、回滚 |
| [backup-and-restore.md](backup-and-restore.md) | PostgreSQL 备份与恢复 |
| [observability.md](observability.md) | Grafana LGTM、告警、排障 |
| [production-topology.md](production-topology.md) | 生产部署拓扑 |
| [native-auth-qa-checklist.md](native-auth-qa-checklist.md) | 原生认证 QA 检查清单 |

## 环境

| 环境 | 用途 | 规则 |
|------|------|------|
| development | 日常开发 | 可加载 seed；允许 `migrate-down-one` |
| staging | 预发布 | 禁止 seed；`develop` 自动部署；必须先迁移再 verify |
| production | 正式运营 | 禁止 seed；`main` 先构建，再手工触发 `deploy_production`；发布前备份；发布后 verify |

## 远端控制面

生产 / staging 主机都遵循同一条原则：

- **目标机自持部署控制面**：`${DEPLOY_APP_DIR}/.deploy/remote.env`
- **CI / Ansible 只负责上传 bundle + 传递 release metadata**（`TAG`、`*_IMAGE_REF`、`ROLLBACK_TAG`）
- **secret backend 选择、env 文件位置、registry 凭据引用** 都由目标机的 `.deploy/remote.env` 决定

如果控制面配置变更，直接在目标机执行：

```bash
cd /opt/stuhelper
./infra/ops/init-remote-deploy-config.sh
```

## 铁律

- 生产环境不执行 `seed.sql`
- 上线先备份，再迁移，再发布
- 远端部署前必须保证 `.deploy/remote.env` 已就位并通过 `remote-preflight.sh`
- 远端发布前必须同时具备 `BACKUP_OBJECT_STORAGE_ENDPOINT` / `BACKUP_OBJECT_STORAGE_BUCKET` / `BACKUP_OBJECT_STORAGE_ACCESS_KEY_ID` / `BACKUP_OBJECT_STORAGE_SECRET_ACCESS_KEY`
- 生产主机要先装好逻辑备份 / base backup / backup sync 定时任务
- 数据库权威来源是 `server/migrations/`；仓库内不再维护并执行独立 schema 快照
- Smoke Check 全绿才算发布完成
- 发布后确认 Grafana / Prometheus / Loki / Tempo / Alertmanager 全绿
- 手动回滚记录触发原因、执行人、时间和结果
