package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func validConfigForValidation() *Config {
	return &Config{
		App: AppConfig{
			Env:             "development",
			HMACSecret:      "0123456789abcdef0123456789abcdef",
			CORSOrigins:     []string{"http://localhost:3000"},
			MetricsPassword: "metrics-password",
		},
		Security: SecurityConfig{
			DocAESActiveKeyID: 1,
			DocAESKeys: map[uint8][]byte{
				1: make([]byte, 32),
			},
		},
		Database: DatabaseConfig{
			QueryTimeout: 5,
			MaxConns:     20,
			MinConns:     2,
		},
		Token: TokenConfig{
			AccessTokenTTL:  300,
			RefreshTokenTTL: 604800,
			CookieSecure:    true,
		},
		ObjectStorage: ObjectStorageConfig{
			PresignTTL: 600,
		},
		Casdoor: CasdoorConfig{
			Issuer:                    "https://sso.example.com",
			ClientID:                  "client-id",
			ClientSecret:              "client-secret",
			RedirectURI:               "https://api.example.com/api/v1/auth/callback",
			IntrospectionClientID:     "introspection-client",
			IntrospectionClientSecret: "introspection-secret",
			Organization:              "stuhelper",
		},
		OpenFGA: OpenFGAConfig{
			StoreID:              "store-id",
			AuthorizationModelID: "model-id",
			APIUrl:               "http://openfga:8080",
		},
		RateLimit: ReviewRateLimitConfig{
			PostLimit:       5,
			VoteLimit:       30,
			ReportLimit:     10,
			ReplyLimit:      10,
			WriteLimit:      10,
			SearchAnonLimit: 5,
			SearchUserLimit: 60,
			BatchAnonLimit:  5,
			BatchUserLimit:  60,
		},
	}
}

func TestValidate_RejectsParseErrorsInDevelopment(t *testing.T) {
	cfg := validConfigForValidation()

	err := cfg.validate([]string{"DB_MAX_CONNS must be an integer"})

	require.Error(t, err)
	require.Contains(t, err.Error(), "DB_MAX_CONNS must be an integer")
}
