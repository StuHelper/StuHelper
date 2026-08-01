package user

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"

	"github.com/StuHelper/StuHelper/server/internal/pkg/ctxutil"
	"github.com/StuHelper/StuHelper/server/internal/pkg/schoolauth"
)

// #nosec G101 -- this is a credential kind identifier, not a credential value.
const userVerificationCredentialKindSchoolEmailOTP = "school_email_otp"

const (
	studentEmailOTPLength          = 6
	studentEmailOTPMinLength       = 4
	studentEmailOTPMaxLength       = 12
	studentEmailOTPTTL             = 5 * time.Minute
	studentEmailOTPCooldown        = time.Minute
	studentEmailOTPCooldownSeconds = int(studentEmailOTPCooldown / time.Second)
	studentEmailOTPMaxAttempts     = 5
	studentEmailOTPCleanupTimeout  = 5 * time.Second
)

const (
	studentEmailOTPKeyPrefix      = "user:student_email_otp:"
	studentEmailOTPCooldownSuffix = ":cooldown"
	studentEmailOTPAttemptsSuffix = ":attempts"
)

type StudentEmailOTPInput struct {
	UserID      int64
	SchoolID    int64
	Email       string
	StudentID   string
	StudentName string
}

type StudentEmailOTPVerifyInput struct {
	UserID   int64
	SchoolID int64
	Email    string
	Code     string
	Consent  bool
}

type StudentEmailAcademicMatchInput struct {
	UserID      int64
	SchoolID    int64
	StudentID   string
	StudentName string
}

type StudentEmailAcademicMatchResponse struct {
	Matched   bool   `json:"matched"`
	Email     string `json:"email,omitempty"`
	StudentID string `json:"studentID,omitempty"`
	Message   string `json:"message,omitempty"`
}

type StudentEmailOTPResponse struct {
	Email           string `json:"email"`
	StudentID       string `json:"studentID,omitempty"`
	CooldownSeconds int    `json:"cooldownSeconds"`
}

type studentEmailOTPRecord struct {
	Email       string `json:"email"`
	Code        string `json:"code"`
	StudentID   string `json:"studentID,omitempty"`
	StudentName string `json:"studentName,omitempty"`
}

func (s *Service) MatchStudentEmailAcademicStudent(
	ctx context.Context,
	input StudentEmailAcademicMatchInput,
) (*StudentEmailAcademicMatchResponse, error) {
	if err := validateUserID(input.UserID); err != nil {
		return nil, err
	}
	existing, err := s.repo.GetProfileByUserID(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("MatchStudentEmailAcademicStudent check existing: %w", err)
	}
	if err := validateStudentVerificationTransition(existing); err != nil {
		return nil, err
	}
	school, err := s.loadEnabledSchoolConfig(ctx, input.SchoolID)
	if err != nil {
		return nil, fmt.Errorf("MatchStudentEmailAcademicStudent get school config: %w", err)
	}
	settings := schoolauth.ParseAdmissionSettings(school.ManualFormFields)
	if settings.EmailIdentityPolicy == nil || !settings.EmailIdentityPolicy.IsAcademicStudentEmail() {
		return nil, ErrStudentEmailDomainNotAllowed
	}
	_, email, studentID, _, err := s.resolveAcademicStudentEmailOTPIdentity(
		ctx,
		school,
		settings.EmailIdentityPolicy,
		StudentEmailOTPInput{
			UserID:      input.UserID,
			SchoolID:    input.SchoolID,
			StudentID:   input.StudentID,
			StudentName: input.StudentName,
		},
	)
	if err != nil {
		if errors.Is(err, ErrStudentNotFound) || errors.Is(err, ErrStudentNameMismatch) {
			return &StudentEmailAcademicMatchResponse{
				Matched: false,
				Message: "学号和姓名不匹配，请核对后再发送验证码。",
			}, nil
		}
		return nil, err
	}
	return &StudentEmailAcademicMatchResponse{
		Matched:   true,
		Email:     email,
		StudentID: studentID,
		Message:   "学号和姓名已匹配。",
	}, nil
}

