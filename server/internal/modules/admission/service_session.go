package admission

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/audit"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/id"
)

const initialReminderGrace = time.Duration(DefaultInitialReminderGraceSeconds) * time.Second
const maxAdmissionJoinTokenCreateAttempts = 5

const (
	maxAdmissionBotPlatformRunes  = 32
	maxAdmissionBotGuildIDRunes   = 128
	maxAdmissionBotChannelIDRunes = 128
	maxAdmissionBotQQIDRunes      = 64
	maxAdmissionBotSelfIDRunes    = 64
)

func (s *Service) CreateBotSession(ctx context.Context, input BotSessionCreateInput) (*CreatedAdmissionSession, error) {
	input = normalizeBotSessionCreateInput(input)
	if err := validateBotSessionCreateInput(input); err != nil {
		return nil, err
	}
	if err := s.ensureBotSessionMemberAllowed(ctx, input); err != nil {
		return nil, err
	}
	policy, err := s.loadPolicy(ctx, input.Platform, input.GuildID)
	if err != nil {
		return nil, err
	}
	verifiedUserID, err := s.repo.GetVerifiedAdmissionUserByQQ(ctx, input.QQID, policy.SchoolID, s.now())
	if err != nil {
		return nil, err
	}
	if verifiedUserID != nil {
		return s.createVerifiedBotSession(ctx, input, policy, *verifiedUserID)
	}
	return s.createPendingBotSession(ctx, input, policy)
}

func (s *Service) createPendingBotSession(
	ctx context.Context,
	input BotSessionCreateInput,
	policy *AdmissionPolicy,
) (*CreatedAdmissionSession, error) {
	for attempt := 0; attempt < maxAdmissionJoinTokenCreateAttempts; attempt++ {
		token, err := s.generateJoinToken()
		if err != nil {
			return nil, fmt.Errorf("CreateBotSession token: %w", err)
		}
		session, err := s.newAdmissionSession(input, policy, token)
		if err != nil {
			return nil, err
		}
		if err := s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
			if err := s.repo.CreateSessionTx(ctx, tx, session); err != nil {
				return err
			}
			return s.queueInitialBotActionsTx(ctx, tx, session, s.now())
		}); err != nil {
			if isAdmissionSessionActiveSubjectUniqueViolation(err) {
				return s.reuseActiveBotSessionAfterCreateConflict(ctx, input)
			}
			if isAdmissionSessionTokenHashUniqueViolation(err) {
				continue
			}
			return nil, err
		}
		if err := s.attachAdmissionFailureContext(ctx, session, policy); err != nil {
			return nil, err
		}
		return &CreatedAdmissionSession{
			Session: session,
			Token:   token,
			AuthURL: session.AuthURL,
		}, nil
	}
	return nil, fmt.Errorf("CreateBotSession token: exhausted %d token attempts", maxAdmissionJoinTokenCreateAttempts)
}

func (s *Service) createVerifiedBotSession(
	ctx context.Context,
	input BotSessionCreateInput,
	policy *AdmissionPolicy,
	userID int64,
) (*CreatedAdmissionSession, error) {
	created, _, err := s.createVerifiedBotSessionWithCancelledIDs(ctx, input, policy, userID)
	return created, err
}

func (s *Service) createVerifiedBotSessionWithCancelledIDs(
	ctx context.Context,
	input BotSessionCreateInput,
	policy *AdmissionPolicy,
	userID int64,
) (*CreatedAdmissionSession, []string, error) {
	subject := botSessionCreateSubject(input)
	for attempt := 0; attempt < maxAdmissionJoinTokenCreateAttempts; attempt++ {
		token, err := s.generateJoinToken()
		if err != nil {
			return nil, nil, fmt.Errorf("CreateBotSession verified token: %w", err)
		}
		session, err := s.newVerifiedAdmissionSession(input, policy, token, userID)
		if err != nil {
			return nil, nil, err
		}
		var cancelledIDs []string
		if err := s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
			var err error
			cancelledIDs, err = s.repo.CancelInProgressSessionsBySubjectTx(ctx, tx, subject, s.now())
			if err != nil {
				return err
			}
			if err := s.repo.CreateSessionTx(ctx, tx, session); err != nil {
				return err
			}
			if _, err = s.repo.ResetAdmissionFailureCountTx(ctx, tx, input.Platform, input.GuildID, input.QQID, s.now()); err != nil {
				return err
			}
			return s.queueInitialBotActionsTx(ctx, tx, session, s.now())
		}); err != nil {
			if isAdmissionSessionTokenHashUniqueViolation(err) {
				continue
			}
			return nil, nil, err
		}
		applyAdmissionFailureContext(session, nil, policy)
		return &CreatedAdmissionSession{
			Session: session,
			Token:   token,
			AuthURL: session.AuthURL,
		}, cancelledIDs, nil
	}
	return nil, nil, fmt.Errorf("CreateBotSession verified token: exhausted %d token attempts", maxAdmissionJoinTokenCreateAttempts)
}

