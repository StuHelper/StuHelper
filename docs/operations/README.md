# 运维文档

## 权威来源

1. `docker-compose.yml` — 容器编排
2. `server/migrations/*.sql` — 数据库 schema
3. `infra/ops/*.sh` — 备份、恢复、smoke check
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

## 环境

| 环境 | 用途 | 规则 |
|------|------|------|
| development | 日常开发 | 可加载 seed；允许 `migrate-down-one` |
| staging | 预发布 | 禁止 seed；`develop` 自动部署；必须先迁移再 verify |
| production | 正式运营 | 禁止 seed；`main` 先构建，再手工触发 `deploy_production`；发布前备份；发布后 verify |

## 铁律

- 生产环境不执行 `seed.sql`
- 上线先备份，再迁移，再发布
- 生产主机要先装好逻辑备份 / base backup 定时任务，`remote-preflight.sh` 必须通过
- 数据库权威来源是 `migrations/`，不是 `init.sql`
- Smoke Check 全绿才算发布完成
- 发布后确认 Grafana / Prometheus / Loki / Tempo / Alertmanager 全绿
- 手动回滚记录触发原因、执行人、时间和结果
