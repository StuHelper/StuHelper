package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/pkg/config"
	"github.com/StuHelper/StuHelper/server/internal/pkg/crypto/pii"
	"github.com/StuHelper/StuHelper/server/internal/pkg/oidc"
	"github.com/StuHelper/StuHelper/server/internal/pkg/token"
	"github.com/StuHelper/StuHelper/server/internal/testutil/redisfixture"
)

type providerTokenFamilyRevokeCall struct {
	appKey       string
	accessToken  string
	refreshToken string
}

type fakeProviderTokenFamilyRevoker struct {
	calls         []providerTokenFamilyRevokeCall
	err           error
	requireAccess bool
	ctxErr        error
	onRevoke      func(ctx context.Context, appKey, accessToken, refreshToken string)
}

func (f *fakeProviderTokenFamilyRevoker) RevokeTokenFamilyForApplication(
	ctx context.Context,
	appKey, accessToken, refreshToken string,
) error {
	f.ctxErr = ctx.Err()
	if f.onRevoke != nil {
		f.onRevoke(ctx, appKey, accessToken, refreshToken)
	}
	f.calls = append(f.calls, providerTokenFamilyRevokeCall{
		appKey:       appKey,
		accessToken:  accessToken,
		refreshToken: refreshToken,
	})
	if f.requireAccess && accessToken == "" {
		return errors.New("provider access token is required")
	}
	return f.err
}

func newAuthServiceWithProviderRevoker(
	t *testing.T,
	revoker ProviderTokenFamilyRevoker,
) (*Service, *token.Service) {
	t.Helper()
	fixture := redisfixture.Start(t)
	tokenSvc, err := token.NewService(token.ServiceConfig{RedisClient: fixture.Client, AccessTTL: 300, RefreshTTL: 600})
	require.NoError(t, err)
	t.Cleanup(tokenSvc.Close)

	cipher, err := pii.NewCipher(1, map[uint8][]byte{1: []byte("0123456789abcdef0123456789abcdef")})
	require.NoError(t, err)
	tokenCfg := config.TokenConfig{AccessTokenTTL: 300, RefreshTokenTTL: 600}
	svc := NewService(tokenCfg, tokenSvc, &fakeUserSyncRepo{}, WithProviderTokenFamilyRevocation(revoker, cipher))
	return svc, tokenSvc
}

func TestProviderTokenFamilyRevokedOnSessionLifecycle(t *testing.T) {
	revoker := &fakeProviderTokenFamilyRevoker{}
	svc, tokenSvc := newAuthServiceWithProviderRevoker(t, revoker)

	_, err := svc.CreateSession(t.Context(), "sid-oidc", "user-1", "old-access", "old-refresh", "oidc", "browser")
	require.NoError(t, err)
	session := requireSession(t, tokenSvc, "sid-oidc")
	assert.NotEmpty(t, session.ProviderAccessTokenEnc)
	assert.NotEmpty(t, session.ProviderRefreshTokenEnc)
	assert.NotContains(t, session.ProviderAccessTokenEnc, "old-access")
	assert.NotContains(t, session.ProviderRefreshTokenEnc, "old-refresh")

	err = svc.RotateSession(
		t.Context(),
		"sid-oidc",
		"user-1",
		"old-refresh",
		"new-access",
		futureAccessTokenExpiryUnix(),
		"new-provider-access",
		"new-refresh",
	)
	require.NoError(t, err)
	assert.Empty(t, revoker.calls)

	session = requireSession(t, tokenSvc, "sid-oidc")
	rawProviderAccess, err := svc.decryptProviderToken(session.ProviderAccessTokenEnc)
	require.NoError(t, err)
	assert.Equal(t, "new-provider-access", rawProviderAccess)
	rawProviderRefresh, err := svc.decryptProviderToken(session.ProviderRefreshTokenEnc)
	require.NoError(t, err)
	assert.Equal(t, "new-refresh", rawProviderRefresh)

	err = svc.RevokeSession(t.Context(), "sid-oidc", "user-1", "new-access", "new-refresh", futureAccessTokenExpiry())
	require.NoError(t, err)
	assert.Equal(t, []providerTokenFamilyRevokeCall{{
		appKey:       oidc.ApplicationWeb,
		accessToken:  "new-provider-access",
		refreshToken: "new-refresh",
	}}, revoker.calls)
	assert.Nil(t, requireSession(t, tokenSvc, "sid-oidc"))
}

