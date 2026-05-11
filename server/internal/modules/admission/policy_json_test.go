package admission

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAdmissionPolicyJSONMatchesOpenAPIShape(t *testing.T) {
	policy := normalizeAdmissionPolicyForOutput(AdmissionPolicy{
		ID:                         " policy-1 ",
		Platform:                   " qq ",
		GuildID:                    " guild-1 ",
		AutoApproveJoin:            true,
		InitialMuteDurationSeconds: 10,
		LinkWaitSeconds:            20,
		SubmissionWaitSeconds:      30,
		ManualReviewTimeoutSeconds: 40,
		ReminderIntervalSeconds:    50,
		FailedJoinLimit:            3,
		FreshmanChannelEnabled:     true,
		FreshmanChannelClosesAt:    time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC),
		FreshmanDefaultExpiresAt:   time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC),
		ForwardRawMaterialToQQ:     true,
		MaxMaterialBytes:           1024,
		MaxExtensionDays:           90,
	})

	payload, err := json.Marshal(policy)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(payload, &decoded))
	require.Equal(t, "policy-1", decoded["id"])
	require.Equal(t, "qq", decoded["platform"])
	require.Equal(t, "guild-1", decoded["guildID"])
	require.Equal(t, []any{}, decoded["managementGuildIDs"])
	require.NotContains(t, decoded, "ID")
	require.NotContains(t, decoded, "ManagementGuildIDs")
	require.NotContains(t, decoded, "schoolID")
}

func TestFreshmanApplicationJSONMatchesOpenAPIShape(t *testing.T) {
	app := FreshmanApplication{
		ID:                  "freshman-1",
		UserID:              42,
		SchoolID:            1,
		Status:              FreshmanApplicationPending,
		ApplicantNameMasked: "A***",
		MaterialType:        MaterialAdmissionNotice,
		CreatedAt:           time.Date(2026, 5, 11, 0, 0, 0, 0, time.UTC),
	}

	payload, err := json.Marshal(app)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(payload, &decoded))
	require.Equal(t, "freshman-1", decoded["id"])
	require.Equal(t, float64(42), decoded["userID"])
	require.Equal(t, "A***", decoded["applicantNameMasked"])
	require.NotContains(t, decoded, "ID")
	require.NotContains(t, decoded, "ApplicantNameMasked")
}
