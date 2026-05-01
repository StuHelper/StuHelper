package auth

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/token"
)

type ServiceOption func(*Service)

type ProviderRefreshTokenRevoker interface {
	RevokeRefreshToken(ctx context.Context, refreshToken string) error
}

type providerRefreshTokenCipher interface {
	Encrypt(plaintext string) ([]byte, error)
	Decrypt(ciphertext []byte) (string, error)
}

type providerTokenCoordinator struct {
	revoker ProviderRefreshTokenRevoker
	cipher  providerRefreshTokenCipher
}

func WithProviderRefreshTokenRevocation(
	revoker ProviderRefreshTokenRevoker,
	cipher providerRefreshTokenCipher,
) ServiceOption {
	if revoker == nil || cipher == nil {
		panic("auth.WithProviderRefreshTokenRevocation: revoker and cipher are required")
	}
	return func(s *Service) {
		s.providerTokens = providerTokenCoordinator{revoker: revoker, cipher: cipher}
	}
}

func (c providerTokenCoordinator) enabled() bool {
	return c.revoker != nil && c.cipher != nil
}

func (s *Service) encryptProviderRefreshToken(loginMethod, refreshToken string) (string, error) {
	if !s.providerTokens.enabled() || !providerRefreshTokenTracked(loginMethod) || refreshToken == "" {
		return "", nil
	}
	ciphertext, err := s.providerTokens.cipher.Encrypt(refreshToken)
	if err != nil {
		return "", fmt.Errorf("encrypt provider refresh token: %w", err)
	}
	return base64.RawStdEncoding.EncodeToString(ciphertext), nil
}

func (s *Service) revokeProviderRefreshTokenFromSession(ctx context.Context, session *token.SessionData) error {
	if !s.providerTokens.enabled() || session == nil || session.ProviderRefreshTokenEnc == "" {
		return nil
	}
	rawToken, err := s.decryptProviderRefreshToken(session.ProviderRefreshTokenEnc)
	if err != nil {
		return err
	}
	return s.revokeRawProviderRefreshToken(ctx, rawToken)
}

func (s *Service) revokeProviderRefreshTokensForUser(ctx context.Context, userID string) error {
	if !s.providerTokens.enabled() {
		return nil
	}
	sessions, err := s.tokenService.GetSessionStore().ListUserSessions(ctx, userID)
	if err != nil {
		return fmt.Errorf("list provider sessions: %w", err)
	}

	var revokeErr error
	for _, session := range sessions {
		if err := s.revokeProviderRefreshTokenFromSession(ctx, &session); err != nil {
			revokeErr = errors.Join(revokeErr, fmt.Errorf("session %s: %w", session.SessionID, err))
		}
	}
	if revokeErr != nil {
		return fmt.Errorf("revoke provider refresh tokens: %w", revokeErr)
	}
	return nil
}

func (s *Service) decryptProviderRefreshToken(encoded string) (string, error) {
	ciphertext, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("decode provider refresh token: %w", err)
	}
	rawToken, err := s.providerTokens.cipher.Decrypt(ciphertext)
	if err != nil {
		return "", fmt.Errorf("decrypt provider refresh token: %w", err)
	}
	if rawToken == "" {
		return "", errors.New("provider refresh token decrypted to empty value")
	}
	return rawToken, nil
}

func (s *Service) revokeRawProviderRefreshToken(ctx context.Context, refreshToken string) error {
	if !s.providerTokens.enabled() || refreshToken == "" {
		return nil
	}
	if err := s.providerTokens.revoker.RevokeRefreshToken(ctx, refreshToken); err != nil {
		return fmt.Errorf("revoke provider refresh token: %w", err)
	}
	return nil
}

func providerRefreshTokenTracked(loginMethod string) bool {
	return loginMethod == "oidc" || loginMethod == "oidc-native"
}
