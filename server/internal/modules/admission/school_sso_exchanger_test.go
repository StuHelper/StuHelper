package admission

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/oidc"
)

func TestOIDCSchoolSSOExchangerUsesVerifiedIDTokenClaims(t *testing.T) {
	client := &testOIDCSchoolSSOClient{
		token: (&oauth2.Token{}).WithExtra(map[string]any{"id_token": "raw-id-token"}),
		claims: &oidc.Claims{
			Sub:               "official-subject",
			Name:              "Official Student",
			PreferredUsername: "student-id",
		},
	}
	exchanger := NewOIDCSchoolSSOExchanger(client)

	identity, err := exchanger.ExchangeSchoolSSO(context.Background(), SchoolSSOExchangeInput{
		SchoolID:     1,
		Code:         " code-value ",
		CodeVerifier: " verifier-value ",
	})

	require.NoError(t, err)
	assert.Equal(t, "code-value", client.code)
	assert.Equal(t, "verifier-value", client.codeVerifier)
	assert.Equal(t, oidc.ApplicationWeb, client.appKey)
	assert.Equal(t, "official-subject", identity.Subject)
	assert.Equal(t, "Official Student", identity.SubjectDisplay)
	assert.Equal(t, "raw-id-token", client.rawIDToken)
}

func TestOIDCSchoolSSOExchangerRejectsMissingIDToken(t *testing.T) {
	client := &testOIDCSchoolSSOClient{token: &oauth2.Token{}}
	exchanger := NewOIDCSchoolSSOExchanger(client)

	_, err := exchanger.ExchangeSchoolSSO(context.Background(), SchoolSSOExchangeInput{Code: "code"})

	require.ErrorIs(t, err, ErrAdmissionSSOExchangeFailed)
}

func TestOIDCSchoolSSOExchangerClassifiesProviderUnavailable(t *testing.T) {
	client := &testOIDCSchoolSSOClient{err: oidc.ErrProviderUnavailable}
	exchanger := NewOIDCSchoolSSOExchanger(client)

	_, err := exchanger.ExchangeSchoolSSO(context.Background(), SchoolSSOExchangeInput{Code: "code"})

	require.ErrorIs(t, err, ErrAdmissionSSOProviderUnavailable)
}

func TestOIDCSchoolSSOExchangerClassifiesVerifyError(t *testing.T) {
	client := &testOIDCSchoolSSOClient{
		token:     (&oauth2.Token{}).WithExtra(map[string]any{"id_token": "raw-id-token"}),
		verifyErr: errors.New("signature mismatch"),
	}
	exchanger := NewOIDCSchoolSSOExchanger(client)

	_, err := exchanger.ExchangeSchoolSSO(context.Background(), SchoolSSOExchangeInput{Code: "code"})

	require.ErrorIs(t, err, ErrAdmissionSSOExchangeFailed)
}

type testOIDCSchoolSSOClient struct {
	token        *oauth2.Token
	claims       *oidc.Claims
	err          error
	verifyErr    error
	appKey       string
	code         string
	codeVerifier string
	rawIDToken   string
}

func (c *testOIDCSchoolSSOClient) ExchangeCodeForApplication(
	_ context.Context,
	appKey string,
	code string,
	codeVerifier string,
) (*oauth2.Token, error) {
	c.appKey = appKey
	c.code = code
	c.codeVerifier = codeVerifier
	if c.err != nil {
		return nil, c.err
	}
	return c.token, nil
}

func (c *testOIDCSchoolSSOClient) VerifyIDTokenForApplication(_ context.Context, appKey, rawIDToken string) (*oidc.Claims, error) {
	c.appKey = appKey
	c.rawIDToken = rawIDToken
	if c.verifyErr != nil {
		return nil, c.verifyErr
	}
	if c.claims == nil {
		return nil, errors.New("missing claims")
	}
	return c.claims, nil
}
