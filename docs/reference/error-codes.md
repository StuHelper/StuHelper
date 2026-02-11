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
  │     │    └── 4位数字：具体错误序号 (0001-9999)
  │     └─────── 3位数字：业务模块 (000-999)
  └───────────── 1位字母：错误类别 (A/B/C)

示例：A0110100 = A(客户端错误) + 011(评课模块) + 0100(投票错误)
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
| 通用 | 000 | 通用错误，不属于特定模块 |
| 认证 | 001 | 登录、Token、权限相关 |
| 用户 | 002 | 用户信息、账号相关 |
| 课程 | 010 | 课程、院系、教师相关 |
| 评课 | 011 | 测评、投票、评分相关 |
| 文件 | 020 | 文件上传、下载相关 |
| 通知 | 030 | 消息推送、订阅相关 |
| 040-099 | - | 业务模块扩展预留 |
| 100-999 | - | 未来扩展预留 |

## 错误响应格式

```json
{
  "success": false,
  "error": {
    "code": "A0010001",
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

### A000xxxx - 通用客户端错误

> 序号使用 HTTP 状态码便于记忆

| 错误码 | HTTP | 常量名 | 中文说明 | 英文说明 |
|--------|------|--------|----------|----------|
| `A0000400` | 400 | `ErrBadRequest` | 请求参数错误 | Bad request |
| `A0000401` | 400 | `ErrInvalidParam` | 参数格式无效 | Invalid parameter format |
| `A0000402` | 400 | `ErrMissingParam` | 缺少必填参数 | Missing required parameter |
| `A0000403` | 400 | `ErrParamOutOfRange` | 参数超出范围 | Parameter out of range |
| `A0000404` | 404 | `ErrNotFound` | 资源不存在 | Resource not found |
| `A0000405` | 405 | `ErrMethodNotAllowed` | 请求方法不允许 | Method not allowed |
| `A0000409` | 409 | `ErrConflict` | 资源冲突 | Resource conflict |
| `A0000413` | 413 | `ErrPayloadTooLarge` | 请求体过大 | Payload too large |
| `A0000415` | 415 | `ErrUnsupportedMedia` | 不支持的媒体类型 | Unsupported media type |
| `A0000422` | 422 | `ErrValidation` | 数据验证失败 | Validation failed |
| `A0000429` | 429 | `ErrRateLimited` | 请求频率超限 | Rate limit exceeded |

### A001xxxx - 认证与授权错误

> 分组: 0001-0099 Token | 0100-0199 登录 | 0200-0299 权限 | 0300-0399 OAuth

| 错误码 | HTTP | 常量名 | 中文说明 | 英文说明 |
|--------|------|--------|----------|----------|
| `A0010001` | 401 | `ErrTokenExpired` | Token 已过期 | Token expired |
| `A0010002` | 401 | `ErrTokenInvalid` | Token 无效 | Invalid token |
| `A0010003` | 401 | `ErrTokenMissing` | 未提供 Token | Token missing |
| `A0010004` | 401 | `ErrTokenRevoked` | Token 已被撤销 | Token revoked |
| `A0010005` | 401 | `ErrRefreshTokenExpired` | Refresh Token 已过期 | Refresh token expired |
| `A0010006` | 401 | `ErrRefreshTokenInvalid` | Refresh Token 无效 | Invalid refresh token |
| `A0010100` | 401 | `ErrLoginRequired` | 请先登录 | Login required |
| `A0010101` | 401 | `ErrLoginFailed` | 登录失败 | Login failed |
| `A0010102` | 401 | `ErrAccountDisabled` | 账号已禁用 | Account disabled |
| `A0010103` | 401 | `ErrAccountLocked` | 账号已锁定 | Account locked |
| `A0010200` | 403 | `ErrForbidden` | 权限不足 | Permission denied |
| `A0010201` | 403 | `ErrAccessDenied` | 访问被拒绝 | Access denied |
| `A0010202` | 403 | `ErrCSRFTokenInvalid` | CSRF Token 无效 | Invalid CSRF token |
| `A0010300` | 401 | `ErrOAuthFailed` | OAuth 认证失败 | OAuth authentication failed |
| `A0010301` | 401 | `ErrOAuthStateInvalid` | OAuth State 无效 | Invalid OAuth state |
| `A0010302` | 401 | `ErrOAuthCodeInvalid` | OAuth 授权码无效 | Invalid OAuth code |

### A002xxxx - 用户相关错误

| 错误码 | HTTP | 常量名 | 中文说明 | 英文说明 |
|--------|------|--------|----------|----------|
| `A0020001` | 404 | `ErrUserNotFound` | 用户不存在 | User not found |
| `A0020002` | 409 | `ErrUserExists` | 用户已存在 | User already exists |
| `A0020003` | 400 | `ErrUsernameTaken` | 用户名已被占用 | Username already taken |
| `A0020004` | 400 | `ErrEmailTaken` | 邮箱已被占用 | Email already taken |
| `A0020005` | 400 | `ErrPasswordWeak` | 密码强度不足 | Password too weak |
| `A0020006` | 400 | `ErrPasswordMismatch` | 密码不匹配 | Password mismatch |

### A010xxxx - 课程模块错误

| 错误码 | HTTP | 常量名 | 中文说明 | 英文说明 |
|--------|------|--------|----------|----------|
| `A0100001` | 404 | `ErrCourseNotFound` | 课程不存在 | Course not found |
| `A0100002` | 404 | `ErrDepartmentNotFound` | 院系不存在 | Department not found |
| `A0100003` | 404 | `ErrTeacherNotFound` | 教师不存在 | Teacher not found |
| `A0100004` | 404 | `ErrTermNotFound` | 学期不存在 | Term not found |

### A011xxxx - 评课模块错误

> 分组: 0001-0099 测评基础 | 0100-0199 投票 | 0200-0299 评分 | 0300-0399 内容审核

#### 测评基础错误 (0001-0099)

| 错误码 | HTTP | 常量名 | 中文说明 | 英文说明 |
|--------|------|--------|----------|----------|
| `A0110001` | 404 | `ErrReviewNotFound` | 测评不存在 | Review not found |
| `A0110002` | 409 | `ErrReviewExists` | 已评价过该课程 | Review already exists |
| `A0110003` | 400 | `ErrReviewContentTooShort` | 测评内容过短 | Review content too short |
| `A0110004` | 400 | `ErrReviewContentTooLong` | 测评内容过长 | Review content too long |

#### 投票相关错误 (0100-0199)

| 错误码 | HTTP | 常量名 | 中文说明 | 英文说明 |
|--------|------|--------|----------|----------|
| `A0110100` | 409 | `ErrVoteExists` | 已投票过该测评 | Vote already exists |
| `A0110101` | 400 | `ErrVoteTypeInvalid` | 投票类型无效 | Invalid vote type |
| `A0110102` | 403 | `ErrVoteSelfReview` | 不能给自己的测评投票 | Cannot vote on own review |

#### 评分相关错误 (0200-0299)

| 错误码 | HTTP | 常量名 | 中文说明 | 英文说明 |
|--------|------|--------|----------|----------|
| `A0110200` | 400 | `ErrRatingInvalid` | 评分无效 | Invalid rating |
| `A0110201` | 400 | `ErrRatingDimensionMissing` | 缺少必填评分维度 | Missing required rating dimension |

#### 内容审核错误 (0300-0399)

| 错误码 | HTTP | 常量名 | 中文说明 | 英文说明 |
|--------|------|--------|----------|----------|
| `A0110300` | 400 | `ErrDangerousContent` | 内容包含危险元素 | Content contains dangerous elements |

### B000xxxx - 系统通用错误

| 错误码 | HTTP | 常量名 | 中文说明 | 英文说明 |
|--------|------|--------|----------|----------|
| `B0000001` | 500 | `ErrInternal` | 服务器内部错误 | Internal server error |
| `B0000002` | 500 | `ErrDatabaseError` | 数据库错误 | Database error |
| `B0000003` | 500 | `ErrCacheError` | 缓存错误 | Cache error |
| `B0000004` | 503 | `ErrServiceUnavailable` | 服务暂时不可用 | Service unavailable |
| `B0000005` | 503 | `ErrServiceOverloaded` | 服务过载 | Service overloaded |
| `B0000006` | 504 | `ErrTimeout` | 请求超时 | Request timeout |
| `B0000007` | 500 | `ErrConfigError` | 配置错误 | Configuration error |

### C000xxxx - 第三方服务通用错误

| 错误码 | HTTP | 常量名 | 中文说明 | 英文说明 |
|--------|------|--------|----------|----------|
| `C0000001` | 502 | `ErrUpstreamError` | 上游服务错误 | Upstream service error |
| `C0000002` | 504 | `ErrUpstreamTimeout` | 上游服务超时 | Upstream service timeout |
| `C0000003` | 503 | `ErrUpstreamUnavailable` | 上游服务不可用 | Upstream service unavailable |

### C001xxxx - SSO 服务错误

| 错误码 | HTTP | 常量名 | 中文说明 | 英文说明 |
|--------|------|--------|----------|----------|
| `C0010001` | 502 | `ErrSSOError` | SSO 服务错误 | SSO service error |
| `C0010002` | 504 | `ErrSSOTimeout` | SSO 服务超时 | SSO service timeout |
| `C0010003` | 503 | `ErrSSOUnavailable` | SSO 服务不可用 | SSO service unavailable |

---

## 兼容性说明

为保持向后兼容，系统同时支持旧版错误码。旧版错误码将在 v2.0 版本移除。

| 旧版错误码 | 新版错误码 | 说明 |
|------------|------------|------|
| `BAD_REQUEST` | `A0000400` | 请求参数错误 |
| `UNAUTHORIZED` | `A0010100` | 未授权 |
| `FORBIDDEN` | `A0010200` | 权限不足 |
| `NOT_FOUND` | `A0000404` | 资源不存在 |
| `CONFLICT` | `A0000409` | 资源冲突 |
| `VALIDATION_ERROR` | `A0000422` | 验证失败 |
| `RATE_LIMIT_EXCEEDED` | `A0000429` | 频率超限 |
| `INTERNAL_ERROR` | `B0000001` | 内部错误 |
| `SERVICE_UNAVAILABLE` | `B0000004` | 服务不可用 |

---

## 前端处理建议

### 错误码国际化

```typescript
// i18n/errors.ts
export const errorMessages: Record<string, Record<string, string>> = {
  'zh-CN': {
    // 通用错误
    A0000400: '请求参数错误',
    A0000404: '资源不存在',
    A0000422: '数据验证失败',
    A0000429: '请求过于频繁，请稍后再试',
    // 认证错误
    A0010001: '登录已过期，请重新登录',
    A0010002: '登录凭证无效',
    A0010100: '请先登录',
    A0010200: '权限不足',
    // 评课错误
    A0110002: '您已评价过该课程',
    A0110003: '测评内容至少需要10个字',
    // 系统错误
    B0000001: '服务器开小差了，请稍后再试',
    B0000004: '服务暂时不可用',
  },
  'en-US': {
    A0000400: 'Bad request',
    A0000404: 'Resource not found',
    A0000422: 'Validation failed',
    A0000429: 'Too many requests, please try again later',
    A0010001: 'Session expired, please login again',
    A0010002: 'Invalid credentials',
    A0010100: 'Please login first',
    A0010200: 'Permission denied',
    A0110002: 'You have already reviewed this course',
    A0110003: 'Review content must be at least 10 characters',
    B0000001: 'Server error, please try again later',
    B0000004: 'Service unavailable',
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
    return messages['A0000400'] || '请求错误';
  }
  if (error.code.startsWith('B')) {
    return messages['B0000001'] || '服务器错误';
  }
  if (error.code.startsWith('C')) {
    return messages['B0000004'] || '服务暂时不可用';
  }

  // 兜底返回原始消息
  return error.message;
}

// 判断是否需要重新登录
export function shouldRelogin(code: string): boolean {
  return ['A0010001', 'A0010002', 'A0010003', 'A0010004', 'A0010100'].includes(code);
}

// 判断是否可以重试
export function isRetryable(code: string): boolean {
  return code.startsWith('B') || code.startsWith('C') || code === 'A0000429';
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
| `A001xxxx` | INFO | > 100/min | 认证错误，可能有攻击 |
| `A0000429` | WARN | > 1000/min | 频率限制触发过多 |
| `B000xxxx` | ERROR | > 10/min | 系统错误需要关注 |
| `C000xxxx` | WARN | > 50/min | 第三方服务异常 |
