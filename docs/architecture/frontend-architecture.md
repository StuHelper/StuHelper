# 前端架构

这份文档只描述当前代码树已经落地的前端结构，不保留旧的 uni-app 单入口方案。

## 当前包结构

| 包 | 作用 | 当前状态 |
| --- | --- | --- |
| `clients/web` | 主站 Web SPA，承载课程、评课、教师、用户中心和一套嵌入式后台路由 | 主入口 |
| `clients/admin` | 独立后台控制台，聚焦评课管理、用户系统和 RBAC 管理 | 已实现 |
| `clients/shared` | OpenAPI 生成类型、`openapi-fetch` 客户端、共享 capability 常量 | 契约中心 |
| `clients/uniappx` | 跨端实验入口 | 预研态 |

`clients/pnpm-workspace.yaml` 当前工作区包含这四个包。

## 路由与部署形态

主站路由集中在 `clients/web/src/router/index.ts`，核心路径包括：

| 路径 | 说明 |
| --- | --- |
| `/` | 主站首页 |
| `/course` | 教学门户入口 |
| `/review` | 评课社区入口 |
| `/courses/:id` | 课程概览 |
| `/courses/:id/reviews` | 课程测评列表 |
| `/courses/:id/reviews/post` | 发布测评 |
| `/teachers/:id` | 教师主页 |
| `/user/*` | 用户中心 |
| `/admin/*` | 主站内嵌后台页面 |

独立后台在 `clients/admin`，路由基地址是 `/admin`。它和 `clients/web` 共用同一套 `/api/v1/auth/me` 用户态与 capability 契约，但页面骨架、菜单和视图独立维护。

当前仓库里保留了两套后台入口：

- `clients/web` 里的后台路由，用于主站一体化集成
- `clients/admin` 独立控制台，用于后台能力集中开发

## 认证与权限边界

前端统一使用 Cookie 会话，不在业务代码里保存 access token。

当前登录流是：

1. 前端请求 `/api/v1/auth/login` 或 `/api/v1/auth/signup`
2. 浏览器跳转到 `https://sso.stuhelper.com`
3. Casdoor 回跳前端 `/auth/callback`
4. 前端再调用 `/api/v1/auth/callback`
5. 后端写入 `HttpOnly` 的 access token、refresh token 和 `csrf_token` Cookie

后台门禁不再使用 `isAdmin`。当前前端只认三件事：

- `/api/v1/auth/me` 返回的 `capabilities`
- `/api/v1/auth/me` 返回的 `canAccessAdmin`
- 路由级 `requiredCapabilities`

`clients/web` 和 `clients/admin` 都用同一套 capability 常量做菜单过滤、页面守卫和按钮控制。

## 共享契约

前后端通过 `server/api/openapi.yaml` 协作，生成产物是：

- `server/internal/api/gen/`
- `clients/shared/src/types/api.gen.ts`

共享层的职责边界如下：

| 位置 | 职责 |
| --- | --- |
| `clients/shared/src/types/api.gen.ts` | OpenAPI 生成的传输层类型 |
| `clients/shared/src/api/` | 基础 API client 和共享封装 |
| `clients/shared/src/constants/` | capability、业务常量、跨端共享枚举 |
| `clients/web/src/api/` | 浏览器侧 Cookie、CSRF、刷新会话适配 |
| `clients/admin/src/api/` | 后台侧 API 封装 |

如果前端接口形状变了，应该先改 OpenAPI，再重新生成，不要在 `web` 或 `admin` 里手写第二份契约。
