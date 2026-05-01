package errs

import (
	"reflect"
	"strings"
	"testing"
)

// allErrorCodes 返回当前包中定义的所有 ErrorCode 常量。
// 使用显式列表而非反射遍历以便测试失败时定位到具体常量。
func allErrorCodes() map[string]ErrorCode {
	return map[string]ErrorCode{
		// A000xxxx - 通用客户端错误
		"ErrBadRequest":       ErrBadRequest,
		"ErrInvalidParam":     ErrInvalidParam,
		"ErrMissingParam":     ErrMissingParam,
		"ErrParamOutOfRange":  ErrParamOutOfRange,
		"ErrNotFound":         ErrNotFound,
		"ErrMethodNotAllowed": ErrMethodNotAllowed,
		"ErrConflict":         ErrConflict,
		"ErrPayloadTooLarge":  ErrPayloadTooLarge,
		"ErrUnsupportedMedia": ErrUnsupportedMedia,
		"ErrValidation":       ErrValidation,
		"ErrRateLimited":      ErrRateLimited,

		// A001xxxx - 认证与授权
		"ErrTokenExpired":        ErrTokenExpired,
		"ErrTokenInvalid":        ErrTokenInvalid,
		"ErrTokenMissing":        ErrTokenMissing,
		"ErrTokenRevoked":        ErrTokenRevoked,
		"ErrRefreshTokenExpired": ErrRefreshTokenExpired,
		"ErrRefreshTokenInvalid": ErrRefreshTokenInvalid,
		"ErrLoginRequired":       ErrLoginRequired,
		"ErrLoginFailed":         ErrLoginFailed,
		"ErrAccountDisabled":     ErrAccountDisabled,
		"ErrAccountLocked":       ErrAccountLocked,
		"ErrForbidden":           ErrForbidden,
		"ErrAccessDenied":        ErrAccessDenied,
		"ErrCSRFTokenInvalid":    ErrCSRFTokenInvalid,
		"ErrCSRFTokenMissing":    ErrCSRFTokenMissing,
		"ErrMFARequired":         ErrMFARequired,
		"ErrStepUpRequired":      ErrStepUpRequired,
		"ErrOAuthFailed":         ErrOAuthFailed,
		"ErrOAuthStateInvalid":   ErrOAuthStateInvalid,
		"ErrOAuthCodeInvalid":    ErrOAuthCodeInvalid,
		"ErrPhoneOTPFailed":      ErrPhoneOTPFailed,
		"ErrPhoneOTPExpired":     ErrPhoneOTPExpired,
		"ErrPhoneOTPCooldown":    ErrPhoneOTPCooldown,

		// A002xxxx - 用户
		"ErrUserNotFound":     ErrUserNotFound,
		"ErrUserExists":       ErrUserExists,
		"ErrUsernameTaken":    ErrUsernameTaken,
		"ErrEmailTaken":       ErrEmailTaken,
		"ErrPasswordWeak":     ErrPasswordWeak,
		"ErrPasswordMismatch": ErrPasswordMismatch,

		// A003xxxx - 实名认证
		"ErrIdentityNotFound":        ErrIdentityNotFound,
		"ErrIdentityAlreadyExists":   ErrIdentityAlreadyExists,
		"ErrIdentityDocInvalid":      ErrIdentityDocInvalid,
		"ErrIdentityNameMismatch":    ErrIdentityNameMismatch,
		"ErrIdentityPhotoRequired":   ErrIdentityPhotoRequired,
		"ErrIdentityVerifyFailed":    ErrIdentityVerifyFailed,
		"ErrIdentityAlreadyVerified": ErrIdentityAlreadyVerified,

		// A004xxxx - 学生认证
		"ErrProfileNotFound":            ErrProfileNotFound,
		"ErrProfileAlreadyVerified":     ErrProfileAlreadyVerified,
		"ErrProfileSchoolNotFound":      ErrProfileSchoolNotFound,
		"ErrProfileSchoolDisabled":      ErrProfileSchoolDisabled,
		"ErrProfileLDAPFailed":          ErrProfileLDAPFailed,
		"ErrProfileConsentRequired":     ErrProfileConsentRequired,
		"ErrProfilePendingReview":       ErrProfilePendingReview,
		"ErrProfilePhoneRequired":       ErrProfilePhoneRequired,
		"ErrProfilePhoneMismatch":       ErrProfilePhoneMismatch,
		"ErrProfileAcademicTable":       ErrProfileAcademicTable,
		"ErrAcademicTableNotConfigured": ErrAcademicTableNotConfigured,
		"ErrSchoolLDAPConfigMissing":    ErrSchoolLDAPConfigMissing,
		"ErrLDAPConfigInvalid":          ErrLDAPConfigInvalid,
		"ErrSystemConfigNotFound":       ErrSystemConfigNotFound,

		// A005xxxx - RBAC
		"ErrRoleNotFound":               ErrRoleNotFound,
		"ErrRoleNameTaken":              ErrRoleNameTaken,
		"ErrRoleIsSystem":               ErrRoleIsSystem,
		"ErrPermissionNotFound":         ErrPermissionNotFound,
		"ErrPermissionDenied":           ErrPermissionDenied,
		"ErrGroupNotFound":              ErrGroupNotFound,
		"ErrGroupNameTaken":             ErrGroupNameTaken,
		"ErrPermissionSelectionInvalid": ErrPermissionSelectionInvalid,
		"ErrRolePermissionClearConfirm": ErrRolePermissionClearConfirm,
		"ErrRoleSelectionInvalid":       ErrRoleSelectionInvalid,
		"ErrUserSelectionInvalid":       ErrUserSelectionInvalid,

		// A010xxxx - 课程
		"ErrCourseNotFound":     ErrCourseNotFound,
		"ErrDepartmentNotFound": ErrDepartmentNotFound,
		"ErrTeacherNotFound":    ErrTeacherNotFound,
		"ErrTermNotFound":       ErrTermNotFound,

		// A011xxxx - 评课
		"ErrReviewNotFound":         ErrReviewNotFound,
		"ErrReviewExists":           ErrReviewExists,
		"ErrReviewContentTooShort":  ErrReviewContentTooShort,
		"ErrReviewContentTooLong":   ErrReviewContentTooLong,
		"ErrReplyNotFound":          ErrReplyNotFound,
		"ErrDraftNotFound":          ErrDraftNotFound,
		"ErrReportNotFound":         ErrReportNotFound,
		"ErrSensitiveWordNotFound":  ErrSensitiveWordNotFound,
		"ErrContentEmpty":           ErrContentEmpty,
		"ErrNotReviewOwner":         ErrNotReviewOwner,
		"ErrNotReplyOwner":          ErrNotReplyOwner,
		"ErrVoteExists":             ErrVoteExists,
		"ErrVoteTypeInvalid":        ErrVoteTypeInvalid,
		"ErrVoteSelfReview":         ErrVoteSelfReview,
		"ErrAlreadyReported":        ErrAlreadyReported,
		"ErrInvalidVoteAction":      ErrInvalidVoteAction,
		"ErrRatingInvalid":          ErrRatingInvalid,
		"ErrRatingDimensionMissing": ErrRatingDimensionMissing,
		"ErrDangerousContent":       ErrDangerousContent,
		"ErrSensitiveContent":       ErrSensitiveContent,
		"ErrInvalidTransition":      ErrInvalidTransition,

		// B000xxxx - 系统
		"ErrInternal":           ErrInternal,
		"ErrDatabaseError":      ErrDatabaseError,
		"ErrCacheError":         ErrCacheError,
		"ErrServiceUnavailable": ErrServiceUnavailable,
		"ErrServiceOverloaded":  ErrServiceOverloaded,
		"ErrTimeout":            ErrTimeout,
		"ErrConfigError":        ErrConfigError,

		// C000xxxx - 第三方通用
		"ErrUpstreamError":       ErrUpstreamError,
		"ErrUpstreamTimeout":     ErrUpstreamTimeout,
		"ErrUpstreamUnavailable": ErrUpstreamUnavailable,

		// C001xxxx - SSO
		"ErrSSOError":       ErrSSOError,
		"ErrSSOTimeout":     ErrSSOTimeout,
		"ErrSSOUnavailable": ErrSSOUnavailable,
	}
}

