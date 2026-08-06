package studentverification

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"

	appcrypto "github.com/StuHelper/StuHelper/server/internal/pkg/crypto"
	"github.com/StuHelper/StuHelper/server/internal/pkg/crypto/pii"
)

const (
	applicationTTL            = 30 * time.Minute
	continuationHashDomain    = "student-verification-continuation:v1:"
	allowedSnapshotFutureSkew = 5 * time.Minute
)

var schoolCodePattern = regexp.MustCompile(`^\d{10}$`)

type ContinuationVerifier interface {
	VerifyContinuation(ctx context.Context, rawToken string) (expiresAt time.Time, err error)
}

type StudentEmailSender interface {
	SendStudentVerificationOTP(ctx context.Context, email string, code string) error
}

type SchoolAccountAuthenticator interface {
	Authenticate(
		ctx context.Context,
		request SchoolAccountAuthenticationRequest,
	) (*SchoolAccountAuthenticationResult, error)
}

// PhoneAuthority is the only path allowed to mutate or read the current
// account phone. Its implementation must resolve the user's Casdoor subject
// internally; raw subjects never enter phone-domain persistence or events.
type PhoneAuthority interface {
	GetPhone(ctx context.Context, userID int64) (string, error)
	SetPhone(ctx context.Context, userID int64, phone string) error
	ClearPhone(ctx context.Context, userID int64) error
}

type PhoneOTPService interface {
	Issue(ctx context.Context, mainlandPhone string) error
	Check(ctx context.Context, mainlandPhone, code string) error
	Consume(ctx context.Context, mainlandPhone, code string) error
	CooldownSeconds() int
}

type ManualReviewMaterialStore interface {
	PutManualReviewMaterial(ctx context.Context, objectKey string, content []byte, contentType string) error
	DeleteManualReviewMaterial(ctx context.Context, objectKey string) error
	GetManualReviewMaterialURL(ctx context.Context, objectKey string) (string, error)
}

// EligibilityEventConsumer receives invalidation signals only. It must
// reevaluate current eligibility through the service boundary and must not
// treat an event payload as an authorization decision.
type EligibilityEventConsumer interface {
	ReevaluateStudentEligibility(
		ctx context.Context,
		userID int64,
		schoolID int64,
		minimumRevision int64,
	) error
}

type ServiceOption func(*Service)

func WithContinuationVerifier(verifier ContinuationVerifier) ServiceOption {
	return func(service *Service) {
		service.continuationVerifier = verifier
	}
}

func WithClock(now func() time.Time) ServiceOption {
	return func(service *Service) {
		if now != nil {
			service.now = now
		}
	}
}

func WithRedisClient(client *redis.Client) ServiceOption {
	return func(service *Service) {
		service.redisClient = client
	}
}

func WithStudentEmailSender(sender StudentEmailSender) ServiceOption {
	return func(service *Service) {
		service.emailSender = sender
	}
}

func WithOTPGenerator(generator func() (string, error)) ServiceOption {
	return func(service *Service) {
		if generator != nil {
			service.generateOTP = generator
		}
	}
}

func WithSchoolAccountAuthenticator(adapterID string, authenticator SchoolAccountAuthenticator) ServiceOption {
	return func(service *Service) {
		adapterID = strings.TrimSpace(adapterID)
		if adapterID != "" && authenticator != nil {
			service.schoolAuthenticators[adapterID] = authenticator
		}
	}
}

func WithPhoneAuthority(authority PhoneAuthority) ServiceOption {
	return func(service *Service) { service.phoneAuthority = authority }
}

func WithPhoneOTPService(otp PhoneOTPService) ServiceOption {
	return func(service *Service) { service.phoneOTP = otp }
}

func WithPhoneProjectionCipher(cipher pii.EncryptDecryptor, encryptionKeyVersion int) ServiceOption {
	return func(service *Service) {
		service.phoneCipher = cipher
		service.phoneEncryptionKeyVersion = encryptionKeyVersion
	}
}

