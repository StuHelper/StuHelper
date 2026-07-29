package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/modules/user"
	"github.com/StuHelper/StuHelper/server/internal/pkg/config"
	"github.com/StuHelper/StuHelper/server/internal/pkg/middleware"
)

func TestAdminMFAMiddlewaresSkippedInDevelopment(t *testing.T) {
	require.Empty(t, adminMFAMiddlewares(config.EnvDevelopment, nil))
}

func TestAdminMFAMiddlewaresRequireRepositoryOutsideDevelopment(t *testing.T) {
	require.Panics(t, func() {
		adminMFAMiddlewares(config.EnvProduction, nil)
	})
	require.Panics(t, func() {
		adminMFAMiddlewares(config.EnvProdParity, nil)
	})
}

func TestAdminMFAMiddlewaresEnforcePrivilegedEnrollmentAndFreshProof(t *testing.T) {
	configureRBACAuthorizer(config.EnvProduction)
	t.Cleanup(func() {
		configureRBACAuthorizer(config.EnvProduction)
	})

	t.Run("missing enrollment", func(t *testing.T) {
		w, called := exerciseAdminMFAMiddlewares(
			t,
			&adminMFAContextRepo{userID: 42},
			time.Time{},
		)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.False(t, called)
	})

	t.Run("missing fresh proof", func(t *testing.T) {
		w, called := exerciseAdminMFAMiddlewares(
			t,
			&adminMFAContextRepo{
				userID:     42,
				enrollment: &user.MFAEnrollment{Active: true, Methods: []string{user.MFAMethodTOTP}},
			},
			time.Time{},
		)

		assert.Equal(t, http.StatusPreconditionRequired, w.Code)
		assert.False(t, called)
	})

	t.Run("active enrollment and fresh proof", func(t *testing.T) {
		w, called := exerciseAdminMFAMiddlewares(
			t,
			&adminMFAContextRepo{
				userID:     42,
				enrollment: &user.MFAEnrollment{Active: true, Methods: []string{user.MFAMethodTOTP}},
			},
			time.Now().Add(-time.Minute),
		)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.True(t, called)
	})
}

func exerciseAdminMFAMiddlewares(
	t *testing.T,
	repo *adminMFAContextRepo,
	proofAt time.Time,
) (*httptest.ResponseRecorder, bool) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set(middleware.CtxKeyUserID, "casdoor-admin")
		c.Set(middleware.CtxKeyRoles, []string{"super_admin"})
		middleware.SetMFAProofVerifiedAt(c, proofAt)
		c.Next()
	})

	called := false
	handlers := append(adminMFAMiddlewares(config.EnvProduction, repo), func(c *gin.Context) {
		called = true
		c.Status(http.StatusOK)
	})
	engine.GET("/admin", handlers...)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	engine.ServeHTTP(w, req)
	return w, called
}

type adminMFAContextRepo struct {
	userID        int64
	userErr       error
	enrollment    *user.MFAEnrollment
	enrollmentErr error
}

func (r *adminMFAContextRepo) GetInternalUserID(context.Context, string) (int64, error) {
	if r.userErr != nil {
		return 0, r.userErr
	}
	return r.userID, nil
}

func (r *adminMFAContextRepo) GetMFAEnrollment(context.Context, int64) (*user.MFAEnrollment, error) {
	if r.enrollmentErr != nil {
		return nil, r.enrollmentErr
	}
	return r.enrollment, nil
}

var _ user.MFAContextRepository = (*adminMFAContextRepo)(nil)
