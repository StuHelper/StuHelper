package studentverification

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
	"github.com/StuHelper/StuHelper/server/internal/pkg/phoneutil"
)

const (
	phoneOperationTTL       = 15 * time.Minute
	phoneHMACKeyVersion     = 1
	phoneWorkerPollInterval = 2 * time.Second
	phoneWorkerLease        = 2 * time.Minute
	phoneWorkerBatchSize    = 16
)

func (s *Service) CreatePhoneOperation(
	ctx context.Context,
	input CreatePhoneOperationInput,
) (*PhoneBindingOperation, error) {
	if input.UserID <= 0 || (input.Kind != PhoneOperationBind && input.Kind != PhoneOperationChange) {
		return nil, ErrPhoneOperationConflict
	}
	if s.phoneCipher == nil || s.phoneEncryptionKeyVersion <= 0 || s.phoneAuthority == nil {
		return nil, ErrDependencyUnavailable
	}
	mainlandPhone, canonicalPhone, ok := normalizeMainlandPhone(input.Phone)
	if !ok {
		return nil, ErrPhoneInvalid
	}
	phoneHash, err := phoneutil.HashLookupWithKey(canonicalPhone, s.hmacKey)
	if err != nil {
		return nil, fmt.Errorf("hash phone projection: %w", err)
	}
	phoneEnc, err := s.phoneCipher.Encrypt(canonicalPhone)
	if err != nil {
		return nil, fmt.Errorf("encrypt phone operation target: %w", err)
	}
	masked := maskCanonicalPhone(mainlandPhone)
	encryptionVersion := s.phoneEncryptionKeyVersion
	hmacVersion := phoneHMACKeyVersion
	now := s.now()
	operationID, err := newID()
	if err != nil {
		return nil, err
	}
	operation := PhoneBindingOperation{
		ID: operationID, UserID: input.UserID, OperationKind: input.Kind,
		Status:         PhoneOperationPendingVerification,
		TargetPhoneEnc: phoneEnc, TargetPhoneHash: &phoneHash,
		TargetPhoneMasked: &masked, EncryptionKeyVersion: &encryptionVersion,
		HMACKeyVersion: &hmacVersion, Revision: 1,
		ExpiresAt: now.Add(phoneOperationTTL), CreatedAt: now, UpdatedAt: now,
	}

	evidence, err := s.matchBUAAPhoneRoster(ctx, input, mainlandPhone, now)
	if err != nil {
		return nil, err
	}
	if evidence == nil && s.phoneOTP == nil {
		return nil, ErrDependencyUnavailable
	}
	if err := s.repo.CreatePhoneOperation(ctx, operation, evidence, now); err != nil {
		return nil, normalizePhoneRepositoryError(err)
	}
	if evidence != nil {
		// Casdoor is external to the PostgreSQL transaction. A failure here leaves
		// a durable, non-authorizing operation for the worker to retry.
		if processErr := s.ProcessPhoneOperation(ctx, input.UserID, operation.ID); processErr != nil {
			logger.FromContext(ctx).Warn("phone operation deferred to reconciliation", zap.Error(processErr))
		}
	}
	return s.GetPhoneOperation(ctx, input.UserID, operation.ID)
}

func (s *Service) matchBUAAPhoneRoster(
	ctx context.Context,
	input CreatePhoneOperationInput,
	mainlandPhone string,
	now time.Time,
) (*PhoneRosterEvidence, error) {
	schoolCode := strings.TrimSpace(input.SchoolCode)
	if schoolCode == "" {
		schoolCode = BUAASchoolCode
	}
	if schoolCode != BUAASchoolCode {
		return nil, nil
	}
	schoolID, _, err := s.repo.GetSchoolByCode(ctx, schoolCode)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("resolve school for phone evidence: %w", err)
	}

	var studentIDHash, nameHash *string
	studentIDInput := strings.TrimSpace(input.StudentID)
	nameInput := strings.TrimSpace(input.Name)
	if studentIDInput != "" || nameInput != "" {
		studentID, studentOK := s.buaa.NormalizeStudentID(studentIDInput)
		name, nameOK := s.buaa.NormalizeName(nameInput)
		if !studentOK || !nameOK {
			return nil, nil
		}
		studentHash, err := ComputeRosterBlindIndex(s.hmacKey, schoolID, BlindIndexStudentID, studentID)
		if err != nil {
			return nil, err
		}
		resolvedNameHash, err := ComputeRosterBlindIndex(s.hmacKey, schoolID, BlindIndexName, name)
		if err != nil {
			return nil, err
		}
		studentIDHash = &studentHash
		nameHash = &resolvedNameHash
	}
	phoneHash, err := ComputeRosterBlindIndex(
		s.hmacKey,
		schoolID,
		BlindIndexPhone,
		mainlandPhone,
	)
	if err != nil {
		return nil, err
	}
	evidence, err := s.repo.FindBUAAPhoneRosterEvidence(
		ctx,
		input.UserID,
		schoolCode,
		studentIDHash,
		nameHash,
		phoneHash,
		now,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("evaluate school phone evidence: %w", err)
	}
	if !snapshotFresh(evidence.SourceCutoffAt, evidence.HardExpiry, now) {
		return nil, nil
	}
	return evidence, nil
}

