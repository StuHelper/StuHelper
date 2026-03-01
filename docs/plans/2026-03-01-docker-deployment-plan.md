# Docker 全栈部署重构 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 删除所有冗余部署基础设施，统一为单 docker-compose.yml + profiles 模式，支持混合开发、全 Docker 开发和全 Docker 生产部署，CI/CD 自动推镜像到 registry.stuhelper.com。

**Architecture:** 单个根目录 `docker-compose.yml` 使用 Docker Compose profiles 区分三种模式：默认（PG+Redis 基础设施）、dev-full（全容器开发）、prod（生产部署从 registry 拉预构建镜像）。Gitea Actions 负责 CI 校验和 CD 构建/推送/部署。

**Tech Stack:** Docker Compose v2 (profiles), Docker BuildKit, Gitea Actions, nginx (前端容器), Go 1.24, Node 24 + pnpm

---

### Task 1: 删除冗余部署文件

**Files:**
- Delete: `deploy/deploy.sh`
- Delete: `deploy/deploy.env.example`
- Delete: `server/deployments/docker-compose.yml`
- Delete: `server/deployments/docker-compose.dev.yml`
- Delete: `server/deployments/docker-compose.prod.yml`
- Delete: `server/deployments/.env.example`
- Delete: `server/deployments/docker/` (entire directory)
- Delete: `docs/guides/deploy-script.md`
- Delete: `docs/guides/deployment.md`

**Step 1: 删除 deploy/ 目录**

```bash
rm -rf deploy/
```

**Step 2: 删除旧 compose 文件和 env 模板**

```bash
rm server/deployments/docker-compose.yml
rm server/deployments/docker-compose.dev.yml
rm server/deployments/docker-compose.prod.yml
rm server/deployments/.env.example
rm -rf server/deployments/docker/
```

**Step 3: 删除旧部署文档**

```bash
rm docs/guides/deploy-script.md
rm docs/guides/deployment.md
```

**Step 4: 验证删除**

Run: `git status`
Expected: 上述文件全部出现在 "deleted" 列表中，不能有遗漏

**Step 5: Commit**

```bash
git add -A
git commit -m "chore: remove legacy deployment scripts and compose files"
```

---

### Task 2: 创建根目录 .env.example

**Files:**
- Create: `.env.example` (项目根目录)

**Step 1: 创建 .env.example**

参考旧的 `server/deployments/.env.example` 和 `server/deployments/.env`，结合 `server/internal/pkg/config/config.go` 中实际使用的环境变量名。

```env
# StuHelper 环境配置
# 复制为 .env 并填入实际值: cp .env.example .env

# ==================== Docker Compose 公共 ====================
POSTGRES_USER=stuhelper
POSTGRES_PASSWORD=dev123
POSTGRES_DB=stuhelper
REDIS_PASSWORD=dev123

# ==================== 数据库配置 ====================
# 混合开发（Go 在宿主机运行）: host = localhost
# 全 Docker 开发 / 生产: host = postgres
DATABASE_URL=postgres://stuhelper:dev123@localhost:5432/stuhelper?sslmode=disable

# 连接池
DB_MAX_CONNS=20
DB_MIN_CONNS=2
DB_MAX_CONN_LIFETIME=30
DB_MAX_CONN_IDLE_TIME=5
DB_QUERY_TIMEOUT=5

# SSL（生产环境建议 verify-full）
DB_SSL_MODE=disable
# DB_SSL_ROOT_CERT=
# DB_SSL_CERT=
# DB_SSL_KEY=

# ==================== Redis 配置 ====================
# 混合开发: localhost / 全 Docker 开发 / 生产: redis
REDIS_HOST=localhost
REDIS_PORT=6379

# ==================== 应用配置 ====================
APP_ENV=development
APP_PORT=8080
LOG_LEVEL=debug
LOG_FORMAT=console
LOG_OUTPUT=stdout

# ==================== 安全配置 ====================
CORS_ORIGINS=http://localhost:5173
TRUSTED_PROXIES=
# HMAC 密钥（>= 32 字符，生产环境必须用强密钥）
# 生成命令: openssl rand -hex 32
HMAC_SECRET=dev_hmac_secret_change_in_production_32ch

# Prometheus Metrics（生产环境必须配置密码）
METRICS_USER=prometheus
METRICS_PASSWORD=

# ==================== Casdoor SSO 配置 ====================
CASDOOR_ENDPOINT=https://sso.stuhelper.com
CASDOOR_CLIENT_ID=
CASDOOR_CLIENT_SECRET=
CASDOOR_ORGANIZATION=stuhelper
CASDOOR_APPLICATION=stuhelper
CASDOOR_REDIRECT_URI=http://localhost:5173/auth/callback
# 二选一：证书内容（适合容器）或文件路径（适合本地开发）
CASDOOR_CERTIFICATE=
CASDOOR_CERTIFICATE_FILE=configs/certs/casdoor.pem

# ==================== Token 配置 ====================
TOKEN_ACCESS_TTL=900
TOKEN_REFRESH_TTL=604800
TOKEN_COOKIE_SECURE=false
TOKEN_COOKIE_DOMAIN=

# ==================== 部署相关 ====================
# Docker 镜像 tag（CI 自动设置，开发者一般不需要改）
TAG=latest
```

