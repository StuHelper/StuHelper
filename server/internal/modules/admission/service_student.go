package admission

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/redis/go-redis/v9"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/ctxutil"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/schoolauth"
)

const (
	admissionEmailOTPLength          = 6
	admissionEmailOTPMinLength       = 4
	admissionEmailOTPMaxLength       = 12
	admissionEmailOTPTTL             = 5 * time.Minute
	admissionEmailOTPCooldown        = time.Minute
	admissionEmailOTPCooldownSeconds = int(admissionEmailOTPCooldown / time.Second)
	admissionEmailOTPMaxAttempts     = 5
	admissionEmailOTPCleanupTimeout  = 5 * time.Second
)

const (
	admissionEmailOTPKeyPrefix      = "admission:email_otp:"
	admissionEmailOTPCooldownSuffix = ":cooldown"
	admissionEmailOTPAttemptsSuffix = ":attempts"
)

type admissionEmailOTPRecord struct {
	Email       string `json:"email"`
	Code        string `json:"code"`
	StudentID   string `json:"studentID,omitempty"`
	StudentName string `json:"studentName,omitempty"`
}

func (s *Service) MatchSchoolEmailAcademicStudent(
	ctx context.Context,
	input SchoolEmailAcademicMatchInput,
) (*SchoolEmailAcademicMatchResponse, error) {
	if err := validateAdmissionUserAndSchool(input.UserID, input.SchoolID); err != nil {
		return nil, err
	}
	if _, err := s.requireLinkedSession(ctx, input.UserID, input.AdmissionSessionID); err != nil {
		return nil, err
	}
	config, err := s.loadAcademicEmailOTPConfig(ctx, input.SchoolID)
	if err != nil {
		return nil, err
	}
	email, studentID, _, err := s.resolveAcademicStudentEmail(ctx, config, SchoolEmailOTPInput{
		UserID:             input.UserID,
		SchoolID:           input.SchoolID,
		AdmissionSessionID: input.AdmissionSessionID,
		StudentID:          input.StudentID,
		StudentName:        input.StudentName,
	})
	if err != nil {
		if errors.Is(err, ErrAdmissionStudentRecordNotFound) ||
			errors.Is(err, ErrAdmissionStudentNameMismatch) {
			return &SchoolEmailAcademicMatchResponse{
				Matched: false,
				Message: "学号和姓名不匹配，请核对后再发送验证码。",
			}, nil
		}
		return nil, err
	}
	return &SchoolEmailAcademicMatchResponse{
		Matched:   true,
		Email:     email,
		StudentID: studentID,
		Message:   "学号和姓名已匹配。",
	}, nil
}

func (s *Service) RequestSchoolEmailOTP(
	ctx context.Context,
	input SchoolEmailOTPInput,
) (*SchoolEmailOTPResponse, error) {
	if err := validateAdmissionUserAndSchool(input.UserID, input.SchoolID); err != nil {
		return nil, err
	}
	if err := s.requireEmailOTPDependencies(); err != nil {
		return nil, err
	}
	if _, err := s.requireLinkedSession(ctx, input.UserID, input.AdmissionSessionID); err != nil {
		return nil, err
	}
	config, email, studentID, studentName, err := s.loadEmailOTPConfig(ctx, input)
	if err != nil {
		return nil, err
	}
	if !schoolauth.EmailDomainAllowed(email, config.EmailDomains) {
		return nil, ErrAdmissionEmailDomainNotAllowed
	}
	code, err := s.issueEmailOTP(ctx, input.UserID, input.SchoolID, email, studentID, studentName)
	if err != nil {
		return nil, err
	}
	if err := s.emailSender.SendAdmissionOTP(ctx, email, code); err != nil {
		if cleanupErr := s.cleanupEmailOTPCodeOnlyAfterSendFailure(ctx, input.UserID, input.SchoolID); cleanupErr != nil {
			return nil, fmt.Errorf("RequestSchoolEmailOTP send: %w; cleanup: %w", err, cleanupErr)
		}
		return nil, fmt.Errorf("RequestSchoolEmailOTP send: %w", err)
	}
	return &SchoolEmailOTPResponse{
		CooldownSeconds: admissionEmailOTPCooldownSeconds,
		Email:           email,
		StudentID:       studentID,
	}, nil
}