func (s *Service) reuseActiveBotSessionAfterCreateConflict(
	ctx context.Context,
	input BotSessionCreateInput,
) (*CreatedAdmissionSession, error) {
	session, err := s.repo.GetLatestSessionBySubject(ctx, botSessionCreateSubject(input))
	if err != nil {
		return nil, err
	}
	if err := s.validateResendableSession(session); err != nil {
		return nil, err
	}
	policy, err := s.loadPolicy(ctx, input.Platform, input.GuildID)
	if err != nil {
		return nil, err
	}
	if err := s.attachAdmissionFailureContext(ctx, session, policy); err != nil {
		return nil, err
	}
	return &CreatedAdmissionSession{
		Session: session,
		Token:   admissionTokenFromAuthURL(session.AuthURL),
		AuthURL: session.AuthURL,
	}, nil
}

func (s *Service) GetBotAdmissionSession(ctx context.Context, input BotSessionSubjectInput) (*AdmissionSession, error) {
	input = normalizeBotSessionSubjectInput(input)
	if err := validateBotSessionSubjectInput(input); err != nil {
		return nil, err
	}
	session, err := s.repo.GetLatestSessionBySubject(ctx, input)
	if err != nil {
		return nil, err
	}
	policy, err := s.loadPolicy(ctx, input.Platform, input.GuildID)
	if err != nil {
		return nil, err
	}
	if err := s.attachAdmissionFailureContext(ctx, session, policy); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *Service) ResendBotAdmissionSession(ctx context.Context, input BotSessionSubjectInput) (*AdmissionSession, error) {
	session, err := s.GetBotAdmissionSession(ctx, input)
	if err != nil {
		return nil, err
	}
	if err := s.validateResendableSession(session); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *Service) ResendAdminAdmissionSession(
	ctx context.Context,
	input AdminAdmissionSessionActionInput,
) (*AdmissionSession, error) {
	input.SessionID = strings.TrimSpace(input.SessionID)
	if input.SessionID == "" || input.OperatorUserID <= 0 {
		return nil, ErrAdmissionInvalidInput
	}
	session, err := s.getAdminActionSession(ctx, input.SessionID)
	if err != nil {
		return nil, err
	}
	if err := s.validateResendableSession(session); err != nil {
		return nil, err
	}
	var queued *AdmissionSession
	now := s.now()
	err = s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		var err error
		queued, err = s.repo.QueueAdmissionReminderNowTx(ctx, tx, session.ID, now)
		if err != nil {
			return err
		}
		return s.queueBotActionTx(ctx, tx, queued, BotActionRemind, now, now)
	})
	if err != nil {
		if errors.Is(err, ErrAdmissionTokenNotFound) {
			return nil, ErrAdmissionInvalidStatus
		}
		return nil, err
	}
	s.auditAdminSessionAction(ctx, "resend", queued, input.OperatorUserID, nil)
	return queued, nil
}

func (s *Service) RegenerateBotAdmissionSession(
	ctx context.Context,
	input BotSessionCreateInput,
) (*CreatedAdmissionSession, error) {
	created, cancelledIDs, err := s.regenerateAdmissionSession(ctx, input, nil)
	if err != nil {
		return nil, err
	}
	s.auditBotSessionRegenerated(ctx, created.Session, cancelledIDs)
	return created, nil
}

func (s *Service) SkipBotAdmissionSession(
	ctx context.Context,
	input BotSessionOperatorInput,
) (*AdmissionSession, error) {
	input = normalizeBotSessionOperatorInput(input)
	if err := validateBotSessionOperatorInput(input); err != nil {
		return nil, err
	}
	subject := botSessionOperatorSubject(input)
	if _, err := s.loadPolicy(ctx, input.Platform, input.GuildID); err != nil {
		return nil, err
	}
	var skipped *AdmissionSession
	if err := s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		session, err := s.repo.GetLatestSessionBySubjectForUpdateTx(ctx, tx, subject)
		if err != nil {
			return err
		}
		if err := validateSkippableBotSession(session); err != nil {
			return err
		}
		skipped, err = s.repo.CancelInProgressSessionByIDTx(ctx, tx, session.ID, s.now())
		return err
	}); err != nil {
		return nil, err
	}
	s.auditBotSessionSkipped(ctx, skipped, input.OperatorQQID)
	return skipped, nil
}

func (s *Service) ResetBotAdmissionFailureCount(
	ctx context.Context,
	input BotSessionOperatorInput,
) (*AdmissionFailureResetResult, error) {
	input = normalizeBotSessionOperatorInput(input)
	if err := validateBotSessionOperatorInput(input); err != nil {
		return nil, err
	}
	if _, err := s.loadPolicy(ctx, input.Platform, input.GuildID); err != nil {
		return nil, err
	}
	var previous int
	if err := s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		var err error
		previous, err = s.repo.ResetAdmissionFailureCountTx(ctx, tx, input.Platform, input.GuildID, input.QQID, s.now())
		return err
	}); err != nil {
		return nil, err
	}
	result := &AdmissionFailureResetResult{
		Platform:             input.Platform,
		GuildID:              input.GuildID,
		QQID:                 input.QQID,
		PreviousFailureCount: previous,
	}
	s.auditBotAdmissionFailureCountReset(ctx, result, input.OperatorQQID)
	return result, nil
}