func TestProviderTokenFamilyNotRevokedWhenRotationValidationFails(t *testing.T) {
	revoker := &fakeProviderTokenFamilyRevoker{}
	svc, _ := newAuthServiceWithProviderRevoker(t, revoker)

	_, err := svc.CreateSession(t.Context(), "sid-oidc-mismatch", "user-1", "old-access", "old-refresh", "oidc", "browser")
	require.NoError(t, err)

	err = svc.RotateSession(
		t.Context(),
		"sid-oidc-mismatch",
		"user-1",
		"other-refresh",
		"new-access",
		futureAccessTokenExpiryUnix(),
		"new-provider-access",
		"new-refresh",
	)
	require.ErrorIs(t, err, errSessionRefreshTokenMismatch)
	assert.Empty(t, revoker.calls)
}

func TestProviderTokenFamilyInputsAreNormalized(t *testing.T) {
	revoker := &fakeProviderTokenFamilyRevoker{}
	svc, tokenSvc := newAuthServiceWithProviderRevoker(t, revoker)

	_, err := svc.CreateSessionForApplication(
		t.Context(),
		"sid-provider-trim",
		"user-1",
		"access-token",
		futureAccessTokenExpiryUnix(),
		" \tprovider-access\n ",
		" \tprovider-refresh\n ",
		"oidc-native",
		" \t"+oidc.ApplicationUniapp+"\n ",
		"ios",
	)
	require.NoError(t, err)

	session := requireSession(t, tokenSvc, "sid-provider-trim")
	require.NotEmpty(t, session.ProviderAccessTokenEnc)
	require.NotEmpty(t, session.ProviderRefreshTokenEnc)
	assert.Equal(t, oidc.ApplicationUniapp, session.ProviderAppKey)

	rawProviderAccess, err := svc.decryptProviderToken(session.ProviderAccessTokenEnc)
	require.NoError(t, err)
	assert.Equal(t, "provider-access", rawProviderAccess)
	rawProviderRefresh, err := svc.decryptProviderToken(session.ProviderRefreshTokenEnc)
	require.NoError(t, err)
	assert.Equal(t, "provider-refresh", rawProviderRefresh)

	refreshSession, err := svc.OIDCSessionForRefresh(t.Context(), "sid-provider-trim", " \tprovider-refresh\n ")
	require.NoError(t, err)
	assert.Equal(t, oidc.ApplicationUniapp, refreshSession.applicationKey)
	assert.Equal(t, "user-1", refreshSession.subject)

	err = svc.RevokeSession(t.Context(), "sid-provider-trim", "user-1", "access-token", " \tprovider-refresh\n ", futureAccessTokenExpiry())
	require.NoError(t, err)
	assert.Equal(t, []providerTokenFamilyRevokeCall{
		{appKey: oidc.ApplicationUniapp, accessToken: "provider-access", refreshToken: "provider-refresh"},
	}, revoker.calls)
}