func (s *Service) RequestStudentEmailOTP(ctx context.Context, input StudentEmailOTPInput) (*StudentEmailOTPResponse, error) {
	if err := validateUserID(input.UserID); err != nil {
		return nil, err
	}
	if err := s.requireStudentEmailOTPDependencies(); err != nil {
		return nil, err
	}
	existing, err := s.repo.GetProfileByUserID(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("RequestStudentEmailOTP check existing: %w", err)
	}
	if err := validateStudentVerificationTransition(existing); err != nil {
		return nil, err
	}
	school, email, studentID, studentName, err := s.resolveStudentEmailOTPIdentity(ctx, input)
	if err != nil {
		return nil, err
	}
	settings := schoolauth.ParseAdmissionSettings(school.ManualFormFields)
	if !schoolauth.EmailDomainAllowed(email, settings.EmailDomains) {
		return nil, ErrStudentEmailDomainNotAllowed
	}
	code, err := s.issueStudentEmailOTP(ctx, input.UserID, input.SchoolID, email, studentID, studentName)
	if err != nil {
		return nil, err
	}
	if err := s.studentEmailSender.SendStudentVerificationOTP(ctx, email, code); err != nil {
		if cleanupErr := s.cleanupStudentEmailOTPCodeOnlyAfterSendFailure(ctx, input.UserID, input.SchoolID); cleanupErr != nil {
			return nil, fmt.Errorf("RequestStudentEmailOTP send: %w; cleanup: %w", err, cleanupErr)
		}
		return nil, fmt.Errorf("RequestStudentEmailOTP send: %w", err)
	}
	return &StudentEmailOTPResponse{
		Email:           email,
		StudentID:       studentID,
		CooldownSeconds: studentEmailOTPCooldownSeconds,
	}, nil
}

