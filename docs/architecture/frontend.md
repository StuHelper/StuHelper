# 前端架构

本文档描述当前已经落地的前端架构，而不是历史规划态。时间点以 2026 年 3 月的代码树为准。

## 先记住这几个结论

- Web 主站使用 `stuhelper.com` 根域名加子路径，不再为评课单独使用子域名。
- SSO 仍由 `https://sso.stuhelper.com` 上的 Casdoor 提供。
- API 契约的权威来源是 `server/api/openapi.yaml`，前后端都围绕 OpenAPI 3 协作。
- 前端 Monorepo 由 `clients/web`、`clients/uniappx`、`clients/shared` 三个包组成，其中 `clients/uniappx` 当前仍处于实验性脚手架阶段。
- Web 端 API 调用已经从 Axios 迁移到 `openapi-fetch`，并使用 Cookie 会话、CSRF 头、Refresh Token 自动续期。

## 当前技术栈

| 领域         | 选型                           | 说明                                             |
| ------------ | ------------------------------ | ------------------------------------------------ |
| Web 框架     | Vue 3.5 + Vue Router 4         | 主站 SPA                                         |
| 构建工具     | Vite 6                         | Web 与 uni-app x 都使用 Vite 体系                |
| 语言         | TypeScript 5.7+                | 严格类型检查                                     |
| 状态管理     | Pinia 2                        | 跨页面共享状态                                   |
| 样式与 UI    | Tailwind CSS v4 + Element Plus | 页面样式与管理后台表单能力                       |
| API 客户端   | `openapi-fetch`                | 基于 OpenAPI 3 生成的类型安全客户端              |
| 组件开发     | Storybook 8                    | 组件文档与独立调试                               |
| 单元测试     | Vitest 4                       | 工具函数、状态、组合式函数测试                   |
| E2E 测试     | Playwright 1.58                | 路由和页面冒烟测试                               |
| 管理后台增强 | Vben effects 组件              | 当前已接入计数组件等能力，后台骨架仍由本项目维护 |

## Monorepo 结构

| 包                | 作用             | 关键内容                                                    |
| ----------------- | ---------------- | ----------------------------------------------------------- |
| `clients/web`     | Web 主站         | 路由、页面、浏览器认证适配、Storybook、Vitest、Playwright   |
| `clients/uniappx` | uni-app x 客户端 | H5 / 小程序 / App 的实验性脚手架入口，后续逐步收口到共享 API 与类型 |
| `clients/shared`  | 跨端共享层       | `openapi-fetch` 基础客户端、OpenAPI 生成类型、通用 API 包装 |

当前工作区声明见 `clients/pnpm-workspace.yaml`：

```yaml
packages:
  - "web"
  - "uniappx"
  - "shared"
```

## 域名与路由设计

### Web 主路由

| 路径                        | 说明                 |
| --------------------------- | -------------------- |
| `/`                         | 主站首页             |
| `/course`                   | 教学门户入口         |
| `/review`                   | 评课社区首页         |
| `/courses`                  | 课程列表             |
| `/courses/:id`              | 课程概览             |
| `/courses/:id/reviews`      | 课程测评列表         |
| `/courses/:id/reviews/post` | 发布测评页，需要登录 |
| `/teachers/:id`             | 教师主页             |
| `/user/reviews`             | 我的测评             |
| `/user/votes`               | 我的投票             |
| `/user/favorites`           | 我的收藏             |
| `/notifications`            | 通知中心             |
| `/admin/*`                  | 管理后台             |

### 兼容性重定向

为了兼容旧链接，路由层仍保留以下重定向：

- `/review/courses` → `/courses`
- `/review/courses/:id` → `/courses/:id/reviews`
- `/review/teachers/:id` → `/teachers/:id`
- `/courses/:id/review` → `/courses/:id/reviews`

### 为什么发布测评页设计成 `/courses/:id/reviews/post`

这是当前推荐的规范路径，原因很直接：

