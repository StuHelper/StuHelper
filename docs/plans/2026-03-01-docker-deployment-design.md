# Docker 全栈部署重构设计

> 日期: 2026-03-01
> 状态: 已批准

## 背景

项目存在多套互相矛盾的部署系统：

- `server/deployments/` 下三个 docker-compose 文件（base + dev + prod 覆盖模式），但 base 里硬编码 `APP_ENV: production` 且引用不存在的 `stuhelper:latest` 镜像
- `deploy/deploy.sh` — SSH+rsync 裸金属部署，与 Docker 无关
- `server/Dockerfile` 存在但无人使用（没有 CI/CD 推镜像）
- `server/Makefile` 的 build target 与 deploy.sh 的构建逻辑不复用
- `.env` 同时供 Docker Compose 容器变量和本地 Go 进程使用，职责混乱
- `quick-start.md` 指引的命令会因 `stuhelper:latest` 不存在而失败

## 目标

1. **统一为 Docker 部署**：开发用 Docker 起基础设施（可选全容器化），生产强制全 Docker
2. **使用自建 registry** `registry.stuhelper.com` 管理镜像
3. **删除所有冗余部署脚本**，只保留一套方案
4. **CI/CD 自动化**：Gitea Actions 自动构建推镜像，SSH 触发服务器更新

## 决策

### 文件组织：单 docker-compose.yml + profiles

选择 Docker Compose profiles 在一个文件内区分开发/生产，理由：
- 服务数量少（PG、Redis、app、frontend 四个），一个文件能管住
- profiles 是 Docker 原生特性，避免三文件覆盖的复杂性
- 减少维护负担

### 删除清单

| 文件/目录 | 原因 |
|-----------|------|
| `deploy/` 整个目录 | SSH+rsync 方式废弃 |
| `server/deployments/docker-compose.yml` | 旧三文件模式 |
| `server/deployments/docker-compose.dev.yml` | 同上 |
| `server/deployments/docker-compose.prod.yml` | 同上 |
| `server/deployments/.env.example` | 旧模板 |
| `server/deployments/docker/` | 空目录 |
| `docs/guides/deploy-script.md` | deploy.sh 文档 |
| `docs/guides/deployment.md` | 旧部署文档（重写） |

### 新文件结构

```
/
├── docker-compose.yml          # 唯一 compose（profiles 控制）
├── .env.example                # 环境变量模板
├── .env                        # 实际配置（gitignore）
├── server/
│   ├── Dockerfile              # 后端镜像（优化现有）
│   └── Makefile                # 不动
├── clients/web/course/
│   └── Dockerfile              # 新增：前端 nginx 镜像
├── .gitea/workflows/
│   ├── ci.yml                  # 合并后端+前端 CI
│   └── cd.yml                  # 新增：CD 流水线
└── docs/
    ├── tutorials/quick-start.md # 重写
    └── guides/deployment.md     # 重写
```

### docker-compose.yml 设计

```yaml
services:
  # --- 基础设施（始终启动）---
  postgres:
    image: postgres:18-alpine
    # ... 端口、卷、健康检查

  redis:
    image: redis:8-alpine
    # ... 配置

  # --- 生产服务（--profile prod）---
  app:
    profiles: [prod]
    image: registry.stuhelper.com/stuhelper/backend:${TAG:-latest}
    # ... 环境变量、依赖

  frontend:
    profiles: [prod]
    image: registry.stuhelper.com/stuhelper/frontend:${TAG:-latest}
    # ... nginx 服务前端

  # --- 全 Docker 开发（--profile dev-full）---
  app-dev:
    profiles: [dev-full]
    build: ./server
    volumes: [./server:/app/src]
    # ... 开发模式配置
```

使用方式：
- `docker compose up` — 只起 PG+Redis（混合开发）
- `docker compose --profile dev-full up` — 全 Docker 开发
- `docker compose --profile prod up -d` — 生产部署

### 环境变量策略

单一 `.env` 文件分区管理：

```env
# === Docker Compose 公共 ===
POSTGRES_USER=stuhelper
POSTGRES_PASSWORD=xxx
REDIS_PASSWORD=xxx

# === 后端运行时 ===
APP_ENV=development
DATABASE_URL=postgres://...
HMAC_SECRET=xxx
# Casdoor 配置...

# === 部署相关 ===
TAG=latest
```

混合开发时 `DATABASE_URL` host 用 `localhost`，全 Docker 开发时用 `postgres`。

### CD 流水线

```
main 分支 push
  → ci.yml: lint + test + build 校验
  → cd.yml:
    1. docker build 后端 → push registry.stuhelper.com/stuhelper/backend:latest
    2. docker build 前端 → push registry.stuhelper.com/stuhelper/frontend:latest
    3. SSH 到服务器:
       docker compose --profile prod pull
       docker compose --profile prod up -d
```

### 前端 Dockerfile

```dockerfile
# Build stage
FROM node:24-alpine AS builder
WORKDIR /app
COPY package.json pnpm-lock.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile
COPY . .
RUN pnpm build

# Serve stage
FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
```

## 不变的部分

- `server/Makefile` — 开发工具，不动
- `.gitea/workflows/` 下现有 CI 逻辑 — 扩展而非替换
- `server/Dockerfile` 基本结构 — 优化但不重写
- `clients/web/course/.env.example` — Vite 前端配置，不动