func TestProviderSessionRequiresProviderAccessToken(t *testing.T) {
	revoker := &fakeProviderTokenFamilyRevoker{}
	svc, tokenSvc := newAuthServiceWithProviderRevoker(t, revoker)

	_, err := svc.CreateSessionForApplication(
		t.Context(),
		"sid-provider-access-required",
		"user-1",
		"client-access-token",
		futureAccessTokenExpiryUnix(),
		" \t\n ",
		"provider-refresh-token",
		"oidc",
		oidc.ApplicationWeb,
		"browser",
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider access token is required")
	assert.Nil(t, requireSession(t, tokenSvc, "sid-provider-access-required"))
}

func TestProviderSessionRotationRequiresProviderAccessTokenBeforeTouch(t *testing.T) {
	revoker := &fakeProviderTokenFamilyRevoker{}
	svc, tokenSvc := newAuthServiceWithProviderRevoker(t, revoker)
	_, err := svc.CreateSession(
		t.Context(),
		"sid-provider-rotation-access-required",
		"user-1",
		"old-access",
		"old-refresh",
		"oidc",
		"browser",
	)
	require.NoError(t, err)

	err = svc.RotateSession(
		t.Context(),
		"sid-provider-rotation-access-required",
		"user-1",
		"old-refresh",
		"new-client-access",
		futureAccessTokenExpiryUnix(),
		" \t\n ",
		"new-refresh",
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider access token is required")
	session := requireSession(t, tokenSvc, "sid-provider-rotation-access-required")
	oldAccessHash, hashErr := hashTokenForSession("old-access")
	require.NoError(t, hashErr)
	oldRefreshHash, hashErr := hashTokenForSession("old-refresh")
	require.NoError(t, hashErr)
	assert.Equal(t, oldAccessHash, session.AccessTokenHash)
	assert.Equal(t, oldRefreshHash, session.RefreshTokenHash)
	assert.Empty(t, revoker.calls)
}

func TestProviderTokenFamilyRevokesAccessWhenRefreshTokenIsBlank(t *testing.T) {
	revoker := &fakeProviderTokenFamilyRevoker{}
	svc, tokenSvc := newAuthServiceWithProviderRevoker(t, revoker)

	_, err := svc.CreateSessionForApplication(
		t.Context(),
		"sid-provider-blank",
		"user-1",
		"access-token",
		futureAccessTokenExpiryUnix(),
		"access-token",
		" \t\n ",
		"oidc-native",
		" \t\n ",
		"ios",
	)
	require.NoError(t, err)

	session := requireSession(t, tokenSvc, "sid-provider-blank")
	assert.NotEmpty(t, session.ProviderAccessTokenEnc)
	assert.Empty(t, session.ProviderRefreshTokenEnc)
	assert.Equal(t, oidc.ApplicationUniapp, session.ProviderAppKey)

	err = svc.RevokeSession(t.Context(), "sid-provider-blank", "user-1", "access-token", " \t\n ", futureAccessTokenExpiry())
	require.NoError(t, err)
	assert.Equal(t, []providerTokenFamilyRevokeCall{{
		appKey:      oidc.ApplicationUniapp,
		accessToken: "access-token",
	}}, revoker.calls)
}

func TestCurrentDeviceLogoutUsesVerifiedAccessTokenForLegacyProviderSession(t *testing.T) {
	revoker := &fakeProviderTokenFamilyRevoker{requireAccess: true}
	svc, tokenSvc := newAuthServiceWithProviderRevoker(t, revoker)

	accessHash, err := hashTokenForSession("legacy-access")
	require.NoError(t, err)
	refreshHash, err := hashTokenForSession("legacy-refresh")
	require.NoError(t, err)
	providerRefreshEnc, err := svc.encryptProviderToken("oidc", "legacy-refresh")
	require.NoError(t, err)
	require.NoError(t, tokenSvc.GetSessionStore().Create(t.Context(), token.SessionData{
		SessionID:               "sid-provider-legacy",
		UserID:                  "user-provider-legacy",
		AccessTokenHash:         accessHash,
		RefreshTokenHash:        refreshHash,
		ProviderRefreshTokenEnc: providerRefreshEnc,
		ProviderAppKey:          oidc.ApplicationWeb,
		LoginMethod:             "oidc",
	}))

	err = svc.RevokeSession(
		t.Context(),
		"sid-provider-legacy",
		"user-provider-legacy",
		"legacy-access",
		"legacy-refresh",
		futureAccessTokenExpiry(),
	)

	require.NoError(t, err)
	assert.Equal(t, []providerTokenFamilyRevokeCall{{
		appKey:       oidc.ApplicationWeb,
		accessToken:  "legacy-access",
		refreshToken: "legacy-refresh",
	}}, revoker.calls)
}

func TestRevokeRawProviderTokenFamilyNormalizesInputs(t *testing.T) {
	revoker := &fakeProviderTokenFamilyRevoker{}
	svc, _ := newAuthServiceWithProviderRevoker(t, revoker)

	err := svc.revokeRawProviderTokenFamily(
		t.Context(),
		" \t"+oidc.ApplicationUniapp+"\n ",
		" \tprovider-access\n ",
		" \tprovider-refresh\n ",
	)
	require.NoError(t, err)
	assert.Equal(t, []providerTokenFamilyRevokeCall{
		{appKey: oidc.ApplicationUniapp, accessToken: "provider-access", refreshToken: "provider-refresh"},
	}, revoker.calls)

	err = svc.revokeRawProviderTokenFamily(t.Context(), oidc.ApplicationWeb, " \t\n ", " \t\n ")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider credentials are required")
	assert.Len(t, revoker.calls, 1)
}

func TestRevokeAllSessionsRevokesProviderTokenFamilies(t *testing.T) {
	revoker := &fakeProviderTokenFamilyRevoker{}
	svc, tokenSvc := newAuthServiceWithProviderRevoker(t, revoker)

	createTrackedSession(t, svc, trackedSessionSeed{
		SessionID: "sid-a", UserID: "user-1", RefreshToken: "provider-refresh-a", LoginMethod: "oidc",
	})
	createTrackedSession(t, svc, trackedSessionSeed{
		SessionID: "sid-b", UserID: "user-1", RefreshToken: "provider-refresh-b", LoginMethod: "oidc-native",
	})
	createTrackedSession(t, svc, trackedSessionSeed{
		SessionID: "sid-phone", UserID: "user-1", RefreshToken: "self-refresh", LoginMethod: "phone",
	})
	revoker.onRevoke = func(_ context.Context, _, _, _ string) {
		sessions, listErr := tokenSvc.GetSessionStore().ListUserSessions(t.Context(), "user-1")
		require.NoError(t, listErr)
		assert.Empty(t, sessions, "local sessions must be revoked before provider calls")
	}

	err := svc.RevokeAllSessions(t.Context(), "user-1")

	require.NoError(t, err)
	assert.ElementsMatch(t, []providerTokenFamilyRevokeCall{
		{
			appKey:       oidc.ApplicationWeb,
			accessToken:  "access-sid-a",
			refreshToken: "provider-refresh-a",
		},
		{
			appKey:       oidc.ApplicationUniapp,
			accessToken:  "access-sid-b",
			refreshToken: "provider-refresh-b",
		},
	}, revoker.calls)
	sessions, err := tokenSvc.GetSessionStore().ListUserSessions(t.Context(), "user-1")
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestRevokeAllSessionsRevokesLocalSessionsWhenProviderRevokeFails(t *testing.T) {
	revoker := &fakeProviderTokenFamilyRevoker{err: errors.New("provider revoke failed")}
	svc, tokenSvc := newAuthServiceWithProviderRevoker(t, revoker)
	createTrackedSession(t, svc, trackedSessionSeed{
		SessionID: "sid-a", UserID: "user-1", RefreshToken: "provider-refresh-a", LoginMethod: "oidc",
	})

	err := svc.RevokeAllSessions(t.Context(), "user-1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider revoke failed")
	assert.Nil(t, requireSession(t, tokenSvc, "sid-a"))
}

func TestRevokeAllSessionsFailsClosedWhenLegacyProviderCredentialsAreMissing(t *testing.T) {
	revoker := &fakeProviderTokenFamilyRevoker{}
	svc, tokenSvc := newAuthServiceWithProviderRevoker(t, revoker)
	accessHash, err := hashTokenForSession("legacy-access")
	require.NoError(t, err)
	refreshHash, err := hashTokenForSession("legacy-refresh")
	require.NoError(t, err)
	require.NoError(t, tokenSvc.GetSessionStore().Create(t.Context(), token.SessionData{
		SessionID:            "sid-provider-credentials-missing",
		UserID:               "user-provider-credentials-missing",
		AccessTokenHash:      accessHash,
		AccessTokenExpiresAt: futureAccessTokenExpiryUnix(),
		RefreshTokenHash:     refreshHash,
		ProviderAppKey:       oidc.ApplicationWeb,
		LoginMethod:          "oidc",
	}))

	err = svc.RevokeAllSessions(t.Context(), "user-provider-credentials-missing")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider credentials are required")
	assert.Empty(t, revoker.calls)
	assert.Nil(t, requireSession(t, tokenSvc, "sid-provider-credentials-missing"))
}

func TestRevokeSessionRevokesLocalSessionWhenProviderRevokeFails(t *testing.T) {
	revoker := &fakeProviderTokenFamilyRevoker{err: errors.New("provider revoke failed")}
	svc, tokenSvc := newAuthServiceWithProviderRevoker(t, revoker)
	createTrackedSession(t, svc, trackedSessionSeed{
		SessionID: "sid-a", UserID: "user-1", RefreshToken: "provider-refresh-a", LoginMethod: "oidc",
	})

	err := svc.RevokeSession(t.Context(), "sid-a", "user-1", "access-sid-a", "provider-refresh-a", futureAccessTokenExpiry())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider revoke failed")
	assert.Nil(t, requireSession(t, tokenSvc, "sid-a"))
}

func TestRotateOIDCSessionRevokesNewProviderTokenFamilyOnLocalFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	revoker := &fakeProviderTokenFamilyRevoker{}
	svc, _ := newAuthServiceWithProviderRevoker(t, revoker)
	createTrackedSession(t, svc, trackedSessionSeed{
		SessionID: "sid-local-failure", UserID: "other-user", RefreshToken: "old-provider-refresh", LoginMethod: "oidc",
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	h := &Handler{svc: svc}

	ok := h.rotateOIDCSession(c, oidcSessionRotation{
		sessionID:       "sid-local-failure",
		appKey:          oidc.ApplicationWeb,
		userID:          "oidc-user-1",
		oldRefreshToken: "old-provider-refresh",
		payload: oidcRefreshPayload{
			rawIDToken:          "new-id-token",
			providerAccessToken: "new-provider-access",
			refreshToken:        "new-provider-refresh",
			userID:              "oidc-user-1",
		},
	})

	require.False(t, ok)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, []providerTokenFamilyRevokeCall{{
		appKey:       oidc.ApplicationWeb,
		accessToken:  "new-provider-access",
		refreshToken: "new-provider-refresh",
	}}, revoker.calls)
	assertNoIssuedTokenCookies(t, w)
}

func TestRotateOIDCSessionProviderTokenFamilyCompensationSurvivesRequestCancellation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	revoker := &fakeProviderTokenFamilyRevoker{}
	svc, _ := newAuthServiceWithProviderRevoker(t, revoker)
	createTrackedSession(t, svc, trackedSessionSeed{
		SessionID: "sid-canceled-rotation", UserID: "oidc-user-1", RefreshToken: "old-provider-refresh", LoginMethod: "oidc",
	})

	reqCtx, cancelReq := context.WithCancel(context.Background())
	cancelReq()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil).WithContext(reqCtx)
	h := &Handler{svc: svc}

	ok := h.rotateOIDCSession(c, oidcSessionRotation{
		sessionID:       "sid-canceled-rotation",
		appKey:          oidc.ApplicationWeb,
		userID:          "oidc-user-1",
		oldRefreshToken: "old-provider-refresh",
		payload: oidcRefreshPayload{
			rawIDToken:          "new-id-token",
			providerAccessToken: "new-provider-access",
			refreshToken:        "new-provider-refresh",
			userID:              "oidc-user-1",
		},
	})

	require.False(t, ok)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, []providerTokenFamilyRevokeCall{{
		appKey:       oidc.ApplicationWeb,
		accessToken:  "new-provider-access",
		refreshToken: "new-provider-refresh",
	}}, revoker.calls)
	assert.NoError(t, revoker.ctxErr)
}

type trackedSessionSeed struct {
	SessionID    string
	UserID       string
	RefreshToken string
	LoginMethod  string
}

func createTrackedSession(t *testing.T, svc *Service, seed trackedSessionSeed) {
	t.Helper()
	_, err := svc.CreateSession(
		context.Background(),
		seed.SessionID,
		seed.UserID,
		"access-"+seed.SessionID,
		seed.RefreshToken,
		seed.LoginMethod,
		"test-device",
	)
	require.NoError(t, err)
}

func requireSession(t *testing.T, tokenSvc *token.Service, sessionID string) *token.SessionData {
	t.Helper()
	session, err := tokenSvc.GetSessionStore().Get(t.Context(), sessionID)
	require.NoError(t, err)
	return session
}
