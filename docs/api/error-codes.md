# API 错误码规范

本文档定义了 StuHelper API 的企业级错误码体系。

## 设计原则

1. **结构化编码**：采用 `{类别}{模块}{序号}` 格式，便于分类和追溯
2. **分层分类**：区分客户端错误、系统错误、第三方服务错误
3. **国际化友好**：错误码与错误消息分离，前端根据错误码进行翻译
4. **可扩展**：预留模块和序号空间，便于后续扩展

## 错误码格式

```
{类别}{模块}{序号}
  │     │    │
  │     │    └── 3位数字：具体错误序号 (001-999)
  │     └─────── 2位数字：业务模块 (00-99)
  └───────────── 1位字母：错误类别 (A/B/C)
```

**类别说明**：

| 类别 | 含义 | HTTP 状态码范围 | 说明 |
|------|------|-----------------|------|
| `A` | 客户端错误 | 4xx | 用户输入、权限、认证等问题 |
| `B` | 系统错误 | 5xx | 服务器内部错误、资源不足等 |
| `C` | 第三方服务错误 | 5xx | 依赖的外部服务异常 |

**模块编号**：

| 模块 | 编号 | 说明 |
|------|------|------|
| 通用 | 00 | 通用错误，不属于特定模块 |
| 认证 | 01 | 登录、Token、权限相关 |
| 用户 | 02 | 用户信息、账号相关 |
| 课程 | 10 | 课程、院系、教师相关 |
| 评课 | 11 | 测评、投票、评分相关 |
| 文件 | 20 | 文件上传、下载相关 |
| 通知 | 30 | 消息推送、订阅相关 |

## 错误响应格式

