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
		SchoolID: 1,
		Code:     " code-value ",
	})

	require.NoError(t, err)
	assert.Equal(t, "code-value", client.code)
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

type testOIDCSchoolSSOClient struct {
	token      *oauth2.Token
	claims     *oidc.Claims
	err        error
	verifyErr  error
	code       string
	rawIDToken string
}

func (c *testOIDCSchoolSSOClient) ExchangeCodeForApplication(
	_ context.Context,
	_ string,
	code string,
	_ string,
) (*oauth2.Token, error) {
	c.code = code
	if c.err != nil {
		return nil, c.err
	}
	return c.token, nil
}

func (c *testOIDCSchoolSSOClient) VerifyIDToken(_ context.Context, rawIDToken string) (*oidc.Claims, error) {
	c.rawIDToken = rawIDToken
	if c.verifyErr != nil {
		return nil, c.verifyErr
	}
	if c.claims == nil {
		return nil, errors.New("missing claims")
	}
	return c.claims, nil
}
