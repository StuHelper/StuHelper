# OpenAPI 3 开发指南

StuHelper 当前采用 OpenAPI 3 Spec-First 流程。OpenAPI 是前后端共享的契约源。

## 权威来源在哪里

```text
server/api/openapi.yaml
```

这个文件和它拆分出去的 `paths/`、`components/` 目录，定义了：

- 路径
- 参数
- 请求体
- 响应体
- 错误结构

## 生成链路

运行：

```bash
cd server
make generate
```

会产生两类结果：

1. 后端生成代码：`server/internal/api/gen/`
2. 前端类型：`clients/shared/src/types/api.gen.ts`

前端随后通过这几个位置接入：

- `clients/shared/src/api/client.ts`：基础 `openapi-fetch` 客户端
- `clients/shared/src/api/*.ts`：领域级 API 包装
- `clients/web/src/api/client.ts`：浏览器端 Cookie / CSRF / refresh 适配
- `clients/web/src/api/index.ts`：Web 项目兼容层与聚合出口

## 新增接口的标准流程

### 1. 先改规范

例如你要新增“获取课程详情”：

1. 在 `server/api/paths/` 补路径定义
2. 在 `server/api/components/schemas/` 补响应结构
3. 给操作命名 `operationId`

### 2. 重新生成

```bash
cd server
make generate
```

### 3. 实现后端

在 `server/internal/modules/<domain>/` 中完成：

- Repository
- Service
- Handler

### 4. 接入前端

如果前端还没有该领域方法：

1. 在 `clients/shared/src/api/` 新增或扩展领域 API 包装
2. 必要时在 `clients/web/src/api/index.ts` 补 Web 侧兼容包装
3. 页面中通过 `api.xxx.yyy()` 或兼容包装函数调用

## 前端调用约定

### 推荐

```typescript
import { api } from "@/api";

const res = await api.course.getCourse("123");
const course = res.data?.data;
```

### 当前链路之外的写法

```typescript
const res = await fetch("/api/v1/course/courses/123");
const json = await res.json();
```

裸请求缺少 OpenAPI 类型、Cookie 会话、CSRF 头和刷新逻辑。

## 什么时候需要重新生成

只要发生以下任一情况，就应该重新跑 `make generate`：

- 新增接口
- 修改参数
- 修改 schema
- 修改错误响应
- 调整字段是否可选

## 检查漂移

提交前至少运行：

```bash
cd server
make check-drift
```

如果生成产物与规范不一致，这一步会直接失败。

## 常见错误

### 后端改了字段，前端没报错

通常说明你没有重新生成类型，或者页面绕开了共享 API 层。

### 前端类型更新了，但接口还是调用失败

类型保证契约一致，后端实现还需要继续检查：

- 后端 handler 是否已注册
- 路径和方法是否与规范一致
- 实际返回值是否遵守统一响应格式

## 常用命令

```bash
cd server
make lint-spec
make generate
make check-drift
```
