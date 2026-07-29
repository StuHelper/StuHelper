package oidc

import (
	"context"
	"encoding/json"
	"fmt"

	gooidc "github.com/coreos/go-oidc/v3/oidc"
	"go.uber.org/zap"

	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
)

// VerifyIDToken 验证 ID Token 并返回解析后的 Claims
func (c *Client) VerifyIDToken(ctx context.Context, rawIDToken string) (*Claims, error) {
	return c.verifyIDToken(ctx, "", rawIDToken)
}

func (c *Client) VerifyIDTokenForApplication(ctx context.Context, appKey, rawIDToken string) (*Claims, error) {
	cfg, err := c.oauth2ConfigForApplication(appKey)
	if err != nil {
		return nil, err
	}
	return c.verifyIDToken(ctx, cfg.ClientID, rawIDToken)
}

func (c *Client) verifyIDToken(ctx context.Context, expectedClientID, rawIDToken string) (*Claims, error) {
	rawIDToken, err := normalizeRequiredIDToken(rawIDToken)
	if err != nil {
		return nil, err
	}
	if err := validateJWTSigningAlgorithm(rawIDToken); err != nil {
		return nil, err
	}
	idToken, err := c.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("oidc: id_token verification failed: %w", classifyVerifierError(err))
	}
	if err := c.verifyIDTokenAudience(idToken.Audience, expectedClientID); err != nil {
		return nil, err
	}
	rawJSON, err := marshalIDTokenClaims(idToken)
	if err != nil {
		return nil, fmt.Errorf("oidc: failed to parse raw claims: %w", err)
	}
	if err := c.verifyIDTokenAuthorizedParty(rawJSON, expectedClientID); err != nil {
		return nil, err
	}

	claims := &Claims{}
	if err := idToken.Claims(claims); err != nil {
		return nil, fmt.Errorf("oidc: failed to parse claims: %w", err)
	}
	c.decorateIDTokenClaims(rawJSON, claims)
	return claims, nil
}

func (c *Client) decorateIDTokenClaims(rawJSON []byte, claims *Claims) {
	claims.AppID = appIDFromRawClaims(rawJSON)
	roles, parseErr := ParseProviderRolesFromRaw(rawJSON, c.rolesClaim)
	if parseErr != nil {
		logger.L().Warn("oidc: failed to parse roles from id_token", zap.Error(parseErr))
		return
	}
	claims.Roles = roles
}

func (c *Client) verifyIDTokenAudience(audience []string, expectedClientID string) error {
	for _, aud := range audience {
		if expectedClientID != "" && aud == expectedClientID {
			return nil
		}
		if expectedClientID == "" && c.ApplicationKeyForClientID(aud) != "" {
			return nil
		}
	}
	return fmt.Errorf("%w: %v", ErrInvalidAudience, audience)
}

func (c *Client) verifyIDTokenAuthorizedParty(rawJSON []byte, expectedClientID string) error {
	if expectedClientID == "" {
		return nil
	}
	appID := authorizedPartyFromRawClaims(rawJSON)
	if appID == "" || appID == expectedClientID {
		return nil
	}
	return fmt.Errorf("%w: authorized party %s", ErrInvalidAudience, appID)
}

// marshalIDTokenClaims 将 ID Token 的 claims 序列化为 JSON 以提取动态字段
func marshalIDTokenClaims(idToken *gooidc.IDToken) ([]byte, error) {
	var raw json.RawMessage
	if err := idToken.Claims(&raw); err != nil {
		return nil, err
	}
	return raw, nil
}
