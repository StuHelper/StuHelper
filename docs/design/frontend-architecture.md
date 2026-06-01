---
type: design
audience: frontend-dev
status: current
authoritative-source: this file
last-verified: 2026-05-30
---

# 前端 Monorepo 架构

> 状态：现行

## 包结构

| 包 | 用途 |
|----|------|
| `clients/web` | 主站 SPA |
| `clients/admin` | 独立管理后台（Vben Admin + Element Plus） |
| `clients/shared` | OpenAPI 生成类型、共享 API 客户端、常量 |
| `clients/uniappx` | 实验性跨端入口 |

## 路由

- Web 路由权威来源：`clients/web/src/router/index.ts`
- Admin 路由权威来源：`clients/admin/apps/web-ele/src/router/`
- 人工 API / 页面索引不在本设计文档重复维护，统一看：
  - [../guides/frontend-development.md](../guides/frontend-development.md)
  - [../reference/api-overview.md](../reference/api-overview.md)

## 共享契约链路

```
server/api/openapi.bundled.yaml
  ├── server/internal/api/gen/              # Go 类型
  └── clients/shared/src/types/api.gen.ts   # TS 类型
        ↓
      clients/shared/dist/*                 # 包导出面（应用只消费导出）
        ↓
      clients/web/src/api/index.ts          # 主站 api 对象
      clients/admin/apps/web-ele/src/api/*  # 管理后台 API
      clients/uniappx/src/api/index.ts      # UniApp X API
```

先改 OpenAPI → 生成类型 → 改实现。

## API 调用链

主站请求链路：

1. `clients/web/src/api/client.ts` — `authenticatedFetch`
2. `@stuhelper/shared/api` — 共享 API 封装
3. `openapi-fetch` — 底层 HTTP

`authenticatedFetch` 负责：Cookie 携带、CSRF header 注入、401 自动 refresh、统一 `ApiError`。

## 登录流程

```
业务页面 → /login?redirect=<当前业务目标>
  → LoginPage 调用 /api/v1/auth/login 启动 sso.stuhelper.com Casdoor 登录
  → OIDC 回调 stuhelper.com/api/v1/auth/callback
  → 后端写入 Cookie
  → AuthCallbackPage 或后端 302 回业务目标页
  → auth store 拉 /api/v1/auth/me
  → 路由守卫按登录态和 capabilities 放行
```

前端不再把受保护入口跨域改写到 `id.stuhelper.com`。账号中心、开发者应用、授权应用、学生认证和 QQ 绑定都在 `stuhelper.com` 主站路由内承载；入群验证只从 `join.stuhelper.com/verify/<token>?qq=<qq>` 进入。

## 状态管理

Pinia store：`auth` / `notification` / `draft` / `courseReview` / `user` / `verification` / `theme` / `locale`

原则：
- 服务端状态优先 API 即取即用
- 跨页面状态才放 store
- 页面临时状态用 `ref` / `reactive`

## 开发与校验

```bash
cd clients
pnpm install
# 以下为持久化 dev server，需在各自终端中运行
pnpm dev:web       # 主站
pnpm dev:admin     # 管理后台（另开终端）
pnpm dev:uni       # UniApp X（另开终端）
pnpm type-check:all && pnpm lint:all && pnpm test:web && pnpm test:e2e
pnpm build:web && pnpm build:admin && pnpm build:uni:h5
```

`pnpm test:e2e` 覆盖 Web、Admin 与 UniAppX H5 三个前端入口；Koishi Console 使用独立工作区的 `make e2e-koishi`。

或在仓库根目录执行 `make dev-up` 连基础设施一起启动。
