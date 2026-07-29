package admission

import (
	"context"
	"errors"
	"strings"

	"golang.org/x/oauth2"

	"github.com/StuHelper/StuHelper/server/internal/pkg/oidc"
)

type schoolSSOOIDCClient interface {
	ExchangeCodeForApplication(ctx context.Context, appKey, code, codeVerifier string) (*oauth2.Token, error)
	VerifyIDTokenForApplication(ctx context.Context, appKey, rawIDToken string) (*oidc.Claims, error)
}

type oidcSchoolSSOExchanger struct {
	client schoolSSOOIDCClient
}

// oidcSchoolSSOExchanger expects school SSO to be federated through Casdoor
// and exchanged through the StuHelper web OIDC application.
func NewOIDCSchoolSSOExchanger(client schoolSSOOIDCClient) SchoolSSOExchanger {
	if client == nil {
		return nil
	}
	return &oidcSchoolSSOExchanger{client: client}
}

func (e *oidcSchoolSSOExchanger) ExchangeSchoolSSO(
	ctx context.Context,
	input SchoolSSOExchangeInput,
) (SchoolSSOIdentity, error) {
	token, err := e.client.ExchangeCodeForApplication(
		ctx,
		oidc.ApplicationWeb,
		strings.TrimSpace(input.Code),
		strings.TrimSpace(input.CodeVerifier),
	)
	if err != nil {
		return SchoolSSOIdentity{}, classifySchoolSSOExchangeError(err)
	}
	claims, err := e.verifiedClaims(ctx, token)
	if err != nil {
		return SchoolSSOIdentity{}, err
	}
	return SchoolSSOIdentity{Subject: claims.GetUserID(), SubjectDisplay: schoolSSODisplayName(claims)}, nil
}

func (e *oidcSchoolSSOExchanger) verifiedClaims(ctx context.Context, token *oauth2.Token) (*oidc.Claims, error) {
	rawIDToken := oidc.ExtractIDToken(token)
	if rawIDToken == "" {
		return nil, ErrAdmissionSSOExchangeFailed
	}
	claims, err := e.client.VerifyIDTokenForApplication(ctx, oidc.ApplicationWeb, rawIDToken)
	if err != nil {
		return nil, classifySchoolSSOExchangeError(err)
	}
	return claims, nil
}

func classifySchoolSSOExchangeError(err error) error {
	if errors.Is(err, oidc.ErrProviderUnavailable) {
		return errors.Join(ErrAdmissionSSOProviderUnavailable, err)
	}
	return errors.Join(ErrAdmissionSSOExchangeFailed, err)
}

func schoolSSODisplayName(claims *oidc.Claims) string {
	if claims == nil {
		return ""
	}
	if name := strings.TrimSpace(claims.GetDisplayName()); name != "" {
		return name
	}
	return strings.TrimSpace(claims.GetUsername())
}
