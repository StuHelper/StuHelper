package rbac

import (
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
)

func TestSubjectFromGinIncludesMFAState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	proofTime := time.Date(2026, 5, 2, 13, 0, 0, 0, time.UTC)
	c, _ := gin.CreateTestContext(nil)
	c.Set(middleware.CtxKeyUserID, "user-1")
	c.Set(middleware.CtxKeyAppID, "stuhelper-web")
	middleware.SetMFAContext(c, middleware.MFAContext{
		EnrollmentActive: true,
		ProofVerifiedAt:  proofTime,
	})

	subject := subjectFromGin(c)

	assert.Equal(t, "user-1", subject.UserID)
	assert.Equal(t, "stuhelper-web", subject.AppID)
	assert.True(t, subject.MFAEnrollmentActive)
	assert.Equal(t, proofTime, subject.MFAProofVerifiedAt)
}
