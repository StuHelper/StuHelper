package studentverification

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	appcrypto "github.com/StuHelper/StuHelper/server/internal/pkg/crypto"
)

const (
	inboundEmailDefaultTTL       = 10 * time.Minute
	inboundEmailSenderHashDomain = "student-verification-inbound-sender:v1:"
	inboundEmailCodeHashDomain   = "student-verification-inbound-code:v1:"
	inboundEmailEventHashDomain  = "student-verification-inbound-event:v1:"
	inboundEmailBodyPrefix       = "StuHelper-Code: "
)

func (s *Service) CreateInboundEmailChallenge(
	ctx context.Context,
	input StudentEmailIdentityInput,
) (*InboundEmailChallenge, error) {
	if s.inboundEmailTargetResolver == nil || s.rosterCipher == nil ||
		s.rosterEncryptionKeyVersion <= 0 {
		return nil, ErrDependencyUnavailable
	}
	prepared, err := s.prepareStudentEmailIdentity(ctx, input, MethodStudentEmailInbound)
	if err != nil {
		return nil, err
	}
	application, config := prepared.Application, prepared.Config
	studentID, nameHash := prepared.StudentID, prepared.NameHash
	studentIDHash, subjectHash := prepared.StudentIDHash, prepared.SubjectHash
	record, now := prepared.Record, prepared.PreparedAt
	domain, ok := buaaEmailDomain(config.EmailDomains)
	if !ok {
		return nil, ErrMethodUnavailable
	}
	expectedSender, ok := normalizeInboundMailbox(studentID + "@" + domain)
	if !ok {
		return nil, ErrMethodUnavailable
	}
	targetAddress, err := s.inboundEmailTargetResolver.TargetAddress(ctx, application.SchoolCode)
	if err != nil {
		return nil, fmt.Errorf("%w: resolve inbound email target", ErrDependencyUnavailable)
	}
	targetAddress, ok = normalizeInboundMailbox(targetAddress)
	if !ok {
		return nil, ErrMethodUnavailable
	}
	ttl, ok := inboundEmailTTL(config.RiskPolicy)
	if !ok {
		return nil, ErrMethodUnavailable
	}
	challengeValue, err := generateInboundEmailChallengeValue()
	if err != nil {
		return nil, err
	}
	challengeID, err := newID()
	if err != nil {
		return nil, err
	}
	expectedSenderHash, err := appcrypto.HMACHashWithKey(
		inboundEmailSenderHashDomain+expectedSender,
		s.hmacKey,
	)
	if err != nil {
		return nil, err
	}
	challengeValueHash, err := appcrypto.HMACHashWithKey(
		inboundEmailCodeHashDomain+challengeID+":"+challengeValue,
		s.hmacKey,
	)
	if err != nil {
		return nil, err
	}
	challengeValueEnc, err := s.rosterCipher.Encrypt(challengeValue)
	if err != nil {
		return nil, fmt.Errorf("encrypt inbound email challenge: %w", err)
	}
	routingSubject := "StuHelper student verification " + challengeID
	challenge := storedInboundEmailChallenge{
		InboundEmailChallenge: InboundEmailChallenge{
			ID: challengeID, ApplicationID: application.ID, UserID: input.UserID,
			SchoolID: config.SchoolID, Status: "waiting", TargetAddress: targetAddress,
			ExpectedSenderMasked: maskEmail(expectedSender), Subject: routingSubject,
			ChallengeValue: challengeValue, ExpiresAt: now.Add(ttl),
		},
		ExpectedSenderHash: expectedSenderHash, ChallengeValueEnc: challengeValueEnc,
		ChallengeValueHash:   challengeValueHash,
		EncryptionKeyVersion: s.rosterEncryptionKeyVersion, HMACKeyVersion: RosterHMACKeyVersion,
		StudentIDHash: studentIDHash, NameHash: nameHash,
		EnrollmentSubjectHash: subjectHash, StudentIDDisplay: maskStudentID(studentID),
		SnapshotID: record.SnapshotID, SnapshotRevision: record.SnapshotRevision,
		PrivacyNoticeVersion: input.PrivacyNoticeVersion, CreatedAt: now,
	}
	err = s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		lockedApplication, lockErr := s.repo.GetApplicationForUpdateTx(ctx, tx, application.ID, input.UserID)
		if lockErr != nil {
			return lockErr
		}
		if !applicationIsMutable(lockedApplication, now) {
			return ErrApplicationState
		}
		lockedConfig, lockErr := s.repo.GetMethodConfigTx(ctx, tx, application.SchoolCode, MethodStudentEmailInbound)
		if lockErr != nil || !methodHealthy(lockedConfig) ||
			lockedConfig.PrivacyNoticeVersion != input.PrivacyNoticeVersion ||
			lockedConfig.AdapterID != BUAAAdapterID || lockedConfig.RosterDependency != "required" {
			return ErrMethodUnavailable
		}
		lockedRecord, lockErr := s.repo.GetActiveRosterRecordTx(ctx, tx, config.SchoolID, studentIDHash)
		if lockErr != nil || !s.emailRosterRecordMatches(lockedRecord, lockedConfig, nameHash, now) {
			return ErrDependencyUnavailable
		}
		challenge.SnapshotID = lockedRecord.SnapshotID
		challenge.SnapshotRevision = lockedRecord.SnapshotRevision
		if err := s.repo.CancelWaitingInboundEmailChallengesTx(ctx, tx, application.ID, now); err != nil {
			return err
		}
		if err := s.repo.InsertInboundEmailChallengeTx(ctx, tx, challenge); err != nil {
			return err
		}
		return s.repo.ProgressApplicationMethodTx(
			ctx, tx, application.ID, MethodStudentEmailInbound,
			input.PrivacyNoticeVersion, now,
		)
	})
	if err != nil {
		return nil, err
	}
	view := challenge.InboundEmailChallenge
	return &view, nil
}