```json
{
  "success": false,
  "error": {
    "code": "A01001",
    "message": "token expired",
    "details": {
      "expired_at": "2026-01-28T10:00:00Z"
    }
  }
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `code` | string | 是 | 错误码，用于程序判断和国际化 |
| `message` | string | 是 | 错误描述（英文），用于开发调试 |
| `details` | object | 否 | 错误详情，包含上下文信息 |

---

## 错误码列表

### A00xxx - 通用客户端错误

| 错误码 | HTTP | 常量名 | 中文说明 | 英文说明 |
|--------|------|--------|----------|----------|
| `A00400` | 400 | `ErrBadRequest` | 请求参数错误 | Bad request |
| `A00401` | 400 | `ErrInvalidParam` | 参数格式无效 | Invalid parameter format |
| `A00402` | 400 | `ErrMissingParam` | 缺少必填参数 | Missing required parameter |
| `A00403` | 400 | `ErrParamOutOfRange` | 参数超出范围 | Parameter out of range |
| `A00404` | 404 | `ErrNotFound` | 资源不存在 | Resource not found |
| `A00405` | 405 | `ErrMethodNotAllowed` | 请求方法不允许 | Method not allowed |
| `A00409` | 409 | `ErrConflict` | 资源冲突 | Resource conflict |
| `A00413` | 413 | `ErrPayloadTooLarge` | 请求体过大 | Payload too large |
| `A00415` | 415 | `ErrUnsupportedMedia` | 不支持的媒体类型 | Unsupported media type |
| `A00422` | 422 | `ErrValidation` | 数据验证失败 | Validation failed |
| `A00429` | 429 | `ErrRateLimited` | 请求频率超限 | Rate limit exceeded |

### A01xxx - 认证与授权错误

| 错误码 | HTTP | 常量名 | 中文说明 | 英文说明 |
|--------|------|--------|----------|----------|
| `A01001` | 401 | `ErrTokenExpired` | Token 已过期 | Token expired |
| `A01002` | 401 | `ErrTokenInvalid` | Token 无效 | Invalid token |
| `A01003` | 401 | `ErrTokenMissing` | 未提供 Token | Token missing |
| `A01004` | 401 | `ErrTokenRevoked` | Token 已被撤销 | Token revoked |
| `A01005` | 401 | `ErrRefreshTokenExpired` | Refresh Token 已过期 | Refresh token expired |
| `A01006` | 401 | `ErrRefreshTokenInvalid` | Refresh Token 无效 | Invalid refresh token |
| `A01010` | 401 | `ErrLoginRequired` | 请先登录 | Login required |
| `A01011` | 401 | `ErrLoginFailed` | 登录失败 | Login failed |
| `A01012` | 401 | `ErrAccountDisabled` | 账号已禁用 | Account disabled |
| `A01013` | 401 | `ErrAccountLocked` | 账号已锁定 | Account locked |
| `A01020` | 403 | `ErrForbidden` | 权限不足 | Permission denied |
| `A01021` | 403 | `ErrAccessDenied` | 访问被拒绝 | Access denied |
| `A01022` | 403 | `ErrCSRFTokenInvalid` | CSRF Token 无效 | Invalid CSRF token |
| `A01030` | 401 | `ErrOAuthFailed` | OAuth 认证失败 | OAuth authentication failed |
| `A01031` | 401 | `ErrOAuthStateInvalid` | OAuth State 无效 | Invalid OAuth state |
| `A01032` | 401 | `ErrOAuthCodeInvalid` | OAuth 授权码无效 | Invalid OAuth code |

### A02xxx - 用户相关错误

| 错误码 | HTTP | 常量名 | 中文说明 | 英文说明 |
|--------|------|--------|----------|----------|
| `A02001` | 404 | `ErrUserNotFound` | 用户不存在 | User not found |
| `A02002` | 409 | `ErrUserExists` | 用户已存在 | User already exists |
| `A02003` | 400 | `ErrUsernameTaken` | 用户名已被占用 | Username already taken |
| `A02004` | 400 | `ErrEmailTaken` | 邮箱已被占用 | Email already taken |
| `A02005` | 400 | `ErrPasswordWeak` | 密码强度不足 | Password too weak |
| `A02006` | 400 | `ErrPasswordMismatch` | 密码不匹配 | Password mismatch |

### A10xxx - 课程模块错误

| 错误码 | HTTP | 常量名 | 中文说明 | 英文说明 |
|--------|------|--------|----------|----------|
| `A10001` | 404 | `ErrCourseNotFound` | 课程不存在 | Course not found |
| `A10002` | 404 | `ErrDepartmentNotFound` | 院系不存在 | Department not found |
| `A10003` | 404 | `ErrTeacherNotFound` | 教师不存在 | Teacher not found |
| `A10004` | 404 | `ErrTermNotFound` | 学期不存在 | Term not found |

### A11xxx - 评课模块错误

| 错误码 | HTTP | 常量名 | 中文说明 | 英文说明 |
|--------|------|--------|----------|----------|
| `A11001` | 404 | `ErrReviewNotFound` | 测评不存在 | Review not found |
| `A11002` | 409 | `ErrReviewExists` | 已评价过该课程 | Review already exists |
| `A11003` | 400 | `ErrReviewContentTooShort` | 测评内容过短 | Review content too short |
| `A11004` | 400 | `ErrReviewContentTooLong` | 测评内容过长 | Review content too long |
| `A11005` | 400 | `ErrRatingInvalid` | 评分无效 | Invalid rating |
| `A11006` | 400 | `ErrRatingDimensionMissing` | 缺少必填评分维度 | Missing required rating dimension |
| `A11010` | 409 | `ErrVoteExists` | 已投票过该测评 | Vote already exists |
| `A11011` | 400 | `ErrVoteTypeInvalid` | 投票类型无效 | Invalid vote type |
| `A11012` | 403 | `ErrVoteSelfReview` | 不能给自己的测评投票 | Cannot vote on own review |

### B00xxx - 系统通用错误

| 错误码 | HTTP | 常量名 | 中文说明 | 英文说明 |
|--------|------|--------|----------|----------|
| `B00001` | 500 | `ErrInternal` | 服务器内部错误 | Internal server error |
| `B00002` | 500 | `ErrDatabaseError` | 数据库错误 | Database error |
| `B00003` | 500 | `ErrCacheError` | 缓存错误 | Cache error |
| `B00004` | 503 | `ErrServiceUnavailable` | 服务暂时不可用 | Service unavailable |
| `B00005` | 503 | `ErrServiceOverloaded` | 服务过载 | Service overloaded |
| `B00006` | 504 | `ErrTimeout` | 请求超时 | Request timeout |
| `B00007` | 500 | `ErrConfigError` | 配置错误 | Configuration error |

### C00xxx - 第三方服务错误

| 错误码 | HTTP | 常量名 | 中文说明 | 英文说明 |
|--------|------|--------|----------|----------|
| `C00001` | 502 | `ErrUpstreamError` | 上游服务错误 | Upstream service error |
| `C00002` | 504 | `ErrUpstreamTimeout` | 上游服务超时 | Upstream service timeout |
| `C00003` | 503 | `ErrUpstreamUnavailable` | 上游服务不可用 | Upstream service unavailable |

### C01xxx - SSO 服务错误

| 错误码 | HTTP | 常量名 | 中文说明 | 英文说明 |
|--------|------|--------|----------|----------|
| `C01001` | 502 | `ErrSSOError` | SSO 服务错误 | SSO service error |
| `C01002` | 504 | `ErrSSOTimeout` | SSO 服务超时 | SSO service timeout |
| `C01003` | 503 | `ErrSSOUnavailable` | SSO 服务不可用 | SSO service unavailable |

---

## 兼容性说明

为保持向后兼容，系统同时支持旧版错误码。旧版错误码将在 v2.0 版本移除。

| 旧版错误码 | 新版错误码 | 说明 |
|------------|------------|------|
| `BAD_REQUEST` | `A00400` | 请求参数错误 |
| `UNAUTHORIZED` | `A01010` | 未授权 |
| `FORBIDDEN` | `A01020` | 权限不足 |
| `NOT_FOUND` | `A00404` | 资源不存在 |
| `CONFLICT` | `A00409` | 资源冲突 |
| `VALIDATION_ERROR` | `A00422` | 验证失败 |
| `RATE_LIMIT_EXCEEDED` | `A00429` | 频率超限 |
| `INTERNAL_ERROR` | `B00001` | 内部错误 |
| `SERVICE_UNAVAILABLE` | `B00004` | 服务不可用 |

---

## 前端处理建议

### 错误码国际化

```typescript
// i18n/errors.ts
export const errorMessages: Record<string, Record<string, string>> = {
  'zh-CN': {
    // 通用错误
    A00400: '请求参数错误',
    A00404: '资源不存在',
    A00422: '数据验证失败',
    A00429: '请求过于频繁，请稍后再试',
    // 认证错误
    A01001: '登录已过期，请重新登录',
    A01002: '登录凭证无效',
    A01010: '请先登录',
    A01020: '权限不足',
    // 评课错误
    A11002: '您已评价过该课程',
    A11003: '测评内容至少需要10个字',
    // 系统错误
    B00001: '服务器开小差了，请稍后再试',
    B00004: '服务暂时不可用',
  },
  'en-US': {
    A00400: 'Bad request',
    A00404: 'Resource not found',
    A00422: 'Validation failed',
    A00429: 'Too many requests, please try again later',
    A01001: 'Session expired, please login again',
    A01002: 'Invalid credentials',
    A01010: 'Please login first',
    A01020: 'Permission denied',
    A11002: 'You have already reviewed this course',
    A11003: 'Review content must be at least 10 characters',
    B00001: 'Server error, please try again later',
    B00004: 'Service unavailable',
  },
};
```

### 错误处理示例

```typescript
// utils/error-handler.ts
import { errorMessages } from '@/i18n/errors';

