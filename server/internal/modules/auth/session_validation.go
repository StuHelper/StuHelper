package auth

import (
	"context"
	"errors"
	"fmt"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/token"
)

var (
	errSessionUserMismatch         = errors.New("session user mismatch")
	errSessionAccessTokenMismatch  = errors.New("session access token mismatch")
	errSessionRefreshTokenMismatch = errors.New("session refresh token mismatch")
)

type trackedSessionExpectation struct {
	userID       string
	accessToken  string
	refreshToken string
}

func loadTrackedSession(ctx context.Context, store *token.SessionStore, sessionID string) (*token.SessionData, error) {
	if store == nil {
		return nil, errors.New("session store is required")
	}

	session, err := store.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, token.ErrSessionNotFound
	}
	return session, nil
}

func validateTrackedSession(session *token.SessionData, expectation trackedSessionExpectation) error {
	if session == nil {
		return token.ErrSessionNotFound
	}
	if expectation.userID != "" && session.UserID != expectation.userID {
		return errSessionUserMismatch
	}
	if err := validateTrackedTokenHash(expectation.accessToken, session.AccessTokenHash, errSessionAccessTokenMismatch); err != nil {
		return err
	}
	if err := validateTrackedTokenHash(expectation.refreshToken, session.RefreshTokenHash, errSessionRefreshTokenMismatch); err != nil {
		return err
	}
	return nil
}

func validateTrackedTokenHash(rawToken, storedHash string, mismatchErr error) error {
	if rawToken == "" {
		return nil
	}
	if storedHash == "" {
		return mismatchErr
	}

	hash, err := hashTokenForSession(rawToken)
	if err != nil {
		return fmt.Errorf("hash tracked token: %w", err)
	}
	if hash != storedHash {
		return mismatchErr
	}
	return nil
}