func (s *Service) GetPhoneOperation(
	ctx context.Context,
	userID int64,
	operationID string,
) (*PhoneBindingOperation, error) {
	operation, err := s.repo.GetPhoneOperation(ctx, operationID, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPhoneOperationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get phone operation: %w", err)
	}
	operation.VerificationStep = phoneVerificationStep(operation)
	return operation, nil
}

func (s *Service) SendPhoneSMS(
	ctx context.Context,
	userID int64,
	operationID string,
) (*PhoneBindingOperation, error) {
	if s.phoneOTP == nil || s.phoneCipher == nil {
		return nil, ErrDependencyUnavailable
	}
	operation, err := s.GetPhoneOperation(ctx, userID, operationID)
	if err != nil {
		return nil, err
	}
	if operation.Status != PhoneOperationPendingVerification || !operation.ExpiresAt.After(s.now()) {
		return nil, ErrPhoneOperationConflict
	}
	mainlandPhone, err := s.phoneOperationMainlandNumber(operation)
	if err != nil {
		return nil, err
	}
	if err := s.phoneOTP.Issue(ctx, mainlandPhone); err != nil {
		return nil, normalizePhoneOTPError(err)
	}
	now := s.now()
	resendAt := now.Add(time.Duration(s.phoneOTP.CooldownSeconds()) * time.Second)
	if err := s.repo.RecordPhoneSMSIssued(ctx, operation.ID, userID, resendAt, now); err != nil {
		return nil, normalizePhoneRepositoryError(err)
	}
	return s.GetPhoneOperation(ctx, userID, operation.ID)
}

func (s *Service) VerifyPhoneSMS(
	ctx context.Context,
	input VerifyPhoneSMSInput,
) (*PhoneBindingOperation, error) {
	if s.phoneOTP == nil || s.phoneCipher == nil {
		return nil, ErrDependencyUnavailable
	}
	operation, err := s.GetPhoneOperation(ctx, input.UserID, input.OperationID)
	if err != nil {
		return nil, err
	}
	if operation.Status != PhoneOperationPendingVerification || !operation.ExpiresAt.After(s.now()) {
		return nil, ErrPhoneOperationConflict
	}
	mainlandPhone, err := s.phoneOperationMainlandNumber(operation)
	if err != nil {
		return nil, err
	}
	if err := s.phoneOTP.Check(ctx, mainlandPhone, input.Code); err != nil {
		return nil, normalizePhoneOTPError(err)
	}
	if err := s.repo.MarkPhoneSMSVerified(ctx, operation.ID, input.UserID, s.now()); err != nil {
		return nil, normalizePhoneRepositoryError(err)
	}
	// The durable state transition is authoritative. OTP cleanup is
	// best-effort and conditional on the same code so it cannot erase a newer
	// challenge. Operation locking prevents a checked code from authorizing two
	// transitions for the same operation.
	if consumeErr := s.phoneOTP.Consume(ctx, mainlandPhone, input.Code); consumeErr != nil {
		logger.FromContext(ctx).Warn("failed to consume verified phone challenge", zap.Error(consumeErr))
	}
	if processErr := s.ProcessPhoneOperation(ctx, input.UserID, operation.ID); processErr != nil {
		logger.FromContext(ctx).Warn("verified phone operation deferred to reconciliation", zap.Error(processErr))
	}
	return s.GetPhoneOperation(ctx, input.UserID, operation.ID)
}

func (s *Service) CreatePhoneUnbindOperation(
	ctx context.Context,
	userID int64,
) (*PhoneBindingOperation, error) {
	if userID <= 0 || s.phoneAuthority == nil {
		return nil, ErrDependencyUnavailable
	}
	now := s.now()
	operationID, err := newID()
	if err != nil {
		return nil, err
	}
	operation := PhoneBindingOperation{
		ID: operationID, UserID: userID, OperationKind: PhoneOperationUnbind,
		Status: PhoneOperationCasdoorUpdatePending, Revision: 1,
		ExpiresAt: now.Add(phoneOperationTTL), CreatedAt: now, UpdatedAt: now,
	}
	if err := s.repo.CreatePhoneUnbindOperation(ctx, operation, now); err != nil {
		return nil, normalizePhoneRepositoryError(err)
	}
	if processErr := s.ProcessPhoneOperation(ctx, userID, operation.ID); processErr != nil {
		logger.FromContext(ctx).Warn("phone unbind operation deferred to reconciliation", zap.Error(processErr))
	}
	return s.GetPhoneOperation(ctx, userID, operation.ID)
}