**Step 2: 验证文件创建**

Run: `cat .env.example | head -5`
Expected: 看到文件头部注释

**Step 3: Commit**

```bash
git add .env.example
git commit -m "chore: add root .env.example for unified config"
```

---

### Task 3: 更新 .gitignore

**Files:**
- Modify: `.gitignore`

**Step 1: 更新 .gitignore**

在 "环境与密钥" 部分：
- 删除 `deploy/deploy.env`（目录已删）
- 确保根目录 `.env` 已覆盖
- 确保 `server/deployments/.env` 已覆盖（保留兼容性或删除旧条目）

旧内容（第18-25行）:
```
.env
.env.*
!.env.example
deploy/deploy.env
credentials.json
secrets.yaml
config.local.*
```

新内容:
```
.env
.env.*
!.env.example
credentials.json
secrets.yaml
config.local.*
```

就是删除 `deploy/deploy.env` 行，因为 deploy/ 目录已经不存在了。

**Step 2: 验证 .gitignore**

Run: `grep -n "deploy" .gitignore`
Expected: 不应再出现 deploy 相关行

**Step 3: Commit**

```bash
git add .gitignore
git commit -m "chore: clean up .gitignore after deploy/ removal"
```

---

### Task 4: 更新 main.go 中的 .env 加载路径

**Files:**
- Modify: `server/cmd/stuhelper/main.go:53`

**Step 1: 修改 godotenv 加载路径**

旧代码 (`server/cmd/stuhelper/main.go:52-56`):
```go
if os.Getenv("APP_ENV") != "production" && os.Getenv("GIN_MODE") != "release" {
    if err := godotenv.Load("deployments/.env"); err != nil {
        fmt.Fprintf(os.Stderr, "debug: .env not loaded: %v\n", err)
    }
}
```

新代码:
```go
if os.Getenv("APP_ENV") != "production" && os.Getenv("GIN_MODE") != "release" {
    if err := godotenv.Load("../.env"); err != nil {
        fmt.Fprintf(os.Stderr, "debug: .env not loaded: %v\n", err)
    }
}
```

路径从 `deployments/.env` 改为 `../.env`，因为 `go run` 的工作目录是 `server/`，根目录的 `.env` 相对路径为 `../.env`。

**Step 2: 验证修改**

Run: `cd server && grep -n 'godotenv.Load' cmd/stuhelper/main.go`
Expected: 显示 `../.env`

**Step 3: 验证编译**

Run: `cd server && go build ./cmd/stuhelper`
Expected: 编译成功，无错误

**Step 4: Commit**

```bash
git add server/cmd/stuhelper/main.go
git commit -m "fix: update .env path from deployments/.env to ../.env"
```

---

### Task 5: 创建根目录 docker-compose.yml

**Files:**
- Create: `docker-compose.yml` (项目根目录)

**Step 1: 创建 docker-compose.yml**

