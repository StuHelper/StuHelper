# Error Code Unification Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Eliminate the redundant `ErrorCode` enum and `backendCode` bridge from the frontend, making backend 8-digit codes the single error identifier for all API errors.

**Architecture:** Frontend `ApiError.code` becomes a plain `string` holding either a backend 8-digit code (e.g. `A0110001`) or one of 3 client-only codes (`NETWORK_ERROR`, `OFFLINE`, `TIMEOUT`). The `ErrorCode` enum, `backendCode` field, `statusMap`, and `createErrorFromStatus()` are deleted. Behavior classification uses code prefix (`A001` = auth). i18n uses flat single-level lookup.

**Tech Stack:** TypeScript, Vue 3, vue-i18n, axios

**Design doc:** `docs/plans/2026-03-03-error-code-unification-design.md`

---

## Task 1: Rewrite `api/errors.ts`

**Files:**
- Rewrite: `clients/web/course/src/api/errors.ts` (entire file)

This is the core change. The file is fully replaced.

**Step 1: Replace the entire file**

```ts
/**
 * API 错误类型定义
 * 统一的错误处理机制 — 错误码即后端 8 位码或客户端专属码
 *
 * 错误码值域:
 *   - API 错误: 后端 8 位结构化码 (A0110001, B0000001 等)
 *   - 客户端专属: NETWORK_ERROR, OFFLINE, TIMEOUT
 *
 * 行为判断通过前缀: A001=认证, B/C=可重试
 * 详见 docs/reference/error-codes.md
 */
import i18n from '@/i18n'

// 错误严重级别
export type ErrorSeverity = 'info' | 'warning' | 'error' | 'critical'

// API 错误类
export class ApiError extends Error {
  readonly code: string
  readonly status?: number
  readonly severity: ErrorSeverity
  readonly details?: Record<string, unknown>
  readonly timestamp: Date
  readonly requestID?: string

  constructor(options: {
    message: string
    code: string
    status?: number
    severity?: ErrorSeverity
    details?: Record<string, unknown>
    requestID?: string
  }) {
    super(options.message)
    Object.setPrototypeOf(this, ApiError.prototype)
    this.name = 'ApiError'
    this.code = options.code
    this.status = options.status
    this.severity = options.severity ?? 'error'
    this.details = options.details
    this.timestamp = new Date()
    this.requestID = options.requestID
  }

  // 获取用户友好的错误消息（单层 i18n 查找）
  getUserMessage(): string {
    const { t, te } = i18n.global
    const key = `errors.${this.code}`
    return te(key) ? t(key) : this.message
  }

  toJSON() {
    return {
      name: this.name,
      message: this.message,
      code: this.code,
      status: this.status,
      severity: this.severity,
      details: this.details,
      timestamp: this.timestamp.toISOString(),
      requestID: this.requestID
    }
  }
}

// ---- 行为判断工具函数（基于错误码前缀） ----

// 认证相关错误 (A001xxxx)
export function isAuthError(code: string): boolean {
  return code.startsWith('A001')
}

// 网络/客户端错误（后端不可能返回）
export function isNetworkError(code: string): boolean {
  return code === 'NETWORK_ERROR' || code === 'OFFLINE' || code === 'TIMEOUT'
}

// 可重试错误（系统错误、第三方服务错误、网络错误）
export function isRetryable(code: string): boolean {
  return code.startsWith('B') || code.startsWith('C') || isNetworkError(code)
}

// 类型守卫
export function isApiError(error: unknown): error is ApiError {
  return error instanceof ApiError
}

// 后端未返回 code 时的 HTTP 状态码兜底映射
export function httpStatusToDefaultCode(status: number): string {
  const map: Record<number, string> = {
    400: 'A0000400',
    401: 'A0010100',
    403: 'A0010200',
    404: 'A0000404',
    409: 'A0000409',
    422: 'A0000422',
    429: 'A0000429',
    500: 'B0000001',
    502: 'C0000001',
    503: 'B0000004',
    504: 'B0000006',
  }
  return map[status] || 'B0000001'
}
```

**Step 2: Run type-check to verify (will fail — consumers still import old exports)**

Run: `cd clients/web/course && npx vue-tsc --noEmit 2>&1 | head -30`