func (s *Service) ProcessPhoneOperation(ctx context.Context, userID int64, operationID string) error {
	if s.phoneAuthority == nil || s.phoneCipher == nil || s.phoneEncryptionKeyVersion <= 0 {
		return ErrDependencyUnavailable
	}
	operation, err := s.GetPhoneOperation(ctx, userID, operationID)
	if err != nil {
		return err
	}
	if operation.Status == PhoneOperationCompleted {
		return nil
	}
	if operation.Status == PhoneOperationCasdoorUpdatePending {
		switch operation.OperationKind {
		case PhoneOperationUnbind:
			if err := s.phoneAuthority.ClearPhone(ctx, userID); err != nil {
				return fmt.Errorf("clear Casdoor phone: %w", err)
			}
		case PhoneOperationBind, PhoneOperationChange:
			_, canonical, err := s.phoneOperationNumber(operation)
			if err != nil {
				return err
			}
			if err := s.phoneAuthority.SetPhone(ctx, userID, canonical); err != nil {
				return fmt.Errorf("update Casdoor phone: %w", err)
			}
		default:
			return ErrPhoneOperationConflict
		}
		if err := s.repo.MarkPhoneCasdoorUpdated(ctx, operation.ID, userID, s.now()); err != nil {
			return normalizePhoneRepositoryError(err)
		}
		operation, err = s.GetPhoneOperation(ctx, userID, operationID)
		if err != nil {
			return err
		}
	}
	if operation.Status != PhoneOperationProjectionPending {
		return ErrPhoneOperationConflict
	}

	readback, err := s.phoneAuthority.GetPhone(ctx, userID)
	if err != nil {
		return fmt.Errorf("read Casdoor phone projection: %w", err)
	}
	if operation.OperationKind == PhoneOperationUnbind {
		if strings.TrimSpace(readback) != "" {
			return ErrDependencyUnavailable
		}
		return s.repo.FinalizePhoneProjection(
			ctx, operation.ID, userID, nil, nil, nil, nil, nil, s.now(),
		)
	}
	mainland, canonicalReadback, ok := normalizeMainlandPhone(readback)
	if !ok {
		return ErrDependencyUnavailable
	}
	_, canonicalTarget, err := s.phoneOperationNumber(operation)
	if err != nil {
		return err
	}
	if subtle.ConstantTimeCompare([]byte(canonicalReadback), []byte(canonicalTarget)) != 1 {
		return ErrDependencyUnavailable
	}
	readbackEnc, err := s.phoneCipher.Encrypt(canonicalReadback)
	if err != nil {
		return fmt.Errorf("encrypt Casdoor phone projection: %w", err)
	}
	readbackHash, err := phoneutil.HashLookupWithKey(canonicalReadback, s.hmacKey)
	if err != nil {
		return err
	}
	masked := maskCanonicalPhone(mainland)
	encryptionVersion := s.phoneEncryptionKeyVersion
	hmacVersion := phoneHMACKeyVersion
	return s.repo.FinalizePhoneProjection(
		ctx,
		operation.ID,
		userID,
		readbackEnc,
		&readbackHash,
		&masked,
		&encryptionVersion,
		&hmacVersion,
		s.now(),
	)
}

func (s *Service) GetPhoneStatus(ctx context.Context, userID int64) (*PhoneStatus, error) {
	if userID <= 0 {
		return nil, ErrPhoneOperationNotFound
	}
	status, err := s.repo.GetPhoneStatus(ctx, userID, s.now())
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPhoneOperationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get phone status: %w", err)
	}
	return status, nil
}

