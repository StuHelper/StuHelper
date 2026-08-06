package studentverification

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	appcrypto "github.com/StuHelper/StuHelper/server/internal/pkg/crypto"
	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
)

const (
	emailOTPCodeLength         = 6
	emailOTPDefaultTTL         = 5 * time.Minute
	emailOTPDefaultCooldown    = time.Minute
	emailOTPDefaultMaxAttempts = 5
	emailOTPChallengeKeyPrefix = "student-verification:email-otp:challenge:"
	emailOTPAttemptsKeyPrefix  = "student-verification:email-otp:attempts:"
	emailOTPCooldownKeyPrefix  = "student-verification:email-otp:cooldown:"
	emailOTPHashDomain         = "student-verification-email-otp:v1:"
)

type emailOTPPolicy struct {
	TTL         time.Duration
	Cooldown    time.Duration
	MaxAttempts int
}

type storedEmailOTPChallenge struct {
	ApplicationID     string    `json:"applicationId"`
	UserID            int64     `json:"userId"`
	SchoolID          int64     `json:"schoolId"`
	Email             string    `json:"email"`
	StudentIDHash     string    `json:"studentIdHash"`
	NameHash          string    `json:"nameHash"`
	SubjectHash       string    `json:"subjectHash"`
	SubjectDisplay    string    `json:"subjectDisplay"`
	CodeHash          string    `json:"codeHash"`
	SnapshotID        string    `json:"snapshotId"`
	SnapshotRevision  int64     `json:"snapshotRevision"`
	PrivacyNotice     string    `json:"privacyNoticeVersion"`
	CreatedAt         time.Time `json:"createdAt"`
	ExpiresAt         time.Time `json:"expiresAt"`
	ResendAvailableAt time.Time `json:"resendAvailableAt"`
}

type preparedStudentEmailIdentity struct {
	Application   *Application
	Config        *MethodConfig
	StudentID     string
	NameHash      string
	StudentIDHash string
	SubjectHash   string
	Record        *RosterRecord
	PreparedAt    time.Time
}

func generateNumericOTP() (string, error) {
	result := make([]byte, emailOTPCodeLength)
	for index := range result {
		value, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", fmt.Errorf("generate student email otp: %w", err)
		}
		result[index] = byte('0' + value.Int64())
	}
	return string(result), nil
}

func (s *Service) RequestStudentEmailOTP(
	ctx context.Context,
	input StudentEmailIdentityInput,
) (*StudentEmailOTPChallenge, error) {
	if s.redisClient == nil || s.emailSender == nil {
		return nil, ErrDependencyUnavailable
	}
	prepared, err := s.prepareStudentEmailIdentity(ctx, input, MethodStudentEmailOutboundOTP)
	if err != nil {
		return nil, err
	}
	application, config := prepared.Application, prepared.Config
	studentID, nameHash := prepared.StudentID, prepared.NameHash
	studentIDHash, subjectHash := prepared.StudentIDHash, prepared.SubjectHash
	record, now := prepared.Record, prepared.PreparedAt
	policy, ok := parseEmailOTPPolicy(config.RiskPolicy)
	if !ok {
		return nil, ErrMethodUnavailable
	}
	domain, ok := buaaEmailDomain(config.EmailDomains)
	if !ok {
		return nil, ErrMethodUnavailable
	}
	targetEmail := studentID + "@" + domain
	if err := s.reserveEmailOTPCooldown(ctx, application.ID, policy.Cooldown); err != nil {
		return nil, err
	}
	code, err := s.generateOTP()
	if err != nil {
		return nil, err
	}
	codeHash, err := appcrypto.HMACHashWithKey(emailOTPHashDomain+application.ID+":"+code, s.hmacKey)
	if err != nil {
		return nil, err
	}
	challenge := storedEmailOTPChallenge{
		ApplicationID: application.ID, UserID: input.UserID, SchoolID: config.SchoolID,
		Email: targetEmail, StudentIDHash: studentIDHash, NameHash: nameHash,
		SubjectHash: subjectHash, SubjectDisplay: maskStudentID(studentID),
		CodeHash: codeHash, SnapshotID: record.SnapshotID,
		SnapshotRevision: record.SnapshotRevision,
		PrivacyNotice:    input.PrivacyNoticeVersion, CreatedAt: now,
		ExpiresAt: now.Add(policy.TTL), ResendAvailableAt: now.Add(policy.Cooldown),
	}
	if err := s.storeEmailOTPChallenge(ctx, challenge, policy.TTL); err != nil {
		return nil, err
	}
	if err := s.emailSender.SendStudentVerificationOTP(ctx, targetEmail, code); err != nil {
		cleanupErr := s.deleteEmailOTPChallenge(context.WithoutCancel(ctx), application.ID)
		return nil, errors.Join(
			fmt.Errorf("%w: send student email otp", ErrDependencyUnavailable),
			cleanupErr,
		)
	}
	return emailOTPChallengeView(challenge, policy.MaxAttempts), nil
}