Expected: Errors in `index.ts`, `auth.ts`, `courseReview.ts`, `AuthCallbackPage.vue` referencing `ErrorCode` or `createErrorFromStatus`. This confirms the blast radius matches our plan.

**Step 3: Commit (partial — will fix consumers next)**

```bash
git add clients/web/course/src/api/errors.ts
git commit -m "refactor(errors): replace ErrorCode enum with unified string codes"
```

---

## Task 2: Rewrite `api/index.ts` — update imports and `transformError`

**Files:**
- Modify: `clients/web/course/src/api/index.ts`

**Step 1: Update imports (line 12)**

Change:
```ts
import { ApiError, ErrorCode, createErrorFromStatus } from './errors'
```
To:
```ts
import { ApiError, isNetworkError, httpStatusToDefaultCode } from './errors'
```

**Step 2: Update `createAuthError()` (lines 155-161)**

Change:
```ts
  private createAuthError(): ApiError {
    return new ApiError({
      message: i18n.global.t('errors.TOKEN_EXPIRED'),
      code: ErrorCode.TOKEN_EXPIRED,
      status: 401
    })
  }
```
To:
```ts
  private createAuthError(): ApiError {
    return new ApiError({
      message: i18n.global.t('errors.A0010001'),
      code: 'A0010001',
      status: 401
    })
  }
```

**Step 3: Update `addOfflineInterceptor()` (lines 170-175)**

Change:
```ts
        new ApiError({
          message: i18n.global.t('errors.OFFLINE'),
          code: ErrorCode.OFFLINE
        })
```
To:
```ts
        new ApiError({
          message: i18n.global.t('errors.OFFLINE'),
          code: 'OFFLINE'
        })
```

**Step 4: Rewrite `transformError()` (lines 196-242)**

Replace the entire function:
```ts
function transformError(error: AxiosError): ApiError {
  if (error.code === 'ECONNABORTED' || error.message.includes('timeout')) {
    return new ApiError({
      message: i18n.global.t('errors.TIMEOUT'),
      code: 'TIMEOUT'
    })
  }

  if (!error.response) {
    return new ApiError({
      message: i18n.global.t('errors.NETWORK_ERROR'),
      code: 'NETWORK_ERROR'
    })
  }

  const { status, data } = error.response
  const responseData = isErrorResponseBody(data) ? data : undefined

  let code: string | undefined
  let message: string | undefined
  let details: Record<string, unknown> | undefined

  if (
    responseData?.error &&
    typeof responseData.error === 'object' &&
    responseData.error !== null
  ) {
    if ('code' in responseData.error && typeof responseData.error.code === 'string') {
      code = responseData.error.code
    }
    if ('message' in responseData.error) {
      message = responseData.error.message
    }
    if ('details' in responseData.error) {
      details = responseData.error.details
    }
  } else {
    message = responseData?.message ||
      (typeof responseData?.error === 'string' ? responseData.error : undefined)
  }

  // 后端未返回 code 时，用 HTTP 状态码映射到默认 8 位码
  if (!code) {
    code = httpStatusToDefaultCode(status)
  }

  return new ApiError({
    message: message || `HTTP ${status}`,
    code,
    status,
    details
  })
}
```

**Step 5: Commit**

```bash
git add clients/web/course/src/api/index.ts
git commit -m "refactor(api): simplify transformError to use backend codes directly"
```

---

## Task 3: Flatten `i18n/locales/zh-CN/errors.ts`

**Files:**
- Rewrite: `clients/web/course/src/i18n/locales/zh-CN/errors.ts`

**Step 1: Replace the entire file**

