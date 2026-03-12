# StuHelper 前端项目

## 开发环境要求

- Node.js >= 18
- pnpm >= 8

## 安装依赖

```bash
pnpm install
```

## 启动开发服务器

### Web 项目 (H5)

```bash
cd clients
pnpm dev:web
```

访问: http://localhost:3000

### uni-app x 项目 (H5)

```bash
cd clients
pnpm dev:uni
```

端口以开发服务器输出为准。

## 构建

### Web 项目

```bash
cd clients
pnpm build:web
```

### uni-app x 项目 (小程序)

```bash
cd clients
pnpm build:uni:mp
```

## 类型检查

```bash
cd clients
pnpm type-check
```
