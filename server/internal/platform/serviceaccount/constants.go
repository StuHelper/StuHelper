package serviceaccount

import "github.com/StuHelper/StuHelper/server/internal/pkg/botcredential"

const (
	KoishiRuntimeCredentialName   = botcredential.KoishiRuntimeCredentialName
	AudienceBotAPI                = botcredential.AudienceBotAPI
	ScopeBotQQBindingConsume      = botcredential.ScopeBotQQBindingConsume
	ScopeBotQQVerificationRead    = botcredential.ScopeBotQQVerificationRead
	ScopeBotAdmissionSession      = botcredential.ScopeBotAdmissionSession
	ScopeBotAdmissionEvent        = botcredential.ScopeBotAdmissionEvent
	ScopeBotAdmissionReview       = botcredential.ScopeBotAdmissionReview
	ScopeBotAdmissionForward      = botcredential.ScopeBotAdmissionForward
	ScopeBotMemberBlacklistRead   = botcredential.ScopeBotMemberBlacklistRead
	ScopeBotMemberBlacklistManage = botcredential.ScopeBotMemberBlacklistManage
)

func KoishiRuntimeScopes() []string {
	return botcredential.KoishiRuntimeScopes()
}
