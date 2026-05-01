package serviceaccount

const (
	KoishiRuntimeCredentialName = "koishi-runtime"
	AudienceBotAPI              = "/api/v1/bot/*"
	ScopeBotQQBindingConsume    = "bot.qq_binding.consume"
	ScopeBotQQVerificationRead  = "bot.qq_verification.read"
)

func KoishiRuntimeScopes() []string {
	return []string{
		ScopeBotQQBindingConsume,
		ScopeBotQQVerificationRead,
	}
}
