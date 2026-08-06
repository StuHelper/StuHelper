package studentverification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	appcrypto "github.com/StuHelper/StuHelper/server/internal/pkg/crypto"
)

const (
	schoolAccountRequestTimeout = 8 * time.Second
	schoolAccountSubjectDomain  = "student-verification-school-account-subject:v1:"
)

type schoolSSOQualification struct {
	RosterRequired bool
	RosterRecord   *RosterRecord
}

func (s *Service) VerifySchoolSSO(ctx context.Context, input VerifySchoolSSOInput) (*ApplicationView, error) {
	application, err := s.repo.GetApplication(ctx, input.ApplicationID, input.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrApplicationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get school account verification application: %w", err)
	}
	now := s.now()
	if !applicationIsMutable(application, now) {
		if !application.ExpiresAt.After(now) {
			return nil, ErrApplicationExpired
		}
		return nil, ErrApplicationState
	}
	config, err := s.repo.GetMethodConfig(ctx, application.SchoolCode, MethodSchoolSSO)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrMethodUnavailable
	}
	if err != nil {
		return nil, fmt.Errorf("load school account verification method: %w", err)
	}
	if !methodHealthy(config) || config.ConnectorOperation == nil || strings.TrimSpace(*config.ConnectorOperation) == "" {
		return nil, ErrMethodUnavailable
	}
	if !input.SensitiveDataConsent || config.PrivacyNoticeVersion == "" ||
		config.PrivacyNoticeVersion != input.PrivacyNoticeVersion {
		return nil, ErrConsentRequired
	}
	studentID, ok := s.buaa.NormalizeStudentID(input.StudentID)
	if config.SchoolCode != BUAASchoolCode || !ok || len(input.Password) == 0 || len(input.Password) > 256 {
		return nil, ErrSchoolAccountRejected
	}
	authenticator := s.schoolAuthenticators[config.AdapterID]
	if authenticator == nil {
		return nil, ErrMethodUnavailable
	}

	authCtx, cancel := context.WithTimeout(ctx, schoolAccountRequestTimeout)
	result, authErr := authenticator.Authenticate(authCtx, SchoolAccountAuthenticationRequest{
		ApplicationID: application.ID, SchoolID: config.SchoolID, AdapterID: config.AdapterID,
		AdapterVersion: config.AdapterVersion, ConnectorOperation: *config.ConnectorOperation,
		StudentID: studentID, Password: input.Password,
	})
	cancel()
	if authErr != nil {
		switch {
		case errors.Is(authErr, ErrSchoolAccountRejected),
			errors.Is(authErr, ErrSchoolAccountLocked),
			errors.Is(authErr, ErrSchoolAccountNotStudent):
			return nil, s.failAttempt(
				ctx, application, config, "school_account_rejected",
				ErrSchoolAccountRejected, nil, nil, now,
			)
		default:
			return nil, s.failAttempt(
				ctx, application, config, "school_account_unavailable",
				ErrDependencyUnavailable, nil, nil, now,
			)
		}
	}
	if result == nil || strings.TrimSpace(result.AccountSubject) == "" {
		return nil, s.failAttempt(
			ctx, application, config, "school_account_invalid_result",
			ErrDependencyUnavailable, nil, nil, now,
		)
	}
	verifiedStudentID, verifiedOK := s.buaa.NormalizeStudentID(result.StudentID)
	if !verifiedOK || !constantTimeStringEqual(studentID, verifiedStudentID) {
		return nil, s.failAttempt(
			ctx, application, config, "school_account_subject_mismatch",
			ErrSchoolAccountRejected, nil, nil, now,
		)
	}
	studentIDHash, err := ComputeRosterBlindIndex(s.hmacKey, config.SchoolID, BlindIndexStudentID, studentID)
	if err != nil {
		return nil, err
	}
	subjectHash, err := ComputeRosterBlindIndex(s.hmacKey, config.SchoolID, BlindIndexSubject, studentID)
	if err != nil {
		return nil, err
	}
	accountSubjectHash, err := appcrypto.HMACHashWithKey(
		fmt.Sprintf("%s%d:%s", schoolAccountSubjectDomain, config.SchoolID, strings.TrimSpace(result.AccountSubject)),
		s.hmacKey,
	)
	if err != nil {
		return nil, err
	}
	qualification, err := s.resolveSchoolSSOQualification(ctx, config, result, studentIDHash, now)
	if err != nil {
		resultErr := ErrSchoolAccountRejected
		if errors.Is(err, ErrDependencyUnavailable) || errors.Is(err, ErrMethodUnavailable) {
			resultErr = ErrDependencyUnavailable
		}
		return nil, s.failAttempt(
			ctx, application, config, "student_qualification_unavailable",
			resultErr, snapshotIDOf(qualificationRecord(qualification)),
			snapshotRevisionOf(qualificationRecord(qualification)), now,
		)
	}

	var outcome error
	err = s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		lockedApplication, err := s.repo.GetApplicationForUpdateTx(ctx, tx, application.ID, input.UserID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrApplicationNotFound
		}
		if err != nil {
			return err
		}
		if !applicationIsMutable(lockedApplication, now) {
			if !lockedApplication.ExpiresAt.After(now) {
				return ErrApplicationExpired
			}
			return ErrApplicationState
		}
		lockedConfig, err := s.repo.GetMethodConfigTx(ctx, tx, application.SchoolCode, MethodSchoolSSO)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("lock school account method: %w", err)
		}
		if errors.Is(err, pgx.ErrNoRows) || !sameSchoolSSOConfig(config, lockedConfig, input.PrivacyNoticeVersion) {
			outcome = ErrMethodUnavailable
			return s.repo.insertAttemptAndProgressTx(ctx, tx, lockedApplication, attemptResultFor(
				config, "unavailable", "method_configuration_changed", nil, nil,
				input.PrivacyNoticeVersion, now,
			), now)
		}

		var finalRecord *RosterRecord
		if qualification.RosterRequired {
			finalRecord, err = s.repo.GetActiveRosterRecordTx(ctx, tx, lockedConfig.SchoolID, studentIDHash)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("lock school account roster record: %w", err)
			}
			if errors.Is(err, pgx.ErrNoRows) || !rosterStudentEligible(finalRecord, lockedConfig, now) {
				outcome = ErrDependencyUnavailable
				return s.repo.insertAttemptAndProgressTx(ctx, tx, lockedApplication, attemptResultFor(
					lockedConfig, "unavailable", "roster_changed",
					snapshotIDOf(qualification.RosterRecord), snapshotRevisionOf(qualification.RosterRecord),
					input.PrivacyNoticeVersion, now,
				), now)
			}
		}
		if err := s.repo.LockEnrollmentSubjectTx(ctx, tx, lockedConfig.SchoolID, subjectHash); err != nil {
			return err
		}
		subject, err := s.repo.GetActiveEnrollmentSubjectTx(ctx, tx, lockedConfig.SchoolID, subjectHash)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			subjectID, idErr := newID()
			if idErr != nil {
				return idErr
			}
			subject = &EnrollmentSubject{
				ID: subjectID, UserID: input.UserID, SchoolID: lockedConfig.SchoolID,
				SubjectHash: subjectHash, StudentIDHash: studentIDHash,
				StudentDisplay: maskStudentID(studentID), BindingStatus: "active",
			}
			if err := s.repo.CreateEnrollmentSubjectTx(
				ctx, tx, *subject, documentHashOf(finalRecord), MethodSchoolSSO,
				snapshotIDOf(finalRecord), snapshotRevisionOf(finalRecord), now,
			); err != nil {
				return err
			}
		case err != nil:
			return err
		case subject.UserID != input.UserID:
			if err := s.repo.CreateSubjectConflictTx(ctx, tx, lockedApplication, subjectHash, subject.UserID, now); err != nil {
				return err
			}
			outcome = ErrSubjectConflict
			if err := s.repo.insertAttemptAndProgressTx(ctx, tx, lockedApplication, attemptResultFor(
				lockedConfig, "failed", "subject_conflict", snapshotIDOf(finalRecord),
				snapshotRevisionOf(finalRecord), input.PrivacyNoticeVersion, now,
			), now); err != nil {
				return err
			}
			return s.repo.BumpEligibilityRevisionTx(
				ctx, tx, input.UserID, lockedConfig.SchoolID, "subject_conflict", now,
			)
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
			Method: MethodSchoolSSO, Status: CredentialActive,
			CredentialClass: "formal_student", SubjectHash: subjectHash,
			SubjectDisplay: maskStudentID(studentID), EnrollmentID: &subject.ID,
			RosterDependency: lockedConfig.RosterDependency,
			VerifiedAt:       now, ExpiresAt: expiresAt, Revision: 1,
		}
		metadata, err := json.Marshal(map[string]any{
			"evidence_path":           "school_account_authentication",
			"account_subject_hash":    accountSubjectHash,
			"qualification_satisfied": true,
			"roster_satisfied":        finalRecord != nil,
		})
		if err != nil {
			return err
		}
		if err := s.repo.CreateCredentialTx(
			ctx, tx, credential, lockedApplication.ID, lockedConfig.AdapterID,
			lockedConfig.AdapterVersion, snapshotIDOf(finalRecord), snapshotRevisionOf(finalRecord),
			metadata, now,
		); err != nil {
			return err
		}
		if err := s.repo.CompleteApplicationTx(ctx, tx, lockedApplication, attemptResultFor(
			lockedConfig, "succeeded", "verified", snapshotIDOf(finalRecord),
			snapshotRevisionOf(finalRecord), input.PrivacyNoticeVersion, now,
		), now); err != nil {
			return err
		}
		return s.repo.BumpEligibilityRevisionTx(
			ctx, tx, input.UserID, lockedConfig.SchoolID, "credential_activated", now,
		)
	})
	if err != nil {
		return nil, err
	}
	if outcome != nil {
		return nil, outcome
	}
	return s.GetApplication(ctx, input.UserID, input.ApplicationID)
}

