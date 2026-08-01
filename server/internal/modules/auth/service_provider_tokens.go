package auth

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/StuHelper/StuHelper/server/internal/pkg/oidc"
	"github.com/StuHelper/StuHelper/server/internal/pkg/token"
)

type ServiceOption func(*Service)

type ProviderTokenFamilyRevoker interface {
	RevokeTokenFamilyForApplication(
		ctx context.Context,
		appKey, accessToken, refreshToken string,
	) error
}

type providerTokenCipher interface {
	Encrypt(plaintext string) ([]byte, error)
	Decrypt(ciphertext []byte) (string, error)
}

type providerTokenCoordinator struct {
	revoker ProviderTokenFamilyRevoker
	cipher  providerTokenCipher
}

func WithProviderTokenFamilyRevocation(
	revoker ProviderTokenFamilyRevoker,
	cipher providerTokenCipher,
) ServiceOption {
	if revoker == nil || cipher == nil {
		panic("auth.WithProviderTokenFamilyRevocation: revoker and cipher are required")
	}
	return func(s *Service) {
		s.providerTokens = providerTokenCoordinator{revoker: revoker, cipher: cipher}
	}
}

func (c providerTokenCoordinator) enabled() bool {
	return c.revoker != nil && c.cipher != nil
}

func (s *Service) encryptProviderToken(loginMethod, rawToken string) (string, error) {
	rawToken = normalizeProviderToken(rawToken)
	if !s.providerTokens.enabled() || !providerTokenFamilyTracked(loginMethod) || rawToken == "" {
		return "", nil
	}
	ciphertext, err := s.providerTokens.cipher.Encrypt(rawToken)
	if err != nil {
		return "", fmt.Errorf("encrypt provider token: %w", err)
	}
	return base64.RawStdEncoding.EncodeToString(ciphertext), nil
}

func (s *Service) revokeProviderTokenFamilyFromSession(
	ctx context.Context,
	session *token.SessionData,
	accessTokenFallback string,
) error {
	if !s.providerTokens.enabled() || session == nil || !providerTokenFamilyTracked(session.LoginMethod) {
		return nil
	}

	accessToken, err := s.decryptProviderToken(session.ProviderAccessTokenEnc)
	if err != nil {
		return fmt.Errorf("decrypt provider access token: %w", err)
	}
	if accessToken == "" {
		// Rolling-upgrade compatibility for current-device logout: the request
		// already proved this raw access token matches the tracked session hash.
		// Logout-all has no raw token fallback; the Casdoor adapter can instead
		// rotate the encrypted legacy refresh token and revoke the replacement.
		accessToken = normalizeProviderToken(accessTokenFallback)
	}
	refreshToken, err := s.decryptProviderToken(session.ProviderRefreshTokenEnc)
	if err != nil {
		return fmt.Errorf("decrypt provider refresh token: %w", err)
	}
	return s.revokeRawProviderTokenFamily(ctx, session.ProviderAppKey, accessToken, refreshToken)
}

func (s *Service) loadProviderTokenFamiliesForUser(
	ctx context.Context,
	userID string,
) ([]token.SessionData, error) {
	if !s.providerTokens.enabled() {
		return nil, nil
	}
	sessions, err := s.tokenService.GetSessionStore().ListUserSessions(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list provider sessions: %w", err)
	}
	return sessions, nil
}

func (s *Service) revokeProviderTokenFamilies(
	ctx context.Context,
	sessions []token.SessionData,
) error {
	var revokeErr error
	for _, session := range sessions {
		if err := s.revokeProviderTokenFamilyFromSession(ctx, &session, ""); err != nil {
			revokeErr = errors.Join(revokeErr, fmt.Errorf("session %s: %w", session.SessionID, err))
		}
	}
	if revokeErr != nil {
		return fmt.Errorf("revoke provider token families: %w", revokeErr)
	}
	return nil
}

func (s *Service) decryptProviderToken(encoded string) (string, error) {
	encoded = strings.TrimSpace(encoded)
	if encoded == "" {
		return "", nil
	}
	ciphertext, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("decode provider token: %w", err)
	}
	rawToken, err := s.providerTokens.cipher.Decrypt(ciphertext)
	if err != nil {
		return "", fmt.Errorf("decrypt provider token: %w", err)
	}
	return normalizeProviderToken(rawToken), nil
}

func (s *Service) revokeRawProviderTokenFamily(
	ctx context.Context,
	appKey, accessToken, refreshToken string,
) error {
	accessToken = normalizeProviderToken(accessToken)
	refreshToken = normalizeProviderToken(refreshToken)
	if !s.providerTokens.enabled() {
		return nil
	}
	if accessToken == "" && refreshToken == "" {
		return errors.New("revoke provider token family: provider credentials are required")
	}
	appKey = strings.TrimSpace(appKey)
	if err := s.providerTokens.revoker.RevokeTokenFamilyForApplication(
		ctx,
		appKey,
		accessToken,
		refreshToken,
	); err != nil {
		return fmt.Errorf("revoke provider token family: %w", err)
	}
	return nil
}

func providerTokenFamilyTracked(loginMethod string) bool {
	loginMethod = strings.TrimSpace(loginMethod)
	return loginMethod == "oidc" || loginMethod == "oidc-native"
}

func providerAppKeyForSession(loginMethod, appKey string) string {
	loginMethod = strings.TrimSpace(loginMethod)
	appKey = strings.TrimSpace(appKey)
	if !providerTokenFamilyTracked(loginMethod) {
		return ""
	}
	if appKey != "" {
		return appKey
	}
	if loginMethod == "oidc-native" {
		return oidc.ApplicationUniapp
	}
	return oidc.ApplicationWeb
}

func normalizeProviderToken(rawToken string) string {
	return strings.TrimSpace(rawToken)
}

func normalizeProviderRefreshToken(refreshToken string) string {
	return normalizeProviderToken(refreshToken)
}
