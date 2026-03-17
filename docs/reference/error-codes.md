# API 错误码

当前错误码采用 8 位结构化格式，前后端都围绕这套编码做判断和展示。

## 当前格式

```text
{类别}{模块}{序号}
  │     │    │
  │     │    └── 4位数字：具体错误序号
  │     └─────── 3位数字：业务模块
  └───────────── 1位字母：错误类别
```

示例：`A0110100`

## 当前类别

| 类别 | HTTP 范围 | 说明 |
| --- | --- | --- |
| `A` | 4xx | 客户端错误 |
| `B` | 5xx | 系统错误 |
| `C` | 5xx | 第三方服务错误 |

## 当前模块编号

| 模块 | 编号 | 说明 |
| --- | --- | --- |
| 通用 | `000` | 通用错误 |
| 认证 | `001` | 登录、Token、权限 |
| 用户 | `002` | 用户信息、账号、认证 |
| 实名认证 | `003` | 实名认证、身份审核 |
| 学生认证 | `004` | 学生认证、审核、学籍 |
| RBAC | `005` | 角色、权限、用户组 |
| 课程 | `010` | 课程、院系、教师 |
| 评课 | `011` | 测评、投票、评分、审核 |

## 当前响应格式

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

## 当前错误码

### A000xxxx 通用客户端错误

| 错误码 | HTTP | 常量名 | 中文说明 | 英文说明 |
| --- | --- | --- | --- | --- |
| `A0000400` | 400 | `ErrBadRequest` | 请求参数错误 | Bad request |
| `A0000401` | 400 | `ErrInvalidParam` | 参数格式无效 | Invalid parameter format |
| `A0000402` | 400 | `ErrMissingParam` | 缺少必填参数 | Missing required parameter |
| `A0000403` | 400 | `ErrParamOutOfRange` | 参数超出范围 | Parameter out of range |
| `A0000404` | 404 | `ErrNotFound` | 资源不存在 | Resource not found |
| `A0000405` | 405 | `ErrMethodNotAllowed` | 请求方法不允许 | Method not allowed |
| `A0000409` | 409 | `ErrConflict` | 资源冲突 | Resource conflict |
| `A0000413` | 413 | `ErrPayloadTooLarge` | 请求体过大 | Payload too large |
| `A0000415` | 415 | `ErrUnsupportedMedia` | 媒体类型不支持 | Unsupported media type |
| `A0000422` | 422 | `ErrValidation` | 数据验证失败 | Validation failed |
| `A0000429` | 429 | `ErrRateLimited` | 请求频率超限 | Rate limit exceeded |

### A001xxxx 认证与授权错误

| 错误码 | HTTP | 常量名 | 中文说明 | 英文说明 |
| --- | --- | --- | --- | --- |
| `A0010001` | 401 | `ErrTokenExpired` | Token 已过期 | Token expired |
| `A0010002` | 401 | `ErrTokenInvalid` | Token 无效 | Invalid token |
| `A0010003` | 401 | `ErrTokenMissing` | Token 缺失 | Token missing |
| `A0010004` | 401 | `ErrTokenRevoked` | Token 已撤销 | Token revoked |
| `A0010005` | 401 | `ErrRefreshTokenExpired` | Refresh Token 已过期 | Refresh token expired |
| `A0010006` | 401 | `ErrRefreshTokenInvalid` | Refresh Token 无效 | Invalid refresh token |
| `A0010100` | 401 | `ErrLoginRequired` | 需要登录 | Login required |
| `A0010101` | 401 | `ErrLoginFailed` | 登录失败 | Login failed |
| `A0010102` | 401 | `ErrAccountDisabled` | 账号已禁用 | Account disabled |
| `A0010103` | 401 | `ErrAccountLocked` | 账号已锁定 | Account locked |
| `A0010200` | 403 | `ErrForbidden` | 权限不足 | Permission denied |
| `A0010201` | 403 | `ErrAccessDenied` | 访问被拒绝 | Access denied |
| `A0010202` | 403 | `ErrCSRFTokenInvalid` | CSRF Token 无效 | Invalid CSRF token |
| `A0010203` | 403 | `ErrCSRFTokenMissing` | CSRF Token 缺失 | CSRF token missing |
| `A0010300` | 401 | `ErrOAuthFailed` | OAuth 认证失败 | OAuth authentication failed |
| `A0010301` | 401 | `ErrOAuthStateInvalid` | OAuth State 无效 | Invalid OAuth state |
| `A0010302` | 401 | `ErrOAuthCodeInvalid` | OAuth 授权码无效 | Invalid OAuth code |