func (s *Service) GetInboundEmailChallenge(
	ctx context.Context,
	userID int64,
	applicationID string,
) (*InboundEmailChallenge, error) {
	challenge, err := s.repo.GetInboundEmailChallenge(ctx, applicationID, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInboundEmailChallengeNotFound
	}
	if err != nil {
		return nil, err
	}
	now := s.now()
	if challenge.Status == "waiting" && !challenge.ExpiresAt.After(now) {
		if err := s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
			return s.repo.ExpireInboundEmailChallengeTx(ctx, tx, challenge.ID, now)
		}); err != nil {
			return nil, err
		}
		challenge.Status = "expired"
	}
	view := challenge.InboundEmailChallenge
	if challenge.Status == "waiting" {
		value, decryptErr := s.rosterCipher.Decrypt(challenge.ChallengeValueEnc)
		if decryptErr != nil {
			return nil, ErrDependencyUnavailable
		}
		view.ChallengeValue = value
	}
	return &view, nil
}

func (s *Service) ProcessInboundEmailEvent(ctx context.Context, event InboundEmailEvent) error {
	if s.rosterCipher == nil || strings.TrimSpace(event.EventReference) == "" ||
		len(event.EventReference) > 500 || len(event.Subject) > 200 ||
		len(event.TextBody) > 64*1024 || event.ReceivedAt.IsZero() {
		return ErrInboundEmailEventInvalid
	}
	now := s.now()
	if event.ReceivedAt.After(now.Add(5 * time.Minute)) {
		return ErrInboundEmailEventInvalid
	}
	eventReferenceHash, err := appcrypto.HMACHashWithKey(
		inboundEmailEventHashDomain+event.EventReference,
		s.hmacKey,
	)
	if err != nil {
		return err
	}
	return s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		replayed, err := s.repo.InboundEmailEventExistsTx(ctx, tx, eventReferenceHash)
		if err != nil {
			return err
		}
		if replayed {
			return nil
		}
		challenge, err := s.repo.GetInboundEmailChallengeBySubjectForUpdateTx(ctx, tx, event.Subject)
		if errors.Is(err, pgx.ErrNoRows) {
			return s.repo.InsertInboundEmailEventTx(
				ctx, tx, eventReferenceHash, nil, false, false, false,
				"challenge_not_found", event.ReceivedAt, now,
			)
		}
		if err != nil {
			return err
		}
		challengeID := challenge.ID
		if event.ReceivedAt.Before(challenge.CreatedAt.Add(-5 * time.Minute)) {
			return ErrInboundEmailEventInvalid
		}
		if challenge.Status != "waiting" {
			return s.repo.InsertInboundEmailEventTx(
				ctx, tx, eventReferenceHash, &challengeID, false, false, false,
				"replayed", event.ReceivedAt, now,
			)
		}
		if !challenge.ExpiresAt.After(now) || event.ReceivedAt.After(challenge.ExpiresAt) {
			if err := s.repo.ExpireInboundEmailChallengeTx(ctx, tx, challenge.ID, now); err != nil {
				return err
			}
			return s.repo.InsertInboundEmailEventTx(
				ctx, tx, eventReferenceHash, &challengeID, false, false, false,
				"challenge_expired", event.ReceivedAt, now,
			)
		}
		envelopeFrom, envelopeOK := normalizeInboundMailbox(event.EnvelopeFrom)
		headerFrom, headerOK := normalizeInboundMailbox(event.HeaderFrom)
		envelopeHash, err := appcrypto.HMACHashWithKey(inboundEmailSenderHashDomain+envelopeFrom, s.hmacKey)
		if err != nil {
			return err
		}
		headerHash, err := appcrypto.HMACHashWithKey(inboundEmailSenderHashDomain+headerFrom, s.hmacKey)
		if err != nil {
			return err
		}
		senderVerified := envelopeOK && headerOK &&
			subtle.ConstantTimeCompare([]byte(envelopeHash), []byte(challenge.ExpectedSenderHash)) == 1 &&
			subtle.ConstantTimeCompare([]byte(headerHash), []byte(challenge.ExpectedSenderHash)) == 1
		if !senderVerified {
			return s.repo.InsertInboundEmailEventTx(
				ctx, tx, eventReferenceHash, &challengeID, false, false, false,
				"sender_mismatch", event.ReceivedAt, now,
			)
		}
		mailAuthenticationVerified := event.SPFPass && event.DKIMPass && event.DMARCPass
		if !mailAuthenticationVerified {
			return s.repo.InsertInboundEmailEventTx(
				ctx, tx, eventReferenceHash, &challengeID, true, false, false,
				"mail_authentication_failed", event.ReceivedAt, now,
			)
		}
		challengeValue, err := s.rosterCipher.Decrypt(challenge.ChallengeValueEnc)
		if err != nil {
			return ErrDependencyUnavailable
		}
		challengeHash, err := appcrypto.HMACHashWithKey(
			inboundEmailCodeHashDomain+challenge.ID+":"+challengeValue,
			s.hmacKey,
		)
		if err != nil {
			return err
		}
		challengeVerified := subtle.ConstantTimeCompare(
			[]byte(challengeHash), []byte(challenge.ChallengeValueHash),
		) == 1 && inboundEmailBodyHasExactChallenge(event.TextBody, challengeValue)
		if !challengeVerified {
			return s.repo.InsertInboundEmailEventTx(
				ctx, tx, eventReferenceHash, &challengeID, true, true, false,
				"challenge_mismatch", event.ReceivedAt, now,
			)
		}
		return s.completeInboundEmailVerificationTx(
			ctx, tx, challenge, eventReferenceHash, event.ReceivedAt, now,
		)
	})
}

