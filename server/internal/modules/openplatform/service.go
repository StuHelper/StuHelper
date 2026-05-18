package openplatform

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	platformcasdoor "git.stuhelper.com/StuHelper/StuHelper/internal/platform/casdoor"
)

const (
	clientSecretBytes  = 32
	clientIDBytes      = 16
	consentTokenBytes  = 24
	consentTokenTTL    = 5 * time.Minute
	consentRedisPrefix = "open_platform:consent:"
)

type appProvisioner interface {
	CreateApplication(ctx context.Context, spec platformcasdoor.ApplicationSpec) error
	UpdateApplication(ctx context.Context, spec platformcasdoor.ApplicationSpec) error
}

type oidcAuthURLBuilder interface {
	GetAuthURL(clientID string, redirectURI string, scopes []string, state string) string
}

type phoneDecryptor interface {
	Decrypt(ciphertext []byte) (string, error)
}

type casdoorPhoneReader interface {
	GetPhone(ctx context.Context, subject string) (string, error)
}

type Service struct {
	repo           *Repository
	rdb            *redis.Client
	provisioner    appProvisioner
	oidc           oidcAuthURLBuilder
	phoneCipher    phoneDecryptor
	phoneReader    casdoorPhoneReader
	consentBaseURL string
}

type ServiceOption func(*Service)

func WithAppProvisioner(provisioner appProvisioner) ServiceOption {
	return func(s *Service) {
		s.provisioner = provisioner
	}
}

func WithPhoneDecryptor(cipher phoneDecryptor) ServiceOption {
	return func(s *Service) {
		s.phoneCipher = cipher
	}
}

func WithCasdoorPhoneReader(reader casdoorPhoneReader) ServiceOption {
	return func(s *Service) {
		s.phoneReader = reader
	}
}

func WithOIDCAuthURLBuilder(builder oidcAuthURLBuilder) ServiceOption {
	return func(s *Service) {
		s.oidc = builder
	}
}

func WithConsentBaseURL(baseURL string) ServiceOption {
	return func(s *Service) {
		s.consentBaseURL = strings.TrimSpace(baseURL)
	}
}

func NewService(repo *Repository, rdb *redis.Client, opts ...ServiceOption) (*Service, error) {
	if repo == nil {
		return nil, fmt.Errorf("openplatform.NewService: repo is required")
	}
	if rdb == nil {
		return nil, fmt.Errorf("openplatform.NewService: redis client is required")
	}
	svc := &Service{repo: repo, rdb: rdb}
	for _, opt := range opts {
		if opt != nil {
			opt(svc)
		}
	}
	return svc, nil
}

type RegisterAppInput struct {
	OwnerUserID      int64
	DisplayName      string
	Description      string
	HomepageURL      string
	PrivacyPolicyURL string
	RedirectURIs     []string
	Scopes           []ScopeRequestInput
}

type ScopeRequestInput struct {
	Scope  string
	Reason string
}

type RegisteredApp struct {
	App          *App
	ClientSecret string
}

func (s *Service) RegisterApp(ctx context.Context, input RegisterAppInput) (*RegisteredApp, error) {
	scopes, err := normalizeScopeRequests(input.Scopes)
	if err != nil {
		return nil, err
	}
	app, secret, err := buildNewApp(input)
	if err != nil {
		return nil, err
	}
	requests := make([]ScopeRequest, 0, len(scopes))
	for _, scope := range scopes {
		requests = append(requests, ScopeRequest{
			Scope:  scope.Scope,
			Reason: scope.Reason,
			Status: ScopeStatusPending,
		})
	}
	if err := s.repo.CreateApp(ctx, app, requests); err != nil {
		return nil, fmt.Errorf("RegisterApp create app: %w", err)
	}
	return &RegisteredApp{App: app, ClientSecret: secret}, nil
}

func buildNewApp(input RegisterAppInput) (*App, string, error) {
	name, err := randomHex("third-party-", clientIDBytes)
	if err != nil {
		return nil, "", err
	}
	clientID, err := randomHex("op_", clientIDBytes)
	if err != nil {
		return nil, "", err
	}
	app := &App{
		CasdoorApplicationName: name,
		OwnerUserID:            input.OwnerUserID,
		ClientID:               clientID,
		ClientSecretHash:       "",
		DisplayName:            strings.TrimSpace(input.DisplayName),
		Description:            strings.TrimSpace(input.Description),
		HomepageURL:            strings.TrimSpace(input.HomepageURL),
		PrivacyPolicyURL:       strings.TrimSpace(input.PrivacyPolicyURL),
		RedirectURIs:           normalizeRedirects(input.RedirectURIs),
		Status:                 AppStatusPending,
	}
	if err := validateAppInput(app); err != nil {
		return nil, "", err
	}
	return app, "", nil
}

func hashClientSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

func normalizeScopeRequests(inputs []ScopeRequestInput) ([]ScopeRequestInput, error) {
	raw := make([]string, 0, len(inputs))
	reasons := make(map[string]string, len(inputs))
	for _, input := range inputs {
		scope := strings.TrimSpace(input.Scope)
		raw = append(raw, scope)
		reasons[scope] = strings.TrimSpace(input.Reason)
	}
	scopes, err := NormalizeScopes(raw)
	if err != nil {
		return nil, err
	}
	result := make([]ScopeRequestInput, 0, len(scopes))
	for _, scope := range scopes {
		result = append(result, ScopeRequestInput{Scope: scope, Reason: reasons[scope]})
	}
	return result, nil
}

func validateAppInput(app *App) error {
	if app.OwnerUserID <= 0 || app.DisplayName == "" {
		return fmt.Errorf("open platform app owner and display name are required")
	}
	if err := validateHTTPSURL("homepage URL", app.HomepageURL); err != nil {
		return err
	}
	if err := validateHTTPSURL("privacy policy URL", app.PrivacyPolicyURL); err != nil {
		return err
	}
	if len(app.RedirectURIs) == 0 {
		return ErrRedirectURINotAllowed
	}
	for _, redirect := range app.RedirectURIs {
		if err := validateRedirectURI(redirect); err != nil {
			return err
		}
	}
	return nil
}

func validateHTTPSURL(label, value string) error {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.Fragment != "" {
		return fmt.Errorf("open platform %s must be an https URL without fragment", label)
	}
	return nil
}

func validateRedirectURI(value string) error {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.Fragment != "" {
		return ErrRedirectURINotAllowed
	}
	if strings.Contains(value, "*") {
		return ErrRedirectURINotAllowed
	}
	return nil
}

func normalizeRedirects(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func randomHex(prefix string, size int) (string, error) {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("open platform random: %w", err)
	}
	return prefix + hex.EncodeToString(bytes), nil
}
