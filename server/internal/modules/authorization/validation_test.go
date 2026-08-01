package authorization

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/pkg/fga"
)

func TestNormalizeCreateGrantInputAcceptsOnlyFixedRoleScopePairs(t *testing.T) {
	schoolID := int64(4111010006)
	sectionID := fga.ReviewModerationSectionID("4111010006")

	tests := []struct {
		name  string
		input CreateGrantInput
	}{
		{
			name: "school admin",
			input: CreateGrantInput{
				SubjectUserID: 1,
				ActorUserID:   2,
				Role:          RoleSchoolAdmin,
				SchoolID:      &schoolID,
				Reason:        "school operations",
			},
		},
		{
			name: "section moderator",
			input: CreateGrantInput{
				SubjectUserID: 1,
				ActorUserID:   2,
				Role:          RoleSectionModerator,
				SchoolID:      &schoolID,
				SectionID:     &sectionID,
				Reason:        "review moderation",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			normalized, err := normalizeCreateGrantInput(test.input)
			require.NoError(t, err)
			assert.Equal(t, test.input.Role, normalized.Role)
		})
	}
}

func TestNormalizeCreateGrantInputRejectsArbitraryOrMismatchedScope(t *testing.T) {
	schoolID := int64(4111010006)
	otherSchoolID := int64(4111010007)
	validSection := fga.ReviewModerationSectionID("4111010006")
	arbitrarySection := "finance"

	tests := []CreateGrantInput{
		{SubjectUserID: 1, ActorUserID: 2, Role: Role("owner"), Reason: "unsupported"},
		{SubjectUserID: 1, ActorUserID: 2, Role: RoleSchoolAdmin, Reason: "missing school"},
		{
			SubjectUserID: 1,
			ActorUserID:   2,
			Role:          RoleSectionAdmin,
			SchoolID:      &schoolID,
			SectionID:     &arbitrarySection,
			Reason:        "arbitrary tuple",
		},
		{
			SubjectUserID: 1,
			ActorUserID:   2,
			Role:          RoleSectionAdmin,
			SchoolID:      &otherSchoolID,
			SectionID:     &validSection,
			Reason:        "mismatched school",
		},
	}

	for index, input := range tests {
		_, err := normalizeCreateGrantInput(input)
		require.ErrorIs(t, err, ErrInvalidGrant, "case %d", index)
	}
}

func TestNormalizeCreateGrantInputRejectsProviderManagedSuperAdmin(t *testing.T) {
	_, err := normalizeCreateGrantInput(CreateGrantInput{
		SubjectUserID: 1,
		ActorUserID:   2,
		Role:          RoleSuperAdmin,
		Reason:        "manual platform administrator",
	})

	require.ErrorIs(t, err, ErrProviderManagedRole)
}

func TestTupleForGrantUsesOnlyInternalUserAndFixedRelations(t *testing.T) {
	schoolID := int64(4111010006)
	sectionID := fga.ReviewModerationSectionID("4111010006")

	tests := []struct {
		grant Grant
		want  fga.Tuple
	}{
		{
			grant: Grant{SubjectUserID: 42, Role: RoleSuperAdmin},
			want:  fga.Tuple{User: "user:42", Relation: "super_admin", Object: "ecosystem:stuhelper"},
		},
		{
			grant: Grant{SubjectUserID: 42, Role: RoleSchoolAdmin, SchoolID: &schoolID},
			want:  fga.Tuple{User: "user:42", Relation: "admin", Object: "school:4111010006"},
		},
		{
			grant: Grant{
				SubjectUserID: 42,
				Role:          RoleSectionModerator,
				SchoolID:      &schoolID,
				SectionID:     &sectionID,
			},
			want: fga.Tuple{
				User:     "user:42",
				Relation: "section_moderator",
				Object:   "section:school_4111010006_review_moderation",
			},
		},
	}

	for _, test := range tests {
		got, err := tupleForGrant(test.grant)
		require.NoError(t, err)
		assert.Equal(t, test.want, got)
	}
}
