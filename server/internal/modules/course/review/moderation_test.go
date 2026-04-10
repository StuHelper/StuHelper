package review

import "testing"

func TestBuildReviewModerationDecision(t *testing.T) {
	tests := []struct {
		name       string
		result     *ContentCheckResult
		wantStatus string
		wantFlag   *string
		wantErr    error
	}{
		{
			name:       "clean content stays published",
			result:     &ContentCheckResult{IsValid: true},
			wantStatus: StatusPublished,
		},
		{
			name:       "warn content stays published with warn flag",
			result:     &ContentCheckResult{IsValid: true, Level: ContentFlagWarn, MatchCount: 1},
			wantStatus: StatusPublished,
			wantFlag:   strPtr(ContentFlagWarn),
		},
		{
			name:       "review content enters pending review",
			result:     &ContentCheckResult{IsValid: true, Level: ContentFlagReview, MatchCount: 2},
			wantStatus: StatusPendingReview,
			wantFlag:   strPtr(ContentFlagReview),
		},
		{
			name:    "blocked content is rejected",
			result:  &ContentCheckResult{IsValid: false, Level: "block", MatchCount: 1},
			wantErr: ErrSensitiveContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision, err := buildReviewModerationDecision(tt.result)
			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if decision.Status != tt.wantStatus {
				t.Fatalf("expected status %q, got %q", tt.wantStatus, decision.Status)
			}
			if !sameStringPtr(decision.ContentFlag, tt.wantFlag) {
				t.Fatalf("expected flag %v, got %v", tt.wantFlag, decision.ContentFlag)
			}
		})
	}
}

func TestBuildReplyModerationStatus(t *testing.T) {
	status, err := buildReplyModerationStatus(&ContentCheckResult{IsValid: true, Level: ContentFlagReview, MatchCount: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != StatusPendingReview {
		t.Fatalf("expected %q, got %q", StatusPendingReview, status)
	}
}

func strPtr(value string) *string {
	return &value
}

func sameStringPtr(left, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}
