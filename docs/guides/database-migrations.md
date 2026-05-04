---
type: guide
audience: backend-dev, ops
status: current
authoritative-source: server/migrations/000001_initial_schema.up.sql
last-verified: 2026-05-04
---

# 数据库初始化运行手册

## 权威文件

- `server/migrations/000001_initial_schema.up.sql`：当前 PostgreSQL 全量初始化 schema
- `server/migrations/000001_initial_schema.down.sql`：破坏性本地 reset
- `server/scripts/seed.sql`：仅开发环境示例数据

## 维护规则

- 当前项目按绿地 schema 管理，不保留增量迁移兼容链路。
- 结构变更直接更新 `000001_initial_schema.up.sql`。
- `down` 文件只用于本地/测试 reset，不作为生产回滚手段。

## 日常命令

```bash
cd server

# 初始化当前 schema
DATABASE_URL='postgres://...' make migrate-up

# 查看当前版本
DATABASE_URL='postgres://...' make migrate-status

# 仅本地可用：删除当前 schema baseline
DATABASE_URL='postgres://...' make migrate-down-one

# 仅开发库：加载示例数据
DATABASE_URL='postgres://...' make seed-dev
```

## Docker Compose 行为

- `docker compose up -d`
  - 默认 profile **不包含**迁移服务，需要手动运行迁移或使用带 profile 的启动命令
- `docker compose --profile dev-full up -d`
  - 会自动启动 `migrate-dev` 服务并应用当前 baseline schema
  - 会额外执行 `seed-dev` 一次性服务
- `docker compose -f docker-compose.yml -f docker-compose.observability.yml -f docker-compose.prod.yml --profile prod up -d`
  - 会自动启动 `migrate` 服务并应用当前 baseline schema
  - **不会**执行任何 seed

## staging / production 初始化前检查

1. 确认目标环境 `DATABASE_URL` 指向正确数据库
2. 执行一次备份（见 [backup-and-restore.md](backup-and-restore.md)）
3. 评审本次 schema 变更是否包含：
   - 删除列 / 删除表
   - 大表回填
   - 唯一索引新增
   - 非空约束收紧
4. 如果包含破坏性变更，先在 staging 演练完整发布

## production 执行顺序

1. 备份数据库
2. 发布代码 / 镜像
3. 执行当前 baseline schema 初始化（CI/CD 中由 `migrate` 服务自动完成）
4. 等待 `app`、`frontend`、`admin` 健康
5. 执行 `./infra/ops/smoke-check.sh`
6. 记录 schema 版本、执行时间、验证结果

## 回滚规则

### 本地

可以执行：

```bash
cd server
DATABASE_URL='postgres://...' make migrate-down-one
```

### production

默认 **不要** 直接执行 `down`。生产回滚优先级：

1. 切回上一版镜像 / `TAG`
2. 如果本次 schema 变更是破坏性变更，再使用备份恢复数据库
3. 恢复后重新执行 Smoke Check

## 验证清单

- `make migrate-status` 能返回当前版本
- `docker compose ps` 中 `migrate` 已退出且 exit code = 0
- `curl http://127.0.0.1:8080/health/ready` 返回 200
- 关键页面和后台首页可访问