```ts
/**
 * 错误消息 - 中文
 * 错误码与后端 8 位码或客户端码对应，见 docs/reference/error-codes.md
 */
export default {
  // 客户端专属码（后端不会返回）
  NETWORK_ERROR: '网络连接失败，请检查网络设置',
  OFFLINE: '当前无网络连接，请检查网络后重试',
  TIMEOUT: '请求超时，请稍后重试',

  // A000xxxx - 通用客户端错误
  A0000400: '请求参数错误',
  A0000404: '请求的资源不存在',
  A0000409: '资源冲突',
  A0000422: '数据验证失败',
  A0000429: '请求过于频繁，请稍后重试',

  // A001xxxx - 认证与授权
  A0010001: '登录已过期，请重新登录',
  A0010002: '登录信息无效，请重新登录',
  A0010003: '请先登录',
  A0010100: '请先登录',
  A0010200: '没有权限执行此操作',
  A0010201: '访问被拒绝',
  A0010202: 'CSRF 验证失败，请刷新页面重试',
  A0010203: 'CSRF 令牌缺失，请刷新页面重试',

  // A010xxxx - 课程模块
  A0100001: '课程不存在',
  A0100003: '教师不存在',

  // A011xxxx - 评课模块
  A0110001: '测评不存在',
  A0110002: '你已评价过该课程',
  A0110005: '回复不存在',
  A0110006: '草稿不存在',
  A0110007: '举报记录不存在',
  A0110008: '敏感词不存在',
  A0110010: '你无权操作此测评',
  A0110011: '你无权操作此回复',
  A0110100: '你已投票过该测评',
  A0110102: '不能给自己的测评投票',
  A0110103: '你已举报过该内容',
  A0110301: '内容包含敏感词，请修改后重试',
  A0110302: '无效的状态转换',

  // B000xxxx - 系统错误
  B0000001: '服务器错误，请稍后重试',
  B0000004: '服务暂时不可用，请稍后重试',
  B0000006: '请求超时，请稍后重试',

  // C000xxxx - 第三方服务错误
  C0000001: '上游服务错误',

  // 错误页面
  notFound: {
    title: '页面不存在',
    description: '你访问的页面可能已被移除或地址有误',
    backHome: '返回首页',
    goBack: '返回上页'
  },
  loadError: {
    title: '加载失败',
    description: '页面加载出现问题，请刷新重试',
    reload: '刷新页面'
  },

  // 错误边界
  boundary: {
    title: '页面出现了问题',
    description: '渲染过程中发生了意外错误，请尝试刷新页面',
    reload: '刷新页面'
  },

  // 认证回调
  authCallback: {
    loading: '正在登录中...',
    error: '登录失败',
    backToLogin: '返回登录',
    missingCode: '缺少授权码',
    missingState: '缺少 state 参数',
    loginFailed: '登录失败，请重试',
    orgMismatch: '当前 SSO 账户无权访问本系统',
    orgMismatchHint: '请注销当前账户后，使用正确的账户登录',
    ssoLogout: '注销并重新登录'
  }
}
```

**Step 2: Commit**

```bash
git add clients/web/course/src/i18n/locales/zh-CN/errors.ts
git commit -m "refactor(i18n): flatten zh-CN error translations to single level"
```

---

## Task 4: Flatten `i18n/locales/en-US/errors.ts`

**Files:**
- Rewrite: `clients/web/course/src/i18n/locales/en-US/errors.ts`

**Step 1: Replace the entire file**

