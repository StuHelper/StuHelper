package admission

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"golang.org/x/oauth2"
)

const (
	admissionSSOStateTTL            = 10 * time.Minute
	defaultAdmissionReturnURLOrigin = "https://auth.stuhelper.com"
)

const admissionSchoolSSOStateKeyPrefix = "admission:school_sso_state:"

type admissionSSOStateRecord struct {
	UserID       int64  `json:"userID"`
	SchoolID     int64  `json:"schoolID"`
	ReturnURL    string `json:"returnURL"`
	CodeVerifier string `json:"codeVerifier"`
}

type studentCredentialInput struct {
	UserID         int64
	SchoolID       int64
	Kind           VerificationCredentialKind
	Subject        string
	SubjectDisplay string
}

func (s *Service) StartSchoolSSO(ctx context.Context, input SchoolSSOStartInput) (*SchoolSSOStartResult, error) {
	if s.redisClient == nil {
		return nil, ErrAdmissionRedisUnavailable
	}
	if _, err := s.requireLinkedSession(ctx, input.UserID); err != nil {
		return nil, err
	}
	config, err := s.loadSchoolSSOConfig(ctx, input.SchoolID)
	if err != nil {
		return nil, err
	}
	if !s.admissionReturnURLAllowed(input.ReturnURL) {
		return nil, ErrAdmissionReturnURLNotAllowed
	}
	state, record, err := s.storeSchoolSSOState(ctx, input)
	if err != nil {
		return nil, err
	}
	return &SchoolSSOStartResult{RedirectURL: appendSSOState(config.SSOLoginURL, state, record.CodeVerifier), State: state}, nil
}

func (s *Service) CompleteSchoolSSO(
	ctx context.Context,
	input SchoolSSOCompleteInput,
) (*SchoolSSOCompleteResult, error) {
	if s.redisClient == nil {
		return nil, ErrAdmissionRedisUnavailable
	}
	state, err := s.loadSchoolSSOState(ctx, input)
	if err != nil {
		return nil, err
	}
	input.CodeVerifier = state.CodeVerifier
	identity, err := s.exchangeSchoolSSO(ctx, input)
	if err != nil {
		return nil, err
	}
	_, err = s.storeStudentCredential(ctx, studentCredentialInput{
		UserID:         input.UserID,
		SchoolID:       input.SchoolID,
		Kind:           CredentialSchoolSSO,
		Subject:        identity.Subject,
		SubjectDisplay: identity.SubjectDisplay,
	})
	if err != nil {
		return nil, err
	}
	return &SchoolSSOCompleteResult{ReturnURL: state.ReturnURL}, nil
}

func (s *Service) exchangeSchoolSSO(
	ctx context.Context,
	input SchoolSSOCompleteInput,
) (SchoolSSOIdentity, error) {
	if s.schoolSSO == nil {
		return SchoolSSOIdentity{}, ErrAdmissionSSONotConfigured
	}
	identity, err := s.schoolSSO.ExchangeSchoolSSO(ctx, SchoolSSOExchangeInput{
		SchoolID:     input.SchoolID,
		Code:         strings.TrimSpace(input.Code),
		CodeVerifier: strings.TrimSpace(input.CodeVerifier),
	})
	if err != nil {
		return SchoolSSOIdentity{}, err
	}
	return normalizeSchoolSSOIdentity(identity)
}

func (s *Service) storeStudentCredential(ctx context.Context, input studentCredentialInput) (*AdmissionSession, error) {
	var verified *AdmissionSession
	err := s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		session, err := s.repo.GetLinkedSessionByUserIDTx(ctx, tx, input.UserID)
		if err != nil {
			return err
		}
		if session == nil {
			return ErrAdmissionLinkedSessionRequired
		}
		if err := s.repo.CreateVerificationCredentialTx(ctx, tx, s.newStudentCredential(input)); err != nil {
			return err
		}
		if err := s.repo.MarkUserLinkedSessionsVerifiedTx(ctx, tx, input.UserID, s.now()); err != nil {
			return err
		}
		verified = markSessionVerifiedInMemory(session, s.now())
		return nil
	})
	return verified, err
}

func (s *Service) newStudentCredential(input studentCredentialInput) VerificationCredential {
	return VerificationCredential{
		UserID:         input.UserID,
		SchoolID:       input.SchoolID,
		Kind:           input.Kind,
		SubjectHash:    s.hashCredentialSubject(input.SchoolID, input.Subject),
		SubjectDisplay: input.SubjectDisplay,
		VerifiedAt:     s.now(),
	}
}