### A002xxxx 用户相关错误

| 错误码 | HTTP | 常量名 | 中文说明 | 英文说明 |
| --- | --- | --- | --- | --- |
| `A0020001` | 404 | `ErrUserNotFound` | 用户不存在 | User not found |
| `A0020002` | 409 | `ErrUserExists` | 用户已存在 | User already exists |
| `A0020003` | 400 | `ErrUsernameTaken` | 用户名已被占用 | Username already taken |
| `A0020004` | 400 | `ErrEmailTaken` | 邮箱已被占用 | Email already taken |
| `A0020005` | 400 | `ErrPasswordWeak` | 密码强度不足 | Password too weak |
| `A0020006` | 400 | `ErrPasswordMismatch` | 密码不匹配 | Password mismatch |

### A003xxxx 实名认证错误

| 错误码 | HTTP | 常量名 | 中文说明 | 英文说明 |
| --- | --- | --- | --- | --- |
| `A0030001` | 404 | `ErrIdentityNotFound` | 实名认证记录不存在 | Identity record not found |
| `A0030002` | 409 | `ErrIdentityAlreadyExists` | 已提交实名认证 | Identity already submitted |
| `A0030003` | 400 | `ErrIdentityDocInvalid` | 证件信息无效 | Invalid identity document |
| `A0030004` | 400 | `ErrIdentityNameMismatch` | 姓名与证件不匹配 | Identity name mismatch |
| `A0030005` | 400 | `ErrIdentityPhotoRequired` | 需要上传证件照片 | Identity photo required |
| `A0030006` | 400 | `ErrIdentityVerifyFailed` | 实名认证验证失败 | Identity verification failed |
| `A0030007` | 409 | `ErrIdentityAlreadyVerified` | 已通过实名认证 | Identity already verified |

### A004xxxx 学生认证错误

| 错误码 | HTTP | 常量名 | 中文说明 | 英文说明 |
| --- | --- | --- | --- | --- |
| `A0040001` | 404 | `ErrProfileNotFound` | 学生认证记录不存在 | Student profile not found |
| `A0040002` | 409 | `ErrProfileAlreadyVerified` | 已通过学生认证 | Student profile already verified |
| `A0040003` | 404 | `ErrProfileSchoolNotFound` | 学校配置不存在 | School configuration not found |
| `A0040004` | 400 | `ErrProfileSchoolDisabled` | 学校认证通道未开启 | School verification channel disabled |
| `A0040005` | 400 | `ErrProfileLDAPFailed` | LDAP 认证失败 | LDAP verification failed |
| `A0040006` | 400 | `ErrProfileConsentRequired` | 需要同意数据使用授权 | Consent is required |
| `A0040007` | 409 | `ErrProfilePendingReview` | 认证正在审核中 | Student verification is pending review |
| `A0040008` | 400 | `ErrProfilePhoneRequired` | 需要绑定手机号 | Phone number is required |
| `A0040009` | 400 | `ErrProfilePhoneMismatch` | 手机号需要验证 | Phone verification is required |
| `A0040011` | 400 | `ErrProfileAcademicTable` | 学籍表配置无效 | Academic table configuration is invalid |
| `A0040012` | 400 | `ErrAcademicTableNotConfigured` | 学校未配置学籍表 | School academic table is not configured |
| `A0040013` | 400 | `ErrSchoolLDAPConfigMissing` | 学校未配置 LDAP 连接 | LDAP configuration is missing for the school |
| `A0040014` | 400 | `ErrLDAPConfigInvalid` | 学校 LDAP 配置无效 | LDAP configuration is invalid for the school |