```yaml
# StuHelper Docker Compose
#
# 使用方式:
#   开发（仅基础设施）:  docker compose up
#   全 Docker 开发:      docker compose --profile dev-full up
#   生产部署:            docker compose --profile prod up -d

services:
  # ─── 基础设施（始终启动）───────────────────────────
  postgres:
    image: postgres:18-alpine
    container_name: stuhelper-postgres
    environment:
      POSTGRES_USER: ${POSTGRES_USER:-stuhelper}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:?POSTGRES_PASSWORD is required}
      POSTGRES_DB: ${POSTGRES_DB:-stuhelper}
    ports:
      - "127.0.0.1:5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./server/scripts/init.sql:/docker-entrypoint-initdb.d/01-init.sql:ro
      - ./server/scripts/seed.sql:/docker-entrypoint-initdb.d/02-seed.sql:ro
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U $${POSTGRES_USER:-stuhelper}"]
      interval: 5s
      timeout: 5s
      retries: 5
    restart: unless-stopped

  redis:
    image: redis:8-alpine
    container_name: stuhelper-redis
    command: >-
      redis-server
      --appendonly yes
      --maxmemory 256mb
      --maxmemory-policy volatile-lru
      --requirepass ${REDIS_PASSWORD:?REDIS_PASSWORD is required}
    ports:
      - "127.0.0.1:6379:6379"
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD-SHELL", "redis-cli -a $${REDIS_PASSWORD} ping | grep -q PONG"]
      interval: 5s
      timeout: 5s
      retries: 5
    restart: unless-stopped

  # ─── 生产服务（--profile prod）──────────────────────
  app:
    profiles: [prod]
    image: registry.stuhelper.com/stuhelper/backend:${TAG:-latest}
    container_name: stuhelper-app
    env_file: .env
    environment:
      APP_ENV: production
      DATABASE_URL: postgres://${POSTGRES_USER:-stuhelper}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB:-stuhelper}?sslmode=${DB_SSL_MODE:-disable}
      REDIS_HOST: redis
      REDIS_PORT: "6379"
    ports:
      - "127.0.0.1:8080:8080"
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/health/live"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 10s
    restart: always
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 1G

  frontend:
    profiles: [prod]
    image: registry.stuhelper.com/stuhelper/frontend:${TAG:-latest}
    container_name: stuhelper-frontend
    ports:
      - "127.0.0.1:3000:80"
    depends_on:
      app:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:80/"]
      interval: 10s
      timeout: 5s
      retries: 3
    restart: always

  # ─── 全 Docker 开发（--profile dev-full）────────────
  app-dev:
    profiles: [dev-full]
    build:
      context: ./server
      dockerfile: Dockerfile
    container_name: stuhelper-app-dev
    env_file: .env
    environment:
      APP_ENV: development
      DATABASE_URL: postgres://${POSTGRES_USER:-stuhelper}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB:-stuhelper}?sslmode=disable
      REDIS_HOST: redis
      REDIS_PORT: "6379"
    ports:
      - "127.0.0.1:8080:8080"
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/health/live"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 15s

volumes:
  postgres_data:
  redis_data:

networks:
  default:
    name: stuhelper
    driver: bridge
```

**Step 2: 验证 compose 语法**

Run: `docker compose config --profiles prod 2>&1 | head -20`
Expected: 输出有效的 YAML，没有语法错误（需要 .env 文件存在才能完全通过）

Run: `cp .env.example .env && docker compose config 2>&1 | head -5`
Expected: services 列表中只有 postgres 和 redis

**Step 3: Commit**

```bash
git add docker-compose.yml
git commit -m "feat: add unified docker-compose.yml with profiles"
```

---

### Task 6: 创建前端 Dockerfile 和 nginx 配置

**Files:**
- Create: `clients/web/course/Dockerfile`
- Create: `clients/web/course/nginx.conf`
- Create: `clients/web/course/.dockerignore`

**Step 1: 创建前端 Dockerfile**

```dockerfile
# Build stage
FROM node:24-alpine AS builder

WORKDIR /app

# 启用 pnpm
RUN corepack enable && corepack prepare pnpm@latest --activate

# 先复制依赖描述文件，利用缓存
COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

# 复制源码并构建
COPY . .
RUN pnpm build

# Serve stage
FROM nginx:1.27-alpine

# 复制构建产物
COPY --from=builder /app/dist /usr/share/nginx/html

# 复制 nginx 配置
COPY nginx.conf /etc/nginx/conf.d/default.conf

# 非 root 用户运行
RUN chown -R nginx:nginx /usr/share/nginx/html && \
    chown -R nginx:nginx /var/cache/nginx && \
    chown -R nginx:nginx /var/log/nginx && \
    touch /var/run/nginx.pid && \
    chown -R nginx:nginx /var/run/nginx.pid

USER nginx

EXPOSE 80

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:80/ || exit 1
```

