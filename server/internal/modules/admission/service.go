package admission

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"math/big"
	"net/url"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/StuHelper/StuHelper/server/internal/pkg/db"
)

const admissionTokenBytes = 32
const admissionJoinTokenLength = 10
const admissionJoinCharacters = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

type QQBindingGateway interface {
	EnsureQQBindingForUserTx(context.Context, db.Tx, int64, string) error
}

type SchoolEmailSender interface {
	SendAdmissionOTP(ctx context.Context, email string, code string) error
}

type AcademicStudentLookupGateway interface {
	GetAcademicInfo(ctx context.Context, schoolID int64, studentID string) (*AcademicStudent, error)
}

type AcademicStudent struct {
	StudentID string
	Name      *string
}

type OperatorAccessGateway interface {
	UserHasCapabilityInSchool(
		ctx context.Context,
		userID int64,
		capabilityName string,
		schoolID int64,
	) (bool, error)
}

type SchoolSSOExchanger interface {
	ExchangeSchoolSSO(ctx context.Context, input SchoolSSOExchangeInput) (SchoolSSOIdentity, error)
}

// StudentEligibilityGateway is the only target-state boundary through which
// admission may consume student identity. Implementations return a derived
// decision and monotonic revision, never evidence or raw student attributes.
type StudentEligibilityGateway interface {
	EvaluateStudentEligibility(
		ctx context.Context,
		userID int64,
		schoolID int64,
	) (StudentEligibilityDecision, error)
}

type StudentEligibilityDecision struct {
	Eligible        bool
	Revision        int64
	CredentialClass string
}

type Service struct {
	repo               *Repository
	qqGateway          QQBindingGateway
	hmacKey            []byte
	now                func() time.Time
	generateToken      func() (string, error)
	generateJoinToken  func() (string, error)
	generateOTP        func() (string, error)
	generateState      func() (string, error)
	authBaseURL        string
	returnURLOrigin    string
	materialStore      AdmissionMaterialStore
	redisClient        *redis.Client
	emailSender        SchoolEmailSender
	academicLookup     AcademicStudentLookupGateway
	operatorAccess     OperatorAccessGateway
	schoolSSO          SchoolSSOExchanger
	studentEligibility StudentEligibilityGateway

	beforeFreshmanApplicationCreate               func()
	beforeFreshmanCameraHandoffCreate             func()
	beforeFreshmanCameraHandoffMarkUploaded       func()
	beforeFreshmanCameraHandoffContinuationChoose func()
}

type ServiceOption func(*Service)

func WithAdmissionMaterialStore(store AdmissionMaterialStore) ServiceOption {
	return func(s *Service) { s.materialStore = store }
}

func WithAdmissionRedisClient(client *redis.Client) ServiceOption {
	return func(s *Service) { s.redisClient = client }
}

func WithSchoolEmailSender(sender SchoolEmailSender) ServiceOption {
	return func(s *Service) { s.emailSender = sender }
}

func WithAcademicStudentLookupGateway(gateway AcademicStudentLookupGateway) ServiceOption {
	return func(s *Service) { s.academicLookup = gateway }
}

func WithOperatorAccessGateway(gateway OperatorAccessGateway) ServiceOption {
	return func(s *Service) { s.operatorAccess = gateway }
}

func WithSchoolSSOExchanger(exchanger SchoolSSOExchanger) ServiceOption {
	return func(s *Service) { s.schoolSSO = exchanger }
}

func WithAdmissionPublicBaseURL(baseURL string) ServiceOption {
	return func(s *Service) {
		normalized := normalizeAdmissionPublicBaseURL(baseURL)
		if normalized == "" {
			return
		}
		s.authBaseURL = normalized + "/verify/"
		s.returnURLOrigin = normalized
	}
}

func WithStudentEligibilityGateway(gateway StudentEligibilityGateway) ServiceOption {
	return func(s *Service) { s.studentEligibility = gateway }
}

func NewService(repo *Repository, qqGateway QQBindingGateway, hmacKey []byte, opts ...ServiceOption) (*Service, error) {
	if repo == nil {
		return nil, errors.New("admission.NewService: repo must not be nil")
	}
	if qqGateway == nil {
		return nil, errors.New("admission.NewService: qqGateway must not be nil")
	}
	if len(hmacKey) == 0 {
		return nil, errors.New("admission.NewService: hmacKey must not be empty")
	}
	svc := &Service{
		repo:              repo,
		qqGateway:         qqGateway,
		hmacKey:           hmacKey,
		now:               time.Now,
		generateToken:     generateAdmissionToken,
		generateJoinToken: generateAdmissionJoinToken,
		generateOTP:       generateAdmissionOTPCode,
		generateState:     generateAdmissionState,
		authBaseURL:       defaultAdmissionPublicBaseURL + "/verify/",
		returnURLOrigin:   defaultAdmissionPublicBaseURL,
	}
	if gateway, ok := qqGateway.(AcademicStudentLookupGateway); ok {
		svc.academicLookup = gateway
	}
	for _, opt := range opts {
		if opt != nil {
			opt(svc)
		}
	}
	return svc, nil
}

func (s *Service) ResolveSchoolIDByCode(ctx context.Context, schoolCode string) (int64, error) {
	code := strings.TrimSpace(schoolCode)
	if code == "" {
		return 0, ErrAdmissionSchoolNotFound
	}
	config, err := s.repo.GetAdmissionSchoolConfigByCode(ctx, code)
	if err != nil {
		return 0, err
	}
	if config == nil {
		return 0, ErrAdmissionSchoolNotFound
	}
	if !config.Enabled {
		return 0, ErrAdmissionSchoolDisabled
	}
	return config.SchoolID, nil
}

func validateAdmissionUserID(userID int64) error {
	if userID <= 0 {
		return ErrAdmissionInvalidInput
	}
	return nil
}

func validateAdmissionSchoolID(schoolID int64) error {
	if schoolID <= 0 {
		return ErrAdmissionInvalidInput
	}
	return nil
}

func validateAdmissionUserAndSchool(userID int64, schoolID int64) error {
	if err := validateAdmissionUserID(userID); err != nil {
		return err
	}
	return validateAdmissionSchoolID(schoolID)
}

func generateAdmissionToken() (string, error) {
	buf := make([]byte, admissionTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func generateAdmissionJoinToken() (string, error) {
	var builder strings.Builder
	builder.Grow(admissionJoinTokenLength)
	alphabetSize := big.NewInt(int64(len(admissionJoinCharacters)))
	for i := 0; i < admissionJoinTokenLength; i++ {
		index, err := rand.Int(rand.Reader, alphabetSize)
		if err != nil {
			return "", err
		}
		builder.WriteByte(admissionJoinCharacters[index.Int64()])
	}
	return builder.String(), nil
}

func generateAdmissionState() (string, error) {
	return generateAdmissionToken()
}

func normalizeAdmissionPublicBaseURL(raw string) string {
	raw = strings.TrimRight(strings.TrimSpace(raw), "/")
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ""
	}
	parsed.Path = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return strings.TrimRight(parsed.String(), "/")
}