func (s *Service) RegenerateAdminAdmissionSession(
	ctx context.Context,
	input AdminAdmissionSessionActionInput,
) (*CreatedAdmissionSession, error) {
	input.SessionID = strings.TrimSpace(input.SessionID)
	if input.SessionID == "" || input.OperatorUserID <= 0 {
		return nil, ErrAdmissionInvalidInput
	}
	session, err := s.getAdminActionSession(ctx, input.SessionID)
	if err != nil {
		return nil, err
	}
	now := s.now()
	created, cancelledIDs, err := s.regenerateAdmissionSession(ctx, BotSessionCreateInput{
		Platform:  session.Platform,
		GuildID:   session.GuildID,
		ChannelID: session.ChannelID,
		QQID:      session.QQID,
		BotSelfID: session.BotSelfID,
	}, &now)
	if err != nil {
		return nil, err
	}
	s.auditAdminSessionAction(ctx, "regenerate", created.Session, input.OperatorUserID, map[string]any{
		"sourceSessionID":       session.ID,
		"cancelledSessionIDs":   cancelledIDs,
		"cancelledSessionCount": len(cancelledIDs),
	})
	return created, nil
}

func (s *Service) CancelAdminAdmissionSession(
	ctx context.Context,
	input AdminAdmissionSessionActionInput,
) (*AdmissionSession, error) {
	input.SessionID = strings.TrimSpace(input.SessionID)
	if input.SessionID == "" || input.OperatorUserID <= 0 {
		return nil, ErrAdmissionInvalidInput
	}
	if _, err := s.getAdminActionSession(ctx, input.SessionID); err != nil {
		return nil, err
	}
	cancelled, err := s.repo.CancelInProgressSessionByID(ctx, input.SessionID, s.now())
	if err != nil {
		if errors.Is(err, ErrAdmissionTokenNotFound) {
			return nil, ErrAdmissionInvalidStatus
		}
		return nil, err
	}
	s.auditAdminSessionAction(ctx, "cancel", cancelled, input.OperatorUserID, nil)
	return cancelled, nil
}

func (s *Service) PreviewToken(ctx context.Context, token string, qqQuery string) (*AdmissionSession, error) {
	session, err := s.repo.GetSessionByTokenHash(ctx, s.hashToken(token))
	if err != nil {
		return nil, err
	}
	if err := s.validateTokenSession(session, qqQuery); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *Service) LinkTokenToUser(ctx context.Context, input AdmissionTokenLinkInput) (*AdmissionSession, error) {
	var linked *AdmissionSession
	err := s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		session, err := s.repo.GetSessionByTokenHashForUpdate(ctx, tx, s.hashToken(input.Token))
		if err != nil {
			return err
		}
		if err := s.validateTokenSession(session, input.QQQuery); err != nil {
			if errors.Is(err, ErrAdmissionTokenConsumed) && sessionLinkedToUser(session, input.UserID) {
				if err := s.ensureConsumedSessionCanResume(session); err != nil {
					return err
				}
				linked = session
				return nil
			}
			return err
		}
		policy, err := s.loadPolicy(ctx, session.Platform, session.GuildID)
		if err != nil {
			return err
		}
		if err := s.qqGateway.EnsureQQBindingForUserTx(ctx, tx, input.UserID, session.QQID); err != nil {
			return err
		}
		now := s.now()
		deadline := now.Add(time.Duration(policy.SubmissionWaitSeconds) * time.Second)
		linked, err = s.repo.MarkTokenConsumedAndLinked(ctx, tx, session.ID, input.UserID, now, deadline)
		if err != nil {
			return err
		}
		linked, err = s.resolveLinkedSessionVerificationTx(ctx, tx, linked, input.UserID, policy, now)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("LinkTokenToUser: %w", err)
	}
	return linked, nil
}

func (s *Service) resolveLinkedSessionVerificationTx(
	ctx context.Context,
	tx pgx.Tx,
	linked *AdmissionSession,
	userID int64,
	policy *AdmissionPolicy,
	now time.Time,
) (*AdmissionSession, error) {
	if linked == nil || policy == nil {
		return linked, ErrAdmissionInvalidInput
	}
	credential, err := s.repo.GetLatestCredentialForUserSchoolTx(ctx, tx, userID, policy.SchoolID, now)
	if err != nil {
		return nil, err
	}
	if credential == nil {
		return linked, s.queueInitialBotActionsTx(ctx, tx, linked, now)
	}
	verified, err := s.repo.MarkVerifiedTx(ctx, tx, linked.ID, now)
	if err != nil {
		return nil, err
	}
	return verified, s.queueBotActionTx(ctx, tx, verified, BotActionRelease, now, now)
}

func (s *Service) validateResendableSession(session *AdmissionSession) error {
	if session == nil {
		return ErrAdmissionSessionNotFound
	}
	now := s.now()
	switch session.Status {
	case StatusJoinedMuted:
		if now.After(session.TokenExpiresAt) || now.After(session.LinkWaitDeadlineAt) {
			return ErrAdmissionTokenExpired
		}
		return nil
	case StatusLinked:
		if now.After(session.SubmissionWaitDeadlineAt) {
			return ErrAdmissionInvalidStatus
		}
		return nil
	case StatusMaterialSubmitted:
		if session.ManualReviewDeadlineAt != nil && now.After(*session.ManualReviewDeadlineAt) {
			return ErrAdmissionInvalidStatus
		}
		return nil
	default:
		return ErrAdmissionInvalidStatus
	}
}

