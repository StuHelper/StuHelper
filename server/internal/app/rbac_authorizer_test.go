package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/platform/authorization"
)

func TestNewRBACAuthorizerDisablesMFAGatesInDevelopment(t *testing.T) {
	authorizer := newRBACAuthorizer("development")

	decision := authorizer.Authorize(
		context.Background(),
		authorization.Subject{UserID: "user-1"},
		authorization.ActionStepUpMFARequire,
		authorization.StepUpMFAResource(authorization.DefaultStepUpWindow),
	)

	assert.True(t, decision.Allow)
	assert.NoError(t, decision.Error)
	assert.Equal(t, "mfa enforcement disabled", decision.Reason)
}

func TestNewRBACAuthorizerKeepsMFAGatesOutsideDevelopment(t *testing.T) {
	authorizer := newRBACAuthorizer("production")

	decision := authorizer.Authorize(
		context.Background(),
		authorization.Subject{UserID: "user-1"},
		authorization.ActionStepUpMFARequire,
		authorization.StepUpMFAResource(authorization.DefaultStepUpWindow),
	)

	require.ErrorIs(t, decision.Error, authorization.ErrMFAEnrollmentRequired)
	assert.False(t, decision.Allow)
}
