package fga

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReviewModerationSectionID(t *testing.T) {
	assert.Equal(t, "school_4111010006_review_moderation", ReviewModerationSectionID(" 4111010006 "))
}

func TestParseSyntheticSectionID(t *testing.T) {
	schoolID, kind, ok := ParseSyntheticSectionID("school_4111010006_review_moderation")
	assert.True(t, ok)
	assert.Equal(t, "4111010006", schoolID)
	assert.Equal(t, SectionKindReviewModeration, kind)
}

func TestParseReviewModerationSectionID(t *testing.T) {
	tests := []struct {
		name      string
		sectionID string
		wantID    string
		wantOK    bool
	}{
		{name: "valid", sectionID: "school_4111010006_review_moderation", wantID: "4111010006", wantOK: true},
		{name: "wrong suffix", sectionID: "school_4111010006_qa", wantOK: false},
		{name: "wrong prefix", sectionID: "section_4111010006_review_moderation", wantOK: false},
		{name: "empty id", sectionID: "school__review_moderation", wantOK: false},
		{name: "non numeric", sectionID: "school_abc_review_moderation", wantOK: false},
		{name: "zero", sectionID: "school_0_review_moderation", wantOK: false},
		{name: "leading zero", sectionID: "school_010006_review_moderation", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotOK := ParseReviewModerationSectionID(tt.sectionID)
			assert.Equal(t, tt.wantOK, gotOK)
			assert.Equal(t, tt.wantID, gotID)
		})
	}
}
