package course

import "testing"

func TestSortTermsCurrentFirst(t *testing.T) {
	terms := []Term{
		{ID: "2025-1", Name: "2025 春", IsCurrent: false},
		{ID: "2025-2", Name: "2025 秋", IsCurrent: true},
	}

	sorted := sortTermsForResponse(terms)
	if len(sorted) != 2 {
		t.Fatalf("expected 2 terms, got %d", len(sorted))
	}
	if sorted[0].ID != "2025-2" || sorted[1].ID != "2025-1" {
		t.Fatalf("expected current term first, got %#v", sorted)
	}
}
