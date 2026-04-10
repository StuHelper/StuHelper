package review

// moderationDecision describes the persisted review state derived from content moderation.
type moderationDecision struct {
	Status      string
	ContentFlag *string
}

func buildReviewModerationDecision(result *ContentCheckResult) (moderationDecision, error) {
	decision := moderationDecision{Status: StatusPublished}
	if result == nil || result.MatchCount == 0 {
		return decision, nil
	}

	if !result.IsValid && result.Level == "block" {
		return moderationDecision{}, ErrSensitiveContent
	}

	switch result.Level {
	case ContentFlagWarn:
		flag := ContentFlagWarn
		decision.ContentFlag = &flag
	case ContentFlagReview:
		flag := ContentFlagReview
		decision.Status = StatusPendingReview
		decision.ContentFlag = &flag
	}

	return decision, nil
}

func buildReplyModerationStatus(result *ContentCheckResult) (string, error) {
	decision, err := buildReviewModerationDecision(result)
	if err != nil {
		return "", err
	}
	return decision.Status, nil
}

func isPublicReviewStatus(status string) bool {
	return status == StatusPublished
}