func (s *Service) ensureRegeneratableSession(ctx context.Context, subject BotSessionSubjectInput) error {
	session, err := s.repo.GetLatestSessionBySubject(ctx, subject)
	if err != nil {
		if errors.Is(err, ErrAdmissionSessionNotFound) {
			return nil
		}
		return err
	}
	if session.Status == StatusVerified {
		return ErrAdmissionInvalidStatus
	}
	return nil
}

func (s *Service) regenerateAdmissionSession(
	ctx context.Context,
	input BotSessionCreateInput,
	nextReminderAt *time.Time,
) (*CreatedAdmissionSession, []string, error) {
	input = normalizeBotSessionCreateInput(input)
	if err := validateBotSessionCreateInput(input); err != nil {
		return nil, nil, err
	}
	subject := botSessionCreateSubject(input)
	if err := s.ensureBotSessionMemberAllowed(ctx, input); err != nil {
		return nil, nil, err
	}
	if err := s.ensureRegeneratableSession(ctx, subject); err != nil {
		return nil, nil, err
	}
	policy, err := s.loadPolicy(ctx, input.Platform, input.GuildID)
	if err != nil {
		return nil, nil, err
	}
	verifiedUserID, err := s.repo.GetVerifiedAdmissionUserByQQ(ctx, input.QQID, policy.SchoolID, s.now())
	if err != nil {
		return nil, nil, err
	}
	if verifiedUserID != nil {
		return s.createVerifiedBotSessionWithCancelledIDs(ctx, input, policy, *verifiedUserID)
	}
	for attempt := 0; attempt < maxAdmissionJoinTokenCreateAttempts; attempt++ {
		token, err := s.generateJoinToken()
		if err != nil {
			return nil, nil, fmt.Errorf("RegenerateBotAdmissionSession token: %w", err)
		}
		session, err := s.newAdmissionSession(input, policy, token)
		if err != nil {
			return nil, nil, err
		}
		if nextReminderAt != nil {
			session.nextReminderAt = nextReminderAt
		}
		var cancelledIDs []string
		if err := s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
			cancelledIDs, err = s.repo.CancelInProgressSessionsBySubjectTx(ctx, tx, subject, s.now())
			if err != nil {
				return err
			}
			if err := s.repo.CreateSessionTx(ctx, tx, session); err != nil {
				return err
			}
			return s.queueInitialBotActionsTx(ctx, tx, session, s.now())
		}); err != nil {
			if isAdmissionSessionTokenHashUniqueViolation(err) {
				continue
			}
			return nil, nil, fmt.Errorf("RegenerateBotAdmissionSession: %w", err)
		}
		if err := s.attachAdmissionFailureContext(ctx, session, policy); err != nil {
			return nil, nil, err
		}
		return &CreatedAdmissionSession{
			Session: session,
			Token:   token,
			AuthURL: session.AuthURL,
		}, cancelledIDs, nil
	}
	return nil, nil, fmt.Errorf("RegenerateBotAdmissionSession token: exhausted %d token attempts", maxAdmissionJoinTokenCreateAttempts)
}

func validateSkippableBotSession(session *AdmissionSession) error {
	if session == nil {
		return ErrAdmissionSessionNotFound
	}
	switch session.Status {
	case StatusJoinedMuted, StatusLinked, StatusMaterialSubmitted:
		return nil
	default:
		return ErrAdmissionInvalidStatus
	}
}

func (s *Service) getAdminActionSession(ctx context.Context, sessionID string) (*AdmissionSession, error) {
	session, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, ErrAdmissionTokenNotFound) {
			return nil, ErrAdmissionSessionNotFound
		}
		return nil, err
	}
	return session, nil
}

func (s *Service) auditBotSessionRegenerated(
	ctx context.Context,
	session *AdmissionSession,
	cancelledIDs []string,
) {
	if session == nil {
		return
	}
	audit.LogContext(ctx, audit.Event{
		Type:         audit.EventDataUpdate,
		Category:     "admission",
		ActorType:    "bot",
		Resource:     "group_admission_sessions",
		ResourceType: "group_admission_session",
		ResourceID:   session.ID,
		Action:       "regenerate",
		Result:       "success",
		Details: map[string]any{
			"platform":              session.Platform,
			"guildID":               session.GuildID,
			"qqID":                  session.QQID,
			"botSelfID":             session.BotSelfID,
			"cancelledSessionIDs":   cancelledIDs,
			"cancelledSessionCount": len(cancelledIDs),
		},
		Timestamp: s.now(),
	})
}

func (s *Service) auditBotSessionSkipped(
	ctx context.Context,
	session *AdmissionSession,
	operatorQQID string,
) {
	if session == nil {
		return
	}
	audit.LogContext(ctx, audit.Event{
		Type:         audit.EventDataUpdate,
		Category:     "admission",
		ActorType:    "qq_operator",
		UserID:       operatorQQID,
		Resource:     "group_admission_sessions",
		ResourceType: "group_admission_session",
		ResourceID:   session.ID,
		Action:       "skip_group_verification",
		Result:       "success",
		Details: map[string]any{
			"platform":      session.Platform,
			"guildID":       session.GuildID,
			"qqID":          session.QQID,
			"botSelfID":     session.BotSelfID,
			"operatorQQID":  operatorQQID,
			"studentStatus": "unchanged",
		},
		Timestamp: s.now(),
	})
}