**Step 2: 创建 nginx.conf**

```nginx
server {
    listen 80;
    server_name _;
    root /usr/share/nginx/html;
    index index.html;

    # SPA 路由：所有非文件请求 fallback 到 index.html
    location / {
        try_files $uri $uri/ /index.html;
    }

    # 静态资源缓存（Vite 构建带 hash）
    location /assets/ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }

    # 后端 API 反向代理
    location /api/ {
        proxy_pass http://stuhelper-app:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # 健康检查端点代理
    location /health {
        proxy_pass http://stuhelper-app:8080;
    }

    # Swagger UI 代理（开发用）
    location /docs/ {
        proxy_pass http://stuhelper-app:8080;
    }

    # 安全头
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;

    # gzip
    gzip on;
    gzip_types text/plain text/css application/json application/javascript text/xml application/xml text/javascript image/svg+xml;
    gzip_min_length 256;
}
```

**Step 3: 创建 .dockerignore**

```
node_modules/
dist/
.env
.env.*
*.log
.DS_Store
```

**Step 4: 验证 Dockerfile 语法**

Run: `cd clients/web/course && docker build --check .`
Expected: 无语法错误（或者如果 `--check` 不支持，用 `docker build --dry-run .` 或直接构建测试）

**Step 5: Commit**

```bash
git add clients/web/course/Dockerfile clients/web/course/nginx.conf clients/web/course/.dockerignore
git commit -m "feat: add frontend Dockerfile with nginx"
```

---

### Task 7: 优化后端 Dockerfile

**Files:**
- Modify: `server/Dockerfile`

**Step 1: 更新 Dockerfile**

优化点：
- alpine 版本从 3.19 升级到 3.21
- 添加 `.dockerignore`
- 注入构建版本信息（version/commit/buildTime）

更新 `server/Dockerfile`:

```dockerfile
# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# 构建参数：版本信息由 CI 注入
ARG TARGETARCH
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_TIME=unknown

RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build \
    -ldflags="-s -w -X main.version=${VERSION} -X main.gitCommit=${GIT_COMMIT} -X main.buildTime=${BUILD_TIME}" \
    -o /app/bin/stuhelper \
    ./cmd/stuhelper

# Final stage
FROM alpine:3.21

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/bin/stuhelper /app/stuhelper

RUN adduser -D -u 1000 appuser

RUN mkdir -p /app/tmp && chown appuser:appuser /app/tmp && chmod 700 /app/tmp

RUN chmod -R 555 /app && chmod 700 /app/tmp

ENV TMPDIR=/app/tmp

USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/stuhelper"]
```

**Step 2: 创建后端 .dockerignore**

创建 `server/.dockerignore`:

```
bin/
build/
deployments/
*.test
*.out
.DS_Store
.env
.env.*
```

**Step 3: 验证编译**

Run: `cd server && docker build -t stuhelper-test .`
Expected: 构建成功

**Step 4: Commit**

```bash
git add server/Dockerfile server/.dockerignore
git commit -m "feat: optimize backend Dockerfile with build args and .dockerignore"
```

---

### Task 8: 创建 CD 工作流

**Files:**
- Create: `.gitea/workflows/cd.yml`

注意：此文件放在项目根的 `.gitea/workflows/`，不是 `server/.gitea/workflows/`。已有的 CI 工作流分别在 `server/.gitea/workflows/ci.yml` 和 `clients/web/course/.gitea/workflows/ci.yml`，保持不变。

**Step 1: 创建 .gitea/workflows/cd.yml**

