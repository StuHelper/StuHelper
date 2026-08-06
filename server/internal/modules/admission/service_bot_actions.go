package admission

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"go.uber.org/zap"

	"github.com/StuHelper/StuHelper/server/internal/pkg/ctxutil"
	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
)

const (
	maxAdmissionPendingActionFilterRunes = 64
	botActionClaimFinalizeTimeout        = 5 * time.Second
	maxBotActionPreparationErrorBytes    = 1000
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
	contexts, err := s.pendingActionContexts(ctx, sessions)
	if err != nil {
		return nil, err
	}
	return s.pendingActionsFromSessions(sessions, seeds, contexts)
}

func (s *Service) ClaimQueuedAdmissionActions(
	ctx context.Context,
	filter AdmissionPendingActionFilter,
) ([]AdmissionPendingAction, error) {
	ctx = ctxutil.Normalize(ctx)
	normalized, err := normalizePendingActionFilter(filter)
	if err != nil {
		return nil, err
	}
	now := s.now()
	rows, err := s.repo.ClaimDueBotActions(ctx, normalized, now)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []AdmissionPendingAction{}, nil
	}

	activeClaims := make([]bool, len(rows))
	sessions := make([]AdmissionSession, len(rows))
	for i := range rows {
		activeClaims[i] = true
		sessions[i] = rows[i].Session
	}
	seeds := pendingActionSeeds(sessions, now)
	contexts, err := s.pendingActionContexts(ctx, sessions)
	if err != nil {
		return nil, errors.Join(err, s.abandonClaimedBotActions(ctx, rows))
	}

	actions := make([]AdmissionPendingAction, 0, len(rows))
	for i := range rows {
		if err := ctx.Err(); err != nil {
			return nil, errors.Join(
				err,
				s.abandonClaimedBotActions(ctx, activeBotActionClaims(rows, activeClaims)),
			)
		}

		action, stale, preparationErr := s.pendingActionFromQueuedRow(
			&rows[i],
			seeds[i],
			contexts,
		)
		if preparationErr != nil {
			activeClaims[i] = false
			finalizeErr := s.markBotActionPreparationFailed(ctx, &rows[i], preparationErr)
			if finalizeErr != nil && !errors.Is(finalizeErr, ErrAdmissionBotActionLeaseLost) {
				abandonErr := s.abandonClaimedBotActions(ctx, rows[i:i+1])
				if abandonErr != nil && !errors.Is(abandonErr, ErrAdmissionBotActionLeaseLost) {
					return nil, errors.Join(
						preparationErr,
						finalizeErr,
						abandonErr,
						s.abandonClaimedBotActions(ctx, activeBotActionClaims(rows, activeClaims)),
					)
				}
			}
			logger.L().Warn(
				"admission bot action preparation failed",
				zap.Int64("action_id", rows[i].ID),
				zap.String("session_id", rows[i].SessionID),
				zap.Int("dispatch_attempt", rows[i].AttemptCount),
				zap.Error(preparationErr),
				zap.NamedError("finalize_error", finalizeErr),
			)
			continue
		}

		if stale {
			activeClaims[i] = false
			finalizeErr := s.markBotActionStale(ctx, &rows[i])
			if finalizeErr != nil && !errors.Is(finalizeErr, ErrAdmissionBotActionLeaseLost) {
				abandonErr := s.abandonClaimedBotActions(ctx, rows[i:i+1])
				if abandonErr != nil && !errors.Is(abandonErr, ErrAdmissionBotActionLeaseLost) {
					return nil, errors.Join(
						finalizeErr,
						abandonErr,
						s.abandonClaimedBotActions(ctx, activeBotActionClaims(rows, activeClaims)),
					)
				}
			}
			if finalizeErr != nil {
				logger.L().Warn(
					"admission bot action stale finalization failed",
					zap.Int64("action_id", rows[i].ID),
					zap.String("session_id", rows[i].SessionID),
					zap.Int("dispatch_attempt", rows[i].AttemptCount),
					zap.Bool("lease_lost", errors.Is(finalizeErr, ErrAdmissionBotActionLeaseLost)),
					zap.Error(finalizeErr),
				)
			}
			continue
		}
		actions = append(actions, action)
	}
	if err := ctx.Err(); err != nil {
		return nil, errors.Join(
			err,
			s.abandonClaimedBotActions(ctx, activeBotActionClaims(rows, activeClaims)),
		)
	}
	return actions, nil
}

func activeBotActionClaims(
	rows []AdmissionBotActionOutboxRow,
	active []bool,
) []AdmissionBotActionOutboxRow {
	claimed := make([]AdmissionBotActionOutboxRow, 0, len(rows))
	for i := range rows {
		if i < len(active) && active[i] {
			claimed = append(claimed, rows[i])
		}
	}
	return claimed
}

// abandonClaimedBotActions is deliberately synchronous and single-shot. The
// repository returns the retry budget, so retrying an ambiguous cleanup later
// could reuse a DispatchAttempt and break the lease fence.
func (s *Service) abandonClaimedBotActions(
	parent context.Context,
	rows []AdmissionBotActionOutboxRow,
) error {
	if len(rows) == 0 {
		return nil
	}
	finalizeCtx, cancel := ctxutil.DetachedTimeout(parent, botActionClaimFinalizeTimeout)
	defer cancel()
	affected, err := s.repo.abandonBotActionClaims(finalizeCtx, rows, s.now())
	if err != nil {
		return err
	}
	if affected != int64(len(rows)) {
		return fmt.Errorf(
			"abandon admission bot action claims: released %d of %d: %w",
			affected,
			len(rows),
			ErrAdmissionBotActionLeaseLost,
		)
	}
	return nil
}