func TestErrorCode_String(t *testing.T) {
	tests := []struct {
		name string
		code ErrorCode
		want string
	}{
		{"通用 400", ErrBadRequest, "A0000400"},
		{"Token 过期", ErrTokenExpired, "A0010001"},
		{"用户不存在", ErrUserNotFound, "A0020001"},
		{"服务器内部错误", ErrInternal, "B0000001"},
		{"SSO 服务错误", ErrSSOError, "C0010001"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.code.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
			// String() 返回值应与 string(code) 一致
			if got != string(tt.code) {
				t.Errorf("String() = %q, expected identical to string(code) = %q", got, string(tt.code))
			}
		})
	}
}

func TestErrorCode_StringMethodSignature(t *testing.T) {
	// 验证 ErrorCode 类型的 String() 方法确实存在且签名正确
	var code ErrorCode = "X9999999"
	result := code.String()
	if result != "X9999999" {
		t.Errorf("String() = %q, want %q", result, "X9999999")
	}
	// 验证底层类型
	if reflect.TypeOf(code).Kind() != reflect.String {
		t.Errorf("ErrorCode underlying kind = %v, want %v", reflect.TypeOf(code).Kind(), reflect.String)
	}
}

func TestErrorCodes_AllUnique(t *testing.T) {
	codes := allErrorCodes()
	seen := make(map[ErrorCode]string, len(codes))

	for name, code := range codes {
		if existing, dup := seen[code]; dup {
			t.Errorf("duplicate error code %q used by both %s and %s", code, existing, name)
			continue
		}
		seen[code] = name
	}

	if len(seen) != len(codes) {
		t.Errorf("expected %d unique codes, got %d", len(codes), len(seen))
	}
}

