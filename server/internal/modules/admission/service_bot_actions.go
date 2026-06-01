package admission

import (
	"context"
	"strings"
	"time"
)

func (s *Service) ListPendingAdmissionActions(
	ctx context.Context,
	filter AdmissionPendingActionFilter,
) ([]AdmissionPendingAction, error) {
	normalized, err := normalizePendingActionFilter(filter)
	if err != nil {
		return nil, err
	}
	sessions, err := s.repo.ListPendingActionSessions(ctx, normalized, s.now())
	if err != nil {
		return nil, err
	}
	now := s.now()
	seeds := pendingActionSeeds(sessions, now)
	contexts, err := s.pendingActionContexts(ctx, sessions, seeds)
	if err != nil {
		return nil, err
	}
	return s.pendingActionsFromSessions(sessions, seeds, contexts)
}

func (s *Service) ListPendingFreshmanForwards(ctx context.Context) ([]FreshmanForwardItem, error) {
	records, err := s.repo.ListPendingFreshmanForwards(ctx)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return []FreshmanForwardItem{}, nil
	}
	if s.materialStore == nil {
		return nil, ErrAdmissionMaterialStoreUnavailable
	}
	return s.freshmanForwardItems(ctx, records)
}

func (s *Service) ViewFreshmanApplicationFromBot(
	ctx context.Context,
	input BotFreshmanCommandInput,
) (*FreshmanApplication, error) {
	if err := s.authorizeBotCommandForApplication(ctx, input); err != nil {
		return nil, err
	}
	return s.repo.GetFreshmanApplicationByID(ctx, strings.TrimSpace(input.ApplicationID))
}

func (s *Service) pendingActionsFromSessions(
	sessions []AdmissionSession,
	seeds []pendingActionSeed,
	contexts pendingActionContexts,
) ([]AdmissionPendingAction, error) {
	actions := make([]AdmissionPendingAction, 0, len(sessions))
	for i := range sessions {
		action, err := s.pendingActionFromSession(&sessions[i], seeds[i], contexts)
		if err != nil {
			return nil, err
		}
		actions = append(actions, action)
	}
	return actions, nil
}

func (s *Service) pendingActionFromSession(
	session *AdmissionSession,
	seed pendingActionSeed,
	contexts pendingActionContexts,
) (AdmissionPendingAction, error) {
	if seed.action != BotActionKick {
		return admissionPendingAction(session, seed.action, seed.deadline), nil
	}
	action, err := resolveKickAction(session, contexts)
	if err != nil {
		return AdmissionPendingAction{}, err
	}
	return admissionPendingAction(session, action, seed.deadline), nil
}

func resolveKickAction(session *AdmissionSession, contexts pendingActionContexts) (BotAction, error) {
	policy := contexts.policyFor(session)
	if policy == nil {
		return "", ErrAdmissionPolicyNotFound
	}
	failure := contexts.failureFor(session)
	if nextFailureReachesBlacklist(failure, policy) {
		return BotActionBlacklist, nil
	}
	return BotActionKick, nil
}

func nextFailureReachesBlacklist(failure *AdmissionFailure, policy *AdmissionPolicy) bool {
	count := 0
	if failure != nil {
		count = failure.FailureCount
	}
	return count+1 >= policy.FailedJoinLimit
}

func resolvePendingAction(session *AdmissionSession, now time.Time) (BotAction, time.Time) {
	switch {
	case session.Status == StatusVerified:
		return BotActionRelease, session.InitialMuteUntil
	case session.Status == StatusJoinedMuted && now.After(session.LinkWaitDeadlineAt):
		return BotActionKick, session.LinkWaitDeadlineAt
	case session.Status == StatusLinked && now.After(session.SubmissionWaitDeadlineAt):
		return BotActionKick, session.SubmissionWaitDeadlineAt
	case session.ManualReviewDeadlineAt != nil && now.After(*session.ManualReviewDeadlineAt):
		return BotActionKick, *session.ManualReviewDeadlineAt
	default:
		return BotActionRemind, session.LinkWaitDeadlineAt
	}
}

func (s *Service) freshmanForwardItems(
	ctx context.Context,
	records []freshmanForwardRecord,
) ([]FreshmanForwardItem, error) {
	items := make([]FreshmanForwardItem, 0, len(records))
	for i := range records {
		item, err := s.freshmanForwardItem(ctx, records[i])
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *Service) freshmanForwardItem(ctx context.Context, record freshmanForwardRecord) (FreshmanForwardItem, error) {
	materialURL, err := s.materialStore.GetAdmissionMaterialURL(ctx, record.ObjectKey)
	if err != nil {
		return FreshmanForwardItem{}, err
	}
	return FreshmanForwardItem{
		Application:        &record.Application,
		MaterialURL:        materialURL,
		ManagementGuildIDs: record.ManagementGuildIDs,
		Platform:           record.Platform,
		BotSelfID:          record.BotSelfID,
		SchoolName:         record.SchoolName,
		QQID:               record.QQID,
	}, nil
}

func (s *Service) authorizeBotCommandForApplication(ctx context.Context, input BotFreshmanCommandInput) error {
	reviewInput := BotFreshmanReviewInput{
		ApplicationID: input.ApplicationID,
		OperatorQQID:  input.OperatorQQID,
		GuildID:       input.GuildID,
		ChannelID:     input.ChannelID,
		RawCommand:    input.RawCommand,
	}
	_, err := s.authorizeBotFreshmanReviewer(ctx, normalizeBotFreshmanReviewInput(reviewInput))
	return err
}

func admissionPendingAction(
	session *AdmissionSession,
	action BotAction,
	deadline time.Time,
) AdmissionPendingAction {
	return AdmissionPendingAction{
		SessionID:  session.ID,
		Action:     action,
		Platform:   session.Platform,
		BotSelfID:  session.BotSelfID,
		GuildID:    session.GuildID,
		ChannelID:  session.ChannelID,
		QQID:       session.QQID,
		AuthURL:    session.AuthURL,
		DeadlineAt: deadline,
	}
}

func normalizePendingActionFilter(filter AdmissionPendingActionFilter) (AdmissionPendingActionFilter, error) {
	normalized := AdmissionPendingActionFilter{
		Platform:  strings.TrimSpace(filter.Platform),
		BotSelfID: strings.TrimSpace(filter.BotSelfID),
		Limit:     normalizePendingActionLimit(filter.Limit),
	}
	if normalized.Platform == "" || normalized.BotSelfID == "" {
		return AdmissionPendingActionFilter{}, ErrAdmissionPendingActionFilterInvalid
	}
	return normalized, nil
}

func normalizePendingActionLimit(limit int) int {
	if limit <= 0 {
		return defaultPendingActionLimit
	}
	if limit > maxPendingActionLimit {
		return maxPendingActionLimit
	}
	return limit
}
