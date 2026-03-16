# 生产环境部署指南

StuHelper 生产环境使用 Docker Compose 部署，镜像通过 GitLab CI/CD 构建并推送到 `registry.stuhelper.com`。

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

当前对外访问使用根域名 `https://stuhelper.com`，主站通过子路径区分模块。Casdoor 运行在 `https://sso.stuhelper.com`。

当前根目录 `docker-compose.yml` 的 `prod` profile 只部署 `app`、`frontend`、`postgres` 和 `redis`。`clients/admin` 已经存在，但还没有接进根 compose 的生产 profile。

## 首次部署

### 1. 准备部署目录

```bash
mkdir -p /opt/stuhelper
cd /opt/stuhelper
```

### 2. 获取配置文件

```bash
curl -o docker-compose.yml https://git.stuhelper.com/stuhelper/StuHelper/-/raw/main/docker-compose.yml
curl -o .env.example https://git.stuhelper.com/stuhelper/StuHelper/-/raw/main/.env.example
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
DOC_AES_ACTIVE_KEY_ID=1
DOC_AES_KEYS=1:<openssl-rand-hex-32>

CASDOOR_ENDPOINT=https://sso.stuhelper.com
CASDOOR_CLIENT_ID=<from-casdoor-admin>
CASDOOR_CLIENT_SECRET=<from-casdoor-admin>
CASDOOR_REDIRECT_URI=https://stuhelper.com/auth/callback

TOKEN_COOKIE_SECURE=true
CORS_ORIGINS=https://stuhelper.com
```

这两个密钥不要漏掉：

- `HMAC_SECRET`：用于生成稳定的 `person_uid`。
- `DOC_AES_KEYS`：用于加密实名认证证件号。

`DOC_AES_KEYS` 的格式固定为 `keyID:hex`，例如：

```bash
DOC_AES_ACTIVE_KEY_ID=1
DOC_AES_KEYS=1:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
```

`DOC_AES_ACTIVE_KEY_ID` 和 `DOC_AES_KEYS` 通过启动校验后，后端才会进入运行状态。

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
- [前端架构](../architecture/frontend-architecture.md)
- [错误码](../reference/error-codes.md)
