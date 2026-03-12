# 生产环境部署指南

StuHelper 生产环境使用 Docker Compose 部署，镜像通过 Gitea Actions 构建并推送到 `registry.stuhelper.com`。

## 部署形态

```text
registry.stuhelper.com
  ├── stuhelper/backend:latest
  └── stuhelper/frontend:latest

服务器（docker compose --profile prod）
  ├── postgres
  ├── redis
  ├── app
  └── frontend
```

对外访问建议统一收敛到根域名 `https://stuhelper.com`，前端再通过子路径区分模块。Casdoor 仍作为独立 SSO 服务运行在 `https://sso.stuhelper.com`。

## 首次部署

### 1. 准备部署目录

```bash
mkdir -p /opt/stuhelper
cd /opt/stuhelper
```

### 2. 获取配置文件

```bash
curl -o docker-compose.yml https://gitea.stuhelper.com/StuHelper/StuHelper/raw/branch/main/docker-compose.yml
curl -o .env.example https://gitea.stuhelper.com/StuHelper/StuHelper/raw/branch/main/.env.example
cp .env.example .env
```

### 3. 配置环境变量

至少确认这些值：

```bash
POSTGRES_PASSWORD=<strong-password>
REDIS_PASSWORD=<strong-password>
DATABASE_URL=postgres://stuhelper:<strong-password>@postgres:5432/stuhelper?sslmode=disable
APP_ENV=production
HMAC_SECRET=<openssl-rand-hex-32>

CASDOOR_ENDPOINT=https://sso.stuhelper.com
CASDOOR_CLIENT_ID=<from-casdoor-admin>
CASDOOR_CLIENT_SECRET=<from-casdoor-admin>
CASDOOR_REDIRECT_URI=https://stuhelper.com/auth/callback

TOKEN_COOKIE_SECURE=true
CORS_ORIGINS=https://stuhelper.com
```

### 4. 启动服务

```bash
docker login registry.stuhelper.com
docker compose --profile prod pull
docker compose --profile prod up -d
```

### 5. 验证

```bash
docker compose --profile prod ps
curl http://localhost:8080/health
curl http://localhost:3000/
```

## 手动更新

```bash
cd /opt/stuhelper
docker compose --profile prod pull
docker compose --profile prod up -d --remove-orphans
```

## 回滚

```bash
TAG=<git-short-sha> docker compose --profile prod up -d
```

## 相关文档

- [快速开始](../tutorials/quick-start.md)
- [前端架构](../architecture/frontend.md)
- [错误码](../reference/error-codes.md)