func (s *Service) VerifyStudentEmailOTP(ctx context.Context, input StudentEmailOTPVerifyInput) (*Profile, error) {
	if err := validateUserID(input.UserID); err != nil {
		return nil, err
	}
	if s.redisClient == nil {
		return nil, ErrStudentEmailRedisUnavailable
	}
	if !input.Consent {
		return nil, ErrConsentRequired
	}
	existing, err := s.repo.GetProfileByUserID(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("VerifyStudentEmailOTP check existing: %w", err)
	}
	if err := validateStudentVerificationTransition(existing); err != nil {
		return nil, err
	}
	school, err := s.loadEnabledSchoolConfig(ctx, input.SchoolID)
	if err != nil {
		return nil, fmt.Errorf("VerifyStudentEmailOTP get school config: %w", err)
	}
	record, err := s.loadStudentEmailOTPRecord(ctx, input.UserID, input.SchoolID)
	if err != nil {
		return nil, err
	}
	email := strings.TrimSpace(input.Email)
	if email == "" {
		email = record.Email
	} else {
		email, err = normalizeStudentEmail(email)
		if err != nil {
			return nil, err
		}
	}
	code, err := normalizeStudentEmailOTPCode(input.Code)
	if err != nil {
		return nil, err
	}
	check := studentEmailOTPCheckInput{
		UserID: input.UserID, SchoolID: input.SchoolID, Email: email, Code: code, Record: record,
	}
	if err := s.checkStudentEmailOTP(ctx, check); err != nil {
		return nil, err
	}

	now := time.Now()
	schoolID := input.SchoolID
	method := VerifyMethodSchoolEmailOTP
	profile := &Profile{
		UserID:             input.UserID,
		SchoolID:           &schoolID,
		VerificationStatus: StatusPending,
		VerificationMethod: &method,
		StudentIDs:         []string{},
		ConsentGivenAt:     &now,
	}
	if record.StudentID != "" {
		studentID := record.StudentID
		profile.StudentIDs = []string{studentID}
		profile.ActiveStudentID = &studentID
	}
	if school.IsAutoApprove() {
		profile.VerificationStatus = StatusVerified
		profile.VerifiedAt = &now
	}
	profile.ManualFormData = studentEmailManualFormData(record)

	if err := s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		txExisting, err := s.repo.GetProfileByUserIDForUpdateTx(ctx, tx, input.UserID)
		if err != nil {
			return fmt.Errorf("VerifyStudentEmailOTP check existing tx: %w", err)
		}
		if err := validateStudentVerificationTransition(txExisting); err != nil {
			return err
		}
		if txExisting != nil {
			profile.CreatedAt = txExisting.CreatedAt
			if err := s.repo.UpdateProfileTx(ctx, tx, profile); err != nil {
				return fmt.Errorf("VerifyStudentEmailOTP update profile tx: %w", err)
			}
		} else {
			if err := s.repo.CreateProfileTx(ctx, tx, profile); err != nil {
				return fmt.Errorf("VerifyStudentEmailOTP create profile tx: %w", err)
			}
		}
		if err := s.ensureProfileVerificationCredentialTx(ctx, tx, profile); err != nil {
			return fmt.Errorf("VerifyStudentEmailOTP ensure credential: %w", err)
		}
		if err := s.enqueueVerificationProjectionTx(ctx, tx, input.UserID, profile.VerificationStatus); err != nil {
			return fmt.Errorf("VerifyStudentEmailOTP enqueue projections: %w", err)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	if err := s.cleanupStudentEmailOTPCode(ctx, input.UserID, input.SchoolID); err != nil {
		return nil, fmt.Errorf("VerifyStudentEmailOTP cleanup code: %w", err)
	}

	result, err := s.repo.GetProfileByUserID(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("VerifyStudentEmailOTP reload: %w", err)
	}
	if result != nil {
		if err := s.hydrateProfilePhone(result); err != nil {
			return nil, fmt.Errorf("VerifyStudentEmailOTP hydrate profile phone: %w", err)
		}
	}
	return result, nil
}

func (s *Service) resolveStudentEmailOTPIdentity(
	ctx context.Context,
	input StudentEmailOTPInput,
) (*SchoolConfig, string, string, string, error) {
	school, err := s.loadEnabledSchoolConfig(ctx, input.SchoolID)
	if err != nil {
		return nil, "", "", "", err
	}
	settings := schoolauth.ParseAdmissionSettings(school.ManualFormFields)
	if settings.EmailIdentityPolicy != nil && settings.EmailIdentityPolicy.IsAcademicStudentEmail() {
		return s.resolveAcademicStudentEmailOTPIdentity(ctx, school, settings.EmailIdentityPolicy, input)
	}
	email, err := normalizeStudentEmail(input.Email)
	if err != nil {
		return nil, "", "", "", err
	}
	return school, email, "", "", nil
}

func (s *Service) resolveAcademicStudentEmailOTPIdentity(
	ctx context.Context,
	school *SchoolConfig,
	policy *schoolauth.EmailIdentityPolicy,
	input StudentEmailOTPInput,
) (*SchoolConfig, string, string, string, error) {
	studentID := schoolauth.NormalizeStudentID(input.StudentID)
	if studentID == "" {
		return nil, "", "", "", ErrStudentIDRequired
	}
	if !schoolauth.IsValidStudentID(studentID) {
		return nil, "", "", "", ErrStudentIDInvalid
	}
	studentName := schoolauth.NormalizeAcademicName(input.StudentName)
	if policy.RequireStudentName && studentName == "" {
		return nil, "", "", "", ErrStudentNameRequired
	}
	if policy.RequireStudentName && !schoolauth.IsValidAcademicName(studentName) {
		return nil, "", "", "", ErrStudentNameInvalid
	}
	student, err := s.lookupAcademicStudentForSchool(ctx, school, studentID)
	if err != nil {
		return nil, "", "", "", fmt.Errorf("resolveAcademicStudentEmailOTPIdentity lookup: %w", err)
	}
	if student == nil {
		return nil, "", "", "", ErrStudentNotFound
	}
	if policy.RequireStudentName {
		recordName := ""
		if student.XM != nil {
			recordName = schoolauth.NormalizeAcademicName(*student.XM)
		}
		if recordName == "" || recordName != studentName {
			return nil, "", "", "", ErrStudentNameMismatch
		}
	}
	email := schoolauth.DeriveStudentEmail(studentID, policy.StudentIDEmailDomain)
	if email == "" {
		return nil, "", "", "", ErrStudentEmailDomainNotAllowed
	}
	if input.Email != "" {
		inputEmail, err := normalizeStudentEmail(input.Email)
		if err != nil {
			return nil, "", "", "", err
		}
		if inputEmail != email {
			return nil, "", "", "", ErrStudentEmailDomainNotAllowed
		}
	}
	return school, email, studentID, studentName, nil
}

func (s *Service) requireStudentEmailOTPDependencies() error {
	if s.redisClient == nil {
		return ErrStudentEmailRedisUnavailable
	}
	if s.studentEmailSender == nil {
		return ErrStudentEmailSenderUnavailable
	}
	return nil
}

func (s *Service) issueStudentEmailOTP(
	ctx context.Context,
	userID int64,
	schoolID int64,
	email string,
	studentID string,
	studentName string,
) (string, error) {
	if err := s.reserveStudentEmailOTPCooldown(ctx, userID, schoolID); err != nil {
		return "", err
	}
	code, err := s.generateOTP()
	if err != nil {
		return "", err
	}
	record := studentEmailOTPRecord{
		Email:       email,
		Code:        code,
		StudentID:   studentID,
		StudentName: studentName,
	}
	if err := s.storeStudentEmailOTPRecord(ctx, studentEmailOTPStoreInput{
		UserID: userID, SchoolID: schoolID, Record: record,
	}); err != nil {
		return "", err
	}
	return code, nil
}

func (s *Service) reserveStudentEmailOTPCooldown(ctx context.Context, userID, schoolID int64) error {
	key := studentEmailOTPKey(userID, schoolID) + studentEmailOTPCooldownSuffix
	result, err := s.redisClient.SetArgs(ctx, key, "1", redis.SetArgs{
		Mode: "NX",
		TTL:  studentEmailOTPCooldown,
	}).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return ErrStudentEmailOTPCooldown
		}
		return fmt.Errorf("%w: reserve student email otp cooldown: %w", ErrStudentEmailRedisUnavailable, err)
	}
	if result != "OK" {
		return ErrStudentEmailOTPCooldown
	}
	return nil
}