### A005xxxx RBAC 错误

| 错误码 | HTTP | 常量名 | 中文说明 | 英文说明 |
| --- | --- | --- | --- | --- |
| `A0050001` | 404 | `ErrRoleNotFound` | 角色不存在 | Role not found |
| `A0050002` | 409 | `ErrRoleNameTaken` | 角色名已存在 | Role name already taken |
| `A0050003` | 403 | `ErrRoleIsSystem` | 系统角色不可修改 | System role cannot be modified |
| `A0050004` | 404 | `ErrPermissionNotFound` | 权限不存在 | Permission not found |
| `A0050005` | 403 | `ErrPermissionDenied` | 权限不足 | Permission denied |
| `A0050006` | 404 | `ErrGroupNotFound` | 用户组不存在 | Group not found |
| `A0050007` | 409 | `ErrGroupNameTaken` | 用户组名已存在 | Group name already taken |
| `A0050008` | 400 | `ErrPermissionSelectionInvalid` | 权限选择无效 | One or more selected permissions are invalid |
| `A0050009` | 400 | `ErrRolePermissionClearConfirm` | 清空角色权限需要显式确认 | Clearing all role permissions requires confirmation |
| `A0050010` | 400 | `ErrRoleSelectionInvalid` | 角色选择无效 | One or more selected roles are invalid |
| `A0050011` | 400 | `ErrUserSelectionInvalid` | 用户选择无效 | One or more selected users are invalid |

### A010xxxx 课程错误

| 错误码 | HTTP | 常量名 | 中文说明 | 英文说明 |
| --- | --- | --- | --- | --- |
| `A0100001` | 404 | `ErrCourseNotFound` | 课程不存在 | Course not found |
| `A0100002` | 404 | `ErrDepartmentNotFound` | 院系不存在 | Department not found |
| `A0100003` | 404 | `ErrTeacherNotFound` | 教师不存在 | Teacher not found |
| `A0100004` | 404 | `ErrTermNotFound` | 学期不存在 | Term not found |

### A011xxxx 评课错误

| 错误码 | HTTP | 常量名 | 中文说明 | 英文说明 |
| --- | --- | --- | --- | --- |
| `A0110001` | 404 | `ErrReviewNotFound` | 测评不存在 | Review not found |
| `A0110002` | 409 | `ErrReviewExists` | 已评价过该课程 | Review already exists |
| `A0110003` | 400 | `ErrReviewContentTooShort` | 测评内容过短 | Review content too short |
| `A0110004` | 400 | `ErrReviewContentTooLong` | 测评内容过长 | Review content too long |
| `A0110005` | 404 | `ErrReplyNotFound` | 回复不存在 | Reply not found |
| `A0110006` | 404 | `ErrDraftNotFound` | 草稿不存在 | Draft not found |
| `A0110007` | 404 | `ErrReportNotFound` | 举报记录不存在 | Report not found |
| `A0110008` | 404 | `ErrSensitiveWordNotFound` | 敏感词不存在 | Sensitive word not found |
| `A0110009` | 400 | `ErrContentEmpty` | 内容为空 | Content is empty |
| `A0110010` | 403 | `ErrNotReviewOwner` | 测评 owner 校验失败 | Not review owner |
| `A0110011` | 403 | `ErrNotReplyOwner` | 回复 owner 校验失败 | Not reply owner |
| `A0110100` | 409 | `ErrVoteExists` | 已投票过该测评 | Vote already exists |
| `A0110101` | 400 | `ErrVoteTypeInvalid` | 投票类型无效 | Invalid vote type |
| `A0110102` | 403 | `ErrVoteSelfReview` | 本人测评投票动作受限 | Cannot vote on own review |
| `A0110103` | 409 | `ErrAlreadyReported` | 已举报过该内容 | Report already exists |
| `A0110104` | 400 | `ErrInvalidVoteAction` | 投票动作无效 | Invalid vote action |
| `A0110200` | 400 | `ErrRatingInvalid` | 评分无效 | Invalid rating |
| `A0110201` | 400 | `ErrRatingDimensionMissing` | 缺少必填评分维度 | Missing required rating dimension |
| `A0110300` | 400 | `ErrDangerousContent` | 内容包含危险元素 | Content contains dangerous elements |
| `A0110301` | 400 | `ErrSensitiveContent` | 内容包含敏感词 | Content contains sensitive words |
| `A0110302` | 400 | `ErrInvalidTransition` | 状态流转无效 | Invalid status transition |