func (s *Service) VerifySchoolEmailOTP(ctx context.Context, input SchoolEmailOTPVerifyInput) (*AdmissionSession, error) {
	if err := validateAdmissionUserAndSchool(input.UserID, input.SchoolID); err != nil {
		return nil, err
	}
	if s.redisClient == nil {
		return nil, ErrAdmissionRedisUnavailable
	}
	if _, err := s.requireLinkedSession(ctx, input.UserID, input.AdmissionSessionID); err != nil {
		return nil, err
	}
	email, err := normalizeAdmissionEmail(input.Email)
	if err != nil {
		return nil, err
	}
	code, err := normalizeAdmissionOTPCode(input.Code)
	if err != nil {
		return nil, err
	}
	record, err := s.loadEmailOTPRecord(ctx, input.UserID, input.SchoolID)
	if err != nil {
		return nil, err
	}
	check := emailOTPCheckInput{
		UserID: input.UserID, SchoolID: input.SchoolID, Email: email, Code: code, Record: record,
	}
	if err := s.checkEmailOTP(ctx, check); err != nil {
		return nil, err
	}
	session, err := s.storeStudentCredential(ctx, studentCredentialInput{
		UserID:             input.UserID,
		SchoolID:           input.SchoolID,
		AdmissionSessionID: input.AdmissionSessionID,
		Kind:               CredentialSchoolEmailOTP,
		Subject:            email,
		SubjectDisplay:     maskAdmissionEmail(email),
		StudentID:          record.StudentID,
		StudentName:        record.StudentName,
	})
	if err != nil {
		return nil, err
	}
	if err := s.cleanupEmailOTPCode(ctx, input.UserID, input.SchoolID); err != nil {
		return nil, fmt.Errorf("VerifySchoolEmailOTP cleanup code: %w", err)
	}
	return session, nil
}

func (s *Service) requireEmailOTPDependencies() error {
	if s.redisClient == nil {
		return ErrAdmissionRedisUnavailable
	}
	if s.emailSender == nil {
		return ErrAdmissionEmailSenderUnavailable
	}
	return nil
}

func (s *Service) loadEmailOTPConfig(
	ctx context.Context,
	input SchoolEmailOTPInput,
) (*AdmissionSchoolConfig, string, string, string, error) {
	config, err := s.loadAdmissionEmailOTPConfig(ctx, input.SchoolID)
	if err != nil {
		return nil, "", "", "", err
	}
	if config.EmailIdentityPolicy != nil && strings.EqualFold(
		strings.TrimSpace(config.EmailIdentityPolicy.Type),
		schoolauth.EmailIdentityPolicyAcademicStudentEmail,
	) {
		email, studentID, studentName, err := s.resolveAcademicStudentEmail(ctx, config, input)
		if err != nil {
			return nil, "", "", "", err
		}
		return config, email, studentID, studentName, nil
	}
	email, err := normalizeAdmissionEmail(input.Email)
	if err != nil {
		return nil, "", "", "", err
	}
	return config, email, "", "", nil
}

func (s *Service) loadAdmissionEmailOTPConfig(ctx context.Context, schoolID int64) (*AdmissionSchoolConfig, error) {
	config, err := s.repo.GetAdmissionSchoolConfig(ctx, schoolID)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, ErrAdmissionSchoolNotFound
	}
	if !config.Enabled {
		return nil, ErrAdmissionSchoolDisabled
	}
	return config, nil
}

func (s *Service) loadAcademicEmailOTPConfig(ctx context.Context, schoolID int64) (*AdmissionSchoolConfig, error) {
	config, err := s.loadAdmissionEmailOTPConfig(ctx, schoolID)
	if err != nil {
		return nil, err
	}
	if config.EmailIdentityPolicy == nil || !strings.EqualFold(
		strings.TrimSpace(config.EmailIdentityPolicy.Type),
		schoolauth.EmailIdentityPolicyAcademicStudentEmail,
	) {
		return nil, ErrAdmissionEmailDomainNotAllowed
	}
	return config, nil
}