func WithRosterCipher(cipher pii.EncryptDecryptor, encryptionKeyVersion int) ServiceOption {
	return func(service *Service) {
		service.rosterCipher = cipher
		service.rosterEncryptionKeyVersion = encryptionKeyVersion
	}
}

func WithInboundEmailTargetResolver(resolver InboundEmailTargetResolver) ServiceOption {
	return func(service *Service) { service.inboundEmailTargetResolver = resolver }
}

func WithManualReviewMaterialStore(store ManualReviewMaterialStore) ServiceOption {
	return func(service *Service) { service.manualMaterialStore = store }
}

func WithManualReviewPublicBaseURL(baseURL string) ServiceOption {
	return func(service *Service) {
		service.manualReviewPublicBaseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	}
}

func WithManualReviewMaterialAccessTTL(ttl time.Duration) ServiceOption {
	return func(service *Service) {
		service.manualMaterialAccessTTL = ttl
	}
}

type Service struct {
	repo                       *Repository
	hmacKey                    []byte
	now                        func() time.Time
	continuationVerifier       ContinuationVerifier
	buaa                       BUAAAdapter
	redisClient                *redis.Client
	emailSender                StudentEmailSender
	generateOTP                func() (string, error)
	schoolAuthenticators       map[string]SchoolAccountAuthenticator
	phoneAuthority             PhoneAuthority
	phoneOTP                   PhoneOTPService
	phoneCipher                pii.EncryptDecryptor
	phoneEncryptionKeyVersion  int
	rosterCipher               pii.EncryptDecryptor
	rosterEncryptionKeyVersion int
	inboundEmailTargetResolver InboundEmailTargetResolver
	manualMaterialStore        ManualReviewMaterialStore
	manualReviewPublicBaseURL  string
	manualMaterialAccessTTL    time.Duration
	eligibilityEventConsumer   EligibilityEventConsumer
}

func NewService(repo *Repository, hmacKey []byte, options ...ServiceOption) (*Service, error) {
	if repo == nil {
		return nil, errors.New("studentverification.NewService: repository is required")
	}
	if len(hmacKey) == 0 {
		return nil, errors.New("studentverification.NewService: HMAC key is required")
	}
	service := &Service{
		repo:                 repo,
		hmacKey:              append([]byte(nil), hmacKey...),
		now:                  func() time.Time { return time.Now().UTC() },
		generateOTP:          generateNumericOTP,
		schoolAuthenticators: make(map[string]SchoolAccountAuthenticator),
	}
	for _, option := range options {
		if option != nil {
			option(service)
		}
	}
	return service, nil
}

// SetEligibilityEventConsumer is called during application composition, before
// the background publisher starts. It avoids a package cycle between the
// student-verification producer and its admission consumer.
func (s *Service) SetEligibilityEventConsumer(consumer EligibilityEventConsumer) {
	s.eligibilityEventConsumer = consumer
}

func (s *Service) ListSchools(ctx context.Context) ([]VerificationSchool, error) {
	schools, err := s.repo.ListAvailableSchools(ctx)
	if err != nil {
		return nil, fmt.Errorf("list verification schools: %w", err)
	}
	result := make([]VerificationSchool, 0, len(schools))
	for _, school := range schools {
		available := false
		for index := range school.Methods {
			method := &school.Methods[index]
			if method.Availability == "available" && !s.methodRuntimeAvailable(ctx, school, method) {
				method.Availability = "temporarily_unavailable"
				method.UnavailableCode = "method_dependency_unavailable"
			}
			available = available || method.Availability == "available"
		}
		if available {
			result = append(result, school)
		}
	}
	return result, nil
}

