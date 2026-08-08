package admission

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// ReevaluateStudentEligibility consumes an invalidation event. The event only
// supplies a minimum monotonic fence; the authorization decision is always
// recomputed through StudentEligibilityGateway before any session or bot action
// changes.
func (s *Service) ReevaluateStudentEligibility(
	ctx context.Context,
	userID int64,
	schoolID int64,
	minimumRevision int64,
) error {
	if userID <= 0 || schoolID <= 0 || minimumRevision <= 0 || s.studentEligibility == nil {
		return ErrAdmissionProjectionUnavailable
	}
	decision, err := s.studentEligibility.EvaluateStudentEligibility(ctx, userID, schoolID)
	if err != nil || decision.Revision < minimumRevision || decision.Revision <= 0 {
		return ErrAdmissionProjectionUnavailable
	}
	now := s.now()
	return s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		sessions, err := s.repo.ApplyStudentEligibilityDecisionTx(
			ctx,
			tx,
			userID,
			schoolID,
			decision.Eligible,
			decision.CredentialClass,
			decision.Revision,
			now,
		)
		if err != nil {
			return err
		}
		for index := range sessions {
			if err := s.queueBotActionTx(
				ctx,
				tx,
				&sessions[index],
				BotActionRelease,
				now,
				now,
			); err != nil {
				return err
			}
		}
		return nil
	})
}
