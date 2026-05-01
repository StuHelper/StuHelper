package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSetClaimsToContextPropagatesMFAProofOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/me", nil)
	proofAt := time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)

	setClaimsToContext(c, &authResult{
		userID:      "casdoor-user-1",
		username:    "alice",
		displayName: "Alice",
		roles:       []string{"school_admin"},
		mfaProofAt:  proofAt,
	})

	assert.Equal(t, "casdoor-user-1", GetUserID(c))
	assert.Equal(t, "alice", GetUsername(c))
	assert.True(t, GetMFAProofVerifiedAt(c).Equal(proofAt))
	assert.False(t, GetMFAEnrollmentActive(c))
}