### B000xxxx 系统错误

| 错误码 | HTTP | 常量名 | 中文说明 | 英文说明 |
| --- | --- | --- | --- | --- |
| `B0000001` | 500 | `ErrInternal` | 服务器内部错误 | Internal server error |
| `B0000002` | 500 | `ErrDatabaseError` | 数据库错误 | Database error |
| `B0000003` | 500 | `ErrCacheError` | 缓存错误 | Cache error |
| `B0000004` | 503 | `ErrServiceUnavailable` | 服务暂时不可用 | Service unavailable |
| `B0000005` | 503 | `ErrServiceOverloaded` | 服务过载 | Service overloaded |
| `B0000006` | 504 | `ErrTimeout` | 请求超时 | Request timeout |
| `B0000007` | 500 | `ErrConfigError` | 配置错误 | Configuration error |

### C000xxxx 第三方服务错误

| 错误码 | HTTP | 常量名 | 中文说明 | 英文说明 |
| --- | --- | --- | --- | --- |
| `C0000001` | 502 | `ErrUpstreamError` | 上游服务错误 | Upstream service error |
| `C0000002` | 504 | `ErrUpstreamTimeout` | 上游服务超时 | Upstream service timeout |
| `C0000003` | 503 | `ErrUpstreamUnavailable` | 上游服务不可用 | Upstream service unavailable |

### C001xxxx SSO 服务错误

| 错误码 | HTTP | 常量名 | 中文说明 | 英文说明 |
| --- | --- | --- | --- | --- |
| `C0010001` | 502 | `ErrSSOError` | SSO 服务错误 | SSO service error |
| `C0010002` | 504 | `ErrSSOTimeout` | SSO 服务超时 | SSO service timeout |
| `C0010003` | 503 | `ErrSSOUnavailable` | SSO 服务不可用 | SSO service unavailable |

## 中间件错误码

| 错误码 | HTTP | 常量名 | 说明 |
| --- | --- | --- | --- |
| `A0010203` | 403 | `ErrCSRFTokenMissing` | 缺少 CSRF Token |
| `A0010202` | 403 | `ErrCSRFTokenInvalid` | CSRF Token 无效 |
| `A0000413` | 413 | `ErrPayloadTooLarge` | 请求体超出大小限制 |

## 前端处理

前端 `ApiError.code` 直接保存后端错误码，也会保存浏览器侧网络错误码：

| 来源 | 错误码示例 | 说明 |
| --- | --- | --- |
| 后端 API | `A0110001`、`B0000001` | 读取 `error.code` |
| 浏览器网络层 | `NETWORK_ERROR`、`OFFLINE`、`TIMEOUT` | 前端生成的网络态错误码 |

前端通过错误码前缀判断行为类别：

```typescript
isAuthError(code); // A001*
isNetworkError(code); // NETWORK_ERROR | OFFLINE | TIMEOUT
isRetryable(code); // B* | C* | 网络错误
```

翻译键直接使用错误码：

```typescript
export default {
  NETWORK_ERROR: "网络连接失败",
  TIMEOUT: "请求超时",
  A0110001: "测评不存在",
  A0010001: "登录已过期，请重新登录",
  B0000001: "服务器错误",
};
```

兜底路径会在缺少 `error.code` 时使用 `httpStatusToDefaultCode(status)` 生成默认码。
