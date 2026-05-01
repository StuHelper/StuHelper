package serviceaccount

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServiceAccountCredentialAuditEvent(t *testing.T) {
	event := serviceAccountCredentialAuditEvent(KoishiRuntimeCredentialName, 42, "rotated")

	assert.Equal(t, "iam.service_account.rotated", string(event.Type))
	assert.Equal(t, "admin_operation", event.Category)
	assert.Equal(t, "system", event.ActorType)
	assert.Equal(t, "iam.service_account", event.ResourceType)
	assert.Equal(t, KoishiRuntimeCredentialName, event.ResourceID)
	assert.Equal(t, "rotated", event.Action)
	assert.Equal(t, "success", event.Result)
	assert.Equal(t, int64(42), event.Details["credential_id"])
	assert.Equal(t, KoishiRuntimeCredentialName, event.Details["name"])
}
