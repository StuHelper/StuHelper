# PostgreSQL 备份与恢复手册

## 脚本位置

- `infra/ops/backup-postgres.sh`
- `infra/ops/run-scheduled-backup.sh`
- `infra/ops/sync-postgres-backups.sh`
- `infra/ops/fetch-postgres-backups.sh`
- `infra/ops/restore-postgres.sh`
- `infra/ops/restore-postgres-basebackup.sh`
- `infra/ops/install-backup-timers.sh`

## 逻辑备份（pg_dump）

```bash
export BACKUP_DATABASE_URL='postgres://...'
./infra/ops/backup-postgres.sh backups/stuhelper-$(date +%F-%H%M%S).dump
```

脚本会：

- 生成 PostgreSQL custom-format dump
- 生成同名 `.sha256` 校验文件

## Base Backup（pg_basebackup）

```bash
export REPLICATION_DATABASE_URL='postgres://...'
BACKUP_MODE=basebackup ./infra/ops/backup-postgres.sh backups/stuhelper-$(date +%F-%H%M%S).tar.gz
```

适用场景：

- 需要保留更完整的实例级恢复点
- 配合 WAL 归档做 PITR

## 定时备份

仓库已经提供统一入口：

```bash
./infra/ops/run-scheduled-backup.sh dump
./infra/ops/run-scheduled-backup.sh basebackup
```

说明：

- `run-scheduled-backup.sh` 会在本地生成备份文件
- 它会清理超出保留期的 logical / base / WAL 文件
- 最后会自动调用 `./infra/ops/sync-postgres-backups.sh`，把逻辑备份、base backup、WAL 归档镜像到对象存储

生产机建议直接安装 systemd timer：

```bash
sudo ./infra/ops/install-backup-timers.sh
```

默认计划：

- 每天 `03:15` 做逻辑备份
- 每周日 `03:45` 做 base backup
- 每 15 分钟执行一次 backup artifact sync
- WAL 归档目录按 `WAL_ARCHIVE_RETENTION_DAYS` 清理

## 从对象存储取回恢复工件

恢复前如果目标机本地没有备份文件，先从对象存储同步回来：

```bash
./infra/ops/fetch-postgres-backups.sh all
```

也可以只拉某一类：

```bash
./infra/ops/fetch-postgres-backups.sh logical
./infra/ops/fetch-postgres-backups.sh base
./infra/ops/fetch-postgres-backups.sh wal
```

默认会把对象存储中的内容拉回：

- `backups/postgres/logical`
- `backups/postgres/base`
- `infra/generated/postgres/wal-archive`

## 逻辑备份恢复

> 恢复是破坏性操作，必须先确认目标库、确认当前连接字符串、确认业务窗口。

```bash
export DATABASE_URL='postgres://...'
ALLOW_DESTRUCTIVE=1 ./infra/ops/restore-postgres.sh backups/stuhelper-2026-03-30-120000.dump
```

## Base Backup 恢复 / PITR

先恢复 base backup：

```bash
ALLOW_DESTRUCTIVE=1 ./infra/ops/restore-postgres-basebackup.sh \
  backups/stuhelper-2026-03-30T120000Z.tar.gz \
  /var/lib/postgresql/data
```

如果还要继续追 WAL：

```bash
ALLOW_DESTRUCTIVE=1 \
WAL_ARCHIVE_DIR=/opt/stuhelper/infra/generated/postgres/wal-archive \
./infra/ops/restore-postgres-basebackup.sh \
  backups/stuhelper-2026-03-30T120000Z.tar.gz \
  /var/lib/postgresql/data
```

脚本会写入 `restore_command` 和 `recovery.signal`，让 PostgreSQL 从 WAL 归档继续恢复。

## 生产规则

- 每次 production 发布前至少做一次人工备份
- 生产机必须启用逻辑备份 / base backup / backup sync timer
- 任何包含破坏性迁移的发布，必须先做恢复演练
- 恢复前先用 `fetch-postgres-backups.sh` 验证对象存储工件能拉回
- 恢复后必须跑 `./infra/ops/smoke-check.sh`
- 远端发布前必须通过 `./infra/ops/remote-preflight.sh`

## 建议保留策略

- 每日逻辑备份：保留 7 天
- 每周 base backup：保留 2 周
- WAL 归档：保留 7 天
- 每次重大上线前额外生成一次人工备份

## 演练清单

- [ ] 备份文件能正常生成
- [ ] `.sha256` 校验通过
- [ ] 逻辑备份 / base backup / WAL 均能同步到对象存储
- [ ] 对象存储中的工件能重新拉回目标机
- [ ] staging 库可完整恢复（逻辑备份）
- [ ] base backup + WAL 归档可恢复到目标时间点
- [ ] 恢复后应用健康检查通过
- [ ] Web / Admin Smoke Check 通过
