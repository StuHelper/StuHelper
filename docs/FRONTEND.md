# 前端开发规范

## 基本约定

- 组件统一用 `<script setup lang="ts">`
- 不手改 `clients/shared/src/types/api.gen.ts`
- 请求默认统一使用 `clients/shared`
  - 合法例外：OIDC callback 浏览器跳转、`sendBeacon` / `keepalive` 可观测性上报、框架/浏览器原生基础设施请求
  - 例外必须在调用点保留注释，说明为何不能走 shared API
- 本地状态够用就不引入 Pinia；跨路由共享的状态才用 store

## 工作区结构

```
clients/
├── shared/
│   ├── src/                # 共享 API / 类型 / 常量源码
│   └── dist/               # 包导出产物（应用只消费这里的导出面）
├── web/src/                # 主站 SPA
│   ├── api/
│   ├── components/
│   ├── composables/
│   ├── constants/
│   ├── design-system/
│   ├── directives/
│   ├── i18n/
│   ├── modules/
│   ├── router/
│   ├── stores/
│   ├── styles/
│   ├── types/
│   └── utils/
├── admin/apps/web-ele/src/ # 独立管理后台
└── uniappx/src/            # 实验性跨端
```

## Web 模块

主站模块以 `clients/web/src/modules/` 为准，当前主要分为：
- `auth`
- `course`
- `review`
- `user`
- `home`
- `common`
- `errors`

主站无嵌入式 `/admin/*` 路由，后台独立部署在 `clients/admin`。

## Admin

代码入口：`clients/admin/apps/web-ele`。部署基路径 `/admin/`。

```bash
make dev-up          # 推荐：连后端一起启动
pnpm dev:admin       # 单独启动
```

## 共享契约链路

```
server/api/openapi.bundled.yaml
  ↓
clients/shared/src/types/api.gen.ts
  ↓
clients/shared/dist/*
  ↓
web / admin / uniappx 封装
```

改接口流程：改 OpenAPI → 重新生成 → 改 shared → 改页面和 store。

## 路由权威来源

- Web 路由权威来源：`clients/web/src/router/index.ts`
- Admin 路由权威来源：`clients/admin/apps/web-ele/src/router/`
- 对外的人工接口索引只维护在：
  - `/Users/zxy/Code/StuHelper/docs/references/api-overview.md`

本页不再重复维护完整页面路径表；需要查看具体页面入口时，直接以路由源码为准。

## 状态管理

跨路由共享状态优先放 Pinia；局部表单、弹窗与一次性页面交互留在组件内部。
当前主站共享状态以 `clients/web/src/stores/` 为准，本页不再重复列出具体 store 清单。

## 开发命令

```bash
cd clients
pnpm install
pnpm dev:web
pnpm dev:admin
pnpm dev:uni
pnpm type-check && pnpm lint
pnpm test:web
pnpm test:e2e:web && pnpm test:e2e:admin
pnpm build:web && pnpm build:admin && pnpm build:uni:h5
```

## 提交前检查

- [ ] 请求走了共享 API
- [ ] 组件和 store 没有 `any`
- [ ] OpenAPI 变更后已重新生成
- [ ] 错误交给统一错误处理
- [ ] `pnpm type-check` / `pnpm lint` / `pnpm test:web` 通过
