---
type: guide
audience: backend-dev, ops
status: current
authoritative-source: server/migrations/
last-verified: 2026-07-30
---

# 数据库迁移运行手册

## 权威文件

- `server/migrations/*.up.sql`：按版本号顺序组成当前 PostgreSQL schema
- `server/migrations/*.down.sql`：对应版本的受控回退脚本
- `server/migrations/000001_initial_schema.*.sql`：不可变的初始基线，只负责第一个版本
- `server/scripts/seed.sql`：仅开发环境示例数据

`000001_initial_schema.up.sql` 不是当前完整 schema。空数据库必须从 `000001` 开始依次应用所有
尚未执行的 `.up.sql`；已有数据库只应用版本表之后的增量 migration。

## 维护规则

- 已进入版本库的编号 migration 视为不可变，尤其禁止修改 `000001` 来发布新结构。
- 每个结构或数据演进都新增下一编号的 `.up.sql` / `.down.sql` 文件对；修正已发布 migration
  也必须使用新的编号。
- migration 必须支持从上一已发布版本前向升级。涉及大表回填、约束收紧或删除字段时，采用
  expand / migrate / contract，保证应用与 schema 的滚动兼容。
- 不要普遍使用 `IF NOT EXISTS` / `IF EXISTS` 把 migration 伪装成可重跑；这会掩盖部分执行、
  dirty version 和 schema drift。只有语义本身明确需要幂等时才能使用，并在评审中说明。
- `.down.sql` 主要用于可丢弃的本地/CI 数据库和经批准的 staging 演练，不是默认生产回滚手段。

创建 migration 时使用 `golang-migrate` 的顺序编号，名称描述一个原子变更：

```bash
cd server
go run -tags 'postgres' \
  github.com/golang-migrate/migrate/v4/cmd/migrate@v4.18.3 \
  create -ext sql -dir migrations -seq add_example_column
```

提交前确认新编号未与主分支冲突，并同时评审 up、down、数据兼容性和回滚策略。

## 日常命令

```bash
cd server

# 应用全部尚未执行的 migration
DATABASE_URL='postgres://...' make migrate-up

# 查看当前版本
DATABASE_URL='postgres://...' make migrate-status

# 仅可丢弃的本地/CI 数据库：回退最新一个 migration
DATABASE_URL='postgres://...' make migrate-down-one

# 可丢弃数据库：应用全部 -> 回退最新一个 -> 重新应用最新一个
DATABASE_URL='postgres://...' make migrate-verify

# 仅开发库：加载示例数据
DATABASE_URL='postgres://...' make seed-dev
```

## Docker Compose 行为

- `docker compose up -d`
  - 默认 profile **不包含**迁移服务，需要手动运行迁移或使用带 profile 的启动命令
- `docker compose --profile dev-full up -d`
  - 会自动启动 `migrate-dev` 服务并按版本应用全部 pending migrations
  - 会额外执行 `seed-dev` 一次性服务
- `docker compose -f docker-compose.yml -f docker-compose.observability.yml -f docker-compose.prod.yml --profile prod up -d`
  - 会自动启动 `migrate` 服务并按版本应用全部 pending migrations
  - **不会**执行任何 seed

## staging / production 执行前检查

1. 确认目标环境 `DATABASE_URL` 指向正确数据库
2. 执行一次备份（见 [backup-and-restore.md](backup-and-restore.md)）
3. 记录当前 `make migrate-status` 版本，并确认 migration 状态不是 dirty
4. 评审本次新增 migration 是否包含：
   - 删除列 / 删除表
   - 大表回填
   - 唯一索引新增
   - 非空约束收紧
5. 如果包含长事务、锁表或破坏性变更，先在 staging 使用生产量级数据演练升级和应用兼容性
6. 确认应用回滚时仍可运行在迁移后的 schema 上；否则必须先拆成 expand / contract 多阶段发布

## production 执行顺序

1. 备份数据库
2. 记录当前 migration 版本和待执行文件
3. 按发布编排启动 `migrate` 服务；它只会应用 pending migrations，并在 dirty 状态下拒绝继续
4. migration 成功后启动或切换 `app`、`frontend`、`admin`
5. 等待 readiness 健康并执行 `./infra/ops/smoke-check.sh`
6. 记录迁移前后版本、执行时间、备份锚点和验证结果

## 回滚规则

### 可丢弃的本地 / CI 数据库

可以执行：

```bash
cd server
DATABASE_URL='postgres://...' make migrate-down-one
```

### production

默认**不要**直接执行 `down`，镜像回滚也不会自动回退 schema。生产处置优先级：

1. 对向后兼容的 migration，保留新 schema 并切回上一版镜像 / `TAG`
2. 优先新增一个前向修正 migration，而不是改写或强制回退已执行文件
3. 只有在已评审 `.down.sql`、数据损失范围和应用兼容性后，才可受控回退单个版本
4. 需要恢复备份时，按恢复 runbook 在隔离环境验证后执行，并记录恢复点和数据损失窗口
5. 处置后重新运行 migration status、readiness 和 Smoke Check

## 验证清单

- `make migrate-status` 返回预期最新版本且不是 dirty
- 可丢弃数据库上的 `make migrate-verify` 成功完成“up all → down latest → up latest”
- `docker compose ps` 中 `migrate` 已退出且 exit code = 0
- `curl http://127.0.0.1:8080/health/ready` 返回 200
- 本次 migration 涉及的读写链路、关键页面和后台入口通过
