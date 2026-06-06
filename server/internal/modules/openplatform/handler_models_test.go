package openplatform

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestScopeRequestToJSONAddsScopeDefinition(t *testing.T) {
	resp := scopeRequestToJSON(ScopeRequest{
		ID:     1,
		Scope:  ScopeEmailRead,
		Reason: "send email notifications",
		Status: ScopeStatusApproved,
	})

	definition := scopeCatalog[ScopeEmailRead]
	require.Equal(t, ScopeEmailRead, resp.Scope)
	require.Equal(t, definition.DisplayName, resp.DisplayName)
	require.Equal(t, definition.Sensitivity, resp.Sensitivity)
	require.Equal(t, definition.Fields, resp.Fields)
}

func TestScopeRequestToJSONLeavesUnknownScopePresentationEmpty(t *testing.T) {
	resp := scopeRequestToJSON(ScopeRequest{
		ID:     1,
		Scope:  "legacy.unknown.scope",
		Status: ScopeStatusApproved,
	})

	require.Equal(t, "legacy.unknown.scope", resp.Scope)
	require.Empty(t, resp.DisplayName)
	require.Empty(t, resp.Sensitivity)
	require.Empty(t, resp.Fields)
}

func TestUserConsentsToJSONAddsScopeDefinition(t *testing.T) {
	grantedAt := time.Now().UTC()
	resp := userConsentsToJSON([]UserAuthorizedApp{
		{
			App: &App{ID: 10, ClientID: "client-id", DisplayName: "App"},
			Scopes: []UserConsentScope{
				{Scope: ScopePhoneRead, GrantedAt: grantedAt, GrantSource: "consent", Reason: "phone verification"},
			},
		},
	})

	require.Len(t, resp.Apps, 1)
	require.Len(t, resp.Apps[0].Scopes, 1)
	scope := resp.Apps[0].Scopes[0]
	definition := scopeCatalog[ScopePhoneRead]
	require.Equal(t, ScopePhoneRead, scope.Scope)
	require.Equal(t, definition.DisplayName, scope.DisplayName)
	require.Equal(t, definition.Sensitivity, scope.Sensitivity)
	require.Equal(t, definition.Fields, scope.Fields)
}

func TestAdminUserConsentListToJSONAddsScopeDefinition(t *testing.T) {
	grantedAt := time.Now().UTC()
	resp := adminUserConsentListToJSON(AdminUserConsentListResult{
		List: []AdminUserAuthorizedApp{
			{
				UserID: 42,
				App:    &App{ID: 10, ClientID: "client-id", DisplayName: "App"},
				Scopes: []UserConsentScope{
					{Scope: ScopeStudentSchoolRead, GrantedAt: grantedAt, GrantSource: "consent", Reason: "school sync"},
				},
			},
		},
		Total: 1,
	})

	require.Len(t, resp.List, 1)
	require.Len(t, resp.List[0].Scopes, 1)
	scope := resp.List[0].Scopes[0]
	definition := scopeCatalog[ScopeStudentSchoolRead]
	require.Equal(t, ScopeStudentSchoolRead, scope.Scope)
	require.Equal(t, definition.DisplayName, scope.DisplayName)
	require.Equal(t, definition.Sensitivity, scope.Sensitivity)
	require.Equal(t, definition.Fields, scope.Fields)
}

func TestScopeDefinitionForResponseCopiesFields(t *testing.T) {
	first, ok := scopeDefinitionForResponse(ScopeEmailRead)
	require.True(t, ok)
	require.NotEmpty(t, first.Fields)

	first.Fields[0] = "mutated"

	second, ok := scopeDefinitionForResponse(ScopeEmailRead)
	require.True(t, ok)
	require.Equal(t, scopeCatalog[ScopeEmailRead].Fields, second.Fields)
}
