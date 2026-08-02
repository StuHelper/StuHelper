---
type: guide
audience: ops
status: current
authoritative-source: infra/ops/*.sh
last-verified: 2026-07-30
---

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

命令行显式导出的 `BACKUP_DATABASE_URL` / `REPLICATION_DATABASE_URL` 优先于环境文件；未显式导出时才读取标准 StuHelper 环境与 secret backend。样板 URL 中的密码占位符只在进程内展开并进行 URL 编码，不会把真实密码回写到共享配置文件。

脚本会：

- 生成 PostgreSQL custom-format dump
- 生成同名 `.sha256` 校验文件
- 先在同步目录之外生成工件，以目标目录中的 `.partial.<pid>` 名称完成跨文件系统复制，
  再原子改名发布；失败不会留下可被同步器当成完整备份的最终文件
- 通过非 root、只读、仅挂载客户端 CA 的一次性 `postgres-client` 容器连接数据库；宿主机无需安装 PostgreSQL 客户端，备份任务不会接触数据库数据卷或服务端 TLS 私钥

## Base Backup（pg_basebackup）

```bash
export REPLICATION_DATABASE_URL='postgres://...'
BACKUP_MODE=basebackup ./infra/ops/backup-postgres.sh backups/stuhelper-$(date +%F-%H%M%S).tar.gz
```

适用场景：

- 需要保留更完整的实例级恢复点
- 配合 WAL 归档做 PITR

物理备份不是把 tar 流直接写入最终目录。脚本会在独立 staging 中使用
`pg_basebackup --format=plain --wal-method=stream` 生成完整 PGDATA，使用临时 replication slot
保护备份期间所需 WAL，以 `pg_verifybackup` 校验 manifest 和全部文件后才压缩、生成 SHA256
并原子发布。这里不创建需要运维生命周期管理的持久复制槽，也不使用 `fetch` 模式依赖
`wal_keep_size` 在长备份期间保留全部 WAL。

目标文件或 sidecar 已存在时脚本会拒绝覆盖。直接运行时，默认 staging 是用户 local-state
下的 `postgres/backup-staging`；systemd 安装器会以 deploy 用户、`0700` 权限创建并注入
`/var/lib/stuhelper/postgres/backup-staging`。需要放到专用磁盘时设置
`BACKUP_STAGING_DIR` 并确保 timer 用户可写。staging 至少要容纳一份未压缩 PGDATA 和正在
生成的压缩包。

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
- 同步器显式排除 `*.partial*`、WAL 归档的 `*.tmp*` 和 staging 路径；只有已经原子发布的工件会上传

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
- `${POSTGRES_WAL_RESTORE_DIR:-$HOME/Library/Application Support/StuHelper/postgres/wal-restore}`（Linux 默认 `~/.local/state/stuhelper/postgres/wal-restore`）

## 备份 evidence

```bash
./infra/ops/postgres-backup-evidence.sh
```

该入口会同时验证：

- 本地最新逻辑备份与物理 base backup 的 SHA256 和新鲜度
- 从对象存储重新取回的最新逻辑备份与物理 base backup 的 SHA256 和新鲜度
- 两份物理 tar.gz 都可完整遍历
- 可检查时，三个 systemd timer 的安装与启用状态

默认逻辑备份最长允许 36 小时，周度物理备份最长允许 8 天；可分别用
`POSTGRES_BACKUP_EVIDENCE_MAX_LOGICAL_AGE_SECONDS` 和
`POSTGRES_BACKUP_EVIDENCE_MAX_BASE_AGE_SECONDS` 收紧。生成的
`infra/generated/postgres-backup-evidence.json` 不含连接串或对象存储密钥。

**evidence 通过不等于 PITR 已验收。**它证明近期工件存在、完整且能从远端取回；仍须按下方
演练清单，在隔离实例实际启动恢复后的 PGDATA，并在目标时间点恢复场景中验证 WAL 连续性。

对象存储“可取回”也不自动代表异机灾备：同一生产主机上的 MinIO 会与数据库一起丢失。生产配置
只有在备份端点位于独立故障域、且已验证生产主机完全丢失后仍可访问时，才能设置
`BACKUP_OBJECT_STORAGE_OFF_HOST_CONFIRMED=true`。`remote-preflight.sh` 和 `prod-deploy.sh` 会对此
失败关闭，并拒绝单标签 Compose 服务名、旧式缩写数字 IPv4 及解析到本机接口的 FQDN。生产备份
systemd service 也固定要求这项门禁，不能由共享 env 将要求降级；配置漂移后定时同步会失败并留给
systemd/告警处理，而不是继续把同机副本计作灾备。升级已有节点后须由 root 重新运行
`./infra/ops/install-backup-timers.sh`；预检会验证三个 service 的有效环境，拒绝继续使用缺少门禁的旧单元。

## 逻辑备份恢复

> 恢复是破坏性操作，必须先确认目标库、确认当前连接字符串、确认业务窗口。

```bash
export DATABASE_URL='postgres://...'
ALLOW_DESTRUCTIVE=1 ./infra/ops/restore-postgres.sh backups/stuhelper-2026-03-30-120000.dump
```

逻辑恢复同样使用一次性 `postgres-client` 容器，连接目标可为内置或外部 PostgreSQL；恢复容器只读取标准输入中的 dump 和客户端 CA，不挂载活动数据库数据卷。

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
WAL_ARCHIVE_DIR=/var/lib/stuhelper/postgres/wal-restore \
./infra/ops/restore-postgres-basebackup.sh \
  backups/stuhelper-2026-03-30T120000Z.tar.gz \
  /var/lib/postgresql/data
```

脚本会写入 `restore_command` 和 `recovery.signal`，让 PostgreSQL 从 WAL 归档继续恢复。运行中的 PostgreSQL 实例使用 Docker named volume `postgres_wal_archive` 保存在线 WAL 归档；恢复演练则显式使用本地 restore cache 目录，避免把活动 WAL 写回仓库树。

恢复脚本会先按 sidecar 中的摘要验证实际输入文件，再检查压缩包可读，之后才清空明确传入的
PGDATA 目录；提取后还会要求 `PG_VERSION` 和 `backup_manifest` 存在。sidecar 中只依赖摘要，
因此对象存储取回到不同目录后仍可验证。

## 生产规则

- 每次 production 发布前至少做一次人工备份
- 备份对象存储必须位于独立故障域；同机 MinIO 只能作为迁移前临时副本，不能设置
  `BACKUP_OBJECT_STORAGE_OFF_HOST_CONFIRMED=true`
- 生产机必须启用逻辑备份 / base backup / backup sync timer
- 发布前 `postgres-backup-evidence.sh` 必须同时显示四份（本地/取回 × 逻辑/物理）工件
  `sha256Verified=true`、`fresh=true`，物理工件还须 `archiveReadable=true`
- 任何包含破坏性迁移的发布，必须先做恢复演练
- 恢复前先用 `fetch-postgres-backups.sh` 验证对象存储工件能拉回
- 恢复后必须跑 `./infra/ops/smoke-check.sh`
- 远端发布前必须通过 `./infra/ops/remote-preflight.sh`

## 建议保留策略

- 每日逻辑备份：保留 14 天
- 每周 base backup：保留 30 天
- WAL 归档：保留 14 天
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

## 恢复演练

### 恢复验证步骤

恢复完成后，按以下顺序确认数据完整性：

1. **校验文件完整性**：对比恢复所用 dump 文件的 `.sha256` 是否与备份时生成的一致
2. **核心表行数比对**：恢复前记录各核心表的行数，恢复后重新查询并比对
   ```bash
   psql "$DATABASE_URL" -c "
     SELECT 'users' AS tbl, count(*) FROM users
     UNION ALL SELECT 'courses', count(*) FROM courses
     UNION ALL SELECT 'reviews', count(*) FROM reviews
     UNION ALL SELECT 'reports', count(*) FROM reports;
   "
   ```
3. **最新记录时间戳检查**：确认恢复后数据库中最新一条记录的 `created_at` 与预期备份时间点一致
   ```bash
   psql "$DATABASE_URL" -c "SELECT max(created_at) FROM reviews;"
   ```
4. **应用层冒烟检查**：运行 `./infra/ops/smoke-check.sh`，确认所有业务端点正常响应
5. **认证流程验证**：手动执行一次登录 → 访问受保护端点 → 登出流程，确认 OIDC + token 链路完整
6. **外键与约束检查**：确认数据库约束未被破坏
   ```bash
   psql "$DATABASE_URL" -c "
     SELECT conname, conrelid::regclass
     FROM pg_constraint
     WHERE contype = 'f' AND NOT convalidated;
   "
   ```

### 定期演练建议

| 演练类型 | 频率 | 负责人 | 说明 |
|---------|------|--------|------|
| 逻辑备份恢复 | 每月一次 | 运维 / DBA | 在 staging 环境还原最新逻辑备份并执行上述验证步骤 |
| Base Backup + PITR | 每季度一次 | 运维 / DBA | 还原 base backup 并追 WAL 到指定时间点，验证数据一致性 |
| 对象存储拉取 | 每月一次 | 运维 | 从对象存储拉取全部备份工件到干净机器，验证文件完整 |
| 全链路演练 | 每季度一次 | 全团队 | 模拟生产故障：停库 → 拉取备份 → 恢复 → 冒烟检查 → 业务验证 |

### 演练记录模板

每次演练完成后填写记录，归档至 [../internal/drill-logs/README.md](../internal/drill-logs/README.md) 对应目录：

```
日期：YYYY-MM-DD
演练类型：逻辑备份恢复 / PITR / 全链路
执行人：
备份文件：
恢复耗时：
验证结果：通过 / 未通过
发现问题：
改进措施：
```
