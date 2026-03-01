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
```

### 2. 获取配置文件

只需两个文件：

```bash
# 从仓库获取 docker-compose.yml 和 .env.example
# 方式 1: 直接下载
curl -o docker-compose.yml https://gitea.stuhelper.com/StuHelper/StuHelper/raw/branch/main/docker-compose.yml
curl -o .env.example https://gitea.stuhelper.com/StuHelper/StuHelper/raw/branch/main/.env.example

# 方式 2: clone 后复制
git clone --depth 1 https://gitea.stuhelper.com/StuHelper/StuHelper.git /tmp/stuhelper-src
cp /tmp/stuhelper-src/docker-compose.yml /tmp/stuhelper-src/.env.example .
rm -rf /tmp/stuhelper-src
```

### 3. 配置环境变量

```bash
cp .env.example .env
```

编辑 `.env`，填入生产配置：

```bash
# 必须修改
POSTGRES_PASSWORD=<strong-password>
REDIS_PASSWORD=<strong-password>
DATABASE_URL=postgres://stuhelper:<strong-password>@postgres:5432/stuhelper?sslmode=disable
APP_ENV=production
HMAC_SECRET=<openssl rand -hex 32>

# Casdoor SSO
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

### 4. 启动服务

```bash
docker login registry.stuhelper.com
docker compose --profile prod pull
docker compose --profile prod up -d
```

### 5. 验证

```bash
docker compose --profile prod ps       # 所有服务 healthy
curl http://localhost:8080/health       # 后端
curl http://localhost:3000/             # 前端
```

## CI/CD 自动部署

合并到 `main` 分支后，Gitea Actions 自动执行：

1. 构建后端 Docker 镜像 -> push 到 registry
2. 构建前端 Docker 镜像 -> push 到 registry
3. SSH 到服务器执行 `docker compose --profile prod pull && up -d`

### Gitea 仓库 Secrets 配置

| Secret | 说明 |
|--------|------|
| `REGISTRY_USERNAME` | Registry 用户名 |
| `REGISTRY_PASSWORD` | Registry 密码 |
| `DEPLOY_HOST` | 服务器 IP |
| `DEPLOY_PORT` | SSH 端口 |
| `DEPLOY_USER` | SSH 用户 |
| `DEPLOY_SSH_KEY` | SSH 私钥 |
| `DEPLOY_APP_DIR` | 部署目录（如 `/opt/stuhelper`） |

## 手动更新

```bash
cd /opt/stuhelper
docker compose --profile prod pull
docker compose --profile prod up -d --remove-orphans
```

## 回滚

每个构建都打了 git commit short hash 的 tag：

```bash
# 回滚到指定版本
TAG=abc1234 docker compose --profile prod up -d
```

## 数据库安全

生产环境建议配置数据库 SSL：

```bash
DB_SSL_MODE=verify-full
DB_SSL_ROOT_CERT=/etc/ssl/certs/pg-ca.crt
```

详细说明见 `.env.example` 中 `DB_SSL_*` 相关注释。

## 相关文档

- [快速开始](../tutorials/quick-start.md) — 开发环境
- [错误码参考](../reference/error-codes.md)