func (s *Service) methodRuntimeAvailable(ctx context.Context, school VerificationSchool, method *MethodCapability) bool {
	if method == nil {
		return false
	}
	switch method.Method {
	case MethodSchoolSSO:
		return s.schoolAuthenticators[method.adapterID] != nil
	case MethodStudentEmailInbound:
		if s.inboundEmailTargetResolver == nil || s.rosterCipher == nil {
			return false
		}
	case MethodManualMaterialReview:
		if s.manualMaterialStore == nil || s.rosterCipher == nil ||
			s.rosterEncryptionKeyVersion <= 0 || s.manualReviewPublicBaseURL == "" ||
			s.manualMaterialAccessTTL < time.Minute || s.manualMaterialAccessTTL > 15*time.Minute {
			return false
		}
	case MethodStudentEmailOutboundOTP:
		if s.redisClient == nil || s.emailSender == nil {
			return false
		}
	}
	config, err := s.repo.GetMethodConfig(ctx, school.Code, method.Method)
	if err != nil || config == nil {
		return false
	}
	if method.Method == MethodManualMaterialReview {
		if _, _, err := decodeManualReviewConfiguration(config); err != nil {
			return false
		}
	}
	if config.RosterDependency == "required" {
		state, err := s.repo.GetActiveRosterState(ctx, school.ID)
		return err == nil && snapshotFresh(state.SourceCutoffAt, config.SnapshotHardExpiry, s.now())
	}
	return true
}

func (s *Service) CreateApplication(ctx context.Context, input CreateApplicationInput) (*ApplicationView, error) {
	if input.UserID <= 0 || !schoolCodePattern.MatchString(input.SchoolCode) {
		return nil, ErrSchoolNotFound
	}
	available, err := s.repo.SchoolHasAvailableMethod(ctx, input.SchoolCode)
	if err != nil {
		return nil, fmt.Errorf("check school availability: %w", err)
	}
	if !available {
		return nil, ErrSchoolUnavailable
	}
	schoolID, schoolName, err := s.repo.GetSchoolByCode(ctx, input.SchoolCode)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSchoolNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("resolve verification school: %w", err)
	}
	if existing, err := s.repo.GetActiveApplication(ctx, input.UserID, schoolID); err == nil {
		if existing.ExpiresAt.After(s.now()) {
			return s.applicationView(ctx, existing)
		}
		if _, expireErr := s.repo.ExpireApplication(ctx, existing.ID, input.UserID, s.now()); expireErr != nil {
			return nil, fmt.Errorf("expire stale verification application: %w", expireErr)
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("get active verification application: %w", err)
	}

	now := s.now()
	var continuationHash *string
	var continuationExpiry *time.Time
	if input.ContinuationToken != "" {
		if s.continuationVerifier == nil {
			return nil, ErrContinuationInvalid
		}
		expiresAt, err := s.continuationVerifier.VerifyContinuation(ctx, input.ContinuationToken)
		if err != nil || !expiresAt.After(now) {
			return nil, ErrContinuationInvalid
		}
		hash, err := appcrypto.HMACHashWithKey(continuationHashDomain+input.ContinuationToken, s.hmacKey)
		if err != nil {
			return nil, fmt.Errorf("hash verification continuation: %w", err)
		}
		continuationHash = &hash
		continuationExpiry = &expiresAt
	}
	applicationID, err := newID()
	if err != nil {
		return nil, err
	}
	application := Application{
		ID: applicationID, UserID: input.UserID, SchoolID: schoolID,
		SchoolCode: input.SchoolCode, SchoolName: schoolName,
		Status: ApplicationCreated, Revision: 1,
		CreatedAt: now, UpdatedAt: now, ExpiresAt: now.Add(applicationTTL),
	}
	if err := s.repo.CreateApplication(ctx, application, continuationHash, continuationExpiry); err != nil {
		if isUniqueViolation(err, "student_verification_applications_active_user_school_uidx") {
			existing, getErr := s.repo.GetActiveApplication(ctx, input.UserID, schoolID)
			if getErr != nil {
				return nil, fmt.Errorf("load concurrent verification application: %w", getErr)
			}
			return s.applicationView(ctx, existing)
		}
		return nil, err
	}
	return s.applicationView(ctx, &application)
}

func (s *Service) GetApplication(ctx context.Context, userID int64, applicationID string) (*ApplicationView, error) {
	application, err := s.repo.GetApplication(ctx, applicationID, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrApplicationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get verification application: %w", err)
	}
	if !application.ExpiresAt.After(s.now()) && applicationIsNonTerminal(application) {
		application, err = s.repo.ExpireApplication(ctx, application.ID, userID, s.now())
		if err != nil {
			return nil, fmt.Errorf("expire verification application: %w", err)
		}
	}
	return s.applicationView(ctx, application)
}