1. 路由先表达对象，再表达动作，URL 一眼能看出“给哪门课发测评”。
2. 页面可以天然依赖课程上下文，不需要额外 query 参数补齐课程 ID。
3. 深链接、权限守卫、回跳逻辑都更稳定，后续也容易扩展成草稿恢复页。

## OpenAPI 3 与类型安全

StuHelper 采用 Spec-First 流程。权威接口定义在：

```text
server/api/openapi.yaml
```

生成流程如下：

1. 修改 `server/api/` 下的 OpenAPI 3 规范。
2. 运行：

```bash
cd server
make generate
```

3. 后端会更新 `server/internal/api/gen/`。
4. 前端会更新 `clients/shared/src/types/api.gen.ts`。
5. `clients/shared/src/api/client.ts` 用 `openapi-fetch` 创建基础客户端。
6. `clients/web/src/api/client.ts` 在浏览器端补上 Cookie、CSRF、刷新会话逻辑。
7. `clients/web/src/api/index.ts` 提供 Web 端兼容包装，逐步承接旧调用方式。

这意味着前端不应该再手写一套与后端脱节的接口类型，也不应该直接用裸 `fetch` 或旧 Axios 实例请求业务接口。

## 认证与会话模型

### 当前实现

1. 前端调用 `/api/v1/auth/login` 或 `/api/v1/auth/signup` 获取 Casdoor 跳转地址。
2. 浏览器跳转到 `https://sso.stuhelper.com`。
3. Casdoor 回跳到 Web 前端的 `/auth/callback`。
4. 前端回调页再调用 `/api/v1/auth/callback`，由后端完成授权码换 token。
5. 后端写入 `HttpOnly` 的 access token 和 refresh token Cookie。
6. 浏览器请求 API 时统一使用 `credentials: 'include'`。
7. 变更型请求会从 `csrf_token` Cookie 读取值并写入 `X-CSRF-Token` 头。
8. Web 端发现本地会话临近过期或收到 `401` 时，会调用 `/api/v1/auth/refresh` 尝试续期。
9. 刷新失败则清空本地用户态，并要求重新登录。

### 为什么这样做

- Cookie 会话避免了把 access token 暴露给业务代码。
- Refresh Token 放在 `HttpOnly` Cookie 中，由后端和 Casdoor 通信完成刷新。
- 前端只保存最小会话信息和过期时间，用来做路由预检与体验优化。

### Casdoor 官方行为参考

- Refresh Token 流程参考 Casdoor 官方文档中的 [Refresh token](https://casdoor.org/docs/basic/server-side-auth/token/#refresh-token)。
- 清理 SSO 登录态时，参考 Casdoor 官方文档中的 [Logout](https://casdoor.org/docs/basic/server-side-auth/token/#logout)。

当前项目里，后端登出接口会返回 `ssoLogoutURL`，前端通过顶级导航或弹窗访问该地址，确保 Casdoor 的浏览器会话也被清理。

## 测试与质量控制

| 能力            | 命令                                                    | 用途               |
| --------------- | ------------------------------------------------------- | ------------------ |
| TypeScript 检查 | `pnpm --dir clients --filter @stuhelper/web type-check` | 发现类型漂移       |
| 单元测试        | `pnpm --dir clients --filter @stuhelper/web test`       | 工具函数与状态测试 |
| Storybook       | `pnpm --dir clients --filter @stuhelper/web storybook`  | 组件独立开发       |
| E2E 测试        | `pnpm --dir clients --filter @stuhelper/web test:e2e`   | 页面冒烟与路由验证 |

## 开发入口

常用命令集中在 `clients/package.json`：

```bash
cd clients
pnpm install
pnpm dev:web
pnpm storybook:web
pnpm test:web
pnpm test:e2e:web
pnpm type-check
```

如果你要继续前端开发，下一步直接看 [前端开发指南](../guides/frontend-development.md)。