interface APIError {
  code: string;
  message: string;
  details?: Record<string, any>;
}

export function getErrorMessage(error: APIError, locale = 'zh-CN'): string {
  const messages = errorMessages[locale] || errorMessages['zh-CN'];

  // 优先使用国际化消息
  if (messages[error.code]) {
    return messages[error.code];
  }

  // 按错误类别返回通用消息
  if (error.code.startsWith('A')) {
    return messages['A00400'] || '请求错误';
  }
  if (error.code.startsWith('B')) {
    return messages['B00001'] || '服务器错误';
  }
  if (error.code.startsWith('C')) {
    return messages['B00004'] || '服务暂时不可用';
  }

  // 兜底返回原始消息
  return error.message;
}

// 判断是否需要重新登录
export function shouldRelogin(code: string): boolean {
  return ['A01001', 'A01002', 'A01003', 'A01004', 'A01010'].includes(code);
}

// 判断是否可以重试
export function isRetryable(code: string): boolean {
  return code.startsWith('B') || code.startsWith('C') || code === 'A00429';
}
```

---

## 后端使用示例

```go
// 使用预定义错误码
response.Error(c, http.StatusUnauthorized, errs.ErrTokenExpired, "token expired")

// 带详情的错误
response.ErrorWithDetails(c, http.StatusBadRequest, errs.ErrValidation, "validation failed", map[string]string{
    "field": "email",
    "reason": "invalid format",
})

// 业务错误
response.Error(c, http.StatusConflict, errs.ErrReviewExists, "already reviewed this course")
```

---

## 监控与告警

建议根据错误码类别配置不同的告警策略：

| 错误类别 | 告警级别 | 告警阈值 | 说明 |
|----------|----------|----------|------|
| `A01xxx` | INFO | > 100/min | 认证错误，可能有攻击 |
| `A00429` | WARN | > 1000/min | 频率限制触发过多 |
| `B00xxx` | ERROR | > 10/min | 系统错误需要关注 |
| `C00xxx` | WARN | > 50/min | 第三方服务异常 |