func (s *Service) CancelApplication(ctx context.Context, userID int64, applicationID string) (*ApplicationView, error) {
	application, err := s.repo.CancelApplication(ctx, applicationID, userID, s.now())
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrApplicationNotFound
	}
	if err != nil {
		return nil, err
	}
	return s.applicationView(ctx, application)
}

func (s *Service) applicationView(ctx context.Context, application *Application) (*ApplicationView, error) {
	credential, err := s.repo.GetCredentialByApplication(ctx, application.ID, application.UserID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("get verification application credential: %w", err)
	}
	application.Credential = credential
	return &ApplicationView{
		ID:            application.ID,
		School:        SchoolReference{Code: application.SchoolCode, Name: application.SchoolName},
		Status:        application.Status,
		CurrentMethod: application.CurrentMethod,
		TerminalCode:  application.TerminalCode,
		Revision:      application.Revision,
		NextActions:   nextApplicationActions(application),
		Credential:    credential,
		CreatedAt:     application.CreatedAt,
		UpdatedAt:     application.UpdatedAt,
		ExpiresAt:     application.ExpiresAt,
	}, nil
}

func nextApplicationActions(application *Application) []string {
	if application == nil {
		return []string{}
	}
	switch application.Status {
	case ApplicationCreated:
		return []string{"choose_method"}
	case ApplicationInProgress:
		return []string{"retry_current_method", "choose_another_method", "open_account_recovery"}
	case ApplicationPendingManualReview:
		return []string{"wait_for_review"}
	case ApplicationApproved:
		return []string{"return_to_consumer"}
	case ApplicationRejected, ApplicationExpired:
		return []string{"choose_another_method", "open_account_recovery"}
	default:
		return []string{}
	}
}

