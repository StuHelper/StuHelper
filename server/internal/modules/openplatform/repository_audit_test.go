package openplatform

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuditEventTypeAllowlists(t *testing.T) {
	t.Run("developer app audit events include enterprise operations events", func(t *testing.T) {
		requireNoDuplicateAuditEventTypes(t, developerAppAuditEventTypes)

		assert.Contains(t, developerAppAuditEventTypes, "open_platform.app.approved")
		assert.Contains(t, developerAppAuditEventTypes, "open_platform.app.approved_app_ensured")
		assert.Contains(t, developerAppAuditEventTypes, "open_platform.app.token_probe.runtime.failed")
		assert.Contains(t, developerAppAuditEventTypes, "open_platform.resource_access.checked")
		assert.Contains(t, developerAppAuditEventTypes, "open_platform.resource_access.granted")
		assert.Contains(t, developerAppAuditEventTypes, "open_platform.resource_access.revoked")
	})

	t.Run("user consent audit events stay limited to user-visible consent and disclosure", func(t *testing.T) {
		requireNoDuplicateAuditEventTypes(t, userConsentAuditEventTypes)

		assert.Equal(t, []string{
			"open_platform.consent.granted",
			"open_platform.consent.denied",
			"open_platform.consent.revoked",
			"open_platform.disclosure.granted",
			"open_platform.disclosure.denied",
			"open_platform.disclosure.replay_detected",
		}, userConsentAuditEventTypes)
	})
}

func requireNoDuplicateAuditEventTypes(t *testing.T, eventTypes []string) {
	t.Helper()

	seen := make(map[string]struct{}, len(eventTypes))
	for _, eventType := range eventTypes {
		require.NotEmpty(t, eventType)
		if _, ok := seen[eventType]; ok {
			t.Fatalf("duplicate Open Platform audit event type %q", eventType)
		}
		seen[eventType] = struct{}{}
	}
}
