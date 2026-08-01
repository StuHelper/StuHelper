package casdoor

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/StuHelper/StuHelper/server/internal/pkg/audit"
	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
)

func TestCasdoorAdminAuditEventRedactsCredentialsAndRecordsPurpose(t *testing.T) {
	t.Parallel()

	credential := Credential{
		Purpose:      PurposeUserProfile,
		Endpoint:     "https://sso.example.test",
		ClientID:     "user-profile-client",
		ClientSecret: "super-secret",
		Organization: "stuhelper",
		Application:  "casdoor-admin-user-profile",
	}

	event := casdoorAdminAuditEvent(credential, "update user profile", nil)

	assert.Equal(t, "iam.casdoor_admin_api.call", string(event.Type))
	assert.Equal(t, "admin_operation", event.Category)
	assert.Equal(t, "system", event.ActorType)
	assert.Equal(t, string(PurposeUserProfile), event.UserID)
	assert.Equal(t, "success", event.Result)
	assert.Equal(t, "user-profile-client", event.Details["client_id"])
	assert.NotContains(t, event.Details, "client_secret")
	assert.NotContains(t, event.Details, "certificate")
}

func TestCasdoorAdminAuditEventCapturesFailureReason(t *testing.T) {
	t.Parallel()

	event := casdoorAdminAuditEvent(
		Credential{Purpose: PurposeUserLookup, Organization: "stuhelper", Application: "casdoor-admin-user-lookup"},
		"lookup user subject-1",
		errors.New("casdoor unavailable"),
	)

	assert.Equal(t, "failure", event.Result)
	assert.Equal(t, "casdoor unavailable", event.Reason)
	assert.Equal(t, string(PurposeUserLookup), event.Details["purpose"])
}

func TestCasdoorAdminAuditEventUsesContextRequestID(t *testing.T) {
	t.Parallel()

	ctx := logger.WithRequestID(context.Background(), "req-casdoor-1")
	event := audit.EventFromContext(ctx, casdoorAdminAuditEvent(
		Credential{Purpose: PurposeUserProfile, Organization: "stuhelper", Application: "casdoor-admin-user-profile"},
		"update user profile",
		nil,
	))

	assert.Equal(t, "req-casdoor-1", event.RequestID)
}

func TestCasdoorApplicationAuditEventRecordsLifecycleWithoutSecrets(t *testing.T) {
	t.Parallel()

	credential := Credential{
		Purpose:      PurposeAppProvisioning,
		ClientID:     "app-provisioning-client",
		ClientSecret: "super-secret",
		Certificate:  "private-cert",
		Organization: "stuhelper",
		Application:  "casdoor-admin-app-provisioning",
	}

	event := casdoorApplicationAuditEvent(credential, "third-party-demo", casdoorApplicationActionCreate)

	assert.Equal(t, "iam.casdoor_app.created", string(event.Type))
	assert.Equal(t, "admin_operation", event.Category)
	assert.Equal(t, "system", event.ActorType)
	assert.Equal(t, string(PurposeAppProvisioning), event.UserID)
	assert.Equal(t, "casdoor.application", event.ResourceType)
	assert.Equal(t, "third-party-demo", event.ResourceID)
	assert.Equal(t, "created", event.Action)
	assert.Equal(t, "success", event.Result)
	assert.Equal(t, string(PurposeAppProvisioning), event.Details["purpose"])
	assert.Equal(t, "stuhelper", event.Details["organization"])
	assert.Equal(t, "casdoor-admin-app-provisioning", event.Details["admin_application"])
	assert.Equal(t, "third-party-demo", event.Details["app_name"])
	assert.NotContains(t, event.Details, "client_secret")
	assert.NotContains(t, event.Details, "certificate")
	assert.NotContains(t, event.Details, "secret")
}