```ts
/**
 * Error messages - English
 * Error codes correspond to backend 8-digit codes or client codes, see docs/reference/error-codes.md
 */
export default {
  // Client-only codes (backend never returns these)
  NETWORK_ERROR: 'Network connection failed, please check your network',
  OFFLINE: 'You are offline. Please check your connection and try again',
  TIMEOUT: 'Request timed out, please try again',

  // A000xxxx - General client errors
  A0000400: 'Invalid request parameters',
  A0000404: 'The requested resource was not found',
  A0000409: 'Resource conflict',
  A0000422: 'Validation failed',
  A0000429: 'Too many requests, please try again later',

  // A001xxxx - Auth and authorization
  A0010001: 'Session expired, please log in again',
  A0010002: 'Invalid session, please log in again',
  A0010003: 'Please log in first',
  A0010100: 'Please log in first',
  A0010200: 'You do not have permission to perform this action',
  A0010201: 'Access denied',
  A0010202: 'CSRF validation failed, please refresh the page',
  A0010203: 'CSRF token missing, please refresh the page',

  // A010xxxx - Course module
  A0100001: 'Course not found',
  A0100003: 'Teacher not found',

  // A011xxxx - Review module
  A0110001: 'Review not found',
  A0110002: 'You have already reviewed this course',
  A0110005: 'Reply not found',
  A0110006: 'Draft not found',
  A0110007: 'Report not found',
  A0110008: 'Sensitive word not found',
  A0110010: 'You do not own this review',
  A0110011: 'You do not own this reply',
  A0110100: 'You have already voted on this review',
  A0110102: 'You cannot vote on your own review',
  A0110103: 'You have already reported this content',
  A0110301: 'Content contains sensitive words, please revise',
  A0110302: 'Invalid status transition',

  // B000xxxx - System errors
  B0000001: 'Server error, please try again later',
  B0000004: 'Service temporarily unavailable',
  B0000006: 'Request timed out, please try again later',

  // C000xxxx - Third-party service errors
  C0000001: 'Upstream service error',

  // Error pages
  notFound: {
    title: 'Page Not Found',
    description: 'The page you visited may have been removed or the address is incorrect',
    backHome: 'Back to Home',
    goBack: 'Go Back'
  },
  loadError: {
    title: 'Load Failed',
    description: 'An error occurred while loading the page. Please refresh and try again',
    reload: 'Refresh Page'
  },

  // Error boundary
  boundary: {
    title: 'Something went wrong',
    description: 'An unexpected error occurred while rendering the page. Please try refreshing.',
    reload: 'Refresh Page'
  },

  // Auth callback
  authCallback: {
    loading: 'Logging in...',
    error: 'Login Failed',
    backToLogin: 'Back to Login',
    missingCode: 'Missing authorization code',
    missingState: 'Missing state parameter',
    loginFailed: 'Login failed, please retry',
    orgMismatch: 'Your SSO account does not have access to this system',
    orgMismatchHint: 'Please log out and sign in with the correct account',
    ssoLogout: 'Log out and sign in again'
  }
}
```

**Step 2: Commit**

```bash
git add clients/web/course/src/i18n/locales/en-US/errors.ts
git commit -m "refactor(i18n): flatten en-US error translations to single level"
```

---

## Task 5: Update consumer — `stores/auth.ts`

**Files:**
- Modify: `clients/web/course/src/stores/auth.ts`

**Step 1: Update import (line 8)**

Change:
```ts
import { isApiError } from '@/api/errors'
```
To:
```ts
import { isApiError, isNetworkError } from '@/api/errors'
```

**Step 2: Update `handleError()` (line 54)**

Change:
```ts
      if (err.isNetworkError()) {
```
To:
```ts
      if (isNetworkError(err.code)) {
```

**Step 3: Update `fetchUser()` catch block (line 141)**

Change:
```ts
      if (isApiError(err) && !err.isNetworkError()) {
```
To:
```ts
      if (isApiError(err) && !isNetworkError(err.code)) {
```

**Step 4: Commit**

```bash
git add clients/web/course/src/stores/auth.ts
git commit -m "refactor(auth): use standalone isNetworkError function"
```

---

## Task 6: Update consumer — `stores/courseReview.ts`

**Files:**
- Modify: `clients/web/course/src/stores/courseReview.ts`

**Step 1: Update import (line 9)**

Change:
```ts
import { isApiError } from '@/api/errors'
```
To:
```ts
import { isApiError, isNetworkError } from '@/api/errors'
```

**Step 2: Update `handleError()` (line 111)**

Change:
```ts
      if (err.isNetworkError()) {
```
To:
```ts
      if (isNetworkError(err.code)) {
```

**Step 3: Commit**

```bash
git add clients/web/course/src/stores/courseReview.ts
git commit -m "refactor(courseReview): use standalone isNetworkError function"
```

---

## Task 7: Update consumer — `views/AuthCallbackPage.vue`

**Files:**
- Modify: `clients/web/course/src/views/AuthCallbackPage.vue`

**Step 1: Update import (line 45)**

Change:
```ts
import { isApiError, ErrorCode } from '@/api/errors'
```
To:
```ts
import { isApiError } from '@/api/errors'
```

**Step 2: Update error check (line 94)**

Change:
```ts
    if (isApiError(err) && err.code === ErrorCode.FORBIDDEN) {
```
To:
```ts
    if (isApiError(err) && err.status === 403) {
```

