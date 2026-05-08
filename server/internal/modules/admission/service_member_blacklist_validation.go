package admission

import "strings"

func validateMemberBlacklistAccessQuery(input MemberBlacklistAccessQuery) error {
	if input.Platform == "" || input.SubjectType == "" || input.SubjectID == "" || input.GuildID == "" {
		return ErrMemberBlacklistInvalidInput
	}
	return nil
}

func validateMemberBlacklistCreateInput(input MemberBlacklistCreateInput) error {
	if input.Platform == "" || input.SubjectType == "" || input.SubjectID == "" {
		return ErrMemberBlacklistInvalidInput
	}
	if input.ScopeType != MemberBlacklistScopeGuild && input.ScopeType != MemberBlacklistScopeGlobal {
		return ErrMemberBlacklistInvalidInput
	}
	if input.ScopeType == MemberBlacklistScopeGuild && input.GuildID == "" {
		return ErrMemberBlacklistInvalidInput
	}
	if input.ScopeType == MemberBlacklistScopeGlobal && input.GuildID != "" {
		return ErrMemberBlacklistInvalidInput
	}
	if input.Source == "" || input.ReasonCode == "" || input.CreatedByType == "" {
		return ErrMemberBlacklistInvalidInput
	}
	if !validMemberBlacklistSource(input.Source) || !validMemberBlacklistActorType(input.CreatedByType) {
		return ErrMemberBlacklistInvalidInput
	}
	if !validMemberBlacklistCreatedFrom(input.CreatedFrom) {
		return ErrMemberBlacklistInvalidInput
	}
	if input.CreatedByID == "" || input.CreatedFrom == "" {
		return ErrMemberBlacklistInvalidInput
	}
	return nil
}

func validateMemberBlacklistReleaseBySubjectInput(input MemberBlacklistReleaseBySubjectInput) error {
	if input.Platform == "" || input.SubjectType == "" || input.SubjectID == "" {
		return ErrMemberBlacklistInvalidInput
	}
	if input.ScopeType == MemberBlacklistScopeGuild && input.GuildID == "" {
		return ErrMemberBlacklistInvalidInput
	}
	if input.ScopeType == MemberBlacklistScopeGlobal && input.GuildID != "" {
		return ErrMemberBlacklistInvalidInput
	}
	if input.ScopeType != MemberBlacklistScopeGuild && input.ScopeType != MemberBlacklistScopeGlobal {
		return ErrMemberBlacklistInvalidInput
	}
	if input.ReleasedByID == "" || input.ReleaseReasonCode == "" {
		return ErrMemberBlacklistInvalidInput
	}
	return nil
}

func normalizeMemberBlacklistReleaseInput(input MemberBlacklistReleaseInput) MemberBlacklistReleaseInput {
	input.ID = strings.TrimSpace(input.ID)
	input.ReleasedByID = strings.TrimSpace(input.ReleasedByID)
	input.ReleaseReasonCode = strings.TrimSpace(input.ReleaseReasonCode)
	input.ReleaseReason = strings.TrimSpace(input.ReleaseReason)
	if input.ReleasedByType == "" {
		input.ReleasedByType = MemberBlacklistActorServiceAccount
	}
	return input
}

func normalizeMemberBlacklistReleaseBySubjectInput(
	input MemberBlacklistReleaseBySubjectInput,
) MemberBlacklistReleaseBySubjectInput {
	input.Platform = strings.TrimSpace(input.Platform)
	input.SubjectID = strings.TrimSpace(input.SubjectID)
	input.GuildID = strings.TrimSpace(input.GuildID)
	input.ReleasedByID = strings.TrimSpace(input.ReleasedByID)
	input.ReleaseReasonCode = strings.TrimSpace(input.ReleaseReasonCode)
	input.ReleaseReason = strings.TrimSpace(input.ReleaseReason)
	if input.SubjectType == "" {
		input.SubjectType = MemberBlacklistSubjectQQUser
	}
	if input.ReleasedByType == "" {
		input.ReleasedByType = MemberBlacklistActorServiceAccount
	}
	return input
}

func normalizeMemberBlacklistListFilter(input MemberBlacklistListFilter) MemberBlacklistListFilter {
	input.Platform = strings.TrimSpace(input.Platform)
	input.SubjectID = strings.TrimSpace(input.SubjectID)
	input.GuildID = strings.TrimSpace(input.GuildID)
	if input.PageSize <= 0 {
		input.PageSize = defaultMemberBlacklistPageSize
	}
	if input.PageSize > maxMemberBlacklistPageSize {
		input.PageSize = maxMemberBlacklistPageSize
	}
	if input.Offset < 0 {
		input.Offset = 0
	}
	return input
}

func nonNilMetadata(metadata map[string]any) map[string]any {
	if metadata == nil {
		return map[string]any{}
	}
	return metadata
}

func nullableGuildID(scope MemberBlacklistScopeType, guildID string) *string {
	if scope != MemberBlacklistScopeGuild {
		return nil
	}
	value := strings.TrimSpace(guildID)
	return &value
}

func validMemberBlacklistSource(value MemberBlacklistSource) bool {
	switch value {
	case MemberBlacklistSourceAdmissionFailure, MemberBlacklistSourceManualAdmin,
		MemberBlacklistSourceKickBlacklist, MemberBlacklistSourceModerationAction,
		MemberBlacklistSourceLegacyKoishi, MemberBlacklistSourceLegacyAdmission:
		return true
	default:
		return false
	}
}

func validMemberBlacklistActorType(value MemberBlacklistActorType) bool {
	switch value {
	case MemberBlacklistActorSystem, MemberBlacklistActorAdminUser,
		MemberBlacklistActorQQOperator, MemberBlacklistActorServiceAccount:
		return true
	default:
		return false
	}
}

func validMemberBlacklistCreatedFrom(value MemberBlacklistCreatedFrom) bool {
	switch value {
	case MemberBlacklistFromAdmissionWorker, MemberBlacklistFromQQCommand,
		MemberBlacklistFromKoishiConsole, MemberBlacklistFromAdminConsole,
		MemberBlacklistFromModerationReview:
		return true
	default:
		return false
	}
}