func (s *Service) resolveAcademicStudentEmail(
	ctx context.Context,
	config *AdmissionSchoolConfig,
	input SchoolEmailOTPInput,
) (string, string, string, error) {
	if s.academicLookup == nil {
		return "", "", "", ErrAdmissionAcademicLookupUnavailable
	}
	studentID := schoolauth.NormalizeStudentID(input.StudentID)
	if studentID == "" {
		return "", "", "", ErrAdmissionStudentIDRequired
	}
	studentName := schoolauth.NormalizeAcademicName(input.StudentName)
	if config.EmailIdentityPolicy.RequireStudentName && studentName == "" {
		return "", "", "", ErrAdmissionStudentNameRequired
	}
	student, err := s.academicLookup.GetAcademicInfo(ctx, config.SchoolID, studentID)
	if err != nil {
		return "", "", "", fmt.Errorf("resolveAcademicStudentEmail lookup: %w", err)
	}
	if student == nil {
		return "", "", "", ErrAdmissionStudentRecordNotFound
	}
	if config.EmailIdentityPolicy.RequireStudentName {
		recordName := ""
		if student.Name != nil {
			recordName = schoolauth.NormalizeAcademicName(*student.Name)
		}
		if recordName == "" || recordName != studentName {
			return "", "", "", ErrAdmissionStudentNameMismatch
		}
	}
	email := schoolauth.DeriveStudentEmail(studentID, config.EmailIdentityPolicy.StudentIDEmailDomain)
	if email == "" {
		return "", "", "", ErrAdmissionEmailDomainNotAllowed
	}
	if normalizedInputEmail := strings.TrimSpace(input.Email); normalizedInputEmail != "" {
		inputEmail, err := normalizeAdmissionEmail(normalizedInputEmail)
		if err != nil {
			return "", "", "", err
		}
		if inputEmail != email {
			return "", "", "", ErrAdmissionEmailDomainNotAllowed
		}
	}
	return email, studentID, studentName, nil
}

func (s *Service) issueEmailOTP(
	ctx context.Context,
	userID int64,
	schoolID int64,
	email string,
	studentID string,
	studentName string,
) (string, error) {
	if err := s.reserveEmailOTPCooldown(ctx, userID, schoolID); err != nil {
		return "", err
	}
	code, err := s.generateOTP()
	if err != nil {
		return "", err
	}
	record := admissionEmailOTPRecord{
		Email:       email,
		Code:        code,
		StudentID:   studentID,
		StudentName: studentName,
	}
	store := emailOTPStoreInput{UserID: userID, SchoolID: schoolID, Record: record}
	if err := s.storeEmailOTPRecord(ctx, store); err != nil {
		return "", err
	}
	return code, nil
}

func (s *Service) reserveEmailOTPCooldown(ctx context.Context, userID, schoolID int64) error {
	key := admissionEmailOTPKey(userID, schoolID) + admissionEmailOTPCooldownSuffix
	result, err := s.redisClient.SetArgs(ctx, key, "1", redis.SetArgs{
		Mode: "NX",
		TTL:  admissionEmailOTPCooldown,
	}).Result()
	if errors.Is(err, redis.Nil) || result != "OK" {
		return ErrAdmissionOTPCooldown
	}
	if err != nil {
		return fmt.Errorf("reserveEmailOTPCooldown: %w", err)
	}
	return nil
}

func (s *Service) storeEmailOTPRecord(ctx context.Context, input emailOTPStoreInput) error {
	payload, err := json.Marshal(input.Record)
	if err != nil {
		return err
	}
	pipe := s.redisClient.Pipeline()
	pipe.Set(ctx, admissionEmailOTPKey(input.UserID, input.SchoolID), payload, admissionEmailOTPTTL)
	pipe.Del(ctx, admissionEmailOTPAttemptsKey(input.UserID, input.SchoolID))
	_, err = pipe.Exec(ctx)
	return err
}

func (s *Service) loadEmailOTPRecord(ctx context.Context, userID, schoolID int64) (admissionEmailOTPRecord, error) {
	raw, err := s.redisClient.Get(ctx, admissionEmailOTPKey(userID, schoolID)).Bytes()
	if errors.Is(err, redis.Nil) {
		return admissionEmailOTPRecord{}, ErrAdmissionOTPExpired
	}
	if err != nil {
		return admissionEmailOTPRecord{}, fmt.Errorf("loadEmailOTPRecord: %w", err)
	}
	var record admissionEmailOTPRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		return admissionEmailOTPRecord{}, err
	}
	return record, nil
}

