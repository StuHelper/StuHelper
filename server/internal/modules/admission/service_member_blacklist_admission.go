package admission

import "time"

func admissionFailureBlacklistInput(
	input successfulBotEventTxInput,
	failureCount int,
	now time.Time,
) MemberBlacklistCreateInput {
	return MemberBlacklistCreateInput{
		Platform:      input.Session.Platform,
		SubjectType:   MemberBlacklistSubjectQQUser,
		SubjectID:     input.Session.QQID,
		ScopeType:     MemberBlacklistScopeGuild,
		GuildID:       input.Session.GuildID,
		Source:        MemberBlacklistSourceAdmissionFailure,
		ReasonCode:    "admission_timeout_limit",
		ReasonText:    "admission failure limit reached",
		CreatedByType: MemberBlacklistActorSystem,
		CreatedByID:   "system",
		CreatedFrom:   MemberBlacklistFromAdmissionWorker,
		ExpiresAt:     admissionFailureBlacklistExpiresAt(input, now),
		Metadata: map[string]any{
			"admissionSessionID": input.Session.ID,
			"failureCount":       failureCount,
			"failedJoinLimit":    input.Policy.FailedJoinLimit,
			"platform":           input.Session.Platform,
			"guildID":            input.Session.GuildID,
			"botSelfID":          input.Session.BotSelfID,
		},
	}
}

func admissionFailureBlacklistExpiresAt(input successfulBotEventTxInput, now time.Time) *time.Time {
	if input.Policy.BlacklistDurationSeconds == nil {
		return nil
	}
	value := now.Add(time.Duration(*input.Policy.BlacklistDurationSeconds) * time.Second)
	return &value
}