func (s *Service) auditBotAdmissionFailureCountReset(
	ctx context.Context,
	result *AdmissionFailureResetResult,
	operatorQQID string,
) {
	if result == nil {
		return
	}
	audit.LogContext(ctx, audit.Event{
		Type:         audit.EventDataUpdate,
		Category:     "admission",
		ActorType:    "qq_operator",
		UserID:       operatorQQID,
		Resource:     "group_admission_failures",
		ResourceType: "group_admission_failure",
		ResourceID:   fmt.Sprintf("%s:%s:%s", result.Platform, result.GuildID, result.QQID),
		Action:       "reset_failure_count",
		Result:       "success",
		Details: map[string]any{
			"platform":             result.Platform,
			"guildID":              result.GuildID,
			"qqID":                 result.QQID,
			"operatorQQID":         operatorQQID,
			"previousFailureCount": result.PreviousFailureCount,
		},
		Timestamp: s.now(),
	})
}

func (s *Service) auditAdminSessionAction(
	ctx context.Context,
	action string,
	session *AdmissionSession,
	operatorUserID int64,
	extra map[string]any,
) {
	if session == nil {
		return
	}
	details := map[string]any{
		"platform":       session.Platform,
		"guildID":        session.GuildID,
		"qqID":           session.QQID,
		"botSelfID":      session.BotSelfID,
		"operatorUserID": operatorUserID,
	}
	for key, value := range extra {
		details[key] = value
	}
	audit.LogContext(ctx, audit.Event{
		Type:         audit.EventDataUpdate,
		Category:     "admission",
		ActorType:    "admin",
		UserID:       fmt.Sprintf("%d", operatorUserID),
		Resource:     "group_admission_sessions",
		ResourceType: "group_admission_session",
		ResourceID:   session.ID,
		Action:       action,
		Result:       "success",
		Details:      details,
		Timestamp:    s.now(),
	})
}

func sessionLinkedToUser(session *AdmissionSession, userID int64) bool {
	if session == nil || session.UserID == nil || *session.UserID != userID {
		return false
	}
	switch session.Status {
	case StatusLinked, StatusMaterialSubmitted, StatusVerified:
		return true
	default:
		return false
	}
}

func (s *Service) ensureConsumedSessionCanResume(session *AdmissionSession) error {
	if session == nil {
		return ErrAdmissionSessionNotFound
	}
	switch session.Status {
	case StatusLinked:
		return s.ensureLinkedSessionAcceptsSubmission(session)
	case StatusMaterialSubmitted:
		if session.ManualReviewDeadlineAt != nil && s.now().After(*session.ManualReviewDeadlineAt) {
			return ErrAdmissionTokenExpired
		}
		return nil
	case StatusVerified:
		return nil
	default:
		return ErrAdmissionTokenExpired
	}
}

func (s *Service) MarkMaterialSubmitted(ctx context.Context, sessionID string) (*AdmissionSession, error) {
	session, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session.Status != StatusLinked {
		return nil, ErrAdmissionInvalidStatus
	}
	if err := s.ensureLinkedSessionAcceptsSubmission(session); err != nil {
		return nil, err
	}
	policy, err := s.loadPolicy(ctx, session.Platform, session.GuildID)
	if err != nil {
		return nil, err
	}
	deadline := s.now().Add(time.Duration(policy.ManualReviewTimeoutSeconds) * time.Second)
	var submitted *AdmissionSession
	if err := s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		var err error
		submitted, err = s.repo.MarkMaterialSubmittedTx(ctx, tx, session.ID, deadline)
		if err != nil {
			return err
		}
		return s.queueInitialBotActionsTx(ctx, tx, submitted, s.now())
	}); err != nil {
		return nil, err
	}
	return submitted, nil
}

func (s *Service) MarkVerified(ctx context.Context, sessionID string) (*AdmissionSession, error) {
	session, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session.Status != StatusLinked && session.Status != StatusMaterialSubmitted {
		return nil, ErrAdmissionInvalidStatus
	}
	if err := s.ensureSessionAcceptsVerification(session); err != nil {
		return nil, err
	}
	var verified *AdmissionSession
	if err := s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		var err error
		verified, err = s.repo.MarkVerifiedTx(ctx, tx, session.ID, s.now())
		if err != nil {
			return err
		}
		return s.queueBotActionTx(ctx, tx, verified, BotActionRelease, s.now(), s.now())
	}); err != nil {
		return nil, err
	}
	return verified, nil
}

func (s *Service) ProjectStudentVerification(ctx context.Context, userID int64, schoolID int64, approved bool) error {
	if userID <= 0 || schoolID <= 0 || !approved {
		return nil
	}
	return s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.MarkUserLinkedSessionsVerifiedTx(ctx, tx, userID, schoolID, s.now()); err != nil {
			return err
		}
		return s.queueVerifiedUserReleaseActionsTx(ctx, tx, userID, schoolID, s.now())
	})
}

