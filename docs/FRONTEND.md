# 前端开发规范

## 基本约定

- 组件统一用 `<script setup lang="ts">`
- 不手改 `clients/shared/src/types/api.gen.ts`
- 请求统一使用 `clients/shared`，页面不直接 `fetch`
- 本地状态够用就不引入 Pinia；跨路由共享的状态才用 store

## 工作区结构

```
clients/
├── shared/src/             # 共享 API / 类型 / 常量
├── web/src/                # 主站 SPA
│   ├── api/
│   ├── components/
│   ├── composables/
│   ├── modules/
│   ├── stores/
│   └── router/
├── admin/apps/web-ele/src/ # 独立管理后台
└── uniappx/                # 实验性跨端
```

## Web 模块

`clients/web/src/modules/`：

- `auth` — 登录、OIDC 回调
- `course` — 课程列表、教师、门户
- `review` — 评课、搜索、发帖
- `user` — 用户中心、认证、通知
- `home` / `errors`

主站无嵌入式 `/admin/*` 路由，后台独立部署在 `clients/admin`。

## Admin

代码入口：`clients/admin/apps/web-ele`。部署基路径 `/admin/`。

```bash
make dev-up          # 推荐：连后端一起启动
pnpm dev:admin       # 单独启动
```

## 共享契约链路

```
server/api/openapi.yaml
  ↓
clients/shared/src/types/api.gen.ts
  ↓
clients/shared/src/api/*
  ↓
web / admin 封装
```

改接口流程：改 OpenAPI → 重新生成 → 改 shared → 改页面和 store。

## 主站路由

| 路径 | 页面 |
|------|------|
| `/` | 首页 |
| `/course` | 教学门户 |
| `/courses` / `/courses/:id` | 课程列表 / 详情 |
| `/courses/:id/reviews/post` | 发评课 |
| `/teachers` / `/teachers/:id` | 教师 |
| `/search` | 搜索 |
| `/login` / `/auth/callback` | 登录 |
| `/user/*` | 用户中心 |
| `/notifications` | 通知 |

## 状态管理

Pinia store 当前用于：认证会话、通知、课程评课聚合、草稿、主题/语言。

组件级表单、弹窗、局部交互留在页面内部。

## 开发命令

```bash
cd clients
pnpm install
pnpm type-check && pnpm lint
pnpm test:web
pnpm test:e2e:web && pnpm test:e2e:admin
pnpm build:web && pnpm build:admin
```

## 提交前检查

- [ ] 请求走了共享 API
- [ ] 组件和 store 没有 `any`
- [ ] OpenAPI 变更后已重新生成
- [ ] 错误交给统一错误处理
- [ ] `pnpm type-check` / `pnpm lint` / `pnpm test:web` 通过
