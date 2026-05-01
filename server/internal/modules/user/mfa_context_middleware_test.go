package user

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
)

func TestMFAContextMiddlewareLoadsActiveEnrollmentAndKeepsProof(t *testing.T) {
	proofAt := time.Now().Add(-time.Minute).UTC()
	repo := &fakeMFAContextRepo{
		userID:     42,
		enrollment: &MFAEnrollment{Active: true, Methods: []string{MFAMethodTOTP}},
	}
	called := false

	w := exerciseMFAContextMiddleware(t, repo, "casdoor-admin", proofAt, func(c *gin.Context) {
		called = true
		assert.True(t, middleware.GetMFAEnrollmentActive(c))
		assert.Equal(t, proofAt, middleware.GetMFAProofVerifiedAt(c))
	})

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
	assert.Equal(t, "casdoor-admin", repo.seenSubject)
	assert.Equal(t, int64(42), repo.seenEnrollmentUserID)
}

func TestMFAContextMiddlewareTreatsResetRequiredAsInactive(t *testing.T) {
	repo := &fakeMFAContextRepo{
		userID: 42,
		enrollment: &MFAEnrollment{
			Active:        true,
			Methods:       []string{MFAMethodTOTP},
			ResetRequired: true,
		},
	}

	w := exerciseMFAContextMiddleware(t, repo, "casdoor-admin", time.Time{}, func(c *gin.Context) {
		assert.False(t, middleware.GetMFAEnrollmentActive(c))
	})

	require.Equal(t, http.StatusOK, w.Code)
}

func TestMFAContextMiddlewareRejectsUnprovisionedUser(t *testing.T) {
	repo := &fakeMFAContextRepo{userErr: pgx.ErrNoRows}

	w := exerciseMFAContextMiddleware(t, repo, "casdoor-admin", time.Time{}, func(c *gin.Context) {
		t.Fatal("handler must not run")
	})

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestMFAContextMiddlewareReturns503OnEnrollmentError(t *testing.T) {
	repo := &fakeMFAContextRepo{
		userID:        42,
		enrollmentErr: errors.New("database unavailable"),
	}

	w := exerciseMFAContextMiddleware(t, repo, "casdoor-admin", time.Time{}, func(c *gin.Context) {
		t.Fatal("handler must not run")
	})

	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestMFAContextMiddlewareRequiresAuthenticatedSubject(t *testing.T) {
	repo := &fakeMFAContextRepo{userID: 42}

	w := exerciseMFAContextMiddleware(t, repo, "", time.Time{}, func(c *gin.Context) {
		t.Fatal("handler must not run")
	})

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func exerciseMFAContextMiddleware(
	t *testing.T,
	repo *fakeMFAContextRepo,
	casdoorSubject string,
	proofAt time.Time,
	handler gin.HandlerFunc,
) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		if casdoorSubject != "" {
			c.Set(middleware.CtxKeyUserID, casdoorSubject)
		}
		middleware.SetMFAProofVerifiedAt(c, proofAt)
		c.Next()
	})
	engine.GET("/test", MFAContextMiddleware(repo), handler)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	engine.ServeHTTP(w, req)
	return w
}

type fakeMFAContextRepo struct {
	userID               int64
	userErr              error
	enrollment           *MFAEnrollment
	enrollmentErr        error
	seenSubject          string
	seenEnrollmentUserID int64
}

func (f *fakeMFAContextRepo) GetInternalUserID(_ context.Context, casdoorSubject string) (int64, error) {
	f.seenSubject = casdoorSubject
	if f.userErr != nil {
		return 0, f.userErr
	}
	return f.userID, nil
}

func (f *fakeMFAContextRepo) GetMFAEnrollment(_ context.Context, userID int64) (*MFAEnrollment, error) {
	f.seenEnrollmentUserID = userID
	if f.enrollmentErr != nil {
		return nil, f.enrollmentErr
	}
	return f.enrollment, nil
}
