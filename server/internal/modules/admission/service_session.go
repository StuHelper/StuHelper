package admission

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/id"
)

func (s *Service) CreateBotSession(ctx context.Context, input BotSessionCreateInput) (*CreatedAdmissionSession, error) {
	input = normalizeBotSessionCreateInput(input)
	policy, err := s.loadPolicy(ctx, input.Platform, input.GuildID)
	if err != nil {
		return nil, err
	}
	token, err := s.generateToken()
	if err != nil {
		return nil, fmt.Errorf("CreateBotSession token: %w", err)
	}
	session, err := s.newAdmissionSession(input, policy, token)
	if err != nil {
		return nil, err
	}
	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, err
	}
	return &CreatedAdmissionSession{
		Session: session,
		Token:   token,
		AuthURL: session.AuthURL,
	}, nil
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
			return err
		}
		policy, err := s.loadPolicy(ctx, session.Platform, session.GuildID)
		if err != nil {
			return err
		}
		if _, err := s.qqGateway.EnsureQQBindingForUserTx(ctx, tx, input.UserID, session.QQID, session.QQNickname); err != nil {
			return err
		}
		deadline := s.now().Add(time.Duration(policy.SubmissionWaitSeconds) * time.Second)
		linked, err = s.repo.MarkTokenConsumedAndLinked(ctx, tx, session.ID, input.UserID, s.now(), deadline)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("LinkTokenToUser: %w", err)
	}
	return linked, nil
}

func (s *Service) MarkMaterialSubmitted(ctx context.Context, sessionID string) (*AdmissionSession, error) {
	session, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session.Status != StatusLinked {
		return nil, ErrAdmissionInvalidStatus
	}
	policy, err := s.loadPolicy(ctx, session.Platform, session.GuildID)
	if err != nil {
		return nil, err
	}
	deadline := s.now().Add(time.Duration(policy.ManualReviewTimeoutSeconds) * time.Second)
	return s.repo.MarkMaterialSubmitted(ctx, session.ID, deadline)
}

func (s *Service) MarkVerified(ctx context.Context, sessionID string) (*AdmissionSession, error) {
	session, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session.Status != StatusLinked && session.Status != StatusMaterialSubmitted {
		return nil, ErrAdmissionInvalidStatus
	}
	return s.repo.MarkVerified(ctx, session.ID, s.now())
}

func (s *Service) RecordBotEvent(ctx context.Context, sessionID string, event BotEventInput) error {
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

func (s *Service) applySuccessfulBotEventTx(ctx context.Context, input successfulBotEventTxInput) error {
	switch input.Action {
	case BotActionRemind:
		return s.repo.MarkReminderSentTx(ctx, markReminderSentTxInput{
			Tx:      input.Tx,
			Session: input.Session,
			Policy:  input.Policy,
			Now:     s.now(),
		})
	case BotActionRelease:
		return s.repo.MarkBotReleaseCompletedTx(ctx, markBotSessionTxInput{
			Tx:        input.Tx,
			SessionID: input.Session.ID,
			Now:       s.now(),
		})
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
		Policy:  input.Policy,
		Now:     s.now(),
	})
	if err != nil {
		return err
	}
	if count < input.Policy.FailedJoinLimit {
		return nil
	}
	now := s.now()
	_, err = s.repo.CreateMemberBlacklistTx(ctx, input.Tx, MemberBlacklistCreateTxInput{
		Entry: admissionFailureBlacklistInput(input, count, now),
		Now:   now,
	})
	return err
}

type successfulBotEventTxInput struct {
	Tx      pgx.Tx
	Session *AdmissionSession
	Policy  *AdmissionPolicy
	Action  BotAction
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
	return &AdmissionSession{
		ID:                       sessionID,
		Platform:                 input.Platform,
		BotSelfID:                input.BotSelfID,
		GuildID:                  input.GuildID,
		ChannelID:                input.ChannelID,
		QQID:                     input.QQID,
		QQNickname:               input.QQNickname,
		TokenHash:                s.hashToken(token),
		AuthURL:                  s.buildAuthURL(token, input.QQID),
		TokenExpiresAt:           now.Add(time.Duration(policy.LinkWaitSeconds) * time.Second),
		Status:                   StatusJoinedMuted,
		LinkWaitDeadlineAt:       now.Add(time.Duration(policy.LinkWaitSeconds) * time.Second),
		SubmissionWaitDeadlineAt: now.Add(time.Duration(policy.SubmissionWaitSeconds) * time.Second),
		InitialMuteUntil:         now.Add(time.Duration(policy.InitialMuteDurationSeconds) * time.Second),
	}, nil
}

func (s *Service) validateTokenSession(session *AdmissionSession, qqQuery string) error {
	if session == nil {
		return ErrAdmissionTokenNotFound
	}
	if query := strings.TrimSpace(qqQuery); query != "" && query != session.QQID {
		return ErrAdmissionQQMismatch
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
	return defaultAdmissionPolicy(platform, guildID, s.now()), nil
}

func (s *Service) hashToken(token string) string {
	mac := hmac.New(sha256.New, s.hmacKey)
	mac.Write([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(mac.Sum(nil))
}

func (s *Service) buildAuthURL(token string, qqID string) string {
	values := url.Values{}
	values.Set("qq", qqID)
	return s.authBaseURL + url.PathEscape(token) + "?" + values.Encode()
}

func normalizeBotSessionCreateInput(input BotSessionCreateInput) BotSessionCreateInput {
	input.Platform = strings.TrimSpace(input.Platform)
	input.GuildID = strings.TrimSpace(input.GuildID)
	input.ChannelID = strings.TrimSpace(input.ChannelID)
	input.QQID = strings.TrimSpace(input.QQID)
	input.QQNickname = normalizeStringPtr(input.QQNickname)
	input.BotSelfID = strings.TrimSpace(input.BotSelfID)
	return input
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

func normalizeBotEventError(event BotEventInput) string {
	trimmed := strings.TrimSpace(event.Error)
	if trimmed != "" {
		return trimmed
	}
	return "bot action failed"
}