func (s *Service) VerifyStudentEmailOTP(
	ctx context.Context,
	input VerifyStudentEmailOTPInput,
) (*ApplicationView, error) {
	if s.redisClient == nil {
		return nil, ErrDependencyUnavailable
	}
	code := strings.TrimSpace(input.Code)
	if len(code) < 4 || len(code) > 10 || strings.IndexFunc(code, func(r rune) bool { return r < '0' || r > '9' }) >= 0 {
		return nil, ErrEmailOTPInvalid
	}
	challenge, err := s.loadEmailOTPChallenge(ctx, input.ApplicationID)
	if err != nil {
		return nil, err
	}
	now := s.now()
	if challenge.UserID != input.UserID || !challenge.ExpiresAt.After(now) {
		return nil, errors.Join(
			ErrEmailOTPExpired,
			s.deleteEmailOTPChallenge(context.WithoutCancel(ctx), input.ApplicationID),
		)
	}
	application, err := s.repo.GetApplication(ctx, input.ApplicationID, input.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrApplicationNotFound
	}
	if err != nil {
		return nil, err
	}
	if application.SchoolID != challenge.SchoolID {
		return nil, ErrEmailOTPExpired
	}
	config, err := s.repo.GetMethodConfig(ctx, application.SchoolCode, MethodStudentEmailOutboundOTP)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("load student email method: %w", err)
	}
	if errors.Is(err, pgx.ErrNoRows) || !methodHealthy(config) {
		return nil, ErrMethodUnavailable
	}
	policy, ok := parseEmailOTPPolicy(config.RiskPolicy)
	if !ok {
		return nil, ErrMethodUnavailable
	}
	wantHash, err := appcrypto.HMACHashWithKey(emailOTPHashDomain+input.ApplicationID+":"+code, s.hmacKey)
	if err != nil {
		return nil, err
	}
	if subtle.ConstantTimeCompare([]byte(challenge.CodeHash), []byte(wantHash)) != 1 {
		return nil, s.recordEmailOTPFailure(ctx, input.ApplicationID, policy)
	}

	var outcome error
	err = s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		application, err := s.repo.GetApplicationForUpdateTx(ctx, tx, input.ApplicationID, input.UserID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrApplicationNotFound
		}
		if err != nil {
			return err
		}
		if !applicationIsMutable(application, now) {
			if !application.ExpiresAt.After(now) {
				return ErrApplicationExpired
			}
			return ErrApplicationState
		}
		lockedConfig, err := s.repo.GetMethodConfigTx(ctx, tx, application.SchoolCode, MethodStudentEmailOutboundOTP)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("lock student email method: %w", err)
		}
		if errors.Is(err, pgx.ErrNoRows) || !methodHealthy(lockedConfig) ||
			lockedConfig.AdapterID != BUAAAdapterID || lockedConfig.RosterDependency != "required" ||
			lockedConfig.PrivacyNoticeVersion != challenge.PrivacyNotice {
			outcome = ErrMethodUnavailable
			return s.repo.insertAttemptAndProgressTx(ctx, tx, application, attemptResultFor(
				config, "unavailable", "method_configuration_changed", &challenge.SnapshotID,
				&challenge.SnapshotRevision, challenge.PrivacyNotice, now,
			), now)
		}
		record, err := s.repo.GetActiveRosterRecordTx(ctx, tx, lockedConfig.SchoolID, challenge.StudentIDHash)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("lock email roster record: %w", err)
		}
		if errors.Is(err, pgx.ErrNoRows) || !s.emailRosterRecordMatches(record, lockedConfig, challenge.NameHash, now) {
			outcome = ErrDependencyUnavailable
			return s.repo.insertAttemptAndProgressTx(ctx, tx, application, attemptResultFor(
				lockedConfig, "unavailable", "roster_changed", &challenge.SnapshotID,
				&challenge.SnapshotRevision, challenge.PrivacyNotice, now,
			), now)
		}
		if err := s.repo.LockEnrollmentSubjectTx(ctx, tx, lockedConfig.SchoolID, challenge.SubjectHash); err != nil {
			return err
		}
		subject, err := s.repo.GetActiveEnrollmentSubjectTx(ctx, tx, lockedConfig.SchoolID, challenge.SubjectHash)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			subjectID, idErr := newID()
			if idErr != nil {
				return idErr
			}
			subject = &EnrollmentSubject{
				ID: subjectID, UserID: input.UserID, SchoolID: lockedConfig.SchoolID,
				SubjectHash: challenge.SubjectHash, StudentIDHash: challenge.StudentIDHash,
				StudentDisplay: challenge.SubjectDisplay, BindingStatus: "active",
			}
			if err := s.repo.CreateEnrollmentSubjectTx(
				ctx, tx, *subject, record.DocumentNumberHash, MethodStudentEmailOutboundOTP,
				&record.SnapshotID, &record.SnapshotRevision, now,
			); err != nil {
				return err
			}
		case err != nil:
			return err
		case subject.UserID != input.UserID:
			if err := s.repo.CreateSubjectConflictTx(ctx, tx, application, challenge.SubjectHash, subject.UserID, now); err != nil {
				return err
			}
			outcome = ErrSubjectConflict
			if err := s.repo.insertAttemptAndProgressTx(ctx, tx, application, attemptResultFor(
				lockedConfig, "failed", "subject_conflict", &record.SnapshotID,
				&record.SnapshotRevision, challenge.PrivacyNotice, now,
			), now); err != nil {
				return err
			}
			return s.repo.BumpEligibilityRevisionTx(ctx, tx, input.UserID, lockedConfig.SchoolID, "subject_conflict", now)
		}

		credentialID, err := newID()
		if err != nil {
			return err
		}
		var expiresAt *time.Time
		if lockedConfig.CredentialTTL != nil {
			expires := now.Add(*lockedConfig.CredentialTTL)
			expiresAt = &expires
		}
		credential := Credential{
			ID: credentialID, UserID: input.UserID, SchoolID: lockedConfig.SchoolID,
			Method: MethodStudentEmailOutboundOTP, Status: CredentialActive,
			CredentialClass: "formal_student", SubjectHash: challenge.SubjectHash,
			SubjectDisplay: challenge.SubjectDisplay, EnrollmentID: &subject.ID,
			RosterDependency: "required", VerifiedAt: now, ExpiresAt: expiresAt, Revision: 1,
		}
		metadata := json.RawMessage(`{"evidence_path":"canonical_student_email_otp","roster_satisfied":true}`)
		if err := s.repo.CreateCredentialTx(
			ctx, tx, credential, application.ID, lockedConfig.AdapterID,
			lockedConfig.AdapterVersion, &record.SnapshotID, &record.SnapshotRevision,
			metadata, now,
		); err != nil {
			return err
		}
		if err := s.repo.CompleteApplicationTx(ctx, tx, application, attemptResultFor(
			lockedConfig, "succeeded", "verified", &record.SnapshotID,
			&record.SnapshotRevision, challenge.PrivacyNotice, now,
		), now); err != nil {
			return err
		}
		return s.repo.BumpEligibilityRevisionTx(ctx, tx, input.UserID, lockedConfig.SchoolID, "credential_activated", now)
	})
	if err != nil {
		return nil, err
	}
	if outcome != nil {
		return nil, outcome
	}
	// The approved application is the replay fence even if best-effort Redis
	// cleanup is temporarily unavailable. Never turn a committed credential
	// into a user-visible failure that invites a duplicate retry.
	if cleanupErr := s.deleteEmailOTPChallenge(
		context.WithoutCancel(ctx), input.ApplicationID,
	); cleanupErr != nil {
		logger.FromContext(ctx).Warn("failed to remove consumed student email challenge", zap.Error(cleanupErr))
	}
	return s.GetApplication(ctx, input.UserID, input.ApplicationID)
}

