package authorization

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
)

func TestAuthorizeCapability(t *testing.T) {
	service := NewService()
	subject := Subject{UserID: "user-1", Capabilities: []string{capability.AdminLogsView}}

	decision := service.Authorize(context.Background(), subject, ActionCapabilityRequire, CapabilityResource(capability.AdminLogsView))

	assert.True(t, decision.Allow)
	assert.NoError(t, decision.Error)
}

func TestAuthorizeCapabilityDenied(t *testing.T) {
	service := NewService()
	subject := Subject{UserID: "user-1", Capabilities: []string{capability.ReviewListBrief}}

	decision := service.Authorize(context.Background(), subject, ActionCapabilityRequire, CapabilityResource(capability.AdminLogsView))

	assert.False(t, decision.Allow)
	assert.NoError(t, decision.Error)
	assert.Equal(t, "capability denied", decision.Reason)
}

func TestAuthorizeAnyCapability(t *testing.T) {
	service := NewService()
	subject := Subject{UserID: "user-1", Capabilities: []string{capability.AdminLogsView}}

	decision := service.Authorize(
		context.Background(),
		subject,
		ActionCapabilityRequireAny,
		AnyCapabilityResource(capability.AdminReviewsManage, capability.AdminLogsView),
	)

	assert.True(t, decision.Allow)
}

func TestAuthorizeGlobalCapabilityRequiresGlobalGrant(t *testing.T) {
	service := NewService()
	subject := Subject{
		UserID: "user-1",
		CapabilityGrants: []capability.Grant{{
			Name:   capability.UserSystemRead,
			Global: true,
		}},
	}

	decision := service.Authorize(context.Background(), subject, ActionCapabilityRequireGlobal, GlobalCapabilityResource(capability.UserSystemRead))

	assert.True(t, decision.Allow)
}

func TestAuthorizeRejectsMissingSubject(t *testing.T) {
	service := NewService()

	decision := service.Authorize(context.Background(), Subject{}, ActionCapabilityRequire, CapabilityResource(capability.AdminLogsView))

	require.ErrorIs(t, decision.Error, ErrInvalidSubject)
	assert.False(t, decision.Allow)
}

func TestBatchAuthorizePreservesOrder(t *testing.T) {
	service := NewService()
	subject := Subject{UserID: "user-1", Capabilities: []string{capability.AdminLogsView}}

	decisions := service.BatchAuthorize(context.Background(), subject, []Check{
		{Action: ActionCapabilityRequire, Resource: CapabilityResource(capability.AdminLogsView)},
		{Action: ActionCapabilityRequire, Resource: CapabilityResource(capability.AdminReviewsManage)},
	})

	require.Len(t, decisions, 2)
	assert.True(t, decisions[0].Allow)
	assert.False(t, decisions[1].Allow)
}
