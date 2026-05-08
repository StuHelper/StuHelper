# 前端工作区

| 目录 | 用途 |
|------|------|
| `web/` | 主站 SPA |
| `admin/` | 独立管理后台（Vben Admin + Element Plus） |
| `shared/` | OpenAPI 生成类型、共享 API 封装 |
| `uniappx/` | 实验性跨端入口 |

## 环境

- Node.js 20.19+ / 22.18+ / 24+
- pnpm 10+

## 命令

```bash
cd clients
pnpm install

pnpm dev:web && pnpm dev:admin && pnpm dev:uni
pnpm type-check:all && pnpm lint:all
pnpm test:web && pnpm test:e2e
pnpm build:web && pnpm build:admin && pnpm build:uni:h5
```

连后端一起启动：`make dev-up`（仓库根目录）。