func (s *Service) resolveSchoolSSOQualification(
	ctx context.Context,
	config *MethodConfig,
	result *SchoolAccountAuthenticationResult,
	studentIDHash string,
	now time.Time,
) (*schoolSSOQualification, error) {
	switch config.RosterDependency {
	case "independent":
		return &schoolSSOQualification{}, nil
	case "required":
		record, err := s.repo.GetActiveRosterRecord(ctx, config.SchoolID, studentIDHash)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrSchoolAccountNotStudent
			}
			return nil, ErrDependencyUnavailable
		}
		if !rosterStudentEligible(record, config, now) {
			return nil, ErrSchoolAccountNotStudent
		}
		return &schoolSSOQualification{RosterRequired: true, RosterRecord: record}, nil
	case "conditional":
		var policy struct {
			Type              string `json:"type"`
			RequiredAttribute string `json:"requiredAttribute"`
		}
		if json.Unmarshal(config.ConditionalPolicy, &policy) != nil ||
			policy.Type != "adapter_assertion" || strings.TrimSpace(policy.RequiredAttribute) == "" {
			return nil, ErrMethodUnavailable
		}
		if result.Attributes == nil || !result.Attributes[policy.RequiredAttribute] {
			return nil, ErrSchoolAccountNotStudent
		}
		return &schoolSSOQualification{}, nil
	default:
		return nil, ErrMethodUnavailable
	}
}

