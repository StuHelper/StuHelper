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

func TestServiceAccountCallAuditEvent(t *testing.T) {
	record := &credentialRecord{ID: 42, Name: KoishiRuntimeCredentialName}
	event := serviceAccountCallAuditEvent(serviceAccountCallAudit{
		Input: verifyInput{
			Audience:        "/api/v1/bot/qq-binding/consume",
			Scope:           ScopeBotQQBindingConsume,
			TokenHashPrefix: "12345678",
		},
		Credential: record,
		Result:     "success",
	})

	assert.Equal(t, "iam.service_account.call", string(event.Type))
	assert.Equal(t, "admin_operation", event.Category)
	assert.Equal(t, "service_account", event.ActorType)
	assert.Equal(t, "iam.service_account", event.ResourceType)
	assert.Equal(t, KoishiRuntimeCredentialName, event.ResourceID)
	assert.Equal(t, "call", event.Action)
	assert.Equal(t, "success", event.Result)
	assert.Equal(t, int64(42), event.Details["credential_id"])
	assert.Equal(t, KoishiRuntimeCredentialName, event.Details["name"])
	assert.Equal(t, ScopeBotQQBindingConsume, event.Details["scope"])
	assert.NotContains(t, event.Details, "token_hash_prefix")
}

func TestServiceAccountCallAuditEventForUnknownCredentialOnlyKeepsHashPrefix(t *testing.T) {
	event := serviceAccountCallAuditEvent(serviceAccountCallAudit{
		Input: verifyInput{
			Audience:        "/api/v1/bot/qq-binding/consume",
			Scope:           ScopeBotQQBindingConsume,
			TokenHashPrefix: "abcdef12",
		},
		Result: "failure",
		Reason: invalidCredentialReason,
	})

	assert.Equal(t, "unknown", event.ResourceID)
	assert.Equal(t, "failure", event.Result)
	assert.Equal(t, invalidCredentialReason, event.Reason)
	assert.Equal(t, "abcdef12", event.Details["token_hash_prefix"])
	assert.NotContains(t, event.Details, "name")
}
