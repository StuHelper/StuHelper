# 快速开始

本文档帮助新开发者搭建完整的全栈开发环境。

提供两种开发模式：
- **混合模式（推荐）**：Docker 运行 PG + Redis，后端和前端在宿主机运行
- **全 Docker 模式**：所有服务都在 Docker 容器中运行，支持热重载

## 环境要求

### 混合模式

| 工具 | 版本 | 安装 |
|------|------|------|
| Docker & Compose | 24+ | [docker.com](https://www.docker.com/) |
| Go | 1.24+ | `brew install go` |
| Node.js | 24+ | `brew install node` |
| pnpm | 10+ | `npm install -g pnpm` |

### 全 Docker 模式

只需要 Docker & Compose。

## 1. 克隆项目

```bash
git clone https://gitea.stuhelper.com/StuHelper/StuHelper.git
cd StuHelper
```

## 2. 配置环境变量

```bash
cp .env.example .env
```

编辑 `.env`，修改以下关键项（其余保持默认即可）：

```bash
# 数据库密码（自定义，两处保持一致）
POSTGRES_PASSWORD=dev123
DATABASE_URL=postgres://stuhelper:dev123@localhost:5432/stuhelper?sslmode=disable

# Redis 密码
REDIS_PASSWORD=dev123

# Casdoor SSO（联系管理员获取，或先留空跳过认证功能）
CASDOOR_CLIENT_ID=
CASDOOR_CLIENT_SECRET=
```

## 3A. 混合模式开发（推荐）

### 启动基础设施

```bash
docker compose up -d
```

验证服务状态：

```bash
docker compose ps
```

两个容器都应显示 `healthy`。数据库会通过 `init.sql` + `seed.sql` 自动初始化。

### 启动后端

```bash
cd server
make run
```

访问 http://localhost:8080/health 验证，返回 `{"status":"ok"}` 即成功。

API 文档：http://localhost:8080/docs/

### 启动前端

新开一个终端：

```bash
cd clients/web/course
pnpm install
pnpm dev
```

浏览器打开 http://localhost:5173。

## 3B. 全 Docker 模式开发

无需修改 `.env`，compose 内部已自动配置容器间网络连接。

一键启动所有服务：

```bash
docker compose --profile dev-full up
```

- 后端使用 [air](https://github.com/air-verse/air) 热重载（修改 Go 代码自动重编译）
- 前端使用 pnpm dev（Vite HMR 热更新）
- 后端：http://localhost:8080
- 前端：http://localhost:5173

查看容器日志：

```bash
docker compose --profile dev-full logs -f app-dev     # 后端日志
docker compose --profile dev-full logs -f frontend-dev # 前端日志
```

## 日常开发流程

### 混合模式

```bash
# 终端 1: 基础设施（首次启动后常驻）
docker compose up -d

# 终端 2: 后端（修改 Go 代码后 Ctrl+C 重启）
cd server && make run

# 终端 3: 前端（Vite HMR 自动热更新）
cd clients/web/course && pnpm dev
```

### 全 Docker 模式

```bash
# 一键启动（后台运行）
docker compose --profile dev-full up -d

# 查看日志
docker compose --profile dev-full logs -f
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
make generate         # 重新生成 OpenAPI 代码
```

### 前端

```bash
cd clients/web/course
pnpm dev              # 开发服务器
pnpm build            # 生产构建
pnpm run type-check   # TypeScript 类型检查
pnpm run generate:types # 生成 TS 类型
```

### Docker

```bash
docker compose up -d          # 启动基础设施
docker compose down           # 停止
docker compose logs -f        # 查看日志
docker compose down -v        # 停止并删除数据卷（重置数据库）
```

## 常见问题

### 端口冲突

- 后端 `8080`，前端 `5173`，PG `5432`，Redis `6379`
- 如果端口被占用，修改 `.env` 中对应的配置

### 数据库连接失败

- 确认 Docker 容器状态为 `healthy`：`docker compose ps`
- 混合模式下 `DATABASE_URL` 的 host 必须是 `localhost`
- 确认 `POSTGRES_PASSWORD` 和 `DATABASE_URL` 中的密码一致

### 重置数据库

```bash
docker compose down -v
docker compose up -d
```

## 相关文档

- 开发规范：`.project_rule/project_rules.md`
- 部署指南：`docs/guides/deployment.md`
- API 概览：`docs/reference/api-overview.md`
- OpenAPI 规范：`server/api/openapi.yaml`
- Swagger UI：http://localhost:8080/docs/