```yaml
name: CD

on:
  push:
    branches: [main]

env:
  REGISTRY: registry.stuhelper.com
  BACKEND_IMAGE: registry.stuhelper.com/stuhelper/backend
  FRONTEND_IMAGE: registry.stuhelper.com/stuhelper/frontend

jobs:
  build-and-push-backend:
    name: Build & Push Backend
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Login to registry
        uses: docker/login-action@v3
        with:
          registry: ${{ env.REGISTRY }}
          username: ${{ secrets.REGISTRY_USERNAME }}
          password: ${{ secrets.REGISTRY_PASSWORD }}

      - name: Extract metadata
        id: meta
        run: |
          echo "sha_short=$(git rev-parse --short HEAD)" >> "$GITHUB_OUTPUT"
          echo "build_time=$(date -u +'%Y-%m-%dT%H:%M:%SZ')" >> "$GITHUB_OUTPUT"

      - name: Build and push backend
        uses: docker/build-push-action@v6
        with:
          context: ./server
          push: true
          tags: |
            ${{ env.BACKEND_IMAGE }}:latest
            ${{ env.BACKEND_IMAGE }}:${{ steps.meta.outputs.sha_short }}
          build-args: |
            VERSION=${{ steps.meta.outputs.sha_short }}
            GIT_COMMIT=${{ github.sha }}
            BUILD_TIME=${{ steps.meta.outputs.build_time }}
          cache-from: type=registry,ref=${{ env.BACKEND_IMAGE }}:buildcache
          cache-to: type=registry,ref=${{ env.BACKEND_IMAGE }}:buildcache,mode=max

  build-and-push-frontend:
    name: Build & Push Frontend
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Login to registry
        uses: docker/login-action@v3
        with:
          registry: ${{ env.REGISTRY }}
          username: ${{ secrets.REGISTRY_USERNAME }}
          password: ${{ secrets.REGISTRY_PASSWORD }}

      - name: Extract metadata
        id: meta
        run: |
          echo "sha_short=$(git rev-parse --short HEAD)" >> "$GITHUB_OUTPUT"

      - name: Build and push frontend
        uses: docker/build-push-action@v6
        with:
          context: ./clients/web/course
          push: true
          tags: |
            ${{ env.FRONTEND_IMAGE }}:latest
            ${{ env.FRONTEND_IMAGE }}:${{ steps.meta.outputs.sha_short }}
          cache-from: type=registry,ref=${{ env.FRONTEND_IMAGE }}:buildcache
          cache-to: type=registry,ref=${{ env.FRONTEND_IMAGE }}:buildcache,mode=max

  deploy:
    name: Deploy to Production
    runs-on: ubuntu-latest
    needs: [build-and-push-backend, build-and-push-frontend]
    steps:
      - name: Deploy via SSH
        uses: appleboy/ssh-action@v1
        with:
          host: ${{ secrets.DEPLOY_HOST }}
          username: ${{ secrets.DEPLOY_USER }}
          key: ${{ secrets.DEPLOY_SSH_KEY }}
          port: ${{ secrets.DEPLOY_PORT }}
          script: |
            cd ${{ secrets.DEPLOY_APP_DIR }}
            docker compose --profile prod pull
            docker compose --profile prod up -d --remove-orphans
            echo "Waiting for health check..."
            sleep 5
            docker compose --profile prod ps
```

**Step 2: 验证 YAML 语法**

Run: `python3 -c "import yaml; yaml.safe_load(open('.gitea/workflows/cd.yml'))"`
Expected: 无报错

**Step 3: Commit**

```bash
git add .gitea/workflows/cd.yml
git commit -m "feat: add CD workflow for Docker build, push and deploy"
```

---

### Task 9: 重写 quick-start.md

**Files:**
- Modify: `docs/tutorials/quick-start.md`

**Step 1: 重写文档**

```markdown
# 快速开始

本文档帮助新开发者搭建完整的全栈开发环境。

提供两种开发模式：
- **混合模式（推荐）**：Docker 运行 PG + Redis，后端和前端在宿主机运行
- **全 Docker 模式**：所有服务都在 Docker 容器中运行

## 环境要求

| 工具 | 版本 | 安装 |
|------|------|------|
| Docker & Compose | 24+ | [docker.com](https://www.docker.com/) |
| Go | 1.24+ | `brew install go` |
| Node.js | 24+ | `brew install node` |
| pnpm | 10+ | `npm install -g pnpm` |

> 全 Docker 模式只需要 Docker。

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

## 3A. 混合模式（推荐）

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

## 3B. 全 Docker 模式

修改 `.env` 中的连接地址：

```bash
DATABASE_URL=postgres://stuhelper:dev123@postgres:5432/stuhelper?sslmode=disable
REDIS_HOST=redis
```

启动所有服务：

```bash
docker compose --profile dev-full up
```

后端在 http://localhost:8080，前端需要另外在宿主机启动或等待前端容器化开发支持。

## 日常开发流程

```bash
# 终端 1: 基础设施（首次启动后常驻）
docker compose up -d

# 终端 2: 后端（修改 Go 代码后 Ctrl+C 重启）
cd server && make run

