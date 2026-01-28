// Package errs 定义企业级错误码
// 错误码格式: {类别}{模块}{序号}
// - 类别: A(客户端错误) B(系统错误) C(第三方服务错误)
// - 模块: 00(通用) 01(认证) 02(用户) 10(课程) 11(评课) 20(文件) 30(通知)
// - 序号: 001-999
//
// 详细说明见 docs/api/error-codes.md
package errs

// ============================================================================
// A00xxx - 通用客户端错误
// ============================================================================

const (
	ErrBadRequest       = "A00400" // 请求参数错误
	ErrInvalidParam     = "A00401" // 参数格式无效
	ErrMissingParam     = "A00402" // 缺少必填参数
	ErrParamOutOfRange  = "A00403" // 参数超出范围
	ErrNotFound         = "A00404" // 资源不存在
	ErrMethodNotAllowed = "A00405" // 请求方法不允许
	ErrConflict         = "A00409" // 资源冲突
	ErrPayloadTooLarge  = "A00413" // 请求体过大
	ErrUnsupportedMedia = "A00415" // 不支持的媒体类型
	ErrValidation       = "A00422" // 数据验证失败
	ErrRateLimited      = "A00429" // 请求频率超限
)

// ============================================================================
// A01xxx - 认证与授权错误
// ============================================================================

const (
	ErrTokenExpired        = "A01001" // Token 已过期
	ErrTokenInvalid        = "A01002" // Token 无效
	ErrTokenMissing        = "A01003" // 未提供 Token
	ErrTokenRevoked        = "A01004" // Token 已被撤销
	ErrRefreshTokenExpired = "A01005" // Refresh Token 已过期
	ErrRefreshTokenInvalid = "A01006" // Refresh Token 无效
	ErrLoginRequired       = "A01010" // 请先登录
	ErrLoginFailed         = "A01011" // 登录失败
	ErrAccountDisabled     = "A01012" // 账号已禁用
	ErrAccountLocked       = "A01013" // 账号已锁定
	ErrForbidden           = "A01020" // 权限不足
	ErrAccessDenied        = "A01021" // 访问被拒绝
	ErrCSRFTokenInvalid    = "A01022" // CSRF Token 无效
	ErrOAuthFailed         = "A01030" // OAuth 认证失败
	ErrOAuthStateInvalid   = "A01031" // OAuth State 无效
	ErrOAuthCodeInvalid    = "A01032" // OAuth 授权码无效
)

// ============================================================================
// A02xxx - 用户相关错误
// ============================================================================

const (
	ErrUserNotFound     = "A02001" // 用户不存在
	ErrUserExists       = "A02002" // 用户已存在
	ErrUsernameTaken    = "A02003" // 用户名已被占用
	ErrEmailTaken       = "A02004" // 邮箱已被占用
	ErrPasswordWeak     = "A02005" // 密码强度不足
	ErrPasswordMismatch = "A02006" // 密码不匹配
)

// ============================================================================
// A10xxx - 课程模块错误
// ============================================================================

const (
	ErrCourseNotFound     = "A10001" // 课程不存在
	ErrDepartmentNotFound = "A10002" // 院系不存在
	ErrTeacherNotFound    = "A10003" // 教师不存在
	ErrTermNotFound       = "A10004" // 学期不存在
)

// ============================================================================
// A11xxx - 评课模块错误
// ============================================================================

const (
	ErrReviewNotFound         = "A11001" // 测评不存在
	ErrReviewExists           = "A11002" // 已评价过该课程
	ErrReviewContentTooShort  = "A11003" // 测评内容过短
	ErrReviewContentTooLong   = "A11004" // 测评内容过长
	ErrRatingInvalid          = "A11005" // 评分无效
	ErrRatingDimensionMissing = "A11006" // 缺少必填评分维度
	ErrVoteExists             = "A11010" // 已投票过该测评
	ErrVoteTypeInvalid        = "A11011" // 投票类型无效
	ErrVoteSelfReview         = "A11012" // 不能给自己的测评投票
)

// ============================================================================
// B00xxx - 系统通用错误
// ============================================================================

const (
	ErrInternal           = "B00001" // 服务器内部错误
	ErrDatabaseError      = "B00002" // 数据库错误
	ErrCacheError         = "B00003" // 缓存错误
	ErrServiceUnavailable = "B00004" // 服务暂时不可用
	ErrServiceOverloaded  = "B00005" // 服务过载
	ErrTimeout            = "B00006" // 请求超时
	ErrConfigError        = "B00007" // 配置错误
)

// ============================================================================
// C00xxx - 第三方服务错误
// ============================================================================

const (
	ErrUpstreamError       = "C00001" // 上游服务错误
	ErrUpstreamTimeout     = "C00002" // 上游服务超时
	ErrUpstreamUnavailable = "C00003" // 上游服务不可用
)

// ============================================================================
// C01xxx - SSO 服务错误
// ============================================================================

const (
	ErrSSOError       = "C01001" // SSO 服务错误
	ErrSSOTimeout     = "C01002" // SSO 服务超时
	ErrSSOUnavailable = "C01003" // SSO 服务不可用
)
