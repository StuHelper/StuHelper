# 前端开发指南

这份指南给已经把项目跑起来的开发者使用。它回答的是“我现在要在这个前端里继续开发，应该怎么接入现有体系”。

## 先理解这 3 个包

| 包                | 作用             | 你通常会改什么                  |
| ----------------- | ---------------- | ------------------------------- |
| `clients/web`     | Web 主站         | 页面、路由、浏览器适配、测试    |
| `clients/uniappx` | uni-app x 客户端 | 跨端页面与平台适配              |
| `clients/shared`  | 共享 API 与类型  | OpenAPI 生成类型、跨端 API 包装 |

多数 Web 功能改动会同时碰到 `clients/web` 和 `clients/shared`。

## 新增一个页面时怎么做

### 1. 先放对目录

- 页面放在 `clients/web/src/modules/<module>/views/`
- 业务组件放在 `clients/web/src/components/business/<domain>/`
- 通用组件放在 `clients/web/src/components/common/`

### 2. 遵守当前路由约定

当前项目使用“对象优先”的命名方式：

- 列表页用名词复数，如 `/courses`
- 详情页挂在对象下，如 `/courses/:id`
- 某对象上的动作继续下钻，如 `/courses/:id/reviews/post`

不要再新增 `/review/post?id=...` 这类脱离上下文的路径，除非真的是全局动作。

### 3. 在路由元信息里声明权限

- 登录态：`meta.requiresAuth`
- 仅游客访问：`meta.guest`

Web 端的路由守卫会在 access token 即将过期时尝试刷新会话；刷新失败才会跳登录。

后台或受限功能不要再用 `meta.requiresAdmin` 这种“平台管理员即应用管理员”的旧做法。正确做法是：

- 路由先判断 `requiresAuth`
- 页面菜单、按钮、路由跳转再根据应用能力 `capabilities / effective permissions` 控制

## 新增一个 API 时怎么做

StuHelper 不建议先写前端请求，再去猜后端返回结构。正确顺序是：

1. 先改 `server/api/openapi.yaml` 及其拆分文件。
2. 运行：

```bash
cd server
make generate
```

3. 确认这两个生成结果已同步：

- `server/internal/api/gen/`
- `clients/shared/src/types/api.gen.ts`

4. 在 `clients/shared/src/api/` 中补充领域 API 包装。
5. 如需兼容现有 Web 调用方式，再在 `clients/web/src/api/index.ts` 增加包装函数。

Web 端基础客户端已经统一在：

- `clients/shared/src/api/client.ts`
- `clients/web/src/api/client.ts`

不要重新引入旧 Axios 实例，也不要在页面里直接写裸 `fetch`。

## 认证相关开发注意点

### 当前会话模型

- Access Token 和 Refresh Token 都由后端写入 Cookie。
- 浏览器请求 API 时统一使用 `credentials: 'include'`。
- 变更型请求会自动带 `X-CSRF-Token`。
- 401 场景会先走 `/api/v1/auth/refresh`，失败后再清本地会话。

### 登录回跳

如果你的页面必须登录后再访问，直接在路由上加 `requiresAuth`。路由守卫会把当前地址带到登录页，登录完成后自动回跳。

如果你的新功能还需要草稿恢复、临时跳转之类的逻辑，优先复用现有的 `post_login_redirect` 和 `draft_redirect` 约定，不要再发明一套新的 session key。

## 组件、单测和 E2E

### Storybook

用来做独立组件开发和视觉回归前的人工检查：

```bash
cd clients
pnpm storybook:web
```

### Vitest

适合这些内容：

- 工具函数
- store
- composable
- 轻量组件逻辑

运行：

```bash
cd clients
pnpm test:web
pnpm test:coverage:web
```

### Playwright

适合这些内容：

- 路由跳转
- 页面基础渲染
- 登录前后守卫
- 关键用户路径冒烟

运行：

```bash
cd clients
pnpm test:e2e:web
```

## 管理后台约定

当前后台仍以本项目自己的页面骨架为主，已经接入 Vben effects 组件能力做增强。含义是：

- 可以继续在现有 `clients/web/src/modules/admin/views/` 上迭代。
- 如果只需要数字动画、权限效果等能力，优先复用现有 Vben effects 接入。
- 如果未来要整站迁入完整的 Vue Vben Admin scaffold，应作为单独重构处理，而不是在当前页面里渐进式混搭。

## 提交前检查

前端改动合并前，至少跑一遍：

```bash
cd clients
pnpm type-check
pnpm test:web
```

涉及路由、首屏、鉴权或关键页面时，再补：

```bash
cd clients
pnpm test:e2e:web
```

涉及组件库或通用组件时，再补：

```bash
cd clients
pnpm build:storybook:web
```

## 常见入口文件

| 文件                                  | 用途                       |
| ------------------------------------- | -------------------------- |
| `clients/web/src/router/index.ts`     | Web 路由与守卫             |
| `clients/web/src/api/client.ts`       | 浏览器认证、CSRF、刷新逻辑 |
| `clients/web/src/api/index.ts`        | Web API 包装与兼容层       |
| `clients/shared/src/api/client.ts`    | OpenAPI 基础客户端         |
| `clients/shared/src/types/api.gen.ts` | OpenAPI 生成类型           |

如果你不确定某个改动该从哪里下手，先从路由文件和 API 包装文件开始查，通常最快。