func (s *Service) RecordBotEvent(ctx context.Context, sessionID string, event BotEventInput) error {
	event = normalizeBotEventInput(event)
	return s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		session, err := s.repo.GetSessionByIDForUpdate(ctx, tx, sessionID)
		if err != nil {
			return err
		}
		if !event.Success {
			return s.repo.UpdateLastBotErrorTx(ctx, updateLastBotErrorTxInput{
				Tx:        tx,
				SessionID: sessionID,
				BotError:  normalizeBotEventError(event),
			})
		}
		if event.Action == BotActionRelease {
			return s.applySuccessfulBotEventTx(ctx, successfulBotEventTxInput{
				Tx:      tx,
				Session: session,
				Action:  event.Action,
			})
		}
		policy, err := s.loadPolicy(ctx, session.Platform, session.GuildID)
		if err != nil {
			return err
		}
		return s.applySuccessfulBotEventTx(ctx, successfulBotEventTxInput{
			Tx:      tx,
			Session: session,
			Policy:  policy,
			Action:  event.Action,
		})
	})
}

func (s *Service) RecordBotActionEvent(ctx context.Context, actionID string, event BotEventInput) error {
	event = normalizeBotEventInput(event)
	botActionID, err := strconv.ParseInt(strings.TrimSpace(actionID), 10, 64)
	if err != nil || botActionID <= 0 {
		return ErrAdmissionInvalidInput
	}
	return s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		action, err := s.repo.GetBotActionForUpdateTx(ctx, tx, botActionID)
		if err != nil {
			return err
		}
		if event.Action == "" {
			event.Action = action.Action
		}
		if event.Action != action.Action && (action.Action != BotActionKick || event.Action != BotActionBlacklist) {
			return ErrAdmissionInvalidStatus
		}
		session := &action.Session
		if !event.Success {
			if err := s.repo.UpdateLastBotErrorTx(ctx, updateLastBotErrorTxInput{
				Tx:        tx,
				SessionID: session.ID,
				BotError:  normalizeBotEventError(event),
			}); err != nil {
				return err
			}
			return s.repo.MarkBotActionFailedTx(ctx, tx, botActionID, event, s.now(), action.AttemptCount)
		}
		if event.Action == BotActionRelease {
			if err := s.applySuccessfulBotEventTx(ctx, successfulBotEventTxInput{
				Tx:      tx,
				Session: session,
				Action:  event.Action,
			}); err != nil {
				return err
			}
			return s.repo.MarkBotActionSucceededTx(ctx, tx, botActionID, event, s.now())
		}
		policy, err := s.loadPolicy(ctx, session.Platform, session.GuildID)
		if err != nil {
			return err
		}
		if err := s.applySuccessfulBotEventTx(ctx, successfulBotEventTxInput{
			Tx:      tx,
			Session: session,
			Policy:  policy,
			Action:  event.Action,
		}); err != nil {
			return err
		}
		return s.repo.MarkBotActionSucceededTx(ctx, tx, botActionID, event, s.now())
	})
}

func (s *Service) applySuccessfulBotEventTx(ctx context.Context, input successfulBotEventTxInput) error {
	switch input.Action {
	case BotActionRemind:
		if err := s.repo.MarkReminderSentTx(ctx, markReminderSentTxInput{
			Tx:      input.Tx,
			Session: input.Session,
			Policy:  input.Policy,
			Now:     s.now(),
		}); err != nil {
			return err
		}
		return s.queueNextReminderTx(ctx, input.Tx, input.Session, input.Policy, s.now())
	case BotActionRelease:
		if err := s.repo.MarkBotReleaseCompletedTx(ctx, markBotSessionTxInput{
			Tx:        input.Tx,
			SessionID: input.Session.ID,
			Now:       s.now(),
		}); err != nil {
			return err
		}
		_, err := s.repo.ResetAdmissionFailureCountTx(
			ctx,
			input.Tx,
			input.Session.Platform,
			input.Session.GuildID,
			input.Session.QQID,
			s.now(),
		)
		return err
	case BotActionKick, BotActionBlacklist:
		updated, err := s.repo.MarkBotKickCompletedTx(ctx, markBotSessionTxInput{
			Tx:        input.Tx,
			SessionID: input.Session.ID,
			Now:       s.now(),
		})
		if err != nil {
			return err
		}
		if !updated {
			return nil
		}
		return s.incrementFailureFromKickEventTx(ctx, input)
	default:
		return ErrAdmissionInvalidStatus
	}
}

func (s *Service) incrementFailureFromKickEventTx(ctx context.Context, input successfulBotEventTxInput) error {
	count, err := s.repo.IncrementFailureFromKickEventTx(ctx, admissionFailureIncrementTxInput{
		Tx:      input.Tx,
		Session: input.Session,
		Now:     s.now(),
	})
	if err != nil {
		return err
	}
	if count < input.Policy.FailedJoinLimit {
		return nil
	}
	now := s.now()
	_, err = s.createAdmissionFailureBlacklistTx(ctx, input.Tx, admissionFailureBlacklistInput(input, count, now))
	return err
}

type successfulBotEventTxInput struct {
	Tx      pgx.Tx
	Session *AdmissionSession
	Policy  *AdmissionPolicy
	Action  BotAction
}

func admissionFailureBlacklistInput(
	input successfulBotEventTxInput,
	failureCount int,
	now time.Time,
) MemberBlacklistCreateInput {
	return MemberBlacklistCreateInput{
		Platform: input.Session.Platform, SubjectType: BlacklistSubjectQQUser, SubjectID: input.Session.QQID,
		ScopeType: BlacklistScopeGuild, GuildID: &input.Session.GuildID, Source: BlacklistSourceAdmissionFailure,
		ReasonCode: BlacklistReasonAdmissionTimeoutLimit, ReasonText: "admission failure limit reached",
		CreatedByType: BlacklistActorSystem, CreatedByID: "system",
		CreatedFrom: BlacklistCreatedFromAdmissionWorker, ExpiresAt: admissionFailureBlacklistExpiresAt(input, now),
		Metadata: admissionFailureBlacklistMetadata(input, failureCount),
	}
}

