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
// 详细说明见 docs/references/error-codes.md
package errs

// ErrorCode 错误码类型，提供编译期拼写检查
type ErrorCode string

// String 返回错误码字符串
func (c ErrorCode) String() string { return string(c) }

// ============================================================================
// A000xxxx - 通用客户端错误
// 序号使用 HTTP 状态码便于记忆
// ============================================================================

const (
	ErrBadRequest       ErrorCode = "A0000400" // 请求参数错误
	ErrInvalidParam     ErrorCode = "A0000401" // 参数格式无效
	ErrMissingParam     ErrorCode = "A0000402" // 缺少必填参数
	ErrParamOutOfRange  ErrorCode = "A0000403" // 参数超出范围
	ErrNotFound         ErrorCode = "A0000404" // 资源不存在
	ErrMethodNotAllowed ErrorCode = "A0000405" // 请求方法不允许
	ErrConflict         ErrorCode = "A0000409" // 资源冲突
	ErrPayloadTooLarge  ErrorCode = "A0000413" // 请求体过大
	ErrUnsupportedMedia ErrorCode = "A0000415" // 不支持的媒体类型
	ErrValidation       ErrorCode = "A0000422" // 数据验证失败
	ErrRateLimited      ErrorCode = "A0000429" // 请求频率超限
)

// ============================================================================
// A001xxxx - 认证与授权错误
// 分组: 0001-0099 Token | 0100-0199 登录 | 0200-0299 权限 | 0300-0399 OAuth
// ============================================================================

const (
	// Token 相关 (0001-0099)
	ErrTokenExpired        ErrorCode = "A0010001" // Token 已过期
	ErrTokenInvalid        ErrorCode = "A0010002" // Token 无效
	ErrTokenMissing        ErrorCode = "A0010003" // 未提供 Token
	ErrTokenRevoked        ErrorCode = "A0010004" // Token 已被撤销
	ErrRefreshTokenExpired ErrorCode = "A0010005" // Refresh Token 已过期
	ErrRefreshTokenInvalid ErrorCode = "A0010006" // Refresh Token 无效

	// 登录相关 (0100-0199)
	ErrLoginRequired   ErrorCode = "A0010100" // 请先登录
	ErrLoginFailed     ErrorCode = "A0010101" // 登录失败
	ErrAccountDisabled ErrorCode = "A0010102" // 账号已禁用
	ErrAccountLocked   ErrorCode = "A0010103" // 账号已锁定

	// 权限相关 (0200-0299)
	ErrForbidden        ErrorCode = "A0010200" // 权限不足
	ErrAccessDenied     ErrorCode = "A0010201" // 访问被拒绝
	ErrCSRFTokenInvalid ErrorCode = "A0010202" // CSRF Token 无效
	ErrCSRFTokenMissing ErrorCode = "A0010203" // CSRF Token 缺失

	// OAuth 相关 (0300-0399)
	ErrOAuthFailed       ErrorCode = "A0010300" // OAuth 认证失败
	ErrOAuthStateInvalid ErrorCode = "A0010301" // OAuth State 无效
	ErrOAuthCodeInvalid  ErrorCode = "A0010302" // OAuth 授权码无效
	ErrPhoneOTPFailed    ErrorCode = "A0010400" // 手机验证码验证失败
	ErrPhoneOTPExpired   ErrorCode = "A0010401" // 手机验证码已过期
	ErrPhoneOTPCooldown  ErrorCode = "A0010402" // 手机验证码发送冷却中
)

// ============================================================================
// A002xxxx - 用户相关错误
// ============================================================================

const (
	ErrUserNotFound     ErrorCode = "A0020001" // 用户不存在
	ErrUserExists       ErrorCode = "A0020002" // 用户已存在
	ErrUsernameTaken    ErrorCode = "A0020003" // 用户名已被占用
	ErrEmailTaken       ErrorCode = "A0020004" // 邮箱已被占用
	ErrPasswordWeak     ErrorCode = "A0020005" // 密码强度不足
	ErrPasswordMismatch ErrorCode = "A0020006" // 密码不匹配
)

// ============================================================================
// A003xxxx - 实名认证相关错误
// ============================================================================

const (
	ErrIdentityNotFound        ErrorCode = "A0030001" // 实名认证记录不存在
	ErrIdentityAlreadyExists   ErrorCode = "A0030002" // 已提交实名认证
	ErrIdentityDocInvalid      ErrorCode = "A0030003" // 证件信息无效
	ErrIdentityNameMismatch    ErrorCode = "A0030004" // 姓名与证件不匹配
	ErrIdentityPhotoRequired   ErrorCode = "A0030005" // 需要上传证件照片
	ErrIdentityVerifyFailed    ErrorCode = "A0030006" // 实名认证验证失败
	ErrIdentityAlreadyVerified ErrorCode = "A0030007" // 已通过实名认证
)

// ============================================================================
// A004xxxx - 学生认证相关错误
// ============================================================================

const (
	ErrProfileNotFound            ErrorCode = "A0040001" // 学生认证记录不存在
	ErrProfileAlreadyVerified     ErrorCode = "A0040002" // 已通过学生认证
	ErrProfileSchoolNotFound      ErrorCode = "A0040003" // 学校配置不存在
	ErrProfileSchoolDisabled      ErrorCode = "A0040004" // 学校认证通道未开启
	ErrProfileLDAPFailed          ErrorCode = "A0040005" // LDAP 认证失败
	ErrProfileConsentRequired     ErrorCode = "A0040006" // 需要同意数据使用授权
	ErrProfilePendingReview       ErrorCode = "A0040007" // 认证正在审核中
	ErrProfilePhoneRequired       ErrorCode = "A0040008" // 需要绑定手机号
	ErrProfilePhoneMismatch       ErrorCode = "A0040009" // 手机号需要验证
	ErrProfileAcademicTable       ErrorCode = "A0040011" // 学籍表配置无效
	ErrAcademicTableNotConfigured ErrorCode = "A0040012" // 学校未配置学籍表
	ErrSchoolLDAPConfigMissing    ErrorCode = "A0040013" // 学校未配置 LDAP 连接
	ErrLDAPConfigInvalid          ErrorCode = "A0040014" // 学校 LDAP 配置无效
	ErrSystemConfigNotFound       ErrorCode = "A0040015" // 系统配置不存在
)

