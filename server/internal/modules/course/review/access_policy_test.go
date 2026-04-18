package review

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/reviewaccess"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/systemconfig"
)

type fakeReviewAccessReader struct {
	subject *reviewaccess.Subject
}

func (f fakeReviewAccessReader) ListReviewAccessSchoolConfigs(context.Context) ([]reviewaccess.SchoolConfig, error) {
	return []reviewaccess.SchoolConfig{{SchoolID: 10006}}, nil
}

func (f fakeReviewAccessReader) ListReviewAccessSystemConfigs(context.Context) ([]reviewaccess.SystemConfig, error) {
	return nil, nil
}

func (f fakeReviewAccessReader) GetReviewAccessSubject(context.Context, string) (*reviewaccess.Subject, error) {
	return f.subject, nil
}

func TestBuildReviewAccessPolicy_UsesConfiguredValues(t *testing.T) {
	policy, err := buildReviewAccessPolicy(
		[]reviewaccess.SchoolConfig{
			{SchoolID: 10006},
			{SchoolID: 10007},
		},
		[]reviewaccess.SystemConfig{
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
		[]reviewaccess.SchoolConfig{
			{SchoolID: 10006},
			{SchoolID: 10007},
		},
		[]reviewaccess.SystemConfig{
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

func TestResolveAccessFacts_RequiresCapabilityAndVerificationFacts(t *testing.T) {
	service := &Service{
		accessReader: fakeReviewAccessReader{
			subject: &reviewaccess.Subject{
				IdentityVerified: true,
				StudentVerified:  true,
				SchoolID:         int64Ptr(10006),
			},
		},
	}

	facts, err := service.ResolveAccessFacts(context.Background(), "user-1", []string{
		capability.ReviewListFull,
		capability.ReviewCreate,
		capability.ReviewEditOwn,
		capability.ReviewDeleteOwn,
	})
	require.NoError(t, err)
	assert.True(t, facts.CanViewFull)
	assert.True(t, facts.CanPostReview)
	assert.True(t, facts.CanEditOwn)
	assert.True(t, facts.CanDeleteOwn)

	facts, err = service.ResolveAccessFacts(context.Background(), "user-1", []string{
		capability.ReviewListFull,
	})
	require.NoError(t, err)
	assert.True(t, facts.CanViewFull)
	assert.False(t, facts.CanPostReview)
	assert.False(t, facts.CanEditOwn)
	assert.False(t, facts.CanDeleteOwn)
}

func int64Ptr(v int64) *int64 {
	return &v
}
