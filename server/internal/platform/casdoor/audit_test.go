package casdoor

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCasdoorAdminAuditEventRedactsCredentialsAndRecordsPurpose(t *testing.T) {
	t.Parallel()

	credential := Credential{
		Purpose:      PurposeRoleSync,
		Endpoint:     "https://sso.example.test",
		ClientID:     "role-sync-client",
		ClientSecret: "super-secret",
		Organization: "stuhelper",
		Application:  "casdoor-admin-role-sync",
	}

	event := casdoorAdminAuditEvent(credential, "update role users verified_student", nil)

	assert.Equal(t, "iam.casdoor.admin_api", string(event.Type))
	assert.Equal(t, "admin_operation", event.Category)
	assert.Equal(t, "system", event.ActorType)
	assert.Equal(t, string(PurposeRoleSync), event.UserID)
	assert.Equal(t, "success", event.Result)
	assert.Equal(t, "role-sync-client", event.Details["client_id"])
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
