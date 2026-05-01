package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
)

const (
	CtxKeyMFAEnrollmentActive = "mfa_enrollment_active"
	CtxKeyMFAProofVerifiedAt  = "mfa_proof_verified_at"
)

type MFAContext struct {
	EnrollmentActive bool
	ProofVerifiedAt  time.Time
}

func SetMFAContext(c *gin.Context, ctx MFAContext) {
	c.Set(CtxKeyMFAEnrollmentActive, ctx.EnrollmentActive)
	if !ctx.ProofVerifiedAt.IsZero() {
		c.Set(CtxKeyMFAProofVerifiedAt, ctx.ProofVerifiedAt)
	}
}

func GetMFAEnrollmentActive(c *gin.Context) bool {
	value, exists := c.Get(CtxKeyMFAEnrollmentActive)
	if !exists {
		return false
	}
	active, ok := value.(bool)
	return ok && active
}

func GetMFAProofVerifiedAt(c *gin.Context) time.Time {
	value, exists := c.Get(CtxKeyMFAProofVerifiedAt)
	if !exists {
		return time.Time{}
	}
	verifiedAt, ok := value.(time.Time)
	if !ok {
		return time.Time{}
	}
	return verifiedAt
}
