package admission

func adminMemberBlacklistCreateInput(
	req botMemberBlacklistCreateHTTPRequest,
	userID int64,
) MemberBlacklistCreateInput {
	return MemberBlacklistCreateInput{
		Platform:      req.Platform,
		SubjectType:   req.SubjectType,
		SubjectID:     req.SubjectID,
		ScopeType:     req.ScopeType,
		GuildID:       req.GuildID,
		Source:        MemberBlacklistSourceManualAdmin,
		ReasonCode:    req.ReasonCode,
		ReasonText:    req.ReasonText,
		CreatedByType: MemberBlacklistActorAdminUser,
		CreatedByID:   adminMemberBlacklistActorID(userID),
		CreatedFrom:   MemberBlacklistFromAdminConsole,
		ExpiresAt:     req.ExpiresAt,
		Metadata:      req.Metadata,
	}
}

func adminMemberBlacklistReleaseInput(
	id string,
	req memberBlacklistReleaseHTTPRequest,
	userID int64,
) MemberBlacklistReleaseInput {
	return MemberBlacklistReleaseInput{
		ID:                id,
		ReleasedByType:    MemberBlacklistActorAdminUser,
		ReleasedByID:      adminMemberBlacklistActorID(userID),
		ReleaseReasonCode: req.ReleaseReasonCode,
		ReleaseReason:     req.ReleaseReason,
	}
}

func adminMemberBlacklistReleaseBySubjectInput(
	req memberBlacklistReleaseBySubjectHTTPRequest,
	userID int64,
) MemberBlacklistReleaseBySubjectInput {
	return MemberBlacklistReleaseBySubjectInput{
		Platform:          req.Platform,
		SubjectType:       req.SubjectType,
		SubjectID:         req.SubjectID,
		ScopeType:         req.ScopeType,
		GuildID:           req.GuildID,
		ReleasedByType:    MemberBlacklistActorAdminUser,
		ReleasedByID:      adminMemberBlacklistActorID(userID),
		ReleaseReasonCode: req.ReleaseReasonCode,
		ReleaseReason:     req.ReleaseReason,
	}
}