# 终端 3: 前端（Vite HMR 自动热更新）
cd clients/web/course && pnpm dev
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

### 基础设施

```bash
docker compose up -d          # 启动
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
```

**Step 2: 验证文档**

Run: 检查文档中的所有命令路径是否正确

**Step 3: Commit**

```bash
git add docs/tutorials/quick-start.md
git commit -m "docs: rewrite quick-start.md for unified docker-compose"
```

---

### Task 10: 重写 deployment.md

**Files:**
- Create: `docs/guides/deployment.md`

**Step 1: 创建新的部署文档**

```markdown
# 生产环境部署指南

StuHelper 生产环境使用 Docker Compose 部署，镜像通过 Gitea Actions CI/CD 自动构建并推送到 `registry.stuhelper.com`。

## 架构

```
registry.stuhelper.com
  ├── stuhelper/backend:latest    # Go API 服务
  └── stuhelper/frontend:latest   # Nginx + Vue SPA

服务器 (docker compose --profile prod)
  ├── postgres    # PostgreSQL 18
  ├── redis       # Redis 8
  ├── app         # 后端 (从 registry 拉取)
  └── frontend    # 前端 (从 registry 拉取)
```

## 首次部署

### 1. 服务器准备

```bash
# 安装 Docker
curl -fsSL https://get.docker.com | sh

# 创建部署目录
mkdir -p /opt/stuhelper && cd /opt/stuhelper

# 从仓库获取必要文件
# 只需要 docker-compose.yml 和 .env
curl -o docker-compose.yml <raw-url>/docker-compose.yml
cp .env.example .env
```

### 2. 配置环境变量

编辑 `.env`，填入生产配置：

```bash
# 必须修改的项
POSTGRES_PASSWORD=<strong-password>
REDIS_PASSWORD=<strong-password>
DATABASE_URL=postgres://stuhelper:<strong-password>@postgres:5432/stuhelper?sslmode=disable
APP_ENV=production
HMAC_SECRET=<openssl rand -hex 32>

# Casdoor
CASDOOR_ENDPOINT=https://sso.stuhelper.com
CASDOOR_CLIENT_ID=<from-casdoor-admin>
CASDOOR_CLIENT_SECRET=<from-casdoor-admin>
CASDOOR_REDIRECT_URI=https://course.stuhelper.com/auth/callback
CASDOOR_CERTIFICATE=<pem-content>

# 安全
TOKEN_COOKIE_SECURE=true
CORS_ORIGINS=https://course.stuhelper.com
TRUSTED_PROXIES=127.0.0.1/32
METRICS_PASSWORD=<strong-password>
```

### 3. 登录 Registry 并启动

```bash
docker login registry.stuhelper.com
docker compose --profile prod pull
docker compose --profile prod up -d
```

### 4. 验证

```bash
docker compose --profile prod ps     # 所有服务 healthy
curl http://localhost:8080/health     # 后端健康检查
curl http://localhost:3000/           # 前端页面
```

## CI/CD 自动部署

合并到 `main` 分支时，Gitea Actions 自动执行：

1. 构建后端 Docker 镜像 → push 到 registry
2. 构建前端 Docker 镜像 → push 到 registry
3. SSH 到服务器执行 `docker compose --profile prod pull && up -d`

### 所需 Secrets（Gitea 仓库设置）

| Secret | 说明 |
|--------|------|
| `REGISTRY_USERNAME` | Registry 用户名 |
| `REGISTRY_PASSWORD` | Registry 密码 |
| `DEPLOY_HOST` | 服务器 IP |
| `DEPLOY_PORT` | SSH 端口 |
| `DEPLOY_USER` | SSH 用户 |
| `DEPLOY_SSH_KEY` | SSH 私钥 |
| `DEPLOY_APP_DIR` | 部署目录路径 |

## 手动更新

```bash
cd /opt/stuhelper
docker compose --profile prod pull
docker compose --profile prod up -d --remove-orphans
```

## 回滚

```bash
# 回滚到指定版本（用 git commit short hash）
TAG=abc1234 docker compose --profile prod up -d
```

## 数据库安全

生产环境数据库 SSL 配置详见 `.env.example` 中 `DB_SSL_*` 相关说明。

建议使用 `verify-full` 模式：

```bash
DB_SSL_MODE=verify-full
DB_SSL_ROOT_CERT=/etc/ssl/certs/pg-ca.crt
```

## 相关文档

- [快速开始](../tutorials/quick-start.md) — 开发环境
- [错误码参考](../reference/error-codes.md)
```