func TestErrorCodes_FormatCompliance(t *testing.T) {
	// 按包注释约定：8 位长度，首字符为 A/B/C，其余为数字
	validCategories := map[byte]string{
		'A': "客户端错误",
		'B': "系统错误",
		'C': "第三方服务错误",
	}

	for name, code := range allErrorCodes() {
		t.Run(name, func(t *testing.T) {
			s := code.String()
			if len(s) != 8 {
				t.Errorf("%s = %q: length = %d, want 8", name, s, len(s))
				return
			}
			category := s[0]
			if _, ok := validCategories[category]; !ok {
				t.Errorf("%s = %q: invalid category prefix %q, want one of A/B/C", name, s, string(category))
			}
			for i := 1; i < 8; i++ {
				if s[i] < '0' || s[i] > '9' {
					t.Errorf("%s = %q: char at index %d is %q, want digit", name, s, i, string(s[i]))
				}
			}
		})
	}
}

func TestErrorCodes_CategoryPrefixes(t *testing.T) {
	// 抽样验证各类别前缀分配正确
	tests := []struct {
		name         string
		code         ErrorCode
		wantCategory byte
	}{
		{"ErrBadRequest 为客户端错误", ErrBadRequest, 'A'},
		{"ErrValidation 为客户端错误", ErrValidation, 'A'},
		{"ErrUserNotFound 为客户端错误", ErrUserNotFound, 'A'},
		{"ErrInternal 为系统错误", ErrInternal, 'B'},
		{"ErrDatabaseError 为系统错误", ErrDatabaseError, 'B'},
		{"ErrUpstreamError 为第三方错误", ErrUpstreamError, 'C'},
		{"ErrSSOError 为第三方错误", ErrSSOError, 'C'},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.code.String()
			if s[0] != tt.wantCategory {
				t.Errorf("code %q category = %q, want %q", s, string(s[0]), string(tt.wantCategory))
			}
		})
	}
}

func TestErrorCode_Equality(t *testing.T) {
	// ErrorCode 是 string 的类型别名，支持值比较
	a := ErrBadRequest
	b := ErrBadRequest
	c := ErrNotFound

	if a != b {
		t.Errorf("expected ErrBadRequest == ErrBadRequest")
	}
	if a == c {
		t.Errorf("expected ErrBadRequest != ErrNotFound")
	}

	// 与字面量比较
	if ErrTokenExpired != ErrorCode("A0010001") {
		t.Errorf("ErrTokenExpired (%q) should equal ErrorCode(\"A0010001\")", ErrTokenExpired)
	}
}

func TestErrorCode_ModulePrefixGrouping(t *testing.T) {
	// 按包注释约定：第 2-4 位为模块号。验证各模块的代码归类在同一前缀段。
	tests := []struct {
		name       string
		code       ErrorCode
		wantPrefix string // 前 4 位 (类别+模块)
	}{
		{"通用客户端错误 A000", ErrBadRequest, "A000"},
		{"认证模块 A001", ErrTokenExpired, "A001"},
		{"用户模块 A002", ErrUserNotFound, "A002"},
		{"实名认证 A003", ErrIdentityNotFound, "A003"},
		{"学生认证 A004", ErrProfileNotFound, "A004"},
		{"RBAC A005", ErrRoleNotFound, "A005"},
		{"课程 A010", ErrCourseNotFound, "A010"},
		{"评课 A011", ErrReviewNotFound, "A011"},
		{"系统通用 B000", ErrInternal, "B000"},
		{"第三方通用 C000", ErrUpstreamError, "C000"},
		{"SSO C001", ErrSSOError, "C001"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.code.String()
			if !strings.HasPrefix(s, tt.wantPrefix) {
				t.Errorf("code %q: want prefix %q", s, tt.wantPrefix)
			}
		})
	}
}
