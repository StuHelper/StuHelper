# 快速开始

本文档帮助新开发者在 5 分钟内搭建完整的全栈开发环境。

采用**混合模式**：Docker 运行基础设施（PostgreSQL + Redis），本地运行后端和前端，兼顾隔离性和开发体验。

## 环境要求

| 工具 | 版本 | 安装 |
|------|------|------|
| Docker & Compose | 24+ | [docker.com](https://www.docker.com/) |
| Go | 1.23+ | `brew install go` |
| Node.js | 20+ | `brew install node` |

## 1. 克隆项目

```bash
git clone https://gitea.stuhelper.com/StuHelper/StuHelper.git
cd StuHelper
```

## 2. 启动 PostgreSQL + Redis

```bash
cd server/deployments
cp .env.example .env
```

编辑 `.env`，修改以下关键项（其余保持默认即可）：

```bash
# 数据库密码（自定义，两处保持一致）
POSTGRES_PASSWORD=dev123
DATABASE_URL=postgres://stuhelper:dev123@localhost:5432/stuhelper?sslmode=disable

# Redis 密码
REDIS_PASSWORD=dev123

# 本地开发必须改为 localhost（Go 进程在宿主机运行）
REDIS_HOST=localhost

# HMAC 密钥（开发环境随便填，≥32 字符）
HMAC_SECRET=dev-hmac-secret-at-least-32-chars!!

# Casdoor SSO（联系管理员获取，或先留空跳过认证功能）
CASDOOR_CLIENT_ID=
CASDOOR_CLIENT_SECRET=
```

> **注意**: `.env` 中 `DATABASE_URL` 的 host 是 `localhost`（不是 `postgres`），因为 Go 进程在宿主机运行，通过映射端口访问 Docker 内的 PG。

启动基础设施：

```bash
docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d
```

验证服务状态：

```bash
docker compose -f docker-compose.yml -f docker-compose.dev.yml ps
```

两个容器都应显示 `healthy`。数据库表会通过挂载的 `scripts/init.sql` 自动初始化。

## 3. 启动后端

```bash
cd ../../server   # 回到 server 目录
make run
# 或者: go run ./cmd/stuhelper
```

访问 http://localhost:8080/health 验证，返回 `{"status":"ok"}` 即成功。

API 交互式文档：http://localhost:8080/docs/ （Swagger UI）

## 4. 启动前端

新开一个终端：

```bash
cd clients/web
npm install
npm run dev
```

浏览器打开 http://localhost:5173 即可看到页面。

## 5. 加载种子数据（可选）

首次开发时可导入测试数据，方便调试：

```bash
cd server/deployments
docker compose -f docker-compose.yml -f docker-compose.dev.yml exec postgres \
  psql -U stuhelper -d stuhelper -f /docker-entrypoint-initdb.d/seed.sql
```

> 种子数据文件位于 `server/scripts/seed.sql`，包含院系、教师、课程和测评示例数据。
> 如果需要重新加载，先清空再导入即可。

## 日常开发流程

```bash
# 终端 1: 基础设施（首次启动后常驻，无需每次重启）
cd server/deployments
docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d

# 终端 2: 后端（修改 Go 代码后 Ctrl+C 重启）
cd server && make run

# 终端 3: 前端（Vite HMR 自动热更新）
cd clients/web && npm run dev
```

## 常用命令

### 后端

```bash
cd server
make run              # 运行后端
make test             # 运行测试
make lint             # 代码检查
make fmt              # 格式化代码
make build            # 构建二进制
make generate         # 重新生成 OpenAPI 相关代码（修改 API 规范后必须执行）
make lint-spec        # 验证 OpenAPI 规范
```

### 前端

```bash
cd clients/web
npm run dev           # 开发服务器
npm run build         # 生产构建
npm run type-check    # TypeScript 类型检查
npm run test          # 运行单元测试
npm run generate:types # 从 OpenAPI 规范生成 TS 类型
```

### 基础设施

```bash
cd server/deployments
# 以下命令均需指定两个 compose 文件
docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d      # 启动
docker compose -f docker-compose.yml -f docker-compose.dev.yml down       # 停止
docker compose -f docker-compose.yml -f docker-compose.dev.yml logs -f    # 查看日志
docker compose -f docker-compose.yml -f docker-compose.dev.yml down -v    # 停止并删除数据卷（重置数据库）
```

## 常见问题

### 端口冲突

- 后端默认 `8080`，前端默认 `5173`，PG `5432`，Redis `6379`
- 如果 8080 被占用，修改 `.env` 中的 `APP_PORT`

### 数据库连接失败

- 确认 Docker 容器状态为 `healthy`
- 确认 `.env` 中 `DATABASE_URL` 的 host 是 `localhost`，不是 `postgres`
- 确认密码一致：`POSTGRES_PASSWORD` 和 `DATABASE_URL` 中的密码

### 重置数据库

```bash
cd server/deployments
docker compose -f docker-compose.yml -f docker-compose.dev.yml down -v
docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d
```

删除数据卷后重新启动，`init.sql` 会自动重新执行。

## 相关文档

- 开发规范：`.project_rule/project_rules.md`
- 后端详细指南：`docs/guides/backend-quickstart.md`
- 模块文档：`docs/modules/`
- API 概览：`docs/reference/api-overview.md`
- OpenAPI 规范：`server/api/openapi.yaml`
- Swagger UI：http://localhost:8080/docs/ （开发环境）
