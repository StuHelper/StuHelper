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

命令行显式导出的 `BACKUP_DATABASE_URL` / `REPLICATION_DATABASE_URL` 优先于环境文件；未显式导出时才读取标准 StuHelper 环境与 secret backend。样板 URL 中的密码占位符只在进程内展开并进行 URL 编码，不会把真实密码回写到共享配置文件。生产使用内置 PostgreSQL 时，两条 URL 还必须固定到受检 Compose `postgres` 服务、配置的数据库和默认服务端口，不允许用 query 参数改写路由；实际连接返回的服务端 IP 必须属于同一个健康的规范 PostgreSQL 容器，且不能是 standby。该检查在每次实际生成备份前重新执行，避免首次激活后配置漂移到另一个可达集群。

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

- `run-scheduled-backup.sh` 会先在本地生成备份文件，再调用 `./infra/ops/sync-postgres-backups.sh`，把逻辑备份、base backup、WAL 归档复制到对象存储
- 只有本轮全部远端复制成功后，它才会清理超出保留期的本地 logical / base / WAL 文件；认证、网络或对象存储故障不会触发本地删除
- 同步器显式排除 `*.partial*`、WAL 归档的 `*.tmp*` 和 staging 路径；只有已经原子发布的工件会上传
- 生产 systemd unit 固定要求异机门禁；门禁会在创建备份或执行 logical / base / WAL 保留期清理之前检查，失败时不会删除任何尚未上传的本地工件
- `current-release.env` 尚未产生时，只有 Docker daemon 可确认本机 PostgreSQL 容器和 data volume 都不存在、未选择外部 PostgreSQL，且没有 `releases.log` 或 per-tag 历史证据，timer 才允许在真正空白的 bootstrap 主机上成功延期。已有 datastore 时仍会失败关闭，不能把“未备份”伪装成成功 no-op
- 仅当发布账本已完整丢失、又必须保护经过人工审计的内置 PostgreSQL 存量库时，才可在完成隔离 base + WAL replay 演练后，由部署用户显式运行 `./infra/ops/activate-existing-postgres-backups.sh`。运行前必须把 `BACKUP_OBJECT_STORAGE_PREFIX` 配置为以实时 `system_identifier` 结尾的集群专属前缀，例如 `postgres/1234567890123456789`；同步器和激活账本会共同校验并绑定该值，确保更换集群后 logical/base/WAL 进入新的恢复命名空间，不能与前任集群的同名 WAL 段混用。脚本先取得与生产发布相同的主机锁，并在任何持久写入前核对固定 digest、Compose 身份、data/WAL volume、实时 `system_identifier`、集群专属前缀、归档进度和异机目标；随后强制从该实时集群新建 logical dump 与 base backup，同步完整 logical/base/WAL 集合，再把两份新工件从异机对象存储取回并交叉验证文件名和 SHA256。每份证据的修改时间都必须不早于当前激活尝试，72 小时年龄上限仅作为覆盖受保护生成与传输容量窗口的额外边界。只有这些步骤全部成功，才会把两份恢复工件的名称与摘要写入 `0600` 的 `postgres-backup-activation.json` 及不可变副本。定时任务会在每次运行时重新把当前记录绑定到实时容器、集群和对象前缀；镜像、volume、system identifier 或前缀漂移会让调度立即失败，但不会把历史记录变成永久死锁。完成新的隔离恢复审计后再次运行激活脚本，会先验证已有不可变记录形成从当前指针回溯到首条记录的无环 SHA256 链，再生成和取回一组全新恢复工件，并以显式 superseding 记录链接前任；缺失指针、孤儿记录、链路篡改或未显式完成新鲜证据都拒绝替换。它不能覆盖残存或损坏的 `current-release.env`、`releases.log`、per-tag 账本，也不支持拿内置库记录授权外部 PostgreSQL；脚本内置的工件校验仍不能替代此前由操作者完成的隔离 WAL replay 演练
- StuHelper 环境加载器将配置文件按数据解析，拒绝 `PATH`、`PYTHON*`、`LD_*`、`DYLD_*`、`BASH_ENV` / `ENV`、`GCONV_PATH`、`NODE_OPTIONS` 等进程控制变量，并在解析前与加载后清除父进程继承的同类变量；定时任务调用备份和同步子脚本时仍使用非登录、无 profile 的隔离 Bash，防止子进程启动钩子改变门禁或清理顺序

生产机建议直接安装 systemd timer：

```bash
sudo ./infra/ops/install-backup-timers.sh
```

