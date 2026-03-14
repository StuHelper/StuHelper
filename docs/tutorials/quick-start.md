# 快速开始

这份文档帮你把 StuHelper 跑起来，并知道第一次应该从哪里开始改代码。

当前仓库是一个前后端混合 Monorepo：

- `server/` 是 Go 后端。
- `clients/web` 是 Vue Web 主站。
- `clients/uniappx` 是 uni-app x 客户端。
- `clients/shared` 是跨端共享 API 与类型。

推荐先用“混合模式”开发：数据库和 Redis 用 Docker，后端和前端在宿主机启动。这样排查问题最直接。

## 环境要求

| 工具             | 版本建议 | 安装方式                              |
| ---------------- | -------- | ------------------------------------- |
| Docker & Compose | 24+      | [docker.com](https://www.docker.com/) |
| Go               | 1.24+    | `brew install go`                     |
| Node.js          | 24+      | `brew install node`                   |
| pnpm             | 10+      | `npm install -g pnpm`                 |

## 1. 克隆仓库

```bash
git clone https://gitea.stuhelper.com/StuHelper/StuHelper.git
cd StuHelper
```

## 2. 配置后端环境变量

```bash
cp .env.example .env
```

至少确认下面这些值：

```bash
POSTGRES_PASSWORD=dev123
DATABASE_URL=postgres://stuhelper:dev123@localhost:5432/stuhelper?sslmode=disable
REDIS_PASSWORD=dev123
HMAC_SECRET=dev_hmac_secret_change_in_production_32ch
DOC_AES_ACTIVE_KEY_ID=1
DOC_AES_KEYS=1:<openssl-rand-hex-32>

CASDOOR_ENDPOINT=https://sso.stuhelper.com
CASDOOR_CLIENT_ID=
CASDOOR_CLIENT_SECRET=
CASDOOR_REDIRECT_URI=http://localhost:3000/auth/callback
```

说明：

- 本项目当前依赖已部署的 Casdoor `https://sso.stuhelper.com`。
- 后端现在会强制校验 PII 加密密钥；如果 `DOC_AES_ACTIVE_KEY_ID` 或 `DOC_AES_KEYS` 没配好，服务不会启动。
- `DOC_AES_KEYS` 的格式是 `keyID:hex`，例如 `1:0123...abcd`。本地开发也不能省略。
- 生成一把可用的证件号加密密钥：

```bash
openssl rand -hex 32
```

- 如果本地只做匿名浏览或接口联调，可以先不配置完整 SSO。
- 如果要走完整登录流程，前端回调地址必须与 Casdoor 应用配置一致。

## 3. 配置前端环境变量

Web 端默认读取 `clients/web/.env`：

```bash
cp clients/web/.env.example clients/web/.env
```

本地常用配置如下：

```bash
VITE_API_URL=http://localhost:8080
VITE_SSO_URL=https://sso.stuhelper.com
VITE_CASDOOR_CLIENT_ID=<same-as-backend-app>
```

如果你只是本地访问同源反向代理，也可以把 `VITE_API_URL` 改为 `/api`。

## 4A. 混合模式开发（推荐）

### 启动基础设施

```bash
docker compose up -d
docker compose ps
```

`postgres` 和 `redis` 都应为 `healthy`。

### 启动后端

```bash
cd server
make run
```

验证：

- 健康检查：`http://localhost:8080/health`
- Swagger UI：`http://localhost:8080/docs/`

### 启动前端

新开一个终端：

```bash
cd clients
pnpm install
pnpm dev:web
```

默认打开：

- Web 主站：`http://localhost:3000`

## 4B. 全 Docker 模式开发

如果你希望前后端都在容器里跑：

```bash
docker compose --profile dev-full up
```

默认地址：

- 后端：`http://localhost:8080`
- 前端：`http://localhost:3000`

查看日志：

```bash
docker compose --profile dev-full logs -f app-dev
docker compose --profile dev-full logs -f frontend-dev
```

## 第一次启动后建议做的事

先确认这几个命令都能跑通：

```bash
cd server
make generate

cd ../clients
pnpm type-check
pnpm test:web
```

如果你要做组件开发，再跑：

```bash
cd clients
pnpm storybook:web
```

如果你要做页面或路由改动，再跑：

```bash
cd clients
pnpm test:e2e:web
```

## 日常开发命令

### 后端

```bash
cd server
make run
make test
make lint
make fmt
make build
make generate
make lint-spec
```

### 前端

```bash
cd clients
pnpm dev:web
pnpm dev:uni
pnpm type-check
pnpm test:web
pnpm test:e2e:web
pnpm storybook:web
pnpm build:web
```

## 常见问题

### 前端提示接口跨域或连不上

- 检查 `clients/web/.env` 里的 `VITE_API_URL`。
- 宿主机开发通常使用 `http://localhost:8080`。
- 走反向代理时可改成 `/api`。

### 登录后没有回跳到原页面

- 检查 Casdoor 应用里的回调地址是否包含 `http://localhost:3000/auth/callback`。
- 检查浏览器是否拦截了第三方 Cookie 或弹窗。

### OpenAPI 类型不一致

先重新生成：

```bash
cd server
make generate
```

这会同时更新后端生成代码和 `clients/shared/src/types/api.gen.ts`。

## 下一步看什么

- 前端开发看 [前端开发指南](../guides/frontend-development.md)
- 后端开发看 [后端开发指南](../guides/backend-quickstart.md)
- API 协作看 [OpenAPI 3 开发指南](../guides/openapi-development-guide.md)
