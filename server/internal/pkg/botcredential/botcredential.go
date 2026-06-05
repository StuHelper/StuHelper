package botcredential

import "errors"

const (
	KoishiRuntimeCredentialName   = "koishi-runtime"
	AudienceBotAPI                = "/api/v1/bot/*"
	ScopeBotQQBindingConsume      = "bot.qq_binding.consume"
	ScopeBotQQVerificationRead    = "bot.qq_verification.read"
	ScopeBotAdmissionSession      = "bot.admission.session"
	ScopeBotAdmissionEvent        = "bot.admission.event"
	ScopeBotAdmissionReview       = "bot.admission.review"
	ScopeBotAdmissionForward      = "bot.admission.forward"
	ScopeBotMemberBlacklistRead   = "bot.member_blacklist.read"
	ScopeBotMemberBlacklistManage = "bot.member_blacklist.manage"
)

var (
	ErrCredentialNotConfigured    = errors.New("service account credential is not configured")
	ErrCredentialInvalid          = errors.New("service account credential is invalid")
	ErrCredentialForbidden        = errors.New("service account credential lacks required audience or scope")
	ErrCredentialStoreUnavailable = errors.New("service account credential store unavailable")
)

func KoishiRuntimeScopes() []string {
	return []string{
		ScopeBotQQBindingConsume,
		ScopeBotQQVerificationRead,
		ScopeBotAdmissionSession,
		ScopeBotAdmissionEvent,
		ScopeBotAdmissionReview,
		ScopeBotAdmissionForward,
		ScopeBotMemberBlacklistRead,
		ScopeBotMemberBlacklistManage,
	}
}
