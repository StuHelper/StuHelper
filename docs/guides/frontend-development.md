# 前端开发指南

这份文档面向已经把项目跑起来的开发者，重点说明前端 Monorepo 的开发入口、改动顺序和协作边界。

> 环境准备先看 [快速开始](../tutorials/quick-start.md)。

## 四个包分别干什么

| 包                | 用途            | 常改位置                               |
| ----------------- | --------------- | -------------------------------------- |
| `clients/web`     | 主站 Web 应用   | 页面、路由、浏览器适配、用户体验       |
| `clients/admin`   | 独立管理后台    | 后台页面、菜单、RBAC 管理流程          |
| `clients/shared`  | 共享 API 和类型 | OpenAPI 生成类型、共享客户端、能力常量 |
| `clients/uniappx` | 跨端实验入口    | 实验性页面和平台适配                   |

大多数功能改动都会落在 `clients/web` 或 `clients/admin`，再配合 `clients/shared` 一起改。

## 新增页面放哪里

- 主站页面放在 `clients/web/src/modules/<module>/views/`
- 后台页面按各自应用的 `views/` 和 `router/` 结构组织
- 业务组件放在 `clients/web/src/components/business/<domain>/` 或后台对应模块目录
- 共享契约、能力常量和公共 API 包装放在 `clients/shared`

## 路由和访问控制

当前项目的路由习惯是：

- 列表页用复数对象，比如 `/courses`
- 详情页挂在对象下，比如 `/courses/:id`
- 对象动作继续往下挂，比如 `/courses/:id/reviews/post`

登录态由现有路由守卫处理。后台页和受限页通常有三层控制：

1. 路由声明 `requiresAuth`
2. 路由或菜单声明 `requiredCapabilities`
3. 页面内按钮继续判断 `capabilities` 和 `canAccessAdmin`

`clients/web` 和 `clients/admin` 都按这套规则工作。

## 新增接口时的顺序

前端接新接口，顺序固定：

1. 先改 `server/api/openapi.yaml` 和拆分出的规范文件
2. 在 `server/` 里运行 `make generate`
3. 确认 `server/internal/api/gen/` 和 `clients/shared/src/types/api.gen.ts` 已更新
4. 在 `clients/shared/src/api/` 补领域 API 包装
5. 再回到 `web` 或 `admin` 页面接入

共享链路是：

```text
OpenAPI -> clients/shared -> web/admin
```

不要绕开这条链，直接在页面里手写裸请求。

## 认证相关注意点

- Access Token 和 Refresh Token 由后端写入 Cookie
- 浏览器请求使用 `credentials: 'include'`
- 写操作会自动带上 `X-CSRF-Token`
- 碰到 401 时，客户端会先尝试 `/api/v1/auth/refresh`
- 登录后是否能进后台，最终以 `/api/v1/auth/me` 返回的 `capabilities`、`canAccessAdmin`、`isPlatformAdmin` 为准

其中 `isPlatformAdmin` 只表示 Casdoor 平台管理员，不等于航小伴业务管理员。

## 提交前校验

前端改动提交前，至少跑：

```bash
cd clients
pnpm type-check
pnpm lint
```

如果动了主站核心流程，再补：

```bash
cd clients
pnpm test:web
pnpm test:e2e:web
```

## 常看文件

| 文件                                  | 用途                              |
| ------------------------------------- | --------------------------------- |
| `clients/web/src/router/index.ts`     | 主站路由和守卫                    |
| `clients/admin/src/router/index.ts`   | 后台路由和权限控制                |
| `clients/web/src/api/client.ts`       | 浏览器 Cookie、CSRF、refresh 逻辑 |
| `clients/admin/src/api/`              | 后台 API 包装                     |
| `clients/shared/src/api/`             | 共享 API 包装                     |
| `clients/shared/src/types/api.gen.ts` | OpenAPI 生成类型                  |

## 相关文档

- [前端架构](../architecture/frontend-architecture.md)
- [OpenAPI 开发指南](openapi-development-guide.md)
- [API 概览](../reference/api-overview.md)