灾难恢复后若应用发布账本已无法找回、但内置 PostgreSQL 已完成独立恢复验收，先保留 timer
为 stopped 状态，再执行一次受控激活并实跑三个 service；不要手写 JSON 或伪造应用 release：

```bash
cd /opt/stuhelper
sudo -u stuhelper ./infra/ops/activate-existing-postgres-backups.sh
sudo systemctl start stuhelper-postgres-backup-sync.service
sudo systemctl start stuhelper-postgres-dump-backup.service
sudo systemctl start stuhelper-postgres-basebackup.service
```

三次 service 均成功后再启动 timer。后续正式应用发布仍必须通过 `prod-deploy.sh` 生成正常的
release ledger；datastore backup activation 不是应用发布成功证明，也不能用于回滚应用镜像。

默认计划：

- 每天 `03:15` 做逻辑备份
- 每周日 `03:45` 做 base backup
- 每 15 分钟执行一次 backup artifact sync
- 三个 timer 的 coalescing accuracy 固定为一分钟且不允许 randomized delay，防止 drop-in 将工件新鲜度悄然延后
- 内置生产 PostgreSQL 必须使用 `POSTGRES_ARCHIVE_MODE=on`、`POSTGRES_ARCHIVE_TIMEOUT=15min` 和仓库定义的幂等归档命令；同步前会验证实时归档配置、当前 postmaster 启动后的成功进度及 WAL 文件确实位于受保护 volume，本次同步缺少可信进度时会用一次 `pg_switch_wal()` 实探
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

在 `APP_ENV=production` 或显式要求异机门禁时，取回脚本不会复用父进程留下的旧地址固定：它会先加载最终环境，重新解析并验证实际传输主机，再把本轮验证结果固定给 rclone。门禁失败时不会发起取回，也不能把同机对象存储响应计作生产恢复 evidence。

`BACKUP_OBJECT_STORAGE_PREFIX` 是恢复输入的一部分，必须选择目标 base backup 对应记录中的
`backupObjectStoragePrefix`；不要为了“找得到更多 WAL”改回父级 `postgres` 前缀。不同
`system_identifier` 的对象必须保留在各自命名空间。

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
独立测试可以显式覆盖 `POSTGRES_BACKUP_EVIDENCE_FETCH_COMMAND`，但存量生产库激活流程会
强制使用仓库内的 `fetch-postgres-backups.sh`；环境文件或父进程中的测试 fixture 不能替代
异机下载证据。

**evidence 通过不等于 PITR 已验收。**它证明近期工件存在、完整且能从远端取回；仍须按下方
演练清单，在隔离实例实际启动恢复后的 PGDATA，并在目标时间点恢复场景中验证 WAL 连续性。

使用 `EXTERNAL_POSTGRES_ENABLED=true` 时，StuHelper 不拥有外部实例的实时 WAL volume，因此仓库脚本只同步 logical dump 和包含流式 WAL 的一致性 base backup，不会创建或读取本地 WAL 卷。外部平台的连续 WAL 归档、保留策略、故障域和 PITR 演练必须由其 DBA/供应商单独验收并留证；只有下述机器可验证的证据通过，部署和同步才会继续。

外部平台的 root 管理任务必须原子刷新 `/etc/stuhelper/external-postgres-pitr-evidence.json`。目录和文件 owner 必须是 root，且 group/other 不可写；部署用户只需只读权限。该文件不得由 StuHelper 发布流程自我签发，也不得放入 Git 或 secret env。证据结构如下（示例值不是生产事实）：

```json
{
  "schemaVersion": 1,
  "evidenceId": "provider-check-20260803T025000Z",
  "provider": "external-postgres-platform",
  "evidenceUri": "https://evidence.example.com/postgres/check-20260803T025000Z",
  "clusterSystemIdentifier": "1234567890123456789",
  "observedAt": "2026-08-03T02:50:00Z",
  "expiresAt": "2026-08-03T03:30:00Z",
  "continuousArchiving": {
    "enabled": true,
    "offHost": true,
    "rpoSeconds": 900,
    "retentionHours": 168,
    "lastArchivedAt": "2026-08-03T02:45:00Z"
  },
  "restoreDrill": {
    "status": "passed",
    "completedAt": "2026-07-15T12:00:00Z",
    "isolatedTarget": true,
    "baseBackupVerified": true,
    "walReplayVerified": true
  }
}
```

门禁会实时读取当前外部集群的 `pg_control_system().system_identifier` 并与证据精确比较；外部 DBA 需要允许 `stuhelper_backup` 执行这个只读系统信息函数。证据观测与最新 WAL 均不得超过 30 分钟，`rpoSeconds` 不得大于 900，异机保留不得少于 168 小时，证据有效期不得超过一小时，隔离恢复演练不得早于 90 天。`evidenceUri` 必须是无内嵌凭据和 fragment 的 HTTPS 报告地址。任一条件不满足时，不允许用布尔确认或 logical/base backup 绕过。

