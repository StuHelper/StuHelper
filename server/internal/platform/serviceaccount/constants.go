package serviceaccount

const (
	KoishiRuntimeCredentialName = "koishi-runtime"
	AudienceBotAPI              = "/api/v1/bot/*"
	ScopeBotQQBindingConsume    = "bot.qq_binding.consume"
	ScopeBotQQVerificationRead  = "bot.qq_verification.read"
	ScopeBotAdmissionSession    = "bot.admission.session"
	ScopeBotAdmissionEvent      = "bot.admission.event"
	ScopeBotAdmissionReview     = "bot.admission.review"
	ScopeBotAdmissionForward    = "bot.admission.forward"
	ScopeBotMemberBlacklist     = "bot.member_blacklist"
)

func KoishiRuntimeScopes() []string {
	return []string{
		ScopeBotQQBindingConsume,
		ScopeBotQQVerificationRead,
		ScopeBotAdmissionSession,
		ScopeBotAdmissionEvent,
		ScopeBotAdmissionReview,
		ScopeBotAdmissionForward,
		ScopeBotMemberBlacklist,
	}
}