func (s *Service) VerifyRealName(ctx context.Context, input VerifyRealNameInput) (*ApplicationView, error) {
	application, err := s.repo.GetApplication(ctx, input.ApplicationID, input.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrApplicationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get real-name verification application: %w", err)
	}
	now := s.now()
	if !applicationIsMutable(application, now) {
		if !application.ExpiresAt.After(now) {
			return nil, ErrApplicationExpired
		}
		return nil, ErrApplicationState
	}
	config, err := s.repo.GetMethodConfig(ctx, application.SchoolCode, MethodRealNameIdentityCheck)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrMethodUnavailable
	}
	if err != nil {
		return nil, fmt.Errorf("load real-name method: %w", err)
	}
	if !methodHealthy(config) || config.AdapterID != BUAAAdapterID || config.RosterDependency != "required" {
		return nil, ErrMethodUnavailable
	}
	if !input.SensitiveDataConsent || config.PrivacyNoticeVersion == "" || input.PrivacyNoticeVersion != config.PrivacyNoticeVersion {
		return nil, ErrConsentRequired
	}
	studentID, studentOK := s.buaa.NormalizeStudentID(input.StudentID)
	name, nameOK := s.buaa.NormalizeName(input.Name)
	documentNumber, documentOK := s.buaa.NormalizeMainlandDocumentNumber(input.DocumentNumber)
	if !studentOK || !nameOK || !documentOK {
		return nil, s.failAttempt(ctx, application, config, "information_mismatch", ErrInformationMismatch, nil, nil, now)
	}
	studentIDHash, err := ComputeRosterBlindIndex(s.hmacKey, config.SchoolID, BlindIndexStudentID, studentID)
	if err != nil {
		return nil, err
	}
	nameHash, err := ComputeRosterBlindIndex(s.hmacKey, config.SchoolID, BlindIndexName, name)
	if err != nil {
		return nil, err
	}
	documentHash, err := ComputeRosterBlindIndex(s.hmacKey, config.SchoolID, BlindIndexDocumentNumber, documentNumber)
	if err != nil {
		return nil, err
	}
	subjectHash, err := ComputeRosterBlindIndex(s.hmacKey, config.SchoolID, BlindIndexSubject, studentID)
	if err != nil {
		return nil, err
	}

	state, err := s.repo.GetActiveRosterState(ctx, config.SchoolID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, s.failAttempt(ctx, application, config, "roster_unavailable", ErrDependencyUnavailable, nil, nil, now)
	}
	if err != nil {
		return nil, fmt.Errorf("get active roster state: %w", err)
	}
	if !snapshotFresh(state.SourceCutoffAt, config.SnapshotHardExpiry, now) {
		return nil, s.failAttempt(
			ctx, application, config, "roster_stale", ErrDependencyUnavailable,
			&state.SnapshotID, &state.SnapshotRevision, now,
		)
	}
	record, err := s.repo.GetActiveRosterRecord(ctx, config.SchoolID, studentIDHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, s.failAttempt(
			ctx, application, config, "information_mismatch", ErrInformationMismatch,
			&state.SnapshotID, &state.SnapshotRevision, now,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("lookup active roster record: %w", err)
	}
	if !s.realNameRecordMatches(record, config, nameHash, documentHash, now) {
		return nil, s.failAttempt(
			ctx, application, config, "information_mismatch", ErrInformationMismatch,
			&record.SnapshotID, &record.SnapshotRevision, now,
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
		lockedConfig, err := s.repo.GetMethodConfigTx(ctx, tx, application.SchoolCode, MethodRealNameIdentityCheck)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("lock real-name method configuration: %w", err)
		}
		if errors.Is(err, pgx.ErrNoRows) || !methodHealthy(lockedConfig) ||
			lockedConfig.AdapterID != BUAAAdapterID ||
			lockedConfig.RosterDependency != "required" ||
			lockedConfig.PrivacyNoticeVersion != input.PrivacyNoticeVersion {
			outcome = ErrMethodUnavailable
			return s.repo.insertAttemptAndProgressTx(ctx, tx, lockedApplication, attemptResultFor(
				config, "unavailable", "method_configuration_changed", &record.SnapshotID,
				&record.SnapshotRevision, input.PrivacyNoticeVersion, now,
			), now)
		}
		lockedRecord, err := s.repo.GetActiveRosterRecordTx(ctx, tx, config.SchoolID, studentIDHash)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("lock active roster record: %w", err)
		}
		if errors.Is(err, pgx.ErrNoRows) || !s.realNameRecordMatches(lockedRecord, lockedConfig, nameHash, documentHash, now) {
			outcome = ErrInformationMismatch
			return s.repo.insertAttemptAndProgressTx(ctx, tx, lockedApplication, attemptResultFor(
				lockedConfig, "failed", "information_mismatch", &record.SnapshotID,
				&record.SnapshotRevision, input.PrivacyNoticeVersion, now,
			), now)
		}
		if err := s.repo.LockEnrollmentSubjectTx(ctx, tx, config.SchoolID, subjectHash); err != nil {
			return err
		}
		subject, err := s.repo.GetActiveEnrollmentSubjectTx(ctx, tx, config.SchoolID, subjectHash)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			subjectID, idErr := newID()
			if idErr != nil {
				return idErr
			}
			subject = &EnrollmentSubject{
				ID: subjectID, UserID: input.UserID, SchoolID: config.SchoolID,
				SubjectHash: subjectHash, StudentIDHash: studentIDHash,
				StudentDisplay: maskStudentID(studentID), BindingStatus: "active",
			}
			if err := s.repo.CreateEnrollmentSubjectTx(
				ctx, tx, *subject, &documentHash, MethodRealNameIdentityCheck,
				&lockedRecord.SnapshotID, &lockedRecord.SnapshotRevision, now,
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
				lockedConfig, "failed", "subject_conflict", &lockedRecord.SnapshotID,
				&lockedRecord.SnapshotRevision, input.PrivacyNoticeVersion, now,
			), now); err != nil {
				return err
			}
			return s.repo.BumpEligibilityRevisionTx(ctx, tx, input.UserID, config.SchoolID, "subject_conflict", now)
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
			ID: credentialID, UserID: input.UserID, SchoolID: config.SchoolID,
			Method: MethodRealNameIdentityCheck, Status: CredentialActive,
			CredentialClass: "formal_student", SubjectHash: subjectHash,
			SubjectDisplay: maskStudentID(studentID), EnrollmentID: &subject.ID,
			RosterDependency: "required", VerifiedAt: now, ExpiresAt: expiresAt, Revision: 1,
		}
		metadata := json.RawMessage(`{"evidence_path":"local_roster_hmac_match","roster_satisfied":true}`)
		if err := s.repo.CreateCredentialTx(
			ctx, tx, credential, lockedApplication.ID, lockedConfig.AdapterID,
			lockedConfig.AdapterVersion, &lockedRecord.SnapshotID,
			&lockedRecord.SnapshotRevision, metadata, now,
		); err != nil {
			return err
		}
		if err := s.repo.CompleteApplicationTx(ctx, tx, lockedApplication, attemptResultFor(
			lockedConfig, "succeeded", "verified", &lockedRecord.SnapshotID,
			&lockedRecord.SnapshotRevision, input.PrivacyNoticeVersion, now,
		), now); err != nil {
			return err
		}
		return s.repo.BumpEligibilityRevisionTx(ctx, tx, input.UserID, config.SchoolID, "credential_activated", now)
	})
	if err != nil {
		return nil, err
	}
	if outcome != nil {
		return nil, outcome
	}
	return s.GetApplication(ctx, input.UserID, application.ID)
}