func (s *Service) storeStudentEmailOTPRecord(ctx context.Context, input studentEmailOTPStoreInput) error {
	payload, err := json.Marshal(input.Record)
	if err != nil {
		return err
	}
	pipe := s.redisClient.Pipeline()
	pipe.Set(ctx, studentEmailOTPKey(input.UserID, input.SchoolID), payload, studentEmailOTPTTL)
	pipe.Del(ctx, studentEmailOTPAttemptsKey(input.UserID, input.SchoolID))
	_, err = pipe.Exec(ctx)
	return err
}

func (s *Service) loadStudentEmailOTPRecord(ctx context.Context, userID, schoolID int64) (studentEmailOTPRecord, error) {
	raw, err := s.redisClient.Get(ctx, studentEmailOTPKey(userID, schoolID)).Bytes()
	if errors.Is(err, redis.Nil) {
		return studentEmailOTPRecord{}, ErrStudentEmailOTPExpired
	}
	if err != nil {
		return studentEmailOTPRecord{}, fmt.Errorf("loadStudentEmailOTPRecord: %w", err)
	}
	var record studentEmailOTPRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		return studentEmailOTPRecord{}, err
	}
	return record, nil
}

func (s *Service) checkStudentEmailOTP(ctx context.Context, input studentEmailOTPCheckInput) error {
	codeMatches := subtle.ConstantTimeCompare([]byte(input.Record.Code), []byte(input.Code)) == 1
	if input.Record.Email != input.Email || !codeMatches {
		return s.recordStudentEmailOTPFailure(ctx, input.UserID, input.SchoolID)
	}
	return nil
}

func (s *Service) recordStudentEmailOTPFailure(ctx context.Context, userID, schoolID int64) error {
	key := studentEmailOTPAttemptsKey(userID, schoolID)
	attempts, err := s.redisClient.Incr(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("recordStudentEmailOTPFailure: %w", err)
	}
	if attempts == 1 {
		if err := s.redisClient.Expire(ctx, key, studentEmailOTPTTL).Err(); err != nil {
			return fmt.Errorf("recordStudentEmailOTPFailure expire attempts: %w", err)
		}
	}
	if attempts >= studentEmailOTPMaxAttempts {
		if err := s.cleanupStudentEmailOTPCode(ctx, userID, schoolID); err != nil {
			return fmt.Errorf("recordStudentEmailOTPFailure cleanup: %w", err)
		}
		return ErrStudentEmailOTPMaxAttempts
	}
	return ErrStudentEmailOTPInvalid
}

func (s *Service) cleanupStudentEmailOTPCode(ctx context.Context, userID, schoolID int64) error {
	return s.redisClient.Del(
		ctx,
		studentEmailOTPKey(userID, schoolID),
		studentEmailOTPAttemptsKey(userID, schoolID),
		studentEmailOTPKey(userID, schoolID)+studentEmailOTPCooldownSuffix,
	).Err()
}

func (s *Service) cleanupStudentEmailOTPCodeOnly(ctx context.Context, userID, schoolID int64) error {
	return s.redisClient.Del(
		ctx,
		studentEmailOTPKey(userID, schoolID),
		studentEmailOTPAttemptsKey(userID, schoolID),
	).Err()
}

func (s *Service) cleanupStudentEmailOTPCodeOnlyAfterSendFailure(ctx context.Context, userID, schoolID int64) error {
	cleanupCtx, cancel := studentEmailOTPCleanupContext(ctx)
	defer cancel()
	return s.cleanupStudentEmailOTPCodeOnly(cleanupCtx, userID, schoolID)
}

func studentEmailOTPCleanupContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return ctxutil.DetachedTimeout(ctx, studentEmailOTPCleanupTimeout)
}