对象存储“可取回”也不自动代表异机灾备：同一生产主机上的 MinIO 会与数据库一起丢失。生产配置
只有在备份端点位于独立故障域、且已验证生产主机完全丢失后仍可访问时，才能设置
`BACKUP_OBJECT_STORAGE_OFF_HOST_CONFIRMED=true`。`remote-preflight.sh` 和 `prod-deploy.sh` 会对此
失败关闭，并拒绝单标签 Compose 服务名、旧式缩写数字 IPv4、带 zone identifier 的 IPv6 及解析到本机接口的 FQDN。使用 virtual-hosted S3 时，门禁会分别解析、校验并固定 endpoint 与 `bucket.endpoint`；bucket 必须能组成合法的小写 ASCII DNS 主机名，不能让实际传输绕过基础 endpoint 的验证。生产备份
systemd service 也固定要求这项门禁，不能由共享 env 将要求降级；配置漂移后定时同步会失败并留给
systemd/告警处理，而不是继续把同机副本计作灾备。升级已有节点后须由 root 重新运行
`./infra/ops/install-backup-timers.sh`；预检会把三个 service 的有效 `Environment`、pre-exec
`UnsetEnvironment` 与安装器定义精确比对；有效 `User` / `Group` 还必须与当前非 root 部署账号及其主组一致，任何 root 或其他身份的 unit/drop-in 都会失败关闭。服务先清除动态加载器变量，再以 `env -i` 丢弃 manager
defaults 和其他继承环境，只传固定 `PATH`、配置路径与异机门禁；额外的 `LD_PRELOAD`、`PYTHONPATH`、`PATH`
等进程控制变量会失败关闭。预检同时拒绝 `EnvironmentFile=`、`PassEnvironment=` 以及不精确的
`UnsetEnvironment=` unit/drop-in，不能继续使用缺少或可降级门禁的旧单元。
服务生命周期还必须没有 `ExecReload`、`ExecStop` 或 `ExecStopPost` 钩子，避免额外命令在备份退出或人工停止时改写恢复工件。
三个 oneshot 分别使用逻辑备份 `18h`、base backup `26h`、独立同步 `12h` 的有限启动上限，并固定 `TimeoutStopUSec=2min`、`KillMode=control-group`、`SendSIGKILL=yes`。逻辑与 base 服务会在生成后内联执行同一异机同步，因此总预算分别由 `4h/12h` 生成窗口、`12h` 最大传输窗口和 `2h` 校验/压缩/重试余量相加得出；独立同步仍固定 `12h`。生产容量规划必须保证一份最大完整 base backup 加待补 WAL 能在 12 小时内以最低持续异机带宽传完。若实测不满足该支持边界，应先扩容链路或拆分备份，而不是用 drop-in 放宽门禁。预检拒绝 drop-in 缩短、延长或关闭上限，避免挂死任务永久吞掉后续 timer。
预检还通过 systemd D-Bus 校验有效 `Conditions` 与 `Asserts` 都为空，避免 `ConditionPathExists` 等 drop-in 把每次备份静默变成 skipped activation。

生产主机位于 1:1 NAT、hairpin LB 或公网边缘之后时，还必须把所有能路由回本机的地址/CIDR 写入
`BACKUP_OBJECT_STORAGE_LOCAL_IDENTITY_CIDRS`；无额外身份时显式写 `none`。该清单用于补足
`ip address` 看不到的公网身份，但仍不能替代“关闭/丢失生产主机后从独立环境取回并恢复”的演练。

腾讯云同地域 CVM 使用官方 regional COS endpoint 时，DNS 可能返回 provider-private service
VIP。只有配置 `BACKUP_OBJECT_STORAGE_PROVIDER=TencentCOS`、官方 regional hostname、匹配
region、完整 `BucketName-APPID`、virtual-hosted addressing、默认 HTTPS 端口和公开 CA 校验后，
才可显式设置 `BACKUP_OBJECT_STORAGE_PROVIDER_PRIVATE_ENDPOINT=tencent-cos`。该例外仅允许通过
TLS 主机名验证保护的腾讯云服务 VIP；任意其他 link-local、错误 provider/region/bucket、私有
CA 或禁用证书校验仍会失败。这个模式不代表灾备已经完成，独立环境取回与
`BACKUP_OBJECT_STORAGE_OFF_HOST_CONFIRMED=true` 仍需分别验收。

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
