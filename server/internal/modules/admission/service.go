package admission

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/user"
)

const admissionTokenBytes = 32

type QQBindingGateway interface {
	EnsureQQBindingForUserTx(context.Context, pgx.Tx, int64, string, *string) (*user.QQBinding, error)
}

type SchoolEmailSender interface {
	SendAdmissionOTP(ctx context.Context, email string, code string) error
}

type OperatorAccessGateway interface {
	UserHasCapability(ctx context.Context, userID int64, capabilityName string) (bool, error)
}

type FreshmanProjectionGateway interface {
	EnqueueFreshmanProvisionalRoleSyncTx(ctx context.Context, tx pgx.Tx, userID int64, approved bool) error
}

type SchoolSSOExchanger interface {
	ExchangeSchoolSSO(ctx context.Context, input SchoolSSOExchangeInput) (SchoolSSOIdentity, error)
}

type Service struct {
	repo            *Repository
	qqGateway       QQBindingGateway
	hmacKey         []byte
	now             func() time.Time
	generateToken   func() (string, error)
	generateOTP     func() (string, error)
	generateState   func() (string, error)
	authBaseURL     string
	returnURLOrigin string
	materialStore   AdmissionMaterialStore
	redisClient     *redis.Client
	emailSender     SchoolEmailSender
	operatorAccess  OperatorAccessGateway
	projection      FreshmanProjectionGateway
	schoolSSO       SchoolSSOExchanger
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

func WithOperatorAccessGateway(gateway OperatorAccessGateway) ServiceOption {
	return func(s *Service) { s.operatorAccess = gateway }
}

func WithFreshmanProjectionGateway(gateway FreshmanProjectionGateway) ServiceOption {
	return func(s *Service) { s.projection = gateway }
}

func WithSchoolSSOExchanger(exchanger SchoolSSOExchanger) ServiceOption {
	return func(s *Service) { s.schoolSSO = exchanger }
}

func WithAdmissionReturnURLOrigin(origin string) ServiceOption {
	return func(s *Service) { s.returnURLOrigin = origin }
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
		repo:            repo,
		qqGateway:       qqGateway,
		hmacKey:         hmacKey,
		now:             time.Now,
		generateToken:   generateAdmissionToken,
		generateOTP:     generateAdmissionOTPCode,
		generateState:   generateAdmissionState,
		authBaseURL:     defaultAdmissionAuthBaseURL,
		returnURLOrigin: defaultAdmissionReturnURLOrigin,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(svc)
		}
	}
	return svc, nil
}

func generateAdmissionToken() (string, error) {
	buf := make([]byte, admissionTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func generateAdmissionState() (string, error) {
	return generateAdmissionToken()
}
