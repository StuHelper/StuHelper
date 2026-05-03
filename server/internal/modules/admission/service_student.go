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

	"github.com/redis/go-redis/v9"
)

const (
	admissionEmailOTPLength          = 6
	admissionEmailOTPTTL             = 5 * time.Minute
	admissionEmailOTPCooldown        = time.Minute
	admissionEmailOTPCooldownSeconds = int(admissionEmailOTPCooldown / time.Second)
	admissionEmailOTPMaxAttempts     = 5
)

const (
	admissionEmailOTPKeyPrefix      = "admission:email_otp:"
	admissionEmailOTPCooldownSuffix = ":cooldown"
	admissionEmailOTPAttemptsSuffix = ":attempts"
)

type admissionEmailOTPRecord struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

func (s *Service) RequestSchoolEmailOTP(
	ctx context.Context,
	input SchoolEmailOTPInput,
) (*SchoolEmailOTPResponse, error) {
	if err := s.requireEmailOTPDependencies(); err != nil {
		return nil, err
	}
	if _, err := s.requireLinkedSession(ctx, input.UserID); err != nil {
		return nil, err
	}
	config, email, err := s.loadEmailOTPConfig(ctx, input)
	if err != nil {
		return nil, err
	}
	if !emailDomainAllowed(email, config.EmailDomains) {
		return nil, ErrAdmissionEmailDomainNotAllowed
	}
	code, err := s.issueEmailOTP(ctx, input.UserID, input.SchoolID, email)
	if err != nil {
		return nil, err
	}
	if err := s.emailSender.SendAdmissionOTP(ctx, email, code); err != nil {
		_ = s.cleanupEmailOTPCode(ctx, input.UserID, input.SchoolID)
		return nil, fmt.Errorf("RequestSchoolEmailOTP send: %w", err)
	}
	return &SchoolEmailOTPResponse{CooldownSeconds: admissionEmailOTPCooldownSeconds}, nil
}

func (s *Service) VerifySchoolEmailOTP(ctx context.Context, input SchoolEmailOTPVerifyInput) (*AdmissionSession, error) {
	if s.redisClient == nil {
		return nil, ErrAdmissionRedisUnavailable
	}
	email, err := normalizeAdmissionEmail(input.Email)
	if err != nil {
		return nil, err
	}
	record, err := s.loadEmailOTPRecord(ctx, input.UserID, input.SchoolID)
	if err != nil {
		return nil, err
	}
	check := emailOTPCheckInput{
		UserID: input.UserID, SchoolID: input.SchoolID, Email: email, Code: input.Code, Record: record,
	}
	if err := s.checkEmailOTP(ctx, check); err != nil {
		return nil, err
	}
	return s.storeStudentCredential(ctx, studentCredentialInput{
		UserID:         input.UserID,
		SchoolID:       input.SchoolID,
		Kind:           CredentialSchoolEmailOTP,
		Subject:        email,
		SubjectDisplay: maskAdmissionEmail(email),
	})
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
) (*AdmissionSchoolConfig, string, error) {
	config, err := s.repo.GetAdmissionSchoolConfig(ctx, input.SchoolID)
	if err != nil {
		return nil, "", err
	}
	if config == nil || !config.Enabled {
		return nil, "", ErrAdmissionEmailDomainNotAllowed
	}
	email, err := normalizeAdmissionEmail(input.Email)
	if err != nil {
		return nil, "", err
	}
	return config, email, nil
}

func (s *Service) issueEmailOTP(ctx context.Context, userID, schoolID int64, email string) (string, error) {
	if err := s.reserveEmailOTPCooldown(ctx, userID, schoolID); err != nil {
		return "", err
	}
	code, err := s.generateOTP()
	if err != nil {
		return "", err
	}
	record := admissionEmailOTPRecord{Email: email, Code: code}
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
	return s.cleanupEmailOTPCode(ctx, input.UserID, input.SchoolID)
}

func (s *Service) recordEmailOTPFailure(ctx context.Context, userID, schoolID int64) error {
	key := admissionEmailOTPAttemptsKey(userID, schoolID)
	attempts, err := s.redisClient.Incr(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("recordEmailOTPFailure: %w", err)
	}
	if attempts == 1 {
		_ = s.redisClient.Expire(ctx, key, admissionEmailOTPTTL).Err()
	}
	if attempts >= admissionEmailOTPMaxAttempts {
		_ = s.cleanupEmailOTPCode(ctx, userID, schoolID)
		return ErrAdmissionOTPMaxAttempts
	}
	return ErrAdmissionOTPInvalid
}

func (s *Service) cleanupEmailOTPCode(ctx context.Context, userID, schoolID int64) error {
	return s.redisClient.Del(
		ctx,
		admissionEmailOTPKey(userID, schoolID),
		admissionEmailOTPAttemptsKey(userID, schoolID),
	).Err()
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
	email := strings.ToLower(strings.TrimSpace(value))
	if email == "" || strings.Count(email, "@") != 1 {
		return "", ErrAdmissionEmailDomainNotAllowed
	}
	return email, nil
}

func emailDomainAllowed(email string, domains []string) bool {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}
	for _, domain := range domains {
		if strings.EqualFold(parts[1], domain) {
			return true
		}
	}
	return false
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
