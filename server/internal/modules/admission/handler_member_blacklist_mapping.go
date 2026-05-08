package admission

import "git.stuhelper.com/StuHelper/StuHelper/internal/platform/serviceaccount"

func botMemberBlacklistCreateInput(req botMemberBlacklistCreateHTTPRequest) MemberBlacklistCreateInput {
	actorType, actorID := botMemberBlacklistActor(req.CreatedFrom, req.OperatorID)
	return MemberBlacklistCreateInput{
		Platform:      req.Platform,
		SubjectType:   req.SubjectType,
		SubjectID:     req.SubjectID,
		ScopeType:     req.ScopeType,
		GuildID:       req.GuildID,
		Source:        req.Source,
		ReasonCode:    req.ReasonCode,
		ReasonText:    req.ReasonText,
		CreatedByType: actorType,
		CreatedByID:   actorID,
		CreatedFrom:   req.CreatedFrom,
		ExpiresAt:     req.ExpiresAt,
		Metadata:      req.Metadata,
	}
}

func memberBlacklistReleaseInput(id string, req memberBlacklistReleaseHTTPRequest) MemberBlacklistReleaseInput {
	actorType, actorID := botMemberBlacklistActor(MemberBlacklistFromKoishiConsole, req.OperatorID)
	return MemberBlacklistReleaseInput{
		ID:                id,
		ReleasedByType:    actorType,
		ReleasedByID:      actorID,
		ReleaseReasonCode: req.ReleaseReasonCode,
		ReleaseReason:     req.ReleaseReason,
	}
}

func memberBlacklistReleaseBySubjectInput(
	req memberBlacklistReleaseBySubjectHTTPRequest,
) MemberBlacklistReleaseBySubjectInput {
	actorType, actorID := botMemberBlacklistActor(MemberBlacklistFromKoishiConsole, req.OperatorID)
	return MemberBlacklistReleaseBySubjectInput{
		Platform:          req.Platform,
		SubjectType:       req.SubjectType,
		SubjectID:         req.SubjectID,
		ScopeType:         req.ScopeType,
		GuildID:           req.GuildID,
		ReleasedByType:    actorType,
		ReleasedByID:      actorID,
		ReleaseReasonCode: req.ReleaseReasonCode,
		ReleaseReason:     req.ReleaseReason,
	}
}

func botMemberBlacklistActor(
	createdFrom MemberBlacklistCreatedFrom,
	operatorID string,
) (MemberBlacklistActorType, string) {
	if operatorID != "" && createdFrom != MemberBlacklistFromKoishiConsole {
		return MemberBlacklistActorQQOperator, operatorID
	}
	return MemberBlacklistActorServiceAccount, serviceaccount.KoishiRuntimeCredentialName
}

func validBotMemberBlacklistCreateRequest(req botMemberBlacklistCreateHTTPRequest) bool {
	switch req.Source {
	case MemberBlacklistSourceManualAdmin:
		return req.CreatedFrom == MemberBlacklistFromKoishiConsole ||
			req.CreatedFrom == MemberBlacklistFromQQCommand
	case MemberBlacklistSourceKickBlacklist:
		return req.CreatedFrom == MemberBlacklistFromQQCommand
	case MemberBlacklistSourceModerationAction:
		return req.CreatedFrom == MemberBlacklistFromModerationReview
	default:
		return false
	}
}
