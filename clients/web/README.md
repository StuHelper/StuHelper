# Web 主站

Vue 3 + Vite + TypeScript SPA。

## 目录

```
src/
├── api/           # 主站 API 入口（基于 @stuhelper/shared）
├── components/    # 通用组件
├── composables/   # 组合式函数
├── modules/       # auth / course / review / user / home / errors
├── router/
├── stores/        # Pinia
├── utils/
└── i18n/
```

## 技术栈

Vue 3.5 / Vite 6 / TypeScript 5 / Pinia / Vue Router 4 / Element Plus / ECharts

## API 调用

统一从 `src/api/index.ts` 的 `api` 对象调用，不直接 `fetch`。类型来自 `@stuhelper/shared`。

## 开发

```bash
cd clients
pnpm install && pnpm dev:web
```

## 校验

```bash
cd clients
pnpm --filter @stuhelper/web lint
pnpm --filter @stuhelper/web type-check
pnpm --filter @stuhelper/web test
pnpm --filter @stuhelper/web test:e2e
pnpm --filter @stuhelper/web build
```
