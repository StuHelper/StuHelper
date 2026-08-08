package studentverification

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	appcrypto "github.com/StuHelper/StuHelper/server/internal/pkg/crypto"
	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
)

const (
	manualEmailOTPChallengePrefix = "student-verification:manual-email-otp:challenge:"
	manualEmailOTPAttemptsPrefix  = "student-verification:manual-email-otp:attempts:"
	manualEmailOTPCooldownPrefix  = "student-verification:manual-email-otp:cooldown:"
	manualEmailOTPHashDomain      = "student-verification-manual-email-otp:v1:"
	manualEmailOTPTTL             = 10 * time.Minute
	manualEmailOTPCooldown        = time.Minute
	manualEmailOTPMaxAttempts     = 5
)

type storedManualEmailOTPChallenge struct {
	CaseID            string    `json:"caseId"`
	ApplicationID     string    `json:"applicationId"`
	UserID            int64     `json:"userId"`
	EmailHash         string    `json:"emailHash"`
	CodeHash          string    `json:"codeHash"`
	CreatedAt         time.Time `json:"createdAt"`
	ExpiresAt         time.Time `json:"expiresAt"`
	ResendAvailableAt time.Time `json:"resendAvailableAt"`
}

func (s *Service) RequestManualReviewEmailOTP(
	ctx context.Context,
	userID int64,
	applicationID string,
) (*ManualEmailOTPChallenge, error) {
	if s.redisClient == nil || s.emailSender == nil {
		return nil, ErrDependencyUnavailable
	}
	reviewCase, config, _, err := s.loadManualReviewMaterialContext(ctx, userID, applicationID)
	if err != nil {
		return nil, err
	}
	if reviewCase.EmailHash == nil {
		return nil, ErrManualReviewInvalidForm
	}
	formValues, err := s.decryptManualReviewForm(reviewCase)
	if err != nil {
		return nil, err
	}
	email := manualReviewEmailFromForm(formValues)
	emailHash, err := appcrypto.HMACHashWithKey(
		manualEmailHashDomain+config.SchoolCode+":"+email, s.hmacKey,
	)
	if err != nil || !constantTimeStringEqual(emailHash, *reviewCase.EmailHash) {
		return nil, ErrDependencyUnavailable
	}
	reserved, err := s.redisClient.SetNX(
		ctx, manualEmailOTPCooldownPrefix+reviewCase.ID, "1", manualEmailOTPCooldown,
	).Result()
	if err != nil {
		return nil, ErrDependencyUnavailable
	}
	if !reserved {
		return nil, ErrManualEmailOTPCooldown
	}
	code, err := s.generateOTP()
	if err != nil {
		return nil, err
	}
	codeHash, err := appcrypto.HMACHashWithKey(
		manualEmailOTPHashDomain+reviewCase.ID+":"+code, s.hmacKey,
	)
	if err != nil {
		return nil, err
	}
	now := s.now()
	challenge := storedManualEmailOTPChallenge{
		CaseID: reviewCase.ID, ApplicationID: applicationID, UserID: userID,
		EmailHash: emailHash, CodeHash: codeHash, CreatedAt: now,
		ExpiresAt:         now.Add(manualEmailOTPTTL),
		ResendAvailableAt: now.Add(manualEmailOTPCooldown),
	}
	encoded, err := json.Marshal(challenge)
	if err != nil {
		return nil, err
	}
	pipe := s.redisClient.TxPipeline()
	pipe.Set(ctx, manualEmailOTPChallengePrefix+reviewCase.ID, encoded, manualEmailOTPTTL)
	pipe.Del(ctx, manualEmailOTPAttemptsPrefix+reviewCase.ID)
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, ErrDependencyUnavailable
	}
	if err := s.emailSender.SendStudentVerificationOTP(ctx, email, code); err != nil {
		cleanupErr := s.redisClient.Del(
			context.WithoutCancel(ctx), manualEmailOTPChallengePrefix+reviewCase.ID,
		).Err()
		return nil, errors.Join(
			fmt.Errorf("%w: send manual review email otp", ErrDependencyUnavailable),
			cleanupErr,
		)
	}
	return &ManualEmailOTPChallenge{
		CaseID: reviewCase.ID, MaskedEmail: maskEmail(email),
		ExpiresAt: challenge.ExpiresAt, ResendAvailableAt: challenge.ResendAvailableAt,
		RemainingAttempts: manualEmailOTPMaxAttempts,
	}, nil
}

