package systemconfig

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestReviewAccessPolicySnapshotCache_IsolatedCopies(t *testing.T) {
	SetReviewAccessPolicySnapshot(ReviewAccessPolicySnapshot{
		AllowedSchoolIDs:         []string{"4111010006"},
		GuestPreviewContentRunes: 32,
		PreviewContentRunes:      160,
		PreviewContentPct:        50,
		LoadedAt:                 time.Now(),
	})
	t.Cleanup(InvalidateReviewAccessPolicySnapshot)

	snapshot := GetReviewAccessPolicySnapshot()
	snapshot.AllowedSchoolIDs[0] = "mutated"

	current := GetReviewAccessPolicySnapshot()
	assert.Equal(t, []string{"4111010006"}, current.AllowedSchoolIDs)
	assert.Equal(t, 32, current.GuestPreviewContentRunes)
}

func TestInvalidateReviewAccessPolicySnapshot_RestoresDefaults(t *testing.T) {
	SetReviewAccessPolicySnapshot(ReviewAccessPolicySnapshot{
		AllowedSchoolIDs:         []string{"4111010006"},
		GuestPreviewContentRunes: 32,
		PreviewContentRunes:      160,
		PreviewContentPct:        50,
		LoadedAt:                 time.Now(),
	})

	InvalidateReviewAccessPolicySnapshot()

	current := GetReviewAccessPolicySnapshot()
	assert.Nil(t, current.AllowedSchoolIDs)
	assert.Equal(t, DefaultReviewAccessPolicySnapshot().GuestPreviewContentRunes, current.GuestPreviewContentRunes)
	assert.Equal(t, DefaultReviewAccessPolicySnapshot().PreviewContentRunes, current.PreviewContentRunes)
	assert.Equal(t, DefaultReviewAccessPolicySnapshot().PreviewContentPct, current.PreviewContentPct)
	assert.True(t, current.LoadedAt.IsZero())
}
