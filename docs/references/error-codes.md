# API 错误码

权威来源：`server/internal/pkg/errs/codes.go`

## 编码结构

```
{类别}{模块}{序号}
  1位   3位   4位
```

## 类别

| 类别 | HTTP | 说明 |
|------|------|------|
| `A` | 4xx | 客户端错误 |
| `B` | 5xx | 系统错误 |
| `C` | 5xx | 第三方服务错误 |

## 模块

| 模块 | 编号 |
|------|------|
| 通用 | `000` |
| 认证 | `001` |
| 用户 | `002` |
| 实名认证 | `003` |
| 学生认证 | `004` |
| 历史 RBAC | `005` |
| 课程 | `010` |
| 评课 | `011` |

## 响应格式

```json
{"success": false, "error": {"code": "A0010001", "message": "token expired"}}
```

## A000 通用

| 码 | HTTP | 常量 | 说明 |
|----|------|------|------|
| A0000400 | 400 | ErrBadRequest | 请求参数错误 |
| A0000401 | 400 | ErrInvalidParam | 参数格式无效 |
| A0000402 | 400 | ErrMissingParam | 缺少必填参数 |
| A0000403 | 400 | ErrParamOutOfRange | 参数超出范围 |
| A0000404 | 404 | ErrNotFound | 资源不存在 |
| A0000405 | 405 | ErrMethodNotAllowed | 方法不允许 |
| A0000409 | 409 | ErrConflict | 资源冲突 |
| A0000413 | 413 | ErrPayloadTooLarge | 请求体过大 |
| A0000415 | 415 | ErrUnsupportedMedia | 不支持的媒体类型 |
| A0000422 | 422 | ErrValidation | 验证失败 |
| A0000429 | 429 | ErrRateLimited | 频率超限 |

## A001 认证

| 码 | HTTP | 常量 | 说明 |
|----|------|------|------|
| A0010001 | 401 | ErrTokenExpired | Token 过期 |
| A0010002 | 401 | ErrTokenInvalid | Token 无效 |
| A0010003 | 401 | ErrTokenMissing | Token 缺失 |
| A0010004 | 401 | ErrTokenRevoked | Token 已撤销 |
| A0010005 | 401 | ErrRefreshTokenExpired | Refresh Token 过期 |
| A0010006 | 401 | ErrRefreshTokenInvalid | Refresh Token 无效 |
| A0010100 | 401 | ErrLoginRequired | 需要登录 |
| A0010101 | 401 | ErrLoginFailed | 登录失败 |
| A0010102 | 401 | ErrAccountDisabled | 账号禁用 |
| A0010103 | 401 | ErrAccountLocked | 账号锁定 |
| A0010200 | 403 | ErrForbidden | 权限不足 |
| A0010201 | 403 | ErrAccessDenied | 访问拒绝 |
| A0010202 | 403 | ErrCSRFTokenInvalid | CSRF Token 无效 |
| A0010203 | 403 | ErrCSRFTokenMissing | CSRF Token 缺失 |
| A0010300 | 401 | ErrOAuthFailed | OAuth 失败 |
| A0010301 | 401 | ErrOAuthStateInvalid | OAuth State 无效 |
| A0010302 | 401 | ErrOAuthCodeInvalid | 授权码无效 |
| A0010400 | 401 | ErrPhoneOTPFailed | 手机验证码失败 |
| A0010401 | 401 | ErrPhoneOTPExpired | 手机验证码过期 |
| A0010402 | 429 | ErrPhoneOTPCooldown | 验证码冷却中 |

## A002 用户

| 码 | HTTP | 常量 | 说明 |
|----|------|------|------|
| A0020001 | 404 | ErrUserNotFound | 用户不存在 |
| A0020002 | 409 | ErrUserExists | 用户已存在 |
| A0020003 | 400 | ErrUsernameTaken | 用户名占用 |
| A0020004 | 400 | ErrEmailTaken | 邮箱占用 |
| A0020005 | 400 | ErrPasswordWeak | 密码强度不足 |
| A0020006 | 400 | ErrPasswordMismatch | 密码不匹配 |

## A003 实名认证

| 码 | HTTP | 常量 | 说明 |
|----|------|------|------|
| A0030001 | 404 | ErrIdentityNotFound | 记录不存在 |
| A0030002 | 409 | ErrIdentityAlreadyExists | 已提交 |
| A0030003 | 400 | ErrIdentityDocInvalid | 证件无效 |
| A0030004 | 400 | ErrIdentityNameMismatch | 姓名不匹配 |
| A0030005 | 400 | ErrIdentityPhotoRequired | 需上传照片 |
| A0030006 | 400 | ErrIdentityVerifyFailed | 验证失败 |
| A0030007 | 409 | ErrIdentityAlreadyVerified | 已通过 |

## A004 学生认证