func (s *Service) prepareStudentEmailIdentity(
	ctx context.Context,
	input StudentEmailIdentityInput,
	method Method,
) (*preparedStudentEmailIdentity, error) {
	application, err := s.repo.GetApplication(ctx, input.ApplicationID, input.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrApplicationNotFound
	}
	if err != nil {
		return nil, err
	}
	now := s.now()
	if !applicationIsMutable(application, now) {
		if !application.ExpiresAt.After(now) {
			return nil, ErrApplicationExpired
		}
		return nil, ErrApplicationState
	}
	config, err := s.repo.GetMethodConfig(ctx, application.SchoolCode, method)
	if errors.Is(err, pgx.ErrNoRows) || !methodHealthy(config) {
		return nil, ErrMethodUnavailable
	}
	if err != nil {
		return nil, err
	}
	if config.AdapterID != BUAAAdapterID || config.RosterDependency != "required" {
		return nil, ErrMethodUnavailable
	}
	if !input.SensitiveDataConsent || config.PrivacyNoticeVersion == "" ||
		config.PrivacyNoticeVersion != input.PrivacyNoticeVersion {
		return nil, ErrConsentRequired
	}
	studentID, studentOK := s.buaa.NormalizeStudentID(input.StudentID)
	name, nameOK := s.buaa.NormalizeName(input.Name)
	if !studentOK || !nameOK {
		return nil, s.failAttempt(
			ctx, application, config, "information_mismatch", ErrInformationMismatch, nil, nil, now,
		)
	}
	studentIDHash, err := ComputeRosterBlindIndex(s.hmacKey, config.SchoolID, BlindIndexStudentID, studentID)
	if err != nil {
		return nil, err
	}
	nameHash, err := ComputeRosterBlindIndex(s.hmacKey, config.SchoolID, BlindIndexName, name)
	if err != nil {
		return nil, err
	}
	subjectHash, err := ComputeRosterBlindIndex(s.hmacKey, config.SchoolID, BlindIndexSubject, studentID)
	if err != nil {
		return nil, err
	}
	record, err := s.repo.GetActiveRosterRecord(ctx, config.SchoolID, studentIDHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, s.failAttempt(
			ctx, application, config, "information_mismatch", ErrInformationMismatch, nil, nil, now,
		)
	}
	if err != nil {
		return nil, err
	}
	if !s.emailRosterRecordMatches(record, config, nameHash, now) {
		return nil, s.failAttempt(
			ctx, application, config, "information_mismatch", ErrInformationMismatch,
			&record.SnapshotID, &record.SnapshotRevision, now,
		)
	}
	return &preparedStudentEmailIdentity{
		Application: application, Config: config, StudentID: studentID,
		NameHash: nameHash, StudentIDHash: studentIDHash, SubjectHash: subjectHash,
		Record: record, PreparedAt: now,
	}, nil
}