func (s *Service) checkEmailOTP(ctx context.Context, input emailOTPCheckInput) error {
	codeMatches := subtle.ConstantTimeCompare([]byte(input.Record.Code), []byte(input.Code)) == 1
	if input.Record.Email != input.Email || !codeMatches {
		return s.recordEmailOTPFailure(ctx, input.UserID, input.SchoolID)
	}
	return nil
}

func (s *Service) recordEmailOTPFailure(ctx context.Context, userID, schoolID int64) error {
	key := admissionEmailOTPAttemptsKey(userID, schoolID)
	attempts, err := s.redisClient.Incr(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("recordEmailOTPFailure: %w", err)
	}
	if attempts == 1 {
		if err := s.redisClient.Expire(ctx, key, admissionEmailOTPTTL).Err(); err != nil {
			return fmt.Errorf("recordEmailOTPFailure expire attempts: %w", err)
		}
	}
	if attempts >= admissionEmailOTPMaxAttempts {
		if err := s.cleanupEmailOTPCode(ctx, userID, schoolID); err != nil {
			return fmt.Errorf("recordEmailOTPFailure cleanup: %w", err)
		}
		return ErrAdmissionOTPMaxAttempts
	}
	return ErrAdmissionOTPInvalid
}

func (s *Service) cleanupEmailOTPCode(ctx context.Context, userID, schoolID int64) error {
	return s.redisClient.Del(
		ctx,
		admissionEmailOTPKey(userID, schoolID),
		admissionEmailOTPAttemptsKey(userID, schoolID),
		admissionEmailOTPKey(userID, schoolID)+admissionEmailOTPCooldownSuffix,
	).Err()
}

func (s *Service) cleanupEmailOTPCodeOnly(ctx context.Context, userID, schoolID int64) error {
	return s.redisClient.Del(
		ctx,
		admissionEmailOTPKey(userID, schoolID),
		admissionEmailOTPAttemptsKey(userID, schoolID),
	).Err()
}

func (s *Service) cleanupEmailOTPCodeOnlyAfterSendFailure(ctx context.Context, userID, schoolID int64) error {
	cleanupCtx, cancel := admissionEmailOTPCleanupContext(ctx)
	defer cancel()
	return s.cleanupEmailOTPCodeOnly(cleanupCtx, userID, schoolID)
}

func admissionEmailOTPCleanupContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return ctxutil.DetachedTimeout(ctx, admissionEmailOTPCleanupTimeout)
}

type emailOTPStoreInput struct {
	UserID   int64
	SchoolID int64
	Record   admissionEmailOTPRecord
}

type emailOTPCheckInput struct {
	UserID   int64
	SchoolID int64
	Email    string
	Code     string
	Record   admissionEmailOTPRecord
}

func admissionEmailOTPKey(userID, schoolID int64) string {
	return admissionEmailOTPKeyPrefix + strconv.FormatInt(userID, 10) + ":" + strconv.FormatInt(schoolID, 10)
}

func admissionEmailOTPAttemptsKey(userID, schoolID int64) string {
	return admissionEmailOTPKey(userID, schoolID) + admissionEmailOTPAttemptsSuffix
}

func normalizeAdmissionEmail(value string) (string, error) {
	email := schoolauth.NormalizeEmailAddress(value)
	if email == "" {
		return "", ErrAdmissionEmailDomainNotAllowed
	}
	return email, nil
}

func normalizeAdmissionOTPCode(value string) (string, error) {
	code := strings.TrimSpace(value)
	length := utf8.RuneCountInString(code)
	if length < admissionEmailOTPMinLength || length > admissionEmailOTPMaxLength {
		return "", ErrAdmissionInvalidInput
	}
	return code, nil
}

func maskAdmissionEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email
	}
	return maskAdmissionEmailLocal(parts[0]) + "@" + parts[1]
}

func maskAdmissionEmailLocal(local string) string {
	runes := []rune(local)
	if len(runes) <= 2 {
		return string(runes[0]) + "*"
	}
	return string(runes[0]) + strings.Repeat("*", len(runes)-2) + string(runes[len(runes)-1])
}

func generateAdmissionOTPCode() (string, error) {
	maxValue := new(big.Int).Exp(big.NewInt(10), big.NewInt(admissionEmailOTPLength), nil)
	n, err := rand.Int(rand.Reader, maxValue)
	if err != nil {
		return "", fmt.Errorf("generate admission otp: %w", err)
	}
	return fmt.Sprintf("%0*d", admissionEmailOTPLength, n), nil
}