func (s *Service) completeInboundEmailVerificationTx(
	ctx context.Context,
	tx pgx.Tx,
	challenge *storedInboundEmailChallenge,
	eventReferenceHash string,
	receivedAt time.Time,
	now time.Time,
) error {
	challengeID := challenge.ID
	application, err := s.repo.GetApplicationForUpdateTx(ctx, tx, challenge.ApplicationID, challenge.UserID)
	if err != nil || !applicationIsMutable(application, now) {
		return s.repo.InsertInboundEmailEventTx(
			ctx, tx, eventReferenceHash, &challengeID, true, true, true,
			"application_state_changed", receivedAt, now,
		)
	}
	config, err := s.repo.GetMethodConfigTx(ctx, tx, application.SchoolCode, MethodStudentEmailInbound)
	if err != nil || !methodHealthy(config) || config.AdapterID != BUAAAdapterID ||
		config.RosterDependency != "required" ||
		config.PrivacyNoticeVersion != challenge.PrivacyNoticeVersion {
		return s.repo.InsertInboundEmailEventTx(
			ctx, tx, eventReferenceHash, &challengeID, true, true, true,
			"application_state_changed", receivedAt, now,
		)
	}
	record, err := s.repo.GetActiveRosterRecordTx(ctx, tx, challenge.SchoolID, challenge.StudentIDHash)
	if err != nil || !s.emailRosterRecordMatches(record, config, challenge.NameHash, now) {
		if err := s.repo.insertAttemptAndProgressTx(ctx, tx, application, attemptResultFor(
			config, "unavailable", "roster_changed", &challenge.SnapshotID,
			&challenge.SnapshotRevision, challenge.PrivacyNoticeVersion, now,
		), now); err != nil {
			return err
		}
		return s.repo.InsertInboundEmailEventTx(
			ctx, tx, eventReferenceHash, &challengeID, true, true, true,
			"roster_changed", receivedAt, now,
		)
	}
	if err := s.repo.LockEnrollmentSubjectTx(ctx, tx, config.SchoolID, challenge.EnrollmentSubjectHash); err != nil {
		return err
	}
	subject, err := s.repo.GetActiveEnrollmentSubjectTx(ctx, tx, config.SchoolID, challenge.EnrollmentSubjectHash)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		subjectID, idErr := newID()
		if idErr != nil {
			return idErr
		}
		subject = &EnrollmentSubject{
			ID: subjectID, UserID: challenge.UserID, SchoolID: config.SchoolID,
			SubjectHash: challenge.EnrollmentSubjectHash, StudentIDHash: challenge.StudentIDHash,
			StudentDisplay: challenge.StudentIDDisplay, BindingStatus: "active",
		}
		if err := s.repo.CreateEnrollmentSubjectTx(
			ctx, tx, *subject, record.DocumentNumberHash, MethodStudentEmailInbound,
			&record.SnapshotID, &record.SnapshotRevision, now,
		); err != nil {
			return err
		}
	case err != nil:
		return err
	case subject.UserID != challenge.UserID:
		if err := s.repo.CreateSubjectConflictTx(ctx, tx, application, challenge.EnrollmentSubjectHash, subject.UserID, now); err != nil {
			return err
		}
		if err := s.repo.insertAttemptAndProgressTx(ctx, tx, application, attemptResultFor(
			config, "failed", "subject_conflict", &record.SnapshotID,
			&record.SnapshotRevision, challenge.PrivacyNoticeVersion, now,
		), now); err != nil {
			return err
		}
		return s.repo.InsertInboundEmailEventTx(
			ctx, tx, eventReferenceHash, &challengeID, true, true, true,
			"application_state_changed", receivedAt, now,
		)
	}

	credentialID, err := newID()
	if err != nil {
		return err
	}
	var expiresAt *time.Time
	if config.CredentialTTL != nil {
		expires := now.Add(*config.CredentialTTL)
		expiresAt = &expires
	}
	credential := Credential{
		ID: credentialID, UserID: challenge.UserID, SchoolID: config.SchoolID,
		Method: MethodStudentEmailInbound, Status: CredentialActive,
		CredentialClass: "formal_student", SubjectHash: challenge.EnrollmentSubjectHash,
		SubjectDisplay: challenge.StudentIDDisplay, EnrollmentID: &subject.ID,
		RosterDependency: "required", VerifiedAt: now, ExpiresAt: expiresAt, Revision: 1,
	}
	metadata := json.RawMessage(`{"evidence_path":"canonical_student_email_inbound","mail_authentication":true,"roster_satisfied":true}`)
	if err := s.repo.CreateCredentialTx(
		ctx, tx, credential, application.ID, config.AdapterID, config.AdapterVersion,
		&record.SnapshotID, &record.SnapshotRevision, metadata, now,
	); err != nil {
		return err
	}
	if err := s.repo.CompleteApplicationTx(ctx, tx, application, attemptResultFor(
		config, "succeeded", "verified", &record.SnapshotID, &record.SnapshotRevision,
		challenge.PrivacyNoticeVersion, now,
	), now); err != nil {
		return err
	}
	if err := s.repo.CompleteInboundEmailChallengeTx(ctx, tx, challenge.ID, eventReferenceHash, now); err != nil {
		return err
	}
	if err := s.repo.InsertInboundEmailEventTx(
		ctx, tx, eventReferenceHash, &challengeID, true, true, true,
		"verified", receivedAt, now,
	); err != nil {
		return err
	}
	return s.repo.BumpEligibilityRevisionTx(
		ctx, tx, challenge.UserID, challenge.SchoolID, "credential_activated", now,
	)
}