func sameSchoolSSOConfig(original, current *MethodConfig, notice string) bool {
	return methodHealthy(current) && original != nil &&
		current.AdapterID == original.AdapterID && current.AdapterVersion == original.AdapterVersion &&
		current.RosterDependency == original.RosterDependency &&
		current.ConnectorOperation != nil && original.ConnectorOperation != nil &&
		*current.ConnectorOperation == *original.ConnectorOperation &&
		string(current.ConditionalPolicy) == string(original.ConditionalPolicy) &&
		current.PrivacyNoticeVersion == notice
}

func rosterStudentEligible(record *RosterRecord, config *MethodConfig, now time.Time) bool {
	return record != nil && config != nil && record.HMACKeyVersion == RosterHMACKeyVersion &&
		record.EligibilityStatus == "eligible" && snapshotFresh(record.SourceCutoffAt, config.SnapshotHardExpiry, now)
}

func qualificationRecord(qualification *schoolSSOQualification) *RosterRecord {
	if qualification == nil {
		return nil
	}
	return qualification.RosterRecord
}

func snapshotIDOf(record *RosterRecord) *string {
	if record == nil {
		return nil
	}
	return &record.SnapshotID
}

func snapshotRevisionOf(record *RosterRecord) *int64 {
	if record == nil {
		return nil
	}
	return &record.SnapshotRevision
}

func documentHashOf(record *RosterRecord) *string {
	if record == nil {
		return nil
	}
	return record.DocumentNumberHash
}