func (s *Service) hashCredentialSubject(schoolID int64, subject string) string {
	mac := hmac.New(sha256.New, s.hmacKey)
	mac.Write([]byte(strconv.FormatInt(schoolID, 10) + "|" + strings.TrimSpace(subject)))
	return hex.EncodeToString(mac.Sum(nil))
}

func (s *Service) loadSchoolSSOConfig(ctx context.Context, schoolID int64) (*AdmissionSchoolConfig, error) {
	config, err := s.repo.GetAdmissionSchoolConfig(ctx, schoolID)
	if err != nil {
		return nil, err
	}
	if config == nil || !config.Enabled || strings.TrimSpace(config.SSOLoginURL) == "" {
		return nil, ErrAdmissionSSONotConfigured
	}
	return config, nil
}

func (s *Service) storeSchoolSSOState(ctx context.Context, input SchoolSSOStartInput) (string, admissionSSOStateRecord, error) {
	state, err := s.generateState()
	if err != nil {
		return "", admissionSSOStateRecord{}, err
	}
	record := newSchoolSSOStateRecord(input)
	payload, err := json.Marshal(record)
	if err != nil {
		return "", admissionSSOStateRecord{}, err
	}
	key := admissionSchoolSSOStateKeyPrefix + state
	return state, record, s.redisClient.Set(ctx, key, payload, admissionSSOStateTTL).Err()
}

func (s *Service) loadSchoolSSOState(
	ctx context.Context,
	input SchoolSSOCompleteInput,
) (admissionSSOStateRecord, error) {
	raw, err := s.redisClient.GetDel(ctx, admissionSchoolSSOStateKeyPrefix+input.State).Bytes()
	if errors.Is(err, redis.Nil) {
		return admissionSSOStateRecord{}, ErrAdmissionSSOStateInvalid
	}
	if err != nil {
		return admissionSSOStateRecord{}, err
	}
	return parseSchoolSSOState(raw, input)
}

func newSchoolSSOStateRecord(input SchoolSSOStartInput) admissionSSOStateRecord {
	return admissionSSOStateRecord{
		UserID:       input.UserID,
		SchoolID:     input.SchoolID,
		ReturnURL:    strings.TrimSpace(input.ReturnURL),
		CodeVerifier: oauth2.GenerateVerifier(),
	}
}

func parseSchoolSSOState(raw []byte, input SchoolSSOCompleteInput) (admissionSSOStateRecord, error) {
	var record admissionSSOStateRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		return admissionSSOStateRecord{}, err
	}
	if record.UserID != input.UserID || record.SchoolID != input.SchoolID {
		return admissionSSOStateRecord{}, ErrAdmissionSSOStateInvalid
	}
	if strings.TrimSpace(record.CodeVerifier) == "" {
		return admissionSSOStateRecord{}, ErrAdmissionSSOStateInvalid
	}
	return record, nil
}

func normalizeSchoolSSOIdentity(identity SchoolSSOIdentity) (SchoolSSOIdentity, error) {
	identity.Subject = strings.TrimSpace(identity.Subject)
	identity.SubjectDisplay = strings.TrimSpace(identity.SubjectDisplay)
	if identity.Subject == "" {
		return SchoolSSOIdentity{}, ErrAdmissionSSOIdentityInvalid
	}
	if identity.SubjectDisplay == "" {
		identity.SubjectDisplay = identity.Subject
	}
	return identity, nil
}

func markSessionVerifiedInMemory(session *AdmissionSession, now time.Time) *AdmissionSession {
	verified := *session
	verified.Status = StatusVerified
	verified.VerifiedAt = &now
	return &verified
}

func (s *Service) admissionReturnURLAllowed(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return false
	}
	origin := parsed.Scheme + "://" + parsed.Host
	return origin == s.returnURLOrigin
}

func appendSSOState(loginURL string, state string, codeVerifier string) string {
	parsed, err := url.Parse(loginURL)
	if err != nil {
		return loginURL
	}
	values := parsed.Query()
	values.Set("state", state)
	values.Set("code_challenge", oauth2.S256ChallengeFromVerifier(codeVerifier))
	values.Set("code_challenge_method", "S256")
	parsed.RawQuery = values.Encode()
	return parsed.String()
}