func (s *Service) emailRosterRecordMatches(record *RosterRecord, config *MethodConfig, nameHash string, now time.Time) bool {
	return record != nil && config != nil && record.HMACKeyVersion == RosterHMACKeyVersion &&
		record.EligibilityStatus == "eligible" && snapshotFresh(record.SourceCutoffAt, config.SnapshotHardExpiry, now) &&
		constantTimeStringEqual(record.NameHash, nameHash)
}

func parseEmailOTPPolicy(raw json.RawMessage) (emailOTPPolicy, bool) {
	policy := emailOTPPolicy{TTL: emailOTPDefaultTTL, Cooldown: emailOTPDefaultCooldown, MaxAttempts: emailOTPDefaultMaxAttempts}
	if len(raw) == 0 || string(raw) == "{}" {
		return policy, true
	}
	var configured struct {
		TTLSeconds      int `json:"otpTtlSeconds"`
		CooldownSeconds int `json:"otpCooldownSeconds"`
		MaxAttempts     int `json:"otpMaxAttempts"`
	}
	if json.Unmarshal(raw, &configured) != nil {
		return emailOTPPolicy{}, false
	}
	if configured.TTLSeconds != 0 {
		policy.TTL = time.Duration(configured.TTLSeconds) * time.Second
	}
	if configured.CooldownSeconds != 0 {
		policy.Cooldown = time.Duration(configured.CooldownSeconds) * time.Second
	}
	if configured.MaxAttempts != 0 {
		policy.MaxAttempts = configured.MaxAttempts
	}
	if policy.TTL < time.Minute || policy.TTL > 15*time.Minute ||
		policy.Cooldown < 15*time.Second || policy.Cooldown > 10*time.Minute ||
		policy.MaxAttempts < 1 || policy.MaxAttempts > 10 {
		return emailOTPPolicy{}, false
	}
	return policy, true
}