// ============================================================================
// A005xxxx - RBAC 权限相关错误
// ============================================================================

const (
	ErrRoleNotFound               ErrorCode = "A0050001" // 角色不存在
	ErrRoleNameTaken              ErrorCode = "A0050002" // 角色名已存在
	ErrRoleIsSystem               ErrorCode = "A0050003" // 系统角色不可修改
	ErrPermissionNotFound         ErrorCode = "A0050004" // 权限不存在
	ErrPermissionDenied           ErrorCode = "A0050005" // 权限不足（RBAC 检查失败）
	ErrGroupNotFound              ErrorCode = "A0050006" // 用户组不存在
	ErrGroupNameTaken             ErrorCode = "A0050007" // 用户组名已存在
	ErrPermissionSelectionInvalid ErrorCode = "A0050008" // 权限选择无效
	ErrRolePermissionClearConfirm ErrorCode = "A0050009" // 清空角色权限需显式确认
	ErrRoleSelectionInvalid       ErrorCode = "A0050010" // 角色选择无效
	ErrUserSelectionInvalid       ErrorCode = "A0050011" // 用户选择无效
)

// ============================================================================
// A010xxxx - 课程模块错误
// ============================================================================

const (
	ErrCourseNotFound     ErrorCode = "A0100001" // 课程不存在
	ErrDepartmentNotFound ErrorCode = "A0100002" // 院系不存在
	ErrTeacherNotFound    ErrorCode = "A0100003" // 教师不存在
	ErrTermNotFound       ErrorCode = "A0100004" // 学期不存在
)

// ============================================================================
// A011xxxx - 评课模块错误
// 分组: 0001-0099 测评基础 | 0100-0199 投票 | 0200-0299 评分 | 0300-0399 内容审核
// ============================================================================

const (
	// 测评基础 (0001-0099)
	ErrReviewNotFound        ErrorCode = "A0110001" // 测评不存在
	ErrReviewExists          ErrorCode = "A0110002" // 已评价过该课程
	ErrReviewContentTooShort ErrorCode = "A0110003" // 测评内容过短
	ErrReviewContentTooLong  ErrorCode = "A0110004" // 测评内容过长
	ErrReplyNotFound         ErrorCode = "A0110005" // 回复不存在
	ErrDraftNotFound         ErrorCode = "A0110006" // 草稿不存在
	ErrReportNotFound        ErrorCode = "A0110007" // 举报记录不存在
	ErrSensitiveWordNotFound ErrorCode = "A0110008" // 敏感词不存在
	ErrContentEmpty          ErrorCode = "A0110009" // 内容为空
	ErrNotReviewOwner        ErrorCode = "A0110010" // 非测评所有者
	ErrNotReplyOwner         ErrorCode = "A0110011" // 非回复所有者

	// 投票相关 (0100-0199)
	ErrVoteExists        ErrorCode = "A0110100" // 已投票过该测评
	ErrVoteTypeInvalid   ErrorCode = "A0110101" // 投票类型无效
	ErrVoteSelfReview    ErrorCode = "A0110102" // 不能给自己的测评投票
	ErrAlreadyReported   ErrorCode = "A0110103" // 已举报过
	ErrInvalidVoteAction ErrorCode = "A0110104" // 无效投票操作

	// 评分相关 (0200-0299)
	ErrRatingInvalid          ErrorCode = "A0110200" // 评分无效
	ErrRatingDimensionMissing ErrorCode = "A0110201" // 缺少必填评分维度

	// 内容审核 (0300-0399)
	ErrDangerousContent  ErrorCode = "A0110300" // 内容包含危险元素
	ErrSensitiveContent  ErrorCode = "A0110301" // 内容包含敏感词
	ErrInvalidTransition ErrorCode = "A0110302" // 无效状态转换
)

// ============================================================================
// B000xxxx - 系统通用错误
// ============================================================================

const (
	ErrInternal           ErrorCode = "B0000001" // 服务器内部错误
	ErrDatabaseError      ErrorCode = "B0000002" // 数据库错误
	ErrCacheError         ErrorCode = "B0000003" // 缓存错误
	ErrServiceUnavailable ErrorCode = "B0000004" // 服务暂时不可用
	ErrServiceOverloaded  ErrorCode = "B0000005" // 服务过载
	ErrTimeout            ErrorCode = "B0000006" // 请求超时
	ErrConfigError        ErrorCode = "B0000007" // 配置错误
)

// ============================================================================
// C000xxxx - 第三方服务通用错误
// ============================================================================

const (
	ErrUpstreamError       ErrorCode = "C0000001" // 上游服务错误
	ErrUpstreamTimeout     ErrorCode = "C0000002" // 上游服务超时
	ErrUpstreamUnavailable ErrorCode = "C0000003" // 上游服务不可用
)

// ============================================================================
// C001xxxx - SSO 服务错误
// ============================================================================

const (
	ErrSSOError       ErrorCode = "C0010001" // SSO 服务错误
	ErrSSOTimeout     ErrorCode = "C0010002" // SSO 服务超时
	ErrSSOUnavailable ErrorCode = "C0010003" // SSO 服务不可用
)