func (s *Service) failAttempt(
	ctx context.Context,
	application *Application,
	config *MethodConfig,
	resultCode string,
	resultErr error,
	snapshotID *string,
	snapshotRevision *int64,
	now time.Time,
) error {
	result := attemptResultFor(
		config,
		map[bool]string{true: "unavailable", false: "failed"}[errors.Is(resultErr, ErrDependencyUnavailable)],
		resultCode,
		snapshotID,
		snapshotRevision,
		config.PrivacyNoticeVersion,
		now,
	)
	if err := s.repo.RecordAttempt(ctx, application.UserID, application.ID, result, now); err != nil {
		return fmt.Errorf("%w: record verification attempt: %v", resultErr, err)
	}
	return resultErr
}

func attemptResultFor(
	config *MethodConfig,
	status string,
	resultCode string,
	snapshotID *string,
	snapshotRevision *int64,
	privacyNotice string,
	now time.Time,
) attemptResult {
	return attemptResult{
		Status: status, ResultCode: resultCode, Method: config.Method,
		AdapterID: config.AdapterID, AdapterVersion: config.AdapterVersion,
		SnapshotID: snapshotID, SnapshotRevision: snapshotRevision,
		PrivacyNotice: &privacyNotice, ConsentedAt: &now,
	}
}

func (s *Service) realNameRecordMatches(
	record *RosterRecord,
	config *MethodConfig,
	nameHash string,
	documentHash string,
	now time.Time,
) bool {
	if record == nil || config == nil || record.HMACKeyVersion != RosterHMACKeyVersion {
		return false
	}
	if record.EligibilityStatus != "eligible" || !snapshotFresh(record.SourceCutoffAt, config.SnapshotHardExpiry, now) {
		return false
	}
	if record.DocumentType == nil || record.DocumentNumberHash == nil ||
		!s.buaa.SupportsMainlandDocumentType(*record.DocumentType, config.EnrollmentPolicy) {
		return false
	}
	return constantTimeStringEqual(record.NameHash, nameHash) &&
		constantTimeStringEqual(*record.DocumentNumberHash, documentHash)
}

func constantTimeStringEqual(left, right string) bool {
	return subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}

func methodHealthy(config *MethodConfig) bool {
	return config != nil && (config.HealthStatus == "healthy" || config.HealthStatus == "degraded")
}

