# 前端开发指南

这份指南面向已经把项目跑起来的开发者。它回答的是当前前端 Monorepo 里该怎么继续开发，不再描述旧的单入口方案。

## 先理解这 4 个包

| 包 | 作用 | 你通常会改什么 |
| --- | --- | --- |
| `clients/web` | Web 主站 | 页面、路由、浏览器适配、用户态体验 |
| `clients/admin` | 独立后台 | 后台页面、菜单、RBAC 管理流 |
| `clients/shared` | 共享 API 与类型 | OpenAPI 生成类型、共享 client、capability 常量 |
| `clients/uniappx` | 跨端实验入口 | 预研页面和平台适配 |

多数线上功能改动会同时碰到 `clients/web` 或 `clients/admin`，再加上 `clients/shared`。

## 新增页面时怎么放

- 主站页面放在 `clients/web/src/modules/<module>/views/`
- 后台页面按各自应用的 `views/` 和 `router/` 结构放
- 业务组件放在 `clients/web/src/components/business/<domain>/` 或 admin 对应模块内
- 共享契约和 capability 放在 `clients/shared`

## 路由和权限怎么接

当前项目的路由约定是对象优先：

- 列表页用名词复数，比如 `/courses`
- 详情页挂在对象下，比如 `/courses/:id`
- 某对象上的动作继续下钻，比如 `/courses/:id/reviews/post`

登录态交给现有路由守卫处理。后台或受限页面不要再用 `requiresAdmin` 这种旧语义，统一按下面三层接：

- 路由声明 `requiresAuth`
- 路由或菜单声明 `requiredCapabilities`
- 页面按钮再按 `capabilities` 和 `canAccessAdmin` 收口

`clients/web` 和 `clients/admin` 都走这套规则。

## 新增接口时怎么接

正确顺序不是前端先猜返回结构，而是：

1. 先改 `server/api/openapi.yaml` 及其拆分文件
2. 在 `server` 下运行 `make generate`
3. 确认 `server/internal/api/gen/` 和 `clients/shared/src/types/api.gen.ts` 已同步
4. 在 `clients/shared/src/api/` 补领域 API 包装
5. 再在 `web` 或 `admin` 侧接具体页面逻辑

不要重新引入裸 `fetch`、旧 Axios 实例或手写第二份 DTO。

## 认证相关注意点

- Access Token 和 Refresh Token 由后端写入 Cookie
- 浏览器请求统一 `credentials: 'include'`
- 变更型请求自动带 `X-CSRF-Token`
- 401 场景先走 `/api/v1/auth/refresh`，失败后再清本地会话
- 登录完成后的用户态与后台可达性以 `/api/v1/auth/me` 返回的 `capabilities`、`canAccessAdmin`、`isPlatformAdmin` 为准

其中 `isPlatformAdmin` 只是生态平台身份，不是航小伴业务后台门禁。

## 提交前检查

前端改动至少跑一遍：

```bash
cd clients
pnpm type-check
pnpm lint
```

涉及主站关键流时，再补：

```bash
cd clients
pnpm test:web
pnpm test:e2e:web
```

## 常见入口文件

| 文件 | 用途 |
| --- | --- |
| `clients/web/src/router/index.ts` | 主站路由与守卫 |
| `clients/admin/src/router/index.ts` | 独立后台路由与门禁 |
| `clients/web/src/api/client.ts` | 浏览器 Cookie、CSRF、刷新逻辑 |
| `clients/admin/src/api/` | 后台 API 封装 |
| `clients/shared/src/api/` | 共享基础客户端 |
| `clients/shared/src/types/api.gen.ts` | OpenAPI 生成类型 |