func (s *Service) GetPhoneGateEligibility(ctx context.Context, userID int64) (*PhoneGateEligibility, error) {
	status, err := s.GetPhoneStatus(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &PhoneGateEligibility{
		Eligible:    status.PublishingRequirementSatisfied,
		Method:      status.Method,
		ExpiresAt:   status.ExpiresAt,
		EvaluatedAt: s.now(),
		Revision:    status.Revision,
	}, nil
}

func (s *Service) phoneOperationMainlandNumber(operation *PhoneBindingOperation) (string, error) {
	mainland, _, err := s.phoneOperationNumber(operation)
	return mainland, err
}

func (s *Service) phoneOperationNumber(operation *PhoneBindingOperation) (string, string, error) {
	if operation == nil || len(operation.TargetPhoneEnc) == 0 || s.phoneCipher == nil {
		return "", "", ErrPhoneOperationConflict
	}
	plaintext, err := s.phoneCipher.Decrypt(operation.TargetPhoneEnc)
	if err != nil {
		return "", "", fmt.Errorf("decrypt phone operation target: %w", err)
	}
	mainland, canonical, ok := normalizeMainlandPhone(plaintext)
	if !ok {
		return "", "", ErrPhoneOperationConflict
	}
	if operation.TargetPhoneHash != nil {
		hash, err := phoneutil.HashLookupWithKey(canonical, s.hmacKey)
		if err != nil {
			return "", "", err
		}
		if subtle.ConstantTimeCompare([]byte(hash), []byte(*operation.TargetPhoneHash)) != 1 {
			return "", "", ErrPhoneOperationConflict
		}
	}
	return mainland, canonical, nil
}

func normalizeMainlandPhone(value string) (string, string, bool) {
	normalized := strings.NewReplacer(" ", "", "-", "", "(", "", ")", "").Replace(strings.TrimSpace(value))
	switch {
	case strings.HasPrefix(normalized, "+86"):
		normalized = strings.TrimPrefix(normalized, "+86")
	case strings.HasPrefix(normalized, "86") && len(normalized) == 13:
		normalized = strings.TrimPrefix(normalized, "86")
	}
	if !phoneutil.IsValidMainlandPhone(normalized) {
		return "", "", false
	}
	return normalized, "+86" + normalized, true
}

func maskCanonicalPhone(mainland string) string {
	return "+86 " + phoneutil.Mask(mainland)
}

func phoneVerificationStep(operation *PhoneBindingOperation) string {
	if operation == nil {
		return "none"
	}
	switch operation.Status {
	case PhoneOperationPendingVerification:
		return "sms_otp"
	case PhoneOperationVerificationSucceeded, PhoneOperationCasdoorUpdatePending,
		PhoneOperationCasdoorUpdated, PhoneOperationProjectionPending:
		return "syncing"
	default:
		return "none"
	}
}

func normalizePhoneRepositoryError(err error) error {
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return ErrPhoneOperationNotFound
	case errors.Is(err, ErrPhoneAlreadyBound), errors.Is(err, ErrPhoneNotBound),
		errors.Is(err, ErrPhoneOwnershipConflict), errors.Is(err, ErrPhoneOperationConflict):
		return err
	default:
		return err
	}
}

func normalizePhoneOTPError(err error) error {
	switch {
	case errors.Is(err, ErrPhoneOTPCooldown), errors.Is(err, ErrPhoneOTPExpired),
		errors.Is(err, ErrPhoneOTPInvalid), errors.Is(err, ErrPhoneOTPMaxAttempts),
		errors.Is(err, ErrPhoneOTPRateLimited):
		return err
	default:
		return fmt.Errorf("phone OTP dependency: %w", err)
	}
}

func (s *Service) StartPhoneBackgroundJobs(ctx context.Context, start func(string, func(context.Context))) {
	if start == nil {
		panic("studentverification.Service.StartPhoneBackgroundJobs: starter is required")
	}
	if s.phoneAuthority == nil || s.phoneCipher == nil {
		return
	}
	start("student verification phone reconciliation", s.runPhoneWorker)
}

func (s *Service) runPhoneWorker(ctx context.Context) {
	owner, err := newID()
	if err != nil {
		logger.FromContext(ctx).Warn("failed to claim phone reconciliation jobs", zap.Error(err))
		return
	}
	ticker := time.NewTicker(phoneWorkerPollInterval)
	defer ticker.Stop()
	for {
		s.processPhoneOutboxBatch(ctx, owner)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Service) processPhoneOutboxBatch(ctx context.Context, owner string) {
	now := s.now()
	jobs, err := s.repo.ClaimPhoneOutboxJobs(
		ctx, owner, phoneWorkerBatchSize, phoneWorkerLease, now,
	)
	if err != nil {
		return
	}
	for _, job := range jobs {
		userID, err := s.repo.ResolvePhoneOperationUser(ctx, job.OperationID)
		if err == nil {
			err = s.ProcessPhoneOperation(ctx, userID, job.OperationID)
		}
		if err == nil {
			if completeErr := s.repo.CompletePhoneOutboxJob(
				ctx, job.ID, owner, s.now(),
			); completeErr != nil {
				logger.FromContext(ctx).Warn("failed to complete phone reconciliation job", zap.Error(completeErr))
			}
			continue
		}
		backoff := time.Duration(1<<min(job.AttemptCount, 6)) * time.Second
		if retryErr := s.repo.RetryPhoneOutboxJob(
			ctx, job.ID, owner, s.now().Add(backoff), "dependency_unavailable", s.now(),
		); retryErr != nil {
			logger.FromContext(ctx).Warn("failed to reschedule phone reconciliation job", zap.Error(retryErr))
		}
	}
}