func snapshotFresh(sourceCutoff time.Time, hardExpiry time.Duration, now time.Time) bool {
	if hardExpiry <= 0 || sourceCutoff.IsZero() || sourceCutoff.After(now.Add(allowedSnapshotFutureSkew)) {
		return false
	}
	return now.Sub(sourceCutoff) <= hardExpiry
}

func applicationIsMutable(application *Application, now time.Time) bool {
	if application == nil || !application.ExpiresAt.After(now) {
		return false
	}
	return application.Status == ApplicationCreated || application.Status == ApplicationInProgress
}

func applicationIsNonTerminal(application *Application) bool {
	if application == nil {
		return false
	}
	return application.Status == ApplicationCreated ||
		application.Status == ApplicationInProgress ||
		application.Status == ApplicationPendingManualReview
}

func (s *Service) ListCredentials(ctx context.Context, userID int64) ([]Credential, error) {
	credentials, err := s.repo.ListCredentials(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list verification credentials: %w", err)
	}
	if credentials == nil {
		return []Credential{}, nil
	}
	return credentials, nil
}

func (s *Service) RevokeCredential(ctx context.Context, userID int64, credentialID string) (*Credential, error) {
	credential, err := s.repo.RevokeCredential(ctx, credentialID, userID, s.now())
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCredentialNotFound
	}
	if err != nil {
		return nil, err
	}
	return credential, nil
}

func (s *Service) GetEligibility(ctx context.Context, userID int64, schoolCode string) (*Eligibility, error) {
	if userID <= 0 || !schoolCodePattern.MatchString(schoolCode) {
		return nil, ErrSchoolNotFound
	}
	schoolID, _, err := s.repo.GetSchoolByCode(ctx, schoolCode)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSchoolNotFound
	}
	if err != nil {
		return nil, err
	}
	now := s.now()
	credentials, err := s.repo.ListQualifyingCredentials(ctx, userID, schoolID, now)
	if err != nil {
		return nil, fmt.Errorf("derive student eligibility: %w", err)
	}
	revision, err := s.repo.GetEligibilityRevision(ctx, userID, schoolID)
	if err != nil {
		return nil, fmt.Errorf("read student eligibility revision: %w", err)
	}
	result := &Eligibility{
		Eligible: false, SchoolCode: schoolCode, CredentialMethods: []Method{},
		EvaluatedAt: now, Revision: revision,
	}
	if len(credentials) == 0 {
		return result, nil
	}
	result.Eligible = true
	credentialClass := credentials[0].CredentialClass
	for _, credential := range credentials {
		if !slices.Contains(result.CredentialMethods, credential.Method) {
			result.CredentialMethods = append(result.CredentialMethods, credential.Method)
		}
		if credential.CredentialClass == "formal_student" {
			credentialClass = credential.CredentialClass
		}
		if credential.ExpiresAt == nil {
			result.ExpiresAt = nil
			break
		}
		if result.ExpiresAt == nil || credential.ExpiresAt.After(*result.ExpiresAt) {
			expires := *credential.ExpiresAt
			result.ExpiresAt = &expires
		}
	}
	result.CredentialClass = &credentialClass
	return result, nil
}

// GetEligibilityForSchoolID is the internal domain boundary used by consumers
// such as group admission. It deliberately exposes only the derived decision
// and revision; callers do not receive verification evidence or roster data.
func (s *Service) GetEligibilityForSchoolID(ctx context.Context, userID, schoolID int64) (*Eligibility, error) {
	if userID <= 0 || schoolID <= 0 {
		return nil, ErrSchoolNotFound
	}
	schoolCode, err := s.repo.GetSchoolCodeByID(ctx, schoolID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSchoolNotFound
	}
	if err != nil {
		return nil, err
	}
	return s.GetEligibility(ctx, userID, schoolCode)
}

func (s *Service) GetCurrentStudentStatus(ctx context.Context, userID int64) (*CurrentStudentStatus, error) {
	if userID <= 0 {
		return nil, ErrSchoolNotFound
	}
	status, err := s.repo.GetCurrentStudentStatus(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("derive current student status: %w", err)
	}
	return status, nil
}
