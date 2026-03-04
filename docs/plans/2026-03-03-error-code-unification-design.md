# 错误码体系统一设计

**日期**: 2026-03-03
**状态**: 已批准

## 背景

当前错误系统存在 6 个重叠概念：
1. 后端 8 位结构化码 (`errs.ErrorCode`)
2. 后端便捷函数隐含的 HTTP 状态码 (`response.NotFound()` → 404)
3. 前端 `ErrorCode` 枚举（17 个 HTTP 语义值）
4. 前端 `backendCode` 桥接字段
5. 前端 `statusMap` HTTP→枚举映射
6. 前端三级 i18n 回退

前端的 `ErrorCode` 枚举本质上是 HTTP 状态码的重命名，而后端已有精确的 8 位码。中间层完全冗余。

## 设计决策

### 单一错误码标识

`ApiError.code` 统一为 `string` 类型，值域：
- **API 错误**：后端 8 位码（`A0110001`、`B0000001` 等）
- **客户端专属**：3 个字符串常量（`NETWORK_ERROR`、`OFFLINE`、`TIMEOUT`）

### 行为判断基于前缀

```ts
function isAuthError(code: string): boolean {
  return code.startsWith('A001')
}
function isNetworkError(code: string): boolean {
  return code === 'NETWORK_ERROR' || code === 'OFFLINE' || code === 'TIMEOUT'
}
function isRetryable(code: string): boolean {
  return code.startsWith('B') || code.startsWith('C') || isNetworkError(code)
}
```

### 单层 i18n

所有错误码翻译全部平级，不再有 `backend` 命名空间：

```ts
errors: {
  NETWORK_ERROR: '网络连接失败',
  OFFLINE: '无网络连接',
  TIMEOUT: '请求超时',
  A0110001: '测评不存在',
  A0010001: '登录已过期',
  B0000001: '服务器错误',
  // ...
}
```

`getUserMessage()` 简化为一次查找：
```ts
getUserMessage(): string {
  const key = `errors.${this.code}`
  return te(key) ? t(key) : this.message
}
```

### 兜底映射

当后端未返回 `code` 字段时（异常情况），用 HTTP 状态码映射到默认 8 位码：

| HTTP | 默认码 |
|------|--------|
| 400 | A0000400 |
| 401 | A0010100 |
| 403 | A0010200 |
| 404 | A0000404 |
| 409 | A0000409 |
| 422 | A0000422 |
| 429 | A0000429 |
| 500 | B0000001 |
| 502 | C0000001 |
| 503 | B0000004 |
| 504 | B0000006 |

## 删除清单

| 删除项 | 文件 |
|--------|------|
| `ErrorCode` 枚举（17 值） | `errors.ts` |
| `backendCode` 字段 | `errors.ts` |
| `statusMap` 映射 | `errors.ts` |
| `createErrorFromStatus()` | `errors.ts` |
| `isAuthError()` 实例方法 | `errors.ts` |
| `isRetryable()` 实例方法 | `errors.ts` |
| i18n `backend` 命名空间 | `zh-CN/errors.ts`, `en-US/errors.ts` |
| i18n 旧 HTTP 语义 key | `zh-CN/errors.ts`, `en-US/errors.ts` |

## 新增清单

| 新增项 | 文件 |
|--------|------|
| `isAuthError(code)` 独立函数 | `errors.ts` |
| `isNetworkError(code)` 独立函数 | `errors.ts` |
| `isRetryable(code)` 独立函数 | `errors.ts` |
| `httpStatusToDefaultCode()` 兜底函数 | `errors.ts` |
| 全部 8 位码平级 i18n key | `zh-CN/errors.ts`, `en-US/errors.ts` |

## 影响范围

- **后端**：零改动
- **前端改动文件**：5 个
  - `api/errors.ts` — 主要重构
  - `api/index.ts` — transformError 简化
  - `i18n/locales/zh-CN/errors.ts` — 扁平化
  - `i18n/locales/en-US/errors.ts` — 扁平化
  - `stores/auth.ts` — `err.isNetworkError()` → `isNetworkError(err.code)`
  - `stores/courseReview.ts` — 同上
  - `views/AuthCallbackPage.vue` — `ErrorCode.FORBIDDEN` → `err.status === 403`

## 重构前后对比

| 维度 | 重构前 | 重构后 |
|------|--------|--------|
| 错误标识层数 | 3 层 | 1 层 |
| i18n 查找 | 3 级回退 | 1 次 |
| ApiError 字段数 | 8 个 | 6 个 |
| 前端枚举值 | 17 个 | 0（纯 string） |
| 后端改动 | - | 0 |

## 更新文档

- `docs/reference/error-codes.md` — 更新前端处理章节