func buaaEmailDomain(domains []string) (string, bool) {
	for _, domain := range domains {
		normalized := strings.ToLower(strings.TrimSpace(domain))
		if normalized == "buaa.edu.cn" {
			return normalized, true
		}
	}
	return "", false
}

func (s *Service) reserveEmailOTPCooldown(ctx context.Context, applicationID string, cooldown time.Duration) error {
	result, err := s.redisClient.SetArgs(ctx, emailOTPCooldownKeyPrefix+applicationID, "1", redis.SetArgs{
		Mode: "NX", TTL: cooldown,
	}).Result()
	if errors.Is(err, redis.Nil) || (err == nil && result != "OK") {
		return ErrEmailOTPCooldown
	}
	if err != nil {
		return fmt.Errorf("%w: reserve student email otp cooldown", ErrDependencyUnavailable)
	}
	return nil
}

func (s *Service) storeEmailOTPChallenge(ctx context.Context, challenge storedEmailOTPChallenge, ttl time.Duration) error {
	payload, err := json.Marshal(challenge)
	if err != nil {
		return err
	}
	pipe := s.redisClient.TxPipeline()
	pipe.Set(ctx, emailOTPChallengeKeyPrefix+challenge.ApplicationID, payload, ttl)
	pipe.Del(ctx, emailOTPAttemptsKeyPrefix+challenge.ApplicationID)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("%w: store student email otp challenge", ErrDependencyUnavailable)
	}
	return nil
}

func (s *Service) loadEmailOTPChallenge(ctx context.Context, applicationID string) (*storedEmailOTPChallenge, error) {
	payload, err := s.redisClient.Get(ctx, emailOTPChallengeKeyPrefix+applicationID).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrEmailOTPExpired
	}
	if err != nil {
		return nil, fmt.Errorf("%w: load student email otp challenge", ErrDependencyUnavailable)
	}
	var challenge storedEmailOTPChallenge
	if json.Unmarshal(payload, &challenge) != nil || challenge.ApplicationID != applicationID {
		return nil, ErrEmailOTPExpired
	}
	return &challenge, nil
}

func (s *Service) recordEmailOTPFailure(ctx context.Context, applicationID string, policy emailOTPPolicy) error {
	key := emailOTPAttemptsKeyPrefix + applicationID
	attempts, err := s.redisClient.Incr(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("%w: record student email otp failure", ErrDependencyUnavailable)
	}
	if attempts == 1 {
		if err := s.redisClient.Expire(ctx, key, policy.TTL).Err(); err != nil {
			return fmt.Errorf("%w: expire student email otp attempts", ErrDependencyUnavailable)
		}
	}
	if attempts >= int64(policy.MaxAttempts) {
		return errors.Join(
			ErrEmailOTPMaxAttempts,
			s.deleteEmailOTPChallenge(context.WithoutCancel(ctx), applicationID),
		)
	}
	return ErrEmailOTPInvalid
}

func (s *Service) deleteEmailOTPChallenge(ctx context.Context, applicationID string) error {
	return s.redisClient.Del(
		ctx,
		emailOTPChallengeKeyPrefix+applicationID,
		emailOTPAttemptsKeyPrefix+applicationID,
	).Err()
}

func emailOTPChallengeView(challenge storedEmailOTPChallenge, maxAttempts int) *StudentEmailOTPChallenge {
	return &StudentEmailOTPChallenge{
		ApplicationID:     challenge.ApplicationID,
		MaskedEmail:       maskEmail(challenge.Email),
		ExpiresAt:         challenge.ExpiresAt,
		ResendAvailable:   challenge.ResendAvailableAt,
		RemainingAttempts: maxAttempts,
	}
}

func maskEmail(value string) string {
	parts := strings.Split(value, "@")
	if len(parts) != 2 {
		return "***"
	}
	return maskStudentID(parts[0]) + "@" + parts[1]
}
