package review

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/pkg/capability"
	"github.com/StuHelper/StuHelper/server/internal/pkg/reviewaccess"
	"github.com/StuHelper/StuHelper/server/internal/pkg/systemconfig"
)

type fakeReviewAccessReader struct {
	subject *reviewaccess.Subject
}

func (f fakeReviewAccessReader) ListReviewAccessSchoolConfigs(context.Context) ([]reviewaccess.SchoolConfig, error) {
	return []reviewaccess.SchoolConfig{{SchoolID: 4111010006}}, nil
}

func (f fakeReviewAccessReader) ListReviewAccessSystemConfigs(context.Context) ([]reviewaccess.SystemConfig, error) {
	return nil, nil
}

func (f fakeReviewAccessReader) GetReviewAccessSubject(context.Context, string) (*reviewaccess.Subject, error) {
	return f.subject, nil
}

func (f fakeReviewAccessReader) PhonePublishingRequirementSatisfied(context.Context, int64) (bool, error) {
	return f.subject != nil && f.subject.IdentityVerified, nil
}

func TestBuildReviewAccessPolicy_UsesConfiguredValues(t *testing.T) {
	policy, err := buildReviewAccessPolicy(
		[]reviewaccess.SchoolConfig{
			{SchoolID: 4111010006},
			{SchoolID: 4111010007},
		},
		[]reviewaccess.SystemConfig{
			{Key: systemconfig.ReviewAccessSchoolIDsKey, Value: `["4111010007","4111010008"]`},
			{Key: systemconfig.ReviewGuestPreviewContentCharsKey, Value: "36"},
			{Key: systemconfig.ReviewPreviewContentCharsKey, Value: "180"},
			{Key: systemconfig.ReviewPreviewContentPercentKey, Value: "40"},
		},
	)
	require.NoError(t, err)

	assert.True(t, policy.AllowsSchool("4111010007"))
	assert.True(t, policy.AllowsSchool("4111010008"))
	assert.False(t, policy.AllowsSchool("4111010006"))
	assert.Equal(t, 36, policy.GuestPreviewContentRunes)
	assert.Equal(t, 180, policy.PreviewContentRunes)
	assert.Equal(t, 40, policy.PreviewContentPct)
}

func TestBuildReviewAccessPolicy_UsesEnabledSchoolsAndPreviewKeys(t *testing.T) {
	policy, err := buildReviewAccessPolicy(
		[]reviewaccess.SchoolConfig{
			{SchoolID: 4111010006},
			{SchoolID: 4111010007},
		},
		[]reviewaccess.SystemConfig{
			{Key: systemconfig.ReviewPreviewContentCharsKey, Value: "96"},
			{Key: systemconfig.ReviewPreviewContentPercentKey, Value: "25"},
		},
	)
	require.NoError(t, err)

	assert.True(t, policy.AllowsSchool("4111010006"))
	assert.True(t, policy.AllowsSchool("4111010007"))
	assert.Equal(t, systemconfig.DefaultReviewAccessPolicySnapshot().GuestPreviewContentRunes, policy.GuestPreviewContentRunes)
	assert.Equal(t, 96, policy.PreviewContentRunes)
	assert.Equal(t, 25, policy.PreviewContentPct)
}

func TestPreviewText_RespectsConfiguredPercent(t *testing.T) {
	value := strings.Repeat("评", 200)
	preview := previewText(value, 120, 20)
	assert.Equal(t, strings.Repeat("评", 40)+"...", preview)
}

func TestStripReviewsForResponse_LocksContentWithoutFullAccess(t *testing.T) {
	fullContent := "第一行会作为安全预览显示\n第二行正文不应该返回给无完整查看权限的用户"
	reviews := []Review{{
		ID:      "review-1",
		Title:   "保留课程标题",
		Content: fullContent,
		Status:  StatusPublished,
	}}

	result := stripReviewsForResponse(reviews, ReviewAccessFacts{
		Authenticated:       true,
		CanViewFull:         false,
		PreviewContentRunes: 8,
		PreviewContentPct:   100,
	})

	require.Len(t, result, 1)
	assert.Equal(t, "保留课程标题", result[0].Title)
	assert.Equal(t, "第一行会作为安全...", result[0].Content)
	assert.NotContains(t, result[0].Content, "第二行正文")
	assert.Equal(t, fullContent, reviews[0].Content)
}

func TestStripReviewsForResponse_AppliesAuthenticatedPreviewPolicyToFirstLine(t *testing.T) {
	firstLine := strings.Repeat("评", 20)
	reviews := []Review{{
		ID:      "review-percent",
		Title:   "标题保持可见",
		Content: firstLine + "\n第二行不得泄露",
		Status:  StatusPublished,
	}}

	result := stripReviewsForResponse(reviews, ReviewAccessFacts{
		Authenticated:       true,
		PreviewContentRunes: 96,
		PreviewContentPct:   25,
	})

	require.Len(t, result, 1)
	assert.Equal(t, "标题保持可见", result[0].Title)
	assert.Equal(t, strings.Repeat("评", 5)+"...", result[0].Content)
	assert.NotContains(t, result[0].Content, "第二行")
}

func TestStripReviewsForResponse_HidesAnonymousContentAndTitle(t *testing.T) {
	reviews := []Review{{
		ID:      "review-1",
		Title:   "匿名不可见标题",
		Content: "\n匿名可以看到第一行安全预览\n但不能看到后文",
		Status:  StatusPublished,
	}}

	result := stripReviewsForResponse(reviews, ReviewAccessFacts{
		GuestPreviewContentRunes: 12,
	})

	require.Len(t, result, 1)
	assert.Empty(t, result[0].Title)
	assert.Equal(t, "匿名可以看到第一行安全预...", result[0].Content)
	assert.NotContains(t, result[0].Content, "但不能看到后文")
}

func TestResolveAccessFacts_RequiresCapabilityAndVerificationFacts(t *testing.T) {
	reader := fakeReviewAccessReader{
		subject: &reviewaccess.Subject{
			InternalUserID:   42,
			IdentityVerified: true,
			StudentVerified:  true,
			SchoolID:         int64Ptr(4111010006),
		},
	}
	service := &Service{
		accessReader: reader,
		phoneGate:    reader,
	}

	facts, err := service.ResolveAccessFacts(context.Background(), "user-1", []string{
		capability.ReviewListFull,
		capability.ReviewCreate,
		capability.ReviewEditOwn,
		capability.ReviewDeleteOwn,
	}, nil)
	require.NoError(t, err)
	assert.True(t, facts.CanViewFull)
	assert.True(t, facts.CanPostReview)
	assert.True(t, facts.CanEditOwn)
	assert.True(t, facts.CanDeleteOwn)

	facts, err = service.ResolveAccessFacts(context.Background(), "user-1", []string{
		capability.ReviewListFull,
	}, nil)
	require.NoError(t, err)
	assert.True(t, facts.CanViewFull)
	assert.False(t, facts.CanPostReview)
	assert.False(t, facts.CanEditOwn)
	assert.False(t, facts.CanDeleteOwn)
}

func int64Ptr(v int64) *int64 {
	return &v
}
