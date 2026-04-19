package user

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appmiddleware "git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
)

func setupQQBindingUserRouter(t *testing.T, repo Repo) *gin.Engine {
	t.Helper()

	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	h := NewHandler(svc, nil, nil, nil)
	r := gin.New()
	api := r.Group("/api/v1")
	authMW := func(c *gin.Context) {
		c.Set(appmiddleware.CtxKeyUserID, "external-user-123")
		c.Next()
	}
	h.RegisterRoutes(api, authMW)
	return r
}

func setupQQBindingBotRouter(t *testing.T, repo Repo, serviceToken string) *gin.Engine {
	t.Helper()

	svc, err := NewService(repo, []byte("test-hmac-key-at-least-32-chars!"), &fakeEncryptor{})
	require.NoError(t, err)

	h := NewBotHandler(svc, serviceToken)
	r := gin.New()
	api := r.Group("/api/v1")
	h.RegisterRoutes(api)
	return r
}

func TestHandleCreateQQBindingCode_ReturnsCreatedCode(t *testing.T) {
	repo := newQQBindingMockRepo()
	repo.onGetInternalUserID = func(_ context.Context, externalID string) (int64, error) {
		assert.Equal(t, "external-user-123", externalID)
		return 42, nil
	}
	repo.onUpsertQQBindingCode = func(_ context.Context, code *QQBindingCode) error {
		assert.Equal(t, int64(42), code.UserID)
		return nil
	}

	r := setupQQBindingUserRouter(t, repo)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/user/qq-binding/code", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Code      string    `json:"code"`
			ExpiresAt time.Time `json:"expiresAt"`
		} `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.NotEmpty(t, resp.Data.Code)
	assert.False(t, resp.Data.ExpiresAt.IsZero())
}

func TestHandleGetQQBinding_ReturnsNotFoundWhenUserHasNoBinding(t *testing.T) {
	repo := newQQBindingMockRepo()
	repo.onGetInternalUserID = func(_ context.Context, _ string) (int64, error) {
		return 42, nil
	}

	r := setupQQBindingUserRouter(t, repo)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/qq-binding", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleGetQQBinding_ReturnsBinding(t *testing.T) {
	repo := newQQBindingMockRepo()
	boundAt := time.Now().Add(-time.Hour).UTC()
	repo.onGetInternalUserID = func(_ context.Context, _ string) (int64, error) {
		return 42, nil
	}
	repo.onGetQQBindingByUserID = func(_ context.Context, userID int64) (*QQBinding, error) {
		return &QQBinding{
			UserID:     userID,
			QQID:       "123456789",
			QQNickname: ptr("航小伴"),
			BoundAt:    boundAt,
		}, nil
	}

	r := setupQQBindingUserRouter(t, repo)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/qq-binding", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"qqID":"123456789"`)
}

func TestBotConsumeQQBinding_RejectsMissingServiceToken(t *testing.T) {
	repo := newQQBindingMockRepo()
	r := setupQQBindingBotRouter(t, repo, "")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/bot/qq-binding/consume", strings.NewReader(`{"code":"ABCD1234","qqID":"123456789"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestBotConsumeQQBinding_RejectsUnauthorizedToken(t *testing.T) {
	repo := newQQBindingMockRepo()
	r := setupQQBindingBotRouter(t, repo, "expected-token")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/bot/qq-binding/consume", strings.NewReader(`{"code":"ABCD1234","qqID":"123456789"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer wrong-token")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestBotConsumeQQBinding_ReturnsBindingResult(t *testing.T) {
	repo := newQQBindingMockRepo()
	repo.onWithTx = func(ctx context.Context, fn func(ctx context.Context, tx pgx.Tx) error) error {
		return fn(ctx, nil)
	}
	repo.onGetQQBindingCodeByHashTx = func(_ context.Context, _ pgx.Tx, _ string) (*QQBindingCode, error) {
		return &QQBindingCode{
			UserID:    42,
			CodeHash:  "hash",
			ExpiresAt: time.Now().Add(time.Minute),
		}, nil
	}
	repo.onCreateQQBindingTx = func(_ context.Context, _ pgx.Tx, _ *QQBinding) error {
		return nil
	}
	repo.onMarkQQBindingCodeConsumedTx = func(_ context.Context, _ pgx.Tx, _ int64, _ time.Time) error {
		return nil
	}
	repo.onGetQQBindingByQQID = func(_ context.Context, qqID string) (*QQBinding, error) {
		return &QQBinding{UserID: 42, QQID: qqID}, nil
	}
	repo.onGetProfileByUserID = func(_ context.Context, _ int64) (*Profile, error) {
		return &Profile{UserID: 42, VerificationStatus: StatusVerified}, nil
	}

	r := setupQQBindingBotRouter(t, repo, "expected-token")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/bot/qq-binding/consume", strings.NewReader(`{"code":"ABCD1234","qqID":"123456789","qqNickname":"航小伴"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer expected-token")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"verificationState":"verified"`)
}

func TestBotGetQQVerification_ReturnsVerificationState(t *testing.T) {
	repo := newQQBindingMockRepo()
	repo.onGetQQBindingByQQID = func(_ context.Context, qqID string) (*QQBinding, error) {
		return &QQBinding{UserID: 42, QQID: qqID}, nil
	}
	repo.onGetProfileByUserID = func(_ context.Context, _ int64) (*Profile, error) {
		return &Profile{UserID: 42, VerificationStatus: StatusPending}, nil
	}

	r := setupQQBindingBotRouter(t, repo, "expected-token")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/bot/qq-users/123456789/verification", nil)
	req.Header.Set("Authorization", "Bearer expected-token")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"verificationState":"bound_unverified"`)
}
