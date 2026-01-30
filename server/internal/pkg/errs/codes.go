// Package errs 定义企业级错误码
//
// 错误码格式: {类别1位}{模块3位}{序号4位} = 8位
//   - 类别: A(客户端错误) B(系统错误) C(第三方服务错误)
//   - 模块: 000(通用) 001(认证) 002(用户) 010(课程) 011(评课) 020(文件) 030(通知)
//   - 序号: 0001-9999
//
// 模块编号规划:
//   - 000-009: 基础模块 (通用、认证、用户等)
//   - 010-019: 教学模块 (课程、评课等)
//   - 020-029: 资源模块 (文件等)
//   - 030-039: 通信模块 (通知等)
//   - 040-099: 业务扩展预留
//   - 100-999: 未来扩展预留
//
// 详细说明见 docs/api/error-codes.md
package errs

// ============================================================================
// A000xxxx - 通用客户端错误
// 序号使用 HTTP 状态码便于记忆
// ============================================================================

const (
	ErrBadRequest       = "A0000400" // 请求参数错误
	ErrInvalidParam     = "A0000401" // 参数格式无效
	ErrMissingParam     = "A0000402" // 缺少必填参数
	ErrParamOutOfRange  = "A0000403" // 参数超出范围
	ErrNotFound         = "A0000404" // 资源不存在
	ErrMethodNotAllowed = "A0000405" // 请求方法不允许
	ErrConflict         = "A0000409" // 资源冲突
	ErrPayloadTooLarge  = "A0000413" // 请求体过大
	ErrUnsupportedMedia = "A0000415" // 不支持的媒体类型
	ErrValidation       = "A0000422" // 数据验证失败
	ErrRateLimited      = "A0000429" // 请求频率超限
)

// ============================================================================
// A001xxxx - 认证与授权错误
// 分组: 0001-0099 Token | 0100-0199 登录 | 0200-0299 权限 | 0300-0399 OAuth
// ============================================================================

const (
	// Token 相关 (0001-0099)
	ErrTokenExpired        = "A0010001" // Token 已过期
	ErrTokenInvalid        = "A0010002" // Token 无效
	ErrTokenMissing        = "A0010003" // 未提供 Token
	ErrTokenRevoked        = "A0010004" // Token 已被撤销
	ErrRefreshTokenExpired = "A0010005" // Refresh Token 已过期
	ErrRefreshTokenInvalid = "A0010006" // Refresh Token 无效

	// 登录相关 (0100-0199)
	ErrLoginRequired   = "A0010100" // 请先登录
	ErrLoginFailed     = "A0010101" // 登录失败
	ErrAccountDisabled = "A0010102" // 账号已禁用
	ErrAccountLocked   = "A0010103" // 账号已锁定

	// 权限相关 (0200-0299)
	ErrForbidden        = "A0010200" // 权限不足
	ErrAccessDenied     = "A0010201" // 访问被拒绝
	ErrCSRFTokenInvalid = "A0010202" // CSRF Token 无效

	// OAuth 相关 (0300-0399)
	ErrOAuthFailed       = "A0010300" // OAuth 认证失败
	ErrOAuthStateInvalid = "A0010301" // OAuth State 无效
	ErrOAuthCodeInvalid  = "A0010302" // OAuth 授权码无效
)

// ============================================================================
// A002xxxx - 用户相关错误
// ============================================================================

const (
	ErrUserNotFound     = "A0020001" // 用户不存在
	ErrUserExists       = "A0020002" // 用户已存在
	ErrUsernameTaken    = "A0020003" // 用户名已被占用
	ErrEmailTaken       = "A0020004" // 邮箱已被占用
	ErrPasswordWeak     = "A0020005" // 密码强度不足
	ErrPasswordMismatch = "A0020006" // 密码不匹配
)

// ============================================================================
// A010xxxx - 课程模块错误
// ============================================================================

const (
	ErrCourseNotFound     = "A0100001" // 课程不存在
	ErrDepartmentNotFound = "A0100002" // 院系不存在
	ErrTeacherNotFound    = "A0100003" // 教师不存在
	ErrTermNotFound       = "A0100004" // 学期不存在
)

// ============================================================================
// A011xxxx - 评课模块错误
// 分组: 0001-0099 测评基础 | 0100-0199 投票 | 0200-0299 评分 | 0300-0399 内容审核
// ============================================================================

const (
	// 测评基础 (0001-0099)
	ErrReviewNotFound        = "A0110001" // 测评不存在
	ErrReviewExists          = "A0110002" // 已评价过该课程
	ErrReviewContentTooShort = "A0110003" // 测评内容过短
	ErrReviewContentTooLong  = "A0110004" // 测评内容过长

	// 投票相关 (0100-0199)
	ErrVoteExists      = "A0110100" // 已投票过该测评
	ErrVoteTypeInvalid = "A0110101" // 投票类型无效
	ErrVoteSelfReview  = "A0110102" // 不能给自己的测评投票

	// 评分相关 (0200-0299)
	ErrRatingInvalid          = "A0110200" // 评分无效
	ErrRatingDimensionMissing = "A0110201" // 缺少必填评分维度

	// 内容审核 (0300-0399)
	ErrDangerousContent = "A0110300" // 内容包含危险元素
)

// ============================================================================
// B000xxxx - 系统通用错误
// ============================================================================

const (
	ErrInternal           = "B0000001" // 服务器内部错误
	ErrDatabaseError      = "B0000002" // 数据库错误
	ErrCacheError         = "B0000003" // 缓存错误
	ErrServiceUnavailable = "B0000004" // 服务暂时不可用
	ErrServiceOverloaded  = "B0000005" // 服务过载
	ErrTimeout            = "B0000006" // 请求超时
	ErrConfigError        = "B0000007" // 配置错误
)

// ============================================================================
// C000xxxx - 第三方服务通用错误
// ============================================================================

const (
	ErrUpstreamError       = "C0000001" // 上游服务错误
	ErrUpstreamTimeout     = "C0000002" // 上游服务超时
	ErrUpstreamUnavailable = "C0000003" // 上游服务不可用
)

// ============================================================================
// C001xxxx - SSO 服务错误
// ============================================================================

const (
	ErrSSOError       = "C0010001" // SSO 服务错误
	ErrSSOTimeout     = "C0010002" // SSO 服务超时
	ErrSSOUnavailable = "C0010003" // SSO 服务不可用
)