func (s *Service) VerifyManualReviewEmailOTP(
	ctx context.Context,
	userID int64,
	applicationID string,
	code string,
) (*ManualReviewCase, error) {
	if s.redisClient == nil {
		return nil, ErrDependencyUnavailable
	}
	code = strings.TrimSpace(code)
	if len(code) < 4 || len(code) > 10 || strings.IndexFunc(code, func(r rune) bool {
		return r < '0' || r > '9'
	}) >= 0 {
		return nil, ErrManualEmailOTPInvalid
	}
	reviewCase, err := s.repo.GetManualReviewCaseForUser(ctx, applicationID, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrManualReviewNotFound
	}
	if err != nil {
		return nil, err
	}
	encoded, err := s.redisClient.Get(
		ctx, manualEmailOTPChallengePrefix+reviewCase.ID,
	).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrManualEmailOTPExpired
	}
	if err != nil {
		return nil, ErrDependencyUnavailable
	}
	var challenge storedManualEmailOTPChallenge
	if json.Unmarshal(encoded, &challenge) != nil || challenge.CaseID != reviewCase.ID ||
		challenge.ApplicationID != applicationID || challenge.UserID != userID ||
		!challenge.ExpiresAt.After(s.now()) {
		return nil, errors.Join(
			ErrManualEmailOTPExpired,
			s.deleteManualEmailOTPChallenge(context.WithoutCancel(ctx), reviewCase.ID),
		)
	}
	wantHash, err := appcrypto.HMACHashWithKey(
		manualEmailOTPHashDomain+reviewCase.ID+":"+code, s.hmacKey,
	)
	if err != nil {
		return nil, err
	}
	if subtle.ConstantTimeCompare([]byte(challenge.CodeHash), []byte(wantHash)) != 1 {
		attempts, incrementErr := s.redisClient.Incr(
			ctx, manualEmailOTPAttemptsPrefix+reviewCase.ID,
		).Result()
		if incrementErr != nil {
			return nil, ErrDependencyUnavailable
		}
		if expireErr := s.redisClient.Expire(
			ctx, manualEmailOTPAttemptsPrefix+reviewCase.ID, manualEmailOTPTTL,
		).Err(); expireErr != nil {
			return nil, ErrDependencyUnavailable
		}
		if attempts >= manualEmailOTPMaxAttempts {
			return nil, errors.Join(
				ErrManualEmailOTPMaxAttempts,
				s.deleteManualEmailOTPChallenge(context.WithoutCancel(ctx), reviewCase.ID),
			)
		}
		return nil, ErrManualEmailOTPInvalid
	}
	now := s.now()
	err = s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		lockedCase, err := s.repo.GetManualReviewCaseForApplicationUpdateTx(
			ctx, tx, applicationID, userID,
		)
		if err != nil {
			return err
		}
		if lockedCase.EmailHash == nil ||
			!constantTimeStringEqual(*lockedCase.EmailHash, challenge.EmailHash) ||
			(lockedCase.Status != ManualReviewDraft && lockedCase.Status != ManualReviewSupplementRequired) {
			return ErrManualReviewState
		}
		config, err := s.repo.GetMethodConfigTx(
			ctx, tx, lockedCase.SchoolCode, MethodManualMaterialReview,
		)
		if err != nil || !methodHealthy(config) ||
			config.PrivacyNoticeVersion != lockedCase.PrivacyNoticeVersion {
			return ErrMethodUnavailable
		}
		return s.repo.SetManualReviewEmailVerifiedTx(
			ctx, tx, lockedCase.ID, userID, challenge.EmailHash, now,
		)
	})
	if err != nil {
		return nil, err
	}
	if cleanupErr := s.deleteManualEmailOTPChallenge(
		context.WithoutCancel(ctx), reviewCase.ID,
	); cleanupErr != nil {
		logger.FromContext(ctx).Warn("failed to remove consumed manual review email challenge", zap.Error(cleanupErr))
	}
	return s.GetManualReview(ctx, userID, applicationID)
}

func (s *Service) deleteManualEmailOTPChallenge(ctx context.Context, caseID string) error {
	return s.redisClient.Del(
		ctx,
		manualEmailOTPChallengePrefix+caseID,
		manualEmailOTPAttemptsPrefix+caseID,
	).Err()
}