func generateInboundEmailChallengeValue() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func normalizeInboundMailbox(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 320 || strings.ContainsAny(value, "\r\n") {
		return "", false
	}
	address, err := mail.ParseAddress(value)
	if err != nil || address.Address != value || strings.Count(value, "@") != 1 {
		return "", false
	}
	parts := strings.SplitN(value, "@", 2)
	if parts[0] == "" || parts[1] == "" {
		return "", false
	}
	return strings.ToLower(value), true
}

func inboundEmailBodyHasExactChallenge(body string, challengeValue string) bool {
	expected := inboundEmailBodyPrefix + challengeValue
	for _, line := range strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n") {
		if subtle.ConstantTimeCompare([]byte(strings.TrimSpace(line)), []byte(expected)) == 1 {
			return true
		}
	}
	return false
}

func inboundEmailTTL(raw json.RawMessage) (time.Duration, bool) {
	if len(raw) == 0 || string(raw) == "{}" {
		return inboundEmailDefaultTTL, true
	}
	var policy struct {
		TTLSeconds int `json:"inboundTTLSeconds"`
	}
	if json.Unmarshal(raw, &policy) != nil {
		return 0, false
	}
	if policy.TTLSeconds == 0 {
		return inboundEmailDefaultTTL, true
	}
	ttl := time.Duration(policy.TTLSeconds) * time.Second
	return ttl, ttl >= 2*time.Minute && ttl <= 30*time.Minute
}