func (s *Service) markBotActionPreparationFailed(
	parent context.Context,
	row *AdmissionBotActionOutboxRow,
	preparationErr error,
) error {
	finalizeCtx, cancel := ctxutil.DetachedTimeout(parent, botActionClaimFinalizeTimeout)
	defer cancel()
	return s.repo.MarkBotActionPreparationFailed(
		finalizeCtx,
		row.ID,
		row.AttemptCount,
		truncateBotActionPreparationError(preparationErr),
		s.now(),
	)
}

func (s *Service) markBotActionStale(
	parent context.Context,
	row *AdmissionBotActionOutboxRow,
) error {
	finalizeCtx, cancel := ctxutil.DetachedTimeout(parent, botActionClaimFinalizeTimeout)
	defer cancel()
	return s.repo.MarkBotActionStale(
		finalizeCtx,
		row.ID,
		row.AttemptCount,
		s.now(),
	)
}

func truncateBotActionPreparationError(err error) string {
	if err == nil {
		return "bot action preparation failed"
	}
	message := strings.ToValidUTF8(strings.TrimSpace(err.Error()), "\uFFFD")
	if message == "" {
		return "bot action preparation failed"
	}
	if len(message) <= maxBotActionPreparationErrorBytes {
		return message
	}
	end := maxBotActionPreparationErrorBytes
	for end > 0 && !utf8.RuneStart(message[end]) {
		end--
	}
	return message[:end]
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
	if policy := contexts.policyFor(session); policy != nil {
		applyAdmissionFailureContext(session, contexts.failureFor(session), policy)
	}
	if seed.action != BotActionKick {
		return admissionPendingAction(session, seed.action, seed.deadline), nil
	}
	action, err := resolveKickAction(session, contexts)
	if err != nil {
		return AdmissionPendingAction{}, err
	}
	return admissionPendingAction(session, action, seed.deadline), nil
}

func (s *Service) pendingActionFromQueuedRow(
	row *AdmissionBotActionOutboxRow,
	seed pendingActionSeed,
	contexts pendingActionContexts,
) (AdmissionPendingAction, bool, error) {
	if row == nil {
		return AdmissionPendingAction{}, true, nil
	}
	session := row.Session
	if !sessionCanDispatchQueuedBotAction(&session) {
		return AdmissionPendingAction{}, true, nil
	}
	action, err := s.pendingActionFromSession(&session, seed, contexts)
	if err != nil {
		return AdmissionPendingAction{}, false, err
	}
	if !queuedActionCanDispatch(row.Action, action.Action) {
		return AdmissionPendingAction{}, true, nil
	}
	action.ActionID = strconv.FormatInt(row.ID, 10)
	action.DispatchAttempt = row.AttemptCount
	return action, false, nil
}

func queuedActionCanDispatch(queued BotAction, current BotAction) bool {
	if queued == current {
		return true
	}
	return queued == BotActionKick && current == BotActionBlacklist
}

func sessionCanDispatchQueuedBotAction(session *AdmissionSession) bool {
	if session == nil || session.CancelledAt != nil {
		return false
	}
	switch session.Status {
	case StatusJoinedMuted, StatusLinked, StatusMaterialSubmitted, StatusVerified:
		return true
	default:
		return false
	}
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
	case session.Status == StatusJoinedMuted:
		return BotActionRemind, session.LinkWaitDeadlineAt
	case session.Status == StatusLinked && now.After(session.SubmissionWaitDeadlineAt):
		return BotActionKick, session.SubmissionWaitDeadlineAt
	case session.Status == StatusLinked:
		return BotActionRemind, session.SubmissionWaitDeadlineAt
	case session.ManualReviewDeadlineAt != nil && now.After(*session.ManualReviewDeadlineAt):
		return BotActionKick, *session.ManualReviewDeadlineAt
	case session.Status == StatusMaterialSubmitted && session.ManualReviewDeadlineAt != nil:
		return BotActionRemind, *session.ManualReviewDeadlineAt
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
		SessionID:              session.ID,
		Action:                 action,
		Platform:               session.Platform,
		BotSelfID:              session.BotSelfID,
		GuildID:                session.GuildID,
		ChannelID:              session.ChannelID,
		QQID:                   session.QQID,
		AuthURL:                session.AuthURL,
		DeadlineAt:             deadline,
		EligibilityRevision:    session.EligibilityRevision,
		FailureCount:           session.FailureCount,
		RemainingRetryCount:    session.RemainingRetryCount,
		WillBlacklistOnTimeout: session.WillBlacklistOnTimeout,
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
	if isAdmissionBotFieldTooLong(normalized.Platform, maxAdmissionPendingActionFilterRunes) ||
		isAdmissionBotFieldTooLong(normalized.BotSelfID, maxAdmissionPendingActionFilterRunes) {
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