type studentEmailOTPStoreInput struct {
	UserID   int64
	SchoolID int64
	Record   studentEmailOTPRecord
}

type studentEmailOTPCheckInput struct {
	UserID   int64
	SchoolID int64
	Email    string
	Code     string
	Record   studentEmailOTPRecord
}

func studentEmailOTPKey(userID, schoolID int64) string {
	return studentEmailOTPKeyPrefix + strconv.FormatInt(userID, 10) + ":" + strconv.FormatInt(schoolID, 10)
}

func studentEmailOTPAttemptsKey(userID, schoolID int64) string {
	return studentEmailOTPKey(userID, schoolID) + studentEmailOTPAttemptsSuffix
}

func normalizeStudentEmail(value string) (string, error) {
	email := schoolauth.NormalizeEmailAddress(value)
	if email == "" {
		return "", ErrStudentEmailDomainNotAllowed
	}
	return email, nil
}

func normalizeStudentEmailOTPCode(value string) (string, error) {
	code := strings.TrimSpace(value)
	length := utf8.RuneCountInString(code)
	if length < studentEmailOTPMinLength || length > studentEmailOTPMaxLength {
		return "", ErrStudentEmailOTPInvalid
	}
	return code, nil
}

func studentEmailManualFormData(record studentEmailOTPRecord) json.RawMessage {
	fields := map[string]string{
		"schoolEmail": record.Email,
	}
	if record.StudentID != "" {
		fields["studentID"] = record.StudentID
	}
	if record.StudentName != "" {
		fields["studentName"] = record.StudentName
	}
	raw, err := json.Marshal(fields)
	if err != nil {
		return nil
	}
	return raw
}

func (s *Service) ensureProfileVerificationCredentialTx(ctx context.Context, tx pgx.Tx, profile *Profile) error {
	if profile == nil || profile.VerificationStatus != StatusVerified {
		return nil
	}
	if profile.SchoolID == nil || profile.VerificationMethod == nil {
		return nil
	}
	switch *profile.VerificationMethod {
	case VerifyMethodSchoolEmailOTP:
		email, err := schoolEmailFromProfile(profile.ManualFormData)
		if err != nil {
			return err
		}
		if email == "" {
			return fmt.Errorf("school email otp profile is verified but schoolEmail is missing")
		}
		verifiedAt := time.Now()
		if profile.VerifiedAt != nil {
			verifiedAt = *profile.VerifiedAt
		}
		return s.repo.EnsureVerificationCredentialTx(ctx, tx, VerificationCredentialProjection{
			UserID:         profile.UserID,
			SchoolID:       *profile.SchoolID,
			Kind:           userVerificationCredentialKindSchoolEmailOTP,
			SubjectHash:    s.hashVerificationCredentialSubject(*profile.SchoolID, email),
			SubjectDisplay: maskStudentEmail(email),
			VerifiedAt:     verifiedAt,
		})
	default:
		return nil
	}
}

func schoolEmailFromProfile(raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		return "", nil
	}
	var fields map[string]string
	if err := json.Unmarshal(raw, &fields); err != nil {
		return "", fmt.Errorf("decode student profile manual form data: %w", err)
	}
	email := strings.TrimSpace(fields["schoolEmail"])
	if email == "" {
		return "", nil
	}
	return normalizeStudentEmail(email)
}

func (s *Service) hashVerificationCredentialSubject(schoolID int64, subject string) string {
	mac := hmac.New(sha256.New, s.hmacKey)
	mac.Write([]byte(strconv.FormatInt(schoolID, 10) + "|" + strings.TrimSpace(subject)))
	return hex.EncodeToString(mac.Sum(nil))
}

func maskStudentEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 || parts[0] == "" {
		return email
	}
	return maskStudentEmailLocal(parts[0]) + "@" + parts[1]
}

func maskStudentEmailLocal(local string) string {
	runes := []rune(local)
	if len(runes) <= 2 {
		return string(runes[0]) + "*"
	}
	return string(runes[0]) + strings.Repeat("*", len(runes)-2) + string(runes[len(runes)-1])
}

func generateStudentEmailOTPCode() (string, error) {
	maxValue := new(big.Int).Exp(big.NewInt(10), big.NewInt(studentEmailOTPLength), nil)
	n, err := rand.Int(rand.Reader, maxValue)
	if err != nil {
		return "", fmt.Errorf("generate student email otp: %w", err)
	}
	return fmt.Sprintf("%0*d", studentEmailOTPLength, n), nil
}