**Step 2: Commit**

```bash
git add docs/guides/deployment.md
git commit -m "docs: rewrite deployment guide for Docker-first approach"
```

---

### Task 11: 清理 server/deployments/ 残留

**Files:**
- Check: `server/deployments/` 目录状态

**Step 1: 检查残留**

Task 1 删除了 compose 文件、.env.example 和 docker/ 目录。检查 `server/deployments/` 目录下是否还有文件。

Run: `ls -la server/deployments/`
Expected: 可能还有 `.env`（实际开发配置，在 gitignore 里）和 `.DS_Store`

**Step 2: 清理**

如果 `server/deployments/` 目录只剩 `.env`（gitignored）和 `.DS_Store`：
- `.env` 文件：提醒用户手动迁移内容到根目录 `.env`，然后删除
- 如果目录为空（tracked 文件角度），考虑删除整个目录

Run: `git ls-files server/deployments/`
Expected: 应该没有 tracked 文件了（compose 和 .env.example 已在 Task 1 删除）

如果没有 tracked 文件，可以安全删除目录：
```bash
rm -rf server/deployments/
```

**Step 3: Commit（如果有改动）**

```bash
git add -A
git commit -m "chore: remove empty server/deployments/ directory"
```

---

### Task 12: 更新 archiving.md

**Files:**
- Modify: `.project_rule/archiving.md`

**Step 1: 在文件顶部（日期最新的位置）添加记录**

在 `## 2026-03-01` 部分之后（或更新已有的 2026-03-01 条目）添加：

```markdown
### Docker 全栈部署重构

- **关键文件**:
  - 新增: `docker-compose.yml`（根目录，profiles 模式）
  - 新增: `.env.example`（根目录）
  - 新增: `clients/web/course/Dockerfile`、`nginx.conf`
  - 新增: `.gitea/workflows/cd.yml`（CI/CD 自动部署）
  - 修改: `server/Dockerfile`（优化 + 构建参数）
  - 修改: `server/cmd/stuhelper/main.go`（.env 路径更新）
  - 删除: `deploy/` 整个目录（SSH+rsync 废弃）
  - 删除: `server/deployments/` 下旧 compose 文件和模板
  - 删除: `docs/guides/deploy-script.md`
- **决策原因**: 原项目存在多套互相矛盾的部署系统（docker-compose 三文件覆盖 + SSH rsync 脚本 + 未使用的 Dockerfile），统一为单 docker-compose.yml + profiles 模式。开发支持混合模式（PG/Redis 容器 + 宿主机 Go）和全 Docker 模式；生产强制全 Docker，镜像推送到 registry.stuhelper.com。
- **注意事项**:
  - `go run` 时 .env 加载路径改为 `../.env`（从 server/ 目录出发）
  - 生产部署需在 Gitea 仓库配置 Secrets（REGISTRY_*, DEPLOY_*）
  - 前端 nginx 容器反向代理 `/api/` 到后端，生产环境不再需要外部 nginx 配置 API 代理
```

**Step 2: Commit**

```bash
git add .project_rule/archiving.md
git commit -m "docs: update archiving with Docker deployment refactor"
```

---

## 实施顺序总结

| Task | 描述 | 依赖 |
|------|------|------|
| 1 | 删除冗余文件 | 无 |
| 2 | 创建 .env.example | 无 |
| 3 | 更新 .gitignore | Task 1 |
| 4 | 更新 main.go .env 路径 | Task 2 |
| 5 | 创建 docker-compose.yml | Task 1, 2 |
| 6 | 创建前端 Dockerfile + nginx | 无 |
| 7 | 优化后端 Dockerfile | 无 |
| 8 | 创建 CD 工作流 | Task 6, 7 |
| 9 | 重写 quick-start.md | Task 5 |
| 10 | 重写 deployment.md | Task 5, 8 |
| 11 | 清理残留目录 | Task 1 |
| 12 | 更新 archiving.md | 全部完成后 |

可并行的任务组：
- 组 A: Task 1, 2, 6, 7（互不依赖）
- 组 B: Task 3, 4, 5（依赖组 A）
- 组 C: Task 8, 9, 10, 11（依赖组 B）
- 组 D: Task 12（最后执行）