| 码 | HTTP | 常量 | 说明 |
|----|------|------|------|
| A0040001 | 404 | ErrProfileNotFound | 记录不存在 |
| A0040002 | 409 | ErrProfileAlreadyVerified | 已通过 |
| A0040003 | 404 | ErrProfileSchoolNotFound | 学校配置不存在 |
| A0040004 | 400 | ErrProfileSchoolDisabled | 学校通道未开启 |
| A0040005 | 400 | ErrProfileLDAPFailed | LDAP 失败 |
| A0040006 | 400 | ErrProfileConsentRequired | 需同意授权 |
| A0040007 | 409 | ErrProfilePendingReview | 审核中 |
| A0040008 | 400 | ErrProfilePhoneRequired | 需绑定手机号 |
| A0040009 | 400 | ErrProfilePhoneMismatch | 手机号需验证 |
| A0040011 | 400 | ErrProfileAcademicTable | 学籍表配置无效 |
| A0040012 | 400 | ErrAcademicTableNotConfigured | 学校未配置学籍表 |
| A0040013 | 400 | ErrSchoolLDAPConfigMissing | 学校未配置 LDAP |
| A0040014 | 400 | ErrLDAPConfigInvalid | LDAP 配置无效 |
| A0040015 | 404 | ErrSystemConfigNotFound | 系统配置不存在 |

## A005 历史 RBAC

保留兼容，不代表仍有完整本地 RBAC 模块。

| 码 | HTTP | 常量 | 说明 |
|----|------|------|------|
| A0050001 | 404 | ErrRoleNotFound | 角色不存在 |
| A0050002 | 409 | ErrRoleNameTaken | 角色名占用 |
| A0050003 | 403 | ErrRoleIsSystem | 系统角色 |
| A0050004 | 404 | ErrPermissionNotFound | 权限不存在 |
| A0050005 | 403 | ErrPermissionDenied | 权限不足 |

## A010 课程

| 码 | HTTP | 常量 | 说明 |
|----|------|------|------|
| A0100001 | 404 | ErrCourseNotFound | 课程不存在 |
| A0100002 | 404 | ErrDepartmentNotFound | 院系不存在 |
| A0100003 | 404 | ErrTeacherNotFound | 教师不存在 |
| A0100004 | 404 | ErrTermNotFound | 学期不存在 |

## A011 评课

| 码 | HTTP | 常量 | 说明 |
|----|------|------|------|
| A0110001 | 404 | ErrReviewNotFound | 评课不存在 |
| A0110002 | 409 | ErrReviewExists | 已评价 |
| A0110003 | 400 | ErrReviewContentTooShort | 内容过短 |
| A0110004 | 400 | ErrReviewContentTooLong | 内容过长 |
| A0110005 | 404 | ErrReplyNotFound | 回复不存在 |
| A0110006 | 404 | ErrDraftNotFound | 草稿不存在 |
| A0110007 | 404 | ErrReportNotFound | 举报不存在 |
| A0110008 | 404 | ErrSensitiveWordNotFound | 敏感词不存在 |
| A0110009 | 400 | ErrContentEmpty | 内容为空 |
| A0110010 | 403 | ErrNotReviewOwner | 非作者 |
| A0110011 | 403 | ErrNotReplyOwner | 非回复作者 |
| A0110100 | 409 | ErrVoteExists | 已投票 |
| A0110101 | 400 | ErrVoteTypeInvalid | 投票类型无效 |
| A0110102 | 403 | ErrVoteSelfReview | 不能给自己投票 |
| A0110103 | 409 | ErrAlreadyReported | 已举报 |
| A0110104 | 400 | ErrInvalidVoteAction | 投票动作无效 |
| A0110200 | 400 | ErrRatingInvalid | 评分无效 |
| A0110201 | 400 | ErrRatingDimensionMissing | 缺评分维度 |
| A0110300 | 400 | ErrDangerousContent | 危险内容 |
| A0110301 | 400 | ErrSensitiveContent | 敏感内容 |
| A0110302 | 400 | ErrInvalidTransition | 状态流转无效 |

## B000 系统

| 码 | HTTP | 常量 | 说明 |
|----|------|------|------|
| B0000001 | 500 | ErrInternal | 内部错误 |
| B0000002 | 500 | ErrDatabaseError | 数据库错误 |
| B0000003 | 500 | ErrCacheError | 缓存错误 |
| B0000004 | 503 | ErrServiceUnavailable | 服务不可用 |
| B0000005 | 503 | ErrServiceOverloaded | 服务过载 |
| B0000006 | 504 | ErrTimeout | 超时 |
| B0000007 | 500 | ErrConfigError | 配置错误 |

## C000 第三方

| 码 | HTTP | 常量 | 说明 |
|----|------|------|------|
| C0000001 | 502 | ErrUpstreamError | 上游错误 |
| C0000002 | 504 | ErrUpstreamTimeout | 上游超时 |
| C0000003 | 503 | ErrUpstreamUnavailable | 上游不可用 |
| C0010001 | 502 | ErrSSOError | SSO 错误 |
| C0010002 | 504 | ErrSSOTimeout | SSO 超时 |
| C0010003 | 503 | ErrSSOUnavailable | SSO 不可用 |

## 前端处理

前端读 `error.code`，按前缀做鉴权跳转、重试和本地化映射。网络异常生成 `NETWORK_ERROR` / `OFFLINE` / `TIMEOUT`。