Note: Using `err.status === 403` here because AuthCallbackPage cares about the HTTP-level behavior (403 = org mismatch from SSO). The backend may return any `A001020x` code for different 403 reasons, but all 403s in callback context mean org mismatch.

**Step 3: Commit**

```bash
git add clients/web/course/src/views/AuthCallbackPage.vue
git commit -m "refactor(auth-callback): use HTTP status instead of ErrorCode enum"
```

---

## Task 8: Update documentation — `docs/reference/error-codes.md`

**Files:**
- Modify: `docs/reference/error-codes.md` (lines 234-300 — "前端处理建议" section)

**Step 1: Replace the "前端处理建议" section (lines 234-300)**

Replace everything from `## 前端处理建议` through end of that section with:

```markdown
## 前端处理

### 错误码使用

前端 `ApiError.code` 直接存储后端 8 位码或 3 个客户端专属码：

| 来源 | 错误码示例 | 说明 |
|------|-----------|------|
| 后端 API 响应 | `A0110001`, `B0000001` | 从 `error.code` 字段直接提取 |
| 客户端（离线/超时） | `NETWORK_ERROR`, `OFFLINE`, `TIMEOUT` | 前端自行生成，后端不会返回 |

```typescript
// api/errors.ts
export class ApiError extends Error {
  readonly code: string  // 'A0110001' 或 'NETWORK_ERROR'

  getUserMessage(): string {
    const key = `errors.${this.code}`
    return te(key) ? t(key) : this.message
  }
}
```

### 行为判断

通过错误码前缀判断行为类别，不再使用枚举：

```typescript
isAuthError(code)   // code.startsWith('A001') — 认证错误
isNetworkError(code) // NETWORK_ERROR | OFFLINE | TIMEOUT — 网络错误
isRetryable(code)   // B* | C* | 网络错误 — 可重试
```

### 错误国际化

所有错误码翻译全部平级，不存在嵌套命名空间：

```typescript
// i18n/locales/zh-CN/errors.ts
export default {
  NETWORK_ERROR: '网络连接失败',
  TIMEOUT: '请求超时',
  A0110001: '测评不存在',
  A0010001: '登录已过期，请重新登录',
  B0000001: '服务器错误',
  // ...
}
```

### 兜底机制

当后端响应不包含 `error.code` 字段时（异常情况），前端使用 `httpStatusToDefaultCode(status)` 将 HTTP 状态码映射到默认 8 位码。正常情况下后端总会返回 `code` 字段。
```

**Step 2: Commit**

```bash
git add docs/reference/error-codes.md
git commit -m "docs(error-codes): update frontend handling section for unified codes"
```

---

## Task 9: Verify

**Step 1: Run TypeScript type-check**

Run: `cd clients/web/course && npx vue-tsc --noEmit`

Expected: Zero errors.

**Step 2: Run ESLint**

Run: `cd clients/web/course && npx eslint src/api/errors.ts src/api/index.ts src/stores/auth.ts src/stores/courseReview.ts src/views/AuthCallbackPage.vue`

Expected: Zero errors (or only pre-existing warnings).

**Step 3: Run Vite build**

Run: `cd clients/web/course && npx vite build`

Expected: Build succeeds.

**Step 4: Verify no old references remain**

Run: `grep -rn 'ErrorCode\.' clients/web/course/src/` — Expected: zero matches.
Run: `grep -rn 'backendCode' clients/web/course/src/` — Expected: zero matches.
Run: `grep -rn 'createErrorFromStatus' clients/web/course/src/` — Expected: zero matches.
Run: `grep -rn 'BUSINESS_ERROR\|UNKNOWN\|SERVER_ERROR\|SERVICE_UNAVAILABLE\|UNAUTHORIZED\|TOKEN_EXPIRED\|INVALID_TOKEN\|FORBIDDEN\|BAD_REQUEST\|NOT_FOUND\|VALIDATION_ERROR\|CONFLICT\|RATE_LIMIT_EXCEEDED' clients/web/course/src/i18n/` — Expected: zero matches (old HTTP-semantic keys removed from i18n).

**Step 5: Squash or keep commits, update archiving.md**

Update `.project_rule/archiving.md` with the change record.