func admissionFailureBlacklistExpiresAt(input successfulBotEventTxInput, now time.Time) *time.Time {
	if input.Policy.BlacklistDurationSeconds == nil {
		return nil
	}
	expiresAt := now.Add(time.Duration(*input.Policy.BlacklistDurationSeconds) * time.Second)
	return &expiresAt
}

func admissionFailureBlacklistMetadata(
	input successfulBotEventTxInput,
	failureCount int,
) map[string]any {
	return map[string]any{
		"admissionSessionID": input.Session.ID,
		"failureCount":       failureCount,
		"failedJoinLimit":    input.Policy.FailedJoinLimit,
		"platform":           input.Session.Platform,
		"guildID":            input.Session.GuildID,
		"botSelfID":          input.Session.BotSelfID,
	}
}

func (s *Service) newAdmissionSession(
	input BotSessionCreateInput,
	policy *AdmissionPolicy,
	token string,
) (*AdmissionSession, error) {
	sessionID, err := id.New()
	if err != nil {
		return nil, fmt.Errorf("newAdmissionSession id: %w", err)
	}
	now := s.now()
	nextReminderAt := initialReminderNotBefore(now, policy)
	return &AdmissionSession{
		ID:                       sessionID,
		Platform:                 input.Platform,
		BotSelfID:                input.BotSelfID,
		GuildID:                  input.GuildID,
		ChannelID:                input.ChannelID,
		QQID:                     input.QQID,
		TokenHash:                s.hashToken(token),
		AuthURL:                  s.buildAuthURL(token),
		TokenExpiresAt:           now.Add(time.Duration(policy.LinkWaitSeconds) * time.Second),
		Status:                   StatusJoinedMuted,
		LinkWaitDeadlineAt:       now.Add(time.Duration(policy.LinkWaitSeconds) * time.Second),
		SubmissionWaitDeadlineAt: now.Add(time.Duration(policy.SubmissionWaitSeconds) * time.Second),
		InitialMuteUntil:         now.Add(time.Duration(policy.InitialMuteDurationSeconds) * time.Second),
		nextReminderAt:           &nextReminderAt,
	}, nil
}

func (s *Service) newVerifiedAdmissionSession(
	input BotSessionCreateInput,
	policy *AdmissionPolicy,
	token string,
	userID int64,
) (*AdmissionSession, error) {
	session, err := s.newAdmissionSession(input, policy, token)
	if err != nil {
		return nil, err
	}
	now := s.now()
	session.UserID = &userID
	session.TokenConsumedAt = &now
	session.Status = StatusVerified
	session.VerifiedAt = &now
	session.nextReminderAt = nil
	return session, nil
}

func initialReminderNotBefore(now time.Time, policy *AdmissionPolicy) time.Time {
	grace := initialReminderGrace
	if policy != nil && policy.ReminderIntervalSeconds > 0 {
		interval := time.Duration(policy.ReminderIntervalSeconds) * time.Second
		if interval < grace {
			grace = interval
		}
	}
	return now.Add(grace)
}

func (s *Service) attachAdmissionFailureContext(
	ctx context.Context,
	session *AdmissionSession,
	policy *AdmissionPolicy,
) error {
	if session == nil || policy == nil {
		return nil
	}
	failure, err := s.repo.GetAdmissionFailure(ctx, session.Platform, session.GuildID, session.QQID)
	if err != nil {
		return err
	}
	applyAdmissionFailureContext(session, failure, policy)
	return nil
}

func applyAdmissionFailureContext(
	session *AdmissionSession,
	failure *AdmissionFailure,
	policy *AdmissionPolicy,
) {
	if session == nil || policy == nil {
		return
	}
	count := 0
	if failure != nil && failure.FailureCount > 0 {
		count = failure.FailureCount
	}
	remaining := policy.FailedJoinLimit - count - 1
	if remaining < 0 {
		remaining = 0
	}
	session.FailureCount = count
	session.RemainingRetryCount = remaining
	session.WillBlacklistOnTimeout = count+1 >= policy.FailedJoinLimit
}

func (s *Service) validateTokenSession(session *AdmissionSession, qqQuery string) error {
	if session == nil {
		return ErrAdmissionTokenNotFound
	}
	if query := strings.TrimSpace(qqQuery); query != "" {
		return ErrAdmissionQQMismatch
	}
	if session.Status == StatusCancelled || session.Status == StatusExpiredKicked {
		return ErrAdmissionTokenExpired
	}
	if session.TokenConsumedAt != nil || session.Status != StatusJoinedMuted {
		return ErrAdmissionTokenConsumed
	}
	if s.now().After(session.TokenExpiresAt) {
		return ErrAdmissionTokenExpired
	}
	return nil
}

func (s *Service) loadPolicy(ctx context.Context, platform, guildID string) (*AdmissionPolicy, error) {
	policy, err := s.repo.GetPolicy(ctx, platform, guildID)
	if err != nil {
		return nil, err
	}
	if policy != nil {
		return policy, nil
	}
	return nil, ErrAdmissionPolicyNotFound
}

