package review

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/user"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/systemconfig"
)

func TestBuildReviewAccessPolicy_UsesConfiguredValues(t *testing.T) {
	policy, err := buildReviewAccessPolicy(
		[]user.SchoolConfig{
			{SchoolID: "10006", Enabled: true},
			{SchoolID: "10007", Enabled: true},
		},
		[]user.SystemConfig{
			{Key: systemconfig.ReviewAccessSchoolIDsKey, Value: `["10007","10008"]`},
			{Key: systemconfig.ReviewPreviewTitleCharsKey, Value: "36"},
			{Key: systemconfig.ReviewPreviewContentCharsKey, Value: "180"},
			{Key: systemconfig.ReviewPreviewContentPercentKey, Value: "40"},
		},
	)
	require.NoError(t, err)

	assert.True(t, policy.AllowsSchool("10007"))
	assert.True(t, policy.AllowsSchool("10008"))
	assert.False(t, policy.AllowsSchool("10006"))
	assert.Equal(t, 36, policy.PreviewTitleRunes)
	assert.Equal(t, 180, policy.PreviewContentRunes)
	assert.Equal(t, 40, policy.PreviewContentPct)
}

func TestBuildReviewAccessPolicy_UsesEnabledSchoolsAndPreviewKeys(t *testing.T) {
	policy, err := buildReviewAccessPolicy(
		[]user.SchoolConfig{
			{SchoolID: "10006", Enabled: true},
			{SchoolID: "10007", Enabled: true},
		},
		[]user.SystemConfig{
			{Key: systemconfig.ReviewPreviewContentCharsKey, Value: "96"},
			{Key: systemconfig.ReviewPreviewContentPercentKey, Value: "25"},
		},
	)
	require.NoError(t, err)

	assert.True(t, policy.AllowsSchool("10006"))
	assert.True(t, policy.AllowsSchool("10007"))
	assert.Equal(t, systemconfig.DefaultReviewAccessPolicySnapshot().PreviewTitleRunes, policy.PreviewTitleRunes)
	assert.Equal(t, 96, policy.PreviewContentRunes)
	assert.Equal(t, 25, policy.PreviewContentPct)
}

func TestPreviewText_RespectsConfiguredPercent(t *testing.T) {
	value := strings.Repeat("评", 200)
	preview := previewText(value, 120, 20)
	assert.Equal(t, strings.Repeat("评", 40)+"...", preview)
}
