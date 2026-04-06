package course

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ---- sortTermsForResponse: real sorting behavior ----

func TestSortTermsForResponse_CurrentFirst(t *testing.T) {
	terms := []Term{
		{ID: "2024-fall", Name: "2024 秋", IsCurrent: false},
		{ID: "2025-spring", Name: "2025 春", IsCurrent: true},
		{ID: "2024-spring", Name: "2024 春", IsCurrent: false},
	}

	sorted := sortTermsForResponse(terms)

	assert.Equal(t, "2025-spring", sorted[0].ID, "current term should be first")
	assert.True(t, sorted[0].IsCurrent)
	// Non-current terms sorted by ID descending (2024-spring > 2024-fall lexicographically)
	assert.Equal(t, "2024-spring", sorted[1].ID)
	assert.Equal(t, "2024-fall", sorted[2].ID)
}

func TestSortTermsForResponse_NoCurrentTerm(t *testing.T) {
	terms := []Term{
		{ID: "2024-spring", Name: "2024 春", IsCurrent: false},
		{ID: "2025-fall", Name: "2025 秋", IsCurrent: false},
		{ID: "2024-fall", Name: "2024 秋", IsCurrent: false},
	}

	sorted := sortTermsForResponse(terms)

	// Sorted by ID descending when no current
	assert.Equal(t, "2025-fall", sorted[0].ID)
	assert.Equal(t, "2024-fall", sorted[len(sorted)-1].ID)
}

func TestSortTermsForResponse_MultipleCurrentTermsStableOrder(t *testing.T) {
	// Edge case: if multiple terms are marked current, they should all come first
	terms := []Term{
		{ID: "c", IsCurrent: true},
		{ID: "a", IsCurrent: false},
		{ID: "b", IsCurrent: true},
	}

	sorted := sortTermsForResponse(terms)

	// Both current terms come before non-current
	assert.True(t, sorted[0].IsCurrent)
	assert.True(t, sorted[1].IsCurrent)
	assert.False(t, sorted[2].IsCurrent)
}

func TestSortTermsForResponse_Empty(t *testing.T) {
	sorted := sortTermsForResponse(nil)
	assert.Empty(t, sorted)
}

func TestSortTermsForResponse_SingleElement(t *testing.T) {
	sorted := sortTermsForResponse([]Term{{ID: "only", IsCurrent: false}})
	assert.Len(t, sorted, 1)
	assert.Equal(t, "only", sorted[0].ID)
}

func TestSortTermsForResponse_DoesNotMutateOriginal(t *testing.T) {
	original := []Term{
		{ID: "b", IsCurrent: false},
		{ID: "a", IsCurrent: true},
	}
	originalFirst := original[0].ID

	_ = sortTermsForResponse(original)

	assert.Equal(t, originalFirst, original[0].ID, "original slice should not be mutated")
}

// ---- sentinel error ----

func TestErrCourseNotFound_IsSentinel(t *testing.T) {
	// Verify it's a distinct sentinel error usable with errors.Is
	assert.True(t, errors.Is(ErrCourseNotFound, ErrCourseNotFound))
	assert.False(t, errors.Is(ErrCourseNotFound, errors.New("course not found")))
}