func (s *Service) hashToken(token string) string {
	mac := hmac.New(sha256.New, s.hmacKey)
	mac.Write([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(mac.Sum(nil))
}

func (s *Service) buildAuthURL(token string) string {
	return s.authBaseURL + url.PathEscape(token)
}

func admissionTokenFromAuthURL(authURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(authURL))
	if err != nil {
		return ""
	}
	segments := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(segments) < 2 || segments[len(segments)-2] != "verify" {
		return ""
	}
	token, err := url.PathUnescape(segments[len(segments)-1])
	if err != nil {
		return ""
	}
	return token
}

func normalizeBotSessionCreateInput(input BotSessionCreateInput) BotSessionCreateInput {
	input.Platform = strings.TrimSpace(input.Platform)
	input.GuildID = strings.TrimSpace(input.GuildID)
	input.ChannelID = strings.TrimSpace(input.ChannelID)
	input.QQID = strings.TrimSpace(input.QQID)
	input.BotSelfID = strings.TrimSpace(input.BotSelfID)
	return input
}

func normalizeBotSessionSubjectInput(input BotSessionSubjectInput) BotSessionSubjectInput {
	input.Platform = strings.TrimSpace(input.Platform)
	input.GuildID = strings.TrimSpace(input.GuildID)
	input.QQID = strings.TrimSpace(input.QQID)
	return input
}

func normalizeBotSessionOperatorInput(input BotSessionOperatorInput) BotSessionOperatorInput {
	input.Platform = strings.TrimSpace(input.Platform)
	input.GuildID = strings.TrimSpace(input.GuildID)
	input.QQID = strings.TrimSpace(input.QQID)
	input.OperatorQQID = strings.TrimSpace(input.OperatorQQID)
	return input
}

func validateBotSessionCreateInput(input BotSessionCreateInput) error {
	if input.Platform == "" || input.GuildID == "" || input.ChannelID == "" || input.QQID == "" || input.BotSelfID == "" {
		return ErrAdmissionInvalidInput
	}
	if isAdmissionBotFieldTooLong(input.Platform, maxAdmissionBotPlatformRunes) ||
		isAdmissionBotFieldTooLong(input.GuildID, maxAdmissionBotGuildIDRunes) ||
		isAdmissionBotFieldTooLong(input.ChannelID, maxAdmissionBotChannelIDRunes) ||
		isAdmissionBotFieldTooLong(input.QQID, maxAdmissionBotQQIDRunes) ||
		isAdmissionBotFieldTooLong(input.BotSelfID, maxAdmissionBotSelfIDRunes) {
		return ErrAdmissionInvalidInput
	}
	return nil
}

func validateBotSessionSubjectInput(input BotSessionSubjectInput) error {
	if input.Platform == "" || input.GuildID == "" || input.QQID == "" {
		return ErrAdmissionInvalidInput
	}
	if isAdmissionBotFieldTooLong(input.Platform, maxAdmissionBotPlatformRunes) ||
		isAdmissionBotFieldTooLong(input.GuildID, maxAdmissionBotGuildIDRunes) ||
		isAdmissionBotFieldTooLong(input.QQID, maxAdmissionBotQQIDRunes) {
		return ErrAdmissionInvalidInput
	}
	return nil
}

func validateBotSessionOperatorInput(input BotSessionOperatorInput) error {
	if input.Platform == "" || input.GuildID == "" || input.QQID == "" || input.OperatorQQID == "" {
		return ErrAdmissionInvalidInput
	}
	if isAdmissionBotFieldTooLong(input.Platform, maxAdmissionBotPlatformRunes) ||
		isAdmissionBotFieldTooLong(input.GuildID, maxAdmissionBotGuildIDRunes) ||
		isAdmissionBotFieldTooLong(input.QQID, maxAdmissionBotQQIDRunes) ||
		isAdmissionBotFieldTooLong(input.OperatorQQID, maxAdmissionBotQQIDRunes) {
		return ErrAdmissionInvalidInput
	}
	return nil
}

func isAdmissionBotFieldTooLong(value string, maxRunes int) bool {
	return utf8.RuneCountInString(value) > maxRunes
}

func botSessionCreateSubject(input BotSessionCreateInput) BotSessionSubjectInput {
	return BotSessionSubjectInput{
		Platform: input.Platform,
		GuildID:  input.GuildID,
		QQID:     input.QQID,
	}
}

func botSessionOperatorSubject(input BotSessionOperatorInput) BotSessionSubjectInput {
	return BotSessionSubjectInput{
		Platform: input.Platform,
		GuildID:  input.GuildID,
		QQID:     input.QQID,
	}
}

func normalizeStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func normalizeBotEventInput(event BotEventInput) BotEventInput {
	return BotEventInput{
		Action:    BotAction(strings.TrimSpace(string(event.Action))),
		Success:   event.Success,
		MessageID: strings.TrimSpace(event.MessageID),
		Error:     strings.TrimSpace(event.Error),
	}
}

func normalizeBotEventError(event BotEventInput) string {
	trimmed := strings.TrimSpace(event.Error)
	if trimmed != "" {
		return trimmed
	}
	return "bot action failed"
}
